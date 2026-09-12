#!/bin/bash
set -euo pipefail

TARGET_DIR="$1"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/../fixture-repo-templates/opsx-propose-task" && pwd)"

mkdir -p "$TARGET_DIR"
cp -r "$TEMPLATE/." "$TARGET_DIR/"
cd "$TARGET_DIR"

# openspec init also scaffolds an empty archive/ dir under changes/ -- git can't track an empty
# dir, so it's created here rather than stored in the template.
mkdir -p openspec/changes/archive

git init -q -b main
git config user.name "Trial Agent"
git config user.email "trial@example.com"
git add -A
git commit -qm "chore: initial widget service with a list-only widget-api spec"

echo "Fixture repo initialized at $TARGET_DIR"
