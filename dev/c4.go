//go:build targ

package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/toejough/targ"
	"golang.org/x/tools/go/packages"
)

func init() {
	targ.Register(targ.Targ(c4Audit).Name("c4-audit").
		Description("Structurally audit a C4 L1 markdown file. Exits 1 on any finding."))
	targ.Register(targ.Targ(c4L1Build).Name("c4-l1-build").
		Description("Build canonical C4 L1 markdown from a JSON spec next to the input file."))
	targ.Register(targ.Targ(c4L1Externals).Name("c4-l1-externals").
		Description("Walk the repo with Go AST analysis and emit external-system candidates as JSON."))
	targ.Register(targ.Targ(c4History).Name("c4-history").
		Description("Wrap git log and emit structured JSON of commit metadata + bodies."))
}

// C4AuditArgs configures the c4-audit target.
type C4AuditArgs struct {
	File string `targ:"flag,name=file,desc=Markdown file to audit (required)"`
	JSON bool   `targ:"flag,name=json,desc=Emit findings as JSON"`
}

// C4HistoryArgs configures the c4-history target.
type C4HistoryArgs struct {
	Limit int    `targ:"flag,name=limit,desc=Max commits (default: 50)"`
	Since string `targ:"flag,name=since,desc=git log --since"`
	Grep  string `targ:"flag,name=grep,desc=git log --grep"`
	Paths string `targ:"flag,name=paths,desc=Comma-separated path filters"`
}

// C4L1BuildArgs configures the c4-l1-build target.
type C4L1BuildArgs struct {
	Input     string `targ:"flag,name=input,desc=JSON spec path (required)"`
	Check     bool   `targ:"flag,name=check,desc=Verify existing .md matches generated; non-zero on diff"`
	NoConfirm bool   `targ:"flag,name=noconfirm,desc=Overwrite existing .md without prompting"`
}

// C4L1ExternalsArgs configures the c4-l1-externals target.
type C4L1ExternalsArgs struct {
	Root         string `targ:"flag,name=root,desc=Module root (default: .)"`
	Packages     string `targ:"flag,name=packages,desc=Packages pattern (default: ./...)"`
	IncludeTests bool   `targ:"flag,name=includetests,desc=Include _test.go files"`
}

// Finding records a single structural problem detected by the audit.
type Finding struct {
	ID     string `json:"id"`
	Line   int    `json:"line"`
	Detail string `json:"detail"`
}

type L1CrossLinks struct {
	RefinedBy []L1RefinedBy `json:"refined_by"`
}

type L1DriftNote struct {
	Date   string `json:"date"`
	Detail string `json:"detail"`
	Reason string `json:"reason"`
}

type L1Element struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	Subtitle       *string `json:"subtitle,omitempty"`
	Responsibility string  `json:"responsibility"`
	SystemOfRecord string  `json:"system_of_record"`
}

type L1RefinedBy struct {
	File string `json:"file"`
	Note string `json:"note"`
}

type L1Relationship struct {
	From          string `json:"from"`
	To            string `json:"to"`
	Description   string `json:"description"`
	Protocol      string `json:"protocol"`
	Bidirectional bool   `json:"bidirectional,omitempty"`
}

// L1Spec is the JSON-source-of-truth representation of a C4 L1 diagram.
type L1Spec struct {
	SchemaVersion string           `json:"schema_version"`
	Level         int              `json:"level"`
	Name          string           `json:"name"`
	Parent        *string          `json:"parent"`
	Preamble      string           `json:"preamble"`
	Elements      []L1Element      `json:"elements"`
	Relationships []L1Relationship `json:"relationships"`
	DriftNotes    []L1DriftNote    `json:"drift_notes"`
	CrossLinks    L1CrossLinks     `json:"cross_links"`
}

// unexported constants.
const (
	fmFence = "---"
)

// unexported variables.
var (
	anchorInCellRe    = regexp.MustCompile(`<a\s+id="([^"]+)"></a>`)
	catalogHeaderRe   = regexp.MustCompile(`(?m)^##\s+Element Catalog\s*$`)
	dataFormatMethods = map[string]bool{
		"Marshal": true, "MarshalIndent": true, "Unmarshal": true,
		"NewEncoder": true, "NewDecoder": true,
		"Encode": true, "Decode": true,
	}
	dataFormatPackages = map[string]string{
		"encoding/json":              "json",
		"encoding/xml":               "xml",
		"encoding/gob":               "gob",
		"github.com/BurntSushi/toml": "toml",
		"go.yaml.in/yaml/v3":         "yaml",
		"gopkg.in/yaml.v3":           "yaml",
	}
	// edgeIDPrefix accepts R<n> (direct call) or D<n> (DI back-edge) labels.
	// Both prefixes appear at L2-L4; L1 only uses R<n> but D<n> never appears
	// there so the union is harmless.
	edgeIDPrefix  = regexp.MustCompile(`^[RD]\d+\s*:`)
	errNoArgs     = errors.New("--file is required")
	httpMethodSet = map[string]bool{
		"NewRequest": true, "NewRequestWithContext": true,
		"Get": true, "Post": true, "Put": true, "Delete": true, "Head": true, "PostForm": true,
	}
	levelPrefixRe      = regexp.MustCompile(`^c[1-4]-`)
	mermaidClassRe     = regexp.MustCompile(`^\s*class\s+([\w,-]+)\s+(\w+)\s*$`)
	mermaidClickRe     = regexp.MustCompile(`^\s*click\s+([\w-]+)\s+href\s+"#([^"]+)"`)
	mermaidEdgeRe      = regexp.MustCompile(`^\s*([\w-]+?)\s+<?[-.=]+>+\s*(?:\|([^|]*)\|\s*)?([\w-]+)\s*$`)
	mermaidFenceRe     = regexp.MustCompile("(?m)^```mermaid")
	mermaidIDPrefix    = regexp.MustCompile(`^[SNMP]\d+(?:-[SNMP]\d+)*`)
	mermaidNodeRe      = regexp.MustCompile(`^\s*([\w-]+)\s*[(\[]+(.*?)[)\]]+\s*$`)
	mermaidSubgraphRe  = regexp.MustCompile(`^\s*subgraph\s+([\w-]+)\s*\[(.*?)\]\s*$`)
	nameStatusLineRe   = regexp.MustCompile(`^[A-Z][A-Z0-9]*\t`)
	relsHeaderRe       = regexp.MustCompile(`(?m)^##\s+Relationships\s*$`)
	sinceShorthandRe   = regexp.MustCompile(`^(\d+)([dwmy])$`)
	slugSplitRe        = regexp.MustCompile(`[^a-z0-9]+`)
	svgEmbedRe         = regexp.MustCompile(`!\[[^\]]*\]\(svg/([A-Za-z0-9._-]+)\.svg\)`)
	tableRowFirstRe    = regexp.MustCompile(`^\s*\|\s*(?:<a\s+id="([^"]+)"></a>)?\s*([SNMPR]\d+(?:-[SNMP]\d+)*)\s*\|`)
	validKinds         = map[string]bool{"person": true, "external": true, "container": true}
	validNameRe        = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	validRefinedByFile = regexp.MustCompile(`^c2-[a-z0-9-]+\.md$`)
)

type auditJSON struct {
	SchemaVersion string    `json:"schema_version"`
	File          string    `json:"file"`
	Findings      []Finding `json:"findings"`
}

type catalogRow struct {
	id       string
	anchorID string
	name     string
	line     int
}

