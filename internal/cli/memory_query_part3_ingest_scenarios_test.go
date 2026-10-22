package cli_test

// Scenario tests for the Memory Query product capability, part 3 (861c2zn): when a chunk was first indexed
// (792a7p). Spec: docs/specs/product/memory-query.md. The scenarios run cli.RunIngest over the in-memory file
// system of ingest_test.go (memFS), with the real transcript reader (transcript.NewJSONLReader) over JSONL rows
// stamped the way a session stamps them, and read the index file back with chunk.DecodeRecords. The helpers carry
// the mq3 prefix. The scenarios of 792a7o and later are in memory_query_part3_scenarios_test.go.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/transcript"
)

// Test792a7p1a_RunIngest_AMarkdownChunkTakesTheIngestTime verifies 792a7p1a (A Markdown chunk takes the ingest
// time): of a Markdown document ingested at 2026-06-15 12:00, each record of its index file has an ingested_at of
// 2026-06-15 12:00.
func Test792a7p1a_RunIngest_AMarkdownChunkTakesTheIngestTime(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const source = "/docs/doc.md"

	ingestedAt := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	files := &memFS{files: map[string][]byte{source: []byte(mq3MarkdownDocument())}}

	deps := cli.IngestDeps{
		ReadFile: files.read, WriteFile: files.write, ReadTranscript: transcriptReader(""),
		Embedder: fakeIngestEmbedder{}, Now: func() time.Time { return ingestedAt },
	}

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Markdowns: []string{source}, ChunksDir: mq3ChunksDir}, deps, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	records := mq3ReadIndex(t, files, source)

	const sections = 2

	g.Expect(records).To(HaveLen(sections), "one record per section")

	for _, record := range records {
		g.Expect(record.IngestedAt).To(BeTemporally("==", ingestedAt), "the ingest time of %s", record.Anchor)
	}
}

// Test792a7p1b_RunIngest_ATranscriptChunkTakesTheSessionsLastTimestamp verifies 792a7p1b (A transcript chunk takes
// the session's last timestamp): of a transcript whose last entry is stamped 2026-05-10 14:00, ingested on
// 2026-06-01, each record of its index file has an ingested_at of 2026-05-10 14:00.
func Test792a7p1b_RunIngest_ATranscriptChunkTakesTheSessionsLastTimestamp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const source = "/sessions/session.jsonl"

	lastEntry := time.Date(2026, time.May, 10, 14, 0, 0, 0, time.UTC)
	ingestDay := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	files := &memFS{files: map[string][]byte{source: mq3SessionRows(t, 2, lastEntry)}}

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Transcripts: []string{source}, ChunksDir: mq3ChunksDir},
		mq3TranscriptDeps(files, ingestDay, nil), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	records := mq3ReadIndex(t, files, source)

	const turns = 2

	g.Expect(records).To(HaveLen(turns), "one record per turn")

	for _, record := range records {
		g.Expect(record.IngestedAt).To(BeTemporally("==", lastEntry), "the last timestamp of %s", record.Anchor)
	}
}

