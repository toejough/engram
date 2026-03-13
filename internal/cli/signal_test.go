package cli //nolint:testpackage // white-box tests for unexported signal CLI functions

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/promote"
	regpkg "engram/internal/registry"
	"engram/internal/signal"
)

func TestClaudeMDScannerAdapter_DemotionCandidatesError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &errorRegistry{err: os.ErrPermission}
	claudeMDPromoter := &promote.ClaudeMDPromoter{Registry: reg}
	adapter := &claudeMDScannerAdapter{promoter: claudeMDPromoter}

	_, err := adapter.DemotionCandidates()
	g.Expect(err).To(HaveOccurred())
}

func TestClaudeMDScannerAdapter_PromotionCandidatesError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &errorRegistry{err: os.ErrPermission}
	claudeMDPromoter := &promote.ClaudeMDPromoter{Registry: reg}
	adapter := &claudeMDScannerAdapter{promoter: claudeMDPromoter}

	_, err := adapter.PromotionCandidates(1)
	g.Expect(err).To(HaveOccurred())
}

func TestPromotionScannerAdapter_CandidatesError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &errorRegistry{err: os.ErrPermission}
	promoter := &promote.Promoter{Registry: reg}
	adapter := &promotionScannerAdapter{promoter: promoter}

	_, err := adapter.Candidates(1)
	g.Expect(err).To(HaveOccurred())
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
	adapter := &registryUpdaterAdapter{reg: reg}

	// UpdateContentHash is a no-op stub; it should always return nil.
	err := adapter.UpdateContentHash("any-id", "any-hash")
	g.Expect(err).NotTo(HaveOccurred())
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

	registryPath := filepath.Join(dataDir, registryFilename)
	regWriteErr := os.WriteFile(registryPath, []byte(""), 0o644)
	g.Expect(regWriteErr).NotTo(HaveOccurred())

	if regWriteErr != nil {
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
`
	writeErr := os.WriteFile(memPath, []byte(tomlContent), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	// Create registry file with an entry for the memory.
	registryPath := filepath.Join(dataDir, registryFilename)
	regEntry := `{"id":"` + memPath + `","source_type":"memory","enforcement_level":"advisory"}`
	regWriteErr := os.WriteFile(registryPath, []byte(regEntry+"\n"), 0o644)
	g.Expect(regWriteErr).NotTo(HaveOccurred())

	if regWriteErr != nil {
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
	registryPath := filepath.Join(dataDir, registryFilename)
	_ = os.WriteFile(registryPath, []byte(""), 0o644)

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

	// Create registry file.
	registryPath := filepath.Join(dataDir, registryFilename)
	regWriteErr := os.WriteFile(registryPath, []byte(""), 0o644)
	g.Expect(regWriteErr).NotTo(HaveOccurred())

	if regWriteErr != nil {
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

	registryPath := filepath.Join(dataDir, registryFilename)
	regWriteErr := os.WriteFile(registryPath, []byte(""), 0o644)
	g.Expect(regWriteErr).NotTo(HaveOccurred())

	if regWriteErr != nil {
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

func TestRunSignalDetect_AggregateError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	// Create "evaluations" as a file (not dir) so Aggregate() fails.
	evalFile := filepath.Join(dataDir, "evaluations")
	writeErr := os.WriteFile(evalFile, []byte("not a dir"), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	err := runSignalDetect([]string{"--data-dir", dataDir})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("aggregating effectiveness")))
}

func TestRunSignalDetect_EmptyDataDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	// Create evaluations dir so effectiveness.Aggregate doesn't error.
	err := os.MkdirAll(filepath.Join(dataDir, "evaluations"), 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Create memories dir so tracking map doesn't fail.
	err = os.MkdirAll(filepath.Join(dataDir, "memories"), 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Create registry file so openRegistry has something to load.
	registryPath := filepath.Join(dataDir, registryFilename)
	writeErr := os.WriteFile(registryPath, []byte(""), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	err = runSignalDetect([]string{"--data-dir", dataDir})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunSignalDetect_InvalidFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := runSignalDetect([]string{"--unknown-flag"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("signal-detect")))
}

func TestRunSignalDetect_MissingDataDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := runSignalDetect([]string{})
	g.Expect(err).To(MatchError(ContainSubstring("signal-detect")))
}

func TestRunSignalDetect_WithPreexistingQueue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	err := os.MkdirAll(filepath.Join(dataDir, "evaluations"), 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = os.MkdirAll(filepath.Join(dataDir, "memories"), 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	registryPath := filepath.Join(dataDir, registryFilename)
	writeErr := os.WriteFile(registryPath, []byte(""), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	// Write a pre-existing queue entry so Prune executes the existsCheck lambda.
	queuePath := filepath.Join(dataDir, signalQueueFilename)
	store := signal.NewQueueStore()
	preExisting := []signal.Signal{
		{
			Type:       signal.TypeMaintain,
			SourceID:   "/nonexistent/memory.toml", // will be pruned (file doesn't exist)
			SignalKind: signal.KindNoiseRemoval,
			Summary:    "old signal",
			DetectedAt: time.Now(),
		},
	}

	appendErr := store.Append(preExisting, queuePath)
	g.Expect(appendErr).NotTo(HaveOccurred())

	if appendErr != nil {
		return
	}

	err = runSignalDetect([]string{"--data-dir", dataDir})
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunSignalSurface_EmptyQueue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var buf bytes.Buffer

	err := runSignalSurface([]string{"--data-dir", dataDir}, &buf)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(BeEmpty())
}

func TestRunSignalSurface_MissingDataDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := runSignalSurface([]string{}, &bytes.Buffer{})
	g.Expect(err).To(MatchError(ContainSubstring("signal-surface")))
}

func TestRunSignalSurface_TextFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	memPath := filepath.Join(memDir, "test.toml")
	tomlContent := "title = \"Test\"\ncontent = \"content\"\n" +
		"concepts = []\nkeywords = [\"kw\"]\nanti_pattern = \"\"\n" +
		"principle = \"principle\"\nupdated_at = \"2024-01-01T00:00:00Z\"\n"
	writeErr := os.WriteFile(memPath, []byte(tomlContent), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	queuePath := filepath.Join(dataDir, signalQueueFilename)
	store := signal.NewQueueStore()
	signals := []signal.Signal{
		{
			Type:       signal.TypeMaintain,
			SourceID:   memPath,
			SignalKind: signal.KindLeechRewrite,
			Summary:    "Leech signal",
			DetectedAt: time.Now(),
		},
	}

	appendErr := store.Append(signals, queuePath)
	g.Expect(appendErr).NotTo(HaveOccurred())

	if appendErr != nil {
		return
	}

	var buf bytes.Buffer

	err = runSignalSurface([]string{"--data-dir", dataDir}, &buf)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).NotTo(BeEmpty())
}

func TestRunSignalSurface_WithSignalsJSONFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	memPath := filepath.Join(memDir, "test.toml")
	tomlContent := "title = \"Test\"\ncontent = \"content\"\n" +
		"concepts = []\nkeywords = [\"kw\"]\nanti_pattern = \"\"\n" +
		"principle = \"principle\"\nupdated_at = \"2024-01-01T00:00:00Z\"\n"
	writeErr := os.WriteFile(memPath, []byte(tomlContent), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	queuePath := filepath.Join(dataDir, signalQueueFilename)
	store := signal.NewQueueStore()
	signals := []signal.Signal{
		{
			Type:       signal.TypeMaintain,
			SourceID:   memPath,
			SignalKind: signal.KindNoiseRemoval,
			Summary:    "Test signal",
			DetectedAt: time.Now(),
		},
	}

	appendErr := store.Append(signals, queuePath)
	g.Expect(appendErr).NotTo(HaveOccurred())

	if appendErr != nil {
		return
	}

	var buf bytes.Buffer

	err = runSignalSurface([]string{"--data-dir", dataDir, "--format", "json"}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).NotTo(BeEmpty())

	var output struct {
		Summary string `json:"summary"`
		Context string `json:"context"`
	}

	decodeErr := json.NewDecoder(&buf).Decode(&output)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	g.Expect(output.Summary).To(ContainSubstring("signal"))
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

// errorRegistry implements promote.RegistryReader, always returning an error on List.
type errorRegistry struct {
	err error
}

func (r *errorRegistry) Get(_ string) (*regpkg.InstructionEntry, error) {
	return nil, r.err
}

func (r *errorRegistry) List() ([]regpkg.InstructionEntry, error) {
	return nil, r.err
}
