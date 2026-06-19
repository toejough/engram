# Lazy compositional L2 synthesis — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Build the lazy compositional L2 synthesis feature per `docs/superpowers/specs/2026-06-09-lazy-l2-synthesis-design.md` — the seven locked decisions D1–D7: single-pass clustering, in-place `engram amend`, exclude `Related to:` from the embed source, chunks-in-index with frontmatter provenance, append-only chunk history + per-chunk recency, recency-weighted distillation, and agent-judged coverage with top-K centroid candidate nomination.

**Architecture:** Synthesis runs at recall over one clustering of matched chunks + L2 notes; the binary nominates top-K candidate L2s per cluster (centroid cosine) and the recall skill's agent decides coverage (activate / update-in-place / create), writing note↔note `[[wikilinks]]` via `engram amend` and recording chunk sources as frontmatter provenance. Chunks stay in the append-only index (per-chunk `IngestedAt` recency); materialization/episode layers are dropped.

**Tech Stack:** Go 1.26; `internal/cli` + `internal/chunk` + `internal/embed`; build/test via `targ test` / `targ check-full` only; tests imptest + rapid + gomega, `package cli_test` blackbox via `export_test.go` `Export*` aliases; DI everywhere; SKILL.md edits via `superpowers:writing-skills`.

**Scope decision (recorded dissent).** Considered building the minimal value-slice first (`amend` + recall-skill note-linking over the existing note clustering); the user chose the **full §7 build** for the complete designed system. The value-proof experiment (eager-vs-lazy A/B, spec §3.6 / §7 step 8) remains **blocked on #642/#643** and is **excluded from this plan** (not deferred to a later task here — it is a separate tracked effort) — so this build is **not value-validatable** until those clear. Recorded per the anti-sycophantic resolution rule.

**Build order (dependency-sequenced).** Component 2 (append-only chunks + per-chunk recency) → 3 (unified clustering + top-K nomination) → 4 (exclude `Related to:` from embed) → 5 (`amend` + `learn --chunk-source`; depends on 4) → 6 (recall-skill rewrite; depends on 2, 3, 5) → 7 (reconcile docs). Each component's tasks are independently TDD'd and committed; Gate B (design-fit) runs after each refactor during execution.

**Task-dependency index.** Before starting a dependent component, verify its deps merged (e.g. `grep IngestedAt internal/chunk/index.go`; `targ check-full` green).

| Component | Local task IDs | Depends on |
| --- | --- | --- |
| 2 — append-only chunks + per-chunk recency (D5) | 2.1–2.11 | — |
| 3 — unified clustering + top-K centroid nomination (D1/D7) | 3.1–3.4 | sequence after 2 |
| 4 — exclude `Related to:` from embed (D3) | 4.1–4.8 | — |
| 5 — `engram amend` + `learn --chunk-source` (D2) | 5.1–5.10 | 4; reuses 2's `buildChunkIDSet` (`source#anchor`) |
| 6 — recall-skill agent-judged coverage (D6/D7) | 6.1–6.8 (6.1a/b/c) | 2, 3, 5 |
| 7 — reconcile c1 + learn SKILL.md | 7.1–7.9 | 6 |

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

> **CA-01 fix:** `IngestDeps.Now` must be added to the struct BEFORE the new `rebuildIndex` body references `deps.Now()`. The steps below enforce this ordering explicitly.

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

- [ ] **Step 1: Add `Now func() time.Time` to `IngestDeps`** (must happen BEFORE rewriting `rebuildIndex`). In `/Users/joe/repos/personal/engram/internal/cli/ingest.go`, add to the `IngestDeps` struct (after `Embedder embed.Embedder`):

  ```go
  // Now returns the current wall-clock time for IngestedAt stamping. Nil-safe:
  // callers guard with "if deps.Now != nil" before calling. Wire time.Now in
  // newOsIngestDeps.
  Now func() time.Time
  ```

  Also add `Now: time.Now,` to `newOsIngestDeps()`.

- [ ] **Step 2: Replace `rebuildIndex` with merge-append implementation.** In `/Users/joe/repos/personal/engram/internal/cli/ingest.go`, replace the body of `rebuildIndex` so it:
  1. Loads `priorRecords = loadPriorRecords(indexPath, deps)`
  2. Builds a `merged []chunk.Record` starting with all prior records (order preserved).
  3. Builds `existingHashes` set from prior records.
  4. For each new `piece` in `chunks`: if `hash` already in `existingHashes`, mark `reused++`, skip; else embed, create new record with `IngestedAt: now`, append to `merged`, mark `embedded++`.
  5. `total = len(merged)` (prior + new).
  6. Encodes and writes `merged`.

  The function signature gains a `now time.Time` param and a `backfillTime func(source string) time.Time` param (Task 2.5 adds this — add it now to avoid a second intermediate broken build per CA-04):

```go
// rebuildIndex merge-appends: prior records are preserved (never deleted), new
// hashes are embedded and stamped with ingestTime, and zero-IngestedAt legacy
// records are backfilled via backfillTime (nil = no backfill).
func rebuildIndex(
	ctx context.Context,
	source string,
	chunks []chunk.Chunk,
	chunksDir string,
	deps IngestDeps,
	ingestTime time.Time,
	backfillTime func(source string) time.Time,
) (total, reused, embedded int, err error) {
	indexPath := filepath.Join(chunksDir, sourceSlug(source)+jsonlExt)
	priorRecords := loadPriorRecords(indexPath, deps)

	// Preserve prior records first (append-only: never delete).
	existingHashes := make(map[string]bool, len(priorRecords))
	merged := make([]chunk.Record, 0, len(priorRecords)+len(chunks))

	for _, r := range priorRecords {
		existingHashes[r.ContentHash] = true
		if r.IngestedAt.IsZero() && backfillTime != nil {
			r.IngestedAt = backfillTime(r.Source)
		}
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
			IngestedAt:  ingestTime,
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

  Note: `ingestTime` and `backfillTime` are both threaded from `ingestSource`. The full wiring appears in Task 2.4 (where `chunkSource` grows its return value) and Task 2.5 (where `backfillTime` is built from the manifest). For now, update the `ingestSource` call site temporarily:

  ```go
  rebuilt, reused, embedded, err := rebuildIndex(ctx, source, chunks, chunksDir, deps, deps.Now(), nil)
  ```

  Guard `deps.Now()` at the call site:
  ```go
  var ingestTime time.Time
  if deps.Now != nil {
      ingestTime = deps.Now()
  }
  rebuilt, reused, embedded, err := rebuildIndex(ctx, source, chunks, chunksDir, deps, ingestTime, nil)
  ```

- [ ] **Run GREEN.**
  ```
  targ test
  ```
  Expected: both new tests pass; all existing ingest tests pass.

- [ ] **`targ check-full`** — clean.

---

#### Task 2.4 — Thread transcript per-session timestamp as `IngestedAt` for transcript chunks

> **docs-F2 / contract item 4:** Threading `ReadResult.LastTimestamp` as `IngestedAt` for all chunks produced from one transcript call is a **per-session approximation**, not a per-row timestamp. A single transcript produces multiple chunks (turn-1 through turn-N), and all receive the same `IngestedAt` (the last turn's timestamp from `ReadResult.LastTimestamp`). This is an accepted approximation per spec §3.2 caveat: "intra-session time spread is negligible for recency; cross-session ordering is already distinguished since each session is its own source." The migration backfill similarly accepts per-source (not per-row) granularity. Future work to thread per-row timestamps would require `chunk.Chunk` to carry a source timestamp field — deferred as YAGNI.

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

	// Transcript chunks use the per-session LastTimestamp (a per-session approximation
	// — all chunks of one transcript share IngestedAt; intra-session spread is negligible
	// for recency; cross-session ordering is distinguished since each session is its own source).
	g.Expect(records[0].IngestedAt).To(gomega.Equal(ts),
		"transcript chunk IngestedAt must be the LastTimestamp from ReadResult (per-session approximation)")
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
  Expected: `TestIngestTranscriptSetsIngestedAtFromPerRowTimestamp` fails (IngestedAt will be `Now()`, not the per-session timestamp). `TestIngestMarkdownSetsIngestedAtFromNow` may also fail if `Now` nil-guard is missing.

- [ ] **Thread the per-session timestamp.** Change `chunkSource` to return `([]chunk.Chunk, time.Time, error)` where the `time.Time` is `ReadResult.LastTimestamp` for transcripts, zero for markdown. In `/Users/joe/repos/personal/engram/internal/cli/ingest.go`, update `chunkSource` (currently at line 163):

```go
// chunkSource dispatches by extension: transcripts strip+turn-chunk, markdown
// heading-chunks. Returns (chunks, sourceTimestamp, err). For transcripts,
// sourceTimestamp is ReadResult.LastTimestamp (used as IngestedAt for all chunks
// from this call — a per-session approximation; see Task 2.4 comment). For
// markdown, sourceTimestamp is zero (caller uses deps.Now()).
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

- [ ] **Pass the source timestamp into `ingestSource` → `rebuildIndex`.** In `ingestSource`, replace the current (Task 2.3) call:
  ```go
  chunks, err := chunkSource(source, raw, deps)
  if err != nil {
      return false, err
  }

  var ingestTime time.Time
  if deps.Now != nil {
      ingestTime = deps.Now()
  }
  rebuilt, reused, embedded, err := rebuildIndex(ctx, source, chunks, chunksDir, deps, ingestTime, nil)
  ```
  with:
  ```go
  chunks, sourceTS, err := chunkSource(source, raw, deps)
  if err != nil {
      return false, err
  }

  // For transcripts, use the per-session LastTimestamp (per-session approximation,
  // see Task 2.4 doc comment); for markdown, fall back to Now().
  // Guard deps.Now for test fixtures that omit it.
  ingestTime := sourceTS
  if ingestTime.IsZero() && deps.Now != nil {
      ingestTime = deps.Now()
  }

  rebuilt, reused, embedded, err := rebuildIndex(ctx, source, chunks, chunksDir, deps, ingestTime, nil)
  ```

- [ ] **Run GREEN.**
  ```
  targ test
  ```
  Expected: both new tests pass.

- [ ] **`targ check-full`** — clean.

---

#### Task 2.5 — Migration backfill: populate `IngestedAt` from manifest mtime on first merge

> **CA-04 / CA-10 verification:** `ingestSource` (at line 251 in `ingest.go`) has the signature `func ingestSource(ctx context.Context, source, chunksDir string, deps IngestDeps, manifest ingestManifest, stdout io.Writer)` — `manifest ingestManifest` IS in scope. The backfill closure is built inside `ingestSource` from the already-present `manifest` parameter before calling `rebuildIndex`. No threading change to `ingestSource`'s signature is needed.

> **CLR-005 design decision (explicit):** The backfill-time lookup is injected as a closure (`backfillTime func(source string) time.Time`) from `ingestSource`. This keeps `rebuildIndex` testable via DI (no manifest I/O inside the index-builder). Alternatives considered and rejected: (a) read the manifest inside `rebuildIndex` — couples the index-builder to manifest I/O, less testable; (b) pass the manifest as a map param — less type-safe than a closure. Chosen: (c) closure injection from the higher-level caller where the manifest is already in scope.

