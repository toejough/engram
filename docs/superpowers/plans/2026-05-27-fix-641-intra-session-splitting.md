# Fix #641 — Intra-Session Splitting for `engram transcript --mark`

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans (or inline). Steps use checkbox (`- [ ]`) syntax.

**Goal:** `engram transcript --mark` advances the marker reliably even when a single Claude or OpenCode session is larger than `--max-bytes`. Each run emits as much as fits, intra-session, and the marker advances to the timestamp of the last row included so the next run resumes mid-session.

**Architecture:** Two related fixes folded into one change:

1. **Filter bug** (compounding cause). `filterBySourceMarkers` uses `!entry.Mtime.Before(from)` (≥) when checking whether a session entry is newer than the marker. With marker = Mtime of last fully-included session, the boundary session is re-included every run — so the byte cap halts at the same boundary forever. Flip to strict-greater (`entry.Mtime.After(from)`).
2. **Intra-session splitting** (the user-requested feature). Today `emitTranscripts` reads the whole session via the `Reader` (callers pass `math.MaxInt32` as budget) and decides at the session level whether to include or exclude. A single session larger than `--max-bytes` is "always included" (first-entry progress guarantee) but the marker can't represent "we emitted up to row X." Refactor the Reader to take a `fromTime` filter and a real `budgetBytes`, and return both the emitted content and the `LastTimestamp` of the last row included. `emitTranscripts` advances the marker to the per-source `LastTimestamp` when a partial read happens, and to the session's `Mtime` when a full file is consumed.

**The cheap alternative — explicitly NOT taken.** Flipping the filter alone (1) would unstick the reported repro loop (run 2's filter excludes session-1, session-2 becomes first → first-entry guarantee includes it whole, marker jumps to session-2's Mtime). But the user explicitly said "We need to be able to split session log reads." Splitting is a feature, not an incidental requirement — without (2), a single session larger than the cap still emits whole on every run (because of first-entry-always-included), spending more budget than the user allocated. (2) is what the user asked for; (1) alone is a half-fix.

**Tech Stack:** Pure Go; `imptest`/`rapid`/`gomega` tests; `targ` for build/test/check; `modernc.org/sqlite` (pure-Go) for OpenCode. JSONL parsing via `encoding/json` line-by-line.

---

## File Structure

**Core changes:**

- `internal/transcript/transcript.go` — `Reader` interface signature; `JSONLReader.Read` becomes row-aware, emits forward from `fromTime`, returns `ReadResult`.
- `internal/transcript/opencode.go` — `OpencodeTranscriptReader.Read` becomes row-aware; SQL adds `time_created` to the SELECT and threads it through `queryParts` → strip → emit.
- `internal/cli/transcript.go` — `emitTranscripts` calls reader with the per-source marker + remaining budget; advances `lastIncluded[src]` to `LastTimestamp` (intra-session) when `Partial`, else to `entry.Mtime` (full file); filter switches to strict-greater.
- `internal/transcript/range.go` — confirm whether `RangeReader.ReadRange` (used by `engram learn episode`) shares any code path that needs to update. (Suspect no — that one already operates on explicit start/end.) Touch if and only if it shares a helper.

**Tests:**

- `internal/transcript/transcript_test.go` — new tests: row-filter, partial read, last-timestamp accuracy, null-timestamp handling.
- `internal/transcript/opencode_test.go` — same, plus `time_created` SQL contract.
- `internal/cli/transcript_test.go` (or wherever the existing `emitTranscripts` tests live) — new tests: marker advance to intra-session timestamp; strict-greater filter; continuation message after partial read.
- `internal/cli/cli_test.go` — end-to-end: write fixture with one oversized session, run twice, assert each run advances the marker past previously-emitted rows.

**Docs:**

- `README.md` — section "Transcript progress tracking" — update the byte-cap-continuation paragraph.
- `skills/learn/SKILL.md` — byte-cap-continuation paragraph in §1.
- `docs/architecture/c1-system-context.md` if the transcript sequence diagram names file-level granularity.

---

## Design notes

**Marker semantics — unchanged on disk.** The marker file still stores a single RFC3339 timestamp. The *interpretation* widens: that timestamp is the upper bound of "what has been emitted so far for this source." For full-file reads, that's the file's Mtime (same as today). For partial reads, that's the timestamp of the last row included. No schema migration; old markers continue to work because file Mtimes are always ≥ any row timestamp inside the file, so the strict-greater filter still picks the right files.

