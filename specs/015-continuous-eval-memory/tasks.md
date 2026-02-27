# Tasks: Continuous Evaluation Memory Pipeline

**Input**: Design documents from `/specs/015-continuous-eval-memory/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md
**Tests**: Included — TDD required per project conventions (red/green/refactor)
**Organization**: Tasks grouped by user story; each story independently testable after completion.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks in this phase)
- **[Story]**: Which user story (US1-US6) — omitted for setup/foundational/polish phases

---

## Phase 1: Foundational (Schema + Shared Types)

**Purpose**: Database schema migrations and shared type definitions that ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T001 Write failing tests verifying surfacing_events table exists with all columns per data-model.md, embeddings has new columns (importance_score, impact_score, effectiveness, quadrant, leech_count), and metadata has default keys (alpha_weight=0.5, leech_threshold=5, importance_threshold=0.0, impact_threshold=0.3, last_autotune_at) in internal/memory/surfacing_test.go
- [x] T002 Add schema migrations to initEmbeddingsDB() in internal/memory/embeddings.go — CREATE TABLE IF NOT EXISTS surfacing_events per data-model.md with 3 indexes (memory_id, timestamp, session_id), 5 ALTER TABLE embeddings ADD COLUMN statements, 5 INSERT OR IGNORE INTO metadata statements (make T001 pass)
- [x] T003 [P] Define SurfacingEvent struct in internal/memory/surfacing.go and FilterResult struct in internal/memory/filter.go per data-model.md entity definitions

**Checkpoint**: Schema ready — user story implementation can begin

---

## Phase 2: User Story 1 — Relevant Memories Only (Priority: P1) MVP

**Goal**: Only relevant memories reach the agent. Noise suppressed, synthesis combines related memories when it adds value.

**Independent Test**: Verify that surfaced memories pass Haiku relevance check before reaching agent, irrelevant matches are suppressed, and synthesis triggers when filter recommends combining.

**Contract**: [filter.md](contracts/filter.md), [surfacing-events.md](contracts/surfacing-events.md)

### Tests for User Story 1

> Write these tests FIRST, ensure they FAIL before implementation

- [x] T004 [P] [US1] Write failing tests for Filter() in internal/memory/filter_test.go — verify FilterResult with correct relevant/noise/should-be-hook/should-be-earlier tags, empty candidates returns empty slice (no API call), LLM failure returns all candidates as relevant (graceful degradation), ShouldSynthesize flag set per-candidate
- [x] T005 [P] [US1] Write failing tests for LogSurfacingEvent (insert with all fields), GetSessionSurfacingEvents (session-scoped query ordered by timestamp), GetMemorySurfacingHistory (memory-scoped query ordered by timestamp desc with limit) in internal/memory/surfacing_test.go

### Implementation for User Story 1

- [x] T006 [US1] Add Filter(ctx context.Context, query string, candidates []QueryResult) ([]FilterResult, error) to LLMExtractor interface and implement on DirectAPIExtractor in internal/memory/llm_api.go — single Haiku API call with structured JSON response, parse into FilterResult slice, graceful degradation on API error/timeout/malformed JSON per filter contract (make T004 pass)
- [x] T007 [P] [US1] Implement LogSurfacingEvent, GetSessionSurfacingEvents, GetMemorySurfacingHistory in internal/memory/surfacing.go — SQL INSERT/SELECT per surfacing-events contract, log both kept and filtered candidates (make T005 pass)
- [x] T008 [US1] Add TierFiltered output formatting in internal/memory/format.go — format FilterResult slice for hook output, include synthesized guidance paragraph when synthesis was triggered, handle zero-result case (no output)
- [x] T009 [US1] Integrate filter into hook pipeline in internal/memory/hooks.go — replace Curate() call path with Filter() for relevant hooks, log surfacing events for all candidates (kept and filtered) via LogSurfacingEvent, trigger Synthesize() with Sonnet model when 2+ candidates have ShouldSynthesize=true, use TierFiltered formatting, delegate contradictory memories to synthesis per spec edge case resolution

**Checkpoint**: US1 complete — memories are filtered for relevance, surfacing events logged, synthesis works. Existing Curate() path remains for backwards compat.

---

## Phase 3: User Story 2 + User Story 6 — Impact Tracking, Quadrants & Auto-Scoring (Priority: P2/P6)

**Goal**: Track whether surfaced memories actually help. Classify into quadrants. Auto-tune thresholds. Score automatically at session end.

**Independent Test**: Surface memories across interactions, record outcomes, verify quadrant classification reflects effectiveness. Verify scoring runs automatically at session end (Stop/PreCompact hooks).

**Contract**: [scoring.md](contracts/scoring.md)

### Tests for User Story 2

- [x] T010 [US2] Write failing tests for ScoreSession (100% evaluation of all haiku_relevant=true events — no sampling, faithfulness update via Haiku post-eval, impact_score recalculation as recency-weighted average, leech_count increment when faithfulness < 0.3 / reset otherwise), ClassifyQuadrants (threshold-based quadrant assignment), AutoTuneThresholds (EMA adjustment, ±0.05 step, 0.1 cap per run, leech target 5-15%), GetQuadrantSummary in internal/memory/scoring_test.go

### Implementation for User Story 2 + 6

- [x] T011 [US2] Implement ScoreSession in internal/memory/scoring.go — retrieve session events via GetSessionSurfacingEvents, call Haiku post-eval for ALL haiku_relevant=true events with faithfulness=null (100% evaluation, no sampling, bounded context: memory+response excerpt+next message), update faithfulness/outcome_signal via UpdateSurfacingOutcome, recalculate impact_score and effectiveness, update leech_count
- [x] T012 [US2] Implement ClassifyQuadrants, AutoTuneThresholds, GetQuadrantSummary in internal/memory/scoring.go — read thresholds from metadata, classify memories into quadrants, EMA-based leech% targeting (5-15%) with capped adjustments, clamp thresholds to [0.0, 1.0] (make T010 pass)
- [x] T013 [P] [US2] Extend GetActivationStats() in internal/memory/embeddings.go — add α × impact_score term to base-level activation (effectiveness = B_i + α × impact) per FR-016 and research R5, with test in existing embeddings_test.go
- [x] T014 [P] [US2] Add spreading activation to Query() in internal/memory/memory.go — after E5 top-K retrieval, perform secondary vector search for memories with similarity > 0.7 to top-K results, boost activation temporarily (interaction-scoped, not persisted) per FR-017, with test
- [x] T015 [US6] Wire ScoreSession into Stop, PreCompact, and SessionStart (matcher: "clear") hook commands in internal/memory/hooks.go — call ScoreSession after existing hook logic completes, for SessionStart-clear use the most recent session with unscored surfacing events, no-op on empty session (no surfacing events), log errors without failing the hook per FR-009

**Checkpoint**: US2+US6 complete — impact scoring, quadrants, auto-tune, activation formula, spreading activation, automatic session-end scoring all functional.

---

## Phase 4: User Story 3 — Leech Diagnosis (Priority: P3)

**Goal**: Diagnose why high-importance memories have low impact and propose actionable remediation.

**Independent Test**: Create a memory surfaced repeatedly with low faithfulness, verify it reaches "leech" classification, verify root cause analysis with correct diagnosis type and proposed action.

**Contract**: [leech-diagnosis.md](contracts/leech-diagnosis.md)

### Tests for User Story 3

- [x] T016 [US3] Write failing tests for GetLeeches (quadrant='leech' AND leech_count >= threshold), DiagnoseLeech (4 categories with priority: wrong_tier > enforcement_gap > content_quality > retrieval_mismatch, first match wins, content_quality leaves SuggestedContent empty, promote/convert populate Recommendation with category+description+evidence), PreviewLeechRewrite (generates rewrite via LLM for content_quality diagnoses, returns empty string on LLM failure, error on non-content_quality diagnosis), ApplyLeechAction (rewrite/narrow_scope execute directly, promote/convert return Recommendation and mark action_recommended) in internal/memory/leech_test.go

### Implementation for User Story 3

- [x] T017 [US3] Implement GetLeeches, DiagnoseLeech, and PreviewLeechRewrite in internal/memory/leech.go — DiagnoseLeech: pure data analysis querying leeches from embeddings, analyzing surfacing_events history per diagnosis category signals (content_quality: low faithfulness + no agent reference; wrong_tier: surfaced after correction; enforcement_gap: agent references but user corrects; retrieval_mismatch: >50% haiku_relevant=false), SuggestedContent left empty; PreviewLeechRewrite: LLM call (Haiku) to generate rewrite preview for content_quality diagnoses only, returns empty string on LLM failure (make T016 pass)
- [x] T018 [US3] Implement ApplyLeechAction in internal/memory/leech.go — rewrite (update content + re-embed, projctl-internal), narrow_scope (re-embed with narrowed content, projctl-internal), promote_to_claude_md (return Recommendation with category "claude-md-promotion", mark as action_recommended), convert_to_hook (return Recommendation with category "hook-conversion", mark as action_recommended). projctl does NOT write to CLAUDE.md, hook configs, or skill files.
- [x] T019 [US3] Expose leech diagnosis in CLI — add diagnosis output to existing optimize command or new `projctl memory diagnose` subcommand in internal/memory/memory.go and cmd/projctl/main.go. For content_quality diagnoses, call PreviewLeechRewrite to populate SuggestedContent before presenting. Collect all Recommendations from non-memory actions, print summary to stdout, offer to save full details to timestamped markdown file (e.g., `memory-recommendations-2026-02-20.md`).

**Checkpoint**: US3 complete — leech memories detected, diagnosed with root cause, remediation proposed to user.

---

## Phase 5: User Story 4 — CLAUDE.md Quality Gate (Priority: P4)

**Goal**: Gate all CLAUDE.md changes on measured effectiveness with multi-dimensional quality scoring.

**Independent Test**: Create a "working knowledge" memory effective across 3+ projects, verify quality gate checks fire, verify section-routed proposal and grade report.

**Contract**: [claude-md-quality.md](contracts/claude-md-quality.md)

### Tests for User Story 4

- [x] T020 [US4] Write failing tests for ParseClaudeMDSections (type classification: commands/architecture/gotchas/code_style/testing/other), ProposeClaudeMDChange (5 gate checks: working knowledge, universal 3+ projects, actionable, non-redundant, right tier), ScoreClaudeMD (5 dimensions with weights, grade A-F), EnforceClaudeMDBudget (over-budget demotion proposals sorted by effectiveness) in internal/memory/claudemd_quality_test.go

### Implementation for User Story 4

- [x] T021 [US4] Implement ParseClaudeMDSections in internal/memory/claudemd_quality.go — parse markdown heading structure, classify by content patterns (tables→commands, directory trees→architecture, NEVER/ALWAYS bullets→gotchas, short rules→code_style, test patterns→testing, else→other) per contract
- [x] T022 [US4] Implement ProposeClaudeMDChange in internal/memory/claudemd_quality.go — 5 quality gate checks (working quadrant, surfaced 3+ projects, Haiku actionability check, similarity-based redundancy check, Haiku tier appropriateness), section classification by content type, populate Recommendation with category "claude-md-promotion" describing what to add and where. No tool names. projctl does NOT write to CLAUDE.md (make T020 pass)
- [x] T023 [US4] Implement ScoreClaudeMD and EnforceClaudeMDBudget in internal/memory/claudemd_quality.go — scoring weights (precision 20%, faithfulness 25%, currency 20%, conciseness 15%, coverage 20%), grade scale (A=90+, B=80-89, C=70-79, D=60-69, F=<60), budget enforcement (default 100 lines, sort by effectiveness, demotion proposals as tool-agnostic Recommendations: "convert to skill", "convert to hook", "demote to memory" — no tool names, projctl does NOT edit CLAUDE.md) per contract (make T020 pass)
- [x] T024 [US4] Integrate quality gate into optimize CLI in internal/memory/memory.go — add ScoreClaudeMD to `optimize --review`, automatic budget check during optimize, collect all Recommendations from proposals, print summary to stdout, offer to save full details to timestamped markdown file

**Checkpoint**: US4 complete — CLAUDE.md promotions gated on quality, scoring reports available, budget enforced with demotion proposals.

---

## Phase 6: User Story 5 — User Feedback Commands (Priority: P5)

**Goal**: Users provide explicit feedback on surfaced memories via `/memory helpful|wrong|unclear` commands.

**Independent Test**: Surface a memory, run `/memory helpful`, verify surfacing event updated and impact score increased by +0.1.

**Contract**: [user-feedback.md](contracts/user-feedback.md)

### Tests for User Story 5

- [x] T025 [US5] Write failing tests for UpdateSurfacingFeedback (finds most recent event where haiku_relevant=true for session, updates user_feedback, returns error if no surfacing event), RecordMemoryFeedback (immediate impact adjustment: helpful +0.1, wrong -0.2, unclear -0.05, capped to [-1.0, 1.0], invalid type returns error) in internal/memory/surfacing_test.go

### Implementation for User Story 5

- [x] T026 [US5] Implement UpdateSurfacingFeedback in internal/memory/surfacing.go — find most recent surfacing_event WHERE haiku_relevant=true AND session_id=? ORDER BY timestamp DESC LIMIT 1, update user_feedback, last-wins on multiple calls per session per contract (make T025 pass)
- [x] T027 [US5] Implement RecordMemoryFeedback in internal/memory/memory.go — call UpdateSurfacingFeedback then adjust impact_score on the memory (helpful +0.1 capped at 1.0, wrong -0.2 floored at -1.0, unclear -0.05) per user-feedback contract, 1 explicit ≈ 10 implicit signals
- [x] T028 [US5] Add `projctl memory feedback --type=helpful|wrong|unclear --session-id=<id>` subcommand in cmd/projctl/main.go — validate feedback type (return error for invalid), attempt session ID derivation from environment if not provided per contract

**Checkpoint**: US5 complete — explicit feedback commands functional, impact scores adjust immediately.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Observability, validation, and cleanup across all stories

- [x] T029 Add structured logging (NFR-005) across pipeline decision points — filter decisions (kept/dropped counts per query) in internal/memory/filter.go, scoring summaries (events evaluated, scores changed) and auto-tune adjustments (old/new thresholds, trigger reason) in internal/memory/scoring.go, quadrant reclassifications (memory ID, old/new quadrant) in internal/memory/scoring.go, leech diagnosis results in internal/memory/leech.go
- [x] T030 [P] Run full test suite (`go test -tags sqlite_fts5 ./internal/memory/...`) and verify all spec acceptance scenarios (US1-US6) are exercised by at least one test
- [x] T031 [P] Validate quickstart.md — verify build command (`targ install-projctl`), test command, and all new CLI subcommands (memory feedback, memory diagnose) work per quickstart.md
- [x] T032 [P] Add benchmark tests for NFR-001 through NFR-003 in internal/memory/ — filter latency <200ms (NFR-001), synthesis latency <1s (NFR-002), ScoreSession with 50 events <5s (NFR-003). Use Go benchmark tests (`func BenchmarkX`) with mock LLM to validate timing constraints. Add cost estimation test for NFR-004: assert per-call costs × typical daily volume stays under $0.15.

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1: Foundational ─── no dependencies, start immediately
    │
Phase 2: US1 (P1) ────── depends on Phase 1 [MVP]
    │
    ├── Phase 3: US2+US6 (P2/P6) ── depends on US1 (surfacing events)
    │       │
    │       ├── Phase 4: US3 (P3) ── depends on US2 (quadrant classification)
    │       │
    │       └── Phase 5: US4 (P4) ── depends on US2 (effectiveness data)
    │
    └── Phase 6: US5 (P5) ────────── depends on US1 (surfacing CRUD) only
                                      can run PARALLEL with US2+US6

Phase 7: Polish ─────── depends on all desired stories complete
```

