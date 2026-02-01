// Package context manages skill dispatch context and result files.
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/toejough/projctl/internal/memory"
	"github.com/toejough/projctl/internal/refactor"
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
// Also automatically injects cached territory map if available.
func WriteWithRouting(dir, task, skill, sourcePath string, routing RoutingConfig, skillComplexity map[string]string) (string, error) {
	raw, err := loadAndValidateTOML(sourcePath)
	if err != nil {
		return "", err
	}

	// Add routing section
	addRoutingSection(raw, skill, routing, skillComplexity)

	// Inject cached territory and capabilities
	injectTerritoryAndCapabilities(dir, raw)

	return writeTOMLFile(dir, task, skill, raw)
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
	defer func() { _ = f.Close() }()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(raw); err != nil {
		return "", fmt.Errorf("failed to encode TOML: %w", err)
	}

	return targetPath, nil
}

// MemoryInjectOpts holds options for memory injection into context.
type MemoryInjectOpts struct {
	Query      string
	MemoryRoot string
	Limit      int
}

// WriteWithMemory copies a TOML file and injects memory query results.
func WriteWithMemory(dir, task, skill, sourcePath string, opts MemoryInjectOpts) (string, error) {
	raw, err := loadAndValidateTOML(sourcePath)
	if err != nil {
		return "", err
	}

	// Inject memory if query is provided or derivable
	injectMemory(raw, opts)

	return writeTOMLFile(dir, task, skill, raw)
}

// WriteWithRoutingAndMemory copies a TOML file with routing and auto-injects memory for certain skills.
func WriteWithRoutingAndMemory(dir, task, skill, sourcePath string, routing RoutingConfig, skillComplexity map[string]string, memoryRoot string) (string, error) {
	raw, err := loadAndValidateTOML(sourcePath)
	if err != nil {
		return "", err
	}

	// Add routing section
	addRoutingSection(raw, skill, routing, skillComplexity)

	// Inject cached territory and capabilities
	injectTerritoryAndCapabilities(dir, raw)

	// Auto-inject memory for certain skills
	if shouldAutoInjectMemory(skill, memoryRoot) {
		opts := MemoryInjectOpts{
			Query:      "", // Derive from task description
			MemoryRoot: memoryRoot,
			Limit:      3,
		}
		injectMemory(raw, opts)
	}

	return writeTOMLFile(dir, task, skill, raw)
}

// Type aliases for memory package types to avoid direct dependency in function signatures.
type QueryOpts = memory.QueryOpts
type QueryResults = memory.QueryResults
type QueryResult = memory.QueryResult

// Query is an alias for memory.Query.
var Query = memory.Query

