# SBIA Feedback Model for Engram Extraction

**Status:** Brainstorming (in progress)
**Date:** 2026-03-29

## Problem Statement

Engram's current memory model captures *what to do* (principle) and *what not to do* (anti_pattern), but doesn't explicitly model *when* the correction applies. Keywords attempt to proxy for situation context, but they're term-level matches that frequently surface in wrong contexts (13 memories currently flagged for irrelevant surfacing due to overly generic keywords).

The SBIA framework (Situation, Behavior, Impact, Action) — a validated feedback model from organizational psychology — could provide better structure for behavioral correction by anchoring each memory to the observable situation where it applies.

## Core Insight

Engram already has raw memory via session logs and `/recall`. What makes structured memories valuable is recognizing **similar situations** and applying the right **corrective action**. The current schema stores the correction but not the situation that triggers it.

## Current Schema vs. SBIA Mapping

| SBIA Dimension | Current Field | Quality of Fit |
|----------------|--------------|----------------|
| **Situation** (when does this apply?) | `keywords` + `concepts` | **Poor.** Keywords are topic tags, not scenario descriptors. No field captures "when are you in this situation?" |
| **Behavior** (what's the default/wrong action?) | `anti_pattern` | **Partial.** Captures "what not to do" but not "what you naturally default to." Empty for tier C, optional for B. |
| **Impact** (what goes wrong?) | `rationale` | **Partial.** Explains why the principle matters but doesn't explicitly describe the negative outcome of the default behavior. |
| **Action** (what to do instead?) | `principle` | **Good.** Strongest mapping — "what to do instead." |

**The critical gap is Situation.** No field answers: *"What would the agent be doing when this correction applies?"*

## Research Support

### Encoding Specificity Principle (Tulving & Thomson, 1973)

Memory retrieval is most effective when the retrieval cue matches the encoding context. If you encode *what the agent was doing when the correction happened*, you can match on *what the agent is doing now*. Current keywords match on topic; SBIA would match on activity context.

### Case-Based Reasoning (Kolodner 1993, Aamodt & Plaza 1994)

AI systems that retrieve past solutions by matching *problem situations* consistently outperform those that match on abstract rules or keyword indices. CBR explicitly indexes cases by situational features — goal, constraints, what went wrong. SBIA moves engram from rule-based retrieval toward case-based retrieval.

### SBI/SBIA Model (Center for Creative Leadership)

One of the most validated feedback frameworks in organizational psychology. Its power comes from anchoring feedback to *observable behavior in a specific situation*, making it actionable rather than abstract. The "A" (Action/alternative) extension maps directly to engram's use case.

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
keywords = ["running-tests-directly", "go-toolchain-invocation", "build-system-bypass"]
confidence = "A"
```

Note how the SBIA version makes the keywords activity-level ("running-tests-directly") rather than topic-level ("go-test"), which is what the current extraction prompt already *asks for* but keywords alone can't capture.

## Decision: Option A (Full Restructure)

Option B (layer) rejected — extra fields is not the goal. Options A and C differ only in whether `title`, `content`, `observation_type`, and `concepts` survive alongside SBIA fields. Since SBIA fields *are* the content, those fields are redundant. Go with A: simplify the data model and what we extract/store.

## Current Field Usage Across Pipeline

Traced every field through every pipeline stage to understand migration impact.

| Field | Extract | Dedup | Write | Surface | Evaluate | Maintain/Signal |
|-------|---------|-------|-------|---------|----------|-----------------|
| **title** | writes | - | persists | SearchText (BM25) | - | display, BM25 adapter, cluster ID |
| **content** | writes | - | persists | SearchText (BM25) | - | - |
| **principle** | writes | - | persists | SearchText (BM25), **displayed to user** | - | display, BM25 adapter, rewrite target, consolidation text |
| **anti_pattern** | writes | - | persists | - (not in SearchText) | - | signal apply (rewrite target) |
| **rationale** | writes | - | persists | - (not in SearchText, not displayed) | - | - |
| **keywords** | writes | **primary axis** (>50% overlap) | persists | SearchText (BM25), common-keyword filter | - | consolidation text (`keywords + principle`) |
| **concepts** | writes | - | persists | SearchText (BM25) | - | - |
| **observation_type** | writes | - | persists | - | - | render only (display on creation) |
| **filename_summary** | writes | - | filename gen | - | - | - |
| **generalizability** | writes | filter (<2) | persists | GenFactor (cross-project penalty) | - | migration |
| **confidence/tier** | writes | - | persists | frecency tier boost (A=1.2, B=0.2) | - | quadrant diagnosis |

### Key Findings

1. **`rationale`** — write-only dead end. Extracted, persisted, never read by any downstream stage. Nothing surfaces it, searches it, or displays it.

2. **`observation_type`** — only read by `render.go` to display "Type: correction" at creation time. Never used for retrieval, matching, or maintenance.

3. **`concepts`** — only consumed by `SearchText()` for BM25. No other pipeline stage reads them. Secondary retrieval signal, redundant if situation text provides the same terms.

4. **`anti_pattern`** — not in `SearchText()`, so it doesn't help retrieval. Only consumed by `signal/apply.go` for rewrites during maintenance.

5. **`keywords`** — the one structurally load-bearing field. Dedup depends on keyword set overlap (>50%). Surface uses them in SearchText. Signal consolidation uses them. **Dropping keywords means replacing the dedup strategy** (likely with TF-IDF cosine similarity on SBIA text, which already exists in the codebase).

6. **`principle`** — the most consumed content field. It's the **only content field displayed to users** during surfacing (filename slug + principle). Also used in consolidation matching via signal/bm25_adapter.

### Pipeline Impact of SBIA Restructure

| Pipeline Stage | Current Dependency | SBIA Equivalent | Migration Difficulty |
|----------------|-------------------|-----------------|---------------------|
| **Extract** | Produces 10+ fields | Produces 4 SBIA fields + metadata | Low — simpler prompt |
| **Dedup** | Keyword set overlap >50% | TF-IDF cosine on SBIA text (infra exists) | Medium — new strategy |
| **Write** | MemoryRecord with all fields | MemoryRecord with SBIA fields | Low — fewer fields |
| **Surface (retrieval)** | BM25 on `SearchText()` (title+content+principle+keywords+concepts) | BM25 on `SearchText()` (situation+behavior+impact+action) | Low — same mechanism, different input |
| **Surface (display)** | Shows `principle` to user | Shows `action` to user | Low — rename |
| **Evaluate** | Tracks counters (orthogonal to content) | No change | None |
| **Maintain/Signal** | Reads title, principle, anti_pattern, keywords for consolidation/rewrite | Reads action, behavior, situation for consolidation/rewrite | Medium — field name changes |

## Decision: Corrections-Only Extraction with Sonnet

### Revised Extraction Model

Replace the dual extraction paths (real-time `correct` + batch `learn`) with a single, richer correction path:

**Current flow:**
1. UserPromptSubmit: Haiku classifies user message → writes memory with whatever it can infer from one sentence
2. Stop async: Haiku scans full transcript → extracts batch learnings → dedup → write

**New flow:**
1. UserPromptSubmit: Detect correction (fast-path keywords + Haiku classification) → pull session context via recall-like code → **Sonnet** extracts all four SBIA dimensions from correction + surrounding conversation → dedup → write
2. Stop async: **Eliminated** (or reduced to surfacing-log cleanup)

### Rationale

A correction message like "always use targ" contains only the **Action**. The Situation ("when running tests/builds"), Behavior ("invoking go test directly"), and Impact ("bypasses coverage/lint") are in the *surrounding conversation* — what Claude was doing, what went wrong, why the user intervened. Haiku operating on a single message can't reconstruct these dimensions. Sonnet with full conversation context can.

### Key Consequences

1. **Batch extraction (`learn`) eliminated.** The stop.sh async hook either goes away or becomes trivially simple (surfacing-log cleanup only). The entire `extract` package, `learn` package, and `flush` CLI command simplify dramatically or are removed.

2. **Sonnet replaces Haiku for extraction.** Higher quality, higher cost — but corrections are rare per session (typically 0-3), so the cost increase is negligible.

3. **Recall code reused for context retrieval.** The infrastructure to read and strip session transcripts already exists in the recall pipeline. Reuse it to pull the conversation window around the correction.

4. **Tier C eliminated.** Corrections are inherently behavioral (tier A/B). No contextual facts to classify. The tier C filtering that already exists in the learn pipeline becomes moot.

5. **Dedup volume drops.** With corrections-only extraction, far fewer candidates per session. TF-IDF cosine on SBIA text is sufficient for the lower volume (keyword-set overlap strategy no longer needed).

6. **Keywords dropped.** With SBIA text available for both retrieval (BM25 on SearchText) and dedup (TF-IDF cosine), the keywords array no longer serves a unique purpose.

### Future: Indirect Triggers

"What about X?" and similar patterns aren't explicit corrections but may signal implicit learnings worth extracting. These indirect triggers are out of scope for the initial SBIA restructure but worth revisiting once the correction-only path is stable. The detection mechanism (Haiku classification of non-fast-path messages) already exists and could be extended to flag these for deferred SBIA extraction.

## Decision: Sonnet-Driven Dedup via SBIA Decision Tree

### The Problem with Binary Dedup

"Is this a duplicate?" is too simplistic. Each SBIA dimension can vary independently, and the combination determines the correct disposition. The user may have changed their mind, technology may have changed, or the situation nuance may justify different actions.

### Revised Extraction Flow

1. **Detect** correction (fast-path keywords + Haiku classification)
2. **Retrieve context** — session conversation via recall-like code
3. **Find candidates** — TF-IDF cosine on existing memories (cheap, fast retrieval)
4. **Sonnet call** — correction + conversation context + similar existing memories → Sonnet walks the decision tree → outputs SBIA fields + disposition

One Sonnet call handles both extraction and dedup.

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

| Outcome | System Action |
|---------|---------------|
| **DUPLICATE** | Don't create. Log surfacing/listening failure for self-diagnosis. |
| **CONTRADICTION** | Surface to user for resolution. Supersede or keep both. |
| **REFINEMENT** | Flag for user review (unusual case). |
| **GENERALIZATION** | Consider merging into broader situation description. |
| **LEGITIMATE SEPARATE** | Store both. Situation nuance justifies different actions. |
| **STORE BOTH** | Independent lessons, no meaningful overlap. |
| **STORE** | No similar memories found. Write directly. |

### Self-Diagnosis on DUPLICATE

When the user re-teaches the same lesson, the system should investigate why the existing memory failed:

- **Not surfaced:** The memory exists but wasn't retrieved. Retrieval problem — situation text didn't match the current context via BM25. Potential fix: broaden the situation description.
- **Surfaced but ignored:** The memory was injected but the agent didn't follow it. Effectiveness tracking problem — increment contradicted/ignored count. May need escalation (e.g., promote to CLAUDE.md or rules).

This turns duplicate corrections into system feedback, not just redundant data.

## Decision: Surface via Situation Matching at UserPromptSubmit

### Behavior = Decision Logic, Not Tool Calls

The "behavior" in SBIA isn't the tool call — it's the decision logic that led to it. "Invoking `go test` directly" is the observable symptom, but the behavior is "deciding to use raw Go commands instead of the project's build system." The same faulty decision could manifest as `go test`, `go vet`, `go build` — different tools, same underlying behavior. Some behaviors aren't tool calls at all (not invoking a skill, making an architectural choice in code).

This means matching on literal tool commands (PreToolUse) is too narrow. The primary intervention point is **UserPromptSubmit** — surface situation-matched guidance *before* the LLM starts reasoning, targeting the decision layer.

### Two-Stage Model (Stage 1 only for now)

| Stage | Hook | Matches On | Purpose | Priority |
|-------|------|-----------|---------|----------|
| **Stage 1** | UserPromptSubmit | User message → `situation` fields | Proactive guidance before LLM acts | **Now** |
| Stage 2 | PreToolUse | Tool + args → `behavior` fields | Guardrail for mechanically detectable violations | Future |

Stage 2 (PreToolUse) is a narrow safety net for the most literally matchable cases. Defer to follow-up work.

### Surfacing Mechanism

**Current:** BM25 on `SearchText()` = title + content + principle + keywords + concepts.

**SBIA:** BM25 on `SearchText()` = situation + behavior + impact + action. The `situation` field is the richest matching signal — "when running tests in a project that uses targ" contains more retrieval-relevant terms than keywords `["targ", "build"]`.

**Display to LLM changes:** Currently shows `principle` (the action to take). With SBIA, surface the full context:
- **Situation** (so the LLM recognizes it's in the matching scenario)
- **Action** (what to do instead)
- Optionally **behavior** + **impact** (reinforces *why* — may help compliance)

### Extraction Context

The session transcript JSONL includes tool use entries, but the conversation usually *describes* what went wrong ("you ran go test directly", "you should have used brainstorming first"). Sonnet with conversation context can likely infer the behavior without needing raw tool call data. If insufficient, expand recall context to include tool use entries as a later optimization.

## Decision: Automated Evaluate via Pending Evaluation in Memory TOML

### The Problem with LLM Self-Report

The current evaluate mechanism relies on the LLM calling `engram feedback --name <name> --relevant|--irrelevant --used|--notused` after each turn. This is the LLM grading its own homework — no independent verification that the action was actually taken or the behavior was actually avoided.

### Design: Pending Evaluation in Memory File

Store evaluation state in the memory file itself — no separate log file.

**At surface time (UserPromptSubmit):** Write a pending evaluation section into each surfaced memory's TOML:

```toml
[[pending_evaluations]]
surfaced_at = "2026-03-29T12:00:00Z"
user_prompt = "run the tests"
session_id = "abc123"
project_slug = "engram"
```

Multiple agents or consecutive turns can surface the same memory before evaluation runs. TOML array of tables handles this — each entry is independent.

**At stop hook (async, replaces the eliminated learn path):** For each memory with pending evaluations matching this `session_id`:
1. Read the transcript delta (agent's response after surfacing)
2. Haiku call: "Given this memory (situation/behavior/action) and what the agent did (transcript delta): was the situation relevant? Was the action taken? Was the behavior avoided?"
3. Map result to counters and remove only this session's pending entry (other sessions' entries remain)

### Simplified Counters

The current model has five counters (surfaced, followed, contradicted, ignored, irrelevant). SBIA simplifies to three:

| Counter | Meaning | Haiku Assessment |
|---------|---------|-----------------|
| `followed` | Situation matched, action was taken | "Was the situation relevant? Yes. Was the action taken? Yes." |
| `not_followed` | Situation matched, action was not taken | "Was the situation relevant? Yes. Was the action taken? No." |
| `irrelevant` | Situation didn't match | "Was the situation relevant? No." |

**Why three, not five:** "Contradicted" and "ignored" collapse into `not_followed`. Whether the agent did the problematic behavior or something else entirely, the outcome is the same — the memory didn't work. The fix is the same — rewrite the action or escalate. The distinction adds complexity without actionable signal.

`surfaced_count` is dropped as a quality signal (see Maintain below) but retained as a raw counter for diagnostics.

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

Maintain operates on one axis: **effectiveness** = `followed / (followed + not_followed + irrelevant)`. The counter breakdown diagnoses *why* it's failing and *which SBIA field* to fix:

| Signal | Diagnosis | Remediation |
|--------|-----------|-------------|
| High `followed` rate | Working | Keep |
| High `not_followed` rate | Agent sees the advice but doesn't follow it | Rewrite `action` for clarity, or escalate to CLAUDE.md/rules |
| High `irrelevant` rate | Situation matching is too broad | Narrow `situation` description |
| Never surfaced | Situation hasn't occurred (yet) | No action needed |
| Low total evaluations | Insufficient data | Wait for more data |

### Triage Actions (replacing current apply-proposal)

| Old Action | SBIA Equivalent |
|------------|----------------|
| `broaden_keywords` | **Removed.** "Hidden gem" quadrant doesn't exist. |
| `refine_keywords` | **Removed.** Keywords are gone. |
| `rewrite` (principle/title) | Rewrite `action` (or `situation` if too broad/narrow) |
| `consolidate` | Merge memories with similar situations into a broader one |
| `remove` | Remove (unchanged) |
| `escalate` | Promote to CLAUDE.md/rules (unchanged) |

### `/memory-triage` Skill Changes

The skill simplifies. Instead of presenting noise/hidden gem/leech/working quadrants, it presents:

1. **Not followed** — memories with high not_followed rate. For each: "This memory is being surfaced when relevant but the agent isn't following it. Rewrite the action or escalate?"
2. **Irrelevant** — memories with high irrelevant rate. For each: "This memory is surfacing in wrong contexts. Narrow the situation description?"
3. **Consolidation** — memories with similar situations that could be merged.
4. **Remove** — memories with both high not_followed and high irrelevant (nothing is working).

### `/adapt` Changes

Adapt policies shift from keyword-focused dimensions to SBIA-focused dimensions:
- Instead of "de-prioritize keyword X": "situation descriptions matching pattern X are too broad"
- Instead of "extraction guidance": "common extraction quality issues (vague situations, non-actionable actions)"
- Policy effectiveness measurement stays the same (before/after snapshots on corpus metrics)

## Skill Operations and Pipeline Mapping

### /recall

**LLM does directly:** Nothing — runs command, interprets text output.

**Command:** `engram recall [--query "..."]`

| Operation | Files | Pipeline Stage |
|-----------|-------|----------------|
| Reads session transcripts | `~/.claude/projects/{slug}/*.jsonl` | — |
| Reads memory files (query mode) | `{dataDir}/memories/*.toml` | Surface (retrieval) |
| Reads CLAUDE.md + rules (suppression) | `~/.claude/CLAUDE.md`, `~/.claude/rules/*.md` | Surface (suppression) |
| Writes surfaced_count + last_surfaced_at | `{dataDir}/memories/*.toml` | Surface (tracking) |
| API: Haiku extract relevant content | Anthropic Messages API (query mode only) | — |

**SBIA impact:** Low. `SearchText()` changes input fields but the skill just runs a command.

### /memory-triage

**LLM does directly:** Reads triage output from session-start hook (injected into prompt). Presents to user. Runs commands based on user decisions.

**Command:** `engram apply-proposal --action <action> --memory <path> [--keywords/--fields]`

| Action | Reads | Writes | API | Fields Referenced |
|--------|-------|--------|-----|-------------------|
| `remove` | Target TOML | **Deletes** file | None | — |
| `broaden_keywords` | Target TOML | Appends to `keywords` | None | **`keywords`** |
| `rewrite` | Target TOML | Updates fields | None | **`title`, `content`, `principle`, `anti_pattern`** |
| `refine_keywords` | Target TOML | Removes/adds keywords, clears `irrelevant_queries` | None | **`keywords`, `irrelevant_queries`** |
| `consolidate` | Survivor + member TOMLs | Overwrites survivor; archives members to `{dataDir}/archive/` | Haiku (synthesize principle) | **`principle`** (synthesized) |

**SBIA impact: High.** Every action except `remove` references current field names by name:
- `broaden_keywords` and `refine_keywords` depend on `keywords` existing
- `rewrite` passes `{"title":"...","principle":"..."}` — field names change to SBIA equivalents
- `consolidate` synthesizes a "generalized principle" → becomes "generalized action"

### /adapt

**LLM does directly:** Runs status command, interprets output, presents to user. Runs approve/reject/retire.

**Command:** `engram adapt --data-dir "$ENGRAM_DATA_DIR" [--approve/--reject/--retire <id>]`

| Action | Reads | Writes | API |
|--------|-------|--------|-----|
| status | `policy.toml` | Nothing | None |
| approve | `policy.toml` + all `memories/*.toml` (corpus metrics) | `policy.toml` (status, snapshot) | None |
| reject | `policy.toml` | `policy.toml` | None |
| retire | `policy.toml` | `policy.toml` | None |

**SBIA impact: Medium.** Keyword-focused proposals become less relevant if keywords are dropped. Extraction guidance shifts to situation-quality advice. Policy dimensions may need updating.

## Hook Operations and Pipeline Mapping

### SessionStart (`session-start.sh`)

**Sync output:** Static system reminder ("Say /recall") + correction instructions.

**Async background fork:**
1. Rebuilds binary if stale
2. Runs `engram maintain` → JSON proposals
3. Reads `policy.toml` for adaptation proposal count
4. Writes `~/.claude/engram/pending-maintenance.json` (consumed later by UserPromptSubmit)

| Command | Reads | Writes | API |
|---------|-------|--------|-----|
| `engram maintain` | All `memories/*.toml`, `policy.toml` | stdout JSON only | Haiku (optional, rewrite suggestions) |

**Fields read by maintain:** `surfaced_count`, `followed_count`, `contradicted_count`, `ignored_count`, `irrelevant_count`, `keywords`, `principle`, `title`, `anti_pattern`, `confidence`

**SBIA impact: Medium.** Proposal generation reads field names. Proposal actions reference `keywords`, `principle`, `anti_pattern`.

### UserPromptSubmit (`user-prompt-submit.sh`)

Consumes `pending-maintenance.json` (atomic read + delete), then runs two commands:

#### 1. `engram correct --message "$USER_MESSAGE"`

**Pipeline stage:** Detect → Context Retrieval → SBIA Extraction → Dedup → Write

**Current:**

| Operation | Files | Fields |
|-----------|-------|--------|
| Reads | User message + optional transcript | — |
| Writes | New `memories/<slug>.toml` if correction detected | `title`, `content`, `keywords`, `concepts`, `principle`, `anti_pattern`, `rationale`, `observation_type`, `confidence`, `generalizability` |
| API | Haiku — classifies into tier A/B/C/null | JSON schema defines output fields |

**SBIA redesign:**

| Operation | Files | Fields |
|-----------|-------|--------|
| Detect | User message (fast-path keywords + Haiku classification) | — |
| Context | Session transcript via recall-like code | — |
| Extract | Correction + context → **Sonnet** extracts SBIA | `situation`, `behavior`, `impact`, `action` |
| Dedup | TF-IDF cosine against existing `memories/*.toml` | — |
| Write | New `memories/<slug>.toml` | `situation`, `behavior`, `impact`, `action` + tracking metadata |

**SBIA impact: High.** This becomes the primary (and only) extraction path. Haiku detects; Sonnet extracts.

#### 2. `engram surface --mode prompt --message "$USER_MESSAGE" --format json`

**Pipeline stage:** Surface (retrieval + tracking)

| Operation | Files | Fields |
|-----------|-------|--------|
| Reads | All `memories/*.toml`, `policy.toml`, `surfacing-log.jsonl`, `~/.claude/CLAUDE.md` + rules | Matching: `SearchText()` (title+content+principle+keywords+concepts). Ranking: `surfaced_count`, `followed_count`, `contradicted_count`, `ignored_count`, `irrelevant_count`, `irrelevant_queries`, `generalizability`, `confidence`, `last_surfaced_at` |
| Writes | Increments `surfaced_count` + `last_surfaced_at` on matched memories; appends to `surfacing-log.jsonl` | — |
| API | None | — |
| Displays | `principle` (+ filename slug) to user | — |

**SBIA impact: Medium.** `SearchText()` changes input fields. Display changes from `principle` to `action`.

**LLM instructions injected:** `<system-reminder>` with memory names + principles; correction notification if detected.

### Stop surface (`stop-surface.sh`)

**Pipeline stage:** Surface (stop mode) — same as prompt-mode surface but matches against agent output instead of user message. Blocks response if conflicting memories found.

| Command | Reads | Writes | API |
|---------|-------|--------|-----|
| `engram surface --mode stop` | Transcript JSONL + all `memories/*.toml` | Same as prompt-mode surface | None |

**SBIA impact: Low.** Same as prompt-mode surface.

### Stop async (`stop.sh`)

**Current pipeline stage:** Extract → Dedup → Write (end-of-turn learning)

| Command | Reads | Writes | API |
|---------|-------|--------|-----|
| `engram flush` (→ `engram learn`) | Transcript JSONL (incremental via `learn-offset.json`), all `memories/*.toml` | New `memories/<slug>.toml` per surviving candidate; `learn-offset.json`; adaptation proposals in `policy.toml`; deletes stale `surfacing-log.jsonl` | Haiku — extraction prompt |

**SBIA redesign: Eliminated or reduced to cleanup.** With corrections-only extraction moved to UserPromptSubmit, the batch extraction path is no longer needed. This hook either goes away entirely or is reduced to surfacing-log cleanup.

### Hook Impact Summary

| Hook | Current | SBIA Redesign |
|------|---------|---------------|
| **SessionStart** | Runs `maintain`, writes pending file | Field names change in proposals. Otherwise similar. |
| **UserPromptSubmit (correct)** | Haiku classifies single message | **Primary extraction path.** Haiku detects → recall context → Sonnet SBIA extraction. |
| **UserPromptSubmit (surface)** | BM25 on SearchText, displays principle | SearchText uses SBIA fields. Display shows `action`. |
| **Stop (surface)** | Surface on agent output | Same as surface above. |
| **Stop (learn)** | Full batch extraction pipeline | **Eliminated.** Reduced to cleanup or removed entirely. |

## Resolved Questions

1. **Keywords:** Dropped. BM25 on SBIA text for surfacing; TF-IDF cosine for candidate retrieval in dedup.
2. **Tier C:** Dropped. Corrections are inherently behavioral. No contextual facts in SBIA model.
3. **Dedup strategy:** Sonnet-driven via SBIA decision tree. TF-IDF finds candidate similar memories (cheap retrieval), then Sonnet evaluates each SBIA dimension independently and determines disposition (duplicate/contradiction/generalization/separate/store). One Sonnet call handles both extraction and dedup. Duplicate detections trigger self-diagnosis (surfacing vs. listening failure).
4. **Evaluate counters:** Simplified to three: `followed`, `not_followed`, `irrelevant`. "Contradicted" and "ignored" collapse — the distinction isn't actionable. Automated via Haiku at stop hook; LLM self-report dropped.
5. **Quadrants:** Eliminated. Surfacing frequency measures situation rarity, not memory quality. Maintain operates on effectiveness only, with counter breakdown diagnosing which SBIA field to fix.
6. **Triage actions:** `broaden_keywords`/`refine_keywords` removed. Replaced by: rewrite `action`, narrow/broaden `situation`, escalate, consolidate, remove.
7. **Adapt policies:** Shift from keyword dimensions to SBIA dimensions (situation breadth, action clarity, extraction quality).

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

| Current Tier | Action |
|-------------|--------|
| **A** (explicit instructions) | Sonnet converts to SBIA fields. These have the clearest principle/anti_pattern to map. |
| **B** (inferred corrections) | Archive. Weak signal, not worth converting. New corrections will be properly extracted with full SBIA context. |
| **C** (contextual facts) | Archive. Already filtered out of extraction pipeline. |

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
