# Doc-surface enumeration: learn-at-subagent-return

**The invariant that changes** is where a subagent's `LESSONS:` line is judged.

- **Before:** the lines are collected without judgment ("not judged in-flight") and judged only by the closing `/learn`.
- **After:** each line is judged by the orchestrator at the return. A lesson that clears learn's bar is captured with a fast-path `/learn` before the next dispatch. The closing `/learn` sweeps and captures only lines not yet captured.

**Search** (2026-09-29, worktree `runbook-vs-skill` on branch `learn-at-subagent-return` @ `67911117`):

```
grep -rn -i -E "LESSONS:|LESSONS line|collected LESSONS|closing (\`/)?learn|mid-cycle|fast path|learn-firing|learn\.md" --include='*.md' .
```

The hits were then filtered by hand:

- **Excluded:**
  - `openspec/changes/archive/`, which is history;
  - `dev/eval/cumulative/`, frozen eval fixtures and encodings;
  - `dev/eval/audit/results*/`, raw eval output;
  - `docs/research/`, dated research notes.
- **Also read in full:** `openspec/changes/learn-rate-skill-only/`, the sibling change (vault note 757).
- **Checked by hand:** `docs/FEATURES.md`, `docs/GLOSSARY.md`, `docs/architecture/c1-system-context.md`, `c2-containers.md` and `adr.md`, for "closing learn", "LESSONS" and "guidance".

Every row below was read at its cited line before its disposition was written.

**Disposition key:**

- *rewrite*: replace the text;
- *append*: add text without editing what is there;
- *no change*: the reason is in the row;
- *close-out*: done in the close-out task group.

**Verification rule for performing rows (vault note 1072).** After each row's edit:

- grep for the new text; it must be present;
- grep for the old text, quoted in the row; it must be absent (for *append* rows, the old text must still be present);
- only then tick the row.

