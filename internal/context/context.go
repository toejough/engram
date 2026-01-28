// Package context manages skill dispatch context and result files.
package context

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ContextDir is the subdirectory for context files.
const ContextDir = "context"

// Filename returns the context file path for a given task and skill.
func Filename(task, skill string) string {
	return fmt.Sprintf("%s-%s.toml", task, skill)
}

// ResultFilename returns the result file path for a given task and skill.
func ResultFilename(task, skill string) string {
	return fmt.Sprintf("%s-%s.result.toml", task, skill)
}

// Write copies a TOML file into the context directory with the correct naming convention.
// Returns an error if the target file already exists.
func Write(dir, task, skill, sourcePath string) (string, error) {
	contextDir := filepath.Join(dir, ContextDir)
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create context directory: %w", err)
	}

	// Validate source is parseable TOML
	if _, err := os.Stat(sourcePath); err != nil {
		return "", fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	var raw any
	if _, err := toml.DecodeFile(sourcePath, &raw); err != nil {
		return "", fmt.Errorf("source file is not valid TOML: %w", err)
	}

	targetName := Filename(task, skill)
	targetPath := filepath.Join(contextDir, targetName)

	if _, err := os.Stat(targetPath); err == nil {
		return "", fmt.Errorf("context file already exists: %s", targetPath)
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	if err := os.WriteFile(targetPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write context file: %w", err)
	}

	return targetPath, nil
}

// Read reads a context or result file for a given task and skill.
func Read(dir, task, skill string, isResult bool) (string, error) {
	var name string
	if isResult {
		name = ResultFilename(task, skill)
	} else {
		name = Filename(task, skill)
	}

	filePath := filepath.Join(dir, ContextDir, name)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read context file %s: %w", name, err)
	}

	return string(data), nil
}
