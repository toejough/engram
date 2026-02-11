//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// TestRefactorRenameCommand tests the CLI interface for projctl refactor rename
func TestRefactorRenameCommand(t *testing.T) {
	g := NewWithT(t)

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a simple Go file with a symbol
	testFile := filepath.Join(tempDir, "example.go")
	err := os.WriteFile(testFile, []byte(`package example

type OldName struct {
	Field string
}

func NewOldName() *OldName {
	return &OldName{}
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod for LSP
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	_ = modCmd.Run()

	// Run the refactor rename command
	cmd := exec.Command("projctl", "refactor", "rename",
		"--dir", tempDir,
		"--symbol", "OldName",
		"--to", "NewName")
	output, err := cmd.CombinedOutput()

	// Command should succeed
	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(MatchRegexp(`Renamed OldName in \d+ files`))

	// Verify file was updated
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("type NewName struct"))
}

// TestRefactorRenameSuccess tests successful symbol rename
func TestRefactorRenameSuccess(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Create a Go file with a type to rename
	file1 := filepath.Join(tempDir, "file1.go")
	err := os.WriteFile(file1, []byte(`package example

type Widget struct {
	Name string
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Create another file that references the type
	file2 := filepath.Join(tempDir, "file2.go")
	err = os.WriteFile(file2, []byte(`package example

func NewWidget() *Widget {
	return &Widget{Name: "default"}
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod for LSP to work
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Run refactor rename on the TYPE (not function)
	cmd := exec.Command("projctl", "refactor", "rename",
		"--dir", tempDir,
		"--symbol", "Widget",
		"--to", "Gadget")
	output, err := cmd.CombinedOutput()

	// Test will fail until implementation
	g.Expect(err).ToNot(HaveOccurred(), "Rename should succeed: %s", string(output))
	g.Expect(string(output)).To(MatchRegexp(`Renamed Widget in \d+ files`), "Should report renamed symbol and file count")

	// Verify type was renamed in file1
	content1, err := os.ReadFile(file1)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content1)).To(ContainSubstring("type Gadget struct"))
	g.Expect(string(content1)).ToNot(ContainSubstring("type Widget"))

	// Verify type references were updated in file2
	content2, err := os.ReadFile(file2)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content2)).To(ContainSubstring("*Gadget"))
	g.Expect(string(content2)).To(ContainSubstring("&Gadget{"))
	// Note: Function name NewWidget stays the same - gopls only renames the type itself
}

// TestRefactorRenameSymbolNotFound tests error handling when symbol doesn't exist
func TestRefactorRenameSymbolNotFound(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Create a Go file without the target symbol
	testFile := filepath.Join(tempDir, "example.go")
	err := os.WriteFile(testFile, []byte(`package example

type Something struct {
	Field string
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Try to rename non-existent symbol
	cmd := exec.Command("projctl", "refactor", "rename",
		"--dir", tempDir,
		"--symbol", "NonExistent",
		"--to", "NewName")
	output, err := cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred(), "Should fail with exit code 1")
	g.Expect(string(output)).To(ContainSubstring("symbol not found"))

	// Verify original file unchanged
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("type Something struct"))
}

// TestRefactorRenameConflict tests error handling when rename would cause a conflict
func TestRefactorRenameConflict(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Create a Go file with two symbols where rename would conflict
	testFile := filepath.Join(tempDir, "example.go")
	err := os.WriteFile(testFile, []byte(`package example

type OldName struct {
	Field string
}

type NewName struct {
	Other int
}
`), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Try to rename to a name that already exists
	cmd := exec.Command("projctl", "refactor", "rename",
		"--dir", tempDir,
		"--symbol", "OldName",
		"--to", "NewName")
	output, err := cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred(), "Should fail with exit code 1 on conflict")
	g.Expect(string(output)).To(ContainSubstring("conflict"))

	// Verify original file unchanged (atomic operation)
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("type OldName struct"))
	g.Expect(string(content)).To(ContainSubstring("type NewName struct"))
}

// TestRefactorRenameAtomic tests that rename is atomic (no partial changes on failure)
func TestRefactorRenameAtomic(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Create files
	file1 := filepath.Join(tempDir, "file1.go")
	originalContent1 := `package example

type Symbol struct {
	Field string
}
`
	err := os.WriteFile(file1, []byte(originalContent1), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	file2 := filepath.Join(tempDir, "file2.go")
	originalContent2 := `package example

func UseSymbol() *Symbol {
	return &Symbol{}
}
`
	err = os.WriteFile(file2, []byte(originalContent2), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Create a conflicting symbol
	file3 := filepath.Join(tempDir, "file3.go")
	originalContent3 := `package example

type NewSymbol struct {
	Other int
}
`
	err = os.WriteFile(file3, []byte(originalContent3), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize go.mod
	modCmd := exec.Command("go", "mod", "init", "example")
	modCmd.Dir = tempDir
	err = modCmd.Run()
	g.Expect(err).ToNot(HaveOccurred())

	// Try to rename to conflicting name (should fail)
	cmd := exec.Command("projctl", "refactor", "rename",
		"--dir", tempDir,
		"--symbol", "Symbol",
		"--to", "NewSymbol")
	_, err = cmd.CombinedOutput()
	g.Expect(err).To(HaveOccurred(), "Rename should fail due to conflict")

	// Verify NO files were changed (atomic operation)
	content1, err := os.ReadFile(file1)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content1)).To(Equal(originalContent1), "file1 should be unchanged")

	content2, err := os.ReadFile(file2)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content2)).To(Equal(originalContent2), "file2 should be unchanged")

	content3, err := os.ReadFile(file3)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content3)).To(Equal(originalContent3), "file3 should be unchanged")
}

