# Learn skill — baseline test index

These are reusable RED/GREEN scenario inputs for `superpowers:writing-skills` TDD — re-run the
relevant baseline before editing `agent-instructions/skills/learn/SKILL.md` (CLAUDE.md mandates it).

| baseline | locks which behavior | re-run before editing |
|---|---|---|
| `baseline-project-specific.md` | An explicit "remember" save-request naming project-specific identifiers (repo, function, task) still triggers a write, but `--situation` must be generalized into a retrieval-shaped handle — not the verbatim project-specific description. | Step 2 (situation generalization) |
| `baseline-clean-write.md` | An explicit save-request: Step 1 sweep runs first, then exactly one clean `engram learn` write at `--position top` — no three-gate logic, no Luhmann-continuation/`--target` write path (both removed from the current skill). | Step 1, Step 2 |
| `baseline-information-not-knowledge.md` | An explicit "remember" on low-signal pure information (no actionable principle) still triggers a write, or an out-loud override rationale — the old Gate-3/Knowledge formal-gate silent-drop is removed. | Step 2 (write-vs-override judgment) |
| `baseline-hindsight-framing.md` | An explicit save-request with hindsight-baked wording ("when fixing X") still triggers a write, but `--situation` must be reframed into forward-looking, pre-fix phrasing — not the verbatim hindsight description. | Step 2 (situation reframing) |
| `baseline-autonomous-trigger.md` | On autonomous (no-user-prompt) self-fire, all four Step-2 kinds apply — including self-discovered ones (kind 3 reversals and kind 4b self-validated bets), which need no user turn; a plain discovered fact (matching no kind) and one-off trivial fixes stay unwritten, left to the chunk index. | Step 2 (autonomous scan scope) |
| `baseline-confirmed-approach.md` | Kind 4 (confirmed approaches): pure user praise of a specific behavior (4a) and a self-validated bet with observable confirmation (4b) each WRITE a `feedback` handoff; generic pleasantries, routine successes with no bet, and unconfirmed guesses do NOT fire. RED (pre-kind-4 skill) writes nothing for the GREEN scenarios. | Step 2 (kind-4 capture + over-capture guard) |

The dated `*-RED-results.md` / `*-GREEN-results.md` files are NOT indexed here — they delete this
cycle (docs-restructure). Run records live in git history.
