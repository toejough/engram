# Design: PM Interview Enforcement

**Issue:** ISSUE-54
**Created:** 2026-02-04
**Status:** Draft

**Traces to:** ISSUE-54

---

## Overview

This design specifies the user experience for PM interview interactions, focusing on minimal friction while ensuring adequate user confirmation. The design implements progressive disclosure: start with high-level confirmation, drill down only when user requests clarification.

**Core principle:** User experience should be frictionless by default, detailed when needed.

---

## Design Elements

### DES-001: Progressive Disclosure Interview Pattern

The PM interview uses progressive disclosure: present high-level summary first, expand details only if user requests clarification.

**Interaction flow:**
1. PM skill presents structured summary (Problem, Solution, Acceptance criteria)
2. User can respond: `y` (confirm all), `n` (reject/rework), or `clarify` (expand details)
3. On `clarify`, skill drills into specific areas: problem statement, solution approach, acceptance criteria
4. User can request clarification on specific aspects: "clarify problem", "clarify solution", "clarify acceptance"

**Benefit:** Most issues are well-defined; user can confirm quickly. Complex issues get detailed attention.

**Traces to:** REQ-2, REQ-4

---

### DES-002: Structured Confirmation Format

The initial confirmation prompt presents information in structured list format for easy scanning.

**Format:**
```
Based on the issue, here's my understanding:

**Problem:** [Brief problem statement]
**Solution:** [High-level solution approach]
**Acceptance:** [Key acceptance criteria]

Confirm all? (y/n/clarify)
```

**Response options:**
- `y` or `yes` → Accept, move to SYNTHESIZE
- `n` or `no` → Reject, restart GATHER with different interpretation
- `clarify` → Expand into detailed questions on each section
- `clarify problem`, `clarify solution`, `clarify acceptance` → Drill into specific section

**Traces to:** REQ-2

---

### DES-003: Natural Language Skip Instructions

Users express interview skip preference using natural language at conversation start.

**Skip phrases:**
- "skip interviews"
- "skip the PM interview"
- "skip the design interview"
- "skip the arch interview"
- "no interview needed"

**User experience:**
1. User includes skip phrase in initial command or message: `projctl project start ISSUE-54 skip interviews`
2. If issue description contains sufficient information, user sees no interview questions - proceeds directly to next phase
3. If issue description lacks critical information, user still sees minimal questions asking only for fundamentally missing information
4. Skip preference means "interview only if absolutely necessary", not "never interview"

**Traces to:** REQ-3, REQ-1

---

### DES-004: Adaptive Interview Based on Available Information

Interview questions adapt to available information: when issue description is complete, user experiences brief confirmation; when details are missing, user sees targeted questions filling gaps.

**User experience:**
1. Issue with complete description → Brief summary confirmation (DES-002 structured format)
2. Issue with incomplete description → Targeted questions asking only for missing information
3. User never asked for information already available in issue or prior artifacts
4. Question flow feels natural and efficient - system "knows" what's already defined

**Benefit:** User perception is that the system intelligently uses context to minimize repetitive questions.

**Traces to:** REQ-1, REQ-4

---

### DES-005: Clarification Drill-Down

When user responds with `clarify`, the skill expands the structured summary into detailed questions.

**Drill-down sequence:**

1. **Problem clarification:**
   - "What problem are we solving?"
   - "Who experiences this problem?"
   - "What's the impact if unsolved?"

2. **Solution clarification:**
   - "What's the proposed solution approach?"
   - "What are the key components/steps?"
   - "Are there alternative approaches considered?"

3. **Acceptance clarification:**
   - "How do we know this is complete?"
   - "What are the testable criteria?"
   - "What's out of scope?"

**User can skip sections:** If user responds `clarify solution`, skip problem and acceptance questions, focus only on solution.

**Traces to:** REQ-2, REQ-4

---

### DES-006: Minimum Interaction Guarantee

User must respond to at least one confirmation question before requirements are produced (unless explicit skip preference provided).

**User experience:**
1. User starts project: `projctl project start ISSUE-54`
2. PM interview skill presents at least one question (minimum: structured confirmation from DES-002)
3. User responds to question
4. Requirements are produced
5. If user provided no response, requirements are not produced - user sees error requesting interaction

**Skip preference exception:**
- When user explicitly says "skip interviews" at start, this guarantee is waived
- User may see zero questions if issue is complete
- User may see minimal questions if critical information missing

