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
	"strconv"
	"strings"

	"github.com/toejough/targ"
)

func init() {
	targ.Register(targ.Targ(c4L2Build).Name("c4-l2-build").
		Description("Build canonical C4 L2 markdown from a JSON spec next to the input file."))
}

// C4L2BuildArgs configures the c4-l2-build target.
type C4L2BuildArgs struct {
	Input     string `targ:"flag,name=input,desc=JSON spec path (required)"`
	Check     bool   `targ:"flag,name=check,desc=Verify existing .md matches generated; non-zero on diff"`
	NoConfirm bool   `targ:"flag,name=noconfirm,desc=Overwrite existing .md without prompting"`
}

// L2Element is one element on an L2 diagram. Each element has an explicit
// hierarchical ID: carried-over L1 elements use the parent's S<n> ID; new
// containers under the focus use <focusPath>-N<m>. Exactly one element must
// set InScope=true; that element is rendered as a mermaid subgraph wrapping
// all internal containers and is the L1 element being refined. Its ID must
// be a level-1 (S<n>) path.
type L2Element struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	Subtitle       *string `json:"subtitle,omitempty"`
	Responsibility string  `json:"responsibility"`
	SystemOfRecord string  `json:"system_of_record"`
	InScope        bool    `json:"in_scope,omitempty"`
}

// L2Spec is the JSON-source-of-truth representation of a C4 L2 diagram.
type L2Spec struct {
	SchemaVersion string           `json:"schema_version"`
	Level         int              `json:"level"`
	Name          string           `json:"name"`
	Parent        string           `json:"parent"`
	Preamble      string           `json:"preamble"`
	Elements      []L2Element      `json:"elements"`
	Relationships []L1Relationship `json:"relationships"`
	DriftNotes    []L1DriftNote    `json:"drift_notes"`
	CrossLinks    L1CrossLinks     `json:"cross_links"`
}

