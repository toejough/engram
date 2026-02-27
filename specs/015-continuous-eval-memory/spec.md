# Feature Specification: Continuous Evaluation Memory Pipeline

**Feature Branch**: `015-continuous-eval-memory`
**Created**: 2026-02-20
**Status**: Draft
**Input**: User description: "based on the design doc"
**Design Doc**: [Continuous Evaluation Memory Pipeline Design](../../docs/plans/2026-02-20-continuous-evaluation-memory-design.md)
**Issue**: ISSUE-235

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Relevant Memories Only (Priority: P1)

When the memory system surfaces context during my work, I only see memories that are actually relevant to what I'm doing right now. Noise and irrelevant matches are filtered out before they reach me, so surfaced memories are consistently useful rather than distracting.

**Why this priority**: The current system surfaces raw retrieval results with no quality filter. Irrelevant memories waste attention and erode trust in the system. Filtering is the foundation — every other feature depends on having clean signal.

**Independent Test**: Can be fully tested by verifying that surfaced memories pass a relevance check before reaching the agent, and that irrelevant matches are suppressed. Delivers immediate value by reducing noise in every interaction.

**Acceptance Scenarios**:

1. **Given** the memory system retrieves 5 candidate memories for a user prompt, **When** 3 are relevant and 2 are noise, **Then** only the 3 relevant memories are surfaced to the agent.
2. **Given** a memory is retrieved that would be better enforced deterministically (e.g., a commit trailer format), **When** the filter evaluates it, **Then** it is tagged as "should-be-hook" for later diagnosis.
3. **Given** multiple relevant memories are surfaced, **When** the filter determines combining them would produce more actionable guidance, **Then** a synthesized guidance paragraph is presented instead of raw individual memories.
4. **Given** zero memories pass the relevance filter, **When** no relevant context exists, **Then** no memory context is surfaced (no empty or placeholder output).

---

### User Story 2 - Impact Tracking and Quadrant Classification (Priority: P2)

The system tracks whether surfaced memories actually help me. Each memory is scored on two dimensions — how often it comes up (importance) and whether it improves outcomes when surfaced (impact). Memories are classified into four quadrants (working, leech, gem, noise) that drive different actions.

**Why this priority**: Without impact measurement, the system cannot distinguish between a memory surfaced 100 times and followed every time versus one surfaced 100 times and ignored every time. Impact tracking is the core differentiator from the current count-based system.

**Independent Test**: Can be tested by surfacing memories across multiple interactions, recording outcomes, and verifying that quadrant classification reflects actual effectiveness patterns.

**Acceptance Scenarios**:

1. **Given** a memory is surfaced and the agent's response aligns with the guidance, **When** the session ends, **Then** the memory's impact score increases.
2. **Given** a memory is surfaced but the user corrects the agent in the same area, **When** the session ends, **Then** the memory's impact score decreases.
3. **Given** a memory has high importance and high impact scores, **When** quadrant classification runs, **Then** it is classified as "working knowledge."
4. **Given** a memory has high importance but low impact, **When** quadrant classification runs, **Then** it is classified as "leech."
5. **Given** scoring parameters (importance/impact thresholds), **When** the distribution of memories across quadrants drifts outside target ranges, **Then** the system auto-adjusts thresholds to rebalance.
6. **Given** a memory is retrieved with high activation, **When** semantically related memories exist (similarity > 0.7), **Then** those related memories receive a temporary activation boost for the current interaction only (spreading activation).

---

### User Story 3 - Leech Diagnosis (Priority: P3)

When a memory keeps matching contexts but doesn't improve outcomes (a "leech"), the system diagnoses *why* it's failing and presents me with actionable options: rewrite the content, move it to an earlier-loaded tier, convert it to a deterministic hook, or narrow its retrieval scope.

**Why this priority**: Leeches are the highest-value diagnostic signal — they indicate something systematically broken. Diagnosing root cause is more valuable than any automated fix, and directly reduces the most frustrating failure mode (repeatedly surfaced but unhelpful memories).

**Independent Test**: Can be tested by creating a memory that is repeatedly surfaced but never followed, verifying it reaches "leech" classification, and confirming that a root cause analysis with options is presented.

**Acceptance Scenarios**:

