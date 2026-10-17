//go:build targ

package dev

import (
	"regexp"
	"slices"
)

// unexported variables.
var (
	// scenarioLine matches the first line of a Gherkin scenario of change 793: `Scenario: <ID> <title>`, the ID
	// beginning with 793a for the product spec or 793d for the technical spec.
	scenarioLine = regexp.MustCompile(`(?m)^[ \t]*Scenario: (793[ad][0-9a-z]+) `)
)

// scenarioIDs returns the IDs of the scenarios a spec holds, each once, in the order they first appear.
func scenarioIDs(spec string) []string {
	var ids []string

	for _, match := range scenarioLine.FindAllStringSubmatch(spec, -1) {
		if !slices.Contains(ids, match[1]) {
			ids = append(ids, match[1])
		}
	}

	return ids
}
