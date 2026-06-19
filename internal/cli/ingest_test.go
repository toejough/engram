package cli_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/transcript"
)

func TestChunkIDSetContainsLoadedRecords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Build two records from two different sources.
	recA := chunk.Record{
		Source: "/sessions/a.jsonl", Anchor: "turn-1",
		ContentHash: "sha256:r1", Text: "chunk one", Vector: []float32{0.1},
	}
	recB := chunk.Record{
		Source: "/docs/b.md", Anchor: "Heading",
		ContentHash: "sha256:r2", Text: "chunk two", Vector: []float32{0.2},
	}

	encoded1, err := chunk.EncodeRecords([]chunk.Record{recA})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	encoded2, err := chunk.EncodeRecords([]chunk.Record{recB})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs := &memFS{files: map[string][]byte{
		"/chunks/" + cli.ExportIndexFileName("/sessions/a.jsonl"): encoded1,
		"/chunks/" + cli.ExportIndexFileName("/docs/b.md"):        encoded2,
	}}

	idSet, err := cli.ExportBuildChunkIDSet("/chunks", func(string) ([]string, error) {
		paths := make([]string, 0, len(fs.files))
		for k := range fs.files {
			if strings.HasSuffix(k, ".jsonl") {
				paths = append(paths, k)
			}
		}

		return paths, nil
	}, fs.read)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(idSet["/sessions/a.jsonl#turn-1"]).To(gomega.BeTrue(), "r1 id must be in set")
	g.Expect(idSet["/docs/b.md#Heading"]).To(gomega.BeTrue(), "r2 id must be in set")
	g.Expect(idSet["nonexistent#anchor"]).To(gomega.BeFalse(), "absent id must not be in set")
}

func TestChunkIDSetErrorsPropagate(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// listIndexes error propagates.
	_, listErr := cli.ExportBuildChunkIDSet("/chunks",
		func(string) ([]string, error) { return nil, errBoom },
		func(string) ([]byte, error) { return nil, nil })
	g.Expect(listErr).To(gomega.MatchError(errBoom), "listIndexes error must propagate")

	// readFile error propagates.
	_, readErr := cli.ExportBuildChunkIDSet("/chunks",
		func(string) ([]string, error) { return []string{"/chunks/x.jsonl"}, nil },
		func(string) ([]byte, error) { return nil, errBoom })
	g.Expect(readErr).To(gomega.MatchError(errBoom), "readFile error must propagate")

	// decode error propagates (malformed JSONL line).
	_, decodeErr := cli.ExportBuildChunkIDSet("/chunks",
		func(string) ([]string, error) { return []string{"/chunks/x.jsonl"}, nil },
		func(string) ([]byte, error) { return []byte("not json\n"), nil })
	g.Expect(decodeErr).To(gomega.HaveOccurred(), "decode error must propagate")
}

func TestIngestIsIdempotentByHash(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stripped := "USER: same content as before, long enough to clear the noise floor easily"
	fs := &memFS{files: map[string][]byte{"/sessions/s1.jsonl": []byte("{raw jsonl}")}}
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(stripped),
		Embedder:       fakeIngestEmbedder{},
	}
	args := cli.IngestArgs{Transcripts: []string{"/sessions/s1.jsonl"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	first := fs.files["/chunks/"+cli.ExportIndexFileName("/sessions/s1.jsonl")]

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	second := fs.files["/chunks/"+cli.ExportIndexFileName("/sessions/s1.jsonl")]

	g.Expect(second).To(gomega.Equal(first), "second ingest adds nothing")

	records, err := chunk.DecodeRecords(second)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).To(gomega.HaveLen(1))
}

func TestIngestMarkdownFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	md := "## Conventions\nAlways name constants instead of magic numbers in this codebase.\n"
	fs := &memFS{files: map[string][]byte{"/docs/conventions.md": []byte(md)}}
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(""),
		Embedder:       fakeIngestEmbedder{},
	}

	err := cli.RunIngest(context.Background(), cli.IngestArgs{
		Markdowns: []string{"/docs/conventions.md"},
		ChunksDir: "/chunks",
	}, deps, io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	records, err := chunk.DecodeRecords(fs.files["/chunks/"+cli.ExportIndexFileName("/docs/conventions.md")])
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).To(gomega.HaveLen(1))

	if len(records) != 1 {
		return
	}

	g.Expect(records[0].Anchor).To(gomega.Equal("Conventions"))
}

