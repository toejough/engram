#!/bin/bash
# Regenerates vault-template/ (the recall-glance scenario's seed vault) with the real
# `engram learn`. Fictional domain: a small community garden co-op -- distinct from
# write-memory's 3D-printer farm, learn's home-brewing club, and curate's beekeeping vault
# (note 996's contamination guard).
# Usage: build_vault_template.sh   (run from anywhere; overwrites vault-template/)
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
V="$HERE/vault-template"
rm -rf "$V"; mkdir -p "$V"
export ENGRAM_VAULT_PATH="$V"
L() { engram learn "$@" --source "garden co-op fixture seed" --position top >/dev/null; }

# --- note 1: an OLDER theory about the dark leaf spotting (later corrected by note 2 below). ---
L fact --slug tomato-leaf-spotting-splashback --tag plant/tomato \
  --situation "diagnosing dark spots appearing on tomato leaves in the raised beds" \
  --subject "dark leaf spotting on tomato leaves" --predicate "is caused by" \
  --object "nutrient and soil splash-back during overhead watering; switch to soaker hoses at the soil line to avoid it"

# --- note 2: a NEWER feedback note that corrects note 1 (recency-weight / reversal-cue test:
# "assumed X... but Y instead... it's actually Z, not X"). This is the candidate a glance pass
# must read and weight correctly (Step 2.5A/B) to answer the task-prompt right. ---
L feedback --slug tomato-leaf-spotting-blight-correction --tag plant/tomato \
  --situation "dark leaf spotting on tomato leaves keeps spreading after switching to soaker hoses" \
  --behavior "assumed the dark leaf spotting was splash-back and switched Plot 14 to soaker hoses" \
  --impact "the spotting spread across the bed instead of stopping -- it was never splash-back" \
  --action "it is early blight (a fungal disease), not splash-back -- at the first dark spots, remove the affected leaves and apply a copper fungicide within a day; soaker hoses alone do not fix it"

# --- unrelated/near-topic distractor notes: realistic co-op knowledge, none of it the answer. ---
L fact --slug blossom-end-rot-watering --tag plant/tomato \
  --situation "tomato fruit developing a sunken dark patch on the blossom end" \
  --subject "blossom-end rot" --predicate "is caused by" \
  --object "irregular watering, not a disease -- keep a consistent watering schedule to prevent it"
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

engram check --vault "$V" >/dev/null
echo "vault-template built: $(ls "$V"/*.md | wc -l) notes"
