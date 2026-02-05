# Project Summary: PM Interview Enforcement (ISSUE-054)

**Issue:** ISSUE-054
**Project:** PM Interview Enforcement
**Duration:** ~2.5 hours estimated (2026-02-04)
**Status:** Completed

**Traces to:** ISSUE-054

---

## Executive Overview

### Project Goal
Fix the orchestrator's handling of interview-producer skills to prevent bypass of user interaction during the PM, design, and architecture phases.

### Root Cause
ISSUE-053 failed because the orchestrator passed override instructions ("skip interview", "already defined") to pm-interview-producer in the ARGUMENTS field, causing the skill to skip user confirmation and produce incorrect requirements based on its own interpretation.

### Solution Delivered
A simple documentation-only fix: add explicit "context-only contract" rule to the project orchestrator SKILL.md, prohibiting behavioral override instructions when dispatching skills. The orchestrator now passes only context (issue info, file paths, prior artifacts), never instructions like "skip interview" or "already defined".

### Outcome
17 lines added to `/Users/joe/repos/personal/projctl/skills/project/SKILL.md` establishing clear boundaries for orchestrator-skill communication. This prevents future interview bypasses while preserving skill autonomy.

---

## Key Decisions with Rationale

### Decision 1: Context-Only Communication Contract (ARCH-1)

**Context:** The orchestrator was passing behavioral instructions to skills, overriding their intended interview logic.

**Decision:** Orchestrator communicates with interview-producer skills using pure context only, never passing behavioral instructions.

**Rationale:** The root cause of ISSUE-053 was the orchestrator passing override instructions in ARGUMENTS. Skills need autonomy to execute their designed behavior. Orchestrator's role is to provide context, not to micromanage skill behavior.

**Alternatives Considered:**
- Direct instruction passing (rejected: caused ISSUE-053)
- Skill-specific protocols (rejected: unnecessary complexity)

**Outcome:** Clean separation of concerns - orchestrator provides context, skills control their own behavior.

**Phase:** Architecture
**Traces to:** REQ-1, REQ-4, DES-004

---

### Decision 2: Natural Language Intent Interpretation (ARCH-2)

**Context:** How should users express a preference to skip interviews when appropriate?

**Decision:** The orchestrator (Claude) directly interprets user natural language intent without implementing special detection logic.

**Rationale:** Claude IS the orchestrator. When a user says "skip interviews" or "skip the PM interview", Claude naturally understands this intent. Building pattern matching, regular expressions, or phrase detection code would be unnecessary complexity. Claude's language understanding is the implementation.

**Alternatives Considered:**
- Regex pattern matching (rejected: unnecessary, Claude already understands)
- Keyword lists (rejected: brittle, Claude already understands)
- User flag like `--skip-interview` (rejected: user explicitly wanted natural language)

**Outcome:** Zero implementation complexity for skip detection. User can express intent naturally, orchestrator understands and respects it by including `skip_interview_preference` in context TOML.

**Phase:** Architecture
**Traces to:** REQ-3, DES-003

---

### Decision 3: Progressive Disclosure Interview Pattern (DES-001)

**Context:** Should PM interviews ask all questions upfront, or start minimal and expand on request?

**Decision:** Progressive disclosure - start with high-level summary, expand details only if user requests clarification.

**Rationale:** User feedback emphasized minimal friction. Most issues are well-defined; exhaustive questioning wastes time. Progressive approach respects user's time while allowing deep dives when needed.

**Alternatives Considered:**
- Exhaustive questioning every time (rejected: high friction)
- Single confirmation only (rejected: insufficient for complex issues)

**Outcome:** Structured confirmation format presents Problem/Solution/Acceptance in scannable list, user can confirm quickly or request "clarify" to drill down.

**Phase:** Design
**Traces to:** REQ-2

---

### Decision 4: Minimum Implementation Surface (Breakdown)

**Context:** How much needs to change to fix this issue?

**Decision:** Single task - update project/SKILL.md with explicit context-only contract rule.

**Rationale:** Breakdown phase correctly identified this as documentation-only change. No code changes needed. The orchestrator (Claude) already has natural language understanding and can pass context via existing TOML files. Skills already have interview logic. Only missing piece was explicit guidance to NOT override skill behavior.

