# Architecture: PM Interview Enforcement

**Issue:** ISSUE-54
**Created:** 2026-02-04
**Status:** Draft

**Traces to:** ISSUE-54

---

## Overview

This architecture implements the PM interview enforcement feature with minimal complexity. The core principle is simple: the orchestrator (Claude) passes only context to skills, never behavioral override instructions. The orchestrator naturally interprets user intent from natural language without special detection code.

**Key architectural decision:** Leverage Claude's natural language understanding rather than building detection logic.

---

## Architecture Decisions

### ARCH-1: Context-Only Communication Contract

**Decision:** Orchestrator communicates with interview-producer skills using pure context only, never passing behavioral instructions.

**Rationale:** The root cause of ISSUE-53 was the orchestrator passing override instructions ("skip interview", "problem already defined") in the ARGUMENTS field to pm-interview-producer. This violated the skill's autonomy and caused it to bypass its intended interview logic.

**Implementation:**
- Orchestrator constructs context files (context-pm.toml, context-design.toml, context-arch.toml)
- Context files contain: issue_id, file paths, prior artifact references, user preferences
- Context files do NOT contain: instructions like "skip interview", "already defined", "do not conduct"
- Skills read context and decide their own behavior

**Alternatives considered:**
- Direct instruction passing (rejected: caused ISSUE-53)
- Skill-specific protocols (rejected: unnecessary complexity)

**Traces to:** REQ-1, REQ-4, DES-001, DES-002, DES-004, DES-005

---

### ARCH-2: Natural Language Intent Interpretation

**Decision:** The orchestrator (Claude) directly interprets user natural language intent without implementing special detection logic.

**Rationale:** Claude is the orchestrator. When a user says "skip interviews" or "skip the PM interview", Claude naturally understands this intent. There's no need for pattern matching, regular expressions, or phrase detection code.

**Implementation:**
- Claude reads user's initial message
- Claude identifies skip preference from natural language
- Claude adds `skip_interview_preference = true` to context TOML if user expressed skip intent
- No special detection code needed - Claude's language understanding is the implementation

**Alternatives considered:**
- Regex pattern matching (rejected: unnecessary, Claude already understands)
- Keyword lists (rejected: brittle, Claude already understands)
- User flag like `--skip-interview` (rejected: user wanted natural language)

**Traces to:** REQ-3, DES-003

---

### ARCH-3: TOML Context File Format

**Decision:** Use TOML files (context-<phase>.toml) to pass context including skip preference to skills.

**Rationale:** The project already uses TOML files for context passing (context-pm.toml, context-design.toml exist in project directories). This is a proven pattern that's structured, persistent, and easy to extend.

**Context file structure:**
```toml
[input]
issue_id = "ISSUE-NNN"
requirements_path = "path/to/requirements.md"
design_path = "path/to/design.md"
skip_interview_preference = false  # Added field

[output]
yield_path = "path/to/yield.toml"
artifact_path = "path/to/artifact.md"
```

**Skip preference field:**
- `skip_interview_preference = true` when user expressed skip intent
- `skip_interview_preference = false` (default) when no skip intent
- Field is optional - absence means false

**Alternatives considered:**
- Environment variables (rejected: global scope pollution, hard to test)
- JSON in ARGUMENTS (rejected: breaks REQ-1 if not carefully structured)
- Skill-specific flags (rejected: inconsistent, requires signature changes)

**Traces to:** REQ-1, REQ-3, DES-003

---

### ARCH-4: Orchestrator Validation of Minimum Interaction

**Decision:** Orchestrator validates that at least one user-response file exists before accepting `complete` yield from interview-producer skills (unless skip preference was provided).

**Rationale:** REQ-2 requires minimum user interaction. The orchestrator enforces this by checking for evidence of interaction (user-response-N.toml files) after the skill yields `complete`.

**Implementation:**
- When pm-interview-producer yields `complete`:
  - Orchestrator checks if `skip_interview_preference = true` in context
  - If skip preference: accept completion immediately (no validation)
  - If no skip preference: check for at least one user-response-*.toml file
  - If no user-response files found: reject completion, request user interaction

**File check logic:**
```
files = list_files(project_dir, "user-response-*.toml")
if skip_interview_preference == true:
    accept_completion()
else if len(files) > 0:
    accept_completion()
else:
    error("Minimum user interaction required: no user-response files found")
```

