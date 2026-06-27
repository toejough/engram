# Lever #1 — lazy matched-chunk content (defer bulk chunk text to on-demand)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans. Steps use `- [ ]` checkboxes.

**Goal:** Deliver #1's substance — *defer bulk chunk text to on-demand* so the load-bearing content (notes + cluster structure) dominates what the agent reads. After the recent-fill cut (−28%), the remaining payload bulk is the matched-**chunk** content (~58 KB: 15 full + ~204 160-rune snippets). Add a `--lazy-chunks` mode that renders matched **chunk** items path/score only (NO content); the agent pages a specific chunk's evidence on-demand via `engram show-chunk`. **Notes (fact/feedback) keep full content — the wins are untouched.**

**On "clusters-first" + "one read" (per Gate A — being precise, not over-claiming):**
- **Clusters-first is implicit, not a silent drop.** Notes already score above chunks, so they arrive first in `items:`; with chunk content emptied, the first agent read is phrases + full notes + cluster `candidate_l2s` paths. Code review confirmed clusters/`candidate_l2s` are independent of item content. An *explicit* reorder of the `clusters:` block was assessed as **marginal** (notes already lead) and is **not pursued** — recorded here, not dropped.
- **The win is the deterministic payload cut, NOT a literal "one read."** ~165 KB → ~107 KB (~−35% further; ~−53% vs the original 230 KB). At ~25-30 KB/read that's still ~3-4 reads, not one. Task 3 measures the byte/token delta (deterministic) and qualitatively observes the agent's reads in the spot-check; "one read" is aspirational and is NOT claimed as the success metric.

**Scope vs roadmap #1:** this IS #1's core mechanism (lazy-content). Step 5 marks #1's lazy-content **DONE** and records that explicit clusters-block reordering was assessed marginal and skipped. Not "half" — the substance, with the one marginal sub-piece explicitly accounted for.

**Decision recorded (anti-sycophantic):** considered pivoting to #2 (async learn — safer, bigger, fully-measurable); Joe weighed the risk and chose #1. Proceeding, with a **chunk-evidence spot-check** (Task 3) covering the risk the note-based trap gate cannot measure.

**Known risk + tension:** lazy chunks trade "read ~219 snippets upfront" for "fetch the few you actually need" — a net win ONLY if the agent rarely needs full chunk evidence (notes are load-bearing; chunks are supplementary — note 72). If the agent ends up `engram show-chunk`-ing many chunks, this pulls against #4 (inline candidate content). The spot-check measures whether chunk-evidence quality holds. **RESOLVED 2026-06-27:** 0 fetches across 13 realistic uninstructed recalls (no iterative tax) and 2/2 on-target sole-source fetches (no evidence drop) — see Outcome below.

## Global Constraints
- Go → **`targ`** (`targ test`, `targ check-full`); binary install `go install ./cmd/engram` (no `targ build`).
- **Skill edit → `superpowers:writing-skills` TDD** (note 26 + CLAUDE.md): RED baseline → edit → pressure tests. Non-waivable.
- Render-path only, flag-gated, query stays read-only (note 56). `--lazy-chunks` is **OPT-IN** (recall sets it); never a changed default (eval harnesses + `retrieval_probe.py` confirmed unaffected by code review).
- **Standing guardrail:** trap gate before/after (GREEN — notes/clusters/candidate_l2s untouched, code-confirmed) + free payload delta + chunk-evidence spot-check. Never touch the win-nucleus.

