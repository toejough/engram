# Lever #1 — lazy matched-chunk content (clusters-first one-read payload)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans. Steps use `- [ ]` checkboxes.

**Goal:** The structural half of #1. After the recent-fill cut (−28%), the remaining payload bulk is the **matched-chunk content** (~58 KB: 15 full + ~204 160-rune snippets). Add a `--lazy-chunks` mode that renders matched **chunk** items as path/score/cluster only (NO content), so the agent reads notes + cluster structure + paths in ~one read and pages a specific chunk's evidence **on-demand** (`engram show`). **Notes (fact/feedback) keep full content — the wins are untouched.** Latency/experience win (~another payload halving), NOT a dollar win.

**Decision recorded (per /please anti-sycophantic):** considered pivoting to #2 (async learn — safer, bigger, fully-measurable latency win); Joe weighed the risk and chose #1. Proceeding, with a **manual chunk-evidence spot-check** added (Step 4) to cover the risk the note-based trap gate cannot measure.

**Known risk + tension:** lazy chunks trade "read 219 snippets upfront" for "fetch the few you actually need" — a net win ONLY if the agent rarely needs full chunk evidence (notes are the load-bearing part; chunks are supplementary — note 72). If the agent ends up `engram show`-ing many chunks, this pulls against #4 (inline candidate content). The spot-check measures whether chunk-evidence quality holds.

## Global Constraints
- Go change → **`targ`** (`targ test`, `targ check-full`); binary install via `go install ./cmd/engram` (no `targ build`).
- **Skill edit → `superpowers:writing-skills` TDD** (note 26 + engram CLAUDE.md): RED baseline of the old behavior → edit → pressure tests. Non-waivable.
- Query-output field/mode pattern (note 56): flag-gated, render-path only, query stays read-only (no sidecar writes).
- **Standing guardrail (non-negotiable):** trap gate before/after (must stay GREEN — notes untouched) + free payload delta + the chunk-evidence spot-check. Never touch the win-nucleus (note content, Step-3 conventions-as-requirements, Step-2.5B recency-weight, matched-note retrieval, description).

## Files
- Modify: `internal/cli/query.go` — add `LazyChunks bool` flag (mirror `ContentBudget` tag style); in the render path, when set, leave matched chunk items' `Content` empty (skip `capChunkContent`'s snippet/full for chunks → path-only). Notes unaffected.
- Modify: `internal/cli/export_test.go` + a `query_*_test.go` — test the lazy render (chunk items get empty Content; note items keep Content).
- Modify: `skills/recall/SKILL.md` — Step 2 query command gains `--lazy-chunks`; Channel-1 description: chunks are path-only, `engram show <source#anchor>` on-demand for evidence; notes carry full content as before. Update the Step-2.5 / red-flag rows that assume inline chunk content.
- Step 5 docs: C4 query-output description (c1/c2/c3), GLOSSARY (two-channel), ROADMAP #1.

---

### Task 1: `--lazy-chunks` binary mode (TDD)

**Interfaces (mirror `ContentBudget`):**
- Flag: `LazyChunks bool targ:"flag,name=lazy-chunks,env=ENGRAM_LAZY_CHUNKS,desc=render matched chunk items as path/score only (no content); the agent pages a chunk's evidence on-demand via engram show — shrinks the recall payload"`.
- Render: when `LazyChunks`, matched `kind: chunk` items emit empty `Content` (note items keep full content). Implement as a guard in the chunk-content path (e.g., before/in `capChunkContent`, or a `clearChunkContent(items)` helper applied when lazy). Read-path only.

- [ ] **Step 1: RED** — test (mirror `query_cap_test.go` style, `cli.Export*` seam): build a small `[]queryItem` mix (1 note w/ content, 2 chunks w/ content); assert the lazy transform clears chunk Content but preserves note Content. (Confirm the exact exported seam by reading how `capChunkContent`/`resolveContentBudget` are tested.)
- [ ] **Step 2: Run RED** — `targ test` → FAIL.
- [ ] **Step 3: GREEN** — add the flag + the lazy render guard + the `Export*` test wrapper.
- [ ] **Step 4: Run GREEN** — `targ test` → PASS. **Step 5: `targ check-full`** → clean. **Step 6: Commit** — `feat(query): --lazy-chunks mode (path-only matched chunks; notes keep content)`

### Task 2: recall SKILL.md — consume lazy chunks (writing-skills TDD)

**Invoke `superpowers:writing-skills`.** The edit: Step-2 query command adds `--lazy-chunks`; Channel-1 text changes from "chunks carry content inline, extract the principle" to "chunk items carry path/source only — `engram show <source#anchor>` on-demand when a chunk's evidence is needed; notes carry full content, apply directly." Update any red-flag/Step-2.5 row that assumes inline chunk content.

- [ ] **Step 1: RED baseline** — per writing-skills: capture the current behavior (the skill reads inline chunk snippets) as the baseline the edit changes; identify the rationalization loopholes ("I'll just read whatever content is there").
- [ ] **Step 2: Edit (GREEN)** — make the Channel-1 + Step-2 command changes; keep the note-handling (load-bearing) unchanged.
- [ ] **Step 3: Pressure tests** — per writing-skills, verify the edited skill resists the loopholes (agent fetches chunk evidence on-demand rather than assuming it's inline, and doesn't skip notes).
- [ ] **Step 4: Commit** — `feat(recall): consume --lazy-chunks — fetch chunk evidence on-demand, notes inline`

### Task 3: Verify (safety + win + the unmeasured-risk spot-check)

- [ ] **Step 1: Free payload delta.** Rebuild (`go install ./cmd/engram`); run the baseline 10-phrase query WITH `--lazy-chunks` (as recall now does) vs without; record size/token delta. Expect ~165 KB → ~107 KB (~−35% further; ~−53% vs the original 230 KB).
- [ ] **Step 2: Trap gate (safety, ~$3).** `gate.py --tier smoke` with the rebuilt binary + the edited skill → **GREEN** (notes untouched). RED/INCONCLUSIVE → stop.
- [ ] **Step 3: Chunk-evidence spot-check (the gate-blind risk).** Run 2–3 real recalls (the edited skill) on tasks where a *chunk* (not a note) carries the load-bearing evidence; confirm the agent fetches it on-demand via `engram show` and uses it — i.e. lazy chunks didn't silently drop evidence the agent needed. Record the outcome honestly (if the agent skips needed evidence, that's a finding → reconsider the default).
