# Requirements

Requirements and design items derived from UC-3 (Remember & Correct) and UC-2 (Hook-Time Surfacing & Enforcement).

---

## REQ-1: Fast-path keyword detection + unified LLM classifier

When a user message is submitted (UserPromptSubmit hook), the system uses a two-stage detection:

**Stage 1: Fast-path (no LLM).** Three keywords trigger immediate tier-A classification:
- `remember` — explicit instruction to remember
- `always` — standing instruction for future behavior
- `never` — standing prohibition for future behavior

Fast-path messages skip the classifier LLM call and proceed directly to enrichment.

**Stage 2: Unified LLM classifier (everything else).** For messages without fast-path keywords, a single API call to claude-haiku-4-5-20251001 classifies the signal tier (A/B/C/null) and extracts structured memory fields in one call. Returns JSON with `tier` field: `"A"` (explicit instruction), `"B"` (teachable correction), `"C"` (contextual fact), or `null` (not a signal).

Null classification → skip entirely, no file created.

- Traces to: UC-3 (detection + classification)
- AC: (1) Fast-path keywords are checked first; if present, tier-A classification is immediate. (2) Other messages invoke the unified classifier LLM. (3) Classifier returns `tier` field (A/B/C/null). (4) Null tier → no file created, no system reminder. (5) A/B/C tier → proceed to enrichment and write.
- Verification: deterministic (keyword presence check, JSON schema validation)

---

## REQ-2: Unified LLM call classifies and enriches

For messages without fast-path keywords, a single API call to claude-haiku-4-5-20251001 receives the user message plus recent transcript context (~2000 tokens). The call returns JSON with both classification and structured memory fields:

**Response fields:**
- `tier` — `"A"`, `"B"`, `"C"`, or `null` (required)
- `title` — 5-10 word summary
- `content` — Full message verbatim
- `observation_type` — Category label
- `concepts` — Key concept tags
- `keywords` — Searchable keywords
- `principle` — Positive rule to follow
- `anti_pattern` — Anti-pattern value (tier-gated: required for A, optional for B, empty for C)
- `rationale` — Why this matters
- `filename_summary` — 3-5 words for slug generation

- Traces to: UC-3 (unified LLM classification + enrichment)
- AC: (1) API call uses claude-haiku-4-5-20251001. (2) Input includes user message + recent transcript context (REQ-X, see below). (3) Response is parsed as JSON with all required fields. (4) `tier` field drives downstream behavior (null → skip, A/B/C → write). (5) Invalid or unparseable responses return an error.
- Verification: deterministic (JSON schema validation of LLM response)

---

## REQ-X: Transcript context reading for unified classifier

The unified LLM classifier (REQ-2) receives recent session context to improve classification accuracy. When the UserPromptSubmit hook invokes `engram correct`, the hook JSON input includes a `transcript_path` field pointing to the current session transcript. The Go binary reads the last ~2000 tokens from this file and includes them in the classifier LLM call.

- Traces to: UC-3 (unified LLM classifier context)
- AC: (1) Hook input JSON includes `transcript_path` field (available in UserPromptSubmit hook). (2) Go binary reads `transcript_path` file. (3) Extracts recent portion (~2000 tokens, or entire file if shorter). (4) Includes recent context in the classifier LLM prompt. (5) Missing or unreadable transcript_path → proceed without context (non-fatal, context is advisory).
- Verification: deterministic (file read, token counting)

---

## REQ-3: Enriched memory written as TOML file

The enriched memory is written to `<data-dir>/memories/<slug>.toml` where slug is the slugified filename summary (3-5 hyphenated lowercase words). The TOML file contains all structured fields plus confidence tier and timestamps.

- Traces to: UC-3 (TOML file output)
- AC: (1) File is written to the memories subdirectory. (2) Filename is 3-5 hyphenated words, lowercase, `.toml` extension. (3) TOML contains: title, content, observation_type, concepts (array), keywords (array), principle, anti_pattern, rationale, confidence, created_at (RFC 3339), updated_at (RFC 3339). (4) File is valid TOML, human-readable, and hand-editable.
- Verification: deterministic (file exists, TOML parses, fields present)

---

## REQ-4: System reminder feedback on memory creation — user-visible

After a memory file is created, the system outputs creation feedback that is visible to the user via the hook's `systemMessage` field. The feedback confirms: the tier classification, the memory title, and the file path.

When surface matches also exist in the same hook invocation, creation feedback and surface summary are both included in `systemMessage` so the user sees both. Creation feedback is never buried in `additionalContext` (model-only context).

- Traces to: UC-3 (feedback, user visibility)
- AC: (1) Memory creation always outputs feedback. (2) Feedback goes into the hook's `systemMessage` field for user visibility. (3) When surface matches co-occur, both creation and surface summary appear in `systemMessage`. (4) Feedback includes memory tier (A/B/C), title, and file path. (5) Format uses `[engram]` prefix. (6) Null classification → no output (no file created).
- Verification: deterministic (hook output field check, user visibility confirmation)

---

## REQ-6: Go binary with `correct` subcommand

The system is implemented as a Go binary (`engram`) with a `correct` subcommand: `engram correct --message <text> --data-dir <path>`. Pure Go, no CGO.

- Traces to: UC-3 (Go binary CLI)
- AC: (1) Binary compiles with `CGO_ENABLED=0`. (2) `engram correct --message <text> --data-dir <path>` runs the detection → enrichment → write → feedback pipeline. (3) Exit 0 always — errors logged to stderr, never propagated as exit codes.
- Verification: deterministic (build succeeds, subcommand runs)

---

## REQ-7: Unified A/B/C confidence tier criteria

All memories (real-time and post-session) use identical tier criteria:

| Tier | What | Anti-pattern | Example |
|------|------|-------------|---------|
| **A** | Explicit instruction | Always generated | "Remember: always use fish", "from now on use targ" |
| **B** | Teachable correction | When generalizable (LLM decides) | "No, use targ — don't run raw go test" |
| **C** | Contextual fact | Never generated | "The port is 3001", architectural decision |

Fast-path keywords (`remember`, `always`, `never`) produce tier A immediately. The unified classifier (REQ-2) assigns A/B/C based on signal content. UC-1 extraction uses the same criteria.

Anti-pattern generation is tier-gated: A always generates `anti_pattern`, B generates it when the correction implies a generalizable pattern (LLM decides), C never generates `anti_pattern`.

- Traces to: UC-3 (confidence tiers, anti-pattern gating), UC-1 (tier assignment, anti-pattern gating)
- AC: (1) Fast-path keywords → tier A. (2) Classifier output `tier` field determines memory tier. (3) UC-1 extraction classifies using same criteria. (4) `anti_pattern` field populated only for A and (sometimes) B, never for C. (5) Tier is written to the TOML `confidence` field.
- Verification: deterministic (tier assignment, anti-pattern presence matches tier)

---

## DES-1: Correction feedback reminder format

When a memory is created, the agent sees:

```
<system-reminder source="engram">
[engram] Memory captured (tier A).
  Created: "<title>"
  Type: <observation_type>
  File: <file_path>
</system-reminder>
```

Format rules:
- Header: `[engram] Memory captured (tier <tier>).` where tier is A/B/C
- Action: `Created:` with memory title in quotes
- Type: the observation_type field
- File: relative path to the TOML file
- Concise — appears in the same hook response

- Traces to: UC-3 (feedback, tier classification)

---

## DES-3: Hook wiring — UserPromptSubmit

The UserPromptSubmit hook is registered in `hooks/hooks.json` and invokes `hooks/user-prompt-submit.sh`. The hook reads the user prompt and transcript path from stdin JSON, and invokes two independent operations:

1. **Correction (UC-3):** `engram correct --message "$USER_MESSAGE" --data-dir "$ENGRAM_DATA"`
2. **Surfacing (UC-2):** `engram surface --mode prompt --message "$USER_MESSAGE" --data-dir "$ENGRAM_DATA" --format json`

The `transcript_path` field is available in the UserPromptSubmit hook JSON input and is read by the Go binary for context (REQ-X). The hook also self-builds the binary if missing (`go build -o "$ENGRAM_BIN" ./cmd/engram/`).

Token retrieval is platform-aware:
- **macOS:** Attempt to read OAuth token from Claude Code Keychain via `security find-generic-password`. On failure, fall back to `ENGRAM_API_TOKEN` env var.
- **Non-macOS (Linux, etc.):** Use `ENGRAM_API_TOKEN` env var directly.

The hook exports `ENGRAM_API_TOKEN` from whichever source succeeds.

**Hook output JSON:**
- If surface matches exist: `{systemMessage: (<surface_summary> + "\n" + <creation_output>), additionalContext: <surface_context>}`
  - User sees both creation feedback AND surface summary in one message.
  - Surfaced memory details go to model context for analysis.
- If only creation exists (no surface matches): `{systemMessage: <creation_output>, additionalContext: ""}`
- If neither creation nor surface matches: `{}` (empty output)

Creation feedback must always appear in `systemMessage` (user-visible), never relegated to `additionalContext`.

- Traces to: UC-3 (hook wiring, transcript context reading, user-visible creation feedback)

---

## REQ-8: Binary build mechanism

The UserPromptSubmit hook self-builds the binary on first invocation if missing. Build produces `~/.claude/engram/bin/engram`.

Build command: `go build -o "$ENGRAM_BIN" ./cmd/engram/`

- Build requires Go toolchain on `PATH`.
- `bin/` is gitignored (binary not committed).
- Go's build cache makes repeated builds fast (sub-second no-op if source unchanged).
- Build failure logs to stderr but does not fail the hook (exit 0 always).
- Traces to: UC-3 (binary must exist for UserPromptSubmit hook to work)
- AC: (1) UserPromptSubmit hook builds binary if missing. (2) Binary is produced at `~/.claude/engram/bin/engram`. (3) Build failure does not break Claude Code session. (4) `bin/` is in `.gitignore`.

---

## DES-4: Plugin installation and setup UX

Installation steps for a user:

1. **Prerequisites:** Go toolchain installed and on `PATH`.
2. **Clone:** `git clone` the engram repo.
3. **Register:** `claude --plugin-dir /path/to/engram` (development mode).
4. **Verify:** Start a Claude Code session. The `SessionStart` hook auto-builds the binary. Then send a message like "remember to always use targ" — a memory TOML file should be created.

No manual build step required — the `SessionStart` hook handles it (REQ-8). README documents these steps.

- Traces to: UC-3 (user must be able to install and exercise the plugin)

---

# UC-2 Requirements and Design

---

## REQ-9: SessionStart surfacing — top 20 by frecency

When a session starts (SessionStart hook), the system reads all memory TOML files from the data directory, ranks them by frecency activation score (ACT-R model), and surfaces the top 20 as a system reminder.

- Traces to: UC-2 (SessionStart surfacing)
- AC: (1) All `.toml` files in `<data-dir>/memories/` are read and parsed. (2) Frecency activation score computed per memory using ACT-R formula (REQ-46). (3) Sorted by activation score descending. (4) Top 20 are included in the system reminder. (5) If fewer than 20 memories exist, all are surfaced. (6) If no memories exist, no reminder is emitted (empty stdout). (7) Memories with no surfacing history fall back to updated_at for recency component.
- Verification: deterministic (file listing, frecency scoring, sort, count)

---

## REQ-10: UserPromptSubmit surfacing — BM25 + frecency ranking

When a user message is submitted (UserPromptSubmit hook), the system ranks all memories using BM25 scoring, then re-ranks the top candidates by frecency activation score, and surfaces the top 10 results as a system reminder.

- Traces to: UC-2 (UserPromptSubmit surfacing)
- AC: (1) BM25 index is built per call from concatenated text fields (title, content, principle, keywords, concepts). (2) User message is scored against each memory. (3) BM25 top candidates are selected. (4) Candidates are re-ranked by combined score: BM25 relevance × frecency activation (REQ-46). (5) Top 10 ranked results (or fewer if fewer than 10 memories exist) are surfaced with title and file path. (6) If no memories exist or all scores are zero, no surfacing reminder is emitted. (7) Surfacing runs alongside UC-3 correction detection — both outputs are concatenated.
- Verification: probabilistic (BM25 + frecency scoring, ranking changes with message input and memory usage history)

---

## REQ-11: PreToolUse advisory surfacing — BM25 + frecency ranking on anti-pattern candidates

When a tool call is about to execute (PreToolUse hook), the system ranks anti-pattern memories using BM25 scoring against the tool name and input, re-ranks by frecency, and surfaces the top 5 results as an advisory system reminder.

- Traces to: UC-2 (PreToolUse advisory surfacing, tier-aware anti-pattern filtering)
- AC: (1) Only memories with a non-empty `anti_pattern` field are candidates (tier A always, tier B sometimes, tier C never per REQ-7). (2) BM25 index is built from candidate memories' text fields (title, principle, anti_pattern, keywords). (3) Tool name and input are concatenated and scored against each candidate. (4) BM25 top candidates are re-ranked by combined score: BM25 relevance × frecency activation (REQ-46). (5) Top 5 ranked results (or fewer if fewer candidates exist) are surfaced as a system reminder with title, principle, and file path. (6) If no candidates exist or all scores are zero, no output is emitted (zero overhead — no LLM call, no advisory). (7) The agent has full session context to exercise judgment on whether the tool call violates the memory's principle.
- Verification: probabilistic (BM25 + frecency scoring, ranking changes with tool input and usage history)

---

## REQ-14: Go binary `surface` subcommand

The system adds a `surface` subcommand to the engram binary: `engram surface --mode <session-start|prompt|tool> --data-dir <path> [--format json]`. Mode-specific flags: `--message <text>` for prompt mode, `--tool-name <name> --tool-input <json>` for tool mode. The `--format json` flag outputs a JSON object with `summary` (brief human-readable message) and `context` (full system-reminder XML) instead of raw XML.

- Traces to: UC-2 (CLI entry point)
- AC: (1) `engram surface --mode session-start --data-dir <path>` runs SessionStart surfacing. (2) `engram surface --mode prompt --message <text> --data-dir <path>` runs UserPromptSubmit surfacing. (3) `engram surface --mode tool --tool-name <name> --tool-input <json> --data-dir <path>` runs PreToolUse enforcement. (4) Exit 0 always — errors logged to stderr. (5) `--format json` outputs `{"summary": "...", "context": "..."}` instead of raw XML. (6) No matches produce empty stdout regardless of format.
- Verification: deterministic (subcommand runs, correct mode dispatch)

---

## DES-5: SessionStart surfacing reminder format

When memories are surfaced at session start, the output includes two sections:

1. **Creation report (if creation log exists):** Before surfacing recency-based memories, the system reports memories created in prior sessions:

```
<system-reminder source="engram">
[engram] Created N memories since last session:
  - "<title1>" [A] (file1.toml)
  - "<title2>" [B] (file2.toml)
</system-reminder>
```

2. **Recency surfacing:** Then the top 20 by recency:

```
<system-reminder source="engram">
[engram] Loaded M memories.
  - "<title3>" (file3.toml)
  - "<title4>" (file4.toml)
  ...
</system-reminder>
```

Format rules:
- Creation report: "Created N memories since last session:" with title, tier, file path for each
- Recency report: "Loaded M memories." with title and file path for each
- Ordered by recency (most recent first) for recency section
- If no creation log exists, skip creation report
- If no memories exist for recency section, emit only creation report (if present)
- With `--format json`: wrap both sections in `{"summary": "[engram] ...", "context": "<system-reminder>..."}`. Hook script reshapes into `{systemMessage, additionalContext}` for Claude Code visibility.

- Traces to: UC-2 (SessionStart feedback, creation visibility)

---

## DES-6: UserPromptSubmit surfacing reminder format

When memories match the user's message, the agent sees:

```
<system-reminder source="engram">
[engram] Relevant memories:
  - "<title1>" (file1.toml) [matched: keyword1, keyword2]
  - "<title2>" (file2.toml) [matched: concept1]
</system-reminder>
```

Format rules:
- Header: `[engram] Relevant memories:`
- Each memory: title, file path, matched keywords/concepts in brackets
- If no matches, no output
- With `--format json`: wrapped in `{"summary": "[engram] N relevant memories.", "context": "<system-reminder>..."}`. Hook script reshapes into `{systemMessage, additionalContext}`, merging any UC-3 correction output into additionalContext.

- Traces to: UC-2 (UserPromptSubmit feedback)

---

## DES-7: PreToolUse advisory reminder format

When memories match a tool call's keyword filter, the hook returns a system reminder. Note: only tier A and B memories (with anti-patterns per REQ-7) can match; tier C contextual facts have no anti-patterns and do not appear.

```
<system-reminder source="engram">
[engram] Tool call advisory:
  - "<title1>" — <principle1> (file1.toml) [tier A]
  - "<title2>" — <principle2> (file2.toml) [tier B]
</system-reminder>
```

Format rules:
- Header: `[engram] Tool call advisory:`
- Each memory: title in quotes, principle, file path and tier in brackets
- If no matches, no output (tool call allowed silently)
- The agent exercises judgment with full session context — not a block, an advisory for consideration
- With `--format json`: wrapped in `{"summary": "[engram] N tool advisories.", "context": "<system-reminder>..."}`. Hook script reshapes into `{systemMessage, hookSpecificOutput: {additionalContext}}` per PreToolUse hook API.

- Traces to: UC-2 (PreToolUse advisory surfacing, tier-aware filtering)

---

## DES-8: Hook wiring — SessionStart and PreToolUse

