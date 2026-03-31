# SBIA Migration Overview

> **Source spec:** `docs/superpowers/specs/2026-03-29-sbia-feedback-model-design.md`

**Goal:** Incrementally migrate engram from the current keyword/principle memory model to the SBIA (Situation, Behavior, Impact, Action) feedback model, keeping engram functional end-to-end at every step.

**Constraint:** Each step cuts over one pipeline stage completely, removing the old code path. No dual paths, no fallbacks, no deferred cleanup. Breakage between steps is visible and intentional, not silently masked.

---

## Migration Steps

### Step 1: Schema + Migration
- [ ] Convert MemoryRecord to SBIA fields (situation, behavior, impact, action, project_scoped, project_slug)
- [ ] Replace old tracking counters with SBIA counters (surfaced_count, followed_count, not_followed_count, irrelevant_count)
- [ ] Add pending_evaluations support to TOML reader/writer
- [ ] Run Sonnet migration: convert tier A memories to SBIA, archive tiers B and C
- [ ] Remove old fields from struct and TOML writer
- [ ] Update `engram show` to display SBIA fields
- [ ] **After:** Memory files are SBIA-only. `engram show` works. Extraction/surfacing/maintain are temporarily broken (they still reference old fields).

**Plan:** `docs/superpowers/plans/2026-03-30-sbia-step1-schema-migration.md`

### Step 2: Extract Pipeline
- [ ] Add SBIA strip mode to context.Strip (StripConfig parameter)
- [ ] Add policy.toml schema ([parameters] + [prompts] sections)
- [ ] Rewrite `engram correct` to: detect (fast-path + Haiku) → context (SBIA strip) → BM25 candidates → Sonnet extraction + dedup decision tree → write
- [ ] Add disposition handling (STORE, DUPLICATE, CONTRADICTION, etc.)
- [ ] Remove old extract, learn, flush, classify packages
- [ ] **After:** New memories are created via Sonnet with full SBIA fields. Old batch extraction gone.

**Plan:** `docs/superpowers/plans/2026-03-30-sbia-step2-extract.md`

### Step 3: Surface Pipeline
- [ ] Rewrite `engram surface` BM25 to use SBIA text (situation+behavior+impact+action)
- [ ] Add irrelevance penalty to BM25 scoring
- [ ] Add Haiku semantic gate (surface_gate_haiku prompt)
- [ ] Add cold-start budget for unproven memories
- [ ] Update display format to show all 4 SBIA fields via surface_injection_preamble
- [ ] Write pending_evaluations at surface time (tracking step)
- [ ] Remove old SearchText, token budget, principle-only display
- [ ] Update /recall SearchText for SBIA fields
- [ ] **After:** Surfacing uses situation matching + Haiku gate. Full SBIA display.

**Plan:** `docs/superpowers/plans/2026-03-30-sbia-step3-surface.md`

### Step 4: Evaluate
- [ ] Implement `engram evaluate` command
- [ ] Read pending_evaluations from memory TOMLs for current session
- [ ] Haiku assessment: situation relevance + action compliance
- [ ] Increment followed_count / not_followed_count / irrelevant_count
- [ ] Remove consumed pending_evaluation entries
- [ ] Wire into stop.sh async hook (replacing old flush/learn)
- [ ] Remove surfacing-log.jsonl, learn-offset.json
- [ ] **After:** Memories accumulate effectiveness data automatically. No LLM self-report needed.

**Plan:** `docs/superpowers/plans/2026-03-30-sbia-step4-evaluate.md`

### Step 5: Maintain + Adapt + Triage
- [ ] Rewrite `engram maintain` to: effectiveness decision tree + consolidation analysis + Sonnet adapt analysis → unified proposals
- [ ] Implement `engram apply-proposal <id>` and `engram reject-proposal <id>`
- [ ] Add change_history to policy.toml
- [ ] Update /memory-triage skill for new proposal flow
- [ ] Remove old quadrant analysis, signal packages, policy lifecycle, approval streaks
- [ ] **After:** Complete SBIA pipeline operational. All old code removed.

**Plan:** `docs/superpowers/plans/2026-03-30-sbia-step5-maintain-adapt.md`

---

## Dependencies

```
Step 1 (Schema) ─→ Step 2 (Extract) ─→ Step 4 (Evaluate)
                ─→ Step 3 (Surface) ─→ Step 4 (Evaluate)
                                       Step 4 ─→ Step 5 (Maintain)
```

Steps 2 and 3 can be developed in parallel after Step 1. Step 4 requires both 2 and 3. Step 5 requires Step 4.
