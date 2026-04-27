//go:build targ

package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestT42_L3BuildRegistered(t *testing.T) {
	t.Parallel()

	out, err := exec.CommandContext(context.Background(), "targ").CombinedOutput()
	if err != nil {
		t.Fatalf("targ: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "c4-l3-build") {
		t.Errorf("targ list missing c4-l3-build\noutput: %s", out)
	}
}

func TestT43_L3Validates_AcceptsValid(t *testing.T) {
	t.Parallel()

	_, err := loadAndValidateL3Spec("testdata/c4/valid_l3/c3-foo-internal.json")
	if err != nil {
		t.Fatalf("loadAndValidateL3Spec: %v", err)
	}
}

func TestT44_L3Validates_RejectsBadSchemas(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		mutate    func(*L3Spec)
		wantError string
	}{
		{
			name:      "wrong level",
			mutate:    func(s *L3Spec) { s.Level = 2 },
			wantError: "level: want 3",
		},
		{
			name:      "empty parent",
			mutate:    func(s *L3Spec) { s.Parent = "" },
			wantError: "parent",
		},
		{
			name:      "empty preamble",
			mutate:    func(s *L3Spec) { s.Preamble = "" },
			wantError: "preamble",
		},
		{
			name:      "bad focus id",
			mutate:    func(s *L3Spec) { s.Focus.ID = "X1" },
			wantError: "focus.id",
		},
		{
			name:      "empty focus name",
			mutate:    func(s *L3Spec) { s.Focus.Name = "" },
			wantError: "focus.name",
		},
		{
			name: "component without code_pointer",
			mutate: func(s *L3Spec) {
				for index := range s.Elements {
					if s.Elements[index].Kind == "component" {
						s.Elements[index].CodePointer = ""
						return
					}
				}
			},
			wantError: "code_pointer",
		},
		{
			name: "non-component with code_pointer",
			mutate: func(s *L3Spec) {
				for index := range s.Elements {
					if s.Elements[index].Kind != "component" {
						s.Elements[index].CodePointer = "../../README.md"
						return
					}
				}
			},
			wantError: "code_pointer is only valid",
		},
		{
			name: "duplicate id with focus",
			mutate: func(s *L3Spec) {
				s.Elements[0].ID = s.Focus.ID
			},
			wantError: "duplicate id",
		},
		{
			name: "relationship to unknown element",
			mutate: func(s *L3Spec) {
				s.Relationships[0].To = "S9-N9-M9"
			},
			wantError: "not in elements",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			spec := loadValidL3Spec(t)
			c.mutate(spec)
			err := validateL3Spec(spec)
			if err == nil {
				t.Fatalf("want error containing %q, got nil", c.wantError)
			}
			if !strings.Contains(err.Error(), c.wantError) {
				t.Errorf("want error containing %q, got %q", c.wantError, err.Error())
			}
		})
	}
}

func TestT45_L3EmitContainsExpectedStructure(t *testing.T) {
	t.Parallel()

	spec := loadValidL3Spec(t)
	var buf bytes.Buffer
	if err := emitL3Markdown(&buf, spec, "abc1234", nil); err != nil {
		t.Fatalf("emitL3Markdown: %v", err)
	}
	got := buf.String()
	wantSubstrings := []string{
		"level: 3",
		"name: foo-internal",
		"parent: \"c2-foo-system.md\"",
		"# C3 — Foo (Component)",
		"![C3 foo-internal component diagram](svg/c3-foo-internal.svg)",
		"> Diagram source: [svg/c3-foo-internal.mmd](svg/c3-foo-internal.mmd)",
		"## Element Catalog",
		"| Code Pointer |",
		"<a id=\"s1-n2-foo\"></a>S1-N2 | Foo | Container in focus",
		"<a id=\"s1-n2-m1-worker\"></a>S1-N2-M1 | Worker | Component",
		"[./worker.go](./worker.go)",
		"## Relationships",
		"R1 | S2 | S1-N2-M1",
		"## Cross-links",
		"Parent: [c2-foo-system.md]",
		"Refined by: *(none yet)*",
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(got, want) {
			t.Errorf("output missing substring %q\nfull output:\n%s", want, got)
		}
	}
}

func TestT45b_L3MermaidContainsExpectedStructure(t *testing.T) {
	t.Parallel()

	spec := loadValidL3Spec(t)
	var buf bytes.Buffer
	emitL3Mermaid(&buf, spec)
	got := buf.String()
	wantSubstrings := []string{
		"%%{init: {'flowchart': {'defaultRenderer': 'elk'}}}%%",
		"flowchart LR",
		"classDef component",
		"subgraph s1-n2 [S1-N2 · Foo]",
		"s1-n2-m1[S1-N2-M1 · Worker",
		"s1-n2-m2[S1-N2-M2 · Loader",
		"s2([S2 · Operator",
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(got, want) {
			t.Errorf("mermaid output missing substring %q\nfull output:\n%s", want, got)
		}
	}
}

func TestT46_L3EmitIdempotent(t *testing.T) {
	t.Parallel()

	spec := loadValidL3Spec(t)
	var first, second bytes.Buffer
	if err := emitL3Markdown(&first, spec, "abc1234", []string{"c3-other.md"}); err != nil {
		t.Fatalf("first emit: %v", err)
	}
	if err := emitL3Markdown(&second, spec, "abc1234", []string{"c3-other.md"}); err != nil {
		t.Fatalf("second emit: %v", err)
	}
	if first.String() != second.String() {
		t.Errorf("emit not idempotent")
	}
}

