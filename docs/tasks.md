# projctl Implementation Tasks

Task breakdown derived from [review-2025-01.md](./review-2025-01.md). Tasks are grouped by timeline and ordered by dependency.

---

## This Week (Foundation)

These tasks establish the control loop basics and must be completed first.

---

### TASK-001: Define result.toml schema

**Phase:** 0
**Priority:** High
**Timeline:** This Week

Define the structured result format that all skills must return. This is the foundation for the control loop's steps 7-8 (extracting decisions/learnings from skill outputs).

**Acceptance Criteria:**
- [x] Schema documented in `~/.claude/skills/shared/RESULT.md`
- [x] `[status]` section with `success` boolean
- [x] `[outputs]` section with `files_modified` list
- [x] `[[decisions]]` with `context`, `choice`, `reason`, `alternatives`
- [x] `[[learnings]]` with `content`
- [x] Example valid TOML included
- [x] Example invalid TOML with explanation of errors

**Test Requirements:**
- Unit: Schema parsing accepts valid TOML
- Unit: Schema parsing rejects missing required sections
- Property: Random valid TOML round-trips correctly

**Dependencies:** None

**Traces to:** Phase 0

---

### TASK-002: Implement projctl result validate command

**Phase:** 0
**Priority:** High
**Timeline:** This Week

CLI command to validate result.toml files against the schema. Used by orchestrator in control loop step 7.

**Acceptance Criteria:**
- [x] `projctl result validate --file PATH` validates against schema
- [x] Returns exit code 0 on success, 1 on failure
- [x] Error messages specify which section/field is invalid
- [x] `--format json` outputs structured validation results
- [x] Validates `[decisions]` items have required fields
- [x] Validates `[learnings]` items have required fields
- [x] Accepts empty `[[items]]` arrays (section present but no entries)

**Test Requirements:**
- Unit: `internal/result` package validates each field type
- Integration: CLI accepts valid files, rejects invalid
- Property: Any valid schema-conforming TOML passes validation

**Dependencies:** TASK-001

**Traces to:** Phase 0

---

### TASK-003: Update skills with result format documentation

**Phase:** 0
**Priority:** High
**Timeline:** This Week

Add `## Result Format` section to all 23 skills documenting the required result.toml structure.

**Acceptance Criteria:**
- [x] All skills in `~/.claude/skills/*/SKILL.md` have `## Result Format` section
- [x] Each section references shared/RESULT.md for schema
- [x] Each section shows skill-specific example with relevant decisions/learnings
- [x] Skills that don't produce files still have `[outputs]` with empty list
- [x] Compliance check: `grep -L "Result Format" ~/.claude/skills/*/SKILL.md` returns empty

**Test Requirements:**
- Integration: Script verifies all skills have result format section
- Integration: Script verifies each skill's example is valid TOML

**Dependencies:** TASK-001

**Traces to:** Phase 0

---

### TASK-004: Implement projctl context write command

**Phase:** 1
**Priority:** High
**Timeline:** This Week

Write skill context TOML files to the context directory for skill handoff. Skills reference this but it doesn't exist.

**Acceptance Criteria:**
- [x] `projctl context write --dir DIR --task TASK --skill SKILL --file INPUT` creates `context/{TASK}-{SKILL}.toml`
- [x] Content matches input file exactly
- [x] Creates `context/` directory if it doesn't exist
- [x] Territory map auto-injected via WriteWithRouting (no explicit flag)
- [x] Overwrites existing file without error

**Test Requirements:**
- Unit: File creation in correct location
- Unit: Directory creation when missing
- Unit: Content matches input
- Integration: Round-trip write then read produces same content

**Dependencies:** None

**Traces to:** Phase 1

---

### TASK-005: Implement projctl context read command

**Phase:** 1
**Priority:** High
**Timeline:** This Week

Read skill context or result files. Complements context write for bidirectional skill handoff.

**Acceptance Criteria:**
- [x] `projctl context read --dir DIR --task TASK --skill SKILL` reads `context/{TASK}-{SKILL}.toml`
- [x] `--result` flag reads `.result.toml` instead of `.toml`
- [x] Returns exit code 1 if file doesn't exist
- [x] `--format json` converts TOML to JSON output
- [x] Outputs raw content to stdout by default

**Test Requirements:**
- Unit: Correct file path resolution
- Unit: Result flag changes suffix
- Integration: Write then read produces same content

**Dependencies:** TASK-004

**Traces to:** Phase 1

---

### TASK-006: Implement projctl escalation commands

**Phase:** 1
**Priority:** High
**Timeline:** This Week

Full escalation workflow: list, write, resolve. Skills reference these for managing blockers.

**Acceptance Criteria:**
- [x] `projctl escalation write --dir DIR --id ESC-NNN --category CAT --question TEXT` creates entry
- [x] `projctl escalation list --dir DIR` shows all escalations with status
- [x] `--status pending|resolved` filters list
- [x] `projctl escalation resolve --dir DIR --id ESC-NNN --decision TEXT` marks resolved
- [x] Maintains markdown format compatible with existing escalations.md

**Test Requirements:**
- Unit: ID parsing and incrementing
- Unit: Category validation
- Integration: Full write/list/resolve workflow
- Property: Any valid question text round-trips correctly

**Dependencies:** None

**Traces to:** Phase 1

---

### TASK-007: Implement projctl integrate features command

**Phase:** 1
**Priority:** High
**Timeline:** This Week

Merge feature-*.md files into consolidated artifact files. End-of-command sequence is broken without this.

**Acceptance Criteria:**
- [x] `projctl integrate features --dir DIR` merges `docs/feature-*.md` into main artifacts
- [x] Handles empty/no feature files gracefully

**Test Requirements:**
- Unit: ID renumbering logic
- Unit: Trace reference updates
- Integration: Multiple feature files merge correctly
- Property: No ID collisions after merge

**Dependencies:** None

**Traces to:** Phase 1

---

### TASK-008: Implement projctl conflict commands

**Phase:** 1
**Priority:** Medium
**Timeline:** This Week

Conflict tracking workflow: create, check, list. Skills reference these but they don't exist.

**Acceptance Criteria:**
- [x] `projctl conflict create --dir DIR --skills X --description TEXT` creates entry
- [x] `projctl conflict list --dir DIR` shows all conflicts
- [x] `projctl conflict check --dir DIR` returns exit code based on open conflicts
- [x] Auto-increments ID

