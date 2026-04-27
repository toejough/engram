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
	ConcreteAdapter string   `json:"concrete_adapter"`
	Properties      []string `json:"properties"`
}

// L4Diagram describes the L4 context-strip mermaid graph.
type L4Diagram struct {
	Nodes []L4Node `json:"nodes"`
	Edges []L4Edge `json:"edges"`
}

// L4Edge is one R or D edge on the context-strip diagram.
type L4Edge struct {
	ID     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
	Label  string `json:"label"`
	Dotted bool   `json:"dotted,omitempty"`
}

// L4Focus identifies the L3 component being refined.
type L4Focus struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	L3Container string `json:"l3_container"`
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
	WiredAdapter  string `json:"wired_adapter"`
	ConcreteValue string `json:"concrete_value"`
	ConsumerID    string `json:"consumer_id"`
	ConsumerName  string `json:"consumer_name"`
	ConsumerL3    string `json:"consumer_l3"`
	ConsumerL4    string `json:"consumer_l4,omitempty"`
	ConsumerField string `json:"consumer_field"`
}

// unexported variables.
var (
	dEdgeIDPrefix    = regexp.MustCompile(`^[RD]\d+$`)
	propertyIDPrefix = regexp.MustCompile(`^P\d+$`)
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
	return writeOrCheckMarkdown(mmdPath, mmdBuf.Bytes(), args.Check, args.NoConfirm)
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
			"> separate bidirectional R/D edges between the same node pair.\n\n",
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
	buf.WriteString("Each row is one adapter this component wires into a consumer. Reciprocal entries\n")
	buf.WriteString("live in the consumer's L4 under \"Dependency Manifest\".\n\n")
	buf.WriteString("| Wired adapter | Concrete value | Consumer | Consumer field |\n")
	buf.WriteString("|---|---|---|---|\n")
	for _, row := range spec.DIWires {
		emitL4WireRow(buf, row)
	}
	buf.WriteString("\n")
}

func emitL4DepRow(buf *bytes.Buffer, row L4DepRow) {
	wiredBy := fmt.Sprintf("[%s · %s](%s#%s)",
		row.WiredByID, row.WiredByName, row.WiredByL3, l3AnchorID(row.WiredByID, row.WiredByName))
	if row.WiredByL4 != "" {
		wiredBy += fmt.Sprintf(" ([%s](%s))", row.WiredByL4, row.WiredByL4)
	} else {
		wiredBy += fmt.Sprintf(" (L4: c4-%s.md — TBD)", row.WiredByName)
	}
	fmt.Fprintf(buf, "| `%s` | `%s` | %s | %s | %s |\n",
		row.Field, row.Type, wiredBy, row.ConcreteAdapter, formatPropertyList(row.Properties))
}

func emitL4DependencyManifest(buf *bytes.Buffer, spec *L4Spec) {
	buf.WriteString("## Dependency Manifest\n\n")
	buf.WriteString("Each row is one injected dependency the focus component receives. Manifest expands the\n")
	buf.WriteString("Rdi back-edge into per-dep wiring rows. Reciprocal entries live in the wirer's L4 under\n")
	buf.WriteString("\"DI Wires\" — those two sections must stay in sync.\n\n")
	buf.WriteString("| Dep field | Type | Wired by | Concrete adapter | Properties |\n")
	buf.WriteString("|---|---|---|---|---|\n")
	for _, row := range spec.DependencyManifest {
		emitL4DepRow(buf, row)
	}
	buf.WriteString("\n")
}

func emitL4FocusBlockquote(buf *bytes.Buffer, spec *L4Spec) {
	fmt.Fprintf(buf, "> Component in focus: **%s · %s** (refines L3 %s).\n",
		spec.Focus.ID, spec.Focus.Name, spec.Focus.L3Container)
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
	for _, edge := range spec.Diagram.Edges {
		emitL4MermaidEdge(buf, edge)
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

func emitL4MermaidEdge(buf *bytes.Buffer, edge L4Edge) {
	from := strings.ToLower(edge.From)
	to := strings.ToLower(edge.To)
	arrow := "-->"
	if edge.Dotted {
		arrow = "-.->"
	}
	label := fmt.Sprintf("%s: %s", edge.ID, edge.Label)
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
		l4PropertyAnchor(prop.ID, prop.Name),
		prop.ID, prop.Name, prop.Statement, enforcedCell, testedCell, notes)
}

func emitL4WireRow(buf *bytes.Buffer, row L4WireRow) {
	consumer := fmt.Sprintf("[%s · %s](%s#%s)",
		row.ConsumerID, row.ConsumerName, row.ConsumerL3, l3AnchorID(row.ConsumerID, row.ConsumerName))
	if row.ConsumerL4 != "" {
		consumer += fmt.Sprintf(" ([%s](%s))", row.ConsumerL4, row.ConsumerL4)
	}
	fmt.Fprintf(buf, "| %s | %s | %s | `%s` |\n",
		row.WiredAdapter, row.ConcreteValue, consumer, row.ConsumerField)
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
// ["P2","P3","P4","P5","P6","P7","P8"] -> "P2–P8".
// Single items and non-contiguous lists are joined with ", ".
func formatPropertyList(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	const minRunLength = 3
	nums := make([]int, 0, len(ids))
	for _, id := range ids {
		num, err := strconv.Atoi(strings.TrimPrefix(id, "P"))
		if err != nil {
			return strings.Join(ids, ", ")
		}
		nums = append(nums, num)
	}
	sort.Ints(nums)
	var groups []string
	runStart := 0
	for index := 1; index <= len(nums); index++ {
		if index < len(nums) && nums[index] == nums[index-1]+1 {
			continue
		}
		runLen := index - runStart
		if runLen >= minRunLength {
			groups = append(groups, fmt.Sprintf("P%d–P%d", nums[runStart], nums[index-1]))
		} else {
			for inner := runStart; inner < index; inner++ {
				groups = append(groups, fmt.Sprintf("P%d", nums[inner]))
			}
		}
		runStart = index
	}
	return strings.Join(groups, ", ")
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

// l4PropertyAnchor returns the lowercase HTML anchor ID for a property row,
// e.g. "p1-env-precedence".
func l4PropertyAnchor(id, name string) string {
	number := strings.TrimPrefix(id, "P")
	return fmt.Sprintf("p%s-%s", number, slug(name))
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
	if err := validateL4Spec(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

func validateL4Properties(props []L4Property) error {
	seenID := map[string]bool{}
	for index, prop := range props {
		if !propertyIDPrefix.MatchString(prop.ID) {
			return fmt.Errorf("properties[%d]: id %q must match P<n>", index, prop.ID)
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

func validateL4Spec(spec *L4Spec) error {
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
	if !mermaidIDPrefix.MatchString(spec.Focus.ID) {
		return fmt.Errorf("focus.id %q must match E<n>", spec.Focus.ID)
	}
	if strings.TrimSpace(spec.Focus.Name) == "" {
		return errors.New("focus.name: must be non-empty")
	}
	if strings.TrimSpace(spec.Focus.L3Container) == "" {
		return errors.New("focus.l3_container: must be non-empty")
	}
	if strings.TrimSpace(spec.ContextProse) == "" {
		return errors.New("context_prose: must be non-empty")
	}
	return validateL4Properties(spec.Properties)
}
