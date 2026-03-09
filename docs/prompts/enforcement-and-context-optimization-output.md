# Instruction Adherence & Context Optimization: Analysis

## 1. Decision Framework

Given an instruction that's being violated — from any source (CLAUDE.md, skill, engram memory, rule, convention) — evaluate in this order:

### Step 1: Can it become deterministic code? (Dimension 2)

**Test:** Is the instruction a mechanical rule with no judgment component?

| Signal                          | Example                               | Action                          |
| ------------------------------- | ------------------------------------- | ------------------------------- |
| "always X before Y"             | "always lint before committing"       | Pre-commit hook, CI gate        |
| "never X" where X is detectable | "never use `go test` directly"        | Shell alias/wrapper that errors |
| Format/naming convention        | "use Conventional Commits"            | Linter, formatter, git hook     |
| Procedural checklist step       | "run pressure tests on skill updates" | CLI command that automates it   |

**If yes:** Implement as deterministic automation. Zero context cost, 100% reliable. Done.

**If partial:** Extract the mechanical portion into tooling, keep only the judgment portion as an instruction (Dimension 4). A 200-line skill with 150 mechanical lines → 50-line skill + CLI tool.

### Step 2: Can it become a rule? (New surface — between CLAUDE.md and skill)

**Test:** Is the instruction scoped to specific file types or patterns, and always relevant when those files are active?

| Signal                        | Example                           | Action                  |
| ----------------------------- | --------------------------------- | ----------------------- |
| File-type-specific convention | "Go tests must use t.Parallel()"  | Rule scoped to `*.go`   |
| Pattern-specific guard        | "never delete .toml memory files" | Rule scoped to `*.toml` |
| Language idiom                | "use http.NewRequestWithContext"  | Rule scoped to `*.go`   |

**If yes:** Create a rule (global or pattern-scoped). Rules are always loaded when matching files are active — lower context cost than skills (loaded by similarity) and more targeted than CLAUDE.md (always loaded regardless of relevance).

### Step 3: Can the instruction itself be improved? (Dimension 7 — Content Quality)

**Test:** Is the instruction being ignored because of *how* it's written, not *where* it appears?

Before escalating enforcement, diagnose whether the instruction's content is the problem. A perfectly-placed, well-enforced instruction still fails if the model can't act on it.

| Symptom | Diagnosis | Fix |
|---------|-----------|-----|
| Followed sometimes, ignored when task is complex | Too abstract — model loses it under cognitive load | Add concrete example or anti-pattern |
| Contradicted (model does the opposite) | Framing mismatch — stated as principle but model needs prohibition | Reframe as anti-pattern with "instead, do X" |
| Ignored consistently despite surfacing | Too vague — model can't determine when/how to apply it | Add trigger conditions ("when you see X, do Y") |
| Followed for one case, missed for analogous cases | Too narrow — instruction doesn't generalize | Broaden with multiple examples or extract the general principle |
| Memory duplicates exist (e.g., `always-use-targ-reminder.toml` + `always-use-targ-reminder-2.toml`) | Original wasn't effective, user created a second attempt | Merge into one improved version |

**Source-specific refinement:**

| Source | Current Refinement | Gap |
|--------|-------------------|-----|
| Engram memories | UC-16 maintain: leech rewrite, hidden gem broadening | Rewrites are keyword/principle-focused — doesn't diagnose *framing* or *specificity* problems |
| CLAUDE.md | Manual only | No automated diagnosis — a CLAUDE.md entry that's ignored has no feedback loop |
| Skills | Manual only | No mechanism to identify which *lines* of a skill are being violated vs. followed |
| Rules | N/A (not yet created) | No effectiveness tracking for rules |

**Action:** If the instruction can be improved, improve it *before* escalating to enforcement hooks. Escalation is for instructions that are well-written but lose salience — not for instructions that are poorly expressed.

This step integrates with:
- UC-16 maintain (existing: memory-specific refinement proposals)
- UC-20 (proposed: extends refinement to all instruction sources — see Section 3)

### Step 4: Can an LLM hook enforce it? (Dimension 1)


**Test:** Is the instruction about _what not to do_ (anti-pattern), and is the violation detectable from tool name/input or recent context?