**Test Requirements:**
- Unit: ID parsing and incrementing
- Unit: Status filtering
- Integration: Full create/check/list workflow

**Dependencies:** None

**Traces to:** Phase 1

---

### TASK-009: Implement state transition preconditions

**Phase:** 11
**Priority:** High
**Timeline:** This Week

Add precondition checks to state transitions. Prevents skipping critical workflow steps.

**Acceptance Criteria:**
- [x] `projctl state transition` checks preconditions before executing
- [x] `pm-complete` requires requirements.md exists with REQ-NNN IDs
- [x] `design-complete` requires design.md exists with DES-NNN IDs
- [x] `architect-complete` requires trace validation passes
- [x] `tdd-green` requires test files exist
- [x] `tdd-refactor` requires tests pass
- [x] `task-complete` requires trace validation and AC complete
- [x] Error messages specify what's missing

**Test Requirements:**
- Unit: Each precondition check in isolation
- Integration: Valid transitions succeed
- Integration: Invalid transitions fail with actionable errors
- Property: No valid state can transition to invalid state

**Dependencies:** None

**Traces to:** Phase 11

---

### TASK-010: Prevent phase skipping in state machine

**Phase:** 11
**Priority:** High
**Timeline:** This Week

Extend TASK-009 to prevent invalid phase sequences (e.g., pm-interview directly to implementation).

**Acceptance Criteria:**
- [x] State machine defines valid transition graph (IsLegalTransition)
- [x] Direct jumps to implementation from early phases fail
- [x] Error includes legal targets
- [x] `--force` flag allows override for recovery scenarios

**Test Requirements:**
- Unit: Each invalid transition path rejected
- Integration: Full workflow follows valid path
- Property: Random valid transition sequence always succeeds

**Dependencies:** TASK-009

**Traces to:** Phase 11

---

### TASK-011: Implement projctl state next command

**Phase:** 12
**Priority:** High
**Timeline:** This Week

Determine next action (continue/stop) based on current state. Drives relentless continuation.

**Acceptance Criteria:**
- [x] `projctl state next --dir DIR` returns JSON with `action`, `reason`
- [x] `action: continue` includes `next_task`, `next_phase`
- [x] `action: stop` with `reason: all_complete` at terminal states
- [x] `action: stop` with `reason: error_pending` when error in state
- [x] `action: stop` with `reason: validation_failed` when AC incomplete

**Test Requirements:**
- Unit: Each action/reason combination
- Integration: Complete workflow shows continue until all_complete
- Property: No action: continue when blockers exist

**Dependencies:** TASK-009

**Traces to:** Phase 12

---

### TASK-012: Update /project skill with continuation rule

**Phase:** 12
**Priority:** High
**Timeline:** This Week

Add explicit continuation rule to /project skill. Eliminates "No response requested" pattern.

**Acceptance Criteria:**
- [x] SKILL-full.md has `## Continuation Rule (CRITICAL)` section
- [x] Compressed SKILL.md has "Continue" rule in Critical Rules table
- [x] Rule enforces checking `projctl state next` after each task

**Test Requirements:**
- Integration: Skill file contains continuation rule
- Behavioral: (manual) Workflow completes with 1 prompt instead of N

**Dependencies:** TASK-011

**Traces to:** Phase 12

---

### TASK-013: Define control loop in /project skill

**Phase:** 0 (coherence)
**Priority:** High
**Timeline:** This Week

Document the 11-step control loop explicitly in /project skill so orchestration is deterministic.

**Acceptance Criteria:**
- [x] SKILL.md has `## Control Loop` section (table format)
- [x] SKILL-full.md documents detailed control loop
- [x] Marks deterministic [D] vs agentic [A] steps

**Test Requirements:**
- Integration: Control loop section exists and is complete

**Dependencies:** TASK-001, TASK-004, TASK-011

**Traces to:** Phase 0 (coherence from Part 4.5)

---

### TASK-059: Implement AC validation function

**Phase:** 12
**Priority:** High
**Timeline:** This Week
**Status:** COMPLETE

Add function to validate acceptance criteria from tasks.md.

**Acceptance Criteria:**
- [x] `ValidateAcceptanceCriteria(dir, taskID)` parses acceptance criteria from tasks.md
- [x] Validates checkboxes: `- [x]` = complete, `- [ ]` = incomplete
- [x] Returns structured output: `{complete: N, incomplete: M, items: [...]}`
- [x] Error messages list specific incomplete AC items

**Test Requirements:**
- Unit: AC checkbox parsing from markdown
- Unit: Complete/incomplete counting

**Dependencies:** TASK-009

**Traces to:** Phase 12

---

### TASK-063: Wire AC validation into state transitions

**Phase:** 12
**Priority:** High
**Timeline:** This Week

Integrate AC validation as precondition for task-complete transitions.

**Acceptance Criteria:**
- [x] `projctl state transition --to task-complete` checks AC via precondition
- [x] Transition fails with error when AC incomplete
- [x] `--force` flag bypasses validation (for recovery only)
- [x] Exit code 1 when validation fails

**Test Requirements:**
- Integration: task-complete blocked when AC incomplete
- Integration: task-complete succeeds when all AC complete
- Integration: Force flag bypasses check

**Dependencies:** TASK-059

**Traces to:** Phase 12

---

### TASK-064: Wire AC validation into state.Next()

**Phase:** 12
**Priority:** High
**Timeline:** This Week

Integrate AC validation into state.Next() to return validation_failed when appropriate.

**Acceptance Criteria:**
- [x] `state.Next()` checks AC status for current task
- [x] Returns `action: stop, reason: validation_failed` when AC incomplete
- [x] Includes `details` with list of incomplete AC items
- [x] Only checks when current phase is task-audit or later

**Test Requirements:**
- Unit: state.Next returns validation_failed when AC incomplete
- Integration: Control loop stops when AC incomplete

**Dependencies:** TASK-059, TASK-063

**Traces to:** Phase 12

---

### TASK-065: Implement context budget alerting

**Phase:** 12
**Priority:** High
**Timeline:** This Week

Add automated context budget checking that warns/blocks when thresholds exceeded.

**Acceptance Criteria:**
- [x] `projctl context check --dir DIR` reads recent log entries with context_estimate
- [x] Compares estimate to configured thresholds (default: 80K warning, 90K limit)
- [x] Exit code 0 if under warning, 1 if over warning, 2 if over limit
- [x] Output includes recommendation: "Context at N% - consider compaction"
- [x] Thresholds configurable in project-config.toml `[budget]` section
- [x] Control loop can call this after each skill dispatch

