# Use Cases

## UC-3: Remember & Correct

**Description:** When the user says "remember XYZ" or explicitly corrects the agent, turn the input into a structured, enriched memory file that is human-readable, editable, and shareable.

**Starting state:** The user explicitly tells the agent to remember something, or corrects a mistake the agent made.

**End state:** An enriched TOML memory file exists in the memories directory with structured metadata. A system reminder confirms what was captured and where the file is.

**Actor:** System (Go binary triggered by UserPromptSubmit hook).

**Key interactions:**
- **Detection (UserPromptSubmit):** Deterministic pattern matcher checks the user's prompt for correction/remember signals (40 patterns across 13 categories: direct corrections, interruptions, prohibitions, negations, re-teaching, omission feedback, standing instructions, retrospective corrections, repeated instructions, contrast/preference, rejection, and prospective corrections). On match → LLM enrichment.
- **LLM enrichment:** A single API call (claude-haiku-4-5-20251001) takes the user's message and produces structured memory fields as JSON: title, content, observation_type, concepts, keywords, principle, anti_pattern, rationale, and a 3-5 word filename summary. The Go code parses the JSON response and writes TOML.
- **TOML file output:** Memory written to `<data-dir>/memories/<slug>.toml` where slug is the slugified filename summary (3-5 hyphenated words). Example:

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

- **Feedback:** System reminder injected showing: the correction/remember detected, the memory file created, key fields captured, and the file path.
- **Graceful degradation:** If no API key is configured (`ANTHROPIC_API_KEY`), create a degraded memory with the raw message as content, minimal metadata, and filename from first few words. Log a warning. Don't error.
- **Confidence tiers:** A (user explicitly stated — "remember X"), B (user correction — "no, do Y instead").
- Session-end extraction (deferred UC-1) deduplicates against mid-session corrections by checking existing files in the memories directory.

---

## UC-2: Hook-Time Surfacing & Enforcement

**Description:** Surface relevant memories at hook time and enforce behavioral anti-patterns by blocking tool calls that violate learned principles.

**Starting state:** The memory store contains TOML files written by UC-3. A hook fires (SessionStart, UserPromptSubmit, or PreToolUse).

**End state:** Relevant memories are surfaced as system reminders at all three hook points (SessionStart, UserPromptSubmit, PreToolUse). The agent uses these advisories with full session context to exercise judgment.

**Actor:** System (hook scripts invoke Go binary for retrieval and enforcement).

**Key interactions:**

- **SessionStart — passive surfacing:** Surface the top 20 memories by recency as a system reminder. No matching needed — recency is the only signal. Provides context priming at session start. The reminder lists each surfaced memory's title and file path so the user can inspect or edit them.

- **UserPromptSubmit — passive surfacing:** Keyword/concept match the user's message against memory `keywords` and `concepts` fields. Surface matching memories as a system reminder alongside the existing UC-3 correction detection. No blocking — informational only. The reminder lists each matched memory's title, file path, and which keywords matched.

- **PreToolUse — advisory surfacing:**
  1. **Keyword pre-filter (fast, no LLM):** Scan memory TOML files. For each memory with an `anti_pattern` field, check if any of its `keywords` appear in the tool name or tool input. Most tool calls won't match → zero overhead.
  2. **Advisory output:** For memories that pass the keyword pre-filter, emit a `<system-reminder>` listing each matched memory's title, principle, and file path. No blocking, no LLM call. Claude exercises judgment with full session context — better accuracy than haiku judging in isolation.

- **No graceful degradation needed:** No LLM calls in the PreToolUse path means no API token requirement and no timeout failure mode.

- **Pure Go, no CGO:** Same constraint as UC-3. Keyword matching is string operations on TOML files.

- **Memory format unchanged:** Reads the same TOML files UC-3 writes. The `keywords`, `anti_pattern`, and `principle` fields already exist and drive the matching/enforcement.

- **Ranking deferred:** Frecency (recency × impact) ranking is deferred until UC-6 provides evaluation/impact signals. See issue #18 comment for details.

---

## UC-1: Session Learning

**Description:** At context compaction and session end, review the session transcript with an LLM to extract learnings that weren't captured by real-time correction detection (UC-3). Write them as enriched TOML memory files.

**Starting state:** A session is active and a PreCompact or SessionEnd hook fires. The transcript contains corrections the pattern matcher missed, architectural decisions, discovered constraints, working solutions, and implicit patterns.

**End state:** New enriched TOML memory files exist for learnings not already captured by UC-3. No duplicates of existing memories.

**Actor:** System (Go binary triggered by PreCompact and SessionEnd hooks).

**Key interactions:**

- **Trigger — PreCompact + SessionEnd:** Fires on both events. PreCompact captures learnings before context window compression loses transcript detail. SessionEnd captures end-of-session learnings regardless of how the session ends (user quit, timeout, error). Both triggers invoke the same extraction pipeline.

- **LLM extraction:** A single API call (claude-haiku-4-5-20251001) receives the session transcript (or the portion about to be compacted for PreCompact) and produces a list of candidate learnings. Each candidate has the same structured fields as UC-3 memories: title, content, observation_type, concepts, keywords, principle, anti_pattern, rationale, filename_summary.

- **Extraction scope:** The LLM looks for:
  - Corrections the pattern matcher missed (~15% — observational corrections like "you didn't shut them down" that lack imperative language)
  - Architectural decisions made during the session
  - Discovered constraints (e.g., "targ's reorder step modifies files after tests run")
  - Working solutions and patterns that proved effective
  - Implicit preferences the user demonstrated but didn't explicitly state

- **Quality gate:** Extracted memories must be specific and actionable. The LLM prompt instructs rejection of: mechanical patterns (e.g., "ran tests before committing"), vague generalizations (e.g., "code should be clean"), and observations that are project-specific but too narrow to be useful again.

- **Deduplication against existing memories:** Before writing each candidate, check existing TOML files in the memories directory. Compare by keyword overlap and semantic similarity (via the LLM). If a candidate substantially overlaps an existing memory, skip it. UC-3 mid-session captures take priority — session-end extraction never duplicates what was already captured.

- **Confidence tier C:** All session-extracted learnings are tier C ("agent-inferred post-session — user never saw the extraction, zero validation"). This is the lowest confidence tier, below A (user explicitly stated) and B (user correction). Tier C memories may be surfaced with lower priority and are candidates for pruning if never validated.

- **No graceful degradation:** If no API token is configured, emit a loud stderr warning (`[engram] Error: session learning skipped — no API token configured`) and do not create any memory files. Never write degraded memories. See also issue #32 for aligning UC-3 to this policy.

- **TOML file output:** Same format and directory as UC-3. Memory written to `<data-dir>/memories/<slug>.toml`. The `confidence = "C"` field distinguishes session-extracted from real-time-captured memories.

- **Idempotency:** If both PreCompact and SessionEnd fire in the same session, the second invocation deduplicates against memories created by the first. Multiple PreCompact events in a long session each extract from the new transcript portion only.

- **Pure Go, no CGO:** Same constraint as UC-3.

---

Deferred UCs (UC-4 through UC-14) are archived in issue #18 for review.
