# projctl Requirements

Requirements derived from [review-2025-01.md](./review-2025-01.md) executive summary.

---

### REQ-001: Dependable Agent Orchestrator

Build a dependable Claude Code agent orchestrator with:
- Maximum autonomy with minimum human intervention
- Cheapest agents + smallest context possible
- Dedicated deterministic tooling over LLM judgment
- Learning from corrections
- Behavioral correctness with tests
- Confident traceability from idea to implementation
- Support for existing codebases, alignment, and new projects

**Source:** User stated goals in review-2025-01.md

---

### REQ-002: Context-Aware Interview Skills

As a user, I want interview skills to understand existing context before asking questions, so that I'm not asked about information that's already available.

**Acceptance Criteria:**
- [ ] Interview skills (pm-interview-producer, architect-interview-producer, design-interview-producer) gather context BEFORE asking user questions
- [ ] Skills use existing infrastructure: territory map, memory query, context-explorer
- [ ] Skills assess what's already answered by issue + gathered context
- [ ] Skills only ask questions about genuinely missing information
- [ ] When context gathering fails (territory map error, memory query timeout), skill sends message to lead with diagnostic information
- [ ] When gathered context contains contradictory information, skill sends message to lead with conflicting statements for user resolution

**Priority:** P0

**Depends on:** None

**Source:** ISSUE-61

---

### REQ-003: Adaptive Interview Depth

As a user, I want interview depth to adapt based on information gaps, so that simple issues get quick confirmation while complex issues get thorough exploration.

