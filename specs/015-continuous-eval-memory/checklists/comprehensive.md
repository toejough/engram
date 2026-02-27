# Comprehensive Requirements Quality Checklist: Continuous Evaluation Memory Pipeline

**Purpose**: Formal-gate audit validating requirement completeness, clarity, consistency, and spec-to-contract alignment across all 6 user stories, 19 FRs, and 6 contracts
**Created**: 2026-02-20
**Feature**: [spec.md](../spec.md)
**Depth**: Formal gate (~50 items)
**Focus**: All domains — data model, LLM integration, user-facing behavior, cross-cutting concerns
**Scope**: spec.md + contracts/ + data-model.md cross-validation

## Requirement Completeness

- [x] CHK001 - Are implicit signal types (FR-018: "absence of correction, user corrections, re-teaching, explicit agent references") individually defined with detection criteria? [Gap, Spec §FR-018]
  > **Resolved**: Signals are detected holistically by Haiku post-eval (scoring.md post-eval prompt: "Did the agent's response align with the guidance?") rather than mechanically per-signal. Leech diagnosis (leech-diagnosis.md) defines specific data signatures for each diagnosis category (content_quality, wrong_tier, enforcement_gap, retrieval_mismatch) that map to these signal types. The FR-018 list serves as motivating examples of what Haiku evaluates — not separate detection mechanisms.

- [x] CHK002 - Is the `Rewrite()` method referenced in leech-diagnosis.md contract (behavior §1) defined in any interface specification? [Gap, Contract leech-diagnosis §Behavior]
  > **Resolved**: No `Rewrite()` method exists in the current contract. The contract defines `PreviewLeechRewrite()` which generates the rewrite preview, and `ApplyLeechAction()` which executes the rewrite (update content + re-embed). This checklist item appears to reference an earlier contract draft; the current contract is self-consistent.

- [x] CHK003 - Are requirements specified for what happens when a memory's quadrant changes (e.g., from "working" back to "leech" after regression)? [Gap, Spec §FR-005]
  > **Resolved**: Quadrants are recomputed every scoring run by `ClassifyQuadrants()` (scoring.md §ClassifyQuadrants). Regression is handled naturally — a working memory whose impact degrades below threshold reclassifies to leech on the next scoring run. US6-AC3 specifies: "changes are persisted but no tier changes happen automatically (proposals queued for next interaction)." NFR-005 requires logging quadrant reclassifications.

- [x] CHK004 - Are requirements defined for the spreading activation feature (FR-017) in any user story acceptance scenario? [Gap, Spec §FR-017]
  > **Resolved**: Added US2-AC6: "Given a memory is retrieved with high activation, When semantically related memories exist (similarity > 0.7), Then those related memories receive a temporary activation boost for the current interaction only (spreading activation)."

- [x] CHK005 - Are requirements specified for how synthesized guidance (FR-003) is attributed back to source memories for impact tracking? [Gap, Spec §FR-003 / §FR-004]
  > **Resolved**: Added to spec edge cases: "Synthesis attribution: all source memories with ShouldSynthesize=true receive the same faithfulness score from the synthesized output's post-eval." Also added to filter contract §Synthesis Follow-Up as scoring attribution rule.

- [x] CHK006 - Is the filter tag taxonomy ("relevant", "noise", "should-be-hook", "should-be-earlier") documented in the spec or only in the filter contract? [Gap, Spec §FR-002 vs Contract filter §Output]
  > **Resolved**: The spec uses tags in acceptance scenarios (US1-AC2: "should-be-hook") and edge cases. The complete taxonomy lives in the filter contract (filter.md §Output), which is the correct abstraction — the spec defines behavior ("tagged for later diagnosis"), the contract defines the exact tag values. No duplication needed.

- [x] CHK007 - Are requirements defined for what happens to a memory's surfacing history and impact data when a leech action rewrites or re-embeds the content? [Gap, Spec §FR-008]
  > **Resolved**: Rewrite/narrow_scope actions update the memory in-place (same `id` in embeddings table). Old surfacing_events retain their FK to the memory and their historical faithfulness scores. Post-rewrite surfacings generate new events that reflect the rewritten content's effectiveness. The impact_score naturally shifts as new (hopefully better) events accumulate. This is the correct behavior — preserving history enables measuring whether the rewrite actually helped.

