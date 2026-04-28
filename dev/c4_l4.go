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
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/toejough/targ"
)

func init() {
	targ.Register(targ.Targ(c4L4Build).Name("c4-l4-build").
		Description("Build canonical C4 L4 markdown + mermaid context-strip from a JSON spec next to the input file."))
}

// C4L4BuildArgs configures the c4-l4-build target.
type C4L4BuildArgs struct {
	Input     string `targ:"flag,name=input,desc=JSON spec path (required)"`
	Check     bool   `targ:"flag,name=check,desc=Verify existing .md/.mmd match generated; non-zero on diff"`
	NoConfirm bool   `targ:"flag,name=noconfirm,desc=Overwrite existing .md/.mmd without prompting"`
}

// L4CodeLink is one file:line reference rendered as a markdown link.
type L4CodeLink struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

// L4DepRow is one row of the consumer-side Dependency Manifest table.
type L4DepRow struct {
	Field           string   `json:"field"`
	Type            string   `json:"type"`
	WiredByID       string   `json:"wired_by_id"`
	WiredByName     string   `json:"wired_by_name"`
	WiredByL3       string   `json:"wired_by_l3"`
	WiredByL4       string   `json:"wired_by_l4,omitempty"`
	WrappedEntityID string   `json:"wrapped_entity_id"`
	Properties      []string `json:"properties"`
}

// L4Diagram describes the L4 context-strip mermaid graph.
type L4Diagram struct {
	Nodes []L4Node `json:"nodes"`
	Edges []L4Edge `json:"edges"`
}

