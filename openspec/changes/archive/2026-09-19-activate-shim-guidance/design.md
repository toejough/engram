## Context

`shim.md` exists, is tested (`shim-question-stop-genuine-ambiguity`, closing #762), and is already
deployed by `engram update --with-guidance` to every harness's engram-owned root — confirmed this
session: `~/.claude/engram/guidance/shim.md` present, `~/.claude/engram/shim.md` compat symlink
present, byte-identical to the repo source. What's missing is purely the import line in a real
`CLAUDE.md`/`AGENTS.md`; this session's own `~/.claude/CLAUDE.md` currently imports only
`recall.md`, `delegate.md`, `learn.md` (verified by `grep "@~" ~/.claude/CLAUDE.md`).

The archived `runbook-shim-follow-frame` change (D0) already decided the shape this activation
must take: *"`recall.md`/`delegate.md`/`learn.md` stay as they are for sessions that still install
the skills; reconciliation belongs to the later recall-replacement effort."* That later effort is
issue #760 — scoped to promoting `recall`/`learn`/`write-memory` to production runbooks, retiring
their `SKILL.md` files, and only then deciding whether `shim.md` can fully replace the three
guidance files. None of that promotion has happened yet, so full replacement is not available to
this change; this change activates the additive form D0 already anticipated.

## Goals / Non-Goals

**Goals:**
- Decide and document the concrete activation approach (additive import, not replacement) with the
  reasoning that rules out replacement today.
- Verify the existing `engram update --with-guidance` deployment path already supports this without
  any code change — confirm, don't assume.
- Produce the exact import line(s) for `~/.claude/CLAUDE.md` and, if in scope, `~/.pi/agent/`'s
  `AGENTS.md`, as a documented step for the orchestrator to apply directly (outside this change's
  own artifacts, since the target file is the user's personal config, not a repo file).

**Non-Goals:**
- Promoting `recall`/`learn`/`write-memory` to production runbooks or retiring their `SKILL.md`
  files — #760's larger, separately-deferred scope.
- Any code change to `internal/update/` — verified unnecessary (see Decisions, D2).
- Resolving the `/recall glance` naming gap once `shim.md` is the sole guidance (#760 names this;
  it does not arise under an additive activation, since `recall.md`'s own `/recall glance` cue
  stays live and unchanged).

## Decisions

**D1. Additive activation: add `shim.md`'s import alongside the existing three, remove nothing.**
Alternatives considered:
- *Full replacement* (drop `recall.md`/`delegate.md`/`learn.md`, keep only `shim.md`) — rejected.
  `shim.md`'s own body does not reproduce their content: no 10-phrase/clustering/coverage-judgment
  procedure (`recall.md`'s `/recall glance`/`/recall deep`), no four-kind crystallization scan
  (`learn.md`), no orchestration-vs-object-level routing doctrine (`delegate.md`). Removing them
  now would silently drop that behavior for every request, not just runbook-shaped ones.
  `shim.md`'s own header comment already states this file is "NOT a replacement... which stay as-is
  for sessions that still install the recall/learn/write-memory skills" — confirming this is the
  file's own documented intent, not a new interpretation.
- *Wait for #760 and do a single combined cutover* — rejected as this change's own scope, though
  it remains the eventual end state. Waiting blocks every runbook conversion already landed or
  in-flight (route, and future please/curate/openspec ones) from ever being followable in a real
  session, for an indefinite period tied to a much larger, separately-scoped effort. Activating
  additively now unblocks those immediately without foreclosing the later full cutover.

**D2. No `internal/update/` code change — verified, not assumed.** `update-deploy-sync`'s existing
"Guidance opt-in gates management, not removal" requirement already covers `shim.md` exactly like
`recall.md`/`delegate.md`/`learn.md`: it deploys to the engram-owned root when `--with-guidance` is
passed (already run and confirmed working this session), and never touches the harness's own
config file. There is no requirement gap to close — the missing step has always been the manual
import line, which `engram update` correctly does not write on the user's behalf (per that same
requirement — auto-editing the user's own dotfile would violate "gates management, not removal" by
making an unrequested edit). This change's proposal.md accordingly lists no Modified Capabilities.

**D3. The dotfile edit is documented, not executed, by this change's own artifacts.** Per this
session's standing convention (vault note 637: work with no tracked-file footprint in the repo
belongs to the orchestrator, not a change's task automation) and the fact that `~/.claude/CLAUDE.md`
lies outside `allowedEditRoots` for this OpenSpec change, tasks.md states the exact line(s) to add
as an instruction, not a step this change applies itself.

## Risks / Trade-offs

- **[Risk] Redundant instruction overlap between `shim.md` and `recall.md`.** `shim.md`'s "run
  `engram query` before your first tool call, on every request" and its own "Re-entry: four moments
  to query again" section overlap substantially with `recall.md`'s "`/recall glance` at these
  cues" — both name near-identical situational-query moments (proposed-approach endorsement,
  before declaring done, after an unexplained failure, before building a new approach) in similar
  wording. → Mitigation: this is redundancy, not contradiction — both files converge on the same
  behavior from different angles (shim.md is universal/every-request; recall.md is moment-keyed).
  Not a blocker for this change; a candidate cleanup once #760's fuller reconciliation happens, not
  before.
- **[Risk] Activating `shim.md` changes real-session behavior for every future runbook-shaped
  task, not just this session's route conversion.** → Mitigation: this is the intended effect —
  `shim.md`'s follow-frame is already validated (D8-equivalent bars met for route, history-rewrite,
  bisect-before-fix in the archived eval and this session's rerun). No new behavior is introduced
  by this change beyond making already-tested behavior reachable.
- **[Trade-off] This change has no code and no spec-level requirement change** — it is a decision +
  verification + a documented manual step. Proposal.md's Capabilities sections are correspondingly
  empty; this is a deliberate, honest reflection of the change's actual shape, not an oversight.

## Migration Plan

1. Confirm (re-verify at apply time, not just cite this session's earlier check) that
   `~/.claude/engram/guidance/shim.md` and its compat symlink are current and byte-identical to
   `agent-instructions/guidance/shim.md`; re-run `engram update --with-guidance` first if not.
2. Add `@~/.claude/engram/shim.md` to `~/.claude/CLAUDE.md`'s existing `@~/.claude/engram/recall.md`
   / `@~/.claude/engram/delegate.md` / `@~/.claude/engram/learn.md` block — same line shape, same
   compat-symlink resolution convention the other three already use.
3. If `~/.pi/agent/`'s `AGENTS.md` is in scope for this activation (check whether it currently
   imports the other three guidance files the same way `CLAUDE.md` does before assuming symmetry),
   add the matching `@~/.pi/agent/guidance/shim.md` line.
4. No rollback machinery beyond removing the added import line(s) — this is a single-line,
   trivially-reversible dotfile edit outside version control.

## Open Questions

- Does `~/.pi/agent/`'s `AGENTS.md` currently import `recall.md`/`delegate.md`/`learn.md` the same
  way `~/.claude/CLAUDE.md` does? Not verified this session (no Pi harness inspected) — tasks.md
  should check before assuming step 3 above applies symmetrically.
