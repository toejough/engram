# Task 5.1 enumeration: retiring `agent-instructions/skills/write-memory/SKILL.md`

Status: 2026-09-22, built during promotion (task 4.x). Per the task's own instructions this list is
enumeration ONLY — no updates are performed here. A fresh-context reviewer independently verifies
this list and runs its own discovery pass (per the please/curate pattern) before 5.2 performs any
row's action. **5.1 itself stays unticked until that review completes.**

## Grep run

`grep -rIln "write-memory" .` over the whole repo, excluding `.git` and `dev/eval` (per task
instructions, historical results live there), then a targeted follow-up scan of
`dev/eval/cumulative/runbook_vs_skill/phase2/{probe_phase2.py,test_probe_phase2.py}` and
`dev/eval/audit/*.py` for any LIVE (non-eval-history) script hardcoding the retired skill path —
mirroring the please/curate retirements' own `please_step*_probe`/`skill_src` hardcoded-default
check.

## Exclusions (stated)

- `dev/eval/cumulative/runbook_vs_skill/phase2/results/**`: dated measurement records for this and
  prior changes' eval runs.
- `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskWriteMemory/**` and
  `.../fixtures/write-memory/**`: this change's own fixture/report artifacts (the frozen `recall`/
  `learn` skill copies, the fixture runbook note, the fidelity report, the task harness files) — not
  live references; the fixture is deliberately frozen at conversion time and is not resynced from
  the live skill sources going forward.
- `openspec/changes/recall-learn-writememory-to-runbook/{proposal,design,tasks,enumeration}.md` and
  its own delta specs (`specs/write-memory-worker/spec.md`, `specs/learn-adhoc-qa-capture/spec.md`,
  `specs/guidance-runbook-follow-frame/spec.md`, `specs/learn-runbook-capture/spec.md`,
  `specs/learn-branching-disposition/spec.md`): this change's own self-artifacts. (The last two were
  added post-review, N/A — this change's own artifact.)
- `openspec/changes/archive/**` (`2026-09-21-{please-skill-to-runbook,runbook-lexical-triggers,
  curate-skill-to-runbook}`, `2026-09-19-{route-skill-to-runbook,activate-shim-guidance}`,
  `2026-08-24-runbook-note-kind`, `2026-09-17-runbook-shim-follow-frame`,
  `2026-08-09-restore-luhmann-branching-disposition`, `2026-09-08-memory-loop-audit`,
  `2026-08-08-harden-route-skill-mechanics`): dated, already-archived change records — same
  treatment curate's own enumeration gave the please/route archive dirs.
- `docs/research/2026-08-30-memory-taxonomy-engram-map.md`: dated research record (not restated as
  current), matches the pattern applied to `docs/research/**` in prior conversions.

## Per-file disposition

Legend: keep / update / rewrite / N/A / delete / excluded.

### Root and agent-facing docs

| file:line | disposition | reason |
|---|---|---|
| `README.md:10` | rewrite | top description: "Two skills — `recall` and `learn`... hand off to `write-memory`, a worker skill that composes..." must drop "worker skill" language and fold write-memory into the "further workflows are vault runbooks" sentence alongside please/route/curate, mirroring how CLAUDE.md's own top paragraph already treats please/route/curate |
| `README.md:48-49` (skills table) | rewrite | `write-memory` row removed from the skills table (recall/learn become the only 2 skill rows); its description folded into a runbook-list sentence like the please/route/curate precedent |
| `README.md:51` | update | drop the `agent-instructions/skills/write-memory/SKILL.md` link from "See ... for the full skill definitions" |
| `README.md:53` | update | "`recall`, `learn`, `write-memory`, and `curate` shell out to the `engram` binary" → drop write-memory from that skill-listing clause; add write-memory to the "have no SKILL.md" runbook list alongside please/route/curate |
| `README.md:149` (tree comment) | update | "Source for the recall, learn, and write-memory skills" → "Source for the recall and learn skills" |
| `CLAUDE.md:3` | rewrite | top paragraph: "at their write sites they hand off to a third skill, `write-memory`, which composes and executes..." → fold write-memory into the "further workflows are no longer skills but vault runbooks" sentence, alongside please/route/curate, matching the exact pattern the curate conversion used for its own fold-in |
| `CLAUDE.md:26` (dir tree comment) | update | "Source for the recall, learn, and write-memory skills" → "Source for the recall and learn skills" |
| `CLAUDE.md:38` (Key Files) | update | `` `agent-instructions/skills/{learn,recall,write-memory}/SKILL.md` — Skill definitions (`please`, `route`, and `curate` are vault runbooks, not skills)`` → drop `write-memory` from the glob, add it to the parenthetical's runbook list |

