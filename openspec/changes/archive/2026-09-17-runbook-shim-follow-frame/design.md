## Context

The engram shim today is `agent-instructions/guidance/recall.md`: four decision-moment firing cues (plus the #735 task-start cue, drafted into eval fixtures only). It is find-only. Everything about *doing* memory lives in skills (`recall`, `learn`, `write-memory`, `route`, `please`, ~100 KB in `~/.claude/skills`). The runbook note kind (#719) carries `situation`, `done_when`, and a body of steps, and ranks by situation-similarity like a fact.

The eval checkpoint (2026-09-13, Sonnet 5, history-rewrite, n=3 per arm) measured the skill row at found 3 / restated-as-plan 3 / every step 3 / end result 2, and both note arms at restated 0 / end 0 despite found 3. The nine transcripts show the difference is the loading frame: a skill lands as the invoked instruction and the agent restates its steps before acting; a note lands as one item in a ~40-item query result and is consulted, not executed. Claude Code's docs confirm skills have no enforcement beyond the model reading the loaded body (hooks are the only deterministic layer), so "as reliable as skills" is a bar about what lands in context and in what frame.

Ultimate goal (Joe): every skill, including recall and learn, becomes a runbook; the shim is the only custom CLAUDE.md text. The design must therefore bootstrap without any engram skill present.

## Goals / Non-Goals

**Goals:**
- A shim that, with no engram skills installed, gets an agent to (a) run the first `engram query`, (b) recognize a matched runbook, (c) restate and execute its steps, (d) hold under pressure via a uniform behavioral floor, (e) treat `done_when` as the completion bar, (f) follow wikilinked runbooks transitively.
- A single optional schema field for runbook-specific red flags, so task-specific failure modes have a machine-known home and the shim can name what to do with them.
- Measure against the skill row on the same tasks with a shim-only configuration.

**Non-Goals:**
- Hooks or any harness-specific mechanism (engram must not rely on them).
- A roster of runbook situations loaded into CLAUDE.md at session start (does not scale with vault size).
- Per-runbook restatement of general behavioral rules (that is the drift the uniform floor prevents).
- Capture-time prompting for red flags (#751).
- Converting the remaining skills (route, please, curate) — a follow-on change once the frame holds.

## Decisions

**D0. The shim is a fresh guidance file, `agent-instructions/guidance/shim.md`, and in the runbook trial arm it is the only guidance in CLAUDE.md.** Alternative: extend `recall.md`. Rejected: `recall.md` names `/recall glance`, which does not exist in a shim-only session, so extending it would test a hybrid; and a fresh file keeps the "only custom text" claim literally true and measurable. `recall.md`/`delegate.md`/`learn.md` stay as they are for sessions that still install the skills; reconciliation belongs to the later recall-replacement effort.

**D1. The frame lives in the shim, not the recall skill.** Alternatives: (a) recall skill step after 2.5; (b) `engram query` output text. Rejected (a) because recall itself becomes a runbook; the frame must exist before any runbook is loaded. (b) is kept as a *reinforcement* (the payload may carry a one-line "kind: runbook — see the frame in your guidance") but cannot be the home, since the agent must know to run the query in the first place.

**D2. The shim has exactly four parts and nothing else.** First action (literal `engram query --phrase … ` before the first tool call on EVERY user request, no skill required — the trigger cannot be "multi-step" or "done before", since that is what the runbook lookup decides; task-init over-fire is accepted, measured 3.4x with a per-fire cost of one query); treatment of each returned kind (fact = knowledge and context; feedback = a standing instruction from the user, do not repeat the named mistake; runbook = the follow frame; chunk = raw evidence, fetch only to fill a gap) with the frame itself for runbooks; general behavioral floor; the four re-entry cues. The query is not filtered to runbooks: all kinds return, so the shim must say what each is for, in one line each. Replacing the deep recall procedure (ten phrases, clustering, coverage judgment, crystallization) with the shim is a separate, larger effort; recall stays a runbook the first query can surface (D1). Wording follows the measured rules: name the action, not the purpose (vault note 137); forbid the substitute explicitly; implementation-intention phrasing.

**D3. Frame contents (adapted from writing-skills, Anthropic's authoring guide, skill-creator, and the exemplar skills).** Announce the runbook by name before the first action. Restate its steps as the plan; where the harness has a todo list, one todo per step; do them in order. `done_when` is the completion bar: verify it holds, do not assume. If a step is unclear or you would deviate from it, stop and ask — a question stop is a clarity signal for the caller (vault note 1030), never "proceed anyway". A wikilinked runbook is fetched with `engram show` and followed the same way. A matched runbook is shown in full (`engram show`) when the query payload truncates it.

**D4. General behavioral floor, uniform across runbooks.** The urge to shortcut a step is the cue to reread it, not a reason. Substituting a related action for the named one is a skip. The letter of a step is its spirit. These are the rules a skill's Common-Mistakes/Red-Flags sections restate locally; here they are stated once.

**D5. Runbook-specific red flags: a structured field (option 1), not a body heading.** New optional frontmatter field `red_flags` (list of strings) on runbook notes; `engram learn runbook --red-flag <text>` (repeatable); rendered in query/show payloads; `write-memory` passes it through. The shim says: read `red_flags` before starting; when one fires, stop and reread the step. Rejected the heading convention: nothing enforces it and general-rule drift creeps back through free-form bodies. Field name settled as `red_flags` (see Settled, below).

**D6. The frame keys on `kind: runbook`.** A fact with the same body gets no frame. This preserves the eval's original question (does a vanilla fact match?) and gives the runbook kind a reason to exist beyond schema.

**D7. Eval configuration is shim-only.** No engram skills in the trial config dir; recall and learn present in the trial vault as runbooks (converted from their SKILL.md bodies, split into a top runbook plus sub-runbooks if the body exceeds what a query item comfortably carries); the shim in CLAUDE.md. Otherwise the test measures a hybrid that will not exist.

**D8. Pass bar, pre-registered.** History-rewrite first, runbook arm n=3: found 3, restated 3, every step 3, end result within one trial of the skill's 2/3. Question stops are reported as instruction-clarity findings and the shim amended before the next stage (Joe: "stop and really validate" staging). Then bisect-before-fix and route.

## Risks / Trade-offs

- [The recall runbook does not rank for shim-only phrasing, so the first query never surfaces "how to recall"] → the first-action wording includes the two phrase shapes the runbook's situation is written against (the task in your words; the situation kind); measured in stage 1 before spend on later stages.
- [A 26 KB procedure does not survive as one query item] → top runbook + wikilinked sub-runbooks (D3 transitive follow); `engram show` for full content.
- [Prose mechanisms asymptote below strict bars (vault note 198)] → the bar is parity with the skill row, itself prose-only, not 100%.
- [Fact carrier lacks `tags:` (checkpoint validation)] → rebuild the fact carrier preserving tags before the rerun so schema is the only difference.
- [Over-fire of the first action on trivial requests] → accepted by design: the fire-unit is task-init (audited 3.4x, LEDGER `739-audit-fire-unit-over-fire`), not per-tool-call (the 147x–380x that rejected hooks), and the per-fire cost is one `engram query` with no crystallization; measure wall-clock per fire in stage 1 and report it.
- [The skill arm carried a behavioral layer the notes lacked, so parity may need the red_flags field populated] → the history-rewrite runbook carrier gets its skill-derived red flags moved into `red_flags`; the fact carrier does not (D6).

## Open Questions

- Whether `engram query` output should carry the one-line frame reminder (D1 reinforcement) or stay pure data.
- How much of the learn runbook's 10-step procedure needs splitting before it ranks and loads cleanly.

## Settled

- Field name: `red_flags` (not `watch_for`) — implemented in learn-runbook-capture and
  recall-runbook-surfacing spec deltas: optional frontmatter list on runbook notes, populated via
  repeatable `engram learn runbook --red-flag <text>`.
