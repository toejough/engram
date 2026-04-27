//go:build targ

package dev

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestT18_L2BuildValidates_AcceptsValid(t *testing.T) {
	t.Parallel()

	spec, err := loadAndValidateL2Spec("testdata/c4/valid_l2.json")
	if err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}
	if spec.Level != 2 {
		t.Errorf("level: want 2, got %d", spec.Level)
	}
	if spec.Name != "foo-system" {
		t.Errorf("name: want foo-system, got %q", spec.Name)
	}
	if spec.Parent != "c1-foo-system.md" {
		t.Errorf("parent: want c1-foo-system.md, got %q", spec.Parent)
	}
}

func TestT18_L2BuildValidates_RejectsBadSchemas(t *testing.T) {
	t.Parallel()

	cases := []struct {
		file   string
		errSub string
	}{
		{"testdata/c4/invalid_l2_no_in_scope.json", "in_scope"},
		{"testdata/c4/invalid_l2_two_in_scope.json", "exactly one"},
		{"testdata/c4/invalid_l2_dup_id.json", "duplicate id"},
		{"testdata/c4/invalid_l2_empty_parent.json", "parent"},
		{"testdata/c4/invalid_l2_dangling_rel.json", "elements"},
	}
	for _, testCase := range cases {
		t.Run(testCase.file, func(t *testing.T) {
			t.Parallel()
			_, err := loadAndValidateL2Spec(testCase.file)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", testCase.errSub)
			}
			if !strings.Contains(err.Error(), testCase.errSub) {
				t.Errorf("error %q does not contain %q", err.Error(), testCase.errSub)
			}
		})
	}
}

func TestT19_L2BuildEmitsCanonicalMarkdown(t *testing.T) {
	t.Parallel()

	spec, err := loadAndValidateL2Spec("testdata/c4/valid_l2.json")
	if err != nil {
		t.Fatalf("loadAndValidateL2Spec: %v", err)
	}
	var buf bytes.Buffer
	if err := emitL2Markdown(&buf, spec, "df51bc93"); err != nil {
		t.Fatalf("emitL2Markdown: %v", err)
	}
	want, err := os.ReadFile("testdata/c4/valid_l2.md")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if buf.String() != string(want) {
		t.Errorf("output diff:\n--- want\n%s\n+++ got\n%s", want, buf.String())
	}
}

func TestT19_L2BuildIdempotent(t *testing.T) {
	t.Parallel()

	spec, err := loadAndValidateL2Spec("testdata/c4/valid_l2.json")
	if err != nil {
		t.Fatalf("loadAndValidateL2Spec: %v", err)
	}
	var buf1, buf2 bytes.Buffer
	if err := emitL2Markdown(&buf1, spec, "abc1234"); err != nil {
		t.Fatalf("first emit: %v", err)
	}
	if err := emitL2Markdown(&buf2, spec, "abc1234"); err != nil {
		t.Fatalf("second emit: %v", err)
	}
	if buf1.String() != buf2.String() {
		t.Error("emitL2Markdown is not deterministic")
	}
}

func TestT19_L2ValidateIDs_RequiresInScope(t *testing.T) {
	t.Parallel()

	elements := []L2Element{
		{ID: "S1", Name: "Person", Kind: "person"},
		{ID: "S2", Name: "Sys", Kind: "container"},
	}
	_, err := validateL2ElementIDs(elements)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no in_scope element") {
		t.Errorf("want error mentioning 'no in_scope element', got %q", err.Error())
	}
}

func TestT20_L2BuildLiveC2_AuditsClean(t *testing.T) {
	t.Parallel()

	livePath := "../architecture/c4/c2-engram-plugin.json"
	if _, err := os.Stat(livePath); err != nil {
		t.Skipf("live c2 spec not present yet: %v", err)
	}
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "c2-engram-plugin.json")
	src, err := os.ReadFile(livePath)
	if err != nil {
		t.Fatalf("read source spec: %v", err)
	}
	if err := os.WriteFile(specPath, src, 0o600); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	parentSrc, err := os.ReadFile("../architecture/c4/c1-engram-system.md")
	if err != nil {
		t.Fatalf("read parent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "c1-engram-system.md"), parentSrc, 0o600); err != nil {
		t.Fatalf("write parent: %v", err)
	}
	cmd := exec.CommandContext(context.Background(),
		"targ", "c4-l2-build", "--input", specPath, "--noconfirm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("c4-l2-build: %v\n%s", err, out)
	}
	mdPath := filepath.Join(tmpDir, "c2-engram-plugin.md")
	findings, err := auditFile(context.Background(), mdPath)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected zero findings on built file, got %d:\n%+v", len(findings), findings)
	}
}

func TestT20_L2ValidateIDs_AcceptsHierarchical(t *testing.T) {
	t.Parallel()

	elements := []L2Element{
		{ID: "S1", Name: "Person", Kind: "person"},
		{ID: "S2", Name: "Focus", Kind: "container", InScope: true},
		{ID: "S3", Name: "Peer", Kind: "external"},
		{ID: "S2-N1", Name: "Inner1", Kind: "container"},
		{ID: "S2-N2", Name: "Inner2", Kind: "container"},
	}
	ids, err := validateL2ElementIDs(elements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantIDs := []string{"S1", "S2", "S3", "S2-N1", "S2-N2"}
	for index, want := range wantIDs {
		if ids[index].ID != want {
			t.Errorf("ids[%d]: want %s, got %s", index, want, ids[index].ID)
		}
	}
	if ids[3].AnchorID != "s2-n1-inner1" {
		t.Errorf("inner1 anchor: want s2-n1-inner1, got %s", ids[3].AnchorID)
	}
	if ids[0].AnchorID != "s1-person" {
		t.Errorf("person anchor: want s1-person, got %s", ids[0].AnchorID)
	}
}

func TestT21_L2ValidateIDs_RejectsBadDepth(t *testing.T) {
	t.Parallel()

	elements := []L2Element{
		{ID: "S2", Name: "Focus", Kind: "container", InScope: true},
		{ID: "S2-N3-M5", Name: "TooDeep", Kind: "container"},
	}
	_, err := validateL2ElementIDs(elements)
	if err == nil {
		t.Fatal("expected error rejecting depth-3 id, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported L2 id depth") {
		t.Errorf("want error mentioning 'unsupported L2 id depth', got %q", err.Error())
	}
}

func TestT22_L2ValidateIDs_RejectsOutOfFocusN(t *testing.T) {
	t.Parallel()

	elements := []L2Element{
		{ID: "S2", Name: "Focus", Kind: "container", InScope: true},
		{ID: "S3-N1", Name: "Wrong", Kind: "container"},
	}
	_, err := validateL2ElementIDs(elements)
	if err == nil {
		t.Fatal("expected error rejecting N-id outside focus, got nil")
	}
	if !strings.Contains(err.Error(), "not under focus") {
		t.Errorf("want error mentioning 'not under focus', got %q", err.Error())
	}
}
