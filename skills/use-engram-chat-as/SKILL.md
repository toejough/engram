---
name: use-engram-chat-as
description: Use when independently-launched agents need to coordinate, when joining a multi-agent session, when told to communicate via engram chat, when using /use-engram-chat-as. Agents broadcast intent before acting and wait for objections before proceeding. Symptoms: agents missing messages, needing to coordinate before acting, multiple agents in separate terminals.
---

# Engram Chat Protocol

The binary owns all mechanics (cursor tracking, file writes, ack-wait, process lifecycle). Agents own judgment: WHEN to use each protocol element, HOW to argue, WHAT counts as a fact.

**Agent roles:** Active agents broadcast intent before acting and wait for ACK or WAIT from all TO recipients. Reactive agents never broadcast their own intent — only react to others. Active agents MUST parrot user input as `info` messages so reactive agents see corrections.

## Prefix Markers (Headless Workers)

| Prefix | When |
|--------|------|
| `READY:` | First output after launch |
| `INTENT: TO: name1, name2\nSituation: X.\nBehavior: Y.` | Before any significant action |
| `ACK:` | No objection |
| `WAIT:` | Objection or relevant context — state concern on same line |
| `DONE:` | Task complete |
| `LEARNED:` | Reusable fact for engram-agent |
| `INFO:` | Status update |
| `ESCALATE:` | Unresolved argument needs lead |

**HARD RULE: After `INTENT:`, end your turn immediately.** Do not act in the same turn. Next turn delivers `Proceed.` or `WAIT from <agent>: [text]`.

**TO field:** Always include `engram-agent` (memory must see every intent). Binary defaults to `engram-agent` if TO: absent.

## Intent Protocol

Use before: modifying shared files, running build/test/coverage tools, committing/pushing, architectural decisions. Skip for: reading files, searching, informational messages.

## RESUME_REASON Handling

Every resume prompt includes `RESUME_REASON`:
- `shutdown` — say `DONE:` and stop immediately.
- `wait` — read `WAIT_FROM`, `WAIT_TEXT`, `ARGUMENT_TURN`. Engage argument protocol as initiator (factual defense or `ACK:` to concede; `ESCALATE:` if turn ≥ 3 and unresolved).
- `intent` — process normally.

If WAIT arrives while still in-session (ACTIVE), respond in the same turn. Do not complete the task first.

## Argument Protocol

- **Initiator** (whose intent was challenged): factual response — state reasoning and evidence.
- **Reactor** (who posted WAIT): aggressive response — push back hard. Agents default to thinking well of their own work; the reactor counterbalances this.

Weak reactor: "Maybe you're right, I'll defer." ← concession without engagement

Strong reactor: "Your reasoning assumes the file is unlocked, but I see no done message in chat. Show me the done message or I'm holding." ← factual challenge with specific ask

3-input cap: objection → response → counter. Still unresolved → `ESCALATE:` to lead (or initiating agent's UX if no lead). Reactor posts `info` recording the resolution.

## Shutdown

On `shutdown` or `RESUME_REASON=shutdown`: complete in-flight work, post `DONE:`, exit. Accept no new work after shutdown.