**Test Requirements:**
- Unit: Threshold comparison logic
- Unit: Exit codes match thresholds
- Integration: Reads from actual log entries

**Dependencies:** TASK-061

**Traces to:** Phase 12

---

### TASK-060: Enforce sub-agent dispatch in /project skill

**Phase:** 12
**Priority:** High
**Timeline:** This Week

Update /project skill to mandate sub-agent dispatch for all skill work. Orchestrator should never read/write code files directly - only dispatch and collect results.

**Acceptance Criteria:**
- [x] /project SKILL.md has sub-agent mandate (Critical Rules table: "Dispatch | ALL code work via Skill/Task tool")
- [x] Rule enforced: all skill work dispatched, not inline
- [x] Full rules documented in SKILL-full.md

**Test Requirements:**
- Integration: Skill file contains sub-agent mandate section
- Behavioral: (manual) Orchestrator dispatches instead of inline work

**Dependencies:** TASK-012

**Traces to:** Phase 12

---

### TASK-061: Add context budget tracking to /project skill

**Phase:** 12
**Priority:** High
**Timeline:** This Week

Track context usage during orchestration and warn when approaching limits. Enables proactive compaction before control loop degrades.

**Acceptance Criteria:**
- [x] /project SKILL.md has context budget rule (Critical Rules table: "Context | At 80% warn, 90% compact")
- [x] Thresholds enforced: 80% warning, 90% compaction trigger
- [x] `projctl log write --contextestimate N` parameter implemented
- [x] `projctl context check` validates budget against thresholds
- [x] Full rules documented in SKILL-full.md

**Test Requirements:**
- Unit: Context estimate parameter accepted
- Integration: Log entries include context estimate
- Behavioral: (manual) Warning appears at threshold

**Dependencies:** None

**Traces to:** Phase 12

---

### TASK-062: Minimize /project SKILL.md token footprint

**Phase:** 12
**Priority:** High
**Timeline:** This Week

Compress /project skill to absolute minimum while preserving control loop. Move detailed docs to SKILL-full.md.

**Acceptance Criteria:**
- [x] /project SKILL.md < 1500 characters (currently 1260)
- [x] Control loop table preserved (essential)
- [x] Stop conditions table preserved (essential)
- [x] Sub-agent mandate compressed to single rule line
- [x] Context budget compressed to single rule line
- [x] Detailed examples moved to SKILL-full.md
- [x] `projctl skills docs --skillname project` returns full content

**Test Requirements:**
- Integration: Character count under limit
- Integration: Full docs retrievable via command

**Dependencies:** TASK-060, TASK-061

**Traces to:** Phase 12

---

### TASK-055: Create skills directory structure in projctl repo

**Phase:** 15
**Priority:** High
**Timeline:** This Week

Set up the skills/ directory in projctl repo and migrate existing skill definitions from ~/.claude/skills/.

**Acceptance Criteria:**
- [x] `skills/` directory exists in projctl repo root
- [x] All 23+ skills copied from ~/.claude/skills/
- [x] Each skill has SKILL.md (and any supporting files)
- [x] Directory structure matches ~/.claude/skills/ layout
- [x] Skills compile/validate correctly from new location

**Test Requirements:**
- Unit: Directory structure validation
- Integration: Skills readable from repo location

**Dependencies:** None

**Traces to:** Phase 15

---

### TASK-056: Implement projctl skills install command

**Phase:** 15
**Priority:** High
**Timeline:** This Week

Create symlinks from repo skills/ to ~/.claude/skills/.

**Acceptance Criteria:**
- [x] `projctl skills install` symlinks all skills to ~/.claude/skills/
- [x] `projctl skills install <name>` symlinks specific skill
- [x] Existing non-symlink directories trigger warning (conflict)
- [x] Existing symlinks are updated if target changed
- [x] `--force` flag overwrites conflicts
- [x] Reports what was linked

**Test Requirements:**
- Unit: Symlink creation, conflict detection
- Integration: Full install cycle creates working symlinks
- Behavioral: Claude Code can invoke skills after install

**Dependencies:** TASK-055

**Traces to:** Phase 15

---

### TASK-057: Implement projctl skills status command

**Phase:** 15
**Priority:** High
**Timeline:** This Week

Show which skills are symlinked vs local-only.

**Acceptance Criteria:**
- [x] `projctl skills status` lists all skills in repo
- [x] Shows "linked" for skills symlinked to ~/.claude/skills/
- [x] Shows "local" for skills only in ~/.claude/skills/ (not in repo)
- [x] Shows "conflict" for non-symlink directories with same name
- [x] Shows "missing" for repo skills not installed
- [x] Exit code reflects status (0=all linked, 1=some missing)

**Test Requirements:**
- Unit: Status detection for each state
- Integration: Accurate status after install/uninstall cycles

**Dependencies:** TASK-055

**Traces to:** Phase 15

---

### TASK-058: Implement projctl skills uninstall command

**Phase:** 15
**Priority:** Medium
**Timeline:** This Week

Remove symlinks without affecting local-only skills.

**Acceptance Criteria:**
- [x] `projctl skills uninstall` removes all symlinks to repo skills
- [x] `projctl skills uninstall <name>` removes specific symlink
- [x] Only removes symlinks pointing to repo (preserves local skills)
- [x] Reports what was removed
- [x] Idempotent (safe to run multiple times)

**Test Requirements:**
- Unit: Only symlinks removed, real directories preserved
- Integration: Uninstall then status shows "missing"

**Dependencies:** TASK-056

**Traces to:** Phase 15

---

## This Month (Reliability)

These tasks improve error handling and agent compliance.

---

### TASK-014: Implement model routing configuration

**Phase:** 2
**Priority:** Medium
**Timeline:** This Month

Add routing configuration schema and loading. Foundation for cost optimization.

**Acceptance Criteria:**
- [x] `project-config.toml` supports `[routing]` section
- [x] Fields: `simple`, `medium`, `complex` with model names
- [x] `threshold_lines` for automatic complexity classification
- [x] Default values when section missing: all "sonnet"
- [x] Validates model names against known list (haiku, sonnet, opus)

**Test Requirements:**
- Unit: Config parsing with valid values
- Unit: Defaults when missing
- Unit: Invalid model name rejected
- Property: Any valid config round-trips correctly

