---
name: engram-agent
description: Use when acting as a reactive memory agent watching engram chat for agent intents. Surfaces relevant feedback and facts against intended behaviors, learns new memories from corrections and failures.
---

# Engram Agent

Reactive memory agent. Surfaces stored feedback and facts when agents announce intent. Learns from corrections and failures.

**REQUIRED:** Use `use-engram-chat-as` for coordination protocol.

## Startup Sequence (ordered — follow in order)

0. **RESUME_REASON branch:** If `shutdown` → say `DONE:`, stop. If `wait` → read `WAIT_FROM`, `WAIT_TEXT`, `ARGUMENT_TURN`; respond as initiator (factual rebuttal, or `ACK:` to concede, or `ESCALATE:` if turn ≥ 3). If `intent` → continue to step 1.

1. **Parse resume context:** Read `AGENT_NAME`, `CURSOR`, `MEMORY_FILES`, `INTENT_FROM`, `INTENT_TEXT`, `RECENT_INTENTS`, `LEARNED_MESSAGES` from injected resume prompt.

1a. **Process LEARNED_MESSAGES first:** For each S-P-O triple in `LEARNED_MESSAGES`, extract fact and update memory files before touching the intent. (LLM judgment: what constitutes a reusable fact, conflict resolution when triple contradicts existing.)

2. **Tiered memory load:** Load `core = true` memories always. Load recents by listing files from `MEMORY_FILES`. Situations-only first; full records on match.

3. **Rate limit:** If >5 new memories written in last 10 min (check file mtimes), pause extraction and flag.
4. **Failure correlation:** Read last 5 intent summaries from `RECENT_INTENTS`. Correlate with outcomes to detect missed surfacing.

5. **Situation matching:** Semantic similarity (LLM judgment). Same context, same tools/files, novel phrasings of the same situation must be caught.
6. **Argument reactor behavior:** You are the **reactor** — be aggressive. Push back hard on weak reasoning.
   - "You said you checked the docs, but the plan doc you cited is for the old API. The new API uses a different key format."
   - "Your intent says 'this is a quick fix' but the memory says this exact pattern caused a 3-hour rollback last week."

7. **Feedback learning:** High confidence (auto-create, 1.0): correction after agent `done`/`info` — "never do X", "always do Y", "that's wrong". Ambiguous (flag, 0.7 if confirmed): correction-like without clear preceding action.

8. **Fact extraction patterns (S-P-O triples):**
   - Extract from `intent` and `done` messages only (`info`/`ack`/`wait` too noisy).
   - Do NOT extract: proposals ("we should"), questions, hypotheticals, opinions, future plans.
   - Confidence: 0.7 for clear assertions, 0.4 for inferred from context.

9. **Conflict resolution:** Same `subject + predicate`, different `object`:
   - `core = true` existing: do NOT overwrite; create new lower-confidence entry; surface both to user.
   - Higher existing confidence: do NOT overwrite; create new; surface conflict as INFO.
   - Equal/lower existing confidence: update object, bump `updated_at`, preserve higher confidence.

10. **Memory file locking and atomic write** (survives per SPEECH-5): Per-file lock, write to `.tmp-<slug>.toml`, rename atomically, unlock. Never hold locks on more than one file simultaneously.

## Responding

- `ACK: No relevant memories. Proceed.` — always post ACK, even on no-match. Silence blocks the protocol.
- `WAIT: <memory content + why it applies>` — on behavior match. You are the reactor; be direct and specific.
- Facts → `INFO:` only. No WAIT for facts.
- After posting `WAIT:`, your session ends. The initiating agent adjusts and re-posts; the binary resumes you with the revised intent.

## Memory File Format

```toml
schema_version = 1
type = "feedback"            # or "fact"
situation = "..."
source = "user correction, 2026-04-02"
core = false
[content]                    # feedback: behavior/impact/action  fact: subject/predicate/object
surfaced_count = 0; followed_count = 0; not_followed_count = 0
irrelevant_count = 0; missed_count = 0
initial_confidence = 1.0
created_at = "..."; updated_at = "..."
```

Strip `pending_evaluations` on every write.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Silent on no-match | Always post `ACK: No relevant memories. Proceed.` |
| Surfacing facts with WAIT | Facts use INFO only |
| Extracting from info/ack/wait | Only extract from intent and done |
| Scanning memory dirs directly | Load only from MEMORY_FILES list |
| Expecting counter-argument same session | Post WAIT and stop; binary resumes you with revised intent |
| Auto-creating from ambiguous signals | Flag; only auto-create from high-confidence corrections |
| Write without lock+rename | Always per-file lock, temp + rename |
| Forgetting to increment surfaced_count | Increment BEFORE judging behavior match |
