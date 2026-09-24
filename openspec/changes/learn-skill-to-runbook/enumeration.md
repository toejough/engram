# Task 5.1 enumeration (DRAFT): retiring `agent-instructions/skills/learn/SKILL.md`

**STATUS: DRAFT — built ahead of the paid validation gate (2026-09-23), before section 3/4 of
tasks.md have run.** This is a static-analysis pass over the CURRENT repo state, useful because no
eval outcome changes which docs mention `learn` as a skill today. It MUST be re-verified for
freshness (re-grep everything below) immediately before task 5.1's real execution, since repo
state may drift between now and then (this change's own tasks 1.4–4.4 will touch files, and the
sibling `recall-glance-skill-to-runbook` change may land first or land concurrently — see the
"Ordering" note below). Do NOT treat this file as the gate-cleared enumeration; task 5.1 itself
still requires an independent fresh-context reviewer running its own discovery pass AFTER a fresh
re-grep, per the write-memory/curate/please precedent. **Task 5.1 is NOT ticked in tasks.md.**

## Ordering uncertainty (read before acting on this list)

At the time of this draft, neither `learn-skill-to-runbook` nor `recall-glance-skill-to-runbook`
has archived, and both are being worked in the same worktree. This list assumes **learn retires
alone, with `recall` still a skill** (the scope of this change). Two real orderings are possible
by the time this actually executes:

- **Learn retires first (recall still a skill):** the skill count in `agent-instructions/skills/`
  drops from two to **one** (`recall`). Every row below that says "drop to one: `recall`" applies
  as written.
- **Recall has already retired by the time this lands:** the skill count drops to **zero**, and
  several rows below (anything saying "engram ships one: recall" or listing `recall` as the
  remaining skill) need a further edit this draft cannot pre-write, since it depends on exactly
  how `recall-glance-skill-to-runbook`'s own doc updates land. **Re-check this explicitly at
  execution time — do not assume either ordering.**

## Grep run

Case-insensitive, whole repo: `\blearn skill\b`, `\bthe learn skill\b`, `skill=learn`,
`Skill.*skill=learn`, plus a broad scan of `recall|learn` mentions in `CLAUDE.md`, `README.md`,
`docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,c3-components,adr}.md`,
`docs/ROADMAP.md`, `agent-instructions/**`, and the `openspec/specs/{learn-*,write-memory-worker,
guidance-runbook-follow-frame}/spec.md` main specs. Targeted follow-ups: the literal path string
`agent-instructions/skills/learn` across `*.py`/`*.go`/`*.json`/`*.toml` (to find any LIVE script
hardcoding it as a default, per the "generators, not just describers" lesson — see Finding 2
below), and `dev/eval/audit/extract.py` for hardcoded `Skill(learn)` transcript-parsing logic.

## Exclusions (stated)

- `dev/eval/cumulative/runbook_vs_skill/phase2/results/**`: dated measurement records.
- `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/{taskLearn,taskRecall,taskPlease,
  taskRoute,taskCurate,shim}/**`, `.../encodings/learn-conversion-fidelity-report.md`,
  `.../fixtures/learn/**`: this change's own (and sibling changes') fixture/report artifacts —
  frozen copies, not live references, not resynced from the live skill going forward.
- `openspec/changes/learn-skill-to-runbook/{proposal,design,tasks,enumeration}.md` and its own
  delta specs (`specs/{learn-adhoc-qa-capture,learn-branching-disposition,write-memory-worker}/
  spec.md`): this change's own self-artifacts.
