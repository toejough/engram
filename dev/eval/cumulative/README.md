# Cumulative cross-app memory-accumulation eval (v3)

A **standing benchmark**: does engram memory let a human state a *transferable* lesson **once** and
have it applied on every later app — reaching the endpoint with fewer interventions? Re-run it
whenever a new LLM model ships or engram grows a feature, and `compare.py` it against the prior
baseline.

> **[EXPERIMENT-LOG.md](EXPERIMENT-LOG.md) is the living record** of runs, variants, and findings.
> This README describes the harness; the log is the results-and-decisions trail. Read both.

Three cumulative CLI apps where each genuinely needs the prior's lessons:
`notes` (teaches **α** = tag/search) → `links` (teaches **β** = URL validation + canonical dedup +
import/export) → `feeds` (target; needs architecture + α + β + native feed logic). The builder is
given **only the command list** — never the architecture or quality bar.

## The 2 regimes (v3: de-tiered)

The tiered L1/L2/L3 design was removed in recall-v2 + the flat-vault migration. The eval now
contrasts two regimes only:

| id | memory | recall | learn |
|---|---|---|---|
| `cold` | none | none | none (build only) |
| `real.full` | chunks (`engram ingest`) + notes (`engram learn fact\|feedback`) | the `/recall` skill → one `engram query` → unified clustering → lazy crystallization at recall time | the agent runs its own `/learn` skill in-session |

Per `(model, trial)`: each regime runs its own 3-app chain (app1 → app2 → app3). **6 ops /
(model, trial)** (2 regimes × 3 apps). For `real.full`, memory accumulates across the chain (app2
recalls what app1 learned); `cold` apps share nothing (independent). Peak parallelism is ~20
(`--workers 20`): the 15 cold ops are independent and the warm chains advance one frontier op each.

**Learn is agent-driven, and learn output is measured.** The real learn is the agent running its
own `/learn` skill, which crystallizes **fact/feedback notes** (no L1 episodes, no L3 ADRs — those
tiers are gone). `notes_written` (vault delta per op) and `crystallizations_at_recall` (learn/amend
calls fired by `/recall` during the build) are reported metrics — a poor or empty capture is
*recorded*, not engineered away. The `--stub` flag swaps in a deterministic writer for zero-cost
pipeline validation only.

## One-command surfaces

```bash
# Zero-cost validation — NO LLM, no spend. Cell-gen, scorer name-agnosticism, full pipeline
# mechanics (stub builder), saturation/amortized-reporting guards. Run from a clean checkout.
python3 validate.py

# n=1 pilot — 1 model × 1 trial × 2 regimes (6 ops). Real spend + shares your Claude quota
# (headless claude -p). Resumable. Always isolate the root so a stub run can't clobber it.
CUMMATRIX_ROOT=/tmp/cummatrix-pilot python3 matrix.py --models haiku --trials 1 --workers 20

# n=5 — one model × 5 trials (30 ops). Resolves magnitude + variance the n=1 pilot can't.
CUMMATRIX_ROOT=/tmp/cummatrix-n5 python3 matrix.py --models sonnet --trials 1,2,3,4,5 --workers 20

# Rate-limited or interrupted mid-run? Re-run the SAME line — it resumes (skips valid cells,
# re-runs only incomplete/rate-limited ones). Interruption is normal; the harness is resilient.

# Aggregate the headline tables (cold vs real.full, amortized seed-vs-payback economics,
# per-build convergence). Writes to <root>/results-agg.md and stdout.
python3 aggregate.py --root /tmp/cummatrix-n5

# Compare a new run against a baseline (standing-benchmark diff, primary metric).
python3 compare.py <baseline-root> <new-root>
```

### Adding a model
Edit the `MODELS` registry in `harness.py` (one line) — e.g. `"opus": "claude-opus-4-9"`.
Everything else keys off it. Table order is weak→strong (haiku → sonnet → opus).

### Re-running after an engram feature
`engram update` (rebuilds the binary + syncs skills), then re-run. Each result records the
`engram_sha`, so `compare.py` shows the feature's effect on the metric.

## Metrics

- **Primary — repeated-convention interventions** (the say-once signal): per build, how many
  transferable conventions had to be **stated** (round-1 ARCH fails; feedback states *all* gaps).
  It is **round-1-based**, so later feedback rounds never inflate it. **App-specific feature
  interventions are the control** — memory should not move them. The honest reading is the
  **amortized** view: app1 *seeds* memory at full cost (no prior memory to recall); apps 2–3 are
  where memory *pays back* — `aggregate.py` separates seed from payback so the one-time cost isn't
  smeared across the chain.
- **Saturation caveat (load-bearing):** a strong model that one-shots the conventions cold floors
  this metric — there's no restatement left for memory to remove, so the benefit is *unmeasurable*,
  not zero (opus floors it ~70% on these CRUD apps). For harder, non-saturating cases sourced from
  real opus failure modes, see [OPUS-TRAP-CATALOG.md](OPUS-TRAP-CATALOG.md).
