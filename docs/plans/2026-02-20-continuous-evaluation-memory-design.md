# Continuous Evaluation Memory Pipeline Design

**Date:** 2026-02-20
**Issue:** ISSUE-235
**Status:** Draft
**Evolves:** [Self-Reinforcing Learning Design](2026-02-14-self-reinforcing-learning-design.md)
**Scope:** Global memory system architecture

## Problem

The memory system has a solid foundation (embeddings, skills, CLAUDE.md, hooks) but
the pipeline connecting these tiers is mechanical where it should be qualitative:

1. **Promotion is count-based, not quality-based.** 5 retrievals + 3 projects = promote.
   No measurement of whether the memory actually helps when surfaced.

2. **CLAUDE.md is an append-only log.** Everything promoted lands in "## Promoted Learnings"
   as flat bullets. No section routing, no quality gate, no size discipline.

3. **No impact tracking.** We know how often a memory is retrieved (importance) but not
   whether retrieval improves outcomes (impact). A memory surfaced 100 times and ignored
   every time looks identical to one surfaced 100 times and followed every time.

4. **No inline evaluation.** Memories are surfaced raw. The agent receives unfiltered
   retrieval results as system reminders, with no relevance filtering or synthesis.

5. **Batch optimization runs periodically** instead of learning continuously from every
   interaction. The system only gets smarter when someone runs `optimize`.

## Vision

Replace the batch promote/demote pipeline with a continuous inline evaluation loop that:

- Filters and synthesizes surfaced memories in real-time (E5 → Haiku → Sonnet)
- Tracks both importance (how often it comes up) and impact (does it help when it does)
- Routes content to the right tier based on measured effectiveness, not mechanical thresholds
- Presents all tier changes as user-approved proposals, never auto-writes
- Diagnoses systematic failures (leeches) with root cause analysis

## Architecture: The Evaluation Loop

```
  User prompt / tool call
          ↓
    Hook fires (existing)
          ↓
    E5 retrieves top-K (local, ~50ms, free)
          ↓
    ┌─────────────────────────────────────┐
    │  [NEW] Haiku Filter (~200ms, ~$0.0001)  │
    │  - Relevant / noise / should-be-hook    │
    │  - Logs surfacing_event per candidate   │
    │  - Context precision = kept/total       │
    └─────────────────────────────────────┘
          ↓
    ┌─────────────────────────────────────┐
    │  [NEW] Sonnet Synthesis (conditional)   │
    │  - Only if 2+ relevant memories         │
    │  - Actionable guidance paragraph        │
    │  - Side-effect: routing proposal        │
    │  (~1s, ~$0.005)                         │
    └─────────────────────────────────────┘
          ↓
    Synthesized context → agent
          ↓
    Agent responds, user interacts
          ↓
    ┌─────────────────────────────────────┐
    │  [NEW] Post-Interaction Eval (async)    │
    │  - Sampled ~10% of interactions         │
    │  - Haiku: "Was guidance followed?"      │
    │  - Implicit: correction = negative      │
    │  - Updates surfacing_event outcome      │
    └─────────────────────────────────────┘
          ↓
    Continuous scoring feeds into optimize
```

### Cost Per Day (~100 interactions)

| Step              | Rate         | Unit Cost         | Daily Cost |
| ----------------- | ------------ | ----------------- | ---------- |
| E5 retrieval      | 100%         | free (local ONNX) | $0.00      |
| Haiku filter      | 100%         | ~$0.0001          | $0.01      |
| Sonnet synthesis  | ~20% trigger | ~$0.005           | $0.10      |
| Post-eval (Haiku) | ~10% sample  | ~$0.0001          | $0.001     |
| **Total**         |              |                   | **~$0.11** |

## The Importance × Impact Matrix

Every memory gets scored on two dimensions:

**Importance** = ACT-R activation (frequency × recency × cross-session spread).
Already implemented. Captures "how often does this come up?"

**Impact** = effectiveness when surfaced. NEW.
Captures "does surfacing this actually help?"

### Quadrant Classification

