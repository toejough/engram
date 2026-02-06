# Tasks: PM Interview Enforcement

**Issue:** ISSUE-54
**Created:** 2026-02-04
**Status:** Draft

**Traces to:** ISSUE-54

---

## Tasks

### TASK-1: Update project/SKILL.md to enforce context-only contract

**Description:** Update the project orchestrator SKILL.md to explicitly prohibit passing override instructions to interview-producer skills. The orchestrator should only pass context (issue description, file paths, prior artifacts), never behavioral instructions like "skip interview" or "already defined".

**Status:** Ready

**Acceptance Criteria:**
- [x] project/SKILL.md explicitly states: pass context only, never override instructions
- [x] No phrases like "skip interview", "already defined", "do not conduct" in skill dispatch
- [x] If user says "skip interviews", orchestrator respects that naturally (Claude understands)

**Files:**
- skills/project/SKILL.md

**Dependencies:** None

**Traces to:** REQ-1, ARCH-1
