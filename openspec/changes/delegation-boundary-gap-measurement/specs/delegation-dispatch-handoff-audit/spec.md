## ADDED Requirements

### Requirement: A dispatch-handoff auditor SHALL measure the recall-to-dispatch transfer gap

Any harness or script that measures whether "notes the orchestrator recalled are actually present in dispatch prompts sent to subagents" SHALL employ a two-tier approach: structural proxy (fast, full-corpus) validated against LLM-calibration on a sample of cases, per the house pattern (vault note feedback_validate_cheap_tier_framing_beats_model_tier). The structural proxy is the primary measurement; the LLM calibration confirms the proxy's accuracy before the full-corpus result is trusted.

#### Scenario: Structural proxy identifies dispatch gaps

- **WHEN** an auditor runs a structural proxy against orchestrator transcripts (extracting recalled-note basenames from recall calls, then substring/fuzzy-matching them against subsequent dispatch prompts in the same transcript)
- **THEN** the proxy flags as "missing" any recalled note that does not appear (via substring or fuzzy-match with Levenshtein distance ≤2) in the dispatch prompt
- **AND** the proxy reports the total count of fresh-context dispatches with ≥1 missing note vs. total fresh-context dispatches

A fast structural audit runs against the entire transcript corpus; false positives from paraphrasing are confirmed separately via LLM calibration.

#### Scenario: LLM calibration confirms or adjusts proxy results

- **WHEN** the structural proxy flags ≥ X cases as missing (e.g., 10–15 representative samples)
- **THEN** an LLM judge SHALL review each flagged case and determine: (a) transferred paraphrased (the note's content is present but reworded), (b) genuinely missing (no content from the note appeared), or (c) ambiguous (note only partially relevant or unclear)
- **AND** the calibration results SHALL be used to adjust the proxy's false-positive rate (reduce the count of "missing" by subtracting cases the LLM confirms were transferred paraphrased)

Detecting the gap accurately requires distinguishing intentional/implicit transfer from genuine absence.

### Requirement: Fork-dispatch exclusions SHALL be documented and tracked separately

A dispatch-handoff audit SHALL exclude fork-dispatch subagents from the read-side gap measurement's denominator and numerator, since fork dispatches structurally inherit the orchestrator's full conversation context (including recalled notes) and cannot have a handoff gap.

#### Scenario: Fork-dispatch accounting

- **WHEN** auditing a transcript corpus that includes both fork and fresh-context dispatches
- **THEN** the auditor SHALL: (a) count fresh-context dispatches with missing notes separately from fork dispatches, (b) compute the gap percentage using only fresh-context dispatches (denominator), (c) report the fork-dispatch count explicitly in the results

Counting forks would deflate the measured gap and understate the addressable ceiling for mechanisms (like Phase 2's dispatch injection) that target only fresh-context dispatches.

### Requirement: A dispatch-handoff audit SHALL pre-register the go/no-go gate before running

The auditor's success threshold (e.g., "≤30% of fresh-context dispatches have missing notes") SHALL be pre-registered before any LLM calibration or full-corpus measurement is performed, per the house discipline (vault note feedback_measure_ceiling_and_pin_fire_unit_before_do_x_more).

#### Scenario: Pre-registered gate

- **WHEN** an auditor is configured to measure the dispatch-handoff gap
- **THEN** before any proxy runs or LLM calls, the implementer SHALL set an explicit threshold (e.g., "This audit PASSES if ≤ X% of fresh-context dispatches lack at least one relevant recalled note")
- **AND** document the rationale for the chosen threshold

The gate's outcome (green/red for Phase 2 unblock) depends on the threshold, and it must not be chosen *after* seeing the numbers.