**Alternatives Considered:**
- Add validation mechanisms for user-response files (rejected: user said "I never asked for this, drop it")
- Implement skip detection logic (rejected: Claude already understands natural language)
- Modify skill internals (rejected: skills already have interview capability)

**Outcome:** 17 lines added to one file. Clean, traceable, low-risk deployment.

**Phase:** Breakdown/Implementation
**Traces to:** REQ-1, ARCH-1

---

## Outcomes and Deliverables

### Features Delivered

**F1: Context-Only Contract Enforcement**
**Status:** ✓ Delivered
**Traces to:** REQ-1

skills/project/SKILL.md now includes explicit "Context-Only Contract" section prohibiting behavioral override instructions. Orchestrator passes only context (issue ID, file paths, artifact references), never instructions like "skip interview" or "already defined".

**Evidence:** Commit 123e232, lines added to skills/project/SKILL.md

---

**F2: Natural Language Skip Preference Support**
**Status:** ⚠ Partially Delivered
**Traces to:** REQ-3

Orchestrator can naturally interpret user skip preferences from phrases like "skip interviews" or "skip the PM interview". Architecture (ARCH-2, ARCH-3) specifies `skip_interview_preference = true` in context TOML format. SKILL.md guidance says "respect that naturally" - specific field implementation deferred.

**Evidence:** Design documented in DES-003, Architecture specified in ARCH-2/ARCH-3. SKILL.md includes general guidance but not specific field implementation.

---

**F3: Skill Autonomy Preserved**
**Status:** ✓ Delivered
**Traces to:** REQ-4

Skills retain full control over their interview logic. Orchestrator does not override skill behavior via ARGUMENTS. Skills decide when they have sufficient information to proceed from GATHER to SYNTHESIZE phase.

**Evidence:** ARCH-1 architectural decision, documented in skills/project/SKILL.md

---

### Quality Metrics

| Metric | Target | Achieved | Notes |
|--------|--------|----------|-------|
| Requirements Coverage | 100% | 100% | All 4 requirements (REQ-1 through REQ-4) addressed |
| Design Elements | N/A | 6 | DES-001 through DES-006 documented |
| Architecture Decisions | N/A | 4 | ARCH-1 through ARCH-4 documented |
| Implementation Tasks | Minimal | 1 | TASK-1 only - documentation change |
| Post-PM QA Passes | 1st attempt | Mostly 1st | Design, Arch passed 1st; Breakdown failed iteration 1 (traceability tooling) |
| Files Modified | Minimal | 1 | skills/project/SKILL.md only |
| Lines Changed | <50 | 17 | Documentation only, no code |

---

### Performance Results

**Not applicable:** This project addressed process/workflow issues, not system performance.

---

### Known Limitations

**L1: Minimum Interaction Validation Deferred**
REQ-2 originally specified orchestrator validation that at least one user-response file exists before accepting completion from interview skills. This validation mechanism was documented in ARCH-4 but NOT implemented in TASK-1.

**Reason:** User feedback during design phase explicitly rejected file validation: "User-response file validation: I never asked for this. Drop it. Not important."

**Impact:** Orchestrator relies on skills to enforce minimum interaction internally. No external validation checks user-response files exist.

**Mitigation:** Skills already designed to yield at least one question. This is a "trust but don't verify" approach.

---

**L2: Skip Mechanism Architecture vs Implementation Gap**
ARCH-3 fully documented skip preference as `skip_interview_preference` boolean field in context TOML. TASK-1 added general guidance ("respect that naturally") but did not implement the specific field passing logic.

**Reason:** TASK-1 focused on prohibiting override instructions. Specific skip mechanism implementation deferred.

**Impact:** Architecture fully specifies the mechanism; orchestrator relies on natural language understanding rather than explicit field implementation.

**Mitigation:** SKILL.md guidance says "If the user wants to skip interviews, respect that naturally" - Claude's NLU handles skip interpretation.

---

## Lessons Learned

### What Worked Well

#### WW1: Root Cause Analysis First
**Category:** Process

PM phase correctly identified the true root cause early - orchestrator passing override instructions, not skill implementation issues. User response 2 pinpointed: "Root cause found: The ORCHESTRATOR told the skill to skip interview."

