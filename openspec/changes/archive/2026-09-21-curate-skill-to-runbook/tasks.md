## Status

Sections 1, 2, 2a, 2b, 3, 3a, and 4 are done (2026-09-21): the curate skill is retired, the runbook is
promoted to the production vault (`1049.2026-09-21.curate-review-pending-offers`), live docs are updated,
and `targ check-full` / `openspec validate --all --strict` are green. **Remaining:** 5.1 (sync the delta
spec to the main spec, archive the change) — held per instruction until Joe reviews this stage's work
(including an independent fresh-context re-verification of the `enumeration.md` disposition list, task 4.3).

## 1. Eval baseline (done in the previous stage)

- [x] 1.1 Curate eval fixture (fictional beekeeping vault, three offers covered/near/absent), harness `vault_template` support, skill and bare baselines (`results/1.3_curate_baseline_sonnet5.md`; skill row end_state 3/3, followed_all 2/3; bare 0/3, 0/3); commit b55080f1

## 2. Conversion

- [x] 2.1 Tag every `curate/SKILL.md` section carry / drop-as-shim-floor / adapted, place the seven red-flag rows, and record the deltas in `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/curate-conversion-fidelity-report.md`
- [x] 2.2 Write the top runbook `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskCurate/Curate-R/vault/<luhmann>.<date>.curate-review-pending-offers.md` (type runbook, tier, process-shaped `situation`, `done_when`, `red_flags` <= 1200 bytes, `triggers`, body = SKILL.md steps verbatim where possible; keep host-local-only and never-hand-off-to-write-memory; keep the `grep -l '^pending: true$'` scan)
- [x] 2.3 Verify `red_flags` bytes against the real truncation budget (`internal/cli/redflags_truncation.go`, `engram show --vault <fixture>` prints no omission marker)
- [x] 2.4 Rebuild sidecars: `engram embed apply --vault <fixture vault> --all`, 0 stale
- [x] 2.5 Wire `carrier_r_src` in `fixtures/curate/task.json`; `probe_phase2.py --task curate --arms R --shim-only --setup-only` builds (no spend) and the trial vault contains the runbook plus the fixture offers
- [x] 2.6 Run the phase2 tests (`python3 -m pytest dev/eval/cumulative/runbook_vs_skill/phase2/test_probe_phase2.py`)
- [x] 2.7 Fresh-context reviewer checks the fidelity report against `SKILL.md` for lost content — reviewed 2026-09-21 by a separate Opus reviewer, verdict PASS-WITH-FIXES, fixes applied in commits 4beb714f (fixture + fidelity report), b983e29d (retrieval check), 5c4cdbaf (D5 confirmation, payload-hint decision, fidelity nits — confirmed this commit touches `curate-conversion-fidelity-report.md`)

## 2a. Notice reword (D4)

- [x] 2a.1 RED: change `TestWriteUpdateReport_PendingOfferHint` (`internal/cli/update_test.go`) and add an `offer_test.go` case for the write nudge to assert the trigger query (`engram query --text "curate pending offers"`) and no skill naming; `targ test` fails
- [x] 2a.2 GREEN: reword `pendingOfferUpdateNotice` and `pendingOfferWriteNudge` in `internal/cli/offer.go`; `targ test` passes
- [x] 2a.3 Reword "curation skill" / "offer-curation skill" to "curation runbook" in comments and flag descriptions (`amend.go`, `learn.go`, `update.go`, `serve.go`, `targets.go`, tests)
- [x] 2a.4 `targ check-full` green; smoke: run the reworded notice through a scratch-home `engram update` (or show the unit-test evidence); never touch the real vault

## 2b. Payload hint (D9)

