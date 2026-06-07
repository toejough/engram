# Cumulative cross-app memory-accumulation eval (v2)

A **standing benchmark**: does engram memory let a human state a *transferable* lesson
**once** and have it applied on every later app — reaching the endpoint with fewer
interventions, faster, cheaper? Re-run it whenever a new LLM model ships or engram grows a
feature, and `compare.py` it against the prior baseline.

Three cumulative CLI apps where each genuinely needs the prior's lessons:
`notes` (teaches **α** = tag/search) → `links` (teaches **β** = URL validation + canonical
dedup + import/export) → `feeds` (target; needs architecture + α + β + native feed logic).
The builder is given **only the command list** — never the architecture or quality bar.

## The 7 regimes — write-tier (what `/learn` writes) × read-subset (what `/recall` surfaces)

| id | write-tier | read subset | recall encoding |
|---|---|---|---|
| `cold` | nothing | nothing | no recall |
| `l1` | L1 | {L1} | `engram query --tier L1` |
| `l2.l1l2` | L1+L2 | {L1,L2} | `engram query` (blended; vault holds only L1+L2) |
| `l2.l2` | L1+L2 | {L2} | `engram query --tier L2` |
| `l3.l1l2l3` | L1+L2+L3 | {L1,L2,L3} | `engram query` (blended; full vault) |
| `l3.l2l3` | L1+L2+L3 | {L2,L3} | `engram query --tier L2 --tier L3` |
| `l3.l3` | L1+L2+L3 | {L3} | `engram query --tier L3` |

Write-tiers collapse to 4 seed vaults: `none`(cold), `L1`, `L2`(l2.\*), `L3`(l3.\*). Tier-read
regimes are **not blinded** — surfaced notes carry `outbound_links`, and the build prompt tells
the agent to follow them on demand with `engram show <basename>` (direct-provision vs
follow-on-demand, not a handicap). `link_followed` is recorded per cell.

Per `(model, trial)`: app1 built **cold once** → 4 write-tier learns → `v1[none|L1|L2|L3]`; then 7
regimes branch (app2 reads `v1[write]` under its read-subset → builds → learns → `v2[regime]`;
app3 reads `v2[regime]`, terminal). **18 cells/(model,trial)** × 3 models × 5 trials = **270**.

**Learn is agent-driven, and learn-quality is measured.** The real learn is the agent running its
own `/learn` skill (one episode at L1; +one fact per *stated* convention at L2; +a synthesized ADR
at L3) — so the **whole** memory system is exercised, not just recall. The harness then enforces
the write-tier ceiling (the experimental condition) and **scores capture quality**: for each
stated convention, did the agent persist a note covering it (name-agnostic)? That coverage is a
reported metric — a poor or empty capture is *recorded*, not engineered away. The `--stub` flag
swaps in a deterministic `engram learn` writer for zero-cost pipeline validation only.

## One-command surfaces

```bash
# Zero-cost validation — NO LLM, no spend. Cell-gen, scorer name-agnosticism,
# full pipeline mechanics (stub builder), clean room. Run this from a clean checkout.
python3 validate.py

# Pilot — 1 model × 1 trial × 7 regimes (~26 ops). Real spend + shares your Claude quota
# (headless claude -p --permission-mode bypassPermissions). Resumable / budget-capped.
python3 matrix.py --models sonnet --trials 1 --budget 60

# Full run — 3 models × 5 trials = 270 cells. Launch only after the pilot calibrates cost.
python3 matrix.py --budget 1500

# Aggregate the §5 headline tables into results-v2.md (also prints to stdout).
python3 aggregate.py

# Compare a new run against a baseline (standing-benchmark diff, primary metric).
python3 compare.py <baseline-root> <new-root>
```

### Adding a model
Edit the `MODELS` registry in `harness.py` (one line) — e.g.
`"opus": "claude-opus-4-9"`. Everything else keys off it.

### Re-running after an engram feature
`engram update` (rebuilds the binary + syncs skills), then re-run the pilot/full. Each result
records the `engram_sha`, so `compare.py` shows the feature's effect on the metric.

## Metrics (§5)

- **Primary — repeated-convention interventions** (the say-once signal): per build, how many
  transferable conventions had to be **stated** (round-1 ARCH fails; feedback states *all* gaps).
  Chain-summed: memory ≈ |conv| once; no-memory ≈ |conv| × 3. **App-specific feature
  interventions are the control** — memory should not move them.
- **Secondary:** round-1 conformance, **β-bucket accumulation** (does β transfer once `links`'s
  memory is present), **direct-vs-followed** on the tier-read regimes.

## Confounds designed out (§4)

- **Force the memory path + assert it fired** (`recall_ok`): headless agents don't self-fire
  recall, so a warm cell that ran zero `engram query` is flagged.
- **Name-agnostic scorer** (`archscore.py`): keys on the *pattern* (any injected persistence
  interface, sentinel+`%w`+`errors.Is`, temp+rename, …), never the vault's vocabulary —
  a baseline writing `Repository` instead of `Store` still scores DI. Validated in `validate.py`.
- **Identical reviewer feedback across arms**: the metric *is* "how many times the human stated
  each convention," so teaching cold the conventions is the measurement, not a confound.
- **Clean room**: every build runs with no `CLAUDE.md`/`AGENTS.md` and a cfg carrying only the
  recall/learn skills (built from the repo + keychain creds — no `/tmp` source dependency).
- **Cost decomposed** from token counts (`verify_cost2.py`, 1.00×) before any cost claim.

## Files

| file | role |
|---|---|
| `harness.py` | one operation — `build` (recall→converge loop, uses the LLM) or `learn` (deterministic write-tier capture via `engram learn`, **no LLM**). `MODELS` registry, `REGIMES`, `CONVENTION_FACTS`. |
| `matrix.py` | orchestrator: 7-regime operation DAG, durable cfg pool, resumable, budget-capped. |
| `score.py` + `archscore.py` + `behavioral.py` + `dimensions.py` | deterministic name-agnostic scorer (runs the binary). |
| `{notes,links,feeds}_spec.json` | the three app specs (blind command list + hidden rubric). |
| `aggregate.py` / `compare.py` | §5 tables + `results-v2.md`; cross-run deltas. |
| `validate.py` | zero-cost validation suite (no LLM). |
| `testdata/{good,naive}/` | scorer-validation Go apps (NOT experiment apps; under `testdata/` so repo Go tooling ignores them). |
| `verify_cost2.py` + `token_table.py` | cost audit (reconstruct cost from tokens × price sheet). |

## Where results land

`$CUMMATRIX_ROOT` (default `/tmp/cummatrix`): `results/*.json` (per-op, versioned schema +
`engram_sha` provenance + `run-manifest.json`), `vaults/` (seed + accumulated), `ws/` (build
workdirs). Commit the aggregated `results-vN.md`; archive raw per-cell JSON under a dated dir
or `.gitignore` it with the doc committed.