| Hook Point       | Detectable                    | Example                                               |
| ---------------- | ----------------------------- | ----------------------------------------------------- |
| PreToolUse       | Tool name + input JSON        | "Don't use `rm` on memory files"                      |
| PostToolUse      | Tool output + preceding input | "If linter fails, don't retry — collect all failures" |
| UserPromptSubmit | User message + transcript     | "Don't skip interview steps in skills"                |
| Stop             | Full session transcript       | "Did the agent follow the skill's checklist?"         |

**Advisory vs. Blocking:** Start advisory (system-reminder). If effectiveness < 40% after 5+ surfacings, escalate:

1. Advisory with emphasis (bold, "CRITICAL" prefix)
2. Blocking with explanation (PreToolUse returns `continue: false`)
3. If blocking causes more harm → revert to advisory, flag for human review

### Step 5: Can context be pruned to make room? (Dimension 3)

**Test:** Is the instruction being crowded out by lower-value context?

Audit current context sources:

| Source                  | Current Load                       | Pruning Opportunity                               |
| ----------------------- | ---------------------------------- | ------------------------------------------------- |
| CLAUDE.md (global)      | ~1500 tokens                       | Demote narrow rules to project CLAUDE.md or rules |
| CLAUDE.md (project)     | ~1200 tokens                       | Move file-specific conventions to rules           |
| Skills (active)         | ~2000 tokens/skill                 | Compress, extract mechanical steps to tooling     |
| Engram SessionStart     | ~400 tokens (1.6KB)                | Reduce top-N memories, compress creation log      |
| Engram UserPromptSubmit | ~300 tokens (1.3KB)                | Reduce top-N, tighten BM25 threshold              |
| Engram PreToolUse       | ~350 tokens (1.5KB) per invocation | Raise relevance threshold, reduce top-N from 5→3  |
| Session context (UC-14) | ~650 tokens (2.6KB)                | Enforce rolling size cap (~1KB)                   |
| MEMORY.md (auto-memory) | ~1500 tokens                       | Compress, deduplicate with engram memories        |

**Pruning heuristics:**

- If an instruction is in both CLAUDE.md and an engram memory → remove the engram memory (CLAUDE.md has higher salience)
- If a skill instruction is never violated → it's working, leave it alone
- If a skill instruction is always violated → the problem isn't context (it's either wrong, or needs enforcement)
- If a memory has effectiveness < 20% after 10 surfacings → it's noise, remove it

### Step 6: Can we detect non-compliance proactively? (Dimension 6)

**Test:** Would a timely reminder prevent the violation?

The `traced` pressure-test example is canonical: the model _knows_ the instruction but loses it in the skill's complexity. A PostToolUse reminder after skill file writes would cost ~50 tokens and catch the violation at the point of action.

**Detection points:**
| When | What | Cost |
|------|------|------|
| PostToolUse (Write/Edit on skill files) | "Did you pressure-test?" | ~50 tokens |
| PostToolUse (Bash with `git commit`) | "Did you use `/commit`?" | ~50 tokens |
| Stop (session end) | Full checklist audit | ~500 tokens (one-time) |
| PreCompact | "Are there unaddressed requirements?" | ~200 tokens |

### Step 7: Rank the options (Dimension 5)

When multiple approaches exist, rank by:

1. **Deterministic automation** — Perfect reliability, zero context cost, requires code maintenance
2. **Instruction refinement** — Fix the content before adding enforcement. Low cost, often sufficient alone. If the instruction is unclear/abstract/poorly-framed, no amount of enforcement will help.
3. **Rules** — High reliability (always in context for matching files), low context cost, easy to maintain
4. **LLM hook (blocking)** — High reliability, low context cost (~50 tokens per invocation), requires testing
5. **LLM hook (advisory)** — Moderate reliability, low context cost, easy to iterate
6. **Context pruning** — Indirect benefit (makes room for what matters), no enforcement guarantee

**Tiebreaker:** If two approaches have similar effectiveness, prefer the one with lower maintenance burden.

---

## 2. Architecture Sketch

### Current Architecture