// Test792a7p1c_RunIngest_IngestingAgainKeepsTheFirstTime verifies 792a7p1c (Ingesting again keeps the first
// time): of an ingested transcript whose two chunks are unchanged on a second ingest (which also finds a new
// turn), each of the two chunks has the ingested_at of the first ingest; the new turn takes the second.
func Test792a7p1c_RunIngest_IngestingAgainKeepsTheFirstTime(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const source = "/sessions/session.jsonl"

	firstEntry := time.Date(2026, time.May, 10, 14, 0, 0, 0, time.UTC)
	secondEntry := time.Date(2026, time.June, 20, 9, 0, 0, 0, time.UTC)
	ingestDay := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	args := cli.IngestArgs{Transcripts: []string{source}, ChunksDir: mq3ChunksDir}

	files := &memFS{files: map[string][]byte{source: mq3SessionRows(t, 2, firstEntry)}}

	err := cli.RunIngest(context.Background(), args, mq3TranscriptDeps(files, ingestDay, nil), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	firstIndex := mq3ReadIndex(t, files, source)

	// The session grew by a turn: the same two turns, then a third stamped later.
	files.files[source] = mq3SessionRows(t, 3, secondEntry)

	err = cli.RunIngest(context.Background(), args, mq3TranscriptDeps(files, ingestDay, nil), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	secondIndex := mq3ReadIndex(t, files, source)

	g.Expect(firstIndex).To(HaveLen(2))
	g.Expect(secondIndex).To(HaveLen(3), "the two chunks it held, and the new turn")

	for _, before := range firstIndex {
		after, found := mq3FindRecord(secondIndex, before.ContentHash)
		g.Expect(found).To(BeTrue(), "%s is still held", before.Anchor)
		g.Expect(after.IngestedAt).To(BeTemporally("==", firstEntry), "%s keeps the first time", before.Anchor)
	}

	for _, record := range secondIndex {
		if _, held := mq3FindRecord(firstIndex, record.ContentHash); !held {
			g.Expect(record.IngestedAt).To(BeTemporally("==", secondEntry), "the new turn takes the second time")
		}
	}
}

// Test792a7p1d_RunIngest_AnOldChunkGetsItsSourcesModificationTime verifies 792a7p1d (An old chunk gets its
// source's modification time): of an index record with no ingested_at whose source the manifest records as
// modified 2026-01-20, ingesting the source again gives the record an ingested_at of 2026-01-20, not the current
// modification time of the source or the time of the ingest.
func Test792a7p1d_RunIngest_AnOldChunkGetsItsSourcesModificationTime(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const source = "/sessions/session.jsonl"

	recordedMtime := time.Date(2026, time.January, 20, 8, 0, 0, 0, time.UTC)
	currentMtime := time.Date(2026, time.June, 1, 8, 0, 0, 0, time.UTC)
	ingestDay := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	lastEntry := time.Date(2026, time.January, 19, 17, 0, 0, 0, time.UTC)
	args := cli.IngestArgs{Transcripts: []string{source}, ChunksDir: mq3ChunksDir}

	// The first ingest, as the system does it, records the source's modification time in the manifest.
	files := &memFS{files: map[string][]byte{source: mq3SessionRows(t, 1, lastEntry)}}
	oldStat := func(string) (cli.SourceStat, error) {
		return cli.SourceStat{MtimeUnixNano: recordedMtime.UnixNano(), Size: 1}, nil
	}

	err := cli.RunIngest(context.Background(), args, mq3TranscriptDeps(files, ingestDay, oldStat), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// The index as an earlier version wrote it: the same record, with no ingested_at.
	legacy := mq3ReadIndex(t, files, source)
	g.Expect(legacy).To(HaveLen(1))

	for index := range legacy {
		legacy[index].IngestedAt = time.Time{}
	}

	encoded, err := chunk.EncodeRecords(legacy)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	files.files[mq3IndexPath(source)] = encoded

	// The source is touched (a later modification time, different bytes), its chunks the same.
	files.files[source] = append(files.files[source], '\n')
	newStat := func(string) (cli.SourceStat, error) {
		return cli.SourceStat{MtimeUnixNano: currentMtime.UnixNano(), Size: 2}, nil
	}

	err = cli.RunIngest(context.Background(), args, mq3TranscriptDeps(files, ingestDay, newStat), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	records := mq3ReadIndex(t, files, source)
	g.Expect(records).To(HaveLen(1))

	for _, record := range records {
		g.Expect(record.IngestedAt).To(BeTemporally("==", recordedMtime), "the manifest's modification time")
	}
}

// unexported constants.
const (
	mq3ChunksDir = "/chunks"
	mq3TurnChars = 520
)

// mq3FileReader serves the transcript reader from the in-memory file system.
type mq3FileReader struct {
	files *memFS
}

// Read returns the bytes of the file at path.
func (r mq3FileReader) Read(path string) ([]byte, error) { return r.files.read(path) }

// mq3FindRecord returns the record with the given content hash.
func mq3FindRecord(records []chunk.Record, contentHash string) (chunk.Record, bool) {
	for _, record := range records {
		if record.ContentHash == contentHash {
			return record, true
		}
	}

	return chunk.Record{}, false
}

// mq3IndexPath is the path of the index file of a source in the chunks directory.
func mq3IndexPath(source string) string {
	return mq3ChunksDir + "/" + cli.ExportIndexFileName(source)
}

// mq3MarkdownDocument is a Markdown document of two sections, each long enough to be a chunk.
func mq3MarkdownDocument() string {
	return "## First section\nThe first section holds enough text to be kept as a chunk of the index.\n\n" +
		"## Second section\nThe second section holds enough text to be kept as a chunk of the index.\n"
}

// mq3ReadIndex decodes the index file of a source.
func mq3ReadIndex(t *testing.T, files *memFS, source string) []chunk.Record {
	t.Helper()

	data, found := files.files[mq3IndexPath(source)]
	if !found {
		t.Fatalf("no index file for %s", source)
	}

	records, err := chunk.DecodeRecords(data)
	if err != nil {
		t.Fatalf("decoding the index of %s: %v", source, err)
	}

	return records
}

// mq3SessionRows returns the JSONL of a session of the given number of turns (a user row and an assistant row
// each, a minute apart), stamped so that the last row carries the given time. Each user turn is long enough to be
// a chunk of its own, and no two turns are alike.
func mq3SessionRows(t *testing.T, turns int, last time.Time) []byte {
	t.Helper()

	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type row struct {
		Type      string  `json:"type"`
		Timestamp string  `json:"timestamp"`
		Message   message `json:"message"`
	}

	var out []byte

	rows := 2 * turns

	for index := range rows {
		role, text := "user", fmt.Sprintf("turn %d: %s", index/2+1, mq3WordsOfLength(mq3TurnChars))

		if index%2 == 1 {
			role, text = "assistant", "acknowledged"
		}

		stamp := last.Add(-time.Duration(rows-1-index) * time.Minute)

		entry := row{Type: role, Timestamp: stamp.Format(time.RFC3339), Message: message{Role: role, Content: text}}

		encoded, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("encoding a session row: %v", err)
		}

		out = append(out, encoded...)
		out = append(out, '\n')
	}

	return out
}

// mq3TranscriptDeps returns the ingest dependencies over the in-memory file system with the real transcript
// reader, the fixed clock and, when given, a stat function.
func mq3TranscriptDeps(
	files *memFS, now time.Time, stat func(string) (cli.SourceStat, error),
) cli.IngestDeps {
	reader := transcript.NewJSONLReader(mq3FileReader{files: files})

	return cli.IngestDeps{
		ReadFile: files.read, WriteFile: files.write, ReadTranscript: reader.ReadFrom,
		Embedder: fakeIngestEmbedder{}, Now: func() time.Time { return now }, Stat: stat,
	}
}
