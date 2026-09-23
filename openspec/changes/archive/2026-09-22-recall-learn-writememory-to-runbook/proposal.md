## Why

`route`, `please`, and `curate` are now retired skills, live in production as runbooks the shim discovers via `engram query --text` and literal `triggers:`. `recall`, `learn`, and `write-memory` — the three skills at the center of engram's own memory loop — are the last ones still shipping as `SKILL.md` files. Their runbook bodies already exist, but only inside an eval fixture (`dev/eval/cumulative/runbook_vs_skill/phase2/encodings/shim/recall-learn/vault/`), never promoted (GitHub #760, follow-on to `runbook-shim-follow-frame`). Two production guidance files, `shim.md`'s four re-entry cues and `recall.md`'s four firing cues, are near-verbatim duplicates and both load into every session today — a redundancy tax nothing has claimed responsibility for closing.

`write-memory` is the smallest of the three (140 lines, no red-flags table, no `tests/` baseline directory, a pure worker invoked by `recall`/`learn` with no depth-dial or lesson-kind branching) and has none of `recall`'s open design question (the glance-vs-deep split, resolved separately as two runbook notes rather than tackled here). Converting it first keeps the please/curate pattern warm on a low-risk target and produces the first real end-to-end proof that a *worker* skill (never trigger-invoked directly by a user, always invoked by a parent skill's handoff) survives the conversion.

## What Changes

- Convert `agent-instructions/skills/write-memory/SKILL.md` into a single runbook note (the compose/execute/verify/report procedure fits in one query item; the eval-fixture note `write-memory-compose-execute-verify` at `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/shim/recall-learn/vault/` is the starting material, already spot-checked faithful to the real skill).
- Because write-memory is never invoked by a user's own words — it is always the next action a parent skill (`recall`, `learn`) takes after judging what to capture — its retrieval path is **not** the shim's first-action query. `recall` and `learn` must instead be told, once each stays a skill, to fetch and follow the write-memory runbook directly (by basename or wikilink) as their next action, the same way `please`'s runbook already wikilinks route's promoted runbook. This is the one piece of design this change adds beyond the please/curate template: a *worker* runbook invoked by name from another skill's own text, not by trigger.
- Build a `write-memory` eval task and skill-arm baseline (none exists today; please and curate each needed one built from scratch) in `dev/eval/cumulative/runbook_vs_skill/phase2/`. Because write-memory has no independent trigger surface — it is only ever reached via `recall`/`learn`'s handoff — the eval's shim-only arm keeps `recall`/`learn` as skills and swaps only write-memory for the runbook, measuring whether a parent skill successfully invokes and the worker successfully composes/executes/reports.
- Populate `red_flags` from write-memory's existing rules (flag-mixing across kinds, `--tags` vs `--tag`, qa taking no tag flags, wikilink-not-plain-text citations) — small enough that no 1200-byte pressure is expected, unlike `learn`'s 7-row table or `recall`'s 28-row table.
- Retire `agent-instructions/skills/write-memory/SKILL.md` in this same change, gated on the same three-part bar please/curate used: the shim-only eval within one trial of the skill row, retrieval-by-name verified in the real vault, and `shim.md` confirmed imported in the real target `CLAUDE.md`(s) (already true today).
- Update every live reference to `write-memory` as a skill (`recall/SKILL.md`, `learn/SKILL.md`, `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,c3-components}.md`) so they wikilink/name the runbook instead.

## Deferred (not built in this change)

This change is scoped to `write-memory` only, per the ordering agreed in exploration. The following remain open GitHub-#760 scope, each its own future change:

- **`learn`-to-runbook** (293 lines, 7-row red-flags table, 4-kind crystallization branching, hands off to write-memory — structurally closer to `please`/`curate`'s shape than to a worker).
- **`recall`-to-runbook**, gated on the glance/deep design already decided (two separate runbook notes, `recall-glance` and `recall-deep`, not a schema tier field or single-note escalation logic) but not yet built; `recall` is 344 lines with a 28-row red-flags table that will need sub-runbooks (like `please`'s conversion) to fit the 1200-byte cap.
- **Retiring `agent-instructions/guidance/{recall,delegate,learn}.md`** and resolving the `shim.md`/`recall.md` firing-cue redundancy — only sound once `recall` and `learn` themselves are real runbooks, not just write-memory.
- Re-validating `engram update --with-guidance`'s sync behavior once `shim.md` is closer to being the sole production guidance file.

## Capabilities

### New Capabilities

(none — applies the existing runbook/follow-frame/lexical-triggers capabilities to a new carrier)

### Modified Capabilities

- `write-memory-worker`: requirements naming "the write-memory skill" as the carrier update to name the write-memory runbook; the worker-invocation requirement gains a scenario for how a parent skill (`recall`, `learn`) locates and fetches the runbook by name/wikilink rather than by the shim's first-action query (since write-memory has no user-facing trigger of its own). Behavior (accept handoff, compose, execute, verify, report) is unchanged.
- `guidance-runbook-follow-frame`: gains a requirement (or a scenario on an existing one) covering a runbook reached by a skill's own explicit reference rather than by `engram query`'s first-action trigger or similarity match — the follow-frame's announce/restate/`done_when`/`red_flags` obligations apply the same way once such a runbook is in hand.
- `learn-runbook-capture`: the write-memory-compose requirement naming "The `write-memory` skill SHALL compose…" updates the carrier to name the write-memory runbook (reached by basename/wikilink, not `engram query`). Behavior (compose/execute/verify/report a runbook-note write) is unchanged.
- `learn-branching-disposition`: the write-memory handoff-contract requirement naming "The write-memory skill's handoff contract…" updates the carrier the same way. Behavior (accept position/target, pass through to the composed command) is unchanged.

## Impact

- **Removed**: `agent-instructions/skills/write-memory/SKILL.md` (and its directory), after gates pass.
- **Added**: write-memory's promoted runbook in the production vault; a `write-memory` eval task, skill-arm baseline, and conversion-fidelity report under `dev/eval/cumulative/runbook_vs_skill/phase2/`.
- **Modified skills**: `agent-instructions/skills/{recall,learn}/SKILL.md` — the sentence(s) invoking write-memory change from "invoke the write-memory skill" to naming/fetching the runbook.
- **Docs**: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,c3-components}.md` wherever they list write-memory as a skill or diagram the recall/learn→write-memory handoff.
- **Deployment**: `engram update` stops shipping the write-memory skill directory and removes deployed copies in both Claude Code and Pi harnesses.
- **Eval spend**: new fixture/baseline plus a shim-only probe run; estimate and confirm cost before running (please's and curate's each ran $1–5 per stage).
- **Issues**: partial progress on #760 (closes the write-memory slice; `learn` and `recall` slices remain open, to be filed as their own changes once this lands).
