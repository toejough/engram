---
name: pm-interview
description: Structured problem discovery interview producing requirements with traceability IDs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# PM Interview Skill

Interview users to define problems and desired outcomes, then produce a structured product specification with traceability IDs.

## Purpose

Guide a discovery conversation that produces a clear product specification without diving into implementation details. The output is a requirements.md file that captures:
- The problem being solved
- Current state of the world
- Desired future state
- Success criteria and edge cases
- Solution guidance (not implementation)

Every requirement, user story, and acceptance criterion gets a `REQ-NNN` traceability ID.

## Domain Ownership

This skill owns the **problem space**: understanding what problem we're solving and for whom.

**Owns:**
- What problem exists and who has it
- Why it matters (impact, frequency, severity)
- What success looks like (measurable outcomes)
- Constraints on solutions (budget, timeline, compliance)
- Priority of different aspects (P0/P1/P2)

**Does NOT own (defer to other phases):**
- How users interact with solutions → Design
- How solutions are built technically → Architecture
- Specific UI layouts, flows, or screens → Design
- Technology choices, data models, APIs → Architecture

When users raise out-of-scope concerns, acknowledge them, note for the appropriate phase, and redirect: "That sounds like a Design/Architecture concern. I'll note it for that phase. For now, let's focus on understanding the problem."

## Interview Phases

### 1. PROBLEM - Understand the Pain

**Goal:** Identify what's broken, frustrating, or missing.

**Questions to explore:**
- What problem are you trying to solve?
- Who experiences this problem?
- How often does it occur?
- What triggers it?
- What's the impact when it happens?
- Why hasn't it been solved already?

**Listen for:**
- Symptoms vs. root causes
- Frequency and severity
- Workarounds currently in use
- Emotional language (frustration, confusion, fear)

**Do NOT proceed until you can articulate the problem in one sentence.**

### 2. CURRENT STATE - Map the Present

**Goal:** Understand how things work today.

**Questions to explore:**
- Walk me through what happens today
- What tools/processes are involved?
- What works well that we should preserve?
- What are the pain points in the current flow?
- What data or context is available?
- Who are the actors involved?

**Listen for:**
- Steps in the current workflow
- Integration points
- Dependencies and constraints
- What's manual vs. automated

**Capture:**
- User journey map (steps, actors, decisions)
- Current limitations
- Existing assets to leverage

### 3. FUTURE STATE - Define Success

**Goal:** Describe what the world looks like when the problem is solved.

**Questions to explore:**
- If this problem were solved, what would be different?
- How would you know it's working?
- What would users be able to do that they can't today?
- What would no longer happen?
- What does "good enough" look like vs. "ideal"?

**Listen for:**
- Measurable outcomes
- Behavioral changes
- Removed friction
- New capabilities

**Capture:**
- Success criteria (testable statements)
- User stories (As a... I want... So that...)
- Acceptance criteria

### 4. EDGE CASES - Anticipate Complexity

**Goal:** Identify boundary conditions and exceptional scenarios.

**Questions to explore:**
- What could go wrong?
- What happens at the extremes? (0, 1, many)
- What about partial success or failure?
- What if inputs are missing or malformed?
- What about concurrent or conflicting actions?
- What should never happen?

**Listen for:**
- Error scenarios
- Race conditions
- Permission boundaries
- Data edge cases

**Capture:**
- List of edge cases with expected behavior
- Invariants that must always hold
- Failure modes and recovery

### 5. GUIDANCE - Gather Constraints

**Goal:** Collect direction on solution space without designing the solution.

**Questions to explore:**
- Are there approaches you've considered?
- Are there approaches you want to avoid?
- What constraints exist? (time, tech, team, budget)
- What's negotiable vs. non-negotiable?
- Are there similar solutions to reference?
- What's the timeline expectation?

**Listen for:**
- Technical constraints
- Business constraints
- Preferences and anti-patterns
- Prior art references

**Do NOT propose solutions. Capture guidance only.**

### 6. SYNTHESIZE - Write the Spec

**Goal:** Produce a structured product specification document with traceability IDs.

**Output file:** `requirements.md` in the project directory (or as specified by orchestrator context).

**Document structure:**

