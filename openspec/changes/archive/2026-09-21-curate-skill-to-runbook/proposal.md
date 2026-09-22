## Why

engram's shim-and-runbook architecture replaces agent-instruction skills with runbook notes surfaced by
`engram query`. `route` (2026-09-19) and `please` (#758) are the first two conversions; `curate` is next
(#759). `agent-instructions/skills/curate/SKILL.md` is small (83 lines, 5,724 bytes) and fits one runbook
note, so it is the cheapest conversion and the first one that exercises **automatic** firing: a skill's
`description` used to make the harness offer `curate` when engram itself printed a pending-offers signal
(query payload flag, `engram update` notice, write-path nudge, served-write acceptance). A runbook has no
description the harness reads, so those signals must carry the trigger themselves.

The `curate` eval already exists (built in the previous stage): a fictional beekeeping vault with three
pending offers (one covered, one near, one absent). Baseline, sonnet5, n=3: skill row `end_state` 3/3,
`followed_all` 2/3; bare row 0/3 and 0/3 (the bare agent clears the near offer instead of folding it in and
discarding it).

## What Changes

- Convert `curate/SKILL.md` into ONE top runbook note `curate` (D1): body is the four steps verbatim where
  possible, `red_flags` from the 7-row table (at most 1200 bytes), `triggers` (D3). Content the shim already
  states uniformly is dropped and recorded in a conversion-fidelity report.
- Rewrite the pending-offer notices in `internal/cli/offer.go` (`engram update` notice and write-path
  nudge) to tell the agent to run `engram query --text "curate pending offers" --phrase ...`, a trigger
  phrase, instead of naming a skill (D4). Reword "curation skill" / "offer-curation skill" to "curation
  runbook" in user-facing help text and in comments (`amend.go`, `learn.go`, `update.go`, `serve.go`,
  tests).
- Reconcile `vault-offer-curation`: the spec says the marker is cleared "in every case", but the skill (and
  the eval end states) DISCARD covered and near offers and clear the marker only for absent ones (D5).
  Name the carrier as "the curate runbook".
- Build a runbook fixture (`Curate-R`), a conversion-fidelity report, a retrieval check (harvested real
  agent phrases plus over-fire probes), then a shim-only R arm on the explicit-ask task and a second eval
  task `curate-signal` that tests the notice-driven path with arms S (skill) / N (bare) / R (runbook)
  (D6).
- Retire `agent-instructions/skills/curate/SKILL.md` and promote the runbook to the production vault only
  in a LATER stage, behind the three-part gate plus an independently reviewed doc-surface enumeration
  (D7). This change's tasks list those steps but they are not executed in the conversion stage.

## Capabilities

### New Capabilities

(none, applies the existing runbook/follow-frame and lexical-trigger capabilities to a new carrier)

### Modified Capabilities

- `vault-offer-curation`: (a) the curation requirement names the curate runbook as its carrier and states
  the per-outcome behavior that matches `engram amend` and the skill (covered and near are discarded;
  absent clears the marker); (b) the surfacing requirement states that the update notice and the
  write-path nudge carry a copy-pasteable trigger query for the curate runbook.

## Impact

- **Code**: `internal/cli/offer.go` (two constants), comments and flag descriptions in `amend.go`,
  `learn.go`, `update.go`, `serve.go`, `targets.go`, tests in `update_test.go`, `offer_test.go`,
  `serve_client_test.go`. `--discard` / `--clear-pending` flag descriptions stay behaviorally identical.
- **Eval**: `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskCurate/Curate-R/`,
  `encodings/curate-conversion-fidelity-report.md`, `results/3.1_curate_retrieval_check.md`, later
  `curate-signal` fixture and run results.
- **Later stage (not in this stage)**: delete `agent-instructions/skills/curate/`, update live docs
  (`CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,adr}.md`,
  `docs/ROADMAP.md`), promote via `engram learn runbook`, `engram update`.
- **Shim**: unchanged. The shim's four re-entry cues are not keyed on tool-output notices, so mid-turn
  firing rides on the notice text (D8); a generic fifth re-entry cue is a possible later shim change.
- **Issues**: relates to #759; no closing keyword in this change's commits.
