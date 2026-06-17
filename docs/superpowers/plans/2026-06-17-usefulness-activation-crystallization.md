# Usefulness-Activation: Recency for L2 Notes + Useful-Triggered Crystallization

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make *usefulness* an activation signal: a memory that **surfaces in a recall payload AND clears a relevance cutoff** gets activated — useful L2 notes get their recency refreshed (`LastUsed`), useful chunk clusters crystallize into L2s (even when a chunk already states the idea, because chunks decay), and L2 notes now decay and carry a most-recently-used floor band, mirroring chunks.

**Architecture:** A new additive `LastUsed` field on the `.vec.json` sidecar (no schema bump, excluded from `ContentHash` → no re-embed). `rankCandidates` gains a recency multiplier keyed on `LastUsed` (falling back to `created`), reusing `recency.go`. `RunQuery` applies ONE combined floor band (newest chunks ∪ most-recently-used notes). **`engram query` stays read-only** — it emits an `activated` flag per returned NOTE item (surfaced AND base-cosine ≥ cutoff). A new **`engram activate <note-path>...`** command bumps `LastUsed` (atomic per-file sidecar write, no lock). The `recall` skill calls `engram activate` with the flagged note paths, crystallizes above-cutoff chunk clusters per the three bands, and for `≥0.95` clusters calls `engram activate` on the covering L2.

**Tech Stack:** Go 1.26 (no CGO), `internal/embed` (sidecar), `internal/cli` (query/recency/activate), gomega + blackbox `*_test` packages via `export_test.go` aliases, `targ test`/`targ check-full`, `superpowers:writing-skills` for the recall SKILL.md change.

