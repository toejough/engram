# Requirements: Memory Tiering Policy Documentation

**Issue:** ISSUE-182: Define explicit memory tiering policy in CLAUDE.md

**Created:** 2026-02-10

**Traces to:** ISSUE-182, ISSUE-184, ISSUE-186, ISSUE-189

---

## Overview

This document defines requirements for documenting an explicit 3-tier memory policy in CLAUDE.md. The memory system uses a hierarchical tiering strategy to optimize context usage and retrieval performance.

---

## Requirements

### REQ-1: Three-Tier Memory Architecture Documentation

As a Claude Code user, I want clear documentation of the three-tier memory architecture in CLAUDE.md, so that I understand what information lives where and how it's accessed.

**Acceptance Criteria:**

- [ ] CLAUDE.md contains a dedicated section describing the 3-tier memory architecture
- [ ] Each tier is clearly labeled (Tier 1, Tier 2, Tier 3)
- [ ] The documentation appears in a logical location within CLAUDE.md structure
- [ ] The section is concise and fits within the overall <100 line budget for Tier 1 content

**Priority:** P0

**Traces to:** ISSUE-182

---

### REQ-2: Tier 1 Definition (Always-Loaded Context)

As a Claude Code user, I want to know what qualifies for Tier 1 storage, so that I can understand what context is always available.

**Acceptance Criteria:**

- [ ] Tier 1 is defined as "always loaded" content
- [ ] Size constraint is documented (<100 lines total for CLAUDE.md)
- [ ] Examples of Tier 1 content are provided (e.g., core principles, critical warnings, workflow tier selection)
- [ ] Access pattern is documented (loaded on every session start)
- [ ] Token budget is specified (part of base context)

**Priority:** P0

**Traces to:** ISSUE-182

---

### REQ-3: Tier 2 Definition (On-Demand Retrieval)

As a Claude Code user, I want to know what qualifies for Tier 2 storage, so that I can understand when content is retrieved based on relevance.

**Acceptance Criteria:**

- [ ] Tier 2 is defined as "retrieved on demand" content
- [ ] Token budget is documented (~2000 tokens per retrieval)
- [ ] Examples of Tier 2 content are provided (e.g., generated skills, synthesized patterns)
- [ ] Retrieval mechanism is described (semantic search, skill matching)
- [ ] Storage location is specified (generated skills directory, skill database)

**Priority:** P0

**Traces to:** ISSUE-182, ISSUE-186

---

### REQ-4: Tier 3 Definition (Dynamic Lookup)

As a Claude Code user, I want to know what qualifies for Tier 3 storage, so that I can understand the full memory database.

**Acceptance Criteria:**

- [ ] Tier 3 is defined as "dynamic lookup" content
- [ ] Scope is documented (all memory DB entries, full learning history)
- [ ] Access pattern is described (explicit `projctl memory query` or skill-triggered retrieval)
- [ ] Storage location is specified (~/.claude/memory/embeddings.db)
- [ ] Query mechanism is documented (hybrid BM25 + vector search)

**Priority:** P0

**Traces to:** ISSUE-182

---

### REQ-5: Movement Between Tiers

As a system maintainer, I want to understand how memories move between tiers, so that I can manage the memory lifecycle effectively.

**Acceptance Criteria:**

- [ ] Promotion criteria are documented (Tier 3 → Tier 2 → Tier 1)
- [ ] Demotion criteria are documented (Tier 1 → Tier 2 → Tier 3)
- [ ] The `projctl memory optimize` pipeline role is explained
- [ ] Manual promotion process is documented (if applicable)
- [ ] Pruning/decay thresholds are referenced (link to optimization documentation)

**Priority:** P1

**Traces to:** ISSUE-182, ISSUE-184

---

### REQ-6: Integration with Optimize Pipeline

As a developer, I want the tier documentation to reference the existing optimize pipeline, so that I understand how tiers are managed automatically.

**Acceptance Criteria:**

- [ ] Reference to `projctl memory optimize` command is included
- [ ] Key optimization steps relevant to tiering are mentioned (decay, promote, auto-demote, skill generation)
- [ ] Link to detailed optimization documentation is provided (if separate doc exists)
- [ ] Relationship between tiers and optimization phases is clear

**Priority:** P1

**Traces to:** ISSUE-182, ISSUE-184

---

### REQ-7: Skill Generation as Tier 2 Mechanism

As a Claude Code user, I want to understand how dynamically generated skills serve as Tier 2 storage, so that I can see how knowledge compilation works.