// L4Edge is one R edge on the context-strip diagram.
type L4Edge struct {
	ID         string   `json:"id"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Label      string   `json:"label"`
	Properties []string `json:"properties,omitempty"`
}

// L4Focus identifies the L3 component being refined. ID must be a level-3
// (S<n>-N<m>-M<k>) hierarchical path.
type L4Focus struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// L4Node is one node on the context-strip diagram. Kind "focus" gets the focus
// classDef; other kinds match L1/L2/L3 conventions (person/external/container/
// component).
type L4Node struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Subtitle string `json:"subtitle,omitempty"`
	Kind     string `json:"kind"`
}

// L4Property is one row of the property/invariant ledger.
type L4Property struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Statement  string       `json:"statement"`
	EnforcedAt []L4CodeLink `json:"enforced_at"`
	TestedAt   []L4CodeLink `json:"tested_at"`
	Notes      string       `json:"notes,omitempty"`
}

// L4Spec is the JSON-source-of-truth representation of a C4 L4 ledger.
type L4Spec struct {
	SchemaVersion      string        `json:"schema_version"`
	Level              int           `json:"level"`
	Name               string        `json:"name"`
	Parent             string        `json:"parent"`
	Focus              L4Focus       `json:"focus"`
	Sources            []string      `json:"sources"`
	ContextProse       string        `json:"context_prose"`
	LegendItems        []string      `json:"legend_items,omitempty"`
	Diagram            L4Diagram     `json:"diagram"`
	DependencyManifest []L4DepRow    `json:"dependency_manifest,omitempty"`
	DIWires            []L4WireRow   `json:"di_wires,omitempty"`
	Properties         []L4Property  `json:"properties"`
	DriftNotes         []L1DriftNote `json:"drift_notes,omitempty"`
}

// L4WireRow is one row of the provider-side DI Wires table.
type L4WireRow struct {
	Field           string `json:"field"`
	Type            string `json:"type"`
	ConsumerID      string `json:"consumer_id"`
	ConsumerName    string `json:"consumer_name"`
	ConsumerL3      string `json:"consumer_l3"`
	ConsumerL4      string `json:"consumer_l4,omitempty"`
	WrappedEntityID string `json:"wrapped_entity_id"`
}

// unexported variables.
var (
	rEdgeIDPrefix = regexp.MustCompile(`^R\d+$`)
)

func c4L4Build(ctx context.Context, args C4L4BuildArgs) error {
	if args.Input == "" {
		return errors.New("--input is required")
	}
	spec, err := loadAndValidateL4Spec(args.Input)
	if err != nil {
		return err
	}
	sha, shaErr := currentGitShortSHA(ctx)
	if shaErr != nil {
		return fmt.Errorf("git rev-parse: %w", shaErr)
	}
	mdPath := strings.TrimSuffix(args.Input, ".json") + ".md"
	mmdPath := filepath.Join(filepath.Dir(args.Input), "svg",
		strings.TrimSuffix(filepath.Base(args.Input), ".json")+".mmd")
	wiringMmdPath := filepath.Join(filepath.Dir(args.Input), "svg",
		strings.TrimSuffix(filepath.Base(args.Input), ".json")+"-wiring.mmd")
	siblings := discoverL4Siblings(args.Input, spec.Parent)
	var mdBuf bytes.Buffer
	if emitErr := emitL4Markdown(&mdBuf, spec, sha, siblings); emitErr != nil {
		return emitErr
	}
	var mmdBuf bytes.Buffer
	emitL4Mermaid(&mmdBuf, spec)
	if err := writeOrCheckMarkdown(mdPath, mdBuf.Bytes(), args.Check, args.NoConfirm); err != nil {
		return err
	}
	if err := writeOrCheckMarkdown(mmdPath, mmdBuf.Bytes(), args.Check, args.NoConfirm); err != nil {
		return err
	}
	// Skip the wiring mmd entirely when there is nothing to show. The
	// markdown emit applies the same gate (len(manifest) > 0) so the two
	// stay aligned: no wiring section in the markdown, no wiring file on
	// disk.
	if len(spec.DependencyManifest) == 0 {
		return nil
	}
	var wiringBuf bytes.Buffer
	emitL4WiringMermaid(&wiringBuf, spec)
	return writeOrCheckMarkdown(wiringMmdPath, wiringBuf.Bytes(), args.Check, args.NoConfirm)
}

// discoverL4Siblings walks the directory of inputPath and returns relative
// markdown filenames for any c4-*.json whose front-matter `parent` matches the
// caller's parent. Errors are silenced — siblings are best-effort discovery.
func discoverL4Siblings(inputPath, parent string) []string {
	dir := filepath.Dir(inputPath)
	myBase := strings.TrimSuffix(filepath.Base(inputPath), ".json") + ".md"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	siblings := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || name == myBase {
			continue
		}
		if !strings.HasPrefix(name, "c4-") || !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec // dev tool
		if readErr != nil {
			continue
		}
		matter, ok := parseFrontMatter(raw)
		if !ok || matter.parent != parent {
			continue
		}
		siblings = append(siblings, name)
	}
	sort.Strings(siblings)
	return siblings
}

func emitL4ContextSection(buf *bytes.Buffer, spec *L4Spec) {
	buf.WriteString("## Context (from L3)\n\n")
	buf.WriteString(strings.TrimRight(spec.ContextProse, "\n"))
	buf.WriteString("\n\n")
	mmdName := "c4-" + spec.Name
	fmt.Fprintf(buf, "![C4 %s context diagram](svg/%s.svg)\n\n", spec.Name, mmdName)
	fmt.Fprintf(buf,
		"> Diagram source: [svg/%s.mmd](svg/%s.mmd). Re-render with\n"+
			"> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/%s.mmd -o architecture/c4/svg/%s.svg`.\n"+
			"> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to\n"+
			"> separate bidirectional R-edges between the same node pair.\n\n",
		mmdName, mmdName, mmdName, mmdName)
}

func emitL4CrossLinks(buf *bytes.Buffer, spec *L4Spec, siblings []string) {
	buf.WriteString("## Cross-links\n\n")
	fmt.Fprintf(buf, "- Parent: [%s](%s) (refines **%s · %s**)\n",
		spec.Parent, spec.Parent, spec.Focus.ID, spec.Focus.Name)
	if len(siblings) == 0 {
		buf.WriteString("- Siblings: *(none)*\n\n")
	} else {
		buf.WriteString("- Siblings:\n")
		for _, sibling := range siblings {
			fmt.Fprintf(buf, "  - [%s](%s)\n", sibling, sibling)
		}
		buf.WriteString("\n")
	}
	buf.WriteString("See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property\n")
	buf.WriteString("discipline.\n\n")
}

func emitL4DIWires(buf *bytes.Buffer, spec *L4Spec) {
	buf.WriteString("## DI Wires\n\n")
	buf.WriteString("Each row is one DI seam this component wires into a consumer. Reciprocal entries\n")
	buf.WriteString("live in the consumer's L4 under \"Dependency Manifest\".\n\n")
	buf.WriteString("| Field | Type | Consumer | Wrapped entity |\n")
	buf.WriteString("|---|---|---|---|\n")
	for _, row := range spec.DIWires {
		emitL4WireRow(buf, row)
	}
	buf.WriteString("\n")
}

