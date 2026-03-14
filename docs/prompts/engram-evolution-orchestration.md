# Engram Evolution: Orchestration Prompt

You are orchestrating a project to make engram — a self-correcting memory system for LLM agents — dramatically more effective at its topline goals:

1. **Reduce the incidence and severity of violations of ANY instructions the LLM is supposed to follow**
2. **Reduce the incidence and severity of forgetting important context in general**
3. **Get the work done in the most token and time-efficient ways possible**

This is not a greenfield build. Engram is a working Claude Code plugin with 40k lines of Go, 17 CLI subcommands, 6 hook integration points, and a comprehensive specification tree. Your job is to evolve it based on hard-won lessons from research, competitive analysis, and our own history.

---

## Scope Discipline

**Engram is a memory system.** Its unique differentiator is the effectiveness feedback loop: surface → track → evaluate → self-correct. No other tool in the Claude Code ecosystem does this. Auto-memory has no feedback. CLAUDE.md is manually maintained. Rules, skills, and hooks are static.

**Engram owns:**
- Memory lifecycle: learn → correct → surface → evaluate → maintain → evolve → link
- Memory graph: link memories to each other with spreading activation
- Context budget: aggressively manage engram's own injection footprint
- Cross-source awareness (READ-ONLY): scan CLAUDE.md, rules, skills to suppress duplicates and detect contradictions — but never write to them

**Engram signals but does NOT manage:**
- "Memory X contradicts CLAUDE.md entry Y — suppressing memory, flagging for review"
- "Memory Z has been a leech through 3 escalation levels — consider promoting to a rule or hook"
- "These 5 memories cluster around 'targ usage' — consider consolidating into a skill"

**Explicitly NOT engram's job:**
- Tracking effectiveness of CLAUDE.md entries, rules, or skills
- Proposing or writing changes to CLAUDE.md, rules, skills, settings.json, linters, or CI
- Managing enforcement mechanisms above the `reminder` level (blocking hooks, settings.json deny, linters, CI gates)

Other tools handle those concerns (claude-md-management plugin, manual user action, future tooling). Engram's signals can feed into them without owning the lifecycle.

---

## Context You Must Read First

Before planning anything, read these files in order. They contain the full history, architecture, and lessons that inform every decision:

1. **`docs/lessons.md`** — What worked, what failed, patterns to carry forward vs abandon. 24 lessons from 15+ features across two prior systems. Non-negotiable constraints are here.
2. **`docs/prompts/enforcement-and-context-optimization-output.md`** — The 7-step decision framework for handling instruction violations. The instruction salience hierarchy. Context budget model with measured token costs. Read this for the FULL landscape of enforcement mechanisms — engram operates in the bottom half of this hierarchy but should understand the whole picture to produce useful graduation signals.
3. **`docs/prompts/unified-instruction-registry.md`** — The original design for a unified registry tracking ALL instruction sources. **Note: this prompt descopes the registry to memory-focused with read-only awareness of other sources.** Read it for context on the full vision, but implement the focused version described in Package 0 below.
4. **`docs/specs/use-cases.md`** — All 28 use cases. UC-1 through UC-28 are implemented or in progress.
5. **`docs/specs/requirements.md`** — REQ-1 through REQ-67+. Includes REQ-46 (ACT-R frecency scoring).
6. **`docs/specs/architecture.md`** — ARCH-1 through ARCH-78+. Includes ARCH-35 (frecency scorer).
7. **`docs/specs/design.md`** — Interaction designs for all user-facing UCs.
8. **`docs/plans/memory-effectiveness-plan.md`** — The 3-phase effectiveness roadmap.
9. **`docs/prompt.md`** — The original rebuild prompt that started this project. Contains the model hierarchy, DI constraints, and phase structure.

Also read the current plugin integration points:
- `hooks/hooks.json` + all `hooks/*.sh` scripts — understand every hook event and what it does
- `.claude/rules/go.md` — current rules
- `CLAUDE.md` — project instructions
- `~/.claude/projects/-Users-joe-repos-personal-engram/memory/MEMORY.md` — auto-memory state

---

## Research Findings That Must Inform the Plan

### BMO (ngrok) — Self-Improving Coding Agent

Joel Hans at ngrok built BMO, a self-improving TypeScript agent, and ran ~100 sessions over 2 weeks. Key findings:

