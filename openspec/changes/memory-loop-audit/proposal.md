## Why

#739's Phase 1 was scoped to subagent transcripts and two narrow instruments. Joe reversed that scope in review (thread `5974d361…`, 2026-08-30, on this change's own proposal): audit **main transcripts as well** — "otherwise we don't know how well the main agent is doing at recalling relevant memories" — with a per-moment rubric spanning the entire memory loop, successes included. ADR-0028 (accepted 2026-08-30) generalizes the reversal: engram is the learning/reinforcement layer for statically-weighted LLMs, and this audit is its **outcome labeler** — the one missing stage of the experience → label → update loop. This change builds the audit's **observe mode** as a standing instrument, runs the first measurement, and produces the pre-registered gate that #739 Phase 2 (dispatch-side injection + completion-report LESSONS harvest) is blocked on. It restores the 2026-06-28 mining scope (main + subagent, cross-repo) and widens the rubric from failure-coverage to the full funnel.

## What Changes

- **Mechanical extractor** (`dev/eval/audit/`, Python, unit-tested): parses Claude Code and Pi transcripts into per-transcript structured event records — recall invocations (phrases; payload items with path/kind/provenance/rank), dispatch events (prompt text, fork vs fresh vs workflow, injected note refs), learn/write-memory skill calls, `engram learn/amend/activate` invocations, completion reports. Full-corpus, no LLM calls. Spec'd implementation-agnostic so the credit-loop change can promote it into an `engram audit` subcommand without re-deriving requirements (ADR-0028 D5).
- **Semantic auditor** (LLM, windowed): detects moments (failure / correction / rework / **success** / **dispatch**), answers the judgment columns of ADR-0028 D2's funnel, and joins with the mechanical records into per-moment JSONL. Semantic detection, over-match→prune, miss-rate gated, calibrated on hand labels with a scorer hand-read before any paid full run.
- **Existed-at-T oracle**: per moment, derives its own query phrases and judges "did a relevant memory exist at time T" against the vault as of T — notes via a read-only `git worktree` checkout at the last commit ≤ T (DERIVED; `created ≤ T` filter fallback labeled ESTIMATE), chunks via `IngestedAt ≤ T`. Isolated per #708; the live vault is never written.
- **Credit-record emission**: draft (note ref, situation, outcome, evidence pointer) records emitted as data files. Emission only — consuming them (judged amend/supersede/resituate/prune, experience entries) is the follow-on credit-loop change.
- **First measurement + report**: the funnel table with pinned fire-units and DERIVED/ESTIMATE labels; the #739 slice; a gap harvest (residual moments → remedies laddered per vault note 495: vault notes by default, repo issues only where a note cannot reach per note 803); `dev/eval/LEDGER.md` rows plus the initial coverage-map index section.
- **Phase 2 gate**: thresholds pre-registered before the full run; GREEN/YELLOW/RED decision recorded on #739.

**Out of scope:** any vault write by the audit (credit records are data); experience records; replay and mine modes; the `engram audit` binary subcommand; any production code, skill, or guidance edit; #739 Phase 2 itself.

## Capabilities

### New Capabilities

- `memory-loop-audit`: the requirements any instrument measuring the memory loop on real transcripts must meet — semantic moment detection including successes and dispatch moments; funnel-complete per-moment records with pinned fire-units and DERIVED/ESTIMATE labels; the existed-at-T oracle discipline; injected-counts-as-searched-and-surfaced; credit-record emission (emission-only); pre-registered gates; LEDGER + coverage-map recording.

This change's two previously-proposed capabilities (`subagent-correction-detection`, `delegation-dispatch-handoff-audit` — never synced to `openspec/specs/`) are absorbed into `memory-loop-audit`: their detection semantics, miss-rate gating, and two-tier proxy+calibration method survive as requirements there. The old fork-exclusion rule (prior D3) is replaced by injected-counts-as-surfaced, which covers forks structurally.

## Impact

- New reusable instrument under `dev/eval/audit/`; no production code, skill, or guidance changes.
- `dev/eval/LEDGER.md`: new dated rows + a new coverage-map index section (all existing rows and anchors preserved).
- Read-only against live transcripts and the live vault (oracle via read-only git worktrees; #708 isolation for every engram invocation).
- On completion: #739 body rewritten as the current brief with Phase 2 blocked on this gate (vault note 819); follow-on issues filed for the credit loop and replay+mine changes; #718 narrowed; #742 annotated — all per ADR-0028 Consequences.
- Authority for the loop design: `docs/architecture/adr.md` ADR-0028.
