# Optimize LLM Eval Pipeline Design

## Goal

Replace the current mechanical-only optimize flow (scanner proposals → human review) with a three-stage pipeline: mechanical scanners → LLM evaluation → human review. Move purely mechanical operations (prune/decay) to automatic stop hook execution.

## Architecture

### Stop Hook: Auto-Maintenance

Prune and decay run automatically at every session end. No prompting, no LLM.

- **Prune:** Delete embeddings with confidence < 0.3
- **Decay:** Multiply confidence by 0.5 for entries >90 days old with <5 retrievals
- **Output:** One-line summary if work done (`Memory maintenance: pruned 2, decayed 5`), silent if nothing to do
- **No backups or transactions** — prune removes noise, decay is reversible through retrievals

### Optimize --review: LLM Eval Pipeline

```
Mechanical Scanners (consolidate, promote, demote, split only)
    ↓
Haiku Triage (parallel)
    ↓
Sonnet Behavioral Testing (per survivor)
    ↓
Human Review
```

#### Stage 1: Mechanical Scanners

Same as today, minus prune/decay (moved to stop hook):

| Action | Tier | Trigger |
|--------|------|---------|
| consolidate | embeddings | similarity > 0.92 |
| consolidate | skills | similarity > 0.85 |
| consolidate | claude-md | similarity > 0.9 |
| promote | embeddings → skills | retrieval >= 10, confidence >= 0.8, 3+ projects |
| promote | skills → claude-md | utility >= 0.8, 3+ projects |
| demote | claude-md → skills | domain-specific keywords |
| split | claude-md | >100 tokens |

Refinement proposals (rewrite, add-rationale) are already LLM-generated and skip triage/behavioral testing — shown directly in human review.

#### Stage 2: Haiku Triage

Parallel evaluation of all judgment-call proposals. Drops false positives before they reach Sonnet or the user.

**Concurrency:** `runtime.NumCPU()` semaphore with exponential backoff retry (same pattern as batch extraction).

**Input per proposal:**
```json
{
  "action": "consolidate",
  "reason": "similarity 0.92",
  "entry_a": "<full text of entry A>",
  "entry_b": "<full text of entry B>"
}
```

**System prompt:** "You are evaluating whether a maintenance proposal is valid. Judge the actual content, not the mechanical signal. Two entries can share vocabulary but teach different lessons."

**Output:**
```json
{
  "valid": true,
  "rationale": "These entries address the same concept from different angles but the kept entry covers both."
}
```

- `valid: false` → proposal logged and dropped, never shown to user
- `valid: true` → proposal moves to Sonnet with Haiku's rationale attached

**Model:** `claude-haiku-4-5-20251001` (fast, cheap, well-defined judgment task)

#### Stage 3: Sonnet Behavioral Testing

Sequential evaluation of proposals that survived Haiku triage. Tests the behavioral impact of each change against the full memory ecosystem.

**Three steps per proposal:**

**Step 1 — Identify intent and preservation requirements:**
```json
{
  "change_intent": "Merge two agent coordination memories into one",
  "expected_change": "Single consolidated entry covers both delegation and polling",
  "must_preserve": [
    "Guidance on delegating authority to coordinators",
    "Guidance on agents self-serving from task lists"
  ]
}
```

**Step 2 — Generate 2-3 test scenarios:**

LLM-generated scenarios that mirror real user prompts:
```json
{
  "scenarios": [
    {
      "user_prompt": "How should I structure my multi-agent team?",
      "should_surface": "delegation authority to coordinator",
      "type": "preserve"
    },
    {
      "user_prompt": "My agents are idle waiting for instructions",
      "should_surface": "self-serve from task lists, active polling",
      "type": "preserve"
    }
  ]
}
```

**Step 3 — Simulate before/after:**

For each scenario, Sonnet receives the assembled context window (CLAUDE.md + relevant skills + relevant memories) as it would look before and after the proposed change. It checks whether expected guidance surfaces in each version.

**Output:**
```json
{
  "recommend": "skip",
  "confidence": "high",
  "change_analysis": "Consolidation would lose specific polling advice",
  "preservation_report": [
    {"scenario": "team structure", "preserved": true},
    {"scenario": "idle agents", "preserved": false, "lost": "explicit polling instruction"}
  ]
}
```

**Context assembly:** Build the prompt that *would* result from the full pipeline — retrieved memories + matched skills + CLAUDE.md — without actually running retrieval. This isolates content quality testing from retrieval bugs.

**Model:** `claude-sonnet-4-5-20250929` (nuanced quality judgment)

#### Stage 4: Human Review

The user sees the complete picture with consistent language:

```
━━━ Proposed Change: Delete duplicate memory ━━━

  What changes:
    DELETE: "When using multi-agent teams, explicitly instruct agents
            to self-serve from task lists..."
    KEEP:   "When managing teams of agents with task dependencies,
            assign full authority to a designated coordinator..."

  Why proposed: 0.92 similarity (mechanical threshold: 0.92)

  Haiku: Valid concern — entries share vocabulary, but they teach
         different lessons (delegation vs polling behavior).

  Sonnet recommends: Skip this change
    ✓ "team structure" → delegation guidance still surfaces
    ✗ "idle agents" → loses explicit polling instruction
    Summary: These look similar but the deleted entry contains
             actionable advice not present in the kept entry.

  [a]pply change / [s]kip change?
```

**Language alignment rules:**
- "Proposed Change" — states what will happen
- "What changes" — concrete before/after in DELETE/KEEP/PROMOTE/DEMOTE terms
- Haiku uses "valid/invalid concern"
- Sonnet uses "apply/skip" — same verbs as the user prompt
- User prompt is `[a]pply / [s]kip` — action-oriented, not yes/no

### CLI Interface

```
projctl memory optimize --review          # Full pipeline (mechanical + LLM + human)
projctl memory optimize --review --no-llm-eval  # Mechanical + human only (like today)
```

### Proposal Flow Summary

| Action | Where | Haiku Triage | Sonnet Behavioral | Human Review |
|--------|-------|-------------|-------------------|-------------|
| prune | Stop hook | No | No | No |
| decay | Stop hook | No | No | No |
| consolidate | optimize | Yes | Yes | Yes |
| promote | optimize | Yes | Yes | Yes |
| demote | optimize | Yes | Yes | Yes |
| split | optimize | Yes | Yes | Yes |
| rewrite | optimize | No (already LLM) | No | Yes |
| add-rationale | optimize | No (already LLM) | No | Yes |

## Research Synthesis Alignment

This design follows the research synthesis findings:

- **Content quality > mechanical sophistication:** LLM eval replaces blind threshold trust with semantic judgment
- **Model selection by tier:** Haiku for fast triage, Sonnet for quality judgment (Section 3.2)
- **Enforcement hierarchy:** Stop hook (automatic) > optimize review (manual) matches hooks > CLAUDE.md > skills
- **Ecosystem behavioral testing:** Tests the full stack (CLAUDE.md + skills + embeddings) not just individual entries
- **"Can't optimize what we don't measure":** Behavioral testing provides concrete evidence for each transition
