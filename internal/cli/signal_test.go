package cli //nolint:testpackage // white-box tests for unexported signal CLI functions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/effectiveness"
	"engram/internal/memory"
	"engram/internal/retrieve"
	"engram/internal/signal"
)

func TestEffectivenessReaderAdapter_Found(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	adapter := &effectivenessReaderAdapter{
		stats: map[string]effectiveness.Stat{
			"memories/test.toml": {EffectivenessScore: 0.75},
		},
	}

	score, hasData, err := adapter.EffectivenessScore("memories/test.toml")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(hasData).To(BeTrue())
	g.Expect(score).To(Equal(0.75))
}

func TestEffectivenessReaderAdapter_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	adapter := &effectivenessReaderAdapter{
		stats: map[string]effectiveness.Stat{},
	}

	score, hasData, err := adapter.EffectivenessScore("nonexistent.toml")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(hasData).To(BeFalse())
	g.Expect(score).To(Equal(0.0))
}

func TestFuncEnforcementApplier_SetEnforcementLevel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedPath, capturedLevel, capturedReason string

	applier := &funcEnforcementApplier{
		fn: func(path, level, reason string) error {
			capturedPath = path
			capturedLevel = level
			capturedReason = reason

			return nil
		},
	}

	err := applier.SetEnforcementLevel("memories/test.toml", "advisory", "test reason")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(capturedPath).To(Equal("memories/test.toml"))
	g.Expect(capturedLevel).To(Equal("advisory"))
	g.Expect(capturedReason).To(Equal("test reason"))
}

// TestIsCoveredBySource covers both the no-keywords early return and the keyword-match path.
func TestIsCoveredBySource(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	t.Run("returns false when memory has no keywords", func(t *testing.T) {
		t.Parallel()

		checker := &sourceCrossRefChecker{
			sources:  []crossRefSourceEntry{{id: "CLAUDE.md", text: "use targ"}},
			keywords: map[string][]string{"/memory.toml": {}},
		}

		covered, _, err := checker.IsCoveredBySource("/memory.toml")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(covered).To(BeFalse())
	})

	t.Run("returns true when keyword appears in source", func(t *testing.T) {
		t.Parallel()

		checker := &sourceCrossRefChecker{
			sources:  []crossRefSourceEntry{{id: "CLAUDE.md", text: "use targ for builds"}},
			keywords: map[string][]string{"/memory.toml": {"targ", "build"}},
		}

		covered, source, err := checker.IsCoveredBySource("/memory.toml")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(covered).To(BeTrue())
		g.Expect(source).To(Equal("CLAUDE.md"))
	})
}

// TestLoadCrossRefSources covers the rules directory path.
func TestLoadCrossRefSources(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	claudeDir := t.TempDir()
	rulesDir := filepath.Join(claudeDir, "rules")
	g.Expect(os.MkdirAll(rulesDir, 0o755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# Rules\n\n- use targ\n"), 0o640)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(rulesDir, "go.md"), []byte("Use gofmt"), 0o640)).To(Succeed())

	sources := loadCrossRefSources(claudeDir)

	g.Expect(sources).NotTo(BeEmpty())

	ids := make([]string, 0, len(sources))
	for _, s := range sources {
		ids = append(ids, s.id)
	}

	g.Expect(ids).To(ContainElement("go.md"))

	// Verify CLAUDE.md bullet entries also appear in sources (I-3).
	texts := make([]string, 0, len(sources))
	for _, s := range sources {
		texts = append(texts, s.text)
	}

	g.Expect(texts).To(ContainElement(ContainSubstring("targ")),
		"CLAUDE.md bullet with 'targ' must appear in sources")
}

func TestReadStoredMemory_DecodeError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	writeErr := os.WriteFile(path, []byte("not valid toml = = ="), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	_, err := readStoredMemory(path)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("decoding memory TOML")))
}

func TestReadStoredMemory_FileNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := readStoredMemory("/nonexistent/path/memory.toml")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("reading memory file")))
}

