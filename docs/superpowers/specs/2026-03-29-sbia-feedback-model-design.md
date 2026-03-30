# SBIA Feedback Model for Engram Extraction

**Status:** Design complete
**Date:** 2026-03-29

## Problem Statement

Engram's current memory model captures _what to do_ (principle) and _what not to do_ (anti_pattern), but doesn't explicitly model _when_ the correction applies. Keywords attempt to proxy for situation context, but they're term-level matches that frequently surface in wrong contexts (13 memories currently flagged for irrelevant surfacing due to overly generic keywords).

The SBIA framework (Situation, Behavior, Impact, Action) — a validated feedback model from organizational psychology — provides better structure for behavioral correction by anchoring each memory to the observable situation where it applies.

## Core Insight

Engram already has raw memory via session logs and `/recall`. What makes structured memories valuable is recognizing **similar situations** and applying the right **corrective action**. The current schema stores the correction but not the situation that triggers it.

## Current Schema vs. SBIA Mapping

| SBIA Dimension                                  | Current Field           | Quality of Fit                                                                                                                |
| ----------------------------------------------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| **Situation** (when does this apply?)           | `keywords` + `concepts` | **Poor.** Keywords are topic tags, not scenario descriptors. No field captures "when are you in this situation?"              |
| **Behavior** (what's the default/wrong action?) | `anti_pattern`          | **Partial.** Captures "what not to do" but not "what you naturally default to." Empty for tier C, optional for B.             |
| **Impact** (what goes wrong?)                   | `rationale`             | **Partial.** Explains why the principle matters but doesn't explicitly describe the negative outcome of the default behavior. |
| **Action** (what to do instead?)                | `principle`             | **Good.** Strongest mapping — "what to do instead."                                                                           |

**The critical gap is Situation.** No field answers: _"What would the agent be doing when this correction applies?"_

## Research Support

### Encoding Specificity Principle (Tulving & Thomson, 1973)

Memory retrieval is most effective when the retrieval cue matches the encoding context. If you encode _what the agent was doing when the correction happened_, you can match on _what the agent is doing now_. Current keywords match on topic; SBIA would match on activity context.

### Case-Based Reasoning (Kolodner 1993, Aamodt & Plaza 1994)

AI systems that retrieve past solutions by matching _problem situations_ consistently outperform those that match on abstract rules or keyword indices. CBR explicitly indexes cases by situational features — goal, constraints, what went wrong. SBIA moves engram from rule-based retrieval toward case-based retrieval.

### SBI/SBIA Model (Center for Creative Leadership)

One of the most validated feedback frameworks in organizational psychology. Its power comes from anchoring feedback to _observable behavior in a specific situation_, making it actionable rather than abstract. The "A" (Action/alternative) extension maps directly to engram's use case.

### Situated Cognition (Brown, Collins, Duguid 1989)

Knowledge abstracted away from the situation of use is harder to apply than knowledge connected to its context of use. Current `principle` fields are abstracted rules ("always use targ"). SBIA keeps situational grounding: "when running tests/builds (S), you invoke go test directly (B), which bypasses coverage and lint (I), so use targ instead (A)."

## Example: Current vs. SBIA

### Current Memory

```toml
title = "Use targ for builds"
content = "Always use targ build system instead of raw go commands"
principle = "Use targ test, targ check, targ build for all operations"
anti_pattern = "Running go test or go vet directly"
rationale = "targ wraps build/test/lint with project-specific configuration"
keywords = ["targ", "build", "go-test", "go-vet"]
confidence = "A"
```

### SBIA-Structured Memory

```toml
situation = "When running tests, builds, lint checks, or any Go toolchain operation in a project that uses targ as its build system"
behavior = "Invoking go test, go build, go vet, or other Go commands directly"
impact = "Bypasses project-specific coverage thresholds, lint rules, and build configuration that targ enforces, leading to false confidence in test results"
action = "Use targ test, targ check-full, targ build for all operations"
project_scoped = true
```

Note how the SBIA `situation` field captures activity-level context ("when running tests, builds, lint checks") that keywords like `["targ", "build"]` can't express. BM25 retrieval matches against the full SBIA text, so no separate keyword array is needed.

## Decision: Option A (Full Restructure)

Option B (layer) rejected — extra fields is not the goal. Options A and C differ only in whether `title`, `content`, `observation_type`, and `concepts` survive alongside SBIA fields. Since SBIA fields _are_ the content, those fields are redundant. Go with A: simplify the data model and what we extract/store.

## Current Field Usage Across Pipeline

Traced every field through every pipeline stage to understand migration impact.

| Field                | Extract | Dedup                           | Write        | Surface                                  | Evaluate | Maintain/Signal                                           |
| -------------------- | ------- | ------------------------------- | ------------ | ---------------------------------------- | -------- | --------------------------------------------------------- |
| **title**            | writes  | -                               | persists     | SearchText (BM25)                        | -        | display, BM25 adapter, cluster ID                         |
| **content**          | writes  | -                               | persists     | SearchText (BM25)                        | -        | -                                                         |
| **principle**        | writes  | -                               | persists     | SearchText (BM25), **displayed to user** | -        | display, BM25 adapter, rewrite target, consolidation text |
| **anti_pattern**     | writes  | -                               | persists     | - (not in SearchText)                    | -        | signal apply (rewrite target)                             |
| **rationale**        | writes  | -                               | persists     | - (not in SearchText, not displayed)     | -        | -                                                         |
| **keywords**         | writes  | **primary axis** (>50% overlap) | persists     | SearchText (BM25), common-keyword filter | -        | consolidation text (`keywords + principle`)               |
| **concepts**         | writes  | -                               | persists     | SearchText (BM25)                        | -        | -                                                         |
| **observation_type** | writes  | -                               | persists     | -                                        | -        | render only (display on creation)                         |
| **filename_summary** | writes  | -                               | filename gen | -                                        | -        | -                                                         |
| **generalizability** | writes  | filter (<2)                     | persists     | GenFactor (cross-project penalty)        | -        | migration                                                 |
| **confidence/tier**  | writes  | -                               | persists     | frecency tier boost (A=1.2, B=0.2)       | -        | quadrant diagnosis                                        |

### Key Findings

1. **`rationale`** — write-only dead end. Extracted, persisted, never read by any downstream stage. Nothing surfaces it, searches it, or displays it.

2. **`observation_type`** — only read by `render.go` to display "Type: correction" at creation time. Never used for retrieval, matching, or maintenance.

3. **`concepts`** — only consumed by `SearchText()` for BM25. No other pipeline stage reads them. Secondary retrieval signal, redundant with SBIA situation text.

4. **`anti_pattern`** — not in `SearchText()`, so it doesn't help retrieval. Only consumed by `signal/apply.go` for rewrites during maintenance.

5. **`keywords`** — the one structurally load-bearing field. Dedup depends on keyword set overlap (>50%). Surface uses them in SearchText. Signal consolidation uses them. **Dropping keywords means replacing the dedup strategy** with BM25 on SBIA text, which already exists in the codebase.

6. **`principle`** — the most consumed content field. It's the **only content field displayed to users** during surfacing (filename slug + principle). Also used in consolidation matching via signal/bm25_adapter.

### Pipeline Impact of SBIA Restructure

| Pipeline Stage          | Current Dependency                                                       | SBIA Equivalent                                             | Migration Difficulty                  |
| ----------------------- | ------------------------------------------------------------------------ | ----------------------------------------------------------- | ------------------------------------- |
| **Extract**             | Produces 10+ fields                                                      | Produces 4 SBIA fields + metadata                           | Low — simpler prompt                  |
| **Dedup**               | Keyword set overlap >50%                                                 | BM25 on SBIA text (infra exists)                            | Medium — new strategy                 |
| **Write**               | MemoryRecord with all fields                                             | MemoryRecord with SBIA fields                               | Low — fewer fields                    |
| **Surface (retrieval)** | BM25 on `SearchText()` (title+content+principle+keywords+concepts)       | BM25 on `SearchText()` (situation+behavior+impact+action)   | Low — same mechanism, different input |
| **Surface (display)**   | Shows `principle` to user                                                | Shows `action` to user                                      | Low — rename                          |
| **Evaluate**            | Tracks counters (orthogonal to content)                                  | No change                                                   | None                                  |
| **Maintain/Signal**     | Reads title, principle, anti_pattern, keywords for consolidation/rewrite | Reads action, behavior, situation for consolidation/rewrite | Medium — field name changes           |

## Decision: Corrections-Only Extraction with Sonnet

### Revised Extraction Model

Replace the dual extraction paths (real-time `correct` + batch `learn`) with a single, richer correction path:

**Current flow:**

1. UserPromptSubmit: Haiku classifies user message → writes memory with whatever it can infer from one sentence
2. Stop async: Haiku scans full transcript → extracts batch learnings → dedup → write

**New flow:**

1. UserPromptSubmit: Detect correction (fast-path keywords + Haiku classification) → read current session transcript tail → **Sonnet** extracts all four SBIA dimensions from correction + preceding conversation → dedup → write
2. Stop async: **Repurposed for evaluate** — Haiku assesses pending surfacings against transcript

### Rationale

A correction message like "always use targ" contains only the **Action**. The Situation ("when running tests/builds"), Behavior ("invoking go test directly"), and Impact ("bypasses coverage/lint") are in the _surrounding conversation_ — what Claude was doing, what went wrong, why the user intervened. Haiku operating on a single message can't reconstruct these dimensions. Sonnet with full conversation context can.

### Key Consequences

1. **Batch extraction (`learn`) eliminated.** The stop.sh async hook is repurposed for the **evaluate** stage — processing `[[pending_evaluations]]` in memory TOML files. The `extract`, `learn`, and `flush` packages are removed or dramatically simplified.

2. **Sonnet replaces Haiku for extraction.** Higher quality, higher cost — but corrections are rare per session (typically 0-3), so the cost increase is negligible.

3. **Transcript context retrieval.** See "Decision: Context Retrieval for SBIA Extraction" below.

4. **Tier C eliminated.** Corrections are inherently behavioral. No contextual facts to classify. The tier C filtering that already exists in the learn pipeline becomes moot.

5. **Dedup volume drops.** With corrections-only extraction, far fewer candidates per session. BM25 on SBIA text is sufficient for the lower volume (keyword-set overlap strategy no longer needed).

6. **Keywords dropped.** With SBIA text available for both retrieval and dedup via BM25, the keywords array no longer serves a unique purpose.

### Future: Indirect Triggers

"What about X?" and similar patterns aren't explicit corrections but may signal implicit learnings worth extracting. These indirect triggers are out of scope for the initial SBIA restructure but worth revisiting once the correction-only path is stable. The detection mechanism (Haiku classification of non-fast-path messages) already exists and extends naturally to flag these for deferred SBIA extraction.

## Decision: Context Retrieval for SBIA Extraction

### What Sonnet Needs

To extract all four SBIA dimensions, Sonnet needs the conversation leading up to the correction:

- **Situation:** What task/goal was the agent working on? (May be established many turns ago)
- **Behavior:** What did the agent do wrong? (Usually the last 1-3 assistant turns, often visible in tool calls)
- **Impact:** What went wrong as a result? (Often in tool results or the correction message itself)
- **Action:** What to do instead? (In the correction message)

### Why Tail Works

Corrections are detected at UserPromptSubmit — the correction is always the latest user message. The transcript tail *is* the preceding conversation. No windowing or seeking to a midpoint required.

### Context Budget: Sonnet as Semantic Gate

Rather than building heuristic logic to detect topic/episode boundaries (which requires its own LLM call or fragile rules), provide Sonnet a generous transcript tail and let it focus on the relevant context. Sonnet is good at ignoring irrelevant earlier conversation — its attention is the semantic gate.

**Budget: `context_byte_budget`** (default 50KB, matching recall's existing per-session budget). Well within Sonnet's context window. At 0-3 corrections per session, the cost per correction is ~$0.01-0.02 — negligible. The byte budget is a ceiling for pathological cases (8-hour sessions), not the semantic boundary.

### Reuse from Recall

`TranscriptReader.Read` and `context.Strip` already handle reading session JSONL, stripping noise, and returning tail content with a byte budget. Reuse these directly — no need for recall's multi-session discovery or Haiku extraction layer.

### SBIA Strip Mode: Include Tool Calls

`context.Strip` currently drops all `tool_use` and `tool_result` blocks. This is correct for recall (conversation flow summary) but wrong for SBIA extraction. Tool calls are often the literal **Behavior** — "ran `go test ./...`" is a tool call. Tool results are the **Impact** evidence — "tests passed but missed coverage threshold."

The assistant's text describes what it *intended* to do; tool calls are what it *actually* did. For recall, the assistant's reasoning is the source of truth. For SBIA, the assistant's reasoning is exactly what we're correcting — its tool calls are evidence of the faulty behavior, not noise.

**Design: `StripConfig` parameter on `Strip`** rather than a separate function:

| Element | Recall mode | SBIA mode |
|---------|------------|-----------|
| User messages | Keep | Keep |
| Assistant text | Keep | Keep |
| Tool name | Drop | **Keep** |
| Tool arguments | Drop | **Keep, truncated** (`context_tool_args_truncate`) |
| Tool result status | Drop | **Keep** (success/error) |
| Tool result body | Drop | **Keep, truncated** (`context_tool_result_truncate`) |
| System reminders | Drop | Drop |
| Base64 data | Drop | Drop |

Truncation lengths are approximate — the goal is: tool name + enough args to know what was called + enough result to know what happened. A `Bash` call with `go test ./...` that errored is ~100 bytes. A `Read` call on a 5000-line file doesn't need the full output — just knowing the file was read and the first few lines of result is enough.

## Decision: Sonnet-Driven Dedup via SBIA Decision Tree

### The Problem with Binary Dedup

"Is this a duplicate?" is too simplistic. Each SBIA dimension can vary independently, and the combination determines the correct disposition. The user may have changed their mind, technology may have changed, or the situation nuance may justify different actions.

### Revised Extraction Flow

1. **Detect** correction (fast-path keywords from `detect_fast_path_keywords` + Haiku classification via `detect_haiku` prompt)
2. **Retrieve context** — current session transcript tail (`context_byte_budget`, SBIA strip mode — tool args truncated to `context_tool_args_truncate`, results to `context_tool_result_truncate`)
3. **Find candidates** — BM25 on existing memories using correction message + transcript context as query. Return top `extract_candidate_count_min` to `extract_candidate_count_max` candidates scoring ≥ `extract_bm25_threshold`. If fewer than min score above threshold, return however many do (including zero).
4. **Sonnet call** (via `extract_sonnet` prompt) — correction + conversation context + candidate memories → Sonnet extracts SBIA fields from the correction, then walks the decision tree against each candidate → outputs SBIA fields + per-candidate disposition

One Sonnet call handles both extraction and dedup. With zero candidates, Sonnet extracts SBIA fields and the disposition is STORE.

### SBIA Similarity Decision Tree

When a correction arrives and similar existing memories are found:

```
1. How similar is the Situation?
   ├── Same situation
   │   ├── Same behavior
   │   │   ├── Same impact, same action → DUPLICATE (don't create)
   │   │   │   └── Why are we here? Surfacing or listening failure?
   │   │   │       ├── Memory wasn't surfaced → retrieval problem
   │   │   │       └── Memory was surfaced but ignored → effectiveness problem
   │   │   ├── Same impact, different action → CONTRADICTION
   │   │   │   ├── User changed their mind → supersede old memory
   │   │   │   ├── Tech/policy changed → supersede with updated context
   │   │   │   └── Genuine disagreement → ask user which to keep
   │   │   └── Different impact, different action → REFINEMENT
   │   │       (Unusual — same situation + behavior yielding different impact.
   │   │        Flag for user review.)
   │   └── Different behavior
   │       └── Different lesson in same situation → STORE BOTH
   │           (Two different mistakes possible in the same context)
   ├── Similar situation (related but not identical)
   │   ├── Same behavior
   │   │   ├── Same impact → POTENTIAL GENERALIZATION
   │   │   │   (Consider merging into a broader situation description)
   │   │   └── Different impact → LEGITIMATE SEPARATE MEMORIES
   │   │       (Situation nuance changes the impact; different actions warranted)
   │   └── Different behavior → STORE BOTH (independent lessons)
   └── Different situation → STORE (no relationship)
```

### Disposition Outcomes

| Outcome                 | System Action                                                     |
| ----------------------- | ----------------------------------------------------------------- |
| **DUPLICATE**           | Don't create. Log surfacing/listening failure for self-diagnosis. |
| **CONTRADICTION**       | Surface to user for resolution. Supersede or keep both.           |
| **REFINEMENT**          | Flag for user review (unusual case).                              |
| **GENERALIZATION**      | Merge into broader situation description.                         |
| **LEGITIMATE SEPARATE** | Store both. Situation nuance justifies different actions.         |
| **STORE BOTH**          | Independent lessons, no meaningful overlap.                       |
| **STORE**               | No similar memories found. Write directly.                        |

### Self-Diagnosis on DUPLICATE

When the user re-teaches the same lesson, the system should investigate why the existing memory failed:

- **Not surfaced:** The memory exists but wasn't retrieved. Retrieval problem — situation text didn't match the current context via BM25. Potential fix: broaden the situation description.
- **Surfaced but not followed:** The memory was injected but the agent didn't follow it. Effectiveness problem — increment `not_followed_count`. May need escalation (e.g., promote to CLAUDE.md or rules).

This turns duplicate corrections into system feedback, not just redundant data.

## Decision: Surface via Situation Matching at UserPromptSubmit

### Behavior = Decision Logic, Not Tool Calls

The "behavior" in SBIA isn't the tool call — it's the decision logic that led to it. "Invoking `go test` directly" is the observable symptom, but the behavior is "deciding to use raw Go commands instead of the project's build system." The same faulty decision could manifest as `go test`, `go vet`, `go build` — different tools, same underlying behavior. Some behaviors aren't tool calls at all (not invoking a skill, making an architectural choice in code).

This means matching on literal tool commands (PreToolUse) is too narrow. The primary intervention point is **UserPromptSubmit** — surface situation-matched guidance _before_ the LLM starts reasoning, targeting the decision layer.

### Two-Stage Model (Stage 1 only for now)

| Stage       | Hook             | Matches On                        | Purpose                                          | Priority |
| ----------- | ---------------- | --------------------------------- | ------------------------------------------------ | -------- |
| **Stage 1** | UserPromptSubmit | User message → `situation` fields | Proactive guidance before LLM acts               | **Now**  |
| Stage 2     | PreToolUse       | Tool + args → `behavior` fields   | Guardrail for mechanically detectable violations | Future   |

Stage 2 (PreToolUse) is a narrow safety net for the most literally matchable cases. Defer to follow-up work.

### Surfacing Pipeline

**Current:** BM25 on `SearchText()` = title + content + principle + keywords + concepts. Top 2 results, 250-token budget, display `principle` only.

**SBIA pipeline:**

```
1. Build query context:
   - User prompt (always)
   - Recent transcript context (up to `context_byte_budget`)
   BM25 needs more than just the latest message — "do it" matches
   nothing, but the preceding conversation about "running tests"
   would match the targ memory.

2. BM25 on SBIA text → top `surface_candidate_count_min` to
   `surface_candidate_count_max` candidates (score ≥ `surface_bm25_threshold`)
   If fewer than min score above threshold, return however many do.

3. project_scoped hard filter
   Exclude memories scoped to a different project.

4. Irrelevance penalty on BM25 scores
   Half-life of `surface_irrelevance_half_life`.

5. Cold-start budget (max `surface_cold_start_budget` unproven per invocation)
   Unproven = never surfaced. Proven memories bypass this limit.

6. Haiku semantic gate (single batched call, prompt from `surface_gate_haiku`):
   Input: query context + candidate SBIA fields
   → Returns passing subset, or empty if none match.

   This fixes the core problem — BM25 keyword overlap surfaces
   memories in wrong contexts. Haiku asking "is this situation
   actually happening?" catches what BM25 can't.

7. Surface passing candidates with full SBIA fields
   No token budget — all four fields for each passing memory.
   The top-level LLM has the richest context to make the final
   relevance decision. Surface 0 memories if none pass the gate.
```

**Display format:** All four SBIA fields per memory, wrapped in the `surface_injection_preamble` prompt. Example output:

```
These memories may apply to your current task. Apply a memory
only if its situation matches what you're doing:

1. [use-targ]
   Situation: When running tests, builds, or lint in a targ project
   Behavior to avoid: Invoking go test, go build, go vet directly
   Impact if ignored: Bypasses coverage thresholds and lint rules
   Action: Use targ test, targ check-full, targ build
```

The preamble shapes how the LLM interprets and applies memories — it's as influential as any other pipeline prompt and must be configurable via `[prompts]`.

## Decision: Automated Evaluate via Pending Evaluation in Memory TOML

### The Problem with LLM Self-Report

The current evaluate mechanism relies on the LLM calling `engram feedback --name <name> --relevant|--irrelevant --used|--notused` after each turn. This is the LLM grading its own homework — no independent verification that the action was actually taken or the behavior was actually avoided.

### Design: Pending Evaluation in Memory File

Store evaluation state in the memory file itself — no separate log file.

**Invariant: Memory files are only ever read and written by `engram` commands.** The LLM never directly edits memory TOML. All pipeline operations (surface, evaluate, maintain, correct) go through engram CLI commands that handle concurrency, validation, and atomic writes.

**At surface time:** `engram surface` writes a pending evaluation section into each surfaced memory's TOML:

```toml
[[pending_evaluations]]
surfaced_at = "2026-03-29T12:00:00Z"
user_prompt = "run the tests"
session_id = "abc123"
project_slug = "engram"
```

Multiple agents or consecutive turns can surface the same memory before evaluation runs. TOML array of tables handles this — each entry is independent.

**At stop hook (async):** `engram evaluate` processes pending evaluations for this session:

1. Read the transcript delta (agent's response after surfacing)
2. Haiku call (via `evaluate_haiku` prompt): assess situation relevance and action compliance
3. Increment counters and remove this session's pending entry (other sessions' entries remain)

### Simplified Counters

The current model has five counters (surfaced, followed, contradicted, ignored, irrelevant). SBIA simplifies to three:

| Counter        | Meaning                                 | Haiku Assessment                                              |
| -------------- | --------------------------------------- | ------------------------------------------------------------- |
| `followed`     | Situation matched, action was taken     | "Was the situation relevant? Yes. Was the action taken? Yes." |
| `not_followed` | Situation matched, action was not taken | "Was the situation relevant? Yes. Was the action taken? No."  |
| `irrelevant`   | Situation didn't match                  | "Was the situation relevant? No."                             |

**Why three, not five:** "Contradicted" and "ignored" collapse into `not_followed`. Whether the agent did the problematic behavior or something else entirely, the outcome is the same — the memory didn't work. The fix is the same — rewrite the action or escalate. The distinction adds complexity without actionable signal.

`surfaced_count` is retained — useful at a glance for understanding how often a situation arises. Not used in effectiveness calculation but available for diagnostics and adapt metrics.

### Why This Works

- **No separate log file** — the memory TOML is already the source of truth for tracking. Pending evaluation is just the in-flight version.
- **Atomic read-modify-write** — existing pattern handles concurrent access.
- **Async stop slot is free** — with learn eliminated, the 120s async stop hook is available for evaluation.
- **Haiku is cheap** — typically 1-2 surfaced memories per turn; one Haiku call per surfaced memory.
- **SBIA makes evaluation possible** — explicit `behavior` and `action` fields are checkable assertions, not vague principles.
- **LLM self-report dropped** — `engram feedback` calls become unnecessary. The stop hook evaluates automatically.

## Decision: Maintain — Effectiveness-Only, No Quadrants

### Why Quadrants Don't Work in SBIA

The current quadrant model uses two axes: surfacing frequency × effectiveness. But surfacing frequency measures **situation rarity**, not memory quality. A memory about a rare situation (e.g., "when migrating database schemas") might surface once in 50 sessions and be followed every time. That's a working memory, not a "hidden gem" needing broadening.

The "hidden gem" quadrant disappears — there's no action to take on a memory that's effective but rare. "Noise" (rarely surfaced + low effectiveness) is just "low effectiveness" — the rarely-surfaced part is irrelevant.

### Single-Axis Model: Effectiveness

#### Stored Counters

| Counter | Incremented when | Stored in memory TOML |
|---------|-----------------|----------------------|
| `surfaced_count` | Memory passes Haiku gate at surface time | Yes |
| `followed_count` | Evaluate: situation matched, action taken | Yes |
| `not_followed_count` | Evaluate: situation matched, action not taken | Yes |
| `irrelevant_count` | Evaluate: situation didn't match | Yes |

#### Derived Metrics (computed on read, not stored)

| Metric | Formula | What it tells you |
|--------|---------|-------------------|
| `effectiveness` | `followed_count / surfaced_count` | Overall — is this memory helping? |
| `not_followed_rate` | `not_followed_count / surfaced_count` | Is the action being ignored when relevant? |
| `irrelevant_rate` | `irrelevant_count / surfaced_count` | Is the situation description too broad? |

#### Maintain Decision Tree (checked in priority order)

| Priority | Condition | Diagnosis | Action |
|----------|-----------|-----------|--------|
| 1 | `surfaced_count < maintain_min_surfaced` | Insufficient data | Skip |
| 2 | `effectiveness < maintain_effectiveness_threshold` AND `irrelevant_rate ≥ maintain_irrelevance_threshold` | Situation wrong and action failing | Remove |
| 3 | `irrelevant_rate ≥ maintain_irrelevance_threshold` | Situation too broad | Narrow `situation` |
| 4 | `not_followed_rate ≥ maintain_not_followed_threshold` | Action not compelling or clear | Rewrite `action`, or escalate |
| 5 | `effectiveness ≥ maintain_effectiveness_threshold` | Working | Keep (no action) |

### Triage Actions

Each action is a dedicated `engram` CLI command. The `/memory-triage` skill runs these commands — the LLM never edits memory files directly.

| Command | When | What it does |
|---------|------|-------------|
| `engram triage remove` | Both situation and action failing | Delete memory file |
| `engram triage narrow` | High irrelevant rate | LLM rewrites `situation` via `maintain_rewrite` prompt |
| `engram triage rewrite` | High not_followed rate | LLM rewrites `action` via `maintain_rewrite` prompt |
| `engram triage escalate` | Persistent not_followed despite clear action | Promote `action` to CLAUDE.md/rules |
| `engram triage consolidate` | Similar situations across memories | LLM synthesizes via `maintain_consolidate` prompt |

### `/memory-triage` Skill Changes

The skill presents memories by maintain priority:

1. **Remove** — memories failing both thresholds (`maintain_effectiveness_threshold` + `maintain_irrelevance_threshold`). "Nothing is working. Remove?"
2. **Irrelevant** — memories exceeding `maintain_irrelevance_threshold`. "This memory is surfacing in wrong contexts. Narrow the situation description?"
3. **Not followed** — memories exceeding `maintain_not_followed_threshold`. "This memory is being surfaced when relevant but the agent isn't following it. Rewrite the action or escalate?"
4. **Consolidation** — memories with similar situations that could be merged.

### `/adapt` Redesign: Config Surface + Metrics + LLM Analysis

Replace the current 5-dimension analysis code with a simpler model: every tunable behavior is a parameter in `policy.toml`, every pipeline stage emits metrics, and a Sonnet call maps metric trends to parameter adjustments.

#### Tunable Parameters

All stored in `policy.toml`, readable and writable by the adapt pipeline:

```toml
[parameters]
# Detect
detect_fast_path_keywords = ["remember", "always", "never", "don't", "stop"]

# Context
context_byte_budget = 51200
context_tool_args_truncate = 200
context_tool_result_truncate = 500

# Extract + Dedup
extract_candidate_count_min = 3
extract_candidate_count_max = 8
extract_bm25_threshold = 0.3

# Surface
surface_candidate_count_min = 3
surface_candidate_count_max = 8
surface_bm25_threshold = 0.3
surface_cold_start_budget = 2
surface_irrelevance_half_life = 5

# Maintain
maintain_effectiveness_threshold = 50.0
maintain_min_surfaced = 5
maintain_irrelevance_threshold = 60.0
maintain_not_followed_threshold = 50.0

[prompts]
# Each LLM prompt stored here — versionable and tunable
detect_haiku = "..."              # "Is this user message a correction?"
extract_sonnet = "..."            # "Extract SBIA fields, walk dedup decision tree"
surface_gate_haiku = "..."        # "Which memories match this situation/behavior?"
surface_injection_preamble = "..."# Preamble injected with surfaced memories
evaluate_haiku = "..."            # "Was the situation relevant? Was the action taken?"
adapt_sonnet = "..."              # "Analyze metrics, propose parameter adjustments"
maintain_rewrite = "..."          # "Rewrite this action/situation for clarity"
maintain_consolidate = "..."      # "Synthesize these similar memories into one"
```

#### Pipeline Failure Modes and Observable Signals

| Stage | Failure Mode | Observable Signal | Likely Parameter |
|-------|-------------|-------------------|------------------|
| **Detect** | Correction missed | Duplicate correction (user re-teaches) | `detect_haiku` prompt; fast-path keywords |
| **Detect** | Non-correction triggers extraction | High extraction rate + low follow-through on new memories | `detect_haiku` prompt |
| **Extract** | Vague situation | High `irrelevant` rate on new memories | `extract_sonnet` prompt |
| **Extract** | Weak action | High `not_followed` rate on new memories | `extract_sonnet` prompt |
| **Extract** | Wrong dedup disposition | Duplicate corrections (missed dedup) or missing memories | `extract_sonnet` prompt; BM25 threshold; candidate count |
| **Surface** | Poor BM25 candidates | Relevant memory exists but wasn't surfaced (user re-corrects) | `surface_bm25_threshold`; candidate count |
| **Surface** | Haiku gate false positive | High `irrelevant` rate on surfaced memories | `surface_gate_haiku` prompt |
| **Surface** | Haiku gate false negative | Hard to observe; proxy: user re-corrects when memory exists | `surface_gate_haiku` prompt |
| **Surface** | Unproven memories crowd proven | High irrelevance on unproven memories | `surface_cold_start_budget`; `surface_irrelevance_half_life` |
| **Evaluate** | Haiku misclassifies outcome | Requires human audit (no automated signal) | `evaluate_haiku` prompt |
| **Maintain** | Wrong diagnosis | Maintain action doesn't improve effectiveness | `maintain_*` thresholds |

#### Adapt Flow: Sonnet Analyzes Metrics → Proposes Parameter Changes

```
1. Collect metrics (rolling window, per-session counters)
2. Sonnet call: "Here are the metrics over the last N sessions.
   Here are the current parameters. What's trending wrong
   and what would you adjust?"
3. Sonnet returns proposed parameter changes with rationale
4. User approves/rejects via /adapt skill
5. On approval: snapshot current corpus metrics as before-state
6. After measurement window: compare before/after → validate or revert
```

No custom analysis dimensions in Go. The analysis logic is a Sonnet prompt — flexible, evolvable, and simple to maintain.

## Skill Operations and Pipeline Mapping

### /recall

**LLM does directly:** Nothing — runs command, interprets text output.

**Command:** `engram recall [--query "..."]`

| Operation                                | Files                                         | Pipeline Stage        |
| ---------------------------------------- | --------------------------------------------- | --------------------- |
| Reads session transcripts                | `~/.claude/projects/{slug}/*.jsonl`           | —                     |
| Reads memory files (query mode)          | `{dataDir}/memories/*.toml`                   | Surface (retrieval)   |
| Reads CLAUDE.md + rules (suppression)    | `~/.claude/CLAUDE.md`, `~/.claude/rules/*.md` | Surface (suppression) |
| Writes surfaced_count + last_surfaced_at | `{dataDir}/memories/*.toml`                   | Surface (tracking)    |
| API: Haiku extract relevant content      | Anthropic Messages API (query mode only)      | —                     |

**SBIA impact:** Low. `SearchText()` changes input fields but the skill just runs a command.

### /memory-triage

**LLM does directly:** Reads triage output from session-start hook (injected into prompt). Presents to user. Runs commands based on user decisions.

**Current command:** `engram apply-proposal --action <action> --memory <path> [--keywords/--fields]`

| Action             | Reads                   | Writes                                                        | API                          | Fields Referenced                                   |
| ------------------ | ----------------------- | ------------------------------------------------------------- | ---------------------------- | --------------------------------------------------- |
| `remove`           | Target TOML             | **Deletes** file                                              | None                         | —                                                   |
| `broaden_keywords` | Target TOML             | Appends to `keywords`                                         | None                         | **`keywords`**                                      |
| `rewrite`          | Target TOML             | Updates fields                                                | None                         | **`title`, `content`, `principle`, `anti_pattern`** |
| `refine_keywords`  | Target TOML             | Removes/adds keywords, clears `irrelevant_queries`            | None                         | **`keywords`, `irrelevant_queries`**                |
| `consolidate`      | Survivor + member TOMLs | Overwrites survivor; archives members to `{dataDir}/archive/` | Haiku (synthesize principle) | **`principle`** (synthesized)                       |

**SBIA redesign:** `broaden_keywords` and `refine_keywords` are eliminated (no keywords). Remaining actions:

| Action        | Reads                   | Writes                                                        | API                       | Fields Referenced                               |
| ------------- | ----------------------- | ------------------------------------------------------------- | ------------------------- | ----------------------------------------------- |
| `remove`      | Target TOML             | **Deletes** file                                              | None                      | —                                               |
| `rewrite`     | Target TOML             | Updates fields                                                | None                      | **`situation`, `behavior`, `impact`, `action`** |
| `consolidate` | Survivor + member TOMLs | Overwrites survivor; archives members to `{dataDir}/archive/` | Haiku (synthesize action) | **`action`** (synthesized)                      |
| `escalate`    | Target TOML             | Promotes to CLAUDE.md/rules                                   | None                      | **`action`**                                    |

### /adapt

**LLM does directly:** Runs status command, interprets output, presents to user. Runs approve/reject/retire.

**Command:** `engram adapt --data-dir "$ENGRAM_DATA_DIR" [--approve/--reject/--retire <id>]`

| Action  | Reads                                                  | Writes                           | API  |
| ------- | ------------------------------------------------------ | -------------------------------- | ---- |
| status  | `policy.toml`                                          | Nothing                          | None |
| approve | `policy.toml` + all `memories/*.toml` (corpus metrics) | `policy.toml` (status, snapshot) | None |
| reject  | `policy.toml`                                          | `policy.toml`                    | None |
| retire  | `policy.toml`                                          | `policy.toml`                    | None |

**SBIA redesign:** Keyword-focused proposal dimensions are removed. Extraction guidance shifts to situation-quality advice. Policy dimensions update to SBIA-focused concerns (situation breadth, action clarity).

## Hook Operations and Pipeline Mapping

### SessionStart (`session-start.sh`)

**Sync output:** Static system reminder ("Say /recall") + correction instructions.

**Async background fork:**

1. Rebuilds binary if stale
2. Runs `engram maintain` → JSON proposals
3. Reads `policy.toml` for adaptation proposal count
4. Writes `~/.claude/engram/pending-maintenance.json` (consumed later by UserPromptSubmit)

| Command           | Reads                                | Writes           | API                                   |
| ----------------- | ------------------------------------ | ---------------- | ------------------------------------- |
| `engram maintain` | All `memories/*.toml`, `policy.toml` | stdout JSON only | Haiku (optional, rewrite suggestions) |

**Current fields read by maintain:** `surfaced_count`, `followed_count`, `contradicted_count`, `ignored_count`, `irrelevant_count`, `keywords`, `principle`, `title`, `anti_pattern`, `confidence`

**SBIA fields read by maintain:** `surfaced_count`, `followed_count`, `not_followed_count`, `irrelevant_count`, `situation`, `behavior`, `impact`, `action`, `project_scoped`

### UserPromptSubmit (`user-prompt-submit.sh`)

Consumes `pending-maintenance.json` (atomic read + delete), then runs two commands:

#### 1. `engram correct --message "$USER_MESSAGE"`

**Pipeline stage:** Detect → Context Retrieval → SBIA Extraction → Dedup → Write

**Current:**

| Operation | Files                                             | Fields                                                                                                                                     |
| --------- | ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Reads     | User message + optional transcript                | —                                                                                                                                          |
| Writes    | New `memories/<slug>.toml` if correction detected | `title`, `content`, `keywords`, `concepts`, `principle`, `anti_pattern`, `rationale`, `observation_type`, `confidence`, `generalizability` |
| API       | Haiku — classifies into tier A/B/C/null           | JSON schema defines output fields                                                                                                          |

**SBIA redesign:**

| Operation | Files                                                    | Fields                                                          |
| --------- | -------------------------------------------------------- | --------------------------------------------------------------- |
| Detect    | User message (fast-path keywords + Haiku classification)                  | —                                                               |
| Context   | Current session transcript tail (`context_byte_budget`, SBIA strip mode)  | —                                                               |
| Candidates | BM25 on correction + context → top candidates from `memories/*.toml` (per `extract_*` config) | —                                                               |
| Extract+Dedup | Correction + context + candidates → **Sonnet** extracts SBIA + disposition per candidate | `situation`, `behavior`, `impact`, `action` + per-candidate disposition |
| Write     | New `memories/<slug>.toml` (if disposition is not DUPLICATE)              | `situation`, `behavior`, `impact`, `action` + tracking metadata |

**SBIA impact: High.** This becomes the primary (and only) extraction path. Haiku detects; Sonnet extracts and deduplicates in one call.

#### 2. `engram surface --mode prompt --message "$USER_MESSAGE" --format json`

**Pipeline stage:** Surface (retrieval + tracking)

**Current:**

| Operation | Files                                                                                                  | Fields                                                                                                                                                                                                                                                   |
| --------- | ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Reads     | All `memories/*.toml`, `policy.toml`, `surfacing-log.jsonl`, `~/.claude/CLAUDE.md` + rules             | Matching: `SearchText()` (title+content+principle+keywords+concepts). Ranking: `surfaced_count`, `followed_count`, `contradicted_count`, `ignored_count`, `irrelevant_count`, `irrelevant_queries`, `generalizability`, `confidence`, `last_surfaced_at` |
| Writes    | Increments `surfaced_count` + `last_surfaced_at` on matched memories; appends to `surfacing-log.jsonl` | —                                                                                                                                                                                                                                                        |
| API       | None                                                                                                   | —                                                                                                                                                                                                                                                        |
| Displays  | `principle` (+ filename slug) to user                                                                  | —                                                                                                                                                                                                                                                        |

**SBIA redesign:**

| Step | Operation | Details |
| ---- | --------- | ------- |
| 1. Query context | Build from user prompt + transcript tail | Up to `context_byte_budget` of transcript |
| 2. BM25 candidates | Top candidates from `memories/*.toml` (per `surface_*` config) | Matching on `SearchText()` (situation+behavior+impact+action) |
| 3. Filters | `project_scoped` hard filter, irrelevance penalty (`surface_irrelevance_half_life`), cold-start budget (`surface_cold_start_budget`) | Pre-Haiku filtering on cheap signals |
| 4. Haiku gate | Single batched call: query context + candidate SBIA fields | "Which memories describe a situation the agent is in, with a behavior it might exhibit?" → returns passing subset or empty |
| 5. Track | Increments `surfaced_count` + `last_surfaced_at`; writes `[[pending_evaluations]]` entry | Only for memories that pass the gate |
| 6. Display | Full SBIA fields for each passing memory | No token budget — all four fields per memory, presented as candidates |
| API | Haiku (semantic gate via `surface_gate_haiku` prompt) | One call per prompt, candidates batched |

**LLM instructions injected:** `<system-reminder>` with candidate memories (situation, behavior, impact, action per memory); instruction to apply only if situation matches current task; correction notification if detected.

### Stop surface (`stop-surface.sh`)

**Pipeline stage:** Surface (stop mode) — same as prompt-mode surface but matches against agent output instead of user message. Blocks response if conflicting memories found.

| Command                      | Reads                                    | Writes                      | API  |
| ---------------------------- | ---------------------------------------- | --------------------------- | ---- |
| `engram surface --mode stop` | Transcript JSONL + all `memories/*.toml` | Same as prompt-mode surface | None |

**SBIA impact: Low.** Same as prompt-mode surface.

### Stop async (`stop.sh`)

**Current pipeline stage:** Extract → Dedup → Write (end-of-turn learning)

| Command                           | Reads                                                                         | Writes                                                                                                                                              | API                       |
| --------------------------------- | ----------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| `engram flush` (→ `engram learn`) | Transcript JSONL (incremental via `learn-offset.json`), all `memories/*.toml` | New `memories/<slug>.toml` per surviving candidate; `learn-offset.json`; adaptation proposals in `policy.toml`; deletes stale `surfacing-log.jsonl` | Haiku — extraction prompt |

**SBIA redesign:** Batch extraction eliminated. This hook becomes the **evaluate** stage:

| Command           | Reads                                                                                        | Writes                                                                                                                  | API                                                         |
| ----------------- | -------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| `engram evaluate` | Transcript JSONL, all `memories/*.toml` with `[[pending_evaluations]]` matching this session | Increments `followed_count`/`not_followed_count`/`irrelevant_count`; removes consumed `[[pending_evaluations]]` entries | Haiku — evaluates situation relevance and action compliance |

### Hook Impact Summary

| Hook                           | Current                                | SBIA Redesign                                                                         |
| ------------------------------ | -------------------------------------- | ------------------------------------------------------------------------------------- |
| **SessionStart**               | Runs `maintain`, writes pending file   | Field names change in proposals. Otherwise similar.                                   |
| **UserPromptSubmit (correct)** | Haiku classifies single message        | **Primary extraction path.** Haiku detects (`detect_haiku`) → transcript context (`context_byte_budget`, SBIA strip) → BM25 candidates (`extract_*` config) → Sonnet extracts SBIA + dedup disposition (`extract_sonnet`). |
| **UserPromptSubmit (surface)** | BM25 on SearchText, displays principle | BM25 candidates (`surface_*` config) → Haiku semantic gate (`surface_gate_haiku`) → surface full SBIA fields for passing memories. No token budget. |
| **Stop (surface)**             | Surface on agent output                | Same as surface above.                                                                |
| **Stop (evaluate)**            | Full batch extraction pipeline         | **Replaced.** Evaluates pending surfacings via Haiku; updates counters.               |

## Resolved Questions

1. **Keywords:** Dropped. BM25 on SBIA text for both surfacing and candidate retrieval in dedup.
2. **Tier C:** Dropped. Corrections are inherently behavioral. No contextual facts in SBIA model.
3. **Dedup strategy:** Sonnet-driven via SBIA decision tree. BM25 finds 3-8 candidates (score ≥ 0.3), then Sonnet evaluates each SBIA dimension independently and determines disposition per candidate. One Sonnet call handles both extraction and dedup. Duplicate detections trigger self-diagnosis (surfacing vs. listening failure).
4. **Evaluate counters:** Simplified to three: `followed`, `not_followed`, `irrelevant`. "Contradicted" and "ignored" collapse — the distinction isn't actionable. Automated via Haiku at stop hook; LLM self-report dropped.
5. **Quadrants:** Eliminated. Surfacing frequency measures situation rarity, not memory quality. Maintain operates on effectiveness only, with counter breakdown diagnosing which SBIA field to fix.
6. **Triage actions:** `broaden_keywords`/`refine_keywords` removed. Replaced by: rewrite `action`, narrow/broaden `situation`, escalate, consolidate, remove.
7. **Adapt redesign:** Replace 5-dimension Go analysis code with: config surface (all tunable parameters in `policy.toml`), per-stage metrics collection, and a Sonnet call that maps metric trends to parameter adjustments. LLM prompts are stored in config and tunable by adapt. No custom analysis dimensions — Sonnet analyzes metrics directly.
8. **Candidate counts:** Configurable via `extract_candidate_count_*` and `surface_candidate_count_*` (default 3-8). Score threshold via `*_bm25_threshold` (default 0.3). Fewer than min is fine if the corpus doesn't have close matches.
9. **Surfacing semantic gate:** Haiku validates BM25 candidates before injection via `surface_gate_haiku` prompt. Single batched call with query context (user prompt + transcript) + candidate SBIA fields. Prevents false-positive surfacing from keyword overlap without situational match.
10. **Token budget:** Dropped. Surface full SBIA fields (all four) for every memory that passes the Haiku gate. The top-level LLM decides final relevance from the situation descriptions.
11. **Context retrieval:** Stripped transcript tail (`context_byte_budget`, default 50KB, SBIA strip mode — includes truncated tool calls). Sonnet's attention is the semantic gate, not a heuristic boundary detector. Byte budget is a ceiling for pathological sessions.
12. **SBIA strip mode:** `StripConfig` on `Strip` function. SBIA mode includes tool name, truncated args (`context_tool_args_truncate`, default ~200 chars), result status, truncated result body (`context_tool_result_truncate`, default ~500 chars). Tool calls are Behavior evidence; tool results are Impact evidence. Recall mode continues to drop tool blocks.
13. **Staleness check:** Dropped. If a working memory becomes outdated, the user will correct it and the correction pipeline handles the update. No timer-based nagging.
14. **`surfaced_count`:** Kept as stored counter — useful at a glance. Derived metrics (`effectiveness`, `not_followed_rate`, `irrelevant_rate`) use it as denominator.
15. **Maintain decision tree:** All thresholds configurable via `maintain_*` parameters. Priority order: insufficient data → remove → narrow situation → rewrite action → keep.
16. **All parameters in config:** Every tunable value lives in `policy.toml` `[parameters]` section. Pipeline descriptions reference parameter names, not hardcoded values. Defaults are set in config, not in code.

## Resolved: Generalizability, Tiers, Migration

### Generalizability → `project_scoped: bool`

Replace the 1-5 generalizability scale with a boolean. Default is **not project-scoped** (universal). Most corrections transfer across projects. Only mark `project_scoped = true` when the advice is meaningless outside this specific project.

- **Not project-scoped (default):** Surfaces in all projects. "Don't remove t.Parallel() to fix test failures" applies everywhere.
- **Project-scoped:** Hard filter — memory does not surface outside its origin project. "This project uses targ" is genuinely project-specific.
- **Sonnet extraction bias:** "Only mark as project_scoped if the advice is meaningless outside this specific project. Most corrections transfer. When in doubt, leave it universal."

No graduated penalty scoring. Binary filter.

### Tiers → Dropped

The A/B/C tier system is eliminated. All memories are corrections extracted via SBIA. The A vs B distinction (explicit vs. inferred) doesn't drive different behavior in the SBIA model — both produce the same four fields. Ranking is driven by effectiveness data, not a priori classification.

### Migration

One-time Sonnet migration script:

| Current Tier                  | Action                                                                                                         |
| ----------------------------- | -------------------------------------------------------------------------------------------------------------- |
| **A** (explicit instructions) | Sonnet converts to SBIA fields. These have the clearest principle/anti_pattern to map.                         |
| **B** (inferred corrections)  | Archive. Weak signal, not worth converting. New corrections will be properly extracted with full SBIA context. |
| **C** (contextual facts)      | Archive. Already filtered out of extraction pipeline.                                                          |

Failures (memories Sonnet can't meaningfully convert to SBIA) go to archive.

## All Open Questions Resolved

No remaining open questions. The full SBIA pipeline design is complete:

```
Detect → Context → Extract+Dedup (Sonnet) → Write → Surface → Evaluate → Maintain
```

### Final SBIA Memory Schema

```toml
# Content (SBIA)
situation = "When running tests, builds, or lint in a project that uses targ"
behavior = "Invoking go test, go build, go vet directly"
impact = "Bypasses coverage thresholds and lint rules, leading to false confidence"
action = "Use targ test, targ check-full, targ build"

# Scope
project_scoped = false
project_slug = "engram"

# Tracking
surfaced_count = 0
followed_count = 0
not_followed_count = 0
irrelevant_count = 0

# Pending evaluations (written at surface, consumed at stop)
# [[pending_evaluations]]
# surfaced_at = "2026-03-29T12:00:00Z"
# user_prompt = "run the tests"
# session_id = "abc123"
# project_slug = "engram"

# Provenance
source_type = "correction"
content_hash = "abc123"
created_at = "2026-03-29T12:00:00Z"
updated_at = "2026-03-29T12:00:00Z"

# Relationships
# [[absorbed]]
# from = "path/to/merged.toml"
# merged_at = "2026-03-29T12:00:00Z"
```
