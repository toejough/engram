# Proposal: learn-at-subagent-return

## Why

`learn-rate-skill-only` v1 routes every subagent `LESSONS:` line to one place: the closing `/learn`, which judges the collected list at the end of the cycle. The two measurements taken after v1 shipped show that parents do not capture that way:

- **The closing learn does not run.** In the parent-side capture check (`dev/eval/audit/REPORT-parent-capture-2026-09-29.md`), neither of the 2 parent sessions that received non-`none` LESSONS blocks ran a `learn` after its last LESSONS receipt (C0 = 0/2). The capture that happened came from direct `engram learn` writes made in the middle of each session (38 of the 53 captured items were written by the parent's own transcript), plus subagents writing notes themselves.
- **Mid-session capture is real but loose.** The judge scored 93/114 = 81.6% [74.6, 87.7] of worth-capturing items as captured or already covered. That is an upper bound: the judge is lenient (an unblinded spot-check held for only 4/10 captured and 6/8 covered matches), and removing matches to route-dispatch/route-evidence records alone drops it to 80/114 = 70.2% [62.3, 78.1]. The population is 2 clustered sessions.
- **The W2 re-measure could not see this path.** `dev/eval/audit/REPORT-re-measure-2026-09-28.md` found W2 = 3/66 = 4.5% [0.0, 10.6] against the 5.6% [1.4, 11.3] baseline, not distinguishable (−1.1 pp [−8.5, +6.3]). Its caveat 3a says the instrument counts learn only inside audited subagent windows, so a lesson the parent captured still scores as "learn didn't fire".

Joe, 2026-09-29: "update the guidance to lean into the agents' tendency to [learn] midstream … after sub agents return, in particular." Agents already capture midstream; the v1 text tells them not to ("collects lines without judging them in-flight"; please step 4: "not judged in-flight"). This change moves the judgment to the moment the lesson is in hand, the return, and keeps the closing learn as the backstop.

## What Changes

1. **Learn guidance gains a return cue** (`agent-instructions/guidance/learn.md`). When a subagent returns and its report's `LESSONS:` line is not `none`, the orchestrator judges each lesson then. A lesson that clears learn's Step-2 bar (one of the four kinds; confirmed, not hypothesized; a general reusable principle) is captured with `/learn` on the mid-cycle fast path (one note per lesson, no sweep) before the next dispatch. `none`, trivial, or unconfirmed lessons are left alone. The rule lives in ambient guidance, not in route or please templates (vault note 855: a rule that must hold at every delegation, whichever skill mediated it, belongs in always-loaded guidance).
2. **Learn skill applies the LESSONS rule per return** (`agent-instructions/skills/learn/SKILL.md`, the "Collected LESSONS lines are an explicit scan input" paragraph and the mid-cycle fast-path block). The rule now applies to each returned result on the fast path. The closing learn keeps its sweep and picks up any LESSONS lines not yet captured, skipping lines already captured at return.
3. **Consistency edits, scoped to wording that contradicts the new cue:**
   - `agent-instructions/skills/please/SKILL.md` step 4 ("is not judged in-flight; it is handed to the closing `/learn` as-is") and step 7's hand-off sentence: the running list is still kept verbatim, each line is judged at return per the learn guidance, captured lines are marked, and the full marked list still goes to the closing learn.
   - `agent-instructions/skills/route/SKILL.md` completion-report bullet ("offers for the closing `/learn` to judge"): the offers are judged by the orchestrator when the report returns, with the closing `/learn` as backstop. The contract itself (mandatory line, format, one re-ask) does not change.
   - Nothing else in route, please, or the `lessons-contract` capability changes.
4. **Resolves `learn-rate-skill-only` task 3.4.** 3.4 asks for an escalation decision on the parked layers (watcher, mechanical validator, hooks). This change is that decision: none of the three escalates now; the next lever is a further md-only edit (this change), and the parked layers stay parked with their D-F revisit condition. Close-out writes `dev/eval/audit/escalation-decision-2026-09-29.md` pointing here, ticks 3.4, and archives `learn-rate-skill-only` before this change archives (see Impact).

Both markdown edits are tested before they ship: the guidance edit by confined headless RED/GREEN arms with a pre-registered pass bar and false-fire bar; each SKILL.md edit by writing-skills TDD (design D5–D7).

## Capabilities

### New Capabilities

- `guidance-learn-moments`: the ambient learn guidance fires `/learn` on the fast path at a subagent return whose `LESSONS:` line carries a lesson that clears learn's bar, before the next dispatch, and does not fire for `none`, trivial, or unconfirmed lessons.

### Modified Capabilities

- `learn`: "Closing learn curates collected LESSONS lines" becomes per-return fast-path judgment plus a closing backstop that sweeps and captures only lines not yet captured.
- `please`: two requirements change.
  - "Orchestrator collects LESSONS lines from all dispatched work" keeps verbatim collection but drops "collection is not judging". Each line is judged at return per the learn guidance, and captured lines are marked in the list.
  - "Please passes collected LESSONS lines to closing learn" hands over the list with those marks, and the closing learn judges only the unmarked lines.

`learn` and `please` exist today only as delta specs of the unarchived `learn-rate-skill-only` change; see Impact for the archive order. `route` and `lessons-contract` requirements are unchanged (route's edit is wording in the skill body only).

## Impact

- **Files:** `agent-instructions/guidance/learn.md`; `agent-instructions/skills/{learn,please,route}/SKILL.md`; a new confined-arm harness under `dev/eval/learn-at-return/`; `dev/eval/LEDGER.md` (new row); `docs/ROADMAP.md` (NOW-table row 4); `dev/eval/audit/escalation-decision-2026-09-29.md` (new); `openspec/changes/learn-rate-skill-only/tasks.md` (tick 3.4). Full doc-surface list: `enumeration.md`.
- **No Go code, no CLI change, no storage change.**
- **Archive order:** `learn-rate-skill-only` must archive first so that `openspec/specs/learn` and `openspec/specs/please` exist for this change's MODIFIED deltas to apply to. `openspec validate --strict` does not check that today; `openspec archive` does.
- **Deploy:** `engram update --with-guidance` for learn.md; `engram update` for the three skills.
- **Spend:** the guidance eval runs about 45 headless arms (design D6); the estimate is confirmed with Joe before the first batch.
- **Risk:** prose-only firing stays below ~95% per trial (vault note 198). This change raises the rate and moves the capture moment earlier; it guarantees nothing. The parked mechanical layers remain the escalation path.
