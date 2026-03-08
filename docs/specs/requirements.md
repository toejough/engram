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

New `engram context-update` subcommand. Flags: `--transcript-path` (required), `--session-id` (required), `--data-dir` (required). Pipeline: read watermark from context file → extract delta from transcript → strip low-value content → if delta non-empty, summarize via Haiku → write context file with updated watermark. Exit 0 always (fire-and-forget per ARCH-6).

- Traces to: UC-14 (CLI wiring), REQ-40 through REQ-44
- AC: (1) Missing transcript file → exit 0, no context file written. (2) Empty delta → exit 0, context file unchanged. (3) Successful update → context file written with new watermark. (4) API error → exit 0, context file unchanged.

---

## DES-22: SessionStart context injection

The SessionStart hook script reads `.claude/engram/session-context.md` (if it exists) and includes the summary in the `additionalContext` field of its JSON output, alongside the existing memory surfacing context. If the file doesn't exist, no context is injected (no error).

- Traces to: UC-14 (restore on SessionStart), REQ-45
- AC: (1) Context file exists → summary appears in additionalContext. (2) No file → additionalContext contains only memory context (existing behavior). (3) Context is clearly labeled so the model knows it's a session resumption summary.

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
- Responsibilities: (1) Accept CLI flags: --transcript-path, --session-id, --data-dir. (2) Read watermark from context file (or 0 if missing). (3) Extract transcript delta from ARCH-28. (4) Check session ID mismatch: if current session ID ≠ stored session ID, reset offset to 0. (5) Strip content via ARCH-29. (6) If delta non-empty, summarize via ARCH-30. (7) Write updated context file via ARCH-31 with new offset. (8) Exit 0 always (fire-and-forget per ARCH-6).
- Behavioral contract: No error output on missing transcript file, empty delta, or API error. All errors are silent (logged but not surfaced). Exit code always 0.
- Entry point: CLI binary, invoked by UserPromptSubmit hook (background) and PreCompact hook (synchronous).

---

## ARCH-33: Hook integration wiring

Specifies how context updates are triggered and how context is restored.

- Traces to: DES-18 (UserPromptSubmit pipeline), DES-19 (PreCompact flush), DES-22 (SessionStart injection)
- UserPromptSubmit hook: Context-update runs as a separate async hook entry (`"async": true` in hooks.json) via `user-prompt-submit-async.sh`. The synchronous `user-prompt-submit.sh` handles correct/surface only.
- PreCompact hook: Call `engram context-update` synchronously with same flags. Wait for completion (60s timeout available per hook config).
- SessionStart hook: Read context file (if exists) via ARCH-31. Extract summary and inject as `additionalContext` in hook JSON output. Annotate clearly so the model knows this is a session resumption summary.
- Behavioral contract: Hooks remain responsive; background calls don't block. SessionStart always succeeds (missing file → no injection, no error).

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

## REQ-72: Cross-source instruction scanning

The system scans all instruction sources: CLAUDE.md (project + global), engram memories, .claude/rules/ files, and skill files. Each instruction is extracted as a structured item with: source, location, text, keywords, and effectiveness data (if available).

- Traces to: UC-20 (cross-source scanning)
- AC: (1) CLAUDE.md entries are extracted by section header. (2) Memories are loaded from <data-dir>/memories/. (3) Rules files from .claude/rules/. (4) Skill files from plugin skill directories. (5) Each item has source type, file path, and text content. (6) Effectiveness data joined from evaluation pipeline where available.
- Verification: deterministic (file scanning, structured extraction)

---

## REQ-73: Deduplication detection across sources

Compare instructions across all sources to find semantic duplicates. Two instructions are considered duplicates if they share >80% keyword overlap or have near-identical principle text. Report which source to keep based on salience hierarchy: CLAUDE.md > rules > memories.

- Traces to: UC-20 (deduplication)
- AC: (1) Pairwise comparison of all instruction items. (2) Duplicate detection by keyword overlap (>80%) or principle text similarity. (3) Recommendation: keep higher-salience source, remove lower. (4) Output includes both items with source paths and recommendation.
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

Generate rewritten versions of diagnosed instructions. Memory proposals use maintain-compatible format (same as UC-16). CLAUDE.md/skills/rules proposals are diff suggestions showing before/after.

- Traces to: UC-20 (refinement proposals)
- AC: (1) Each diagnosed instruction gets a rewrite proposal. (2) Memory proposals: JSON with proposed TOML field changes (maintain-compatible). (3) CLAUDE.md/rules proposals: unified diff format. (4) Proposal includes rationale explaining what changed and why.
- Verification: deterministic (format validation)

---

## REQ-76: Gap analysis

Compare instruction anti-patterns against observed tool actions in evaluation data to find common violation patterns with no corresponding instruction. Report as gap candidates.

- Traces to: UC-20 (gap analysis)
- AC: (1) Load evaluation data with contradicted outcomes. (2) Extract patterns from contradictions (file types, tool names, common mistakes). (3) Cross-reference against existing instructions. (4) Patterns not covered by any instruction → gap candidates. (5) Output: list of gap candidates with evidence (violation count, example).
- Verification: deterministic (pattern extraction, cross-reference)

---

## REQ-77: Skill decomposition

For skills with low per-line effectiveness, identify which lines are followed vs. ignored. Propose extraction of effective lines or compression of verbose sections.

- Traces to: UC-20 (skill decomposition)
- AC: (1) For each skill file, compute per-line effectiveness (if data available). (2) Lines with <20% follow rate → candidates for removal or rewrite. (3) Lines with >80% follow rate → effective, keep. (4) Proposal: extract effective lines, compress or remove ineffective.
- Verification: deterministic (per-line effectiveness calculation)

---

## REQ-78: CLI command `engram instruct audit`

New subcommand: `engram instruct audit --data-dir <path>`. Outputs a JSON report with: duplicates, quality diagnoses, refinement proposals, gap analysis, skill decomposition.

- Traces to: UC-20 (CLI command)
- AC: (1) Subcommand `instruct audit` registered. (2) Output is JSON with sections: duplicates, diagnoses, proposals, gaps, skills. (3) Exit 0 always. (4) Empty sections are empty arrays.
- Verification: deterministic (CLI registration, JSON output)

---

## REQ-79: No graceful degradation on API failure

If no API token, skip LLM-dependent steps (quality diagnosis, refinement proposals). Deduplication, gap analysis, and skill decomposition still run. Output JSON includes skipped sections as empty arrays with a `skipped_reason` field.

- Traces to: UC-20 (error handling)
- AC: (1) Missing API token → skip REQ-74 and REQ-75. (2) REQ-72, REQ-73, REQ-76, REQ-77 still run. (3) Skipped sections include `{"skipped_reason": "no API token"}`. (4) Exit 0 regardless.
- Verification: deterministic (error condition handling)

---
