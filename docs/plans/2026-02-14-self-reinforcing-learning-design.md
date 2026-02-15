# Self-Reinforcing Learning System Design

**Date:** 2026-02-14
**Status:** Approved
**Scope:** Global (all Claude Code sessions, all projects)

## Problem

The memory system has sophisticated plumbing for moving knowledge between tiers, but:
- Content quality is poor (mechanical synthesis produces noise like "important pattern for review" 56+ times)
- No verification that learning actually works end-to-end
- No visibility into what the system learns or how it changes over time
- No automated testing that skills are effective before deployment
- Optimize commands exist but their distinction is unclear and their triggering is manual
- Corrections don't reliably stick — the same mistake can recur

## Vision

A self-reinforcing learning system that:
1. Automatically extracts learnings from every session (corrections, patterns, behavioral shifts)
2. Measures whether those learnings actually prevent future mistakes
3. Auto-tests skill candidates before deploying them
4. Provides visibility without requiring constant attention
5. Learns from its own corrections — when a human adjusts the system, that adjustment feeds back in

## Architecture: The Core Loop

```
EXTRACT → STORE → RETRIEVE → MEASURE
   ↑                            |
   └────── OPTIMIZE ←───────────┘
```

### EXTRACT (automatic, every session)

**Trigger:** Stop hook and PreCompact hook run `extract-session`.

**What it does:**
- Parses session transcript for corrections, patterns, behavioral conventions
- Classifies into Tier A (confidence 1.0, explicit corrections), Tier B (confidence 0.7, inferred patterns), Tier C (ancillary observations)
- Detects "remember X" / "don't forget X" / "always do X" as explicit Tier A signals (confidence 1.0, normal decay applies — no floor)
- Detects contradictions: "remember targ" supersedes "remember mage" — old entry decayed aggressively
- Enriches via DirectAPIExtractor (Haiku, direct API, ~730ms/call)

**Quality requirement:** LLM synthesis must be the default path. The mechanical `generatePattern()` that produces "important pattern for review" must be replaced or gated behind LLM validation. Every synthesized learning must be actionable, specific, and non-redundant.

### STORE (automatic, immediate)

**What it does:**
- Stores learnings in `~/.claude/memory/embeddings.db` with full metadata (confidence, observation_type, concepts, enriched_content)
- Every storage event logged to `~/.claude/memory/changelog.jsonl`

**Changelog entry format:**
```jsonl
{"timestamp":"...","action":"store","tier":"embeddings","content":"Use AI-Used trailer, not Co-Authored-By","confidence":1.0,"reason":"explicit correction","session_id":"..."}
```

### RETRIEVE (automatic, every interaction)

**Trigger:** SessionStart, UserPromptSubmit, PreToolUse hooks.

**What it does:**
- Hybrid search (vector cosine similarity + BM25 keyword + Reciprocal Rank Fusion)
- **Similarity threshold (0.7 default):** Do not return matches below threshold. Return empty rather than garbage.
- Primacy ordering: corrections prioritized over observations
- Confidence floor per hook (SessionStart: 0.3, UserPromptSubmit: 0.3, PreToolUse: 0.5)

**Retrieval logging:** Every retrieval logged to `~/.claude/memory/retrievals.jsonl`:
```jsonl
{"timestamp":"...","hook":"UserPromptSubmit","query":"...","results":[{"id":"...","content":"...","score":0.82,"tier":"embeddings"}],"session_id":"..."}
```

### MEASURE (automatic, continuous)

**What it tracks:**

1. **Correction recurrence:** Same correction happens again despite existing embedding on that topic. Signal: learning failed to stick. Action: escalate to skill candidate.

2. **Retrieval relevance:** Of memories injected into context, were they actually useful? Inferred from: was the agent corrected on a topic where a relevant memory was already injected? If yes: memory content is insufficient — flag for refinement.

3. **Hook violation trends:** Are violations declining over time? Tracked per-rule. Declining = learning is working. Persistent = rule isn't being internalized, escalate to CLAUDE.md.

4. **Skill deployment success rate:** Of skills that passed auto-testing and were deployed, how many required later correction?

**Storage:** `~/.claude/memory/metrics.jsonl` — periodic snapshots of composite metrics.

### OPTIMIZE (manual trigger via `projctl memory optimize`)

Uses accumulated metrics from MEASURE to make informed decisions across all three tiers.

#### Embeddings tier operations:
- **Decay:** Confidence degrades over time (0.9^days). Retrieval-informed: frequently-retrieved memories decay slower.
- **Dedup:** Merge entries >0.95 cosine similarity.
- **Prune:** Archive entries below 0.1 confidence (moved to append-only archive, never deleted).
- **Contradiction:** When new learning contradicts existing, old entry decayed aggressively.
- **Promote to skill:** When 3+ similar embeddings cluster, compile into skill candidate.

