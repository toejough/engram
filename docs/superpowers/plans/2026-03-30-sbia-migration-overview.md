# SBIA Migration Overview

> **Source spec:** `docs/superpowers/specs/2026-03-29-sbia-feedback-model-design.md`

**Goal:** Incrementally migrate engram from the current keyword/principle memory model to the SBIA (Situation, Behavior, Impact, Action) feedback model, keeping engram functional end-to-end at every step.

**Constraint:** Every step delivers a working system. Hooks are updated alongside the code they call — no step removes a command without updating the hooks that invoke it. "Temporarily broken" is not acceptable.

---

## Migration Steps

### Step 1: Schema + Migration ✅

- [x] Convert MemoryRecord to SBIA fields (situation, behavior, impact, action, project_scoped, project_slug)
- [x] Replace old tracking counters with SBIA counters (surfaced_count, followed_count, not_followed_count, irrelevant_count)
- [x] Add pending_evaluations support to TOML reader/writer
- [x] Run Sonnet migration: convert tier A memories to SBIA, archive tiers B and C
- [x] Remove old fields from struct and TOML writer
- [x] Update `engram show` to display SBIA fields
- [x] SearchText updated to concatenate SBIA fields
- **After:** Memory files are SBIA-only. `engram show` works. Surface BM25 works on SBIA fields.
- **Known breakage (to be fixed in Step 2):** `correct` stubbed (no new memories), `feedback` command removed (stop hook injects stale instructions), `flush` command removed (`stop.sh` async hook silently fails).

**Plan:** `docs/superpowers/plans/2026-03-30-sbia-step1-schema-migration.md`

### Step 2: Extract Pipeline + System Restoration ✅

**Core — build the SBIA extract pipeline:**
- [x] Add SBIA strip mode to context.Strip (StripConfig parameter)
- [x] Add policy.toml schema ([parameters] + [prompts] sections)
- [x] Rewrite `engram correct` to: detect (fast-path + Haiku) → context (SBIA strip) → BM25 candidates → Sonnet extraction + dedup decision tree → write
- [x] Add disposition handling (STORE, DUPLICATE, CONTRADICTION, etc.)
- [x] Add `engram refine` command — run extraction pipeline retroactively on existing memories using original session transcripts

**System restoration — fix hooks broken by Step 1:**
- [x] Update surface.go feedback injection — remove "call `engram feedback`" text, replace with SBIA display format (show all 4 fields per memory, no self-report instruction)
- [x] Restore `engram feedback` as thin shim — accepts same flags, increments SBIA counters directly (followed/not_followed/irrelevant). ~30 lines. Placeholder until Step 4 replaces with automated Haiku evaluation.
- [x] Update `stop.sh` — replace `engram flush` call with no-op or remove. Async stop slot reserved for `engram evaluate` in Step 4.
- **After:** Correct creates new SBIA memories via Sonnet. Surface finds and displays them with full SBIA fields. Feedback shim records basic counters. Flush cleanly removed. Refine upgrades passthrough-migrated memories. **System works end-to-end.**

**Plan:** `docs/superpowers/plans/2026-03-31-sbia-step2-extract.md`

### Step 3: Surface Upgrades ✅

Surface already works (BM25 on SBIA fields). This step makes it smarter.

- [x] Add Haiku semantic gate (surface_gate_haiku prompt) — filters BM25 candidates by situation relevance
- [x] Add cold-start budget for unproven memories (max `surface_cold_start_budget` per invocation)
- [x] Add irrelevance penalty half-life to BM25 scoring (configurable via policy.toml)
- [x] Write pending_evaluations at surface time (tracking for Step 4)
- [x] Update display format via configurable surface_injection_preamble prompt
- [x] Update /recall to use SBIA display format
- **After:** Surfacing uses situation matching + Haiku gate. Pending evaluations tracked. Feedback shim still works for counter recording.

**Plan:** `docs/superpowers/plans/2026-03-31-sbia-step3-surface.md`

### Step 4: Evaluate

- [ ] Implement `engram evaluate` command
- [ ] Read pending_evaluations from memory TOMLs for current session
- [ ] Haiku assessment: situation relevance + action compliance
- [ ] Increment followed_count / not_followed_count / irrelevant_count
- [ ] Remove consumed pending_evaluation entries
- [ ] Update `stop.sh` async hook — call `engram evaluate` (replaces no-op from Step 2)
- [ ] Update `stop-surface.sh` / surface.go — remove "call `engram feedback`" instruction (no longer needed)
- [ ] Remove `engram feedback` shim (replaced by automated evaluation)
- [ ] Remove surfacing-log.jsonl, learn-offset.json if still present
- [ ] **After:** Memories accumulate effectiveness data automatically via Haiku. No LLM self-report. All hooks updated.

**Plan:** `docs/superpowers/plans/2026-03-31-sbia-step4-evaluate.md`

### Step 5: Maintain + Adapt + Triage

- [ ] Rewrite `engram maintain` to: effectiveness decision tree + consolidation analysis + Sonnet adapt analysis → unified proposals
- [ ] Implement `engram apply-proposal <id>` and `engram reject-proposal <id>`
- [ ] Add change_history to policy.toml
- [ ] Update /memory-triage skill for new proposal flow
- [ ] Update session-start.sh background maintain to use new proposal format
- [ ] Remove old quadrant analysis, signal packages, policy lifecycle, approval streaks
- [ ] **After:** Complete SBIA pipeline operational. All old code removed. All hooks updated.

**Plan:** `docs/superpowers/plans/2026-03-31-sbia-step5-maintain-adapt.md`

---

## Dependencies

```
Step 1 (Schema) ─→ Step 2 (Extract + Restore) ─→ Step 4 (Evaluate)
                ─→ Step 3 (Surface Upgrades)   ─→ Step 4 (Evaluate)
                                                  Step 4 ─→ Step 5 (Maintain)
```

Steps 2 and 3 can be developed in parallel after Step 1. Step 4 requires both 2 and 3. Step 5 requires Step 4.

---

## Invariant: Hooks Match Code

Every step that changes a command also updates every hook that calls it. Checklist for each step:

| Hook | Script | What it calls | Verify after each step |
|------|--------|---------------|----------------------|
| SessionStart | `session-start.sh` | `engram maintain` (async bg) | maintain works or is no-op |
| UserPromptSubmit | `user-prompt-submit.sh` | `engram correct`, `engram surface` | both commands functional |
| Stop (sync) | `stop-surface.sh` | `engram surface --mode stop` | surface works, injected text matches available commands |
| Stop (async) | `stop.sh` | Step 1: `flush` → Step 2: no-op → Step 4: `evaluate` | called command exists |