1. **Given** a memory has been surfaced 5+ consecutive times with low faithfulness, **When** leech diagnosis triggers, **Then** the user is presented with a root cause analysis identifying the likely failure mode.
2. **Given** a leech is diagnosed as "wrong tier" (surfaced after mistake already happened), **When** the diagnosis is presented, **Then** the proposed action is to move it to an earlier-loaded tier.
3. **Given** a leech is diagnosed as "enforcement gap" (agent understands but doesn't comply), **When** the diagnosis is presented, **Then** the proposed action is to convert it to a deterministic hook.
4. **Given** the user selects a proposed action for a leech, **When** they approve the change, **Then** the system executes the tier change. No changes happen without approval.

---

### User Story 4 - CLAUDE.md Quality Gate (Priority: P4)

CLAUDE.md content management is driven by measured effectiveness rather than mechanical thresholds. Promotion to CLAUDE.md requires proven impact across multiple projects, passes a quality gate (actionable, universal, non-redundant, right tier), is routed to the appropriate section, and is always presented as a proposal for my approval.

**Why this priority**: CLAUDE.md is the highest-impact tier (always loaded), so its quality directly affects every interaction. But this depends on having impact data (P2) and diagnosis capabilities (P3) in place first.

**Independent Test**: Can be tested by creating a memory that reaches "working knowledge" status across 3+ projects, verifying the quality gate checks fire, and confirming a section-routed proposal is presented.

**Acceptance Scenarios**:

1. **Given** a memory reaches "working knowledge" quadrant and has been effective across 3+ projects, **When** the quality gate evaluates it, **Then** a promotion proposal is presented to the user.
2. **Given** a proposed CLAUDE.md entry, **When** the quality gate evaluates it, **Then** it checks: is the entry actionable? Universal (not project-specific)? Non-redundant with existing hooks/skills/entries? In the right tier?
3. **Given** CLAUDE.md exceeds the line budget, **When** size enforcement runs, **Then** the lowest-effectiveness entries are proposed for demotion with destinations determined by diagnosis logic.
4. **Given** a CLAUDE.md quality score is requested, **When** the scoring function runs, **Then** a grade (A through F) is reported across precision, faithfulness, currency, conciseness, and coverage dimensions.
5. **Given** a promotion proposal, **When** the user rejects it, **Then** no change is made to CLAUDE.md.

---

### User Story 5 - User Feedback Commands (Priority: P5)

I can provide explicit feedback on surfaced memories using simple commands (`/memory helpful`, `/memory wrong`, `/memory unclear`). This feedback carries high weight in impact scoring, giving me direct influence over which memories the system considers effective.

**Why this priority**: Explicit user feedback is the highest-quality signal but depends on the impact tracking infrastructure (P2) being in place. Low implementation effort once the foundation exists.

**Independent Test**: Can be tested by surfacing a memory, issuing a feedback command, and verifying the impact score changes accordingly.

**Acceptance Scenarios**:

1. **Given** a memory was just surfaced, **When** the user runs `/memory helpful`, **Then** the most recent surfacing event is updated with positive feedback and the memory's impact score increases.
2. **Given** a memory was just surfaced, **When** the user runs `/memory wrong`, **Then** the most recent surfacing event is updated with negative feedback and the memory's impact score decreases.
3. **Given** no memory was surfaced in the current session, **When** the user runs `/memory helpful`, **Then** the system responds with an appropriate message indicating no recent surfacing to rate.

---

### User Story 6 - End-of-Session Scoring (Priority: P6)

Impact scoring runs automatically at the end of every session (on compact, clear, or exit) without requiring me to run a manual optimization command. The system learns from every session, not just when I remember to run `optimize`.

**Why this priority**: Makes the learning loop automatic. Depends on surfacing events (P1) and the scoring model (P2), but is what makes the system genuinely self-improving rather than batch-optimized.

**Independent Test**: Can be tested by completing a session with surfacing events, triggering session end, and verifying that importance/impact scores and quadrant classifications are updated.

**Acceptance Scenarios**:

1. **Given** a session has recorded surfacing events, **When** the session ends (compact, clear, or exit), **Then** all surfacing events from the session are evaluated and scores updated.
2. **Given** a session ends with no surfacing events, **When** end-of-session scoring triggers, **Then** no scoring work is done and no errors occur.
3. **Given** scoring runs at session end, **When** quadrant classifications change, **Then** changes are persisted but no tier changes happen automatically (proposals queued for next interaction).

---

### Edge Cases

- **Empty retrieval set** (E5 returns no candidates): Filter returns empty slice without making an API call. No surfacing events logged, no memory context surfaced.
- **Conflicting memories** (two relevant memories with contradictory guidance): Delegated to synthesis. When both memories have ShouldSynthesize=true, Sonnet resolves the contradiction during synthesis. Otherwise, both are surfaced individually and the agent reconciles.
- **Oscillating impact score** (sometimes helpful, sometimes not): Recency-weighted average in impact scoring naturally trends toward recent behavior. No special handling needed; persistent oscillation eventually classifies as "leech" for diagnosis.
- **Short sessions** (single interaction before clear/exit): Scored normally. If the single surfacing event isn't sampled for post-evaluation, only implicit signals contribute. Importance still tracks the surfacing.
- **Contradictory explicit feedback across sessions**: Each `/memory` feedback command applies to its own surfacing event. Feedback from session N and session N+1 affect different surfacing events; impact score naturally reflects the pattern over time.
- **No post-interaction signal available** (agent responded but user didn't react): Unevaluated surfacings (faithfulness=NULL) are excluded from impact score calculation. They still contribute to importance (surfacing count) only.
- **Synthesis attribution**: When synthesis combines multiple memories, all source memories that had ShouldSynthesize=true receive the same faithfulness score from the synthesized output's post-eval. Each source memory's surfacing event is logged individually (haiku_relevant=true) and scored against the synthesis result.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST filter retrieved memories for relevance before surfacing them to the agent.
- **FR-002**: System MUST log each surfacing event with filter results, relevance scores, and original similarity scores.
- **FR-003**: System MUST synthesize multiple relevant memories into actionable guidance when combining them adds value, as determined by the filter's assessment.
- **FR-004**: System MUST track impact (effectiveness when surfaced) as a distinct dimension from importance (frequency/recency of surfacing).
- **FR-005**: System MUST classify each memory into a quadrant (working, leech, gem, noise) based on measured importance and impact scores.
- **FR-006**: System MUST auto-tune scoring thresholds by tracking quadrant distribution and adjusting to maintain target ranges (leech quadrant: 5-15% of scored memories; other quadrant distributions are derived).
- **FR-007**: System MUST diagnose leech memories with root cause analysis after a configurable number of consecutive low-impact surfacings (default: 5, where "low impact" = faithfulness < 0.3).
- **FR-008**: System MUST present leech diagnoses as proposals with actionable options (rewrite content, move to earlier-loaded tier, convert to deterministic hook, narrow retrieval scope).
- **FR-009**: System MUST evaluate memory impact at session end (compact, clear, exit) without requiring manual optimization.
- **FR-010**: System MUST gate CLAUDE.md promotions on measured effectiveness, universality, actionability, non-redundancy, and tier appropriateness.
- **FR-011**: System MUST present all non-memory changes (promotion, demotion, hook conversion, skill creation/merge) as tool-agnostic Recommendations describing the desired outcome, never writing to CLAUDE.md, hook configs, or skill files. Recommendations MUST be offered as a saveable markdown file since they may be disruptive.
- **FR-012**: System MUST accept explicit user feedback (`/memory helpful`, `/memory wrong`, `/memory unclear`) and weight it highly in impact scoring (1 explicit signal ≈ 10 implicit signals).
- **FR-013**: System MUST score CLAUDE.md quality across precision, faithfulness, currency, conciseness, and coverage dimensions.
- **FR-014**: System MUST enforce a line budget for CLAUDE.md (default: 100 lines, configurable), proposing demotion of lowest-effectiveness entries when over budget.
- **FR-015**: System MUST route promoted content to appropriate CLAUDE.md sections based on content type.
- **FR-016**: System MUST incorporate impact scores into the existing activation formula (effectiveness = base activation + α × impact, where α starts at 0.5), boosting high-impact memories and penalizing low-impact ones.
- **FR-017**: System MUST boost activation of semantically related memories when a memory is retrieved (spreading activation), scoped to the current interaction only.
- **FR-018**: System MUST collect implicit signals (absence of correction, user corrections, re-teaching, explicit agent references) without additional cost.
- **FR-019**: System MUST evaluate all relevant surfacing events at session end (100% — no sampling). Haiku post-eval cost (~$0.0001/call) is negligible relative to the $0.15/day budget, and sampling distorts threshold-based counters (leech_count). For long sessions, post-eval calls SHOULD be batched or concurrent to meet NFR-003.

### Non-Functional Requirements

- **NFR-001**: Haiku filter latency MUST be under 200ms per interaction.
- **NFR-002**: Sonnet synthesis latency MUST be under 1 second when triggered.
- **NFR-003**: End-of-session scoring MUST complete within 5 seconds.
- **NFR-004**: Daily operational cost of the evaluation pipeline MUST stay under $0.15 for typical usage (~100 interactions/day).
- **NFR-005**: System MUST produce structured log output for key pipeline decision points: filter decisions (kept/dropped counts per query), scoring run summaries (events evaluated, scores changed), auto-tune adjustments (old/new thresholds, trigger reason), and quadrant reclassifications (memory ID, old/new quadrant).

### Key Entities

- **Surfacing Event**: A record of a memory being retrieved and evaluated for a specific query context. Captures filter results, relevance scores, faithfulness evaluation, outcome signals, and user feedback.
- **Memory Quadrant**: Classification of a memory based on its importance and impact scores — working (keep/promote), leech (diagnose), gem (surface more), noise (natural decay).
- **Leech Diagnosis**: Root cause analysis of why a high-importance memory has low impact, with proposed remediation actions.
- **CLAUDE.md Proposal**: A structured suggestion for adding, modifying, or removing CLAUDE.md content, requiring explicit user approval.
- **Quality Score**: Multi-dimensional assessment of CLAUDE.md content across precision, faithfulness, currency, conciseness, and coverage.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: At least 70% of memories surfaced to the agent pass the relevance filter (context precision), compared to the current baseline of unfiltered retrieval.
- **SC-002**: The system identifies and diagnoses leech memories within 5 sessions of the leech pattern emerging.
- **SC-003**: All CLAUDE.md changes go through the proposal workflow — zero auto-writes to CLAUDE.md.
- **SC-004**: Impact scoring runs automatically at every session end without user intervention.
- **SC-005**: Daily operational cost of the evaluation pipeline stays under $0.15 for typical usage (~100 interactions/day).
- **SC-006**: Scoring thresholds self-calibrate within 20 sessions of initial deployment, maintaining quadrant distributions within target ranges (calibrated = leech distribution maintained within 5-15% for 3+ consecutive scoring runs).
- **SC-007**: Users can provide explicit feedback on any surfaced memory in under 5 seconds using `/memory` commands.
- **SC-008**: CLAUDE.md quality scores improve or remain stable over a 30-day period, with no regression in any individual dimension.

## Assumptions

- The existing E5-small-v2 local embedding model and SQLite storage remain adequate and are not bottlenecks.
- Claude Code hook infrastructure (Stop, PreCompact, SessionStart, PreToolUse, UserPromptSubmit) is stable and available.
- The existing DirectAPIExtractor provides reliable access to Haiku and Sonnet models for evaluation calls.
- Session boundaries (compact, clear, exit) fire hook events reliably. The `/clear` command triggers a `SessionStart` event with `source: "clear"`, which can be matched to run end-of-session scoring for the preceding session.
- The existing Learn/Query core functions, extraction pipeline, and skill testing harness remain unchanged.
- Starting with a blank slate for impact scores (no retroactive bootstrapping from historical data).

## Clarifications

### Session 2026-02-20

- Q: Is "hidden gem" the canonical term for the low-importance/high-impact quadrant? → A: No. Canonical term is "gem" (normalized across spec; "hidden" was descriptive only).
- Q: What does "weight it highly" mean for explicit feedback (FR-012)? → A: 1 explicit signal ≈ 10 implicit signals. Immediate score adjustments defined in user-feedback contract.
- Q: What is the default leech threshold (FR-007)? → A: 5 consecutive low-impact surfacings (configurable via metadata). "Low impact" = faithfulness < 0.3.
- Q: What is the CLAUDE.md line budget (FR-014)? → A: Default 100 lines, configurable via metadata table.
- Q: What is the evaluation rate (FR-019)? → A: 100% — no sampling. Cost is negligible (~$0.03/day at typical usage). Sampling was removed because it distorts threshold-based counters (leech_count).
- Q: What does "re-tier" mean in FR-008? → A: "Move to earlier-loaded tier" — normalized to match contract terminology. Most common target: CLAUDE.md.
- Q: Is FR-016 "incorporate impact into activation" the same as the effectiveness formula? → A: Yes. effectiveness = base ACT-R activation + α × impact_score, where α starts at 0.5.
- Q: What are the "target percentages" in FR-006? → A: Only leech quadrant is actively targeted (5-15%). Other quadrant distributions are derived.
- Q: What happens with an empty retrieval set? → A: Filter returns empty slice, no API call, no surfacing events, no memory context surfaced.
- Q: How do short sessions score? → A: Normally. Single events processed like any other; unevaluated events contribute to importance only.
- Q: How are contradictory feedback signals across sessions resolved? → A: Each feedback applies to its own surfacing event. Per-event design naturally handles cross-session contradictions.
- Q: How do oscillating impact scores behave? → A: Recency-weighted average trends toward recent behavior. Persistent oscillation eventually classifies as "leech" for diagnosis.
- Q: How are unevaluated surfacings (no post-interaction signal) handled? → A: faithfulness=NULL events excluded from impact calculation; contribute to importance (surfacing count) only.
- Q: Are performance targets specified? → A: Added as NFR-001 through NFR-003 (filter <200ms, synthesis <1s, scoring <5s).
- Q: What does "calibrated" mean for SC-006? → A: Leech distribution maintained within 5-15% for 3+ consecutive scoring runs.
- Q: How should conflicting memories be handled? → A: Delegate to synthesis. Sonnet resolves contradictions during synthesis when both have ShouldSynthesize=true; otherwise surface both individually.
- Q: What observability requirements should the pipeline have? → A: Structured logging for key decision points: filter decisions, scoring runs, auto-tune adjustments, quadrant reclassifications. Added as NFR-005.
