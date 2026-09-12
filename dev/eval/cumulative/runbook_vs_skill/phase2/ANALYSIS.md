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

## `--summarize` output, verbatim

### `python3 probe_phase2.py --summarize results/opus_A.jsonl`

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
    "shim_loss_total": 0,
    "shim_loss_given_found": 0,
    "note_quality_F_total": 0,
    "note_quality_F_given_found": 0,
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

### `python3 probe_phase2.py --summarize results/opus_B.invalidated.jsonl`

```
=== Task B ===
metric                      S               R               F               Rdirect
--------------------------------------------------------------------------------------------
FOUND (k/n)                 5/5             5/5             5/5             n/a
FOLLOWED all-steps (k/n)    4/5             4/5             3/5             1/4
FOLLOWED (mean k/N)         5.80/6          5.80/6          5.60/6          5.25/6
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
      "followed_all": "3 of 5"
    },
    "note_ceiling_Rdirect": {
      "end_state": "4/4",
      "followed_all": "1/4"
    },
    "shim_loss_total": -1,
    "shim_loss_given_found": -1,
    "note_quality_F_total": 1,
    "note_quality_F_given_found": 1,
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
5 explicit staging, 6 verify) — S/R/F n=5/arm valid, Rdirect n=4/4 valid + 1 invalidated:

| arm | step 3 missed | step 5 (staging) missed | all other steps |
|---|---|---|---|
| S | 0/5 | 1/5 | 0 missed |
| R | 0/5 | 1/5 | 0 missed |
| F | 1/5 | 1/5 | 0 missed |
| Rdirect (valid, n=4) | 0/4 | 3/4 | 0 missed |

The invalidated `B-Rdirect#4` trial's own `followed_steps` shows only step 1 credited (steps
2/3/4/5/6 all missed) — consistent with the session being cut off 5 turns in, before any real
procedure work; it is excluded from the table above and is not evidence of a step-5 defect, only of
the mid-session kill.

**Reading:** Task A misses are dominated by step 1 (the `.jj` VCS check) across every arm, heaviest
in S (5/5) and lightest in F/Rdirect (2/5 each); step 3 (git log) is a smaller secondary miss in
S (2/5) and F (1/5). Task B misses concentrate on step 5 (explicit staging) — once each in S/R/F,
and 3 of 4 valid Rdirect trials (the shim arm, with no vault carrier or retrieval step, misses
explicit staging hardest).

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

| population | Task A shim loss (Rdirect END-STATE − R END-STATE) | Task B shim loss |
|---|---|---|
| total (all valid) | 5 − 5 = **0** | 4 − 5 = **−1** |
| given-found (R found=true subset) | 5 − 5 = **0** | 4 − 5 = **−1** |

**Measured: shim loss ≤ 0 in both tasks** — the retrieved runbook (R) did at least as well as the
pasted-into-CLAUDE.md runbook text (Rdirect) on END-STATE in both tasks, and strictly better in
Task B (Rdirect end-state 4/4 valid but only because the failing case was invalidated out, vs R's
untouched 5/5; more strikingly on FOLLOWED-all, Task B's pasted control did **worse**: Rdirect 1/4
valid vs R 4/5 — the shim arm without a retrieval step or a vault carrier performed *worse* than the
arm that had to retrieve the note first).

**Hypothesis, not established:** the retrieval path's own procedure — recall's Step-0 plan
statement plus the act of reading a single, focused note during Step 2.5 — may cause more
attentive application of the note's content than pasting the same text directly into a long
CLAUDE.md, where it competes with everything else already loaded (the marker, the task prompt, the
harness's attribution injection, and whatever else the trial's CLAUDE.md carries). This is offered
as one explanation for why the shim (no-retrieval) arm did not out-perform retrieval, contrary to
the naive expectation that removing a retrieval step could only help; it is not independently tested
here.

## Decision, per Joe's rule

**Rule:** "runbook needs special build → type survives; vanilla fact matches the skill → drop
runbook, keep facts+feedback+shim."

**What the data supports at n=5:**
- **END-STATE:** S vs R and S vs F are `cant_distinguish` in **both** tasks (Task A: 5/5 all three;
  Task B: 5/5 S/R/F, 4/4 valid Rdirect) — the vanilla fact matches the skill on END-STATE
  everywhere measured.