- [x] 2b.1 RED: tests for `pending_offers_hint` present iff `pending_offers` is true, equal to the shared constant, byte-identical payload without offers, merged query, served round-trip, and notices embedding the same constant; `targ test` fails
- [x] 2b.2 GREEN: `pendingOfferCurateInstruction` shared constant, `queryPayload.PendingOffersHint`, set in `runQuery` and `mergeQueryPayloads`; `targ test` passes (commit b050db1d)
- [x] 2b.3 Smoke with a scratch-built binary against a scratch vault: hint present with offers, absent without
- [x] 2b.5 (D10, D11) RED: flag and hint before `items:` in a >100 KB payload, first 1500 bytes contain the hint; instruction says expected upkeep / after the request / without asking; GREEN: field order in `queryPayload`, reworded `pendingOfferCurateInstruction`, notice and nudge prefixes adjusted (commit 013f591e)
- [x] 2b.4 Follow-ups after retirement: `curate/SKILL.md` lines 5 and 28 (deleted in 4.5); consider `shim.md` only if option B (D8) is built — moot: `curate/SKILL.md` was deleted whole in 4.5 (its lines 5/28 no longer exist anywhere); option B (D8's "generic fifth shim re-entry cue") was never built — the shim.md edit that did land is D12's differently-shaped standing rule (task 3a.1), not option B, so this follow-up's own condition was not triggered

## 3. Validation (gates retirement)

- [x] 3.1 Retrieval check (no spend): phrases harvested from the kept baseline transcripts and shim-shaped forms, `--text` variants, the notice command, plus over-fire probes; table probe -> top runbook rank/provenance in `results/3.1_curate_retrieval_check.md`; first verify the scratch binary has whole-word matching
- [x] 3.2 Shim-only R arm n=3 on the explicit-ask task; bar: runbook surfaces with `trigger` provenance 3/3 and `end_state` >= 2/3 (within one trial of the skill row's 3/3); validity gate (marker-in-transcript, shadowing scan) before scoring; cost confirmed first — done 2026-09-21, `results/3.3_curate_shim_only_sonnet5.md`: runbook surfaced by `trigger` 3/3, `end_state` 3/3, bar MET; spend $1.09 (within approved $1-3)
- [x] 3.3 Build the `curate-signal` task: routine work in a vault with pending offers where engram's mid-turn notice appears; arms S / N / R; scorer for "followed the notice's instruction, surfaced the runbook, curated"
- [x] 3.4 Run `curate-signal` (arms S, N, R, n=3 each); if R fails to follow the notice, evaluate option B (generic fifth shim re-entry cue) as a separate change
  - Run done 2026-09-21 (see `dev/eval/cumulative/runbook_vs_skill/phase2/results/1.3_curate_signal_baseline_sonnet5.md`, `3.3_curate_signal_shim_only_sonnet5.md`): offers curated S 0/3, N 0/3, R 0/3.
  - 2026-09-21 rerun of R with top-of-payload stronger hint (D10, D11): smoke 0/1 curated; agent saw the hint in the preview and the reworded write warning and still declined (`results/3.4_curate_signal_stronger_hint_sonnet5.md`). n=3 not run; Joe redirected to D11's own fallback instead of the literal option-B path.
  - 2026-09-21 D11's own fallback (D12, shim standing rule) succeeded: 4/4 curated, see task 3a and `results/3.5_curate_signal_shim_standing_rule_sonnet5.md`. The literal "evaluate option B" branch was superseded by Joe's D12 decision and was not separately run; recorded as such rather than left open.
- [x] 3.5 Record results; D6 bars met or Joe redirects — D6 bars met via D11/D12 (Joe's redirect): task 3a.2 `curate-signal` R arm with the shim standing rule, `end_state` 4/4 (`results/3.5_curate_signal_shim_standing_rule_sonnet5.md`); task 3a.3 `route` dispatch-tier regression found 3/3 (`results/3.5b_route_regression_shim_standing_rule.md`)

## 3a. Shim standing rule (D12, D11's own fallback)

- [x] 3a.1 Add the generic standing rule to `agent-instructions/guidance/shim.md` (+433 bytes / 7 lines,
  after "What each returned item is for"); record D12 in design.md (reason: D9-update's in-band instruction
  still failed per `results/3.4_curate_signal_stronger_hint_sonnet5.md`); delta spec (ADDED requirement,
  `specs/guidance-runbook-follow-frame/spec.md`)
- [x] 3a.2 Smoke (n=1) `curate-signal`'s R arm with the new shim as the trial CLAUDE.md — curated (4/4
  when followed by n=3 fresh); bar `end_state` >= 2/3 MET at 4/4 (`results/3.5_curate_signal_shim_standing_rule_sonnet5.md`)
- [x] 3a.3 `route` dispatch-tier regression (`probe_phase2.py --task route --arms R --shim-only --model
  sonnet5 --n 3 --keep`) with the new shim; bar found 3/3 MET (`results/3.5b_route_regression_shim_standing_rule.md`)

## 4. Promotion and retirement (later stage, gated)

- [x] 4.1 Promote via `engram learn runbook` (real CLI fields, never a file copy) with the fixture's situation, `done_when`, `red_flags`, `triggers`; verify retrieval against the production vault (task phrase, notice command, `/curate`; casual "curated"/"accurate" do not fire) — done 2026-09-21: promoted as `1049.2026-09-21.curate-review-pending-offers` via `engram learn runbook`; `engram embed status` clean (983/983); no `red_flags` truncation marker; retrieval verified against the real ~1000-note vault, see `dev/eval/cumulative/runbook_vs_skill/phase2/results/4.1_curate_real_vault_retrieval_check.md` (trigger and semantic channels both rank 1; over-fire probes behave as designed)
- [x] 4.2 Confirm `shim.md` is imported in the real `~/.claude/CLAUDE.md` (vault note 1031a) — confirmed: `~/.claude/CLAUDE.md:27` imports `@~/.claude/engram/shim.md`; deployed `~/.claude/engram/guidance/shim.md` is byte-identical to this repo's `agent-instructions/guidance/shim.md` (carries D12's standing rule) after `engram update --with-guidance`
- [x] 4.3 Doc-surface enumeration (grep the term, synonyms, hyphenated forms and OLD echoes) with a per-file disposition list, verified independently by a fresh-context reviewer — disposition list built: `openspec/changes/curate-skill-to-runbook/enumeration.md` (grep run, exclusions, false positives, per-file table, counts). Independent fresh-context verification of this list is a SEPARATE step Joe will dispatch (not performed in this pass, per instruction)
- [x] 4.4 Update live references: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,adr}.md`, `docs/ROADMAP.md` — done per `enumeration.md`'s update/rewrite rows: CLAUDE.md, README.md (skills table + 3 prose spots), docs/GLOSSARY.md (`skill` entry), docs/architecture/c1-system-context.md (S3 row), docs/architecture/c2-containers.md (mermaid + C1 row), docs/ROADMAP.md (NOW row removed/renumbered, Provenance row added); docs/architecture/adr.md needed no edit (verified: no ADR entry asserts curate is a skill)
- [x] 4.5 Delete `agent-instructions/skills/curate/`; `engram update` (sync-as-removal cleans deployed copies); real-query check; `targ check-full` — done 2026-09-21: `git rm -r agent-instructions/skills/curate/`; `engram update --with-guidance` dry-run showed only the curate skill deletion (no other unexpected removals), then run for real: deleted from both `~/.claude/skills/curate` and `~/.pi/agent/skills/curate` (dangling links cleaned); real-query from `~` cwd surfaces `1049.2026-09-21.curate-review-pending-offers` (rank 1, direct+trigger); `targ test` (all packages ok, phase2 pytest 451 passed/2 skipped — including the now-skipping `test_curate_skill_src_is_byte_identical_to_the_live_skill`); `targ check-full` 8/9 PASS (check-uncommitted FAILs only because the change is not yet committed, expected); `openspec validate curate-skill-to-runbook --strict` and `--all --strict` (44/44) both valid

## 5. Close-out

- [x] 5.1 Sync the delta spec to `openspec/specs/vault-offer-curation/spec.md`, archive the change — done 2026-09-21: delta merged into `vault-offer-curation` and `guidance-runbook-follow-frame` main specs, change moved to `openspec/changes/archive/2026-09-21-curate-skill-to-runbook/`
- [x] 5.2 Joe's call on D5 near-case wording recorded in the spec and design (confirmed 2026-09-21: near offer discarded after folding)
