# Requirements

Requirements and design items derived from UC-3 (Remember & Correct).

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
35. `\bleave\s+\w+\s+alone\b` — hands-off signal ("leave X alone")
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

Each memory is assigned a confidence tier: A (user explicitly stated — "remember X" patterns) or B (user correction — "no, do Y" patterns). The tier is determined by which pattern matched.

- Traces to: UC-3 (confidence tiers)
- AC: (1) Patterns `\bremember\s+(that|to)` produce confidence A. (2) All other correction patterns produce confidence B. (3) Confidence is written to the TOML file.
- Verification: deterministic (tier matches pattern type)

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