**Null/missing row timestamps.** Claude JSONL has at least one row with `timestamp: null` (`file-history-snapshot` at the top of every file). Rule: when a row's timestamp is null/missing, **inherit the previous row's timestamp**; if there is no previous row (the first row in the file), use **zero time** as the row's effective timestamp. Behavioral implication: null-timestamp leading rows are always emitted (they pass any `fromTime` filter) and contribute to `bytesUsed`; `LastTimestamp` advances only when a row with a non-null timestamp is emitted. (Null-only files are pathological — would behave as if no progress made, just like an empty file.)

**First-row progress guarantee.** Today: "first session always included regardless of size." After: "at least one row of the first session is always emitted, advancing the marker even if that row alone exceeds the budget." This avoids the degenerate case where row 1 happens to be larger than `--max-bytes` and the scan would otherwise loop forever.

**Strip + budget interaction.** `sessionctx.StripWithConfig` already operates on `[]string` lines and is purely line-level. After stripping, each surviving line is still parseable JSON, so the readers can extract the `timestamp` field from output lines directly — no need to thread parallel timestamp arrays through `StripWithConfig`.

**Edge cases considered:**

- *Partially-read session deleted/truncated before next run.* The session will not appear in `Finder.Find` results next time, so the scan simply moves on. No special handling needed; the marker remains at the row timestamp from the partial read, future sessions are filtered by strict-greater (`Mtime > marker`).
- *Marker exactly equals file Mtime, file has no new rows.* Strict-greater filter excludes the file. Correct — nothing new to emit.
- *Reader budget so tight that nothing fits.* The first session's first row is always emitted (guarantee). `LastTimestamp` reflects that row; subsequent runs make further progress.

---

## Task 1: Define `ReadResult` and the new Reader signature

**Files:**
- Modify: `internal/transcript/transcript.go` (struct + interface)

- [ ] **Step 1: Write a failing test** — pure-data, no I/O. Add to `internal/transcript/transcript_test.go`:

```go
func TestReadResult_ZeroValueIsExplicit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	var r transcript.ReadResult
	g.Expect(r.Content).To(BeEmpty())
	g.Expect(r.BytesUsed).To(Equal(0))
	g.Expect(r.LastTimestamp.IsZero()).To(BeTrue())
	g.Expect(r.Partial).To(BeFalse())
}
```

This is a smoke test that the struct exists with the documented zero-values.

- [ ] **Step 2: Run — verify FAIL** (compile error: undefined `transcript.ReadResult`).

```bash
targ test
```

- [ ] **Step 3: Add to `internal/transcript/transcript.go`**:

```go
// ReadResult bundles a partial session read's output with the
// information emitTranscripts needs to advance the per-source marker.
// LastTimestamp is the timestamp of the last row included in Content
// (zero when Content is empty). Partial reports whether budgetBytes
// halted the read before the file was exhausted.
type ReadResult struct {
	Content       string
	BytesUsed     int
	LastTimestamp time.Time
	Partial       bool
}
```

- [ ] **Step 4: Verify** `targ test` recompiles.

- [ ] **Step 5: Don't change `Reader` interface yet** — Tasks 2-3 add the new method alongside the old one to keep tests green incrementally.

---

## Task 2: `JSONLReader.ReadFrom` — row-aware Claude read

**Files:**
- Modify: `internal/transcript/transcript.go`
- Modify: `internal/transcript/transcript_test.go`

- [ ] **Step 1: Write failing tests** for the new behavior. Three cases:

```go
func TestJSONLReader_ReadFrom_EmitsRowsAfterMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Fixture: 3 user rows at t0, t1, t2 (chronological).
	// Marker = t1. Expect only the t2 row in Content; LastTimestamp = t2;
	// Partial = false (budget plenty).
	content := strings.Join([]string{
		`{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":[{"type":"text","text":"a"}]}}`,
		`{"type":"user","timestamp":"2026-01-01T00:01:00Z","message":{"role":"user","content":[{"type":"text","text":"b"}]}}`,
		`{"type":"user","timestamp":"2026-01-01T00:02:00Z","message":{"role":"user","content":[{"type":"text","text":"c"}]}}`,
	}, "\n") + "\n"

	r := transcript.NewJSONLReader(&stubFileReader{data: []byte(content)})
	from, _ := time.Parse(time.RFC3339, "2026-01-01T00:01:00Z")
	result, err := r.ReadFrom("/fake.jsonl", from, 1<<20)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Content).To(ContainSubstring("\"text\":\"c\""))
	g.Expect(result.Content).NotTo(ContainSubstring("\"text\":\"a\""))
	g.Expect(result.Content).NotTo(ContainSubstring("\"text\":\"b\""))
	g.Expect(result.LastTimestamp.Equal(must(time.Parse(time.RFC3339, "2026-01-01T00:02:00Z")))).To(BeTrue())
	g.Expect(result.Partial).To(BeFalse())
}

