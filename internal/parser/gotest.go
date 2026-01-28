package parser

// TestFunction represents a detected test function.
type TestFunction struct {
	Name string // Function name (e.g., "TestSomething")
	File string // Source file path
	Line int    // Line number of function declaration
}

// ParseTestFunctions parses Go source code and returns test functions.
// Detects functions starting with "Test" or "Benchmark".
func ParseTestFunctions(filename, src string) ([]TestFunction, error) {
	return nil, nil
}
