package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// TestFunction represents a detected test function.
type TestFunction struct {
	Name    string // Function name (e.g., "TestSomething")
	File    string // Source file path
	Line    int    // Line number of function declaration
	Comment string // Doc comment containing trace info (empty if none)
}

// ParseTestFunctions parses Go source code and returns test functions.
// Detects functions starting with "Test" or "Benchmark".
func ParseTestFunctions(filename, src string) ([]TestFunction, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var funcs []TestFunction

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		name := fn.Name.Name
		if !strings.HasPrefix(name, "Test") && !strings.HasPrefix(name, "Benchmark") {
			continue
		}

		pos := fset.Position(fn.Pos())
		funcs = append(funcs, TestFunction{
			Name: name,
			File: filename,
			Line: pos.Line,
		})
	}

	return funcs, nil
}
