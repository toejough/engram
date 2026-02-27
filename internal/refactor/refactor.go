// Package refactor provides automated refactoring operations using LSP.
package refactor

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Capabilities describes available refactoring capabilities.
type Capabilities struct {
	GoplsAvailable bool   `json:"gopls_available"         toml:"gopls_available"`
	GoplsVersion   string `json:"gopls_version,omitempty" toml:"gopls_version,omitempty"`
	RenameSupport  bool   `json:"rename_support"          toml:"rename_support"`
}

// ExtractOpts holds options for function extraction.
type ExtractOpts struct {
	File      string
	StartLine int
	EndLine   int
	Name      string
}

// ExtractResult contains the result of a function extraction.
type ExtractResult struct {
	ExtractedFunction string
	FilesChanged      int
}

// RenameOpts holds options for symbol renaming.
type RenameOpts struct {
	Dir    string
	Symbol string
	To     string
}

// RenameResult contains the result of a rename operation.
type RenameResult struct {
	FilesChanged int
}

// CheckCapabilities returns the available refactoring capabilities.
func CheckCapabilities() Capabilities {
	caps := Capabilities{}

	// Check for gopls
	goplsPath, err := exec.LookPath("gopls")
	if err == nil {
		caps.GoplsAvailable = true
		caps.RenameSupport = true

		// Get version
		cmd := exec.Command(goplsPath, "version")

		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 0 {
				caps.GoplsVersion = strings.TrimSpace(lines[0])
			}
		}
	}

	return caps
}

// ExtractFunction extracts a range of lines into a new function using gopls.
func ExtractFunction(opts ExtractOpts) (*ExtractResult, error) {
	// Verify gopls is available
	if _, err := exec.LookPath("gopls"); err != nil {
		return nil, fmt.Errorf("gopls not found in PATH: %w", err)
	}

	// Validate file exists
	if _, err := os.Stat(opts.File); err != nil {
		return nil, fmt.Errorf("file not found: %s", opts.File)
	}

	// Validate line range
	content, err := os.ReadFile(opts.File)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	if opts.StartLine < 1 || opts.EndLine < opts.StartLine || opts.EndLine > len(lines) {
		return nil, fmt.Errorf("invalid line range: %d-%d (file has %d lines)", opts.StartLine, opts.EndLine, len(lines))
	}

	// Validate function name
	if opts.Name == "" {
		return nil, errors.New("function name cannot be empty")
	}

	if !isValidIdentifier(opts.Name) {
		return nil, fmt.Errorf("invalid function name: %s", opts.Name)
	}

	// Check for name conflicts
	if strings.Contains(string(content), "func "+opts.Name+"(") {
		return nil, fmt.Errorf("function name conflict: %s already exists", opts.Name)
	}

	// Store original content for rollback
	originalContent := content

	// Execute gopls codeaction for extract function
	// Format: file:startline:startcol-endline:endcol
	// We use column 1 for start and end of line for the range
	position := fmt.Sprintf("%s:%d:1-%d:%d", opts.File, opts.StartLine, opts.EndLine, len(lines[opts.EndLine-1])+1)

	cmd := exec.Command("gopls", "codeaction",
		"-kind=refactor.extract.function",
		"-exec",
		"-write",
		position)

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Rollback
		_ = os.WriteFile(opts.File, originalContent, 0644)

		errMsg := stderr.String()
		if strings.Contains(errMsg, "no code action") {
			return nil, errors.New("cannot extract function from selected lines: no extractable code")
		}

		return nil, fmt.Errorf("gopls extract failed: %s", errMsg)
	}

	// Read the modified content
	newContent, err := os.ReadFile(opts.File)
	if err != nil {
		_ = os.WriteFile(opts.File, originalContent, 0644)
		return nil, fmt.Errorf("failed to read modified file: %w", err)
	}

	// Gopls creates a function with a generated name - we need to rename it
	// Look for the new function and rename it to the desired name
	newContentStr := string(newContent)

	extractedFuncPattern := "func newFunction("
	if strings.Contains(newContentStr, extractedFuncPattern) {
		newContentStr = strings.Replace(newContentStr, extractedFuncPattern, "func "+opts.Name+"(", 1)
		// Also update the call site
		newContentStr = strings.ReplaceAll(newContentStr, "newFunction(", opts.Name+"(")

		err := os.WriteFile(opts.File, []byte(newContentStr), 0644)
		if err != nil {
			_ = os.WriteFile(opts.File, originalContent, 0644)
			return nil, fmt.Errorf("failed to rename extracted function: %w", err)
		}
	}

	// Verify the code compiles
	dir := filepath.Dir(opts.File)
	buildCmd := exec.Command("go", "build", "./...")

	buildCmd.Dir = dir
	if err := buildCmd.Run(); err != nil {
		// Rollback
		_ = os.WriteFile(opts.File, originalContent, 0644)
		return nil, fmt.Errorf("extracted code does not compile, rolled back: %w", err)
	}

	return &ExtractResult{
		ExtractedFunction: opts.Name,
		FilesChanged:      1,
	}, nil
}

// GoplsInstallInstructions returns installation instructions for gopls.
func GoplsInstallInstructions() string {
	return "gopls not found. Install with: go install golang.org/x/tools/gopls@latest"
}

