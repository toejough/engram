# Ground Truth: Expected Audit Results for Dedup Cycle

This table encodes the expected audit verdicts per surprise. The `reachable_by_existing_audit` column is computed by the algorithm specified below, never by manual judgment.

## Algorithm for Computing `reachable_by_existing_audit`

1. Each surprise has an "originating record ID" (a commit SHA or gate_log.txt entry) authored by hand below.
2. For each originating record, scan its FULL text for EXACTLY these patterns (case-sensitive where noted):
   - STOP: regex `\bSTOP\b` (case-sensitive)
   - Gate FAIL: regex `\bGate [A-D]\b[^\n]{0,60}\bFAIL\b` (case-sensitive on "Gate" and "FAIL"), OR a gate_log.txt entry with `verdict: FAIL` and matching `unit:` field
   - CORRECTION-class: regex `\bCORRECTION\b` (all-caps, case-sensitive) OR `\bsupersede[ds]?\b` (case-insensitive) OR `\binstrument-invalid\b` (case-insensitive) OR `\bredraw[n]?\b` (case-insensitive)
   - Escalation: regex `\bAskUserQuestion\b` OR `\bescalat(e|ed|ion)\b` (case-insensitive)
3. If ANY check matches anywhere in the originating record's text: `reachable_by_existing_audit = true`
4. If NO checks match: `reachable_by_existing_audit = false`

## Expected Results

| Surprise ID | Marker | Classification | Originating Record | Reachable by Existing Audit? |
|---|---|---|---|---|
| `discriminator_1` | S5 | (c) never captured until this cycle's own review | fb2c9f45 | NO |
| `discriminator_2` | S6 | (c) never captured until this cycle's own review | 33481996 | NO |
| `adr_hedge` | S6 | (b) present-but-stale (Gate D caught it) | gate_log.txt:adr_hedge | YES |
| `uncited_number` | S7 | (c) never captured before Gate D | gate_log.txt:uncited_number | YES |

## Justification

**discriminator_1** (record-subset premise false):
- Originating record: commit fb2c9f45 (fix(dedup): gate eviction on record subset)
- Checks: STOP (no), Gate FAIL (no), CORRECTION-class (word "corrections" appears lowercase, not all-caps "CORRECTION"; no "superseded/supersede"; no "instrument-invalid"; no "redraw"), Escalation (no)
- Result: NO — the fix exists but carries no audit marker

**discriminator_2** (live-index near-miss during Unit 5 verification):
- Originating record: commit 33481996 (fix(ingest): exclude .claude/jobs from the --auto sweep)
- Checks: STOP (no), Gate FAIL (no), CORRECTION-class (no), Escalation (no)
- Result: NO — the fix exists but carries no audit marker

**adr_hedge** (ADR documentation hedge — Gate D FAIL):
- Originating record: gate_log.txt entry with id "adr_hedge"
- Checks in commit b0985c4d: "superseded" appears in "an earlier '0 unsafe removals' claim in cb6b9540 is superseded by name"
- Gate log: verdict is "FAIL"
- Result: YES — Gate D FAIL marker present

**uncited_number** (LEDGER figure lacks citation — Gate D FAIL):
- Originating record: gate_log.txt entry with id "uncited_number"
- Checks: Gate log entry has verdict "FAIL"
- Result: YES — Gate D FAIL marker present
