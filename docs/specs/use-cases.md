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

**Description:** Incrementally extract learnings from session transcript deltas rather than full transcripts. Extract at context compaction (PreCompact) and session end (Stop/SessionEnd) using only the new transcript content since the last extraction. Write enriched TOML memory files for learnings not captured by real-time correction (UC-3).

**Starting state:** A session is active and a PreCompact or Stop hook fires. The transcript has accumulated since the last extraction. Engram tracks the byte offset of the last extraction point per session. If this is a new session (session ID changed), the offset is reset to 0.

**End state:** New enriched TOML memory files exist for learnings not already captured by UC-3. No duplicates of existing memories. Each learning classified as A/B/C using the same tier criteria as real-time capture. The extraction offset is updated for the next extraction in this session.

**Actor:** System (Go binary triggered by PreCompact and Stop/SessionEnd hooks).

**Key interactions:**

- **Trigger — PreCompact + Stop:** Fires on both events. PreCompact captures learnings before context window compression. Stop/SessionEnd captures end-of-session learnings regardless of how the session ends. Both triggers invoke the same incremental extraction pipeline.

- **Incremental extraction with offset tracking:** The `engram learn` subcommand accepts `--transcript-path` and `--session-id` flags. On each invocation:
  1. Read the learn offset from persistent storage (e.g., `<data-dir>/learn-offset.json`). Key by session ID.
  2. If session ID differs from the stored session, reset offset to 0 (new session detected).
  3. Read the transcript file from `--transcript-path` starting at the byte offset.
  4. If the delta is empty, skip extraction (no API call, no cost for idle periods).
  5. If the delta has content, preprocess it with the Strip operation (remove low-value content: tool results, base64/binary, repeated schemas) to reduce tokens sent to the LLM.
  6. Send the stripped delta (not full transcript) to the LLM for extraction.
  7. Update the offset to the current file end position.

- **LLM extraction with unified tier criteria:** A single API call (claude-haiku-4-5-20251001) receives the stripped transcript delta and produces a list of candidate learnings. Each candidate is classified using the same A/B/C tier criteria as UC-3's real-time classifier:

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

- **Deduplication against existing memories:** Before writing each candidate, check existing TOML files in the memories directory. Compare by keyword overlap. If a candidate substantially overlaps an existing memory (>50% keyword match), skip it. UC-3 mid-session captures take priority — session-end extraction never duplicates what was already captured.

- **Confidence tiers:** Session-extracted learnings are classified as A, B, or C using the same criteria as UC-3. Most will be tier C (contextual facts, discovered constraints), but the LLM may identify missed tier-A or tier-B signals. Anti-pattern generation follows the same tier gating: A always, B when generalizable, C never.

- **No graceful degradation:** If no API token is configured, emit a loud stderr error (`[engram] Error: session learning skipped — no API token configured`) and do not create any memory files. Never write degraded memories.

- **TOML file output:** Same format and directory as UC-3. Memory written to `<data-dir>/memories/<slug>.toml`. The `confidence` field reflects the classified tier (A, B, or C).

- **Creation visibility (deferred):** PreCompact and Stop/SessionEnd hooks have no output mechanism to show the user what was created. Instead, creation events are logged to a file (`<data-dir>/creation-log.jsonl`) with timestamp, title, tier, and file path. UC-2's SessionStart surfacing reports these at the start of the next session so the user sees what was learned. The log is cleared after successful reporting.

- **Idempotency:** If both PreCompact and Stop fire in the same session, each invocation extracts from its own transcript delta (determined by byte offset). Multiple PreCompact events in a long session each process only the new content since the previous PreCompact. Dedup handles any overlap across multiple extractions in the same session.

- **Session boundary handling:** When a new session starts (session ID changes), the learn offset resets to 0. This prevents loss of learnings at session boundaries — the next PreCompact or Stop in the new session will extract from the beginning of the new transcript.

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

## UC-16: Unified Memory Maintenance

**Description:** Diagnose actionable memories across the four effectiveness quadrants (working/leech/hidden gem/noise) and generate specific maintenance proposals. Proposals surface as structured output for the agent to present to the user. The user confirms or skips each proposal; the agent executes confirmed actions (edit TOML, delete file, broaden keywords).

**Starting state:** Memories exist with surfacing instrumentation (UC-2) and evaluation history (UC-15). UC-6 provides quadrant classification. At least some memories have 5+ evaluations.

**End state:** Each actionable memory has a specific, evidence-backed proposal. Confirmed proposals result in updated or removed TOML files. Skipped proposals are logged but not acted on.

**Actor:** Developer via `engram maintain --data-dir <path>` CLI command.

**Key interactions:**

- **Quadrant partitioning:** Reuse UC-6's effectiveness aggregation and matrix classification (REQ-35 median split, REQ-36 threshold flagging). No new classification logic — `maintain` consumes the same data `review` does.