### User Story Dependencies

- **US1 (P1)**: Foundational only — no other story dependencies. **MVP scope.**
- **US2+US6 (P2/P6)**: Depends on US1 (needs surfacing events to score)
- **US3 (P3)**: Depends on US2 (needs quadrant classification to identify leeches)
- **US4 (P4)**: Depends on US2 (needs effectiveness data for quality gate)
- **US5 (P5)**: Depends on US1 only (needs surfacing CRUD + impact_score column from Foundational)

### Within Each User Story

1. Tests MUST be written first and FAIL before implementation
2. Interface/type definitions before implementations
3. Data layer before business logic
4. Business logic before CLI/hook integration
5. Integration tasks last (they wire everything together)

### Parallel Opportunities

- **Phase 1**: T003 can run parallel with T001+T002 (different files)
- **Phase 2**: T004+T005 parallel (different test files); T006 and T007 parallel (llm_api.go vs surfacing.go)
- **Phase 3**: T013+T014 parallel with T011+T012 (embeddings.go/memory.go vs scoring.go)
- **Phase 4+5**: US3 and US4 can run fully in parallel after US2 completes (different files, no cross-dependencies)
- **Phase 6**: US5 can run in parallel with US2+US6 (only needs US1 complete)

---

## Parallel Example: After US1 Completes

