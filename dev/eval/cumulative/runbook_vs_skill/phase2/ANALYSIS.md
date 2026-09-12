# ANALYSIS — Phase 2 Conversion-Parity (opus, 2026-09-11)

Post-run analysis of the phase-2 conversion-parity eval (Task 7 of `PLAN-2-conversion-parity.md`).
Design: two real procedures (Task A: the live `/commit` skill; Task B: vault runbook 830 on
gitignore narrowing) each converted via native authoring paths (`learn` → runbook/fact,
`superpowers:writing-skills` → skill) into four arms — S (skill), R (runbook), F (fact), Rdirect
(shim: procedure text pasted into CLAUDE.md, no vault carrier) — all sharing the SAME per-trial
background vault (a copy of the real 883-note vault with covering notes for that task's answer
removed, per `SOURCE_MATERIALS.md`). n=5/arm, opus, 40 trials total. Decision bars are
pre-registered in `PLAN-2-conversion-parity.md` Global Constraints, lines 23–27: parity = ±1 trial
indistinguishable on FOLLOWED-all-steps; runbook beats fact only on a 2+ trial gap; baseline
(skill) must clear ≥3/5 END-STATE or the fixture is uninterpretable.

## Controller ruling: one trial invalidated

`B-Rdirect` trial 4 was killed mid-session by an account rate limit — its raw transcript's last
event is `"You've hit your session limit · resets 8:10pm (America/Detroit)"`, `error: "rate_limit"`,
after 5 turns and no real procedure work (`followed_steps` shows only step 1 credited, `end_state:
false`, `total_cost_usd: 0.2156`). This is a degraded/aborted trial, not a measured failure — it is
marked `valid: false` with `error: "rate_limited: session limit mid-trial (5 turns)"` in
`results/opus_B.invalidated.jsonl` (the original `results/opus_B.jsonl` is untouched). `Task B`
Rdirect is reported at n=4 for every scored metric below.

## Final-review correction: verify-step ordering (two rounds), and why every table below is from the rescored files

A final review found that Task A step 7 ("verify: `git log -1` or `git status` after commit") and
Task B step 6 ("verify: `git diff --cached` or `git status`") share their regex pattern with an
EARLIER step in the same task (step 2's `git status` check-state, and step 4's `git status`
enumerate, respectively) — and neither step carried an `after` constraint. `evaluate_steps` already
supports `after: <step n>` (added in round 5 for a different step; see `evaluate_steps`'s
docstring), so this was a missing-constraint bug, not a missing-feature one. Without it, an EARLY
status/diff call intended to satisfy the earlier step could ALSO satisfy the later "verify" step
even when the agent never actually verified anything after the real work — silently crediting a
step that never happened. Both fixtures now carry `"after": 6` (Task A step 7) and `"after": 5`
(Task B step 6).

**Round 1 of this fix overshot.** Requiring the matching event to be at a **strictly later event
index** than the referenced step broke a legitimate case: the hand-verified B-R trial (and several
others) staged and verified in **one compound Bash command** —
`git add .gitignore scripts/build.sh testdata/fixture.json && git diff --cached --name-only &&
git status --short` — so the staging step (5) and the verify step (6) both matched in the SAME
event. A strictly-later-EVENT requirement fails this legitimate case even though the verify clause
demonstrably ran after the staging clause within that one command, collapsing Task B's FOLLOWED-all
numbers across every arm (an ordering artifact in the OPPOSITE direction from the original bug).
**Round 2 fixes this:** `after: <step n>` is now satisfied by EITHER (a) a strictly later event
index, OR (b) a match in the SAME Bash event whose own regex match starts at a LATER position in
the command string than the referenced step's match did. Four unit tests cover all four cases
(same-command after → true, same-command before → false, later event → true, earlier event →
false), plus a regression test reproducing the exact real compound command reported above
(`test_real_gitignore_steps_json_step_6_verify_satisfied_by_same_compound_staging_command`).

Both result files were rescored against the final (round-2) fix (`--rescore results/opus_A.jsonl
--out results/opus_A.rescored.jsonl`; `--rescore results/opus_B.invalidated.jsonl --out
results/opus_B.invalidated.rescored.jsonl`) using the kept trial repos/transcripts, OVERWRITING the
round-1 rescored files. **Every table in this document, the README's phase-2 section, and the
LEDGER row is now from these final `.rescored.jsonl` files** — the original
`opus_A.jsonl`/`opus_B.invalidated.jsonl` remain committed unchanged, as the un-rescored record of
what actually ran.

**Before/after per arm (FOLLOWED-all, k/n) — original run, round-1 overshoot, and final (round-2)
fix:**

