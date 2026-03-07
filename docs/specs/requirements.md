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

## REQ-9: SessionStart surfacing — top 20 by recency

When a session starts (SessionStart hook), the system reads all memory TOML files from the data directory, sorts by `updated_at` descending, and surfaces the top 20 as a system reminder.

- Traces to: UC-2 (SessionStart surfacing)
- AC: (1) All `.toml` files in `<data-dir>/memories/` are read and parsed. (2) Sorted by `updated_at` descending. (3) Top 20 are included in the system reminder. (4) Each entry shows title and file path. (5) If fewer than 20 memories exist, all are surfaced. (6) If no memories exist, no reminder is emitted (empty stdout).
- Verification: deterministic (file listing, sort, count)

---

## REQ-10: UserPromptSubmit surfacing — keyword match

When a user message is submitted (UserPromptSubmit hook), the system matches the message against memory `keywords` and `concepts` fields. Memories with at least one keyword or concept appearing in the message are surfaced as a system reminder.

- Traces to: UC-2 (UserPromptSubmit surfacing)
- AC: (1) Each memory's `keywords` and `concepts` arrays are checked for whole-word matches in the user message (case-insensitive). (2) Matching memories are surfaced with title, file path, and which keywords matched. (3) If no memories match, no surfacing reminder is emitted. (4) Surfacing runs alongside UC-3 correction detection — both outputs are concatenated.
- Verification: deterministic (keyword presence check)

---

## REQ-11: PreToolUse keyword pre-filter and advisory surfacing

When a tool call is about to execute (PreToolUse hook), the system scans memory TOML files for keyword matches against the tool name and tool input. Only memories with an `anti_pattern` field are candidates (tier A always, tier B when generalizable per REQ-7 — tier C memories never have anti-patterns). Matching memories are surfaced as an advisory system reminder for the agent to evaluate with full session context.

- Traces to: UC-2 (PreToolUse advisory surfacing, tier-aware anti-pattern filtering)
- AC: (1) Only memories with a non-empty `anti_pattern` field are scanned (tier A always, tier B sometimes, tier C never per REQ-7). (2) Each candidate memory's `keywords` are checked for whole-word matches (case-insensitive) in the tool name or tool input arguments. (3) Memories with at least one keyword match are surfaced as a system reminder with title, principle, and file path. (4) If no memories match, no output is emitted (zero overhead — no LLM call, no advisory). (5) The agent has full session context to exercise judgment on whether the tool call violates the memory's principle.
- Verification: deterministic (keyword presence check, tier-aware anti-pattern filtering)

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

## REQ-15: LLM session transcript extraction with unified tier criteria

When a PreCompact or SessionEnd hook fires, the system sends the session transcript to an LLM (claude-haiku-4-5-20251001) which extracts candidate learnings and classifies each using the same A/B/C tier criteria as UC-3 (REQ-7). Each candidate has the same structured fields as UC-3 memories: title, content, observation_type, concepts, keywords, principle, anti_pattern (tier-gated), rationale, filename_summary.

The LLM extracts:
- Explicit instructions the real-time classifier missed (tier A)
- Teachable corrections with generalizable principles (tier B)
- Contextual facts: architectural decisions, discovered constraints, working solutions, implicit preferences (tier C)

- Traces to: UC-1 (LLM extraction, tier classification, anti-pattern gating)
- AC: (1) API call uses claude-haiku-4-5-20251001. (2) The prompt includes the full transcript (or the portion about to be compacted for PreCompact). (3) Response is parsed as a JSON array of candidate objects, each with required fields plus `tier` field (A/B/C). (4) Invalid or unparseable responses return an error. (5) Anti-pattern field is populated per REQ-7: required for A, optional for B, empty for C. (6) The LLM applies quality gate (REQ-16) and tier assignment simultaneously.
- Verification: deterministic (JSON schema validation, tier assignment accuracy)

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

## REQ-20: CLI `learn` subcommand

The system adds a `learn` subcommand to the engram binary: `engram learn --data-dir <path>`. The transcript is read from stdin (not a flag) to avoid command-line length limits.

- Traces to: UC-1 (CLI entry point)
- AC: (1) `engram learn --data-dir <path>` reads transcript from stdin. (2) Runs the extraction → dedup → write pipeline. (3) Exit 0 always — errors logged to stderr. (4) Pure Go, no CGO.
- Verification: deterministic (subcommand runs, correct pipeline)

---

## REQ-25: Creation log write for deferred visibility

During UC-1 learning (PreCompact/SessionEnd), each memory file successfully written is also logged to `<data-dir>/creation-log.jsonl` (see REQ-23 for format). Logging happens after the TOML file is written. See REQ-23 for JSONL format and fire-and-forget error handling.

- Traces to: UC-1 (creation visibility, deferred reporting)
- AC: (1) For each memory file written, append one JSONL line to `creation-log.jsonl`. (2) Entry includes timestamp (RFC 3339), memory title, tier (A/B/C), and filename. (3) Appends are atomic (temp write + rename). (4) Write errors logged to stderr, don't fail the learning operation. (5) Creation log enables deferred visibility at next SessionStart (REQ-24).
- Verification: deterministic (JSONL format check, file append success)

---

## DES-9: Hook wiring — PreCompact and SessionEnd

PreCompact hook (`hooks/pre-compact.sh`) and SessionEnd hook (`hooks/session-end.sh`) are registered in `hooks/hooks.json`. Both invoke the same pipeline: read the transcript from stdin, pass to `engram learn --data-dir <path>`.

The hook reads the transcript from the stdin JSON payload (field depends on hook event — `transcript` or `conversation`). Token retrieval uses the same platform-aware mechanism as DES-3 (macOS Keychain fallback to env var).

- Traces to: UC-1 (hook wiring)

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

## DES-18: UserPromptSubmit context-update pipeline

The UserPromptSubmit hook script launches `engram context-update` in parallel (background) with the existing `engram correct` + `engram surface` calls. The context-update call receives `--transcript-path`, `--session-id`, and `--data-dir` flags. It runs concurrently — the hook does not wait for it before returning output from correct/surface.

- Traces to: UC-14 (piggybacked on UserPromptSubmit)
- AC: (1) context-update runs in background (trailing `&`). (2) Hook still returns correct/surface output promptly. (3) context-update receives transcript path and session ID from hook JSON stdin.

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
