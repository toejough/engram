# TASK-6 Test Coverage Map

**Task:** TASK-6 - Implement adaptive interview logic
**Test File:** internal/interview/interview_test.go
**Implementation File:** internal/interview/interview.go

## Coverage Summary

| Acceptance Criterion | Test Count | Status |
|---------------------|-----------|---------|
| AC-1: Code structure | 2 | ✓ Red |
| AC-2: Small gap behavior | 3 | ✓ Red |
| AC-3: Medium gap behavior | 2 | ✓ Red |
| AC-4: Large gap behavior | 2 | ✓ Red |
| AC-5: Context references | 2 | ✓ Red |
| AC-6: Skip answered topics | 2 | ✓ Red |

**Total:** 10 tests (9 failing, 1 passing - red state verified)

## Detailed Coverage

### AC-1: Code added after gap assessment, before yield

**Requirement:** Implementation must accept gap assessment results and determine which questions to ask.

**Tests:**
1. **TestSelectQuestions_SmallGap_OneCriticalUnanswered** (TEST-300)
   - Verifies function accepts GapAnalysis, KeyQuestions, and gathered context
   - Status: FAIL (stub returns 0 questions, expects ≥1)

2. **TestSelectQuestions_AllAnswered_EmptyResult** (TEST-307)
   - Verifies edge case: when all questions answered, returns empty list
   - Status: PASS (stub behavior matches expected edge case)

---

### AC-2: Small gap (≥80%) yields 1-2 confirmation questions for critical unanswered items only

**Requirement:** Small gaps ask minimal questions, focusing only on critical unanswered items.

**Tests:**
1. **TestSelectQuestions_SmallGap_OneCriticalUnanswered** (TEST-300)
   - Setup: 85% coverage, one critical unanswered (tech-stack)
   - Expects: 1-2 questions
   - Verifies: Only unanswered critical question included
   - Status: FAIL (stub returns 0 questions)

2. **TestSelectQuestions_SmallGap_NoCriticalUnanswered** (TEST-301)
   - Setup: 85% coverage, no critical unanswered, one important + one optional unanswered
   - Expects: 1-2 questions prioritizing important over optional
   - Verifies: Important question selected when no critical available
   - Status: FAIL (stub returns 0 questions)

3. **TestSelectQuestions_MediumGap_PriorityOrdering** (TEST-308)
   - Setup: Mixed priorities, verifies critical questions come first
   - Expects: Critical questions prioritized in selection
   - Status: FAIL (stub returns 0 questions)

---

### AC-3: Medium gap (50-79%) yields 3-5 clarification questions prioritizing critical then important

**Requirement:** Medium gaps ask moderate number of questions with priority ordering.

**Tests:**
1. **TestSelectQuestions_MediumGap** (TEST-302)
   - Setup: 65% coverage, 2 critical + 1 important + 1 optional unanswered
   - Expects: 3-5 questions
   - Verifies: Both critical questions included, answered questions skipped
   - Status: FAIL (stub returns 0 questions)

2. **TestSelectQuestions_MediumGap_PriorityOrdering** (TEST-308)
   - Setup: All priorities unanswered, medium gap (60% coverage)
   - Expects: 3-5 questions with critical first
   - Verifies: Critical questions appear before important/optional
   - Status: FAIL (stub returns 0 questions)

---

### AC-4: Large gap (<50%) yields full interview sequence covering all key questions

**Requirement:** Large gaps ask comprehensive questions covering all unanswered topics.

**Tests:**
1. **TestSelectQuestions_LargeGap** (TEST-303)
   - Setup: 30% coverage, 6 questions unanswered
   - Expects: ≥6 questions
   - Verifies: All unanswered questions included, answered questions skipped
   - Status: FAIL (stub returns 0 questions)

2. **TestSelectQuestions_LargeGap_AllPriorities** (TEST-309)
   - Setup: 20% coverage, all questions unanswered across all priorities
   - Expects: ≥6 questions
   - Verifies: Critical, important, and optional questions all included
   - Status: FAIL (stub returns 0 questions)

---

### AC-5: Question text references gathered context where relevant

**Requirement:** Questions should include context information from gathered sources when available.

