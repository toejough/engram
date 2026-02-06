#!/bin/bash
# SKILL_test.sh - Tests for simplified step-driven project orchestrator
# ISSUE-90: Simplify orchestrator SKILL.md for step-driven execution

set -euo pipefail

SKILL_FILE="$(dirname "$0")/SKILL.md"
SKILL_FULL_FILE="$(dirname "$0")/SKILL-full.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

FAILURES=0

pass() { echo -e "${GREEN}PASS${NC}: $1"; }
fail() { echo -e "${RED}FAIL${NC}: $1"; FAILURES=$((FAILURES + 1)); }

echo "=== Simplified Project Orchestrator Tests ==="
echo ""

# --- Structural tests ---

echo "--- Structure ---"

echo "Test: SKILL.md exists"
[[ -f "$SKILL_FILE" ]] || { fail "SKILL.md not found"; }
[[ -f "$SKILL_FILE" ]] && pass "SKILL.md exists"

echo "Test: Has required frontmatter fields"
if grep -q "^name: project" "$SKILL_FILE" && grep -q "^user-invocable: true" "$SKILL_FILE"; then
  pass "Has required frontmatter (name, user-invocable)"
else
  fail "Missing name or user-invocable frontmatter field"
fi

echo "Test: Frontmatter sets model to haiku"
if grep -q "^model: haiku" "$SKILL_FILE"; then
  pass "Frontmatter sets model: haiku"
else
  fail "Frontmatter missing model: haiku"
fi

# --- Content that must REMAIN ---

echo ""
echo "--- Required Content ---"

echo "Test: Team lifecycle - spawn team"
if grep -qi "spawnTeam\|spawn.*team" "$SKILL_FILE"; then
  pass "Documents team spawning"
else
  fail "Missing team spawning documentation"
fi

echo "Test: Team lifecycle - shutdown"
if grep -qi "shutdown" "$SKILL_FILE"; then
  pass "Documents shutdown"
else
  fail "Missing shutdown documentation"
fi

echo "Test: Intake flow present"
if grep -qi "intake.*flow\|intake-evaluator" "$SKILL_FILE"; then
  pass "Documents intake flow"
else
  fail "Missing intake flow"
fi

echo "Test: Context-only contract present"
if grep -qi "context-only contract\|context.*only" "$SKILL_FILE"; then
  pass "Documents context-only contract"
else
  fail "Missing context-only contract"
fi

echo "Test: Looper pattern present"
if grep -qi "looper.*pattern\|looper" "$SKILL_FILE"; then
  pass "Documents looper pattern"
else
  fail "Missing looper pattern"
fi

echo "Test: Escalation handling present"
if grep -qi "escalat" "$SKILL_FILE"; then
  pass "Documents escalation handling"
else
  fail "Missing escalation handling"
fi

echo "Test: End-of-command sequence present"
if grep -qi "end-of-command\|end of command" "$SKILL_FILE"; then
  pass "Documents end-of-command sequence"
else
  fail "Missing end-of-command sequence"
fi

# --- Step-driven loop ---

echo ""
echo "--- Step-Driven Loop ---"

echo "Test: References projctl step next"
if grep -q "projctl step next" "$SKILL_FILE"; then
  pass "References projctl step next"
else
  fail "Missing projctl step next"
fi

echo "Test: References projctl step complete"
if grep -q "projctl step complete" "$SKILL_FILE"; then
  pass "References projctl step complete"
else
  fail "Missing projctl step complete"
fi

echo "Test: Documents step loop control flow"
if grep -qi "step.*loop\|control.*loop\|step-driven" "$SKILL_FILE"; then
  pass "Documents step-driven control loop"
else
  fail "Missing step-driven control loop documentation"
fi

echo "Test: Documents spawn-producer action"
if grep -q "spawn-producer" "$SKILL_FILE"; then
  pass "Documents spawn-producer action"
else
  fail "Missing spawn-producer action"
fi

echo "Test: Documents spawn-qa action"
if grep -q "spawn-qa" "$SKILL_FILE"; then
  pass "Documents spawn-qa action"
else
  fail "Missing spawn-qa action"
fi

echo "Test: Documents commit action"
if grep -q '"commit"' "$SKILL_FILE" || grep -q "'commit'" "$SKILL_FILE" || grep -qi "action.*commit" "$SKILL_FILE"; then
  pass "Documents commit action"
else
  fail "Missing commit action"
fi

echo "Test: Documents transition action"
if grep -q "transition" "$SKILL_FILE"; then
  pass "Documents transition action"
else
  fail "Missing transition action"
fi

echo "Test: Documents all-complete action"
if grep -q "all-complete" "$SKILL_FILE"; then
  pass "Documents all-complete action"
else
  fail "Missing all-complete action"
fi

echo "Test: Documents JSON output structure from step next"
if grep -q '"action"' "$SKILL_FILE" && grep -q '"skill"' "$SKILL_FILE" && grep -q '"skill_path"' "$SKILL_FILE"; then
  pass "Documents step next JSON output structure"
