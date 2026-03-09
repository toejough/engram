package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/promote"
	regpkg "engram/internal/registry"
	"engram/internal/retrieve"
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

	result := buildEscalationMemories(classified, memoryMap)

	// Only leeches included; working is filtered out.
	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0].Path).To(Equal("/m/leech.toml"))
	g.Expect(result[0].Content).To(Equal("leech content"))
	g.Expect(result[1].Path).To(Equal("/m/leech-no-stored.toml"))
	g.Expect(result[1].Content).To(BeEmpty()) // nil stored → empty content

	// Verify maintain.EscalationMemory type is used.
	var _ = result[0]
}

func TestBuildExtractor_AllTypes(t *testing.T) {
	t.Parallel()

	types := []string{"claude-md", "memory-md", "rule", "skill"}

	for _, st := range types {
		t.Run(st, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			ext, err := buildExtractor(st, "test-path", "some content")
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

		_, err := buildExtractor("bogus", "path", "content")
		g.Expect(err).To(HaveOccurred())
	})
}

func TestCliConfirmer_Confirm_AutoApprove(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	confirmer := &cliConfirmer{
		stdout:      &buf,
		stdin:       strings.NewReader(""),
		autoConfirm: true,
	}

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

	confirmer := &cliConfirmer{
		stdout:      &buf,
		stdin:       strings.NewReader("n\n"),
		autoConfirm: false,
	}

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

	confirmer := &cliConfirmer{
		stdout:      &buf,
		stdin:       strings.NewReader("y\n"),
		autoConfirm: false,
	}

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

	hash := contentHashForRegistry("hello")
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

	adapter := &evaluateRegistryAdapter{reg: store}

	err := adapter.RecordEvaluation("mem-1", "followed")
	g.Expect(err).NotTo(HaveOccurred())
}

func TestLearnRegistryAdapter_RegisterMemory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	store := newTestStore(t)
	adapter := &learnRegistryAdapter{
		reg: store,
		now: time.Now,
	}

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

	ret := retrieve.New()

	mc, err := loadMemoryContent(ret, memPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(mc.Title).To(Equal("Test Mem"))
	g.Expect(mc.Principle).To(Equal("be good"))
}

func TestLoadSkillContent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	skillPath := filepath.Join(dir, "test-skill.md")

	g.Expect(os.WriteFile(skillPath, []byte("# My Skill Title\n\nBody here.\n"), 0o644)).
		To(Succeed())

	sc, err := loadSkillContent(skillPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(sc.Title).To(Equal("My Skill Title"))
	g.Expect(sc.Content).To(ContainSubstring("Body here."))
}

func TestOsClaudeMDStore_ReadWrite(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "CLAUDE.md")
	store := &osClaudeMDStore{path: path}

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

	reader := &osCreationLogReader{dataDir: dataDir}

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

	reader := &osCreationLogReader{dataDir: t.TempDir()}

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

	reader := &osEvaluationsReader{dataDir: t.TempDir()}

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

	reader := &osEvaluationsReader{dataDir: dataDir}

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

func TestOsMemoryRemover_Remove(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "test.toml")
	g.Expect(os.WriteFile(path, []byte("data"), 0o644)).To(Succeed())

	remover := &osMemoryRemover{}
	g.Expect(remover.Remove(path)).To(Succeed())

	// File should be gone.
	_, err := os.Stat(path)
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

func TestOsMemoryRemover_RemoveError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	remover := &osMemoryRemover{}
	err := remover.Remove("/nonexistent/path/file.toml")
	g.Expect(err).To(HaveOccurred())
}

func TestOsSkillWriter_Write(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := filepath.Join(t.TempDir(), "skills")
	writer := &osSkillWriter{dir: dir}

	path, err := writer.Write("my-skill", "# My Skill\nContent here.")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(path).To(ContainSubstring("my-skill.md"))

	data, readErr := os.ReadFile(path)
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

	reader := &osSurfacingLogReader{dataDir: dataDir}

	result, err := reader.AggregateSurfacing()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(2))
	g.Expect(result["/m/a.toml"].Count).To(Equal(2))
	g.Expect(result["/m/b.toml"].Count).To(Equal(1))
}

// resolveSkillsDir: returns skills subdir when CLAUDE_PLUGIN_ROOT is set.
func TestResolveSkillsDir_Set(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewWithT(t)

	t.Setenv("CLAUDE_PLUGIN_ROOT", "/home/user/.claude/plugins/engram")

	result := resolveSkillsDir()
	g.Expect(result).To(Equal("/home/user/.claude/plugins/engram/skills"))
}

// resolveSkillsDir: returns empty when CLAUDE_PLUGIN_ROOT is unset.
func TestResolveSkillsDir_Unset(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewWithT(t)

	t.Setenv("CLAUDE_PLUGIN_ROOT", "")

	result := resolveSkillsDir()
	g.Expect(result).To(BeEmpty())
}

func TestStdinConfirmer_Apply(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	confirmer := &stdinConfirmer{
		stdout: &buf,
		stdin:  strings.NewReader("a\n"),
	}

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

	confirmer := &stdinConfirmer{
		stdout: &buf,
		stdin:  strings.NewReader("q\n"),
	}

	_, err := confirmer.Confirm("preview")
	g.Expect(err).To(MatchError(maintain.ErrUserQuit))
}

func TestStdinConfirmer_Skip(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	confirmer := &stdinConfirmer{
		stdout: &buf,
		stdin:  strings.NewReader("s\n"),
	}

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

	gen := &templateClaudeMDGenerator{}

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

	gen := &templateGenerator{}

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
	result := truncateTitle(long)
	// len() counts bytes; "…" is 3 bytes in UTF-8, so maxTitleLength-1 chars + 3 bytes.
	g.Expect(len(result)).To(BeNumerically("<", len(long)))
	g.Expect(result).To(HaveSuffix("…"))
}

func TestTruncateTitle_Short(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(truncateTitle("Short")).To(Equal("Short"))
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
