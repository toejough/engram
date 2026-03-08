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

### T-30: BM25 ranking surfaces top relevant memories

**Given** memories A (title: "Using Git", content: "Commit workflow...") and B (title: "Build System", content: "targ build target..."), and a user message "how do I commit?",
**When** surface is called with mode prompt,
**Then** memory A is ranked higher than B by BM25 relevance score (higher score due to term "commit" in content). Top 10 ranked results are surfaced in DES-6 format.

- Traces to: ARCH-9, ARCH-12, REQ-10, DES-6
- Type: example-based (BM25 deterministic, ranking depends on term frequency and document frequency)

### T-31: Low-relevance memories produce empty output

**Given** memories with content unrelated to user query, and a user message "hello world",
**When** surface is called with mode prompt,
**Then** BM25 scores all memories as zero or near-zero (no query terms found). No surfacing reminder is emitted (zero overhead).

- Traces to: ARCH-9, ARCH-12, REQ-10

### T-32: BM25 scores term frequency within memory text

**Given** a memory with title "Git Workflow" and content mentioning "commit" 5 times, and a user message "commit commit commit",
**When** surface is called with mode prompt,
**Then** BM25 scores the memory based on term frequency (TF-IDF). Higher frequency in both query and memory = higher score.

- Traces to: ARCH-9, REQ-10
- Type: example-based (BM25 term frequency scoring)

---

## PreToolUse BM25 Candidate Pruning (ARCH-10)

### T-33: Pre-filter ranks anti-pattern candidates by BM25

**Given** memories with anti_pattern (candidates: "manual git commit", "avoid hardcoding secrets") with searchable text, and a tool call {name: "Bash", input: "git commit -m 'fix'"},
**When** surface is called with mode tool,
**Then** BM25 scores each anti_pattern candidate against the tool input. Top 5 ranked candidates are surfaced. Unrelated candidates may score zero.

- Traces to: ARCH-10, REQ-11

### T-34: Pre-filter skips memories without anti_pattern

**Given** memories: one with anti_pattern (candidate), one with empty anti_pattern (not a candidate), and a tool call containing relevant terms,
**When** surface is called with mode tool,
**Then** only anti_pattern memories are indexed and scored. Non-anti_pattern memories are excluded from candidate set before BM25 indexing (tier-aware per REQ-7).

- Traces to: ARCH-10, REQ-11

### T-35: Pre-filter returns empty when no candidates rank above zero

**Given** anti_pattern memories with searchable text (candidates), and a tool call with no overlapping terms,
**When** surface is called with mode tool,
**Then** BM25 scores candidates as zero or near-zero (no query terms found). No surfacing reminder is emitted (zero overhead, zero advisory).

- Traces to: ARCH-10, REQ-11

### T-162: BM25 top-N limit — only top 10 results surfaced for prompt mode

**Given** 15 memories with varying relevance to a user message (all with non-zero BM25 scores),
**When** surface is called with mode prompt,
**Then** only the top 10 ranked memories are surfaced (by BM25 relevance score). Lower-ranked memories 11–15 are not included.

- Traces to: ARCH-9, REQ-10
- Type: example-based (verify top-N limiting in BM25 ranking)

### T-163: BM25 top-N limit — only top 5 results surfaced for tool mode

**Given** 10 anti-pattern memories with varying relevance to a tool input (all with non-zero BM25 scores),
**When** surface is called with mode tool,
**Then** only the top 5 ranked candidates are surfaced (by BM25 relevance score). Lower-ranked candidates 6–10 are not included.

- Traces to: ARCH-10, REQ-11
- Type: example-based (verify top-N limiting in PreToolUse)

### T-164: BM25 handles zero-score memories (relevance below threshold)

**Given** memories with content: one closely matching query terms, others with no overlap,
**When** surface is called with matching and non-matching memories,
**Then** BM25 computes zero or near-zero scores for non-matching memories. They are not surfaced (no threshold gate needed — naturally ranked below matching memories).

- Traces to: ARCH-9, REQ-10
- Type: example-based (BM25 natural zero-scoring behavior)

---

## Frecency Activation Scoring (ARCH-35)

### T-165: Frecency activation computation — all components present