else
  fail "Missing step next JSON output structure"
fi

echo "Test: QA approved uses --qa-verdict approved flag"
if grep -q "\-\-qa-verdict approved" "$SKILL_FILE"; then
  pass "QA approved uses correct --qa-verdict flag"
else
  fail "QA approved missing --qa-verdict approved flag (code requires it)"
fi

echo "Test: QA improvement-request uses --qa-verdict and --qa-feedback flags"
if grep -q "\-\-qa-verdict improvement-request" "$SKILL_FILE" && grep -q "\-\-qa-feedback" "$SKILL_FILE"; then
  pass "QA improvement-request uses correct flags"
else
  fail "QA improvement-request missing --qa-verdict/--qa-feedback flags"
fi

echo "Test: No --status retry (invalid status value)"
if grep -q "\-\-status retry" "$SKILL_FILE"; then
  fail "Contains --status retry which is not a valid status value"
else
  pass "No invalid --status retry"
fi

# --- Model Validation (ISSUE-98, TASK-5) ---

echo ""
echo "--- Model Validation ---"

echo "Test: References expected_model field"
if grep -q "expected_model" "$SKILL_FILE"; then
  pass "References expected_model field"
else
  fail "Missing expected_model reference"
fi

echo "Test: Contains --reported-model flag usage"
if grep -q "\-\-reported-model" "$SKILL_FILE"; then
  pass "Contains --reported-model flag usage"
else
  fail "Missing --reported-model flag usage"
fi

echo "Test: Contains escalate-user handling instructions"
if grep -q "escalate-user" "$SKILL_FILE"; then
  pass "Contains escalate-user handling instructions"
else
  fail "Missing escalate-user handling instructions"
fi

echo "Test: References task_params for spawn execution"
if grep -qi "task_params" "$SKILL_FILE"; then
  pass "References task_params for spawn execution"
else
  fail "Missing task_params reference for spawn execution"
fi

echo "Test: Contains model validation/handshake instructions"
if grep -qi "model.*valid\|handshake\|verify.*model\|model.*verif" "$SKILL_FILE"; then
  pass "Contains model validation/handshake instructions"
else
  fail "Missing model validation/handshake instructions"
fi

# --- Content that must be REMOVED ---

echo ""
echo "--- Removed Content ---"

echo "Test: No skill dispatch table"
# The old SKILL.md had a table mapping Phase -> Producer -> QA
# with explicit skill names like pm-interview-producer, design-interview-producer, etc.
# The new one should NOT have a table with all phase-to-skill mappings
# (projctl step next returns the skill name)
if grep -q "^| Phase " "$SKILL_FILE" && grep -q "pm-interview-producer" "$SKILL_FILE" && grep -q "arch-interview-producer" "$SKILL_FILE"; then
  fail "Still contains skill dispatch table (projctl step next provides skill names)"
else
  pass "No skill dispatch table"
fi

echo "Test: No PAIR LOOP pattern section"
# The old SKILL.md had a "## PAIR LOOP Pattern" section with the 6-step pattern
# The new one delegates this to projctl step next/complete
if grep -q "## PAIR LOOP Pattern" "$SKILL_FILE" || grep -q "## PAIR LOOP" "$SKILL_FILE"; then
  fail "Still contains PAIR LOOP Pattern section (projctl enforces this)"
else
  pass "No PAIR LOOP pattern section"
fi

echo "Test: No phase dispatch tables with hardcoded phase order"
# Old SKILL.md had a Flows table showing PM -> Design -> Arch -> Breakdown -> Implementation
# The new one should not hardcode the phase order (projctl step next knows the order)
if grep -q "PM.*Design.*Arch.*Breakdown.*Implementation" "$SKILL_FILE"; then
  fail "Still contains hardcoded phase order (projctl step next provides order)"
else
  pass "No hardcoded phase order"
fi

echo "Test: No resume map"
# Old SKILL-full.md had a Resume Map section with phase-to-action mappings
# The new SKILL.md should not contain a resume map
if grep -qi "resume map" "$SKILL_FILE"; then
  fail "Still contains resume map (projctl tracks state)"
else
  pass "No resume map"
fi

echo "Test: No reference to SKILL-full.md"
if grep -qi "SKILL-full" "$SKILL_FILE"; then
  fail "Still references SKILL-full.md (should be eliminated)"
else
  pass "No reference to SKILL-full.md"
fi

echo "Test: SKILL-full.md eliminated"
if [[ -f "$SKILL_FULL_FILE" ]]; then
  fail "SKILL-full.md still exists (should be eliminated)"
else
  pass "SKILL-full.md eliminated"
fi

# --- Summary ---

echo ""
if [[ $FAILURES -eq 0 ]]; then
  echo "=== All tests passed ==="
else
  echo "=== $FAILURES test(s) FAILED ==="
  exit 1
fi
