## Context

`dev/eval/traps/run.py`'s `build_cold_cfg(dst)` builds a Claude Code config directory with, by its own docstring, "NO CLAUDE.md, NO skills — a true cold/no-memory room." `dev/eval/traps/wrun.py`'s `build_warm_cfg(dst)` is supposed to be the other half of that contrast: call `build_cold_cfg` first, then copy the real `recall`/`learn` skills in, so a WARM trial genuinely has the product's memory surface available and a COLD trial genuinely doesn't. `c4_idio.py`, `c5.py`, and `c6_clean.py` all `from wrun import build_warm_cfg` — every axis of the project's only adversarially-verified capability gate depends on this one function actually doing what its name says.

It stopped doing that on 2026-07-24 (`a0b1ba92`, #697), when the repo restructure moved `skills/` to `agent-instructions/skills/`. `wrun.py`'s copy source was never updated. The miss was silent: `if os.path.isdir(src): shutil.copytree(...)` with no `else`, so the function returned successfully having installed nothing. This went undetected for a month because nothing checked that the WARM config actually contained what it claimed to — the correctness of `build_warm_cfg` was assumed, not verified, and nothing was written down saying it needed to be.

`REPO` (wrun.py:17-18) is actually computed correctly first — `os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))` correctly resolves `dev/eval/traps/../../..` to the repo root — and then immediately overridden by a hardcoded absolute path on the next line for no functional reason. The correct, portable computation already existed; it was just shadowed.

## Goals / Non-Goals

**Goals:**
- Restore `build_warm_cfg` to actually install the real skills (path fix).
- Make it structurally impossible for this exact failure mode to recur silently: a missing or empty skill source must raise, not no-op.
- Prove the fix works by testing runtime behavior, not by re-asserting a path string (the same class of assertion that just broke).
- Get a genuine, current capability-regression signal from `gate.py` for the first time since 2026-06-30, and commit it to `dev/eval/LEDGER.md` so it can't silently go stale again without at least being dated.
- Write down, as a spec, the general principle this bug violated — so it's a documented requirement future changes can be checked against, not a one-off patch.

**Non-Goals:**
- Diagnosing or fixing any genuine product-capability regression the corrected gate might reveal — if `gate.py` still comes back RED after this fix, that's a new, separate finding with its own root-cause cycle (per #732's own scope boundary), not something this change absorbs.
- A periodic/CI trigger to catch this class of drift automatically going forward — worth its own follow-up discussion; this change makes the ONE broken function self-verifying, it doesn't build monitoring infrastructure.
- Auditing every other config-setup step in `dev/eval/` for similar drift (e.g. `build_cold_cfg`'s own assumptions, other harnesses' hardcoded paths) — scoped narrowly to the confirmed-broken function; a broader audit is a reasonable follow-up but not bundled here.
- Retrofitting the OTHER trap harnesses' own risk of hardcoded/stale paths beyond what they inherit automatically by sharing `build_warm_cfg`.

## Decisions

**1. Fix the path by deleting the override, not by hardcoding a new one.** `REPO`'s line-17 computation (`os.path.dirname(__file__)` + three `..`) already resolves correctly to the repo root — delete line 18's hardcoded override entirely, and change only the skill subpath from `"skills"` to `"agent-instructions", "skills"`. Alternative considered: hardcode the corrected absolute path — rejected, since that's the exact same fragility class that caused this bug (a path that silently goes stale the next time something moves), just with a fresher timestamp. Deriving from `__file__` means the function keeps working if the repo is cloned elsewhere or `agent-instructions/` moves again in a way that's still relative to the eval tree's own position — and if it truly can't resolve, Decision 2 below means that fails loud instead of silently.

