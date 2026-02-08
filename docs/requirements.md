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
- [ ] QA validates every CHECK-N in the producer's contract YAML
- [ ] QA reports findings with severity (error/warning)
- [ ] QA yields: approved, improvement-request, or escalate-phase
- [ ] QA never duplicates contract checks — contract is single source of truth
- [ ] QA handles missing/corrupted artifacts gracefully
- [ ] Each QA finding references the specific CHECK-N from the producer contract

**Priority:** P0

**Source:** ISSUE-53

---

## ISSUE-152: Integrate Semantic Memory into Orchestration Workflow

Requirements for integrating the existing `projctl memory` package (ONNX-based semantic similarity search) into the orchestration workflow and skills.

---

### REQ-006: Accurate Semantic Embeddings

As a user, I want semantic similarity search to return relevant results, so that memory queries surface related learnings from past projects.

**Acceptance Criteria:**
- [ ] Embeddings use proper BERT WordPiece tokenization (not hash-based)
- [ ] Model uses actual e5-small-v2 (not all-MiniLM-L6-v2)
- [ ] Query text prefixed with "query: " before tokenization
- [ ] Indexed content (learnings, decisions, sessions) prefixed with "passage: " before tokenization
- [ ] Vocabulary loaded from e5-small-v2/vocab.txt (30522 tokens)
- [ ] Token IDs wrapped with [CLS] ... [SEP] special tokens
- [ ] TestIntegration_SemanticSimilarityExampleErrorAndException passes
- [ ] TestIntegration_SemanticSimilarityRanksRelatedHigher passes
- [ ] Custom tokenizer is tested with property-based tests (rapid)

**Priority:** P0 (BLOCKING — all other memory tasks depend on this)

**Traces to:** ISSUE-152, TASK-1

**Source:** ISSUE-152 plan, TASK-1

---

### REQ-007: Automatic Session-End Capture

As a user, I want project learnings captured automatically at completion, so that future projects benefit from past experience.

**Acceptance Criteria:**
- [ ] Orchestrator runs `projctl memory session-end` at project completion
- [ ] Session-end runs BEFORE integrate/trace commands
- [ ] Session-end reads today's decisions from `~/.claude/memory/decisions/{DATE}-{PROJECT}.jsonl`
- [ ] Session-end generates markdown summary (max 2000 chars)
- [ ] Session-end writes to `~/.claude/memory/sessions/{DATE}-{PROJECT}.md`
- [ ] Session summary is auto-indexed on next memory query
- [ ] Session-end receives project/issue ID from orchestrator context

**Priority:** P1

**Traces to:** ISSUE-152, TASK-2

**Source:** ISSUE-152 plan, TASK-2

---

### REQ-008: Producer Memory Reads in GATHER Phase

As a producer, I want to query past learnings during GATHER, so that I avoid repeating decisions and mistakes from prior projects.

**Acceptance Criteria:**
- [ ] ALL 18 LLM-driven skills query memory during GATHER phase
- [ ] Interview producers (PM, Design, Arch) query past requirements/design/architecture decisions
- [ ] Infer producers (PM, Design, Arch) query past requirements/design/architecture decisions
- [ ] TDD producers (red, green, refactor) query test patterns, implementation patterns, refactoring patterns
- [ ] Breakdown producer queries task decomposition patterns
- [ ] Alignment producer queries common alignment errors and domain boundary violations
- [ ] Doc/Summary/Next-Steps producers query documentation conventions and summary patterns
- [ ] QA queries known failure patterns as verification backstop
- [ ] Retro queries past retrospective patterns and recurring challenges
- [ ] Context-explorer auto-enriches queries with memory when no explicit memory query provided
- [ ] All memory queries are non-blocking (graceful degradation if memory unavailable)
- [ ] Each skill queries for "known failures in <artifact-type> validation" to proactively avoid past QA failures

**Priority:** P1

**Traces to:** ISSUE-152, TASK-3, TASK-4, TASK-5, TASK-7, TASK-8, TASK-9, TASK-11, TASK-12, TASK-13, TASK-15, TASK-16, TASK-17, TASK-18

