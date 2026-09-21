#!/bin/bash
# End-state checks for the curate task: three pending offers seeded in a fictional beekeeping vault
# (7 covered by note 3, 8 near note 2, 9 absent). The outcome the curate skill prescribes:
#   covered (7): note 3 reinforced (--activate -> sidecar last_used set), offer 7 DISCARDED (file gone)
#   near    (8): note 2 amended to carry the offer's extra claim (repeat 3x at 5-day intervals),
#                offer 8 DISCARDED (file gone)
#   absent  (9): pending marker cleared, file KEPT with its content untouched
#   whole vault: no pending offers left, `engram check` passes, no new notes (no `engram learn`),
#                untouched notes 1/4/5/6 byte-identical to the seed.
# Repo-observable only; transcript moves are scored by steps.json. The vault is $ENGRAM_VAULT_PATH
# (the harness's trial env), or <trial_dir>/vault-final for a --rescore of a kept trial.
set -euo pipefail

REPO_DIR="$1"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE="$HERE/vault-template"
V="${ENGRAM_VAULT_PATH:-}"
if [ -z "$V" ] || [ ! -d "$V" ]; then
  V="$(cd "$REPO_DIR/.." && pwd)/vault-final"
fi
if [ ! -d "$V" ]; then
  echo "FAIL: no vault to check (ENGRAM_VAULT_PATH unset/missing and no vault-final next to the repo)"
  exit 1
fi

# Fixture notes are dated 2026-09-20. The shim-only R arm also puts the curate runbook (id 10) and the
# recall/learn/write-memory runbook carrier (ids 1-8, dated 2026-09-14, colliding with fixture ids) in the
# vault; those are the arm's treatment, not fixture notes, so they are excluded from the note-set check
# and never matched by id.
CARRIERS="$( (ls "$HERE"/../../encodings/shim/recall-learn/vault/*.md "$HERE"/../../encodings/taskCurate/Curate-R/vault/*.md 2>/dev/null) | xargs -n1 basename)"
note() { ls "$V"/"$1".2026-09-20.*.md 2>/dev/null | head -n 1; }
sidecar_of() { echo "${1%.md}.vec.json"; }

# Check 1: vault invariants hold.
if ! engram check --vault "$V" >/dev/null 2>&1; then
  echo "FAIL: 'engram check' does not pass on the vault"
  exit 1
fi

# Check 2: no pending offer remains anywhere.
if grep -l '^pending: true$' "$V"/*.md >/dev/null 2>&1; then
  echo "FAIL: pending offer(s) still in the vault: $(grep -l '^pending: true$' "$V"/*.md | xargs -n1 basename | tr '\n' ' ')"
  exit 1
fi

# Check 3: exactly the expected note set (1-6 plus the kept absent offer 9): covered and near
# offers are gone (discarded, not cleared-and-kept) and nothing new was written (no `engram learn`).
EXPECTED="1 2 3 4 5 6 9"
ACTUAL=$(ls "$V"/*.md | xargs -n1 basename | grep -vxF -f <(echo "$CARRIERS") | sed 's/\..*//' | sort -n | tr '\n' ' ' | sed 's/ $//')
if [ "$ACTUAL" != "$EXPECTED" ]; then
  echo "FAIL: vault note ids are [$ACTUAL], expected [$EXPECTED] (offers 7 and 8 must be discarded, 9 kept, nothing new written)"
  exit 1
fi
for id in 7 8; do
  if [ -n "$(note $id)" ] || ls "$V"/$id.2026-09-20.*.vec.json >/dev/null 2>&1; then
    echo "FAIL: offer $id was not fully discarded (note or sidecar remains)"
    exit 1
  fi
done

# Check 4: covered — note 3 keeps its claim intact and was reinforced (sidecar last_used set).
N3="$(note 3)"
if ! grep -q '^action: wait until at least 80 percent of each frame is capped before extracting$' "$N3"; then
  echo "FAIL: note 3 (honey-extraction-moisture) lost or changed its claim"
  exit 1
fi
if ! grep -q '"last_used"' "$(sidecar_of "$N3")"; then
  echo "FAIL: note 3 was not reinforced (no last_used on its sidecar: --activate missing)"
  exit 1
fi

# Check 5: near — note 2 keeps its original claim AND now carries the offer's extra claim.
N2="$(note 2)"
if ! grep -q 'broodless' "$N2" || ! grep -q 'does not penetrate wax cappings' "$N2"; then
  echo "FAIL: note 2 (oxalic-vapor-timing) lost its original claim"
  exit 1
fi
if ! grep -qi 'three times\|3 times\|thrice\|3x\|three treatments' "$N2" \
   || ! grep -qi 'five-day\|5-day\|five days\|5 days\|every five\|every 5' "$N2"; then
  echo "FAIL: note 2 was not amended with the offer's claim (repeat three times at five-day intervals)"
  exit 1
fi

# Check 6: absent — note 9 kept, content untouched, marker gone (Check 2 covers the marker).
N9="$(note 9)"
for needle in 'a swarm trap' 'south-east' 'lemongrass oil' '100 metres'; do
  if ! grep -q "$needle" "$N9"; then
    echo "FAIL: note 9 (swarm-trap-placement) lost or changed its content ('$needle' missing)"
    exit 1
  fi
done

# Check 7: notes nobody had reason to touch (1, 4, 5, 6) are byte-identical to the seed.
for id in 1 4 5 6; do
  if ! cmp -s "$(note $id)" "$TEMPLATE/$(basename "$(note $id)")"; then
    echo "FAIL: untouched note $id was modified"
    exit 1
  fi
done

echo "PASS: covered offer discarded + note 3 reinforced; near offer folded into note 2 + discarded; absent offer accepted; vault clean"
