# #684 Payload Restructure Implementation Plan — measure first, build conditionally

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the segmented recall-time baseline (#684's own gate: the post-cuts binary/agent split is unmeasured), size the payload-consumption slice the restructure can actually reach, present the number + recommendation to Joe at a mid-cycle checkpoint, and build the clusters-first/deduped payload ONLY if Joe green-lights on the measured ceiling.

**Architecture:** One instrumentation task extends `dev/eval/traps/recall_time.py` with phase segmentation (transcript-timestamp spans per recall phase) and a payload byte census (bytes per payload section, from the trial's own `engram query` output). A pre-registered checkpoint mapping recommends build/close to Joe. The conditional build task reorders `queryPayload` (clusters before items) and dedupes candidate-note content out of `items[]`, with the recall skill's consumption contract updated under writing-skills TDD and note-107's two-gate lazy-retrieval validation.

**Tech Stack:** Python (instrumentation), Go (`internal/cli/query.go` payload rendering, conditional), skill markdown (conditional), targ gates, trap-gate smoke brackets.

## Global Constraints

- `targ` only for test/lint/check; check-full runs POST-commit (includes check-uncommitted), fix + `git commit --amend` on the unpushed branch until green.
- Every commit subject ≤72 bytes `wc -c`-measured; trailer `AI-Used: [claude]`.
- SKILL.md edits only via superpowers:writing-skills. Production vault/chunks are read-only copy sources; all trials sandboxed (recall_time.py's existing pattern). Diff-scope check before every commit.
- Branch `684-payload-restructure`; review before push; ff-only merge.
- Trap gate (controller-run): BEFORE on the unchanged tree, AFTER on the final tree IF the build proceeds (binary + skill are behavior surface); a measurement-only cycle needs no AFTER (no behavior surface changed) — the BEFORE run then serves as the cycle's gate record. C5b-honored-only RED at n=1 → one same-tree re-run; expensive runs are canary-gated after any degradation signal (note 252).
- **Prior-outcome reconciliation (note 80, surfaced at orientation):** a 2026-06-24 recall restructure (host/haiku split) was ROLLED BACK — quality held but the op-level delta (~14%) didn't justify the complexity. This cycle's lever is different (payload shape, not model split), so the prior does not close it — but its moral is codified here as the checkpoint gate: no build without a measured ceiling that clears the bar.
- **Measured preconditions** (2026-07-12, live tree; executors re-verify):
  - `internal/cli/query.go`: `queryPayload` struct order = Version, Phrases, **Items, Clusters**, Budget (YAML renders in struct order); `matchSetCap = 300`; under `lazyChunks`, `clearChunkContent` strips chunk content but NOTE items keep full content in `items[]` WHILE the same notes' content also rides inline in `candidate_l2s` (O2) — candidate-note content is duplicated in today's payload.
  - Baseline from #657 (LEDGER `recall-time-remeasure`): recall-only vault-copy median 51.9s, range 39.3–63.6s (n=3 directional). The binary/agent split post-cuts is UNMEASURED (the pre-cuts ~3.5s binary / ~43–63% paging figures are superseded vintage). Small-fixture floor 39.7s.
  - `dev/eval/traps/recall_time.py` exists (vault-copy mode, pinned delimiters, caffeinate + span>wall + degraded-cost gates); trial transcripts carry per-message timestamps and tool-call records (the segmentation source).
- **Pre-registered CHECKPOINT mapping** (recommends — Joe disposes, per note 254; fires after Task 1):
  - Segment the recall span into: (a) pre-query (session start → the `engram query` tool call), (b) the query call itself, (c) payload consumption (query return → the first Step-2.5 write/activate/synthesis-start marker), (d) the remainder (synthesis + activation + closing). The restructure's reach is (c) plus any (b) render share — call it the **addressable slice**.
  - addressable slice median **< 15s** → recommend **close #684 measured-out** (even total elimination beats the 51.9s median by <30%; the complexity moral of note 80 applies). Source of the constant: ~12s is the measured corpus-scale-attributable gap between the vault-copy median and the small-fixture floor (51.9 − 39.7), i.e. the payload-scale component's indirect measurement from #657; 15s adds slack above it. A judgment constant, labeled as such.
  - addressable slice median **≥ 15s** → recommend **build** (Tasks 2–4).
  - Either way the checkpoint goes to Joe via AskUserQuestion with the segmented numbers + evidence pointers BEFORE any build task dispatches.
- Scope guards: the recall skill's Step 2.5 judgment contract (agent judges coverage; binary never decides) is untouched in any branch; no new payload fields beyond the restructure; `--lazy-chunks` semantics for chunks unchanged.

---

## Controller pre-flight (before Task 1)

- [ ] Branch: `git checkout -b 684-payload-restructure` from main.
- [ ] Trap gate BEFORE: canary first if any degradation signal; `python3 dev/eval/traps/gate.py --tier smoke` → GREEN; log to `$CLAUDE_JOB_DIR/tmp/684-gate-before.log`.

### Task 1: Phase segmentation + payload census (the measure)

**Files:**
- Modify: `dev/eval/traps/recall_time.py`

**Interfaces:**
- Consumes: the existing trial/delimiter/validity machinery.
- Produces: per-trial `phases` dict (pre_query_s, query_call_s, payload_consumption_s, remainder_s) + `payload_census` dict (total_bytes, items_notes_content_bytes, items_meta_bytes, clusters_candidate_content_bytes, clusters_meta_bytes, recent_bytes, duplicated_note_content_bytes) in the artifact; the checkpoint's addressable-slice number.

- [ ] **Step 1: Segmentation design, verified against a REAL transcript first** (note 237): before writing parser code, take one #657 vault-copy trial transcript (paths in `dev/eval/cumulative/recall_time/657-recall-time.json` → durable copies alongside the artifact; if absent, run one fresh trial) and locate mechanically: the first `engram query` tool-call record (assistant tool_use), its tool_result record, and the Step-2.5 start (first `engram amend`/`activate`/write-side tool call after the query return, or the synthesis message if no write occurs). Print the three timestamps + the derived phase spans. STOP if any marker cannot be located mechanically — escalate the method.
- [ ] **Step 2: Implement** the segmentation + census in recall_time.py (census parses the trial's captured query payload — save the payload YAML per trial; measure section bytes by re-rendering or slicing the saved YAML; duplicated_note_content_bytes = sum of content bytes appearing in BOTH items[] notes and candidate_l2s for the same path).
- [ ] **Step 3: Run** n=3 vault-copy trials (canary-gated). Artifact: append under a `segmented` key in a NEW artifact file `$CLAUDE_JOB_DIR/tmp/684-recall-segments.json`; per-trial phases + census + median/range per phase. Same validity gates; discard-never-pool.
- [ ] **Step 4: Commit** the script change — subject: `feat(eval): phase segmentation + payload census for recall-time (#684)` (70 bytes, `wc -c`-measured) + trailer. check-full post-commit.
- [ ] **Step 5: Report** — the addressable slice median + range, the census (how many KB the duplication and the items-before-clusters ordering actually cost), evidence pointers, and which checkpoint band the median lands in.

### CHECKPOINT (controller): Joe disposes build vs measured-out

- [ ] Present via AskUserQuestion: the segmented spans, the census, the mapping's recommendation, evidence pointers. Joe's call gates Tasks 2–4. If Joe says close: skip to Task 5 with the measured-out disposition.

### Task 2 (conditional): Binary — clusters-first, deduped payload

**Files:**
- Modify: `internal/cli/query.go` (+ tests per repo conventions: imptest/rapid/gomega, t.Parallel, nilaway patterns)

- [ ] RED: table-driven tests asserting the NEW payload shape: `Clusters` renders before `Items` (struct order swap); note items whose path appears in any cluster's `candidate_l2s` render path/score/provenance only (content omitted — the candidate copy is authoritative); notes NOT in any candidate list keep inline content; chunk/lazy semantics unchanged; budget gains `items_content_deduped` count (no-silent-caps rule). Run RED.
- [ ] GREEN: minimal implementation. REFACTOR + Gate B (design-fit reviewer, sonnet).
- [ ] `go install ./cmd/engram` and verify with the real binary from a non-data-dir cwd (note 125): a real query shows clusters first and deduped items.

### Task 3 (conditional): Recall skill consumption contract (writing-skills TDD)

**Files:**
- Modify: `skills/recall/SKILL.md` (Step 2/2.5 reading order + the items[]-content cross-reference sentences)

- [ ] RED probes on the OLD text against a NEW-shape payload fixture (does the agent stumble — look for content in items[], mis-order its reading?); GREEN edit (read clusters first — they lead the payload; candidate content lives ONLY in candidate_l2s; items[] is the match-overview + recent channel); pressure probe; deploy via `engram update` + wrap-insensitive greps with measured baselines.
- [ ] Note-107 two-gate validation (the deduped items are the "deferred content"): (a) SPARING — across 3 varied sandbox recalls, the agent's `engram show` fetch count for deduped items stays ~0 (they're in candidate_l2s already — expected 0; any fetch is a red flag); (b) CAPABLE — one sole-source fixture where a non-candidate matched note's content matters → the agent fetches it unprompted. Both must hold.

### Task 4 (conditional): Re-measure + gates

- [ ] n=3 segmented vault-copy trials on the new binary+skill → the win is the measured delta on the addressable slice AND the total, reported directional with n + range. Trap gate AFTER → GREEN.

### Task 5: Docs + disposition

- [ ] LEDGER: `recall-time-split` row (the segmented baseline; plus the post-build delta row if built). ROADMAP Track B: #684 outcome record per what happened (built + delta, or measured-out). Concept-level verification greps. Commit — subject: `docs(684): baseline split recorded; disposition per measurement` (63 bytes, measured) + trailer.

## Controller close-out

- Final whole-branch review; Gate C over touched docs; Gate D over outward prose (#684 comment carries: the segmented baseline, the checkpoint decision and Joe's call, the build outcome or measured-out disposition, gate records).
- Rebase if main moved → check-full → ff-only merge → push → #684 disposition per Joe (the checkpoint already captured it; the close comment records it). Plan doc retired in a follow-up commit.
