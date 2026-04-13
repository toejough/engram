# Engram Server & MCP — Plan Decomposition

**Spec:** `docs/superpowers/specs/2026-04-12-engram-server-mcp-design.md`

## Why sub-stages?

The spec defines 3 stages (1, 1.5, 2). Stage 1 was split into sub-stages during planning because the writing-plans skill produces detailed step-by-step plans with full code — a single plan for all of Stage 1 would have been 3000+ lines and unmanageable for subagent execution.

The split is by **implementation dependency**, not by architectural isolation:
- 0: CLI client — must exist before the server can be tested e2e
- 1a: Server core — must exist before the engram-agent can be wired in
- 1b: Engram-agent — must exist before hooks make sense (hooks trigger the agent)
- 1c+1d+1.5: Hooks, skill rewrites, old code deletion — these are independent of each other and could be done in any order or collapsed into one stage

## Plan Sequence

| # | Plan | Status | What it delivers |
|---|------|--------|-----------------|
| 0 | CLI client commands | DONE | HTTP client library + new CLI commands |
| 1a | API server core | DONE | HTTP server, chat file watching, goroutine fan-out, validation, skill refresh |
| 1b | Engram-agent management | DONE | claude -p lifecycle, stream parser, error recovery |
| 1c+1d+1.5 | Hooks + skill rewrites + retire old code | TODO | Hook scripts, 5 skill rewrites, delete old dispatch/chat/agent commands |
| 2 | MCP server | TODO | MCP server wrapping the API, async push |
| 3 | Observability tuning | TODO | Debug log refinement, skill contract adjustments |

## Remaining work

**1c+1d+1.5** can be done as a single stage:
- **Hooks** (1c): 3 shell scripts in hooks.json — small, mechanical
- **Skill rewrites** (1d): 5 SKILL.md files — content work, not code
- **Retire old code** (1.5): delete old commands and backing code — mechanical deletion

These have no code dependencies between them and can be a single plan with 3 sections.
