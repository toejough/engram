---
name: project
description: State-machine-driven project orchestrator
user-invocable: true
---

# Project Orchestrator

Manage projects through structured phases using a durable state machine. Dispatches single-responsibility skills and uses `projctl` for deterministic operations.

## Critical Rules (read on every resume)

1. **Use `projctl` for all state transitions** - Never modify state.toml directly
2. **Use `projctl log write` for all logging** - Structured JSONL via CLI
3. **Use `projctl context write/read` for skill handoffs** - TOML context/result files
4. **Dispatch skills via Skill tool** - Each skill runs as a forked subagent
5. **Commits between TDD phases** - Red → commit → green → commit → refactor → commit
6. **Never skip audits** - Audit loop runs until zero defects
7. **Bounded negotiation** - Max 2 rounds of /negotiate, then escalate
8. **Stop only when all tasks blocked** - Continue with unblocked tasks when conflicts exist
9. **Always run end-of-command sequence** - integrate → repair → validate before completing

## End-of-Command Sequence (CRITICAL)

**Every** `/project` command MUST run this sequence before completing:

```bash
# 1. Integrate any feature-specific docs into top-level
projctl integrate features --dir <project-dir>

# 2. Repair traceability issues (auto-fix where possible)
projctl trace repair --dir <project-dir>

# 3. Validate traceability matrix
projctl trace validate --dir <project-dir>
```

### If Validation Fails

When validation returns issues:

1. **Check for pending escalations:**
   ```bash
   projctl escalation list --dir <project-dir>
   ```

2. **Present interactive prompt to user:**
   ```
   ⚠️  Validation Issues Found

   Unlinked IDs: 3
     REQ-005: No downstream design or architecture
     DES-008: No downstream architecture
     ARCH-012: No downstream task

   Pending Escalations: 2
     ESC-001: [category] [question]
     ESC-002: [category] [question]

   Options:
   1. Resolve escalations now (opens interactive resolution)
   2. Defer to issues (creates ISSUE-NNN for each)
   3. Abort command (leaves state as-is for manual intervention)
   ```

3. **Handle user choice:**
   - **Option 1 (Resolve):** For each escalation, prompt for decision, apply to artifacts
   - **Option 2 (Defer):** Create ISSUE-NNN for each unresolved item
   - **Option 3 (Abort):** Exit without completing command

4. **Loop until clean:**
   ```
   repeat:
     run repair → run validate
     if pass: exit loop
     if fail: show prompt, handle choice
     if abort: exit command
   ```

### Commands That Run This Sequence

- `/project new` - After completion phase
- `/project adopt` - After adopt-complete
- `/project align` - After align-complete
- `/project integrate` - After integrate-complete
- `/project continue` - When resuming a terminal state

**Note:** This sequence is non-negotiable. The project is not "done" until validation passes or user explicitly aborts.

## Invocation

```
/project              # Show dashboard + open projects
/project new          # Start a new project (interactive, per-project folder)
/project adopt        # Adopt existing codebase (infer docs, batch escalations)
/project align        # Lightweight alignment check (update existing docs)
/project integrate    # Merge per-project docs to top-level
/project continue     # Resume an open project
```

## Dashboard (no arguments)

When invoked without arguments:

1. Look for project directories with `state.toml` files
2. Parse state for each project
3. Display:

```
Project Orchestrator

Open Projects:
  project-name     phase              progress        issues
  my-cli-tool      implementation     7/12 tasks      1 escalated
  api-server       design-complete    awaiting align  -

Commands:
  /project new        Start a new project (interactive)
  /project adopt      Adopt existing codebase
  /project align      Lightweight alignment check
  /project integrate  Merge per-project to top-level
  /project continue   Resume an open project
```

---

## /project new

Start a new project through all phases.

### Initialization

```bash
# Create project directory and initialize state
projctl state init --name <project-name> --dir <project-dir>
projctl state transition --dir <project-dir> --to pm-interview
projctl log write --dir <project-dir> --level phase --subject phase-change --message "Starting PM interview"
```

### Phase 1: PM Interview

1. Write context file for /pm-interview:
   ```bash
   projctl context write --dir <project-dir> --task PHASE --skill pm-interview --file <context.toml>
   ```
2. Dispatch `/pm-interview` via Skill tool
3. Read result:
   ```bash
   projctl context read --dir <project-dir> --task PHASE --skill pm-interview --result
   ```