| # | Location (verified) | Current text (abridged) | Disposition | Replacement / reason | Old-absent grep |
|---|---|---|---|---|---|
| 1 | agent-instructions/guidance/learn.md:15–16 | "crystallize the one confirmed correction now (the `/learn` skill's mid-cycle fast path …" | rewrite | "crystallize the one confirmed lesson now …" (design D2 item 1) | `the one confirmed correction now` |
| 2 | agent-instructions/guidance/learn.md:21–29 | "Fire at these cues:" with three bullets (review/user rejection; failed check; self-caught reversal) | append | a fourth bullet, the subagent-return cue (design D2 item 2); the three existing bullets stay byte-identical | n/a (append); new-present grep: `isn't \`none\`` |
| 3 | agent-instructions/skills/learn/SKILL.md:21–22 | "When you fire at a CORRECTION moment mid-task — a review, a failing check, or the user just rejected your approach …" | rewrite | the trigger also names "a subagent return whose `LESSONS:` line carries a lesson that clears the bar"; the rest of the block is unchanged (design D3) | none: an extension; new-present grep: `subagent return` in lines 21–27 |
| 4 | agent-instructions/skills/learn/SKILL.md:89–95 | "**Collected LESSONS lines are an explicit scan input.** When the orchestrator (following the `please` runbook) hands you the session's collected `LESSONS:` lines …" | rewrite | per-return fast-path judgment, plus a closing backstop that skips captured lines (design D3; `learn` spec delta) | `When the orchestrator (following the \`please\` runbook) hands` |
| 5 | agent-instructions/skills/learn/SKILL.md:158–162 | kind-4 "Completion-moment anchor … the moment you read a dispatched subagent's completion report (and its `LESSONS:` line)" | no change | Already names the return as a kind-4 scan moment; consistent with the new cue. | — |
| 6 | agent-instructions/skills/learn/SKILL.md:1–9 (description) | "… the moment a review, failing check, or user correction confirms a lesson …" | no change | The ambient guidance does the firing, and the description already covers "confirms a lesson". A description edit changes triggering everywhere and is out of scope. | — |
| 7 | agent-instructions/skills/learn/SKILL.md:29, 46–47 | "the closing learn always sweeps" | no change | The backstop keeps the sweep (design D3). | — |
| 8 | agent-instructions/skills/please/SKILL.md:95–99 | "record that line verbatim … This list is separate from the lessons audit (step 7) and is not judged in-flight; it is handed to the closing `/learn` as-is." | rewrite | Keep verbatim recording; judge at return per the learn guidance; mark captured entries; the full marked list goes to the closing learn (design D4). | `is not judged in-flight` |
| 9 | agent-instructions/skills/please/SKILL.md:119–122 | "handing it the full list of `LESSONS:` lines … `/learn`'s Step 2 curates that list against its own quality bar; step 7 does not pre-filter it." | rewrite | hand over the list with captured marks; the closing learn skips the marked lines and curates the rest; step 7 still drops nothing (design D4) | `curates that list against` |
| 10 | agent-instructions/skills/please/SKILL.md:155 (red-flag row) | "You reached step 7 without a running list of step 4's `LESSONS:` lines" | no change | The list is still kept. | — |
| 11 | agent-instructions/skills/route/SKILL.md:72–73 | "these are offers for the closing `/learn` to judge, not vault notes the subagent decides on itself" | rewrite | "offers the orchestrator judges when the report returns (fast-path `/learn` for those that clear the bar), with the closing `/learn` as backstop — not vault notes the subagent decides on itself" (design D4) | `offers for the closing \`/learn\` to judge` |
| 12 | agent-instructions/skills/route/SKILL.md:70–71, 74–75 | the mandatory line, its format, and the one-re-ask rule | no change | The contract is unchanged. | — |
| 13 | docs/ROADMAP.md:135 (NOW table, rank 4) | "**running 2026-09-28 (in progress); tasks 3.2–3.4 have started, no result yet**" | close-out | Rewrite: "3.2–3.3 done (W2 not distinguishable; parent capture C1 ≤ 82%, closing learn 0/2); 3.4 decided 2026-09-29: next lever `learn-at-subagent-return`, D-F layers stay parked". | `no result yet` |
| 14 | docs/ROADMAP.md:141–143 | "Parked behind rank 4's result: mechanical LESSONS validator, watcher/metacognition layer, hooks" | close-out | Rewrite "Parked behind rank 4's result" to "Parked (3.4 decision, 2026-09-29; revisit per D-F)". | `Parked behind rank 4's result` |
| 15 | dev/eval/LEDGER.md (after row `learn-rate-parent-capture`) | — | append | New row `learn-at-return-guidance`: the D5 RED/GREEN result, with P and N counts per arm, the effect label, discards and question-stops, cost, and the path to the results. | n/a (append) |
| 16 | dev/eval/LEDGER.md:103–104 | rows `learn-rate-remeasure-w2`, `learn-rate-parent-capture` | no change | Measured results; the new row cites them. | — |
| 17 | docs/GLOSSARY.md:103–113 (lessons audit) | "positive reinforcement … is captured by the closing `/learn`'s Step-2 scan, not here" | no change | Still true: the closing scan still captures kind 4; the return cue adds an earlier moment. The entry is about the audit, not LESSONS lines. | — |
| 18 | docs/GLOSSARY.md:838–846, 920–930 | guidance file list: "`learn.md` (learn-firing)" | no change | The file's role is unchanged. | — |
| 19 | CLAUDE.md:27; README.md:36, 155 | "learn-firing (`learn.md`)" | no change | The file's role is unchanged. | — |
| 20 | docs/architecture/c2-containers.md:215 | learn flow: "closing learn always sweeps" | no change | Still true. | — |
| 21 | docs/architecture/c1-system-context.md:418, 444 | learn flow; please "7 · closing /learn — capture lessons" | no change | The diagrams show learn's internal steps and please's step order; neither changes. | — |
| 22 | openspec/specs/production-guidance-activation/spec.md | which guidance files are imported | no change | Import set unchanged. | — |
| 23 | openspec/specs/write-memory-worker/spec.md:122–140 | please step 7 lessons audit | no change | The audit is unchanged. | — |
| 24 | openspec/changes/learn-rate-skill-only/design.md (D-A, D-B) | "collects lines without judging them in-flight" | no change | That change's design record. This change's design cites and supersedes it, and it becomes history at archive. | — |
| 25 | openspec/changes/learn-rate-skill-only/tasks.md:3.4 | `- [ ] 3.4 Decide escalation per design D-F …` | close-out | Tick, with a pointer to `dev/eval/audit/escalation-decision-2026-09-29.md` and this change (design D8). | `- [ ] 3.4` |
| 26 | dev/eval/guards/candidate/please.md | a frozen candidate copy of an older please text (no step-4 LESSONS paragraph) | no change | An eval fixture, not a live surface. | — |
