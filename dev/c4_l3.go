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
	"sort"
	"strconv"
	"strings"

	"github.com/toejough/targ"
)

func init() {
	targ.Register(targ.Targ(c4L3Build).Name("c4-l3-build").
		Description("Build canonical C4 L3 markdown from a JSON spec next to the input file."))
}

// C4L3BuildArgs configures the c4-l3-build target.
type C4L3BuildArgs struct {
	Input     string `targ:"flag,name=input,desc=JSON spec path (required)"`
	Check     bool   `targ:"flag,name=check,desc=Verify existing .md matches generated; non-zero on diff"`
	NoConfirm bool   `targ:"flag,name=noconfirm,desc=Overwrite existing .md without prompting"`
}

// L3Element is one element on an L3 diagram. Components live inside the focus
// subgraph; carry-over neighbors (kind person/external/container) live outside.
// Each element has an explicit hierarchical ID: carried-over L1/L2 elements
// use their parent S<n> or S<n>-N<m> ID; new components under the focus use
// <focusPath>-M<k>. The focus must have a level-2 (S<n>-N<m>) ID.
type L3Element struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	Subtitle       *string `json:"subtitle,omitempty"`
	Responsibility string  `json:"responsibility"`
	CodePointer    string  `json:"code_pointer,omitempty"`
}

// L3Focus identifies the parent L2 container being refined. ID + Name together
// make the focus subgraph and the L3 file's first catalog row. The L3 file
// owns the catalog row; the parent L2 owns the canonical definition.
type L3Focus struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Responsibility string `json:"responsibility,omitempty"`
}

// L3Spec is the JSON-source-of-truth representation of a C4 L3 diagram.
type L3Spec struct {
	SchemaVersion string           `json:"schema_version"`
	Level         int              `json:"level"`
	Name          string           `json:"name"`
	Parent        string           `json:"parent"`
	Preamble      string           `json:"preamble"`
	Focus         L3Focus          `json:"focus"`
	Elements      []L3Element      `json:"elements"`
	Relationships []L1Relationship `json:"relationships"`
	DriftNotes    []L1DriftNote    `json:"drift_notes"`
	CrossLinks    L1CrossLinks     `json:"cross_links"`
}

func c4L3Build(ctx context.Context, args C4L3BuildArgs) error {
	if args.Input == "" {
		return errors.New("--input is required")
	}
	spec, err := loadAndValidateL3Spec(args.Input)
	if err != nil {
		return err
	}
	sha, shaErr := currentGitShortSHA(ctx)
	if shaErr != nil {
		return fmt.Errorf("git rev-parse: %w", shaErr)
	}
	outPath := strings.TrimSuffix(args.Input, ".json") + ".md"
	siblings := discoverL3Siblings(args.Input, spec.Parent)
	var buf bytes.Buffer
	if emitErr := emitL3Markdown(&buf, spec, sha, siblings); emitErr != nil {
		return emitErr
	}
	return writeOrCheckMarkdown(outPath, buf.Bytes(), args.Check, args.NoConfirm)
}

