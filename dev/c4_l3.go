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
// All elements carry an explicit ID — no auto-assignment.
type L3Element struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	Subtitle       *string `json:"subtitle,omitempty"`
	Responsibility string  `json:"responsibility"`
	CodePointer    string  `json:"code_pointer,omitempty"`
	FromParent     bool    `json:"from_parent,omitempty"`
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
	if regErr := validateL3SpecAgainstRegistry(ctx, args.Input, spec); regErr != nil {
		return regErr
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

func checkFocusAgainstRegistry(focus L3Focus, elementByID map[string]RegistryElement) error {
	entry, ok := elementByID[focus.ID]
	if !ok {
		// No peer registers this ID — accept (the parent L2 may not be in this
		// directory yet, e.g. during bootstrap). Audit will catch the markdown
		// later if the parent is missing.
		return nil
	}
	if !containsString(entry.Names, focus.Name) {
		return fmt.Errorf("focus.name %q for %s conflicts with registry %v",
			focus.Name, focus.ID, entry.Names)
	}
	return nil
}

func checkL3ElementAgainstRegistry(element L3Element, elementByID map[string]RegistryElement) error {
	entry, ok := elementByID[element.ID]
	if !ok {
		return nil
	}
	if element.FromParent {
		if !containsString(entry.Names, element.Name) {
			return fmt.Errorf("element %s name %q does not match registry %v (from_parent)",
				element.ID, element.Name, entry.Names)
		}
		return nil
	}
	if !containsString(entry.Names, element.Name) {
		return fmt.Errorf(
			"element %s name %q collides with existing registry IDs (%v) — pick a different ID",
			element.ID, element.Name, entry.Names)
	}
	return nil
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
		l3AnchorID(spec.Focus.ID, spec.Focus.Name), spec.Focus.ID, spec.Focus.Name, focusResp)
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
		l3AnchorID(element.ID, element.Name),
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
		strings.ToLower(spec.Focus.ID), l3AnchorID(spec.Focus.ID, spec.Focus.Name), spec.Focus.Name)
	for _, element := range spec.Elements {
		fmt.Fprintf(buf, "    click %s href \"#%s\" \"%s\"\n",
			strings.ToLower(element.ID), l3AnchorID(element.ID, element.Name), element.Name)
	}
}

func emitL3MermaidEdges(buf *bytes.Buffer, spec *L3Spec) {
	idByName := map[string]string{
		spec.Focus.Name: strings.ToLower(spec.Focus.ID),
	}
	for _, element := range spec.Elements {
		idByName[element.Name] = strings.ToLower(element.ID)
	}
	relIDs := assignRelationshipIDs(spec.Relationships)
	emitMermaidEdges(buf, relIDs, idByName)
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

func l3AnchorID(id, name string) string {
	number := strings.TrimPrefix(id, "E")
	return fmt.Sprintf("e%s-%s", number, slug(name))
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
	names := map[string]bool{spec.Focus.Name: true}
	for _, element := range spec.Elements {
		names[element.Name] = true
	}
	for index, rel := range spec.Relationships {
		if !names[rel.From] {
			return fmt.Errorf("relationships[%d]: from %q not in elements or focus", index, rel.From)
		}
		if !names[rel.To] {
			return fmt.Errorf("relationships[%d]: to %q not in elements or focus", index, rel.To)
		}
	}
	return nil
}

func validateL3SingleElement(index int, element L3Element) error {
	if !mermaidIDPrefix.MatchString(element.ID) {
		return fmt.Errorf("elements[%d]: id %q must match E<n>", index, element.ID)
	}
	if strings.TrimSpace(element.Name) == "" {
		return fmt.Errorf("elements[%d]: name must be non-empty", index)
	}
	switch element.Kind {
	case "component":
		if element.FromParent {
			return fmt.Errorf("elements[%d]: kind=component cannot be from_parent", index)
		}
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
	if !mermaidIDPrefix.MatchString(spec.Focus.ID) {
		return fmt.Errorf("focus.id %q must match E<n>", spec.Focus.ID)
	}
	if strings.TrimSpace(spec.Focus.Name) == "" {
		return errors.New("focus.name: must be non-empty")
	}
	return validateL3Elements(spec)
}

// validateL3SpecAgainstRegistry derives the registry from peer specs in the
// same directory (excluding the input itself) and verifies that every
// from_parent element + the focus reference an existing registry entry by both
// ID and name. New (component) elements must not collide with registry IDs
// under a different name.
func validateL3SpecAgainstRegistry(ctx context.Context, inputPath string, spec *L3Spec) error {
	dir := filepath.Dir(inputPath)
	files, records, err := scanRegistryDir(ctx, dir)
	if err != nil {
		return fmt.Errorf("registry scan: %w", err)
	}
	inputBase := filepath.Base(inputPath)
	peerFiles := make([]string, 0, len(files))
	peerRecords := make([]registryRecord, 0, len(records))
	for _, file := range files {
		if file == inputBase {
			continue
		}
		peerFiles = append(peerFiles, file)
	}
	for _, rec := range records {
		if rec.File == inputBase {
			continue
		}
		peerRecords = append(peerRecords, rec)
	}
	view := deriveRegistry(dir, peerFiles, peerRecords)
	elementByID := map[string]RegistryElement{}
	for _, element := range view.Elements {
		elementByID[element.ID] = element
	}
	if err := checkFocusAgainstRegistry(spec.Focus, elementByID); err != nil {
		return err
	}
	for _, element := range spec.Elements {
		if err := checkL3ElementAgainstRegistry(element, elementByID); err != nil {
			return err
		}
	}
	return nil
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
