# Use Cases

## UC-3: Remember & Correct

**Description:** When the user says "remember XYZ" or explicitly corrects the agent, turn the input into a structured, enriched memory file that is human-readable, editable, and shareable.

**Starting state:** The user explicitly tells the agent to remember something, or corrects a mistake the agent made.

**End state:** An enriched TOML memory file exists in the memories directory with structured metadata. A system reminder confirms what was captured and where the file is.

**Actor:** System (Go binary triggered by UserPromptSubmit hook).

**Key interactions:**
- **Detection (UserPromptSubmit):** Deterministic pattern matcher checks the user's prompt for correction/remember signals (~15 patterns covering ~85% of explicit corrections: `^no,`, `^wait`, `^hold on`, `\bwrong\b`, `\bdon't\s+[verb]`, `\bstop\s+[verb]ing`, `\btry again`, `\bgo back`, `\bthat's not`, `^actually,`, `\bremember\s+(that|to)`, `\bstart over`, `\bpre-?existing`, `\byou're still`, `\bincorrect`). On match → LLM enrichment.
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

Deferred UCs (UC-1, UC-2, UC-4 through UC-14) are archived in issue #18 for review after UC-3 implementation.