4. Save requirements.md from skill output
5. Transition state:
   ```bash
   projctl state transition --dir <project-dir> --to pm-complete
   projctl log write --dir <project-dir> --level phase --subject phase-result --message "PM interview complete. N requirements captured."
   ```

### Phase 2: Design

1. Transition to `design-interview`
2. Write context with requirements.md summary
3. Dispatch `/design-interview`
4. Save design.md and .pen files
5. Transition to `design-complete`
6. Run alignment check (see below)

### Phase 3: Architecture

1. Transition to `architect-interview`
2. Write context with requirements.md (+ design.md if exists)
3. Dispatch `/architect-interview`
4. Save architecture.md
5. Transition to `architect-complete`
6. Run alignment check

### Alignment Check (after each artifact phase)

```bash
projctl state transition --dir <project-dir> --to alignment-check
projctl trace validate --dir <project-dir>
```

If validation fails:
1. Dispatch `/alignment-check` to interpret gaps and propose fixes
2. Apply fixes (edit `**Traces to:**` fields in artifact files)
3. Re-run `projctl trace validate`
4. Max 2 iterations, then escalate unresolved gaps

### Phase 4: Task Breakdown

1. Transition to `task-breakdown`
2. Write context with all artifacts
3. Dispatch `/task-breakdown`
4. Save tasks.md
5. Transition to `planning-complete`
6. Run alignment check
7. Transition to `implementation`

### Phase 5: Implementation Loop

For each task in dependency order:

```
projctl state transition --dir . --to task-start --task TASK-NNN
```

#### TDD Cycle with Commits

**Red phase:**
```bash
projctl state transition --dir . --to tdd-red --task TASK-NNN
# Write context with task description, acceptance criteria, architecture notes
projctl context write --dir . --task TASK-NNN --skill tdd-red --file <context.toml>
```
Dispatch `/tdd-red` → read result
```bash
projctl state transition --dir . --to commit-red
# Write context for /commit with red phase info
projctl context write --dir . --task TASK-NNN --skill commit-red --file <context.toml>
```
Dispatch `/commit` → read result

**Green phase:**
```bash
projctl state transition --dir . --to tdd-green --task TASK-NNN
# Write context with red phase result, test file locations
projctl context write --dir . --task TASK-NNN --skill tdd-green --file <context.toml>
```
Dispatch `/tdd-green` → read result
```bash
projctl state transition --dir . --to commit-green
projctl context write --dir . --task TASK-NNN --skill commit-green --file <context.toml>
```
Dispatch `/commit` → read result

**Refactor phase:**
```bash
projctl state transition --dir . --to tdd-refactor --task TASK-NNN
# Write context with green phase result, implementation files
projctl context write --dir . --task TASK-NNN --skill tdd-refactor --file <context.toml>
```
Dispatch `/tdd-refactor` → read result
```bash
projctl state transition --dir . --to commit-refactor
projctl context write --dir . --task TASK-NNN --skill commit-refactor --file <context.toml>
```
Dispatch `/commit` → read result

**Task audit:**
```bash
projctl state transition --dir . --to task-audit --task TASK-NNN
projctl context write --dir . --task TASK-NNN --skill task-audit --file <context.toml>
```
Dispatch `/task-audit` → read result

**Handle result:**
- **Pass:** `projctl state transition --dir . --to task-complete`
- **Fail (attempts < 3):** `projctl state transition --dir . --to task-retry` → back to tdd-red
- **Fail (attempts >= 3):** `projctl state transition --dir . --to task-escalated`

After each task, log progress:
```bash
projctl log write --dir . --level status --subject task-status --message "TASK-NNN: complete" --task TASK-NNN
```

Continue until all tasks complete or escalated:
```bash
projctl state transition --dir . --to implementation-complete
```

#### Escalation Handling

When a task is escalated, continue with other unblocked tasks. When all remaining tasks are blocked by escalations, present to user:

```
Implementation paused: N tasks escalated after 3 attempts.

TASK-005: [description]
  Attempt 1: [failure reason]
  Attempt 2: [failure reason]
  Attempt 3: [failure reason]

Options:
1. Provide guidance for escalated tasks
2. Mark as won't-fix and continue
3. Pause project
```

### Phase 6: Audit Loop

```bash
projctl state transition --dir . --to audit
```

Dispatch audit skills in sequence:
1. `/pm-audit` — requirements coverage
2. `/design-audit` — visual accuracy (if design was done)
3. `/architect-audit` — architecture adherence

