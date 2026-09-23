#!/bin/bash
# Minimal repo for the write-memory task. The task's real state is the trial VAULT
# (vault-template/, copied to $ENGRAM_VAULT_PATH by the harness); the repo is only the agent's
# working directory and the place the harness commits CLAUDE.md (+ the arm-S write-memory skill).
#
# Unlike curate/please, this task needs recall AND learn deployed as real skills in EVERY arm --
# they are the fixed background (design.md D4/proposal.md: "the eval's shim-only arm keeps
# recall/learn as skills and swaps only write-memory for the runbook"). Only write-memory toggles
# between arm S (skill, via task.json's skill_name/skill_src) and arm R (runbook, task 2.x, not
# built yet). So recall/learn are deployed HERE, unconditionally, from the fixture's own frozen
# copies (encodings/taskWriteMemory/recall-learn/{recall,learn}/SKILL.md) -- never the live
# agent-instructions/ files, so this fixture stays stable while the real skills evolve.
set -euo pipefail
TARGET_DIR="$1"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENCODINGS="$HERE/../../encodings/taskWriteMemory"

mkdir -p "$TARGET_DIR"
cd "$TARGET_DIR"
git init -q -b main
git config user.name "Trial Agent"
git config user.email "trial@example.com"

mkdir -p .claude/skills/recall .claude/skills/learn
cp "$ENCODINGS/recall-learn/recall/SKILL.md" .claude/skills/recall/SKILL.md
cp "$ENCODINGS/recall-learn/learn/SKILL.md" .claude/skills/learn/SKILL.md

cat > README.md <<'README'
# Printer farm shop notes

Working notes for the maker-space's 3D-printer farm. Field knowledge lives in the engram memory
vault, not in this repo.
README
git add -A
git commit -qm "chore: initial printer farm shop notes"