// elementID pairs a source element with its assigned hierarchical identifier
// and anchor slug.
type elementID struct {
	ID       string
	AnchorID string
	Element  L1Element
}

// externalFinding is one detected outbound dependency from the scanned code.
type externalFinding struct {
	Kind     string `json:"kind"`
	Target   string `json:"target"`
	Source   string `json:"source"`
	Evidence string `json:"evidence"`
}

// frontMatter holds the parsed YAML front-matter fields for a C4 markdown file.
// Each field tracks whether it appeared and on which source line, to support
// precise findings.
type frontMatter struct {
	present                bool
	startLine              int
	endLine                int
	hasLevel               bool
	level                  int
	levelLine              int
	hasName                bool
	name                   string
	nameLine               int
	hasParent              bool
	parentNull             bool
	parent                 string
	parentLine             int
	hasChildren            bool
	children               []string
	childrenLine           int
	hasLastReviewedCommit  bool
	lastReviewedCommit     string
	lastReviewedCommitLine int
}

type historyCommit struct {
	SHA          string              `json:"sha"`
	Date         string              `json:"date"`
	Author       string              `json:"author"`
	Subject      string              `json:"subject"`
	Body         string              `json:"body"`
	FilesChanged []historyFileChange `json:"files_changed"`
}

type historyFileChange struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

type historyOptions struct {
	root  string
	paths []string
	since string
	limit int
	grep  string
}

// mermaidBlock holds the parsed contents of a ```mermaid fenced code block.
type mermaidBlock struct {
	startLine   int
	endLine     int
	body        string
	hasClassDef bool
	classes     map[string]bool
	nodes       []mermaidNode
	edges       []mermaidEdge
	clicks      []mermaidClick
}

type mermaidClick struct {
	node       string
	hrefAnchor string
	line       int
}

type mermaidEdge struct {
	from  string
	to    string
	label string
	line  int
}

type mermaidNode struct {
	id    string
	label string
	line  int
}

// relationshipID pairs a source relationship with its assigned R<n> identifier
// and anchor slug.
type relationshipID struct {
	ID       string
	AnchorID string
	Rel      L1Relationship
}

type relationshipsRow struct {
	id       string
	anchorID string
	from     string
	to       string
	line     int
}

// assignElementIDs reads explicit S<n> IDs from each element and returns them
// paired with their lowercase-ID + slug anchor. AnchorID is
// "<lower(id)>-<slug(name)>"; collisions append "-2", "-3".... Returns an error
// if any element has an empty id or one that is not a level-1 hierarchical
// path (i.e. exactly "S<n>").
func assignElementIDs(elements []L1Element) ([]elementID, error) {
	result := make([]elementID, 0, len(elements))
	used := map[string]int{}
	for index, element := range elements {
		if element.ID == "" {
			return nil, fmt.Errorf("elements[%d]: missing id", index)
		}
		if err := ValidateElementID(1, IDPath{}, element.ID); err != nil {
			return nil, fmt.Errorf("elements[%d]: %w", index, err)
		}
		base := Anchor(element.ID, element.Name)
		anchor := base
		if used[base] > 0 {
			used[base]++
			anchor = fmt.Sprintf("%s-%d", base, used[base])
		} else {
			used[base] = 1
		}
		result = append(result, elementID{
			ID:       element.ID,
			AnchorID: anchor,
			Element:  element,
		})
	}
	return result, nil
}

// assignRelationshipIDs returns relationships paired with sequential R1..Rn IDs.
// AnchorID is "r<n>-<slug(from)>-<slug(to)>"; collisions append "-2"....
// At L2/L3 the from/to fields hold S/N/M-IDs directly, producing anchors like
// "r1-s2-n3". L1 callers pre-substitute names into from/to via
// l1RelsWithNames before calling this so anchors slug from display names.
func assignRelationshipIDs(rels []L1Relationship) []relationshipID {
	result := make([]relationshipID, 0, len(rels))
	used := map[string]int{}
	for index, rel := range rels {
		base := fmt.Sprintf("r%d-%s-%s", index+1, slug(rel.From), slug(rel.To))
		anchor := base
		if used[base] > 0 {
			used[base]++
			anchor = fmt.Sprintf("%s-%d", base, used[base])
		} else {
			used[base] = 1
		}
		result = append(result, relationshipID{
			ID:       fmt.Sprintf("R%d", index+1),
			AnchorID: anchor,
			Rel:      rel,
		})
	}
	return result
}

// auditAnchorsAndClicks emits click_target_unresolved for click anchors that
// don't match any catalog or relationships row anchor, and anchor_missing for
// table rows lacking an anchor. The earlier click_missing check (every node
// must have a click directive) was dropped when diagrams moved to
// pre-rendered SVG: click handlers don't carry through static SVG, so they're
// optional in the .mmd source. Anchors on catalog/relationships rows remain
// load-bearing because in-page links into them work in the markdown body.
func auditAnchorsAndClicks(block *mermaidBlock, catalog []catalogRow, rels []relationshipsRow) []Finding {
	if block == nil {
		return nil
	}
	findings := []Finding{}
	anchorSet := map[string]bool{}
	for _, row := range catalog {
		if row.anchorID != "" {
			anchorSet[row.anchorID] = true
		} else {
			findings = append(findings, Finding{
				ID: "anchor_missing", Line: row.line,
				Detail: fmt.Sprintf("catalog row %s has no <a id=\"...\"></a>", row.id),
			})
		}
	}
	for _, row := range rels {
		if row.anchorID != "" {
			anchorSet[row.anchorID] = true
		} else {
			findings = append(findings, Finding{
				ID: "anchor_missing", Line: row.line,
				Detail: fmt.Sprintf("relationships row %s has no <a id=\"...\"></a>", row.id),
			})
		}
	}
	for _, click := range block.clicks {
		if !anchorSet[click.hrefAnchor] {
			findings = append(findings, Finding{
				ID: "click_target_unresolved", Line: click.line,
				Detail: fmt.Sprintf("click %s href #%s does not match any catalog/relationships anchor",
					click.node, click.hrefAnchor),
			})
		}
	}
	return findings
}

// auditFile reads path and returns all structural findings.
// It returns an error only when the file itself cannot be read.
//
// L1–L3 markdown is checked against the full set of structural rules
// (mermaid block, element catalog, relationships table, click directives,
// JSON registry cross-check). L4 ledgers use a different schema (Property
// Ledger + Dependency Manifest + DI Wires; no Element Catalog table; no
// JSON spec; rendered to SVG with no click handlers) so the L1–L3 checks
// are skipped for level 4 — only front-matter and code-pointer checks run.
func auditFile(ctx context.Context, path string) ([]Finding, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	findings := []Finding{}
	findings = append(findings, auditFrontMatter(ctx, path, raw)...)
	matter, _ := parseFrontMatter(raw)
	findings = append(findings, checkCodePointers(matter, raw, path)...)
	findings = append(findings, checkPropertyLinks(matter, raw, path)...)
	if matter.level == 4 {
		// L4 ledgers use a different schema (no Element Catalog, no JSON
		// registry cross-check at audit time, SVG-rendered with no click
		// handlers). Diagram-id discipline is enforced by c4-l4-build at
		// generation time (#598), so audit only runs front-matter and
		// code-pointer checks for L4.
		return findings, nil
	}
	block, mermaidFindings := parseMermaidBlock(raw, path)
	findings = append(findings, mermaidFindings...)
	catalog, rels := parseTables(raw)
	findings = append(findings, auditOrphans(block, catalog, rels)...)
	findings = append(findings, auditAnchorsAndClicks(block, catalog, rels)...)
	return findings, nil
}

