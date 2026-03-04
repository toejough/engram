// Package render provides system reminder formatting for engram memories.
package render

import (
	"fmt"
	"strings"

	"engram/internal/memory"
)

// Renderer formats enriched memories as system reminder strings.
type Renderer struct{}

// New creates a new Renderer.
func New() *Renderer {
	return &Renderer{}
}

// Render formats an enriched memory as a system reminder string.
// If degraded is true, the DES-2 format is used; otherwise DES-1.
func (r *Renderer) Render(mem *memory.Enriched, filePath string, degraded bool) string {
	var sb strings.Builder

	if degraded {
		fmt.Fprintf(&sb, "<system-reminder source=\"engram\">\n")
		fmt.Fprintf(&sb, "[engram] Memory captured (degraded \u2014 no API key).\n")
		fmt.Fprintf(&sb, "  Created: \"%s\"\n", mem.Title)
		fmt.Fprintf(&sb, "  File: %s\n", filePath)
		fmt.Fprintf(&sb, "  Note: Set ANTHROPIC_API_KEY for enriched memories.\n")
		fmt.Fprintf(&sb, "</system-reminder>\n")
	} else {
		fmt.Fprintf(&sb, "<system-reminder source=\"engram\">\n")
		fmt.Fprintf(&sb, "[engram] Memory captured.\n")
		fmt.Fprintf(&sb, "  Created: \"%s\"\n", mem.Title)
		fmt.Fprintf(&sb, "  Type: %s\n", mem.ObservationType)
		fmt.Fprintf(&sb, "  File: %s\n", filePath)
		fmt.Fprintf(&sb, "</system-reminder>\n")
	}

	return sb.String()
}
