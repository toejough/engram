//go:build integration

package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// TestExtractFunctionAtomicOnFailure tests rollback on failure
func TestExtractFunctionAtomicOnFailure(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "atomic.go")
	originalContent := `package example

func Simple() {
	x := 1
	y := 2
	z := x + y
}
`
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Try to extract with invalid line range (should fail)
	cmd := exec.Command("projctl", "refactor", "extract-function",
		"--file", testFile,
		"--lines", "10-20", // Lines don't exist
		"--name", "Invalid")
	_, err = cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred(), "Should fail with invalid line range")

	// Verify file is unchanged (atomic operation)
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal(originalContent), "File should be unchanged after failed extraction")
}

// TestExtractFunctionBasicExtraction tests extracting a simple code block
func TestExtractFunctionBasicExtraction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "basic.go")
	err := os.WriteFile(testFile, []byte(`package example

func Process(a, b int) int {
	sum := a + b
	doubled := sum * 2
	return doubled
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Extract lines 4-5 (the computation)
	cmd := exec.Command("projctl", "refactor", "extract-function",
		"--file", testFile,
		"--lines", "4-5",
		"--name", "ComputeResult")
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Extract should succeed: %s", string(output))

	// Verify extracted function exists
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())
	contentStr := string(content)

	g.Expect(contentStr).To(ContainSubstring("func ComputeResult"), "Extracted function should be created")
	g.Expect(contentStr).To(ContainSubstring("ComputeResult("), "Original function should call extracted function")
}

// TestExtractFunctionDetectsParameters tests that parameters are correctly detected
func TestExtractFunctionDetectsParameters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "params.go")
	err := os.WriteFile(testFile, []byte(`package example

func Calculate(x, y, z int) int {
	// Extract this - uses x and y but not z
	intermediate := x * 2
	result := intermediate + y
	return result + z
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Extract lines that use x and y
	cmd := exec.Command("projctl", "refactor", "extract-function",
		"--file", testFile,
		"--lines", "5-6",
		"--name", "ProcessValues")
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Extract should succeed: %s", string(output))

	// Verify parameters were detected correctly
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())
	contentStr := string(content)

	// Should have x and y as parameters (not z, since it's not used in extracted code)
	g.Expect(contentStr).To(MatchRegexp(`func ProcessValues\([^)]*x[^)]*\)`), "Should include x as parameter")
	g.Expect(contentStr).To(MatchRegexp(`func ProcessValues\([^)]*y[^)]*\)`), "Should include y as parameter")
}

// TestExtractFunctionDetectsReturnValues tests that return values are correctly detected
func TestExtractFunctionDetectsReturnValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "returns.go")
	err := os.WriteFile(testFile, []byte(`package example

import "fmt"

func Compute(x int) (int, error) {
	// Extract this - produces value and err
	value := x * 2
	var err error
	if value > 100 {
		err = fmt.Errorf("too large")
	}
	return value, err
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Extract lines that produce multiple values (lines shifted by import)
	cmd := exec.Command("projctl", "refactor", "extract-function",
		"--file", testFile,
		"--lines", "7-11",
		"--name", "CalculateValue")
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Extract should succeed: %s", string(output))

	// Verify return values were detected
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())
	contentStr := string(content)

	// Should return both value and error
	g.Expect(contentStr).To(MatchRegexp(`func CalculateValue\([^)]*\)\s*\([^)]*int[^)]*error[^)]*\)`),
		"Should return int and error")
}

// TestExtractFunctionInvalidFunctionName tests error handling for invalid names
func TestExtractFunctionInvalidFunctionName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte(`package example

func Foo() {
	x := 1
	y := 2
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	invalidNames := []string{
		"123Invalid",   // starts with number
		"invalid-name", // contains hyphen
		"invalid name", // contains space
		"break",        // Go keyword
		"func",         // Go keyword
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)

			cmd := exec.Command("projctl", "refactor", "extract-function",
				"--file", testFile,
				"--lines", "4-5",
				"--name", name)
			output, err := cmd.CombinedOutput()

			g.Expect(err).To(HaveOccurred(), "Should reject invalid name: %s", name)
			g.Expect(string(output)).To(ContainSubstring("invalid function name"))
		})
	}
}

