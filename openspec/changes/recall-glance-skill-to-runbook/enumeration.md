# Task 5.1 enumeration (DRAFT): retiring `agent-instructions/skills/recall/SKILL.md`

**STATUS: DRAFT — built ahead of the paid validation gate (2026-09-23), before sections 3/4 of
tasks.md have run.** This is a static-analysis pass over the CURRENT repo state, useful because no
eval outcome changes which docs mention `recall` as a skill today. It MUST be re-verified for
freshness (re-grep everything below) immediately before task 5.1's real execution, since repo
state may drift between now and then (this change's own tasks 1.5–4.4, and the sibling
`learn-skill-to-runbook` change, may land before this one does — see "Ordering uncertainty" below).
Do NOT treat this file as the gate-cleared enumeration; task 5.1 itself still requires an
independent fresh-context reviewer running its own discovery pass AFTER a fresh re-grep, per the
write-memory/curate/please precedent. **Task 5.1 is NOT ticked in tasks.md.**

`recall` is referenced more widely than any prior conversion in this session (please, route,
curate, write-memory) — it is the memory-lookup mechanism every other skill/runbook cites, and it
has an unusually large standing measurement-instrument corpus under `dev/eval/` treating it as the
object of study (see Finding 1). Treat this enumeration as thorough-but-not-exhaustive; the
independent reviewer's own fresh pass matters more here than for any prior conversion.

## Ordering uncertainty (read before acting on this list)

At the time of this draft, neither `recall-glance-skill-to-runbook` nor `learn-skill-to-runbook`
has archived, and both are being worked in the same worktree. This list assumes **recall retires
alone, with `learn` still a skill** (the scope of this change). The real ordering may differ:

- **Recall retires first (learn still a skill):** the skill count in `agent-instructions/skills/`
  drops from two to **one** (`learn`). Rows below assuming this hold as written.
- **Learn has already retired by the time this lands:** the skill count drops to **zero** once this
  change also lands. Several rows below (anything saying "engram ships one: learn" or listing
  `learn` as the remaining skill) need a further edit this draft cannot pre-write, since it depends
  on exactly how `learn-skill-to-runbook`'s own doc updates landed. **Re-check this explicitly at
  execution time.**

## Grep run

Case-insensitive, whole repo: `\brecall skill\b`, `\bthe recall skill\b`, references to `glance`/
`deep` mode as skill-argument selection, `Step 0`–`Step 4` pipeline-stage prose tied to "the recall
skill", plus a broad scan of `recall|learn` mentions in `CLAUDE.md`, `README.md`,
`docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,c3-components,adr}.md`,
`docs/ROADMAP.md`, `agent-instructions/**`, and every `openspec/specs/recall-*` main spec plus
`write-memory-worker` and `guidance-runbook-follow-frame`. Targeted follow-ups: the literal path
string `agent-instructions/skills/recall` across `*.md`/`*.py`/`*.go`/`*.json`/`*.toml` repo-wide
(to catch dead-path citations inside spec Purpose lines, not just requirement bodies — see
Finding 2), and `dev/eval/audit/extract.py` for hardcoded `Skill(recall)` transcript-parsing logic.

## Exclusions (stated)

- `dev/eval/cumulative/runbook_vs_skill/phase2/results/**`: dated measurement records.
- `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/{taskRecall,taskLearn,taskPlease,
  taskRoute,taskCurate,shim}/**`, `.../encodings/recall-conversion-fidelity-report.md`,
  `.../fixtures/{recall-glance,recall-escalation}/**`: this change's own (and sibling changes')
  fixture/report artifacts — frozen copies, not live references.
- `openspec/changes/recall-glance-skill-to-runbook/{proposal,design,tasks,enumeration}.md` and its
  own delta specs (`specs/{recall-payload-cuts,write-memory-worker}/spec.md`): self-artifacts.
- `openspec/changes/learn-skill-to-runbook/**`: the sibling change's own self-artifacts — its own
  5.1 enumeration is `openspec/changes/learn-skill-to-runbook/enumeration.md`, drafted separately.
