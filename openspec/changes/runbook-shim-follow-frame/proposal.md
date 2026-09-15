## Why

The runbook-vs-skill eval (`dev/eval/cumulative/runbook_vs_skill/phase2`, checkpoint 2026-09-13) shows runbook notes are **found** as reliably as skills but not **followed**: on history-rewrite the skill arm restated the procedure's steps as its plan 3/3 and reached the end state 2/3; the runbook and fact arms restated 0/6 and reached it 0/6, with the same six steps in hand. The deployed shim (the #735 task-start cue) tells the agent to look for a runbook and says nothing about what to do with one; the skill path gets an implicit follow frame ("you invoked this") plus a behavioral layer (red flags, common mistakes) the runbook kind was scoped without.

The ultimate goal (Joe, 2026-09-13) is for every skill, including recall and learn, to become a runbook, with one CLAUDE.md shim as the only custom instruction text. This change builds that shim's follow half and the schema field it needs, and measures it against the skill row on the same tasks.

## What Changes

- **Shim: a new guidance file, `agent-instructions/guidance/shim.md`** (deployed via `engram update --with-guidance` alongside the existing files; the existing `recall.md` is not edited). In the runbook trial arm it is the only guidance in CLAUDE.md. It carries: (1) the literal first action, an `engram query` before the first tool call on every user request (not gated on "multi-step" or "done before", which is what the lookup itself decides), so the shim bootstraps without a recall skill; (2) one line on how each returned kind is treated — a `fact` is knowledge and context, taken as true for the task unless the repository contradicts it; a `feedback` note is a correction the user already gave, a standing instruction not to repeat the mistake it names; a `chunk` is raw evidence, fetched only when the notes leave a gap; a `runbook` is executed — and, for a `kind: runbook` whose `situation` matches the task, the frame: announce it by name, restate its steps as the plan (one todo per step where the harness has todos), adopt `done_when` as the completion bar, and stop and ask when a step is unclear or would be deviated from; (3) the general behavioral floor, applied to every runbook uniformly (the urge to shortcut is the cue to reread the step; substituting a related action is a skip; a question stop is a clarity signal, not a failure); (4) wikilinked runbooks are fetched (`engram show`) and followed the same way.
- **Schema: one optional structured field for runbook-specific red flags** (`red_flags`, name final in design) alongside `situation`/`done_when`/body, on `engram learn runbook`, in the query/show rendering, and in `write-memory`'s runbook composition. Notes carry task-specific failure modes only; they must not restate the general floor.
- **Eval: shim-only configuration.** The phase-2 harness gains an arm configuration with no engram skills in the config dir, recall and learn present in the trial vault as runbooks, the new shim in CLAUDE.md. The runbook and fact arms are rerun on history-rewrite (then bisect-before-fix, route) against the existing skill row. The frame keys on `kind: runbook`, so the fact arm stays vanilla; that is the runbook-vs-fact test the eval was built to answer.
- `guidance-recall-moments` (`recall.md`) is not modified. The shim restates the four decision-moment cues in its own file as `engram query` actions, since `/recall glance` does not exist in a shim-only session; reconciling the two files belongs to the later recall-replacement effort.
- Rejected on the record: a hook-based frame (harness-specific; engram must not rely on hooks) and a runbook-situation roster loaded into CLAUDE.md (does not scale).
- Follow-up, not in scope: learn prompting for red flags at capture time (#751).

## Capabilities

### New Capabilities
- `guidance-runbook-follow-frame`: what the shim requires of an agent once a runbook is returned: announce, restate steps as plan, done_when as bar, stop-and-ask on unclarity, general behavioral floor, wikilinked runbooks followed transitively, bootstrap first action.

### Modified Capabilities
- `learn-runbook-capture`: runbook notes gain an optional `red_flags` field (schema question four: what do people get wrong doing this); `engram learn runbook` accepts it; `write-memory` passes it through.
- `recall-runbook-surfacing`: a surfaced runbook renders its `red_flags` field; a matched runbook's full content is available via `engram show`.

## Impact

- New `agent-instructions/guidance/shim.md` (authored via a writing-skills-equivalent RED/GREEN/pressure cycle); `engram update --with-guidance` learns the new file; deployed copy at `~/.claude/engram/guidance/shim.md` is a sync artifact. `recall.md`, `delegate.md`, `learn.md` untouched.
- `internal/` runbook note parsing/rendering and `engram learn runbook` flags; `agent-instructions/skills/write-memory/SKILL.md` runbook compose block; recall payload rendering.
- `dev/eval/cumulative/runbook_vs_skill/phase2/probe_phase2.py`: shim-only config option; recall and learn skills converted to runbook notes for the trial vault; scorer's restate-as-plan signal.
- `dev/eval/LEDGER.md` row; `openspec/specs/` delta files above.
- Pass bar (pre-registered): on history-rewrite, runbook arm at n=3 matches the skill row on found (3), restated (3), every step (3), and end result within one trial (skill 2/3); any question stop is reported as an instruction-clarity finding and the shim amended before the next stage.