func TestRunApplyProposal_BroadenKeywordsAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	memPath := filepath.Join(memoriesDir, "broaden.toml")
	tomlContent := `title = "Test"
content = "content"
concepts = []
keywords = ["original"]
anti_pattern = ""
principle = "principle"
updated_at = "2024-01-01T00:00:00Z"
`
	writeErr := os.WriteFile(memPath, []byte(tomlContent), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var buf bytes.Buffer

	err = runApplyProposal([]string{
		"--data-dir", dataDir,
		"--action", "broaden_keywords",
		"--memory", memPath,
		"--keywords", "new-kw,another-kw",
	}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result signal.ApplyResult

	decodeErr := json.NewDecoder(&buf).Decode(&result)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	g.Expect(result.Success).To(BeTrue())
}

func TestRunApplyProposal_EscalateAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	// Create a dummy memory TOML file.
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	memPath := filepath.Join(memoriesDir, "test.toml")
	tomlContent := `title = "Test"
content = "content"
concepts = []
keywords = []
anti_pattern = ""
principle = "principle"
updated_at = "2024-01-01T00:00:00Z"
enforcement_level = "advisory"
`
	writeErr := os.WriteFile(memPath, []byte(tomlContent), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var buf bytes.Buffer

	err = runApplyProposal([]string{
		"--data-dir", dataDir,
		"--action", "escalate",
		"--memory", memPath,
		"--level", "2",
	}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result signal.ApplyResult

	decodeErr := json.NewDecoder(&buf).Decode(&result)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	g.Expect(result.Success).To(BeTrue())
}

func TestRunApplyProposal_InvalidFieldsJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	err := runApplyProposal([]string{
		"--data-dir", dataDir,
		"--action", "rewrite",
		"--memory", "some/path.toml",
		"--fields", `{invalid json`,
	}, &bytes.Buffer{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("parsing --fields")))
}

func TestRunApplyProposal_MissingFlags(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := runApplyProposal([]string{}, &bytes.Buffer{})
	g.Expect(err).To(MatchError(ContainSubstring("apply-proposal")))
}

func TestRunApplyProposal_RemoveAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	// Create a dummy memory TOML file.
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	memPath := filepath.Join(memoriesDir, "to-remove.toml")
	tomlContent := `title = "Test"
content = "content"
concepts = []
keywords = []
anti_pattern = ""
principle = "principle"
updated_at = "2024-01-01T00:00:00Z"
`
	writeErr := os.WriteFile(memPath, []byte(tomlContent), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var buf bytes.Buffer

	err = runApplyProposal([]string{
		"--data-dir", dataDir,
		"--action", "remove",
		"--memory", memPath,
	}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result signal.ApplyResult

	decodeErr := json.NewDecoder(&buf).Decode(&result)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	g.Expect(result.Success).To(BeTrue())

	// File should be gone.
	_, statErr := os.Stat(memPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

func TestRunApplyProposal_RewriteAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	memPath := filepath.Join(memoriesDir, "rewrite.toml")
	tomlContent := `title = "Old Title"
content = "old content"
concepts = []
keywords = []
anti_pattern = ""
principle = "principle"
updated_at = "2024-01-01T00:00:00Z"
`
	writeErr := os.WriteFile(memPath, []byte(tomlContent), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var buf bytes.Buffer

	err = runApplyProposal([]string{
		"--data-dir", dataDir,
		"--action", "rewrite",
		"--memory", memPath,
		"--fields", `{"title":"New Title","content":"new content"}`,
	}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result signal.ApplyResult

	decodeErr := json.NewDecoder(&buf).Decode(&result)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	g.Expect(result.Success).To(BeTrue())
}

// TestRunMaintainDryRun_EncodeError covers the encode-error branch in runMaintainDryRun.
func TestRunMaintainDryRun_EncodeError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	// Plan returns an empty list for an empty memories dir — encode then fails on failWriter.
	g.Expect(os.MkdirAll(filepath.Join(dataDir, "memories"), 0o755)).To(Succeed())

	err := runMaintainDryRun(context.Background(), retrieve.New(), dataDir, &failWriter{})
	g.Expect(err).To(MatchError(ContainSubstring("encoding plan")))
}

func TestStoredMemoryWriter_Write_CloseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "memory.toml")
	stored := &memory.Stored{Title: "Test", Content: "c"}

	writer := newStoredMemoryWriter()
	writer.createTemp = func(d, pattern string) (*os.File, error) {
		f, err := os.CreateTemp(d, pattern)
		if err != nil {
			return nil, err
		}
		// Close it immediately so the second Close() in Write() errors.
		_ = f.Close()

		return f, nil
	}

	err := writer.Write(path, stored)
	g.Expect(err).To(HaveOccurred())
}

func TestStoredMemoryWriter_Write_CreateTempError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stored := &memory.Stored{Title: "Test", Content: "c"}

	writer := newStoredMemoryWriter()
	writer.createTemp = func(_, _ string) (*os.File, error) {
		return nil, os.ErrPermission
	}

	err := writer.Write("/any/path.toml", stored)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("creating temp file")))
}

