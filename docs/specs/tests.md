# Tests

Behavioral test list for UC-3 (Remember & Correct) and UC-2 (Hook-Time Surfacing & Enforcement). BDD Given/When/Then format. Default property-based via rapid; example-based justified inline.

---

## Pattern Matching (ARCH-2)

### T-1: Correction pattern matches

**Given** a message matching any of the 40 correction patterns,
**When** Match is called,
**Then** a PatternMatch is returned with the matched pattern's label.

Property-based: generate messages containing each pattern. All must match.

- Traces to: ARCH-2, REQ-1

### T-2: Non-matching message returns nil

**Given** a message containing no correction/remember patterns,
**When** Match is called,
**Then** nil is returned.

Property-based: generate arbitrary strings that don't contain any pattern.

- Traces to: ARCH-2, REQ-1

### T-3: Remember/standing-instruction patterns produce confidence A

**Given** a message matching `\bremember\s+(that|to)` or `\bfrom\s+now\s+on\b`,
**When** Match is called,
**Then** PatternMatch.Confidence is "A".

- Traces to: ARCH-2, REQ-7

### T-4: Correction patterns produce confidence B

**Given** a message matching any correction pattern except "remember",
**When** Match is called,
**Then** PatternMatch.Confidence is "B".

Property-based: generate messages for each non-remember pattern.

- Traces to: ARCH-2, REQ-7

---

## LLM Enrichment (ARCH-3)

### T-5: Enrichment with token produces all structured fields

**Given** a message, pattern match, and a valid OAuth token,
**When** Enrich is called,
**Then** an EnrichedMemory is returned with all fields populated: title, content, observation_type, concepts, keywords, principle, anti_pattern, rationale, filename_summary, confidence, timestamps. The HTTP request uses `Authorization: Bearer` header (not `X-Api-Key`).

Uses fake HTTP transport returning canned JSON.

- Traces to: ARCH-3, REQ-2

### T-6: Enrichment without token returns error

**Given** a message and pattern match but no token,
**When** Enrich is called,
**Then** ErrNoToken is returned and no HTTP call is made.

- Traces to: ARCH-3, REQ-2

### T-7: Invalid LLM response returns error

**Given** a message and pattern match, and the LLM returns invalid JSON,
**When** Enrich is called,
**Then** an error is returned (not a degraded memory).

- Traces to: ARCH-3, REQ-2

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

### T-13: Memory produces DES-1 format

**Given** an EnrichedMemory and file path,
**When** Render is called,
**Then** output matches DES-1 format: `[engram] Memory captured.` header, Created/Type/File fields.

- Traces to: ARCH-5, REQ-4, DES-1

---

## Pipeline (ARCH-1)

### T-15: Full pipeline — match → enrich → write → render

**Given** a message matching a pattern, with all pipeline stages wired,
**When** Run is called,
**Then** the stages execute in order and a system reminder string is returned.

Uses fakes for all four DI interfaces. Verifies call order.

- Traces to: ARCH-1, REQ-1, REQ-2, REQ-3, REQ-4

### T-16: No match — pipeline short-circuits

**Given** a message that doesn't match any pattern,
**When** Run is called,
**Then** empty string is returned and Enricher/Writer/Renderer are never called.

- Traces to: ARCH-1, REQ-1

---

## CLI Wiring (ARCH-6)

### T-18: `correct` subcommand without token returns error

**Given** `engram correct --message "remember to use targ" --data-dir <tmpdir>` with no `ENGRAM_API_TOKEN` set,
**When** Run is called,
**Then** an error containing "no API token" is returned.

- Traces to: ARCH-6, REQ-6

### DES-3: Static hook script matches expected content

**Given** the static hook script at `hooks/user-prompt-submit.sh`,
**When** its content is read,
**Then** it references `correct`, `bin/engram`, `jq`, `.prompt`, `CLAUDE_PLUGIN_ROOT`, and `ENGRAM_API_TOKEN`.

- Traces to: ARCH-6, DES-3

---

### T-19: `correct` with non-matching message produces empty stdout

**Given** `engram correct --message "hello world" --data-dir <tmpdir>`,
**When** Run is called,
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

---

## Hook Script Integration (ARCH-13)

### T-43: SessionStart hook calls surface after build

**Given** the session-start hook script,
**When** its content is read,
**Then** it calls `engram surface --mode session-start` after the build step.

- Traces to: ARCH-13, DES-8

### T-44: UserPromptSubmit hook calls both correct and surface

**Given** the user-prompt-submit hook script,
**When** its content is read,
**Then** it calls both `engram correct` and `engram surface --mode prompt`, concatenating their outputs.

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

# UC-1 Tests

## Transcript Extraction (ARCH-15)

### T-47: Extraction with token produces CandidateLearnings with all fields

**Given** a transcript string and a valid OAuth token,
**When** test calls Extract,
**Then** Extract returns a non-empty slice of CandidateLearning, each with all fields populated: title, content, observation_type, concepts, keywords, principle, anti_pattern, rationale, filename_summary. The HTTP request uses `Authorization: Bearer` header with `Anthropic-Beta: oauth-2025-04-20`.

Uses fake HTTP transport returning canned JSON array.

- Traces to: ARCH-15, REQ-15
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

### T-60: Written memories have confidence tier C

**Given** a transcript, with Extractor returning candidates (no confidence field set),
**When** test calls Learner.Run,
**Then** every memory passed to Writer has Confidence = "C".

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
| ARCH-14 | T-57, T-58, T-59, T-60 |
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

All L2C items have test coverage. All ARCH-14–18 items have test coverage.
