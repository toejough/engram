// Package config provides project configuration loading and management.
package config

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
}

// PathsConfig defines artifact paths.
type PathsConfig struct {
	Readme       string `toml:"readme"`
	DocsDir      string `toml:"docs_dir"`
	Requirements string `toml:"requirements"`
	Design       string `toml:"design"`
	Architecture string `toml:"architecture"`
	Tasks        string `toml:"tasks"`
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
	return nil, nil
}

// Default returns the default configuration.
func Default() *ProjectConfig {
	return nil
}

// ResolvePath resolves an artifact name to its full path.
func (c *ProjectConfig) ResolvePath(artifact string) string {
	return ""
}
