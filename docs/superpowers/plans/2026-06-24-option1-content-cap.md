# Option 1 — content-cap implementation + min-spend sweep

> **For agentic workers:** TDD each unit (RED → GREEN → REFACTOR). Steps use `- [ ]`.

**Goal:** Add a content-budget cap to `engram query` that trims chunk bulk from the recall payload,
then binary-search the cap down to the **knee** — the smallest cap that still preserves quality on
the currently-passing tests (Go units + C3/C4/C5/C6 warm eval arms) — and report the min-spend point.

**Architecture:** A single knob caps how many **chunk** items (in existing rank order) render with
full content; lower-ranked chunks (the ~200-chunk recency channel ranks last, so it trims first) get
a one-line snippet. **Note items are never capped** — crystallized lessons are the gold and few.
The knob is read from `--content-budget N` (flag) or `ENGRAM_CONTENT_BUDGET` (env; flag wins). The
env path lets the sweep inject the cap into the *real* recall skill (which calls `engram query` with
no flag) without a per-value skill edit. `0` = unlimited (current behavior, the safe default until
the knee is known). Query stays **read-only** (note 51).

**Tech stack:** Go (`internal/cli/query.go`), imptest/rapid/gomega tests, Python eval harnesses,
`targ` for build/test/check.

## Global Constraints

- Query stays read-only — no sidecar writes (note 51).
- `targ check-full` clean on first commit; nilaway/gomega patterns per `.claude/rules/go.md`.
- Default behavior unchanged until the knee is chosen (`--content-budget` default `0` = unlimited).
- Cost claims report **both** payload tokens AND $; a "quality holds" call must clear the **noise
  floor** sized from the same warm-vs-warm contrast, n≥3 — a one-trial dip is not a regression.

---

### Task 1: Content-cap knob in `engram query` (TDD)

**Files:**
- Modify: `internal/cli/query.go` (renderItems / payload; flag + env wiring; budget report field)
- Test: `internal/cli/query_test.go` (or the existing query test file)

**Interfaces:**
- Produces: `--content-budget int` flag + `ENGRAM_CONTENT_BUDGET` env read; `renderItems` (or a
  post-pass) snippets chunk Content beyond the budget.

- [ ] **Step 1 — RED.** Test: build `resolved` with 2 notes + 4 chunks (rank order: chunkA, chunkB,
  chunkC, chunkD each with multi-line `content`), call the render path with budget=2. Assert: both
  notes keep full content; chunkA, chunkB full; chunkC, chunkD → snippet (first non-empty line,
  ≤160 chars, ends with `…`). Run → fails (cap not implemented).
- [ ] **Step 2 — GREEN.** Implement: a `capChunkContent(items []queryItem, budget int)` helper
  applied in `renderQueryPayload` after `renderItems`. `budget<=0` → no-op. Walk items in order,
  count chunk items; once chunk-count > budget, replace that chunk's `Content` with
  `snippet(Content)`. Notes skipped. Wire the flag (default 0) and `ENGRAM_CONTENT_BUDGET` (flag
  overrides env when set). Add `content_budget` + `chunks_snippeted` to the budget report.
- [ ] **Step 3 — verify.** `targ test` green; `targ check-full` clean.
- [ ] **Step 4 — REFACTOR + gate B.** DRY/SRP pass on the helper; gate B on the diff.
- [ ] **Step 5 — commit.**

### Task 2: Free cost-curve measurement (no opus)

**Files:** Create `dev/eval/traps/cap_cost_curve.py`

- [ ] Run `engram query` on a fixed 10-phrase set at budgets {unlimited, 60, 30, 15, 8, 4, 2} with
  `ENGRAM_CONTENT_BUDGET`; record payload **bytes** and `items_with_full_content` per budget.
  This is the deterministic, free cost axis — characterize the whole curve here so opus is spent
  only on quality verification.

### Task 3: Quality canary + knee search (opus, economical)

**Files:** Create `dev/eval/traps/cap_quality.py` (wraps the existing c5/c4_idio/c3/c6 warm arms,
injecting `ENGRAM_CONTENT_BUDGET`)

- [ ] **C5 is the canary** (R lives in the recency channel — the first content the cap snippets).
  Run C5 warm at descending budgets, n≥3, binary-search the breakpoint where surfaced/honored drops
  below the warm baseline beyond noise.
- [ ] At the candidate knee from C5, **verify C3 (traps), C4 (c4_idio warm-XXp), C6 (c6_clean warm)**
  hold at n≥3 — these are note-driven, so should be robust to a chunk-only cap, but confirm.
- [ ] If any breaks, step the budget back up to the last passing value. The **knee = smallest budget
  where all four warm arms hold within noise.**

### Task 4: Bake the default + report

- [ ] Set `--content-budget` default to the knee value so every recall benefits without a skill edit
  (the skill calls `engram query` with no flag → default applies). Keep env override.
- [ ] Report: the cost curve (Task 2) + the knee + quality table (Task 3) + the chosen default, as a
  labeled two-axis table (payload tokens & $ saved, with quality held).

## Self-review

- Spec coverage: knob (T1), cost curve (T2), quality knee (T3), wiring + report (T4). ✓
- Risk: if C5 breaks even at a high budget (snippet drops R's marker), the knee is high and savings
  modest — that is a finding to report, not a failure to hide.
- The free cost curve (T2) front-loads the expensive search so opus spend is a few verification
  points, not a linear sweep — honoring "minimum spend for the benefit."
