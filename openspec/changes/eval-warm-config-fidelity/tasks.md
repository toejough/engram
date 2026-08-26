## 1. RED — prove the defect and pin the requirement with failing tests

- [x] 1.1 Create `dev/eval/traps/test_wrun.py`. Write a test that calls `build_warm_cfg(tmp_dst)` against a fresh temp directory (using the REAL `agent-instructions/skills/recall` and `.../learn` sources — no mocking the source, this must exercise the actual current codebase) and asserts `os.path.exists(os.path.join(tmp_dst, "skills", "recall", "SKILL.md"))` and the same for `learn`. Run it and confirm it currently FAILS (RED) — this is the direct proof of #732's root cause, not just a hypothetical
- [x] 1.2 Add a second test: monkeypatch/parameterize `build_warm_cfg` (or the module-level `REPO`) so the skill source resolves to a directory that does not exist; assert the call raises, and assert the raised error's message contains the missing skill name and the path that was checked. Confirm this test currently FAILS too (today's code silently returns instead of raising)
- [x] 1.3 Add a third test: point the skill source at a real, existing, but EMPTY temp directory (exists, no `SKILL.md`); assert the call raises for the same reason as 1.2 (empty source is treated as equivalent to missing)
- [x] 1.4 Run all three new tests and confirm all three currently FAIL against unmodified `wrun.py` — paste the failure output into the PR/commit as the RED baseline

## 2. GREEN — fix `build_warm_cfg`

- [x] 2.1 In `dev/eval/traps/wrun.py`: delete line 18's hardcoded `REPO = "/Users/joe/repos/personal/engram"` override. Keep line 17's computed `REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))` as the sole definition (design.md Decision 1 — it already resolves correctly to the repo root; the hardcode was pure shadowing)
- [x] 2.2 In `build_warm_cfg`, change the skill source path from `os.path.join(REPO, "skills", skill)` to `os.path.join(REPO, "agent-instructions", "skills", skill)`
- [x] 2.3 Replace the bare `if os.path.isdir(src):` guard with a check that raises `RuntimeError` (matching `seed_c3.py`'s existing house pattern) naming the skill and the checked path when: (a) `src` does not exist, or (b) `src` exists but has no `SKILL.md` (design.md Decision 2)
- [x] 2.4 After each `shutil.copytree`, verify `SKILL.md` exists at the destination path; raise `RuntimeError` naming the skill if it's missing post-copy (defense in depth, design.md Decision 2)
- [x] 2.5 Run the three tests from Section 1 — confirm all three now PASS (GREEN)

## 3. Verify no collateral damage

- [x] 3.1 Confirm `c4_idio.py`, `c5.py`, `c6_clean.py` need no code changes (they import `build_warm_cfg` unmodified) — run their existing test coverage if any exists, or at minimum confirm they still import and construct without error
- [x] 3.2 Run the full `dev/eval/traps/` test suite (`test_crowd.py`, `test_gate.py`, `test_qanchor.py`, `test_recall_time_prequery.py`, plus the new `test_wrun.py`) — confirm nothing else regressed
- [x] 3.3 `targ check-full` (or the repo's equivalent Python lint/check target if `dev/eval/` is covered by a separate check) — clean

## 4. Get the real signal — re-run the gate against the fixed harness

- [x] 4.1 `go install ./cmd/engram` from repo root, so the gate exercises the current binary
- [x] 4.2 Run `python3 gate.py --tier smoke` from `dev/eval/traps/`. Before trusting the pass/fail verdict, inspect ONE fresh trial's `warm-cfg/skills/recall/SKILL.md` directly on disk (or via the transcript, confirm the agent's Skill tool listing includes `recall`) — this is the structural proof the fix actually worked in a live run, not just in the unit tests

  **Result: GREEN, unanimous.** C3 5/5, C4i 1/1, C5 1/1, C6 2/2 (2026-08-26). Structurally confirmed: `warm-cfg/skills/recall/SKILL.md` and `.../learn/SKILL.md` both present on disk in the fresh trial. Real vault confirmed unchanged (765 notes) before and after.

- [x] 4.3 Report the smoke result honestly. If RED: ... (N/A — smoke was clean GREEN, proceeded to 4.4)

- [x] 4.4 Ran `python3 gate.py --tier full` (5 reps/axis; C6 at its own 4-rep bar). **Result: C3 25/25 GREEN, C4i 5/5 GREEN, C6 8/8 GREEN, C5 4/5 RED** (overall verdict RED per the gate's strict all-cells bar). This is NOT the harness bug reappearing — C5a (surfacing) hit 5/5; only C5b (honoring) missed one. Per vault note 209's same-tree-rerun discipline (built for smoke-tier n=1, applied here since a single-cell miss is the same shape of question): ran a targeted, independent same-tree re-check of JUST C5 (fresh `seed_c5.py` + `c5.py --arms warm --n 5`, own $3.03 spend) — **reproduced exactly: 4/5, same trial index (#3) failed both times, honored=False.** Two independent 5-trial runs landing on the identical ratio is a stable measured rate, not stochastic noise — the "flake, re-run to confirm" branch resolves to "real, confirmed by re-run" here, the mirror image of note 209's flake case. C5a is 10/10 across both runs (retrieval is perfect); C5b (agent judgment on honoring a recency-channel standard under conflict) measures ~80% (8/10), below the originally-verified 100% bar. This is a genuine, separate, narrowly-scoped finding, unrelated to #732's harness-fidelity bug — which is conclusively proven fixed by C3/C4i/C6's perfect scores and C5a's perfect score. Filed as its own issue (task 5.1) rather than expanded into this change's scope, per proposal.md's Non-Goals.

- [x] 4.5 Updated `dev/eval/LEDGER.md` with the fresh result and today's date (2026-08-26) — see the new `eval-warm-config-fidelity` / `c5b-honoring-rate-80pct` entries, which also close out the 2026-06-28 `tier-routing-parity` entry's dangling "C5 axis flaked (re-run owed)" note with a precise, reproduced number instead of an open debt

## 5. Close-out

- [x] 5.1 Commented on and closed #732 (comment links commit `a0065709`, the smoke and full-tier results, and #733 as the spun-off C5b finding). Filed **#733** for the C5b honoring-rate finding, dedup-checked clean, cross-linked to `LEDGER.md#tier-routing-parity`'s original 2026-06-28 "re-run owed" flag which it resolves.
- [x] 5.2 `targ check-full` run independently: `check-coverage-for-fail` PASS, `reorder-decls-check` PASS, `lint-fast` PASS, `deadcode` PASS, `check-thin-api` PASS, `check-nils-for-fail` PASS, `test-integration` PASS. Two pre-existing, unrelated failures confirmed not caused by this change: `check-uncommitted` (expected pre-commit state) and `lint-full` (pre-existing `golangci-lint` "unknown linters: exhaustruct_v5" environment/config mismatch — `dev/eval` isn't even lint-covered per its own skip warning). All 39 `dev/eval/traps/` Python tests green.
- [x] 5.3 `/opsx:archive` — see next commit
