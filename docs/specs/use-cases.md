# Use Cases

## UC-3: Remember & Correct

**Description:** When the user provides a learning signal — explicit instruction, teachable correction, or contextual fact — classify and enrich it in a single LLM call, then write it as a structured TOML memory file.

**Starting state:** The user sends a message during a session. The UserPromptSubmit hook fires with the prompt text and transcript path.

**End state:** If the message contains a learning signal: an enriched TOML memory file exists with tier-appropriate metadata and anti-pattern generation. If not a signal: no file created, no side effects.

**Actor:** System (Go binary triggered by UserPromptSubmit hook).

**Key interactions:**

- **Fast-path detection (no LLM):** Three keywords — `remember`, `always`, `never` — trigger immediate tier-A classification. Skip the classifier LLM call; go straight to enrichment. This handles the highest-confidence signals with zero latency.

- **Unified LLM classifier (everything else):** A single API call (claude-haiku-4-5-20251001) receives the user's message plus ~2000 tokens of recent transcript context (read from `transcript_path` in the hook JSON input). The call both classifies the signal tier and extracts structured memory fields. Returns JSON:

  | Field | Description |
  |-------|-------------|
  | `tier` | `"A"`, `"B"`, `"C"`, or `null` (not a signal) |
  | `title` | 5-10 word summary |
  | `content` | Full message verbatim |
  | `observation_type` | Category label |
  | `concepts` | Key concept tags |
  | `keywords` | Searchable keywords |
  | `principle` | Positive rule to follow |
  | `anti_pattern` | Negative pattern to avoid (tier-gated, see below) |
  | `rationale` | Why this matters |
  | `filename_summary` | 3-5 words for slug generation |

  If `tier` is `null` → skip entirely, no file created.

- **Tier semantics:**

  | Tier | What | Anti-pattern | Example |
  |------|------|-------------|---------|
  | **A** | Explicit instruction | Always generated | "Remember: always use fish" |
  | **B** | Teachable correction | When generalizable (LLM decides) | "No, use targ — don't run raw go test" |
  | **C** | Contextual fact | Never generated | "The port is 3001" |
  | **—** | Not a signal | — (skip) | "Hold on" / "Try again" / "It's broken" |

- **TOML file output:** Memory written to `<data-dir>/memories/<slug>.toml` where slug is the slugified filename summary (3-5 hyphenated words). Same format as before:

```toml
title = "Use targ test not go test"
content = "This project uses the targ build system. Always use targ test, targ check, and targ build instead of raw go commands."
observation_type = "correction"
concepts = ["targ", "build-system", "testing"]
keywords = ["targ", "test", "go-test", "build", "check"]
principle = "Use project-specific build tools"
anti_pattern = "Running go test directly"
rationale = "targ wraps go test with proper flags and coverage requirements"
confidence = "A"
created_at = "2026-03-03T18:00:00Z"
updated_at = "2026-03-03T18:00:00Z"
```

- **Feedback:** System reminder injected showing: the tier classification, the memory file created, key fields captured, and the file path.

