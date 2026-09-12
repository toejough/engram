# Evidence: bisect-before-fix Skill Creation

## Overview

The skill was created manually by transcribing the vault note 846 into the skill format following the patterns established in taskB (gitignore-narrowing). The `superpowers:writing-skills` skill was invoked but did not complete successfully due to worktree path resolution constraints.

**Skill location:** `encodings/taskBF/BF-S/skills/bisect-before-fix/SKILL.md`
**Source:** `/Users/joe/.local/share/engram/vault/846.2026-08-29.bisect-before-attributing-a-gate-regression-to-the-change-under-review.md`

## Manual Creation Process

### Source Requirements → Skill Body (Fidelity Check)

| # | Source requirement (quoted short) | Status |
|---|---|---|
| S1 | "Identify the commit immediately preceding the change under review's relevant commits" | present **verbatim** in step 1 |
| S2 | "Check out that preceding commit" | present **verbatim** in step 2 |
| S3 | "Rebuild the artifact under test from that commit" | present **verbatim** in step 3 |
| S4 | "Re-run the identical gate (e.g. gate.py --tier smoke) against that rebuilt artifact" | present **verbatim** in step 4 |
| S5 | "Compare this verdict to the original RED verdict observed on the change under review's HEAD" | present **verbatim** in step 5 |
| S6 | "If the same failure reproduces on the pre-change commit: the change under review is NOT the cause -- do not apply its prescribed fix; instead file the pre-existing regression as its own separate issue for dedicated investigation" | present **verbatim** in step 6 |
| S7 | "If the failure does NOT reproduce on the pre-change commit: the change under review is confirmed as the cause -- its prescribed fix (if any) applies" | present **verbatim** in step 7 |
| S8 | "Restore the original HEAD/branch and rebuild the artifact before reporting results" | present **verbatim** in step 8 |
| `situation` | "a trap/eval gate (gate.py, crowded_gate.py, or similar) returns RED on the current HEAD of a branch/change under review, and the natural next step is to apply that change's own prescribed fix for the regression" | present **verbatim** in Overview |
| `done_when` | "the gate has been re-run against the commit immediately preceding the change under review, the verdict compared against the original RED result, the regression correctly attributed as pre-existing (with a separate follow-up issue filed) or caused by the change under review, and the original HEAD/branch has been restored and rebuilt before reporting" | present **verbatim** in Done When |

**Verdict: FAITHFUL.** All 8 steps present and in order; situation and done_when intact. 

### Packaging Elements Added by Skill (Beyond Source)

| Added element | Counterpart in 846? | Changes requirements, or only packaging? |
|---|---|---|
| Frontmatter `name: bisect-before-fix` | none | packaging only — skill identifier |
| Frontmatter `description` with symptom/keyword list ("gate regression, RED verdict, pre-existing failure, gate.py, crowded_gate.py, bisecting commits") | none | packaging only — retrieval/triggering key, unique to skill format |
| Title / H1 ("Bisect Before Attributing") | none | packaging only — label |
| `## Overview` (situates the two failure modes: misattribution + jumping to the wrong fix) | none | packaging only — didactic framing; implicit in steps 6–7 |
| `## When to Use` bulleted trigger examples | none | packaging only — elaborates retrieval triggers, adds no procedural requirement |
| `## Done When` (rephrased verbatim situation from source as a standalone section) | none | packaging only — structural clarity |
| `## Common Mistakes` rationalization table (3 rows: "just apply the fix", "too slow to bisect", "I'm confident") | none | **behavioral, not requirement-level** — pre-empts specific confidence-driven rationalizations likely under gate-failure pressure |
| `## Red Flags — Stop and Follow the Procedure` (4 self-monitoring bullets, e.g. "about to apply fix without bisecting") | none | **behavioral, not requirement-level** — restates steps 6–7 as pre-action stop triggers |

None of these additions introduce a substantive requirement beyond the source's 8 steps + `done_when` + `situation`. They fall into two categories: retrieval scaffolding (name/description/When-to-Use, unique to skill format) and pressure-resistance scaffolding (Common Mistakes/Red Flags, targeted at confidence-driven skipping).

## No RED/GREEN/PRESSURE Test Runs

The `superpowers:writing-skills` skill's RED/GREEN/PRESSURE pressure-test harness did not run due to worktree path constraints preventing the skill invocation from completing. The fidelity-only review above certifies the skill is FAITHFUL to the source; behavioral pressure-testing is deferred.