func emitL4DepRow(buf *bytes.Buffer, row L4DepRow) {
	wiredBy := fmt.Sprintf("[%s · %s](%s#%s)",
		row.WiredByID, row.WiredByName, row.WiredByL3, Anchor(row.WiredByID, row.WiredByName))
	if row.WiredByL4 != "" {
		wiredBy += fmt.Sprintf(" ([%s](%s))", row.WiredByL4, row.WiredByL4)
	}
	fmt.Fprintf(buf, "| `%s` | `%s` | %s | `%s` | %s |\n",
		row.Field, row.Type, wiredBy, row.WrappedEntityID, formatPropertyList(row.Properties))
}

func emitL4DependencyManifest(buf *bytes.Buffer, spec *L4Spec) {
	buf.WriteString("## Dependency Manifest\n\n")
	buf.WriteString("Each row is one DI seam the focus consumes. The wrapped entity is the diagram\n")
	buf.WriteString("node (component or external) the seam ultimately drives behavior against; it\n")
	buf.WriteString("must also appear on the call diagram. The wiring diagram dedupes manifest\n")
	buf.WriteString("rows by wrapped entity.\n\n")
	buf.WriteString("| Field | Type | Wired by | Wrapped entity | Properties |\n")
	buf.WriteString("|---|---|---|---|---|\n")
	for _, row := range spec.DependencyManifest {
		emitL4DepRow(buf, row)
	}
	buf.WriteString("\n")
}

func emitL4FocusBlockquote(buf *bytes.Buffer, spec *L4Spec) {
	fmt.Fprintf(buf, "> Component in focus: **%s · %s**.\n",
		spec.Focus.ID, spec.Focus.Name)
	if len(spec.Sources) == 0 {
		buf.WriteString("\n")
		return
	}
	buf.WriteString("> Source files in scope:\n")
	for _, src := range spec.Sources {
		fmt.Fprintf(buf, "> - [%s](%s)\n", src, src)
	}
	buf.WriteString("\n")
}

func emitL4FrontMatter(buf *bytes.Buffer, spec *L4Spec, lastReviewedCommit string) {
	fmt.Fprintf(buf,
		"---\nlevel: %d\nname: %s\nparent: %s\nchildren: []\nlast_reviewed_commit: %s\n---\n",
		spec.Level, spec.Name, strconv.Quote(spec.Parent), lastReviewedCommit)
}

func emitL4Legend(buf *bytes.Buffer, spec *L4Spec) {
	buf.WriteString("**Legend:**\n")
	for _, item := range spec.LegendItems {
		fmt.Fprintf(buf, "- %s\n", item)
	}
	buf.WriteString("\n")
}

func emitL4Markdown(w io.Writer, spec *L4Spec, lastReviewedCommit string, siblings []string) error {
	var buf bytes.Buffer
	emitL4FrontMatter(&buf, spec, lastReviewedCommit)
	fmt.Fprintf(&buf, "\n# C4 — %s (Property/Invariant Ledger)\n\n", spec.Name)
	emitL4FocusBlockquote(&buf, spec)
	emitL4ContextSection(&buf, spec)
	if len(spec.LegendItems) > 0 {
		emitL4Legend(&buf, spec)
	}
	if len(spec.DependencyManifest) > 0 {
		emitL4WiringSection(&buf, spec)
		emitL4DependencyManifest(&buf, spec)
	}
	if len(spec.DIWires) > 0 {
		emitL4DIWires(&buf, spec)
	}
	emitL4PropertyLedger(&buf, spec)
	emitL4CrossLinks(&buf, spec, siblings)
	emitDriftNotes(&buf, spec.DriftNotes)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write markdown: %w", err)
	}
	return nil
}

func emitL4Mermaid(buf *bytes.Buffer, spec *L4Spec) {
	buf.WriteString("%%{init: {'flowchart': {'defaultRenderer': 'elk'}}}%%\n")
	buf.WriteString("flowchart LR\n")
	buf.WriteString("    classDef person      fill:#08427b,stroke:#052e56,color:#fff\n")
	buf.WriteString("    classDef external    fill:#999,   stroke:#666,   color:#fff\n")
	buf.WriteString("    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff\n")
	buf.WriteString("    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000\n")
	buf.WriteString("    classDef focus       fill:#facc15,stroke:#a16207,color:#000\n\n")
	for _, node := range spec.Diagram.Nodes {
		emitL4MermaidNode(buf, node)
	}
	if len(spec.Diagram.Edges) > 0 {
		buf.WriteString("\n")
	}
	diTargets := wrappedEntityIDSet(spec)
	for _, edge := range spec.Diagram.Edges {
		emitL4MermaidEdge(buf, edge, diTargets)
	}
	buf.WriteString("\n")
	emitL4MermaidClasses(buf, spec)
}

