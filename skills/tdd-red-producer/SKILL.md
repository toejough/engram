---
name: tdd-red-producer
description: Write failing tests for a task's acceptance criteria (TDD red phase)
context: inherit
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
| Input | Context from spawn prompt: task ID, acceptance criteria, architecture notes |
| Output | Test files that fail (red state verified) |
| Traces | TASK-N acceptance criteria |

## Workflow Context

- **Phase**: `tdd_red_produce` (states.tdd_red_produce)
- **Upstream**: Work item selection (`worktree_create`), or retry from red QA (`tdd_red_decide`)
- **Downstream**: `tdd_red_qa` → `tdd_red_decide` → retry, escalate, or advance to green phase
- **Model**: opus (default_model in workflows.toml)

This skill writes failing tests (red) for a single work item in the TDD loop.

---

## Workflow

Follows GATHER -> SYNTHESIZE -> PRODUCE pattern.

### GATHER

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode)
2. Load task description and acceptance criteria
3. Load architecture notes relevant to the task
4. Load project conventions (test tooling, patterns)
5. Query memory for test patterns: `projctl memory query "test patterns for <domain>"`
6. Query memory for known failures: `projctl memory query "known test failures for <feature-area>"`
   If memory is unavailable, proceed gracefully without blocking
7. If missing information, request context from team lead

### SYNTHESIZE

1. Map acceptance criteria to specific test scenarios
2. Identify test file locations following project conventions
3. Determine test categories (example-based vs property-based)
4. Plan test structure (subtests, table-driven, etc.)
5. If blocked, report blocker to team lead

### PRODUCE

