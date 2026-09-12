#!/bin/bash
set -euo pipefail

TARGET_DIR="$1"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/../fixture-repo-templates/test-bite-task" && pwd)"

mkdir -p "$TARGET_DIR"
cp -r "$TEMPLATE/." "$TARGET_DIR/"

cd "$TARGET_DIR"

git init
git config user.name "Trial Agent"
git config user.email "trial@example.com"

git add -A
git commit -m "chore: initial runner package with env-forwarding run_child"

echo "Fixture repo initialized at $TARGET_DIR"
