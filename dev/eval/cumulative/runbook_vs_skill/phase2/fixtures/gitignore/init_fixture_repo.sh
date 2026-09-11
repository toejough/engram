#!/bin/bash
set -euo pipefail

TARGET_DIR="$1"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/../fixture-repo-templates/gitignore-task" && pwd)"

# Copy template files to target directory
mkdir -p "$TARGET_DIR"
cp -r "$TEMPLATE/." "$TARGET_DIR/"

cd "$TARGET_DIR"

# Initialize git repo
git init
git config user.name "Trial Agent"
git config user.email "trial@example.com"

# Create the fixture commit with the over-broad .gitignore
git add .gitignore src
git commit -m "chore: initial project with broad ignore"

# Generate the big.bin file in the ignored directory
dd if=/dev/zero bs=1024 count=64 2>/dev/null > testdata/generated/big.bin

# Create decoy file (untracked)
cat > tmp.log <<'EOF'
temporary log file
EOF

echo "Fixture repo initialized at $TARGET_DIR"
