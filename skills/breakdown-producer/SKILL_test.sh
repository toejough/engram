#!/bin/bash
# Tests for breakdown-producer SKILL.md

set -e

SKILL_FILE="skills/breakdown-producer/SKILL.md"
REPO_ROOT="$(git rev-parse --show-toplevel)"

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

# Test 4: Simplicity is holistic (SYNTHESIZE), not per-task (no per-task field in template)
if grep -A 50 "## Task Format" "$SKILL_FILE" | grep -q "Simplicity Assessment"; then
  echo "FAIL: Task format should NOT include per-task Simplicity Assessment (moved to SYNTHESIZE)"
  exit 1
fi
echo "PASS: Task format does not include per-task Simplicity Assessment"

# Test 5: No per-task simplicity contract check (CHECK-012 removed)
if grep -q "CHECK-012" "$SKILL_FILE"; then
  echo "FAIL: CHECK-012 (per-task simplicity) should be removed"
  exit 1
fi
echo "PASS: CHECK-012 removed from contract"

# Test 6: Simplicity rationale is in tasks.md header (PRODUCE phase output)
if ! grep -q "Simplicity rationale" "$SKILL_FILE"; then
  echo "FAIL: PRODUCE phase should include simplicity rationale in tasks.md header"
  exit 1
fi
echo "PASS: PRODUCE phase includes simplicity rationale in header"

# Test 7: SYNTHESIZE phase asks the right simplicity question
if ! grep -q "simpler approach" "$SKILL_FILE"; then
  echo "FAIL: SYNTHESIZE phase does not ask about simpler approaches"
  exit 1
fi
echo "PASS: SYNTHESIZE phase asks about simpler approaches"

# Test 8: SYNTHESIZE simplicity step documents alternatives considered
if ! grep -A 10 "Assess simplicity" "$SKILL_FILE" | grep -q "alternatives considered"; then
  echo "FAIL: SYNTHESIZE simplicity step does not mention documenting alternatives"
  exit 1
fi
echo "PASS: SYNTHESIZE simplicity step documents alternatives"

echo ""
echo "All tests passed!"
