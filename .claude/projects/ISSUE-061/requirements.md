# ISSUE-061 Requirements: Adaptive Interview Depth

**Issue:** ISSUE-061 - Decision needed: What's the right PM interview depth?

**Global Requirements Updated:** [docs/requirements.md](../../../docs/requirements.md)

## Requirements Created

### REQ-002: Context-Aware Interview Skills (P0)

As a user, I want interview skills to understand existing context before asking questions, so that I'm not asked about information that's already available.

**Acceptance Criteria:**
- [ ] Interview skills (pm-interview-producer, architect-interview-producer, design-interview-producer) gather context BEFORE yielding `need-user-input`
- [ ] Skills use existing infrastructure: territory map, memory query, context-explorer
- [ ] Skills assess what's already answered by issue + gathered context
- [ ] Skills only ask questions about genuinely missing information
- [ ] When context gathering fails (territory map error, memory query timeout), skill yields `blocked` with diagnostic information
- [ ] When gathered context contains contradictory information, skill yields `need-decision` with conflicting statements for user resolution

**Source:** ISSUE-061

---

### REQ-003: Adaptive Interview Depth (P1)

As a user, I want interview depth to adapt based on information gaps, so that simple issues get quick confirmation while complex issues get thorough exploration.

**Acceptance Criteria:**
- [ ] Skills assess information gaps against their key responsibilities (as documented in each skill's SKILL.md)
- [ ] Gap size determined by percentage of key questions answerable from context: ≥80% = small gap, 50-79% = medium gap, <50% = large gap
- [ ] Small gaps: skill yields 1-2 confirmation questions maximum
- [ ] Medium gaps: skill yields 3-5 clarification questions
- [ ] Large gaps: skill yields full interview sequence (6+ questions covering all phases)
- [ ] Depth decision is explicit and traceable in yield context (includes gap percentage and question count)
- [ ] When both issue description AND gathered context are sparse (<20% coverage), skill yields full interview by default

**Depends on:** REQ-002

**Source:** ISSUE-061

---

### REQ-004: Consistent Interview Pattern (P1)

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

**Depends on:** REQ-002, REQ-003

**Source:** ISSUE-061
