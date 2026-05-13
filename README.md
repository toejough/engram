# Engram

> ⚠️ **Breaking change.** The pre-vault TOML memory-record storage layer
> (`~/.local/share/engram/memory/`) was removed. Engram now writes only
> to an agent-memory vault (Permanent/ and MOCs/). Migration from the
> old layout is not automated — see commit history for context.

## Overview

Engram is a plugin for **Claude Code** and **OpenCode** that gives agents persistent memory via a zettelkasten-style vault. Two skills — `recall` and `learn` — read from and write to an agent-memory vault on demand.

## Installing

Both hosts need the `engram` binary and the plugin source. Requires Go 1.25+ on `PATH`.

1. Install the binary:

   ```bash
   go install github.com/toejough/engram/cmd/engram@latest
   ```

   Make sure `$GOBIN` (or `$GOPATH/bin`, default `~/go/bin`) is on your `PATH`.

2. Clone the repo somewhere stable — both hosts read skills and commands from this location:

   ```bash
   git clone https://github.com/toejough/engram ~/src/engram
   ```

### Claude Code

Add the plugin via the marketplace (`/plugin`) pointed at your local clone, or configure the marketplace path in your Claude Code settings.

### OpenCode

Add the plugin path to `~/.config/opencode/opencode.json`:

```json
{
  "plugin": ["~/src/engram/opencode"]
}
```

### Upgrading

Run `engram update` to refresh both the binary and the harness skill/command
files in one step. The command auto-detects local clone vs. remote module:

```bash
engram update            # install + copy
engram update --dry-run  # show what would change
```

If you're inside a local clone, `engram update` re-runs `go install ./cmd/engram/`
and copies the current `skills/` and `opencode/commands/` files into every
detected harness (Claude Code at `~/.claude/`, OpenCode at `~/.config/opencode/`).
Otherwise it pulls the latest published module via `go install …@latest` and
copies from the module cache.

## Skills

| Skill | What it does |
|-------|--------------|
| `recall` | Walks an agent-memory vault (Permanent/ + MOCs/) via explicit query plus situational baseline. Cascades through the wikilink graph and returns surfaced notes. |
| `learn` | Captures lessons from completed work as permanent vault notes. Each candidate passes three gates — Recurs + Activity-and-domain + Knowledge — before writing. |

See `skills/recall/SKILL.md` and `skills/learn/SKILL.md` for the full skill definitions.

## Vault location

Engram reads and writes a zettelkasten vault. Pass `--vault <path>` to every `engram` invocation, or set `ENGRAM_VAULT_PATH`. Vault layout:

```
<vault>/
  Permanent/   atomic principle-stated notes; <luhmann-id>.<YYYY-MM-DD>.<slug>.md
  MOCs/        Maps of Content with framing prose
```

## Binary commands

```
engram recall                          Surface anchors, recent notes, or follow basenames from a vault
engram transcript                      Read Claude Code session transcripts in a date range
engram learn feedback --slug ... --vault ... --source ... --situation ... --behavior ... --impact ... --action ...
engram learn fact     --slug ... --vault ... --source ... --situation ... --subject ... --predicate ... --object ...
engram learn moc      --slug ... --vault ... --source ... --topic ...
engram update                          Refresh binary and harness skills/commands ([--dry-run])
```

## Project structure

```
cmd/engram/          CLI entry point (thin wiring layer)
internal/            Business logic (DI boundaries)
  cli/               CLI command wiring (targ targets)
  context/           Transcript processing
  debuglog/          Structured debug logging
  luhmann/           Luhmann-ID allocation under file lock
  tokenresolver/     API token resolution
  transcript/        Session transcript reading
  update/            Self-refresh subcommand
  vaultgraph/        Vault traversal (MOCs/Permanent, anchors, follow)
skills/              Plugin skills (recall, learn)
.claude-plugin/      Plugin manifest
opencode/            OpenCode plugin (commands, skills)
```

## Development

- `targ build` — build the `engram` binary
- `targ test` — run unit + integration tests
- `targ check-full` — lint + coverage (use this to see ALL errors at once)
- Never run `go test` / `go build` / `go vet` directly — use `targ`

## Design principles

- **DI everywhere** — No function in `internal/` calls `os.*`, `http.*`, or any I/O directly. All I/O through injected interfaces, wired at CLI edges.
- **Pure Go, no CGO** — external API for LLM operations only.
- **Plugin form factor** — skills for behavior, slim Go binary for computation.
