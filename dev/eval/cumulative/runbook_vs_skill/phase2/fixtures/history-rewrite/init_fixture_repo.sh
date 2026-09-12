#!/bin/bash
set -euo pipefail

TARGET_DIR="$1"
TEMPLATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/../fixture-repo-templates/history-rewrite-task" && pwd)"

PARENT_DIR="$(mkdir -p "$(dirname "$TARGET_DIR")" && cd "$(dirname "$TARGET_DIR")" && pwd)"
EVAL_DIR="$PARENT_DIR/.eval"
REMOTE_DIR="$PARENT_DIR/remote.git"
rm -rf "$EVAL_DIR" "$REMOTE_DIR"
mkdir -p "$EVAL_DIR"

# Bare remote, standing in for a hosted origin
git init --bare -q -b main "$REMOTE_DIR"

mkdir -p "$TARGET_DIR"
cp -r "$TEMPLATE/." "$TARGET_DIR/"
cd "$TARGET_DIR"

git init -q -b main
git config user.name "Trial Agent"
git config user.email "trial@example.com"
git remote add origin "$REMOTE_DIR"

# C1
git add file1.txt
git commit -qm "chore: add file1"

# C2: introduces secrets.env
cat > secrets.env <<'EOF'
API_KEY=fake-not-a-real-secret-1234
EOF
git add secrets.env
git commit -qm "chore: add local env config"

# C3: removes secrets.env (it's gone from the tip, but the commit that ADDED it is still in history)
git rm -q secrets.env
git commit -qm "fix: stop tracking local env config"

# C4
git add file2.txt
git commit -qm "chore: add file2"

# C5
git add file3.txt
git commit -qm "chore: add file3"

git push -q origin main

ORIG_TIP=$(git rev-parse HEAD)
echo "$ORIG_TIP" > "$EVAL_DIR/original_tip"
echo "$REMOTE_DIR" > "$EVAL_DIR/remote_path"

echo "Fixture repo initialized at $TARGET_DIR"
