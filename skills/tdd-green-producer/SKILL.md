---
name: tdd-green-producer
description: Write minimal implementation to make tests pass (TDD green phase)
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: tdd-green
---

# TDD Green Producer

Write minimal implementation code to make failing tests pass.

## Workflow: GATHER -> SYNTHESIZE -> PRODUCE

This skill follows the producer pattern from [PRODUCER-TEMPLATE](../shared/PRODUCER-TEMPLATE.md).

### GATHER Phase

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode):
   - Test file locations
   - Architecture notes
   - TASK-N being implemented
2. Check for `[query_results]` (resuming after need-context)
3. If missing test files or architecture context:
   - Yield `need-context` with file queries
4. Proceed to SYNTHESIZE when test expectations are clear

### SYNTHESIZE Phase

1. Analyze failing tests to understand expected behavior
2. Identify minimal code changes needed
3. Check existing patterns in codebase
4. If blocked by missing information, yield `blocked`
5. Prepare implementation plan

### PRODUCE Phase

1. Write minimal implementation to pass tests
2. **All targeted tests must pass** - no exceptions
3. Run full test suite to verify no regressions
4. Yield `complete` with files modified

## Visual Verification

When the task has `[visual]` marker in its title (e.g., `TASK-5: [visual] Add carousel buttons`):

### Detection

Check if the task title contains `[visual]`. This marker indicates user-visible output that requires visual verification beyond passing tests.

### Capture Visual Evidence

After tests pass, capture visual evidence:

| Interface | Capture Method | Tool |
|-----------|----------------|------|
| Web UI | Browser screenshot | `mcp__chrome-devtools__take_screenshot` |
| CLI | Output redirection | `command > output.txt 2>&1` |
| CLI (ANSI) | Script recording | `script -q output.txt command` |
| Desktop app | Manual screenshot | System screenshot tool |

### Compare Against Expectation

- **If design spec/baseline exists**: `projctl screenshot diff --baseline <spec> --current <screenshot>`
- **If no baseline**: Manual review of captured output to verify it matches acceptance criteria

### Document Evidence

Include visual verification in yield payload:
- `visual_verified = true` confirms verification was performed
- `visual_evidence` path to screenshot or output capture

### No Screenshot Capture Tool

If dedicated capture tooling is unavailable:

1. **Web UI**: Use Chrome DevTools MCP - `mcp__chrome-devtools__take_screenshot`
2. **CLI plain text**: Redirect output to file - `cmd > output.txt 2>&1`
3. **CLI with ANSI colors**: Use `script` command - `script -q output.txt command`
4. **Desktop/native apps**: Use system screenshot tools manually

Visual verification is required even without dedicated tooling. The above methods provide sufficient evidence for QA review.

## Rules

| Rule | Rationale |
|------|-----------|
| MINIMAL code only | Don't over-engineer |
| NO refactoring | That's tdd-refactor's job |
| ALL tests must pass | Non-negotiable exit criteria |
| Follow arch patterns | Consistency with codebase |
| NO new tests | That's tdd-red's job |

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

Minimal fix - add exactly the section the test expects:
```markdown
## Acceptance Criteria

[Add criteria here]
```

**Example 2: Semantic test fails**

Test: README must explain "how to install the tool" (similarity >= 0.7)
Failure: Score is 0.45 - concept not conveyed strongly enough

Minimal fix - add installation section with clear, direct language:
```markdown
## Installation

To install the tool, run:
\`\`\`bash
go install github.com/example/tool@latest
\`\`\`
```

Don't add extra sections or elaborate beyond what's needed to pass the semantic match.

## Debugging Heuristics

| Issue | Check |
|-------|-------|
| Struct field changes | Grep for copy/clone logic - new fields stay zero |
| Multiple code paths | Similar operations have multiple paths - fix all |
| Accumulated flags | Flags that only turn on need phase/state checks |
| Still failing | Trace what happens AFTER the code runs |

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full specification.

### Complete Yield

When all tests pass:

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
artifact = "internal/foo/bar.go"
files_modified = ["internal/foo/bar.go", "internal/foo/baz.go"]
tests_passing = ["TestFoo", "TestBar"]

[[payload.decisions]]
context = "Implementation approach"
choice = "Used existing pattern from pkg/util"
reason = "Consistency with codebase"

[context]
phase = "tdd-green"
task = "TASK-5"
subphase = "complete"
```

### Complete Yield with Visual Evidence

For tasks with `[visual]` marker:

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
artifact = "components/Button.tsx"
files_modified = ["components/Button.tsx", "components/Button.css"]
tests_passing = ["TestButtonLoading", "TestButtonDisabled"]
visual_verified = true
visual_evidence = "screenshots/button-loading-state.png"

[[payload.decisions]]
context = "Visual verification"
choice = "Screenshot captured and reviewed"
reason = "Matches design spec"

[context]
phase = "tdd-green"
task = "TASK-7"
subphase = "complete"
```

### Need-Context Yield

When missing information:

```toml
[yield]
type = "need-context"
timestamp = 2026-02-02T10:35:00Z

[[payload.queries]]
type = "file"
path = "internal/foo/bar_test.go"

[[payload.queries]]
type = "semantic"
question = "How is error handling implemented in this package?"

[context]
phase = "tdd-green"
task = "TASK-5"
subphase = "GATHER"
awaiting = "context-results"
```

### Blocked Yield

When cannot proceed:

```toml
[yield]
type = "blocked"
timestamp = 2026-02-02T10:40:00Z

[payload]
blocker = "Test expects behavior not defined in architecture"
details = "TestFoo expects caching but ARCH-3 doesn't mention it"
suggested_resolution = "Clarify caching requirements in architecture"

[context]
phase = "tdd-green"
task = "TASK-5"
awaiting = "blocker-resolution"
```

## Failure Recovery

| Symptom | Action |
|---------|--------|
| Tests still fail after implementation | Re-read test expectations carefully |
| Existing tests break | Fix them - never dismiss as "pre-existing" |
| Stuck after 3 attempts | Yield `blocked` with detailed findings |
| Architecture unclear | Yield `need-context` for semantic exploration |

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
- Artifact paths (implementation files created/modified)
- Test results summary (all passing)
- Files modified
- Visual evidence path (for `[visual]` tasks)
- Key decisions made

### Legacy Mode (yield protocol)

| Yield Type | When Used |
|------------|-----------|
| `complete` | All tests pass with minimal implementation |
| `need-context` | Need test files, architecture context |
| `blocked` | Cannot proceed (test expectations unclear) |
| `error` | Something failed |

See [YIELD.md](../shared/YIELD.md) for yield format examples.

---

## Contract

```yaml
contract:
  outputs:
    - path: "<implementation-file>"
      id_format: "N/A"

  traces_to:
    - "docs/tasks.md"
    - "<test-file>"

  checks:
    - id: "CHECK-001"
      description: "All new tests from red phase pass"
      severity: error

    - id: "CHECK-002"
      description: "All existing tests still pass (no regressions)"
      severity: error

    - id: "CHECK-003"
      description: "Implementation is minimal (no over-engineering)"
      severity: error

    - id: "CHECK-004"
      description: "No new tests added (that's tdd-red's job)"
      severity: error

    - id: "CHECK-005"
      description: "Build succeeds with no errors"
      severity: error

    - id: "CHECK-006"
      description: "Implementation follows architecture patterns"
      severity: warning

    - id: "CHECK-007"
      description: "Visual verification for tasks with [visual] marker"
      severity: warning
```
