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

# Check all yield types documented (DES-005)
YIELD_TYPES=("approved" "improvement-request" "escalate-phase" "escalate-user" "error")
for yield_type in "${YIELD_TYPES[@]}"; do
    if grep -q "$yield_type" "$SKILL_FILE"; then
        echo "PASS: Yield type '$yield_type' documented"
    else
        echo "FAIL: Yield type '$yield_type' NOT documented"
        exit 1
    fi
done

# Check prose fallback documented (ARCH-024)
if grep -qi "fallback" "$SKILL_FILE" || grep -qi "prose" "$SKILL_FILE"; then
    echo "PASS: Fallback/prose handling documented"
else
    echo "FAIL: Fallback/prose handling not documented"
    exit 1
fi

# Check iteration tracking documented (ARCH-028)
if grep -qi "iteration" "$SKILL_FILE" && grep -q "3" "$SKILL_FILE"; then
    echo "PASS: Iteration tracking (max 3) documented"
else
    echo "FAIL: Iteration tracking not documented"
    exit 1
fi

# Check error handling for malformed yield (DES-006)
if grep -qi "malformed" "$SKILL_FILE" || grep -qi "parse error" "$SKILL_FILE" || grep -qi "invalid.*yield" "$SKILL_FILE"; then
    echo "PASS: Malformed yield handling documented"
else
    echo "FAIL: Malformed yield handling not documented"
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
CONTEXT_FIELDS=("producer_skill_path" "producer_yield_path" "artifact_paths")
for field in "${CONTEXT_FIELDS[@]}"; do
    if grep -q "$field" "$SKILL_FILE"; then
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

# Check at least 2 TOML examples (yield examples)
TOML_COUNT=$(grep -c '```toml' "$SKILL_FILE" || true)
if [[ $TOML_COUNT -ge 2 ]]; then
    echo "PASS: $TOML_COUNT TOML examples found (>= 2 required)"
else
    echo "FAIL: Only $TOML_COUNT TOML examples (>= 2 required)"
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
