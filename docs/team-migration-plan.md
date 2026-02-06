# Migration Plan: projctl Orchestration to Claude Code Agent Teams

> **Historical Note:** This migration plan was executed in two phases during January-February 2026. Phase 1 (ISSUE-69 through ISSUE-72) and Phase 2 (ISSUE-73 through ISSUE-80) successfully completed the migration from the yield protocol to Claude Code's native team mode. This document is preserved for historical reference.

## Summary

Replace projctl's custom coordination layer (yield protocol, context TOML files,
parallel looper) with Claude Code's built-in agent teams while preserving the
phase state machine, traceability, QA contracts, memory, and territory systems.

**Core changes:**
1. `/project` becomes a team lead in delegate mode
2. Skills dispatched as teammates (not Skill tool in main context)
3. Yield TOML → SendMessage between teammates and lead
4. Context TOML → spawn prompts + messages
5. Task runtime coordination → built-in TaskList
6. Interview skills use AskUserQuestion directly (no yield-resume relay)

**Unchanged:** Phase state machine, traceability, QA contracts, memory system,
territory mapping, issue management, TDD discipline, git worktrees.

---

## Architecture: Before and After

### Before (Current)
```
User → /project (Skill tool, main context)
         │
         ├── projctl context write → TOML file
         ├── Skill tool → skill runs in main context
         ├── Skill writes yield TOML
         ├── projctl context read → parse yield
         ├── Handle yield type (need-user-input → prompt user → resume)
         └── projctl state transition
```

### After (Target)
```
User → /project (team lead, delegate mode)
         │
         ├── Teammate spawnTeam
         ├── TaskCreate (from tasks.md for implementation phase)
         ├── Spawn producer teammate → invokes skill via Skill tool
         │     ├── Teammate uses AskUserQuestion directly
         │     └── Teammate sends SendMessage to lead on completion
         ├── Spawn QA teammate → invokes /qa skill
         │     └── Teammate sends SendMessage with verdict
         ├── projctl state transition (on QA approval)
         └── Shutdown teammates, cleanup team
```

### PAIR LOOP with Teams
```
Lead:  spawn producer-teammate(context: phase, artifacts, prior results)
Teammate: invokes Skill("pm-interview-producer")
Teammate: uses AskUserQuestion for interview (direct, no relay)
Teammate: produces artifacts
Teammate: SendMessage → lead: "complete: docs/requirements.md, IDs: REQ-1..REQ-5"

Lead:  spawn qa-teammate(context: producer SKILL.md path, artifact paths, iteration=1)
Teammate: invokes Skill("qa")
Teammate: reads contract from producer SKILL.md, validates
Teammate: SendMessage → lead: "approved" | "improvement-request: [issues]"

Lead (if improvement-request, iteration < 3):
  spawn new producer-teammate(context: QA feedback + prior context)
Lead (if approved):
  projctl state transition --to next-phase
```

---

## Yield Type Mapping

| Current Yield | Teams Equivalent | Notes |
|---|---|---|
| `complete` | SendMessage to lead | Include artifact paths, IDs, files modified |
| `approved` | SendMessage to lead | QA approval |
| `improvement-request` | SendMessage to lead | QA feedback for producer |
| `need-user-input` | AskUserQuestion (direct) | Teammate asks user, no lead relay |
| `need-user-input (inferred)` | AskUserQuestion (direct) | CLASSIFY step happens inline |
| `need-context` | **Eliminated** | Teammates read files directly |
| `need-decision` | AskUserQuestion or SendMessage→lead | Depends on who decides |
| `need-agent` | SendMessage→lead | Lead spawns requested teammate |
| `blocked` | SendMessage→lead | Lead presents to user |
| `error` | SendMessage→lead | Lead retries or escalates |
| `escalate-phase` | SendMessage→lead | Lead re-enters prior phase |
| `escalate-user` | SendMessage→lead→user | Lead relays |

**Key win:** `need-context` disappears entirely. `need-user-input` for interviews
becomes direct. These two were the most complex yield-resume cycles.

---

## tasks.md vs TaskList

**tasks.md stays as the traced artifact.** It contains TASK-NNN IDs, acceptance
criteria, dependencies, and `**Traces to:**` fields. It is the source of truth
for traceability.

**TaskList is runtime coordination only.** During the implementation phase, the
lead parses tasks.md (via `projctl tasks deps`), creates TaskList entries with
matching IDs in metadata, maps dependencies to addBlockedBy/addBlocks, and tracks
status as teammates complete work.

---

## Phase 1: Foundation + Proof of Concept

**Goal:** Team-aware orchestrator + one complete phase (PM) working end-to-end.

**Issues:** ISSUE-69, ISSUE-70, ISSUE-71, ISSUE-72

### ISSUE-69: Create team-mode project orchestrator

**Files to create:**
- `skills/project/SKILL.md` — Rewrite for team lead (delegate mode)
- `skills/project/SKILL-full.md` — Updated control loop, PAIR LOOP, resume map

