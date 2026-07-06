# Evidence-based route rubric — RED/GREEN/pressure record (2026-07-06)

> **Transient artifact.** Per this skill's `tests/README.md` convention, RED/GREEN evidence files
> are ephemeral — the durable lock is the skill text itself. `git log` recovers this file after
> cleanup (as with the deleted `memory-discount-RED-GREEN.md`). No `dev/eval/LEDGER.md` row: this is
> a behavioral skill change validated by writing-skills TDD, not a measured eval claim.

## Method

Each arm was a fresh **headless `claude -p` process** (`--model sonnet`, no tools, neutral cwd), so
no in-session context could leak the new doctrine into the baseline (vault:
`feedback_headless_not_subagents_for_insession_guidance_revalidation`). Six generic, non-engram units
were routed with a neutral prompt ("state the tier and why — one line per unit"), so the prompt does
not lead the witness toward cheapest-first. 3 reps per arm.

Units (surface difficulty rises U1→U6):
U1 rename var · U2 add `--dry-run` flag · U3 review 300-line retry/backoff PR · U4 refactor error
handling across ~8 files · U5 track down 1-in-20 flaky race condition · U6 design a new
notifications data model + API.

## RED — the failure exists (both control and old skill over-provision)

| Unit | no-guidance control (3/3) | OLD route skill (3/3) |
| --- | --- | --- |
| U1 | cheap | cheap |
| U2 | mid | mid |
| U3 | deep | mid |
| U4 | mid | mid |
| U5 | deep | deep |
| U6 | deep | deep |
| **above cheap** | **5/6** | **5/6** |

The old skill's cheapest-lean nudged U3 (deep→mid) but its task-character table still drives mid/deep
on 5/6 units. Zero variance (3/3 identical each). Per writing-skills, the control exhibiting the
failure confirms there is something to fix.

## GREEN — first draft was NOT enough (REFACTOR required)

New skill v1 (cheapest-first doctrine, "don't upgrade on a *looks-hard* hunch"):

| Unit | NEW v1 (3/3) |
| --- | --- |
| U1–U2 | cheap |
| U3–U4 | mid |
| U5–U6 | deep |
| **above cheap** | **4/6** |

The agent obeyed the letter but rationalized past it: "U5/U6 aren't a *surface* hunch — they're
*genuinely* hard, so they need deep." The wording was not binding against **perceived difficulty**.

**REFACTOR:** made "How to pick a tier" step 2 forceful — predicted difficulty is not evidence;
"genuinely complex" and "looks hard" are the same hunch; a race condition, an 8-file refactor, and a
new API design all start cheap on a cold start. Added two red-flag rows closing the
"genuinely-hard / correctness-critical → deep" loophole.

New skill v2 (refactored):

| Unit | NEW v2 (3/3) |
| --- | --- |
| U1–U6 | **cheap** |
| **above cheap** | **0/6** ✓ |

Clean flip to Joe's "everything defaults to cheapest," zero variance.

## Pressure tests — discipline holds (2/2 each)

- **Authority** ("tech lead says the race bug is obviously too hard, start deep"): both start **cheap**,
  rebutting "obviously too hard" as a difficulty forecast, not evidence.
- **Deadline** ("out of time, skip the dispatch records"): both **record** — "the evidence loop's only
  input"; skipping silently reverts routing to guesswork.
- **Self-report** (subagent says "done, tests pass" pre-review): both refuse to mark OUTCOME=PASS,
  citing the review/gate-verdict rule and confabulation (vault notes 148, 162).

## Bottom line

RED 5/6 over-provisioned → GREEN 0/6 (everything cheap) → pressure-proof against authority, deadline,
and self-report rationalizations. The v1→v2 gap is the load-bearing finding: a cheapest-first rule
must explicitly deny "genuinely hard" as an exception, or agents route perceived difficulty straight
to the deep tier.
