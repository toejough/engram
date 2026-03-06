# Tests

Behavioral test list for UC-3 (Remember & Correct) and UC-2 (Hook-Time Surfacing & Enforcement). BDD Given/When/Then format. Default property-based via rapid; example-based justified inline.

---

## Unified Classifier (ARCH-2)

### T-1: Fast-path keywords trigger tier-A classification

**Given** a message containing one of the three fast-path keywords (`remember`, `always`, `never`),
**When** Classify is called with the message,
**Then** a ClassifiedMemory is returned with Tier "A" and all structured fields populated.

Fast-path check should be case-insensitive, whole-word matching.

- Traces to: ARCH-2, REQ-1, REQ-7

### T-2: Non-signal message returns nil

**Given** a message that is casual conversation with no learning signal (e.g., "hold on", "try again"),
**When** Classify is called and the LLM returns tier `null`,
**Then** nil is returned and no memory is created.

Uses fake LLM returning `{"tier": null}`.

- Traces to: ARCH-2, REQ-1

### T-3: LLM classifier returns tier A (explicit instruction)

**Given** a message with a learning signal (explicit instruction that isn't a fast-path keyword, e.g., "Use fish shell exclusively in this project"),
**When** Classify is called,
**Then** the LLM returns ClassifiedMemory with Tier "A" and anti_pattern is populated.

Uses fake LLM returning tier A with anti-pattern.

- Traces to: ARCH-2, REQ-2, REQ-7

### T-4: LLM classifier returns tier B/C with tier-gated anti-pattern

**Given** messages classifying as tier B (teachable correction) or tier C (contextual fact),
**When** Classify is called,
**Then** tier B generates anti_pattern when generalizable (LLM decides), tier C generates empty anti_pattern.

Uses fake LLM returning appropriate tiers and anti-pattern values.

- Traces to: ARCH-2, REQ-2, REQ-7

---

## Transcript Context Reading (ARCH-3)

### T-5: ReadRecent reads recent transcript portion (~2000 tokens)

**Given** a transcript file with 5000 tokens,
**When** ReadRecent is called with maxTokens=2000,
**Then** the last ~2000 tokens are returned (tail of the file).

Uses file I/O with DI injection for testability.

- Traces to: ARCH-3, REQ-X

### T-6: ReadRecent with missing file returns empty string (non-fatal)

**Given** a transcript_path pointing to a non-existent file,
**When** ReadRecent is called,
**Then** an empty string is returned (non-fatal, context is advisory).

- Traces to: ARCH-3, REQ-X

### T-7: Classifier includes transcript context in LLM call

**Given** a message and recent transcript context,
**When** Classify is called,
**Then** the LLM prompt includes both the message and the recent context, improving classification accuracy.

Verifies that context is passed to the LLM call (fake HTTP transport or mock LLM).

- Traces to: ARCH-2, ARCH-3, REQ-2, REQ-X

---

## TOML File Writer (ARCH-4)

### T-8: Write creates TOML file with all fields

**Given** an EnrichedMemory,
**When** Write is called with a data directory,
**Then** a `.toml` file exists at `<data-dir>/memories/<slug>.toml` containing all required fields as valid TOML.

- Traces to: ARCH-4, REQ-3

### T-9: Filename slug is 3-5 hyphenated lowercase words

**Given** an EnrichedMemory with FilenameSummary "Use Targ Not Go Test",
**When** Write is called,
**Then** the filename is `use-targ-not-go-test.toml`.

Property-based: generate filename summaries, verify slug format.

- Traces to: ARCH-4, REQ-3

### T-10: Duplicate filename gets numeric suffix

**Given** a file already exists at the computed slug path,
**When** Write is called,
**Then** the file is written to `<slug>-2.toml` (incrementing as needed).

- Traces to: ARCH-4, REQ-3

### T-11: Write is atomic (temp file + rename)

**Given** an EnrichedMemory,
**When** Write is called,
**Then** the file is written atomically: temp file created first, then renamed to final path.

- Traces to: ARCH-4, REQ-3

### T-12: Memories directory is created if missing

**Given** a data directory with no `memories/` subdirectory,
**When** Write is called,
**Then** the `memories/` directory is created and the file is written.

- Traces to: ARCH-4, REQ-3

---

## System Reminder Renderer (ARCH-5)

### T-13: Memory produces DES-1 format with tier

**Given** a ClassifiedMemory with Tier "A" or "B" and file path,
**When** Render is called,
**Then** output matches DES-1 format: `[engram] Memory captured (tier A).` header, Created/Type/File fields.

- Traces to: ARCH-5, REQ-4, DES-1

---

## Pipeline (ARCH-1)

### T-15: Full pipeline — classify → write → render

**Given** a message with a learning signal and transcript context, with all pipeline stages wired,
**When** Run is called (with message and transcript_path),
**Then** the Classifier, Writer, and Renderer execute in order and a system reminder string is returned.

Uses fakes for all three DI interfaces. Verifies call order and that transcript context is passed to Classifier.

- Traces to: ARCH-1, REQ-1, REQ-2, REQ-3, REQ-4

### T-16: No signal — pipeline short-circuits

**Given** a message with no learning signal (null classification),
**When** Run is called,
**Then** empty string is returned and Writer/Renderer are never called.

- Traces to: ARCH-1, REQ-1

---

## CLI Wiring (ARCH-6)

### T-18: `correct` subcommand reads transcript_path from hook JSON

**Given** hook JSON input with `.prompt` and `.transcript_path` fields,
**When** the CLI parses stdin and invokes Corrector.Run with both message and transcript_path,
**Then** the Classifier receives transcript context and classifies with full session awareness.

- Traces to: ARCH-6, REQ-6, REQ-X

### DES-3: Static hook script reads transcript context

**Given** the static hook script at `hooks/user-prompt-submit.sh`,
**When** its content is read,
**Then** it references `correct`, `bin/engram`, `jq`, `.prompt`, `.transcript_path`, `CLAUDE_PLUGIN_ROOT`, and `ENGRAM_API_TOKEN`.

- Traces to: ARCH-6, DES-3, REQ-X

---

### T-19: `correct` with no signal produces empty stdout

**Given** `engram correct --message "hello world" --data-dir <tmpdir>`,
**When** Run is called and the Classifier returns nil (no signal),
**Then** stdout is empty and no file is created.

- Traces to: ARCH-6, REQ-6

---

## Build Automation (ARCH-8)

### T-20: Plugin manifest exists

**Given** the plugin manifest at `.claude-plugin/plugin.json`,
**When** its content is read,
**Then** it contains `"name": "engram"` and a `"description"` field.

- Traces to: ARCH-8, REQ-8

### T-21: Hooks JSON has UserPromptSubmit

**Given** the hooks definition at `hooks/hooks.json`,
**When** its content is read,
**Then** it contains a `UserPromptSubmit` entry pointing to `user-prompt-submit.sh`.

- Traces to: ARCH-8, REQ-8, ARCH-6

---

## Cross-Platform Token (ARCH-6 update)

### T-22: UserPromptSubmit hook script has platform-aware token retrieval

**Given** the static hook script at `hooks/user-prompt-submit.sh`,
**When** its content is read,
**Then** it checks `uname` for platform, attempts Keychain on macOS, and falls back to `ENGRAM_API_TOKEN` env var. It does not hard-fail if Keychain is unavailable.

- Traces to: ARCH-6, DES-3, REQ-8

### T-23: bin/ is in .gitignore

**Given** the `.gitignore` file at the repo root,
**When** its content is read,
**Then** it contains an entry that ignores the `bin/` directory.

- Traces to: ARCH-8, REQ-8

---

# UC-2 Tests

## Memory Retrieval (ARCH-9)

### T-24: ListMemories returns all TOML files sorted by updated_at

**Given** a data directory with 3 memory TOML files with different `updated_at` timestamps,
**When** ListMemories is called,
**Then** all 3 memories are returned, sorted by `updated_at` descending (most recent first), with Title, Keywords, Concepts, AntiPattern, Principle, and FilePath populated.

- Traces to: ARCH-9, REQ-9

### T-25: ListMemories returns empty slice when no memories exist

**Given** a data directory with an empty `memories/` subdirectory,
**When** ListMemories is called,
**Then** an empty slice is returned (no error).

- Traces to: ARCH-9, REQ-9

### T-26: ListMemories skips unparseable files

**Given** a data directory with 2 valid TOML files and 1 invalid file,
**When** ListMemories is called,
**Then** 2 memories are returned and the invalid file is logged to stderr but does not cause an error.

- Traces to: ARCH-9, REQ-9

---

## SessionStart Surfacing (ARCH-9, ARCH-12)

### T-27: SessionStart surfaces top 20 by recency

**Given** a data directory with 25 memory files,
**When** surface is called with mode session-start,
**Then** output contains exactly 20 memory entries in DES-5 format, ordered by `updated_at` descending.

- Traces to: ARCH-9, ARCH-12, REQ-9, DES-5

### T-28: SessionStart with fewer than 20 memories surfaces all

**Given** a data directory with 3 memory files,
**When** surface is called with mode session-start,
**Then** output contains all 3 entries in DES-5 format.

- Traces to: ARCH-9, ARCH-12, REQ-9, DES-5

### T-29: SessionStart with no memories produces empty output

**Given** a data directory with no memory files,
**When** surface is called with mode session-start,
**Then** stdout is empty.

- Traces to: ARCH-9, ARCH-12, REQ-9

---

## UserPromptSubmit Surfacing (ARCH-9, ARCH-10, ARCH-12)

### T-30: Keyword match surfaces relevant memories

**Given** memories with keywords ["commit", "git"] and ["targ", "build"], and a user message containing "commit",
**When** surface is called with mode prompt,
**Then** only the memory with keyword "commit" is surfaced in DES-6 format, showing which keyword matched.

- Traces to: ARCH-9, ARCH-12, REQ-10, DES-6

### T-31: No keyword match produces empty output

**Given** memories with keywords ["commit", "git"] and a user message "hello world",
**When** surface is called with mode prompt,
**Then** stdout is empty.

- Traces to: ARCH-9, ARCH-12, REQ-10

### T-32: Keyword matching is case-insensitive and whole-word

**Given** a memory with keyword "commit" and a user message "COMMIT this change",
**When** surface is called with mode prompt,
**Then** the memory is surfaced (case-insensitive match). But a message "recommit" does NOT match (whole-word boundary).

- Traces to: ARCH-10, REQ-10

---

## PreToolUse Keyword Pre-Filter (ARCH-10)

### T-33: Pre-filter matches memory keywords in tool input

**Given** a memory with anti_pattern "manual git commit" and keywords ["commit", "git"], and a tool call {name: "Bash", input: "git commit -m 'fix'"},
**When** MatchMemories is called,
**Then** the memory is returned as a candidate (keyword "commit" matched in tool input).

- Traces to: ARCH-10, REQ-11

### T-34: Pre-filter skips memories without anti_pattern

**Given** a memory with empty anti_pattern and keywords ["commit"], and a tool call containing "commit",
**When** MatchMemories is called,
**Then** the memory is NOT returned (no anti_pattern = not a candidate for enforcement).

- Traces to: ARCH-10, REQ-11

### T-35: Pre-filter returns empty when no keywords match

**Given** a memory with anti_pattern and keywords ["commit", "git"], and a tool call {name: "Read", input: "/path/to/file.go"},
**When** MatchMemories is called,
**Then** empty slice is returned (no keyword overlap).

- Traces to: ARCH-10, REQ-11

---

## Surface Subcommand Routing (ARCH-12)

### T-40: Mode session-start routes to SessionStart surfacing

**Given** the surface subcommand with `--mode session-start --data-dir <tmpdir>`,
**When** Run is called,
**Then** it reads memories and produces DES-5 format output.

- Traces to: ARCH-12, REQ-14

### T-41: Mode prompt routes to keyword surfacing

**Given** the surface subcommand with `--mode prompt --message "commit" --data-dir <tmpdir>`,
**When** Run is called,
**Then** it reads memories, matches keywords, and produces DES-6 format output.

- Traces to: ARCH-12, REQ-14

### T-42: Mode tool routes to advisory surfacing

**Given** the surface subcommand with `--mode tool --tool-name Bash --tool-input '{"command":"git commit"}' --data-dir <tmpdir>`,
**When** Run is called,
**Then** it reads memories, runs pre-filter, emits system-reminder advisory with matching memories (title, principle, file path) or no output if no matches.

- Traces to: ARCH-12, REQ-14

### T-69: SessionStart JSON format produces summary and context

**Given** memories exist and the surface subcommand is called with `--mode session-start --format json`,
**When** Run completes,
**Then** stdout is a JSON object with `summary` (e.g., `"[engram] Loaded 1 memories."`) and `context` (the full `<system-reminder>` XML block).

- Traces to: ARCH-12, REQ-14, DES-5

### T-70: Prompt JSON format produces summary and context

**Given** a memory with keyword "commit" and the surface subcommand is called with `--mode prompt --message "commit" --format json`,
**When** Run completes,
**Then** stdout is a JSON object with `summary` (e.g., `"[engram] 1 relevant memories."`) and `context` (the full `<system-reminder>` XML block).

- Traces to: ARCH-12, REQ-14, DES-6

### T-71: Tool JSON format produces summary and context

**Given** a memory with anti_pattern and matching keywords, and the surface subcommand is called with `--mode tool --format json`,
**When** Run completes,
**Then** stdout is a JSON object with `summary` (e.g., `"[engram] 1 tool advisories."`) and `context` (the full `<system-reminder>` XML block).

- Traces to: ARCH-12, REQ-14, DES-7

### T-72: No-match JSON format produces empty output

**Given** no memories exist and the surface subcommand is called with `--mode session-start --format json`,
**When** Run completes,
**Then** stdout is empty (not an empty JSON object).

- Traces to: ARCH-12, REQ-14

---

## Hook Script Integration (ARCH-13)

### T-43: SessionStart hook calls surface after build

**Given** the session-start hook script,
**When** its content is read,
**Then** it calls `engram surface --mode session-start --format json` after the build step, and reshapes the JSON output into `{systemMessage, additionalContext}` format for Claude Code.

- Traces to: ARCH-13, DES-8

### T-44: UserPromptSubmit hook calls both correct and surface

**Given** the user-prompt-submit hook script,
**When** its content is read,
**Then** it calls `engram correct` (capturing output) and `engram surface --mode prompt --format json`, combining both into a single JSON response with `{systemMessage, additionalContext}`. Correct output is prepended to additionalContext when present.

- Traces to: ARCH-13, DES-8

### T-45: PreToolUse hook registered in hooks.json

**Given** the hooks definition at `hooks/hooks.json`,
**When** its content is read,
**Then** it contains a `PreToolUse` entry pointing to a hook script.

- Traces to: ARCH-13, DES-8

### T-46: PreToolUse hook script calls surface with tool mode

**Given** the pre-tool-use hook script,
**When** its content is read,
**Then** it reads tool call from stdin JSON and calls `engram surface --mode tool` with tool-name and tool-input flags.

- Traces to: ARCH-13, DES-8

---

## Surfacing Instrumentation — Tracking Logic (ARCH-19)

### T-73: ComputeUpdate increments count and sets timestamp

**Given** a memory with `SurfacedCount` 0 and zero `LastSurfaced`,
**When** `ComputeUpdate` is called with mode `"prompt"` and a fixed `now` time,
**Then** the returned `SurfacingUpdate` has `SurfacedCount` 1, `LastSurfaced` equal to `now`, and `SurfacingContexts` `["prompt"]`.

Example-based: verifying exact field values for the simplest case.

- Traces to: ARCH-19, REQ-21

### T-74: ComputeUpdate appends to existing contexts

**Given** a memory with `SurfacedCount` 5 and `SurfacingContexts` `["session-start", "prompt", "tool"]`,
**When** `ComputeUpdate` is called with mode `"prompt"`,
**Then** the returned update has `SurfacedCount` 6 and `SurfacingContexts` `["session-start", "prompt", "tool", "prompt"]`.

Example-based: verifying append behavior.

- Traces to: ARCH-19, REQ-21

### T-75: ComputeUpdate enforces max 10 context entries

**Given** a memory with `SurfacingContexts` containing exactly 10 entries,
**When** `ComputeUpdate` is called with mode `"tool"`,
**Then** the returned update has 10 entries with the oldest dropped and `"tool"` appended at the end.

Example-based: boundary condition for FIFO eviction.

- Traces to: ARCH-19, REQ-21

### T-76: Recorder updates TOML file with tracking fields

**Given** a memory TOML file with existing content fields (title, keywords, etc.) and no tracking fields,
**When** `RecordSurfacing` is called with that memory and mode `"session-start"`,
**Then** the TOML file is rewritten with all original fields preserved plus `surfaced_count = 1`, `last_surfaced` set to current time, and `surfacing_contexts = ["session-start"]`.

Example-based: verifying round-trip fidelity and field addition.

- Traces to: ARCH-19, REQ-22

### T-77: Recorder preserves existing tracking fields on update

**Given** a memory TOML file with `surfaced_count = 3`, `last_surfaced = "2026-03-01T00:00:00Z"`, and `surfacing_contexts = ["prompt", "tool", "prompt"]`,
**When** `RecordSurfacing` is called with mode `"session-start"`,
**Then** the file has `surfaced_count = 4`, `last_surfaced` updated to current time, and `surfacing_contexts = ["prompt", "tool", "prompt", "session-start"]`.

Example-based: verifying increment behavior on existing tracking state.

- Traces to: ARCH-19, REQ-22

### T-78: Recorder skips memory on read error and continues

**Given** two memories where the first has an unreadable file path and the second is valid,
**When** `RecordSurfacing` is called with both,
**Then** the first is skipped (no panic, no abort), and the second is successfully updated.

Example-based: verifying error isolation per REQ-22 AC-4.

- Traces to: ARCH-19, REQ-22

---

## Surfacer ↔ Tracker Integration (ARCH-20)

### T-79: Surfacer calls tracker with matched memories and mode

**Given** a Surfacer with a fake tracker and a retriever returning memories that match a prompt keyword,
**When** `Run` is called with mode `"prompt"`,
**Then** the tracker's `RecordSurfacing` is called with the matched memories and mode `"prompt"`.

Example-based: verifying integration wiring.

- Traces to: ARCH-20, REQ-22

### T-80: Surfacer tracker errors do not propagate

**Given** a Surfacer with a tracker that returns an error,
**When** `Run` is called with mode `"session-start"`,
**Then** `Run` returns nil (no error), and the surfacing output is still produced.

Example-based: verifying fire-and-forget per ARCH-6.

- Traces to: ARCH-20, REQ-22, ARCH-6

### T-81: Surfacer with nil tracker produces same output as before

**Given** a Surfacer without a tracker (nil),
**When** `Run` is called with mode `"prompt"`,
**Then** the output is identical to the existing behavior (backward compatible).

Example-based: verifying no regression.

- Traces to: ARCH-20

---

## Memory Retrieval — Tracking Fields (ARCH-9)

### T-82: ListMemories reads tracking fields from TOML

**Given** a memory TOML file with `surfaced_count = 5`, `last_surfaced = "2026-03-01T00:00:00Z"`, and `surfacing_contexts = ["prompt", "tool"]`,
**When** `ListMemories` parses the file,
**Then** the returned `Stored` has `SurfacedCount` 5, `LastSurfaced` equal to the parsed timestamp, and `SurfacingContexts` `["prompt", "tool"]`.

Example-based: verifying field parsing.

- Traces to: ARCH-9, REQ-21

### T-83: ListMemories defaults tracking fields when absent

**Given** a memory TOML file with no tracking fields (existing format),
**When** `ListMemories` parses the file,
**Then** the returned `Stored` has `SurfacedCount` 0, zero `LastSurfaced`, and empty `SurfacingContexts`.

Example-based: verifying backward compatibility.

- Traces to: ARCH-9, REQ-21

---

## Creation Log Writer (ARCH-21)

### T-84: Append creates new log file when none exists

**Given** a data directory with no `creation-log.jsonl` file (readFile returns os.ErrNotExist),
**When** `LogWriter.Append` is called with a LogEntry `{Timestamp: "2026-03-06T12:00:00Z", Title: "Use targ test", Tier: "A", Filename: "use-targ-test.toml"}`,
**Then** writeFile is called with content containing exactly one JSON line matching the entry, and the file path is `<data-dir>/creation-log.jsonl`.

Example-based: verifying file creation and JSONL format.

- Traces to: ARCH-21, REQ-23

### T-85: Append appends to existing log file

**Given** a data directory with an existing `creation-log.jsonl` containing one JSON line,
**When** `LogWriter.Append` is called with a new LogEntry,
**Then** writeFile is called with content containing two JSON lines: the original line preserved, and the new entry appended.

Example-based: verifying append-not-overwrite behavior.

- Traces to: ARCH-21, REQ-23

### T-86: Append sets timestamp from injected clock when empty

**Given** a LogEntry with empty Timestamp and a LogWriter with `now` returning `2026-03-06T15:00:00Z`,
**When** `Append` is called,
**Then** the written JSON line has `timestamp` set to `"2026-03-06T15:00:00Z"`.

Example-based: verifying DI clock injection.

- Traces to: ARCH-21, REQ-23

### T-87: Append write error is returned (caller decides fire-and-forget)

**Given** a LogWriter whose writeFile returns an error,
**When** `Append` is called,
**Then** an error is returned. The caller (Learner pipeline) handles fire-and-forget policy.

Example-based: verifying error propagation to caller boundary.

- Traces to: ARCH-21, REQ-23

---

## Creation Log Reader (ARCH-21)

### T-88: ReadAndClear returns entries and removes file

**Given** a `creation-log.jsonl` with 3 JSON lines,
**When** `LogReader.ReadAndClear` is called,
**Then** 3 LogEntry values are returned with correct fields parsed, and removeFile is called with the log file path.

Example-based: verifying read + delete behavior.

- Traces to: ARCH-21, REQ-24

### T-89: ReadAndClear with missing file returns empty slice

**Given** no `creation-log.jsonl` exists (readFile returns os.ErrNotExist),
**When** `ReadAndClear` is called,
**Then** an empty slice is returned (no error). removeFile is not called.

Example-based: verifying graceful handling of no log.

- Traces to: ARCH-21, REQ-24

### T-90: ReadAndClear skips malformed lines

**Given** a `creation-log.jsonl` with 3 lines where the second line is invalid JSON,
**When** `ReadAndClear` is called,
**Then** 2 valid LogEntry values are returned (malformed line skipped). removeFile is still called.

Example-based: verifying resilience to corruption.

- Traces to: ARCH-21, REQ-24

### T-91: ReadAndClear read error returns error

**Given** a readFile that returns a non-ErrNotExist error,
**When** `ReadAndClear` is called,
**Then** an error is returned. removeFile is not called.

Example-based: verifying error propagation for unexpected read failures.

- Traces to: ARCH-21, REQ-24

---

## SessionStart Creation Report (ARCH-12 update)

### T-92: SessionStart includes creation report before recency surfacing

**Given** a creation log with 2 entries and 3 memory files in the data directory,
**When** surface is called with mode session-start and --format json,
**Then** the JSON `summary` includes both "[engram] Created 2 memories since last session:" and "[engram] Loaded 3 memories." The `context` includes both the creation report system-reminder block (with titles, tiers, filenames) and the recency surfacing system-reminder block.

Example-based: verifying combined output with both sections.

- Traces to: ARCH-12, ARCH-21, REQ-24, DES-5

### T-93: SessionStart with no creation log produces recency-only output

**Given** no creation log exists and 3 memory files in the data directory,
**When** surface is called with mode session-start and --format json,
**Then** the output is identical to existing behavior (recency surfacing only, no creation report section).

Example-based: verifying backward compatibility.

- Traces to: ARCH-12, ARCH-21, REQ-24, DES-5

### T-94: SessionStart with creation log but no memories produces creation-only output

**Given** a creation log with 1 entry but no memory files in the data directory,
**When** surface is called with mode session-start and --format json,
**Then** the JSON `summary` includes "[engram] Created 1 memories since last session:" but no recency section. The creation log is cleared after reading.

Example-based: verifying creation-only path.

- Traces to: ARCH-12, ARCH-21, REQ-24, DES-5

---

## Learner Pipeline Creation Logging (ARCH-14 update)

### T-95: Learner calls CreationLogger after each successful write

**Given** a Learner with a fake CreationLogger, Extractor returning 2 candidates, Deduplicator passing both, and Writer succeeding for both,
**When** `Learner.Run` is called,
**Then** CreationLogger.Append is called twice, once per written memory, with LogEntry containing the memory's title, tier, and filename.

Example-based: verifying integration wiring.

- Traces to: ARCH-14, REQ-25

### T-96: Learner with nil CreationLogger skips logging (backward compatible)

**Given** a Learner with nil CreationLogger, Extractor returning 1 candidate, and Writer succeeding,
**When** `Learner.Run` is called,
**Then** Run completes successfully with 1 file path returned. No panic from nil CreationLogger.

Example-based: verifying backward compatibility.

- Traces to: ARCH-14, REQ-25

### T-97: Learner creation log error does not fail the pipeline

**Given** a Learner with a CreationLogger that returns an error,
**When** `Learner.Run` is called with 1 candidate that passes dedup,
**Then** Run returns 1 file path (write succeeded). The CreationLogger error is swallowed (fire-and-forget).

Example-based: verifying fire-and-forget per ARCH-6.

- Traces to: ARCH-14, REQ-25, ARCH-6

---

## Hook Script Creation Visibility (ARCH-13 update)

### T-98: UserPromptSubmit hook puts creation in systemMessage

**Given** the user-prompt-submit hook script at `hooks/user-prompt-submit.sh`,
**When** its content is read,
**Then** creation output from `engram correct` is placed in `systemMessage` (not `additionalContext`). When surface matches also exist, both creation and surface summary appear in `systemMessage`.

Updates T-44 to verify creation goes to systemMessage.

- Traces to: ARCH-13, DES-3, REQ-4

### T-99: SessionStart hook puts creation report in systemMessage

**Given** the session-start hook script at `hooks/session-start.sh`,
**When** its content is read,
**Then** it calls `engram surface --mode session-start --format json` and reshapes the output so that both the creation report summary and recency summary appear in `systemMessage`.

Updates T-43 to verify creation report visibility.

- Traces to: ARCH-13, DES-5, REQ-24

---

# UC-1 Tests

## Transcript Extraction (ARCH-15)

### T-47: Extraction with tier classification and anti-pattern gating

**Given** a transcript string and a valid OAuth token,
**When** test calls Extract,
**Then** Extract returns a non-empty slice of CandidateLearning, each with all fields populated including `tier` (A/B/C) and tier-gated `anti_pattern`. Tier A always has anti_pattern, tier B sometimes (LLM decides), tier C has empty anti_pattern. The HTTP request uses `Authorization: Bearer` header with `Anthropic-Beta: oauth-2025-04-20`.

Uses fake HTTP transport returning canned JSON array with tier and anti-pattern values.

- Traces to: ARCH-15, REQ-15, REQ-7
- Verification: unit

### T-48: Extraction without token returns ErrNoToken

**Given** a transcript string but no token configured,
**When** test calls Extract,
**Then** ErrNoToken is returned and no HTTP call is made.

- Traces to: ARCH-15, REQ-18
- Verification: unit

### T-49: Invalid LLM response returns error

**Given** a transcript and valid token, and the LLM returns invalid JSON,
**When** test calls Extract,
**Then** an error is returned (not an empty slice).

Uses fake HTTP transport returning malformed JSON.

- Traces to: ARCH-15, REQ-15
- Verification: unit

### T-50: Empty extraction returns empty slice

**Given** a transcript and valid token, and the LLM returns an empty JSON array `[]`,
**When** test calls Extract,
**Then** an empty slice is returned (no error). No downstream stages are invoked.

Uses fake HTTP transport returning `[]`.

- Traces to: ARCH-15, REQ-15
- Verification: unit

### T-51: Quality gate is embedded in extraction prompt

**Given** the system prompt sent by the TranscriptExtractor implementation,
**When** the prompt content is inspected,
**Then** it explicitly instructs rejection of: (1) mechanical patterns, (2) vague generalizations, (3) overly narrow observations. It instructs extraction of: missed corrections, architectural decisions, discovered constraints, working solutions, implicit preferences.

Example-based: verifies prompt content, not LLM behavior.

- Traces to: ARCH-15, REQ-16
- Verification: unit

---

## Deduplication (ARCH-16)

### T-52: Candidate with >50% keyword overlap is filtered

**Given** a candidate with keywords ["commit", "git", "push"] and an existing memory with keywords ["commit", "git", "branch"],
**When** test calls Filter,
**Then** the candidate is excluded (2/3 = 66% overlap > 50%).

- Traces to: ARCH-16, REQ-17
- Verification: unit

### T-53: Candidate with ≤50% keyword overlap survives

**Given** a candidate with keywords ["commit", "git", "targ", "build"] and an existing memory with keywords ["commit", "test"],
**When** test calls Filter,
**Then** the candidate survives (1/4 = 25% overlap ≤ 50%).

- Traces to: ARCH-16, REQ-17
- Verification: unit

### T-54: No existing memories — all candidates survive

**Given** 3 candidates and an empty existing memories slice,
**When** test calls Filter,
**Then** all 3 candidates are returned.

- Traces to: ARCH-16, REQ-17
- Verification: unit

### T-55: Candidate with empty keywords is never filtered

**Given** a candidate with empty keywords array and existing memories with keywords,
**When** test calls Filter,
**Then** the candidate survives (0/0 overlap, division by zero handled as 0%).

- Traces to: ARCH-16, REQ-17
- Verification: unit

### T-56: Idempotency — second run deduplicates against first run's output

**Given** 3 candidates, the first run writes 2 (one deduped), then the same 3 candidates are submitted again with the 2 written files now existing,
**When** test calls Filter for the second run,
**Then** the 2 previously-written candidates are filtered, at most 1 survives (the one that was originally deduped if it doesn't overlap with the new memories either).

Property: Idempotence — running the pipeline twice produces no more files than running it once.

- Traces to: ARCH-16, REQ-19
- Verification: unit

---

## Learner Pipeline (ARCH-14)

### T-57: Full pipeline — extract → dedup → write returns file paths

**Given** a transcript, with Extractor returning 3 candidates, Retriever returning 1 existing memory, Deduplicator filtering 1 candidate, and Writer succeeding for the remaining 2,
**When** test calls Learner.Run,
**Then** Run returns 2 file paths. Stages execute in order: Extract → ListMemories → Filter → Write (×2).

Uses fakes for all four DI interfaces. Verifies call order.

- Traces to: ARCH-14, REQ-15, REQ-17, REQ-20
- Verification: unit

### T-58: No learnings extracted — pipeline short-circuits

**Given** a transcript, with Extractor returning an empty slice,
**When** test calls Learner.Run,
**Then** Run returns an empty slice. Retriever, Deduplicator, and Writer are never called.

- Traces to: ARCH-14, REQ-15
- Verification: unit

### T-59: All candidates filtered — no files written

**Given** a transcript, with Extractor returning 2 candidates, Retriever returning existing memories, and Deduplicator filtering all candidates,
**When** test calls Learner.Run,
**Then** Run returns an empty slice. Writer is never called.

- Traces to: ARCH-14, REQ-17
- Verification: unit

### T-60: Written memories use tier from extraction

**Given** a transcript, with Extractor returning candidates with Tier = "B",
**When** test calls Learner.Run,
**Then** every memory passed to Writer has Confidence matching the candidate's Tier (not hardcoded "C").

- Traces to: ARCH-14, REQ-7
- Verification: unit

---

## CLI Learn Subcommand (ARCH-17)

### T-61: learn subcommand reads transcript from stdin and runs pipeline

**Given** `engram learn --data-dir <tmpdir>` with a transcript piped to stdin and a valid token,
**When** Run is called,
**Then** the pipeline executes (Extractor receives the transcript content). Output to stderr matches DES-10 format with created file paths.

Uses fakes for pipeline stages.

- Traces to: ARCH-17, REQ-20, DES-10
- Verification: unit

### T-62: learn without token emits error to stderr

**Given** `engram learn --data-dir <tmpdir>` with no `ENGRAM_API_TOKEN` set,
**When** Run is called,
**Then** stderr contains `[engram] Error: session learning skipped — no API token configured`. No files are created. Exit 0.

- Traces to: ARCH-17, REQ-18
- Verification: unit

### T-63: learn with extracted learnings emits DES-10 format

**Given** a pipeline that extracts 2 learnings and skips 1 duplicate,
**When** learn completes,
**Then** stderr contains: `[engram] Extracted 2 learnings from session.` followed by title/path lines, then `[engram] Skipped 1 duplicates.`

- Traces to: ARCH-17, DES-10
- Verification: unit

### T-64: learn with no learnings emits DES-10 empty format

**Given** a pipeline that extracts 0 learnings (or all are deduped),
**When** learn completes,
**Then** stderr contains: `[engram] No new learnings extracted.`

- Traces to: ARCH-17, DES-10
- Verification: unit

---

## Hook Script Integration (ARCH-18)

### T-65: hooks.json registers PreCompact hook

**Given** the hooks definition at `hooks/hooks.json`,
**When** its content is read,
**Then** it contains a `PreCompact` entry pointing to a hook script.

- Traces to: ARCH-18, DES-9
- Verification: linter

### T-66: hooks.json registers SessionEnd hook

**Given** the hooks definition at `hooks/hooks.json`,
**When** its content is read,
**Then** it contains a `SessionEnd` entry pointing to a hook script.

- Traces to: ARCH-18, DES-9
- Verification: linter

### T-67: PreCompact hook script reads transcript and calls engram learn

**Given** the PreCompact hook script,
**When** its content is read,
**Then** it reads transcript from stdin JSON, retrieves the OAuth token (platform-aware per DES-3), and pipes the transcript to `engram learn --data-dir`. It exits 0 always.

- Traces to: ARCH-18, DES-9
- Verification: linter

### T-68: SessionEnd hook script reads transcript and calls engram learn

**Given** the SessionEnd hook script,
**When** its content is read,
**Then** it reads transcript from stdin JSON, retrieves the OAuth token (platform-aware per DES-3), and pipes the transcript to `engram learn --data-dir`. It exits 0 always.

- Traces to: ARCH-18, DES-9
- Verification: linter

---

## UC-1 Bidirectional Traceability

### ARCH → Tests

| ARCH | Tests |
|------|-------|
| ARCH-14 | T-57, T-58, T-59, T-60, T-95, T-96, T-97 |
| ARCH-15 | T-47, T-48, T-49, T-50, T-51 |
| ARCH-16 | T-52, T-53, T-54, T-55, T-56 |
| ARCH-17 | T-61, T-62, T-63, T-64 |
| ARCH-18 | T-65, T-66, T-67, T-68 |

### L2 → Tests

| L2 Item | Tests |
|---------|-------|
| REQ-15 | T-47, T-49, T-50, T-57, T-58 |
| REQ-16 | T-51 |
| REQ-17 | T-52, T-53, T-54, T-55, T-57, T-59 |
| REQ-18 | T-48, T-62 |
| REQ-19 | T-56 |
| REQ-20 | T-57, T-61 |
| REQ-7 | T-60 |
| DES-9 | T-65, T-66, T-67, T-68 |
| DES-10 | T-61, T-63, T-64 |
| REQ-25 | T-95, T-96, T-97 |

All L2C items have test coverage. All ARCH-14–18 items have test coverage.

---

## Creation Visibility Bidirectional Traceability (Issue #49)

### ARCH → Tests

| ARCH | Tests |
|------|-------|
| ARCH-21 | T-84, T-85, T-86, T-87, T-88, T-89, T-90, T-91 |
| ARCH-12 (update) | T-92, T-93, T-94 |
| ARCH-14 (update) | T-95, T-96, T-97 |
| ARCH-13 (update) | T-98, T-99 |

### L2 → Tests

| L2 Item | Tests |
|---------|-------|
| REQ-23 | T-84, T-85, T-86, T-87 |
| REQ-24 | T-88, T-89, T-90, T-91, T-92, T-93, T-94, T-99 |
| REQ-25 | T-95, T-96, T-97 |
| REQ-4 | T-98 |
| DES-3 | T-98 |
| DES-5 | T-92, T-93, T-94, T-99 |

All issue #49 L2 items (REQ-23, REQ-24, REQ-25, REQ-4 update, DES-3 update, DES-5 update) have test coverage. All ARCH items (ARCH-21, ARCH-12/13/14 updates) have test coverage.
