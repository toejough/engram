#!/usr/bin/env bash
# Behavioral tests for skills/engram-tmux-lead/SKILL.md
# Run these to verify expected skill content.
# Usage: bash behavioral_test.sh
# After #483 changes, update EXPECTED_* constants at top to match new target behavior.

SKILL="$(dirname "$0")/../SKILL.md"
PASS=0
FAIL=0
FAILURES=()

pass() { echo "PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $1"; FAIL=$((FAIL + 1)); FAILURES+=("$1"); }

assert_contains() {
  local desc="$1"
  local pattern="$2"
  if grep -qF "$pattern" "$SKILL"; then
    pass "$desc"
  else
    fail "$desc [pattern not found: '$pattern']"
  fi
}

assert_not_contains() {
  local desc="$1"
  local pattern="$2"
  if ! grep -qF "$pattern" "$SKILL"; then
    pass "$desc"
  else
    fail "$desc [pattern should NOT be present: '$pattern']"
  fi
}

assert_count_gte() {
  local desc="$1"
  local pattern="$2"
  local min="$3"
  local count
  count=$(grep -cF "$pattern" "$SKILL" || true)
  if [ "$count" -ge "$min" ]; then
    pass "$desc (count=$count >= min=$min)"
  else
    fail "$desc (count=$count < min=$min, pattern='$pattern')"
  fi
}

assert_count_eq() {
  local desc="$1"
  local pattern="$2"
  local expected="$3"
  local count
  count=$(grep -cF "$pattern" "$SKILL" || true)
  if [ "$count" -eq "$expected" ]; then
    pass "$desc (count=$count)"
  else
    fail "$desc (count=$count != expected=$expected, pattern='$pattern')"
  fi
}

echo "=== engram-tmux-lead SKILL.md behavioral tests ==="
echo ""

# ---------------------------------------------------------------------------
# Group 1: State machine — transitions and states
# These tests describe the EXPECTED state after #483 is fully implemented.
# Before implementation, tests in this group will FAIL (RED).
# After implementation, they must PASS (GREEN).
# ---------------------------------------------------------------------------

echo "--- Group 1: State machine (post-#483 expected) ---"

# 12 transitions expected (arrows in state diagram)
# Count "──>" patterns in the state diagram block
TRANSITION_COUNT=$(grep -c '──>' "$SKILL" || true)
if [ "$TRANSITION_COUNT" -ge 12 ]; then
  pass "State diagram has >= 12 transitions (found $TRANSITION_COUNT)"
else
  fail "State diagram needs >= 12 transitions (found $TRANSITION_COUNT, need 12 for #483)"
fi

# PENDING-RELEASE state must exist in state table
assert_contains "PENDING-RELEASE state in state table" "**PENDING-RELEASE**"

# PENDING-RELEASE → DONE transition present
assert_contains "PENDING-RELEASE → DONE transition" "PENDING-RELEASE ──(last incoming hold dissolved)──> DONE"

# PENDING-RELEASE → PENDING-RELEASE no-op on repeated done
assert_contains "PENDING-RELEASE no-op self-transition" "PENDING-RELEASE ──(agent posts done again, HAS incoming holds)──> PENDING-RELEASE"

# PENDING-RELEASE → SILENT transition
assert_contains "PENDING-RELEASE → SILENT transition" "PENDING-RELEASE ──(no message for silence_threshold)──> SILENT"

# ACTIVE → DONE only when no holds
assert_contains "ACTIVE → DONE requires no holds" "ACTIVE ──(agent posts done, NO incoming holds)──> DONE"

# ACTIVE → PENDING-RELEASE when has holds
assert_contains "ACTIVE → PENDING-RELEASE when has holds" "ACTIVE ──(agent posts done, HAS incoming holds)──> PENDING-RELEASE"

# PENDING-RELEASE-specific nudge text
assert_contains "PENDING-RELEASE nudge text" "You are held in PENDING-RELEASE and may receive further instructions"

echo ""

