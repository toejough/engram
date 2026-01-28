package parser

import (
	"fmt"

	"github.com/toejough/projctl/internal/trace"
)

// DocResult contains parsed document items and metadata.
type DocResult struct {
	Items      []*trace.TraceItem
	Format     string // "yaml" or "toml"
	Deprecated bool   // True if TOML format used
}

// ParseDoc parses a document, detecting format and delegating to appropriate parser.
func ParseDoc(content string) (*DocResult, error) {
	return nil, fmt.Errorf("not implemented")
}
