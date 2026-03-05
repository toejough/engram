# Requirements

Requirements and design items derived from UC-3 (Remember & Correct) and UC-2 (Hook-Time Surfacing & Enforcement).

---

## REQ-1: Inline detection via deterministic pattern matching

When a user message is submitted (UserPromptSubmit hook), the system runs it against a correction pattern corpus. If a pattern matches, the message is flagged for LLM enrichment and memory creation.

Pattern corpus (40 patterns across 2 tiers):

**Original patterns (15):**
1. `^no,` — direct negation
2. `^wait` — interruption
3. `^hold on` — interruption
4. `\bwrong\b` — explicit wrongness
5. `\bdon't\s+\w+` — prohibition (don't + verb)
6. `\bstop\s+\w+ing` — cease action (stop + gerund)
7. `\btry again` — retry request
8. `\bgo back` — reversal request
9. `\bthat's not` — negation of output
10. `^actually,` — correction opener
11. `\bremember\s+(that|to)` — re-teaching
12. `\bstart over` — full reset
13. `\bpre-?existing` — flagging prior state
14. `\byou're still` — persistent error
15. `\bincorrect` — explicit wrongness

**Expanded patterns (10, issue #23):**
16. `\bfrom\s+now\s+on\b` — standing instruction (tier A)
17. `\byou\s+should\s+have\b` — retrospective correction
18. `\byou\s+(forgot|overlooked)\s+to\b` — omission feedback
19. `\byou\s+missed\b` — omission (broad)
20. `\bI\s+(told|already\s+told)\s+you\b` — repeated instruction
21. `\bI\s+already\s+(said|asked|mentioned)\b` — repeated request
22. `\brather\s+than\b` — contrast/preference
23. `\bnot\s+\w+,?\s+(but|instead)\b` — contrast correction
24. `\bthat's\s+not\s+what\s+I\b` — explicit rejection
25. `\bnext\s+time\b` — prospective correction

**New patterns (15, issue #24):**

*Scope / Over-engineering:*
26. `\bjust\s+wanted\b` — scope complaint ("I just wanted X")
27. `\bover-?engineer` — explicit over-engineering complaint
28. `\bI\s+only\s+asked\b` — scope restriction ("I only asked for X")

*Quality Complaints:*
29. `\bdoes(?:n't| not)\s+work\b` — broken output ("doesn't work")
30. `\bit(?:'s| is)\s+broken\b` — broken output ("it's broken")
31. `\bnot\s+working\b` — broken output ("not working")

*Style / Convention:*
32. `\bwe\s+use\b` — team convention signal ("we use X")
33. `\bthe\s+convention\b` — explicit convention reference
34. `\bin\s+this\s+(?:project|repo|codebase)\b` — project-scoped norm

*Permission Boundaries:*
35. `\bleave\s+\w+\s+alone\b` — hands-off signal ("leave it alone")
36. `\bhands\s+off\b` — prohibition signal
37. `\boff\s+limits\b` — prohibition signal

*Confusion / Misunderstanding:*
38. `\byou\s+misunderstood\b` — explicit misunderstanding
39. `\bno,?\s+I\s+mean\b` — clarification after misparse ("no I mean...")
40. `\bmisinterpreted\b` — explicit misinterpretation

- Traces to: UC-3 (detection)
- AC: (1) Pattern corpus is embedded in the binary with at least the 40 patterns above. (2) Pattern matching runs on every invocation of `engram correct`. (3) On match, LLM enrichment is triggered. (4) On no match, empty stdout (no system reminder).
- Verification: deterministic (pattern match)

---

## REQ-2: LLM enrichment produces structured memory fields

When a pattern match is detected, a single API call to claude-haiku-4-5-20251001 takes the user's message and produces structured memory fields as JSON: title, content, observation_type, concepts, keywords, principle, anti_pattern, rationale, and a 3-5 word filename summary.

- Traces to: UC-3 (LLM enrichment)
- AC: (1) API call uses claude-haiku-4-5-20251001. (2) Response is parsed as JSON with all required fields. (3) Invalid or unparseable responses return an error.
- Verification: deterministic (JSON schema validation of LLM response)

---

## REQ-3: Enriched memory written as TOML file

The enriched memory is written to `<data-dir>/memories/<slug>.toml` where slug is the slugified filename summary (3-5 hyphenated lowercase words). The TOML file contains all structured fields plus confidence tier and timestamps.

- Traces to: UC-3 (TOML file output)
- AC: (1) File is written to the memories subdirectory. (2) Filename is 3-5 hyphenated words, lowercase, `.toml` extension. (3) TOML contains: title, content, observation_type, concepts (array), keywords (array), principle, anti_pattern, rationale, confidence, created_at (RFC 3339), updated_at (RFC 3339). (4) File is valid TOML, human-readable, and hand-editable.
- Verification: deterministic (file exists, TOML parses, fields present)

---

## REQ-4: System reminder feedback on memory creation

After a memory file is created, the system outputs a system reminder to stdout confirming: what was detected, the memory title, key fields, and the file path.

- Traces to: UC-3 (feedback)
- AC: (1) Stdout contains a system reminder when a memory is created. (2) Reminder includes the memory title, observation type, and file path. (3) Format uses `[engram]` prefix.
- Verification: deterministic (stdout content check)

---

## REQ-6: Go binary with `correct` subcommand

The system is implemented as a Go binary (`engram`) with a `correct` subcommand: `engram correct --message <text> --data-dir <path>`. Pure Go, no CGO.

- Traces to: UC-3 (Go binary CLI)
- AC: (1) Binary compiles with `CGO_ENABLED=0`. (2) `engram correct --message <text> --data-dir <path>` runs the detection → enrichment → write → feedback pipeline. (3) Exit 0 always — errors logged to stderr, never propagated as exit codes.
- Verification: deterministic (build succeeds, subcommand runs)

---

## REQ-7: Confidence tier assignment

Each memory is assigned a confidence tier: A (user explicitly stated — "remember X" patterns), B (user correction — "no, do Y" patterns), or C (agent-inferred post-session — extracted by LLM from transcript, user never saw the extraction). The tier is determined by the source: UC-3 pattern matching produces A or B, UC-1 session extraction always produces C.

- Traces to: UC-3 (confidence tiers), UC-1 (confidence tier C)
- AC: (1) Patterns `\bremember\s+(that|to)` and `\bfrom\s+now\s+on\b` produce confidence A. (2) All other correction patterns produce confidence B. (3) Session-extracted learnings (UC-1) produce confidence C. (4) Confidence is written to the TOML file.
- Verification: deterministic (tier matches source)

---

## DES-1: Correction feedback reminder format

When a memory is created, the agent sees:

```
<system-reminder source="engram">
[engram] Memory captured.
  Created: "<title>"
  Type: <observation_type>
  File: <file_path>
</system-reminder>
```

Format rules:
- Header: `[engram] Memory captured.`
- Action: `Created:` with memory title in quotes
- Type: the observation_type field
- File: relative path to the TOML file
- Concise — appears in the same hook response

- Traces to: UC-3 (feedback)

---

## DES-3: Hook wiring — UserPromptSubmit

The UserPromptSubmit hook is registered in `hooks/hooks.json` and invokes `hooks/user-prompt-submit.sh`. The hook reads the user prompt from stdin JSON (`{"prompt": "..."}` via `jq -r '.prompt // empty'`) and passes it to the binary:
```bash
USER_MESSAGE="$(jq -r '.prompt // empty')"
"$ENGRAM_BIN" correct --message "$USER_MESSAGE" --data-dir "$ENGRAM_DATA"
```

The hook also self-builds the binary if missing (`go build -o "$ENGRAM_BIN" ./cmd/engram/`).

Token retrieval is platform-aware:
- **macOS:** Attempt to read OAuth token from Claude Code Keychain via `security find-generic-password`. On failure, fall back to `ENGRAM_API_TOKEN` env var.
- **Non-macOS (Linux, etc.):** Use `ENGRAM_API_TOKEN` env var directly.

The hook exports `ENGRAM_API_TOKEN` from whichever source succeeds. Stdout from the binary becomes the system reminder. Empty stdout = no reminder.

- Traces to: UC-3 (hook wiring)

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

## REQ-11: PreToolUse keyword pre-filter

When a tool call is about to execute (PreToolUse hook), the system scans memory TOML files for keyword matches against the tool name and tool input. Only memories with an `anti_pattern` field are candidates.

- Traces to: UC-2 (PreToolUse enforcement, pass 1)
- AC: (1) Only memories with a non-empty `anti_pattern` field are scanned. (2) Each candidate memory's `keywords` are checked for whole-word matches (case-insensitive) in the tool name or tool input arguments. (3) Memories with at least one keyword match are passed to the LLM judgment stage. (4) If no memories match, the tool call is allowed with no overhead (no LLM call, no output).
- Verification: deterministic (keyword presence check)

---

## REQ-12: PreToolUse LLM judgment

For each memory that passes the keyword pre-filter (REQ-11), the system makes a single LLM call (claude-haiku-4-5-20251001) to determine whether the tool call violates the memory's anti-pattern.

- Traces to: UC-2 (PreToolUse enforcement, pass 2)
- AC: (1) LLM receives: tool name, tool input, memory principle, memory anti_pattern. (2) LLM returns a structured JSON decision: `{"violated": true/false, "reasoning": "..."}`. (3) If violated → the tool call is blocked. (4) If not violated → the tool call is allowed silently. (5) If multiple memories match the pre-filter, each gets a separate judgment call; any violation blocks.
- Verification: deterministic (JSON schema of LLM response)

---

## REQ-13: Graceful degradation with notification

If no API token is configured or the LLM judgment call times out, the system allows the tool call and emits a stderr warning.

- Traces to: UC-2 (graceful degradation)
- AC: (1) Missing token → allow tool call, emit `[engram] Warning: enforcement skipped (no API token). Tool call allowed.` to stderr. (2) LLM timeout → allow tool call, emit `[engram] Warning: enforcement skipped (timeout). Tool call allowed.` to stderr. (3) Never block when judgment is unavailable.
- Verification: deterministic (stderr output check)

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

## DES-7: PreToolUse block response format

When a tool call is blocked, the hook returns:

```json
{"decision": "block", "reason": "[engram] Blocked: \"<title>\" — <principle>. Memory file: <file_path>"}
```

Format rules:
- `decision` is `"block"` (Claude Code PreToolUse protocol)
- `reason` includes `[engram]` prefix, memory title in quotes, the principle, and the file path
- If no violation, no output (allow silently)

- Traces to: UC-2 (PreToolUse enforcement feedback)

---

## DES-8: Hook wiring — SessionStart and PreToolUse

SessionStart hook (`hooks/session-start.sh`) is extended to call `engram surface --mode session-start` after the existing build step. PreToolUse hook is registered in `hooks/hooks.json` and invokes `hooks/pre-tool-use.sh`, which reads the tool call from stdin JSON and calls `engram surface --mode tool`.

UserPromptSubmit hook (`hooks/user-prompt-submit.sh`) is extended to also call `engram surface --mode prompt` alongside the existing `engram correct` call. Both outputs are concatenated to stdout.

- Traces to: UC-2 (hook wiring)

---

# UC-1 Requirements and Design

---

## REQ-15: LLM session transcript extraction

When a PreCompact or SessionEnd hook fires, the system sends the session transcript to an LLM (claude-haiku-4-5-20251001) which extracts a list of candidate learnings. Each candidate has the same structured fields as UC-3 memories: title, content, observation_type, concepts, keywords, principle, anti_pattern, rationale, filename_summary.

- Traces to: UC-1 (LLM extraction)
- AC: (1) API call uses claude-haiku-4-5-20251001. (2) The prompt includes the full transcript (or the portion about to be compacted for PreCompact). (3) Response is parsed as a JSON array of candidate objects, each with all required fields. (4) Invalid or unparseable responses return an error. (5) The LLM extracts: missed corrections, architectural decisions, discovered constraints, working solutions, and implicit preferences.
- Verification: deterministic (JSON schema validation of LLM response)

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

If no API token is configured, the system emits a loud stderr warning and does not create any memory files. No degraded memories are ever written.

- Traces to: UC-1 (no graceful degradation)
- AC: (1) Missing API token → emit `[engram] Error: session learning skipped — no API token configured` to stderr. (2) No TOML files are created. (3) Exit 0 (don't break the hook chain). (4) See also issue #32 for aligning UC-3.
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
[engram] Extracted N learnings from session.
  - "<title1>" (file1.toml)
  - "<title2>" (file2.toml)
  ...
[engram] Skipped M duplicates.
```

Format rules:
- Header: `[engram] Extracted N learnings from session.`
- Each learning: title in quotes, file path in parentheses
- Duplicate count: only if M > 0
- If zero learnings after dedup, emit: `[engram] No new learnings extracted.`

- Traces to: UC-1 (feedback)
