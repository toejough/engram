package coverage_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/config"
	"github.com/toejough/projctl/internal/coverage"
)

// Mock file system for coverage tests
type mockFS struct {
	files map[string]string // path -> content
	dirs  map[string]bool   // directories that exist
}

func (m *mockFS) ReadFile(path string) (string, error) {
	content, exists := m.files[path]
	if !exists {
		return "", &fileNotFoundError{path: path}
	}
	return content, nil
}

func (m *mockFS) FileExists(path string) bool {
	_, exists := m.files[path]
	return exists
}

func (m *mockFS) DirExists(path string) bool {
	return m.dirs[path]
}

func (m *mockFS) Walk(root string, fn func(path string, isDir bool) error) error {
	// Walk directories first, then files
	for dir := range m.dirs {
		if err := fn(dir, true); err != nil {
			continue // skip directory
		}
	}
	for path := range m.files {
		_ = fn(path, false)
	}
	return nil
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

// TEST-193 traces: TASK-003
// Test empty repo returns 0% coverage and migrate recommendation
func TestAnalyze_EmptyRepo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockFS{
		files: map[string]string{},
		dirs:  map[string]bool{"/project": true},
	}

	cfg := config.Default()
	result, err := coverage.Analyze("/project", cfg, fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DocumentedCount).To(Equal(0))
	g.Expect(result.InferredCount).To(Equal(0))
	g.Expect(result.CoverageRatio).To(Equal(0.0))
	g.Expect(result.Recommendation).To(Equal("migrate"))
}

// TEST-194 traces: TASK-003
// Test fully documented repo returns preserve recommendation
func TestAnalyze_FullyDocumented(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockFS{
		files: map[string]string{
			"/project/requirements.md": `
# Requirements

## REQ-001: User Authentication
Users must be able to log in.

## REQ-002: User Registration
Users must be able to register.
`,
			"/project/design.md": `
# Design

## DES-001: Login Flow
Traces: REQ-001
`,
			"/project/architecture.md": `
# Architecture

## ARCH-001: Auth Service
Traces: DES-001
`,
			"/project/tasks.md": `
# Tasks

## TASK-001: Implement login
Traces: ARCH-001
`,
		},
		dirs: map[string]bool{
			"/project": true,
		},
	}

	cfg := config.Default()
	result, err := coverage.Analyze("/project", cfg, fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DocumentedCount).To(BeNumerically(">", 0))
	g.Expect(result.CoverageRatio).To(BeNumerically(">=", 0.6))
	g.Expect(result.Recommendation).To(Equal("preserve"))
}

// TEST-195 traces: TASK-003
// Test partial coverage returns evaluate recommendation
func TestAnalyze_PartialCoverage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockFS{
		files: map[string]string{
			"/project/requirements.md": `
# Requirements

## REQ-001: Feature One
`,
			// Go files with public interfaces (inferred items)
			"/project/api.go": `
package project

// PublicFunc is exported
func PublicFunc() {}

// AnotherExport is also exported
type AnotherExport struct{}
`,
			"/project/internal/impl.go": `
package internal

// privateFunc is not exported
func privateFunc() {}
`,
		},
		dirs: map[string]bool{
			"/project":          true,
			"/project/internal": true,
		},
	}

	cfg := config.Default()
	result, err := coverage.Analyze("/project", cfg, fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DocumentedCount).To(Equal(1))            // REQ-001
	g.Expect(result.InferredCount).To(BeNumerically(">", 0)) // Public exports
	g.Expect(result.Recommendation).To(BeElementOf("migrate", "evaluate"))
}

// TEST-196 traces: TASK-003
// Test custom thresholds from config are respected
func TestAnalyze_CustomThresholds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockFS{
		files: map[string]string{
			"/project/docs/requirements.md": `
## REQ-001: Only requirement
`,
			"/project/api.go": `
package project

func Export1() {}
func Export2() {}
`,
		},
		dirs: map[string]bool{
			"/project":      true,
			"/project/docs": true,
		},
	}

	// Custom config with lower preserve threshold
	cfg := &config.ProjectConfig{
		Paths: config.PathsConfig{
			DocsDir:      "docs",
			Requirements: "requirements.md",
		},
		Heuristics: config.HeuristicsConfig{
			PreserveThreshold: 0.30, // Very low threshold
			MigrateThreshold:  0.10,
		},
	}

	result, err := coverage.Analyze("/project", cfg, fs)

	g.Expect(err).ToNot(HaveOccurred())
	// With low thresholds, even partial docs should be "preserve"
	g.Expect(result.Recommendation).To(Equal("preserve"))
}

// TEST-197 traces: TASK-003
// Test coverage counts test files with traceability comments
func TestAnalyze_TestsWithTraceability(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockFS{
		files: map[string]string{
			"/project/requirements.md": `
## REQ-001: Feature
`,
			"/project/tasks.md": `
## TASK-001: Implement feature
`,
			"/project/feature_test.go": `
package project_test

// TEST-001 traces: TASK-001
func TestFeature(t *testing.T) {}
`,
		},
		dirs: map[string]bool{
			"/project": true,
		},
	}

	cfg := config.Default()
	result, err := coverage.Analyze("/project", cfg, fs)

	g.Expect(err).ToNot(HaveOccurred())
	// Should count documented items including TEST-001
	g.Expect(result.DocumentedCount).To(BeNumerically(">=", 3)) // REQ-001, TASK-001, TEST-001
}

// TEST-198 traces: TASK-003
// Test low coverage returns migrate recommendation
func TestAnalyze_LowCoverage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockFS{
		files: map[string]string{
			// No documentation at all
			"/project/api.go": `
package project

func PublicAPI() {}
type PublicType struct{}
func AnotherFunc() {}
type AnotherType interface{}
`,
			"/project/internal/impl.go": `
package internal

func Helper() {}
`,
		},
		dirs: map[string]bool{
			"/project":          true,
			"/project/internal": true,
		},
	}

	cfg := config.Default()
	result, err := coverage.Analyze("/project", cfg, fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DocumentedCount).To(Equal(0))
	g.Expect(result.InferredCount).To(BeNumerically(">", 0))
	g.Expect(result.CoverageRatio).To(Equal(0.0))
	g.Expect(result.Recommendation).To(Equal("migrate"))
}
