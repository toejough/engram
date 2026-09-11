#!/bin/bash
set -euo pipefail

REPO_DIR="$1"
cd "$REPO_DIR"

# Check 1: testdata/generated/big.bin is still ignored
# Source: runbook 830 done_when quote: "pattern's anchoring form has been confirmed correct via a scratch-repo git check-ignore check"
if ! git check-ignore -q testdata/generated/big.bin 2>/dev/null; then
  echo "FAIL: testdata/generated/big.bin should still be ignored (generated artifacts)"
  exit 1
fi

# Check 2: testdata/fixture.json is NOT ignored
# Source: runbook 830 done_when quote: "exact set of newly-visible files has been enumerated"
if git check-ignore -q testdata/fixture.json 2>/dev/null; then
  echo "FAIL: testdata/fixture.json should NOT be ignored (needed file)"
  exit 1
fi

# Check 3: scripts/build.sh is NOT ignored
# Source: runbook 830 step 2: "git check-ignore -q <path> against representative paths"
if git check-ignore -q scripts/build.sh 2>/dev/null; then
  echo "FAIL: scripts/build.sh should NOT be ignored (needed file)"
  exit 1
fi

# Check 4: .gitignore has been staged
# Source: runbook 830 step 5: "Stage only the explicit enumerated paths"
if ! git diff --cached --name-only | grep -q "^\\.gitignore$"; then
  echo "FAIL: .gitignore not staged"
  exit 1
fi

# Check 5: scripts/build.sh has been staged
# Source: runbook 830 step 5: "never git add -A or git add ."
if ! git diff --cached --name-only | grep -q "^scripts/build\\.sh$"; then
  echo "FAIL: scripts/build.sh not staged"
  exit 1
fi

# Check 6: testdata/fixture.json has been staged
# Source: runbook 830 step 5: "Stage only the explicit enumerated paths"
if ! git diff --cached --name-only | grep -q "^testdata/fixture\\.json$"; then
  echo "FAIL: testdata/fixture.json not staged"
  exit 1
fi

# Check 7: Staged set contains ONLY .gitignore, scripts/build.sh, testdata/fixture.json (exactly 3 files)
STAGED_COUNT=$(git diff --cached --name-only | wc -l | xargs)
if [ "$STAGED_COUNT" != "3" ]; then
  echo "FAIL: Expected 3 staged files, found $STAGED_COUNT"
  git diff --cached --name-only
  exit 1
fi

# Check 8: tmp.log is NOT staged
if git diff --cached --name-only | grep -q "tmp\\.log"; then
  echo "FAIL: tmp.log should not be staged (decoy file)"
  exit 1
fi

# Check 9: tmp.log is neither staged nor committed (decoy file must not be swept in)
# Source: runbook 830 done_when quote: "exact set of newly-visible files has been enumerated and staged"
# (tmp.log is decoy to test that only ENUMERATED files are staged, not a sweep)
# Regardless of whether tmp.log is ignored or untracked, it must NOT be in the commit
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

echo "PASS: Gitignore task end-state verified"
exit 0
