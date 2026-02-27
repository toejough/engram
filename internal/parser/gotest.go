package parser

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/toejough/projctl/internal/trace"
)

// GoTestFileResult contains parsed test items and any warnings.
type GoTestFileResult struct {
	Items    []*trace.TraceItem
	Warnings []string
}

// TestFunction represents a detected test function.
type TestFunction struct {
	Name    string // Function name (e.g., "TestSomething")
	File    string // Source file path
	Line    int    // Line number of function declaration
	Comment string // Doc comment containing trace info (empty if none)
}

// TraceCommentResult contains parsed trace comment data.
type TraceCommentResult struct {
	TestID  string   // The TEST-NNN ID
	Targets []string // Target IDs (TASK-NNN, REQ-NNN, etc.)
}

// ParseGoTestFile parses a Go test file and extracts traced test items.
// Returns items for all functions with valid trace comments.
// Returns error if duplicate TEST IDs are found.
func ParseGoTestFile(filename, src, project string) (*GoTestFileResult, error) {
	funcs, err := ParseTestFunctions(filename, src)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	var (
		items    []*trace.TraceItem
		warnings []string
	)

	seenIDs := make(map[string]bool)

	for _, fn := range funcs {
		if fn.Comment == "" {
			continue
		}

		parsed, err := ParseTraceComment(fn.Comment)
		if err != nil {
			// Malformed comment - warn and continue
			warnings = append(warnings, fmt.Sprintf("%s:%d: %v", filename, fn.Line, err))
			continue
		}

		// Check for duplicate TEST ID
		if seenIDs[parsed.TestID] {
			return nil, fmt.Errorf("duplicate TEST ID %q in %s", parsed.TestID, filename)
		}

		seenIDs[parsed.TestID] = true

		item := &trace.TraceItem{
			ID:       parsed.TestID,
			Type:     trace.NodeTypeTEST,
			Project:  project,
			Title:    fn.Name,
			Status:   "active",
			TracesTo: parsed.Targets,
			Location: filename,
			Line:     fn.Line,
			Function: fn.Name,
		}

		items = append(items, item)
	}

	return &GoTestFileResult{
		Items:    items,
		Warnings: warnings,
	}, nil
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

// ParseTraceComment parses a trace comment string into structured data.
// Expected format: "// TEST-NNN traces: TARGET1, TARGET2"
func ParseTraceComment(comment string) (*TraceCommentResult, error) {
	if comment == "" {
		return nil, errors.New("empty comment")
	}

	matches := traceCommentPattern.FindStringSubmatch(comment)
	if matches == nil {
		return nil, fmt.Errorf("malformed trace comment: %q", comment)
	}

	testID := strings.ToUpper(matches[1])
	targetsStr := matches[2]

	// Split and clean targets
	rawTargets := strings.Split(targetsStr, ",")

	targets := make([]string, 0, len(rawTargets))
	for _, t := range rawTargets {
		t = strings.TrimSpace(t)
		if t != "" {
			targets = append(targets, strings.ToUpper(t))
		}
	}

	return &TraceCommentResult{
		TestID:  testID,
		Targets: targets,
	}, nil
}

// unexported variables.
var (
	traceCommentPattern = regexp.MustCompile(`(?i)^//\s*(TEST-\d{3,})\s+traces:\s*(.+)$`)
)

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