> **Note:** `rebuildIndex` already has the `backfillTime` parameter from Task 2.3 (added there to avoid a second broken-build cycle per CA-04). This task only wires the closure from `ingestSource`.

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
	g.Expect(records[0].IngestedAt).To(gomega.Equal(manifestMtime),
		"legacy zero-IngestedAt record must be backfilled from manifest mtime on first merge")
}
```

- [ ] **Run RED.**
  ```
  targ test
  ```
  Expected: `TestMergeAppendBackfillsIngestedAtFromManifestMtime` fails — backfill not yet wired.

- [ ] **Wire the `backfillTime` closure in `ingestSource`.** In `/Users/joe/repos/personal/engram/internal/cli/ingest.go`, inside `ingestSource`, replace the call to `rebuildIndex` (added in Task 2.4):

  ```go
  rebuilt, reused, embedded, err := rebuildIndex(ctx, source, chunks, chunksDir, deps, ingestTime, nil)
  ```

  with:

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

  `manifest` is the `ingestManifest` parameter already in scope in `ingestSource`. `entry.MtimeUnixNano` is the `MtimeUnixNano int64` field on the embedded `SourceStat` struct (confirmed: `manifestEntry` embeds `SourceStat` which has `MtimeUnixNano int64`).

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

> **F10 / pre-task grep:** Before modifying `applyChunkRecency`, run the following to identify ALL callers (including `recency_eval_test.go`) that use the old 4-arg form with `ageDaysBySource`:
>
> ```
> grep -rn "ExportApplyChunkRecency\|applyChunkRecency" /Users/joe/repos/personal/engram/internal/cli/
> ```
>
> Known callers requiring migration:
> - `recency_test.go:25` — `TestApplyChunkRecencyLiftsRecentOverStaleHighCosine` (passes `ages map[string]float64` as second arg)
> - `recency_eval_test.go:221` — `rankOf` helper (passes `ages map[string]float64` as second arg)
> - `query.go` — the production caller (passes `ages` from `chunkSourceAges`)
>
> All four must be updated in the GREEN step. The `ExportApplyChunkRecency` var alias auto-updates its type when `applyChunkRecency` changes, but every call site that passes the old `ages map[string]float64` argument will break at compile time.

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

- [ ] **Run RED.**
  ```
  targ test
  ```
  Expected: compile error — `ExportApplyChunkRecencyByTime` and `ExportNewScoredChunkWithIngestedAt` do not exist.

- [ ] **Change `applyChunkRecency` signature.** In `/Users/joe/repos/personal/engram/internal/cli/recency.go`, replace the function (currently at lines 37–63):

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

- [ ] **Update the production caller in `query.go`.** Find the call to `applyChunkRecency` in `/Users/joe/repos/personal/engram/internal/cli/query.go` (which currently passes `ages` from `chunkSourceAges`) and replace:
  ```go
  // Old:
  scored = applyChunkRecency(scored, ages, maxTurnBySource(records), params)
  // New:
  scored = applyChunkRecency(scored, deps.Now(), maxTurnBySource(records), params)
  ```
  Remove the `ages` variable and the `chunkSourceAges` call that sets it.

- [ ] **Add new exports to `export_test.go`.** In `/Users/joe/repos/personal/engram/internal/cli/export_test.go`:

  Remove the existing `ExportApplyChunkRecency = applyChunkRecency` line from the `var (...)` block (it will be replaced by the function form below to avoid confusion — the var alias auto-updates its type but callers still need updating, and the explicit function form prevents silent breakage):

  Add:
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

- [ ] **REQUIRED GREEN step: Update `TestApplyChunkRecencyLiftsRecentOverStaleHighCosine` in `recency_test.go`.** This test currently passes `ages map[string]float64` as the second arg — it MUST be rewritten before `targ test` can pass. Replace the full test body:

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

- [ ] **REQUIRED GREEN step: Update `recency_eval_test.go`.** The `rankOf` helper at line 221 calls `cli.ExportApplyChunkRecency(pool, ages, maxTurn, p)` with the old 4-arg signature. This file also calls `cli.ExportNewestChunkItems(scored, ages, floor)` at line 238 with the old 3-arg form. Both must be updated.

  In `recency_eval_test.go`, update `buildSyntheticPool` to set `IngestedAt` on each record using `time.Unix(0, 0).Add(-ageDays * 24 * time.Hour)` relative to a fixed `now`, and update `rankOf` to:
  - Replace `cli.ExportApplyChunkRecency(pool, ages, maxTurn, p)` with `cli.ExportApplyChunkRecencyByTime(pool, now, maxTurn, p)` where `now` is a fixed reference time (e.g. `time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)`).
  - Replace `cli.ExportNewestChunkItems(scored, ages, floor)` with `cli.ExportNewestChunkItemsByTime(scored, floor)` (new 2-arg form, added in Task 2.8).

  Since Task 2.8 adds `ExportNewestChunkItemsByTime`, the eval test update for `ExportNewestChunkItems` must be done in Task 2.8's GREEN step. For Task 2.7's GREEN step, focus on the `ExportApplyChunkRecency` → `ExportApplyChunkRecencyByTime` migration in `recency_eval_test.go`.

  Specifically in `buildSyntheticPool`, add `IngestedAt` to each record. Change the record construction for weeksold chunks:
  ```go
  evalNow := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
  // weeksold chunks: IngestedAt = now - 21 days
  weeksOldTime := evalNow.Add(-21 * 24 * time.Hour)
  for _, turn := range []string{"turn-10", "turn-5", "turn-1"} {
      rec := chunk.Record{
          Source:      "weeksold.jsonl",
          Anchor:      turn,
          Text:        "ASSISTANT: session narration at " + turn,
          ContentHash: "sha256:weeksold-" + turn,
          IngestedAt:  weeksOldTime,
      }
      pool = append(pool, cli.ExportNewScoredChunk(rec, weeksOldCosine))
  }
  ```

  And for distractor tiers:
  ```go
  for _, tier := range tiers {
      tierTime := evalNow.Add(time.Duration(-tier.ageDays * float64(24 * time.Hour)))
      // ... existing maxTurn setup ...
      for i := range distractorsPerTier {
          rec := chunk.Record{
              Source:      tier.source,
              Anchor:      "turn-" + itoa(i),
              ContentHash: "sha256:" + tier.source + itoa(i),
              IngestedAt:  tierTime,
          }
          pool = append(pool, cli.ExportNewScoredChunk(rec, tier.cosine))
      }
  }
  ```

  Update `rankOf` signature to drop `ages` and `maxTurn` params (they are now derived from records' `IngestedAt` and anchors), passing `evalNow` instead:
  ```go
  func rankOf(
  	targetPath string,
  	pool []cli.ExportScoredChunk,
  	maxTurn map[string]int,
  	p cli.ExportRecencyParams,
  	limit int,
  ) int {
  	evalNow := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
  	scored := cli.ExportApplyChunkRecencyByTime(pool, evalNow, maxTurn, p)
  	// ... rest of function unchanged ...
  ```

  Update all callers of `rankOf` / `weeksOldRankOf` to remove the `ages` argument.

- [ ] **Run GREEN.**
  ```
  targ test
  ```
  Expected: all tests pass including the new `TestApplyChunkRecencyUsesIngestedAt`.

- [ ] **`targ check-full`** — clean.

---

#### Task 2.8 — Re-key `newestChunkItems`: sort key = `IngestedAt`, drop `ages` param

> **CA-03 fix:** Three existing tests call `ExportNewestChunkItems` with the old 3-arg form (`scored, ages, n`). Their rewrite is a REQUIRED step in the GREEN phase — the build cannot pass without it. The three tests are:
> - `TestNewestChunkItemsOrdersByAgeAscending` (recency_test.go:344)
> - `TestNewestChunkItemsTieBreaksByTurnDesc` (recency_test.go:369)
> - `TestNewestChunkItemsNZeroReturnsNil` (recency_test.go:329)
>
> Additionally, `recency_eval_test.go:238` calls `ExportNewestChunkItems(scored, ages, floor)` with the old form — it must also be updated.
>
> `ExportNewestChunkItemsByTime` and `ExportNewestChunkItems` are identical wrappers for the same 2-arg function. Having two names is intentional: `ExportNewestChunkItems` keeps the existing name stable for external test references; `ExportNewestChunkItemsByTime` is the semantically-named form for new tests. This is not dead code — both are exported test helpers with distinct call-site uses.

- [ ] **Write the failing tests.** Add to `recency_test.go`:

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

- [ ] **Change `newestChunkItems` signature** in `/Users/joe/repos/personal/engram/internal/cli/recency.go`: drop the `ages map[string]float64` param; sort by `s.record.IngestedAt` descending (newest first); zero `IngestedAt` sorts last (maximally old — these are unknown-age legacy records, not maximally recent; the floor band's job is to surface the actually newest chunks):

  Replace the entire `newestChunkItems` function (currently lines 212–258):

  ```go
  // newestChunkItems returns the n chunk items with the largest IngestedAt
  // (most recently ingested first). Chunks with zero IngestedAt (legacy, not
  // yet backfilled) sort last — treated as maximally old since their recency is
  // unknown. Tie-breaking on equal IngestedAt uses descending turn-N (latest
  // turn first). Returns nil when n<=0.
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

- [ ] **Update the caller in `query.go`** (wherever `newestChunkItems` is called with `ages` as the second arg):
  ```go
  // Old:
  chunkMust = newestChunkItems(scored, ages, params.floor)
  // New:
  chunkMust = newestChunkItems(scored, params.floor)
  ```

- [ ] **Update `ExportNewestChunkItems` and add `ExportNewestChunkItemsByTime` in `export_test.go`.** Replace the existing `ExportNewestChunkItems` function (currently 3-arg at line 334):

  ```go
  // ExportNewestChunkItems exposes newestChunkItems (new 2-arg signature: no ages map).
  func ExportNewestChunkItems(scored []scoredChunk, n int) []resolvedItem {
  	return newestChunkItems(scored, n)
  }

  // ExportNewestChunkItemsByTime is an alias for tests that use the IngestedAt-keyed sort.
  // Both names wrap the same 2-arg newestChunkItems; ExportNewestChunkItems keeps the
  // existing test-helper name stable, ExportNewestChunkItemsByTime is the semantic form.
  func ExportNewestChunkItemsByTime(scored []scoredChunk, n int) []resolvedItem {
  	return newestChunkItems(scored, n)
  }
  ```

- [ ] **REQUIRED GREEN step: Rewrite the three existing tests** that call `ExportNewestChunkItems` with the old 3-arg form. All three are in `recency_test.go`:

  **`TestNewestChunkItemsNZeroReturnsNil`** (line 329) — replace:
  ```go
  func TestNewestChunkItemsNZeroReturnsNil(t *testing.T) {
  	t.Parallel()

  	scored := []cli.ExportScoredChunk{
  		cli.ExportNewScoredChunkWithIngestedAt(
  			chunk.Record{Source: "a.jsonl", Anchor: "turn-1"}, 0.5,
  			time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
  	}

  	out := cli.ExportNewestChunkItems(scored, 0)

  	if out != nil {
  		panic("expected nil for n=0")
  	}
  }
  ```

  **`TestNewestChunkItemsOrdersByAgeAscending`** (line 344) — replace with IngestedAt-based version:
  ```go
  func TestNewestChunkItemsOrdersByAgeAscending(t *testing.T) {
  	t.Parallel()
  	g := NewWithT(t)

  	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
  	// Three sources: recent (0.5d), mid (14d), old (60d). newestChunkItems
  	// should return the floor-newest by IngestedAt, ignoring cosine score.
  	scored := []cli.ExportScoredChunk{
  		cli.ExportNewScoredChunkWithIngestedAt(
  			chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.90,
  			now.Add(-60*24*time.Hour)),
  		cli.ExportNewScoredChunkWithIngestedAt(
  			chunk.Record{Source: "recent.jsonl", Anchor: "turn-7"}, 0.20,
  			now.Add(-12*time.Hour)),
  		cli.ExportNewScoredChunkWithIngestedAt(
  			chunk.Record{Source: "mid.jsonl", Anchor: "turn-5"}, 0.50,
  			now.Add(-14*24*time.Hour)),
  	}

  	out := cli.ExportNewestChunkItems(scored, 2)

  	g.Expect(out).To(HaveLen(2))

  	if len(out) < 2 {
  		return
  	}

  	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("recent.jsonl#turn-7"), "slot 0 must be newest source")
  	g.Expect(cli.ExportResolvedItemPath(out[1])).To(Equal("mid.jsonl#turn-5"), "slot 1 must be second-newest source")
  }
  ```

  **`TestNewestChunkItemsTieBreaksByTurnDesc`** (line 369) — replace with IngestedAt-based version:
  ```go
  func TestNewestChunkItemsTieBreaksByTurnDesc(t *testing.T) {
  	t.Parallel()
  	g := NewWithT(t)

  	// Same IngestedAt → descending turn-N wins.
  	sameTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
  	scored := []cli.ExportScoredChunk{
  		cli.ExportNewScoredChunkWithIngestedAt(
  			chunk.Record{Source: "a.jsonl", Anchor: "turn-2"}, 0.5, sameTime),
  		cli.ExportNewScoredChunkWithIngestedAt(
  			chunk.Record{Source: "a.jsonl", Anchor: "turn-9"}, 0.5, sameTime),
  		cli.ExportNewScoredChunkWithIngestedAt(
  			chunk.Record{Source: "a.jsonl", Anchor: "turn-5"}, 0.5, sameTime),
  	}

  	out := cli.ExportNewestChunkItems(scored, 2)

  	g.Expect(out).To(HaveLen(2))

  	if len(out) < 2 {
  		return
  	}

  	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("a.jsonl#turn-9"), "highest turn first on tie")
  	g.Expect(cli.ExportResolvedItemPath(out[1])).To(Equal("a.jsonl#turn-5"), "second-highest turn second")
  }
  ```

- [ ] **REQUIRED GREEN step: Update `recency_eval_test.go:238`** — replace `cli.ExportNewestChunkItems(scored, ages, floor)` (3-arg) with `cli.ExportNewestChunkItemsByTime(scored, floor)` (2-arg). Also remove the `ages` parameter from `rankOf` / `weeksOldRankOf` signatures and all callers if not already done in Task 2.7.

- [ ] **Run GREEN.**
  ```
  targ test
  ```
  Expected: all new and rewritten tests pass.

- [ ] **`targ check-full`** — clean.

---

#### Task 2.9 — Remove `chunkSourceAges` and tidy the `ages` variable in `query.go`

> **F9 fix:** After removing `chunkSourceAges`, `readManifest` loses its only caller in `recency.go` (it was called only inside `chunkSourceAges` at recency.go:76). `readManifest` is defined in `ingest.go` (not `recency.go`) and is also called by `RunIngest` (line 78 in `ingest.go`). Removing `chunkSourceAges` does NOT create a dead `readManifest` — the function lives in `ingest.go` and keeps its callers there. `sourceAgeDays` is defined in `recency.go` and is ONLY called from `chunkSourceAges`. After removing `chunkSourceAges`, `sourceAgeDays` becomes dead code and must also be removed (or moved if still needed elsewhere). Verify with grep before deleting.

- [ ] **Write the failing test.** (None needed — this is dead-code removal; the compile will fail if any caller remains.)

  Before deleting, run:
  ```
  grep -n "chunkSourceAges" /Users/joe/repos/personal/engram/internal/cli/*.go
  ```
  Expected: only the definition in `recency.go` (all callers removed in Tasks 2.7–2.8).

  Also verify `sourceAgeDays` callers:
  ```
  grep -n "sourceAgeDays" /Users/joe/repos/personal/engram/internal/cli/*.go
  ```
  Expected: definition in `recency.go` + the `ExportSourceAgeDays` alias in `export_test.go` + the test in `recency_test.go:498`. If `ExportSourceAgeDays` and `TestSourceAgeDays` are the only remaining references, keep `sourceAgeDays` (it is still exercised by the existing test and exported for test coverage). If it has NO non-test callers and the test is no longer meaningful post-migration, remove both. Default: retain `sourceAgeDays` since it is independently useful (converts mtimes to age-days) and is tested.

- [ ] **Delete `chunkSourceAges`** from `/Users/joe/repos/personal/engram/internal/cli/recency.go`. Remove the full function body (currently lines 73–88).

- [ ] **Confirm `readManifest` placement.** Run:
  ```
  grep -n "func readManifest" /Users/joe/repos/personal/engram/internal/cli/*.go
  ```
  Expected: `ingest.go` only (already there; no move needed). The call at `recency.go:76` inside `chunkSourceAges` is being deleted with the function itself — `readManifest` remains in `ingest.go` with its existing callers (`RunIngest`).

- [ ] **Remove the `ages` local variable in `query.go`**: confirm the `chunkSourceAges` call that used to populate `ages` is already gone from Task 2.7. Verify the recency block now reads:
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
  Expected: clean. No dead-code lint from `readManifest` (it stays in `ingest.go` with callers). `sourceAgeDays` retained (tested via `ExportSourceAgeDays`/`TestSourceAgeDays`).

---

#### Task 2.10 — Stable chunk-id helper + in-memory id-set (`buildChunkIDSet` returns `map[string]bool`)

> **Contract item 1+2 verification:** Chunk-id = `source#anchor` per spec §3.2. `buildChunkIDSet` keys by `r.Source+"#"+r.Anchor` (NOT `ContentHash`). The function is DI-compliant: injected `listIndexes` and `readFile` funcs, no direct `os.*` calls. Returns `map[string]bool` (not `map[string]struct{}`). Component 5 reuses this function via `AmendDeps` — it does NOT implement a second os.*-based loader.

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
// DI-compliant: accepts injected listIndexes and readFile funcs (no os.* calls).
// Returns map[string]bool for consistent membership testing (Component 5 reuses this).
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

- [ ] **Add the export** to `/Users/joe/repos/personal/engram/internal/cli/export_test.go`:
  ```go
  // ExportBuildChunkIDSet exposes buildChunkIDSet for validation tests.
  // Component 5 reuses buildChunkIDSet (not a second implementation) via AmendDeps injection.
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

- [ ] **Commit.** Stage ALL changed files from Tasks 2.1–2.10 together in one commit (do not commit incrementally after each task):
  ```
  git add internal/chunk/index.go internal/chunk/ingestedat_test.go \
          internal/cli/ingest.go internal/cli/ingest_test.go \
          internal/cli/recency.go internal/cli/recency_test.go \
          internal/cli/recency_eval_test.go \
          internal/cli/export_test.go internal/cli/query.go
  git commit -m "$(cat <<'EOF'
  feat(chunk): append-only index + per-chunk IngestedAt recency (D5)

  - chunk.Record gains IngestedAt time.Time (json omitempty; zero = legacy)
  - IngestDeps gains Now func() time.Time (nil-safe; wired to time.Now in prod)
  - loadPriorVectors → loadPriorRecords (returns full Record, preserves IngestedAt)
  - rebuildIndex → merge-append: keep prior records, add only new-hash chunks, never delete
  - Thread transcript per-session LastTimestamp as IngestedAt (per-session approximation:
    intra-session spread negligible; cross-session distinguished by source; YAGNI for per-row)
  - Markdown chunks use deps.Now() as IngestedAt
  - Migration backfill: zero-IngestedAt records get manifest mtime on first merge (closure-injected)
  - applyChunkRecency drops ageDaysBySource param; reads r.record.IngestedAt directly
  - newestChunkItems drops ages param; sorts by IngestedAt descending (zero sorts last)
  - chunkSourceAges removed (no callers); sourceAgeDays retained (tested)
  - buildChunkIDSet: O(total) load + source#anchor id-set; DI-compliant; map[string]bool
  - recency_eval_test.go migrated to IngestedAt-based pool + ExportApplyChunkRecencyByTime
  - ExportApplyChunkRecencyByTime, ExportNewestChunkItemsByTime, ExportNewScoredChunkWithIngestedAt added

  AI-Used: [claude]
  EOF
  )"
  ```

---

### Risks/notes

1. **`ExportApplyChunkRecency` alias removed:** The `ExportApplyChunkRecency = applyChunkRecency` var alias in `export_test.go` is replaced by the explicit `ExportApplyChunkRecencyByTime` function form. The synthesizer must also grep `recency_eval_test.go` for `ExportApplyChunkRecency` before executing Task 2.7 — that file uses the old 4-arg form at line 221 and must be migrated as part of Task 2.7's GREEN step.

2. **`readManifest` stays in `ingest.go`:** After removing `chunkSourceAges` (Task 2.9), the only call to `readManifest` in `recency.go` is gone. `readManifest` is defined in `ingest.go` (not `recency.go`) where it keeps its other callers (`RunIngest` at line 78). No dead-code issue. `sourceAgeDays` (defined in `recency.go`) is retained because `TestSourceAgeDays` + `ExportSourceAgeDays` still exercise it — confirm with `targ check-full`.

3. **`IngestDeps.Now` nil-safety across existing tests:** Adding `Now func() time.Time` to `IngestDeps` means all existing test fixtures that construct `IngestDeps` literals without `Now` will have `nil` for that field. The plan guards with `if ingestTime.IsZero() && deps.Now != nil` in `ingestSource` — no panic risk. Synthesizer should verify no other added call site omits this guard.

4. **`rebuildIndex` signature includes `backfillTime` from Task 2.3:** Both params (`ingestTime` and `backfillTime`) are added in Task 2.3's GREEN step to avoid two intermediate broken builds (CA-04). Task 2.5 only wires the closure from `ingestSource` — `rebuildIndex` signature is already correct.

5. **`buildChunkIDSet` is the SINGLE id-set implementation:** Component 5 (`AmendDeps.LoadChunkIDs`) must wire this function (not implement a second `os.ReadDir`-based loader). Component 5's `AmendDeps` should inject `listIndexes func(dir string)([]string,error)` and `readFile func(path string)([]byte,error)` and call `buildChunkIDSet`. Production wiring supplies `os.ReadDir`/`os.ReadFile` in `newOsAmendDeps`. The `map[string]bool` return type (not `map[string]struct{}`) applies consistently across Components 2 and 5.

### Component 3: Unified clustering + top-K candidate nomination (D1 + D7)

**Files:** `internal/cli/query.go`, `internal/cli/query_synthesis_test.go`, `internal/cli/query_subgraph_test.go`, `internal/cli/synthesize_l2_property_test.go`, `internal/cli/query_unified_test.go`

**Dependency gate:** Before starting, confirm Component 2 landed:
- [ ] `grep IngestedAt internal/chunk/index.go` — shows `time.Time` field.
- [ ] `targ check-full` — all checks pass (Component 2 is clean).

---

#### Task 3.1 — Rename `nearest_l2` → `candidate_l2s` (struct + payload field)

**Scope:** Rename the singular `queryNearestL2` type and the `NearestL2` field on `queryCluster` to the plural `queryCandidateL2` / `CandidateL2s []queryCandidateL2`. Wire a temporary stub that returns a 1-element slice so existing behavior is preserved. RED-then-GREEN: the new test asserts the payload emits `candidate_l2s` (a sequence) and has no `nearest_l2` key.

- [ ] **Write the failing test.** Add to `internal/cli/query_synthesis_test.go`:

```go
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

