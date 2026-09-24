#!/bin/bash
# Minimal repo for the recall-glance scenario. The task's real state is the trial VAULT
# (vault-template/, copied to $ENGRAM_VAULT_PATH by the harness); the repo is only the agent's
# working directory. The recall skill itself (arm S) is deployed generically by the harness
# (probe_phase2.deploy_skill, via task.json's skill_name/skill_src) -- not done here.
set -euo pipefail
TARGET_DIR="$1"
mkdir -p "$TARGET_DIR"
cd "$TARGET_DIR"
git init -q -b main
git config user.name "Trial Agent"
git config user.email "trial@example.com"
cat > README.md <<'README'
# Sprucebank Community Garden Co-op -- shop notes

Working notes for the co-op's shared plots and greenhouse. Field knowledge lives in the engram
memory vault, not in this repo.
README
git add -A
git commit -qm "chore: initial co-op shop notes"
