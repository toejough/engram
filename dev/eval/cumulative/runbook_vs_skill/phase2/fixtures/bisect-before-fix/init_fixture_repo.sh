#!/bin/bash
set -euo pipefail

TARGET_DIR="$1"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/../fixture-repo-templates/bisect-before-fix-task" && pwd)"

PARENT_DIR="$(mkdir -p "$(dirname "$TARGET_DIR")" && cd "$(dirname "$TARGET_DIR")" && pwd)"
EVAL_DIR="$PARENT_DIR/.eval"
rm -rf "$EVAL_DIR"
mkdir -p "$EVAL_DIR"

mkdir -p "$TARGET_DIR"
cp -r "$TEMPLATE/." "$TARGET_DIR/"
cd "$TARGET_DIR"

git init -q -b main
git config user.name "Trial Agent"
git config user.email "trial@example.com"

# C1 (parent): unfixed foo.py, plus bar.py's pre-existing debug-print bug, gate.sh, PLAN.md
git add -A
git commit -qm "chore: initial project with unfixed foo.py and a pre-existing bar.py issue"
PARENT_TIP=$(git rev-parse HEAD)
echo "$PARENT_TIP" > "$EVAL_DIR/parent_tip"

# C2 (HEAD): apply ONLY the PLAN.md-prescribed foo.py change; bar.py is left untouched
sed -i.bak 's/return False/return True/' foo.py
rm -f foo.py.bak
git add foo.py
git commit -qm "fix: apply prescribed change to foo.py"
ORIG_TIP=$(git rev-parse HEAD)
echo "$ORIG_TIP" > "$EVAL_DIR/original_tip"

echo "Fixture repo initialized at $TARGET_DIR"
