#!/bin/bash
# Regenerates vault-template/ (the recall-escalation scenario's seed vault) with the real
# `engram learn`. Same fictional domain as fixtures/recall-glance/ (community garden co-op) --
# a shared background, distinct new task-prompt/decision.
#
# Unlike glance's vault, this scenario also needs write-memory's REAL promoted runbook note
# (1053.2026-09-22.write-memory-compose-execute-verify) physically present so `engram show
# 1053...` resolves inside a fixture-only trial vault (no real-vault copy for vault_template
# tasks -- same approach learn's own fixture used, see fixtures/learn/build_vault_template.sh
# and its TASK-RATIONALE.md "REAL vault note copied into the fixture" section). Copied
# byte-for-byte from the real vault ($HOME/.local/share/engram/vault, overridable via
# $ENGRAM_REAL_VAULT) as a read-only reference every arm carries unmodified.
# Usage: build_vault_template.sh   (run from anywhere; overwrites vault-template/)
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
V="$HERE/vault-template"
REAL_VAULT="${ENGRAM_REAL_VAULT:-$HOME/.local/share/engram/vault}"
REF_BASENAME="1053.2026-09-22.write-memory-compose-execute-verify"

rm -rf "$V"; mkdir -p "$V"
export ENGRAM_VAULT_PATH="$V"
L() { engram learn "$@" --source "garden co-op fixture seed" --position top >/dev/null; }

# --- note 1: the OLD, soon-to-be-superseded convention. Channel 1 will match this directly on
# the task-prompt's own wording ("which starter mix"). ---
L fact --slug seedling-starter-mix-peat --tag plant/seedlings \
  --situation "starting broccoli and other fall greens seedlings in the greenhouse trays" \
  --subject "the seedling starter mix" --predicate "should be" \
  --object "a peat-based seed starting mix -- it holds moisture evenly through germination"

# --- unrelated/near-topic distractor notes: realistic co-op knowledge, none of it the answer. ---
L fact --slug greenhouse-watering-schedule --tag plant/seedlings \
  --situation "watering seedling trays in the greenhouse before they're transplanted" \
  --subject "seedling trays" --predicate "should be watered" \
  --object "from the bottom (a saturated tray, drained after 20 minutes) to avoid damping-off"
L feedback --slug overwatered-squash-mildew --tag plant/squash \
  --situation "watering the squash beds during a stretch of humid weather" \
  --behavior "watered the squash raised beds overhead every day during a humid week" \
  --impact "powdery mildew bloomed across the squash leaves within days" \
  --action "water squash at the soil line and skip a day when humidity is high"
L fact --slug compost-turn-schedule --tag process/composting \
  --situation "maintaining the co-op's shared compost bin" \
  --subject "the compost bin" --predicate "should be turned" \
  --object "every 2 weeks, to keep the pile aerobic and odor-free"
L fact --slug tool-shed-checkout-log --tag process/tool-shed \
  --situation "borrowing a tool from the co-op's shared tool shed" \
  --subject "anyone borrowing a shed tool" --predicate "must sign" \
  --object "the checkout clipboard and return the tool by end of day"

# --- write-memory's real production runbook, copied byte-for-byte (md + sidecar). ---
if [ ! -f "$REAL_VAULT/$REF_BASENAME.md" ]; then
  echo "FATAL: $REAL_VAULT/$REF_BASENAME.md not found -- set \$ENGRAM_REAL_VAULT to a vault that has it" >&2
  exit 1
fi
cp "$REAL_VAULT/$REF_BASENAME.md" "$V/$REF_BASENAME.md"
cp "$REAL_VAULT/$REF_BASENAME.vec.json" "$V/$REF_BASENAME.vec.json"

engram check --vault "$V" >/dev/null
echo "vault-template built: $(ls "$V"/*.md | wc -l) notes"
