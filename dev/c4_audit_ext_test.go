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
		"| <a id=\"s1-foo\"></a>S1 | Foo | Component | does | [missing](./does-not-exist) |\n")
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
		"| <a id=\"s1-foo\"></a>S1 | Foo | Component | does | [missing](./does-not-exist) |\n")
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
		"| <a id=\"s1-foo\"></a>S1 | Foo | Component | does | [exists](./exists.txt) |\n")
	findings := checkCodePointers(frontMatter{hasLevel: true, level: 3}, raw, mdPath)
	if len(findings) != 0 {
		t.Errorf("want 0 findings when target exists, got %+v", findings)
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
		"    s1([S1 · User])\n" +
		"    s2[S2 · Tiny]\n" +
		"    s1 -->|\"R1: uses\"| s2\n" +
		"    class s1 person\n" +
		"    class s2 container\n" +
		"    click s1 href \"#s1-user\"\n" +
		"    click s2 href \"#s2-tiny\"\n" +
		"```\n\n" +
		"## Element Catalog\n\n" +
		"| ID | Name | Type | Responsibility | System of Record |\n" +
		"|---|---|---|---|---|\n" +
		"| <a id=\"s1-user\"></a>S1 | User | Person | uses | human |\n" +
		"| <a id=\"s2-tiny\"></a>S2 | Tiny | The system in scope | t | r |\n\n" +
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
