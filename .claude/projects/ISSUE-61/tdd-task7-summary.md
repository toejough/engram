# TASK-7 TDD Red Phase Summary

## Task Details

**Task ID:** TASK-7
**Description:** Add yield context enrichment
**Summary:** Enrich yield [context] section with gap analysis metadata for observability and debugging

## Acceptance Criteria Coverage

### AC-1: Yield includes [context.gap_analysis] section
**Tests:**
- TEST-400: Validates yield with gap_analysis section is accepted
- TEST-416: Validates gap_analysis is optional for non-interview yields

### AC-2: Required fields present
**Tests:**
- TEST-401: All required fields can be parsed and accessed
- TEST-402: Missing total_key_questions fails validation
- TEST-403: Missing questions_answered fails validation
- TEST-404: Missing coverage_percent fails validation
- TEST-405: Missing gap_size fails validation
- TEST-406: Missing question_count fails validation
- TEST-411: Invalid gap_size value fails validation
- TEST-412: Valid gap_size values (small/medium/large) pass validation
- TEST-413: coverage_percent must be between 0 and 100
- TEST-414: question_count must be non-negative

### AC-3: Sources array field
**Tests:**
- TEST-407: Missing sources field fails validation
- TEST-408: Sources array correctly lists context mechanisms (territory/memory/context-files)

### AC-4: unanswered_critical field
**Tests:**
- TEST-409: unanswered_critical field is optional
- TEST-410: unanswered_critical lists critical questions not answered

### AC-5: Yield validates against schema
**Tests:**
- TEST-415: Complete yield with gap_analysis passes full validation

## Test Files Created

**File:** `/Users/joe/repos/personal/projctl/internal/yield/enrichment_test.go`
- 17 test functions (TEST-400 through TEST-416)
- Uses gomega matchers for readable assertions
- Tests cover positive and negative cases
- Tests verify field presence, types, and validation rules

## Red State Verification

**Build Status:** FAIL (expected)
```
internal/yield/enrichment_test.go:76:31: undefined: yield.ParseContent
internal/yield/enrichment_test.go:305:31: undefined: yield.ParseContent
internal/yield/enrichment_test.go:374:31: undefined: yield.ParseContent
```

**Reason for Failure:**
1. `yield.ParseContent()` function does not exist yet (needed for tests that verify parsed field values)
2. `ContextSection.GapAnalysis` field does not exist yet (will fail when ParseContent is implemented)
3. Gap analysis validation logic does not exist yet (will fail once parsing works)

These failures confirm the tests are correctly written for the RED phase - they specify the expected behavior before implementation exists.

## Implementation Guidance

To make these tests pass (GREEN phase):

1. **Add GapAnalysis struct to yield.go:**
   ```go
   type GapAnalysis struct {
       TotalKeyQuestions   int      `toml:"total_key_questions"`
       QuestionsAnswered   int      `toml:"questions_answered"`
       CoveragePercent     float64  `toml:"coverage_percent"`
       GapSize             string   `toml:"gap_size"`
       QuestionCount       int      `toml:"question_count"`
       Sources             []string `toml:"sources"`
       UnansweredCritical  []string `toml:"unanswered_critical"`
   }
   ```

2. **Add GapAnalysis field to ContextSection:**
   ```go
   type ContextSection struct {
       Phase        string        `toml:"phase"`
       Subphase     string        `toml:"subphase"`
       Iteration    int           `toml:"iteration"`
       Task         string        `toml:"task"`
       Awaiting     string        `toml:"awaiting"`
       Role         string        `toml:"role"`
       GapAnalysis  *GapAnalysis  `toml:"gap_analysis"`
   }
   ```

3. **Add ParseContent function:**
   ```go
   func ParseContent(content string) (*YieldFile, error) {
       var y YieldFile
       _, err := toml.Decode(content, &y)
       if err != nil {
           return nil, err
       }
       return &y, nil
   }
   ```

4. **Add validation for gap_analysis fields in ValidateContent:**
   - Check required fields when gap_analysis section exists
   - Validate gap_size is one of: small, medium, large
   - Validate coverage_percent is 0-100
   - Validate question_count is non-negative
   - Validate sources array is not empty

## Test Coverage Summary

| Category | Tests | Coverage |
|----------|-------|----------|
| Structure | 2 | Section presence, optionality |
| Required Fields | 8 | All 6 required fields + validation |
| Optional Fields | 2 | unanswered_critical behavior |
| Validation Rules | 5 | Range checks, enum validation |
| **Total** | **17** | **Complete AC coverage** |

## Traceability

All tests trace to TASK-7 acceptance criteria:
- AC-1: Gap analysis section → TEST-400, TEST-416
- AC-2: Required fields → TEST-401-406, TEST-411-414
- AC-3: Sources array → TEST-407, TEST-408
- AC-4: unanswered_critical → TEST-409, TEST-410
- AC-5: Validation → TEST-415

## Next Steps

1. Implement the missing types and functions in `/Users/joe/repos/personal/projctl/internal/yield/yield.go`
2. Run tests again to verify GREEN state
3. Update `/Users/joe/.claude/skills/arch-interview-producer/SKILL.md` with example yield showing gap_analysis section (AC-6)
