## 1. Baseline (Red)

- [x] 1.1 Per `superpowers:writing-skills`, capture the route skill's current baseline behavior for the three touched sections: the cold-start-priors roster line + red-flags table, the aggregate-update procedure + count-audit section, and the evidence-note handoff spec. Confirm the baseline test(s) fail/are-absent against the desired post-edit behavior (RED).

## 2. Edit: roster wording (#702)

- [x] 2.1 Reword `SKILL.md`'s "Current roster: cheap = haiku, mid = sonnet, deep = opus" line to state it as an EXAMPLE, and add the environment-resolution rule: cheap tier = cheapest agent available in THIS environment; free local models (e.g. an LMStudio provider) are the cheapest possible tier and are tried first; paid API models enter the roster only above free-local.
- [x] 2.2 Add a red-flag row to the "Red flags — STOP and re-read" table: dispatching cheap work to a paid API model while a free local model was available → resolve the roster against the environment first.

## 3. Edit: aggregate uniqueness + merge (#679)

- [x] 3.1 Add the uniqueness check to the aggregate-update procedure: on a no-match lookup result, before creating a new aggregate, run a deterministic vault basename listing/glob for `route-evidence-<work-kind>*` (exact-slug match after stripping `.md` and taking the final `.`-split segment, mirroring the existing lookup's match rule) to rule out a missed lookup.
- [x] 3.2 Document the merge procedure for the duplicate case: union the evidence wikilinks, recompute the tally via the existing `engram count --group-by tags` commands, merge into one aggregate, and record the incident as ADR-0019 drowning-remedy evidence.
- [x] 3.3 Wire the uniqueness check to the same trigger moments as the existing count audit (periodic consolidation + doubted-tally checks).

## 4. Edit: copyedits (#680)

- [x] 4.1 Add one sentence to the aggregate-update branch (or nearby) stating the write-path split explicitly: the evidence note goes through write-memory (parents judge, worker writes); the aggregate amend-or-create is composed directly by the route-executing agent because write-memory has no amend form.
- [x] 4.2 Trim the "kind=fact" repetition (currently named 4x in 5 sentences) in the evidence-note handoff spec down to a single clear statement.
- [x] 4.3 Add a half-line warning in the count-audit section: `--group-by tier` runs without error but groups the wrong attribute (the pre-existing L1/L2/L3 note-tier frontmatter field, not the `tier/` evidence tag family) — use `--group-by tags --filter tags=tier/<t>` instead.

## 5. Verify (Green + pressure test)

- [x] 5.1 Re-run the baseline test(s) from 1.1 against the edited skill; confirm GREEN.
- [x] 5.2 Run the skill's pressure-test pass per `superpowers:writing-skills` to confirm no regression in triggering or behavior.
- [x] 5.3 N/A: the trap gate smoke (`dev/eval/cumulative/lever_recheck/smoke_655_e2e*`) exercises `recall`/`learn` retrieval quality only — it never invokes `route` and shares no code path with this change's edits (route's dispatch/aggregate prose, and its "read via plain recall" path, are unchanged). It was inherited into #679's acceptance criteria as this repo's standard skill-edit regression gate (originally built for #655), not because it covers route's dispatch/tier/aggregate logic. Running it would have zero power to detect a regression in this diff, at real API cost (~$0.67+/run). Confirmed with Joe 2026-08-08.

## 6. Close out

- [x] 6.1 Confirm each of the three source issues' acceptance criteria are met by the merged edits (#702, #679, #680).
- [x] 6.2 Sync the `route-dispatch-evidence` and `route-evidence-rubric` delta specs to `openspec/specs/` at archive time.
