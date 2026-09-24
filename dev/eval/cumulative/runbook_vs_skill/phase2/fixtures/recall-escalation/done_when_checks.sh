#!/bin/bash
# End-state checks for the recall-escalation scenario: a community-garden vault whose one topical
# note (the OLD peat-based starter-mix convention) is CONTRADICTED by a brand-new standard that
# only appears in the agent's own Step 0.5 sweep of docs/co-op-notes-2026-09-22.md (a recency-
# channel / Channel-2 item, per recall's own C5 rule -- glance surfaces it but does not elevate it
# to a requirement unless escalation happens, #661). The correct outcome REQUIRES escalating to
# `deep`: the closing synthesis crystallizes the new standard (Step 4 persist, since the source is
# a Channel-2 chunk, never a Channel-1 cluster candidate, so the normal Step 2.5C coverage table
# never reaches it) via write-memory's real promoted runbook
# (1053.2026-09-22.write-memory-compose-execute-verify, copied into vault-template so `engram show
# 1053...` resolves in a fixture-only trial vault, per learn's own fixture precedent).
#
# Verified against the real binary (no LLM call): `engram learn fact --supersedes
# "<old-basename>|updates|<claim>"` does NOT mutate the target note's file at all -- the inverse
# link is graph-computed at read time, never written back (confirmed via byte-diff before/after).
# So "every seed note including the old peat note is byte-identical to the template" is the
# correct invariant, not a relaxed one.
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

CARRIERS="$( (ls "$HERE"/../../../encodings/taskRecall/Recall-R/vault/*.md 2>/dev/null) | xargs -n1 basename 2>/dev/null || true)"
note_by_slug() { ls "$V"/*."$1".md 2>/dev/null | head -n 1; }

# Check 1: vault invariants hold.
if ! engram check --vault "$V" >/dev/null 2>&1; then
  echo "FAIL: 'engram check' does not pass on the vault"
  exit 1
fi

# Check 2: every seed note (the six template notes, including the write-memory reference 1053 and
# the old peat-mix note) is byte-identical to the template -- confirmed above that a `--supersedes`
# write never mutates its target.
TEMPLATE_FILES=$(ls "$TEMPLATE"/*.md | xargs -n1 basename | sort)
for f in $TEMPLATE_FILES; do
  if [ ! -f "$V/$f" ] || ! cmp -s "$V/$f" "$TEMPLATE/$f"; then
    echo "FAIL: seed/reference note $f is missing or was modified -- no seed note is ever amended"
    echo "  by this task (the old peat note is superseded, never amended in place)"
    exit 1
  fi
done

# Check 3: exactly ONE new note exists beyond the template set (plus any R-arm carriers, excluded).
ALL_FILES=$(ls "$V"/*.md | xargs -n1 basename | sort)
if [ -n "$CARRIERS" ]; then
  ALL_FILES=$(echo "$ALL_FILES" | grep -vxF -f <(echo "$CARRIERS") || true)
fi
NEW_FILES=$(comm -13 <(echo "$TEMPLATE_FILES") <(echo "$ALL_FILES"))
NEW_COUNT=$(echo "$NEW_FILES" | grep -c . || true)
if [ "$NEW_COUNT" -ne 1 ]; then
  echo "FAIL: expected exactly ONE new note (the crystallized coir-switch standard), found $NEW_COUNT:"
  echo "$NEW_FILES"
  exit 1
fi

NEW_NOTE="$V/$NEW_FILES"

# Check 4: the new note's substantive BODY PROSE (excluding the auto-generated "Supersedes:"
# backlink line, whose target basename happens to contain the literal substring "peat" as part of
# the old note's slug -- a note could satisfy a naive "grep the whole file for peat" check purely
# from that auto-generated line, never having stated the standard itself) carries the new
# standard's content -- coconut coir named as the replacement AND peat named as what it replaces.
NEW_NOTE_BODY="$(awk '/^---$/{c++; next} c>=2 && $0 !~ /^Supersedes:/' "$NEW_NOTE")"
if ! grep -qiE 'coir|coconut' <<<"$NEW_NOTE_BODY"; then
  echo "FAIL: the new note's body does not name the coconut-coir replacement mix"
  exit 1
fi
if ! grep -qiE 'peat' <<<"$NEW_NOTE_BODY"; then
  echo "FAIL: the new note's body does not name peat as the retired/superseded mix"
  exit 1
fi

# Check 5: the new note actually supersedes the OLD peat-mix note (the escalation, not a
# free-floating new fact unconnected to the corrected convention).
OLD_PEAT_BASENAME="$(basename "$(note_by_slug seedling-starter-mix-peat)")"
if [ -z "$OLD_PEAT_BASENAME" ]; then
  echo "FAIL: could not resolve the old peat-mix note's basename from the template"
  exit 1
fi
# Match with or without the trailing .md -- the skill's own supersedes examples cite a "basename"
# the same way `engram show`/wikilinks do (no extension), but the flag's literal string value is
# never normalized by the binary (confirmed: it echoes back whatever the caller passed), so either
# form is a faithful completion.
OLD_PEAT_BASENAME_NOEXT="${OLD_PEAT_BASENAME%.md}"
if ! grep -qF "$OLD_PEAT_BASENAME_NOEXT" "$NEW_NOTE"; then
  echo "FAIL: the new note does not supersede/cite the old peat-mix note ($OLD_PEAT_BASENAME)"
  exit 1
fi
