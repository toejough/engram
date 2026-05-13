# Engram — OpenCode Plugin

Memory for LLM agents. Surfaces relevant notes from an agent-memory vault, and captures lessons from completed work as permanent vault notes.

## Installation

Engram is distributed as source — clone the repo and point OpenCode at the cloned directory. Requires Go 1.25+ on your `PATH`.

1. Clone the repo:

   ```bash
   git clone https://github.com/toejough/engram ~/src/engram
   ```

2. Build the `engram` binary:

   ```bash
   cd ~/src/engram && go build -o ~/.local/bin/engram ./cmd/engram/
   ```

   Make sure `~/.local/bin` is on your `PATH`.

3. Add the plugin path to `~/.config/opencode/opencode.json`:

   ```json
   {
     "plugin": ["~/src/engram/opencode"]
   }
   ```

To upgrade, `git pull` in the cloned directory and re-run the `go build` command.

## What's included

### Commands

| Command | Description |
|---------|-------------|
| `/recall [query]` | Invoke the recall skill against the agent-memory vault |
| `/learn` | Invoke the learn skill to capture lessons from completed work |

### Skills

- `recall` — Retrieves relevant notes from the agent-memory vault using explicit-query and situational-baseline cascades.
- `learn` — Captures lessons as permanent vault notes when they pass the Recurs / Activity-and-domain / Knowledge gates.

Both skills shell out to the `engram` binary for vault traversal.

## Configuration

Vault path is set via `--vault <path>` on `engram recall` / `engram learn ...`, or via the `ENGRAM_VAULT_PATH` environment variable.
