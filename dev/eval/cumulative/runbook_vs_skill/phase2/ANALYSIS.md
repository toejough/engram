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

## Final-review correction: verify-step ordering (three rounds), and why every table below is from the rescored files

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

**Round 2 fixed the same-event case, but incompletely.** It accepted a match in the SAME Bash event
whose own regex match starts at a LATER position than the referenced step's match — but it located
that later match with `pattern.search()`, which only ever returns the FIRST (leftmost) occurrence
in the string. Five real Task B trials (`B-S#0/1/2`, `B-F#1/3`) run a compound command shaped like
`git status --porcelain -uall && git add <paths> && git diff --cached --name-only && git status
--short`: the verify step's pattern (`git (diff --cached|status)`) first matches the LEADING `git
status` — which sits BEFORE the `git add` staging match — so `search()` never sees the later,
genuinely-after `git diff --cached`/`git status --short` occurrence in the same command, and the
step wrongly failed even though a legitimate later match existed. **Round 3 fixes this by scanning
every occurrence via `pattern.finditer()`** and taking the first one whose start position is after
the referenced step's match, instead of stopping at the first (possibly too-early) occurrence.

**A second, independent bug surfaced during the same review: an unmatched referenced step made
`after` vacuous.** `match_idx.get(after_n, -1)` defaults to `-1` when the referenced step never
matched anything — but `-1` satisfies the "strictly later than" check against ANY event, so a
"verify" step with nothing to verify after would wrongly match anywhere in the transcript, as if
there were no ordering constraint at all. **Ruling (round 3):** if the referenced step is unmatched,
the dependent step is now unconditionally FALSE. This surfaced a real instance: `B-R#2` stages with
`git add -- .gitignore scripts/build.sh testdata/fixture.json` — the explicit-path-terminator
(`--`) form — which the fixture's step-5 regex did not recognize (it required `git add` to be
immediately followed by one of the three paths, with no room for `--`). **Step 5's regex is now
widened** to `git\s+add\s+(--\s+)?(...)`, accepting this legitimate form; this widening turned out
to matter far more broadly than just `B-R#2` — the `--` form appears across multiple trials in
multiple arms: Task B valid trials using `git add -- <paths>` — S 1 of 5, R 1 of 5, F 1 of 5, Rdirect 3 of 4. The widening therefore moved the control arm most: Rdirect 1/4 → 4/4.

Both result files were rescored against the final (round-3) fix (`--rescore results/opus_A.jsonl
--out results/opus_A.rescored.jsonl`; `--rescore results/opus_B.invalidated.jsonl --out
results/opus_B.invalidated.rescored.jsonl`) using the kept trial repos/transcripts, OVERWRITING the
round-1 and round-2 rescored files. **Every table in this document, the README's phase-2 section,
and the LEDGER row is now from these final `.rescored.jsonl` files** — the original
`opus_A.jsonl`/`opus_B.invalidated.jsonl` remain committed unchanged, as the un-rescored record of
what actually ran.

**Before/after per arm (FOLLOWED-all, k/n) — all four states: original run, round-1 overshoot,
round-2 (still incomplete), and round 3 (final):**

| task | arm | original (no `after`) | round 1 (strict-event, overshot) | round 2 (same-command, incomplete) | round 3 (final) |
|---|---|---|---|---|---|
| A | S | 0/5 | 0/5 | 0/5 | 0/5 |
| A | R | 1/5 | 1/5 | 1/5 | 1/5 |
| A | F | 2/5 | 2/5 | 2/5 | 2/5 |
| A | Rdirect | 3/5 | 3/5 | 3/5 | 3/5 |
| B | S | 4/5 | 0/5 | 1/5 | 5/5 |
| B | R | 4/5 | 1/5 | 4/5 | 5/5 |
| B | F | 3/5 | 0/5 | 2/5 | 4/5 |
| B | Rdirect | 1/4 | 0/4 | 1/4 | 4/4 |

**Task A is unchanged across every round** — every trial's step 7 verify call genuinely happened as
its own separate Bash event after step 6's commit call, and step 5 (Task A has no equivalent
staging-regex-width issue) never hit either round-3 bug, so no round of this fix touches Task A.
**Task B moved three times, and the direction of the story changed each time.** Round 1
over-corrected (collapsed every arm). Round 2 partially recovered but still undercounted every
arm — most severely S and F, whose real compound commands happen to lead with an early `git status`
before the `git add`/`git diff --cached` pair, which round 2's leftmost-match search couldn't see
past. **Round 3 (the actual fix) shows nearly every Task B trial, in every arm, genuinely staged
with explicit paths AND verified afterward** — the earlier "R clears the bar over F" reading from
round 2 was ITSELF a scoring artifact, not a real finding; it is retracted below. The ONLY
remaining Task B FOLLOWED-all miss is `F#3`, on step 3 (gitignore-narrowing correctness), unrelated
to staging/verification ordering.

