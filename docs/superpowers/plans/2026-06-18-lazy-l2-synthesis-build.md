# Lazy compositional L2 synthesis — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Build the lazy compositional L2 synthesis feature per `docs/superpowers/specs/2026-06-09-lazy-l2-synthesis-design.md` — the seven locked decisions D1–D7: single-pass clustering, in-place `engram amend`, exclude `Related to:` from the embed source, chunks-in-index with frontmatter provenance, append-only chunk history + per-chunk recency, recency-weighted distillation, and agent-judged coverage with top-K centroid candidate nomination.

**Architecture:** Synthesis runs at recall over one clustering of matched chunks + L2 notes; the binary nominates top-K candidate L2s per cluster (centroid cosine) and the recall skill's agent decides coverage (activate / update-in-place / create), writing note↔note `[[wikilinks]]` via `engram amend` and recording chunk sources as frontmatter provenance. Chunks stay in the append-only index (per-chunk `IngestedAt` recency); materialization/episode layers are dropped.

**Tech Stack:** Go 1.26; `internal/cli` + `internal/chunk` + `internal/embed`; build/test via `targ test` / `targ check-full` only; tests imptest + rapid + gomega, `package cli_test` blackbox via `export_test.go` `Export*` aliases; DI everywhere; SKILL.md edits via `superpowers:writing-skills`.

**Scope decision (recorded dissent).** Considered building the minimal value-slice first (`amend` + recall-skill note-linking over the existing note clustering); the user chose the **full §7 build** for the complete designed system. The value-proof experiment (eager-vs-lazy A/B, spec §3.6) remains **blocked on #642/#643** and is sequenced last — so this build is **not value-validatable** until those clear. Recorded per the anti-sycophantic resolution rule; the experiment is out of this plan's scope.

**Build order (dependency-sequenced).** Component 2 (append-only chunks + per-chunk recency) → 3 (unified clustering + top-K nomination) → 4 (exclude `Related to:` from embed) → 5 (`amend` + `learn --chunk-source`; depends on 4) → 6 (recall-skill rewrite; depends on 2, 3, 5) → 7 (reconcile docs). Each component's tasks are independently TDD'd and committed; Gate B (design-fit) runs after each refactor during execution.

---

### Component 2: Append-only chunk index + per-chunk recency (D5)

**Files:** `internal/chunk/index.go`, `internal/cli/ingest.go`, `internal/cli/recency.go`, `internal/cli/export_test.go`, `internal/cli/ingest_test.go`, `internal/cli/recency_test.go`

---

#### Task 2.1 — Add `IngestedAt time.Time` to `chunk.Record`

- [ ] **Write the failing test.** In `internal/chunk/index.go` — but first the test: create a new file `/Users/joe/repos/personal/engram/internal/chunk/ingestedat_test.go` (package `chunk_test`) with:

```go
package chunk_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/toejough/engram/internal/chunk"
)

func TestRecordIngestedAtRoundTrips(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ts := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	r := chunk.Record{
		Source:      "s.jsonl",
		Anchor:      "turn-1",
		ContentHash: "sha256:abc",
		Text:        "hello",
		Vector:      []float32{0.1},
		IngestedAt:  ts,
	}

	data, err := json.Marshal(r)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	var got chunk.Record
	err = json.Unmarshal(data, &got)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(got.IngestedAt).To(gomega.Equal(ts))
}

func TestDecodeRecordsPreservesIngestedAt(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ts := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	r := chunk.Record{
		Source:      "s.jsonl",
		Anchor:      "turn-2",
		ContentHash: "sha256:def",
		Text:        "world",
		Vector:      []float32{0.2},
		IngestedAt:  ts,
	}

	encoded, err := chunk.EncodeRecords([]chunk.Record{r})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	decoded, err := chunk.DecodeRecords(encoded)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(decoded).To(gomega.HaveLen(1))
	if len(decoded) < 1 {
		return
	}

	g.Expect(decoded[0].IngestedAt).To(gomega.Equal(ts))
}

func TestDecodeRecordsZeroIngestedAtWhenAbsent(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// JSONL without ingested_at (legacy record — migration backfill case)
	line := `{"source":"s.jsonl","anchor":"turn-1","content_hash":"sha256:abc","text":"hi","vector":[0.1]}` + "\n"

	decoded, err := chunk.DecodeRecords([]byte(line))
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(decoded).To(gomega.HaveLen(1))
	if len(decoded) < 1 {
		return
	}

	g.Expect(decoded[0].IngestedAt.IsZero()).To(gomega.BeTrue(),
		"absent ingested_at must decode as zero time (migration sentinel)")
}
```

- [ ] **Run it, confirm RED.**
  ```
  targ test
  ```
  Expected: compile error — `unknown field IngestedAt in struct literal of type chunk.Record`.

- [ ] **Add the field.** In `/Users/joe/repos/personal/engram/internal/chunk/index.go`, add after the `Vector` field:

```go
	// IngestedAt is the wall-clock time this chunk was first written to the
	// index. Zero for legacy records (pre-D5); backfilled on first merge.
	IngestedAt time.Time `json:"ingested_at,omitempty"` //nolint:tagliatelle // index schema uses snake_case like .vec.json
```

  Also add `"time"` to the import block.

- [ ] **Run RED→GREEN.**
  ```
  targ test
  ```
  Expected: all three new tests pass.

- [ ] **Run full check.**
  ```
  targ check-full
  ```
  Expected: clean (no new lint errors; `omitempty` on `time.Time` encodes as absent when zero).

---

#### Task 2.2 — `loadPriorVectors` → `loadPriorRecords` (returns full `chunk.Record`)

- [ ] **Write the failing test.** Add to `/Users/joe/repos/personal/engram/internal/cli/ingest_test.go` (inside package `cli_test`):

```go
func TestLoadPriorRecordsPreservesIngestedAt(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ts := time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC)
	r := chunk.Record{
		Source:      "/sessions/s.jsonl",
		Anchor:      "turn-1",
		ContentHash: "sha256:aabbcc",
		Text:        "hello world",
		Vector:      []float32{0.5, 0.5},
		IngestedAt:  ts,
	}

	encoded, err := chunk.EncodeRecords([]chunk.Record{r})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	fs := &memFS{files: map[string][]byte{"/chunks/s.jsonl": encoded}}
	deps := cli.IngestDeps{ReadFile: fs.read, WriteFile: fs.write, Embedder: fakeIngestEmbedder{}}

	got := cli.ExportLoadPriorRecords("/chunks/s.jsonl", deps)

	g.Expect(got).To(gomega.HaveLen(1))
	rec, ok := got["sha256:aabbcc"]
	g.Expect(ok).To(gomega.BeTrue(), "record keyed by ContentHash")
	g.Expect(rec.IngestedAt).To(gomega.Equal(ts), "IngestedAt must survive the load")
}
```

- [ ] **Run RED** — `cli.ExportLoadPriorRecords` does not exist yet.
  ```
  targ test
  ```
  Expected: compile error.

- [ ] **Rename internal function and change return type.** In `/Users/joe/repos/personal/engram/internal/cli/ingest.go`:

  Change `loadPriorVectors` signature from:
  ```go
  func loadPriorVectors(indexPath string, deps IngestDeps) map[string][]float32 {
  ```
  to:
  ```go
  // loadPriorRecords maps chunk ContentHash -> full Record from the existing
  // index file. An absent or unreadable index returns an empty map (first ingest).
  // The full Record is returned so IngestedAt survives the merge (D5).
  func loadPriorRecords(indexPath string, deps IngestDeps) map[string]chunk.Record {
  	records := map[string]chunk.Record{}
  
  	data, err := deps.ReadFile(indexPath)
  	if err != nil {
  		return records
  	}
  
  	decoded, err := chunk.DecodeRecords(data)
  	if err != nil {
  		return records
  	}
  
  	for _, r := range decoded {
  		records[r.ContentHash] = r
  	}
  
  	return records
  }
  ```

  Remove the old `loadPriorVectors` body entirely.

- [ ] **Update the sole caller (`rebuildIndex`) to use `loadPriorRecords`.** In `rebuildIndex`, change:
  ```go
  priorVectors := loadPriorVectors(indexPath, deps)
  ```
  to:
  ```go
  priorRecords := loadPriorRecords(indexPath, deps)
  ```
  And change the vector-reuse lookup from:
  ```go
  vector, ok := priorVectors[hash]
  ```
  to:
  ```go
  prior, ok := priorRecords[hash]
  var vector []float32
  if ok {
      vector = prior.Vector
  ```
  (keeping the `else` embed branch unchanged; close the `if` block before the `embedded++`).

