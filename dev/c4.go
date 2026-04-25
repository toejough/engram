//go:build targ

package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/toejough/targ"
)

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

type L1Element struct {
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	IsSystem       bool    `json:"is_system,omitempty"`
	Subtitle       *string `json:"subtitle,omitempty"`
	Responsibility string  `json:"responsibility"`
	SystemOfRecord string  `json:"system_of_record"`
}

type L1Relationship struct {
	From          string `json:"from"`
	To            string `json:"to"`
	Description   string `json:"description"`
	Protocol      string `json:"protocol"`
	Bidirectional bool   `json:"bidirectional,omitempty"`
}

type L1DriftNote struct {
	Date   string `json:"date"`
	Detail string `json:"detail"`
	Reason string `json:"reason"`
}

type L1CrossLinks struct {
	RefinedBy []L1RefinedBy `json:"refined_by"`
}

type L1RefinedBy struct {
	File string `json:"file"`
	Note string `json:"note"`
}

// C4AuditArgs configures the c4-audit target.
type C4AuditArgs struct {
	File string `targ:"flag,name=file,desc=Markdown file to audit (required)"`
	JSON bool   `targ:"flag,name=json,desc=Emit findings as JSON"`
}

// C4L1BuildArgs configures the c4-l1-build target.
type C4L1BuildArgs struct {
	Input     string `targ:"flag,name=input,desc=JSON spec path (required)"`
	Check     bool   `targ:"flag,name=check,desc=Verify existing .md matches generated; non-zero on diff"`
	NoConfirm bool   `targ:"flag,name=noconfirm,desc=Overwrite existing .md without prompting"`
}

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

// Finding records a single structural problem detected by the audit.
type Finding struct {
	ID     string `json:"id"`
	Line   int    `json:"line"`
	Detail string `json:"detail"`
}

type auditJSON struct {
	SchemaVersion string    `json:"schema_version"`
	File          string    `json:"file"`
	Findings      []Finding `json:"findings"`
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
	hasLastReviewedCommit  bool
	lastReviewedCommit     string
	lastReviewedCommitLine int
}

const fmFence = "---"

var (
	errNoArgs    = errors.New("--file is required")
	validNameRe  = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	slugSplitRe  = regexp.MustCompile(`[^a-z0-9]+`)
)

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
	outPath := strings.TrimSuffix(args.Input, ".json") + ".md"
	var buf bytes.Buffer
	if err := emitMarkdown(&buf, spec, sha); err != nil {
		return err
	}
	if args.Check {
		existing, readErr := os.ReadFile(outPath)
		if readErr != nil {
			return fmt.Errorf("read existing %s: %w", outPath, readErr)
		}
		if !bytes.Equal(existing, buf.Bytes()) {
			fmt.Fprintln(os.Stderr, "diff between source-of-truth JSON and rendered markdown:")
			return errors.New("markdown out of sync with JSON")
		}
		return nil
	}
	if !args.NoConfirm {
		existing, readErr := os.ReadFile(outPath)
		if readErr == nil && !bytes.Equal(existing, buf.Bytes()) {
			fmt.Fprintf(os.Stderr, "%s already exists and differs. Overwrite? [y/N] ", outPath)
			var resp string
			_, _ = fmt.Fscanln(os.Stdin, &resp)
			if !strings.EqualFold(resp, "y") {
				return errors.New("aborted")
			}
		}
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}

