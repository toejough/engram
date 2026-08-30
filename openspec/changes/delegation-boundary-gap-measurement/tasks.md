## 1. Corpus discovery and scope verification

- [ ] 1.1 Verify cross-repo transcript paths are accessible and available. (a) Check `~/.claude/projects/*/subagents/` directories exist for Claude Code (engram + any other active repos). (b) Search for Pi subagent transcripts under `~/.pi/` or analogous paths (verify against live system, not assumption); document the actual path structure. (c) For each discovered `subagents/` directory, locate its parent orchestrator/session transcript file (sibling of the `subagents/` directory) — these are needed for task 3.1's read-side auditor to extract recall calls and dispatch prompts. Document the discovered orchestrator transcript paths alongside subagent paths. (d) If Pi paths are unavailable or absent, scope tasks 1.2–3.x to Claude Code only with documented rationale.
- [ ] 1.2 List and categorize all subagent transcript files across repos by dispatch type (fork vs. fresh-context) and project. Document the sample size (total count, broken down by repo and dispatch type). Also enumerate the corresponding orchestrator-level parent transcript files discovered in 1.1(c).

## 2. Write-side classifier (re-derive + apply)

- [ ] 2.1 Design a fresh semantic classifier for "a correction-shaped moment in a subagent transcript that never produced a vault note." Follow the design principle (D1): detect semantically (not by keyword/word-match — aim for 68%+ catch on subtle failures), over-match aggressively then prune to reduce false positives, gate on miss-rate (false-negatives) not false-positive rate, and keep injection-point/context open for high-value finding discovery. Document the classifier's detection heuristics (e.g., patterns in feedback/rejection language, self-caught error signals, decision-point descriptions) and the rationale for each.

