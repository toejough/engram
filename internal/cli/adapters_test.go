package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/promote"
	regpkg "engram/internal/registry"
	reviewpkg "engram/internal/review"
)

func TestBuildEscalationMemories(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	classified := []reviewpkg.ClassifiedMemory{
		{
			Name:               "/m/leech.toml",
			Quadrant:           reviewpkg.Leech,
			EffectivenessScore: 0.2,
		},
		{
			Name:     "/m/working.toml",
			Quadrant: reviewpkg.Working,
		},
		{
			Name:               "/m/leech-no-stored.toml",
			Quadrant:           reviewpkg.Leech,
			EffectivenessScore: 0.1,
		},
	}

	memoryMap := map[string]*memory.Stored{
		"/m/leech.toml": {Content: "leech content"},
	}

	result := cli.ExportBuildEscalationMemories(classified, memoryMap)

	// Only leeches included; working is filtered out.
	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0].Path).To(Equal("/m/leech.toml"))
	g.Expect(result[0].Content).To(Equal("leech content"))
	g.Expect(result[1].Path).To(Equal("/m/leech-no-stored.toml"))
	g.Expect(result[1].Content).To(BeEmpty()) // nil stored → empty content

	// Verify maintain.EscalationMemory type is used.
	_ = result[0]
}

func TestBuildExtractor_AllTypes(t *testing.T) {
	t.Parallel()

	types := []string{"claude-md", "memory-md", "rule", "skill"}

	for _, sourceType := range types {
		t.Run(sourceType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			ext, err := cli.ExportBuildExtractor(sourceType, "test-path", "some content")
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(ext).NotTo(BeNil())
		})
	}

	t.Run("unknown type returns error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		_, err := cli.ExportBuildExtractor("bogus", "path", "content")
		g.Expect(err).To(HaveOccurred())
	})
}

func TestCliConfirmer_Confirm_AutoApprove(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	confirmer := cli.ExportNewCliConfirmer(&buf, strings.NewReader(""), true)

	approved, err := confirmer.Confirm("preview text")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(approved).To(BeTrue())
	g.Expect(buf.String()).To(ContainSubstring("Auto-confirmed"))
}

func TestCliConfirmer_Confirm_UserDecline(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	confirmer := cli.ExportNewCliConfirmer(&buf, strings.NewReader("n\n"), false)

	approved, err := confirmer.Confirm("preview text")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(approved).To(BeFalse())
}

func TestCliConfirmer_Confirm_UserInput(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	confirmer := cli.ExportNewCliConfirmer(&buf, strings.NewReader("y\n"), false)

	approved, err := confirmer.Confirm("preview text")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(approved).To(BeTrue())
}

func TestContentHashForRegistry(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	hash := cli.ExportContentHash("hello")
	g.Expect(hash).To(HaveLen(64)) // SHA-256 hex = 64 chars
}

func TestEvaluateRegistryAdapter_RecordEvaluation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	store := newTestStore(t)

	// Seed a registry entry so RecordEvaluation finds it.
	g.Expect(store.Register(regpkg.InstructionEntry{
		ID:         "mem-1",
		SourceType: "memory",
		SourcePath: "mem-1",
		Title:      "Test",
	})).To(Succeed())

	adapter := cli.ExportNewEvaluateRegistryAdapter(store)

	err := adapter.RecordEvaluation("mem-1", "followed")
	g.Expect(err).NotTo(HaveOccurred())
}

func TestLearnRegistryAdapter_RegisterMemory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	store := newTestStore(t)
	adapter := cli.ExportNewLearnRegistryAdapter(store)

	err := adapter.RegisterMemory(
		"/tmp/test.toml", "Test Memory", "content body", time.Now(),
	)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestLoadMemoryContent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	memPath := filepath.Join(memDir, "test-mem.toml")

	g.Expect(os.WriteFile(memPath, []byte(
		"title = \"Test Mem\"\nprinciple = \"be good\"\nupdated_at = \"2025-01-01T00:00:00Z\"\n",
	), 0o644)).To(Succeed())

	retriever := cli.ExportNewRetriever()

	memContent, err := cli.ExportLoadMemoryContent(retriever, memPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(memContent.Title).To(Equal("Test Mem"))
	g.Expect(memContent.Principle).To(Equal("be good"))
}