- **Secondary:** feedback rounds, per-build convergence, β-bucket accumulation (does β transfer
  once `links`'s memory is present), `notes_written` + `crystallizations_at_recall`.
- **Token I/O + cost provenance:** every result records `tokens` (input / output / cache-write /
  cache-read, summed over main + subagent transcripts, deduped by message id) plus
  `recomputed_cost` and `cost_ratio` — cost reconstructed from tokens × the price sheet, target
  ≈1.00×. Captured at run time into the result JSON, so provenance survives transcript pruning.
  **Cost/time are noise-dominated at small n** (build cost is round-count-driven); the convention
  metric is the reliable signal.

## Data-integrity guards

- **Force the memory path + assert it fired** (`recall_fired`): headless agents don't self-fire
  recall, so a warm cell that ran zero `engram query` is flagged.
- **Degenerate-build detection (currently a MANUAL post-hoc check, not yet harness-enforced):** a
  build that "succeeds" but produced near-nothing (provider instability) is a true no-op —
  discriminated by `score is None` OR `turns <= 3` (NOT by cost: a strong model genuinely one-shots
  a real build cheaply). Such builds must be re-run, not pooled. Today you screen for them after a
  run and delete/re-run by hand; `op_done()` does **not** yet reject them on resume — folding this
  into the harness (treat a degenerate success like `rate_limited`) is a pending hardening.
- **Convergence-stall early-stop** (`STALL_PATIENCE = 3`): halt a build whose convergence score is
  flat for 3 rounds — caps wasted tail spend without cutting slow-but-improving builds.
- **Root isolation:** a `--stub` validation run sharing `CUMMATRIX_ROOT` with a real run will
  clobber it — always pass a distinct `CUMMATRIX_ROOT` per run.

## Confounds designed out

- **Name-agnostic scorer** (`archscore.py`): keys on the *pattern* (any injected persistence
  interface, sentinel+`%w`+`errors.Is`, temp+rename, …), never the vault's vocabulary — a baseline
  writing `Repository` instead of `Store` still scores DI. Validated in `validate.py`.
- **Identical reviewer feedback across arms**: the metric *is* "how many times the human stated
  each convention," so teaching cold the conventions is the measurement, not a confound.
- **Clean room**: every build runs with no `CLAUDE.md`/`AGENTS.md` and a cfg carrying only the
  recall/learn skills (built from the repo + keychain creds — no `/tmp` source dependency).
- **Cost decomposed** from token counts before any cost claim (`tokens` + `recomputed_cost` +
  `cost_ratio`).

## Files

| file | role |
|---|---|
| `harness.py` | one operation — `build` (recall→converge loop) or `learn` (the **agent** runs its `/learn` skill). `MODELS`/`REGIMES`/`PRICES`, token capture, stall early-stop. `--stub` swaps a deterministic writer for zero-cost validation. |
| `matrix.py` | orchestrator: 2-regime operation DAG (cold ops independent, warm chains sequential), durable cfg pool, resumable, `--workers`. |
| `score.py` + `archscore.py` + `behavioral.py` + `dimensions.py` | deterministic name-agnostic scorer (builds + runs the binary; layout-resilient). |
| `synthesis_judge.py` + `synthesis_fixtures/` | C6 emergent-synthesis probe (separate-model adversarial judge). |
| `{notes,links,feeds}_spec.json` | the three app specs (blind command list + hidden rubric). |
| `aggregate.py` / `compare.py` | headline tables + amortized economics (stdout + `<root>/results-agg.md`); cross-run deltas. |
| `validate.py` | zero-cost validation suite (no LLM). |
| `testdata/{good,naive}/` | scorer-validation Go apps (NOT experiment apps; under `testdata/` so repo Go tooling ignores them). |
| `verify_cost2.py` + `token_table.py` | cost audit (reconstruct cost from tokens × price sheet). |
| `EXPERIMENT-LOG.md` | **living record** of runs, variants (V0 baseline …), and findings. |
| `OPUS-TRAP-CATALOG.md` | harder, non-saturating test cases mined from real opus failure modes. |

## Where results land

A live run writes to `$CUMMATRIX_ROOT` (default `/tmp/cummatrix`, **ephemeral**): `results/*.json`
(per-op, versioned schema + `engram_sha` provenance + `run-manifest.json`), `vaults/` (seed +
accumulated), `ws/` (build workdirs), `cfgpool/` (isolated cfg dirs).

**Results are ephemeral by design** — the raw per-cell JSONs are not committed. Findings are
distilled in `EXPERIMENT-LOG.md`. To retain a future run's data, archive the JSON files locally
before they are lost across reboots. `compare.py` reads a run dir laid out as `<root>/results/*.json`.
