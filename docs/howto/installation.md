# Installation

Engram is a Claude Code plugin. It lives in a git repository and integrates via Claude Code's plugin system.

## Enable the Plugin

From within Claude Code, run:

```
/plugin
```

Point it at the engram repository directory. Claude Code will register the hooks defined in `hooks/hooks.json`.

## Binary

The Go binary is built automatically by hook scripts on first use or when source files change. It installs to:

```
~/.claude/engram/bin/engram
```

To build manually:

```bash
cd /path/to/engram
targ build
```

The `SessionStart` hook also creates a symlink at `~/.local/bin/engram` for PATH access.

## Data Directory

All runtime data is stored in:

```
~/.claude/engram/data/
```

Contents:
- `memories/` -- individual TOML memory files
- `creation-log.jsonl` -- memory creation audit log
- `policy.toml` -- adaptive policy configuration and directives

## Configuration

System behavior is configured via `policy.toml` in the data directory. This file contains:

- Adaptive policies (proposed/approved/active) that tune extraction, surfacing, and maintenance behavior.
- Approval streaks per dimension.
- Adaptation thresholds (cluster size, feedback events, measurement window, etc.).

Policies are managed through the `/adapt` skill or the `engram adapt` CLI subcommand.

## Skills

Three interactive skills are available:

| Skill | Purpose |
|-------|---------|
| `/recall` | Load context from previous sessions or search session history |
| `/adapt` | Review and manage adaptive policy proposals |
| `/memory-triage` | Interactive walkthrough of maintenance proposals |
