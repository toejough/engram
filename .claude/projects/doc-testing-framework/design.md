# Design: Documentation Testing Framework

**Project:** doc-testing-framework
**Created:** 2026-02-04

## Overview

This project adds documentation testing guidance to existing TDD skill files. The design specifies what sections to add to each file.

## DES-001: tdd-red-producer Documentation Testing Section

**File:** `~/.claude/skills/tdd-red-producer/SKILL.md`

**Add section:** "## Documentation Tests"

**Content structure:**
```markdown
## Documentation Tests

When the task involves documentation changes (.md files, SKILL.md, README, etc.),
write tests that validate the documentation's intent.

### Test Types

| Type | When to Use | Example Test |
|------|-------------|--------------|
| Word/phrase matching | Specific terms must appear | `grep -q "## Acceptance Criteria" file.md` |
| Semantic matching | Concepts must be conveyed | `projctl memory query --text "concept" \| grep -q "score: 0\.[7-9]"` |
| Structural | Required sections/format | `grep -c "^## " file.md` to count H2 sections |

### Examples

**Word matching test:**
```bash
# Test: SKILL.md must document yield types
test_yield_types_documented() {
    grep -q "## Yield Types" skills/my-skill/SKILL.md
}
```

**Semantic matching test:**
```bash
# Test: README explains the project purpose
test_readme_explains_purpose() {
    score=$(projctl memory query --text "project purpose and goals" --limit 1 | grep -oP 'score: \K[0-9.]+')
    [ $(echo "$score >= 0.7" | bc) -eq 1 ]
}
```

**Structural test:**
```bash
# Test: SKILL.md has required sections
test_skill_structure() {
    grep -q "^## Purpose" SKILL.md &&
    grep -q "^## Usage" SKILL.md &&
    grep -q "^## Output" SKILL.md
}
```
```

**Traces to:** REQ-001a

## DES-002: tdd-green-producer Documentation Editing Section

**File:** `~/.claude/skills/tdd-green-producer/SKILL.md`

**Add section:** "## Making Documentation Tests Pass"

**Content structure:**
```markdown
## Making Documentation Tests Pass

When doc tests fail, edit the documentation minimally to make them pass.

### Principles

1. **Add only what's needed** - Don't over-document
2. **Match the test's expectation** - If test checks for "## Yield Types", add that exact heading
3. **Preserve existing content** - Don't remove working content while adding new

### Examples

**Example 1: Word matching test fails**

Test: `grep -q "## Acceptance Criteria" SKILL.md`
Failure: Section doesn't exist

Minimal fix:
```markdown
## Acceptance Criteria

[Add criteria here]
```

**Example 2: Semantic test fails**

Test: README must explain "how to install the tool" (similarity >= 0.7)
Failure: Score is 0.45

Minimal fix: Add installation section with clear language:
```markdown
## Installation

To install the tool, run:
\`\`\`bash
go install github.com/example/tool@latest
\`\`\`
```
```

**Traces to:** REQ-001b

## DES-003: tdd-refactor-producer Documentation Refactoring Section

**File:** `~/.claude/skills/tdd-refactor-producer/SKILL.md`

**Add section:** "## Refactoring Documentation"

**Content structure:**
```markdown
## Refactoring Documentation

After doc tests pass, refactor for clarity and organization while keeping tests green.

### Documentation Best Practices

| Practice | Description |
|----------|-------------|
| Progressive disclosure | Most important info first, details later |
| Clarity and conciseness | Remove filler words, be direct |
| Consistent structure | Same heading hierarchy, same patterns |
| Remove redundancy | Don't repeat information across sections |
| Doc-type-specific | READMEs need quick start; API docs need exhaustive detail |

### Refactoring Checklist

- [ ] Tests still pass after each change
- [ ] Most important content is near the top
- [ ] No redundant sections saying the same thing
- [ ] Consistent heading levels (H2 for main sections, H3 for subsections)
- [ ] Code examples are minimal and runnable
- [ ] Links work and point to correct locations
```

**Traces to:** REQ-001c

## DES-004: Orchestrator Doc-Focused Task Guidance

**File:** `~/.claude/skills/project/SKILL-full.md`

**Add section:** (integrate into existing TDD dispatch section)

**Content to add:**
```markdown
### Documentation-Focused Tasks

Documentation tasks get full TDD treatment when ANY of these indicators are present:

| Indicator | Example |
|-----------|---------|
| Issue mentions docs | "Update SKILL.md with new yield types" |
| Task AC target .md files | "- [ ] README.md includes installation" |
| Task explicitly about docs | "Document the API endpoints" |

**Do NOT skip TDD for doc tasks.** Apply the same red-green-refactor cycle:
- RED: Write tests for what the doc should contain
- GREEN: Write the doc to make tests pass
- REFACTOR: Improve clarity, structure, readability

Only skip TDD for truly incidental doc updates (typo fixes, minor clarifications during code work).
```

**Traces to:** REQ-002a

## Design Rationale

1. **Sections, not new files** - Adding to existing SKILL.md files keeps guidance co-located with related content
2. **Concrete examples** - Each section includes runnable test examples
3. **Minimal additions** - Only add what's necessary to satisfy requirements
4. **Testable output** - The design itself can be tested (grep for section headers)
