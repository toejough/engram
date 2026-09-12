---
name: bisect-before-fix
description: Use when a trap/eval gate returns RED on the current branch and you need to determine whether the regression is pre-existing or caused by the change under review, before applying any prescribed fix. Symptoms/keywords — gate regression, RED verdict, pre-existing failure, change attribution, gate.py, crowded_gate.py, bisecting commits, gate rebuilding
---

# Bisect Before Attributing: Isolate Pre-Existing Regressions from Change-Caused Failures

## Overview

When a trap/eval gate returns RED on your branch, the natural next step is to apply that change's own prescribed fix. But the regression might be pre-existing (a bug introduced in an earlier commit) rather than caused by your change. Applying a fix to a pre-existing failure masks the real issue and delays its investigation. This skill is the bisection procedure that isolates the cause before any fix is applied.

## When to Use

When a trap/eval gate (gate.py, crowded_gate.py, or similar) returns RED on the current HEAD of a branch or change under review, and you need to determine whether the regression is pre-existing or caused by this change before applying the change's prescribed fix. Triggers include:

- Gate runs RED on the current branch HEAD and suggests a fix.
- You need to know whether the fix is appropriate (change caused the failure) or whether the failure existed before your changes.
- Investigating a regression that might be introduced by recent commits.
- A gate failure that may or may not be related to the current change under review.

## Procedure

1. Identify the commit immediately preceding the change under review's relevant commits.
2. Check out that preceding commit.
3. Rebuild the artifact under test from that commit.
4. Re-run the identical gate (e.g. `gate.py --tier smoke`) against that rebuilt artifact.
5. Compare this verdict to the original RED verdict observed on the change under review's HEAD.
6. If the same failure reproduces on the pre-change commit: the change under review is NOT the cause -- do not apply its prescribed fix; instead file the pre-existing regression as its own separate issue for dedicated investigation.
7. If the failure does NOT reproduce on the pre-change commit: the change under review is confirmed as the cause -- its prescribed fix (if any) applies.
8. Restore the original HEAD/branch and rebuild the artifact before reporting results.

## Done When

The regression has been correctly attributed as pre-existing (with a separate follow-up issue filed) or caused by the change under review, and the original HEAD/branch has been restored and rebuilt before reporting.

## Common Mistakes

| Excuse under pressure | Reality |
|---|---|
| "The gate is failing, just apply the fix it suggests" | The gate's prescribed fix is only correct if the gate's failure was caused by THIS change. Pre-existing failures have different fixes in different layers. Bisect first, apply the appropriate fix second. |
| "We don't have time to test on the pre-change commit, let's just fix it now" | Testing the pre-change commit takes minutes. Misattributing a pre-existing failure and shipping a wrong fix takes hours to diagnose and days to undo. Bisect first. |
| "I'm confident this change caused the failure, I wrote it" | Gate failures under stress (time pressure, sunk cost) can feel obviously attributable to recent changes when they're actually pre-existing. Confidence is not evidence. Bisect anyway. |

## Red Flags — Stop and Follow the Procedure

- Gate returns RED and you're about to apply its prescribed fix without bisecting first.
- "I'm sure this is my change's fault" — a conviction, not a fact. Bisect anyway.
- Skipping the pre-change rebuild step to save time — the rebuild is 80% of the time; the bisect is cheap.
- Committing or reporting results before restoring the original HEAD and rebuilding.

