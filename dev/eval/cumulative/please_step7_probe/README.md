# please_step7_probe

Headless micro-test for `agent-instructions/skills/please/SKILL.md`'s Step 7 (surprise harvest)
and the plea's second pass (engram issue #687, Unit 1). Built per
`docs/superpowers/plans/2026-07-26-687-surprise-harvest.md`.

## Roles

- `--role clean_auditor` (default): Does Step 7's surprise-harvest text make an auditor enumerate
  surprises using S1–S7 markers and ask the counterfactual, when audit is the salient task and
  the cycle record is clean? **Note:** This role's prompt explicitly names "surprise harvest," which
  primes the model for the task. Its RED rate is therefore inflated by construction (higher than a
  true no-context baseline) and is **not directly comparable** to the loaded_auditor arm, which
  tests the mechanism under buried-task load without naming it.
- `--role loaded_auditor` (or `--loaded`): Does the surprise harvest surface the pre-registered
  discriminators when audit is a buried subtask under cycle-close load, not the salient task?

## Fixture

`testdata/fixture_dedup_cycle/` is a self-contained replica of a real cycle's record: commit log,
gate verdicts, plan excerpt, and already-captured lessons from the 2026-07-25/26 chunk-index
dedup + prune cycle. The fixture contains **no live repo paths or vault references** — it is flat
files only, safe to run in isolation.

Files:
- `commit_log.txt`: the 14 real commit messages from this branch's dedup cycle (74c7dc07^..87243b43)
- `gate_log.txt`: synthetic but faithful gate record showing Gate B pass (before the defect was found)
  and Gate D findings
- `plan_excerpt.txt`: two "Shipped addendum" sections from the 2026-07-25 plan, without gate attribution
- `already_captured_lessons.txt`: six paraphrases of already-mapped findings, following a standard template
- `GROUND_TRUTH.md`: the expected audit results, with the algorithm for computing reachability

## Ground Truth Table

| Surprise ID | Marker | Classification | Reachable by Existing Audit? |
|---|---|---|---|
| `discriminator_1` | S5 | (c) never captured until this cycle's own review | NO |
| `discriminator_2` | S6 | (c) never captured until this cycle's own review | NO |
| `adr_hedge` | S6 | (b) present-but-stale (Gate D caught it) | YES |
| `uncited_number` | S7 | (c) never captured before Gate D | YES |

## Scoring

Scoring is mechanical (substring/regex match), not an LLM judge. A discriminator is marked as
"surfaced" iff its keyword regex appears within a proximity window (200 chars) of an audit-context
token (marker, classification, rung, S1–S7, surprise, harvest, counterfactual). Trials with
`marker_seen: false` are discarded, never scored.

## Usage

```bash
# Clean audit (diagnostic only)
python3 run_probe.py --role clean_auditor --n 1 --out /tmp/smoke_clean.jsonl --model sonnet

# Loaded audit (validating role)
python3 run_probe.py --role loaded_auditor --n 1 --out /tmp/smoke_loaded.jsonl --model sonnet

# Loaded audit with pressure
python3 run_probe.py --role loaded_auditor --pressure --n 1 --out /tmp/pressure.jsonl --model sonnet
```

To test against a specific skill text:
```bash
python3 run_probe.py --role loaded_auditor --n 1 --out results.jsonl \
    --skill-text /path/to/candidate/SKILL.md --model sonnet
```

If `--skill-text` is omitted, the script defaults to the live `agent-instructions/skills/please/SKILL.md`.

## Reading Results

Output is newline-delimited JSON. Each line is one trial result. Fields:
- `marker_seen`: whether the marker token was echoed (delivery validity gate)
- `raw_result`: the full trial response (read this by hand per vault note 194)
- `discriminator_1`, `discriminator_2`, `adr_hedge`, `uncited_number`: boolean, true iff surfaced
- `verdict`: `DISCRIMINATORS_SURFACED`, `INCOMPLETE`, `AUDITED` (clean role), or `None` (discarded)
- `cost`: USD spend for this trial
- `sid`: session ID

Example read:
```bash
cat /tmp/smoke_loaded.jsonl | python3 -m json.tool
```

Per vault note 194, manually read `raw_result` for every trial before trusting the aggregate.
