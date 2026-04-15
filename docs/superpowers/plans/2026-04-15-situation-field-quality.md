# Situation Field Quality Fix — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix hindsight bias in memory situation fields so memories are matchable from a pre-insight perspective.

**Architecture:** Three skill files get instruction updates (learn, remember, prepare). Then a triage pass classifies all 406 existing memories and produces a review document. After user approval, changes are executed.

**Tech Stack:** Skill markdown files, engram CLI (`engram update`, file deletion)

**IMPORTANT:** Tasks 1-3 modify SKILL.md files. Per project rules, the executor MUST use the `superpowers:writing-skills` skill for each, which enforces TDD: baseline behavior test (RED), update skill (GREEN), verify behavioral change via pressure test.

---

### Task 1: Update learn skill with situation-writing guidance

**Files:**
- Modify: `skills/learn/SKILL.md` (Step 2, after the autonomous/interactive paragraphs)

- [ ] **Step 1: Invoke writing-skills skill**

The executor must invoke `superpowers:writing-skills` before editing. This skill will guide a baseline pressure test, the edit, and a verification pressure test.

- [ ] **Step 2: Add situation guidance block to Step 2**

Insert the following after the "Interactive review only when the user explicitly invokes /learn" paragraph and before "### Step 3: Persist memories":

```markdown
#### Writing good situations

Write the situation from the perspective of an agent who needs this lesson but doesn't know it yet. Describe the *activity* they'd be doing, not the *problem* they'd encounter. Ask yourself: "What would I be doing right before I need this?"

**Litmus test:** Before persisting, check: would an agent who hasn't learned this lesson yet plausibly search for or be described by this situation? If the situation contains the diagnosis, the symptom, or the fix — you've baked in hindsight. Strip it back to just the activity and domain.

| Bad (hindsight-biased) | Good (pre-insight activity) |
|---|---|
| "When implementing hooks that depend on environment variables set by the agent" | "When implementing Claude Code plugin hooks" |
| "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |
```

- [ ] **Step 3: Verify via writing-skills pressure test**

The writing-skills skill will guide a pressure test. The test scenario should be: "An agent just discovered that `engram update` silently drops fields when the TOML has an unexpected schema version. Draft a feedback memory." The situation should come out as something like "When using engram CLI to update memories" — NOT "When engram update encounters unexpected schema versions."

- [ ] **Step 4: Commit**

```bash
git add skills/learn/SKILL.md
git commit -m "docs(skills): add situation-writing guidance to learn skill

Prevents hindsight bias in memory situation fields by instructing agents
to write from the pre-insight perspective.

AI-Used: [claude]"
```

---

### Task 2: Update remember skill with situation-writing guidance

**Files:**
- Modify: `skills/remember/SKILL.md` (Step 2, after the examples and before "Ask the user to approve or edit the fields")

- [ ] **Step 1: Invoke writing-skills skill**

The executor must invoke `superpowers:writing-skills` before editing.

- [ ] **Step 2: Add situation guidance block to Step 2**

Insert the following after the Fact example block and before "Ask the user to approve or edit the fields.":

```markdown
#### Writing good situations

Write the situation from the perspective of an agent who needs this lesson but doesn't know it yet. Describe the *activity* they'd be doing, not the *problem* they'd encounter. Ask yourself: "What would I be doing right before I need this?"

**Litmus test:** Before persisting, check: would an agent who hasn't learned this lesson yet plausibly search for or be described by this situation? If the situation contains the diagnosis, the symptom, or the fix — you've baked in hindsight. Strip it back to just the activity and domain.

| Bad (hindsight-biased) | Good (pre-insight activity) |
|---|---|
| "When implementing hooks that depend on environment variables set by the agent" | "When implementing Claude Code plugin hooks" |
| "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |
```

- [ ] **Step 3: Verify via writing-skills pressure test**

The test scenario should be: "User says 'remember that targ check stops at the first error, use targ check-full instead'." The situation should come out as something like "When running quality checks in this project" — NOT "When targ check stops early and misses errors."

- [ ] **Step 4: Commit**

```bash
git add skills/remember/SKILL.md
git commit -m "docs(skills): add situation-writing guidance to remember skill

Same pre-insight framing guidance as learn skill, applied to
user-explicit memory captures.

AI-Used: [claude]"
```

---

### Task 3: Update prepare skill with query construction guidance

**Files:**
- Modify: `skills/prepare/SKILL.md` (Step 2, after "Choose queries that would surface relevant prior work, decisions, patterns, and pitfalls.")

- [ ] **Step 1: Invoke writing-skills skill**

