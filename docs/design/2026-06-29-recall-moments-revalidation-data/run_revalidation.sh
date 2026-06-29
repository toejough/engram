#!/bin/bash
set -u
BASE="/private/tmp/claude-501/-Users-joe-repos-personal-engram/95570838-0d05-483c-95e7-fe004909b499/scratchpad"
cd "$BASE" || exit 1

IDS=(CF1 CF2 CF3 CF4 CF5)
SCN=(
"You are working in the acme-billing Java service. You just added retry logic to the invoice-sender using a plain for-loop with exponential backoff, and the unit tests pass. The diff compiles cleanly. Your task was 'add retries to invoice sending.'"
"You are working in the northwind-web TypeScript app. You added a date-formatting helper using the native Intl.DateTimeFormat, and the tests pass. The code reads cleanly. Your task -- 'add a date formatter for the order list' -- looks complete."
"You are working in the acme-billing service. Your integration test fails with 'connection refused' against the payments sandbox. You re-ran it -- same error. It passed yesterday and you changed nothing in that area."
"You are working in the northwind-web monorepo. A build step fails with 'module not found' for an internal package that clearly exists in the tree. You re-ran the build -- same error. You did not touch that package."
"You are working in the acme-billing service. The in-process job queue occasionally drops jobs under load. You've decided the cleanest fix is to switch to a new Kafka-based queue, and you're about to start wiring it in."
)
PREFIX="You are a capable software engineering agent with Read, Edit, Bash, Grep, and the engram /recall skill (run 'engram query' to surface relevant memory from past sessions and crystallized lessons)."
SUFFIX="What is your next step? You do NOT need to complete the task and must NOT call any tools. Respond with ONLY two lines -- STEP: <one sentence>; ACTION: <the single first action or command you would take>."

OUT="$BASE/results.txt"
: > "$OUT"
for i in 0 1 2 3 4; do
  P="$PREFIX ${SCN[$i]} $SUFFIX"
  {
    echo "===== RED ${IDS[$i]} ====="
    ( cd "$BASE/red-proj" && claude -p "$P" 2>&1 )
    echo
    echo "===== GREEN ${IDS[$i]} ====="
    ( cd "$BASE/green-proj" && claude -p "$P" 2>&1 )
    echo
  } >> "$OUT"
done
echo "DONE -> $OUT"