**Source:** ISSUE-152 plan, TASKs 3-9, 11-18

---

### REQ-009: Universal Yield Capture

As a maintainer, I want producer decisions captured automatically, so that all producers contribute to memory without per-producer wiring.

**Acceptance Criteria:**
- [ ] Orchestrator runs `projctl memory extract` after every spawn-producer completion
- [ ] Extract reads result.toml from producer yield
- [ ] Extract captures [[decisions]] and [[learnings]] from yield TOML
- [ ] Extracted decisions include context, choice, reason, alternatives
- [ ] Extract is best-effort (log warning and continue on failure)
- [ ] Extract applies to ALL producers universally (single integration point)
- [ ] New producers get memory capture for free (no per-producer SKILL.md changes needed)

**Priority:** P1

**Traces to:** ISSUE-152, TASK-14

**Source:** ISSUE-152 plan, TASK-14

---

### REQ-010: QA Failure Persistence

As a QA skill, I want to persist failure findings to memory, so that future projects can avoid known pitfalls.

**Acceptance Criteria:**
- [ ] QA runs `projctl memory learn` when reporting improvement-request or escalate-phase
- [ ] Each error-severity finding is persisted separately
- [ ] Persisted message format: "QA failure in <artifact-type>: <check-id> - <failure description>"
- [ ] Persistence includes project/issue ID tagging
- [ ] Approved verdicts do NOT persist (only failures are worth learning from)
- [ ] Future QA reads can surface these failure patterns during LOAD phase

**Priority:** P1

**Traces to:** ISSUE-152, TASK-6

**Source:** ISSUE-152 plan, TASK-6

---

### REQ-011: Retrospective Learning Capture

As a retro-producer, I want to persist key learnings to memory, so that future retrospectives can identify recurring patterns.

**Acceptance Criteria:**
- [ ] Retro-producer reads past retrospective patterns during GATHER
- [ ] Retro-producer queries "retrospective challenges" and "process improvement recommendations"
- [ ] Retro-producer persists successes via `projctl memory learn`
- [ ] Retro-producer persists challenges via `projctl memory learn`
- [ ] Retro-producer persists High/Medium recommendations via `projctl memory learn`
- [ ] All persisted learnings include project/issue ID tagging
- [ ] Learnings are queryable by future project phases

**Priority:** P1

**Traces to:** ISSUE-152, TASK-7

**Source:** ISSUE-152 plan, TASK-7

---

### REQ-012: Orchestrator Startup Memory Context

As an orchestrator, I want to query past learnings at startup, so that every spawned producer has awareness of past experience.

**Acceptance Criteria:**
- [ ] Orchestrator queries memory during startup (before entering step loop)
- [ ] Orchestrator queries "lessons from past projects"
- [ ] Orchestrator queries "common challenges in <workflow-type> projects"
- [ ] Query results are included in orchestrator's working context
- [ ] Queries surface session summaries (REQ-007 writes)
- [ ] Queries surface retro learnings (REQ-011 writes)
- [ ] Queries surface QA failure patterns (REQ-010 writes)
- [ ] Graceful degradation if memory unavailable

**Priority:** P1

**Traces to:** ISSUE-152, TASK-8

**Source:** ISSUE-152 plan, TASK-8

---

### REQ-013: Memory Promotion Pipeline

As a user, I want high-value memories promoted to permanent knowledge, so that consistently useful patterns become part of CLAUDE.md.

**Acceptance Criteria:**
- [ ] Embeddings DB tracks retrieval_count per entry (incremented on query result)
- [ ] Embeddings DB tracks last_retrieved timestamp
- [ ] Embeddings DB tracks projects_retrieved (deduplicated list)
- [ ] `projctl memory promote` command lists promotion candidates
- [ ] Candidates: retrieval_count >= 3 AND unique projects >= 2
- [ ] Promotion query returns content, retrieval count, project count, last retrieved timestamp
- [ ] Retro-producer checks for promotion candidates during GATHER
- [ ] Retro includes promotion recommendations: "Consider promoting to CLAUDE.md: <content>"
- [ ] Promotion is recommendation only (human approves before CLAUDE.md edit)