## Files
- `internal/cli/query.go` — add `LazyChunks bool` flag (mirror `ContentBudget` tag + `//nolint:lll`); add `clearChunkContent(items []queryItem) []queryItem` (zero `Content` for `Kind == chunkItemKind`); in `renderQueryPayload` (~line 1305) call it **before** `capChunkContent` when `LazyChunks` (capChunkContent still runs — harmless no-op on emptied chunks; `ChunksSnippeted` reports 0). Notes never touched (capChunkContent already guards `Kind != chunkItemKind`).
- `internal/cli/export_test.go` — add `ExportClearChunkContent(kinds, contents []string) []string` mirroring `ExportCapChunkContent` (export_test.go:147).
- `internal/cli/query_cap_test.go` — the lazy unit test.
- `skills/recall/SKILL.md` — the three enumerated edits (below), via writing-skills TDD.
- Step 5 docs: c2-containers.md:97 (+ c1/c3 as needed) + GLOSSARY items[]-structure; ROADMAP #1.

---

### Task 1: `--lazy-chunks` binary mode (TDD)

**Interfaces:** `LazyChunks bool` flag `targ:"flag,name=lazy-chunks,env=ENGRAM_LAZY_CHUNKS,desc=render matched chunk items path/score only (no content); the agent fetches a chunk's evidence on-demand via engram show-chunk — shrinks the recall payload"` `//nolint:lll`. `clearChunkContent` zeroes Content for chunk items only.

- [ ] **Step 1: RED** — in `query_cap_test.go`, mirror the **`ExportCapChunkContent` pattern (export_test.go:147)**: build items via `ExportClearChunkContent(kinds=["fact","chunk","chunk"], contents=["noteC","c1","c2"])`; assert the two chunk contents are cleared (`""`) and the note content (`"noteC"`) is preserved.
- [ ] **Step 2: Run RED** — `targ test` → FAIL. **Step 3: GREEN** — add the flag, `clearChunkContent`, the `renderQueryPayload` guard, and `ExportClearChunkContent`. **Step 4: PASS** — `targ test`. **Step 5: `targ check-full`** → clean. **Step 6: Commit** — `feat(query): --lazy-chunks mode (path-only matched chunks; notes keep content)`

### Task 2: recall SKILL.md — consume lazy chunks (writing-skills TDD)

**Invoke `superpowers:writing-skills`.** The RED baseline must surface that the current skill text assumes inline chunk content — **three enumerated edits**:

1. **Channel-1 (lines 82-90):** add after the chunk/note bullet list: *"With `--lazy-chunks` (recall's default invocation), chunk items carry path + source/anchor but NO content; `engram show-chunk <source#anchor>` to fetch a chunk's evidence on-demand. Notes (fact/feedback) always carry full content inline — apply directly."*
2. **Step 2.5A (lines 115-122, esp. line 121):** replace *"For chunk members not in `items[]`, use the chunk content from the cluster's `members` list"* (already a dead letter — `queryClusterMember` has no content field) with: *"For chunk members, the content is not in the payload — `engram show-chunk <source#anchor>` on-demand to read the evidence. Note members in `items[]` carry `content` — use it directly."*
3. **Red-flag row (line 277):** make conditional — *"Note members in `items[]` carry `content` — use directly. Chunk items carry no content under `--lazy-chunks` — `engram show-chunk` them when you need the evidence."*
- The Step-2 query command (Step 2 block) gains `--lazy-chunks`.

- [ ] **Step 1: RED baseline** — run the current skill on a real recall (existing vault); capture the transcript showing the agent reading inline chunk snippets. That transcript + the 3 lines above (which instruct/assume inline chunk content) are the baseline the edit changes. Loophole to pressure-test: "the chunk content might be inline somewhere, I'll just read what's there" (must instead `engram show-chunk`).
- [ ] **Step 2: Edit (GREEN)** — make the 3 edits + the command change; leave note handling unchanged.
- [ ] **Step 3: Pressure tests** — construct a recall where chunk items have no content (lazy on) and a chunk holds needed evidence; verify the edited skill (a) emits `engram show-chunk <source#anchor>` for the needed chunk, (b) applies the fetched content, (c) reads notes inline without fetching, (d) does not skip notes. 1-2 scenarios.
- [ ] **Step 4: Commit** — `feat(recall): consume --lazy-chunks — fetch chunk evidence on-demand, notes inline`

### Task 3: Verify (safety + win + the gate-blind spot-check)

