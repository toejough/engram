package track_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/track"
)

// T-76: No tracking fields → after RecordSurfacing, file has surfaced_count=1
// and all original fields preserved.
func TestT76_RecordSurfacingNoExistingTracking(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "memory.toml")
	err := os.WriteFile(filePath, []byte(baseTOML), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	fixedNow := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	recorder := track.NewRecorder(track.WithNow(func() time.Time { return fixedNow }))

	memories := []*memory.Stored{
		{FilePath: filePath},
	}

	err = recorder.RecordSurfacing(context.Background(), memories, "session-start")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Read back and verify.
	data, err := os.ReadFile(filePath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var record fullTOMLRecord

	_, err = toml.Decode(string(data), &record)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Tracking fields updated.
	g.Expect(record.SurfacedCount).To(Equal(1))
	g.Expect(record.LastSurfaced).To(Equal("2026-03-05T12:00:00Z"))
	g.Expect(record.SurfacingContexts).To(Equal([]string{"session-start"}))

	// Original fields preserved.
	g.Expect(record.Title).To(Equal("Test Memory"))
	g.Expect(record.Content).To(Equal("some content"))
	g.Expect(record.ObservationType).To(Equal("correction"))
	g.Expect(record.Concepts).To(Equal([]string{"testing"}))
	g.Expect(record.Keywords).To(Equal([]string{"test"}))
	g.Expect(record.Principle).To(Equal("test principle"))
	g.Expect(record.AntiPattern).To(Equal("test anti-pattern"))
	g.Expect(record.Rationale).To(Equal("test rationale"))
	g.Expect(record.Confidence).To(Equal("B"))
	g.Expect(record.CreatedAt).To(Equal("2025-01-01T00:00:00Z"))
	g.Expect(record.UpdatedAt).To(Equal("2025-06-01T00:00:00Z"))
}

// T-77: Existing tracking fields → count incremented, contexts appended.
func TestT77_RecordSurfacingWithExistingTracking(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "memory.toml")
	existingTOML := baseTOML + `surfaced_count = 3
last_surfaced = "2026-02-01T00:00:00Z"
surfacing_contexts = ["prompt", "tool", "session-start"]
`
	err := os.WriteFile(filePath, []byte(existingTOML), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	fixedNow := time.Date(2026, 3, 5, 14, 0, 0, 0, time.UTC)
	recorder := track.NewRecorder(track.WithNow(func() time.Time { return fixedNow }))

	memories := []*memory.Stored{
		{
			FilePath:          filePath,
			SurfacedCount:     3,
			SurfacingContexts: []string{"prompt", "tool", "session-start"},
		},
	}

	err = recorder.RecordSurfacing(context.Background(), memories, "tool")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, err := os.ReadFile(filePath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var record fullTOMLRecord

	_, err = toml.Decode(string(data), &record)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(record.SurfacedCount).To(Equal(4))
	g.Expect(record.LastSurfaced).To(Equal("2026-03-05T14:00:00Z"))
	g.Expect(record.SurfacingContexts).To(Equal(
		[]string{"prompt", "tool", "session-start", "tool"},
	))
}

// T-78: Two memories, first has unreadable path → first skipped, second updated.
func TestT78_PartialFailureContinues(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	// Second memory has a valid file.
	validPath := filepath.Join(dir, "valid.toml")
	err := os.WriteFile(validPath, []byte(baseTOML), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	fixedNow := time.Date(2026, 3, 5, 16, 0, 0, 0, time.UTC)

	failRead := func(path string) ([]byte, error) {
		if path == "/nonexistent/bad.toml" {
			return nil, errors.New("file not found")
		}

		return os.ReadFile(path)
	}

	recorder := track.NewRecorder(
		track.WithNow(func() time.Time { return fixedNow }),
		track.WithReadFile(failRead),
		track.WithCreateTemp(os.CreateTemp),
		track.WithRename(os.Rename),
		track.WithRemove(os.Remove),
	)

	memories := []*memory.Stored{
		{FilePath: "/nonexistent/bad.toml"},
		{FilePath: validPath},
	}

	err = recorder.RecordSurfacing(context.Background(), memories, "prompt")

	// Should return an error for the first memory but still process the second.
	g.Expect(err).To(HaveOccurred())

	// Second file should be updated.
	data, readErr := os.ReadFile(validPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var record fullTOMLRecord

	_, decodeErr := toml.Decode(string(data), &record)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(record.SurfacedCount).To(Equal(1))
	g.Expect(record.LastSurfaced).To(Equal("2026-03-05T16:00:00Z"))
	g.Expect(record.SurfacingContexts).To(Equal([]string{"prompt"}))
}

// TestWriteAtomicCreateTempFailure verifies writeAtomic error path when
// createTemp fails.
func TestWriteAtomicCreateTempFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "memory.toml")
	err := os.WriteFile(filePath, []byte(baseTOML), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	failCreateTemp := func(_, _ string) (*os.File, error) {
		return nil, errors.New("disk full")
	}

	recorder := track.NewRecorder(
		track.WithCreateTemp(failCreateTemp),
	)

	memories := []*memory.Stored{
		{FilePath: filePath},
	}

	err = recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("disk full"))
}

// TestWriteAtomicEncodeFailure verifies writeAtomic error path when
// the temp file is not writable (encode fails on write).
func TestWriteAtomicEncodeFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "memory.toml")
	err := os.WriteFile(filePath, []byte(baseTOML), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// createTemp returns a file that's already closed, so encoding will fail.
	failCreateTemp := func(tmpDir, pattern string) (*os.File, error) {
		f, createErr := os.CreateTemp(tmpDir, pattern)
		if createErr != nil {
			return nil, createErr
		}

		// Close immediately so the encoder write fails.
		_ = f.Close()

		return f, nil
	}

	recorder := track.NewRecorder(
		track.WithCreateTemp(failCreateTemp),
	)

	memories := []*memory.Stored{
		{FilePath: filePath},
	}

	err = recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).To(HaveOccurred())
}