# ---------------------------------------------------------------------------
# Group 2: Section 3.5 — Hold-based agent retention (must exist post-#483)
# ---------------------------------------------------------------------------

echo "--- Group 2: Section 3.5 Hold-Based Agent Retention ---"

assert_contains "Section 3.5 header exists" "### 3.5 Hold-Based Agent Retention"
assert_contains "Hold primitive definition" "A **hold** is a directed keep-alive relationship"
assert_contains "Hold definition struct" "holder:  string"
assert_contains "Hold release conditions table" "| Condition | Syntax | Fires When |"
assert_contains "lead_release condition" "lead_release"
assert_contains "done(agent) release condition" "done(agent)"
assert_contains "first_intent(agent) release condition" "first_intent(agent)"
assert_contains "Hold registry section" "Hold Registry (In-Context State)"
assert_contains "Hold detection background task pattern" "Hold Detection (Background Tasks)"
assert_contains "Persistent hold watcher (while true loop)" "while true; do"
assert_contains "When a hold fires steps" "When a hold fires"
assert_contains "Never create holds retroactively" "NEVER create holds retroactively"

echo ""

# ---------------------------------------------------------------------------
# Group 3: Documented patterns (in Section 3.5)
# ---------------------------------------------------------------------------

echo "--- Group 3: Documented patterns ---"

assert_contains "Pattern: Pair (Review)" "Pattern: Pair (Review)"
assert_contains "Pattern: Handoff" "Pattern: Handoff"
assert_contains "Pattern: Fan-In (Research Synthesis)" "Pattern: Fan-In"
assert_contains "Pattern: Merge Queue" "Pattern: Merge Queue"
assert_contains "Pattern: Barrier (Co-Design)" "Pattern: Barrier"
assert_contains "Pattern: Expert Consultation" "Pattern: Expert Consultation"

echo ""

# ---------------------------------------------------------------------------
# Group 4: Section 4.2 — Plan-Execute-Review pipeline with holds
# ---------------------------------------------------------------------------

echo "--- Group 4: Plan-Execute-Review pipeline holds ---"

assert_contains "Phase 1b mandatory plan review section" "**Phase 1b: PLAN REVIEW (mandatory)**"
assert_contains "Plan-review hold created" "plan-review hold"
assert_contains "Plan-review hold uses lead_release (Fix B)" "release: lead_release(\"plan-review-N\")"
assert_contains "Plan-handoff hold created in Phase 2" "plan-handoff hold"
assert_contains "Phase 2 calls lead_release for plan-review hold" "lead_release(\"plan-review-N\")"
assert_contains "Atomic handoff: plan-handoff created before plan-review released" "atomic handoff"
assert_contains "Impl-review hold created in Phase 3" "impl-review hold"
assert_contains "Executor stays in PENDING-RELEASE for review" "Executor enters PENDING-RELEASE"

echo ""

# ---------------------------------------------------------------------------
# Group 5: Section 4.3 — Merge queue (replaces old merge strategy)
# ---------------------------------------------------------------------------

echo "--- Group 5: Merge queue in Section 4.3 ---"

assert_contains "Lead-coordinated merge queue section" "lead-coordinated merge queue"
assert_contains "Merge queue uses merge-process holds" "merge-process"
assert_contains "Merge queue sequential procedure" "Sequential merge procedure"
assert_not_contains "Old merge strategy not present" "Merge each worktree branch back one at a time"

echo ""

# ---------------------------------------------------------------------------
# Group 6: Sections 4.5 and 4.6 (new routing patterns)
# ---------------------------------------------------------------------------

echo "--- Group 6: Research Synthesis (4.5) and Co-Design (4.6) ---"

assert_contains "Section 4.5 Research Synthesis exists" "### 4.5 Research Synthesis"
assert_contains "Section 4.6 Co-Design exists" "### 4.6 Co-Design"
assert_contains "Fan-in holds referenced in 4.5" "fan-in holds"
assert_contains "Barrier holds referenced in 4.6" "barrier holds"
assert_contains "Routing table has Research Synthesis row" "Research Synthesis"
assert_contains "Routing table has Co-Design row" "Co-Design"

