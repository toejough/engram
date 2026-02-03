package config_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/config"
)

// Mock file system for config tests
type mockConfigFS struct {
	files map[string]string // path -> content
}

func (m *mockConfigFS) ReadFile(path string) (string, error) {
	content, exists := m.files[path]
	if !exists {
		return "", &fileNotFoundError{path: path}
	}
	return content, nil
}

func (m *mockConfigFS) FileExists(path string) bool {
	_, exists := m.files[path]
	return exists
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

// TEST-185 traces: TASK-001
// Test Load reads config from repo .claude/project-config.toml
func TestLoad_RepoConfig(t *testing.T) {
	g := NewWithT(t)

	fs := &mockConfigFS{
		files: map[string]string{
			"/project/.claude/project-config.toml": `
[paths]
readme = "README.md"
docs_dir = "documentation"
requirements = "reqs.md"

[heuristics]
preserve_threshold = 0.70
migrate_threshold = 0.30

[traceability]
requirement_prefix = "REQ"
`,
		},
	}

	cfg, err := config.Load("/project", "/home/user", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(cfg.Paths.DocsDir).To(Equal("documentation"))
	g.Expect(cfg.Paths.Requirements).To(Equal("reqs.md"))
	g.Expect(cfg.Heuristics.PreserveThreshold).To(Equal(0.70))
	g.Expect(cfg.Heuristics.MigrateThreshold).To(Equal(0.30))
}

// TEST-186 traces: TASK-001
// Test Load falls back to global config when repo config missing
func TestLoad_GlobalFallback(t *testing.T) {
	g := NewWithT(t)

	fs := &mockConfigFS{
		files: map[string]string{
			"/home/user/.claude/project-config.toml": `
[paths]
docs_dir = "docs"
requirements = "requirements.md"

[heuristics]
preserve_threshold = 0.60
`,
		},
	}

	cfg, err := config.Load("/project", "/home/user", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(cfg.Paths.DocsDir).To(Equal("docs"))
	g.Expect(cfg.Heuristics.PreserveThreshold).To(Equal(0.60))
}

// TEST-187 traces: TASK-001
// Test Load returns defaults when no config exists
func TestLoad_Defaults(t *testing.T) {
	g := NewWithT(t)

	fs := &mockConfigFS{
		files: map[string]string{}, // No config files
	}

	cfg, err := config.Load("/project", "/home/user", fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Check defaults - DocsDir is empty by default (artifacts at project root)
	g.Expect(cfg.Paths.Readme).To(Equal("README.md"))
	g.Expect(cfg.Paths.DocsDir).To(Equal(""))
	g.Expect(cfg.Paths.Requirements).To(Equal("requirements.md"))
	g.Expect(cfg.Paths.Design).To(Equal("design.md"))
	g.Expect(cfg.Paths.Architecture).To(Equal("architecture.md"))
	g.Expect(cfg.Paths.Tasks).To(Equal("tasks.md"))
	g.Expect(cfg.Paths.Issues).To(Equal("issues.md"))
	g.Expect(cfg.Paths.Glossary).To(Equal("glossary.md"))
	g.Expect(cfg.Paths.Traceability).To(Equal("traceability.toml"))
	g.Expect(cfg.Paths.ProjectsDir).To(Equal("projects"))

	g.Expect(cfg.Heuristics.PreserveThreshold).To(Equal(0.60))
	g.Expect(cfg.Heuristics.MigrateThreshold).To(Equal(0.40))
	g.Expect(cfg.Heuristics.AnalysisDepth).To(Equal("deep"))

	g.Expect(cfg.Traceability.RequirementPrefix).To(Equal("REQ"))
	g.Expect(cfg.Traceability.DesignPrefix).To(Equal("DES"))
	g.Expect(cfg.Traceability.ArchitecturePrefix).To(Equal("ARCH"))
	g.Expect(cfg.Traceability.TaskPrefix).To(Equal("TASK"))
	g.Expect(cfg.Traceability.TestPrefix).To(Equal("TEST"))
	g.Expect(cfg.Traceability.IssuePrefix).To(Equal("ISSUE"))
}

// TEST-188 traces: TASK-001
// Test Load repo config overrides global config
func TestLoad_RepoOverridesGlobal(t *testing.T) {
	g := NewWithT(t)

	fs := &mockConfigFS{
		files: map[string]string{
			"/home/user/.claude/project-config.toml": `
[paths]
docs_dir = "global-docs"

[heuristics]
preserve_threshold = 0.50
`,
			"/project/.claude/project-config.toml": `
[paths]
docs_dir = "repo-docs"
`,
		},
	}

	cfg, err := config.Load("/project", "/home/user", fs)
	g.Expect(err).ToNot(HaveOccurred())
	// Repo config wins for docs_dir
	g.Expect(cfg.Paths.DocsDir).To(Equal("repo-docs"))
	// Global config used for preserve_threshold (not in repo config)
	g.Expect(cfg.Heuristics.PreserveThreshold).To(Equal(0.50))
}

// TEST-189 traces: TASK-001
// Test ResolvePath combines docs_dir with artifact path
func TestResolvePath(t *testing.T) {
	g := NewWithT(t)

	cfg := &config.ProjectConfig{
		Paths: config.PathsConfig{
			Readme:       "README.md",
			DocsDir:      "docs",
			Requirements: "requirements.md",
			Design:       "design.md",
			Architecture: "architecture.md",
			Tasks:        "tasks.md",
			Issues:       "issues.md",
			Traceability: "traceability.toml",
			ProjectsDir:  "projects",
		},
	}

	// Docs artifacts use docs_dir
	g.Expect(cfg.ResolvePath("requirements")).To(Equal(filepath.Join("docs", "requirements.md")))
	g.Expect(cfg.ResolvePath("design")).To(Equal(filepath.Join("docs", "design.md")))
	g.Expect(cfg.ResolvePath("architecture")).To(Equal(filepath.Join("docs", "architecture.md")))
	g.Expect(cfg.ResolvePath("tasks")).To(Equal(filepath.Join("docs", "tasks.md")))
	g.Expect(cfg.ResolvePath("issues")).To(Equal(filepath.Join("docs", "issues.md")))
	g.Expect(cfg.ResolvePath("traceability")).To(Equal(filepath.Join("docs", "traceability.toml")))
	g.Expect(cfg.ResolvePath("projects")).To(Equal(filepath.Join("docs", "projects")))

	// README is at root, not in docs_dir
	g.Expect(cfg.ResolvePath("readme")).To(Equal("README.md"))
}

// TEST-190 traces: TASK-001
// Test ResolvePath with custom docs_dir
func TestResolvePath_CustomDocsDir(t *testing.T) {
	g := NewWithT(t)

	cfg := &config.ProjectConfig{
		Paths: config.PathsConfig{
			DocsDir:      "documentation",
			Requirements: "reqs.md",
		},
	}

	g.Expect(cfg.ResolvePath("requirements")).To(Equal(filepath.Join("documentation", "reqs.md")))
}

// TEST-191 traces: TASK-001
// Test ResolvePath returns empty for unknown artifact
func TestResolvePath_Unknown(t *testing.T) {
	g := NewWithT(t)

	cfg := config.Default()
	g.Expect(cfg.ResolvePath("unknown")).To(Equal(""))
}

// TEST-192 traces: TASK-001
// Test Default returns valid config
func TestDefault(t *testing.T) {
	g := NewWithT(t)

	cfg := config.Default()
	g.Expect(cfg).ToNot(BeNil())
	g.Expect(cfg.Paths.DocsDir).To(Equal("")) // Empty by default - artifacts at project root
	g.Expect(cfg.Heuristics.PreserveThreshold).To(BeNumerically(">", 0))
}

// TEST-400 traces: TASK-014
// Test routing config has default values (all sonnet)
func TestRouting_Defaults(t *testing.T) {
	g := NewWithT(t)

	cfg := config.Default()
	g.Expect(cfg.Routing.Simple).To(Equal("sonnet"))
	g.Expect(cfg.Routing.Medium).To(Equal("sonnet"))
	g.Expect(cfg.Routing.Complex).To(Equal("sonnet"))
	g.Expect(cfg.Routing.ThresholdLines).To(Equal(100))
}

// TEST-401 traces: TASK-014
// Test routing config loads from TOML
func TestRouting_Load(t *testing.T) {
	g := NewWithT(t)

	fs := &mockConfigFS{
		files: map[string]string{
			"/project/.claude/project-config.toml": `
[routing]
simple = "haiku"
medium = "sonnet"
complex = "opus"
threshold_lines = 50
`,
		},
	}

	cfg, err := config.Load("/project", "/home/user", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(cfg.Routing.Simple).To(Equal("haiku"))
	g.Expect(cfg.Routing.Medium).To(Equal("sonnet"))
	g.Expect(cfg.Routing.Complex).To(Equal("opus"))
	g.Expect(cfg.Routing.ThresholdLines).To(Equal(50))
}

// TEST-402 traces: TASK-014
// Test routing config validates model names
func TestRouting_ValidateModels(t *testing.T) {
	g := NewWithT(t)

	// Valid models
	g.Expect(config.IsValidModel("haiku")).To(BeTrue())
	g.Expect(config.IsValidModel("sonnet")).To(BeTrue())
	g.Expect(config.IsValidModel("opus")).To(BeTrue())

	// Invalid models
	g.Expect(config.IsValidModel("gpt-4")).To(BeFalse())
	g.Expect(config.IsValidModel("")).To(BeFalse())
	g.Expect(config.IsValidModel("claude")).To(BeFalse())
}

// TEST-403 traces: TASK-014
// Test GetRouting returns routing config
func TestGetRouting(t *testing.T) {
	g := NewWithT(t)

	cfg := config.Default()
	routing := cfg.GetRouting()
	g.Expect(routing.Simple).To(Equal("sonnet"))
	g.Expect(routing.Medium).To(Equal("sonnet"))
	g.Expect(routing.Complex).To(Equal("sonnet"))
}