// auditFrontMatter parses the YAML front-matter and emits findings for missing
// blocks, missing fields, invalid values, and broken cross-references.
func auditFrontMatter(ctx context.Context, path string, raw []byte) []Finding {
	matter, ok := parseFrontMatter(raw)
	if !ok {
		return []Finding{{ID: "front_matter_missing", Line: 1, Detail: "no leading --- block"}}
	}
	findings := []Finding{}
	findings = append(findings, checkRequiredFields(matter)...)
	findings = append(findings, checkLevel(matter)...)
	findings = append(findings, checkName(matter, path)...)
	findings = append(findings, checkParent(matter, path)...)
	findings = append(findings, checkLastReviewedCommit(ctx, matter)...)
	findings = append(findings, checkChildren(matter)...)
	return findings
}

// auditOrphans cross-references mermaid IDs with catalog/relationships rows and
// emits node_orphan, catalog_orphan, edge_orphan, relationships_orphan findings.
func auditOrphans(block *mermaidBlock, catalog []catalogRow, rels []relationshipsRow) []Finding {
	if block == nil {
		return nil
	}
	findings := []Finding{}
	mermaidNodeIDs := map[string]int{}
	for _, node := range block.nodes {
		if matched := mermaidIDPrefix.FindString(strings.TrimSpace(node.label)); matched != "" {
			mermaidNodeIDs[matched] = node.line
		}
	}
	mermaidEdgeIDs := map[string]int{}
	for _, edge := range block.edges {
		label := strings.TrimSpace(edge.label)
		if matched := regexp.MustCompile(`^R\d+`).FindString(label); matched != "" {
			mermaidEdgeIDs[matched] = edge.line
		}
	}
	catalogIDs := map[string]int{}
	for _, row := range catalog {
		catalogIDs[row.id] = row.line
	}
	relsIDs := map[string]int{}
	for _, row := range rels {
		relsIDs[row.id] = row.line
	}
	for nodeID, line := range mermaidNodeIDs {
		if _, ok := catalogIDs[nodeID]; !ok {
			findings = append(findings, Finding{
				ID: "node_orphan", Line: line,
				Detail: fmt.Sprintf("mermaid node %s has no catalog row", nodeID),
			})
		}
	}
	for catID, line := range catalogIDs {
		if _, ok := mermaidNodeIDs[catID]; !ok {
			findings = append(findings, Finding{
				ID: "catalog_orphan", Line: line,
				Detail: fmt.Sprintf("catalog row %s has no mermaid node", catID),
			})
		}
	}
	for edgeID, line := range mermaidEdgeIDs {
		if _, ok := relsIDs[edgeID]; !ok {
			findings = append(findings, Finding{
				ID: "edge_orphan", Line: line,
				Detail: fmt.Sprintf("mermaid edge %s has no relationships row", edgeID),
			})
		}
	}
	for relID, line := range relsIDs {
		if _, ok := mermaidEdgeIDs[relID]; !ok {
			findings = append(findings, Finding{
				ID: "relationships_orphan", Line: line,
				Detail: fmt.Sprintf("relationships row %s has no mermaid edge", relID),
			})
		}
	}
	return findings
}

func c4Audit(ctx context.Context, args C4AuditArgs) error {
	if args.File == "" {
		return errNoArgs
	}
	findings, err := auditFile(ctx, args.File)
	if err != nil {
		return err
	}
	if args.JSON {
		if err := writeFindingsJSON(os.Stdout, args.File, findings); err != nil {
			return err
		}
	} else {
		writeFindingsText(os.Stdout, args.File, findings)
	}
	if len(findings) > 0 {
		return fmt.Errorf("%d finding(s)", len(findings))
	}
	return nil
}

func c4History(ctx context.Context, args C4HistoryArgs) error {
	limit := args.Limit
	if limit == 0 {
		limit = 50
	}
	var paths []string
	if args.Paths != "" {
		paths = strings.Split(args.Paths, ",")
	}
	commits, err := scanHistory(ctx, historyOptions{
		root: ".", paths: paths, since: translateSinceShorthand(args.Since), limit: limit, grep: args.Grep,
	})
	if err != nil {
		return err
	}
	type filters struct {
		Paths []string `json:"paths"`
		Since string   `json:"since"`
		Limit int      `json:"limit"`
		Grep  string   `json:"grep"`
	}
	type out struct {
		SchemaVersion string          `json:"schema_version"`
		Filters       filters         `json:"filters"`
		Commits       []historyCommit `json:"commits"`
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out{
		SchemaVersion: "1",
		Filters:       filters{Paths: paths, Since: args.Since, Limit: limit, Grep: args.Grep},
		Commits:       commits,
	}); err != nil {
		return fmt.Errorf("encode history: %w", err)
	}
	return nil
}

func c4L1Build(ctx context.Context, args C4L1BuildArgs) error {
	if args.Input == "" {
		return errors.New("--input is required")
	}
	spec, err := loadAndValidateSpec(args.Input)
	if err != nil {
		return err
	}
	sha, err := currentGitShortSHA(ctx)
	if err != nil {
		return fmt.Errorf("git rev-parse: %w", err)
	}
	mdPath := strings.TrimSuffix(args.Input, ".json") + ".md"
	mmdPath := filepath.Join(filepath.Dir(args.Input), "svg",
		strings.TrimSuffix(filepath.Base(args.Input), ".json")+".mmd")
	var mdBuf bytes.Buffer
	if err := emitMarkdown(&mdBuf, spec, sha); err != nil {
		return err
	}
	elementIDs, err := assignElementIDs(spec.Elements)
	if err != nil {
		return fmt.Errorf("assign element ids: %w", err)
	}
	nameByID := nameByElementID(elementIDs)
	relsByName := l1RelsWithNames(spec.Relationships, nameByID)
	relIDs := assignRelationshipIDs(relsByName)
	var mmdBuf bytes.Buffer
	emitMermaid(&mmdBuf, elementIDs, relIDs)
	if err := writeOrCheckMarkdown(mdPath, mdBuf.Bytes(), args.Check, args.NoConfirm); err != nil {
		return err
	}
	return writeOrCheckMarkdown(mmdPath, mmdBuf.Bytes(), args.Check, args.NoConfirm)
}

func c4L1Externals(ctx context.Context, args C4L1ExternalsArgs) error {
	root := args.Root
	if root == "" {
		root = "."
	}
	pattern := args.Packages
	if pattern == "" {
		pattern = "./..."
	}
	findings, err := scanExternals(ctx, root, pattern, args.IncludeTests)
	if err != nil {
		return err
	}
	type out struct {
		SchemaVersion string            `json:"schema_version"`
		ScannedRoot   string            `json:"scanned_root"`
		Pattern       string            `json:"pattern"`
		Findings      []externalFinding `json:"findings"`
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out{SchemaVersion: "1", ScannedRoot: root, Pattern: pattern, Findings: findings}); err != nil {
		return fmt.Errorf("encode externals: %w", err)
	}
	return nil
}

func callOnPackage(pkg *packages.Package, node ast.Node) (*ast.CallExpr, *ast.SelectorExpr, string) {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return nil, nil, ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, nil, ""
	}
	xIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil, nil, ""
	}
	if pkg.TypesInfo == nil {
		return nil, nil, ""
	}
	use, ok := pkg.TypesInfo.Uses[xIdent].(*types.PkgName)
	if !ok {
		return nil, nil, ""
	}
	return call, sel, use.Imported().Path()
}

