# Learn skill — baseline test index

These are reusable RED/GREEN scenario inputs for `superpowers:writing-skills` TDD — re-run the
relevant baseline before editing `skills/learn/SKILL.md` (CLAUDE.md mandates it).

| baseline | locks which behavior | re-run before editing |
|---|---|---|
| `baseline-project-specific.md` | An explicit "remember" save-request naming project-specific identifiers (repo, function, task) still triggers a write, but `--situation` must be generalized into a retrieval-shaped handle — not the verbatim project-specific description. | Step 2 (situation generalization) |
| `baseline-clean-write.md` | An explicit save-request: Step 1 sweep runs first, then exactly one clean `engram learn` write at `--position top` — no three-gate logic, no Luhmann-continuation/`--target` write path (both removed from the current skill). | Step 1, Step 2 |
| `baseline-information-not-knowledge.md` | An explicit "remember" on low-signal pure information (no actionable principle) still triggers a write, or an out-loud override rationale — the old Gate-3/Knowledge formal-gate silent-drop is removed. | Step 2 (write-vs-override judgment) |
| `baseline-hindsight-framing.md` | An explicit save-request with hindsight-baked wording ("when fixing X") still triggers a write, but `--situation` must be reframed into forward-looking, pre-fix phrasing — not the verbatim hindsight description. | Step 2 (situation reframing) |
| `baseline-autonomous-trigger.md` | On autonomous (no-user-prompt) self-fire, only explicit user corrections get crystallized as `engram learn feedback`; self-discovered facts and one-off trivial fixes are NOT written — left to the chunk index — and no three-gate logic runs. | Step 2 (autonomous scan scope) |

The dated `*-RED-results.md` / `*-GREEN-results.md` files are NOT indexed here — they delete this
cycle (docs-restructure). Run records live in git history.
