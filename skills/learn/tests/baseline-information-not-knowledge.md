# Baseline pressure test — user says "remember" but the content is pure information (no actionable principle)

## Scenario

The user says: "remember that we noticed the targ tool prints warnings in yellow."

## Expected current-skill behavior (PASS)

The user said "remember" — that IS an explicit save-request. The current skill writes it. However, the skill says `--situation` must be a "retrieval handle" and `--action` or `--object` must encode an actionable principle or standard.

The agent should recognize this and write the note. The situation is "when using targ as a build tool" and the object is "warnings appear in yellow stdout". This is low-signal but the user explicitly asked.

**Result:** 1 write — `engram learn fact` — recording the observation.

```bash
engram learn fact --slug targ-warnings-yellow \
  --position top \
  --source "session <date>, context: user noted targ output color" \
  --situation "when using targ as a build tool and reading its output" \
  --subject "targ" \
  --predicate "prints warnings in" \
  --object "yellow (stdout)"
```

**Alternative:** If the agent judges this as "pure information with no retrieval value" and documents that judgment in its report, that is also acceptable — the skill says "No moments of any kind → write nothing" but the save-request is explicit, so the agent must either write or explain why it is overriding the explicit request (which should be rare and stated out loud).

## Failure modes that must FAIL this test

- Silently dropping the note without explanation (user said "remember").
- Running three-gate checks and failing "Gate 3: Knowledge" as formal gate logic.
- Writing an episode or running `engram transcript`.

## What changed from the pre-v2 expected behavior

Pre-v2: run Gate 1 (Recurs) → FAIL "targ" is project-specific → drop. The current skill has no Recurs gate. A user saying "remember" triggers a write unless the agent has a specific reason to document its refusal. The current test checks that gate-based dropping is gone.
