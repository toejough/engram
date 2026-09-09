#!/bin/bash
set -e

TARGET_DIR="${1:-.}"

# Copy template files to target directory
cp -r dev/eval/cumulative/runbook_vs_skill/fixture-repo-template/* "$TARGET_DIR/"
cp dev/eval/cumulative/runbook_vs_skill/fixture-repo-template/.gitignore "$TARGET_DIR/"

# Initialize git repo
cd "$TARGET_DIR"
git init
git config user.name "fixture"
git config user.email "fixture@example.com"
git add -A
git commit -m "Initial commit: sensor system v1"
