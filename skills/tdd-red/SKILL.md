---
name: tdd-red
description: Write failing tests for a task (TDD red phase)
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# TDD Red Phase Skill

Write failing tests for a given task. This is the "red" phase of TDD.

## Purpose

Define expected behavior through tests that don't pass yet. Receives a task description with acceptance criteria and produces failing tests that fully specify the expected behavior.

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Task ID, description, and acceptance criteria
- Files to create/modify
- Architecture notes relevant to the task
- Traceability IDs (REQ, DES, ARCH references)
- Project conventions (test tooling, patterns)

## Process

1. **Understand the task** - Read context file, understand what needs to be tested
2. **Identify test cases** - Map acceptance criteria to specific test scenarios
3. **Write tests** - Create test files following project conventions
4. **Run tests** - Confirm they FAIL (this is required)
5. **Verify failures are correct** - Tests should fail because the feature doesn't exist, not because tests are broken

## Test Tooling Philosophy

- **Human-readable matchers**: Use assertion libraries that read like sentences. Test failures should be self-documenting.
- **Randomized property exploration**: Use libraries that generate random inputs to verify properties hold across many cases. This catches edge cases hand-picked examples miss.
- **Subtest organization**: Group related tests under a single parent function using subtests.

### Go Conventions
- **Blackbox tests**: Use `package foo_test`, not `package foo`
- **Testing stack**: gomega (matchers), rapid (property exploration)
- **No whitebox tests for unexported functions** - test behavior, not implementation

### TypeScript Conventions
- **Test files**: Use `.test.ts` suffix
- **Testing stack**: vitest matchers (human-readable), fast-check (property exploration)
- **Component tests**: Test behavior, not implementation details

## Rules

1. **Write tests ONLY** - No implementation code whatsoever
2. **Tests must FAIL** - If they pass, either the tests are wrong or code already exists. Stop and report.
3. **Cover all acceptance criteria** - Each criterion should have at least one test
4. **Include property-based tests** where applicable - not just example-based tests
5. **Follow project conventions** - test file location, naming, tooling
6. **No test helpers that hide behavior** - Tests should be readable without deep helper chains
7. **Test BEHAVIOR, not just structure** - See below

## Testing Behavior vs. Structure

**Structural tests (necessary but insufficient):**
- Element exists in DOM
- Component renders without error
- Attribute is set correctly

**Behavioral tests (REQUIRED for interactive elements):**
- Click button → event is emitted with correct payload
- Event is emitted → handler is called
- Handler is called → state changes
- State changes → UI updates

**For every interactive UI element, tests must cover the FULL behavior chain:**

```
User action → Event emitted → Event handled → State change → UI update
```

If any link in this chain is untested, the feature is not fully specified.

**Example - Bad (structural only):**
```typescript
it('renders carousel navigation dots', () => {
  expect(shadow.querySelectorAll('[data-nav-dot]').length).toBe(5);
});
```

**Example - Good (behavioral):**
```typescript
it('clicking nav dot emits navigate-card event with index', () => {
  const handler = vi.fn();
  element.addEventListener('navigate-card', handler);

  (shadow.querySelector('[data-nav-dot][data-card-index="2"]') as HTMLElement).click();

  expect(handler).toHaveBeenCalledWith(expect.objectContaining({
    detail: { index: 2 }
  }));
});
```

**For orchestration/integration, also test:**
```typescript
it('navigate-card event updates displayed card', () => {
  // Setup with card at index 0
  appShell.dispatchEvent(new CustomEvent('navigate-card', { detail: { index: 2 } }));
  // Verify card at index 2 is now displayed
});
```

## What NOT to Do

- Do not write any implementation code
- Do not create stub implementations to make tests compile (unless the language requires it for type-checking, in which case make them panic/throw)
- Do not skip acceptance criteria
- Do not write tests that test implementation details rather than behavior
- Do not dismiss existing test failures as "pre-existing"

## Structured Result

When tests are written and confirmed failing, produce:

```
Status: success | failure | blocked
Summary: Wrote N tests across M test files. All failing as expected.
Files created: [list of test files]
Files modified: [list]
Tests: N total, 0 passing, N failing
Test categories:
  - Example-based: X tests
  - Property-based: Y tests
Acceptance criteria coverage:
  - AC-1: covered by test_foo, test_bar
  - AC-2: covered by test_baz
Traceability: [REQ/DES/ARCH IDs addressed]
Context for next phase: [test file locations, key test descriptions, any stubs needed for compilation]
```