```
# US5 can start immediately (only needs US1):
Agent A: T025 → T026 → T027 → T028

# US2+US6 proceeds in parallel:
Agent B: T010 → T011 → T012 → T015
Agent C: T013 (activation formula, parallel with Agent B)
Agent D: T014 (spreading activation, parallel with Agent B)
```

## Parallel Example: After US2 Completes

```
# US3 and US4 can run fully in parallel:
Agent A: T016 → T017 → T018 → T019 (leech diagnosis)
Agent B: T020 → T021 → T022 → T023 → T024 (quality gate)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Foundational (T001-T003)
2. Complete Phase 2: US1 — Relevant Memories Only (T004-T009)
3. **STOP and VALIDATE**: Filter working, surfacing events logged, synthesis functional
4. This delivers immediate value: reduced noise in every interaction

### Incremental Delivery

1. Foundational → US1 → **Validate MVP** (filter + surfacing)
2. US2+US6 → **Validate** (scoring + quadrants + auto-scoring at session end)
3. US5 in parallel → **Validate** (feedback commands)
4. US3 + US4 in parallel → **Validate** (leech diagnosis + quality gate)
5. Polish → **Final validation**

### Parallel Team Strategy (4 agents)

1. All agents complete Foundational together
2. Agent A: US1 (MVP) → US5 (feedback, needs only US1)
3. Agent B: waits for US1 → US2+US6 (scoring)
4. After US2: Agent C: US3 (leech) | Agent D: US4 (quality gate)
5. All agents: Polish phase

---

## Notes

- [P] tasks = different files, no dependencies within phase
- [Story] label maps task to user story for traceability
- Each user story independently completable and testable at its checkpoint
- TDD required: write tests first, verify they fail, then implement
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Test command: `go test -tags sqlite_fts5 ./internal/memory/...`
- Build command: `targ install-projctl` or `go install -tags sqlite_fts5 ./cmd/projctl`
