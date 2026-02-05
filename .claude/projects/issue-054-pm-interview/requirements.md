# Requirements: PM Interview Enforcement

**Issue:** ISSUE-054
**Created:** 2026-02-04
**Status:** Draft

**Traces to:** ISSUE-054

---

## Overview

Ensure the orchestrator enforces minimum user interaction during PM interview phase. The root cause of ISSUE-053 was that the orchestrator passed override instructions ("skip interview", "problem already defined") to pm-interview-producer, causing it to skip the interview entirely and produce requirements based on its own interpretation.

**Core principle:** Orchestrator should only pass context to skills, never override instructions.

---

## Requirements

### REQ-1: Orchestrator Context-Only Contract

As a project orchestrator, I want to pass only context (issue description, files, prior artifacts) to interview-producer skills, so that skills can conduct their interviews without external interference.

**Acceptance Criteria:**
- [ ] Orchestrator does not pass phrases like "skip interview", "already defined", "do not conduct", or similar override instructions in ARGUMENTS
- [ ] Orchestrator only includes: issue context, file paths, prior artifact references
- [ ] ARGUMENTS field contains pure context, not behavioral instructions

**Priority:** P0

**Traces to:** ISSUE-054

---

### REQ-2: Minimum User Interaction

As a user invoking pm-interview-producer, I want to be presented with the problem and proposed solution for confirmation, so that I can verify the skill understood my needs correctly before requirements are produced.

**Acceptance Criteria:**
- [ ] pm-interview-producer yields at least one `need-user-input` during GATHER phase
- [ ] Orchestrator enforces: at least one user response must be recorded before accepting `complete` yield from skill
- [ ] Orchestrator validation: check that at least one `user-response-N.toml` file exists before accepting completion
- [ ] Minimum interaction: present problem + solution summary, ask "Is this correct?"
- [ ] Single confirmation question is sufficient (don't overthink it)

**Priority:** P0

**Traces to:** ISSUE-054

**Enforcement:** Orchestrator validates that at least one user-response file exists before accepting `complete` yield.

---

### REQ-3: Explicit Skip Mechanism

As a user, I want the ability to explicitly skip the PM interview when I'm confident the issue description is complete, so that I can move directly to requirements production when appropriate.

**Acceptance Criteria:**
- [ ] User can pass explicit skip instruction in orchestrator invocation
- [ ] Skip instruction is included in context (NOT as override instruction in ARGUMENTS)
- [ ] Without explicit skip instruction from user, interview must occur
- [ ] Orchestrator does not decide to skip on user's behalf

**Priority:** P1

**Traces to:** ISSUE-054

**Notes:** Mechanism details (flag syntax, context format) are design-phase concerns. Requirements specify the contract: user provides skip instruction → orchestrator includes in context → skill reads from context.

---

### REQ-4: Interview Producer Independence

As an interview-producer skill, I want to decide when I have sufficient information to proceed, so that I'm not forced to ask unnecessary questions when issue context is genuinely complete.

**Acceptance Criteria:**
- [ ] Skill can decide to move from GATHER to SYNTHESIZE when it has enough information
- [ ] Orchestrator does not override skill's GATHER logic via ARGUMENTS
- [ ] Orchestrator does not pass behavioral instructions like "skip interview", "already defined", "do not conduct"
- [ ] REQ-2 minimum interaction requirement is enforced by orchestrator regardless of skill's assessment

**Priority:** P1

**Traces to:** ISSUE-054

**Notes:** This requirement clarifies the boundary: orchestrator enforces minimum interaction externally (by checking for user-response files), while skill controls interview depth and logic internally. The orchestrator's role is to validate minimum interaction occurred, NOT to instruct skills on whether to interview.

**Relationship to REQ-2:** REQ-4 ensures orchestrator doesn't bypass skill logic. REQ-2 ensures orchestrator validates minimum interaction after skill completes. No conflict: skill decides interview depth (REQ-4), orchestrator validates minimum occurred (REQ-2).

---

## Edge Cases

### EC-1: Issue Description Appears Complete

**Scenario:** Issue description is detailed and seems to contain all requirements.

**Behavior:**
- pm-interview-producer still yields at least one confirmation question (REQ-2)
- Example: "Based on the issue, you want X. Is this correct? Any clarifications?"
- User can confirm or clarify

**Priority:** P0

---

### EC-2: User Provides Skip Instruction

**Scenario:** User explicitly requests to skip interview via flag or instruction.

**Behavior:**
- User provides explicit skip instruction to orchestrator
- Orchestrator includes skip instruction in context configuration
- pm-interview-producer reads skip instruction from context and proceeds to SYNTHESIZE
- Requirements are produced directly from issue description
- REQ-2 minimum interaction requirement is waived when explicit skip instruction present

**Priority:** P1

**Notes:** Consistent with REQ-3 - skip instruction comes from user, not orchestrator's judgment. Implementation details (context format, flag syntax) are design concerns.

---

### EC-3: Other Interview Producers (design, arch)

**Scenario:** design-interview-producer or arch-interview-producer are invoked.

**Behavior:**
- Same contract applies (context-only, minimum interaction)
- Each interview producer enforces its own minimum interaction
- Orchestrator does not bypass any interview phase

**Priority:** P1

**Notes:** This is a systemic fix, not specific to pm-interview-producer.

---

## Out of Scope

The following are explicitly out of scope for this issue:

1. **Validation mechanisms** - No complex validation checks for "is interview sufficient". Keep it simple: at least one question/response.
2. **Interview quality assessment** - Not evaluating whether questions were "good enough", just that they happened.
3. **Skill behavioral changes** - pm-interview-producer already has interview capability; no skill rewrites needed.
4. **Skip mechanism design** - REQ-3 mechanism (flag name, syntax) deferred to design phase.

---

## Success Criteria

This issue is resolved when:

1. **REQ-1 verification:** Orchestrator code review shows no override instructions ("skip interview", "already defined", etc.) are passed in ARGUMENTS to interview producers
2. **REQ-2 verification:** Test case confirms orchestrator checks for at least one `user-response-N.toml` file before accepting `complete` yield from pm-interview-producer
3. **REQ-3 verification:** Test case with explicit user skip instruction demonstrates requirements are produced without interview, and test without skip instruction demonstrates interview occurs
4. **REQ-4 verification:** Orchestrator code review shows ARGUMENTS only contains context (issue, files, artifacts), never behavioral instructions

**Test descriptions:**
- **Test 1 (REQ-1, REQ-4):** Code inspection of orchestrator's skill invocation logic to verify ARGUMENTS field construction contains only context
- **Test 2 (REQ-2):** Run orchestrator with issue description, verify pm-interview-producer asks at least one question, verify orchestrator rejects completion if no user-response files exist
- **Test 3 (REQ-3):** Run orchestrator with explicit skip flag, verify interview is skipped; run without flag, verify interview occurs

---

## Traceability

**Upstream:** ISSUE-054
**Downstream:** (Design and Architecture phases will reference these REQ-IDs)