**Alternatives considered:**
- Trust skill to enforce minimum interaction (rejected: no external validation)
- Count questions yielded (rejected: skill could yield questions but not wait for responses)
- Check conversation history (rejected: fragile, depends on conversation state)

**Traces to:** REQ-2, REQ-3, DES-006

---

## System Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                           User                               │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            │ "projctl project start ISSUE-54 [skip interviews]"
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Orchestrator (Claude)                     │
│                                                              │
│  1. Parse user message (natural language interpretation)    │
│  2. Detect skip preference (if present)                     │
│  3. Create context-<phase>.toml with skip field             │
│  4. Invoke interview-producer skill                          │
│  5. Validate user-response files (if no skip preference)    │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            │ context-pm.toml
                            │ {issue_id, paths, skip_interview_preference}
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              pm-interview-producer (Skill)                   │
│                                                              │
│  1. Read context-pm.toml                                     │
│  2. Check skip_interview_preference                          │
│  3. If skip=true & issue sufficient: SYNTHESIZE              │
│  4. If skip=true & info missing: yield need-user-input       │
│  5. If skip=false: yield need-user-input (minimum 1)         │
│  6. Yield complete when done                                 │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            │ yield.toml
                            │ user-response-N.toml (0 or more)
                            │ requirements.md
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Orchestrator (Claude)                     │
│                                                              │
│  6. Receive complete yield                                   │
│  7. If skip=true: accept completion                          │
│  8. If skip=false: check user-response files exist           │
│  9. If no files: reject, request interaction                 │
│  10. If files exist: accept completion                       │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **User → Orchestrator:** Natural language command with optional skip phrase
2. **Orchestrator → Context File:** TOML file with issue, paths, skip preference
3. **Context File → Skill:** Skill reads TOML at startup
4. **Skill → Yield File:** Skill writes yield.toml with status (need-user-input or complete)
5. **Skill → User Response File:** User responses written to user-response-N.toml
6. **Yield File → Orchestrator:** Orchestrator reads yield to determine next action
7. **User Response Files → Orchestrator:** Orchestrator checks file count for validation

---

## Implementation Guidance

### Orchestrator Changes

**Location:** Orchestrator skill invocation logic (likely in orchestrator SKILL.md or project management code)

**Changes needed:**

1. **Natural language interpretation (no code):**
   - Claude reads user's initial message
   - Claude identifies skip intent from phrases like "skip interviews", "skip the PM interview", etc.
   - Claude sets internal flag: `skip_interview_preference = true/false`

2. **Context file generation:**
   - When creating context-pm.toml, context-design.toml, context-arch.toml
   - Add field: `skip_interview_preference = <value>` based on Claude's interpretation
   - Default to `false` if no skip intent detected

3. **Validation after completion:**
   - When receiving `complete` yield from interview-producer
   - Read `skip_interview_preference` from context file
   - If `true`: accept completion (no validation)
   - If `false`: check for user-response-*.toml files
   - If no files: reject completion, inform user "Minimum user interaction required"

### Skill Changes

**Location:** pm-interview-producer, design-interview-producer, arch-interview-producer

**Changes needed:**

1. **Read skip preference from context:**
   - At skill startup, read context-<phase>.toml
   - Check for `skip_interview_preference` field (default false if absent)

2. **Adjust GATHER logic:**
   - If `skip_interview_preference = true`:
     - Evaluate if issue/prior artifacts contain sufficient information
     - If sufficient: skip to SYNTHESIZE
     - If insufficient: yield `need-user-input` for critical missing info only
   - If `skip_interview_preference = false`:
     - Proceed with normal interview (minimum one confirmation question per REQ-2)

**No changes to SYNTHESIZE or PRODUCE phases.**

### No New Components

This architecture requires NO new components:
- No skip detection module (Claude already understands)
- No validation service (orchestrator does simple file check)
- No new skill types (existing interview-producers already have interview logic)

---

## Security and Error Handling

### Error Cases

**E1: User response file missing (no skip preference)**
- **Scenario:** pm-interview-producer yields `complete` but no user-response-*.toml files exist, and `skip_interview_preference = false`
- **Handling:** Orchestrator rejects completion, logs error "Minimum user interaction required: no user-response files found", prompts user for input
- **Recovery:** Skill remains in GATHER phase until user responds

