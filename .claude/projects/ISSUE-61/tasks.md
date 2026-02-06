# ISSUE-61 Tasks: Adaptive Interview Depth

**Issue:** ISSUE-61 - Decision needed: What's the right PM interview depth?

## Dependency Graph

```
TASK-1 (shared pattern doc)
    |
    +-----> TASK-2 (key questions registry)
    |           |
    |           +-----> TASK-3 (coverage calculation)
    |                       |
    +-----> TASK-4 (context gathering)
            |
            +-----> TASK-5 (gap assessment)
                        |
                        +-----> TASK-6 (adaptive interview logic)
                                    |
                                    +-----> TASK-7 (yield enrichment)
                                                |
                                                +-----> TASK-8 (integration tests)

TASK-9 (validation on real issues) - depends on TASK-8
```

---

## Tasks

### TASK-1: Create shared interview pattern documentation

**Description:** Document the standardized GATHER → ASSESS → INTERVIEW → SYNTHESIZE → PRODUCE flow that all interview skills will follow. This establishes the contract for adaptive interview depth.

**Status:** Ready

**Acceptance Criteria:**
- [ ] File created at `~/.claude/skills/shared/INTERVIEW-PATTERN.md`
- [ ] Documents five-phase flow with clear phase boundaries
- [ ] Includes context gathering mechanism descriptions (territory, memory, context-explorer)
- [ ] Defines gap assessment approach (coverage calculation, depth tiers)
- [ ] Specifies error handling patterns for context failures
- [ ] Includes yield context enrichment format with gap analysis metadata
- [ ] Documents when to yield `blocked`, `need-decision`, or continue with partial context

**Files:** `~/.claude/skills/shared/INTERVIEW-PATTERN.md`

**Dependencies:** None

**Traces to:** ARCH-004, REQ-004

---

### TASK-2: Add key questions registry to arch-interview-producer

**Description:** Add structured "Key Questions" section to arch-interview-producer SKILL.md that defines the 8-12 questions representing minimum viable context for architecture phase, with priority levels (critical/important/optional).

**Status:** Ready

**Acceptance Criteria:**
- [ ] `## Key Questions` section added to `arch-interview-producer/SKILL.md`
- [ ] 8-12 questions defined covering: technology stack, scale, deployment, integrations, performance, security, data durability, observability
- [ ] Each question tagged with priority: critical (2-4), important (3-5), or optional (2-3)
- [ ] Question format includes topic, question text, and priority level
- [ ] Coverage weight values documented: critical=-15%, important=-10%, optional=-5%
- [ ] Examples provided showing how questions map to architecture decisions

