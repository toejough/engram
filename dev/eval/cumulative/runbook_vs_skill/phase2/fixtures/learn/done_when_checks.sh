#!/bin/bash
# End-state checks for the learn task: a home-brewing club vault seeded with 6 field-note notes
# plus a read-only reference copy of the REAL production write-memory runbook (note 1053), and a
# task-prompt carrying (1) a mid-task correction (batch #14's false stall reading, caused by a
# clogged airlock under a krausen cap -- not temperature) and (2) an explicit save-request (Campden
# tablets need 24h, not 1h, to dissipate chlorine before pitching). The outcome learn/SKILL.md's
# own text prescribes:
#   kind-1 correction -> write-memory writes ONE new feedback note about the airlock/krausen false
#     stall finding, tagged under whichever process/equipment family fits (task-prompt explicitly
#     asks for a tag -- "tagged under whichever process category fits" -- so tag presence is a
#     hard requirement, not an optional nudge; the exact family is the agent's own judgment call).
#   kind-2 explicit save-request -> write-memory writes ONE new fact note about the Campden
#     dissipation-time rule, tagged sanitation/<value> exactly (task-prompt names this category).
#   whole vault: `engram check` passes, the six seed notes AND the write-memory runbook note 1053
#     are byte-identical to the template (1053 must never be edited -- it is a read-only production
#     reference, not something learn is asked to write to), exactly two new notes exist (no extra
#     writes, no amends to any existing note).
#
# Note-ID caveat (discovered validating this script against a hand-built ideal run): with note
# 1053 present in the vault, `engram learn ... --position top` numbers new top-level notes AFTER
# the vault's highest existing Luhmann number (1054, 1055, ...), NOT after the fixture's own local
# max (6) -- so unlike write-memory/curate's done_when_checks.sh, new notes here are found by
# CONTENT, never by an assumed id ("7"/"8" would be wrong).
#
# Repo-observable only; transcript moves (sweep, both runbook fetches, correct --tag vs --tags,
# --source/--situation) are scored by steps.json. The vault is $ENGRAM_VAULT_PATH (the harness's
# trial env), or <trial_dir>/vault-final for a --rescore of a kept trial.
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

NOTE_1053_BASENAME="1053.2026-09-22.write-memory-compose-execute-verify"
# The shim-only R arm (task 2.x, not built yet) will add the learn runbook (and its sub-runbooks)
# to the vault; those are the arm's treatment, not fixture notes, so they must be excluded from the
# note-set check the same way curate/write-memory's done_when_checks.sh exclude their own runbook
# carriers. Tolerate the directory not existing yet (2>/dev/null) so this check needs no redesign
# once section 2 lands. Note 1053 itself is EXCLUDED from this check (it is a fixed background
# reference present in every arm, not a scored carrier or a scored write) and verified separately.
CARRIERS="$( (ls "$HERE"/../../encodings/taskLearn/Learn-R/vault/*.md 2>/dev/null) | xargs -n1 basename 2>/dev/null || true)"

# Check 1: vault invariants hold.
if ! engram check --vault "$V" >/dev/null 2>&1; then
  echo "FAIL: 'engram check' does not pass on the vault"
  exit 1
fi

# Check 2: the six seed notes are untouched (byte-identical to the template) -- neither
# write-memory handoff should ever amend an existing note; both are NEW writes.
for id in 1 2 3 4 5 6; do
  SEED="$(ls "$TEMPLATE/$id".*.md | head -n 1)"
  N="$V/$(basename "$SEED")"
  if [ ! -f "$N" ] || ! cmp -s "$N" "$SEED"; then
    echo "FAIL: seed note $id was modified, renamed, or removed -- both writes must be NEW notes, not amends"
    exit 1
  fi
done