**Traces to:** REQ-2, REQ-3

---

## User Experience Flow Diagrams

### Typical Flow (No Skip)

```
User: projctl project start ISSUE-54
  ↓
Orchestrator: Invokes pm-interview-producer
  ↓
PM Skill: [Yields need-user-input with structured summary]
  ↓
User sees:
  Problem: Orchestrator bypassed PM interview in ISSUE-53
  Solution: Enforce minimum user interaction
  Acceptance: At least one confirmation question asked

  Confirm all? (y/n/clarify)
  ↓
User: y
  ↓
PM Skill: [Yields complete with requirements.md]
  ↓
Orchestrator: Validates user-response file exists, accepts completion
```

### Skip Preference Flow

```
User: projctl project start ISSUE-54 --skip-interviews
  ↓
Orchestrator: Detects skip phrase, adds to context, invokes pm-interview-producer
  ↓
PM Skill: Reads skip_interview_preference from context
          Evaluates: Is issue description sufficient?
          Yes → Proceeds to SYNTHESIZE
          No → Yields need-user-input for critical missing info
  ↓
PM Skill: [Yields complete with requirements.md]
  ↓
Orchestrator: Sees skip preference in context, accepts completion without user-response check
```

### Clarification Flow

```
User: projctl project start ISSUE-54
  ↓
[PM Skill presents structured summary]
  ↓
User: clarify solution
  ↓
PM Skill: [Yields need-user-input with solution questions]

  What's the proposed solution approach?
  ↓
User: Orchestrator should validate user-response files exist
  ↓
PM Skill: [Yields need-user-input with follow-up]

  What are the key components?
  ↓
User: File count check, error handling for zero files
  ↓
PM Skill: Sufficient info gathered
  ↓
PM Skill: [Yields complete with requirements.md]
```

---

## Design Decisions

### Decision 1: Progressive vs. Exhaustive

**Context:** Should PM interview ask all questions upfront, or start minimal and expand?

**Choice:** Progressive disclosure (start minimal, expand on request)

**Reason:** User feedback emphasized minimal friction. Most issues are well-defined; exhaustive questioning wastes time. Progressive approach respects user's time while allowing deep dives when needed.

**Alternatives considered:**
- Exhaustive questioning every time (rejected: high friction)
- Single confirmation only (rejected: insufficient for complex issues)

**Traces to:** REQ-2

---

### Decision 2: Skip Mechanism - Flag vs. Natural Language

**Context:** How should users express skip preference?

**Choice:** Natural language detection by orchestrator

**Reason:** User explicitly chose option D in response: "Natural language: if user starts with 'skip interviews' or 'skip the X interview', tell the skill to only interview if absolutely necessary."

**Alternatives considered:**
- CLI flag `--skip-interview` (rejected: user wanted natural language)
- Issue metadata marker (rejected: requires pre-planning)
- Skill-initiated prompt "Skip interview?" (rejected: adds interaction instead of removing it)

**Traces to:** REQ-3

---

### Decision 3: Context Passing Format

**Context:** How should orchestrator communicate skip preference to skills?

**Choice:** Structured context with `skip_interview_preference` boolean field

**Reason:** Clean separation between user intent (skip preference) and orchestrator instruction (context field). Skill reads preference and decides behavior. Orchestrator doesn't instruct skill to skip, only informs of user preference.

**Alternatives considered:**
- Instruction in ARGUMENTS (rejected: violates REQ-1 context-only contract)
- Separate skill parameter (rejected: requires skill signature changes)

**Traces to:** REQ-1, REQ-3

---

## Validation Criteria

This design is validated when:

1. **DES-001 verification:** User test confirms PM interview starts with structured summary, expands when user requests "clarify"
2. **DES-002 verification:** User test confirms structured format (Problem/Solution/Acceptance) is presented to user
3. **DES-003 verification:** User test with "skip interviews" phrase shows no interview occurs (if issue complete) or minimal questions only (if info missing)
4. **DES-004 verification:** Architecture review confirms skills receive pure context without behavioral instructions
5. **DES-005 verification:** User test confirms drill-down questions appear when user responds "clarify" or "clarify [section]"
6. **DES-006 verification:** User test confirms system requires at least one user response before producing requirements (unless skip preference given)

---

## Traceability

**Upstream:** requirements.md (REQ-1, REQ-2, REQ-3, REQ-4)
**Downstream:** (Architecture phase will reference these DES-IDs)
