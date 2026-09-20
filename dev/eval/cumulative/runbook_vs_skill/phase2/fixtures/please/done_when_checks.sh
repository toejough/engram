#!/bin/bash
# End-state checks for the please task (rename tally's `--out` flag to `--output`, end-to-end).
# Repo-observable outcomes of please's workflow only: the rename landed everywhere the flag is
# echoed (the doc-surface enumeration grep's payoff), the plan was committed BEFORE any code
# change and carries a per-file disposition list (Step 3), planning artifacts were deleted and
# the work committed (Step 6). Transcript-observable behavior (gates dispatched, recall/learn
# bracket, lessons-audit report) is scored by steps.json, not here.
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
if ! git cat-file -e "$ORIG_TIP^{commit}" 2>/dev/null; then
  echo "FAIL: recorded original_tip $ORIG_TIP is not a commit in this repo"
  exit 1
fi

# The surface files that echo the flag. Every one must be updated (CHANGELOG.md is the historical
# record and is checked separately).
# tests/test_tally.py is deliberately NOT a surface file: a correct landing adds a test asserting the old
# spelling is rejected, which must mention `--out`; a test still invoking `--out` fails pytest above.
SURFACE_FILES=(tally.py README.md docs/usage.md docs/architecture.md completions/tally.bash)
# Old-flag literal: `--out` NOT followed by a letter (so `--output` never matches).
OLD_FLAG_RE='--out([^a-zA-Z]|$)'

# Check 1: behavior. --output writes the report to a file and nothing to stdout; the old spelling
# is a hard error (the parser sets allow_abbrev=False, so `--out` cannot silently prefix-match).
SCRATCH=$(mktemp -d)
trap 'rm -rf "$SCRATCH"' EXIT
if ! python3 tally.py sample.txt --output "$SCRATCH/report.txt" > "$SCRATCH/stdout.txt" 2>"$SCRATCH/stderr.txt"; then
  echo "FAIL: 'python3 tally.py sample.txt --output <path>' did not exit 0: $(cat "$SCRATCH/stderr.txt")"
  exit 1
fi
if [ -s "$SCRATCH/stdout.txt" ]; then
  echo "FAIL: --output printed to stdout (expected the report only in the file)"
  exit 1
fi
if [ "$(head -n 1 "$SCRATCH/report.txt")" != "$(printf 'the\t3')" ]; then
  echo "FAIL: --output report's first line is not 'the<TAB>3'"
  exit 1
fi
if python3 tally.py sample.txt --out "$SCRATCH/old.txt" >/dev/null 2>&1; then
  echo "FAIL: the old '--out' spelling is still accepted (hard rename required, no alias)"
  exit 1
fi

# Check 2: the test suite passes, and the tests exercise --output and no longer the old flag.
if ! python3 -m pytest -q >"$SCRATCH/pytest.txt" 2>&1; then
  echo "FAIL: pytest did not pass"
  cat "$SCRATCH/pytest.txt"
  exit 1
fi
if ! grep -q -- '--output' tests/test_tally.py; then
  echo "FAIL: tests/test_tally.py does not exercise --output"
  exit 1
fi

# Check 3: no surface file still echoes the old flag (docs, diagram label, completion script,
# code comment, help text) -- the whole point of the doc-surface enumeration grep.
STALE=()
for f in "${SURFACE_FILES[@]}"; do
  if [ ! -f "$f" ]; then
    echo "FAIL: expected surface file $f is missing"
    exit 1
  fi
  if grep -Eq -- "$OLD_FLAG_RE" "$f"; then
    STALE+=("$f")
  fi
done
if [ "${#STALE[@]}" -gt 0 ]; then
  echo "FAIL: stale '--out' echoes remain in: ${STALE[*]}"
  exit 1
fi
# ...and nowhere else in the tracked tree either, other than the historical CHANGELOG.
OTHER_STALE=$(git grep -lE -e "$OLD_FLAG_RE" -- . ':!CHANGELOG.md' ':!tests/test_tally.py' || true)
if [ -n "$OTHER_STALE" ]; then
  echo "FAIL: stale '--out' echoes remain in tracked files: $OTHER_STALE"
  exit 1