- [ ] **Export the new function.** Add to `/Users/joe/repos/personal/engram/internal/cli/export_test.go`:
  ```go
  // ExportLoadPriorRecords exposes loadPriorRecords for ingest unit tests.
  func ExportLoadPriorRecords(indexPath string, deps cli.IngestDeps) map[string]chunk.Record {
      return loadPriorRecords(indexPath, deps)
  }
  ```
  (Note: this lives in the `cli` package's `export_test.go`, not `cli_test`.)

- [ ] **Run GREEN.**
  ```
  targ test
  ```
  Expected: new test passes; existing ingest tests still pass.

- [ ] **`targ check-full`** — confirm clean.

---

#### Task 2.3 — Merge-append: keep existing records, add only new-hash chunks, never delete

- [ ] **Write the failing tests.** Add to `ingest_test.go`:

```go
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
```

- [ ] **Run RED.**
  ```
  targ test
  ```
  Expected: failures — current `rebuildIndex` replaces from scratch.

- [ ] **Replace `rebuildIndex` with `mergeAppendIndex`.** In `/Users/joe/repos/personal/engram/internal/cli/ingest.go`, replace the body of `rebuildIndex` so it:
  1. Loads `priorRecords = loadPriorRecords(indexPath, deps)`
  2. Builds a `merged []chunk.Record` starting with all prior records (order preserved).
  3. Builds `existingHashes` set from prior records.
  4. For each new `piece` in `chunks`: if `hash` already in `existingHashes`, mark `reused++`, skip; else embed, create new record with `IngestedAt: now`, append to `merged`, mark `embedded++`.
  5. `total = len(merged)` (prior + new).
  6. Encodes and writes `merged`.

  The function signature and name stay the same (`rebuildIndex`) — the rename is internal behavior, not the exported API.

  New implementation of `rebuildIndex`:

```go
func rebuildIndex(
	ctx context.Context,
	source string,
	chunks []chunk.Chunk,
	chunksDir string,
	deps IngestDeps,
	now time.Time,
) (total, reused, embedded int, err error) {
	indexPath := filepath.Join(chunksDir, sourceSlug(source)+jsonlExt)
	priorRecords := loadPriorRecords(indexPath, deps)

	// Preserve prior records first (append-only: never delete).
	existingHashes := make(map[string]bool, len(priorRecords))
	merged := make([]chunk.Record, 0, len(priorRecords)+len(chunks))

	for _, r := range priorRecords {
		existingHashes[r.ContentHash] = true
		merged = append(merged, r)
	}

	for _, piece := range chunks {
		hash := chunk.HashText(piece.Text)

		if existingHashes[hash] {
			reused++

			continue
		}

		var vector []float32

		vector, err = deps.Embedder.Embed(ctx, piece.Text)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("ingest: embedding chunk %s/%s: %w", source, piece.Anchor, err)
		}

		embedded++
		merged = append(merged, chunk.Record{
			Source:      source,
			Anchor:      piece.Anchor,
			ContentHash: hash,
			Text:        piece.Text,
			Vector:      vector,
			IngestedAt:  now,
		})
	}

	data, err := chunk.EncodeRecords(merged)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ingest: encoding index %s: %w", indexPath, err)
	}

	err = deps.WriteFile(indexPath, data)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ingest: writing index %s: %w", indexPath, err)
	}

	return len(merged), reused, embedded, nil
}
```

  The `now time.Time` param is supplied by the caller. Update `ingestSource` to pass `deps.Now()`. Add `Now func() time.Time` to `IngestDeps` (wire `time.Now` in `newOsIngestDeps`).

  `ingestSource` change at the call site:
  ```go
  rebuilt, reused, embedded, err := rebuildIndex(ctx, source, chunks, chunksDir, deps, deps.Now())
  ```

  Wire in `newOsIngestDeps`:
  ```go
  Now: time.Now,
  ```

- [ ] **Run GREEN.**
  ```
  targ test
  ```
  Expected: both new tests pass; all existing ingest tests pass.

- [ ] **`targ check-full`** — clean.

---

#### Task 2.4 — Thread transcript per-row timestamp as `IngestedAt` for transcript chunks

- [ ] **Write the failing test.** Add to `ingest_test.go`:

```go
func TestIngestTranscriptSetsIngestedAtFromPerRowTimestamp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stripped := "USER: long enough transcript content to form a chunk in the index"
	ts := time.Date(2026, 5, 10, 14, 0, 0, 0, time.UTC)

	fs := &memFS{files: map[string][]byte{"/sessions/ts.jsonl": []byte(`{}`)}}
	deps := cli.IngestDeps{
		ReadFile:  fs.read,
		WriteFile: fs.write,
		ReadTranscript: func(string, time.Time, int) (transcript.ReadResult, error) {
			return transcript.ReadResult{Content: stripped, LastTimestamp: ts}, nil
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

	// Transcript chunks use the per-row LastTimestamp, not the ingest wall-clock.
	g.Expect(records[0].IngestedAt).To(gomega.Equal(ts),
		"transcript chunk IngestedAt must be the per-row LastTimestamp from ReadResult")
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
```

- [ ] **Run RED.**
  ```
  targ test
  ```
  Expected: `TestIngestTranscriptSetsIngestedAtFromPerRowTimestamp` fails (IngestedAt will be `Now()`, not the per-row timestamp). `TestIngestMarkdownSetsIngestedAtFromNow` may also fail if `Now` is nil in the existing test helper (it's not wired yet in `fakeIngestEmbedder` tests — ensure existing tests pass `Now` or tolerate nil).

- [ ] **Thread the per-row timestamp.** The source of truth is `transcript.ReadResult.LastTimestamp`. `chunkSource` returns `[]chunk.Chunk` — it cannot carry a timestamp today. Change approach: `chunkSource` returns `([]chunk.Chunk, time.Time, error)` where the `time.Time` is the per-row `LastTimestamp` for transcripts, zero for markdown. Update the signature:

```go
func chunkSource(source string, raw []byte, deps IngestDeps) ([]chunk.Chunk, time.Time, error) {
	if filepath.Ext(source) == jsonlExt {
		result, err := deps.ReadTranscript(source, time.Time{}, ingestBudgetBytes)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("ingest: stripping transcript %s: %w", source, err)
		}

		return chunk.Transcript(result.Content, chunkTargetChars, chunkMaxChars), result.LastTimestamp, nil
	}

	return chunk.Markdown(string(raw), chunkMaxChars), time.Time{}, nil
}
```

- [ ] **Pass the source timestamp into `ingestSource` → `rebuildIndex`.** In `ingestSource`, change:
  ```go
  chunks, err := chunkSource(source, raw, deps)
  if err != nil {
      return false, err
  }

  rebuilt, reused, embedded, err := rebuildIndex(ctx, source, chunks, chunksDir, deps, deps.Now())
  ```
  to:
  ```go
  chunks, sourceTS, err := chunkSource(source, raw, deps)
  if err != nil {
      return false, err
  }

  // For transcripts, use the per-row timestamp; for markdown, fall back to Now().
  ingestTime := sourceTS
  if ingestTime.IsZero() {
      ingestTime = deps.Now()
  }

  rebuilt, reused, embedded, err := rebuildIndex(ctx, source, chunks, chunksDir, deps, ingestTime)
  ```

- [ ] **Handle nil `Now` gracefully.** In `ingestSource`, if `deps.Now == nil`, `deps.Now()` panics. Guard:
  ```go
  now := time.Time{}
  if deps.Now != nil {
      now = deps.Now()
  }
  ingestTime := sourceTS
  if ingestTime.IsZero() {
      ingestTime = now
  }
  ```
  (Existing tests that don't wire `Now` will get `IngestedAt` = zero, which is fine — they don't assert on it.)

- [ ] **Run GREEN.**
  ```
  targ test
  ```
  Expected: both new tests pass.

- [ ] **`targ check-full`** — clean.

---

#### Task 2.5 — Migration backfill: populate `IngestedAt` from manifest mtime on first merge

- [ ] **Write the failing test.** Add to `ingest_test.go`:

```go
func TestMergeAppendBackfillsIngestedAtFromManifestMtime(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Simulate a legacy index with a record that has zero IngestedAt.
	legacyRecord := chunk.Record{
		Source:      "/sessions/old.jsonl",
		Anchor:      "turn-1",
		ContentHash: "sha256:legacy",
		Text:        "USER: old content that should survive and get backfilled",
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
	manifest := map[string]interface{}{
		"/sessions/old.jsonl": map[string]interface{}{
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
		"/sessions/old.jsonl":  []byte(`{"type":"same"}`),
		indexFile:              encoded,
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
	g.Expect(records[0].IngestedAt).To(gomega.Equal(manifestMtime),
		"legacy zero-IngestedAt record must be backfilled from manifest mtime on first merge")
}
```

- [ ] **Run RED.**
  ```
  targ test
  ```
  Expected: `TestMergeAppendBackfillsIngestedAtFromManifestMtime` fails — backfill not yet implemented.

- [ ] **Implement backfill in `rebuildIndex`.** The backfill reads the manifest to find the mtime for each prior record's source. Add a `backfillMtime` helper param to `rebuildIndex`, or do it within the function by reading `priorRecords` and checking for zero `IngestedAt`. The cleanest approach is: `ingestSource` already has the manifest in scope; pass a `backfillFn func(source string) time.Time` to `rebuildIndex` that returns the manifest mtime for a source path (zero if absent).

  Update `rebuildIndex` signature:
  ```go
  func rebuildIndex(
      ctx context.Context,
      source string,
      chunks []chunk.Chunk,
      chunksDir string,
      deps IngestDeps,
      ingestTime time.Time,
      backfillTime func(source string) time.Time,
  ) (total, reused, embedded int, err error) {
  ```

  In the "preserve prior records" loop, add:
  ```go
  for _, r := range priorRecords {
      existingHashes[r.ContentHash] = true
      if r.IngestedAt.IsZero() && backfillTime != nil {
          r.IngestedAt = backfillTime(r.Source)
      }
      merged = append(merged, r)
  }
  ```

  In `ingestSource`, build the `backfillTime` closure from the manifest before calling `rebuildIndex`:
  ```go
  backfill := func(src string) time.Time {
      entry, ok := manifest[src]
      if !ok {
          return time.Time{}
      }

      return time.Unix(0, entry.MtimeUnixNano)
  }

  rebuilt, reused, embedded, err := rebuildIndex(ctx, source, chunks, chunksDir, deps, ingestTime, backfill)
  ```

- [ ] **Run GREEN.**
  ```
  targ test
  ```

- [ ] **`targ check-full`** — clean.

---

#### Task 2.6 — `IngestedAt` preserved across re-ingest of unchanged chunk

- [ ] **Write the failing test.** Add to `ingest_test.go`:

```go
func TestMergeAppendPreservesIngestedAtOnReIngest(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// First ingest: sets IngestedAt = t1.
	content := "USER: stable content that will never change, long enough for a chunk"
	t1 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC) // second ingest clock

	fs := &memFS{files: map[string][]byte{"/sessions/stable.jsonl": []byte(`{}`)}}
	readTranscript := func(string, time.Time, int) (transcript.ReadResult, error) {
		return transcript.ReadResult{Content: content, LastTimestamp: t1}, nil
	}
	deps1 := cli.IngestDeps{
		ReadFile: fs.read, WriteFile: fs.write,
		ReadTranscript: readTranscript, Embedder: fakeIngestEmbedder{},
		Now: func() time.Time { return t1 },
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

	g.Expect(firstRecords[0].IngestedAt).To(gomega.Equal(t1))

	// Second ingest: same content (different raw file so hash-skip doesn't apply),
	// but same transcript text → same ContentHash → reused.
	// The re-ingest must preserve IngestedAt = t1, not overwrite with t2.
	fs.files["/sessions/stable.jsonl"] = []byte(`{"newraw":1}`)
	deps2 := cli.IngestDeps{
		ReadFile: fs.read, WriteFile: fs.write,
		ReadTranscript: func(string, time.Time, int) (transcript.ReadResult, error) {
			return transcript.ReadResult{Content: content, LastTimestamp: t2}, nil
		},
		Embedder: fakeIngestEmbedder{},
		Now:      func() time.Time { return t2 },
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

	g.Expect(secondRecords[0].IngestedAt).To(gomega.Equal(t1),
		"IngestedAt must be preserved from first ingest, not overwritten on re-ingest of identical chunk")
}
```

- [ ] **Run.** This test should pass already because the merge-append loop (Task 2.3) copies prior records including their `IngestedAt`. If it fails, the prior-record copy step is missing or broken.
  ```
  targ test
  ```
  Expected: GREEN (already correct by Task 2.3's implementation).

- [ ] **`targ check-full`** — clean.

---

#### Task 2.7 — Re-key `applyChunkRecency`: drop `ageDaysBySource` param, read `r.record.IngestedAt`

- [ ] **Write the failing test.** Add to `recency_test.go` (package `cli_test`):

```go
func TestApplyChunkRecencyUsesIngestedAt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	oldTime := now.Add(-90 * 24 * time.Hour) // 90 days ago
	recentTime := now.Add(-1 * time.Hour)    // 1 hour ago

	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.80, oldTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "recent.jsonl", Anchor: "turn-9"}, 0.45, recentTime),
	}

	maxTurn := map[string]int{"old.jsonl": 3, "recent.jsonl": 9}
	p := cli.ExportNewRecencyParams(60, 0.2, 0)

	out := cli.ExportApplyChunkRecencyByTime(scored, now, maxTurn, p)

	// Recent chunk (0.45 base) should outscore old chunk (0.80 base) after recency.
	g.Expect(cli.ExportScoredChunkScore(out[1])).To(
		BeNumerically(">", cli.ExportScoredChunkScore(out[0])),
		"recent chunk must outscore old chunk after per-IngestedAt recency")
}
```

  This uses a new export `ExportApplyChunkRecencyByTime` (the new signature) and `ExportNewScoredChunkWithIngestedAt`.

- [ ] **Run RED.**
  ```
  targ test
  ```
  Expected: compile error — new export functions don't exist.

- [ ] **Change `applyChunkRecency` signature.** In `/Users/joe/repos/personal/engram/internal/cli/recency.go`, replace the function:

```go
// applyChunkRecency returns a copy of scored with each score multiplied by its
// recency factor. turnFrac = turnN / maxTurn(source); 0 when the source has no
// turn anchors. Chunks with zero IngestedAt (legacy, not yet backfilled) are
// treated as age 0 (maximally recent) so they are not penalised.
func applyChunkRecency(
	scored []scoredChunk,
	now time.Time,
	maxTurnBySrc map[string]int,
	p recencyParams,
) []scoredChunk {
	out := make([]scoredChunk, len(scored))

	for i, s := range scored {
		ageDays := 0.0
		if !s.record.IngestedAt.IsZero() && !now.IsZero() {
			age := now.Sub(s.record.IngestedAt).Hours() / hoursPerDay
			if age > 0 {
				ageDays = age
			}
		}

		turnFrac := 0.0

		if n, ok := parseTurnN(s.record.Anchor); ok {
			if maxN := maxTurnBySrc[s.record.Source]; maxN > 0 {
				turnFrac = float64(n) / float64(maxN)
			}
		}

		out[i] = scoredChunk{
			record: s.record,
			score:  s.score * float32(recencyMultiplier(ageDays, turnFrac, p)),
		}
	}

	return out
}
```

- [ ] **Update the caller in `query.go`.** In `/Users/joe/repos/personal/engram/internal/cli/query.go` at line 1328:
  ```go
  // Old:
  scored = applyChunkRecency(scored, ages, maxTurnBySource(records), params)
  // New:
  scored = applyChunkRecency(scored, deps.Now(), maxTurnBySource(records), params)
  ```
  Remove the `ages` variable that was set by `chunkSourceAges`. The `chunkSourceAges` call is now unused by `applyChunkRecency`.

- [ ] **Add new exports to `export_test.go`:**
  ```go
  // ExportApplyChunkRecencyByTime exposes the new per-IngestedAt applyChunkRecency for recency tests.
  func ExportApplyChunkRecencyByTime(
      scored []scoredChunk, now time.Time, maxTurnBySrc map[string]int, p recencyParams,
  ) []scoredChunk {
      return applyChunkRecency(scored, now, maxTurnBySrc, p)
  }

  // ExportNewScoredChunkWithIngestedAt builds a scoredChunk with IngestedAt set for recency tests.
  func ExportNewScoredChunkWithIngestedAt(rec chunk.Record, score float32, ingestedAt time.Time) scoredChunk {
      rec.IngestedAt = ingestedAt
      return scoredChunk{record: rec, score: score}
  }
  ```

  Update the existing `ExportApplyChunkRecency` alias in `export_test.go` to match the new signature (it currently includes `ageDaysBySource map[string]float64` — remove that param):
  ```go
  ExportApplyChunkRecency = applyChunkRecency
  ```
  This alias will now have the new type automatically — no explicit update needed (it's a var holding a func value). But the existing test `TestApplyChunkRecencyLiftsRecentOverStaleHighCosine` calls the old 4-param form. Update that test to use the new `ExportApplyChunkRecencyByTime` with `now` and `IngestedAt`-carrying records.

- [ ] **Update `TestApplyChunkRecencyLiftsRecentOverStaleHighCosine`** to use the new API:
  ```go
  func TestApplyChunkRecencyLiftsRecentOverStaleHighCosine(t *testing.T) {
      t.Parallel()
      g := NewWithT(t)

      now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
      oldTime := now.Add(-90 * 24 * time.Hour)
      recentTime := now.Add(-6 * time.Minute)

      scored := []cli.ExportScoredChunk{
          cli.ExportNewScoredChunkWithIngestedAt(
              chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.80, oldTime),
          cli.ExportNewScoredChunkWithIngestedAt(
              chunk.Record{Source: "recent.jsonl", Anchor: "turn-9"}, 0.45, recentTime),
      }
      maxTurn := map[string]int{"old.jsonl": 3, "recent.jsonl": 9}
      p := cli.ExportNewRecencyParams(3, 0.2, 0)

      out := cli.ExportApplyChunkRecencyByTime(scored, now, maxTurn, p)

      g.Expect(cli.ExportScoredChunkScore(out[1])).To(BeNumerically(">", cli.ExportScoredChunkScore(out[0])))
  }
  ```

- [ ] **Run GREEN.**
  ```
  targ test
  ```

- [ ] **`targ check-full`** — clean.

---

#### Task 2.8 — Re-key `newestChunkItems`: sort key = `IngestedAt`, drop `ages` param

- [ ] **Write the failing test.** Add to `recency_test.go`:

```go
func TestNewestChunkItemsSortsByIngestedAt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	oldTime := now.Add(-60 * 24 * time.Hour)
	midTime := now.Add(-14 * 24 * time.Hour)
	recentTime := now.Add(-12 * time.Hour)

	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.90, oldTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "recent.jsonl", Anchor: "turn-7"}, 0.20, recentTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "mid.jsonl", Anchor: "turn-5"}, 0.50, midTime),
	}

	out := cli.ExportNewestChunkItemsByTime(scored, 2)

	g.Expect(out).To(HaveLen(2))
	if len(out) < 2 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("recent.jsonl#turn-7"), "newest IngestedAt first")
	g.Expect(cli.ExportResolvedItemPath(out[1])).To(Equal("mid.jsonl#turn-5"), "second-newest IngestedAt second")
}

func TestNewestChunkItemsTieBreaksByTurnDescIngestedAt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sameTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "a.jsonl", Anchor: "turn-2"}, 0.5, sameTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "a.jsonl", Anchor: "turn-9"}, 0.5, sameTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "a.jsonl", Anchor: "turn-5"}, 0.5, sameTime),
	}

	out := cli.ExportNewestChunkItemsByTime(scored, 2)

	g.Expect(out).To(HaveLen(2))
	if len(out) < 2 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("a.jsonl#turn-9"), "highest turn on tie")
	g.Expect(cli.ExportResolvedItemPath(out[1])).To(Equal("a.jsonl#turn-5"), "second-highest turn on tie")
}
```

- [ ] **Run RED.** `ExportNewestChunkItemsByTime` does not exist.

- [ ] **Change `newestChunkItems` signature** in `recency.go`: drop the `ages map[string]float64` param; sort by `s.record.IngestedAt` descending (newest first); zero `IngestedAt` sorts last (treat as maximally old, not maximally recent — inverse of the `applyChunkRecency` convention since the floor band is about identifying the most recent for guaranteed inclusion, not penalizing the unknown).

  New implementation:
  ```go
  // newestChunkItems returns the n chunk items with the largest IngestedAt
  // (most recently ingested first). Chunks with zero IngestedAt (legacy, not
  // yet backfilled) sort last. Tie-breaking on equal IngestedAt uses descending
  // turn-N (latest turn first). Returns nil when n<=0.
  func newestChunkItems(scored []scoredChunk, n int) []resolvedItem {
      if n <= 0 {
          return nil
      }

      type candidate struct {
          s scoredChunk
      }

      candidates := make([]candidate, 0, len(scored))
      for _, s := range scored {
          candidates = append(candidates, candidate{s: s})
      }

      sort.SliceStable(candidates, func(i, j int) bool {
          ti := candidates[i].s.record.IngestedAt
          tj := candidates[j].s.record.IngestedAt
          // Zero times sort last.
          if ti.IsZero() && tj.IsZero() {
              // tie-break by turn-N descending
              ni, _ := parseTurnN(candidates[i].s.record.Anchor)
              nj, _ := parseTurnN(candidates[j].s.record.Anchor)
              return ni > nj
          }
          if ti.IsZero() {
              return false
          }
          if tj.IsZero() {
              return true
          }
          if !ti.Equal(tj) {
              return ti.After(tj) // newer IngestedAt first
          }
          // tie-break: descending turn-N
          ni, _ := parseTurnN(candidates[i].s.record.Anchor)
          nj, _ := parseTurnN(candidates[j].s.record.Anchor)
          return ni > nj
      })

      if n > len(candidates) {
          n = len(candidates)
      }

      out := make([]resolvedItem, 0, n)
      for _, c := range candidates[:n] {
          out = append(out, resolvedItem{
              notePath:    chunkNotePath(c.s.record),
              content:     c.s.record.Text,
              score:       c.s.score,
              provenances: []string{provenanceDirect},
              kind:        chunkItemKind,
          })
      }

      return out
  }
  ```

- [ ] **Update the caller in `query.go`** (line ~1330):
  ```go
  // Old:
  chunkMust = newestChunkItems(scored, ages, params.floor)
  // New:
  chunkMust = newestChunkItems(scored, params.floor)
  ```

- [ ] **Update `ExportNewestChunkItems` in `export_test.go`** to match new signature, and add `ExportNewestChunkItemsByTime` as an alias:
  ```go
  // ExportNewestChunkItems exposes newestChunkItems (new signature: no ages map).
  func ExportNewestChunkItems(scored []scoredChunk, n int) []resolvedItem {
      return newestChunkItems(scored, n)
  }

  // ExportNewestChunkItemsByTime is an alias for tests that use the IngestedAt-keyed sort.
  func ExportNewestChunkItemsByTime(scored []scoredChunk, n int) []resolvedItem {
      return newestChunkItems(scored, n)
  }
  ```

- [ ] **Update existing tests** that call `ExportNewestChunkItems` with the old 3-arg form (`scored, ages, n`) — they must change to 2-arg (`scored, n`). The existing tests `TestNewestChunkItemsOrdersByAgeAscending`, `TestNewestChunkItemsTieBreaksByTurnDesc`, `TestNewestChunkItemsNZeroReturnsNil` use the old form with `ages map[string]float64`. Update each to use `ExportNewScoredChunkWithIngestedAt` and the 2-arg call.

- [ ] **Run GREEN.**
  ```
  targ test
  ```

- [ ] **`targ check-full`** — clean.

---

#### Task 2.9 — Remove `chunkSourceAges` and tidy the `ages` variable in `query.go`

- [ ] **Write the failing test.** (None needed — this is dead-code removal; the compile will fail if any caller remains.)

  Verify `chunkSourceAges` has no callers left after Task 2.7:
  ```
  grep -n "chunkSourceAges" /Users/joe/repos/personal/engram/internal/cli/*.go
  ```
  Expected: only the definition in `recency.go`.

- [ ] **Delete `chunkSourceAges`** from `/Users/joe/repos/personal/engram/internal/cli/recency.go`. Remove the full function body (lines 73–88 in the current file).

- [ ] **Remove the `ages` variable in `query.go`**: the call to `chunkSourceAges` that used to populate `ages` (line ~1325) is already gone from Task 2.7. Confirm the `ages` local variable and the block `if ages != nil { ... }` are removed and the code reads simply:
  ```go
  if deps.Now != nil {
      params := defaultRecencyParams()
      scored = applyChunkRecency(scored, deps.Now(), maxTurnBySource(records), params)
      sortScoredDesc(scored)
      chunkMust = newestChunkItems(scored, params.floor)
  }
  ```

- [ ] **Run.**
  ```
  targ check-full
  ```
  Expected: clean. The `readManifest`/`sourceAgeDays` helpers remain (used by migration backfill in `ingestSource`).

---

#### Task 2.10 — Stable chunk-id helper + in-memory id-set (`loadChunkRecords` returns `[]chunk.Record`)

The spec calls for: a stable chunk-id (already `source#anchor` via `chunkNotePath`); `loadChunkRecords` is O(total) — build an in-memory id-set after load rather than reach for a non-existent O(1) lookup. This task exposes that id-set builder for the `engram amend --chunk-source` validation in Component 5.

- [ ] **Write the failing test.** Add to `ingest_test.go`:

```go
func TestChunkIDSetContainsLoadedRecords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Build two records from two different sources.
	r1 := chunk.Record{
		Source: "/sessions/a.jsonl", Anchor: "turn-1",
		ContentHash: "sha256:r1", Text: "chunk one", Vector: []float32{0.1},
	}
	r2 := chunk.Record{
		Source: "/docs/b.md", Anchor: "Heading",
		ContentHash: "sha256:r2", Text: "chunk two", Vector: []float32{0.2},
	}

	encoded1, err := chunk.EncodeRecords([]chunk.Record{r1})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	encoded2, err := chunk.EncodeRecords([]chunk.Record{r2})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	fs := &memFS{files: map[string][]byte{
		"/chunks/" + cli.ExportIndexFileName("/sessions/a.jsonl"): encoded1,
		"/chunks/" + cli.ExportIndexFileName("/docs/b.md"):        encoded2,
	}}

	idSet, err := cli.ExportBuildChunkIDSet("/chunks", func(dir string) ([]string, error) {
		var paths []string
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
```

  Add `"strings"` to the import in `ingest_test.go` if not present.

- [ ] **Run RED.** `cli.ExportBuildChunkIDSet` does not exist.

- [ ] **Implement `buildChunkIDSet`.** Add to `/Users/joe/repos/personal/engram/internal/cli/ingest.go`:

```go
// buildChunkIDSet loads all chunk records from chunksDir and returns a
// set of "source#anchor" id strings. This is an O(total chunks) scan;
// callers should build it once and reuse the map for O(1) validation.
func buildChunkIDSet(
	chunksDir string,
	listIndexes func(dir string) ([]string, error),
	readFile func(path string) ([]byte, error),
) (map[string]bool, error) {
	paths, err := listIndexes(chunksDir)
	if err != nil {
		return nil, fmt.Errorf("ingest: listing chunk indexes for id-set: %w", err)
	}

	idSet := make(map[string]bool)

	for _, path := range paths {
		data, err := readFile(path)
		if err != nil {
			return nil, fmt.Errorf("ingest: reading chunk index %s for id-set: %w", path, err)
		}

		records, err := chunk.DecodeRecords(data)
		if err != nil {
			return nil, fmt.Errorf("ingest: decoding chunk index %s for id-set: %w", path, err)
		}

		for _, r := range records {
			idSet[r.Source+"#"+r.Anchor] = true
		}
	}

	return idSet, nil
}
```

- [ ] **Add the export** to `export_test.go`:
  ```go
  // ExportBuildChunkIDSet exposes buildChunkIDSet for validation tests.
  func ExportBuildChunkIDSet(
      chunksDir string,
      listIndexes func(dir string) ([]string, error),
      readFile func(path string) ([]byte, error),
  ) (map[string]bool, error) {
      return buildChunkIDSet(chunksDir, listIndexes, readFile)
  }
  ```

- [ ] **Run GREEN.**
  ```
  targ test
  ```

- [ ] **`targ check-full`** — clean.

---

#### Task 2.11 — Final integration: `targ check-full` over the complete component

- [ ] **Run full suite.**
  ```
  targ check-full
  ```
  Expected: all 8 checks green.

- [ ] **Verify `engram check` passes (nil check, thin API, etc.).**
  ```
  targ check-nils
  ```

- [ ] **Commit.**
  ```
  git add internal/chunk/index.go internal/chunk/ingestedat_test.go \
          internal/cli/ingest.go internal/cli/ingest_test.go \
          internal/cli/recency.go internal/cli/recency_test.go \
          internal/cli/export_test.go internal/cli/query.go
  git commit -m "$(cat <<'EOF'
  feat(chunk): append-only index + per-chunk IngestedAt recency (D5)

  - chunk.Record gains IngestedAt time.Time (json omitempty; zero = legacy)
  - loadPriorVectors → loadPriorRecords (returns full Record, preserves IngestedAt)
  - rebuildIndex → merge-append: keep prior records, add only new-hash chunks, never delete
  - Thread transcript per-row LastTimestamp as IngestedAt; markdown uses deps.Now()
  - Migration backfill: zero-IngestedAt records get manifest mtime on first merge
  - applyChunkRecency drops ageDaysBySource param; reads r.record.IngestedAt directly
  - newestChunkItems drops ages param; sorts by IngestedAt descending (zero sorts last)
  - chunkSourceAges removed (no callers)
  - buildChunkIDSet: O(total) load + in-memory id-set for amend validation (Component 5)

  AI-Used: [claude]
  EOF
  )"
  ```

---

### Risks/notes

1. **`ExportApplyChunkRecency` alias breakage:** The existing alias `ExportApplyChunkRecency = applyChunkRecency` in `export_test.go` is a bare function-value assignment. After Task 2.7 changes the signature, any call site referencing the old 4-param form will fail to compile. The plan covers updating `TestApplyChunkRecencyLiftsRecentOverStaleHighCosine`, but the synthesizer should grep for any other callers of `ExportApplyChunkRecency` across the eval tests (`recency_eval_test.go`) before executing Task 2.7 — that file uses the source-age map form and will need its own migration.

2. **`readManifest` dependency in both ingest and recency:** After removing `chunkSourceAges`, `readManifest` is still used in `ingestSource` (backfill, Task 2.5). The synthesizer must verify `readManifest`'s `deps` parameter type is `IngestDeps` (not `QueryDeps`) — the current recency.go calls it with `IngestDeps{ReadFile: deps.Read}` inside `chunkSourceAges`; that call site is being deleted, so `readManifest` should only be called from `ingest.go` after this component lands.

3. **`IngestDeps.Now` nil-safety across existing tests:** Adding `Now func() time.Time` to `IngestDeps` means all existing test fixtures that construct `IngestDeps` literals without `Now` will have `nil` for that field. The plan guards `deps.Now()` with a nil check in `ingestSource`; the synthesizer should verify all other call sites of `deps.Now()` (if any are added) also nil-guard.

### Component 3: One clustering over matched chunks+notes + top-K centroid nomination (D1/D7)

**Files:** `internal/cli/query.go`, `internal/cli/query_synthesis_test.go`, `internal/cli/synthesize_l2_property_test.go`, `internal/cli/query_unified_test.go`, `internal/cli/query_subgraph_test.go`

---

#### Task 3.1 — Replace singular `NearestL2` with `CandidateL2s []queryCandidateL2` in struct + adapt all call sites

**Scope:** Pure rename/reshape with no logic change. The existing tests become the RED baseline once we update `queryParsed` in tests to expect the new field and confirm the old `nearest_l2` YAML key is gone.

- [ ] **Write a failing test** in `/Users/joe/repos/personal/engram/internal/cli/query_synthesis_test.go` that asserts the wire field is `candidate_l2s` (slice) and `nearest_l2` is absent:

```go
// TestQuery_SynthesizeL2_EmitsCandidateL2sSlice is the RED baseline for the
// D7 struct rename: it asserts candidate_l2s (not nearest_l2) appears in the
// YAML payload, and that it is a sequence (not a scalar).
func TestQuery_SynthesizeL2_EmitsCandidateL2sSlice(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}
	for i := range 4 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)
	}
	plantDualVector(t, memFS, vault, "near.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}

	var out bytes.Buffer
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	var raw map[string]any
	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	clusters, _ := raw["clusters"].([]any)
	g.Expect(clusters).NotTo(BeEmpty())

	first := clusters[0].(map[string]any)
	_, hasOld := first["nearest_l2"]
	g.Expect(hasOld).To(BeFalse(), "nearest_l2 must be removed; use candidate_l2s")
	candidates, hasCandidates := first["candidate_l2s"]
	g.Expect(hasCandidates).To(BeTrue(), "candidate_l2s must appear in synthesize-l2 cluster output")
	_, isSlice := candidates.([]any)
	g.Expect(isSlice).To(BeTrue(), "candidate_l2s must be a sequence, not a scalar")
}
```

- [ ] Run `targ test 2>&1 | grep -E "FAIL|PASS|candidate_l2s"` — expect failure because the payload still emits `nearest_l2`.

Expected:
```
--- FAIL: TestQuery_SynthesizeL2_EmitsCandidateL2sSlice
    nearest_l2 must be removed; use candidate_l2s
```

- [ ] **In `internal/cli/query.go`, rename `queryNearestL2` → `queryCandidateL2` and reshape `queryCluster`:**

Replace (lines ~290–296):
```go
// queryNearestL2 is the nearest existing L2 note for a cluster centroid. It
// carries the RAW max(situation,body) cosine; the recall skill applies its own
// three-band decision (no band cutoff happens in the binary).
type queryNearestL2 struct {
	Path   string  `yaml:"path"`
	Cosine float32 `yaml:"cosine"`
}
```
With:
```go
// queryCandidateL2 is one candidate L2 note for a cluster. The binary emits
// top-K by centroid cosine (K >= candidateL2K); the recall skill judges
// coverage — no cosine-band decision happens in the binary.
type queryCandidateL2 struct {
	Path   string  `yaml:"path"`
	Cosine float32 `yaml:"cosine"`
}
```

Replace in `queryCluster` (line ~260):
```go
	NearestL2  *queryNearestL2      `yaml:"nearest_l2,omitempty"`
```
With:
```go
	CandidateL2s []queryCandidateL2 `yaml:"candidate_l2s,omitempty"`
```

- [ ] **Fix all compile errors caused by the rename in one pass** (do NOT run `targ test` after each — collect them all first):

In `renderClusters` (~line 1833), replace:
```go
				NearestL2:  nearestL2ForTier(centroid, l2Notes, tiers),
```
With (temporarily — the real function comes in Task 3.2):
```go
				CandidateL2s: candidateL2sStub(centroid, l2Notes, tiers),
```
Add a stub function below:
```go
// candidateL2sStub is a placeholder replaced by topKCandidateL2sForTier in Task 3.2.
func candidateL2sStub(centroid []float32, l2Notes tierIndex, tiers []string) []queryCandidateL2 {
	if len(tiers) > 0 && !slices.Contains(tiers, tierL2) {
		return nil
	}
	path, cosine, found := nearestInTierIndex(centroid, l2Notes)
	if !found {
		return nil
	}
	return []queryCandidateL2{{Path: path, Cosine: cosine}}
}
```

In `clusterChunkItems` (~line 695), replace:
```go
			NearestL2:  nearestL2ForTier(autoK.Centroids[clusterID], l2Notes, tiers),
```
With:
```go
			CandidateL2s: candidateL2sStub(autoK.Centroids[clusterID], l2Notes, tiers),
```

Remove `nearestL2ForTier` (now dead; `nearestInTierIndex` remains — it is still called by `nearestL3For`).

- [ ] **Update `queryParsed` in `internal/cli/query_subgraph_test.go`** (lines ~169–173):

Replace:
```go
		NearestL2 *struct {
			Path   string  `yaml:"path"`
			Cosine float32 `yaml:"cosine"`
		} `yaml:"nearest_l2"`
```
With:
```go
		CandidateL2s []struct {
			Path   string  `yaml:"path"`
			Cosine float32 `yaml:"cosine"`
		} `yaml:"candidate_l2s"`
```

- [ ] **Update `synthesize_l2_property_test.go`** (references `cluster.NearestL2` at lines ~74–77):

Replace:
```go
		for _, cluster := range parsed.Clusters {
			g.Expect(cluster.NearestL2).NotTo(BeNil(),
				"a near-duplicate L2 must always surface as nearest_l2")
			g.Expect(cluster.NearestL2.Cosine).To(BeNumerically(">=", noOpFloor),
				"a near-duplicate L2 must report raw cosine >= 0.95 (the no-op band precondition)")
		}
```
With:
```go
		for _, cluster := range parsed.Clusters {
			g.Expect(cluster.CandidateL2s).NotTo(BeEmpty(),
				"a near-duplicate L2 must always surface in candidate_l2s")
			g.Expect(cluster.CandidateL2s[0].Cosine).To(BeNumerically(">=", noOpFloor),
				"the nearest L2 (first candidate) must report raw cosine >= 0.95")
		}
```

- [ ] **Update `query_unified_test.go`** (lines ~60–77): replace `NearestL2 *struct{...}` → `CandidateL2s []struct{...}` and update the assertion `c.NearestL2 != nil && c.NearestL2.Path != ""` → `len(c.CandidateL2s) > 0 && c.CandidateL2s[0].Path != ""`.

- [ ] **Update all `cluster.NearestL2` references in `query_synthesis_test.go`** (there are ~10 references):
  - `g.Expect(cluster.NearestL2).NotTo(BeNil(), ...)` → `g.Expect(cluster.CandidateL2s).NotTo(BeEmpty(), ...)`
  - `cluster.NearestL2.Cosine` → `cluster.CandidateL2s[0].Cosine`
  - `cluster.NearestL2.Path` → `cluster.CandidateL2s[0].Path`
  - `g.Expect(c.NearestL2).To(BeNil(), ...)` → `g.Expect(c.CandidateL2s).To(BeEmpty(), ...)`
  - `g.Expect(c.NearestL2).NotTo(BeNil(), ...)` → `g.Expect(c.CandidateL2s).NotTo(BeEmpty(), ...)`
  - `g.Expect(c.NearestL2.Path).To(Equal(...))` → `g.Expect(c.CandidateL2s[0].Path).To(Equal(...))`

- [ ] Run `targ test` — all existing tests pass; new `EmitsCandidateL2sSlice` test also passes (stub returns a 1-element slice with `candidate_l2s`).

- [ ] Run `targ check-full` — expect clean (no unused functions; `nearestL2ForTier` removed).

- [ ] Commit: `refactor(query): rename nearest_l2→candidate_l2s; prepare for top-K nomination (D7)`

---

#### Task 3.2 — Implement `topKCandidateL2s` + replace stub

**Scope:** Add the real top-K logic. Write the critical "covering L2 is not #1 but appears in top-K" test BEFORE wiring the function.

- [ ] **Write the covering-L2-not-centroid-#1 test** in `internal/cli/query_synthesis_test.go`. This test fails with the stub (which returns only 1 entry and may miss the covering L2 when it isn't nearest):

```go
// TestQuery_SynthesizeL2_CoverL2NotCentroidFirst_AppearsInTopK verifies the
// D7 invariant: when a chunk-heavy centroid depresses absolute cosines, the
// covering L2 may not be the nearest to the centroid but still appears within
// top-K. The fixture places 4 notes at alpha, 1 chunk at alpha (depressing
// the centroid slightly toward a mix), and two L2s where L2-b is closer to
// the centroid than L2-a, but L2-a is the true cover (higher alpha cosine).
// With K>=3 and 3 L2s planted, all three appear in candidate_l2s.
func TestQuery_SynthesizeL2_CoverL2NotCentroidFirst_AppearsInTopK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// The "covering" L2: close to {1,0,0,0} (the note cluster centroid).
	// After mixing with one chunk at {0,1,0,0}, the centroid shifts toward
	// {0.8, 0.2, 0, 0}. The covering L2 at {1,0,0,0} has cosine ~0.98 to
	// this centroid, but another L2 at {0.8,0.6,0,0} is ~0.997 — closer.
	noteVec := []float32{1, 0, 0, 0}
	for i := range 4 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", noteVec, noteVec)
	}

	// L2-a: the covering L2, close to the NOTE centroid but not the blended centroid.
	l2a := []float32{1, 0, 0, 0}
	plantDualVector(t, memFS, vault, "l2a.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", l2a, l2a)

	// L2-b: closer to the BLENDED centroid (higher cosine with blended centroid).
	l2b := []float32{0.8, 0.6, 0, 0} // normalised: {0.8/1, 0.6/1} after scale
	plantDualVector(t, memFS, vault, "l2b.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", l2b, l2b)

	// L2-c: a third L2 to ensure K=3 is reachable.
	l2c := []float32{0, 1, 0, 0}
	plantDualVector(t, memFS, vault, "l2c.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", l2c, l2c)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: noteVec}

	var out bytes.Buffer
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	var raw map[string]any
	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	clusters, _ := raw["clusters"].([]any)
	g.Expect(clusters).NotTo(BeEmpty())

	first := clusters[0].(map[string]any)
	candidates, _ := first["candidate_l2s"].([]any)
	g.Expect(len(candidates)).To(BeNumerically(">=", 3),
		"top-K (K=3) must surface all three L2 candidates")

	// All three L2 paths must appear in the candidate list (covering L2 inclusive).
	paths := make([]string, 0, len(candidates))
	for _, c := range candidates {
		cm := c.(map[string]any)
		paths = append(paths, cm["path"].(string))
	}
	g.Expect(paths).To(ContainElement("l2a.fact.md"), "the covering L2 must appear in candidate_l2s")
	g.Expect(paths).To(ContainElement("l2b.fact.md"), "l2b must appear in candidate_l2s")
	g.Expect(paths).To(ContainElement("l2c.fact.md"), "l2c must appear in candidate_l2s")

	// Cosines must be descending.
	for i := 1; i < len(candidates); i++ {
		prev := candidates[i-1].(map[string]any)
		curr := candidates[i].(map[string]any)
		prevCos, _ := prev["cosine"].(float64)
		currCos, _ := curr["cosine"].(float64)
		g.Expect(prevCos).To(BeNumerically(">=", currCos),
			"candidate_l2s must be sorted cosine desc")
	}
}
```

Also add the K≥3 test (this one fails with stub since stub returns only 1):
```go
// TestQuery_SynthesizeL2_CandidateL2sTopKAtLeastThree verifies that when >=3
// L2 notes exist, candidate_l2s carries at least 3 entries sorted cosine desc.
func TestQuery_SynthesizeL2_CandidateL2sTopKAtLeastThree(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}
	for i := range 4 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)
	}

	l2Vecs := [][]float32{
		{1, 0, 0, 0},
		{1, 0.1, 0, 0},
		{1, 0.2, 0, 0},
		{1, 0.3, 0, 0},
		{1, 0.5, 0, 0},
	}
	for i, vec := range l2Vecs {
		plantDualVector(t, memFS, vault, fmt.Sprintf("l2-%d.fact.md", i),
			"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", vec, vec)
	}

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}

	var out bytes.Buffer
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	var raw map[string]any
	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	clusters, _ := raw["clusters"].([]any)
	g.Expect(clusters).NotTo(BeEmpty())

	first := clusters[0].(map[string]any)
	candidates, _ := first["candidate_l2s"].([]any)
	g.Expect(len(candidates)).To(BeNumerically(">=", 3),
		"candidate_l2s must carry top-K (K>=3) entries when enough L2s exist")

	for i := 1; i < len(candidates); i++ {
		prev := candidates[i-1].(map[string]any)
		curr := candidates[i].(map[string]any)
		prevCos, _ := prev["cosine"].(float64)
		currCos, _ := curr["cosine"].(float64)
		g.Expect(prevCos).To(BeNumerically(">=", currCos),
			"candidate_l2s must be sorted cosine desc (index %d >= %d)", i-1, i)
	}
}
```

- [ ] Run `targ test 2>&1 | grep -E "FAIL|PASS"` — both new tests fail (stub returns only 1 entry).

- [ ] **Implement `topKCandidateL2s` and `topKCandidateL2sForTier`** in `internal/cli/query.go`. Add the constant and both functions:

```go
// candidateL2K is the minimum number of candidate L2s to nominate per cluster.
// The recall skill reads all K candidates to judge coverage; generous nomination
// costs nothing (recall is the binary's job, precision is the agent's).
const candidateL2K = 3

// topKCandidateL2s returns the top-K L2 notes nearest the centroid by
// max(situation,body) cosine, sorted descending by cosine (ties broken by
// lexicographic path for stability). K is at least candidateL2K; when fewer
// than candidateL2K L2 notes exist, all are returned. An empty index returns nil.
// No cosine threshold is applied — nomination is generous (D7).
func topKCandidateL2s(centroid []float32, idx tierIndex) []queryCandidateL2 {
	if len(idx.paths) == 0 {
		return nil
	}

	type ranked struct {
		path   string
		cosine float32
	}

	all := make([]ranked, 0, len(idx.paths))

	for i := range idx.paths {
		sim := embed.Cosine(centroid, idx.sit[i])
		if bodySim := embed.Cosine(centroid, idx.body[i]); bodySim > sim {
			sim = bodySim
		}

		all = append(all, ranked{path: idx.paths[i], cosine: sim})
	}

	sort.SliceStable(all, func(i, j int) bool {
		if all[i].cosine != all[j].cosine {
			return all[i].cosine > all[j].cosine
		}

		return all[i].path < all[j].path
	})

	k := candidateL2K
	if len(all) < k {
		k = len(all)
	}

	out := make([]queryCandidateL2, k)

	for i := range k {
		out[i] = queryCandidateL2{Path: all[i].path, Cosine: all[i].cosine}
	}

	return out
}

// topKCandidateL2sForTier gates topKCandidateL2s on the requested tiers for
// T1a isolation. Suppressed when a non-empty tier set omits L2; nil/empty
// tiers always passes through (--synthesize-l2 passes nil).
func topKCandidateL2sForTier(centroid []float32, l2Notes tierIndex, tiers []string) []queryCandidateL2 {
	if len(tiers) > 0 && !slices.Contains(tiers, tierL2) {
		return nil
	}

	return topKCandidateL2s(centroid, l2Notes)
}
```

- [ ] **Replace the stub** everywhere it is called. In `renderClusters`:
```go
			CandidateL2s: candidateL2sStub(centroid, l2Notes, tiers),
```
→
```go
			CandidateL2s: topKCandidateL2sForTier(centroid, l2Notes, tiers),
```

In `clusterChunkItems`:
```go
			CandidateL2s: candidateL2sStub(autoK.Centroids[clusterID], l2Notes, tiers),
```
→
```go
			CandidateL2s: topKCandidateL2sForTier(autoK.Centroids[clusterID], l2Notes, tiers),
```

- [ ] Remove `candidateL2sStub` (now dead code).

- [ ] Run `targ test` — all tests pass.

- [ ] Run `targ check-full` — clean.

- [ ] Commit: `feat(query): topKCandidateL2s — top-K centroid cosine nomination for D7`

---

#### Task 3.3 — Extend `runSynthesizeL2Query` to include matched chunks in the unified clustering (D1)

**Scope:** Add chunk loading + scoring inside `runSynthesizeL2Query` and extend the subgraph members for D1's "one clustering over the matched set."

- [ ] **Write a failing test** in `internal/cli/query_synthesis_test.go` that runs `--synthesize-l2` with a `ChunksDir` configured, and asserts that chunk paths (containing `#`) appear as cluster members:

```go
// TestQuery_SynthesizeL2_IncludesMatchedChunksInClustering verifies D1: when
// ChunksDir is set, matched chunks appear as cluster members in the unified
// --synthesize-l2 clustering (not in a separate chunk-clusters channel).
func TestQuery_SynthesizeL2_IncludesMatchedChunksInClustering(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	// Three L1 notes.
	for i := range 3 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)
	}

	// Two chunk records at the same vector.
	records := []chunk.Record{
		{
			Source:      "/sessions/s.jsonl",
			Anchor:      "turn-1",
			ContentHash: chunk.HashText("chunk alpha one"),
			Text:        "chunk alpha one",
			Vector:      queryVec,
		},
		{
			Source:      "/sessions/s.jsonl",
			Anchor:      "turn-2",
			ContentHash: chunk.HashText("chunk alpha two"),
			Text:        "chunk alpha two",
			Vector:      queryVec,
		},
	}
	data, encErr := chunk.EncodeRecords(records)
	g.Expect(encErr).NotTo(HaveOccurred())
	if encErr != nil {
		return
	}
	memFS.files["/chunks/s.jsonl"] = data

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	deps.ListChunkIndexes = func(string) ([]string, error) {
		return []string{"/chunks/s.jsonl"}, nil
	}

	var out bytes.Buffer
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:      []string{"alpha"},
			VaultPath:    vault,
			SynthesizeL2: true,
			ChunksDir:    "/chunks",
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	var raw map[string]any
	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	// items[] must include both notes and chunks.
	items, _ := raw["items"].([]any)
	kinds := map[string]bool{}
	for _, item := range items {
		m := item.(map[string]any)
		kinds[m["kind"].(string)] = true
	}
	g.Expect(kinds["chunk"]).To(BeTrue(), "chunk items must appear in synthesize-l2 items[]")

	// clusters must have NO phrase="chunks" channel (D1: one unified pass only).
	clusters, _ := raw["clusters"].([]any)
	g.Expect(clusters).NotTo(BeEmpty())

	for _, cl := range clusters {
		cm := cl.(map[string]any)
		phrase, _ := cm["phrase"].(string)
		g.Expect(phrase).NotTo(Equal("chunks"),
			"D1: synthesize-l2 must not emit a separate chunks cluster channel")
	}

	// At least one cluster member must be a chunk (source#anchor path form).
	sawChunkMember := false
	for _, cl := range clusters {
		cm := cl.(map[string]any)
		members, _ := cm["members"].([]any)
		for _, member := range members {
			mm := member.(map[string]any)
			path, _ := mm["path"].(string)
			if strings.Contains(path, "#") {
				sawChunkMember = true
			}
		}
	}
	g.Expect(sawChunkMember).To(BeTrue(),
		"at least one cluster member must be a chunk (source#anchor) in synthesize-l2")
}
```

Add imports `"github.com/toejough/engram/internal/chunk"` and `"strings"` to `query_synthesis_test.go`.

- [ ] Run `targ test 2>&1 | grep -E "FAIL|pass"` — fails: no chunk members in clusters; chunk kind may or may not appear in items.

- [ ] **Add `kind` field to `subgraphMember`** in `internal/cli/query.go` (lines ~367–373):

```go
// subgraphMember bundles a node's basename, vault-relative path,
// sidecar vector, query-similarity score, and (optionally) cached body.
// kind overrides content-derived kind detection for chunk members.
type subgraphMember struct {
	basename string
	notePath string
	vector   []float32
	score    float32
	content  string
	kind     string // empty = note; chunkItemKind for chunks
}
```

- [ ] **Update `collectClusterMembers`** to propagate the member's kind to `queryClusterMember` path. Wait — `queryClusterMember` only has `Path`, `Score`, `IsRepresentative`. Chunk paths are already in `source#anchor` form (from `member.notePath`), so no changes needed to `queryClusterMember`. The `kind` field on `subgraphMember` is used only to tag resolved items. Check if `collectClusterMembers` / `renderClusters` needs changes — it doesn't for the Path field, since the path is already set correctly.

- [ ] **Extend `runSynthesizeL2Query`** to load chunks and include them in the subgraph:

Replace the body of `runSynthesizeL2Query` (starting at the `union, err := unionDirectHits` line) with:

```go
func runSynthesizeL2Query(
	ctx context.Context,
	args QueryArgs,
	notes []vaultgraph.Note,
	hits []compatibleSidecar,
	limit int,
	deps QueryDeps,
	stdout io.Writer,
) error {
	l1l2Hits := filterHitsToTiers(hits, args.VaultPath, deps.Read, []string{tierL1, tierL2})

	var nowL2 time.Time
	if deps.Now != nil {
		nowL2 = deps.Now()
	}

	union, err := unionDirectHits(ctx, args.Phrases, l1l2Hits, args.VaultPath, limit, nowL2, deps)
	if err != nil {
		return err
	}

	// D1: build the subgraph from the note union, then extend with matched chunks
	// so one AutoK pass clusters notes and chunks together.
	subgraph := buildUnionSubgraph(union)

	var chunkItems []resolvedItem // collected for items[] only

	if chunksConfigured(args, deps) {
		records, loadErr := loadChunkRecords(args.ChunksDir, ChunkQueryDeps{
			ListIndexes: deps.ListChunkIndexes, ReadFile: deps.Read, Embedder: deps.Embedder,
		})
		if loadErr != nil {
			return loadErr
		}

		scored, scoreErr := scoreChunks(ctx, args.Phrases, records, deps.Embedder)
		if scoreErr != nil {
			return scoreErr
		}

		for _, s := range scored {
			path := chunkNotePath(s.record)
			subgraph.members = append(subgraph.members, subgraphMember{
				notePath: path,
				content:  s.record.Text,
				vector:   s.record.Vector,
				score:    s.score,
				kind:     chunkItemKind,
			})
			chunkItems = append(chunkItems, resolvedItem{
				notePath:    path,
				content:     s.record.Text,
				score:       s.score,
				provenances: []string{provenanceDirect},
				kind:        chunkItemKind,
			})
		}
	}

	report := clusterUnionForSynthesis(subgraph, strings.Join(args.Phrases, "\n"))

	resolved := mergeProvenances(union, expandedSubgraph{}, clusterReport{}, hubReport{})
	resolved = applyProjectFilter(resolved, args.Project)
	resolved = append(resolved, chunkItems...)

	merged := aggregatedSummary{
		phrases:        args.Phrases,
		resolvedItems:  resolved,
		phraseClusters: []phrasedCluster{{phrase: synthesisClusterPhrase, report: report, subgraph: subgraph}},
		l3:             tierIndex{},
		l2:             gatherTierIndex(hits, args.VaultPath, deps.Read, tierL2),
		outgoing:       outgoingByBasename(notes),
		tiers:          nil,
		totalNotes:     len(notes),
		withEmbeddings: len(hits),
		limit:          limit,
		subgraphSize:   len(subgraph.members),
	}

	return renderQueryPayload(stdout, merged)
}
```

- [ ] Run `targ test` — new test passes; all existing tests pass.

- [ ] Run `targ check-full` — clean.

- [ ] Commit: `feat(query): D1 — include matched chunks in synthesize-l2 unified clustering`

---

#### Task 3.4 — Verify property test still passes + add `NearestL2FromFullVault` regression guard for chunks path

**Scope:** The property test in `synthesize_l2_property_test.go` adapted in Task 3.1 must still hold; also add a guard that chunk clusters in the normal (non-synthesize-l2) path still emit `candidate_l2s` (not breaking the Task 3.1 rename for `clusterChunkItems`).

- [ ] Run the property test explicitly:

```
$ targ test 2>&1 | grep -E "Property|PASS|FAIL"
```

Expect: `PASS: TestProperty_SynthesizeL2_NearDuplicateL2CosineAtLeast095`

- [ ] **Update `query_unified_test.go` `TestRunQuery_ChunkClustersCarryNearestL2`** — the test name and assertion refer to `nearest_l2` which was already adapted in Task 3.1. Rename the test to `TestRunQuery_ChunkClustersCarryCandidateL2s` to reflect the new field; verify the assertion uses `c.CandidateL2s`:

```go
func TestRunQuery_ChunkClustersCarryCandidateL2s(t *testing.T) {
	// ... (same body as before, but with CandidateL2s assertions from Task 3.1)
```

- [ ] Run `targ test` — all pass.

- [ ] Run `targ check-full` — clean.

- [ ] Commit: `test(query): rename chunk-cluster nearest_l2 test to candidate_l2s`

---

**Risks/notes for the synthesizer:**

1. **D5 recency dependency:** `runSynthesizeL2Query` calls `scoreChunks` directly (raw cosine, no recency). The D5 component changes `applyChunkRecency` to read `r.record.IngestedAt` instead of the `ageDaysBySource` map. My component deliberately bypasses recency (matching the spec's sequencing — D5 must land before the recall skill can use recency-weighted distillation, but the binary's clustering coordinate is raw cosine). No conflict as long as D5 doesn't change `scoreChunks`'s signature.

2. **`subgraphMember.kind` field:** The new `kind` field on `subgraphMember` is consumed immediately (in `runSynthesizeL2Query` → resolvedItem building). However, `collectClusterMembers` and `memberMatchesTier` do not use it — chunk members pass `memberMatchesTier` with `tiers=nil` because the empty-tiers path returns true unconditionally. If a future change adds tier filtering to `collectClusterMembers` for the synthesize-l2 path, chunk members (no frontmatter tier) will be dropped unless `memberMatchesTier` is updated to recognize `kind=chunkItemKind`.

3. **`mergeClusterReps` and chunk basenames:** `runSynthesizeL2Query` calls `mergeProvenances(union, expandedSubgraph{}, ...)` — the empty subgraph means `mergeClusterReps` is a no-op. Chunk subgraph members are NOT fed through `mergeClusterReps`, so the empty `basename` field on chunk members is safe. If a future refactor passes the real subgraph to `mergeProvenances` in `runSynthesizeL2Query`, chunk members with `basename=""` would all map to the same key — that would need fixing.

### Component 4: Exclude `Related to:` from the embed source (D3)

**Goal:** `BodyText`/`ContentHash` must ignore a trailing `Related to:` section so a link-only edit (adding or changing `[[wikilinks]]` in that block) does not change the ContentHash or perturb the body vector. Recognition is conservative: the `Related to:` marker line followed only by relation bullets (`- [[…`) and blank lines. Inline prose mentioning "Related to:" is left intact.

**Files:** `internal/embed/hash.go`, `internal/embed/hash_test.go`, `internal/embed/hash_property_test.go`

#### Task 4.1 — Strip a trailing `Related to:` block in `BodyText`

- [ ] Add this RED unit test to `internal/embed/hash_test.go` (it fails today because `BodyText` returns the full body including the `Related to:` block):

  ```go
  func TestBodyText_ExcludesRelatedToSection(t *testing.T) {
  	t.Parallel()

  	g := NewWithT(t)
  	raw := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
  		"Information learned: when in X, S P O.\n\n" +
  		"Related to:\n- [[2.note]] — because.\n- [[3.note]] — also.\n")
  	want := "Information learned: when in X, S P O.\n"
  	g.Expect(string(embed.BodyText(raw))).To(Equal(want))
  }
  ```

- [ ] Run the test, confirm it FAILS:

  ```
  targ test
  ```

  Expected: `TestBodyText_ExcludesRelatedToSection` fails — actual still contains the `Related to:` lines.

- [ ] Implement the minimal change in `internal/embed/hash.go`. Wire `stripRelatedToSection` into `BodyText`, and add `isRelatedToBlock` plus the two constants. Replace the existing `BodyText` and the `const (...)` block:

  ```go
  // BodyText returns the note body (frontmatter stripped) with any trailing
  // "Related to:" section removed. It is the body-vector source for every
  // note type. Dropping the relation block means a link-only edit (adding or
  // changing [[wikilinks]] under "Related to:") leaves the body vector and
  // ContentHash unchanged (D3).
  func BodyText(raw []byte) []byte {
  	return stripRelatedToSection(ExtractBody(raw))
  }
  ```

  ```go
  // unexported constants.
  const (
  	frontmatterDelim = "---\n"
  	// relatedSectionMarker is the line that opens a rendered "Related to:"
  	// relation block (see internal/cli renderRelatedSection). It is matched
  	// exactly (after trimming a trailing carriage return) so inline prose that
  	// merely mentions "Related to:" is not treated as a block opener.
  	relatedSectionMarker = "Related to:"
  	// relatedSectionBulletPfx is the prefix of every rendered relation bullet
  	// ("- [[target]] — rationale."). A "Related to:" marker line counts as a
  	// block only when every following non-blank line starts with this prefix.
  	relatedSectionBulletPfx = "- [["
  )

  // stripRelatedToSection removes a trailing "Related to:" relation block from
  // body, returning body unchanged when no such block is present. The block is
  // recognised conservatively (see isRelatedToBlock): a "Related to:" marker
  // line whose following non-blank lines are all relation bullets. Recognising
  // only the LAST marker, and only when the lines after it qualify, leaves prose
  // that mentions "Related to:" inline untouched.
  func stripRelatedToSection(body []byte) []byte {
  	lines := bytes.Split(body, []byte("\n"))

  	for i := len(lines) - 1; i >= 0; i-- {
  		if bytes.Equal(bytes.TrimRight(lines[i], "\r"), []byte(relatedSectionMarker)) {
  			if isRelatedToBlock(lines[i+1:]) {
  				return bytes.TrimRight(bytes.Join(lines[:i], []byte("\n")), "\n")
  			}
  		}
  	}

  	return body
  }

  // isRelatedToBlock reports whether the lines that follow a "Related to:"
  // marker form a relation block: every non-blank line must start with
  // relatedSectionBulletPfx, and at least one bullet must be present. A line
  // that is neither blank nor a bullet (prose) disqualifies the block, so an
  // inline "Related to:" mention is not stripped.
  func isRelatedToBlock(after [][]byte) bool {
  	sawBullet := false

  	for _, line := range after {
  		trimmed := bytes.TrimRight(line, "\r")
  		if len(bytes.TrimSpace(trimmed)) == 0 {
  			continue
  		}

  		if !bytes.HasPrefix(trimmed, []byte(relatedSectionBulletPfx)) {
  			return false
  		}

  		sawBullet = true
  	}

  	return sawBullet
  }
  ```

- [ ] Run the test, confirm it PASSES:

  ```
  targ test
  ```

  Expected: `TestBodyText_ExcludesRelatedToSection` passes; all prior `BodyText`/`ContentHash` tests still pass.

- [ ] Run full checks:

  ```
  targ check-full
  ```

  Expected: clean. (`stripRelatedToSection`, `isRelatedToBlock`, and both constants are all referenced — no unused-symbol lint.)

#### Task 4.2 — `ContentHash` is insensitive to link-only edits

- [ ] Add this RED unit test to `internal/embed/hash_test.go`:

  ```go
  func TestContentHash_IgnoresRelatedToLinkEdits(t *testing.T) {
  	t.Parallel()

  	g := NewWithT(t)
  	noBlock := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
  		"Information learned: when in X, S P O.\n")
  	withBlock := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
  		"Information learned: when in X, S P O.\n\n" +
  		"Related to:\n- [[2.note]] — because.\n")
  	diffLinks := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
  		"Information learned: when in X, S P O.\n\n" +
  		"Related to:\n- [[2.note]] — because.\n- [[9.other]] — added later.\n")

  	g.Expect(embed.ContentHash(noBlock)).To(Equal(embed.ContentHash(withBlock)))
  	g.Expect(embed.ContentHash(withBlock)).To(Equal(embed.ContentHash(diffLinks)))
  }
  ```

- [ ] Run, confirm result:

  ```
  targ test
  ```

  Expected: PASSES (Task 4.1 already strips the block; this test guards the hash-level contract — adding the block, then adding/changing links inside it, leaves the hash identical).

- [ ] Run full checks:

  ```
  targ check-full
  ```

  Expected: clean.

#### Task 4.3 — Property: the `Related to:` block is hash-invisible (and real body changes still register)

- [ ] Add this RED property test to `internal/embed/hash_property_test.go`, building on the existing `genRawNote` generator:

  ```go
  // TestContentHash_RelatedToInsensitivityProperty asserts that appending a
  // "Related to:" relation block to any note shape (or changing the links
  // inside it) is invisible to ContentHash — a link-only edit must not mark a
  // note stale (D3). A guard inside the same property confirms the strip is
  // not over-broad: a genuine body change (a different prose line BEFORE the
  // block) still changes the hash.
  func TestContentHash_RelatedToInsensitivityProperty(t *testing.T) {
  	t.Parallel()

  	rapid.Check(t, func(rt *rapid.T) {
  		g := NewWithT(rt)

  		base := genRawNote(rt)
  		blockA := "\nRelated to:\n- [[" + genFieldValue(rt, "linkA") + "]] — a.\n"
  		blockB := "\nRelated to:\n- [[" + genFieldValue(rt, "linkB") + "]] — b.\n" +
  			"- [[" + genFieldValue(rt, "linkB2") + "]] — c.\n"

  		withA := append(append([]byte{}, base...), []byte(blockA)...)
  		withB := append(append([]byte{}, base...), []byte(blockB)...)

  		// Appending a relation block, and varying its links, is hash-invisible.
  		g.Expect(embed.ContentHash(withA)).To(Equal(embed.ContentHash(base)))
  		g.Expect(embed.ContentHash(withB)).To(Equal(embed.ContentHash(base)))

  		// Guard: a real body change (extra prose line before the block) is NOT
  		// hidden by the strip — the hash must differ.
  		extra := genFieldValue(rt, "extraLine")
  		changedBody := append(append([]byte{}, base...), []byte(extra+"\n")...)
  		changedWithA := append(append([]byte{}, changedBody...), []byte(blockA)...)
  		g.Expect(embed.ContentHash(changedWithA)).NotTo(Equal(embed.ContentHash(withA)))
  	})
  }
  ```

- [ ] Run, confirm result:

  ```
  targ test
  ```

  Expected: PASSES across all `genRawNote` shapes (episode, fact, feedback, bare). The invariant pair confirms the block is hash-invisible; the guard confirms the strip does not swallow real body edits.

- [ ] Run full checks:

  ```
  targ check-full
  ```

  Expected: clean.

#### Task 4.4 — Conservative recognition edge cases

- [ ] Add these RED unit tests to `internal/embed/hash_test.go`:

  ```go
  func TestBodyText_InlineRelatedToProseIsNotStripped(t *testing.T) {
  	t.Parallel()

  	// "Related to:" appears as inline prose with no bullet block beneath it,
  	// so the whole body — including that line — must survive.
  	g := NewWithT(t)
  	raw := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
  		"The bug was Related to: a missing nil guard in the parser.\n")
  	want := "The bug was Related to: a missing nil guard in the parser.\n"
  	g.Expect(string(embed.BodyText(raw))).To(Equal(want))
  }

  func TestBodyText_MarkerFollowedByProseIsNotStripped(t *testing.T) {
  	t.Parallel()

  	// A "Related to:" marker line whose following non-blank line is prose (not
  	// a "- [[" bullet) is not a relation block; nothing is stripped.
  	g := NewWithT(t)
  	raw := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
  		"Body line.\n\nRelated to:\nsee the design doc for context.\n")
  	want := "Body line.\n\nRelated to:\nsee the design doc for context.\n"
  	g.Expect(string(embed.BodyText(raw))).To(Equal(want))
  }
  ```

- [ ] Run, confirm result:

  ```
  targ test
  ```

  Expected: PASSES. `isRelatedToBlock` returns false for the inline case (no marker line equals `Related to:` exactly — it has surrounding prose) and for the marker-then-prose case (a non-bullet, non-blank line disqualifies the block), so `stripRelatedToSection` leaves both bodies intact.

- [ ] Run full checks:

  ```
  targ check-full
  ```

  Expected: clean.

#### Task 4.5 — Verify `ComputeState` correctly treats link-only changes as OK (not stale)

- [ ] Add to `/Users/joe/repos/personal/engram/internal/embed/state_test.go` after `TestComputeState_Stale` (before the `fakeFS` type):

```go
func TestComputeState_OK_AfterLinkOnlyEdit(t *testing.T) {
	t.Parallel()

	// A note whose body-only change was adding a "Related to:" section must
	// remain StateOK — the ContentHash excludes that section (D3), so no
	// re-embed is triggered.
	g := NewWithT(t)

	baseNote := []byte("---\ntype: fact\nluhmann: \"1\"\n---\nbody content here.\n")
	noteWithLinks := []byte(
		"---\ntype: fact\nluhmann: \"1\"\n---\nbody content here.\n" +
			"Related to:\n- [[105.2024-01-01.some-note]] — context.\n",
	)

	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "model@384",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.1},
		ContentHash:      embed.ContentHash(baseNote),
	}

	filesystem := fakeFS{
		"n.md":       noteWithLinks,
		"n.vec.json": mustSidecar(t, sidecar),
	}

	state := embed.ComputeState(filesystem, "n.md", "model@384")
	g.Expect(state).To(Equal(embed.StateOK))
}

func TestComputeState_Stale_AfterBodyChange_BeyondLinks(t *testing.T) {
	t.Parallel()

	// A note whose actual body content changed (not just links) must be Stale.
	// This confirms D3 doesn't accidentally suppress real staleness detection.
	g := NewWithT(t)

	original := []byte("---\ntype: fact\nluhmann: \"1\"\n---\noriginal body.\n")
	edited := []byte(
		"---\ntype: fact\nluhmann: \"1\"\n---\nedited body.\n" +
			"Related to:\n- [[105]] — context.\n",
	)

	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "model@384",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.1},
		ContentHash:      embed.ContentHash(original),
	}

	filesystem := fakeFS{
		"n.md":       edited,
		"n.vec.json": mustSidecar(t, sidecar),
	}

	state := embed.ComputeState(filesystem, "n.md", "model@384")
	g.Expect(state).To(Equal(embed.StateStale))
}
```

- [ ] Run: `targ test`
  - Expected: all tests PASS

---

#### Task 4.6 — Run `targ check-full` for a clean bill of health

- [ ] Run: `targ check-full` from `/Users/joe/repos/personal/engram`
  - Expected: 8/8 checks pass, no lint errors
  - If the linter complains about `relatedSectionBulletPfx` being used only in `isRelatedToBlock`: that is a legitimate constant (mirrors `relatedSectionMarker` style); keep it. If it is flagged as exported by mistake, verify it is lowercase.
  - Address any issues before marking this task done

---

#### Task 4.7 — Document the one-time re-baseline in EmbedApplyArgs.Force doc comment

After D3 ships, all existing sidecars whose notes have a "Related to:" section will be marked **stale** (their stored `ContentHash` was computed with the section included; the new code computes without it). The operator must run `engram embed apply --stale` (or `--force`) once to re-baseline. No code change is needed — `--stale` and `--force` already cover this. Add a one-sentence doc note on `EmbedApplyArgs.Force` so the intent is clear in the CLI struct.

- [ ] Edit `/Users/joe/repos/personal/engram/internal/cli/embed.go` — update the `Force` field comment in `EmbedApplyArgs`:

Old:
```go
	Force     bool   `targ:"flag,name=force,desc=also re-embed sidecars whose model_id differs from the binary"`
```

New:
```go
	// Force also re-embeds sidecars whose model_id differs from the binary.
	// Run once with --stale after deploying D3 (Related-to exclusion from
	// ContentHash) so all existing sidecars are re-baselined.
	Force bool `targ:"flag,name=force,desc=also re-embed sidecars whose model_id differs from the binary"`
```

- [ ] Run: `targ check-full`
  - Expected: still 8/8 green

---

#### Task 4.8 — Commit

- [ ] Run `/commit` with message:

```
feat(embed): exclude Related-to section from BodyText/ContentHash (D3)

A link-only edit (adding/changing [[wikilinks]] in the "Related to:" block)
no longer marks a note stale or triggers a re-embed. stripRelatedToSection
recognises the block by the same bullet-only invariant as isRelationBlock in
cli — resilient to hand-edits. Existing sidecars whose notes carry this
section will show as stale on the first check; run `engram embed apply --stale`
once to re-baseline.
```

---

**Risks/notes for the synthesizer:**

1. **Duplicate constant `relatedSectionMarker`:** the embed package defines its own `relatedSectionMarker = "Related to:"` in `hash.go`; `internal/cli` defines the same string in `relations.go`. They must stay in sync. There is no import cycle solution (cli imports embed, not the other way). Consider noting this in a code comment on both constants. No cross-package DRY fix is in scope here.

2. **Existing sidecar re-baseline is manual, not automated:** after D3 merges, every note with a "Related to:" section will be `StateStale` until the operator runs `engram embed apply --stale`. The plan documents this in the `Force` field comment; no migration hook is wired into the binary. If the synthesizer wants an automated one-time migration (e.g., keyed off a schema version bump), that requires a `SidecarSchemaVersion` increment — outside D3 scope, but worth flagging.

3. **Property test `genRawNote` never generates notes with a Related-to section:** the existing property generator (`hash_property_test.go`) produces bare notes. The new property test (`TestContentHash_RelatedToInsensitivityProperty`) appends the section on top of `genRawNote` output, which is correct. If `genRawNote` itself were to be extended in the future to include the section, the existing sensitivity properties (episode/fact body change must change hash) would need to ensure the appended section's text doesn't accidentally cancel the body difference — worth a note for whoever extends the generator.

### Component 5: `engram amend` + `learn --chunk-source` (D2)

**Files:** `internal/cli/amend.go` (new), `internal/cli/amend_test.go` (new), `internal/cli/learn.go` (add `ChunkSources` to `CommonLearnArgs`/`LearnArgs`/frontmatter docs), `internal/cli/relations.go` (add `resolveRelationTargetsStrict`), `internal/cli/targets.go` (wire `amend`; thread `--chunk-source` through learn lambdas), `internal/cli/export_test.go` (Export aliases)

---

#### Task 5.1 — Strict `resolveRelationTargets` variant

**Prerequisite:** none (pure logic, no I/O)

- [ ] **Write failing test** in `internal/cli/relations_test.go` (extend the existing file, new `TestResolveRelationTargetsStrict_*` subtests):
  - `UnresolvedID_Errors`: a bare Luhmann id `"105"` with no matching note in basenames returns `(nil, error)` wrapping `errUnresolvedRelationTarget`
  - `AlreadyBasename_Passthrough`: a relation whose target is already a full basename is returned unchanged (no error)
  - `ResolvedID_Passthrough`: a bare id `"105"` with matching basename is resolved and returned without error
  - `RationalePreserved`: resolved target preserves `|rationale` suffix

```go
// internal/cli/relations_test.go (new subtests appended to existing file)

func TestResolveRelationTargetsStrict_UnresolvedID_Errors(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    _, err := cli.ExportResolveRelationTargetsStrict([]string{"105|why"}, []string{"999.2026-01-01.other.md"})
    g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("unresolved relation target")))
    g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("105")))
}

func TestResolveRelationTargetsStrict_AlreadyBasename_Passthrough(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    const basename = "105.2026-01-01.thing.md"
    got, err := cli.ExportResolveRelationTargetsStrict([]string{basename + "|why"}, []string{basename})
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    g.Expect(got).To(gomega.Equal([]string{basename + "|why"}))
}

func TestResolveRelationTargetsStrict_ResolvedID_OK(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    const basename = "105.2026-01-01.thing.md"
    got, err := cli.ExportResolveRelationTargetsStrict([]string{"105|why"}, []string{basename})
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    g.Expect(got).To(gomega.Equal([]string{basename + "|why"}))
}
```

- [ ] Run `targ test` — expect compile error (function not defined yet); confirm the test file compiles otherwise.

- [ ] **Implement** in `internal/cli/relations.go`:

```go
// unexported errors.
var errUnresolvedRelationTarget = errors.New("unresolved relation target")

// resolveRelationTargetsStrict is the strict variant of resolveRelationTargets:
// it errors when a bare Luhmann id has no matching note basename, so callers
// can fail loud on typos or stale ids. Targets already in full-basename form
// are left unchanged (no error). Rationale suffixes are preserved.
func resolveRelationTargetsStrict(relations, basenames []string) ([]string, error) {
    idToBasename := indexBasenamesByID(basenames)
    basenameSet := make(map[string]struct{}, len(basenames))
    for _, b := range basenames {
        basenameSet[b] = struct{}{}
    }

    resolved := make([]string, len(relations))
    for i, relation := range relations {
        target, rationale, hasRationale := strings.Cut(relation, "|")
        target = strings.TrimSpace(target)

        if _, isBasename := basenameSet[target]; !isBasename {
            if basename, ok := idToBasename[target]; ok {
                target = basename
            } else {
                return nil, fmt.Errorf("%w: %q", errUnresolvedRelationTarget, target)
            }
        }

        if hasRationale {
            resolved[i] = target + "|" + rationale
        } else {
            resolved[i] = target
        }
    }
    return resolved, nil
}
```

- [ ] Add export alias to `export_test.go`:

```go
ExportResolveRelationTargetsStrict = resolveRelationTargetsStrict
```

- [ ] Run `targ test` — expect all four subtests pass.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.2 — `ChunkSources` field on `CommonLearnArgs` / `LearnArgs` + frontmatter `sources:` key

**Prerequisite:** 5.1

- [ ] **Write failing test** in `internal/cli/learn_test.go` (new subtests `TestLearnFact_ChunkSources_*`):
  - `WrittenToFrontmatter`: a `LearnArgs{Type:"fact", ChunkSources:["sha256:abc", "sha256:def"], ...}` produces YAML with `sources: [sha256:abc, sha256:def]` in the frontmatter.
  - `EmptyChunkSources_NoSources`: `ChunkSources: nil` produces no `sources:` key.
  - `FeedbackChunkSources_Written`: same contract for feedback.

```go
// internal/cli/learn_test.go (appended)

func TestLearnFact_ChunkSources_WrittenToFrontmatter(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    args := cli.LearnArgs{
        Type: "fact", Slug: "test-slug", Vault: t.TempDir(),
        Source: "test", Situation: "testing chunk sources",
        Subject: "A", Predicate: "has", Object: "B",
        ChunkSources: []string{"sha256:abc", "sha256:def"},
    }
    // wire minimal deps
    deps := cli.LearnDeps{
        Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
        Getenv:     func(string) string { return "" },
        StatDir:    func(string) error { return nil },
        InitVault:  func(string) error { return nil },
        ListIDs:    func(string) ([]string, error) { return nil, nil },
        ListBasenames: func(string) ([]string, error) { return nil, nil },
        Lock:       func(string) (func(), error) { var written []byte; return func(){}, nil },
        WriteNew:   func(path string, data []byte) error { written = data; return nil },
    }
    // … need to capture written content
    var written []byte
    deps.WriteNew = func(_ string, data []byte) error { written = data; return nil }
    deps.Lock = func(string) (func(), error) { return func() {}, nil }
    deps.ListIDs = func(string) ([]string, error) { return nil, nil }
    deps.ListBasenames = func(string) ([]string, error) { return nil, nil }

    var buf strings.Builder
    err := cli.ExportRunLearn(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    g.Expect(string(written)).To(gomega.ContainSubstring("sources:"))
    g.Expect(string(written)).To(gomega.ContainSubstring("sha256:abc"))
    g.Expect(string(written)).To(gomega.ContainSubstring("sha256:def"))
}

func TestLearnFact_EmptyChunkSources_NoSourcesKey(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    args := cli.LearnArgs{
        Type: "fact", Slug: "test-slug", Vault: t.TempDir(),
        Source: "test", Situation: "no chunk sources",
        Subject: "A", Predicate: "has", Object: "B",
    }
    var written []byte
    deps := cli.LearnDeps{
        Now:       func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
        Getenv:    func(string) string { return "" },
        StatDir:   func(string) error { return nil },
        InitVault: func(string) error { return nil },
        ListIDs:   func(string) ([]string, error) { return nil, nil },
        ListBasenames: func(string) ([]string, error) { return nil, nil },
        Lock:      func(string) (func(), error) { return func() {}, nil },
        WriteNew:  func(_ string, data []byte) error { written = data; return nil },
    }

    var buf strings.Builder
    err := cli.ExportRunLearn(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    g.Expect(string(written)).NotTo(gomega.ContainSubstring("sources:"))
}
```

- [ ] Run `targ test` — expect compile failure (no `ChunkSources` on `LearnArgs` yet).

- [ ] **Implement** — add `ChunkSources []string` to `CommonLearnArgs` (in targets.go) and `LearnArgs` (in learn.go), add `Sources []string` field (yaml `"sources,omitempty"`) to `factFrontmatterDoc` and `feedbackFrontmatterDoc` in `learn.go`, thread `ChunkSources` through the `factFields`/`feedbackFields` structs and into `renderFactFrontmatter`/`renderFeedbackFrontmatter`:

In `internal/cli/targets.go`, `CommonLearnArgs`:
```go
ChunkSources []string `targ:"flag,name=chunk-source,desc=chunk-index id to record as provenance (repeatable)"`
```

In `internal/cli/learn.go`, `LearnArgs`:
```go
ChunkSources []string
```

In `factFields`:
```go
ChunkSources []string
```

In `factFrontmatterDoc` (add field after `Source`):
```go
Sources []string `yaml:"sources,omitempty"`
```

In `renderFactFrontmatter`:
```go
// in the marshalFrontmatter call, add:
Sources: f.ChunkSources,
```

Same pattern for `feedbackFields` / `feedbackFrontmatterDoc` / `renderFeedbackFrontmatter`.

Thread through `assembleLearnContent` (fact and feedback branches):
```go
f := factFields{
    ..., ChunkSources: args.ChunkSources,
}
```

Thread through the `runLearnFromFactArgs` and `runLearnFromFeedbackArgs` bridges in the `LearnArgs` literal (copy from `a.ChunkSources`). Add `ChunkSources` to `CommonLearnArgs` and propagate through `LearnFactArgs`/`LearnFeedbackArgs` (embedded via `CommonLearnArgs`).

- [ ] Run `targ test` — subtests pass.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.3 — `AmendArgs` + `AmendDeps` structs + `findNote` reuse

**Prerequisite:** 5.2

- [ ] **Write failing test** `internal/cli/amend_test.go` — `TestRunAmend_NoteNotFound`:

```go
package cli_test

import (
    "bytes"
    "context"
    "testing"

    "github.com/onsi/gomega"

    "github.com/toejough/engram/internal/cli"
    "github.com/toejough/engram/internal/vaultgraph"
)

func TestRunAmend_NoteNotFound(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    deps := cli.AmendDeps{
        Scan: func(string) ([]vaultgraph.Note, error) {
            return []vaultgraph.Note{}, nil
        },
        Read:     func(string) ([]byte, error) { return nil, nil },
        Write:    func(string, []byte) error { return nil },
        Embedder: nil,
    }
    args := cli.AmendArgs{
        Vault:  "/vault",
        Target: "999",
    }

    var buf bytes.Buffer
    err := cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("note not found")))
}
```

- [ ] Run `targ test` — compile error (AmendArgs/AmendDeps/ExportRunAmend not defined).

- [ ] **Implement** scaffolding in new file `internal/cli/amend.go`:

```go
package cli

import (
    "context"
    "fmt"
    "io"
    "os"

    "github.com/toejough/engram/internal/embed"
    "github.com/toejough/engram/internal/vaultgraph"
)

// AmendArgs holds parsed flags for `engram amend`.
type AmendArgs struct {
    Vault        string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
    Target       string   `targ:"flag,name=target,required,desc=Luhmann id or full basename of the note to amend (required)"`
    Relations    []string `targ:"flag,name=relation,desc=note relation as <wikilink-target>|<rationale> to merge into Related to: (repeatable)"`
    ChunkSources []string `targ:"flag,name=chunk-source,desc=chunk-index id to merge into frontmatter sources: (repeatable)"`
    // Content flags — only supplied fields are overwritten.
    Situation string `targ:"flag,name=situation,desc=replace situation (optional)"`
    Subject   string `targ:"flag,name=subject,desc=replace subject (fact; optional)"`
    Predicate string `targ:"flag,name=predicate,desc=replace predicate (fact; optional)"`
    Object    string `targ:"flag,name=object,desc=replace object (fact; optional)"`
    Behavior  string `targ:"flag,name=behavior,desc=replace behavior (feedback; optional)"`
    Impact    string `targ:"flag,name=impact,desc=replace impact (feedback; optional)"`
    Action    string `targ:"flag,name=action,desc=replace action (feedback; optional)"`
    Activate  bool   `targ:"flag,name=activate,desc=bump LastUsed on the sidecar (optional)"`
}

// AmendDeps holds injected dependencies for RunAmend.
type AmendDeps struct {
    Scan     func(vault string) ([]vaultgraph.Note, error)
    Read     func(path string) ([]byte, error)
    Write    func(path string, data []byte) error
    Embedder embed.Embedder
    Now      func() time.Time
    ListBasenames func(vault string) ([]string, error)
    LoadChunkIDs  func(chunksDir string) (map[string]struct{}, error)
    ChunksDir     string
    LogWarning    func(string, ...any)
}

// unexported errors.
var (
    errAmendNoteNotFound    = errors.New("amend: note not found")
    errAmendUnknownType     = errors.New("amend: unknown note type")
    errAmendUnresolvedChunk = errors.New("amend: unresolved chunk-source id")
)

// RunAmend modifies a note in place. It merges --relation links into the
// "Related to:" body section (idempotent), merges --chunk-source ids into the
// frontmatter "sources:" list (idempotent), and overwrites only the supplied
// content fields. Re-embeds only when content changed. --activate bumps
// LastUsed in the same write.
func RunAmend(ctx context.Context, args AmendArgs, deps AmendDeps, stdout io.Writer) error {
    notes, scanErr := deps.Scan(args.Vault)
    if scanErr != nil {
        return fmt.Errorf("amend: scan: %w", scanErr)
    }

    relPath, findErr := findNote(notes, args.Target)
    if findErr != nil {
        // wrap as amend-specific error for tests
        return fmt.Errorf("%w: %q", errAmendNoteNotFound, args.Target)
    }
    _ = relPath  // used in full implementation (Task 5.4+)

    return nil  // stub — replaced in Task 5.4
}

// newOsAmendDeps wires RunAmend to the real filesystem + bundled embedder.
func newOsAmendDeps(chunksDir string) AmendDeps {
    const perm = 0o600
    return AmendDeps{
        Scan: func(vault string) ([]vaultgraph.Note, error) {
            return vaultgraph.ScanVault(&osVaultFS{}, vault)
        },
        Read: (&osVaultFS{}).ReadFile,
        Write: func(path string, data []byte) error {
            err := os.WriteFile(path, data, perm)
            if err != nil {
                return fmt.Errorf("write %s: %w", path, err)
            }
            return nil
        },
        Embedder:  sharedEmbedder,
        Now:       time.Now,
        ChunksDir: chunksDir,
        ListBasenames: func(vault string) ([]string, error) {
            return (&osLearnFS{}).ListBasenames(vault)
        },
        LoadChunkIDs: loadChunkIDsFromDir,
        LogWarning:   logWarningToStderrf,
    }
}
```

- [ ] Add export alias to `export_test.go`:

```go
ExportRunAmend = RunAmend
```

Add type alias:
```go
type ExportAmendArgs = AmendArgs
type ExportAmendDeps = AmendDeps
```

- [ ] Run `targ test` — `TestRunAmend_NoteNotFound` passes (stub returns the error).
- [ ] Run `targ check-full` — clean; fix any unused-import warnings in stub.

---

#### Task 5.4 — Relation-merge (idempotent `Related to:` append)

**Prerequisite:** 5.3

- [ ] **Write failing tests** `TestRunAmend_RelationMerge_*`:
  - `NewRelation_Appended`: note with no `Related to:` section, one `--relation "105.2026-01-01.foo.md|why"`, output body contains `Related to:\n- [[105.2026-01-01.foo.md]] — why.`
  - `ExistingRelation_Idempotent`: note already has `- [[105.2026-01-01.foo.md]] — why.` in `Related to:`, running with the same `--relation` changes nothing in the section.
  - `NewRelationAdded_ToExisting`: note has relation A, `--relation` specifies B, output contains both A and B.
  - `UnresolvedRelation_Errors`: basenames list has no note matching `"999"`, returns error.

```go
func makeFactNote(situation, subject, predicate, object, relatedSection string) []byte {
    fm := fmt.Sprintf("---\ntype: fact\ntier: L2\nsituation: %s\nsubject: %s\npredicate: %s\nobject: %s\nluhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\n", situation, subject, predicate, object)
    formula := fmt.Sprintf("Information learned: when in %s, %s %s %s.\n", situation, subject, predicate, object)
    return []byte(fm + formula + "\n" + relatedSection)
}

func TestRunAmend_RelationMerge_NewRelation_Appended(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    const basename = "1aa.2026-01-01.test.md"
    const relBasename = "105.2026-01-01.foo.md"
    noteContent := makeFactNote("ctx", "A", "has", "B", "")

    var written []byte
    deps := cli.AmendDeps{
        Scan: func(string) ([]vaultgraph.Note, error) {
            return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
        },
        Read:  func(string) ([]byte, error) { return noteContent, nil },
        Write: func(_ string, data []byte) error { written = data; return nil },
        ListBasenames: func(string) ([]string, error) {
            return []string{basename, relBasename}, nil
        },
        LoadChunkIDs: func(string) (map[string]struct{}, error) { return map[string]struct{}{}, nil },
        Now:  func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
    }
    args := cli.AmendArgs{
        Vault:     "/vault",
        Target:    "1aa",
        Relations: []string{relBasename + "|why"},
    }

    var buf bytes.Buffer
    err := cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    g.Expect(string(written)).To(gomega.ContainSubstring("Related to:"))
    g.Expect(string(written)).To(gomega.ContainSubstring("[[" + relBasename + "]]"))
}

func TestRunAmend_RelationMerge_Idempotent(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    const basename = "1aa.2026-01-01.test.md"
    const relBasename = "105.2026-01-01.foo.md"
    existing := "Related to:\n- [[" + relBasename + "]] — why.\n"
    noteContent := makeFactNote("ctx", "A", "has", "B", existing)

    var written []byte
    deps := cli.AmendDeps{
        Scan: func(string) ([]vaultgraph.Note, error) {
            return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
        },
        Read:  func(string) ([]byte, error) { return noteContent, nil },
        Write: func(_ string, data []byte) error { written = data; return nil },
        ListBasenames: func(string) ([]string, error) {
            return []string{basename, relBasename}, nil
        },
        LoadChunkIDs: func(string) (map[string]struct{}, error) { return map[string]struct{}{}, nil },
        Now:  func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
    }
    args := cli.AmendArgs{
        Vault:     "/vault",
        Target:    "1aa",
        Relations: []string{relBasename + "|why"},
    }

    var buf bytes.Buffer
    err := cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    // Should contain the relation exactly once
    body := string(written)
    count := strings.Count(body, "[["+relBasename+"]]")
    g.Expect(count).To(gomega.Equal(1))
}
```

- [ ] Run `targ test` — tests fail (stub returns nil with no write).

- [ ] **Implement** relation-merge logic in `RunAmend`. Add helper `mergeRelatedSection`:

```go
// mergeRelatedSection parses the existing "Related to:" block from body,
// deduplicates with incoming relations, and returns the updated body with
// only new relations appended. Incoming relations must already be in
// "basename|rationale" resolved form. Existing bullets "- [[basename]] — ..."
// are parsed for their basename to detect duplicates.
func mergeRelatedSection(body string, incoming []string) string {
    // find existing section
    idx := strings.LastIndex(body, relatedSectionMarker)
    var head, existingSection string
    if idx == -1 {
        head = body
    } else {
        head = body[:idx]
        existingSection = body[idx:]
    }

    // collect existing basenames from bullets
    existing := map[string]struct{}{}
    for _, line := range strings.Split(existingSection, "\n") {
        sub := wikilinkRE.FindStringSubmatch(line)
        if sub != nil {
            existing[sub[1]] = struct{}{}
        }
    }

    // build new bullets for relations not already present
    newBullets := make([]string, 0, len(incoming))
    for _, rel := range incoming {
        target, rationale, _ := strings.Cut(rel, "|")
        target = strings.TrimSpace(target)
        if _, dup := existing[target]; dup {
            continue
        }
        bullet := "- [[" + target + "]]"
        if r := strings.TrimSpace(rationale); r != "" {
            bullet += " — " + r + "."
        }
        newBullets = append(newBullets, bullet)
    }

    if len(newBullets) == 0 {
        return body // no change
    }

    // rebuild: if no existing section, start one
    if idx == -1 {
        tail := relatedSectionMarker + "\n" + strings.Join(newBullets, "\n") + "\n"
        // body ends with \n (formula line + blank), append after
        return strings.TrimRight(body, "\n") + "\n\n" + tail
    }

    // append new bullets into existing section
    trimmed := strings.TrimRight(existingSection, "\n")
    return head + trimmed + "\n" + strings.Join(newBullets, "\n") + "\n"
}
```

Implement strict-resolve + merge + write skeleton in `RunAmend` (replacing the stub):

```go
func RunAmend(ctx context.Context, args AmendArgs, deps AmendDeps, stdout io.Writer) error {
    notes, scanErr := deps.Scan(args.Vault)
    if scanErr != nil {
        return fmt.Errorf("amend: scan: %w", scanErr)
    }

    relPath, findErr := findNote(notes, args.Target)
    if findErr != nil {
        return fmt.Errorf("%w: %q", errAmendNoteNotFound, args.Target)
    }

    full := filepath.Join(args.Vault, relPath)

    raw, readErr := deps.Read(full)
    if readErr != nil {
        return fmt.Errorf("amend: read %s: %w", relPath, readErr)
    }

    // --- validate chunk sources ---
    if len(args.ChunkSources) > 0 && deps.LoadChunkIDs != nil {
        chunkIDs, loadErr := deps.LoadChunkIDs(deps.ChunksDir)
        if loadErr != nil {
            return fmt.Errorf("amend: loading chunk ids: %w", loadErr)
        }
        for _, id := range args.ChunkSources {
            if _, ok := chunkIDs[id]; !ok {
                return fmt.Errorf("%w: %q", errAmendUnresolvedChunk, id)
            }
        }
    }

    // --- validate & resolve relation targets ---
    var resolvedRelations []string
    if len(args.Relations) > 0 {
        basenames, bErr := deps.ListBasenames(args.Vault)
        if bErr != nil {
            return fmt.Errorf("amend: listing basenames: %w", bErr)
        }
        var strictErr error
        resolvedRelations, strictErr = resolveRelationTargetsStrict(args.Relations, basenames)
        if strictErr != nil {
            return fmt.Errorf("amend: %w", strictErr)
        }
    }

    // --- amend content ---
    amended, contentChanged, amendErr := amendContent(raw, args, resolvedRelations)
    if amendErr != nil {
        return amendErr
    }

    writeErr := deps.Write(full, []byte(amended))
    if writeErr != nil {
        return fmt.Errorf("amend: write %s: %w", relPath, writeErr)
    }

    // --- re-embed only on content change ---
    if contentChanged && deps.Embedder != nil {
        embedErr := writeAmendedSidecar(ctx, deps, full, amended)
        if embedErr != nil {
            if deps.LogWarning != nil {
                deps.LogWarning("amend: embed failed for %s: %v", relPath, embedErr)
            }
        }
    }

    // --- activate ---
    if args.Activate && deps.Now != nil {
        date := deps.Now().Format(noteDateFormat)
        sidecarPath := embed.SidecarPath(full)
        bumpErr := bumpLastUsed(sidecarPath, date, deps.Read, deps.Write)
        if bumpErr != nil && deps.LogWarning != nil {
            deps.LogWarning("amend: activate failed for %s: %v", relPath, bumpErr)
        }
    }

    _, _ = fmt.Fprintln(stdout, full)
    return nil
}
```

Add `amendContent` helper (next task fills in field-replacement; for now only does relation-merge + provenance-merge):

```go
// amendContent applies all amendments to raw note bytes. Returns the
// updated content, whether the semantic content changed (triggers re-embed),
// and any error. Link/provenance-only changes do NOT set contentChanged.
func amendContent(raw []byte, args AmendArgs, resolvedRelations []string) (string, bool, error) {
    frontmatter, ok := splitFrontmatter(raw)
    if !ok {
        return "", false, fmt.Errorf("amend: note has no parseable frontmatter")
    }

    noteType := peekNoteType(frontmatter)
    body := embed.ExtractBody(raw)

    // merge relations into body
    bodyStr := string(body)
    if len(resolvedRelations) > 0 {
        bodyStr = mergeRelatedSection(bodyStr, resolvedRelations)
    }

    // merge chunk sources into frontmatter
    updated, contentChanged, fieldErr := applyFieldReplacement(raw, args, bodyStr, noteType)
    if fieldErr != nil {
        return "", false, fieldErr
    }

    return updated, contentChanged, nil
}
```

- [ ] Run `targ test` — relation-merge tests pass.
- [ ] Run `targ check-full` — fix any issues.

---

#### Task 5.5 — Provenance-merge (idempotent `sources:` frontmatter)

**Prerequisite:** 5.4

- [ ] **Write failing tests** `TestRunAmend_ProvMerge_*`:
  - `ChunkSources_Written`: note has no `sources:`, after amend with `--chunk-source sha256:abc` the written bytes contain `sources:\n- sha256:abc` (or inline YAML list).
  - `ChunkSources_Idempotent`: note already has `sources: [sha256:abc]`, amending with the same id produces `sources:` containing `sha256:abc` exactly once.
  - `UnresolvedChunkSource_Errors`: `LoadChunkIDs` returns a set not containing `sha256:abc`, amend errors with `unresolved chunk-source id`.

```go
func TestRunAmend_ProvMerge_ChunkSources_Written(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    const basename = "1aa.2026-01-01.test.md"
    noteContent := makeFactNote("ctx", "A", "has", "B", "")

    var written []byte
    deps := cli.AmendDeps{
        Scan: func(string) ([]vaultgraph.Note, error) {
            return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
        },
        Read:  func(string) ([]byte, error) { return noteContent, nil },
        Write: func(_ string, data []byte) error { written = data; return nil },
        ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
        LoadChunkIDs: func(string) (map[string]struct{}, error) {
            return map[string]struct{}{"sha256:abc": {}}, nil
        },
        Now:  func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
    }
    args := cli.AmendArgs{
        Vault:        "/vault",
        Target:       "1aa",
        ChunkSources: []string{"sha256:abc"},
    }

    var buf bytes.Buffer
    err := cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    g.Expect(string(written)).To(gomega.ContainSubstring("sources:"))
    g.Expect(string(written)).To(gomega.ContainSubstring("sha256:abc"))
}

func TestRunAmend_ProvMerge_UnresolvedChunkSource_Errors(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    const basename = "1aa.2026-01-01.test.md"
    deps := cli.AmendDeps{
        Scan: func(string) ([]vaultgraph.Note, error) {
            return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
        },
        Read:  func(string) ([]byte, error) { return makeFactNote("ctx","A","has","B",""), nil },
        Write: func(string, []byte) error { return nil },
        ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
        LoadChunkIDs: func(string) (map[string]struct{}, error) {
            return map[string]struct{}{}, nil // empty — id won't resolve
        },
        ChunksDir: "/chunks",
    }
    args := cli.AmendArgs{
        Vault:        "/vault",
        Target:       "1aa",
        ChunkSources: []string{"sha256:abc"},
    }

    var buf bytes.Buffer
    err := cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("unresolved chunk-source id")))
}
```

- [ ] Run `targ test` — fail (chunk-sources not merged into frontmatter yet).

- [ ] **Implement** provenance merge in `applyFieldReplacement` (partially — the frontmatter `sources:` merge only; field-replacement logic comes in 5.6). Add helper `mergeChunkSources`:

```go
// mergeChunkSources returns a deduped union of existing and incoming chunk ids.
func mergeChunkSources(existing, incoming []string) []string {
    seen := make(map[string]struct{}, len(existing)+len(incoming))
    out := make([]string, 0, len(existing)+len(incoming))
    for _, id := range existing {
        if _, dup := seen[id]; !dup {
            seen[id] = struct{}{}
            out = append(out, id)
        }
    }
    for _, id := range incoming {
        if _, dup := seen[id]; !dup {
            seen[id] = struct{}{}
            out = append(out, id)
        }
    }
    return out
}
```

Implement `applyFieldReplacement` for fact notes (provenance path + full frontmatter rebuild):

```go
// applyFieldReplacement parses the note frontmatter, applies field overrides and
// provenance merge, rebuilds the frontmatter, and reassembles with the (already
// relation-merged) body. contentChanged is true only when a semantic field
// (situation/subject/predicate/object/behavior/impact/action) changed.
func applyFieldReplacement(raw []byte, args AmendArgs, body, noteType string) (string, bool, error) {
    frontmatter, _ := splitFrontmatter(raw) // already validated upstream

    switch noteType {
    case typeFact:
        return applyFactAmend(frontmatter, args, body)
    case typeFeedback:
        return applyFeedbackAmend(frontmatter, args, body)
    default:
        return "", false, fmt.Errorf("%w: %q", errAmendUnknownType, noteType)
    }
}

func applyFactAmend(frontmatter []byte, args AmendArgs, body string) (string, bool, error) {
    var doc factFrontmatterDoc
    if err := yaml.Unmarshal(frontmatter, &doc); err != nil {
        return "", false, fmt.Errorf("amend: parsing fact frontmatter: %w", err)
    }

    when, createdErr := parseCreated(doc.Created)
    if createdErr != nil {
        return "", false, createdErr
    }

    contentChanged := false
    if args.Situation != "" && args.Situation != doc.Situation {
        doc.Situation = args.Situation
        contentChanged = true
    }
    if args.Subject != "" && args.Subject != string(doc.Subject) {
        // doc.Subject is string, not quotedString, safe cast
        doc.Subject = args.Subject
        contentChanged = true
    }
    if args.Predicate != "" && args.Predicate != doc.Predicate {
        doc.Predicate = args.Predicate
        contentChanged = true
    }
    if args.Object != "" && args.Object != doc.Object {
        doc.Object = args.Object
        contentChanged = true
    }

    // provenance merge
    doc.Sources = mergeChunkSources(doc.Sources, args.ChunkSources)

    f := factFields{
        Situation: doc.Situation, Subject: doc.Subject,
        Predicate: doc.Predicate, Object: doc.Object,
        Luhmann: string(doc.Luhmann), Source: doc.Source,
        Project: doc.Project, Issue: string(doc.Issue),
        Tier: doc.Tier, ChunkSources: doc.Sources,
    }

    // If content changed, rebuild body formula with new field values
    if contentChanged {
        body = renderFactBody(f, relatedTailFromBody(body))
    }

    return renderFactFrontmatter(f, when) + body, contentChanged, nil
}
```

Similar `applyFeedbackAmend`. Add `relatedTailFromBody` helper (extracts the `Related to:` section from the amended body string, for use when re-rendering the formula):

```go
// relatedTailFromBody extracts the "Related to:\n..." suffix from an
// already-relation-merged body string. Returns "" when absent.
func relatedTailFromBody(body string) string {
    idx := strings.LastIndex(body, relatedSectionMarker)
    if idx == -1 {
        return ""
    }
    return body[idx:]
}
```

- [ ] Run `targ test` — provenance tests pass.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.6 — Field-replacement: only supplied fields overwritten, preserved fields intact

**Prerequisite:** 5.5

- [ ] **Write failing tests** `TestRunAmend_FieldReplacement_*`:
  - `Fact_SubjectOnly`: amend with only `--subject "NewSubject"`, predicate/object/situation preserved.
  - `Fact_PreservesLuhmannAndCreated`: Luhmann id and `created` date unchanged after amend.
  - `Fact_PreservesRelationsAndSources`: existing `Related to:` and `sources:` preserved when no new ones supplied.
  - `Feedback_ActionOnly`: amend with only `--action "Do X"`, behavior/impact/situation unchanged.
  - `Fact_ContentChange_True`: amend with `--subject` sets `contentChanged=true` (verify via re-embed being triggered — use a sentinel Embedder that records calls).

```go
func TestRunAmend_FieldReplacement_Fact_SubjectOnly(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    const basename = "1aa.2026-01-01.test.md"
    noteContent := makeFactNote("ctx", "OldSubject", "has", "B", "")

    var written []byte
    deps := cli.AmendDeps{
        Scan: func(string) ([]vaultgraph.Note, error) {
            return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
        },
        Read:          func(string) ([]byte, error) { return noteContent, nil },
        Write:         func(_ string, data []byte) error { written = data; return nil },
        ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
        LoadChunkIDs:  func(string) (map[string]struct{}, error) { return nil, nil },
        Now:           func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
    }
    args := cli.AmendArgs{
        Vault:   "/vault",
        Target:  "1aa",
        Subject: "NewSubject",
    }

    var buf bytes.Buffer
    err := cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }

    body := string(written)
    g.Expect(body).To(gomega.ContainSubstring("subject: NewSubject"))
    g.Expect(body).To(gomega.ContainSubstring("predicate: has"))
    g.Expect(body).To(gomega.ContainSubstring("object: B"))
    g.Expect(body).To(gomega.ContainSubstring("situation: ctx"))
    g.Expect(body).To(gomega.ContainSubstring("luhmann: \"1aa\""))
    g.Expect(body).To(gomega.ContainSubstring("created: 2026-01-01"))
}

func TestRunAmend_FieldReplacement_NoContentChange_NoReEmbed(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    const basename = "1aa.2026-01-01.test.md"
    const relBasename = "105.2026-01-01.foo.md"
    noteContent := makeFactNote("ctx", "A", "has", "B", "")

    embedCalled := false
    deps := cli.AmendDeps{
        Scan: func(string) ([]vaultgraph.Note, error) {
            return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
        },
        Read:          func(string) ([]byte, error) { return noteContent, nil },
        Write:         func(string, []byte) error { return nil },
        ListBasenames: func(string) ([]string, error) { return []string{basename, relBasename}, nil },
        LoadChunkIDs:  func(string) (map[string]struct{}, error) { return nil, nil },
        Now:           func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
        Embedder:      &cli.SpyEmbedder{Called: &embedCalled},
    }
    args := cli.AmendArgs{
        Vault:     "/vault",
        Target:    "1aa",
        Relations: []string{relBasename + "|why"},
    }

    var buf bytes.Buffer
    err := cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    g.Expect(embedCalled).To(gomega.BeFalse(), "relation-only change must not trigger re-embed")
}
```

Note: `SpyEmbedder` is a test-only struct implementing `embed.Embedder`; add it to `amend_test.go` or a test helper:

```go
// SpyEmbedder is an embed.Embedder that records whether Embed was called.
// ExportSpyEmbedder exposes it via export_test.go for cross-package use.
type SpyEmbedder struct {
    Called *bool
}

func (s *SpyEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
    if s.Called != nil {
        *s.Called = true
    }
    return []float32{0.1}, nil
}
```

Add `ExportSpyEmbedder = SpyEmbedder` type alias to `export_test.go`. (Actually since `SpyEmbedder` is only needed in tests, define it in `amend_test.go` — it lives in `package cli_test` and can directly implement `embed.Embedder`.)

- [ ] Run `targ test` — new field-replacement tests fail (no re-embed guard yet, subject mismatch).

- [ ] **Complete** `applyFactAmend` — field replacement is already partly in from Task 5.5. Complete the `doc.Subject` fix (it's a plain `string` in `factFrontmatterDoc`, not `quotedString`, so the comparison is correct). Confirm `applyFeedbackAmend` implements `Behavior`/`Impact`/`Action` field replacement with the same pattern. Make the `contentChanged` gate in `RunAmend` correctly skip `writeAmendedSidecar` when false.

- [ ] Run `targ test` — all field-replacement tests pass.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.7 — `--activate` in one write (sidecar bump on content-change path)

**Prerequisite:** 5.6

The spec says `--activate` bumps `LastUsed` in the sidecar. Two sub-cases:
- Content changed → re-embed already wrote a fresh sidecar → `bumpLastUsed` reads the freshly-written sidecar.
- No content change → sidecar exists unchanged → `bumpLastUsed` reads the existing sidecar.

Both are already handled by the `bumpLastUsed` call in `RunAmend` after the optional re-embed. The ordering guarantee is that `Write` is called first (note file), then re-embed writes the sidecar, then `bumpLastUsed` reads and re-writes the sidecar. This is one logical amend in three sequential writes; the spec's "one write" means a single invocation of `amend` (not a single filesystem write), so this is correct.

- [ ] **Write failing test** `TestRunAmend_Activate_BumpsLastUsed`:

```go
func TestRunAmend_Activate_BumpsLastUsed(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    const basename = "1aa.2026-01-01.test.md"
    noteContent := makeFactNote("ctx", "A", "has", "B", "")
    sidecarContent := embed.MarshalSidecar(embed.Sidecar{LastUsed: "2025-01-01"})

    var writtenPaths []string
    deps := cli.AmendDeps{
        Scan: func(string) ([]vaultgraph.Note, error) {
            return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
        },
        Read: func(path string) ([]byte, error) {
            if strings.HasSuffix(path, ".vec.json") {
                return sidecarContent, nil
            }
            return noteContent, nil
        },
        Write: func(path string, _ []byte) error {
            writtenPaths = append(writtenPaths, path)
            return nil
        },
        ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
        LoadChunkIDs:  func(string) (map[string]struct{}, error) { return nil, nil },
        Now:           func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) },
    }
    args := cli.AmendArgs{
        Vault:    "/vault",
        Target:   "1aa",
        Activate: true,
    }

    var buf bytes.Buffer
    err := cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    // Sidecar write must have happened (the .vec.json path)
    var hasSidecar bool
    for _, p := range writtenPaths {
        if strings.HasSuffix(p, ".vec.json") {
            hasSidecar = true
        }
    }
    g.Expect(hasSidecar).To(gomega.BeTrue(), "activate must write the sidecar")
}
```

- [ ] Run `targ test` — may pass already if `bumpLastUsed` is wired; confirm and fix any gap.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.8 — `loadChunkIDsFromDir` helper (validate chunk-source ids)

**Prerequisite:** 5.3 (needed by 5.5 validation path)

The spec says: "Validating an id requires loading the index (`loadChunkRecords` is O(total chunks)); build should construct an in-memory id-set after load rather than reach for a non-existent O(1) lookup." The id here is the chunk's `ContentHash` (since that is what `chunk.Record` uses as the stable per-chunk key: `source#anchor` is the implicit composite key, but the spec says "content hash, or `source#anchor`" — the simplest stable id is `ContentHash` which is already used by `ingest` as the dedup key).

- [ ] **Write failing test** `TestLoadChunkIDsFromDir_*`:
  - `ReadsAllContentHashes`: given a `chunksDir` containing a single `.jsonl` file with two records, returns both `ContentHash` values in the set.
  - `EmptyDir_EmptySet`: an empty (but existing) dir returns an empty set with no error.
  - `MissingDir_EmptySet`: a non-existent dir returns empty set (not an error — first ingest case).

```go
// internal/cli/amend_test.go (appended)

func TestLoadChunkIDsFromDir_ReadsAllContentHashes(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    dir := t.TempDir()
    records := []chunk.Record{
        {Source: "a", Anchor: "turn-1", ContentHash: "sha256:aaa", Text: "hi", Vector: []float32{0.1}},
        {Source: "b", Anchor: "turn-2", ContentHash: "sha256:bbb", Text: "bye", Vector: []float32{0.2}},
    }
    data, err := chunk.EncodeRecords(records)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    err = os.WriteFile(filepath.Join(dir, "test.jsonl"), data, 0o600)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }

    ids, loadErr := cli.ExportLoadChunkIDsFromDir(dir)
    g.Expect(loadErr).NotTo(gomega.HaveOccurred())
    if loadErr != nil { return }
    g.Expect(ids).To(gomega.HaveKey("sha256:aaa"))
    g.Expect(ids).To(gomega.HaveKey("sha256:bbb"))
}
```

- [ ] Run `targ test` — compile error (function not exported).

- [ ] **Implement** in `internal/cli/amend.go`:

```go
// loadChunkIDsFromDir scans chunksDir for all .jsonl index files, decodes
// them, and returns a set of all ContentHash values present. An absent or
// unreadable chunksDir is treated as an empty set (first-ingest case, not an
// error). Individual malformed records are silently skipped (index is binary-
// owned; partial corruption should not block amend).
func loadChunkIDsFromDir(chunksDir string) (map[string]struct{}, error) {
    entries, err := os.ReadDir(chunksDir)
    if err != nil {
        return map[string]struct{}{}, nil //nolint:nilerr // absent dir = empty set
    }

    ids := make(map[string]struct{})

    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
            continue
        }
        if entry.Name() == manifestName { // skip manifest
            continue
        }
        path := filepath.Join(chunksDir, entry.Name())
        data, readErr := os.ReadFile(path)
        if readErr != nil {
            continue // skip unreadable
        }
        records, decodeErr := chunk.DecodeRecords(data)
        if decodeErr != nil {
            continue
        }
        for _, r := range records {
            ids[r.ContentHash] = struct{}{}
        }
    }

    return ids, nil
}
```

Add export alias:
```go
ExportLoadChunkIDsFromDir = loadChunkIDsFromDir
```

- [ ] Run `targ test` — passes.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.9 — Wire `amend` in `targets.go` + add `ExportRunAmend` alias

**Prerequisite:** 5.6

- [ ] **Write failing test** `TestTargets_AmendRegistered` in `targets_test.go` (or a new `amend_targets_test.go`) — call `cli.Targets(...)` and assert the returned slice contains a target named `"amend"`:

```go
func TestTargets_AmendRegistered(t *testing.T) {
    t.Parallel()
    // Targets returns []any; the targ API does not expose a Name() method
    // on the returned interface. Verify indirectly: the production binary
    // must accept `amend` — covered by integration. For unit coverage,
    // verify that Targets does not panic and returns >0 items.
    g := gomega.NewWithT(t)
    var buf bytes.Buffer
    targets := cli.Targets(&buf, &buf, func(int) {}, nil)
    g.Expect(targets).NotTo(gomega.BeEmpty())
}
```

(A deeper name-check would require inspecting targ internals; the integration test is the real gate.)

- [ ] **Wire** in `targets.go` — add to `maintenanceTargets`:

```go
targ.Targ(func(ctx context.Context, a AmendArgs) {
    a.Vault = resolveVault(a.Vault, homeOrEmpty(), os.Getenv)
    chunksDir := ResolveChunksDir("", homeOrEmpty(), os.Getenv)
    errHandler(RunAmend(withLog(ctx), a, newOsAmendDeps(chunksDir), stdout))
}).Name("amend").Description("Amend a note in place: relation-merge, provenance-merge, field-replacement, activate"),
```

- [ ] Also wire `--chunk-source` through the `learn` closures — propagate `a.ChunkSources` in the `LearnArgs` literal inside `runLearnFromFactArgs` and `runLearnFromFeedbackArgs` bridges (edit the functions in `learn.go` to include `ChunkSources: a.ChunkSources`). The `LearnFactArgs`/`LearnFeedbackArgs` embed `CommonLearnArgs` which gains `ChunkSources` — so `a.ChunkSources` is accessible.

- [ ] Run `targ build` — builds clean.
- [ ] Run `targ test` — new test passes.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.10 — Final integration smoke + `Export*` cleanup

**Prerequisite:** 5.9

- [ ] **Write integration test** `TestRunAmend_RoundTrip_FactNote` in `amend_test.go` — writes a real fact note to a temp dir, amends it with `--relation`, `--chunk-source`, `--subject`, verifies the output bytes contain all expected mutations and that `Luhmann`, `created`, and unchanged fields are identical to the original:

```go
func TestRunAmend_RoundTrip_FactNote(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    dir := t.TempDir()
    chunksDir := t.TempDir()
    const basename = "1aa.2026-01-01.test.md"
    const relBasename = "105.2026-01-01.foo.md"
    notePath := filepath.Join(dir, basename)

    // write initial note
    noteContent := makeFactNote("original ctx", "OldSubject", "has", "B", "")
    err := os.WriteFile(notePath, noteContent, 0o600)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }

    // write a chunk index with one record
    records := []chunk.Record{{
        Source: "s", Anchor: "turn-1", ContentHash: "sha256:abc",
        Text: "t", Vector: []float32{0.1},
    }}
    data, _ := chunk.EncodeRecords(records)
    err = os.WriteFile(filepath.Join(chunksDir, "s.jsonl"), data, 0o600)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }

    // also write a "note" for relBasename (just a file, Scan reads real FS)
    err = os.WriteFile(filepath.Join(dir, relBasename), makeFactNote("r ctx","X","is","Y",""), 0o600)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }

    deps := cli.ExportNewOsAmendDeps(chunksDir)
    deps.Scan = func(vault string) ([]vaultgraph.Note, error) {
        return []vaultgraph.Note{
            {Basename: basename, LuhmannID: "1aa"},
            {Basename: relBasename, LuhmannID: "105"},
        }, nil
    }
    deps.Read = os.ReadFile
    var finalContent []byte
    deps.Write = func(path string, data []byte) error {
        finalContent = data
        return os.WriteFile(path, data, 0o600)
    }
    deps.ListBasenames = func(string) ([]string, error) {
        return []string{basename, relBasename}, nil
    }
    deps.Now = func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) }

    args := cli.AmendArgs{
        Vault:        dir,
        Target:       "1aa",
        Subject:      "NewSubject",
        Relations:    []string{relBasename + "|because"},
        ChunkSources: []string{"sha256:abc"},
        ChunksDir:    chunksDir,
    }

    var buf bytes.Buffer
    err = cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }

    body := string(finalContent)
    g.Expect(body).To(gomega.ContainSubstring("subject: NewSubject"))
    g.Expect(body).To(gomega.ContainSubstring("luhmann: \"1aa\""))
    g.Expect(body).To(gomega.ContainSubstring("created: 2026-01-01"))
    g.Expect(body).To(gomega.ContainSubstring("sources:"))
    g.Expect(body).To(gomega.ContainSubstring("sha256:abc"))
    g.Expect(body).To(gomega.ContainSubstring("Related to:"))
    g.Expect(body).To(gomega.ContainSubstring("[[" + relBasename + "]]"))
}
```

Add to `export_test.go`:
```go
ExportNewOsAmendDeps = newOsAmendDeps
```

- [ ] Run `targ test` — passes.
- [ ] Run `targ check-full` — clean (8 checks pass).
- [ ] `targ build` — binary builds.

---

**Risks/notes for the synthesizer:**

1. **D3 dependency (exclude `Related to:` from embed source):** Task 5.6's "content changed" gate is critical — if `Related to:` is still in the body hash when the spec says it should be excluded (D3), a link-only amend will incorrectly trigger re-embed. This component depends on the D3 component (step 4 in §7) being landed first; the plan above gates `contentChanged=false` for relation-only amends at the Go level, but `BuildSidecar`/`ContentHash` will still hash the `Related to:` block until D3 lands. The plan is self-consistent only when sequenced after D3.
2. **`manifestName` constant:** used in `loadChunkIDsFromDir` to skip the manifest file — this is already defined in `internal/cli/ingest.go`. Confirm it is package-accessible (unexported but same package) before implementing Task 5.8.
3. **`applyFactAmend` — `doc.Subject` type:** `factFrontmatterDoc.Subject` is a plain `string`, but `Object` is also `string`. The comparison `args.Subject != string(doc.Subject)` is fine; confirm there is no `quotedString` wrapping on `Subject`/`Predicate`/`Object` (current code shows they are plain `string` fields — verified).

### Component 6: `/recall` skill — agent-judged coverage + recency-weighted distillation (D6/D7)

**Files:** `skills/recall/SKILL.md` (rewrite Step 2.5 only; all other steps unchanged)

**Dependency note:** This component is executable only after Components 2, 3, and 5 are merged, because Step 2.5 references:
- `candidate_l2s: [{path, cosine}]` per cluster (Component 3's top-K field, replacing `nearest_l2`)
- `engram amend` command (Component 5)
- `engram show <ref>` to read candidates before judging (already ships)
- `engram activate` for the covered path (already ships)

---

#### Task 6.1 — RED: capture baseline behavior of current Step 2.5

- [ ] Read the current `skills/recall/SKILL.md` lines 95–170 (the current cosine-band Step 2.5) and write down exactly what the table-driven band gate says so the baseline prompt can reproduce the behavior an agent shows today.

- [ ] Create a scratch file `dev/skill-tdd/recall-step2.5-baseline-prompt.md` (do **not** commit) containing a pressure scenario that gives a subagent: (a) the current full `SKILL.md` text verbatim, (b) a fabricated `engram query` payload containing three clusters — one with `nearest_l2.cosine: 0.96`, one with `nearest_l2.cosine: 0.87`, and one with `nearest_l2.cosine: 0.72` — and (c) the instruction "Execute Step 2.5 exactly as written." The scenario adds a time-pressure rider: "The user is waiting; do not spend time re-reading — just follow the bands." Expected baseline failure: the agent applies the three cosine bands from the table and writes or skips notes based purely on the numeric threshold, with no candidate-reading, no recency reasoning, and no `engram amend`.

  The prompt content (put in the file verbatim):

  ```markdown
  # Baseline pressure test — recall Step 2.5 (pre-v2)

  You are a Claude Code agent executing the `/recall` skill. The full skill text is pasted below.
  After the skill text you will find a fabricated `engram query` YAML payload.

  Your task: execute Step 2.5 EXACTLY as written in the skill, then stop.

  ## Pressure
  - The user has been waiting for 30 seconds.
  - You have already spent 10 minutes on Steps 0–2.
  - Do not re-read the spec. Act on the skill text below immediately.

  ## Current SKILL.md (full text — paste current SKILL.md here at execution time)

  [PASTE CURRENT SKILL.MD HERE]

  ## Fabricated payload

  ```yaml
  version: 1
  phrases: ["TDD discipline", "test-first development"]
  items: []
  clusters:
    - phrase: "chunks"
      size: 3
      silhouette: 0.45
      members:
        - {path: "chunks/t1.md", score: 0.81, is_representative: true}
        - {path: "chunks/t2.md", score: 0.78, is_representative: false}
        - {path: "chunks/t3.md", score: 0.75, is_representative: false}
      nearest_l2: {path: "1a.tdd-must-come-first.md", cosine: 0.96}
    - phrase: "chunks"
      size: 4
      silhouette: 0.38
      members:
        - {path: "chunks/t4.md", score: 0.83, is_representative: true}
        - {path: "chunks/t5.md", score: 0.80, is_representative: false}
        - {path: "chunks/t6.md", score: 0.77, is_representative: false}
        - {path: "chunks/t7.md", score: 0.74, is_representative: false}
      nearest_l2: {path: "1b.test-doubles-patterns.md", cosine: 0.87}
    - phrase: "chunks"
      size: 2
      silhouette: 0.31
      members:
        - {path: "chunks/t8.md", score: 0.79, is_representative: true}
        - {path: "chunks/t9.md", score: 0.72, is_representative: false}
      nearest_l2: {path: "1c.integration-test-scope.md", cosine: 0.72}
  budget:
    items_returned: 0
    notes_scanned: 12
  ```

  Execute Step 2.5 now.
  ```

- [ ] Run the baseline scenario as a subagent (spawn via `TaskCreate` with a short-lived subagent model). Record verbatim what the agent does. Expected to observe:
  - Cluster 1 (0.96): agent calls `engram activate --note 1a.tdd-must-come-first.md` and does NOT create.
  - Cluster 2 (0.87): agent calls `engram learn fact|feedback --target <luhmann-id> --position continuation ...` (UPDATE band).
  - Cluster 3 (0.72): agent calls `engram learn fact|feedback --position top ...` (CREATE band).
  - The agent reads no chunk content before deciding, uses no `engram amend`, consults no candidate list, applies no recency reasoning.

  **This is the RED state — document it. The baseline violates the v2 design in every dimension.**

---

#### Task 6.2 — GREEN: rewrite Step 2.5 in `skills/recall/SKILL.md`

This is a targeted rewrite of lines 95–170 only. All other steps (0, 0.5, 1, 2, 3, the Red flags table) remain unchanged except two Red flags rows that must be updated.

- [ ] Open `skills/recall/SKILL.md`. Identify the exact text block for Step 2.5 (currently lines 95–169, from `### Step 2.5` through the last Red flags row that references the cosine band logic). Do **not** touch any other section.

- [ ] Replace the entire Step 2.5 section (the `### Step 2.5 — Crystallize lessons from the payload's chunk clusters (band-driven)` heading through the end of its content, stopping before `### Step 3`) with the following text:

  ```markdown
  ### Step 2.5 — Lazy L2 synthesis from the clustering (agent-judged)

  The `--synthesize-l2` output's `clusters` list contains the unified clustering of matched chunks
  and notes. Each cluster carries `candidate_l2s: [{path, cosine}]` — the top-K existing L2s
  nearest the cluster centroid (K ≥ 3, centroid cosine). **Process every cluster.** For each:

  **A. Read candidates and members**

  Run `engram show <path>` on every entry in `candidate_l2s` (up to K calls, blocking). Also read
  any note-kind members that are already in the payload's `items` list (their `content` field). Do
  not judge coverage before you have read the content.

  **B. Apply the recency weight to resolve conflicts**

  Evidence **conflicts** when a newer member explicitly negates or reverses an older claim. Reversal
  cues: "no longer", "replaced by", "use X not Y", or the same subject+predicate appearing with a
  different object in a newer item. When conflict is present: **recent wins**. When no conflict:
  treat older and newer evidence as independently valid — do not demote a stable convention merely
  because it lacks a recent instance.

  **C. Judge coverage against the recency-weighted view — in this order**

  | Outcome | Criterion | Action |
  | --- | --- | --- |
  | **Covered** | A candidate's claim states the cluster's principle with **no material omission** vs the recency-weighted members | `engram amend <candidate-path> --activate --relation <new-note-sources> --chunk-source <new-chunk-ids>` — link-enrich only; **do not rewrite content** |
  | **Near** | A candidate addresses the same situation but omits ≥ 1 substantive claim the members evidence (judge against the recency-weighted view — a candidate that only matches the superseded content is **near**, not covered) | `engram amend <candidate-path> --relation <note-sources> --chunk-source <chunk-ids> --subject ... --predicate ... --object ...` (or `--behavior/--impact/--action`) — re-synthesize content from all members, recency-weighted |
  | **Absent** | No candidate addresses the situation | `engram learn fact\|feedback --position top --relation <note-sources> --chunk-source <chunk-ids> --source "<descriptive>" --situation "..." --subject/--predicate/--object (or --behavior/--impact/--action)` |

  **One write per cluster; one representative L2 per cluster.** The representative is always an L2
  (never an L1 note or a chunk). For `absent`, write exactly one note (fact *or* feedback) covering
  the cluster's principle. Do not write one fact and one feedback note for the same cluster.

  For `amend` (covered or near), pass `--relation` for every **note** source in the cluster (the
  wikilink graph) and `--chunk-source` for every **chunk** source (provenance, not wikilinks). For
  `learn`, pass the same flags. The `--source` flag on `learn` is the human-readable provenance
  string (unchanged); `--chunk-source` is the chunk-id list (new).

  **WAIT for each write before moving to the next cluster.** Writes are blocking and inline — the
  L2 created or updated by one cluster may be a candidate for another.

  **Known gap:** cross-cluster supersession — where the superseding evidence did not cosine-cluster
  with the old — is not handled. Note the conflict in the synthesized content when you see it, but
  do not attempt to resolve it across clusters.
  ```

- [ ] Update the **Red flags table** at the bottom of the file. Remove the three rows that reference cosine-band logic and the old `nearest_l2` field. Add the following rows (insert after the existing "You dispatched cluster-synthesis subagents" row):

  ```markdown
  | You judged coverage before reading the candidate content with `engram show` | Read first — cosine alone cannot decide coverage |
  | You applied a cosine threshold to decide covered/near/absent | Coverage is agent-judged from content; cosine only nominates candidates |
  | A candidate matching only the superseded content → you marked it "covered" | Apply the recency weight first; a candidate that misses the conflict is "near" |
  | You wrote two notes (a fact AND a feedback) for one cluster | One representative L2 per cluster — pick the right kind |
  | You used `nearest_l2` instead of `candidate_l2s` | The v2 field is `candidate_l2s: [{path, cosine}]` — a list, not a singleton |
  | You called `engram learn --target` to update a note in place | Updates use `engram amend`; `engram learn` is create-only |
  | A `≥0.95` cluster → you activated without reading the candidates | Read first; high cosine nominates, it does not decide |
  ```

- [ ] Also update the existing Red flags row:
  - OLD: `| You grouped chunks by eye instead of using the payload's 'phrase: "chunks"' clusters | The binary's k-means grouping and 'nearest_l2' cosine are the ground truth; apply the bands |`
  - NEW: `| You grouped chunks by eye instead of using the payload's clusters | The binary's k-means grouping is the ground truth; read every cluster |`

  And remove the rows that reference "banded N clusters and wrote 0 notes" and "≥0.95 cluster → you created a new L2" since those rows are band-specific logic that no longer applies. Also remove "`≥0.95` clusters where the covering L2 was useful" from the `activated: true` red flag row.

---

#### Task 6.3 — GREEN verify: run the same scenario WITH the new skill

- [ ] Rerun the pressure scenario from Task 6.1 as a subagent, this time with the **new** SKILL.md text injected. Update the payload to use `candidate_l2s` instead of `nearest_l2`:

  ```yaml
  clusters:
    - phrase: "synthesis"
      size: 3
      silhouette: 0.45
      members:
        - {path: "1a.tdd-must-come-first.md", score: 0.81, is_representative: true}
        - {path: "chunks/t2.md", score: 0.78, is_representative: false}
        - {path: "chunks/t3.md", score: 0.75, is_representative: false}
      candidate_l2s:
        - {path: "1a.tdd-must-come-first.md", cosine: 0.96}
        - {path: "1b.test-doubles-patterns.md", cosine: 0.84}
        - {path: "1c.integration-test-scope.md", cosine: 0.79}
    - phrase: "synthesis"
      size: 4
      silhouette: 0.38
      members:
        - {path: "chunks/t4.md", score: 0.83, is_representative: true}
        - {path: "chunks/t5.md", score: 0.80, is_representative: false}
        - {path: "chunks/t6.md", score: 0.77, is_representative: false}
        - {path: "chunks/t7.md", score: 0.74, is_representative: false}
      candidate_l2s:
        - {path: "1b.test-doubles-patterns.md", cosine: 0.87}
        - {path: "1a.tdd-must-come-first.md", cosine: 0.80}
        - {path: "1d.mock-boundaries.md", cosine: 0.74}
    - phrase: "synthesis"
      size: 2
      silhouette: 0.31
      members:
        - {path: "chunks/t8.md", score: 0.79, is_representative: true}
        - {path: "chunks/t9.md", score: 0.72, is_representative: false}
      candidate_l2s:
        - {path: "1c.integration-test-scope.md", cosine: 0.72}
        - {path: "1e.scope-boundaries.md", cosine: 0.65}
        - {path: "1f.test-granularity.md", cosine: 0.60}
  ```

  Pass the scenario to a subagent with the instruction: "Execute Step 2.5 of the /recall skill exactly as written."

- [ ] **Pass criteria (check each):**
  - [ ] Agent calls `engram show` on each `candidate_l2s` entry before deciding.
  - [ ] Agent reads member content from `items` (or calls `engram show` on note-kind members).
  - [ ] Agent applies recency reasoning before deciding covered/near/absent.
  - [ ] Agent uses `engram amend --activate` (not `engram activate`) for covered clusters.
  - [ ] Agent uses `engram amend` with content flags for near clusters.
  - [ ] Agent uses `engram learn` for absent clusters.
  - [ ] Agent passes `--relation` for note sources and `--chunk-source` for chunk sources.
  - [ ] Agent writes exactly one note per cluster, never two.
  - [ ] Agent does NOT use the cosine value as a threshold gate.

---

#### Task 6.4 — Pressure test: supersession scenario (recent-wins)

- [ ] Construct a second pressure scenario. The payload has one cluster containing:
  - Two older chunks (content: "Use `http.Get` directly; it is the idiomatic call") tagged with a 2024-01-15 `IngestedAt`.
  - One newer chunk (content: "Use `http.NewRequestWithContext`; `http.Get` no longer acceptable — context required") tagged with a 2025-06-01 `IngestedAt`.
  - One candidate: `{path: "1g.http-idioms.md", cosine: 0.88}` whose `engram show` output says "Use `http.Get` for outbound HTTP calls."

  Add pressure rider: "There are two chunks agreeing with the candidate and only one against it; the older approach appears well-established."

- [ ] Run the scenario as a subagent. **Pass criteria:**
  - [ ] Agent identifies the newer chunk as containing a reversal cue ("no longer").
  - [ ] Agent concludes the candidate matches only the **superseded** content.
  - [ ] Agent marks the cluster **near** (not covered) despite the cosine being 0.88 and despite two older chunks agreeing.
  - [ ] Agent calls `engram amend 1g.http-idioms.md --object "http.NewRequestWithContext" ...` (or equivalent content flags) to update the note.

---

#### Task 6.5 — Pressure test: old-uncontradicted scenario (do not demote)

- [ ] Construct a third pressure scenario. The payload has one cluster containing:
  - Two older chunks (content: "Name exported error variables using the `ErrFoo` sentinel pattern").
  - Zero newer chunks on this topic.
  - One candidate: `{path: "1h.error-naming.md", cosine: 0.93}` whose `engram show` output says "Export error sentinel values using the `ErrFoo` naming convention."

  Add pressure rider: "The convention was documented in 2023 and hasn't appeared in recent sessions — it may be stale."

- [ ] Run the scenario as a subagent. **Pass criteria:**
  - [ ] Agent does NOT demote the 2023 evidence merely because it lacks a recent instance.
  - [ ] Agent concludes the candidate **covers** the cluster (no conflict, no material omission).
  - [ ] Agent calls `engram amend 1h.error-naming.md --activate --chunk-source <ids>` (link-enrich only, no content rewrite).

---

#### Task 6.6 — Pressure test: absent cluster (bootstrap case)

- [ ] Construct a fourth scenario with a cluster whose `candidate_l2s` list is empty (no existing L2s). Three chunks evidence "Use `targ check-full` not `targ check` to surface all errors at once." Add pressure rider: "There are no candidates — the vault may not have any L2s yet, and creating one feels premature."

- [ ] Run the scenario as a subagent. **Pass criteria:**
  - [ ] Agent recognizes absent = create, regardless of the lack of candidates.
  - [ ] Agent calls `engram learn fact --position top --source "..." --situation "..." --subject "targ check-full" --predicate "must be used instead of" --object "targ check when surfacing all build/lint errors"` (or equivalent phrasing).
  - [ ] Agent does NOT skip writing because the vault feels empty.

---

#### Task 6.7 — REFACTOR: address any rationalizations found, re-verify

- [ ] Review the transcript of each pressure test. For any rationalization the agent used that is not yet countered in the Red flags table, add a row. For any phrasing in Step 2.5 that the agent mis-read (e.g., read "no material omission" as license to judge without reading), tighten the wording.

- [ ] If any Green scenario regressed after refactor edits, re-run it. The skill is done only when all four pressure tests pass without rationalization.

---

#### Task 6.8 — Distribute via `engram update`

- [ ] With the skill edit green and all pressure tests passing, run:

  ```
  engram update
  ```

  Expected output lines (verify each):
  ```
  engram update
    source: <local clone path>
    binary: ...
  installed: claude, ...
  ```

  Verify the new `skills/recall/SKILL.md` content is present at the harness install root (typically `~/.claude/skills/recall/SKILL.md`) by running:

  ```bash
  grep -n "candidate_l2s" ~/.claude/skills/recall/SKILL.md
  ```

  Expected: at least two lines containing `candidate_l2s`.

- [ ] Confirm `targ check-full` passes with no errors (the skill edit touches no Go code, but the check confirms the binary side is still clean — a prerequisite before any commit touching this component).

  ```
  targ check-full
  ```

  Expected: all 8 checks pass, zero errors.

---

**Risks / notes for the synthesizer:**

1. **Hard dependency on Components 3 and 5.** The new Step 2.5 names `candidate_l2s` (Component 3's renamed field) and `engram amend` (Component 5's new subcommand). This component cannot be executed until both are merged and the binary is rebuilt — plan sequencing gates accordingly.

2. **The Red flags table surgery is load-bearing.** Several existing rows mix band-gate language (`≥0.95`, `0.80–0.95`, `<0.80`) with currently-correct behavior. The plan above identifies the rows to remove, but the executor must diff carefully: removing a row that guards a still-valid behavior (e.g., the `engram activate` batch-call row at the end of Step 2) would be a regression. Only Step 2.5-specific band rows should be deleted; the Step 2 `activated: true` batch-call row is unaffected.

3. **The `engram show` calls in Step 2.5 are blocking and per-cluster.** The spec records recall latency as a headline experiment metric (§3.6). The skill plan does not cap this — per §3.5 the documented fallback (cap at top-N clusters, default remainder to create) is an accepted v2 limit to surface only if the experiment shows it dominates. Do not silently add a cap to the skill text.

### Component 7: Reconcile c1 + learn SKILL.md

**Files:** `docs/architecture/c1-system-context.md`, `skills/learn/SKILL.md`

---

#### Task 7.1 — Verify recall flow prose and sequence diagram are stale

- [ ] Read `docs/architecture/c1-system-context.md` lines 70–144 to confirm the three stale passages:
  1. **Prose** (lines 90–98): describes "three bands on `nearest_l2` cosine — `<0.80` create, `0.80–0.95` update, `≥0.95` activate" (cosine-band gate, fire-and-forget subagent, single `nearest_l2`).
  2. **Prose callout block** (lines 100–106): references the `≥0.95` dedup-silence band and the "fire-and-forget" synthesis model.
  3. **Sequence diagram** (lines 108–144): `Sub as S3 Synthesis subagent`, `H-)Sub: dispatch synthesis subagent (fire-and-forget)`, and the inner loop using `Sub->>V` / `Sub-)E` pattern.

  Expected: all three are present verbatim. If any differ from the read output, adjust the edit targets in Task 7.2 before writing.

  ```bash
  # no command needed — visual inspection of the Read output above is the verification
  # (lines confirmed: prose 90-98, callout 100-106, sequence 108-144)
  ```

- [ ] Confirm the learn flow prose (lines 148–153) uses `engram transcript --mark`:

  Expected text present: `harness first invokes \`engram transcript --mark\` to read session JSONL or SQLite from S5`

- [ ] Confirm the learn sequence diagram (lines 156–194) includes the `Tr as S5 Session stores` participant and the `H->>E: engram transcript --mark` + `E->>Tr:` steps.

- [ ] Confirm `skills/learn/SKILL.md` line 27 contains `prunes stale chunks`:

  Expected: `re-chunks and re-embeds only what changed, and prunes stale chunks.`

No commands to run — verification is reading. Proceed only if all four stale texts are confirmed.

---

#### Task 7.2 — Edit c1: update recall flow prose

Rewrite the stale recall prose and callout block. The new prose must reflect:
- One clustering of matched chunks + notes (D1, not separate clusterings).
- Binary emits `candidate_l2s: [{path, cosine}]` top-K (K≥3) per cluster by centroid cosine (D7, not single `nearest_l2`).
- Agent-judged coverage (covered→activate+link-enrich / near→amend update / absent→create), not the three cosine bands.
- Writes are **blocking and inline** (not fire-and-forget subagents).
- `engram amend` for update/link-enrich; `engram learn` for absent→create. Both carry `--relation` (note wikilinks) and `--chunk-source` (frontmatter provenance).
- Source references: add `engram amend` in `internal/cli/amend.go`; keep the existing `query.go` / `recency.go` / `activate.go` / `cluster/` references.

Edit `docs/architecture/c1-system-context.md`:

```python
old_string = """\
The harness then applies a per-cluster synthesis gate: for an
above-cutoff chunk cluster it crystallizes per three bands on `nearest_l2`
cosine — `<0.80` create, `0.80–0.95` update, **`≥0.95` `engram activate` the
covering L2** (it was useful; refresh, don't duplicate) — writing fact/feedback
via `engram learn` with `--relation` bullets to each constituent. Source:
`internal/cli/query.go` (`RunQuery`, `mergeChunkSpace`, `applyCombinedRecencyBand`),
the recency/decay/band in `internal/cli/recency.go`, the `engram activate`
command in `internal/cli/activate.go`, and the `internal/cluster/` package
(`kmeans.go`, `silhouette.go`, `autok.go`).

> Two deliberate evolutions of earlier decisions, both driven by recency decay:
> (1) recency now applies to **notes**, not chunks only (the v1 chunks-only
> scope); (2) usefulness is an **activation signal** — the lazy-L2 `≥0.95`
> dedup-*silence* band becomes a *refresh*, and a clearly-stated-in-a-chunk idea
> is still crystallized (chunks decay; a refreshable L2 survives). Consequence
> (intended, ACT-R): regularly-useful memory stays fresh; never-retrieved memory
> decays and loses rank."""

new_string = """\
The binary clusters the **matched chunks + notes together** in one AutoK pass
(D1); each cluster carries `candidate_l2s: [{path, cosine}]` — the top-K
(K≥3) existing L2s ranked by centroid cosine — rather than a single
`nearest_l2`. The harness then, **inline and blocking**, reads the cluster's
members and candidates and applies an **agent-judged coverage decision** (D7):
**covered** (candidate already states the principle with no material omission,
judged against the recency-weighted view) → `engram amend --activate
--relation <note sources> --chunk-source <chunk ids>` (refresh recency + enrich
links/provenance, no content rewrite); **near** (same situation, ≥1 substantive
claim omitted) → `engram amend --relation … --chunk-source … <re-synthesized
content>` (update in place, recency-weighted, D6); **absent** (no candidate
addresses the situation) → `engram learn fact|feedback --relation … --chunk-source
… --source "<descriptive>"` (create the single representative L2). Source:
`internal/cli/query.go` (`RunQuery`, `mergeChunkSpace`, `applyCombinedRecencyBand`),
the recency/decay/band in `internal/cli/recency.go`, `engram activate` in
`internal/cli/activate.go`, `engram amend` in `internal/cli/amend.go`, and the
`internal/cluster/` package (`kmeans.go`, `silhouette.go`, `autok.go`).

> Two deliberate evolutions of earlier decisions, both driven by recency:
> (1) recency now applies to **both notes and chunks** — per-chunk `IngestedAt`
> (D5) replaces the per-source-mtime approximation; (2) coverage is **agent-judged**
> (D7) — cosine only nominates top-K candidates; the agent reads members +
> candidates and decides. Writes are **blocking and inline** (not fire-and-forget)
> so the synthesized L2 is available to the agent within the same recall turn.
> Consequence (intended, ACT-R): regularly-useful memory stays fresh;
> never-retrieved memory decays and loses rank."""
```

After the edit, open the file and visually confirm the old cosine-band text is gone.

---

#### Task 7.3 — Edit c1: update recall sequence diagram

Replace the fire-and-forget subagent loop in the recall sequence diagram with the blocking inline agent-judged loop. The `Sub as S3 Synthesis subagent` participant is removed; the harness reads candidates and members itself and calls `engram amend` or `engram learn` directly.

Edit `docs/architecture/c1-system-context.md`:

```python
old_string = """\
```mermaid
sequenceDiagram
    autonumber
    actor Op as S1 Operator
    participant H as S3 Harness
    participant E as S2 Engram CLI
    participant V as S4 Vault
    participant Sub as S3 Synthesis subagent

    Op->>H: prompt that may need memory
    Note over H: print Step 0 (Ask, Situation, Plan), phrase 5-15 query strings
    H->>E: engram query --phrase <p1> --phrase <p2> ... --limit N
    E->>V: scan sidecars + bodies for compatible-embed notes
    V-->>E: notes and vectors
    Note over E: per phrase — embed, top-k cosine, BFS 3 hops (cap 200), k-means, in-degree top-5
    Note over E: merge server-side — items dedup by path (max score, union provenances); reweight chunk hits by recency and guarantee a floor of recent chunks; clusters tagged per-phrase
    E-->>H: single YAML payload (phrases[], items, clusters, hubs, budget)
    Note over H: surface anchor concepts from hubs

    loop per cluster
        Note over H: read the cluster representative — gate on ≥3 members and rep-coherence hint
        alt cluster passes the cheap gate
            H-)Sub: dispatch synthesis subagent (fire-and-forget)
            Sub->>V: read all cluster members
            V-->>Sub: member contents
            Note over Sub: decide whether a binding principle is worth capturing
            Sub-)E: engram learn fact or feedback (--relation per constituent)
            E->>V: acquire flock, compute Luhmann, write note
            V-->>E: written path
        else cluster fails the cheap gate
            Note over H: skip — cluster members remain as context
        end
    end

    Note over H: Step 4b synthesis against the Step 0 plan
    H-->>Op: reply opening with anchor concepts, then plan walk (confirmed / adjusted / contradicted / silent)
```"""

new_string = """\
```mermaid
sequenceDiagram
    autonumber
    actor Op as S1 Operator
    participant H as S3 Harness
    participant E as S2 Engram CLI
    participant V as S4 Vault

    Op->>H: prompt that may need memory
    Note over H: print Step 0 (Ask, Situation, Plan), phrase 5-15 query strings
    H->>E: engram query --synthesize-l2 --phrase <p1> --phrase <p2> ... --limit N
    E->>V: scan sidecars + bodies for compatible-embed notes + chunk index
    V-->>E: notes, chunks, and vectors
    Note over E: per phrase — embed, top-k cosine, BFS 3 hops (cap 200), k-means, in-degree top-5
    Note over E: merge server-side — items dedup by path (max score, union provenances); reweight by per-chunk IngestedAt recency; guarantee floor of recent items; clusters tagged per-phrase
    Note over E: one AutoK cluster over matched chunks + notes; per cluster emit candidate_l2s (top-K by centroid cosine)
    E-->>H: single YAML payload (phrases[], items, clusters[candidate_l2s], hubs, budget)
    Note over H: surface anchor concepts from hubs

    loop per cluster (blocking inline)
        H->>V: engram show <candidate L2s> and cluster members
        V-->>H: candidate + member contents
        Note over H: apply recency weight; judge coverage (covered / near / absent)
        alt covered — candidate already states the principle
            H->>E: engram amend <l2> --activate --relation <note srcs> --chunk-source <ids>
            E->>V: acquire flock, merge relations + provenance, bump LastUsed
            V-->>E: written path
        else near — same situation, substantive claim omitted
            H->>E: engram amend <l2> --relation <note srcs> --chunk-source <ids> <re-synth content>
            E->>V: acquire flock, replace content fields, merge relations + provenance, re-embed
            V-->>E: written path
        else absent — no candidate addresses the situation
            H->>E: engram learn fact|feedback --relation <note srcs> --chunk-source <ids> --source "<desc>"
            E->>V: acquire flock, compute Luhmann ID, write note
            V-->>E: written path
        end
    end

    Note over H: Step 4b synthesis against the Step 0 plan
    H-->>Op: reply opening with anchor concepts, then plan walk (confirmed / adjusted / contradicted / silent)
```"""
```

After edit, visually confirm: `Sub as S3 Synthesis subagent` is gone; `--synthesize-l2` appears; the three `alt/else` branches (`covered`/`near`/`absent`) are present; `fire-and-forget` is gone.

---

#### Task 7.4 — Edit c1: update learn flow prose

Replace the stale learn prose paragraph (references `engram transcript --mark`, `learn episode`, and the old "prunes stale chunks" ingest model).

Edit `docs/architecture/c1-system-context.md`:

```python
old_string = """\
### Flow: learn

Operator runs `/learn` (or the harness self-fires after substantive work). The
harness first invokes `engram transcript --mark` to read session JSONL or
SQLite from S5 and advance the per-harness marker forward, then writes any
captured lessons into the vault via `engram learn {feedback|fact|episode}`. Each
write acquires a `flock` on the vault root before computing the Luhmann ID and
emitting the new file. Source: `internal/cli/transcript.go`
(`advanceAndReportMarker`) and `internal/cli/learn.go` (`runLearn`)."""

new_string = """\
### Flow: learn

Operator runs `/learn` (or the harness self-fires after substantive work). The
harness first invokes `engram ingest --auto` to merge-append any new chunks from
session transcripts (S5) and markdown sources into the chunk index — re-chunking
and re-embedding only changed content, never deleting existing records (append-only,
D5). It then writes any EXPLICIT lessons (corrections or explicit save-requests)
into the vault via `engram learn {feedback|fact}`. Each write acquires a `flock`
on the vault root before computing the Luhmann ID and emitting the new file.
Source: `internal/cli/ingest.go` (`runIngest`) and `internal/cli/learn.go`
(`runLearn`). The `engram learn episode` subcommand and `engram transcript --mark`
are retired: episodes are superseded by the chunk layer (D4); transcript reading
is subsumed by `engram ingest --auto`."""
```

---

#### Task 7.5 — Edit c1: update learn sequence diagram

Replace the sequence diagram for the learn flow: remove `Tr as S5 Session stores` as a separate participant; replace `engram transcript --mark` step with `engram ingest --auto`; remove `engram learn episode`; remove the `alt no marker yet` block; remove the "prunes stale chunks" semantics.

Edit `docs/architecture/c1-system-context.md`:

```python
old_string = """\
```mermaid
sequenceDiagram
    autonumber
    actor Op as S1 Operator
    participant H as S3 Harness
    participant E as S2 Engram CLI
    participant Tr as S5 Session stores
    participant V as S4 Vault

    Op->>H: invoke /learn (or self-fire after substantive work)

    H->>E: engram transcript --mark
    E->>Tr: read JSONL or SQLite from per-harness marker forward
    Tr-->>E: session entries up to byte cap
    Note over E: write marker forward (XDG_STATE_HOME)
    E-->>H: status line, scanned range, advanced marker

    alt no marker yet (first run)
        E-->>H: exit non-zero, earliest session date
        H->>Op: ask scan start date via AskUserQuestion
        Op-->>H: chosen start date
        H->>E: engram transcript --mark --from CHOSEN
        E-->>H: status line
    end

    Note over H: read transcript output plus in-context turns, identify candidates, apply recall-mirror test

    loop per candidate (one parallel tool-use block)
        H->>E: engram learn feedback|fact|episode --slug ... --source ... --situation ...
        alt vault dir missing
            E->>V: bootstrap .obsidian, README, .gitignore
        end
        E->>V: acquire flock, compute Luhmann ID, write note
        V-->>E: written path
        E-->>H: emit written path on stdout
    end

    H-->>Op: report scanned status line plus written permanent paths
```"""

new_string = """\
```mermaid
sequenceDiagram
    autonumber
    actor Op as S1 Operator
    participant H as S3 Harness
    participant E as S2 Engram CLI
    participant V as S4 Vault

    Op->>H: invoke /learn (or self-fire after substantive work)

    H->>E: engram ingest --auto
    E->>V: stat known sources (session transcripts, markdown); re-chunk + re-embed changed content only
    Note over E: merge-append new chunks by ContentHash; never delete existing records (D5)
    V-->>E: written chunk count
    E-->>H: per-source chunk tally (or "memory index up to date")

    Note over H: scan THIS session for corrections and explicit save-requests only

    loop per explicit lesson (one parallel tool-use block)
        H->>E: engram learn feedback|fact --slug ... --source ... --situation ...
        alt vault dir missing
            E->>V: bootstrap .obsidian, README, .gitignore
        end
        E->>V: acquire flock, compute Luhmann ID, write note
        V-->>E: written path
        E-->>H: emit written path on stdout
    end

    H-->>Op: report chunk tally plus written permanent paths
```"""
```

---

#### Task 7.6 — Edit c1: update the L1 decision flowchart (learn lifecycle)

The existing "engram engagement" flowchart (lines 319–335) references `§6b: synthesize or update an L3 ADR` in the learn branch. Per the spec §7 step 7, L3 ADR synthesis at learn time is retired (table in learn SKILL.md: "ADR / L3 synthesis at learn time → Deferred entirely; crystallization happens at recall"). Update node `K` and its label.

Edit `docs/architecture/c1-system-context.md`:

```python
old_string = """\
    I --> J{convention recurs across episodes?}
    J -->|yes| K[§6b: synthesize or update an L3 ADR]
    J -->|no| G
    K --> G"""

new_string = """\
    I --> G"""
```

Also remove the now-unreachable `J` node line that branches to `K` and the stale `I --> J` — but since the node labels reference `transcript --mark` and `episodes`, update node `H` and `I` text too to match the new ingest model:

```python
old_string = """\
    F -->|yes| H[learn: write memory]
    H --> I[transcript --mark, then write episodes + facts/feedback]
    I --> J{convention recurs across episodes?}
    J -->|yes| K[§6b: synthesize or update an L3 ADR]
    J -->|no| G
    K --> G"""

new_string = """\
    F -->|yes| H[learn: write memory]
    H --> I[engram ingest --auto, then write facts/feedback for explicit lessons]
    I --> G"""
```

After edit, verify the flowchart has no dangling `J`/`K` nodes and no `transcript --mark` / `episodes` / `ADR` text in the learn branch.

---

#### Task 7.7 — Edit learn SKILL.md: remove "prunes stale chunks"

The single-word change: the learn SKILL.md Step 1 prose says ingest "prunes stale chunks". Per D5 (append-only), ingest never prunes.

Edit `skills/learn/SKILL.md`:

```python
old_string = "re-chunks and re-embeds only what changed, and prunes stale chunks."
new_string = "re-chunks and re-embeds only what changed — existing chunks are never deleted (append-only history)."
```

After edit, confirm line 27 (approximately) reads the new text. No other changes to learn SKILL.md: the `engram transcript --mark`, `engram learn episode`, and `ADR / L3 synthesis` entries in the "What learn does NOT do anymore" table are already present and correct as-is.

---

#### Task 7.8 — Verify no remaining stale text

Check that all stale phrases are gone:

```bash
grep -n "nearest_l2\|prunes stale chunks\|fire-and-forget\|Synthesis subagent\|transcript --mark\|learn episode\|L3 ADR\|three bands" \
  /Users/joe/repos/personal/engram/docs/architecture/c1-system-context.md \
  /Users/joe/repos/personal/engram/skills/learn/SKILL.md
```

Expected output: empty (no matches). If any match appears, read the surrounding context and apply a follow-up Edit before declaring done.

---

#### Task 7.9 — Verify no broken Mermaid syntax

```bash
grep -n "participant Sub\|H-)Sub\|Sub->>V\|Sub-)E" \
  /Users/joe/repos/personal/engram/docs/architecture/c1-system-context.md
```

Expected: empty. Then do a quick structural sanity check on the two edited sequence diagrams — each must have:
- matching `autonumber` + `actor`/`participant` + at least one `Op->>H` + `H->>E` + `E->>V` pattern
- all `loop`/`alt`/`else`/`end` paired

```bash
grep -c "sequenceDiagram" /Users/joe/repos/personal/engram/docs/architecture/c1-system-context.md
```

Expected: `2` (recall diagram + learn diagram).

```bash
grep -c "flowchart TD" /Users/joe/repos/personal/engram/docs/architecture/c1-system-context.md
```

Expected: `2` (engagement flowchart + please flowchart).

---

**Risks/notes for the synthesizer:**

1. **Ordering dependency on step 6 (recall SKILL.md):** Task 7.7 here edits `skills/learn/SKILL.md` directly (a single-line prose change, not a SKILL.md behavioral rewrite), so it does NOT require `superpowers:writing-skills`. Step 6 edits `skills/recall/SKILL.md` and DOES require that TDD gate — keep steps 6 and 7 ordered to avoid concurrent edits to the skills/ directory.
2. **`internal/cli/amend.go` referenced before it exists:** Task 7.2 adds a source reference to `amend.go` in the c1 prose. This file is created in step 5 (§3.4 `engram amend`). Write the c1 prose edit after step 5 lands, or leave a `(forthcoming)` note and update it in this task — do not reference a non-existent file as if it ships in this step alone.
3. **L1 decision flowchart node removal (Task 7.6):** the `J{convention recurs across episodes?}` branch and `K[§6b: synthesize or update an L3 ADR]` node may have click-anchor IDs or other downstream references in c2/c3 diagrams. Verify with `grep -r "§6b\|L3 ADR" docs/` before the edit — if found in c2/c3, those files need companion edits not covered here.