The executor must invoke `superpowers:writing-skills` before editing.

- [ ] **Step 2: Add query guidance block to Step 2**

Insert the following after the "Choose queries that would surface relevant prior work, decisions, patterns, and pitfalls." line:

```markdown
**Query by activity, not by fear.** Construct queries around what you're *about to do*, not what you're worried about or what might go wrong. Memories are indexed by the activity where the lesson was learned, not by failure mode.

Examples:
- About to write hooks → query "implementing Claude Code hooks"
- About to write tests → query "writing Go tests" or "testing in [specific domain]"
- About to do a git push → query "git push workflow"
- DON'T query "common mistakes when writing hooks" — no memory is indexed that way
```

- [ ] **Step 3: Verify via writing-skills pressure test**

The test scenario should be: "Agent is about to implement a new CLI subcommand in a Go project that uses targ." Queries should come out as things like "implementing CLI commands", "working with targ" — NOT "things that go wrong with CLI commands" or "CLI implementation pitfalls."

- [ ] **Step 4: Commit**

```bash
git add skills/prepare/SKILL.md
git commit -m "docs(skills): add query construction guidance to prepare skill

Aligns recall queries with the new pre-insight situation framing so
queries match how situations are written.

AI-Used: [claude]"
```

---

### Task 4: Triage existing memories

**Files:**
- Read: all TOML files under `/Users/joe/.local/share/engram/memory/feedback/` and `/Users/joe/.local/share/engram/memory/facts/`
- Create: `docs/superpowers/plans/2026-04-15-memory-triage.md` (review document)

- [ ] **Step 1: Read all memories and classify each**

Read every memory TOML file. For each, read the full content (situation + content fields) and classify into one of four buckets:

| Bucket | Criteria |
|---|---|
| **Fine** | Situation already describes a pre-insight activity. No hindsight, not over-specific, not phase-locked. |
| **Rewrite** | Lesson is reusable but situation has hindsight bias, over-specificity, or solution-aware framing. |
| **Delete (situational)** | Memory is a project-phase snapshot, diary entry, or one-time event that will never match again. |
| **Delete (content)** | Lesson itself is too narrow, redundant, or no longer relevant regardless of situation quality. |

For each rewrite, draft the new situation following the pre-insight framing: describe the activity the agent would be doing, not the problem they'd encounter.

- [ ] **Step 2: Produce the triage review document**

Write a markdown document to `docs/superpowers/plans/2026-04-15-memory-triage.md` with this format:

```markdown
# Memory Triage Results

## Summary
- Fine: N
- Rewrite: N
- Delete (situational): N
- Delete (content): N

## Fine (no changes)

| Memory | Current Situation |
|---|---|
| memory-slug | "current situation text" |

## Rewrite

| Memory | Current Situation | Proposed Situation |
|---|---|---|
| memory-slug | "old situation" | "new situation" |

## Delete (situational)

| Memory | Current Situation | Reason |
|---|---|---|
| memory-slug | "current situation" | phase-locked / diary entry / one-time |

## Delete (content)

| Memory | Current Situation | Reason |
|---|---|---|
| memory-slug | "current situation" | too narrow / redundant / obsolete |
```

- [ ] **Step 3: Present to user for review**

Tell the user: "Triage complete — review document at `docs/superpowers/plans/2026-04-15-memory-triage.md`. Please review before I execute any changes."

**STOP HERE.** Do not proceed to Task 5 until the user has reviewed and approved the triage.

---

### Task 5: Execute approved triage changes

**Depends on:** Task 4 + user approval of triage document.

- [ ] **Step 1: Execute rewrites**

For each memory in the "Rewrite" bucket that the user approved:

```bash
engram update --name "<memory-slug>" --situation "<new situation>"
```

- [ ] **Step 2: Execute deletes**

For each memory in the "Delete (situational)" and "Delete (content)" buckets that the user approved:

```bash
rm /Users/joe/.local/share/engram/memory/feedback/<memory-slug>.toml
# or
rm /Users/joe/.local/share/engram/memory/facts/<memory-slug>.toml
```

Check which subdirectory the file is in before deleting.

- [ ] **Step 3: Verify final state**

```bash
# Count remaining memories
find /Users/joe/.local/share/engram/memory -name "*.toml" -type f | wc -l
```

Report the before/after count to the user.

- [ ] **Step 4: Commit triage document**

```bash
git add docs/superpowers/plans/2026-04-15-memory-triage.md
git commit -m "docs: memory triage results for situation field quality fix

Classified 406 memories: N fine, N rewritten, N deleted.
See triage document for full details.

AI-Used: [claude]"
```
