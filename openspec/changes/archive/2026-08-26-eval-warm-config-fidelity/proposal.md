## Why

`dev/eval/traps/wrun.py`'s `build_warm_cfg()` copies the `recall`/`learn` skills into each WARM trial's isolated Claude Code config from a hardcoded path (`REPO/skills/<name>`) that stopped existing on 2026-07-24 (`a0b1ba92`, #697 moved `skills/` → `agent-instructions/skills/`). The copy is guarded by a bare `if os.path.isdir(src):` with no `else` branch — the miss is silent. Every WARM trial across all four verified capability axes (C3/C4i/C5/C6 — `c4_idio.py`, `c5.py`, `c6_clean.py` all share this function) has run in a config with **zero skills installed** for over a month, discovered only when `gate.py --tier smoke` came back RED during unrelated work and a transcript-level investigation (#732) found the agent's own words: *"I don't see a `/recall` skill in my available skills list... I'll proceed with the task cleanly."*

Nothing in the codebase or docs stated a requirement that a WARM trial's config must actually, verifiably carry the real installed skill surface it claims to grant — so a routine repo restructure silently invalidated the project's only adversarially-verified capability signal, and no test or check caught it. The bug itself (a two-line path fix) is trivial; the absence of a documented, enforced requirement is what let it stand undetected for a month. This change writes that requirement down as a spec and makes it self-enforcing, in addition to fixing the immediate defect.

## What Changes

- New requirement: a WARM trial's Claude Code config SHALL be verified — structurally, by an automated check, not by inspection or trust — to contain the real recall/learn skill sources before any trial runs against it. If the source doesn't exist or is empty, config construction SHALL fail loudly (raise), never silently proceed.
- Fix `dev/eval/traps/wrun.py`'s `build_warm_cfg()`: correct the skill-source path to the real location (`agent-instructions/skills/<name>`), remove the hardcoded absolute `REPO` path in favor of a path derived from the file's own location, and replace the silent `if os.path.isdir(src):` no-op with a loud failure when the source is missing or empty.
- Add a regression test that asserts this requirement holds — the kind of test that would have caught #697's silent breakage immediately instead of a month later.
- Re-run `gate.py` (smoke, then full) against the corrected harness to get the first genuine capability-regression signal since 2026-06-30, and record the result in `dev/eval/LEDGER.md` with a date.
- Closes #732.

## Capabilities

### New Capabilities
- `eval-warm-config-fidelity`: the requirement that any eval harness config built to represent a "warm" (memory-equipped) trial condition must be structurally verified to actually carry the real, currently-installed product surface (skills) it claims to grant, and must fail loudly rather than silently degrade to a cold condition when that surface is missing.

### Modified Capabilities
(none — `dev/eval/` has no prior spec coverage; this is the first capability documented for the eval harness itself, not a change to any existing product-facing capability)

## Impact

- `dev/eval/traps/wrun.py` — `build_warm_cfg()` (the function at fault) and its `REPO` constant.
- `dev/eval/traps/c4_idio.py`, `dev/eval/traps/c5.py`, `dev/eval/traps/c6_clean.py` — no code changes expected (they import `build_warm_cfg` unmodified), but they inherit the fix and the new verification automatically since they share the function.
- New test file (or an addition to an existing `dev/eval/traps/test_*.py`) asserting the fidelity requirement.
- `dev/eval/LEDGER.md` — a fresh, dated GREEN (or honestly-reported RED, if the fix reveals a genuine separate regression) entry, replacing the stale 2026-06-30-era record as the citable current status.
- Issue #732 — closed by this change once the fix lands and the gate is re-verified.
