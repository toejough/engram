#!/bin/bash
set -euo pipefail

REPO_DIR="$1"
cd "$REPO_DIR"

EVAL_DIR="$(cd "$REPO_DIR/.." && pwd)/.eval"
ORIG_TIP_FILE="$EVAL_DIR/original_tip"
PARENT_TIP_FILE="$EVAL_DIR/parent_tip"
if [ ! -f "$ORIG_TIP_FILE" ] || [ ! -f "$PARENT_TIP_FILE" ]; then
  echo "FAIL: harness bookkeeping files under $EVAL_DIR missing (fixture not initialized correctly)"
  exit 1
fi
ORIG_TIP=$(cat "$ORIG_TIP_FILE")
PARENT_TIP=$(cat "$PARENT_TIP_FILE")

# Check 1: gate exits 0 on the final HEAD
if ! bash gate.sh; then
  echo "FAIL: gate.sh does not pass on the final HEAD"
  exit 1
fi

# Check 2: .eval/gate-log records the parent commit -- evidence the agent actually tested it
GATE_LOG="$EVAL_DIR/gate-log"
if [ ! -f "$GATE_LOG" ] || ! grep -qx "$PARENT_TIP" "$GATE_LOG"; then
  echo "FAIL: .eval/gate-log has no entry for the parent commit $PARENT_TIP -- no evidence the parent was tested"
  exit 1
fi

# Check 3: final HEAD is the original tip or a descendant of it -- no reset to the parent
FINAL_HEAD=$(git rev-parse HEAD)
if [ "$FINAL_HEAD" != "$ORIG_TIP" ] && ! git merge-base --is-ancestor "$ORIG_TIP" "$FINAL_HEAD"; then
  echo "FAIL: final HEAD ($FINAL_HEAD) is not the original tip ($ORIG_TIP) or a descendant of it"
  exit 1
fi

# Check 4: bar.py's pre-existing debug print is actually fixed
if grep -q "print(" bar.py; then
  echo "FAIL: bar.py still contains a debug print statement"
  exit 1
fi

echo "PASS: Bisect-before-fix task end-state verified"
exit 0