**Given** a memory with SurfacedCount=10, LastSurfaced=2h ago, SurfacingContexts=["session-start","prompt","tool"], and effectiveness score of 80%,
**When** Activation is computed,
**Then** the result combines: log(11) × 1/(1+2) × log(4) × 0.8. All four components are multiplied together to produce a positive activation score.

- Traces to: ARCH-35, REQ-46
- Type: example-based (verify formula components)

### T-166: Frecency activation — never-surfaced memory uses UpdatedAt fallback

**Given** a memory with SurfacedCount=0, LastSurfaced=zero time, UpdatedAt=24h ago, empty SurfacingContexts, no effectiveness data,
**When** Activation is computed,
**Then** frequency=log(1)=0, so activation is 0.0 (frequency of zero dominates). This ensures never-surfaced memories rank below actively-used ones.

- Traces to: ARCH-35, REQ-46
- Type: example-based (fallback behavior for new memories)

### T-167: Frecency activation — effectiveness defaults to 0.5 when no data

**Given** a memory with surfacing history but no evaluation data (effectiveness map has no entry for this memory),
**When** Activation is computed,
**Then** effectiveness component uses default 0.5 (neutral — neither boosted nor penalized).

- Traces to: ARCH-35, REQ-46
- Type: example-based (default effectiveness)

### T-168: Combined BM25 + frecency score preserves BM25 zero

**Given** a memory with high frecency activation but BM25 score of 0.0,
**When** CombinedScore is computed,
**Then** combined score is 0.0 (BM25 of zero stays zero regardless of frecency). Frecency cannot promote irrelevant memories.

- Traces to: ARCH-35, REQ-46, REQ-10
- Type: example-based (multiplicative combination)

### T-169: SessionStart uses pure frecency ranking

**Given** 25 memories with varying surfacing history and effectiveness data,
**When** surface is called with mode session-start,
**Then** memories are ranked by pure frecency activation score (not BM25). Top 20 are surfaced. Higher frequency + recency + spread + effectiveness = higher rank.

- Traces to: ARCH-35, REQ-9
- Type: example-based (SessionStart frecency ranking)

### T-170: Prompt mode re-ranks BM25 top-N by combined score

**Given** 3 memories with BM25 scores [10.0, 8.0, 6.0] and frecency activations [0.1, 0.5, 0.3],
**When** surface is called with mode prompt,
**Then** combined scores are [10×1.1=11.0, 8×1.5=12.0, 6×1.3=7.8]. Memory 2 re-ranks to top position because its frecency boost overcomes its lower BM25 score.

- Traces to: ARCH-35, REQ-10
- Type: example-based (frecency re-ranking changes BM25 order)

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

### T-100: Tool mode with no matching memories produces empty output

**Given** memories exist but none have keywords matching the tool input,
**When** surface is called with mode tool and a tool input containing no matching keywords,
**Then** stdout is empty (no advisory emitted).

- Traces to: ARCH-12, ARCH-10, REQ-11

---

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
**Then** stderr contains: `[engram] Extracted 2 learnings from session.` followed by title/path lines with tier breakdown `(A: X, B: Y, C: Z)`, then `[engram] Skipped 1 duplicates.`

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

---

## Surfacing Log Infrastructure (ARCH-22)

### T-101: Surfacing log append writes JSONL entry

**Given** a SurfacingLogger with an injected write function,
**When** LogSurfacing is called with a memory path, mode "prompt", and timestamp,
**Then** a single JSONL line is appended to surfacing-log.jsonl with memory_path, mode, and surfaced_at (RFC 3339).

Uses DI-injected file append function.

- Traces to: ARCH-22, REQ-26, DES-11

### T-102: Surfacing log append for multiple memories in one event

**Given** a surfacing event that matches 3 memories,
**When** LogSurfacing is called once per matched memory,
**Then** surfacing-log.jsonl contains 3 JSONL lines, one per memory, all with the same mode and timestamp.

- Traces to: ARCH-22, REQ-26

### T-103: Surfacing log append error is fire-and-forget

**Given** a SurfacingLogger with an injected write function that returns an error,
**When** LogSurfacing is called,
**Then** the error is returned to the caller (surfacer swallows it per ARCH-6).

- Traces to: ARCH-22, REQ-26

### T-104: Surfacing log read-and-clear returns events and removes file

**Given** a surfacing-log.jsonl with 5 entries,
**When** ReadAndClear is called,
**Then** it returns 5 SurfacingEvent structs with correct fields, and the file is removed.

