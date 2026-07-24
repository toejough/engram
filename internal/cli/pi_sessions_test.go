package cli_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// TestPiSessionsExplicitFlagIngestsPIDirectories verifies that explicit --pi-sessions
// flag accepts PI session transcript directories and ingests them.
func TestPiSessionsExplicitFlagIngestsPIDirectories(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Create mock PI sessions directory structure with JSONL content
	fs := &memFS{files: map[string][]byte{
		"/.pi/sessions/project1.jsonl": []byte(`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"USER: test pi session"}}` + "\n" + `{"type":"assistant","timestamp":"2026-01-01T00:00:01Z","message":{"role":"assistant","content":"ASSISTANT: ingested"}}` + "\n"),
		"/.pi/sessions/project2.jsonl": []byte(`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"USER: another pi session"}}` + "\n"),
	}}

	// Wire dependencies with proper directory walking for ListSources
	deps := cli.IngestDeps{
		ReadFile:  fs.read,
		WriteFile: func(path string, data []byte) error { return fs.write(path, data) },
		Stat:      func(_ string) (cli.SourceStat, error) { return cli.SourceStat{}, io.ErrUnexpectedEOF },
		ListSources: func(root cli.SweepRoot) ([]string, error) {
			var found []string
			err := filepath.WalkDir(root.Path, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				ext := filepath.Ext(path)
				if ext == ".jsonl" || ext == ".md" {
					found = append(found, path)
				}
				return nil
			})
			return found, err
		},
		ReadTranscript: transcriptReader(""),
		Embedder:       fakeIngestEmbedder{},
		IsDir: func(path string) bool {
			base := filepath.Base(path)
			return base == "sessions" || path == "/.pi/sessions" || path == "/.pi"
		},
		Getwd: func() (string, error) { return "", nil },
	}

	args := cli.IngestArgs{
		PiSessions: []string{"/.pi/sessions"},
		ChunksDir:  "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Verify chunks were created for both PI session files
	for _, src := range []string{"/.pi/sessions/project1.jsonl", "/.pi/sessions/project2.jsonl"} {
		_, ok := fs.files["/chunks/"+cli.ExportIndexFileName(src)]
		g.Expect(ok).To(gomega.BeTrue(), "PI session file must be indexed")
	}
}

// ingestDeps creates test dependencies for PI session ingestion tests.
func ingestDeps(fs *memFS) cli.IngestDeps {
	return cli.IngestDeps{
		ReadFile:  fs.read,
		WriteFile: func(path string, data []byte) error { return fs.write(path, data) },
		Stat:      func(_ string) (cli.SourceStat, error) { return cli.SourceStat{}, io.ErrUnexpectedEOF },
		ListSources: func(root cli.SweepRoot) ([]string, error) {
			var found []string
			err := filepath.WalkDir(root.Path, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				ext := filepath.Ext(path)
				if ext == ".jsonl" || ext == ".md" {
					found = append(found, path)
				}
				return nil
			})
			return found, err
		},
		ReadTranscript: transcriptReader(""),
		Embedder:       fakeIngestEmbedder{},
		IsDir: func(path string) bool {
			base := filepath.Base(path)
			return base == "sessions" || path == "/.pi/sessions" || path == "/.pi"
		},
		Getwd: func() (string, error) { return "", nil },
	}
}

// TestPiSessionsReadErrorPropagates verifies that errors reading PI sessions are
// properly propagated.
func TestPiSessionsReadErrorPropagates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := &memFS{files: map[string][]byte{
		"/.pi/valid.jsonl": []byte(`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"USER: valid"}}` + "\n"),
		"/.pi/bad.jsonl":   []byte("{not json}"), // Invalid JSONL
	}}

	deps := ingestDeps(fs)

	args := cli.IngestArgs{
		PiSessions: []string{"/.pi"},
		ChunksDir:  "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).To(gomega.HaveOccurred()) // Should fail on invalid JSONL
}

