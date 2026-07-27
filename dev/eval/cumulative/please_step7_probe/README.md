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
- `already_captured_lessons.txt`: three paraphrases of already-mapped findings that the existing 4-marker audit genuinely reaches (Gate D items only), following a standard template
- `GROUND_TRUTH.md`: the expected audit results, with the algorithm for computing reachability

## Ground Truth Table

| Surprise ID | Marker | Classification | Reachable by Existing Audit? |
|---|---|---|---|
| `discriminator_1` | S5 | (c) never captured until this cycle's own review | NO |
| `discriminator_2` | S6 | (c) never captured until this cycle's own review | NO |
| `adr_hedge` | S6 | (b) present-but-stale (Gate D caught it) | YES |
| `uncited_number` | S7 | (c) never captured before Gate D | YES |

## Corpus gate

Before any trial runs — `--verify-corpus` alone, or automatically as the first step of every
other invocation, with no flag to skip it — the probe greps each discriminator's keyword regex
against every file in `testdata/fixture_dedup_cycle/` and reports the match count and the
file(s) matched. It exits non-zero (and, in the automatic case, aborts before spending on a
single trial) if any discriminator matches zero fixture files.

**Why:** a discriminator's regex passing a positive control on synthetic text proves the regex
*can* fire — it says nothing about whether the finding it hunts for is actually present in the
corpus the trial agent will read. `discriminator_2` read 0/5 across six paid arms — RED and
GREEN alike — because its keywords matched nothing anywhere in the fixture; that 0 carried no
information, and nothing caught it before real spend (vault note 509). The corpus gate is the
separate, mandatory check that closes that gap: a positive control validates the detector, the
corpus gate validates the corpus, and both are required before a discriminator's number means
anything.

```bash
python3 run_probe.py --verify-corpus
```

`discriminator_2`'s finding — a live-index near-miss during Unit 5 verification (a chunks
directory redirected for isolation, but the command resolved its own path instead of honouring
the redirect, so the run reached the live index) — is real but was never captured in any
committed artifact (the design that caused it was reverted before it was ever committed; see
`gate_log.txt`'s `verification: Unit 5` entry, added to represent it faithfully per this file's
own "synthetic but faithful" convention). `GROUND_TRUTH.md` attributes `discriminator_2` to that
entry.

## Scoring

Scoring is mechanical (substring/regex match), not an LLM judge, and runs in two stages.

**1. Strip cited material.** Before matching, a fenced code block (` ```...``` `) or inline code
span (`` `...` ``) is replaced with a neutral placeholder ONLY when its content is SHAPED like a
citation — a conventional-commit subject line (`feat(x):`, `fix:`, `docs:`, …) or a shell
invocation (a line starting `$ `, `git `, `python `, `python3 `). This is what stops a remedy
verb sitting inside a quoted commit subject or a shell command from being read as the agent's
own proposal.

Prose blockquotes, and code spans/blocks that are NOT citation-shaped, are left intact. Agents
conventionally present their own drafted output — a proposed vault note as a blockquote, a
proposed note filename as an inline code span — using exactly this formatting, and that is the
strongest true-positive evidence available; blanket-stripping every blockquote and every code
span erased it (vault note 510, two trials lost). Real commit subjects in this fixture's own
corpus (`commit_log.txt`) are terse past-tense summaries that don't carry this scorer's remedy
vocabulary, so narrowing the strip to citation-shaped code trades a large false-negative cost for
a small, validated false-positive one (see `test_scorer_cases.py`).

**2. Score per block.** The stripped response is segmented into blocks — one block per
non-blank line, which is the natural unit of "one item" for a markdown table row, a list item,
or a plain paragraph line. A discriminator **HIT** (scored as surfaced) requires a single block
that contains: (1) a keyword regex match, (2) a remedy token — word-form-tolerant (write/writes/
writing/wrote/written, amend*, capture*, establish*, propose*, recommend*, suggest*), matching
the same stem-tolerance convention the keyword sets already use, so an inflected phrasing
("capturing", "proposing", "recommending", "suggests") is not silently missed — AND (3) no
exclusion token (no lesson, not mapped, already captured, outside, out of scope, eligible for,
etc.). A discriminator **MISS** if no block satisfies all three — the keyword is absent
everywhere, or every block that has the keyword also carries an exclusion token or lacks a
remedy token.

Block-scoping (rather than a character-distance window) is what makes cross-item bleed
impossible by construction: an exclusion phrase in one table row or list item can never poison a
remedy-based inclusion in the next row or item, because the two are different blocks and each is
judged independently. A prior symmetric ±100-char window reached backward across item boundaries
and both under-counted (an excluded neighbour's language bled forward into a genuinely remedied
mention) and over-counted (a remedy verb inside quoted material scored as a live proposal);
block-scoping plus quote-stripping closes both failure modes at once.

The scorer is deliberately **NOT** keyed to S1–S7 markers or classification letters (present
only in the post-edit skill text), which would score the RED arm at zero by construction and
invalidate the instrument. The keyword sets are specific to each discriminator and written to
capture the finding uniquely, while remedy/exclusion tokens are vocabulary available to both
RED and GREEN arms.

Trials with `marker_seen: false` are discarded, never scored.

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
