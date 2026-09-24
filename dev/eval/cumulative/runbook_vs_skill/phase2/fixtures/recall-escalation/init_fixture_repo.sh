#!/bin/bash
# Minimal repo for the recall-escalation scenario. The task's real state is the trial VAULT
# (vault-template/, copied to $ENGRAM_VAULT_PATH by the harness); the repo is only the agent's
# working directory, the place the harness commits CLAUDE.md (+ the recall skill for arm S, via
# probe_phase2.deploy_skill / task.json's skill_name/skill_src -- NOT done here), and -- unique to
# this scenario -- the source of a "recent activity" markdown doc that the agent's own Step 0.5
# sweep (`engram ingest --auto`) chunks into the recency channel (Channel 2). That doc states a
# brand-new co-op standard (switch from peat to coconut-coir starter mix) nowhere near strongly
# enough matched by the task's own query phrases to enter Channel 1 (verified empirically against
# the real binary -- see TASK-RATIONALE.md's "Channel-2 placement" section); it only surfaces via
# pure ingest recency, which is exactly the C5 scenario recall's own SKILL.md describes (glance
# surfaces a Channel-2 standard but doesn't elevate it to a requirement unless it escalates to
# deep).
set -euo pipefail
TARGET_DIR="$1"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "$TARGET_DIR"
cd "$TARGET_DIR"
git init -q -b main
git config user.name "Trial Agent"
git config user.email "trial@example.com"

mkdir -p docs
cp "$HERE/docs/co-op-notes-2026-09-22.md" docs/co-op-notes-2026-09-22.md

cat > README.md <<'README'
# Sprucebank Community Garden Co-op -- shop notes

Working notes for the co-op's shared plots and greenhouse. Field knowledge lives in the engram
memory vault, not in this repo. `docs/` holds meeting notes and other working documents.
README
git add -A
git commit -qm "chore: initial co-op shop notes"
