## Context

#739 identified a structural gap at the delegation boundary: the delegate-everything doctrine (delegate.md) dispatches work into subagent contexts where orchestrator-side memory operations (recall surfacing runbooks, learn harvesting corrections) cannot reach. ADR-0027 (accepted 2026-08-30) categorized this as a gap-map item under "multi-agent memory routing" — the highest-value unowned item — and Joe's decision comment resolved the approach as "measure → inject": Phase 1 measures the gap's size and reachability, Phase 2 (gated by Phase 1) injects fixes. This change executes Phase 1 only.

**Relevant existing state:**

- `engram ingest --auto` already sweeps subagent transcripts (`~/.claude/projects/<mangled-path>/**/subagents/**`) — episodic capture across the boundary exists, but consolidation is pull-triggered (ADR-0027).
- `ingest --auto` feeds a unified embedding ingestion and vector search (no distinction between orchestrator sessions and subagent sessions in the vault itself).
- `recall` skill fires in orchestrator context only; dispatch handoff is orchestrator-manual (copy relevant notes into the dispatch prompt).
- `learn` skill fires where the correction happens; orchestrator-side learns occur inside the orchestrator context; subagent-side learns occur inside a subagent context that is discarded at completion.
- `route` skill records dispatch metadata as fact notes (agent type, model, effort, outcome) — the one memory loop already crossing the boundary.
- Joe's operational decision (vault note 855, explore-mode walkthrough, 2026-08-30, D4): Phase 2's fix (dispatch-side injection + completion-report harvest) belongs in `delegate.md` ambient guidance, not in specific skill templates, because the rule must apply to every delegation act regardless of which skill (if any) mediated it — not in route/SKILL.md or please/SKILL.md.
- Joe's Option B decision (2026-08-30, D7): Full subagent memory loops (bidirectional recall + learn across the boundary) is PARKED, not rejected. Scoping evidence: route dispatch record 429 shows gate reviewers already self-recall, reducing the immediate pressure for subagent-side recall surfacing. Revisit condition: if measured ceiling on Phase 1's read-side gap disappoints, or if long-running subagents become the norm, the full-loop option re-enters scope. Phase 1's read-side focus (fresh-context builder/executor dispatches) excludes reviewers for this cycle.

**Measurement strategy per design decisions (D1–D6):**

- **D1 — Write-side classifier method:** Re-derive a semantic classifier (purpose-built for subagent-specific detection), informed by but not bound to the deleted 2026-06-28 failure-eval methodology. Detect semantically (68% of mined failures were subtle; word-match missed 2/3), over-match then prune (to reduce false positives), gate on miss-rate (false-negatives) not false-positive rate, keep injection-point/context analysis open (uncovered cues are high-value findings). Classify each finding per vault runbook 824 (addressable / capture-ceiling / application-or-ranking-miss) and label every derived number DERIVED vs ESTIMATE, with fire-unit pinned before computing ratios.