func TestIngestMarkdownSetsIngestedAtFromNow(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	md := "## Section\nSome markdown content long enough to form a chunk in the index.\n"
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	fs := &memFS{files: map[string][]byte{"/docs/doc.md": []byte(md)}}
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(""),
		Embedder:       fakeIngestEmbedder{},
		Now:            func() time.Time { return now },
	}
	args := cli.IngestArgs{Markdowns: []string{"/docs/doc.md"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	indexKey := "/chunks/" + cli.ExportIndexFileName("/docs/doc.md")
	records, err := chunk.DecodeRecords(fs.files[indexKey])
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(records).To(gomega.HaveLen(1))

	if len(records) < 1 {
		return
	}

	g.Expect(records[0].IngestedAt).To(gomega.Equal(now),
		"markdown chunk IngestedAt must be the ingest wall-clock (deps.Now())")
}

func TestIngestTranscriptSetsIngestedAtFromPerRowTimestamp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stripped := "USER: long enough transcript content to form a chunk in the index"
	sessionTS := time.Date(2026, 5, 10, 14, 0, 0, 0, time.UTC)

	fs := &memFS{files: map[string][]byte{"/sessions/ts.jsonl": []byte(`{}`)}}
	deps := cli.IngestDeps{
		ReadFile:  fs.read,
		WriteFile: fs.write,
		ReadTranscript: func(string, time.Time, int) (transcript.ReadResult, error) {
			return transcript.ReadResult{Content: stripped, LastTimestamp: sessionTS}, nil
		},
		Embedder: fakeIngestEmbedder{},
		Now:      func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) },
	}
	args := cli.IngestArgs{Transcripts: []string{"/sessions/ts.jsonl"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	indexKey := "/chunks/" + cli.ExportIndexFileName("/sessions/ts.jsonl")
	records, err := chunk.DecodeRecords(fs.files[indexKey])
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(records).To(gomega.HaveLen(1))

	if len(records) < 1 {
		return
	}

	// Transcript chunks use the per-session LastTimestamp (a per-session approximation
	// — all chunks of one transcript share IngestedAt; intra-session spread is negligible
	// for recency; cross-session ordering is distinguished since each session is its own source).
	g.Expect(records[0].IngestedAt).To(gomega.Equal(sessionTS),
		"transcript chunk IngestedAt must be the LastTimestamp from ReadResult (per-session approximation)")
}

func TestIngestTranscriptWritesChunkIndex(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stripped := strings.Join([]string{
		"USER: please add the linter config and wire it into the build system",
		"ASSISTANT: added golangci config and wired into targ check, all green",
	}, "\n")
	fs := &memFS{files: map[string][]byte{"/sessions/abc.jsonl": []byte("{raw jsonl}")}}
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(stripped),
		Embedder:       fakeIngestEmbedder{},
	}

	err := cli.RunIngest(context.Background(), cli.IngestArgs{
		Transcripts: []string{"/sessions/abc.jsonl"},
		ChunksDir:   "/chunks",
	}, deps, io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	data, ok := fs.files["/chunks/"+cli.ExportIndexFileName("/sessions/abc.jsonl")]
	g.Expect(ok).To(gomega.BeTrue(), "index file written")

	records, err := chunk.DecodeRecords(data)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).To(gomega.HaveLen(1))

	if len(records) != 1 {
		return
	}

	g.Expect(records[0].Source).To(gomega.Equal("/sessions/abc.jsonl"))
	g.Expect(records[0].Vector).NotTo(gomega.BeEmpty())
	g.Expect(records[0].ContentHash).To(gomega.HavePrefix("sha256:"))
}

func TestLoadPriorRecordsPreservesIngestedAt(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ingestedAt := time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC)
	rec := chunk.Record{
		Source:      "/sessions/s.jsonl",
		Anchor:      "turn-1",
		ContentHash: "sha256:aabbcc",
		Text:        "hello world",
		Vector:      []float32{0.5, 0.5},
		IngestedAt:  ingestedAt,
	}

	encoded, err := chunk.EncodeRecords([]chunk.Record{rec})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs := &memFS{files: map[string][]byte{"/chunks/s.jsonl": encoded}}
	deps := cli.IngestDeps{ReadFile: fs.read, WriteFile: fs.write, Embedder: fakeIngestEmbedder{}}

	got := cli.ExportLoadPriorRecords("/chunks/s.jsonl", deps)

	g.Expect(got).To(gomega.HaveLen(1))

	loaded, ok := got["sha256:aabbcc"]
	g.Expect(ok).To(gomega.BeTrue(), "record keyed by ContentHash")
	g.Expect(loaded.IngestedAt).To(gomega.Equal(ingestedAt), "IngestedAt must survive the load")
}