In `internal/cli/query.go`, locate lines ~290–296 containing the `queryNearestL2` struct definition. Replace:
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

In `queryCluster` (line ~260), locate the `NearestL2` field. Replace:
```go
	NearestL2  *queryNearestL2      `yaml:"nearest_l2,omitempty"`
```
With:
```go
	CandidateL2s []queryCandidateL2 `yaml:"candidate_l2s,omitempty"`
```

- [ ] **Fix all compile errors caused by the rename in one pass.** Do NOT run `targ test` after each change — collect all sites first, then fix together.

  **CA-12 note:** `nearestL2ForTier` is called from exactly two places: `renderClusters` (~line 1833) and `clusterChunkItems` (~line 695). It MUST NOT be deleted until BOTH call sites are replaced with `candidateL2sStub` below. Replace both call sites first, then remove `nearestL2ForTier`.

  In `renderClusters` (~line 1833), replace:
  ```go
  				NearestL2:  nearestL2ForTier(centroid, l2Notes, tiers),
  ```
  With (temporarily — the real function comes in Task 3.2):
  ```go
  				CandidateL2s: candidateL2sStub(centroid, l2Notes, tiers),
  ```

  In `clusterChunkItems` (~line 695), replace:
  ```go
  			NearestL2:  nearestL2ForTier(autoK.Centroids[clusterID], l2Notes, tiers),
  ```
  With:
  ```go
  			CandidateL2s: candidateL2sStub(autoK.Centroids[clusterID], l2Notes, tiers),
  ```

  Add the stub function below `clusterChunkItems` (or below `renderClusters`):
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

  Now that both call sites are replaced, remove `nearestL2ForTier` (now dead; `nearestInTierIndex` remains — it is still called by `nearestL3For` and by the stub above).

- [ ] **Update `queryParsed` in `internal/cli/query_subgraph_test.go`** (lines ~169–173). Locate the `NearestL2` field definition in the `queryParsed` struct. Replace:
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

- [ ] **Update `synthesize_l2_property_test.go`** (references `cluster.NearestL2` at lines ~74–77). Locate the assertion block. Replace:
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
  				"the nearest L2 (first candidate) must report raw centroid cosine >= 0.95")
  		}
  ```

- [ ] **Update `query_unified_test.go`** (lines ~60–77). Locate `NearestL2 *struct{...}` in the local parsed struct. Replace `NearestL2 *struct{...} \`yaml:"nearest_l2"\`` → `CandidateL2s []struct{...} \`yaml:"candidate_l2s"\`` and update the assertion `c.NearestL2 != nil && c.NearestL2.Path != ""` → `len(c.CandidateL2s) > 0 && c.CandidateL2s[0].Path != ""`.

