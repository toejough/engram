# Unified `/learn` Skill — Design

**Date:** 2026-05-10
**Status:** Approved (brainstorming phase)
**Replaces:**
- `~/.claude/skills/capturing-fleeting-notes/SKILL.md` (global)
- `~/.claude/skills/promoting-to-permanent-notes/SKILL.md` (global)
- `skills/learn/SKILL.md` (in-repo, current)
- `skills/remember/SKILL.md` (in-repo)

## Motivation

The agent-memory vault has been operated as a two-stage system: capture fleeting notes, then promote a subset to permanent notes. The two-stage rhythm is inherited from human zettelkasten practice, where the cost asymmetry (cheap to scribble, expensive to distill, lossy memory) makes deferred curation valuable.

For an LLM operator, both costs are roughly the same prose-generation step. Empirically, fleetings convert to permanents 1:1 — the promotion gate filters nothing. The two-stage process is ceremony.

A separate observation: in-repo skills `learn` and `remember` enforce quality gates (Recurs, Right-Home) before writing to engram's internal memory store. The global vault skills enforce a weaker "information→knowledge" criterion. The vault contains notes that violate the stricter gates (project-named situations, hindsight-baked framings).

This design collapses the two-stage flow into a single skill that applies the stricter gates at write time.

## Goals

1. Replace four existing skills with one unified skill named `learn`.
2. Apply the **Recurs** gate (reject project-specific situations) and **activity-and-domain framing** discipline at write time.
3. Support **autonomous triggering at task boundaries** alongside user invocation.
4. Vault-only backend; remove the orphaned `engram learn` subcommand and any code reachable only from removed paths.

## Non-goals

- **Vault-search-before-write.** Linking remains limited to notes loaded in the current context. Deferred; tracked separately.
- **Source attribution (`--source human|agent`).** Vault `source` frontmatter stays as session-log citation; no human/agent dichotomy.
- **Right-Home gate.** The grep-the-docs-for-prior-coverage step from in-repo `/remember` is not folded in.
- **DUPLICATE / CONTRADICTION return codes from `engram promote`.** Overlaps with vault-search; deferred together.
- **Retroactive sweep of existing permanents.** The 65 existing permanents are not re-evaluated against the new gates. New writes only.

## Architecture

One skill, `learn` (slash command `/learn`). Two trigger paths:

- **User-invoked** — phrases such as `/learn`, "remember this", "save that for later", "write up what we just did". Input grain is determined from context: a single observation when the user flags a specific moment; a session-batch sweep when the user asks at the end of a chunk of work.
- **Autonomous at task boundaries** — at the end of a discrete task (feature shipped, bug fixed, plan step closed, direction changed), the skill self-fires to sweep the just-completed work using the same gate sequence and write discipline.

Backend: vault only, via the existing `engram promote {feedback|fact|moc}` binary. Same Luhmann ID assignment under vault lock. Same permanent and MOC formats. Same `Related to:` bullets with per-link rationale.

No fleeting tier. No `Fleeting/` directory. No `--delete-fleeting` flag.

## Workflow

```
trigger
  ├─ user-invoked: /learn, "remember this", "save that", "write up what we just did"
  └─ autonomous: end of task (feature shipped, bug fixed, plan step closed, direction changed)

1. Identify candidates
     scan in-context conversation (default) or session logs (when source isn't loaded)
     for: user corrections, failed approaches, discovered facts, recurring patterns

2. For each candidate, run gates in order — single failure drops it:
     a. Recurs       — strip situation to activity+domain; reject if project-specific
     b. Activity+domain framing — situation phrased as agent would query before lesson is known
     c. Knowledge bar — restateable as a principle with applicability beyond the originating event

3. For each survivor, decide disposition:
     - new permanent (feedback or fact)
     - merge into existing (sharpens wording / adds example, no new claim)
     - split (one candidate bundles multiple principles → multiple permanents)

4. Decide Luhmann position per write:
     - find most-related existing note → choose relation: continuation | sibling | top
     - the binary computes the actual ID under vault lock

5. Draft body in LLM voice:
     - formulaic first line (Lesson learned / Information learned)
     - Related to: bullets with per-link rationale (linking to what's in context)

6. Cluster check (judgement, no threshold):
     - if a real framing paragraph emerges across ≥2 notes, draft a MOC
     - if not, skip

7. Write via engram promote {feedback|fact|moc} — one tool-use block, parallel calls

8. Report: candidates considered, gates passed/failed (with reasons),
          permanents written, MOCs written/updated, contradictions surfaced
```

### Gate definitions

**a. Recurs.** Strip the situation to *activity + domain*. Reject if it names:
- this project (engram / traced / etc.), its internals, or its architecture
- phase numbers, issue IDs, commit hashes, or dates
- one-time events ("user said X today"), diary entries, or status snapshots

