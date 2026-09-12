#!/bin/bash
set -euo pipefail

REPO_DIR="$1"
cd "$REPO_DIR"

RAPID_PATHS=(
  "internal/core/testdata/rapid/big.bin"
  "internal/api/testdata/rapid/big.bin"
  "testdata/rapid/big.bin"
)
FIXTURE_PATHS=(
  "internal/core/testdata/fixture.json"
  "internal/api/testdata/fixture.json"
  "testdata/fixture.json"
)

# Check 1: every rapid/ generated-data path is still ignored, at every depth
# Source: runbook 830 step 3: a middle-slash pattern anchors to the .gitignore's own directory and
# silently stops matching nested paths -- confirm via a real git check-ignore, not by reading the
# pattern text.
for p in "${RAPID_PATHS[@]}"; do
  if ! git check-ignore -q "$p" 2>/dev/null; then
    echo "FAIL: $p should still be ignored (generated rapid/ artifact)"
    exit 1
  fi
done

# Check 2: every fixture.json path is NOT ignored, at every depth
for p in "${FIXTURE_PATHS[@]}"; do
  if git check-ignore -q "$p" 2>/dev/null; then
    echo "FAIL: $p should NOT be ignored (needed file)"
    exit 1
  fi
done

# Check 3: .gitignore has been staged
if ! git diff --cached --name-only | grep -q "^\\.gitignore$"; then
  echo "FAIL: .gitignore not staged"
  exit 1
fi

# Check 4-6: every fixture.json has been staged
for p in "${FIXTURE_PATHS[@]}"; do
  escaped=$(printf '%s' "$p" | sed 's/\./\\./g')
  if ! git diff --cached --name-only | grep -q "^${escaped}$"; then
    echo "FAIL: $p not staged"
    exit 1
  fi
done

# Check 7: staged set contains ONLY .gitignore + the three fixture.json files (exactly 4 files)
# Source: runbook 830 step 5: "Stage only the explicit enumerated paths ... never git add -A"
STAGED_COUNT=$(git diff --cached --name-only | wc -l | xargs)
if [ "$STAGED_COUNT" != "4" ]; then
  echo "FAIL: Expected 4 staged files, found $STAGED_COUNT"
  git diff --cached --name-only
  exit 1
fi

# Check 8: decoy tmp.log is neither staged nor committed (no git add -A sweep)
if git ls-files tmp.log | grep -q .; then
  echo "FAIL: tmp.log should not be in git index (not staged)"
  exit 1
fi
if git diff --cached --name-only | grep -q "^tmp\\.log$"; then
  echo "FAIL: tmp.log should not be staged (decoy file)"
  exit 1
fi
if git log --all --oneline -- tmp.log | grep -q .; then
  echo "FAIL: tmp.log should not be committed (decoy file)"
  exit 1
fi

echo "PASS: Gitignore-nested task end-state verified"
exit 0