**Acceptance Criteria:**
- [ ] Skills assess information gaps against their key responsibilities (as documented in each skill's SKILL.md)
- [ ] Gap size determined by percentage of key questions answerable from context: ≥80% = small gap, 50-79% = medium gap, <50% = large gap
- [ ] Small gaps: skill asks 1-2 confirmation questions maximum
- [ ] Medium gaps: skill asks 3-5 clarification questions
- [ ] Large gaps: skill conducts full interview sequence (6+ questions covering all phases)
- [ ] Depth decision is explicit and traceable in message context (includes gap percentage and question count)
- [ ] When both issue description AND gathered context are sparse (<20% coverage), skill conducts full interview by default

**Priority:** P1

**Depends on:** REQ-002

**Source:** ISSUE-61

---

### REQ-004: Consistent Interview Pattern

As a maintainer, I want all interview skills to follow the same context-aware pattern, so that the system has predictable behavior across all phases.

**Acceptance Criteria:**
- [ ] Pattern documented in shared producer template or interview guidelines
- [ ] pm-interview-producer implements pattern
- [ ] architect-interview-producer implements pattern
- [ ] design-interview-producer implements pattern
- [ ] Pattern includes: context gathering, gap assessment, adaptive questioning
- [ ] Pattern defines error handling for context gathering failures
- [ ] Pattern defines resolution process for contradictory context
- [ ] Pattern allows skill-specific adaptations where justified and documented

**Priority:** P1

**Depends on:** REQ-002, REQ-003

**Source:** ISSUE-61

---

## ISSUE-53: Universal QA Skill

Requirements for replacing 13 phase-specific QA skills with one universal QA skill that validates producers against their SKILL.md contracts.

---

### REQ-005: Universal QA Skill

As a maintainer, I want one universal `/qa` skill that validates any producer against its SKILL.md contract, so that QA logic is centralized and drift-free.

**Acceptance Criteria:**
- [ ] Single `skills/qa/SKILL.md` replaces all 13 phase-specific QA skills
- [ ] QA skill receives producer's SKILL.md as context input
- [ ] QA skill receives producer's output (what it claims it did)
- [ ] QA skill receives the produced artifacts (what actually exists)
- [ ] QA validates: does reality match the contract?
- [ ] QA uses Haiku model (fast, cheap, capable enough)
- [ ] QA supports all existing message types: `approved`, `improvement-request`, `escalate-phase`, `escalate-user`
- [ ] When output is malformed (invalid format, missing required fields), QA sends `improvement-request` with parse error details
- [ ] When artifacts are missing (file not found), QA sends `improvement-request` listing missing files
- [ ] When producer SKILL.md is missing or unreadable, QA sends `error` (cannot validate without contract)

**Priority:** P0

**Depends on:** REQ-006

**Traces to:** ISSUE-53

---

### REQ-006: Contract Standard Definition

As a maintainer, I want a standard contract format for producer SKILL.md files, so that QA can programmatically extract validation criteria.

**Acceptance Criteria:**
- [ ] Standard documented in `skills/shared/CONTRACT.md`
- [ ] Format uses YAML code blocks within a "Contract" markdown section
- [ ] Contract includes: requirements table with ID, description, severity (error/warning)
- [ ] Contract includes: expected outputs (artifact paths, ID formats)
- [ ] Contract includes: required traces (what upstream artifacts must be referenced)
- [ ] Contract format is extensible for future needs
- [ ] When prose requirements don't fit the format, update CONTRACT.md to accommodate (format evolves with needs)
- [ ] CONTRACT.md includes version field; QA logs warning if producer uses older version but continues validation

**Priority:** P0

**Depends on:** None

**Traces to:** ISSUE-53

---

### REQ-007: Producer Skill Contract Sections

As a maintainer, I want all producer skills to have contract sections in their SKILL.md, so that QA can validate them consistently.

**Acceptance Criteria:**
- [ ] All producer skills updated to include Contract section per REQ-006 format
- [ ] Contract sections capture everything the producer is responsible for
- [ ] Existing prose requirements converted to structured contract format
- [ ] Producer skills affected: pm-interview-producer, pm-infer-producer, design-interview-producer, design-infer-producer, arch-interview-producer, arch-infer-producer, breakdown-producer, tdd-red-producer, tdd-green-producer, tdd-refactor-producer, doc-producer, alignment-producer, retro-producer, summary-producer

**Priority:** P0

**Depends on:** REQ-006

**Traces to:** ISSUE-53

---

### REQ-008: Gap Analysis Before QA Deletion

As a maintainer, I want gap analysis performed before deleting QA skills, so that no validation logic is lost.

**Acceptance Criteria:**
- [ ] For each QA skill, compare its checklist against corresponding producer's contract
- [ ] If QA checks something the producer doesn't document, flag as gap
- [ ] Gaps require user decision: add to producer contract OR explicitly drop
- [ ] Do NOT port QA checks blindly - each gap requires explicit confirmation
- [ ] Gap analysis documented before any QA skill deletion
- [ ] QA skills (13 total): pm-qa, design-qa, arch-qa, breakdown-qa, tdd-qa, tdd-red-qa, tdd-green-qa, tdd-refactor-qa, doc-qa, context-qa, alignment-qa, retro-qa, summary-qa

**Priority:** P0

**Depends on:** REQ-007

**Traces to:** ISSUE-53

---

### REQ-009: QA Skill Deletion

As a maintainer, I want the 13 phase-specific QA skills deleted after migration, so that there's a single source of truth for QA.

**Acceptance Criteria:**
- [ ] All 13 QA skills deleted from `skills/` directory
- [ ] Deletion only after: (1) gap analysis complete (REQ-008), (2) producer contracts complete (REQ-007), (3) universal QA skill functional (REQ-005)
- [ ] Hard cutover - no parallel operation period
- [ ] QA-TEMPLATE.md updated or removed as appropriate
- [ ] Any references to deleted skills updated (orchestrator, documentation)

**Priority:** P1

**Depends on:** REQ-005, REQ-007, REQ-008

**Traces to:** ISSUE-53

---

### REQ-010: Orchestrator Updates for Universal QA

As a maintainer, I want the orchestrator to dispatch the universal QA skill correctly, so that it receives the right context.

**Acceptance Criteria:**
- [ ] Orchestrator passes producer's SKILL.md path to QA
- [ ] Orchestrator passes producer's output to QA
- [ ] Orchestrator passes artifact paths to QA
- [ ] Orchestrator uses single `/qa` skill for all phases (no phase-specific dispatch)
- [ ] QA context format documented

**Priority:** P0

**Depends on:** REQ-005

**Traces to:** ISSUE-53

---

### REQ-011: Contract-Based Fallback Heuristics

As a maintainer, I want QA to fall back to heuristics when a producer lacks a formal contract, so that validation still works during migration.

**Acceptance Criteria:**
- [ ] If producer SKILL.md has no Contract section, QA reads entire SKILL.md
- [ ] QA extracts implicit requirements from prose (best effort)
- [ ] QA logs warning that producer should add contract section
- [ ] Fallback is transitional - all producers should eventually have contracts
- [ ] Contract format is the norm; prose fallback is the exception

**Priority:** P1

**Depends on:** REQ-005, REQ-006

**Traces to:** ISSUE-53

---

## ISSUE-56: Warn When Specs Exceed User Requests

Requirements for adding explicit warnings when producer skills infer specifications beyond what the user explicitly requested.

---

### REQ-012: Inferred Specification Message Type

As a user, I want producer skills to send a distinct message with an `inferred` flag when they add specifications not explicitly requested, so that the orchestrator pauses and I can accept or reject each inferred item before it becomes part of the artifact.

**Acceptance Criteria:**
- [ ] New message type: inferred specification message with `inferred = true` field
- [ ] Message includes: the inferred specification text, the reasoning for inference (context, edge case, or best practice), and the source that triggered the inference
- [ ] Orchestrator pauses on `inferred` messages and presents them to the user for accept/reject
- [ ] User response is recorded: accepted inferred items proceed into the artifact, rejected items are dropped
- [ ] Message format documented in producer communication patterns

**Priority:** P0

**Depends on:** None

**Traces to:** ISSUE-56

---

### REQ-013: Producer Skills Flag Inferred Specifications

As a user, I want all producer skills (both interview and infer variants) to identify when they are adding specifications beyond what was explicitly requested, so that nothing is silently added to my project artifacts.

**Acceptance Criteria:**
- [ ] pm-interview-producer uses AskUserQuestion with `inferred` flag for requirements not directly stated by the user or issue
- [ ] pm-infer-producer uses AskUserQuestion with `inferred` flag for requirements not directly present in analyzed code/docs
- [ ] design-interview-producer uses AskUserQuestion with `inferred` flag for design decisions not directly requested
- [ ] design-infer-producer uses AskUserQuestion with `inferred` flag for design decisions not directly present in analyzed code/docs
- [ ] arch-interview-producer uses AskUserQuestion with `inferred` flag for architecture decisions not directly requested
- [ ] arch-infer-producer uses AskUserQuestion with `inferred` flag for architecture decisions not directly present in analyzed code/docs
- [ ] Each inferred question includes reasoning: what triggered the inference (edge case, best practice, implicit need)
- [ ] Producers distinguish between "user said this" and "I think this is needed" for every specification item

**Priority:** P0

**Depends on:** REQ-012

**Traces to:** ISSUE-56

---

### REQ-014: Orchestrator Handles Inferred Specification Messages

As a user, I want the orchestrator to present inferred specifications for my approval before they are included in artifacts, so that I have explicit control over scope.

**Acceptance Criteria:**
- [ ] Orchestrator detects `inferred = true` messages from producers
- [ ] Orchestrator presents inferred item to user with the reasoning and asks for accept/reject
- [ ] Accepted items: orchestrator sends acceptance to producer
- [ ] Rejected items: orchestrator sends rejection to producer, producer drops the item from the artifact
- [ ] Multiple inferred items can be batched into a single user prompt (not one-by-one)
- [ ] User response is logged for traceability

**Priority:** P0

**Depends on:** REQ-012

**Traces to:** ISSUE-56

---

### REQ-015: Shared Producer Guidelines for Inference Detection

As a maintainer, I want shared documentation that defines how producers distinguish explicit from inferred specifications, so that the behavior is consistent across all producer skills.

**Acceptance Criteria:**
- [ ] Guidelines documented in shared producer template or dedicated shared doc
- [ ] Guidelines define "explicit": directly stated in user input, issue description, or gathered context
- [ ] Guidelines define "inferred": added by the producer based on best practices, edge cases, implicit needs, or professional judgment
- [ ] Guidelines provide examples of each category
- [ ] All 6 affected producer skills reference the shared guidelines

**Priority:** P1

**Depends on:** REQ-012, REQ-013

**Traces to:** ISSUE-56

---

## ISSUE-104: Orchestrator as Haiku Teammate

Requirements for splitting the orchestrator into a team lead (opus) and orchestrator teammate (haiku) to reduce cost by using the cheapest model for mechanical step loop work.

---

### REQ-016: Split Orchestrator into Team Lead and Teammate Roles

As a user, I want the orchestrator to run as a haiku teammate instead of in the main opus conversation, so that the expensive opus model is preserved for user interaction and high-level decisions.

**Acceptance Criteria:**
- [ ] Team lead (opus) owns team creation, teammate spawning, and shutdown coordination
- [ ] Orchestrator teammate (haiku) runs the `projctl step next` → dispatch → `projctl step complete` loop
- [ ] Team lead spawns orchestrator teammate on `/project` invocation
- [ ] Team lead does NOT edit files or produce artifacts directly (delegates to teammates)
- [ ] Orchestrator teammate does NOT spawn other teammates directly (sends spawn requests to team lead)
- [ ] SKILL.md documents the two-role split with clear responsibilities

**Priority:** P0

**Depends on:** None

**Traces to:** ISSUE-104

---

### REQ-017: Orchestrator Spawn Request Protocol

As a developer, I want the orchestrator teammate to send spawn requests to the team lead with complete task_params, so that the team lead can spawn teammates on the orchestrator's behalf.

**Acceptance Criteria:**
- [ ] When `projctl step next` returns `spawn-producer` or `spawn-qa`, orchestrator sends message to team lead
- [ ] Message includes full `task_params` from step next output (subagent_type, name, model, prompt)
- [ ] Message includes expected_model for handshake validation
- [ ] Team lead receives spawn request and calls Task tool with provided task_params
- [ ] Team lead performs model handshake validation after spawning
- [ ] Team lead sends confirmation message to orchestrator after successful spawn and handshake
- [ ] Team lead sends failure message to orchestrator if handshake fails

**Priority:** P0

**Depends on:** REQ-016

**Traces to:** ISSUE-104

---

### REQ-018: Orchestrator Shutdown Request Protocol

As a developer, I want the orchestrator teammate to send shutdown requests to the team lead when work is complete, so that the team lead handles graceful teammate termination.

**Acceptance Criteria:**
- [ ] When `projctl step next` returns `all-complete`, orchestrator sends message to team lead
- [ ] Message indicates project is complete and orchestrator is ready to shut down
- [ ] Team lead receives completion message and initiates end-of-command sequence
- [ ] Team lead sends shutdown_request to all active teammates (including orchestrator)
- [ ] Team lead calls TeamDelete after all teammates confirm shutdown
- [ ] Team lead reports completion status to user

**Priority:** P0

**Depends on:** REQ-016

**Traces to:** ISSUE-104

---

### REQ-019: Orchestrator Error Handling with Auto-Retry

As a developer, I want the orchestrator to automatically retry failed operations with backoff before escalating, so that transient errors don't require manual intervention.

**Acceptance Criteria:**
- [ ] Orchestrator detects errors from `projctl step next` or `projctl step complete`
- [ ] Orchestrator retries failed operations up to 3 times with exponential backoff
- [ ] Backoff delays: 1s, 2s, 4s between retries
- [ ] After 3 failed attempts, orchestrator sends error message to team lead with details
- [ ] Team lead escalates to user with error context and requests guidance
- [ ] Orchestrator logs retry attempts for debugging

**Priority:** P1

**Depends on:** REQ-016

**Traces to:** ISSUE-104

---

### REQ-020: Orchestrator Resumption Support

As a user, I want the orchestrator teammate to be resumable if it gets terminated mid-session, so that work can continue from the last saved state without restarting the entire project.

**Acceptance Criteria:**
- [ ] Project state is persisted via `projctl state set` commands during orchestrator execution
- [ ] Team lead can respawn orchestrator teammate after termination
- [ ] Respawned orchestrator calls `projctl step next` and resumes from saved state
- [ ] State includes: current phase, sub-phase, pair loop iteration, active tasks
- [ ] Orchestrator does not repeat completed work after resumption
- [ ] SKILL.md documents resumption flow and state dependencies

**Priority:** P1

**Depends on:** REQ-016

**Traces to:** ISSUE-104

---

### REQ-021: Team Lead Spawn Confirmation Flow

As a developer, I want the team lead to confirm successful spawns back to the orchestrator, so that the orchestrator knows when teammates are ready to receive work.

**Acceptance Criteria:**
- [ ] Team lead spawns teammate using task_params from orchestrator's spawn request
- [ ] Team lead validates model handshake (expected_model matches teammate's first message)
- [ ] On successful handshake, team lead sends confirmation message to orchestrator with teammate name
- [ ] Orchestrator receives confirmation and proceeds with step completion
- [ ] On handshake failure, team lead sends error message to orchestrator
- [ ] Orchestrator handles spawn failures per REQ-019 (retry with backoff)

**Priority:** P0

**Depends on:** REQ-016, REQ-017

**Traces to:** ISSUE-104

---

### REQ-022: State Persistence for Resumption

As a developer, I want project state to be saved after each significant step, so that resumption can pick up from the last completed action.

**Acceptance Criteria:**
- [ ] Orchestrator calls `projctl state set` after each `projctl step complete`
- [ ] State file persists: current phase, sub-phase, workflow type, active issue
- [ ] State file persists: pair loop iteration counts and QA feedback
- [ ] State file persists: task completion status and dependencies
- [ ] State file is atomic (writes complete or fail, no partial states)
- [ ] Respawned orchestrator reads state before first `projctl step next` call

**Priority:** P1

**Depends on:** REQ-016, REQ-020

**Traces to:** ISSUE-104

---

### REQ-023: Retry Limit to Prevent Infinite Loops

As a developer, I want retry attempts capped at a maximum limit, so that the orchestrator doesn't loop infinitely on persistent errors.

**Acceptance Criteria:**
- [ ] Maximum 3 retry attempts for any failed operation
- [ ] Retry counter resets on successful operation
- [ ] After reaching retry limit, orchestrator escalates to team lead
- [ ] Team lead escalates to user with error details and retry history
- [ ] Retry logic applies to: `projctl step next`, `projctl step complete`, spawn requests

**Priority:** P1

**Depends on:** REQ-016, REQ-019

**Traces to:** ISSUE-104

---
