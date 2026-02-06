#!/bin/bash
# Test script for ISSUE-88: Clean up remaining yield references
# This script should FAIL (exit non-zero) before cleanup and PASS (exit 0) after cleanup

set -e  # Exit on first error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# Color output for readability
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

# Helper function to run a check
run_check() {
    local check_name="$1"
    local check_command="$2"

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "CHECK $TOTAL_CHECKS: $check_name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    if eval "$check_command"; then
        echo -e "${GREEN}✓ PASS${NC}: $check_name"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}: $check_name"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi
}

# TASK-13 verification: grep -ri "yield" returns zero matches (excluding allowlisted files)
check_yield_in_docs() {
    echo "Searching for 'yield' references in documentation..."

    # Define allowlist patterns (files that are allowed to contain "yield")
    local allowlist=(
        ".claude/projects/issue-88/"
        "retro.md"
        "scripts/test-yield-cleanup.sh"
    )

    # Build grep exclude arguments
    local exclude_args=""
    for pattern in "${allowlist[@]}"; do
        exclude_args="$exclude_args --exclude-dir='*${pattern}*'"
    done

    # Search for "yield" in markdown files
    local matches
    matches=$(grep -ri "yield" \
        --include="*.md" \
        docs/ \
        .claude/skills/ \
        .claude/projects/ \
        README.md \
        CONTRIBUTING.md \
        2>/dev/null \
        | grep -v ".claude/projects/issue-88/" \
        | grep -v "retro.md" \
        | grep -v "scripts/test-yield-cleanup.sh" \
        || true)

    if [ -z "$matches" ]; then
        echo "No yield references found in documentation."
        return 0
    else
        echo -e "${YELLOW}Found yield references:${NC}"
        echo "$matches"
        return 1
    fi
}

# TASK-13 verification: grep for yield in SKILL.md files
check_yield_in_skills() {
    echo "Searching for 'yield' references in SKILL.md files..."

    local matches
    matches=$(find .claude/skills -name "SKILL.md" -type f -exec grep -Hi "yield" {} + 2>/dev/null || true)

    if [ -z "$matches" ]; then
        echo "No yield references found in SKILL.md files."
        return 0
    else
        echo -e "${YELLOW}Found yield references:${NC}"
        echo "$matches"
        return 1
    fi
}

# TASK-12 verification: grep for yield_path and producer_yield_path in .toml files
check_yield_in_toml() {
    echo "Searching for 'yield_path' and 'producer_yield_path' in .toml files..."

    local matches
    matches=$(grep -r "yield_path\|producer_yield_path" \
        --include="*.toml" \
        . \
        2>/dev/null || true)

    if [ -z "$matches" ]; then
        echo "No yield_path or producer_yield_path references found in .toml files."
        return 0
    else
        echo -e "${YELLOW}Found yield path references:${NC}"
        echo "$matches"
        return 1
    fi
}

# TASK-15 verification: check for broken references to "Yield Protocol" sections
check_broken_yield_links() {
    echo "Checking for broken references to 'Yield Protocol' sections..."

    # Search for markdown links that might reference yield sections
    local yield_section_refs
    yield_section_refs=$(grep -ri "\[.*\](#.*yield.*)" \
        --include="*.md" \
        docs/ \
        .claude/skills/ \
        2>/dev/null \
        | grep -v ".claude/projects/issue-88/" \
        | grep -v "retro.md" \
        || true)

    # Also search for direct section references
    local yield_sections
    yield_sections=$(grep -ri "^##.*[Yy]ield.*[Pp]rotocol" \
        --include="*.md" \
        docs/ \
        .claude/skills/ \
        2>/dev/null \
        | grep -v ".claude/projects/issue-88/" \
        || true)

    if [ -z "$yield_section_refs" ] && [ -z "$yield_sections" ]; then
        echo "No broken references to Yield Protocol sections found."
        return 0
    else
        if [ -n "$yield_section_refs" ]; then
            echo -e "${YELLOW}Found links to yield sections:${NC}"
            echo "$yield_section_refs"
        fi
        if [ -n "$yield_sections" ]; then
            echo -e "${YELLOW}Found Yield Protocol section headers:${NC}"
            echo "$yield_sections"
        fi
        return 1
    fi
}

# TASK-11 verification: verify root-level yield files don't exist
check_root_yield_files() {
    echo "Checking for root-level yield.toml files..."

    if [ -f "yield.toml" ]; then
        echo -e "${YELLOW}Found: yield.toml${NC}"
        return 1
    fi

    if [ -f ".claude/yield.toml" ]; then
        echo -e "${YELLOW}Found: .claude/yield.toml${NC}"
        return 1
    fi

    echo "No root-level yield.toml files found."
    return 0
}

# TASK-14 verification: mage check passes
check_mage() {
    echo "Running 'mage check'..."

    if ! command -v mage &> /dev/null; then
        echo -e "${YELLOW}mage not found, skipping${NC}"
        return 0
    fi

    if mage check; then
        echo "mage check passed."
        return 0
    else
        echo -e "${YELLOW}mage check failed.${NC}"
        return 1
    fi
}

# Additional check: verify yield.type references removed
check_yield_type() {
    echo "Searching for 'yield.type' references..."

    local matches
    matches=$(grep -r "yield\.type" \
        --include="*.md" \
        --include="*.toml" \
        --include="*.go" \
        . \
        2>/dev/null \
        | grep -v ".claude/projects/issue-88/" \
        | grep -v "retro.md" \
        | grep -v "scripts/test-yield-cleanup.sh" \
        || true)

    if [ -z "$matches" ]; then
        echo "No yield.type references found."
        return 0
    else
        echo -e "${YELLOW}Found yield.type references:${NC}"
        echo "$matches"
        return 1
    fi
}

# Run all checks
echo "════════════════════════════════════════════"
echo "ISSUE-88: Yield Reference Cleanup Tests"
echo "════════════════════════════════════════════"
echo ""
echo "Running verification checks..."

run_check "TASK-13: No yield references in docs/" check_yield_in_docs || true
run_check "TASK-13: No yield references in SKILL.md files" check_yield_in_skills || true
run_check "TASK-12: No yield_path in .toml files" check_yield_in_toml || true
run_check "TASK-15: No broken Yield Protocol references" check_broken_yield_links || true
run_check "TASK-11: No root-level yield.toml files" check_root_yield_files || true
run_check "TASK-12: No yield.type references" check_yield_type || true
run_check "TASK-14: mage check passes" check_mage || true

# Print summary
echo ""
echo "════════════════════════════════════════════"
echo "TEST SUMMARY"
echo "════════════════════════════════════════════"
echo "Total checks: $TOTAL_CHECKS"
echo -e "${GREEN}Passed: $PASSED_CHECKS${NC}"
echo -e "${RED}Failed: $FAILED_CHECKS${NC}"
echo ""

if [ $FAILED_CHECKS -eq 0 ]; then
    echo -e "${GREEN}✓ ALL CHECKS PASSED${NC}"
    echo "Yield reference cleanup is complete!"
    exit 0
else
    echo -e "${RED}✗ SOME CHECKS FAILED${NC}"
    echo "Yield references still exist. Cleanup needed."
    exit 1
fi
