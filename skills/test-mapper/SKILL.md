---
name: test-mapper
description: Map tests to traceability IDs
user-invocable: false
---

# Test Mapper

Map existing tests to traceability IDs by analyzing test functions and correlating with tasks/requirements. Used by `/project adopt` for codebase adoption.

## When Invoked

This skill is dispatched by `/project` orchestrator with a context file containing:
- Project directory and config paths
- Requirements, design, and architecture summaries
- Tasks.md content (if exists)

## Purpose

Connect existing tests to the traceability chain:
```
REQ → DES → ARCH → TASK → TEST
```

Tests trace to the **closest upstream** item in the chain:
- TASK (preferred - if a specific task exists for the tested functionality)
- ARCH (if no task, but architecture decision covers it)
- DES (if no task or arch, but design decision covers it)
- REQ (fallback - trace to requirement if nothing more specific exists)

## Analysis Order

1. **Discover test files** - Find all *_test.go files
2. **Parse existing comments** - Preserve existing TEST-NNN comments
3. **Analyze test functions** - Extract test names and descriptions
4. **Correlate with tasks** - Match tests to TASK-NNN
5. **Assign TEST-NNN IDs** - Add traceability comments
6. **Update traceability** - Add TEST→TASK links

## Workflow

### 1. Read Context

```bash
projctl context read --dir <project-dir> --skill test-mapper
```

Extract:
- `inputs.tasks_summary` - TASK-NNN items and descriptions
- `inputs.architecture_summary` - ARCH-NNN items

### 2. Discover Test Files

```bash
find <project-dir> -name "*_test.go" -not -path "*/vendor/*"
```

Collect all test file paths.

### 3. Parse Existing TEST Comments

Look for existing traceability comments:
```go
// TEST-001 traces: TASK-005
func TestSomething(t *testing.T) {...}
```

Preserve these - don't reassign IDs.

### 4. Analyze Test Functions

For each test file, extract:
- Function name: `TestParsePositionalArgs`
- Test description (from comments if present)
- Package being tested

Build understanding of what each test verifies.

### 5. Correlate with Upstream Items

Match tests to the **closest upstream** in priority order: TASK > ARCH > DES > REQ

**Step 1: Try to match to TASK**
- **Name correlation**: `TestParseFlag` → TASK for flag parsing
- **Package correlation**: `internal/config` tests → TASK for config
- **Comment hints**: Doc comments mentioning features

**Step 2: If no TASK match, try ARCH**
- Look for architecture decisions that cover the code being tested
- `TestConfigFS` → ARCH for "dependency injection" or "testable file system"

**Step 3: If no ARCH match, try DES**
- Look for design decisions about the behavior being tested
- `TestHelpFormat` → DES for help text formatting

**Step 4: Fallback to REQ**
- Find the requirement that the test ultimately verifies
- `TestBasicExecution` → REQ for core functionality

Create mapping (showing the hierarchy):
```
TestParsePositionalArgs → TASK-003 (Implement argument parsing)
TestConfigFS → ARCH-014 (Dependency injection for testability)
TestHelpFormat → DES-005 (Help text layout)
TestBasicExecution → REQ-001 (Basic executable behavior)
```

### 6. Handle Unmapped Tests

For tests that don't clearly map:
- Group by package/functionality
- Create escalation for confirmation

```toml
[[escalations]]
id = "ESC-005"
category = "test-mapping"
context = "Test TestUtilHelper in internal/util"
question = "Which task does this utility test relate to?"
suggested_answer = "Appears to be general utility, may not need traceability"
```

### 7. Assign TEST-NNN IDs

For tests without existing TEST-NNN:
- Assign sequential IDs: TEST-001, TEST-002, ...
- Group by task for readability

Add comment before function:
```go
// TEST-015 traces: TASK-003
func TestParsePositionalArgs(t *testing.T) {
```

### 8. Update Test Source Files

**Important:** Test tracing is done via comments IN the test source files, NOT via a separate tests.md artifact.

