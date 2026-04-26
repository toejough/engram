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
			name: "from_parent component",
			mutate: func(s *L3Spec) {
				for index := range s.Elements {
					if s.Elements[index].Kind == "component" {
						s.Elements[index].FromParent = true
						return
					}
				}
			},
			wantError: "from_parent",
		},
		{
			name: "relationship to unknown element",
			mutate: func(s *L3Spec) {
				s.Relationships[0].To = "Nonexistent"
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
		"classDef component",
		"subgraph e2 [E2 · Foo]",
		"e10[E10 · Worker",
		"e11[E11 · Loader",
		"e1([E1 · Operator",
		"## Element Catalog",
		"| Code Pointer |",
		"<a id=\"e2-foo\"></a>E2 | Foo | Container in focus",
		"<a id=\"e10-worker\"></a>E10 | Worker | Component",
		"[./worker.go](./worker.go)",
		"## Relationships",
		"R1 | Operator | Worker",
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

func TestT48_L3BuildRegistryRejection_FromParentNameMismatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Peer L1 spec defines E1 as "Owner", not "Operator".
	peerJSON := []byte(`{"schema_version":"1","level":1,"name":"foo-system","parent":null,
		"preamble":"x","elements":[
		{"name":"Owner","kind":"person","responsibility":"u","system_of_record":"h"},
		{"name":"Foo","kind":"container","is_system":true,"responsibility":"t","system_of_record":"r"}
		],"relationships":[{"from":"Owner","to":"Foo","description":"uses","protocol":"tty"}],
		"drift_notes":[],"cross_links":{"refined_by":[]}}`)
	if err := os.WriteFile(filepath.Join(tmpDir, "c1-foo-system.json"), peerJSON, 0o600); err != nil {
		t.Fatalf("write peer: %v", err)
	}
	// L3 spec claims E1 = "Operator" with from_parent. Should be rejected.
	src, err := os.ReadFile("testdata/c4/valid_l3/c3-foo-internal.json")
	if err != nil {
		t.Fatalf("read source spec: %v", err)
	}
	specPath := filepath.Join(tmpDir, "c3-foo-internal.json")
	if err := os.WriteFile(specPath, src, 0o600); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	cmd := exec.CommandContext(context.Background(),
		"targ", "c4-l3-build", "--input", specPath, "--noconfirm")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("want build failure on registry mismatch, got success:\n%s", out)
	}
	if !strings.Contains(string(out), "Operator") || !strings.Contains(string(out), "Owner") {
		t.Errorf("expected error to cite both Operator and Owner, got: %s", out)
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