Uses DI-injected read and remove functions.

- Traces to: ARCH-22, REQ-26

### T-105: Surfacing log read-and-clear with missing file returns empty slice

**Given** no surfacing-log.jsonl exists,
**When** ReadAndClear is called,
**Then** it returns an empty slice and no error.

- Traces to: ARCH-22, REQ-26, REQ-34

---

## Outcome Evaluation Pipeline (ARCH-23)

### T-106: Evaluator classifies surfaced memories via LLM

**Given** a surfacing log with 2 entries, each memory's TOML loaded, and a transcript,
**When** Evaluate is called,
**Then** the LLM receives the full transcript + 2 memory summaries and returns outcomes for each.

Uses fake LLM returning `[{"memory_path": "...", "outcome": "followed", "evidence": "..."}, ...]`.

- Traces to: ARCH-23, REQ-27, DES-12

### T-107: Evaluator handles empty surfacing log — no LLM call, no output

**Given** an empty surfacing log (or missing file),
**When** Evaluate is called,
**Then** no LLM call is made, no evaluation log is written, and no error is returned.

- Traces to: ARCH-23, REQ-27, REQ-34

### T-108: Evaluator writes per-session evaluation log

**Given** the LLM returns outcomes for 3 surfaced memories,
**When** Evaluate completes,
**Then** a JSONL file is created at `<data-dir>/evaluations/<timestamp>.jsonl` with 3 lines, each containing memory_path, outcome, evidence, evaluated_at.

Uses DI-injected write function. Timestamp in filename has colons replaced by hyphens.

- Traces to: ARCH-23, REQ-28, DES-13

### T-109: Evaluator creates evaluations directory if missing

**Given** no `<data-dir>/evaluations/` directory exists,
**When** Evaluate writes results,
**Then** the directory is created before writing the file.

- Traces to: ARCH-23, REQ-28

### T-110: Evaluator with unparseable LLM response returns error

**Given** an LLM that returns invalid JSON,
**When** Evaluate is called,
**Then** an error is returned and no evaluation log is written.

- Traces to: ARCH-23, REQ-27

### T-111: Evaluator clears surfacing log after reading

**Given** a surfacing-log.jsonl with entries,
**When** Evaluate reads the log,
**Then** the surfacing log file is removed (ensuring idempotency for second trigger).

- Traces to: ARCH-23, REQ-26, REQ-34

---

## Effectiveness Aggregation (ARCH-24)

### T-112: Aggregate computes effectiveness from evaluation logs

**Given** an evaluations directory with 3 session files, where memory A was evaluated 5 times (3 followed, 1 contradicted, 1 ignored),
**When** Aggregate is called,
**Then** memory A's stat shows FollowedCount=3, ContradictedCount=1, IgnoredCount=1, EffectivenessScore=60.0.

Uses DI-injected directory reader and file reader.

- Traces to: ARCH-24, REQ-29

### T-113: Aggregate with missing evaluations directory returns empty map

**Given** no evaluations directory exists,
**When** Aggregate is called,
**Then** an empty map is returned and no error.

- Traces to: ARCH-24, REQ-29

### T-114: Aggregate skips malformed JSONL lines

**Given** an evaluation log with 3 valid lines and 1 malformed line,
**When** Aggregate is called,
**Then** 3 outcomes are aggregated and the malformed line is skipped.

- Traces to: ARCH-24, REQ-29

### T-115: Effectiveness annotation rendered when data exists

**Given** a surfaced memory with aggregated stats (surfaced 5 times, followed 80%),
**When** the surfacer formats output,
**Then** the annotation "(surfaced 5 times, followed 80%)" is appended to the memory's line.

- Traces to: ARCH-24, REQ-30, DES-14

### T-116: No annotation when no evaluation data exists

**Given** a surfaced memory with no evaluation log entries,
**When** the surfacer formats output,
**Then** no annotation is appended (backward compatible output).

- Traces to: ARCH-24, REQ-30, DES-14

---

## Hook Integration — evaluate CLI (ARCH-25)

### T-117: evaluate subcommand runs full pipeline

**Given** a data directory with a surfacing log and memory TOML files,
**When** `runEvaluate` is called with transcript on stdin,
**Then** evaluation log is written and summary output is produced on stdout.

Uses DI-injected dependencies. Verifies end-to-end wiring.

