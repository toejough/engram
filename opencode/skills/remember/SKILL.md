---
name: remember
description: >
  Use when the user says "remember this", "remember that", "don't forget",
  "save this for later", or /remember. Captures explicit knowledge as
  feedback or fact memories with user approval.
---

The user wants to explicitly save something to memory.

## Flow

### Step 1: Classify

Determine what the user wants to remember:
- **Feedback** (behavioral): situation → behavior → impact → action
- **Fact** (knowledge): situation → subject → predicate → object
- Could be **multiple** memories (e.g., "DI means Dependency Injection, not Do It" = two facts)

### Step 2: Quality gate

For each candidate, check every gate in order. A single failure drops the candidate — tell the user why and suggest the right home.

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
| Home contains it AND surfaced in time this session (via /recall, /prepare, CLAUDE.md auto-load — OR we just wrote it into the home now); no re-learning needed | **Move on.** Home owns it. Tell the user and suggest no save. |
| Home contains it BUT didn't surface in time (we re-learned it or were corrected this session — reading the home during this verification doesn't count as surfacing) | **Ask the user:** "This is already in `<home>` — reinforce via memory, or just note it?" If reinforce: persist new memory. If engram dedup returns DUPLICATE, update the existing memory's situation via `engram update --name ... --situation "..."`. |
| Home lacks the lesson, or no home fits | **Ask the user:** "The home for this would be `<home>` but it's not there. Save to memory anyway?" If yes: persist new memory. |

### Step 3: Draft and present

For each surviving candidate, draft all fields and present for approval.

**Situation field:** Describe the task the agent would be embarking on, in terms of **activity + domain** — not the diagnosis, symptom, or fix. A good situation matches how `/prepare` queries would be phrased *before* the lesson is known.

| Bad (bakes in hindsight) | Good (activity + domain) |
|---|---|
| "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |

### Step 4: Save

```bash
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source human
engram learn fact --situation "..." --subject "..." --predicate "..." --object "..." --source human
```

### Step 5: Handle results

- **CREATED** — Confirm to user.
- **DUPLICATE** — System already knew this. Diagnose why /recall/prepare missed it. Broaden existing memory's situation via `engram update`. Never dismiss — surfacing failed.
- **CONTRADICTION** — Present conflict. Ask user: update existing, replace, or keep both.
