// Package coverage analyzes documentation coverage for project adoption decisions.
package coverage

import (
	"github.com/toejough/projctl/internal/config"
)

// CoverageFS provides file system operations for coverage analysis.
type CoverageFS interface {
	ReadFile(path string) (string, error)
	FileExists(path string) bool
	DirExists(path string) bool
	Walk(root string, fn func(path string, isDir bool) error) error
}

// Result holds the coverage analysis results.
type Result struct {
	DocumentedCount int     // Count of documented items (REQ, DES, ARCH, TASK, TEST)
	InferredCount   int     // Count of inferred items (public interfaces, tests)
	CoverageRatio   float64 // documented / (documented + inferred)
	Recommendation  string  // "preserve", "migrate", or "evaluate"
}

// Analyze performs coverage analysis on a project.
func Analyze(root string, cfg *config.ProjectConfig, fs CoverageFS) (*Result, error) {
	// TODO: implement
	return &Result{}, nil
}