func TestLoadMemoryContent_ListError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	retriever := cli.ExportNewRetriever()

	_, err := cli.ExportLoadMemoryContent(
		retriever, "/nonexistent/dir/memories/test.toml",
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("loading memories"))
	}
}

func TestLoadMemoryContent_NotFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	g.Expect(os.WriteFile(
		filepath.Join(memDir, "exists.toml"),
		[]byte("title = \"Exists\"\nprinciple = \"test\"\nupdated_at = \"2025-01-01T00:00:00Z\"\n"),
		0o644,
	)).To(Succeed())

	retriever := cli.ExportNewRetriever()

	_, err := cli.ExportLoadMemoryContent(
		retriever, filepath.Join(memDir, "nonexistent.toml"),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("memory not found"))
	}
}

func TestLoadSkillContent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	skillPath := filepath.Join(dir, "test-skill.md")

	g.Expect(os.WriteFile(
		skillPath,
		[]byte("# My Skill Title\n\nBody here.\n"),
		0o644,
	)).To(Succeed())

	skillContent, err := cli.ExportLoadSkillContent(skillPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(skillContent.Title).To(Equal("My Skill Title"))
	g.Expect(skillContent.Content).To(ContainSubstring("Body here."))
}

func TestNoopTranscriptReader_ReadRecent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	reader := cli.ExportNewNoopTranscriptReader()

	result, err := reader.ReadRecent(10)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

func TestOsClaudeMDStore_ReadWrite(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	storePath := filepath.Join(t.TempDir(), "CLAUDE.md")
	store := cli.ExportNewOsClaudeMDStore(storePath)

	// Read non-existent file returns empty.
	content, err := store.Read()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(content).To(BeEmpty())

	// Write and read back.
	g.Expect(store.Write("# Test\nContent.")).To(Succeed())

	content, err = store.Read()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(content).To(Equal("# Test\nContent."))
}

func TestOsCreationLogReader_CreationTimes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()

	lines := `{"timestamp":"2025-06-01T12:00:00Z","title":"Memory A","tier":"A","filename":"memory-a.toml"}
{"timestamp":"2025-06-02T14:30:00Z","title":"Memory B","tier":"B","filename":"memory-b.toml"}
{"timestamp":"bad-timestamp","title":"Bad","tier":"C","filename":"bad.toml"}
`

	g.Expect(os.WriteFile(
		filepath.Join(dataDir, "creation-log.jsonl"),
		[]byte(lines),
		0o644,
	)).To(Succeed())

	reader := cli.ExportNewOsCreationLogReader(dataDir)

	result, err := reader.CreationTimes()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Two valid entries; bad timestamp skipped.
	g.Expect(result).To(HaveLen(2))
	g.Expect(result).To(HaveKey("memory-a.toml"))
	g.Expect(result).To(HaveKey("memory-b.toml"))
	g.Expect(result["memory-a.toml"].Year()).To(Equal(2025))
}

func TestOsCreationLogReader_MissingFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	reader := cli.ExportNewOsCreationLogReader(t.TempDir())

	result, err := reader.CreationTimes()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

// AggregateEvaluations: returns empty map when evaluations dir is missing.
func TestOsEvaluationsReader_AggregateEvaluations_MissingDir(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	reader := cli.ExportNewOsEvaluationsReader(t.TempDir())

	result, err := reader.AggregateEvaluations()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

// AggregateEvaluations: returns counters from evaluation JSONL files.
func TestOsEvaluationsReader_AggregateEvaluations_WithData(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	evalDir := filepath.Join(dataDir, "evaluations")

	g.Expect(os.MkdirAll(evalDir, 0o750)).To(Succeed())

	lines := `{"memory_path":"/m/a.toml","outcome":"followed","evaluated_at":"2025-01-01T00:00:00Z"}
{"memory_path":"/m/a.toml","outcome":"contradicted","evaluated_at":"2025-01-02T00:00:00Z"}
{"memory_path":"/m/b.toml","outcome":"ignored","evaluated_at":"2025-01-01T00:00:00Z"}
`

	g.Expect(os.WriteFile(
		filepath.Join(evalDir, "session.jsonl"),
		[]byte(lines),
		0o640,
	)).To(Succeed())

	reader := cli.ExportNewOsEvaluationsReader(dataDir)

	result, err := reader.AggregateEvaluations()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(2))
	g.Expect(result["/m/a.toml"].Followed).To(Equal(1))
	g.Expect(result["/m/a.toml"].Contradicted).To(Equal(1))
	g.Expect(result["/m/b.toml"].Ignored).To(Equal(1))
}

