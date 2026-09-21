#!/bin/bash
# Regenerates vault-template/ (the curate task's seed vault) with the real `engram learn`, then
# marks the three offer notes `pending: true` (only a served write sets that; no CLI flag does).
# Usage: build_vault_template.sh   (run from anywhere; overwrites vault-template/)
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
V="$HERE/vault-template"
rm -rf "$V"; mkdir -p "$V"
export ENGRAM_VAULT_PATH="$V"
L() { engram learn "$@" --source "apiary fixture seed" --position top >/dev/null; }

# --- six existing notes (1-6) ---
L fact --slug summer-inspection-interval --tag vocab/apiary-routine \
  --situation "planning routine hive inspections during the summer swarm season" \
  --subject "a hive in swarm season" --predicate "should be inspected" \
  --object "every 7 to 10 days, because queen cells are capped about 9 days after they are started"
L fact --slug oxalic-vapor-timing --tag vocab/varroa-treatment \
  --situation "treating a colony for varroa mites with oxalic acid vapor" \
  --subject "oxalic acid vapor treatment" --predicate "should only be applied" \
  --object "when the colony is broodless (late autumn or winter), because the vapor does not penetrate wax cappings"
L feedback --slug honey-extraction-moisture --tag vocab/honey-harvest \
  --situation "deciding when to pull honey frames for extraction" \
  --behavior "pulled frames that were mostly uncapped" \
  --impact "honey moisture was above 18.6 percent and the batch fermented in the jar" \
  --action "wait until at least 80 percent of each frame is capped before extracting"
L fact --slug queen-marking-color --tag vocab/apiary-routine \
  --situation "marking a newly installed queen with paint" \
  --subject "queen paint color" --predicate "follows" \
  --object "the international year code: blue for 2026, red for 2027"
L feedback --slug smoker-fuel --tag vocab/apiary-routine \
  --situation "lighting the smoker before opening a hive" \
  --behavior "burned scraps of treated lumber as smoker fuel" \
  --impact "acrid chemical smoke agitated the colony and tainted the wax" \
  --action "burn only untreated burlap, pine needles, or dry cardboard"
L fact --slug site-b-flood-line --tag vocab/apiary-sites \
  --situation "moving hives between apiary sites in autumn" \
  --subject "apiary site B" --predicate "floods below" \
  --object "the white marker post, so hives stay above it from October to March"

# --- three offers (7-9), written then marked pending ---
# 7 COVERED: restates note 3 (honey-extraction-moisture) in different words, nothing omitted.
L feedback --slug spun-uncapped-frames --tag vocab/honey-harvest \
  --situation "about to spin out supers after a short nectar flow" \
  --behavior "extracted frames that were mostly uncapped because the calendar said harvest weekend" \
  --impact "the honey tested over 18.6 percent water and began to ferment" \
  --action "confirm roughly four fifths of the comb surface is capped before spinning any frame"
# 8 NEAR: same topic as note 2 (oxalic-vapor-timing) but adds a claim it omits (repeat dosing).
L fact --slug oxalic-vapor-repeat-dosing --tag vocab/varroa-treatment \
  --situation "running an oxalic acid vapor treatment on a broodless colony" \
  --subject "a single oxalic acid vapor dose" --predicate "misses mites under late-emerging brood, so the treatment should be repeated" \
  --object "three times at five-day intervals"
# 9 ABSENT: nothing in notes 1-6 addresses bait hives.
L fact --slug swarm-trap-placement --tag vocab/swarm-control \
  --situation "setting out a bait hive to catch a spring swarm" \
  --subject "a swarm trap" --predicate "attracts the most scout bees when hung" \
  --object "three to four metres up a tree, facing south-east, about 100 metres from the home apiary, baited with lemongrass oil"

for f in "$V"/7.*.md "$V"/8.*.md "$V"/9.*.md; do
  # insert the marker right after the tier: line of the frontmatter
  sed -i.bak 's/^tier: L2$/tier: L2\npending: true/' "$f"; rm -f "$f.bak"
done
engram check --vault "$V" >/dev/null
echo "vault-template built: $(ls "$V"/*.md | wc -l) notes, $(grep -l '^pending: true$' "$V"/*.md | wc -l) pending"