- [ ] **Update all `cluster.NearestL2` references in `query_synthesis_test.go`** (~10 references). Apply these substitutions throughout the file:
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

**Scope:** Add the real top-K logic. Write the critical "covering L2 is not #1 but appears in top-K" test BEFORE wiring the function. The spec confirms: top-K by **centroid cosine** (not max-member cosine — rejected because it overfits to a cluster fragment and masks multi-theme clusters). The sort key is `max(situation-cosine, body-cosine)` to the centroid, consistent with `nearestInTierIndex`.

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

	// Cosines must be descending (centroid cosine sort order).
	for i := 1; i < len(candidates); i++ {
		prev := candidates[i-1].(map[string]any)
		curr := candidates[i].(map[string]any)
		prevCos, _ := prev["cosine"].(float64)
		currCos, _ := curr["cosine"].(float64)
		g.Expect(prevCos).To(BeNumerically(">=", currCos),
			"candidate_l2s must be sorted centroid-cosine desc")
	}
}
```

Also add the K≥3 test (this one fails with the stub since stub returns only 1):
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
			"candidate_l2s must be sorted centroid-cosine desc (index %d >= %d)", i-1, i)
	}
}
```

- [ ] Run `targ test 2>&1 | grep -E "FAIL|PASS"` — both new tests fail (stub returns only 1 entry).

- [ ] **Implement `topKCandidateL2s` and `topKCandidateL2sForTier`** in `internal/cli/query.go`. Add after `nearestInTierIndex` (~line 1578) and before `nearestL3ForTier`:

```go
// candidateL2K is the minimum number of candidate L2s to nominate per cluster.
// The recall skill reads all K candidates to judge coverage; generous nomination
// costs nothing (recall is the binary's job, precision is the agent's).
const candidateL2K = 3

// topKCandidateL2s returns the top-K L2 notes nearest the centroid by
// max(situation,body) cosine, sorted descending by centroid cosine (ties broken
// by lexicographic path for stability). K is at least candidateL2K; when fewer
// than candidateL2K L2 notes exist, all are returned. An empty index returns nil.
// No cosine threshold is applied — nomination is generous (D7).
// Sort key is CENTROID cosine (per spec §3.3: "top-K by centroid cosine");
// max-member cosine was rejected because it overfits to a cluster fragment.
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

- [ ] **Replace the stub** everywhere it is called. In `renderClusters` (~line 1833):
  ```go
  			CandidateL2s: candidateL2sStub(centroid, l2Notes, tiers),
  ```
  →
  ```go
  			CandidateL2s: topKCandidateL2sForTier(centroid, l2Notes, tiers),
  ```

  In `clusterChunkItems` (~line 695):
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

**Scope:** Add chunk loading + scoring inside `runSynthesizeL2Query` and extend the subgraph members for D1's "one clustering over the matched set." Diff context: in `internal/cli/query.go`, the current `runSynthesizeL2Query` body (lines ~2062–2094) is a complete replacement. Preserve the function signature `func runSynthesizeL2Query(ctx, args, notes, hits, limit, deps, stdout)` and the opening `l1l2Hits := filterHitsToTiers(...)` + `nowL2` block unchanged. Replace from the `union, err := unionDirectHits(...)` line (line ~2069) through `return renderQueryPayload(stdout, merged)` (line ~2094) with the new body below.

**docs-F3 justification:** The plan keeps `mergeProvenances(union, expandedSubgraph{}, ...)` (empty subgraph). This is deliberate and correct per spec §2 step 2: items below the match threshold are context for the agent, not synthesis inputs; the resolved items for `--synthesize-l2` come from the direct hits union (returned by `unionDirectHits`) plus chunk items appended separately. Cluster representatives are not promoted into `items[]` in this mode because the L2 representative is agent-decided, not binary-computed — passing the real subgraph to `mergeProvenances` would incorrectly promote cluster reps as items, bypassing the agent's coverage judgment. The `phrasedCluster` carries `subgraph` (with chunk members) for `renderClusters` → `collectClusterMembers`, so cluster member paths (including chunks) are correctly reported in the YAML output. The empty `expandedSubgraph{}` passed to `mergeProvenances` means only `mergeClusterReps` and `mergeHubItems` are no-ops — the direct-hit items path (`for _, hit := range directHits`) is unaffected.

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

- [ ] **Add `kind` field to `subgraphMember`** in `internal/cli/query.go`. Locate the struct at lines ~365–373. Replace:

```go
// subgraphMember bundles a node's basename, vault-relative path,
// sidecar vector, query-similarity score, and (optionally) cached body.
type subgraphMember struct {
	basename string
	notePath string
	vector   []float32
	score    float32
	content  string
}
```
With:
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

- [ ] **CA-09 fix — `collectClusterMembers` works correctly for chunk members.** Verify the analysis:

  `collectClusterMembers` (lines ~798–840) uses `member.notePath` directly for `queryClusterMember.Path` — no `basename` lookup. `memberMatchesTier` (line ~1283) calls `itemMatchesTier(resolvedItem{content: member.content}, tiers)`. In `runSynthesizeL2Query`, `tiers=nil` flows through `aggregatedSummary.tiers=nil` to `renderClusters`, so `memberMatchesTier` returns `true` unconditionally for chunk members (the `len(tiers)==0` early-return at line 1284 fires). Chunk `notePath` is already `source#anchor` (from `chunkNotePath()`), so `Path` in the YAML output is correct.

  The `basename` field on chunk `subgraphMember` will be `""` (empty string). This is safe because `mergeProvenances` receives `expandedSubgraph{}` (empty) in this function — `mergeClusterReps` and `mergeHubItems` never iterate the chunk members. However, set `basename` on chunk members to the chunk notePath to prevent `byBasename[""]` key collisions if any future refactor passes the real subgraph to `mergeProvenances`:

  In the chunk-append loop in the new `runSynthesizeL2Query` body below, set `basename: path` on the chunk `subgraphMember`.

- [ ] **Replace `runSynthesizeL2Query` body.** In `internal/cli/query.go`, locate `runSynthesizeL2Query` starting at line ~2053. Keep the function signature and the opening block (lines ~2062–2067: `l1l2Hits`, `nowL2`) unchanged. Replace from the `union, err := unionDirectHits(...)` line (~2069) through `return renderQueryPayload(stdout, merged)` (~2094) with:

