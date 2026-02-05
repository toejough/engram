#!/usr/bin/env bash
# Tests for ISSUE-055: Establish 'User Experience First' design principle
# Target file: ~/.claude/skills/design-interview-producer/SKILL.md
set -euo pipefail

SKILL_FILE="$HOME/.claude/skills/design-interview-producer/SKILL.md"
FAILURES=0
PASSES=0

fail() {
    echo "FAIL: $1"
    FAILURES=$((FAILURES + 1))
}

pass() {
    echo "PASS: $1"
    PASSES=$((PASSES + 1))
}

# Test 1: SKILL.md contains explicit UX-first principle statement
if grep -qi "user experience" "$SKILL_FILE" && grep -qi "interaction patterns" "$SKILL_FILE"; then
    pass "Contains 'user experience' and 'interaction patterns' language"
else
    fail "Missing 'user experience' and/or 'interaction patterns' language"
fi

# Test 2: SKILL.md explicitly states implementation details belong in Architecture
if grep -qi "implementation.*architecture" "$SKILL_FILE" || grep -qi "architecture.*implementation" "$SKILL_FILE"; then
    pass "States implementation details belong in Architecture phase"
else
    fail "Missing statement that implementation details belong in Architecture phase"
fi

# Test 3: The principle is prominent (in a section header, rule table, or dedicated guideline block)
# It should NOT just be buried in prose - it needs to be findable
if grep -qE "^(#|##|###|\|)" "$SKILL_FILE" | head -1 && grep -qi "user experience" "$SKILL_FILE"; then
    # Check if UX language appears near a structural element (header or table row)
    # Use awk to find if "user experience" or "UX" appears within 3 lines of a header or table row
    if awk '/^#|^\|/{found=NR} /[Uu]ser [Ee]xperience|UX/{if(found && NR-found<=3) exit 0} END{exit 1}' "$SKILL_FILE"; then
        pass "UX principle is prominently placed (near header or table)"
    else
        fail "UX principle exists but is not prominently placed"
    fi
else
    fail "UX principle not found in structural elements"
fi

# Test 4: No pseudocode or validation logic examples in the SKILL.md
# The skill should not contain code examples showing implementation (pseudocode, validation, data structures)
# TOML yield examples are fine - those are protocol, not implementation
if grep -qE "(func |def |class |if.*err|switch.*case|for.*range)" "$SKILL_FILE" 2>/dev/null; then
    fail "SKILL.md contains implementation code patterns (pseudocode/validation logic)"
else
    pass "No implementation code patterns in SKILL.md"
fi

# Test 5: Interview questions should focus on UX, not implementation
# The "Yield need-user-input for:" section should mention UX-oriented topics
if grep -A10 "need-user-input" "$SKILL_FILE" | grep -qi "visual\|layout\|interaction\|accessibility\|workflow\|user"; then
    pass "Interview questions focus on UX topics"
else
    fail "Interview questions don't clearly focus on UX topics"
fi

# Test 6: Explicit anti-pattern guidance - what NOT to ask about in design
if grep -qi "do not\|avoid\|never" "$SKILL_FILE" && grep -qi "file format\|validation logic\|data structure\|implementation detail" "$SKILL_FILE"; then
    pass "Contains explicit anti-pattern guidance (what to avoid in design)"
else
    fail "Missing explicit anti-pattern guidance for implementation details"
fi

# Test 7: The GATHER phase mentions UX focus
if awk '/### 1. GATHER/,/### 2. SYNTHESIZE/' "$SKILL_FILE" | grep -qi "user experience\|UX\|interaction pattern\|workflow"; then
    pass "GATHER phase mentions UX/interaction focus"
else
    fail "GATHER phase does not mention UX/interaction focus"
fi

echo ""
echo "Results: $PASSES passed, $FAILURES failed"
exit $FAILURES
