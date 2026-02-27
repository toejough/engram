package parser

import (
	"path/filepath"

	"github.com/toejough/projctl/internal/trace"
)

// CollectResult contains collected trace items and any errors.
type CollectResult struct {
	Items  []*trace.TraceItem
	Errors []error
}

// CollectableFS combines interfaces needed for trace item collection.
type CollectableFS interface {
	FileSystem
	WalkableFS
	ReadFile(path string) (string, error)
}

// CollectTraceItems discovers and parses all trace sources in a project.
// Finds documentation files (YAML/TOML) and Go test files, parsing each.
// Returns combined items and any non-fatal errors encountered.
func CollectTraceItems(root string, fs CollectableFS) (*CollectResult, error) {
	result := &CollectResult{}

	var projectName string

	// 1. Discover and parse documentation files
	discovery := DiscoverDocs(root, fs)
	for _, path := range discovery.Paths {
		content, err := fs.ReadFile(path)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		docResult, err := ParseDoc(content)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		result.Items = append(result.Items, docResult.Items...)

		// Extract project name from first item found
		if projectName == "" && len(docResult.Items) > 0 {
			projectName = docResult.Items[0].Project
		}
	}

	// 2. Discover and parse test files
	testPaths := DiscoverTestFiles(root, fs)
	for _, path := range testPaths {
		content, err := fs.ReadFile(path)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		// Use project name from docs, or directory name as fallback
		testProject := projectName
		if testProject == "" {
			testProject = filepath.Base(root)
		}

		testResult, err := ParseGoTestFile(path, content, testProject)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		result.Items = append(result.Items, testResult.Items...)
	}

	return result, nil
}
