#!/bin/bash
# TASK-2: Key Questions section validation tests
# Verifies arch-interview-producer SKILL.md has Key Questions section with proper structure
# Run: bash .claude/projects/ISSUE-061/tests/task2_key_questions_test.sh

set -e
SKILL_FILE="$HOME/.claude/skills/arch-interview-producer/SKILL.md"

echo "=== TASK-2: Key Questions Section Validation Tests ==="

# AC-1: ## Key Questions section exists
if grep -q '^## Key Questions' "$SKILL_FILE"; then
    echo "PASS: AC-1 - ## Key Questions section exists"
else
    echo "FAIL: AC-1 - ## Key Questions section missing"
    exit 1
fi

# AC-2a: 8-12 questions defined - check minimum count
question_count=$(grep -c '^\*\*[^*]*\*\* -' "$SKILL_FILE" || echo 0)
if [[ $question_count -ge 8 ]] && [[ $question_count -le 12 ]]; then
    echo "PASS: AC-2a - Question count within range (found: $question_count)"
else
    echo "FAIL: AC-2a - Question count not in 8-12 range (found: $question_count)"
    exit 1
fi

# AC-2b: Technology stack question exists
if grep -qi 'technology stack' "$SKILL_FILE" || grep -qi 'languages.*frameworks' "$SKILL_FILE"; then
    echo "PASS: AC-2b - Technology stack question exists"
else
    echo "FAIL: AC-2b - Technology stack question missing"
    exit 1
fi

# AC-2c: Scale question exists
if grep -qi 'scale' "$SKILL_FILE" && (grep -qi 'users' "$SKILL_FILE" || grep -qi 'data volume' "$SKILL_FILE"); then
    echo "PASS: AC-2c - Scale question exists"
else
    echo "FAIL: AC-2c - Scale question missing"
    exit 1
fi

# AC-2d: Deployment question exists
if grep -qi 'deployment' "$SKILL_FILE" || grep -qi 'where.*run' "$SKILL_FILE"; then
    echo "PASS: AC-2d - Deployment question exists"
else
    echo "FAIL: AC-2d - Deployment question missing"
    exit 1
fi

# AC-2e: Integrations question exists
if grep -qi 'integration' "$SKILL_FILE" || grep -qi 'external system' "$SKILL_FILE"; then
    echo "PASS: AC-2e - Integrations question exists"
else
    echo "FAIL: AC-2e - Integrations question missing"
    exit 1
fi

# AC-2f: Performance question exists
if grep -qi 'performance' "$SKILL_FILE" || grep -qi 'response time' "$SKILL_FILE" || grep -qi 'SLA' "$SKILL_FILE"; then
    echo "PASS: AC-2f - Performance question exists"
else
    echo "FAIL: AC-2f - Performance question missing"
    exit 1
fi

# AC-2g: Security question exists
if grep -qi 'security' "$SKILL_FILE" || grep -qi 'authentication' "$SKILL_FILE" || grep -qi 'authorization' "$SKILL_FILE"; then
    echo "PASS: AC-2g - Security question exists"
else
    echo "FAIL: AC-2g - Security question missing"
    exit 1
fi

# AC-2h: Data durability question exists
if grep -qi 'data durability' "$SKILL_FILE" || grep -qi 'data loss' "$SKILL_FILE"; then
    echo "PASS: AC-2h - Data durability question exists"
else
    echo "FAIL: AC-2h - Data durability question missing"
    exit 1
fi

# AC-2i: Observability question exists
if grep -qi 'observability' "$SKILL_FILE" || grep -qi 'logging' "$SKILL_FILE" || grep -qi 'monitoring' "$SKILL_FILE"; then
    echo "PASS: AC-2i - Observability question exists"
else
    echo "FAIL: AC-2i - Observability question missing"
    exit 1
fi

# AC-3a: Questions tagged with "critical" priority
if grep -qi '(critical)' "$SKILL_FILE"; then
    echo "PASS: AC-3a - Critical priority tag exists"
else
    echo "FAIL: AC-3a - Critical priority tag missing"
    exit 1
fi

# AC-3b: Questions tagged with "important" priority
if grep -qi '(important)' "$SKILL_FILE"; then
    echo "PASS: AC-3b - Important priority tag exists"