// collectL4MermaidFindings is the L4 counterpart to collectMermaidFindings.
// It validates that every node label is a hierarchical path ID (S<n>-N<m>-M<k>
// or shallower) and that every edge label starts with R<n>: (consumer-side
// relationship) or D<n>: (DI back-edge). No other prefixes are allowed — the
// node ID space is closed to the L4 context strip and the edge ID space is
// closed to the two documented namespaces.
func checkLastReviewedCommit(ctx context.Context, matter frontMatter) []Finding {
	if !matter.hasLastReviewedCommit {
		return nil
	}
	if matter.lastReviewedCommit == "" {
		return []Finding{{
			ID:     "last_reviewed_commit_invalid",
			Line:   matter.lastReviewedCommitLine,
			Detail: "last_reviewed_commit is empty",
		}}
	}
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(os.Stderr, "warning: git not on PATH; skipping last_reviewed_commit verification")
		return nil
	}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--quiet", "--verify", matter.lastReviewedCommit+"^{commit}")
	if err := cmd.Run(); err != nil {
		return []Finding{{
			ID:     "last_reviewed_commit_invalid",
			Line:   matter.lastReviewedCommitLine,
			Detail: fmt.Sprintf("git rev-parse rejected %q", matter.lastReviewedCommit),
		}}
	}
	return nil
}

func checkLevel(matter frontMatter) []Finding {
	if !matter.hasLevel {
		return nil
	}
	if matter.level < 1 || matter.level > 4 {
		return []Finding{{
			ID:     "level_invalid",
			Line:   matter.levelLine,
			Detail: fmt.Sprintf("level %d not in [1..4]", matter.level),
		}}
	}
	return nil
}

func checkName(matter frontMatter, path string) []Finding {
	if !matter.hasName {
		return nil
	}
	base := strings.TrimSuffix(filepath.Base(path), ".md")
	base = levelPrefixRe.ReplaceAllString(base, "")
	expected := slug(base)
	if matter.name != expected {
		return []Finding{{
			ID:     "name_filename_mismatch",
			Line:   matter.nameLine,
			Detail: fmt.Sprintf("name %q does not match filename slug %q", matter.name, expected),
		}}
	}
	if !validNameRe.MatchString(matter.name) {
		return []Finding{{
			ID:     "name_filename_mismatch",
			Line:   matter.nameLine,
			Detail: fmt.Sprintf("name %q must match %s", matter.name, validNameRe),
		}}
	}
	return nil
}

func checkParent(matter frontMatter, path string) []Finding {
	if !matter.hasParent || matter.parentNull {
		return nil
	}
	resolved := filepath.Join(filepath.Dir(path), matter.parent)
	if _, err := os.Stat(resolved); err != nil {
		return []Finding{{
			ID:     "parent_missing",
			Line:   matter.parentLine,
			Detail: fmt.Sprintf("parent %q resolves to %q but does not exist", matter.parent, resolved),
		}}
	}
	return nil
}

func checkRequiredFields(matter frontMatter) []Finding {
	findings := []Finding{}
	required := []struct {
		present bool
		name    string
	}{
		{matter.hasLevel, "level"},
		{matter.hasName, "name"},
		{matter.hasParent, "parent"},
		{matter.hasChildren, "children"},
		{matter.hasLastReviewedCommit, "last_reviewed_commit"},
	}
	for _, field := range required {
		if !field.present {
			findings = append(findings, Finding{
				ID:     "front_matter_field_missing",
				Line:   matter.startLine,
				Detail: fmt.Sprintf("missing required field %q", field.name),
			})
		}
	}
	return findings
}

func classFor(element L1Element) string {
	if element.Kind == "person" {
		return "person"
	}
	if element.Kind == "container" {
		return "container"
	}
	return "external"
}

func collectMermaidFindings(block *mermaidBlock) []Finding {
	findings := []Finding{}
	requiredClasses := []string{"person", "external", "container"}
	for _, name := range requiredClasses {
		if !block.classes[name] {
			findings = append(findings, Finding{
				ID:     "classdef_missing",
				Line:   block.startLine,
				Detail: fmt.Sprintf("classDef %q not defined", name),
			})
		}
	}
	defined := map[string]mermaidNode{}
	for _, node := range block.nodes {
		defined[node.id] = node
		if !mermaidIDPrefix.MatchString(strings.TrimSpace(node.label)) {
			findings = append(findings, Finding{
				ID:     "node_id_missing",
				Line:   node.line,
				Detail: fmt.Sprintf("node %q label %q does not start with a hierarchical ID (S<n>, S<n>-N<n>, …)", node.id, node.label),
			})
		}
	}
	for _, edge := range block.edges {
		if _, ok := defined[edge.from]; !ok {
			findings = append(findings, Finding{
				ID:     "node_id_missing",
				Line:   edge.line,
				Detail: fmt.Sprintf("edge endpoint %q has no node definition", edge.from),
			})
		}
		if _, ok := defined[edge.to]; !ok {
			findings = append(findings, Finding{
				ID:     "node_id_missing",
				Line:   edge.line,
				Detail: fmt.Sprintf("edge endpoint %q has no node definition", edge.to),
			})
		}
		if !edgeIDPrefix.MatchString(strings.TrimSpace(edge.label)) {
			findings = append(findings, Finding{
				ID:     "edge_id_missing",
				Line:   edge.line,
				Detail: fmt.Sprintf("edge %q->%q label %q does not start with R<n>: or D<n>:", edge.from, edge.to, edge.label),
			})
		}
	}
	return findings
}

func consumeFrontMatterLine(matter *frontMatter, lineNum int, raw string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return
	}
	key, value, ok := strings.Cut(trimmed, ":")
	if !ok {
		return
	}
	value = strings.TrimSpace(value)
	switch strings.TrimSpace(key) {
	case "level":
		matter.hasLevel = true
		matter.levelLine = lineNum
		if n, err := strconv.Atoi(value); err == nil {
			matter.level = n
		} else {
			matter.level = -1
		}
	case "name":
		matter.hasName = true
		matter.nameLine = lineNum
		matter.name = strings.Trim(value, `"'`)
	case "parent":
		matter.hasParent = true
		matter.parentLine = lineNum
		if value == "null" || value == "~" || value == "" {
			matter.parentNull = true
		} else {
			matter.parent = strings.Trim(value, `"'`)
		}
	case "children":
		matter.hasChildren = true
		matter.childrenLine = lineNum
		matter.children = parseInlineYAMLArray(value)
	case "last_reviewed_commit":
		matter.hasLastReviewedCommit = true
		matter.lastReviewedCommitLine = lineNum
		matter.lastReviewedCommit = strings.Trim(value, `"'`)
	}
}

// containerCount returns the number of elements with kind="container". At L1
// exactly one element is the system in scope, marked by kind=container.
func containerCount(elements []L1Element) int {
	count := 0
	for _, element := range elements {
		if element.Kind == "container" {
			count++
		}
	}
	return count
}