#### Skills tier operations:
- **Refine:** Skill is retrieved but corrections still follow on that topic. LLM rewrites skill with accumulated counter-examples. Auto-tested before redeployment.
- **Split:** Skill covers too many topics (incoherence detected via internal similarity). Re-cluster into focused sub-skills. Each sub-skill auto-tested.
- **Merge:** Two skills >0.85 similarity. Consolidate into one richer skill. Auto-tested.
- **Demote:** Skill never retrieved (retrieval_count=0 after N sessions) or low utility (<0.4 after 5+ retrievals). Content moves to high-confidence embedding — knowledge preserved, just not loaded as a skill.
- **Promote to CLAUDE.md:** Utility >=0.8, used across 3+ projects, 5+ retrievals. Auto-tested at highest bar before promotion.

#### CLAUDE.md tier operations:
- **Demote to skill:** Entry is domain-specific or procedural. Content becomes a richer skill (CLAUDE.md one-liner expands into full skill with examples and anti-patterns).
- **Demote to hook:** Entry is deterministic and always applies. Migrated to hook enforcement — zero context window cost.
- **Demote to embedding:** Entry is narrow or situational. Stored as high-confidence embedding, retrievable when relevant.
- **Consolidate:** Redundant entries merged.
- **Prune:** Entries that are stale or superseded. Content ALWAYS moves to a lower tier first — CLAUDE.md demotion is never deletion.
- **Size enforcement:** Budget target <100 lines. When over budget, lowest-utility entries demoted (with safe destination).

**Critical invariant: CLAUDE.md is a view, not a store.** It surfaces the highest-value content from the system. When something leaves CLAUDE.md, it moves to a tier where it remains accessible — just not always-loaded. The budget constraint forces prioritization, not information loss.

#### Measurement signals driving optimize decisions:

| Signal | What it tells us | Action |
|---|---|---|
| Skill retrieved, no correction follows | Skill is working | Boost confidence |
| Skill retrieved, correction follows on same topic | Skill is insufficient | Refine skill content |
| Skill never retrieved | Not discoverable or not needed | Improve description or demote |
| Same correction happens twice | Learning failed to stick | Escalate: create/improve skill |
| CLAUDE.md entry overlaps with skill | Redundant | Demote from CLAUDE.md |
| Hook violation declining | Rule is internalized | Consider relaxing hook |
| Hook violation persistent | Rule isn't being learned | Escalate: add to CLAUDE.md |

## Automated Skill Testing

When `memory optimize` compiles, refines, merges, or promotes a skill, it does not deploy immediately. It tests first.

### Test protocol

**Step 1: Derive pressure scenario from original failures.**

The skill candidate exists because multiple embeddings clustered around a theme. Each embedding came from a session where a correction happened. The system extracts the context of those corrections to build a realistic test scenario.

**Step 2: RED — Run scenario without skill (N=3 runs).**

- Direct Anthropic API call (Haiku, temperature 0.0)
- No Claude Code process, no plugins, no hooks, no memory retrievals — clean room
- System prompt contains only the scenario context
- Pass condition: >=2 of 3 runs exhibit the failure mode
- If the agent already gets it right without the skill: skill is redundant, don't deploy

**Step 3: GREEN — Run scenario with skill (N=3 runs).**

- Same API setup, same scenario
- Skill content injected into system prompt
- Pass condition: >=2 of 3 runs show correct behavior

**Step 4: Deploy or reject.**

| RED result | GREEN result | Action |
|---|---|---|
| Fails as expected | Passes | Deploy skill, log to changelog |
| Already correct | N/A | Skill redundant, don't deploy, log reason |
| Fails as expected | Still fails | Skill ineffective, don't deploy, flag for refinement |
| Mixed results | Any | Don't deploy, flag for human review |

**Judging criteria:** Prefer deterministic string match (did the response include the correct command/pattern?). Fall back to LLM-as-judge via separate API call when behavior is nuanced. Judge call is also clean-room — no contamination.

**Cost:** ~$0.01-0.02 per skill test (6 Haiku API calls at ~$0.001-0.003 each). N configurable via `--test-runs`.

**What gets tested:**
- New skill candidates (from embedding clusters)
- Refined skills (after LLM rewrite)
- Merged/split skills
- CLAUDE.md promotion candidates (highest bar)

**What doesn't get tested (safe operations):**
- Demotions (content moves down — still accessible)
- Pruning (below confidence threshold, validated by decay)
- Embedding storage (append-only)

## Visibility & Traceability

Three layers, from least to most effort to consume.

### Layer 1: Session-end summary (automatic, zero effort)

Printed by Stop hook after extraction:

```
── Learning Summary ──────────────────────
Extracted:
  • correction: "Use AI-Used trailer, not Co-Authored-By" (confidence: 1.0)
  • correction: "targ check, not mage check" (supersedes existing, decayed old entry)
  • pattern: "chi router middleware ordering matters for auth" (confidence: 0.7)

Retrievals: 14 this session (12 relevant, 2 filtered by similarity threshold)

Pending optimization:
  • skill candidate: 4 embeddings about "Go test tags" clustering — run `memory optimize` to compile
  • CLAUDE.md demotion candidate: "Use make([]T, 0, capacity)" is narrow — would move to embedding
  • skill refinement: "tdd-red-producer" retrieved 3x but correction followed twice
──────────────────────────────────────────
```

