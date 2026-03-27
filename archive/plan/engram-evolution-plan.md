# Engram Evolution Plan

Parallelizable, trackable plan for evolving engram from a memory store into a self-correcting memory system with graph structure, cross-source awareness, and aggressive context optimization.

**Source:** `docs/prompts/engram-evolution-orchestration.md`
**Created:** 2026-03-10

---

## Dependency Graph

```
Phase A-1: Simplification (all 6 items parallel)
  S1 ─┐
  S2 ─┤
  S3 ─┼──→ Phase A-2: Foundation
  S4 ─┤
  S5 ─┤
  S6 ─┘

Phase A-2: Foundation (3 parallel streams)
  P0a (enforcement_level)  ←── S2, S4
  P0c (crossref scanner)   ←── S3
  P2  (precompact reinjection) ←── nothing (independent)

  P0b (transition tracking) ←── P0a

Phase A-3: High-Impact (3 parallel streams)
  P1      (contradiction)       ←── P0c
  P4-early (budget quick wins)  ←── P0a
  P6-early (wire escalation)    ←── P0a, S2

Phase B-1: Graph + Evolution (3 parallel streams)
  P3      (memory graph)        ←── P0a
  P5-core (merge logic)         ←── P0a
  P6-full (graduation signals)  ←── P6-early

Phase B-2: Integration (2 parallel streams)
  P4-full (cluster dedup + cross-source suppression) ←── P3, P0c
  P5-full (link recompute after merge)               ←── P3, P5-core
```

---

## Work Items

### Phase A-1: Simplification

All items are independent — run all 6 in parallel. Each is a single atomic commit with test updates. Run `targ check` after each.

| ID | Description | Files Touched | AC |
|----|-------------|---------------|-----|
| **S1** | Remove `internal/automate/` package + `engram automate` CLI subcommand | `internal/automate/` (delete), `internal/cli/` (remove wiring), `hooks/` (if referenced) | Package deleted, CLI help doesn't list `automate`, `targ check` clean |
| **S2** | Remove `LevelPretoolBlock` + `LevelAutomationCandidate` from escalation ladder | `internal/maintain/escalation.go`, `internal/maintain/escalation_test.go` | Ladder is 3 levels: advisory → emphasized_advisory → reminder. `predictImpact` logic for removed levels gone. `targ check` clean |
| **S3** | Relocate non-memory extractors to `internal/crossref/` | `internal/registry/extract.go` → `internal/crossref/extract.go`, move tests, keep parsing logic intact | `internal/crossref/` exists with bullet extraction + rule parsing. Registry extract.go only handles memory source type. `targ check` clean |
| **S4** | Simplify registry to memory-only source type | `internal/registry/classify.go`, `internal/registry/entry.go` | `alwaysLoadedSources` map removed. `SourceType` only accepts "memory" for persisted entries. `registry merge` scoped to memory-to-memory only. `targ check` clean |
| **S5** | Replace promotion signal types with graduation | `internal/signal/detector.go`, `internal/signal/signal.go`, `internal/signal/detector_test.go` | `KindSkillToClaudeMD`, `KindClaudeMDDemotion` replaced with `KindGraduation`. Signal carries recommendation text, not target. `targ check` clean |
| **S6** | Descope `internal/instruct/` to memory-only quality audit | `internal/instruct/audit.go`, `internal/instruct/scanner.go`, tests | Cross-source dedup removed (moves to surface pipeline in P4-full). Audit only examines memories. `targ check` clean |

**Parallel strategy:** 6 teammates, one per S-item. No shared files between S1-S6 (verified: automate, maintain, registry, signal, instruct are separate packages).

---

### Phase A-2: Foundation

Three parallel streams after A-1 completes. P0b is sequential after P0a.

