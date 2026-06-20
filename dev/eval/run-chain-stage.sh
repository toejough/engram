#!/usr/bin/env bash
#
# One stage of the accumulation chain: a headless build that recalls (tier-capped)
# from the accumulating vault, builds the app, then learns ONLY the arm's tier
# back into the same vault.
#
#   Usage: bash run-chain-stage.sh <L1|L2|L3> <todo|bookmarks|contacts> <vault> <stage 0|1|2> <learn yes|no>
#
# Recall and learn both go through the SKILLS (the cfg carries recall+learn). The
# build is a fresh clean-room claude -p; the vault is threaded by the caller so it
# accumulates across stages.
set -euo pipefail

LAYER="${1:?layer L1|L2|L3}"; APP="${2:?app}"; VAULT="${3:?vault path}"; STAGE="${4:?stage}"; DOLEARN="${5:?yes|no}"
REPO="/Users/joe/repos/personal/engram"
CFG="${CLAUDE_CFG_DIR:-$REPO/dev/eval/.layer-run/cfg}"
MODEL="${MODEL:-claude-sonnet-4-6}"
RESULTS="$REPO/dev/eval/.layer-run/chain"
ENGRAM_DIR="$(dirname "$(command -v engram)")"
mkdir -p "$VAULT" "$RESULTS"
WS="$(mktemp -d "/tmp/chain-${LAYER}-s${STAGE}-${APP}-XXXX")"

# --- app specs (comparable feature complexity; same architecture target) ---
case "$APP" in
  todo)      SPEC="a command-line TODO app: add <text>, list, done <n>, with a priority and a status; persisted to a JSON file";;
  bookmarks) SPEC="a command-line BOOKMARKS manager: add <url>, list, search <q>, delete <n>, tag <n> <tag>; persisted to a JSON file";;
  contacts)  SPEC="a command-line CONTACTS manager: add, list, show <id>, edit <id>, rm <id>, search <q>; persisted to a JSON file";;
  *) echo "unknown app $APP" >&2; exit 1;;
esac

# --- layer-specific learn instruction ---
case "$LAYER" in
  L1) LEARN="Use the /learn skill to capture exactly ONE episode of THIS build (a concrete record of what you built — files, interfaces, patterns). Write nothing else.";;
  L2) LEARN="Use the /learn skill to capture FACTS only (one fact per architecture convention you applied — DI, atomic storage, sentinel errors, table tests, output, etc.). Do NOT write episodes or run L3 synthesis.";;
  L3) LEARN="Use the /learn skill to capture facts AND then run the §6b L3 synthesis (scenario-seeded ADRs over the L2 clusters, update-or-create by nearest_l3 cosine). Do NOT write episodes.";;
esac
[ "$DOLEARN" = "no" ] && LEARN="Do NOT write any memory this stage."

PROMPT="You are a fresh Go developer. Build ${SPEC} from scratch in the current directory (run: go mod init ${APP}).

STEP 1 — consult memory: use your recall skill, querying engram with \`engram query --phrase \"...\"\`. The vault may hold learnings from earlier app builds (or be empty = cold). Read every surfaced note and APPLY every architecture convention you find.

STEP 2 — build the app in this directory and make \`go test ./...\` pass. Keep going until it builds and tests pass.

STEP 3 — ${LEARN}

NON-INTERACTIVE: never stop to ask; continue autonomously."

echo "[chain ${LAYER} s${STAGE} ${APP}] vault=${VAULT} ws=${WS} learn=${DOLEARN}"
cd "$WS"
env CLAUDE_CONFIG_DIR="$CFG" ENGRAM_VAULT_PATH="$VAULT" PATH="$ENGRAM_DIR:$PATH" \
  claude -p "$PROMPT" --output-format json --model "$MODEL" --add-dir "$WS" --permission-mode bypassPermissions \
  > "$RESULTS/${LAYER}-s${STAGE}-${APP}.json" 2> "$RESULTS/${LAYER}-s${STAGE}-${APP}.err" || {
    echo "  STAGE FAILED — see $RESULTS/${LAYER}-s${STAGE}-${APP}.err" >&2; exit 1; }
echo "$WS" > "$RESULTS/${LAYER}-s${STAGE}-${APP}.workspace"
echo "  done. result=$RESULTS/${LAYER}-s${STAGE}-${APP}.json ws=$WS"
