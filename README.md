# Engram

Self-correcting memory for LLM agents. A [Claude Code plugin](https://docs.anthropic.com/en/docs/claude-code/plugins) that learns from sessions, surfaces relevant memories, and measures impact — memories that don't improve outcomes get diagnosed and fixed.

## How it works

Engram runs as hook-driven middleware in Claude Code sessions:

1. **Learn** — Extracts structured memories from user corrections, instructions, and contextual facts (TOML files with tier-based metadata)
2. **Surface** — Injects relevant memories at prompt submission and tool use via BM25 keyword matching, within a configurable context budget
3. **Evaluate** — Measures whether surfaced memories were followed, contradicted, or ignored
4. **Maintain** — Diagnoses underperforming memories and proposes fixes: rewrites, keyword broadening, escalation, or removal

## Installation

Requires Go 1.25+.

```bash
git clone https://github.com/toejough/engram.git
cd engram
go build -o engram ./cmd/engram/
```

Install as a Claude Code plugin by adding the repo path to your Claude Code plugin configuration.

## Architecture

Pure Go, no CGO. DI everywhere — all I/O through injected interfaces, wired at CLI edges. Hook scripts call the `engram` binary with subcommands.

### Hook integration

| Hook | Script | Purpose |
|------|--------|---------|
| SessionStart | `session-start.sh` | Initialize session context |
| UserPromptSubmit | `user-prompt-submit.sh` | Surface memories, detect learning signals |
| UserPromptSubmit (async) | `user-prompt-submit-async.sh` | Incremental learning extraction |
| PreToolUse | `pre-tool-use.sh` | Surface tool-specific memories |
| PostToolUse | `post-tool-use.sh` | Proactive reminders after file edits |
| PreCompact | `pre-compact.sh` | Incremental learning before context compaction |
| Stop | `stop.sh` | Session audit, outcome evaluation, learning |

### CLI subcommands

| Command | Description |
|---------|-------------|
| `surface` | Retrieve and inject relevant memories |
| `learn` | Extract memories from transcript |
| `evaluate` | Measure memory effectiveness |
| `correct` | Apply user corrections to memories |
| `maintain` | Generate/apply maintenance proposals |
| `review` | Classify instructions by effectiveness quadrant |
| `promote` | Promote memories to skills or CLAUDE.md |
| `demote` | Demote CLAUDE.md entries to skills |
| `registry` | Manage the unified instruction registry |
| `audit` | Session compliance audit |
| `remind` | Proactive post-tool-use reminders |
| `instruct` | Instruction quality analysis |
| `automate` | Extract mechanical rules for automation |
| `context-update` | Session continuity context |

### Effectiveness quadrants

The registry classifies instructions into four quadrants based on surfacing frequency and follow-through rate:

- **Working** — High surfacing, high effectiveness. Keep as-is.
- **Leech** — High surfacing, low effectiveness. Rewrite or escalate.
- **HiddenGem** — Low surfacing, high effectiveness. Broaden keywords.
- **Noise** — Low surfacing, low effectiveness. Remove.

### Instruction tiers

Memories flow through a three-tier promotion ladder:

1. **Memory** (TOML files) — Surfaced by keyword matching on every prompt
2. **Skill** (plugin skills) — Loaded by context similarity, lower per-prompt cost
3. **CLAUDE.md** (always loaded) — Highest trust, highest cost, reserved for universal guidance

Promotion and demotion are driven by measured effectiveness, confirmed by the user.

## Project structure

```
cmd/engram/          CLI entry point
internal/            Business logic (30 packages, DI boundaries)
hooks/               Shell scripts for Claude Code hook integration
.claude-plugin/      Plugin manifest
docs/specs/          Specification artifacts (use cases, requirements, architecture, tests)
```

## Specification

18 use cases implemented across 5 specification layers (UC → REQ/DES → ARCH → TEST → IMPL). 402 tests. Spec artifacts in `docs/specs/`.

| UC | Name | Status |
|----|------|--------|
| UC-1 | Session Learning | Complete |
| UC-2 | Hook-Time Surfacing & Enforcement | Complete |
| UC-3 | Remember & Correct | Complete |
| UC-4 | Skill Generation | Complete |
| UC-5 | CLAUDE.md Management | Complete |
| UC-6 | Memory Effectiveness Review | Complete |
| UC-14 | Structured Session Continuity | Complete |
| UC-15 | Automatic Outcome Signal | Complete |
| UC-16 | Unified Memory Maintenance | Complete |
| UC-17 | Context Budget Management | Complete |
| UC-18 | PostToolUse Proactive Reminders | Complete |
| UC-19 | Stop Session Audit | Complete |
| UC-20 | Instruction Quality & Gap Analysis | Complete |
| UC-21 | Enforcement Escalation Ladder | Complete |
| UC-22 | Mechanical Instruction Extraction | Complete |
| UC-23 | Unified Instruction Registry | Complete |
| UC-24 | Proposal Application | Complete |
| UC-25 | Evaluate Strip Preprocessing | Complete |

## License

See [LICENSE](LICENSE) for details.
