---
name: tdd-red-producer
description: Write failing tests for a task's acceptance criteria (TDD red phase)
context: fork
model: sonnet
user-invocable: false
role: producer
phase: tdd-red
---

# TDD Red Producer

Write failing tests that specify expected behavior before implementation. This is the "red" phase of TDD.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML with task ID, acceptance criteria, architecture notes |
| Output | Test files that fail (red state verified) |
| Traces | TASK-N acceptance criteria |

## Workflow

Follows GATHER -> SYNTHESIZE -> PRODUCE pattern.

### GATHER

1. Read context from `[inputs]` section
2. Load task description and acceptance criteria
3. Load architecture notes relevant to the task
4. Load project conventions (test tooling, patterns)
5. If missing information, yield `need-context` with queries

### SYNTHESIZE

1. Map acceptance criteria to specific test scenarios
2. Identify test file locations following project conventions
3. Determine test categories (example-based vs property-based)
4. Plan test structure (subtests, table-driven, etc.)
5. If blocked, yield `blocked` with details

### PRODUCE

1. Write test files following project conventions
2. Run tests to verify they FAIL (tests must fail - this is required)
3. Verify failures are correct (tests fail because feature doesn't exist, not because tests are broken)
4. Yield `complete` with test file paths and coverage summary

## Test Philosophy

- **Tests must fail**: If tests pass unexpectedly, stop and report - either the feature exists or tests are wrong
- **Cover all acceptance criteria**: Each criterion should have at least one test
- **Test behavior, not structure**: Verify action -> event -> handler -> state -> UI chains
- **Human-readable matchers**: Use assertion libraries that read like sentences
- **Property exploration**: Include randomized property tests to catch edge cases

### Language Conventions

| Language | Blackbox | Stack |
|----------|----------|-------|
| Go | `package foo_test` | gomega + rapid |
| TypeScript | `.test.ts` | vitest + fast-check |

## Documentation Tests

When the task involves documentation changes (.md files, SKILL.md, README, etc.), write tests that validate the documentation's intent.

### Test Types

| Type | When to Use | Example Test |
|------|-------------|--------------|
| Word/phrase matching | Specific terms must appear | `grep -q "## Acceptance Criteria" file.md` |
| Semantic matching | Concepts must be conveyed | `projctl memory query` against doc content |
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
# Uses ONNX embeddings via projctl memory query
test_readme_explains_purpose() {
    # Index the README content first
    projctl memory learn --content "$(cat README.md)" --source README
    # Query for the concept - score >= 0.7 means semantic match
    projctl memory query --text "project purpose and goals" --limit 1 | grep -qE "score: 0\.[7-9]|score: 1\.0"
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

### Applying to Task AC

When a task's acceptance criteria include documentation deliverables:

```markdown
**Acceptance Criteria:**
- [ ] README includes installation instructions
- [ ] API.md documents all endpoints
```

Write tests that verify these exist and convey the right meaning, just as you would for code features.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol.

### Yield Types

| Type | When |
|------|------|
| `complete` | Tests written and verified failing |
| `need-context` | Need files, architecture, or conventions |
| `blocked` | Cannot proceed (missing task details) |
| `error` | Something failed |

### Complete Yield Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
artifact = "internal/foo/foo_test.go"
files_modified = ["internal/foo/foo_test.go"]
test_summary = { total = 5, passing = 0, failing = 5 }

[[payload.test_coverage]]
criterion = "AC-1: User can authenticate"
tests = ["TestAuthentication_ValidCredentials", "TestAuthentication_InvalidCredentials"]

[[payload.test_coverage]]
criterion = "AC-2: Session persists across restarts"
tests = ["TestSessionPersistence"]

[context]
phase = "tdd-red"
subphase = "complete"
task = "TASK-5"
```

## Traceability

Tests trace to upstream task acceptance criteria:

```go
// TestAuthentication verifies TASK-5 AC-1: User can authenticate
func TestAuthentication(t *testing.T) {
    // ...
}
```

## Result Format

`result.toml`: `[status]`, files modified, test summary, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname tdd-red-producer` or see SKILL-full.md