func TestOsMemoryLoader_LoadPrinciple_Found(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	g.Expect(os.WriteFile(
		filepath.Join(memDir, "test-mem.toml"),
		[]byte(
			"title = \"Test\"\nprinciple = \"always test\"\nupdated_at = \"2025-01-01T00:00:00Z\"\n",
		),
		0o644,
	)).To(Succeed())

	loader := cli.ExportNewOsMemoryLoader(dataDir)

	principle, err := loader.LoadPrinciple(context.Background(), "test-mem")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(principle).To(Equal("always test"))
}

func TestOsMemoryLoader_LoadPrinciple_NotFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	loader := cli.ExportNewOsMemoryLoader(dataDir)

	principle, err := loader.LoadPrinciple(context.Background(), "nonexistent")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(principle).To(BeEmpty())
}

func TestOsMemoryRemover_Remove(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	filePath := filepath.Join(t.TempDir(), "test.toml")
	g.Expect(os.WriteFile(filePath, []byte("data"), 0o644)).To(Succeed())

	remover := cli.ExportNewOsMemoryRemover()
	g.Expect(remover.Remove(filePath)).To(Succeed())

	// File should be gone.
	_, err := os.Stat(filePath)
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

func TestOsMemoryRemover_RemoveError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	remover := cli.ExportNewOsMemoryRemover()
	err := remover.Remove("/nonexistent/path/file.toml")
	g.Expect(err).To(HaveOccurred())
}

func TestOsRemindConfigReader_ReadConfig_Missing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	reader := cli.ExportNewOsRemindConfigReader(t.TempDir())

	config, err := reader.ReadConfig()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(config).To(BeNil())
}

func TestOsRemindConfigReader_ReadConfig_WithFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()

	tomlContent := `["*.go"]
instructions = ["mem-1", "mem-2"]

["*.py"]
instructions = ["mem-3"]
`

	g.Expect(os.WriteFile(
		filepath.Join(dataDir, "reminders.toml"),
		[]byte(tomlContent),
		0o644,
	)).To(Succeed())

	reader := cli.ExportNewOsRemindConfigReader(dataDir)

	config, err := reader.ReadConfig()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(config).To(HaveLen(2))
	g.Expect(config["*.go"]).To(ConsistOf("mem-1", "mem-2"))
	g.Expect(config["*.py"]).To(ConsistOf("mem-3"))
}

func TestOsSkillWriter_Write(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := filepath.Join(t.TempDir(), "skills")
	writer := cli.ExportNewOsSkillWriter(dir)

	writtenPath, err := writer.Write("my-skill", "# My Skill\nContent here.")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writtenPath).To(ContainSubstring("my-skill.md"))

	data, readErr := os.ReadFile(writtenPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(Equal("# My Skill\nContent here."))

	// Writing again should fail (already exists).
	_, err = writer.Write("my-skill", "duplicate")
	g.Expect(err).To(HaveOccurred())
}

func TestOsSurfacingLogReader_AggregateSurfacing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()

	// Write a surfacing log with two entries.
	logPath := filepath.Join(dataDir, "surfacing-log.jsonl")
	lines := `{"memory_path":"/m/a.toml","mode":"session-start","surfaced_at":"2025-01-01T00:00:00Z"}
{"memory_path":"/m/a.toml","mode":"prompt","surfaced_at":"2025-01-02T00:00:00Z"}
{"memory_path":"/m/b.toml","mode":"session-start","surfaced_at":"2025-01-01T00:00:00Z"}
`

	g.Expect(os.WriteFile(logPath, []byte(lines), 0o644)).To(Succeed())

	reader := cli.ExportNewOsSurfacingLogReader(dataDir)

	result, err := reader.AggregateSurfacing()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(2))
	g.Expect(result["/m/a.toml"].Count).To(Equal(2))
	g.Expect(result["/m/b.toml"].Count).To(Equal(1))
}

