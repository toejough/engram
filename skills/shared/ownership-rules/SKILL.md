---
name: ownership-rules
description: Domain ownership and structured result rules for all project skills
user-invocable: false
---

# Domain Ownership Rules

These rules apply to every skill in the project orchestration system. They are non-negotiable.

## Ownership

You own your domain completely. Every finding you report is YOUR finding.

- **Do not** classify findings as "pre-existing" or "unrelated to changes."
- **Do not** modify specs, tests, requirements, or designs to match implementation unless explicitly asked.
- **Do not** say "these failures were already there." If something is wrong, report it as wrong. Period.
- **Do not** say "the audit PASSES because the issues were pre-existing." If there are issues, the audit FAILS.
- **Do not** weaken tests, relax linter rules, or adjust acceptance criteria to make failures go away.

If a test fails, make it pass. If a design doesn't match, report the mismatch. If a requirement isn't met, flag it. The origin of the problem is irrelevant. Your job is to assess the current state against the expected state.

## Evidence-Based Findings

Every finding must include evidence. Assertions without evidence are not findings.

- **Good:** "Login button is missing from the header. Screenshot shows header with only logo and nav links. Expected: design DES-004 shows login button at top-right."
- **Bad:** "Login flow looks correct."
- **Good:** "Test `test_parse_heading` fails with: expected 'H1' got 'H2'. See test output below."
- **Bad:** "Some tests are failing but they seem pre-existing."

When auditing:
1. State what you checked
2. State what you expected (with traceability ID if applicable)
3. State what you found
4. Provide the evidence (test output, screenshot, code reference, file path + line number)

## Structured Results

When your work is complete, produce a structured result summary. The orchestrator will read this to determine next steps. Include:

1. **Status:** success, failure, or blocked
2. **Summary:** 1-3 sentences describing what you did
3. **Files affected:** created, modified, or deleted
4. **Findings:** issues discovered, classified as DEFECT, SPEC_GAP, SPEC_REVISION, or CROSS_SKILL
5. **Traceability:** which REQ/DES/ARCH/TASK items you addressed
6. **Context for next phase:** what the next skill in the pipeline should know

## Traceability

Reference traceability IDs (REQ-NNN, DES-NNN, ARCH-NNN, TASK-NNN) in your findings and outputs. When you create new artifacts, assign IDs using the appropriate prefix. When you discover something that affects an upstream artifact, reference the specific ID.

## Classification of Findings

| Classification | Meaning | Action |
|----------------|---------|--------|
| **DEFECT** | Implementation doesn't match spec | Fix implementation |
| **SPEC_GAP** | Spec missed something implementation handles well | Propose spec addition |
| **SPEC_REVISION** | Spec was impractical, implementation found better way | Propose spec change |
| **CROSS_SKILL** | Finding affects another skill's domain | Flag for orchestrator resolution |
