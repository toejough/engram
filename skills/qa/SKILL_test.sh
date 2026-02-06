#!/bin/bash
# SKILL.md validation tests for universal QA skill
# Run: bash skills/qa/SKILL_test.sh

set -e
SKILL_FILE="skills/qa/SKILL.md"

echo "=== Universal QA Skill Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# Check frontmatter exists
if ! grep -q "^---" "$SKILL_FILE"; then
    echo "FAIL: No frontmatter found"
    exit 1
fi
echo "PASS: Frontmatter exists"

# Check required frontmatter fields
if grep -q "^name: qa$" "$SKILL_FILE"; then
    echo "PASS: name: qa present"
else
    echo "FAIL: name must be 'qa'"
    exit 1
fi

if grep -q "^model: haiku$" "$SKILL_FILE"; then
    echo "PASS: model: haiku present"
else
    echo "FAIL: model must be 'haiku'"
    exit 1
fi

if grep -q "^role: qa$" "$SKILL_FILE"; then
    echo "PASS: role: qa present"
else
    echo "FAIL: role must be 'qa'"
    exit 1
fi

# Check required workflow phases documented
PHASES=("LOAD" "VALIDATE" "RETURN")
for phase in "${PHASES[@]}"; do
    if grep -qi "$phase" "$SKILL_FILE"; then
        echo "PASS: Phase '$phase' documented"
    else
        echo "FAIL: Phase '$phase' NOT documented"
        exit 1
    fi
done

# Check contract extraction is documented (ARCH-021)
if grep -qi "contract" "$SKILL_FILE" && grep -qi "extract" "$SKILL_FILE"; then
    echo "PASS: Contract extraction documented"
else
    echo "FAIL: Contract extraction not documented"
    exit 1
fi

# Check all response types documented (DES-005)
RESPONSE_TYPES=("approved" "improvement-request" "escalate" "error")
for response_type in "${RESPONSE_TYPES[@]}"; do
    if grep -q "$response_type" "$SKILL_FILE"; then
        echo "PASS: Response type '$response_type' documented"
    else
        echo "FAIL: Response type '$response_type' NOT documented"
        exit 1
    fi
done

# Check iteration tracking documented (ARCH-028)
if grep -qi "iteration" "$SKILL_FILE" && grep -q "3" "$SKILL_FILE"; then
    echo "PASS: Iteration tracking (max 3) documented"
else
    echo "FAIL: Iteration tracking not documented"
    exit 1
fi

# Check error handling for missing artifacts (DES-007)
if grep -qi "missing.*artifact" "$SKILL_FILE" || grep -qi "artifact.*not found" "$SKILL_FILE" || grep -qi "file not found" "$SKILL_FILE"; then
    echo "PASS: Missing artifact handling documented"
else
    echo "FAIL: Missing artifact handling not documented"
    exit 1
fi

# Check error handling for unreadable SKILL.md (DES-009)
if grep -qi "unreadable" "$SKILL_FILE" || grep -qi "cannot read" "$SKILL_FILE"; then
    echo "PASS: Unreadable SKILL.md handling documented"
else
    echo "FAIL: Unreadable SKILL.md handling not documented"
    exit 1
fi

# Check full checklist output format (DES-003)
if grep -q "\[x\]" "$SKILL_FILE" || grep -q "\[ \]" "$SKILL_FILE"; then
    echo "PASS: Checklist output format documented"
else
    echo "FAIL: Checklist output format not documented"
    exit 1
fi

# Check QA context inputs documented (DES-004)
CONTEXT_FIELDS=("SKILL.md path" "Artifact paths" "Iteration")
for field in "${CONTEXT_FIELDS[@]}"; do
    if grep -qi "$field" "$SKILL_FILE"; then
        echo "PASS: Context field '$field' documented"
    else
        echo "FAIL: Context field '$field' NOT documented"
        exit 1
    fi
done

# Check references to CONTRACT.md
if grep -q "CONTRACT.md" "$SKILL_FILE"; then
    echo "PASS: References CONTRACT.md"
else
    echo "FAIL: Missing reference to CONTRACT.md"
    exit 1
fi

# Check user-invocable is true
if grep -q "^user-invocable: true$" "$SKILL_FILE"; then
    echo "PASS: user-invocable: true present"
else
    echo "FAIL: user-invocable must be true"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
