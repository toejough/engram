#!/bin/bash
set -euo pipefail

TARGET_DIR="$1"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/../fixture-repo-templates/commit-task" && pwd)"

# Copy template files to target directory
mkdir -p "$TARGET_DIR"
cp -r "$TEMPLATE/." "$TARGET_DIR/"

cd "$TARGET_DIR"

# Initialize git repo
git init
git config user.name "Trial Agent"
git config user.email "trial@example.com"

# Create the fixture commit (empty repo with version.go and notes/scratch.txt created but unstaged)
git add pkg/version.go
git commit -m "chore: initial version constant"

# Modify version.go (leave unstaged) and create untracked notes/scratch.txt
cat > pkg/version.go <<'EOF'
package pkg

const Version = "1.1.0"
EOF

echo "Fixture repo initialized at $TARGET_DIR"