**Key changes in orchestrator:**
- On start: `Teammate(operation: "spawnTeam", team_name: "<project>")`
- Phase dispatch: `Task(subagent_type: "general-purpose", team_name: ...,
  prompt: "Invoke /pm-interview-producer. Context: ...")`
- Result handling: receive SendMessage from teammate, parse result
- PAIR LOOP: spawn producer → receive result → spawn QA → receive verdict → iterate or advance
- Phase transitions: `projctl state transition` unchanged
- End: shutdown teammates, `Teammate(operation: "cleanup")`
- Delegate mode: lead never edits files, only coordinates

**Critical design: context injection via spawn prompt.** Instead of TOML:
```
Invoke the /pm-interview-producer skill for this project.

Project: <name>
Issue: ISSUE-NNN
Phase: pm
Docs dir: docs/
Requirements path: docs/requirements.md

Prior context:
<territory map summary>
<memory query results>
<issue description>

When complete, send me a message with:
- Artifact path
- IDs created (REQ-NNN list)
- Files modified
- Key decisions made
```

### ISSUE-70: Migrate pm-interview-producer to support direct user interaction

**File to modify:** `skills/pm-interview-producer/SKILL.md`

**Changes:**
- Remove "write yield TOML to output.yield_path" instructions
- Remove need-user-input yield for interview questions
- Add: use AskUserQuestion directly during INTERVIEW phase
- Add: use AskUserQuestion for CLASSIFY (inferred spec approval)
- Add: on completion, send SendMessage to team lead with results
- Keep: GATHER→ASSESS→INTERVIEW→SYNTHESIZE→CLASSIFY→PRODUCE workflow
- Keep: contract section (QA still reads it)
- Keep: `**Traces to:**` in output artifacts

**Backward compat:** Skill detects context source. If TOML context file exists
at expected path, use legacy mode. If invoked with context in conversation, use
team mode. This is natural — Claude reads whatever context is available.

### ISSUE-71: Migrate QA skill to team mode

**File to modify:** `skills/qa/SKILL.md`

**Changes:**
- Remove TOML context reading instructions
- Add: receive context via message (producer SKILL.md path, artifact paths, iteration)
- Remove yield TOML writing
- Add: send verdict via SendMessage (approved | improvement-request with issues)
- Keep: contract extraction from producer SKILL.md `## Contract` section
- Keep: three-phase workflow (LOAD → VALIDATE → RETURN)

### ISSUE-72: Update shared templates for team mode

**Files to modify:**
- `skills/shared/PRODUCER-TEMPLATE.md` — Add "Team Mode" section for context
  reception and result reporting alongside existing TOML instructions
- `skills/shared/INTERVIEW-PATTERN.md` — Add "Team Mode" section for direct
  AskUserQuestion usage (keep yield-resume docs for legacy reference)

**Files unchanged:** CONTRACT.md, ownership-rules/

### Validation (Phase 1)
- `/project new` creates a team, spawns PM teammate
- PM teammate asks user questions directly via AskUserQuestion
- PM teammate produces requirements.md with REQ-NNN IDs and traces
- Lead spawns QA teammate, QA validates against contract
- On approval: lead advances state via `projctl state transition`
- On improvement-request: lead spawns new PM teammate with feedback
- `projctl trace validate` passes after PM phase

---

## Phase 2: All Skills Migrated

**Goal:** Every producer and QA skill works as a teammate.

**Issues:** ISSUE-73, ISSUE-74, ISSUE-75, ISSUE-76, ISSUE-77

### ISSUE-73: Migrate interview producers (design + arch)

**Files to modify:**
- `skills/design-interview-producer/SKILL.md`
- `skills/arch-interview-producer/SKILL.md`

Same pattern as pm-interview-producer: direct AskUserQuestion, SendMessage results.

### ISSUE-74: Migrate inference producers

**Files to modify:**
- `skills/pm-infer-producer/SKILL.md`
- `skills/design-infer-producer/SKILL.md`
- `skills/arch-infer-producer/SKILL.md`

These don't need AskUserQuestion (they analyze, not interview). Changes:
context from message, results via SendMessage.

### ISSUE-75: Migrate remaining producers

**Files to modify:**
- `skills/breakdown-producer/SKILL.md`
- `skills/doc-producer/SKILL.md`
- `skills/alignment-producer/SKILL.md`
- `skills/retro-producer/SKILL.md`
- `skills/summary-producer/SKILL.md`

Same pattern: context from message, results via SendMessage.

### ISSUE-76: Migrate TDD skills

**Files to modify:**
- `skills/tdd-producer/SKILL.md` (composite)
- `skills/tdd-red-producer/SKILL.md`
- `skills/tdd-green-producer/SKILL.md`
- `skills/tdd-refactor-producer/SKILL.md`
- `skills/tdd-red-infer-producer/SKILL.md`

The TDD composite becomes: lead spawns red-teammate → QA → green-teammate →
QA → refactor-teammate → QA, with commit skill invoked between each.

### ISSUE-77: Wire all phases into orchestrator

**Files to modify:** `skills/project/SKILL.md`, `skills/project/SKILL-full.md`

