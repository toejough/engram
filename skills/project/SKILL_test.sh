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

# --- ISSUE-104 TASK-4: Model Handshake Validation ---

echo ""
echo "--- ISSUE-104 TASK-4: Model Handshake Validation ---"

echo "Test: spawn-producer requires model handshake validation"
if grep -A 20 "#### spawn-producer" "$SKILL_FILE" | grep -qi "handshake\|validate.*model\|verify.*model"; then
  pass "spawn-producer requires model handshake validation"
else
  fail "spawn-producer missing model handshake validation requirement"
fi

echo "Test: spawn-qa requires model handshake validation"
if grep -A 20 "#### spawn-producer.*spawn-qa\|#### spawn-qa" "$SKILL_FILE" | grep -qi "handshake\|validate.*model\|verify.*model"; then
  pass "spawn-qa requires model handshake validation"
else
  fail "spawn-qa missing model handshake validation requirement"
fi

echo "Test: Handshake validation is case-insensitive"
if grep -qi "case-insensitive.*match\|case.*insensitive.*substring" "$SKILL_FILE"; then
  pass "Handshake validation specified as case-insensitive"
else
  fail "Missing case-insensitive requirement for handshake validation"
fi

echo "Test: Handshake success path documented"
if grep -qi "match.*send.*confirmation\|success.*confirmation\|handshake.*success" "$SKILL_FILE"; then
  pass "Handshake success path documented"
else
  fail "Missing handshake success path documentation"
fi

echo "Test: Handshake failure path uses --status failed"
if grep -q "\-\-status failed" "$SKILL_FILE"; then
  pass "Handshake failure path uses --status failed"
else
  fail "Missing --status failed for handshake failure"
fi

echo "Test: Handshake failure reports model with --reported-model flag"
# Already tested above, but verify it's in context of handshake failure
if grep -B 5 -A 5 "\-\-reported-model" "$SKILL_FILE" | grep -qi "mismatch\|fail\|handshake"; then
  pass "Handshake failure includes --reported-model flag"
else
  fail "Missing --reported-model in handshake failure context"
fi

echo "Test: Team lead validates first message from teammate"
if grep -qi "first message\|teammate.*first\|after spawning.*read" "$SKILL_FILE"; then
  pass "Team lead validates first message from teammate"
else
  fail "Missing instruction to read/validate first message from teammate"
fi

echo "Test: Handshake failure prevents teammate from continuing work"
if grep -B 2 -A 2 "Mismatch\|handshake fail" "$SKILL_FILE" | grep -qi "do not.*continue\|immediately\|not.*let.*continue"; then
  pass "Handshake failure prevents teammate from continuing work"
else
  fail "Missing instruction to prevent work on handshake failure"
fi

echo "Test: Handshake success sends confirmation message to orchestrator"
if grep -qi "sendmessage.*orchestrator\|send.*message.*to.*orchestrator.*confirmation" "$SKILL_FILE"; then
  pass "Handshake success sends confirmation via SendMessage"
else
  fail "Missing SendMessage instruction for handshake success confirmation"
fi

echo "Test: Confirmation message includes spawn-confirmed format"
if grep -q "spawn-confirmed" "$SKILL_FILE"; then
  pass "Confirmation message uses spawn-confirmed format"
else
  fail "Missing spawn-confirmed message format specification"
fi

echo "Test: Handshake failure sends failure message to orchestrator"
if grep -B 5 -A 5 "Mismatch\|handshake.*fail" "$SKILL_FILE" | grep -qi "send.*failure.*message\|send.*error.*message.*orchestrator"; then
  pass "Handshake failure sends error message to orchestrator"
else
  fail "Missing instruction to send failure message to orchestrator on handshake failure"
fi

# --- ISSUE-104 TASK-6: Error Handling with Retry-Backoff ---

echo ""
echo "--- ISSUE-104 TASK-6: Error Handling with Retry-Backoff ---"

echo "Test: SKILL-full.md documents wrapping projctl step next with retry"
if grep -qi "wrap.*step next.*retry\|retry.*wrap.*step next" "$SKILL_FULL_FILE"; then
  pass "Documents wrapping projctl step next with retry"
else
  fail "Missing documentation for wrapping projctl step next with retry"
fi

echo "Test: SKILL-full.md documents wrapping projctl step complete with retry"
if grep -qi "wrap.*step complete.*retry\|retry.*wrap.*step complete" "$SKILL_FULL_FILE"; then
  pass "Documents wrapping projctl step complete with retry"
else
  fail "Missing documentation for wrapping projctl step complete with retry"
fi

echo "Test: SKILL-full.md documents spawn confirmation timeout/retry"
if grep -qi "timeout.*spawn.*confirmation.*retry\|spawn.*confirmation.*timeout.*retry\|spawn.*confirmation.*wait.*retry" "$SKILL_FULL_FILE"; then
  pass "Documents spawn confirmation timeout/retry"
else
  fail "Missing spawn confirmation timeout/retry documentation"
fi

echo "Test: SKILL-full.md documents backoff pattern"
if grep -qi "backoff\|exponential.*backoff" "$SKILL_FULL_FILE"; then
  pass "Documents backoff pattern"