SessionStart hook (`hooks/session-start.sh`) calls `engram surface --mode session-start --format json` after the existing build step, then reshapes into `{systemMessage, additionalContext}`. PreToolUse hook (`hooks/pre-tool-use.sh`) reads the tool call from stdin JSON, calls `engram surface --mode tool --format json`, and reshapes into `{systemMessage, hookSpecificOutput: {additionalContext}}`.

UserPromptSubmit hook (`hooks/user-prompt-submit.sh`) captures `engram correct` output and `engram surface --mode prompt --format json` output separately, then combines into a single JSON response with `{systemMessage, additionalContext}`. Correct output is prepended to additionalContext when present.

- Traces to: UC-2 (hook wiring)

---

## REQ-21: Per-memory surfacing instrumentation fields

Each memory TOML file supports three optional tracking fields updated during surface events:

- `surfaced_count` (int) — total number of times this memory has been surfaced. Defaults to 0 if absent.
- `last_surfaced` (string, RFC 3339) — timestamp of the most recent surfacing event. Defaults to zero time if absent.
- `surfacing_contexts` (string array) — bounded list (max 10) of recent context types. Each entry is the surface mode that triggered the event: `session-start`, `prompt`, or `tool`. When the list exceeds 10 entries, the oldest are dropped.

These fields are read during retrieval (alongside existing fields) and written back during surface events. They are purely additive — existing memories without these fields work unchanged (zero/empty defaults).

- Traces to: UC-2 (surfacing instrumentation)
- AC: (1) `surfaced_count` is an integer, defaults to 0 when absent. (2) `last_surfaced` is RFC 3339, defaults to zero time when absent. (3) `surfacing_contexts` is a string array, max 10 entries, FIFO eviction. (4) Fields are optional — memories without them parse and function normally. (5) Fields round-trip through TOML read/write without data loss.
- Verification: deterministic (TOML parse, field presence, value bounds)

---

## REQ-22: In-place TOML update on surfacing events

After each surfacing mode (session-start, prompt, tool) determines which memories matched, the system updates each matched memory's TOML file in-place:

1. Read the existing TOML file (full content, all fields).
2. Compute updated tracking fields: increment `surfaced_count`, set `last_surfaced` to current time, append mode to `surfacing_contexts` (with FIFO eviction at 10).
3. Write the updated TOML file atomically (temp file + rename).

All existing fields are preserved on round-trip — the update must not drop or modify any field other than the three tracking fields.

- Traces to: UC-2 (surfacing instrumentation, fire-and-forget)
- AC: (1) After surfacing, each matched memory's TOML file is updated with new tracking values. (2) Existing fields (title, content, keywords, etc.) are preserved exactly. (3) Writes are atomic (temp file + rename). (4) Instrumentation errors are logged to stderr but never fail the surface operation (ARCH-6 exit-0 contract). (5) If no memories match, no TOML updates occur.
- Verification: deterministic (file content before/after, field preservation, error isolation)

---

## REQ-23: Creation log format for deferred visibility

PreCompact and SessionEnd hooks cannot output to the user (no hook output mechanism). Instead, UC-1 learning logs creation events to `<data-dir>/creation-log.jsonl` in JSONL format (one JSON object per line). Each log entry contains:

```json
{
  "timestamp": "2026-03-06T12:34:56Z",
  "title": "Memory title",
  "tier": "A",
  "filename": "memory-filename.toml"
}
```

One entry per memory created. The log file is created if missing, appended to if exists. Fire-and-forget: creation log write failures are logged to stderr but don't fail the learn operation (ARCH-6 exit-0 contract).

- Traces to: UC-1 (creation visibility, deferred reporting)
- AC: (1) Each memory created by UC-1 appends one JSONL line to `creation-log.jsonl`. (2) Entry includes timestamp (RFC 3339), title, tier (A/B/C), and filename. (3) File is appended-to, created if missing. (4) Appends are atomic (temp write + rename to avoid corruption). (5) Write errors are logged to stderr, don't fail the operation.
- Verification: deterministic (JSONL format validation, file content check)

---

## REQ-24: SessionStart creation report — read and clear creation log

When a session starts (SessionStart hook), before surfacing the top 20 memories by recency, the system checks for a creation log (`<data-dir>/creation-log.jsonl`). If it exists:

1. Read all entries from the log
2. Format them as a report: "N memories created since last session: [titles and tiers]"
3. Include the report in the hook's `systemMessage` so the user sees what was learned
4. Delete the creation log after successful reporting

Fire-and-forget: log read/delete errors are logged to stderr but don't fail the SessionStart operation (ARCH-6 exit-0 contract).

- Traces to: UC-2 (SessionStart surfacing, creation visibility)
- AC: (1) SessionStart checks for `creation-log.jsonl`. (2) If present, reads all JSONL entries. (3) Formats a report: "N memories created since last session: [list]". (4) Includes report in `systemMessage` alongside the recency surfacing. (5) Deletes the log after reporting. (6) If no log exists, no report is included. (7) Read/delete errors logged to stderr, don't fail the operation.
- Verification: deterministic (log parsing, report format, file deletion)

---

# UC-1 Requirements and Design

---

## REQ-15: LLM extraction from transcript delta with unified tier criteria

When a PreCompact or Stop hook fires, the system extracts only the new transcript content since the last extraction (delta), preprocesses it with Strip (removing low-value content), and sends the cleaned delta to an LLM (claude-haiku-4-5-20251001) which extracts candidate learnings and classifies each using the same A/B/C tier criteria as UC-3 (REQ-7). Each candidate has the same structured fields as UC-3 memories: title, content, observation_type, concepts, keywords, principle, anti_pattern (tier-gated), rationale, filename_summary.

The LLM extracts from the delta:
- Explicit instructions the real-time classifier missed (tier A)
- Teachable corrections with generalizable principles (tier B)
- Contextual facts: architectural decisions, discovered constraints, working solutions, implicit preferences (tier C)

- Traces to: UC-1 (incremental LLM extraction, tier classification, anti-pattern gating)
- AC: (1) API call uses claude-haiku-4-5-20251001. (2) The prompt includes only the stripped transcript delta (new content since last extraction), not the full transcript. (3) Delta is obtained by reading transcript file from byte offset (REQ-26). (4) Delta is preprocessed with Strip to remove noise (tool results, base64, repeated schemas). (5) Response is parsed as a JSON array of candidate objects, each with required fields plus `tier` field (A/B/C). (6) Invalid or unparseable responses return an error. (7) Anti-pattern field is populated per REQ-7: required for A, optional for B, empty for C. (8) The LLM applies quality gate (REQ-16) and tier assignment simultaneously.
- Verification: deterministic (JSON schema validation, tier assignment accuracy, delta size < full transcript)

---

## REQ-16: Quality gate for extracted learnings

The LLM extraction prompt instructs rejection of low-quality candidates. Mechanical patterns (e.g., "ran tests before committing"), vague generalizations (e.g., "code should be clean"), and observations too narrow to be useful again are excluded from the candidate list.

- Traces to: UC-1 (quality gate)
- AC: (1) The system prompt explicitly instructs the LLM to reject mechanical patterns, vague generalizations, and overly narrow observations. (2) Only specific, actionable learnings with clear principles or anti-patterns are included. (3) The quality gate is embedded in the prompt, not a separate filtering step.
- Verification: LLM judgment (quality is prompt-enforced, verified via behavioral tests)

---

## REQ-17: Deduplication against existing memories

Before writing each candidate learning, the system checks existing TOML files in the memories directory. Candidates that substantially overlap an existing memory (by keyword overlap) are skipped. UC-3 mid-session captures take priority — session-end extraction never duplicates what was already captured.

- Traces to: UC-1 (deduplication)
- AC: (1) All existing `.toml` files in `<data-dir>/memories/` are read before writing. (2) For each candidate, keywords are compared against existing memories' keywords. (3) If keyword overlap exceeds a threshold (>50% of candidate's keywords match an existing memory's keywords), the candidate is skipped. (4) Deduplication is logged to stderr. (5) If zero candidates survive dedup, no files are written and no error is emitted.
- Verification: deterministic (keyword overlap calculation)

---

## REQ-18: Fail loudly when no API token

If no API token is configured, the system emits a loud stderr error and does not create any memory files. No degraded memories are ever written.