For each audit result:
- **DEFECT findings:** Create fix tasks, run through TDD cycle, re-audit
- **SPEC_GAP / SPEC_REVISION:** Collect proposals
- **CROSS_SKILL:** Enter negotiation

#### Audit Fix Loop

```
audit → find defects → create fix tasks → TDD cycle → re-audit → repeat until clean
```

Only exit when zero defects.

#### Conflict Resolution (Bounded Negotiation)

When cross-skill conflicts arise:

```bash
projctl conflict create --dir . --skills "pm,architect" --traceability "REQ-001,ARCH-003" --description "..."
```

Negotiation (max 2 rounds):
1. Dispatch `/negotiate` with Side A position
2. Dispatch `/negotiate` with Side B counter-argument
3. Dispatch `/negotiate` with Side A round 2
4. Dispatch `/negotiate` with Side B round 2

If agreed: apply resolution, update artifacts
If not agreed: write to conflicts.md as open, continue with unblocked tasks

When all remaining work blocked by open conflicts, present to user.

```bash
projctl state transition --dir . --to audit-complete
```

### Phase 7: Completion

```bash
projctl state transition --dir . --to completion
```

1. Generate final-report.md with:
   - Summary of what was built
   - Requirements coverage matrix
   - Implementation stats (tasks, retries, escalations)
   - Learnings consolidated
   - Process improvement suggestions

2. Trigger `/meta-audit`:
   ```bash
   projctl context write --dir . --task PHASE --skill meta-audit --file <context.toml>
   ```
   Dispatch `/meta-audit` → present proposals to user

3. **Run end-of-command sequence** (see above - CRITICAL):
   ```bash
   projctl integrate features --dir .
   projctl trace repair --dir .
   projctl trace validate --dir .
   ```
   Loop until validation passes or user aborts.

4. Log completion:
   ```bash
   projctl log write --dir . --level phase --subject phase-change --message "Project complete"
   ```

---

## /project adopt

Adopt an existing codebase by inferring documentation from code and tests. Uses *-infer skills instead of *-interview skills, batches all escalations for editor handoff at the end.

### Prerequisites

Before adopting, run coverage analysis:

```bash
projctl coverage report --dir <repo-dir>
```

If recommendation is "migrate" (< 40% coverage), proceed with adopt.
If "preserve" (> 60% coverage), consider `/project align` instead.
If "evaluate" (40-60%), discuss with user.

### Initialization

```bash
# Ensure project config exists
projctl config init --dir <repo-dir>

# Initialize state for adopt mode
projctl state init --name <project-name> --dir <repo-dir> --mode adopt
projctl state transition --dir <repo-dir> --to adopt-analyze
projctl log write --dir <repo-dir> --level phase --subject phase-change --message "Starting codebase adoption"
```

### Phase 1: Analysis

Analyze codebase structure to understand what exists:

```bash
projctl coverage analyze --dir <repo-dir>
```

Review:
- README.md content and structure
- Existing docs/ files (if any)
- Public interfaces (exported functions, types)
- CLI help output (if applicable)
- Test file coverage

### Phase 2: Infer Requirements (pm-infer)

```bash
projctl state transition --dir <repo-dir> --to adopt-infer-pm
```

1. Write context for `/pm-infer`:
   - README summary
   - Existing requirements.md (if any)
   - Public API surface
   - Test names and descriptions
2. Dispatch `/pm-infer` via Skill tool
3. Skill generates requirements.md with REQ-NNN IDs
4. Skill collects escalations for ambiguous items
5. Transition: `projctl state transition --dir <repo-dir> --to adopt-infer-pm-complete`

### Phase 3: Infer Design (design-infer)

```bash
projctl state transition --dir <repo-dir> --to adopt-infer-design
```

1. Write context for `/design-infer`:
   - Requirements.md summary
   - CLI help / public interfaces
   - UI screenshots (if applicable)
2. Dispatch `/design-infer` via Skill tool
3. Skill generates design.md with DES-NNN IDs
4. Skill collects escalations
5. Transition: `projctl state transition --dir <repo-dir> --to adopt-infer-design-complete`

### Phase 4: Infer Architecture (architect-infer)

```bash
projctl state transition --dir <repo-dir> --to adopt-infer-arch
```