For each test needing a TEST-NNN assignment, add a two-line comment directly in the test file:

```go
// TEST-015: Positional argument parsing
// traces: TASK-003
func TestParsePositionalArgs(t *testing.T) {
```

**Do NOT create tests.md** - all test tracing lives in source files.

Priority for `traces:` target: TASK > ARCH > DES > REQ (pick closest upstream)

### 9. Generate Report

```markdown
## Test Mapping Report

### Mapped Tests: 45

| Test ID | Test Name | Task |
|---------|-----------|------|
| TEST-001 | TestConfigLoad | TASK-001 |
| TEST-002 | TestConfigDefaults | TASK-001 |
| TEST-015 | TestParsePositionalArgs | TASK-003 |
...

### Unmapped Tests: 3

- TestUtilHelper (escalated)
- TestBenchmarkX (benchmark, not mapped)
- TestExampleY (example, not mapped)

### Coverage

Tasks with tests: 8/12 (67%)
Tasks without tests: TASK-009, TASK-010, TASK-011, TASK-012
```

### 10. Write Result

```toml
[result]
skill = "test-mapper"
status = "needs_escalation"
timestamp = 2024-01-15T14:00:00Z

[result.summary]
text = "Mapped 45 tests to 8 tasks. 3 tests unmapped, 4 tasks need tests."

[outputs.test_files]
files_modified = 12
comments_added = 45

[outputs.traceability]
links_added = 45

[[escalations]]
id = "ESC-005"
category = "test-mapping"
context = "TestUtilHelper"
question = "Which task for utility test?"

[context_for_next]
[context_for_next.coverage]
tasks_with_tests = ["TASK-001", "TASK-002", "TASK-003", ...]
tasks_without_tests = ["TASK-009", "TASK-010", ...]
```

## TEST-NNN Comment Format

Standard format for Go:
```go
// TEST-NNN: Brief description of what test verifies
// traces: TASK-NNN
func TestFunctionName(t *testing.T) {
```

Tracing to other upstream types:
```go
// TEST-042: Verifies dependency injection pattern
// traces: ARCH-014
func TestConfigFS(t *testing.T) {

// TEST-043: Verifies help text formatting
// traces: DES-005
func TestHelpFormat(t *testing.T) {

// TEST-044: Verifies basic execution works
// traces: REQ-001
func TestBasicExecution(t *testing.T) {
```

Multiple traces (integration tests spanning multiple concerns):
```go
// TEST-050: Integration test for config loading
// traces: TASK-001, ARCH-014
func TestConfigIntegration(t *testing.T) {
```

## Mapping Heuristics

### By Name
- `TestParse*` → parsing tasks
- `TestConfig*` → config tasks
- `TestValidate*` → validation tasks

### By Package
- `internal/config/*_test.go` → config-related tasks
- `internal/parser/*_test.go` → parser-related tasks

### By Existing Comments
```go
// Tests the flag parsing functionality
func TestParseFlags(t *testing.T) {
```
→ Look for tasks mentioning "flag parsing"

## What Not to Map

- **Benchmark functions**: `BenchmarkX` - not requirements coverage
- **Example functions**: `ExampleX` - documentation, not tests
- **Helper functions**: Non-exported test helpers

## Handling Updates (Mode: update)

When re-running test-mapper:
1. Parse existing TEST-NNN comments
2. Keep existing mappings
3. Only add new tests
4. Flag tests for removed tasks

## Error Handling

- **Parse error**: Skip file, log error, continue
- **No tests found**: Report as success (nothing to map)
- **Conflicting mapping**: Escalate for user decision

## Result Format

See [shared/RESULT.md](../shared/RESULT.md) for the complete schema.

```toml
[status]
success = true

[outputs]
files_modified = ["docs/tasks.md"]

[[decisions]]
context = "Task granularity"
choice = "One task per acceptance criterion"
reason = "Clearer progress tracking"
alternatives = ["Larger tasks with multiple criteria"]

[[learnings]]
content = "Tasks should include traceability IDs"
```