- Traces to: ARCH-25, REQ-32

### T-118: evaluate without API token emits error and exits 0

**Given** no API token configured,
**When** `runEvaluate` is called,
**Then** stderr contains `[engram] Error: evaluation skipped — no API token configured` and no evaluation log is created.

- Traces to: ARCH-25, REQ-33

### T-119: evaluate summary output format

**Given** an evaluation with 3 memories: 2 followed, 1 ignored,
**When** the evaluation summary is rendered,
**Then** stdout contains `[engram] Evaluated 3 memories: 2 followed, 0 contradicted, 1 ignored.`

- Traces to: ARCH-25, REQ-31

### T-120: Hook scripts invoke engram evaluate after learn

**Given** the PreCompact and SessionEnd hook scripts,
**When** the script content is examined,
**Then** `engram evaluate` is invoked after `engram learn`, with `--data-dir` and transcript piped via stdin.

- Traces to: ARCH-25, DES-15

### T-121: Surfacer writes surfacing log during surfacing events

**Given** a Surfacer with an injected SurfacingLogger,
**When** SessionStart, Prompt, or Tool mode surfaces memories,
**Then** each matched memory is logged via SurfacingLogger.LogSurfacing with correct mode.

- Traces to: ARCH-22, REQ-26

### T-161: evaluate CLI applies Strip preprocessing to transcript

**Given** a transcript on stdin containing toolResult JSON bodies and base64 data,
**When** `runEvaluate` processes it,
**Then** the transcript passed to the Evaluator has toolResult bodies and base64 data removed (Strip applied at CLI wiring level).

- Traces to: ARCH-23, REQ-27
- Type: example-based (verify stripped content reaches LLM)

---

## UC-15 Bidirectional Traceability

### ARCH → Tests

| ARCH | Tests |
|------|-------|
| ARCH-22 | T-101, T-102, T-103, T-104, T-105, T-121 |
| ARCH-23 | T-106, T-107, T-108, T-109, T-110, T-111, T-161 |
| ARCH-24 | T-112, T-113, T-114, T-115, T-116 |
| ARCH-25 | T-117, T-118, T-119, T-120 |

### L2 → Tests

| L2 Item | Tests |
|---------|-------|
| REQ-26 | T-101, T-102, T-103, T-104, T-105, T-111, T-121 |
| DES-11 | T-101 |
| REQ-27 | T-106, T-107, T-110, T-161 |
| DES-12 | T-106 |
| REQ-28 | T-108, T-109 |
| DES-13 | T-108 |
| REQ-29 | T-112, T-113, T-114 |
| REQ-30 | T-115, T-116 |
| DES-14 | T-115, T-116 |
| REQ-31 | T-119 |
| DES-15 | T-120 |
| REQ-32 | T-117 |
| REQ-33 | T-118 |
| REQ-34 | T-105, T-107, T-111 |

All UC-15 L2 items have test coverage. All ARCH-22..25 items have test coverage.

---

# UC-6: Memory Effectiveness Review

---

## Matrix Classifier (ARCH-26)

### T-122: Correct quadrant assignment based on median + effectiveness threshold

- **Given** 6 memories: 3 with surfaced_count above median and effectiveness >= 50% (Working), 1 with surfaced_count below median and effectiveness >= 50% (Hidden Gem), 1 with surfaced_count above median and effectiveness < 50% (Leech), 1 with surfaced_count below median and effectiveness < 50% (Noise). All have 5+ evaluations.
- **When** Classify is called
- **Then** each memory is assigned the correct quadrant: Working, Hidden Gem, Leech, or Noise
- **Traces to:** REQ-35 (matrix classification)
- **Type:** example-based (specific quadrant assignments need deterministic verification)

### T-123: Memories with fewer than 5 evaluations classified as InsufficientData

- **Given** a memory with 3 total evaluations (followed + contradicted + ignored)
- **When** Classify is called
- **Then** the memory's Quadrant is InsufficientData and Flagged is false
- **Traces to:** REQ-35 (insufficient data exclusion)
- **Type:** example-based

### T-124: Memory with 5+ evaluations and effectiveness < 40% is flagged

- **Given** a memory with 6 evaluations and effectiveness score 33%
- **When** Classify is called
- **Then** Flagged is true
- **Traces to:** REQ-36 (threshold flagging)
- **Type:** example-based

