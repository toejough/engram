// Package toolgate provides frecency-based gating for tool calls,
// suppressing repetitive suggestions that have low impact.
package toolgate

import "strings"

// CommandKey extracts a stable identity from a bash command string.
// Strips leading VAR=val env assignments, takes first two tokens,
// drops the second token if it starts with "-" (flags aren't identity).
func CommandKey(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}

	// Strip leading env var assignments (contain '=' but not as first char).
	for len(fields) > 0 && strings.Contains(fields[0], "=") && !strings.HasPrefix(fields[0], "=") {
		fields = fields[1:]
	}

	if len(fields) == 0 {
		return ""
	}

	if len(fields) == 1 {
		return fields[0]
	}

	// Drop second token if it's a flag.
	if strings.HasPrefix(fields[1], "-") {
		return fields[0]
	}

	return fields[0] + " " + fields[1]
}
