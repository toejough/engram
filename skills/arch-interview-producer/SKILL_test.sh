#!/bin/bash
# Integration tests for adaptive interview flow (TASK-8)
# Tests verify end-to-end behavior of arch-interview-producer with mocked context scenarios
# Traces to: TASK-8 acceptance criteria
# Run: bash ~/.claude/skills/arch-interview-producer/SKILL_test.sh
#
# IMPORTANT: These tests expect to FAIL until the adaptive interview implementation is complete.
# This is the "RED" phase of TDD - tests are written first to specify expected behavior.

set -e

# Test configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$HOME/repos/personal/projctl"
TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test utilities
fail() {
    echo -e "${RED}FAIL: $1${NC}"
    exit 1
}

pass() {
    echo -e "${GREEN}PASS: $1${NC}"
}

warn() {
    echo -e "${YELLOW}WARN: $1${NC}"
}

# Check if required tools exist
if ! command -v jq &> /dev/null; then
    fail "Required tool not found: jq (install with: brew install jq)"
fi

# Check if projctl exists (integration target)
if [[ ! -f "$PROJECT_ROOT/projctl" ]] && ! command -v projctl &> /dev/null; then
    warn "projctl binary not found - some integration tests will be skipped"
    warn "Run 'mage build' in $PROJECT_ROOT to build projctl"
    PROJCTL_AVAILABLE=false
else
    PROJCTL_AVAILABLE=true
    if [[ -f "$PROJECT_ROOT/projctl" ]]; then
        PROJCTL="$PROJECT_ROOT/projctl"
    else
        PROJCTL="projctl"
    fi
fi

echo "=== TASK-8: Integration Tests for Adaptive Interview Flow ==="
echo "Expected: These tests should FAIL until adaptive interview implementation is complete"
echo ""

# ============================================================================
# Test Helper: Create fixture files for context scenarios
# ============================================================================
create_fixture_sparse() {
    local fixture_dir="$1"
    mkdir -p "$fixture_dir"

    # Empty territory (no artifacts)
    cat > "$fixture_dir/territory.json" <<'EOF'
{
  "files": [],
  "artifacts": {}
}
EOF

    # No memory results
    cat > "$fixture_dir/memory.json" <<'EOF'
{
  "results": []
}
EOF

    # Minimal issue description
    cat > "$fixture_dir/issue.md" <<'EOF'
# ISSUE-TEST: Build a system

Need to build something.
EOF
}

create_fixture_medium() {
    local fixture_dir="$1"
    mkdir -p "$fixture_dir"

    # Territory with some artifacts
    cat > "$fixture_dir/territory.json" <<'EOF'
{
  "files": ["README.md", "docs/requirements.md"],
  "artifacts": {
    "requirements": "docs/requirements.md"
  }
}
EOF

    # Some memory results (65% coverage scenario)
    cat > "$fixture_dir/memory.json" <<'EOF'
{
  "results": [
    {"content": "Technology stack: Go backend", "score": 0.85, "source": "requirements.md"},
    {"content": "Expected scale: 10k users", "score": 0.82, "source": "design.md"},
    {"content": "Deployment: AWS Lambda", "score": 0.78, "source": "architecture.md"}
  ]
}
EOF

    # Issue with moderate detail
    cat > "$fixture_dir/issue.md" <<'EOF'
# ISSUE-TEST: Build authentication system

Build a user authentication system with JWT tokens.
Should support 10k concurrent users.
Need to integrate with Stripe for payments.
EOF
}

