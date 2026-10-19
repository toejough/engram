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
	// removedHeading matches the heading of a Removed operation: `### <ID> Removed: <ID of the scenario> <title>`.
	removedHeading = regexp.MustCompile(`(?m)^#{2,4} [0-9a-z]+ Removed: (793[ad][0-9a-z]+) `)
	// scenarioLine matches the first line of a Gherkin scenario of change 793 or 855: `Scenario: <ID> <title>`, the ID
	// beginning with 793a or 855a for the product spec and 793d or 855d for the technical spec (855b45).
	scenarioLine = regexp.MustCompile(`(?m)^[ \t]*Scenario: ((?:793|855)[ad][0-9a-z]+) `)
)

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

// exceptIDs returns the IDs less those of the exempt ones, in their order.
func exceptIDs(ids, exempt []string) []string {
	return slices.DeleteFunc(slices.Clone(ids), func(id string) bool { return slices.Contains(exempt, id) })
}

// removedScenarioIDs returns the IDs of the scenarios of 793 that a Removed operation of a spec names, each in the
// heading `### <ID of the operation> Removed: <ID of the scenario> <title>` (855b45).
func removedScenarioIDs(spec string) []string {
	matches := removedHeading.FindAllStringSubmatch(spec, -1)
	ids := make([]string, 0, len(matches))

	for _, match := range matches {
		ids = append(ids, match[1])
	}

	return ids
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

// scenariosWithoutTest returns the IDs, in their order, that no test source declares a test for: a function declared
// with the name `Test<ID>_...`, as the tests of a scenario are named (793e2).
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
