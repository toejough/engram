#!/bin/bash
# Minimal repo for the learn task. The task's real state is the trial VAULT (vault-template/,
# copied to $ENGRAM_VAULT_PATH by the harness); the repo is only the agent's working directory and
# the place the harness commits CLAUDE.md (+ the learn skill, arm S). Unlike write-memory's own
# fixture, this task needs no OTHER skill deployed as fixed background -- `learn` is directly
# user/shim-triggered (curate's pattern), not a worker reached by another skill's name, so only
# `learn` itself toggles between arm S (skill, via task.json's skill_name/skill_src) and arm R
# (runbook, task 2.x, not built yet).
set -euo pipefail
TARGET_DIR="$1"
mkdir -p "$TARGET_DIR"
cd "$TARGET_DIR"
git init -q -b main
git config user.name "Trial Agent"
git config user.email "trial@example.com"
cat > README.md <<'README'
# Homebrew club field notes

Working notes for the club's shared brew days. Field knowledge lives in the engram memory vault,
not in this repo.
README
git add -A
git commit -qm "chore: initial homebrew club field notes"
