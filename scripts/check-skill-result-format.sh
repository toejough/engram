#!/bin/bash
# Check that all skills have a Result Format section
# TASK-003: Update skills with result format documentation

set -e

SKILLS_DIR="${1:-skills}"
MISSING=()

for skill_dir in "$SKILLS_DIR"/*/; do
    skill_file="$skill_dir/SKILL.md"
    if [[ -f "$skill_file" ]]; then
        if ! grep -q "## Result Format" "$skill_file"; then
            MISSING+=("$skill_file")
        fi
    fi
done

if [[ ${#MISSING[@]} -gt 0 ]]; then
    echo "Skills missing '## Result Format' section:"
    for f in "${MISSING[@]}"; do
        echo "  $f"
    done
    echo ""
    echo "Total: ${#MISSING[@]} skills need updating"
    exit 1
fi

echo "All skills have Result Format section"
exit 0