1. Write context for `/architect-infer`:
   - Requirements.md and design.md summaries
   - Package/module structure
   - Dependency graph
   - Code patterns observed
2. Dispatch `/architect-infer` via Skill tool
3. Skill generates architecture.md with ARCH-NNN IDs
4. Skill collects escalations
5. Transition: `projctl state transition --dir <repo-dir> --to adopt-infer-arch-complete`

### Phase 5: Map Tests (test-mapper)

```bash
projctl state transition --dir <repo-dir> --to adopt-map-tests
```

1. Write context for `/test-mapper`:
   - All artifact summaries
   - Test file locations
2. Dispatch `/test-mapper` via Skill tool
3. Skill adds TEST-NNN comments to test files
4. Skill adds traceability links (TEST → TASK)
5. Transition: `projctl state transition --dir <repo-dir> --to adopt-map-tests-complete`

### Phase 6: Escalation Resolution

```bash
projctl state transition --dir <repo-dir> --to adopt-escalations
```

1. Collect all escalations from previous phases
2. Write escalations.md file:
   ```bash
   projctl escalation write --dir <repo-dir> --id ESC-001 --category requirement --context "Analyzing README" --question "Is feature X required?"
   ```
3. Open in $EDITOR for user review:
   ```bash
   projctl escalation review --dir <repo-dir>
   ```
4. Parse user responses:
   - **resolved:** Apply decision to artifacts
   - **deferred:** Create ISSUE-NNN
   - **issue:** Create ISSUE-NNN with user description
5. Update artifacts based on resolutions
6. Transition: `projctl state transition --dir <repo-dir> --to adopt-escalations-complete`

### Phase 7: Generate Traceability

```bash
projctl state transition --dir <repo-dir> --to adopt-generate
```

1. Run `/alignment-check` to identify gaps
2. Ensure all `**Traces to:**` fields are present in artifacts
3. Validate completeness:
   ```bash
   projctl trace validate --dir <repo-dir>
   ```
4. Transition: `projctl state transition --dir <repo-dir> --to adopt-complete`

### Completion

1. **Run end-of-command sequence** (CRITICAL):
   ```bash
   projctl integrate features --dir <repo-dir>
   projctl trace repair --dir <repo-dir>
   projctl trace validate --dir <repo-dir>
   ```
   Loop until validation passes or user aborts.

2. Log completion:
   ```bash
   projctl log write --dir <repo-dir> --level phase --subject phase-result --message "Adoption complete. N requirements, N design decisions, N architecture decisions documented."
   ```

3. Present summary to user:
   ```
   Codebase Adoption Complete

   Artifacts generated:
     requirements.md    12 requirements (REQ-001 to REQ-012)
     design.md          8 design decisions (DES-001 to DES-008)
     architecture.md    15 architecture decisions (ARCH-001 to ARCH-015)

   Traceability: 45 **Traces to:** links embedded in artifacts

   Issues created: 3 (from deferred escalations)

   The codebase is now ready for /project new to add features.
   ```

---

## /project align

Lightweight alignment check for existing documented codebases. Re-analyzes code and updates documentation without full re-inference.

### When to Use

- After significant code changes
- When documentation drift is suspected
- Periodic housekeeping

### Flow

```bash
projctl state init --name <project-name>-align --dir <repo-dir> --mode align
projctl state transition --dir <repo-dir> --to align-analyze
```

### Phase 1: Analyze Drift

```bash
projctl coverage analyze --dir <repo-dir>
```

Compare artifacts to code:
- Check for new public interfaces not in architecture
- Check for new commands not in design
- Check for removed items still documented

Identify format and coverage gaps:
- **Format issues:** Table-format requirements, missing ARCH-NNN IDs, etc.
- **Missing artifacts:** No design.md, no architecture.md, feature-specific vs project-level
- **Prose content:** Requirements without IDs

### Phase 2: Normalize Formats

Before running infer skills, normalize existing artifacts to standard format.

**Requirements (if table format detected):**

Convert from:
```markdown
| ID | Capability | What it enables |
| -- | ---------- | --------------- |
| REQ-001 | Basic | Executable behavior |
```

To:
```markdown
### REQ-001: Basic

Executable behavior.

**Category:** Capability
```

Preserve all content, just restructure. Keep any Implementation Status sections as-is.

**Architecture (if missing ARCH-NNN IDs):**

Add ARCH-NNN IDs to major sections:
- Each `## Section` or significant architectural decision gets an ARCH-NNN
- Add `**Traces to:** REQ-NNN` fields based on what the section implements