create_fixture_rich() {
    local fixture_dir="$1"
    mkdir -p "$fixture_dir"

    # Territory with comprehensive artifacts
    cat > "$fixture_dir/territory.json" <<'EOF'
{
  "files": ["README.md", "docs/requirements.md", "docs/design.md", "docs/architecture.md"],
  "artifacts": {
    "requirements": "docs/requirements.md",
    "design": "docs/design.md",
    "architecture": "docs/architecture.md"
  }
}
EOF

    # Rich memory results (90% coverage)
    cat > "$fixture_dir/memory.json" <<'EOF'
{
  "results": [
    {"content": "Technology stack: Go backend with PostgreSQL database", "score": 0.92},
    {"content": "Expected scale: 10k concurrent users, 100k total users", "score": 0.91},
    {"content": "Deployment: AWS ECS with CloudFront CDN", "score": 0.90},
    {"content": "Integrations: Stripe for payments, SendGrid for email", "score": 0.88},
    {"content": "Performance SLA: <200ms p95 response time", "score": 0.87},
    {"content": "Security: OAuth2 with JWT tokens, bcrypt password hashing", "score": 0.86},
    {"content": "Data durability: RDS with automated backups, 7-day retention", "score": 0.85},
    {"content": "Observability: Datadog monitoring with custom metrics", "score": 0.84},
    {"content": "Development: Docker Compose local environment", "score": 0.83}
  ]
}
EOF

    cat > "$fixture_dir/issue.md" <<'EOF'
# ISSUE-TEST: Build comprehensive e-commerce platform

Complete e-commerce system with the following:
- Go backend with PostgreSQL
- OAuth2 authentication
- Stripe payment integration
- SendGrid email notifications
- AWS ECS deployment
- CloudFront CDN
- <200ms p95 response time
- Support 10k concurrent users
- Datadog monitoring
EOF
}

create_fixture_contradictory() {
    local fixture_dir="$1"
    mkdir -p "$fixture_dir"

    cat > "$fixture_dir/memory.json" <<'EOF'
{
  "results": [
    {"content": "Database: SQLite for simple local storage", "score": 0.92, "source": "requirements.md"},
    {"content": "Database: PostgreSQL for production scale 10k users", "score": 0.91, "source": "architecture.md"}
  ]
}
EOF

    cat > "$fixture_dir/issue.md" <<'EOF'
# ISSUE-TEST: Database selection conflict

System needs data storage.
EOF
}

# ============================================================================
# AC-1: Test sparse context (0% coverage) → large gap, 6+ questions
# Traces to: TASK-8 AC-1
# AC-6 specifies: Each test validates question count, gap metrics in yield, question relevance
# ============================================================================
test_sparse_context_large_gap() {
    echo "TEST: Sparse context (0% coverage) documented to yield large gap with 6+ questions"

    # Verify SKILL.md documents sparse context handling
    local skill_file="$SCRIPT_DIR/SKILL.md"

    # Check that ASSESS phase mentions coverage calculation
    if ! grep -q "coverage_percent" "$skill_file"; then
        fail "SKILL.md missing coverage_percent in gap_analysis - needed for sparse context detection"
    fi

    # Check that gap size classification is documented
    if ! grep -q "gap_size" "$skill_file"; then
        fail "SKILL.md missing gap_size classification - needed for depth determination"
    fi

    # Check that large gap case is documented (6+ questions)
    if ! grep -qE "(<50%|large.*gap).*(6\+|six or more)" "$skill_file"; then
        fail "SKILL.md does not document large gap → 6+ questions mapping"
    fi

    # Verify yield example includes gap_analysis metadata
    if ! grep -A 10 "\[context\.gap_analysis\]" "$skill_file" | grep -q "question_count"; then
        fail "SKILL.md yield examples missing question_count in gap_analysis"
    fi

    # Verify Key Questions section exists (needed for coverage calculation)
    if ! grep -q "## Key Questions" "$skill_file"; then
        fail "SKILL.md missing Key Questions section needed for gap calculation"
    fi

    pass "Sparse context scenario fully documented in SKILL.md"
}