- **No graceful degradation:** If no API token is configured, emit a loud stderr error (`[engram] Error: memory capture skipped — no API token configured`) and create no file. Never write degraded memories. (Closes #32.)

- **Session-end extraction (UC-1)** deduplicates against mid-session corrections by checking existing files in the memories directory.

---

## UC-2: Hook-Time Surfacing & Enforcement

**Description:** Surface relevant memories at hook time as advisory system reminders. The agent uses these with full session context to exercise judgment.

**Starting state:** The memory store contains TOML files written by UC-3. A hook fires (SessionStart, UserPromptSubmit, or PreToolUse).

**End state:** Relevant memories are surfaced as system reminders at all three hook points (SessionStart, UserPromptSubmit, PreToolUse). The agent uses these advisories with full session context to exercise judgment. Each surfacing event is recorded in the memory's TOML file (count, timestamp, context type) for effectiveness measurement.

**Actor:** System (hook scripts invoke Go binary for retrieval and surfacing).

**Key interactions:**

- **SessionStart — passive surfacing:** Surface the top 20 memories by recency as a system reminder. No matching needed — recency is the only signal. Provides context priming at session start. The reminder lists each surfaced memory's title and file path so the user can inspect or edit them.

- **UserPromptSubmit — passive surfacing:** Keyword/concept match the user's message against memory `keywords` and `concepts` fields. Surface matching memories as a system reminder alongside the existing UC-3 correction detection. No blocking — informational only. The reminder lists each matched memory's title, file path, and which keywords matched.

- **PreToolUse — advisory surfacing:**
  1. **Keyword pre-filter (fast, no LLM):** Scan memory TOML files. For each memory with an `anti_pattern` field (tier A always, tier B when generalizable — tier C memories never have anti-patterns), check if any of its `keywords` appear in the tool name or tool input. Most tool calls won't match → zero overhead.
  2. **Advisory output:** For memories that pass the keyword pre-filter, emit a `<system-reminder>` listing each matched memory's title, principle, and file path. No blocking, no LLM call. Claude exercises judgment with full session context — better accuracy than haiku judging in isolation.

- **No graceful degradation needed:** No LLM calls in the PreToolUse path means no API token requirement and no timeout failure mode.

- **Pure Go, no CGO:** Same constraint as UC-3. Keyword matching is string operations on TOML files.

- **Memory format extended for instrumentation:** Reads the same TOML files UC-3 writes. The `keywords`, `anti_pattern`, and `principle` fields already exist and drive the matching/enforcement. Three new optional fields are added to the TOML format for surfacing instrumentation:
  - `surfaced_count` (int) — incremented each time the memory is surfaced
  - `last_surfaced` (RFC 3339 timestamp) — updated to the current time on each surfacing event
  - `surfacing_contexts` (string array) — bounded list of recent context types (`session-start`, `prompt`, `tool`), capped at 10 entries

- **Surfacing instrumentation:** After each surfacing mode determines which memories matched, the system updates each matched memory's TOML file in-place with the new tracking fields. This is fire-and-forget: instrumentation errors are logged to stderr but never fail the surfacing operation (ARCH-6 exit-0 contract). The data collected here is the foundation for Phase 2 (outcome signals) and Phase 3 (effectiveness diagnosis).

- **Ranking deferred:** Frecency (recency × impact) ranking is deferred until surfacing instrumentation (this phase) and outcome signals (Phase 2) provide the data. See issue #18 comment and docs/plans/memory-effectiveness-plan.md for details.

---

## UC-1: Session Learning

**Description:** At context compaction and session end, review the session transcript with an LLM to extract learnings that weren't captured by real-time correction detection (UC-3). Write them as enriched TOML memory files.

**Starting state:** A session is active and a PreCompact or SessionEnd hook fires. The transcript contains learning signals the real-time classifier missed, architectural decisions, discovered constraints, working solutions, and implicit patterns.

**End state:** New enriched TOML memory files exist for learnings not already captured by UC-3. No duplicates of existing memories. Each learning classified as A/B/C using the same tier criteria as real-time capture.

**Actor:** System (Go binary triggered by PreCompact and SessionEnd hooks).

**Key interactions:**

- **Trigger — PreCompact + SessionEnd:** Fires on both events. PreCompact captures learnings before context window compression loses transcript detail. SessionEnd captures end-of-session learnings regardless of how the session ends (user quit, timeout, error). Both triggers invoke the same extraction pipeline.

- **LLM extraction with unified tier criteria:** A single API call (claude-haiku-4-5-20251001) receives the session transcript (or the portion about to be compacted for PreCompact) and produces a list of candidate learnings. Each candidate is classified using the same A/B/C tier criteria as UC-3's real-time classifier:

  | Tier | What | Anti-pattern | Example |
  |------|------|-------------|---------|
  | **A** | Explicit instruction the real-time path missed | Always generated | User said "always X" but fast-path didn't trigger |
  | **B** | Teachable correction | When generalizable (LLM decides) | Implicit correction pattern without imperative language |
  | **C** | Contextual fact | Never generated | Architectural decision, discovered constraint |

  Each candidate has the same structured fields as UC-3 memories: title, content, observation_type, concepts, keywords, principle, anti_pattern (tier-gated), rationale, filename_summary.

- **Extraction scope:** The LLM looks for:
  - Corrections the real-time classifier missed (observational corrections like "you didn't shut them down" that lack imperative language)
  - Architectural decisions made during the session
  - Discovered constraints (e.g., "targ's reorder step modifies files after tests run")
  - Working solutions and patterns that proved effective
  - Implicit preferences the user demonstrated but didn't explicitly state

- **Quality gate:** Extracted memories must be specific and actionable. The LLM prompt instructs rejection of: mechanical patterns (e.g., "ran tests before committing"), vague generalizations (e.g., "code should be clean"), and observations that are project-specific but too narrow to be useful again.

- **Deduplication against existing memories:** Before writing each candidate, check existing TOML files in the memories directory. Compare by keyword overlap and semantic similarity (via the LLM). If a candidate substantially overlaps an existing memory, skip it. UC-3 mid-session captures take priority — session-end extraction never duplicates what was already captured.

- **Confidence tiers:** Session-extracted learnings are classified as A, B, or C using the same criteria as UC-3. Most will be tier C (contextual facts, discovered constraints), but the LLM may identify missed tier-A or tier-B signals. Anti-pattern generation follows the same tier gating: A always, B when generalizable, C never.

- **No graceful degradation:** If no API token is configured, emit a loud stderr error (`[engram] Error: session learning skipped — no API token configured`) and do not create any memory files. Never write degraded memories.

- **TOML file output:** Same format and directory as UC-3. Memory written to `<data-dir>/memories/<slug>.toml`. The `confidence` field reflects the classified tier (A, B, or C).

- **Idempotency:** If both PreCompact and SessionEnd fire in the same session, the second invocation deduplicates against memories created by the first. Multiple PreCompact events in a long session each extract from the new transcript portion only.

- **Pure Go, no CGO:** Same constraint as UC-3.

---

Deferred UCs (UC-4 through UC-14) are archived in issue #18 for review.
