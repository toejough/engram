# Lever 4 — hard-requirement (checklist) handoff: cut build rounds

**Goal:** Test whether turning recalled conventions into a **gating self-verification checklist** (the
build must verify its code against each before finishing) cuts **feedback rounds-to-converge** vs the
current soft handoff ("apply them as requirements"). Rounds are where the build's tokens/$ go (note 77:
the build is the cost), so fewer rounds = lower C1/C2.

## Key design calls (from orientation)

- **Measure rounds-to-converge primary**, not $ — it's an integer count the lever directly targets and
  is far less noisy than $ (which has been un-pin-downable all session). Pass-rate (arch score) and
  $/op are secondary.
- **Run on sonnet, not opus.** Opus already one-shots most builds (run-8 cold conv `[14,3,0,1,1]` →
  rounds≈1 on saturated tasks), so it has almost no rounds to cut — little headroom. Sonnet takes 2-3
  rounds and shows clean memory front-loading, so it's where the lever can move the metric AND it's
  ~4× cheaper. Note in the writeup: opus headroom is small by saturation; sonnet is the proof-of-concept.
- **A/B is warm-soft vs warm-checklist** (cold is the known baseline, not re-run).

## The change (harness)

`dev/eval/cumulative/harness.py`:
- `build_prompt(app, interface, read_mode)`: add a third `read_mode == "skill-checklist"` that emits
  the same recall directive PLUS a gating block: *"Before you finish, write out every convention and
  decision the recall surfaced as an explicit checklist, and verify your code satisfies EACH one. Fix
  any miss in THIS pass. Do not declare done until every checklist item passes AND `go test ./...`
  passes."*
- `REGIMES`: add `"real.checklist": {"write": "skill", "read_mode": "skill-checklist"}`.

`dev/eval/cumulative/matrix.py`: add `"real.checklist"` to `REAL_REGIMES` (so `--regimes
real.full,real.checklist` selects the A/B pair).

**TDD (light, it's a prompt):** unit-assert `build_prompt(app, iface, "skill-checklist")` contains the
gating self-verification language AND the recall directive; `"skill"` does not contain the gating
language (the contrast). Run `python3 -c` assertion or a small pytest-style check. Gate B on the diff.

## Run the A/B

`CUMMATRIX_ROOT=/tmp/l4-sonnet python3 matrix.py --models sonnet --trials 1,2,3,4,5 --regimes
real.full,real.checklist --workers 20`

Per regime, from the per-op JSON, extract per app: **rounds-to-converge** (the round index where the
arch score first reaches the convergence threshold, or max_rounds if never), final arch score
(pass-rate), $/op. Estimated cost: ~$25-35 (sonnet, 2 regimes × 3 apps × 5 trials).

## Verdict rule (locked, noise-aware)

- **VIABLE** if mean rounds-to-converge (checklist) < soft by more than the warm-vs-warm noise floor
  (size it from the soft regime's own trial spread), **AND** pass-rate (final arch score) ≥ soft
  within noise (no quality regression). 
- **NOT VIABLE / backfires** if checklist rounds ≥ soft, OR pass-rate drops (over-constraining caused
  churn — the named risk). Report rounds AND pass-rate together; a rounds win with a pass-rate drop is
  not a win.
- Report **both axes** ($ and rounds/time); state the noise floor explicitly.

## Caveats to report

- Sonnet proof-of-concept; opus headroom is small (saturation) — if sonnet shows the effect, an opus
  confirmation is optional and likely muted.
- The current handoff is already "apply as requirements" — the lever's delta is the *gating
  self-verification*, a narrow change; the effect may be small.
- $ is secondary and noisy; the rounds count is the trustworthy signal.
