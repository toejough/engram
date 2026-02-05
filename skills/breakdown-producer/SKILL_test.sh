#!/bin/bash
# Tests for breakdown-producer SKILL.md

set -e

SKILL_FILE="skills/breakdown-producer/SKILL.md"
REPO_ROOT="/Users/joe/repos/personal/projctl"

cd "$REPO_ROOT"

echo "Running breakdown-producer SKILL.md tests..."

# Test 1: SKILL.md exists
if [[ ! -f "$SKILL_FILE" ]]; then
  echo "FAIL: $SKILL_FILE does not exist"
  exit 1
fi
echo "PASS: SKILL.md exists"

# Test 2: SYNTHESIZE phase mentions simplicity assessment
if ! grep -q "simplicity" "$SKILL_FILE"; then
  echo "FAIL: SYNTHESIZE phase does not mention simplicity assessment"
  exit 1
fi
echo "PASS: SYNTHESIZE phase mentions simplicity"

# Test 3: SYNTHESIZE phase has explicit simplicity check step
if ! grep -A 10 "## SYNTHESIZE Phase" "$SKILL_FILE" | grep -q "simplicity"; then
  echo "FAIL: SYNTHESIZE phase does not include simplicity assessment step"
  exit 1
fi
echo "PASS: SYNTHESIZE phase includes simplicity assessment step"

# Test 4: PRODUCE phase task format includes Simplicity Assessment section
if ! grep -q "Simplicity Assessment" "$SKILL_FILE"; then
  echo "FAIL: Task format does not include Simplicity Assessment section"
  exit 1
fi
echo "PASS: Task format includes Simplicity Assessment section"

# Test 5: Contract checks include simplicity assessment validation
if ! grep -A 100 "## Contract" "$SKILL_FILE" | grep -i "simplicity"; then
  echo "FAIL: Contract does not validate simplicity assessment"
  exit 1
fi
echo "PASS: Contract validates simplicity assessment"

# Test 6: Simplicity Assessment appears in task format example
if ! grep -A 50 "## Task Format" "$SKILL_FILE" | grep -q "Simplicity Assessment"; then
  echo "FAIL: Task format example does not include Simplicity Assessment"
  exit 1
fi
echo "PASS: Task format example includes Simplicity Assessment"

# Test 7: SYNTHESIZE phase asks the right simplicity question
if ! grep -q "simpler approach" "$SKILL_FILE"; then
  echo "FAIL: SYNTHESIZE phase does not ask about simpler approaches"
  exit 1
fi
echo "PASS: SYNTHESIZE phase asks about simpler approaches"

# Test 8: Document explains what simplicity assessment should contain
if ! grep -A 10 "Simplicity Assessment" "$SKILL_FILE" | grep -q "alternative"; then
  echo "FAIL: Simplicity Assessment section does not mention alternatives"
  exit 1
fi
echo "PASS: Simplicity Assessment explains content requirements"

echo ""
echo "All tests passed!"