func currentGitShortSHA(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func c4L1Externals(_ context.Context) error { return errors.New("c4-l1-externals: not implemented") }
func c4History(_ context.Context) error     { return errors.New("c4-history: not implemented") }

// auditFile reads path and returns all structural findings.
// It returns an error only when the file itself cannot be read.
func auditFile(ctx context.Context, path string) ([]Finding, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	findings := []Finding{}
	findings = append(findings, auditFrontMatter(ctx, path, raw)...)
	block, mermaidFindings := parseMermaidBlock(raw)
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
	return findings
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
	case "last_reviewed_commit":
		matter.hasLastReviewedCommit = true
		matter.lastReviewedCommitLine = lineNum
		matter.lastReviewedCommit = strings.Trim(value, `"'`)
	}
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

var levelPrefixRe = regexp.MustCompile(`^c[1-4]-`)

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

type mermaidNode struct {
	id    string
	label string
	line  int
}

type mermaidEdge struct {
	from  string
	to    string
	label string
	line  int
}

type mermaidClick struct {
	node       string
	hrefAnchor string
	line       int
}

var (
	mermaidNodeRe   = regexp.MustCompile(`^\s*(\w+)\s*[(\[]+(.*?)[)\]]+\s*$`)
	mermaidEdgeRe   = regexp.MustCompile(`^\s*(\w+)\s*<?-+>+\s*(?:\|([^|]*)\|\s*)?(\w+)\s*$`)
	mermaidClickRe  = regexp.MustCompile(`^\s*click\s+(\w+)\s+href\s+"#([^"]+)"`)
	mermaidClassRe  = regexp.MustCompile(`^\s*class\s+([\w,]+)\s+(\w+)\s*$`)
	mermaidIDPrefix = regexp.MustCompile(`^E\d+`)
	edgeIDPrefix    = regexp.MustCompile(`^R\d+\s*:`)
)

// parseMermaidBlock locates the first ```mermaid fenced block in raw and parses
// its contents. Returns nil block + a single mermaid_block_missing finding when
// no block is present. Returns parsed-block findings (classdef/node/edge id) too.
func parseMermaidBlock(raw []byte) (*mermaidBlock, []Finding) {
	openFence := []byte("```mermaid")
	idx := bytes.Index(raw, openFence)
	if idx < 0 {
		return nil, []Finding{{ID: "mermaid_block_missing", Line: 1, Detail: "no ```mermaid fenced block"}}
	}
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
	return block, collectMermaidFindings(block)
}

func parseMermaidLines(block *mermaidBlock) {
	for offset, line := range strings.Split(block.body, "\n") {
		lineNum := block.startLine + offset
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
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
			block.edges = append(block.edges, mermaidEdge{
				from: matched[1], label: matched[2], to: matched[3], line: lineNum,
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
				Detail: fmt.Sprintf("node %q label %q does not start with E<n>", node.id, node.label),
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
				Detail: fmt.Sprintf("edge %q->%q label %q does not start with R<n>:", edge.from, edge.to, edge.label),
			})
		}
	}
	return findings
}

type catalogRow struct {
	id       string
	anchorID string
	name     string
	line     int
}

type relationshipsRow struct {
	id       string
	anchorID string
	from     string
	to       string
	line     int
}

var (
	catalogHeaderRe = regexp.MustCompile(`(?m)^##\s+Element Catalog\s*$`)
	relsHeaderRe    = regexp.MustCompile(`(?m)^##\s+Relationships\s*$`)
	tableRowFirstRe = regexp.MustCompile(`^\s*\|\s*(?:<a\s+id="([^"]+)"></a>)?\s*([ER]\d+)\s*\|`)
	anchorInCellRe  = regexp.MustCompile(`<a\s+id="([^"]+)"></a>`)
)

