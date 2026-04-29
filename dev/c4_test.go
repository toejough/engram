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

func TestAuditFile_L4CarryoverEmitsFindings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// L3 with focus F and one neighbor M.
	l3 := []byte(`{"schema_version":"1","level":3,"name":"x","parent":"c2-x.md","focus":{"id":"S2-N3","name":"x","responsibility":"r"},"elements":[{"id":"F","name":"focus","kind":"component"},{"id":"M","name":"memory","kind":"component"}],"relationships":[{"from":"F","to":"M","description":"writes","protocol":"go"}]}`)
	if err := os.WriteFile(filepath.Join(dir, "c3-x.json"), l3, 0o600); err != nil {
		t.Fatalf("write c3-x.json: %v", err)
	}

	// L4 referencing F as focus, no M, plus an extra "GHOST" node — two violations.
	l4JSON := []byte(`{"schema_version":"1","level":4,"name":"focus","parent":"c3-x.md","focus":{"id":"F","name":"focus"},"diagram":{"nodes":[{"id":"F","name":"focus","kind":"focus"},{"id":"GHOST","name":"ghost","kind":"component"}],"edges":[]},"dependency_manifest":[],"di_wires":[]}`)
	if err := os.WriteFile(filepath.Join(dir, "c4-focus.json"), l4JSON, 0o600); err != nil {
		t.Fatalf("write c4-focus.json: %v", err)
	}

	jsonPath := filepath.Join(dir, "c4-focus.json")
	findings, err := auditFile(context.Background(), jsonPath)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	var carryover []Finding
	for _, f := range findings {
		if f.ID == "l4_carryover" {
			carryover = append(carryover, f)
		}
	}
	if len(carryover) != 2 {
		t.Fatalf("expected 2 l4_carryover findings (extra GHOST + missing M), got %d: %+v", len(carryover), carryover)
	}
}

func TestAuditFile_L4CarryoverUnreadableWhenL3ParentMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Valid L4 JSON pointing at an L3 parent that does not exist on disk.
	l4JSON := []byte(`{"schema_version":"1","level":4,"name":"focus","parent":"c3-missing.md","focus":{"id":"F","name":"focus"},"diagram":{"nodes":[{"id":"F","name":"focus","kind":"focus"}],"edges":[]},"dependency_manifest":[],"di_wires":[]}`)
	jsonPath := filepath.Join(dir, "c4-focus.json")
	if err := os.WriteFile(jsonPath, l4JSON, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	findings, err := auditFile(context.Background(), jsonPath)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	var seen int
	for _, f := range findings {
		if f.ID == "l4_carryover_unreadable" {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("expected 1 l4_carryover_unreadable finding, got %d: %+v", seen, findings)
	}
}

func TestT10_BuildLiveC1_AuditsClean(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "c1-engram-system.json")
	src, err := os.ReadFile("../architecture/c4/c1-engram-system.json")
	if err != nil {
		t.Fatalf("read source spec: %v", err)
	}
	if err := os.WriteFile(specPath, src, 0o600); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	cmd := exec.CommandContext(context.Background(),
		"targ", "c4-l1-build", "--input", specPath, "--noconfirm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("c4-l1-build: %v\n%s", err, out)
	}
	findings, err := auditFile(context.Background(), specPath)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected zero findings on built file, got %d:\n%+v", len(findings), findings)
	}
}

func TestT11_ExternalsHTTPDetection(t *testing.T) {
	t.Parallel()

	findings, err := scanExternals(context.Background(), "testdata/c4/scanrepo", "./...", false)
	if err != nil {
		t.Fatalf("scanExternals: %v", err)
	}
	httpFindings := []externalFinding{}
	for _, finding := range findings {
		if finding.Kind == "http_call" {
			httpFindings = append(httpFindings, finding)
		}
	}
	if len(httpFindings) < 1 {
		t.Fatalf("want >=1 http_call finding, got %d:\n%+v", len(httpFindings), httpFindings)
	}
	literalSeen := false
	for _, finding := range httpFindings {
		if finding.Target == "https://api.example.com/v1/things" {
			literalSeen = true
		}
	}
	if !literalSeen {
		t.Errorf("expected literal-URL finding among:\n%+v", httpFindings)
	}
}

