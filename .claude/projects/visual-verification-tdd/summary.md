# Visual Verification TDD - Project Summary

**Project:** visual-verification-tdd
**Issues Resolved:** ISSUE-007, ISSUE-014
**Type:** Documentation-only (skill file updates)
**Duration:** ~23 minutes

---

## Overview

This project integrated visual/interaction verification into the TDD workflow documentation. The core insight is that UI, CLI, and API are all user interfaces that follow the same testing model: structure + behavior + properties. Visual verification is not a special case to be handled differently, but part of the standard TDD discipline.

---

## Key Accomplishments

### 1. Unified Interface Testing Model (tdd-red-producer)

Added comprehensive "Testing User Interfaces" section establishing that all interfaces follow the same three-layer testing model:

| Layer | Question | What to Test |
|-------|----------|--------------|
| Structure | Does it exist? | Element presence, argument parsing, endpoint existence |
| Behavior | Does it work? | Interaction -> event -> handler -> state -> output |
| Properties | Does it hold? | Invariants across all screens/commands/endpoints |

Includes concrete test examples for UI (TypeScript), CLI (Go), and API (Go).

### 2. Visual Task Marker Convention (breakdown-producer)

Added `[visual]` tag convention for task titles, with detection heuristics:
- Files modified include UI components, CSS, CLI output formatting
- Description mentions display, render, button, dialog, output format
- Acceptance criteria reference visual properties or design spec

Example: `### TASK-7: [visual] Add loading state to submit button`

### 3. Visual Verification Step (tdd-green-producer)

Added capture mechanisms and verification workflow:

| Interface | Capture Method | Tool |
|-----------|----------------|------|
| Web UI | Browser screenshot | Chrome DevTools MCP `take_screenshot` |
| CLI | Output redirection | `command > output.txt 2>&1` |
| CLI (ANSI) | Script recording | `script -q output.txt command` |

Yield payload now includes `visual_verified` and `visual_evidence` fields.

### 4. Visual Evidence Requirement (tdd-qa)

Added validation rules for `[visual]` tasks:
- Check for `visual_verified = true` in producer yield
- Check for `visual_evidence` path
- Return `improvement-request` if evidence missing
- Waiver process: producer explains, QA escalates to user

### 5. CLAUDE.md Lessons Expanded

Existing UI-specific lessons generalized to all user interfaces:
- "UI testing verifies visual correctness" -> "Interface testing verifies correctness"
- "UI validation is critical" -> "Interface validation is critical"
- Added property-based testing for interfaces as standard practice

---

## Design Decisions

### DD-1: Single Model vs Separate Sections

**Decision:** Unified model with examples for each interface type.

**Rationale:** The testing approach is identical across UI/CLI/API. Separate sections would duplicate concepts and miss the key insight that these are all user interfaces.

### DD-2: Marker Syntax

**Decision:** `[visual]` in task title.

**Rationale:** Visible in task list at a glance, simple grep detection, no schema changes, clearer than `[ui]` since CLI output is also "visual".

### DD-3: Screenshot Capture Tooling

**Decision:** Document existing tools (Chrome DevTools MCP, shell redirection) instead of implementing `projctl screenshot capture`.

**Rationale:** MCP already provides browser screenshots, shell handles CLI output, `projctl screenshot diff` handles comparison. New command adds maintenance burden for marginal value.

---

## Files Changed

| File | Changes |
|------|---------|
| `~/.claude/skills/tdd-red-producer/SKILL.md` | +80 lines: Testing User Interfaces section with examples |
| `~/.claude/skills/breakdown-producer/SKILL.md` | +25 lines: Visual Task Detection section |
| `~/.claude/skills/tdd-green-producer/SKILL.md` | +50 lines: Visual Verification section |
| `~/.claude/skills/tdd-qa/SKILL.md` | +30 lines: Visual Task Validation subsection |
| `~/.claude/CLAUDE.md` | ~20 lines: Expanded interface testing lessons |

---

## Traceability

All changes traced through the full chain:
- ISSUE-007, ISSUE-014 -> REQ-1 through REQ-11 -> DES-1 through DES-9 -> TASK-1 through TASK-5

---

## Outcome

Visual verification is now a documented, required part of the TDD workflow for all user-facing changes. Skills detect visual tasks via the `[visual]` marker, producers capture evidence, and QA enforces evidence requirements. The unified interface testing model ensures consistent treatment across UI, CLI, and API work.
