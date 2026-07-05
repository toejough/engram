# Docs Restructure — Review Findings & Concrete Suggestions

**Date:** 2026-07-04 · **Status:** awaiting Joe's decisions (§7), then a follow-up execution cycle (§8) · **Read first:** §7's decision table, then §1's target tree.

**Method:** six review angles — **A1** liveness, **A2** code-correctness, **A3** skills-correctness, **A4** SRP, **A5** diagrams, **A6** completeness — ran over a frozen ground truth (258 tracked `*.md`; a 601-entry inbound-reference graph, probe-verified). Every correctness finding was adversarially verified against code by an independent agent (28/28 CONFIRMED, 0 refuted; §4 owns the tally) and every HIST-OBLIGATED disposition was verbatim-verified. Label key used throughout: **DP1–DP7** = the seven decision points (§7) · **E1–E13** = extraction items (§3) · **S1/S2** = reference-scrub lists (§6). A 5-row random sample of HIST-DEAD calls flipped 2, so the sample was superseded by a **full** live-citation cross-check over all 103 initially-HIST-DEAD rows: 30 carried live-surface refs — 3 of the 30 now sit in HIST-OBLIGATED, the other 27 stay HIST-DEAD with §6 scrub entries (14 repo-cited → S1, 13 vault/auto-memory-only → S2; S1 additionally carries 2 targets the pattern-based check missed because ROADMAP cites them by bare filename or data-directory — realvault-glance-cost, revalidation-data). The single authoritative flip-provenance account sits under §2's table. Numbers in this report are copied from the verifying agents' evidence pointers, vintage 2026-07-04.

---

## 1. Summary + target doc tree

The corpus is 258 tracked markdown files. Only ~28 are genuinely live; ~106 are historical records of completed work (plans, dated design/results docs, superseded research) and ~124 are test fixtures or deployable skill sources. The live set has real correctness drift (21 verified defect rows per §4's tally — 16 misleads-design, 5 minor-drift — including a documented `targ build` target that doesn't exist, four CLI blocks with wrong flags, a glossary describing a retired learn design, and three stale defect markers contradicted by existing tests) and real SRP scatter (decisions live in four places; ROADMAP is ~⅔ shipped-work changelog; features have no home).

**Target tree** (charters from the SRP angle; every doc has exactly one responsibility). The tree shows the **recommended end-state**: where it embeds one of §7's open decisions it shows the recommended option — DP1 (**decided 2026-07-05**: one consolidated results ledger), DP2 (per-skill `tests/README.md` indexes get added under `skills/`), DP3 (README keeps the CLI reference), DP4 (decisions land in adr.md), DP7 (diagrams stay in `docs/architecture/`). Picking the other option on any DP adjusts the tree as described in §7; nothing here pre-empts those choices.

```
README.md            [orientation: what engram is, install, quickstart, CLI reference — points into docs/]
CLAUDE.md            [agent instructions only — terse rules, pointers to authority docs]
docs/
  README.md          [NEW — the index: names the one doc per responsibility + the workspace lifecycle rule]
  GLOSSARY.md        [terms only — one definition per term; cross-links adr.md for decision-origin terms;
                      absorbs triage.md's items as a trailing "Open Questions" section]
  ROADMAP.md         [future-only — open tracks, parked items with triggers; NO shipped narrative,
                      NO inline measured numbers (cite the eval ledger)]
  FEATURES.md        [NEW — one entry per shipped user-visible capability: what it does, 1-2 lines,
                      pointer to its ADR entry (why) and eval-ledger entry (validation); never restates either]
  architecture/
    adr.md           [the ONE standards/decisions doc — gains entries for: vocab tag-nomination,
                      D5′ QA participation, Track-0 flock/write-safety, tier-routing, plus
                      DESIGN-HISTORY's still-binding constraints]
    c1-system-context.md / c2-containers.md / c3-components.md   [C4 diagrams, kept, corrected per §4]
    memory-invariants.md      [relocated from docs/superpowers/specs/ — declared-live invariant catalog]
    memory-system-rigor.md    [relocated alongside it]
  design/            [workspace: in-flight undecided work ONLY; conclusions graduate to
                      FEATURES/ROADMAP/ADR and the file is deleted the same cycle — steady-state near-empty]
  research/          [workspace: same graduate-then-delete rule]
  images/            [assets]
dev/eval/            [DP1 DECIDED: one consolidated ledger of tested results (claim rows: verdict,
                      figure, vintage, supersession, raw-data link); per-harness files become linked
                      raw data, not narratives]
skills/ commands/ guidance/   [deployable behavior sources — unchanged, except DP2-B adds skills/<skill>/tests/README.md indexes]
```

"One obvious place to go": `docs/README.md` is the index; the answer to "where do I look/update X?" is one hop from it. `docs/superpowers/` disappears entirely (plans/specs are historical or relocated); `DESIGN-HISTORY.md`, `triage.md`, `questions.md`, `validation-harness-restatement.md` disappear after extraction.

## 2. Per-file disposition table

Disposition vocabulary:

