# OpenCode Plugin Plan

> Created: 2026-04-30
> Status: Approved, ready for implementation

## Overview

Port the Engram Claude Code plugin to also work as an OpenCode plugin. OpenCode uses a different extension model (TypeScript/JS plugins + markdown commands + skills) rather than Claude Code's shell hooks + `.claude-plugin` manifest.

## Validated Assumptions

### Confirmed

| # | Assumption | Evidence |
|---|---|---|
| 1 | OpenCode reads `.claude/skills/*/SKILL.md` | Skills docs explicitly list Claude-compatible paths |
| 2 | Skill names must match `^[a-z0-9]+(-[a-z0-9]+)*$` | Skills docs — regex explicitly stated |
| 3 | Commands as `.md` files → `/command-name` | Commands docs — filename becomes command name |
| 4 | Commands support `$ARGUMENTS`, `!command`, `@file` | Commands docs — all three documented |
| 5 | Custom tools via `tool()` from `@opencode-ai/plugin` | npm package exports `./tool` subpath |
| 6 | Only generic `event` hook exists (no specific event hooks) | `@opencode-ai/plugin` Hooks interface audit |
| 7 | No `command` hook for plugin-registered slash commands | Issue #10262 + source audit |
| 8 | `experimental.chat.system.transform` injects into system prompt | [opencode-rules plugin](https://github.com/frap129/opencode-rules), multiple examples |
| 9 | `message.updated` has empty `parts` — can't read user prompt text | [Issue #22831](https://github.com/anomalyco/opencode/issues/22831) |
| 10 | Binary reads zero Claude Code env vars | Grep of all Go files — only `ENGRAM_API_TOKEN`, `XDG_DATA_HOME`, `ENGRAM_DATA_DIR` |
| 11 | `bun install` at startup for plugin deps | Plugin docs — Dependencies section |
| 12 | npm package convention: `opencode-plugin-*`, `"type": "module"` | `opencode-plugin-opencoder` + other published plugins |

### Invalidated (plan adjusted accordingly)

| # | Original Assumption | Reality | Plan Adjustment |
|---|---|---|---|
| 16 | Skill descriptions trigger auto-invocation | OpenCode does NOT auto-invoke skills — "list-and-let-the-agent-choose" model | Use `experimental.chat.system.transform` for per-prompt reminders |
| 18 | Binary works unchanged from OpenCode | Binary hardcodes `.claude/` paths for transcripts + external sources | Add `--transcript-dir` flag, toggle external sources |
| 19 | Binary build path `~/.claude/engram/` | Wrong for OpenCode-only users | Use `~/.local/share/engram/bin/engram` |

## Architecture

### Claude Code vs OpenCode Mapping

| Claude Code | OpenCode | How |
|---|---|---|
| `SessionStart` hook | `event` → `session.created` | Async binary build triggered on session start |
| `PostToolUse` hook | `experimental.chat.system.transform` | Per-prompt reminder injected into system prompt |
| `UserPromptSubmit` hook | `experimental.chat.system.transform` | Same — merged into single system transform |
| Skill frontmatter auto-invocation | `experimental.chat.system.transform` | OpenCode doesn't auto-invoke skills |
| Slash commands (skill names) | `.opencode/commands/*.md` | Markdown files with frontmatter |
| `.claude-plugin/manifest.json` | `opencode/package.json` | npm package for distribution |
| Hook shell scripts | TypeScript plugin | `engram.ts` using Bun shell API |

### File Structure

```
opencode/
├── plugins/
│   └── engram.ts              # Main TypeScript plugin
├── commands/
│   ├── recall.md              # /recall [query]
│   ├── remember.md            # /remember "..."
│   ├── prepare.md             # /prepare
│   └── learn.md               # /learn
├── skills/                    # Copies for self-containment
│   ├── learn/SKILL.md
│   ├── recall/SKILL.md
│   ├── prepare/SKILL.md
│   └── remember/SKILL.md
├── package.json               # npm distribution
└── README.md                  # OpenCode install instructions
```

## Implementation Phases

### Phase 1: Binary Adaptations (Go)

**Goal:** Make the `engram` binary work in OpenCode contexts without breaking Claude Code.

#### 1a. Transcript directory flag

**File:** `internal/cli/cli.go`

- Add `--transcript-dir` flag to `engram recall`
- Default: `$HOME/.claude/projects/<slug>/` (backward compat)
- OpenCode users set to their transcript directory

**File:** `internal/cli/targets.go`

- Add `ENGRAM_TRANSCRIPT_DIR` env var support via targ flag tag

#### 1b. External sources toggle

**File:** `internal/externalsources/discover.go`

- Add `--no-external-sources` flag to `engram recall`
- When set, skip CLAUDE.md, rules, auto-memory, skills discovery
- Prevents silent empty results in OpenCode-only contexts

#### 1c. Neutralize build path

**File:** `hooks/session-start.sh`

- Change `ENGRAM_BIN` from `$HOME/.claude/engram/bin/engram` to `$HOME/.local/share/engram/bin/engram`
- Symlink remains at `$HOME/.local/bin/engram`

### Phase 2: OpenCode Plugin (TypeScript)

**File:** `opencode/plugins/engram.ts`

#### Hooks

**`event`** — filter on `event.type === "session.created"`:
- Spawn async binary build (same logic as `session-start.sh`)
- Check if `~/.local/share/engram/bin/engram` exists
- Check if Go sources are newer than cached binary
- Rebuild if needed using `$` Bun shell API
- Create symlink at `~/.local/bin/engram`

**`experimental.chat.system.transform`** — per-prompt reminder:
- Inject reminder block about `/prepare` and `/learn` boundaries
- Merge into primary system block (not `push`) to avoid vLLM/Qwen breakage (Issue #23660)
- Only inject on first turn per session (track via session ID set)

**`tool`** — 5 custom tools:

| Tool | Wraps | Args |
|---|---|---|
| `engram_recall` | `engram recall` | `query?` (string) |
| `engram_learn_feedback` | `engram learn feedback` | `situation`, `behavior`, `impact`, `action`, `source` |
| `engram_learn_fact` | `engram learn fact` | `situation`, `subject`, `predicate`, `object`, `source` |
| `engram_show` | `engram show` | `name` |
| `engram_list` | `engram list` | (none) |

All tools execute via `$` Bun shell API, capture stdout/stderr.

### Phase 3: OpenCode Commands (Markdown)

**Directory:** `opencode/commands/`

Each command is a markdown file with YAML frontmatter. Content becomes the prompt template.

#### `recall.md`

```markdown
---
description: Recall recent session context or search memories
---
Use the engram_recall tool to recall context.
$ARGUMENTS
```

#### `remember.md`

```markdown
---
description: Remember something explicitly for future sessions
---
Use the engram_learn_fact or engram_learn_feedback tool to save knowledge.
$ARGUMENTS
```

#### `prepare.md`

```markdown
---
description: Load relevant context before starting new work
---
Use the engram_recall tool to prepare for the pending task.
Break the task into 2-3 sub-topic queries and recall each.
```

#### `learn.md`

```markdown
---
description: Review recent session for lessons worth persisting
---
Review the recent session for learnable moments:
- User corrections
- Failed approaches
- Discovered facts
- Recurring patterns

Use engram_learn_feedback for behavioral corrections (SBIA).
Use engram_learn_fact for declarative knowledge (SPO).
```

### Phase 4: Skills (Copies)

**Directory:** `opencode/skills/`

Copy existing skills from repo root. Skill names already compatible (lowercase, no hyphens needed).

Skills to copy:
- `learn/SKILL.md`
- `recall/SKILL.md`
- `prepare/SKILL.md`
- `remember/SKILL.md`

Note: OpenCode already reads `.claude/skills/` natively, so these copies are for self-containment of the npm package.

### Phase 5: Package & Distribution

**File:** `opencode/package.json`

```json
{
  "name": "opencode-plugin-engram",
  "version": "0.1.0",
  "description": "Self-correcting memory for LLM agents",
  "type": "module",
  "main": "plugins/engram.ts",
  "files": ["plugins/", "commands/", "skills/"],
  "keywords": ["opencode", "opencode-plugin", "memory", "ai-agents"],
  "dependencies": {
    "@opencode-ai/plugin": "^1.1.25"
  }
}
```

**File:** `opencode/README.md`

Install instructions for both local and npm distribution.

### Phase 6: Tests

- Plugin: basic test that hooks return correct shape
- Commands: verify markdown frontmatter parses correctly
- Binary: test new `--transcript-dir` and `--no-external-sources` flags

## Risk Mitigations

| Risk | Mitigation |
|---|---|
| `experimental.chat.system.transform` breaks vLLM backends | Merge into primary system block, don't `push` new entry |
| Binary external sources return empty in OpenCode | `--no-external-sources` flag, graceful degradation |
| Skill discovery unreliable in OpenCode | Commands + tools provide direct invocation paths |
| Binary build fails (no Go on PATH) | Async build, errors silently logged, user can build manually |

## Success Criteria

1. `engram` binary builds and runs from OpenCode plugin
2. `/recall`, `/remember`, `/prepare`, `/learn` commands work in OpenCode TUI
3. Custom tools (`engram_recall`, etc.) are available to the agent
4. Per-prompt reminders about `/learn` and `/prepare` boundaries fire correctly
5. Binary works with both Claude Code paths (backward compat) and OpenCode paths
6. npm package installs correctly via `opencode.json` plugin array
