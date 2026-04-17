# Learn/Remember Quality Gate Verification — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sharpen the three-gate quality filter in `skills/learn/SKILL.md` and `skills/remember/SKILL.md` so it drops project-internal and already-captured memories (#583), verifies the claimed alternative home before dropping (#582), and catches the five bad-memory categories from the April 15 triage (#581). Closes #581, #582, #583.

**Architecture:** Pure skill-content change. Two SKILL.md files get a rewritten Step 2 (Quality gate with verification procedure) and a tightened Step 3 (Draft memories). The `/learn` flow is autonomous; the `/remember` flow asks the user on rows 2/3 of the verification outcomes. No Go-side write-path changes. The 2026-04-16 integration spec's "write path untouched" non-goal is respected — verification is agent-executed via `git log` + file reads + grep, prescribed in skill prose.

**Tech Stack:** Markdown (SKILL.md), `git log`, agent-executed shell commands. Skill editing discipline enforced by `superpowers:writing-skills` (per `CLAUDE.md` Skill Editing rule).

**Spec:** `docs/superpowers/specs/2026-04-17-learn-remember-quality-gate-verification-design.md`

---

### Task 1: Update `skills/learn/SKILL.md` via superpowers:writing-skills

**Files:**
- Modify: `skills/learn/SKILL.md` (full replacement of body below the frontmatter)

**Sub-skill:** `superpowers:writing-skills` is REQUIRED per `CLAUDE.md` Skill Editing rule. It enforces baseline pressure test (RED) → edit → verification pressure test (GREEN).

---

- [ ] **Step 1: Invoke `superpowers:writing-skills`**

Announce: "I'm using the writing-skills skill to edit `skills/learn/SKILL.md` per the quality-gate verification spec." Then call the skill.

---

- [ ] **Step 2: Baseline behavioral test (RED) — PT-1, PT-3 must fail on current skill**

Run four pressure tests against the *current* (pre-edit) `skills/learn/SKILL.md`. For each, dispatch a fresh general-purpose agent with only the current SKILL.md text loaded as its operating guide, hand it the candidate memory scenario, and record what the agent decides.