- **FOLLOWED-all:** Task A — S vs F is `better` for F (F 2/5 vs S 0/5, a 2-trial gap, the pre-
  registered bar); S vs R is `cant_distinguish` (R 1/5 vs S 0/5, gap 1). Task B — both S vs R and
  S vs F are `cant_distinguish` (R 4/5, F 3/5, S 4/5 — all within 1 of each other).
- **R vs F (type effect):** within 1 trial of each other on every metric in both tasks (Task A:
  R−F FOLLOWED-all = −1, i.e. F is 1 ahead of R, not the 2+ the pre-registered "runbook > fact"
  bar requires; Task B: R−F FOLLOWED-all = +1, R 1 ahead of F, same sub-bar gap). **Runbook never
  clears the 2+ trial bar over fact on any metric in either task — in Task A the fact is
  numerically ahead of the runbook, not behind it.**

**Conclusion the data supports:** the fact note matches the skill (END-STATE, both tasks) and, in
Task A, the fact even exceeds the skill on FOLLOWED-all (F 2/5 vs S 0/5) — the "vanilla fact matches
the skill" branch of Joe's rule is satisfied in both tasks. The runbook shows no evidence it "needs
special build" — it never beats the fact by the pre-registered 2+ margin on any metric, in either
task. Applying the rule literally: **drop the distinct runbook type for these generic procedures;
keep facts+feedback+shim.**

**What the data does NOT support:**
- It does not show the fact is *causally superior* to the runbook — every comparison is either
  `cant_distinguish` or within 1 trial, which is the ±1-trial noise floor, not a proven ranking. F's
  2-trial edge over S on Task A FOLLOWED-all is the one comparison that clears the pre-registered
  bar; nothing here clears the 2-trial bar in the runbook's favor.
- It does not generalize beyond **generic** (non-idiosyncratic) procedures — this eval was scoped
  exactly to fill the gap noted in vault note 853a; the runbook type's demonstrated wins on
  idiosyncratic content (prior LEDGER rows, e.g. `crowded-vault-capability-robustness`) are
  untouched by this finding.
- It does not establish anything about skill vs. memory-in-general: S itself performs comparably to
  R/F on every scored metric here (the "baseline usability" bar, S END-STATE ≥3/5, is cleared 5/5 in
  both tasks) — this is a runbook-vs-fact type question, not a memory-vs-no-memory one.
- It is n=5 per arm; a single flipped trial in either direction can move a `cant_distinguish`
  verdict to `better`/`worse` or vice versa.

## Caveats

1. **n=5 per arm, opus only.** No sonnet/haiku cross-check at this scale; all figures are opus-only.
2. **The harness attribution injection** (a `remote_session_change` instruction added to every
   trial's context, per Joe's 2026-09-11 ruling) overrides every arm's own trailer convention; the
   trailer is reported above for transparency but excluded from END-STATE/FOLLOWED-all scoring.
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
9. **Total spend arithmetic:**

   ```
   opus_A valid trials (20 of 20):           $11.29
   opus_B valid trials (19 of 20, per invalidation): $13.14
   ---------------------------------------------------------
   full opus run (Task 6) subtotal:                   $24.43

   smoke run 1 (8 trials, both tasks):                 $4.56
   smoke run 2 (4 trials, Task A only, post-fix):      $2.46
   plumbing/isolation probes (Task 3 validation):      $1.20 (approx.)
   conversions (Task 4: A-R/A-F/B-F sonnet + B-S fable RED/GREEN/PRESSURE): $5.92
   ---------------------------------------------------------
   TOTAL PHASE-2 SPEND:                              ≈ $38.57
   ```

   ($24.43 + $4.56 + $2.46 + $1.20 + $5.92 = $38.57 — under the plan's $60 re-confirmation gate;
   matches the controller's own mid-run projection of "≈$39" in `progress.md`.)

## Retrieval at real-vault scale

FOUND was 20/20 across every R/F trial in both tasks that reached a retrieval step (Task A: R 5/5,
F 5/5; Task B: R 5/5, F 5/5) — the converted carrier surfaced as the top (or a top) `engram query`
hit in every trial, at 883-note real-vault scale, once the marker-only `first_procedure_step_index`
regex bug (Caveat 3a) was fixed. Rdirect is `n/a` for FOUND by design (no retrieval attempted).

## Reproduction

```bash
cd dev/eval/cumulative/runbook_vs_skill/phase2
python3 probe_phase2.py --summarize results/opus_A.jsonl
python3 probe_phase2.py --summarize results/opus_B.invalidated.jsonl
```