**E2: Context file missing skip field**
- **Scenario:** context-<phase>.toml doesn't have `skip_interview_preference` field
- **Handling:** Skill treats as `false` (default behavior: conduct interview)
- **Recovery:** No recovery needed - default behavior is safe

**E3: Invalid skip field value**
- **Scenario:** `skip_interview_preference = "maybe"` (not boolean)
- **Handling:** Skill logs warning, treats as `false`
- **Recovery:** Proceed with normal interview

### Security Considerations

**No security concerns:** This architecture only affects interaction flow, not data access or permissions. Context files are already trusted artifacts in the project directory.

---

## Performance and Scalability

**Performance impact:** Negligible
- Reading one TOML file: <1ms
- Listing files in directory: <10ms
- No network calls, no database queries

**Scalability:** Not applicable - this is a user interaction pattern, not a data processing system.

---

## Testing Strategy

### Unit Tests

**T1: Context file generation**
- Given user message "skip interviews"
- When orchestrator creates context-pm.toml
- Then `skip_interview_preference = true`

**T2: Context file generation (no skip)**
- Given user message "start project for ISSUE-54"
- When orchestrator creates context-pm.toml
- Then `skip_interview_preference = false` (or field absent)

**T3: Skill skip preference reading**
- Given context-pm.toml with `skip_interview_preference = true`
- When pm-interview-producer starts
- Then skill recognizes skip preference

**T4: Orchestrator validation (skip preference present)**
- Given pm-interview-producer yields `complete`
- And context has `skip_interview_preference = true`
- When orchestrator processes completion
- Then orchestrator accepts without checking user-response files

**T5: Orchestrator validation (no skip preference, has user-response)**
- Given pm-interview-producer yields `complete`
- And context has `skip_interview_preference = false`
- And user-response-1.toml exists
- When orchestrator processes completion
- Then orchestrator accepts completion

**T6: Orchestrator validation (no skip preference, no user-response)**
- Given pm-interview-producer yields `complete`
- And context has `skip_interview_preference = false`
- And no user-response-*.toml files exist
- When orchestrator processes completion
- Then orchestrator rejects completion with error

### Integration Tests

**T7: End-to-end with skip**
- User: "projctl project start ISSUE-54 skip interviews"
- Orchestrator creates context with skip=true
- pm-interview-producer reads context, evaluates issue, yields complete (no questions)
- Orchestrator accepts completion
- requirements.md produced

**T8: End-to-end without skip**
- User: "projctl project start ISSUE-54"
- Orchestrator creates context with skip=false
- pm-interview-producer yields need-user-input (structured confirmation)
- User responds "y"
- pm-interview-producer yields complete
- Orchestrator validates user-response-1.toml exists, accepts completion
- requirements.md produced

---

## Traceability

### Requirements Coverage

| Requirement | Architectural Elements |
|-------------|----------------------|
| REQ-1: Context-Only Contract | ARCH-1, ARCH-3 |
| REQ-2: Minimum User Interaction | ARCH-4 |
| REQ-3: Explicit Skip Mechanism | ARCH-2, ARCH-3 |
| REQ-4: Interview Producer Independence | ARCH-1 |

### Design Coverage

| Design Element | Architectural Elements |
|----------------|----------------------|
| DES-001: Progressive Disclosure | (Skill implementation detail) |
| DES-002: Structured Confirmation | (Skill implementation detail) |
| DES-003: Natural Language Skip | ARCH-2, ARCH-3 |
| DES-004: Adaptive Interview | ARCH-1 |
| DES-005: Clarification Drill-Down | (Skill implementation detail) |
| DES-006: Minimum Interaction Guarantee | ARCH-4 |

---

## Summary

This architecture implements PM interview enforcement with minimal changes:

1. **Orchestrator:** Claude interprets skip intent naturally, adds `skip_interview_preference` to context TOML, validates user-response files after completion
2. **Skills:** Read skip preference from context, adjust interview behavior accordingly
3. **No new components:** Leverages existing TOML context pattern and Claude's natural language understanding

The architecture is simple, testable, and directly addresses the root cause of ISSUE-53 by enforcing a context-only communication contract between orchestrator and skills.
