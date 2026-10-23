//go:build targ

package dev

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"slices"
	"strings"
)

// unexported variables.
var (
	// reasonReference matches what a reason must name: a decision row (`861d27`) or an issue (`#831`).
	reasonReference = regexp.MustCompile(`\b[0-9]{3}d[0-9]+\b|#[0-9]+`)
	// scenarioLine matches the first line of a Gherkin scenario of any root: `Scenario: <ID> <title>` or
	// `Scenario Outline: <ID> <title>`, the ID being three digits, a letter, then letters and digits (`792b28c1a4`).
	scenarioLine = regexp.MustCompile(`(?m)^[ \t]*Scenario(?: Outline)?: ([0-9]{3}[a-z][0-9a-z]*) `)
)

// allowlistEntry is a scenario ruled out of the gate: the ID and the reason, which names its decision row or issue.
type allowlistEntry struct {
	id     string
	reason string
}

// declaredFunctions returns the names of the functions the Go sources declare, methods left out. Text that only reads
// as a declaration (in a comment or a string literal) is no declaration; a source that does not parse declares none.
func declaredFunctions(sources []string) []string {
	var names []string

	fileSet := token.NewFileSet()

	for _, source := range sources {
		file, err := parser.ParseFile(fileSet, "", source, parser.SkipObjectResolution)
		if err != nil {
			continue
		}

		for _, declaration := range file.Decls {
			function, isFunction := declaration.(*ast.FuncDecl)
			if isFunction && function.Recv == nil {
				names = append(names, function.Name.Name)
			}
		}
	}

	return names
}

// malformedAllowlistEntries returns the IDs of the entries whose reason names neither a decision row nor an issue, and
// of the entries that do not sort after the entry before them (out of order or listed twice).
func malformedAllowlistEntries(allowlist []allowlistEntry) []string {
	var malformed []string

	for index, entry := range allowlist {
		unsorted := index > 0 && entry.id <= allowlist[index-1].id

		if unsorted || !reasonReference.MatchString(entry.reason) {
			malformed = append(malformed, entry.id)
		}
	}

	return malformed
}

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

// scenariosNeedingATest returns the IDs, in their order, that no test source declares a test for and the allowlist does
// not rule out.
func scenariosNeedingATest(ids, testSources []string, allowlist []allowlistEntry) []string {
	var needing []string

	for _, id := range scenariosWithoutTest(ids, testSources) {
		if !slices.ContainsFunc(allowlist, func(entry allowlistEntry) bool { return entry.id == id }) {
			needing = append(needing, id)
		}
	}

	return needing
}

// scenariosWithoutTest returns the IDs, in their order, that no test source declares a test for: a function declared
// with the name `Test<ID>_...`, as the tests of a scenario are named.
func scenariosWithoutTest(ids, testSources []string) []string {
	var missing []string

	declared := declaredFunctions(testSources)

	for _, id := range ids {
		prefix := "Test" + id + "_"

		if !slices.ContainsFunc(declared, func(name string) bool { return strings.HasPrefix(name, prefix) }) {
			missing = append(missing, id)
		}
	}

	return missing
}

// staleAllowlistEntries returns the IDs of the entries, in their order, that rule out a scenario no spec holds or one
// a test now covers.
func staleAllowlistEntries(ids, testSources []string, allowlist []allowlistEntry) []string {
	var stale []string

	withoutTest := scenariosWithoutTest(ids, testSources)

	for _, entry := range allowlist {
		if !slices.Contains(ids, entry.id) || !slices.Contains(withoutTest, entry.id) {
			stale = append(stale, entry.id)
		}
	}

	return stale
}
