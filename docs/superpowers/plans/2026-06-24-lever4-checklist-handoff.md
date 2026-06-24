# Lever 4 — hard-requirement (checklist) handoff: cut build rounds

**Goal:** Test whether turning recalled conventions into a **gating self-verification checklist** (the
build must verify its code against each before finishing) cuts **feedback rounds-to-converge** vs the
current soft handoff ("apply them as requirements"). Rounds are where the build's tokens/$ go (note 77:
the build is the cost), so fewer rounds = lower C1/C2.

## Key design calls (from orientation)

- **Measure rounds-to-converge primary**, not $ — it's an integer count the lever directly targets and
  is far less noisy than $ (which has been un-pin-downable all session). Pass-rate (arch score) and
  $/op are secondary.
- **Run on sonnet, not opus — for MEASURABILITY, not cost.** A lever that cuts *rounds* cannot be
  measured on an arm that already converges in ~1 round. Opus one-shots most builds (run-8 cold conv
  `[14,3,0,1,1]` → rounds≈1 on saturated tasks) — no rounds to cut. Sonnet takes 2-3 rounds (run-7) and
  shows clean memory front-loading, so the metric can actually move there. (Cost is roughly comparable
  between the two on these chains — do NOT claim sonnet is "cheaper"; the decisive reason is headroom,
  i.e. test-where-it-bites.) **The opus payoff therefore remains formally OPEN:** a positive sonnet
  result proves the *mechanism* works, not that it helps the saturated production-opus chain.
- **A/B is warm-soft vs warm-checklist** (cold is the known baseline, not re-run).

## The change (harness)

`dev/eval/cumulative/harness.py`:
- **Carry the checklist as a FLAG, keep `read_mode == "skill"`** (CRITICAL — the recall-fired
  enforcement + crystallization gates at harness.py:745,746,785,797 match the literal string `"skill"`;
  a new read_mode would silently bypass the recall-validity guard and break the A/B premise that both
  arms recalled). So: `build_prompt(app, interface, read_mode, checklist=False)` — when `checklist`,
  append a gating block after the recall directive: *"Before you finish, write out every convention and
  decision the recall surfaced as an explicit checklist, and verify your code satisfies EACH one. Fix
  any miss in THIS pass. Do not declare done until every checklist item passes AND `go test ./...`
  passes."* The build op reads `regime.get("checklist", False)` and passes it through.
- `REGIMES`: add `"real.checklist": {"write": "skill", "read_mode": "skill", "checklist": True}` — same
  read_mode (all recall enforcement applies), flag toggles the gating block.

`dev/eval/cumulative/matrix.py`: add `"real.checklist"` to `REAL_REGIMES`. (Caveat: this also adds it
to the default unfiltered set; benign since the A/B always passes `--regimes real.full,real.checklist`.)

**TDD (light, it's a prompt):** `dev/eval/cumulative/test_build_prompt.py` (pytest):
`build_prompt(app, iface, "skill", checklist=True)` contains the gating self-verification language AND
the recall directive; `checklist=False` lacks the gating language (the contrast). Run
`python3 -m pytest dev/eval/cumulative/test_build_prompt.py -q`. Gate B on the diff.

## Run the A/B

`CUMMATRIX_ROOT=/tmp/l4-sonnet python3 matrix.py --models sonnet --trials 1,2,3,4,5 --regimes
real.full,real.checklist --workers 20`

Per regime, from the per-op JSON, read the harness's already-logged fields (no threshold to define —
the harness emits these via the `converged()` gate at harness.py:808): **`rounds_to_converge`** (int,
or `None`/max_rounds if never), final arch score (pass-rate), `axis_c2_cost_usd`. Aggregate per regime:
mean ± SD of rounds-to-converge across the 15 cells (5 trials × 3 apps), mean final arch score, mean $.
Estimated cost: ~$25-35 (sonnet, 2 regimes × 3 apps × 5 trials).

## Verdict rule (locked, noise-aware)

- **Noise floor** = SD of the soft regime's `rounds_to_converge` across its 15 cells. **VIABLE** if
  `mean(checklist) < mean(soft) − 1·SD(soft)` (the means differ by more than the soft regime's own
  spread) **AND** `mean_final_arch(checklist) ≥ mean_final_arch(soft) − 0.05` (no pass-rate regression
  beyond a small margin).
- **BACKFIRES** if checklist rounds ≥ soft, OR final arch drops > 0.05 below soft (over-constraining
  caused churn — the named risk). Report rounds AND pass-rate together; a rounds win with a pass-rate
  drop is not a win.
- **n=5 is underpowered** — report mean ± SD; if the gap is inside 1 SD, the verdict is "can't
  distinguish" (directional), not "tie" or "win" (gap-below-noise lesson).
- **Opus payoff stays OPEN regardless of the sonnet result** — a sonnet win proves the mechanism cuts
  rounds where there are rounds to cut, NOT that it helps the saturated opus chain (which there's
  little to cut). State this in the verdict, not as a footnote.
- Report **both axes** ($ and rounds); state the measured noise floor explicitly.

## Caveats to report

- Sonnet proof-of-concept; opus headroom is small (saturation) — if sonnet shows the effect, an opus
  confirmation is optional and likely muted.
- The current handoff is already "apply as requirements" — the lever's delta is the *gating
  self-verification*, a narrow change; the effect may be small.
- $ is secondary and noisy; the rounds count is the trustworthy signal.
