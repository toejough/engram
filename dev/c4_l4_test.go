//go:build targ

package dev

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitL4MermaidEdge_AppendsPropertySuffix(t *testing.T) {
	t.Parallel()
	edge := L4Edge{
		ID: "R8", From: "S2-N3-M3", To: "S2-N3-M4",
		Label: "strips transcript", Properties: []string{"P3", "P4", "P9"},
	}
	var buf bytes.Buffer
	emitL4MermaidEdge(&buf, edge, nil)
	out := buf.String()
	if !strings.Contains(out, "R8: strips transcript [P3, P4, P9]") {
		t.Fatalf("expected bracketed property suffix, got: %s", out)
	}
}

func TestEmitL4MermaidEdge_DottedWhenTargetIsDIWrapped(t *testing.T) {
	t.Parallel()
	edge := L4Edge{
		ID: "R10", From: "S2-N3-M3", To: "S2-N3-M7",
		Label: "ranks via SummarizerI",
	}
	diTargets := map[string]bool{"S2-N3-M7": true}
	var buf bytes.Buffer
	emitL4MermaidEdge(&buf, edge, diTargets)
	out := buf.String()
	if !strings.Contains(out, "-.->") {
		t.Fatalf("expected dotted arrow for DI-mediated R-edge, got: %s", out)
	}
}

func TestEmitL4MermaidEdge_OmitsSuffixWhenNoProperties(t *testing.T) {
	t.Parallel()
	edge := L4Edge{
		ID: "R3", From: "S2-N3-M2", To: "S2-N3-M3",
		Label: "constructs + invokes",
	}
	var buf bytes.Buffer
	emitL4MermaidEdge(&buf, edge, nil)
	out := buf.String()
	if strings.Contains(out, "[") {
		t.Fatalf("expected no brackets, got: %s", out)
	}
}

func TestEmitL4MermaidEdge_SolidWhenTargetIsDirectCall(t *testing.T) {
	t.Parallel()
	edge := L4Edge{
		ID: "R8", From: "S2-N3-M3", To: "S2-N3-M4",
		Label: "strips transcript",
	}
	diTargets := map[string]bool{"S2-N3-M7": true}
	var buf bytes.Buffer
	emitL4MermaidEdge(&buf, edge, diTargets)
	out := buf.String()
	if strings.Contains(out, "-.->") {
		t.Fatalf("expected solid arrow for non-DI R-edge, got: %s", out)
	}
	if !strings.Contains(out, "-->") {
		t.Fatalf("expected solid arrow, got: %s", out)
	}
}

func TestEmitL4WiringMermaid_DedupesByWrappedEntity(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	spec.DependencyManifest = []L4DepRow{
		{Field: "f1", Type: "T", WiredByID: "S2-N3-M2", WiredByName: "cli", WiredByL3: "x.md", WrappedEntityID: "S3"},
		{Field: "f2", Type: "T", WiredByID: "S2-N3-M2", WiredByName: "cli", WiredByL3: "x.md", WrappedEntityID: "S3"},
		{Field: "f3", Type: "T", WiredByID: "S2-N3-M2", WiredByName: "cli", WiredByL3: "x.md", WrappedEntityID: "S2-N3-M7"},
	}
	spec.Diagram.Nodes = append(spec.Diagram.Nodes,
		L4Node{ID: "S3", Name: "Claude Code", Kind: "external"},
		L4Node{ID: "S2-N3-M7", Name: "anthropic", Kind: "component"},
	)
	var buf bytes.Buffer
	emitL4WiringMermaid(&buf, spec)
	out := buf.String()
	// Expect exactly two cli→focus edges, labeled "S3" and "S2-N3-M7".
	s3Count := strings.Count(out, `|"S3"|`)
	antCount := strings.Count(out, `|"S2-N3-M7"|`)
	if s3Count != 1 || antCount != 1 {
		t.Fatalf("expected one S3 edge and one S2-N3-M7 edge, got s3=%d ant=%d in:\n%s", s3Count, antCount, out)
	}
	// Wrapped-entity nodes must NOT appear as standalone nodes — their SNM
	// IDs are conveyed by the edge labels alone.
	if strings.Contains(out, `s3[`) || strings.Contains(out, `s3(`) {
		t.Errorf("wrapped entity S3 unexpectedly rendered as a node:\n%s", out)
	}
	if strings.Contains(out, `s2-n3-m7[`) || strings.Contains(out, `s2-n3-m7(`) {
		t.Errorf("wrapped entity S2-N3-M7 unexpectedly rendered as a node:\n%s", out)
	}
}

