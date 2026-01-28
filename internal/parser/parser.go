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
	format, err := DetectFormat(content)
	if err != nil {
		return nil, fmt.Errorf("format detection failed: %w", err)
	}

	switch format {
	case "yaml":
		results, errs := ParseDocument(content)
		if len(errs) > 0 {
			return nil, errs[0]
		}

		items := make([]*trace.TraceItem, len(results))
		for i, r := range results {
			items[i] = r.Item
		}

		return &DocResult{
			Items:      items,
			Format:     "yaml",
			Deprecated: false,
		}, nil

	case "toml":
		result, err := ParseTOML(content)
		if err != nil {
			return nil, err
		}

		return &DocResult{
			Items:      result.Items,
			Format:     "toml",
			Deprecated: result.Deprecated,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