### T-125: Memory with effectiveness exactly 40% is not flagged

- **Given** a memory with 5 evaluations and effectiveness score exactly 40%
- **When** Classify is called
- **Then** Flagged is false
- **Traces to:** REQ-36 (threshold boundary — strictly less than 40%)
- **Type:** example-based (boundary condition)

### T-126: Memory with effectiveness exactly 50% classified as high follow-through

- **Given** a memory with 10 evaluations and effectiveness score exactly 50%
- **When** Classify is called
- **Then** the memory is in a high follow-through quadrant (Working or Hidden Gem depending on surfacing frequency)
- **Traces to:** REQ-35 (follow-through threshold — >= 50%)
- **Type:** example-based (boundary condition)

### T-127: Empty input produces empty output

- **Given** empty effectiveness and tracking maps
- **When** Classify is called
- **Then** result is an empty slice
- **Traces to:** REQ-35 (edge case)
- **Type:** example-based

### T-128: Memories with tracking data but no evaluations classified as InsufficientData

- **Given** 3 memories with surfaced_count > 0 but zero evaluations
- **When** Classify is called
- **Then** all are classified as InsufficientData with Flagged false
- **Traces to:** REQ-35 (insufficient data — zero evaluations)
- **Type:** example-based

---

## Review CLI (ARCH-27)

### T-129: Review with data outputs all four DES-16 sections

- **Given** evaluation logs and tracking data exist with memories across all four quadrants
- **When** `engram review --data-dir <path>` is run (via RunReview with injected I/O)
- **Then** stdout contains: summary line, quadrant table, flagged list, insufficient-data list
- **Traces to:** REQ-38 (review CLI output), DES-16 (output format)
- **Type:** example-based (format verification)

### T-130: Review with no evaluation directory outputs no-data message

- **Given** data-dir exists but evaluations subdirectory does not
- **When** `engram review --data-dir <path>` is run
- **Then** stdout contains "[engram] No evaluation data found." and exit 0
- **Traces to:** REQ-39 (no-data behavior — missing directory)
- **Type:** example-based

### T-131: Review without --data-dir outputs usage error

- **Given** no --data-dir argument provided
- **When** `engram review` is run
- **Then** output contains usage error message and exit 0
- **Traces to:** REQ-38 (--data-dir required)
- **Type:** example-based

### T-132: Flagged memories sorted by effectiveness ascending

- **Given** 3 flagged memories with effectiveness scores 33%, 20%, 10%
- **When** review is run
- **Then** flagged section lists them in order: 10%, 20%, 33%
- **Traces to:** REQ-38 (sorted by effectiveness ascending)
- **Type:** example-based

### T-133: Insufficient-data section omitted when all memories have 5+ evaluations

- **Given** all memories have 5+ evaluations
- **When** review is run
- **Then** "Insufficient data" section does not appear in output
- **Traces to:** REQ-38 (section omitted when empty)
- **Type:** example-based

---

## Session Continuity Components (UC-14: ARCH-28 through ARCH-34)

### T-134: TranscriptDeltaReader reads from offset 0

- **Given** a transcript JSONL file with 10 lines
- **When** TranscriptDeltaReader.Read is called with offset 0
- **Then** all 10 lines are returned and new offset equals file size
- **Traces to:** ARCH-28, REQ-40
- **Type:** example-based (file I/O via DI injection)

### T-135: TranscriptDeltaReader reads from mid-file offset

- **Given** a transcript JSONL file with 10 lines, each 100 bytes
- **When** TranscriptDeltaReader.Read is called with offset 500 (byte 5, line 5)
- **Then** lines 6-10 are returned and new offset equals file size
- **Traces to:** ARCH-28, REQ-40
- **Type:** example-based (byte offset calculation)

### T-136: TranscriptDeltaReader resets to 0 when file is shorter than offset

- **Given** a transcript file is 1000 bytes and stored offset is 2000
- **When** TranscriptDeltaReader.Read is called with offset 2000
- **Then** entire file (1000 bytes) is returned and new offset is file size
- **Traces to:** ARCH-28, REQ-40
- **Type:** example-based (watermark reset on rotation)

### T-137: TranscriptDeltaReader returns empty delta for empty file

- **Given** a transcript file is empty
- **When** TranscriptDeltaReader.Read is called
- **Then** empty line array is returned and new offset is 0
- **Traces to:** ARCH-28, REQ-40
- **Type:** example-based