**Dependencies:** None

**Traces to:** Phase 2

---

### TASK-015: Inject model hint into skill context

**Phase:** 2
**Priority:** Medium
**Timeline:** This Month

Context write includes model recommendation from routing config.

**Acceptance Criteria:**
- [x] `projctl context write` adds `[routing]` section to output (default behavior)
- [x] Model field set based on skill name and routing config
- [x] Skills mapped to complexity via SkillComplexity map
- [x] Context includes `suggested_model` and `reason` fields

**Test Requirements:**
- Unit: Skill-to-model mapping
- Integration: Context file contains routing section

**Dependencies:** TASK-004, TASK-014

**Traces to:** Phase 2

---

### TASK-016: Log model field in skill dispatch

**Phase:** 2
**Priority:** Medium
**Timeline:** This Month

Log entries include which model was suggested/used. Enables cost analysis.

**Acceptance Criteria:**
- [x] `projctl log write` supports `--model MODEL` parameter
- [x] Model field included in JSONL output
- [x] Field is optional (backwards compatible)
- [x] `projctl log read` filters by model

**Test Requirements:**
- Unit: Model field in log entry
- Integration: Filter by model works

**Dependencies:** None

**Traces to:** Phase 2

---

### TASK-017: Document model routing limitations

**Phase:** 2
**Priority:** Medium
**Timeline:** This Month

Document that inline work is advisory-only; subagent dispatch can enforce model.

**Acceptance Criteria:**
- [x] README or design doc explains routing limitations
- [x] Explains: Task tool subagents can use specified model
- [x] Explains: Inline work in main session uses session model
- [x] Recommends: Use subagent dispatch for cost-critical skills
- [x] Provides example of Task tool with model hint

**Test Requirements:**
- Integration: Documentation exists and is accurate

**Dependencies:** TASK-015

**Traces to:** Phase 2

---

### TASK-018: Capture error details in state.toml

**Phase:** 7
**Priority:** High
**Timeline:** This Month

Failed transitions store error information for recovery. Foundation for graceful degradation.

**Acceptance Criteria:**
- [x] Failed `projctl state transition` writes `[error]` section to state.toml
- [x] Fields: `last_phase`, `last_task`, `target_phase`, `error_type`, `message`, `timestamp`
- [x] `retry_count` increments on repeated failures
- [x] `projctl state get` shows error section when present
- [x] Error cleared on successful transition

**Test Requirements:**
- Unit: Error capture on failure
- Unit: Retry count increment
- Integration: Error visible in state get output

**Dependencies:** TASK-009

**Traces to:** Phase 7

---

### TASK-019: Implement recovery actions in state machine

**Phase:** 7
**Priority:** High
**Timeline:** This Month

Offer retry/skip/escalate options when transitions fail.

**Acceptance Criteria:**
- [x] `projctl state recovery --dir DIR` shows recovery info
- [x] `available_actions` lists options
- [x] `projctl state retry --dir DIR` re-attempts last failed transition

**Test Requirements:**
- Unit: Recovery action availability logic
- Unit: Skip unblocks dependents
- Integration: Retry after fix succeeds

**Dependencies:** TASK-018

**Traces to:** Phase 7

---

### TASK-020: Continue with unblocked tasks after failure

**Phase:** 7
**Priority:** High
**Timeline:** This Month

State next identifies work that can proceed despite one task failing.

**Acceptance Criteria:**
- [x] `projctl state next` considers failed task as blocked, not complete
- [x] Returns `next_task` from `unblocked_tasks` if any exist
- [x] Only returns `action: stop` if ALL tasks blocked or complete
- [x] Log entry when continuing despite failure: "Continuing with unblocked work"

**Test Requirements:**
- Unit: Unblocked task identification
- Integration: Failure doesn't stop independent tasks

**Dependencies:** TASK-011, TASK-019

**Traces to:** Phase 7

---

### TASK-021: Identify critical rules in skills

**Phase:** 9
**Priority:** High
**Timeline:** This Month

Audit all 23 skills for MUST/NEVER/ALWAYS/CRITICAL rules. First step of CLAUDE.md migration.

**Acceptance Criteria:**
- [x] Script extracts rules matching MUST|NEVER|ALWAYS|CRITICAL from all skills
- [x] Output categorized: TDD discipline, traceability, commit format, evidence-based, other
- [x] Duplicates identified (same rule in multiple skills)
- [x] Output stored in `docs/critical-rules-audit.md`
- [x] At least 10 critical rules identified

**Test Requirements:**
- Integration: Audit script runs and produces output

**Dependencies:** None

**Traces to:** Phase 9

---

### TASK-022: Migrate critical rules to CLAUDE.md

**Phase:** 9
**Priority:** High
**Timeline:** This Month

Move identified rules to ~/.claude/CLAUDE.md with skill references.

**Acceptance Criteria:**
- [x] TDD discipline rule in CLAUDE.md: "Never weaken tests to pass"
- [x] Traceability rule in CLAUDE.md: "Use `**Traces to:**` inline, never traceability.toml"
- [x] Commit format rule in CLAUDE.md: "AI-Used: [claude] trailer"
- [x] Evidence-based rule in CLAUDE.md: "All audit findings require concrete proof"
- [x] Each rule references source skill for full details
- [x] No full rule duplication - CLAUDE.md has summary, skill has details

**Test Requirements:**
- Integration: Rules present in CLAUDE.md
- Integration: Skills reference CLAUDE.md, don't duplicate

**Dependencies:** TASK-021

**Traces to:** Phase 9

---

### TASK-023: Verify CLAUDE.md token budget

**Phase:** 9
**Priority:** High
**Timeline:** This Month

Ensure CLAUDE.md stays under 3000 tokens after migration.

**Acceptance Criteria:**
- [x] CLAUDE.md character count / 4 < 3000 (rough token estimate)
- [x] Script to check: `wc -c ~/.claude/CLAUDE.md | awk '{print int($1/4)}'`
- [x] If over budget, compress rules using pipe-delimited indices
- [x] Compression technique documented

**Test Requirements:**
- Integration: Token budget check script

**Dependencies:** TASK-022

**Traces to:** Phase 9

---

### TASK-024: Add visual acceptance criteria to CLAUDE.md

**Phase:** 14
**Priority:** Medium
**Timeline:** This Month

Add rule requiring visual verification for UI tasks. Per lessons learned on UI testing.

