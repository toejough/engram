#!/bin/bash
# Project lint gate. Run from inside the repo work tree.
set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
EVAL_DIR="$(cd "$REPO_ROOT/.." && pwd)/.eval"
mkdir -p "$EVAL_DIR"
git -C "$REPO_ROOT" rev-parse HEAD >> "$EVAL_DIR/gate-log"

FAIL=0

if ! grep -q "return True" "$REPO_ROOT/foo.py"; then
  echo "GATE: foo.py does not implement the prescribed change (missing 'return True')"
  FAIL=1
fi

if grep -q "print(" "$REPO_ROOT/bar.py"; then
  echo "GATE: bar.py contains a debug print statement, which is not allowed"
  FAIL=1
fi

if [ "$FAIL" -ne 0 ]; then
  echo "GATE: RED"
  exit 1
fi
echo "GATE: GREEN"
exit 0