### T-138: ContentStripper removes tool result blocks

- **Given** JSONL lines with toolResult role blocks
- **When** ContentStripper.Strip is called
- **Then** tool result blocks are omitted from output
- **Traces to:** ARCH-29, REQ-41
- **Type:** example-based (role-based filtering)

### T-139: ContentStripper replaces base64 strings

- **Given** JSONL lines with base64-encoded strings >100 chars
- **When** ContentStripper.Strip is called
- **Then** base64 strings are replaced with `[base64 removed]`
- **Traces to:** ARCH-29, REQ-41
- **Type:** property-based (generate base64 strings, verify replacement)

### T-140: ContentStripper truncates oversized content blocks

- **Given** JSONL lines with content blocks >2000 characters
- **When** ContentStripper.Strip is called
- **Then** oversized blocks are truncated with `[truncated]` suffix
- **Traces to:** ARCH-29, REQ-41
- **Type:** example-based (size threshold validation)

### T-141: ContentStripper preserves user messages

- **Given** JSONL lines with role=user messages
- **When** ContentStripper.Strip is called
- **Then** user messages are preserved verbatim
- **Traces to:** ARCH-29, REQ-41
- **Type:** example-based

### T-142: ContentStripper preserves assistant text

- **Given** JSONL lines with role=assistant text messages
- **When** ContentStripper.Strip is called
- **Then** assistant text messages are preserved verbatim
- **Traces to:** ARCH-29, REQ-41
- **Type:** example-based

### T-143: ContentStripper preserves tool names, removes tool results

- **Given** JSONL lines with toolUse (with name/command) and toolResult blocks
- **When** ContentStripper.Strip is called
- **Then** tool names and commands are preserved, tool results are omitted
- **Traces to:** ARCH-29, REQ-41
- **Type:** example-based

### T-144: ContextSummarizer returns previous summary on empty delta

- **Given** a previous summary "Current task: foo" and empty stripped delta
- **When** ContextSummarizer.Summarize is called
- **Then** previous summary is returned unchanged (no API call)
- **Traces to:** ARCH-30, REQ-43
- **Type:** example-based (mocked HaikuClient)

### T-145: ContextSummarizer updates summary on non-empty delta

- **Given** a previous summary and a non-empty stripped delta
- **When** ContextSummarizer.Summarize is called
- **Then** HaikuClient is called with combined context and updated summary is returned
- **Traces to:** ARCH-30, REQ-43
- **Type:** example-based (mocked HaikuClient)

### T-146: ContextSummarizer creates new summary from delta without previous

- **Given** an empty previous summary and a non-empty delta
- **When** ContextSummarizer.Summarize is called
- **Then** HaikuClient is called with delta only and new summary is returned
- **Traces to:** ARCH-30, REQ-43
- **Type:** example-based (mocked HaikuClient)

### T-147: ContextSummarizer skips API call when token is empty

- **Given** an empty API token and a non-empty delta
- **When** ContextSummarizer.Summarize is called
- **Then** no API call is made and previous summary is returned unchanged
- **Traces to:** ARCH-30, REQ-43
- **Type:** example-based

### T-148: ContextSummarizer returns previous summary on API error

- **Given** a previous summary and a mocked HaikuClient that returns error
- **When** ContextSummarizer.Summarize is called
- **Then** previous summary is returned unchanged (error is silent)
- **Traces to:** ARCH-30, REQ-43
- **Type:** example-based (mocked HaikuClient)

### T-149: SessionContextFile parses HTML metadata

- **Given** a context file with HTML comment: `<!-- engram session context | updated: 2026-03-07T00:00:00Z | offset: 1000 | session: abc123 -->`
- **When** SessionContextFile.Read is called
- **Then** metadata is parsed into (offset: 1000, sessionID: "abc123")
- **Traces to:** ARCH-31, REQ-42
- **Type:** example-based (string parsing)

### T-150: SessionContextFile extracts markdown summary

- **Given** a context file with HTML comment on first line, blank line, then markdown summary
- **When** SessionContextFile.Read is called
- **Then** markdown summary is returned (HTML comment excluded)
- **Traces to:** ARCH-31, REQ-45
- **Type:** example-based

### T-151: SessionContextFile writes atomically

