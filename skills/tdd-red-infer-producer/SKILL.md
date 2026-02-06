---
name: tdd-red-infer-producer
description: Infer needed tests from existing implementation code
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: tdd-red
variant: infer
---

# TDD Red Infer Producer

Analyze existing implementation to infer what tests are needed. Produces test files with failing tests.

**Template:** [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md)

---

## Purpose

Deduce needed tests from existing code without explicit requirements. Used for:
- Adding test coverage to untested code
- Documenting implicit behavior through tests
- Creating test artifacts from implemented logic

---

## Workflow: GATHER → SYNTHESIZE → PRODUCE

### 1. GATHER

Collect information about existing implementation:

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode)
2. Check for `[query_results]` (resuming after need-context)
3. If missing implementation details, yield `need-context`:

```toml
[yield]
type = "need-context"
timestamp = 2026-02-02T10:30:00Z

[[payload.queries]]
type = "file"
path = "internal/parser/parser.go"  # Implementation to test

[[payload.queries]]
type = "semantic"
question = "What are the public functions and their expected behaviors?"

[[payload.queries]]
type = "territory"
scope = "implementation"  # Related implementation files

[context]
phase = "tdd-red"
subphase = "GATHER"
awaiting = "context-results"
```

**Sources to analyze:**
| Source | Extract |
|--------|---------|
| Function signatures | What to test, parameter types |
| Error handling | Error conditions to verify |
| Edge case guards | Boundary conditions |
| Comments/docs | Expected behavior descriptions |
| Existing tests | Gaps in coverage |
| Dependencies | Mock requirements |

### 2. SYNTHESIZE

Process gathered implementation information:

1. Identify testable behaviors from code
2. Categorize tests (unit, edge case, error handling)
3. Map tests to source functions
4. Check for conflicts with existing test files
5. If blocked, yield `blocked` with details

### 3. PRODUCE

Create test file artifact:

1. Write failing tests with proper structure:

```go
func TestParseConfig_ValidInput(t *testing.T) {
    // Arrange
    input := `key: value`

    // Act
    result, err := ParseConfig(input)

    // Assert - this should fail (RED)
    g := NewGomegaWithT(t)
    g.Expect(err).To(BeNil())
    g.Expect(result.Key).To(Equal("value"))
}
```

2. Include test cases for:
   - Happy path behavior
   - Error conditions
   - Edge cases (empty input, nil, boundaries)
   - Documented invariants

3. Write to configured path from context
4. Yield `complete`:

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T11:00:00Z

[payload]
artifact = "internal/parser/parser_test.go"
test_count = 5
files_modified = ["internal/parser/parser_test.go"]

[[payload.decisions]]
context = "Test structure inference"
choice = "Table-driven tests for ParseConfig"
reason = "Multiple input variants observed in code comments"
alternatives = ["Individual test functions", "Subtest hierarchy"]

[context]
phase = "tdd-red"
subphase = "complete"
```

---

## Input Context


```toml
[invocation]
mode = "infer"
timestamp = 2026-02-02T10:00:00Z

[inputs]
task_id = "TASK-5"
source_path = "internal/parser/parser.go"
test_path = "internal/parser/parser_test.go"
territory_path = "context/territory.toml"

[config]
preserve_existing_tests = true
output_path = "internal/parser/parser_test.go"

[output]
```

---

## Yield Types

| Type | When to Use |
|------|-------------|
| `complete` | Test file created with failing tests |
| `need-context` | Need source files, semantic exploration of code |
| `need-decision` | Multiple valid test approaches |
| `blocked` | Cannot proceed (missing source, unclear behavior) |
| `error` | Something failed (retryable) |

---

## Test Inference Patterns

When analyzing existing code, look for:

| Pattern | Test Implication |
|---------|------------------|
| `if err != nil { return err }` | Error propagation test |
| `if x == nil { return default }` | Nil handling test |
| `if len(s) == 0` | Empty input test |
| `switch type.(type)` | Type-specific behavior tests |
| `// TODO:` comments | Missing behavior to document |
| Panic guards | Boundary condition tests |

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
- Inferred test rationale (why each test was created)
- Key decisions made

---

## Rules

1. **Tests must fail** - This is RED phase; tests should not pass yet
2. **Preserve existing tests** - Never overwrite working tests
3. **Focus on public API** - Test exported functions primarily
4. **Document inferences** - Explain why each test was inferred
5. **Cover error paths** - Error handling is as important as happy path

---

## Modes

| Mode | Action |
|------|--------|
| infer | Create _test.go from implementation analysis |
| update | Add new tests, preserve existing |
| gap | Identify untested code paths only |

---

## Contract

```yaml
contract:
  outputs:
    - path: "<test-file>"
      id_format: "N/A"

  traces_to:
    - "docs/tasks.md"
    - "<source-implementation>"

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
      description: "No compilation or import errors"
      severity: error

    - id: "CHECK-005"
      description: "No implementation code beyond minimal stubs"
      severity: error

    - id: "CHECK-006"
      description: "Inferred tests cover observable code behaviors"
      severity: error

    - id: "CHECK-007"
      description: "Tests describe expected behavior clearly"
      severity: warning

    - id: "CHECK-008"
      description: "Tests are blackbox (test public API, not internals)"
      severity: warning

    - id: "CHECK-009"
      description: "Existing tests preserved when updating"
      severity: warning
```