- [ ] 2.2 Pre-register a false-negative tolerance (the go/no-go threshold for trusting the classifier's full-corpus result) before running on the full sample. Hand-label a representative sample (e.g., 10–20 subagent transcripts, diverse by repo and dispatch type) and measure the classifier's precision/recall against the hand-labels to set the tolerance. Produces: calibration spreadsheet (CSV/JSON format) with columns: transcript_id, hand_labeled_correction_count, classifier_detected_count, false_negative_rate, pre_registered_threshold.

- [ ] 2.3 Apply the classifier to all subagent transcripts (or a pre-decided sample, scoped in 1.2) across the corpus. For each detected correction moment, extract: (a) the transcript path and timestamp, (b) the correction narrative (feedback/self-caught error), (c) whether a corresponding vault note was found (scan the transcript for `/learn` commands or ingest-time note hashes), (d) the injection point (when during the task did the correction occur).

- [ ] 2.4 Classify each detected correction moment per vault runbook 824 (addressable / capture-ceiling / application-or-ranking-miss). Capture the rationale for each classification (e.g., "capture-ceiling: no clear or confirmable lesson existed to capture, even though feedback occurred" vs. "addressable: a clear correction moment existed and no structural barrier prevented firing `/learn`, but it wasn't fired").

- [ ] 2.5 Compute aggregate statistics per classification: (a) count and % of addressable findings (reachable by Phase 2's mechanism), (b) count and % of capture-ceiling findings (unreachable even with Phase 2's proposed scope), (c) count and % of application-or-ranking-miss findings (orthogonal to the memory-routing gap). Pin the fire-unit explicitly (e.g., "per-subagent-transcript" vs. "per-correction-moment") before computing any ratio. Label all derived numbers as DERIVED, estimates/projections as ESTIMATE.

## 3. Read-side auditor (proxy + LLM calibration)

- [ ] 3.1 Build a structural proxy script (read-only, no LLM calls): (a) for each transcript, find all orchestrator `recall` calls (e.g., lines matching "/recall glance" or equivalent LLM invocation tracing a recall in agent context); (b) extract the recalled note basenames/luhmann IDs from the recall output (e.g., parse the vault note list payload); (c) find all subsequent dispatch calls in the same transcript (route/route_agent invocations or equivalent subagent launch events); (d) for each dispatch, check whether the recalled note basenames appear (via substring match and fuzzy-match, e.g., Levenshtein distance ≤2 on note slugs) in the dispatch prompt text; (e) flag as "missing" if not found. Document the matching heuristics and tolerance thresholds. Produces: Python script (`dev/eval/dispatch_handoff_auditor.py`) with docstrings, accepting a transcript corpus path and emitting a CSV report (columns: transcript_id, dispatch_id, recalled_notes[], dispatch_note_coverage, missing_notes[], dispatch_type).

- [ ] 3.2 Apply D3 (fork-dispatch exclusion): filter the proxy results to exclude any subagent dispatch with `subagent_type: "fork"` from the read-side gap count. Count separately: (a) fresh-context dispatches with missing notes (numerator), (b) total fresh-context dispatches (denominator), (c) note the fork-dispatch count excluded for transparency.

- [ ] 3.3 Pre-register a go/no-go gate threshold for read-side gap (e.g., "≤ 30% of fresh-context dispatches lack at least one relevant recalled note" or a different criterion) before running the LLM calibration. Set before running anything, per house discipline.

- [ ] 3.4 Run the structural proxy across all transcripts, producing a flagged-as-missing candidate list. Estimate the full-corpus gap (e.g., "X% of fresh-context dispatches) using the proxy's raw results.

- [ ] 3.5 LLM-calibration sample: sample the proxy's flagged-as-missing cases (e.g., 10–15 cases, stratified by repo and phrasing style) and have an LLM-graded judge review each one — did the dispatch prompt actually contain (paraphrased or verbatim) the relevant recalled note content, or was it genuinely absent? Document the calibration sample size, criteria for "paraphrase counts as transferred," and the LLM judge's verdict on each case (e.g., "transferred paraphrased," "genuinely missing," "ambiguous — note only partially relevant").

- [ ] 3.6 Recompute the read-side gap using the calibration results to adjust the proxy's false-positive rate (over-flagging from paraphrases). State the adjusted gap estimate and the confidence interval (based on calibration sample size and agreement rate).

## 4. Results and LEDGER entry

- [ ] 4.1 Compute the Phase 1 report: a labeled-criteria table with columns for each metric (write-side addressable count, write-side % of total, read-side gap %, read-side confidence, fire-unit for each), rows for "DERIVED (measured)" and "ESTIMATE (projected," unit labels explicit. Include a one-paragraph narrative for each major finding (e.g., "write-side addressable findings: N corrections were detected in subagent transcripts but never captured by orchestrator-side learn, of which A% are addressable by Phase 2's proposed dispatch/harvest mechanism").

- [ ] 4.2 Write a `dev/eval/LEDGER.md` row (using the existing format as precedent, e.g., the "734-runbook-retrieval-probe" entry) recording: (a) the write-side and read-side findings (counts, %, addressable ceiling), (b) the fire-unit pinned for each axis, (c) the vintage date (2026-08-30), (d) references to the classifier script and proxy auditor in `dev/eval/`, (e) raw data paths (transcript corpus location, calibration sample results). The row's verdict should state the go/no-go gate decision (see task 5.1).

- [ ] 4.3 Document the classifier script and proxy auditor as reusable artifacts under `dev/eval/` (e.g., `dev/eval/subagent_correction_classifier.py`, `dev/eval/dispatch_handoff_auditor.py`) with docstrings explaining the methodology, detection heuristics, output format, and usage (so Phase 2 design can build on them without re-deriving).

- [ ] 4.4 Sync/archive this change (`/opsx:sync` or `openspec archive delegation-boundary-gap-measurement`) to propagate the two new capability specs (`subagent-correction-detection`, `delegation-dispatch-handoff-audit`) into `openspec/specs/`, mirroring the precedent in runbook-retrieval-probe's task 4.4.

## 5. Go/no-go gate for Phase 2

- [ ] 5.1 State the explicit go/no-go decision for opening Phase 2: (a) if the write-side addressable count / capture-ceiling ratio shows Phase 2's dispatch-inject and completion-harvest mechanism can reach ≥ [pre-registered threshold]% of lost lessons (analogous to #734's "green = unblocks #737"), state "GREEN: Phase 2 is unblocked"; (b) if the read-side gap or write-side ceiling disappoints, state "YELLOW/RED with rationale: Phase 2 scope requires revision before proceeding" (e.g., expanding to cover capture-ceiling moments, adding a different mechanism for application-miss findings, etc.). (c) Include a sentence summarizing the decision's dependency on these Phase 1 findings and any revisit conditions (e.g., if the corpus grows materially, re-measure before Phase 2 ships).

- [ ] 5.2 Update #739 with Phase 1 results (linked LEDGER row, go/no-go gate, any revised Phase 2 scope if needed) and link to the Phase 2 issue to be filed once Phase 1 is accepted (do not file Phase 2 yet; leave its filing contingent on Phase 1's acceptance and gate).