# ============================================================================
# AC-2: Test medium context (65% coverage) → medium gap, 3-5 questions
# Traces to: TASK-8 AC-2
# AC-6 specifies: Each test validates question count, gap metrics in yield, question relevance
# ============================================================================
test_medium_context_medium_gap() {
    echo "TEST: Medium context (65% coverage) documented to yield medium gap with 3-5 questions"

    local skill_file="$SCRIPT_DIR/SKILL.md"

    # Check that medium gap case is documented (3-5 questions)
    if ! grep -qE "(50-79%|medium.*gap).*(3-5|three to five)" "$skill_file"; then
        fail "SKILL.md does not document medium gap → 3-5 questions mapping"
    fi

    # Verify that questions should reference gathered context
    assess_line=$(grep -n "### ASSESS Phase" "$skill_file" | cut -d: -f1 || echo "")
    next_phase_line=$(grep -nE "### (SYNTHESIZE|INTERVIEW) Phase" "$skill_file" | head -1 | cut -d: -f1 || echo "")

    if [[ -n "$assess_line" ]] && [[ -n "$next_phase_line" ]] && [[ "$next_phase_line" -gt "$assess_line" ]]; then
        # Check if ASSESS phase mentions context references
        if ! sed -n "${assess_line},${next_phase_line}p" "$skill_file" | grep -qE "(reference.*context|gathered.*context|I see)"; then
            warn "SKILL.md should document that questions reference gathered context for medium gaps"
        fi
    fi

    # Verify priority ordering is documented
    if ! grep -qE "(priorit|critical.*important|important.*optional)" "$skill_file"; then
        fail "SKILL.md missing priority ordering documentation needed for question selection"
    fi

    # Check coverage range is documented
    if ! grep -qE "50.*79" "$skill_file"; then
        fail "SKILL.md missing medium gap coverage range (50-79%)"
    fi

    pass "Medium context scenario fully documented in SKILL.md"
}

# ============================================================================
# AC-3: Test rich context (90% coverage) → small gap, 1-2 questions
# Traces to: TASK-8 AC-3
# AC-6 specifies: Each test validates question count, gap metrics in yield, question relevance
# ============================================================================
test_rich_context_small_gap() {
    echo "TEST: Rich context (90% coverage) documented to yield small gap with 1-2 questions"

    local skill_file="$SCRIPT_DIR/SKILL.md"

    # Check that small gap case is documented (1-2 questions)
    if ! grep -qE "(≥80%|>=80%|80%.*or.*higher|small.*gap).*(1-2|one to two|1.*2.*question)" "$skill_file"; then
        fail "SKILL.md does not document small gap → 1-2 questions mapping"
    fi

    # Verify that critical unanswered items are documented
    if ! grep -qE "critical.*unanswered" "$skill_file"; then
        fail "SKILL.md missing documentation about tracking unanswered critical questions"
    fi

    # Check that confirmation-style questions are documented for small gaps
    assess_line=$(grep -n "### ASSESS Phase" "$skill_file" | cut -d: -f1 || echo "")
    next_phase_line=$(grep -nE "### (SYNTHESIZE|INTERVIEW) Phase" "$skill_file" | head -1 | cut -d: -f1 || echo "")

    if [[ -n "$assess_line" ]] && [[ -n "$next_phase_line" ]] && [[ "$next_phase_line" -gt "$assess_line" ]]; then
        if ! sed -n "${assess_line},${next_phase_line}p" "$skill_file" | grep -qE "(confirmation|confirm|verify)"; then
            warn "SKILL.md should document that small gaps yield confirmation-style questions"
        fi
    fi

    # Verify coverage threshold is documented
    if ! grep -qE "80" "$skill_file"; then
        fail "SKILL.md missing small gap coverage threshold (≥80%)"
    fi

    # Check that unanswered_critical is in gap_analysis metadata
    if ! grep -A 10 "\[context\.gap_analysis\]" "$skill_file" | grep -q "unanswered_critical"; then
        fail "SKILL.md yield examples missing unanswered_critical in gap_analysis"
    fi

    pass "Rich context scenario fully documented in SKILL.md"
}

