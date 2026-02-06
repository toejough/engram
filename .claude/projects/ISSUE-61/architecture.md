# ISSUE-061 Architecture: Adaptive Interview Depth

**Issue:** ISSUE-061 - Decision needed: What's the right PM interview depth?

**Global Architecture Updated:** [docs/architecture.md](../../../docs/architecture.md)

## Architecture Decisions

### ARCH-001: Context-First Interview Pattern

Interview skills will gather context BEFORE yielding user questions, using existing infrastructure.

**Decision:** Pre-interview context gathering using three mechanisms:
1. **Territory map** - File structure and available artifacts (`projctl territory map`)
2. **Memory query** - Semantic search for domain-specific knowledge (`projctl memory query`)
3. **Context explorer** - File reads, web fetches, semantic exploration (existing skill)

**Rationale:**
- Avoids asking questions about information already available in docs, code, or memory
- Leverages existing tools rather than building new infrastructure
- Follows the principle of "gather before synthesize"

**Error Handling:**
- Territory map failures: Yield `blocked` with diagnostic info
- Memory query timeouts: Continue with available context, note limitation in yield
- Contradictory context: Yield `need-decision` with conflicts for user resolution

**Alternatives Considered:**
- Direct file scanning: Less semantic understanding, more brittle
- LLM-only inference: Higher cost, less precise for factual lookup
- Skip context gathering: Would continue asking redundant questions (status quo problem)

**Traces to:** REQ-002

---

### ARCH-002: Gap-Based Depth Calculation

Interview depth adapts based on percentage of key questions answerable from gathered context.

**Decision:** Three-tier depth model based on coverage:
- **Small gap** (≥80% coverage): 1-2 confirmation questions
- **Medium gap** (50-79% coverage): 3-5 clarification questions
- **Large gap** (<50% coverage): 6+ questions, full interview

**Key Questions Definition:** Each interview skill defines 8-12 "key questions" representing core responsibilities:
- PM: Problem, stakeholders, success criteria, constraints, edge cases
- Architect: Technology stack, scale requirements, integrations, deployment, non-functionals
- Design: User personas, interaction patterns, screen flow, component needs

**Coverage Calculation:**
```
coverage = (questions_answered_by_context / total_key_questions) * 100
```

**Rationale:**
- Quantifiable heuristic that's debuggable and improvable
- Balances efficiency (small gaps = fast) with thoroughness (large gaps = deep)
- Preserves domain expertise (each skill defines what matters for their phase)

**Edge Case:** When both issue description AND context are sparse (<20% total coverage), default to full interview. Better to over-gather than miss critical info.

**Alternatives Considered:**
- Fixed depth (always full): Too slow for simple issues
- Fixed depth (always minimal): Misses important context for complex issues
- User-specified depth: Adds friction, user may not know what's needed
- LLM judgment: Less predictable, harder to debug/improve

**Traces to:** REQ-003

---

### ARCH-003: Yield Context Enrichment

All yields include gap assessment metadata for observability and debugging.

**Decision:** Extend yield `[context]` section with gap analysis:
```toml
[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70
gap_size = "medium"
question_count = 4
sources = ["territory", "memory", "context-files"]
```

**Rationale:**
- Makes depth decisions traceable and auditable
- Enables future optimization (identify which sources are most valuable)
- Supports debugging when depth feels wrong

**Alternatives Considered:**
- Implicit depth: Less transparent, harder to improve
- Separate analytics file: Violates single-source-of-truth for yield state

**Traces to:** REQ-003

---

### ARCH-004: Consistent Interview Protocol

All interview producers follow standardized GATHER → ASSESS → INTERVIEW → SYNTHESIZE → PRODUCE flow.

**Decision:** Formalize shared interview pattern:

1. **GATHER context** (before any user questions)
   - Territory map for file structure
   - Memory queries for domain knowledge
   - Context-explorer for specific files/docs

2. **ASSESS gaps**
   - Compare context against key questions
   - Calculate coverage percentage
   - Determine depth tier

3. **INTERVIEW** (adaptive depth)
   - Small gap: Confirm critical decisions only
   - Medium gap: Clarify ambiguous areas
   - Large gap: Full structured interview

4. **SYNTHESIZE** (unchanged)
   - Aggregate responses
   - Structure for output