fi
# The docs must actually carry the new flag (not merely have dropped the old one).
for f in README.md docs/usage.md docs/architecture.md completions/tally.bash; do
  if ! grep -q -- '--output' "$f"; then
    echo "FAIL: $f does not mention --output"
    exit 1
  fi
done

# Check 4: CHANGELOG keeps every original line verbatim (history is a record, not a surface to
# scrub) and gains a new entry that names --output.
while IFS= read -r line; do
  if ! grep -Fxq -- "$line" CHANGELOG.md; then
    echo "FAIL: CHANGELOG.md lost or altered a historical line: $line"
    exit 1
  fi
done < <(git show "$ORIG_TIP:CHANGELOG.md")
if ! grep -q -- '--output' CHANGELOG.md; then
  echo "FAIL: CHANGELOG.md has no entry naming --output (repo norm: every user-visible change gets one)"
  exit 1
fi

# Check 5: the work is committed and the tree is clean; planning artifacts are gone.
if [ "$(git rev-parse HEAD)" = "$ORIG_TIP" ]; then
  echo "FAIL: HEAD is still the original tip -- nothing was committed"
  exit 1
fi
if [ -n "$(git status --porcelain)" ]; then
  echo "FAIL: working tree is not clean:"
  git status --porcelain
  exit 1
fi
LEFTOVER_PLANS=$(git ls-files | grep -Ei '(^|/)(plan|design|spec|todo|scratch)[^/]*\.(md|txt)$|(^|/)docs/(plans|superpowers)/' || true)
if [ -n "$LEFTOVER_PLANS" ]; then
  echo "FAIL: planning artifacts still tracked at HEAD: $LEFTOVER_PLANS"
  exit 1
fi

# Check 6 (Step 3): the plan was committed BEFORE the first commit that touches tally.py, and the
# plan carries a per-file disposition list -- every surface file named on a line with a
# disposition word (keep / update / rewrite / N/A). The plan file was deleted at Step 6, so read
# it back from history: files added after ORIG_TIP that are not tracked at HEAD.
COMMITS=$(git rev-list --reverse "$ORIG_TIP..HEAD")
FIRST_CODE_COMMIT=""
for c in $COMMITS; do
  if git diff-tree --no-commit-id --name-only -r "$c" | grep -qx 'tally.py'; then
    FIRST_CODE_COMMIT="$c"
    break
  fi
done
if [ -z "$FIRST_CODE_COMMIT" ]; then
  echo "FAIL: no commit after the original tip modifies tally.py"
  exit 1
fi
PLAN_TEXT=""
PLAN_COMMIT_FOUND=0
for c in $COMMITS; do
  if [ "$c" = "$FIRST_CODE_COMMIT" ]; then
    break
  fi
  while IFS= read -r added; do
    [ -z "$added" ] && continue
    if git cat-file -e "HEAD:$added" 2>/dev/null; then
      continue  # still tracked at HEAD: an ordinary repo file, not a deleted plan artifact
    fi
    PLAN_COMMIT_FOUND=1
    PLAN_TEXT+=$'\n'"$(git show "$c:$added")"
  done < <(git diff-tree --no-commit-id --name-only -r --diff-filter=A "$c")
done
if [ "$PLAN_COMMIT_FOUND" -ne 1 ]; then
  echo "FAIL: no plan file was committed (and later deleted) before the first tally.py change"
  exit 1
fi
for f in README.md docs/usage.md docs/architecture.md completions/tally.bash; do
  if ! grep -F -- "$f" <<<"$PLAN_TEXT" | grep -Eiq '(^|[^a-z])(keep|update|rewrite|n/a)([^a-z]|$)'; then
    echo "FAIL: committed plan has no disposition line (keep/update/rewrite/N/A) for $f"
    exit 1
  fi
done

echo "PASS: please task end-state verified"
exit 0
