**Note on task order:** this change documents work already implemented, tested, and verified against real production data before this OpenSpec artifact set was written (time-constrained one-shot execution, per the orchestrator's explicit instruction). Every task below reflects what was actually done, checked accordingly — not a plan yet to execute.

## 1. Reproduce and measure

- [x] 1.1 Measure the real failure on the production route runbook (vault note `1036.2026-09-18.route-dispatch-tier-selection`): confirmed `red_flags` starts at byte 654 (already early per `runbookFrontmatterDoc`'s field order) but the list itself spans to byte 4766 (~4.1KB across 15 entries) — past the ~2KB external truncation point regardless of field position.
- [x] 1.2 Confirm `engram show`'s existing `RunShow` offers no protection for a large note (read `internal/cli/show.go` before this change: raw file body printed unmodified).

## 2. Implement the display-time cap

- [x] 2.1 RED: write `internal/cli/redflags_truncation_internal_test.go` reproducing the failure (an oversized red_flags list loses its newest entry with no cap in place) — confirmed compile failure (`undefined: capRedFlagsForPreview`) before implementation.
- [x] 2.2 GREEN: implement `capRedFlagsForPreview` (`internal/cli/redflags_truncation.go`) — keeps entries from the end of the list within a 1200-byte budget, replaces dropped entries with an explicit marker.
- [x] 2.3 Wire the transform into `internal/cli/query.go`'s candidate content assignment and `internal/cli/show.go`'s `RunShow`.
- [x] 2.4 Add integration-level tests (`query_runbook_test.go`, `show_test.go`) proving the wiring survives an oversized list end-to-end through `RunQuery`/`RunShow`, not just the unit-level helper.

## 3. Verify

- [x] 3.1 `targ test` — all packages green.
- [x] 3.2 `targ check-nils-for-fail` — fixed one real nilaway finding (an unassigned-slice-return flow in `scanRedFlagsEntries`), then clean.
- [x] 3.3 `targ check-coverage-for-fail` — green (84.0% on `internal/cli`).
- [x] 3.4 `targ lint-full` — confirmed pre-existing, unrelated failure (`unknown linters: 'exhaustruct_v5'`, a golangci-lint version/config mismatch) via git-stash bisection: fails identically with or without this change. Not introduced by this fix; out of scope to repair here.
- [x] 3.5 `targ reorder-decls` — applied; also reordered declarations in 3 unrelated pre-existing fixture files the whole-repo check swept up as a side effect (no behavior change, auto-fix per this repo's own `issues.fix = true` convention).
- [x] 3.6 End-to-end verification against the real production vault note via the installed binary (`go install ./cmd/engram`): confirmed the newest `red_flags` entry now lands at byte offset ~1650 (`engram show`) / ~1940 (`engram query`), both comfortably within any 2KB truncation window, with the omission marker visible near the top of the red_flags block in both.

## 4. Close out (reserved for the orchestrator — not done here)

- [x] 4.3 Loop back to route's D8 probe (issue #757) now that both #762 (question-stop scoping) and #763 (this fix) are applied, to see whether route's D8 bar is now met — per Joe's explicit sequencing instruction, this must happen before either #762 or #763 (or #757 itself) is considered closed. — **done**: rerun (`probe_phase2.py --task route --model sonnet5 --n 3 --arms R --shim-only`, $1.43) found 3/3, followed_all 3/3, end_state 3/3 — D8 bar MET, exceeding the skill row's 2/3 reference. Route's `SKILL.md` subsequently retired.
- [x] 4.1 Comment on GitHub issue #763 with the outcome and evidence.
- [x] 4.2 `openspec archive runbook-redflags-truncation-safety` once 4.1 is done.