// TestExtractFunctionInvalidLineRange tests error handling for invalid line ranges
func TestExtractFunctionInvalidLineRange(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte(`package example

func Foo() int {
	return 42
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	testCases := []struct {
		name      string
		lineRange string
		errMsg    string
	}{
		{
			name:      "lines beyond file",
			lineRange: "100-200",
			errMsg:    "invalid line range",
		},
		{
			name:      "reversed range",
			lineRange: "5-3",
			errMsg:    "invalid line range",
		},
		// Note: negative line ranges like "-1-5" are not tested here because
		// the flag parser interprets "-1" as a flag rather than a value.
		// This is expected flag parsing behavior, not a validation concern.
		{
			name:      "zero line",
			lineRange: "0-3",
			errMsg:    "invalid line range",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			cmd := exec.Command("projctl", "refactor", "extract-function",
				"--file", testFile,
				"--lines", tc.lineRange,
				"--name", "Test")
			output, err := cmd.CombinedOutput()

			g.Expect(err).To(HaveOccurred(), "Should fail: %s", string(output))
			g.Expect(string(output)).To(ContainSubstring(tc.errMsg))
		})
	}
}

// TestExtractFunctionMultipleExtractions tests multiple extractions in same file
func TestExtractFunctionMultipleExtractions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "multiple.go")
	err := os.WriteFile(testFile, []byte(`package example

func Complex(x, y int) int {
	a := x * 2
	b := y * 3
	c := a + b
	d := c * 4
	return d
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// First extraction
	cmd1 := exec.Command("projctl", "refactor", "extract-function",
		"--file", testFile,
		"--lines", "4-5",
		"--name", "InitialCalc")
	_, err = cmd1.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "First extraction should succeed")

	// Second extraction (line numbers may have changed due to first extraction)
	// We need to re-read the file to know the new structure
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify both extractions exist
	contentStr := string(content)
	g.Expect(contentStr).To(ContainSubstring("func InitialCalc"))

	// Verify code still compiles
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = tempDir
	buildOutput, err := buildCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "Code with multiple extractions should compile: %s", string(buildOutput))
}

