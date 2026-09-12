---
name: bisect-before-fix
description: Use when a trap/eval gate returns RED on the current branch and you need to determine whether the regression is pre-existing or caused by the change under review, before applying any prescribed fix. Keywords — gate regression, RED verdict, pre-existing failure, change attribution, gate.py, crowded_gate.py, bisecting commits, gate rebuilding, pre-change commit
---

# Bisect Before Attributing: Isolate Pre-Existing Regressions from Change-Caused Failures

## Overview

When a trap/eval gate returns RED on your branch, it names a failure and often suggests a fix. But the regression might be pre-existing (a bug introduced in an earlier commit) rather than caused by your change. Applying a fix to a pre-existing failure masks the real issue, hides a regression that belongs in its own issue, and mislabels the cause. This skill is the bisection procedure that isolates the cause before any fix is applied.

## When to Use

A trap/eval gate (gate.py, crowded_gate.py, or similar) has returned RED on the current HEAD of a branch or change under review. You have a suggested fix and pressure (time, deadline, authority, false confidence) to apply it immediately without verification. Triggers include:

- Gate returns RED with a suggested fix ("use X at line Y")
- You are under pressure (time window, deadline, peer confidence) to merge now
- Authority figure says "we know what this is, just apply the fix"
- Peer says "I've seen this pattern, the fix works every time"
- You feel confident the change caused it

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
| "Just apply the fix, we know it works — I've seen the same pattern in three other PRs" | Pattern matching on the *symptom* ("bare error at line X") is not pattern matching on the *cause*. A pre-existing bare error looks identical to a newly-introduced one. Three other PRs verified on their code, not this line. Verify this cause first. |
| "The deadline is tight, we don't have time to bisect — just apply it" | Bisection takes two minutes: checkout, rebuild, re-run the gate. A wrong fix means a second gate failure, another merge attempt, and the deadline is closed anyway. The two-minute check is faster than guessing. |
| "I'm confident this change caused it because I wrote the PR" | Confidence is not causation. A change-unrelated file can still trigger a gate failure if main was already broken. The only evidence is the gate verdict on the pre-change commit. Verify it. |
| "The senior engineer already reviewed the code and said it's the cause" | Code review is strong on *what* the PR contains, not on *whether main is green*. Main might have been red for days. The lead is not claiming they tested main; only that the diff looks plausible. Verify main is green. |

## Red Flags — Stop and Follow the Procedure

- About to apply a gate's suggested fix without checking whether the failure reproduces on main/the pre-change commit.
- Peer says "I've seen this before, it's definitely X" — past experience describes their code, not yours. Verify yours.
- Deadline approaching and gate just failed — the bisection check (checkout, rebuild, re-run) is *faster* than wrong merge + revert. Run it.
- Feeling confident about the cause because the change is "obvious" — read the gate verdict on the pre-change commit instead.
- Skipping the check because "the senior engineer already reviewed it" — they reviewed the PR, not main. Verify main green separately.

