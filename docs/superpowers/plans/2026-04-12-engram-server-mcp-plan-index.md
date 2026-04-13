# Engram Server & MCP — Plan Decomposition

**Spec:** `docs/superpowers/specs/2026-04-12-engram-server-mcp-design.md`

**Status: COMPLETE** — All stages implemented and merged to main.

## Plan Sequence

| # | Plan | Status | What it delivered |
|---|------|--------|-----------------|
| 0 | CLI client commands | DONE | HTTP client library + new CLI commands |
| 1a | API server core | DONE | HTTP server, chat file watching, goroutine fan-out, validation, skill refresh |
| 1b | Engram-agent management | DONE | claude -p lifecycle, stream parser, error recovery |
| 1c+1d+1.5 | Hooks + skill rewrites + retire old code | DONE | Hook scripts, 5 skill rewrites, 13k lines deleted |
| 2 | MCP server | DONE | MCP tools, channel push via notifications/claude/channel, auto-start |
| 3 | Observability tuning | DONE | Debug logging notes in all skills |