func currentGitShortSHA(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func detectDataFormat(pkg *packages.Package, node ast.Node) []externalFinding {
	call, sel, pkgName := callOnPackage(pkg, node)
	if call == nil {
		return nil
	}
	format, ok := dataFormatPackages[pkgName]
	if !ok {
		return nil
	}
	if !dataFormatMethods[sel.Sel.Name] {
		return nil
	}
	return []externalFinding{{
		Kind:     "data_format",
		Target:   format,
		Source:   positionOf(pkg, call.Pos()),
		Evidence: exprText(pkg, call),
	}}
}

func detectEnvRead(pkg *packages.Package, node ast.Node) []externalFinding {
	call, sel, pkgName := callOnPackage(pkg, node)
	if call == nil || pkgName != "os" {
		return nil
	}
	if sel.Sel.Name != "Getenv" && sel.Sel.Name != "LookupEnv" {
		return nil
	}
	if len(call.Args) == 0 {
		return nil
	}
	target := stringValueOf(pkg, call.Args[0])
	if target == "" {
		return nil
	}
	return []externalFinding{{
		Kind:     "env_read",
		Target:   target,
		Source:   positionOf(pkg, call.Pos()),
		Evidence: exprText(pkg, call),
	}}
}

func detectExec(pkg *packages.Package, node ast.Node) []externalFinding {
	call, sel, pkgName := callOnPackage(pkg, node)
	if call == nil || pkgName != "os/exec" {
		return nil
	}
	if sel.Sel.Name != "Command" && sel.Sel.Name != "CommandContext" {
		return nil
	}
	argIdx := 0
	if sel.Sel.Name == "CommandContext" {
		argIdx = 1
	}
	if len(call.Args) <= argIdx {
		return nil
	}
	target := stringValueOf(pkg, call.Args[argIdx])
	if target == "" {
		return nil
	}
	return []externalFinding{{
		Kind:     "exec",
		Target:   target,
		Source:   positionOf(pkg, call.Pos()),
		Evidence: exprText(pkg, call),
	}}
}

func detectFSPath(pkg *packages.Package, node ast.Node) []externalFinding {
	call, sel, pkgName := callOnPackage(pkg, node)
	if call == nil || pkgName != "os" {
		return nil
	}
	switch sel.Sel.Name {
	case "UserHomeDir", "UserConfigDir", "UserCacheDir":
		return []externalFinding{{
			Kind:     "fs_path",
			Target:   "$HOME",
			Source:   positionOf(pkg, call.Pos()),
			Evidence: exprText(pkg, call),
		}}
	case "Open", "OpenFile", "ReadFile", "WriteFile", "Create", "CreateTemp", "MkdirAll", "Mkdir", "Remove", "RemoveAll":
		target := firstStringArg(pkg, call)
		if target == "" {
			target = "<dynamic>"
		}
		return []externalFinding{{
			Kind:     "fs_path",
			Target:   target,
			Source:   positionOf(pkg, call.Pos()),
			Evidence: exprText(pkg, call),
		}}
	}
	return nil
}

func detectHTTPCall(pkg *packages.Package, node ast.Node) []externalFinding {
	call, sel, pkgName := callOnPackage(pkg, node)
	if call == nil || pkgName != "net/http" || !httpMethodSet[sel.Sel.Name] {
		return nil
	}
	target := firstStringArg(pkg, call)
	if target == "" {
		target = "<dynamic>"
	}
	return []externalFinding{{
		Kind:     "http_call",
		Target:   target,
		Source:   positionOf(pkg, call.Pos()),
		Evidence: exprText(pkg, call),
	}}
}

func emitCatalog(buf *bytes.Buffer, elementIDs []elementID) {
	buf.WriteString("## Element Catalog\n\n")
	buf.WriteString("| ID | Name | Type | Responsibility | System of Record |\n")
	buf.WriteString("|---|---|---|---|---|\n")
	for _, item := range elementIDs {
		typeCell := typeCellFor(item.Element)
		fmt.Fprintf(buf, "| <a id=\"%s\"></a>%s | %s | %s | %s | %s |\n",
			item.AnchorID, item.ID, item.Element.Name, typeCell,
			item.Element.Responsibility, item.Element.SystemOfRecord)
	}
	buf.WriteString("\n")
}

func emitCrossLinks(buf *bytes.Buffer, links L1CrossLinks) {
	buf.WriteString("## Cross-links\n\n")
	buf.WriteString("- Parent: none (L1 is the root).\n")
	if len(links.RefinedBy) == 0 {
		buf.WriteString("- Refined by: *(none yet)*\n")
		return
	}
	buf.WriteString("- Refined by:\n")
	for _, link := range links.RefinedBy {
		fmt.Fprintf(buf, "  - [`%s`](./%s) — %s\n", link.File, link.File, link.Note)
	}
}

func emitDriftNotes(buf *bytes.Buffer, notes []L1DriftNote) {
	if len(notes) == 0 {
		return
	}
	buf.WriteString("\n## Drift Notes\n\n")
	for _, note := range notes {
		fmt.Fprintf(buf, "- **%s** — %s. Reason: %s.\n", note.Date, note.Detail, note.Reason)
	}
}

func emitFrontMatter(buf *bytes.Buffer, spec *L1Spec, lastReviewedCommit string) {
	parent := "null"
	if spec.Parent != nil {
		parent = strconv.Quote(*spec.Parent)
	}
	fmt.Fprintf(buf, "---\nlevel: %d\nname: %s\nparent: %s\nchildren: []\nlast_reviewed_commit: %s\n---\n",
		spec.Level, spec.Name, parent, lastReviewedCommit)
}

// emitMarkdown renders spec to canonical L1 markdown. lastReviewedCommit is
// inserted into the front-matter; callers compute it (typically `git rev-parse
// --short HEAD`).
func emitMarkdown(w io.Writer, spec *L1Spec, lastReviewedCommit string) error {
	elementIDs, err := assignElementIDs(spec.Elements)
	if err != nil {
		return fmt.Errorf("assign element ids: %w", err)
	}
	nameByID := nameByElementID(elementIDs)
	relsByName := l1RelsWithNames(spec.Relationships, nameByID)
	relIDs := assignRelationshipIDs(relsByName)
	systemName := findSystemName(elementIDs)
	var buf bytes.Buffer
	emitFrontMatter(&buf, spec, lastReviewedCommit)
	fmt.Fprintf(&buf, "\n# C1 — %s (System Context)\n\n%s\n\n", systemName, strings.TrimRight(spec.Preamble, "\n"))
	emitSVGEmbed(&buf, "c1-"+spec.Name, fmt.Sprintf("C1 %s system context", spec.Name))
	emitCatalog(&buf, elementIDs)
	emitRelationships(&buf, elementIDs, relIDs)
	emitCrossLinks(&buf, spec.CrossLinks)
	emitDriftNotes(&buf, spec.DriftNotes)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write markdown: %w", err)
	}
	return nil
}

// emitMermaid writes the canonical L1 mermaid source (with the ELK render
// directive at the top) suitable for writing to architecture/c4/svg/<stem>.mmd
// and rendering to SVG via mmdc.
func emitMermaid(buf *bytes.Buffer, elementIDs []elementID, relIDs []relationshipID) {
	buf.WriteString("%%{init: {'flowchart': {'defaultRenderer': 'elk'}}}%%\n")
	buf.WriteString("flowchart LR\n")
	buf.WriteString("    classDef person      fill:#08427b,stroke:#052e56,color:#fff\n")
	buf.WriteString("    classDef external    fill:#999,   stroke:#666,   color:#fff\n")
	buf.WriteString("    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff\n\n")
	for _, item := range elementIDs {
		shape := mermaidShapeFor(item.Element)
		label := mermaidLabelFor(item)
		mermaidID := strings.ToLower(item.ID)
		fmt.Fprintf(buf, "    %s%s%s%s\n", mermaidID, shape[0], label, shape[1])
	}
	buf.WriteString("\n")
	emitMermaidEdges(buf, relIDs, mermaidIDByName(elementIDs))
	buf.WriteString("\n")
	emitMermaidClasses(buf, elementIDs)
	buf.WriteString("\n")
	emitMermaidClicks(buf, elementIDs)
}