```
Hook Scripts (bash) → engram binary (Go) → memories (TOML)
                                          → surfacing log (JSONL)
                                          → evaluations (JSONL)
                                          → session context (MD)
```

### Proposed Evolution

```
                    ┌─────────────────────────────────────────┐
                    │          Instruction Registry           │
                    │  (memories + CLAUDE.md + skills + rules) │
                    └───────────┬─────────────────────────────┘
                                │
                    ┌───────────▼───────────────────┐
                    │     Enforcement Engine         │
                    │  ┌─────────┐ ┌──────────────┐ │
                    │  │ Advisory │ │  Blocking    │ │
                    │  │ Surface  │ │  PreToolUse  │ │
                    │  └─────────┘ └──────────────┘ │
                    │  ┌─────────┐ ┌──────────────┐ │
                    │  │PostTool │ │  Stop Audit   │ │
                    │  │Reminder │ │  (checklist)  │ │
                    │  └─────────┘ └──────────────┘ │
                    └───────────┬───────────────────┘
                                │
                    ┌───────────▼───────────────────┐
                    │     Effectiveness Pipeline     │
                    │  track → evaluate → review     │
                    │          → maintain → escalate  │
                    └───────────┬───────────────────┘
                                │
                    ┌───────────▼───────────────────┐
                    │     Context Budget Manager     │
                    │  token counting, cap enforcement│
                    │  source prioritization          │
                    └─────────────────────────────────┘
```

### New Components

**A. Instruction Registry** (`internal/registry/`)

- Unified index of all instruction sources: engram memories, CLAUDE.md entries, skill instructions, rules
- Each entry has: source, content hash, last-violated timestamp, effectiveness score
- Purpose: Cross-source deduplication, unified effectiveness tracking, gap analysis
- _Not_ a replacement for individual sources — a read-only overlay that indexes them

**B. PostToolUse Hook** (`hooks/post-tool-use.sh`)

- New hook point for proactive non-compliance detection
- Fires after Write/Edit on tracked file patterns (skills, CLAUDE.md, specs)
- Injects targeted reminders: "You just edited a skill file — did you pressure-test?"
- Low context cost (~50 tokens per reminder)

**C. Stop Audit Hook** (enhance existing `hooks/stop.sh`)

- At session end, run a lightweight checklist audit against active instructions
- Compare session actions to high-priority instruction requirements
- Report violations as learning signal for future sessions
- Feed results into effectiveness pipeline

**D. Context Budget Manager** (`internal/budget/`)

- Token estimation for each context source (1 token ≈ 4 chars for English text)
- Hard cap on total engram context injection per hook event
- Priority-based allocation: high-effectiveness memories first, then by relevance score
- Report budget utilization in `engram review` output

**E. Escalation Engine** (extend `internal/maintain/`)

- When maintain detects a leech memory: propose enforcement escalation
- Escalation ladder: advisory → emphasized advisory → PostToolUse reminder → PreToolUse block
- Each escalation step is a maintain proposal requiring user confirmation
- De-escalation: if blocking causes harm (measured by new contradictions), propose reverting

**F. Automation Generator** (extend `internal/maintain/`)

- When maintain detects a mechanical instruction being violated: propose deterministic automation
- Output: shell script, pre-commit hook, or rule definition
- User confirms, generator writes the file and updates instruction registry

### Hook Architecture Changes

Current:

```
SessionStart    → surface (session-start mode)
UserPromptSubmit → correct + surface (prompt mode)
PreToolUse      → surface (tool mode, advisory only)
PreCompact      → learn + evaluate + context-update
Stop            → learn + context-update
```

Proposed additions:

```
PostToolUse     → reminder (pattern-matched, targeted)    [NEW]
Stop            → learn + context-update + audit          [ENHANCED]
PreToolUse      → surface (advisory OR blocking, per-memory escalation level)  [ENHANCED]
```

---

## 3. Candidate UCs

### UC-17: Context Budget Management

**Summary:** Track and cap total context injection across all engram hook points. Prioritize high-effectiveness memories within budget.

**Scope:**

- Token estimation for all context output (surface, correct, session-context)
- Configurable per-hook budget caps (e.g., SessionStart: 500 tokens, PreToolUse: 200 tokens)
- Priority allocation: sort by effectiveness × relevance, fill until budget reached
- Budget utilization reporting in `engram review`
- Budget warnings when a hook consistently hits its cap

