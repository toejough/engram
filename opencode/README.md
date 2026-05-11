# Engram — OpenCode Plugin

Self-correcting memory for LLM agents. Tracks what works, what doesn't, and what you've learned across sessions.

## Installation

Engram is distributed as source — clone the repo and point OpenCode at the cloned directory. Requires Go 1.25+ on your `PATH`.

1. Clone the repo:

```bash
git clone https://github.com/toejough/engram ~/src/engram
```

2. Install the plugin's runtime dependencies inside the cloned `opencode/` directory:

```bash
cd ~/src/engram/opencode && npm install   # or `bun install`
```

OpenCode resolves the plugin's `import { Plugin } from "@opencode-ai/plugin"` from a `node_modules` directory in the plugin's parent chain, so this `node_modules` must exist next to the plugin's `package.json`. Without it, OpenCode logs `Cannot find module '@opencode-ai/plugin'` at startup and silently disables the plugin.

3. Add the plugin path to `~/.config/opencode/opencode.json`:

```json
{
  "plugin": ["~/src/engram/opencode"]
}
```

That's it. On first session, the plugin compiles `engram` to `~/.local/bin/engram` via `go build`. On subsequent sessions, `engram build-self --if-stale` rebuilds only when source changes. To upgrade, `git pull` in the cloned directory and re-run `npm install` if `opencode/package.json` changed — the next session picks up the change automatically.

## Features

### Commands

| Command | Description |
|---------|-------------|
| `/recall [query]` | Recall recent session context or search memories |
| `/prepare` | Load relevant context before starting new work |
| `/learn` | Review recent session for lessons worth persisting; also the explicit-capture command ("remember this" / "save that for later") |

### Tools

| Tool | Description |
|------|-------------|
| `engram_recall` | Recall recent session context or search memories |
| `engram_learn_feedback` | Learn from behavioral feedback (SBIA format) |
| `engram_learn_fact` | Learn a factual statement (SPO format) |
| `engram_show` | Display full memory details |
| `engram_list` | List all memories |

### Skills

Copies of the core Engram skills are included for self-containment:
- `learn` — How to capture lessons from session outcomes
- `recall` — How to retrieve relevant context
- `prepare` — How to load context before new work
- `remember` — How to persist knowledge across sessions

## Binary

The plugin builds the `engram` binary at `~/.local/bin/engram` on first session. Bootstrap uses `go build` (chicken-and-egg: `engram build-self` can't run before the binary exists). Subsequent rebuilds are handled by `engram build-self --if-stale` which re-runs only when Go source files are newer than the binary.

Requires Go 1.25+ on `PATH`. If you need to rebuild manually:

```bash
go build -o ~/.local/bin/engram ./cmd/engram/
```

## Configuration

### Environment variables

| Variable | Description |
|----------|-------------|
| `ENGRAM_API_TOKEN` | Anthropic API token for memory classification |
| `ENGRAM_DATA_DIR` | Override data directory (default: `~/.local/share/engram`) |
| `ENGRAM_TRANSCRIPT_DIR` | Override transcript directory for recall |

### CLI flags

| Flag | Description |
|------|-------------|
| `--transcript-dir` | Override transcript directory |
| `--no-external-sources` | Skip CLAUDE.md, rules, auto-memory, skills discovery |
| `--data-dir` | Path to data directory |
| `--memories-only` | Search only memory files |
| `--query` | Search query for recall |
| `--limit` | Max memories to return (default 10) |