else
  fail "Missing backoff pattern documentation"
fi

echo "Test: SKILL-full.md documents specific backoff delays (1s, 2s, 4s)"
if grep -qi "1s.*2s.*4s\|delays.*1.*2.*4" "$SKILL_FULL_FILE"; then
  pass "Documents backoff delays: 1s, 2s, 4s"
else
  fail "Missing specific backoff delay pattern (1s, 2s, 4s)"
fi

echo "Test: SKILL-full.md documents escalation after 3 failed attempts"
if grep -qi "after.*3.*attempt\|max.*3.*attempt\|3.*fail.*attempt" "$SKILL_FULL_FILE"; then
  pass "Documents escalation after 3 failed attempts"
else
  fail "Missing escalation after 3 attempts documentation"
fi

echo "Test: SKILL-full.md documents orchestrator sends error message to team lead"
if grep -qi "send.*error.*team.lead\|escalate.*error.*team.lead\|orchestrator.*send.*message.*error" "$SKILL_FULL_FILE"; then
  pass "Documents orchestrator sends error message to team lead"
else
  fail "Missing orchestrator error message to team lead documentation"
fi

echo "Test: Error message includes action and phase fields"
if grep -qi "error.*message.*action.*phase\|error.*field.*action.*phase\|message.*includes.*action.*phase" "$SKILL_FULL_FILE"; then
  pass "Error message includes action and phase fields"
else
  fail "Missing documentation for error message including action and phase fields"
fi

echo "Test: Error message includes error output field"
if grep -qi "error.*message.*error.*output\|error.*field.*error.*output\|message.*includes.*error.*output" "$SKILL_FULL_FILE"; then
  pass "Error message includes error output field"
else
  fail "Missing documentation for error message including error output field"
fi

echo "Test: Error message includes retry history field"
if grep -qi "error.*message.*retry.*history\|error.*field.*retry.*history\|message.*includes.*retry.*history\|message.*includes.*attempt.*history" "$SKILL_FULL_FILE"; then
  pass "Error message includes retry history field"
else
  fail "Missing documentation for error message including retry history field"
fi

echo "Test: Team lead escalates errors to user"
if grep -qi "team.lead.*escalate.*user\|escalate.*error.*user" "$SKILL_FILE" "$SKILL_FULL_FILE"; then
  pass "Team lead escalates errors to user"
else
  fail "Missing team lead escalation to user documentation"
fi

echo "Test: Uses AskUserQuestion for error escalation"
if grep -qi "AskUserQuestion.*error\|error.*AskUserQuestion\|escalate.*AskUserQuestion" "$SKILL_FILE" "$SKILL_FULL_FILE"; then
  pass "Uses AskUserQuestion for error escalation"
else
  fail "Missing AskUserQuestion usage for error escalation"
fi

echo "Test: Orchestrator logs retry attempts"
if grep -qi "log.*retry.*attempt\|logging.*retry\|orchestrator.*log.*retry" "$SKILL_FULL_FILE"; then
  pass "Orchestrator logs retry attempts"
else
  fail "Missing orchestrator retry logging documentation"
fi

# --- ISSUE-156: Task owner and status on spawn/complete ---

echo ""
echo "--- ISSUE-156: Task Owner and Status ---"

echo "Test: Control loop includes TaskUpdate on spawn"
if grep -qi "TaskUpdate.*in_progress\|TaskUpdate.*owner" "$SKILL_FILE"; then
  pass "Control loop includes TaskUpdate for owner/status on spawn"
else
  fail "Missing TaskUpdate for owner/status on spawn in control loop"
fi

echo "Test: Control loop includes TaskUpdate on complete"
if grep -qi "TaskUpdate.*completed\|status.*completed" "$SKILL_FILE"; then
  pass "Control loop includes TaskUpdate for completed status"
else
  fail "Missing TaskUpdate for completed status in control loop"
fi

echo "Test: Control loop clears owner on failed status"
if grep -B 5 -A 5 "\-\-status failed" "$SKILL_FILE" | grep -qi 'owner.*""\|owner:.*""'; then
  pass "Control loop clears owner on failed status"
else
  fail "Missing owner clear on failed status"
fi

# --- ISSUE-157: Top-level orchestration task ---

echo ""
echo "--- ISSUE-157: Top-level Orchestration Task ---"

echo "Test: Startup creates top-level TaskCreate"
if grep -qi "TaskCreate.*ISSUE\|top-level.*task\|orchestration.*task" "$SKILL_FILE"; then
  pass "Startup creates top-level orchestration task"
else
  fail "Missing top-level TaskCreate in Startup section"
fi

echo "Test: Startup prefixes phase tasks with issue ID"
if grep -qi "ISSUE-NNN.*Create project plan\|prefix.*phase.*ISSUE" "$SKILL_FILE"; then
  pass "Startup prefixes phase tasks with issue ID"
else
  fail "Missing issue ID prefix for phase tasks"
fi