- **Given** a SessionContext struct and a target file path
- **When** SessionContextFile.Write is called
- **Then** file is written atomically (via temp file + rename) and `.claude/engram/` directory is created if missing
- **Traces to:** ARCH-31, REQ-44
- **Type:** example-based (file I/O via DI injection)

### T-152: SessionContextFile creates directory if missing

- **Given** `.claude/engram/` directory does not exist
- **When** SessionContextFile.Write is called
- **Then** directory is created with all required parent directories
- **Traces to:** ARCH-31, REQ-44
- **Type:** example-based

### T-153: SessionContextFile returns empty on missing file

- **Given** context file does not exist
- **When** SessionContextFile.Read is called
- **Then** ("", 0, "") is returned (empty summary, offset 0, empty session ID)
- **Traces to:** ARCH-31, REQ-45
- **Type:** example-based

### T-154: ContextUpdateOrchestrator exits 0 on missing transcript file

- **Given** --transcript-path pointing to non-existent file
- **When** ContextUpdateOrchestrator.Run is called
- **Then** exit code 0, no context file written
- **Traces to:** ARCH-32, REQ-40
- **Type:** example-based (fire-and-forget error handling)

### T-155: ContextUpdateOrchestrator skips API call on empty delta

- **Given** a transcript file with no new content (delta empty)
- **When** ContextUpdateOrchestrator.Run is called
- **Then** ContextSummarizer is not called, context file is not written, exit 0
- **Traces to:** ARCH-32, REQ-41
- **Type:** example-based

### T-156: ContextUpdateOrchestrator writes file with updated watermark

- **Given** a transcript file with new lines and existing context file with old offset
- **When** ContextUpdateOrchestrator.Run is called with non-empty delta
- **Then** context file is written with updated byte offset (watermark) in metadata
- **Traces to:** ARCH-32, REQ-42, REQ-44
- **Type:** example-based

### T-157: ContextUpdateOrchestrator exits 0 on API error

- **Given** a mocked ContextSummarizer that returns error
- **When** ContextUpdateOrchestrator.Run is called
- **Then** error is silent, context file is not written, exit 0
- **Traces to:** ARCH-32, REQ-43
- **Type:** example-based

### T-158: Hook integration — context-update runs as separate async hook

- **Given** UserPromptSubmit hooks are configured in hooks.json
- **When** hooks.json is inspected
- **Then** there are two UserPromptSubmit entries: (1) synchronous entry running `user-prompt-submit.sh` (correct + surface), (2) async entry (`"async": true`) running `user-prompt-submit-async.sh` (context-update only). The synchronous script does not contain nohup/disown or background spawning of context-update.
- **Traces to:** ARCH-33, DES-18
- **Type:** example-based (hook configuration verification)

### T-159: Hook integration — PreCompact calls context-update synchronously

- **Given** PreCompact hook is triggered
- **When** hook script runs
- **Then** `engram context-update` is called synchronously (no `&`) and waits for completion before returning
- **Traces to:** ARCH-33, DES-19
- **Type:** example-based (hook script execution)

### T-160: Hook integration — SessionStart reads and injects context

- **Given** context file exists at `.claude/engram/session-context.md` with summary "Task: foo"
- **When** SessionStart hook runs
- **Then** summary is read and injected into hook JSON output additionalContext field, labeled as session resumption context
- **Traces to:** ARCH-33, DES-22, REQ-45
- **Type:** example-based (hook script JSON generation)

---

## L2 → Test Traceability (UC-6)

| L2 Item | Test Coverage |
|---------|--------------|
| REQ-35 | T-122, T-123, T-126, T-127, T-128 |
| REQ-36 | T-124, T-125 |
| REQ-37 | Already covered by T-115 (existing UC-15 annotation test) |
| DES-17 | Already covered by T-115 (existing UC-15 annotation format) |
| REQ-38 | T-129, T-131, T-132, T-133 |
| DES-16 | T-129 |
| REQ-39 | T-130 |

All UC-6 L2 items have test coverage. All ARCH-26..27 items have test coverage.

---

## Proposal Generator (ARCH-36)

### T-171: Working memory within staleness threshold produces no proposal

**Given** a classified memory in the Working quadrant with `updated_at` less than 90 days ago,
**When** Generate is called,
**Then** no proposal is produced for that memory.

