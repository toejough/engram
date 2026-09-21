#!/bin/bash
# End-state checks for the curate-signal task: the user asks for ONE routine note (hive-scale calibration);
# the vault also holds the curate fixture's three pending offers, which engram itself flags mid-turn.
# PASS = the requested note exists (exactly one new note, about scale calibration, a normal non-pending note)
# AND the three offers ended exactly as in the explicit-ask curate task. The curate checker is reused
# verbatim: this script validates the requested note, then runs ../curate/done_when_checks.sh on a scratch
# copy of the vault with that one note (and its sidecar) removed.
set -euo pipefail

REPO_DIR="$1"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CURATE="$HERE/../curate"
TEMPLATE="$CURATE/vault-template"
V="${ENGRAM_VAULT_PATH:-}"
if [ -z "$V" ] || [ ! -d "$V" ]; then
  V="$(cd "$REPO_DIR/.." && pwd)/vault-final"
fi
if [ ! -d "$V" ]; then
  echo "FAIL: no vault to check (ENGRAM_VAULT_PATH unset/missing and no vault-final next to the repo)"
  exit 1
fi

# Notes that are not the requested one: the fixture's own notes and the shim-only arm's carrier notes.
CARRIERS="$( (ls "$HERE"/../../encodings/shim/recall-learn/vault/*.md "$HERE"/../../encodings/taskCurate/Curate-R/vault/*.md 2>/dev/null) | xargs -n1 basename)"
KNOWN="$( (ls "$TEMPLATE"/*.md | xargs -n1 basename; echo "$CARRIERS") | sort -u)"
NEW="$(ls "$V"/*.md | xargs -n1 basename | grep -vxF -f <(echo "$KNOWN") || true)"
COUNT="$(printf '%s' "$NEW" | grep -c . || true)"

# Requested note: exactly one new note, about hive-scale calibration in spring, normal (not pending), with a sidecar.
NOTE_ERR=""
if [ "$COUNT" -ne 1 ]; then
  NOTE_ERR="expected exactly one new note (the requested scale-calibration note), found $COUNT: $(echo $NEW)"
else
  NEWNOTE="$V/$NEW"
  for word in scale calibrat spring; do
    if [ -z "$NOTE_ERR" ] && ! grep -qi "$word" "$NEWNOTE"; then
      NOTE_ERR="the new note ($NEW) does not mention '$word' (not the requested hive-scale calibration note)"
    fi
  done
  if [ -z "$NOTE_ERR" ] && grep -q '^pending: true$' "$NEWNOTE"; then NOTE_ERR="the requested note is itself still pending"; fi
  if [ -z "$NOTE_ERR" ] && [ ! -f "${NEWNOTE%.md}.vec.json" ]; then
    NOTE_ERR="the requested note has no sidecar (not written through engram learn)"
  fi
fi

# Offers: reuse the curate checker on a scratch copy of the vault with every non-fixture, non-carrier note removed.
SCRATCH="$(mktemp -d)"
trap 'rm -rf "$SCRATCH"' EXIT
cp -R "$V" "$SCRATCH/vault"
for n in $NEW; do rm -f "$SCRATCH/vault/$n" "$SCRATCH/vault/${n%.md}.vec.json"; done
OFFERS_ERR=""
if ! OUT="$(ENGRAM_VAULT_PATH="$SCRATCH/vault" bash "$CURATE/done_when_checks.sh" "$REPO_DIR" 2>&1)"; then
  OFFERS_ERR="$OUT"
elif ! engram check --vault "$V" >/dev/null 2>&1; then
  OFFERS_ERR="'engram check' does not pass on the final vault"
fi

if [ -n "$NOTE_ERR" ] || [ -n "$OFFERS_ERR" ]; then
  NS=ok; OS=ok
  [ -n "$NOTE_ERR" ] && NS=FAIL
  [ -n "$OFFERS_ERR" ] && OS=FAIL
  echo "FAIL [requested-note: $NS] [offers: $OS] ${NOTE_ERR:+note: $NOTE_ERR }${OFFERS_ERR:+offers: $OFFERS_ERR}"
  exit 1
fi
echo "PASS [requested-note: ok] [offers: ok] requested note $NEW written (normal note); $OUT"