| ID | Description | Depends On | Files Touched | AC |
|----|-------------|------------|---------------|-----|
| **P0a** | Add `enforcement_level` field to registry entries | S2, S4 | `internal/registry/entry.go`, `internal/registry/jsonl_store.go`, tests | `InstructionEntry` has `EnforcementLevel` field (advisory/emphasized_advisory/reminder/graduated). Default: advisory. Persisted in JSONL. Backfill existing entries. `targ check` clean |
| **P0b** | Track enforcement level transitions with timestamps | P0a | `internal/registry/entry.go`, `internal/registry/jsonl_store.go`, tests | `EnforcementTransition` struct: `{From, To, At, Reason}`. `InstructionEntry.Transitions []EnforcementTransition`. Registry records transitions on level change. `targ check` clean |
| **P0c** | Build `internal/crossref/` Scanner | S3 | `internal/crossref/scanner.go`, `internal/crossref/index.go`, tests | `Scanner.Scan(paths)` reads CLAUDE.md + rules + skills. Returns `[]CrossRefEntry{Source, ContentHash, Keywords, PrincipleText}`. Index rebuilt per session (ephemeral, not persisted). Uses relocated parsing logic from S3. `targ check` clean |
| **P2** | PreCompact memory re-injection | — | `internal/surface/`, `hooks/pre-compact.sh`, `internal/cli/` | New surface mode `precompact`: ranks by effectiveness only (no BM25, no query), top-5 within 500 token budget, skips effectiveness < 40%. Format: concise principle statements only, not full memory content. Hook calls `engram surface --mode precompact --budget 500`. Output: `[engram] Preserving top memories through compaction:\n- <principle>...`. `targ check` clean |

**Parallel strategy:** 3 teammates: (P0a then P0b), P0c, P2. No file conflicts.

---

### Phase A-3: High-Impact Fixes

Three parallel streams after A-2 completes.

| ID | Description | Depends On | Files Touched | AC |
|----|-------------|------------|---------------|-----|
| **P1** | Contradiction detection (read-only cross-source) | P0c | New `internal/contradict/` package, wire into `internal/surface/` | `Detector.Check(memory, crossRefIndex)` returns contradiction signals. Two-pass detection: (1) keyword heuristic (same subject, opposing verbs) + BM25 between memory principle and cross-ref entries, then (2) Haiku classifier for ambiguous cases that pass heuristic but need judgment (max 3 calls/surface). Contradicting memories suppressed at surface time. Also checks memory-vs-memory contradictions within top-N selection. Signal logged: `{type: "contradiction", memory, contradicts, recommendation}`. New signal kind `KindContradiction`. Contradiction signals surfaced in SessionStart alongside other pending signals, and via `engram review`. Per-session contradiction count derived from signal queue: count signals where `kind=contradiction AND created_at >= session_start` (no new logging needed — signals already have timestamps). `targ check` clean |
| **P4-early** | Budget quick wins: effectiveness gating + BM25 floor | P0a | `internal/surface/`, `internal/hooks/` | SessionStart: rank by effectiveness (not frecency), select top-7 with effectiveness > 40% or insufficient data (< 5 surfacings), target 600 tokens (from ~1100). UserPromptSubmit: raise BM25 floor, target 250 tokens (from ~300). PreToolUse: top-2 (not 5), effectiveness floor 40%, target 150 tokens (from ~350). PostToolUse: already capped at ~100 tokens. Overall session target: ~18,000 tokens (from ~24,700). Each surface invocation records its output token count on the existing surfacing event log. Stop hook sums token counts across the session and logs the total. `targ check` clean |
| **P6-early** | Wire escalation to registry enforcement_level | P0a, S2 | `internal/maintain/escalation.go`, `internal/registry/`, `internal/signal/` | Escalation proposals update `enforcement_level` field in registry. 3-level ladder + graduation: advisory → emphasized_advisory → reminder → graduated. Surface pipeline respects `enforcement_level`: advisory = normal format; emphasized_advisory = "IMPORTANT:" prefix + bold principle, higher budget priority; reminder = generate PostToolUse reminder pattern for memory's file glob patterns and add to remind configuration. Graduation emits `KindGraduation` signal with recommendation (mechanical → settings.json; file-scoped → rule; behavioral → CLAUDE.md; procedural → skill). `targ check` clean |

**Note:** The orchestration prompt describes Packages 4, 5, and 6 as single units. This plan splits each into early/full phases to enable finer-grained parallelism and clearer dependency tracking. P4-early = budget quick wins (gating + floors); P4-full = cluster dedup + cross-source + transcript suppression. P5-core = merge-on-write logic; P5-full = link recompute after merge. P6-early = wire escalation to enforcement_level; P6-full = graduation signals + de-escalation. P6-early is moved from the prompt's Phase B to A-3 — its dependencies (P0a, S2) are satisfied by A-2, and early completion unblocks P6-full sooner.

