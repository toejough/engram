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

**End state:** Relevant memories are surfaced as system reminders (SessionStart, UserPromptSubmit), or a tool call is blocked with a corrective message when it violates a memory's anti-pattern (PreToolUse).

**Actor:** System (hook scripts invoke Go binary for retrieval and enforcement).

**Key interactions:**

- **SessionStart — passive surfacing:** Surface the top 20 memories by recency as a system reminder. No matching needed — recency is the only signal. Provides context priming at session start. The reminder lists each surfaced memory's title and file path so the user can inspect or edit them.

- **UserPromptSubmit — passive surfacing:** Keyword/concept match the user's message against memory `keywords` and `concepts` fields. Surface matching memories as a system reminder alongside the existing UC-3 correction detection. No blocking — informational only. The reminder lists each matched memory's title, file path, and which keywords matched.

- **PreToolUse — two-pass enforcement:**
  1. **Keyword pre-filter (fast, no LLM):** Scan memory TOML files. For each memory with an `anti_pattern` field, check if any of its `keywords` appear in the tool name or tool input. Most tool calls won't match → zero overhead.
  2. **LLM judgment (haiku, candidates only):** For memories that pass the keyword pre-filter, make a single LLM call: "Given this tool call (tool name, arguments) and this memory (principle, anti_pattern), is the anti-pattern being violated?" The LLM can distinguish context — e.g., `/commit` skill calling `git commit` is fine, agent hand-rolling `git commit` is a violation.
  3. **Decision:** If the LLM says violated → return `{"decision": "block", "reason": "<principle from memory>"}` including the memory title and file path so the user can review or edit the rule. Otherwise → allow silently.

- **Graceful degradation with notification:** If no API token is configured or the LLM call times out, allow the tool call but emit a stderr warning: `[engram] Warning: enforcement skipped (no token / timeout). Tool call allowed.` Never block when we can't judge.

- **Pure Go, no CGO:** Same constraint as UC-3. Keyword matching is string operations on TOML files.

- **Memory format unchanged:** Reads the same TOML files UC-3 writes. The `keywords`, `anti_pattern`, and `principle` fields already exist and drive the matching/enforcement.

- **Ranking deferred:** Frecency (recency × impact) ranking is deferred until UC-6 provides evaluation/impact signals. See issue #18 comment for details.

---

Deferred UCs (UC-1, UC-4 through UC-14) are archived in issue #18 for review.