// parseTables locates the "## Element Catalog" and "## Relationships" sections
// and parses each table row's first cell to extract the anchor ID and the
// E<n>/R<n> identifier.
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
		// Skip catalog/rels prefix mismatches: catalog has E<n>, rels has R<n>.
		identifier := matched[2]
		if isCatalog && !strings.HasPrefix(identifier, "E") {
			continue
		}
		if !isCatalog && !strings.HasPrefix(identifier, "R") {
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

// auditAnchorsAndClicks emits click_missing for nodes lacking a click directive,
// click_target_unresolved for click anchors that don't match any catalog or
// relationships row anchor, and anchor_missing for table rows lacking an anchor.
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
	clickedNodes := map[string]bool{}
	for _, click := range block.clicks {
		clickedNodes[click.node] = true
		if !anchorSet[click.hrefAnchor] {
			findings = append(findings, Finding{
				ID: "click_target_unresolved", Line: click.line,
				Detail: fmt.Sprintf("click %s href #%s does not match any catalog/relationships anchor",
					click.node, click.hrefAnchor),
			})
		}
	}
	for _, node := range block.nodes {
		if !clickedNodes[node.id] {
			findings = append(findings, Finding{
				ID: "click_missing", Line: node.line,
				Detail: fmt.Sprintf("node %s has no click directive", node.id),
			})
		}
	}
	return findings
}

var (
	validKinds          = map[string]bool{"person": true, "external": true, "container": true}
	validRefinedByFile  = regexp.MustCompile(`^c2-[a-z0-9-]+\.md$`)
)

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

func validateElements(elements []L1Element) error {
	systemCount := 0
	seen := map[string]bool{}
	for _, element := range elements {
		if element.IsSystem {
			systemCount++
		}
		if seen[element.Name] {
			return fmt.Errorf("elements: duplicate name %q", element.Name)
		}
		seen[element.Name] = true
	}
	if systemCount != 1 {
		return fmt.Errorf("expected exactly one is_system: true, got %d", systemCount)
	}
	for index, element := range elements {
		if !validKinds[element.Kind] {
			return fmt.Errorf("elements[%d]: kind %q not in {person, external, container}", index, element.Kind)
		}
		if element.IsSystem && element.Kind != "container" {
			return fmt.Errorf("elements[%d]: is_system=true requires kind=container, got %q", index, element.Kind)
		}
	}
	return nil
}