**Tests:**
1. **TestSelectQuestions_ReferencesGatheredContext** (TEST-304)
   - Setup: 85% coverage, one critical unanswered, context available for answered question
   - Expects: ≥1 question with text field populated
   - Verifies: Question structure includes text property
   - Status: FAIL (stub returns 0 questions)

2. **TestSelectQuestions_IncludesContextInformation** (TEST-305)
   - Setup: 65% coverage, related context available
   - Expects: ≥2 questions with context field available
   - Verifies: InterviewQuestion structure supports context field
   - Status: FAIL (stub returns 0 questions)

---

### AC-6: Questions skip topics fully answered by context

**Requirement:** Questions should only ask about genuinely unanswered topics, not information already gathered.

**Tests:**
1. **TestSelectQuestions_SkipsAnsweredTopics** (TEST-306)
   - Setup: 65% coverage, 2 unanswered + 2 answered questions
   - Expects: Only unanswered questions in result
   - Verifies: scale (answered) and integrations (answered) are NOT in result
   - Verifies: tech-stack (unanswered) and deployment (unanswered) ARE in result
   - Status: FAIL (stub returns empty, doesn't include unanswered)

2. **TestSelectQuestions_AllAnswered_EmptyResult** (TEST-307)
   - Setup: 100% coverage, all questions answered
   - Expects: Empty list (no questions to ask)
   - Verifies: Function returns empty when nothing left to ask
   - Status: PASS (stub correctly returns empty)

---

## Test Patterns Used

### Example-Based Testing
All 10 tests use concrete examples with specific:
- Gap percentages (85%, 65%, 30%, 20%, 100%)
- Question counts and priorities
- Gathered context scenarios

### Behavioral Verification
Tests verify behavior chains:
1. **Structure:** Function accepts correct parameters
2. **Count:** Returns appropriate number of questions for gap size
3. **Priority:** Orders questions by priority (critical → important → optional)
4. **Filtering:** Skips answered, includes unanswered
5. **Context:** Makes context available for question text

### Gomega Assertions
- `BeNumerically(">=", n)` - Verify minimum question counts
- `BeNumerically("<=", n)` - Verify maximum question counts
- `Equal(value)` - Verify exact values (priority, ID)
- `BeTrue()` / `BeFalse()` - Verify boolean conditions
- `BeEmpty()` - Verify empty results
- `NotTo(Equal(value))` - Verify exclusions

## Test Data Patterns

### Key Question Sets
Tests use realistic question sets:
- **Minimal:** 2 questions (edge case testing)
- **Small:** 4-5 questions (typical small project)
- **Full:** 8+ questions (comprehensive architecture)

### Coverage Scenarios
- **100%:** All questions answered (small gap edge case)
- **85%:** One question unanswered (small gap)
- **65%:** Multiple questions unanswered (medium gap)
- **30%:** Most questions unanswered (large gap)
- **20%:** Very sparse context (large gap edge case)

### Priority Distributions
- **Critical only:** 2-4 questions
- **Mixed:** Critical + important + optional
- **All priorities:** Full spectrum testing

## Implementation Guidance

Based on test expectations, the implementation should:

1. **Filter unanswered questions:**
   ```go
   unansweredSet := make(map[string]bool)
   for _, id := range gapAnalysis.UnansweredQuestions {
       unansweredSet[id] = true
   }
   ```

2. **Sort by priority:**
   - Critical questions first
   - Important questions second
   - Optional questions last

3. **Apply gap size limits:**
   - Small: Take first 1-2 questions
   - Medium: Take first 3-5 questions
   - Large: Take all unanswered questions (6+)

4. **Build InterviewQuestion with context:**
   - Copy ID, Text, Priority from KeyQuestion
   - Look up context in gathered map
   - Populate Context field if available

## Traceability

**Tests trace to:**
- TASK-6 acceptance criteria (6 criteria, all covered)
- ARCH-002: Gap-Based Depth Calculation
- ARCH-004: Consistent Interview Protocol
- REQ-003: Adaptive Interview Depth

**Tests depend on:**
- internal/interview/gap.go (TASK-3) for GapAnalysis type
- KeyQuestion type with Priority enum
- Project convention: gomega assertions

## Red State Verification

✓ **Confirmed:** Tests fail because stub returns empty list
✓ **Expected:** Implementation will make tests pass by returning appropriate questions
✓ **Edge case:** One test passes (all answered → empty result) as expected
