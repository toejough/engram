// Package surface formats surfaced memories into system reminders for hooks.
package surface

import (
	"fmt"
	"strings"

	"engram/internal/store"
)

// FormatSurfacing builds a system reminder string from surfaced memories.
// hookType determines the format: "pre-tool-use" uses compact single-line,
// all others use the full numbered list format (DES-1).
func FormatSurfacing(memories []store.ScoredMemory, hookType string) string {
	if len(memories) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(`<system-reminder source="engram">` + "\n")

	if hookType == "pre-tool-use" {
		m := memories[0]
		fmt.Fprintf(
			&b,
			"[engram] %s (%s, %s)\n",
			m.Memory.Title,
			m.Memory.Confidence,
			impactLabel(m.Memory),
		)
	} else {
		noun := "memories"
		if len(memories) == 1 {
			noun = "memory"
		}

		fmt.Fprintf(&b, "[engram] %d %s for this context:\n", len(memories), noun)

		for i, scored := range memories {
			fmt.Fprintf(
				&b,
				"\n%d. %s (%s, %s)\n",
				i+1,
				scored.Memory.Title,
				scored.Memory.Confidence,
				impactLabel(scored.Memory),
			)

			if scored.Memory.Content != "" {
				fmt.Fprintf(&b, "   %s\n", scored.Memory.Content)
			}
		}
	}

	b.WriteString("</system-reminder>")

	return b.String()
}

// unexported constants.
const (
	highImpactThreshold = 0.75
	medImpactThreshold  = 0.25
)

func impactLabel(m store.Memory) string {
	if m.SurfacingCount == 0 {
		return "new"
	}

	switch {
	case m.ImpactScore >= highImpactThreshold:
		return "high"
	case m.ImpactScore >= medImpactThreshold:
		return "medium"
	default:
		return "low"
	}
}