func TestL4DepRow_HasSlimSchema(t *testing.T) {
	t.Parallel()
	row := L4DepRow{
		Field: "summarizer", Type: "SummarizerI",
		WiredByID: "S2-N3-M2", WiredByName: "cli", WiredByL3: "c3-engram-cli-binary.md",
		WrappedEntityID: "S2-N3-M7",
		Properties:      []string{"P5", "P6"},
	}
	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(raw)
	for _, want := range []string{"field", "wired_by_id", "wrapped_entity_id"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q: %s", want, s)
		}
	}
	for _, gone := range []string{"concrete_adapter", "wired_adapter", "concrete_value", "consumer_field"} {
		if strings.Contains(s, gone) {
			t.Errorf("legacy field %q still present: %s", gone, s)
		}
	}
}

func TestL4Spec_RejectsDEdges(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	spec.Diagram.Edges = append(spec.Diagram.Edges, L4Edge{
		ID: "D1", From: "S2-N3-M3", To: "S2-N3-M2", Label: "legacy",
	})
	err := validateL4Spec(spec, nil)
	if err == nil || !strings.Contains(err.Error(), "R<n>") {
		t.Fatalf("expected D-edge rejection mentioning R<n>, got: %v", err)
	}
}

func TestL4Spec_RejectsManifestWrappedEntityNotInDiagram(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	spec.DependencyManifest = []L4DepRow{
		{
			Field: "ghost", Type: "Ghost",
			WiredByID: "S2-N3-M2", WiredByName: "cli", WiredByL3: "c3-engram-cli-binary.md",
			WrappedEntityID: "S99-NOT-IN-DIAGRAM",
			Properties:      nil,
		},
	}
	err := validateL4Spec(spec, nil)
	if err == nil || !strings.Contains(err.Error(), "S99-NOT-IN-DIAGRAM") {
		t.Fatalf("expected wrapped-entity validation failure, got: %v", err)
	}
}

func TestLoadL3Parent_MissingFileWrapsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l4 := &L4Spec{Parent: "c3-nope.md"}
	_, err := loadL3Parent(l4, dir)
	if err == nil || !strings.Contains(err.Error(), "c3-nope.json") {
		t.Fatalf("expected wrapped error mentioning c3-nope.json, got: %v", err)
	}
}

func TestLoadL3Parent_ReadsSiblingJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l3JSON := []byte(`{"schema_version":"1","level":3,"name":"engram-cli-binary","parent":"c2-engram-plugin.md","focus":{"id":"S2-N3","name":"engram CLI binary","responsibility":"x"},"elements":[],"relationships":[]}`)
	if err := os.WriteFile(filepath.Join(dir, "c3-engram-cli-binary.json"), l3JSON, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	l4 := &L4Spec{Parent: "c3-engram-cli-binary.md"}
	l3, err := loadL3Parent(l4, dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if l3 == nil || l3.Name != "engram-cli-binary" {
		t.Fatalf("bad load: %+v", l3)
	}
}

func TestT52_ValidateL4Spec_AcceptsValidSpec(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	if err := validateL4Spec(spec, nil); err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}
}

func TestT53_ValidateL4Spec_RequiresFocusLevel3(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	spec.Focus.ID = "S2-N3" // level 2, not level 3
	err := validateL4Spec(spec, nil)
	if err == nil {
		t.Fatal("expected error for level-2 focus id, got nil")
	}
	if !strings.Contains(err.Error(), "level 3") {
		t.Errorf("want error mentioning 'level 3', got %q", err.Error())
	}
}

