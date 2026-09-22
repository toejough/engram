# Task 4.3 enumeration: retiring `agent-instructions/skills/curate/SKILL.md`

Status: 2026-09-21, performed by the promotion/retirement agent. Every `update`/`rewrite` row below
was performed and verified in the same pass. Per the change's own instructions, this list is built
here but is NOT independently re-verified by a second fresh-context reviewer in this pass — that
verification is a separate step Joe will dispatch.

## Grep run

Case-insensitive, whole repo, word-boundary aware:
`\bcurate\b|\bcurates\b|\bcurating\b|\bcuration\b|/curate|curate skill|offer-curation|curation skill|pending.offer|pending_offer`
over `*.md`, `*.go`, `*.py`, `*.toml`, `*.json`. Plus targeted follow-ups: the literal path string
`agent-instructions/skills/curate` (repo-wide, any file), and `skills/curate/SKILL` under `dev/` to
check for a hardcoded live-skill default in any probe script (the `please_step*_probe` pattern from
the please retirement).

Surfaces covered: `CLAUDE.md`, `README.md`, `.claude/` (n/a — no hits), `agent-instructions/`
(skills, guidance), `docs/` (GLOSSARY, ROADMAP, architecture, research), `openspec/specs/**`,
`openspec/changes/*` (active and archived), `internal/`, `cmd/`, `dev/` (eval harness + non-eval).

## Exclusions (stated)

- `openspec/changes/archive/**`: dated records (e.g. `2026-08-22-serve-vault-api` — curate's original
  authoring task 9.1 — and `2026-09-21-please-skill-to-runbook`, `2026-09-21-runbook-lexical-triggers`,
  `2026-09-19-route-skill-to-runbook`, `2026-08-22-serve-client-declared-identity`).
- `openspec/changes/curate-skill-to-runbook/{proposal,design,tasks}.md` and its own delta specs
  (`specs/vault-offer-curation/spec.md`, `specs/guidance-runbook-follow-frame/spec.md`): this change's
  own self-artifacts.
- `dev/eval/cumulative/runbook_vs_skill/phase2/results/**`: dated measurement records (86 files: raw
  jsonl/log/md results for `curate` and `curate-signal`, all prior tasks in this same change).
- `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/**` (the `curate-conversion-fidelity-report.md`,
  `taskCurate/Curate-S/skills/curate/` frozen copy, `taskCurate/Curate-R/vault/` fixture runbook +
  sidecar): this change's own fixture/report artifacts, not live references.
- `dev/eval/cumulative/runbook_vs_skill/phase2/fixtures/{curate,curate-signal}/**`: this task's own
  self-contained fixture copies (`skill_src: encodings/taskCurate/Curate-S/skills/curate`, never the
  live `agent-instructions/skills/curate/`).
- `docs/research/**`: dated research records (not restated as current).
- `openspec/changes/learn-rate-skill-only/**`, `dev/eval/audit/**`, `dev/eval/traps/reasoning_recall_eval.py`,
  `dev/eval/cumulative/runbook_vs_skill/phase2/fixtures/route/task.json`, `openspec/changes/runbook-trigger-whole-word/**`:
  see "False positives" below — the word matched but the reference is unrelated to the curate skill/runbook.

## False positives (grep hit, not the curate skill/runbook)

| Location | Why N/A |
|---|---|
| `openspec/changes/learn-rate-skill-only/{proposal,design}.md`, `specs/learn/spec.md` | "capture-then-curate" (a quality-posture name) and "closing learn curates collected LESSONS lines" — generic English "curate", unrelated to the retired skill |
| `dev/eval/audit/REPORT-2026-09-02.md:639` | "capture-then-curate with zero new plumbing" — same generic usage |
| `dev/eval/traps/reasoning_recall_eval.py:5` | "the current recall skill (surfaces + curates memory...)" — generic verb describing recall, not the curate skill |
| `dev/eval/cumulative/runbook_vs_skill/phase2/fixtures/route/task.json:35` | vault-note basename `...allow-curation-not-custom-code` — an unrelated note about enforcement-checks config |
| `openspec/changes/runbook-trigger-whole-word/**` (active, unarchived, separate change) | uses `curate` purely as an illustrative example word for whole-word trigger-matching false positives ("accurate"/"curated"); not about the curate skill/runbook's carrier status. Not touched — it is a separate change with its own archival task (4.3) |
| `internal/update/update.go:346-353` | `VaultHasPendingOffers` doc comment names the spec `vault-offer-curation` and the generic word "curation" — describes the concept, not a skill |
| `internal/embed/assets/model/tokenizer.json` | vocabulary token, binary model asset (matched by grep only) |
| `docs/architecture/adr.md` (lines 979, 997, 1026, 1032, 1035, 1041, 1047, 1058) | dated ADR narrative (ADR-0027/0028) using "curate"/"curate gate"/"curate-shaped consumer" as an architectural *concept* (the judged-write pattern), never asserting the carrier is a skill; no factual claim to fix |
| `docs/GLOSSARY.md:438-450` (`trigger`/`trigger hit` entries) | uses `curate` as the worked example of a real, distinctive bare-word trigger — still accurate now that curate IS a runbook trigger |
| `agent-instructions/skills/write-memory/SKILL.md:86-89` | trigger-authoring example (`curate` as a distinctive single word) — still correct after retirement |
| `agent-instructions/skills/{recall,learn}/SKILL.md` | no hits |
| `internal/cli/**` non-test code comments citing `curate-skill-to-runbook D9/D10/D11` | citing this change's own name as a code-comment provenance marker, not "curate skill" wording needing a reword (task 2a.3 already rewrote the actual "curation skill"/"offer-curation skill" user-facing strings) |