**Parallel strategy:** 3 teammates. P1 touches contradict + surface. P4-early touches surface config/thresholds (different code paths than P1's post-ranking filter). P6-early touches maintain + registry + signal. Minimal overlap — P1 and P4-early both touch surface, so assign to same teammate or sequence them.

**Revised parallel strategy:** 2 teammates: (P1 then P4-early — both touch surface), P6-early.

---

### Phase B-1: Graph + Evolution Core

Three parallel streams after A-3 completes.

| ID | Description | Depends On | Files Touched | AC |
|----|-------------|------------|---------------|-----|
| **P3** | Memory graph with spreading activation | P0a | New `internal/graph/` package, extend `internal/frecency/`, extend `internal/registry/entry.go` | `Links []Link` field on `InstructionEntry`: `{Target, Weight, Basis}`. Link building: concept Jaccard (at learn time), co-surfacing (at surface time), evaluation correlation (at evaluate time), BM25 content similarity (at learn time). Spreading activation: `total = base + 0.3 × Σ(linked.base × weight)`. Surfacing: linked memories above threshold get cluster notes `(Related: <principle>)`, max 2 per surfaced memory, ~20 tokens each. Link pruning: weight < 0.1 after 10+ co-surfacing opportunities. `targ check` clean |
| **P5-core** | Merge-on-write logic | P0a | `internal/dedup/`, `internal/learn/`, `internal/registry/` | When learn detects >50% keyword overlap: merge instead of skip. LLM-assisted merge (Haiku) combines principles. Fallback: keyword union + longer principle. Registry records merge in `Absorbed` field. Preserves effectiveness history. `targ check` clean |
| **P6-full** | Graduation signals with specific recommendations | P6-early | `internal/maintain/`, `internal/signal/`, `internal/cli/` | Graduation signal includes content-based recommendation: mechanical rule → settings.json deny or linter; file-scoped → .claude/rules/; behavioral → CLAUDE.md; procedural → skill. De-escalation: if effectiveness improves post-escalation, propose de-escalation. SessionStart surfaces pending graduation signals in `additionalContext` with an instruction for the LLM to ask the user whether to create a GitHub issue (e.g., "Ask the user if they'd like to create an issue to track this. If yes, run `engram graduate accept <id>`. If no, run `engram graduate dismiss <id>`."). Signal persists across sessions until accepted or dismissed — if the LLM drops it, it re-surfaces next session. `engram graduate accept <id>` creates a GitHub issue with the recommendation details and records `accepted`. `engram graduate dismiss <id>` records `dismissed`. Metric: `accepted / (accepted + dismissed)`. `targ check` clean |

**Parallel strategy:** 3 teammates. P3 (graph + frecency), P5-core (dedup + learn), P6-full (maintain + signal + review). No file conflicts.

---

### Phase B-2: Integration

Two parallel streams after B-1 completes.

| ID | Description | Depends On | Files Touched | AC |
|----|-------------|------------|---------------|-----|
| **P4-full** | Cluster dedup + cross-source suppression + transcript suppression | P3, P0c | `internal/surface/` | Cluster dedup: when two linked memories would both surface, keep highest-effectiveness + cluster note. Cross-source suppression: if memory principle covered by CLAUDE.md/rule/skill (via crossref index), skip it — higher-salience source already delivers it. Transcript suppression: if principle keywords appear in recent ~500 token transcript (which includes ALL hooks' output), skip — this is the practical mechanism for deduplicating against other plugins' hook output, since hooks can't observe each other in real-time. Total target: ~13,500 tokens/session (45% reduction from ~24,700). Each suppression decision logged on the surface event log: `{memory_id, reason: "cross_source" | "cluster_dedup" | "transcript", suppressed_by}`. Suppression rate = `suppressions / (surfaced + suppressed)`. Also serves debugging: "why didn't memory X surface?" `targ check` clean |
| **P5-full** | Re-compute links after merge | P3, P5-core | `internal/graph/`, `internal/learn/` | After merge-on-write, re-compute concept overlap links for the merged memory. Remove stale links to absorbed source. `targ check` clean |

**Parallel strategy:** 2 teammates. P4-full (surface), P5-full (graph + learn). No file conflicts.

---

## Phase Summary

| Phase | Items | Max Parallelism | Blocking Dependencies | Estimated Commits |
|-------|-------|-----------------|----------------------|-------------------|
| A-1 | S1-S6 | 6 | None (entry point) | 6 |
| A-2 | P0a, P0b, P0c, P2 | 3 | A-1 complete | 4 |
| A-3 | P1, P4-early, P6-early | 2-3 | P0a, P0c | 3 |
| B-1 | P3, P5-core, P6-full | 3 | P0a, P6-early | 3 |
| B-2 | P4-full, P5-full | 2 | P3, P0c, P5-core | 2 |
| **Total** | **18 items** | | | **~18 commits** |

---

## Spec Process Integration

Each work item becomes a UC or extends an existing UC through the traced spec process:

| Item | UC Strategy |
|------|-------------|
| S1-S6 | No new UCs — simplification of existing UC-21, UC-22, UC-23, UC-26, UC-28 |
| P0a-P0b | Extend UC-23 (registry) |
| P0c | New UC-29: Cross-Source Scanner |
| P1 | New UC-30: Contradiction Detection |
| P2 | Extend UC-17 (Context Budget) or new UC-31: PreCompact Re-Injection |
| P3 | New UC-32: Memory Graph & Spreading Activation |
| P4-early | Extend UC-17 (Context Budget) |
| P4-full | Extend UC-17 (Context Budget) |
| P5-core | New UC-33: Memory Evolution (Merge-on-Write) |
| P5-full | Extend UC-33 |
| P6-early | Extend UC-21 (Enforcement Escalation) |
| P6-full | Extend UC-21 (Enforcement Escalation) |

---

## Context Budget Targets

Per-hook token budget (from orchestration prompt). P4-early implements gating/floors; P4-full implements cluster dedup + suppression.

| Hook | Current | Target | Strategy | Owner |
|------|---------|--------|----------|-------|
| SessionStart | ~1,100 | 600 | Top-7 by effectiveness (not top-20 by frecency), cluster dedup, cross-source suppression | P4-early (gating), P4-full (dedup + suppression) |
| UserPromptSubmit | ~300 | 250 | Raise BM25 floor, cluster dedup | P4-early |
| PreToolUse | ~350 | 150 | Top-2 (not 5), effectiveness floor 40%, transcript suppression | P4-early (top-2 + floor), P4-full (transcript suppression) |
| PostToolUse | ~100 | 100 | Already capped | — |
| PreCompact | 0 (new) | 500 | Top-5 by effectiveness | P2 |
| **Total/session** | **~24,700** | **~13,500** | **45% reduction** | |

---

## Success Metrics

Track after each phase gate:

| Metric | Baseline | Phase A Target | Phase B Target |
|--------|----------|----------------|----------------|
| Memory compliance rate | ~60% | >70% | >80% |
| Context tokens/session | ~24,700 | ~18,000 | <13,500 |
| Contradictions detected/session | 0 | Trending to 0 (after initial spike) | Near 0 |
| Memories with >=1 link | 0% | 0% (graph not built yet) | >60% |
| Cross-source suppression rate | 0% | >20% | >40% |
| Graduation signal acceptance rate | N/A | N/A (no graduations yet) | >50% accepted via `engram graduate accept/dismiss` |

---

## Execution Checklist

- [ ] **A-1: S1** — Remove `internal/automate/`
- [ ] **A-1: S2** — Remove top 2 escalation levels
- [ ] **A-1: S3** — Relocate extractors to `internal/crossref/`
- [ ] **A-1: S4** — Simplify registry to memory-only
- [ ] **A-1: S5** — Replace promotion signals with graduation
- [ ] **A-1: S6** — Descope instruct to memory-only
- [ ] **A-2: P0a** — Add enforcement_level to registry
- [ ] **A-2: P0b** — Track enforcement level transitions
- [ ] **A-2: P0c** — Build crossref scanner
- [ ] **A-2: P2** — PreCompact re-injection
- [ ] **A-3: P1** — Contradiction detection
- [ ] **A-3: P4-early** — Budget quick wins
- [ ] **A-3: P6-early** — Wire escalation to enforcement_level
- [ ] **B-1: P3** — Memory graph + spreading activation
- [ ] **B-1: P5-core** — Merge-on-write logic
- [ ] **B-1: P6-full** — Graduation signals
- [ ] **B-2: P4-full** — Cluster dedup + cross-source suppression
- [ ] **B-2: P5-full** — Link recompute after merge

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| S3 relocating extractors breaks imports across codebase | Blocks P0c | Grep all imports of `internal/registry/extract` before moving. Update in same commit. |
| P1 contradiction detection has high false-positive rate | Wasted tokens on suppression signals | Start with high-confidence only (exact keyword match + opposing verbs). Haiku classifier as second pass. Tune threshold with real data. |
| P3 spreading activation creates runaway boosting | Irrelevant memories surfaced via transitive links | Cap spreading depth at 1 hop. Decay factor 0.3. Link pruning removes weak edges. |
| P4 aggressive budget cuts suppress important memories | Compliance rate drops | Gate behind effectiveness data — only suppress memories with sufficient data (5+ surfacings). Keep "insufficient data" memories at current behavior. |
| P5 merge-on-write loses nuance from original memories | Information loss | Record full absorbed history. Merge only on >50% keyword overlap + LLM confirmation. Keep original as absorbed record. |