func emitL4MermaidClasses(buf *bytes.Buffer, spec *L4Spec) {
	groups := map[string][]string{}
	classOrder := []string{"person", "external", "container", "component", "focus"}
	for _, node := range spec.Diagram.Nodes {
		groups[node.Kind] = append(groups[node.Kind], strings.ToLower(node.ID))
	}
	for _, class := range classOrder {
		ids := groups[class]
		if len(ids) == 0 {
			continue
		}
		fmt.Fprintf(buf, "    class %s %s\n", strings.Join(ids, ","), class)
	}
}

// emitL4MermaidClassesForNodes writes class assignments restricted to the
// referenced node set, mirroring emitL4MermaidClasses' structure.
func emitL4MermaidClassesForNodes(buf *bytes.Buffer, spec *L4Spec, referenced map[string]bool) {
	groups := map[string][]string{}
	for _, n := range spec.Diagram.Nodes {
		if !referenced[n.ID] {
			continue
		}
		groups[n.Kind] = append(groups[n.Kind], strings.ToLower(n.ID))
	}
	for _, class := range []string{"person", "external", "container", "component", "focus"} {
		ids := groups[class]
		if len(ids) == 0 {
			continue
		}
		fmt.Fprintf(buf, "    class %s %s\n", strings.Join(ids, ","), class)
	}
}

func emitL4MermaidEdge(buf *bytes.Buffer, edge L4Edge, diTargets map[string]bool) {
	from := strings.ToLower(edge.From)
	to := strings.ToLower(edge.To)
	label := fmt.Sprintf("%s: %s", edge.ID, edge.Label)
	if len(edge.Properties) > 0 {
		label = fmt.Sprintf("%s [%s]", label, strings.Join(edge.Properties, ", "))
	}
	arrow := "-->"
	if diTargets[edge.To] {
		arrow = "-.->"
	}
	fmt.Fprintf(buf, "    %s %s|%q| %s\n", from, arrow, label, to)
}

func emitL4MermaidNode(buf *bytes.Buffer, node L4Node) {
	label := fmt.Sprintf("%s · %s", node.ID, node.Name)
	if node.Subtitle != "" {
		label = fmt.Sprintf("%s<br/>%s", label, node.Subtitle)
	}
	open, close := l4NodeShape(node.Kind)
	mermaidID := strings.ToLower(node.ID)
	// Wrap label in quotes so subtitles may contain parens / brackets without
	// breaking the mermaid parser. Mermaid recognises "..." inside any shape.
	fmt.Fprintf(buf, "    %s%s\"%s\"%s\n", mermaidID, open, label, close)
}

func emitL4PropertyLedger(buf *bytes.Buffer, spec *L4Spec) {
	buf.WriteString("## Property Ledger\n\n")
	buf.WriteString("| ID | Property | Statement | Enforced at | Tested at | Notes |\n")
	buf.WriteString("|---|---|---|---|---|---|\n")
	for _, prop := range spec.Properties {
		emitL4PropertyRow(buf, prop)
	}
	buf.WriteString("\n")
}

func emitL4PropertyRow(buf *bytes.Buffer, prop L4Property) {
	enforcedCell := formatLinkList(prop.EnforcedAt)
	testedCell := formatLinkList(prop.TestedAt)
	if testedCell == "" {
		testedCell = "**⚠ UNTESTED**"
	}
	notes := prop.Notes
	if notes == "" {
		notes = " "
	}
	fmt.Fprintf(buf, "| <a id=\"%s\"></a>%s | %s | %s | %s | %s | %s |\n",
		Anchor(prop.ID, prop.Name),
		prop.ID, prop.Name, prop.Statement, enforcedCell, testedCell, notes)
}

