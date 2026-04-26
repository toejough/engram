//go:build targ

package dev

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestT29_AuditLiveC1_StillClean(t *testing.T) {
	t.Parallel()

	findings, err := auditFile(context.Background(), "../architecture/c4/c1-engram-system.md")
	if err != nil {
		t.Fatalf("auditFile: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("want 0 findings on live c1, got %d:\n%+v", len(findings), findings)
	}
}

func TestT30_AuditLiveC2_StillClean(t *testing.T) {
	t.Parallel()

	findings, err := auditFile(context.Background(), "../architecture/c4/c2-engram-plugin.md")
	if err != nil {
		t.Fatalf("auditFile: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("want 0 findings on live c2, got %d:\n%+v", len(findings), findings)
	}
}

func TestT31_ParseInlineYAMLArray(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  []string
	}{
		{`["foo.md", "bar.md"]`, []string{"foo.md", "bar.md"}},
		{`[]`, nil},
		{`['a', 'b']`, []string{"a", "b"}},
		{`[ unquoted ]`, []string{"unquoted"}},
		{`not an array`, nil},
	}
	for _, c := range cases {
		got := parseInlineYAMLArray(c.input)
		if !sliceEqual(got, c.want) {
			t.Errorf("parseInlineYAMLArray(%q): want %v, got %v", c.input, c.want, got)
		}
	}
}

func TestT32_CheckChildren_BadPrefix(t *testing.T) {
	t.Parallel()

	matter := frontMatter{
		hasLevel: true, level: 1,
		hasChildren: true,
		children:    []string{"c3-foo.md"},
	}
	findings := checkChildren(matter)
	if len(findings) != 1 || findings[0].ID != "child_prefix_invalid" {
		t.Errorf("want one child_prefix_invalid finding, got %+v", findings)
	}
}

func TestT33_CheckChildren_GoodPrefix(t *testing.T) {
	t.Parallel()

	matter := frontMatter{
		hasLevel: true, level: 1,
		hasChildren: true,
		children:    []string{"c2-foo.md", "c2-bar.md"},
	}
	findings := checkChildren(matter)
	if len(findings) != 0 {
		t.Errorf("want 0 findings, got %+v", findings)
	}
}

func TestT34_CheckChildren_L4ForbidsAny(t *testing.T) {
	t.Parallel()

	matter := frontMatter{
		hasLevel: true, level: 4,
		hasChildren: true,
		children:    []string{"c5-anything.md"},
	}
	findings := checkChildren(matter)
	if len(findings) != 1 || findings[0].ID != "child_prefix_invalid" {
		t.Errorf("want one child_prefix_invalid finding for L4, got %+v", findings)
	}
}

func TestT35_CheckCodePointers_MissingFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	mdPath := filepath.Join(tmp, "c3-x.md")
	raw := []byte("---\nlevel: 3\n---\n\n## Element Catalog\n\n" +
		"| ID | Name | Type | Responsibility | Code Pointer |\n" +
		"|---|---|---|---|---|\n" +
		"| <a id=\"e1\"></a>E1 | Foo | Component | does | [missing](./does-not-exist) |\n")
	if err := os.WriteFile(mdPath, raw, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	findings := checkCodePointers(frontMatter{hasLevel: true, level: 3}, raw, mdPath)
	if len(findings) != 1 || findings[0].ID != "code_pointer_unresolved" {
		t.Errorf("want one code_pointer_unresolved finding, got %+v", findings)
	}
}

func TestT36_CheckCodePointers_NotL3(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	mdPath := filepath.Join(tmp, "c1-x.md")
	raw := []byte("---\nlevel: 1\n---\n\n## Element Catalog\n\n" +
		"| ID | Name | Type | Responsibility | Code Pointer |\n" +
		"|---|---|---|---|---|\n" +
		"| <a id=\"e1\"></a>E1 | Foo | Component | does | [missing](./does-not-exist) |\n")
	findings := checkCodePointers(frontMatter{hasLevel: true, level: 1}, raw, mdPath)
	if len(findings) != 0 {
		t.Errorf("want 0 findings on L1, got %+v", findings)
	}
}

func TestT37_CheckCodePointers_PathExists(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "exists.txt")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	mdPath := filepath.Join(tmp, "c3-x.md")
	raw := []byte("---\nlevel: 3\n---\n\n## Element Catalog\n\n" +
		"| ID | Name | Type | Responsibility | Code Pointer |\n" +
		"|---|---|---|---|---|\n" +
		"| <a id=\"e1\"></a>E1 | Foo | Component | does | [exists](./exists.txt) |\n")
	findings := checkCodePointers(frontMatter{hasLevel: true, level: 3}, raw, mdPath)
	if len(findings) != 0 {
		t.Errorf("want 0 findings when target exists, got %+v", findings)
	}
}

func TestT38_RegistryCrossCheck_NoJSONsSkips(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	mdPath := filepath.Join(tmp, "c1-x.md")
	if err := os.WriteFile(mdPath, []byte(minimalL1Markdown), 0o600); err != nil {
		t.Fatalf("write md: %v", err)
	}
	findings := checkRegistryCrossCheck(context.Background(), []byte(minimalL1Markdown), mdPath)
	if len(findings) != 0 {
		t.Errorf("want 0 findings when no JSONs in dir, got %+v", findings)
	}
}

