---
name: learn
description: >
  Use after completing a task, finishing work, changing direction, or when the
  user says "review what we learned" or /learn. Should be called after
  implementation, after resolving a bug, after completing a plan step.
  Reviews the recent session for learnable moments.
---

You are reviewing the recent session for things worth remembering.

## Flow

### Step 1: Identify learnable moments

Review your current session for:
- **User corrections** — the user told you to do something differently
- **Failed approaches** — something was tried and didn't work
- **Discovered facts** — new knowledge about the codebase, tools, or domain
- **Patterns** — recurring behaviors that should be codified

### Step 2: Quality gate

For each candidate, check every gate in order. A single failure drops the candidate.

#### 1. Recurs — would an agent on an unrelated project face this?

Strip the situation to **activity + domain** ("implementing Claude Code hooks", "writing Go tests with context"). If the situation names:

- This project (engram / traced / etc.), its internals, or its architecture
- Phase numbers, issue IDs, commit hashes, or dates
- One-time events ("user said X today"), diary entries, or status snapshots

…it fails Recurs. An agent working on a web app, a game, or a data pipeline should plausibly hit this situation too.

#### 2. Actionable — does it change what an agent would DO?

A passing memory names a concrete action. Fails on vague observations ("things can go wrong"), inert facts ("X exists"), or raw debug logs.

#### 3. Right home — is memory the correct place?

**a. Name the alternative home:** code, a doc, a skill, CLAUDE.md, a `.claude/rules/*.md` file, or a spec/plan under `docs/`.

**b. Verify against the claimed home:**

```bash
git log --since='14 days ago' --name-only --pretty=format: -- \
    docs/ specs/ plans/ skills/ CLAUDE.md .claude/ | sort -u
```

Read the listed files; grep/scan for the candidate's content.

**c. Act on the result:**

| Verification outcome | Action |
|---|---|
| Home contains it AND surfaced/was applied this session (via read, search, recall, or we just wrote it) | **Move on.** Home owns it; working. |
| Home contains it BUT didn't surface this session (we had to re-learn it) | **Persist new memory** for task-triggered reinforcement. If engram dedup returns DUPLICATE, **update** the existing memory's situation (`engram update --name ... --situation "..."`) so it surfaces next time. |
| Home lacks the lesson, or no home fits | **Persist new memory.** Engram is the write target we control. |

### Step 3: Draft memories

For each surviving candidate:
- Corrections/failures → feedback (SBIA)
- Knowledge/patterns → fact (situation + subject/predicate/object)

**Situation field:** Describe the task the agent would be embarking on, in terms of **activity + domain** — not the diagnosis, symptom, or fix. A good situation matches how `/prepare` queries would be phrased *before* the lesson is known.

| Bad (bakes in hindsight) | Good (activity + domain) |
|---|---|
| "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |

### Step 4: Persist

**Autonomous by default** — at task boundaries, persist with `--source agent`.
**Interactive when user invokes /learn** — present findings for approval first.

```bash
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source agent
engram learn fact --situation "..." --subject "..." --predicate "..." --object "..." --source agent
```

### Step 5: Handle results

- **CREATED** — Done. If interactive, confirm to user.
- **DUPLICATE** — System knew this but didn't surface. Diagnose why /recall/prepare missed it. Broaden the existing memory's situation with `engram update --name <name> --situation "..."`. Never dismiss — surfacing failed.
- **CONTRADICTION** — Interactive: ask user (update, replace, keep both). Autonomous: skip.
