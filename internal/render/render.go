// Package render provides system reminder formatting for engram memories.
package render

import (
	"fmt"
	"strings"

	"engram/internal/memory"
)

// Renderer formats stored memories as system reminder strings.
type Renderer struct{}

// New creates a new Renderer.
func New() *Renderer {
	return &Renderer{}
}

// RenderStored formats a stored memory as a system reminder string.
func (r *Renderer) RenderStored(mem *memory.Stored, filePath string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "<system-reminder source=\"engram\">\n")
	fmt.Fprintf(&sb, "[engram] Memory captured.\n")
	fmt.Fprintf(&sb, "  Situation: %s\n", mem.Situation)
	fmt.Fprintf(&sb, "  Action: %s\n", mem.Action)
	fmt.Fprintf(&sb, "  File: %s\n", filePath)
	fmt.Fprintf(&sb, "</system-reminder>\n")

	return sb.String()
}
