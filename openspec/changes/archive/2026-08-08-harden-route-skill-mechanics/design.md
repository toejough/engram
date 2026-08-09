## Context

Three independently-specified fixes to `agent-instructions/skills/route/SKILL.md`, bundled to
share one `writing-skills` TDD pass. Each fix's acceptance criteria are already fully stated in
its source issue (#702, #679, #680) — no technical decisions remain open. This design doc is
intentionally thin: none of the "include design.md" triggers (cross-cutting change, new
dependency, security/perf/migration complexity, unresolved ambiguity) apply here. It exists to
record the ordering/verification approach across the three edits, not to make new decisions.

## Goals / Non-Goals

**Goals:**
- Land all three fixes in one `writing-skills` TDD pass (one baseline RED, one edit set, one
  GREEN, one pressure test) since they touch the same file with no interaction between edits.
- Keep the aggregate-uniqueness check (#679) and the roster reword (#702) both purely additive to
  the skill's existing procedure — no change to the read path (`route` still reads evidence via
  plain recall, never a special query — `route-dispatch-evidence`'s existing "read through plain
  recall" requirement is untouched).

**Non-Goals:**
- No change to `internal/` or `cmd/` code — this is skill-instructions-only.
- No resolution of #670 (rubric-refit threshold), #671 (parallel-builders evaluation), or #672
  (pricing table) — those need decisions this bundle doesn't make; tracked separately.
- No touching the `[[work-kind-definition]]`-vs-"vault note N" citation-style inconsistency
  (#680's optional 4th item) — explicitly gated on #678.

## Decisions

- **One combined TDD pass, not three.** The three edits are independent (different sections of
  the same file, no shared state) and each is already fully specified by its issue — sequencing
  them as three separate baseline/edit/verify cycles would triple the ceremony cost for zero
  extra safety. A single baseline test capturing pre-edit skill behavior, followed by all three
  edits, followed by one GREEN re-run and one pressure-test pass, covers all three.
- **Uniqueness check is a vault basename listing, not a new CLI flag (#679).** Aggregates are
  deliberately untagged (`route-dispatch-evidence`'s "Aggregates SHALL use wikilinks ... SHALL
  NOT carry their own tags" requirement), so `engram count --group-by` cannot enumerate them.
  Adding a new count/query capability just for this check would be new product surface for a
  documentation-only fix; a glob/grep over vault basenames for `route-evidence-<work-kind>*.md`
  is deterministic, requires no code change, and matches the existing count-audit section's
  style (shell commands documented in the skill, not new engram subcommands).
  - Alternative considered: extend `engram count` with a slug-uniqueness mode. Rejected — scope
    creep for a documentation fix; revisit only if the glob approach proves unreliable in
    practice.
- **Roster reword keeps the table, doesn't remove it (#702).** The issue's suggested fix reframes
  the existing "Current roster: cheap = haiku, mid = sonnet, deep = opus" line as an explicit
  EXAMPLE plus an environment-resolution rule, rather than deleting the table (some concrete
  example remains useful for orienting a cold-start agent). This matches the issue's suggested
  wording verbatim.
- **Merge procedure for duplicate aggregates recomputes via existing count commands, doesn't
  invent new ones (#679).** The count-audit section already documents the exact `engram count
  --group-by tags --filter ...` invocations; the merge procedure reuses them rather than
  introducing a new recomputation method.

## Risks / Trade-offs

- [Risk] Bundling three unrelated fixes into one PR makes the diff harder to review atomically.
  → Mitigation: each fix is a self-contained section of the file; the proposal and tasks list
  keep them clearly separated so a reviewer can verify each against its source issue
  independently even though they land together.
- [Risk] The vault-basename-glob uniqueness check (#679) could report a false duplicate if two
  unrelated work-kinds happen to share a slug prefix. → Mitigation: the match rule is exact-slug
  equality after stripping `.md` and taking the final `.`-split segment (the same deterministic
  rule already documented for the aggregate lookup itself) — no fuzzy matching.
- [Risk] Trimming "kind=fact" repetition (#680.2) could under-specify the field for a model that
  needs the reminder. → Mitigation: guarded by a GREEN pressure-test re-run per the issue's own
  acceptance criterion; if the trim causes a regression the pressure test catches it before
  merge.

## Migration Plan

Single-PR, additive-only change to skill instructions; no runtime migration. Steps:
1. Baseline RED: capture current route-skill behavior for the three touched sections (roster
   line/red-flags, aggregate-update procedure, evidence-note handoff spec) per
   `superpowers:writing-skills`.
2. Apply all three edits (#702, #679, #680).
3. GREEN: re-run baseline tests against the edited skill.
4. Pressure test: run the skill's pressure-test pass to confirm no regression in triggering or
   behavior.
5. Trap gate smoke: confirm GREEN before and after (per #679's stated acceptance criterion,
   extended to the whole bundle).

## Open Questions

None — all three fixes are fully specified by their source issues.