**PT-1 (#583) candidate:**
```
Situation: "When designing dedup or gating behavior for a memory or notes
system that overlaps with content in higher-authority external sources"
Subject: Default behavior when overlap is detected
Predicate: should be
Object: save the candidate as task-triggered reinforcement rather than block
```

Agent session context: "You just committed a spec paragraph at
`docs/superpowers/specs/2026-04-16-engram-claude-memory-integration-design.md`
that articulates this rationale verbatim."

**Expected on CURRENT skill:** agent persists the memory (failure — should drop at Recurs / Right home).

**PT-2 (#582) candidate:**
```
Type: feedback
Situation: "When implementing new Claude Code features"
Behavior: "Adding env-var feature flags speculatively for rollback"
Impact: "Unnecessary complexity, YAGNI violation"
Action: "Don't add env-var toggles — just change the code"
Claimed home: CLAUDE.md (already covered by YAGNI rule)
```

Agent session context: "You drafted this feedback memory but the agent's
judgement said 'belongs in CLAUDE.md — already covered.'"

**Expected on CURRENT skill:** agent drops without reading CLAUDE.md (failure — should verify and, if absent, persist).

**PT-3 (#581) candidate:**
```
Situation: "When checking Phase 2 implementation status"
Subject: ACK-wait Phase 2 readiness
Predicate: requires
Object: verification via the engram chat protocol before claiming complete
```

**Expected on CURRENT skill:** ambiguous — may persist (failure — should drop at Recurs for phase-locked framing).

**PT-4 (#582 row-2 variant) candidate:**
```
Type: feedback
Situation: "When running quality checks in this project"
Behavior: "Used `targ check` instead of `targ check-full`"
Impact: "Missed multiple errors on first pass"
Action: "Use `targ check-full` to see all errors at once"
Agent session context: "CLAUDE.md has this guidance verbatim. The agent
didn't recall it when running checks."
```

**Expected on CURRENT skill:** agent drops (belongs in CLAUDE.md) without flagging that the existing CLAUDE.md coverage failed to surface. Lesson lost as reinforcement.

Record the baseline outcomes. If any pressure test unexpectedly passes on the current skill, pause and investigate before proceeding.

---

- [ ] **Step 3: Apply the edit (full-body replacement below frontmatter)**

Overwrite the body of `skills/learn/SKILL.md` (everything below the frontmatter `---` closing line) with this exact content:

````markdown
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
````

Keep the frontmatter exactly as it is today:

```yaml
---
name: learn
description: >
  Use after completing a task, finishing work, changing direction, or when the
  user says "review what we learned" or /learn. Should be called after
  implementation, after resolving a bug, after completing a plan step.
  Reviews the recent session for learnable moments.
---
```

---

- [ ] **Step 4: Verification pressure tests (GREEN) — PT-1, PT-2, PT-3, PT-4 all expected**

Re-run the four pressure tests from Step 2 against the *edited* `skills/learn/SKILL.md`. For each, dispatch a fresh general-purpose agent with only the edited SKILL.md text loaded as its operating guide, hand it the same candidate memory scenario, and record the decision.

Expected outcomes:

| PT | Expected outcome |
|---|---|
| PT-1 (#583 fresh-write) | **Drops** at Recurs ("designing dedup behavior for a memory system" = engram-internal). If Recurs weakly passes, Right home row 1 (home has it + we just wrote it) also drops. |
| PT-2 (#582 home-missing) | **Persists new memory.** Right home verification reads CLAUDE.md, grep finds no YAGNI/env-var-toggle coverage → row 3 → persist. |
| PT-3 (#581 phase-locked) | **Drops** at Recurs ("checking Phase 2" names phase number). |
| PT-4 (#582 home-had-it-didn't-surface) | **Row 2 outcome.** Agent persists new memory for reinforcement; if engram dedup returns DUPLICATE on existing `targ-check-full-not-check`, agent updates its situation via `engram update`. |

If any pressure test does not produce the expected outcome, iterate on the skill text (without widening scope) until it does. Re-record the outcome.

---

- [ ] **Step 5: Stage + commit**

```bash
git add skills/learn/SKILL.md
git commit -m "$(cat <<'EOF'
feat(skills/learn): sharpen quality gate with home-verification

Recurs gate now applies an explicit arbitrary-project litmus: situations
naming this project, its internals, phase numbers, or issue IDs fail.
Right home gate requires verifying the claimed alternative home (via
recent git log + grep) and acts on one of three outcomes — move on,
reinforce via memory, or persist to fill a gap.

Closes a subset of #581/#582/#583 (learn side; remember updated separately).

AI-Used: [claude]
EOF
)"
```

---

### Task 2: Update `skills/remember/SKILL.md` via superpowers:writing-skills

**Files:**
- Modify: `skills/remember/SKILL.md` (full replacement of body below frontmatter)

**Sub-skill:** `superpowers:writing-skills` is REQUIRED per `CLAUDE.md` Skill Editing rule.

**Why separate from Task 1:** distinct file, distinct commit. Parallelizable with Task 1 but sequentialized here for simpler review (reviewer can verify /learn behavior first, then /remember).

---

- [ ] **Step 1: Invoke `superpowers:writing-skills`**

Announce the skill invocation.

---

- [ ] **Step 2: Baseline behavioral test (RED) — adapted pressure tests for interactive flow**

Run three pressure tests against the *current* (pre-edit) `skills/remember/SKILL.md`. Dispatch a fresh general-purpose agent with only the current SKILL.md as its operating guide. Simulate interactive invocations.

**PT-R1 (interactive analog of PT-2):** user says: "Remember: don't add env-var toggles speculatively for rollback — YAGNI."
- **Expected on CURRENT skill:** agent drops ("belongs in CLAUDE.md") without reading CLAUDE.md (failure).

**PT-R2 (interactive analog of PT-1):** user says: "Remember: when designing dedup behavior for a memory system, default to task-triggered reinforcement."
- **Expected on CURRENT skill:** agent persists (failure — should drop at Recurs).

**PT-R3 (interactive analog of PT-4):** user says: "Remember: always use `targ check-full` instead of `targ check`." Session context: CLAUDE.md has this verbatim.
- **Expected on CURRENT skill:** agent drops without flagging the surfacing failure (failure — should offer user the reinforce/note option).

Record baseline outcomes.

---

- [ ] **Step 3: Apply the edit (full-body replacement below frontmatter)**

Overwrite the body of `skills/remember/SKILL.md` (everything below the frontmatter `---` closing line) with this exact content:

````markdown
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
| Home contains it AND surfaced/was applied this session (via read, search, recall, or we just wrote it) | **Move on.** Home owns it; working. Tell the user and suggest no save. |
| Home contains it BUT didn't surface this session (we had to re-learn it) | **Ask the user:** "This is already in `<home>` — reinforce via memory, or just note it?" If reinforce: persist new memory. If engram dedup returns DUPLICATE, update the existing memory's situation via `engram update --name ... --situation "..."`. |
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
````

Keep the frontmatter exactly as it is today:

```yaml
---
name: remember
description: >
  Use when the user says "remember this", "remember that", "don't forget",
  "save this for later", or /remember. Captures explicit knowledge as
  feedback or fact memories with user approval.
---
```

---

- [ ] **Step 4: Verification pressure tests (GREEN) — PT-R1, PT-R2, PT-R3 all expected**

Re-run the three pressure tests from Step 2 against the *edited* `skills/remember/SKILL.md`. Dispatch a fresh general-purpose agent with only the edited SKILL.md as its operating guide.

Expected outcomes:

| PT | Expected outcome |
|---|---|
| PT-R1 (home-missing) | Agent runs verification, reads CLAUDE.md, grep finds nothing → row 3 → asks user: "The home for this would be CLAUDE.md but it's not there. Save to memory anyway?" |
| PT-R2 (engram-internal) | Drops at Recurs. Tells user why: situation names engram implementation work; no arbitrary-project agent faces this. Suggests no save. |
| PT-R3 (home-had-it-didn't-surface) | Row 2 → asks user: "This is already in CLAUDE.md — reinforce via memory, or just note it?" |

If any pressure test does not produce the expected outcome, iterate on the skill text (without widening scope) until it does.

---

- [ ] **Step 5: Stage + commit**

```bash
git add skills/remember/SKILL.md
git commit -m "$(cat <<'EOF'
feat(skills/remember): sharpen quality gate with home-verification

Mirrors the /learn gate rewrite: Recurs applies arbitrary-project litmus;
Right home requires verifying the claimed home via recent git log + grep.
/remember's interactive flavor defers rows 2 and 3 of the verification
table to user choice (reinforce vs note; save anyway vs skip).

Closes #581, #582, #583.

AI-Used: [claude]
EOF
)"
```

---

### Task 3: Cross-skill verification + issue closure

**Files:** None modified. This task validates the combined state and closes the three issues.

---

- [ ] **Step 1: Run a full pressure-test sweep**

Dispatch one final general-purpose agent with BOTH edited skills loaded. Confirm:

- `/learn` PT-1 drops at Recurs.
- `/learn` PT-2 persists via row 3.
- `/learn` PT-3 drops at Recurs.
- `/learn` PT-4 updates existing memory via row 2.
- `/remember` PT-R1 prompts user via row 3.
- `/remember` PT-R2 drops at Recurs with explanation.
- `/remember` PT-R3 prompts user via row 2.

All seven scenarios must produce the expected outcome.

---

- [ ] **Step 2: Length sanity check**

Run `wc -l skills/learn/SKILL.md skills/remember/SKILL.md` and compare to the pre-edit baseline. Target: same or shorter. If noticeably longer (>10% increase), iterate once to trim non-load-bearing prose.

---

- [ ] **Step 3: Close GitHub issues**

```bash
gh issue close 581 --comment "$(cat <<'EOF'
Fixed in `skills/learn/SKILL.md` and `skills/remember/SKILL.md` via the
unified quality-gate rewrite in docs/superpowers/specs/2026-04-17-learn-remember-quality-gate-verification-design.md.

- Recurs's new arbitrary-project litmus catches phase/issue/date-locked,
  one-time events, and project-internal framing.
- Actionable's sharpened fail criteria catch vague/abstract candidates.
- Right home's verification procedure catches redundancy with committed
  docs/skills/CLAUDE.md and prevents silent drops when the home lacks the
  content.

Pressure tests PT-1, PT-2, PT-3, PT-4, PT-R1, PT-R2, PT-R3 pass.
EOF
)"

gh issue close 582 --comment "$(cat <<'EOF'
Fixed via the unified quality-gate rewrite. The Right home verification
procedure now requires reading the claimed alternative home before
dropping. The three-row verification-outcomes table covers:

- Row 1: home has it, surfaced/applied → move on (no silent loss since
  the content already landed).
- Row 2: home has it but didn't surface → persist memory (reinforcement)
  or update existing memory's situation. Interactive: ask user.
- Row 3: home lacks the content → persist new memory (no silent loss).

Pressure tests PT-2, PT-4, PT-R1, PT-R3 pass.
EOF
)"

gh issue close 583 --comment "$(cat <<'EOF'
Fixed via the unified quality-gate rewrite. Recurs's arbitrary-project
litmus drops engram-implementation memories framed as generic patterns.
Right home row 1 additionally drops candidates whose content is already
in a just-committed artifact.

Pressure test PT-1 (the exact dedup-design-pattern candidate from the
issue) drops at Recurs.

Side observation about `engram delete`: kept as follow-up. The currently-
stuck memory can only be updated, not removed.
EOF
)"
```

---

- [ ] **Step 4: Report**

Emit a summary report to the user including: files changed, line counts before/after, all seven pressure-test outcomes, and three issue closures.

---

## Self-review

- **Spec coverage:** Goals 1-5 all have tasks. Non-goals are preserved (no Go, no new CLI, no writes to external homes). All four primary pressure tests from the spec map to Task 1/Task 2 steps.
- **Placeholder scan:** Steps contain literal SKILL.md content for both edits, literal pressure-test candidates, literal commit messages, literal close-issue comments. No "TBD" / "similar to above" / unnamed functions.
- **Type consistency:** Gate names (Recurs / Actionable / Right home), row numbers, PT IDs, and file paths are consistent across all tasks.
- **Length check:** Expected skill lengths after edit: `/learn` ~ 72 lines (vs 65 pre-edit, +11%), `/remember` ~ 74 lines (vs 59, +25%). `/remember`'s increase is from the three user-prompt variants in the Right home table. Task 3 Step 2 flags iteration if the increase exceeds 10% — for `/remember` this will likely trigger a trim pass on Step 5 prose or Step 4 comment lines.
