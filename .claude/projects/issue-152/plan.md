# Plan: ISSUE-152 — Integrate Semantic Memory into Orchestration Workflow

**Issue:** ISSUE-152
**Workflow:** new
**Date:** 2026-02-08

---

## 1. Problem Space

### Current State

The `projctl memory` package provides 6 CLI commands (`learn`, `query`, `decide`, `grep`, `session-end`, `extract`) backed by ONNX-based semantic similarity search (e5-small-v2, 384-dim embeddings, SQLite-vec storage). The infrastructure is functional but **disconnected from the orchestration workflow**.

### Gaps Identified

| Gap | Impact |
|-----|--------|
| **No automatic session-end capture** | Project learnings evaporate after each session. Next project starts cold. |
| **Memory is read-only in skills** | 7 skills query memory in GATHER phase but none persist discoveries back. |
| **QA has no memory** | QA repeats the same failure pattern detection each run. Past failures aren't indexed. |
| **context-explorer uses memory but shallowly** | Queries run but results aren't prioritized alongside file/territory results. |
| **Failing semantic ranking test** | `TestIntegration_SemanticSimilarityExampleErrorAndException` — the simplified word-hash tokenizer loses semantic meaning, causing "error handling" to rank "ui design" above "exception management". |

### What Success Looks Like

1. A completed project automatically persists its key decisions and learnings to memory
2. Interview producers surface relevant prior decisions without extra prompting
3. QA leverages known failure patterns to catch recurring issues
4. context-explorer returns memory results alongside file/territory results
5. The semantic ranking test passes (correct similarity ordering)

---

## 2. UX Solution Space

### User-Facing Changes

**None for day-to-day CLI usage.** All integration is internal to the orchestration workflow. Users benefit indirectly:

- Interview questions become more targeted (prior decisions pre-loaded)
- QA catches issues faster (pattern recognition from past projects)
- Less repetitive discovery across projects

### Skill-Level UX

| Skill | Change | User Experience |
|-------|--------|----------------|
| pm/design/arch interview producers | Auto-load prior decisions in GATHER | Interviewer starts with "Based on prior decisions about X..." |
| QA | Query past failure patterns before validation | QA says "This matches a known failure pattern from ISSUE-NNN" |
| context-explorer | Memory results in aggregated output | Context includes semantic matches alongside file matches |
| project orchestrator | Auto `session-end` + `learn` at completion | Transparent — user sees "Session learnings captured" in output |

---

## 3. Implementation Solution Space

### Architecture Approach

**Minimal integration points, maximum leverage.** Rather than deep rewiring, add memory calls at natural workflow boundaries:

1. **Write hooks at project completion** (orchestrator end-of-command)
2. **Read hooks at phase entry** (skill GATHER phases)
3. **Fix the tokenizer** (core infrastructure bug)

### Proposed Changes

#### A. Fix Semantic Ranking (Foundation)

**Problem:** Simplified word-hash tokenizer (`hash(word) % 30000`) loses semantic relationships. "error" and "exception" hash to unrelated positions.

**Fix options:**
1. **Replace with proper tokenizer** — Load the actual SentencePiece/WordPiece vocab from the model. Most correct but adds ~2MB vocab file + tokenizer logic.
2. **Relax the assertion** — Change test to only verify that results contain both entries without strict ordering. Defers the real fix.
3. **Use character n-gram hashing** — Better semantic overlap than word-level hashing while staying simple.

**Recommendation:** Option 1 (proper tokenizer). The memory system's value depends on accurate semantic search. A broken tokenizer undermines the entire feature.

#### B. Session-End Capture (Orchestrator Integration)

Add to the project orchestrator's end-of-command sequence:

```bash
# Existing:
projctl integrate features --dir .
projctl trace repair --dir .
projctl trace validate --dir .

# New:
projctl memory session-end -p "<project-name>"
```

This captures today's decisions into a session summary. The session summary is auto-indexed on next `projctl memory query`.

**Files:** `~/.claude/skills/project/SKILL.md` (end-of-command section)

#### C. Decision Persistence in Producers

After interview producers complete, persist key decisions:

```bash
projctl memory decide -c "<context>" --choice "<decision>" -r "<reason>" -p "<project>"
```

This happens inside the producer skill itself (not orchestrator), since the producer has the decision context.

**Files:** `~/.claude/skills/pm-interview-producer/SKILL.md`, `~/.claude/skills/arch-interview-producer/SKILL.md`, `~/.claude/skills/design-interview-producer/SKILL.md`

#### D. QA Memory Integration

Add GATHER step to QA skill: query memory for known failure patterns related to the current artifact type.

```bash
projctl memory query "common failures in <artifact-type> validation"
```

Include matches in QA context for pattern-aware validation.

**Files:** `~/.claude/skills/qa/SKILL.md`

#### E. context-explorer Enhancement

context-explorer already handles `memory` query type. Enhancement: when no explicit memory query is provided, auto-run a memory query derived from the requester's context (issue description, current phase).

**Files:** `~/.claude/skills/context-explorer/SKILL.md`

### Dependency Order

```
TASK-1: Fix tokenizer (foundation — everything depends on accurate embeddings)
   |
TASK-2: Session-end in orchestrator (write path — captures data)
   |
   +--- TASK-3: Decision persistence in producers (write path — more data)
   |
   +--- TASK-4: QA memory integration (read path — uses data)
   |
   +--- TASK-5: context-explorer enhancement (read path — uses data)
```

TASK-1 is blocking. TASK-2 is next. TASK-3/4/5 are parallel after TASK-2.

### Scope Boundaries

**In scope:**
- Fix semantic ranking test
- Add session-end to orchestrator end-of-command
- Add decision persistence to interview producers
- Add memory queries to QA GATHER phase
- Enhance context-explorer with auto-memory queries

**Out of scope:**
- Memory retention/expiration policies
- Memory UI/dashboard
- Cross-machine memory sync
- Memory-based interview depth prediction

---

## 4. Risk Assessment

| Risk | Mitigation |
|------|------------|
| Proper tokenizer adds complexity | Use existing Go tokenizer libraries; keep the simple hash as fallback for environments without vocab file |
| ONNX model download on first use (~90MB) | Already handled — existing download logic is robust |
| Memory queries add latency to GATHER | Queries run alongside file/territory lookups (already parallel in context-explorer) |
| Stale memories mislead future projects | Session summaries include dates; skills can filter by recency |

---

## 5. Acceptance Criteria Summary

- [ ] `TestIntegration_SemanticSimilarityExampleErrorAndException` passes
- [ ] Project completion triggers `projctl memory session-end`
- [ ] Interview producers persist key decisions via `projctl memory decide`
- [ ] QA skill queries memory for failure patterns during GATHER
- [ ] context-explorer auto-queries memory when no explicit memory query given
- [ ] All changes have tests (TDD: red → green → refactor)
