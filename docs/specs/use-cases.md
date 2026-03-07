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

- **Feedback (user-visible):** Memory creation is reported in the hook's `systemMessage` output so the user sees it in their terminal. Shows: the tier classification, the memory title, and the file path. This must appear in `systemMessage` regardless of whether surface matches co-occur — creation visibility is never buried in model-only context.

- **No graceful degradation:** If no API token is configured, emit a loud stderr error (`[engram] Error: memory capture skipped — no API token configured`) and create no file. Never write degraded memories. (Closes #32.)

- **Session-end extraction (UC-1)** deduplicates against mid-session corrections by checking existing files in the memories directory.

---

## UC-2: Hook-Time Surfacing & Enforcement

**Description:** Surface relevant memories at hook time as advisory system reminders. The agent uses these with full session context to exercise judgment.

**Starting state:** The memory store contains TOML files written by UC-3. A hook fires (SessionStart, UserPromptSubmit, or PreToolUse).

**End state:** Relevant memories are surfaced as system reminders at all three hook points (SessionStart, UserPromptSubmit, PreToolUse). The agent uses these advisories with full session context to exercise judgment. Each surfacing event is recorded in the memory's TOML file (count, timestamp, context type) for effectiveness measurement.

**Actor:** System (hook scripts invoke Go binary for retrieval and surfacing).

**Key interactions:**

- **SessionStart — passive surfacing:** Surface the top 20 memories by recency as a system reminder. No matching needed — recency is the only signal. Provides context priming at session start. The reminder lists each surfaced memory's title and file path so the user can inspect or edit them. Additionally, if a creation log exists (`<data-dir>/creation-log.jsonl`), report the memories created during prior sessions (by UC-1 at PreCompact/SessionEnd) in the `systemMessage` so the user sees what was learned. Clear the log after reporting.

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

- **Creation visibility (deferred):** PreCompact and SessionEnd hooks have no output mechanism to show the user what was created. Instead, creation events are logged to a file (`<data-dir>/creation-log.jsonl`) with timestamp, title, tier, and file path. UC-2's SessionStart surfacing reports these at the start of the next session so the user sees what was learned. The log is cleared after successful reporting.

- **Idempotency:** If both PreCompact and SessionEnd fire in the same session, the second invocation deduplicates against memories created by the first. Multiple PreCompact events in a long session each extract from the new transcript portion only.

- **Pure Go, no CGO:** Same constraint as UC-3.

---

## UC-15: Automatic Outcome Signal

**Description:** At context compaction and session end, automatically assess whether memories surfaced during the session were followed, contradicted, or ignored by reviewing the transcript. Write per-session evaluation results to a log file. When surfacing memories in future sessions, compute and display effectiveness annotations from evaluation history.

**Starting state:** A session is active. Memories were surfaced via UC-2 hooks (SessionStart, UserPromptSubmit, PreToolUse). A surfacing log file records which memories were surfaced and in what context. A PreCompact or SessionEnd hook fires.

**End state:** A per-session evaluation log file exists with outcome classifications for each surfaced memory. Future surfacing events display effectiveness annotations ("surfaced N times, followed M%") computed on-the-fly from evaluation logs.

**Actor:** System (Go binary triggered by PreCompact and SessionEnd hooks, after `engram learn`).

**Key interactions:**

- **Surfacing log (written by UC-2, read by UC-15):** During each surfacing event, write an entry to `<data-dir>/surfacing-log.jsonl` recording the memory file path, mode (session-start/prompt/tool), and timestamp. This is the session-scoped record of what was surfaced. The evaluate pass reads and clears this log.

- **Evaluation pass (new `engram evaluate` subcommand):** After `engram learn` completes in the PreCompact/SessionEnd hook, invoke `engram evaluate`. The evaluator:
  1. Reads the surfacing log to determine which memories were surfaced this session
  2. Reads each surfaced memory's TOML file to get its content, principle, and anti-pattern
  3. Sends the full transcript + surfaced memory list to an LLM (claude-haiku-4-5-20251001)
  4. The LLM classifies each surfaced memory's outcome: `followed` (agent acted consistently with the memory), `contradicted` (agent acted against the memory), or `ignored` (memory was surfaced but not relevant to any decision in the session)
  5. Writes results to a per-session evaluation log file at `<data-dir>/evaluations/<timestamp>.jsonl`

- **Per-session evaluation log format:** Each line is a JSON object:
  ```json
  {"memory_path": "...", "outcome": "followed|contradicted|ignored", "evidence": "brief LLM explanation", "evaluated_at": "RFC3339"}
  ```
  The session identity is implicit from the file. No unbounded growth — each session produces one small file.

- **Effectiveness annotations (read path):** When UC-2 surfaces memories, compute effectiveness on-the-fly by reading all evaluation log files in `<data-dir>/evaluations/`. For each surfaced memory, aggregate outcomes across sessions and display: "(surfaced N times, followed M%)". This adds no LLM cost — pure file reads and arithmetic.

- **Visibility:**
  - **SessionEnd summary:** The evaluate pass outputs a summary of outcomes for the current session (e.g., "3 memories surfaced: 2 followed, 1 ignored") via hook `systemMessage` so the user sees it.
  - **Inline annotations:** When memories surface in future sessions, effectiveness context appears alongside the memory: title, principle, and "(surfaced 5 times, followed 80%)".
  - **CLI `engram review`:** Shows per-memory effectiveness stats aggregated from all evaluation logs. Deferred to a later issue if scope is too large.

- **No graceful degradation:** If no API token is configured, emit a loud stderr error and skip evaluation. Never write degraded evaluations.

- **Idempotency:** If both PreCompact and SessionEnd fire, the second evaluate invocation reads an empty surfacing log (cleared by the first) and produces no evaluation file.

- **Pure Go, no CGO:** Same constraint as UC-1/2/3.

---

## UC-6: Memory Effectiveness Review

**Description:** Classify memories into a 2x2 effectiveness matrix (working/leech/hidden gem/noise) based on surfacing frequency and outcome signals. Flag memories for action when effectiveness drops below threshold. Display effectiveness annotations when memories surface. Provide a CLI review command for the full matrix.

**Starting state:** Memories exist with surfacing instrumentation (UC-2) and evaluation history (UC-15). Evaluation log files in `<data-dir>/evaluations/` contain per-session outcome classifications.

**End state:** Each memory is classified into one of four quadrants. Memories below the effectiveness threshold are flagged for action. Surfacing events include effectiveness annotations. The `engram review` command displays the full matrix.

**Actor:** System (Go binary, `engram review` CLI + effectiveness annotations wired into UC-2 surfacing).

**Key interactions:**

- **2x2 matrix classification:** Combine two signals per memory:
  - **Surfacing frequency** from tracking fields: `surfaced_count` (high = above median, low = at or below median across all memories)
  - **Follow-through rate** from evaluation aggregation: `EffectivenessScore` (high = >= 50%, low = < 50%)

  |  | Often Surfaced | Rarely Surfaced |
  |--|---|---|
  | **High Follow-Through** | **Working** — maintain | **Hidden Gem** — broaden triggers |
  | **Low Follow-Through** | **Leech** — diagnose and fix | **Noise** — prune candidate |

  Memories with fewer than 5 evaluations are classified as **insufficient data** — no quadrant assignment, no action flagged.

- **Threshold flagging:** Flag a memory for action when: (a) it has 5+ evaluations, AND (b) its effectiveness score is below 40%. Flagged memories are reported in `engram review` output with their quadrant and stats. This implements the CLAUDE.md rule: "Pruned when utility < 0.4 after 5+ retrievals."

- **Effectiveness annotations (surfacing path):** When UC-2 surfaces memories, annotate each with effectiveness context if evaluation data exists: "(surfaced N times, followed M%)". Computed on-the-fly from evaluation logs — no LLM, no pre-computation. Fire-and-forget: annotation failures never break surfacing (ARCH-6 exit-0). Memories with no evaluation history show no annotation.

- **`engram review` CLI command:** New subcommand that reads tracking data + evaluation logs and outputs:
  1. Per-quadrant summary (count of memories in each quadrant)
  2. Flagged memories with stats (name, quadrant, surfaced count, effectiveness score, evaluation count)
  3. Insufficient-data memories (name, surfaced count, evaluation count)

  Output is human-readable text to stdout. Machine-readable `--format json` deferred unless needed by downstream UCs.

- **No graceful degradation:** If evaluation directory is missing or empty, `engram review` reports "no evaluation data" and exits 0. No degraded classifications.

- **No LLM calls:** All classification is pure arithmetic on existing data. Zero API cost.

- **Pure Go, no CGO.**

---

## UC-14: Structured Session Continuity

**Description:** Incrementally maintain a task-focused working summary that survives session boundaries (`/clear`, `/exit`, context compaction). Piggyback on existing Haiku API calls to minimize latency. Automatically restore the summary at session start.

**Starting state:** A session is active. The UserPromptSubmit hook already calls Haiku for memory classification (UC-3). A transcript JSONL file exists at `transcript_path`.

**End state:** A human-readable session context file exists at `.claude/engram/session-context.md` containing a task-focused working summary. On next SessionStart, the summary is injected as `additionalContext`.

**Actor:** System (Go binary triggered by UserPromptSubmit and SessionStart hooks).

**Key interactions:**

- **Incremental context update (piggybacked on UserPromptSubmit):** Each time the UserPromptSubmit hook fires, a parallel Haiku call runs concurrently with the existing `correct` classification:
  1. Read transcript from `transcript_path`, starting from the byte offset watermark stored in the context file
  2. Strip low-value content: tool results, base64/binary, repeated schemas. Keep: user messages, assistant text, tool names, errors
  3. If the stripped delta is non-empty, send `{previous_summary + stripped_delta}` to Haiku with prompt: "Update this task-focused working summary. Focus on what's being worked on, decisions made, progress, and open questions. Not a dissertation — just what's relevant."
  4. Write updated summary to `.claude/engram/session-context.md`

- **Final flush (PreCompact):** Same pipeline as UserPromptSubmit, ensures any remaining transcript delta is captured before context compaction.

- **Restore on SessionStart:** If `.claude/engram/session-context.md` exists, read it and inject contents as `additionalContext`. Always load regardless of age — user can delete the file to clear it.

- **Context file format:** Plain markdown with HTML comment metadata (invisible when rendered). User can read, edit, or `rm` it.

```markdown
<!-- engram session context | updated: 2026-03-07T03:15:00Z | offset: 34521 | session: abc123 -->

Working on session continuity for engram (#45)...
```

- **Summary scope:** Task-focused only — what's being worked on, decisions made, progress, and open questions. NOT discovered constraints or patterns (those are captured as memories by UC-3).

- **No hard size limit:** Haiku decides what's relevant. Natural summarization keeps it concise.

- **No graceful degradation:** If no API token, skip the context update silently (the `correct` call already emits the loud error). Never write a degraded summary.

- **File location:** `.claude/engram/session-context.md` — local to the project, visible in the file tree, easily deletable by the user.

- **Pure Go, no CGO.**

---

Deferred UCs (UC-4 through UC-13, excluding UC-6) are archived in issue #18 for review.