func TestT54_ValidateL4NodeIDs_AcceptsSiblings(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	// S2-N3-M5 is a sibling of focus S2-N3-M3 (same depth, same parent S2-N3)
	spec.Diagram.Nodes = append(spec.Diagram.Nodes,
		L4Node{ID: "S2-N3-M5", Name: "memory", Kind: "component"})
	if err := validateL4NodeIDs(spec); err != nil {
		t.Fatalf("sibling node rejected: %v", err)
	}
}

func TestT55_ValidateL4NodeIDs_AcceptsAncestors(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	// S2 is a level-1 ancestor of focus S2-N3-M3
	spec.Diagram.Nodes = append(spec.Diagram.Nodes,
		L4Node{ID: "S2", Name: "engram", Kind: "container"})
	if err := validateL4NodeIDs(spec); err != nil {
		t.Fatalf("ancestor node rejected: %v", err)
	}
}

func TestT56_ValidateL4NodeIDs_RejectsDescendant(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	// S2-N3-M3-P1 is a descendant of focus S2-N3-M3 — not allowed at diagram level
	spec.Diagram.Nodes = append(spec.Diagram.Nodes,
		L4Node{ID: "S2-N3-M3-P1", Name: "property", Kind: "component"})
	err := validateL4NodeIDs(spec)
	if err == nil {
		t.Fatal("expected error for descendant node, got nil")
	}
	if !strings.Contains(err.Error(), "S2-N3-M3-P1") {
		t.Errorf("want error mentioning descendant id, got %q", err.Error())
	}
}

func TestT57_ValidateL4NodeIDs_RejectsUnrelatedM(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	// S3-N1-M1 is level-3 but shares no parent with focus S2-N3-M3
	spec.Diagram.Nodes = append(spec.Diagram.Nodes,
		L4Node{ID: "S3-N1-M1", Name: "unrelated", Kind: "component"})
	err := validateL4NodeIDs(spec)
	if err == nil {
		t.Fatal("expected error for unrelated M-node, got nil")
	}
	if !strings.Contains(err.Error(), "S3-N1-M1") {
		t.Errorf("want error mentioning node id, got %q", err.Error())
	}
}

func TestT57b_ValidateL4NodeIDs_AcceptsAnyL1L2CarryOver(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	// L1 peer (different system) and L2 aunt/uncle container — both accepted as carry-over.
	spec.Diagram.Nodes = append(spec.Diagram.Nodes,
		L4Node{ID: "S5", Name: "Anthropic", Kind: "external"},
		L4Node{ID: "S2-N1", Name: "skills", Kind: "container"})
	if err := validateL4NodeIDs(spec); err != nil {
		t.Fatalf("L1/L2 carry-over rejected: %v", err)
	}
}

func TestT58_ValidateL4NodeIDs_RejectsLetterSuffixedEdge(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	spec.Diagram.Edges[0].ID = "R2a"
	err := validateL4NodeIDs(spec)
	if err == nil || !strings.Contains(err.Error(), "R2a") {
		t.Fatalf("want error naming R2a; got %v", err)
	}
}

func TestT59_ValidateL4NodeIDs_AggregatesAllViolations(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	spec.Diagram.Nodes = append(spec.Diagram.Nodes,
		L4Node{ID: "S3-N1-M1", Name: "unrelated", Kind: "component"})
	spec.Diagram.Edges[0].ID = "R2a"
	err := validateL4NodeIDs(spec)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "S3-N1-M1") || !strings.Contains(err.Error(), "R2a") {
		t.Errorf("want aggregated error naming both S3-N1-M1 and R2a; got %v", err)
	}
}

func TestT60_ValidateL4Properties_RequiresLevel4ID(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	spec.Properties[0].ID = "S2-N3-M3" // level 3, not level 4
	err := validateL4Spec(spec, nil)
	if err == nil {
		t.Fatal("expected error for level-3 property id, got nil")
	}
	if !strings.Contains(err.Error(), "level 4") {
		t.Errorf("want error mentioning 'level 4', got %q", err.Error())
	}
}