## Per-file disposition

Legend: keep / update / rewrite / N/A / delete.

### Root and agent-facing docs

| file:line | disposition | reason |
|---|---|---|
| CLAUDE.md:3 | update (done) | "A fourth skill, `curate`..." folded curate into the runbook sentence alongside please/route; now "Two skills... third skill, write-memory... Three further workflows are no longer skills but vault runbooks: please, route, and curate" |
| CLAUDE.md:26 | update (done) | dir comment "Source for the recall, learn, write-memory, and curate skills" → drop curate |
| CLAUDE.md:38 | update (done) | Key Files glob `{learn,recall,write-memory,curate}/SKILL.md` → drop curate; parenthetical now names all three retired-to-runbook items |
| README.md:10 | update (done) | top description reworded: three skills, three runbooks (please/route/curate) |
| README.md:45-50 (skills table) | rewrite (done) | curate row removed from the 4-row skills table (now 3 rows: recall/learn/write-memory) |
| README.md:52 | update (done) | "See ... for the full skill definitions" dropped the curate SKILL.md link |
| README.md:53-54 | rewrite (done) | paragraph now says "please, route, and curate have no SKILL.md" and adds curate's description + trigger words, pointing at `curate-skill-to-runbook` |
| README.md:110 (two-doors section) | update (done) | "a curation pass (the `curate` skill)" → "the `curate` runbook" |
| README.md:149 (tree comment) | update (done) | "recall, learn, write-memory, and curate skills" → drop curate |

### agent-instructions

| file:line | disposition | reason |
|---|---|---|
| agent-instructions/skills/curate/ (whole dir, 1 file) | delete (this task) | the retirement itself, performed after this enumeration and the D6/D7 gate |
| agent-instructions/guidance/shim.md | update (done in task 3.x — commit 2a0af513) | D12's standing rule is generic (names no skill/runbook); already deployed |
| agent-instructions/guidance/{recall,learn,delegate}.md | keep | no curate references |
| agent-instructions/skills/write-memory/SKILL.md | keep | `curate` used only as a trigger-authoring example (still correct) |

### docs