func validateRelationships(elements []L1Element, rels []L1Relationship, links L1CrossLinks) error {
	names := map[string]bool{}
	for _, element := range elements {
		names[element.Name] = true
	}
	for index, rel := range rels {
		if !names[rel.From] {
			return fmt.Errorf("relationships[%d]: from %q not in elements", index, rel.From)
		}
		if !names[rel.To] {
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

// elementID pairs a source element with its assigned E<n> identifier and
// anchor slug.
type elementID struct {
	ID       string
	AnchorID string
	Element  L1Element
}

// relationshipID pairs a source relationship with its assigned R<n> identifier
// and anchor slug.
type relationshipID struct {
	ID       string
	AnchorID string
	Rel      L1Relationship
}

// assignElementIDs returns elements paired with sequential E1..En IDs in source
// order. AnchorID is "e<n>-<slug(name)>"; collisions append "-2", "-3"....
func assignElementIDs(elements []L1Element) []elementID {
	result := make([]elementID, 0, len(elements))
	used := map[string]int{}
	for index, element := range elements {
		base := fmt.Sprintf("e%d-%s", index+1, slug(element.Name))
		anchor := base
		if used[base] > 0 {
			used[base]++
			anchor = fmt.Sprintf("%s-%d", base, used[base])
		} else {
			used[base] = 1
		}
		result = append(result, elementID{
			ID:       fmt.Sprintf("E%d", index+1),
			AnchorID: anchor,
			Element:  element,
		})
	}
	return result
}

// assignRelationshipIDs returns relationships paired with sequential R1..Rn IDs.
// AnchorID is "r<n>-<slug(from)>-<slug(to)>"; collisions append "-2"....
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

// emitMarkdown renders spec to canonical L1 markdown. lastReviewedCommit is
// inserted into the front-matter; callers compute it (typically `git rev-parse
// --short HEAD`).
func emitMarkdown(w io.Writer, spec *L1Spec, lastReviewedCommit string) error {
	elementIDs := assignElementIDs(spec.Elements)
	relIDs := assignRelationshipIDs(spec.Relationships)
	systemName := findSystemName(elementIDs)
	var buf bytes.Buffer
	emitFrontMatter(&buf, spec, lastReviewedCommit)
	fmt.Fprintf(&buf, "\n# C1 — %s (System Context)\n\n%s\n\n", systemName, strings.TrimRight(spec.Preamble, "\n"))
	emitMermaid(&buf, elementIDs, relIDs)
	emitCatalog(&buf, elementIDs)
	emitRelationships(&buf, elementIDs, relIDs)
	emitCrossLinks(&buf, spec.CrossLinks)
	emitDriftNotes(&buf, spec.DriftNotes)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write markdown: %w", err)
	}
	return nil
}

func findSystemName(elementIDs []elementID) string {
	for _, item := range elementIDs {
		if item.Element.IsSystem {
			return item.Element.Name
		}
	}
	return ""
}

func emitFrontMatter(buf *bytes.Buffer, spec *L1Spec, lastReviewedCommit string) {
	parent := "null"
	if spec.Parent != nil {
		parent = strconv.Quote(*spec.Parent)
	}
	fmt.Fprintf(buf, "---\nlevel: %d\nname: %s\nparent: %s\nchildren: []\nlast_reviewed_commit: %s\n---\n",
		spec.Level, spec.Name, parent, lastReviewedCommit)
}

func emitMermaid(buf *bytes.Buffer, elementIDs []elementID, relIDs []relationshipID) {
	buf.WriteString("```mermaid\nflowchart LR\n")
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
	buf.WriteString("```\n\n")
}

func mermaidShapeFor(element L1Element) [2]string {
	switch {
	case element.Kind == "person":
		return [2]string{"([", "])"}
	case element.IsSystem:
		return [2]string{"[", "]"}
	case element.Kind == "container":
		return [2]string{"[", "]"}
	default:
		return [2]string{"(", ")"}
	}
}

func mermaidLabelFor(item elementID) string {
	label := fmt.Sprintf("%s · %s", item.ID, item.Element.Name)
	if item.Element.Subtitle != nil && *item.Element.Subtitle != "" {
		label = fmt.Sprintf("%s<br/>%s", label, *item.Element.Subtitle)
	}
	return label
}

func mermaidIDByName(elementIDs []elementID) map[string]string {
	out := map[string]string{}
	for _, item := range elementIDs {
		out[item.Element.Name] = strings.ToLower(item.ID)
	}
	return out
}

func emitMermaidEdges(buf *bytes.Buffer, relIDs []relationshipID, idByName map[string]string) {
	for _, rel := range relIDs {
		arrow := "-->"
		if rel.Rel.Bidirectional {
			arrow = "<-->"
		}
		fmt.Fprintf(buf, "    %s %s|%s: %s| %s\n",
			idByName[rel.Rel.From], arrow, rel.ID, rel.Rel.Description, idByName[rel.Rel.To])
	}
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

func classFor(element L1Element) string {
	if element.Kind == "person" {
		return "person"
	}
	if element.Kind == "container" {
		return "container"
	}
	return "external"
}

func emitMermaidClicks(buf *bytes.Buffer, elementIDs []elementID) {
	for _, item := range elementIDs {
		mermaidID := strings.ToLower(item.ID)
		fmt.Fprintf(buf, "    click %s href \"#%s\" \"%s\"\n", mermaidID, item.AnchorID, item.Element.Name)
	}
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

func typeCellFor(element L1Element) string {
	switch {
	case element.IsSystem:
		return "The system in scope"
	case element.Kind == "person":
		return "Person"
	case element.Kind == "container":
		return "Container"
	default:
		return "External system"
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

// slug lowercases s and collapses non-[a-z0-9] runs into a single "-",
// trimming leading and trailing "-" runs.
func slug(s string) string {
	lower := strings.ToLower(s)
	collapsed := slugSplitRe.ReplaceAllString(lower, "-")
	return strings.Trim(collapsed, "-")
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