func TestT61_ValidateL4Properties_RequiresFocusAncestry(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	spec.Properties[0].ID = "S2-N3-M5-P1" // under sibling M5, not focus M3
	err := validateL4Spec(spec, nil)
	if err == nil {
		t.Fatal("expected error for property not under focus, got nil")
	}
	if !strings.Contains(err.Error(), "is not valid at level 4") {
		t.Errorf("want error mentioning 'is not valid at level 4', got %q", err.Error())
	}
}

func TestT62_ValidateL4Properties_RequiresSequentialIndex(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	spec.Properties[0].ID = "S2-N3-M3-P2" // should be P1 at index 0
	err := validateL4Spec(spec, nil)
	if err == nil {
		t.Fatal("expected error for wrong P index, got nil")
	}
	if !strings.Contains(err.Error(), "P1") {
		t.Errorf("want error mentioning expected P1, got %q", err.Error())
	}
}

func TestT63_Anchor_UsesHierarchicalID(t *testing.T) {
	t.Parallel()
	anchor := Anchor("S2-N3-M3-P1", "Sessions sorted newest-first")
	want := "s2-n3-m3-p1-sessions-sorted-newest-first"
	if anchor != want {
		t.Errorf("want %q, got %q", want, anchor)
	}
}

func TestT64_FormatPropertyList_CollapsesSamePrefix(t *testing.T) {
	t.Parallel()
	ids := []string{"S2-N3-M3-P2", "S2-N3-M3-P3", "S2-N3-M3-P4", "S2-N3-M3-P5", "S2-N3-M3-P6"}
	got := formatPropertyList(ids)
	want := "S2-N3-M3-P2–P6"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestT65_FormatPropertyList_JoinsDifferentPrefixes(t *testing.T) {
	t.Parallel()
	ids := []string{"S2-N3-M3-P1", "S2-N3-M4-P1"}
	got := formatPropertyList(ids)
	if !strings.Contains(got, "S2-N3-M3-P1") || !strings.Contains(got, "S2-N3-M4-P1") {
		t.Errorf("want both IDs in output, got %q", got)
	}
}

func TestValidateL4Carryover_FocusKindRelaxed(t *testing.T) {
	t.Parallel()
	l4 := &L4Spec{
		Focus:   L4Focus{ID: "F", Name: "focus"},
		Parent:  "c3-x.md",
		Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}}},
	}
	l3 := &L3Spec{
		Focus:    L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
		Elements: []L3Element{{ID: "F", Name: "focus", Kind: "component"}},
	}
	if err := validateL4Carryover(l4, l3); err != nil {
		t.Fatalf("focus kind relaxation should pass, got: %v", err)
	}
}

func TestValidateL4Carryover_FocusMissingFromL3(t *testing.T) {
	t.Parallel()
	l4 := &L4Spec{Focus: L4Focus{ID: "S2-N3-M3", Name: "recall"}, Parent: "c3-x.md"}
	l3 := &L3Spec{Elements: []L3Element{}}
	err := validateL4Carryover(l4, l3)
	if err == nil || !strings.Contains(err.Error(), "S2-N3-M3") {
		t.Fatalf("expected focus-id error, got: %v", err)
	}
}

func TestValidateL4Carryover_HappyPath(t *testing.T) {
	t.Parallel()
	l4 := &L4Spec{
		Focus:  L4Focus{ID: "F", Name: "focus"},
		Parent: "c3-x.md",
		Diagram: L4Diagram{Nodes: []L4Node{
			{ID: "F", Name: "focus", Kind: "focus"},
			{ID: "M", Name: "memory", Kind: "component"},
			{ID: "C", Name: "cli", Kind: "component"},
		}},
	}
	l3 := &L3Spec{
		Focus: L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
		Elements: []L3Element{
			{ID: "F", Name: "focus", Kind: "component"},
			{ID: "M", Name: "memory", Kind: "component"},
			{ID: "C", Name: "cli", Kind: "component"},
		},
		Relationships: []L1Relationship{
			{From: "F", To: "M", Description: "writes"},
			{From: "C", To: "F", Description: "calls"},
		},
	}
	if err := validateL4Carryover(l4, l3); err != nil {
		t.Fatalf("happy path should pass: %v", err)
	}
}

