# Lever #5-early — configurable recent-fill to shrink the recall payload

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans. Steps use `- [ ]` checkboxes.

**Scope honesty (per Gate A):** this delivers roadmap **#5's** recency-channel cut, NOT roadmap **#1**
(the clusters-first / lazy-content restructure of the *matched* set). It is done first because the
verification showed the recency channel is the **biggest single, safest** payload reducer. **Roadmap
#1 remains explicitly open** as the next step — shipping this does NOT close #1; the matched-set
restructure is a separate, larger win to take up regardless of this slice's measured delta.

**Goal:** Make `recentFillChunks` (the 200 newest chunks appended as Channel 2 — verified at **205 of
426 items / ~230 KB**, which the recall skill itself marks non-load-bearing "situational continuity")
a configurable flag defaulting to **25**. Binary-only; matched set (clusters / candidate_l2s / notes)
provably untouched (Gate A confirmed `buildRecentFillItems` only builds the recency channel → trap
gate must stay GREEN).

## Global Constraints
- Go change → **`targ`** (`targ test`, `targ check-full`) — NEVER `go test`.
- Engram stack: blackbox `package cli_test`; **unexported `resolveRecentFill` tested via an
  `ExportResolveRecentFill` wrapper in `export_test.go`** (the established pattern — `resolveContentBudget`
  is unexported, exposed as `ExportResolveContentBudget`, tested in `query_cap_test.go`). gomega +
  nilaway patterns (`.claude/rules/go.md`); `t.Parallel()`; named constants; line < 120.
- **Standing guardrail:** trap gate before/after (no capability regression) + `$METER` to measure the
  win. Never touch the win-nucleus.

## Files
- Modify: `internal/cli/query.go` — args/flag struct (~26-40, add `RecentFill`), the `recentFillChunks`
  const (the `= 200` constant defined at line 128 → `defaultRecentFill = 25`), add `resolveRecentFill`,
  and the **inline** call at line 1433 (Gate A BLOCKER #1: `merged` does not exist yet at 1433, so do
  NOT thread through `aggregatedSummary` — replace `recentFillChunks` directly with
  `resolveRecentFill(args.RecentFill)`).
- Modify: `internal/cli/export_test.go` — add `ExportResolveRecentFill`.
- Test: `internal/cli/query_cap_test.go` — extend (mirrors `ExportResolveContentBudget` tests).
- Modify: `internal/cli/query_cluster_phase2_test.go` — 7 stale `recentFillChunks (200)` comments
  (lines 6, 9, 30, 49, 80, 101, 108) → `defaultRecentFill (25)`.
- Modify (Step 5 doc scrub): `docs/architecture/{c1-system-context.md (line 99 + seq diagram ~145),
  c2-containers.md (C2 entry + seq note), c3-components.md (K8 entry)}` — the "200 newest chunks" /
  `recentFillChunks=200` references → the configurable default (25), document the *mechanism* not the
  bare number.

---

### Task 1: `resolveRecentFill` + `--recent-fill` flag (TDD)

**Interfaces (mirror `ContentBudget` exactly):**
- Flag: `RecentFill int targ:"flag,name=recent-fill,env=ENGRAM_RECENT_FILL,desc=newest-by-ingest chunks in the recency channel (0=default 25); a smaller payload the agent pages faster"`.
- `defaultRecentFill = 25` (replaces `recentFillChunks = 200`).
- `resolveRecentFill(n int) int` — unexported: `0 → defaultRecentFill`; `>0 → n`; `<0 → 0` (channel off). Same shape as `resolveContentBudget`.
- `ExportResolveRecentFill(raw int) int { return resolveRecentFill(raw) }` in `export_test.go`.

- [ ] **Step 1: RED** — in `query_cap_test.go` (next to the ExportResolveContentBudget tests):
```go
func TestResolveRecentFill_DefaultsAndOverrides(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(cli.ExportResolveRecentFill(0)).To(Equal(25))   // 0 -> baked default
	g.Expect(cli.ExportResolveRecentFill(5)).To(Equal(5))    // explicit honored
	g.Expect(cli.ExportResolveRecentFill(-1)).To(Equal(0))   // negative -> channel off
}
```
- [ ] **Step 2: Run RED** — `targ test` → FAIL (undefined `ExportResolveRecentFill`).
- [ ] **Step 3: GREEN** — implement: replace `recentFillChunks = 200` with `defaultRecentFill = 25`; add `resolveRecentFill`; add the `RecentFill` flag to the args struct; at line 1433 replace `recentFillChunks` with `resolveRecentFill(args.RecentFill)` (inline — no struct threading); add `ExportResolveRecentFill` to `export_test.go`.
- [ ] **Step 4: Run GREEN** — `targ test` → PASS.
- [ ] **Step 5: Update the 7 stale comments** in `query_cluster_phase2_test.go` (200 → 25).
- [ ] **Step 6: Full check** — `targ check-full` → clean (lint + coverage + nilaway).
- [ ] **Step 7: Commit** — `feat(query): configurable --recent-fill (default 25) — shrink the recency-channel payload`

### Task 2: Verify the win + no regression (and keep #1 open)

- [ ] **Step 1: Free payload delta.** Run the exact baseline 10-phrase query before vs after and `wc -c` + count items. **Baseline phrases (verbatim, for comparability):** `building a command-line app in Go conventions` · `error handling and wrapping in Go` · `engram recall cost and payload` · `memory eval trap results` · `crowded vault generalization` · `cost meter recall_cost schema` · `writing plans TDD` · `git commit conventions` · `skill editing writing-skills` · `verified benefit ledger`. Baseline = 230 KB / 426 items / 205 recent. Expect ~170 KB / ~246 items / ~25 recent. Record the delta.
- [ ] **Step 2: Trap gate (safety, ~$2-3).** `cd dev/eval/traps && python3 gate.py --tier smoke` with the rebuilt `engram` on PATH → **GREEN** (matched set untouched). RED/INCONCLUSIVE → stop.
- [ ] **Step 3: `$METER` (~$7, recommended).** One warm `matrix.py --models opus --trials 1 --regimes real.full --max-rounds 2` cell; read `recall_s`/`recall_cost`; compare to the 2026-06-26 baseline (recall_s ~117–169s @ recentFill 200). This quantifies the actual paging-time win.
- [ ] **Step 4: Decision (keep #1 honest).** Record the measured delta. **Regardless of the number, roadmap #1 (matched-set clusters-first/lazy-content restructure) stays the next item** — this slice is the cheap safe reducer, not a substitute for the structural restructure. Update ROADMAP: this is #5-done (or #5-partial); #1 unchanged/open.

---

## Self-Review (vs Gate A)
- ask: relabeled #5-early, #1 explicitly open + not gated away (Task 2 Step 4) ✓; scope front-and-center ✓.
- code: inline replace at 1433 (no aggregatedSummary) ✓; unexported + ExportResolveRecentFill wrapper, tested via cli.ExportResolveRecentFill ✓; 7 stale comments fixed ✓; matched-set isolation confirmed ✓; no eval consumer of 200 ✓; targ ✓.
- docs: C4 diagram scrub (c1/c2/c3) in Step 5 ✓; skill needs no edit (no count reference) ✓; naming mirrors content-budget ✓.
- clarity: const-vs-line-number disambiguated ("the `= 200` constant at line 128") ✓; threading resolved (inline) ✓; baseline phrases embedded ✓.