| file:line | disposition | reason |
|---|---|---|
| docs/GLOSSARY.md `skill` entry (~L29-34) | rewrite (done) | "Engram ships four... `curate`" → three (recall/learn/write-memory); curate joins please/route in the "once shipped as skills, now runbooks" sentence |
| docs/GLOSSARY.md `trigger`/`trigger hit` entries | keep | curate used as the accurate worked example of a bare-word trigger |
| docs/architecture/c1-system-context.md:47 (S3 row) | update (done) | "Engram skills (recall, learn, write-memory, curate)..." → curate moved into the "please, route, and curate procedures are vault runbooks" clause |
| docs/architecture/c1-system-context.md (rest) | N/A | no other curate hits; no dedicated curate flow diagram exists (out of this change's scope, matches proposal's non-goals) |
| docs/architecture/c2-containers.md:18 (mermaid skills box) | update (done) | "C1 · Skills (learn / recall / write-memory / curate)" → drop curate |
| docs/architecture/c2-containers.md:25 (mermaid edge label) | update (done) | "/learn, /recall, /write-memory, /curate" → drop `/curate` |
| docs/architecture/c2-containers.md:41 (C1 row) | rewrite (done) | curate moved out of the C1 skills description into the "no longer skills... type: runbook vault notes" sentence alongside please/route |
| docs/architecture/adr.md | keep (all) | no ADR entry asserts curate is a skill (its origin — archived `serve-vault-api` task 9.1 — was never written into adr.md); all live mentions are the dated "curate gate" architectural-pattern narrative in ADR-0027/0028, unaffected by the carrier change |
| docs/architecture/memory-invariants.md | N/A | no curate hits |
| docs/ROADMAP.md:86 (NOW rank 4, #759) | update (done) | row removed from the NOW table (work is done); ranks 5→4 (#761), 6→5 (#760) renumbered; their "Why here" text updated to say "#759 (closed)" |
| docs/ROADMAP.md:219 (#758 Provenance row) | update (done) | "Unblocks the follow-on conversions #759, #760, #761 (NOW ranks 4-6)" → "#759 (closed), #760, #761 (NOW ranks 4-5)" |
| docs/ROADMAP.md (new row) | update (done) | added a `#759 · curate-skill-to-runbook` Shipped/Provenance row, mirroring the `#758` row's shape, naming the promoted runbook `1049.2026-09-21.curate-review-pending-offers` and its triggers |
| docs/research/**, docs/superpowers/**, docs/design/** | N/A | dated historical records, excluded |

### openspec

| file:line | disposition | reason |
|---|---|---|
| openspec/specs/vault-offer-curation/spec.md | keep (this task); update at archive (task 5.1, later) | main spec has no "curate is a skill" wording to fix now; this change's own delta (`specs/vault-offer-curation/spec.md`) already carries the carrier-naming and per-outcome corrections and syncs into the main spec at archive time, out of this step's scope |
| openspec/specs/guidance-runbook-follow-frame/spec.md | N/A (delta ADDs a new requirement, no main-spec conflict) | this change's `guidance-runbook-follow-frame` delta is additive; nothing to reconcile now |
| openspec/changes/curate-skill-to-runbook/{proposal,design,tasks}.md and its own `specs/**` | N/A | self-artifacts |
| openspec/changes/archive/2026-08-22-serve-vault-api/tasks.md:47 | keep (excluded) | archived, dated record of curate's original authoring (task 9.1) |
| openspec/changes/archive/2026-09-21-{please-skill-to-runbook,runbook-lexical-triggers}/**, archive/2026-09-19-route-skill-to-runbook/**, archive/2026-08-22-serve-client-declared-identity/** | keep (excluded) | archived, dated records that mention curate only as a sibling/next-conversion or an example word |
| openspec/changes/runbook-trigger-whole-word/** | keep (not touched) | separate, active, unarchived change; uses `curate` only as an illustrative example word; its own task 4.3 is a different archival step, not this change's concern |
| openspec/changes/learn-rate-skill-only/** | keep (false positive) | "capture-then-curate"/"closing learn curates..." are generic usage, not the curate skill |
| openspec/specs/vault-serve-api/spec.md, archive/2026-08-22-serve-client-declared-identity/specs/vault-serve-api/spec.md | N/A | mentions "pending offers" for the write path only, no carrier-naming to fix |

### code and dev

| file:line | disposition | reason |
|---|---|---|
| internal/cli/{query,offer,amend,learn,update,serve,targets}.go and tests | keep | task 2a.3 (already done, prior commit) reworded all user-facing "curation skill"/"offer-curation skill" text to "curation runbook"; remaining `curate` mentions are code comments citing this change's own name (`curate-skill-to-runbook D9/D10/D11`) as provenance, not carrier wording |
| internal/update/update.go:346-353 | keep (false positive) | doc comment names the spec `vault-offer-curation` and the generic word "curation", not a skill |
| dev/eval/cumulative/runbook_vs_skill/phase2/probe_phase2.py | keep | `_CURATE_MUTATING_BASH_RE` etc. are the harness's own task-name constants, not a live-skill-path default |
| dev/eval/cumulative/runbook_vs_skill/phase2/test_probe_phase2.py:4834-4838 (`test_curate_skill_src_is_byte_identical_to_the_live_skill`) | keep (self-adapting) | already `pytest.skip("live curate skill already retired")` when the live path is gone — no edit needed, verified it skips cleanly post-deletion |
| dev/eval/cumulative/runbook_vs_skill/phase2/fixtures/{curate,curate-signal}/task.json | keep | `skill_src` points at the frozen fixture copy (`encodings/taskCurate/Curate-S/skills/curate`), never the live path — confirmed by grep, no `please_step*_probe`-style hardcoded live default exists for curate |
| No `curate_step*_probe` analog script exists | N/A | please's dedicated step-probe scripts (`please_step7_probe`, `please_step3_probe`) have no curate equivalent; nothing to repoint |

## Counts by disposition (table rows above; grouped rows count once)

- update: 13
- rewrite: 4
- keep: 15 (includes false-positive rows)
- N/A: 8
- excluded: 6 (directory-level)
- delete: 1 (the skill dir itself)

## What `engram update` sync removes

`agent-instructions/skills/curate/` is the only sync source deleted by this change. After
`git rm -r agent-instructions/skills/curate/` and `engram update`, the deployed copies at
`~/.claude/skills/curate/` and `~/.pi/agent/skills/curate/` are removed (sync-as-removal), verified
below in tasks.md/the closing report.
