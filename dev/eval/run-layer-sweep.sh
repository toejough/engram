#!/usr/bin/env bash
#
# Layer-isolation experiment — full sweep (headless claude -p).
#
#   Usage:  bash dev/eval/run-layer-sweep.sh [N_TRIALS] [arm ...]
#   Calib:  bash dev/eval/run-layer-sweep.sh 1 cold l1l2l3
#   Full:   bash dev/eval/run-layer-sweep.sh 5
#   Default: N=1 over all 6 arms.
#
# Self-contained: builds a clean CLAUDE_CONFIG_DIR (only the recall+learn
# skills + your settings + a fresh credential read from your macOS Keychain —
# the SAME source dev/eval's own harness uses via readKeychainCredential), then
# runs one headless contacts build per (arm, trial) under that arm's layer vault.
#
# RUN THIS YOURSELF: the `security find-generic-password` line reads your
# Keychain — that's your authorized action, which is why the agent hands you
# this script instead of reading your credential itself.
set -euo pipefail

N="${1:-1}"; [ $# -gt 0 ] && shift || true
ARMS=("$@"); [ ${#ARMS[@]} -eq 0 ] && ARMS=(cold l1 l2 l3 l1l2 l1l2l3)

REPO="/Users/joe/repos/personal/engram"
HOME_CFG="$HOME/.claude"
CFG="$REPO/dev/eval/.layer-run/cfg"
RESULTS="$REPO/dev/eval/.layer-run"

command -v engram >/dev/null || { echo "engram not on PATH" >&2; exit 1; }
[ -d "$HOME_CFG/skills/recall" ] && [ -d "$HOME_CFG/skills/learn" ] || {
  echo "recall/learn skills not found under $HOME_CFG/skills (run 'engram update')" >&2; exit 1; }

# --- provision a clean, isolated cfg (recall+learn only) ---
rm -rf "$CFG"; mkdir -p "$CFG/skills" "$RESULTS"
cp -R "$HOME_CFG/skills/recall" "$CFG/skills/recall"
cp -R "$HOME_CFG/skills/learn"  "$CFG/skills/learn"
[ -f "$HOME_CFG/settings.json" ] && cp "$HOME_CFG/settings.json" "$CFG/settings.json"
security find-generic-password -s "Claude Code-credentials" -w > "$CFG/.credentials.json"
chmod 600 "$CFG/.credentials.json"
echo "cfg ready: $CFG  (skills: recall, learn; cred: fresh from Keychain)"
echo "arms: ${ARMS[*]}   trials each: $N"
echo

# --- run sweep (sequential: one cfg, no concurrent-session conflicts) ---
export CLAUDE_CFG_DIR="$CFG"
fail=0
for arm in "${ARMS[@]}"; do
  t=1
  while [ "$t" -le "$N" ]; do
    echo "================ arm=$arm trial=$t/$N ================"
    if ! bash "$REPO/dev/eval/run-layer-arm.sh" "$arm" "$t"; then
      echo "  !! arm=$arm trial=$t FAILED (see $RESULTS/${arm}-t${t}.err); continuing"
      fail=$((fail+1))
    fi
    t=$((t+1))
  done
done

echo
echo "sweep complete. results: $RESULTS/   (failures: $fail)"
echo "next: the parent agent scores each build's workspace against contacts-rubric.md"