**Design decisions (verified against the code; reversals named candidly):**
- **Activation = surfaced (in the returned payload, post-decay+cap) AND base cosine ≥ `activationCosineCutoff`.** (Operator's choice.) Judged on **base cosine** (`max(situation,body)`, pre-decay) for the *cutoff*, but the item must also have *surfaced* — so a dormant note that decayed out of the payload deactivates (ACT-R-correct), bounded by the most-recently-used band + gentle 60d decay.
- **`LastUsed` is additive metadata:** `omitempty`, NOT in `ContentHash` (verified: `hash.go:20-27` hashes situation+body of the raw note only), NO `SidecarSchemaVersion` bump → old sidecars decode `LastUsed=""` and stay valid (`UnmarshalSidecar` ignores unknown keys; schema check passes). No `engram embed apply` migration.
- **Reversal 1 (named):** v1 kept recency on chunks only (notes pure cosine, per c1-system-context.md). This applies decay to notes too — operator-directed ("apply the same recency checks to L2s"). This is *universal note decay*, broader than the lazy-L2 spec's recency-bias-on-divergence tiebreaker; it supersedes that framing for ranking.
- **Reversal 2 (named):** the lazy-L2 `≥0.95` band was a dedup *no-op* (silence). It becomes a *refresh*: the covering L2 was useful → bump its `LastUsed`. Consequence (intended ACT-R feedback loop): regularly-useful L2s stay fresh; never-retrieved L2s decay and lose rank. No new L2 is created in this band (dedup-create still holds; `<0.80` handles uncovered topics).
- **Read-only query preserved:** plain `engram query` writes nothing; activation is a separate explicit command. No flock needed (verified: the vault flock guards Luhmann ID sequencing, not sidecar files; `os.WriteFile` is atomic per-file).
- Reuse `recencyMultiplier`/`sourceAgeDays`/`defaultRecencyParams`/`hoursPerDay` (recency.go) and `fillRecencyBand` (item-agnostic); do not duplicate.

---

## File Structure

| File | Responsibility | Action |
|---|---|---|
| `internal/embed/embedder.go` | Add `LastUsed string` (omitempty) to `Sidecar` | Modify |
| `internal/embed/sidecar_test.go` (or existing embed test) | Round-trip + backward-compat (old sidecar → `LastUsed=""`) | Modify/Create |
| `internal/cli/recency.go` | `noteAgeDays(lastUsed, created, now)`, `parseCreatedFromNote(note)`, `mostRecentlyUsedNoteItems(...)`; reuse `recencyMultiplier` | Modify |
| `internal/cli/query.go` | Note recency in `rankCandidates` (both callers) + `baseScore`; `activated` flag in payload; combined band in `RunQuery` | Modify |
| `internal/cli/activate.go` | `bumpLastUsed(...)` + `RunActivate(args, deps)`; `engram activate` target | Create |
| `internal/cli/targets.go` | wire `engram activate <note-path>...` | Modify |
| `internal/cli/{recency,query,activate}_test.go`, `export_test.go` | TDD + aliases (`package cli_test`) | Modify/Create |
| `skills/recall/SKILL.md` | call `engram activate` for flagged notes; crystallize above-cutoff clusters; ≥0.95 → activate covering L2 | Modify (writing-skills) |
| `docs/architecture/c1-system-context.md` | recall flow: note decay + band + activation/refresh | Modify (Gate C) |

---

## Phase 1 — Sidecar `LastUsed` field (additive, no migration)

### Task 1.1: Add `LastUsed` to `Sidecar`

**Files:** Modify `internal/embed/embedder.go`; Test the embed test package.

- [ ] **Step 1: Write the failing test** (use `strconv.Itoa`, NOT an undefined `itoa`; import `strconv`):

```go
func TestSidecarLastUsedRoundTripsAndIsOptional(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	s := embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "minilm-l6-v2@384", Dims: 1,
		SituationVector: []float32{0.1}, BodyVector: []float32{0.2}, ContentHash: "sha256:x", LastUsed: "2026-06-17",
	}
	got, err := embed.UnmarshalSidecar(embed.MarshalSidecar(s))
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }
	g.Expect(got.LastUsed).To(Equal("2026-06-17"))

	// A sidecar WITHOUT last_used (the pre-feature shape) still decodes; LastUsed empty.
	noKey := []byte(`{"schema_version":` + strconv.Itoa(embed.SidecarSchemaVersion) +
		`,"embedding_model_id":"minilm-l6-v2@384","dims":1,"situation_vector":[0.1],"body_vector":[0.2],"content_hash":"sha256:x"}`)
	got2, err2 := embed.UnmarshalSidecar(noKey)
	g.Expect(err2).NotTo(HaveOccurred())
	if err2 != nil { return }
	g.Expect(got2.LastUsed).To(Equal(""))
}
```

- [ ] **Step 2: Run** `targ test` → FAIL (`unknown field LastUsed`).

- [ ] **Step 3: Implement** — add after `ContentHash` in `Sidecar` (embedder.go:74):

```go
	// LastUsed is the date (YYYY-MM-DD) this note last surfaced as a useful
	// (above-cutoff) recall hit. Additive metadata: omitempty, EXCLUDED from
	// ContentHash (hash.go hashes situation+body of the raw note, not this), and
	// it does NOT bump SidecarSchemaVersion — old sidecars decode LastUsed=""
	// ("never used"). Never feed LastUsed into any hash: bumping it must not
	// mark a note stale.
	LastUsed string `json:"last_used,omitempty"` //nolint:tagliatelle // sidecar JSON keys are spec contract
```

- [ ] **Step 4: Run** `targ test` → PASS.
- [ ] **Step 5:** `targ check-full` → 8/8 (confirms `engram check` / staleness unaffected — the field is hash-excluded). If `check` flags sidecars, the field leaked into a hash; fix.
- [ ] **Step 6: Commit** `feat(embed): additive LastUsed sidecar field (no migration)` + `AI-Used: [claude]`.

---

## Phase 2 — Note recency decay in `rankCandidates`

### Task 2.1: `noteAgeDays` (LastUsed → created fallback)

**Files:** Modify `internal/cli/recency.go`, `recency_test.go`, `export_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestNoteAgeDaysPrefersLastUsedThenCreated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	g.Expect(cli.ExportNoteAgeDays("2026-06-15", "2026-01-01", now)).To(BeNumerically("~", 2.0, 0.01))
	g.Expect(cli.ExportNoteAgeDays("", "2026-06-10", now)).To(BeNumerically("~", 7.0, 0.01))
	g.Expect(cli.ExportNoteAgeDays("", "", now)).To(BeNumerically("~", 0.0, 0.01))
}
```

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement** (append to recency.go; `hoursPerDay` already defined there at line 18):

```go
const noteDateFormat = "2006-01-02"

// noteAgeDays returns a note's age in days for recency decay, preferring LastUsed
// (when it last proved useful) over created. Empty/unparseable → 0 (treat as
// fresh; a malformed date must never penalize).
func noteAgeDays(lastUsed, created string, now time.Time) float64 {
	stamp := lastUsed
	if stamp == "" {
		stamp = created
	}

	parsed, err := time.Parse(noteDateFormat, stamp)
	if err != nil {
		return 0
	}

	age := now.Sub(parsed).Hours() / hoursPerDay
	if age < 0 {
		age = 0
	}

	return age
}
```

Add `ExportNoteAgeDays = noteAgeDays` to export_test.go.

- [ ] **Step 4: Run** `targ test` → PASS. **Commit** `feat(cli): note age (LastUsed→created) for recency decay (STM)`.

### Task 2.2: `parseCreatedFromNote` (fresh — does NOT reuse resituate.parseCreated)

**Files:** Modify `internal/cli/recency.go`, `recency_test.go`, `export_test.go`

> Verified: `resituate.go:126 parseCreated(string)(time.Time,error)` parses an *already-extracted* string — it does NOT extract from raw note bytes. Write `parseCreatedFromNote` fresh.

- [ ] **Step 1: Write the failing test**

```go
func TestParseCreatedFromNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	note := []byte("---\ntype: fact\ncreated: 2026-06-10\nsituation: x\n---\nbody")
	g.Expect(cli.ExportParseCreatedFromNote(note)).To(Equal("2026-06-10"))
	g.Expect(cli.ExportParseCreatedFromNote([]byte("no frontmatter"))).To(Equal(""))
}
```

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement** (append to recency.go):

```go
// parseCreatedFromNote extracts the `created:` frontmatter date (YYYY-MM-DD)
// from a note's raw bytes, or "" when absent.
func parseCreatedFromNote(note []byte) string {
	for _, line := range strings.Split(string(note), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "created:"); ok {
			return strings.TrimSpace(rest)
		}
	}

	return ""
}
```

Add `ExportParseCreatedFromNote = parseCreatedFromNote`.

- [ ] **Step 4: Run** `targ test` → PASS. **Commit** `feat(cli): created-date extractor for note recency (STM)`.

### Task 2.3: Apply recency multiplier + keep `baseScore` in `rankCandidates`

**Files:** Modify `internal/cli/query.go`, `query_test.go`

`rankCandidates` is at query.go:1660; `scoredCandidate` at query.go:337. **TWO callers** must thread `now`: `runSinglePhraseQuery` (query.go:1859) and `runSynthesisQuery` (query.go:2066) — both pass `deps.Now()`.

- [ ] **Step 1: Write the failing test** (blackbox via the existing rankCandidates test fixture pattern in query_test.go): two notes, equal base cosine, one `LastUsed` 2 days ago, one 120 days ago, fixed `now` → fresh ranks first; assert both retain their pre-decay `baseScore`.

- [ ] **Step 2: Run** `targ test` → FAIL (signature/recency).

- [ ] **Step 3: Implement:**
  - `scoredCandidate` (query.go:337) gains `baseScore float32`, `lastUsed string`, `created string` (pre-decay cosine + the two recency dates).
  - **`resolvedItem` (query.go:322) ALSO gains `baseScore float32`, `lastUsed string`, `created string`** — these MUST flow to the `RunQuery` level, because the combined band (Phase 4) and the `activated` renderer (Phase 3.3) only ever see `[]resolvedItem`, never `[]scoredCandidate` (the candidate slice is created inside `runSinglePhraseQuery` and never returned up). Populate them at the `scoredCandidate`→`resolvedItem` conversion (note items only; chunk items leave them zero — chunks get their recency from the manifest, not these fields).
  - `rankCandidates` gains a `now time.Time` param. After `score, coord := bestVector(...)`:
    ```go
    created := parseCreatedFromNote(noteBytes)
    base := score
    recencyScore := base
    if !now.IsZero() {
        ageDays := noteAgeDays(hit.sidecar.LastUsed, created, now)
        recencyScore = base * float32(recencyMultiplier(ageDays, 0, defaultRecencyParams()))
    }
    candidates = append(candidates, scoredCandidate{
        notePath: notePath, basename: hit.note.Basename,
        score: recencyScore, baseScore: base, coord: coord,
        lastUsed: hit.sidecar.LastUsed, created: created,
        content: stripWikilinks(string(noteBytes)),
    })
    ```
    (`turnFrac=0` for notes — no session-tail concept.) Sort stays on `score`.
  - **Both call sites pass `deps.Now()`:** `runSinglePhraseQuery` (query.go:1859) directly, and the call at **query.go:2066 inside `unionDirectHits`** (~line 2050, invoked by `runSynthesisQuery` ~1886) — thread `now` through `unionDirectHits`'s signature to reach it. **Fallback:** when `deps.Now == nil`, pass `time.Time{}` → `now.IsZero()` skips decay (pure cosine; existing tests unchanged).
  - At the `scoredCandidate`→`resolvedItem` conversion, copy `baseScore`/`lastUsed`/`created` across (grep for where `scoredCandidate` becomes `resolvedItem` — the aggregation/resolve step).

- [ ] **Step 4: Run** `targ test` → PASS (new + all existing query tests). **Commit** `feat(cli): recency decay on note ranking, both callers (STM)`.

---

## Phase 3 — `engram activate` command + read-only `activated` flag

### Task 3.1: `bumpLastUsed` sidecar writer

**Files:** Create `internal/cli/activate.go`, `activate_test.go`, `export_test.go`

- [ ] **Step 1: Write the failing test** (in-memory read/write; preserves vectors/hash):

```go
func TestBumpLastUsedSetsDatePreservesVectors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	orig := embed.Sidecar{SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@1", Dims: 1,
		SituationVector: []float32{0.1}, BodyVector: []float32{0.2}, ContentHash: "sha256:x"}
	store := map[string][]byte{"n.vec.json": embed.MarshalSidecar(orig)}
	read := func(p string) ([]byte, error) { return store[p], nil }
	write := func(p string, b []byte) error { store[p] = b; return nil }

	err := cli.ExportBumpLastUsed("n.vec.json", "2026-06-17", read, write)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }
	got, derr := embed.UnmarshalSidecar(store["n.vec.json"])
	g.Expect(derr).NotTo(HaveOccurred())
	if derr != nil { return }
	g.Expect(got.LastUsed).To(Equal("2026-06-17"))
	g.Expect(got.ContentHash).To(Equal("sha256:x"))
	g.Expect(got.BodyVector).To(Equal([]float32{0.2}))
}
```

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement** (`activate.go`):

```go
package cli

import (
	"fmt"

	"github.com/toejough/engram/internal/embed"
)

// bumpLastUsed reads a note's sidecar, sets LastUsed=date, and rewrites it.
// Vectors/ContentHash are preserved (LastUsed is metadata) so it never triggers
// a re-embed. Idempotent for the same date. No lock: sidecar writes are atomic
// per-file and the vault flock guards only Luhmann ID sequencing.
func bumpLastUsed(sidecarPath, date string, read func(string) ([]byte, error), write func(string, []byte) error) error {
	data, err := read(sidecarPath)
	if err != nil {
		return fmt.Errorf("activate: reading sidecar %s: %w", sidecarPath, err)
	}

	sidecar, err := embed.UnmarshalSidecar(data)
	if err != nil {
		return fmt.Errorf("activate: parsing sidecar %s: %w", sidecarPath, err)
	}

	if sidecar.LastUsed == date {
		return nil
	}

	sidecar.LastUsed = date

	if err := write(sidecarPath, embed.MarshalSidecar(sidecar)); err != nil {
		return fmt.Errorf("activate: writing sidecar %s: %w", sidecarPath, err)
	}

	return nil
}
```

Add `ExportBumpLastUsed = bumpLastUsed`. (Also add a test: read-failure error wraps the path + `activate:` prefix.)

- [ ] **Step 4: Run** `targ test` → PASS. **Commit** `feat(cli): bumpLastUsed sidecar writer (STM activation)`.

### Task 3.2: `engram activate <note-path>...` command

**Files:** Modify `internal/cli/activate.go`, `internal/cli/targets.go`, `activate_test.go`

- [ ] **Step 1: Write the failing test**: `RunActivate` over given note paths bumps each note's sidecar `LastUsed` to today (via injected `Now`/`Read`/`Write`); a missing sidecar is logged and skipped, not fatal.

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement:**
  - `ActivateArgs struct { Notes []string `targ:"flag,name=note,desc=note path to mark used (repeatable)"` }`.
  - `ActivateDeps struct { Now func() time.Time; Read func(string)([]byte,error); Write func(string,[]byte) error; LogWarning func(string, ...any) }`.
  - `RunActivate(args ActivateArgs, deps ActivateDeps) error`: `date := deps.Now().Format(noteDateFormat)`; for each note path → `embed.SidecarPath(path)` → `bumpLastUsed(...)`; collect errors, log-and-continue (a bad path must not fail the batch). Return nil unless ALL failed.
  - Wire in targets.go: `engram activate` target calling `RunActivate` with `newOsActivateDeps()` (`Now: time.Now, Read: os.ReadFile, Write: os.WriteFile, LogWarning: logWarningToStderrf`).

- [ ] **Step 4: Run** `targ test` → PASS. **Commit** `feat(cli): engram activate command — mark notes used (STM)`.

### Task 3.3: `activated` flag in the query payload (read-only)

**Files:** Modify `internal/cli/query.go`, `query_test.go`

- [ ] **Step 1: Write the failing test**: a query whose returned items include a NOTE with `baseScore ≥ activationCosineCutoff` and another below → payload marks the first `activated: true`, the second omits/false; CHUNK items are never marked (crystallization is the skill's job). Plain query writes nothing (no sidecar mutation).

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement:**
  - `const activationCosineCutoff = 0.5` (provisional default — a sanity floor for "genuine hit", NOT an empirically-tuned optimum; see Task 6.2 and #646).
  - When rendering note items, emit `activated: true` iff the item is a note AND its `baseScore >= activationCosineCutoff`. The renderer reads `resolvedItem.baseScore` directly (added to `resolvedItem` in Task 2.3) — no parallel structure or recomputation needed.
  - No write. Query remains pure-read.

- [ ] **Step 4: Run** `targ test` → PASS. **Commit** `feat(cli): emit activated flag for above-cutoff note hits (read-only) (STM)`.

---

## Phase 4 — Combined floor band (newest chunks ∪ most-recently-used notes)

> The shipped chunk band runs INSIDE `mergeChunkSpace` (query.go:1281-1283). To avoid the two bands evicting each other, LIFT the band application to `RunQuery` and run ONE `fillRecencyBand` over a COMBINED must-include set. `mergeChunkSpace` stops applying the band and instead RETURNS its newest-chunk must-include set; `RunQuery` unions it with the note must-include and applies a single band after the full merge+cap.

### Task 4.1: `mostRecentlyUsedNoteItems`

**Files:** Modify `internal/cli/recency.go`, `recency_test.go`, `export_test.go`

- [ ] **Step 1: Write the failing test**: given a `[]resolvedItem` containing note items (kind != chunk) with varied `lastUsed`/`created` plus some chunk items, `mostRecentlyUsedNoteItems(items, now, 3)` returns the 3 freshest-used NOTE items (newest age first), ignoring chunk items.

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement** — operates on the merged `[]resolvedItem` (which now carry `lastUsed`/`created`/`baseScore` from Task 2.3); selects note-kind items only (chunks get their floor from `mergeChunkSpace`'s manifest-mtime path). `chunkItemKind` is the existing chunk-kind constant (query.go:127):

```go
// mostRecentlyUsedNoteItems returns the n note items (kind != chunkItemKind)
// with the smallest noteAgeDays (freshest LastUsed→created), newest first — the
// note side of the combined floor band. Operates on the merged resolvedItems,
// which carry lastUsed/created (Task 2.3).
func mostRecentlyUsedNoteItems(items []resolvedItem, now time.Time, n int) []resolvedItem {
	if n <= 0 || now.IsZero() {
		return nil
	}

	notes := make([]resolvedItem, 0, len(items))
	for _, it := range items {
		if it.kind != chunkItemKind {
			notes = append(notes, it)
		}
	}

	sort.SliceStable(notes, func(i, j int) bool {
		return noteAgeDays(notes[i].lastUsed, notes[i].created, now) <
			noteAgeDays(notes[j].lastUsed, notes[j].created, now)
	})

	if n > len(notes) {
		n = len(notes)
	}

	return notes[:n]
}
```

Add `ExportMostRecentlyUsedNoteItems` + an `ExportNewNoteResolvedItem(notePath string, lastUsed, created string)` test constructor (sets kind to the note default) as needed.

- [ ] **Step 4: Run** `targ test` → PASS. **Commit** `feat(cli): most-recently-used note items for the floor band (STM)`.

### Task 4.2: Single combined band in `RunQuery`

**Files:** Modify `internal/cli/query.go`, `query_test.go`

- [ ] **Step 1: Write the failing test**: merged+capped items are all stale; the 3 newest chunks AND 3 most-recently-used notes are absent → after the combined band, ALL 6 appear (none evicts another), lowest-ranked non-must (pure-relevance) items displaced, budget = limit preserved. Add a case where combined must-include (6) < limit (20) — no relevance starvation.

- [ ] **Step 2: Run** `targ test` → FAIL.

- [ ] **Step 3: Implement:**
  - `mergeChunkSpace`: remove the `fillRecencyBand` call (query.go:1281-1283); instead RETURN the newest-chunk must-include `[]resolvedItem` it built (so the band runs once, later). Adjust its signature/return; update its test.
  - At the level where `mergeChunkSpace` runs and `merged.resolvedItems` holds the unified note+chunk items (the re-confirm located this at the `RunQuery`/`aggregatePhraseSummaries` level, where `mergeChunkSpace` is already called): build `noteMust := mostRecentlyUsedNoteItems(merged.resolvedItems, deps.Now(), defaultRecencyFloor)` — it reads `lastUsed`/`created` off the merged note items (Task 2.3 put them there), so NO `[]scoredCandidate` is needed at this level. Then `combined := append(chunkMust, noteMust...)`; `merged.resolvedItems = fillRecencyBand(merged.resolvedItems, combined, limit)`. Guard on `deps.Now != nil`. One call, one protected set → no mutual eviction. (`fillRecencyBand` already clamps `len(mustInclude) > limit`.)

- [ ] **Step 4: Run** `targ test` + `targ check-full` → green. **Commit** `feat(cli): single combined recency band (chunks ∪ recently-used notes) (STM)`.

---

## Phase 5 — recall skill (REQUIRED: superpowers:writing-skills)

**Files:** Modify `skills/recall/SKILL.md`

- [ ] **Step 1:** Invoke `superpowers:writing-skills`. Baseline (RED): Step 2 query has no activation follow-through; Step 2.5 crystallizes only on novelty bands + a vocabulary-coincidence gate; useful L2s aren't refreshed.
- [ ] **Step 2:** Edit:
  - After the `engram query`, **call `engram activate --note <path> ...`** for every returned item flagged `activated: true` (the binary computed it — the agent just forwards those paths). One batched call.
  - Reframe Step 2.5 around **usefulness**: crystallize an above-cutoff chunk cluster **even if a chunk states the idea clearly** (chunks decay), keeping the vocabulary-coincidence gate. Bands: `<0.80` → create the L2; `0.80–0.95` → update the nearest L2 (a write, which also refreshes it); **`≥0.95` → call `engram activate --note <covering-L2-path>`** (the covering L2 was useful → refresh its `LastUsed`; do NOT create a duplicate).
- [ ] **Step 3:** GREEN — verify behavioral change. Pressure tests (concrete): (1) a returned note flagged `activated` → the agent issues `engram activate` for its path; (2) an above-cutoff chunk cluster with `nearest_l2 ≥0.95` → agent activates the covering L2 (no new L2); (3) `<0.80` cluster → agent creates a new L2; (4) a below-cutoff weak-tail note → NOT activated.
- [ ] **Step 4: Commit** `feat(recall): usefulness-driven activation + crystallization (STM)`.

---

## Phase 6 — Docs (Gate C) + cutoff sanity

### Task 6.1: Document the model

**Files:** `docs/architecture/c1-system-context.md` (recall flow)

- [ ] Update the recall-flow narrative: notes now recency-weighted (decay by `LastUsed`→`created`) and share a combined most-recently-used/newest floor band with chunks; query emits `activated` flags (surfaced AND above cutoff, read-only); `engram activate` refreshes `LastUsed`; crystallization is usefulness-triggered; the `≥0.95` band now refreshes rather than no-ops. Name the two reversals (chunks-only → notes-too; dedup-silence → refresh) and the feedback loop (used notes stay fresh; unused decay). Cite `query.go`/`recency.go`/`activate.go`. Gate C reviews.

### Task 6.2: Activation-cutoff sanity (honest scope)

**Files:** `internal/cli/recency_eval_test.go` (extend — the file EXISTS from prior STM work)

- [ ] Add `TestActivationCutoffSeparatesHitsFromTail`: synthetic note `baseScore`s spanning 0.2–0.8 → only `≥ activationCosineCutoff` flag `activated`. **State honestly in the test comment:** this validates the cutoff is *wired and non-degenerate* (not 0, not 1) — it is NOT an empirically-tuned optimum; true tuning (does 0.5 separate useful from noise on real recalls) is the e2e value-proof (#646).

---

## Self-Review

- **Spec coverage:** "crystallize useful even if stated clearly" → Phase 5 (`<0.80` create regardless of chunk clarity; vocabulary-coincidence gate retained). "usefulness = activation signal" → Phase 3.3 (surfaced AND base-cosine cutoff). "useful L2 → bump mtime" → Phase 1 (`LastUsed`) + Phase 3 (`engram activate`) + Phase 5 (`≥0.95` refresh). "apply same recency checks to L2s" + "decay + most-recently-used floor" → Phase 2 (decay) + Phase 4 (combined band, note side keyed on `LastUsed`). ✓
- **Reversals named:** Reversal 1 (note decay) and Reversal 2 (≥0.95 refresh) stated in Design decisions + documented in Phase 6 with the feedback-loop consequence. ✓
- **Read-only query preserved + no flock:** Phase 3 splits the write into `engram activate`; `bumpLastUsed` takes no lock (justified). ✓
- **Both `rankCandidates` callers** (1859, 2066) threaded (Task 2.3). ✓
- **No migration:** Phase 1 additive, hash-excluded, no schema bump; Task 1.1 Step 5 verifies. ✓
- **Mutual-eviction fixed:** single combined band (Task 4.2), not two sequential calls. ✓
- **No false reuse:** `parseCreatedFromNote` written fresh (Task 2.2); reuses recency.go helpers + `fillRecencyBand`. ✓
- **Placeholder scan:** `strconv.Itoa` (not `itoa`); flock instruction removed; ≥0.95 mechanism decided (`engram activate`), not OR. ✓
- **Type consistency:** `Sidecar.LastUsed`, `scoredCandidate.{baseScore,lastUsed,created}`, `noteAgeDays`, `parseCreatedFromNote`, `bumpLastUsed`, `RunActivate`/`ActivateArgs`/`ActivateDeps`, `activationCosineCutoff`, `mostRecentlyUsedNoteItems`, combined-band in RunQuery — consistent. ✓
- **Provisional honesty:** `activationCosineCutoff=0.5` flagged provisional (Task 3.3, 6.2); eval is a wiring sanity check, not tuning (deferred to #646). ✓
