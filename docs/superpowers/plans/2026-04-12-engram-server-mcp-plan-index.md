# Engram Server & MCP — Plan Decomposition

**Spec:** `docs/superpowers/specs/2026-04-12-engram-server-mcp-design.md`

## Plan Sequence

Each plan produces working, testable software and is a prerequisite for the next.

| # | Plan | What it delivers |
|---|------|-----------------|
| 0 | Stage 0: CLI client commands | HTTP client library + new CLI commands (post, intent, learn, subscribe, status) |
| 1a | Stage 1a: API server core | HTTP server, chat file watching, goroutine fan-out, validation, skill refresh |
| 1b | Stage 1b: Engram-agent management | claude -p lifecycle, new stream parser, structured output contract, error recovery |
| 1c | Stage 1c: Hooks | UserPromptSubmit, Stop, SubagentStop hook scripts |
| 1d | Stage 1d: Skill rewrites | engram-agent, use-engram-chat-as, engram-lead, engram-up, engram-down |
| 1.5 | Stage 1.5: Retire old CLI/dispatch | Delete old commands, dispatch, holds, agent spawn/kill, backing code |
| 2 | Stage 2: MCP server | MCP server wrapping the API, async push, engram_intent/learn/post tools |
| 3 | Stage 3: Observability tuning | Debug log refinement, skill contract adjustments from real-world usage |