func TestMergeAppendBackfillsIngestedAtFromManifestMtime(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Simulate a legacy index with a record that has zero IngestedAt. The
	// ContentHash matches the real hash of Text so the re-ingest reuses it
	// (no new chunk) and the merge must backfill the legacy record in place.
	legacyText := "USER: old content that should survive and get backfilled"
	legacyRecord := chunk.Record{
		Source:      "/sessions/old.jsonl",
		Anchor:      "turn-1",
		ContentHash: chunk.HashText(legacyText),
		Text:        legacyText,
		Vector:      []float32{0.1, 0.2},
		// IngestedAt intentionally zero — simulating a pre-D5 record.
	}

	encoded, err := chunk.EncodeRecords([]chunk.Record{legacyRecord})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	indexFile := "/chunks/" + cli.ExportIndexFileName("/sessions/old.jsonl")
	manifestMtime := time.Date(2026, 1, 20, 8, 0, 0, 0, time.UTC)
	manifest := map[string]any{
		"/sessions/old.jsonl": map[string]any{
			"mtime_unix_nano": manifestMtime.UnixNano(),
			"size":            100,
			"file_hash":       "sha256:xyz",
		},
	}

	manifestBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// New source content is identical (same hash) so no new chunks are added.
	// The merge-append will load prior records and must backfill zero IngestedAt.
	fs := &memFS{files: map[string][]byte{
		"/sessions/old.jsonl":   []byte(`{"type":"same"}`),
		indexFile:               encoded,
		"/chunks/manifest.json": manifestBytes,
	}}

	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(legacyRecord.Text),
		Embedder:       fakeIngestEmbedder{},
		Now:            func() time.Time { return now },
	}
	args := cli.IngestArgs{Transcripts: []string{"/sessions/old.jsonl"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	records, err := chunk.DecodeRecords(fs.files[indexFile])
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(records).To(gomega.HaveLen(1))

	if len(records) < 1 {
		return
	}

	// The backfilled IngestedAt must equal the manifest mtime for this source.
	// Compare instants (BeTemporally) because time.Unix(0, nano) yields a
	// Local-zone time that represents the same instant as the UTC fixture.
	g.Expect(records[0].IngestedAt).To(gomega.BeTemporally("==", manifestMtime),
		"legacy zero-IngestedAt record must be backfilled from manifest mtime on first merge")
}

func TestMergeAppendKeepsPriorRecordsOnChange(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// First ingest: one chunk.
	first := "USER: first unique session content that is long enough"
	fs := &memFS{files: map[string][]byte{"/sessions/s.jsonl": []byte(`{"type":"raw"}`)}}
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(first),
		Embedder:       fakeIngestEmbedder{},
	}
	args := cli.IngestArgs{Transcripts: []string{"/sessions/s.jsonl"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	indexKey := "/chunks/" + cli.ExportIndexFileName("/sessions/s.jsonl")
	firstData := make([]byte, len(fs.files[indexKey]))
	copy(firstData, fs.files[indexKey])

	firstRecords, err := chunk.DecodeRecords(firstData)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(firstRecords).To(gomega.HaveLen(1))

	if len(firstRecords) < 1 {
		return
	}

	firstHash := firstRecords[0].ContentHash

	// Second ingest: source changed — contains BOTH original text AND new text.
	// Merge-append must keep the first chunk and add the second.
	both := first + "\nUSER: brand new second chunk content added later, also long enough"
	deps2 := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(both),
		Embedder:       fakeIngestEmbedder{},
	}
	// Touch the file so the mtime-skip doesn't short-circuit.
	fs.files["/sessions/s.jsonl"] = []byte(`{"type":"raw2"}`)

	g.Expect(cli.RunIngest(context.Background(), args, deps2, io.Discard)).To(gomega.Succeed())

	secondData := fs.files[indexKey]
	secondRecords, err := chunk.DecodeRecords(secondData)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Must contain the original chunk (preserved) plus at least one new one.
	g.Expect(secondRecords).To(gomega.HaveLen(2), "merge-append: prior chunk retained + new chunk added")

	hashes := make(map[string]bool, len(secondRecords))
	for _, r := range secondRecords {
		hashes[r.ContentHash] = true
	}

	g.Expect(hashes[firstHash]).To(gomega.BeTrue(), "original chunk must be retained by merge-append")
}

func TestMergeAppendNeverDeletesOnContentChange(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Ingest: content A.
	contentA := "USER: original content A that is definitely long enough to make a chunk"
	fs := &memFS{files: map[string][]byte{"/docs/note.md": []byte(contentA)}}
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(""),
		Embedder:       fakeIngestEmbedder{},
	}
	args := cli.IngestArgs{Markdowns: []string{"/docs/note.md"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	indexKey := "/chunks/" + cli.ExportIndexFileName("/docs/note.md")
	firstRecords, err := chunk.DecodeRecords(fs.files[indexKey])
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(firstRecords).To(gomega.HaveLen(1))

	if len(firstRecords) < 1 {
		return
	}

	oldHash := firstRecords[0].ContentHash

	// Source file changed: content B replaces A.
	contentB := "## New Heading\nCompletely different content that forms its own chunk, long enough."
	fs.files["/docs/note.md"] = []byte(contentB)

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	secondRecords, err := chunk.DecodeRecords(fs.files[indexKey])
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Both old AND new chunk must be in the index — no deletion.
	g.Expect(len(secondRecords)).To(gomega.BeNumerically(">=", 2),
		"merge-append must never delete; old chunk from content A must survive")

	hashes := make(map[string]bool, len(secondRecords))
	for _, r := range secondRecords {
		hashes[r.ContentHash] = true
	}

	g.Expect(hashes[oldHash]).To(gomega.BeTrue(), "old chunk must not be deleted on content change")
}

func TestMergeAppendPreservesIngestedAtOnReIngest(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// First ingest: sets IngestedAt = firstTime.
	content := "USER: stable content that will never change, long enough for a chunk"
	firstTime := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	secondTime := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC) // second ingest clock

	fs := &memFS{files: map[string][]byte{"/sessions/stable.jsonl": []byte(`{}`)}}
	readTranscript := func(string, time.Time, int) (transcript.ReadResult, error) {
		return transcript.ReadResult{Content: content, LastTimestamp: firstTime}, nil
	}
	deps1 := cli.IngestDeps{
		ReadFile: fs.read, WriteFile: fs.write,
		ReadTranscript: readTranscript, Embedder: fakeIngestEmbedder{},
		Now: func() time.Time { return firstTime },
	}
	args := cli.IngestArgs{Transcripts: []string{"/sessions/stable.jsonl"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps1, io.Discard)).To(gomega.Succeed())

	indexKey := "/chunks/" + cli.ExportIndexFileName("/sessions/stable.jsonl")
	firstRecords, err := chunk.DecodeRecords(fs.files[indexKey])
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(firstRecords).To(gomega.HaveLen(1))

	if len(firstRecords) < 1 {
		return
	}

	g.Expect(firstRecords[0].IngestedAt).To(gomega.Equal(firstTime))

	// Second ingest: same content (different raw file so hash-skip doesn't apply),
	// but same transcript text → same ContentHash → reused.
	// The re-ingest must preserve IngestedAt = firstTime, not overwrite with secondTime.
	fs.files["/sessions/stable.jsonl"] = []byte(`{"newraw":1}`)
	deps2 := cli.IngestDeps{
		ReadFile: fs.read, WriteFile: fs.write,
		ReadTranscript: func(string, time.Time, int) (transcript.ReadResult, error) {
			return transcript.ReadResult{Content: content, LastTimestamp: secondTime}, nil
		},
		Embedder: fakeIngestEmbedder{},
		Now:      func() time.Time { return secondTime },
	}

	g.Expect(cli.RunIngest(context.Background(), args, deps2, io.Discard)).To(gomega.Succeed())

	secondRecords, err := chunk.DecodeRecords(fs.files[indexKey])
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(secondRecords).To(gomega.HaveLen(1), "still one chunk — same content hash")

	if len(secondRecords) < 1 {
		return
	}

	g.Expect(secondRecords[0].IngestedAt).To(gomega.Equal(firstTime),
		"IngestedAt must be preserved from first ingest, not overwritten on re-ingest of identical chunk")
}

// fakeIngestEmbedder returns a fixed-dim vector derived from text length so
// tests can assert vectors landed without a real model.
type fakeIngestEmbedder struct{}

func (fakeIngestEmbedder) Dims() int { return 2 }

func (fakeIngestEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return []float32{float32(len(text)), 1}, nil
}

func (fakeIngestEmbedder) ModelID() string { return "fake@2" }

// memFS is an in-memory filesystem for ingest deps.
type memFS struct {
	files map[string][]byte
}

func (m *memFS) read(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}

	return data, nil
}

func (m *memFS) write(path string, data []byte) error {
	m.files[path] = data

	return nil
}

func transcriptReader(stripped string) func(string, time.Time, int) (transcript.ReadResult, error) {
	return func(string, time.Time, int) (transcript.ReadResult, error) {
		return transcript.ReadResult{Content: stripped}, nil
	}
}