func emitL4WireRow(buf *bytes.Buffer, row L4WireRow) {
	consumer := fmt.Sprintf("[%s · %s](%s#%s)",
		row.ConsumerID, row.ConsumerName, row.ConsumerL3, Anchor(row.ConsumerID, row.ConsumerName))
	if row.ConsumerL4 != "" {
		consumer += fmt.Sprintf(" ([%s](%s))", row.ConsumerL4, row.ConsumerL4)
	}
	fmt.Fprintf(buf, "| `%s` | `%s` | %s | `%s` |\n",
		row.Field, row.Type, consumer, row.WrappedEntityID)
}

// emitL4WiringMermaid emits the L4 wiring diagram: wirer→focus edges,
// one per (wirer, wrapped_entity) pair, label = WrappedEntityID. Multiple
// manifest rows that share both fields collapse into a single edge.
func emitL4WiringMermaid(buf *bytes.Buffer, spec *L4Spec) {
	buf.WriteString("%%{init: {'flowchart': {'defaultRenderer': 'elk'}}}%%\n")
	buf.WriteString("flowchart LR\n")
	buf.WriteString("    classDef person      fill:#08427b,stroke:#052e56,color:#fff\n")
	buf.WriteString("    classDef external    fill:#999,   stroke:#666,   color:#fff\n")
	buf.WriteString("    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff\n")
	buf.WriteString("    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000\n")
	buf.WriteString("    classDef focus       fill:#facc15,stroke:#a16207,color:#000\n\n")

	// Collect distinct nodes referenced by the wiring diagram. Only wirers
	// and the focus appear as nodes; wrapped entities are conveyed by edge
	// labels (their SNM IDs) and would otherwise float disconnected.
	referenced := map[string]bool{spec.Focus.ID: true}
	type edgeKey struct{ wirer, wrapped string }
	seen := map[edgeKey]bool{}
	edges := []edgeKey{}
	for _, row := range spec.DependencyManifest {
		referenced[row.WiredByID] = true
		key := edgeKey{row.WiredByID, row.WrappedEntityID}
		if seen[key] {
			continue
		}
		seen[key] = true
		edges = append(edges, key)
	}

	// Emit nodes in deterministic order (the order they appear on the call diagram).
	for _, n := range spec.Diagram.Nodes {
		if !referenced[n.ID] {
			continue
		}
		emitL4MermaidNode(buf, n)
	}
	buf.WriteString("\n")

	// Emit deduped wiring edges.
	for _, e := range edges {
		from := strings.ToLower(e.wirer)
		to := strings.ToLower(spec.Focus.ID)
		fmt.Fprintf(buf, "    %s -->|%q| %s\n", from, e.wrapped, to)
	}
	buf.WriteString("\n")

	emitL4MermaidClassesForNodes(buf, spec, referenced)
}

func emitL4WiringSection(buf *bytes.Buffer, spec *L4Spec) {
	fmt.Fprintf(buf, "## Wiring\n\n")
	fmt.Fprintf(buf,
		"Each edge is one or more DI seams the wirer plugs into %s, deduped by the\n"+
			"wrapped entity (label = SNM ID). The Dependency Manifest below shows the\n"+
			"per-seam breakdown.\n\n",
		spec.Focus.Name)
	fmt.Fprintf(buf, "![C4 %s wiring diagram](svg/c4-%s-wiring.svg)\n\n", spec.Name, spec.Name)
	fmt.Fprintf(buf,
		"> Diagram source: [svg/c4-%s-wiring.mmd](svg/c4-%s-wiring.mmd). Re-render with\n"+
			"> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-%s-wiring.mmd "+
			"-o architecture/c4/svg/c4-%s-wiring.svg`.\n\n",
		spec.Name, spec.Name, spec.Name, spec.Name)
}

func formatFirstLink(link L4CodeLink) string {
	if link.Line == 0 {
		return fmt.Sprintf("[%s](../../%s)", link.Path, link.Path)
	}
	return fmt.Sprintf("[%s:%d](../../%s#L%d)", link.Path, link.Line, link.Path, link.Line)
}

// formatLinkList renders a slice of CodeLinks as comma-separated markdown links.
// First link uses full path text; subsequent links use ":line" shorthand.
// Mirrors the hand-authored convention seen in c4-tokenresolver.md.
func formatLinkList(links []L4CodeLink) string {
	if len(links) == 0 {
		return ""
	}
	parts := make([]string, 0, len(links))
	parts = append(parts, formatFirstLink(links[0]))
	for _, link := range links[1:] {
		parts = append(parts, formatNextLink(link))
	}
	return strings.Join(parts, ", ")
}

