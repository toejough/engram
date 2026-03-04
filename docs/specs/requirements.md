# Requirements

Requirements and design items derived from UC-3 (Remember & Correct).

---

## REQ-1: Inline detection via deterministic pattern matching

When a user message is submitted (UserPromptSubmit hook), the system runs it against a correction pattern corpus. If a pattern matches, the message is flagged for LLM enrichment and memory creation.

Initial pattern corpus (15 patterns):
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

- Traces to: UC-3 (detection)
- AC: (1) Pattern corpus is embedded in the binary with at least the 15 patterns above. (2) Pattern matching runs on every invocation of `engram correct`. (3) On match, LLM enrichment is triggered. (4) On no match, empty stdout (no system reminder).
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

The UserPromptSubmit hook is registered in `plugin.json` and invokes `hooks/user-prompt-submit.sh`:
```bash
"$ENGRAM_BIN" correct --message "$CLAUDE_USER_MESSAGE" --data-dir "$ENGRAM_DATA"
```

The hook script reads the OAuth token from the Claude Code Keychain via `security find-generic-password` and exports it as `ENGRAM_API_TOKEN`. Stdout from the binary becomes the system reminder. Empty stdout = no reminder.

- Traces to: UC-3 (hook wiring)
