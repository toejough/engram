---
name: use-engram-chat-as
description: Use when independently-launched agents need to coordinate via engram, when joining a multi-agent session, or when told to communicate via engram chat. Covers the CLI interaction model, learn message format, and skill refresh protocol.
---

# Engram Chat Protocol

The API server owns all mechanics (file writes, routing, validation, lifecycle). Agents own judgment: WHEN to announce intent, WHAT constitutes a reusable learning, HOW to structure learn messages.

**Hooks handle posting automatically.** `UserPromptSubmit` and `Stop` hooks call `engram intent` on every user message and agent turn. `SubagentStop` hooks call `engram post` for subagent output. You do not need to call these yourself for routine turns — hooks cover it.

## CLI Commands

Use these for explicit interaction with the engram API server:

```bash
# Announce intent before significant actions (synchronous — blocks until memories returned)
engram intent --from <agent-name> --to engram-agent \
  --situation "<what you're about to do>" \
  --planned-action "<specific action>"

# Post a general message (fire-and-forget)
engram post --from <agent-name> --to <recipient> --text "<message>"

# Record a structured learning
engram learn --from <agent-name> --type feedback \
  --situation "<context>" \
  --behavior "<what happened>" \
  --impact "<what it caused>" \
  --action "<what to do instead>"

engram learn --from <agent-name> --type fact \
  --situation "<context>" \
  --subject "<entity>" \
  --predicate "<relationship>" \
  --object "<value>"

# Subscribe to messages addressed to you (cursor-based, resumable)
engram subscribe --agent <agent-name> [--after-cursor <N>]

# Check server health
engram status
```

## Learn Message Format

The server validates `engram learn` messages against the memory format.

**Feedback** (corrections, behavior patterns):
- Required fields: `situation`, `behavior`, `impact`, `action`

**Fact** (subject-predicate-object assertions):
- Required fields: `situation`, `subject`, `predicate`, `object`

Validation errors include format guidance. Retry with corrected fields (up to 3 attempts). After 3 failures the server accepts the raw content and asks engram-agent to interpret it best-effort.

## When to Call `engram intent` Explicitly

Hooks cover routine turns. Call `engram intent` explicitly when:
- About to make an architectural decision
- About to modify a shared or critical file
- About to commit or push
- Starting a significant subtask where prior failures might be relevant

`engram intent` blocks until the engram-agent responds with surfaced memories. Read those memories before proceeding.

## Skill Refresh

The server tracks message counts per agent. Every 13 messages delivered to you, the server posts a refresh reminder to chat:

> "Reload your engram skills: `/use-engram-chat-as` and `/engram-lead`."

When you receive this message, reload both skills using the Skill tool before continuing.

## What Is Retired

The following patterns are **retired**, not deferred:
- `INTENT:` / `ACK:` / `WAIT:` / `DONE:` / `LEARNED:` speech markers
- `engram chat watch` subprocess spawning
- Cursor tracking by agents
- Ack-wait blocking protocol
- Argument protocol (3-turn cap, `ESCALATE:`)
- `RESUME_REASON` handling
- `engram dispatch` commands

The server handles routing, serialization, and retry. Agents use CLI commands.

## Troubleshooting

Debug logging is available at the server log file (specified with \`--log-file\` on \`engram server up\`). If engram is not working as expected, check the server log: \`tail -f <log-file> | jq .\`