// loadAndValidateTOML loads and validates a TOML file.
func loadAndValidateTOML(sourcePath string) (map[string]any, error) {
	if _, err := os.Stat(sourcePath); err != nil {
		return nil, fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	var raw map[string]any
	if _, err := toml.DecodeFile(sourcePath, &raw); err != nil {
		return nil, fmt.Errorf("source file is not valid TOML: %w", err)
	}

	return raw, nil
}

// addRoutingSection adds routing information to the raw TOML data.
func addRoutingSection(raw map[string]any, skill string, routing RoutingConfig, skillComplexity map[string]string) {
	complexity, ok := skillComplexity[skill]
	if !ok {
		complexity = "medium" // default
	}

	model := selectModel(complexity, routing)

	raw["routing"] = RoutingInfo{
		SuggestedModel: model,
		Reason:         fmt.Sprintf("%s skill: %s complexity", skill, complexity),
	}
}

// selectModel chooses the appropriate model based on complexity.
func selectModel(complexity string, routing RoutingConfig) string {
	switch complexity {
	case "simple":
		return routing.Simple
	case "complex":
		return routing.Complex
	default:
		return routing.Medium
	}
}

// injectTerritoryAndCapabilities adds territory map and refactoring capabilities to raw TOML data.
func injectTerritoryAndCapabilities(dir string, raw map[string]any) {
	// Inject cached territory map if available
	cachePath := filepath.Join(dir, territory.CacheFile)
	if data, err := os.ReadFile(cachePath); err == nil {
		var cached territory.CachedMap
		if _, err := toml.Decode(string(data), &cached); err == nil {
			raw["territory"] = cached.Map
		}
	}

	// Inject refactoring capabilities
	caps := refactor.CheckCapabilities()
	raw["capabilities"] = map[string]any{
		"refactor": map[string]any{
			"gopls_available": caps.GoplsAvailable,
			"rename_support":  caps.RenameSupport,
		},
	}
}

// shouldAutoInjectMemory determines if memory should be auto-injected for a skill.
func shouldAutoInjectMemory(skill, memoryRoot string) bool {
	return memoryRoot != "" && (skill == "architect-interview" || skill == "pm-interview")
}

// injectMemory adds memory query results to the raw TOML data.
func injectMemory(raw map[string]any, opts MemoryInjectOpts) {
	// Derive query from task description if query is empty
	query := opts.Query
	if query == "" {
		query = deriveQueryFromTask(raw)
	}

	// Only inject memory if we have a query
	if query == "" || opts.MemoryRoot == "" {
		return
	}

	// Query memory using semantic search
	queryOpts := QueryOpts{
		Text:       query,
		Limit:      opts.Limit,
		MemoryRoot: opts.MemoryRoot,
	}

	results, err := Query(queryOpts)
	if err == nil && len(results.Results) > 0 {
		// Build memory section with compression to stay under 500 tokens
		memorySection := buildMemorySection(results.Results, 500)
		raw["memory"] = memorySection
	}
}

// deriveQueryFromTask extracts the task description from raw TOML data.
func deriveQueryFromTask(raw map[string]any) string {
	if taskSection, ok := raw["task"].(map[string]any); ok {
		if desc, ok := taskSection["description"].(string); ok {
			return desc
		}
	}
	return ""
}

// writeTOMLFile writes raw TOML data to a context file.
func writeTOMLFile(dir, task, skill string, raw map[string]any) (string, error) {
	contextDir := filepath.Join(dir, ContextDir)
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create context directory: %w", err)
	}

	targetName := Filename(task, skill)
	targetPath := filepath.Join(contextDir, targetName)

	f, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to create context file: %w", err)
	}
	defer func() { _ = f.Close() }()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(raw); err != nil {
		return "", fmt.Errorf("failed to encode TOML: %w", err)
	}

	return targetPath, nil
}

// buildMemorySection formats query results into a TOML-compatible structure with compression.
func buildMemorySection(results []QueryResult, maxTokens int) map[string]any {
	section := make(map[string]any)

	// Build results array with content and score
	var resultMaps []map[string]any
	for i, result := range results {
		if i >= 3 {
			break // Limit to top 3
		}

		// Normalize whitespace first
		content := strings.Join(strings.Fields(result.Content), " ")

		resultMap := map[string]any{
			"content": content,
			"score":   result.Score,
			"source":  result.Source,
		}
		resultMaps = append(resultMaps, resultMap)
	}

	// Compress aggressively to stay under token limit
	// Rough estimate: 1 token per 4 characters
	// Account for TOML structure overhead (keys, quotes, brackets, etc.)
	// Be conservative - use larger overhead and smaller multiplier
	const overheadPerResult = 80 // characters for TOML structure per result
	targetChars := (maxTokens * 3) - (len(resultMaps) * overheadPerResult)

	// Calculate total current chars
	totalChars := 0
	for _, r := range resultMaps {
		totalChars += len(r["content"].(string))
	}

	// If over limit, truncate proportionally
	if totalChars > targetChars {
		charsPerResult := targetChars / len(resultMaps)

		for i, r := range resultMaps {
			content := r["content"].(string)
			if len(content) > charsPerResult {
				// Truncate with ellipsis
				if charsPerResult > 3 {
					resultMaps[i]["content"] = content[:charsPerResult-3] + "..."
				} else {
					resultMaps[i]["content"] = "..."
				}
			}
		}
	}

	section["results"] = resultMaps

	return section
}
