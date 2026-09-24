#!/bin/bash
# End-state checks for the recall-glance scenario: a community-garden vault seeded with 6 notes,
# one of which (note 2, a feedback note) CORRECTS an older note (note 1) about the cause of dark
# leaf spotting on tomatoes. Glance mode is read-only with respect to vault knowledge (recall's own
# SKILL.md: "SKIP Step 2.5C -- it is the write side... continue to Step 2.7 (activate)" and
# "SKIP Step 4 -- write side"), so the ENTIRE point of this scenario's end-state is that NOTHING
# gets written or amended, regardless of the coverage judgment (covered/near/absent never reaches
# a write call under glance). The read-side judgment itself (did the agent apply the recency
# weight and cite the CORRECTING note 2, not the superseded note 1) is scored by steps.json
# (activation + a text_regex on the final reply), not here -- this script only checks vault state.
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

# The shim-only R arm (task 2.x, not built yet) will add the recall runbook set (and possibly a
# write-memory runbook copy) to the vault; those are the arm's treatment, not fixture notes, so
# they are excluded from the note-set check the same way curate/write-memory's own
# done_when_checks.sh exclude their own runbook carriers. Tolerate the directory not existing yet.
CARRIERS="$( (ls "$HERE"/../../../encodings/taskRecall/Recall-R/vault/*.md 2>/dev/null) | xargs -n1 basename 2>/dev/null || true)"

# Check 1: vault invariants hold.
if ! engram check --vault "$V" >/dev/null 2>&1; then
  echo "FAIL: 'engram check' does not pass on the vault"
  exit 1
fi

# Check 2: EXACTLY the six seed notes -- no new note, no amend that would change the note count.
if [ -n "$CARRIERS" ]; then
  ACTUAL_FILES=$(ls "$V"/*.md | xargs -n1 basename | grep -vxF -f <(echo "$CARRIERS") | sort)
else
  ACTUAL_FILES=$(ls "$V"/*.md | xargs -n1 basename | sort)
fi
EXPECTED_FILES=$(ls "$TEMPLATE"/*.md | xargs -n1 basename | sort)
if [ "$ACTUAL_FILES" != "$EXPECTED_FILES" ]; then
  echo "FAIL: vault note set changed -- glance must not write or remove any note. Expected:"
  echo "$EXPECTED_FILES"
  echo "Actual:"
  echo "$ACTUAL_FILES"
  exit 1
fi

# Check 3: every seed note is byte-identical to the template -- glance must not amend (not even a
# provenance-only 'Covered' amend) any note.
for f in $EXPECTED_FILES; do
  if ! cmp -s "$V/$f" "$TEMPLATE/$f"; then
    echo "FAIL: seed note $f was modified -- glance mode must never write, including a"
    echo "  provenance-only 'Covered' amend (Step 2.5C is skipped entirely under glance)"
    exit 1
  fi
done
