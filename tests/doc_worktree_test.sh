#!/usr/bin/env bash
# Tests for ISSUE-50: Document worktree workflow for parallel execution
# Validates worktree documentation in orchestration-system.md and parallel-looper SKILL.md

set -uo pipefail
# Note: -e omitted so tests can continue after failures

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

ORCH_DOC="$REPO_ROOT/docs/orchestration-system.md"
PARALLEL_SKILL="$REPO_ROOT/skills/parallel-looper/SKILL.md"

PASS=0
FAIL=0

test_pass() {
    echo "✓ $1"
    ((PASS++))
}

test_fail() {
    echo "✗ $1"
    ((FAIL++))
}

# === orchestration-system.md tests ===

# Test: Commands table exists in Section 6.5
test_orch_commands_table() {
    if grep -q "| Command | Purpose |" "$ORCH_DOC"; then
        test_pass "orchestration-system.md has commands table"
    else
        test_fail "orchestration-system.md missing commands table"
    fi
}

# Test: Uses correct --taskid flag (not --task)
test_orch_correct_flag() {
    if grep -q "projctl worktree create --taskid" "$ORCH_DOC"; then
        test_pass "orchestration-system.md uses --taskid flag"
    else
        test_fail "orchestration-system.md should use --taskid not --task"
    fi
}

# Test: Documents all worktree commands
test_orch_all_commands() {
    local missing=()
    grep -q "worktree create" "$ORCH_DOC" || missing+=("create")
    grep -q "worktree merge" "$ORCH_DOC" || missing+=("merge")
    grep -q "worktree cleanup" "$ORCH_DOC" || missing+=("cleanup")
    grep -q "worktree cleanup-all" "$ORCH_DOC" || missing+=("cleanup-all")
    grep -q "worktree list" "$ORCH_DOC" || missing+=("list")

    if [ ${#missing[@]} -eq 0 ]; then
        test_pass "orchestration-system.md documents all worktree commands"
    else
        test_fail "orchestration-system.md missing commands: ${missing[*]}"
    fi
}

# === parallel-looper SKILL.md tests ===

# Test: Has worktree section
test_parallel_worktree_section() {
    if grep -q "## Git Worktrees" "$PARALLEL_SKILL" || grep -q "## Worktree" "$PARALLEL_SKILL"; then
        test_pass "parallel-looper SKILL.md has worktree section"
    else
        test_fail "parallel-looper SKILL.md missing worktree section"
    fi
}

# Test: Documents worktree lifecycle
test_parallel_lifecycle() {
    if grep -q "Worktree Lifecycle" "$PARALLEL_SKILL" || grep -q "CREATE.*WORK.*MERGE" "$PARALLEL_SKILL"; then
        test_pass "parallel-looper SKILL.md documents worktree lifecycle"
    else
        test_fail "parallel-looper SKILL.md missing worktree lifecycle"
    fi
}

# Test: Documents merge-on-complete pattern
test_parallel_merge_pattern() {
    if grep -q "Merge-on-Complete" "$PARALLEL_SKILL" || grep -q "merge-on-complete" "$PARALLEL_SKILL"; then
        test_pass "parallel-looper SKILL.md documents merge-on-complete pattern"
    else
        test_fail "parallel-looper SKILL.md missing merge-on-complete pattern"
    fi
}

# Test: Documents conflict handling
test_parallel_conflicts() {
    if grep -q "Conflict" "$PARALLEL_SKILL" && grep -q "Rebase conflict" "$PARALLEL_SKILL"; then
        test_pass "parallel-looper SKILL.md documents conflict handling"
    else
        test_fail "parallel-looper SKILL.md missing conflict handling"
    fi
}

# Test: Has commands reference
test_parallel_commands() {
    if grep -q "projctl worktree" "$PARALLEL_SKILL"; then
        test_pass "parallel-looper SKILL.md has projctl worktree commands"
    else
        test_fail "parallel-looper SKILL.md missing projctl worktree commands"
    fi
}

# Run all tests
echo "=== Testing ISSUE-50: Worktree Documentation ==="
echo ""
echo "--- orchestration-system.md ---"
test_orch_commands_table
test_orch_correct_flag
test_orch_all_commands

echo ""
echo "--- parallel-looper SKILL.md ---"
test_parallel_worktree_section
test_parallel_lifecycle
test_parallel_merge_pattern
test_parallel_conflicts
test_parallel_commands

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [ $FAIL -gt 0 ]; then
    exit 1
fi