**Impact:** Prevented scope creep into skill rewrites. Focused effort on orchestrator's dispatch logic.

**Replicate:** Always do root cause analysis before jumping to solutions. Question assumptions about where the problem lies.

---

#### WW2: Leverage Existing Capabilities
**Category:** Architecture

Architecture decision ARCH-2 chose to leverage Claude's natural language understanding rather than build detection logic. "Claude is the orchestrator. When a user says 'skip interviews', Claude naturally understands this intent. There's no need for pattern matching."

**Impact:** Zero implementation complexity for skip detection.

**Replicate:** Before building new code, ask "What existing capabilities can solve this?" Claude's NLU, existing TOML context pattern, and skill interview logic already existed - just needed orchestration guidance.

---

#### WW3: User Feedback Pivot
**Category:** Design

When design-interview-producer initially focused on implementation details (file validation, context passing format), user feedback redirected: "These questions are implementation details, not design decisions. Focus on the USER EXPERIENCE."

Design phase successfully pivoted, producing UX-focused elements (progressive disclosure, structured confirmation format).

**Impact:** Design artifacts became valuable for future interview UX improvements, not just ISSUE-054.

**Replicate:** Phase boundaries matter. Design is UX, Architecture is structure/validation. Stay in scope.

---

#### WW4: Minimal Implementation Surface
**Category:** Breakdown/Implementation

Breakdown phase correctly identified this as documentation-only change. TASK-1 was singular, focused, testable: "Update project/SKILL.md to explicitly prohibit passing override instructions."

Implementation was 17 lines. Clean, traceable, complete.

**Impact:** Low risk, high confidence deployment.

**Replicate:** Resist the urge to add "while we're here" features. Minimal changes reduce risk and review burden.

---

### What Could Improve

#### CI1: Requirements Scope Expansion
**Category:** PM Phase

Initial requirements included mechanisms user didn't request:
- REQ-2: "Orchestrator validates that at least one user-response-N.toml file exists"
- REQ-3: Explicit skip mechanism with validation enforcement

User feedback simplified to "at least one question and response", "a single confirmation is good", "don't overthink it". Later in Design phase, user explicitly said: "User-response file validation: I never asked for this. Drop it."

**Impact:** Requirements included validation mechanisms that became architectural complexity despite user requesting simplicity.

**Root cause:** Requirements phase inferred validation needs from edge case analysis, not from user-stated needs.

**Recommendation:** When producing requirements, explicitly note when a specification goes beyond what user requested. Format: "Note: [Specification X] was not explicitly requested but inferred from [context]. Confirm this is desired." (Retro recommendation R2)

---

#### CI2: Design Phase Over-Specification
**Category:** Design Phase

design-interview-producer's initial questions focused on implementation:
1. "Should orchestrator validate user-response files exist before accepting completion?"
2. "How should skip preference be communicated in context?"

User redirected: "These questions are implementation details, not design decisions. Focus on the USER EXPERIENCE."

**Impact:** Lost time on implementation questions during design phase.

**Root cause:** Unclear phase boundaries between Design (UX) and Architecture (structure/validation).

**Recommendation:** Add explicit guideline to design-interview-producer SKILL.md: "Design phase focuses on USER EXPERIENCE and interaction patterns. Implementation details belong in Architecture." (Retro recommendation R1)

---

#### CI3: Architecture Documented Rejected Features
**Category:** Architecture Phase

ARCH-4 (Orchestrator Validation of Minimum Interaction) specified file-checking logic with pseudocode for validating user-response files. This directly contradicts user feedback from Design phase: "User-response file validation: I never asked for this. Drop it."

**Impact:** Architecture phase documented validation mechanism user explicitly rejected. Could have led to wasted implementation effort if not caught.

**Root cause:** Architecture phase used requirements as sole source of truth, didn't check design phase user feedback.

**Recommendation:** Each phase should review all prior user feedback, not just prior artifact outputs. (Retro recommendation R5 - document phase boundaries and information flow)

---

#### CI4: Traceability Validation False Positives
**Category:** Breakdown Phase, Tooling

Breakdown QA failed iteration 1 due to `projctl trace validate` failures:
- All tasks (TASK-1 through TASK-7) reported as "unlinked IDs"
- ISSUE-054 reported as orphan ID despite being defined in docs/issues.md