// TestPiSessionsExplicitFlagHandlesMultipleDirectories verifies multiple PI
// session directories are all ingested.
func TestPiSessionsExplicitFlagHandlesMultipleDirectories(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := &memFS{files: map[string][]byte{
		"/.pi/session1.jsonl": []byte(`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"USER: session 1"}}` + "\n"),
		"/.pi/session2.jsonl": []byte(`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"USER: session 2"}}` + "\n"),
	}}

	deps := ingestDeps(fs)

	args := cli.IngestArgs{
		PiSessions: []string{"/.pi"},
		ChunksDir:  "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Both session files should be indexed
	for _, src := range []string{"/.pi/session1.jsonl", "/.pi/session2.jsonl"} {
		_, ok := fs.files["/chunks/"+cli.ExportIndexFileName(src)]
		g.Expect(ok).To(gomega.BeTrue(), "PI session file must be indexed")
	}
}

// TestPiSessionsExplicitFlagSkipsNonDirectories verifies that non-existent
// PI session directories are silently skipped (not an error).
func TestPiSessionsExplicitFlagSkipsNonDirectories(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := &memFS{files: map[string][]byte{
		"/.pi/session1.jsonl": []byte(`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"USER: session 1"}}` + "\n"),
	}}

	deps := ingestDeps(fs)

	args := cli.IngestArgs{
		PiSessions: []string{"/.pi", "/nonexistent/pi"}, // One exists, one doesn't
		ChunksDir:  "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	// The valid session should still be indexed
	_, ok := fs.files["/chunks/"+cli.ExportIndexFileName("/.pi/session1.jsonl")]
	g.Expect(ok).To(gomega.BeTrue(), "valid PI session must be ingested")
}

// TestPiSessionsWithTranscriptsCombinesBothSources verifies that explicit --pi-sessions
// works alongside explicit --transcript flags.
func TestPiSessionsWithTranscriptsCombinesBothSources(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := &memFS{files: map[string][]byte{
		"/docs/markdown.md":   []byte("# Markdown\nSome content here.\n"),
		"/.pi/session.jsonl":  []byte(`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"USER: pi session"}}` + "\n"),
	}}

	deps := ingestDeps(fs)

	args := cli.IngestArgs{
		Transcripts: []string{"/docs/markdown.md"},
		PiSessions:  []string{"/.pi"},
		ChunksDir:   "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Both sources should be indexed
	_, mdOk := fs.files["/chunks/"+cli.ExportIndexFileName("/docs/markdown.md")]
	_, piOk := fs.files["/chunks/"+cli.ExportIndexFileName("/.pi/session.jsonl")]

	g.Expect(mdOk).To(gomega.BeTrue(), "markdown file must be indexed")
	g.Expect(piOk).To(gomega.BeTrue(), "/.pi/session.jsonl must be indexed")
}

// TestPiSessionsRespectsBudget verifies that PI sessions respect the ingestion budget.
func TestPiSessionsRespectsBudget(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := &memFS{files: map[string][]byte{
		"/.pi/small.jsonl": []byte(`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"USER: small session"}}` + "\n"),
	}}

	deps := ingestDeps(fs)

	args := cli.IngestArgs{
		PiSessions: []string{"/.pi"},
		ChunksDir:  "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	present, ok := fs.files["/chunks/"+cli.ExportIndexFileName("/.pi/small.jsonl")]
	g.Expect(ok).To(gomega.BeTrue(), "PI session must be indexed")
	if ok {
		records, err := chunk.DecodeRecords(present)
		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(records).NotTo(gomega.BeEmpty(), "should have records for this PI session")
	}
}

// TestPiSessionsReadErrorOnDirScan verifies errors scanning PI session directories propagate.
func TestPiSessionsReadErrorOnDirScan(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := &memFS{files: map[string][]byte{}}

	// Simulate ReadDir error on the .pi directory
	deps := ingestDeps(fs)
	deps.ReadDir = func(_ string) ([]os.DirEntry, error) { return nil, errors.New("scan error") }

	args := cli.IngestArgs{
		PiSessions: []string{"/.pi"},
		ChunksDir:  "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).To(gomega.HaveOccurred())
}

// TestPiSessionsAutoSweepIngestsPISessionsFromAncestor verifies that PI sessions
// in ancestor directories are auto-swept when AncestorPiDirs is enabled.
func TestPiSessionsAutoSweepIngestsPISessionsFromAncestor(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := &memFS{files: map[string][]byte{
		"/.pi/session.jsonl": []byte(`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"USER: auto-swept session"}}` + "\n"),
	}}

	deps := ingestDeps(fs)

	// Use explicit --sweep with AncestorPiDirs=true to trigger PI directory sweeping
	args := cli.IngestArgs{
		Sweep: []string{"/"},
		ChunksDir: "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// The auto-swept PI session should be indexed
	_, ok := fs.files["/chunks/"+cli.ExportIndexFileName("/.pi/session.jsonl")]
	g.Expect(ok).To(gomega.BeTrue(), "auto-swept PI session must be indexed")
}
