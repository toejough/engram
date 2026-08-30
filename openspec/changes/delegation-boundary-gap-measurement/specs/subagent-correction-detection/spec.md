## ADDED Requirements

### Requirement: A subagent-correction classifier SHALL detect correction moments semantically

Any harness or script that identifies "a correction-shaped moment inside a subagent transcript that never produced a vault note" (for the purpose of Phase 1's write-side gap measurement or Phase 2's correction harvesting) SHALL do so by detecting the semantic signal of correction/feedback/rejection/self-caught-error, not by keyword or word-match heuristics. Semantic detection rates must be ≥68% on subtle corrections (per vault note feedback_semantic_overmatch_false_negative_open_injection); word-match-based approaches miss ~2/3 of subtle cases and are therefore insufficient as a primary detection method.

#### Scenario: Semantic detection of feedback-shaped corrections

- **WHEN** a subagent transcript contains a feedback/rejection/self-caught-error moment (e.g., "Your approach is wrong, do X instead"; or the subagent's own error recovery narrative; or a gate reviewer's rejection text)
- **THEN** the classifier reports the moment as a detected correction, regardless of whether it matches a pre-defined keyword list

Approximately 68% of detectable corrections in transcripts are subtle and would be missed by word-match alone.

### Requirement: A subagent-correction classifier SHALL gate on miss-rate, not false-positive rate

The classifier's quality bar SHALL be measured and gated on false-negative rate (missed corrections) not false-positive rate (over-matched non-corrections). The miss-rate SHALL be calibrated and pre-registered before applying the classifier to the full corpus.

#### Scenario: Miss-rate pre-registration

- **WHEN** a classifier is applied to a corpus of subagent transcripts
- **THEN** the implementer SHALL first hand-label a representative sample (≥10–20 transcripts, stratified by project and dispatch type) and measure the classifier's false-negative rate (% of hand-labeled corrections missed by the classifier)
- **AND** SHALL pre-register a miss-rate tolerance (go/no-go threshold) before running on the full corpus

Capturing addressable lessons is the priority (miss-rate is the loss); false positives (over-matching non-corrections) are cheaper to prune later.

### Requirement: A subagent-correction classifier SHALL preserve injection-point and context analysis

The classifier output SHALL include, for each detected correction: the timestamp/transcript location, the correction narrative text, the type of correction (feedback vs. self-caught vs. decision-point), and the injection point (when during the task did the correction occur). This context is high-value for Phase 2 design and future improvements.

#### Scenario: Injection-point preservation

- **WHEN** the classifier reports a detected correction
- **THEN** it SHALL include the transcript path, timestamp, full correction narrative, and phase/stage in the task when the correction occurred

Uncovered cues about where corrections cluster inform Phase 2's mechanism placement (e.g., "most corrections happen at mid-task decision moments" shapes where dispatch-injected runbooks should be positioned).

