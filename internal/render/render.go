// Package render provides system reminder formatting for engram memories.
package render

import (
	"fmt"
	"strings"

	"engram/internal/memory"
)

// Renderer formats classified memories as system reminder strings.
type Renderer struct{}

// New creates a new Renderer.
func New() *Renderer {
	return &Renderer{}
}

// Render formats a classified memory as a system reminder string (DES-1 format with tier).
func (r *Renderer) Render(mem *memory.ClassifiedMemory, filePath string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "<system-reminder source=\"engram\">\n")
	fmt.Fprintf(&sb, "[engram] Memory captured (tier %s).\n", mem.Tier)
	fmt.Fprintf(&sb, "  Created: %q\n", mem.Title)
	fmt.Fprintf(&sb, "  Type: %s\n", mem.ObservationType)
	fmt.Fprintf(&sb, "  File: %s\n", filePath)
	fmt.Fprintf(&sb, "</system-reminder>\n")

	return sb.String()
}