else
    echo "FAIL: AC-3b - Important priority tag missing"
    exit 1
fi

# AC-3c: Questions tagged with "optional" priority
if grep -qi '(optional)' "$SKILL_FILE"; then
    echo "PASS: AC-3c - Optional priority tag exists"
else
    echo "FAIL: AC-3c - Optional priority tag missing"
    exit 1
fi

# AC-3d: Critical priority count within range (2-4)
critical_count=$(grep -ci '(critical)' "$SKILL_FILE" || echo 0)
if [[ $critical_count -ge 2 ]] && [[ $critical_count -le 4 ]]; then
    echo "PASS: AC-3d - Critical priority count within range (found: $critical_count)"
else
    echo "FAIL: AC-3d - Critical priority count not in 2-4 range (found: $critical_count)"
    exit 1
fi

# AC-3e: Important priority count within range (3-5)
important_count=$(grep -ci '(important)' "$SKILL_FILE" || echo 0)
if [[ $important_count -ge 3 ]] && [[ $important_count -le 5 ]]; then
    echo "PASS: AC-3e - Important priority count within range (found: $important_count)"
else
    echo "FAIL: AC-3e - Important priority count not in 3-5 range (found: $important_count)"
    exit 1
fi

# AC-3f: Optional priority count within range (2-3)
optional_count=$(grep -ci '(optional)' "$SKILL_FILE" || echo 0)
if [[ $optional_count -ge 2 ]] && [[ $optional_count -le 3 ]]; then
    echo "PASS: AC-3f - Optional priority count within range (found: $optional_count)"
else
    echo "FAIL: AC-3f - Optional priority count not in 2-3 range (found: $optional_count)"
    exit 1
fi

# AC-4: Question format includes topic, question text, and priority level
# Check for pattern like: **Topic** - Question text? (priority)
if grep -qE '\*\*[^*]+\*\* - .+\? \((critical|important|optional)\)' "$SKILL_FILE"; then
    echo "PASS: AC-4 - Question format includes topic, question text, and priority"
else
    echo "FAIL: AC-4 - Question format does not match expected pattern"
    exit 1
fi

# AC-5a: Coverage weight for critical documented
if grep -qi 'critical.*-15%' "$SKILL_FILE" || grep -qi '-15%.*critical' "$SKILL_FILE"; then
    echo "PASS: AC-5a - Critical weight (-15%) documented"
else
    echo "FAIL: AC-5a - Critical weight (-15%) not documented"
    exit 1
fi

# AC-5b: Coverage weight for important documented
if grep -qi 'important.*-10%' "$SKILL_FILE" || grep -qi '-10%.*important' "$SKILL_FILE"; then
    echo "PASS: AC-5b - Important weight (-10%) documented"
else
    echo "FAIL: AC-5b - Important weight (-10%) not documented"
    exit 1
fi

# AC-5c: Coverage weight for optional documented
if grep -qi 'optional.*-5%' "$SKILL_FILE" || grep -qi '-5%.*optional' "$SKILL_FILE"; then
    echo "PASS: AC-5c - Optional weight (-5%) documented"
else
    echo "FAIL: AC-5c - Optional weight (-5%) not documented"
    exit 1
fi

# AC-6: Examples showing how questions map to architecture decisions
# Look for "example" or "map" keywords near question section
section_start=$(grep -n '^## Key Questions' "$SKILL_FILE" | cut -d: -f1)
section_end=$(tail -n +$section_start "$SKILL_FILE" | grep -n '^## ' | head -2 | tail -1 | cut -d: -f1)
if [[ -z "$section_end" ]]; then
    # If no next section found, check to end of file
    section_content=$(tail -n +$section_start "$SKILL_FILE")
else
    section_content=$(tail -n +$section_start "$SKILL_FILE" | head -n $section_end)
fi

if echo "$section_content" | grep -qi 'example'; then
    echo "PASS: AC-6 - Examples showing question mapping provided"
else
    echo "FAIL: AC-6 - Examples showing question mapping missing"
    exit 1
fi

echo ""
echo "=== All TASK-2 tests passed ==="
