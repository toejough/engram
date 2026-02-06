# Retrospective: PM Interview Enforcement (ISSUE-54)

**Issue:** ISSUE-54
**Created:** 2026-02-04
**Completed:** 2026-02-04
**Duration:** ~2.5 hours

**Traces to:** ISSUE-54

---

## Project Summary

### Scope
Fix orchestrator's handling of interview-producer skills to prevent bypass of user interaction. Root cause analysis revealed ISSUE-53 failed because orchestrator passed override instructions ("skip interview", "already defined") to pm-interview-producer, causing it to skip user confirmation and produce wrong requirements.

### Deliverables
- Requirements: 4 requirements (context-only contract, minimum interaction, skip mechanism, skill independence)
- Design: 6 design elements (progressive disclosure, structured confirmation, natural language skip, adaptive interview, drill-down, minimum interaction guarantee)
- Architecture: 4 architectural decisions (context-only communication, natural language interpretation, TOML context format, minimum interaction validation)
- Implementation: 1 task - add context-only contract rule to project/SKILL.md
- Result: 17 lines added to skills/project/SKILL.md explicitly prohibiting behavioral override instructions

### Team
- Orchestrator: Claude (project management)
- Skills: pm-interview-producer, design-interview-producer, arch-interview-producer, breakdown-producer, tdd-red-producer, tdd-green-producer, tdd-refactor-producer
- QA: pm-qa, design-qa, architect-qa, breakdown-qa, task-audit
- User: Joe (requirements clarification, design direction, architecture decisions)

---

## What Went Well (Successes)

### S1: Root Cause Analysis Depth
**Area:** PM Phase

The PM interview correctly identified the true root cause early. User response 2 pinpointed: "Root cause found: The ORCHESTRATOR told the skill to skip interview. Evidence: ARGUMENTS passed said 'not conduct a new interview' and 'problem and solution are already defined'."

This prevented scope creep into skill rewrites and focused effort on the orchestrator's dispatch logic.

**Impact:** Saved significant implementation time by targeting the right component.

---

### S2: User Feedback Integration
**Area:** Design Phase

When design-interview-producer initially focused on implementation details (file validation, context passing format), user feedback (user-response-design-1) redirected: "These questions are implementation details, not design decisions. Focus on the USER EXPERIENCE."

Design phase pivoted successfully, producing UX-focused elements (progressive disclosure, structured confirmation format, minimal friction path).

**Impact:** Design artifacts became valuable for future interview UX improvements, not just ISSUE-54.

---

### S3: Pragmatic Architecture
**Area:** Architecture Phase

Architecture decision ARCH-2 chose to leverage Claude's natural language understanding rather than build detection logic: "Claude is the orchestrator. When a user says 'skip interviews', Claude naturally understands this intent. There's no need for pattern matching, regular expressions, or phrase detection code."

**Impact:** Zero implementation complexity for skip detection - Claude already handles it.

---

### S4: Minimal Implementation Surface
**Area:** Breakdown/Implementation

Breakdown phase correctly identified this as documentation-only change. TASK-1 was singular, focused, and testable: "Update project/SKILL.md to explicitly prohibit passing override instructions."

Implementation was 17 lines added to one file. Clean, traceable, complete.

**Impact:** Low risk, high confidence deployment.

---

### S5: First-Pass QA Success (Post-Requirements)
**Area:** Design, Architecture, Implementation

After requirements were approved, Design QA, Architecture QA, and Task Audit all passed on first attempt. No rework loops after PM phase.

**Impact:** Efficient phase transitions once requirements were solid.

---

## What Could Improve (Challenges)

### C1: Requirements Scope Expansion
**Area:** PM Phase

Initial requirements (REQ-2, REQ-3) included mechanisms user didn't request:
- REQ-2: "Orchestrator validates that at least one user-response-N.toml file exists"
- REQ-3: Explicit skip mechanism with validation enforcement

User feedback in PM phase (user-response-1) simplified to "at least one question and response", "a single confirmation is good", "don't overthink it".

Later in Design phase, user explicitly said: "User-response file validation: I never asked for this. Drop it. Not important."

**Impact:** Requirements included validation mechanisms that became architectural complexity despite user requesting simplicity.

---

### C2: Design Phase Over-Specification
**Area:** Design Phase

Design-interview-producer's initial questions focused on implementation:
1. "Should orchestrator validate user-response files exist before accepting completion?"
2. "How should skip preference be communicated in context?"
3. "Where should natural language skip detection live?"

User redirected: "These questions are implementation details, not design decisions. Focus on the USER EXPERIENCE."

**Impact:** Lost time on implementation questions during design phase. Design phase should focus on user interaction patterns, not validation logic.

---

### C3: Architecture Added Unwanted Complexity
**Area:** Architecture Phase

