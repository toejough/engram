# Engram

> ⚠️ **Breaking change.** The pre-vault TOML memory-record storage layer
> (`~/.local/share/engram/memory/`) was removed. Engram now writes only
> to an agent-memory vault (Permanent/ and MOCs/). Migration from the
> old layout is not automated — see commit history for context.

## Overview

Engram gives Claude Code and OpenCode agents persistent memory via a zettelkasten-style vault. Two skills — `recall` and `learn` — read from and write to an agent-memory vault on demand. Both skills shell out to the `engram` binary for vault traversal and writes.

## Installing

Requires Go 1.25+ on `PATH`.

1. Install the binary:

   ```bash
   go install github.com/toejough/engram/cmd/engram@latest
   ```

   Make sure `$GOBIN` (or `$GOPATH/bin`, default `~/go/bin`) is on your `PATH`.

2. Copy the skills and commands into every detected harness's user directory:

   ```bash
   engram update            # install / refresh
   engram update --dry-run  # show what would change
   ```

   `engram update` writes Claude Code skills to `~/.claude/skills/` and OpenCode skills + commands to `~/.config/opencode/{skills,commands}/`. Run it again any time to upgrade — it also reinstalls the binary via `go install`.

## Skills

| Skill | What it does |
|-------|--------------|
| `recall` | Walks an agent-memory vault (Permanent/ + MOCs/) via explicit query plus situational baseline. Cascades through the wikilink graph and returns surfaced notes. |
| `learn` | Captures lessons from completed work as permanent vault notes. Each candidate passes three gates — Recurs + Activity-and-domain + Knowledge — before writing. |

See `skills/recall/SKILL.md` and `skills/learn/SKILL.md` for the full skill definitions.

## Vault location

Engram reads and writes a zettelkasten vault. Resolution order:

1. `--vault <path>` flag
2. `ENGRAM_VAULT_PATH` environment variable
3. `$XDG_DATA_HOME/engram/vault` (fallback: `~/.local/share/engram/vault`)

On first `engram learn` against a missing vault, the directory is
bootstrapped with `Permanent/`, `MOCs/`, `MEMORY.md`, a minimal
`.obsidian/` config so Obsidian recognizes it, a `.gitignore`, and a
`README.md`. `engram recall` does not bootstrap — it errors with
"vault not found" so the user notices.

Vault layout:

```
<vault>/
  Permanent/   atomic principle-stated notes; <luhmann-id>.<YYYY-MM-DD>.<slug>.md
  MOCs/        Maps of Content with framing prose
```

## Binary commands

```
engram recall                          Surface anchors, recent notes, or follow paths from a vault
engram transcript                      Read session transcripts since last /learn (Claude Code + OpenCode)
engram transcript --mark               Same, then advance per-harness progress markers
engram transcript --from <date>        Override marker; scan from explicit date
engram transcript --max-bytes <n>      Set byte budget (default 200000)
engram learn feedback --slug ... --source ... --situation ... --behavior ... --impact ... --action ...
engram learn fact     --slug ... --source ... --situation ... --subject ... --predicate ... --object ...
engram learn moc      --slug ... --source ... --topic ...
engram update                          Refresh binary and harness skills/commands ([--dry-run])
```

## Transcript progress tracking

`engram transcript` tracks a separate progress marker per harness (`last-learn-at-claude`, `last-learn-at-opencode`) under `${XDG_STATE_HOME:-$HOME/.local/state}/engram/projects/<slug>/`. Each marker is advanced independently by `--mark` so that sessions from one harness don't skip unprocessed sessions from the other. On first run (no marker), the scan defaults to the last 24 hours; use `--from` to include older history.

## Project structure

```
cmd/engram/          CLI entry point (thin wiring layer)
internal/            Business logic (DI boundaries)
  cli/               CLI command wiring (targ targets)
  context/           Transcript processing
  debuglog/          Structured debug logging
  learnmarker/       Per-harness progress marker (read/write/FS interface)
  luhmann/           Luhmann-ID allocation under file lock
  tokenresolver/     API token resolution
  transcript/        Session transcript reading (Claude Code JSONL + OpenCode SQLite)
  update/            Self-refresh subcommand
  vaultgraph/        Vault traversal (MOCs/Permanent, anchors, follow)
skills/              Source for the recall and learn skills
commands/            Source for OpenCode slash commands
```

## Development

- `targ build` — build the `engram` binary
- `targ test` — run unit + integration tests
- `targ check-full` — lint + coverage (use this to see ALL errors at once)
- Never run `go test` / `go build` / `go vet` directly — use `targ`

## Design principles

- **DI everywhere** — No function in `internal/` calls `os.*`, `http.*`, or any I/O directly. All I/O through injected interfaces, wired at CLI edges.
- **Pure Go, no CGO** — external API for LLM operations only.
- **Skills for behavior, slim Go binary for computation.**
