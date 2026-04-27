//go:build targ

package dev

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateL4DiagramIDs_AcceptsValidSpec(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l4RegistryFixture(t, dir)
	if err := validateL4DiagramIDs(context.Background(), validL4Spec(), dir); err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}
}

func TestValidateL4DiagramIDs_AggregatesAllViolations(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l4RegistryFixture(t, dir)
	spec := validL4Spec()
	spec.Diagram.Nodes = append(spec.Diagram.Nodes,
		L4Node{ID: "EXT1", Name: "fabricated", Kind: "external"})
	spec.Diagram.Edges[0].ID = "R2a"
	err := validateL4DiagramIDs(context.Background(), spec, dir)
	if err == nil {
		t.Fatalf("want error, got nil")
	}
	if !strings.Contains(err.Error(), "EXT1") || !strings.Contains(err.Error(), "R2a") {
		t.Errorf("want aggregated error naming both EXT1 and R2a; got %v", err)
	}
}

func TestValidateL4DiagramIDs_RejectsLetterSuffixedEdge(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l4RegistryFixture(t, dir)
	spec := validL4Spec()
	spec.Diagram.Edges[0].ID = "R2a"
	err := validateL4DiagramIDs(context.Background(), spec, dir)
	if err == nil || !strings.Contains(err.Error(), "R2a") {
		t.Fatalf("want error naming R2a; got %v", err)
	}
}

func TestValidateL4DiagramIDs_RejectsNodeNotInRegistry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l4RegistryFixture(t, dir)
	spec := validL4Spec()
	// E99 passes the E<n> shape check but is not in the staged registry
	// (E1..E5 only).
	spec.Diagram.Nodes = append(spec.Diagram.Nodes,
		L4Node{ID: "E99", Name: "fabricated", Kind: "component"})
	err := validateL4DiagramIDs(context.Background(), spec, dir)
	if err == nil || !strings.Contains(err.Error(), "E99") {
		t.Fatalf("want error naming unknown E99; got %v", err)
	}
}

func TestValidateL4DiagramIDs_RejectsNonEPrefixedNode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l4RegistryFixture(t, dir)
	spec := validL4Spec()
	spec.Diagram.Nodes = append(spec.Diagram.Nodes,
		L4Node{ID: "EXT1", Name: "fabricated external", Kind: "external"})
	err := validateL4DiagramIDs(context.Background(), spec, dir)
	if err == nil || !strings.Contains(err.Error(), "EXT1") {
		t.Fatalf("want error naming EXT1; got %v", err)
	}
}

func TestValidateL4DiagramIDs_RejectsWrongNamespaceEdge(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l4RegistryFixture(t, dir)
	spec := validL4Spec()
	spec.Diagram.Edges[0].ID = "X1"
	err := validateL4DiagramIDs(context.Background(), spec, dir)
	if err == nil || !strings.Contains(err.Error(), "X1") {
		t.Fatalf("want error naming X1; got %v", err)
	}
}

// l4RegistryFixture stages c1+c2 specs in dir so scanRegistryDir derives a
// registry containing E1..E5 (matching dev/testdata/c4/registry_clean/).
func l4RegistryFixture(t *testing.T, dir string) {
	t.Helper()
	srcs := []string{
		"testdata/c4/registry_clean/c1-thing.json",
		"testdata/c4/registry_clean/c2-thing-internal.json",
	}
	for _, src := range srcs {
		raw, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read fixture %s: %v", src, err)
		}
		dst := filepath.Join(dir, filepath.Base(src))
		if writeErr := os.WriteFile(dst, raw, 0o644); writeErr != nil {
			t.Fatalf("write fixture %s: %v", dst, writeErr)
		}
	}
}

// validL4Spec returns a spec that should pass validateL4DiagramIDs against the
// registry staged by l4RegistryFixture (E1..E5 known, E2 chosen as focus).
func validL4Spec() *L4Spec {
	return &L4Spec{
		SchemaVersion: "1",
		Level:         4,
		Name:          "thing",
		Parent:        "c2-thing-internal.md",
		Focus:         L4Focus{ID: "E2", Name: "Thing", L3Container: "c2-thing-internal.md"},
		ContextProse:  "fixture prose",
		Diagram: L4Diagram{
			Nodes: []L4Node{
				{ID: "E2", Name: "Thing", Kind: "focus"},
				{ID: "E4", Name: "Frontend", Kind: "component"},
				{ID: "E5", Name: "Backend", Kind: "component"},
			},
			Edges: []L4Edge{
				{ID: "R1", From: "E4", To: "E2", Label: "calls"},
				{ID: "D1", From: "E2", To: "E4", Label: "DI back-edge", Dotted: true},
			},
		},
	}
}
