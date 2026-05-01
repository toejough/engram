# Engram — OpenCode Plugin

Self-correcting memory for LLM agents. Tracks what works, what doesn't, and what you've learned across sessions.

## Installation

### Local installation

1. Add the plugin to your `opencode.json`:

```json
{
  "plugin": ["./path/to/engram/opencode"]
}
```

Or copy the `opencode/` folder contents to your OpenCode config directory:

```bash
cp -r opencode/plugins ~/.config/opencode/
cp -r opencode/commands ~/.config/opencode/
cp -r opencode/skills ~/.config/opencode/
```

2. Add the dependency to `~/.config/opencode/package.json`:

```json
{
  "dependencies": {
    "@opencode-ai/plugin": "^1.1.25"
  }
}
```

### npm installation

```bash
npm install opencode-plugin-engram
```

Then add to your `opencode.json`:

```json
{
  "plugin": ["opencode-plugin-engram"]
}
```

## Features

### Commands

| Command | Description |
|---------|-------------|
| `/recall [query]` | Recall recent session context or search memories |
| `/remember "..."` | Save something explicitly for future sessions |
| `/prepare` | Load relevant context before starting new work |
| `/learn` | Review recent session for lessons worth persisting |

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

The Engram binary (`engram`) is built automatically on session start. It is stored at `~/.local/share/engram/bin/engram` with a symlink at `~/.local/bin/engram`.

The binary requires Go to be installed on your PATH for automatic building. If Go is not available, build manually:

```bash
go build -o ~/.local/share/engram/bin/engram ./cmd/engram/
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
