#!/bin/bash
set -e

TARGET_DIR="${1:-.}"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/fixture-repo-template"

# Create target directory if it doesn't exist
mkdir -p "$TARGET_DIR"

# Copy template files to target directory
cp -r "$TEMPLATE/." "$TARGET_DIR/"

# Initialize git repo
cd "$TARGET_DIR"
git init
git config user.name "fixture"
git config user.email "fixture@example.com"
git add -A
git commit -m "Initial commit: sensor system v1"
