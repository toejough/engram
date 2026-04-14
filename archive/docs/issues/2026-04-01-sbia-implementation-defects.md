# SBIA Implementation Defects

**Date:** 2026-04-01
**Audited against:** `docs/superpowers/specs/2026-03-29-sbia-feedback-model-design.md`

## Critical

### 1. Extraction prompt missing SBIA decision tree — [#457](https://github.com/toejough/engram/issues/457)

**File:** `internal/policy/policy.go:298-316`

The prompt uses binary `is_new`/`duplicate_of` instead of the spec's 8-way disposition tree (STORE, DUPLICATE, CONTRADICTION, REFINEMENT, IMPACT UPDATE, POTENTIAL GENERALIZATION, LEGITIMATE SEPARATE MEMORIES, STORE BOTH). Per-candidate disposition reasoning is also missing.

**Spec reference:** Lines 203-259 (Decision: Sonnet-Driven Dedup via SBIA Decision Tree)

### 2. Refine command uses wrong prompt — [#458](https://github.com/toejough/engram/issues/458)

**File:** `internal/cli/refine.go:251`

Refine reuses `pol.ExtractSonnetPrompt` which says "find new memories" instead of "rewrite existing memory's SBIA fields." This caused 46 memories to be incorrectly refined on 2026-04-01.

**Missing:** No `RefineSonnetPrompt` exists in `internal/policy/policy.go`. Refine needs a dedicated prompt: "Given this existing memory and its original transcript, rewrite the SBIA fields to be clearer and more specific."

### 3. 215 memories still contain Keywords blobs — [#459](https://github.com/toejough/engram/issues/459)

**Location:** `~/.claude/engram/data/memories/`

Situation fields contain `\nKeywords: foo, bar` suffixes from the old schema. These pollute BM25 retrieval.

**Spec reference:** Line 151: "**Keywords dropped.** With SBIA text available for both retrieval and dedup via BM25, the keywords array no longer serves a unique purpose." Line 69: "BM25 retrieval matches against the full SBIA text, so no separate keyword array is needed."

### 4. 46 memories incorrectly refined on 2026-04-01 — [#460](https://github.com/toejough/engram/issues/460)

**Cause:** Defect #2 above. The extraction prompt treated each memory's `action` field as a "correction message" and re-extracted from the transcript, instead of rewriting the existing SBIA fields. Damage assessment and recovery strategy needed.

**Spec reference:** Lines 119-155 (Decision: Corrections-Only Extraction with Sonnet) — extraction is designed for the `correct` flow where a user provides a correction message and Sonnet reconstructs SBIA from the surrounding conversation. Refine is not part of the spec; it was added separately to batch-reprocess existing memories. The extraction prompt is architecturally wrong for that purpose.

## Medium

### 5. Extraction prompt missing candidate decision instructions — merged into [#457](https://github.com/toejough/engram/issues/457)

**File:** `internal/policy/policy.go:298-316`

No instruction for HOW Sonnet should evaluate candidates against the SBIA decision tree. The prompt should include the decision walkthrough (same situation? same behavior? same impact?) from spec lines 230-259.

## Low

### 6. Filename slug handling unclear — [#461](https://github.com/toejough/engram/issues/461)

**File:** `internal/memory/record.go`

`filename_slug` is in the extraction output schema but not stored in `MemoryRecord`. Confirm whether it's transient (used only for naming the file) or should be stored.

### 7. Surface preamble says "call engram show for full details" — [#462](https://github.com/toejough/engram/issues/462)

**File:** `internal/policy/policy.go:344`

But all 4 SBIA fields are already displayed inline. The preamble is misleading since full details are already shown.

### 8. Refine action-empty check ordering — [#463](https://github.com/toejough/engram/issues/463)

**File:** `internal/cli/refine.go:237`

Checks action emptiness after the already-refined guard. Should happen before the extraction attempt to avoid unnecessary LLM calls.

## Passing

- **MemoryRecord schema:** Correct SBIA fields, old fields removed
- **Surface display:** All 4 SBIA fields shown correctly per spec
- **Evaluate hook:** Properly wired to stop.sh for pending evaluation processing
