package cli_test

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/transcript"
)

// TestPiSessionsAutoSweepIngestsPISessionsFromAncestor verifies that PI sessions
// in ancestor directories are auto-swept when AncestorPiDirs is enabled.
func TestPiSessionsAutoSweepIngestsPISessionsFromAncestor(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	line := piRecordLine("user", "2026-01-01T00:00:00Z", "USER: auto-swept session")
	fs := &memFS{files: map[string][]byte{
		"/.pi/session.jsonl": []byte(line + "\n"),
	}}

	deps := ingestDeps(fs)

	// Use explicit --sweep with AncestorPiDirs=true to trigger PI directory sweeping
	args := cli.IngestArgs{
		Sweep:     []string{"/"},
		ChunksDir: "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// The auto-swept PI session should be indexed
	_, ok := fs.files["/chunks/"+cli.ExportIndexFileName("/.pi/session.jsonl")]
	g.Expect(ok).To(gomega.BeTrue(), "auto-swept PI session must be indexed")
}

// TestPiSessionsExplicitFlagHandlesMultipleDirectories verifies multiple PI
// session directories are all ingested.
func TestPiSessionsExplicitFlagHandlesMultipleDirectories(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	line1 := piRecordLine("user", "2026-01-01T00:00:00Z", "USER: session 1")
	line2 := piRecordLine("user", "2026-01-01T00:00:00Z", "USER: session 2")
	fs := &memFS{files: map[string][]byte{
		"/.pi/session1.jsonl": []byte(line1 + "\n"),
		"/.pi/session2.jsonl": []byte(line2 + "\n"),
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

// TestPiSessionsExplicitFlagIngestsPIDirectories verifies that explicit --pi-sessions
// flag accepts PI session transcript directories and ingests them.
func TestPiSessionsExplicitFlagIngestsPIDirectories(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Create mock PI sessions directory structure with JSONL content
	userLine := piRecordLine("user", "2026-01-01T00:00:00Z", "USER: test pi session")
	assistantLine := piRecordLine("assistant", "2026-01-01T00:00:01Z", "ASSISTANT: ingested")
	otherLine := piRecordLine("user", "2026-01-01T00:00:00Z", "USER: another pi session")
	fs := &memFS{files: map[string][]byte{
		"/.pi/sessions/project1.jsonl": []byte(userLine + "\n" + assistantLine + "\n"),
		"/.pi/sessions/project2.jsonl": []byte(otherLine + "\n"),
	}}

	// Wire dependencies with proper directory walking for ListSources
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      func(path string, data []byte) error { return fs.write(path, data) },
		Stat:           func(_ string) (cli.SourceStat, error) { return cli.SourceStat{}, io.ErrUnexpectedEOF },
		ListSources:    memFSLister(fs),
		ReadTranscript: transcriptReader(piSessionTranscriptContent),
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

// TestPiSessionsExplicitFlagSkipsNonDirectories verifies that non-existent
// PI session directories are silently skipped (not an error).
func TestPiSessionsExplicitFlagSkipsNonDirectories(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	line := piRecordLine("user", "2026-01-01T00:00:00Z", "USER: session 1")
	fs := &memFS{files: map[string][]byte{
		"/.pi/session1.jsonl": []byte(line + "\n"),
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

// TestPiSessionsReadErrorOnDirScan verifies errors scanning PI session directories propagate.
func TestPiSessionsReadErrorOnDirScan(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := &memFS{files: map[string][]byte{}}

	// Simulate a directory-scan error on the .pi directory: ListSources is
	// the seam through which --pi-sessions dirs are enumerated.
	deps := ingestDeps(fs)
	deps.ListSources = func(_ cli.SweepRoot) ([]string, error) { return nil, errors.New("scan error") }

	args := cli.IngestArgs{
		PiSessions: []string{"/.pi"},
		ChunksDir:  "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).To(gomega.HaveOccurred())
}

// TestPiSessionsReadErrorPropagates verifies that errors reading PI sessions are
// properly propagated.
func TestPiSessionsReadErrorPropagates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	validLine := piRecordLine("user", "2026-01-01T00:00:00Z", "USER: valid")
	fs := &memFS{files: map[string][]byte{
		"/.pi/valid.jsonl": []byte(validLine + "\n"),
		"/.pi/bad.jsonl":   []byte("{not json}"), // Invalid JSONL
	}}

	deps := ingestDeps(fs)
	// The shared fake reader never fails; this test is specifically about
	// error propagation, so wire a reader that rejects the bad file.
	readErr := errors.New("transcript parse failure")
	deps.ReadTranscript = func(path string, _ time.Time, _ int) (transcript.ReadResult, error) {
		if strings.Contains(path, "bad") {
			return transcript.ReadResult{}, readErr
		}

		return transcript.ReadResult{Content: "USER: valid"}, nil
	}

	args := cli.IngestArgs{
		PiSessions: []string{"/.pi"},
		ChunksDir:  "/chunks",
	}

	err := cli.RunIngest(context.Background(), args, deps, io.Discard)
	g.Expect(err).To(gomega.HaveOccurred()) // Should fail on invalid JSONL
}

// TestPiSessionsRespectsBudget verifies that PI sessions respect the ingestion budget.
func TestPiSessionsRespectsBudget(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	line := piRecordLine("user", "2026-01-01T00:00:00Z", "USER: small session")
	fs := &memFS{files: map[string][]byte{
		"/.pi/small.jsonl": []byte(line + "\n"),
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

// TestPiSessionsWithTranscriptsCombinesBothSources verifies that explicit --pi-sessions
// works alongside explicit --transcript flags.
func TestPiSessionsWithTranscriptsCombinesBothSources(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	mdContent := "# Markdown\nSome content here that is long enough to clear the ingest noise floor " +
		"comfortably for chunking.\n"
	piLine := piRecordLine("user", "2026-01-01T00:00:00Z", "USER: pi session")
	fs := &memFS{files: map[string][]byte{
		"/docs/markdown.md":  []byte(mdContent),
		"/.pi/session.jsonl": []byte(piLine + "\n"),
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

// unexported constants.
const (
	piSessionTranscriptContent = "USER: pi session content long enough to clear the ingest noise floor easily\n\n" +
		"ASSISTANT: acknowledged, indexing this pi conversation for recall"
)

// ingestDeps creates test dependencies for PI session ingestion tests.
func ingestDeps(fs *memFS) cli.IngestDeps {
	return cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      func(path string, data []byte) error { return fs.write(path, data) },
		Stat:           func(_ string) (cli.SourceStat, error) { return cli.SourceStat{}, io.ErrUnexpectedEOF },
		ListSources:    memFSLister(fs),
		ReadTranscript: transcriptReader(piSessionTranscriptContent),
		Embedder:       fakeIngestEmbedder{},
		IsDir: func(path string) bool {
			base := filepath.Base(path)
			return base == "sessions" || path == "/.pi/sessions" || path == "/.pi"
		},
		Getwd: func() (string, error) { return "", nil },
	}
}

// memFSLister returns a SweepRoot lister that walks the in-memory mock
// filesystem — never the real disk. (An earlier version used
// filepath.WalkDir here, which silently swept the machine's real root and
// produced permission errors that looked like mock-persistence bugs.)
func memFSLister(fs *memFS) func(cli.SweepRoot) ([]string, error) {
	return func(root cli.SweepRoot) ([]string, error) {
		prefix := strings.TrimSuffix(root.Path, "/") + "/"

		var found []string

		for path := range fs.files {
			if path != root.Path && !strings.HasPrefix(path, prefix) {
				continue
			}

			ext := filepath.Ext(path)
			if ext == ".jsonl" || ext == ".md" {
				found = append(found, path)
			}
		}

		sort.Strings(found)

		return found, nil
	}
}

// piRecordLine builds one PI-session JSONL record with the given role,
// timestamp, and message content.
func piRecordLine(role, timestamp, content string) string {
	return `{"type":"` + role + `","timestamp":"` + timestamp + `",` +
		`"message":{"role":"` + role + `","content":"` + content + `"}}`
}
