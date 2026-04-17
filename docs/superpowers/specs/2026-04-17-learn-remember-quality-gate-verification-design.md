# Learn/Remember Quality Gate Verification

**Date:** 2026-04-17
**Status:** Spec — awaiting plan
**Scope:** Sharpen the three-gate quality filter in `skills/learn/SKILL.md` and `skills/remember/SKILL.md` so it:
  1. Drops project-specific implementation memories already captured in committed specs (#583).
  2. Verifies the claimed alternative home actually contains the lesson before dropping (#582).
  3. Catches the five bad memory categories identified in the April 15 triage (#581).

Closes #581, #582, #583.

---

## Context

Two complementary skills — `/learn` (autonomous, task-boundary review) and `/remember` (interactive, explicit save) — share a quality gate consisting of three yes/no questions: **Recurs**, **Actionable**, **Right home**. The gate was added 2026-04-16 after the April 15 memory triage (107 deletes, 106 rewrites).

Three failure modes have since been observed:

1. **#583** — memories describing engram-internal implementation work get rephrased as generic patterns and persisted. Their rationale usually lives in a just-committed spec; memory creation duplicates the spec. Current gate: neither Recurs ("will a future agent face this?") nor Right home ("is memory the only place?") was applied rigorously enough to catch the case.
2. **#582** — Right home drops candidates that "belong in CLAUDE.md" without reading CLAUDE.md to verify. When the home lacks the lesson, the lesson is silently lost.
3. **#581** — the 2026-04-15 triage identified five bad-memory categories not caught by the current gate: phase/date/issue-locked, one-time events, implementation details, vague/abstract, redundant with code/docs/skills. The existing gate asks the right shape of question but doesn't force the rigor.

Both skills currently carry the same gate wording and situation guidance verbatim — the fix applies to both.

## Goals

- Sharpen **Recurs** with an explicit "arbitrary project" litmus that drops project-internal implementation work and phase/issue/date-locked situations.
- Extend **Right home** with a mandatory verification procedure (git log recent files, grep the candidate's content) and a three-row outcomes table that distinguishes fresh captures / already-surfaced homes from surfacing failures from content gaps.
- Preserve engram's self-contained write posture — engram never writes to external homes (CLAUDE.md, skills, specs, rules). The only write actions available are: write new memory, update existing memory, or move on.
- Keep both skills at the same or shorter total length (per #581 AC: no long lists of dos/don'ts).
- Apply the same changes identically to `/remember`, adjusted for the interactive flow.

## Non-goals

- **No Go-side write-path changes.** The 2026-04-16 integration spec declares cross-source dedup on write a non-goal. This change respects that — all verification is agent-executed via shell commands prescribed in skill prose.
- **No new CLI subcommands.** No `engram learn verify-home` helper. The simpler ad-hoc check (#583 AC #1 "in the interim") is the deliverable.
- **No writes to external homes.** Engram never edits CLAUDE.md, skills, specs, or rules on behalf of the user.
- **No change to the Step 1 (identify learnable moments), Step 4 (persist), Step 5 (handle results) structure.**
- **No recall-pipeline changes.** This is a write-side skill edit only.

---

## Architecture

Two files change:

- `skills/learn/SKILL.md`
- `skills/remember/SKILL.md`

No Go code. No new dependencies. Verification happens via agent-executed `git log`, file reads, and grep/scan — all within the agent's existing toolset.

---

## Step 2: Quality gate redesign

The Step 2 block becomes:

```markdown
### Step 2: Quality gate

For each candidate, check every gate in order. A single failure drops the candidate.

#### 1. Recurs — would an agent on an unrelated project face this?

Strip the situation to **activity + domain** ("implementing Claude Code hooks", "writing Go tests with context"). If the situation names:
- This project (engram / traced / etc.), its internals, or its architecture
- Phase numbers, issue IDs, commit hashes, or dates
- One-time events ("user said X today"), diary entries, or status snapshots

…it fails Recurs. An agent working on a web app, a game, or a data pipeline should plausibly hit this situation too.

#### 2. Actionable — does it change what an agent would DO?

A passing memory names a concrete action. Fails if it's a vague observation
("things can go wrong"), an inert fact ("X exists"), or a raw debug log.

#### 3. Right home — is memory the correct place?

**a. Name the alternative home:** code, a doc, a skill, CLAUDE.md, a
`.claude/rules/*.md` file, or a spec/plan under `docs/`.

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
```

**Key properties:**
- Recurs's arbitrary-project litmus catches #583 (engram-internal framing) and #581's phase/issue-locked category.
- Actionable catches #581's vague/abstract category.
- Right home's verification step catches both #582 (drops without verifying) and #583 (persists when home owns it).
- Row 1 (home has it, surfaced/fresh) — drop. Avoids reinforcing content we just wrote or already applied.
- Row 2 (home has it, didn't surface) — persist for task-triggered reinforcement. Matches the 2026-04-16 integration spec's "desirable reinforcement" rationale.
- Row 3 (home lacks content / no home) — persist. Prevents #582's silent-loss failure.

---

## Step 3: Situation guidance

Pared down to the essentials (the Recurs gate already handles the "arbitrary project" test):

```markdown
### Step 3: Draft memories

For each surviving candidate:
- Corrections/failures → feedback (SBIA)
- Knowledge/patterns → fact (situation + subject/predicate/object)

**Situation field:** Describe the task the agent would be embarking on, in
terms of **activity + domain** — not the diagnosis, symptom, or fix. A good
situation matches how `/prepare` queries would be phrased *before* the lesson
is known.

| Bad (bakes in hindsight) | Good (activity + domain) |
|---|---|
| "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |
```

Dropped from the current section: the Litmus sentence (redundant with Recurs) and the surrounding prose.

---

## /remember differences from /learn

`/remember` is always interactive — the user explicitly invokes it. In the Right home verification-outcomes table, rows 2 and 3 default to **asking the user** instead of acting autonomously:

- Row 2 prompt: "This is already in `<home>` — reinforce via memory, or just note it?"
- Row 3 prompt: "The home for this would be `<home>` but it's not there. Save to memory anyway?"

Everything else (gate wording, Step 3 situation guidance, dedup / contradiction handling) is identical. Step 4 uses `--source human` instead of `--source agent`. Step 5's "tell the user why and suggest the right home" on drop stays.

---

## Pressure tests

Verified outside the SKILL.md files, via the `superpowers:writing-skills` skill's pressure-test mechanism. These scenarios are the acceptance tests for the gate rewrite:

**PT-1 (#583) — Fresh-write reinforcement.** Agent just committed a spec paragraph articulating a design rationale. Drafts: "When designing dedup or gating behavior for a memory system… default to task-triggered reinforcement."
- **Expected:** Recurs fails ("designing dedup behavior for a memory system" describes engram-internal implementation work; no arbitrary-project agent faces this). Drop at Recurs. (Fallback: even if Recurs weakly passes, Right home row 1 — home has it and we just wrote it — also drops.)

**PT-2 (#582) — Home claimed but missing.** Agent drafts a feedback memory: "Don't add env-var toggles speculatively — YAGNI." Claims home is CLAUDE.md.
- **Expected:** Recurs passes (generalizes). Actionable passes. Right home verification reads CLAUDE.md, grep finds no match → row 3 → persist new memory. No silent loss.

**PT-3 (#581) — Phase-locked.** Agent drafts: "When checking Phase 2 implementation status, verify ACK-wait works before claiming complete."
- **Expected:** Recurs fails (names phase number, internal plan). Drop at Recurs.

**PT-4 (#582 row-2 variant) — Home had it, didn't surface.** Agent runs `targ check` and discovers an error late; should have used `targ check-full`. CLAUDE.md has this guidance but the agent didn't recall it.
- **Expected:** Right home verification finds content in CLAUDE.md (row 2). Engram dedup returns DUPLICATE on existing `targ-check-full-not-check` → update its situation to broaden surfacing.

---

## Rollout

Single skill edit, committed atomically per file:
1. `skills/learn/SKILL.md`
2. `skills/remember/SKILL.md`

Both skills must use `superpowers:writing-skills` for the edit (per `CLAUDE.md` Skill Editing rule): baseline behavioral pressure test (RED), apply edit (GREEN), verify behavioral change via the four pressure tests above.

No feature flag. Changes take effect immediately for all invocations of `/learn` and `/remember`.

---

## Risks

| Risk | Mitigation |
|---|---|
| Agent skips or under-runs the `git log` verification step | Skill prose is direct and includes the exact command. Pressure tests verify behavior change. |
| `git log --since='14 days ago'` yields too many files to grep efficiently | The filtered paths (`docs/ specs/ plans/ skills/ CLAUDE.md .claude/`) keep the result bounded; typical projects have <100 files touched in 14 days. |
| Agent judgement on "did it surface this session" varies | The three-row outcomes table gives concrete signals (read, search, recall, fresh write). Still a judgement call, but more grounded than before. |
| Gate becomes noticeably longer than current 65/59-line skills | Folding the separate situation-guidance section into the gate's Recurs litmus absorbs the budget. Target: ≤ current length. |

---

## Open implementation details (resolved during planning)

- Exact phrasing of row 2's reinforcement-vs-update decision in skill prose.
- Whether to include a per-skill one-line example near the gate.
- Whether the pressure tests live as a comment inside `superpowers:writing-skills` invocation or as a docs/ file.