// TestStoredMemoryWriter_Write_PreservesAllCounters verifies that Write preserves all
// counter fields from memory.Stored, not just SurfacedCount (#353).
func TestStoredMemoryWriter_Write_PreservesAllCounters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "counter-memory.toml")

	stored := &memory.Stored{
		Title:             "Counter Test",
		Content:           "content",
		SurfacedCount:     5,
		FollowedCount:     3,
		ContradictedCount: 2,
		IgnoredCount:      1,
		IrrelevantCount:   4,
	}

	writer := newStoredMemoryWriter()
	err := writer.Write(path, stored)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	roundTripped, readErr := readStoredMemory(path)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(roundTripped.SurfacedCount).To(Equal(5))
	g.Expect(roundTripped.FollowedCount).To(Equal(3))
	g.Expect(roundTripped.ContradictedCount).To(Equal(2))
	g.Expect(roundTripped.IgnoredCount).To(Equal(1))
	g.Expect(roundTripped.IrrelevantCount).To(Equal(4))
}

func TestStoredMemoryWriter_Write_RenameError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "memory.toml")
	stored := &memory.Stored{Title: "Test", Content: "c"}

	writer := newStoredMemoryWriter()
	writer.rename = func(_, _ string) error {
		return os.ErrPermission
	}

	err := writer.Write(path, stored)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("renaming temp file")))
}

func TestStoredMemoryWriter_Write_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test-memory.toml")
	tomlContent := `title = "Initial"
content = "initial content"
concepts = []
keywords = ["kw"]
anti_pattern = ""
principle = "some principle"
updated_at = "2024-01-01T00:00:00Z"
`
	writeErr := os.WriteFile(path, []byte(tomlContent), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	stored := &memory.Stored{
		Title:     "Updated Title",
		Content:   "updated content",
		Keywords:  []string{"kw1", "kw2"},
		Principle: "updated principle",
	}

	writer := newStoredMemoryWriter()
	err := writer.Write(path, stored)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("Updated Title"))
}

// TestToRelID_ErrorFallback covers the defensive fallback when filepath.Rel fails.
// filepath.Rel fails when basepath is relative and targpath is absolute.
func TestToRelID_ErrorFallback(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	absPath := "/absolute/path/memory.toml"
	result := toRelID("relative/base", absPath)

	g.Expect(result).To(Equal(absPath))
}

// TestToRelID_HappyPath covers the normal case where the path is under dataDir.
func TestToRelID_HappyPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result := toRelID("/data", "/data/memories/test.toml")
	g.Expect(result).To(Equal("memories/test.toml"))
}

// failWriter always returns an error on Write.
type failWriter struct{}

func (failWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}
