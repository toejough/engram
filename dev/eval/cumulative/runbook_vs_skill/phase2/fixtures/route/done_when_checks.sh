#!/bin/bash
set -euo pipefail

REPO_DIR="$1"
cd "$REPO_DIR"

# Check 1: Verify trial vault path is available
if [ -z "${ENGRAM_VAULT_PATH:-}" ]; then
  echo "FAIL: ENGRAM_VAULT_PATH not set in environment (harness-side plumbing missing)"
  exit 1
fi

VAULT_DIR="$ENGRAM_VAULT_PATH"
if [ ! -d "$VAULT_DIR" ]; then
  echo "FAIL: ENGRAM_VAULT_PATH points to non-existent directory: $VAULT_DIR"
  exit 1
fi

# Check 2: The CLI --version flag prints VERSION content and exits 0
VERSION_CONTENT=$(cat VERSION)
ACTUAL_OUTPUT=$(python3 cli.py --version 2>&1 || true)
ACTUAL_RC=$?

if [ "$ACTUAL_RC" -ne 0 ]; then
  echo "FAIL: 'python3 cli.py --version' exited with code $ACTUAL_RC (expected 0)"
  exit 1
fi

if [ "$ACTUAL_OUTPUT" != "$VERSION_CONTENT" ]; then
  echo "FAIL: 'python3 cli.py --version' output '$ACTUAL_OUTPUT' does not match VERSION '$VERSION_CONTENT'"
  exit 1
fi

# Check 3: pytest passes
if ! python3 -m pytest -q; then
  echo "FAIL: pytest did not pass"
  exit 1
fi

# Check 4: A test referencing --version exists
if ! grep -rq "\-\-version" tests/; then
  echo "FAIL: no test file references '--version'"
  exit 1
fi

# Check 5: Vault contains at least one NEW note with work-kind/, tier/, outcome/ tags
# A new note is one whose basename is not in the background set (which is created by removing
# the fixture notes before the trial starts).
# We count .md files (excluding .vec.json and qa.* files) in the vault.
NEW_NOTES=()
for note in "$VAULT_DIR"/*.md; do
  if [ ! -f "$note" ]; then
    continue
  fi
  basename=$(basename "$note" .md)

  # Parse the frontmatter to extract tags
  if ! grep -q "^tags:" "$note" 2>/dev/null; then
    continue
  fi

  # Check for required tag patterns
  has_work_kind=0
  has_tier=0
  has_outcome=0

  if grep -q "work-kind/" "$note" 2>/dev/null; then
    has_work_kind=1
  fi
  if grep -q "tier/" "$note" 2>/dev/null; then
    has_tier=1
  fi
  if grep -q "outcome/" "$note" 2>/dev/null; then
    has_outcome=1
  fi

  if [ "$has_work_kind" -eq 1 ] && [ "$has_tier" -eq 1 ] && [ "$has_outcome" -eq 1 ]; then
    NEW_NOTES+=("$basename")
  fi
done

if [ "${#NEW_NOTES[@]}" -lt 1 ]; then
  echo "FAIL: no notes found with required tags (work-kind/, tier/, outcome/)"
  exit 1
fi

# Check 6: A note with slug starting with route-evidence- wikilinks one of the evidence notes
ROUTE_EVIDENCE_NOTES=()
for note in "$VAULT_DIR"/*.md; do
  if [ ! -f "$note" ]; then
    continue
  fi
  basename=$(basename "$note" .md)
  # Check if the slug (part after the date) starts with route-evidence-
  # Luhmann format: NNNN.YYYY-MM-DD.slug
  if [[ "$basename" =~ ^[0-9]+\.[0-9]{4}-[0-9]{2}-[0-9]{2}\.route-evidence- ]]; then
    ROUTE_EVIDENCE_NOTES+=("$basename")
  fi
done

if [ "${#ROUTE_EVIDENCE_NOTES[@]}" -lt 1 ]; then
  echo "FAIL: no route-evidence-* notes found"
  exit 1
fi

# Verify that at least one route-evidence note wikilinks one of the NEW_NOTES
FOUND_WIKILINK=0
for evidence_note in "${ROUTE_EVIDENCE_NOTES[@]}"; do
  evidence_file="$VAULT_DIR/$evidence_note.md"
  for new_note in "${NEW_NOTES[@]}"; do
    # Check for wikilinks like [[basename]] or [[basename.something]]
    if grep -q "\[\[$new_note" "$evidence_file" 2>/dev/null; then
      FOUND_WIKILINK=1
      break 2
    fi
  done
done

if [ "$FOUND_WIKILINK" -ne 1 ]; then
  echo "FAIL: no route-evidence-* note wikilinks one of the dispatch evidence notes"
  exit 1
fi

echo "PASS: route task end-state verified"
exit 0