**Priority:** P2

**Traces to:** ISSUE-152, TASK-10

**Source:** ISSUE-152 plan, TASK-10

---

### REQ-014: External Knowledge Capture

As a skill, I want to capture relevant external best practices, so that projects benefit from current community knowledge.

**Acceptance Criteria:**
- [ ] Embeddings DB has source_type column: "internal" (default) or "external"
- [ ] `projctl memory learn` accepts --source flag
- [ ] External memories get lower initial confidence (0.7 vs 1.0 for internal)
- [ ] Arch-interview-producer searches for "<technology> architecture patterns 2026"
- [ ] Design-interview-producer searches for "<domain> UX best practices 2026"
- [ ] TDD-green-producer searches for "<technology> best practices 2026"
- [ ] High-value web findings stored via `projctl memory learn --source external`
- [ ] External knowledge capture is optional (skip if domain well-covered or web unavailable)
- [ ] Web search results include source attribution in stored message

**Priority:** P2

**Traces to:** ISSUE-152, TASK-19

**Source:** ISSUE-152 plan, TASK-19

---

### REQ-015: Memory Hygiene and Confidence Decay

As a maintainer, I want stale memories to decay and be pruned, so that memory quality doesn't degrade over time.

**Acceptance Criteria:**
- [ ] Embeddings DB has confidence column (default 1.0 for internal, 0.7 for external)
- [ ] Search ranking uses (cosine_similarity * confidence) instead of raw similarity
- [ ] `projctl memory decay` command multiplies confidence by factor (default 0.9)
- [ ] Decay only affects entries NOT retrieved since last decay
- [ ] Retrieved entries keep their confidence (retrieval = validation of usefulness)
- [ ] `projctl memory prune` command removes entries below confidence threshold (default 0.1)
- [ ] Orchestrator runs decay + prune at project end (after session-end, before integrate)
- [ ] Pruning is explicit (not automatic) — requires command invocation

**Priority:** P2

**Traces to:** ISSUE-152, TASK-20

**Source:** ISSUE-152 plan, TASK-20

---

### REQ-016: Memory Conflict Detection

As a user, I want to be warned about contradictory memories, so that I can resolve conflicts before they cause confusion.

