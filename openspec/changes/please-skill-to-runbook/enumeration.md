# Task 5.1 enumeration: retiring `agent-instructions/skills/please/SKILL.md`

Status: 2026-09-20, independent fresh-context review applied (corrections below marked `[review]`); every
`update`/`rewrite` row performed and verified in the same pass. Task 5.1 ticked with evidence in tasks.md.

## Grep run

Case-insensitive, whole repo, `-I`:
`\bplease\b|/please|please-doc|orchestrat|end-to-end skill|gate [A-D]\b|lessons.audit|Lessons audit`
plus by eye: `please skill`, `please SKILL.md`, `please-doc-enumeration-gate`, `lessons audit`, `Step 7`.

Surfaces covered: `CLAUDE.md`, `README.md`, `.claude/` (rules, commands, skills, project-config), `agent-instructions/`
(skills, guidance), `docs/` (GLOSSARY, ROADMAP, README index, agents/, architecture/, research/, design/, superpowers/),
`openspec/specs/**`, `openspec/changes/*` (active), `cmd/`, `internal/`, `dev/` (non-eval).

## Exclusions (stated)

- `openspec/changes/archive/**` (dated records).
- `dev/eval/**` result files, fixtures, harness history and LEDGER rows only: historical measurement records; the
  fixture runbooks live there by design. Not edited by retirement. [review] NOT excluded: live harness code and
  procedure docs (`please_step7_probe/run_probe.py` + README, `please_step3_probe/README.md`, `traps/README.md`),
  see the dev/eval rows below.
- `openspec/changes/please-skill-to-runbook/{proposal,design,tasks}.md` and `openspec/changes/runbook-lexical-triggers/**` is a different, shipped change (N-A, dated record); its
  tasks.md:82 names `please/SKILL.md` historically. [review] Only this change's own proposal/design/tasks are self-artifacts (N-A).
- `internal/embed/assets/model/tokenizer.json`: vocabulary token "please" (N-A, matched by grep only).
- `.claude/worktrees/**` other worktrees.

Result of grep over `.claude/`, `docs/agents/`, `dev/skillsmeta`, `dev/targs.go`, `docs/README.md`, `cmd/`: **no hits**
except `.claude/skills/engram-go-conventions.md:209` ("thin orchestrator" code-comment example; N-A).
`internal/**` non-test code: no `please` hits (only "merge orchestrator" comments, N-A; `Gate B` in
`prune_duplicates.go` is an unrelated measurement label).

## Per-file disposition

Legend: keep / update / rewrite / N-A.

### Root and agent-facing docs

| file:line | disposition | reason |
|---|---|---|
| CLAUDE.md:3 | update | says "A third skill, `please`, orchestrates end-to-end work..." and that `route` consulted by please: now a vault runbook (note 1045) surfaced by triggers, not a skill; reword the skill count sentence |
| CLAUDE.md:26 | update | dir comment "Source for the recall, learn, write-memory, please, and route skills" drop please |
| CLAUDE.md:38 | update | Key Files glob `{learn,recall,write-memory,please,route}/SKILL.md` drop please |
| CLAUDE.md:40 | update | "four key flows (recall, learn, please, update)": please flow now documented as a runbook flow, adjust wording once c1 is rewritten |
| README.md:10 | update | "A further skill, `please`, orchestrates..." now a runbook |
| README.md:50 | rewrite | skills-table row for `please` (seven-step description) becomes a runbook description or a note under the table |
| README.md:51 | update | route row mentions please consulting it for gate reviewers: rephrase |
| README.md:53 | update | "See ...`please/SKILL.md`" link dangles after deletion |
| README.md:149 | update | tree comment lists please among skills |
| README.md:92 (prune line, now :93) | update (done) | [review] 'Not part of the recall/learn/please flows' is the `prune` entry, not unrelated; reworded to 'recall/learn flows' (same string as GLOSSARY:690) |

### agent-instructions

| file:line | disposition | reason |
|---|---|---|
| agent-instructions/skills/please/SKILL.md (whole dir, 1 file) | delete (task 5.3) | the retirement itself; do after 5.1/5.2 and gate (b),(c) confirmed |
| agent-instructions/guidance/delegate.md:10 | update | "please's gate verdicts" names the skill; say the please runbook's gates |
| agent-instructions/guidance/delegate.md:31 | update | "for a full end-to-end ask, `/please`" still valid as a trigger; reword to "the please runbook (surfaced by `/please`)" |
| agent-instructions/guidance/delegate.md:5,8 | keep | "orchestrator" is the persona, generic |
| agent-instructions/guidance/shim.md | keep | mentions no please; carries `--text` and phrase-two re-check (already deployed) |
| agent-instructions/guidance/recall.md, learn.md | keep | no please references |
| agent-instructions/skills/learn/SKILL.md:89 | update | "the orchestrator (`please`) hands ..." refer to the please runbook (SKILL.md edit needs writing-skills TDD) |
| agent-instructions/skills/learn/SKILL.md:83 | N-A | polite "Please schedule" in prose |
| agent-instructions/skills/write-memory/SKILL.md:35,85,86 | keep | `/please` and "please" appear as trigger-authoring examples; still correct |
| agent-instructions/skills/recall/SKILL.md, curate/SKILL.md | keep | no please hits |

