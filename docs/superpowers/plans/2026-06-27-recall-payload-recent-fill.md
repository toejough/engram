# Lever #1 — compact recall payload via configurable recent-fill

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans. Steps use `- [ ]` checkboxes.

**Goal:** Cut the dominant non-load-bearing bulk of the `engram query` payload — the recency channel (Channel 2), currently the 200 newest chunks (~half of the 426 items / ~230 KB) — to a small configurable default (25), so the agent pages less on recall. **Binary-only** (the recall skill doesn't reference the count). Matched set (the wins) untouched.

**Why this slice:** verified this session — the query payload is **230 KB / ~59K tokens / 426 items**, of which **205 are `provenance: recent`**, which the recall skill itself marks NOT-used-for-coverage-or-synthesis ("situational continuity" only). Cutting it is the biggest, safest reducer. The deeper matched-set lazy-content restructure is **deferred** (higher risk, needs a skill edit) — gated on whether the `$METER` shows this slice suffices.

## Global Constraints
- Go change → **`targ`** for build/test/check (NEVER `go test` directly). `targ check-full` for all errors.
- Follow the existing knob pattern exactly: `ContentBudget` (`internal/cli/query.go:35` flag + `resolveContentBudget` + threaded to the builder). Mirror it for `RecentFill`.
- Engram conventions: name constants (`defaultRecentFill`), wrap errors, line < 120, `t.Parallel()` + no shared state, nilaway/gomega patterns.
- **Standing guardrail:** run the trap gate (no capability regression) + measure with the `$METER` before declaring done. Never touch the win-nucleus (matched-note content).

## Files
- Modify: `internal/cli/query.go` — flag struct (~line 35), `recentFillChunks` const (128 → default), `buildRecentFillItems` call site (1433), add `resolveRecentFill`.
- Test: `internal/cli/query_test.go` (Go, blackbox `package cli_test` where the suite already is) — `resolveRecentFill` defaulting + that the recent channel honors the resolved count.

---

### Task 1: `resolveRecentFill` + the `--recent-fill` flag (TDD)

**Interfaces (mirror `ContentBudget`):**
- Flag on the query args struct: `RecentFill int targ:"flag,name=recent-fill,env=ENGRAM_RECENT_FILL,desc=newest-by-ingest chunks in the recency channel (0=default 25); a smaller payload the agent pages faster"`.
- `defaultRecentFill = 25` (replaces the `recentFillChunks = 200` const).
- `resolveRecentFill(n int) int` — `0 → defaultRecentFill`; `>0 → n`; `<0 → 0` (allow opting the channel out). Mirror `resolveContentBudget`'s shape.

- [ ] **Step 1: RED — write failing tests** in `query_test.go`:
```go
func TestResolveRecentFill_DefaultsAndOverrides(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ResolveRecentFill(0)).To(Equal(25))   // 0 -> baked default
	g.Expect(cli.ResolveRecentFill(5)).To(Equal(5))    // explicit honored
	g.Expect(cli.ResolveRecentFill(-1)).To(Equal(0))   // negative -> channel off
}
```
(If `resolveRecentFill` must stay unexported, test it via a thin exported wrapper or assert the behavior through `RunQuery` output item-count — match how `contentBudget`/`resolveContentBudget` are currently tested; check `query_test.go` first and follow that seam.)

- [ ] **Step 2: Run RED** — `targ test` → FAIL (undefined `ResolveRecentFill`).
- [ ] **Step 3: GREEN — implement.** Replace `recentFillChunks = 200` with `defaultRecentFill = 25`; add `resolveRecentFill`; add the `RecentFill` flag to the args struct; thread it through to the `buildRecentFillItems(..., n)` call at line 1433 as `resolveRecentFill(args.RecentFill)` (plumb via `aggregatedSummary`/`merged` if that's how `contentBudget` flows — follow the same path).
- [ ] **Step 4: Run GREEN** — `targ test` → PASS.
- [ ] **Step 5: Full check** — `targ check-full` → clean (lint + coverage + nilaway).
- [ ] **Step 6: Commit** — `feat(query): configurable --recent-fill (default 25) — shrink the recency-channel payload`

### Task 2: Verify the win is real + no regression

- [ ] **Step 1: Free payload delta.** Run the same 10-phrase `engram query` and `wc -c` the output before vs after (before = `git stash`/prior; after = built binary). Expect a large drop in item count (426 → ~246) and bytes (~230 KB → ~170 KB). Record the numbers.
- [ ] **Step 2: Trap gate (safety, ~$2-3).** `cd dev/eval/traps && python3 gate.py --tier smoke` with the rebuilt `engram` on PATH → **GREEN** (matched set untouched, so the wins must hold). If RED/INCONCLUSIVE, stop — investigate.
- [ ] **Step 3: `$METER` (optional, ~$7).** One warm `matrix.py --models opus --trials 1 --regimes real.full --max-rounds 2` cell; read `recall_s`/`recall_cost` and compare to the 2026-06-26 baseline (recall_s ~117–169s @ recentFill 200). Confirms the paging-time drop. (Flag the spend; the free payload delta is the primary proxy if skipping.)

---

## Self-Review
- Spec coverage: configurable recent-fill (T1) ✓; default 25 ✓; binary-only (no skill edit — skill doesn't reference count) ✓; gate + meter verification (T2) ✓; matched set untouched ✓.
- YAGNI: one knob, mirrors ContentBudget; no new output structure (note-56 territory) needed since we're cutting count, not adding a field.
- Risk: 25 may be too few recent items for continuity → mitigated by configurability + the channel being supplementary; the deeper restructure stays deferred pending the measured delta.