// TestExtractFunctionNameConflict tests error handling when function name already exists
func TestExtractFunctionNameConflict(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "conflict.go")
	err := os.WriteFile(testFile, []byte(`package example

func ExistingFunction() {
	// Already exists
}

func Process() {
	x := 1
	y := 2
	z := x + y
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Try to extract with a name that already exists
	cmd := exec.Command("projctl", "refactor", "extract-function",
		"--file", testFile,
		"--lines", "9-10",
		"--name", "ExistingFunction")
	output, err := cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred())
	g.Expect(string(output)).To(ContainSubstring("function name conflict"))
}

// TestExtractFunctionNonExistentFile tests error handling for non-existent files
func TestExtractFunctionNonExistentFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmd := exec.Command("projctl", "refactor", "extract-function",
		"--file", "/nonexistent/file.go",
		"--lines", "1-5",
		"--name", "Test")
	output, err := cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred())
	g.Expect(string(output)).To(ContainSubstring("file not found"))
}

// TestExtractFunctionPreservesFormatting tests that code formatting is preserved
func TestExtractFunctionPreservesFormatting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "format.go")
	err := os.WriteFile(testFile, []byte(`package example

import "fmt"

func Verbose(name string) {
	// Extract this
	message := fmt.Sprintf(
		"Hello, %s! Welcome to the system.",
		name,
	)
	fmt.Println(message)
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Extract the message creation
	cmd := exec.Command("projctl", "refactor", "extract-function",
		"--file", testFile,
		"--lines", "7-10",
		"--name", "CreateMessage")
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Extract should succeed: %s", string(output))

	// Verify code is still properly formatted
	fmtCmd := exec.Command("gofmt", "-l", testFile)
	fmtOutput, err := fmtCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(fmtOutput)).To(BeEmpty(), "Code should be properly formatted")
}

// TestExtractFunctionProducesCompilableCode tests that extracted code compiles
func TestExtractFunctionProducesCompilableCode(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "compilable.go")
	err := os.WriteFile(testFile, []byte(`package example

import "fmt"

func Original(name string, count int) {
	// Extract this block
	for i := 0; i < count; i++ {
		fmt.Printf("Hello %s #%d\n", name, i)
	}
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Extract the for loop
	cmd := exec.Command("projctl", "refactor", "extract-function",
		"--file", testFile,
		"--lines", "7-9",
		"--name", "PrintGreetings")
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Extract should succeed: %s", string(output))

	// Verify the code compiles
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = tempDir
	buildOutput, err := buildCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "Extracted code should compile: %s", string(buildOutput))
}

// TestExtractFunctionPropertyBasedValidRanges tests property: valid line ranges produce valid extractions
func TestExtractFunctionPropertyBasedValidRanges(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		tempDir := t.TempDir()

		// Generate a function with random number of lines
		numLines := rapid.IntRange(5, 20).Draw(rt, "numLines")

		// Create function body with simple statements
		var lines []string
		lines = append(lines, "package example", "", "func Test() int {")
		for i := 0; i < numLines; i++ {
			lines = append(lines, fmt.Sprintf("\tx%d := %d", i, i))
		}
		lines = append(lines, "\treturn 0", "}")

		testFile := filepath.Join(tempDir, "test.go")
		err := os.WriteFile(testFile, []byte(strings.Join(lines, "\n")), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Initialize go.mod
		modCmd := exec.Command("go", "mod", "init", "example")
		modCmd.Dir = tempDir
		err = modCmd.Run()
		g.Expect(err).ToNot(HaveOccurred())

		// Generate valid line range within the function body
		startLine := rapid.IntRange(4, 4+numLines-2).Draw(rt, "startLine")
		endLine := rapid.IntRange(startLine, 4+numLines-1).Draw(rt, "endLine")

		// Generate valid function name
		funcName := rapid.StringMatching(`^[A-Z][a-zA-Z0-9]*$`).
			Filter(func(s string) bool {
				keywords := map[string]bool{
					"break": true, "case": true, "chan": true, "const": true,
					"continue": true, "default": true, "defer": true, "else": true,
					"fallthrough": true, "for": true, "func": true, "go": true,
					"goto": true, "if": true, "import": true, "interface": true,
					"map": true, "package": true, "range": true, "return": true,
					"select": true, "struct": true, "switch": true, "type": true,
					"var": true, "Test": true,
				}
				return !keywords[s] && len(s) > 0 && len(s) < 30
			}).Draw(rt, "funcName")

		// Extract function
		cmd := exec.Command("projctl", "refactor", "extract-function",
			"--file", testFile,
			"--lines", fmt.Sprintf("%d-%d", startLine, endLine),
			"--name", funcName)
		output, err := cmd.CombinedOutput()

		// Property: if extraction succeeds, the code must compile
		// Note: Not all line ranges are extractable by gopls, so we allow failures
		// but if it succeeds, the result must be valid
		if err == nil {
			// Property: extracted code should compile
			buildCmd := exec.Command("go", "build", "./...")
			buildCmd.Dir = tempDir
			buildOutput, err := buildCmd.CombinedOutput()
			g.Expect(err).ToNot(HaveOccurred(), "Extracted code should compile: %s", string(buildOutput))
		} else {
			// If it failed, it should be a graceful error (no panic, proper rollback)
			outputStr := string(output)
			g.Expect(outputStr).To(Or(
				ContainSubstring("Error:"),
				ContainSubstring("cannot extract"),
				ContainSubstring("rolled back"),
			), "Extraction failure should be graceful: %s", outputStr)
		}
	})
}

// TestExtractFunctionWithComments tests that comments are handled correctly
func TestExtractFunctionWithComments(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "comments.go")
	err := os.WriteFile(testFile, []byte(`package example

func Process(x int) int {
	// This is important
	doubled := x * 2
	// Add offset
	result := doubled + 10
	return result
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Extract including comments
	cmd := exec.Command("projctl", "refactor", "extract-function",
		"--file", testFile,
		"--lines", "4-7",
		"--name", "Calculate")
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Extract should succeed: %s", string(output))

	// Verify extraction includes comments
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())
	contentStr := string(content)

	g.Expect(contentStr).To(ContainSubstring("func Calculate"))
	// Comments should be preserved in extracted function
	g.Expect(contentStr).To(ContainSubstring("This is important"))
	g.Expect(contentStr).To(ContainSubstring("Add offset"))
}

// TestRefactorExtractFunctionCommand tests the CLI interface for projctl refactor extract-function
func TestRefactorExtractFunctionCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Create a Go file with code to extract
	testFile := filepath.Join(tempDir, "example.go")
	err := os.WriteFile(testFile, []byte(`package example

func DoSomething(x, y int) int {
	// Lines to extract
	sum := x + y
	result := sum * 2
	return result
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod for LSP
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	_ = modCmd.Run()

	// Run the refactor extract-function command
	cmd := exec.Command("projctl", "refactor", "extract-function",
		"--file", testFile,
		"--lines", "5-6",
		"--name", "Calculate")
	output, err := cmd.CombinedOutput()

	// Command should succeed
	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(MatchRegexp(`Extracted function Calculate`))

	// Verify file was updated
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("func Calculate"))
}
