package memory_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// TestYieldReferencesRemoved verifies that yield references have been cleaned up
// from the codebase. This test is part of ISSUE-88: Clean up remaining yield references.
//
// This test should FAIL before cleanup (RED) and PASS after cleanup (GREEN).
func TestYieldReferencesRemoved(t *testing.T) {
	repoRoot := findRepoRoot(t)

	t.Run("no root-level yield.toml files exist", func(t *testing.T) {
		g := NewWithT(t)

		yieldTOMLPath := filepath.Join(repoRoot, "yield.toml")
		_, err := os.Stat(yieldTOMLPath)
		g.Expect(os.IsNotExist(err)).To(BeTrue(),
			"yield.toml should not exist at repo root (TASK-11)")

		claudeYieldPath := filepath.Join(repoRoot, ".claude", "yield.toml")
		_, err = os.Stat(claudeYieldPath)
		g.Expect(os.IsNotExist(err)).To(BeTrue(),
			".claude/yield.toml should not exist (TASK-11)")
	})

	t.Run("no yield_path references in active config files", func(t *testing.T) {
		g := NewWithT(t)

		// Check skills directory for yield_path in context.toml files
		skillsDir := filepath.Join(repoRoot, ".claude", "skills")
		if _, err := os.Stat(skillsDir); err == nil {
			err := filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() || !strings.HasSuffix(info.Name(), ".toml") {
					return nil
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				// Check for yield_path or producer_yield_path
				if strings.Contains(string(content), "yield_path") ||
					strings.Contains(string(content), "producer_yield_path") {
					t.Errorf("Found yield_path reference in %s (TASK-12)", path)
				}

				return nil
			})
			g.Expect(err).ToNot(HaveOccurred())
		}
	})
}

// findRepoRoot walks up the directory tree to find the repository root
// (the directory containing go.mod)
func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root (no go.mod found)")
		}
		dir = parent
	}
}