func TestParseRemindersToml_EmptyInput(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result, err := cli.ExportParseRemindersToml([]byte(""))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

func TestParseRemindersToml_InstructionsWithoutSection(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Instructions line without a preceding section header is ignored.
	input := []byte(`instructions = ["rule-1"]`)

	result, err := cli.ExportParseRemindersToml(input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

func TestParseRemindersToml_NoEqualsSign(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	input := []byte("[\"*.go\"]\ninstructions [\"rule-1\"]\n")

	result, err := cli.ExportParseRemindersToml(input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// The line without = is skipped; section has no entries.
	g.Expect(result).To(BeEmpty())
}

func TestParseRemindersToml_ValidInput(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	input := []byte(
		"# Comment line\n[\"*.go\"]\n" +
			"instructions = [\"rule-1\", \"rule-2\"]\n\n" +
			"[\"*.py\"]\ninstructions = [\"rule-3\"]\n",
	)

	result, err := cli.ExportParseRemindersToml(input)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(2))
	g.Expect(result["*.go"]).To(Equal([]string{"rule-1", "rule-2"}))
	g.Expect(result["*.py"]).To(Equal([]string{"rule-3"}))
}

// resolveSkillsDir: returns skills subdir when CLAUDE_PLUGIN_ROOT is set.
func TestResolveSkillsDir_Set(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewWithT(t)

	t.Setenv("CLAUDE_PLUGIN_ROOT", "/home/user/.claude/plugins/engram")

	result := cli.ExportResolveSkillsDir()
	g.Expect(result).To(Equal("/home/user/.claude/plugins/engram/skills"))
}

// resolveSkillsDir: returns empty when CLAUDE_PLUGIN_ROOT is unset.
func TestResolveSkillsDir_Unset(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewWithT(t)

	t.Setenv("CLAUDE_PLUGIN_ROOT", "")

	result := cli.ExportResolveSkillsDir()
	g.Expect(result).To(BeEmpty())
}

func TestStdinConfirmer_Apply(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	confirmer := cli.ExportNewStdinConfirmer(&buf, strings.NewReader("a\n"))

	approved, err := confirmer.Confirm("preview")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(approved).To(BeTrue())
}

func TestStdinConfirmer_Quit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	confirmer := cli.ExportNewStdinConfirmer(&buf, strings.NewReader("q\n"))

	_, err := confirmer.Confirm("preview")
	g.Expect(err).To(MatchError(maintain.ErrUserQuit))
}

func TestStdinConfirmer_Skip(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	confirmer := cli.ExportNewStdinConfirmer(&buf, strings.NewReader("s\n"))

	approved, err := confirmer.Confirm("preview")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(approved).To(BeFalse())
}

func TestTemplateClaudeMDGenerator_Generate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	gen := cli.ExportNewTemplateClaudeMDGenerator()

	result, err := gen.Generate(
		context.Background(),
		promote.SkillContent{Title: "My Skill", Content: "Skill body."},
		"skill:my-skill",
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("My Skill"))
}

func TestTemplateGenerator_Generate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	gen := cli.ExportNewTemplateGenerator()

	result, err := gen.Generate(context.Background(), promote.MemoryContent{
		Title:     "Test",
		Principle: "do the right thing",
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("Test"))
}

func TestTruncateTitle_Long(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	long := strings.Repeat("a", 50)
	result := cli.ExportTruncateTitle(long)
	// len() counts bytes; "…" is 3 bytes in UTF-8, so maxTitleLength-1 chars + 3 bytes.
	g.Expect(len(result)).To(BeNumerically("<", len(long)))
	g.Expect(result).To(HaveSuffix("…"))
}

func TestTruncateTitle_Short(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportTruncateTitle("Short")).To(Equal("Short"))
}

func newTestStore(t *testing.T) *regpkg.JSONLStore {
	t.Helper()

	registryPath := filepath.Join(t.TempDir(), "instruction-registry.jsonl")

	return regpkg.NewJSONLStore(
		registryPath,
		regpkg.WithReader(os.ReadFile),
		regpkg.WithWriter(func(path string, data []byte) error {
			const filePerms = 0o644
			return os.WriteFile(path, data, filePerms)
		}),
	)
}
