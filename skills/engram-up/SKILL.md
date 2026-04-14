---
name: engram-up
description: "Use when the user says /engram, /engram-up, \"start engram\", or wants to begin a multi-agent orchestrated session with memory. Shorthand entry point."
---

# Engram Up

**IMMEDIATE ACTION REQUIRED — invoke TWO skills in this exact order:**

1. Use the Skill tool to invoke `engram:use-engram-chat-as` — this loads the chat coordination protocol you need
2. Use the Skill tool to invoke `engram:engram-tmux-lead` — this loads the orchestrator behavior

Do not respond to the user first. Do not ask questions. Invoke both skills immediately, in order. The real instructions are in those skills — this is just the entry point.
