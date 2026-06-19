# GREEN results — bootstrap create (agent-judged model, empty candidate_l2s)

Run against the current SKILL.md (agent-judged covered/near/absent + `candidate_l2s` list).

## Result: PASS — all five criteria met

| Cluster | candidate_l2s | Expected | Agent action | Result |
|---------|--------------|----------|-------------|--------|
| 0 | empty | CREATE | `engram learn fact --position top --relation "3|storage build notes" --source "..."` inline, wait | PASS |
| 1 | empty | CREATE | `engram learn fact --position top --relation "4|atomic writes" --relation "5|fsync notes" --source "..."` inline, wait | PASS |
| 2 | 0.97 (content in payload) | COVERED → amend --activate | Read content from `items[]` field directly (no `engram show`); judge: covers "FS interface injection" with no material omission; `engram amend --target 6.2026-05-01.filestore-interface.md --activate --chunk-source ...` | PASS |
| 3 | 0.85, 0.71, 0.58 | COVERED → amend top candidate | `engram show` on all three candidates; top candidate (0.85) covers storage-format principle; `engram amend --target 7.2026-03-15.storage-format.md --activate --chunk-source 8.2026-05-20.format-migration-notes.md` | PASS |

**Key evidence the agent would produce:**

- Cluster 0/1: *"Step 2.5 C: `candidate_l2s` is empty — no candidate addresses this cluster's
  principle. Outcome: **absent**. Action: `engram learn fact --position top ...`"*
- Cluster 2: *"Candidate content is already in the payload's `items[]` field — no `engram show`
  call needed. Content covers the principle with no material omission. Outcome: **covered**.
  Action: `engram amend --target ... --activate`."*
- No cosine-band gate, no size precondition, no legacy episode framing anywhere in the agent's reasoning.

## Verdict: GREEN

Empty `candidate_l2s` → absent → CREATE; non-empty candidates → agent reads and judges. Fresh
vault bootstraps L2 notes from the first recall run. All writes are inline and blocking.