ARCH-4 (Orchestrator Validation of Minimum Interaction) specified file-checking logic:
```
files = list_files(project_dir, "user-response-*.toml")
if skip_interview_preference == true:
    accept_completion()
else if len(files) > 0:
    accept_completion()
```

This directly contradicts user feedback from Design phase: "User-response file validation: I never asked for this. Drop it."

**Impact:** Architecture phase documented validation mechanism user explicitly rejected. This could have led to wasted implementation effort if not caught.

---

### C4: Breakdown QA Traceability Failure
**Area:** Breakdown Phase

Breakdown QA failed on iteration 2 due to traceability validation issues:
- All tasks (TASK-1 through TASK-7 initially) reported as "unlinked IDs"
- ISSUE-54 reported as orphan ID
- `projctl trace validate` failed, blocking completion per SKILL guidelines

**Root cause:** Tasks were missing reverse `**Traces to:**` fields linking them back into the traceability chain.

**Impact:** Rework required to fix traceability before breakdown could complete. This was a technical/tooling issue, not a conceptual breakdown problem.

---

### C5: Multiple Interview Iterations in PM Phase
**Area:** PM Phase

PM phase required 4 user responses before requirements were approved. While thorough, this suggests either:
1. Initial questions weren't focused enough, OR
2. Issue description (ISSUE-54) wasn't sufficiently detailed for quick convergence

**Evidence:** User responses 1-2 clarified scope and root cause. User responses 3-4 confirmed simplicity preference.

**Impact:** Extended PM phase duration. Not necessarily bad (thoroughness is valuable), but worth examining if questions could be more targeted upfront.

---

## Process Improvement Recommendations

### R1: Establish "User Experience First" Design Principle
**Priority:** High

**Action:** Add explicit guideline to design-interview-producer SKILL.md: "Design phase focuses on USER EXPERIENCE and interaction patterns. Implementation details (file formats, validation logic, data structures) belong in Architecture phase."

**Rationale:** Design phase in ISSUE-54 initially asked implementation questions, requiring user redirect. Clear phase boundaries prevent this.

**Measurable outcome:** Design artifacts focus on UX scenarios, flows, and interaction patterns. No pseudocode or validation logic in design.md.

**Traces to:** C2

---

### R2: Warn When Specs Exceed User Requests
**Priority:** High

**Action:** When producing requirements, design, or architecture, explicitly note when a specification goes beyond what user requested. Format: "Note: [Specification X] was not explicitly requested but inferred from [context/edge case/best practice]. Confirm this is desired."

**Rationale:** REQ-2 validation mechanism, ARCH-4 file checking, and other features were added without user requesting them. User had to explicitly reject these during interviews.

**Measurable outcome:** User sees explicit callouts for inferred requirements/design/architecture, can accept or reject before implementation.

**Traces to:** C1, C3

---

### R3: Fix projctl trace validate Issue Recognition
**Priority:** Medium

**Action:** Investigate why `projctl trace validate` reports ISSUE-54 as orphan ID when issue is defined in docs/issues.md at line 2382. Fix issue ID recognition logic.

**Rationale:** Breakdown QA failed due to traceability validation issues. If the tool doesn't recognize properly-defined issues, it creates false failure signals and rework.

**Measurable outcome:** `projctl trace validate` correctly recognizes issues defined in docs/issues.md and doesn't report them as orphan IDs.

**Traces to:** C4

---

### R4: Add "Simplicity Check" to Breakdown Phase
**Priority:** Medium

**Action:** Breakdown-producer should perform simplicity check: "Could this be done with fewer tasks/components/changes?" Include explicit question in GATHER phase: "Is there a simpler approach that achieves the same outcome?"

**Rationale:** ISSUE-54 was ultimately a 17-line documentation change. If breakdown had asked "Is there a simpler approach?", it might have caught the validation mechanism complexity earlier.

**Measurable outcome:** Breakdown artifacts include explicit "Simplicity assessment" section discussing alternatives considered and why current approach is appropriately scoped.

**Traces to:** C1, C3

---

### R5: Document Phase Boundaries in Orchestrator Guide
**Priority:** Low

**Action:** Add "Phase Boundaries" section to project/SKILL.md or orchestrator documentation clarifying what belongs in each phase:
- PM: What problem, what solution, what acceptance criteria (user-facing outcomes)
- Design: User experience, interaction patterns, flows
- Architecture: System structure, component decisions, data formats, validation logic
- Breakdown: Task decomposition, implementation order

**Rationale:** PM, Design, and Architecture phases in ISSUE-54 all had some scope bleed (PM added validation, Design asked implementation questions, Architecture documented rejected features).

**Measurable outcome:** Orchestrator can reference phase boundaries when dispatching skills. Skills have clearer focus.

**Traces to:** C1, C2, C3

---

## Open Questions

### Q1: Should Skills Have "Scope Creep" Detection?

