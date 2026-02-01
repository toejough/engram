# Critical Rules Audit

Extracted from all skills in `~/.claude/skills/*/SKILL.md` by scanning for MUST, NEVER, ALWAYS, and CRITICAL keywords.

## TDD Discipline

1. **ALL tests MUST pass** - No exceptions (task-audit)
2. **Fix ALL failures when found** - NEVER dismiss test failures as "pre-existing" or "unrelated" (task-audit)
3. **NEVER wait for user confirmation between TDD phases** - red → commit → green → commit → refactor → commit is one atomic sequence (project)
4. **Every acceptance criterion MUST specify an ACTION and OBSERVABLE RESULT** (pm-interview)

## Traceability

5. **NEVER create traceability.toml** - All links are inline in artifacts via `**Traces to:**` (alignment-check)
6. **Each ARCH ID MUST reference at least one upstream REQ or DES ID** (architect-interview)
7. **Each DES ID MUST reference at least one upstream REQ ID** (design-interview)
8. **NEVER modify existing REQ-NNN/DES-NNN/ARCH-NNN** without proper traceability updates (various)

## Commit Format

9. **NEVER amend pushed commits** - Check `git status` for "ahead of" first (commit)
10. **NEVER use dangerous commands** - No `git checkout -- .`, `git restore .`, `git reset --hard` (commit)

## Evidence-Based

11. **Evidence-based only** - Every claim MUST reference a specific artifact and traceability ID (architect-audit)
12. **CRITICAL: "Feature exists" ≠ "Feature works"** - MUST verify behavior, not just presence (design-audit)
13. **NEVER audit from text descriptions alone** - Prose specs describe intent; visual specs show exact layout (design-audit)
14. **NEVER silently skip a screen or viewport** during visual audit (design-audit)
15. **Every clickable element MUST be clicked, every input MUST receive input** (design-audit)

## Control Loop

16. **NEVER say "No response requested"** or stop with just a summary when work remains (project)
17. **NEVER ask "Should I continue?"** if `projctl state next` returns `continue` (project)
18. **The control loop NEVER stops between deterministic steps** - task-start through task-complete is atomic (project)
19. **ALWAYS run end-of-command sequence** - integrate → repair → validate before completing (project)
20. **Use `projctl` for all state transitions** - NEVER modify state.toml directly (project)
21. **NEVER skip audits** - Audit loop runs until zero defects (project)

## Other

22. **Interactive interview is ALWAYS required** - Do not skip the interview and just analyze (architect-interview)
23. **Fix and report** - NEVER ask for confirmation (alignment-check)

## Duplicates Identified

The following rules appear in multiple skills:

| Rule Pattern | Skills |
|--------------|--------|
| NEVER modify existing *-NNN | alignment-check, architect-infer, design-infer |
| Evidence-based/must reference | architect-audit, task-audit |
| End-of-command sequence | project (5 references) |
| Tests MUST pass | task-audit (2 references) |

## Summary

- **Total critical rules identified:** 23
- **Categories:** TDD discipline (4), Traceability (4), Commit format (2), Evidence-based (5), Control loop (6), Other (2)