// discoverL3Siblings returns relative-path entries for any other c3-*.md whose
// front-matter `parent` matches this spec's parent. Read-only; sibling files
// are not edited. Errors are silenced — siblings are best-effort discovery.
func discoverL3Siblings(inputPath, parent string) []string {
	dir := filepath.Dir(inputPath)
	myBase := strings.TrimSuffix(filepath.Base(inputPath), ".json") + ".md"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	siblings := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == myBase {
			continue
		}
		if !strings.HasPrefix(name, "c3-") || !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec // path under inputPath dir
		if err != nil {
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

// emitL3Catalog emits the Element Catalog with a Code Pointer column. The
// focus container is always the first row, followed by every element in spec
// order.
func emitL3Catalog(buf *bytes.Buffer, spec *L3Spec) {
	buf.WriteString("## Element Catalog\n\n")
	buf.WriteString("| ID | Name | Type | Responsibility | Code Pointer |\n")
	buf.WriteString("|---|---|---|---|---|\n")
	focusResp := spec.Focus.Responsibility
	if focusResp == "" {
		focusResp = "Container in focus — refined from " + spec.Parent + "."
	}
	fmt.Fprintf(buf, "| <a id=\"%s\"></a>%s | %s | Container in focus | %s | — |\n",
		Anchor(spec.Focus.ID, spec.Focus.Name), spec.Focus.ID, spec.Focus.Name, focusResp)
	for _, element := range spec.Elements {
		emitL3CatalogRow(buf, element)
	}
	buf.WriteString("\n")
}

func emitL3CatalogRow(buf *bytes.Buffer, element L3Element) {
	typeCell := l3TypeCell(element.Kind)
	codePointerCell := "—"
	if element.CodePointer != "" {
		codePointerCell = fmt.Sprintf("[%s](%s)", element.CodePointer, element.CodePointer)
	}
	fmt.Fprintf(buf, "| <a id=\"%s\"></a>%s | %s | %s | %s | %s |\n",
		Anchor(element.ID, element.Name),
		element.ID,
		element.Name,
		typeCell,
		element.Responsibility,
		codePointerCell)
}

// emitL3CrossLinks emits the Cross-links section: parent reference (always),
// auto-discovered siblings (other c3-*.md with same parent), and refined-by
// L4 entries from spec.
func emitL3CrossLinks(buf *bytes.Buffer, spec *L3Spec, siblings []string) {
	buf.WriteString("## Cross-links\n\n")
	fmt.Fprintf(buf, "- Parent: [%s](%s) (refines **%s · %s**)\n",
		spec.Parent, spec.Parent, spec.Focus.ID, spec.Focus.Name)
	if len(siblings) == 0 {
		buf.WriteString("- Siblings: *(none)*\n")
	} else {
		buf.WriteString("- Siblings:\n")
		for _, sibling := range siblings {
			fmt.Fprintf(buf, "  - [%s](%s)\n", sibling, sibling)
		}
	}
	if len(spec.CrossLinks.RefinedBy) == 0 {
		buf.WriteString("- Refined by: *(none yet)*\n")
		return
	}
	buf.WriteString("- Refined by:\n")
	for _, link := range spec.CrossLinks.RefinedBy {
		fmt.Fprintf(buf, "  - [`%s`](./%s) — %s\n", link.File, link.File, link.Note)
	}
}

// emitL3FocusSubgraph emits the wrapping subgraph for the focus container,
// containing every component element.
func emitL3FocusSubgraph(buf *bytes.Buffer, spec *L3Spec) {
	scopeID := strings.ToLower(spec.Focus.ID)
	fmt.Fprintf(buf, "    subgraph %s [%s · %s]\n", scopeID, spec.Focus.ID, spec.Focus.Name)
	for _, element := range spec.Elements {
		if element.Kind != "component" {
			continue
		}
		label := l3MermaidLabel(element.ID, element.Name, element.Subtitle)
		mermaidID := strings.ToLower(element.ID)
		fmt.Fprintf(buf, "        %s[%s]\n", mermaidID, label)
	}
	buf.WriteString("    end\n")
}

func emitL3FrontMatter(buf *bytes.Buffer, spec *L3Spec, lastReviewedCommit string) {
	fmt.Fprintf(buf,
		"---\nlevel: %d\nname: %s\nparent: %s\nchildren: []\nlast_reviewed_commit: %s\n---\n",
		spec.Level, spec.Name, strconv.Quote(spec.Parent), lastReviewedCommit)
}

// emitL3Markdown renders spec to canonical L3 markdown.
func emitL3Markdown(w io.Writer, spec *L3Spec, lastReviewedCommit string, siblings []string) error {
	if _, err := validateL3ElementIDs(spec); err != nil {
		return fmt.Errorf("validate l3 element ids: %w", err)
	}
	var buf bytes.Buffer
	emitL3FrontMatter(&buf, spec, lastReviewedCommit)
	fmt.Fprintf(&buf, "\n# C3 — %s (Component)\n\n%s\n\n",
		spec.Focus.Name, strings.TrimRight(spec.Preamble, "\n"))
	emitL3Mermaid(&buf, spec)
	emitL3Catalog(&buf, spec)
	emitL3Relationships(&buf, spec)
	emitL3CrossLinks(&buf, spec, siblings)
	emitDriftNotes(&buf, spec.DriftNotes)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write markdown: %w", err)
	}
	return nil
}

func emitL3Mermaid(buf *bytes.Buffer, spec *L3Spec) {
	buf.WriteString("```mermaid\nflowchart LR\n")
	buf.WriteString("    classDef person      fill:#08427b,stroke:#052e56,color:#fff\n")
	buf.WriteString("    classDef external    fill:#999,   stroke:#666,   color:#fff\n")
	buf.WriteString("    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff\n")
	buf.WriteString("    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000\n\n")

	emitL3NeighborNodes(buf, spec)
	buf.WriteString("\n")
	emitL3FocusSubgraph(buf, spec)
	buf.WriteString("\n")
	emitL3MermaidEdges(buf, spec)
	buf.WriteString("\n")
	emitL3MermaidClasses(buf, spec)
	buf.WriteString("\n")
	emitL3MermaidClicks(buf, spec)
	buf.WriteString("```\n\n")
}

// emitL3MermaidClasses emits class statements grouping mermaid IDs by class.
// The focus subgraph is always assigned `container`. Other elements get the
// class matching their kind.
func emitL3MermaidClasses(buf *bytes.Buffer, spec *L3Spec) {
	groups := map[string][]string{}
	classOrder := []string{"person", "external", "container", "component"}
	for _, element := range spec.Elements {
		groups[element.Kind] = append(groups[element.Kind], strings.ToLower(element.ID))
	}
	for _, class := range classOrder {
		ids := groups[class]
		if len(ids) == 0 {
			continue
		}
		fmt.Fprintf(buf, "    class %s %s\n", strings.Join(ids, ","), class)
	}
	fmt.Fprintf(buf, "    class %s container\n", strings.ToLower(spec.Focus.ID))
}

func emitL3MermaidClicks(buf *bytes.Buffer, spec *L3Spec) {
	fmt.Fprintf(buf, "    click %s href \"#%s\" \"%s\"\n",
		strings.ToLower(spec.Focus.ID), Anchor(spec.Focus.ID, spec.Focus.Name), spec.Focus.Name)
	for _, element := range spec.Elements {
		fmt.Fprintf(buf, "    click %s href \"#%s\" \"%s\"\n",
			strings.ToLower(element.ID), Anchor(element.ID, element.Name), element.Name)
	}
}

func emitL3MermaidEdges(buf *bytes.Buffer, spec *L3Spec) {
	idByElementID := map[string]string{
		spec.Focus.ID: strings.ToLower(spec.Focus.ID),
	}
	for _, element := range spec.Elements {
		idByElementID[element.ID] = strings.ToLower(element.ID)
	}
	relIDs := assignRelationshipIDs(spec.Relationships)
	emitMermaidEdges(buf, relIDs, idByElementID)
}

// emitL3NeighborNodes emits one mermaid node per non-component element. These
// sit outside the focus subgraph.
func emitL3NeighborNodes(buf *bytes.Buffer, spec *L3Spec) {
	for _, element := range spec.Elements {
		if element.Kind == "component" {
			continue
		}
		shape := mermaidShapeFor(L1Element{Kind: element.Kind})
		label := l3MermaidLabel(element.ID, element.Name, element.Subtitle)
		mermaidID := strings.ToLower(element.ID)
		fmt.Fprintf(buf, "    %s%s%s%s\n", mermaidID, shape[0], label, shape[1])
	}
}

func emitL3Relationships(buf *bytes.Buffer, spec *L3Spec) {
	relIDs := assignRelationshipIDs(spec.Relationships)
	emitRelationships(buf, nil, relIDs)
}

func l3MermaidLabel(id, name string, subtitle *string) string {
	label := fmt.Sprintf("%s · %s", id, name)
	if subtitle != nil && *subtitle != "" {
		label = fmt.Sprintf("%s<br/>%s", label, *subtitle)
	}
	return label
}

func l3TypeCell(kind string) string {
	switch kind {
	case "person":
		return "Person"
	case "external":
		return "External system"
	case "container":
		return "Container"
	case "component":
		return "Component"
	default:
		return kind
	}
}

// loadAndValidateL3Spec reads a JSON L3 spec from path, decodes strictly, and
// validates the L3 schema rules.
func loadAndValidateL3Spec(path string) (*L3Spec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var spec L3Spec
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&spec); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := validateL3Spec(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

// validateL3ElementIDs validates that the focus has a level-2 (S<n>-N<m>) ID
// and that each element has an explicit hierarchical path ID. Elements may be:
//   - L1 paths (S<n>): carried-over system-level peers.
//   - L2 paths (S<n>-N<m>): carried-over containers; equals focus or peers.
//   - L3 paths (S<n>-N<m>-M<k>) under the focus: new components.
//
// On success returns elementIDs with anchors of the form
// "<lower-id>-<slug(name)>" (e.g. "s2-n3-m9-tokenresolver").
func validateL3ElementIDs(spec *L3Spec) ([]elementID, error) {
	focusPath, err := ParseIDPath(spec.Focus.ID)
	if err != nil {
		return nil, fmt.Errorf("focus.id: %w", err)
	}
	if focusPath.Level != 2 {
		return nil, fmt.Errorf("focus.id %q must be level 2 (S<n>-N<m>), got level %d",
			spec.Focus.ID, focusPath.Level)
	}
	result := make([]elementID, 0, len(spec.Elements))
	used := map[string]int{}
	for _, element := range spec.Elements {
		if err := ValidateElementID(3, focusPath, element.ID); err != nil {
			return nil, fmt.Errorf("element %q: %w", element.Name, err)
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
			Element: L1Element{
				ID:             element.ID,
				Name:           element.Name,
				Kind:           element.Kind,
				Subtitle:       element.Subtitle,
				Responsibility: element.Responsibility,
			},
		})
	}
	return result, nil
}

func validateL3Elements(spec *L3Spec) error {
	seenName := map[string]bool{spec.Focus.Name: true}
	seenID := map[string]bool{spec.Focus.ID: true}
	for index, element := range spec.Elements {
		if err := validateL3SingleElement(index, element); err != nil {
			return err
		}
		if seenName[element.Name] {
			return fmt.Errorf("elements: duplicate name %q (focus or another element)", element.Name)
		}
		seenName[element.Name] = true
		if seenID[element.ID] {
			return fmt.Errorf("elements: duplicate id %q (focus or another element)", element.ID)
		}
		seenID[element.ID] = true
	}
	return validateL3Relationships(spec)
}

func validateL3Relationships(spec *L3Spec) error {
	ids := map[string]bool{spec.Focus.ID: true}
	for _, element := range spec.Elements {
		ids[element.ID] = true
	}
	for index, rel := range spec.Relationships {
		if !ids[rel.From] {
			return fmt.Errorf("relationships[%d]: from %q not in elements or focus", index, rel.From)
		}
		if !ids[rel.To] {
			return fmt.Errorf("relationships[%d]: to %q not in elements or focus", index, rel.To)
		}
	}
	return nil
}

func validateL3SingleElement(index int, element L3Element) error {
	if _, err := ParseIDPath(element.ID); err != nil {
		return fmt.Errorf("elements[%d]: id %q must be a hierarchical path (S<n>, S<n>-N<m>, or S<n>-N<m>-M<k>): %w",
			index, element.ID, err)
	}
	if strings.TrimSpace(element.Name) == "" {
		return fmt.Errorf("elements[%d]: name must be non-empty", index)
	}
	switch element.Kind {
	case "component":
		if strings.TrimSpace(element.CodePointer) == "" {
			return fmt.Errorf("elements[%d]: kind=component requires code_pointer", index)
		}
	case "person", "external", "container":
		if strings.TrimSpace(element.CodePointer) != "" {
			return fmt.Errorf("elements[%d]: code_pointer is only valid for kind=component", index)
		}
	default:
		return fmt.Errorf("elements[%d]: kind %q not in {person, external, container, component}",
			index, element.Kind)
	}
	return nil
}

func validateL3Spec(spec *L3Spec) error {
	if spec.SchemaVersion != "1" {
		return fmt.Errorf("unknown schema_version %q (want \"1\")", spec.SchemaVersion)
	}
	if spec.Level != 3 {
		return fmt.Errorf("level: want 3, got %d", spec.Level)
	}
	if !validNameRe.MatchString(spec.Name) {
		return fmt.Errorf("name %q must match %s", spec.Name, validNameRe)
	}
	if strings.TrimSpace(spec.Parent) == "" {
		return errors.New("parent: must be non-empty at L3")
	}
	if strings.TrimSpace(spec.Preamble) == "" {
		return errors.New("preamble: must be non-empty")
	}
	if _, err := ParseIDPath(spec.Focus.ID); err != nil {
		return fmt.Errorf("focus.id %q must be a hierarchical path: %w", spec.Focus.ID, err)
	}
	if strings.TrimSpace(spec.Focus.Name) == "" {
		return errors.New("focus.name: must be non-empty")
	}
	return validateL3Elements(spec)
}

// writeOrCheckMarkdown reuses the same write/check semantics as
// c4-l1-build / c4-l2-build for the L3 build.
func writeOrCheckMarkdown(outPath string, content []byte, check, noConfirm bool) error {
	if check {
		existing, readErr := os.ReadFile(outPath)
		if readErr != nil {
			return fmt.Errorf("read existing %s: %w", outPath, readErr)
		}
		if !bytes.Equal(existing, content) {
			fmt.Fprintln(os.Stderr, "diff between source-of-truth JSON and rendered markdown:")
			return errors.New("markdown out of sync with JSON")
		}
		return nil
	}
	if !noConfirm {
		existing, readErr := os.ReadFile(outPath)
		if readErr == nil && !bytes.Equal(existing, content) {
			fmt.Fprintf(os.Stderr, "%s already exists and differs. Overwrite? [y/N] ", outPath)
			var resp string
			_, _ = fmt.Fscanln(os.Stdin, &resp)
			if !strings.EqualFold(resp, "y") {
				return errors.New("aborted")
			}
		}
	}
	if err := os.WriteFile(outPath, content, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}