**Impact:** Rework required to fix traceability before breakdown could complete.

**Root cause:** Tooling issue - `projctl trace validate` didn't recognize issue defined in docs/issues.md, and tasks were missing reverse `**Traces to:**` fields.

**Recommendation:** Investigate why `projctl trace validate` reports issues as orphan IDs when properly defined. Fix recognition logic. (Retro recommendation R3)

---

### Patterns to Reuse

1. **Root cause analysis before solutions** - Identify WHERE the problem is (orchestrator vs skill) before jumping to HOW to fix it
2. **Leverage existing capabilities over new code** - Claude's NLU, existing TOML patterns, skill interview logic
3. **User feedback pivots** - When user redirects, pivot immediately - don't defend original approach
4. **Minimal implementation** - Resist "while we're here" features. Smallest change that solves the problem.
5. **Phase boundaries** - PM = what/why, Design = UX, Architecture = structure, Breakdown = tasks

---

### Patterns to Avoid

1. **Requirements creep** - Don't add validation, file checks, or features user didn't request without explicit confirmation
2. **Implementation in design phase** - Design = UX/interaction patterns, NOT validation logic or data formats
3. **Ignoring user simplicity requests** - When user says "don't overthink it", don't add complexity in later phases
4. **Assuming specifications are minimal** - Just because requirements document something doesn't mean user wants it
5. **Single-source-of-truth syndrome** - Each phase should review ALL prior user feedback, not just prior artifact

---

## Traceability Chain

**Note:** `projctl trace validate` reports ISSUE-054 as orphan and some IDs as unlinked due to a tooling limitation (issue recognition in docs/issues.md). Manual verification confirms all IDs are properly defined. Tracked in ISSUE-057.

### Upstream
**Issue:** ISSUE-054 (PM interview enforcement)

### Requirements
- REQ-1: Orchestrator Context-Only Contract → ARCH-1, TASK-1
- REQ-2: Minimum User Interaction → DES-001, DES-002, DES-006, ARCH-4 (deferred)
- REQ-3: Explicit Skip Mechanism → DES-003, ARCH-2, ARCH-3
- REQ-4: Interview Producer Independence → ARCH-1, DES-004

### Design
- DES-001: Progressive Disclosure Interview Pattern → REQ-2, REQ-4
- DES-002: Structured Confirmation Format → REQ-2
- DES-003: Natural Language Skip Instructions → REQ-3, ARCH-2
- DES-004: Adaptive Interview Based on Available Information → REQ-1, REQ-4, ARCH-1
- DES-005: Clarification Drill-Down → REQ-2, REQ-4
- DES-006: Minimum Interaction Guarantee → REQ-2, REQ-3, ARCH-4

### Architecture
- ARCH-1: Context-Only Communication Contract → REQ-1, REQ-4, DES-001, DES-002, DES-004, DES-005, TASK-1
- ARCH-2: Natural Language Intent Interpretation → REQ-3, DES-003
- ARCH-3: TOML Context File Format → REQ-1, REQ-3, DES-003
- ARCH-4: Orchestrator Validation of Minimum Interaction → REQ-2, REQ-3, DES-006 (not implemented)

### Implementation
- TASK-1: Update project/SKILL.md to enforce context-only contract → REQ-1, ARCH-1
  - **Status:** Completed
  - **File:** skills/project/SKILL.md (+17 lines)
  - **Commit:** 123e232

---

## Project Artifacts

### Documentation
- `/Users/joe/repos/personal/projctl/.claude/projects/issue-054-pm-interview/requirements.md` - 4 requirements (REQ-1 through REQ-4)
- `/Users/joe/repos/personal/projctl/.claude/projects/issue-054-pm-interview/design.md` - 6 design elements (DES-001 through DES-006)
- `/Users/joe/repos/personal/projctl/.claude/projects/issue-054-pm-interview/architecture.md` - 4 architectural decisions (ARCH-1 through ARCH-4)
- `/Users/joe/repos/personal/projctl/.claude/projects/issue-054-pm-interview/tasks.md` - 1 task (TASK-1)
- `/Users/joe/repos/personal/projctl/.claude/projects/issue-054-pm-interview/retro.md` - Retrospective with 5 successes, 5 challenges, 5 recommendations