func c4L2Build(ctx context.Context, args C4L2BuildArgs) error {
	if args.Input == "" {
		return errors.New("--input is required")
	}
	spec, err := loadAndValidateL2Spec(args.Input)
	if err != nil {
		return err
	}
	sha, err := currentGitShortSHA(ctx)
	if err != nil {
		return fmt.Errorf("git rev-parse: %w", err)
	}
	outPath := strings.TrimSuffix(args.Input, ".json") + ".md"
	var buf bytes.Buffer
	if err := emitL2Markdown(&buf, spec, sha); err != nil {
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

func emitL2CrossLinks(buf *bytes.Buffer, spec *L2Spec, inScope elementID) {
	buf.WriteString("## Cross-links\n\n")
	fmt.Fprintf(buf, "- Parent: [%s](%s) (refines **%s · %s**)\n",
		spec.Parent, spec.Parent, inScope.ID, inScope.Element.Name)
	if len(spec.CrossLinks.RefinedBy) == 0 {
		buf.WriteString("- Refined by: *(none yet)*\n")
		return
	}
	buf.WriteString("- Refined by:\n")
	for _, link := range spec.CrossLinks.RefinedBy {
		fmt.Fprintf(buf, "  - [`%s`](./%s) — %s\n", link.File, link.File, link.Note)
	}
}

func emitL2FrontMatter(buf *bytes.Buffer, spec *L2Spec, lastReviewedCommit string) {
	fmt.Fprintf(buf, "---\nlevel: %d\nname: %s\nparent: %s\nchildren: []\nlast_reviewed_commit: %s\n---\n",
		spec.Level, spec.Name, strconv.Quote(spec.Parent), lastReviewedCommit)
}

// emitL2Markdown renders spec to canonical L2 markdown.
func emitL2Markdown(w io.Writer, spec *L2Spec, lastReviewedCommit string) error {
	elementIDs, err := validateL2ElementIDs(spec.Elements)
	if err != nil {
		return fmt.Errorf("validate l2 element ids: %w", err)
	}
	relIDs := assignRelationshipIDs(spec.Relationships)
	inScope, ok := findInScopeElement(elementIDs, spec.Elements)
	if !ok {
		return errors.New("no in_scope element found")
	}
	var buf bytes.Buffer
	emitL2FrontMatter(&buf, spec, lastReviewedCommit)
	fmt.Fprintf(&buf, "\n# C2 — %s (Container)\n\n%s\n\n", inScope.Element.Name, strings.TrimRight(spec.Preamble, "\n"))
	emitL2Mermaid(&buf, elementIDs, relIDs, inScope)
	emitCatalog(&buf, elementIDs)
	emitRelationships(&buf, elementIDs, relIDs)
	emitL2CrossLinks(&buf, spec, inScope)
	emitDriftNotes(&buf, spec.DriftNotes)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write markdown: %w", err)
	}
	return nil
}

func emitL2Mermaid(buf *bytes.Buffer, elementIDs []elementID, relIDs []relationshipID, inScope elementID) {
	buf.WriteString("```mermaid\nflowchart LR\n")
	buf.WriteString("    classDef person      fill:#08427b,stroke:#052e56,color:#fff\n")
	buf.WriteString("    classDef external    fill:#999,   stroke:#666,   color:#fff\n")
	buf.WriteString("    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff\n\n")
	emitL2NonScopeNodes(buf, elementIDs, inScope)
	buf.WriteString("\n")
	emitL2Subgraph(buf, elementIDs, inScope)
	buf.WriteString("\n")
	emitMermaidEdges(buf, relIDs, mermaidIDByElementID(elementIDs))
	buf.WriteString("\n")
	emitL2MermaidClasses(buf, elementIDs, inScope)
	buf.WriteString("\n")
	emitMermaidClicks(buf, elementIDs)
	buf.WriteString("```\n\n")
}

func emitL2MermaidClasses(buf *bytes.Buffer, elementIDs []elementID, inScope elementID) {
	groups := map[string][]string{}
	classOrder := []string{"person", "external", "container"}
	for _, item := range elementIDs {
		if item.ID == inScope.ID {
			continue
		}
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
	fmt.Fprintf(buf, "    class %s container\n", strings.ToLower(inScope.ID))
}

func emitL2NonScopeNodes(buf *bytes.Buffer, elementIDs []elementID, inScope elementID) {
	for _, item := range elementIDs {
		if item.ID == inScope.ID {
			continue
		}
		if item.Element.Kind == "container" {
			continue
		}
		shape := mermaidShapeFor(item.Element)
		label := mermaidLabelFor(item)
		mermaidID := strings.ToLower(item.ID)
		fmt.Fprintf(buf, "    %s%s%s%s\n", mermaidID, shape[0], label, shape[1])
	}
}

func emitL2Subgraph(buf *bytes.Buffer, elementIDs []elementID, inScope elementID) {
	scopeMermaidID := strings.ToLower(inScope.ID)
	fmt.Fprintf(buf, "    subgraph %s [%s · %s]\n", scopeMermaidID, inScope.ID, inScope.Element.Name)
	for _, item := range elementIDs {
		if item.ID == inScope.ID {
			continue
		}
		if item.Element.Kind != "container" {
			continue
		}
		shape := mermaidShapeFor(item.Element)
		label := mermaidLabelFor(item)
		mermaidID := strings.ToLower(item.ID)
		fmt.Fprintf(buf, "        %s%s%s%s\n", mermaidID, shape[0], label, shape[1])
	}
	buf.WriteString("    end\n")
}

func findInScopeElement(ids []elementID, src []L2Element) (elementID, bool) {
	for index, item := range ids {
		if src[index].InScope {
			return item, true
		}
	}
	return elementID{}, false
}

// loadAndValidateL2Spec reads a JSON L2 spec from path, decodes it strictly,
// and validates the L2 schema rules.
func loadAndValidateL2Spec(path string) (*L2Spec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var spec L2Spec
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&spec); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := validateL2Spec(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

// mermaidIDByElementID maps each element's hierarchical ID to its lowercase
// mermaid node ID. Used at L2+ where relationships are keyed directly by
// hierarchical ID (no name substitution needed).
func mermaidIDByElementID(elementIDs []elementID) map[string]string {
	out := make(map[string]string, len(elementIDs))
	for _, item := range elementIDs {
		out[item.ID] = strings.ToLower(item.ID)
	}
	return out
}

// validateL2ElementIDs validates that each element has an explicit
// hierarchical path ID. The InScope element must have an L1 (S<n>) ID; that
// path becomes the focus. Other elements may be either:
//   - L1 paths (S<n>): carried-over peers from the parent diagram, or
//   - L2 paths (S<n>-N<m>): new containers under the focus.
//
// On success it returns elementIDs with anchors of the form
// "<lower-id>-<slug(name)>" (e.g. "s1-developer", "s2-n1-skills").
func validateL2ElementIDs(elements []L2Element) ([]elementID, error) {
	var focusPath IDPath
	for _, element := range elements {
		path, err := ParseIDPath(element.ID)
		if err != nil {
			return nil, fmt.Errorf("element %q: %w", element.Name, err)
		}
		if element.InScope {
			if path.Level != 1 {
				return nil, fmt.Errorf("in_scope element %q must have an L1 (S<n>) id, got %s",
					element.Name, element.ID)
			}
			focusPath = path
		}
	}
	if focusPath.Level == 0 {
		return nil, errors.New("L2 spec has no in_scope element")
	}
	result := make([]elementID, 0, len(elements))
	used := map[string]int{}
	for _, element := range elements {
		if !element.InScope {
			path, _ := ParseIDPath(element.ID)
			switch path.Level {
			case 1:
				// carried-over peer from L1 (e.g. external/person)
			case 2:
				if !focusPath.IsAncestorOf(path) {
					return nil, fmt.Errorf("element %q id %s is not under focus %s",
						element.Name, element.ID, focusPath.String())
				}
			default:
				return nil, fmt.Errorf("element %q has unsupported L2 id depth %d (%s)",
					element.Name, path.Level, element.ID)
			}
		}
		base := strings.ToLower(element.ID) + "-" + slug(element.Name)
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
				SystemOfRecord: element.SystemOfRecord,
			},
		})
	}
	return result, nil
}

