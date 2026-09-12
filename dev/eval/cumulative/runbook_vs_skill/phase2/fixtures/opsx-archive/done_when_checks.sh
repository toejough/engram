#!/bin/bash
set -euo pipefail

REPO_DIR="$1"
cd "$REPO_DIR"

CHANGE_NAME="add-rate-limiting"

if [ -d "openspec/changes/$CHANGE_NAME" ]; then
  echo "FAIL: openspec/changes/$CHANGE_NAME still exists -- the change was not archived"
  exit 1
fi

mapfile -t ARCHIVED < <(find openspec/changes/archive -mindepth 1 -maxdepth 1 -type d -name "*-$CHANGE_NAME" 2>/dev/null | sort)
if [ "${#ARCHIVED[@]}" -eq 0 ]; then
  echo "FAIL: no archived directory matching openspec/changes/archive/*-$CHANGE_NAME was found"
  exit 1
fi
ARCHIVE_DIR="${ARCHIVED[-1]}"
ARCHIVE_BASENAME="$(basename "$ARCHIVE_DIR")"
case "$ARCHIVE_BASENAME" in
  [0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]-"$CHANGE_NAME") ;;
  *)
    echo "FAIL: archived dir $ARCHIVE_BASENAME does not match the YYYY-MM-DD-$CHANGE_NAME naming convention"
    exit 1
    ;;
esac

for f in proposal.md design.md tasks.md; do
  if [ ! -f "$ARCHIVE_DIR/$f" ]; then
    echo "FAIL: archived change is missing $f"
    exit 1
  fi
done

mapfile -t REMAINING < <(find openspec/changes -mindepth 1 -maxdepth 1 -type d ! -name archive)
if [ "${#REMAINING[@]}" -ne 0 ]; then
  echo "FAIL: leftover active change dir(s) under openspec/changes/: ${REMAINING[*]}"
  exit 1
fi

if ! grep -Eq '^### Requirement: .*[Rr]ate' openspec/specs/widget-api/spec.md; then
  echo "FAIL: main spec openspec/specs/widget-api/spec.md is missing a synced rate-limiting Requirement"
  exit 1
fi
if ! grep -q "429" openspec/specs/widget-api/spec.md; then
  echo "FAIL: main spec openspec/specs/widget-api/spec.md does not contain the synced requirement's scenario content"
  exit 1
fi

if ! openspec validate --all --strict; then
  echo "FAIL: 'openspec validate --all --strict' does not pass after archiving (see output above)"
  exit 1
fi

echo "PASS: opsx-archive task end-state verified"
exit 0