func formatNextLink(link L4CodeLink) string {
	if link.Line == 0 {
		return fmt.Sprintf("[%s](../../%s)", link.Path, link.Path)
	}
	return fmt.Sprintf("[:%d](../../%s#L%d)", link.Line, link.Path, link.Line)
}

// formatPropertyList collapses contiguous P-id runs into ranges, e.g.
// ["S2-N3-M3-P2","S2-N3-M3-P3","S2-N3-M3-P4"] -> "S2-N3-M3-P2–P4".
// IDs must all share the same prefix (everything before the last "-P<n>").
// If prefixes differ or parsing fails, items are joined with ", ".
func formatPropertyList(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	const minRunLength = 3
	type entry struct {
		prefix string
		num    int
		raw    string
	}
	entries := make([]entry, 0, len(ids))
	for _, id := range ids {
		lastDash := strings.LastIndex(id, "-P")
		if lastDash < 0 {
			// bare P<n> (legacy or unexpected) — fall back
			num, err := strconv.Atoi(strings.TrimPrefix(id, "P"))
			if err != nil {
				return strings.Join(ids, ", ")
			}
			entries = append(entries, entry{prefix: "", num: num, raw: id})
			continue
		}
		prefix := id[:lastDash]
		suffix := id[lastDash+2:] // skip "-P"
		num, err := strconv.Atoi(suffix)
		if err != nil {
			return strings.Join(ids, ", ")
		}
		entries = append(entries, entry{prefix: prefix, num: num, raw: id})
	}
	// Verify all share the same prefix
	sharedPrefix := entries[0].prefix
	for _, ent := range entries[1:] {
		if ent.prefix != sharedPrefix {
			return strings.Join(ids, ", ")
		}
	}
	nums := make([]int, 0, len(entries))
	for _, ent := range entries {
		nums = append(nums, ent.num)
	}
	sort.Ints(nums)
	var groups []string
	runStart := 0
	pPrefix := sharedPrefix + "-P"
	if sharedPrefix == "" {
		pPrefix = "P"
	}
	for index := 1; index <= len(nums); index++ {
		if index < len(nums) && nums[index] == nums[index-1]+1 {
			continue
		}
		runLen := index - runStart
		if runLen >= minRunLength {
			groups = append(groups, fmt.Sprintf("%s%d–P%d", pPrefix, nums[runStart], nums[index-1]))
		} else {
			for inner := runStart; inner < index; inner++ {
				groups = append(groups, fmt.Sprintf("%s%d", pPrefix, nums[inner]))
			}
		}
		runStart = index
	}
	return strings.Join(groups, ", ")
}

// kindsMatch reports whether an L4 node kind is compatible with the L3
// element kind. The L4 focus has kind "focus" but the L3 element it
// refines has kind "component"; for that one ID the comparison relaxes.
func kindsMatch(nodeID, l4Kind, l3Kind, focusID string) bool {
	if nodeID == focusID && l4Kind == "focus" && l3Kind == "component" {
		return true
	}
	return l4Kind == l3Kind
}

func l4NodeShape(kind string) (string, string) {
	switch kind {
	case "person":
		return "([", "])"
	case "external":
		return "(", ")"
	default:
		return "[", "]"
	}
}

func loadAndValidateL4Spec(path string) (*L4Spec, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // dev tool
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var spec L4Spec
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&spec); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	l3, err := loadL3Parent(&spec, filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	if err := validateL4Spec(&spec, l3); err != nil {
		return nil, fmt.Errorf("validating %s: %w", path, err)
	}
	return &spec, nil
}