- **D2 — Read-side gap measurement:** Structural proxy (extract note basenames from orchestrator `recall` calls, substring/fuzzy-match them against subsequent dispatch prompts in the same transcript, flag as missing if not found) + LLM-calibration sample (judge a proxy-flagged sample to confirm paraphrased handoffs aren't false-positives). Mirrors the house pattern of cheap-tier validation (vault note feedback_validate_cheap_tier_framing_beats_model_tier).

- **D3 — Fork-dispatch scope:** Exclude fork dispatches from the read-side sample denominator/numerator (they structurally inherit the orchestrator's full context including recalled notes). Write-side (D1) is not affected — subagent transcripts of both fork and fresh-context types can surface correction moments worth mining.

- **D4 — Phase 2 injection site context (informational only; not executed in this change):** Phase 2's two clauses (dispatch-side note handoff, completion-report lessons harvest) live in `delegate.md` as new prose rules, not in route/please/SKILL.md templates. This Phase 1 change must NOT modify delegate.md, learn.md, recall.md, route/SKILL.md, or please/SKILL.md — measurement only.

- **D5 — Cross-repo corpus scope:** delegate.md/learn.md/recall.md are shared guidance across all of Joe's repos (Claude Code + Pi), not engram-specific. A prior lesson (vault note feedback_verify_mined_corpus_is_representative) flagged single-repo retention bias (engram-only sample inverted real cross-repo picture 46%→41% on a trigger). D1's classifier and D2's proxy must handle both Claude Code (`~/.claude/projects/-<mangled-path>/**/subagents/**`) and Pi (`~/.pi/agent/*/subagent/**` or analogous — verify against live paths in tasks) transcript locations.

- **D6 — OpenSpec tracking:** Mirroring #734→#737's structure — this lightweight Phase 1 measurement-only change gates a SEPARATE Phase 2 change to be filed later, once the addressable ceiling is known. Tasks.md must end with an explicit go/no-go decision task (analogous to #734's "No runbook needs engram resituate; #737 is unblocked").

## Goals / Non-Goals

**Goals:**

- Measure, per write-side (subagent corrections never captured), whether the orchestrator-side `learn` mechanism reaches the moments; classify findings per addressable/capture-ceiling/application so Phase 2's reach scope is understood before shipping.
- Measure, per read-side (dispatch handoff gaps), how often orchestrator-recalled notes relevant to a dispatched unit are absent from the dispatch prompt; distinguish "not handoff-transferred" from "intentionally omitted" and "paraphrase-transferred."
- Produce a pre-registered, labeled-table result following the house criteria table format (fire-unit pinned, DERIVED/ESTIMATE-labeled, units named).
- Produce a `dev/eval/LEDGER.md` row recording the findings and the go/no-go gate for Phase 2.
- Produce two reusable tools (write-side classifier, read-side proxy auditor) that survive beyond this measurement as inputs to Phase 2 design and future audits.

**Non-Goals:**

- Does NOT implement Phase 2 (dispatch-side injection, completion-report templates, delegate.md edits) — that is a separate, GATED change that only proceeds once Phase 1 is complete.
- Does NOT fire `/learn` inside subagents or orchestrators as part of Phase 1 — the measurement runs on already-ingested transcripts (snapshots), not live sessions.
- Does NOT change `engram query` behavior or the vault structure — read-only measurement against live transcripts.
- Does NOT assume cross-repo transcript paths without verification — D5 requires discovering and documenting actual paths in tasks.
- Does NOT propose Phase 2 mechanisms — only the Phase 1 gap measurements and the gate decision.

## Decisions

**Re-derive the write-side classifier from scratch, not reuse the 2026-06-28 script.** The 2026-06-28 classifier (`docs/design/2026-06-28-failure-eval-material.md`) was deleted post-extraction; the method is recoverable from git history but the tool no longer exists. Subagent-specific criteria (corrections visible *within* a subagent transcript, never captured by the orchestrator's learn cue) are narrower than the general "failure" scope of the old classifier. Decision: re-derive fresh from the design principle (semantic detection, over-match/prune, miss-rate gate, injection-point analysis open), apply it to the subagent-specific problem, and ship the new classifier as a reusable artifact for Phase 2 design and future audits.

**Structural proxy first, LLM calibration sample only for verification.** The full-corpus read-side audit (checking every dispatch prompt for every recalled note) is expensive if done via LLM. A cheap structural proxy (regex/fuzzy-match on basenames) runs against the entire transcript corpus in seconds; LLM-calibration then checks a representative sample of the proxy's "missing" flags to confirm the proxy isn't silently inflating or deflating the gap (vault note feedback_validate_cheap_tier_framing_beats_model_tier). Alternative considered: full-LLM audit — rejected, the house pattern recommends cheap-then-validated-sample for large corpora.

**Fork-dispatch exclusion from read-side denominator/numerator.** D3 is scoped specifically to the read-side (D2) gap measurement: fork dispatches inherit the orchestrator's full conversation context structurally, so they cannot have a "missing notes" gap — the notes are already in-context. Counting forks would deflate the measured gap and understate the addressable ceiling for fresh-context dispatches (the ones Phase 2's injection mechanism actually targets). The write-side classifier (D1) is not scoped by D3 — subagent transcripts include both fork and fresh-context dispatches, both can surface correction moments.

**Cross-repo corpus over engram-only.** The delegate.md/learn.md guidance applies to all of Joe's repos, not engram-specific. A prior lesson (vault note feedback_verify_mined_corpus_is_representative) demonstrated single-repo retention bias can invert findings. D1 and D2 must traverse Claude Code and Pi transcript paths (or document if Pi paths are not available). Tasks.md must verify actual paths before collection begins.

**OpenSpec tracking via full change, not LEDGER-only entry.** The write-side classifier and read-side proxy auditor are durable reusable artifacts that survive Phase 1 and feed into Phase 2 design and future improvements. An alternative considered: if these outputs remained experimental or temporary, a LEDGER-only entry would have sufficed. Decision: track as a full OpenSpec change to elevate the two capabilities to spec-tracked status (subagent-correction-detection, delegation-dispatch-handoff-audit) and propagate them into openspec/specs/ for cross-tool visibility and future integration.

**Pre-registered addressable ceiling, fire-unit pinned before any ratio.** Following the house discipline (vault note feedback_measure_ceiling_and_pin_fire_unit_before_do_x_more), the Phase 1 report must name the fire-unit (e.g., "per-dispatch," "per-correction"), compute addressable findings at that unit, and state the go/no-go gate before running anything — never inferred from the numbers afterward.

## Risks / Trade-offs

- [Semantic classifier precision is difficult to measure; over/under-matching silently] → the prune-then-LLM-calibration-sample pattern (D2 precedent) is applied to D1: run the classifier, measure its false-negative rate on a hand-labeled sample before trusting the full-corpus result.
- [Cross-repo path discovery is manual; paths may have changed] → Tasks.md includes a path-verification task before collection begins; if Pi paths are unavailable or different, the scope falls back to Claude Code only with documented rationale.
- [Dispatch prompt changes/obfuscation could make string matching brittle] → the structural proxy uses both substring and fuzzy-match (Levenshtein or similar), with a calibration sample catching any systematic over/under-counting before the full-corpus result is trusted.
- [Fork-dispatch exclusion (D3) removes signal from the denominator; the read-side gap could be smaller than measured] → acknowledged as a scope decision, not a bug. Fork dispatches don't leak because they inherit context; fresh-context dispatches do. Reporting both separately (denominator notes: "X/Y fresh-context dispatches, excluding Z forks") preserves visibility.

## Open Questions

- Exact cross-repo path for Pi subagent transcripts (if any exist) — needs discovery in tasks before collection.
- Exact pre-registered go/no-go gate threshold (e.g., "≤ 20% of fresh-context dispatches have missing notes" vs. a different bar) — set before running the auditor.
- Whether the write-side classifier should process *all* transcripts or only a sample for Phase 1 (to constrain cost) — scope decision for tasks.
- Exact false-negative tolerance for the write-side classifier before it is trusted on the full corpus — parameterized in tasks, calibrated on a hand-labeled sample.