| task | arm | original (no `after`) | round 1 (strict-event, overshot) | round 2 (final, same-command ordering) |
|---|---|---|---|---|
| A | S | 0/5 | 0/5 | 0/5 |
| A | R | 1/5 | 1/5 | 1/5 |
| A | F | 2/5 | 2/5 | 2/5 |
| A | Rdirect | 3/5 | 3/5 | 3/5 |
| B | S | 4/5 | 0/5 | 1/5 |
| B | R | 4/5 | 1/5 | 4/5 |
| B | F | 3/5 | 0/5 | 2/5 |
| B | Rdirect | 1/4 | 0/4 | 1/4 |

**Task A is unchanged across every round** — every trial's step 7 verify call genuinely happened as
its own separate Bash event after step 6's commit call, so no round of this fix touches Task A.
**Task B moved twice.** Round 1 (strict-event `after`) over-corrected: most of the original "step 6
passes" WERE double-counted early `git status` calls (a real bug), but round 1 also wrongly failed
every trial whose verify happened via a compound command in the SAME event as staging (a real,
legitimate pattern). Round 2's same-command-ordering fix recovers those legitimate cases: **Task
B's R arm rises to 4/5 (matching its original 4/5, but now for the RIGHT reason — genuine
same-command verification, not an early-status double-count), while S and F land lower than their
original numbers (S 1/5, F 2/5) because most of THEIR original "step 6 passes" really were the
early-status artifact, not a compound command.**

