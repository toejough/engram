# TASK-2 Test Coverage Report

**Task:** Add key questions registry to arch-interview-producer
**Test File:** `.claude/projects/ISSUE-61/tests/task2_key_questions_test.sh`
**Status:** RED (all tests failing as expected)

## Test Summary

- **Total Tests:** 23
- **Passing:** 0
- **Failing:** 23
- **Test Type:** Documentation structure tests (word/phrase matching + structural validation)

## Coverage by Acceptance Criteria

### AC-1: Section Existence
Tests that `## Key Questions` section is added to the SKILL.md file.

**Tests:**
- `AC-1: ## Key Questions section exists` - Verifies section header present

### AC-2: Question Coverage (8-12 questions covering required topics)
Tests that all required topics are covered with appropriate question count.

**Tests:**
- `AC-2a: Question count within range (8-12)` - Counts questions matching format `**Topic** - Question?`
- `AC-2b: Technology stack question exists` - Checks for "technology stack" or "languages/frameworks"
- `AC-2c: Scale question exists` - Checks for "scale" with "users" or "data volume"
- `AC-2d: Deployment question exists` - Checks for "deployment" or "where.*run"
- `AC-2e: Integrations question exists` - Checks for "integration" or "external system"
- `AC-2f: Performance question exists` - Checks for "performance", "response time", or "SLA"
- `AC-2g: Security question exists` - Checks for "security", "authentication", or "authorization"
- `AC-2h: Data durability question exists` - Checks for "data durability" or "data loss"
- `AC-2i: Observability question exists` - Checks for "observability", "logging", or "monitoring"

### AC-3: Priority Tagging
Tests that questions are tagged with priorities and counts match specification.

**Tests:**
- `AC-3a: Critical priority tag exists` - Checks for "(critical)" tag
- `AC-3b: Important priority tag exists` - Checks for "(important)" tag
- `AC-3c: Optional priority tag exists` - Checks for "(optional)" tag
- `AC-3d: Critical priority count within range (2-4)` - Counts critical tags
- `AC-3e: Important priority count within range (3-5)` - Counts important tags
- `AC-3f: Optional priority count within range (2-3)` - Counts optional tags

### AC-4: Question Format
Tests that questions follow the specified format.

**Tests:**
- `AC-4: Question format includes topic, question text, and priority` - Regex check for `**Topic** - Question? (priority)` pattern

### AC-5: Coverage Weights Documentation
Tests that weight values are documented in the section.

**Tests:**
- `AC-5a: Critical weight (-15%) documented` - Checks for "critical" near "-15%"
- `AC-5b: Important weight (-10%) documented` - Checks for "important" near "-10%"
- `AC-5c: Optional weight (-5%) documented` - Checks for "optional" near "-5%"

### AC-6: Examples
Tests that examples are provided showing question-to-decision mapping.

**Tests:**
- `AC-6: Examples showing question mapping provided` - Checks for "example" within Key Questions section

## Test Execution

```bash
# Run tests
bash .claude/projects/ISSUE-61/tests/task2_key_questions_test.sh

# Expected output (red state):
# FAIL: AC-1 - ## Key Questions section missing
# Exit code: 1
```

## Verification of Red State

Test execution confirms red state:
- First test (AC-1) fails immediately because the section doesn't exist
- All 23 tests fail as expected
- Failures are correct (feature doesn't exist, not broken tests)

## Test Philosophy Applied

### Human-Readable Assertions
Each test prints clear PASS/FAIL messages with context:
```bash
echo "PASS: AC-2a - Question count within range (found: $question_count)"
echo "FAIL: AC-2a - Question count not in 8-12 range (found: $question_count)"
```

### Structural Testing
Tests verify document structure through:
- Section headers (`^## Key Questions`)
- Question format patterns (`**Topic** - Question? (priority)`)
- Count ranges (8-12 questions, 2-4 critical, etc.)

### Word/Phrase Matching
Tests check for required terminology:
- Topic keywords (technology stack, scale, deployment, etc.)
- Priority tags (critical, important, optional)
- Weight values (-15%, -10%, -5%)

### Section-Scoped Validation
AC-6 test extracts the Key Questions section content to avoid false positives from other sections:
```bash
section_start=$(grep -n '^## Key Questions' "$SKILL_FILE" | cut -d: -f1)
section_content=$(tail -n +$section_start "$SKILL_FILE" | head -n $section_end)
```

## Traceability

All tests trace to TASK-2 acceptance criteria:
- TASK-2 traces to ARCH-005, REQ-003, REQ-004
- Tests ensure architecture decisions about interview depth are properly documented

## Next Steps

After implementation (tdd-green-producer):
1. Run tests to verify green state
2. All 23 tests should pass
3. Verify question content makes sense (manual review)
4. Proceed to TASK-3 (coverage calculation logic)
