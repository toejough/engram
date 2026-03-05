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

## PreToolUse LLM Judgment (ARCH-11)

### T-36: Violated anti-pattern returns violated=true

**Given** a memory with principle "use /commit" and anti_pattern "manual git commit", and a tool call {name: "Bash", input: "git commit -m 'fix'"}, and the LLM returns `{"violated": true}`,
**When** JudgeViolation is called,
**Then** violated is true.

Uses fake HTTP transport returning canned JSON.

- Traces to: ARCH-11, REQ-12

### T-37: Non-violated anti-pattern returns violated=false

**Given** a memory with principle "use /commit" and anti_pattern "manual git commit", and a tool call {name: "Bash", input: "git commit -m 'fix'"}, and the LLM returns `{"violated": false}`,
**When** JudgeViolation is called,
**Then** violated is false.

Uses fake HTTP transport returning canned JSON.

- Traces to: ARCH-11, REQ-12

### T-38: Missing token returns error (not violated)

**Given** no API token configured,
**When** JudgeViolation is called,
**Then** an error is returned and violated is false (graceful degradation — never block without judgment).

- Traces to: ARCH-11, REQ-13

### T-39: LLM timeout returns error (not violated)

**Given** a valid token but the LLM call times out,
**When** JudgeViolation is called,
**Then** an error is returned and violated is false (graceful degradation).

Uses fake HTTP transport that hangs past deadline.

- Traces to: ARCH-11, REQ-13

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

### T-42: Mode tool routes to enforcement pipeline

**Given** the surface subcommand with `--mode tool --tool-name Bash --tool-input '{"command":"git commit"}' --data-dir <tmpdir>`,
**When** Run is called,
**Then** it reads memories, runs pre-filter, runs LLM judgment on candidates, and produces DES-7 block format or empty output.

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