```go
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
				basename: path, // set to avoid byBasename[""] collisions if subgraph is ever passed to mergeProvenances
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

	// mergeProvenances receives an empty expandedSubgraph{} deliberately:
	// mergeClusterReps/mergeHubItems must not promote cluster reps into items[]
	// because the L2 representative is agent-decided, not binary-computed (spec §2 step 4).
	// Direct-hit items come from the union; chunk items are appended separately below.
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

1. **D5 recency dependency:** `runSynthesizeL2Query` calls `scoreChunks` directly (raw cosine, no recency). This is correct per spec §3.3: clustering coordinates are raw cosine per-item matched vectors (D1). D5 recency weighting applies at distillation time inside the recall skill (D6), not at clustering time in the binary. No conflict as long as D5 doesn't change `scoreChunks`'s signature.

2. **`subgraphMember.kind` field:** The new `kind` field on `subgraphMember` is consumed immediately (in `runSynthesizeL2Query` → resolvedItem building). `collectClusterMembers` and `memberMatchesTier` do not use it — chunk members pass `memberMatchesTier` with `tiers=nil` because the empty-tiers path returns true unconditionally (line 1284). If a future change adds tier filtering to `collectClusterMembers` for the synthesize-l2 path, chunk members (no frontmatter tier) will be dropped unless `memberMatchesTier` is updated to recognize `kind=chunkItemKind`.

3. **`mergeProvenances` with empty subgraph — justified:** The empty `expandedSubgraph{}` passed to `mergeProvenances` in `runSynthesizeL2Query` is correct per spec §2 step 4: the representative L2 is agent-decided, so the binary must not auto-promote cluster reps into items[]. The real subgraph (with chunk members, `basename` set to the chunk notePath) is carried in `phrasedCluster.subgraph` and IS used by `renderClusters` → `collectClusterMembers` to populate cluster member paths in the YAML output. If a future refactor mistakenly passes the real subgraph to `mergeProvenances` here, chunk members have `basename` set (not empty) so they won't collide at `byBasename[""]` — but their presence in items[] would violate the agent-judged coverage model.

4. **CA-12 — `nearestL2ForTier` removal sequence:** `nearestL2ForTier` has exactly two call sites: `renderClusters` (~line 1833) and `clusterChunkItems` (~line 695). Task 3.1 replaces both with `candidateL2sStub` before deleting `nearestL2ForTier`. Task 3.2 then replaces `candidateL2sStub` with `topKCandidateL2sForTier` at both sites before deleting `candidateL2sStub`. Each deletion happens only after ALL call sites of the target function are replaced — never eagerly.

### Component 4: Exclude `Related to:` from the embed source (D3)

**Goal:** `BodyText`/`ContentHash` must ignore a trailing `Related to:` section so a link-only edit (adding or changing `[[wikilinks]]` in that block) does not change the ContentHash or perturb the body vector. Recognition is conservative: the `Related to:` marker line followed only by relation bullets (`- [[…`) and blank lines. Inline prose mentioning "Related to:" is left intact.

**Files:** `internal/embed/hash.go`, `internal/embed/hash_test.go`, `internal/embed/hash_property_test.go`

---

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

- [ ] **Verify that `"bytes"` is already present in the import block.** In `/Users/joe/repos/personal/engram/internal/embed/hash.go`, the existing import block (lines 3–7) already imports `"bytes"` as the first entry — no change needed. (CA-13 false alarm: `bytes` was already imported before this task was planned.)

- [ ] Implement the minimal change in `/Users/joe/repos/personal/engram/internal/embed/hash.go`. The edits are in two places:

  **Where:** The `BodyText` function body (currently lines 9–13) and the `const (...)` block (currently lines 77–80). Replace both as follows:

  In `hash.go`, locate `BodyText` (line 9–13) and replace its body so it calls `stripRelatedToSection`:

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

  In `hash.go`, locate the `const (...)` block (currently lines 77–80, containing only `frontmatterDelim`) and replace it with the extended block that adds `relatedSectionMarker` and `relatedSectionBulletPfx`:

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
  ```

  Then, after the `extractFrontmatterField` function (currently ending at line 96), append the two new unexported functions. **Where:** end of file, after line 96.

  ```go
  // stripRelatedToSection removes a trailing "Related to:" relation block from
  // body, returning body unchanged when no such block is present. The block is
  // recognised conservatively (see isRelatedToBlock): a "Related to:" marker
  // line whose following non-blank lines are all relation bullets. Recognising
  // only the LAST marker, and only when the lines after it qualify, leaves prose
  // that mentions "Related to:" inline untouched.
  //
  // Implementation note: bytes.Split(body, "\n") on a newline-terminated body
  // produces a trailing empty element. Lines[:i] for i pointing at the marker
  // therefore ends with the blank line(s) before the marker — joining with "\n"
  // faithfully restores the body up to and including its final trailing newline.
  // Do NOT bytes.TrimRight the result: that would remove the single trailing
  // newline that is part of the body (CA-15 fix).
  func stripRelatedToSection(body []byte) []byte {
  	lines := bytes.Split(body, []byte("\n"))

  	for i := len(lines) - 1; i >= 0; i-- {
  		if bytes.Equal(bytes.TrimRight(lines[i], "\r"), []byte(relatedSectionMarker)) {
  			if isRelatedToBlock(lines[i+1:]) {
  				return bytes.Join(lines[:i], []byte("\n"))
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

  **Trailing-newline invariant (CA-15 fix):** `bytes.Split("...\nRelated to:\n- bullet.\n", "\n")` produces a trailing empty-string element. When the marker is at index `i`, `lines[:i]` ends with the blank-line elements between the body prose and the marker. `bytes.Join(lines[:i], "\n")` reconstructs the body including its final `"\n"` — the expected `want` value `"Information learned: when in X, S P O.\n"` matches exactly. The removed `bytes.TrimRight(..., "\n")` call from the original plan would have stripped that trailing newline and failed the test.

- [ ] Run the test, confirm it PASSES:

  ```
  targ test
  ```

  Expected: `TestBodyText_ExcludesRelatedToSection` passes; all prior `BodyText`/`ContentHash` tests still pass.

- [ ] Run full checks:

  ```
  targ check-full
  ```

  Expected: clean. (`stripRelatedToSection`, `isRelatedToBlock`, and both new constants are all referenced — no unused-symbol lint.)

---

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

---

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

---

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

---

#### Task 4.5 — Verify `ComputeState` correctly treats link-only changes as OK (not stale)

- [ ] Add to `/Users/joe/repos/personal/engram/internal/embed/state_test.go` after `TestComputeState_Stale` (before the `fakeFS` type):

  **Where:** `internal/embed/state_test.go` — append two new test functions after the existing `TestComputeState_Stale` function.

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

- [ ] **Where:** In `/Users/joe/repos/personal/engram/internal/cli/embed.go`, locate the `Force` field in the `EmbedApplyArgs` struct and replace the bare struct tag line with an explicit doc comment + tag pair:

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

**Files:** `internal/cli/amend.go` (new), `internal/cli/amend_test.go` (new), `internal/cli/learn.go` (add `ChunkSources` to `LearnArgs`/frontmatter docs), `internal/cli/relations.go` (add `resolveRelationTargetsStrict`), `internal/cli/targets.go` (wire `amend`; thread `--chunk-source` through learn lambdas), `internal/cli/export_test.go` (Export aliases)

**Chunk-id scheme (spec §3.2, locked):** The stable chunk id throughout this component is `source#anchor` — the string `r.Source + "#" + r.Anchor`. Frontmatter `sources:` values are `source#anchor` strings; `--chunk-source` arguments are `source#anchor` strings; validation checks membership in a `map[string]bool` keyed by `source#anchor`. `ContentHash` is NOT used as a chunk-id here.

**Strand dependency order (CLR-004):**

```
Strand A (pure logic, no prereq):
  5.1 — resolveRelationTargetsStrict

Strand B (scaffold + relation, depends on A):
  5.2 — ChunkSources field on LearnArgs + frontmatter sources:  (prereq: 5.1)
  5.3 — AmendArgs + AmendDeps scaffold                          (prereq: 5.2)
  5.4 — Relation-merge (idempotent Related to:)                 (prereq: 5.3)

Strand C (chunk-id loader, no prereq / parallel with A):
  5.8 — DI-compliant chunk-id set via buildChunkIDSet           (no prereq)

Strand B+C merge:
  5.5 — Provenance-merge (sources: frontmatter)                  (prereq: 5.3 + 5.8)

Strand D (content + activate, depends on B+C):
  5.6 — Field-replacement (situation/subject/predicate/object)   (prereq: 5.5)
  5.7 — --activate sidecar bump                                  (prereq: 5.6)

Wire:
  5.9 — Wire amend in targets.go; thread --chunk-source in learn (prereq: 5.6)

Integration:
  5.10 — Round-trip integration test                             (prereq: 5.9)
```

Strands A and C can run in parallel. Strand B waits for A. B and C merge at 5.5. D depends on B+C. Wiring (5.9) depends on D. Integration (5.10) is last.

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

- [ ] **Implement** in `internal/cli/relations.go`. `relatedSectionMarker` and `indexBasenamesByID` already live in `relations.go` — add after the existing helpers:

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

Add `"errors"` to the import block of `internal/cli/relations.go` if not present.

- [ ] Add export alias to `export_test.go` (inside `package cli`, not `package cli_test`):

```go
// ExportResolveRelationTargetsStrict exposes resolveRelationTargetsStrict for relation tests.
var ExportResolveRelationTargetsStrict = resolveRelationTargetsStrict
```

- [ ] Run `targ test` — expect all four subtests pass.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.2 — `ChunkSources` field on `LearnArgs` + frontmatter `sources:` key

**Prerequisite:** 5.1

**Dependency note (CA-06):** `applyFactAmend` in Task 5.5 calls `yaml.Unmarshal` into `factFrontmatterDoc` and accesses `doc.Sources`. That field does NOT exist today. This task adds it. If Task 5.2 is skipped or the `Sources` field is missing from `factFrontmatterDoc`, `yaml.Unmarshal` will silently zero `doc.Sources` (Go zero-initializes absent YAML fields into struct fields), and the provenance merge in Task 5.5 will silently drop existing sources. Task 5.5 must not execute until Task 5.2 is complete and `targ check-full` passes.

The chunk-ids passed via `--chunk-source` are `source#anchor` strings (spec §3.2), e.g. `/sessions/s.jsonl#turn-1`. Tests assert the written frontmatter contains the `source#anchor` form, not `sha256:...` content hashes.

- [ ] **Write failing test** in `internal/cli/learn_test.go` (new subtests `TestLearnFact_ChunkSources_*`):

```go
// internal/cli/learn_test.go (appended)

func TestLearnFact_ChunkSources_WrittenToFrontmatter(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    var written []byte
    args := cli.LearnArgs{
        Type: "fact", Slug: "test-slug", Vault: t.TempDir(),
        Source: "test", Situation: "testing chunk sources",
        Subject: "A", Predicate: "has", Object: "B",
        ChunkSources: []string{"/sessions/s.jsonl#turn-1", "/sessions/s.jsonl#turn-2"},
    }
    deps := cli.LearnDeps{
        Now:           func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
        Getenv:        func(string) string { return "" },
        StatDir:       func(string) error { return nil },
        InitVault:     func(string) error { return nil },
        ListIDs:       func(string) ([]string, error) { return nil, nil },
        ListBasenames: func(string) ([]string, error) { return nil, nil },
        Lock:          func(string) (func(), error) { return func() {}, nil },
        WriteNew:      func(_ string, data []byte) error { written = data; return nil },
    }

    var buf strings.Builder
    err := cli.ExportRunLearn(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    g.Expect(string(written)).To(gomega.ContainSubstring("sources:"))
    g.Expect(string(written)).To(gomega.ContainSubstring("/sessions/s.jsonl#turn-1"))
    g.Expect(string(written)).To(gomega.ContainSubstring("/sessions/s.jsonl#turn-2"))
}

func TestLearnFact_EmptyChunkSources_NoSourcesKey(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    var written []byte
    args := cli.LearnArgs{
        Type: "fact", Slug: "test-slug", Vault: t.TempDir(),
        Source: "test", Situation: "no chunk sources",
        Subject: "A", Predicate: "has", Object: "B",
    }
    deps := cli.LearnDeps{
        Now:           func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
        Getenv:        func(string) string { return "" },
        StatDir:       func(string) error { return nil },
        InitVault:     func(string) error { return nil },
        ListIDs:       func(string) ([]string, error) { return nil, nil },
        ListBasenames: func(string) ([]string, error) { return nil, nil },
        Lock:          func(string) (func(), error) { return func() {}, nil },
        WriteNew:      func(_ string, data []byte) error { written = data; return nil },
    }

    var buf strings.Builder
    err := cli.ExportRunLearn(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    g.Expect(string(written)).NotTo(gomega.ContainSubstring("sources:"))
}
```

- [ ] Run `targ test` — expect compile failure (no `ChunkSources` on `LearnArgs` yet).

- [ ] **Implement** — add `ChunkSources []string` to `LearnArgs` in `internal/cli/learn.go`:

In `LearnArgs` (after the existing `Relations []string` field):
```go
// ChunkSources carries chunk-index ids (source#anchor) to record as frontmatter
// provenance. Written to `sources:` when non-empty. Passed via --chunk-source.
ChunkSources []string
```

Add `ChunkSources []string` to `factFields` (in `internal/cli/learn.go`, after `Tier string`):
```go
ChunkSources []string
```

Add `Sources []string` field to `factFrontmatterDoc` (after the `Issue` field):
```go
Sources []string `yaml:"sources,omitempty"`
```

In `renderFactFrontmatter`, add `Sources: f.ChunkSources` to the `marshalFrontmatter(factFrontmatterDoc{...})` call.

In the `factFields` literal inside `assembleLearnContent` (fact branch), add:
```go
ChunkSources: args.ChunkSources,
```

Add `ChunkSources []string` to `feedbackFields` and `Sources []string yaml:"sources,omitempty"` to `feedbackFrontmatterDoc`, and wire identically through `renderFeedbackFrontmatter` and the feedback branch of `assembleLearnContent`.

Add `ChunkSources []string` to `CommonLearnArgs` in `internal/cli/targets.go`:
```go
ChunkSources []string `targ:"flag,name=chunk-source,desc=chunk-index id (source#anchor) to record as provenance (repeatable)"`
```

Thread `ChunkSources` from `CommonLearnArgs` through `runLearnFromFactArgs` and `runLearnFromFeedbackArgs` bridge functions (copy `a.ChunkSources` into the `LearnArgs` literal).

- [ ] Run `targ test` — subtests pass.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.3 — `AmendArgs` + `AmendDeps` structs + `findNote` reuse

**Prerequisite:** 5.2

**Key design notes:**
- `ChunksDir` lives on `AmendArgs` (not `AmendDeps`) — consistent with `IngestArgs.ChunksDir` and `QueryArgs.ChunksDir`. Deps hold I/O functions, not path configuration (F7).
- `LoadChunkIDs` on `AmendDeps` uses the DI-compliant signature from Component 2 Task 2.10's `buildChunkIDSet`: it takes injected `listIndexes` and `readFile` functions, returns `map[string]bool`. The production wiring in `newOsAmendDeps` supplies `os.ReadDir`-based helpers (F4, CA-14).
- The `"time"` import is required in `amend.go` because `AmendDeps.Now` is `func() time.Time` (CA-07).

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

- [ ] **Implement** scaffolding in new file `internal/cli/amend.go`. Note: `relatedSectionMarker` is already defined in `relations.go` (same `cli` package) — do NOT redefine it. The `"time"` import is required (CA-07):

```go
package cli

import (
    "context"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/toejough/engram/internal/embed"
    "github.com/toejough/engram/internal/vaultgraph"
)

// AmendArgs holds parsed flags for `engram amend`. ChunksDir configures where
// chunk indexes live (like IngestArgs.ChunksDir — path config belongs on Args,
// not Deps).
type AmendArgs struct {
    Vault        string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
    Target       string   `targ:"flag,name=target,required,desc=Luhmann id or full basename of the note to amend (required)"`
    Relations    []string `targ:"flag,name=relation,desc=note relation as <wikilink-target>|<rationale> to merge into Related to: (repeatable)"`
    ChunkSources []string `targ:"flag,name=chunk-source,desc=chunk-index id (source#anchor) to merge into frontmatter sources: (repeatable)"`
    ChunksDir    string   `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
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

// AmendDeps holds injected I/O dependencies for RunAmend. Path configuration
// (ChunksDir) lives on AmendArgs, not here.
//
// LoadChunkIDs is DI-compliant: it takes injected listIndexes and readFile
// functions (matching buildChunkIDSet from Component 2) and returns a
// map[string]bool keyed by "source#anchor". The production wiring in
// newOsAmendDeps supplies os.ReadDir/os.ReadFile via closures.
type AmendDeps struct {
    Scan          func(vault string) ([]vaultgraph.Note, error)
    Read          func(path string) ([]byte, error)
    Write         func(path string, data []byte) error
    Embedder      embed.Embedder
    Now           func() time.Time
    ListBasenames func(vault string) ([]string, error)
    LoadChunkIDs  func(
        chunksDir string,
        listIndexes func(dir string) ([]string, error),
        readFile func(path string) ([]byte, error),
    ) (map[string]bool, error)
    ListIndexes func(dir string) ([]string, error)
    LogWarning  func(string, ...any)
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
        return fmt.Errorf("%w: %q", errAmendNoteNotFound, args.Target)
    }
    _ = relPath // used in full implementation (Task 5.4+)

    return nil // stub — replaced in Task 5.4
}

// newOsAmendDeps wires RunAmend to the real filesystem + bundled embedder.
// ChunksDir flows through AmendArgs, not here.
func newOsAmendDeps() AmendDeps {
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
        Embedder: sharedEmbedder,
        Now:      time.Now,
        ListBasenames: func(vault string) ([]string, error) {
            return (&osLearnFS{}).ListBasenames(vault)
        },
        LoadChunkIDs: buildChunkIDSet,
        ListIndexes: func(dir string) ([]string, error) {
            entries, err := os.ReadDir(dir)
            if err != nil {
                return nil, nil // absent dir = empty, not an error
            }
            paths := make([]string, 0, len(entries))
            for _, e := range entries {
                if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") && e.Name() != manifestName {
                    paths = append(paths, filepath.Join(dir, e.Name()))
                }
            }
            return paths, nil
        },
        LogWarning: logWarningToStderrf,
    }
}
```

- [ ] Add export alias to `export_test.go` (inside `package cli`):

```go
// ExportRunAmend exposes RunAmend for amend unit tests.
var ExportRunAmend = RunAmend