- **Proposal generation per quadrant:**

  | Quadrant | Diagnosis | Proposal Type | LLM? |
  |----------|-----------|---------------|------|
  | **Working** | Staleness check: last_updated age, referenced code paths | "Still current" or "May be stale — review content" | No |
  | **Leech** | Root cause: content quality, wrong tier, keyword mismatch | Rewrite content, adjust tier, expand/narrow keywords | Yes (Haiku) |
  | **Hidden Gem** | Under-triggering: effective but rarely surfaced | Add keywords/concepts to broaden surfacing | Yes (Haiku) |
  | **Noise** | Low value: rarely surfaced, ineffective when surfaced | Remove with evidence (surfacing count, follow rate, age) | No |

- **Output format:** JSON array of proposals, each with:
  - `memory_path` — file path of the target memory
  - `quadrant` — working/leech/hidden_gem/noise
  - `diagnosis` — human-readable explanation of why this memory needs attention
  - `action` — proposed action type (review_staleness, rewrite, broaden_keywords, remove)
  - `details` — action-specific payload (new keywords, rewritten content, removal evidence)

- **LLM proposals (Haiku):** For leech and hidden gem quadrants, call claude-haiku-4-5-20251001 with:
  - The memory's current content (title, principle, anti_pattern, keywords, content)
  - The memory's effectiveness stats (surfaced count, follow rate, quadrant)
  - Instruction to propose specific fixes (not vague suggestions)
  - Output: JSON with proposed changes to specific TOML fields

- **No-data behavior:** If no memories have 5+ evaluations, output empty proposals array and exit 0. No error, no degraded output.

- **Fire-and-forget errors (ARCH-6):** LLM failures for individual proposals don't block other proposals. Failed proposals are omitted from output. Command always exits 0.

- **Pure Go, no CGO.**

---

## UC-17: Context Budget Management

**Description:** Track and cap total context injection across all engram hook points. Prioritize high-effectiveness memories within budget. Token estimation for all context output, configurable per-hook budget caps, and priority allocation by effectiveness × relevance.

**Starting state:** Engram injects context at multiple hook points (SessionStart, UserPromptSubmit, PreToolUse). Memory count is growing, and total context injection is unbounded.

**End state:** Each hook point has a configurable token budget cap. Memories are sorted by effectiveness × relevance and filled until budget is reached. Budget utilization is reported in `engram review` output. Warnings are emitted when a hook consistently hits its cap.

**Actor:** System (Go binary, extends existing surface logic with token counting and cutoff).

**Key interactions:**

- **Token estimation:** `len(text) / 4` as conservative estimator for English text with code snippets. No real tokenizer needed — soft caps, not hard limits.
- **Per-hook caps:** Configurable defaults: SessionStart 800 tokens, UserPromptSubmit 300 tokens, PreToolUse 200 tokens, PostToolUse 100 tokens, Stop audit 500 tokens.
- **Priority allocation:** Sort surfaced memories by effectiveness score × relevance score. Fill until budget reached, skip remainder.
- **Budget reporting:** `engram review` includes budget utilization per hook point.
- **Budget warnings:** When a hook hits its cap on >50% of invocations, warn in review output.
- **Pure Go, no CGO.**

**Dependencies:** UC-2 (surface), UC-6 (effectiveness)

---

## UC-18: PostToolUse Proactive Reminders

**Description:** After the model writes or edits tracked files, inject a targeted reminder about commonly-violated instructions relevant to that file type. Pattern-based trigger configuration maps file patterns to reminder sets sourced from the instruction registry.

**Starting state:** The model has just completed a Write or Edit tool call on a tracked file. Relevant instructions exist in memories, CLAUDE.md, or skills.

**End state:** A targeted reminder (≤100 tokens) is injected as a system reminder if relevant instructions match the file pattern. Effectiveness of reminders is tracked. Reminders are suppressed if the model already complied before the reminder.

**Actor:** System (PostToolUse hook script + Go binary for matching).

**Key interactions:**

- **New PostToolUse hook:** Registered in `hooks/hooks.json`. Fires after Write/Edit tool calls.
- **Pattern-based triggers:** Configuration maps file glob patterns to instruction sets (e.g., `*.go` → Go-specific conventions, skill files → pressure-test reminder).
- **Reminder sourcing:** Match against memories with anti-patterns, CLAUDE.md entries, and skill instructions.
- **Budget-capped:** Each reminder ≤100 tokens. Single targeted reminder per invocation, not a memory dump.
- **Suppression logic:** If transcript shows the model already performed the required action, suppress the reminder.
- **Effectiveness tracking:** Did the model comply after the reminder? Fed into evaluation pipeline.
- **Pure Go, no CGO.**

**Dependencies:** UC-17 (budget), UC-2 (surface infrastructure)

---

## UC-19: Stop Session Audit

**Description:** At session end, run a lightweight audit that checks whether high-priority instructions were followed during the session. Produces an audit report and feeds results into the effectiveness pipeline.

**Starting state:** A session is ending (Stop hook fires). Memories were surfaced during the session. The surfacing log and transcript are available.

**End state:** An audit report exists at `<data-dir>/audits/<timestamp>.json` with compliance checks for high-priority instructions. Results feed into the evaluate pipeline.

**Actor:** System (Go binary triggered by Stop hook, after learn and before context-update).

**Key interactions:**

