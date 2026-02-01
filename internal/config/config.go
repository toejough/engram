// Package config provides project configuration loading and management.
package config

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ConfigFS provides file system operations for config loading.
type ConfigFS interface {
	ReadFile(path string) (string, error)
	FileExists(path string) bool
}

// ProjectConfig holds all project configuration.
type ProjectConfig struct {
	Paths        PathsConfig        `toml:"paths"`
	Heuristics   HeuristicsConfig   `toml:"heuristics"`
	Traceability TraceabilityConfig `toml:"traceability"`
	Routing      RoutingConfig      `toml:"routing"`
	Budget       BudgetConfig       `toml:"budget"`
}

// BudgetConfig defines token usage budget thresholds.
type BudgetConfig struct {
	WarningTokens int `toml:"warning_tokens"` // Warn when usage exceeds this
	LimitTokens   int `toml:"limit_tokens"`   // Hard limit (exit code 2)
}

// RoutingConfig defines model routing for different complexity levels.
type RoutingConfig struct {
	Simple          string            `toml:"simple"`           // Model for simple tasks (default: sonnet)
	Medium          string            `toml:"medium"`           // Model for medium tasks (default: sonnet)
	Complex         string            `toml:"complex"`          // Model for complex tasks (default: sonnet)
	ThresholdLines  int               `toml:"threshold_lines"`  // Lines threshold for complexity (default: 100)
	SkillComplexity map[string]string `toml:"skill_complexity"` // Skill name -> complexity level mapping
}

// ValidModels lists known valid model names.
var ValidModels = map[string]bool{
	"haiku":  true,
	"sonnet": true,
	"opus":   true,
}

// IsValidModel checks if a model name is valid.
func IsValidModel(model string) bool {
	return ValidModels[model]
}

// GetRouting returns the routing configuration.
func (c *ProjectConfig) GetRouting() RoutingConfig {
	return c.Routing
}

// PathsConfig defines artifact paths.
type PathsConfig struct {
	Readme       string `toml:"readme"`
	DocsDir      string `toml:"docs_dir"`
	Requirements string `toml:"requirements"`
	Design       string `toml:"design"`
	Architecture string `toml:"architecture"`
	Tasks        string `toml:"tasks"`
	Tests        string `toml:"tests"`
	Issues       string `toml:"issues"`
	Glossary     string `toml:"glossary"`
	Traceability string `toml:"traceability"`
	ProjectsDir  string `toml:"projects_dir"`
}

// HeuristicsConfig defines analysis heuristics.
type HeuristicsConfig struct {
	PreserveThreshold float64 `toml:"preserve_threshold"`
	MigrateThreshold  float64 `toml:"migrate_threshold"`
	AnalysisDepth     string  `toml:"analysis_depth"`
}

// TraceabilityConfig defines ID prefixes.
type TraceabilityConfig struct {
	RequirementPrefix  string `toml:"requirement_prefix"`
	DesignPrefix       string `toml:"design_prefix"`
	ArchitecturePrefix string `toml:"architecture_prefix"`
	TaskPrefix         string `toml:"task_prefix"`
	TestPrefix         string `toml:"test_prefix"`
	IssuePrefix        string `toml:"issue_prefix"`
}

// Load loads project configuration from repo and global paths.
// Repo config takes precedence over global config.
// Returns defaults if no config exists.
func Load(repoDir, homeDir string, fs ConfigFS) (*ProjectConfig, error) {
	cfg := Default()

	// Try global config first (lower precedence)
	globalPath := filepath.Join(homeDir, ".claude", "project-config.toml")
	if fs.FileExists(globalPath) {
		if err := loadInto(globalPath, cfg, fs); err != nil {
			return nil, err
		}
	}

	// Try repo config (higher precedence, overwrites global)
	repoPath := filepath.Join(repoDir, ".claude", "project-config.toml")
	if fs.FileExists(repoPath) {
		if err := loadInto(repoPath, cfg, fs); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// loadInto loads TOML config from path into existing config struct.
func loadInto(path string, cfg *ProjectConfig, fs ConfigFS) error {
	content, err := fs.ReadFile(path)
	if err != nil {
		return err
	}

	_, err = toml.Decode(content, cfg)
	return err
}

// Default returns the default configuration.
func Default() *ProjectConfig {
	return &ProjectConfig{
		Paths: PathsConfig{
			Readme:       "README.md",
			DocsDir:      "docs",
			Requirements: "requirements.md",
			Design:       "design.md",
			Architecture: "architecture.md",
			Tasks:        "tasks.md",
			Tests:        "tests.md",
			Issues:       "issues.md",
			Glossary:     "glossary.md",
			Traceability: "traceability.toml",
			ProjectsDir:  "projects",
		},
		Heuristics: HeuristicsConfig{
			PreserveThreshold: 0.60,
			MigrateThreshold:  0.40,
			AnalysisDepth:     "deep",
		},
		Traceability: TraceabilityConfig{
			RequirementPrefix:  "REQ",
			DesignPrefix:       "DES",
			ArchitecturePrefix: "ARCH",
			TaskPrefix:         "TASK",
			TestPrefix:         "TEST",
			IssuePrefix:        "ISSUE",
		},
		Routing: RoutingConfig{
			Simple:         "sonnet",
			Medium:         "sonnet",
			Complex:        "sonnet",
			ThresholdLines: 100,
			SkillComplexity: map[string]string{
				// Simple skills - lightweight checks
				"alignment-check": "simple",
				// Medium skills - standard development work
				"tdd-red":      "medium",
				"tdd-green":    "medium",
				"tdd-refactor": "medium",
				"commit":       "medium",
				// Complex skills - deep analysis
				"meta-audit":         "complex",
				"architect-interview": "complex",
				"pm-interview":        "complex",
			},
		},
	}
}

// ResolvePath resolves an artifact name to its full path.
// README is at root; other artifacts are relative to docs_dir.
func (c *ProjectConfig) ResolvePath(artifact string) string {
	switch artifact {
	case "readme":
		return c.Paths.Readme
	case "requirements":
		return filepath.Join(c.Paths.DocsDir, c.Paths.Requirements)
	case "design":
		return filepath.Join(c.Paths.DocsDir, c.Paths.Design)
	case "architecture":
		return filepath.Join(c.Paths.DocsDir, c.Paths.Architecture)
	case "tasks":
		return filepath.Join(c.Paths.DocsDir, c.Paths.Tasks)
	case "tests":
		return filepath.Join(c.Paths.DocsDir, c.Paths.Tests)
	case "issues":
		return filepath.Join(c.Paths.DocsDir, c.Paths.Issues)
	case "glossary":
		return filepath.Join(c.Paths.DocsDir, c.Paths.Glossary)
	case "traceability":
		return filepath.Join(c.Paths.DocsDir, c.Paths.Traceability)
	case "projects":
		return filepath.Join(c.Paths.DocsDir, c.Paths.ProjectsDir)
	default:
		return ""
	}
}
