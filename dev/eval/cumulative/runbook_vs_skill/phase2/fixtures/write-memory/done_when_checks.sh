#!/bin/bash
# End-state checks for the write-memory task: a printer-farm vault seeded with 6 notes (none of
# which covers ambient cross-drafts causing corner warping -- an ABSENT cluster for recall) plus
# an explicit learn save-request (the printer-bay breaker rule). The outcome recall/learn's own
# text prescribes:
#   recall Step 2.5C absent -> write-memory writes ONE new fact/feedback note about the
#     draft/warping cause, tagged under a component/<x> family (task-prompt explicitly asks for
#     a tag, so tag presence is a hard requirement here -- not an optional nudge).
#   learn Step 2's explicit save-request -> write-memory writes ONE new fact note about the
#     breaker/preheat rule, tagged component/electrical (task-prompt names the category).
#   whole vault: `engram check` passes, the six seed notes are byte-identical to the template,
#     exactly two new notes exist (no extra writes, no amends to the seed notes).
# Repo-observable only; transcript moves (query, sweep, correct --tag vs --tags) are scored by
# steps.json. The vault is $ENGRAM_VAULT_PATH (the harness's trial env), or <trial_dir>/vault-final
# for a --rescore of a kept trial.
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

# The shim-only R arm (task 2.x, not built yet) will add the write-memory runbook (and possibly
# the recall/learn runbook carrier set, if a future change converts those too) to the vault; those
# are the arm's treatment, not fixture notes, so they must be excluded from the note-set check the
# same way curate's done_when_checks.sh excludes its own runbook carriers. Tolerate the directory
# not existing yet (2>/dev/null) so this check needs no redesign once section 2 lands.
CARRIERS="$( (ls "$HERE"/../../encodings/taskWriteMemory/WM-R/vault/*.md 2>/dev/null) | xargs -n1 basename 2>/dev/null || true)"
note() { ls "$V"/"$1".*.md 2>/dev/null | head -n 1; }

# Check 1: vault invariants hold.
if ! engram check --vault "$V" >/dev/null 2>&1; then
  echo "FAIL: 'engram check' does not pass on the vault"
  exit 1
fi

# Check 2: exactly the expected note set -- the six seed notes plus exactly two new ones (7, 8).
EXPECTED="1 2 3 4 5 6 7 8"
if [ -n "$CARRIERS" ]; then
  ACTUAL=$(ls "$V"/*.md | xargs -n1 basename | grep -vxF -f <(echo "$CARRIERS") | sed 's/\..*//' | sort -n | tr '\n' ' ' | sed 's/ $//')
else
  ACTUAL=$(ls "$V"/*.md | xargs -n1 basename | sed 's/\..*//' | sort -n | tr '\n' ' ' | sed 's/ $//')
fi
if [ "$ACTUAL" != "$EXPECTED" ]; then
  echo "FAIL: vault note ids are [$ACTUAL], expected [$EXPECTED] (exactly one new note for the draft/warping finding and one for the breaker rule, nothing else written or removed)"
  exit 1
fi

# Check 3: the six seed notes are untouched (byte-identical to the template) -- neither write-memory
# handoff should ever amend an existing note; both are Step-2.5C-absent / explicit-save-request
# NEW writes.
for id in 1 2 3 4 5 6; do
  N="$(note "$id")"
  T="$TEMPLATE/$(basename "$N")"
  if [ ! -f "$T" ] || ! cmp -s "$N" "$T"; then
    echo "FAIL: seed note $id was modified (or renamed) -- both writes must be NEW notes, not amends"
    exit 1
  fi
done

# Check 4: note 7 is the corner-warping/draft finding: mentions the cause, tagged under SOME
# component/<x> family (the task prompt explicitly asks for a tag -- "tag it under whichever
# component category fits" -- so tag presence is a hard requirement, not optional).
N7="$(note 7)"
if [ -z "$N7" ]; then
  echo "FAIL: no note 7 (draft/warping finding) was written"
  exit 1
fi
if ! grep -qiE 'draft|drafts' "$N7"; then
  echo "FAIL: note 7 does not name the draft/cross-draft cause"
  exit 1
fi
if ! grep -qiE '^\s*-\s*component/[a-z-]+' "$N7"; then
  echo "FAIL: note 7 was not tagged under a component/<x> family as the task asked"
  exit 1
fi

# Check 5: note 8 is the breaker/preheat rule, tagged component/electrical (the task prompt names
# this exact category).
N8="$(note 8)"
if [ -z "$N8" ]; then
  echo "FAIL: no note 8 (breaker/preheat rule) was written"
  exit 1
fi
if ! grep -qiE 'breaker|preheat' "$N8" || ! grep -qiE 'stagger|two minutes|2 minutes|2-minute|two-minute' "$N8"; then
  echo "FAIL: note 8 does not carry the breaker-trip cause and the stagger-startups rule"
  exit 1
fi
if ! grep -qxF '    - component/electrical' "$N8"; then
  echo "FAIL: note 8 was not tagged component/electrical as the task asked"
  exit 1
fi

echo "PASS: draft/warping finding written and tagged; breaker/preheat rule written and tagged; seed notes untouched; vault clean"
