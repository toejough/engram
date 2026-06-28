# Note-vs-chunk ranking — three tracks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use superpowers:subagent-driven-development or
> executing-plans. Steps use `- [ ]`. Tracks 1–2 are Go/TDD; Track 3 is an analysis (its "tests" are
> the gauges, not unit tests).

**Goal:** Stop a relevance-qualified crystallized note from being drowned by chunks in `engram query`, and
audit whether the notes themselves are question-useful — fixing engram's measured retrieval failure
(real-path note recall@5 = 0.19 vs 0.81 isolation; the de-drowned note is the sole knowledge source ~41% of
drowned cases).

**Architecture:** Three independent tracks, sequenced so each gauge stays clean. **Track 1 (floor)** —
reserve per-phrase slots for floor-clearing notes at the drowning site (`mergePhraseIntoUnion`). **Track 2
(down-weight)** — a separate, separately-gated unit that reduces the score of low-density chunk types
(subagent turn-1 dispatch prompts). **Track 3 (audit)** — analysis: are cluster-driven notes question-useful
vs correction-driven, measured against the 137 failure-mining situations. Tracks 1 and 2 are committed
separately; Track 3 produces a report.

**Tech Stack:** Go (no CGO), `targ` for test/lint/check, the probe harness (`score_probe.py`), the trap gate
(`dev/eval/traps/gate.py`), the `recall_cost` `$METER`.

## Global Constraints

- **Gate every code change with the trap harness AND the probe.** `dev/eval/traps/gate.py` C3–C6 GREEN
  before+after each of Tracks 1 and 2; `score_probe.py` real-path note recall@5 measured before+after (the
  primary gauge); `recall_cost` `$METER` to confirm the payload doesn't balloon. (Standing roadmap rule.)
- **Justify by knowledge-delivery, not item-rank** (note 119): the floor is warranted because the de-drowned
  note is the sole knowledge source ~41% of drowned cases; success = the probe gauge moving, not "notes rank
  higher" tautologically.
- **Never touch the win-nucleus:** Step-3 conventions directive, Step-2.5B recency-weight, the frontmatter
  `description`. This is a binary change to the query merge — it alters matched-note retrieval, so the trap
  gate is mandatory, but it does NOT edit any SKILL.md.
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
- Consumes: `mergePhraseIntoUnion(noteHits []scoredCandidate, chunkHits []scoredChunk, byKey map[string]matchedSetItem)`; `matchRelevanceFloor=0.25`; `matchPhraseLimit=30`.
- Produces: a `capWithNoteFloor(perPhrase []matchedSetItem, limit, noteFloorK int) []matchedSetItem` helper that keeps the top-`limit` by score but guarantees the top `min(noteFloorK, #notes≥floor)` floor-clearing notes are retained (evicting the lowest-score chunks to make room).

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
- Modify: `internal/cli/query.go` — `scoreChunkForPhrase` (~545) or a post-score adjustment on `scoredChunk`.
- Modify: `internal/cli/export_test.go` — accessor if needed.
- Test: `internal/cli/query_test.go` / `query_helpers_test.go`.

**Interfaces:**
- Consumes: `scoreChunkForPhrase(queryVec, records, now, maxTurnBySrc, recency)`; `chunk.Record{Source, Anchor}`.
- Produces: a `chunkDensityPenalty` (const, e.g. 0.7) applied to a chunk's `score` (NOT `baseScore` — keep
  the floor honest) when `record.Anchor == "turn-1"` AND `record.Source` contains `/subagents/` (a dispatch
  prompt, not the lesson).

- [ ] **Step 1: RED.** Test that a subagent turn-1 chunk and a non-dispatch chunk with equal raw cosine end
  up with the dispatch chunk ranked lower after scoring.
- [ ] **Step 2: Verify RED** (`targ test`).
- [ ] **Step 3: GREEN.** Apply `chunkDensityPenalty` to `score` for the dispatch-prompt pattern only;
  document the heuristic + that it's score-only (baseScore/floor untouched).
- [ ] **Step 4: Verify GREEN** (`targ test`).
- [ ] **Step 5: REFACTOR + Gate B** (`targ check-full`; design-fit reviewer on the diff).
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

## Out of scope
Two-channel restructure; per-population normalization; richer note embeddings; implementing question-shaped
crystallization (Track 3 only *measures* whether it's needed). These are the ranked follow-ups, not this run.

## Risks
- **Floor too blunt** (caps relevant notes / promotes a marginal one): mitigated by the relevance-floor gate
  on reserved slots + the trap gate; if C3–C6 regress, retune K_note or switch to normalization (follow-up).
- **Down-weight heuristic wrong** (a dispatch chunk WAS the needed evidence): the trap gate is the backstop;
  Task 2 ships only if the gauge is net-positive.
- **Audit is agent-judged** — bounded by the corpus; report it as a sample, and the path-split as suggestive.