- **LIVE** — keep, maintain
- **SOURCE** — deployable behavior source; out of restructure scope
- **HIST-OBLIGATED** — delete only after its §3 extraction lands (includes docs whose *content* is live but whose *container* dissolves, like triage.md)
- **HIST-DEAD** — historical, no live obligations; commit-then-delete now
- **FIXTURE** — test data; keep with its harness

Dispositions marked **Δ** were adjusted during verification; the single authoritative provenance account for all six is the note under this table.

| path | disposition | rationale |
|---|---|---|
| `.claude/commands/commit.md` | LIVE | active slash command (2026-02-27, 59 lines) |
| `.claude/rules/go.md` | LIVE | active Go/nilaway/gomega rules, loaded every session |
| `.claude/skills/commit.md` | LIVE | active commit skill |
| `.claude/skills/engram-go-conventions.md` | LIVE | deployed convention skill, actively used (2026-04-16, 280 lines) |
| `CLAUDE.md` | LIVE | repo agent instructions, maintained same-day |
| `README.md` | LIVE | top-level project doc, maintained same-day |
| `dev/eval/traps/README.md` | LIVE | companion readme to the live capability ledger |
| `dev/eval/traps/RESULTS.md` | LIVE | the still-cited C3/C4i/C5/C6 capability-verification ledger — the current regression bar |
| `docs/GLOSSARY.md` | LIVE | the one glossary (603 lines, maintained same-day) |
| `docs/ROADMAP.md` | LIVE | the live tracks/status doc — slims to future-only per §1 |
| `docs/architecture/adr.md` | LIVE | the one decisions doc — gains the §3 extracted entries |
| `docs/architecture/c1-system-context.md` | LIVE | L1 C4 + four key-flow sequence diagrams, maintained same-day |
| `docs/architecture/c2-containers.md` | LIVE | L2 container diagram, maintained |
| `docs/architecture/c3-components.md` | LIVE | L3 component diagram, maintained |
| `docs/superpowers/specs/2026-06-04-memory-invariants.md` | LIVE | declared-live carve-out (DESIGN-HISTORY §6); linked from adr/c2/c3 → **relocate** to docs/architecture/ |
| `docs/superpowers/specs/2026-06-04-memory-system-rigor.md` | LIVE | same carve-out; linked from adr.md:6 → **relocate** to docs/architecture/ |
| `docs/superpowers/specs/2026-07-01-engram-recall-subprocess-design.md` | LIVE | ROADMAP:213 "production build ← NEXT" — the active forward design (future-work exempt) |
| `docs/triage.md` | HIST-OBLIGATED **Δ** | active tracker whose content folds into GLOSSARY (E13), then the file deletes |
| `commands/{learn,please,recall,route}.md` | SOURCE | OpenCode slash-command sources (4 files) |
| `guidance/recall.md` | SOURCE | @import-deployed guidance source |
| `skills/{recall,learn,write-memory,please,route}/SKILL.md` | SOURCE | deployable behavior sources (5 files) |
| `docs/DESIGN-HISTORY.md` | HIST-OBLIGATED | settled extract-then-delete; obligations E5–E7 in §3 |
| `docs/design/2026-07-01-memory-system-review.md` | HIST-OBLIGATED **Δ** | sampler flip: ROADMAP:17 cites it as the live home of the goals scorecard + ranked exploration list (E8) |
| `docs/design/2026-07-02-research-followups.md` | HIST-OBLIGATED **Δ** | A1 flagged UNVERIFIED risk: 671-line consolidated followups report may hold parked directions unmirrored in ROADMAP — pre-delete diff required (E9) |
| `docs/design/2026-07-03-qa-memory-proposals.md` | HIST-OBLIGATED | learn skill points at it BY PATH at runtime; pre-registered round-2 bands (E2–E4) |
| `docs/design/2026-07-04-atomic-skills-options.md` | HIST-OBLIGATED | GLOSSARY:40 defers to its Postscript for the instrument-invalid correction (E1) |
| `docs/design/artifacts/2026-07-02-retired-relation-rationales.md` | HIST-OBLIGATED **Δ** | exception-argument row (§7 DP5): ROADMAP:171 names it as the durable archive itself (E10) |
| `docs/superpowers/plans/2026-07-03-qa-memory-exploration.md` | HIST-OBLIGATED **Δ** | full-cross-check flip: vault note 165 directs round-2 validation to its pre-registered P2′ branch set (E11) |
| `skills/recall/tests/baseline-bootstrap-create-RED-results.md` | HIST-OBLIGATED **Δ** | sampler flip: cited from the live fixture `baseline-bootstrap-create.md:90` (E12) |
| `dev/eval/atoms-build/results-2026-07-04.md` | HIST-DEAD | CORRECTION already extracted into ROADMAP:184-187 (scrub S1) |
| `dev/eval/atoms-build/worker/results-2026-07-04.md` | HIST-DEAD | worker-round results, folded into ROADMAP status |
| `dev/eval/atoms/smoke-results-2026-07-04.md` | HIST-DEAD | superseded by atoms-build results |
| `dev/eval/cumulative/EXPERIMENT-LOG.md` | HIST-DEAD | stopped updating 2026-06-26 despite continued July evals — superseded-in-practice by dev/eval/traps (scrub S2) |
| `dev/eval/cumulative/OPUS-TRAP-CATALOG.md` | HIST-DEAD | superseded by dev/eval/traps |
| `dev/eval/cumulative/README.md` | HIST-DEAD | describes the superseded cumulative harness |
| `dev/eval/cumulative/contradiction_recheck/README.md` | HIST-DEAD | superseded harness component |
| `dev/eval/cumulative/lever_recheck/{PLAN,README,RESULTS}.md` | HIST-DEAD | superseded harness components (3 files; note-85 provenance pointer → scrub S2) |
| `dev/eval/guards/results-2026-07-04.md` | HIST-DEAD | conclusions folded into ROADMAP:189-195 |
| `dev/eval/qa/p1_delivery_followup.md` | HIST-DEAD | round-1 followup probe record |
| `dev/eval/qa/results-2026-07-03.md` | HIST-DEAD | round-1 results, folded into ROADMAP |
| `dev/eval/vocab/trigger_replay_results.md` | HIST-DEAD | one round's replay results (note-163 provenance pointer → scrub S2) |
| `docs/design/2026-06-23-{compounding-eval,cross-cluster-linking,synthesis-layer}.md` | HIST-DEAD | superseded designs (3 files; mechanism ⛔ KILLED per ROADMAP:164) |
| `docs/design/2026-06-24-*.md` | HIST-DEAD | cost-investigation round docs (5 files), decided/extracted; note-79/82 provenance → scrub S2 |
| `docs/design/2026-06-25-*.md` | HIST-DEAD | cost/measurement round docs (3 files); note-91/95 provenance → scrub S2 |
| `docs/design/2026-06-27-*.md`, `2026-06-27-recall-trigger-data/README.md` | HIST-DEAD | trigger analysis + method docs (3 files); verdict extracted (ROADMAP:259 → scrub S1) |
| `docs/design/2026-06-28-*.md` + data READMEs | HIST-DEAD | failure-mining/crystallization/probe round (8 files); findings extracted (ROADMAP:24/84/108/125/205/322 → scrub S1) |
| `docs/design/2026-06-29-*.md` + revalidation data | HIST-DEAD | depth-dial/#661/#662 round (4 files); shipped (scrub S1 for ROADMAP:105/239/246) |
| `docs/design/2026-07-01-question-anchored-distillation.md` | HIST-DEAD | eval'd + ⛔ PARKED (ROADMAP:130,140 → scrub S1; note-153 provenance → scrub S2) |
| `docs/design/2026-07-02-link-value-exploration.md` | HIST-DEAD | fully extracted (ROADMAP:164 → scrub S1; note-158 provenance → scrub S2) |
| `docs/design/2026-07-03-vocab-{lifecycle-proposals,notes-build-results}.md` | HIST-DEAD | decided + shipped (2 files; ROADMAP:172 → scrub S1; note-163 → scrub S2) |
| `docs/design/2026-07-04-atomic-skills-research.md` | HIST-DEAD | superseded by the decided build (qa-note provenance → scrub S2) |
| `docs/design/2026-07-04-lesson-capture-blindspot-options.md` | HIST-DEAD | decision + both pre-registered triggers fully restated in ROADMAP:189-195 (scrub S1; note-169 → scrub S2) |
| `docs/design/artifacts/2026-07-01-memory-review-{systems,techniques}-survey.md` | HIST-DEAD | research artifacts of the memory-system review (2 files) |
| `docs/research/2026-06-22-emergent-synthesis-case.md` | HIST-DEAD | superseded by shipped synthesis-layer verdicts; note-68 "See …" pointer → scrub S2 |
| `docs/research/2026-06-23-reasoning-modes.md` | HIST-DEAD | standalone note, no live citation |
| `docs/superpowers/plans/2026-06-*.md` | HIST-DEAD | executed/shipped build plans, 2026-06-20 → 06-30 (21 files; note-vs-chunk-ranking cited at ROADMAP:42 → scrub S1) |
| `docs/superpowers/plans/2026-07-0{1,2,3}-*.md` (except qa-memory-exploration) | HIST-DEAD | executed/shipped build plans (8 files; concurrency plan cited at ROADMAP:73 + subprocess-spec:6 → scrub S1) |
| `docs/superpowers/plans/2026-07-04-*.md` | HIST-DEAD | this cycle's + today's executed plans (6 files incl. this review's own plan, which dies per its own rule when this cycle closes) |
| `docs/superpowers/specs/2026-06-26-*.md` | HIST-DEAD | shipped designs (2 files) |
| `docs/superpowers/specs/2026-06-30-payload-prune-mechanism-design.md` | HIST-DEAD | smoke result fully restated in ROADMAP:219-224 and cited by the LIVE subprocess spec → scrub S1 |
| `docs/validation-harness-restatement.md` | HIST-DEAD | self-labeled working doc, superseded by the built harnesses; zero inbound refs |
| `questions.md` | HIST-DEAD | 2026-06-24 scratch; all five questions addressed by shipped work; zero inbound refs |
| `skills/{learn,recall,route}/tests/*-results.md`, `memory-discount-RED-GREEN.md` | HIST-DEAD | recorded RED/GREEN outcomes of already-shipped skill edits (8 files; the 9th is E12 above) |
| `skills/{learn,recall}/tests/baseline-*.md` (scenario files) | FIXTURE | reusable RED/GREEN scenario inputs for writing-skills TDD (10 files; see §7 DP2) |
| `dev/eval/testdata/**` | FIXTURE | 63 files — synthetic vaults/test data |
| `dev/eval/cumulative/{synthesis_fixtures,lever_recheck/fixture1,contradiction_recheck/cells,skills_auto*}/**` | FIXTURE | 33 files — harness fixtures |
| `dev/eval/{atoms/sandbox-texts,atoms-build/candidate,atoms-build/worker/candidate,guards/candidate,atoms-build/**/fixtures}/**` | FIXTURE | 18 files — candidate skill texts + vault seeds |

**Completeness check:** 144 individual rows + 114 files in FIXTURE/grouped globs = 258 = `git ls-files '*.md' | wc -l` ✓. (triage.md's reframe moves it LIVE→HIST-OBLIGATED without changing the count: 17 LIVE, 10 SOURCE, 9 HIST-OBLIGATED, 98 HIST-DEAD, 124 FIXTURE files.)

**Δ provenance (the one account):**

- `memory-system-review` — sampler flip (ROADMAP:17 citation)
- `baseline-bootstrap-create-RED-results` — sampler flip; its citer is a fixture file outside the cross-check's live-surface list, which is why only the sampler caught it
- `qa-memory-exploration` — full cross-check (vault note 165)
- `retired-relation-rationales` — orchestrator synthesis on the exception argument, corroborated by the cross-check's ROADMAP:171 hit
- `research-followups` — A1's residual-risk note
- `triage.md` — tracker-dissolve reframe at Gate B

## 3. Extraction list (must land BEFORE the corresponding deletion)

"From" is the doc being deleted, with two exceptions: E3 is a **citer-repoint** (its "from" is the live file to edit), and E12's **edit target** is the live fixture that cites the deletable results file.

| # | from (verbatim source) | what | to |
|---|---|---|---|
| E1 | `atomic-skills-options.md:176-182` — "The widely-cited interim \"0/27 mid-procedure dereference\" figure is instrument-invalid… The worker form was validated… W1 3/3, W2 3/3, W3 2/3… boundary violations 0, non-fire 0/6." (live citers, verifier-confirmed: GLOSSARY:40 and ROADMAP:188) | the deployed-measurement correction both citers defer to by pointer | inline into GLOSSARY "atom" entry (or ROADMAP atoms-arc block) |
| E1b | same doc — the O-B confabulation finding: pointer-style "apply X verbatim" references to out-of-context text are a measured anti-pattern *(verifier-found addition)* | anti-pattern lesson | GLOSSARY entry near "non-triggering description", or a vault note |
| E2 | `qa-memory-proposals.md:65` — "pre-registered bands: PASS ≥8, BORDERLINE 6–7, FAIL <6" (+ P2′/P3′ definitions, lines 130-131) | the round-2 gate's exact pre-registered bands (ROADMAP:293-298 has the ≥20-pairs/≥80% frame but NOT these bands) | ROADMAP round-2/round-3 entry |
| E3 | `skills/learn/SKILL.md:50-51` — "Please schedule `docs/design/2026-07-03-qa-memory-proposals.md` round-2 gates…" | runtime path pointer | repoint at ROADMAP once E2 lands |
| E4 | same doc — round-3 scope incl. `engram usage report` (if P3′ shows spread); the ranking-A/B falsifier sketch; the D5′ n=5 caveat *(verifier-found additions)* | pre-registered round-3 licensing details no live surface carries | ROADMAP round-3 bullet; D5′ caveat → the D5′ ADR entry (§1) |
| E5 | `DESIGN-HISTORY.md §2` ← cited by `internal/embed/embedder.go:63` | one-sentence rationale (MiniLM-L6-v2@384 bundled; sidecar keys a frozen format) | inline into the code comment |
| E6 | `DESIGN-HISTORY.md §6:165-172` — the two-specs-stay-live carve-out; also cited by `memory-system-rigor.md:80` | carve-out honored by relocating both specs to docs/architecture/; fix the rigor doc's §6 citation | docs/architecture/ + one-line edit |
| E7 | `DESIGN-HISTORY.md §§1-10` — still-binding constraints (pure-Go/no-CGO, append-only, the D1-D7 supersession chain); adr.md:96's ADR-0005 status line also points into DESIGN-HISTORY and gets updated when this lands | decision rationale | adr.md new/expanded entries (§7 DP4, Option A) |
| E8 | `memory-system-review.md` — goals scorecard (4 ACHIEVED/3 PARTIAL/3 REFUTED/3 UNMEASURED) + ranked exploration list, cited at ROADMAP:17 | still-open explorations → ROADMAP; achieved-goals summary → FEATURES | then delete + repoint ROADMAP:17 |
| E9 | `research-followups.md` (671 lines) | **UNVERIFIED HYPOTHESIS** (A1, budget-bounded): may hold parked directions unmirrored in ROADMAP — run an item-by-item diff vs ROADMAP parked/future sections; mirror what's missing | then delete |
| E10 | `artifacts/2026-07-02-retired-relation-rationales.md`, cited at ROADMAP:171 as "archived in…" | the retired-relation rationale | the new vocab tag-nomination ADR entry; repoint ROADMAP:171 (see §7 DP5) |
| E11 | `plans/2026-07-03-qa-memory-exploration.md` — the pre-registered P2′ attribution-fidelity branch set (vault note 165 directs round-2 validation to it) | the branch set | ROADMAP round-2 entry alongside E2 |
| E12 | `baseline-bootstrap-create-RED-results.md` ← cited by fixture `baseline-bootstrap-create.md:90` ("Capture format") | inline the cited capture-format snippet into the scenario file; the results file then deletes with its siblings | works under either §7 DP2 option |
| E13 | `docs/triage.md` items 4/9/11/13/14/15 | the six open glossary-triage rulings, resolved as: **4** openers vs type names — keep both, add one clarifying sentence to the fact/feedback entries; **9** session vs transcript — adopt the doc's own canonical (session = interaction, transcript = serialized record), one prose sweep; **11** skill/slash-command/command — adopt the doc's own canonical, one clarifying sentence near README:36; **13** Path A/B/C — **moot**: the entries describe a retired learn design and delete per §4 (reconciles A6's keep-as-is with A3's verified finding); **14** slug — already canonical (GLOSSARY:141), close; **15** engram project-vs-binary — qualifier-only-where-ambiguous usage note on the GLOSSARY "engram" entry | GLOSSARY (entries + a trailing "Open Questions" section for anything Joe leaves unruled); then delete triage.md |

## 4. Correctness fixes

Severity legend: **misleads-design** — a reader acting on it would build against dead paths or wrong behavior · **minor-drift** — wrong but low consequence · **gap** — something missing entirely (an A6 completeness finding), not an error. Defect labels appearing in the architecture rows (M4, M8, G5, C1, L3-1, U1) are invariant IDs from `docs/superpowers/specs/2026-06-04-memory-invariants.md`, which the architecture docs' ⚠ annotations cite.

**Tally (this section owns these numbers, arithmetic chain):** 28 findings from the three angles, every one independently verified against code by a refutation-charged agent — 28 CONFIRMED, 0 refuted. Minus 5 cross-angle duplicate copies (targ-build ×3→1, GLOSSARY-targ ×2→1, write-memory omission ×3→1) = 23 unique findings. Minus 2 pair-merges — two findings each spanning the same symbol in two files, rendered as one row (`applyChunkRecency` in c1+c3; the as-built dates in c2+c3) = **21 defect rows (16 misleads-design + 5 minor-drift)**. Plus 2 gap rows from the completeness angle = **23 table rows**.

**README.md**

| loc | defect | severity | fix |
|---|---|---|---|
| :158 | documents `targ build` — no such target exists (`targ` lists check/test/lint families only; CLAUDE.md:54 already states this) | misleads-design | `go install ./cmd/engram` — install the binary (no targ build target) |
| :89 | `vocab bootstrap [--dry-run]` — no `--dry-run`; real flags are required `--seed <yaml>` + `--floor` (default 0.35) (`internal/cli/vocab_commands.go:48-50`) | misleads-design | document `--seed`/`--floor` |
| :90 | `vocab propose <term> --why <r>` — no positional arg, no `--why`; real flags `--term` + `--description`, both required (`vocab_commands.go:80-82`) | misleads-design | document `--term`/`--description` |
| :81 | `resituate --note <ref> [--dry-run]` — no `--dry-run`; omits required `--situation` (`resituate.go:21-22`) | misleads-design | `resituate --note <ref> --situation <text>` |
| :85 | `show <ref> [--ref <ref>...]` — no `--ref` flag; one required positional; output also includes outbound wikilink targets (`show.go:17,80-81`) | misleads-design | correct syntax + output description |
| :43 | learn row claims a "recall-mirror test" gate — no such gate exists in `skills/learn/SKILL.md` (Step 2 = three explicit kinds → write-memory handoff) | misleads-design | rewrite row to the sweep + three-kinds + vocab-liveness reality |
| :47, :152, Skills table | write-memory omitted from the pointer list, the tree line, and the table — while README:10 itself describes it (5 skills, not 4; GLOSSARY:28-33 and CLAUDE.md agree) | misleads-design | add write-memory to all three |
| :142-151 | `internal/` package list missing `chunk/` and `cluster/` (10 packages exist; CLAUDE.md's tree is correct) | minor-drift | add both rows |
| :74-93 | **gap (A6):** `engram learn qa` absent from the otherwise-exhaustive Binary commands block (`internal/cli/qa.go`, shipped 2026-07-03) | gap | add the `learn qa` line |

**docs/GLOSSARY.md**

| loc | defect | severity | fix |
|---|---|---|---|
| :566-569 | targ entry: "wrapping `go test`/`go vet`/`go build`… Always invoke `targ build`…" — same nonexistent target | misleads-design | rewrite; binary install = `go install ./cmd/engram` |
| :320-359 | "recall-mirror test", "injection locus", "scratch list", "Path A/B/C" entries describe a RETIRED learn design with zero correspondence in the shipped skill | misleads-design | delete the four entries; add one entry describing the real Step-2 gate |
| :580-586 | "subagent"/"coordinator" entries describe the retired parallel-writer architecture (recall's own red-flags table: "Gone — Step 2.5 crystallizes inline") | misleads-design | delete or rewrite to current use (please gate reviewers, route dispatch) |
| :228-232 | recall Step map omits Step 0.5 (sweep), Step 2.7 (activate), Step 4 (persist) — all load-bearing | misleads-design | extend to the full stage list incl. glance markers |
| :304-310 | "`engram learn` … Two forms" — three (qa documented 100 lines later in the same file) | minor-drift | "Three forms" |
| :94-96 | binary subcommand list omits `query-chunks` and `check` | minor-drift | add both |
| — | **gap (A6):** no "matched-note floor" entry (the headline 2026-06-28 ranking fix; `query.go:132-138,648-654,1080`) — only a passing mention at :188 | gap | add the entry + one clause in README's query line |

**docs/architecture/**

| loc | defect | severity | fix |
|---|---|---|---|
| adr.md:49 | defect labeled "M8" — the ADR's own Status line (:39), c2:41-42, c3:97/104 all call it M4; M8 is Luhmann-id uniqueness | misleads-design | M8 → M4 |
| c2:42 | "M4: swap silently empties recall (no guard)" — a guard EXISTS for full mismatch (`query.go:90-92` raises `errQueryNoEmbeddings`); only partial migration is silent (c3 states this correctly) | misleads-design | narrow to the partial-migration case |
| c3:99 | G5 listed as an active defect — the invariants doc marks G5 "[RETIRED — episode kind removed…]" | misleads-design | drop/mark RETIRED |
| c3:100 | "C1/L3-1 determinism untested" — `TestInvariant_C1_ClusteringDeterminism` + `TestKMeans_DeterministicAcrossRuns` exist, predating the doc's last commit | misleads-design | mark resolved, cite the tests |
| c3:101 | "U1 idempotence uncaptured" — `TestUpdater_Run_Local_Idempotent_Property` exists (2026-06-12) | misleads-design | mark resolved, cite the test |
| c1:123-124, c3:182 | cite `applyChunkRecency` — no such symbol; real mechanism `recencyMultiplier`/`defaultRecencyParams` (`query.go:1296`, `recency.go:51`) | minor-drift | name the real symbols |
| c2:5-6, c3:4 | "as-built … 2026-06-04" — both files' content and git history run through 2026-07-03/04 | minor-drift | drop the fixed date; state defects re-verified per edit |

No unverified-hypothesis findings remain in this section (the only unverified item in the report is E9, labeled there).

## 5. Mermaid diagram proposals

The restructure's diagram half of the ask: §4 already fixed what the existing diagrams get wrong; this section covers what's *missing* — the shipped features with no diagram at all.

Existing diagrams: verified current in structure; §4's seven architecture fixes are the only staleness found. The `update` flow diagram was checked against `internal/update` and needs nothing.

Six proposals (type · target · what it shows):

1. **Recall query pipeline** · flowchart · c2 (or `docs/architecture/` flow section): per-phrase scoring → note-floor reservation (`noteFloorK=5`) → union/dedup → relevance floor 0.25 → cap 300 → AutoK clustering → `candidate_l2s` top-5 + tag-nomination (cap 40) + supersession ride-along → recency channel appended un-clustered → one YAML payload.
2. **Learn capture kinds** · flowchart · c2 learn flow: corrections / save-requests / **reversals** / QA — the four paths, their write-memory handoffs, D5′ asymmetry on the QA pair, all converging on flock + embed-on-write + vocab assignment; plus please step-7's lessons audit feeding it.
3. **please 7-step + gates** · sequence (swimlane companion to the existing flowchart) · c1: who reviews at A/B/C/D, fresh-per-angle reviewers, argue-to-ACK loops.
4. **Vocab lifecycle** · flowchart · c2/c3 near K6: dual-channel assignment at every write → in-process trigger check (growth ≥40 ∧ ≥14d; untagged >8%; hub >25%) → `refit_pending` persisted → stats verdict line + payload flag → learn Step 1.5 autonomous refit → version bump + index regen.
5. **QA capture path** · sequence · c3: `learn qa` → Q-note + A-note, `Answered by:`/`Answers:` links, A-note competes / Q-note excluded at the four query-pipeline seams, orphan-cleanup on A-write failure, deferred round-3 q-channel.
6. **Ingest/chunking** · flowchart · c2 near the C2→S5 row: `--auto` sweep → manifest mtime/size/hash staleness → non-persistent-prefix skip (`.engram/sweep.json`) → strip → chunk → embed → merge-append under `.manifest.lock`; prune as the separate operator GC.

**c4-skill reconciliation (vault note 171):** recommend **accepting the divergence** — keep diagrams in `docs/architecture/`, don't adopt the deployed c4 skill's pipeline for this repo. The mismatch is deeper than the path: the skill's mechanism is JSON specs + a `targ c4-audit` target, neither of which exists here (verified — `dev/targs.go` has no such target); moving files to `architecture/c4/` would satisfy the path half of the skill's contract while silently failing its audit half. If Joe prefers adopting the skill, that's a deliberate migration project (JSON re-derivation of c1-c3), not a file move. After the decision, amend vault note 171 with the outcome.

## 6. Reference-scrub checklist (execute with the deletions)

**S1 — live-surface repo pointers to delete-recommended docs** (covers both HIST-DEAD and HIST-OBLIGATED targets — anything slated for deletion). Each resolves during the §1 migration: when a ROADMAP Shipped narrative moves to FEATURES/ADR, its evidence pointer either inlines the one-line result or converts to a git-history citation ("deleted 2026-07; see git log"). The complete list (target ← citer): retrieval-probe-results ← ROADMAP:24,322 · failure-eval-material ← ROADMAP:84,108 · revalidation-data ← ROADMAP:105 · crystallization-audit ← ROADMAP:125 · question-anchored-distillation ← ROADMAP:130,140 · link-value-exploration ← ROADMAP:164 · retired-relation-rationales ← ROADMAP:171 (E10) · vocab-notes-build-results ← ROADMAP:172 · atoms-build results ← ROADMAP:186 · lesson-capture-blindspot-options ← ROADMAP:189 · question-shaped-proposals ← ROADMAP:205 · payload-prune smoke design ← ROADMAP:220 **and** the LIVE subprocess spec:11 · recall-depth-dial-design ← ROADMAP:239 · realvault-glance-cost ← ROADMAP:246 · recall-trigger-patterns ← ROADMAP:259 · memory-system-review ← ROADMAP:17 (E8) · note-vs-chunk-ranking plan ← ROADMAP:42 · concurrency plan ← ROADMAP:73 and subprocess spec:6 (update to "Track 0 shipped 2026-07-01", drop the path) · qa-memory-proposals ← ROADMAP:296 and learn skill:50 (E2/E3) · atomic-skills-options ← ROADMAP:188, GLOSSARY:40 (E1) · DESIGN-HISTORY ← embedder.go:63 (E5), memory-system-rigor.md:80 (E6), adr.md:96 (update the ADR-0005 pointer when E7 lands) · baseline-bootstrap-create-RED-results ← the fixture `skills/recall/tests/baseline-bootstrap-create.md:90` (E12) · triage.md ← its GLOSSARY cross-reference at GLOSSARY:1-7's assembly note, if present after E13 (verify at execution) · CLAUDE.md's Directory Structure `docs/` comment ("Active design docs, research prompts, and C4 architecture diagrams") — update to point at the new docs/README.md index when the restructure executes.

**S2 — vault/auto-memory provenance pointers that will dangle (acceptable; no repo action).** These cite deleted docs as *source provenance*, not as content to fetch; git history recovers them. Notes: 26, 27, 68, 79, 82, 85, 91, 95, 99, 102, 118, 119, 120, 140, 149, 153, 158, 163, 169, qa.2026-07-04.atoms + auto-memory `project_failure_mining…`/`project_recall_not_the_cost_bottleneck`/`project_verified_memory_value…`. Optional hygiene: amend the highest-traffic ones (68, 149, 153, 158, 163) with "(doc deleted 2026-07, git history)" — one `engram amend` batch. The load-bearing exception (note 165 → E11) is handled by extraction, and after E2/E3 the learn skill and ROADMAP carry the bands so note 165's pointer degrades gracefully.

## 7. Decision points for Joe

At a glance (details below):

| DP | question | recommendation |
|---|---|---|
| 1 | where do measured results live? | **DECIDED 2026-07-05 (Joe): Option B** — one consolidated ledger of tested results |
| 2 | skill baseline test docs | add per-skill `tests/README.md` indexes; results files delete |
| 3 | FEATURES.md charter boundary | capability entries only; README keeps install/quickstart/CLI reference |
| 4 | where extracted decision content lands | adr.md |
| 5 | retired-relation archive (labeled exception argument) | still delete — rationale folds into the vocab ADR entry |
| 6 | research-followups.md unverified liveness | diff against ROADMAP first, then delete |
| 7 | c4-skill divergence | keep diagrams in docs/architecture/; don't adopt the skill's pipeline |

**DP1 — where do measured results live?** A: each harness keeps its own README+RESULTS pair as sole numeric truth; ROADMAP/FEATURES only ever cite, never restate. B: one consolidated `dev/eval/STATUS.md` ledger. The review recommended A (schema fidelity: $METER vs pass/fail gate vs PASS/BORDERLINE bands differ per harness; EXPERIMENT-LOG's death showed cross-harness ledgers can rot). **DECIDED 2026-07-05 (Joe): Option B** — one ledger of tested results; "one place to go that tells a unified story clearly about what our system is and is not proven to do is much better than several distinct places to look and mentally track, especially when some of them are superseded by others." Considered A; Joe chose B because the unified current-truth story and explicit supersession outweigh per-harness schema fidelity. Execution shape: row unit = the tested claim/capability, carrying verdict (proven / refuted / unmeasured / superseded-by) + the measured figure + vintage + a link to the raw per-harness data (which stays on disk as data, not as a competing narrative); `dev/eval/traps/RESULTS.md`'s live content folds in as the seed rows; ROADMAP/FEATURES cite ledger rows only. The rot risk that killed EXPERIMENT-LOG transfers to the ledger — the execution cycle should make updating it part of every eval's definition-of-done.

**DP2 — skill baseline test docs.** A: leave the 10 scenario files as unindexed fixtures; results files delete. B: add a `skills/<skill>/tests/README.md` index naming which baseline locks which current behavior; results files still delete (E12's cited snippet inlined). **Recommend B** — CLAUDE.md mandates writing-skills TDD for every skill edit, so these are reusable instruments, and an index is exactly "one obvious place to go."

**DP3 — FEATURES.md charter boundary.** A: one entry per shipped user-visible capability (what + why-pointer + validation-pointer); README keeps install/quickstart/CLI reference. B: FEATURES also absorbs README's Binary-commands block. **Recommend A** — the CLI block is reference-while-typing material, isn't duplicated anywhere, and moving it breaks README as a standalone quickstart.

**DP4 — where does extracted DESIGN-HISTORY/decision content land?** A: adr.md (it already owns Accepted/Superseded vocabulary and is the C4 docs' cross-link target). B: a "decisions carried forward" appendix in ROADMAP. **Recommend A** — B recreates the scatter this review exists to fix.

**DP5 — explicitly labeled exception argument (contradicts the stated instruction).** `artifacts/2026-07-02-retired-relation-rationales.md` was created *to be* the durable archive ROADMAP:171 points at. Keeping it contradicts "anything historical should be deleted." **Recommendation: still delete** — fold the rationale into the new vocab tag-nomination ADR entry (E10) and repoint ROADMAP:171; the archive-doc pattern predates this restructure's rule and shouldn't survive it. The keep option exists if you value the verbatim edge-by-edge archive.

**DP6 — `research-followups.md` (E9), the one unverified liveness call.** A: the execution cycle runs the item-by-item diff against ROADMAP's parked/future sections, mirrors anything missing, then deletes (cost: one focused agent pass). B: delete now and accept the risk that unmirrored parked research directions are lost to git history. **Recommend A** — the doc is 671 lines of consolidated followups and the diff is cheap relative to re-deriving a lost direction.

**DP7 — c4-skill divergence (full analysis in §5).** A: keep diagrams in `docs/architecture/`, accept divergence from the deployed c4 skill, note the hand-authored workflow in docs/README.md. B: adopt the c4 skill's pipeline — a deliberate migration project (JSON re-derivation of c1-c3 + a new `targ c4-audit` target), not a file move. **Recommend A** — the skill's mechanism has zero footprint here and a path-only move would fake compatibility. Either way, amend vault note 171 with the outcome.

## 8. Suggested execution order (follow-up cycle)

1. **Correctness fixes (§4)** — independent, no decisions needed, kills all 16 misleads-design rows.
2. **Create `docs/FEATURES.md` + new adr.md entries** (vocab tag-nomination incl. E10 rationale, D5′ incl. E4 caveat, Track-0 flock, tier-routing) — the landing zones.
3. **Extractions E1–E12** + ROADMAP round-2/3 consolidation (E2/E4/E11) + learn-skill repoint (E3 — a SKILL.md edit: writing-skills TDD applies).
4. **ROADMAP slim-down** to future-only, converting §6-S1 pointers as narratives migrate.
5. **Relocate** the two 2026-06-04 specs to `docs/architecture/`; fold triage.md into GLOSSARY per E13's six rulings; create `docs/README.md` index.
6. **Delete**: 98 HIST-DEAD + 8 post-extraction HIST-OBLIGATED + triage.md + the emptied `docs/superpowers/` tree (git is the archive; chunks are already ingested).
7. **Diagrams (§5)** — staged; each new diagram lands with the doc it explains.
8. **DP2 index** + optional §6-S2 vault-note amend batch; amend vault note 171 with the DP7 outcome.
9. **Delete this report and its plan** (`docs/design/2026-07-04-docs-restructure-suggestions.md`, `docs/superpowers/plans/2026-07-04-docs-restructure-review.md`) — the workspace rule applies to them too; their conclusions have graduated by this step.

---

**Ask-element coverage:** (a) live-only/delete-historical → §2, §3, §6, §8-6 · (b) SRP one-doc-per-responsibility → §1, §7 DP3/DP4 · (c) folder + index README, one obvious place → §1 (docs/README.md), §8-5 · (d) mermaid diagrams → §5 · (e) correct vs code and skills → §4 (tally owned there: 28 verified findings → 23 rows; future-work docs exempted: subprocess spec kept LIVE) · (f) complete & concise → §4 gaps (learn qa, matched-note floor), §1 ROADMAP slim-down, §2 deletions · (g) thorough multi-angle + concrete suggestions → the whole report; every edit carries a verbatim anchor; the one unverified item is labeled (E9).