```
                    High Impact              Low Impact
                ┌──────────────────┬──────────────────────┐
  High          │                  │                      │
  Importance    │  WORKING         │  LEECH               │
                │  Keep or promote │  Diagnose root cause │
                │                  │                      │
                ├──────────────────┼──────────────────────┤
  Low           │                  │                      │
  Importance    │  HIDDEN GEM      │  NOISE               │
                │  Surface more    │  Natural decay        │
                │                  │                      │
                └──────────────────┴──────────────────────┘
```

### Quadrant Actions (all as user-approved proposals)

**Working Knowledge** (high importance, high impact):

- Memory is frequently relevant AND consistently effective.
- Action: Keep in current tier. If universal across 3+ projects, propose promotion.
- Proposal: "This learning is consistently helpful across N projects. Add to CLAUDE.md?"

**Leech** (high importance, low impact):

- Memory keeps matching contexts BUT outcomes don't improve when surfaced.
- This is a DIAGNOSTIC signal — the memory itself needs fixing, not more exposure.
- Present root cause analysis with options:

| Diagnosis          | Signal                                  | Proposed Action                        |
| ------------------ | --------------------------------------- | -------------------------------------- |
| Content quality    | Surfaced often, agent doesn't follow    | "Rewrite: [suggested clearer version]" |
| Wrong tier         | Surfaced after mistake already happened | "Move to CLAUDE.md (loaded earlier)"   |
| Enforcement gap    | Agent understands but doesn't comply    | "Convert to hook (deterministic)"      |
| Retrieval mismatch | Surfaced in irrelevant contexts         | "Narrow scope or improve embedding"    |

Inspired by spaced repetition "leech" handling: the problem is the card, not the
learner. Rewrite or re-tier, don't keep hammering.

**Hidden Gem** (low importance, high impact):

- Rarely surfaced, but very effective when it is.
- Action: Surface more aggressively (lower similarity threshold for this memory).
- Proposal: "This learning is rarely triggered but very effective. Make more discoverable?"

**Noise** (low importance, low impact):

- Rarely surfaced AND not effective when it is.
- Action: No proposal needed. Natural confidence decay + pruning handles this.

## ACT-R Enhancements

### Current State (already implemented)

- Base-level activation: `B_i = ln(Σ t_j^(-d))`
- Memory type decay: correction d=0.1 (indefinite), reflection d=0.5 (30-day window)
- Session bonus: 1.5x multiplier when retrievals span multiple sessions (30min gap)

### New: Impact-Weighted Activation

Modify the activation formula to include effectiveness:

```
B_i = ln(Σ t_j^(-d)) + α × effectiveness_i
```

Where:

- `α` = weighting parameter (start at 0.5, tune empirically)
- `effectiveness_i` = impact score from surfacing_events (range -1.0 to 1.0)

