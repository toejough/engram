#!/bin/bash
set -euo pipefail

TARGET_DIR="$1"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/../fixture-repo-templates/gitignore-nested-task" && pwd)"

# Copy template files to target directory
mkdir -p "$TARGET_DIR"
cp -r "$TEMPLATE/." "$TARGET_DIR/"

cd "$TARGET_DIR"

# Move _gitignore to .gitignore (template stores it as _gitignore to avoid the outer repo's own
# git tooling treating it as a real .gitignore and hiding the template's own files)
mv _gitignore .gitignore

# Initialize git repo
git init
git config user.name "Trial Agent"
git config user.email "trial@example.com"

# Create the fixture commit with the over-broad .gitignore. Because "testdata/" (no leading or
# embedded slash) matches a directory named testdata at ANY depth, git add -A respects it and
# leaves every testdata/ path -- at every nesting level -- untracked, including the fixture.json
# files the task needs made trackable.
git add -A
git commit -m "chore: initial project with broad nested ignore"

# Generate the rapid/ generated-data artifacts (at every depth) in the ignored directories,
# after the fixture commit -- these must stay ignored.
mkdir -p internal/core/testdata/rapid internal/api/testdata/rapid testdata/rapid
dd if=/dev/zero bs=1024 count=64 2>/dev/null > internal/core/testdata/rapid/big.bin
dd if=/dev/zero bs=1024 count=64 2>/dev/null > internal/api/testdata/rapid/big.bin
dd if=/dev/zero bs=1024 count=64 2>/dev/null > testdata/rapid/big.bin

# Create decoy file (untracked)
cat > tmp.log <<'EOF'
temporary log file
EOF

echo "Fixture repo initialized at $TARGET_DIR"
