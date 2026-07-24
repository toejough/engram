# Baseline pressure test — explicit save-request with project-specific content

## Scenario

The user says: "remember that the engram learn binary required us to extract writePromoteUnderLock when the cyclomatic complexity check fired on Task 8."

## Expected current-skill behavior (PASS)

The user said "remember" — this is an explicit save-request. The current skill writes it.

- **Step 1 — sweep:** `engram ingest --auto`.
- **Step 2 — explicit save-request detected.** Write one `engram learn fact` note.
- **`--situation` judgment:** the situation names a specific project ("engram"), a specific function (`writePromoteUnderLock`), and a specific task ("Task 8"). The skill says `--situation` must be a "retrieval handle" — phrased as a future task would be described. The agent should generalize to the retrievable principle: "when a cyclomatic complexity linter fires on a locked critical section."
- **`--object`:** the transferable principle is "extract the locked region into a helper function to reduce complexity."

```bash
engram learn fact --slug cyclomatic-complexity-extract-locked-helper \
  --position top \
  --source "session <date>, context: user asked to remember engram Task 8 refactor" \
  --situation "when a cyclomatic complexity check fires on a function containing a locked critical section" \
  --subject "the locked region" \
  --predicate "should be extracted into" \
  --object "a dedicated helper function to satisfy the linter without splitting the lock"
```

**Result:** 1 write. The agent generalizes the principle; does not embed "engram", "writePromoteUnderLock", or "Task 8" in the situation field.

## Failure modes that must FAIL this test

- Not writing anything (user said "remember").
- Writing `--situation "when the engram learn binary's cyclomatic complexity check fired on Task 8"` — project-specific identifiers in the situation field defeat future retrieval.
- Running Gate 1 (Recurs) → FAIL → drop. Gate logic is removed.
- Writing an episode or running `engram transcript`.

## What changed from the pre-v2 expected behavior

Pre-v2: run Gate 1 (Recurs) → FAIL (project-specific) → drop, no write. Current skill: explicit save-request = write; the only judgment is to generalize the `--situation` string so it is retrieval-shaped rather than project-specific.