5. **PRODUCE** (unchanged)
   - Generate artifact with IDs
   - Include traceability

**Shared Implementation:**
- Document pattern in `shared/INTERVIEW-PATTERN.md`
- Each skill implements pattern with domain-specific key questions
- Pattern allows skill-specific adaptations where justified (documented in skill)

**Error Handling Pattern:**
- Context gathering failures: Yield `blocked` (infrastructure problem)
- Contradictory context: Yield `need-decision` (user resolution needed)
- Partial context: Continue with best-effort, note limitation in yield

**Rationale:**
- Predictable behavior across PM, Design, Architecture phases
- Easier to maintain (pattern fixes propagate to all skills)
- Clear extension point for new interview skills

**Alternatives Considered:**
- Skill-specific patterns: More flexibility but inconsistent UX
- Fully unified skill: Would violate phase ownership boundaries

**Traces to:** REQ-004

---

### ARCH-005: Key Questions Registry

Each interview skill defines its key questions in a structured format for automated coverage analysis.

**Decision:** Add `## Key Questions` section to each interview skill:

```markdown
## Key Questions

Questions that define minimum viable context for this phase:

1. **Technology Stack** - What languages/frameworks? (critical)
2. **Scale Requirements** - Expected users/data volume? (critical)
3. **Deployment Target** - Where will it run? (important)
4. **Integrations** - External systems to connect? (important)
5. **Performance SLA** - Response time requirements? (important)
6. **Security Model** - Authentication/authorization needs? (important)
7. **Data Durability** - Acceptable data loss? (optional)
8. **Observability** - Logging/monitoring strategy? (optional)
```

**Priority Levels:**
- `critical`: Must answer for ANY project (2-4 questions)
- `important`: Usually needed, occasionally skippable (3-5 questions)
- `optional`: Nice-to-have, often inferable (2-3 questions)

**Coverage Calculation Weights:**
- Critical unanswered: -15% coverage each
- Important unanswered: -10% coverage each
- Optional unanswered: -5% coverage each

**Rationale:**
- Makes gap assessment algorithmic and debuggable
- Priority weights reflect real importance hierarchy
- Critical questions can't be ignored even if total coverage looks high

**Alternatives Considered:**
- Unweighted questions: Treats optional same as critical
- Fixed question list: Less flexible for different project types
- Implicit priorities: Harder to maintain and improve

**Traces to:** REQ-003, REQ-004

---

### ARCH-006: Incremental Rollout Strategy

Pattern updates roll out to skills one at a time, validating before proceeding.

**Decision:** Implementation order:
1. Document shared pattern (`shared/INTERVIEW-PATTERN.md`)
2. Update `arch-interview-producer` (this issue)
3. Validate with real usage on 2-3 issues
4. Update `pm-interview-producer`
5. Validate with real usage on 2-3 issues
6. Update `design-interview-producer`
7. Final validation across all phases

**Validation Criteria:**
- Gap assessment produces sensible depth decisions
- Context gathering completes without errors
- User questions feel appropriate (not redundant, not sparse)
- Yield metadata enables debugging

**Rollback Strategy:** If validation fails, revert skill to previous version and iterate on pattern.

**Rationale:**
- Lower risk than simultaneous updates to all skills
- Real usage feedback informs improvements to later rollouts
- Each skill can adapt pattern to domain needs during implementation

**Alternatives Considered:**
- Simultaneous rollout: Faster but riskier
- Single unified skill: Would violate phase boundaries

**Traces to:** REQ-004

---

## Implementation Notes

**File Changes:**
- New: `~/.claude/skills/shared/INTERVIEW-PATTERN.md` (shared pattern)
- Modified: `~/.claude/skills/arch-interview-producer/SKILL.md` (implement pattern)
- Modified: `~/.claude/skills/arch-interview-producer/SKILL_test.sh` (test gap assessment)

**Testing:**
- Unit tests for coverage calculation
- Integration test with mocked context (sparse, medium, full)
- Validation on real issues (ISSUE-061 itself is first validation)

**Performance:**
- Context gathering adds ~2-5 seconds before first question
- Acceptable tradeoff for avoiding redundant questions
- Parallel query execution minimizes latency

**Observability:**
- All gap assessments logged in yield context
- Can analyze which sources provide most value
- Can tune key questions and weights based on real data
