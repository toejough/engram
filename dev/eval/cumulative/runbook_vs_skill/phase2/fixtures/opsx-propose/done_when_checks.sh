#!/bin/bash
set -euo pipefail

REPO_DIR="$1"
cd "$REPO_DIR"

if [ ! -d openspec/changes ]; then
  echo "FAIL: openspec/changes directory is missing"
  exit 1
fi

mapfile -t CHANGE_DIRS < <(find openspec/changes -mindepth 1 -maxdepth 1 -type d ! -name archive | sort)
if [ "${#CHANGE_DIRS[@]}" -ne 1 ]; then
  echo "FAIL: expected exactly one new change dir under openspec/changes/ (excluding archive/), found ${#CHANGE_DIRS[@]}: ${CHANGE_DIRS[*]:-none}"
  exit 1
fi
CHANGE_DIR="${CHANGE_DIRS[0]}"
CHANGE_NAME="$(basename "$CHANGE_DIR")"

for f in proposal.md design.md tasks.md; do
  if [ ! -f "$CHANGE_DIR/$f" ]; then
    echo "FAIL: $CHANGE_DIR is missing $f"
    exit 1
  fi
done

mapfile -t SPEC_FILES < <(find "$CHANGE_DIR/specs" -type f -name '*.md' 2>/dev/null | sort)
if [ "${#SPEC_FILES[@]}" -eq 0 ]; then
  echo "FAIL: $CHANGE_DIR has no spec delta file(s) under specs/"
  exit 1
fi

if ! grep -Eq '^## (ADDED|MODIFIED) Requirements' "${SPEC_FILES[@]}"; then
  echo "FAIL: no spec delta file under $CHANGE_DIR/specs uses the ADDED/MODIFIED requirement grammar"
  exit 1
fi

if ! grep -Eq '^- \[[ x]\]' "$CHANGE_DIR/tasks.md"; then
  echo "FAIL: $CHANGE_DIR/tasks.md has no checkbox items"
  exit 1
fi

STATUS_JSON=$(openspec status --change "$CHANGE_NAME" --json 2>&1) || {
  echo "FAIL: 'openspec status --change $CHANGE_NAME --json' failed:"
  echo "$STATUS_JSON"
  exit 1
}
IS_COMPLETE=$(echo "$STATUS_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin).get('isComplete'))" 2>/dev/null || echo "PARSE_ERROR")
if [ "$IS_COMPLETE" != "True" ]; then
  echo "FAIL: openspec status reports the change is not complete (isComplete=$IS_COMPLETE)"
  echo "$STATUS_JSON"
  exit 1
fi

if ! openspec validate "$CHANGE_NAME" --strict; then
  echo "FAIL: 'openspec validate $CHANGE_NAME --strict' does not pass (see output above)"
  exit 1
fi

echo "PASS: opsx-propose task end-state verified"
exit 0