// loadL3Parent reads the L3 spec sibling of an L4 spec from dirPath. The
// filename is derived from l4.Parent by replacing the .md suffix with .json.
func loadL3Parent(l4 *L4Spec, dirPath string) (*L3Spec, error) {
	parentJSON := strings.TrimSuffix(l4.Parent, ".md") + ".json"
	fullPath := filepath.Join(dirPath, parentJSON)
	raw, err := os.ReadFile(fullPath) //nolint:gosec // dev tool
	if err != nil {
		return nil, fmt.Errorf("loading L3 parent %q: %w", parentJSON, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var spec L3Spec
	if err := decoder.Decode(&spec); err != nil {
		return nil, fmt.Errorf("decoding L3 parent %q: %w", parentJSON, err)
	}
	return &spec, nil
}

// sharesParentPath reports whether two same-depth paths share all but the last
// segment (i.e. are siblings under the same parent).
func sharesParentPath(a, b IDPath) bool {
	if len(a.Segments) != len(b.Segments) || len(a.Segments) == 0 {
		return false
	}
	for index := range a.Segments[:len(a.Segments)-1] {
		if a.Segments[index] != b.Segments[index] {
			return false
		}
	}
	return true
}

// validateL4Carryover enforces the L4↔L3 cross-level invariant. Both
// directions are checked; violations after the focus-existence check are
// aggregated via errors.Join.
//
// The L4 focus is rendered with kind "focus" but corresponds to a
// "component" on the L3 parent — that one ID receives a relaxed kind
// comparison.
func validateL4Carryover(l4 *L4Spec, l3 *L3Spec) error {
	l3ByID := map[string]L3Element{}
	for _, el := range l3.Elements {
		l3ByID[el.ID] = el
	}
	if _, ok := l3ByID[l4.Focus.ID]; !ok {
		return fmt.Errorf("focus.id %q: not present on L3 parent %q", l4.Focus.ID, l4.Parent)
	}

	var errs []error
	for i, node := range l4.Diagram.Nodes {
		l3el, ok := l3ByID[node.ID]
		if !ok {
			errs = append(errs, fmt.Errorf("diagram.nodes[%d] %q: not present on L3 parent %q",
				i, node.ID, l4.Parent))
			continue
		}
		if !kindsMatch(node.ID, node.Kind, l3el.Kind, l4.Focus.ID) {
			errs = append(errs, fmt.Errorf("diagram.nodes[%d] %q: kind %q does not match L3 parent kind %q",
				i, node.ID, node.Kind, l3el.Kind))
		}
	}

	l4Nodes := map[string]bool{}
	for _, node := range l4.Diagram.Nodes {
		l4Nodes[node.ID] = true
	}
	connected := map[string]bool{}
	for _, rel := range l3.Relationships {
		switch {
		case rel.From == l4.Focus.ID && rel.To != l4.Focus.ID:
			connected[rel.To] = true
		case rel.To == l4.Focus.ID && rel.From != l4.Focus.ID:
			connected[rel.From] = true
		}
	}
	connectedIDs := make([]string, 0, len(connected))
	for id := range connected {
		connectedIDs = append(connectedIDs, id)
	}
	sort.Strings(connectedIDs)
	for _, id := range connectedIDs {
		if !l4Nodes[id] {
			errs = append(errs, fmt.Errorf("L3 parent %q has node %q connected to focus %q, but %q is missing from L4 diagram.nodes",
				l4.Parent, id, l4.Focus.ID, id))
		}
	}
	return errors.Join(errs...)
}

// validateL4Manifest verifies that every dependency-manifest row's
// wrapped_entity_id matches a node on the diagram. The wrapped entity is the
// label that appears on the wiring diagram, so it must be a node we can render.
func validateL4Manifest(spec *L4Spec) error {
	known := map[string]bool{}
	for _, node := range spec.Diagram.Nodes {
		known[node.ID] = true
	}
	for index, row := range spec.DependencyManifest {
		if row.WrappedEntityID == "" {
			return fmt.Errorf("dependency_manifest[%d]: wrapped_entity_id must be non-empty", index)
		}
		if !known[row.WrappedEntityID] {
			return fmt.Errorf(
				"dependency_manifest[%d]: wrapped_entity_id %q does not match any diagram node",
				index, row.WrappedEntityID)
		}
	}
	return nil
}

// validateL4NodeIDs validates that every diagram node has an explicit
// hierarchical path ID and that edge IDs match the R<n> convention.
// Node IDs must satisfy one of:
//   - equals the focus (level 3, S<n>-N<m>-M<k>)
//   - is an ancestor of the focus (level 1 or 2)
//   - is a sibling of the focus: same depth (level 3) AND shares all but the
//     last segment with the focus (e.g. S2-N3-M5 is a sibling of S2-N3-M3)
//
// All violations are aggregated into one error.
func validateL4NodeIDs(spec *L4Spec) error {
	focusPath, err := ParseIDPath(spec.Focus.ID)
	if err != nil {
		return fmt.Errorf("focus.id: %w", err)
	}
	violations := []string{}
	for index, edge := range spec.Diagram.Edges {
		if !rEdgeIDPrefix.MatchString(edge.ID) {
			violations = append(violations, fmt.Sprintf(
				"diagram.edges[%d].id %q: must match R<n> (call relationship)",
				index, edge.ID))
		}
	}
	for index, node := range spec.Diagram.Nodes {
		if nodeErr := ValidateDiagramNodeID(focusPath, node.ID); nodeErr != nil {
			violations = append(violations, fmt.Sprintf("diagram.nodes[%d].id: %v", index, nodeErr))
		}
	}
	if len(violations) == 0 {
		return nil
	}
	return fmt.Errorf("L4 id validation failed:\n  - %s", strings.Join(violations, "\n  - "))
}

// validateL4PropertiesWithFocus validates each property ID is a level-4 path
// (S<n>-N<m>-M<k>-P<j>) directly under the focus, and that the P<j> segment
// matches the 1-based array index.
func validateL4PropertiesWithFocus(focusPath IDPath, props []L4Property) error {
	seenID := map[string]bool{}
	for index, prop := range props {
		if err := ValidateElementID(4, focusPath, prop.ID); err != nil {
			return fmt.Errorf("properties[%d]: %w", index, err)
		}
		path, _ := ParseIDPath(prop.ID) // safe: ValidateElementID already validated
		if path.Level != 4 {
			return fmt.Errorf("properties[%d]: id %q must be level 4 (S<n>-N<m>-M<k>-P<j>), got level %d",
				index, prop.ID, path.Level)
		}
		expectedSuffix := fmt.Sprintf("P%d", index+1)
		if path.Segments[3] != expectedSuffix {
			return fmt.Errorf("properties[%d]: id %q last segment must be %s (index+1)",
				index, prop.ID, expectedSuffix)
		}
		if seenID[prop.ID] {
			return fmt.Errorf("properties[%d]: duplicate id %q", index, prop.ID)
		}
		seenID[prop.ID] = true
		if strings.TrimSpace(prop.Name) == "" {
			return fmt.Errorf("properties[%d]: name must be non-empty", index)
		}
		if strings.TrimSpace(prop.Statement) == "" {
			return fmt.Errorf("properties[%d]: statement must be non-empty", index)
		}
		if len(prop.EnforcedAt) == 0 {
			return fmt.Errorf("properties[%d]: enforced_at must have at least one link", index)
		}
	}
	return nil
}

func validateL4Spec(spec *L4Spec, l3 *L3Spec) error {
	if spec.SchemaVersion != "1" {
		return fmt.Errorf("unknown schema_version %q (want \"1\")", spec.SchemaVersion)
	}
	if spec.Level != 4 {
		return fmt.Errorf("level: want 4, got %d", spec.Level)
	}
	if !validNameRe.MatchString(spec.Name) {
		return fmt.Errorf("name %q must match %s", spec.Name, validNameRe)
	}
	if strings.TrimSpace(spec.Parent) == "" {
		return errors.New("parent: must be non-empty at L4")
	}
	focusPath, err := ParseIDPath(spec.Focus.ID)
	if err != nil {
		return fmt.Errorf("focus.id %q must be a hierarchical path: %w", spec.Focus.ID, err)
	}
	if focusPath.Level != 3 {
		return fmt.Errorf("focus.id %q must be level 3 (S<n>-N<m>-M<k>), got level %d",
			spec.Focus.ID, focusPath.Level)
	}
	if strings.TrimSpace(spec.Focus.Name) == "" {
		return errors.New("focus.name: must be non-empty")
	}
	if strings.TrimSpace(spec.ContextProse) == "" {
		return errors.New("context_prose: must be non-empty")
	}
	if err := validateL4NodeIDs(spec); err != nil {
		return err
	}
	if err := validateL4Manifest(spec); err != nil {
		return err
	}
	if err := validateL4PropertiesWithFocus(focusPath, spec.Properties); err != nil {
		return err
	}
	if l3 != nil {
		if err := validateL4Carryover(spec, l3); err != nil {
			return err
		}
	}
	return nil
}

// wrappedEntityIDSet returns the set of entity IDs that appear as
// wrapped_entity_id in the dependency manifest. R-edges whose `to` is
// in this set are rendered dotted (the call goes through a DI seam).
func wrappedEntityIDSet(spec *L4Spec) map[string]bool {
	out := map[string]bool{}
	for _, row := range spec.DependencyManifest {
		out[row.WrappedEntityID] = true
	}
	return out
}
