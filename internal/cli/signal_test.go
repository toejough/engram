package cli //nolint:testpackage // white-box tests for unexported signal CLI functions

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/effectiveness"
	"engram/internal/memory"
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

func TestFileMergeExecutor_RemoveError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	survivorPath := filepath.Join(dir, "survivor.toml")

	writeErr := os.WriteFile(survivorPath, []byte(`title = "t"
content = "c"
concepts = []
keywords = []
anti_pattern = ""
principle = "p"
updated_at = "2024-01-01T00:00:00Z"
`), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	survivor := &memory.Stored{
		Title:    "S",
		Content:  "c",
		FilePath: survivorPath,
	}
	absorbed := &memory.Stored{
		Title:    "A",
		Content:  "c",
		FilePath: "/nonexistent/absorbed.toml",
	}

	executor := &fileMergeExecutor{
		writer: newStoredMemoryWriter(),
		remove: func(_ string) error { return os.ErrPermission },
	}

	err := executor.Merge(t.Context(), survivor, absorbed)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("removing absorbed")))
}

func TestFileMergeExecutor_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	survivorPath := filepath.Join(dir, "survivor.toml")
	absorbedPath := filepath.Join(dir, "absorbed.toml")

	for _, p := range []string{survivorPath, absorbedPath} {
		writeErr := os.WriteFile(p, []byte(`title = "t"
content = "c"
concepts = []
keywords = []
anti_pattern = ""
principle = "p"
updated_at = "2024-01-01T00:00:00Z"
`), 0o644)
		g.Expect(writeErr).NotTo(HaveOccurred())
	}

	survivor := &memory.Stored{
		Title:     "Survivor",
		Content:   "content",
		Keywords:  []string{"kw1"},
		Concepts:  []string{"concept1"},
		Principle: "short",
		FilePath:  survivorPath,
	}
	absorbed := &memory.Stored{
		Title:     "Absorbed",
		Content:   "content2",
		Keywords:  []string{"kw1", "kw2"},
		Concepts:  []string{"concept2"},
		Principle: "longer principle",
		FilePath:  absorbedPath,
	}

	executor := &fileMergeExecutor{
		writer: newStoredMemoryWriter(),
		remove: os.Remove,
	}

	err := executor.Merge(t.Context(), survivor, absorbed)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Survivor should have merged keywords, concepts, and longer principle.
	g.Expect(survivor.Keywords).To(ConsistOf("kw1", "kw2"))
	g.Expect(survivor.Concepts).To(ConsistOf("concept1", "concept2"))
	g.Expect(survivor.Principle).To(Equal("longer principle"))

	// Absorbed file should be removed.
	_, statErr := os.Stat(absorbedPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

func TestFileMergeExecutor_WriteError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	survivor := &memory.Stored{
		Title:    "S",
		Content:  "c",
		FilePath: "/nonexistent/dir/survivor.toml",
	}
	absorbed := &memory.Stored{
		Title:    "A",
		Content:  "c",
		FilePath: "/nonexistent/dir/absorbed.toml",
	}

	executor := &fileMergeExecutor{
		writer: newStoredMemoryWriter(),
		remove: os.Remove,
	}

	err := executor.Merge(t.Context(), survivor, absorbed)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("writing survivor")))
}

func TestKeepLongerPrinciple(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	survivor := &memory.Stored{Principle: "short"}
	absorbed := &memory.Stored{Principle: "much longer principle"}

	keepLongerPrinciple(survivor, absorbed)
	g.Expect(survivor.Principle).To(Equal("much longer principle"))
}

func TestKeepLongerPrinciple_SurvivorLonger(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	survivor := &memory.Stored{Principle: "already the longer one"}
	absorbed := &memory.Stored{Principle: "short"}

	keepLongerPrinciple(survivor, absorbed)
	g.Expect(survivor.Principle).To(Equal("already the longer one"))
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

func TestRegistryUpdaterAdapterUpdateContentHash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	reg := openRegistry(dataDir)
	adapter := &registryUpdaterAdapter{reg: reg, dataDir: dataDir}

	// UpdateContentHash is a no-op stub; it should always return nil.
	err := adapter.UpdateContentHash("any-id", "any-hash")
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRegistryUpdaterAdapter_RelIDEmptyDataDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	adapter := &registryUpdaterAdapter{dataDir: ""}

	g.Expect(adapter.relID("memories/test.toml")).To(Equal("memories/test.toml"))
}

func TestRegistryUpdaterAdapter_RelIDRelativePath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	adapter := &registryUpdaterAdapter{dataDir: "/data"}

	g.Expect(adapter.relID("/data/memories/test.toml")).To(Equal("memories/test.toml"))
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

func TestUnionConcepts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	survivor := &memory.Stored{Concepts: []string{"DI", "testing"}}
	absorbed := &memory.Stored{Concepts: []string{"di", "refactoring"}}

	unionConcepts(survivor, absorbed)
	g.Expect(survivor.Concepts).To(ConsistOf("DI", "testing", "refactoring"))
}

func TestUnionKeywords(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	survivor := &memory.Stored{Keywords: []string{"alpha", "beta"}}
	absorbed := &memory.Stored{Keywords: []string{"Beta", "gamma"}}

	unionKeywords(survivor, absorbed)
	g.Expect(survivor.Keywords).To(ConsistOf("alpha", "beta", "gamma"))
}
