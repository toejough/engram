# Note-vs-chunk ranking — three tracks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use superpowers:subagent-driven-development or
> executing-plans. Steps use `- [ ]`. Tracks 1–2 are Go/TDD; Track 3 is an analysis (its "tests" are
> the gauges, not unit tests).

**Goal:** Ensure that when a high-relevance crystallized note is the sole source of the knowledge a query
needs, lower-signal-density items don't bury it — and audit whether the notes themselves are question-useful.
**Success is the knowledge-delivery gauge moving** (probe real-path note recall@5: 0.19 → toward the 0.81
isolation ceiling), **NOT "notes outrank chunks"** (that's the tautology Joe corrected). Chunks are the
*source* the lessons were distilled from — lower signal-density, **not noise**; the fix targets
density-vs-relevance, never type-vs-type. Context: the lesson-carrying note is the sole knowledge source in
~41% of drowned cases (`docs/design/2026-06-28-retrieval-probe-results.md`).

**Architecture:** Three independent tracks, sequenced so each gauge stays clean. **Track 1 (floor)** —
reserve per-phrase slots for relevance-qualified notes at the drowning site (`mergePhraseIntoUnion`). **Track
2 (down-weight)** — a separate, separately-gated unit that reduces the score of low-density chunk types
(subagent turn-1 dispatch prompts). **Track 3 (audit)** — analysis: are cluster-driven notes question-useful
vs correction-driven, measured against the 137 failure-mining situations. Tracks 1 and 2 are committed
separately; Track 3 produces a report.

**Relationship to note 82** (`recall-miss-is-structural-not-retrieval`): note 82 has two facets — (a)
"crystallized notes are outranked by raw chunks even on a lever-keyed query" (the *ranking* facet — this plan
fixes it) and (b) "recall fires once... never re-checks a lever invented during synthesis" (the *timing /
re-entry* facet — recall-before-recommend). This plan addresses (a) only; (b) remains a separate follow-up.

**Tech Stack:** Go (no CGO), `targ` for test/lint/check, the probe harness (`score_probe.py`), the trap gate
(`dev/eval/traps/gate.py`), the `recall_cost` `$METER`.

## Global Constraints

- **Gate every code change with the trap harness AND the probe.** `dev/eval/traps/gate.py` C3–C6 GREEN
  before+after each of Tracks 1 and 2; `score_probe.py` real-path note recall@5 measured before+after (the
  primary gauge); `recall_cost` `$METER` to confirm the payload doesn't balloon. (Standing roadmap rule.)
- **Justify by knowledge-delivery, not item-rank:** the floor is warranted because the value test showed the
  de-drowned note is the *sole* source of needed knowledge in ~41% of drowned cases while the displaced chunks
  scored like noise (mean 1.23 ≈ prior 1.14); success = the probe gauge moving, not "notes rank higher"
  tautologically.
- **Win-nucleus — deliberate, gated exception (not an omission).** The roadmap names *Step-2 matched-note
  retrieval* in the win-nucleus ("never touch"). Tracks 1 and 2 DO modify matched-note retrieval (the binary's
  per-phrase merge + chunk scoring) — stated plainly. This is warranted because (a) the constraint's own safety
  mechanism is the trap harness, run before+after every code change here; and (b) the measured drowning is
  *itself* a win-nucleus degradation — the lessons that produce the C3–C6 wins aren't surfacing — so the floor
  RESTORES the win-nucleus rather than risking it. The C3–C6 gate is the hard backstop: any regression →
  revert/retune. It edits no SKILL.md and does NOT touch the Step-3 directive, Step-2.5B recency-weight, or the
  frontmatter `description`.
- **Gate B** (per the `/please` workflow) = a fresh-context design-fit reviewer dispatched on the refactored
  diff — not a self-check.
- **Respect the existing pipeline** (note 36): the floor operates at `matchedSetItem` level (pre
  `scoredChunk→resolvedItem` conversion), so the recency-ordering hazard does not apply. Do NOT reorder the
  scoredChunk→resolvedItem conversion.
- **Test style** (note 39): white-box helpers via `cli.ExportXxx` accessors in `export_test.go`, package
  `cli_test`, gomega + the nilaway guard patterns; RunQuery integration via `QueryDeps{Scan, Read,
  ListChunkIndexes, Embedder: axisEmbedder, Now}`, manifest keys = `record.Source`. Every test `t.Parallel()`.

---

## Track 1 — Matched-note floor (the surgical fix)

### Task 1: Reserve per-phrase note slots in `mergePhraseIntoUnion`

**Files:**
- Modify: `internal/cli/query.go` — `mergePhraseIntoUnion` (~985–1018) + a new const + a helper.
- Modify: `internal/cli/export_test.go` — add `ExportMergePhraseIntoUnion` (or `ExportCapWithNoteFloor`) accessor.
- Test: `internal/cli/query_helpers_test.go` (white-box unit) + the probe harness as the end-to-end gauge.

**Interfaces:**
- Consumes: `mergePhraseIntoUnion(noteHits []scoredCandidate, chunkHits []scoredChunk, byKey map[string]matchedSetItem)`; `matchRelevanceFloor=0.25`; `matchPhraseLimit=30`. (`scoredCandidate` is unexported → add a test type alias `ExportScoredCandidate = scoredCandidate` in `export_test.go`, mirroring the existing `ExportScoredChunk = scoredChunk`.)
- Produces: `capWithNoteFloor(perPhrase []matchedSetItem, limit, noteFloorK int) []matchedSetItem`. **Formal spec:** a note *qualifies for the floor* iff `item.isChunk == false && item.baseScore >= matchRelevanceFloor` (0.25). Let `Q` = the count of floor-qualifying notes in `perPhrase`, and `reserve = min(noteFloorK, Q)`. The function returns a length-`min(limit, len(perPhrase))` slice that contains the `reserve` highest-score qualifying notes, with the remaining slots filled by the highest-score items overall (chunks or extra notes), evicting the lowest-score chunks to make room. **Edge cases:** `Q == 0` → return the top-`limit` by score unchanged (no notes to protect); `len(perPhrase) <= limit` → return all (nothing evicted); `reserve >= limit` (more qualifying notes than slots) → return the top-`limit` qualifying notes. Pure + deterministic: stable sort, `key` ascending as the score tiebreak (matching the existing `applyFloorAndCap` tiebreak).

- [ ] **Step 1: Baseline gauge.** Run the probe to record the pre-change number:
  `python3 docs/design/2026-06-28-retrieval-probe-data/score_probe.py docs/design/2026-06-28-retrieval-probe-data/probes.json /tmp/probe_before.json` → confirm real-path nuanced note recall@5 ≈ 0.19. Run `python3 dev/eval/traps/gate.py` → record C3–C6 baseline GREEN.

- [ ] **Step 2: Write the failing unit test.** In `query_helpers_test.go`, via the export accessor: build
  `noteHits` = one note with `baseScore=0.40` (≥ floor) and `chunkHits` = 30 chunks with `baseScore`/`score`
  0.50–0.70. Call the merge; assert the note's key IS present in `byKey`.

```go
func TestMergePhraseIntoUnion_FloorKeepsRelevantNoteVsManyChunks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	note := cli.ExportScoredCandidate("vault/relevant-note.md", 0.40, 0.40) // path, score, baseScore
	chunks := cli.ExportNScoredChunks(30, 0.50, 0.70)                        // 30 chunks above the note
	byKey := cli.ExportMergePhraseIntoUnion([]cli.ExportScoredCandidateT{note}, chunks)
	g.Expect(byKey).To(HaveKey("vault/relevant-note.md")) // RED today: evicted by the 30 chunks
}
```

- [ ] **Step 3: Run it — verify RED.** `targ test` (or the focused package). Expected: FAIL — the note is
  absent (`matchPhraseLimit` cut it). If the export accessors don't exist yet, add minimal ones to
  `export_test.go` first (they're test-only re-exports; that is the analogue of writing the test).

- [ ] **Step 4: GREEN — add the floor.** In `query.go`: a const `noteFloorK = 5` (documented: reserve up to
  K floor-clearing notes per phrase so chunks cannot fully evict them). Replace the bare truncation
  `if len(perPhrase) > matchPhraseLimit { perPhrase = perPhrase[:matchPhraseLimit] }` with
  `perPhrase = capWithNoteFloor(perPhrase, matchPhraseLimit, noteFloorK)`. Implement `capWithNoteFloor`:
  sort by score desc (already sorted); take the top-`limit`; if it contains fewer than
  `min(noteFloorK, totalFloorClearingNotes)` notes, swap the lowest-score chunks in the kept set for the
  highest-score notes not yet kept (only notes with `baseScore ≥ matchRelevanceFloor`). Keep it pure +
  deterministic (stable sort, key tiebreak).

- [ ] **Step 5: Run it — verify GREEN.** `targ test` — the unit passes.

- [ ] **Step 6: REFACTOR + Gate B.** Keep `capWithNoteFloor` small, single-responsibility, named clearly;
  ensure `mergePhraseIntoUnion` still reads cleanly. Run `targ check-full` (all errors at once). Then **Gate
  B** (design-fit reviewer) on the diff.

- [ ] **Step 7: Verify with the real gauges.** `go install ./cmd/engram`; re-run the probe →
  `/tmp/probe_after.json`; assert real-path nuanced note recall@5 **rose materially toward 0.81** (report the
  number, not a claim). Re-run `gate.py` → C3–C6 still GREEN (no regression). Run `recall_cost` to confirm the
  payload didn't balloon. **If recall@5 didn't move, the floor isn't reaching the eviction — STOP and
  diagnose, don't tune blindly.**

- [ ] **Step 8: Commit** (Track 1 alone — clean, independently measured).

---

## Track 2 — Down-weight low-density chunk types (separate, separately gated)

### Task 2: Penalize subagent turn-1 dispatch chunks in scoring

**Files:**
- Modify: `internal/cli/query_chunks.go` — `scoreChunkForPhrase` (~203, where each chunk's `score` is
  computed); add a pure helper `applyDensityPenalty(score float32, record chunk.Record) float32`.
- Modify: `internal/cli/export_test.go` — add `ExportApplyDensityPenalty` accessor.
- Test: `internal/cli/query_chunks_test.go` (the existing chunk-scoring white-box test file).

**Interfaces:**
- Consumes: `scoreChunkForPhrase` builds each `scoredChunk{record, score, baseScore}`; `chunk.Record` already
  exposes `Source` and `Anchor` (both in scope — `scoreChunkForPhrase` reads `maxTurnBySrc[record.Source]` and
  `parseTurnN(record.Anchor)`).
- Produces: a const `chunkDensityPenalty = float32(0.7)` and `applyDensityPenalty`, applied to a chunk's
  **`score`** (NOT `baseScore` — `baseScore` is the raw cosine the relevance floor reads, so the floor stays
  honest) when the chunk is a low-density dispatch prompt: `record.Anchor == "turn-1"` AND
  `strings.Contains(record.Source, "/subagents/")`. Modification at the `score` assignment:

```go
// before (~query_chunks.go:203, inside scoreChunkForPhrase per chunk):
score := baseScore * recencyMultiplier // (existing recency-biased score)
// after:
score := applyDensityPenalty(baseScore*recencyMultiplier, record)
```

- [ ] **Step 1: Write the failing test** in `query_chunks_test.go`:

```go
func TestApplyDensityPenalty_DownweightsSubagentDispatchChunk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	const raw = float32(0.60)
	dispatch := cli.ExportChunkRecord("/p/proj/abc/subagents/agent-x.jsonl", "turn-1") // Source, Anchor
	lesson := cli.ExportChunkRecord("/p/proj/abc/session.jsonl", "turn-7")
	g.Expect(cli.ExportApplyDensityPenalty(raw, dispatch)).To(BeNumerically("<", raw)) // RED: no penalty today
	g.Expect(cli.ExportApplyDensityPenalty(raw, lesson)).To(Equal(raw))                // unaffected
}
```

- [ ] **Step 2: Run it — verify RED.** `targ test` → FAIL (`ExportApplyDensityPenalty` returns `raw`
  unchanged because the helper doesn't exist / is a no-op). Add the export accessor + a no-op helper first if
  needed to make it compile-then-fail.
- [ ] **Step 3: GREEN.** Implement `applyDensityPenalty` (multiply by `chunkDensityPenalty` for the
  dispatch-prompt pattern only; identity otherwise) and wire it into the `score` assignment. Document the
  heuristic + that it's score-only (baseScore/floor untouched).
- [ ] **Step 4: Run it — verify GREEN.** `targ test`.
- [ ] **Step 5: REFACTOR + Gate B** (`targ check-full`; the `/please` design-fit reviewer on the diff).
- [ ] **Step 6: Gauge + honesty.** Re-run the probe (does note recall@5 improve further, or is it already
  saturated by Track 1?) and `gate.py` (C3–C6 GREEN — a chunk-evidence-dependent axis must NOT regress; if
  any does, the penalty is hurting a needed chunk → reduce/scope it). Re-run `recall_cost`. **Report the
  marginal gap Track 2 closes over Track 1; if it's ~0, say so plainly** (the floor may already suffice).
- [ ] **Step 7: Commit** (Track 2 alone) — or, if the gauge shows it's net-neutral/negative, do NOT ship it;
  record the negative finding in the doc.

---

## Track 3 — Crystallization-quality audit (analysis, parallel)

### Task 3: Are the notes question-useful? (coverage + utility + path-split)

**Files:**
- Create: `docs/design/2026-06-28-crystallization-audit.md` + `…-crystallization-audit-data/`.
- Reuse: the 137 failure situations (`docs/design/2026-06-28-failure-eval-data/confirmed-failures.json`,
  each carries `first_pass_trigger` + `lesson`) as the real question corpus; the isolation-probe method
  (`ENGRAM_CHUNKS_DIR=<empty> engram query`) to query notes-only.

- [ ] **Step 1: Coverage.** For each failure situation's `first_pass_trigger`, run a notes-only query;
  judge (agent) whether a surfaced note actually answers it. Report the **coverage rate** (what fraction of
  real questions has a question-answering note) and the **uncovered** set (questions no note answers).
- [ ] **Step 2: Utility.** For a sample of existing notes, judge whether each serves *some* real question
  (from the corpus) or is a cluster artifact nothing asks for. Report the **note-utility rate**.
- [ ] **Step 3: Path-split.** Compare cluster-driven notes (recall Step 2.5, `source: synthesized from chunk
  cluster`) vs correction-driven notes (learn Step 2, `source: session … correction`) on question-usefulness
  (situation-handle match to real queries). Note 68: engram does aggregation, not synthesis — so test whether
  cluster-aggregated notes are pitched at the question or the cluster.
- [ ] **Step 4: Verdict.** If cluster-driven notes are measurably less question-useful, the highest-leverage
  upstream fix is **crystallizing question-shaped situations** (derive the handle from anticipated questions,
  not the cluster) — flag it as the next investigation; do NOT implement it here (out of scope).

---

## Sequencing + the three gauges
Floor (Task 1) → its own commit; down-weight (Task 2) → its own commit (or a recorded no-ship); audit (Task 3)
runs in parallel (no code risk). Each code task reads the same three gauges before/after: **probe real-path
note recall@5** (signal surfaced; ceiling 0.81), **gate.py C3–C6** (no regression), **recall_cost** (payload
sanity). The remaining gap to 0.81 after Task 1 decides whether Task 2 / the audit's upstream fix are worth it.

## Docs to update after the code lands (Step 5 — Document)
- `docs/architecture/c1-system-context.md` — the recall-flow merge description (currently "top-30 per phrase
  (notes+chunks); ... cap matched set at ~300") goes stale; add the note floor: "...reserving up to
  `noteFloorK` slots for relevance-qualified notes so chunks cannot fully evict them."
- `docs/ROADMAP.md` — record the floor as the shipped note-vs-chunk-ranking lever with the measured
  before/after probe number; note Track-2/normalization/two-channel as the ranked follow-ups.
- `docs/design/2026-06-28-retrieval-probe-results.md` — append the post-floor real-path note recall@5 number
  (the "first gated experiment" now has a result).

## Out of scope
Two-channel restructure; per-population normalization; richer note embeddings; implementing question-shaped
crystallization (Track 3 only *measures* whether it's needed). These are the ranked follow-ups, not this run.

## Risks
- **Floor too blunt** (caps relevant notes / promotes a marginal one): mitigated by the relevance-floor gate
  on reserved slots + the trap gate; if C3–C6 regress, retune K_note or switch to normalization (follow-up).
- **Down-weight heuristic wrong** (a dispatch chunk WAS the needed evidence): the trap gate is the backstop;
  Task 2 ships only if the gauge is net-positive.
- **Audit is agent-judged** — bounded by the corpus; report it as a sample, and the path-split as suggestive.
