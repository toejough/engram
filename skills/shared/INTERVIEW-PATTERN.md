# Interview Pattern - Adaptive Depth

Standardized pattern for interview skills that adapts question depth based on available context.

**Goal:** Ask only what's necessary. Don't waste user time on questions answerable from docs, code, or memory.

## Why This Pattern?

**Avoid redundant questions:** Small issues with clear context don't need 10-question interviews. Check docs, code, and memory first.

**Adapt to complexity:** Complex issues with sparse context need thorough discovery. Simple issues need only confirmation.

**Respect user time:** Be informed before interviewing. Leverage existing infrastructure (projctl territory, projctl memory).

**Traceability:** Quantified gap assessment makes depth decisions debuggable and improvable over time.

## Overview

```
GATHER → ASSESS → INTERVIEW → SYNTHESIZE → PRODUCE
```

**Phase Summary:**
- **GATHER**: Collect existing context (territory map, memory queries, file reads) before asking anything
- **ASSESS**: Calculate coverage percentage, classify gap size (small/medium/large)
- **INTERVIEW**: Ask questions matching gap size (1-2 / 3-5 / 6+ questions)
- **SYNTHESIZE**: Combine context and responses, check for conflicts
- **PRODUCE**: Generate artifact with traceability and gap metadata

---

## Gap Assessment Formula

**Coverage Calculation:**
```
base_coverage = (questions_answered / total_key_questions) * 100

Apply priority weights for unanswered questions:
- Critical unanswered: -15% each
- Important unanswered: -10% each
- Optional unanswered: -5% each

final_coverage = base_coverage + weight_adjustments
```

**Gap Size Classification:**

| Coverage | Gap Size | Question Count |
|----------|----------|----------------|
| ≥80% | Small gap | 1-2 confirmation questions |
| 50-79% | Medium gap | 3-5 clarification questions |
| <50% | Large gap | 6+ questions, full interview |

**Edge Case:** When total coverage <20% (both issue description AND context are sparse), classify as large gap regardless of calculation.

---

## Phase Details

Each phase has clear boundaries and responsibilities:

### Phase 1: GATHER Context

**Purpose:** Collect existing information before asking user anything.

**Process:**
1. Execute `projctl territory map` to get file structure and artifact locations
2. Execute `projctl memory query` with domain-specific queries (e.g., "architecture decisions", "technology stack")
3. Use context-explorer for targeted file reads or web fetches
4. Parse results into structured data for assessment

**Context Mechanisms:**

| Mechanism | Tool | Purpose |
|-----------|------|---------|
| Territory map | `projctl territory map` | Discover available artifacts, files, and project structure |
| Memory query | `projctl memory query "<query>"` | Semantic search for domain knowledge stored in embeddings |
| Context explorer | Existing skill | Read specific files, fetch web resources, explore semantically |

**Error Handling:**
- Territory map failure → Yield `blocked` with diagnostic info (infrastructure problem)
- Memory query timeout → Continue with available context, note limitation in yield metadata
- Contradictory context → Yield `need-decision` with conflicts for user resolution

**Exit Criteria:** Enough context gathered to proceed to ASSESS, or yield `blocked` if critical infrastructure fails.

---

### Phase 2: ASSESS Gaps

**Purpose:** Determine how much is known vs. unknown by comparing gathered context against key questions.

**Process:**
1. For each key question in registry, determine if answerable from gathered context
2. Calculate coverage using weighted formula (see Gap Assessment Formula above)
3. Classify gap size and determine question count
4. Log assessment results for observability

**Exit Criteria:** Gap size determined, ready to proceed to INTERVIEW.

---

### Phase 3: INTERVIEW (Adaptive Depth)

**Purpose:** Ask user questions to fill remaining gaps, with question count matching gap size (see Gap Assessment Formula above).

**Question Construction:**
- Reference gathered context where relevant: "I see X in docs, confirm Y?"
- Skip topics fully answered by context
- Prioritize by question priority (critical > important > optional)
- Match question count to gap size classification

**Yield Type:** `need-user-input` with questions array sized according to gap assessment.