func emitMermaidClasses(buf *bytes.Buffer, elementIDs []elementID) {
	groups := map[string][]string{}
	classOrder := []string{"person", "external", "container"}
	for _, item := range elementIDs {
		class := classFor(item.Element)
		mermaidID := strings.ToLower(item.ID)
		groups[class] = append(groups[class], mermaidID)
	}
	for _, class := range classOrder {
		ids := groups[class]
		if len(ids) == 0 {
			continue
		}
		fmt.Fprintf(buf, "    class %s %s\n", strings.Join(ids, ","), class)
	}
}

func emitMermaidClicks(buf *bytes.Buffer, elementIDs []elementID) {
	for _, item := range elementIDs {
		mermaidID := strings.ToLower(item.ID)
		fmt.Fprintf(buf, "    click %s href \"#%s\" \"%s\"\n", mermaidID, item.AnchorID, item.Element.Name)
	}
}

func emitMermaidEdges(buf *bytes.Buffer, relIDs []relationshipID, idByName map[string]string) {
	for _, rel := range relIDs {
		arrow := "-->"
		if rel.Rel.Bidirectional {
			arrow = "<-->"
		}
		fmt.Fprintf(buf, "    %s %s|\"%s: %s\"| %s\n",
			idByName[rel.Rel.From], arrow, rel.ID, rel.Rel.Description, idByName[rel.Rel.To])
	}
}

func emitRelationships(buf *bytes.Buffer, _ []elementID, relIDs []relationshipID) {
	buf.WriteString("## Relationships\n\n")
	buf.WriteString("| ID | From | To | Description | Protocol/Medium |\n")
	buf.WriteString("|---|---|---|---|---|\n")
	for _, rel := range relIDs {
		fmt.Fprintf(buf, "| <a id=\"%s\"></a>%s | %s | %s | %s | %s |\n",
			rel.AnchorID, rel.ID, rel.Rel.From, rel.Rel.To, rel.Rel.Description, rel.Rel.Protocol)
	}
	buf.WriteString("\n")
}

// emitSVGEmbed writes the markdown stanza that references a pre-rendered SVG
// under architecture/c4/svg/<mmdName>.svg, plus the boilerplate caption
// describing the source .mmd and the re-render command. All four C4 levels
// share this stanza for cross-level visual consistency.
func emitSVGEmbed(buf *bytes.Buffer, mmdName, label string) {
	fmt.Fprintf(buf, "![%s](svg/%s.svg)\n\n", label, mmdName)
	fmt.Fprintf(buf,
		"> Diagram source: [svg/%s.mmd](svg/%s.mmd). Re-render with\n"+
			"> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/%s.mmd -o architecture/c4/svg/%s.svg`.\n"+
			"> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to\n"+
			"> separate bidirectional R/D edges between the same node pair.\n\n",
		mmdName, mmdName, mmdName, mmdName)
}

func exprText(pkg *packages.Package, node ast.Node) string {
	if pkg.Fset == nil {
		return ""
	}
	start := pkg.Fset.Position(node.Pos())
	end := pkg.Fset.Position(node.End())
	if start.Filename != end.Filename {
		return ""
	}
	data, err := os.ReadFile(start.Filename)
	if err != nil {
		return ""
	}
	if start.Offset < 0 || end.Offset > len(data) || start.Offset > end.Offset {
		return ""
	}
	return string(data[start.Offset:end.Offset])
}

// findSystemName returns the name of the unique container element. At L1 the
// system in scope is identified by kind=container; validateElements guarantees
// exactly one such element exists.
func findSystemName(elementIDs []elementID) string {
	for _, item := range elementIDs {
		if item.Element.Kind == "container" {
			return item.Element.Name
		}
	}
	return ""
}

func firstStringArg(pkg *packages.Package, call *ast.CallExpr) string {
	for _, arg := range call.Args {
		value := stringValueOf(pkg, arg)
		if value != "" && (strings.Contains(value, "://") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~")) {
			return value
		}
	}
	return ""
}

// l1RelsWithNames returns a copy of rels where each from/to has been
// substituted from S-ID to display name via nameByID. It preserves the
// original relationship list (not mutated). Used at L1 only — L2/L3 keep
// IDs in from/to to drive their hierarchical anchors.
func l1RelsWithNames(rels []L1Relationship, nameByID map[string]string) []L1Relationship {
	out := make([]L1Relationship, 0, len(rels))
	for _, rel := range rels {
		copied := rel
		copied.From = nameByID[rel.From]
		copied.To = nameByID[rel.To]
		out = append(out, copied)
	}
	return out
}

