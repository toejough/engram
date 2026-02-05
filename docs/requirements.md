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
- [ ] Interview skills (pm-interview-producer, architect-interview-producer, design-interview-producer) gather context BEFORE yielding `need-user-input`
- [ ] Skills use existing infrastructure: territory map, memory query, context-explorer
- [ ] Skills assess what's already answered by issue + gathered context
- [ ] Skills only ask questions about genuinely missing information
- [ ] When context gathering fails (territory map error, memory query timeout), skill yields `blocked` with diagnostic information
- [ ] When gathered context contains contradictory information, skill yields `need-decision` with conflicting statements for user resolution

**Priority:** P0

**Depends on:** None

**Source:** ISSUE-061

---

### REQ-003: Adaptive Interview Depth

As a user, I want interview depth to adapt based on information gaps, so that simple issues get quick confirmation while complex issues get thorough exploration.

**Acceptance Criteria:**
- [ ] Skills assess information gaps against their key responsibilities (as documented in each skill's SKILL.md)
- [ ] Gap size determined by percentage of key questions answerable from context: ≥80% = small gap, 50-79% = medium gap, <50% = large gap
- [ ] Small gaps: skill yields 1-2 confirmation questions maximum
- [ ] Medium gaps: skill yields 3-5 clarification questions
- [ ] Large gaps: skill yields full interview sequence (6+ questions covering all phases)
- [ ] Depth decision is explicit and traceable in yield context (includes gap percentage and question count)
- [ ] When both issue description AND gathered context are sparse (<20% coverage), skill yields full interview by default

**Priority:** P1

**Depends on:** REQ-002

**Source:** ISSUE-061

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

**Source:** ISSUE-061

---

## ISSUE-053: Universal QA Skill

Requirements for replacing 13 phase-specific QA skills with one universal QA skill that validates producers against their SKILL.md contracts.

---

### REQ-005: Universal QA Skill

As a maintainer, I want one universal `/qa` skill that validates any producer against its SKILL.md contract, so that QA logic is centralized and drift-free.

**Acceptance Criteria:**
- [ ] Single `skills/qa/SKILL.md` replaces all 13 phase-specific QA skills
- [ ] QA skill receives producer's SKILL.md as context input
- [ ] QA skill receives producer's yield (what it claims it did)
- [ ] QA skill receives the produced artifacts (what actually exists)
- [ ] QA validates: does reality match the contract?
- [ ] QA uses Haiku model (fast, cheap, capable enough)
- [ ] QA supports all existing yield types: `approved`, `improvement-request`, `escalate-phase`, `escalate-user`
- [ ] When yield is malformed (invalid TOML, missing required fields), QA yields `improvement-request` with parse error details
- [ ] When artifacts are missing (file not found), QA yields `improvement-request` listing missing files
- [ ] When producer SKILL.md is missing or unreadable, QA yields `error` (cannot validate without contract)

**Priority:** P0

**Depends on:** REQ-006

**Traces to:** ISSUE-053

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

**Traces to:** ISSUE-053

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

**Traces to:** ISSUE-053

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

**Traces to:** ISSUE-053

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

**Traces to:** ISSUE-053

---

### REQ-010: Orchestrator Updates for Universal QA

As a maintainer, I want the orchestrator to dispatch the universal QA skill correctly, so that it receives the right context.

**Acceptance Criteria:**
- [ ] Orchestrator passes producer's SKILL.md path to QA
- [ ] Orchestrator passes producer's yield file to QA
- [ ] Orchestrator passes artifact paths to QA
- [ ] Orchestrator uses single `/qa` skill for all phases (no phase-specific dispatch)
- [ ] QA context file format documented

**Priority:** P0

**Depends on:** REQ-005

**Traces to:** ISSUE-053

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

**Traces to:** ISSUE-053
