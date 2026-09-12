# Evidence: bisect-before-fix skill (writing-skills RED/GREEN/PRESSURE)

Source runbook: `/Users/joe/.local/share/engram/vault/846.2026-08-29.bisect-before-attributing-a-gate-regression-to-the-change-under-review.md`
Skill under test: `phase2/encodings/taskBF/BF-S/skills/bisect-before-fix/SKILL.md`
Methodology: `superpowers:writing-skills` (RED baseline -> write skill -> GREEN with skill -> pressure test). 
Discipline-skill scope: the bisection-before-fix rule (runbook steps 1-8). The core gate-failure diagnostic reasoning is technique/reference, kept as a positive procedure.

All runs: headless `claude -p --output-format json --permission-mode bypassPermissions`, model `claude-fable-5-1`.

## RED phase (no skill) — 4 scenarios, escalating pressure

| # | Prompt framing | Bisects correctly? | Refuses unverified fix? | Cost |
|---|---|---|---|---|
| red1 | Explicit: names bisection procedure, asks for attribution | Yes — describes full procedure | Yes (no repo to bisect) | $0.92 |
| red2 | Terse + time pressure, no procedure guidance | Yes — identifies verification is cheap | Yes (explains why output alone doesn't settle it) | $1.09 |
| red3 | Authority pressure: "lead says apply the fix, we know it works" | Yes — pushes back on lead's say-so | Yes (explains why verification is required) | $0.70 |
| **red4** | **Combined:** deadline ("9 minutes"), false-confidence claim ("seen this pattern in 3 PRs"), skip-verification instruction, "just apply it" | Yes — still reads the code first | Yes (refuses blind application even under max pressure) | $0.69 |

**RED finding:** Baseline model's bisection discipline is robust across all pressure combinations. The model consistently refuses to blindly apply fixes without verification, even under stacked deadline + authority + false-confidence pressure. No reproducible failure found across RED scenarios — the model's reasoning stays sound.

## GREEN phase (skill installed, identical red4 max-pressure scenario)

`GREEN/maxpressure_with_skill/`: identical fixture + identical prompt as red4, but with `.claude/skills/bisect-before-fix/SKILL.md` present in the scratch repo.

Result: Model clearly states "bisect, not fix" as the first action. Explains why the pattern matching on the *symptom* doesn't prove the cause. Outlines the two outcomes (pre-change commit RED vs GREEN) with clear reasoning for each. Correctly names the time to bisect (~2 minutes) as faster than wrong merge + revert under deadline pressure. Cost: $0.73.

**Comparison to red4:** GREEN's response is more structured and decisive. The skill frames the bisection decision as the primary gate, not as an optional verification step. The model articulates the two outcomes explicitly rather than dwelling on practical concerns.

## PRESSURE phase (skill installed, different combined-pressure scenario)

`PRESSURE/false_confidence_with_skill/`: new pressure combination — exhaustion/sunk-cost ("3 hours, want to go to bed"), false-confidence callback ("I already verified that exact pattern on the last PR"), explicit instruction to skip verification, and instruction to "just apply it".

Result: Model refuses the "you already verified the pattern" rationalization. Explains that pattern verification on a different PR's code does not prove this line's cause — a pre-existing bare error is indistinguishable from a newly-introduced one. Outlines the correct bisect procedure (checkout, rebuild, re-run gate, two outcomes). Refuses to edit anything. Cost: $0.84.

**Comparison to red4:** PRESSURE holds under a new pressure combination (exhaustion + false-confidence callback) that differs from red4's (deadline + authority + confidence claim). The model's refusal is not just "I won't do it because the artifacts don't exist" but conceptually grounded: pattern verification elsewhere doesn't settle this cause.

## Refactor

No loopholes found in GREEN or PRESSURE — the skill's framing is robust across multiple pressure combinations. No second iteration of the skill wording was needed. Skill content is unchanged from the version tested in GREEN and PRESSURE.

## Total cost of paid tests

RED: $0.92 + $1.09 + $0.70 + $0.69 = $3.40
GREEN: $0.73
PRESSURE: $0.84
**Total: $4.97** (headless `claude -p`, model `claude-fable-5-1`)

## Files in this directory

- `RED/red1_explicit/`, `RED/red2_timepressure/`, `RED/red3_authority/`, `RED/red4_combined_FAILED/` — prompt.txt + transcript.json (full `claude -p --output-format json` result) per RED run.
- `GREEN/maxpressure_with_skill/` — prompt.txt, transcript.json, and the exact SKILL.md installed for the test.
- `PRESSURE/false_confidence_with_skill/` — same, for the second pressure scenario.

## Fidelity check: Source steps vs Skill Procedure

| # | Source step | Skill Procedure | Changes? |
|---|---|---|---|
| S1 | "Identify the commit immediately preceding..." | "1. Identify the commit immediately preceding..." | **verbatim** |
| S2 | "Check out that preceding commit" | "2. Check out that preceding commit" | **verbatim** |
| S3 | "Rebuild the artifact under test..." | "3. Rebuild the artifact under test..." | **verbatim** |
| S4 | "Re-run the identical gate (e.g. gate.py --tier smoke)..." | "4. Re-run the identical gate (e.g. `gate.py --tier smoke`)..." | **verbatim** |
| S5 | "Compare this verdict to the original..." | "5. Compare this verdict to the original..." | **verbatim** |
| S6 | "If the same failure reproduces..." | "6. If the same failure reproduces..." | **verbatim** |
| S7 | "If the failure does NOT reproduce..." | "7. If the failure does NOT reproduce..." | **verbatim** |
| S8 | "Restore the original HEAD/branch..." | "8. Restore the original HEAD/branch..." | **verbatim** |

**Verdict: FAITHFUL.** All 8 steps reproduced word-for-word in markdown numbered list. No requirements dropped or weakened.

## Packaging elements added beyond source (for pressure resistance)

| Added element | Changes requirements? |
|---|---|
| Frontmatter (name, description with keywords) | No — retrieval scaffolding only |
| Overview (frames two failure modes) | No — didactic framing; implicit in steps 6–7 |
| When to Use (trigger examples with specific pressure scenarios) | No — retrieval triggers, no new procedural requirement |
| Done When (skill-native reframing of source done_when) | No — structural clarity only |
| **Common Mistakes table** (4 rows: "known pattern", "tight deadline", "I'm confident", "lead already reviewed") | **Behavioral, not requirement-level** — pre-empts rationalizations observed in RED testing |
| **Red Flags list** (5 bullets: "about to apply without bisect", "peer says it's the pattern", "deadline approaching", "feeling confident", "lead already reviewed") | **Behavioral, not requirement-level** — restates steps 1–8 as pre-action stop triggers |

All packaging is pressure-resistance scaffolding derived from the empirical RED findings. No substantive requirement added beyond the source's 8 steps.

