package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/maintain"
	"engram/internal/memory"
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

func TestContentHash(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	hash := cli.ExportContentHash("hello")
	g.Expect(hash).To(HaveLen(64)) // SHA-256 hex = 64 chars
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

// TestRunEvaluate_WithDataDir covers the runEvaluate wiring branch that injects
// WithEvaluationRecorder when --data-dir is provided.
func TestRunEvaluate_WithDataDir(t *testing.T) {
	g := NewWithT(t)

	// Unset token so RunEvaluate returns nil after logging a skip — no real LLM call.
	// This exercises the dataDir != "" branch and the WithEvaluationRecorder closure.
	t.Setenv("ENGRAM_API_TOKEN", "")

	var stdout, stderr bytes.Buffer

	err := cli.ExportRunEvaluate(
		[]string{"--data-dir", t.TempDir()},
		&stdout, &stderr, strings.NewReader("transcript"),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stderr.String()).To(ContainSubstring("no API token"))
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

func newTestStore(t *testing.T) *regpkg.TOMLDirectoryStore {
	t.Helper()

	return regpkg.NewTOMLDirectoryStore(t.TempDir())
}
