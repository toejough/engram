# Situation Field Quality Fix

## Problem

Memory situation fields are written with hindsight bias. Agents write situations immediately after learning a lesson, and their post-failure perspective gets baked into the field. This makes memories unmatchable — nobody searches for "I'm about to make this specific mistake" because you don't know you're making it yet.

Four failure modes observed across 406 existing memories:

1. **Hindsight framing** — situation includes the diagnosis the next agent won't have yet. ("When implementing hooks that depend on environment variables set by the agent" — you wouldn't know env vars are the issue until after you fail.)
2. **Over-specificity** — locked to one exact moment that will never recur. ("When implementing engram Phase 4 Task 4")
3. **Solution-aware framing** — describes what you'd know after learning the lesson. ("When dealing with incorrectly refined memories" — you'd be "working with memories," not specifically "incorrectly refined" ones.)
4. **Project status snapshots** — diary entries about a moment in project history. ("After merging the bug fix branches from the 2026-04-04 session")

## Scope

- Skill instruction changes: learn, remember (write side) and prepare (read side)
- Existing memory remediation: triage all 406, delete unreusable ones, rewrite bad situations
- NOT changing the recall pipeline or matching mechanism

## Design

### 1. Learn + Remember skill instruction changes

Add explicit situation-writing guidance in both `skills/learn/SKILL.md` (Step 2) and `skills/remember/SKILL.md` (Step 2).

#### Framing instruction

> Write the situation from the perspective of an agent who needs this lesson but doesn't know it yet. Describe the *activity* they'd be doing, not the *problem* they'd encounter. Ask yourself: "What would I be doing right before I need this?"

#### Litmus test (self-check after drafting)

> Before persisting, check: would an agent who hasn't learned this lesson yet plausibly search for or be described by this situation? If the situation contains the diagnosis, the symptom, or the fix — you've baked in hindsight. Strip it back to just the activity and domain.

#### Anti-patterns with corrections

| Bad (hindsight-biased) | Good (pre-insight activity) |
|---|---|
| "When implementing hooks that depend on environment variables set by the agent" | "When implementing Claude Code plugin hooks" |
| "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |

### 2. Prepare skill instruction changes

Add query construction guidance to `skills/prepare/SKILL.md` (Step 2), after the existing "Choose queries that would surface relevant prior work, decisions, patterns, and pitfalls" line.

#### Query framing

> Construct queries around what you're *about to do*, not what you're worried about or what might go wrong. Describe the activity and domain.

#### Examples

| Scenario | Query |
|---|---|
| About to write hooks | "implementing Claude Code hooks" |
| About to write tests | "writing Go tests" or "testing in [specific domain]" |
| About to do a git push | "git push workflow" |

> DON'T query: "common mistakes when writing hooks" — memories aren't indexed by failure mode, they're indexed by the activity where the failure happened.

### 3. Existing memory triage and remediation

Two-pass process over all 406 memories.

#### Pass 1 — Classify each memory into one of four buckets

| Bucket | Criteria | Action |
|---|---|---|
| **Fine** | Situation already describes a pre-insight activity | No change |
| **Rewrite** | Lesson is reusable but situation has hindsight bias or is too specific | Rewrite situation field |
| **Delete (situational)** | Memory is a project-phase snapshot, diary entry, or one-time event | Delete TOML file |
| **Delete (content)** | Lesson itself is too narrow or redundant regardless of situation | Delete TOML file |

#### Pass 2 — Execute changes

- Rewrites: `engram update --name <slug> --situation "new situation"`
- Deletes: remove the TOML file

#### Review gate

An agent performs the classification and drafts proposed rewrites. The output is a single markdown document listing every memory with its bucket and (for rewrites) the proposed new situation. The user reviews this document before any changes are executed.

No ongoing audit command. The skill instruction changes should prevent the pattern going forward. Recurrence signals the instructions need further tuning, not automated detection.