**Files:** `~/.claude/skills/arch-interview-producer/SKILL.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-005, REQ-003, REQ-004

---

### TASK-3: Implement coverage calculation logic

**Description:** Create pure function that calculates coverage percentage based on key questions, answered questions, and priority weights. This is the core algorithm for gap assessment.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Function accepts: list of key questions (with priorities), list of answered questions
- [ ] Returns: coverage percentage (0-100), gap size classification (small/medium/large), list of unanswered questions
- [ ] Applies priority weights: critical unanswered = -15%, important = -10%, optional = -5%
- [ ] Classifies gaps: ≥80% = small, 50-79% = medium, <50% = large
- [ ] Handles edge case: when total coverage <20%, returns "large" regardless of calculation
- [ ] Unit tests cover: all answered (100%), all unanswered (0%), mixed priorities, edge case
- [ ] Tests use property-based testing for weight application correctness

**Files:** `~/.claude/skills/arch-interview-producer/SKILL.md` (embedded bash function or documented algorithm)

**Dependencies:** TASK-2

**Traces to:** ARCH-002, ARCH-005, REQ-003

---

### TASK-4: Implement context gathering phase

**Description:** Implement GATHER phase that uses territory map, memory query, and context-explorer to collect existing information before asking user questions.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Code added to arch-interview-producer that runs before yielding `need-user-input`
- [ ] Executes `projctl territory map` to get file structure
- [ ] Executes `projctl memory query` with domain-specific queries (e.g., "architecture decisions", "technology stack")
- [ ] Parses territory and memory results into structured data
- [ ] Handles errors: territory map failure yields `blocked`, memory timeout continues with note
- [ ] Logs context sources used and information gathered
- [ ] Context stored in yield metadata for traceability

**Files:** `~/.claude/skills/arch-interview-producer/SKILL.md`

**Dependencies:** TASK-1

**Traces to:** ARCH-001, REQ-002

---

### TASK-5: Implement gap assessment phase

**Description:** Implement ASSESS phase that compares gathered context against key questions to determine coverage percentage and gap size.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Code added after context gathering, before interview
- [ ] For each key question, determines if answerable from: issue description, territory map, memory results, context files
- [ ] Uses coverage calculation function (TASK-3) to compute gap metrics
- [ ] Logs assessment results: total questions, answered count, coverage %, gap size
- [ ] Handles contradictory context: yields `need-decision` with conflicts if detected
- [ ] Stores gap assessment in yield metadata for observability

**Files:** `~/.claude/skills/arch-interview-producer/SKILL.md`

**Dependencies:** TASK-3, TASK-4

**Traces to:** ARCH-002, ARCH-003, REQ-002, REQ-003

---

### TASK-6: Implement adaptive interview logic

**Description:** Implement INTERVIEW phase that adjusts question count based on gap size: small gap = 1-2 questions, medium = 3-5, large = 6+.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Code added after gap assessment, before yield
- [ ] Small gap (≥80%): yields 1-2 confirmation questions for critical unanswered items only
- [ ] Medium gap (50-79%): yields 3-5 clarification questions prioritizing critical then important
- [ ] Large gap (<50%): yields full interview sequence covering all key questions
- [ ] Question text references gathered context where relevant (e.g., "I see X in docs, confirm Y?")
- [ ] Questions skip topics fully answered by context
- [ ] Question count matches documented depth tier

**Files:** `~/.claude/skills/arch-interview-producer/SKILL.md`

**Dependencies:** TASK-5

**Traces to:** ARCH-002, ARCH-004, REQ-003

---

### TASK-7: Add yield context enrichment

**Description:** Enrich yield `[context]` section with gap analysis metadata for observability and debugging.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Yield includes `[context.gap_analysis]` section
- [ ] Fields: `total_key_questions`, `questions_answered`, `coverage_percent`, `gap_size`, `question_count`
- [ ] Field: `sources` array listing which mechanisms provided context (territory/memory/context-files)
- [ ] Field: `unanswered_critical` listing any critical questions not answered (for debugging)
- [ ] Yield validates against schema before output
- [ ] Example yield included in SKILL.md documentation

**Files:** `~/.claude/skills/arch-interview-producer/SKILL.md`

**Dependencies:** TASK-6

**Traces to:** ARCH-003, REQ-003

---

### TASK-8: Create integration tests for adaptive flow

**Description:** Create test suite that validates end-to-end adaptive interview behavior with mocked context scenarios.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Test file created: `~/.claude/skills/arch-interview-producer/SKILL_test.sh`
- [ ] Test case: sparse context (0% coverage) → yields large gap, 6+ questions
- [ ] Test case: medium context (65% coverage) → yields medium gap, 3-5 questions
- [ ] Test case: rich context (90% coverage) → yields small gap, 1-2 questions
- [ ] Test case: contradictory context → yields `need-decision` with conflicts
- [ ] Test case: territory map failure → yields `blocked` with diagnostic
- [ ] Each test validates: question count, gap metrics in yield, question relevance
- [ ] Tests use fixture files for reproducibility

**Files:** `~/.claude/skills/arch-interview-producer/SKILL_test.sh`

**Dependencies:** TASK-7

**Traces to:** REQ-002, REQ-003, REQ-004

---

### TASK-9: Validate pattern on real issues

**Description:** Use updated arch-interview-producer on 2-3 real issues to validate that gap assessment produces sensible depth decisions and gathered context avoids redundant questions.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Pattern applied to at least 2 different issue types (new feature, refactoring)
- [ ] For each validation: document issue ID, gap size, question count, user feedback
- [ ] Verify: context gathering completed without errors
- [ ] Verify: question count matched expected depth tier
- [ ] Verify: questions felt appropriate (not redundant, not too sparse)
- [ ] Verify: yield metadata enabled debugging when needed
- [ ] Document any adjustments needed to key questions or weights
- [ ] Validation summary added to ISSUE-61 retrospective

**Files:** `.claude/projects/ISSUE-61/validation-log.md`

**Dependencies:** TASK-8

**Traces to:** ARCH-006, REQ-004

---

## Notes

**Visual Tasks:** None (all tasks are skill implementation logic with no UI/CLI output formatting)

**Parallel Opportunities:**
- TASK-2 and TASK-4 can be developed in parallel after TASK-1
- TASK-3 is independent once TASK-2 defines question format

**Rollout Strategy:** This task breakdown implements ARCH-006 phase 1-2 (document pattern, update arch-interview-producer). Subsequent rollout to pm-interview-producer and design-interview-producer will be tracked in separate issues after validation completes.

**Testing Philosophy:** Coverage calculation (TASK-3) is pure and thoroughly tested. Integration tests (TASK-8) validate the full flow. Validation (TASK-9) provides real-world feedback for pattern refinement.