**Acceptance Criteria:**

- [ ] Generated skills are identified as Tier 2 content
- [ ] Skill generation process is briefly described (clustering, synthesis, compilation)
- [ ] Skill retrieval mechanism is documented (semantic matching during queries)
- [ ] Reference to ISSUE-186 implementation is included
- [ ] Distinction between static skills (always available) and dynamic skills (Tier 2) is clear

**Priority:** P1

**Traces to:** ISSUE-182, ISSUE-186

---

### REQ-8: Examples and Use Cases

As a Claude Code user, I want concrete examples of each tier, so that I can classify my own learnings appropriately.

**Acceptance Criteria:**

- [ ] At least 2 examples per tier are provided
- [ ] Examples show real content types from the projctl ecosystem
- [ ] Counter-examples are included (what NOT to put in each tier)
- [ ] Decision criteria are provided (how to choose the right tier)

**Priority:** P2

**Traces to:** ISSUE-182

---

## Non-Functional Requirements

### NFR-1: Conciseness

The tier policy documentation must be concise to fit within Tier 1's <100 line budget for CLAUDE.md.

**Acceptance Criteria:**

- [ ] Tier documentation section is ≤30 lines
- [ ] Links to detailed docs are used for deep-dive content
- [ ] Formatting is optimized for readability (tables, bullet points)

**Priority:** P0

**Traces to:** ISSUE-182

---

### NFR-2: Consistency with Existing Implementation

The documented policy must align with the actual implementation in the memory subsystem.

**Acceptance Criteria:**

- [ ] Tier definitions match behavior in optimize.go
- [ ] Token budgets match actual retrieval limits
- [ ] Examples reference real file paths and commands
- [ ] No contradictions with existing memory documentation

**Priority:** P0

**Traces to:** ISSUE-182, ISSUE-184, ISSUE-186

---

## Dependencies

| Requirement | Depends On | Reason |
|-------------|------------|--------|
| REQ-7 | ISSUE-186 Phase 3/4 | Skill lifecycle automation must be implemented |
| REQ-6 | ISSUE-184, ISSUE-189 | Optimize pipeline must be complete |
| REQ-5 | ISSUE-184 | CLAUDE.md maintenance (promote/demote) must exist |

---

## Edge Cases

### EC-1: Content Exceeding Tier Limits

**Scenario:** A learning or pattern is valuable but exceeds the token budget for its tier.

**Expected Behavior:**
- Tier 1: Split into multiple sections or demote to Tier 2
- Tier 2: Chunk into multiple skills or keep in Tier 3
- Documentation includes guidance on handling oversized content

---

### EC-2: Tier Boundary Ambiguity

**Scenario:** Content could reasonably fit in multiple tiers (e.g., a frequently-used but long pattern).

**Expected Behavior:**
- Documentation provides clear decision criteria
- Err on the side of lower tier (higher number) to preserve context budget
- Reference optimization pipeline for automatic rebalancing

---

### EC-3: Tier 1 Budget Exhaustion

**Scenario:** CLAUDE.md approaches or exceeds 100 lines due to tier documentation.

**Expected Behavior:**
- Tier documentation is optimized for brevity
- Detailed content moves to separate linked docs
- Clear warning in documentation about budget constraints

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| CLAUDE.md tier section size | ≤30 lines | Line count |
| User comprehension | >80% understand tier distinctions | Post-implementation survey |
| Implementation accuracy | 100% alignment with code | Code review + doc review |
| Promotion/demotion clarity | No ambiguous cases | Review of edge cases |

---

## Out of Scope

The following are explicitly out of scope for this requirements document:

- **Implementation details** of the optimize pipeline (covered by ISSUE-184, ISSUE-189)
- **Skill generation algorithms** (covered by ISSUE-186)
- **UI/UX design** for memory commands (implementation phase)
- **Performance tuning** of tier thresholds (experimentation phase)
- **Migration of existing content** between tiers (separate operational task)

---

## References

- **ISSUE-182:** Define explicit memory tiering policy in CLAUDE.md
- **ISSUE-184:** CLAUDE.md maintenance — interactive consolidation, synthesis output, prune/decay
- **ISSUE-186:** Dynamic skill generation from memory clusters
- **ISSUE-189:** ISSUE-186 follow-up — missing pipeline steps
- **internal/memory/optimize.go:** Core optimization pipeline implementation
- **internal/memory/skill_gen.go:** Skill generation and utility calculation
- **internal/memory/memory.go:** Query with skill integration