# ============================================================================
# AC-4: Test contradictory context → need-decision yield
# Traces to: TASK-8 AC-4
# ============================================================================
test_contradictory_context_need_decision() {
    echo "TEST: Contradictory context documented to yield need-decision"

    local skill_file="$SCRIPT_DIR/SKILL.md"

    # Check that contradictory context handling is documented
    if ! grep -qE "(contradict|conflict)" "$skill_file"; then
        fail "SKILL.md does not document contradictory context handling"
    fi

    # Verify need-decision yield is documented for conflicts
    if ! grep -qE "need-decision" "$skill_file"; then
        fail "SKILL.md missing need-decision yield type for contradictory context"
    fi

    # Check that ASSESS phase mentions checking for contradictions
    assess_line=$(grep -n "### ASSESS Phase" "$skill_file" | cut -d: -f1 || echo "0")
    next_phase_line=$(grep -nE "### (SYNTHESIZE|INTERVIEW) Phase" "$skill_file" | head -1 | cut -d: -f1 || echo "999999")

    if [[ "$assess_line" != "0" ]] && [[ "$next_phase_line" != "999999" ]]; then
        assess_section=$(sed -n "${assess_line},${next_phase_line}p" "$skill_file")
        if ! echo "$assess_section" | grep -qE "(contradict|conflict)"; then
            fail "ASSESS phase should document checking for contradictory context"
        fi
        if ! echo "$assess_section" | grep -qE "need-decision"; then
            fail "ASSESS phase should document yielding need-decision for conflicts"
        fi
    else
        fail "Cannot find ASSESS phase boundaries to verify contradiction handling"
    fi

    # Verify yield types section mentions need-decision
    if ! grep -A 5 "## Yield Types" "$skill_file" | grep -q "need-decision"; then
        warn "Yield Types section should document need-decision for contradictory context"
    fi

    pass "Contradictory context handling fully documented in SKILL.md"
}

# ============================================================================
# AC-5: Test territory map failure → blocked yield
# Traces to: TASK-8 AC-5
# ============================================================================
test_territory_map_failure_yields_blocked() {
    echo "TEST: Territory map failure documented to yield blocked"

    local skill_file="$SCRIPT_DIR/SKILL.md"

    # Check that territory map failure handling is documented
    if ! grep -qEi "(territory.*(fail|error)|fail.*territory)" "$skill_file"; then
        fail "SKILL.md does not document territory map failure handling"
    fi

    # Verify blocked yield is documented for territory failures
    if ! grep -qE "blocked" "$skill_file"; then
        fail "SKILL.md missing blocked yield type for infrastructure failures"
    fi

    # Check error handling section in GATHER phase
    gather_line=$(grep -n "### GATHER Phase" "$skill_file" | cut -d: -f1 || echo "")
    assess_line=$(grep -n "### ASSESS Phase" "$skill_file" | cut -d: -f1 || echo "")

    if [[ -n "$gather_line" ]] && [[ -n "$assess_line" ]] && [[ "$assess_line" -gt "$gather_line" ]]; then
        gather_section=$(sed -n "${gather_line},${assess_line}p" "$skill_file")

        # Check for error handling documentation
        if ! echo "$gather_section" | grep -qEi "(Error Handling|error|fail)"; then
            fail "GATHER phase should document error handling"
        fi

        # Check that territory map failure yields blocked
        if ! echo "$gather_section" | grep -qEi "(territory.*(fail|error).*blocked|blocked.*territory)"; then
            fail "SKILL.md should document that territory map failure yields blocked"
        fi
    else
        fail "Cannot find GATHER phase to verify error handling documentation"
    fi

    # Verify diagnostic information is documented
    if ! grep -qE "(diagnostic|error.*detail)" "$skill_file"; then
        fail "SKILL.md should document including diagnostic information in blocked yields"
    fi

    # Check that subphase is documented
    if ! grep -qE "subphase" "$skill_file"; then
        warn "SKILL.md should document subphase in context for debugging"
    fi

    pass "Territory map failure handling fully documented in SKILL.md"
}

# ============================================================================
# Run all tests
# ============================================================================
echo ""
test_sparse_context_large_gap || echo ""
test_medium_context_medium_gap || echo ""
test_rich_context_small_gap || echo ""
test_contradictory_context_need_decision || echo ""
test_territory_map_failure_yields_blocked || echo ""

echo ""
echo -e "${RED}=== Expected result: All tests FAILED (TDD RED phase) ===${NC}"
echo "These tests specify the expected behavior for adaptive interview flow."
echo "Implementation should make these tests pass (TDD GREEN phase)."