- Traces to: UC-1 (no graceful degradation)
- AC: (1) Missing API token → emit `[engram] Error: session learning skipped — no API token configured` to stderr. (2) No TOML files are created. (3) Exit 0 (don't break the hook chain). (4) Aligned with UC-3 (see REQ-1, no graceful degradation for classifier).
- Verification: deterministic (stderr check, no files created)

---

## REQ-19: Idempotency across multiple triggers

If both PreCompact and SessionEnd fire in the same session, the second invocation deduplicates against memories created by the first. Multiple PreCompact events in a long session each extract from the new transcript portion only.

- Traces to: UC-1 (idempotency)
- AC: (1) Each invocation reads existing memory files before writing (REQ-17 dedup covers this). (2) Memories written by a prior invocation in the same session are treated as existing memories for dedup. (3) No special session tracking is needed — file-based dedup is sufficient.
- Verification: deterministic (run twice, check no duplicates)

---

## REQ-20: CLI `learn` subcommand with delta tracking

The system adds a `learn` subcommand to the engram binary: `engram learn --data-dir <path> --transcript-path <transcript_file> --session-id <id>`. Transcript is read from file (not stdin) to enable incremental offset tracking.

- Traces to: UC-1 (CLI entry point, incremental learning)
- AC: (1) `engram learn --data-dir <path> --transcript-path <file> --session-id <id>` reads transcript from file. (2) Looks up learn offset for the session ID (REQ-26). (3) If session ID differs from stored session, resets offset to 0. (4) Reads transcript delta from byte offset. (5) If delta is empty, skips extraction (no API call). (6) If delta has content, preprocesses with Strip (REQ-27) and sends to LLM. (7) Updates offset after extraction. (8) Runs the extraction → dedup → write pipeline. (9) Exit 0 always — errors logged to stderr. (10) Pure Go, no CGO.
- Verification: deterministic (subcommand runs, delta extraction, offset persistence)

---

## REQ-25: Creation log write for deferred visibility

During UC-1 learning (PreCompact/SessionEnd), each memory file successfully written is also logged to `<data-dir>/creation-log.jsonl` (see REQ-23 for format). Logging happens after the TOML file is written. See REQ-23 for JSONL format and fire-and-forget error handling.

- Traces to: UC-1 (creation visibility, deferred reporting)
- AC: (1) For each memory file written, append one JSONL line to `creation-log.jsonl`. (2) Entry includes timestamp (RFC 3339), memory title, tier (A/B/C), and filename. (3) Appends are atomic (temp write + rename). (4) Write errors logged to stderr, don't fail the learning operation. (5) Creation log enables deferred visibility at next SessionStart (REQ-24).
- Verification: deterministic (JSONL format check, file append success)

---

## REQ-26: Offset tracking for incremental extraction

The system tracks the byte offset of the last extraction point per session to enable incremental transcript reading. Offset is stored in `<data-dir>/learn-offset.json` as a JSON object mapping session_id → byte_offset.

- Traces to: UC-1 (incremental learning, delta computation)
- AC: (1) Offset file is created if missing. (2) On each learn invocation, read offset for the provided session_id. (3) If session_id is not in the map, treat as new session: set offset to 0. (4) After extraction completes, update offset to current file end position. (5) File updates are atomic (temp write + rename). (6) Write errors logged to stderr, don't fail the learning operation. (7) Empty offset map or missing file → treat all sessions as new (offset 0).
- Verification: deterministic (JSON parsing, offset arithmetic, file atomicity)

---

## REQ-27: Transcript preprocessing with Strip

Before sending transcript delta to the LLM, the system preprocesses the content using Strip operation to remove low-value content and reduce token usage.

- Traces to: UC-1 (incremental learning, token efficiency)
- AC: (1) Strip removes tool results, base64/binary content, and repeated schemas. (2) Preserves user messages, assistant text, tool names, and error messages. (3) Applies same Strip logic used in UC-14 (context continuity, REQ-41). (4) Stripped delta is sent to LLM, not original delta. (5) Stripped size << original size (typically 80-90% reduction). (6) Strip errors logged to stderr, don't fail the operation.
- Verification: deterministic (content presence/absence check, size comparison)

---

## DES-9: Hook wiring — PreCompact and Stop

PreCompact hook (`hooks/pre-compact.sh`) and Stop hook (`hooks/stop.sh`) are registered in `hooks/hooks.json`. Both invoke the same pipeline: extract transcript_path and session_id from stdin JSON, pass to `engram learn --data-dir <path> --transcript-path <file> --session-id <id>`.

The hook reads transcript_path and session_id from the stdin JSON payload. Token retrieval uses the same platform-aware mechanism as DES-3 (macOS Keychain fallback to env var).

Both hooks are synchronous (not async) — they must complete before context compaction or session termination. Fire-and-forget error handling (ARCH-6): errors logged to stderr, exit 0 always.

- Traces to: UC-1 (incremental hook wiring)

---

## DES-10: Session learning feedback format

When learnings are extracted, the system emits to stderr (not stdout, since the session may be ending):

```
[engram] Extracted N learnings from session (A: 2, B: 1, C: 3).
  - "<title1>" [A] (file1.toml)
  - "<title2>" [B] (file2.toml)
  - "<title3>" [C] (file3.toml)
  ...
[engram] Skipped M duplicates.
```

Format rules:
- Header: `[engram] Extracted N learnings from session (A: X, B: Y, C: Z).` with tier breakdown
- Each learning: title in quotes, tier in brackets, file path in parentheses
- Duplicate count: only if M > 0
- If zero learnings after dedup, emit: `[engram] No new learnings extracted.`

- Traces to: UC-1 (feedback, tier classification)

---

## REQ-26: Surfacing log — write during surfacing, read-and-clear during evaluate

During each surfacing event (SessionStart, UserPromptSubmit, PreToolUse), the surfacer appends an entry to `<data-dir>/surfacing-log.jsonl` recording which memories were surfaced. The evaluate pass reads this file to determine what was surfaced during the session, then clears it.

- Traces to: UC-15 (surfacing log, session-scoped record)
- AC: (1) Each surfacing event appends one entry per matched memory to surfacing-log.jsonl. (2) Entry includes memory file path, surfacing mode, and timestamp. (3) Evaluate pass reads all entries and clears the file. (4) Missing file → empty list (no error). (5) Write errors are fire-and-forget (ARCH-6 exit-0 contract). (6) Read-and-clear is atomic — no partial reads.
- Verification: deterministic (file write/read, JSON parse)

---

## DES-11: Surfacing log JSONL format

Each line in `<data-dir>/surfacing-log.jsonl`:

```json
{"memory_path": "/path/to/memory.toml", "mode": "prompt", "surfaced_at": "2026-03-06T10:00:00Z"}
```

Fields:
- `memory_path` (string) — absolute path to the memory TOML file
- `mode` (string) — one of `session-start`, `prompt`, `tool`
- `surfaced_at` (string) — RFC 3339 timestamp

File is append-only within a session, read-and-cleared by `engram evaluate`.

- Traces to: UC-15 (surfacing log format), REQ-26

---

## REQ-27: LLM evaluation of surfaced memories against transcript

The evaluate pass sends the full session transcript plus the list of surfaced memories (with their content, principle, and anti-pattern) to an LLM (claude-haiku-4-5-20251001). The LLM classifies each surfaced memory's outcome:

- **followed** — the agent acted consistently with the memory's principle
- **contradicted** — the agent acted against the memory's principle or repeated the anti-pattern
- **ignored** — the memory was surfaced but not relevant to any decision in the session

The LLM provides brief evidence for each classification.

- Traces to: UC-15 (evaluation pass, outcome classification)
- AC: (1) LLM receives full transcript + surfaced memory list. (2) Each surfaced memory gets exactly one outcome: followed, contradicted, or ignored. (3) Each outcome includes a brief evidence string. (4) LLM response is parsed as JSON. (5) Invalid/unparseable responses return an error.
- Verification: deterministic (JSON schema validation of LLM response)

---

## DES-12: Evaluation LLM prompt design

The evaluation prompt includes:

**System prompt:** You are evaluating whether an AI agent followed, contradicted, or ignored specific memory advisories during a session. For each memory, classify the outcome based on the agent's actual behavior in the transcript.

**User prompt structure:**
1. List of surfaced memories, each with: title, principle, anti_pattern (if any), content
2. Full session transcript
3. Instruction: For each memory, return JSON with outcome and evidence

**Response format:** JSON array of objects, one per surfaced memory:
```json
[
  {"memory_path": "...", "outcome": "followed", "evidence": "Agent used targ test instead of go test at lines 45, 89"},
  {"memory_path": "...", "outcome": "ignored", "evidence": "Memory about fish shell was not relevant to this session's tasks"}
]
```

- Traces to: UC-15 (evaluation LLM), REQ-27

---

## REQ-28: Per-session evaluation log write

After the LLM classifies outcomes, write results to a per-session evaluation log file at `<data-dir>/evaluations/<timestamp>.jsonl`. Each line is one evaluated memory's outcome. The timestamp in the filename is RFC 3339 with colons replaced by hyphens for filesystem compatibility.

- Traces to: UC-15 (evaluation log, per-session storage)
- AC: (1) Evaluation directory is created if missing. (2) One file per evaluate invocation, named by timestamp. (3) Each line is valid JSON with memory_path, outcome, evidence, evaluated_at. (4) File write is atomic (temp + rename). (5) Empty surfacing log → no evaluation file created.
- Verification: deterministic (file exists, JSON parses, fields present)

---

## DES-13: Evaluation log JSONL schema and file naming

File path: `<data-dir>/evaluations/2026-03-06T10-00-00Z.jsonl`

Each line:
```json
{"memory_path": "/path/to/memory.toml", "outcome": "followed", "evidence": "brief explanation", "evaluated_at": "2026-03-06T10:00:00Z"}
```

Fields:
- `memory_path` (string) — absolute path to the evaluated memory TOML file
- `outcome` (string) — one of `followed`, `contradicted`, `ignored`
- `evidence` (string) — brief LLM explanation of the classification
- `evaluated_at` (string) — RFC 3339 timestamp

Session identity is implicit from the file. File naming: RFC 3339 timestamp with colons replaced by hyphens.

- Traces to: UC-15 (evaluation log format), REQ-28

---

## REQ-29: Effectiveness aggregation from evaluation logs

When surfacing memories (UC-2), compute effectiveness on-the-fly by reading all evaluation log files in `<data-dir>/evaluations/`. For each surfaced memory, aggregate outcomes across all sessions: count of followed, contradicted, ignored. Compute effectiveness percentage as `followed / (followed + contradicted + ignored) * 100`.

- Traces to: UC-15 (effectiveness annotations, read path)
- AC: (1) Read all `.jsonl` files in evaluations directory. (2) Parse each line and group by memory_path. (3) Compute per-memory totals: followed_count, contradicted_count, ignored_count. (4) Compute effectiveness percentage. (5) Missing evaluations directory → empty stats (no error). (6) Malformed lines skipped.
- Verification: deterministic (file reads, arithmetic)

---

## REQ-30: Effectiveness annotations during surfacing

When UC-2 surfaces memories, include effectiveness annotations for memories that have evaluation data. Format: "(surfaced N times, followed M%)" appended to the memory's surfacing output.

- Traces to: UC-15 (effectiveness visibility, inline annotations)
- AC: (1) Annotations appear when evaluation data exists for a surfaced memory. (2) Memories with no evaluation data show no annotation (backward compatible). (3) Format is "(surfaced N times, followed M%)" where N is total evaluations and M is effectiveness percentage. (4) Annotations appear in all surfacing modes (session-start, prompt, tool).
- Verification: deterministic (output format check)

---

## DES-14: Effectiveness annotation format and placement

Annotations are appended to each surfaced memory's line in the system reminder:

```
- "Use targ test not go test" — Use project-specific build tools (surfaced 5 times, followed 80%)
```

When no evaluation data exists for a memory, the annotation is omitted entirely — no "(surfaced 0 times)".

The annotation uses the total evaluation count (not surfaced_count from tracking) as N, and effectiveness percentage as M. This ensures the numbers reflect outcome data, not just surfacing events.

- Traces to: UC-15 (inline annotations UX), REQ-30

---

## REQ-31: SessionEnd evaluation summary — user-visible

After the evaluate pass completes, output a summary to the hook's `systemMessage` field so the user sees it. The summary reports the number of memories evaluated and the outcome breakdown.

- Traces to: UC-15 (SessionEnd visibility)
- AC: (1) Summary appears in hook `systemMessage` for user visibility. (2) Format shows total evaluated, followed count, contradicted count, ignored count. (3) If no memories were surfaced (empty surfacing log), no summary output. (4) Summary appears after learn output in the hook script.
- Verification: deterministic (hook output check)

---

## DES-15: Hook wiring — evaluate in PreCompact and SessionEnd

The PreCompact and SessionEnd hook scripts are extended to invoke `engram evaluate` after `engram learn`:

```bash
# UC-1: Extract learnings
$ENGRAM_BIN learn --data-dir "$ENGRAM_DATA" < transcript

# UC-15: Evaluate memory effectiveness
$ENGRAM_BIN evaluate --data-dir "$ENGRAM_DATA" < transcript
```

The evaluate subcommand reads the transcript from stdin (same as learn) and the surfacing log from the data directory. Output goes to stdout for the hook to reshape into `systemMessage`.

- Traces to: UC-15 (hook integration), REQ-31

---

## REQ-32: CLI `evaluate` subcommand

Go binary `engram` with an `evaluate` subcommand: `engram evaluate --data-dir <path>`. Reads transcript from stdin. Pure Go, no CGO.

- Traces to: UC-15 (CLI entry point)
- AC: (1) Binary accepts `evaluate` subcommand. (2) `--data-dir` flag specifies the data directory. (3) Reads transcript from stdin. (4) Exit 0 always — errors logged to stderr, never propagated as exit codes (ARCH-6). (5) Requires API token (same mechanism as correct/learn).
- Verification: deterministic (subcommand runs, exit code)

---

## REQ-33: No graceful degradation for evaluate

If no API token is configured, emit a loud stderr error (`[engram] Error: evaluation skipped — no API token configured`) and skip evaluation entirely. Never write degraded evaluations.

- Traces to: UC-15 (no degradation, same pattern as REQ-18)
- AC: (1) Missing token → stderr error + exit 0. (2) No evaluation log file created. (3) Error message includes `[engram] Error:` prefix.
- Verification: deterministic (error message check)

---

## REQ-34: Idempotency for evaluate across multiple triggers

If both PreCompact and SessionEnd fire in the same session, the second evaluate invocation reads an empty surfacing log (cleared by the first) and produces no evaluation file. No duplicate evaluations.

- Traces to: UC-15 (idempotency, same pattern as REQ-19)
- AC: (1) First evaluate reads and clears surfacing log. (2) Second evaluate finds empty/missing log → no evaluation produced. (3) No duplicate entries in evaluation logs.
- Verification: deterministic (file state check)

---

# UC-6: Memory Effectiveness Review

---

## REQ-35: Matrix classification

Classify each memory into one of four quadrants by combining two signals:

1. **Surfacing frequency:** Read `surfaced_count` from each memory's TOML tracking fields. Compute the median across all memories. Above median = "often surfaced", at or below median = "rarely surfaced".
2. **Follow-through rate:** Read `EffectivenessScore` from evaluation aggregation. >= 50% = "high follow-through", < 50% = "low follow-through".

Resulting quadrants:

|  | Often Surfaced | Rarely Surfaced |
|--|---|---|
| **High Follow-Through** | Working | Hidden Gem |
| **Low Follow-Through** | Leech | Noise |

Memories with fewer than 5 total evaluations (followed + contradicted + ignored) are classified as **insufficient data** — excluded from quadrant assignment and threshold flagging.

- Traces to: UC-6 (2x2 matrix classification)
- AC: (1) Median computed across all memories with tracking data. (2) Four quadrants assigned correctly. (3) Memories with <5 evaluations classified as insufficient data. (4) Memories with zero surfacing data classified as insufficient data.
- Verification: deterministic (arithmetic on known inputs)

---

## REQ-36: Threshold flagging

Flag a memory for action when both conditions are met:
1. It has 5 or more total evaluations (followed + contradicted + ignored)
2. Its effectiveness score is below 40%

Flagged memories include their quadrant assignment (leech or noise — working and hidden gem memories cannot have <40% effectiveness by definition since the high follow-through threshold is 50%).

- Traces to: UC-6 (threshold flagging), CLAUDE.md ("Pruned when utility < 0.4 after 5+ retrievals")
- AC: (1) Only memories with 5+ evaluations can be flagged. (2) Threshold is strictly less than 40%. (3) Flagged memories include quadrant. (4) Memories at exactly 40% are not flagged.
- Verification: deterministic (threshold arithmetic)

---

## REQ-37: Effectiveness annotations during surfacing

When UC-2 surfaces memories (any mode: session-start, prompt, tool), annotate each surfaced memory with effectiveness context if evaluation data exists:

Format: `(surfaced N times, followed M%)`

Where N is the total evaluation count (followed + contradicted + ignored) and M is the effectiveness score rounded to the nearest integer.

Computed on-the-fly by reading evaluation log files. Fire-and-forget: if reading evaluation data fails, omit the annotation silently (ARCH-6 exit-0 contract). Memories with no evaluation history show no annotation.

- Traces to: UC-6 (effectiveness annotations), UC-2 (surfacing output)
- AC: (1) Annotation appears after memory title in surfacing output. (2) N = total evaluations, M = effectiveness score %. (3) Missing/empty evaluations → no annotation (not "(surfaced 0 times, followed 0%)"). (4) Read failure → annotation omitted, surfacing succeeds.
- Verification: deterministic (format check + error path)

---

## DES-17: Annotation format

Effectiveness annotations are appended to the memory identifier line in surfacing output. Example:

```
  - use-targ-not-go-test (matched: test, go) (surfaced 8 times, followed 75%)
  - always-use-fish-shell (matched: shell) (surfaced 3 times, followed 100%)
  - port-is-3001 (matched: port)
```

The third memory has no evaluation data, so no annotation appears. Annotation is parenthesized and follows the keyword match parenthetical.

- Traces to: REQ-37 (effectiveness annotations), UC-2 (surfacing output format)

---

## REQ-38: `engram review` CLI command

New subcommand `engram review --data-dir <path>` that reads memory tracking data and evaluation logs, then outputs the effectiveness matrix.

Output sections (in order):
1. **Summary line:** Total memories, total with sufficient data, total flagged.
2. **Per-quadrant table:** Count of memories in each quadrant.
3. **Flagged memories:** Name, quadrant, surfaced count, effectiveness score, evaluation count. Sorted by effectiveness score ascending (worst first).
4. **Insufficient-data list:** Name, surfaced count, evaluation count. Only shown if such memories exist.

Exit 0 always. `--data-dir` is required.

- Traces to: UC-6 (`engram review` CLI)
- AC: (1) All four sections present when data exists. (2) Flagged sorted by effectiveness ascending. (3) Missing `--data-dir` → usage error, exit 0. (4) Insufficient-data section omitted when all memories have 5+ evaluations.
- Verification: deterministic (output format check)

---

## DES-16: Review output format

Human-readable text output for `engram review`:

```
[engram] Memory Effectiveness Review
  Total: 25 memories, 18 with sufficient data, 3 flagged

  Quadrant Summary:
    Working:    8  (often surfaced, high follow-through)
    Hidden Gem: 4  (rarely surfaced, high follow-through)
    Leech:      2  (often surfaced, low follow-through)
    Noise:      4  (rarely surfaced, low follow-through)

  Flagged for action (effectiveness < 40%, 5+ evaluations):
    eye-mouth-caps-topology    Leech    surfaced: 12  effectiveness: 17%  evaluations: 6
    medial-axis-constraints    Noise    surfaced: 2   effectiveness: 25%  evaluations: 8
    junction-sector-coverage   Noise    surfaced: 1   effectiveness: 33%  evaluations: 9

  Insufficient data (< 5 evaluations):
    new-memory-recent          surfaced: 3  evaluations: 1
    another-new-memory         surfaced: 0  evaluations: 0
```

No evaluation data at all → single line: `[engram] No evaluation data found.`

- Traces to: REQ-38 (`engram review` output)

---

## REQ-39: No-data behavior

When the evaluation directory is missing or contains zero `.jsonl` files, `engram review` outputs `[engram] No evaluation data found.` and exits 0. No quadrant classification attempted.

When tracking data exists but evaluation data does not, all memories are classified as insufficient data.

- Traces to: UC-6 (no graceful degradation)
- AC: (1) Missing eval dir → no-data message, exit 0. (2) Empty eval dir → no-data message, exit 0. (3) Tracking data without eval data → all insufficient. (4) No crash or panic on missing data.
- Verification: deterministic (file absence check)

---

# UC-14: Structured Session Continuity

---

## REQ-40: Transcript delta extraction

Read the transcript JSONL file from a given byte offset (watermark) to end-of-file. Return the raw lines from the offset onward and the new byte offset (file size after read). If the file is shorter than the watermark (new session, file rotated), reset to offset 0 and read the entire file.

- Traces to: UC-14 (incremental context update step 1)
- AC: (1) Reading from offset 0 returns full file. (2) Reading from mid-file offset returns only new lines. (3) File shorter than watermark resets to 0. (4) Empty file returns empty delta and offset 0.
- Verification: deterministic (byte offset math)

---

## REQ-41: Low-value content stripping

Given raw transcript JSONL lines, strip low-value content and retain high-value content. Strip: tool result content blocks, base64-encoded data, content blocks longer than 2000 characters. Retain: user messages (role=user), assistant text messages (role=assistant), tool use names and commands (not full results), error messages.

- Traces to: UC-14 (incremental context update step 2)
- AC: (1) Tool result blocks are removed. (2) Base64 strings (>100 chars of `[A-Za-z0-9+/=]`) are replaced with `[base64 removed]`. (3) Content blocks >2000 chars are truncated with `[truncated]`. (4) User messages preserved verbatim. (5) Assistant text preserved. (6) Tool names preserved, tool results stripped.
- Verification: deterministic (string processing)

---

## REQ-42: Watermark tracking

The byte offset watermark and session ID are persisted in the context file's HTML comment metadata. On each update, the new offset is written. On SessionStart, if the session ID differs from the stored one, the watermark resets to 0 (new session = new transcript file).

- Traces to: UC-14 (incremental context update step 1, context file format)
- AC: (1) Metadata is parseable from HTML comment. (2) Offset updates after each write. (3) Session ID mismatch resets offset to 0. (4) Missing file = offset 0, empty session ID.
- Verification: deterministic (metadata parsing)

---

## REQ-43: Context summarization via Haiku

Given a previous summary (possibly empty) and a stripped transcript delta, call claude-haiku-4-5-20251001 to produce an updated task-focused working summary. The prompt instructs: focus on what's being worked on, decisions made, progress, and open questions. No constraints/patterns (those are memories). If the API token is empty, skip silently and return the previous summary unchanged. If the API call fails, return the previous summary unchanged (no degraded output).

- Traces to: UC-14 (incremental context update step 3, no graceful degradation)
- AC: (1) Empty previous summary + delta → new summary. (2) Existing summary + delta → updated summary. (3) Empty delta → no API call, return previous summary. (4) Empty token → no API call, return previous summary. (5) API error → return previous summary.
- Verification: API call requires mock in tests

---

## REQ-44: Context file write

Write the session context to `.claude/engram/session-context.md` atomically (temp + rename). Format: HTML comment with metadata (updated timestamp, byte offset, session ID) followed by the summary as plain markdown. Create `.claude/engram/` directory if missing.

- Traces to: UC-14 (context file format, file location)
- AC: (1) File contains HTML comment header with metadata fields. (2) Summary follows as plain markdown. (3) Directory created if missing. (4) Atomic write (temp file + rename). (5) Existing file overwritten completely.
- Verification: deterministic (file format check)

---

## REQ-45: Context file read and restore

On SessionStart, read `.claude/engram/session-context.md` if it exists. Extract the markdown summary (skip HTML comment). Return the summary for injection as `additionalContext`. Missing file returns empty string (no error). Always load regardless of file age.

- Traces to: UC-14 (restore on SessionStart)
- AC: (1) Existing file → summary extracted. (2) Missing file → empty string, no error. (3) HTML comment metadata is not included in the returned summary. (4) File age is irrelevant — always loaded.
- Verification: deterministic (file read + parse)

---

## REQ-46: Frecency activation scoring (ACT-R model)

Each memory has a frecency activation score computed from four components:

1. **Frequency:** `log(1 + surfaced_count)` — logarithmic scaling prevents high-frequency memories from dominating.
2. **Recency:** `1 / (1 + hours_since_last_surfaced)` — time decay based on hours since last surfacing event. If never surfaced, uses `updated_at` as fallback.
3. **Spread:** `log(1 + len(surfacing_contexts))` — diversity of contexts the memory was surfaced in. More contexts = more broadly useful.
4. **Effectiveness:** `max(0.1, effectiveness_score / 100)` — from UC-15 evaluations. Defaults to 0.5 when no evaluation data exists (neutral). Floor of 0.1 prevents zero-multiplication.

**Combined activation:** `frequency × recency × spread × effectiveness`

For combined BM25 + frecency ranking (prompt/tool modes): `bm25_score × (1 + activation)` — the `(1 + activation)` factor boosts BM25 scores by frecency without allowing frecency alone to override BM25 relevance (BM25 of zero stays zero).

- Traces to: UC-2 (surfacing ranking)
- AC: (1) Activation score computed from all four components. (2) Never-surfaced memories use updated_at for recency, 0 for frequency/spread, default effectiveness. (3) Combined score preserves BM25 relevance as primary signal. (4) All components are non-negative.
- Verification: deterministic (pure math on stored fields + effectiveness data)

---

## DES-18: UserPromptSubmit context-update pipeline

Context-update runs as a separate async hook entry in hooks.json (`"async": true`), not inside the synchronous UserPromptSubmit script. A dedicated script (`user-prompt-submit-async.sh`) handles only context-update. The existing `user-prompt-submit.sh` handles correct + surface synchronously.

- Traces to: UC-14 (piggybacked on UserPromptSubmit)
- AC: (1) context-update runs via separate async hook entry (not nohup/disown). (2) Synchronous hook still returns correct/surface output promptly. (3) context-update receives transcript path and session ID from hook JSON stdin.

---

## DES-19: PreCompact final flush

The PreCompact hook script calls `engram context-update` with the same flags as UserPromptSubmit. This is a synchronous call (not background) since PreCompact has a 60s timeout and this is the last chance before compaction.

- Traces to: UC-14 (final flush)
- AC: (1) context-update called synchronously. (2) Same pipeline as UserPromptSubmit (delta + strip + summarize + write).

---

## DES-20: Context file format specification

The context file at `.claude/engram/session-context.md` has this format:

```markdown
<!-- engram session context | updated: <RFC3339> | offset: <int> | session: <id> -->

<summary markdown>
```

The HTML comment is a single line. Metadata fields are pipe-delimited key-value pairs. The summary follows after a blank line.

- Traces to: UC-14 (context file format)
- AC: (1) First line is HTML comment with required fields. (2) Fields parseable by simple string splitting. (3) Summary is valid markdown. (4) Human-readable when opened in any text editor.

---

## DES-21: CLI context-update subcommand

New `engram context-update` subcommand. Flags: `--transcript-path` (required), `--session-id` (required), `--data-dir` (required), `--context-path` (optional, overrides default context file location). Pipeline: read watermark from context file → extract delta from transcript → strip low-value content → if delta non-empty, summarize via Haiku → write context file with updated watermark. Exit 0 always (fire-and-forget per ARCH-6).

When `--context-path` is provided, the context file is written to that path instead of `<data-dir>/session-context.md`. This enables per-project context storage.

- Traces to: UC-14 (CLI wiring), REQ-40 through REQ-44
- AC: (1) Missing transcript file → exit 0, no context file written. (2) Empty delta → exit 0, context file unchanged. (3) Successful update → context file written with new watermark. (4) API error → exit 0, context file unchanged. (5) `--context-path` provided → writes to that path instead of default.

---

## DES-22: SessionStart context injection

Session context is stored per-project using a slug derived from `$PWD` (replace `/` with `-`), following Claude Code's MEMORY.md path convention. The SessionStart hook computes the project slug, then reads `~/.claude/engram/data/projects/<slug>/session-context.md` (if it exists) and includes the summary in the `additionalContext` field of its JSON output. The Stop hook writes to the same per-project path via `--context-path`. If the file doesn't exist, no context is injected (no error).

- Traces to: UC-14 (restore on SessionStart), REQ-45
- AC: (1) Context file exists → summary appears in additionalContext. (2) No file → additionalContext contains only memory context (existing behavior). (3) Context is clearly labeled so the model knows it's a session resumption summary. (4) Context is project-specific — switching projects surfaces the correct project's context.

---

# UC-14 Architecture Layer (ARCH)

---

## ARCH-28: TranscriptDeltaReader component

Reads transcript JSONL file from a given byte offset (watermark) to end-of-file, handling watermark reset when file is shorter than the stored offset (session boundary or file rotation).

- Traces to: REQ-40 (transcript delta extraction)
- Responsibilities: (1) Open transcript file at given path. (2) Seek to byte offset. (3) Read lines to EOF. (4) If file size < offset, reset to 0 and re-read entire file. (5) Return (raw lines, new byte offset after read).
- Behavioral contract: Deterministic file I/O; no errors on missing file (return empty delta, offset 0 per ARCH-32 fire-and-forget).

---

## ARCH-29: ContentStripper component

Removes low-value content from raw transcript JSONL lines and retains high-value content.

- Traces to: REQ-41 (low-value content stripping)
- Responsibilities: (1) Parse each JSONL line to extract role and content blocks. (2) Strip tool result content blocks (toolResult role). (3) Replace base64-encoded strings (>100 chars of `[A-Za-z0-9+/=]`) with `[base64 removed]`. (4) Truncate content blocks >2000 chars with `[truncated]`. (5) Retain user messages, assistant text messages, tool use names/commands, error messages.
- Behavioral contract: Lossless on high-value content; destructive on low-value content. Output is valid JSONL or plain text for Haiku summarization.

---

## ARCH-30: ContextSummarizer component

Calls Haiku API to produce an updated task-focused working summary from previous summary and stripped transcript delta.

- Traces to: REQ-43 (context summarization via Haiku)
- Responsibilities: (1) Accept previous summary (possibly empty string), stripped delta (possibly empty). (2) If delta empty, return previous summary unchanged. (3) If delta non-empty, call Haiku API with prompt and system message (mocked in tests). (4) Return Haiku response. (5) On API error, return previous summary unchanged. (6) On empty token, return previous summary unchanged.
- Behavioral contract: Fire-and-forget (no errors thrown; degradation is silent return of previous state). Always returns a valid summary string (possibly empty).
- DI injection: HaikuClient interface for API calls, Timestamper interface for logging/audit.

---

## ARCH-31: SessionContextFile data structure and I/O

Encapsulates the context file format and read/write operations.

- Traces to: REQ-42 (watermark tracking), REQ-44 (context file write), REQ-45 (context file read/restore), DES-20 (file format spec)
- Data structure: SessionContext struct with fields: Timestamp (RFC3339), Offset (int64), SessionID (string), Summary (string).
- File format: HTML comment line with pipe-delimited metadata, blank line, markdown summary.
- Responsibilities: (1) Parse HTML comment to extract metadata. (2) Extract markdown summary (skip HTML comment). (3) Write context file atomically (temp + rename). (4) Create `.claude/engram/` directory if missing.
- Behavioral contract: ReadSessionContext returns (summary, offset, sessionID); WriteSessionContext is atomic (no partial writes). Missing file on read returns ("", 0, ""). File overwrite is total (no merge).
- DI injection: FileReader, FileWriter interfaces for testability.

---

## ARCH-32: ContextUpdateOrchestrator

Orchestrates the full incremental context update pipeline.

- Traces to: DES-21 (CLI context-update subcommand), REQ-40–REQ-44
- Responsibilities: (1) Accept CLI flags: --transcript-path, --session-id, --data-dir, --context-path (optional). (2) Read watermark from context file (or 0 if missing). (3) Extract transcript delta from ARCH-28. (4) Check session ID mismatch: if current session ID ≠ stored session ID, reset offset to 0. (5) Strip content via ARCH-29. (6) If delta non-empty, summarize via ARCH-30. (7) Write updated context file via ARCH-31 with new offset. (8) Exit 0 always (fire-and-forget per ARCH-6). When --context-path is provided, it overrides the default `<data-dir>/session-context.md` location.
- Behavioral contract: No error output on missing transcript file, empty delta, or API error. All errors are silent (logged but not surfaced). Exit code always 0.
- Entry point: CLI binary, invoked by Stop hook (synchronous) with --context-path set to per-project path.

---

## ARCH-33: Hook integration wiring

Specifies how context updates are triggered and how context is restored.

- Traces to: DES-18 (UserPromptSubmit pipeline), DES-19 (PreCompact flush), DES-22 (SessionStart injection)
- UserPromptSubmit hook: Context-update runs as a separate async hook entry (`"async": true` in hooks.json) via `user-prompt-submit-async.sh`. The synchronous `user-prompt-submit.sh` handles correct/surface only.
- PreCompact hook: Call `engram context-update` synchronously with same flags. Wait for completion (60s timeout available per hook config).
- Stop hook: Call `engram context-update` with `--context-path` pointing to per-project path: `~/.claude/engram/data/projects/<slug>/session-context.md`, where `<slug>` is `$PWD` with `/` replaced by `-` (matching Claude Code's MEMORY.md convention).
- SessionStart hook: Compute project slug from `$PWD`, read per-project context file (if exists) via ARCH-31. Extract summary and inject as `additionalContext` in hook JSON output. Annotate clearly so the model knows it's a session resumption summary.
- Behavioral contract: Hooks remain responsive; background calls don't block. SessionStart always succeeds (missing file → no injection, no error). Context is project-specific — switching projects surfaces the correct project's context.

---

## ARCH-34: Dependency injection interfaces for testability

Specifies interfaces for all external I/O to enable unit testing without mocks.

- Traces to: ARCH-28 through ARCH-33 (all components use DI)
- Interfaces:
  - **FileReader:** Read(path string) (content []byte, err error)
  - **FileWriter:** Write(path string, content []byte) error (atomic via temp + rename)
  - **Timestamper:** Now() time.Time (for RFC3339 metadata)
  - **HaikuClient:** Summarize(ctx context.Context, previousSummary, delta string) (string, error)
- Contract: All business logic in internal/ uses these interfaces. Real implementations (os.*, http.*, time) wired only at edges (cmd/ for CLI, root-level for hook integration).
- Testing: Mock implementations injected in imptest; real I/O tested sparingly in integration tests.

---

## REQ-47: Quadrant partitioning for maintenance

When `engram maintain` runs, it reuses the existing `review.Classify` function (REQ-35 median split, REQ-36 threshold flagging) to partition memories into quadrants. No new classification logic. Only memories with 5+ evaluations are actionable — insufficient-data memories are excluded from proposals.

- Traces to: UC-16 (quadrant partitioning)
- AC: Maintain produces proposals only for memories classified into Working/Leech/Hidden Gem/Noise. Insufficient-data memories produce no proposals.

---

## REQ-48: Working quadrant — staleness detection

For memories in the Working quadrant, detect staleness based on `updated_at` age. A memory is stale if `updated_at` is older than a configurable threshold (default: 90 days). Stale working memories get a "review staleness" proposal. Non-stale working memories get no proposal (they're fine).

- Traces to: UC-16 (working quadrant), issue #40
- AC: Working memories older than the staleness threshold produce a proposal with `action: "review_staleness"`. Working memories within threshold produce no proposal.

---

## REQ-49: Leech quadrant — LLM-powered root cause diagnosis

For memories in the Leech quadrant (often surfaced, low follow-through), call Haiku to diagnose root cause and propose fixes. The LLM receives the memory's content (title, principle, anti_pattern, keywords, content) plus effectiveness stats (surfaced count, follow rate). The LLM proposes specific field-level changes: rewritten content, adjusted keywords, or tier change.

- Traces to: UC-16 (leech quadrant), issue #41
- AC: Each leech memory produces a proposal with `action: "rewrite"` containing specific TOML field changes proposed by the LLM.

---

## REQ-50: Hidden gem quadrant — LLM-powered keyword broadening

For memories in the Hidden Gem quadrant (high follow-through, rarely surfaced), call Haiku to propose expanded keywords and concepts. The LLM receives the memory's content plus stats, and proposes additional keywords/concepts to broaden surfacing triggers.

- Traces to: UC-16 (hidden gem quadrant), issue #42
- AC: Each hidden gem memory produces a proposal with `action: "broaden_keywords"` containing proposed keyword additions.

---

## REQ-51: Noise quadrant — removal proposal with evidence

For memories in the Noise quadrant (rarely surfaced, low follow-through), generate a deterministic removal proposal. No LLM needed — the evidence is the stats themselves: surfacing count, follow rate, age, evaluation count. The proposal includes all evidence for user review.

- Traces to: UC-16 (noise quadrant), issue #43
- AC: Each noise memory produces a proposal with `action: "remove"` and evidence fields (surfaced_count, effectiveness_score, evaluation_count, age_days).

---

## REQ-52: Fire-and-forget proposal generation

Individual proposal failures (LLM timeout, parse error) must not block other proposals. Failed proposals are omitted from output. The maintain command always exits 0. Errors logged to stderr.

- Traces to: UC-16 (fire-and-forget), ARCH-6
- AC: If the LLM call for one leech memory fails, other proposals (including other leech memories) still appear in output.

---

## REQ-53: `engram maintain` CLI subcommand

New CLI subcommand: `engram maintain --data-dir <path>`. Reads effectiveness data and memory tracking, generates proposals, writes JSON to stdout. Requires `ANTHROPIC_API_KEY` for leech/hidden-gem LLM calls. If no API key and there are leech/hidden-gem memories, those proposals are skipped (not an error).

- Traces to: UC-16 (CLI command)
- AC: `engram maintain --data-dir /path` outputs JSON array of proposals to stdout. Missing API key skips LLM proposals but still outputs working/noise proposals.

---

## DES-23: Proposal output format

Output is a JSON array of proposal objects. Each proposal has:

```json
{
  "memory_path": "path/to/memory.toml",
  "quadrant": "leech",
  "diagnosis": "Surfaced 15 times but only followed 20%. Keywords may be too broad.",
  "action": "rewrite",
  "details": {
    "proposed_keywords": ["new", "keywords"],
    "proposed_principle": "Reworded principle",
    "rationale": "Why this change helps"
  }
}
```

Action types: `review_staleness`, `rewrite`, `broaden_keywords`, `remove`.
Details vary by action type.

- Traces to: UC-16 (output format)

---

## DES-24: LLM prompt design for leech diagnosis

System prompt instructs Haiku to analyze a memory's content and effectiveness stats, diagnose why it's being ignored, and propose specific TOML field changes. User prompt includes the full memory content and stats. Output format: JSON with proposed field changes.

- Traces to: REQ-49 (leech diagnosis)

---

## DES-25: LLM prompt design for hidden gem broadening

System prompt instructs Haiku to analyze a memory's content and propose additional keywords/concepts that would broaden its surfacing triggers. User prompt includes the full memory content, current keywords, and stats. Output format: JSON with proposed keyword additions.

- Traces to: REQ-50 (hidden gem broadening)

---

## REQ-54: No-data behavior for maintain

If no memories have 5+ evaluations, `engram maintain` outputs an empty JSON array `[]` and exits 0. No error message, no degraded output.

- Traces to: UC-16 (no-data behavior)
- AC: Empty evaluation directory → `[]` output, exit 0.

---

## UC-17: Context Budget Management — Requirements

---

## REQ-55: Token estimation formula

Token estimation uses the formula: `len(text) / 4` as a conservative estimator for English text with code snippets. This formula is applied consistently at every surfacing point (SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop hooks).

- Traces to: UC-17 (token estimation)
- AC: (1) All context strings are tokenized using `len(text) / 4`. (2) Formula is applied uniformly across all hook points. (3) Fractional tokens are truncated (floor). (4) Empty strings → 0 tokens. (5) Formula is documented in code comments.
- Verification: deterministic (arithmetic, string length)

---

## DES-16: Token estimation implementation in surface.go

Token estimation is implemented as a pure function in `internal/surface/surface.go`: `func estimateTokens(text string) int { return len(text) / 4 }`. This function is called for each surfaced memory before adding it to the output. The function signature is exported for testing purposes.

- Traces to: REQ-55 (token estimation)

---

## REQ-56: Per-hook budget caps

Each hook point has a configurable token budget cap. Default values are: SessionStart 800 tokens, UserPromptSubmit 300 tokens, PreToolUse 200 tokens, PostToolUse 100 tokens, Stop audit 500 tokens. Caps are configurable via a config file or environment variables. If a cap is not set, the default is used.

- Traces to: UC-17 (per-hook budget)
- AC: (1) Each hook has a named cap (sessionStartBudget, userPromptBudget, preToolBudget, postToolBudget, stopBudget). (2) Caps default to the specified values. (3) Caps can be overridden via config file or env var with structured naming (e.g., `ENGRAM_BUDGET_SESSION_START=1000`). (4) Invalid cap values (non-positive) fall back to default. (5) Cap values are logged at startup.
- Verification: deterministic (config parsing, value validation)

---

## DES-17: Budget cap configuration UX

Budget caps are configured in `<data-dir>/config.toml` under a `[budget]` section. Default config is generated on first run if missing. Config file format:

```toml
[budget]
session_start = 800
user_prompt = 300
pre_tool = 200
post_tool = 100
stop_audit = 500
```

Users can edit the file to adjust caps. Invalid or missing values fall back to defaults. `engram review` displays the current caps in the output.

- Traces to: REQ-56 (per-hook budget)

---

## REQ-57: Priority allocation by effectiveness × relevance

When surfacing memories for a hook, the system sorts by `effectiveness_score × relevance_score` in descending order. Memories are filled into the output until the remaining budget is exhausted. Memories that don't fit are silently skipped.

- Traces to: UC-17 (priority allocation)
- AC: (1) Memories are sorted by (effectiveness × relevance) in descending order before filling. (2) Effectiveness score is the aggregated score from UC-6. (3) Relevance score is the BM25 score computed by UC-2. (4) Memories are added in order until remaining budget < estimated tokens for next memory. (5) Skipped memories are not logged (silent cutoff for performance).
- Verification: deterministic (sorting, arithmetic)

---

## DES-18: Priority sorting in matchPromptMemories and matchToolMemories

In `internal/surface/surface.go`, both `matchPromptMemories` and `matchToolMemories` functions are updated to:
1. Compute `effectiveness × relevance` for each memory
2. Sort by this score in descending order
3. Apply budget limit by counting tokens and stopping when budget is exhausted

The sorting is applied before the top-N limit is applied. Budget enforcement takes precedence over top-N limits.

- Traces to: REQ-57 (priority allocation)

---

## REQ-58: Budget reporting in engram review

The `engram review` command outputs budget utilization metrics for each hook point. Metrics include: hook name, current cap, total tokens surfaced, utilization percentage, and warning status (see REQ-59).

- Traces to: UC-17 (budget reporting)
- AC: (1) `engram review` output includes a budget summary section. (2) For each hook: name, cap, surfaced tokens, percentage utilization. (3) Percentage is computed as (surfaced tokens / cap) × 100. (4) If no surfacing occurred for a hook, tokens = 0, percentage = 0%. (5) Output is human-readable, one hook per line.
- Verification: deterministic (output format, arithmetic)

---

## DES-19: Budget reporting format in review output

`engram review` output includes a `[Budget Utilization]` section with a table format:

```
[Budget Utilization]
Hook                Budget   Surfaced   Utilization   Warning
SessionStart        800      720        90%            —
UserPromptSubmit    300      280        93%            ⚠ Frequently capped
PreToolUse          200      200        100%           ⚠ Consistently capped
PostToolUse         100      0          0%             —
StopAudit           500      350        70%            —
```

The "Warning" column shows warnings (see REQ-59). The table is written to the review output in plaintext format.

- Traces to: REQ-58 (budget reporting)

---

## REQ-59: Budget warning detection

When a hook's surfacing exceeds its budget cap on >50% of invocations during a session, a warning is emitted in `engram review` output. Warnings are non-fatal and advisory.

- Traces to: UC-17 (budget warnings)
- AC: (1) The surfacing logger (REQ-51, UC-2) records hook name, cap, and actual tokens for each invocation. (2) At review time, compute the percentage of invocations where (actual tokens > cap). (3) If percentage > 50%, emit warning: "Hook X is hitting its budget cap on Z% of invocations. Consider increasing its cap or reviewing memory quality." (4) Warning format: starts with `⚠` symbol. (5) Warnings are written to the "Warning" column in budget reporting table.
- Verification: deterministic (statistical counting, threshold check)

---

## DES-20: Budget warning computation in review

In the review command output, after loading all surfacing logs for the session, compute cap hit rates per hook:

```
cap_hits[hook] = count(invocations where tokens_surfaced > cap)
cap_hit_rate[hook] = cap_hits[hook] / total_invocations[hook]
```

If `cap_hit_rate[hook] > 0.5`, include a warning in the budget table. Warning format: `⚠ Hitting cap on Z% of invocations` (where Z is cap_hit_rate × 100, rounded to nearest integer).

- Traces to: REQ-59 (budget warning detection)

---

## UC-19: Stop Session Audit — Requirements

---

## REQ-60: Stop hook audit phase timing

The Stop hook executes in this order:
1. `engram learn` (incremental learning)
2. `engram evaluate` (update effectiveness scores)
3. `engram audit` (run session audit) ← NEW
4. `engram context-update` (refresh context embeddings)

The audit phase runs after effectiveness evaluation and before context update. This allows audit results to feed into the next session's effectiveness pipeline.

- Traces to: UC-19 (stop hook timing)
- AC: (1) Stop hook script invokes `engram audit` after `engram evaluate` and before `engram context-update`. (2) If audit fails (e.g., no API token), audit phase is skipped and other phases continue. (3) Hook script order is documented in `hooks/stop.sh`.
- Verification: deterministic (script execution order)

---

## DES-21: Stop hook script phase ordering

The `hooks/stop.sh` script is updated to invoke `engram audit` as a new phase. Pseudo-code order:

```bash
engram learn --transcript-path "$TRANSCRIPT_PATH" --session-id "$SESSION_ID"
engram evaluate --data-dir "$DATA_DIR"
engram audit --data-dir "$DATA_DIR" --timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
engram context-update --data-dir "$DATA_DIR"
```

Each phase's errors are logged but do not block subsequent phases (fire-and-forget pattern).

- Traces to: REQ-60 (stop hook timing)

---

## REQ-61: Audit scope definition

The audit scope includes: all high-priority memories surfaced during the session (determined by effectiveness tier), their outcomes (was the memory followed? did it change behavior?), and for skills invoked during the session, verification of critical steps.

- Traces to: UC-19 (audit scope)
- AC: (1) High-priority = effectiveness tier ≥ threshold (default: top 20% by effectiveness score). (2) Surfacing log (from UC-2) is read to extract surfaced memory IDs and timestamps. (3) For each surfaced memory, look up its content and effectiveness score. (4) For skills invoked (skill names from transcript), identify critical steps from skill instructions. (5) Compile scope as a list of (memory_id or skill_name, outcome_evidence).
- Verification: deterministic (scoring, log parsing, threshold check)

---

## DES-22: Audit scope parsing from session data

The audit command parses session data from:
1. Surfacing log: `<data-dir>/logs/surfacing-<session-id>.json` — contains surfaced memory IDs, hook names, timestamps
2. Effectiveness data: `<data-dir>/evaluate/effectiveness.json` — contains effectiveness scores
3. Transcript: passed as argument to audit command — contains transcript text to search for skill invocations and instruction compliance

Scope is compiled as JSON: `[{memory_id, effectiveness_score, surfaced_count}, ...]`

- Traces to: REQ-61 (audit scope)

---

## REQ-62: LLM compliance assessment

A single Haiku API call receives the audit scope (memories + outcomes) and the session transcript. The LLM assesses whether high-priority instructions were followed during the session. Response format: JSON array of compliance assessments.

- Traces to: UC-19 (LLM assessment)
- AC: (1) LLM call uses claude-haiku-4-5-20251001. (2) System prompt instructs Haiku to check instruction compliance. (3) User prompt includes: audit scope (JSON), session transcript excerpt (if feasible). (4) Response format: JSON array with one object per instruction: `{instruction, compliant: bool, evidence: string}`. (5) Invalid responses are logged and omitted from audit report.
- Verification: deterministic (LLM API call, JSON parsing)

---

## DES-23: Compliance assessment prompt structure

System prompt for Haiku:

```
You are auditing a session for compliance with high-priority instructions. You will be given:
- A list of instructions that were surfaced during the session (high-priority memories)
- The transcript of the session

For each instruction, determine whether the model complied:
- If the instruction was followed, answer "compliant: true"
- If the instruction was violated, answer "compliant: false"
- In both cases, provide evidence from the transcript

Output JSON format:
[
  {"instruction": "...", "compliant": true/false, "evidence": "..."},
  ...
]
```

User prompt includes scope JSON and relevant transcript excerpts.

- Traces to: REQ-62 (LLM compliance assessment)

---

## REQ-63: Audit report format

The audit report is written to `<data-dir>/audits/<timestamp>.json`. Format: JSON object with metadata and results.

```json
{
  "session_id": "...",
  "timestamp": "2026-03-08T15:30:00Z",
  "total_instructions_audited": 12,
  "compliant": 10,
  "non_compliant": 2,
  "results": [
    {
      "instruction": "...",
      "compliant": true,
      "evidence": "..."
    },
    ...
  ]
}
```

- Traces to: UC-19 (audit report)
- AC: (1) File is written to `audits/` subdirectory with ISO 8601 timestamp as filename. (2) JSON structure includes metadata and results array. (3) Compliance count matches sum of compliant/non_compliant results. (4) All required fields are present. (5) File is valid JSON.
- Verification: deterministic (file exists, JSON parses, fields present)

---

## REQ-64: Audit results feed into effectiveness pipeline

Audit results (compliance/non-compliance) are fed into the effectiveness evaluation pipeline as an additional outcome signal. Non-compliance with a surfaced memory lowers its effectiveness score in future evaluations.

- Traces to: UC-19 (integration with effectiveness)
- AC: (1) After audit report is written, audit results are parsed. (2) For each non-compliant instruction, look up its memory ID in the effectiveness data. (3) Add a negative outcome signal to that memory's effectiveness history. (4) Next `engram evaluate` run includes this signal in aggregation. (5) Non-compliance lowers effectiveness score (e.g., reduces follow rate or adds penalty).
- Verification: deterministic (outcome signal recording, effectiveness aggregation)

---

## DES-24: Effectiveness signal injection for audit results

In `internal/evaluate/evaluate.go`, add a function `InjectAuditResults(auditReport)` that:
1. Parses audit report JSON
2. For each non_compliant result, looks up memory ID in effectiveness registry
3. Adds a negative outcome signal (outcome_type = "audit_non_compliance", timestamp = audit timestamp)
4. Saves updated effectiveness data

This function is called by the audit command before the audit report is written.

- Traces to: REQ-64 (effectiveness integration)

---

## REQ-65: No graceful degradation on API token failure

If the Haiku API call fails due to missing or invalid API token, the audit phase emits an error to stderr and skips the audit (no report written). The error message is non-fatal; other Stop hook phases continue.

- Traces to: UC-19 (error handling)
- AC: (1) Audit command detects missing/invalid token at runtime. (2) Error is logged to stderr: "audit: API token missing or invalid, skipping audit". (3) Exit code is 1 for the audit command only (fire-and-forget pattern in hook script ignores exit code). (4) No audit report is written. (5) Other hook phases continue.
- Verification: deterministic (error condition handling)

---

## UC-18: PostToolUse Proactive Reminders — Requirements

---

## REQ-66: PostToolUse hook registration

A PostToolUse hook is registered in `hooks/hooks.json` that fires after Write and Edit tool calls. The hook invokes `engram remind --data-dir <path>` with the tool call details (file path, tool name) passed via stdin JSON.

- Traces to: UC-18 (PostToolUse hook)
- AC: (1) hooks.json includes PostToolUse event entry. (2) Hook fires after Write and Edit tool calls. (3) Hook passes tool call JSON (including file_path) via stdin. (4) Hook invokes engram remind subcommand.
- Verification: deterministic (hook registration, stdin JSON schema)

---

## REQ-67: Pattern-based trigger configuration

A configuration file maps file glob patterns to instruction sets. Format: TOML file at `<data-dir>/reminders.toml` with entries like `"*.go" = ["go-conventions", "targ-not-go-test"]`. When a file path matches a glob, the associated instruction set IDs are used to source reminders.

- Traces to: UC-18 (pattern-based triggers)
- AC: (1) Config file is TOML at `<data-dir>/reminders.toml`. (2) Keys are glob patterns, values are arrays of instruction set IDs. (3) File path from tool call is matched against globs in order. (4) First matching glob's instruction IDs are used. (5) No match → no reminder emitted.
- Verification: deterministic (glob matching, config parsing)

---

## DES-26: Reminder configuration format

```toml
# <data-dir>/reminders.toml
["*.go"]
instructions = ["go-conventions", "targ-build-system"]

["*.md"]
instructions = ["markdown-style"]

["**/skills/**"]
instructions = ["pressure-test-reminder"]
```

Each key is a glob pattern. Instructions reference memory titles, CLAUDE.md section headers, or rule file names. Instruction set IDs are resolved at runtime against available sources.

- Traces to: REQ-67 (pattern configuration)

---

## REQ-68: Reminder sourcing from instruction registry

When a file pattern matches, the system resolves instruction set IDs against: (1) engram memories by title/keywords, (2) CLAUDE.md entries by section header, (3) rules files by filename. The highest-effectiveness instruction from the matched set is selected as the reminder.

- Traces to: UC-18 (reminder sourcing)
- AC: (1) Instruction IDs resolve against memories, CLAUDE.md, and rules. (2) Resolution is best-effort (unresolved IDs are skipped). (3) Highest effectiveness instruction is selected. (4) If no instructions resolve, no reminder emitted.
- Verification: deterministic (ID resolution, effectiveness comparison)

---

## REQ-69: Budget cap per reminder

Each reminder is capped at 100 tokens (using UC-17's estimateTokens formula). Single targeted reminder per invocation — not a dump of all matching instructions.

- Traces to: UC-18 (budget cap), UC-17 (token estimation)
- AC: (1) Reminder text ≤100 tokens (estimateTokens). (2) Only one reminder per PostToolUse invocation. (3) If selected instruction exceeds 100 tokens, truncate or summarize.
- Verification: deterministic (token counting)

---

## DES-27: Reminder output format

The PostToolUse hook outputs a system reminder in the format:

```json
{"hookSpecificOutput": {"additionalContext": "[engram] Reminder: <instruction text>"}}
```

The reminder is concise (≤100 tokens), targeted to the file just modified, and prefixed with `[engram] Reminder:`.

- Traces to: REQ-69 (budget cap), REQ-66 (hook output)

---

## REQ-70: Suppression logic

If the transcript shows the model already performed the required action before the reminder fires, the reminder is suppressed. Suppression check reads recent transcript context (last ~500 tokens) and looks for evidence of compliance.

- Traces to: UC-18 (suppression logic)
- AC: (1) Before emitting reminder, read recent transcript (~500 tokens). (2) Check if the instruction's principle or anti-pattern was already addressed. (3) If complied → suppress (emit empty output). (4) If not complied or uncertain → emit reminder. (5) Suppression is conservative: emit reminder when uncertain.
- Verification: deterministic (string matching on transcript)

---

## DES-28: Suppression detection approach

Suppression uses simple keyword matching against the recent transcript excerpt. For each instruction, check if the instruction's `principle` text (or key phrases from it) appears in the transcript. If found → already complied → suppress. No LLM call for suppression (performance-critical path).

- Traces to: REQ-70 (suppression logic)

---

## REQ-71: Effectiveness tracking for reminders

Reminder effectiveness is tracked: did the model comply after the reminder? The PostToolUse surfacing event is logged (same as UC-2 surfacing log), and the next tool call is checked for compliance. Results feed into the evaluation pipeline.

- Traces to: UC-18 (effectiveness tracking)
- AC: (1) Each reminder emission is logged to the surfacing log with hook="PostToolUse", memory_id, timestamp. (2) Evaluation pipeline picks up PostToolUse surfacing events. (3) Compliance signal comes from subsequent tool calls in the same session.
- Verification: deterministic (log format, pipeline integration)

---

## UC-20: Instruction Quality, Deduplication & Gap Analysis — Requirements

---

## REQ-72: Memory-only instruction scanning

The system scans memory entries only: `<data-dir>/memories/*.toml`. Each instruction is extracted as a structured item with: source, location, text, and effectiveness data (if available). CLAUDE.md, rules, and skill sources are not scanned (cross-source scanning moves to surface pipeline P4-full).

- Traces to: UC-20 (memory scanning)
- AC: (1) Memories are loaded from `<data-dir>/memories/`. (2) No CLAUDE.md, rules, or skill sources scanned. (3) Each item has source type "memory", file path, and text content. (4) Effectiveness data joined from evaluation pipeline where available.
- Verification: deterministic (file scanning, structured extraction)

---

## REQ-73: Memory deduplication detection

Compare memory entries to find semantic duplicates. Two memories are considered duplicates if they share >80% keyword overlap. Report both paths and overlap score. No salience hierarchy needed (all entries are memory type).

- Traces to: UC-20 (deduplication)
- AC: (1) Pairwise comparison of all memory items. (2) Duplicate detection by keyword overlap (>80%). (3) Output includes both memory paths and overlap score. (4) No source-salience recommendation (memory-only, cross-source dedup deferred to P4-full).
- Verification: deterministic (keyword overlap calculation)

---

## REQ-74: Quality diagnosis via LLM

For low-effectiveness instructions (bottom 20% by effectiveness score), invoke Haiku to diagnose root cause: too abstract, framing mismatch, missing trigger conditions, too narrow, or too verbose.

- Traces to: UC-20 (quality diagnosis)
- AC: (1) Select instructions in bottom 20% by effectiveness. (2) Single Haiku call per instruction with content + effectiveness stats. (3) Diagnosis JSON: `{diagnosis: string, root_cause: string, suggestion: string}`. (4) Invalid responses logged and skipped.
- Verification: deterministic (LLM output parsing)

---

## DES-29: Quality diagnosis prompt structure

System prompt for Haiku:

```
You are diagnosing why an instruction is ineffective. Common root causes:
- Too abstract: lacks specific trigger conditions
- Framing mismatch: positive instruction for negative pattern (or vice versa)
- Missing trigger: instruction doesn't specify when it applies
- Too narrow: applies to rare situations
- Too verbose: key point buried in text

Given the instruction and its effectiveness data, diagnose the root cause
and suggest a concrete improvement.

Output JSON: {"diagnosis": "...", "root_cause": "...", "suggestion": "..."}
```

- Traces to: REQ-74 (quality diagnosis)

---

## REQ-75: Refinement proposals

Generate rewritten versions of diagnosed memory instructions in maintain-compatible format (same as UC-16).

- Traces to: UC-20 (refinement proposals)
- AC: (1) Each diagnosed memory gets a rewrite proposal. (2) Proposals are maintain-compatible: JSON with path, action, root_cause, suggestion. (3) Proposal includes rationale explaining what changed and why.
- Verification: deterministic (format validation)

---

## REQ-76: Gap analysis

Compare instruction anti-patterns against observed tool actions in evaluation data to find common violation patterns with no corresponding instruction. Report as gap candidates.

- Traces to: UC-20 (gap analysis)
- AC: (1) Load evaluation data with contradicted outcomes. (2) Extract patterns from contradictions (file types, tool names, common mistakes). (3) Cross-reference against existing instructions. (4) Patterns not covered by any instruction → gap candidates. (5) Output: list of gap candidates with evidence (violation count, example).
- Verification: deterministic (pattern extraction, cross-reference)

---

## REQ-77: ~~Skill decomposition~~ — REMOVED (S6 simplification)

**Removed in S6 (Phase A-1).** Skill decomposition required scanning skill files, which are no longer in scope for the audit. Per-skill-line effectiveness analysis will be re-introduced when skill sources re-enter the audit pipeline (P4-full or later).

- Traces to: UC-20 (removed)
- Status: unsatisfiable — skills not scanned in memory-only audit

---

## REQ-78: CLI command `engram instruct audit`

New subcommand: `engram instruct audit --data-dir <path>`. Outputs a JSON report with: duplicates, quality diagnoses, refinement proposals, and gap analysis. (Skill decomposition removed in S6.)

- Traces to: UC-20 (CLI command)
- AC: (1) Subcommand `instruct audit` registered. (2) Output is JSON with sections: duplicates, diagnoses, proposals, gaps. (3) Exit 0 always. (4) Empty sections are empty arrays.
- Verification: deterministic (CLI registration, JSON output)

---

## REQ-79: No graceful degradation on API failure

If no API token, skip LLM-dependent steps (quality diagnosis, refinement proposals). Deduplication and gap analysis still run. Output JSON includes skipped sections with a `skipped_reason` field.

- Traces to: UC-20 (error handling)
- AC: (1) Missing API token → skip REQ-74 and REQ-75. (2) REQ-72, REQ-73, REQ-76 still run. (3) Skipped sections include `{"skipped_reason": "no API token"}`. (4) Exit 0 regardless.
- Verification: deterministic (error condition handling)

---

## UC-21: Enforcement Escalation Ladder — Requirements

---

## REQ-80: Three escalation levels

Each leech memory has an escalation level: (1) advisory — surfaced as system reminder, (2) emphasized_advisory — surfaced with urgency markers, (3) reminder — targeted PostToolUse injection after relevant file edits (UC-18). Beyond `reminder`, a graduation signal (UC-28) is emitted recommending promotion to a higher-salience enforcement mechanism.

- Traces to: UC-21 (escalation levels)
- AC: (1) Three named levels in order: advisory, emphasized_advisory, reminder. (2) Each level has a defined enforcement mechanism. (3) Default level for all memories is "advisory". (4) Level stored in memory TOML file. (5) Levels are ordinal (can compare for escalation/de-escalation).
- Verification: deterministic (enum validation)

---

## DES-30: Escalation level TOML schema

Memory TOML files gain an optional `escalation_level` field:

```toml
escalation_level = "advisory"  # default
escalation_history = [
  { level = "advisory", since = "2026-01-01T00:00:00Z", effectiveness = 0.15 },
  { level = "emphasized_advisory", since = "2026-02-01T00:00:00Z", effectiveness = 0.25 }
]
```

- Traces to: REQ-80 (escalation levels), REQ-84 (TOML storage)

---

## REQ-81: Escalation proposals in maintain output

When `engram maintain` detects a leech memory, it generates an escalation proposal alongside the existing content-rewrite proposal. The proposal includes: current level, proposed level, rationale, and predicted effectiveness improvement based on data from other memories at the proposed level.

- Traces to: UC-21 (escalation proposals)
- AC: (1) Maintain output includes escalation proposals for leech memories. (2) Proposal has: memory_path, current_level, proposed_level, rationale, predicted_impact. (3) Predicted impact based on average effectiveness delta for memories that previously escalated to the proposed level. (4) If no historical data, predicted impact = "unknown".
- Verification: deterministic (proposal format, effectiveness lookup)

---

## DES-31: Escalation proposal format

```json
{
  "memory_path": "memories/example.toml",
  "proposal_type": "escalate",
  "current_level": "advisory",
  "proposed_level": "emphasized_advisory",
  "rationale": "Surfaced 20 times, followed 15%. Content rewrite attempted, no improvement.",
  "predicted_impact": "+10% follow rate based on 3 similar memories"
}
```

- Traces to: REQ-81 (proposal format)

---

## REQ-82: De-escalation detection

If a memory at an elevated enforcement level shows increasing contradictions (compliance rate drops after escalation), propose de-escalation back one level. De-escalation is proposed when post-escalation effectiveness < pre-escalation effectiveness for ≥3 evaluation cycles.

- Traces to: UC-21 (de-escalation)
- AC: (1) Track pre/post escalation effectiveness from escalation_history. (2) If post < pre for ≥3 cycles → de-escalation proposal. (3) Proposal includes evidence (before/after effectiveness values). (4) De-escalation drops one level (not to bottom).
- Verification: deterministic (effectiveness comparison, cycle counting)

---

## REQ-83: ~~REMOVED (S2)~~ Dimension routing before escalation

> **Removed in Phase A-1 (S2).** Engram does not route to automation, rule files, or CLAUDE.md. It escalates within its advisory range (advisory → emphasized_advisory → reminder) and emits graduation signals (UC-28) when that range is exhausted. Mechanical pattern detection and the `route_automation` proposal type have been removed from `internal/maintain/escalation.go`.

- Status: unsatisfiable — requirement removed from scope

---

## REQ-84: Escalation level stored per memory in TOML

The `escalation_level` field is written to each memory's TOML file when escalation or de-escalation is confirmed. The `escalation_history` array tracks all level changes with timestamps and effectiveness at time of change.

- Traces to: UC-21 (tracking)
- AC: (1) Field name is `escalation_level`. (2) Default value is "advisory" (omitted when default). (3) History is append-only array. (4) Each entry has: level, since (ISO 8601), effectiveness (float). (5) TOML file remains valid and hand-editable after update.
- Verification: deterministic (TOML write, field validation)

---

## REQ-85: User confirmation for each escalation step

Every escalation and de-escalation requires explicit user confirmation before the TOML file is updated. The proposal is presented; the user confirms or skips. Skipped proposals are logged but not acted on.

- Traces to: UC-21 (user confirmation)
- AC: (1) Proposals are presented in maintain output (JSON). (2) User confirms via maintain command interaction. (3) Only confirmed proposals update TOML. (4) Skipped proposals logged to maintain log.
- Verification: deterministic (confirmation tracking)

---

## UC-22: Mechanical Instruction Extraction — Requirements

---

## REQ-86: Pattern recognition for mechanical instructions

Analyze leech/noise memories for mechanical patterns: "always X before Y", "never X when Z", "format as...", naming conventions, ordering rules. Classification is deterministic (keyword-based, no LLM).

- Traces to: UC-22 (pattern recognition)
- AC: (1) Scan memory content for mechanical keywords: "always", "never", "before", "after", "format", "name", "convention". (2) Score each memory for mechanical-ness (count of mechanical patterns). (3) Score ≥2 → mechanical candidate. (4) Output: list of candidates with patterns found.
- Verification: deterministic (keyword counting)

---

## REQ-87: LLM generator for automation

For mechanical candidates, invoke Haiku to generate deterministic automation: shell scripts, pre-commit hooks, or rule definitions. Generated code must be self-contained and testable.

- Traces to: UC-22 (generator)
- AC: (1) Haiku call with memory content + instruction type. (2) Output: JSON with automation_type (script/hook/rule), code, description, test_command. (3) Generated code is syntactically valid (shell/Go). (4) Invalid LLM responses are logged and skipped.
- Verification: deterministic (LLM output parsing, syntax check)

---

## DES-32: Automation output format

```json
{
  "memory_path": "memories/example.toml",
  "automation_type": "pre_commit_hook",
  "code": "#!/bin/bash\n# Verify X before Y\n...",
  "description": "Enforces X-before-Y ordering in commits",
  "test_command": "echo 'test input' | ./hooks/pre-commit-check.sh",
  "install_path": ".git/hooks/pre-commit-check.sh"
}
```

- Traces to: REQ-87 (generator output)

---

## REQ-88: Verification of generated automation

Generated automation must pass a test (dry-run) before the instruction is retired. The test_command from the LLM output is executed with sample input. If it fails, the automation is not installed and the proposal is rejected.

- Traces to: UC-22 (verification)
- AC: (1) Execute test_command in a sandboxed environment. (2) Exit 0 → pass, non-zero → fail. (3) Pass → automation is installed to install_path. (4) Fail → automation rejected, error logged, memory unchanged.
- Verification: deterministic (exit code check)

---

## REQ-89: Instruction retirement with retired_by field

Once automation is verified and user confirms, the memory gains a `retired_by` field pointing to the automation file path. Retired memories are no longer surfaced by UC-2.

- Traces to: UC-22 (retirement)
- AC: (1) TOML file updated with `retired_by = "<automation_path>"`. (2) `retired_at` timestamp added. (3) UC-2 surface logic skips memories with non-empty `retired_by`. (4) Memory file is NOT deleted (preserved for audit trail).
- Verification: deterministic (TOML field check, surfacing filter)

---

## REQ-90: CLI command `engram automate` *(removed — Phase A-1/S1)*

**Status:** Unsatisfiable. UC-22 removed. `engram automate` subcommand deleted.

- Traces to: UC-22 (removed)

---

## REQ-91: No graceful degradation for automate *(removed — Phase A-1/S1)*

**Status:** Unsatisfiable. UC-22 removed. `engram automate` subcommand deleted.

- Traces to: UC-22 (removed)

---

## REQ-55: Registry bounded growth — one line per instruction

The registry `instruction-registry.jsonl` contains exactly one line per registered instruction. No unbounded logs, no append-only event files. Each update to an instruction (surfaced, evaluated, merged) rewrites the corresponding line atomically.

- Traces to: UC-23 (data model constraint)
- AC: (1) Each instruction has a unique ID (source_type:source_path:item). (2) One line per ID in JSONL. (3) Updates overwrite the line, not append. (4) File size is O(num_instructions), not O(events).
- Verification: structural (line count = unique IDs)

---

## REQ-56: Registry tracks six instruction source types

Registry entries tag source_type with one of: `claude-md`, `memory-md`, `memory`, `rule`, `skill`, `hook`. Each source type has a defined salience level in the hierarchy (deterministic code > claude-md > rule > memory-md > skill > memory).

- Traces to: UC-23 (instruction taxonomy)
- AC: (1) source_type field is required, must match enum. (2) Salience hierarchy is deterministic — no tie-breaking. (3) Quadrant classification logic respects salience (always-loaded sources have binary quadrant: Working or Leech).
- Verification: structural (enum validation, salience ordering)

---

## REQ-57: Registry computes effectiveness as quantitative ratio

Effectiveness = followed / (followed + contradicted + ignored). Null when insufficient evaluations. Used for quadrant assignment and escalation decisions.

- Traces to: UC-23 (effectiveness signal)
- AC: (1) Three counters: followed, contradicted, ignored. (2) Effectiveness computed on read (not stored). (3) Denominator is sum of all three. (4) Null when denominator < N evaluations (threshold).
- Verification: deterministic (arithmetic)

---

## REQ-58: Registry computes frecency as frequency × recency blend

Frecency weights surfaced_count (frequency) and time-since-last-surfaced (recency) using exponential decay. Higher frecency = rank higher in surfacing.

- Traces to: UC-23 (frecency signal, relates to #60)
- AC: (1) surfaced_count is counter (incremented on each surface event). (2) last_surfaced is timestamp. (3) Decay function has fixed half-life (e.g., 7 days). (4) Computed on read.
- Verification: deterministic (time-based + arithmetic)

---

## REQ-59: Registry stores content_hash to detect instruction changes

Each instruction's content is hashed (SHA256 or similar) and stored. If content changes (e.g., a memory TOML is edited), content_hash changes, triggering re-evaluation.

- Traces to: UC-23 (change detection)
- AC: (1) content_hash is computed from instruction text. (2) Hash updated on register and on rewrite proposal acceptance. (3) Hash mismatch detected during evaluation pipeline.
- Verification: deterministic (hash function)

---

## REQ-60: Registry tracks absorbed history — merged duplicates preserve effectiveness

When `engram registry merge --source <id> --target <id>` runs, the source instruction's counters (surfaced_count, followed/contradicted/ignored) are appended to the target's `absorbed` array as a timestamped record. Source is then deleted.

- Traces to: UC-23 (merge operation + history preservation)
- AC: (1) Merge is idempotent: running twice with same source/target is safe. (2) absorbed array preserves surfaced_count and counters. (3) merged_at timestamp is recorded. (4) Source file deleted after merge. (5) Content_hash of absorbed entries preserved for retrospective analysis.
- Verification: deterministic (array structure, deletion)

---

## REQ-61: Surfacing event atomically increments surfaced_count and updates last_surfaced

Each time the surfacing hook surfaces an instruction, it increments the registry's surfaced_count for that instruction ID and updates last_surfaced to current timestamp. Both updates happen in the same write.

- Traces to: UC-23 (surfacing tracking)
- AC: (1) Hook calls Registry.RecordSurfacing(id). (2) Increments surfaced_count. (3) Sets last_surfaced to current time. (4) Single atomic JSONL line rewrite. (5) No partial updates.
- Verification: integration (hook + registry I/O)

---

## REQ-62: Evaluation event increments followed/contradicted/ignored counters

During evaluation, the system assesses whether the user's behavior complied with each active instruction. For each instruction, one counter increments: followed (user complied), contradicted (user did opposite), ignored (no evidence of awareness).

- Traces to: UC-23 (evaluation tracking)
- AC: (1) Evaluation runs at session end or PreCompact. (2) For each active instruction, exactly one of the three counters increments. (3) Counter update writes to registry atomically. (4) Counters start at 0.
- Verification: integration (evaluate + registry I/O)

---

## REQ-63: Registry merge absorbs all counters into target's absorbed field

When merging source → target, all of source's evaluation counters (followed, contradicted, ignored) and surfacing history (surfaced_count, last_surfaced, content_hash, registered_at) become a single entry in target's `absorbed` array. No counter loss.

- Traces to: UC-23 (merge semantics)
- AC: (1) absorbed array entry has: from (source id), surfaced_count, evaluations object, merged_at timestamp, content_hash. (2) Multiple entries allowed (multiple duplicates can be absorbed). (3) Absorbed entries are readable for escalation engine decisions.
- Verification: structural (JSON schema)

---

## REQ-64: Registry supports concurrent writes from multiple hooks

Multiple hook instances may call Registry.RecordSurfacing or Registry.RecordEvaluation concurrently. Registry write must be safe: no data loss, no partial updates, no corruption.

- Traces to: UC-23 (concurrency + data safety)
- AC: (1) JSONL read-all-on-load, write-full-file strategy is safe for < 10K instructions. (2) No lock files needed (file I/O atomicity sufficient). (3) Worst case: two concurrent writes both see same stale state, one overwrites the other (acceptable for small frequency deltas).
- Verification: integration (concurrent hook calls)

---

## REQ-65: Registry backfill migrates all data from old stores without loss

`engram registry init` reads surfacing-log.jsonl, creation-log.jsonl, evaluations/*.jsonl, and memory TOML metadata fields. For each memory file, creates one registry entry with all aggregated data.

- Traces to: UC-23 (migration / Phase 1)
- AC: (1) surfacing-log.jsonl data aggregated: sum surfaced_count per memory, take max last_surfaced. (2) creation-log.jsonl: set registered_at. (3) evaluations/*.jsonl: sum counters per memory. (4) Memory TOML metadata (surfaced_count, last_surfaced, surfacing_contexts): migrated. (5) No data discarded.
- Verification: integration (read old stores, verify registry counts match)

---

## REQ-66: Backfill handles retirement mapping — covers for retired duplicates

During backfill, if a memory has retired_by set, the backfill identifies the covering instruction (the one with matching title/domain in surviving memories or CLAUDE.md entries). The retired memory's counters are recorded as absorbed history in the covering instruction's entry.

- Traces to: UC-23 (migration with attribution)
- AC: (1) Memory with retired_by="..." is matched to covering instruction by ID or semantic lookup. (2) Retired memory's counters appended to covering instruction's absorbed array. (3) No standalone registry entry created for retired memories. (4) Covering instruction's absorbed field documents the merge.
- Verification: integration (semantic matching, retired_by mapping)

---

## REQ-67: Quadrant classification works across all six source types

`engram review` reads the registry and classifies all instructions (not just memories) into quadrants: Working (high surfacing + high effectiveness), Leech (high surfacing + low effectiveness), HiddenGem (low surfacing + high effectiveness), Noise (low surfacing + low effectiveness). Always-loaded sources (claude-md, memory-md) have binary quadrant: Working or Leech.

- Traces to: UC-23 (cross-source classification)
- AC: (1) Surfacing threshold and effectiveness threshold are configurable but have sensible defaults. (2) Classification is deterministic given thresholds. (3) All six source types are classified. (4) CLAUDE.md entry can be Leech, triggering rewrite proposal (not just memories).
- Verification: deterministic (thresholds + math)

---

## REQ-68: DI boundary — Registry interface in internal/, JSONL I/O at edges

The registry abstraction (interface) lives in `internal/registry/`. Concrete JSONL implementation lives in cli.go or top-level wiring. Tests use mock Registry interface.

- Traces to: UC-23 (DI everywhere principle) + UC-23 constraint (#5: pure Go, no CGO)
- AC: (1) Registry interface defined in internal/ with methods: Register, RecordSurfacing, RecordEvaluation, Merge, Remove, List, Get. (2) Concrete JSONL-based implementation in cli.go (not internal/). (3) No raw file I/O in internal/. (4) Tests inject mock Registry.
- Verification: structural (interface + no os.* calls in internal/)

---

## DES-26: User command `engram registry init` triggers backfill

New CLI subcommand: `engram registry init`. Reads surfacing-log.jsonl, creation-log.jsonl, evaluations/*.jsonl, memory TOML files, and produces instruction-registry.jsonl. Outputs summary: number of entries created, number of duplicates absorbed, any warnings (unmatched retired_by references).

- Traces to: UC-23 (backfill interaction)
- AC: (1) Subcommand registered in CLI. (2) Optional --dry-run flag shows what would be written without writing. (3) Output is human-readable summary + JSON detail. (4) Exit code 0 on success, non-zero on error (failed reads, schema violations).
- Verification: integration (CLI + Registry.Register calls)

---

## DES-27: User command `engram review` reads registry for quadrant classification

Enhanced `engram review` command (if exists) or new subcommand: `engram review --format [table|json]`. Reads instruction-registry.jsonl, classifies all entries by quadrant, outputs summary grouped by source type and quadrant.

- Traces to: UC-23 (classification interaction)
- AC: (1) Output includes quadrant, source_type, instruction ID, title, effectiveness, surfaced_count. (2) Grouped by: source_type (primary), quadrant (secondary). (3) CLAUDE.md leeches flagged prominently (rare, high-salience source). (4) JSON output is array of classification objects.
- Verification: integration (read registry, classify, format output)

---

## DES-28: User command `engram registry merge` absorbs duplicates

New CLI subcommand: `engram registry merge --source <id> --target <id>`. Absorbs all counters from source into target's absorbed field. Deletes source entry and source file (if applicable, e.g., memory TOML). Outputs confirmation.

- Traces to: UC-23 (merge interaction)
- AC: (1) Subcommand registered. (2) --source and --target required. (3) Source can be any instruction type; target must be a surviving instruction (same or broader domain). (4) Merge is idempotent. (5) Output includes absorbed counters for verification. (6) Exit 0 always (merge either succeeds or fails verbosely).
- Verification: integration (Registry.Merge calls, file deletion if needed)

---

## DES-29: System auto-registers new instructions and auto-updates registry

When a new memory is created (via learn pipeline), it is auto-registered in the registry (Registry.Register call). When surfacing or evaluation occurs, registry is updated atomically without user action. No manual registration steps for memories.

- Traces to: UC-23 (auto-integration into pipelines)
- AC: (1) Learn pipeline calls Registry.Register(id, content_hash, source_type, title). (2) Surfacing hook calls Registry.RecordSurfacing(id). (3) Evaluate hook calls Registry.RecordEvaluation(id, followed|contradicted|ignored). (4) All calls are fire-and-forget: failures don't crash hooks (ARCH-6). (5) Registry updates are logged but don't interrupt instruction delivery.
- Verification: integration (hook + registry I/O)

---

## UC-4: Skill Generation Requirements

---

### REQ-92: Promotion threshold computation

When computing promotion candidates, compare surfacing cost (memory loaded via keyword matching on every prompt) against skill slot cost (loaded only when context-similar). A memory is a promotion candidate when its registry surfaced_count exceeds a configurable threshold (default: 50 surfacings). The threshold is a CLI flag, not hardcoded.

- Traces to: UC-4 (candidate detection)
- AC: (1) Registry queried for memories above threshold. (2) Threshold configurable via `--threshold` flag. (3) Only memories (source_type=memory) are candidates, not other instruction types. (4) Memories with Insufficient quadrant classification are excluded.
- Verification: deterministic (threshold comparison)

---

### REQ-93: Skill file generation from memory content

Given a memory TOML with title, content, principle, anti_pattern, keywords, and concepts, an LLM call (claude-haiku-4-5-20251001) generates a valid Claude Code skill file. The skill's triggering description is derived from keywords/concepts. The skill body contains the memory's principle and anti_pattern as actionable guidance.

- Traces to: UC-4 (skill file generation)
- AC: (1) Generated skill file is valid YAML frontmatter + markdown body. (2) Skill description enables context-similarity matching. (3) Memory content faithfully represented (no information loss on principle/anti_pattern). (4) No API token → skip generation, return error.
- Verification: integration (LLM call, file format validation)

---

### REQ-94: Plugin registration of generated skill

After skill file generation, write the file to the plugin's skills directory (`skills/<slug>.md`). The plugin manifest (`plugin.json`) auto-discovers skills from the directory — no manifest update needed.

- Traces to: UC-4 (plugin registration)
- AC: (1) Skill file written to skills/ directory. (2) File name is slugified memory title. (3) File permissions are standard (0644). (4) Existing file with same name → error, not overwrite.
- Verification: deterministic (file existence check)

---

### REQ-95: Source retirement via registry merge

After successful promotion and user confirmation, the source memory's registry entry is merged into the new skill's entry (preserving all effectiveness counters in the absorbed array). The source memory TOML file is deleted.

- Traces to: UC-4 (source retirement)
- AC: (1) Registry.Merge called with source=memory ID, target=skill ID. (2) Source memory TOML deleted from filesystem. (3) Source registry entry removed. (4) Target's absorbed array contains source's counters. (5) If merge fails, no deletion occurs (atomic: merge before delete).
- Verification: integration (registry merge + file deletion)

---

### REQ-96: User confirmation before promotion

Promotion is never automatic. The system presents a preview (source memory summary, generated skill preview, effectiveness data) and requires explicit user confirmation before executing.

- Traces to: UC-4 (user confirmation)
- AC: (1) Preview displayed before any file writes. (2) User must type "y" or "yes" to confirm. (3) Any other input aborts. (4) `--yes` flag skips confirmation (for scripting). (5) Abort produces no side effects.
- Verification: deterministic (input parsing)

---

### DES-33: CLI interaction for `engram promote --to-skill`

New CLI subcommand: `engram promote --to-skill --data-dir <dir> [--threshold N] [--yes]`.

Flow:
1. Query registry for promotion candidates (memories above threshold).
2. Display candidate list with surfacing count, effectiveness score, quadrant.
3. User selects a candidate (by number or ID).
4. LLM generates skill file; preview displayed.
5. User confirms → skill written, registry merged, memory deleted.
6. Output: confirmation with new skill path.

No candidates → "No memories meet the promotion threshold." Exit 0 always.

- Traces to: UC-4 (CLI interaction)
- AC: (1) Subcommand registered. (2) --data-dir required. (3) Candidate list sorted by surfacing count descending. (4) Invalid selection → re-prompt. (5) Exit 0 always.
- Verification: integration (CLI flow)

---

### DES-34: Generated skill file format

```markdown
---
description: "Use when <context derived from keywords/concepts>"
---

# <Memory Title>

<Memory principle as actionable guidance>

## What to avoid

<Memory anti_pattern, if present>

## Context

<Memory content — full original text>
```

Skills without anti_pattern omit the "What to avoid" section.

- Traces to: UC-4 (skill file format)
- AC: (1) YAML frontmatter with description field. (2) Markdown body with title, principle, anti_pattern (optional), content. (3) Description enables context-similarity triggering.
- Verification: deterministic (format check)

---

## UC-5: CLAUDE.md Management Requirements

---

### REQ-97: Promotion candidate detection for CLAUDE.md

Query the registry for skills in the Working quadrant with surfacing frequency above a configurable threshold (default: 100). These are universally useful skills that would benefit from always-loaded status in CLAUDE.md.

- Traces to: UC-5 (promotion detection)
- AC: (1) Only skills (source_type=skill) are candidates. (2) Must be in Working quadrant. (3) Threshold configurable via `--threshold` flag. (4) Candidates sorted by effectiveness descending.
- Verification: deterministic (registry query + quadrant check)

---

### REQ-98: CLAUDE.md entry generation from skill

Given a skill file, an LLM call generates a concise CLAUDE.md entry matching the project's existing CLAUDE.md style. The entry preserves the skill's core guidance in a format appropriate for always-loaded context.

- Traces to: UC-5 (CLAUDE.md generation)
- AC: (1) Generated entry matches existing CLAUDE.md style (bullet points, concise). (2) No information loss on core principle. (3) Entry includes a comment indicating source skill for traceability. (4) No API token → skip generation, return error.
- Verification: integration (LLM call, style matching)

---

### REQ-99: Demotion candidate detection

Query the registry for CLAUDE.md entries (source_type=claude-md) in the Leech quadrant — always loaded but rarely followed. These waste context budget and should be demoted to skills.

- Traces to: UC-5 (demotion detection)
- AC: (1) Only claude-md entries are candidates. (2) Must be in Leech quadrant (binary classification for always-loaded sources). (3) Candidates sorted by effectiveness ascending (worst first).
- Verification: deterministic (registry query + quadrant check)

---

### REQ-100: Demotion execution — CLAUDE.md entry to skill

Convert a CLAUDE.md entry into a skill file using the same generation logic as UC-4 (REQ-93). Remove the entry from CLAUDE.md. The CLAUDE.md file is edited in-place with the entry removed.

- Traces to: UC-5 (demotion execution)
- AC: (1) Skill file generated from CLAUDE.md entry content. (2) CLAUDE.md entry removed from file. (3) Registry merge: claude-md entry → new skill entry. (4) CLAUDE.md file written atomically (temp + rename).
- Verification: integration (file edit + registry merge)

---

### REQ-101: Registry merge on tier transitions

All tier transitions (promotion and demotion) preserve effectiveness history via registry merge. Source entry's counters are absorbed into target's absorbed array before source deletion.

- Traces to: UC-5 (history preservation)
- AC: (1) Same merge semantics as UC-23 REQ-63. (2) Absorbed array includes source_id, all counters, merge timestamp. (3) Idempotent — re-merge doesn't duplicate.
- Verification: deterministic (merge output check)

---

### REQ-102: User confirmation for CLAUDE.md modifications

All CLAUDE.md modifications (promotion and demotion) require explicit user confirmation. The system presents evidence (effectiveness data, quadrant, proposed change) before executing.

- Traces to: UC-5 (user confirmation)
- AC: (1) Diff preview shown before any file writes. (2) Explicit "y"/"yes" confirmation required. (3) `--yes` flag for scripting. (4) Abort → no side effects.
- Verification: deterministic (input parsing)

---

### DES-35: CLI interaction for CLAUDE.md management

Two subcommands:
- `engram promote --to-claude-md --data-dir <dir> [--threshold N] [--yes]` — promote skill to CLAUDE.md
- `engram demote --to-skill --data-dir <dir> [--yes]` — demote CLAUDE.md entry to skill

Both follow the same flow: detect candidates → display list → user selects → preview → confirm → execute.

- Traces to: UC-5 (CLI interaction)
- AC: (1) Both subcommands registered. (2) --data-dir required. (3) No candidates → informational message, exit 0. (4) Exit 0 always.
- Verification: integration (CLI flow)

---

### DES-36: Diff preview format for CLAUDE.md changes

```
[engram] Proposed CLAUDE.md change:

  Action: ADD entry (promoted from skill "use-targ-build")
  Evidence: Working quadrant, effectiveness 92%, surfaced 150 times

  + ## Build System
  + - Use `targ` for all build/test/check operations
  + - Never run `go test`, `go vet`, `go build` directly
  + <!-- promoted from skill:use-targ-build -->

  Confirm? [y/N]
```

Demotion shows removal diff with `-` prefix.

- Traces to: UC-5 (preview format)
- AC: (1) Action and evidence shown. (2) Diff uses +/- prefix for additions/removals. (3) Source traceability comment included.
- Verification: deterministic (format check)

---

## UC-24: Proposal Application Requirements

---

### REQ-103: Proposal ingestion from maintain output

Read JSON proposals from `engram maintain` output (piped or from file via `--proposals <path>`). Each proposal has: quadrant, action, target_path, proposed_change, evidence (effectiveness, surfacing_count).

- Traces to: UC-24 (proposal ingestion)
- AC: (1) Accepts JSON array from stdin or --proposals file. (2) Validates proposal schema. (3) Invalid proposals → skip with warning. (4) Empty proposals → "No proposals to apply." exit 0.
- Verification: deterministic (JSON parsing)

---

### REQ-104: Working staleness — content rewrite

For Working-quadrant proposals with action "update_content", an LLM call rewrites the memory TOML content to reflect current practices. The original content is preserved in the proposal for diff display.

- Traces to: UC-24 (Working staleness)
- AC: (1) LLM receives original content + staleness evidence. (2) Rewrite preserves memory structure (title, keywords, concepts unchanged unless explicitly part of the update). (3) TOML file written atomically. (4) No API token → skip, report "skipped: no token".
- Verification: integration (LLM call, TOML write)

---

### REQ-105: Leech rewrite — root-cause-informed fix

For Leech-quadrant proposals with action "rewrite", an LLM call rewrites the memory content informed by the root cause diagnosis (content quality, wrong keywords, enforcement gap). Different root causes produce different rewrite strategies.

- Traces to: UC-24 (Leech rewrite)
- AC: (1) Root cause from proposal determines rewrite strategy. (2) "content_quality" → rewrite principle/anti_pattern. (3) "wrong_keywords" → adjust keywords/concepts. (4) "enforcement_gap" → add anti_pattern or strengthen existing. (5) No API token → skip.
- Verification: integration (LLM call, strategy selection)

---

### REQ-106: HiddenGem keyword broadening

For HiddenGem-quadrant proposals with action "broaden_keywords", an LLM call suggests additional keywords and concepts based on contexts where the memory would have been relevant but wasn't triggered.

- Traces to: UC-24 (HiddenGem broadening)
- AC: (1) LLM receives current keywords + memory content. (2) Suggestions are additive (no keywords removed). (3) Updated keywords written to TOML. (4) No API token → skip.
- Verification: integration (LLM call, keyword addition)

---

### REQ-107: Noise removal — delete memory and registry entry

For Noise-quadrant proposals with action "remove", delete the memory TOML file and remove its registry entry. This is deterministic — no LLM needed.

- Traces to: UC-24 (Noise removal)
- AC: (1) Memory TOML file deleted. (2) Registry entry removed via Registry.Remove. (3) User confirmation required before deletion. (4) Missing file → skip with warning.
- Verification: deterministic (file deletion + registry removal)

---

### REQ-108: Registry update after proposal application

After each applied proposal: content rewrites update the registry entry's content_hash (via re-registration or explicit update). Deletions remove the entry. All updates are fire-and-forget per ARCH-6.

- Traces to: UC-24 (registry update)
- AC: (1) Rewrite → content_hash updated. (2) Deletion → entry removed. (3) Registry write failures logged but don't block. (4) Updated_at timestamp set.
- Verification: integration (registry I/O)

---

### REQ-109: User confirmation per proposal

Each proposal requires individual user confirmation. Display the proposed change as a diff (for rewrites) or deletion summary (for removals) with evidence.

- Traces to: UC-24 (user confirmation)
- AC: (1) Each proposal confirmed individually. (2) `--yes` flag confirms all. (3) Skip ("s") skips one proposal. (4) Abort ("q") stops all remaining. (5) Applied count reported at end.
- Verification: deterministic (input parsing)

---

### DES-37: CLI interaction for `engram maintain --apply`

Extended subcommand: `engram maintain --apply --data-dir <dir> [--proposals <path>] [--yes]`.

Flow:
1. Generate or read proposals (stdin/file).
2. Group by quadrant, display summary (N Working, N Leech, N HiddenGem, N Noise).
3. Walk proposals one by one: show diff/evidence → confirm → apply.
4. Report: "Applied N/M proposals (K skipped, J failed)."

- Traces to: UC-24 (CLI interaction)
- AC: (1) --apply flag activates application mode (without it, maintain behaves as before). (2) --data-dir required. (3) Summary before individual proposals. (4) Exit 0 always.
- Verification: integration (CLI flow)

---

### DES-38: Proposal display format

```
[engram] Proposal 3/8 — Leech rewrite

  Memory: use-targ-build-system.toml
  Quadrant: Leech (effectiveness: 23%, surfaced: 45 times)
  Root cause: content_quality

  - content = "Always use targ for testing"
  + content = "Use targ test, targ check, and targ build for all build operations. Never run raw go test, go vet, or go build — targ encodes project-specific flags and coverage thresholds."

  [a]pply / [s]kip / [q]uit:
```

- Traces to: UC-24 (display format)
- AC: (1) Proposal number and total shown. (2) Evidence includes quadrant, effectiveness, surfacing count. (3) Diff shown for rewrites. (4) Interactive prompt with a/s/q options.
- Verification: deterministic (format check)

---

## UC-25: Evaluate Strip Preprocessing Requirements

---

### REQ-110: Strip preprocessing in evaluate pipeline

The evaluate pipeline applies `sessionctx.Strip` to the transcript before sending it to the LLM for outcome evaluation. This removes tool results, base64 data, and truncated blocks — the same preprocessing used by learn (incremental) and context-update pipelines.

- Traces to: UC-25 (strip injection)
- AC: (1) Strip applied after reading transcript from stdin. (2) Same `sessionctx.Strip` function as learn and context-update. (3) Stripped content passed to Evaluator. (4) No additional configuration needed.
- Verification: deterministic (content comparison before/after strip)

---

### REQ-111: Empty post-strip transcript handling

If `sessionctx.Strip` produces an empty result (edge case: transcript is entirely tool results with no conversation), the evaluate pipeline skips the LLM evaluation call and returns early with no output.

- Traces to: UC-25 (empty handling)
- AC: (1) Empty post-strip transcript → no LLM call. (2) No error — graceful skip. (3) Exit 0. (4) Stderr message: "[engram] Evaluation skipped: no content after stripping."
- Verification: deterministic (empty input test)

---

### DES-39: StripFunc dependency injection

Strip function injected into the Evaluator as a `StripFunc func([]string) []string` option, consistent with the learn pipeline's pattern (`learn.NewIncrementalLearner` accepts `StripFunc`). Default is no-op (backward compatible). CLI wiring injects `sessionctx.Strip`.

- Traces to: UC-25 (DI pattern)
- AC: (1) Evaluator accepts `WithStripFunc` option. (2) Default strip is identity (no-op). (3) CLI wiring passes `sessionctx.Strip`. (4) Tests can inject custom strip functions.
- Verification: deterministic (DI wiring check)

---

## UC-26: First-Class Non-Memory Instruction Sources

---

### REQ-112: Source discovery at SessionStart

At SessionStart, the system scans all known instruction source locations to discover current sources:
- CLAUDE.md files: project CLAUDE.md (working directory), user global CLAUDE.md (`~/.claude/CLAUDE.md`)
- MEMORY.md: project memory file (`~/.claude/projects/<project>/memory/MEMORY.md`)
- Rules: all files in `.claude/rules/`
- Skills: all skill files in the plugin's skills directory

Source paths are provided as configuration (injected, not hardcoded). Missing directories or files are silently skipped (non-fatal).

- Traces to: UC-26 (auto-registration scan)
- AC: (1) All 4 non-memory source types are scanned. (2) Source paths are injected configuration. (3) Missing paths are silently skipped. (4) Each discovered source is passed to the appropriate extractor (UC-23 extractors). (5) Discovery runs at SessionStart before surfacing.
- Verification: deterministic (path scanning, extractor invocation)

---

### REQ-113: Auto-registration of discovered sources

For each discovered source, the system registers new entries and updates existing ones:
- **New source:** Extract entries using UC-23 extractors, call `Registry.Register` for each.
- **Changed source:** If an entry with the same ID exists but content hash differs, update the entry's `content_hash` and `updated_at` fields.
- **Unchanged source:** No action needed — entry already exists with correct hash.

Registration errors are fire-and-forget (ARCH-6 contract) — logged to stderr, never fail the hook.

- Traces to: UC-26 (auto-registration)
- AC: (1) New entries are registered with correct source type, path, title, content hash. (2) Changed entries get updated content hash and updated_at. (3) Unchanged entries are not modified. (4) Registration errors logged but don't fail the hook. (5) Duplicate ID errors (ErrDuplicateID) are expected for existing entries and handled gracefully.
- Verification: deterministic (registry state after registration)

---

### REQ-114: Stale entry pruning

During auto-registration, build the set of all currently-discoverable non-memory source IDs. Any registry entry whose source type is non-memory (`claude-md`, `memory-md`, `rule`, `skill`) and whose ID is not in the discovered set is removed via `Registry.Remove`.

Memory entries are never pruned by this mechanism — memory pruning is handled by UC-16 Noise removal with user confirmation.

- Traces to: UC-26 (stale pruning)
- AC: (1) Only non-memory entries are candidates for pruning. (2) Entry removed if its ID is not in the discovered set. (3) Memory entries are never removed. (4) Remove errors are fire-and-forget. (5) Pruning runs after registration (so newly discovered entries are not accidentally pruned).
- Verification: deterministic (registry state after pruning)

---

### REQ-115: Implicit surfacing for always-loaded sources

At SessionStart, after auto-registration, call `Registry.RecordSurfacing` for every always-loaded entry (source types: `claude-md`, `memory-md`, `rule`, `skill`). This increments `surfaced_count` and updates `last_surfaced` to reflect that these sources are loaded into every session by Claude Code.

This runs once per session, not per prompt. One surfacing increment per session accurately reflects the "loaded for the duration of the session" reality.

- Traces to: UC-26 (implicit surfacing tracking)
- AC: (1) RecordSurfacing called for every always-loaded entry. (2) Called once per session at SessionStart. (3) Fire-and-forget — errors logged but don't fail. (4) Only entries currently in the registry are surfaced (pruned entries excluded).
- Verification: deterministic (surfaced_count increment, last_surfaced update)

---

### REQ-116: Extended always-loaded source classification

The quadrant classifier (`registry.Classify`) treats `rule` and `skill` source types as always-loaded, alongside existing `claude-md` and `memory-md`. Always-loaded sources get binary classification: Working or Leech only (no Hidden Gem or Noise — they can't be "rarely surfaced").

- Traces to: UC-26 (skills and rules as always-surfaced)
- AC: (1) `alwaysLoadedSources` includes `"rule"` and `"skill"`. (2) Rules and skills get binary Working/Leech classification. (3) Existing claude-md/memory-md classification unchanged. (4) Memory sources still get 4-way classification.
- Verification: deterministic (quadrant assignment for each source type)

---

### REQ-117: Evaluate all surfaced sources at session end

The Stop hook's evaluation step (UC-15) evaluates all instructions that were surfaced during the session, including non-memory sources. The evaluator reads the surfacing log (which now includes always-loaded sources via REQ-115) and judges each surfaced instruction against the transcript.

No changes to the evaluation LLM prompt or outcome classification are needed — the evaluator already works with instruction IDs and content, regardless of source type.

- Traces to: UC-26 (evaluate all sources)
- AC: (1) Evaluation covers all instruction IDs in the surfacing log. (2) Non-memory instructions are evaluated with the same followed/contradicted/ignored outcomes. (3) Evaluation results update the registry entry for each source. (4) No API token → evaluation skipped for all sources (existing behavior).
- Verification: deterministic (evaluation log includes non-memory entries)

---

### REQ-118: Idempotent auto-registration

Running auto-registration twice in the same session produces the same registry state. This is important because SessionStart may re-fire (e.g., plugin reload) and the system must not double-count.

- Traces to: UC-26 (idempotency)
- AC: (1) Registering an already-registered entry with same content hash is a no-op. (2) Surfacing increment is tied to session start, not registration. (3) Pruning the same set twice removes nothing on the second pass. (4) No duplicate entries created.
- Verification: deterministic (registry state comparison after 1 vs 2 runs)

---

### DES-40: SessionStart auto-registration flow

The SessionStart hook invokes `engram surface` (existing behavior). The surface command is extended with an auto-registration phase that runs before memory surfacing:

1. **Discover** — Scan configured source paths for all non-memory sources
2. **Register** — Register new entries, update changed entries (REQ-113)
3. **Prune** — Remove stale non-memory entries (REQ-114)
4. **Record surfacing** — Increment surfaced_count for all always-loaded entries (REQ-115)
5. **Surface memories** — Existing BM25 surfacing behavior (unchanged)

Auto-registration runs synchronously before surfacing so that newly registered entries can participate in the first session's evaluation.

- Traces to: UC-26 (SessionStart flow)
- AC: (1) Auto-registration runs before memory surfacing. (2) Steps execute in order: discover → register → prune → record surfacing → surface memories. (3) Any step failure doesn't block subsequent steps. (4) Total time budget: auto-registration should complete in <100ms for typical source counts (<50 entries).
- Verification: integration (hook flow sequence)

---

## UC-27: Global Binary Installation

### REQ-119: Symlink creation after build

After a successful binary build in SessionStart, the system creates a symlink at `~/.local/bin/engram` pointing to the built binary at `~/.claude/engram/bin/engram`.

- Traces to: UC-27 (symlink creation)
- AC: (1) Symlink created pointing to correct target. (2) Only runs after successful build. (3) Target directory created if missing.
- Verification: deterministic (symlink target check)

---

### REQ-120: Idempotent symlink management

If the symlink already exists and points to the correct target, the operation is a no-op. If the symlink exists but points to a stale target, it is replaced.

- Traces to: UC-27 (idempotent)
- AC: (1) Existing correct symlink → no action. (2) Existing stale symlink → replaced. (3) Running twice produces same result.
- Verification: deterministic (idempotent state check)

---

### REQ-121: No-clobber for non-engram binaries

If `~/.local/bin/engram` exists and is a regular file (not a symlink to our binary), the system does NOT overwrite it. A warning is logged to stderr.

- Traces to: UC-27 (no clobber)
- AC: (1) Regular file at target → skip, log warning. (2) Symlink to different target → skip, log warning. (3) Original file preserved intact.
- Verification: deterministic (file type check)

---

### REQ-122: Fire-and-forget symlink errors

Symlink creation failures (permission denied, read-only filesystem, etc.) are logged to stderr but never block session start.

- Traces to: UC-27 (fire-and-forget, ARCH-6)
- AC: (1) Any symlink error → log to stderr. (2) Session start continues normally. (3) No error propagation to hook output.
- Verification: deterministic (error handling)

---

### DES-42: SessionStart symlink flow

After the binary build step in `session-start.sh`, the hook creates the global symlink:

1. **Build binary** — existing behavior (unchanged)
2. **Ensure target directory** — `mkdir -p ~/.local/bin`
3. **Check existing** — if `~/.local/bin/engram` exists:
   - If symlink pointing to `~/.claude/engram/bin/engram` → done
   - If symlink pointing elsewhere or regular file → log warning, skip
4. **Create symlink** — `ln -s ~/.claude/engram/bin/engram ~/.local/bin/engram`
5. **Continue** — proceed to surface command (existing behavior)

All steps after build are fire-and-forget — errors logged but never block.

- Traces to: UC-27 (SessionStart flow)
- AC: (1) Symlink step runs after build, before surface. (2) Steps are fire-and-forget. (3) New terminals can run `engram` after first session start.
- Verification: integration (hook flow)

---

## UC-28: Automatic Maintenance and Promotion Triggers

---

### REQ-123: Maintenance signal detection at session end

After evaluate in Stop hook, classify all registered memories into quadrants and identify actionable signals: Noise (removal candidate), Leech (rewrite candidate), Working+stale (staleness review), Hidden Gem (keyword broadening candidate). No LLM calls. Reuses `review.Classify()` with registry effectiveness data.

- Traces to: UC-28 (signal detection)
- AC: (1) Correct quadrant assignment matching review.Classify logic. (2) No LLM calls — local-only computation. (3) <2s on local I/O. (4) Fire-and-forget on errors (ARCH-6). (5) Memories with <5 evaluations (InsufficientData) produce no signal.
- Verification: deterministic (quadrant classification)

---

### REQ-124: Promotion/demotion signal detection at session end

Query registry for tier transition candidates using existing threshold logic. Memory→skill candidates via `Promoter.Candidates()`. Skill→CLAUDE.md candidates via `ClaudeMDPromoter.PromotionCandidates()`. CLAUDE.md demotion candidates via `ClaudeMDPromoter.DemotionCandidates()`.

- Traces to: UC-28 (promotion signal detection)
- AC: (1) Same thresholds as existing promote/demote commands. (2) No LLM calls. (3) Fire-and-forget on errors. (4) Missing registry → skip silently.
- Verification: deterministic (threshold filtering)

---

### REQ-125: Proposal queue JSONL file

Signals written to `<data-dir>/proposal-queue.jsonl`. Each line is a JSON object: `{type, source_id, signal, quadrant, summary, detected_at}`. Atomic write via temp+rename (creationlog pattern).

- Traces to: UC-28 (queue persistence)
- AC: (1) Created if absent. (2) Atomic writes (temp file + rename). (3) Malformed lines skipped on read. (4) Bounded growth via pruning (REQ-126).
- Verification: deterministic (file I/O)

---

### REQ-126: Stale signal pruning

Before appending new signals, prune the queue: entries >30 days old, entries for deleted memories (source file no longer exists), entries where quadrant is no longer actionable (re-check against current registry). Deduplicate by source_id + type.

- Traces to: UC-28 (queue hygiene)
- AC: (1) Age pruning (>30 days). (2) Existence check (source file exists). (3) Quadrant re-check against current data. (4) Dedup by source_id + type (keep newest).
- Verification: deterministic (pruning logic)

---

### REQ-127: SessionStart proposal queue surfacing with memory detail

Read queue + load memory content for each signal. Output includes: signal metadata, memory title, memory content summary, effectiveness stats (surfaced count, follow rate), and quadrant rationale. Includes actionable instructions for the conversation model (CLI commands to run). Empty queue = no output. Goes into `additionalContext`.

- Traces to: UC-28 (surfacing)
- AC: (1) Each signal includes memory title + content excerpt. (2) Effectiveness stats included. (3) Empty queue = silent (no output). (4) Queue persists until acted upon. (5) Model-facing action instructions included.
- Verification: deterministic (formatting)

---

### REQ-128: Atomic proposal application via CLI

New `engram apply-proposal` subcommand. Accepts action + memory path + parameters. Actions:
- `--action remove`: delete memory TOML + remove registry entry
- `--action rewrite --fields '{"title":"...","content":"...","keywords":[...]}'`: update TOML fields + registry content hash
- `--action broaden --keywords 'kw1,kw2'`: append keywords to memory TOML
- `--action escalate --level N`: update escalation_level field

- Traces to: UC-28 (apply-proposal)
- AC: (1) File write is atomic (temp+rename). (2) Registry updated in same operation. (3) Clears matching signal from proposal queue. (4) Fire-and-forget on queue cleanup failure. (5) Returns structured JSON result (success/error). (6) Missing memory file → error result.
- Verification: deterministic (file I/O + registry)

---

### REQ-129: Promotion with model-generated content

Extend `engram promote` to accept `--content` flag. When provided, skip LLM generation and use the supplied content. Combined with `--yes` to skip interactive confirmation (model already confirmed with user).
- `engram promote --to-skill --candidate <id> --content '<skill>' --yes`
- `engram promote --to-claude-md --candidate <id> --content '<entry>' --yes`

- Traces to: UC-28 (promote with content)
- AC: (1) `--content` bypasses Generator.Generate LLM call. (2) `--yes` bypasses Confirmer.Confirm. (3) Registry merge still happens. (4) Clears matching promote signal from queue. (5) Existing promote flow unchanged when flags absent.
- Verification: deterministic (flag handling)

---

### DES-43: Proposal queue schema

Each line in `proposal-queue.jsonl`:
```json
{"type":"maintain","source_id":"path/to/memory.toml","signal":"leech_rewrite","quadrant":"Leech","summary":"High surfacing, low follow-through","detected_at":"2026-03-10T12:00:00Z"}
```

Signal values for maintain type: `noise_removal`, `leech_rewrite`, `staleness_review`, `hidden_gem_broadening`, `escalation`.
Signal values for promote type: `memory_to_skill`, `graduation`. (`graduation` replaces `skill_to_claudemd` and `claudemd_demotion` — S5 simplification. Signal carries recommendation text, not a specific target.)

- Traces to: UC-28 (queue format)
- AC: (1) Valid JSON per line. (2) Signal values are one of the defined constants. (3) Timestamps in RFC3339.
- Verification: deterministic (schema validation)

---

### DES-44: SessionStart surfacing format (model-facing)

The surfacing output is designed for the conversation model, not the user. Includes structured signal data + memory excerpts + instruction block:
```
[engram] Pending maintenance signals (present these to the user for review):

Signal 1/3: LEECH — "always-use-targ-test"
  Quadrant: Leech (surfaced 14 times, followed 21%)
  Content: "Use targ test for all test operations..."
  Diagnosis: High surfacing, low follow-through — content may need rewriting
  Action: Propose a rewrite to the user. If approved, call:
    engram apply-proposal --action rewrite --memory <path> --fields '<json>'
```

Goes into `additionalContext`. The model reads this and starts the interview.

- Traces to: UC-28 (surfacing format)
- AC: (1) Human-readable format for model consumption. (2) Includes CLI commands. (3) Includes effectiveness stats.
- Verification: deterministic (output format)

---

### DES-45: Stop hook phase ordering

After existing phases: learn → evaluate → audit → **signal-detect** (new, last, local-only). Signal detection runs last because it depends on evaluate having updated the registry.

- Traces to: UC-28 (hook ordering)
- AC: (1) signal-detect runs after audit. (2) Fire-and-forget (exit 0 always). (3) No LLM calls.
- Verification: integration (hook flow)

---

### DES-46: CLI subcommands

Three new subcommands:
- `engram signal-detect --data-dir <path>` — called by Stop hook, writes queue
- `engram signal-surface --data-dir <path> --format json` — called by SessionStart hook, outputs detailed model-facing context
- `engram apply-proposal --data-dir <path> --action <action> --memory <path> [--fields/--keywords/--level]` — called by conversation model after user confirms

- Traces to: UC-28 (CLI design)
- AC: (1) All three subcommands wired in cli.go. (2) signal-detect and signal-surface are hook-callable. (3) apply-proposal returns JSON result.
- Verification: deterministic (CLI wiring)

---

### DES-41: Source path configuration

Source paths are provided to the auto-registration system via injected configuration, not hardcoded paths. The CLI wiring layer resolves paths from the environment:

| Source type | Resolution |
|-------------|-----------|
| `claude-md` (project) | Working directory + `/CLAUDE.md` |
| `claude-md` (global) | `~/.claude/CLAUDE.md` |
| `memory-md` | `~/.claude/projects/<project-slug>/memory/MEMORY.md` |
| `rule` | Working directory + `/.claude/rules/*` |
| `skill` | Plugin root + `/skills/*/SKILL.md` (or similar convention) |

The project slug for MEMORY.md follows Claude Code's convention (path-based slug of the working directory).

- Traces to: UC-26 (source discovery paths)
- AC: (1) All paths resolved at CLI wiring layer. (2) Paths injected into the registration system. (3) No hardcoded paths in internal packages. (4) Missing paths produce empty entry lists (non-fatal).
- Verification: deterministic (path resolution, DI wiring)

---

## REQ-P0a-1: EnforcementLevel field in InstructionEntry (P0a)

Each `InstructionEntry` in the registry carries an `enforcement_level` field. The level progresses through four values: `advisory` → `emphasized_advisory` → `reminder` → `graduated`. New entries default to `advisory`. Entries loaded from JSONL without the field are backfilled to `advisory` at load time.

- Traces to: UC-23 (data model constraint — registry stores delivery salience per instruction)
- AC: (1) `EnforcementLevel` type defined with four string constants: `advisory`, `emphasized_advisory`, `reminder`, `graduated`. (2) `InstructionEntry.EnforcementLevel` field exists with JSON tag `enforcement_level`. (3) New entries registered with zero-value field are stored as `advisory`. (4) Entries loaded from JSONL with missing `enforcement_level` field are backfilled to `advisory`. (5) Field round-trips through JSONL marshal/unmarshal.
- Verification: deterministic (enum constants, JSON round-trip, backfill)

---