**Acceptance Criteria:**
- [x] CLAUDE.md includes: "For UI tasks, acceptance criteria MUST include visual verification"
- [x] References screenshot SSIM for regression detection
- [x] Explains: "DOM existence is insufficient - verify visual correctness"
- [x] Links to existing screenshot SSIM implementation

**Test Requirements:**
- Integration: Rule present in CLAUDE.md

**Dependencies:** TASK-022

**Traces to:** Phase 14

---

### TASK-025: Add ui flag to task validation

**Phase:** 14
**Priority:** Medium
**Timeline:** This Month

Task breakdown and validation recognize `ui: true/false` flag.

**Acceptance Criteria:**
- [x] tasks.md format supports `**UI:** true/false` field
- [x] `projctl task validate --dir DIR --task TASK` checks for visual evidence if ui=true
- [x] Visual evidence: screenshot file referenced in `**Visual evidence:**` field
- [x] Missing visual evidence returns error: "UI task requires visual verification"
- [x] Non-UI tasks validated without visual evidence

**Test Requirements:**
- Unit: UI flag parsing
- Integration: UI task without evidence fails
- Integration: UI task with evidence passes

**Dependencies:** None

**Traces to:** Phase 14

---

### TASK-026: Graceful degradation for Chrome DevTools MCP

**Phase:** 14
**Priority:** Medium
**Timeline:** This Month

When Chrome DevTools MCP unavailable, allow manual verification flag.

**Acceptance Criteria:**
- [x] Check MCP availability at validation start
- [x] If unavailable for UI task: warn but allow `--manual-visual-verified` flag
- [x] Flag requires explicit acknowledgment: "I manually verified visual correctness"
- [x] Log entry when manual verification used
- [x] Non-UI tasks unaffected by MCP availability

**Test Requirements:**
- Unit: Availability check logic
- Integration: Manual flag accepted when MCP unavailable

**Dependencies:** TASK-025

**Traces to:** Phase 14

---

## Next Month (Efficiency)

These tasks optimize cost and enable parallelism.

---

### TASK-027: Add token estimate to log entries

**Phase:** 3
**Priority:** Medium
**Timeline:** Next Month

Log entries include rough token count for cost tracking.

**Acceptance Criteria:**
- [x] `projctl log write` calculates token estimate from message length (chars/4)
- [x] `tokens_estimate` field in JSONL output
- [x] Estimation works for any message content
- [x] Optional `--tokens N` to override estimate with known value

**Test Requirements:**
- Unit: Token estimation formula
- Integration: Log entries contain token field

**Dependencies:** None

**Traces to:** Phase 3

---

### TASK-028: Implement projctl usage report command

**Phase:** 3
**Priority:** Medium
**Timeline:** Next Month

Aggregate token usage from logs. Provides cost visibility.

**Acceptance Criteria:**
- [x] `projctl usage report --dir DIR` sums tokens from project logs
- [x] `--session SESSION` filters to specific session
- [ ] `--project NAME` filters to specific project (cross-project)
- [x] `--format json` outputs structured data with totals and breakdowns
- [x] Breakdown by model if model field present
- [x] Human-readable default format with summary

**Test Requirements:**
- Unit: Token summation
- Unit: Filtering by session/project
- Property: Totals match sum of individual entries

**Dependencies:** TASK-027

**Traces to:** Phase 3

---

### TASK-029: Implement cost budget alerts

**Phase:** 3
**Priority:** Medium
**Timeline:** Next Month

Warn when session token usage exceeds threshold.

**Acceptance Criteria:**
- [x] `project-config.toml` supports `[budget]` section with `warning_tokens` and `limit_tokens`
- [x] `projctl usage check --session` compares current usage to warning threshold
- [x] Returns exit code 1 if over warning, 2 if over limit
- [x] Output includes recommendation: "consider using haiku for remaining tasks"
- [x] Orchestrator can call this after each skill dispatch

**Test Requirements:**
- Unit: Threshold comparison
- Integration: Exit codes match thresholds

**Dependencies:** TASK-028

**Traces to:** Phase 3

**Status:** Complete - Implemented in internal/usage/usage.go and cmd/projctl/usage.go

---

### TASK-030: Parse task dependencies from tasks.md

**Phase:** 5
**Priority:** Medium
**Timeline:** Next Month

Extract DAG from tasks.md for parallel dispatch analysis.

**Acceptance Criteria:**
- [x] Parse `**Depends on:** TASK-XXX, TASK-YYY` fields
- [x] Build dependency graph in memory
- [x] Detect cycles and report error
- [x] `projctl tasks deps --dir DIR` outputs graph in DOT or JSON format
- [x] Handles tasks with no dependencies (roots)

**Test Requirements:**
- Unit: Dependency parsing
- Unit: Cycle detection
- Property: Valid DAG always acyclic

**Dependencies:** None

**Traces to:** Phase 5

**Status:** Complete - Implemented in internal/task/deps.go and cmd/projctl/tasks.go

---

### TASK-031: Implement projctl tasks parallel command

**Phase:** 5
**Priority:** Medium
**Timeline:** Next Month

Identify tasks that can run concurrently based on dependency analysis.

**Acceptance Criteria:**
- [x] `projctl tasks parallel --dir DIR` returns list of independent tasks
- [x] Independent = no pending blockedBy tasks
- [x] Considers task status (only pending tasks)
- [x] Returns empty list if all tasks blocked or complete
- [x] `--format json` for programmatic use

**Test Requirements:**
- Unit: Independence detection
- Integration: Correct tasks identified as parallel

**Dependencies:** TASK-030

**Traces to:** Phase 5

**Status:** Complete - Implemented in internal/task/parallel.go and cmd/projctl/tasks.go

---

### TASK-032: Prepare parallel context files

**Phase:** 5
**Priority:** Medium
**Timeline:** Next Month

Create multiple context files for concurrent skill dispatch.

**Acceptance Criteria:**
- [x] `projctl context write-parallel --dir DIR --tasks TASK-001,TASK-002` creates context for each
- [x] Each context file named `context/{TASK}-{SKILL}.toml`
- [x] Shared territory map included in all contexts
- [x] Skill determined from task phase (tdd-red for pending implementation tasks)

**Test Requirements:**
- Unit: Multiple file creation
- Integration: All context files valid

**Dependencies:** TASK-004, TASK-031

**Traces to:** Phase 5