### agent-instructions

| file:line | disposition | reason |
|---|---|---|
| `agent-instructions/skills/write-memory/` (whole dir, 1 file: `SKILL.md`) | delete (task 5.3) | the retirement itself, performed after this enumeration and its independent review |
| `agent-instructions/skills/recall/SKILL.md` (all `write-memory` mentions, incl. lines 192, 284, 293, 307, 339) | keep | already repointed at the real runbook basename in task 4.2; remaining prose mentions of the bare word "write-memory" (293, 339) describe the handoff generically and stay accurate post-retirement (the runbook is still named "write-memory") |
| `agent-instructions/skills/learn/SKILL.md` (all `write-memory` mentions, incl. lines 25, 98, 119, 122, 127, 130, 138, 191, 203, 231, 238-239) | keep | same as recall — call sites repointed in task 4.2; prose mentions (25, 98, 122, 130, 238-239) describe the handoff/note generically, still accurate |
| `agent-instructions/skills/learn/tests/baseline-confirmed-approach.md:4` | keep | "the current skill hands each a write-memory handoff" — describes the handoff generically, not "invoke the write-memory skill"; stays accurate whether write-memory is a skill or a runbook |
| `agent-instructions/guidance/shim.md:3` | update | comment block asserts recall.md/delegate.md/learn.md "stay as-is for sessions that still install the recall/learn/write-memory skills" — after retirement, write-memory is no longer an installable skill; drop it from that list (recall/learn only) |
| `agent-instructions/guidance/{recall,delegate,learn}.md` | N/A | no `write-memory` hits (confirmed by the grep run) |

### docs

