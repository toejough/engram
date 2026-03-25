# Stale Spec Cleanup — SessionStart Surfacing Removal (#377)

## Problem

SessionStart memory surfacing was removed in the 2026-03-21 session-recall redesign, but the base specs (`use-cases.toml`, `architecture.toml`, `requirements.toml`, `design.toml`, `tests.toml`) still describe it as active. Additionally, 8 tombstone sections marked `[REMOVED]` clutter the spec files.

## Scope

Surgical update: fix lies, remove tombstones, don't rewrite test specs for new behavior.

## Changes

### use-cases.toml

**UC-2 (Hook-Time Surfacing & Enforcement):**
- Replace SessionStart bullet (line 275) with: "SessionStart — maintain + recall hint: Runs `engram maintain` for background triage and emits a static `/recall` reminder. No memory surfacing — context loading is on-demand via `/recall` skill (session-recall redesign, 2026-03-21)."
- Remove "SessionStart" from starting-state and end-state descriptions (lines 267, 269)
- Remove `"session-start"` from `surfacing_contexts` example (line 290)

**UC-1 (Session Learning):**
- Line 54: Remove creation-log → SessionStart reporting claim. Replace with note that creation log is written but reading is deferred (no caller since SessionStart surfacing was removed).

**Tombstone removal:**
- Delete UC-4 ("Skill Generation [REMOVED — Issue #215]")
- Delete UC-5 ("CLAUDE.md Management [REMOVED — Issue #215]")
- Delete UC-22 ("Mechanical Instruction Extraction *(removed — Phase A-1/S1)*")

### architecture.toml

**ARCH-12 (Hook Integration):**
- Line ~69: Remove `--mode session-start` description (creation log + top 20). Replace with current behavior (maintain + `/recall` hint, no `engram surface` call).
- Line ~97: Update SessionStart hook description to match.

**ARCH-21 (Creation Log):**
- Keep write side (UC-1 still logs creations).
- Mark read-and-clear-at-SessionStart as deferred — reader exists but has no caller.

**Tombstone removal:**
- Delete ARCH-51 ("Automation Generator Pipeline *(removed — Phase A-1/S1)*")
- Delete ARCH-70 ("Extended Always-Loaded Classification (UC-26) — REMOVED (S4)")

### requirements.toml

**REQ-24** ("SessionStart creation report — read and clear creation log"): Mark as removed. The creation log is still written (REQ-23/25), but the read-and-clear-at-SessionStart path is dead — SessionStart no longer surfaces memories or reports creations.

**REQ-9** ("SessionStart surfacing — top 20 by frecency"): Mark as removed. SessionStart no longer surfaces memories.

**REQ-23/25** (creation log write): Keep unchanged — creation log writing still happens.

**Tombstone removal:**
- Delete REQ-83 ("~~REMOVED (S2)~~ Dimension routing before escalation")
- Delete REQ-90 ("CLI command `engram automate` *(removed — Phase A-1/S1)*")
- Delete REQ-91 ("No graceful degradation for automate *(removed — Phase A-1/S1)*")

### design.toml

- Remove/update SessionStart creation report + recency surfacing section (lines ~388–413).

### tests.toml

- Mark SessionStart surfacing tests as removed (~20 tests referencing session-start mode surfacing, creation log read-at-SessionStart).
- Keep creation log write tests unchanged.

## Not in scope

- Rewriting test specs for the new maintain + `/recall` behavior
- Updating UC-15/UC-17/UC-26 references to SessionStart (these are secondary mentions, not definitions)
- Dead code removal (#375) — separate issue