// ExportNewOsAmendDeps exposes newOsAmendDeps for integration tests.
var ExportNewOsAmendDeps = newOsAmendDeps

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
        LoadChunkIDs: func(string, func(string)([]string,error), func(string)([]byte,error)) (map[string]bool, error) {
            return map[string]bool{}, nil
        },
        Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
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
        LoadChunkIDs: func(string, func(string)([]string,error), func(string)([]byte,error)) (map[string]bool, error) {
            return map[string]bool{}, nil
        },
        Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
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
    // Should contain the relation exactly once.
    body := string(written)
    count := strings.Count(body, "[["+relBasename+"]]")
    g.Expect(count).To(gomega.Equal(1))
}
```

- [ ] Run `targ test` — tests fail (stub returns nil with no write).

- [ ] **Implement** relation-merge logic in `RunAmend`. Add helper `mergeRelatedSection` in `amend.go`. Note: `relatedSectionMarker` and `wikilinkRE` are already defined in the `cli` package (`relations.go` and `query.go` respectively) — reuse them, do not redeclare:

```go
// mergeRelatedSection parses the existing "Related to:" block from body,
// deduplicates with incoming relations, and returns the updated body with
// only new relations appended. Incoming relations must already be in
// "basename|rationale" resolved form. Existing bullets "- [[basename]] — ..."
// are parsed for their basename to detect duplicates.
func mergeRelatedSection(body string, incoming []string) string {
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

    if idx == -1 {
        tail := relatedSectionMarker + "\n" + strings.Join(newBullets, "\n") + "\n"
        return strings.TrimRight(body, "\n") + "\n\n" + tail
    }

    trimmed := strings.TrimRight(existingSection, "\n")
    return head + trimmed + "\n" + strings.Join(newBullets, "\n") + "\n"
}
```

Replace the `RunAmend` stub with the full implementation that validates chunk sources, resolves relations, merges content, writes, optionally re-embeds, and optionally activates:

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
        chunkIDs, loadErr := deps.LoadChunkIDs(args.ChunksDir, deps.ListIndexes, deps.Read)
        if loadErr != nil {
            return fmt.Errorf("amend: loading chunk ids: %w", loadErr)
        }
        for _, id := range args.ChunkSources {
            if !chunkIDs[id] {
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

    // merge chunk sources into frontmatter + apply field overrides
    updated, contentChanged, fieldErr := applyFieldReplacement(raw, args, bodyStr, noteType)
    if fieldErr != nil {
        return "", false, fieldErr
    }

    return updated, contentChanged, nil
}

// writeAmendedSidecar re-embeds the amended note and writes its sidecar.
// Modeled on writeResituatedSidecar in resituate.go. Embed and write failures
// are returned to the caller (which may choose to warn-and-continue for amend).
func writeAmendedSidecar(ctx context.Context, deps AmendDeps, notePath, content string) error {
    sidecar, embErr := embed.BuildSidecar(ctx, deps.Embedder, []byte(content))
    if embErr != nil {
        return fmt.Errorf("amend: embedding %s: %w", notePath, embErr)
    }

    writeErr := deps.Write(embed.SidecarPath(notePath), embed.MarshalSidecar(sidecar))
    if writeErr != nil {
        return fmt.Errorf("amend: writing sidecar for %s: %w", notePath, writeErr)
    }

    return nil
}
```

- [ ] Run `targ test` — relation-merge tests pass.
- [ ] Run `targ check-full` — fix any issues.

---

#### Task 5.5 — Provenance-merge (idempotent `sources:` frontmatter)

**Prerequisite:** 5.3 + 5.8

**Dependency note (CA-06, explicit):** `applyFactAmend` below calls `yaml.Unmarshal` into `factFrontmatterDoc` and reads `doc.Sources`. That field is added in Task 5.2. Do NOT execute this task until Task 5.2 is merged and `targ check-full` passes — if `Sources []string` is absent from `factFrontmatterDoc`, `yaml.Unmarshal` will silently zero `doc.Sources` and the merge will silently drop existing provenance.

The `--chunk-source` arguments and the written `sources:` values are `source#anchor` strings (e.g. `/sessions/s.jsonl#turn-1`), validated against the `buildChunkIDSet` id-set. The tests below use `source#anchor` form, not `sha256:...`.

- [ ] **Write failing tests** `TestRunAmend_ProvMerge_*`:

```go
func TestRunAmend_ProvMerge_ChunkSources_Written(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    const basename = "1aa.2026-01-01.test.md"
    const chunkID = "/sessions/s.jsonl#turn-1"
    noteContent := makeFactNote("ctx", "A", "has", "B", "")

    var written []byte
    deps := cli.AmendDeps{
        Scan: func(string) ([]vaultgraph.Note, error) {
            return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
        },
        Read:  func(string) ([]byte, error) { return noteContent, nil },
        Write: func(_ string, data []byte) error { written = data; return nil },
        ListBasenames: func(string) ([]string, error) { return []string{basename}, nil },
        LoadChunkIDs: func(string, func(string)([]string,error), func(string)([]byte,error)) (map[string]bool, error) {
            return map[string]bool{chunkID: true}, nil
        },
        Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
    }
    args := cli.AmendArgs{
        Vault:        "/vault",
        Target:       "1aa",
        ChunkSources: []string{chunkID},
    }

    var buf bytes.Buffer
    err := cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    g.Expect(string(written)).To(gomega.ContainSubstring("sources:"))
    g.Expect(string(written)).To(gomega.ContainSubstring(chunkID))
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
        LoadChunkIDs: func(string, func(string)([]string,error), func(string)([]byte,error)) (map[string]bool, error) {
            return map[string]bool{}, nil // empty — id won't resolve
        },
    }
    args := cli.AmendArgs{
        Vault:        "/vault",
        Target:       "1aa",
        ChunkSources: []string{"/sessions/s.jsonl#turn-1"},
        ChunksDir:    "/chunks",
    }

    var buf bytes.Buffer
    err := cli.ExportRunAmend(context.Background(), args, deps, &buf)
    g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("unresolved chunk-source id")))
}
```

- [ ] Run `targ test` — fail (chunk-sources not merged into frontmatter yet).

- [ ] **Implement** provenance merge. Add helper `mergeChunkSources` and implement `applyFieldReplacement` + `applyFactAmend` in `amend.go`:

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
```

- [ ] Run `targ test` — provenance tests pass.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.6 — Field-replacement: only supplied fields overwritten, preserved fields intact

**Prerequisite:** 5.5

**CA-11 (Issue round-trip):** `factFrontmatterDoc.Issue` is `quotedString` (learn.go:240). When `applyFactAmend` reads `doc.Issue` and populates `factFields.Issue` (which is plain `string`), it must cast `string(doc.Issue)`. When rebuilding via `renderFactFrontmatter`, `factFields.Issue` is passed to `quotedString(f.Issue)`. This round-trip is safe. However, when `factFrontmatterDoc` is populated inside `applyFactAmend` to call `marshalFrontmatter` directly, `Issue` must be set as `quotedString(string(doc.Issue))` to preserve the double-quoted YAML style. The `factFields`-based path through `renderFactFrontmatter` handles this automatically — do not bypass it.

- [ ] **Write failing tests** `TestRunAmend_FieldReplacement_*`:

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
        LoadChunkIDs: func(string, func(string)([]string,error), func(string)([]byte,error)) (map[string]bool, error) {
            return nil, nil
        },
        Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
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
        LoadChunkIDs: func(string, func(string)([]string,error), func(string)([]byte,error)) (map[string]bool, error) {
            return nil, nil
        },
        Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
        Embedder: &spyEmbedder{called: &embedCalled},
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

Add `spyEmbedder` to `amend_test.go` (in `package cli_test`):

```go
// spyEmbedder is an embed.Embedder that records whether Embed was called.
type spyEmbedder struct {
    called *bool
}

func (s *spyEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
    if s.called != nil {
        *s.called = true
    }
    return []float32{0.1}, nil
}
```

- [ ] Run `targ test` — new field-replacement tests fail (no full implementation of `applyFactAmend` yet).

- [ ] **Complete** `applyFactAmend` in `amend.go`. The `Issue` field round-trips via `factFields.Issue = string(doc.Issue)` into `renderFactFrontmatter` which wraps it as `quotedString(f.Issue)` — no field loss (CA-11):

```go
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
    if args.Subject != "" && args.Subject != doc.Subject {
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

    // provenance merge — source#anchor ids, deduped
    doc.Sources = mergeChunkSources(doc.Sources, args.ChunkSources)

    // Round-trip Issue through string(doc.Issue) → factFields.Issue → quotedString(f.Issue)
    // in renderFactFrontmatter. This preserves the double-quoted YAML style (CA-11).
    f := factFields{
        Situation:    doc.Situation,
        Subject:      doc.Subject,
        Predicate:    doc.Predicate,
        Object:       doc.Object,
        Luhmann:      string(doc.Luhmann),
        Source:       doc.Source,
        Project:      doc.Project,
        Issue:        string(doc.Issue),
        Tier:         doc.Tier,
        ChunkSources: doc.Sources,
    }

    // If content changed, rebuild body formula with new field values.
    relatedSection := relatedTailFromBody(body)
    if contentChanged {
        body = renderFactBody(f, relatedSection)
    }

    return renderFactFrontmatter(f, when) + body, contentChanged, nil
}

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

Add `applyFeedbackAmend` with the same pattern (using `feedbackFields.Behavior`/`Impact`/`Action` and `renderFeedbackFrontmatter`).

Add `"go.yaml.in/yaml/v3"` import to `amend.go` import block.

- [ ] Run `targ test` — all field-replacement tests pass.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.7 — `--activate` in one write (sidecar bump on content-change path)

**Prerequisite:** 5.6

The spec says `--activate` bumps `LastUsed` in the sidecar. Two sub-cases:
- Content changed → re-embed already wrote a fresh sidecar → `bumpLastUsed` reads the freshly-written sidecar.
- No content change → sidecar exists unchanged → `bumpLastUsed` reads the existing sidecar.

Both are handled by the `bumpLastUsed` call in `RunAmend` after the optional re-embed. The ordering guarantee is that `Write` is called first (note file), then re-embed writes the sidecar, then `bumpLastUsed` reads and re-writes the sidecar. This is one logical amend in three sequential writes; the spec's "one write" means a single invocation of `amend` (not a single filesystem write), so this is correct.

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
        LoadChunkIDs: func(string, func(string)([]string,error), func(string)([]byte,error)) (map[string]bool, error) {
            return nil, nil
        },
        Now: func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) },
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

#### Task 5.8 — DI-compliant chunk-id set via `buildChunkIDSet` (Component 2 reuse)

**Prerequisite:** none (parallel with Strand A)

**Key design (F4, CA-14, docs-F1):** This task does NOT implement a new `loadChunkIDsFromDir` function with direct `os.ReadDir`/`os.ReadFile` calls — that would violate DI-everywhere. Instead, it reuses `buildChunkIDSet` from Component 2 Task 2.10 (`internal/cli/ingest.go`), which accepts injected `listIndexes` and `readFile` functions and returns `map[string]bool` keyed by `source#anchor`. `AmendDeps.LoadChunkIDs` is typed to match that signature. The production wiring in `newOsAmendDeps` supplies `os.ReadDir`-based closures — all `os.*` calls stay at the wire edge, outside `internal/` business logic.

The chunk-ids in the set are `source#anchor` strings (spec §3.2 locked contract). Test fixtures must use `source#anchor` ids, not `sha256:...` content hashes.

- [ ] **Write failing test** in `internal/cli/amend_test.go` — `TestBuildChunkIDSet_ViaAmendDeps_*`:

  (Note: `buildChunkIDSet` is already tested in Component 2 Task 2.10 via `ExportBuildChunkIDSet`. The test here validates that the `AmendDeps.LoadChunkIDs` wire path correctly routes to it with DI-injected functions and produces `source#anchor` keys.)