**Status:** Complete - Implemented in internal/context/parallel.go and cmd/projctl/context.go

---

### TASK-033: Collect and merge parallel results

**Phase:** 5
**Priority:** Medium
**Timeline:** Next Month

Wait for parallel dispatches and combine results.

**Acceptance Criteria:**
- [x] `projctl result collect --dir DIR --tasks TASK-001,TASK-002` waits for result files
- [x] Timeout configurable (default 10 minutes)
- [x] Merges escalations from all results into single list
- [x] Merges learnings from all results
- [x] Reports: "3/3 tasks complete" or "2/3 tasks complete, 1 failed"
- [x] Failed task details in output

**Test Requirements:**
- Unit: Result merging
- Integration: Multiple results collected
- Integration: Partial failure handled

**Dependencies:** TASK-002

**Traces to:** Phase 5

**Status:** Complete - Implemented in internal/result/collect.go and cmd/projctl/result.go

---

### TASK-034: Implement projctl map command

**Phase:** 6
**Priority:** Medium
**Timeline:** Next Month

Generate compressed territory map for codebase exploration.

**Acceptance Criteria:**
- [x] `projctl map --dir DIR --output PATH` produces territory.toml
- [x] `[structure]` section: root, languages, build_tool, test_framework
- [x] `[entry_points]` section: cli, public_api locations
- [x] `[packages]` section: count, internal package list
- [x] `[tests]` section: pattern, count
- [x] `[docs]` section: readme, artifacts list
- [x] Output < 1000 tokens (< 4000 chars)

**Test Requirements:**
- Unit: Each section generation
- Integration: Map for projctl itself is accurate
- Property: Output always under token budget

**Dependencies:** None

**Traces to:** Phase 6

**Status:** Complete - Implemented in internal/territory/ and cmd/projctl/map.go

---

### TASK-035: Cache territory map

**Phase:** 6
**Priority:** Medium
**Timeline:** Next Month

Reuse territory map within session to avoid redundant exploration.

**Acceptance Criteria:**
- [x] `projctl map --cached` returns existing map if recent (< 1 hour old)
- [x] Cache stored in `context/territory.toml`
- [x] `--force` regenerates regardless of cache
- [x] Cache invalidated if file count changes significantly (> 10%)
- [x] Cache timestamp stored in map

**Test Requirements:**
- Unit: Cache age check
- Unit: Invalidation logic
- Integration: Cached map returned quickly

**Dependencies:** TASK-034

**Traces to:** Phase 6

**Status:** Complete - Implemented in internal/territory/cache.go

---

### TASK-036: Inject territory map into skill context

**Phase:** 6
**Priority:** Medium
**Timeline:** Next Month

Context write includes territory map automatically.

**Acceptance Criteria:**
- [x] `projctl context write` checks for cached territory map
- [x] If exists, merges into context under `[territory]` section
- [x] Skills see territory without requesting it
- [x] Orchestrator doesn't need explicit `--with-territory` flag

**Test Requirements:**
- Integration: Context includes territory when map exists

**Dependencies:** TASK-004, TASK-035

**Traces to:** Phase 6

**Status:** Complete - Implemented in internal/context/context.go (WriteWithRouting injects territory)

---

### TASK-037: Create compressed SKILL.md format

**Phase:** 10
**Priority:** Medium
**Timeline:** Next Month

Define compressed skill format (< 500 tokens) with reference to full docs.

**Acceptance Criteria:**
- [x] Template: Quick Reference, Failure Hints, Output Format, Full Documentation link
- [x] Quick Reference: pipe-delimited key rules
- [x] Failure Hints: common issues and fixes
- [x] Output Format: result.toml requirements
- [x] Full Documentation: link to SKILL-full.md or `projctl skill docs`
- [x] Template documented in `~/.claude/skills/shared/SKILL-TEMPLATE-COMPRESSED.md`

**Test Requirements:**
- Integration: Template exists and is complete

**Dependencies:** None

**Traces to:** Phase 10

**Status:** Complete - Template at ~/.claude/skills/shared/SKILL-TEMPLATE-COMPRESSED.md

---

### TASK-038: Implement projctl skill docs command

**Phase:** 10
**Priority:** Medium
**Timeline:** Next Month

Retrieve full skill documentation on demand.

**Acceptance Criteria:**
- [x] `projctl skill docs SKILL-NAME` outputs full documentation
- [x] Reads from SKILL-full.md if exists, falls back to SKILL.md
- [x] `--section NAME` outputs specific section only
- [x] List available skills with `projctl skill list`

**Test Requirements:**
- Unit: File resolution logic
- Integration: Full docs retrieved correctly

**Dependencies:** None

**Traces to:** Phase 10

**Status:** Complete - Implemented in cmd/projctl/skills.go

---

### TASK-039: Compress existing skills

**Phase:** 10
**Priority:** Medium
**Timeline:** Next Month

Transform all 23 skills to compressed format with SKILL-full.md backup.

**Acceptance Criteria:**
- [x] Each skill has SKILL.md (< 500 tokens) and SKILL-full.md (original content)
- [x] Compressed SKILL.md follows template from TASK-037
- [x] All original content preserved in SKILL-full.md
- [x] Compliance check: no SKILL.md > 2000 chars
- [x] `projctl skill docs` works for all skills

**Test Requirements:**
- Integration: All skills compressed
- Integration: All skills retrievable via docs command
- Property: No information lost (full contains original)

**Dependencies:** TASK-037, TASK-038

**Traces to:** Phase 10

**Status:** Complete - All skills compressed, SKILL-full.md backups created

---

## Next Quarter (Polish)

These tasks add advanced features and learning capabilities.

---

### TASK-040: Implement projctl corrections log command

**Phase:** 4
**Priority:** Lower
**Timeline:** Next Quarter

Track corrections for learning loop. Orchestrator calls this, not agent.

**Acceptance Criteria:**
- [x] `projctl corrections log --dir DIR --message TEXT --context CONTEXT` appends to corrections.jsonl
- [x] Entry includes: timestamp, message, context, session_id
- [x] Session ID from environment variable or `--session` flag
- [x] Cross-project corrections stored in `~/.claude/corrections.jsonl`
- [x] Project-specific corrections in `{dir}/corrections.jsonl`

**Test Requirements:**
- Unit: Entry format
- Integration: Append to existing file
- Property: Any message text stored correctly

**Dependencies:** None

**Traces to:** Phase 4