### docs

| file:line | disposition | reason |
|---|---|---|
| docs/GLOSSARY.md:30-34 (`skill` entry: "Engram ships five... `please`") | rewrite | now four skills (recall, learn, write-memory, curate) + route already a runbook; please a runbook |
| docs/GLOSSARY.md:43 (`atom`: "orchestrate-a-workflow") | keep | historical decomposition framing |
| docs/GLOSSARY.md:95-96 (`lessons audit`) | rewrite | "The please skill's step-7..." -> please runbook / lessons-audit sub-runbook |
| docs/GLOSSARY.md:108,117 (`surprise harvest`) | rewrite | "addition to the please skill's Step-7"; Status cites `agent-instructions/skills/please/SKILL.md (662e50ba)` which will be deleted: point at the runbook or record as folded into the lessons-audit sub-runbook (check note body first) |
| docs/GLOSSARY.md:130 (`escalation provenance`) | rewrite | "The please skill's rule..." |
| docs/GLOSSARY.md:423 | keep | trigger authoring rule example `/please` (correct) |
| docs/GLOSSARY.md:690 (`prune`: "Not part of the recall/learn/please flows") | update | reword "please flows" to "recall/learn flows" or leave as workflow name |
| docs/GLOSSARY.md:866 (`subagent`: "please's gate reviewers") | update | runbook gates |
| docs/GLOSSARY.md:890-892 (`capture guards`) | update | G2/G6 cite the SKILL.md path; repoint to the runbook |
| docs/architecture/c1-system-context.md:23,57 | update | R2 lists `/please` as a slash command the harness invokes: now a trigger-fired runbook, not an installed slash command |
| docs/architecture/c1-system-context.md:46 | update | S2 description says operator tools sit "outside the recall/learn/please/update flows": reword |
| docs/architecture/c1-system-context.md:249-304 (Flow: please) | rewrite | "skill-only orchestration", "load please skill"; steps/gates unchanged, carrier is the runbook fetched via `engram query` (trigger hit) |
| docs/architecture/c1-system-context.md:399-569 (companion: gates A-D, `/please` seven-step flow, refs to `please/SKILL.md`, sibling-of-skills line 569) | rewrite | same carrier change; L425,453,304,541 cite SKILL.md, repoint |
| docs/architecture/c2-containers.md:18,25 | update | C1 Skills box lists please and "runs the 7-step /please" |
| docs/architecture/c2-containers.md:41 | update | container catalog C1 row lists `/please (7-step bracket)` |
| docs/architecture/c2-containers.md:219,235 | update | lessons-audit diagram source cites `please/SKILL.md` (Step 7) |
| docs/architecture/c2-containers.md:265 | N-A | "skill-orchestrated" generic |
| docs/architecture/c3-components.md:10,24,33,154 | N-A | "skills orchestrator"/"query orchestrator" generic |
| docs/architecture/adr.md:353-358 (ADR-0015) | keep | dated decision: "Leave `please` and `route` ..." historical; add nothing |
| docs/architecture/adr.md:953 | update | ADR-0026 amendment says "The parked change `please-skill-to-runbook`": now unparked and retiring; edit the sentence |
| docs/architecture/adr.md:108,418,525,532 | N-A | generic "orchestrated"/"orchestrator" |
| docs/architecture/memory-invariants.md:109 | update | names "please step-ordering" as an RT-only discipline item; carrier now a runbook |
| docs/ROADMAP.md:226,234 | keep | dated shipped/refuted history rows (#685, #687) |
| docs/ROADMAP.md:95,101,117,221,238,243 | N-A | "Gate B/A" review labels, generic, not the skill |
| docs/ROADMAP.md (no #758 row found by grep) | N-A | no row to update; add one only if Joe wants the retirement tracked there |
| docs/research/2026-08-30-*.md (6 files) | N-A | dated research; `memory-taxonomy-engram-map.md:55,136` describe the skill as of 2026-08-30 |
| docs/superpowers/plans/2026-08-26-733-*.md | N-A | dated plan record |
| docs/design/2026-07-01-*.md | N-A | dated design record |
| docs/agents/*, docs/README.md | N-A | no hits |

### openspec

| file:line | disposition | reason |
|---|---|---|
| openspec/specs/please-doc-enumeration-gate/spec.md:1,5 (title + Purpose "The please skill's Step 3") | update at archive (task 5.6) | the delta MODIFIES only the requirements (verified: it modifies "Plan author SHALL run doc-surface enumeration grep..." with a new "Carrier is the runbook, not a skill" scenario and "Gate A docs/diagrams-alignment reviewer..."); deltas cannot edit Purpose or title, so edit them when syncing at archive |
| openspec/specs/please-doc-enumeration-gate/spec.md:9-40 (requirements 3 and 4: disposition-list format, non-waivable) | keep | contain no skill-carrier wording, so no delta needed (verify at review) |
| openspec/specs/please-doc-enumeration-gate/spec.md:20-24 | keep (delta exists) | Gate A requirement covered by the delta |
| openspec/specs/write-memory-worker/spec.md:114-120 (Step 7 lessons audit requirement) | keep (delta exists) | delta `specs/write-memory-worker/spec.md` re-points it at the runbook |
| openspec/specs/write-memory-worker/spec.md:5 | N-A (verified) | Purpose names no please path (grep of the file: only lines 114-120, covered by the delta); no archive-time task needed |
| openspec/specs/guidance-runbook-follow-frame/spec.md | keep (delta exists) | delta present (phrase-two re-check); main spec syncs at archive |
| openspec/specs/guidance-delegate/spec.md:5 | N-A | "orchestrator" reflex persona; does not name please |
| openspec/specs/route-dispatch-evidence, route-evidence-rubric, memory-loop-audit (`orchestrator`) | N-A | generic actor name, not the please skill |
| openspec/changes/learn-rate-skill-only/specs/please/spec.md (whole, "Please Capability") | update | ACTIVE unarchived change with a delta spec for capability `please` that would create `openspec/specs/please`; it describes please as a skill collecting `LESSONS:` lines. Needs a decision (Joe): re-carrier to the runbook wording before its archive, or archive it first. Not covered by this change's deltas |
| openspec/changes/learn-rate-skill-only/specs/lessons-contract/spec.md:28 | update | "The orchestrator (please skill) MUST collect LESSONS lines": same carrier wording issue |
| openspec/changes/learn-rate-skill-only/{proposal,design,tasks}.md, specs/route/spec.md | update (review) | mention please; dated design rationale, wording only if the change is still going to archive |
| openspec/specs/update-deploy-sync/spec.md:13 | keep | [review] scenario 'Removed source artifact disappears on next update' is generic (a deleted skill dir); it is the mechanism 5.3 relies on |
| openspec/changes/learn-rate-skill-only/specs/please/spec.md (ADDED; cites the SKILL.md path twice) | deferred — Joe's call: archive that change first vs reword | not edited by this pass |
| openspec/changes/learn-rate-skill-only/specs/lessons-contract/spec.md:7 ('route skill') and :28 ('please skill') | deferred — Joe's call: archive that change first vs reword | not edited by this pass |
| openspec/changes/runbook-retrieval-probe/** | N-A | no please hits |
| openspec/changes/archive/** | excluded | see exclusions |

### code and dev

| file:line | disposition | reason |
|---|---|---|
| internal/update/sync_test.go:715-781 | keep | uses `please/` only as a fixture name for skill-directory deletion; it is in fact the existing test of the deleted-skill sync path (dangling surface link and emptied dir removal) |
| internal/cli/query_runbook_test.go, learn_test.go, learn_adapters_test.go, amend_test.go, embed/hash_test.go, serve_client_test.go (`/please`, "PLEASE   TAKE   this\tend-to-end") | keep | trigger-matching test fixtures use `/please` as an example string |
| internal/cli/ingest_test.go, ingest_sweep_test.go, query_unified_test.go, internal/context/context_test.go ("please wire the linter", "please help me") | N-A | plain English in transcript fixtures |
| cmd/engram/** , internal/** non-test code, dev/targs.go, dev/*.toml, dev/skillsmeta | N-A | no hits (help text has no please reference) |
| dev/eval/cumulative/please_step7_probe/run_probe.py:~490 | update (done) | [review] defaulted `--skill-text` to `agent-instructions/skills/please/SKILL.md` and exit(1)'d if missing; repointed to `skill-text/please-SKILL.md.frozen` (verbatim `git show HEAD:` copy: the probe measures that skill body, so freezing keeps it reproducible) |
| dev/eval/cumulative/please_step7_probe/README.md:3,138 | update (done) | names the live path; now describes the frozen default |
| dev/eval/cumulative/please_step3_probe/README.md:3,37,40 | update (done) | [review-adjacent] usage examples passed the live SKILL.md path; repointed to the frozen copy |
| dev/eval/traps/README.md:4,42 | update (done) | [review] live procedure; reworded to 'recall/learn skill body or the please runbook' |
| dev/eval/cumulative/runbook_vs_skill/phase2/** (fixtures, `skill_src: encodings/taskPlease/Please-S/skills/please`, encodings) | keep | own self-contained fixture copies; do not read `agent-instructions/skills/please/` |
| other dev/eval results/LEDGER | excluded | see exclusions |

## Counts by disposition (rows above; grouped rows count once)

- update: 24 (+1 update at archive, +1 update (review) = 26)
- rewrite: 7
- keep: 12 (+3 keep with existing delta = 15)
- N-A: 16
- excluded: 2
- delete (the skill dir itself): 1

(Counts are of table rows, computed by script; grouped line ranges count once.)

## What `engram update` sync will remove

Dry run, `engram update --dry-run --with-guidance` (2026-09-20, from this worktree, rev ec5f3735), while
`agent-instructions/skills/please/` still exists in the source:

```
[dry-run] engram update
  source: local clone at ~/repos/personal/engram/.claude/worktrees/runbook-vs-skill (rev ec5f3735)
  Claude Code (~/.claude/):
    agent-instructions/skills/curate/ -> ~/.claude/skills/curate/  (1 file)
    agent-instructions/skills/learn/ -> ~/.claude/skills/learn/  (9 files)
    agent-instructions/skills/please/ -> ~/.claude/skills/please/  (1 file)
    agent-instructions/skills/recall/ -> ~/.claude/skills/recall/  (7 files)
    agent-instructions/skills/write-memory/ -> ~/.claude/skills/write-memory/  (1 file)
  Pi (~/.pi/): same five, incl. agent-instructions/skills/please/ -> ~/.pi/agent/skills/please/  (1 file)
  guidance refreshed (delegate, learn, recall, shim) for both harnesses
```

So the dry run today plans **no deletions** (please is still in the source). The deployed state on this
machine, verified with `ls -ld`:

- `~/.claude/skills/please` is a symlink to `~/.claude/engram/skills/please` (real dir, 1 file)
- `~/.pi/agent/skills/please` is a symlink to `~/.pi/agent/engram/skills/please`

After `agent-instructions/skills/please/` is deleted (task 5.3), sync should (per `internal/update/update.go`
`EngramSyncDelete` "file present under a managed subtree but absent from the intended set", and the
dangling-link removal reported as `DanglingLinksRemoved`, exercised by `sync_test.go:715-781`) remove:

1. `~/.claude/engram/skills/please/SKILL.md` and the emptied `~/.claude/engram/skills/please/`
2. `~/.pi/agent/engram/skills/please/SKILL.md` and the emptied dir (assumed same mechanism; not yet observed)
3. the now-dangling symlinks `~/.claude/skills/please` and `~/.pi/agent/skills/please`

NOT VERIFIED: the post-deletion dry run, since deleting the source is out of scope for this stage. Re-run
`engram update --dry-run --with-guidance` right after the delete in 5.3 and paste the planned deletions here
before running the real update.

## Skill-list reconciliation ([review] fix 3, performed)

`ls agent-instructions/skills/` after deletion: curate, learn, recall, write-memory. route is already a
runbook (retired 2026-09-19), please is a runbook. Lists corrected to match:

| place | now says |
|---|---|
| CLAUDE.md:3 | recall, learn, write-memory (third), curate (fourth); please and route as runbooks |
| CLAUDE.md:26,38 | `recall, learn, write-memory, and curate skills`; Key Files glob `{learn,recall,write-memory,curate}` with a note that please/route are runbooks |
| README.md:10,50-54,150 | four skills (curate row added to the table); please and route described as runbooks below the table |
| docs/GLOSSARY.md `skill` | 'Engram ships four' + please/route now runbooks; new `### runbook` entry |
| docs/architecture/c2-containers.md:18,25,41 | Skills (learn / recall / write-memory / curate); please/route live as runbook notes in C4 |
| docs/architecture/c1-system-context.md:23,47,57,249+,569 | R2 skills are recall/learn; /please is a runbook matched by literal trigger; route is a runbook |
| c3-components.md | no skill list (generic orchestrator wording only), keep |

## Task added for archive time

- please-doc-enumeration-gate main spec title and Purpose still say "please skill's Step 3"; deltas cannot edit
  Purpose or title. Tracked as task 5.6 in tasks.md.
- write-memory-worker main spec Purpose (line 5) cites `agent-instructions/skills/please/SKILL.md (commit 662e50ba)`; a delta cannot edit Purpose. Archive-time row, tracked as task 5.7 in tasks.md.