**Acceptance Criteria:**
- [ ] `projctl memory learn` checks for high-similarity existing entries (cosine > 0.85)
- [ ] When high-similarity entries exist, return conflict info (existing content, source, similarity)
- [ ] Learning still persists (conflicts are warnings, not blockers)
- [ ] CLI outputs conflict warnings: "⚠ Potential conflict with existing memory: ..."
- [ ] Universal yield capture logs conflicts but continues (automated flow shouldn't block)
- [ ] External knowledge capture flags conflicts for review

**Priority:** P2

**Traces to:** ISSUE-152, TASK-20

**Source:** ISSUE-152 plan, TASK-20

---

### REQ-017: Future Skill Memory Requirements

(Placeholder for future requirements)

---

### REQ-018: Graceful Shutdown Protocol

Team lead coordinates graceful project shutdown with end-of-command sequence and teammate notifications.

**Acceptance Criteria:**
- [ ] Orchestrator sends all-complete message when projctl step next returns all-complete
- [ ] Team lead receives all-complete and offers end-of-command options (retro, summary, issue updates)
- [ ] Team lead sends shutdown_request to all active teammates
- [ ] Team lead waits for shutdown confirmations before calling TeamDelete
- [ ] TeamDelete is called only after all teammates shut down

**Priority:** P1

**Traces to:** ARCH-044, ISSUE-104

**Source:** ISSUE-104

---

### REQ-019: Automatic Retry with Exponential Backoff

Orchestrator implements automatic retry with exponential backoff for transient errors before escalating to team lead.

**Acceptance Criteria:**
- [ ] Orchestrator retries failed step operations (step next, step complete, spawn confirmation)
- [ ] Backoff delays: 1s, 2s, 4s (max 3 attempts)
- [ ] Errors triggering retry: command failures, JSON parse errors, spawn confirmation timeout
- [ ] Errors skipped for retry: user cancellation, state file corruption, team lead shutdown
- [ ] After max retries, error escalates to team lead with action/phase/output details
- [ ] Escalation message format includes error output for user investigation

**Priority:** P1

**Traces to:** ARCH-046, ISSUE-104

**Source:** ISSUE-104

---

### REQ-020: Project Resumption After Orchestrator Termination

Team lead can respawn orchestrator after unexpected termination without losing progress.

**Acceptance Criteria:**
- [ ] Orchestrator reads project state via projctl state get on startup
- [ ] If state exists (phase != empty), orchestrator resumes from saved phase
- [ ] If state missing, orchestrator starts new project
- [ ] State persists after every projctl step complete call
- [ ] Completed phases/artifacts/commits preserved across respawn
- [ ] Team lead can respawn orchestrator with same spawn params

**Priority:** P1

**Traces to:** ARCH-045, ARCH-049, ISSUE-104

**Source:** ISSUE-104

---

### REQ-021: Team Lead Spawn Request Protocol

Orchestrator and team lead coordinate teammate spawning via structured message protocol with model validation.

**Acceptance Criteria:**
- [ ] Orchestrator sends spawn request with task_params JSON, expected_model, action, phase
- [ ] Team lead spawns teammate via Task tool using task_params
- [ ] Team lead validates model by substring matching against teammate's first message (case-insensitive)
- [ ] On model match: Team lead sends spawn-confirmed message to orchestrator
- [ ] On model mismatch: Team lead sends spawn-failed message to orchestrator (doesn't let teammate continue)
- [ ] Teammate messages orchestrator directly after completion

**Priority:** P1

**Traces to:** ARCH-043, ISSUE-104

**Source:** ISSUE-104

---

### REQ-022: State Persistence and Resumption

Orchestrator persists state after each step for crash recovery and resumption.

**Acceptance Criteria:**
- [ ] State file format: .claude/projects/<issue>/state.toml
- [ ] State includes: current phase, sub-phase, workflow type, active issue, pair loop iteration count
- [ ] State persisted after every projctl step complete call (atomic write, no partial state)
- [ ] Orchestrator reads state via projctl state get on startup
- [ ] If state exists (phase != empty), orchestrator skips init and resumes from saved phase
- [ ] If state missing, orchestrator calls projctl state init and continues normally

**Priority:** P1

**Traces to:** ARCH-045, ARCH-049, ISSUE-104

**Source:** ISSUE-104

---

### REQ-023: Transient Error Recovery

Orchestrator detects transient vs persistent errors and escalates persistent errors to team lead.

**Acceptance Criteria:**
- [ ] Transient errors (marked for retry): network timeouts, filesystem delays, spawn confirmation timeout
- [ ] Persistent errors (escalate immediately): user cancellation, state file corruption, team lead shutdown
- [ ] Error classification documented in escalation message to team lead
- [ ] Error output includes action, phase, and diagnostic details for user investigation
- [ ] Escalation to team lead allows user to provide guidance or manually fix issue

**Priority:** P1

**Traces to:** ARCH-046, ISSUE-104

**Source:** ISSUE-104

As a maintainer, I want memory integration requirements defined for future skills, so that they're built in from day one.

**Acceptance Criteria:**
- [ ] plan-producer (ISSUE-158) requirements documented
- [ ] plan-producer reads "past project plans for <project-domain>" and "known planning failures"
- [ ] plan-producer writes via universal yield capture (REQ-009)
- [ ] evaluation-producer (ISSUE-158) requirements documented
- [ ] evaluation-producer reads "past evaluation patterns" and "known evaluation failures"
- [ ] evaluation-producer writes via universal yield capture (REQ-009)

**Priority:** P2

**Traces to:** ISSUE-152, TASK-20 (Future Skills section)

**Source:** ISSUE-152 plan, "Future Skills: Memory Integration Requirements"

**Note:** Architecture deferred to ISSUE-158 when plan-producer and evaluation-producer are implemented. These requirements establish the memory integration contract for future work.

---

