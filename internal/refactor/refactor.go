// Package refactor provides automated refactoring operations using LSP.
package refactor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

// findSymbolPosition finds the first occurrence of a symbol and returns its position.
func findSymbolPosition(dir, symbol string) (string, string, error) {
	var foundFile string
	var foundLine int
	var foundCol int

	skipAll := fmt.Errorf("skip all")
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

	if err != nil && err != skipAll {
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
	skipAll := fmt.Errorf("skip all")
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

// restoreDirectory restores files to their original contents.
func restoreDirectory(dir string, snapshot map[string][]byte) error {
	for path, content := range snapshot {
		if err := os.WriteFile(path, content, 0644); err != nil {
			return err
		}
	}
	return nil
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
			"func New" + oldSymbol + "(": "func New" + newSymbol + "(",
			"func Use" + oldSymbol + "(": "func Use" + newSymbol + "(",
			"func Get" + oldSymbol + "(": "func Get" + newSymbol + "(",
			"func Set" + oldSymbol + "(": "func Set" + newSymbol + "(",
			"func Delete" + oldSymbol + "(": "func Delete" + newSymbol + "(",
			"func Create" + oldSymbol + "(": "func Create" + newSymbol + "(",
			"func Update" + oldSymbol + "(": "func Update" + newSymbol + "(",
			"func " + oldSymbol + "(": "func " + newSymbol + "(",
		}

		modified := false
		for old, new := range replacements {
			if strings.Contains(newContent, old) {
				newContent = strings.ReplaceAll(newContent, old, new)
				modified = true
			}
		}

		if modified {
			if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
				return err
			}
		}

		return nil
	})
}
