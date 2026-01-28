package parser

import (
	"fmt"

	"github.com/toejough/projctl/internal/trace"
)

// TOMLResult contains parsed TOML items and deprecation flag.
type TOMLResult struct {
	Items      []*trace.TraceItem
	Deprecated bool // Always true for TOML format
}

// ParseTOML parses TOML-formatted content into TraceItems.
// Returns items with Deprecated=true to signal legacy format.
func ParseTOML(content string) (*TOMLResult, error) {
	return nil, fmt.Errorf("not implemented")
}
