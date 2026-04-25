//go:build targ

package dev

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestT1_TargsRegistered(t *testing.T) {
	t.Parallel()

	want := []string{"c4-audit", "c4-l1-build", "c4-l1-externals", "c4-history"}
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

func TestT2_AuditClean_NoFindings(t *testing.T) {
	t.Parallel()

	findings, err := auditFile(context.Background(), "testdata/c4/audit_clean.md")
	if err != nil {
		t.Fatalf("auditFile: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("want 0 findings, got %d:\n%v", len(findings), findings)
	}
}

func TestT2_AuditDirtyFrontmatter_AllFindings(t *testing.T) {
	t.Parallel()

	findings, err := auditFile(context.Background(), "testdata/c4/audit_dirty_frontmatter.md")
	if err != nil {
		t.Fatalf("auditFile: %v", err)
	}
	wantIDs := map[string]bool{
		"front_matter_field_missing":   false,
		"level_invalid":                false,
		"name_filename_mismatch":       false,
		"parent_missing":               false,
		"last_reviewed_commit_invalid": false,
	}
	for _, finding := range findings {
		if _, ok := wantIDs[finding.ID]; ok {
			wantIDs[finding.ID] = true
		}
	}
	hits := 0
	for _, present := range wantIDs {
		if present {
			hits++
		}
	}
	if hits < 3 {
		t.Errorf("want >=3 distinct front-matter findings, got %d:\n%+v", hits, findings)
	}
}

func TestT3_AuditDirtyMermaid_FindsBlockIssues(t *testing.T) {
	t.Parallel()

	findings, err := auditFile(context.Background(), "testdata/c4/audit_dirty_mermaid.md")
	if err != nil {
		t.Fatalf("auditFile: %v", err)
	}
	wantIDs := []string{"classdef_missing", "node_id_missing", "edge_id_missing"}
	got := map[string]bool{}
	for _, finding := range findings {
		got[finding.ID] = true
	}
	for _, id := range wantIDs {
		if !got[id] {
			t.Errorf("missing finding id %q in:\n%+v", id, findings)
		}
	}
}

func TestT6_AuditCLI_JSONFormat(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(context.Background(),
		"targ", "c4-audit", "--file", "testdata/c4/audit_dirty_orphans.md", "--json")
	out, _ := cmd.CombinedOutput()
	if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit, got %d\nout: %s", cmd.ProcessState.ExitCode(), out)
	}
	if !strings.Contains(string(out), `"schema_version": "1"`) {
		t.Errorf("expected JSON with schema_version, got:\n%s", out)
	}
	if !strings.Contains(string(out), `"findings":`) {
		t.Errorf("expected JSON findings array, got:\n%s", out)
	}
}

func TestT5_AuditDirtyAnchors_FindsClickAndAnchorIssues(t *testing.T) {
	t.Parallel()

	findings, err := auditFile(context.Background(), "testdata/c4/audit_dirty_anchors.md")
	if err != nil {
		t.Fatalf("auditFile: %v", err)
	}
	wantIDs := []string{"click_missing", "click_target_unresolved", "anchor_missing"}
	got := map[string]bool{}
	for _, finding := range findings {
		got[finding.ID] = true
	}
	for _, id := range wantIDs {
		if !got[id] {
			t.Errorf("missing finding %q in:\n%+v", id, findings)
		}
	}
}

func TestT5_AuditLiveC1_ZeroFindings(t *testing.T) {
	t.Parallel()

	findings, err := auditFile(context.Background(), "../architecture/c4/c1-engram-system.md")
	if err != nil {
		t.Fatalf("auditFile: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("live c1 should audit clean; got %d findings:\n%+v", len(findings), findings)
	}
}

func TestT4_AuditDirtyOrphans_FindsBidirectionalMismatch(t *testing.T) {
	t.Parallel()

	findings, err := auditFile(context.Background(), "testdata/c4/audit_dirty_orphans.md")
	if err != nil {
		t.Fatalf("auditFile: %v", err)
	}
	wantIDs := []string{"node_orphan", "catalog_orphan", "edge_orphan", "relationships_orphan"}
	got := map[string]bool{}
	for _, finding := range findings {
		got[finding.ID] = true
	}
	for _, id := range wantIDs {
		if !got[id] {
			t.Errorf("missing finding %q in:\n%+v", id, findings)
		}
	}
}
