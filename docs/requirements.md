# Requirements

Requirements extracted from validated use cases. Each REQ-N traces to one or more UC-N.

---

## Session Learning (UC-1)

**REQ-1: Stop hook triggers extraction via Go binary.**
When a session ends (Stop hook fires), the system must invoke the Go binary with the session transcript to extract learnings.

- Traces to: UC-1
- AC: (1) Stop hook script exists and invokes the Go binary. (2) Session transcript is passed as input. (3) Extraction produces zero or more memories.
- Verification: deterministic (hook fires, binary invoked, memories written)

**REQ-2: Extracted memories have structured metadata and LLM-generated keywords.**
Each extracted memory must include structured metadata (observation_type, concepts, principle, anti_pattern, rationale, enriched_content) and LLM-generated keywords for local similarity retrieval, produced via LLM enrichment at extraction time.

- Traces to: UC-1
- AC: (1) LLM enrichment runs on each extracted learning. (2) All six metadata fields are populated. (3) Keywords are LLM-generated terms and phrases, not verbatim transcript excerpts. (4) Extracted content passes quality gate — memories must be specific and actionable; mechanical patterns and vague generalizations are rejected before storage.
- Verification: sonnet (LLM enrichment) + deterministic (schema validation of stored memory)

**REQ-3: Confidence tiers based on observable validation conditions.**
Each extracted memory must be assigned a confidence tier: A (user explicitly stated the content), B (agent inferred and content was visible to user during session — user had opportunity to correct but didn't), C (agent inferred post-session from transcript patterns — user never saw it).

- Traces to: UC-1
- AC: (1) Every memory has exactly one confidence tier. (2) Tier assignment is based on the observable condition (was user present and able to correct?), not a label.
- Verification: haiku (tier classification from transcript context)

**REQ-4: Retrieval ranking — frecency primary, confidence tiebreaker.**
Results are ranked by frecency (recency × impact harmonic mean) as the primary sort, with confidence tier (A > B > C) as tiebreaker when frecency is equal. During cold start (insufficient evaluation data), impact defaults to a neutral baseline so recency dominates. As evaluation data accumulates, impact becomes the dominant signal.

- Traces to: UC-1, UC-2
- AC: (1) Given two memories of equal frecency, the higher-confidence memory ranks higher. (2) With no evaluation data, ranking equals recency. (3) With evaluation data, a memory surfaced 10 times and never followed ranks below a memory surfaced 3 times and followed every time.
- Verification: deterministic (ranking comparisons)

**REQ-5: Reconciliation before insert via candidate retrieval and haiku gate.**
Before inserting a new memory, the system retrieves the top-K most similar existing memories by local similarity (K defaults to 3, user-configurable) and uses a haiku LLM gate to decide merge vs. create. If haiku identifies genuine overlap, the existing memory is enriched rather than a duplicate being created. Self-correction: merged memories whose frecency declines (surfaced but not followed) naturally drop out of future surfacing — bad merges are contained without manual review.

- Traces to: UC-1
- AC: (1) Local similarity candidate retrieval runs before every insert. (2) Haiku evaluates top-K candidates for genuine overlap. (3) Overlap → existing memory enriched with new context. (4) No overlap → new memory created. (5) Each reconciliation decision logged: memory_id, similarity scores, haiku decision, rationale.
- Verification: local similarity (candidate retrieval) + haiku (overlap decision) + deterministic (memory written/updated, event logged)

**REQ-6: Extraction is implemented as a Go binary.**
Session extraction, retrieval, and scoring operations are implemented in a Go binary (pure Go, no CGO).

- Traces to: UC-1, UC-2
- AC: (1) A compiled Go binary exists. (2) It handles extraction, retrieval, and scoring commands. (3) No CGO dependencies.
- Verification: deterministic (build succeeds without CGO, binary runs)

---

## Hook-Time Surfacing (UC-2)

The three hooks form a retrieval gradient by budget: SessionStart surfaces the most memories (broadest context, no task to narrow on), UserPromptSubmit surfaces fewer (task-scoped), PreToolUse surfaces the fewest (latency-critical). All hooks rank results by frecency (recency × impact harmonic mean) and enforce a user-configurable result budget. The adaptation happens in the frecency scores — memories surfaced but not followed see their impact drop, their frecency drop, and they naturally fall out of the top-K — not in the budgets themselves.

**REQ-7: SessionStart surfaces top-K memories by frecency against project context.**
At SessionStart, the hook invokes the Go binary to retrieve the top-K memories ranked by frecency (REQ-4), queried via local similarity against project context: directory name, project CLAUDE.md, and README. K defaults to 5 (user-configurable).

- Traces to: UC-2
- AC: (1) SessionStart hook invokes Go binary with project context tokens. (2) Results ranked by frecency. (3) At most K results surfaced as system reminders.
- Verification: deterministic (hook fires, ranking order, budget enforced)

**REQ-8: UserPromptSubmit surfaces top-K memories by frecency against user's message.**
At UserPromptSubmit, the hook invokes the Go binary to retrieve the top-K memories ranked by frecency (REQ-4), queried via local similarity against the user's current message. K defaults to 3 (user-configurable).

- Traces to: UC-2
- AC: (1) UserPromptSubmit hook invokes Go binary with the user's message. (2) Local similarity retrieval uses the message as the query, ranked by frecency. (3) At most K results surfaced as system reminders.
- Verification: deterministic (hook fires, ranking order, budget enforced)

**REQ-9: PreToolUse surfaces top-K memories by frecency against tool context.**
At PreToolUse, the hook invokes the Go binary to retrieve the top-K memories ranked by frecency (REQ-4), queried via local similarity against tool context. K defaults to 1 (user-configurable). This is the latency-critical path.

- Traces to: UC-2
- AC: (1) PreToolUse hook invokes Go binary with tool context. (2) Local similarity retrieval runs with frecency ranking. (3) At most K results surfaced as system reminders.
- Verification: deterministic (hook fires, ranking order, budget enforced)

**REQ-10: All hook-time retrieval uses local similarity and frecency only — no LLM calls.**
SessionStart, UserPromptSubmit, and PreToolUse retrieval must use local similarity and frecency ranking only. No LLM calls at retrieval time. Retrieval quality depends entirely on write-time enrichment (REQ-2) and evaluation-time scoring (Group B), not on retrieval-time judgment.

- Traces to: UC-2
- AC: (1) Hook-time retrieval completes without any API calls to an LLM provider. (2) Improving retrieval quality requires improving extraction enrichment (REQ-2) or evaluation scoring, not adding LLM calls to the retrieval path.
- Verification: deterministic (no LLM calls in retrieval path)

**REQ-11: *Dissolved into REQ-10.***
Previously "Retrieval quality depends on write-time enrichment, not retrieval-time judgment." This was a design principle restating the consequence of REQ-10, not a separately testable requirement. Its content is now incorporated as REQ-10's rationale and AC(2).

**REQ-12: Surfaced memories appear as system reminders in the agent's context.**
All hook-time surfacing delivers memories as system reminder text that the agent sees as part of its context, not as separate tool output or side-channel.

- Traces to: UC-2
- AC: Surfaced memories appear in `<system-reminder>` tags in the agent's context window.
- Verification: deterministic (output format check)

---

## Correction Detection & Response (UC-3)

**REQ-13: Inline correction detection via deterministic pattern matching.**
When a user message is submitted (UserPromptSubmit hook), the system runs it against a persisted correction pattern corpus. If a pattern matches, the message is flagged as a correction for reconciliation, reclassification, and feedback.

Initial pattern corpus (15 patterns, derived from 6-week session log analysis covering ~85% of explicit corrections):
1. `^no,` — direct negation
2. `^wait` — interruption
3. `^hold on` — interruption
4. `\bwrong\b` — explicit wrongness
5. `\bdon't\s+\w+` — prohibition (don't + verb)
6. `\bstop\s+\w+ing` — cease action (stop + gerund)
7. `\btry again` — retry request
8. `\bgo back` — reversal request
9. `\bthat's not` — negation of output
10. `^actually,` — correction opener
11. `\bremember\s+(that|to)` — re-teaching
12. `\bstart over` — full reset
13. `\bpre-?existing` — flagging prior state
14. `\byou're still` — persistent error
15. `\bincorrect` — explicit wrongness

The corpus is persisted across sessions and grows via session-end catch-up (REQ-15).

- Traces to: UC-3
- AC: (1) Pattern corpus persists on disk with at least the 15 patterns above. (2) Pattern matching runs at every UserPromptSubmit. (3) On match, downstream processing is triggered (REQ-14, REQ-16, REQ-17).
- Verification: deterministic (pattern match)

**REQ-14: Memory reconciliation on detected correction via local similarity candidates and haiku gate.**
When an inline correction is detected (REQ-13), the system retrieves the top-K most similar existing memories by local similarity (K defaults to 3, user-configurable) and evaluates each with a haiku LLM gate for genuine overlap. If haiku identifies overlap, the existing memory is enriched (trigger terms, refined observation, anti-patterns, rationale, concrete examples). If no overlap, a new enriched memory is created.

The K is a budget — how many candidates haiku evaluates. Corrections are infrequent; K haiku calls per correction is negligible cost. No similarity floor — if the best candidates are poor matches, haiku rejects them. Self-correction operates through frecency: a bad merge produces a memory that gets surfaced but not followed, its impact drops, its frecency drops, and it falls out of future surfacing.

- Traces to: UC-3
- AC: (1) Local similarity retrieves top-K candidates. (2) Haiku evaluates each for genuine overlap. (3) Overlap → existing memory enriched with correction context. (4) No overlap → new enriched memory created. (5) Each reconciliation event logged: memory_id, similarity scores, haiku decision, rationale.
- Verification: local similarity (candidate retrieval) + haiku (overlap decision) + deterministic (memory written/updated, event logged)

**REQ-15: Session-end catch-up for missed corrections.**
At session end (Stop hook), the system evaluates the transcript via LLM for corrections the pattern matcher missed. Newly discovered correction phrases are added to the persisted pattern corpus, so the matcher self-improves across sessions.

- Traces to: UC-3
- AC: (1) LLM identifies corrections not already captured mid-session. (2) New correction phrases are appended to the pattern corpus. (3) Corpus persists across sessions.
- Verification: haiku (LLM evaluation) + deterministic (corpus file updated)

**REQ-16: Immediate reclassification of artifacts on correction.**
When an inline correction is detected, the system immediately reclassifies artifacts (memories, skills, CLAUDE.md entries) that were surfaced during this session and contributed to the agent's incorrect behavior — decreasing their impact score now, not at session-end.

- Traces to: UC-3, UC-8
- AC: (1) Surfacing log for the current session is available. (2) Artifacts surfaced before the correction are evaluated for contribution to the error. (3) Contributing artifacts have their impact score decreased immediately.
- Verification: deterministic (score comparison before/after)

**REQ-17: Correction feedback via system reminder.**
When an inline correction is detected and reconciled, the system injects a system reminder showing: (1) the correction detected, (2) the memory created or enriched, and (3) the keywords added for future retrieval.

- Traces to: UC-3
- AC: System reminder text appears in the agent's context in the same UserPromptSubmit response that flagged the correction.
- Verification: deterministic (system reminder present in hook output)

**REQ-18: Mid-session/session-end deduplication.**
Session-end extraction (UC-1) must not duplicate corrections already captured mid-session (UC-3). Memories created or enriched via inline correction are either excluded from session-end extraction or reconciled if overlapping.

- Traces to: UC-1, UC-3
- AC: After a session with inline corrections, no memory contains the same correction content twice.
- Verification: deterministic (deduplication check) + local similarity (overlap detection)

---

## User-Configurable Budgets

All result budgets and candidate counts are user-configurable via plugin settings. Defaults are sensible starting points:

| Parameter | Default | Used by |
|-----------|---------|---------|
| `surfacing.session_start.budget` | 5 | REQ-7 |
| `surfacing.user_prompt.budget` | 3 | REQ-8 |
| `surfacing.pre_tool_use.budget` | 1 | REQ-9 |
| `reconciliation.candidate_count` | 3 | REQ-5, REQ-14 |

---

## Skill Promotion (UC-4)

**REQ-19: Skill promotion criteria — token economics, not clustering.**
A memory becomes a skill promotion candidate when ALL FOUR conditions are met. Promotion serves the plugin's purpose: a skill provides richer procedural guidance (fewer corrections), loads once instead of surfacing repeatedly (fewer tokens), and has precise trigger descriptions (faster to right answer). Promotion is proposed to the user, not automatic.

**Condition 1 — Token economics: memory surfacing cost exceeds skill cost.**
Tracked per memory over a trailing window of 10 sessions:
- Memory surfacing cost = sum of (surfacing_count_per_session × memory_token_count) across the window.
- Skill cost estimate = (window_sessions × skill_description_tokens) + (estimated_invocation_sessions × skill_content_tokens).
  - `skill_description_tokens`: If 0 existing skills, use 256 (standard estimate). If 1+ existing skills, use measured median description token count across all skills.
  - `skill_content_tokens`: If 0 existing skills, use 2000 (standard estimate). If 1+ existing skills, use measured median content token count across all skills.
  - `estimated_invocation_sessions`: sessions_where_memory_surfaced × memory_followed_rate (proxy for when the skill would be triggered).
- Threshold: memory surfacing cost > skill cost estimate for the same trailing window.

**Condition 2 — Procedural complexity: content has outgrown memory format.**
A memory has outgrown the memory format when ANY of:
- Content exceeds 500 tokens, OR
- Enrichment count ≥ 3 (3+ corrections/observations have been merged into this memory via REQ-14), OR
- Content contains structural indicators: numbered steps, conditional logic ("if X then Y"), or multi-step procedures.

**Condition 3 — Followed rate: memory is consistently effective when surfaced.**
- Measured by haiku post-session evaluation: "did agent behavior align with this memory's guidance?"
- Three signals per surfacing: followed, contradicted, irrelevant.
- Followed rate = followed_count / (followed_count + contradicted_count). Irrelevant surfacings are excluded from the denominator (they indicate retrieval mismatch, not effectiveness).
- Threshold: followed rate ≥ 80%.
- Minimum sample: memory must have been surfaced in ≥ 5 sessions with non-irrelevant evaluations before the followed rate is considered meaningful.

**Condition 4 — Cross-session spread: memory is valuable across contexts.**
- Surfaced AND followed in ≥ 3 distinct sessions.
- Prevents promotion of memories that are contextually useful in one situation but not broadly applicable.

**Measurements tracked per memory (inputs to all four conditions):**
- Per surfacing event: memory_id, session_id, hook_type, memory_token_count, timestamp.
- Per session aggregate: surfacing_count, followed/contradicted/irrelevant counts.
- Cumulative: total_surfacing_tokens (trailing 10 sessions), enrichment_count, content_token_count, distinct_sessions_followed.

**Measurements tracked per skill (inputs to condition 1 cost estimates):**
- Per skill: description_token_count, content_token_count.
- Per session: was the skill invoked? (binary)
- Aggregate: median_description_tokens, median_content_tokens across all skills.

- Traces to: UC-4
- AC: (1) All four conditions evaluated simultaneously. (2) Token cost comparison uses real skill data when available, standard estimates when not. (3) Followed rate excludes irrelevant from denominator and requires ≥5 non-irrelevant sessions. (4) Procedural complexity checked by content size, enrichment count, or structural indicators. (5) Cross-session spread requires ≥3 distinct sessions followed. (6) System proposes promotion to user with supporting data (surfacing cost, followed rate, complexity indicators, session spread).
- Verification: deterministic (threshold comparisons on tracked measurements)

**REQ-20: RED/GREEN behavioral test scenarios before skill deployment.**
Before a promoted skill is deployed, the system must generate RED and GREEN test scenarios that validate the skill's behavioral impact, not just its trigger/discovery.

- **RED test:** Without the skill (and without the source memories), the target behavior does NOT happen. This proves the skill is necessary — it fills a gap that nothing else covers.
- **GREEN test:** With the skill loaded, the target behavior DOES happen. This proves the skill is effective — it actually produces the outcome it claims to.

Scenarios are generated from the source memory's surfacing history (contexts where it was relevant for RED; contexts where it was followed for GREEN).

- Traces to: UC-4
- AC: (1) RED scenarios generated from memory's surfacing history. Without skill/memories present, agent fails to exhibit target behavior. (2) GREEN scenarios validate that with skill loaded, agent exhibits target behavior. (3) Skill passes all scenarios before being proposed to the user. (4) Scenarios are persisted alongside the skill for re-validation after revisions (UC-6).
- Verification: haiku (scenario generation) + sonnet (behavioral evaluation in RED/GREEN runs)

**REQ-21: CLAUDE.md pointer coupled to skill lifecycle.**
When a skill is created from a memory, a CLAUDE.md pointer (a one-liner referencing the skill) must be created simultaneously. The pointer is coupled to the skill lifecycle — created, updated, and removed together with the skill.

- Traces to: UC-4, UC-5
- AC: (1) Skill creation produces both a skill file and a CLAUDE.md pointer in the same operation. (2) Skill removal also removes the pointer. (3) Skill revision triggers pointer review. (4) No orphaned pointers or pointer-less skills.
- Verification: deterministic (coupling check)

---

## Remaining Extraction

Requirements for UC-5 through UC-14 have not yet been extracted. These will be addressed when their respective UC groups (B, C, D) become active.