**2. Fail loud with a specific, diagnosable message.** Replace the bare `if os.path.isdir(src):` with a check that raises (e.g. `RuntimeError`, matching `seed_c3.py`'s existing house pattern of "RAISES on any non-zero exit — fail loud, never silent-pass") naming the exact missing path, when either: (a) `src` doesn't exist, or (b) `src` exists but is empty / has no `SKILL.md` — an empty directory is just as broken as a missing one and deserves the same treatment. Also verify POST-copy that the destination actually landed content (`dst/skills/<name>/SKILL.md` exists) as defense in depth against a silent partial-copy failure, however unlikely. Alternative considered: log a warning instead of raising — rejected; a warning is exactly as invisible as the current silent no-op in an unattended `gate.py --tier full` run nobody is watching live, which is precisely the condition under which this bug went unnoticed for a month.

**3. Test runtime behavior, not a path string.** The regression test calls `build_warm_cfg` against a fresh temp directory and asserts `SKILL.md` actually exists under the copied `recall`/`learn` skill directories — it does NOT hardcode `agent-instructions/skills/recall` as an expected string to assert against source layout. This means the test keeps proving the real thing (the function does what it claims) even if `agent-instructions/` itself moves again someday; it would need Decision 2's raise to fire in that case, and a test asserting on the RAISE's occurrence (via a deliberately-broken monkeypatched source) covers that path explicitly. New file: `dev/eval/traps/test_wrun.py` (no such file exists today; `wrun.py` currently has zero direct test coverage despite being the shared foundation of all four trap axes).

**4. Document the general principle as a spec, scoped by its own violation.** The new `eval-warm-config-fidelity` capability states the requirement generally enough to cover "any product surface a WARM trial config claims to grant" (not just skills specifically), because the failure MODE (an eval harness assumes a config-setup step succeeded instead of verifying it) is the reusable lesson, even though today's concrete fix only touches skills. This is a deliberate, narrow extension of spec coverage into `dev/eval/` — precedent decision: the eval harness underpins the project's only adversarially-verified capability claims (per `dev/eval/traps/README.md`), so its own setup correctness is at least as load-bearing as an ordinary product capability, and deserves the same documented-requirement treatment `openspec/specs/` already gives the CLI/vault behavior it measures.

## Risks / Trade-offs

- **[Risk]** Fixing the harness may reveal a genuine, separate product-capability regression once it's actually testing the real thing again → **Mitigation:** explicitly a Non-Goal; tasks.md re-runs the gate and reports honestly whatever it finds, but a RED result there does not block or expand this change — it spins off its own issue, per #732's own scope boundary.
- **[Risk]** A runtime-behavior test that spins up a real temp Claude Code config could be slow or brittle in CI/sandboxed environments → **Mitigation:** the test only needs `build_warm_cfg`'s filesystem side effects (it doesn't need to invoke `claude -p` or spend any money) — copying two small skill directories into a temp dir is fast and has no external dependency beyond the repo's own filesystem.
- **[Risk]** Extending `openspec/specs/` coverage to internal eval tooling is a departure from this repo's existing all-product-facing spec categories, and could read as scope inflation → **Mitigation:** deliberately named and scoped narrowly (one function's contract, not a general eval-framework spec); the design's own rationale for the precedent is stated plainly above rather than assumed.

## Migration Plan

No data migration. This is a two-line functional fix plus a new test file plus a spec addition — purely additive, no existing behavior for any OTHER function changes. Rollback is a plain revert of the commit; no state to unwind (the fixed function is idempotent and touches only ephemeral trial temp directories, never the real vault or real `~/.claude`).

## Open Questions

- Should a periodic/CI trigger be added so this class of drift is caught within days instead of relying on someone manually running `gate.py` again? Deliberately deferred (Non-Goals) — worth its own follow-up issue once this fix lands and a fresh baseline exists to protect.
- Should `build_cold_cfg` or other harness setup functions get the same fail-loud treatment preemptively, even without a confirmed bug in them today? Deferred — this change fixes the confirmed defect; a broader audit is reasonable future work, not bundled here to keep the change reviewable and its acceptance criteria falsifiable.