- `openspec/changes/recall-glance-skill-to-runbook/**`: the sibling change's own self-artifacts —
  not this task's job (its own 5.1 enumeration is `openspec/changes/recall-glance-skill-to-runbook/
  enumeration.md`, drafted separately).
- `openspec/changes/learn-rate-skill-only/**`: a separate, active, unrelated change (learn-rate
  curation quality). Its "learn skill" mentions concern kind-4 exemplar calibration, not carrier
  status — false positives, not touched here (see below).
- `openspec/changes/archive/**`: dated, already-archived change records.
- `docs/research/**`: dated research records, not restated as current.
- `internal/update/{sync,migration,downgrade,symlink,runner,update}_test.go`,
  `internal/cli/{update,update_deps,invariants_u1}_test.go`: see Finding 1 below — these use
  `"learn"` as an ARBITRARY example skill name in an in-memory fake filesystem to test the
  generic sync/symlink/migration engine. They assert nothing about whether `learn` is currently a
  real shipped skill and do not read the real `agent-instructions/skills/learn/` directory; they
  keep passing unchanged whether or not the real skill exists. Not live references — keep, no
  action.

## Findings worth flagging before executing

**Finding 1 (generators, not just describers — per the invariant-grep lesson from a prior
conversion's missed surfaces): `dev/eval/cumulative/runbook_vs_skill/probe.py::build_cfg_template`
hardcodes the live default `skills=("recall", "learn")`, copying `agent-instructions/skills/
<name>/` into eval cfg dirs.** This is called with its default (both skills) by
`dev/eval/cumulative/runbook_vs_skill/phase2/probe_phase2.py::build_cfg_template_phase2` /
`build_cfg_pool_phase2` — the function used for the "skill-arm baseline" that THIS change's own
task 1.4 must run before learn retires (and that write-memory/curate/please's own task 1.4s already
used). Nothing needs to change for those already-completed runs, but once `agent-instructions/
skills/learn/` (and eventually `recall/`) is deleted, this default silently copies nothing for the
missing skill (the `if os.path.isdir(src)` guard in `build_cfg_template` just skips — the same
footgun its own docstring already documents for `matrix.py`'s equivalent function) — any FUTURE
eval run relying on the "both engram skills installed" warm baseline default will silently produce
a cfg with fewer skills than intended, no error. Flag for the independent reviewer: decide whether
to (a) leave it (acceptable since post-retirement nothing should call the warm-skills default
again), or (b) add a defensive assertion/comment once both learn and recall are gone. Not fixed in
this draft (no code changes).

**Finding 2 (a real gap in the current proposal's scope, worth Joe's attention):
`agent-instructions/guidance/learn.md` — the deployed, `@import`-ed guidance file the proposal
explicitly defers retiring (proposal.md "Deferred" section, framed only as "redundancy with
shim.md's re-entry moments") — literally instructs, twice, in bold: "Fire the engram `/learn`
skill specifically — the `Skill` tool with `skill=learn`". This is not merely redundant with
shim.md; it is a carrier-specific mechanism instruction that will be FACTUALLY WRONG the moment
`agent-instructions/skills/learn/` is deleted and `engram update` removes the deployed skill — an
agent following this guidance verbatim would invoke `Skill(skill=learn)`, which will no longer
exist. This is asymmetric with `recall.md`, whose guidance only ever says `/recall glance` (a
slash-command phrase compatible with either the skill mechanism or a runbook's trigger phrase) —
`recall.md` needs no such fix. `learn.md`'s wording needs at least a minimal correction (e.g.,
naming the runbook or saying "fetch and follow the learn runbook") even if full retirement of
`guidance/learn.md` stays a deferred fast-follow, per the proposal's own Non-Goal. **Not fixed in
this draft** (guidance files are deployed, `@import`-ed production content, well outside "draft
docs only" scope) — flagged here for Joe's explicit decision, and should be raised again at task
5.2 (live-reference update) time regardless of who decides on `guidance/learn.md`'s own
retirement timeline.

## Per-file disposition

Legend: keep / update / rewrite / N/A / delete / excluded.

### Root and agent-facing docs

| file:line | disposition | reason |
|---|---|---|
| `CLAUDE.md:3` | rewrite | "Two skills — `recall` and `learn` — read from and write to the vault on demand" → one skill (`recall`), fold `learn` into the runbook-list sentence alongside please/route/curate/write-memory, mirroring the exact fold-in pattern each prior conversion used for its own retiring skill |
| `CLAUDE.md:26` (dir tree comment) | update | "Source for the recall and learn skills" → "Source for the recall skill" |
| `CLAUDE.md:38` (Key Files) | update | `` `agent-instructions/skills/{learn,recall}/SKILL.md` — Skill definitions (`please`, `route`, `curate`, and `write-memory` are vault runbooks, not skills) `` → drop `learn` from the glob (leaves `agent-instructions/skills/recall/SKILL.md`), add `learn` to the parenthetical's runbook list |
| `CLAUDE.md:49` (Design Principles) | update | "Skills for behavior (learn, recall), slim Go binary for computation" → "Skills for behavior (recall), slim Go binary for computation" |
| `README.md:10` | rewrite | "Two skills — `recall` and `learn` — read from and write to an agent-memory vault on demand. Four further workflows are vault runbooks..." → one skill (`recall`), five further runbooks (add `learn`); the sentence enumerating what each runbook does needs a `learn` clause added, matching the shape already used for `please`/`route`/`curate`/`write-memory` |
| `README.md:12` | keep | "`engram update` installs the skills into Claude Code and Pi... so `recall` and `learn` work the same on each" — describes behavior generically (both stay true; `learn` "works the same on each" as a runbook too); reviewer to confirm no reword needed |
| `README.md:45-49` (skills table) | rewrite | `learn` row removed from the 2-row skills table (now 1 row: `recall`) |
| `README.md:50` | update | drop the `agent-instructions/skills/learn/SKILL.md` link from "See ... for the full skill definitions" (leaves the `recall` link) |
| `README.md:52` | update | "`please`, `route`, `curate`, and `write-memory` have no `SKILL.md`..." → add `learn` to that list and its explanatory sentence |
| `README.md:53` | update | "`recall`, `learn`, `write-memory`, and `curate` shell out to the `engram` binary..." — generic mechanism clause, stays true either way; reviewer to confirm whether it needs a reword to note `learn` is now a runbook (matches how `write-memory`'s own retirement handled the identical sentence: kept the clause, just moved `write-memory` conceptually — this row is analogous) |
| `README.md:149` (tree comment) | update | "Source for the recall and learn skills" → "Source for the recall skill" |

### agent-instructions

| file:line | disposition | reason |
|---|---|---|
| `agent-instructions/skills/learn/` (whole dir: `SKILL.md`, `tests/README.md`, `tests/baseline-*.md`) | delete (task 5.3) | the retirement itself, performed after this enumeration and its independent review |
| `agent-instructions/skills/recall/SKILL.md` (mentions of `write-memory`, `learn` in prose) | keep | describes the handoff/mechanism generically ("hands off to learn" type language, if any) or references write-memory's basename directly (already repointed in a prior change); reviewer to grep fresh and confirm no "the learn skill" carrier claim exists here at execution time |
| `agent-instructions/guidance/shim.md` (top HTML comment) | update | "recall.md/delegate.md/learn.md, which stay as-is for sessions that still install the recall/learn skills" → "the recall skill" (singular), once `learn` is no longer an installable skill |
| `agent-instructions/guidance/learn.md` | **update — see Finding 2 above** | deployed guidance file whose body names `Skill` tool + `skill=learn` explicitly; becomes factually wrong once the skill is deleted. Proposal defers *retiring* this file, but its wording needs at minimum a carrier-neutral correction now — flagged for Joe, not resolved in this draft |
| `agent-instructions/guidance/{recall,delegate}.md` | N/A | no carrier-specific `learn`-as-skill wording (`recall.md` uses carrier-agnostic `/recall glance` phrasing only; `delegate.md` has no `learn` mentions) |
| `agent-instructions/skills/learn/tests/baseline-autonomous-trigger.md:11` | delete (part of dir) | "the learn skill self-fires" — baseline test prose, deleted with the rest of the skill dir, not a live doc reference elsewhere |

### docs

| file:line | disposition | reason |
|---|---|---|
| `docs/GLOSSARY.md:29-40` (`### skill` entry) | rewrite | "Engram ships two: [`recall`]... and [`learn`]... The end-to-end orchestration (`please`)... once shipped as skills too; all four are now runbook notes" → "Engram ships one: `recall`. ... `learn` and all four others once shipped as skills too; all five are now runbook notes" (or "ships zero" if recall has also retired by execution time — re-check ordering) |
| `docs/GLOSSARY.md:62-69` (`### write-memory (worker runbook)` entry) | update | "reached by basename/wikilink from recall/learn's own text (**they remain skills**)" — the parenthetical is wrong once `learn` retires; needs to say `recall` remains a skill while `learn` is now a runbook (a partial-sentence fix, mirrors design D7's own "partial-sentence carrier fix" instruction for the main spec) |
| `docs/GLOSSARY.md:319-322` (`### recall (skill)` entry) | N/A this task | unaffected by learn's own retirement; only touched by recall's own future conversion |
| `docs/GLOSSARY.md:470-472` (`### learn (skill)` heading + body) | rewrite | heading "learn (skill)" → "learn (runbook)" (mirrors the `write-memory (worker skill)` → `write-memory (worker runbook)` reword every prior conversion's glossary entry got); body "The skill at `agent-instructions/skills/learn/SKILL.md`, invoked as `/learn`..." → path swapped for the promoted runbook's basename |
| `docs/GLOSSARY.md:97,104,172,494,797` | keep | describe learn's Step 2/2.5 behavior generically ("the learn skill's third capture kind", "see `agent-instructions/skills/learn/SKILL.md`", "the learn skill's batch mode") — carrier-specific wording naming the dead path at 172 and 797 specifically; **reclassify 172 and 797 as update** (dead-path references, same class as `recall-payload-cuts/spec.md:90`), 97/104/494 as keep (describe the capture-kind concept, not the file path) |
| `docs/architecture/c1-system-context.md:23` (mermaid edge label) | update | `"R2: invokes /recall, /learn (skills) and runs engram CLI..."` → `/learn` moves out of the `(skills)` parenthetical |
| `docs/architecture/c1-system-context.md:47` (S3 row) | update | "Engram skills (recall, learn) are loaded by the harness's skill mechanism; the please, route, curate, and write-memory procedures are vault runbooks..." → `learn` moves into the vault-runbooks clause, leaving `recall` as the sole loaded-skill example |
| `docs/architecture/c1-system-context.md:57` (R2 row) | keep | "Invokes the `/recall` and `/learn` slash commands" — carrier-agnostic (a runbook is still invoked via a slash-shaped trigger phrase through `engram query --text`); reviewer to confirm no reword needed |
| `docs/architecture/c1-system-context.md:216` | keep | "the learn skill's Step 1.5 acts on the verdict autonomously (2026-07-03)" — dated historical narrative describing a past decision, not a current carrier claim |
| `docs/architecture/c2-containers.md:18` (mermaid skills box) | update | `` skills["C1 · Skills (learn / recall)<br/>markdown behavior specs"] `` → drop `learn` (leaves `recall`); learn moves into the C4 runbook annotation, mirroring how please/route/curate/write-memory moved out of this box during their own conversions |
| `docs/architecture/c2-containers.md:25` (mermaid edge label) | update | `` agent -->\|"runs /learn, /recall"\| skills `` → drop `/learn` |
| `docs/architecture/c2-containers.md:41` (C1 row prose) | rewrite | "`/learn` (`ingest --auto` + `fact`/`feedback` for explicit lessons) and `/recall`..." sentence structure needs `learn` moved out of the C1 skills description into the "no longer skills... `type: runbook` vault notes" sentence, mirroring the exact restructuring each prior conversion applied to this row |
| `docs/architecture/c2-containers.md:181` (sequence diagram participant) | keep or update — reviewer to confirm | `` participant Sk as C1 learn skill `` — a labeled diagram participant name; flag for the independent reviewer to judge whether the label itself needs a carrier-word change (mirrors the identical open question the write-memory conversion left for its own sequence-diagram labels) |
| `docs/architecture/c2-containers.md:207,235,250,263` | keep — reviewer to confirm | prose/flowchart-node text citing `agent-instructions/skills/learn/SKILL.md` by path (207, 235, 263) and a flowchart node `"learn skill Step 1.5: run the refit autonomously"` (250); these describe historical build provenance / a diagram node label, not necessarily live carrier claims — reviewer to decide whether path citations need the promoted runbook's basename instead |
| `docs/architecture/c2-containers.md:268` | keep | "The recall skill drives the loop" — recall-specific, unaffected by learn's own retirement |
| `docs/architecture/c3-components.md:62` (mermaid) | keep | `` skills -->\|"shell engram learn (args)"\| learn `` — describes the `engram learn` CLI subcommand invocation, not the skill carrier; the edge label names a CLI command, accurate regardless of carrier |
| `docs/architecture/c3-components.md:136` | keep | "ingest → skill → learn (NOT in-process)" — describes the mechanical `ingest`→CLI-subcommand flow, not a carrier claim |
| `docs/architecture/c3-components.md:234,250` | keep | "invoked when a skill (recall...)" / "args from the skill (write-memory kind=qa handoff, per recall Step 4 / learn Step 2.5)" — generic mechanism references to Step numbers, not carrier-specific to `learn` as a skill |
| `docs/architecture/adr.md:352,374` (ADR-0015) | keep | "the atomic-skills exploration evaluated decomposing the five skills (recall, learn, write-memory, please, route)" — dated 2026-07-04 decision record; unaffected, the decision itself doesn't change |
| `docs/architecture/adr.md:370-374` (ADR-0015 "Historical note (2026-09)") | update | already lists please/route/curate/write-memory's retirements; needs a further clause adding `learn`'s once this change lands, mirroring exactly how write-memory's own conversion appended its clause here |
| `docs/architecture/adr.md:817` | keep | "The learn skill's Step 1.5 runs derive → answer naming requests → apply" — dated historical decision record (refit CLI surface change), not a current carrier claim |
| `docs/ROADMAP.md:87` (NOW table, #760) | N/A this task; update at close-out (task 6) | tracks progress of #760 itself (recall/learn/write-memory runbook conversion); the row's own text and Shipped-table placement is this change's own tracking artifact, updated at close-out per the curate/write-memory precedent, not at 5.1/5.2 |
| `docs/ROADMAP.md:97,179,237,238,244` | keep | dated historical narrative (retrieval-value demonstration discussion, "trap regression harness" gating note, refuted-lever writeups) describing past measurements/decisions using "learn skill"/"recall skill" generically; not restated-as-current carrier claims |
| `docs/architecture/memory-invariants.md` | N/A | `recall`/`learn` mentions describe retired invariants and behavior generically (no `SKILL.md` path, no "the learn skill" carrier phrase) |
| `docs/research/**` | excluded | dated research records, not restated as current |

### openspec (main specs, not this change's own delta)

| file:line | disposition | reason |
|---|---|---|
| `openspec/specs/learn-branching-disposition/spec.md:9,17,22,27,32,36,42` | N/A this task; carrier fixed via this change's own delta (already exists at `openspec/changes/learn-skill-to-runbook/specs/learn-branching-disposition/spec.md`), syncs at archive | "Learn skill decides note placement..." — requirement-body carrier wording; delta already written per design D7, confirmed present |
| `openspec/specs/learn-adhoc-qa-capture/spec.md:20` | N/A this task; carrier fixed via this change's own delta (`specs/learn-adhoc-qa-capture/spec.md`), syncs at archive | "within the learn skill's own Step 2 processing" — same treatment |
| `openspec/specs/write-memory-worker/spec.md:9,11,13,19,22,24-25,93-115` | N/A this task; carrier fixed via this change's own delta (`specs/write-memory-worker/spec.md`), syncs at archive | "invoked by a parent skill (`recall`, `learn`)" and "Learn skill SHALL capture four kinds..." — requirement-body/title carrier wording; delta already exists (and now carries the symmetric Ordering note added by Job 2 of this task, covering the archive-ordering conflict with `recall-glance-skill-to-runbook`'s own delta to the same requirement) |
| `openspec/specs/guidance-runbook-follow-frame/spec.md` | N/A | no new requirement expected for `learn` becoming a runbook (the query-matched/triggered case is already covered); confirm no incidental `learn`-as-skill wording exists at execution-time re-grep |
| `openspec/changes/learn-rate-skill-only/{proposal,design,tasks}.md`, `specs/{learn,please}/spec.md` | keep (false positive) | separate, active, unrelated change (learn-rate curation quality calibration). "learn skill" mentions there concern kind-4 exemplar/curation-bar tuning, e.g. "Edit `agent-instructions/skills/learn/SKILL.md` **Step 2**..." (task 1.3) — this task literally edits the live `SKILL.md` file; if `learn-rate-skill-only` is still active/unmerged when `learn-skill-to-runbook` executes its retirement, the two changes will conflict on the same file (`learn-rate-skill-only`'s task assumes the skill file exists to edit) — **flag for the independent reviewer to check `learn-rate-skill-only`'s current status before deleting `agent-instructions/skills/learn/`**, not resolved in this draft |

### code and dev (live, non-eval-history)

| file:line | disposition | reason |
|---|---|---|
| `dev/eval/cumulative/runbook_vs_skill/probe.py::build_cfg_template` (default `skills=("recall","learn")`), used via `phase2/probe_phase2.py::build_cfg_template_phase2`/`build_cfg_pool_phase2` | keep — reviewer to confirm, see Finding 1 | live generator hardcoding the real skill path as its default; correct for this change's OWN task 1.4 skill-baseline run (needs the real `learn` skill present), but silently degrades for any future caller after retirement — no code change proposed here |
| `dev/eval/audit/extract.py` (skill-name detection: `skill_name == "recall"`, `skill_name in ("learn", "write-memory")`, lines ~562,591 and surrounding transcript-parsing logic) | keep — reviewer to confirm | hardcodes `Skill(learn)`/`Skill(recall)` tool-name detection for parsing PAST session transcripts (correct for historical transcripts that really invoked the Skill tool). Transcripts recorded AFTER `learn` retires will invoke it via `engram show <basename>` in Bash, not `Skill(learn)` — same open risk the write-memory conversion's own enumeration flagged and left unresolved (out of this change's stated scope, but the risk compounds: learn's retirement is the SECOND of the three original recall/learn/write-memory audit-detection paths to go stale) |
| `dev/eval/audit/build_scorecard.py:434`, `dev/eval/audit/results/scorecard-main.json` | keep | "the recall/learn Skill invocation being detected in the transcript" — scorecard label/data describing historical audit results, generic |
| `dev/eval/cumulative/harness.py`, `dev/eval/cumulative/run_recheck.py`, `dev/eval/cumulative/smoke_prune.py`, `dev/eval/cumulative/synthesis_judge.py`, `dev/eval/cumulative/aggregate.py`, `dev/eval/cumulative/lever_recheck/*.py`, `dev/eval/cumulative/endorse_cue/probe.py`, `dev/eval/traps/*.py` | keep | large, live eval-instrument corpus that prompts/detects "the `/learn` skill" as its own arm-under-test — these are a DIFFERENT eval track (the original recall/learn cost-and-value measurement harness, not `runbook_vs_skill`) whose whole premise is comparing the real installed skill's behavior; they are not asserting the skill is permanently installed, they DRIVE it as their treatment. Out of this change's scope (deleting `agent-instructions/skills/learn/` would break these instruments' warm-skill arms, same as it will affect `probe.py`'s default) — **flag prominently for the independent reviewer and for Joe**: this is a materially larger blast radius than any prior conversion (please/route/curate/write-memory had no such standing measurement corpus depending on their skill being installed) |
| `dev/eval/traps/test_wrun.py:23,31-34` | keep — reviewer to confirm | asserts both `recall`/`learn` `SKILL.md` files exist at a deployed destination — a real regression-test assertion that will start failing once `learn`'s directory is deleted and re-deployed via `engram update`. Needs updating (or explicit skip, mirroring curate's own `test_curate_skill_src_is_byte_identical_to_the_live_skill` self-adapting skip) once `learn` retires — flag for the independent reviewer, not fixed here (test-code change, out of "draft docs only" scope) |
| `internal/update/*_test.go`, `internal/cli/{update,update_deps,invariants_u1}_test.go` | keep, no action | see Exclusions/Finding 1 — use `"learn"` as an arbitrary fixture skill name in a fake filesystem testing the generic sync engine; do not assert current shipped-skill status |

## Counts by disposition (draft; re-verify at execution time)

- update: 14
- rewrite: 6
- keep: ~26 (includes reviewer-flagged rows and generic/false-positive-shaped rows)
- N/A (this task; deferred to archive/close-out or already covered by this change's own delta): 6
- excluded (directory/file-level groups): 6
- delete: 1 (the skill dir itself + its `tests/` subdir, task 5.3)

## Open items for the independent reviewer (in addition to a full fresh re-grep)

1. **Finding 2 (guidance/learn.md)** — decide whether to fix its "Skill tool, skill=learn" wording
   now (a small, scoped correction) versus leaving it broken until the deferred guidance-retirement
   fast-follow change. This is a real behavioral regression risk for every session that still
   `@import`s `learn.md`, not just a documentation nit.
2. **`learn-rate-skill-only`'s live status** — confirm whether that change's own task 1.3 (which
   edits the live `agent-instructions/skills/learn/SKILL.md` Step 2) has landed, is abandoned, or
   is still pending before task 5.3 deletes the file out from under it.
3. **Finding 1 (`probe.py`/`extract.py`/the broader `dev/eval/cumulative` + `dev/eval/traps`
   corpus)** — this change has a materially larger live-instrument blast radius than any prior
   skill-to-runbook conversion, because a standing cost/value measurement harness (distinct from
   `runbook_vs_skill`) treats the installed `learn` skill as its object of study. Decide whether
   any of these need an explicit skip/adapter before or immediately after retirement, versus
   accepting they become permanently historical instruments.
4. Confirm which ordering (`learn` first vs `recall` first) is actually true at execution time and
   correct every "drop to one: `recall`" / "engram ships one" row accordingly if `recall` has
   already retired.
5. Re-grep `docs/GLOSSARY.md:172,797` and confirm whether they are dead-path citations needing the
   promoted runbook's basename, or generic enough to leave as prose pointers (this draft's read
   leans toward "dead path, needs update," but wasn't independently re-verified).
