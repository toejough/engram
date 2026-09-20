# please task: why this ask (OpenSpec change `please-skill-to-runbook`, tasks 1.1/1.2)

**The ask** (`task-prompt.txt`): "Please take this end-to-end: rename the tally CLI's `--out` flag to
`--output`. It is a hard rename with no alias for the old spelling. Get it landed properly in this repo."

The fixture is a ~60-line Python CLI (`tally.py`) whose `--out` flag is echoed in seven places: parser
code plus a comment, a test, the README usage block, `docs/usage.md`, a mermaid edge label and prose in
`docs/architecture.md`, a bash completion script, and a historical CHANGELOG entry that must NOT change.

## Why it is representative

| please requirement (SKILL.md) | How the task forces it |
| --- | --- |
| Seven-step spine, natural-language trigger | "Please take this end-to-end" is one of the skill's own trigger phrasings; the ask spans code, tests, docs, and a commit, so the multi-step test fires. |
| Step 3 doc-surface enumeration grep (non-waivable, altering a repeated invariant) | A flag name is exactly a "repeated invariant echoed across docs, diagrams, and skills". A from-memory doc scrub misses the diagram label and the completion script (note 186). The plan must carry a per-file disposition list; the CHANGELOG is a genuine `keep`. |
| Gate A (four angles), Gate C (docs touched), Gate D (commit message) | Plan, docs, and a commit message all exist. Gate B is NOT scored: a rename may have an empty REFACTOR phase, so requiring it would make `followed_all` unreachable for the skill row (the route task's D8 lesson). |
| Step 4 TDD with RED first | The test file is changed first and must fail before the code changes; `steps.json` orders test edit, code edit, pytest. |
| Step 6 deletes planning artifacts and commits | The plan is committed at Step 3, so it must be deleted before the end; `done_when_checks.sh` reads it back from history. |
| Step 7 lessons audit + closing `/learn` | The closing report must enumerate all four audit corpora (STOPs, gate FAILs, CORRECTION commits, escalations); step 18 matches that shape in assistant prose (new `text_regex` signal). |
| Cheap enough for n=3 per arm | Tiny repo, no network, pytest in milliseconds; the cost is the ceremony itself (recall/learn twice, four to seven reviewer subagents), which is the thing under test. |

## What is measured, and what is not

- `steps.json` (19 steps) scores transcript-observable behavior: learn/recall bracket, enumeration grep,
  plan commit, each Gate A angle dispatched, recall-first in reviewer handoffs, RED/GREEN order, Gate C,
  Gate D, final commit, closing learn, the lessons-audit report, and the `LESSONS:` handoff contract.
- `done_when_checks.sh` scores repo end state: behavior of `--output`, old spelling rejected, tests pass,
  every surface echo updated, CHANGELOG history intact plus a new entry, tree clean, plan artifacts gone,
  and a plan committed before the first `tally.py` change with a disposition line per surface file.
- **Wikilink citations are not scored yet.** SKILL.md never asks for `[[wikilink]]` references in the
  audit, so scoring them would fail the skill row by construction (the route eval's lesson). The
  wikilink requirement (design D2) belongs to the converted runbook: add a step for it when conversion
  task 2.5 lands, and score it on the shim arm only.
- Not scored: the anti-sycophancy lean and escalation provenance (no flaw is planted in the ask; those
  are shim-floor drops per design D4).

## Carriers and background vault

- `task.json` carries the skill row only: `skill_src` is a frozen copy of `please/SKILL.md`
  (`encodings/taskPlease/Please-S/`), so the baseline stays comparable after task 5.3 deletes the live
  skill. `carrier_r_src` points at the converted runbook set (`encodings/taskPlease/Please-R/vault`, task 2.x); there is no
  F carrier, and the harness raises on an F arm instead of copying the CWD into the vault.
- `removal_basenames` (97 notes) scrubs the please domain from every arm's background vault
  (note 996): doc-scrub/enumeration-grep/disposition notes, gate and reviewer notes, `route-*` gate
  evidence notes, the please-lean and lessons-audit probe notes. Seven notes that only mention please in
  passing remain (they cover other topics); check one bare-arm transcript's `engram query` results before
  trusting a bare-arm number.

## Validation without paid runs

`test_probe_phase2.py` (please section) builds a hand-followed ideal end state and proves
`done_when_checks.sh` passes it (plan under `plan.md` or `docs/plans/`, any disposition casing), then
proves ten single-defect mutants each fail with a message naming the defect, and that a literal
skill-following transcript satisfies all 19 steps while a bare-agent transcript fails the ceremony.
Before spending, run `python3 probe_phase2.py --baseline please --model sonnet --n 3` (arm N) per note
989 to confirm the bare agent fails the ceremony steps; this is paid and is left to task 1.3.