func TestJSONLReader_ReadFrom_StopsAtBudget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Build a fixture with 3 large user rows (~100 bytes each).
	// Budget = 150 bytes. Expect 1 or 2 rows in Content (depending on stripping),
	// Partial=true, LastTimestamp = timestamp of the last emitted row.
	// Specifics in implementation; the contract: Partial=true and LastTimestamp
	// equals the included row's timestamp, not the next-excluded row's timestamp.
	// ... (test body)
}

func TestJSONLReader_ReadFrom_NullTimestampRowsInheritPrior(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Fixture: row 1 has timestamp null (e.g. file-history-snapshot),
	// row 2 has timestamp t1.
	// Marker = zero time. Expect both rows emitted; LastTimestamp = t1.
	// Marker = t1. Expect only row 2 NOT emitted (its timestamp equals marker);
	// row 1 (null) is excluded too because its effective ts is zero (< marker).
	// ... (test body)
}
```

- [ ] **Step 2: Run — verify FAIL** (compile error: undefined `ReadFrom`).

- [ ] **Step 3: Add the method** alongside the existing `Read`:

```go
// ReadFrom reads the JSONL transcript at path, filters to rows whose
// per-row timestamp is strictly after fromTime, strips noise, and
// returns a ReadResult chronologically. The first row (always) is
// emitted regardless of fromTime IF the caller passed the zero time;
// for non-zero fromTime, only rows with timestamp > fromTime are
// considered. budgetBytes caps total Content size; when the budget
// would be exceeded, the read halts and Partial is set to true. The
// first surviving row is always emitted even if it alone exceeds
// budgetBytes (progress guarantee).
//
// Rows with null/missing timestamps inherit the previous row's
// timestamp; rows preceding any timestamped row inherit zero time
// (so they are excluded when fromTime is non-zero and included when
// fromTime is zero).
func (r *JSONLReader) ReadFrom(path string, fromTime time.Time, budgetBytes int) (ReadResult, error) {
	raw, err := r.reader.Read(path)
	if err != nil {
		return ReadResult{}, fmt.Errorf("reading transcript: %w", err)
	}

	lines := splitNonEmpty(strings.Split(string(raw), "\n"))
	timestamps := extractTimestamps(lines) // []time.Time, len == len(lines)

	// Filter to rows strictly after fromTime.
	kept := make([]string, 0, len(lines))
	keptTimes := make([]time.Time, 0, len(lines))
	for i, line := range lines {
		if !fromTime.IsZero() && !timestamps[i].After(fromTime) {
			continue
		}
		kept = append(kept, line)
		keptTimes = append(keptTimes, timestamps[i])
	}

	stripped := sessionctx.StripWithConfig(kept, sessionctx.StripConfig{ToolSummaryMode: true})
	// stripped lines are still JSON-parseable; re-extract timestamps from them.
	strippedTimes := extractTimestamps(stripped)

	// Emit chronologically, budget-bounded. First survivor always included.
	var builder strings.Builder
	bytesUsed := 0
	lastTs := time.Time{}
	partial := false
	for i, line := range stripped {
		lineLen := len(line) + 1
		if bytesUsed > 0 && bytesUsed+lineLen > budgetBytes {
			partial = true
			break
		}
		builder.WriteString(line)
		builder.WriteByte('\n')
		bytesUsed += lineLen
		if !strippedTimes[i].IsZero() {
			lastTs = strippedTimes[i]
		}
	}

	return ReadResult{
		Content:       builder.String(),
		BytesUsed:     bytesUsed,
		LastTimestamp: lastTs,
		Partial:       partial,
	}, nil
}
```

Add the helpers:

```go
// splitNonEmpty splits content lines and drops empty entries.
func splitNonEmpty(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// extractTimestamps parses each line as JSON and returns the
// per-row timestamp. Rows with null/missing/unparseable timestamps
// inherit the previous row's timestamp; the first row inherits zero.
func extractTimestamps(lines []string) []time.Time {
	out := make([]time.Time, len(lines))
	var carry time.Time
	for i, line := range lines {
		var probe struct {
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal([]byte(line), &probe); err == nil && probe.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339Nano, probe.Timestamp); err == nil {
				carry = t
			}
		}
		out[i] = carry
	}
	return out
}
```

(Add `encoding/json` to the imports.)

- [ ] **Step 4: Run — verify PASS**.

- [ ] **Step 5: Don't delete the old `Read` yet** — Task 4 retires it after `emitTranscripts` is migrated.

---

## Task 3: `OpencodeTranscriptReader.ReadFrom`

**Files:**
- Modify: `internal/transcript/opencode.go`
- Modify: `internal/transcript/opencode_test.go`

- [ ] **Step 1: Write failing test** with a fixture SQLite DB containing 3 parts with `time_created` t0, t1, t2. Marker = t1. Expect only the t2 part in Content; `LastTimestamp` = t2.

- [ ] **Step 2: Add the method** alongside the existing `Read`:

```go
// ReadFrom queries parts whose time_created is strictly after fromTime,
// strips noise, accumulates chronologically until budgetBytes is hit.
// Same Partial / LastTimestamp semantics as JSONLReader.ReadFrom.
func (r *OpencodeTranscriptReader) ReadFrom(path string, fromTime time.Time, budgetBytes int) (ReadResult, error) {
	sessionID := strings.TrimPrefix(path, "opencode://")
	if sessionID == "" {
		return ReadResult{}, ErrEmptySessionID
	}

	rows, err := r.queryPartsAfter(sessionID, fromTime)
	if err != nil {
		return ReadResult{}, err
	}

	// rows is []opencodeRow{ line string; timeCreated time.Time }
	lines := make([]string, len(rows))
	times := make([]time.Time, len(rows))
	for i, row := range rows {
		lines[i] = row.line
		times[i] = row.timeCreated
	}

	stripped := sessionctx.StripWithConfig(lines, sessionctx.StripConfig{ToolSummaryMode: true})
	// Map stripped lines back to row times. StripWithConfig may drop or merge
	// lines; we conservatively re-extract timestamps from the stripped output
	// by parsing JSON (parts that survive stripping still carry the original
	// JSON shape we built in buildJSONLLine). If a stripped line cannot be
	// re-attributed, carry the previous row's time.
	// ... (implementation detail)

	// Emit chronologically with first-row progress guarantee.
	// (same shape as JSONLReader.ReadFrom)
}