// loadAndValidateSpec reads a JSON L1 spec from path, decodes it strictly, and
// validates every rule from the design spec. Returns the parsed spec on success.
func loadAndValidateSpec(path string) (*L1Spec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var spec L1Spec
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&spec); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := validateSpec(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

// loadMermaidBlock locates the first diagram source for the markdown at path.
// It first looks for an inline ```mermaid``` fence in raw; if none is present,
// it looks for an SVG embed referencing svg/<stem>.mmd alongside, reads that
// file, and parses it instead. Returns nil block + a single
// mermaid_block_missing finding when neither source can be located. Unlike
// parseMermaidBlock, this does not run any L-level structural collector — the
// caller chooses which collector(s) to apply.
func loadMermaidBlock(raw []byte, path string) (*mermaidBlock, []Finding) {
	// Anchor the fence to the start of a line so literal ```mermaid inside
	// markdown prose / table cells (used as documentation references) is not
	// mistaken for a real code fence.
	openFence := []byte("```mermaid")
	fenceLoc := mermaidFenceRe.FindIndex(raw)
	if fenceLoc != nil {
		idx := fenceLoc[0]
		startLine := 1 + bytes.Count(raw[:idx], []byte("\n"))
		rest := raw[idx+len(openFence):]
		closeFence := []byte("\n```")
		body, _, found := bytes.Cut(rest, closeFence)
		if !found {
			return nil, []Finding{{ID: "mermaid_block_missing", Line: startLine, Detail: "unterminated ```mermaid block"}}
		}
		block := &mermaidBlock{
			startLine: startLine,
			endLine:   startLine + bytes.Count(body, []byte("\n")),
			body:      string(body),
			classes:   map[string]bool{},
		}
		parseMermaidLines(block)
		return block, nil
	}

	// No inline block — try the pre-rendered SVG embed pattern.
	match := svgEmbedRe.FindSubmatchIndex(raw)
	if match == nil {
		return nil, []Finding{{ID: "mermaid_block_missing", Line: 1, Detail: "no ```mermaid fenced block and no svg/<stem>.svg embed"}}
	}
	stem := string(raw[match[2]:match[3]])
	embedLine := 1 + bytes.Count(raw[:match[0]], []byte("\n"))
	mmdPath := filepath.Join(filepath.Dir(path), "svg", stem+".mmd")
	mmdBody, err := os.ReadFile(mmdPath)
	if err != nil {
		return nil, []Finding{{
			ID:     "mermaid_block_missing",
			Line:   embedLine,
			Detail: fmt.Sprintf("svg embed references %s but %s could not be read: %v", stem+".svg", mmdPath, err),
		}}
	}
	block := &mermaidBlock{
		startLine: 1,
		endLine:   1 + bytes.Count(mmdBody, []byte("\n")),
		body:      string(mmdBody),
		classes:   map[string]bool{},
	}
	parseMermaidLines(block)
	return block, nil
}

// mermaidIDByName maps each element's display name to its lowercase mermaid
// node ID. Used at L1 where the Relationships table is rendered by name and
// relIDs have their from/to substituted from S-ID to name before edge emit.
func mermaidIDByName(elementIDs []elementID) map[string]string {
	out := make(map[string]string, len(elementIDs))
	for _, item := range elementIDs {
		out[item.Element.Name] = strings.ToLower(item.ID)
	}
	return out
}

func mermaidLabelFor(item elementID) string {
	label := fmt.Sprintf("%s · %s", item.ID, item.Element.Name)
	if item.Element.Subtitle != nil && *item.Element.Subtitle != "" {
		label = fmt.Sprintf("%s<br/>%s", label, *item.Element.Subtitle)
	}
	return label
}

func mermaidShapeFor(element L1Element) [2]string {
	switch element.Kind {
	case "person":
		return [2]string{"([", "])"}
	case "container":
		return [2]string{"[", "]"}
	default:
		return [2]string{"(", ")"}
	}
}

// nameByElementID maps each element's S-ID to its display name. Used to
// substitute relationship from/to fields before rendering the Relationships
// table and computing anchors.
func nameByElementID(elementIDs []elementID) map[string]string {
	out := make(map[string]string, len(elementIDs))
	for _, item := range elementIDs {
		out[item.ID] = item.Element.Name
	}
	return out
}

// parseFrontMatter scans for a leading --- ... --- block and extracts known
// scalar fields. Returns ok=false when no front-matter block is present.
func parseFrontMatter(raw []byte) (frontMatter, bool) {
	matter := frontMatter{}
	if !bytes.HasPrefix(raw, []byte(fmFence+"\n")) {
		return matter, false
	}
	rest := raw[len(fmFence)+1:]
	closing := []byte("\n" + fmFence + "\n")
	body, _, found := bytes.Cut(rest, closing)
	if !found {
		return matter, false
	}
	matter.present = true
	matter.startLine = 1
	matter.endLine = 2 + bytes.Count(body, []byte("\n"))
	for line, content := range strings.Split(string(body), "\n") {
		consumeFrontMatterLine(&matter, line+2, content)
	}
	return matter, true
}

// parseGitLog walks the NUL-delimited records produced by
// `--format=%H%x09%aI%x09%an%x09%s%x00%B%x00 --name-status`.
// Layout per commit:
//
//	<sha> TAB <date> TAB <author> TAB <subject> NUL <body> NUL <name-status lines>
func parseGitLog(raw []byte) []historyCommit {
	commits := []historyCommit{}
	for len(raw) > 0 {
		commit, rest, ok := parseSingleCommit(raw)
		if !ok {
			break
		}
		commits = append(commits, commit)
		raw = rest
	}
	return commits
}

// parseMermaidBlock locates the first diagram source for the markdown at path
// and runs the L1-style structural collector against it. Use loadMermaidBlock
// when you need the parsed block for a level-specific collector (e.g. L4).
func parseMermaidBlock(raw []byte, path string) (*mermaidBlock, []Finding) {
	block, findings := loadMermaidBlock(raw, path)
	if block != nil {
		findings = append(findings, collectMermaidFindings(block)...)
	}
	return block, findings
}

func parseMermaidLines(block *mermaidBlock) {
	for offset, line := range strings.Split(block.body, "\n") {
		lineNum := block.startLine + offset
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		if matched := mermaidSubgraphRe.FindStringSubmatch(trimmed); matched != nil {
			block.nodes = append(block.nodes, mermaidNode{
				id: matched[1], label: matched[2], line: lineNum,
			})
			continue
		}
		if strings.HasPrefix(trimmed, "classDef ") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				block.hasClassDef = true
				block.classes[fields[1]] = true
			}
			continue
		}
		if matched := mermaidEdgeRe.FindStringSubmatch(trimmed); matched != nil {
			label := strings.Trim(strings.TrimSpace(matched[2]), `"`)
			block.edges = append(block.edges, mermaidEdge{
				from: matched[1], label: label, to: matched[3], line: lineNum,
			})
			continue
		}
		if matched := mermaidNodeRe.FindStringSubmatch(trimmed); matched != nil {
			block.nodes = append(block.nodes, mermaidNode{
				id: matched[1], label: matched[2], line: lineNum,
			})
			continue
		}
		if matched := mermaidClickRe.FindStringSubmatch(trimmed); matched != nil {
			block.clicks = append(block.clicks, mermaidClick{
				node: matched[1], hrefAnchor: matched[2], line: lineNum,
			})
			continue
		}
		if mermaidClassRe.MatchString(trimmed) {
			continue
		}
	}
}

func parseNameStatusBlock(rest *[]byte) []historyFileChange {
	// Skip the leading newline(s) git emits between body-NUL and either the
	// name-status block or the next commit header.
	*rest = bytes.TrimLeft(*rest, "\n")
	files := []historyFileChange{}
	for len(*rest) > 0 {
		newline := bytes.IndexByte(*rest, '\n')
		var line []byte
		if newline < 0 {
			line = *rest
		} else {
			line = (*rest)[:newline]
		}
		if !nameStatusLineRe.Match(line) {
			break
		}
		fields := strings.SplitN(string(line), "\t", 2)
		files = append(files, historyFileChange{Status: fields[0], Path: fields[1]})
		if newline < 0 {
			*rest = nil
			break
		}
		*rest = (*rest)[newline+1:]
	}
	*rest = bytes.TrimLeft(*rest, "\n")
	return files
}

func parseSingleCommit(raw []byte) (historyCommit, []byte, bool) {
	header, rest, ok := bytes.Cut(raw, []byte{0})
	if !ok {
		return historyCommit{}, nil, false
	}
	headerStr := string(bytes.TrimLeft(header, "\n"))
	parts := strings.SplitN(headerStr, "\t", 4)
	if len(parts) < 4 {
		return historyCommit{}, nil, false
	}
	body, rest, ok := bytes.Cut(rest, []byte{0})
	if !ok {
		return historyCommit{}, nil, false
	}
	commit := historyCommit{
		SHA:     parts[0],
		Date:    parts[1],
		Author:  parts[2],
		Subject: parts[3],
		Body:    strings.TrimSpace(string(body)),
	}
	commit.FilesChanged = parseNameStatusBlock(&rest)
	return commit, rest, true
}

func parseTableSection(text string, headerRe *regexp.Regexp, isCatalog bool) []catalogRow {
	loc := headerRe.FindStringIndex(text)
	if loc == nil {
		return nil
	}
	tail := text[loc[1]:]
	startLine := 1 + strings.Count(text[:loc[0]], "\n")
	rows := []catalogRow{}
	for offset, line := range strings.Split(tail, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			break
		}
		matched := tableRowFirstRe.FindStringSubmatch(line)
		if matched == nil {
			continue
		}
		anchor := matched[1]
		if anchor == "" {
			if a := anchorInCellRe.FindStringSubmatch(line); a != nil {
				anchor = a[1]
			}
		}
		// Skip catalog/rels prefix mismatches: catalog has hierarchical
		// S/N/M/P IDs (validated by ParseIDPath), rels has R<n>.
		identifier := matched[2]
		if isCatalog {
			if _, err := ParseIDPath(identifier); err != nil {
				continue
			}
		} else if !strings.HasPrefix(identifier, "R") {
			continue
		}
		rows = append(rows, catalogRow{
			id:       identifier,
			anchorID: anchor,
			line:     startLine + offset + 1,
		})
	}
	return rows
}

