---
name: task-audit
description: Validate task completion against acceptance criteria with TDD discipline checks
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Task Audit Skill

Validate that a completed task meets its acceptance criteria and followed TDD discipline.

## Purpose

After a TDD cycle completes, audit the work to ensure:
- All acceptance criteria are met with evidence
- TDD discipline was followed (no test weakening, no linter gaming)
- Test quality standards met (property-based tests, blackbox testing)
- Implementation aligns with architecture and design specs

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Task ID (e.g., TASK-004)
- Path to tasks.md (with acceptance criteria)
- TDD phase summaries (red, green, refactor results)
- Project directory
- Traceability references (REQ, DES, ARCH IDs)

## Audit Steps

### 1. Load Task Definition

1. Read tasks.md
2. Find the task by ID
3. Extract acceptance criteria, files, test properties, traceability

### 2. Verify Files Exist

1. Check all "Create" files exist
2. Check all "Modify" files were touched (git diff)
3. Flag missing or unexpected files

### 3. Verify Tests Pass

1. Run test command (`go test ./...`, `npm test`, etc.)
2. **ALL tests must pass** - no exceptions
3. Report any failures

**Fix ALL failures when found.** Never dismiss test failures as "pre-existing" or "unrelated." If tests fail during audit, the audit FAILS.

### 4. TDD Discipline Check

**These are red flags that indicate TDD was circumvented:**

#### 4.1 Test Weakening

Check git diff for:
- Removed or commented-out test cases
- Weakened assertions (e.g., `toBe` to `toBeTruthy`, exact match to contains)
- Added `.skip` or `.only` to tests
- Changed expected values to match buggy output

**If any test was modified to make it pass rather than fixing the code, this is a FAIL.**

#### 4.2 Linter Gaming

Check for:
- New `nolint` directives added
- New lint disable comments
- Changes to linter config files
- New entries in lint exclusion lists
- Threshold changes

**If linter rules were disabled/weakened instead of fixing the code, this is a FAIL.**

#### 4.3 Test Quality Standards

For **Go** projects:
- Tests use blackbox pattern (`package foo_test`)
- Property-based tests use `rapid`
- No whitebox tests of unexported functions

For **TypeScript** projects:
- Tests in `.test.ts` files
- Property-based tests use appropriate library

#### 4.4 Test Properties Implemented

If the task specifies test properties, verify each has a corresponding property-based test.

### 5. UI Verification (For UI Tasks)

**When applicable:** Task creates or modifies visual components.

#### 5.1 Visual Verification
- Verify visual correctness at required viewports
- Compare against design spec if available
- Check responsive behavior
- **Actually look at screenshots** - DOM existence is not enough

#### 5.2 Behavioral Verification (MANDATORY for interactive elements)

**Every interactive element added or modified by this task MUST be exercised.**

For each button, link, input, or interactive element:
1. **Click it / interact with it** - Use Chrome DevTools MCP or equivalent
2. **Verify something happens** - State changes, navigation occurs, event fires
3. **Verify the FULL chain** - Event → Handler → State change → UI update
4. **Document the interaction** - What you did, what happened

**If an element renders but does nothing on interaction, this is a FAIL.**

Test expectations like "button emits event" are necessary but insufficient. The audit must verify that:
- The event is actually emitted when clicked
- Something listens for that event
- The listener does something meaningful
- The UI reflects the change

**"It exists" + "It compiles" + "Tests pass" ≠ "It works"**

### 6. Verify Acceptance Criteria

For each criterion:
1. Locate relevant code/test
2. Confirm behavior is implemented
3. Mark as PASS or FAIL with evidence (file:line, test output, screenshot)

### 7. Cross-Reference Specs

- **Architecture:** Does implementation match specified interfaces, patterns, file locations?
- **Design:** Do components match design spec?
- **Requirements:** Does this task contribute to traced requirements?

**Mismatches are FAILURES.** Spec violations must be fixed.

### 8. End-to-End Usability Check

**For any task that adds or modifies user-facing functionality:**

1. **Identify the user interface** - How will users access this feature?
   - CLI command
   - API endpoint
   - UI element
   - Library function

2. **Verify the interface exists and works**
   - CLI: Can you run the command? Does it produce expected output?
   - API: Can you call the endpoint? Does it return expected response?
   - UI: Can you see and interact with it?
   - Library: Is the function exported and documented?

3. **Trace from implementation to interface**
   - New internal function → Must be called by exposed interface
   - New validation logic → Must be triggerable by user action
   - New data structure → Must be accessible/visible somewhere

**If functionality exists in code but users cannot access it, this is a FAIL.**

Common failure patterns:
- Internal function implemented but no CLI command calls it
- API handler written but no route registered
- Component created but not rendered anywhere
- Library function exists but not exported

**"Code exists" ≠ "Feature is usable"**

## Decision Rules

**PASS if:**
- All acceptance criteria met
- All tests pass
- No TDD discipline violations
- No blocking spec mismatches

**RETRY if:**
- Some criteria not met but fixable
- Any tests fail
- TDD violation detected
- Minor spec misalignment

**ESCALATE if:**
- Spec contradiction discovered
- Ambiguous requirements
- Architecture decision needed
- Repeated failures after retry

## Findings Classification

| Classification | Meaning | Action |
|----------------|---------|--------|
| **DEFECT** | Task criterion not met | Fix implementation |
| **SPEC_GAP** | Discovered undocumented behavior that works well | Propose spec addition |
| **SPEC_REVISION** | Criterion was impractical, found better approach | Propose spec change |
| **CROSS_SKILL** | Finding affects another domain | Flag for resolution |

## Structured Result

```
Status: success | failure | blocked
Summary: Audited TASK-NNN. X/Y acceptance criteria met. TDD discipline: pass|fail.
Acceptance criteria:
  - criterion: <text>
    status: pass | fail
    evidence: <file:line, test output, screenshot>
TDD discipline:
  test_weakening: pass | fail
  linter_gaming: pass | fail
  test_quality: pass | fail
  property_tests: pass | fail | n/a
End-to-end usability:
  user_interface: CLI | API | UI | Library | n/a
  interface_exists: pass | fail
  interface_works: pass | fail
  evidence: <command output, API response, screenshot>
Files verified: X/Y
Tests: N passing, M failing
Findings:
  - classification: DEFECT | SPEC_GAP | SPEC_REVISION | CROSS_SKILL
    description: <what>
    evidence: <proof>
    severity: blocking | warning
Traceability: [TASK, REQ, DES, ARCH IDs]
Recommendation: PASS | RETRY | ESCALATE
```