func TestT39_RegistryCrossCheck_OrphanMarkdown(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	// Place a peer JSON so the dir is non-empty for the registry, but the
	// audited markdown's slug doesn't match.
	peerJSON := []byte(`{"schema_version":"1","level":1,"name":"peer","parent":null,
		"preamble":"x","elements":[
		{"name":"User","kind":"person","responsibility":"u","system_of_record":"h"},
		{"name":"Peer","kind":"container","is_system":true,"responsibility":"t","system_of_record":"r"}
		],"relationships":[{"from":"User","to":"Peer","description":"uses","protocol":"tty"}],
		"drift_notes":[],"cross_links":{"refined_by":[]}}`)
	if err := os.WriteFile(filepath.Join(tmp, "c1-peer.json"), peerJSON, 0o600); err != nil {
		t.Fatalf("write peer: %v", err)
	}
	mdPath := filepath.Join(tmp, "c1-orphan.md")
	if err := os.WriteFile(mdPath, []byte(minimalL1Markdown), 0o600); err != nil {
		t.Fatalf("write md: %v", err)
	}
	findings := checkRegistryCrossCheck(context.Background(), []byte(minimalL1Markdown), mdPath)
	if len(findings) != 1 || findings[0].ID != "registry_orphan" {
		t.Errorf("want one registry_orphan finding, got %+v", findings)
	}
}

func TestT40_RegistryCrossCheck_DriftEmitsFinding(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	// JSON says E2 is "Other Tiny"; the audited markdown's catalog says E2 is
	// "Tiny" — should fire id_name_drift.
	driftJSON := []byte(`{"schema_version":"1","level":1,"name":"tiny","parent":null,
		"preamble":"x","elements":[
		{"name":"User","kind":"person","responsibility":"u","system_of_record":"h"},
		{"name":"Other Tiny","kind":"container","is_system":true,"responsibility":"t","system_of_record":"r"}
		],"relationships":[{"from":"User","to":"Other Tiny","description":"uses","protocol":"tty"}],
		"drift_notes":[],"cross_links":{"refined_by":[]}}`)
	if err := os.WriteFile(filepath.Join(tmp, "c1-tiny.json"), driftJSON, 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}
	mdPath := filepath.Join(tmp, "c1-tiny.md")
	if err := os.WriteFile(mdPath, []byte(minimalL1Markdown), 0o600); err != nil {
		t.Fatalf("write md: %v", err)
	}
	findings := checkRegistryCrossCheck(context.Background(), []byte(minimalL1Markdown), mdPath)
	driftFound := false
	for _, finding := range findings {
		if finding.ID == "id_name_drift" {
			driftFound = true
		}
	}
	if !driftFound {
		t.Errorf("want id_name_drift finding, got %+v", findings)
	}
}

func TestT41_ParseCatalogIDNames_ExtractsRows(t *testing.T) {
	t.Parallel()

	pairs := parseCatalogIDNames([]byte(minimalL1Markdown))
	if len(pairs) != 2 {
		t.Fatalf("want 2 catalog pairs, got %d:\n%+v", len(pairs), pairs)
	}
	wantIDs := []string{"E1", "E2"}
	wantNames := []string{"User", "Tiny"}
	for index, pair := range pairs {
		if pair.id != wantIDs[index] || pair.name != wantNames[index] {
			t.Errorf("pair[%d]: want (%s, %s), got (%s, %s)",
				index, wantIDs[index], wantNames[index], pair.id, pair.name)
		}
	}
}

// unexported constants.
const (
	minimalL1Markdown = "---\nlevel: 1\nname: tiny\nparent: null\n" +
		"children: []\nlast_reviewed_commit: HEAD\n---\n\n" +
		"# C1 — Tiny (System Context)\n\nx\n\n" +
		"```mermaid\n" +
		"flowchart LR\n" +
		"    classDef person      fill:#08427b,stroke:#052e56,color:#fff\n" +
		"    classDef external    fill:#999,   stroke:#666,   color:#fff\n" +
		"    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff\n" +
		"    e1([E1 · User])\n" +
		"    e2[E2 · Tiny]\n" +
		"    e1 -->|\"R1: uses\"| e2\n" +
		"    class e1 person\n" +
		"    class e2 container\n" +
		"    click e1 href \"#e1-user\"\n" +
		"    click e2 href \"#e2-tiny\"\n" +
		"```\n\n" +
		"## Element Catalog\n\n" +
		"| ID | Name | Type | Responsibility | System of Record |\n" +
		"|---|---|---|---|---|\n" +
		"| <a id=\"e1-user\"></a>E1 | User | Person | uses | human |\n" +
		"| <a id=\"e2-tiny\"></a>E2 | Tiny | The system in scope | t | r |\n\n" +
		"## Relationships\n\n" +
		"| ID | From | To | Description | Protocol/Medium |\n" +
		"|---|---|---|---|---|\n" +
		"| <a id=\"r1-user-tiny\"></a>R1 | User | Tiny | uses | tty |\n\n" +
		"## Cross-links\n\n- Parent: none (L1 is the root).\n- Refined by: *(none yet)*\n"
)

// sliceEqual returns true when two string slices have the same length and
// element-wise equality. nil and empty slice are treated as equal.
func sliceEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