**What worked:**
- **Structured triggers > open-ended instructions.** A reflection template with explicit timing worked 100% of the time. Skills requiring sustained judgment worked 3% of the time. Takeaway: injection at specific moments beats "always remember this."
- **Telemetry was the "unsung hero."** Measurement enables evolution. Data-driven decisions consistently outperformed qualitative judgment. (Validates engram's effectiveness-first thesis.)
- **Specialized tools > generic flexibility.** Quote: "flexibility is expensive... the path to 95%+ success isn't making run_command better—it's making it unnecessary." Takeaway: deterministic enforcement > advisory when the rule is mechanical.

**What failed:**
- **System prompt attention decay.** Skills placed in the system prompt were barely used (2 uses in 60+ sessions). "Recent context dominates focus." System prompt becomes distant as conversation grows. **This is engram's biggest risk — we inject memories via additionalContext at session start, which has the same attention decay problem.**
- **"Sustained vigilance" is impossible.** Learning-event-capture (designed to continuously watch for correction patterns) failed completely. LLMs cannot maintain background monitoring while focused on a task.
- **Parallel self-improvement fails.** "Ask to do both task AND introspect while working, things fall apart." Only 2 tools built during active work despite "build IMMEDIATELY" directives. Recency bias makes the immediate task consume all attention.
- **Deferral is path of least resistance.** Despite explicit "build NOW" directives, BMO consistently deferred to maintenance passes. Creating an OPPORTUNITIES.md file paradoxically made deferral easier.

**BMO's conclusion:** "bmo isn't becoming autonomous. Instead, it's becoming a better collaborator." Focus shifted from making the agent smarter to building a better harness.

**GitHub:** `joelhans/bmo-agent` — TypeScript, 11 self-built tools, 7 skills, telemetry.json tracking tool success rates.

### A-Mem (Academic Research) — Agentic Memory for LLM Agents

A-Mem (arxiv 2502.12110) introduces self-organizing memory networks inspired by Zettelkasten:

- **Memory evolution:** When new info arrives, existing related memories UPDATE their contextual descriptions. Not just storage — continuous consolidation. `Updated_memory ← LLM(new_memory ∥ neighbor_set ∥ existing_memory)`.
- **Link generation:** New memories establish connections via two-stage retrieval: embeddings for top-k similar, then LLM analysis for meaningful connections (causal, conceptual, not just similarity).
- **Quality > quantity:** 85-93% token reduction while IMPROVING reasoning. Selective organization enables better reasoning with less context.
- **Connection structure matters more than raw size.** Ablation study: removing links degraded performance more than removing evolution. The graph structure IS the value.
- **Performance plateaus at k=40 retrieved memories.** Beyond that, noise overwhelms signal.

### Context Engineering (Lance Martin, LangChain)

Four strategies: **write** (save externally), **select** (pull relevant into window), **compress** (reduce tokens), **isolate** (split across agents).

- Claude Code's auto-compact fires at 95% context capacity — summarization loses injected memories.
- **PreCompact is an underused injection point.** Re-injecting top memories before compaction could preserve them through summarization.
- Multi-agent isolation uses 15x more tokens but prevents context pollution.

### LLM Context Problem 2026 (LogRocket)

Four failure modes, with measured impact:

| Failure Mode | Description | Measured Impact |
|---|---|---|
| **Context Clash** | Contradictory information in context | **39% average accuracy drop** (worst mode) |
| **Context Distraction** | >100k tokens → over-reliance on context | Llama 405B degrades at 32k tokens |
| **Context Confusion** | Irrelevant information interferes | 46 tools → failure; 19 tools → success |
| **Context Poisoning** | Hallucinated beliefs reinforced from context | Unbounded cascading errors |

**Critical insight:** Tool loadout reduction improved Llama 3.1 8B function-calling by 44%. Fewer, better-selected tools > comprehensive tool catalogs. Same principle applies to memories.

### Episodic Memory (fsck.com)

Cross-project vector search over archived conversations using a Haiku subagent to manage context bloat. Key insight: memory shouldn't be project-siloed — "If I've seen an error message before doing something else, I might still be able to pull up the context."

---

## Engram's Current Architecture (What Already Exists)

### The Full Instruction Landscape

Engram operates within a broader ecosystem of instruction delivery and enforcement. Understanding the full landscape helps engram produce useful graduation signals, even though engram only MANAGES the memory tier.

**Delivery surfaces** (where the model sees instructions, ordered by salience):

```
CLAUDE.md (global + project)                    ← engram READS (for dedup/contradiction)
  → Always loaded every turn, ~2700 tokens total

.claude/rules/*.md (file-pattern-scoped)        ← engram READS (for dedup/contradiction)
  → Loaded when matching files active, targeted context

MEMORY.md + topic files (auto-memory)           ← engram READS (for dedup/contradiction)
  → Always loaded (first 200 lines), model-written

Skills (.claude/skills/*.md, .claude/commands/*.md) ← engram READS (for dedup/contradiction)
  → Loaded by similarity or explicit /command invocation

Engram memories (data/memories/*.toml)          ← engram OWNS
  → Loaded by BM25 match + frecency ranking, effectiveness-tracked

Hook advisory output (system reminders)         ← engram OWNS (for memory-sourced advisories)
  → Injected at point of action, effectiveness-tracked

Subagent/teammate system prompts                ← out of scope
  → Isolated context per agent, inherits from spawner
```

**Enforcement mechanisms** (what ensures compliance, ordered by reliability):

| Level | Mechanism | Reliability | Context Cost | Managed By Engram? |
|---|---|---|---|---|
| `ci_gate` | CI pipeline check | 100% | 0 | No — out of scope |
| `linter` | Static analysis rule | 100% | 0 | No — graduation signal only |
| `settings_deny` | `~/.claude/settings.json` permissions.deny | 100% | 0 | No — graduation signal only |
| `blocking_hook` | PreToolUse hook returning `continue: false` | 100% | ~50 tokens | No — graduation signal only |
| `reminder` | PostToolUse hook injecting advisory | ~70% | ~100 tokens | **Yes — top of engram's ladder** |
| `emphasized_advisory` | Surfaced memory with urgency markers | ~50-60% | ~150 tokens | **Yes** |
| `advisory` | PreToolUse/SessionStart memory surfacing | ~40-60% | ~100-350 tokens | **Yes — default level** |
| `none` | Instruction exists, no enforcement | Variable | 0 from engram | No — not engram's concern |

**Engram's escalation range:** `advisory` → `emphasized_advisory` → `reminder` → **graduation signal** ("this memory isn't working — consider promoting to rule/hook/CLAUDE.md").

### Hook Integration Points (6 Events)

| Hook | Timing | Current Engram Use | Internal LLM Calls | Latency Budget |
|---|---|---|---|---|
| **SessionStart** | Session begins | Surface all memories by frecency, restore session context, surface pending signals, build binary + symlink | None (pure deterministic: BM25, frecency math, file I/O) | 30s sync |
| **UserPromptSubmit** | Each user message | Correct (detect + write memory), Surface (BM25 + frecency for prompt) | Haiku: classify signal tier + extract memory fields (correct). None for surface. | 30s sync + 120s async |
| **PreToolUse** | Before each tool call | Advisory memory surfacing (BM25 for tool name + input) | None (BM25 + frecency only) | 10s sync |
| **PostToolUse** | After each tool call | Proactive reminders for Write/Edit on matched file patterns | None (pattern match + effectiveness lookup) | 10s sync |
| **PreCompact** | Before context compaction | Signal detection (UC-28) | None currently; could add haiku for signal classification | 60s sync |
| **Stop** | Session ends | Learn → Evaluate → Audit → Signal-detect → Context-update (fire-and-forget) | Haiku ×5: extract candidates (learn), classify outcomes (evaluate), assess compliance (audit), classify signals (signal-detect), summarize context (context-update) | 120s async |

**Note on hook-internal LLM calls:** Hooks are deterministic entry points but can invoke LLM calls internally through the Go binary. The model hierarchy still applies inside hooks: deterministic checks first (keyword fast-path, hash comparison, BM25 scoring), LLM only when judgment is required (signal classification, outcome evaluation, context summarization). All hook-internal LLM calls are fire-and-forget (ARCH-6) — failure doesn't block the session.

### Effectiveness Pipeline (What's Measured)

```
Surface memory → Log surfacing event (surfaced_count, context, timestamp)
                      ↓
Session ends → Evaluate: LLM classifies each surfaced memory as followed/contradicted/ignored
                      ↓
Aggregate → effectiveness_score = followed / (followed + contradicted + ignored)
                      ↓
Classify → 2x2 matrix: Working / Leech / Hidden Gem / Noise / InsufficientData
                      ↓
Signal → Detect maintenance/promotion/demotion signals from quadrant + thresholds
                      ↓
Act → Maintain (rewrite/broaden/remove), Escalate (advisory→reminder→graduate)
```

### ACT-R Frecency Scoring (REQ-46, ARCH-35)

Currently implemented for engram memories only:

```
activation = frequency × recency × spread × effectiveness

frequency  = log(1 + surfaced_count)
recency    = 1 / (1 + hours_since_last_surfaced)
spread     = log(1 + len(surfacing_contexts))
effectiveness = max(0.1, effectiveness_score / 100)  [default 0.5]

combined_score (prompt/tool modes) = bm25_score × (1 + activation)
```

**What's missing from ACT-R that the research says matters: spreading activation.** In full ACT-R, activating one memory spreads activation to associated memories. We have frequency, recency, spread, and effectiveness — but no associative links between memories. When memory A fires, related memory B should get a boost. This is the mechanism that A-Mem validated as more important than raw memory count.

---

## The Core Thesis: Memory Excellence with Cross-Source Awareness

The key insight from combining BMO's findings, A-Mem's architecture, and engram's existing infrastructure:

**Make engram's memories the best they can be — higher quality, better connected, surgically surfaced — and give engram read-only awareness of other instruction sources so it doesn't contradict or duplicate them.**

This means:

1. **Memory graph with spreading activation.** Link memories to each other so activating one boosts related memories. Surface clusters, not isolated facts.

2. **Memory evolution, not just accumulation.** When new context arrives that overlaps an existing memory, MERGE and enrich — don't just deduplicate and skip.

3. **Cross-source awareness (read-only).** Before surfacing a memory, check: does CLAUDE.md already say this? Does a rule cover it? If so, suppress the memory (saves tokens, avoids redundancy). Does the memory contradict another source? If so, suppress and signal.

4. **Aggressive budget management.** Surface fewer, better memories. Quality over quantity. Research shows 85-93% token reduction IMPROVES reasoning.

5. **Graduation signals.** When a memory has been escalated through engram's range (advisory → emphasized → reminder) and is still a leech, signal that it needs to graduate to a higher enforcement tier. Engram diagnoses; the user or other tools act.

### Memory Graph Model

```
Node = engram memory (registry entry)
Edge = association between memories (shared concepts, co-surfacing, evaluation correlation)
Weight = association strength

When memory A is surfaced:
  for each linked memory B:
    B.activation += A.activation × edge_weight(A, B)
```

Non-memory instruction sources (CLAUDE.md entries, rules, skills) appear in the graph as **read-only context nodes** — they influence link weights and suppression decisions, but they're not managed by engram.

### Building Links Without Embeddings (Pure Go)

A-Mem uses embeddings for initial link discovery. We bootstrap links from cheaper signals:

1. **Concept overlap:** Memories share `concepts[]` or `keywords[]` → link them. Weight = Jaccard similarity of concept sets.
2. **Co-surfacing:** Memories surfaced in the same session/prompt → link them. Weight = co-occurrence count / max(individual counts).
3. **Evaluation correlation:** Memories that are followed together or violated together → strong link. Weight = Pearson correlation of outcome vectors.
4. **Content similarity:** BM25 between memory principles → link if score > threshold.

For cross-source awareness links (memory ↔ CLAUDE.md entry):
5. **Concept overlap with non-memory sources:** Scan CLAUDE.md/rules/skills for keyword overlap with memories. These links are used for **suppression** (don't surface a memory that duplicates an always-loaded instruction) and **contradiction detection**, not for spreading activation.

No embeddings needed. All computable in pure Go from existing data.

---

## Simplification: Remove Out-of-Scope Complexity

Before building new features, remove existing code and infrastructure that was built for the broader "unified instruction management" vision but is now out of scope. This reduces maintenance burden and clarifies engram's identity.

### Code to Remove

| Target | Current Purpose | Why Remove |
|---|---|---|
| `internal/automate/` (entire package) | UC-22: Generate automation proposals (shell scripts, linter rules, install paths) | Engram doesn't generate enforcement mechanisms. Graduation signals replace this. |
| `engram automate` CLI subcommand | Wires UC-22 into CLI | No backing package after removal. |
| `LevelPretoolBlock` and `LevelAutomationCandidate` in `internal/maintain/escalation.go` | Top 2 escalation levels | Engram's ladder stops at `reminder` + graduation signal. Remove these levels, the `predictImpact` logic for them, and the automation proposal generation at the top of the ladder. |
| Non-memory extractors in `internal/registry/extract.go` | `ClaudeMDExtractor`, `MemoryMDExtractor`, `RuleExtractor`, `SkillExtractor` — register non-memory sources in the persistent registry | Registry is memory-only. Non-memory parsing moves to the ephemeral cross-source scanner (Package 0). **Keep the parsing logic itself** (bullet extraction, rule parsing) — relocate it to `internal/crossref/` for the scanner. Remove the registry registration path. |
| Non-memory test cases in `registry/extract_test.go` | Tests for CLAUDE.md, rule, skill extraction into registry entries | Move to `internal/crossref/` tests when building the scanner. |
| Non-memory `SourceType` values in `registry/classify.go` | `alwaysLoadedSources` map with `claude-md`, `memory-md` | Registry classification only needs to handle `memory` source type. Simplify. |

### Code to Simplify

| Target | Current State | Simplified State |
|---|---|---|
| `escalationLadder` in escalation.go | 5 levels: advisory → emphasized → posttool_reminder → pretool_block → automation_candidate | 3 levels + graduation: advisory → emphasized_advisory → reminder → graduated (signal, not enforcement) |
| `registry/entry.go` `SourceType` field | Accepts any string ("memory", "claude-md", "rule", "skill", "hook") | Only "memory" for persisted entries. Add `enforcement_level` field with values: `advisory`, `emphasized_advisory`, `reminder`, `graduated`. |
| `registry merge` | Can merge any source type | Memory-to-memory only |
| `signal/detector.go` promotion signals | `skill_to_claudemd`, `claudemd_demotion` | Replace with `graduation` signal type — engram recommends the user promote/change, doesn't specify the target |
| `internal/instruct/` (UC-20) | Cross-source deduplication and quality audit scanning CLAUDE.md + skills | Descope to memory-only quality audit. Cross-source dedup becomes a read-only check in the surface pipeline (Package 1/4), not a standalone audit. |

### Process

- Run this simplification as the FIRST step of Phase A, before building new features
- Each removal should be a single atomic commit with test updates
- Run `targ check` after each removal to catch breakage
- The cross-source parsing logic (bullet extraction, etc.) is valuable — relocate to `internal/crossref/`, don't delete

---

## Priority-Ranked Work Packages

Based on impact toward topline goals, research evidence, and dependency ordering. Six packages, phased over ~4 weeks.

### Package 0: Memory Registry Completion + Cross-Source Scanner

**Status:** `internal/registry/` exists with JSONL store. Surfacing and evaluation already wired for memories. No cross-source awareness.

**Remaining work:**

**Registry completion (memory-focused):**
- Add `enforcement_level` field to registry entries: `advisory` | `emphasized_advisory` | `reminder` | `graduated`
- `graduated` means engram has exhausted its escalation range and signaled that this instruction needs a higher-salience delivery surface or deterministic enforcement
- Track enforcement level transitions with timestamps for trend analysis

**Cross-source scanner (read-only):**
- New `internal/crossref/` package with `Scanner` interface
- `Scanner.Scan()` reads CLAUDE.md (global + project), rules, skills — extracts instruction-like content (bullet points with directive language, principle statements, anti-patterns)
- Produces a lightweight index: `{source, content_hash, keywords[], principle_text}` per detected instruction
- Index is rebuilt at SessionStart (cheap — just file reads + keyword extraction, no LLM)
- Used by surface pipeline for suppression and contradiction checks (Packages 1 and 4)
- **Does NOT register non-memory sources in the registry.** The cross-source index is ephemeral — rebuilt each session, not persisted. Only memories get registry entries.

**Why P0:** Everything else depends on the registry having enforcement_level tracking and cross-source awareness existing as an input to surfacing.

**References:** `docs/prompts/unified-instruction-registry.md` (read for full vision context, implement focused version above).

### Package 1: Contradiction Detection (Read-Only Cross-Source)

**Research evidence:** Context clash causes 39% accuracy drop — the single worst failure mode measured in the literature. This is the highest-ROI change for instruction compliance.

**Design:**
- At surfacing time, after selecting top-N memories, check each against the cross-source index (Package 0)
- Contradiction = a memory whose principle opposes a directive in CLAUDE.md, a rule, a skill, or another memory
- Detection: BM25 between memory principle and cross-source index entries + keyword-based heuristic (same subject, opposing verbs: "always X" vs "never X")
- For ambiguous cases: haiku classifier (budget: max 3 LLM calls per surface event)
- **Resolution (within engram's scope):** Suppress the contradicting memory, don't surface it. Log a signal: `{type: "contradiction", memory: "...", contradicts: "claude-md:...", recommendation: "review and resolve"}`
- **Resolution (outside engram's scope):** Surface the signal to the user via SessionStart or `engram review`. The user decides whether to fix the memory, fix the CLAUDE.md entry, or accept the contradiction.
- Also check memory-vs-memory contradictions within the top-N selection

**Implementation:**
- New `internal/contradict/` package with `Detector` interface
- Wire into `surface.go` as a post-ranking filter
- New signal type: `contradiction` in UC-28 signal queue
- Contradiction signals surfaced in SessionStart hook alongside other pending signals

### Package 2: PreCompact Memory Re-Injection

**Research evidence:** BMO found system prompt instructions fade as conversation grows. Claude Code's auto-compact at 95% context summarizes away injected memories. Re-injecting the highest-value memories before compaction preserves them through summarization.

**Design:**
- PreCompact hook already exists (runs signal-detect)
- Add: surface top-5 highest-effectiveness memories as `additionalContext` in PreCompact output
- These get included in the compaction summary, surviving into the compressed context
- Budget: 500 tokens max (5 memories × ~100 tokens each)
- Selection: rank by effectiveness only (no BM25, no query), skip memories with effectiveness < 40%
- Format: concise principle statements only, not full memory content

**Implementation:**
- Extend `pre-compact.sh` to call `engram surface --mode precompact --budget 500`
- New surface mode: `precompact` — ranks by effectiveness score descending, takes top-5 within budget
- Output format: `[engram] Preserving top memories through compaction:\n- <principle 1>\n- <principle 2>...`

**Why this is high priority:** Low effort (one new surface mode + hook extension), directly addresses the attention decay problem BMO identified. Every compaction event without re-injection risks losing the most valuable memories.

### Package 3: Memory Graph & Spreading Activation

**Research evidence:** A-Mem's ablation study: removing links degraded performance more than removing memory evolution. Connection structure IS the value. ACT-R spreading activation is the validated mechanism for associative memory.

**Data model extension to registry:**
```jsonl
{
  "id": "memory:always-use-targ.toml",
  ...existing fields...,
  "enforcement_level": "advisory",
  "links": [
    {"target": "memory:run-targ-check.toml", "weight": 0.7, "basis": "concept_overlap"},
    {"target": "memory:targ-build-not-go-build.toml", "weight": 0.6, "basis": "evaluation_correlation"}
  ]
}
```

Links are memory-to-memory only. Cross-source relationships (memory ↔ CLAUDE.md) are handled by the ephemeral cross-source index (Package 0), not by persistent links.

**Link building (pure Go, no embeddings):**
1. On `registry init` and after `learn`: compute concept/keyword Jaccard between all memory pairs above threshold
2. On each surfacing event: record co-surfacing pairs, update co-surfacing weights
3. On each evaluation: record outcome correlation, update evaluation weights
4. Link pruning: remove links with weight < 0.1 after 10+ co-surfacing opportunities

**Spreading activation extension to frecency (REQ-46 evolution):**
```
base_activation = frequency × recency × spread × effectiveness  (existing)

spreading_activation = Σ (linked_memory.base_activation × edge_weight)
                       for each linked memory

total_activation = base_activation + decay_factor × spreading_activation
```

Where `decay_factor` ∈ [0, 1] controls how much influence neighbors have. Start at 0.3, tune based on effectiveness data.

**Surfacing change:**
- When memory A is selected for surfacing, check if any linked memory B has high weight AND is not already selected
- If B.total_activation > threshold, include it as a "cluster note": `"(Related: <B.principle>)"`
- Budget: cluster notes are compressed (principle only, ~20 tokens each), max 2 per surfaced memory

**Why this matters for token efficiency:** Instead of independently surfacing 3 targ-related memories, the graph recognizes they're one cluster. Surface the highest-effectiveness node + cluster notes. Save ~200 tokens per event.

### Package 4: Aggressive Context Budget Enforcement

**Research evidence:** A-Mem achieves 85-93% token reduction while improving reasoning. Tool loadout reduction = 44% improvement. More context ≠ better outcomes.

**Current measured budget:** ~24,700 tokens/session for engram alone.

**Targets:**

| Hook | Current | Target | Strategy |
|---|---|---|---|
| SessionStart | ~1,100 tokens | 600 tokens | Top-7 by effectiveness (not top-20 by frecency), cluster dedup, suppress cross-source duplicates |
| UserPromptSubmit | ~300 tokens | 250 tokens | Raise BM25 floor, cluster dedup |
| PreToolUse | ~350 tokens | 150 tokens | Top-2 (not 5), effectiveness floor of 40%, suppress if principle keywords in recent context |
| PostToolUse | ~100 tokens | 100 tokens | Already capped |
| PreCompact | 0 (new) | 500 tokens | Top-5 by effectiveness (Package 2) |
| **Total/session** | **~24,700** | **~13,500** | **45% reduction** |

**Key changes:**
- **SessionStart:** effectiveness-gated (only surface memories with effectiveness > 40% or insufficient data). Use cross-source index to suppress memories that duplicate always-loaded CLAUDE.md/rules content.
- **PreToolUse:** higher relevance floor + keyword suppression (if the principle's keywords appear in recent transcript, the model already has it — don't re-inject)
- **Cluster dedup:** when two linked memories would both surface, surface only the highest-effectiveness one + a cluster note
- **Cross-source suppression:** if a memory's principle is covered by a CLAUDE.md entry, rule, or skill (detected via cross-source index), skip it. The higher-salience source already delivers it.
- **Transcript-based suppression (handles hook dedup):** keyword suppression checks the recent transcript (~500 tokens), which includes output from ALL hooks (engram's own prior injections AND other plugins' hook output). If principle keywords already appear in the transcript, the model already has the instruction — don't re-inject. This is the practical mechanism for deduplicating against hook feedback, since hooks can't observe each other's output in real-time.

### Package 5: Memory Evolution (Merge-on-Write)

**Research evidence:** A-Mem's memory evolution — updating existing memories with new context — is their key differentiator from static storage. Engram's current dedup (50% keyword overlap → skip new candidate) discards information.

**Design change:**
- When learn pipeline detects a new candidate overlapping an existing memory (>50% keyword overlap):
  - Instead of skipping: MERGE the candidate into the existing memory
  - Update the existing memory's principle to incorporate new context
  - Append new keywords/concepts that weren't in the original
  - Record the merge in the registry `absorbed` field
  - Preserve the existing memory's effectiveness history (it earned those scores)
  - Update links: re-compute concept overlap links for the merged memory

**Implementation:**
- Extend `internal/learn/dedup.go` with a `MergeStrategy` interface
- Default strategy: LLM-assisted merge (haiku) that combines principles into a single, stronger statement
- Fallback: keyword/concept union + longer principle wins (simple but lossy)
- Registry records the merge event for traceability

**Why merge > skip:** A memory about "use targ for tests" that gets a new candidate "use targ for builds too" should become "use targ for all operations (test, build, check)" — not silently discard the new signal.

### Package 6: Memory-Scoped Escalation with Graduation Signals

**Research evidence:** BMO's #1 finding is "knowing ≠ doing." Engram's escalation engine (`internal/maintain/escalation.go`) exists as library code but is only partially wired to CLI.

**Engram's escalation range (3 levels + graduation):**

```
advisory (default — memory surfaced via additionalContext)
    ↓ effectiveness < 40% after 5+ surfacings
emphasized_advisory (surfaced with urgency markers, higher budget priority)
    ↓ effectiveness still < 40% after 5+ more surfacings
reminder (PostToolUse targeted injection at point of action)
    ↓ effectiveness still < 40% after 5+ more surfacings
GRADUATION SIGNAL: "This memory isn't working as advisory context.
  Consider: .claude/rules/ file, PreToolUse blocking hook,
  settings.json deny rule, or CLAUDE.md entry."
```

**Remaining work:**
- Wire escalation to registry `enforcement_level` transitions: each escalation updates the field
- `advisory` → `emphasized_advisory`: change surfacing format (add "IMPORTANT:" prefix, bold principle)
- `emphasized_advisory` → `reminder`: generate a PostToolUse reminder pattern for the memory's file glob patterns, add to remind configuration
- `reminder` → `graduated`: mark memory as graduated, surface graduation signal to user with specific recommendations based on the memory's content:
  - Mechanical rule → recommend settings.json deny or linter rule
  - File-pattern-scoped → recommend .claude/rules/ file
  - Broad behavioral → recommend CLAUDE.md entry
  - Complex procedural → recommend skill
- De-escalation: if effectiveness improves after escalation, propose de-escalating to the previous level
- User sees escalation proposals in `engram review` and graduation signals in SessionStart

**What engram does NOT do:** Generate the actual rule/hook/CLAUDE.md entry. It diagnoses and recommends. The user (or another tool) acts.

---

## Execution Strategy

### Use `/project` (traced) for orchestration

This work should go through the traced specification process. Each package becomes a UC or set of UCs. The spec tree provides traceability from goals → use cases → requirements → design → architecture → implementation.

### Phasing

**Phase A (Weeks 1-2): Simplify + Foundation + Highest-Impact Fixes**
- **Simplification first:** Remove `internal/automate/`, remove top escalation levels, relocate non-memory extractors to `internal/crossref/`, simplify registry to memory-only source type. See "Simplification" section above.
- Package 0: Memory registry completion + cross-source scanner
- Package 1: Contradiction detection (39% accuracy drop = highest ROI)
- Package 2: PreCompact re-injection (low effort, high impact)
- Package 4: Aggressive budget enforcement (quick wins: reduce top-N, raise relevance floors, cross-source suppression)

**Phase B (Weeks 3-4): Graph + Evolution + Escalation**
- Package 3: Memory graph with spreading activation
- Package 5: Memory evolution (merge-on-write)
- Package 6: Memory-scoped escalation with graduation signals

### Success Metrics

Track these to know if the evolution is working:

1. **Memory compliance rate:** `followed / (followed + contradicted + ignored)` for engram memories. Target: >80% (up from current ~60%).
2. **Context efficiency:** Total engram tokens/session. Target: <15,000 (down from ~24,700).
3. **Contradiction rate:** Contradictions detected and suppressed per session. Target: trending toward 0 as contradictory memories get resolved.
4. **Cluster coverage:** % of memories with ≥1 link in the graph. Target: >60%.
5. **Graduation signal quality:** When engram signals graduation, does the user act on it? (Track manually initially.)
6. **Cross-source suppression rate:** % of surface events where a memory was suppressed because a higher-salience source covers it. Higher = better dedup = less wasted context.

---

## Constraints (Non-Negotiable)

From `docs/lessons.md` and project CLAUDE.md:

1. **Pure Go, no CGO.** All compiled code must be pure Go.
2. **DI everywhere.** No function in `internal/` calls `os.*`, `http.*`, `sql.Open`. All I/O through injected interfaces.
3. **Content quality > mechanical sophistication.** Don't build graph infrastructure before validating memory content quality.
4. **Measure impact, not just frequency.** Every feature must connect to the effectiveness pipeline.
5. **Deterministic first, local analysis second, models last.** Hash → BM25/TF-IDF → Haiku → Sonnet → Opus.
6. **Autonomous memories, proposed artifacts.** Full autonomy over internal memory state. Graduation signals are proposals — the user decides what to do with them.
7. **Fire-and-forget error handling (ARCH-6).** CLI always exits 0. No error propagation to Claude Code.
8. **Plugin form factor.** Everything ships as a Claude Code plugin installable via `~/.claude/plugins/`.
9. **TDD for ALL artifact changes.** Full red/green/refactor. Never skip phases.
10. **Use `targ` for build/test/check.** Never raw `go test`, `go build`, `go vet`.

---

## Key References

| Document | What It Contains | When to Read It |
|---|---|---|
| `docs/lessons.md` | 24 lessons from prior systems | Before any design decision |
| `docs/prompts/unified-instruction-registry.md` | Full registry vision (descoped — read for context) | Before Package 0 |
| `docs/prompts/enforcement-and-context-optimization-output.md` | 7-step violation framework + full enforcement landscape | Before Packages 1, 4, 6 |
| `docs/specs/use-cases.md` | All 28 existing UCs | Before adding new UCs |
| `docs/specs/requirements.md` | REQ-1 through REQ-67+ | Before adding requirements |
| `docs/specs/architecture.md` | ARCH-1 through ARCH-78+ | Before architecture decisions |
| `docs/plans/memory-effectiveness-plan.md` | 3-phase effectiveness roadmap | For Phase B context |
| `docs/prompt.md` | Original rebuild prompt with full constraints | For process/constraint reference |
| `internal/frecency/frecency.go` | Current ACT-R implementation | Before Package 3 |
| `internal/registry/` | Current registry implementation | Before Package 0 |
| `internal/maintain/escalation.go` | Current escalation engine | Before Package 6 |
| `internal/signal/` | Signal detection + queue + apply | Before Packages 0, 1 |

### External Research

| Source | Key Finding | Relevant Package |
|---|---|---|
| BMO (ngrok) `ngrok.com/blog/bmo-self-improving-coding-agent` | "Knowing ≠ doing"; structured triggers > open-ended; telemetry enables evolution | All, especially 6 |
| A-Mem (arxiv 2502.12110) | Memory linking > raw storage; 85-93% token reduction; evolution > accumulation | 3, 5 |
| Context Engineering (Lance Martin) | Write/select/compress/isolate; PreCompact underused | 2, 4 |
| LLM Context Problem 2026 (LogRocket) | Contradiction = 39% drop; >100k tokens degrades; tool loadout reduction = 44% improvement | 1, 4 |
| Episodic Memory (fsck.com) | Cross-project memory; Haiku subagent for context management | Future cross-project work |
