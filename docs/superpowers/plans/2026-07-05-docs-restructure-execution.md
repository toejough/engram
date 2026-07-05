# Docs Restructure Execution Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Execute `docs/design/2026-07-04-docs-restructure-suggestions.md` §8 (all seven DPs ruled 2026-07-05): fix the 23 verified defect/gap rows, create the four new artifacts, land the 13 extractions, slim ROADMAP to future-only, delete the 106 historical files (+3 META self-retirements at close) with a verified reference scrub, and add the six diagrams.

**Architecture:** Units are drawn **per target doc**, not per §8 step — ROADMAP, GLOSSARY, and adr.md are each touched by several §8 steps, so each contended file gets exactly one writer unit. Extract-in and migrate-out happen inside that unit. Deletions run last, gated by a mechanical zero-dangling-refs check. THE SPEC = the report; its §4 rows and §3 E-items are gate-verified verbatim-anchored edit specs and are referenced by ID here, not re-quoted — **every executor receives the report alongside its task card**.

**Tech Stack:** git (mv/rm, atomic commits per unit), targ check-full, engram update (skill redeploy), mermaid (diagram sources in report §5), superpowers:writing-skills (Task 8 only).

## Global Constraints

- **The spec:** report `docs/design/2026-07-04-docs-restructure-suggestions.md` at commit `04cbccc3`+ — §4 fix rows (23), §3 extractions (E1–E13), §5 diagram outlines (6), §6 scrub lists (S1/S2), §7 DECIDED rulings (DP1=B ledger, DP2=B indexes, DP3=A, DP4=A adr.md, DP5=delete, DP6=A diff-first, DP7=A no c4-skill).
- **Extract before delete.** No file in the deletion set is removed until its E-item landed and its S1 citers are repointed. Tasks 1–11 all complete before Task 12.
- **One writer per file.** Parallel tasks must have disjoint file sets; every delegated unit ends with `git diff --stat` scope check — files outside the unit's declared set are reverted before commit (vault note 150).
- **Executor reports are claims.** Verify each unit against the actual files (grep the new content in place, old content gone) before its commit (notes 148/162).
- **`targ check-full` green after every unit that touches a `.go` file** (Task 7's E5) and at Tasks 12/13. Binary refresh, if ever needed: `go install ./cmd/engram` — there is no `targ build`.
- **Skill edit discipline:** Task 8 (skills/learn/SKILL.md) uses superpowers:writing-skills RED→GREEN→pressure-test; no other task touches `skills/*/SKILL.md`.
- **Nothing is pushed mid-cycle** (Gate D may reword local commits at Task 13).
- **Numbers rule:** any figure copied into FEATURES/ledger/ADR carries its vintage and evidence pointer (report §4 tally pattern); ROADMAP keeps zero inline figures.
- **Commit messages:** Conventional Commits, `AI-Used: [claude]` trailer, drafts below; Gate D reviews the full cycle log at Task 13.

## Keep-list (everything else tracked `*.md` deletes in Task 12)

LIVE (14 in place): `README.md`, `CLAUDE.md`, `docs/GLOSSARY.md`, `docs/ROADMAP.md`, `docs/architecture/{adr,c1-system-context,c2-containers,c3-components}.md`, `dev/eval/traps/{README,RESULTS}.md` (stay as linked raw data under DP1-B), `.claude/rules/go.md`, `.claude/skills/{engram-go-conventions,commit}.md`, `.claude/commands/commit.md`.
LIVE (3 relocating): `docs/superpowers/specs/2026-06-04-memory-invariants.md` → `docs/architecture/`, `docs/superpowers/specs/2026-06-04-memory-system-rigor.md` → `docs/architecture/`, `docs/superpowers/specs/2026-07-01-engram-recall-subprocess-design.md` → `docs/design/` (in-flight forward design; the report's tree abolishes `docs/superpowers/` but §2 left this LIVE file homeless — docs/design/ is its charter home; ROADMAP's path citations update in Task 4).
SOURCE (10): `skills/*/SKILL.md` (5), `commands/*.md` (4), `guidance/recall.md`.
FIXTURE: `skills/{learn,recall}/tests/baseline-*.md` scenario files (10, NOT `*-results.md`), `dev/eval/testdata/**`, `dev/eval/cumulative/{synthesis_fixtures,lever_recheck/fixture1,contradiction_recheck/cells,skills_auto*}/**`, `dev/eval/{atoms/sandbox-texts,atoms-build/candidate,atoms-build/worker/candidate,guards/candidate,atoms-build/**/fixtures}/**`.
NEW (6, this plan creates): `docs/FEATURES.md`, `docs/README.md`, `dev/eval/LEDGER.md`, `skills/{recall,learn,route}/tests/README.md`.
META (3, deferred to Task 13's §8-9 self-retirement): `docs/design/2026-07-04-docs-restructure-suggestions.md` (the report), `docs/superpowers/plans/2026-07-04-docs-restructure-review.md` (the review plan), `docs/superpowers/plans/2026-07-05-docs-restructure-execution.md` (this plan).
Deletion count check (derived mechanically, Gate-A-verified 2026-07-05): tracked `*.md` (260 at plan commit) − keep (14+3+10+124 = 151) − META (3) = **106** for Task 12. On mismatch STOP and debug in this order: (a) `git ls-files '*.md' | wc -l` changed since plan commit? (b) a Keep-list path missing/renamed? (c) a NEW file created early? (d) then reconcile against report §2 row-by-row.

---

### Task 0: E9/DP6 — research-followups diff (read-only agent pass)

**Files:** Create: `$SCRATCH/e9-missing-items.md` ($SCRATCH = this session's scratchpad).
**Interfaces:** Produces the list of forward-looking items in `docs/design/2026-07-02-research-followups.md` (671 lines) NOT already mirrored in `docs/ROADMAP.md` — each as `| item | followups line | proposed ROADMAP section | one-line summary |`. Task 4 consumes it.

- [ ] **Step 1:** Dispatch one sonnet agent: parse `research-followups.md` into proposal units (each section heading and each top-level bullet = one unit); for each unit not already mirrored in `docs/ROADMAP.md`, emit one table row (empty table = explicitly state "all mirrored"). No repo writes.
- [ ] **Step 2:** Verify (spot-check 3 rows or the all-mirrored claim against both files). No commit.

### Task 1: adr.md — fixes + the four new ADR entries (DP4)

**Files:** Modify: `docs/architecture/adr.md`.
**Interfaces:** Produces ADR-0011..0014 (exact titles below) — FEATURES.md (Task 3) and ROADMAP (Task 4) point at them.

- [ ] **Step 1:** Apply report §4 row adr.md:49 (M8→M4).
- [ ] **Step 2:** Append four entries in the existing ADR format (Status/Context/Decision/Consequences):
  - **ADR-0011 — Controlled-vocab tag nomination over graph traversal.** Decision + rationale from ROADMAP:159-173 and E10's archive (`docs/design/artifacts/2026-07-02-retired-relation-rationales.md` — fold the retired-relation rationale summary in; DP5). Consequences: nomination in candidate_l2s, supersession ride-along, PPR killed.
  - **ADR-0012 — D5′ asymmetric QA participation.** From GLOSSARY:449-457 + the report §3 E4 row's caveat (D5′ rests on n=5 synthetic pairs; round-2 re-validates at corpus scale). Supersedes D5 full exclusion.
  - **ADR-0013 — Vault flock + atomic-rename write safety.** From ROADMAP:44-74 (Track 0): flock only at Run* entry points, helpers lock-free, `.manifest.lock`/`.luhmann.lock`, atomic temp-rename at all edges.
  - **ADR-0014 — Memory-backed tier discount (route).** From ROADMAP:203-211: route by tier, drop one tier for memory-backed units; bounds noted.
- [ ] **Step 3:** E7: fold DESIGN-HISTORY's still-binding constraints — verify ADR-0001/0002/0003 already carry pure-Go/no-CGO/skills+binary (they do); add a one-line "History" pointer to git (`DESIGN-HISTORY.md deleted 2026-07; git log recovers the narrative`) in the header prose; update adr.md:96's ADR-0005 status pointer from `docs/DESIGN-HISTORY.md` to the same git-history form.
- [ ] **Step 4:** Update the header's stale framing (`adr.md:3-6` cites "Phase 6/7 of the memory-system rigor effort" via `../superpowers/specs/` path) → repoint to `memory-system-rigor.md` (its Task-7 relocation target beside adr.md).
- [ ] **Step 5:** Design-fit check (fresh sonnet reviewer: new entries read as native ADRs, no narrative dumping). Verify by grep: `ADR-0011`..`ADR-0014` present, `M8` gone from line 49's row.
- [ ] **Step 6:** Commit: `docs(adr): M4 fix + ADR-0011..0014 — vocab nomination, D5', flock, tier discount`

### Task 2: dev/eval/LEDGER.md — the consolidated results ledger (DP1=B)

**Files:** Create: `dev/eval/LEDGER.md`.
**Interfaces:** Produces claim-row IDs (kebab slugs, e.g. `matched-note-floor`, `glance-deep-dial`) — FEATURES.md and ROADMAP cite `dev/eval/LEDGER.md#<slug>`.

- [ ] **Step 1:** Write the ledger: header states the charter (one unified account of what engram is and is not proven to do; updating it is part of every eval's definition-of-done) + table `| claim | verdict | figure (vintage) | superseded-by | raw data |`. Verdict vocabulary: `proven / refuted / unmeasured / superseded`. Seed rows from `dev/eval/traps/RESULTS.md` first (C3/C4i/C5/C6 capability bars; on any conflict traps/RESULTS.md is source-of-record), then every measured figure in the **pre-Task-4 (old) ROADMAP** — this seeding beyond DP1's literal "traps/RESULTS.md folds in" text is a **flagged extension**: Task 4's cite-ledger-rows-only rule requires rows to exist for every figure ROADMAP currently states. Where ROADMAP carries two vintages of one claim (e.g. 0.19→0.83 diagnostic vs 0.22→0.83 in Done), record the Done-section figure as canonical and the other as a same-row vintage note (matched-note floor 0.22→0.83 (2026-06-28); glance 2.23×/46% (2026-06-29); payload cuts −33.7%/−28% (2026-06-27); payload-prune smoke −40% build_cost (2026-06-30); tier-routing C3/C4i/C6 parity (2026-06-28); qanchor PARKED — no delivery benefit (2026-07-01); PPR killed / L6×TAG +17.3pp (2026-07-02); vocab refit ≈$0.09 (2026-07-03); QA round-1 shipped, Arm V borderline 63% (2026-07-03); C1/C2 warm-op negatives (2026-06-25, marked pre-payload-cut vintage); recall-moments RED 0/5→GREEN 4/5 headless (2026-06-29, model-scoped)). Each row links its raw source (traps/RESULTS.md, or `git log` path for deleted results docs).
- [ ] **Step 2:** Verify: every figure has vintage + raw-data link; verdict column uses only the four values. Design-fit check (haiku reviewer, three checks: no narrative prose in rows; every row's verdict ∈ {proven, refuted, unmeasured, superseded}; every figure carries vintage + raw-data link). Report pass/fail per check.
- [ ] **Step 3:** Commit: `docs(eval): LEDGER.md — consolidated tested-results ledger (DP1)`

### Task 3: docs/FEATURES.md (DP3=A)

**Files:** Create: `docs/FEATURES.md`.
**Interfaces:** Produces feature entry anchors — docs/README.md and README.md point here.

- [ ] **Step 1:** Write per the DP3-A charter: one entry per shipped user-visible capability — 1-2 lines what-it-does + `why:` ADR pointer + `validation:` LEDGER pointer; **no restated numbers, no install/CLI content**. Entries (13 total, from ROADMAP ✅/Shipped/Done + report A6 + E8):
  - matched-note floor
  - payload cuts (--lazy-chunks, --recent-fill)
  - glance/deep recall dial
  - recall-at-decision-moments guidance (@import)
  - memory tier discount (route)
  - vocab lifecycle (term notes, dual-channel tagging, tag nomination, supersession ride-along, autonomous refit)
  - Q&A memory round-1 (learn qa, D5′)
  - concurrency-safe vault writes
  - write-memory worker + capture guards (reversals, lessons audit, escalation provenance)
  - embed-on-write + dual-vector sidecars
  - unified two-channel recall payload
  - ingest auto-sweep with non-persistent-workspace skip
  - **E8's FEATURES half**: a short "validated goals" block carrying memory-system-review's 4-ACHIEVED summary (its still-open half goes to ROADMAP in Task 4) — this content's source deletes in Task 12, so it MUST land here first.
- [ ] **Step 2:** Verify: zero numeric figures (grep `[0-9]+%|×|pp` → only inside pointers/none); every entry has both pointers. Design-fit check (haiku).
- [ ] **Step 3:** Commit: `docs(features): FEATURES.md — implemented-capability reference (DP3)`

### Task 4: ROADMAP rewrite — future-only

**Files:** Modify: `docs/ROADMAP.md` (full rewrite).
**Interfaces:** Consumes Task 0 output, Task 1 ADR IDs, Task 2 ledger slugs, Task 3 anchors.

- [ ] **Step 1:** Rewrite to future-only content: Track A residuals (#665 value gate, C5 recency-apply follow-up, ranking follow-ups if floor proves blunt); Track B open items (payload-prune production build ← NEXT with subprocess-design pointer at its NEW `docs/design/` path; dedupe double ingest sweep; #657 L3a/O1); Track C rounds 2/3 — fold in E2 (the exact pre-registered bands: "PASS ≥8, BORDERLINE 6–7, FAIL <6" + P2′/P3′ definitions), E11 (the P2′ pre-registered branch set from `plans/2026-07-03-qa-memory-exploration.md`), E4 (round-3 scope incl. `engram usage report` if P3′ shows spread; ranking-A/B falsifier sketch); atoms-arc residuals (G6→G5, G2→G3 triggers, G4 parked — keep, they're future triggers); #659 prune-preserve; #668 positive-reinforcement capture kind; E8's still-open exploration list from memory-system-review + Task 0's unmirrored items; parked items with their revisit conditions (qanchor sub-lever, chunk-down-weight). Every prior evidence pointer converts per §6-S1, rule: (a) measured-result pointer → replace with `dev/eval/LEDGER.md#<slug>`; (b) design/history pointer → replace with a git-history citation ("<basename> deleted 2026-07; git log"); (c) all conversions are inline one-liners, never separate evidence sections. Zero inline figures; zero ✅ SHIPPED narratives (one-line "Shipped work: see docs/FEATURES.md; results: dev/eval/LEDGER.md" up top). The subprocess spec's citation updates to its new `docs/design/` path.
- [ ] **Step 2:** Verify: `grep -cE "SHIPPED|✅" docs/ROADMAP.md` → 0 (or only the FEATURES pointer line); every §6-S1 ROADMAP citation resolved (`grep "docs/design/2026-\|docs/superpowers/plans" docs/ROADMAP.md` → only the subprocess spec's new path + git-history citations); bands E2/E11 present verbatim.
- [ ] **Step 3:** Design-fit check (sonnet — future-only charter held, nothing lost that Tasks 1-3 didn't absorb: reviewer diffs old ROADMAP section-by-section against its landing zone).
- [ ] **Step 4:** Commit: `docs(roadmap): slim to future-only — shipped narratives to FEATURES/ADR/LEDGER`

### Task 5: GLOSSARY unit

**Files:** Modify: `docs/GLOSSARY.md`.

- [ ] **Step 1:** Apply report §4 GLOSSARY rows (7): targ rewrite; DELETE recall-mirror/injection-locus/scratch-list/Path-A-B-C entries (:320-359) + replace with the real Step-2 gate entry (fix text in §4 row); DELETE/rewrite subagent+coordinator (:580-586) to current use; extend the recall Step map (:228-232) per fix text; "Three forms" (:304-310); add query-chunks+check (:94-96); add the matched-note floor entry (A6 gap; content from `internal/cli/query.go:132-138,648-654` — noteFloorK=5, why it exists, what it guarantees).
- [ ] **Step 2:** E1: inline the worker-validation correction (the W1/W2/W3 fire-rate figures quoted verbatim in report §3 row E1) into the "atom" entry (:35-41), replacing the pointer to atomic-skills-options.md; E1b: add the pointer-anti-pattern sentence (report §3 row E1b) near "non-triggering description".
- [ ] **Step 3:** E13: apply the six triage rulings (report E13 row) to their entries; add trailing `## Open Questions` section (empty or with anything Joe left unruled — currently none); D5′ entry (:449-457) slims to term definitions + ADR-0012 pointer (its decision narrative moved in Task 1).
- [ ] **Step 4:** Verify: deleted terms gone (`grep -c "recall-mirror\|injection locus\|scratch list\|Path A" docs/GLOSSARY.md` → 0 outside Open-Questions/history mentions); design-fit check (sonnet — glossary stays terms-only, cross-links not restatements).
- [ ] **Step 5:** Commit: `docs(glossary): retire dead-design entries, fix targ/steps/forms, absorb triage (E1/E13)`

### Task 6: README unit

**Files:** Modify: `README.md`.

- [ ] **Step 1:** Apply report §4 README rows (9): targ build → `go install ./cmd/engram`; vocab bootstrap/propose/resituate/show flag fixes (use the §4 fix texts verbatim); learn skill row rewrite; write-memory added to table+pointer+tree; internal/ list + chunk/+cluster/; add `engram learn qa` line to Binary commands.
- [ ] **Step 2:** Design principles section (:163-168) → one line pointing at `docs/architecture/adr.md` (ADR-0001..0003) — a plan-proposed SRP consolidation consistent with report §1's README charter (orientation + CLI reference; principles' authority lives in adr.md), not a report row; add a `## Documentation` pointer to `docs/README.md`.
- [ ] **Step 3:** Verify: `grep -c "targ build" README.md` → 0; `engram vocab bootstrap --seed` documented; write-memory count = 5 skills everywhere. Design-fit check (haiku).
- [ ] **Step 4:** Commit: `docs(readme): CLI reference corrected against the real binary; write-memory restored`

### Task 7: architecture unit — c1/c2/c3 fixes + relocations + E5

**Files:** Modify: `docs/architecture/{c1-system-context,c2-containers,c3-components}.md`, `internal/embed/embedder.go:63`, `docs/superpowers/specs/2026-06-04-memory-system-rigor.md:80` (DESIGN-HISTORY §6 citation → git-history form, before its mv), `docs/superpowers/specs/2026-07-01-engram-recall-subprocess-design.md:4-6` (concurrency-plan citation → shipped-status form, before its mv); git mv the three relocating specs (targets in Keep-list).

- [ ] **Step 1:** Apply §4 architecture rows: c1:123-124 + c3:182 `applyChunkRecency` → `recencyMultiplier`/`defaultRecencyParams`; c2:5-6 + c3:4 drop fixed as-built dates (re-verified-per-edit phrasing); c3:99 G5 → RETIRED; c3:100 → resolved, cite `TestInvariant_C1_ClusteringDeterminism`; c3:101 → resolved, cite `TestUpdater_Run_Local_Idempotent_Property`; c2:42 M4 cell → narrowed partial-migration wording (§4 fix text).
- [ ] **Step 2:** E5: inline into `internal/embed/embedder.go:63` — replace `(see docs/DESIGN-HISTORY.md §2, the 2026-05-24 query spike)` with the rationale itself: `// MiniLM-L6-v2@384 is the shipped bundled model; the 2026-05-24 query spike froze the snake_case sidecar keys as a file format.` Run `targ check-full` → green.
- [ ] **Step 3a:** BEFORE any mv: edit `docs/superpowers/specs/2026-06-04-memory-system-rigor.md:80` (`(see docs/DESIGN-HISTORY.md §6).` → git-history citation) and the subprocess spec's TWO stale citations (§6-S1 names both): lines 4-6 (`docs/superpowers/plans/2026-07-01-concurrency-write-safety.md` → "Track 0 shipped 2026-07-01; plan in git history") and line 11 (`docs/superpowers/specs/2026-06-30-payload-prune-mechanism-design.md` → `dev/eval/LEDGER.md#payload-prune-smoke` slug, or git-history form if the slug differs after Task 2).
- [ ] **Step 3b:** `git mv` the three specs to their Keep-list targets (basenames unchanged).
- [ ] **Step 3c:** Update citers: c2/c3/adr links to `memory-invariants.md`/`memory-system-rigor.md` become same-directory relative (`memory-invariants.md`, no `../superpowers/` prefix).
- [ ] **Step 4:** Verify: `grep -rn "applyChunkRecency\|docs/DESIGN-HISTORY" docs/architecture/ internal/` → 0 hits (plain-text git-history citations like "DESIGN-HISTORY.md deleted 2026-07" carry no `docs/` prefix and are expected); `grep -rn "superpowers/specs" docs/architecture/` → 0. Design-fit check (sonnet).
- [ ] **Step 5:** Commit: `docs(architecture): stale symbols/dates/defect-markers fixed; live specs relocated (E5/E6)`

### Task 8: E3 — learn-skill repoint (writing-skills TDD)

**Files:** Modify: `skills/learn/SKILL.md:50-51`. Test: RED/GREEN evidence in commit message.

- [ ] **Step 1 (RED):** Baseline: `grep -n "qa-memory-proposals" skills/learn/SKILL.md` → hit at :50 (the doc is deleted in Task 12; the pointer would dangle — that IS the failure).
- [ ] **Step 2 (GREEN):** Edit :50-51 to point at ROADMAP's round-2 entry (which now carries the bands + branch set from Task 4): `"QA round-2 validation is due (≥20 pairs captured). Please schedule the round-2 gates recorded in docs/ROADMAP.md (Track C): P2′ attribution fidelity, P3′ distribution, Arm V larger-n."`
- [ ] **Step 3 (pressure):** Fresh haiku agent reads the edited SKILL.md Step 1.5 + Task-4 ROADMAP; question: "What are the QA round-2 validation bands and where are they defined?" PASS = the agent quotes the band values (PASS ≥8 / BORDERLINE 6–7 / FAIL <6) and names ROADMAP Track C as their location. `grep -c "qa-memory-proposals" skills/learn/SKILL.md` → 0.
- [ ] **Step 4:** `engram update` (redeploy skills) → verify `grep -c "qa-memory-proposals" ~/.claude/skills/learn/SKILL.md` → 0.
- [ ] **Step 5:** Commit: `feat(skills): learn Step 1.5 gate pointer follows the bands to ROADMAP (E3)`

### Task 9: DP2=B — tests indexes + E12

**Files:** Create: `skills/recall/tests/README.md`, `skills/learn/tests/README.md`, `skills/route/tests/README.md`. Modify: `skills/recall/tests/baseline-bootstrap-create.md:90` (E12 inline).

- [ ] **Step 1:** E12: read the "Capture format" citation at :90; inline the referenced snippet from `baseline-bootstrap-create-RED-results.md` so the scenario file is self-contained.
- [ ] **Step 2:** Write each README: one row per baseline scenario file — `| baseline | locks which behavior | re-run before editing |`. "Locks which behavior" = the specific skill behavior the scenario exercises, derived by reading the scenario's setup + expected outcome (e.g. "empty vault → skill skips Step 2.5 without erroring"). Route's tests dir keeps no scenario files (its one combined RED-GREEN record deletes in Task 12) → its README is two lines noting the memory-tier-discount behavior is locked by the skill text + git history.
- [ ] **Step 3:** Verify: every `baseline-*.md` scenario file (10) appears in exactly one index. Commit: `docs(skills): per-skill baseline test indexes (DP2); bootstrap scenario self-contained (E12)`

### Task 10: docs/README.md index + CLAUDE.md ripple

**Files:** Create: `docs/README.md`. Modify: `CLAUDE.md`.

- [ ] **Step 1:** Write the index per the report §1 charter: table `| I want to… | go to |` (understand a term → GLOSSARY; see what's planned → ROADMAP; see what's shipped → FEATURES; why is it built this way → architecture/adr.md; how it's structured → architecture/c1-c3; what's proven → dev/eval/LEDGER.md) + the workspace rule copied verbatim from the report §1 design/ charter: "workspace for in-flight, undecided design work only — a doc's conclusions graduate into FEATURES/ROADMAP/ADR and the file is deleted the same cycle it resolves".
- [ ] **Step 2:** CLAUDE.md: docs/ tree comment → `docs/               # Organized by charter — see docs/README.md`; Key Files gains `docs/README.md — documentation index`; Design Principles section → keep the terse agent-facing bullets, add `(authority: docs/architecture/adr.md ADR-0001..0003)`.
- [ ] **Step 3:** Verify links resolve (all six targets exist). Design-fit check (haiku). Commit: `docs(index): docs/README.md — one obvious place; CLAUDE.md ripple`

### Task 11: diagrams (report §5, six outlines)

**Files:** Modify: `docs/architecture/c2-containers.md` (recall pipeline, learn capture kinds, vocab lifecycle, ingest — four flowcharts in their flow sections), `docs/architecture/c1-system-context.md` (please gates swimlane sequence), `docs/architecture/c3-components.md` (QA capture sequence).

- [ ] **Step 1:** For each §5 proposal, render the outline as mermaid in the existing docs' style (classDef conventions already used in c1-c3); place beside the prose it explains. One agent per target file (3 parallel), each given the §5 outlines + the current file.
- [ ] **Step 2:** Verify each renders (mermaid syntax: no bare `graph` errors — spot-parse with a mermaid-aware reviewer agent) and matches the §5 node list. Design-fit check per file (sonnet, one per target: diagram matches the code truth fixed in Task 7).
- [ ] **Step 3:** Commit: `docs(architecture): six feature diagrams — recall, learn, gates, vocab, QA, ingest`

### Task 12: deletions + reference scrub verification

**Files:** Delete 106 tracked `*.md` (Keep-list complement minus META).

- [ ] **Step 1:** Derive the deletion list mechanically: `git ls-files '*.md'` minus Keep-list (incl. relocated paths) minus NEW minus META. Count MUST equal 106 → else STOP and debug per the Keep-list section's checklist.
- [ ] **Step 2:** `git rm` the derived list — **Step 1's derivation is authoritative; this prose is a cross-check**: `docs/DESIGN-HISTORY.md`, `docs/triage.md`, `questions.md`, `docs/validation-harness-restatement.md`, all `docs/design/2026-*` EXCEPT the relocated subprocess spec and the report (META), explicitly including `docs/design/artifacts/2026-07-02-retired-relation-rationales.md` (the DP5 ruling) and the two `docs/design/artifacts/2026-07-01-memory-review-*-survey.md` files, all `docs/superpowers/plans/*` except the two META plans, `docs/superpowers/specs/2026-06-26-*` + the payload-prune spec, `docs/research/*`, the dev/eval HIST-DEAD results docs, the 9 skills-tests results files. Remove now-empty dirs.
- [ ] **Step 3:** Scrub verification over ALL live surfaces: `grep -rn --include='*.md' --include='*.go' -E 'docs/(design/|superpowers|research|DESIGN-HISTORY|triage|validation-harness)' README.md CLAUDE.md docs/ skills/ guidance/ commands/ internal/ cmd/ dev/ .claude/` — then judge each hit: ALLOWED = the three META files' own contents, references to the subprocess spec's new `docs/design/` home, explicit git-history citation lines (e.g. in LEDGER.md raw-data column), and contents of FIXTURE-glob files (frozen eval inputs — snapshotted skill texts and rubrics are NEVER edited; 7 such hits exist, Gate-A-verified). Anything else = STOP, repoint the citer, re-run.
- [ ] **Step 4:** `targ check-full` → green. Commit: `docs(restructure): delete 106 historical docs — extracted, scrubbed, git-preserved`

### Task 13: close-out

**Files:** Delete: `docs/design/2026-07-04-docs-restructure-suggestions.md`, `docs/superpowers/plans/2026-07-04-docs-restructure-review.md`, this plan; remove emptied `docs/superpowers/`.

- [ ] **Step 1:** Gate C over the restructured set (relevance + clarity/cohesion, fresh reviewers): README, docs/README.md, FEATURES, ROADMAP, GLOSSARY, adr.md, c1-c3, LEDGER. Resolve findings.
- [ ] **Step 2:** Gate D over `git log` for the cycle (reword local commits on findings).
- [ ] **Step 3:** §8-9: `git rm` the three META files (their conclusions have graduated); verify `docs/superpowers/` is empty and remove it. `targ check-full` green; `engram update` final deploy; commit: `docs(restructure): retire the review artifacts — conclusions graduated (§8-9)`
- [ ] **Step 4:** Verify end state by COUNTING THE LISTS, not arithmetic: expected = LIVE in-place (14) + relocated (3) + SOURCE (10) + FIXTURE (124) + NEW (6) = **157**; `git ls-files '*.md' | wc -l` must equal it, and re-running Task 12 Step 3's scrub grep must produce only ALLOWED-class hits (git-history citations + FIXTURE-glob contents; the META files are gone so their self-references no longer appear). Additionally, for EACH "deleted 2026-07" basename cited anywhere in the surviving tree: `git log --diff-filter=D --oneline -- <path>` must return non-empty (catches typo'd basenames the form-level scrub can't); and the subprocess spec must exist at its cited `docs/design/` path. Mismatch = STOP.

## Execution order & parallelism

Task 0 → Tasks 1,2,3 (parallel, disjoint files, all READ old ROADMAP) → Task 4 → Tasks 5,6,7,9,10 (parallel, disjoint) → Task 8 (after 4: pointer target must exist) → Task 11 (after 7: same files) → Task 12 → Task 13. Each delegated unit: fresh agent, file-set declared, recall-first, scope-checked diff, verified, committed by the orchestrator.

## Self-Review

- Spec coverage: §8-1→Tasks 1,5,6,7 (§4's 23 rows partitioned: README 9→T6, GLOSSARY 7→T5, architecture c2/c3 6→T7, adr 1→T1); §8-2→T1,3 (the two artifacts §8-2 names; T2's LEDGER.md is DP1's execution shape consumed by T4, not a numbered §8 step); §8-3→E1+E1b/E13(T5), E2/E4/E11(T4), E3(T8), E5/E6(T7), E7/E10(T1), E8(T3+T4 — split per the report's own two-destination instruction), E9(T0→T4), E12(T9); §8-4→T4; §8-5→T7(mv),T5(triage),T10(index); §8-6→T12; §8-7→T11; §8-8→T9 + S2 optional amend batch (deferred — noted for the close-out /learn, not repo work); DP7's note-171 amend already done 2026-07-05; §8-9→T13. ✓
- Placeholder scan: all edit specs are report §4/§3 row references (gate-verified anchors) or inline text; no TBDs. ✓
- Consistency: ADR-0011..0014 IDs, LEDGER slugs, and relocation paths used identically across T1-T4, T7, T10. ✓ TWO deviations named for the record: (1) subprocess-spec relocation to docs/design/ (the report abolished docs/superpowers/ while leaving this LIVE file unaddressed; docs/design/ chosen because the spec is the NEXT build's active input — Track 0 shipped 2026-07-01 unblocked it — so it will graduate-and-delete per that folder's charter when the build ships; docs/architecture/ was considered and rejected: it houses standards/diagrams/invariants, not build designs); (2) LEDGER seed rows extend beyond DP1's literal "traps/RESULTS.md folds in" to every old-ROADMAP figure — required by DP1's own cite-ledger-rows-only clause (Task 4 needs the rows to exist).
