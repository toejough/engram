# Tests

Behavioral test list for UC-3 (Remember & Correct). BDD Given/When/Then format. Default property-based via rapid; example-based justified inline.

---

## Pattern Matching (ARCH-2)

### T-1: Correction pattern matches

**Given** a message matching any of the 15 correction patterns,
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

### T-3: Remember patterns produce confidence A

**Given** a message matching `\bremember\s+(that|to)`,
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

### T-5: Enrichment with API key produces all structured fields

**Given** a message, pattern match, and a valid API key,
**When** Enrich is called,
**Then** an EnrichedMemory is returned with all fields populated: title, content, observation_type, concepts, keywords, principle, anti_pattern, rationale, filename_summary, confidence, timestamps.

Uses fake HTTP transport returning canned JSON.

- Traces to: ARCH-3, REQ-2

### T-6: Enrichment without API key produces degraded memory

**Given** a message and pattern match but no API key,
**When** Enrich is called,
**Then** an EnrichedMemory is returned with: title = first ~60 chars, content = full message, observation_type = match label, enrichment fields empty, filename_summary = first 3-5 words.

- Traces to: ARCH-3, REQ-5

### T-7: Invalid LLM response falls back to degraded memory

**Given** a message and pattern match, and the LLM returns invalid JSON,
**When** Enrich is called,
**Then** a degraded EnrichedMemory is returned (same as no API key).

- Traces to: ARCH-3, REQ-2, REQ-5

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

### T-13: Normal memory produces DES-1 format

**Given** a normal (non-degraded) EnrichedMemory and file path,
**When** Render is called with degraded=false,
**Then** output matches DES-1 format: `[engram] Memory captured.` header, Created/Type/File fields.

- Traces to: ARCH-5, REQ-4, DES-1

### T-14: Degraded memory produces DES-2 format

**Given** a degraded EnrichedMemory and file path,
**When** Render is called with degraded=true,
**Then** output matches DES-2 format: `[engram] Memory captured (degraded — no API key).` header, Created/File/Note fields.

- Traces to: ARCH-5, REQ-4, DES-2

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

### T-17: Pipeline with degraded enrichment

**Given** a matching message and an enricher that returns a degraded memory,
**When** Run is called,
**Then** Writer and Renderer still execute, producing a degraded memory file and DES-2 format feedback.

- Traces to: ARCH-1, REQ-5

---

## CLI Wiring (ARCH-6)

### T-18: `correct` subcommand runs pipeline

**Given** `engram correct --message "remember to use targ" --data-dir <tmpdir>`,
**When** Run is called,
**Then** a TOML file is created in `<tmpdir>/memories/` and stdout contains a system reminder.

Integration-style: uses real PatternMatcher, fake Enricher (no real API call), real Writer, real Renderer.

- Traces to: ARCH-6, REQ-6, DES-3

### T-19: `correct` with non-matching message produces empty stdout

**Given** `engram correct --message "hello world" --data-dir <tmpdir>`,
**When** Run is called,
**Then** stdout is empty and no file is created.

- Traces to: ARCH-6, REQ-6