```markdown
# <Feature Name> - Product Specification

## Problem Statement
<One paragraph describing the core problem>

### Who is affected
<Actors and their relationship to the problem>

### Impact
<What happens if we don't solve this>

## Current State

### User Journey
<Steps in the current workflow>

### Pain Points
<Specific friction points>

### Constraints
<Technical or business limitations>

## Desired Future State

### Success Criteria
- **SC-01 (REQ-001):** <Measurable outcome>
- **SC-02 (REQ-002):** <Measurable outcome>
...

### User Stories (with explicit actions)
- **US-01 (REQ-NNN):** As a... I want... So that...
  - **Action:** User clicks/taps/types [specific element]
  - **Result:** [Observable outcome - what user sees/experiences]
- **US-02 (REQ-NNN):** ...
...

### Acceptance Criteria (Given/When/Then format REQUIRED)

**Every acceptance criterion must specify an ACTION and OBSERVABLE RESULT:**

- **AC-01 (REQ-NNN):**
  - Given: [precondition]
  - When: User [specific action - clicks, types, navigates]
  - Then: [observable result - UI changes, navigation occurs, data updates]

- **AC-02 (REQ-NNN):**
  - Given: ...
  - When: ...
  - Then: ...
...

**Bad:** "Carousel navigation buttons exist"
**Good:** "Given multiple cards, when user clicks right arrow, then next card is displayed"

**Bad:** "Save button is present"
**Good:** "Given form has content, when user clicks Save, then content is persisted and confirmation shown"

## Edge Cases

### Error Scenarios
- **REQ-NNN:** <What can go wrong and expected behavior>
...

### Boundary Conditions
- **REQ-NNN:** <Extremes and limits>
...

### Invariants
- **INV-01 (REQ-NNN):** <Property that must always hold>
...

## Solution Guidance

### Approaches to Consider
<Ideas mentioned, without commitment>

### Approaches to Avoid
<Anti-patterns or rejected ideas>

### Constraints
<Non-negotiable requirements>

### References
<Similar solutions, prior art>

## Open Questions
<Unresolved items that need further discovery>
```

**Traceability ID assignment:**
- Assign sequential `REQ-NNN` IDs starting from `REQ-001`
- Every success criterion, user story, acceptance criterion, edge case, and invariant gets a REQ ID
- IDs are embedded inline next to the item they identify
- Maintain a running counter across all sections

## Interview Rules

1. **Ask one question at a time** - Let the user fully answer before moving on
2. **Reflect back understanding** - "So what I'm hearing is..." before moving to next topic
3. **Go deeper on vague answers** - "Can you give me an example of that?"
4. **Don't lead the witness** - Ask open questions, not "Would you want X?"
5. **Stay in problem space** - Redirect implementation discussions to guidance
6. **Capture exact language** - User's words often reveal priorities
7. **Identify gaps** - Note when something hasn't been addressed

## Transitions

After each phase, summarize what you've learned and confirm before proceeding:

```
"Let me make sure I understand the problem:
[summary]

Is that accurate, or would you refine anything?"
```

Only move to the next phase when the user confirms understanding.

## Error Handling

**User jumps to solutions:**
- Acknowledge the idea
- Ask "What problem would that solve?"
- Capture in guidance section, return to current phase

**User gives vague answers:**
- Ask for specific examples
- Ask "Walk me through a recent time this happened"
- Ask "What would you see/hear/experience?"

**User doesn't know:**
- Capture as open question
- Ask "Who would know?" or "How could we find out?"
- Move on, don't force answers

**Scope creep:**
- "That sounds like it might be a separate problem. Should we capture it and stay focused on [current topic]?"

## Structured Result

When the interview is complete and requirements.md is written, produce a result summary for the orchestrator:

```
Status: success
Summary: Conducted PM interview across 5 phases. Produced requirements.md with N requirements.
Files created: requirements.md
Traceability: REQ-001 through REQ-NNN assigned
Findings: (any issues discovered)
Context for next phase: (key decisions, constraints, open questions for downstream skills)
```

## Output Expectations

The spec should be:
- **Complete enough** to hand off to design and architecture
- **Focused on behavior** not implementation
- **Testable** - success criteria can be verified
- **Honest about gaps** - open questions clearly marked
- **Traceable** - every requirement has a REQ-NNN ID

## Result Format

See [shared/RESULT.md](../shared/RESULT.md) for the complete schema.

```toml
[status]
success = true

[outputs]
files_modified = ["docs/requirements.md"]

[[decisions]]
context = "Requirements scope"
choice = "Focus on core functionality first"
reason = "Reduces initial complexity"
alternatives = ["Include all features upfront"]

[[learnings]]
content = "User has strong preference for CLI over GUI"
```