**Dependencies:** UC-2 (surface), UC-6 (effectiveness)
**Complexity:** Medium. Mostly extends existing surface logic with token counting and cutoff.

**Why this matters:** 271 memories × ~100 tokens each = ~27K tokens of potential context. Without budgeting, as memory count grows, surfacing quality degrades. This is the foundational constraint that makes all other optimizations meaningful.

### UC-18: PostToolUse Proactive Reminders

**Summary:** After the model writes or edits tracked files, inject a targeted reminder about commonly-violated instructions relevant to that file type.

**Scope:**

- New PostToolUse hook in `hooks/hooks.json`
- Pattern-based trigger configuration: file patterns → reminder sets
- Reminder content sourced from instruction registry (memories, CLAUDE.md, skills with anti-pattern match)
- Budget-capped (≤100 tokens per reminder)
- Effectiveness tracking: did the model comply after the reminder?
- Suppression: if the model already did the thing before the reminder, don't inject it

**Dependencies:** UC-17 (budget), UC-2 (surface infrastructure)
**Complexity:** Medium. New hook wiring + pattern matching logic.

**Why this matters:** The pressure-test example shows that a single well-timed reminder is often sufficient. PostToolUse is the most targeted hook point — it fires at the moment of action, not at the start of a conversation when the action is abstract.

### UC-19: Stop Session Audit

**Summary:** At session end, run a lightweight audit that checks whether high-priority instructions were followed during the session.

**Scope:**

- Enhanced Stop hook with audit phase (after learn, before context-update)
- Audit checks: high-priority memories surfaced during session + their outcomes
- Checklist mode: for skills invoked during session, verify critical steps were performed
- Output: audit report written to `<data-dir>/audits/<timestamp>.json`
- Integration with effectiveness pipeline: audit results feed into evaluate

**Dependencies:** UC-15 (evaluate infrastructure), UC-2 (surfacing log)
**Complexity:** Medium-High. Requires LLM call to assess compliance against instruction set.

**Why this matters:** Current effectiveness evaluation happens at PreCompact (triggered by context pressure). Stop audit catches sessions that end without compaction — which is most sessions.

### UC-20: Instruction Quality, Deduplication & Gap Analysis

**Summary:** Cross-reference all instruction sources (CLAUDE.md, engram memories, rules, skills) to identify duplicates, quality problems, and gaps. Extends UC-16's memory-specific refinement to ALL instruction sources.

**Scope:**

