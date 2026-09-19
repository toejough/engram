## 1. RED baseline

- [ ] 1.1 Confirm the route-R-0 transcript (this session's `route-skill-to-runbook` D8 rerun) is sufficient RED evidence on its own — reread message 87 and confirm it shows the agent stopping despite already-stated clarity, with the CURRENT (unmodified) shim.md in effect; record the transcript path and message index as the RED artifact. If deemed insufficient, get explicit sign-off before running a fresh live trial (real API spend).

## 2. Edit the shim

- [ ] 2.1 Edit `agent-instructions/guidance/shim.md` rule 5 to the exact text in design.md's Decisions (D1) — the bolded lead-in extension plus the one new sentence inserted after the wording-clash sentence. Leave every other word of the rule, and every other rule/section in the file, byte-identical.
- [ ] 2.2 Diff the file to confirm only rule 5 changed (`git diff agent-instructions/guidance/shim.md` shows one hunk, in rule 5 only).

## 3. Sync the spec delta

- [ ] 3.1 Sync this change's `specs/guidance-runbook-follow-frame/spec.md` delta into `openspec/specs/guidance-runbook-follow-frame/spec.md` via `opsx:sync` (or `openspec archive` at close-out) — never a hand-edit of the live spec file.
- [ ] 3.2 Confirm the live spec's "stop and ask" requirement now includes the new sentence and the new scenario, with the requirement title and the pre-existing scenario title unchanged (byte-identical to before, per vault notes 643/651's header-matching convention).
- [ ] 3.3 `openspec validate shim-question-stop-genuine-ambiguity --strict` passes (positional arg, not `--change` — vault note 755).

## 4. GREEN verification

- [ ] 4.1 Re-read the route-R-0 transcript's message 87 against the UPDATED rule 5 text; confirm in writing (in this task's completion note) that the new sentence directly covers this exact case — the agent's own "I still owe the evidence record" statement is precisely the "restated plan already names that later step as required" condition the new sentence names.
- [ ] 4.2 Spot-check that an existing genuine-ambiguity case still stops under the updated rule: pick one of rule 5's own examples (a step naming a missing GitHub remote, or two requirements in tension) and confirm the updated wording still directs a stop there — read the updated text line by line against that example, do not assume no regression.
- [ ] 4.3 If Joe authorizes a fresh live trial to confirm behaviorally (not just textually) that the updated shim stops the route-R-0-shaped halt: run it (cost TBD, get sign-off first per this session's eval-spend convention) and record the result. If not authorized, record explicitly that GREEN verification here is textual/design-level only, not behaviorally re-measured — an honest limitation, not a gap to paper over.

## 5. Deploy

- [ ] 5.1 Run `engram update --with-guidance`; confirm `~/.claude/engram/guidance/shim.md` and `~/.pi/agent/guidance/shim.md` are byte-identical to the repo's updated `agent-instructions/guidance/shim.md`.

## 6. Close out

- [ ] 6.1 Comment on GitHub issue #762 confirming this change is applied: the wording diff, the RED/GREEN evidence from tasks 1 and 4, and whether behavioral re-measurement (4.3) happened.
- [ ] 6.2 `openspec archive shim-question-stop-genuine-ambiguity` once 6.1 is done and `openspec validate --strict` (whole-project, `--all` — vault note 916) is clean.