// parseTables locates the "## Element Catalog" and "## Relationships" sections
// and parses each table row's first cell to extract the anchor ID and the
// hierarchical element ID or relationship ID.
func parseTables(raw []byte) ([]catalogRow, []relationshipsRow) {
	text := string(raw)
	catalog := parseTableSection(text, catalogHeaderRe, true)
	relsRaw := parseTableSection(text, relsHeaderRe, false)
	rels := make([]relationshipsRow, 0, len(relsRaw))
	for _, row := range relsRaw {
		rels = append(rels, relationshipsRow{
			id: row.id, anchorID: row.anchorID, line: row.line,
		})
	}
	return catalog, rels
}

func positionOf(pkg *packages.Package, pos token.Pos) string {
	if pkg.Fset == nil {
		return ""
	}
	position := pkg.Fset.Position(pos)
	return fmt.Sprintf("%s:%d", position.Filename, position.Line)
}

func scanExternals(ctx context.Context, root, pattern string, includeTests bool) ([]externalFinding, error) {
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedImports |
			packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedTypesInfo |
			packages.NeedName,
		Dir:     root,
		Context: ctx,
		Tests:   includeTests,
	}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}
	findings := []externalFinding{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				findings = append(findings, detectHTTPCall(pkg, node)...)
				findings = append(findings, detectFSPath(pkg, node)...)
				findings = append(findings, detectExec(pkg, node)...)
				findings = append(findings, detectEnvRead(pkg, node)...)
				findings = append(findings, detectDataFormat(pkg, node)...)
				return true
			})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		if findings[i].Source != findings[j].Source {
			return findings[i].Source < findings[j].Source
		}
		return findings[i].Target < findings[j].Target
	})
	return findings, nil
}

func scanHistory(ctx context.Context, opts historyOptions) ([]historyCommit, error) {
	args := []string{
		"log",
		"--format=%H%x09%aI%x09%an%x09%s%x00%B%x00",
		"--name-status",
	}
	if opts.limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", opts.limit))
	}
	if opts.since != "" {
		args = append(args, "--since="+opts.since)
	}
	if opts.grep != "" {
		args = append(args, "--grep="+opts.grep)
	}
	if len(opts.paths) > 0 {
		args = append(args, "--")
		args = append(args, opts.paths...)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = opts.root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	return parseGitLog(out), nil
}

// slug lowercases s and collapses non-[a-z0-9] runs into a single "-",
// trimming leading and trailing "-" runs.
func slug(s string) string {
	lower := strings.ToLower(s)
	collapsed := slugSplitRe.ReplaceAllString(lower, "-")
	return strings.Trim(collapsed, "-")
}

func stringValueOf(pkg *packages.Package, expr ast.Expr) string {
	if pkg.TypesInfo == nil {
		return ""
	}
	if tv, ok := pkg.TypesInfo.Types[expr]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value)
	}
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		unquoted, err := strconv.Unquote(lit.Value)
		if err == nil {
			return unquoted
		}
	}
	return ""
}

// translateSinceShorthand converts compact forms like "30d", "2w", "6m", "1y"
// into the long form `git log --since` accepts ("30 days ago" etc.). Anything
// else is returned unchanged so ISO dates and natural-language phrases pass
// through untouched.
func translateSinceShorthand(input string) string {
	matches := sinceShorthandRe.FindStringSubmatch(input)
	if matches == nil {
		return input
	}
	units := map[string]string{"d": "days", "w": "weeks", "m": "months", "y": "years"}
	return fmt.Sprintf("%s %s ago", matches[1], units[matches[2]])
}

// typeCellFor returns the catalog "Type" cell label. At L1 the unique
// container element is the system in scope; person and external have direct
// mappings.
func typeCellFor(element L1Element) string {
	switch element.Kind {
	case "person":
		return "Person"
	case "container":
		return "The system in scope"
	default:
		return "External system"
	}
}

func validateElements(elements []L1Element) error {
	seenName := map[string]bool{}
	seenID := map[string]bool{}
	for index, element := range elements {
		if element.ID == "" {
			return fmt.Errorf("elements[%d]: missing id", index)
		}
		path, err := ParseIDPath(element.ID)
		if err != nil {
			return fmt.Errorf("elements[%d]: %w", index, err)
		}
		if path.Level != 1 {
			return fmt.Errorf(
				"elements[%d]: id %q must be level 1 (S<n>), got level %d",
				index, element.ID, path.Level,
			)
		}
		if seenID[element.ID] {
			return fmt.Errorf("elements: duplicate id %q", element.ID)
		}
		seenID[element.ID] = true
		if seenName[element.Name] {
			return fmt.Errorf("elements: duplicate name %q", element.Name)
		}
		seenName[element.Name] = true
		if !validKinds[element.Kind] {
			return fmt.Errorf("elements[%d]: kind %q not in {person, external, container}", index, element.Kind)
		}
	}
	if count := containerCount(elements); count != 1 {
		return fmt.Errorf("expected exactly one container (the system in scope), got %d", count)
	}
	return nil
}

func validateRelationships(elements []L1Element, rels []L1Relationship, links L1CrossLinks) error {
	ids := map[string]bool{}
	for _, element := range elements {
		ids[element.ID] = true
	}
	for index, rel := range rels {
		if !ids[rel.From] {
			return fmt.Errorf("relationships[%d]: from %q not in elements", index, rel.From)
		}
		if !ids[rel.To] {
			return fmt.Errorf("relationships[%d]: to %q not in elements", index, rel.To)
		}
	}
	for index, link := range links.RefinedBy {
		if !validRefinedByFile.MatchString(link.File) {
			return fmt.Errorf("cross_links.refined_by[%d]: file %q must match %s",
				index, link.File, validRefinedByFile)
		}
	}
	return nil
}

func validateSpec(spec *L1Spec) error {
	if spec.SchemaVersion != "1" {
		return fmt.Errorf("unknown schema_version %q (want \"1\")", spec.SchemaVersion)
	}
	if spec.Level != 1 {
		return fmt.Errorf("level: want 1, got %d", spec.Level)
	}
	if !validNameRe.MatchString(spec.Name) {
		return fmt.Errorf("name %q must match %s", spec.Name, validNameRe)
	}
	if spec.Parent != nil {
		return fmt.Errorf("parent: must be null at L1, got %q", *spec.Parent)
	}
	if strings.TrimSpace(spec.Preamble) == "" {
		return errors.New("preamble: must be non-empty")
	}
	if err := validateElements(spec.Elements); err != nil {
		return err
	}
	return validateRelationships(spec.Elements, spec.Relationships, spec.CrossLinks)
}

func writeFindingsJSON(w io.Writer, file string, findings []Finding) error {
	if findings == nil {
		findings = []Finding{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(auditJSON{SchemaVersion: "1", File: file, Findings: findings}); err != nil {
		return fmt.Errorf("encode findings: %w", err)
	}
	return nil
}

func writeFindingsText(w io.Writer, file string, findings []Finding) {
	if len(findings) == 0 {
		fmt.Fprintf(w, "%s: clean\n", file)
		return
	}
	fmt.Fprintf(w, "%s: %d finding(s)\n\n", file, len(findings))
	for index, finding := range findings {
		fmt.Fprintf(w, "[%d] %s line %d: %s\n", index+1, finding.ID, finding.Line, finding.Detail)
	}
}
