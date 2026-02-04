// Package coverage analyzes documentation coverage for project adoption decisions.
package coverage

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"

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

// idPattern matches traceability IDs in documentation.
var idPattern = regexp.MustCompile(`(REQ|DES|ARCH|TASK|TEST)-\d+`)

// exportPattern matches exported Go symbols (functions, types, vars, consts).
var exportPattern = regexp.MustCompile(`(?m)^(?:func|type|var|const)\s+([A-Z][a-zA-Z0-9_]*)`)

// errSkipDir signals to skip a directory during walking.
var errSkipDir = errors.New("skip")

// Analyze performs coverage analysis on a project.
func Analyze(root string, cfg *config.ProjectConfig, fs CoverageFS) (*Result, error) {
	documented := countDocumentedItems(root, cfg, fs)
	inferred := countInferredItems(root, fs)

	result := &Result{
		DocumentedCount: documented,
		InferredCount:   inferred,
	}

	// Calculate coverage ratio
	total := documented + inferred
	if total > 0 {
		result.CoverageRatio = float64(documented) / float64(total)
	}

	// Determine recommendation based on thresholds
	result.Recommendation = getRecommendation(result.CoverageRatio, cfg)

	return result, nil
}

// countDocumentedItems counts traceability IDs in documentation files.
func countDocumentedItems(root string, cfg *config.ProjectConfig, fs CoverageFS) int {
	ids := make(map[string]bool)

	// Check docs directory
	docsDir := filepath.Join(root, cfg.Paths.DocsDir)
	docFiles := []string{
		cfg.Paths.Requirements,
		cfg.Paths.Design,
		cfg.Paths.Architecture,
		cfg.Paths.Tasks,
	}

	for _, name := range docFiles {
		path := filepath.Join(docsDir, name)
		content, err := fs.ReadFile(path)
		if err != nil {
			continue
		}
		matches := idPattern.FindAllString(content, -1)
		for _, m := range matches {
			ids[m] = true
		}
	}

	// Also scan test files for TEST-NNN patterns
	_ = fs.Walk(root, func(path string, isDir bool) error {
		if isDir {
			base := filepath.Base(path)
			if base == "vendor" || base == ".git" {
				return errSkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, "_test.go") {
			content, err := fs.ReadFile(path)
			if err != nil {
				return nil
			}
			matches := idPattern.FindAllString(content, -1)
			for _, m := range matches {
				ids[m] = true
			}
		}
		return nil
	})

	return len(ids)
}

// countInferredItems counts public exports in Go files (outside internal/).
func countInferredItems(root string, fs CoverageFS) int {
	exports := make(map[string]bool)

	_ = fs.Walk(root, func(path string, isDir bool) error {
		if isDir {
			base := filepath.Base(path)
			if base == "vendor" || base == ".git" || base == "internal" {
				return errSkipDir
			}
			return nil
		}

		// Only scan .go files that aren't tests
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			// Skip files in internal directories
			if strings.Contains(path, "/internal/") {
				return nil
			}

			content, err := fs.ReadFile(path)
			if err != nil {
				return nil
			}

			matches := exportPattern.FindAllStringSubmatch(content, -1)
			for _, m := range matches {
				if len(m) > 1 {
					exports[m[1]] = true
				}
			}
		}
		return nil
	})

	return len(exports)
}

// getRecommendation returns preserve/migrate/evaluate based on coverage and thresholds.
func getRecommendation(ratio float64, cfg *config.ProjectConfig) string {
	if ratio >= cfg.Heuristics.PreserveThreshold {
		return "preserve"
	}
	if ratio < cfg.Heuristics.MigrateThreshold {
		return "migrate"
	}
	return "evaluate"
}
