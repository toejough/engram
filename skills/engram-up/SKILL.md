---
name: engram-up
description: "Use when the user says /engram, /engram-up, \"start engram\", or wants to begin a multi-agent orchestrated session with memory."
---

# Engram Up

## Starting a Session

1. **Run dispatch** (keep it running in the foreground — do not background it):
   ```
   engram dispatch start [--agent engram-agent] [--agent <name>...]
   ```
   Dispatch prints the chat file path on startup. Note it — you will need it for observation.

2. **Open a chat observer** (optional, recommended):
   Watch the chat file path printed by `dispatch start`. This gives real-time visibility into all routing decisions.

3. **Signal readiness:** Post your `ready` message per `engram:use-engram-chat-as`.

4. **Assign work:**
   ```
   engram dispatch assign --agent <name> --task '<task description>'
   ```