func TestValidateL4Carryover_KindMismatch(t *testing.T) {
	t.Parallel()
	l4 := &L4Spec{
		Focus:   L4Focus{ID: "F", Name: "focus"},
		Parent:  "c3-x.md",
		Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}, {ID: "S5", Name: "anthropic", Kind: "component"}}},
	}
	l3 := &L3Spec{
		Focus:    L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
		Elements: []L3Element{{ID: "F", Name: "focus", Kind: "component"}, {ID: "S5", Name: "anthropic", Kind: "external"}},
	}
	err := validateL4Carryover(l4, l3)
	if err == nil || !strings.Contains(err.Error(), "kind") || !strings.Contains(err.Error(), "S5") {
		t.Fatalf("expected kind mismatch citing S5, got: %v", err)
	}
}

func TestValidateL4Carryover_L4ExtraNode(t *testing.T) {
	t.Parallel()
	l4 := &L4Spec{
		Focus:   L4Focus{ID: "F", Name: "focus"},
		Parent:  "c3-x.md",
		Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}, {ID: "X", Name: "ghost", Kind: "component"}}},
	}
	l3 := &L3Spec{
		Focus:    L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
		Elements: []L3Element{{ID: "F", Name: "focus", Kind: "component"}},
	}
	err := validateL4Carryover(l4, l3)
	if err == nil || !strings.Contains(err.Error(), `"X"`) {
		t.Fatalf("expected extra-node error citing X, got: %v", err)
	}
}

func TestValidateL4Carryover_MissingNeighborInbound(t *testing.T) {
	t.Parallel()
	l4 := &L4Spec{
		Focus:   L4Focus{ID: "F", Name: "focus"},
		Parent:  "c3-x.md",
		Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}}},
	}
	l3 := &L3Spec{
		Focus:         L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
		Elements:      []L3Element{{ID: "F", Name: "focus", Kind: "component"}, {ID: "C", Name: "cli", Kind: "component"}},
		Relationships: []L1Relationship{{From: "C", To: "F", Description: "calls"}},
	}
	err := validateL4Carryover(l4, l3)
	if err == nil || !strings.Contains(err.Error(), `"C"`) {
		t.Fatalf("expected missing-neighbor C, got: %v", err)
	}
}

func TestValidateL4Carryover_MissingNeighborOutbound(t *testing.T) {
	t.Parallel()
	l4 := &L4Spec{
		Focus:   L4Focus{ID: "F", Name: "focus"},
		Parent:  "c3-x.md",
		Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}}},
	}
	l3 := &L3Spec{
		Focus:         L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
		Elements:      []L3Element{{ID: "F", Name: "focus", Kind: "component"}, {ID: "M", Name: "memory", Kind: "component"}},
		Relationships: []L1Relationship{{From: "F", To: "M", Description: "writes"}},
	}
	err := validateL4Carryover(l4, l3)
	if err == nil || !strings.Contains(err.Error(), `"M"`) {
		t.Fatalf("expected missing-neighbor M, got: %v", err)
	}
}

func TestValidateL4Carryover_RelationshipNeighborNotInElements(t *testing.T) {
	t.Parallel()
	l4 := &L4Spec{
		Focus:   L4Focus{ID: "F", Name: "focus"},
		Parent:  "c3-x.md",
		Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}}},
	}
	l3 := &L3Spec{
		Focus:         L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
		Elements:      []L3Element{{ID: "F", Name: "focus", Kind: "component"}}, // no "PHANTOM"
		Relationships: []L1Relationship{{From: "F", To: "PHANTOM", Description: "x"}},
	}
	err := validateL4Carryover(l4, l3)
	if err == nil || !strings.Contains(err.Error(), "PHANTOM") {
		t.Fatalf("expected missing-neighbor PHANTOM citing the L3-relationship-only node, got: %v", err)
	}
}

