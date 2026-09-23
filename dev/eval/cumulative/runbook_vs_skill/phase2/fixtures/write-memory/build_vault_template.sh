#!/bin/bash
# Regenerates vault-template/ (the write-memory task's seed vault) with the real `engram learn`.
# Fictional domain: a small maker-space 3D-printer farm -- distinct from curate's beekeeping vault
# so background-vault contamination between fixtures (note 996) cannot occur.
# Usage: build_vault_template.sh   (run from anywhere; overwrites vault-template/)
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
V="$HERE/vault-template"
rm -rf "$V"; mkdir -p "$V"
export ENGRAM_VAULT_PATH="$V"
L() { engram learn "$@" --source "printer farm fixture seed" --position top >/dev/null; }

# --- six existing notes (1-6), each carrying a component/<x> categorical tag: the convention the
# eval nudges the agent's own new writes to follow (task 1.3's tag-flag divergence, see
# TASK-RATIONALE.md). None of these covers ambient cross-drafts causing corner warping. ---
L fact --slug bed-leveling-interval --tag component/bed \
  --situation "leveling the print bed on a photo-etched steel plate before a print run" \
  --subject "the print bed" --predicate "should be releveled" \
  --object "every 15 print-hours, because the steel plate warps slightly with thermal cycling"
L fact --slug petg-nozzle-temp --tag component/hotend \
  --situation "printing PETG filament on the farm's printers" \
  --subject "PETG nozzle temperature" --predicate "should be set" \
  --object "between 230 and 240 C, because PETG strings badly below 225 C"
L feedback --slug filament-humidity-storage --tag component/storage \
  --situation "storing opened filament spools between print jobs" \
  --behavior "left an opened nylon spool on the open shelf overnight" \
  --impact "the next print had visible steam-pop pitting from absorbed moisture" \
  --action "reseal every opened spool in a container with fresh desiccant within 30 minutes of a job finishing"
L fact --slug bed-adhesion-glue --tag component/bed \
  --situation "getting the first layer to stick to the print bed" \
  --subject "first-layer adhesion" --predicate "improves most reliably by using" \
  --object "a thin layer of glue stick on the PEI sheet, reapplied every 3 to 5 prints"
L fact --slug belt-tension-check --tag component/frame \
  --situation "hearing ringing or ghosting artifacts in a finished print" \
  --subject "the X and Y belts" --predicate "should be checked and retensioned" \
  --object "monthly, or immediately if a print shows ghosting near sharp corners"
L feedback --slug duct-cleaning-schedule --tag component/cooling \
  --situation "noticing inconsistent part cooling between printers" \
  --behavior "let the part-cooling duct nozzles cake up with dust for months" \
  --impact "one side of tall prints cooled faster and warped along that edge" \
  --action "blow out every cooling duct with compressed air every two weeks"

engram check --vault "$V" >/dev/null
echo "vault-template built: $(ls "$V"/*.md | wc -l) notes"
