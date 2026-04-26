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

// L2Element is one element on an L2 diagram. Carry-over elements from the
// parent L1 must set FromParent=true and ID to the parent's E-ID. Exactly one
// element must set InScope=true; that element is rendered as a mermaid
// subgraph wrapping all internal containers and is the L1 element being
// refined.
type L2Element struct {
	ID             string  `json:"id,omitempty"`
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	Subtitle       *string `json:"subtitle,omitempty"`
	Responsibility string  `json:"responsibility"`
	SystemOfRecord string  `json:"system_of_record"`
	FromParent     bool    `json:"from_parent,omitempty"`
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

// assignL2ElementIDs returns elements paired with E-IDs. Elements with an
// explicit ID keep it; the rest are assigned the smallest unused E-id (in
// source order), filling gaps left by carry-over IDs from the parent before
// extending past the maximum. AnchorIDs use the same e<n>-<slug> shape as L1.
func assignL2ElementIDs(elements []L2Element) []elementID {
	explicit := map[int]bool{}
	for _, element := range elements {
		if element.ID == "" {
			continue
		}
		if number, err := strconv.Atoi(strings.TrimPrefix(element.ID, "E")); err == nil {
			explicit[number] = true
		}
	}
	result := make([]elementID, 0, len(elements))
	used := map[string]int{}
	next := 1
	for _, element := range elements {
		var id string
		if element.ID != "" {
			id = element.ID
		} else {
			for explicit[next] {
				next++
			}
			id = fmt.Sprintf("E%d", next)
			explicit[next] = true
			next++
		}
		number, _ := strconv.Atoi(strings.TrimPrefix(id, "E"))
		base := fmt.Sprintf("e%d-%s", number, slug(element.Name))
		anchor := base
		if used[base] > 0 {
			used[base]++
			anchor = fmt.Sprintf("%s-%d", base, used[base])
		} else {
			used[base] = 1
		}
		result = append(result, elementID{
			ID:       id,
			AnchorID: anchor,
			Element: L1Element{
				Name:           element.Name,
				Kind:           element.Kind,
				IsSystem:       element.InScope,
				Subtitle:       element.Subtitle,
				Responsibility: element.Responsibility,
				SystemOfRecord: element.SystemOfRecord,
			},
		})
	}
	return result
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
	elementIDs := assignL2ElementIDs(spec.Elements)
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
	emitMermaidEdges(buf, relIDs, mermaidIDByName(elementIDs))
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
		if element.FromParent && element.ID == "" {
			return fmt.Errorf("elements[%d]: from_parent=true requires explicit id", index)
		}
		if element.ID != "" {
			if !mermaidIDPrefix.MatchString(element.ID) {
				return fmt.Errorf("elements[%d]: id %q must match E<n>", index, element.ID)
			}
			if seenID[element.ID] {
				return fmt.Errorf("elements: duplicate id %q", element.ID)
			}
			seenID[element.ID] = true
		}
		if element.InScope {
			inScopeCount++
			if !element.FromParent {
				return fmt.Errorf("elements[%d]: in_scope=true requires from_parent=true", index)
			}
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
