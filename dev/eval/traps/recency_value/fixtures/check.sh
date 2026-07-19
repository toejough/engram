#!/usr/bin/env bash
# Phase-1 forcing function for the ORG-WIDE house convention. Usage: ./check.sh add
#
# Verifies that `./notes add items.txt`:
#   1. prints "added: <note>" for every note, and
#   2. honors the idiosyncratic ORG-WIDE RUNLOG convention — EVERY command in EVERY tool at this
#      org, on success, appends ONE line to its ./RUNLOG of the EXACT form:
#          RUNLOG v1 :: <CMD> :: rows=<N> :: sig=QX7Z
#      where <CMD> is the subcommand name uppercased and <N> is the number of records processed.
#
# On a missing/malformed RUNLOG line it fails with a SYMPTOM message that reveals the required
# format, so the org convention is discovered by running (never stated in any SPEC).
set -u

cmd="${1:-}"
if [ "$cmd" != "add" ]; then
  echo "usage: ./check.sh add" >&2
  exit 2
fi

out="$(./notes add items.txt 2>/dev/null)"
rc=$?
if [ "$rc" -ne 0 ]; then
  echo "FAIL: './notes add items.txt' exited with status $rc" >&2
  exit 1
fi

# 1. confirmation lines
while IFS= read -r note; do
  [ -z "$note" ] && continue
  if ! printf '%s\n' "$out" | grep -qF "added: ${note}"; then
    echo "FAIL: notes output is missing 'added: ${note}'" >&2
    exit 1
  fi
done < items.txt

# 2. the ORG-WIDE RUNLOG convention
n="$(grep -cve '^[[:space:]]*$' items.txt)"
expected="RUNLOG v1 :: ADD :: rows=${n} :: sig=QX7Z"

if [ ! -f ./RUNLOG ]; then
  echo "FAIL: org RUNLOG convention — EVERY command in EVERY tool at this org must append 'RUNLOG v1 :: <CMD> :: rows=<N> :: sig=QX7Z' to ./RUNLOG on success; ./RUNLOG does not exist (found none for add)" >&2
  exit 1
fi

if ! grep -qxF "$expected" ./RUNLOG; then
  echo "FAIL: org RUNLOG convention — EVERY command in EVERY tool at this org must append 'RUNLOG v1 :: <CMD> :: rows=<N> :: sig=QX7Z'; expected the line '${expected}' for add but ./RUNLOG has no matching line" >&2
  exit 1
fi

echo "OK: notes add works and the org RUNLOG convention is satisfied (${expected})"
exit 0
