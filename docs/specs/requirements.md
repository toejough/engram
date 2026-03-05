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

## REQ-4: System reminder feedback on memory creation

After a memory file is created, the system outputs a system reminder to stdout confirming: the tier classification, the memory title, key fields, and the file path.

- Traces to: UC-3 (feedback)
- AC: (1) Stdout contains a system reminder when a memory is created. (2) Reminder includes the memory tier (A/B/C), title, observation type, and file path. (3) Format uses `[engram]` prefix. (4) Null classification → no output (no file created).
- Verification: deterministic (stdout content check, tier accuracy)

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

The UserPromptSubmit hook is registered in `hooks/hooks.json` and invokes `hooks/user-prompt-submit.sh`. The hook reads the user prompt and transcript path from stdin JSON:
```bash
USER_MESSAGE="$(jq -r '.prompt // empty')"
TRANSCRIPT_PATH="$(jq -r '.transcript_path // empty')"
"$ENGRAM_BIN" correct --message "$USER_MESSAGE" --data-dir "$ENGRAM_DATA"
```

The `transcript_path` field is available in the UserPromptSubmit hook JSON input and is read by the Go binary for context (REQ-X). The hook also self-builds the binary if missing (`go build -o "$ENGRAM_BIN" ./cmd/engram/`).

Token retrieval is platform-aware:
- **macOS:** Attempt to read OAuth token from Claude Code Keychain via `security find-generic-password`. On failure, fall back to `ENGRAM_API_TOKEN` env var.
- **Non-macOS (Linux, etc.):** Use `ENGRAM_API_TOKEN` env var directly.

The hook exports `ENGRAM_API_TOKEN` from whichever source succeeds. Stdout from the binary becomes the system reminder. Empty stdout = no reminder.

- Traces to: UC-3 (hook wiring, transcript context reading)

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

The system adds a `surface` subcommand to the engram binary: `engram surface --mode <session-start|prompt|tool> --data-dir <path>`. Mode-specific flags: `--message <text>` for prompt mode, `--tool-name <name> --tool-input <json>` for tool mode.

- Traces to: UC-2 (CLI entry point)
- AC: (1) `engram surface --mode session-start --data-dir <path>` runs SessionStart surfacing. (2) `engram surface --mode prompt --message <text> --data-dir <path>` runs UserPromptSubmit surfacing. (3) `engram surface --mode tool --tool-name <name> --tool-input <json> --data-dir <path>` runs PreToolUse enforcement. (4) Exit 0 always — errors logged to stderr.
- Verification: deterministic (subcommand runs, correct mode dispatch)

---

## DES-5: SessionStart surfacing reminder format

When memories are surfaced at session start, the agent sees:

```
<system-reminder source="engram">
[engram] Loaded N memories.
  - "<title1>" (file1.toml)
  - "<title2>" (file2.toml)
  ...
</system-reminder>
```

Format rules:
- Header: `[engram] Loaded N memories.` where N is the count
- Each memory: title in quotes, file path in parentheses
- Ordered by recency (most recent first)
- If no memories, no output

- Traces to: UC-2 (SessionStart feedback)

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
- Appears alongside any UC-3 correction output (concatenated)

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

- Traces to: UC-2 (PreToolUse advisory surfacing, tier-aware filtering)

---

## DES-8: Hook wiring — SessionStart and PreToolUse

SessionStart hook (`hooks/session-start.sh`) is extended to call `engram surface --mode session-start` after the existing build step. PreToolUse hook is registered in `hooks/hooks.json` and invokes `hooks/pre-tool-use.sh`, which reads the tool call from stdin JSON and calls `engram surface --mode tool`.

UserPromptSubmit hook (`hooks/user-prompt-submit.sh`) is extended to also call `engram surface --mode prompt` alongside the existing `engram correct` call. Both outputs are concatenated to stdout.

- Traces to: UC-2 (hook wiring)

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
