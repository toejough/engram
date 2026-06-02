#!/usr/bin/env bash
#
# Layer-isolation experiment — single-arm contacts build runner.
#
#   Usage:  bash dev/eval/run-layer-arm.sh <arm> [trial]
#   Arms:   cold | l1 | l2 | l3 | l1l2 | l1l2l3
#
# Runs ONE headless `claude -p` contacts build under the chosen layer vault.
# The build agent consults memory via the recall skill (against the arm's
# vault), then builds a contacts CLI. It does NOT write memory (read-only arm).
#
# Requires a clean CLAUDE_CONFIG_DIR holding valid creds + ONLY the recall/learn
# skills (default: the surviving epitest cfg). engram must be on PATH so the
# nested recall skill can run `engram query` against ENGRAM_VAULT_PATH.
#
# NOTE: uses --permission-mode bypassPermissions so the non-interactive build
# agent can edit files and run go/targ without prompts. That capability must be
# authorized by the user (Bash permission rule), which is why the parent agent
# cannot launch this directly.
set -euo pipefail

ARM="${1:?usage: run-layer-arm.sh <arm> [trial]   (arm = cold|l1|l2|l3|l1l2|l1l2l3)}"
TRIAL="${2:-1}"

REPO="/Users/joe/repos/personal/engram"
VAULTS="$REPO/dev/eval/testdata/layer-vaults"
VAULT="$VAULTS/$ARM"
CFG="${CLAUDE_CFG_DIR:-/tmp/epitest/cfg}"
MODEL="${MODEL:-claude-sonnet-4-6}"
RESULTS="$REPO/dev/eval/.layer-run"

[ -d "$VAULT/Permanent" ] || { echo "no vault at $VAULT" >&2; exit 1; }
[ -f "$CFG/.credentials.json" ] || { echo "no creds at $CFG (set CLAUDE_CFG_DIR)" >&2; exit 1; }
command -v engram >/dev/null || { echo "engram not on PATH" >&2; exit 1; }

ENGRAM_DIR="$(dirname "$(command -v engram)")"
PROMPT="$(cat "$VAULTS/contacts-build-prompt.txt")"
WS="$(mktemp -d "/tmp/layer-${ARM}-${TRIAL}-XXXXXX")"
mkdir -p "$RESULTS"

echo "[arm=$ARM trial=$TRIAL] vault=$VAULT"
echo "  workspace=$WS"
echo "  cfg=$CFG  model=$MODEL"

cd "$WS"
env CLAUDE_CONFIG_DIR="$CFG" \
    ENGRAM_VAULT_PATH="$VAULT" \
    PATH="$ENGRAM_DIR:$PATH" \
  claude -p "$PROMPT" \
    --output-format json \
    --model "$MODEL" \
    --add-dir "$WS" \
    --permission-mode bypassPermissions \
  > "$RESULTS/${ARM}-${TRIAL}.json" \
  2> "$RESULTS/${ARM}-${TRIAL}.err" || {
    echo "  RUN FAILED — see $RESULTS/${ARM}-${TRIAL}.err" >&2; exit 1; }

# Record where the built app + transcript live, for scoring.
echo "$WS" > "$RESULTS/${ARM}-${TRIAL}.workspace"
SID="$(python3 -c "import json,sys;print(json.load(open(sys.argv[1])).get('session_id',''))" "$RESULTS/${ARM}-${TRIAL}.json" 2>/dev/null || true)"
echo "  done. result=$RESULTS/${ARM}-${TRIAL}.json  session=$SID  ws=$WS"