- [x] CHK008 - Are requirements specified for how the system handles the first N sessions before sufficient surfacing data exists for meaningful scoring? [Gap, Spec §FR-006]
  > **Resolved**: Spec Assumptions: "Starting with a blank slate for impact scores." Data model defaults: importance_score=0.0, impact_score=0.0, quadrant='noise'. ClassifyQuadrants only processes memories with importance_score > 0 or impact_score > 0 (scoring.md §ClassifyQuadrants). AutoTuneThresholds operates on distribution percentages, so with few memories the adjustments are minimal. System degrades gracefully — new memories stay as "noise" until they accumulate surfacing data.

- [x] CHK009 - Does the spec define requirements for observability/logging of the evaluation pipeline (filter decisions, scoring runs, auto-tune adjustments)? [Gap]
  > **Resolved**: NFR-005 added to spec: "System MUST produce structured log output for key pipeline decision points: filter decisions (kept/dropped counts per query), scoring run summaries (events evaluated, scores changed), auto-tune adjustments (old/new thresholds, trigger reason), and quadrant reclassifications (memory ID, old/new quadrant)."

- [x] CHK010 - Are requirements specified for how `session_id` is obtained and what happens when it is unavailable? [Gap, Data-model §surfacing_events — session_id is nullable]
  > **Resolved**: Session ID comes from Claude Code hook infrastructure (hook events include session context). Data model makes session_id nullable to handle unavailability. User-feedback contract: "Session ID missing → Attempt to derive from environment/stdin." Surfacing events without session_id can still be logged (filter results preserved) but session-scoped queries won't find them. This is acceptable degradation for a nullable field.

## Requirement Clarity

- [x] CHK011 - Is "high weight" for explicit user feedback (FR-012) quantified? Contract says 1 explicit = ~10 implicit, but spec uses vague "weight it highly." [Ambiguity, Spec §FR-012 vs Contract user-feedback §Behavior]
  > **Resolved**: FR-012 now reads "1 explicit signal ≈ 10 implicit signals." Clarifications §2026-02-20 confirms. Contract defines exact adjustments: helpful +0.1, wrong -0.2, unclear -0.05.