**Status:** Complete - Implemented in internal/corrections/ and cmd/projctl/corrections.go

---

### TASK-041: Implement projctl corrections count command

**Phase:** 4
**Priority:** Lower
**Timeline:** Next Quarter

Count corrections for meta-audit trigger threshold.

**Acceptance Criteria:**
- [x] `projctl corrections count --dir DIR` returns count
- [x] `--since TIMESTAMP` filters to recent corrections only
- [x] `--session SESSION` filters to specific session
- [x] Used by orchestrator in control loop step 9.5
- [x] Exit code 0 always (count in output, not exit code)

**Test Requirements:**
- Unit: Counting logic
- Unit: Filtering by time/session

**Dependencies:** TASK-040

**Traces to:** Phase 4

**Status:** Complete - Implemented alongside TASK-040 in cmd/projctl/corrections.go

---

### TASK-042: Implement projctl corrections analyze command

**Phase:** 4
**Priority:** Lower
**Timeline:** Next Quarter

Detect patterns in corrections for CLAUDE.md proposals.

**Acceptance Criteria:**
- [x] `projctl corrections analyze --dir DIR` identifies repeated corrections
- [x] Groups similar corrections (fuzzy matching on keywords)
- [x] Reports patterns with count >= 2
- [x] Proposes CLAUDE.md addition for each pattern
- [x] `--min-occurrences N` changes threshold
- [x] Output includes pattern, count, proposed rule

**Test Requirements:**
- Unit: Pattern detection
- Integration: Proposals generated for repeated corrections

**Dependencies:** TASK-041

**Traces to:** Phase 4

**Status:** Complete - Implemented in internal/corrections/corrections.go

---

### TASK-043: Integrate correction logging into control loop

**Phase:** 4
**Priority:** Lower
**Timeline:** Next Quarter

Orchestrator detects correction signals and logs automatically.

**Acceptance Criteria:**
- [x] /project skill documents correction signal patterns
- [x] Patterns: "that's wrong", "no, do X", "I said X not Y", "remember that"
- [x] Control loop step 0 checks for patterns before skill dispatch
- [x] Pattern match triggers `projctl corrections log` call
- [x] Correction extracted and logged with current phase/task context

**Test Requirements:**
- Integration: Pattern detection in sample messages
- Behavioral: (manual) Corrections logged automatically

**Dependencies:** TASK-013, TASK-040

**Traces to:** Phase 4

**Status:** Complete - Added to /project skill (SKILL.md and SKILL-full.md)

---

### TASK-044: Implement meta-audit trigger in control loop

**Phase:** 4
**Priority:** Lower
**Timeline:** Next Quarter

Auto-inject /meta-audit when correction threshold reached.

**Acceptance Criteria:**
- [x] Control loop step 9.5 calls `projctl corrections count --since=session-start`
- [x] If count >= threshold (default 2), inject /meta-audit as next skill
- [x] Reset session counter after meta-audit runs
- [x] Threshold configurable in project-config.toml
- [x] Log entry: "Correction threshold reached, triggering meta-audit"

**Test Requirements:**
- Unit: Threshold comparison
- Integration: Meta-audit injection when threshold met

**Dependencies:** TASK-041, TASK-013

**Traces to:** Phase 4

**Status:** Complete - Documented in /project skill control loop (step 6 & 8)

---

### TASK-045: Implement projctl refactor rename command

**Phase:** 8
**Priority:** Lower
**Timeline:** Next Quarter

LSP-backed symbol rename. Deterministic, finds all references.

**Acceptance Criteria:**
- [x] `projctl refactor rename --dir DIR --symbol NAME --to NEWNAME` renames symbol
- [x] Uses gopls for Go projects
- [x] Updates all references across package
- [x] Atomic: no changes if rename fails
- [x] Reports: "Renamed X in N files"
- [x] Exit code 1 if symbol not found or conflict exists

**Test Requirements:**
- Unit: gopls command construction
- Integration: Rename across multiple files
- Integration: Conflict detection

**Dependencies:** None

**Traces to:** Phase 8

**Status:** Complete - Implemented in internal/refactor/ and cmd/projctl/refactor.go

---

### TASK-046: Implement projctl refactor extract-function command

**Phase:** 8
**Priority:** Lower
**Timeline:** Next Quarter

LSP-backed function extraction.

**Acceptance Criteria:**
- [ ] `projctl refactor extract-function --file PATH --lines START-END --name NAME` extracts function
- [ ] Uses gopls for Go projects
- [ ] Detects variables that need to be parameters
- [ ] Detects return values
- [ ] Produces compilable code
- [ ] Atomic: no changes if extraction fails

**Test Requirements:**
- Unit: gopls command construction
- Integration: Extract with parameters
- Integration: Code compiles after extraction

**Dependencies:** None

**Traces to:** Phase 8

---

### TASK-047: Add capability detection for LSP

**Phase:** 8
**Priority:** Lower
**Timeline:** Next Quarter

Check LSP availability and use fallbacks when unavailable.

**Acceptance Criteria:**
- [x] `projctl refactor` commands check for gopls availability
- [x] If unavailable: error with installation instructions, don't attempt
- [x] Capability stored in context for skills to check
- [x] Skills can request LSP operations only if available
- [x] Fallback documented: manual refactoring via LLM edit (skills use Edit tool)

**Test Requirements:**
- Unit: Availability check
- Integration: Graceful error when gopls missing

**Dependencies:** TASK-045

**Traces to:** Phase 8

**Status:** Complete - `projctl refactor capabilities` command + context injection

---

### TASK-048: Implement projctl memory learn command

**Phase:** 13
**Priority:** Medium
**Timeline:** Next Quarter

Store explicit learnings in global memory index.

**Acceptance Criteria:**
- [ ] `projctl memory learn --message TEXT` appends to `~/.claude/memory/index.md`
- [ ] Entry includes: timestamp, message, optional project context
- [ ] Format: markdown list item with timestamp prefix
- [ ] Creates index.md if doesn't exist
- [ ] `--project NAME` tags learning with project

**Test Requirements:**
- Unit: Entry format
- Integration: Append to existing index
- Property: Any message text stored correctly

**Dependencies:** None

**Traces to:** Phase 13

---

### TASK-049: Implement projctl memory decide command

**Phase:** 13
**Priority:** Medium
**Timeline:** Next Quarter

Log decisions with reasoning and alternatives.