The apparent Task-B-only "carrier structure" hypothesis floated after round 2 (that the runbook
carrier's two adjacent numbered steps caused agents to chain stage+verify more than the fact
carrier's dense predicate clause did) is **retracted along with the reversal it was explaining** —
round 3 shows S, R, and Rdirect all did this at 4/4 or 5/5 rates, and F did it at 4/5; there is no
carrier-type pattern left to explain.

## `--summarize` output, reproduced from a live run (final, round-3 rescore)

The blocks below are reproduced from actually running the command shown, against the final
`.rescored.jsonl` files — not hand-typed; the JSON decomposition blocks are compacted onto one line each for width, values unchanged.

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
FOLLOWED all-steps (k/n)    5/5             5/5             4/5             4/4
FOLLOWED (mean k/N)         6.00/6          6.00/6          5.80/6          6.00/6
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
      "followed_all": "5 of 5"
    },
    "note_quality_given_delivery_F": {
      "end_state": "5 of 5",
      "followed_all": "4 of 5"
    },
    "note_ceiling_Rdirect": {
      "end_state": "4/4",
      "followed_all": "4/4"
    },
    "shim_loss_total": {"pp_diff": 0.0, "rdirect": "4/4", "r": "5/5"},
    "shim_loss_given_found": {"pp_diff": 0.0, "rdirect": "4/4", "r_given_found": "5/5"},
    "note_quality_F_total": {"pp_diff": 0.0, "f": "5/5", "rdirect": "4/4"},
    "note_quality_F_given_found": {"pp_diff": 0.0, "f_given_found": "5/5", "rdirect": "4/4"},
    "type_effect": {
      "end_state_total": 0,
      "end_state_given_found": 0,
      "followed_all_total": 1,
      "followed_all_given_found": 1
    },
    "parity": {
      "S_vs_R": {
        "end_state": "cant_distinguish",
        "followed_all": "cant_distinguish"
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

**Retraction: the round-2 "R clears the bar over F" finding does NOT survive round 3.**
`type_effect.followed_all_total` is now `1` (R 5/5 vs F 4/5), back within the ±1 noise floor —
`cant_distinguish`. `parity.S_vs_R` is now `cant_distinguish` (both at ceiling: S 5/5, R 5/5) —
the round-2 "`better` for R, gap 3" reading is also retracted; it was driven by the same
leftmost-match bug being fixed here. Task B is now close to fully saturated on FOLLOWED-all: only
`F#3` misses anything (step 3, a gitignore-narrowing-correctness check unrelated to
staging/verification ordering). See the Decision section below — the recommendation returns to the
pre-round-2 reading: no decision-relevant comparison clears the bar in either task.

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
after all three rounds of the verify-step ordering fix** (step 6 no longer double-credited by step
4's earlier `git status` call; correctly credited when a LATER occurrence of its pattern runs in
the SAME compound command as step 5's staging, even past an earlier unrelated occurrence; and step
5's regex widened to accept the `git add -- <paths>` explicit-path-terminator form — see the
Final-review correction section above):

| arm | step 3 missed | all other steps |
|---|---|---|
| S | 0/5 | 0 missed |
| R | 0/5 | 0 missed |
| F | 1/5 | 0 missed |
| Rdirect (valid, n=4) | 0/4 | 0 missed |

Steps 1, 2, 4, 5, and 6 are now missed by NO trial in ANY arm. The only remaining Task B miss
anywhere is `F#3` on step 3 (the gitignore-narrowing-correctness `repo_state` check), unrelated to
staging or verification ordering. **This is a materially different picture from every prior
version of this document**: what looked like real behavioral spread on steps 5/6 across S, R, F,
and Rdirect turned out to be almost entirely a scoring artifact (the leftmost-match bug plus the
`--`-form regex gap) — the underlying agent behavior was already close to uniformly correct.

The invalidated `B-Rdirect#4` trial's own `followed_steps` shows only step 1 credited (steps
2/3/4/5/6 all missed) — consistent with the session being cut off 5 turns in, before any real
procedure work; it is excluded from the table above and is not evidence of any defect, only of the
mid-session kill.

**Reading:** Task A misses are dominated by step 1 (the `.jj` VCS check) across every arm, heaviest
in S (5/5) and lightest in F/Rdirect (2/5 each); step 3 (git log) is a smaller secondary miss in
S (2/5) and F (1/5). **Task B is now essentially at ceiling on FOLLOWED-all across every arm**
(S 5/5, R 5/5, Rdirect 4/4, F 4/5) — see the Ceiling effect caveat, which now applies to more than
just END-STATE in this task. Note also the standing design asymmetry between the two tasks
(Caveat 12): Task B's rubric was derived from the SAME source text that B-R carries verbatim (the
only unconverted carrier in either task), while Task A's original is the SKILL (S), which is the
worst performer on Task A's FOLLOWED-all — the "original wins its own rubric" pattern one might
worry about from Task B's confound does not actually hold in Task A.

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
effect caveat below). **On FOLLOWED-all (final, round-3 rescore): Task B now shows R and Rdirect
both at ceiling (R 5/5, Rdirect 4/4 — 100% each, gap 0, `cant_distinguish`); Task A still shows
Rdirect ahead of R by 2 (R 1/5 vs Rdirect 3/5)**, unaffected by any round of this fix. A prior
version of this section reported Task B as R ahead of Rdirect by 3 (an "opposite direction from
Task A" finding) — that reading was itself a product of the same leftmost-match scoring bug fixed
above and is retracted. The only standing R-vs-Rdirect FOLLOWED-all gap in this data is Task A's,
where the shim (Rdirect) outperforms the retrieval arm (R) by 2 trials.

**Hypothesis, not established:** the retrieval path's own procedure — recall's Step-0 plan
statement plus the act of reading a single, focused note during Step 2.5 — may cause more
attentive application of the note's content than pasting the same text directly into a long
CLAUDE.md, where it competes with everything else already loaded (the marker, the task prompt, the
harness's attribution injection, and whatever else the trial's CLAUDE.md carries). This was
originally offered to explain an apparent Task-B "shim did worse" pattern that no longer exists
(see above); with Task B now at ceiling and Task A pointing the OPPOSITE way (shim ahead, not
behind), this hypothesis has no supporting data left in this eval and should not be carried forward
without a fresh test.

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

**Retraction: the round-2 "task-dependent split" reading below does NOT survive round 3 and is
withdrawn.** Round 2 reported Task B's runbook clearing the 2-trial bar over the fact and
recommended keeping the runbook type for Task B specifically. That reading rested on a leftmost-
match scoring bug (fixed in round 3, see above) that undercounted S and F's genuine step-6 passes
far more than R's. With the bug fixed, Task B is now close to fully saturated on FOLLOWED-all
across every arm, and the round-2 "reversal" evaporates along with the carrier-structure hypothesis
that was built to explain it.

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
- **FOLLOWED-all, Task B (final, round-3 rescore):** S 5/5, R 5/5, F 4/5, Rdirect 4/4 — essentially
  at ceiling in every arm. **S vs R is `cant_distinguish` (gap 0)** — the round-2 "`better` for R,
  gap 3" reading is retracted (it was the leftmost-match bug). **S vs F is `cant_distinguish`
  (gap 1)**.
- **R vs F (type effect, the bar Joe's rule actually turns on for "runbook needs special build")
  — the two tasks now AGREE:** Task A: R−F FOLLOWED-all = −1 at N=7 / +1 at the
  sensitivity-corrected N=6 — small, sign-unstable, never reaching 2. **Task B (final): R−F
  FOLLOWED-all = 5−4 = +1 — also within the ±1 noise floor, `cant_distinguish`.** The round-2
  reading of Task B's R−F gap as +2 is retracted along with the scoring bug that produced it.
  **Runbook never clears the 2+ trial bar over fact on any metric, in either task.**

**Conclusion the data supports — no decision-relevant comparison clears the pre-registered 2-trial
bar in either task; the recommendation is uniform, not task-dependent.** Joe's rule turns on
exactly two comparisons: S-vs-F ("vanilla fact matches the skill") and R-vs-F ("runbook needs
special build to survive"). In BOTH tasks, once Task A's headline is read at its
sensitivity-corrected value, both decision-relevant comparisons are `cant_distinguish` — no
2+-trial finding either way, in either task. Applying the rule's logic: the fact matches the skill
in the weak sense of a null result (not a proven match — n=5 is underpowered to confirm
equivalence), and the runbook shows no evidence anywhere in this data that it "needs special
build" to survive — it never clears the 2+ trial margin over the fact on any metric, in either
task. **Recommendation: drop the distinct runbook type for these generic procedures; keep
facts+feedback+shim** — on the same weak (can't-distinguish) grounds Task A alone supported before
round 2's now-retracted reversal, not on a stronger "fact even exceeds the skill" or "runbook beats
fact in Task B" claim; neither survives final scoring.

**Standing confound, unrelated to the scoring bugs above but worth naming wherever Task B's R-vs-F
comparison is read (Caveat 12):** Task B's rubric (`done_when` + `steps.json`) was derived directly
from vault note 830's own text, and B-R is the only carrier in either task that is byte-identical
to that same source (F/S are conversions of it; R is the original). If anything this would bias Task B's rubric
toward crediting R, yet R and F end up statistically indistinguishable anyway (gap 1) — the
confound did not manufacture a spurious win here, but it remains a reason not to treat Task B's
R-vs-F comparison as a clean test even before the scoring-bug history above. Task A's original is
the SKILL (S) instead, and S is Task A's WORST performer on FOLLOWED-all (0/5 raw, 3/5
sensitivity-corrected) — so "the original wins its own rubric" is not a pattern this eval's own
Task A supports, which somewhat blunts concern that Task B's confound alone explains anything.

**What the data does NOT support:**
- **END-STATE was at ceiling (100%) in all 8 arm×task cells (S/R/F/Rdirect × Task A/Task B), and
  there is no procedure-absent control arm** (no trial with no carrier, no skill, and no procedure
  text at all). A ceiling metric with no floor/absent-condition anchor cannot separate "every
  carrier works equally well" from "opus performs this task correctly with no carrier at all" —
  END-STATE alone cannot support either a parity claim or a difference claim here. **Task B's
  FOLLOWED-all is now ALSO close to ceiling** (S 5/5, R 5/5, Rdirect 4/4, F 4/5) — the parity
  finding this eval can actually stand behind rests mainly on Task A's FOLLOWED-all (which has real
  variance) and on FOUND, not on Task B's FOLLOWED-all or on END-STATE in either task.
- It does not show the fact is *causally superior* to the runbook or to the skill — every
  decision-relevant comparison, sensitivity-corrected, is `cant_distinguish` in both tasks; nothing
  in this data clears the pre-registered 2-trial bar in either direction.
- It does not generalize beyond **generic** (non-idiosyncratic) procedures — this eval was scoped
  exactly to fill the gap noted in vault note 853a; the runbook type's demonstrated wins on
  idiosyncratic content (prior LEDGER rows, e.g. `crowded-vault-capability-robustness`) are
  untouched by this finding.
- It does not establish anything about skill vs. memory-in-general: S itself performs comparably to
  R/F on every scored metric here (the "baseline usability" bar, S END-STATE ≥3/5, is cleared 5/5 in
  both tasks, though see the ceiling caveat above on what that clearance can and cannot mean) — this
  is a runbook-vs-fact type question, not a memory-vs-no-memory one.
- It is n=5 per arm; a single flipped trial in either direction can move a `cant_distinguish`
  verdict to `better`/`worse` or vice versa — and this document's own history is the clearest
  demonstration: Task B's FOLLOWED-all picture moved substantially THREE times, with no new trial
  run, purely from scoring corrections, before settling back near the original run's rough shape.

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
    work. Fixed in three rounds — round 1 (strict-event ordering) over-corrected, wrongly failing
    legitimate compound-command verifications; round 2 (same-command ordering by match position)
    fixed that but only checked the FIRST occurrence of the verify pattern in a command, missing a
    later legitimate occurrence when an earlier, unrelated occurrence of the same pattern preceded
    the referenced step's own match; round 3 (scan all occurrences via `finditer`, plus: an
    unmatched referenced step now forces the dependent step FALSE instead of vacuously passing, and
    step 5's regex was widened to accept the `git add -- <paths>` explicit-path form) is the final
    fix. Rescored three times — see the dedicated "Final-review correction" section above for the
    full before/after/final breakdown; round 3 changed Task B's FOLLOWED-all numbers substantially
    (retracting round 2's runbook-vs-fact reversal) and Task A's not at all.
12. **Task B's rubric has an original-carrier confound: B-R is the only UNCONVERTED carrier in
    either task.** Task B's `done_when` checks and `steps.json` were derived directly from vault
    note 830's own text — the SAME text B-R serves verbatim (B-R's carrier IS note 830 itself,
    never converted), while F and S are authored conversions of it. This could plausibly bias
    Task B's rubric toward crediting whichever arm happens to phrase things
    closest to the source; the final numbers show R and F statistically indistinguishable anyway
    (gap 1), so the confound did not manufacture a spurious result here, but it is a standing
    design asymmetry worth naming whenever Task B's R-vs-F (or R-vs-anything) comparison is read.
    Task A's original is the SKILL (S) instead, and S is Task A's WORST performer on FOLLOWED-all
    (0/5 raw, 3/5 sensitivity-corrected) — the "the original wins its own rubric" concern this
    confound might raise does not hold in Task A, where the original loses.

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

## Sonnet 5 pilot (2026-09-12): 10 runs per form, plus runs with no instructions

Joe asked for an equivalence test on sonnet to measure whether the findings from opus hold at a cheaper model. The pilot runs first to establish a baseline: 10 runs per form (skill, runbook note, fact note) on both tasks, plus 10 runs with no instructions for each task. The pilot measures whether sonnet 5 can even distinguish the forms before committing to a larger run.

### Task A (commit)

```
=== Task A ===
metric                      S               R               F               N               
--------------------------------------------------------------------------------------------
FOUND (k/n)                 4/10            8/10            8/10            n/a             
FOLLOWED all-steps (k/n)    0/10            0/10            1/10            0/10            
FOLLOWED (mean k/N)         5.90/7          6.00/7          6.00/7          5.80/7          
END-STATE (k/n)             10/10           10/10           10/10           10/10           
recall_fired (k/n)          10/10           10/10           10/10           10/10           
cost (mean USD)             $0.36           $0.35           $0.34           $0.37           
duration (mean s)           68              62              61              57              
valid (n)                   10/10           10/10           10/10           10/10           
trailer AI-Used (k/n)       1/10            0/10            0/10            0/10            
trailer Co-Authored-By (k/n)9/10            10/10           10/10           10/10           
  ^ reported, not scored — harness attribution injection overrides carriers
no-instructions baseline: FOLLOWED-all 0/10, END-STATE 10/10
```

### Task B (gitignore narrowing)

```
=== Task B ===
metric                      S               R               F               N               
--------------------------------------------------------------------------------------------
FOUND (k/n)                 9/10            10/10           10/10           n/a             
FOLLOWED all-steps (k/n)    10/10           10/10           10/10           1/10            
FOLLOWED (mean k/N)         6.00/6          6.00/6          6.00/6          5.10/6          
END-STATE (k/n)             10/10           10/10           10/10           10/10           
recall_fired (k/n)          10/10           10/10           10/10           10/10           
cost (mean USD)             $0.43           $0.45           $0.46           $0.32           
duration (mean s)           96              96              93              63              
valid (n)                   10/10           10/10           10/10           10/10           
no-instructions baseline: FOLLOWED-all 1/10, END-STATE 10/10
```

### Findings

**Commit task (Task A):** All forms (skill, runbook, fact) and the no-instructions baseline got the end result right — 10 of 10. They all missed the same step: checking for a `.jj` directory. This step only gets credited when it appears in a shell command; it's a Bash-only signal, not automatic inference. The skill was invoked 4 of 10 times. The runbook and fact notes were found 8 of 10. Conclusion: sonnet 5 does not need the instructions for this task, so it cannot separate the forms.

**Gitignore task (Task B):** The skill, runbook, and fact forms all followed every step 10 of 10 and got the end result right 10 of 10. The no-instructions baseline got the end result right 10 of 10 but only followed every step 1 of 10. They skipped the git check-ignore verification step in 9 of 10 cases. Conclusion: the instructions add exactly one verification step here, and all three forms deliver it equally. Retrieval found the notes in 20 of 20 cases (in the results; rank not measured).

### What this means for an equivalence test

At 10 runs per form there is no gap to measure. Both tasks have a no-instructions baseline at or near 100% on the end result. They cannot show a difference between forms. An equivalence claim (for example, within 20 points) needs tasks where the no-instructions baseline is well below 100% and about 40 runs per form to measure the underlying effect reliably.

### Costs

Task A: $14.16 (40 records × $0.36 mean). Task B: $16.67 (40 records × $0.42 mean). Dead batch (rate-limited, 35 of 40 runs killed mid-session): $2.28. Total: $33.11.

### Trailers

The skill wrote AI-Used in 1 of 10 cases. The runbook and fact notes never wrote AI-Used (both reported 0 of 10). This observation is not scored — the harness's own injected `Co-Authored-By` instruction took precedence in 9 of 10 cases (skill) and 10 of 10 (runbook and fact). See the main phase-2 analysis for the full caveat.

## Reproduction

```bash
cd dev/eval/cumulative/runbook_vs_skill/phase2
python3 probe_phase2.py --summarize results/opus_A.rescored.jsonl
python3 probe_phase2.py --summarize results/opus_B.invalidated.rescored.jsonl
python3 probe_phase2.py --summarize results/pilot_sonnet5_A.jsonl
python3 probe_phase2.py --summarize results/pilot_sonnet5_B.jsonl
```

## Shim-only follow-frame eval (`runbook-shim-follow-frame`, sonnet5, 2026-09-14/17)

A separate, later change (`openspec/changes/runbook-shim-follow-frame/`) revisited the same
runbook-vs-skill question this document's earlier checkpoints (above) were built to answer, with a
different design: the earlier checkpoints all ran the runbook/fact arms with the *existing*
`recall.md` guidance plus the recall/learn/write-memory *skills still installed* — a hybrid that
never tested Joe's actual end-state, where every skill becomes a runbook and the shim is the only
custom CLAUDE.md text. That change's design.md (D0–D8) builds a fresh, four-part shim
(`agent-instructions/guidance/shim.md`: bootstrap query, per-kind treatment with a runbook follow
frame, a general behavioral floor, and transitive wikilink-following) and a `--shim-only` harness
config with **no engram skills installed at all** — recall, learn, and write-memory exist only as
vault runbooks the shim's own first-query bootstrap must surface. D8 pre-registers the pass bar:
history-rewrite first at n=3, found 3, restated-as-plan 3, every step 3, end result within one
trial of the skill row's 2/3; question-stops are reported as instruction-clarity findings (a
question stop is a clarity signal, not a failure) and excluded from the scoreable population, not
scored as misses; bisect-before-fix and route follow once the bar holds.

### History-rewrite (task 4.3): MET, after two real fixes

The RED baseline (2.1) and shim GREEN rounds (2.3) built the shim itself; three rounds of task 4.3
then pressure-tested it against real runbook content:

- **Round 1** (before any fix): 0/2 scoreable trials after 1 legitimate question-stop, and
  end_state 0/6 across *both* R and F arms — traced to a pre-existing fixture bug, **#752**, not a
  shim or runbook-vs-fact defect.
- **Rerun** (after #752 fixed, $2.85): found 3/3, restated_as_plan 3/3, followed_all 3/3 — full
  parity with the skill row on every step-following metric. All 3 R trials hit the *same*
  legitimate step-6 question-stop (no identifiable remote host plus an ambiguous placeholder
  secret), leaving 0/3 scoreable for end_state. Informationally, `done_when_checks.sh` run directly
  against all 6 kept repos genuinely failed: R failed Check 6 (recoverability durability) in all 3.
  Root cause: runbook 840's `done_when` credited reflog-only recovery as sufficient, but
  `git-filter-repo`'s default post-rewrite gc prunes the reflog by default — the checker correctly
  refused to credit that as durable. Filed as **#753** (a design call between loosening the checker
  or tightening the runbook), not fixed in this round.
- **Rerun 2** (after #753's fix — runbook 840 rebuilt via `engram amend` to require a durable
  external artifact — a `git bundle`, a tag, or a plain-text SHA file — instead of crediting
  reflog/refs alone, $3.01): the *fix itself* introduced a fresh runbook-content gap: my own
  step-1 rewording led 2 of 3 trials to read the bundle as sufficient on its own and skip
  `--refs`/`--no-gc`, so filter-repo's default gc pruned the pre-rewrite objects locally even
  though the external bundle was present and valid — end_state failed on Check 5 (pre-rewrite
  object locally reachable), not Check 6. This was root-caused and fixed in the same session
  (runbook 840 amended again to say the bundle is additional insurance, never a reason to skip
  `--refs`/`--no-gc`) but not re-validated with a fresh run in that session.
- **Rerun 3** (validating the second fix, the 3rd and final attempt, $2.78): **D8 bar MET.**
  found 3/3, restated_as_plan 3/3, end_state 3/3 (the best possible outcome, 0 question-stops to
  exclude), verified directly against `done_when_checks.sh`'s own PASS output for all 3 R trials.
  Raw followed_all read 1/3, but a transcript read showed the other 2 trials ran the fully correct
  sequence using the hyphenated `git-filter-repo` invocation, which `steps.json` step 3's regex
  didn't match — a harness scorer gap, filed as **#754**, not fixed (out of scope; the true
  step-following rate this run is 3/3). Compared to the skill row (found 3/3, restated 3/3,
  followed_all 3/3, end 2/3): R matches or exceeds on every axis.

