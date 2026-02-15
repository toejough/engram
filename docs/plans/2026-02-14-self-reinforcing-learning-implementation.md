# Self-Reinforcing Learning System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the measurement, testing, and visibility layers that make the learning system self-reinforcing and verifiable.

**Architecture:**
The system builds on existing extraction/storage/retrieval infrastructure. New work instruments those pipelines with logging (what happened?), testing (does it work?), and visibility (what changed?). Five parallel workstreams: (1) logging infrastructure, (2) measurement/metrics, (3) skill testing harness, (4) quality gate wiring, (5) visibility/digest CLI.

**Tech Stack:**
Go (existing codebase), SQLite (existing embeddings DB), Anthropic direct API (DirectAPIExtractor pattern), JSONL (append-only logs), projctl CLI commands, optional: jq for diagnostics.

---

## Dependency Graph & Phasing

```
PHASE 1: Foundation (Tasks 1-3) — Logging infrastructure
  Task 1: changelog.jsonl schema + harness
  Task 2: retrievals.jsonl schema + hook instrumentation
  Task 3: metrics.jsonl schema + snapshot infrastructure
      ↓
PHASE 2: Visibility + Measurement (Tasks 11, 4-6) — Start printing feedback
  Task 11: Session-end summary printer (immediate visibility)
  Task 4: Correction recurrence detection
  Task 5: Retrieval relevance scoring
  Task 6: Hook violation trend tracking
      ↓
PHASE 3: Quality Gates (Tasks 9-10) — Prevent noise & data loss
  Task 9: LLM synthesis validation
  Task 10: CLAUDE.md safe demotion checks
      ↓
PHASE 4: Testing & Automation (Tasks 7-8) — Auto-test before deploy
  Task 7: Skill test harness (RED/GREEN via direct API)
  Task 8: Integrate testing into optimize pipeline
      ↓
PHASE 5: CLI & Filtering (Tasks 12-13) — On-demand inspection
  Task 12: `memory digest` CLI command
  Task 13: Similarity threshold filtering to queries
```

