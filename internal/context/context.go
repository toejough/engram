// Package context manages skill dispatch context and result files.
package context

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/toejough/projctl/internal/territory"
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

	// Overwrite existing file if present (no error check needed)

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

// RoutingConfig defines model routing for different complexity levels.
type RoutingConfig struct {
	Simple  string
	Medium  string
	Complex string
}

// RoutingInfo is added to context files.
type RoutingInfo struct {
	SuggestedModel string `toml:"suggested_model"`
	Reason         string `toml:"reason"`
}

// WriteParallel creates context files for multiple tasks using a shared template.
func WriteParallel(dir string, tasks []string, skill, templatePath string) ([]string, error) {
	var paths []string

	for _, task := range tasks {
		path, err := Write(dir, task, skill, templatePath)
		if err != nil {
			return nil, fmt.Errorf("failed to write context for %s: %w", task, err)
		}
		paths = append(paths, path)
	}

	return paths, nil
}

// WriteWithRouting copies a TOML file into the context directory and adds routing information.
func WriteWithRouting(dir, task, skill, sourcePath string, routing RoutingConfig, skillComplexity map[string]string) (string, error) {
	contextDir := filepath.Join(dir, ContextDir)
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create context directory: %w", err)
	}

	// Validate source is parseable TOML
	if _, err := os.Stat(sourcePath); err != nil {
		return "", fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	var raw map[string]any
	if _, err := toml.DecodeFile(sourcePath, &raw); err != nil {
		return "", fmt.Errorf("source file is not valid TOML: %w", err)
	}

	// Determine complexity and model
	complexity, ok := skillComplexity[skill]
	if !ok {
		complexity = "medium" // default
	}

	var model string
	switch complexity {
	case "simple":
		model = routing.Simple
	case "complex":
		model = routing.Complex
	default:
		model = routing.Medium
	}

	// Add routing section
	raw["routing"] = RoutingInfo{
		SuggestedModel: model,
		Reason:         fmt.Sprintf("%s skill: %s complexity", skill, complexity),
	}

	targetName := Filename(task, skill)
	targetPath := filepath.Join(contextDir, targetName)

	f, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to create context file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(raw); err != nil {
		return "", fmt.Errorf("failed to encode TOML: %w", err)
	}

	return targetPath, nil
}

// WriteWithTerritory copies a TOML file and automatically includes cached territory map.
func WriteWithTerritory(dir, task, skill, sourcePath string) (string, error) {
	contextDir := filepath.Join(dir, ContextDir)
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create context directory: %w", err)
	}

	// Validate source is parseable TOML
	if _, err := os.Stat(sourcePath); err != nil {
		return "", fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	var raw map[string]any
	if _, err := toml.DecodeFile(sourcePath, &raw); err != nil {
		return "", fmt.Errorf("source file is not valid TOML: %w", err)
	}

	// Try to load cached territory map
	cachePath := filepath.Join(dir, territory.CacheFile)
	if data, err := os.ReadFile(cachePath); err == nil {
		var cached territory.CachedMap
		if _, err := toml.Decode(string(data), &cached); err == nil {
			// Add territory section with the map contents
			raw["territory"] = cached.Map
		}
	}

	targetName := Filename(task, skill)
	targetPath := filepath.Join(contextDir, targetName)

	f, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to create context file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(raw); err != nil {
		return "", fmt.Errorf("failed to encode TOML: %w", err)
	}

	return targetPath, nil
}