- **Enhanced Stop hook:** Audit phase runs after `engram learn` and `engram evaluate`, before context-update.
- **Audit scope:** High-priority memories surfaced during the session + their outcomes. For skills invoked during the session, verify critical steps were performed.
- **LLM assessment:** Single Haiku call to assess compliance against the instruction set using the transcript.
- **Audit report:** Written to `<data-dir>/audits/<timestamp>.json` with per-instruction compliance status, evidence, and recommendations.
- **Integration with effectiveness:** Audit results feed into the evaluate pipeline as additional outcome signals.
- **No graceful degradation:** If no API token, emit stderr error and skip audit.
- **Pure Go, no CGO.**

**Dependencies:** UC-15 (evaluate infrastructure), UC-2 (surfacing log)

---

## UC-20: Memory Instruction Quality Audit

**Description:** Audit engram memory entries for quality problems and gaps. Examines memories only — cross-source deduplication (CLAUDE.md, rules, skills) moves to the surface pipeline in P4-full. S6 simplification of Phase A-1.

**Starting state:** Memory entries exist in the data directory. Some overlap, some are poorly framed, and some gaps exist where violations occur without corresponding memories.

**End state:** An `engram instruct audit` command reports duplicates among memories, quality diagnoses for low-effectiveness memories, refinement proposals, and gap analysis. Proposals are maintain-compatible.

**Actor:** Developer via `engram instruct audit` CLI command.

**Key interactions:**

- **Memory-only scanning:** Scan memory entries from `<data-dir>/memories/`. No CLAUDE.md, rules, or skill sources.
- **Memory deduplication:** Detect memories with >80% keyword overlap and report pairs.
- **Quality diagnosis (LLM):** For low-effectiveness memories (bottom 20%), diagnose root cause: too abstract, framing mismatch, missing trigger conditions, too narrow, too verbose.
- **Refinement proposals:** Generate rewrite proposals in maintain-compatible format.
- **Gap analysis:** Compare instruction anti-patterns against observed tool actions to find common violation patterns with no corresponding memory.
- **No graceful degradation:** If no API token, skip LLM diagnosis. Deduplication and gap analysis still run.
- **Pure Go, no CGO.**
- **Future:** Cross-source deduplication re-introduced as read-only check in surface pipeline (P4-full).

**Dependencies:** UC-6 (review), UC-16 (maintain), UC-17 (budget — needed to measure context cost)

---

## UC-21: Enforcement Escalation Ladder

**Description:** When maintain detects a leech memory (frequently surfaced, rarely followed), propose graduated escalation from advisory to blocking enforcement. Replaces the binary advisory→blocking jump with a measured ladder. Includes de-escalation when blocking causes harm.

**Starting state:** UC-16 maintain has identified leech memories. Some memories remain ineffective despite content rewrites.

**End state:** Each leech memory has an escalation level stored in its TOML file. Maintain proposals include escalation/de-escalation recommendations with predicted impact. User confirms each step.

**Actor:** Developer via `engram maintain` CLI command (extended).

**Key interactions:**

- **Escalation levels:** advisory → emphasized_advisory → reminder (PostToolUse injection). Engram's top level is `reminder`; beyond that, a graduation signal is emitted (UC-28).
- **Escalation proposals:** Each level change is a maintain proposal with rationale and predicted impact based on effectiveness data at current level.
- **De-escalation:** If an elevated level causes no improvement, propose reverting to a lower level.
- **Tracking:** Escalation level stored per memory in TOML. Effectiveness tracked per escalation level to measure impact of each step.
- **User confirmation:** Every escalation/de-escalation requires explicit user confirmation.
- **Graduation lifecycle (P6-full):** When a memory reaches `graduated`, the signal is written to `graduation-queue.jsonl` with a stable ID and persists until the user accepts or dismisses it. At SessionStart, pending graduation signals are surfaced in `additionalContext` with instructions for the LLM to ask the user about GitHub issue creation. `engram graduate accept --id <id>` creates a GitHub issue and records the entry accepted; `engram graduate dismiss --id <id>` records it dismissed. Quality metric = accepted / (accepted + dismissed).

**Dependencies:** UC-16 (maintain), UC-17 (budget), UC-18 (PostToolUse), UC-28 (signal queue)

---

## UC-22: Mechanical Instruction Extraction *(removed — Phase A-1/S1)*

**Status:** Removed. Engram's role is to diagnose and recommend — it does not generate enforcement mechanisms. Graduation signals (UC-28/Package 6) replace the automation proposal concept. The `internal/automate/` package and `engram automate` CLI command have been deleted.

**Dependencies:** UC-21 (escalation — identifies automation candidates)

---

## UC-23: Unified Instruction Registry