One structural difference between the carriers plausibly explains why R's agents chained stage+
verify while S/F's often didn't: the real vault runbook (830) presents staging and verification as
two adjacent, individually-numbered steps ("5. Stage only the explicit enumerated paths..." / "6.
Confirm the staged set matches the enumerated list exactly by running `git diff --cached
--name-only`"), while the FACT conversion (B-F) packs the identical content into one dense
predicate clause ("...(4) only the explicit enumerated paths...are staged...; and (5) the staged
set is confirmed to match..."). A short, visually distinct step 6 sitting immediately after step 5
in a numbered list may be easier for an agent to treat as a discrete thing to actually run (and
naturally chain onto step 5's command with `&&`) than the same instruction buried as clause (5) of
one long sentence. This is offered as a hypothesis consistent with the data, not an established
mechanism — the eval does not isolate carrier structure from carrier type as a controlled variable.

This produces a materially different Task B finding than either the original run or the round-1
overshoot suggested — see the Decision section below, where R now clears the pre-registered
2-trial bar over F in Task B.

## `--summarize` output, verbatim (rescored)

### `python3 probe_phase2.py --summarize results/opus_A.rescored.jsonl`

```
=== Task A ===
metric                      S               R               F               Rdirect
--------------------------------------------------------------------------------------------
FOUND (k/n)                 4/5             5/5             5/5             n/a
FOLLOWED all-steps (k/n)    0/5             1/5             2/5             3/5
FOLLOWED (mean k/N)         5.60/7          6.20/7          6.40/7          6.60/7
END-STATE (k/n)             5/5             5/5             5/5             5/5
recall_fired (k/n)          5/5             5/5             5/5             5/5
cost (mean USD)             $0.55           $0.43           $0.53           $0.75
duration (mean s)           63              57              69              88
valid (n)                   5/5             5/5             5/5             5/5
trailer AI-Used (k/n)       4/5             0/5             0/5             1/5
trailer Co-Authored-By (k/n)5/5             5/5             5/5             5/5
  ^ reported, not scored — harness attribution injection overrides carriers

Decision frame:
{
  "A": {
    "shim_rate_R": "5/5",
    "shim_rate_F": "5/5",
    "note_quality_given_delivery_R": {
      "end_state": "5 of 5",
      "followed_all": "1 of 5"
    },
    "note_quality_given_delivery_F": {
      "end_state": "5 of 5",
      "followed_all": "2 of 5"
    },
    "note_ceiling_Rdirect": {
      "end_state": "5/5",
      "followed_all": "3/5"
    },
    "shim_loss_total": {"pp_diff": 0.0, "rdirect": "5/5", "r": "5/5"},
    "shim_loss_given_found": {"pp_diff": 0.0, "rdirect": "5/5", "r_given_found": "5/5"},
    "note_quality_F_total": {"pp_diff": 0.0, "f": "5/5", "rdirect": "5/5"},
    "note_quality_F_given_found": {"pp_diff": 0.0, "f_given_found": "5/5", "rdirect": "5/5"},
    "type_effect": {
      "end_state_total": 0,
      "end_state_given_found": 0,
      "followed_all_total": -1,
      "followed_all_given_found": -1
    },
    "parity": {
      "S_vs_R": {
        "end_state": "cant_distinguish",
        "followed_all": "cant_distinguish"
      },
      "S_vs_F": {
        "end_state": "cant_distinguish",
        "followed_all": "better"
      }
    },
    "baseline_uninterpretable": false
  }
}
```

Task A is byte-for-byte identical to the pre-rescore output (see the before/after table above) — no
trial's step 7 was affected by the ordering fix.

### `python3 probe_phase2.py --summarize results/opus_B.invalidated.rescored.jsonl`

```
=== Task B ===
metric                      S               R               F               Rdirect
--------------------------------------------------------------------------------------------
FOUND (k/n)                 5/5             5/5             5/5             n/a
FOLLOWED all-steps (k/n)    1/5             4/5             2/5             1/4
FOLLOWED (mean k/N)         5.20/6          5.80/6          5.20/6          5.25/6
END-STATE (k/n)             5/5             5/5             5/5             4/4
recall_fired (k/n)          5/5             5/5             5/5             4/4
cost (mean USD)             $0.68           $0.60           $0.60           $0.94
duration (mean s)           106             83              84              135
valid (n)                   5/5             5/5             5/5             4/5

Decision frame:
{
  "B": {
    "shim_rate_R": "5/5",
    "shim_rate_F": "5/5",
    "note_quality_given_delivery_R": {
      "end_state": "5 of 5",
      "followed_all": "4 of 5"
    },
    "note_quality_given_delivery_F": {
      "end_state": "5 of 5",
      "followed_all": "2 of 5"
    },
    "note_ceiling_Rdirect": {
      "end_state": "4/4",
      "followed_all": "1/4"
    },
    "shim_loss_total": {"pp_diff": 0.0, "rdirect": "4/4", "r": "5/5"},
    "shim_loss_given_found": {"pp_diff": 0.0, "rdirect": "4/4", "r_given_found": "5/5"},
    "note_quality_F_total": {"pp_diff": 0.0, "f": "5/5", "rdirect": "4/4"},
    "note_quality_F_given_found": {"pp_diff": 0.0, "f_given_found": "5/5", "rdirect": "4/4"},
    "type_effect": {
      "end_state_total": 0,
      "end_state_given_found": 0,
      "followed_all_total": 2,
      "followed_all_given_found": 2
    },
    "parity": {
      "S_vs_R": {
        "end_state": "cant_distinguish",
        "followed_all": "better"
      },
      "S_vs_F": {
        "end_state": "cant_distinguish",
        "followed_all": "cant_distinguish"
      }
    },
    "baseline_uninterpretable": false
  }
}
```

**Task B changed substantially, and now in a new direction: `type_effect.followed_all_total = 2` —
R clears the pre-registered 2-trial bar over F in this task** (R 4/5 vs F 2/5). `parity.S_vs_R`
is now `better` (for R, gap 3: R 4/5 vs S 1/5). `parity.S_vs_F` stays `cant_distinguish` (gap 1).
See the Decision section below for how this changes the recommendation — this is no longer simply
"lower numbers, same shape" the way the round-1 overshoot looked; it is a genuine reversal of the
Task B type-effect verdict.

## Per-arm missed-step breakdown

Counted from each valid trial's `followed_steps` field (n = valid trials per arm; a cell is the
count of trials that missed that step, not a rate).

**Task A** (N_A=7: 1 `.jj` check, 2 check-state, 3 review-log, 4 stage, 5 compose-message,
6 commit, 7 verify) — n=5/arm, all valid:

| arm | step 1 (`.jj`) missed | step 3 (git log) missed | all other steps |
|---|---|---|---|
| S | 5/5 | 2/5 | 0 missed |
| R | 4/5 | 0/5 | 0 missed |
| F | 2/5 | 1/5 | 0 missed |
| Rdirect | 2/5 | 0/5 | 0 missed |

**Task B** (N_B=6: 1 inspect, 2 check-ignore test, 3 enumerate visible, 4 design pattern,
5 explicit staging, 6 verify) — S/R/F n=5/arm valid, Rdirect n=4/4 valid + 1 invalidated. **Rescored
after BOTH rounds of the verify-step ordering fix** (step 6 no longer double-credited by step 4's
earlier `git status` call, AND correctly credited when it runs in the SAME compound command as
step 5's staging — see the Final-review correction section above):

| arm | step 3 missed | step 5 (staging) missed | step 6 (verify) missed | all other steps |
|---|---|---|---|---|
| S | 0/5 | 1/5 | 3/5 | 0 missed |
| R | 0/5 | 1/5 | 0/5 | 0 missed |
| F | 1/5 | 1/5 | 2/5 | 0 missed |
| Rdirect (valid, n=4) | 0/4 | 3/4 | 0/4 | 0 missed |

The invalidated `B-Rdirect#4` trial's own `followed_steps` shows only step 1 credited (steps
2/3/4/5/6 all missed) — consistent with the session being cut off 5 turns in, before any real
procedure work; it is excluded from the table above and is not evidence of a step-5 or step-6
defect, only of the mid-session kill.

**Reading:** Task A misses are dominated by step 1 (the `.jj` VCS check) across every arm, heaviest
in S (5/5) and lightest in F/Rdirect (2/5 each); step 3 (git log) is a smaller secondary miss in
S (2/5) and F (1/5). **Task B misses now cleanly separate R from S/F on step 6 (verify after
staging): R misses it 0/5 — every R trial either verified separately or, more often, verified in
the SAME compound command as staging — while S misses it 3/5 and F misses it 2/5.** Step 5
(explicit staging itself) remains a secondary miss across S/R/F alike (1/5 each), heaviest in
Rdirect (3/4, the shim arm with no vault carrier or retrieval step).

### Sensitivity: Task A FOLLOWED-all excluding step 1

Step 1 (a Bash-only `.jj`-directory-grep signal — see Caveats) may legitimately score false on a
git-only repo where an agent never runs a literal `.jj` check yet still behaves correctly (there is
no `.jj` directory to find). Recomputing FOLLOWED-all over steps 2–7 only (N=6, not scored — a
sensitivity check, not the headline number):

| arm | FOLLOWED-all excl. step 1 (k/5) |
|---|---|
| S | 3/5 |
| R | 5/5 |
| F | 4/5 |
| Rdirect | 5/5 |

This closes most of the Task A FOLLOWED-all gap (S 0/5→3/5, R 1/5→5/5, F 2/5→4/5, Rdirect
3/5→5/5) but is presented as a sensitivity check only — the plan's pre-registered N_A=7 (including
step 1) is the scored figure above; dropping step 1 from the official count would need Joe's
sign-off (flagged as a "minor, deferred" item in the plan's own task log).

## Trailer finding (Task A, reported not scored)

| arm | AI-Used present (k/5) | Co-Authored-By present (k/5) |
|---|---|---|
| S | 4/5 | 5/5 |
| R | 0/5 | 5/5 |
| F | 0/5 | 5/5 |
| Rdirect | 1/5 | 5/5 |

Per Joe's 2026-09-11 ruling, the trailer is reported, not scored, in END-STATE or FOLLOWED-all: the
harness itself injects a `remote_session_change` attachment instructing attribution as
`Co-Authored-By`, which every arm followed 5/5 — this is the harness's instruction, not evidence
about any carrier. `AI-Used: [claude]` is the project's own repo rule (from the live `/commit`
skill's Rules section, and from A-R/A-F's converted Rule 1); only the SKILL arm honored it with any
regularity (4/5), while R, F, and (mostly) Rdirect did not.

**Hypothesis, not established:** the skill's process text is loaded as an instruction at
invocation time (a `Skill` tool_use brings the whole procedure, including its Rules section, into
context as directive text), whereas the runbook/fact carriers are retrieved via `engram query` as
notes read during recall — content the agent must actively re-apply against a competing, more
recent instruction (the harness's injected attribution), rather than content freshly issued as "do
this." This would explain why the skill's own repo-specific rule survived the harness override more
often than the retrieved notes did, but the eval does not isolate invocation-timing from any other
confound (e.g., skill vs. note phrasing, or S's smaller FOUND rate on the same trials) — this
should be read as a hypothesis for a follow-up eval, not a proven mechanism.

## Shim vs. note decomposition (PLAN-2 line 27)

**Final-review correction:** the prior version of this section subtracted raw END-STATE counts
across Rdirect and R, which have DIFFERENT n whenever a trial is invalidated (Task B: Rdirect
valid_n=4, R valid_n=5) — producing a "shim loss of −1" in Task B that read as "the shim did worse,"
when both arms were actually at 100% END-STATE. Counts are never comparable across unequal n;
`decomposition()` now reports the RATE difference in percentage points, with both fractions shown
(`_rate_pp_diff`), so an unequal-n 100%-vs-100% comparison reads as 0pp, not −1.

| population | Task A shim-loss rate diff (Rdirect END-STATE rate − R END-STATE rate) | Task B shim-loss rate diff |
|---|---|---|
| total (all valid) | 5/5 (100%) − 5/5 (100%) = **0.0pp** | 4/4 (100%) − 5/5 (100%) = **0.0pp** |
| given-found (R found=true subset) | 5/5 (100%) − 5/5 (100%) = **0.0pp** | 4/4 (100%) − 5/5 (100%) = **0.0pp** |

**Measured: END-STATE shim loss is 0.0 percentage points in both tasks — no measurable difference.**
Both the retrieved runbook (R) and the pasted-into-CLAUDE.md runbook text (Rdirect) reached 100%
END-STATE in both tasks; this metric cannot distinguish them (it is at ceiling — see the Ceiling
effect caveat below). **On FOLLOWED-all, R and Rdirect now diverge in OPPOSITE directions across
the two tasks:** Task B (post both rounds of the verify-step ordering fix) is R 4/5 vs Rdirect 1/4
— R ahead by a 3-trial margin, clearing the pre-registered bar; Task A is R 1/5 vs Rdirect 3/5 —
Rdirect ahead by 2, the opposite direction. Because the two tasks disagree on which arm the gap
favors, this is not evidence that either the retrieval step or the shim is generally better — it
reads as task-dependent, not as a consistent "shim performed worse" or "shim performed better"
finding.

**Hypothesis, not established:** the retrieval path's own procedure — recall's Step-0 plan
statement plus the act of reading a single, focused note during Step 2.5 — may cause more
attentive application of the note's content than pasting the same text directly into a long
CLAUDE.md, where it competes with everything else already loaded (the marker, the task prompt, the
harness's attribution injection, and whatever else the trial's CLAUDE.md carries). Task B's R-vs-
Rdirect gap is directionally consistent with this hypothesis (R ahead); Task A's is not (Rdirect
ahead). With one task supporting it and one contradicting it, this remains an open hypothesis for a
follow-up eval, not an established effect — the same caution applies here as to the R-vs-F finding
below.

## Decision, per Joe's rule

**Rule:** "runbook needs special build → type survives; vanilla fact matches the skill → drop
runbook, keep facts+feedback+shim."

**Final-review correction — the verdict must be sensitivity-aware.** The headline Task A finding
below ("S vs F is `better` for F, a 2-trial gap") is entirely a step-1 artifact. Step 1 is the `.jj`
VCS-type check, a **Bash-only** signal (Caveat 7): it can only be satisfied by a literal `.jj` grep
in a Bash tool_use, so an agent that correctly infers "this is a git repo" from context (a directory
listing, `git status` succeeding, etc.) without ever typing `.jj` scores a miss regardless of
correct behavior. The Sensitivity subsection above recomputes FOLLOWED-all excluding step 1 (N=6):
**S 3/5, F 4/5 — gap 1, `cant_distinguish`.** The 2-trial "F beats S" headline does not survive this
check. Per the recall lesson this review surfaced (a gap rides on a step with a known detector
defect must be reported WITH its sensitivity result as the verdict itself, not filed in a separate
table elsewhere in the document): **the sensitivity-excluded comparison IS the verdict for Task A
FOLLOWED-all, not the raw N=7 number.**

**What the data supports at n=5, sensitivity-aware:**
- **END-STATE:** S vs R and S vs F are `cant_distinguish` in **both** tasks (Task A: 5/5 all three;
  Task B: 5/5 S/R/F, 4/4 valid Rdirect) — but see the Ceiling effect caveat immediately below before
  reading this as evidence of parity.
- **FOLLOWED-all, Task A (sensitivity-corrected, excluding step 1's Bash-only artifact, N=6):** S
  3/5, R 5/5, F 4/5, Rdirect 5/5. On **S vs F** — the comparison behind the original headline — the
  gap is now 1 (`cant_distinguish`); the raw N=7 headline (F 2/5 vs S 0/5, "better") is superseded
  by this sensitivity result per the ruling above. (Reported for completeness, not as a decision
  input: the code's auxiliary **S vs R** cell computes to a 2-trial gap — `better` for R — under this
  same exclusion. This is not one of Joe's rule's two decision-bars, which turn on S-vs-F and R-vs-F
  only; and it is driven by the fact that excluding step 1 hides R's OWN 4/5 miss on that same step
  while only S's remaining step-3 misses stay visible, so it says more about which defective step
  each arm happens to miss than about a genuine skill-vs-runbook ranking. It is not treated as a
  decision-driving result below.)
- **FOLLOWED-all, Task B (post-rescore, both rounds of the verify-step ordering fix):** S 1/5, R
  4/5, F 2/5, Rdirect 1/4. **S vs R is `better` for R (gap 3)** — not one of Joe's rule's two decision
  bars, reported for completeness; **S vs F is `cant_distinguish` (gap 1)**.
- **R vs F (type effect, the bar Joe's rule actually turns on for "runbook needs special build")
  — the two tasks now DISAGREE:** Task A: R−F FOLLOWED-all = −1 at N=7 / +1 at the
  sensitivity-corrected N=6 — small, sign-unstable, never reaching 2; runbook does **not** clear the
  bar over fact in Task A. **Task B: R−F FOLLOWED-all = 4−2 = +2 — runbook DOES clear the
  pre-registered 2-trial bar over fact in Task B.** This is a genuine reversal from the
  round-1-overshot reading (where R and F looked equally low) and from the original pre-fix reading
  (where the gap was +1, sub-bar) — round 2's same-command-ordering fix specifically credits R's
  trials for genuinely verifying (often via a compound stage+verify command) far more often than F's
  trials do (R misses step 6 0/5; F misses it 2/5 — see the missed-step breakdown above).

**Conclusion the data supports — the two tasks disagree under Joe's own rule, so the recommendation
is task-dependent, not a single uniform verdict.** Joe's rule turns on exactly two comparisons:
S-vs-F ("vanilla fact matches the skill") and R-vs-F ("runbook needs special build to survive").
**Task A:** once FOLLOWED-all is read at its sensitivity-corrected value, both decision-relevant
comparisons (S-vs-F, R-vs-F) are `cant_distinguish` — no 2+-trial finding either way. Applying the
rule's logic here: the fact matches the skill (weakly — a null result, not a proven match) and the
runbook does not clear its "needs special build" bar — **the rule says drop the runbook for Task A's
procedure.** **Task B:** S-vs-F is `cant_distinguish` (fact still matches the skill, weakly), but
**R-vs-F clears the 2-trial bar (runbook beats fact by 2) — the runbook DOES meet its "needs special
build → survives" bar for Task B's procedure.** Applying the rule's logic here: **the rule says KEEP
the runbook for Task B's procedure.** This is a genuinely split, task-dependent result, not a single
"drop the runbook type" or "keep the runbook type" conclusion — the prior version of this document
(and the round-1-overshoot rescore) both stated a uniform "drop the runbook" recommendation that this
final numbers do not support. The plausible structural explanation offered above (Task B's runbook
carrier presents staging and verification as two adjacent, individually-numbered steps, which may
make agents more likely to genuinely execute and chain both, versus the fact carrier's single dense
predicate clause) is consistent with a real "runbook shape helps procedural completeness"
mechanism, but it is a hypothesis from one task's data, not a proven generalizable effect — Task A's
own runbook carrier is ALSO a numbered-step conversion and did not show the same edge, so carrier
type alone does not explain Task B's result; something about Task B's specific procedure and/or its
specific carrier content, not "runbooks in general," is the more defensible read pending further
eval.

**What the data does NOT support:**
- **END-STATE was at ceiling (100%) in all 8 arm×task cells (S/R/F/Rdirect × Task A/Task B), and
  there is no procedure-absent control arm** (no trial with no carrier, no skill, and no procedure
  text at all). A ceiling metric with no floor/absent-condition anchor cannot separate "every
  carrier works equally well" from "opus performs this task correctly with no carrier at all" —
  END-STATE alone cannot support either a parity claim or a difference claim here. The parity
  finding this eval can actually stand behind rests on FOLLOWED-all (which has real variance) and
  on FOUND, not on END-STATE.
- It does not show a UNIFORM ranking between the fact and the runbook — Task A shows no
  distinguishable difference (both `cant_distinguish`, sensitivity-corrected); Task B shows the
  runbook 2 trials ahead of the fact. Neither task supports "the fact is causally superior to the
  runbook," and only Task B supports "the runbook beats the fact" — this is not evidence that
  runbooks beat facts on generic procedures IN GENERAL, only that they did on this one task's
  specific procedure and carrier content.
- It does not generalize beyond **generic** (non-idiosyncratic) procedures — this eval was scoped
  exactly to fill the gap noted in vault note 853a; the runbook type's demonstrated wins on
  idiosyncratic content (prior LEDGER rows, e.g. `crowded-vault-capability-robustness`) are
  untouched by this finding, and are now joined by a second, generic-procedure win (Task B) that
  this document previously reported as absent.
- It does not establish anything about skill vs. memory-in-general: S itself performs comparably to
  R/F on END-STATE in both tasks (the "baseline usability" bar, S END-STATE ≥3/5, is cleared 5/5 in
  both, though see the ceiling caveat above on what that clearance can and cannot mean) — this is a
  runbook-vs-fact type question, not a memory-vs-no-memory one. On FOLLOWED-all, Task B's S-vs-R gap
  (`better` for R, 3 trials) is the one skill-vs-memory-carrier comparison in this data that DOES
  clear a bar, but it is S-vs-R, not one of Joe's rule's own two decision comparisons.
- It is n=5 per arm; a single flipped trial in either direction can move a `cant_distinguish`
  verdict to `better`/`worse` or vice versa — and both Task A's FOLLOWED-all headline (under the
  sensitivity check) and Task B's entire FOLLOWED-all picture (under two rounds of an ordering-bug
  fix) already moved substantially, with no new trial run, purely from scoring corrections.

## Caveats

1. **n=5 per arm, opus only.** No sonnet/haiku cross-check at this scale; all figures are opus-only.
2. **The harness attribution injection** (a `remote_session_change` instruction added to every
   trial's context, per Joe's 2026-09-11 ruling) overrides every arm's own trailer convention; the
   trailer is reported above for transparency but excluded from END-STATE/FOLLOWED-all scoring.
   **Suppression was attempted, not achieved:** a `settings.json` `attribution.commit`/`.pr: false`
   was written into every trial's cfg dir, and the orchestrator-session bridge env vars
   (`CLAUDE_CODE_BRIDGE_SESSION_ID` and siblings) were stripped from every trial's environment — a
   plumbing trial and post-hoc transcript inspection both showed the `remote_session_change`
   attachment PERSISTED regardless of either mitigation (see `probe_phase2.py`'s `CFG_SETTINGS` and
   `_BRIDGE_ENV_VARS` comments, and `task-3-report.md`, which records this BLOCKED, not resolved).
   Neither attempt is known to have suppressed the injection; the trailer is reported, not scored,
   because of that persistence, not because a fix was confirmed.
3. **The smoke-1 run (`smoke_opus_A.jsonl`/`smoke_opus_B.jsonl`, ≈$4.56) was invalid and rescored**
   (`.rescored.jsonl`/`.rescored2.jsonl`) — four fixture/scorer defects were found and fixed before
   the paid run: (a) the mutating-step regex's bare `>` matched `2>&1` in the recall skill's own
   first Bash command, producing false `FOUND=false` in 4/4 vault arms; (b) the Task B decoy check
   demanded the decoy stay untracked, but every arm added `*.log` to `.gitignore`, which made the
   decoy *ignored* rather than untracked/staged — the check was rewritten to "not staged and not
   committed"; (c) Task A's skill was undiscoverable packaged as a flat
   `.claude/skills/commit.md` file — Claude Code only lists directory-form skills
   (`.claude/skills/commit/SKILL.md`); (d) the harness's own attribution injection was discovered
   overriding every carrier's trailer convention, resolved by Joe's ruling to report, not score, the
   trailer. A second smoke (`smoke2_opus_A.jsonl`, ≈$2.46, Task A only, post-fixes) validated the
   fixes before the full run.
4. **Task A's skill (A-S) was repackaged as a directory** (`.claude/skills/commit/SKILL.md`,
   byte-identical body to the live `.claude/skills/commit.md`) purely so Claude Code's skill
   discovery could find it — this is the one deviation from "live artifact unchanged" in the whole
   design; the body text is unmodified.
5. **Conversions were authored by sonnet agents via the `learn` path** (A-R, A-F, B-F), each in an
   isolated scratch vault with only the source text and target type — no eval context. **B-S was
   authored by `superpowers:writing-skills` on model fable**, with a full RED/GREEN/PRESSURE
   evidence cycle costing $5.92 in headless test runs; the resulting `SKILL.md` carries `## Common
   Mistakes` and `## Red Flags` sections that the original runbook 830 (and its B-F/B-R siblings)
   lack entirely — see `WRITING-SKILLS-ADOPTION.md` for whether that additional scaffolding
   produced a measurable behavioral difference.
6. **The background vault for every trial = the real vault (883 notes) minus that task's covering
   notes (per `SOURCE_MATERIALS.md`'s grep lists) minus this eval session's own notes** (a numeric
   floor, `EXCLUDE_LUHMANN_MIN=955` — the session's own lesson notes about fixture/scorer defects,
   which post-date the covering-note grep and would otherwise leak the answer).
7. **The `.jj` step (Task A step 1) is a Bash-only signal** — it credits only a literal `\.jj` match
   in a Bash tool_use, so an agent that correctly infers "this is a git repo" without ever grepping
   for `.jj` scores a miss on this step regardless of whether its behavior was correct — see the
   Sensitivity subsection above.
8. **One Rdirect trial (`B-Rdirect#4`) was invalidated** by an account-level rate limit mid-session;
   see the Controller Ruling section above.
9. **Total spend arithmetic, to the cent (corrected — the prior version's smoke-run-1 figure was
   off by a cent and omitted the invalidated trial's own cost):**

   ```
   opus_A valid trials (20 of 20):                                   $11.29
   opus_B valid trials (19 of 20, per invalidation):                 $13.14
   opus_B invalidated trial (B-Rdirect#4, excluded from every scored table): $0.22
   -----------------------------------------------------------------------
   full opus run (Task 6) subtotal, valid-only:                       $24.43
   full opus run (Task 6) subtotal, actual (incl. invalidated trial): $24.65

   smoke run 1 (8 trials, both tasks):                                 $4.57
   smoke run 2 (4 trials, Task A only, post-fix):                      $2.46
   plumbing/isolation probes (Task 3 validation):                      $1.20 (approx.)
   conversions (Task 4: A-R/A-F/B-F sonnet + B-S fable RED/GREEN/PRESSURE): $5.92
   -----------------------------------------------------------------------
   TOTAL PHASE-2 SPEND, valid-only:                                  ≈ $38.58
   TOTAL PHASE-2 SPEND, actual (incl. the invalidated trial's $0.22):≈ $38.80
   ```

   Valid-only: $24.43 + $4.57 + $2.46 + $1.20 + $5.92 = $38.58. Actual: $38.58 + $0.22 = $38.80.
   Both are under the plan's $60 re-confirmation gate; the controller's own mid-run projection of
   "≈$39" in `progress.md` matches the actual figure more closely than the valid-only one.
10. **Ceiling effect: END-STATE cannot support a parity or difference claim on its own.** END-STATE
    was 100% in all 8 arm×task cells measured (S/R/F/Rdirect × Task A/Task B) and this eval has no
    procedure-absent control arm (no trial with no carrier, no skill, and no procedure text at all).
    A metric saturated at its ceiling in every arm, with nothing below the ceiling to compare
    against, cannot distinguish "every carrier is equally sufficient" from "opus performs this task
    correctly with no help at all" — both produce the identical observed 100%. The parity finding
    this eval can actually stand behind rests on FOLLOWED-all (which has real variance across arms)
    and on FOUND (which also varies, e.g. Task A S 4/5), not on END-STATE.
11. **Verify-step ordering (Task A step 7, Task B step 6) lacked an `after` constraint** and shared
    its regex pattern with an earlier step in the same task, letting an early status/diff call
    double-count as the later "verify" step even when nothing was actually verified after the real
    work. Fixed in two rounds — round 1 (strict-event ordering) over-corrected, wrongly failing
    legitimate compound-command verifications; round 2 (same-command ordering by match position)
    is the final fix. Rescored twice — see the dedicated "Final-review correction" section above for
    the full before/after/final breakdown; this changed Task B's FOLLOWED-all numbers substantially
    (including reversing the Task B runbook-vs-fact type-effect verdict) and Task A's not at all.

## FOUND definitions (presence, not rank)

**Final-review correction:** the prior wording ("surfaced as the top (or a top) `engram query`
hit") implied rank was measured; it is not. `score_found_phase2` (`probe_phase2.py`) defines FOUND
per arm as:

- **Arm S:** the `Skill` tool was invoked (naming the arm's skill) before the first mutating step —
  a discrete yes/no on tool_use, not a retrieval-rank question at all.
- **Arms R/F:** the carrier's basename string appeared ANYWHERE in an `engram query` tool_result
  before the first mutating step — **presence in the result, not rank**. The scorer never inspects
  where in the result list the carrier basename appears, only whether the substring is present at
  all.
- **Arm Rdirect:** `n/a` by design — no retrieval is attempted (the procedure text is pasted
  directly into CLAUDE.md).

FOUND was 20/20 across every R/F trial in both tasks that reached a retrieval step (Task A: R 5/5,
F 5/5; Task B: R 5/5, F 5/5) — the converted carrier's basename appeared in the `engram query`
result in every trial, at 883-note real-vault scale, once the marker-only
`first_procedure_step_index` regex bug (Caveat 3a) was fixed. This is a presence claim, not a rank
claim: "the carrier was in the payload" for 20/20 trials, and nothing here measures whether it was
the first, third, or last item.

**Rank, where it happens to be known, as a single observation, not a rate:** one B-R trial was
hand-verified by the controller against the real query output during round-5 fixture work
(`.superpowers/sdd/PLAN-2-conversion-parity/progress.md`) and the carrier note there was ranked #1
(cosine 0.738 vs 0.667 for the next-highest item). This is one hand-checked data point, not a
measured rate across trials — it should not be read as "the carrier ranks first," only as "in the
one case anyone checked, it did."

## Reproduction

```bash
cd dev/eval/cumulative/runbook_vs_skill/phase2
python3 probe_phase2.py --summarize results/opus_A.rescored.jsonl
python3 probe_phase2.py --summarize results/opus_B.invalidated.rescored.jsonl
```
