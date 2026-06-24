# Option 1 — content-cap implementation + min-spend sweep

> **For agentic workers:** TDD each unit (RED → GREEN → REFACTOR). Steps use `- [ ]`.

**Goal:** Add a content-budget cap to `engram query` that trims chunk bulk from the recall payload,
then binary-search the cap down to the **knee** — the smallest cap that still preserves quality on
the currently-passing tests (Go units + C3/C4/C5/C6 warm eval arms) — and **report** the min-spend
point. Baking the knee as the shipped default is a SEPARATE step, gated on the user's sign-off.

**Architecture:** A single knob caps how many **chunk** items (in existing rank order) render with
full content; lower-ranked chunks (the ~200-chunk recency channel ranks LAST, so it trims first) get
a one-line snippet. **Note items are never capped** — crystallized lessons are the gold and few; the
cap helper gates strictly on `item.Kind == chunkItemKind`. The knob is a single `QueryArgs` field
wired through targ's existing `env=` tag — `ContentBudget int` with
`targ:"flag,name=content-budget,env=ENGRAM_CONTENT_BUDGET,..."`. targ already resolves
flag-overrides-env, so there is **no hand-rolled precedence logic and no `os.Getenv` in `query.go`**
(that would violate the no-`os.*`-in-`internal/` DI rule). `0` = unlimited (current behavior). The
`env=` path lets the sweep inject the cap into the *real* recall skill (which calls `engram query`
with no flag) without a per-value skill edit — the reason the knob is env-reachable at all. Query
stays **read-only** (note 51): the cap is a pure post-pass over the rendered items, no I/O.

**Tech stack:** Go (`internal/cli/query.go`), imptest/rapid/gomega, Python eval harnesses, `targ`.

## Global Constraints

- Query stays read-only; cap helper is a pure function, no `os.*`/I/O in `internal/` (note 51, DI rule).
- `ContentBudget` wired via targ `env=` tag (one field) — not `IntVar`, not `os.Getenv`.
- **Default `content-budget = 0` (unlimited) — behavior unchanged until the knee is chosen AND the
  user approves baking it.** The existing Go query tests MUST stay green at budget=0 (the control).
- `targ check-full` clean on first commit; nilaway/gomega patterns per `.claude/rules/go.md`.
- Cost claims are **two-axis**: payload **tokens** (proxy: `chars/4`, stated as a proxy) AND $.
- A "quality holds" call must clear the **noise floor** sized from the SAME warm-vs-warm contrast at
  full budget, n≥3. A one-trial dip is not a regression (gap-below-noise lesson).
- Results reported as a **labeled table with units** (always-table lesson) — template in Task 4.

---

### Task 1: Content-cap knob in `engram query` (TDD)

**Files:**
- Modify: `internal/cli/query.go` (QueryArgs field; cap helper in `renderQueryPayload`; budget report)
- Test: the existing query test file (`internal/cli/query_test.go` or equivalent)

**Snippet algorithm (locked):** `snippet(s)` = collapse all whitespace/newlines in `s` to single
spaces, trim, truncate to ≤160 runes on a rune boundary, append `…` iff truncation occurred. Pure,
deterministic, no line-vs-char ambiguity.

**Interfaces:**
- Produces: `ContentBudget int` on `QueryArgs` (targ flag+env); `capChunkContent(items []queryItem,
  budget int) (capped []queryItem, snipped int)` applied in `renderQueryPayload` after `renderItems`.

- [ ] **Step 1 — RED.** Test: `resolved` = 2 notes + 4 chunks (rank order chunkA..D, each multi-line
  `content`), budget=2. Assert: both notes full; chunkA, chunkB full; chunkC, chunkD → `snippet(...)`
  (single line, ≤160 runes, trailing `…`); `chunks_snippeted == 2`; `ItemsWithFullContent` counts
  only un-snippeted items. Run → fails.
- [ ] **Step 2 — GREEN.** Add `ContentBudget` field with the targ `env=` tag. Implement
  `capChunkContent`: walk items in order, count `Kind == chunkItemKind`; once chunk-count > budget
  (and budget > 0), replace that chunk's `Content` with `snippet(Content)`. Notes untouched. Call it
  in `renderQueryPayload`; redefine `ItemsWithFullContent` to count items whose content was NOT
  snippeted; add `content_budget` (echo) + `chunks_snippeted` to `queryBudget`.