**Description:** Track effectiveness, frecency, and lifecycle state for all instruction sources (memories, CLAUDE.md, MEMORY.md, rules, skills) in a single bounded registry. Replace fragmented tracking stores (surfacing-log.jsonl, creation-log.jsonl, evaluations/*.jsonl, inline memory TOML stats) with one instruction-registry.jsonl file. Enable cross-source quadrant classification and merge operations that preserve violation history when deleting duplicates.

**Starting state:** Engram has 6 data stores with effectiveness tracked only for memories. CLAUDE.md entries, rules, and skills have no feedback loop.

**End state:** A single instruction-registry.jsonl tracks all registered instructions. Surfacing, evaluation, and learn pipelines write to the registry instead of fragmented stores. `engram review` classifies all registered instructions into quadrants. `engram registry merge` absorbs effectiveness history when deleting duplicates. Old stores (surfacing-log, creation-log, evaluations/) are deleted.

**Actor:** System (Go binary, triggered by hooks) + User (CLI commands for review, merge, register-source).

**Key interactions:**

- **Registration:** New memories auto-registered on creation via learn pipeline.
- **Surfacing tracking:** Each hook surfacing event updates the registry (increment surfaced_count, update last_surfaced) instead of appending to surfacing-log.jsonl.
- **Evaluation tracking:** Compliance evaluation updates registry counters (followed/contradicted/ignored) instead of writing per-session JSONL files.
- **Classification:** `engram review` reads the registry directly for pre-aggregated quadrant classification across all sources.
- **Merge:** `engram registry merge --source <id> --target <id>` absorbs effectiveness history into the target instruction's `absorbed` field, then deletes the source.
- **Backfill:** `engram registry init` migrates data from existing stores into the registry. After verification, old stores can be deleted.
- **No graceful degradation needed:** Registry operations are local file I/O, no API dependency.
- **Pure Go, no CGO.**
- **DI everywhere:** Registry interface in internal/, JSONL I/O wired at edges.

**Constraints:**
1. Bounded growth — one line per instruction, no unbounded logs
2. Backward compatibility — backfill migrates existing data with no loss
3. Fire-and-forget on failure — registry write failures don't crash hooks (ARCH-6)
4. Content-only memory TOMLs — after migration, memory TOMLs lose effectiveness fields
5. Concurrent-write safety — multiple hooks may update the registry

**Dependencies:** None (foundation UC). Depended on by: UC-4, UC-5, UC-7, UC-8, UC-9, UC-10.

**S3 simplification (Phase A-1):** Non-memory extractors (ClaudeMDExtractor, MemoryMDExtractor, RuleExtractor, SkillExtractor) relocated from `internal/registry/extract.go` to `internal/crossref/extract.go`. The registry package now handles only the memory source type. This sets up the correct package boundary for the cross-source scanner (P0c/UC-29). See ARCH-79.

---

## UC-4: Skill Generation

**Description:** Automatically promote memories to Claude Code skills (tier 2) when surfacing cost exceeds skill slot cost. A memory surfaced frequently enough would be cheaper as a skill that loads by context similarity rather than keyword matching on every prompt.

**Starting state:** The instruction registry (UC-23) tracks surfacing frequency and effectiveness for all memories. Some memories have high surfacing counts, indicating they load on many prompts via keyword matching.

**End state:** Memories that cross the promotion threshold are converted to skill files, registered with the Claude Code plugin, and the source memory is retired (merged into the skill's registry entry via UC-23 merge). The skill loads by context similarity instead of keyword matching.

**Actor:** Developer via `engram promote --to-skill` CLI command.

**Key interactions:**

- **Candidate detection:** Query the registry for memories with surfacing_count above the promotion threshold. Compare surfacing cost (loaded every prompt via keyword match) vs skill slot cost (loaded only when context-similar). Candidates are memories where surfacing cost > skill cost.
- **Skill file generation (LLM):** Generate a skill file from memory content — title becomes skill name, content/principle/anti_pattern become skill body, keywords/concepts inform the skill's triggering description.
- **Plugin registration:** Write skill file to the plugin's skills directory. Update plugin manifest if needed.
- **Source retirement:** Merge the memory's registry entry into the new skill's entry (preserving effectiveness history via UC-23 merge). Delete the source memory TOML.
- **User confirmation:** Present the proposed promotion (source memory, generated skill preview) before executing. Never auto-promote.
- **No graceful degradation:** If no API token, skip LLM generation. Candidate detection still works.
- **Pure Go, no CGO.**
- **DI everywhere.**

**Constraints:**
1. User confirms before any promotion — memories are user-curated artifacts
2. Effectiveness history preserved via registry merge (not lost on promotion)
3. Generated skill must be valid Claude Code skill format
4. Fire-and-forget on registry write failures (ARCH-6)

**Dependencies:** UC-23 (registry for effectiveness data and merge operations)

---

## UC-5: CLAUDE.md Management

**Description:** Propose additions and removals to CLAUDE.md (tier 1, always-loaded guidance) based on measured effectiveness. Promote skills that prove universally useful; demote narrowly-specific CLAUDE.md entries back to skills.

**Starting state:** The instruction registry (UC-23) tracks effectiveness for all instruction sources including CLAUDE.md entries and skills. Some skills have high effectiveness across contexts; some CLAUDE.md entries have low effectiveness (Leech quadrant).

**End state:** Promotion candidates (high-effectiveness skills) are proposed for addition to CLAUDE.md. Demotion candidates (low-effectiveness CLAUDE.md entries) are proposed for removal from CLAUDE.md and conversion to skills. User confirms before any changes.

**Actor:** Developer via `engram promote --to-claude-md` and `engram demote --to-skill` CLI commands.

**Key interactions:**

- **Promotion candidate detection:** Query the registry for skills in the Working quadrant with high surfacing frequency. These are universally useful and would benefit from always-loaded status.
- **CLAUDE.md entry generation (LLM):** Generate a concise CLAUDE.md entry from skill content. Must fit the existing CLAUDE.md style and structure.
- **Demotion candidate detection:** Query the registry for CLAUDE.md entries in the Leech quadrant — always loaded but rarely followed. These waste context budget.
- **Skill file generation from CLAUDE.md entry:** Convert the demoted CLAUDE.md entry into a skill file that loads by context similarity.
- **Registry merge:** On promotion, merge the skill's registry entry into the new CLAUDE.md entry's. On demotion, merge the CLAUDE.md entry's into the new skill's.
- **User confirmation:** Present proposed changes with evidence (effectiveness data, quadrant classification) before executing. Never auto-modify CLAUDE.md.
- **No graceful degradation:** If no API token, skip LLM generation. Candidate detection still works.
- **Pure Go, no CGO.**
- **DI everywhere.**

**Constraints:**
1. User confirms before any CLAUDE.md modification — CLAUDE.md is the highest-trust tier
2. Effectiveness history preserved via registry merge
3. CLAUDE.md edits are proposed as diffs, not applied blindly
4. Fire-and-forget on registry write failures (ARCH-6)

**Dependencies:** UC-23 (registry), UC-4 (skill generation for demotion path)

---

## UC-24: Proposal Application

**Description:** Apply the maintenance proposals generated by UC-16's `engram maintain` command. UC-16 generates JSON proposals for all four effectiveness quadrants (Working staleness updates, Leech rewrites, HiddenGem keyword broadening, Noise removal). This UC adds the `--apply` flag that executes selected proposals with user confirmation.

**Starting state:** `engram maintain` has generated a JSON array of proposals, each with a quadrant, action type, target memory path, and proposed change.

**End state:** Selected proposals are applied: memory TOML files are rewritten (Working/Leech/HiddenGem) or deleted (Noise). Registry entries are updated to reflect changes. All modifications are confirmed by the user before execution.

**Actor:** Developer via `engram maintain --apply` CLI command.

**Key interactions:**

- **Proposal review:** Display proposals grouped by quadrant with evidence (effectiveness score, surfacing count, proposed change preview). User selects which proposals to apply.
- **Working — staleness update:** Rewrite memory content to reflect current practices. LLM generates updated content; user confirms the diff.
- **Leech — content rewrite:** Rewrite memory to improve follow-through. Root cause diagnosis informs the rewrite (content quality → rewrite, wrong keywords → adjust, enforcement gap → escalate to UC-21).
- **HiddenGem — keyword broadening:** Add keywords/concepts to increase surfacing coverage. LLM suggests additional keywords based on contexts where the memory would have been relevant.
- **Noise — removal:** Delete the memory TOML file. Registry entry is removed. User confirms with evidence of low utility.
- **Registry update:** After each applied proposal, update the registry entry (content_hash for rewrites, remove for deletions).
- **User confirmation per action:** Each proposal requires explicit confirmation. Batch confirmation (`--yes`) available but not default.
- **No graceful degradation:** If no API token, skip LLM-dependent proposals (Working/Leech/HiddenGem rewrites). Noise removal still works (deterministic).
- **Pure Go, no CGO.**
- **DI everywhere.**

**Constraints:**
1. User confirms each proposal — memories are user-curated artifacts (MEMORY.md rule: never delete directly)
2. Content hash updated in registry after rewrites
3. Noise removal deletes the memory TOML file directly — user confirmation is the safety gate
4. Fire-and-forget on registry write failures (ARCH-6)

**Dependencies:** UC-16 (maintain proposals), UC-23 (registry)

---

## UC-25: Evaluate Strip Preprocessing

**Description:** Apply the same content stripping (removal of tool results, base64 data, truncated blocks) to the evaluate pipeline that learn and context-update already use. This reduces cost (~80-90% content removal) and improves signal quality for LLM evaluation.

**Starting state:** The evaluate command reads the full transcript via stdin without stripping noisy content. The `sessionctx.Strip` function exists and is used by learn (incremental) and context-update pipelines.

**End state:** The evaluate pipeline applies `sessionctx.Strip` to the transcript before sending it to the LLM for outcome evaluation. Evaluation quality improves (LLM evaluates conversation, not tool noise) and cost decreases.

**Actor:** System (Go binary triggered by hooks).

**Key interactions:**

- **Strip injection:** Add `sessionctx.Strip` call in the evaluate CLI path, after reading transcript from stdin and before passing to the Evaluator.
- **DI pattern:** Strip function injected as a dependency (consistent with learn pipeline's `StripFunc` pattern), not hardcoded.
- **Backward compatible:** If Strip produces empty output (edge case: transcript is all tool results), evaluation is skipped gracefully.
- **No graceful degradation needed:** Strip is local computation, no API dependency.
- **Pure Go, no CGO.**
- **DI everywhere.**

**Constraints:**
1. Same Strip function as learn and context-update — no divergent implementations
2. Empty post-strip transcript skips evaluation (no empty LLM calls)
3. Fire-and-forget consistent with evaluate pipeline's existing error handling

**Dependencies:** None (reuses existing Strip function)

---

## UC-26: First-Class Non-Memory Instruction Sources

**Description:** Make rules, skills, CLAUDE.md entries, and MEMORY.md entries full participants in the feedback loop. Currently these sources are registered in the instruction registry but never surfacing-tracked or evaluated — only memories get measured. This UC closes that gap: auto-register all sources at session start, track implicit surfacing for always-loaded sources, evaluate all registered instructions at session end, and prune stale entries when source files disappear.

**Starting state:** The instruction registry (UC-23) has extractors for all 5 source types and can store entries for any of them. But the hooks only call `RecordSurfacing` and `RecordEvaluation` for memory sources. Non-memory entries sit in the registry with zero surfacing counts and zero evaluation data, making quadrant classification meaningless for them.

**End state:** Every session automatically registers all discoverable instruction sources, records implicit surfacing for always-loaded sources, evaluates all surfaced instructions (not just memories) at session end, and removes registry entries whose source files no longer exist. `engram review` shows meaningful quadrant classifications for all instruction types.

**Actor:** System (Go binary triggered by SessionStart and Stop hooks) + Developer (CLI commands).

**Key interactions:**

- **Auto-registration at SessionStart:** On each session start, scan all known instruction source locations:
  - CLAUDE.md files: project CLAUDE.md, user global CLAUDE.md (paths from environment or convention)
  - MEMORY.md: `~/.claude/projects/<project>/memory/MEMORY.md`
  - Rules: all files in `.claude/rules/`
  - Skills: all skill files in the plugin's skills directory
  - Memories: already registered by learn pipeline (UC-1/UC-3), no change needed

  For each discovered source, extract entries using existing extractors (UC-23). Register new entries. Update content hash for existing entries if source content changed. **Remove registry entries whose source file no longer exists** (stale pruning).

- **Implicit surfacing for always-loaded sources:** Claude Code always loads claude-md, memory-md, and rule sources into every session. Engram doesn't control this loading, so it can't observe individual surfacing events. Instead, at SessionStart, increment `surfaced_count` and update `last_surfaced` for all always-loaded entries. This reflects the truth: they are surfaced on every session.

- **Skills as always-surfaced:** Claude Code loads skills by context similarity — engram cannot reliably detect when a specific skill is active. Rather than guess, treat all registered skills as always-surfaced (same as claude-md/memory-md/rule). This overcounts surfacing but produces valid effectiveness ratios. Skills get binary Working/Leech classification. Add `"skill"` and `"rule"` to `alwaysLoadedSources` in classify.go.

- **Evaluate all surfaced sources:** Extend the Stop hook's evaluation step (UC-15) to evaluate all instructions that were surfaced during the session, not just memories. The evaluator already takes a list of surfaced instruction IDs and judges each against the transcript. The change is in what gets surfaced: always-loaded sources are now in the surfacing log, so they appear in the evaluation input.

- **Stale entry pruning:** During auto-registration, build the set of all currently-discoverable source IDs. Any registry entry whose source type is non-memory and whose ID is not in the discovered set gets removed. This handles deleted rules, removed skills, and CLAUDE.md entries that were edited out.

- **No graceful degradation needed:** Auto-registration and surfacing tracking are local file I/O. Evaluation requires an API token (same as UC-15) — if no token, evaluation is skipped for all sources (existing behavior).

- **Pure Go, no CGO.**
- **DI everywhere.**

**Constraints:**
1. Always-loaded sources get implicit surfacing (one increment per session) — no per-prompt tracking
2. Skills treated as always-surfaced — binary Working/Leech classification only
3. Rules added to `alwaysLoadedSources` alongside claude-md and memory-md
4. Stale pruning only applies to non-memory sources — memory pruning is handled by UC-16 Noise removal
5. Auto-registration is idempotent — running twice produces the same registry state
6. Fire-and-forget on registry write failures (ARCH-6 contract)
7. Content hash updates detect when source content changes (e.g., CLAUDE.md edited)

**Dependencies:** UC-23 (registry infrastructure, extractors), UC-15 (evaluation pipeline)

---

---

## UC-27: Global Binary Installation

**Description:** Make the engram binary accessible on the user's PATH so manual operations (maintain, review, promote, demote, registry) work outside of Claude Code sessions. The SessionStart hook already builds the binary — extend it to create a symlink in `~/.local/bin/` so the binary is discoverable.

**Actor:** SessionStart hook (automatic), User (manual invocation after symlink exists)

**Starting state:** The engram binary exists at `~/.claude/engram/bin/engram` after the SessionStart build step. The user has no way to run engram commands without knowing this internal path.

**End state:** A symlink at `~/.local/bin/engram` points to `~/.claude/engram/bin/engram`. The user can run `engram review`, `engram maintain`, etc. from any terminal.

**Key interactions:**
1. SessionStart hook builds binary (existing behavior)
2. After successful build, check if `~/.local/bin/engram` symlink exists and points to the right target
3. If missing or stale: create `~/.local/bin/` if needed, then create symlink
4. If `~/.local/bin/engram` exists and is NOT a symlink to our binary: log warning, don't clobber

**Constraints:**
1. Idempotent — if symlink already correct, skip silently
2. No clobber — never overwrite a non-engram binary at the target path
3. Fire-and-forget — symlink creation failure doesn't block session start (ARCH-6)
4. Create `~/.local/bin/` directory if it doesn't exist
5. Target directory is `~/.local/bin/` (XDG standard)

**Dependencies:** None (extends existing SessionStart build step)

---

## UC-28: Automatic Maintenance and Promotion Triggers

**Description:** Automatically detect when memories need maintenance (rewrite, removal, keyword broadening) or graduation (memory→skill, or skill/CLAUDE.md needing level change), queue those signals, and surface them at session start so the conversation model can interview the user and apply changes. Graduation signals carry a human-readable recommendation; engram diagnoses and recommends — the user decides what to do. The model handles creative work (rewriting, keyword suggestions, skill generation); engram handles atomic I/O (file writes, registry updates). No CLI interaction required from the user.

**Starting state:** learn/evaluate run automatically via hooks; maintain/promote/demote require manual CLI invocation outside Claude Code.

**End state:** Stop hook detects actionable signals and queues them. SessionStart surfaces signals with memory details so the conversation model can present them naturally ("I see 2 memories that aren't working well..."), discuss with the user, generate rewrites/suggestions, and apply changes via `engram apply-proposal` — all within Claude Code.

**Actor:** System (Stop hook for detection, SessionStart hook for surfacing) + Conversation model (interview + creative work) + engram CLI (atomic apply).

**Key interactions:**

- **Signal detection (Stop hook):** After evaluate in Stop hook, run `engram signal-detect` — local-only quadrant classification + promotion threshold checks, no LLM calls. Reuses `review.Classify()` for maintenance signals and `Promoter.Candidates()` / `ClaudeMDPromoter.{Promotion,Demotion}Candidates()` for graduation signals. Promotion and demotion candidates both emit `graduation` signal kind with recommendation text.

- **Proposal queue:** Detected signals written to `<data-dir>/proposal-queue.jsonl` (append-safe, dedup, prune stale). Each line: `{type, source_id, signal, quadrant, summary, detected_at}`. Pruning removes entries >30 days old, entries for deleted memories, and entries where quadrant is no longer actionable.

- **SessionStart surfacing:** `engram signal-surface` reads queue + memory content for each signal. Outputs detailed model-facing context with signal metadata, memory title/content/stats, quadrant rationale, and action instructions (engram CLI commands to execute). Goes into `additionalContext`.

- **Model-driven interview:** Conversation model reads the surfaced signals, presents proposals to the user, user confirms/rejects inline. No CLI interaction — model handles the conversation.

- **Atomic application:** Model applies confirmed changes via `engram apply-proposal` — actions: remove (delete TOML + registry), rewrite (update TOML fields + registry hash), broaden (append keywords), escalate (update level). Atomic file writes via temp+rename.

- **Promotion with external content:** `engram promote --content '<skill>' --yes` — model generates skill/CLAUDE.md content, passes to engram which skips LLM generation and confirmation. Registry merge + file write still happen normally.

- **Queue cleanup:** Applied signals are cleared from queue automatically.

**Constraints:**
1. No LLM calls in detection — local-only classification
2. All creative work done by conversation model, not engram
3. User confirms all destructive actions via conversation
4. Fire-and-forget on errors (ARCH-6)
5. DI everywhere — all I/O through injected interfaces

**Dependencies:** UC-15 (evaluate), UC-16 (maintain — reuses quadrant classification), UC-4/UC-5 (promote/demote — reuses candidate detection), UC-23 (registry)

---

Deferred UCs (UC-7 through UC-13, excluding UC-6) proposal-generation scope consolidated into UC-16; proposal-application scope consolidated into UC-24. Issue #59 (BM25) already implemented. Archives in issue #18.

---

## UC-P1-1: Cross-Source Contradiction Detection

**Description:** At surface time, detect when two or more memories in the top-N selection contradict each other (e.g., one says "always use X", another says "never use X"). Suppress the lower-ranked contradicting memory and emit a `KindContradiction` signal so the user can review and reconcile.

**Actor:** Surface pipeline (automatic, read-only)

**Starting state:** Top-N memories have been ranked by frecency. No contradiction check exists.

**End state:** Contradicting memory pairs are identified via a two-pass heuristic + optional LLM classifier. The lower-ranked memory of each pair is suppressed from the surfaced output. A `KindContradiction` signal is emitted for each suppressed memory so it appears in `engram review`.

**Key interactions:**
1. Post-ranking: `Detector.Check(ctx, topN)` runs after frecency sort and before output formatting.
2. Pass 1 — keyword heuristic: For each pair (A, B), concatenate principle+title+content. Check if they share subject tokens and contain opposing verb patterns (use/avoid, always/never, do/don't, is/isn't, enable/disable). Flag as candidate contradiction if heuristic fires.
3. Pass 2 — BM25 similarity: Score A against B's text. If BM25 score > threshold and heuristic flagged the pair, mark high-confidence. If only BM25 is high (no heuristic), mark as borderline.
4. LLM classifier fallback (max 3 calls per surface event): For borderline pairs, call injected `Classifier.Classify(ctx, a, b)` to confirm. High-confidence pairs from step 3 skip LLM.
5. For each confirmed contradicting pair: suppress the lower-ranked (later in sorted slice) memory from output.
6. Emit `KindContradiction` signal for each suppressed memory into the proposal queue.

**Constraints:**
1. Read-only — no memory writes during detection
2. Max 3 LLM calls per surface event (budget enforced by Detector)
3. DI everywhere — LLM classifier is an injected interface
4. Fire-and-forget — if detector errors, proceed without suppression
5. Only runs on top-N selection (post-ranking), not all memories

**Dependencies:** UC-2 (surfacing), UC-28 (signal queue for KindContradiction)

---

## UC-33: Merge-on-Write

**Description:** When the learn pipeline encounters a candidate learning with >50% keyword overlap with an existing memory, merge the two into a single stronger memory instead of discarding the candidate. Use LLM (Haiku) to combine principle fields where possible; fall back to keyword/concept union + longer principle text when no LLM is available.

**Starting state:** The learn pipeline has extracted candidate learnings. At least one candidate has >50% keyword overlap with an existing memory.

**End state:** The overlapping existing memory is updated in place with a merged principle (and union keywords/concepts). The merge is recorded in the registry's `Absorbed` field on the existing memory's `InstructionEntry`. The candidate is not written as a new memory. Effectiveness counters from the existing memory are preserved.

**Actor:** System (learn pipeline at session end/pre-compact).

**Key interactions:**

- **Merge trigger:** In the deduplication stage, instead of discarding candidates with >50% keyword overlap, flag them as merge candidates and pair them with the matching existing memory.

- **LLM-assisted merge (primary path):** Call Haiku with the existing and candidate principles to produce a single combined principle that is stronger and more specific. If the LLM returns a non-empty principle, use it; update the existing memory file's `principle` field, union keywords and concepts, update `updated_at`.

- **Fallback merge (no LLM / LLM error):** Take the longer of the two principle texts. Union keywords and concepts. Update the existing memory file in place.

- **Registry `Absorbed` record:** After a successful merge, record an `AbsorbedRecord` on the existing memory's `InstructionEntry` with: `from` (candidate title), `content_hash` (candidate keywords hash), `surfaced_count: 0`, and `merged_at` timestamp.

- **Effectiveness preservation:** The existing memory's surfacing count, evaluation counters, and enforcement level are not modified. Only `principle`, `keywords`, `concepts`, and `updated_at` change in the TOML file.

- **No graceful degradation for write failures:** If the existing memory TOML cannot be updated, return an error (do not silently skip).

**Dependencies:** UC-1 (Session Learning), UC-23 (Registry)


---

## UC-32: Memory Graph with Spreading Activation

**Description:** Build a typed, weighted graph of links between memory entries at learn time, surface time, and evaluate time. Use the graph to amplify frecency scores via spreading activation and to surface short cluster notes alongside related memories.

**Actor:** Learn pipeline (concept_overlap + content_similarity at registration), surface pipeline (co_surfacing updates + spreading activation + cluster notes), evaluate pipeline (evaluation_correlation updates), maintenance (pruning)

**Starting state:** Memory entries are scored independently. No graph relationship between memories exists.

**End state:** Memory entries carry `Links []Link` (Target/Weight/Basis/CoSurfacingCount). Frecency scoring includes spreading activation: `total = base + 0.3 × Σ(linked_base × weight)`. Surfaced output includes cluster notes for top-2 linked memories per memory. Dead links (weight < 0.1, CoSurfacingCount ≥ 10) are pruned.

**Key interactions:**
1. **Learn time:** When a memory entry is registered/updated, `graph.Builder` computes concept_overlap (Jaccard ≥ 0.15) and content_similarity (BM25 ≥ 0.05) links to all existing entries. Links stored via `Registry.UpdateLinks`.
2. **Surface time (co_surfacing):** After top-N selection, for each co-surfaced pair, co_surfacing link weight incremented (+0.1, cap 1.0), CoSurfacingCount incremented. Bidirectional. Fire-and-forget on error.
3. **Surface time (spreading activation):** After co_surfacing updates, apply spreading activation: final score = base + 0.3 × Σ(linked_base × weight). Re-rank with activated scores.
4. **Surface time (cluster notes):** For each memory in final output, include up to 2 cluster notes (title of top-2 linked memories by weight, ~20 tokens each).
5. **Evaluate time:** For each pair of memories both receiving "followed" outcome in the same run, evaluation_correlation link weight incremented (+0.05, cap 1.0). Bidirectional.
6. **Pruning:** `graph.Prune` removes links with weight < 0.1 AND CoSurfacingCount ≥ 10. Called at maintenance time.

**Constraints:**
1. DI everywhere — link reads/writes through injected interfaces in surface/evaluate/learn packages
2. `internal/graph/` contains pure logic only (no I/O, no `os.*`)
3. Spreading activation is 1-hop only (no recursive traversal)
4. Max 2 cluster notes per surfaced memory
5. Fire-and-forget on co_surfacing and evaluation_correlation update errors (ARCH-6)

**Dependencies:** UC-23 (registry, UpdateLinks), UC-2 (surface pipeline), UC-15 (evaluate pipeline), UC-1 (learn pipeline)
