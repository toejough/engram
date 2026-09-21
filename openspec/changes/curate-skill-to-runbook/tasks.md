## Status

Stage 1 (this stage): sections 1, 2, 2a, 2b and task 3.1 (no paid runs). Sections 3.2-3.5, 4 and 5 are later
stages (paid runs need cost confirmation; retirement needs the gate).

## 1. Eval baseline (done in the previous stage)

- [x] 1.1 Curate eval fixture (fictional beekeeping vault, three offers covered/near/absent), harness `vault_template` support, skill and bare baselines (`results/1.3_curate_baseline_sonnet5.md`; skill row end_state 3/3, followed_all 2/3; bare 0/3, 0/3); commit b55080f1

## 2. Conversion

- [x] 2.1 Tag every `curate/SKILL.md` section carry / drop-as-shim-floor / adapted, place the seven red-flag rows, and record the deltas in `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/curate-conversion-fidelity-report.md`
- [x] 2.2 Write the top runbook `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskCurate/Curate-R/vault/<luhmann>.<date>.curate-review-pending-offers.md` (type runbook, tier, process-shaped `situation`, `done_when`, `red_flags` <= 1200 bytes, `triggers`, body = SKILL.md steps verbatim where possible; keep host-local-only and never-hand-off-to-write-memory; keep the `grep -l '^pending: true$'` scan)
- [x] 2.3 Verify `red_flags` bytes against the real truncation budget (`internal/cli/redflags_truncation.go`, `engram show --vault <fixture>` prints no omission marker)
- [x] 2.4 Rebuild sidecars: `engram embed apply --vault <fixture vault> --all`, 0 stale
- [x] 2.5 Wire `carrier_r_src` in `fixtures/curate/task.json`; `probe_phase2.py --task curate --arms R --shim-only --setup-only` builds (no spend) and the trial vault contains the runbook plus the fixture offers
- [x] 2.6 Run the phase2 tests (`python3 -m pytest dev/eval/cumulative/runbook_vs_skill/phase2/test_probe_phase2.py`)
- [ ] 2.7 Fresh-context reviewer checks the fidelity report against `SKILL.md` for lost content

## 2a. Notice reword (D4)

- [x] 2a.1 RED: change `TestWriteUpdateReport_PendingOfferHint` (`internal/cli/update_test.go`) and add an `offer_test.go` case for the write nudge to assert the trigger query (`engram query --text "curate pending offers"`) and no skill naming; `targ test` fails
- [x] 2a.2 GREEN: reword `pendingOfferUpdateNotice` and `pendingOfferWriteNudge` in `internal/cli/offer.go`; `targ test` passes
- [x] 2a.3 Reword "curation skill" / "offer-curation skill" to "curation runbook" in comments and flag descriptions (`amend.go`, `learn.go`, `update.go`, `serve.go`, `targets.go`, tests)
- [x] 2a.4 `targ check-full` green; smoke: run the reworded notice through a scratch-home `engram update` (or show the unit-test evidence); never touch the real vault

## 2b. Payload hint (D9)

- [x] 2b.1 RED: tests for `pending_offers_hint` present iff `pending_offers` is true, equal to the shared constant, byte-identical payload without offers, merged query, served round-trip, and notices embedding the same constant; `targ test` fails
- [x] 2b.2 GREEN: `pendingOfferCurateInstruction` shared constant, `queryPayload.PendingOffersHint`, set in `runQuery` and `mergeQueryPayloads`; `targ test` passes (commit b050db1d)
- [x] 2b.3 Smoke with a scratch-built binary against a scratch vault: hint present with offers, absent without
- [ ] 2b.4 Follow-ups after retirement: `curate/SKILL.md` lines 5 and 28 (deleted in 4.5); consider `shim.md` only if option B (D8) is built

## 3. Validation (gates retirement)

- [x] 3.1 Retrieval check (no spend): phrases harvested from the kept baseline transcripts and shim-shaped forms, `--text` variants, the notice command, plus over-fire probes; table probe -> top runbook rank/provenance in `results/3.1_curate_retrieval_check.md`; first verify the scratch binary has whole-word matching
- [ ] 3.2 Shim-only R arm n=3 on the explicit-ask task; bar: runbook surfaces with `trigger` provenance 3/3 and `end_state` >= 2/3 (within one trial of the skill row's 3/3); validity gate (marker-in-transcript, shadowing scan) before scoring; cost confirmed first
- [ ] 3.3 Build the `curate-signal` task: routine work in a vault with pending offers where engram's mid-turn notice appears; arms S / N / R; scorer for "followed the notice's instruction, surfaced the runbook, curated"
- [ ] 3.4 Run `curate-signal` (arms S, N, R, n=3 each); if R fails to follow the notice, evaluate option B (generic fifth shim re-entry cue) as a separate change
- [ ] 3.5 Record results; D6 bars met or Joe redirects

## 4. Promotion and retirement (later stage, gated)

- [ ] 4.1 Promote via `engram learn runbook` (real CLI fields, never a file copy) with the fixture's situation, `done_when`, `red_flags`, `triggers`; verify retrieval against the production vault (task phrase, notice command, `/curate`; casual "curated"/"accurate" do not fire)
- [ ] 4.2 Confirm `shim.md` is imported in the real `~/.claude/CLAUDE.md` (vault note 1031a)
- [ ] 4.3 Doc-surface enumeration (grep the term, synonyms, hyphenated forms and OLD echoes) with a per-file disposition list, verified independently by a fresh-context reviewer
- [ ] 4.4 Update live references: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,adr}.md`, `docs/ROADMAP.md`
- [ ] 4.5 Delete `agent-instructions/skills/curate/`; `engram update` (sync-as-removal cleans deployed copies); real-query check; `targ check-full`

## 5. Close-out

- [ ] 5.1 Sync the delta spec to `openspec/specs/vault-offer-curation/spec.md`, archive the change
- [x] 5.2 Joe's call on D5 near-case wording recorded in the spec and design (confirmed 2026-09-21: near offer discarded after folding)