**Acceptance Criteria:**
- [ ] `projctl memory decide --context CTX --choice CHOICE --reason REASON --alternatives ALT1,ALT2`
- [ ] Appends to `~/.claude/memory/decisions/{DATE}-{PROJECT}.jsonl`
- [ ] Entry: JSON with all fields plus timestamp
- [ ] Creates directory structure if needed
- [ ] Orchestrator extracts from result.toml and calls this

**Test Requirements:**
- Unit: Entry format
- Integration: Decision logged correctly

**Dependencies:** None

**Traces to:** Phase 13

---

### TASK-050: Implement projctl memory session-end command

**Phase:** 13
**Priority:** Medium
**Timeline:** Next Quarter

Generate compressed session summary.

**Acceptance Criteria:**
- [ ] `projctl memory session-end --project NAME` creates session summary
- [ ] Output: `~/.claude/memory/sessions/{DATE}-{PROJECT}.md`
- [ ] Includes: session duration, tasks completed, key decisions, learnings
- [ ] Compressed to < 2000 characters
- [ ] Extracts from project logs and decisions.jsonl

**Test Requirements:**
- Unit: Summary generation
- Property: Output always under size limit

**Dependencies:** TASK-049

**Traces to:** Phase 13

---

### TASK-051: Implement projctl memory grep command

**Phase:** 13
**Priority:** Medium
**Timeline:** Next Quarter

Structured search across memory files.

**Acceptance Criteria:**
- [ ] `projctl memory grep PATTERN` searches index.md and sessions/
- [ ] Returns matching lines with context
- [ ] `--project NAME` limits to specific project
- [ ] `--decisions` also searches decisions/ files
- [ ] Output includes source file and line number

**Test Requirements:**
- Unit: Pattern matching
- Integration: Cross-file search

**Dependencies:** TASK-048

**Traces to:** Phase 13

---

### TASK-052: Implement projctl memory query command

**Phase:** 13
**Priority:** Medium
**Timeline:** Next Quarter

Semantic search using embeddings.

**Acceptance Criteria:**
- [ ] `projctl memory query TEXT` returns semantically similar memories
- [ ] Uses SQLite-vec for vector storage
- [ ] Uses local ONNX model for embeddings (no API calls)
- [ ] Returns top N results (default 5) with similarity scores
- [ ] Searches index.md and session summaries
- [ ] Creates embeddings.db on first use

**Test Requirements:**
- Unit: Embedding generation
- Integration: Semantic matching works

**Dependencies:** TASK-048, TASK-050

**Traces to:** Phase 13

---

### TASK-053: Inject memory into skill context

**Phase:** 13
**Priority:** Medium
**Timeline:** Next Quarter

Context write queries memory and includes relevant results.

**Acceptance Criteria:**
- [ ] `projctl context write --inject-memory QUERY` queries memory
- [ ] Includes top 3 relevant memories in context under `[memory]` section
- [ ] Query derived from task description if not specified
- [ ] Memory injection automatic for certain phases: architect-interview, pm-interview
- [ ] Memories compressed to < 500 tokens total

**Test Requirements:**
- Integration: Memory appears in context
- Property: Memory section under token limit

**Dependencies:** TASK-004, TASK-052

**Traces to:** Phase 13

---

### TASK-054: Delete parser/discovery/collect stubs

**Phase:** Housekeeping
**Priority:** Lower
**Timeline:** Next Quarter

Remove stub files identified in review.

**Acceptance Criteria:**
- [x] Delete `internal/parser/` if stub - N/A: parser is real implementation (3000 LOC)
- [x] Delete `internal/discovery/` if stub - N/A: doesn't exist (consolidated in parser/)
- [x] Delete `internal/collect/` if stub - N/A: doesn't exist (consolidated in parser/)
- [x] Update any imports that reference these - N/A
- [x] Verify build still passes - N/A

**Test Requirements:**
- Integration: Build succeeds after deletion

**Dependencies:** None

**Traces to:** Part 4 Simplifications

**Status:** N/A - Parser is a real ~3000 LOC implementation, discovery/collect don't exist as separate packages

---

## Dependency Summary

```
TASK-001 (result schema)
    └── TASK-002 (result validate)
    └── TASK-003 (skill result docs)
    └── TASK-013 (control loop)

TASK-004 (context write)
    └── TASK-005 (context read)
    └── TASK-015 (model hint injection)
    └── TASK-032 (parallel contexts)
    └── TASK-036 (territory injection)
    └── TASK-053 (memory injection)

TASK-009 (state preconditions)
    └── TASK-010 (phase skipping)
    └── TASK-011 (state next)
    └── TASK-018 (error capture)
    └── TASK-059 (completion gate)

TASK-012 (continuation rule)
    └── TASK-060 (sub-agent mandate)
    └── TASK-061 (context budget)
        └── TASK-065 (context budget alerting)
    └── TASK-062 (minimize /project)

TASK-059 (AC validation function)
    └── TASK-063 (wire into state transitions)
    └── TASK-064 (wire into state.Next)

TASK-011 (state next)
    └── TASK-012 (continuation rule)
    └── TASK-013 (control loop)
    └── TASK-020 (unblocked tasks)

TASK-021 (critical rules audit)
    └── TASK-022 (migrate to CLAUDE.md)
    └── TASK-023 (token budget)
    └── TASK-024 (visual criteria)

TASK-030 (task deps)
    └── TASK-031 (tasks parallel)
    └── TASK-032 (parallel contexts)

TASK-034 (map)
    └── TASK-035 (cache)
    └── TASK-036 (injection)

TASK-037 (compressed template)
    └── TASK-039 (compress skills)

TASK-040 (corrections log)
    └── TASK-041 (corrections count)
    └── TASK-042 (corrections analyze)
    └── TASK-043 (control loop integration)
    └── TASK-044 (meta-audit trigger)

TASK-048 (memory learn)
    └── TASK-051 (memory grep)
    └── TASK-052 (memory query)
    └── TASK-053 (memory injection)

TASK-055 (skills directory)
    └── TASK-056 (skills install)
    └── TASK-057 (skills status)
    └── TASK-058 (skills uninstall)
```

---

**Total:** 65 tasks across 16 phases (0-15) plus housekeeping

**This Week:** 24 tasks (foundation) - includes Phase 15 skill management + orchestrator enforcement
**This Month:** 13 tasks (reliability)
**Next Month:** 13 tasks (efficiency)
**Next Quarter:** 15 tasks (polish)
