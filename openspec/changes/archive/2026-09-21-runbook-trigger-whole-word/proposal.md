## Why

Runbook triggers should work on bare words (`curate`, `please`, `learn`, `recall`), not only slash
forms. The shipped matcher is a plain case-insensitive substring, so a bare-word trigger fires inside
unrelated words: `curate` hits "accurate", "inaccurate", "curated"; `please` hits "pleased" and
"displease". That over-firing is why the authoring rule forbids lone words. Matching whole words
removes the false hits and makes bare-word triggers safe.

## What Changes

- The trigger match becomes whole-word (boundary-aware): after the existing normalization (lowercase,
  whitespace runs collapsed to one space, both sides), a trigger matches when it occurs in the text
  and the character immediately before and after the occurrence (if any) is not a letter or digit.
  A boundary is enforced only on a side where the trigger's own edge character is a letter or digit.
- Every occurrence is tried, not just the first.
- Docs describing the match rule (GLOSSARY, C3 K6 row, README, ADR-0026 amendment) are updated.
- Not changed here: the write-memory authoring rule wording (see tasks.md, deferred: a SKILL.md edit
  needs paid writing-skills tests).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `runbook-lexical-triggers`: requirement "A trigger hit SHALL surface first in `items[]`, regardless
  of similarity" — the match sentence changes from substring to whole-word, with new scenarios.

## Impact

- `internal/cli/query_triggers.go` (`matchTriggers`, new `containsWholeWord`), its tests.
- Existing triggers: `please` no longer matches "pleased"; `/please` and `take this end-to-end`
  behave as before (see design).
- No frontmatter, CLI flag, payload, or sidecar change; no vault migration.
