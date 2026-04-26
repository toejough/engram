//go:build targ

package dev

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestT21_RegistryClean_ZeroConflicts(t *testing.T) {
	t.Parallel()

	files, records, err := scanRegistryDir(context.Background(), "testdata/c4/registry_clean")
	if err != nil {
		t.Fatalf("scanRegistryDir: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("want >=2 files in registry_clean, got %d", len(files))
	}
	view := deriveRegistry("testdata/c4/registry_clean", files, records)
	if len(view.Conflicts) != 0 {
		t.Errorf("want 0 conflicts on clean fixture, got %d:\n%+v", len(view.Conflicts), view.Conflicts)
	}
}

func TestT22_RegistryDetectsIDNameDrift(t *testing.T) {
	t.Parallel()

	files, records, err := scanRegistryDir(context.Background(), "testdata/c4/registry_id_name_drift")
	if err != nil {
		t.Fatalf("scanRegistryDir: %v", err)
	}
	view := deriveRegistry("testdata/c4/registry_id_name_drift", files, records)
	found := false
	for _, conflict := range view.Conflicts {
		if conflict.Kind == "id_name_drift" && conflict.ID == "E2" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected id_name_drift on E2, got conflicts:\n%+v", view.Conflicts)
	}
}

func TestT23_RegistryDetectsNameIDSplit(t *testing.T) {
	t.Parallel()

	files, records, err := scanRegistryDir(context.Background(), "testdata/c4/registry_name_id_split")
	if err != nil {
		t.Fatalf("scanRegistryDir: %v", err)
	}
	view := deriveRegistry("testdata/c4/registry_name_id_split", files, records)
	found := false
	for _, conflict := range view.Conflicts {
		if conflict.Kind == "name_id_split" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected at least one name_id_split, got conflicts:\n%+v", view.Conflicts)
	}
}

func TestT24_RegistryDetectsIDCollisionWithinFile(t *testing.T) {
	t.Parallel()

	files, records, err := scanRegistryDir(context.Background(),
		"testdata/c4/registry_id_collision_within_file")
	if err != nil {
		t.Fatalf("scanRegistryDir: %v", err)
	}
	view := deriveRegistry("testdata/c4/registry_id_collision_within_file", files, records)
	found := false
	for _, conflict := range view.Conflicts {
		if conflict.Kind == "id_collision_within_file" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected id_collision_within_file conflict, got:\n%+v", view.Conflicts)
	}
}

func TestT25_RegistryGracefulOnMalformed(t *testing.T) {
	t.Parallel()

	files, records, err := scanRegistryDir(context.Background(), "testdata/c4/registry_malformed")
	if err != nil {
		t.Fatalf("scanRegistryDir: %v", err)
	}
	if len(files) < 1 {
		t.Errorf("expected at least one good file to be parsed, got %d", len(files))
	}
	if len(records) == 0 {
		t.Errorf("expected records from the good file, got 0")
	}
}

func TestT26_RegistryCLIJSONOutput(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(context.Background(),
		"targ", "c4-registry", "--dir", "testdata/c4/registry_clean")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("targ c4-registry: %v", err)
	}
	var view RegistryView
	if err := json.Unmarshal(out, &view); err != nil {
		t.Fatalf("decode JSON output: %v\noutput: %s", err, out)
	}
	if view.SchemaVersion != "1" {
		t.Errorf("schema_version: want %q, got %q", "1", view.SchemaVersion)
	}
	if len(view.Elements) == 0 {
		t.Errorf("want elements, got 0")
	}
}

func TestT27_RegistryRegisteredInTarg(t *testing.T) {
	t.Parallel()

	out, err := exec.CommandContext(context.Background(), "targ").CombinedOutput()
	if err != nil {
		t.Fatalf("targ: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "c4-registry") {
		t.Errorf("targ list missing c4-registry\noutput: %s", out)
	}
}

func TestT28_RegistryLiveSet_ZeroConflicts(t *testing.T) {
	t.Parallel()

	files, records, err := scanRegistryDir(context.Background(), "../architecture/c4")
	if err != nil {
		t.Fatalf("scanRegistryDir: %v", err)
	}
	view := deriveRegistry("../architecture/c4", files, records)
	if len(view.Conflicts) != 0 {
		t.Errorf("want 0 conflicts on the post-reconciliation live set, got %d:\n%+v",
			len(view.Conflicts), view.Conflicts)
	}
}
