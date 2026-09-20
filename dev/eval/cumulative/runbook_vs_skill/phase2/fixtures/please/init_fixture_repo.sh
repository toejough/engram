#!/bin/bash
set -euo pipefail

TARGET_DIR="$1"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/../fixture-repo-templates/please-task" && pwd)"

PARENT_DIR="$(mkdir -p "$(dirname "$TARGET_DIR")" && cd "$(dirname "$TARGET_DIR")" && pwd)"
EVAL_DIR="$PARENT_DIR/.eval"
rm -rf "$EVAL_DIR"
mkdir -p "$EVAL_DIR"

mkdir -p "$TARGET_DIR"
cp -r "$TEMPLATE/." "$TARGET_DIR/"
cd "$TARGET_DIR"

# Template stores the ignore file as _gitignore so the outer repo's own rules never apply to it.
mv _gitignore .gitignore

git init -q -b main
git config user.name "Trial Agent"
git config user.email "trial@example.com"

# C1: the tally CLI with a `--out` flag echoed across code, tests, README, docs, a diagram, and a
# shell completion script, plus a CHANGELOG whose 0.2.0 entry historically records `--out`.
git add -A
git commit -qm "chore: initial tally CLI, tests, and docs"

ORIG_TIP=$(git rev-parse HEAD)
echo "$ORIG_TIP" > "$EVAL_DIR/original_tip"