**Design (if feature-specific or missing):**

If design.md exists but is feature-specific (covers only one feature):
1. Rename to `design-<feature>.md`
2. Create new project-level `design.md`

If design.md is missing entirely:
1. Flag for `/design-infer` to create

### Phase 3: Backfill Missing Artifacts

For each missing artifact, run the corresponding infer skill in create mode:

```bash
# If no design.md exists or only feature-specific ones
projctl state transition --dir <repo-dir> --to align-backfill-design
```
1. Dispatch `/design-infer` with `mode=create`
2. Skill analyzes CLI help, output formats, error patterns
3. Skill generates project-level design.md with DES-NNN IDs

```bash
# If architecture.md exists but needs ARCH-NNN IDs
projctl state transition --dir <repo-dir> --to align-backfill-arch
```
1. Dispatch `/architect-infer` with `mode=augment`
2. Skill adds ARCH-NNN IDs to existing sections
3. Skill adds `**Traces to:**` fields

### Phase 5: Run Infer Skills (update mode)

Each skill updates docs AND creates traceability links.

```bash
projctl state transition --dir <repo-dir> --to align-infer-pm
```
1. Dispatch `/pm-infer` with `mode=update`
2. Skill adds new REQs, assigns IDs to prose requirements
3. Creates ISSUE→REQ links (if applicable)

```bash
projctl state transition --dir <repo-dir> --to align-infer-design
```
4. Dispatch `/design-infer` with `mode=update`
5. Skill adds new DES items with `**Traces to:** REQ-NNN` fields

```bash
projctl state transition --dir <repo-dir> --to align-infer-arch
```
6. Dispatch `/architect-infer` with `mode=update`
7. Skill adds new ARCH items with `**Traces to:** DES-NNN, REQ-NNN` fields

### Phase 7: Map Tests

```bash
projctl state transition --dir <repo-dir> --to align-map-tests
```

1. Dispatch `/test-mapper`
2. Skill adds TEST-NNN entries to tests.md with `**Traces to:**` fields

### Phase 8: Fill Gaps with Alignment Check

```bash
projctl state transition --dir <repo-dir> --to align-check
```

1. Dispatch `/alignment-check`
2. Skill scans artifacts for IDs and `**Traces to:**` fields
3. Skill edits artifacts to add missing `**Traces to:**` fields
4. Skill runs `projctl trace validate` to verify
5. Reports what was auto-linked vs needs decision

### Phase 9: Validate and Complete

1. **Run end-of-command sequence** (CRITICAL):
   ```bash
   projctl integrate features --dir <repo-dir>
   projctl trace repair --dir <repo-dir>
   projctl trace validate --dir <repo-dir>
   ```
   Loop until validation passes or user aborts.

2. Verify complete traceability:
   - All REQ → DES/ARCH links
   - All DES → ARCH links
   - All ARCH → TASK links
   - All TEST → TASK links

3. Transition and report:
   ```bash
   projctl state transition --dir <repo-dir> --to align-complete
   ```

   ```
   Alignment Complete

   Links created: 45
     REQ→DES: 12
     DES→ARCH: 8
     ARCH→TASK: 15
     TEST→TASK: 10

   New items: 3
   Stale items flagged: 2
   ```

---

## /project integrate

Merge per-project documentation folder into top-level docs. Called automatically at the end of `/project new`, but can be invoked manually.

### When to Use

- After completing a project started with `/project new`
- To merge isolated project work into main documentation

### Prerequisites

- Per-project folder exists: `docs/projects/<project-name>/`
- Project is in a complete state

### Flow

```bash
projctl state init --name integrate-<project-name> --dir <repo-dir> --mode integrate
projctl state transition --dir <repo-dir> --to integrate-commit
```

1. **Commit per-project docs:**
   ```bash
   # Commit any uncommitted changes in per-project folder
   projctl log write --dir <repo-dir> --level status --subject integrate --message "Committing per-project docs"
   ```
   Dispatch `/commit` with message about completing project

2. **Merge into top-level:**
   ```bash
   projctl state transition --dir <repo-dir> --to integrate-merge
   projctl integrate merge --dir <repo-dir> --project <project-name>
   ```

   For each artifact in `docs/projects/<project-name>/`:
   - **requirements.md:** Append requirements to top-level, renumber IDs
   - **design.md:** Append design decisions, renumber IDs
   - **architecture.md:** Append architecture decisions, renumber IDs
   - **tasks.md:** Move completed tasks to history, keep active tasks

