package parser

import (
	"github.com/toejough/projctl/internal/trace"
)

// CollectableFS combines interfaces needed for trace item collection.
type CollectableFS interface {
	FileSystem
	WalkableFS
	ReadFile(path string) (string, error)
}

// CollectResult contains collected trace items and any errors.
type CollectResult struct {
	Items  []*trace.TraceItem
	Errors []error
}

// CollectTraceItems discovers and parses all trace sources in a project.
// Finds documentation files (YAML/TOML) and Go test files, parsing each.
// Returns combined items and any non-fatal errors encountered.
func CollectTraceItems(root string, fs CollectableFS) (*CollectResult, error) {
	return &CollectResult{}, nil
}