### Bisect-before-fix (task 4.4): MET, after one fix round

The first 4.4 attempt (run jointly with route, $7.67 combined) found 0/3 scoreable trials for
bisect-before-fix too, but for a different, genuine design-mismatch reason: runbook 846's step 6
read as an unconditional "attribute, don't fix," in tension with this fixture's task prompt ("fix
what's needed so the gate passes") and its own checker, which both require the pre-existing bug
actually fixed. Rather than flipping step 6 to "always apply the fix" (eroding the runbook's
scope-discipline default for the general case), the fix added a narrow exception: apply the
pre-existing fix, without asking, only when the task explicitly instructs making this specific
gate pass *and* the fix is small and clearly scoped — still filing the follow-up issue, and
preserving the attribute-only default for the abstract "a gate is failing" case.

The rerun ($2.87) showed the automated fields alone (`question_stop=True` on all 3 R trials) look
identical to round 1's failure shape, but a full transcript read found this is **not** the same
failure: all 3 trials correctly reasoned through the new exception and 2 of 3 (R-0, R-2) fully
executed on it — bisected, fixed, restored HEAD, verified `gate.sh` green — and only *then*, after
the task was already done, asked a genuinely separate administrative question (this fixture repo
has no issue tracker to file step 6's required follow-up against). The harness's `question_stop`
scorer flags this identically to a genuine pre-completion stall because it only checks whether a
mutation follows the *last* ambiguity-posing text, not whether one preceded it — filed as part of
**#755**, alongside the residual "no issue tracker" fixture-completeness gap; neither fixed, out of
scope for this narrow-clause fix. R-1 is the one genuine pre-completion stop (asked permission
before applying the fix, left `bar.py` unfixed).

**D8 bar MET on both readings**: raw numbers alone clear it without any exclusion (found 3/3,
restated_as_plan 3/3, end_state 2/3 — matching the skill row's 2/3 exactly); reclassifying by
legitimacy (excluding only R-1's genuine stop) gives an even stronger 2/2 on followed_all and
end_state. No further rerun was needed.

### Route (task 4.4): not met across 6 rounds, but every shim/runbook/checker defect found is fixed

Route never reached a MET state, but its six rounds (full history in tasks.md 4.4) form a clean
case study in the "collect the full list, fix once" principle rather than an argument against the
frame itself. Round-by-round, each fix targeted a confirmed, transcript-verified cause and stuck:

| round | spend | found | followed_all / end_state (scoreable) | defect found & fixed |
|---|---|---|---|---|
| 1 (joint w/ bisect) | ~$2.85 (route share of $7.67) | 0/6 | n/a | none this round — flagged for Joe |
| 2 (`--keep` diagnostic) | $4.66 | 0/6 | n/a | root-caused: query phrased around task surface wording, never the runbook's own situation vocabulary |
| 3 (shim.md step-1 fix) | $4.99 | **6/6** | 0/6 | fixed: shim step-1 now anchors abstraction *before* deciding whether to delegate, with a worked delegation-shaped example |
| 4 (checker-only fixes) | $4.76 | R 3/3, F 2/3 | 0/6 (all 3 R stopped on a write-memory-availability question) | fixed 2 genuine checker gaps: step-10 regex missed the create-new-aggregate branch; step-9 tag-quoting regex rejected quoted `--tag` values |
| 5 (carrier-fixture fix) | $4.58 | R 3/3, F 3/3 | 2/3 followed_all, 0/3 end_state | fixed: `Route-R` had 4 bare, un-wikilinked "write-memory" references task 3.4's sweep had missed on this one extra carrier |
| 6 (`--tags` fix, absolute final) | $5.20 | R 3/3, F 3/3 | 2/3 followed_all (67%, at the skill row's own level), **0/3 end_state** | fixed: both route carriers' evidence-write intro named no real tag flag, inviting the plausible but wrong guess `--tags`; named the real `--tag <family>/<value>` explicitly |

Round 6, the largest fully-assessable population (0 question-stops, 3/3 scoreable), is the clean
read: **followed_all 2/3 (67%) matches the skill row's own reference rate; end_state 0/3 (0%) is
well below the D8 "within one trial of 2/3" bar under any reading** (a 0/3 vs. 2/3 skill-row target
is a 2-trial gap, not one). The blocker, confirmed identically across rounds 4–6 by direct
transcript read: the agent writes the aggregate's evidence reference as plain prose instead of the
`[[wikilink]]` bracket syntax the route runbook's own aggregate-write templates already show,
verbatim, directly above the field being filled in, in both its "amend an existing aggregate" and
"create a new aggregate" branches. This is a genuine agent-following miss — the runbook is
unambiguous and the correct syntax is right there — not a shim, runbook-content, or checker defect,
and per the task's own decision rule ("if the runbook already says to wikilink and the agent just
didn't, report it, don't fix the checker") it was left as a reported finding, not patched into the
checker.

Every other defect route surfaced across all 6 rounds was investigated to a confirmed root cause
and fixed: the retrieval-relevance query-phrasing gap (round 3), the step-10 create-aggregate regex
and step-9 tag-quoting regex (round 4), the `Route-R` carrier's un-wikilinked write-memory
references (round 5), and the `--tags`-vs-`--tag` ambiguity in both route carriers (round 6). None
of those fixes were reverted or needed a second pass once applied — each is confirmed, by direct
transcript re-read of the following round's fresh trials, to have actually stopped recurring
(0/3 write-memory-availability stops in round 5 vs. 3/3 in round 4; 0/3 `--tags` stops in round 6
vs. 1/3 in round 5). Route's D8 shortfall is therefore attributable to exactly one remaining cause,
not an unexamined pile of noise.

### Decision (task 4.5), per D8

**Does the runbook kind, with the shim's follow frame, earn its keep over a plain fact, once the
follow frame exists?** Yes, on the evidence available: history-rewrite and bisect-before-fix both
clear the D8 bar cleanly (one MET after 3 rounds' worth of real fixture/checker fixes, one MET
after 1), matching or exceeding the skill row on found, restated-as-plan, and end-state, with the
fact arm in both tasks lagging behind on `--refs`/`--no-gc` adoption (history-rewrite) and on
followed_all/end_state generally (bisect-before-fix, both raw runs). Route's result does not
contradict this: found and restated-as-plan both reached skill-row parity once its five genuine
defects were fixed, and the one remaining blocker — the wikilink-vs-plaintext miss — is a question
about how carefully an agent copies a shown syntax, not about whether a runbook's follow frame is
weaker than a fact's absence of one; a fact carrier would face the identical templated-wikilink
requirement and no fact-arm evidence in this eval suggests it would do any better on it (route's F
arm never reached the aggregate-write step cleanly enough to compare directly).

**Does the shim-only configuration (no engram skills, runbook notes only) reach parity with the
skill arm?** Same answer, same caveat: 2 of 3 tasks reach full or near-full parity with the
skill-row reference numbers from the original checkpoint (found 3/3, restated 3/3, followed_all
3/3, end 2/3); route's scoreable population (2/3 followed_all, matching the skill row) shows the
frame itself is being followed once an agent gets there, but its end-state comparison is
undecided, not negative — 0/3 is a real shortfall against the bar, but the cause is a specific,
nameable, non-runbook-vs-fact behavior, not a demonstration that shim-only guidance underperforms
skills in general.

**Recommended path forward:** proceed with Joe's stated goal (proposal.md's Why: every skill,
including recall and learn, becomes a runbook, with the shim as the only custom CLAUDE.md text).
The evidence clears that bar on 2 of 3 real procedures outright and is directionally supportive,
not contradictory, on the third. File the route wikilink-following gap as a named follow-up
(task 5.2) rather than treating it as a reason to hold the frame back — it is the kind of
agent-behavior miss that shows up in any note-following task, runbook or not, and the fix belongs
either in the runbook's own wording (make the wikilink requirement even more explicit) or in future
shim wording, not in reversing this change's core design decision.

### Total spend

Summed to the cent from every result file's `total_cost_usd` field (`2.1_redbaseline`,
`2.3_green_round{1,2,3}`, `4.3_stage1*`, `4.4_stage2*`):

| phase | files | spend |
|---|---|---|
| 2.1 RED baseline | 1 | $0.48 |
| 2.3 GREEN (3 rounds) | 3 | $3.64 |
| 4.3 history-rewrite (4 runs) | 4 | $11.75 |
| 4.4 bisect-before-fix (2 runs) | 2 | $5.46 |
| 4.4 route (6 runs) | 6 | $29.27 |
| **total** | 16 | **$50.60** |

### Reproduction (shim-only eval)

```bash
cd dev/eval/cumulative/runbook_vs_skill/phase2
python3 probe_phase2.py --summarize results/2.1_redbaseline_sonnet5_noskills-currentguidance_history-rewrite.jsonl
python3 probe_phase2.py --summarize results/2.3_green_round3_sonnet5_shimonly_history-rewrite.jsonl
python3 probe_phase2.py --summarize results/4.3_stage1_rerun3_sonnet5_shimonly_history-rewrite.jsonl
python3 probe_phase2.py --summarize results/4.4_stage2_rerun_sonnet5_shimonly_bisect-before-fix.jsonl
python3 probe_phase2.py --summarize results/4.4_stage2_rerun5_sonnet5_shimonly_route.jsonl
```