// TestRefactorRenamePropertyBasedValidIdentifiers tests property: valid Go identifiers
func TestRefactorRenamePropertyBasedValidIdentifiers(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		// Generate valid Go identifier names
		identifier := rapid.StringMatching(`^[A-Z][a-zA-Z0-9_]*$`).
			Filter(func(s string) bool {
				// Filter out Go keywords
				keywords := map[string]bool{
					"break": true, "case": true, "chan": true, "const": true,
					"continue": true, "default": true, "defer": true, "else": true,
					"fallthrough": true, "for": true, "func": true, "go": true,
					"goto": true, "if": true, "import": true, "interface": true,
					"map": true, "package": true, "range": true, "return": true,
					"select": true, "struct": true, "switch": true, "type": true,
					"var": true,
				}
				return !keywords[s] && len(s) > 0 && len(s) < 50
			}).Draw(rt, "identifier")

		newIdentifier := rapid.StringMatching(`^[A-Z][a-zA-Z0-9_]*$`).
			Filter(func(s string) bool {
				keywords := map[string]bool{
					"break": true, "case": true, "chan": true, "const": true,
					"continue": true, "default": true, "defer": true, "else": true,
					"fallthrough": true, "for": true, "func": true, "go": true,
					"goto": true, "if": true, "import": true, "interface": true,
					"map": true, "package": true, "range": true, "return": true,
					"select": true, "struct": true, "switch": true, "type": true,
					"var": true,
				}
				return !keywords[s] && len(s) > 0 && len(s) < 50 && s != identifier
			}).Draw(rt, "newIdentifier")

		tempDir := t.TempDir()

		// Create a test file with the generated identifier
		testFile := filepath.Join(tempDir, "test.go")
		content := "package example\n\ntype " + identifier + " struct {\n\tField string\n}\n"
		err := os.WriteFile(testFile, []byte(content), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Initialize go.mod
		modCmd := exec.Command("go", "mod", "init", "example")
		modCmd.Dir = tempDir
		err = modCmd.Run()
		g.Expect(err).ToNot(HaveOccurred())

		// Rename should work for any valid identifier pair
		cmd := exec.Command("projctl", "refactor", "rename",
			"--dir", tempDir,
			"--symbol", identifier,
			"--to", newIdentifier)
		_, err = cmd.CombinedOutput()

		// Property: rename of valid identifiers should succeed
		g.Expect(err).ToNot(HaveOccurred(), "Valid identifier rename should succeed")

		// Verify the rename happened
		resultContent, err := os.ReadFile(testFile)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(resultContent)).To(ContainSubstring("type " + newIdentifier + " struct"))
		g.Expect(string(resultContent)).ToNot(ContainSubstring("type " + identifier + " struct"))
	})
}
