package config_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/config"
)

// TEST-192 traces: TASK-001
// Test Default returns valid config
func TestDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfg := config.Default()
	g.Expect(cfg).ToNot(BeNil())
	g.Expect(cfg.Paths.DocsDir).To(Equal("")) // Empty by default - artifacts at project root
	g.Expect(cfg.Heuristics.PreserveThreshold).To(BeNumerically(">", 0))
}

// TEST-403 traces: TASK-014
// Test GetRouting returns routing config
func TestGetRouting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfg := config.Default()
	routing := cfg.GetRouting()
	g.Expect(routing.Simple).To(Equal("sonnet"))
	g.Expect(routing.Medium).To(Equal("sonnet"))
	g.Expect(routing.Complex).To(Equal("sonnet"))
}

// TEST-187 traces: TASK-001
// Test Load returns defaults when no config exists
func TestLoad_Defaults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &mockConfigFS{
		files: map[string]string{}, // No config files
	}

	cfg, err := config.Load("/project", "/home/user", fs)
	g.Expect(err).ToNot(HaveOccurred())

	if cfg == nil {
		t.Fatal("cfg is nil")
	}

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

// TEST-186 traces: TASK-001
// Test Load falls back to global config when repo config missing
func TestLoad_GlobalFallback(t *testing.T) {
	t.Parallel()
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

	if cfg == nil {
		t.Fatal("cfg is nil")
	}

	g.Expect(cfg.Paths.DocsDir).To(Equal("docs"))
	g.Expect(cfg.Heuristics.PreserveThreshold).To(Equal(0.60))
}

// TEST-185 traces: TASK-001
// Test Load reads config from repo .claude/project-config.toml
func TestLoad_RepoConfig(t *testing.T) {
	t.Parallel()
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

	if cfg == nil {
		t.Fatal("cfg is nil")
	}

	g.Expect(cfg.Paths.DocsDir).To(Equal("documentation"))
	g.Expect(cfg.Paths.Requirements).To(Equal("reqs.md"))
	g.Expect(cfg.Heuristics.PreserveThreshold).To(Equal(0.70))
	g.Expect(cfg.Heuristics.MigrateThreshold).To(Equal(0.30))
}

// TEST-188 traces: TASK-001
// Test Load repo config overrides global config
func TestLoad_RepoOverridesGlobal(t *testing.T) {
	t.Parallel()
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

	if cfg == nil {
		t.Fatal("cfg is nil")
	}

	// Repo config wins for docs_dir
	g.Expect(cfg.Paths.DocsDir).To(Equal("repo-docs"))
	// Global config used for preserve_threshold (not in repo config)
	g.Expect(cfg.Heuristics.PreserveThreshold).To(Equal(0.50))
}

// TestRealFS_FileExists_Exists verifies FileExists returns true for existing file.
func TestRealFS_FileExists_Exists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")
	err := os.WriteFile(path, []byte("content"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	fs := config.RealFS{}
	g.Expect(fs.FileExists(path)).To(BeTrue())
}

// TestRealFS_FileExists_NotExists verifies FileExists returns false for missing file.
func TestRealFS_FileExists_NotExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	fs := config.RealFS{}
	g.Expect(fs.FileExists(filepath.Join(dir, "nonexistent.toml"))).To(BeFalse())
}

// TestRealFS_ReadFile_Error verifies ReadFile returns error for missing file.
func TestRealFS_ReadFile_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	fs := config.RealFS{}
	_, err := fs.ReadFile(filepath.Join(dir, "nonexistent.toml"))
	g.Expect(err).To(HaveOccurred())
}