- [ ] **Step 1: Free payload delta.** `go install ./cmd/engram`; run the baseline 10-phrase query WITH `--lazy-chunks` vs without; record size/token/item delta. Expect ~165 KB → ~107 KB. Also note the qualitative read-count from the spot-check recall (Step 3) — but the headline is the deterministic byte/token cut, not a "one read" claim.
- [ ] **Step 2: Trap gate (safety, ~$3).** `gate.py --tier smoke` with the rebuilt binary + edited skill → **GREEN** (notes/clusters untouched). RED/INCONCLUSIVE → stop.
- [ ] **Step 3: Chunk-evidence spot-check (the gate-blind risk).** **Construct a synthetic fixture vault:** seed a NOTE whose content omits a key fact, and a CHUNK (`engram ingest --markdown`) that uniquely carries that fact (the load-bearing evidence). Run the edited recall (lazy on) on a query that needs the fact. **Acceptance:** the transcript shows `engram show-chunk <source#anchor>` for that chunk AND the agent uses the fetched fact. Run 2-3 such scenarios (note: N=2-3 is low power — report honestly). If the agent skips needed chunk evidence → finding → reconsider making lazy the recall default (it stays opt-in; could narrow to high-confidence cases).

---

## Self-Review (vs Gate A)
- ask: clusters-first reframed as implicit (notes-first by score, code-confirmed) + explicitly assessed-marginal-and-skipped (not silently dropped); "one read" softened to the deterministic −53% payload; #1 lazy-content = the substance, Step 5 closes that + records the marginal remainder; decision/latency framing front-and-center ✓.
- code: `clearChunkContent` between renderItems + capChunkContent (capChunkContent still runs, harmless); notes/clusters/candidate_l2s untouched → gate GREEN; opt-in (other callers safe); ExportClearChunkContent mirrors ExportCapChunkContent; `//nolint:lll` noted ✓.
- docs: the 3 enumerated SKILL.md edits (82-90, 115-122/121, 277) + Step-5 C4 items[]-structure + GLOSSARY ✓.
- clarity: ExportCapChunkContent seam named; baseline = live recall transcript; pressure tests concrete; spot-check = synthetic fixture vault + acceptance ✓.

---

## Outcome (2026-06-27) — SHIPPED

All three Task-3 gates passed; the on-demand-vs-dump risk was measured, not assumed.

- **Task 1 (binary):** `--lazy-chunks` mode + `clearChunkContent` + `budget.lazy_chunks`; Gate B passed; `targ check-full` 8/8.
- **Task 2 (skill):** recall SKILL.md consumes lazy chunks; the fetch command is **`engram show-chunk`** (a new read-only subcommand) — `engram show` resolves vault **notes** only and returned "note not found" on chunk ids, which would have silently dropped evidence. Caught and fixed before merge.
- **Step 1 — payload delta:** **−33.7%** on the 10-phrase baseline (146→97 KB, ~36.5K→24.2K tokens). Stacks on #5 for cumulative ~230→97 KB (~−58%).
- **Step 2 — trap gate:** **GREEN** (C3/C4i/C5/C6 wins hold; matched notes/clusters untouched).
- **Step 3 — fetch-rate + capability (expanded per Joe's challenge that iterative fetch may cost more than a dump):**
  - **Sparing:** 4 varied uninstructed recalls + 9 warm trap recalls = **0 `show-chunk` fetches across 13/13**, despite up to 234 chunks available. Notes carried every synthesis (note 72). No iterative tax exists to trade for.
  - **Capable:** 2/2 sole-source fixtures (a note naming a topic but omitting the load-bearing specific; the value living only in a path-only chunk) — the uninstructed agent reached for `show-chunk` on its own, fetched the right chunk, and surfaced the exact fact. No evidence drop.
  - **Verdict:** selection is reliable both ways. Lazy is a clean net win. No sparing guard added (no over-fetch baseline to fix; a prohibition would risk suppressing the capability case).
