#!/bin/bash
set -euo pipefail

REPO_DIR="$1"
cd "$REPO_DIR"

EVAL_DIR="$(cd "$REPO_DIR/.." && pwd)/.eval"
ORIG_TIP_FILE="$EVAL_DIR/original_tip"
if [ ! -f "$ORIG_TIP_FILE" ]; then
  echo "FAIL: harness bookkeeping file under $EVAL_DIR missing (fixture not initialized correctly)"
  exit 1
fi
ORIG_TIP=$(cat "$ORIG_TIP_FILE")
HEAD_SHA=$(git rev-parse HEAD)

is_test_file() {
  case "$1" in
    test_*.py|*/test_*.py|*_test.py|*/*_test.py) return 0 ;;
    *) return 1 ;;
  esac
}

cleanup() {
  git worktree remove --force "$SCRATCH_REF" >/dev/null 2>&1 || true
  git worktree remove --force "$SCRATCH_HIST" >/dev/null 2>&1 || true
  rm -rf "$SCRATCH_REF" "$SCRATCH_HIST"
}
SCRATCH_REF=$(mktemp -d)
SCRATCH_HIST=$(mktemp -d)
trap cleanup EXIT

# Check 1: the final implementation passes a hidden reference test, run against a scratch
# worktree checked out at HEAD -- never against the trial repo's own working tree.
git worktree add -q --detach "$SCRATCH_REF" "$HEAD_SHA"
cat > "$SCRATCH_REF/test_slugify_reference.py" <<'PYEOF'
from slugify import slugify


def test_lowercases():
    assert slugify("Hello") == "hello"


def test_replaces_non_alnum_runs_with_single_hyphen():
    assert slugify("Hello, World!!") == "hello-world"


def test_trims_leading_and_trailing_hyphens():
    assert slugify("--Hello--") == "hello"
    assert slugify("!!!Wow!!!") == "wow"


def test_collapses_mixed_separator_runs():
    assert slugify("foo   bar___baz") == "foo-bar-baz"
PYEOF
if ! ( cd "$SCRATCH_REF" && python3 -m pytest -q test_slugify_reference.py ); then
  echo "FAIL: final slugify implementation does not pass the hidden reference test"
  exit 1
fi

# Check 2: at least two new commits since the fixture's baseline tip touch a .py file. The
# harness's own "add project config" commit (CLAUDE.md only) never touches a .py file and is
# never counted.
mapfile -t NEW_COMMITS < <(git rev-list --reverse "$ORIG_TIP..$HEAD_SHA")
RELEVANT_COMMITS=()
for c in "${NEW_COMMITS[@]}"; do
  files=$(git show --name-only --pretty=format: "$c" | grep -v '^$' || true)
  if echo "$files" | grep -q '\.py$'; then
    RELEVANT_COMMITS+=("$c")
  fi
done
if [ "${#RELEVANT_COMMITS[@]}" -lt 2 ]; then
  echo "FAIL: fewer than 2 new commits touch a .py file (found ${#RELEVANT_COMMITS[@]})"
  exit 1
fi

# Check 3: no relevant commit touches both a test file and a non-test .py file.
for c in "${RELEVANT_COMMITS[@]}"; do
  py_files=$(git show --name-only --pretty=format: "$c" | grep '\.py$' || true)
  has_test=0
  has_impl=0
  while IFS= read -r f; do
    [ -z "$f" ] && continue
    if is_test_file "$f"; then has_test=1; else has_impl=1; fi
  done <<< "$py_files"
  if [ "$has_test" -eq 1 ] && [ "$has_impl" -eq 1 ]; then
    echo "FAIL: commit $c touches both test and implementation .py files"
    exit 1
  fi
done

# Check 4: the FIRST relevant commit touches ONLY test file(s), and pytest FAILS when checked
# out there -- proof the test was written before the implementation (RED before GREEN).
FIRST="${RELEVANT_COMMITS[0]}"
first_py_files=$(git show --name-only --pretty=format: "$FIRST" | grep '\.py$' || true)
all_test=1
while IFS= read -r f; do
  [ -z "$f" ] && continue
  is_test_file "$f" || all_test=0
done <<< "$first_py_files"
if [ "$all_test" -ne 1 ]; then
  echo "FAIL: first commit touching a .py file ($FIRST) does not touch ONLY test file(s)"
  exit 1
fi

git worktree add -q --detach "$SCRATCH_HIST" "$FIRST"
set +e
( cd "$SCRATCH_HIST" && python3 -m pytest -q ) >/dev/null 2>&1
FIRST_RC=$?
set -e
if [ "$FIRST_RC" -eq 0 ]; then
  echo "FAIL: pytest PASSES at the first test-only commit ($FIRST) -- expected RED before the implementation existed"
  exit 1
fi

# Check 5: the project's own pytest suite passes at final HEAD (GREEN after implementation).
if ! python3 -m pytest -q; then
  echo "FAIL: project's own pytest suite does not pass at final HEAD"
  exit 1
fi

echo "PASS: TDD-order task end-state verified"
exit 0
