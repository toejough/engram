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

// declaredFunctions returns the names of the functions the Go sources declare, methods left out. Text that only reads as
// a declaration (in a comment or a string literal) is no declaration, and a source that does not parse declares none.
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