**Exit Criteria:** User responses received, ready to proceed to SYNTHESIZE.

---

### Phase 4: SYNTHESIZE

**Purpose:** Aggregate gathered context and user responses into structured knowledge.

**Process:**
1. Combine context from GATHER with answers from INTERVIEW
2. Identify key decisions and their rationale
3. Check for conflicts with existing artifacts
4. Structure findings for output format

**Error Handling:**
- Conflicts detected → Yield `need-decision` with options
- Information still insufficient → Yield `blocked` with gap analysis

**Exit Criteria:** Synthesized information is complete and consistent.

---

### Phase 5: PRODUCE

**Purpose:** Generate artifact with proper IDs and traceability.

**Process:**
1. Create artifact with domain-specific ID format (REQ-N, DES-N, ARCH-N, TASK-N)
2. Include `**Traces to:**` links to upstream artifacts
3. Enrich yield context with gap analysis metadata
4. Write artifact to configured path
5. Yield `complete` with artifact details

**Exit Criteria:** Artifact written, yield metadata includes gap analysis for observability.

---

## Yield Context Enrichment

All interview yields include gap analysis metadata for observability and debugging.

**Example:**

```toml
[context.gap_analysis]
total_key_questions = 10
questions_answered = 7
coverage_percent = 70
gap_size = "medium"
question_count = 4
sources = ["territory", "memory", "context-files"]
unanswered_critical = ["Technology Stack", "Scale Requirements"]
```

**Key Fields:**
- `total_key_questions`, `questions_answered`, `coverage_percent` - Input/output of gap assessment formula
- `gap_size` - Classification result ("small", "medium", "large")
- `question_count` - Number of questions asked
- `sources` - Which mechanisms provided context
- `unanswered_critical` - Critical questions still unanswered (for debugging)

**Purpose:** Makes depth decisions traceable, enables optimization, supports debugging.

---

## Error Handling Patterns

| Scenario | Yield Type | Reason |
|----------|------------|--------|
| Territory map fails | `blocked` | Infrastructure problem, cannot proceed safely |
| Memory query times out | Continue with note | Degraded but functional, document limitation in metadata |
| Contradictory context found | `need-decision` | User must resolve conflicts before proceeding |
| Partial context available | Continue | Best-effort with available information, note limitation in yield |
| Critical questions unanswered after INTERVIEW | `blocked` | Cannot produce quality artifact without critical info |

**When to Yield `blocked`:**
- Critical infrastructure failures (territory map, file system access)
- Critical questions remain unanswered after user interview
- Fundamental blockers preventing artifact generation

**When to Yield `need-decision`:**
- Multiple valid architectural approaches
- Contradictory information from different sources
- Ambiguous requirements that impact scope significantly

**When to Continue with Partial Context:**
- Optional questions unanswered
- Memory query timeout (territory still provides value)
- Non-critical information missing but sufficient to produce minimum viable artifact

---

## Key Questions Registry

Each interview skill defines 8-12 key questions representing minimum viable context for their phase.

**Example Format:**

```markdown
## Key Questions

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

| Priority | Count | When Needed | Weight if Unanswered |
|----------|-------|-------------|---------------------|
| Critical | 2-4 | Must answer for ANY project | -15% each |
| Important | 3-5 | Usually needed, occasionally skippable | -10% each |
| Optional | 2-3 | Nice-to-have, often inferable | -5% each |

Each interview skill (pm-interview-producer, arch-interview-producer, design-interview-producer) defines domain-specific questions matching its phase responsibilities.

---

## Implementation Checklist

When adding adaptive interview to a skill:

- [ ] Define key questions registry (8-12 questions with priorities)
- [ ] Implement GATHER phase with territory map, memory query, context-explorer
- [ ] Implement ASSESS phase with coverage calculation
- [ ] Implement INTERVIEW phase with adaptive depth (1-2 / 3-5 / 6+ questions)
- [ ] Enrich yield context with gap analysis metadata
- [ ] Add error handling for territory failures, memory timeouts, contradictory context
- [ ] Document skill-specific adaptations if pattern is modified
- [ ] Add integration tests with mocked sparse/medium/rich context scenarios