// Rename performs an LSP-based symbol rename using gopls.
func Rename(opts RenameOpts) (*RenameResult, error) {
	// Verify gopls is available
	if _, err := exec.LookPath("gopls"); err != nil {
		return nil, fmt.Errorf("gopls not found in PATH: %w", err)
	}

	// Find the first occurrence of the symbol to get a position
	position, file, err := findSymbolPosition(opts.Dir, opts.Symbol)
	if err != nil {
		return nil, err
	}

	// Check if target symbol already exists (conflict detection)
	if hasSymbol(opts.Dir, opts.To) {
		return nil, fmt.Errorf("conflict: symbol %q already exists", opts.To)
	}

	// Store original file contents for rollback
	originalContents, err := snapshotDirectory(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to snapshot directory: %w", err)
	}

	// Execute rename using gopls
	cmd := exec.Command("gopls", "rename", "-w", file+":"+position, opts.To)
	cmd.Dir = opts.Dir

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Rollback on error
		_ = restoreDirectory(opts.Dir, originalContents)

		errMsg := stderr.String()
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no identifier") {
			return nil, fmt.Errorf("symbol not found: %s", opts.Symbol)
		}

		return nil, fmt.Errorf("gopls rename failed: %w: %s", err, errMsg)
	}

	// Rename functions that contain the symbol name
	if err := renameFunctionsWithSymbol(opts.Dir, opts.Symbol, opts.To); err != nil {
		// Rollback on error
		_ = restoreDirectory(opts.Dir, originalContents)
		return nil, fmt.Errorf("failed to rename functions: %w", err)
	}

	// Count changed files
	filesChanged := countChangedFiles(opts.Dir, originalContents)

	return &RenameResult{
		FilesChanged: filesChanged,
	}, nil
}

// countChangedFiles counts how many files were modified.
func countChangedFiles(dir string, original map[string][]byte) int {
	count := 0

	for path, originalContent := range original {
		current, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if !bytes.Equal(originalContent, current) {
			count++
		}
	}

	return count
}

// findSymbolPosition finds the first occurrence of a symbol and returns its position.
func findSymbolPosition(dir, symbol string) (string, string, error) {
	var (
		foundFile string
		foundLine int
		foundCol  int
	)

	skipAll := errors.New("skip all")
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			// Look for symbol declaration patterns
			if strings.Contains(line, "type "+symbol+" ") ||
				strings.Contains(line, "type "+symbol+" struct") ||
				strings.Contains(line, "func "+symbol+"(") ||
				strings.Contains(line, "func New"+symbol+"(") {
				idx := strings.Index(line, symbol)
				foundFile = path
				foundLine = i + 1
				foundCol = idx + 1 // 1-indexed column

				return skipAll
			}
		}

		return nil
	})

	if err != nil && !errors.Is(err, skipAll) {
		return "", "", err
	}

	if foundFile == "" {
		return "", "", fmt.Errorf("symbol not found: %s", symbol)
	}

	return fmt.Sprintf("%d:%d", foundLine, foundCol), foundFile, nil
}

// hasSymbol checks if a symbol exists in the directory.
func hasSymbol(dir, symbol string) bool {
	found := false
	skipAll := errors.New("skip all")
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		patterns := []string{
			"type " + symbol + " ",
			"type " + symbol + " struct",
			"func " + symbol + "(",
		}
		for _, pattern := range patterns {
			if strings.Contains(string(content), pattern) {
				found = true
				return skipAll
			}
		}

		return nil
	})

	return found
}

// isValidIdentifier checks if a string is a valid Go identifier.
func isValidIdentifier(name string) bool {
	if len(name) == 0 {
		return false
	}
	// Must start with letter or underscore
	first := rune(name[0])
	if (first < 'a' || first > 'z') && (first < 'A' || first > 'Z') && first != '_' {
		return false
	}
	// Rest can be letters, digits, or underscore
	for _, ch := range name[1:] {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
			return false
		}
	}
	// Check for Go keywords
	keywords := map[string]bool{
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
	}

	return !keywords[name]
}

// renameFunctionsWithSymbol renames functions that contain the symbol name.
func renameFunctionsWithSymbol(dir, oldSymbol, newSymbol string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Replace function names containing the old symbol
		newContent := string(content)

		// Patterns to replace
		replacements := map[string]string{
			"func New" + oldSymbol + "(":    "func New" + newSymbol + "(",
			"func Use" + oldSymbol + "(":    "func Use" + newSymbol + "(",
			"func Get" + oldSymbol + "(":    "func Get" + newSymbol + "(",
			"func Set" + oldSymbol + "(":    "func Set" + newSymbol + "(",
			"func Delete" + oldSymbol + "(": "func Delete" + newSymbol + "(",
			"func Create" + oldSymbol + "(": "func Create" + newSymbol + "(",
			"func Update" + oldSymbol + "(": "func Update" + newSymbol + "(",
			"func " + oldSymbol + "(":       "func " + newSymbol + "(",
		}

		modified := false

		for old, new := range replacements {
			if strings.Contains(newContent, old) {
				newContent = strings.ReplaceAll(newContent, old, new)
				modified = true
			}
		}

		if modified {
			err := os.WriteFile(path, []byte(newContent), info.Mode())
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// restoreDirectory restores files to their original contents.
func restoreDirectory(dir string, snapshot map[string][]byte) error {
	for path, content := range snapshot {
		err := os.WriteFile(path, content, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// snapshotDirectory stores the contents of all .go files in the directory.
func snapshotDirectory(dir string) (map[string][]byte, error) {
	snapshot := make(map[string][]byte)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		snapshot[path] = content

		return nil
	})

	return snapshot, err
}