echo "Test: End-of-command marks top-level task completed"
END_OF_COMMAND_SECTION=$(sed -n '/^## End-of-Command Sequence/,/^##[^#]/p' "$SKILL_FILE")
if echo "$END_OF_COMMAND_SECTION" | grep -qi "TaskUpdate.*completed\|mark.*top-level.*completed"; then
  pass "End-of-command marks top-level task completed"
else
  fail "Missing TaskUpdate for top-level task in end-of-command"
fi

# --- ISSUE-164/165: Orchestrator owns full spawn lifecycle ---

echo ""
echo "--- ISSUE-164/165: Orchestrator Spawn Lifecycle ---"

echo "Test: ISSUE-164: Control loop requires WAIT for teammate message before step complete"
if grep -qi "WAIT.*teammate.*message\|WAIT.*completion.*message\|WAIT.*QA.*teammate" "$SKILL_FILE"; then
  pass "Control loop requires WAIT for teammate message"
else
  fail "Missing WAIT for teammate message in control loop"
fi

echo "Test: ISSUE-165: spawn-producer step complete includes --producer-transcript"
if grep -q "\-\-producer-transcript" "$SKILL_FILE"; then
  pass "spawn-producer step complete includes --producer-transcript flag"
else
  fail "Missing --producer-transcript flag in spawn-producer step complete"
fi

echo "Test: Action handlers don't call step complete (orchestrator does)"
# Extract the Team Lead Handlers section up to the next #### heading
# Exclude lines that describe what NOT to do (contains "NOT" before "step complete")
HANDLER_SECTION=$(awk '/#### spawn-producer.*spawn-qa.*Team Lead/{found=1;next} /^#### /{found=0} found' "$SKILL_FILE")
if echo "$HANDLER_SECTION" | grep -v "NOT.*step complete\|does not.*step complete" | grep -q "projctl step complete --"; then
  fail "Action handlers still call projctl step complete (orchestrator should own this)"
else
  pass "Action handlers don't call step complete"
fi

echo "Test: Team lead is described as spawn service"
if grep -qi "spawn service\|spawn.*service" "$SKILL_FILE"; then
  pass "Team lead described as spawn service"
else
  fail "Missing spawn service description for team lead"
fi

echo "Test: Teammate messages orchestrator directly"
if grep -qi "teammate.*messages.*orchestrator.*directly\|teammate messages orchestrator" "$SKILL_FILE"; then
  pass "Teammate messages orchestrator directly"
else
  fail "Missing documentation that teammate messages orchestrator directly"
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

# --- ISSUE-152 TASK-24: Session-End Memory Capture ---

echo ""
echo "--- ISSUE-152 TASK-24: Session-End Memory Capture ---"

echo "Test: AC-1: memory session-end in end-of-command block"
# Extract the "End-of-Command Sequence" section
END_OF_COMMAND_SECTION=$(sed -n '/^## End-of-Command Sequence/,/^##[^#]/p' "$SKILL_FILE")
if echo "$END_OF_COMMAND_SECTION" | grep -q 'projctl memory session-end -p "<issue-id>"'; then
  pass "memory session-end in End-of-Command Sequence block"
else
  fail "Expected 'projctl memory session-end -p \"<issue-id>\"' in End-of-Command Sequence section"
fi

echo "Test: AC-2: session-end runs BEFORE integrate"
SESSION_END_LINE=$(echo "$END_OF_COMMAND_SECTION" | grep -n "projctl memory session-end" | cut -d: -f1 | head -1)
INTEGRATE_LINE=$(echo "$END_OF_COMMAND_SECTION" | grep -n "projctl integrate features" | cut -d: -f1 | head -1)
if [[ -z "$SESSION_END_LINE" ]]; then
  fail "projctl memory session-end not found in End-of-Command Sequence"
elif [[ -z "$INTEGRATE_LINE" ]]; then
  fail "projctl integrate features not found in End-of-Command Sequence"
elif [[ $SESSION_END_LINE -lt $INTEGRATE_LINE ]]; then
  pass "session-end (line $SESSION_END_LINE) runs before integrate (line $INTEGRATE_LINE)"
else
  fail "session-end (line $SESSION_END_LINE) should appear before integrate (line $INTEGRATE_LINE)"
fi

echo "Test: AC-3: session-end receives issue ID parameter"
if grep -q 'projctl memory session-end.*-p.*"<[^>]*>"' "$SKILL_FILE"; then
  pass "session-end command includes -p flag with issue ID placeholder"
else
  fail "Expected session-end command to include -p flag with issue ID placeholder"
fi

echo "Test: AC-4: exactly one memory session-end occurrence"
SESSION_END_COUNT=$(grep -c "memory session-end" "$SKILL_FILE" || true)
if [[ $SESSION_END_COUNT -eq 1 ]]; then
  pass "Exactly 1 occurrence of 'memory session-end' found"
else
  fail "Expected exactly 1 occurrence of 'memory session-end', found $SESSION_END_COUNT"
fi

# --- Summary ---

echo ""
if [[ $FAILURES -eq 0 ]]; then
  echo "=== All tests passed ==="
else
  echo "=== $FAILURES test(s) FAILED ==="
  exit 1
fi