func (r *OpencodeTranscriptReader) queryPartsAfter(sessionID string, fromTime time.Time) ([]opencodeRow, error) {
	db, err := sql.Open("sqlite", r.dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening opencode database: %w", err)
	}
	defer db.Close()

	// SELECT now includes p.time_created. Filter is part of the WHERE.
	rows, err := db.QueryContext(
		context.Background(),
		"SELECT p.time_created, json_extract(p.data, '$.type'), json_extract(p.data, '$.text'), "+
			"json_extract(p.data, '$.tool'), json_extract(p.data, '$.state'), "+
			"json_extract(m.data, '$.role') "+
			"FROM part p LEFT JOIN message m ON p.message_id = m.id "+
			"WHERE p.session_id = ? AND p.time_created > ? "+
			"ORDER BY p.time_created",
		sessionID,
		fromTime.UTC().Format(time.RFC3339Nano),
	)
	// ... scan rows into []opencodeRow
}
```

(Note: time_created may be stored as an INTEGER unix-ms or a TEXT RFC3339 in OpenCode's DB. Verify against a real DB if available; fall back to filtering in-Go if the WHERE clause produces nothing because the type is incompatible. Spike with `sqlite3 ~/.local/share/opencode/storage/session/db.sqlite3 'PRAGMA table_info(part)'` to confirm.)

- [ ] **Step 3: Run — verify FAIL then PASS** after implementation.

- [ ] **Step 4: SELECT change without breaking existing `Read`** — keep the old `queryParts` SELECT unchanged; the new method has its own `queryPartsAfter`.

---

## Task 4: Migrate `emitTranscripts` to `ReadFrom`

**Files:**
- Modify: `internal/cli/transcript.go` — `emitTranscripts`, `filterBySourceMarkers`.
- Modify: existing `emitTranscripts` tests.

- [ ] **Step 1: Write failing test** for the new emit semantics:

```go
func TestEmitTranscripts_PartialEmitAdvancesMarkerToRowTimestamp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Stub Reader.ReadFrom returns Partial=true with LastTimestamp=t1
	// for the only entry.
	// Expect emitResult.lastIncluded[src] == t1 (not the file's Mtime).
	// Expect emitResult.firstUnincluded[src] is empty (because we partial-
	// emitted, not excluded; the continuation message is computed separately).
}

