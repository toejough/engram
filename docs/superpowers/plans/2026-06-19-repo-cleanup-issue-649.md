# Repo Cleanup (Issue 649 + stale-artifact sweep) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the physically-retired transcript/episode binary surface (issue 649), fix every doc that still describes it, refresh stale recall test fixtures, prune dead git branches/worktrees, delete stale eval run artifacts, and consolidate historical design docs into one concise decisions record.

**Architecture:** Six independent units (A–F). A is code removal verified by the existing suite + a real-binary smoke test (the change is deletion, so the suite is the oracle). B/C/E/F are documentation/artifact edits verified by `targ check-full`, grep-for-absence, and broken-link scans. D is destructive git hygiene gated on a recorded backup. Object-level work is delegated to subagents per the `route` skill; this orchestrator routes, decomposes, verifies, and synthesizes.

**Tech Stack:** Go (no CGO), `targ` build system, GoMLX/Hugot embedder, git worktrees, `gh` for issues, engram CLI.

**Decisions locked with the user (2026-06-19):**
- Branches/worktrees: delete branches merged into main; **inspect** the unmerged ones and delete only clearly-abandoned scratch (keep salvageable); prune prunable/abandoned worktrees; keep locked worktrees holding live work.
- Historical docs: keep **one** concise history doc of relevant/informative decisions; delete the rest **after confirming git retains them**; **keep** docs still actively referenced by architecture/README/code; update genuinely-active-but-stale docs.
- Issue 649 subcommands: **remove fully** (no episode notes exist; skills don't invoke them; `internal/transcript` stays for `engram ingest`).
- `dev/eval`: delete the 667 raw per-cell run JSONs (git retains); distill key eval conclusions into the history doc and delete the redundant `results-*` tables; keep harness Go, `testdata/`, and `*_spec.json`.

**Verification requirements (from memory):**
- *(L2 note 54)* The binary unit is not done until the **real installed `engram` binary** runs with **real args from a non-data-dir cwd** — in-memory tests and `--help` are necessary but not sufficient.
- *(MEMORY)* No destructive git command without `git status`/backup first. `targ` for all build/test/check. Close issues via `gh`. Don't assume deletion intent — confirmed with the user above.

**Gate-A-confirmed facts (already verified against the tree):**
- `internal/learnmarker` is imported only by `transcript.go` + its own test → safe to delete the whole package.
- `internal/transcript` is imported by `ingest.go` (+ ingest tests) and by `cli.go`'s `osDirLister` (tested by `adapters_test.go`) → **the package and the `cli.go` import STAY.**
- `commands/` has zero references to retired commands (`grep` clean) → no action.
- `cmd/engram/main.go` is `cli.Main(...)` only → no change.
- The 649 episode/transcript surface had MORE consumers than the file lists first captured — `resituate.go` (a KEPT command) and several test files were discovered at execution via `targ check-full`. The file list above is now corrected. `targ check-full` (the repo's gate) flags unused/undefined symbols, so a green check post-cleanup is the real completeness backstop; no dead Go code was identified *outside* the episode/transcript surface, and a deeper whole-repo dead-code audit is out of scope.

---

## Unit A — Remove transcript/episode/migrate-episodes binary surface (Issue 649 §1)

**Delete (whole files):**
- `internal/cli/transcript.go`, `internal/cli/transcript_test.go`
- `internal/cli/episode_body.go`, `internal/cli/episode_format_test.go`, `internal/cli/episode_idempotency_property_test.go`
- `internal/cli/migrate_episodes.go`, `internal/cli/migrate_episodes_test.go`
- `internal/learnmarker/` (whole package incl. `learnmarker_test.go`)

**Modify:**
- `internal/cli/targets.go` — drop the `.Name("transcript")`, `.Name("episode")` (under learn), `.Name("migrate-episodes")` builders + `LearnEpisodeArgs` and episode-only arg fields. Keep `migrate-links`, `resituate`, `ingest`, `learn fact|feedback`, `show`, `amend`, `activate`, `query`, `update`.
- `internal/cli/learn.go` — excise all episode code (`typeEpisode`, `EpisodeRange`, `episodeLink`, `Preceding`, `ListEpisodes`, `runLearnFromEpisodeArgs*`, `resolveEpisodeBody`, `resolveTranscriptSpan`, the episode constants/headings/`errEpisode*`) **and drop the now-unused `import ".../internal/transcript"` at learn.go:20** (it was used only by episode functions). [code-align finding 3]
- `internal/cli/cli.go` — remove the dead `newTranscriptDeps` (lines ~307-322; its only caller was `targets.go`'s deleted transcript builder). **KEEP `osDirLister` + its `internal/transcript` import** — `adapters_test.go` tests it directly. [code-align findings 4, 5]
- `internal/cli/check.go` — remove `|| noteType == typeEpisode` from `isSituationBearing` (line ~226). [code-align finding 9]
- `internal/cli/export_test.go` — remove the wrappers of deleted symbols: `AdvanceAndReportMarkerForTest`, `EmitSegmentsForTest`, `EmitTranscriptsForTest`, `ExportEmitTranscripts`, `ExportNewOsMigrateEpisodesDeps`, `ResolveProjectSlugForTest`, `ResolveStateDirForTest`, `RunLearnFromEpisodeArgsWithReaderForTest`, `RunTranscriptForTest`, `ExportRenderEpisodeBody`, `ExportRenderEpisodeFrontmatter`, `ExportComputePrecedingLinks`, `ExportParseEpisodeBody`, `ExportEpisodeFields`, `ExportEpisodeLink`, `ExportEpisodeRange`, `ParseFromTranscriptRangeForTest`, **and** `NewTranscriptDepsForTest`, `DefaultSessionPathResolverForTest`, `ResolveSessionPathForTest` (re-verify: these three wrap deleted `newTranscriptDeps`/`defaultSessionPathResolver`/`resolveSessionPath`). **KEEP `ExportNewOsDirLister`** (used by `adapters_test.go`). [code-align finding 6 + re-verify]
- `internal/cli/learn_test.go` — delete the **16** episode test functions; criteria "`TestEngramLearn_Episode_*` or references `LearnEpisodeArgs`/episode export stubs" misses these — also delete: `TestTierFrontmatter_EpisodeDefaultsToL1` (uses `Type:"episode"`), `TestDefaultSessionPathResolver`, `TestResolveSessionPath_GetwdError/HappyPath/HomeError`, and the `TestRenderEpisodeFrontmatter_*` render tests (episode-specific). **Keep** fact/feedback/Luhmann + non-episode render tests. [code-align finding 7 + re-verify]
- `internal/cli/targets_test.go` (re-verify NEW-1, KEPT file) — `TestTargets` "returns expected target count" asserts `HaveLen(14)` → **12** (only `transcript` + `migrate-episodes` are *top-level* targets removed; the `learn` group stays so the `episode` sub-subcommand removal doesn't change the count), update the accompanying comment, and delete the subtests that invoke `learn episode` / `transcript` / and `TestTargets_MigrateEpisodes`. (Confirm 12 once the build passes.)
- `internal/cli/signal_test.go` (re-verify NEW-2, KEPT file) — the `HaveLen(14)` target-count assertion (≈line 74) → **12**.
- `internal/cli/resituate.go` (EXECUTION-DISCOVERED, KEPT command) — episodes are retired entirely, so retire episode-resituation: delete `rerenderEpisode` (≈lines 164-194) and the `case typeEpisode:` branch in `resituateContent` (≈line 276) so episode-typed notes (none exist) fall through to `errResituateUnknownType`; update `RunResituate`'s doc comment that references episode bodies. This removes the last consumers of `episodeFrontmatterDoc`/`episodeFields`/`renderEpisodeFrontmatter`/`typeEpisode`.
- `internal/cli/resituate_test.go` (EXECUTION-DISCOVERED) — delete `TestRunResituate_Episode`, the episode subtest in `TestRunResituate_ContentErrors`, and fixtures `resituateEpisodeNote`/`episodeNoteWithCreated`; keep the fact/feedback resituation tests.
- `internal/cli/learn_adapters_test.go` (EXECUTION-DISCOVERED) — delete `TestOsLearnFS_ListEpisodes_*` (3 tests) + the `episodeNoteForListing` fixture (they exercise the deleted `osLearnFS.ListEpisodes` + `cli.EpisodeRange`).
- `internal/cli/cli_test.go` (EXECUTION-DISCOVERED, KEPT file) — delete the real-binary end-to-end tests `TestEngramLearn_Episode_L1_EndToEnd` and `TestEngramTranscript_DateRangeEndToEnd` (they `exec` the removed `learn episode` / `transcript` subcommands); keep the fact/feedback e2e tests + shared helpers.
- `internal/cli/export_test.go` — also remove `ResolveMaxBytesForTest` (EXECUTION-DISCOVERED: wraps the deleted `resolveMaxBytes`).
- `internal/cli/invariants_m7_property_test.go` — the file holds exactly one test (`TestInvariant_M7_MarkerMonotonic`) + a helper, all marker-monotonicity (retired with `learnmarker`); remove the test → the file is empty → **delete the file**. [code-align finding 8 + re-verify]
- `internal/cli/sweepspec.go` — re-verify: **confirmed clean** (its "transcript" strings are `ClaudeExcludeDirs` comments). No action.

- [ ] **Step A1: Record the green baseline** — Run `targ test`; record PASS + count. Pre-change oracle.
- [ ] **Step A2: Delete the standalone files**
```bash
cd /Users/joe/repos/personal/engram
git rm internal/cli/transcript.go internal/cli/transcript_test.go \
       internal/cli/episode_body.go internal/cli/episode_format_test.go \
       internal/cli/episode_idempotency_property_test.go \
       internal/cli/migrate_episodes.go internal/cli/migrate_episodes_test.go
git rm -r internal/learnmarker
```
- [ ] **Step A3: Excise registrations in `targets.go`** — remove the three builders + `LearnEpisodeArgs`/episode args (file list above).
- [ ] **Step A4: Excise episode code from `learn.go` + drop its `internal/transcript` import.**
- [ ] **Step A5: `cli.go` — remove dead `newTranscriptDeps`; keep `osDirLister` + the import. `check.go` — remove the `typeEpisode` disjunct.**
- [ ] **Step A6: Clean test scaffolding** — `export_test.go` (remove the listed wrappers, keep `ExportNewOsDirLister`); `learn_test.go` (remove episode tests); `invariants_m7_property_test.go` (remove marker test / file).
- [ ] **Step A7: Absence grep — no dangling references**
```bash
grep -rn 'learnmarker\|RunTranscript\|runTranscript\|LearnEpisode\|MigrateEpisode\|migrate_episodes\|episodeBody\|EpisodeBody\|typeEpisode\|advanceAndReportMarker\|AdvanceAndReportMarker\|emitSegments\|emitTranscripts\|EpisodeRange\|ParseFromTranscriptRange\|newTranscriptDeps\|NewTranscriptDeps\|renderEpisode\|RenderEpisode\|computePrecedingLinks\|buildEpisodeFields\|episodePrecedingLinks\|episodeRangeFromNote\|activeAtStart\|immediatePrior\|defaultSessionPathResolver\|DefaultSessionPathResolver\|resolveSessionPath\|ResolveSessionPath' internal/ cmd/ --include='*.go'
```
Expected: no output. (Matches inside the kept `internal/transcript/` package for *reading* won't hit these symbols; if one does, inspect.)
- [ ] **Step A8: Build + full check** — `targ check-full`. Expected PASS. Fix all reported errors in one pass.
- [ ] **Step A9: Real-binary smoke test (memory note 54 — non-waivable)**
```bash
cd /Users/joe/repos/personal/engram && targ build
cd /tmp && /Users/joe/repos/personal/engram/engram --help            # transcript/episode/migrate-episodes ABSENT
/Users/joe/repos/personal/engram/engram transcript 2>&1 | head -3    # unknown-subcommand error
/Users/joe/repos/personal/engram/engram learn --help 2>&1 | head -20 # fact + feedback only
/Users/joe/repos/personal/engram/engram ingest --auto 2>&1 | tail -3 # still works
/Users/joe/repos/personal/engram/engram query --phrase "smoke" 2>&1 | tail -3  # still works
```
- [ ] **Step A10: Commit** — `refactor(cli): remove retired transcript/episode/migrate-episodes surface (#649)` with body explaining the lazy-L2 retirement, and `AI-Used: [claude]`.

---

## Unit B — Fix architecture docs that still describe transcript/episode (Issue 649 §2)

Use the `c4` skill for all Mermaid diagram edits (it applies to the full surface below, not just the sequence diagram).

**`docs/architecture/c2-containers.md`** [docs-align finding 2]:
- Node label (≈line 18): drop `transcript` from `engram CLI … transcript/learn/query/embed/update` (and reflect `ingest`).
- C1→C2 edge label (≈line 26): `subprocess engram transcript/learn/query` → current subcommands.
- Container catalog row (≈line 43): rewrite "transcript scan+marker, note write…"; remove the retired `M2-segments` defect reference. **Also** the C5 marker catalog row (≈line 46) `M2-segments over-advance` and (in c3) the `m2[[⚠ M2-segments …]]` node + `clitr -.-> m2` edge (≈c3 lines 93, 96).
- C2→C3 relations row (≈line 53): drop episode embed-routing language. **C2→C5 relations row (≈line 55)**: its stale text is "`transcript --mark` reads the marker, scans `> marker`, advances it" — rewrite to reflect `engram ingest --auto` advancing the marker, **not** episode embed-routing. [re-verify Issue B]
- "Flow: learn" sequence diagram (≈lines 116-142): replace `engram transcript --mark` + `engram learn episode` with `engram ingest --auto` + `engram learn fact|feedback`.

**`docs/architecture/c3-components.md`** [docs-align finding 1 — the big one]:
- Delete the `subgraph PT[engram transcript — process]` block (K1 transcript reader, K2 context.Strip, K1b cli/transcript wiring; ≈lines 25-29).
- Delete the `K3 · learnmarker` node (≈line 48) and the data edges `tr→strip→clitr→markers→lm→sessions` (≈lines 63-68).
- Remove K1/K1b/K2/K3 from the `class … comp` classDef line (≈line 86) — these map to node names `tr,strip,clitr,lm` in `class tr,strip,clitr,…,lm comp`. [re-verify classDef mapping]
- Remove/retitle the `E4` defect annotation referencing `episode embed=situation` (≈line 91).
- Component-catalog table (≈lines 102-106): drop K1/K3 rows; remove episode references from K4/K5 rows.
- Data-contracts prose (≈lines 143-152): rewrite the "`engram transcript` emits the stripped chunk to stdout" bullet as "`engram ingest --auto` scans chunk sources, re-chunks changed content, emits chunk identifiers."
- Delete the entire "Flowchart: marker forward-progress" section (≈lines 228-251).

**`docs/architecture/adr.md`** [docs-align finding 3]:
- ADR-0009 (transcript-marker exactly-once) → **Superseded** (rationale: marker path retired with `engram transcript`; mirror ADR-0005's treatment).
- ADR-0008 (per-arc episodes as L1 evidence) → **Superseded** (episode type retired; chunks are L1 evidence via `engram ingest`).
- ADR-0006 (embed source by kind: episode `situation` vs body) → **Superseded** / annotated (episode routing retired; all notes embed body).
- ADR-0004 (≈line 89) and ADR-0010 (≈line 221): clean the residual `episode → L1 (rigid)` / `episode provenance` language to match reality. ADR-0010's status line also references `newTranscriptDeps` (deleted in Unit A) — update it. [re-verify Issue: ADR-0010 status]
- ADR-0005's Superseded status line (≈line 102) contains a **bare path** `docs/superpowers/specs/2026-06-09-lazy-l2-synthesis-design.md` which Unit E deletes — repoint it to `docs/DESIGN-HISTORY.md` (or a git ref) here in B4. [re-verify Issue C]

- [ ] **Step B1:** Read all three docs end-to-end; produce a line-anchored checklist from the file list above.
- [ ] **Step B2:** Edit `c2-containers.md` (all five locations).
- [ ] **Step B3:** Edit `c3-components.md` (all eight locations — subgraph, node, edges, classDef, defect, catalog, data-contract, flowchart).
- [ ] **Step B4:** Edit `adr.md` (supersede 0009/0008/0006; clean 0004/0010).
- [ ] **Step B5: Verify absence** — broaden the grep beyond command strings:
```bash
grep -rn 'engram transcript\|learn episode\|--mark\|--segments\|transcript marker\|migrate-episodes\|learnmarker\|marker forward-progress\|episode embed\|K3' docs/architecture/
```
Expected: only intentional Superseded/historical mentions remain.
- [ ] **Step B6: Commit** — `docs(arch): retire transcript/episode from C2/C3 + supersede ADR-0006/0008/0009 (#649)` with `AI-Used: [claude]`.

---

## Unit C — Refresh stale recall test fixtures (Issue 649 §3)

**Resolution of the deferred L1-episode question:** L1 episodes are **retired** — `engram learn episode` is removed (Unit A), no `node_type: episode` notes exist in any live vault, chunks are the raw event memory. Fixtures premised on L1 episodes as a live note type are stale in framing. Where a fixture encodes a still-valid invariant (recency-tiebreaker on conflict, bootstrap-on-near-empty, empty-vault skip, multi-query merge), re-derive it against the **agent-judged covered/near/absent** model over notes/chunks — do not mechanically find-replace.

**Delete (definitively stale — retired `nearest_l2` cosine-band gate + fire-and-forget dispatch):** `skills/recall/tests/baseline-three-band-GREEN-results.md`, `baseline-three-band-RED-results.md`, `baseline-three-band-writes.md`, `pressure-three-band-RESULTS.md`, `pressure-three-band.md`.
**Re-derive (drop `node_type: episode`/`provenance.transcript_range`; `nearest_l2`→`candidate_l2s`; three-band→covered/near/absent):** `baseline-recency-l1-episode.md` (+ RED/GREEN results) → recency-tiebreaker over two conflicting **notes** by `created`; `baseline-bootstrap-create.md` (+ RED/GREEN results); `baseline-empty-vault-skip-l2.md`; `baseline-multi-query.md`.
**Untouched (already current):** `baseline-judgement-and-cascade.md`, `baseline-GREEN-results.md`, `baseline-RED-results.md`.

This unit edits **test fixtures only**, not `SKILL.md` (already on the agent-judged model). If re-derivation reveals a genuine SKILL.md gap, STOP and invoke `superpowers:writing-skills` separately.

- [ ] **Step C1:** Read each candidate; confirm three-band set tests only the retired gate (delete) vs. judgment-dependent set encodes a reusable invariant (re-derive).
- [ ] **Step C2:** `git rm` the five three-band fixtures.
- [ ] **Step C3:** Re-derive each judgment-dependent fixture (note-vs-note conflict, not episode-vs-episode).
- [ ] **Step C4: Verify** — `grep -rn 'nearest_l2\|three-band\|node_type: episode\|transcript_range' skills/recall/tests/` returns nothing in retained/rewritten fixtures.
- [ ] **Step C5: Commit** — `test(recall): retire three-band fixtures, re-derive invariants to agent-judged model (#649)` with `AI-Used: [claude]`.

---

## Unit D — Prune dead git branches and worktrees

Destructive — gated on a recorded backup. **Runs last** (after A–C, E, F are committed and the cleanup branch is ready to merge) so the live working branch is never a deletion candidate.

- [ ] **Step D1: Back up every branch tip + worktree list**
```bash
cd /Users/joe/repos/personal/engram
git for-each-ref --format='%(refname:short) %(objectname) %(upstream:short)' refs/heads > /tmp/engram-branch-tips-20260619.txt
git worktree list > /tmp/engram-worktrees-20260619.txt
wc -l /tmp/engram-branch-tips-20260619.txt
```
- [ ] **Step D2: Remove stale/prunable worktrees first** (a branch checked out in a worktree can't be deleted until the worktree is gone)
```bash
git worktree prune -v
git worktree list
```
For remaining `.claude/worktrees/agent-*` and `.worktrees/*` holding no live work: `git worktree remove <path>` (`--force` for locked ones whose work is merged/pushed). Keep worktrees with unpushed, unmerged, salvageable work. Report each removal.
- [ ] **Step D3: Delete branches merged into main** (excluding `main` and the current cleanup branch)
```bash
cur=$(git branch --show-current)
git branch --merged main | grep -vE '^\*| main$' | grep -vF "$cur" | while read b; do git branch -d "$b"; done
```
`-d` refuses anything not actually merged — a safety backstop. Report deleted + any refused (worktree-checked-out → investigate).
- [ ] **Step D4: Inspect the unmerged branches; delete only abandoned scratch** [ask-align finding 1 — exclude the current branch]
```bash
cur=$(git branch --show-current)
for b in $(git branch --no-merged main | grep -vE '^\*' | grep -vF "$cur"); do
  printf '%s\t%s commits\t' "$b" "$(git rev-list --count main..$b)"; git log -1 --format='%ci' "$b"
done
```
Build a table (branch → unique-commit count → age → verdict). **Abandoned** (agent scratch, 1000+ behind, superseded) → `git branch -D`; **salvageable** → keep + report. Delete only after the table is reviewed.
- [ ] **Step D5: Verify** — `git branch | wc -l`, `git worktree list`; report before/after; confirm `main` intact and `git status` clean.
- [ ] **Step D6:** No commit (ref changes). Record the deleted set + verdict table in the step-6 summary.

---

## Unit E — Consolidate historical design docs into one concise record

**Create:** `docs/DESIGN-HISTORY.md` — top-level (peer to `architecture/`) so it's discoverable from the architecture docs that will link to it. Chronological, one short entry per meaningful decision with the *why* and a `git log`-recoverable path; no blow-by-blow.

**Keep (do NOT delete) — docs still actively referenced:** before deleting anything, find every doc under `docs/superpowers/{plans,research,specs}` (+ `docs/plans/`) referenced by an *active* doc (`docs/architecture/*`, `README.md`, `CLAUDE.md`) or by code, and KEEP those as active reference (e.g. `memory-invariants.md` is linked from c2/c3/adr). [docs-align finding 4]

**Delete (after confirming git retains them) the remaining historical docs:** the rest of `docs/superpowers/plans/*`, `docs/superpowers/research/*`, `docs/superpowers/specs/*`, and the single `docs/plans/2026-05-24-engram-query-spike.md`. [ask-align finding 5 — unambiguous path]

**Fix broken cross-links** [docs-align finding 4 + re-verify]: after deletion, scan `docs/` for references into deleted files and repoint to `DESIGN-HISTORY.md` (or the kept doc / git path). Two link forms exist — markdown links AND **bare-path text** (the latter is missed by a naive link grep). Known references (verified):
  - **KEEP (actively linked, do NOT delete):** `docs/superpowers/specs/2026-06-04-memory-invariants.md` (linked from c2:7, c3:5, c3:92, adr:11) and `docs/superpowers/specs/2026-06-04-memory-system-rigor.md` (linked from adr:6). The E2 keep-classification must retain both.
  - **Repoint (deleted, references are bare-path text):** `adr.md:102` → `2026-06-09-lazy-l2-synthesis-design.md` (handled in B4) and `c1-system-context.md:385` → `2026-05-14-tiered-memory-design.md`. Repoint both to `DESIGN-HISTORY.md`.

**Update stale-but-active docs:**
- `README.md` [ask-align finding 6]: remove the now-false command blocks — lines ~75-78 (`engram transcript …`), ~81 (`engram learn episode …`), and the prose ~132-136 (per-harness marker / first-run / byte-cap). This is factual-correctness fallout from Unit A; the broader README recall-narrative rewrite stays with the **open #647** (branch `origin/opencode-plugin`) — leave a one-line note in the commit body coordinating the boundary.
- `docs/triage.md` [docs-align finding 5]: move items 1, 2, 7, 8 (which reference `transcript.go`, the `Package transcript` doc comment, and the `learnmarker` package) to **Decided → retired**.
- `docs/GLOSSARY.md` [docs-align finding 5]: remove/rewrite the entries defining retired vocabulary as current — `binary` (drop `transcript`), `engram learn` (drop `episode`), `Episode (note type)`, `engram transcript (subcommand)`, `marker (progress marker)`/`learnmarker`, `first-run handling`, `subcommand` (drop `transcript`), the `--mark` status row, and the `--project`/`--issue`/`candidate` entries referencing episode/transcript.

**Delete (planning artifact):** `docs/superpowers/plans/2026-06-19-repo-cleanup-issue-649.md` (this plan) in step 6 — git history retains it.

- [ ] **Step E1: Confirm git retains everything** — `git status --short docs/` clean (commit any untracked/modified doc first); `git ls-files docs/superpowers | wc -l`.
- [ ] **Step E2: Classify** (delegate to a reader subagent) — read every file under `docs/superpowers/{plans,research,specs}/` + `docs/plans/2026-05-24-engram-query-spike.md`; output two lists: (a) **active-reference** docs (linked by `docs/architecture/*`/`README`/`CLAUDE.md`/code — keep), (b) **historical** docs (distill + delete).
- [ ] **Step E3: Distill** (same subagent) — write `docs/DESIGN-HISTORY.md` from the historical list: sections tiered-memory research → v2/v3 → eval harness → lazy-L2 → please/route skills; each a short paragraph with decision + rationale + git path. Include the key eval conclusions from `dev/eval/cumulative/results-*.md` (Unit F).
- [ ] **Step E4: Review** the history doc against the originals for material omissions (Gate C also covers this).
- [ ] **Step E5: Delete historical docs** — `git rm` the historical list only (NOT the active-reference list, NOT `DESIGN-HISTORY.md`, NOT `docs/architecture/*`). Verify the kept set survived.
- [ ] **Step E6: Fix broken links + update stale-but-active docs** — repoint dead relative links; edit `README.md`, `triage.md`, `GLOSSARY.md` per the lists above. Verify:
```bash
grep -rn 'engram transcript\|learn episode\|migrate-episodes\|learnmarker' README.md docs/GLOSSARY.md docs/triage.md
# expect only intentional/historical mentions
# broken-link scan — BOTH markdown links AND bare-path text (re-verify Issues C/D):
grep -rnE '(\]\(\.\.)?/?(docs/)?\.{0,2}/?superpowers/(specs|plans|research)/[^) ]+' docs/ \
  | grep -vF 'DESIGN-HISTORY' || echo "no dead specs/plans/research references"
# every surviving hit must point to a KEPT file or be repointed to DESIGN-HISTORY.md
```
- [ ] **Step E7: Commit** — `docs: consolidate design history into DESIGN-HISTORY.md; drop superseded plans/specs/research; fix stale README/GLOSSARY/triage (#649)` with `AI-Used: [claude]`.

---

## Unit F — Delete stale eval run artifacts

**Delete:** the 667 raw per-cell JSONs under `dev/eval/cumulative/runs/**` (git retains); the redundant `dev/eval/cumulative/results-table.md`, `results-table.txt`, `results-v2.md`, `results-lazy-l2-opus.md`, `results-real-skill-haiku.md`, `results-real-skill-opus.md` **after** their conclusions are folded into `DESIGN-HISTORY.md` (Unit E Step E3).
**Keep:** the harness Go (`run.go`, `score.go`, `aggregate.go`, …), `dev/eval/testdata/**`, `dev/eval/cumulative/{notes,links,feeds}_spec.json`, `dev/eval/cumulative/README.md`.

- [ ] **Step F1:** Confirm git tracks the run JSONs (so deletion is recoverable): `git ls-files 'dev/eval/cumulative/runs/**' | wc -l` (expect ~667).
- [ ] **Step F2:** Ensure Unit E Step E3 captured the eval conclusions (sequence: F runs after E3's distillation, or E3 reads the results files before F deletes them).
- [ ] **Step F3:** `git rm -r dev/eval/cumulative/runs` and `git rm dev/eval/cumulative/results-table.md dev/eval/cumulative/results-table.txt dev/eval/cumulative/results-v2.md dev/eval/cumulative/results-lazy-l2-opus.md dev/eval/cumulative/results-real-skill-haiku.md dev/eval/cumulative/results-real-skill-opus.md`.
- [ ] **Step F4: Verify the harness still builds/tests** — `targ check-full` (the eval Go code must not reference the deleted result files as fixtures; if any test reads `runs/**` or `results-*`, STOP and reassess).
- [ ] **Step F5: Commit** — `chore(eval): delete raw run artifacts from 2026-06-08 experiment; conclusions preserved in DESIGN-HISTORY (#649-adjacent)` with `AI-Used: [claude]`.

---

## Self-Review

**Spec coverage:** Issue 649 §1→A, §2→B, §3→C. Ask "branches"→D; "worktrees"→D2; "stale docs"→E; "artifacts"→F (eval) + untracked strays already gitignored (no action, verified); "commands"→A (CLI) + `commands/` verified clean (no action); "code"→A + linter-guarantee documented (no separate audit). README factual fallout→E6, coordinated with open #647.

**Placeholder scan:** verification commands are concrete; the only delegated synthesis is Unit E (classify + distill) and Unit C (fixture re-derivation), both bounded by explicit content specs.

**Type/symbol consistency:** Unit A's removed symbol set is itemized per file from the code-alignment review and closed by the A7 absence-grep; `internal/transcript` (kept) vs `internal/cli/transcript.go` (deleted) vs `internal/learnmarker` (deleted) boundaries are explicit; `cli.go` import KEPT (osDirLister), `learn.go` import DROPPED.

**Ordering:** A, B, C, E file-disjoint → parallel-capable (worktrees if needed). F depends on E3 (conclusions distilled before results deleted). D runs last (after the cleanup branch is ready) so it's never a self-deletion candidate. Each code/doc unit verifies with `targ check-full` + grep-for-absence before its commit; A adds the real-binary smoke test.