| file:line | disposition | reason |
|---|---|---|
| `docs/GLOSSARY.md:33` | update | `skill` entry lists `` [`write-memory`](#write-memory-worker-skill) `` as one of the shipped skills — drop from the skill list, add to whichever "no longer a skill, now a runbook" sentence the glossary uses for please/route/curate (glossary precedent: check how those three were folded in during their own retirements) |
| `docs/GLOSSARY.md:56` | keep | "(read-memory, write-memory, route-a-task, orchestrate-a-workflow). Only `write-memory`..." — names the ADR-0015 atomic-skills concept `write-memory`, not the carrier type; verify at 5.2 whether the surrounding sentence needs a carrier-type update too |
| `docs/GLOSSARY.md:62-104` (`### write-memory (worker skill)` heading + body) | rewrite | heading says "(worker skill)" and body says "The skill at `agent-instructions/skills/write-memory/SKILL.md`" — needs the same carrier-type reword the please/route/curate glossary entries got (heading → "(worker runbook)" or similar, body → path swapped for the vault basename, "Executes..." kept) |
| `docs/GLOSSARY.md:78,95,104,334,340,496,626,894` | keep | describe the write-memory handoff/mechanism generically ("via a write-memory handoff", "hands off to write-memory") — accurate regardless of carrier type, no reword needed |
| `docs/ROADMAP.md:87` | update (task 6, close-out) | this change's own tracking row (`#760`) — moves from NOW to Shipped once merged, per the please/curate/#758 precedent; not this task's job now |
| `docs/ROADMAP.md:233` | keep | Shipped-table row describing the original write-memory-worker authoring (2026-07-04); historical record of a past decision, not a carrier-type claim needing correction |
| `docs/architecture/adr.md:353,356,366,368` (ADR-0015 body) | keep | dated 2026-07-04 decision record naming the `write-memory` atom/worker pattern; the decision itself (extract one worker atom) is unaffected by the carrier-type change |
| `docs/architecture/adr.md:370-374` (ADR-0015 "Historical note (2026-09)") | update | already documents please/route/curate's retirement from skill to runbook; needs a fourth clause adding write-memory's own retirement (`recall-learn-writememory-to-runbook`) once this change lands |
| `docs/architecture/c1-system-context.md:47` (S3 row) | update | "Engram skills (recall, learn, write-memory) are loaded by the harness's skill mechanism; the please, route, and curate procedures are vault runbooks..." → move write-memory into the vault-runbooks clause, leaving recall/learn as the only loaded-skill examples |
| `docs/architecture/c1-system-context.md:103` | update | "hand off to the **write-memory skill**" → reword to name the runbook (not "skill"), consistent with how the covered/near rows already say `engram amend` directly with no skill-name framing |
| `docs/architecture/c1-system-context.md:106` | keep | "since 2026-07-04 executed via the write-memory handoff" — describes the handoff generically, accurate post-retirement |
| `docs/architecture/c2-containers.md:18` (mermaid skills box) | update | `` skills["C1 · Skills (learn / recall / write-memory)<br/>markdown behavior specs"] `` → drop write-memory from the box (leaves learn/recall); C4 box or a new runbook-list annotation should gain it, mirroring how please/route/curate moved from the C1 mermaid box to C4 during their own conversions |
| `docs/architecture/c2-containers.md:25` (mermaid edge label) | update | `` agent -->\|"runs /learn, /recall, /write-memory"\| skills `` → drop `/write-memory` from the skills-edge label |
| `docs/architecture/c2-containers.md:41` (C1 row prose) | rewrite | "The LLM-judgment layer: ... and `/write-memory`, the vault-write worker..." sentence needs write-memory moved out of the C1 skills description into the "no longer skills... `type: runbook` vault notes" sentence alongside please/route/curate, same restructuring curate's own conversion applied to this exact row |
| `docs/architecture/c2-containers.md:136-137,192,194-195,207,221,225` (sequence diagrams) | keep or update — reviewer to confirm | multiple "hand off to write-memory skill"/"write-memory skill: composes + executes" labels inside mermaid sequence diagrams; these describe the mechanism (a labeled diagram node/note), not asserting SKILL-tool carrier status per se — flag for the independent reviewer to judge whether the diagram node label itself needs "skill" → "runbook" wording or can stay as a named step |
| `docs/architecture/c3-components.md:235,250` | keep | "hands off to write-memory with kind=qa" / "args from the skill (write-memory kind=qa handoff...)" — line 250 says "the skill" generically without naming write-memory as the carrier type explicitly enough to require a reword; reviewer to confirm line 250's "the skill" doesn't need to become "the runbook" |
| `docs/research/2026-08-30-memory-taxonomy-engram-map.md` (all hits) | excluded | dated research record, not restated as current (matches the `docs/research/**` exclusion pattern from prior conversions) |

### openspec (main specs, not this change's own delta)

| file:line | disposition | reason |
|---|---|---|
| `openspec/specs/write-memory-worker/spec.md` (whole file) | N/A this task; requirements synced at archive (task 6.3); Purpose line 5 fixed at archive time (task 5.5, see "Task added for archive time" below) | this change's own delta spec (`openspec/changes/recall-learn-writememory-to-runbook/specs/write-memory-worker/spec.md`) carries the corrected carrier-type requirements and syncs into this main spec when the change archives — normal OpenSpec flow, not a 5.2 row; Purpose itself cannot be edited by a delta, so its "write-memory skill" wording is fixed by a dedicated archive-time task |
| `openspec/specs/learn-runbook-capture/spec.md:61-68` | update via delta (task added post-review) | "The `write-memory` skill SHALL compose..." — carrier-type wording (`skill`); a MODIFIED-requirement delta now exists at `openspec/changes/recall-learn-writememory-to-runbook/specs/learn-runbook-capture/spec.md` (added per independent review), syncs at archive (task 6.3) |
| `openspec/specs/route-dispatch-evidence/spec.md:8,10,12,25` | keep | describes route's own hand-off TO write-memory generically ("through write-memory", "write-memory has no amend form") — no carrier-type claim ("skill"/"runbook") appears in these lines; no reword needed |
| `openspec/specs/production-guidance-activation/spec.md:20-22` | keep | scenario title/body explicitly anticipates this exact promotion ("recall/learn/write-memory are promoted... to production runbooks") — already correct, describes the pre-promotion state as a scenario precondition, not a live claim |
| `openspec/specs/runbook-lexical-triggers/spec.md:99,103,107,111,115` | keep | uses "write-memory" as the acting subject that appends `--trigger` flags during another runbook's own conversion — describes write-memory's own compose behavior generically (accurate under either carrier type); no "skill" claim present |
| `openspec/specs/vault-offer-curation/spec.md:45` | keep | "no handoff to `write-memory`" — generic mechanism reference, no carrier-type claim |
| `openspec/specs/learn-branching-disposition/spec.md:9,12,45,51,55` | line 45 update via delta (task added post-review); 9,12 keep | "Learn skill decides note placement before write-memory handoff" (9,12 — generic mechanism reference, no carrier-type claim, kept); "The write-memory skill's handoff contract SHALL accept..." (line 45) explicitly says "skill" — a MODIFIED-requirement delta now exists at `openspec/changes/recall-learn-writememory-to-runbook/specs/learn-branching-disposition/spec.md` (added per independent review), syncs at archive (task 6.3) |
| `openspec/changes/learn-rate-skill-only/{design,proposal}.md`, `specs/learn/spec.md` | keep | separate, active, unrelated change (learn-closing-sweep curation, not write-memory's carrier). Mentions are generic mechanism references ("crystallize via write-memory", "the closing learn passes it to write-memory") except `design.md:7`'s "write-memory subagent", a pre-existing minor misnomer (write-memory has never been a subagent) predating this change and out of its scope — flag only, not fixed here |

### code and dev (live, non-eval-history)

| file:line | disposition | reason |
|---|---|---|
| `dev/eval/cumulative/runbook_vs_skill/phase2/probe_phase2.py` (comment/help-text mentions at lines ~50, 503, 1581, 2031, 2484) | keep | all are comments or `--help` text describing the harness's own fixture-copying behavior ("copy the 8 recall/learn/write-memory runbook fixture notes"); none hardcodes `agent-instructions/skills/write-memory/SKILL.md` as a live default the way please/curate's own step-probe scripts once hardcoded a live skill path — no `write-memory_step*_probe`-style script exists for write-memory (write-memory has no independent trigger, so no such probe was ever built, per design D4) |
| `dev/eval/cumulative/runbook_vs_skill/phase2/test_probe_phase2.py` | keep | no hardcoded live `write-memory/SKILL.md` default found in the targeted grep |
| `dev/eval/audit/extract.py:578-579,587,591` | keep — reviewer to confirm | transcript-parsing logic hardcodes the literal string `"write-memory"` as a Skill-tool name to detect nested `Skill(write-memory)` calls when auditing PAST session transcripts (recorded before this retirement). This is correct for parsing historical transcripts (which really did invoke `Skill{write-memory}`) and doesn't need to change for that purpose; however, transcripts recorded AFTER this retirement will invoke write-memory via `engram show <basename>` in Bash, not `Skill(write-memory)` — `extract.py`'s learn/write-memory-call detection may need a second detection path for the post-retirement shape to keep auditing future sessions correctly. Flagging as a genuine open question for the independent reviewer, not resolving here (audit tooling is outside this change's stated scope, but the risk is real and analogous to what design.md's Risk section calls out for missed call sites) |
| `dev/eval/audit/build_scorecard.py:391` | keep | "Did a learn/write-memory step actually execute?" — a scorecard question label describing behavior generically; accurate under either carrier type |
| `dev/eval/audit/test_extract.py` (multiple) | keep | tests for the `extract.py` behavior above; would need updating only if `extract.py` itself gains a post-retirement detection path (same open question, not resolved here) |

## Counts by disposition

- update: 12
- rewrite: 4
- keep: ~25 (includes reviewer-flagged rows and false-positive-shaped rows)
- N/A (this task; deferred to archive/close-out): 3
- excluded (directory/file-level): 6 groups
- delete: 1 (the skill dir itself, task 5.3)

## Task added for archive time

- write-memory-worker main spec Purpose (line 5) says "The write-memory **skill** is a dedicated
  worker..."; a delta cannot edit Purpose or title. Reword at archive to name the write-memory
  runbook (basename `1053.2026-09-22.write-memory-compose-execute-verify`) instead of "skill".
  Tracked as task 5.5 in tasks.md (mirrors the `please-skill-to-runbook` conversion's task 5.7
  precedent for this exact same spec's Purpose line).

## Open items for the independent reviewer

1. Confirm the `docs/GLOSSARY.md` skill-list fold-in wording (line 33 and the `### write-memory`
   heading) matches whatever pattern please/route/curate's own glossary entries actually ended up
   with — this enumeration describes the *shape* of the needed edit, not exact final prose.
2. Judge whether `docs/architecture/c2-containers.md`'s sequence-diagram labels (lines 136-137, 192,
   194-195, 207, 221, 225) and `c3-components.md:250`'s "the skill" need a literal carrier-word
   change, or whether "write-memory" as a named step/node in a diagram is carrier-agnostic and can
   stay as-is.
3. RESOLVED: `openspec/specs/learn-runbook-capture/spec.md` and
   `openspec/specs/learn-branching-disposition/spec.md` each needed a delta spec, since the
   asserted "write-memory **skill**" text is a requirement body (not a Purpose line) and this
   change is a MODIFIED-capability edit, not archive-time-only. Delta specs added at
   `openspec/changes/recall-learn-writememory-to-runbook/specs/{learn-runbook-capture,
   learn-branching-disposition}/spec.md`.
4. Decide whether `dev/eval/audit/extract.py`'s transcript-parsing needs a post-retirement
   `Skill(write-memory)`-absence detection path (i.e., recognizing the new `engram show <basename>`
   shape in future transcripts) — flagged as a real risk, not resolved here, and arguably outside
   this change's stated scope (design.md's Non-Goals don't mention audit tooling either way).
