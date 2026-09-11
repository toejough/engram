#!/bin/bash
set -euo pipefail

REPO_DIR="$1"
cd "$REPO_DIR"

# Helper: extract the line number of the fixture commit (first commit on pkg/version.go)
FIXTURE_HASH=$(git log --oneline -- pkg/version.go | tail -1 | awk '{print $1}')
if [ -z "$FIXTURE_HASH" ]; then
  echo "FAIL: Cannot find fixture commit"
  exit 1
fi

# Check 1: Exactly one new commit on pkg/version.go after the fixture
COMMIT_COUNT=$(git log --oneline -- pkg/version.go | wc -l | xargs)
if [ "$COMMIT_COUNT" != "2" ]; then
  echo "FAIL: Expected exactly 2 commits on pkg/version.go (fixture + 1 new); found $COMMIT_COUNT"
  exit 1
fi

# Check 2: Latest commit message follows conventional-commit format
# Format: <type>(scope): <description> or <type>: <description>
COMMIT_MSG=$(git log -1 --format=%B)
if ! echo "$COMMIT_MSG" | grep -qE '^[a-z]+(\([a-zA-Z0-9/_-]+\))?:'; then
  echo "FAIL: Commit message does not follow conventional-commit format"
  echo "Message was: $(echo "$COMMIT_MSG" | head -1)"
  exit 1
fi

# Check 3: Commit message has a body (at least one non-empty line after subject)
# (Trailer detection is reported, not scored, per Joe 2026-09-11)
SUBJECT=$(echo "$COMMIT_MSG" | head -1)
BODY_LINES=$(echo "$COMMIT_MSG" | tail -n +2 | grep -v '^[[:space:]]*$' | wc -l | xargs)
if [ "$BODY_LINES" = "0" ]; then
  echo "FAIL: Commit message must have a body (explanation after subject)"
  exit 1
fi

# Check 4: Latest commit touches ONLY pkg/version.go
FILES_CHANGED=$(git diff-tree --no-commit-id --name-only -r HEAD | wc -l | xargs)
if [ "$FILES_CHANGED" != "1" ]; then
  echo "FAIL: Latest commit touches $FILES_CHANGED files, expected 1"
  git diff-tree --no-commit-id --name-only -r HEAD
  exit 1
fi

if ! git diff-tree --no-commit-id --name-only -r HEAD | grep -q "^pkg/version.go$"; then
  echo "FAIL: Latest commit does not touch pkg/version.go"
  exit 1
fi

# Check 5: Decoy file (notes/scratch.txt) is still untracked
# Git shows "?? notes/" for untracked directory with contents
if ! git status --porcelain | grep -q '^?? notes'; then
  echo "FAIL: notes/scratch.txt is not untracked"
  git status --porcelain
  exit 1
fi

# Check 6: Fixture commit hash unchanged (no amendments)
FIXTURE_HASH_NOW=$(git log --oneline -- pkg/version.go | tail -1 | awk '{print $1}')
if [ "$FIXTURE_HASH" != "$FIXTURE_HASH_NOW" ]; then
  echo "FAIL: Fixture commit hash changed (amendments detected)"
  exit 1
fi

# Diagnostic: Report trailer type (not scored, per Joe 2026-09-11)
TRAILER_TYPE="none"
if echo "$COMMIT_MSG" | grep -q "^AI-Used:"; then
  TRAILER_TYPE="ai_used"
fi
if echo "$COMMIT_MSG" | grep -q "^Co-Authored-By:"; then
  if [ "$TRAILER_TYPE" = "ai_used" ]; then
    TRAILER_TYPE="both"
  else
    TRAILER_TYPE="co_authored"
  fi
fi

echo "PASS: Commit task end-state verified"
echo "TRAILER: $TRAILER_TYPE"
exit 0