```go
func TestBuildChunkIDSet_ReturnsSourceAnchorKeys(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    r1 := chunk.Record{
        Source: "/sessions/a.jsonl", Anchor: "turn-1",
        ContentHash: "sha256:aaa", Text: "hi", Vector: []float32{0.1},
    }
    r2 := chunk.Record{
        Source: "/docs/b.md", Anchor: "Heading",
        ContentHash: "sha256:bbb", Text: "bye", Vector: []float32{0.2},
    }

    encoded1, err := chunk.EncodeRecords([]chunk.Record{r1})
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }
    encoded2, err := chunk.EncodeRecords([]chunk.Record{r2})
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }

    files := map[string][]byte{
        "/chunks/a.jsonl": encoded1,
        "/chunks/b.jsonl": encoded2,
    }
    readFile := func(path string) ([]byte, error) {
        data, ok := files[path]
        if !ok {
            return nil, fmt.Errorf("not found: %s", path)
        }
        return data, nil
    }
    listIndexes := func(string) ([]string, error) {
        return []string{"/chunks/a.jsonl", "/chunks/b.jsonl"}, nil
    }

    // Simulate the AmendDeps.LoadChunkIDs call pattern.
    ids, loadErr := cli.ExportBuildChunkIDSet("/chunks", listIndexes, readFile)
    g.Expect(loadErr).NotTo(gomega.HaveOccurred())
    if loadErr != nil { return }

    // Ids are source#anchor, NOT content hashes.
    g.Expect(ids["/sessions/a.jsonl#turn-1"]).To(gomega.BeTrue(), "r1 source#anchor must be in set")
    g.Expect(ids["/docs/b.md#Heading"]).To(gomega.BeTrue(), "r2 source#anchor must be in set")
    g.Expect(ids["sha256:aaa"]).To(gomega.BeFalse(), "content hash must NOT be in set")
    g.Expect(ids["nonexistent#anchor"]).To(gomega.BeFalse(), "absent id must not be in set")
}
```

- [ ] Run `targ test` — this test will pass once Component 2 Task 2.10 is merged (which defines `ExportBuildChunkIDSet` and `buildChunkIDSet`). If Component 2 has not landed yet, this test will show a compile error on `ExportBuildChunkIDSet` — that is the expected RED state for this strand.

- [ ] **Wire `buildChunkIDSet` as `AmendDeps.LoadChunkIDs` in `newOsAmendDeps`.** The implementation is already done in Task 5.3's `newOsAmendDeps` body where `LoadChunkIDs: buildChunkIDSet` is set. Verify that `buildChunkIDSet` (from Component 2 Task 2.10, in `internal/cli/ingest.go`) is accessible here — it is in the same `cli` package, so no import needed. Confirm the function signature matches `AmendDeps.LoadChunkIDs`:
  - `buildChunkIDSet(chunksDir string, listIndexes func(string)([]string,error), readFile func(string)([]byte,error)) (map[string]bool, error)` — matches.

- [ ] Run `targ test` — passes.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.9 — Wire `amend` in `targets.go` + thread `--chunk-source` in learn

**Prerequisite:** 5.6

- [ ] **Write failing test** `TestTargets_AmendRegistered` in `targets_test.go` (or a new `amend_targets_test.go`):

```go
func TestTargets_AmendRegistered(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)
    var buf bytes.Buffer
    targets := cli.Targets(&buf, &buf, func(int) {}, nil)
    g.Expect(targets).NotTo(gomega.BeEmpty())
}
```

- [ ] **Wire** in `targets.go` — add to `maintenanceTargets`. Note: `newOsAmendDeps` no longer takes `chunksDir` as argument (that moved to `AmendArgs`); `chunksDir` is resolved and placed in `a.ChunksDir`:

```go
targ.Targ(func(ctx context.Context, a AmendArgs) {
    a.Vault = resolveVault(a.Vault, homeOrEmpty(), os.Getenv)
    a.ChunksDir = ResolveChunksDir(a.ChunksDir, homeOrEmpty(), os.Getenv)
    errHandler(RunAmend(withLog(ctx), a, newOsAmendDeps(), stdout))
}).Name("amend").Description("Amend a note in place: relation-merge, provenance-merge, field-replacement, activate"),
```

- [ ] Also wire `--chunk-source` through the `learn` closures — propagate `a.ChunkSources` in the `LearnArgs` literal inside `runLearnFromFactArgs` and `runLearnFromFeedbackArgs` bridges (add `ChunkSources: a.ChunkSources`). The `LearnFactArgs`/`LearnFeedbackArgs` embed `CommonLearnArgs` which gained `ChunkSources` in Task 5.2 — so `a.ChunkSources` is accessible.

- [ ] Run `targ build` — builds clean.
- [ ] Run `targ test` — new test passes.
- [ ] Run `targ check-full` — clean.

---

#### Task 5.10 — Final integration smoke + `Export*` cleanup

**Prerequisite:** 5.9

The integration test uses only `args.ChunksDir` (not `deps.ChunksDir` — that field no longer exists after F7). There is exactly one source of truth for the chunks directory.

- [ ] **Write integration test** `TestRunAmend_RoundTrip_FactNote` in `amend_test.go`:

```go
func TestRunAmend_RoundTrip_FactNote(t *testing.T) {
    t.Parallel()
    g := gomega.NewWithT(t)

    dir := t.TempDir()
    chunksDir := t.TempDir()
    const basename = "1aa.2026-01-01.test.md"
    const relBasename = "105.2026-01-01.foo.md"
    const chunkID = "/sessions/s.jsonl#turn-1"
    notePath := filepath.Join(dir, basename)

    // write initial note
    noteContent := makeFactNote("original ctx", "OldSubject", "has", "B", "")
    err := os.WriteFile(notePath, noteContent, 0o600)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }

    // write a chunk index with one record
    records := []chunk.Record{{
        Source: "/sessions/s.jsonl", Anchor: "turn-1",
        ContentHash: chunk.HashText("t"),
        Text: "t", Vector: []float32{0.1},
    }}
    data, _ := chunk.EncodeRecords(records)
    err = os.WriteFile(filepath.Join(chunksDir, "s.jsonl"), data, 0o600)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }

    // write a "note" for relBasename
    err = os.WriteFile(filepath.Join(dir, relBasename), makeFactNote("r ctx","X","is","Y",""), 0o600)
    g.Expect(err).NotTo(gomega.HaveOccurred())
    if err != nil { return }

    deps := cli.ExportNewOsAmendDeps()
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
    deps.ListIndexes = func(dir string) ([]string, error) {
        entries, err := os.ReadDir(dir)
        if err != nil {
            return nil, nil
        }
        paths := make([]string, 0, len(entries))
        for _, e := range entries {
            if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
                paths = append(paths, filepath.Join(dir, e.Name()))
            }
        }
        return paths, nil
    }

    args := cli.AmendArgs{
        Vault:        dir,
        Target:       "1aa",
        Subject:      "NewSubject",
        Relations:    []string{relBasename + "|because"},
        ChunkSources: []string{chunkID},
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
    g.Expect(body).To(gomega.ContainSubstring(chunkID))
    g.Expect(body).To(gomega.ContainSubstring("Related to:"))
    g.Expect(body).To(gomega.ContainSubstring("[[" + relBasename + "]]"))
}
```

- [ ] Run `targ test` — passes.
- [ ] Run `targ check-full` — clean (8 checks pass).
- [ ] `targ build` — binary builds.

---

**Risks/notes for the synthesizer:**

1. **D3 dependency (exclude `Related to:` from embed source):** Task 5.6's `contentChanged` gate is critical — if `Related to:` is still in the body hash, a link-only amend will incorrectly trigger re-embed. This component depends on the D3 component (Component 4) being landed first so that `embed.BuildSidecar`/`ContentHash` excludes the `Related to:` block. The plan is self-consistent only when sequenced after Component 4.

2. **`manifestName` constant:** used in `newOsAmendDeps`'s `ListIndexes` closure to skip the manifest file. This constant is defined as `manifestName = "manifest.json"` in `internal/cli/ingest.go:119`. It is unexported but accessible within the same `cli` package — no redeclaration needed.

3. **`wikilinkRE` is in `query.go`:** `mergeRelatedSection` uses `wikilinkRE` (defined in `internal/cli/query.go:175`). Since `amend.go` is in the same `cli` package, this is a valid reference. No redeclaration needed.

4. **`relatedSectionMarker` is in `relations.go`:** Already package-accessible. Do not redeclare in `amend.go`.

5. **`buildChunkIDSet` must be landed (Component 2 Task 2.10):** Task 5.8 reuses it. If Component 2 is not merged when Component 5 executes, there will be a compile error on `buildChunkIDSet` — this is the correct RED state. Component 5 cannot wire `AmendDeps.LoadChunkIDs: buildChunkIDSet` until Component 2 is done.

6. **`applyFeedbackAmend` mirrors `applyFactAmend`:** Not shown in full above to avoid duplication, but it must implement the same pattern: parse `feedbackFrontmatterDoc`, check `Behavior`/`Impact`/`Action` fields against `args`, merge `Sources` via `mergeChunkSources`, populate `feedbackFields`, call `renderFeedbackFrontmatter`. The `Issue` field round-trip is identical: `Issue: string(doc.Issue)` → `quotedString(f.Issue)` in `renderFeedbackFrontmatter`.

### Component 6: `/recall` skill — agent-judged coverage + recency-weighted distillation (D6/D7)

**Files:** `skills/recall/SKILL.md` (rewrite Step 2.5 only; all other steps unchanged)

