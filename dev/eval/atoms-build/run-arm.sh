#!/usr/bin/env bash
# run-arm.sh <arm-id> <project-dir> <prompt-file> [seed-dir]
# Emits raw/<arm-id>.jsonl (full event stream incl. Skill/tool invocations — T3's
# classification source) and raw/<arm-id>.out (final result text — checkpoint scoring source).
set -euo pipefail
ARM_ID="$1"; PROJ="$2"; PROMPT_FILE="$3"; SEED_DIR="${4:-}"
RAW="$(cd "$(dirname "$0")" && pwd)/raw"; mkdir -p "$RAW"
export ENGRAM_VAULT_PATH="/tmp/oa-build-vault-${ARM_ID}"
mkdir -p "$ENGRAM_VAULT_PATH"
[ -n "$SEED_DIR" ] && cp "$SEED_DIR"/* "$ENGRAM_VAULT_PATH/"
cd "$PROJ"
claude -p "$(cat "$PROMPT_FILE")" \
  --model claude-haiku-4-5 \
  --allowedTools "Bash(engram *) Read Skill" \
  --output-format stream-json --verbose \
  > "$RAW/${ARM_ID}.jsonl" 2>"$RAW/${ARM_ID}.err" || echo "EXIT:$?" >> "$RAW/${ARM_ID}.err"
jq -r 'select(.type=="result") | .result' "$RAW/${ARM_ID}.jsonl" > "$RAW/${ARM_ID}.out" || true