func validateL2Elements(elements []L2Element) error {
	inScopeCount := 0
	seenName := map[string]bool{}
	seenID := map[string]bool{}
	for index, element := range elements {
		if !validKinds[element.Kind] {
			return fmt.Errorf("elements[%d]: kind %q not in {person, external, container}", index, element.Kind)
		}
		if seenName[element.Name] {
			return fmt.Errorf("elements: duplicate name %q", element.Name)
		}
		seenName[element.Name] = true
		if element.ID == "" {
			return fmt.Errorf("elements[%d]: id is required", index)
		}
		if seenID[element.ID] {
			return fmt.Errorf("elements: duplicate id %q", element.ID)
		}
		seenID[element.ID] = true
		if element.InScope {
			inScopeCount++
			if element.Kind != "container" {
				return fmt.Errorf("elements[%d]: in_scope=true requires kind=container", index)
			}
		}
	}
	if inScopeCount != 1 {
		return fmt.Errorf("expected exactly one in_scope: true, got %d", inScopeCount)
	}
	return nil
}

func validateL2Relationships(elements []L2Element, rels []L1Relationship, links L1CrossLinks) error {
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
		// L2 children are L3 files, but allow either pattern; the schema doesn't
		// enforce here since c2 may have c3 children.
		_ = index
		_ = link
	}
	return nil
}

func validateL2Spec(spec *L2Spec) error {
	if spec.SchemaVersion != "1" {
		return fmt.Errorf("unknown schema_version %q (want \"1\")", spec.SchemaVersion)
	}
	if spec.Level != 2 {
		return fmt.Errorf("level: want 2, got %d", spec.Level)
	}
	if !validNameRe.MatchString(spec.Name) {
		return fmt.Errorf("name %q must match %s", spec.Name, validNameRe)
	}
	if strings.TrimSpace(spec.Parent) == "" {
		return errors.New("parent: must be non-empty at L2")
	}
	if strings.TrimSpace(spec.Preamble) == "" {
		return errors.New("preamble: must be non-empty")
	}
	if err := validateL2Elements(spec.Elements); err != nil {
		return err
	}
	return validateL2Relationships(spec.Elements, spec.Relationships, spec.CrossLinks)
}
