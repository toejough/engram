---
name: pm-audit
description: Validate implementation against requirements specification
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# PM Audit Skill

Validate implementation against the product specification (requirements.md).

## Purpose

Test actual workflows end-to-end against requirements. Not just checking that features exist -- verifying that they work correctly and meet success criteria.

**"Feature exists" is not the same as "feature works correctly."**

## Domain Ownership

This skill audits within the **problem space** (same domain as `/pm-interview`).

**Validates:**
- Does implementation solve the stated problem?
- Are success criteria met?
- Do workflows match requirements?

**Does NOT validate:**
- Visual/interaction design quality → `/design-audit`
- Architecture adherence → `/architect-audit`

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Path to requirements.md (with REQ- IDs)
- Project directory
- How to start/access the application (if applicable)
- Traceability references

## Audit Steps

1. **Load requirements spec** - Read requirements.md, extract all REQ- IDs
2. **Start the application** - Ensure app is running and accessible (if applicable)
3. **For each user story (REQ IDs):**
   - Attempt to complete the workflow as a user would
   - **Actually perform the actions** - click buttons, fill forms, navigate
   - Document each step: what you did, what happened
   - Note any friction, confusion, or failures
   - Compare against acceptance criteria

   **CRITICAL: "Feature exists" ≠ "Feature works"**

   For EVERY interactive element mentioned in the user story:
   - Click it / interact with it
   - Verify the expected outcome occurs
   - If nothing happens, this is a DEFECT

   Do not infer that something works because:
   - The code looks right
   - Tests pass
   - The element renders

   The only evidence that matters is: **you did the action and saw the result.**

4. **For each success criterion (REQ IDs):**
   - Verify the criterion is testable
   - Test it explicitly
   - Document: PASS (with evidence) or FAIL (with details)

5. **For each documented edge case (REQ IDs):**
   - Reproduce the edge case scenario
   - Verify expected behavior occurs
   - Document any deviations

6. **Check solution guidance compliance:**
   - "Approaches to avoid" - verify they were avoided
   - Constraints - verify they are respected

7. **Report findings with classification**

## Findings Classification

| Classification | Meaning | Action |
|----------------|---------|--------|
| **DEFECT** | Implementation wrong, spec is correct | Fix implementation |
| **SPEC_GAP** | Spec missed something implementation handles well | Propose spec addition |
| **SPEC_REVISION** | Spec was impractical, implementation found better way | Propose spec change |
| **CROSS_SKILL** | Finding affects design or architecture domain | Flag for resolution |

**When to propose spec changes (not just fail):**
- Implementation handles an edge case the spec didn't anticipate
- User feedback during implementation revealed a better workflow
- A requirement proved impossible/impractical and was reasonably adapted
- The "spirit" of the requirement is met but not the "letter"

**When to flag cross-skill conflicts:**
- A requirement implies UX that design says is problematic
- A requirement implies architecture that architect says is disproportionate
- Meeting the requirement exactly would break design consistency

## Structured Result

```
Status: success | failure | blocked
Summary: Audited N requirements. X pass, Y fail, Z proposals.
Findings:
  defects:
    - id: REQ-NNN
      description: <what's wrong>
      evidence: <test output, screenshot, code reference>
      severity: blocking | warning
  proposals:
    - id: REQ-NNN
      current_spec: <what spec says>
      proposed_change: <what to change>
      rationale: <why>
  cross_skill:
    - id: REQ-NNN
      conflicts_with: <DES-NNN or ARCH-NNN>
      issue: <description>
Traceability: [REQ IDs audited]
User stories verified: X/Y
Success criteria met: X/Y
Edge cases handled: X/Y
Recommendation: PASS | FIX_REQUIRED | PROPOSALS_PENDING
```

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