func TestT47_L3BuildLiveAndAuditClean(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	for _, name := range []string{"c3-foo-internal.json", "worker.go", "loader.go"} {
		src, err := os.ReadFile(filepath.Join("testdata/c4/valid_l3", name))
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, name), src, 0o600); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
	}
	specPath := filepath.Join(tmpDir, "c3-foo-internal.json")
	cmd := exec.CommandContext(context.Background(),
		"targ", "c4-l3-build", "--input", specPath, "--noconfirm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("c4-l3-build: %v\n%s", err, out)
	}
	mdPath := filepath.Join(tmpDir, "c3-foo-internal.md")
	findings, err := auditFile(context.Background(), mdPath)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	// Filter out parent_missing (parent c2-foo-system.md is intentionally not
	// present in the temp dir) and registry_orphan (no peer JSONs in temp dir).
	relevant := []Finding{}
	for _, finding := range findings {
		if finding.ID == "parent_missing" || finding.ID == "registry_orphan" {
			continue
		}
		relevant = append(relevant, finding)
	}
	if len(relevant) != 0 {
		t.Errorf("want 0 structural findings, got %d:\n%+v", len(relevant), relevant)
	}
}

func TestT48_L3ValidateIDs_RequiresFocusLevel2(t *testing.T) {
	t.Parallel()

	spec := loadValidL3Spec(t)
	spec.Focus.ID = "S1" // level 1, not level 2
	_, err := validateL3ElementIDs(spec)
	if err == nil {
		t.Fatal("expected error for level-1 focus id, got nil")
	}
	if !strings.Contains(err.Error(), "level 2") {
		t.Errorf("want error mentioning 'level 2', got %q", err.Error())
	}
}

func TestT49_DiscoverL3Siblings_FiltersByParent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Create three c3 files: two with same parent, one with different.
	matcher := []byte("---\nlevel: 3\nname: a\nparent: \"c2-foo.md\"\nchildren: []\nlast_reviewed_commit: x\n---\n\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "c3-a.md"), matcher, 0o600); err != nil {
		t.Fatalf("write a: %v", err)
	}
	other := []byte("---\nlevel: 3\nname: b\nparent: \"c2-foo.md\"\nchildren: []\nlast_reviewed_commit: x\n---\n\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "c3-b.md"), other, 0o600); err != nil {
		t.Fatalf("write b: %v", err)
	}
	wrong := []byte("---\nlevel: 3\nname: c\nparent: \"c2-bar.md\"\nchildren: []\nlast_reviewed_commit: x\n---\n\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "c3-c.md"), wrong, 0o600); err != nil {
		t.Fatalf("write c: %v", err)
	}
	siblings := discoverL3Siblings(filepath.Join(tmpDir, "c3-a.json"), "c2-foo.md")
	if len(siblings) != 1 || siblings[0] != "c3-b.md" {
		t.Errorf("want [c3-b.md], got %v", siblings)
	}
}

func TestT49_L3ValidateIDs_AcceptsHierarchical(t *testing.T) {
	t.Parallel()

	spec := &L3Spec{
		Focus: L3Focus{ID: "S1-N2", Name: "Foo"},
		Elements: []L3Element{
			{ID: "S2", Name: "Operator", Kind: "person", Responsibility: "x"},
			{ID: "S1-N2-M1", Name: "Worker", Kind: "component", Responsibility: "x", CodePointer: "./w.go"},
			{ID: "S1-N2-M2", Name: "Loader", Kind: "component", Responsibility: "x", CodePointer: "./l.go"},
		},
	}
	ids, err := validateL3ElementIDs(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantIDs := []string{"S2", "S1-N2-M1", "S1-N2-M2"}
	for index, want := range wantIDs {
		if ids[index].ID != want {
			t.Errorf("ids[%d]: want %s, got %s", index, want, ids[index].ID)
		}
	}
	if ids[1].AnchorID != "s1-n2-m1-worker" {
		t.Errorf("worker anchor: want s1-n2-m1-worker, got %s", ids[1].AnchorID)
	}
	if ids[0].AnchorID != "s2-operator" {
		t.Errorf("operator anchor: want s2-operator, got %s", ids[0].AnchorID)
	}
}

func TestT50_L3ValidateIDs_RejectsTooDeep(t *testing.T) {
	t.Parallel()

	spec := &L3Spec{
		Focus: L3Focus{ID: "S1-N2", Name: "Foo"},
		Elements: []L3Element{
			{ID: "S1-N2-M1-P1", Name: "TooDeep", Kind: "component", Responsibility: "x", CodePointer: "./x.go"},
		},
	}
	_, err := validateL3ElementIDs(spec)
	if err == nil {
		t.Fatal("expected error rejecting depth-4 id, got nil")
	}
	if !strings.Contains(err.Error(), "is not valid at level 3") {
		t.Errorf("want error mentioning 'is not valid at level 3', got %q", err.Error())
	}
}

func TestT51_L3ValidateIDs_RejectsOutOfFocusM(t *testing.T) {
	t.Parallel()

	spec := &L3Spec{
		Focus: L3Focus{ID: "S1-N2", Name: "Foo"},
		Elements: []L3Element{
			{ID: "S1-N3-M1", Name: "Wrong", Kind: "component", Responsibility: "x", CodePointer: "./x.go"},
		},
	}
	_, err := validateL3ElementIDs(spec)
	if err == nil {
		t.Fatal("expected error rejecting M-id outside focus, got nil")
	}
	if !strings.Contains(err.Error(), "is not valid at level 3") {
		t.Errorf("want error mentioning 'is not valid at level 3', got %q", err.Error())
	}
}

// loadValidL3Spec loads the canonical L3 fixture, fataling on any error.
func loadValidL3Spec(t *testing.T) *L3Spec {
	t.Helper()
	raw, err := os.ReadFile("testdata/c4/valid_l3/c3-foo-internal.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var spec L3Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &spec
}
