## 1. Phrasing corpus

- [ ] 1.1 Review the drafted 27-runbook × 2–3-phrasing corpus (sent to Joe for review) and apply his edits.
- [ ] 1.2 Check the reviewed corpus into `dev/eval/` as data (one entry per runbook: luhmann_id, slug, phrasings[]).

## 2. Probe script

- [ ] 2.1 Generalize `dev/eval/traps/retrieval_probe.py`'s plumbing (`rank_in_payload`, YAML-payload parsing) from its fixed 3-axis `AXIS_PHRASES`/`AXIS_TARGETS` shape to accept a per-runbook target (single slug + 2–3 phrasings) loaded from the corpus data file, instead of writing a new parser.
- [ ] 2.2 Implement the batched call: one `engram query --vault <scratch> --phrase P1 --phrase P2 [--phrase P3]` per runbook, reading the target's rank from the payload's top-level `items[]` (never `clusters[].candidate_l2s` — see `retrieval-probe-signal-fidelity`).
- [ ] 2.3 Add a regression test asserting the signal-fidelity requirement: given a synthetic payload where the target is present in `items[]` but absent from every cluster's `candidate_l2s`, the probe reports the note as surfaced (mirrors the pattern in `eval-warm-config-fidelity`'s regression test).
- [ ] 2.4 Implement the escalation path (per-phrase isolated `engram query` calls, one phrasing per call) as an opt-in mode, used only for runbooks that miss the batched check.

## 3. Vault isolation & run

- [ ] 3.1 Confirm `engram serve` write activity is quiescent (or accept the small non-atomic-copy risk and note it), then `cp -R` the real vault to a scratch directory; pass `--vault <scratch>` explicitly to every probe call. Never point the probe at the live vault or the real chunk index.
- [ ] 3.2 Pre-register the exact pass-bar tolerance below the ~0.83 matched-note-floor ceiling (open question in design.md) before running anything.
- [ ] 3.3 Run the batched probe across all 27 runbooks against the scratch vault.
- [ ] 3.4 For any runbook that misses the batched check, run the escalation (per-phrase isolated) probe on it to identify which phrasing(s) failed.

## 4. Results & spec correction

- [ ] 4.1 Compute aggregate recall@5 and per-runbook rank/miss; compare against the pre-registered pass bar.
- [ ] 4.2 Write the labeled-criteria-table result (per-runbook rank + aggregate recall@5, arms/columns with units) as a `dev/eval/LEDGER.md` row, following the existing row format.
- [ ] 4.3 For any runbook missing on all its phrasings, record its best rank and a diagnosis (situation-wording issue vs. genuine embedder miss), and list it as needing `engram resituate`.
- [ ] 4.4 Sync/archive this change (`/opsx:sync` or `/opsx:archive`) to propagate the `recall-runbook-surfacing` delta spec (candidate_l2s → items[] correction) into `openspec/specs/`.

## 5. Close-out

- [ ] 5.1 Update #738's tracking table with #734's result (green/red) and, if red, the resituate worklist.
- [ ] 5.2 Close #734 via `Fixes #734` in the closing commit (prose mention alone does not autoclose).
