#!/bin/bash
set -euo pipefail

TARGET_DIR="$1"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/../fixture-repo-templates/route-task" && pwd)"

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

# C1: the stub CLI without --version, plus one existing, passing test.
git add -A
git commit -qm "chore: initial CLI stub and existing test"

ORIG_TIP=$(git rev-parse HEAD)
echo "$ORIG_TIP" > "$EVAL_DIR/original_tip"

echo "Fixture repo initialized at $TARGET_DIR"