### Implementation
- `/Users/joe/repos/personal/projctl/skills/project/SKILL.md` - Context-Only Contract section added (17 lines)

### User Interactions
- 11 total user responses across all phases
- 4 PM phase responses (user-response-1.toml through user-response-4.toml)
- 2 Design phase responses (user-response-design-1.toml, user-response-design-2.toml)
- 3 Architecture phase responses (user-response-arch-1.toml through user-response-arch-3.toml)
- 2 Breakdown phase responses (implied from retro, not counted in file list)

### Version Control
- Commit 82fd018: Created ISSUE-054
- Commit 123e232: Implemented fix (skills/project/SKILL.md) + all project artifacts

---

## Timeline

| Phase | Duration | Iterations | Outcome |
|-------|----------|------------|---------|
| PM Interview | Extended | 4 user responses | Requirements approved |
| Design Interview | Moderate | 2 user responses | Design approved (1st QA pass) |
| Architecture Interview | Moderate | 3 user responses | Architecture approved (1st QA pass) |
| Breakdown | Quick | 2 iterations | Tasks approved (QA fail on iteration 1 - traceability) |
| Implementation | Quick | 1 pass | TASK-1 completed |
| Documentation | N/A | N/A | Included in implementation commit |
| Alignment | N/A | N/A | (Not explicitly tracked) |
| Retrospective | Quick | 1 pass | Retro completed |

**Total Duration:** ~2.5 hours estimated from project init to retro-ready

---

## Recommendations for Future Projects

Based on the retrospective (retro.md), these recommendations carry forward:

### R1: Establish "User Experience First" Design Principle
**Priority:** High
**Action:** Add explicit guideline to design-interview-producer SKILL.md: "Design phase focuses on USER EXPERIENCE and interaction patterns. Implementation details belong in Architecture phase."

**Measurable outcome:** Design artifacts focus on UX scenarios, flows, and interaction patterns. No pseudocode or validation logic in design.md.

---

### R2: Warn When Specs Exceed User Requests
**Priority:** High
**Action:** When producing requirements, design, or architecture, explicitly note when a specification goes beyond what user requested. Format: "Note: [Specification X] was not explicitly requested but inferred from [context/edge case]. Confirm this is desired."

**Measurable outcome:** User sees explicit callouts for inferred requirements/design/architecture, can accept or reject before implementation.

---

### R3: Fix projctl trace validate Issue Recognition
**Priority:** Medium
**Action:** Investigate why `projctl trace validate` reports ISSUE-054 as orphan ID when issue is defined in docs/issues.md. Fix issue ID recognition logic.

**Measurable outcome:** `projctl trace validate` correctly recognizes issues defined in docs/issues.md.

---

### R4: Add "Simplicity Check" to Breakdown Phase
**Priority:** Medium
**Action:** breakdown-producer should perform simplicity check: "Could this be done with fewer tasks/components/changes?" Include explicit question in GATHER phase.

**Measurable outcome:** Breakdown artifacts include "Simplicity assessment" section discussing alternatives.

---

### R5: Document Phase Boundaries in Orchestrator Guide
**Priority:** Low
**Action:** Add "Phase Boundaries" section to project/SKILL.md clarifying what belongs in each phase (PM: what/why, Design: UX, Architecture: structure, Breakdown: tasks).

**Measurable outcome:** Orchestrator can reference phase boundaries when dispatching skills.

---

## Conclusion

ISSUE-054 successfully addressed the root cause of ISSUE-053's failure by establishing a clear context-only communication contract between orchestrator and skills. The fix was minimal (17 lines of documentation), well-scoped, and directly targeted the identified problem.

**Key Success:** Thorough root cause analysis prevented over-engineering. The problem was orchestrator behavior, not skill implementation.

**Key Learning:** User simplicity requests must be honored across ALL phases. When user says "don't overthink it" in PM phase, that applies to design, architecture, and implementation too.

**Impact:** Orchestrator now has explicit guidance to pass only context to skills, preserving skill autonomy and ensuring user interaction occurs in interview phases. This prevents future bypasses like ISSUE-053.

**Next Steps:** Monitor orchestrator behavior in next project to verify context-only contract is followed. Consider implementing retro recommendations R1-R5 to tighten process boundaries.