3. **Update cross-references:**
   - Update all ID references in merged content
   - Update `**Traces to:**` fields with renumbered IDs

4. **Clean up:**
   ```bash
   projctl state transition --dir <repo-dir> --to integrate-cleanup
   projctl integrate cleanup --dir <repo-dir> --project <project-name>
   ```
   - Delete per-project folder
   - Commit cleanup

5. **Run end-of-command sequence** (CRITICAL):
   ```bash
   projctl integrate features --dir <repo-dir>
   projctl trace repair --dir <repo-dir>
   projctl trace validate --dir <repo-dir>
   ```
   Loop until validation passes or user aborts.

6. **Complete:**
   ```bash
   projctl state transition --dir <repo-dir> --to integrate-complete
   projctl log write --dir <repo-dir> --level phase --subject phase-result --message "Integration complete"
   ```

### Output

```
Integration Complete

Merged from: docs/projects/my-feature/
  Requirements: REQ-015 to REQ-018 (4 new)
  Design: DES-012 to DES-014 (3 new)
  Architecture: ARCH-020 to ARCH-023 (4 new)
  Tasks: 8 completed, moved to history

Per-project folder removed.
Validation: PASSED
```

---

## /project continue

Resume an incomplete project.

### Steps:

1. **Read state:**
   ```bash
   projctl state get --dir <project-dir>
   ```

2. **Check for resolved conflicts:**
   ```bash
   projctl conflict check --dir <project-dir>
   ```
   If newly resolved, apply resolutions and continue.

3. **Display summary and wait for confirmation:**
   ```
   Resuming project: my-cli-tool
   Phase: implementation (tdd-green for TASK-004)
   Progress: 3/12 tasks complete
   Open conflicts: 1

   Ready to continue? [y/n]
   ```

4. **Resume from exact sub-phase** using state.toml's current_task and current_subphase

### Resume Map

| State | Resume Action |
|-------|---------------|
| init | Start PM interview |
| pm-complete | Start design interview |
| design-complete | Run alignment check, then architecture |
| architect-complete | Run alignment check, then task breakdown |
| planning-complete | Run alignment check, then start implementation |
| task-start/tdd-* | Resume TDD cycle at current sub-phase |
| commit-* | Resume commit at current phase |
| task-audit | Resume audit for current task |
| implementation-complete | Start audit loop |
| audit/audit-fix | Resume audit loop |
| audit-complete | Start completion |
| adopt-analyze | Resume analysis phase |
| adopt-infer-* | Resume at current inference skill |
| adopt-map-tests | Resume test mapping |
| adopt-escalations | Resume escalation resolution |
| adopt-generate | Resume traceability generation |
| align-analyze | Resume alignment analysis |
| align-infer-* | Resume at current inference skill |
| align-map-tests | Resume test mapping |
| align-check | Resume alignment check |
| align-complete | Report results |
| integrate-commit | Resume commit phase |
| integrate-merge | Resume merge phase |
| integrate-cleanup | Resume cleanup |

---

## Context Curation

Between skill dispatches, curate context:

1. Read previous skill's result file
2. Extract `context_for_next` summary
3. Add relevant project context (architecture notes, known issues)
4. Remove information irrelevant to next skill
5. Write focused context file for next dispatch

This keeps each skill focused and avoids context bloat.

## Manual Correction Tracking

When the user manually corrects something during the project:

```bash
projctl log write --dir . --level status --subject lesson --message "User corrected: [description]"
```

Track correction count. After every 2 corrections, trigger `/meta-audit`.

## Logging Levels

| Level | When | Subjects |
|-------|------|----------|
| **verbose** | Every dispatch, every result | thinking, skill-change |
| **status** | Task completions, findings | skill-result, task-status, alignment, conflict, lesson |
| **phase** | Phase transitions, major milestones | phase-change, phase-result |

## Error Handling

**Skill dispatch fails:**
- Log error, notify user, offer retry or skip

**Task repeatedly fails:**
- After 3 attempts, escalate. Continue with unblocked tasks.

**Conflict negotiation deadlock:**
- After 2 rounds, write as open conflict. Present when all tasks blocked.

**State corruption:**
- `projctl state get` will report errors. User can inspect/edit state.toml.