**Context:** Throughout ISSUE-54, requirements, design, and architecture all added features user didn't request. User had to explicitly reject these during interviews.

**Question:** Should interview-producer skills have built-in "scope creep" detection that flags when a specification exceeds the issue description or user requests?

**Possible approaches:**
1. Skills explicitly ask: "These items weren't in the issue. Should I include them?"
2. Skills mark inferred items with "[INFERRED]" tag for user review
3. No change - rely on user to catch and reject during interviews

**Impact:** Could reduce interview iterations and prevent over-engineering.

**Traces to:** C1, C3

---

### Q2: Should Traceability Validation Be Required or Advisory?

**Context:** Breakdown phase was blocked due to `projctl trace validate` failures related to tooling issues (issue ID recognition, unlinked task IDs).

**Question:** Should traceability validation be a blocking requirement (fail if validation fails) or advisory (warn but allow completion)?

**Tradeoffs:**
- Blocking: Ensures clean traceability but vulnerable to tooling bugs (false negatives)
- Advisory: Allows progress despite tooling issues but risks incomplete traceability chains

**Current state:** Breakdown SKILL guidelines treat `projctl trace validate` as blocking requirement.

**Traces to:** C4

---

### Q3: What's the Right Balance for PM Interview Depth?

**Context:** PM phase required 4 user responses. This was thorough but potentially longer than needed.

**Question:** Should PM interview aim for:
1. Minimal interaction (1-2 questions, quick confirmation)
2. Thorough exploration (3-5+ questions, deep understanding)
3. Adaptive depth (quick for simple issues, deep for complex ones)

**Consideration:** ISSUE-54 was conceptually simple (add documentation rule) but had nuance (skip mechanism, validation). More complex issues might benefit from deeper interviews upfront.

**Traces to:** C5

---

## Metrics

| Metric | Value | Notes |
|--------|-------|-------|
| Project Duration | ~2.5 hours | Init to retro-ready |
| Total Phases | 8 | PM, Design, Arch, Breakdown, Implementation, Documentation, Alignment, Retro |
| QA Iterations (Post-PM) | 0 | All phases passed first QA after requirements approved |
| QA Iterations (PM) | Unknown | PM phase had 4 user responses but iteration count not tracked |
| QA Iterations (Breakdown) | 1 | Breakdown QA failed iteration 1, passed iteration 2 (traceability) |
| Requirements Count | 4 | REQ-1 through REQ-4 |
| Design Elements | 6 | DES-001 through DES-006 |
| Architecture Decisions | 4 | ARCH-1 through ARCH-4 |
| Tasks | 1 | TASK-1 only (initially 7 tasks in breakdown iteration 1) |
| Files Modified | 1 | skills/project/SKILL.md |
| Lines Added | 17 | Documentation only |
| Blockers | 0 | No hard blockers encountered |
| User Responses | 11 | 4 PM, 2 Design, 3 Arch, 2 Breakdown |

---

## Traceability

**Issue:** ISSUE-54
**Requirements:** REQ-1, REQ-2, REQ-3, REQ-4
**Design:** DES-001, DES-002, DES-003, DES-004, DES-005, DES-006
**Architecture:** ARCH-1, ARCH-2, ARCH-3, ARCH-4
**Tasks:** TASK-1
**Commits:** 123e232 ("fix(orchestrator): add context-only contract for skill dispatch")

---

## Lessons Learned

### Positive Patterns to Replicate

1. **Root cause analysis first**: PM phase identified orchestrator as culprit, not skills. Prevented wasted skill rewrites.
2. **User feedback pivots**: Design phase successfully pivoted from implementation to UX focus when redirected.
3. **Leverage existing capabilities**: Architecture chose to use Claude's NLU rather than build detection logic.
4. **Minimal implementation**: Breakdown correctly identified 1-task, documentation-only change.

### Patterns to Avoid

1. **Requirements creep**: Don't add validation mechanisms, file checks, or other features user didn't request without explicit confirmation.
2. **Implementation in design**: Design phase should focus on user experience, not validation logic or data formats.
3. **Ignoring user simplicity requests**: When user says "don't overthink it", don't add complexity in later phases.
4. **Assuming specifications are minimal**: Just because requirements or design document something doesn't mean user wants it implemented.

---

## Summary

ISSUE-54 successfully addressed the root cause of ISSUE-53's failure: orchestrator passing override instructions to interview-producer skills. The fix was minimal (17 lines in project/SKILL.md) and well-scoped.

**Key Success:** Root cause analysis was thorough and targeted the right component (orchestrator, not skills).

**Key Challenge:** Multiple phases added complexity and validation mechanisms user didn't request, requiring explicit rejection during interviews. Process could be tightened with "scope creep" detection and clearer phase boundaries.

**Outcome:** Orchestrator now has explicit context-only contract, preventing future interview bypasses. User interaction is guaranteed for all interview phases going forward.