func TestValidateL4Carryover_SelfLoopIgnored(t *testing.T) {
	t.Parallel()
	l4 := &L4Spec{
		Focus:   L4Focus{ID: "F", Name: "focus"},
		Parent:  "c3-x.md",
		Diagram: L4Diagram{Nodes: []L4Node{{ID: "F", Name: "focus", Kind: "focus"}}},
	}
	l3 := &L3Spec{
		Focus:         L3Focus{ID: "S2-N3", Name: "n", Responsibility: "r"},
		Elements:      []L3Element{{ID: "F", Name: "focus", Kind: "component"}},
		Relationships: []L1Relationship{{From: "F", To: "F", Description: "self"}},
	}
	if err := validateL4Carryover(l4, l3); err != nil {
		t.Fatalf("self-loop should not produce a neighbor: %v", err)
	}
}

func TestValidateL4Spec_RunsCarryover(t *testing.T) {
	t.Parallel()
	spec := validL4Spec()
	spec.Diagram.Nodes = append(spec.Diagram.Nodes, L4Node{ID: "BOGUS", Name: "ghost", Kind: "component"})
	l3 := minimalL3ParentFor(spec)
	// Remove BOGUS from l3 so the carryover check fires.
	filtered := make([]L3Element, 0, len(l3.Elements))
	for _, elem := range l3.Elements {
		if elem.ID != "BOGUS" {
			filtered = append(filtered, elem)
		}
	}
	l3.Elements = filtered
	err := validateL4Spec(spec, l3)
	if err == nil || !strings.Contains(err.Error(), "BOGUS") {
		t.Fatalf("expected carryover error citing BOGUS, got: %v", err)
	}
}

// minimalL3ParentFor returns an L3Spec that exactly satisfies the carryover
// check for the given L4 spec — every L4 node mirrored as an L3 element,
// every L3 neighbor of focus drawn from L4's nodes.
func minimalL3ParentFor(l4 *L4Spec) *L3Spec {
	l3 := &L3Spec{Focus: L3Focus{ID: "S2-N3", Name: "stub", Responsibility: "stub"}}
	for _, node := range l4.Diagram.Nodes {
		kind := node.Kind
		if node.ID == l4.Focus.ID && kind == "focus" {
			kind = "component"
		}
		l3.Elements = append(l3.Elements, L3Element{ID: node.ID, Name: node.Name, Kind: kind})
		if node.ID != l4.Focus.ID {
			l3.Relationships = append(l3.Relationships, L1Relationship{
				From: l4.Focus.ID, To: node.ID, Description: "stub",
			})
		}
	}
	return l3
}

// validL4Spec returns a minimal spec that passes all validateL4Spec checks.
// Focus is S2-N3-M3, siblings are S2-N3-M2 and S2-N3-M4, ancestor is S2-N3.
func validL4Spec() *L4Spec {
	return &L4Spec{
		SchemaVersion: "1",
		Level:         4,
		Name:          "thing",
		Parent:        "c3-engram-cli-binary.md",
		Focus:         L4Focus{ID: "S2-N3-M3", Name: "recall"},
		ContextProse:  "fixture prose",
		Diagram: L4Diagram{
			Nodes: []L4Node{
				{ID: "S2-N3-M3", Name: "recall", Kind: "focus"},
				{ID: "S2-N3-M2", Name: "cli", Kind: "component"},
				{ID: "S2-N3-M4", Name: "context", Kind: "component"},
				{ID: "S2-N3", Name: "engram-cli-binary", Kind: "container"},
			},
			Edges: []L4Edge{
				{ID: "R1", From: "S2-N3-M2", To: "S2-N3-M3", Label: "calls"},
			},
		},
		Properties: []L4Property{
			{
				ID:        "S2-N3-M3-P1",
				Name:      "Sessions sorted newest-first",
				Statement: "For all sessions, Find returns sorted results.",
				EnforcedAt: []L4CodeLink{
					{Path: "internal/recall/recall.go", Line: 44},
				},
				TestedAt: []L4CodeLink{
					{Path: "internal/recall/recall_test.go", Line: 50},
				},
			},
		},
	}
}