echo ""

# ---------------------------------------------------------------------------
# Group 7: Role templates (Section 2.2) — hold awareness
# ---------------------------------------------------------------------------

echo "--- Group 7: Role templates hold-awareness ---"

# ALL role templates must include the "keep watching" line
KEEP_WATCHING_COUNT=$(grep -c "continue watching chat" "$SKILL" || true)
# Expecting the line in executor, planner, reviewer, researcher, synthesizer, co-designer, plan-reviewer = 7+
if [ "$KEEP_WATCHING_COUNT" -ge 7 ]; then
  pass "Keep-watching line present in >= 7 role templates (found $KEEP_WATCHING_COUNT)"
else
  fail "Keep-watching line needs >= 7 occurrences, found $KEEP_WATCHING_COUNT"
fi

assert_contains "Synthesizer template exists" "**Synthesizer:**"
assert_contains "Co-Designer template exists" "**Co-Designer:**"
assert_contains "Plan Reviewer template exists" "**Plan Reviewer:**"
assert_contains "Executor template has keep-watching" "After posting done, continue watching chat for further instructions. You may receive follow-up questions or requests while held in PENDING-RELEASE."
assert_contains "Planner template has keep-watching" "continue watching chat — a reviewer and/or executor may have questions"
assert_contains "Reviewer template has keep-watching" "After posting done, continue watching chat for further instructions."
assert_contains "Researcher template has keep-watching" "continue watching chat — a synthesizer may have follow-up"

echo ""

# ---------------------------------------------------------------------------
# Group 8: Context retention (Section 7.1) — hold registry
# ---------------------------------------------------------------------------

echo "--- Group 8: Context retention includes hold registry ---"

assert_contains "Hold registry in context retention table" "Hold registry (id, holder, target, release, tag, task_id, cursor)"

echo ""

# ---------------------------------------------------------------------------
# Group 9: Background task hygiene (Section 6.4) — hold rules
# ---------------------------------------------------------------------------

echo "--- Group 9: Background task hygiene hold rules ---"

assert_contains "Rule 7 one task per hold" "One persistent background task per hold"
assert_contains "Rule 8 drain on lead_release" "Drain on lead_release"
assert_contains "Rule 9 hold tasks concurrent" "Hold detection tasks do not replace each other"
assert_contains "Rule 10 hold watchers replace standard wait" "Hold watchers replace standard agent wait tasks"

echo ""

# ---------------------------------------------------------------------------
# Group 10: Chat-first diagnostics (#541)
# ---------------------------------------------------------------------------

echo "--- Group 10: Chat-first diagnostics ---"

assert_contains "Chat-First Diagnostics hard rule exists" "HARD RULE: The chat file is the primary diagnostic source"
assert_contains "Preamble capture-pane is last resort" "last resort"
assert_contains "Section 1.5 timeout reads chat first" "Read chat from cursor"
assert_contains "Section 2.1 TIMEOUT reads chat first" "Read chat from cursor"

echo ""

# ---------------------------------------------------------------------------
# Group 10: Post-drain sweep reads full message text (#540)
# ---------------------------------------------------------------------------

echo "--- Group 10: Post-drain sweep reads full message text ---"

assert_contains "Lead reads full text in post-drain sweep" "Read the full"
assert_contains "Conversation messages handled in sweep" "natural-prose signals"
assert_contains "Natural-prose coordination signals mentioned" "natural-prose coordination signals"
assert_contains "Wait handled immediately in sweep" "engage immediately"

echo ""

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

echo "=== Results ==="
echo "PASS: $PASS"
echo "FAIL: $FAIL"
if [ "${#FAILURES[@]}" -gt 0 ]; then
  echo ""
  echo "Failed tests:"
  for f in "${FAILURES[@]}"; do
    echo "  - $f"
  done
fi

if [ "$FAIL" -gt 0 ]; then
  exit 1
else
  echo "All tests passed."
  exit 0
fi
