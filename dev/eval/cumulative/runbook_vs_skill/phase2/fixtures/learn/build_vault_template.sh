#!/bin/bash
# Regenerates vault-template/ (the learn task's seed vault) with the real `engram learn`, then
# copies in a read-only reference copy of the REAL production write-memory runbook note
# (1053.2026-09-22.write-memory-compose-execute-verify) so `engram show 1053...` resolves inside
# the trial vault exactly as it would in production (design D2: the learn skill already names
# write-memory's real basename directly, no fixture-placeholder step).
# Fictional domain: a small home-brewing club's field notes -- distinct from curate's beekeeping
# vault and write-memory's 3D-printer-farm vault so background-vault contamination between
# fixtures (note 996) cannot occur.
# Usage: build_vault_template.sh   (run from anywhere; overwrites vault-template/)
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
V="$HERE/vault-template"
REAL_VAULT="${ENGRAM_REAL_VAULT:-$HOME/.local/share/engram/vault}"
NOTE_1053_BASENAME="1053.2026-09-22.write-memory-compose-execute-verify"

rm -rf "$V"; mkdir -p "$V"
export ENGRAM_VAULT_PATH="$V"
L() { engram learn "$@" --source "homebrew club fixture seed" --position top >/dev/null; }

# --- six existing notes (1-6), each carrying a family/<value> categorical tag: the convention the
# eval nudges the agent's own new writes to follow (mirrors write-memory's component/<x> nudge and
# curate's tag conventions). None of these covers an airlock/krausen false-stall reading, or
# Campden-tablet dissipation time -- both new-write topics stay genuinely absent from the seed set. ---
L fact --slug mash-temp-conversion --tag process/mashing \
  --situation "converting a single-infusion mash schedule to a different grain bill" \
  --subject "the strike water temperature" --predicate "should be calculated" \
  --object "using the standard 0.2 factor (grain absorbs about 0.2 F per pound-quart of heat capacity difference), then verified with a thermometer at dough-in"
L fact --slug hop-utilization-boil-time --tag ingredient/hops \
  --situation "calculating expected bitterness (IBU) for a boil addition" \
  --subject "hop alpha-acid utilization" --predicate "increases with" \
  --object "boil time and wort gravity — a 60-minute addition extracts roughly 25-30% utilization at typical gravities, a 10-minute addition well under half that"
L fact --slug yeast-starter-cell-count --tag ingredient/yeast \
  --situation "building a yeast starter for a high-gravity batch" \
  --subject "a 1-liter starter on a stir plate" --predicate "produces roughly" \
  --object "180-200 billion cells from a single fresh liquid yeast pack, enough for one 5-gallon batch under 1.060 OG"
L fact --slug keg-carbonation-psi-chart --tag equipment/kegging \
  --situation "force-carbonating a keg to a target volumes-of-CO2" \
  --subject "keg pressure for 2.4 volumes of CO2 at 38F" --predicate "should be set to" \
  --object "about 12 psi, held for at least a week for full carbonation at serving temperature"
L feedback --slug star-san-no-rinse-dilution --tag sanitation/equipment \
  --situation "diluting Star San sanitizer for no-rinse use on brewing equipment" \
  --behavior "eyeballed the dilution instead of measuring it, using a much stronger mix than the label called for" \
  --impact "left a visible foam residue in the fermenter that took an extra rinse cycle to clear" \
  --action "measure 1 oz of Star San per 5 gallons of water (roughly 1:640) with a syringe, not by eye"
L feedback --slug secondary-fermenter-headspace --tag equipment/fermenter \
  --situation "racking a batch to a secondary fermenter for extended conditioning" \
  --behavior "racked into a secondary vessel that left several inches of headspace above the beer" \
  --impact "the batch picked up a faint cardboard, oxidized flavor over three weeks of conditioning" \
  --action "match the secondary vessel size to the batch volume so headspace stays under an inch, or skip secondary entirely for anything conditioning less than a month"

# --- read-only reference copy of the real production write-memory runbook, so `engram show
# 1053...` resolves inside this fixture's own trial vault exactly as it does in production. ---
if [ ! -f "$REAL_VAULT/$NOTE_1053_BASENAME.md" ]; then
  echo "FATAL: real vault write-memory runbook not found at $REAL_VAULT/$NOTE_1053_BASENAME.md" >&2
  exit 1
fi
cp "$REAL_VAULT/$NOTE_1053_BASENAME.md" "$V/$NOTE_1053_BASENAME.md"
cp "$REAL_VAULT/$NOTE_1053_BASENAME.vec.json" "$V/$NOTE_1053_BASENAME.vec.json"

engram check --vault "$V" >/dev/null
echo "vault-template built: $(ls "$V"/*.md | wc -l) notes"