- [x] CHK012 - Is "configurable number of consecutive low-impact surfacings" (FR-007) specified with a default value and valid range in the spec? [Ambiguity, Spec §FR-007]
  > **Resolved**: FR-007 specifies "default: 5, where 'low impact' = faithfulness < 0.3." Data model: leech_threshold default "5" in metadata. Valid range not explicitly bounded but implicitly integer >= 1 (you can't have 0 consecutive low-impact surfacings trigger diagnosis).

- [x] CHK013 - Is "when combining them adds value" (FR-003) defined with measurable criteria, or is it entirely delegated to the LLM's judgment? [Ambiguity, Spec §FR-003]
  > **Resolved**: Intentionally delegated to LLM. FR-003: "as determined by the filter's assessment." Research R2: "Haiku predicts synthesis value during the filter call itself. This avoids a mechanical '2+ memories' rule and makes the synthesis decision content-aware at zero additional cost." The ShouldSynthesize flag is a per-candidate LLM judgment — this is by design.

- [x] CHK014 - Are "target percentages" for quadrant distributions (FR-006) specified? Contract only defines leech target (5-15%) — are working/gem/noise targets intentionally omitted? [Ambiguity, Spec §FR-006 vs Contract scoring §AutoTuneThresholds]
  > **Resolved**: Clarifications: "Only leech quadrant is actively targeted (5-15%). Other quadrant distributions are derived." FR-006 confirms. The system adjusts thresholds to keep leeches at 5-15%; working/gem/noise distributions fall out naturally.

- [x] CHK015 - Is the "line budget" for CLAUDE.md (FR-014) quantified in the spec? Contract defaults to 100, but spec says only "a line budget." [Ambiguity, Spec §FR-014 vs Contract claude-md-quality §EnforceClaudeMDBudget]
  > **Resolved**: FR-014 now reads "default: 100 lines, configurable." Clarifications confirm: "Default 100 lines, configurable via metadata table."

- [x] CHK016 - Is "right tier" in the quality gate (FR-010) defined with criteria for distinguishing hook vs. skill vs. CLAUDE.md vs. memory? [Ambiguity, Spec §FR-010]
  > **Resolved**: Intentionally delegated to LLM for general classification. Claude-md-quality contract §ProposeClaudeMDChange check 5: "Haiku evaluates: 'Would a hook or skill be more appropriate?'" For leech diagnosis, specific data signatures distinguish tiers (leech-diagnosis.md §DiagnoseLeech): enforcement_gap → hook, wrong_tier → earlier-loaded tier. The combination of LLM judgment (for promotions) and pattern-based signals (for leech diagnosis) is the intended design.

- [x] CHK017 - Are the 5 CLAUDE.md quality dimensions (FR-013: precision, faithfulness, currency, conciseness, coverage) defined with scoring methodology in the spec, or only in the contract? [Clarity, Spec §FR-013 vs Contract claude-md-quality §ScoreClaudeMD]
  > **Resolved**: Correct abstraction split. Spec FR-013 lists the dimensions (WHAT). Contract claude-md-quality.md §ScoreClaudeMD defines measurement methodology and weights (HOW). This is the expected spec/contract separation.

- [x] CHK018 - Is the sampling rate "~10%" (FR-019) specified with an exact default and configuration mechanism? [Ambiguity, Spec §FR-019]
  > **Resolved**: FR-019 now specifies 100% evaluation with no sampling. Clarifications: "Cost is negligible (~$0.03/day at typical usage). Sampling was removed because it distorts threshold-based counters (leech_count)."

## Requirement Consistency

- [x] CHK019 - Spec FR-008 lists leech actions as "rewrite, re-tier, convert to hook, narrow scope" but contract uses "rewrite, promote_to_claude_md, convert_to_hook, narrow_scope." Is "re-tier" ≡ "promote_to_claude_md" or are these different? [Conflict, Spec §FR-008 vs Contract leech-diagnosis §ApplyLeechAction]
  > **Resolved**: Clarifications: "'Move to earlier-loaded tier' — normalized to match contract terminology. Most common target: CLAUDE.md." The spec's "re-tier" is the user-facing description; the contract's "promote_to_claude_md" is the implementation. Semantically equivalent.

- [x] CHK020 - Spec FR-016 says "incorporate impact scores into the existing activation formula" while contract defines `effectiveness = importance_score + α × impact_score` as a separate score. Are these the same concept or different computations? [Conflict, Spec §FR-016 vs Contract scoring §ScoreSession]
  > **Resolved**: Clarifications: "Yes. effectiveness = base ACT-R activation + α × impact_score, where α starts at 0.5." FR-016 now includes the formula. The scoring contract matches.

- [x] CHK021 - Spec FR-009 lists session-end triggers as "compact, clear, exit" but the plan maps to "Stop, PreCompact hooks." Is there a verified mapping from user actions to hook events for all three? [Conflict, Spec §FR-009 vs Plan §Technical Context]
  > **Resolved**: Mapping: exit→Stop hook, compact→PreCompact hook, clear→documented dependency (Spec Assumptions: "If `/clear` does not currently fire a hookable event, this is a dependency that must be resolved"). This is tracked as a known assumption, not an undiscovered conflict. See CHK047 for the /clear dependency status.

- [x] CHK022 - User Story 1 AC3 says synthesis produces "a synthesized guidance paragraph" while filter contract says synthesis triggers "when 2+ candidates have [ShouldSynthesize] set." Are these consistent — can synthesis occur with only 2 relevant memories? [Consistency, Spec §US1-AC3 vs Contract filter §Synthesis]
  > **Resolved**: Consistent. "Multiple relevant memories" (US1-AC3) and "2+ candidates" (filter contract) are the same condition. 2 is "multiple."

- [x] CHK023 - Spec FR-005 defines four quadrants as "working, leech, gem, noise" but data-model.md uses "working, leech, gem, noise" — is the default quadrant for new memories defined consistently? Data model says "noise" default. [Consistency, Spec §FR-005 vs Data-model §embeddings]
  > **Resolved**: Consistent. Data model: quadrant default 'noise'. Scoring formula: importance < threshold AND impact < threshold → 'noise'. New memories (0.0/0.0) correctly default to noise.

- [x] CHK024 - Scoring contract says `leech_count` increments when `faithfulness < 0.3` but spec uses "low-impact" without specifying a threshold. Are these aligned? [Consistency, Spec §FR-007 vs Contract scoring §ScoreSession step 3]
  > **Resolved**: FR-007 now specifies: "consecutive low-impact surfacings (default: 5, where 'low impact' = faithfulness < 0.3)." Aligned with scoring contract.

## Acceptance Criteria Quality

- [x] CHK025 - Can SC-001 ("at least 70% of memories surfaced pass the relevance filter") be measured without a baseline? The spec says "compared to the current baseline of unfiltered retrieval" but doesn't define how to compute it. [Measurability, Spec §SC-001]
  > **Resolved**: The measurement IS the filter's kept/total ratio, which is stored as `context_precision` in surfacing_events. Before the filter, 100% of retrieved memories are surfaced (context_precision = 1.0). After the filter, context_precision = kept/total. SC-001 requires this ratio averages ≥ 0.70 across interactions.

- [x] CHK026 - Is SC-006 ("self-calibrate within 20 sessions") testable? What constitutes "calibrated" — is there a convergence criterion? [Measurability, Spec §SC-006]
  > **Resolved**: Clarifications: "Calibrated = leech distribution maintained within 5-15% for 3+ consecutive scoring runs." SC-006 now includes this definition.

- [x] CHK027 - Is SC-008 ("quality scores improve or remain stable over a 30-day period") measurable without defining measurement frequency and regression thresholds? [Measurability, Spec §SC-008]
  > **Resolved**: ScoreClaudeMD provides per-dimension scores with letter grades. Measurement mechanism exists. Operational measurement frequency is a deployment concern, not a spec gap. Regression = any dimension's score decreasing. The 30-day period establishes the observation window.

- [x] CHK028 - User Story 3 AC1 says "5+ consecutive times with low faithfulness" — is this aligned with the configurable `leech_threshold` (metadata default: 5), and is "low faithfulness" quantified? [Measurability, Spec §US3-AC1 vs Data-model §metadata]
  > **Resolved**: FR-007 quantifies: "default: 5, where 'low impact' = faithfulness < 0.3." Data model: leech_threshold=5, leech_count increments when faithfulness < 0.3. Aligned.

- [x] CHK029 - User Story 4 AC2 lists quality gate checks (actionable, universal, non-redundant, right tier) but doesn't define pass criteria for each. Can these be objectively evaluated? [Measurability, Spec §US4-AC2]
  > **Resolved**: Pass criteria defined in claude-md-quality contract §ProposeClaudeMDChange: (1) working quadrant, (2) surfaced 3+ projects, (3) Haiku actionability check, (4) similarity-based redundancy check, (5) Haiku tier appropriateness. Each is a binary pass/fail.

## Scenario Coverage

- [x] CHK030 - Are requirements defined for the scenario where `Filter()` degrades (all candidates passed through) — do surfacing events still get logged, and how is impact scoring affected? [Coverage, Exception Flow, Contract filter §Error Handling]
  > **Resolved**: Updated filter contract error handling. Degradation mode: candidates returned with RelevanceScore=-1.0 sentinel. Caller logs surfacing events with haiku_relevant=NULL, haiku_tag=NULL, haiku_relevance_score=NULL. These events are excluded from post-eval scoring (contribute to importance only). Prevents false leech detection from unfiltered noise.

- [x] CHK031 - Are requirements specified for what happens when scoring runs on a very short session (single interaction) with only one surfacing event? [Coverage, Spec §Edge Cases]
  > **Resolved**: Spec edge cases: "Short sessions (single interaction before clear/exit): Scored normally." Single events are processed like any other. Scoring contract: ScoreSession processes all events for the session regardless of count.

- [x] CHK032 - Are requirements defined for the concurrent scoring scenario — what if a new session starts while the previous session's end-of-session scoring is still running? [Coverage, Gap]
  > **Resolved**: The system is a single-user CLI tool with ephemeral sessions. Each session runs in its own process. Concurrent sessions are unlikely. SQLite WAL mode + busy_timeout=5000ms (plan.md) handles concurrent access at the DB level. No additional requirements needed.

- [x] CHK033 - Are recovery requirements specified for when `ApplyLeechAction` fails partway through (e.g., re-embed succeeds but DB update fails)? [Coverage, Recovery Flow, Contract leech-diagnosis §ApplyLeechAction]
  > **Resolved**: Added to leech-diagnosis contract §Constraints: "Memory-internal actions (rewrite, narrow_scope) MUST execute within a single database transaction. If re-embedding fails, the transaction rolls back and the error is returned. The memory's original content and embedding are preserved."

- [x] CHK034 - Are requirements defined for how the system transitions from the current count-based system to the new evaluation-based system (migration path for existing memories with no surfacing data)? [Coverage, Gap]
  > **Resolved**: Spec Assumptions: "Starting with a blank slate for impact scores (no retroactive bootstrapping from historical data)." Data model: new columns have defaults (importance_score=0.0, impact_score=0.0, effectiveness=0.0, quadrant='noise', leech_count=0). Existing memories start as noise and accumulate evaluation data going forward.

- [x] CHK035 - Is the scenario where a user provides feedback on a memory that was already scored at session end addressed? Does the feedback retroactively adjust the score? [Coverage, Gap]
  > **Resolved**: User feedback updates the surfacing event's user_feedback field and immediately adjusts the memory's impact_score (user-feedback contract §Behavior). This works regardless of whether scoring already ran. The next scoring run will use the updated surfacing event data (including user_feedback), but the immediate impact_score adjustment provides real-time response.

## Edge Case Coverage

- [x] CHK036 - The spec lists "conflicting memories (two relevant memories with contradictory guidance)" as an edge case — is this addressed by any FR or contract? [Gap, Spec §Edge Cases]
  > **Resolved**: Spec edge cases: "Delegated to synthesis. When both memories have ShouldSynthesize=true, Sonnet resolves the contradiction during synthesis." Clarifications confirm: "Delegate to synthesis. Sonnet resolves contradictions..." Filter contract §Synthesis Follow-Up describes the mechanism.

- [x] CHK037 - The spec lists "oscillating impact score (sometimes helpful, sometimes not)" — are damping or smoothing requirements defined? [Gap, Spec §Edge Cases]
  > **Resolved**: Spec edge cases: "Recency-weighted average in impact scoring naturally trends toward recent behavior. No special handling needed; persistent oscillation eventually classifies as 'leech' for diagnosis."

- [x] CHK038 - The spec lists "contradictory explicit feedback on the same memory across sessions" — is the resolution strategy specified? [Gap, Spec §Edge Cases]
  > **Resolved**: Spec edge cases: "Each `/memory` feedback command applies to its own surfacing event. Feedback from session N and session N+1 affect different surfacing events; impact score naturally reflects the pattern over time."

- [x] CHK039 - The spec lists "memories that are relevant but for which no post-interaction signal is available" — are requirements defined for handling unsampled/unevaluated surfacings in impact computation? [Gap, Spec §Edge Cases]
  > **Resolved**: Spec edge cases: "Unevaluated surfacings (faithfulness=NULL) are excluded from impact score calculation. They still contribute to importance (surfacing count) only."

- [x] CHK040 - Are requirements defined for what happens when the metadata table keys (alpha_weight, thresholds) are missing or corrupted? [Edge Case, Data-model §metadata]
  > **Resolved**: Data model specifies default values via INSERT OR IGNORE INTO metadata for all 5 keys. Missing keys are created on DB initialization. Corruption (non-numeric values) is an implementation concern — Go's `strconv.ParseFloat` returns errors that should be handled by falling back to defaults. AutoTuneThresholds clamps values to [0.0, 1.0] (scoring contract), preventing drift.

- [x] CHK041 - Are requirements specified for the edge case where all memories are classified into a single quadrant (e.g., all "noise" or all "working")? [Edge Case, Spec §FR-006]
  > **Resolved**: AutoTuneThresholds only adjusts when leech% is outside 5-15% range. If all memories are "noise" (cold start), no active memories have scores > 0, so ClassifyQuadrants doesn't process them and auto-tune has nothing to adjust. If all become "working" (all effective), leech% = 0% → impact_threshold increases by 0.05 per run until some reclassify as leech. This is correct self-correcting behavior.

## Non-Functional Requirements

- [x] CHK042 - Is the $0.15/day cost budget (SC-005) broken down by component (filter calls, post-eval sampling, synthesis, CLAUDE.md scoring) to enable per-component budgeting? [Completeness, Spec §SC-005]
  > **Resolved**: Component costs documented across contracts: filter ~$0.0001/call (filter contract), post-eval ~$0.0001/call × ~300 relevant events/day = ~$0.03/day (scoring contract §Evaluation Strategy), synthesis ~$0.001-0.005/call (Sonnet, triggered occasionally). Total well under $0.15/day. Per-component budgets aren't specified because the aggregate is safely under budget. Task T032 requires a cost estimation benchmark test.

- [x] CHK043 - Are performance requirements (<200ms filter, <1s synthesis, <5s scoring) specified in the spec or only in the plan? [Gap, Plan §Technical Context — not in spec.md]
  > **Resolved**: NFR-001 (filter <200ms), NFR-002 (synthesis <1s), NFR-003 (scoring <5s) are all in the spec.

- [x] CHK044 - Are data retention requirements specified for surfacing_events? Will the table grow unbounded, or is there a pruning/archival policy? [Gap]
  > **Resolved (acceptable risk)**: No pruning policy specified. For a single-user CLI tool with ~100 events/day (~500 per day across all memories × queries), growth is ~182K rows/year. SQLite handles millions of rows efficiently. Indexes ensure query performance. Data retention is a future concern — the table can be pruned manually or by a future feature when it becomes relevant. Not a blocking gap.

- [x] CHK045 - Are privacy/data sensitivity requirements defined for storing query text and session IDs in surfacing_events? [Gap, Data-model §surfacing_events]
  > **Resolved**: This is a local single-user CLI tool. All data (query text, session IDs, embeddings) is stored in the user's own SQLite file on their local machine. No data is transmitted externally (except to Anthropic API for evaluation, which is already the case for the existing pipeline). Privacy risk is minimal. No additional requirements needed.

- [x] CHK046 - Are requirements defined for the system's behavior under SQLite contention (concurrent reads during scoring writes)? [Gap]
  > **Resolved**: Plan specifies "WAL mode, busy_timeout=5000ms." WAL mode allows concurrent readers during writes. Single-user CLI tool makes concurrent access unlikely. SQLite's built-in contention handling is sufficient.

## Dependencies & Assumptions

- [x] CHK047 - Is the assumption "If `/clear` does not currently fire a hookable event" resolved or still open? FR-009 requires clear-triggered scoring. [Assumption, Spec §Assumptions]
  > **Resolved**: /clear triggers `SessionStart` event with `source: "clear"`. A SessionStart hook with matcher "clear" can run end-of-session scoring for the preceding session. Spec Assumptions updated. Task T015 updated to include SessionStart-clear hook.

- [x] CHK048 - Is the assumption "starting with a blank slate for impact scores" validated — are there requirements for bootstrapping scoring from existing activation data? [Assumption, Spec §Assumptions]
  > **Resolved**: Explicit design decision. Spec Assumptions: "Starting with a blank slate for impact scores (no retroactive bootstrapping from historical data)." No bootstrapping is needed — the system learns going forward.

- [x] CHK049 - Is the dependency on the existing `Synthesize()` method documented — does it currently exist, and does it accept a model override parameter for Sonnet? [Dependency, Contract filter §Synthesis Follow-Up]
  > **Resolved**: Synthesize() exists on LLMExtractor interface (internal/memory/llm.go:64). Signature: `Synthesize(ctx context.Context, memories []string) (string, error)`. No model override parameter — implementations use their pre-configured model (Haiku). Filter contract updated: caller must use a Sonnet-configured LLMExtractor instance for synthesis. T009 must create/use a Sonnet extractor instance.

- [x] CHK050 - Are requirements specified for how the system detects "user corrections" (FR-018, sampling at 100%) — is this a new detection mechanism or does it leverage an existing one? [Dependency, Spec §FR-018/FR-019]
  > **Resolved**: User corrections are detected by Haiku post-eval (scoring contract §Post-Eval Prompt). The prompt includes "User's next message (if any)" — Haiku assesses whether the user corrected the agent's behavior. This leverages the existing LLM evaluation infrastructure, not a new mechanical detection system.

## Cross-Artifact Consistency (Spec ↔ Contracts ↔ Data Model)

- [x] CHK051 - Does every spec FR have at least one contract that traces to it? Verify: FR-001→filter, FR-002→filter+surfacing, FR-003→filter, FR-004→surfacing+scoring, FR-005→scoring, FR-006→scoring, FR-007→scoring+leech, FR-008→leech, FR-009→scoring, FR-010→claude-md, FR-011→claude-md+leech, FR-012→user-feedback, FR-013→claude-md, FR-014→claude-md, FR-015→claude-md, FR-016→scoring, FR-017→scoring, FR-018→surfacing, FR-019→scoring. [Traceability]
  > **Resolved**: Updated scoring.md header to include FR-009 and FR-019. All 19 FRs now trace to at least one contract.

- [x] CHK052 - Does the data model define storage for every field in every contract's data types? Cross-check FilterResult, SurfacingEvent, QuadrantSummary, LeechCandidate, LeechDiagnosis, CLAUDEMDProposal, CLAUDEMDScore, CLAUDEMDSection. [Traceability, Data-model vs Contracts]
  > **Resolved**: Data model defines persisted entities: surfacing_events table, embeddings columns, metadata keys, FilterResult, LeechDiagnosis, CLAUDEMDProposal, Recommendation. Non-persisted return types (QuadrantSummary, LeechCandidate, CLAUDEMDScore, CLAUDEMDSection) are defined in their respective contracts. Data model covers storage; contracts cover in-memory types. This is the correct split — the data model doesn't need to enumerate every computed/transient struct.

- [x] CHK053 - Is the contract header FR mapping complete? Verify each contract lists all relevant FRs: e.g., scoring.md lists FR-004/005/006/007/016/017 but omits FR-009 (session-end scoring) despite defining `ScoreSession`. [Consistency, Contract scoring header vs Spec §FR-009]
  > **Resolved**: Same fix as CHK051. scoring.md header updated to include FR-009 and FR-019.

- [x] CHK054 - Are the user-feedback contract's immediate score adjustments (+0.1/-0.2/-0.05) documented in the spec, or are they implementation details leaking into contracts? [Consistency, Contract user-feedback §Behavior vs Spec §FR-012]
  > **Resolved**: Correct abstraction level. Spec FR-012 specifies the ratio: "1 explicit signal ≈ 10 implicit signals." The exact adjustment magnitudes (+0.1/-0.2/-0.05) are implementation details that belong in the contract, not the spec. The spec defines intent (high weight), the contract defines mechanism (exact values).

## Notes

- Check items off as completed: `[x]`
- Add findings/resolutions inline after each item
- Items are numbered CHK001-CHK054 for cross-reference
- This checklist audits **requirements quality**, not implementation correctness