- `openspec/changes/archive/**`: dated, already-archived change records.
- `docs/research/**`, `docs/superpowers/plans/2026-08-26-733-recall-chunk-only-cluster-read-gate.md`:
  dated historical records of a completed cycle (#733), not restated as current — mirrors the
  `docs/research/**` treatment every prior conversion applied. (This plan file has 9 live-path
  citations to `agent-instructions/skills/recall/SKILL.md`, all past-tense/historical — e.g. "Does
  **not** modify ... in this change", "future implementation target" for a target now itself
  historical.)
- `dev/eval/cumulative/runbook_vs_skill/PLAN.md`, `README.md`: this eval's own design docs
  describing the harness's skill-deployment mechanism generically ("discovered the same way the
  recall skill is discovered") — infra description, not a carrier claim about `recall` itself.

## Findings worth flagging before executing

**Finding 1 (materially larger live-instrument blast radius than any prior conversion):** a large,
live, standing eval-instrument corpus under `dev/eval/traps/` and `dev/eval/cumulative/` (distinct
from the `runbook_vs_skill` eval this session's conversions use) treats the installed `recall`
skill as its actual object of measurement — `wrun.py`, `recency_value.py` (+ `recency_value/
{score,test_score}.py`), `compound_fixtures.py`, `reasoning_recall_eval.py`, `cake.py`, `c6.py`,
`c4_idio.py`, `persist_green_check.py`, `graphexpand_warm.py`, `recall_time.py`,
`dev/eval/cumulative/{harness,run_recheck,smoke_prune,synthesis_judge,aggregate}.py`,
`dev/eval/cumulative/lever_recheck/{stub_engram,RESULTS.md}`, `dev/eval/cumulative/endorse_cue/
probe.py`. These prompts literally say things like `"Invoke your /recall skill..."` and detect
`Skill(recall)` tool_use events in transcripts. None of these are asserting recall is permanently a
skill — they DRIVE it as their treatment arm — but every one of them stops working (or silently
degrades to a no-op / a false "recall did not fire" reading) once `agent-instructions/skills/
recall/` is deleted. This is out of THIS change's stated scope (design.md's Non-Goals don't mention
this corpus, mirroring how write-memory's own conversion left `dev/eval/audit/extract.py`'s
analogous risk unresolved) but is considerably larger here — `recall` has an entire historical
cost/value measurement program built around invoking it as a real skill, which `learn`, `please`,
`route`, `curate`, and `write-memory` did not have to the same degree. **Flag prominently for the
independent reviewer and for Joe**: decide whether any of this corpus needs an explicit
freeze/skip/adapter note before or immediately after retirement, or whether it is accepted as
becoming a permanently historical instrument set once `recall` is a runbook.

**Finding 2 (a real gap design.md's D4 didn't cover): `openspec/specs/recall-glance-deep-dial/
spec.md:5` — its Purpose line, not a requirement body — names the dead path directly:**
"Implementation and behavior defined in agent-instructions/skills/recall/SKILL.md (Modes
section)." Design.md's D4 only identified `recall-payload-cuts/spec.md:90` (a requirement body,
fixable via delta) as needing a fix, and stated "the other `recall-*` specs describe behavior in
carrier-agnostic terms and need no delta — confirmed by direct read." That confirmation is correct
for requirement bodies, but this Purpose line was not caught, and a delta **cannot** edit a Purpose
line (the same limitation the write-memory conversion's own Purpose-line fix — its task "added for
archive time" — worked around). **This needs its own archive-time task, mirroring write-memory's
precedent exactly**: reword `recall-glance-deep-dial/spec.md`'s Purpose line at archive time to
name the promoted runbook(s) (`recall-glance`/`recall-deep`/the shared core sub-runbook, whatever
D1's final split settles on) instead of the dead `SKILL.md` path. **Recommend adding this as a new
tasks.md item in section 5 (mirroring `learn-skill-to-runbook`'s equivalent gap, which has none
either, and the write-memory conversion's own task 5.5).** Confirmed by direct read that no other
`recall-*` spec's Purpose line has this problem (`recall-two-channel-payload`, `recall-centroid-
sampling`, `recall-matched-note-floor`, `recall-query-timings`, `recall-runbook-surfacing` — none
mention the skill path).

**Finding 3 (shared with `learn-skill-to-runbook`'s Finding 1): `dev/eval/cumulative/
runbook_vs_skill/probe.py::build_cfg_template`'s live default `skills=("recall", "learn")`** — see
that change's enumeration for detail; applies identically here. Not fixed in this draft.

**Finding 4 (shared with `learn-skill-to-runbook`'s Finding on `extract.py`): `dev/eval/audit/
extract.py`'s hardcoded `skill_name == "recall"` transcript-parsing detection** (and the identical
`Skill(recall)`-shape assumption in `build_scorecard.py`/`test_extract.py`) will not recognize a
post-retirement `engram show <basename>` invocation shape in future transcripts. Same open risk
the write-memory conversion's own enumeration already flagged and left unresolved for its own
carrier; recall is now the second (of three total: write-memory done, learn pending, recall
pending) of the original detection paths to go stale. Not fixed in this draft.

**Finding 5 — no `guidance/recall.md` equivalent of `learn.md`'s Finding 2.** Checked directly:
`agent-instructions/guidance/recall.md`'s body only ever says `` `/recall glance` ``/`` `/recall
deep` `` — a slash-command phrase compatible with either the installed-skill mechanism or a
runbook's own trigger phrase (per design D5, `/recall`, "recall glance", "recall deep" are exactly
the candidate `triggers:` this change considers adding). Unlike `learn.md` (which literally names
the `Skill` tool and `skill=learn`), `recall.md` needs **no** correction regardless of whether
`triggers:` end up being added — confirmed by direct read of the full file, not assumed.

## Per-file disposition

Legend: keep / update / rewrite / N/A / delete / excluded.

### Root and agent-facing docs

| file:line | disposition | reason |
|---|---|---|
| `CLAUDE.md:3` | rewrite | "Two skills — `recall` and `learn`..." → one skill (`learn`, assuming recall retires while learn hasn't yet — re-check ordering), fold `recall` into the runbook-list sentence with its two entry points (`recall-glance`, `recall-deep`) named, mirroring the fold-in pattern each prior conversion used |
| `CLAUDE.md:26` (dir tree comment) | update | "Source for the recall and learn skills" → "Source for the learn skill" (or adjust per actual ordering) |
| `CLAUDE.md:38` (Key Files) | update | drop `recall` from the `{learn,recall}/SKILL.md` glob; add to the runbook parenthetical |
| `CLAUDE.md:49` (Design Principles) | update | "Skills for behavior (learn, recall)..." → drop `recall` |
| `README.md:10` | rewrite | "Two skills — `recall` and `learn`..." sentence restructured the same way as `CLAUDE.md:3`; the runbook-enumeration sentence needs a `recall-glance`/`recall-deep` clause added, distinct from every prior single-entry-point runbook description |
| `README.md:12` | keep | "`recall` and `learn` work the same on each" — generic, stays true; reviewer to confirm |
| `README.md:45-48` (skills table) | rewrite | `recall` row removed from the skills table (leaves `learn` alone, or an empty table if `learn` also retired by then) |
| `README.md:50` | update | drop the `agent-instructions/skills/recall/SKILL.md` link |
| `README.md:52` | update | add `recall` (as `recall-glance`/`recall-deep`) to the "have no SKILL.md" runbook list and its explanatory sentence |
| `README.md:53` | update | "`recall`, `learn`, `write-memory`, and `curate` shell out to the `engram` binary..." — reviewer to confirm whether reword needed (mirrors the analogous row in `learn`'s own enumeration) |
| `README.md:149` (tree comment) | update | "Source for the recall and learn skills" → adjust per ordering |

### agent-instructions

| file:line | disposition | reason |
|---|---|---|
| `agent-instructions/skills/recall/` (whole dir: `SKILL.md`, `tests/README.md`, `tests/baseline-*.md`) | delete (task 5.3) | the retirement itself, after this enumeration and independent review |
| `agent-instructions/skills/learn/SKILL.md` (mentions of `recall` at lines 4,9,105,186,213,242 — generic prose: "after work begun with recall", "recall's Step 4", "future recall") | N/A this task | generic mechanism references, not a "the recall skill" carrier claim; would need re-check only if `learn` is STILL a skill file at the time this executes (depends on ordering — if `learn-skill-to-runbook` has already landed, this file won't exist at all) |
| `agent-instructions/guidance/shim.md` (top HTML comment) | update | "recall.md/delegate.md/learn.md, which stay as-is for sessions that still install the recall/learn skills" → drop `recall` from that list (becomes "the learn skill", or the whole clause is moot if `learn` has also retired — re-check ordering) |
| `agent-instructions/guidance/recall.md` | **N/A — see Finding 5** | confirmed carrier-agnostic (`/recall glance`/`/recall deep` phrasing only); no correction needed regardless of retirement or `triggers:` decision |
| `agent-instructions/guidance/{delegate,learn}.md` | N/A | no `recall`-as-skill carrier wording found (`delegate.md` has none; `learn.md`'s own carrier issue is `learn-skill-to-runbook`'s Finding 2, unrelated to `recall`) |

### docs

| file:line | disposition | reason |
|---|---|---|
| `docs/GLOSSARY.md:29-40` (`### skill` entry) | rewrite | "Engram ships two: [`recall`]... and [`learn`]..." → drop `recall`, fold into the runbook list with its two entry points named; same entry `learn`'s own conversion also touches — **whichever change lands second must re-verify this entry's exact wording against what the first change actually shipped**, not assume |
| `docs/GLOSSARY.md:62-69` (`### write-memory (worker runbook)` entry) | update | "reached by basename/wikilink from recall/learn's own text (**they remain skills**)" — wrong once `recall` retires; same shared-editing caution as above |
| `docs/GLOSSARY.md:319-322` (`### recall (skill)` heading + body) | rewrite | heading → "recall (runbook)" or similar (two entry points may need two sub-headings, `recall-glance (runbook)` / `recall-deep (runbook)`, or one entry describing both — a task-level wording decision); body path swapped for the promoted basenames |
| `docs/GLOSSARY.md:324-330` (`### recall modes — glance / deep`) | rewrite | currently describes glance/deep as a single skill's CLI-mode-like dial ("`deep` (default)... `glance` (opt-in)"); needs to describe them as two separately-triggered runbook entry points instead, per design D1's actual encoding (not a mode flag) |
| `docs/GLOSSARY.md:333-344` (`### Step 0 / Step 1 / …`) | update | "Numbered pipeline stages in the recall skill" → "...in the recall procedure (now the `recall-glance`/`recall-deep`/shared core-procedure runbooks)"; content describing the steps themselves is unchanged (design's Non-Goals: behavior doesn't change) |
| `docs/GLOSSARY.md:396,402,416,438` | keep | describe `--lazy-chunks`, dropped-in-recall-v2 history, `triggers:` frontmatter generically — no "the recall skill" carrier claim |
| `docs/architecture/c1-system-context.md:23` (mermaid edge label) | update | `"R2: invokes /recall, /learn (skills)..."` → `/recall` moves out of the `(skills)` parenthetical |
| `docs/architecture/c1-system-context.md:47` (S3 row) | update | "Engram skills (recall, learn)..." → `recall` moves into the vault-runbooks clause |
| `docs/architecture/c1-system-context.md:57` (R2 row) | keep | "Invokes the `/recall` and `/learn` slash commands" — carrier-agnostic, reviewer to confirm |
| `docs/architecture/c2-containers.md:18` (mermaid skills box) | update | `` skills["C1 · Skills (learn / recall)..."] `` → drop `recall` |
| `docs/architecture/c2-containers.md:25` (mermaid edge label) | update | drop `/recall` |
| `docs/architecture/c2-containers.md:41` (C1 row prose) | rewrite | `/recall` (`query` → agent-judged coverage → `amend`/`learn`) moves out of the C1 skills description into the runbook sentence, same restructuring every prior conversion applied |
| `docs/architecture/c2-containers.md:50,102` | update | "Recall reads candidate/member content via `engram show`..." / "The skill layer no longer reads or edits the vault directly: recall reads candidate/member..." — "the skill layer" phrasing needs a carrier-neutral reword once recall is no longer in that layer |
| `docs/architecture/c2-containers.md:111` (sequence diagram participant) | keep or update — reviewer to confirm | `` participant Sk as C1 recall skill `` — mirrors the identical open question left for `learn`'s own diagram-participant label and for write-memory's own conversion |
| `docs/architecture/c2-containers.md:268` | update | "The recall skill drives the loop" — reword to name the runbook, not "skill" |
| `docs/architecture/c3-components.md:234` | keep | "invoked when a skill (recall...)" — generic; reviewer to confirm whether "skill" needs to become "runbook" here specifically for the recall clause |
| `docs/architecture/adr.md:352,374` (ADR-0015 body) | keep | dated 2026-07-04 decision record; unaffected |
| `docs/architecture/adr.md:370-374` (ADR-0015 "Historical note (2026-09)") | update | needs a further clause for `recall`'s retirement, alongside `learn`'s (both changes touch this same paragraph — **whichever lands second must append to what the first already wrote, not overwrite it**) |
| `docs/architecture/adr.md:817` | N/A | "The learn skill's Step 1.5..." — no `recall` mention at this line (already covered under `learn`'s own enumeration) |
| `docs/architecture/adr.md:870` | keep | dated ADR-0025 narrative: "`agent-instructions/skills/recall/SKILL.md` (task 4.1)... were updated accordingly" — past-tense record of a completed historical task, not a current carrier claim |
| `docs/ROADMAP.md:87` (NOW table, #760) | N/A this task; update at close-out (task 6) | tracks #760 itself; same row `learn`'s enumeration flags — shared, updated once at whichever change closes #760 last |
| `docs/ROADMAP.md:97,179,218,237,238,244` | keep | dated historical narrative (recall-time measurements, refuted levers, trap-gate discipline notes) using "the recall skill" generically to describe past measurements; not restated as current |
| `docs/architecture/memory-invariants.md` | N/A | `recall` mentions describe retired invariants/behavior generically, no `SKILL.md` path, no carrier phrase |

### openspec (main specs, not this change's own delta)

| file:line | disposition | reason |
|---|---|---|
| `openspec/specs/recall-payload-cuts/spec.md:90` | N/A this task; carrier fixed via this change's own delta (`specs/recall-payload-cuts/spec.md`), syncs at archive | already identified by design D4 — confirmed present in the enumeration as instructed |
| `openspec/specs/recall-glance-deep-dial/spec.md:5` | **update — see Finding 2; needs a NEW archive-time task, not covered by any existing delta** | Purpose line names the dead `SKILL.md` path; a delta cannot edit Purpose; mirrors write-memory's own Purpose-line archive-time fix precedent |
| `openspec/specs/write-memory-worker/spec.md:9,11,13,19,22,24-25` | N/A this task; carrier fixed via this change's own delta (`specs/write-memory-worker/spec.md`), syncs at archive | "invoked by a parent skill (`recall`, `learn`)" — requirement-body carrier wording; delta already exists and carries the explicit Ordering note (verified accurate in Job 2 of this same task's analysis, covering the archive-ordering conflict with `learn-skill-to-runbook`'s own delta to the same requirement) |
| `openspec/specs/{recall-two-channel-payload,recall-centroid-sampling,recall-matched-note-floor,
  recall-query-timings,recall-runbook-surfacing}/spec.md` | N/A | confirmed by direct read (Finding 2's check): no Purpose-line or requirement-body carrier claim naming `recall` as a skill or citing the dead `SKILL.md` path |
| `openspec/specs/guidance-recall-moments/spec.md` | keep | `/recall glance`/`/recall deep` phrasing only, carrier-agnostic (same reasoning as Finding 5 for `guidance/recall.md`) |
| `openspec/specs/guidance-runbook-follow-frame/spec.md` | N/A | no new requirement expected; recall's runbooks are query-matched/triggered, the existing case |

### code and dev (live, non-eval-history)

| file:line | disposition | reason |
|---|---|---|
| `dev/eval/cumulative/runbook_vs_skill/probe.py::build_cfg_template` / `phase2/probe_phase2.py::build_cfg_template_phase2` | keep — reviewer to confirm, see Finding 3 | shared finding with `learn`'s enumeration |
| `dev/eval/audit/extract.py`, `build_scorecard.py`, `test_extract.py` | keep — reviewer to confirm, see Finding 4 | shared finding with `learn`'s enumeration; recall's `Skill(recall)` detection path (lines ~562 and the recall-call-state tracking logic around 513-652) is the more heavily-used of the two paths in this file |
| `dev/eval/traps/*.py`, `dev/eval/cumulative/{harness,run_recheck,smoke_prune,synthesis_judge,aggregate}.py`, `dev/eval/cumulative/lever_recheck/*`, `dev/eval/cumulative/endorse_cue/probe.py` | keep — flag prominently, see Finding 1 | live standing measurement corpus driving the real `recall` skill as its treatment arm; largest blast radius of any conversion so far |
| `dev/eval/traps/test_wrun.py:23,31-34` | keep — reviewer to confirm | asserts both `recall`/`learn` `SKILL.md` files exist at a deployed destination; will start failing once `recall`'s directory is deleted — same shared risk flagged in `learn`'s own enumeration |
| `internal/cli/query.go`, `query_chunks.go`, `amend_test.go` (code comments: "the recall skill reads...", "the recall skill judges...", "the recall skill marks...") | keep | comments describing the AGENT-SIDE behavior that consumes the binary's output, for future maintainer context — accurate regardless of carrier type (the agent following the runbook still reads/judges/marks the same way); no path reference, no carrier claim needing correction |
| `internal/update/*_test.go`, `internal/cli/{update,update_deps,invariants_u1}_test.go` | keep, no action | use `"recall"` as an arbitrary fixture skill name in a fake filesystem testing the generic sync engine (same class as `learn`'s equivalent tests) — do not assert current shipped-skill status |

## Counts by disposition (draft; re-verify at execution time)

- update: 16
- rewrite: 7
- keep: ~24 (includes reviewer-flagged rows and generic/false-positive-shaped rows)
- N/A (this task; deferred to archive/close-out, already covered by delta, or confirmed carrier-agnostic): 9
- excluded (directory/file-level groups): 5
- delete: 1 (the skill dir itself + its `tests/` subdir, task 5.3)

## Open items for the independent reviewer (in addition to a full fresh re-grep)

1. **Finding 2** — add the missing archive-time task for `recall-glance-deep-dial/spec.md`'s
   Purpose line (design.md's D4 only caught the requirement-body case). Recommend a tasks.md
   addition mirroring write-memory's own "Task added for archive time" (its task 5.5 precedent).
2. **Finding 1** — decide the disposition of the large `dev/eval/traps`/`dev/eval/cumulative`
   recall-driving instrument corpus; this change's design.md Non-Goals are silent on it.
3. **Shared-editing rows** (`docs/GLOSSARY.md`'s `### skill` entry and `write-memory (worker
   runbook)` entry, `docs/architecture/adr.md`'s ADR-0015 historical note, `docs/ROADMAP.md:87`) —
   both this change and `learn-skill-to-runbook` touch the exact same lines. Whichever change
   executes its task 5.2/6.x second must diff against what the first actually shipped, not this
   draft's independently-imagined wording, and not silently overwrite the other's edit.
4. Confirm which ordering (`recall` first vs `learn` first) is actually true at execution time and
   correct every ordering-dependent row accordingly.
5. Confirm `docs/architecture/c2-containers.md:50,102` and `c3-components.md:234`'s exact reword
   needs — this draft's read leans toward "needs a carrier-neutral tweak" but wasn't independently
   re-verified against final promoted-runbook naming.
