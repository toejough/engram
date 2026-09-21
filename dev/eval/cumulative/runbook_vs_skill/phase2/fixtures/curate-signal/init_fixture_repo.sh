#!/bin/bash
# Minimal repo for the curate task. The task's real state is the trial VAULT (vault-template/,
# copied to $ENGRAM_VAULT_PATH by the harness); the repo is only the agent's working directory and
# the place the harness commits CLAUDE.md (+ the skill, arm S).
set -euo pipefail
TARGET_DIR="$1"
mkdir -p "$TARGET_DIR"
cd "$TARGET_DIR"
git init -q -b main
git config user.name "Trial Agent"
git config user.email "trial@example.com"
cat > README.md <<'README'
# Apiary logbook

Working notes for the co-op's three apiary sites. Field knowledge lives in the engram memory vault,
not in this repo.
README
git add -A
git commit -qm "chore: initial apiary logbook"