# Check 3: the write-memory runbook reference (1053) is present and byte-identical -- it is a
# read-only production reference the agent may fetch (`engram show`) but must never edit.
N1053="$V/$NOTE_1053_BASENAME.md"
T1053="$TEMPLATE/$NOTE_1053_BASENAME.md"
if [ ! -f "$N1053" ]; then
  echo "FAIL: write-memory runbook reference note ($NOTE_1053_BASENAME) is missing from the vault"
  exit 1
fi
if ! cmp -s "$N1053" "$T1053"; then
  echo "FAIL: write-memory runbook reference note ($NOTE_1053_BASENAME) was modified -- it must never be edited"
  exit 1
fi

# Check 4: exactly two notes exist beyond the known set (6 seeds + 1053 + any R-arm carriers) --
# no extra writes (e.g. an unrequested QA pair, per vault note 1052's duplicate-guard fix) and
# nothing removed.
KNOWN="$NOTE_1053_BASENAME"
for id in 1 2 3 4 5 6; do
  KNOWN="$(printf '%s\n%s' "$KNOWN" "$(basename "$(ls "$TEMPLATE/$id".*.md | head -n 1)" .md)")"
done
if [ -n "$CARRIERS" ]; then
  KNOWN="$(printf '%s\n%s' "$KNOWN" "$(echo "$CARRIERS" | sed 's/\.md$//')")"
fi
NEW=$(ls "$V"/*.md | xargs -n1 basename | sed 's/\.md$//' | grep -vxF -f <(echo "$KNOWN") || true)
NEW_COUNT=$(echo "$NEW" | grep -c . || true)
if [ "$NEW_COUNT" -ne 2 ]; then
  echo "FAIL: expected exactly 2 new notes (one airlock/krausen finding, one Campden rule), found $NEW_COUNT: [$(echo "$NEW" | tr '\n' ' ')]"
  exit 1
fi

# Check 5: one new note is the airlock/krausen false-stall finding, tagged under SOME family (the
# task prompt explicitly asks for a tag -- "tagged under whichever process category fits" -- so
# tag presence is a hard requirement, not optional; the exact family name is the agent's own call).
N_AIRLOCK=""
N_CAMPDEN=""
while IFS= read -r base; do
  [ -z "$base" ] && continue
  path="$V/$base.md"
  if grep -qiE 'airlock|krausen' "$path"; then
    N_AIRLOCK="$path"
  fi
  if grep -qi 'campden' "$path"; then
    N_CAMPDEN="$path"
  fi
done <<<"$NEW"

if [ -z "$N_AIRLOCK" ]; then
  echo "FAIL: no new note names the airlock/krausen false-stall cause"
  exit 1
fi
if ! grep -qE '^\s*-\s*[a-z][a-z-]*/[a-z][a-z-]*\s*$' "$N_AIRLOCK"; then
  echo "FAIL: the airlock/krausen finding was not tagged under a <family>/<value> pair as the task asked"
  exit 1
fi

# Check 6: the other new note is the Campden dissipation-time rule, tagged sanitation/<value> (the
# task prompt names this exact category).
if [ -z "$N_CAMPDEN" ]; then
  echo "FAIL: no new note carries the Campden dissipation-time rule"
  exit 1
fi
if [ "$N_CAMPDEN" = "$N_AIRLOCK" ]; then
  echo "FAIL: the airlock/krausen finding and the Campden rule were folded into a single note -- one note per distinct principle"
  exit 1
fi
if ! grep -qiE '24[ -]?hour|24h\b|twenty-?four hours?' "$N_CAMPDEN"; then
  echo "FAIL: the Campden note does not carry the 24-hour dissipation rule"
  exit 1
fi
if ! grep -qE '^\s*-\s*sanitation/[a-z][a-z-]*\s*$' "$N_CAMPDEN"; then
  echo "FAIL: the Campden note was not tagged sanitation/<value> as the task asked"
  exit 1
fi

echo "PASS: airlock/krausen finding written and tagged; Campden dissipation rule written and tagged; write-memory runbook reference untouched; seed notes untouched; vault clean"