// TestWriteAtomicRenameFailure verifies writeAtomic error path when
// rename fails — temp file should be cleaned up.
func TestWriteAtomicRenameFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "memory.toml")
	err := os.WriteFile(filePath, []byte(baseTOML), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	failRename := func(_, _ string) error {
		return errors.New("permission denied")
	}

	var removedPath string

	trackRemove := func(path string) error {
		removedPath = path

		return os.Remove(path)
	}

	recorder := track.NewRecorder(
		track.WithRename(failRename),
		track.WithRemove(trackRemove),
	)

	memories := []*memory.Stored{
		{FilePath: filePath},
	}

	err = recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("permission denied"))

	// Verify temp file cleanup was attempted.
	g.Expect(removedPath).NotTo(BeEmpty())
}

// unexported constants.
const (
	baseTOML = `title = "Test Memory"
content = "some content"
observation_type = "correction"
concepts = ["testing"]
keywords = ["test"]
principle = "test principle"
anti_pattern = "test anti-pattern"
rationale = "test rationale"
confidence = "B"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-06-01T00:00:00Z"
`
)

// fullTOMLRecord mirrors all TOML fields for test verification.
type fullTOMLRecord struct {
	Title             string   `toml:"title"`
	Content           string   `toml:"content"`
	ObservationType   string   `toml:"observation_type"`
	Concepts          []string `toml:"concepts"`
	Keywords          []string `toml:"keywords"`
	Principle         string   `toml:"principle"`
	AntiPattern       string   `toml:"anti_pattern"`
	Rationale         string   `toml:"rationale"`
	Confidence        string   `toml:"confidence"`
	CreatedAt         string   `toml:"created_at"`
	UpdatedAt         string   `toml:"updated_at"`
	SurfacedCount     int      `toml:"surfaced_count"`
	LastSurfaced      string   `toml:"last_surfaced"`
	SurfacingContexts []string `toml:"surfacing_contexts"`
}
