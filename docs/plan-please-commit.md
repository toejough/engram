# Plan — commit the `please` skill

## Scope

The `please` skill was authored in a prior session and exists on disk but is
untracked. Bring it into the repo cleanly.

## Commits

1. **`feat(skills): add please — drive an ask end-to-end through a fixed
   seven-step workflow`**
   - Adds `skills/please/SKILL.md` and `commands/please.md`.
   - Mirrors the `learn` / `recall` structural convention (skill source under
     `skills/<name>/`, slash-command wrapper under `commands/<name>.md`).
   - No code change — `internal/update` discovers skills by walking the
     `skills/` tree, so the new skill is automatically picked up by
     `engram update`.

2. **`docs: mention please skill alongside recall and learn`**
   - `README.md`: update the line "Two skills — `recall` and `learn`" and
     the skill-table to include `please`.
   - `CLAUDE.md`: same update on the line "Two skills — `recall` and
     `learn` — read from and write to the vault on demand". The phrasing
     needs a small refactor since `please` doesn't read/write the vault —
     it's an orchestrator.

3. **`chore: remove root-level please-skill authoring scaffolding`**
   - `rm please.md please-prompt.md` (authoring inputs, not part of the
     shipped skill).

4. **`chore: remove plan artifact for please-skill commit`**
   - `rm docs/plan-please-commit.md` after work is complete.

## Verification

- `targ check-full` clean (skill change is doc-only, so only the
  `check-uncommitted` gate is meaningful — should pass after final
  commit).
- `git status` clean.
- `git log` shows the four commits in order, each focused.

## Out of scope

- No skill-author TDD pressure-test of the `please` skill itself — the
  skill was already authored (per the prompt in `please-prompt.md`,
  generated via `superpowers:writing-skills` TDD); this work is only
  about committing what exists. If RED/GREEN validation of `please` is
  desired, that's a follow-up.
- No automated trigger / hook for `/please`.