func TestT12_ExternalsAllKindsOnScanrepo(t *testing.T) {
	t.Parallel()

	findings, err := scanExternals(context.Background(), "testdata/c4/scanrepo", "./...", false)
	if err != nil {
		t.Fatalf("scanExternals: %v", err)
	}
	want := map[string]bool{
		"http_call": false, "fs_path": false, "exec": false, "env_read": false, "data_format": false,
	}
	for _, finding := range findings {
		want[finding.Kind] = true
	}
	for kind, found := range want {
		if !found {
			t.Errorf("kind %q not detected on scanrepo", kind)
		}
	}
}

func TestT12_ExternalsCLIJSON(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(context.Background(),
		"targ", "c4-l1-externals", "--root", "testdata/c4/scanrepo", "--packages", "./...")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("c4-l1-externals: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `"schema_version": "1"`) {
		t.Errorf("output missing schema_version:\n%s", out)
	}
	if !strings.Contains(string(out), `"http_call"`) {
		t.Errorf("output missing http_call kind:\n%s", out)
	}
}

func TestT13_HistoryParsesCommitsWithFileChanges(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	mustRunIn(t, repoDir, "git", "init")
	mustRunIn(t, repoDir, "git", "config", "user.email", "a@b")
	mustRunIn(t, repoDir, "git", "config", "user.name", "Tester")
	if err := os.WriteFile(filepath.Join(repoDir, "a.txt"), []byte("a\n"), 0o600); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	mustRunIn(t, repoDir, "git", "add", "a.txt")
	mustRunIn(t, repoDir, "git", "commit", "-m", "first")
	if err := os.WriteFile(filepath.Join(repoDir, "b.txt"), []byte("b\n"), 0o600); err != nil {
		t.Fatalf("write b.txt: %v", err)
	}
	mustRunIn(t, repoDir, "git", "add", "b.txt")
	mustRunIn(t, repoDir, "git", "commit", "-m", "second")
	commits, err := scanHistory(context.Background(), historyOptions{root: repoDir, limit: 10})
	if err != nil {
		t.Fatalf("scanHistory: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("want 2 commits, got %d:\n%+v", len(commits), commits)
	}
	if commits[0].Subject != "second" {
		t.Errorf("first commit subject mangled: %q", commits[0].Subject)
	}
	if len(commits[0].FilesChanged) != 1 || commits[0].FilesChanged[0].Path != "b.txt" {
		t.Errorf("commit[0] files: want [b.txt], got %+v", commits[0].FilesChanged)
	}
	if commits[1].Subject != "first" {
		t.Errorf("second commit subject mangled: %q", commits[1].Subject)
	}
	if len(commits[1].FilesChanged) != 1 || commits[1].FilesChanged[0].Path != "a.txt" {
		t.Errorf("commit[1] files: want [a.txt], got %+v", commits[1].FilesChanged)
	}
}

func TestT13_HistoryParsesRealGitLog(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	mustRunIn(t, repoDir, "git", "init")
	mustRunIn(t, repoDir, "git", "config", "user.email", "a@b")
	mustRunIn(t, repoDir, "git", "config", "user.name", "Tester")
	mustRunIn(t, repoDir, "git", "commit", "--allow-empty", "-m", "first\n\nbody line one\nbody line two")
	mustRunIn(t, repoDir, "git", "commit", "--allow-empty", "-m", "second")
	commits, err := scanHistory(context.Background(), historyOptions{root: repoDir, limit: 10})
	if err != nil {
		t.Fatalf("scanHistory: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("want 2 commits, got %d:\n%+v", len(commits), commits)
	}
	if commits[0].Subject != "second" {
		t.Errorf("expected newest first; got subject %q", commits[0].Subject)
	}
	if !strings.Contains(commits[1].Body, "body line one") {
		t.Errorf("expected body content in second commit; got %q", commits[1].Body)
	}
}

func TestT17_DataFormatJSON(t *testing.T) {
	t.Parallel()

	findings, err := scanExternals(context.Background(), "testdata/c4/scanrepo", "./...", false)
	if err != nil {
		t.Fatalf("scanExternals: %v", err)
	}
	saw := false
	for _, finding := range findings {
		if finding.Kind == "data_format" && finding.Target == "json" {
			saw = true
		}
	}
	if !saw {
		t.Errorf("expected a data_format=json finding; got: %+v", findings)
	}
}

func TestT17_FSWriteFileAlwaysEmits(t *testing.T) {
	t.Parallel()

	findings, err := scanExternals(context.Background(), "testdata/c4/scanrepo", "./...", false)
	if err != nil {
		t.Fatalf("scanExternals: %v", err)
	}
	saw := false
	for _, finding := range findings {
		if finding.Kind == "fs_path" && strings.Contains(finding.Evidence, "WriteFile") {
			saw = true
		}
	}
	if !saw {
		t.Errorf("expected an fs_path finding for os.WriteFile; got: %+v", findings)
	}
}

func TestT17_HTTPDynamicURLAlwaysEmits(t *testing.T) {
	t.Parallel()

	findings, err := scanExternals(context.Background(), "testdata/c4/scanrepo", "./...", false)
	if err != nil {
		t.Fatalf("scanExternals: %v", err)
	}
	httpCount := 0
	dynamicCount := 0
	for _, finding := range findings {
		if finding.Kind != "http_call" {
			continue
		}
		httpCount++
		if finding.Target == "<dynamic>" {
			dynamicCount++
		}
	}
	if httpCount < 2 {
		t.Errorf("want >=2 http_call findings (literal + dynamic), got %d", httpCount)
	}
	if dynamicCount < 1 {
		t.Errorf("want >=1 dynamic-URL http_call finding, got %d", dynamicCount)
	}
}

func TestT17_HistorySinceShorthand(t *testing.T) {
	t.Parallel()

	cases := []struct{ in, want string }{
		{"30d", "30 days ago"},
		{"2w", "2 weeks ago"},
		{"6m", "6 months ago"},
		{"1y", "1 years ago"},
		{"yesterday", "yesterday"},
		{"2026-01-01", "2026-01-01"},
		{"", ""},
	}
	for _, testCase := range cases {
		got := translateSinceShorthand(testCase.in)
		if got != testCase.want {
			t.Errorf("translateSinceShorthand(%q): want %q, got %q", testCase.in, testCase.want, got)
		}
	}
}

func TestT1_TargsRegistered(t *testing.T) {
	t.Parallel()

	want := []string{"c4-audit", "c4-l1-build", "c4-l1-externals", "c4-l2-build", "c4-l3-build", "c4-history"}
	out, err := exec.CommandContext(context.Background(), "targ").CombinedOutput()
	if err != nil {
		t.Fatalf("targ: %v\n%s", err, out)
	}
	for _, name := range want {
		if !strings.Contains(string(out), name) {
			t.Errorf("targ list missing %q\noutput: %s", name, out)
		}
	}
}

func TestT5_AuditLiveC1_ZeroFindings(t *testing.T) {
	t.Parallel()

	findings, err := auditFile(context.Background(), "../architecture/c4/c1-engram-system.json")
	if err != nil {
		t.Fatalf("auditFile: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("live c1 should audit clean; got %d findings:\n%+v", len(findings), findings)
	}
}

func TestT7_BuildValidates_AcceptsValid(t *testing.T) {
	t.Parallel()

	spec, err := loadAndValidateSpec("testdata/c4/valid_l1.json")
	if err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}
	if spec.Level != 1 {
		t.Errorf("level: want 1, got %d", spec.Level)
	}
	if spec.Name != "engram-system" {
		t.Errorf("name: want engram-system, got %q", spec.Name)
	}
}

func TestT7_BuildValidates_RejectsBadSchemas(t *testing.T) {
	t.Parallel()

	cases := []struct {
		file   string
		errSub string
	}{
		{"testdata/c4/invalid_schema_version.json", "schema_version"},
		{"testdata/c4/invalid_no_system.json", "exactly one container"},
		{"testdata/c4/invalid_two_systems.json", "exactly one container"},
		{"testdata/c4/invalid_dup_names.json", "duplicate"},
		{"testdata/c4/invalid_dangling_rel.json", "elements"},
		{"testdata/c4/invalid_l1_bad_id.json", "id path"},
		{"testdata/c4/invalid_l1_dup_id.json", "duplicate id"},
		{"testdata/c4/invalid_l1_missing_id.json", "missing id"},
	}
	for _, testCase := range cases {
		t.Run(testCase.file, func(t *testing.T) {
			t.Parallel()
			_, err := loadAndValidateSpec(testCase.file)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", testCase.errSub)
			}
			if !strings.Contains(err.Error(), testCase.errSub) {
				t.Errorf("error %q does not contain %q", err.Error(), testCase.errSub)
			}
		})
	}
}

func TestT8_AssignIDs_ReadsExplicitSIDs(t *testing.T) {
	t.Parallel()

	elements := []L1Element{
		{ID: "S1", Name: "Joe", Kind: "person"},
		{ID: "S2", Name: "Engram plugin", Kind: "container"},
		{ID: "S3", Name: "Claude Code", Kind: "external"},
	}
	ids, err := assignElementIDs(elements)
	if err != nil {
		t.Fatalf("assignElementIDs: %v", err)
	}
	if len(ids) != 3 || ids[0].ID != "S1" || ids[1].ID != "S2" || ids[2].ID != "S3" {
		t.Errorf("unexpected IDs: %+v", ids)
	}
	if ids[1].AnchorID != "s2-engram-plugin" {
		t.Errorf("wrong anchor: %s", ids[1].AnchorID)
	}
	if ids[0].AnchorID != "s1-joe" {
		t.Errorf("wrong anchor: %s", ids[0].AnchorID)
	}
}

func TestT8_AssignIDs_RejectsNonLevel1ID(t *testing.T) {
	t.Parallel()

	elements := []L1Element{
		{ID: "S1-N2", Name: "Joe", Kind: "person"},
	}
	_, err := assignElementIDs(elements)
	if err == nil {
		t.Fatalf("expected error for non-L1 id, got nil")
	}
}

func TestT8_Slug_Cases(t *testing.T) {
	t.Parallel()

	cases := []struct{ in, want string }{
		{"Joe", "joe"},
		{"Engram plugin", "engram-plugin"},
		{"Claude Code memory surfaces", "claude-code-memory-surfaces"},
		{"Foo/Bar Baz", "foo-bar-baz"},
		{"  Trim Me  ", "trim-me"},
		{"---hyphens---", "hyphens"},
	}
	for _, testCase := range cases {
		got := slug(testCase.in)
		if got != testCase.want {
			t.Errorf("slug(%q): want %q, got %q", testCase.in, testCase.want, got)
		}
	}
}

func TestT9_BuildEmitsCanonicalMarkdown(t *testing.T) {
	t.Parallel()

	spec, err := loadAndValidateSpec("testdata/c4/valid_l1.json")
	if err != nil {
		t.Fatalf("loadAndValidateSpec: %v", err)
	}
	var buf bytes.Buffer
	if err := emitMarkdown(&buf, spec, "df51bc93"); err != nil {
		t.Fatalf("emitMarkdown: %v", err)
	}
	want, err := os.ReadFile("testdata/c4/valid_l1.md")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if buf.String() != string(want) {
		t.Errorf("output diff:\n--- want\n%s\n+++ got\n%s", want, buf.String())
	}
}

func TestT9_BuildIdempotent(t *testing.T) {
	t.Parallel()

	spec, err := loadAndValidateSpec("testdata/c4/valid_l1.json")
	if err != nil {
		t.Fatalf("loadAndValidateSpec: %v", err)
	}
	var buf1, buf2 bytes.Buffer
	if err := emitMarkdown(&buf1, spec, "abc1234"); err != nil {
		t.Fatalf("first emit: %v", err)
	}
	if err := emitMarkdown(&buf2, spec, "abc1234"); err != nil {
		t.Fatalf("second emit: %v", err)
	}
	if buf1.String() != buf2.String() {
		t.Error("emitMarkdown is not deterministic")
	}
}

func mustRunIn(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
