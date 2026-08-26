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

- [ ] 4.1 `go install ./cmd/engram` from repo root, so the gate exercises the current binary
- [ ] 4.2 Run `python3 gate.py --tier smoke` from `dev/eval/traps/`. Before trusting the pass/fail verdict, inspect ONE fresh trial's `warm-cfg/skills/recall/SKILL.md` directly on disk (or via the transcript, confirm the agent's Skill tool listing includes `recall`) — this is the structural proof the fix actually worked in a live run, not just in the unit tests
- [ ] 4.3 Report the smoke result honestly. If RED: this is now evidence of a genuine, separate product-capability issue (not the harness bug) — per proposal.md's Non-Goals, do NOT attempt to fix it as part of this change; file it as a new issue with the smoke output attached, and stop this task group there (do not proceed to full-tier or LEDGER update with a false GREEN)
- [ ] 4.4 If smoke is GREEN (or the documented single-cell C5b flake only, per the harness's own known-flake exception): run `python3 gate.py --tier full` (~$15-18, the originally-verified bars at 5 reps/axis)
- [ ] 4.5 Update `dev/eval/LEDGER.md` with the fresh result and today's date, replacing/annotating the stale 2026-06-30-era citations as superseded by this re-verification

## 5. Close-out

- [ ] 5.1 Comment on and close #732, linking this change and the fresh `gate.py --tier full` result (or the new spun-off issue, if 4.3's honest-RED path was taken — in that case #732 is still closed, since ITS scope — the harness bug — is fixed; the new issue tracks the separately-discovered product finding)
- [ ] 5.2 `targ check-full` clean, all tests green
- [ ] 5.3 `/opsx:archive` to archive this change and sync `openspec/specs/`