Content-rich bullet points. Actual learnings, actual reasons. Not just counters.

### Layer 2: CLI digest (`projctl memory digest`, on-demand)

When curious or something feels off. Shows:

- **Recent learnings:** What was extracted in last N sessions, with confidence and source
- **Skill changes:** Skills created/refined/merged/demoted since last digest, with test results
- **CLAUDE.md changes:** What was promoted/demoted/pruned, and where content went
- **Metrics snapshot:** Correction recurrence rate, retrieval precision, hook violation trend
- **Flags:** Things that look wrong (skill never retrieved, same correction 3+ times, retrieval precision dropping)

Filterable: `--since 7d`, `--tier skills`, `--flags-only`.

### Layer 3: Data files (append-only, for future visualization)

All in `~/.claude/memory/`:

| File | What it captures | Each entry contains |
|---|---|---|
| `changelog.jsonl` | Every mutation to any tier | timestamp, action, source tier, destination tier, content summary, reason, session ID |
| `retrievals.jsonl` | Every memory retrieval | timestamp, query, results returned, scores, tier, triggering hook, session ID |
| `metrics.jsonl` | Periodic snapshots | timestamp, correction_recurrence_rate, retrieval_precision, hook_violation_trend, embedding_count, skill_count, claude_md_lines |

These accumulate rich time-series data for future `/playground` visualization or dashboard.

**Traceability story:** "Why did Claude just say X?" → Check `retrievals.jsonl` (what was injected?) → Check `changelog.jsonl` (where did it come from?) → Trace to original session transcript.

**Correction verification story:** Correct Claude → session-end summary shows correction extracted → next `digest` shows it stored at confidence 1.0 → retrieval logs show it injected when relevant → metrics show correction recurrence = 0 for that topic.

## Gap Analysis: What Exists vs. What Needs Building

### Exists and fine (no changes needed): 11 components
- Session extraction pipeline (Stop/PreCompact hooks)
- Tier A/B/C learning classification
- Contradiction detection
- LLM enrichment via DirectAPIExtractor
- Embeddings storage with vector + FTS5
- Archive (append-only audit trail)
- Hybrid search (vector + BM25 + RRF)
- Hook-driven retrieval (SessionStart, UserPromptSubmit, PreToolUse)
- Decay, dedup, prune operations
- Skill split/merge/demotion operations
- Problem surfacing (recurring hook failures)

### Exists but needs wiring: 4 components
- **Skill test gate:** Skill compilation/refinement/merge/promotion exists but deploys without testing. Wire auto-test before deploy.
- **CLAUDE.md safe demotion:** Demotion exists but doesn't guarantee content lands in a lower tier first. Add create-destination-then-remove invariant.
- **LLM synthesis as default:** `generatePattern()` is mechanical. Make LLM path default with quality validation (actionable, specific, non-redundant).
- **Problem surfacing trends:** Exists as point-in-time detection. Add time-series tracking for trend analysis.

### New to build: 8 components
- **Changelog logging** (`changelog.jsonl`): Instrument every mutation across all tiers
- **Retrieval logging** (`retrievals.jsonl`): Instrument every hook-triggered retrieval
- **Correction recurrence tracking:** Detect repeated corrections despite existing learnings
- **Retrieval relevance scoring:** Infer from correction-after-injection pattern
- **Metrics snapshots** (`metrics.jsonl`): Periodic composite metric snapshots
- **Skill test harness:** Scenario derivation, N=3 runs, direct API, deterministic judging
- **Session-end summary:** Content-rich bullet points printed after extraction
- **`projctl memory digest` command:** CLI view of changelog, metrics, flags
- **Similarity threshold filtering:** Query-time threshold (0.7 default), return empty below threshold
- **"Remember X" detection:** Explicit pattern matching for remember/always/never signals

## Design Decisions

1. **Optimize is manual.** Extraction is automatic (safe, append-only). Optimization mutates state. Keep it manual via `projctl memory optimize`. Session-end summary shows what's pending so you know when it's worth running.

2. **No decay floor on "remember" items.** "Remember to use mage" should decay when "remember to use targ" supersedes it. Tier A confidence on ingestion, normal lifecycle thereafter.

3. **Skill tests use direct API, not subprocesses.** No Claude Code process, no plugins, no hooks, no memory retrievals during tests. Clean room evaluation. Prevents test contamination.

4. **Multiple test runs (N=3 default).** LLMs are stochastic. One run isn't enough. 3 runs with >=2 agreement required for deploy. Configurable via `--test-runs`.

5. **Global scope from the start.** Hooks are already global. Learning applies across all projects. Cross-project patterns (like TDD discipline) should propagate everywhere.

6. **CLAUDE.md is a view, not a store.** Demotion never deletes — content moves to skill or embedding first. Budget constraint forces prioritization, not information loss.

7. **Data layer built for future visualization.** JSONL files accumulate time-series data. Today: grep/jq. Tomorrow: `/playground` dashboard or web UI.