// TestRealFS_ReadFile_Success verifies ReadFile returns content for existing file.
func TestRealFS_ReadFile_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")
	err := os.WriteFile(path, []byte("[paths]\nreadme = \"README.md\"\n"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	fs := config.RealFS{}
	content, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(ContainSubstring("README.md"))
}

// TEST-189 traces: TASK-001
// Test ResolvePath combines docs_dir with artifact path
func TestResolvePath(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	g := NewWithT(t)

	cfg := config.Default()
	g.Expect(cfg.ResolvePath("unknown")).To(Equal(""))
}

// TEST-400 traces: TASK-014
// Test routing config has default values (all sonnet)
func TestRouting_Defaults(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

	if cfg == nil {
		t.Fatal("cfg is nil")
	}

	g.Expect(cfg.Routing.Simple).To(Equal("haiku"))
	g.Expect(cfg.Routing.Medium).To(Equal("sonnet"))
	g.Expect(cfg.Routing.Complex).To(Equal("opus"))
	g.Expect(cfg.Routing.ThresholdLines).To(Equal(50))
}

// TEST-402 traces: TASK-014
// Test routing config validates model names
func TestRouting_ValidateModels(t *testing.T) {
	t.Parallel()
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

// TestRunGet_AllKeys verifies RunGet prints all config as JSON when no key specified.
func TestRunGet_AllKeys(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := config.RunGet(config.GetArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunGet_AllSwitchKeys verifies RunGet handles every valid config key in the switch.
func TestRunGet_AllSwitchKeys(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	keys := []string{
		"paths.requirements",
		"paths.design",
		"paths.architecture",
		"paths.tasks",
		"paths.issues",
		"paths.glossary",
		"paths.traceability",
		"paths.projects_dir",
		"heuristics.preserve_threshold",
		"heuristics.migrate_threshold",
		"heuristics.analysis_depth",
	}

	for _, key := range keys {
		err := config.RunGet(config.GetArgs{Dir: dir, Key: key})
		g.Expect(err).ToNot(HaveOccurred())
	}
}

// TestRunGet_EmptyDir verifies RunGet uses current directory when Dir is empty.
func TestRunGet_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := config.RunGet(config.GetArgs{Dir: "", Key: "paths.readme"})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunGet_SpecificKey verifies RunGet prints a specific config value.
func TestRunGet_SpecificKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := config.RunGet(config.GetArgs{Dir: dir, Key: "paths.docs_dir"})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunGet_UnknownKey verifies RunGet returns error for unknown config key.
func TestRunGet_UnknownKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := config.RunGet(config.GetArgs{Dir: dir, Key: "unknown.key"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown config key"))
	}
}

// TestRunInit_AlreadyExists verifies RunInit errors when config already exists.
func TestRunInit_AlreadyExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := config.RunInit(config.InitArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())

	err = config.RunInit(config.InitArgs{Dir: dir})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("already exists"))
	}
}

// TestRunInit_EmptyDir verifies RunInit uses current directory when Dir is empty.
// Not parallel: temporarily changes working directory.
func TestRunInit_EmptyDir(t *testing.T) {
	g := NewWithT(t)

	dir := t.TempDir()
	t.Chdir(dir)

	err := config.RunInit(config.InitArgs{Dir: ""})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunInit_Success verifies RunInit creates a config file.
func TestRunInit_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := config.RunInit(config.InitArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())

	configPath := filepath.Join(dir, ".claude", "project-config.toml")
	g.Expect(configPath).To(BeAnExistingFile())
}

// TestRunPath_EmptyDir verifies RunPath uses current directory when Dir is empty.
func TestRunPath_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := config.RunPath(config.PathArgs{Dir: "", Artifact: "readme"})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunPath_UnknownArtifact verifies RunPath returns error for unknown artifact.
func TestRunPath_UnknownArtifact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := config.RunPath(config.PathArgs{Dir: dir, Artifact: "unknown"})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown artifact"))
	}
}

// TestRunPath_ValidArtifact verifies RunPath prints a path for known artifact.
func TestRunPath_ValidArtifact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := config.RunPath(config.PathArgs{Dir: dir, Artifact: "readme"})
	g.Expect(err).ToNot(HaveOccurred())
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

// Mock file system for config tests
type mockConfigFS struct {
	files map[string]string // path -> content
}

func (m *mockConfigFS) FileExists(path string) bool {
	_, exists := m.files[path]
	return exists
}

func (m *mockConfigFS) ReadFile(path string) (string, error) {
	content, exists := m.files[path]
	if !exists {
		return "", &fileNotFoundError{path: path}
	}

	return content, nil
}
