## Why

engram's shim-and-runbook architecture is meant to replace agent-instruction skills with runbook notes
surfaced by `engram query`. `route` was the first skill retired this way (`route-skill-to-runbook`,
2026-09-19); `please` is next (#758, split from #756). `please/SKILL.md` is the largest remaining
orchestration skill (156 lines, ~3,400 words: seven-step workflow, four adversarial review gates, a
~22-row Red Flags table, a lessons audit, a non-waivable doc-surface enumeration grep). It is also the
skill that decides how other skills are sequenced, so leaving it as the lone skill-shaped orchestrator
beside a runbook-shaped `route` splits the delivery model in two.

Unlike route, **no `please` eval task or skill-arm baseline exists** in
`dev/eval/cumulative/runbook_vs_skill/phase2/` (fixtures cover history-rewrite, bisect-before-fix,
route, and generic gitignore/commit/tdd-order/opsx tasks only). Route was retired only after a shim-only
probe met the D8 bar (3/3 vs the skill row's 2/3); please's size (~20x route's runbook) makes an
unvalidated conversion a real risk, so building the fixture and baseline is in scope, not deferred.

## What Changes

- Convert `agent-instructions/skills/please/SKILL.md` into a runbook note set: a top runbook (the
  seven-step spine, `situation`, `done_when`) plus wikilinked sub-runbooks for the parts that do not
  fit one query item (adversarial review gates, lessons audit, doc-surface enumeration grep). Body is
  preserved verbatim where possible; fidelity deltas are recorded in a conversion report.
- Drop content the shim's follow-frame already states uniformly for every runbook (anti-sycophancy
  lean, escalation provenance, "N/A is a high bar", user-can't-waive-steps); record each drop as a
  fidelity delta rather than duplicating it.
- Populate `red_flags` from the SKILL.md's Red Flags table. `red_flags` output is capped at 1200 bytes,
  so the highest-value please-specific failure modes go in `red_flags` and the overflow moves into
  sub-runbook bodies. Include one entry for the wikilink hazard from route's eval (writing a
  `[[wikilink]]` reference as plain text in a structured field); wikilinks appear only in runbook
  bodies, never in `red_flags`/`done_when`.
- The top runbook's `situation` line carries the trigger discrimination the SKILL.md `description`
  carried today: fires when the user hands over multi-step work to be carried end-to-end (`/please
  <ask>` or equivalent phrasing), not on casual "please" attached to a single trivial action.
- Build a `please` eval task and skill-arm baseline under
  `dev/eval/cumulative/runbook_vs_skill/phase2/`, then run `probe_phase2.py --task please --shim-only`
  against the converted runbook. Retirement is gated on the D8 bar (within one trial of the skill row).
- Promote the validated runbooks to the production vault via `engram learn runbook` (never a hand-edit
  or file copy), then verify retrieval: task-phrase and situation-phrase `engram query` surface the top
  runbook top-ranked (no LLM spend, trial vault).
- Retire `agent-instructions/skills/please/SKILL.md` in this same change, only after (a) D8 is met, (b)
  retrieval is verified, and (c) `shim.md` is confirmed imported in the real target CLAUDE.md(s).
  **BREAKING**: `/please` stops being a Skill-invokable slash command; delivery becomes
  retrieval-dependent via the shim.
- Update every live reference the retirement would break: `CLAUDE.md`, `README.md`,
  `agent-instructions/guidance/delegate.md`, `agent-instructions/skills/learn/SKILL.md`,
  `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,adr}.md`, `docs/ROADMAP.md`.
  `engram update`'s sync-as-removal cleans deployed copies.

## Capabilities

### New Capabilities

(none — applies the existing runbook/follow-frame capability to a new carrier)

### Modified Capabilities

- `please-doc-enumeration-gate`: requirements name "the please skill's Step 3" as the carrier of the
  enumeration grep; after retirement the carrier is the please runbook (sub-runbook) — carrier language
  updates, behavior (non-waivable grep, Gate A reviewer independently verifies and discovers,
  per-file disposition list) is unchanged.
- `write-memory-worker`: line ~120 references `please`; carrier language updated if it names the skill.

- `guidance-runbook-follow-frame`: the first-action requirement gains a generic phrase-two re-check
  (concrete ticket nouns or request paraphrase in the second phrase -> rewrite as a work-handling
  decision) and a second, non-dispatch example. Motivated by the shim-only please run (3.3), where 3/3
  agents kept the deliverable in phrase two and the runbook was never retrieved. (A third, verbatim
  user-message phrase was drafted as D11 and rejected: 0/95 cells changed.) The `red_flags` and
  wikilink-follow requirements are unchanged.

## Impact

- **Removed**: `agent-instructions/skills/please/SKILL.md` (and its directory) after gates pass.
- **Deferred (2026-09-20)**: retirement of `please/SKILL.md` is blocked on reliable runbook triggering and
  moves to the follow-on change `runbook-lexical-triggers`; this change is parked (see `tasks.md` Status).
- **Added**: please runbook + sub-runbooks in the production vault; a `please` eval task, skill-arm
  baseline, and conversion-fidelity report under `dev/eval/cumulative/runbook_vs_skill/phase2/`.
- **Docs/config**: live references listed above; historical references under `dev/eval` and archived
  openspec changes are left as-is.
- **Deployment**: `engram update` stops shipping the please skill directory and removes deployed copies
  in both Claude Code and Pi harnesses.
- **Shim**: `agent-instructions/guidance/shim.md` grows by ~559 bytes / ~97 words (re-check paragraph +
  one example). It changes second-phrase composition for EVERY runbook, so it deploys via
  `engram update --with-guidance` and needs headless RED/GREEN plus a route retrieval regression check.
- **Situation rewrite**: the top please runbook's `situation` is rewritten process-shaped (candidates
  scored in a scratch vault before any fixture edit).
- **Literal-phrase rule (D11)**: REJECTED 2026-09-20 (0/95 cells changed); shim draft reverted. Replaced
  by the follow-on change `runbook-lexical-triggers`.
- **Eval spend**: new fixture/baseline plus a shim-only probe run; estimate and confirm cost before
  running.
- **Issues**: closes #758. Siblings #759 (curate), #760 (recall/learn/write-memory promotion), #761
  (openspec skills) are out of scope.