1. Write test files following project conventions
2. Run tests to verify they FAIL (tests must fail - this is required)
3. Verify failures are correct (tests fail because feature doesn't exist, not because tests are broken)
4. Send a message to team-lead with test file paths and coverage summary

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

## Testing User Interfaces

UI, CLI, and API are all user interfaces. Each follows the same testing model:

| Layer | Question | What to Test |
|-------|----------|--------------|
| Structure | Does it exist? | Element presence, argument parsing, endpoint existence |
| Behavior | Does it work? | Interaction -> event -> handler -> state -> output |
| Properties | Does it hold? | Invariants across all screens/commands/endpoints |

**Structure tests** verify existence:
- UI: Element exists with correct properties
- CLI: Command accepts expected arguments
- API: Endpoint exists, request/response shape correct

**Behavior tests** verify the full chain:
- UI: Click -> handler fires -> state changes -> UI updates
- CLI: Command runs -> processing -> output appears
- API: Request -> processing -> response returned

**Property tests** verify invariants:
- UI: "Every screen has X" verified across all screens
- CLI: "All commands support --help" verified for all commands
- API: "All endpoints return valid JSON" verified exhaustively

### UI Testing Examples

**Structure test (element exists):**
```typescript
it('renders add-note button on every screen', () => {
  for (const screen of allScreens) {
    render(<screen.Component />);
    expect(screen.getByRole('button', { name: /add note/i })).toBeTruthy();
  }
});
```

**Behavior test (full chain):**
```typescript
it('add-note button opens note editor', () => {
  const onOpen = vi.fn();
  render(<Screen onNoteEditorOpen={onOpen} />);
  fireEvent.click(screen.getByRole('button', { name: /add note/i }));
  expect(onOpen).toHaveBeenCalled();
  expect(screen.getByRole('dialog', { name: /new note/i })).toBeVisible();
});
```

**Property test (invariant):**
```typescript
it.prop([fc.constantFrom(...allScreens)])('all screens have add-note affordance', (screen) => {
  render(<screen.Component />);
  return screen.queryByRole('button', { name: /add note/i }) !== null;
});
```

### CLI Testing Examples

**Structure test (command parses):**
```go
func TestCommandAcceptsFlags(t *testing.T) {
    cmd := NewRootCmd()
    cmd.SetArgs([]string{"notes", "add", "--title", "Test"})
    Expect(cmd.Execute()).To(Succeed())
}
```

**Behavior test (output produced):**
```go
func TestCommandOutputsJSON(t *testing.T) {
    var buf bytes.Buffer
    cmd := NewRootCmd()
    cmd.SetOut(&buf)
    cmd.SetArgs([]string{"notes", "list", "--json"})
    Expect(cmd.Execute()).To(Succeed())
    var notes []Note
    Expect(json.Unmarshal(buf.Bytes(), &notes)).To(Succeed())
}
```

### API Testing Examples

**Contract test (shape correct):**
```go
func TestEndpointReturnsExpectedShape(t *testing.T) {
    resp := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/notes", nil)
    handler.ServeHTTP(resp, req)
    Expect(resp.Code).To(Equal(200))
    var body map[string]any
    Expect(json.Unmarshal(resp.Body.Bytes(), &body)).To(Succeed())
    Expect(body).To(HaveKey("notes"))
}
```

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

---

## Communication

### Team Mode (preferred)

| Action | Tool |
|--------|------|
| Read project docs | `Read`, `Glob`, `Grep` tools directly |
| Run tests | `Bash` |
| Report completion | `SendMessage` to team lead |
| Report blocker | `SendMessage` to team lead |

On completion, send a message to the team lead with:
- Artifact paths (test files created)
- Test results summary (total, passing, failing)
- Files modified
- Key decisions made

---

## Contract

```yaml
contract:
  outputs:
    - path: "<test-file>"
      id_format: "N/A"

  traces_to:
    - "docs/tasks.md"

  checks:
    - id: "CHECK-001"
      description: "Test file exists at specified path"
      severity: error

    - id: "CHECK-002"
      description: "Tests fail when run (red phase)"
      severity: error

    - id: "CHECK-003"
      description: "Tests fail for correct reasons (missing implementation, not syntax errors)"
      severity: error

    - id: "CHECK-004"
      description: "Tests cover task acceptance criteria"
      severity: error

    - id: "CHECK-005"
      description: "No compilation or import errors"
      severity: error

    - id: "CHECK-006"
      description: "No implementation code beyond minimal stubs"
      severity: error

    - id: "CHECK-007"
      description: "Tests describe expected behavior clearly"
      severity: warning

    - id: "CHECK-008"
      description: "Property tests used for invariants and edge cases"
      severity: warning

    - id: "CHECK-009"
      description: "Tests are blackbox (test public API, not internals)"
      severity: warning
```

---

## Lessons Learned

**No flaky tests**: Flaky tests are not acceptable. Never dismiss test failures as "flaky" or "race conditions" - fix them. Use dependency injection (imptest) to avoid IO-based flakiness. If a test captures stdout/stderr or uses timing, inject those dependencies instead.

**Failing tests mean implementation bugs**: When a test fails, investigate the implementation first, not the test. Never adjust tests to match code without verifying whether the code has a bug. Tests encode expected behavior - if the test is reasonable, the code is wrong.

**No whitebox tests for unexported functions**: Testing unexported functions directly is testing implementation, not behavior. Use dependency injection with property tests instead. If coverage seems insufficient without whitebox access, stop and discuss the specific situation - the design likely needs adjustment, not the test approach.

**Test tooling expectations**: Tests should use two categories of libraries:
1. **Human-readable matchers** - Assertions that read like sentences (e.g., `Expect(x).To(Equal(y))`, `expect(x).toBe(y)`). These make test failures self-documenting.
2. **Randomized property exploration** - Libraries that generate random inputs to verify properties hold across many cases, not just hand-picked examples. This catches edge cases humans miss.
The specific libraries vary by language (Go: gomega + rapid, TS: vitest matchers + fast-check), but the principle is universal: tests should be readable AND thorough.

**Sketch test structure before writing**: Before writing tests, outline the expected structure (Given/When/Then) and review for simplicity. This catches over-engineering before implementation. Tests refactored during creation (-66 +46 lines in ISSUE-58) indicate premature complexity that should have been caught in planning.

**TDD applies to ALL artifacts, not just code**: Documentation and design/UI work require TDD discipline too. Tests for docs include: word/phrase matching (grep for required terms), semantic matching (projctl memory query for conceptual coverage), structural tests (required sections exist). Tests for design/UI include: visual regression tests (projctl screenshot diff), component presence/behavior tests, accessibility checks. When a task produces .md files, .pen files, SKILL.md updates, or any artifact, write tests that verify the artifact's intent. See `tdd-red-producer` SKILL.md "Documentation Tests" and "Testing User Interfaces" sections.

**No TODO comments for incomplete work**: Never leave TODO/FIXME comments for functionality that should be implemented now. Either implement it fully or ask Joe before deferring. Silent deferral via comments is unacceptable.
