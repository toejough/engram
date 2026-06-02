#!/usr/bin/env bash
#
# Resume one arm's build session with review feedback = one convergence round.
#
#   Usage: bash run-layer-resume.sh <arm> <prev_round> <new_round> <feedback_file>
#   e.g.   bash run-layer-resume.sh cold r1 r2 /tmp/cold-fb.txt
#
# Chains rounds in the SAME workspace + session: reads the prior round's
# session_id + workspace, sends the (fixed, spec-free) feedback, and records the
# new round's result. The feedback file content is the concatenation of the
# failing rubric items' verbatim hints from review-hints.md — identical wording
# across arms (constancy is the control).
set -euo pipefail

ARM="${1:?arm}"; PREV="${2:?prev round}"; NEW="${3:?new round}"; FB_FILE="${4:?feedback file}"
REPO="/Users/joe/repos/personal/engram"
VAULT="$REPO/dev/eval/testdata/layer-vaults/$ARM"
CFG="${CLAUDE_CFG_DIR:-/tmp/epitest/cfg}"
MODEL="${MODEL:-claude-sonnet-4-6}"
RESULTS="$REPO/dev/eval/.layer-run"
PREV_JSON="$RESULTS/${ARM}-${PREV}.json"

[ -f "$PREV_JSON" ]        || { echo "no prior result: $PREV_JSON" >&2; exit 1; }
[ -f "$FB_FILE" ]          || { echo "no feedback file: $FB_FILE" >&2; exit 1; }
SID="$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['session_id'])" "$PREV_JSON")"
WS="$(cat "$RESULTS/${ARM}-${PREV}.workspace")"
[ -d "$WS" ]               || { echo "workspace gone: $WS" >&2; exit 1; }
FB="$(cat "$FB_FILE")"
ENGRAM_DIR="$(dirname "$(command -v engram)")"

echo "[resume arm=$ARM ${PREV}->${NEW}] sid=$SID ws=$WS"
cd "$WS"
env CLAUDE_CONFIG_DIR="$CFG" ENGRAM_VAULT_PATH="$VAULT" PATH="$ENGRAM_DIR:$PATH" \
  claude --resume "$SID" -p "$FB" \
    --output-format json --model "$MODEL" --add-dir "$WS" --permission-mode bypassPermissions \
  > "$RESULTS/${ARM}-${NEW}.json" 2> "$RESULTS/${ARM}-${NEW}.err" || {
    echo "  RESUME FAILED — see $RESULTS/${ARM}-${NEW}.err" >&2; exit 1; }

echo "$WS" > "$RESULTS/${ARM}-${NEW}.workspace"
NEWSID="$(python3 -c "import json,sys;print(json.load(open(sys.argv[1])).get('session_id',''))" "$RESULTS/${ARM}-${NEW}.json" 2>/dev/null || true)"
echo "  done. result=$RESULTS/${ARM}-${NEW}.json  newsid=$NEWSID"