- `engram instruct audit` command: unified analysis across CLAUDE.md, memories, rules, skills
- **Deduplication:** Scan for overlapping instructions across sources. Recommend which source to keep (based on effectiveness data, salience hierarchy: CLAUDE.md > rules > memories).
- **Quality diagnosis:** For instructions with low effectiveness, diagnose *why* using LLM analysis:
  - Too abstract (needs concrete example or anti-pattern)
  - Framing mismatch (principle vs. prohibition, positive vs. negative)
  - Missing trigger conditions (when/where to apply)
  - Too narrow (doesn't generalize to analogous cases)
  - Too verbose (buried in surrounding text, needs extraction)
- **Refinement proposals:** Generate rewritten versions with rationale. For memories, output as maintain-compatible proposals. For CLAUDE.md/skills/rules, output as diff suggestions requiring user confirmation.
- **Gap analysis:** Compare instruction anti-patterns against observed tool actions — are there common violation patterns with no corresponding instruction?
- **Skill decomposition:** For skills with low per-line effectiveness, identify which lines are followed vs. ignored. Propose: extract high-value lines as standalone rules/memories, compress or remove low-value lines.
- Integration with maintain: refinement proposals alongside existing maintain proposals

**Relationship to UC-16:** UC-16 maintain already generates memory-specific proposals (leech rewrites, hidden gem broadening). UC-20 extends this in two ways: (a) it covers non-memory sources, and (b) it adds deeper content diagnosis beyond keywords/principles — framing, specificity, example quality. UC-16's existing handlers remain for the common case; UC-20 handles cross-source analysis and deeper refinement.

**Dependencies:** UC-6 (review), UC-16 (maintain), UC-17 (budget — needed to measure context cost of each instruction)
**Complexity:** Medium-High. Cross-source indexing + LLM diagnosis for quality problems.

**Why this matters:** Several CLAUDE.md entries overlap with engram memories (e.g., "always use targ" appears in CLAUDE.md _and_ as `always-use-targ-reminder.toml` _and_ `always-use-targ-reminder-2.toml`). But beyond duplicates, some instructions fail not because of placement or enforcement but because the instruction itself is unclear, too abstract, or poorly framed. Fixing the content is cheaper and more durable than adding enforcement around bad content.

### UC-21: Enforcement Escalation Ladder

**Summary:** When maintain detects a leech memory (frequently surfaced, rarely followed), propose escalation from advisory to blocking enforcement.

**Scope:**

- Escalation levels: advisory → emphasized advisory → PostToolUse reminder → PreToolUse block
- Each escalation is a maintain proposal with rationale and predicted impact
- De-escalation: if blocking causes new compliance problems, propose reverting
- User confirms each escalation/de-escalation
- Tracking: escalation level stored per memory, effectiveness tracked per level

**Dependencies:** UC-16 (maintain), UC-18 (PostToolUse), UC-17 (budget)
**Complexity:** High. This is issue #44 revised — the escalation ladder replaces the binary advisory→blocking jump.

**Why this matters:** This is the systematic answer to "what do we do when advisory doesn't work?" Instead of jumping straight to blocking (which may cause harm), we walk up a ladder with measurement at each step.

### UC-22: Mechanical Instruction Extraction

**Summary:** Identify instructions that are purely mechanical (no judgment required) and generate deterministic automation to replace them.

**Scope:**

- `engram automate` command: analyze leech/noise memories for mechanical patterns
- Pattern recognition: "always X before Y", "never X when Z", "format as..."
- Generator: produce shell scripts, pre-commit hooks, or rule definitions
- Verification: generated automation must pass a test before replacing the instruction
- Instruction retirement: once automation is verified, memory gets retired with pointer to the automation

**Dependencies:** UC-21 (escalation decision point — "should this become automation?")
**Complexity:** High. LLM-assisted code generation with verification loop.

**Why this matters:** Every instruction that becomes code frees context AND is perfectly enforced. The maintain pipeline already identifies leeches — this UC provides a resolution path beyond "rewrite the memory."

### Priority Ordering

| Priority | UC                  | Rationale                                                              |
| -------- | ------------------- | ---------------------------------------------------------------------- |
| 1        | UC-17 (Budget)      | Foundation — everything else needs budget awareness                    |
| 2        | UC-20 (Dedup)       | Quick context savings, informs what to prune                           |
| 3        | UC-18 (PostToolUse) | High-impact, medium complexity, solves pressure-test class of problems |
| 4        | UC-19 (Stop Audit)  | Closes evaluation gap for sessions without compaction                  |
| 5        | UC-21 (Escalation)  | Subsumes issue #44, requires PostToolUse infrastructure                |
| 6        | UC-22 (Automation)  | Highest long-term value but highest complexity                         |

---

## 4. Context Budget Model

### Current Budget (Measured)

| Source                         | Per-Event Size              | Events/Session | Est. Session Total         |
| ------------------------------ | --------------------------- | -------------- | -------------------------- |
| SessionStart (surface)         | ~400 tokens (1.6KB)         | 1              | 400 tokens                 |
| SessionStart (session-context) | ~650 tokens (2.6KB)         | 1              | 650 tokens                 |
| SessionStart (midturn note)    | ~60 tokens                  | 1              | 60 tokens                  |
| UserPromptSubmit (surface)     | ~300 tokens (1.3KB)         | ~20/session    | 6,000 tokens               |
| UserPromptSubmit (correct)     | ~50 tokens (when triggered) | ~2/session     | 100 tokens                 |
| PreToolUse (surface)           | ~350 tokens (1.5KB)         | ~50/session    | 17,500 tokens              |
| **Engram total**               |                             |                | **~24,700 tokens/session** |

Non-engram context (for reference):
| Source | Est. Size | Notes |
|--------|-----------|-------|
| CLAUDE.md (global) | ~1,500 tokens | Always loaded |
| CLAUDE.md (project) | ~1,200 tokens | Always loaded |
| MEMORY.md | ~1,500 tokens | Always loaded |
| Active skills | ~2,000 tokens/skill | Loaded on invocation |
| Rules | ~200-500 tokens/rule | Loaded when file pattern matches |

### Budget Targets

**Principle:** Engram's total context cost should be ≤5% of available context per turn. At 200K context window, that's ~10K tokens/turn. At 100K effective (after compaction), ~5K tokens/turn.

**Per-hook caps:**

| Hook              | Current       | Proposed Cap | Reduction Strategy                              |
| ----------------- | ------------- | ------------ | ----------------------------------------------- |
| SessionStart      | ~1,100 tokens | 800 tokens   | Top-10 memories (not 20), compress creation log |
| UserPromptSubmit  | ~300 tokens   | 300 tokens   | Already reasonable; tighten BM25 threshold      |
| PreToolUse        | ~350 tokens   | 200 tokens   | Top-3 memories (not 5), raise relevance floor   |
| PostToolUse (new) | N/A           | 100 tokens   | Single targeted reminder, not memory dump       |
| Stop audit (new)  | N/A           | 500 tokens   | One-time cost, high value                       |

**Projected after budget enforcement:**

| Source           | Per-Event | Events/Session | Session Total |
| ---------------- | --------- | -------------- | ------------- |
| SessionStart     | 800       | 1              | 800           |
| UserPromptSubmit | 300       | 20             | 6,000         |
| PreToolUse       | 200       | 50             | 10,000        |
| PostToolUse      | 100       | ~10            | 1,000         |
| Stop audit       | 500       | 1              | 500           |
| **Total**        |           |                | **~18,300**   |

That's a 26% reduction from current, while adding two new enforcement surfaces.

### Scaling Concern

With 271 memories and growing, BM25 matching will surface increasingly marginal matches. The budget cap prevents this from inflating context, but it means newer memories compete harder for slots. The deduplication UC (UC-20) is critical: removing the ~20-30 duplicates estimated from the memory corpus frees slots for genuinely new content.

### Token Estimation Method

For UC-17 implementation: `len(text) / 4` is a conservative estimator for English text with code snippets. No need for a real tokenizer — we're setting soft caps, not hard limits. A 10% error in estimation is acceptable.

---

## 5. Quick Wins

These require no new UCs — just tweaks to existing code and configuration.

### QW-1: Reduce PreToolUse top-N from 5 to 3

**File:** `internal/surface/surface.go`
**Impact:** ~30% reduction in PreToolUse context (the highest-volume hook)
**Risk:** Low — most surfacings beyond top 3 are marginal matches
**Effort:** One constant change

### QW-2: Add relevance floor to BM25 matching

**File:** `internal/surface/surface.go`
**Impact:** Prevents surfacing memories with near-zero relevance scores
**Risk:** Low — irrelevant memories waste context without benefit
**Effort:** Add minimum score threshold, skip memories below it

### QW-3: Cap session-context.md at ~1KB

**File:** `internal/context/file.go` or `summarize.go`
**Impact:** Prevents session context from growing unbounded across long sessions
**Risk:** Low — Haiku summarization already compresses; this adds a safety net
**Effort:** Truncate/re-summarize if file exceeds cap

### QW-4: Reduce SessionStart top-N from 20 to 10

**File:** `internal/surface/surface.go` (session-start mode)
**Impact:** ~50% reduction in SessionStart context
**Risk:** Medium — some relevant memories may be missed at session start. Mitigated by UserPromptSubmit surfacing on first prompt.
**Effort:** One constant change

### QW-5: Deduplicate CLAUDE.md entries against engram memories

**Manual, no code:** Review 271 memories against CLAUDE.md entries. Remove memories that exactly duplicate CLAUDE.md content (CLAUDE.md has higher salience, so the memory adds no value).
**Impact:** Frees memory slots and reduces false-match noise
**Effort:** ~30 minutes of manual review, or a one-off `engram deduplicate` script

### QW-6: Move file-specific CLAUDE.md entries to rules

**Manual, no code:** Entries like "Go tests must use t.Parallel()" and "use http.NewRequestWithContext" belong in a `*.go` rule, not in always-loaded CLAUDE.md.
**Impact:** ~200 tokens freed from CLAUDE.md, moved to targeted context (only loaded when editing Go files)
**Effort:** Create rule file, remove entries from CLAUDE.md

### QW-7: Add PostToolUse pressure-test reminder (standalone, before UC-18)

**Files:** `hooks/hooks.json`, `hooks/post-tool-use.sh`
**Impact:** Solves the specific pressure-test problem immediately
**Risk:** Low — a single hardcoded reminder for a known problem
**Effort:** ~30 lines of bash + hooks.json entry. No Go code needed.

---

## 6. Revision of Issue #44

### Current Scope (Issue #44)

> "When the model repeatedly ignores advisory memories, escalate to deterministic hooks that block the action."

### Problems with Current Scope

1. **Binary jump from advisory to blocking.** The current proposal goes directly from "system-reminder" to "block the tool." This is too aggressive — blocking can cause more harm than the original non-compliance if the blocking rule is too broad or the context doesn't match.

2. **Only addresses one dimension.** Issue #44 focuses entirely on Dimension 1 (LLM hook enforcement). It doesn't consider whether the instruction should instead become deterministic automation (Dimension 2), a rule (new surface), or whether the real problem is context crowding (Dimension 3).

3. **No budget awareness.** Adding blocking hooks adds context (the block message, the explanation). Without a budget, this compounds the enforcement paradox.

4. **No de-escalation path.** If a blocking hook causes problems, there's no mechanism to walk it back.

### Recommendation: Absorb into UC-21

Issue #44 should become **UC-21: Enforcement Escalation Ladder**, with these changes:

1. **Replace binary escalation with a ladder:** advisory → emphasized → PostToolUse reminder → PreToolUse block → deterministic automation
2. **Add budget awareness:** Each escalation step must fit within the context budget (UC-17 dependency)
3. **Add de-escalation:** If an escalated enforcement causes new compliance problems, automatically propose reverting
4. **Add dimension routing:** Before escalating enforcement, check whether the instruction should instead become automation (UC-22) or a rule — escalation is only for instructions that genuinely require LLM judgment
5. **Require PostToolUse infrastructure:** UC-18 must ship first, as it provides the intermediate enforcement surface between advisory and blocking

### Updated Dependencies

```
UC-17 (Budget) ─────────────────────────────┐
                                             │
UC-18 (PostToolUse) ─────┐                  │
                          │                  │
UC-20 (Dedup) ───────────┤                  │
                          ▼                  ▼
               UC-21 (Escalation Ladder, replaces #44)
                          │
                          ▼
               UC-22 (Mechanical Automation)
```

Issue #44 stays open but is relabeled as UC-21 with the revised scope. The original UC-12 number from issue #18 is retired (it was the old SQLite-era numbering).

---

## Summary

The core insight is that **non-compliance is a resource allocation problem, not a knowledge problem.** The model usually _knows_ the instructions — it loses them in the noise. The solution space has three parts:

1. **Improve content:** Instruction refinement (UC-20) — fix poorly-worded, too-abstract, or badly-framed instructions before adding enforcement around them. Extends UC-16's memory-specific maintain to all instruction sources.
2. **Reduce noise:** Budget management (UC-17), deduplication (UC-20), context pruning (QW-1 through QW-6)
3. **Increase signal at the right moment:** PostToolUse reminders (UC-18), Stop audit (UC-19), escalation ladder (UC-21)

The decision framework's key addition: **Step 3 (instruction quality) comes before Step 4 (enforcement hooks).** Escalation is for well-written instructions that lose salience — not for instructions that are unclear. This mirrors what UC-16 maintain already does for memory leeches (diagnose content quality before proposing escalation), but extends it to CLAUDE.md, skills, and rules.

Start with the quick wins (days of effort, immediate impact), then UC-17 (budget foundation), then UC-18+UC-20 (medium complexity, high value). UC-21 and UC-22 are the end-state — a system that automatically routes violated instructions to the most effective resolution: refinement, automation, or enforcement.