- [ ] **Step 3 — verify + control gate.** `targ test` green INCLUDING the pre-existing query tests at
  budget=0 (no-op control — proves unchanged baseline, not assumed). `targ check-full` clean.
- [ ] **Step 4 — REFACTOR + gate B.** DRY/SRP pass; gate B on the diff.
- [ ] **Step 5 — commit.**

### Task 2: Free cost curve (no opus) — the deterministic cost axis

**Files:** Create `dev/eval/traps/cap_cost_curve.py`

**Fixed phrase set (constant across all budgets):** the exact 10 phrases from this session's O1
recall (reproduced verbatim in the script as a constant) so every budget queries identically.

- [ ] Run `engram query` on that fixed 10-phrase set at budgets {0, 60, 30, 15, 8, 4, 2} with
  `ENGRAM_CONTENT_BUDGET`. Per budget record: payload **wire bytes**, **token proxy = total content
  chars / 4** (labeled a proxy), `items_with_full_content`, `chunks_snippeted`. Emit the curve table.
  This characterizes the whole cost axis for free, so opus is spent only on quality verification.

### Task 3: Quality canary + knee search (opus, economical)

**Files:** Create `dev/eval/traps/cap_quality.py` (wraps the existing c5/c4_idio/c3/c6 warm arms,
injecting `ENGRAM_CONTENT_BUDGET` into the agent subprocess env).

- [ ] **Step 0 — baseline + noise floor.** Run C5 warm at budget=0, n≥3. Record mean and SE of
  surfaced/honored. This warm-vs-warm SE IS the noise floor for the search (matched contrast).
- [ ] **Step 1 — C5 canary search.** C5 warm at descending budgets, n≥3 each. **Breakpoint rule
  (locked):** the smallest budget is BROKEN when its mean surfaced/honored < (baseline mean − 1.5·SE),
  i.e. a drop exceeding the warm-vs-warm noise floor; otherwise it HOLDS. Binary-search between the
  last HOLD and first BREAK. (C5 chosen because R lives in the recency channel — the first content the
  cap snippets — so it is where the cut bites first.)
- [ ] **Step 2 — confirm at the knee.** At the smallest HOLDING budget, run C3 (traps), C4
  (c4_idio warm-XXp), C6 (c6_clean warm), n≥3 each; each must hold ≥ its baseline within the same
  noise discipline. If any breaks, step the budget up to the next holding value and re-confirm.
- [ ] **Knee = smallest budget where ALL four warm arms hold within noise.** If C5 breaks even at a
  high budget (snippet drops R's marker), the knee is high / savings modest — report that as the
  finding (the "sensible lower bound" escape), not a failure to hide.

### Task 4: Report (bake is a separate, user-gated step) — DONE

> Results in `docs/design/2026-06-24-engram-cost-reduction-options.md` (Option 1 — BUILT & measured) and the EXPERIMENT-LOG. Knee = budget 8 (−63%); recommend 15–30 default; bake pending user sign-off.

- [ ] Produce the **labeled result table** (template below). Do NOT change the shipped default.
- [ ] Recommend the knee as the default; ask the user to approve baking it (then a one-line change of
  the `ContentBudget` default + re-verify). Baking is out of scope until approved.

**Result table template:**

| budget | payload tokens (proxy) | Δ tokens vs 0 | est. $/recall | C5 surf/hon (n) | C3 (n) | C4 (n) | C6 (n) | verdict |
|--------|------------------------|---------------|---------------|-----------------|--------|--------|--------|---------|
| 0 (baseline) | … | — | … | 5/5 | 25/25 | 5/5 | 8/8 | baseline |
| … | … | −…% | … | … | … | … | … | HOLD/BREAK |
| **knee** | … | −…% | … | … | … | … | … | **min spend, quality held** |

(Token = proxy chars/4; $ from measured opus runs where available, else proxied from token Δ and
labeled as such. Cells are measured at n≥3; HOLD/BREAK per the Task 3 noise rule.)

## Self-review

- Spec coverage: knob (T1), free cost curve (T2), quality knee with stated noise floor + breakpoint
  formula (T3), report + user-gated bake (T4). ✓
- Control: existing query tests green at budget=0 asserted (T1 Step 3), not assumed.
- Risk: a high knee (C5 fragile to snippeting) is a reportable finding, not a hidden failure.
- Min-spend honored on both meters: free cost curve front-loads the search; opus spent only on a few
  C5 verification points + one knee confirmation.