func TestEmitTranscripts_FullEmitKeepsMarkerAtFileMtime(t *testing.T) {
	// Reader returns Partial=false. lastIncluded[src] = entry.Mtime as before.
}

func TestFilterBySourceMarkers_StrictGreaterExcludesBoundary(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Fixture: marker=T, entries with Mtime in {T-1, T, T+1}.
	// Expect only the T+1 entry to remain.
}
```

- [ ] **Step 2: Implement**.

In `filterBySourceMarkers`, swap:

```go
if !entry.Mtime.Before(from) && !entry.Mtime.After(now) {
```
for:
```go
if entry.Mtime.After(from) && !entry.Mtime.After(now) {
```

In `emitTranscripts`, switch the inner loop:

```go
for index, entry := range entries {
	remaining := maxBytes - total
	if remaining <= 0 && index > 0 {
		// Budget exhausted by prior entries; bail.
		recordUnincluded(result.firstUnincluded, entries[index:])
		break
	}

	fromTime := result.lastIncluded[entry.Source]
	// On the first entry of this source in this run, fromTime is zero —
	// we want everything from this file forward. Reader treats zero as
	// "no filter" so this is correct.

	readResult, err := reader.ReadFrom(entry.Path, fromTime, max(remaining, 1))
	if err != nil { return emitResult{}, fmt.Errorf(...) }

	_, err = io.WriteString(stdout, readResult.Content)
	if err != nil { return emitResult{}, fmt.Errorf(...) }

	total += readResult.BytesUsed
	if !readResult.LastTimestamp.IsZero() {
		result.lastIncluded[entry.Source] = readResult.LastTimestamp
	}
	if !readResult.Partial && !entry.Mtime.IsZero() {
		// Whole file consumed — marker advances to file Mtime (covers
		// the case where the file had untimestamped trailing rows).
		result.lastIncluded[entry.Source] = entry.Mtime
	}
	result.hadEntries[entry.Source] = true

	if readResult.Partial {
		recordUnincluded(result.firstUnincluded, entries[index:])
		break
	}
}
```

(Note: the new `Reader.ReadFrom` replaces the old `Read`. Update the `transcript.Reader` interface signature accordingly; remove the old `Read` once tests pass.)

- [ ] **Step 3: Update the `transcript.Reader` interface**:

```go
type Reader interface {
	ReadFrom(path string, fromTime time.Time, budgetBytes int) (ReadResult, error)
}
```

- [ ] **Step 4: Remove the old `Read` methods** from `JSONLReader` and `OpencodeTranscriptReader` once nothing references them. The `RangeReader` in `range.go` is a separate interface for episode body-reading — leave alone.

- [ ] **Step 5: Update existing emitTranscripts tests** that constructed the old-shape stub Reader.

- [ ] **Step 6: Run `targ check-full`** until all green.

- [ ] **Step 7: Commit**:

```bash
git add internal/transcript/ internal/cli/transcript.go internal/cli/*_test.go
git commit -m "$(cat <<'EOF'
feat(transcript): intra-session splitting + strict-greater marker filter (#641)

When a single session exceeds --max-bytes, the reader now emits as much
as fits and returns the timestamp of the last row included. The marker
advances to that row timestamp, so the next run resumes mid-session
from rows newer than it.

Two compounding causes addressed:
  (1) filterBySourceMarkers used a >= comparison on the marker boundary,
      re-including the last fully-emitted session every run. Now uses
      strict-greater so the boundary session does not loop.
  (2) The Reader interface was file-level (math.MaxInt32 budget); a
      session larger than the cap had no way to advance the marker to
      "we got through row X." New ReadResult bundles content, bytes
      used, last-row timestamp, and partial flag. Per-row timestamp
      extraction handles null timestamps (inherit prior row).

Marker file format unchanged — same RFC3339 string; only the
interpretation widens (file Mtime OR row timestamp inside a file,
depending on whether the read was partial). Old markers continue to
work because file Mtimes are >= any row timestamp inside the file.

AI-Used: [claude]
EOF
)"
```

---

## Task 5: End-to-end smoke test

**Files:**
- Modify: `internal/cli/cli_test.go` (or wherever existing E2E tests live).

- [ ] **Step 1: Add a fixture-based test**:

```go
func TestEngramTranscriptMark_OversizedSessionSplits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Build a temp transcript dir with one .jsonl session whose stripped
	// content is ~500 KB (well above --max-bytes 200000).
	// Run engram transcript --mark with default max-bytes.
	// Assert: stdout includes "marker advanced to <T>" with T < file Mtime.
	// Run again. Assert: marker advanced further (T2 > T1).
	// Repeat until "byte cap hit" message stops appearing.
}
```

- [ ] **Step 2: Run, verify PASS** after the implementation lands.

- [ ] **Step 3: Real-world smoke**:

```bash
targ build || go build -o /tmp/engram-641 ./cmd/engram
/tmp/engram-641 transcript --mark 2>&1 | tail -3
# Run a second time:
/tmp/engram-641 transcript --mark 2>&1 | tail -3
# Expect: the second run's marker timestamp is strictly later than the first's,
# even if the byte-cap-hit warning still appears (because the in-flight session
# is still large).
```

---

## Task 6: Docs

- [ ] **Step 1: Update `README.md`** — under "Transcript progress tracking", replace the byte-cap-continuation paragraph to describe intra-session splitting:

```markdown
**Byte-cap continuation.** Each scan stops at `--max-bytes` (default
200000). When the cap halts a scan partway through a session, the
session is partially emitted and the marker advances to the timestamp
of the last row included. The tail line names the cap boundary:
`[engram transcript: byte cap hit; <source> sessions from <date> onward
not yet scanned; run again to continue]`. Run `/learn` again (after
`/clear` if context is tight) to catch up — each subsequent run picks
up where the prior one left off, even within a single oversized session.
```

- [ ] **Step 2: Update `skills/learn/SKILL.md`** byte-cap-continuation paragraph (§1) similarly. (Per project rule, SKILL edits go through `superpowers:writing-skills` — but this is a clarification of factual behavior, not a behavioral rule change. Apply writing-skills if the agent feels it warrants TDD; otherwise edit directly. Use judgement.)

- [ ] **Step 3: Check `docs/architecture/c1-system-context.md`** transcript sequence diagram — update if it depicts file-level granularity.

- [ ] **Step 4: Commit**.

---

## Task 7: Close issue and remove plan doc

- [ ] **Step 1: Close #641**:

```bash
gh issue close 641 -c "Fixed by intra-session splitting + strict-greater marker filter.
Reader interface now takes fromTime + budgetBytes and returns
ReadResult{Content, BytesUsed, LastTimestamp, Partial}. emitTranscripts
advances the marker to LastTimestamp on partial reads. filterBySourceMarkers
switched from >= to strict-greater so the boundary session does not loop.
End-to-end smoke confirms re-runs advance the marker even within a single
oversized session."
```

- [ ] **Step 2: Remove the plan doc**:

```bash
git rm docs/superpowers/plans/2026-05-27-fix-641-intra-session-splitting.md
git commit -m "chore: remove completed plan doc for #641

AI-Used: [claude]"
```

---

## Self-Review

**1. Spec coverage** vs the issue + user directive:
- ✓ "doesn't advance past the byte cap on re-runs" — fixed by strict-greater filter (Task 4).
- ✓ "split session log reads" — fixed by intra-session row-timestamp marker (Tasks 1-4).
- ✓ Per-source semantics preserved (Claude + OpenCode both get the new reader shape).
- ✓ Marker file format unchanged; no migration.
- ✓ Real-world smoke test (Task 5 Step 3).

**2. Placeholder scan** — Task 3 has one "spike to verify SQLite column type" callout; that's a concrete check, not a placeholder.

**3. Type consistency** — `ReadResult` used everywhere; `Reader` interface updated in one place; `Partial` always paired with `LastTimestamp`.

**4. Risk of regression on existing tests** — yes, every test that constructed a stub `Reader` or called `JSONLReader.Read` needs updating. Task 4 Step 5 calls this out.