An agent working on an unrelated project (web app, game, data pipeline) should plausibly hit the same situation.

**b. Activity-and-domain framing.** The `situation:` field describes what an agent would be embarking on, framed as it would be queried *before* the lesson is known. No hindsight, no diagnosis-as-situation.

| Bad (bakes in hindsight) | Good (activity + domain) |
|---|---|
| "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |

**c. Knowledge bar (zettelkasten.de).** "Information is dead and contextless; knowledge adds relevance and context. Translate information into knowledge by enriching it with applicability." A candidate that merely describes what happened is information; it converts only when restated as a principle with applicability beyond the originating event.

Gate order is **Recurs → Framing → Knowledge** because Recurs is the cheapest to evaluate (lexical check on situation string) and rejects the largest share of candidates. Framing is the next-cheapest (rephrasing check). Knowledge requires the most synthesis.

### Autonomous mode

Same gate sequence, same write discipline. Distinction is only the trigger — no user prompt before write. Same `Related to:` linking limited to in-context notes (vault-search deferred). Same MOC judgement.

### Discard path

Implicit. Failing any gate means "don't write." No separate disposition, no escape-hatch type, no working-memory tier.

### Contradictions

When a new permanent contradicts an existing one, write the new permanent with a `Related to:` bullet whose rationale names the discrepancy. Also surface in the report. Don't smooth.

## File layout and migration

**Repo path:**
```
skills/learn/
  SKILL.md
```

**Global symlink (dev):**
```
~/.claude/skills/learn -> /Users/joe/repos/personal/engram-worktrees/opencode-plugin/skills/learn
```

**Migration steps:**

1. Write `skills/learn/SKILL.md` (the new unified skill).
2. Replace the existing in-repo `skills/learn/SKILL.md` with the new content.
3. Delete `skills/remember/` directory.
4. Delete `~/.claude/skills/capturing-fleeting-notes/`.
5. Delete `~/.claude/skills/promoting-to-permanent-notes/`.
6. Create the global symlink at `~/.claude/skills/learn -> <repo>/skills/learn`.
7. Audit CLAUDE.md, other skills, and docs for references to the four old skill names; update to `learn`.
8. **Binary cleanup:** remove the `engram learn` subcommand and any internal packages/types reachable only from it (SBIA store, related dedup/contradiction logic). Remove the `--delete-fleeting` flag on `engram promote`. Audit `cmd/engram/` and `internal/` for any other code reachable only from removed paths.

The plan phase enumerates concrete deletions after reading the current binary surface.

## Testing

Per `superpowers:writing-skills` TDD discipline:

- **Baseline behavioral test (RED).** Define a small set of representative scenarios:
  - User says "remember this" about a project-specific bug fix (current behavior: capture as fleeting / write a project-named permanent; new behavior: Recurs-gate rejects, no write).
  - User says "write up what we just did" after a session of generalizable work (current behavior: capture all as fleetings; new behavior: each candidate runs the gate sequence; survivors written as permanents).
  - Autonomous trigger at end of plan step (current behavior: nothing; new behavior: same gate sequence applied to recent session work).
  - Single-observation invocation where situation passes Recurs but fails Framing (rejected).
  - Single-observation invocation where situation passes both gates but the candidate is information not knowledge (rejected).

  Run each against the current skills; confirm wrong behavior.
- **Update skill (GREEN).** Edit `SKILL.md` until each scenario produces correct behavior.
- **Pressure tests.** Run realistic mixed-quality candidate sets through the gate sequence; verify rejections cite specific gate failures; verify Luhmann placement matches relation choice; verify MOC creation triggers only on genuine framing.

The skill itself is the artifact under test; binary cleanup is validated by `targ check-full` after each deletion.

## Open questions and risks

- **Recall continues to find pre-Recurs-gate permanents.** Existing notes that name "engram", "Task 8", etc. remain in the vault and will keep surfacing in recall. Accepted as a non-goal here; a retroactive sweep can be a follow-up.
- **Autonomous trigger frequency.** "End of task" is judgement-dependent. The skill must avoid firing on trivial micro-tasks (one-file edits) without firing too rarely on real work. The skill prose needs explicit examples of trigger / no-trigger boundaries.
- **Binary cleanup scope creep.** Removing `engram learn` may cascade into removing larger swaths of `internal/`. Plan phase scopes this concretely.

## Success criteria

- All four obsolete skills removed from disk.
- A single `skills/learn/SKILL.md` exists and is symlinked globally.
- A pressure-test run on five representative scenarios produces the expected gate outcomes.
- `targ check-full` passes after binary cleanup.
- No code in `cmd/engram/` or `internal/` is reachable only from removed paths.