Effect: A leech (surfaced often, doesn't help) gets high activation from frequency
but the effectiveness penalty drags it down. A hidden gem gets boosted despite low
frequency because its impact is high.

### New: Spreading Activation

After E5 retrieves top-K memories, boost activation of semantically similar memories
(similarity > 0.7). Makes related knowledge "warm up" together.

Implementation: One additional vector search per query, returning memories similar to
the top-K results but not in the top-K themselves. These get a temporary activation
boost for the current interaction only (not persisted).

### New: Associative Strength

When memory X is surfaced in context Y and proves effective (high faithfulness score),
strengthen the X↔Y association. Over time, memories become specialized for contexts
where they actually work.

Implementation: Track (memory_id, context_embedding) pairs with effectiveness scores
in surfacing_events. During retrieval, boost memories that have historically been
effective in similar contexts.

### Why Not Full ACT-R

- **Partial matching**: Our MinScore threshold is a hard cutoff — memory either crosses
  0.3 similarity or doesn't. ACT-R partial matching scores *degree* of match and adjusts
  activation proportionally (a memory at 0.29 similarity gets reduced-but-nonzero
  activation rather than zero). Our proposed associative strength (boosting memories
  historically effective in similar contexts) gets us most of the way there without
  formal partial matching. **Decision:** Skip for now, but revisit if hidden gems
  consistently sit just below threshold (context recall metrics will surface this).

- **Chunk-based representation**: ACT-R chunks are structured symbolic records with
  named slots (`type: "lesson", domain: "git", action: "never-amend-pushed"`) supporting
  slot-level queries ("find all where domain=git"). Embeddings are dense vectors where
  meaning is distributed across dimensions — they support similarity queries but not
  structured queries. We partially bridge this gap already (memory_type, source,
  project_context columns). **Decision:** Skip formal chunk representation, but if
  context recall metrics are low, explore formalizing slot-based retrieval alongside
  vector retrieval.

- **Production rules**: ACT-R production rules are `IF condition THEN action` fired
  automatically. Our hooks cover some cases (PreToolUse validation, Stop checks) but
  have gaps: no retrieval-time suppression rules, no inter-memory priority rules. The
  Haiku filter layer effectively serves as a flexible soft production rule engine
  ("is this relevant here?" = soft IF/THEN). The leech-to-hook conversion is reactive
  (after diagnosis), not proactive. **Decision:** Hooks + Haiku filter together
  constitute our production rule system. No additional hook types needed initially,
  but this framing should be explicit in the architecture.

We adopt the concepts that fit (spreading activation, associative strength,
impact-weighted activation) and defer the others with clear revisit triggers
based on measured metrics.

## RAGAS-Adapted Metrics

### Context Precision (signal-to-noise of retrieval)

- **Definition**: Of memories E5 returned, what fraction did Haiku keep?
- **Measurement**: Haiku filter ratio per query, logged in surfacing_events.
- **What low precision means**: Embeddings are matching too broadly. The E5 model or
  the similarity threshold needs tuning.
- **Aggregated per memory**: "What's this memory's average precision across all
  surfacings?" A memory that consistently passes Haiku filtering has high precision.

### Faithfulness (was the surfaced memory used?)

- **Definition**: Did the agent's actions align with the surfaced guidance?
- **Measurement**: Haiku post-evaluation on ~10% sample of interactions.
- **What low faithfulness means**: The memory is surfaced but not followed. Either
  the content is poorly written, or it's the wrong tier (needs CLAUDE.md or hook).
- **This IS the impact score.** Faithfulness is the core signal for the matrix.

### Context Recall (did we miss something?)

- **Definition**: When the user corrects the agent, did a relevant memory exist in
  the DB but fail to surface?
- **Measurement**: After user corrections, search DB for matching memories. If found
  but not surfaced → recall failure.
- **What low recall means**: The memory exists but its embedding doesn't match the
  contexts where it's needed. Needs re-embedding or broader matching.

### Why Not Full RAGAS

RAGAS assumes ground-truth answers (reference-based evaluation): you have a question,
a known correct answer, and the system's generated answer, and you score how well the
generated answer matches the reference. Example: Q: "What's the git commit trailer?"
Reference: "AI-Used: [claude]". System answer: "Co-Authored-By: Claude". Score: 0.0.

We don't have reference answers — we don't know in advance what the "right" response
to surfaced memories should be. Instead of comparing against ground truth, we use:

- **Haiku-as-judge**: "Did the agent's behavior align with the guidance?" (subjective
  but cheap, and accumulates signal over many evaluations)
- **Implicit signals**: User corrections, re-teaching, etc. (noisy but free)
- **Explicit feedback**: `/memory helpful` (gold signal but rare)

This is the difference between a scored exam (RAGAS) and peer review + behavioral
observation (us). Less precise per-measurement, but we compensate with volume and
continuous accumulation.

## Data Model Changes

### New Table: `surfacing_events`

```sql
CREATE TABLE IF NOT EXISTS surfacing_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    memory_id INTEGER NOT NULL,
    query_text TEXT NOT NULL,
    hook_event TEXT NOT NULL,        -- "UserPromptSubmit", "PreToolUse", etc.
    timestamp TEXT NOT NULL,         -- RFC3339
    session_id TEXT,
    -- Haiku filter results
    haiku_relevant BOOLEAN,          -- Did Haiku keep this?
    haiku_tag TEXT,                   -- "relevant", "noise", "should-be-hook", etc.
    haiku_relevance_score REAL,      -- 0.0-1.0 confidence
    -- Post-interaction evaluation
    faithfulness REAL,               -- 0.0-1.0, filled async by post-eval
    outcome_signal TEXT,             -- "positive", "negative", null
    user_feedback TEXT,              -- "helpful", "wrong", "unclear", null
    -- Metadata
    e5_similarity REAL,              -- Original E5 cosine similarity score
    context_precision REAL,          -- Kept/total ratio for this query batch
    FOREIGN KEY (memory_id) REFERENCES embeddings(id)
);

CREATE INDEX idx_surfacing_memory ON surfacing_events(memory_id);
CREATE INDEX idx_surfacing_timestamp ON surfacing_events(timestamp);
```

### Modified `embeddings` Table — New Columns

```sql
ALTER TABLE embeddings ADD COLUMN importance_score REAL DEFAULT 0.0;
ALTER TABLE embeddings ADD COLUMN impact_score REAL DEFAULT 0.0;
ALTER TABLE embeddings ADD COLUMN effectiveness REAL DEFAULT 0.0;
ALTER TABLE embeddings ADD COLUMN quadrant TEXT DEFAULT 'noise';
ALTER TABLE embeddings ADD COLUMN leech_count INTEGER DEFAULT 0;
```

### Computed Scores (during optimize or continuous background)

```
importance_score = ACT-R base-level activation (existing formula)

impact_score = avg(faithfulness) from surfacing_events
               WHERE memory_id = this AND faithfulness IS NOT NULL
               -- Weighted by recency: recent evaluations matter more

effectiveness = importance_score + α × impact_score
                -- α starts at 0.5, tunable

quadrant = CASE
    WHEN importance_score >= threshold AND impact_score >= threshold THEN 'working'
    WHEN importance_score >= threshold AND impact_score < threshold  THEN 'leech'
    WHEN importance_score < threshold  AND impact_score >= threshold THEN 'gem'
    ELSE 'noise'
END

leech_count = consecutive surfacings where faithfulness < 0.3
              -- Reset to 0 when faithfulness >= 0.3
              -- Trigger leech diagnosis at leech_count >= 5
```

## Impact Measurement: Three Signal Layers

### Layer 1: Implicit Signals (free, automatic, noisy)

| Signal                             | Detection                                     | Meaning                          |
| ---------------------------------- | --------------------------------------------- | -------------------------------- |
| No user correction in next 3 turns | Absence of correction keywords/patterns       | Weak positive                    |
| User corrects in same area         | Correction detected via extraction pipeline   | Negative                         |
| Same lesson re-taught later        | New Learn() with >0.85 similarity to existing | Strong negative (recall failure) |
| Agent explicitly references memory | Content substring match in agent response     | Strong positive                  |

These accumulate automatically via the existing extraction pipeline. No additional
API calls needed.

### Layer 2: Haiku Evaluation (cheap, sampled, moderate signal)

Run on ~10% of interactions (configurable). Haiku receives:

- The surfaced memories
- The agent's subsequent response
- The user's next message (if available)

Prompt: "Did the agent's response align with the surfaced guidance? Score 0-1."

This produces the faithfulness score stored in surfacing_events.

**Sampling strategy**: Always evaluate when:

- User correction detected (100% sample for negative signals)
- Memory was tagged "should-be-hook" or "should-be-earlier" by Haiku filter
- Memory's leech_count >= 3 (increase sampling for suspected leeches)

### Layer 3: Explicit User Feedback (expensive for user, gold signal)

Commands: `/memory helpful`, `/memory wrong`, `/memory unclear`

Immediately updates the most recent surfacing_event with user_feedback.
High weight in impact scoring (one explicit signal = ~10 implicit signals).

## CLAUDE.md Quality Gate (Resolving ISSUE-235)

With the continuous evaluation pipeline, CLAUDE.md management becomes a consequence
of the importance × impact matrix rather than a separate system.

### How Content Enters CLAUDE.md

1. Memory reaches "working knowledge" quadrant (high importance + high impact)
2. Memory is universal (surfaced across 3+ projects)
3. Quality gate checks:
   - **Actionable?** Can Claude follow this instruction? (Haiku judge)
   - **Universal?** Not project-specific? (existing isNarrowByKeywords + Haiku)
   - **Non-redundant?** Not covered by existing hook, skill, or CLAUDE.md entry?
   - **Right tier?** Would a hook or skill be more appropriate?
4. Content classified into section (Commands, Architecture, Testing, Gotchas, etc.)
5. Formatted for section type (tables for commands, bullets for gotchas)
6. **Proposal presented to user** — never auto-written

### Section-Aware Parser

Extend `ParseCLAUDEMD()` to recognize typed sections. Sections are not a fixed
taxonomy — they're created based on what content warrants them. But the parser
understands common types for formatting:

| Section Type | Format                            | Example                                      |
| ------------ | --------------------------------- | -------------------------------------------- |
| Commands     | Table: Command \| Description     | `go test -tags sqlite_fts5`                  |
| Architecture | Tree diagram or directory listing | `internal/` = implementation                 |
| Gotchas      | Bullets with emphasis             | **NEVER** amend pushed commits               |
| Code Style   | Short rules                       | Use `make([]T, 0, capacity)` when size known |
| Testing      | Commands + patterns               | Blackbox tests: `package foo_test`           |

### ScoreClaudeMD()

Quality scoring function implementing adapted RAGAS metrics:

| Dimension         | Weight | How Measured                                                 |
| ----------------- | ------ | ------------------------------------------------------------ |
| Context Precision | 20%    | Are entries actionable? (Haiku evaluation per entry)         |
| Faithfulness      | 25%    | Do entries actually change behavior? (from surfacing_events) |
| Currency          | 20%    | Do commands work? Do file paths exist? (filesystem check)    |
| Conciseness       | 15%    | Line count, filler detection, redundancy with skills/hooks   |
| Coverage          | 20%    | Are high-impact universal memories represented?              |

Reported during `optimize --review`. Grade scale: A (90+) through F (0-29).

### Size Enforcement

Budget: <100 lines for global CLAUDE.md. When over budget:

1. Score each entry using effectiveness from surfacing_events
2. Lowest-effectiveness entries proposed for demotion
3. Demotion destinations determined by leech diagnosis logic
4. User approves all changes

## What Gets Simpler

| Current Complexity                              | Replaced By                                   |
| ----------------------------------------------- | --------------------------------------------- |
| Promotion thresholds (5 retrievals, 3 projects) | Importance × impact quadrant                  |
| `appendToClaudeMD()` auto-writes                | `proposeCLAUDEMDChange()` — requires approval |
| `isStaleEntry()` 90-day timestamp check         | Actual impact measurement                     |
| Mechanical demotion triggers                    | Leech diagnosis with root cause               |
| 12+ configurable thresholds                     | 2 dimensions (importance, impact) + α weight  |
| `isNarrowByKeywords()` as sole classifier       | Haiku classification during surfacing         |

## What Stays Unchanged

- SQLite + sqlite-vec storage (not the bottleneck)
- E5-small-v2 local ONNX embeddings
- Skills tier and `generated_skills` table
- Hook infrastructure (Stop, PreCompact, SessionStart, etc.)
- `FileSystem` abstraction for testing
- Learn/Query core functions
- Session extraction pipeline
- Skill testing harness (RED/GREEN protocol)
- Changelog/retrieval JSONL logging

## What's New to Build

| Component                                       | Complexity | Dependencies                  |
| ----------------------------------------------- | ---------- | ----------------------------- |
| `surfacing_events` table + schema migration     | Low        | None                          |
| `embeddings` table new columns + migration      | Low        | None                          |
| Haiku filter step in hook pipeline              | Medium     | DirectAPIExtractor            |
| Sonnet synthesis step (conditional)             | Medium     | DirectAPIExtractor            |
| Post-interaction faithfulness evaluation        | Medium     | Haiku filter                  |
| Implicit signal detection                       | Low        | Existing extraction pipeline  |
| `/memory helpful/wrong/unclear` commands        | Low        | surfacing_events table        |
| Importance × impact scoring                     | Medium     | surfacing_events + ACT-R      |
| Quadrant classification                         | Low        | Scoring                       |
| Leech diagnosis engine                          | Medium     | Quadrant + surfacing history  |
| Quality gate for CLAUDE.md entry                | Medium     | Haiku judge                   |
| Section-aware `ParseCLAUDEMD()`                 | Low        | Existing parser               |
| `ScoreClaudeMD()` function                      | Medium     | surfacing_events + filesystem |
| `proposeCLAUDEMDChange()` (replaces auto-write) | Low        | Quality gate                  |
| Spreading activation in retrieval               | Low        | Existing vector search        |
| Impact-weighted ACT-R activation                | Low        | impact_score column           |

## Design Decisions

1. **Haiku filters, Sonnet synthesizes.** Haiku is fast and cheap enough for every
   interaction. Sonnet is reserved for when synthesis adds genuine value (2+ relevant
   memories that need combining). This keeps latency and cost acceptable.

2. **Sampling, not exhaustive evaluation.** Post-interaction faithfulness evaluation
   runs on ~10% of interactions, with 100% sampling for detected corrections. This
   keeps API costs low while still accumulating meaningful signal.

3. **Proposals, never auto-writes.** The self-reinforcing learning design established
   that "CLAUDE.md is a view, not a store." We strengthen this: no tier change to
   CLAUDE.md, skills, or hooks happens without user approval. The system proposes,
   the user disposes.

4. **Two dimensions replace twelve thresholds.** The current system has 12+ configurable
   thresholds (min retrievals, min projects, similarity, utility, confidence, etc.).
   We collapse to two measured dimensions (importance, impact) plus one tunable
   parameter (α weight). Simpler to understand, harder to game.

5. **Leech diagnosis, not leech deletion.** High importance + low impact is the most
   valuable signal in the system. It means something is systematically broken.
   Presenting root cause analysis to the user is more valuable than any automated fix.

6. **ACT-R as foundation, not religion.** We adopt spreading activation, associative
   strength, and impact-weighted activation because they solve real problems. We defer
   partial matching, chunk representation, and production rules because our existing
   abstractions (MinScore threshold, embeddings, hooks + Haiku filter) cover those
   use cases — with clear revisit triggers if metrics indicate gaps.

7. **RAGAS metrics adapted, not adopted.** We borrow context precision, faithfulness,
   and context recall as measurement concepts. We don't adopt RAGAS's reference-based
   evaluation methodology because we don't have ground truth. Haiku-as-judge replaces
   reference comparison.

## Relationship to Existing Design

This design evolves the [Self-Reinforcing Learning Design](2026-02-14-self-reinforcing-learning-design.md):

| Existing Concept                                | Evolution                                                                           |
| ----------------------------------------------- | ----------------------------------------------------------------------------------- |
| EXTRACT → STORE → RETRIEVE → MEASURE → OPTIMIZE | Same loop, but MEASURE happens inline (every interaction) not just at optimize time |
| Retrieval relevance scoring                     | Formalized as RAGAS faithfulness with surfacing_events tracking                     |
| Correction recurrence tracking                  | Becomes a context recall signal (memory exists but wasn't surfaced)                 |
| Skill test harness (RED/GREEN)                  | Unchanged — still validates before deployment                                       |
| Changelog/retrieval JSONL logging               | Unchanged — surfacing_events adds structured evaluation data                        |
| "CLAUDE.md is a view, not a store"              | Strengthened: all changes are proposals, quality-gated, section-aware               |

## Resolved Questions

1. **α weight tuning.** Start at 0.5 and auto-tune. Track metric distributions over
   time and adjust to maintain target percentages (e.g., leeches should be ~5-15% of
   actively surfaced memories — if way higher, threshold is too strict). Avoids
   hardcoded magic numbers while keeping the system self-calibrating.

2. **Leech threshold.** Start at 5 consecutive low-impact surfacings and auto-tune
   using the same distribution-tracking approach as α weight.

3. **Sonnet synthesis trigger.** Replace mechanical "2+ relevant memories" rule with
   Haiku-predicted synthesis value. Haiku is already evaluating relevance — adding
   "would combining these produce more actionable guidance than presenting separately?"
   is a natural extension of the same call, essentially free.

4. **Haiku filter latency budget.** Start with Haiku on all hook types (UserPromptSubmit,
   PreToolUse, etc.) for maximum signal. Instrument latency per hook type. Optimize
   later with data — e.g., if PreToolUse latency is unacceptable, fall back to raw
   E5 results for that hook only.

5. **Retroactive scoring.** Start with a blank slate. Historical data in
   `retrievals.jsonl` is too sparse and noisy to bootstrap meaningful impact scores.
   Clean start avoids contaminating the new system with unreliable data.

6. **When does scoring happen?** End-of-session scoring as the primary mechanism.
   Score updates happen in PreCompact / PreClear / Stop hooks — these natural session
   checkpoints batch-process all surfacing_events from the current session.

   Three options were considered:

   | Option | Mechanism | Pro | Con |
   |--------|-----------|-----|-----|
   | A: In-hook | Score during the hook that surfaces memories | Truly continuous | Adds latency; evaluating *previous* interaction during *current* one requires carrying forward state between invocations |
   | B: Background worker | Async process consuming surfacing_event queue | Zero hook latency | Needs long-running process; CLI sessions are ephemeral |
   | **C: End-of-session** | Batch-process in PreCompact/PreClear/Stop hooks | Automatic; full session context available; no per-interaction latency cost | Not truly continuous within a session |

   **Decision:** Option C. Gives "automatic, not dependent on `optimize`" while
   avoiding background worker complexity. For faithfulness evaluation, only the
   surfacing event + agent's subsequent response + user's next message are needed —
   full session context is not required, so bounded evaluation of each surfacing
   event is feasible. Can evolve to Option A later if end-of-session granularity
   proves insufficient.

## Open Questions

1. **Pre-clear hook.** `/clear` is increasingly used as a session boundary. Need to
   investigate whether Claude Code exposes a hookable event for `/clear`, and if not,
   whether we can add one. This is required for Option C scoring to work reliably
   (without it, `/clear` sessions lose their surfacing_events). Tracked as a meta case.

2. **Auto-tune convergence.** The auto-tuning mechanism for α and leech threshold
   needs a convergence strategy — how quickly should it adapt? Too fast risks
   oscillation, too slow defeats the purpose. Needs empirical testing.

## Clarifications

### Session 2026-02-20

**ACT-R scope decisions:**

- Partial matching deferred, not rejected. MinScore hard threshold covers most cases;
  associative strength provides graduated matching for historically effective memories.
  Revisit trigger: hidden gems consistently sitting just below threshold (measurable
  via context recall metrics).
- Chunk-based representation unnecessary given existing structured columns (memory_type,
  source, project_context) alongside embeddings. Revisit trigger: low context recall
  suggesting slot-based queries would improve retrieval.
- Production rules covered by hooks + Haiku filter as a combined system. Hooks provide
  deterministic enforcement; Haiku filter provides flexible soft matching. Leech-to-hook
  conversion bridges the gap reactively. No additional hook types needed initially.

**RAGAS adaptation rationale:**

- Standard RAGAS requires ground-truth reference answers (scored exam model). We lack
  reference answers, so we substitute Haiku-as-judge + implicit behavioral signals +
  explicit user feedback (peer review + observation model). Less precise per-measurement
  but compensated by volume and continuous accumulation.

**Scoring architecture:**

- End-of-session scoring (Option C) chosen over in-hook scoring (Option A) and
  background workers (Option B). Key factors: CLI sessions are ephemeral (rules out B),
  carrying state between hook invocations is complex (argues against A), and session-end
  hooks (PreCompact/PreClear/Stop) are natural checkpoints. Faithfulness evaluation
  is bounded per-event (surfacing + response + user reply), so full session context
  is not required.

**Auto-tuning approach:**

- Both α weight and leech threshold use distribution-tracking: monitor what percentage
  of memories fall into each quadrant, adjust thresholds to maintain target distributions.
  Avoids hardcoded magic numbers. Convergence strategy still open (question 2).

**Meta case: pre-clear hook:**

- `/clear` increasingly used as a session boundary. Without a pre-clear hook, Option C
  scoring loses surfacing_events for cleared sessions. Needs investigation into Claude
  Code's hookable event surface.