**Rationale for Order:**
- Foundation must complete first (other tasks depend on the logging layer existing)
- Print early (Task 11 after Phase 1) for feedback loop validation
- Quality gates before testing (prevent testing garbage)
- Testing after gates (only good content gets tested)
- CLI/filtering last (nice-to-haves, don't block measurement)

**Parallelization:** Tasks within a phase can run in parallel once predecessor phase completes.

---

## Implementation Tasks

### Task 1: Add changelog.jsonl logging infrastructure

**Purpose:** Every mutation (store, promote, demote, refine, prune, merge, split) gets logged.

**Files to create/modify:**
- Create: `internal/memory/changelog.go`
- Modify: `internal/memory/optimize.go` (add log calls to all mutations)
- Test: `internal/memory/changelog_test.go`

**Key points:**
- ChangelogEntry struct: timestamp, action, source_tier, destination_tier, content_id, content_summary, reason, metadata, session_id
- Append-only JSONL to `~/.claude/memory/changelog.jsonl`
- WriteChangelogEntry() function for all mutation sites
- Each optimize operation logs its action

**Testing:** Unit test verifies WriteChangelogEntry appends valid JSON. Integration test deferred.

---

### Task 2: Add retrievals.jsonl logging infrastructure

**Purpose:** Every hook-driven retrieval gets logged for relevance measurement.

**Files to create/modify:**
- Create: `internal/memory/retrieval_log.go`
- Modify: `internal/memory/query.go` (after returning results)
- Modify: `cmd/projctl/memory_query.go` (hook integration)
- Test: `internal/memory/retrieval_log_test.go`

**Key points:**
- RetrievalLogEntry struct: timestamp, hook (SessionStart/UserPromptSubmit/PreToolUse), query, results[], filtered_count, session_id, metadata
- RetrievalResult: id, content, score, tier
- LogRetrieval() function called after every query returns
- Captures what was shown and scores for later relevance analysis

**Testing:** Unit test verifies LogRetrieval writes valid JSONL.

---

### Task 3: Add metrics.jsonl snapshot infrastructure

**Purpose:** Periodic snapshots of system health metrics.

**Files to create/modify:**
- Create: `internal/memory/metrics.go`
- Test: `internal/memory/metrics_test.go`

**Key points:**
- MetricsSnapshot struct: timestamp, correction_recurrence_rate, retrieval_precision, hook_violation_trend, embedding_count, skill_count, claude_md_lines, average_correction_confidence, skills_awaiting_test, metadata
- TakeMetricsSnapshot() function computes metrics from DB and logs to `~/.claude/memory/metrics.jsonl`
- Called after optimize runs or on-demand

**Testing:** Stub test (actual computation depends on Tasks 4-6).

---

### Task 4: Implement correction recurrence detection

**Purpose:** Detect when same correction needed twice = learning failure.

**Files to create/modify:**
- Modify: `internal/memory/extract_session.go`
- Modify: `internal/memory/optimize.go` (escalation logic)
- Test: `internal/memory/extract_session_test.go`

**Key points:**
- After extracting a correction, query embeddings for prior similar corrections on same topic
- If prior correction exists AND was promoted to skill/CLAUDE.md AND correction recurs: flag as recurrent
- Add recurrence_count metadata to embedding
- Escalate recurrent corrections to skill compilation (boost priority)
- Log in changelog: action="escalate_to_skill_candidate", reason="correction recurred N times"

**Testing:** Integration test once DB available.

---

### Task 5: Implement retrieval relevance scoring

**Purpose:** Measure if retrieved memories actually prevented corrections.

**Files to create/modify:**
- Create: `internal/memory/relevance.go`
- Modify: `internal/memory/extract_session.go`
- Test: `internal/memory/relevance_test.go`

**Key points:**
- ScoreRetrievalRelevance() function: given a retrieval + subsequent corrections, determine if retrieval was relevant
- Heuristic: if no correction on that topic followed within N seconds after retrieval, it was relevant (precision=1.0)
- If correction followed: not relevant (precision=0.0)
- Use topic extraction to match retrieval results to corrections
- Compute average retrieval precision for metrics

**Testing:** Unit test with mock data. Integration test for semantic matching.

---

### Task 6: Implement hook violation trend tracking

**Purpose:** Measure whether hook violations are declining (learning working) or persistent.

**Files to create/modify:**
- Modify: `internal/memory/hooks.go` (log violations)
- Create: `internal/memory/violation_trends.go`
- Test: `internal/memory/violation_trends_test.go`

**Key points:**
- When hook check fails, write to changelog: action="hook_violation", metadata with rule, hook, count
- ComputeViolationTrends() groups violations by rule over time
- Detect trend: linear regression on violation_count per day
- Return per-rule trend (improving/stable/degrading) and recommendation
- Store in metrics snapshot

**Testing:** Unit test with synthetic violation data.

---

### Task 7: Implement skill test harness (RED/GREEN protocol)

**Purpose:** Auto-test skills before deployment via direct API.

**Files to create/modify:**
- Create: `internal/memory/skill_test_harness.go`
- Create: `internal/memory/skill_scenario.go`
- Test: `internal/memory/skill_test_harness_test.go`

**Key points:**
- TestScenario struct: description, skill_name, skill_content, success_criteria, failure_criteria
- DeriveScenarioFromEmbedding() creates realistic test scenario from embedding context
- TestSkillCandidate(scenario, runs=3): runs N times WITHOUT skill (RED), N times WITH skill (GREEN)
- Use direct Anthropic API (temperature=0.0) — no Claude Code process, no hooks, no memory retrievals
- EvaluateTestResults(): requires >=N-1 of N successes in both RED failure and GREEN success
- Return (pass bool, reasoning string)
- SkillTestResult logs response, criteria_met, errors

**Testing:** Unit test with mock API responses. Integration test for actual testing.

---

### Task 8: Integrate skill testing into optimize

**Purpose:** Wire test harness into optimize — skills tested before deploy.

**Files to create/modify:**
- Modify: `internal/memory/optimize.go` (PromoteSkillCandidate, RefineSkill, MergeSkills, SplitSkill)
- Modify: `cmd/projctl/memory_optimize.go` (add --test-skills flag, default true)
- Test: `internal/memory/optimize_test.go`

**Key points:**
- For each skill mutation (promote/refine/merge/split), derive scenario and run tests before mutating
- If tests pass: proceed with mutation, log to changelog with test results
- If tests fail: reject mutation, log rejection, don't deploy
- CLI output shows test status: "RED: 2/3 failures, GREEN: 2/3 successes → PASS → promoting"
- Configurable --test-skills flag (default true), --test-runs for N (default 3)

**Testing:** Integration test after Task 7 completes.

---

### Task 9: Implement LLM synthesis validation

**Purpose:** Validate that synthesized patterns are actionable, specific, non-redundant.

**Files to create/modify:**
- Create: `internal/memory/synthesis_validator.go`
- Modify: `internal/memory/optimize.go` (call validator after generatePattern)
- Test: `internal/memory/synthesis_validator_test.go`

**Key points:**
- SynthesisValidation struct: content, is_actionable, is_specific, is_non_redundant, quality (0.0-1.0), issues[]
- ValidateSynthesis(content, existing_patterns) checks:
  - Actionable: contains imperative keywords (always, never, use, run, add, remove)
  - Specific: >50 chars, >8 words, mentions concrete tools/patterns
  - Non-redundant: Jaccard similarity <0.8 with existing patterns
- Quality score: 1.0 if all pass, 0.33 per criterion if partial
- Quality floor: 0.8 — reject low-quality synthesis
- Don't store synthesis if quality <0.8, log rejection with issues

**Testing:** Unit test with good/bad patterns. Validation before storage.

---

### Task 10: Implement CLAUDE.md safe demotion

**Purpose:** Ensure demoted content lands in a safe destination (skill/embedding/hook).

**Files to create/modify:**
- Create: `internal/memory/demotion_safety.go`
- Modify: `internal/memory/optimize.go` (DemoteFromCLAUDEMD)
- Test: `internal/memory/demotion_safety_test.go`

**Key points:**
- DemotionPlan struct: content, destination_tier (skill/embedding/hook), reasoning, safe (bool), create_action, removal_action
- PlanCLAUDEMDDemotion(content) classifies:
  - Deterministic rules → hook (enforce, not suggest)
  - Procedural workflows → skill (reusable, TDD/git patterns)
  - Situational/narrow → embedding (retrieve when relevant)
- Safe=true only if destination is clear and creatable
- Enforce invariant: CREATE destination FIRST, then REMOVE from CLAUDE.md
- Never delete — always move content

**Testing:** Unit test demotion classification. Integration test safe removal flow.

---

### Task 11: Implement session-end summary printer

**Purpose:** Print informative feedback after every session extraction.

**Files to create/modify:**
- Create: `internal/memory/summary_printer.go`
- Modify: `cmd/projctl/memory_extract_session.go` (call printer at end)
- Test: `internal/memory/summary_printer_test.go`

**Key points:**
- SessionSummary struct: session_id, extracted_at, corrections_count, patterns_count, retrievals_count, retrievals_relevant, skill_candidates[], claude_md_demotions[], skill_refinements[]
- PrintSessionSummary(summary) outputs formatted summary with actual learnings and reasons
- Print to stdout after extraction completes
- Example format:
```
── Learning Summary ──────────────────────
Extracted:
  • correction: "Use AI-Used trailer, not Co-Authored-By" (confidence: 1.0)
  • pattern: "chi middleware ordering" (confidence: 0.7)

Retrievals: 14 this session (12 relevant, 2 filtered)

Pending optimization:
  • skill candidate: "go-test-tags" — run `projctl memory optimize` to compile
  • skill refinement: "tdd-red-producer" — retrieved 3x but correction followed
──────────────────────────────────────────
```

**Testing:** Unit test format. Integration test with real extraction.

---

### Task 12: Implement CLI digest command

**Purpose:** On-demand view of recent learnings, changes, metrics.

**Files to create/modify:**
- Create: `cmd/projctl/memory_digest.go`
- Create: `internal/memory/digest.go`
- Test: `internal/memory/digest_test.go`

**Key points:**
- DigestOptions: since (time.Duration), tier (filter), flags_only (bool), max_lines
- Digest struct: since, generated_at, recent_learnings[], metrics_snapshot, flags[]
- ComputeDigest(opts): read changelog/retrievals/metrics, filter by time/tier, detect flags
- Flags: corrections recurring, retrieval precision low, violations increasing, skills untested
- FormatDigest(): human-readable output
- CLI: `projctl memory digest --since 7d --tier skills --flags-only`

**Testing:** Integration test. Manual verification of output.

---

### Task 13: Add similarity threshold filtering

**Purpose:** Don't return poor-quality matches — prevent noise in context.

**Files to create/modify:**
- Modify: `internal/memory/query.go`
- Modify: `cmd/projctl/memory_query.go`
- Test: `internal/memory/query_test.go`

**Key points:**
- Add threshold parameter to Query() (default 0.7)
- Filter results: only return if score >= threshold
- If all results filtered: return empty array (not garbage matches)
- CLI flag: `--similarity-threshold` (default 0.7)
- Hook integration: consistent threshold across all hooks

**Testing:** Unit test filtering. Integration test with real queries.

---

## Testing Strategy

- **Unit tests:** Each task includes basic unit tests
- **Integration tests:** After predecessor tasks complete
- **End-to-end scenario:**
  1. Extract session with corrections → verify changelog entries
  2. Run optimize with skill candidate → verify skill is tested
  3. Run `memory digest` → verify output format
  4. Check metrics.jsonl for trend data

---

## Execution Notes

- **Start with Phase 1:** Tasks 1-3 are prerequisites for everything else
- **Early visibility:** After Phase 1, run Task 11 so extraction provides feedback
- **Parallel work:** Once Phase 1 completes, Tasks 4-6, 9-10 can work in parallel
- **Frequent commits:** One commit per task, review checkpoints between phases
- **Testing gates:** Each phase must have tests passing before next phase starts

---

## Open Questions

1. Should optimize run on schedule or stay manual? (Current plan: manual via `projctl memory optimize`)
2. Should test-runs be configurable? (Proposed: yes, --test-runs 3-5)
3. Should metrics snapshot be periodic or continuous? (Proposed: periodic, after optimize)
4. Should CLAUDE.md demotion be interactive or automatic? (Proposed: show plan but auto-execute once safe)