- Traces to: ARCH-36, REQ-48
- Type: example-based (boundary: 89 days = no proposal)

### T-172: Working memory beyond staleness threshold produces review proposal

**Given** a classified memory in the Working quadrant with `updated_at` more than 90 days ago,
**When** Generate is called,
**Then** a proposal with `action: "review_staleness"` is produced, including the memory's age in days.

- Traces to: ARCH-36, REQ-48
- Type: example-based (boundary: 91 days = proposal)

### T-173: Leech memory produces LLM-powered rewrite proposal

**Given** a classified memory in the Leech quadrant,
**When** Generate is called with a working LLM caller,
**Then** a proposal with `action: "rewrite"` is produced containing LLM-proposed field changes (keywords, principle, etc.).

- Traces to: ARCH-36, REQ-49, DES-24
- Type: example-based (verify LLM called with memory content + stats, response parsed into proposal)

### T-174: Hidden gem memory produces LLM-powered broadening proposal

**Given** a classified memory in the Hidden Gem quadrant,
**When** Generate is called with a working LLM caller,
**Then** a proposal with `action: "broaden_keywords"` is produced containing proposed keyword additions.

- Traces to: ARCH-36, REQ-50, DES-25
- Type: example-based (verify LLM called, response parsed)

### T-175: Noise memory produces removal proposal with evidence

**Given** a classified memory in the Noise quadrant with surfaced_count=2, effectiveness_score=15.0, evaluation_count=8,
**When** Generate is called,
**Then** a proposal with `action: "remove"` is produced with evidence fields matching the stats.

- Traces to: ARCH-36, REQ-51
- Type: example-based (verify evidence fields match input stats)

### T-176: Insufficient-data memory produces no proposal

**Given** a classified memory with quadrant InsufficientData,
**When** Generate is called,
**Then** no proposal is produced for that memory.

- Traces to: ARCH-36, REQ-47
- Type: example-based (filter check)

### T-177: LLM failure for one memory does not block others

**Given** two leech memories where the LLM caller fails for the first but succeeds for the second,
**When** Generate is called,
**Then** one proposal is returned (for the second memory). The first memory's proposal is omitted.

- Traces to: ARCH-36, REQ-52
- Type: example-based (fire-and-forget error handling)

### T-178: No LLM caller skips leech and hidden gem proposals

**Given** classified memories including leech and hidden gem entries, but no LLM caller configured,
**When** Generate is called,
**Then** only working staleness and noise removal proposals are produced. Leech and hidden gem proposals are skipped.

- Traces to: ARCH-36, REQ-53
- Type: example-based (nil LLM caller behavior)

---

## Maintain CLI Wiring (ARCH-37)

### T-179: maintain subcommand produces JSON proposals to stdout

**Given** a data directory with memories, surfacing logs, and evaluation logs,
**When** `RunMaintain` is called with `--data-dir`,
**Then** stdout contains a JSON array of proposals, each with memory_path, quadrant, diagnosis, action, and details fields.

- Traces to: ARCH-37, REQ-53, DES-23
- Type: example-based (end-to-end CLI wiring)

### T-180: maintain with no evaluation data produces empty array

**Given** a data directory with memories but no evaluation directory,
**When** `RunMaintain` is called,
**Then** stdout contains `[]` and the command exits without error.

- Traces to: ARCH-37, REQ-54
- Type: example-based (no-data behavior)

### T-181: maintain without API key skips LLM proposals

**Given** a data directory with leech and noise memories, but no ANTHROPIC_API_KEY,
**When** `RunMaintain` is called,
**Then** only noise/working proposals appear in output. Leech/hidden-gem proposals are absent.

- Traces to: ARCH-37, REQ-53
- Type: example-based (graceful degradation without API key)

---

## L2 → Test Traceability (UC-16)

| L2 Item | Test Coverage |
|---------|--------------|
| REQ-47 | T-176 |
| REQ-48 | T-171, T-172 |
| REQ-49 | T-173 |
| REQ-50 | T-174 |
| REQ-51 | T-175 |
| REQ-52 | T-177 |
| REQ-53 | T-178, T-179, T-181 |
| REQ-54 | T-180 |
| DES-23 | T-179 |
| DES-24 | T-173 |
| DES-25 | T-174 |

All UC-16 L2 items have test coverage. All ARCH-36..37 items have test coverage.
