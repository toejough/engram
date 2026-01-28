package parser

import (
	"fmt"
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
		comment := extractTraceComment(fn.Doc)
		funcs = append(funcs, TestFunction{
			Name:    name,
			File:    filename,
			Line:    pos.Line,
			Comment: comment,
		})
	}

	return funcs, nil
}

// TraceCommentResult contains parsed trace comment data.
type TraceCommentResult struct {
	TestID  string   // The TEST-NNN ID
	Targets []string // Target IDs (TASK-NNN, REQ-NNN, etc.)
}

// ParseTraceComment parses a trace comment string into structured data.
// Expected format: "// TEST-NNN traces: TARGET1, TARGET2"
func ParseTraceComment(comment string) (*TraceCommentResult, error) {
	return nil, fmt.Errorf("not implemented")
}

// extractTraceComment extracts the trace comment line from a comment group.
// Looks for a line containing "TEST-NNN traces:" pattern.
// Returns empty string if no trace comment found.
func extractTraceComment(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}

	for _, comment := range doc.List {
		text := comment.Text
		// Look for trace comment pattern
		if strings.Contains(text, "traces:") || strings.Contains(text, "Traces:") {
			return text
		}
	}

	return ""
}