Complete the orchestrator to handle all workflows:
- `new`: PM→Design→Arch→Breakdown→Implementation(TDD)→Doc→Alignment→Retro→Summary
- `adopt`: Explore→InferTests→InferArch→InferDesign→InferReqs→Escalations→Doc→...
- `align`: Same as adopt
- `task`: Implementation→Documentation

### Validation (Phase 2)
- Full `/project new` runs end-to-end via team mode
- All phase transitions work
- `projctl trace validate` passes at each phase boundary
- QA pair loop iterates correctly (max 3x)
- TDD red→green→refactor cycle completes with proper commits

---

## Phase 3: Runtime Task Coordination + Parallel Execution

**Goal:** TaskList replaces custom task tracking; parallel tasks via native teams.

**Issues:** ISSUE-78, ISSUE-79

### ISSUE-78: TaskList-based implementation coordination

**Changes in orchestrator:**
- After breakdown phase: parse tasks.md via `projctl tasks deps --format json`
- Create TaskList entries with `TaskCreate` for each TASK-NNN
- Map dependencies to `addBlockedBy`/`addBlocks`
- As teammates complete tasks: `TaskUpdate(status: "completed")`
- Use `TaskList` to find next available work
- tasks.md remains the canonical traced artifact

### ISSUE-79: Native parallel task execution

**Changes in orchestrator:**
- Identify parallel tasks: `projctl tasks parallel` or TaskList with no blockers
- Spawn one teammate per independent task
- Each teammate creates worktree: `projctl worktree create --task TASK-NNN`
- Merge-on-complete: `projctl worktree merge --task TASK-NNN` when teammate finishes
- Replace parallel-looper skill with native team spawning
- Keep consistency-checker for post-batch validation (optional)

### Validation (Phase 3)
- TaskList entries match tasks.md TASK-NNN IDs
- Dependencies correctly reflected in TaskList
- Parallel tasks run simultaneously as concurrent teammates
- Git worktree isolation works (no file conflicts)
- Merge-on-complete preserves work from earlier completions
- `projctl trace validate` passes

---

## Phase 4: Cleanup

**Goal:** Remove legacy infrastructure, simplify codebase.

**Issues:** ISSUE-80, ISSUE-81, ISSUE-82, ISSUE-83

### ISSUE-80: Remove legacy yield infrastructure

**Go code to remove:**
- `internal/yield/yield.go` (~240 lines) — Yield type definitions and TOML validation
- `internal/yield/yield_test.go` — Tests
- `cmd/projctl/yield.go` — CLI commands (yield validate, yield types)

**State machine simplification:**
- Remove `YieldState` from `internal/state/state.go` (pending yield tracking)
- Remove `SetYield()`/`ClearYield()` methods
- Remove `state yield set/clear` CLI commands
- Keep `PairState` (still tracks producer/QA iterations)

### ISSUE-81: Remove legacy context TOML infrastructure

**Go code to remove:**
- `internal/context/context.go` Write/Read/WriteParallel functions (~440 lines)
- `internal/context/yieldpath.go` GenerateYieldPath (~170 lines)
- `cmd/projctl/context.go` — CLI commands (context write, read, write-parallel)

**Keep:**
- `internal/context/budget.go` — Token budget checking (still useful)

### ISSUE-82: Clean up shared templates

**Files to update:**
- `skills/shared/PRODUCER-TEMPLATE.md` — Remove legacy TOML sections
- `skills/shared/INTERVIEW-PATTERN.md` — Remove yield-resume documentation

**Files to remove:**
- `skills/shared/YIELD.md` — No longer used
- `skills/shared/CONTEXT.md` — No longer used
- `skills/shared/QA-TEMPLATE.md` — Already deprecated, delete

Remove backward-compat TOML detection from all migrated SKILL.md files.

### ISSUE-83: Deprecate parallel-looper and consistency-checker skills

- `skills/parallel-looper/` — Mark deprecated, replaced by native team parallelism
- `skills/consistency-checker/` — Keep if still useful for batch QA, otherwise deprecate

### Validation (Phase 4)
- `go test ./...` passes
- `golangci-lint run` passes
- No TOML context or yield files created during execution
- `/project new` still works end-to-end
- `projctl trace validate` passes
- Removed ~850+ lines of Go code

---

## Future Considerations

### ISSUE-84: Explore enforcement and QA via Claude Code hooks

- **PostToolUse hook** for `projctl trace validate` after Write/Edit
- **Stop hook** for QA verification
- **Agent-based hooks** for LLM-as-judge QA

Deferred until core migration (Phases 1-3) is complete.

---

## Risk Mitigations

| Risk | Mitigation |
|---|---|
| Teams are experimental (no /resume for teammates) | projctl state machine still tracks phase; lead can re-spawn teammates after session resume |
| Message size limits for large context | Write large context to temp files, send paths in messages; teammates read directly |
| Teammate crashes mid-skill | Lead detects via idle notification + TaskList status; spawns replacement with same context |
| Cost increase (multiple sessions) | Sequential phases reuse pattern (shutdown prev teammate before spawning next); parallel only where justified |
| Interview UX change | AskUserQuestion is actually better UX than yield-relay; user sees questions directly |