**Dependency note:** This component is executable only after Components 2, 3, and 5 are merged, because Step 2.5 references:
- `candidate_l2s: [{path, cosine}]` per cluster (Component 3's top-K field, replacing `nearest_l2`)
- `engram amend` command (Component 5)
- `engram show <ref>` to read candidates before judging (already ships)
- `engram activate` for the covered path (already ships)

**Component-entry verification:** Before starting, confirm Components 2, 3, and 5 are merged:
- `grep IngestedAt internal/chunk/index.go` — must show `time.Time` field
- `grep candidate_l2s internal/cli/query.go` — must show the top-K field
- `grep RunAmend internal/cli/amend.go` — must exist
- `targ check-full` — all checks pass

---

#### Task 6.1a — Write the baseline-capture scenario

- [ ] Read the current `skills/recall/SKILL.md` lines 95–170 (the current cosine-band Step 2.5). Note exactly what the table-driven band gate says so the baseline prompt can reproduce the behavior an agent shows today.

- [ ] Create the scratch file `dev/skill-tdd/recall-step2.5-baseline-prompt.md` (do **not** commit). The scenario gives a subagent: (a) the current full `SKILL.md` text verbatim, (b) a fabricated `engram query` payload using the **post-Component-3 schema** (`candidate_l2s`, not `nearest_l2` — the binary will emit `candidate_l2s` once Component 3 is merged, so the baseline must use the schema the binary actually emits), and (c) the instruction "Execute Step 2.5 exactly as written."

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
      candidate_l2s:
        - {path: "1a.tdd-must-come-first.md", cosine: 0.96}
        - {path: "1b.test-doubles-patterns.md", cosine: 0.84}
        - {path: "1c.integration-test-scope.md", cosine: 0.79}
    - phrase: "chunks"
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
    - phrase: "chunks"
      size: 2
      silhouette: 0.31
      members:
        - {path: "chunks/t8.md", score: 0.79, is_representative: true}
        - {path: "chunks/t9.md", score: 0.72, is_representative: false}
      candidate_l2s:
        - {path: "1c.integration-test-scope.md", cosine: 0.72}
        - {path: "1e.scope-boundaries.md", cosine: 0.65}
        - {path: "1f.test-granularity.md", cosine: 0.60}
  budget:
    items_returned: 0
    notes_scanned: 12
  ```

  Execute Step 2.5 now.
  ```

  **Why `candidate_l2s` in the baseline, not `nearest_l2`?** The pre-v2 skill text references `nearest_l2`; however, the binary (post-Component-3) now emits `candidate_l2s`. The baseline payload must use the schema the binary actually emits so the RED state captures real agent behavior with real output — not behavior against a phantom field. The pre-v2 skill's band table (`nearest_l2.cosine >= 0.95`, etc.) will cause the agent to look for `nearest_l2` in a payload that has `candidate_l2s`, surfacing the schema mismatch as part of the documented RED failure.

---

#### Task 6.1b — Run baseline scenario + capture transcript

- [ ] Spawn a subagent via `TaskCreate` with the scenario file content from Task 6.1a. Spec for the subagent invocation:
  - **Input:** the full content of `dev/skill-tdd/recall-step2.5-baseline-prompt.md` (expand `[PASTE CURRENT SKILL.MD HERE]` with the literal content of `skills/recall/SKILL.md` at this moment)
  - **Instruction to subagent:** "Execute Step 2.5 of the /recall skill EXACTLY as written in the pasted skill text, using the fabricated payload below. Stop after Step 2.5."
  - **Expected runtime:** ~2 minutes
  - **Cost bound:** $0.50 (three clusters, no real tool calls — the subagent is describing what it would do, not executing live `engram` commands)
  - **Output:** save the full subagent transcript verbatim to `dev/skill-tdd/baseline-transcript.md` (do not commit)

- [ ] Record verbatim what the agent does for each cluster. Expected RED behaviors to observe:
  - Cluster 1 (top cosine 0.96): agent looks for `nearest_l2` field — may get confused or fall back to `candidate_l2s[0]`; applies the `>= 0.95` band and calls `engram activate --note 1a.tdd-must-come-first.md`. Does NOT read candidates first.
  - Cluster 2 (top cosine 0.87): agent applies the `0.80–0.95` band and calls `engram learn fact|feedback --target <luhmann-id> --position continuation ...` (UPDATE band). Does NOT call `engram amend`.
  - Cluster 3 (top cosine 0.72): agent applies the `< 0.80` band and calls `engram learn fact|feedback --position top ...` (CREATE band).
  - In all cases: the agent reads no candidate content before deciding, uses no `engram amend`, consults no recency reasoning.

---

#### Task 6.1c — Extract RED baseline checklist

- [ ] From the transcript in `dev/skill-tdd/baseline-transcript.md`, create `dev/skill-tdd/baseline-red-checklist.md` (do not commit) listing the observed violations. The checklist must have one entry per violation with a transcript line reference. Minimum expected violations (these define what GREEN must fix):

  ```markdown
  # RED baseline violations — recall Step 2.5 (pre-v2)

  ## Violation 1: No candidate reading before coverage judgment
  - **Expected (v2):** Agent calls `engram show <candidate>` before deciding covered/near/absent.
  - **Observed (pre-v2):** Agent applied cosine threshold directly from `candidate_l2s[0].cosine`
    (or `nearest_l2.cosine`) without reading candidate content.
  - **Transcript reference:** [line X]

  ## Violation 2: Cosine threshold as sole gate
  - **Expected (v2):** Coverage is agent-judged from content; cosine only nominates.
  - **Observed (pre-v2):** Agent used `>= 0.95` / `0.80–0.95` / `< 0.80` table directly.
  - **Transcript reference:** [line Y]

  ## Violation 3: `engram learn --target` used for updates (not `engram amend`)
  - **Expected (v2):** Updates use `engram amend <path> ...`.
  - **Observed (pre-v2):** Agent called `engram learn feedback|fact --target <id> --position continuation`.
  - **Transcript reference:** [line Z]

  ## Violation 4: No recency reasoning applied
  - **Expected (v2):** Agent identifies conflicts between older/newer members and applies recency weight.
  - **Observed (pre-v2):** Agent treated all member evidence as equally weighted; no recency step.
  - **Transcript reference:** [line W]
  ```

  **This checklist is the RED state — it defines exactly what GREEN (Task 6.2 + Task 6.3) must change.**

---

#### Task 6.2 — GREEN: rewrite Step 2.5 in `skills/recall/SKILL.md`

This is a targeted rewrite of lines 95–170 only. All other steps (0, 0.5, 1, 2, 3, the Red flags table) remain unchanged except the Red flags rows that are updated below.

- [ ] Open `skills/recall/SKILL.md`. Identify the exact text block for Step 2.5 (currently lines 95–169, from `### Step 2.5` through the last Red flags row that references the cosine band logic). Do **not** touch any other section.

- [ ] Replace the entire Step 2.5 section (the `### Step 2.5 — Crystallize lessons from the payload's chunk clusters (band-driven)` heading through the end of its content, stopping before `### Step 3`) with the following text:

  ```markdown
  ### Step 2.5 — Lazy L2 synthesis from the clustering (agent-judged)

  The `--synthesize-l2` output's `clusters` list contains the unified clustering of matched chunks
  and notes. Each cluster carries `candidate_l2s: [{path, cosine}]` — the top-K existing L2s
  nearest the cluster centroid (K ≥ 3, centroid cosine). **Process every cluster.** For each:

  **A. Read candidates and members**

  Run `engram show <path>` on every entry in `candidate_l2s` (up to K calls, blocking). For
  note-kind cluster members already in the payload's `items[]` list, use their `content` field
  directly — no additional `engram show` call needed on already-surfaced members. For chunk
  members not in `items[]`, use the chunk content from the cluster's `members` list. Do not
  judge coverage before you have read the candidate content.

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
  | You called `engram show` on a note already in `items[]` | Members already in `items[]` carry a `content` field — use it directly; `engram show` is only for candidates not in `items[]` |
  ```

- [ ] Also update the existing Red flags row:
  - OLD: `| You grouped chunks by eye instead of using the payload's 'phrase: "chunks"' clusters | The binary's k-means grouping and 'nearest_l2' cosine are the ground truth; apply the bands |`
  - NEW: `| You grouped chunks by eye instead of using the payload's clusters | The binary's k-means grouping is the ground truth; read every cluster |`

  And remove the rows that reference "banded N clusters and wrote 0 notes" and "≥0.95 cluster → you created a new L2" since those rows are band-specific logic that no longer applies. Also remove `` `≥0.95` clusters where the covering L2 was useful`` from the `activated: true` red flag row.

---

#### Task 6.3 — GREEN verify: run the same scenario WITH the new skill

- [ ] Rerun the pressure scenario from Task 6.1a as a subagent, this time with the **new** SKILL.md text injected. The payload already uses `candidate_l2s` (from Task 6.1a's baseline scenario):

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

  Note: `1a.tdd-must-come-first.md` appears in both `members` (as a note-kind item with `is_representative: true`) and in `candidate_l2s`. The agent must use the `content` field from `items[]` for the note-kind member (it is already surfaced), and run `engram show` only on candidates NOT already in `items[]` (i.e., `1b.test-doubles-patterns.md` and `1c.integration-test-scope.md` for cluster 1).

  Pass the scenario to a subagent with the instruction: "Execute Step 2.5 of the /recall skill exactly as written."

- [ ] **Pass criteria (check each):**
  - [ ] Agent calls `engram show` on each `candidate_l2s` entry that is NOT already in `items[]` before deciding.
  - [ ] Agent uses the `content` field from `items[]` for note-kind members already surfaced — does NOT call `engram show` redundantly on them.
  - [ ] Agent applies recency reasoning before deciding covered/near/absent.
  - [ ] Agent uses `engram amend --activate` (not `engram activate`) for covered clusters.
  - [ ] Agent uses `engram amend` with content flags for near clusters.
  - [ ] Agent uses `engram learn` for absent clusters.
  - [ ] Agent passes `--relation` for note sources and `--chunk-source` for chunk sources.
  - [ ] Agent writes exactly one note per cluster, never two.
  - [ ] Agent does NOT use the cosine value as a threshold gate.
  - [ ] Check against the RED baseline checklist in `dev/skill-tdd/baseline-red-checklist.md` — every violation listed there must be absent from the GREEN transcript.

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

- [ ] **Transient window note:** After this task completes, the installed recall skill references `candidate_l2s` and `engram amend` — both of which exist (Components 3 and 5 are merged). However, `docs/architecture/c1-system-context.md` and `skills/learn/SKILL.md` still contain the stale cosine-band and "prunes stale chunks" text until Component 7 runs. This is a transient inconsistency window: the docs are descriptive artifacts, not executable, so the installed skill works correctly. Component 7 closes the window. Do not skip Component 7 assuming this `engram update` completed the job — the executor needs Component 7 to reconcile the descriptive layer.

---

**Risks / notes for the synthesizer:**

1. **Hard dependency on Components 3 and 5.** The new Step 2.5 names `candidate_l2s` (Component 3's renamed field) and `engram amend` (Component 5's new subcommand). This component cannot be executed until both are merged and the binary is rebuilt — plan sequencing gates accordingly.

2. **The Red flags table surgery is load-bearing.** Several existing rows mix band-gate language (`≥0.95`, `0.80–0.95`, `<0.80`) with currently-correct behavior. The plan above identifies the rows to remove, but the executor must diff carefully: removing a row that guards a still-valid behavior (e.g., the `engram activate` batch-call row at the end of Step 2) would be a regression. Only Step 2.5-specific band rows should be deleted; the Step 2 `activated: true` batch-call row is unaffected.

3. **The `engram show` calls in Step 2.5 are bounded by K candidates, not by K candidates plus all note-kind members.** The spec §3.5 records recall latency as a headline experiment metric. Step 2.5 as written caps `engram show` at K calls per cluster (candidates only). Members already in `items[]` use their `content` field directly. Do not silently add a cap to the skill text beyond what is already specified; do not add redundant `engram show` calls for already-surfaced members.

### Component 7: Reconcile c1 + learn SKILL.md

**Files:** `docs/architecture/c1-system-context.md`, `skills/learn/SKILL.md`

> **Scope note:** `skills/recall/SKILL.md` (Step 2.5 rewrite) is handled in Component 6, per spec §7 step 7's parenthetical: "this is gate 6's skill work" — Component 7 covers only `c1-system-context.md` and `skills/learn/SKILL.md`. The split is intentional. After Component 6's `engram update` (Task 6.8), the installed recall skill is ahead of the c1 doc and learn SKILL.md; Component 7 closes this window. The inconsistency is transient and harmless (the docs are descriptive artifacts, not executable), but is called out here so the executor does not skip Component 7 thinking the update completed the job.

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

**REQUIRED pre-edit step:** Run the grep below to find all `§6b` or `L3 ADR` references across the entire `docs/` tree. This must complete before making any edits. If matches appear outside `c1-system-context.md`, read those files and add companion edit tasks (or block this component until a decision is made) — do not proceed with only the c1 edit while leaving dangling cross-references.

```bash
grep -rn "§6b\|L3 ADR" /Users/joe/repos/personal/engram/docs/
```

Expected ideal: matches only in `c1-system-context.md`. If matches appear in `c2-containers.md`, `c3-components.md`, `adr.md`, or other docs files, those files require companion edits. Create a follow-up task (or inline the edits here) for each file with matches outside c1, covering: (a) sequence diagrams referencing the §6b loop; (b) the §6b update-or-create decision flowchart; (c) prose referencing L3 ADR synthesis at learn time. Do not skip or defer these — a grep match that is not addressed leaves a dangling cross-reference that contradicts the updated c1.

After confirming (or addressing) all matches, apply the c1 edit:

Edit `docs/architecture/c1-system-context.md` — in the file, locate the engagement flowchart (around lines 319–335). Replace the `F → H → I → J → K` learn branch:

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

Check that all stale phrases are gone across all three files that Component 7 touches (c1, learn SKILL.md) plus the recall SKILL.md that Component 6 edits (guarding against any missed residual from Task 6.x):

```bash
grep -n "nearest_l2\|prunes stale chunks\|fire-and-forget\|Synthesis subagent\|transcript --mark\|learn episode\|L3 ADR\|three bands" \
  /Users/joe/repos/personal/engram/docs/architecture/c1-system-context.md \
  /Users/joe/repos/personal/engram/skills/learn/SKILL.md \
  /Users/joe/repos/personal/engram/skills/recall/SKILL.md
```

Expected output: empty (no matches). If any match appears, read the surrounding context and apply a follow-up Edit before declaring done.

> **Note:** `skills/recall/SKILL.md` is included here as a cross-check only — its content was rewritten by Component 6. Component 7 does not edit it; this grep is a residual-validation guard, not a Component 7 responsibility.

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
3. **L1 decision flowchart node removal (Task 7.6):** the `J{convention recurs across episodes?}` branch and `K[§6b: synthesize or update an L3 ADR]` node have cross-references in `docs/architecture/c2-containers.md` (§6b sequence diagram at line 138, flowchart at line 195), `docs/architecture/c3-components.md` (line 147), and `docs/architecture/adr.md` (lines 32, 86, 99, 106, 113). The required grep in Task 7.6 will surface these; companion edits are required for any match outside c1.

