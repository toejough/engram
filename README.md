# Engram

> **Alpha** — Under active development. APIs and data formats may change. Feedback and contributions welcome via [issues](https://github.com/toejough/engram/issues).

Self-correcting memory for LLM agents. A [Claude Code plugin](https://docs.anthropic.com/en/docs/claude-code/plugins) that learns from sessions, surfaces relevant memories, measures whether they're actually followed, and fixes the ones that aren't.

## The problem

Claude Code has several instruction sources — CLAUDE.md, rules, skills — but no way to know if they're working. An instruction that's always loaded but never followed wastes context budget. A great instruction that only matches narrow keywords goes unseen. Without measurement, instruction sets decay: duplicates accumulate, stale guidance persists, and useful patterns stay buried.

## How engram solves it

Engram hooks into every phase of a Claude Code session to create a closed feedback loop:

```
Extract --> Deduplicate --> Write --> Surface --> Evaluate --> Maintain
  ^                                                            |
  +------------------------------------------------------------+
```

1. **Extract** — Learns from session transcripts and real-time corrections. User corrections ("remember that...", "don't do X") are captured immediately; broader patterns are extracted at session end. See [Architecture § Extract](docs/design/architecture.md#core-pipeline), [Session Lifecycle § Stop](docs/design/session-lifecycle.md#4-stop-stopsh-120s-timeout-async), [Memory Lifecycle § Creation](docs/design/memory-lifecycle.md#1-creation), and [`internal/extract/`](internal/extract/).

2. **Deduplicate** — Keyword overlap (>50%) against the existing corpus prevents redundant memories. Near-duplicates are merged on write; post-creation consolidation uses TF-IDF cosine similarity for cluster detection.

3. **Surface** — At every prompt, retrieves relevant memories via BM25 keyword scoring and injects them as context. A per-hook token budget caps injection to avoid overwhelming the model.

4. **Evaluate** — Tracks whether surfaced memories were followed, contradicted, ignored, or irrelevant. Outcome counters are stored directly in each memory's TOML file.

5. **Maintain** — Diagnoses each memory by effectiveness quadrant and proposes targeted fixes: rewrite stale content, broaden keywords for hidden gems, escalate enforcement for persistent violations, remove noise.

## Effectiveness quadrants

|  | High effectiveness | Low effectiveness |
|--|-------------------|------------------|
| **Often surfaced** | **Working** — Keep as-is | **Leech** — Rewrite or escalate |
| **Rarely surfaced** | **Hidden Gem** — Broaden keywords | **Noise** — Remove |

Effectiveness = followed / (followed + contradicted + ignored + irrelevant) * 100. Memories with fewer than 5 evaluations are classified as "insufficient data." See [Formulas](docs/design/formulas.md) for the full scoring model.

## Session lifecycle

Engram hooks into four phases of a Claude Code session: start (build + triage), prompt (surface + correct), stop-blocking (surface agent output), and stop-async (learn from transcript). See [Session Lifecycle](docs/design/session-lifecycle.md) for the full walkthrough and sequence diagram.

Engram registers three hook events:

| Event | Hook | What happens |
|-------|------|-------------|
| `SessionStart` | `session-start.sh` | Auto-build binary if stale. Run maintenance/triage in background. Remind user that `/recall` is available. |
| `UserPromptSubmit` | `user-prompt-submit.sh` | Surface prompt-relevant memories via BM25. Detect inline corrections. |
| `Stop` | `stop-surface.sh` | Surface memories relevant to agent output (blocking). |
| `Stop` | `stop.sh` (async) | Flush pipeline — extract learnings from transcript, record outcomes. |

## Adaptation

Engram observes its own performance and proposes system-level changes via the `adapt` pipeline. When keywords cause too many irrelevant matches, it proposes de-prioritizing them. When memory clusters overlap, it proposes consolidation. Proposals are reviewed and approved via the `/adapt` skill.

Configuration thresholds (cluster size, confidence minimums, feedback windows) are stored in `policy.toml` and can be tuned per-project.

## Memory TOML structure

Each memory is a TOML file with structured fields:

```toml
title = "Use targ for builds"
content = "Always use targ build system instead of raw go commands"
concepts = ["build-system", "tooling"]
keywords = ["targ", "build", "go test", "go vet"]
principle = "Use targ test, targ check, targ build for all operations"
anti_pattern = "Running go test or go vet directly"
confidence = "A"
```

Confidence tiers: **A** (explicit instruction — "always/never/remember"), **B** (teachable correction), **C** (contextual fact). See [Memory Lifecycle](docs/design/memory-lifecycle.md) for how memories are created, surfaced, evaluated, and maintained over time.

## Data files

All data lives in `~/.claude/engram/data/`:

| Path | Purpose |
|------|---------|
| `memories/*.toml` | Structured memory files with embedded metrics |
| `creation-log.jsonl` | Append-only log of memory creation events |
| `surfacing-log.jsonl` | Log of which memories were surfaced and when |
| `learn-offset.json` | Offset tracking for incremental transcript learning |
| `policy.toml` | Adaptation config and policy directives |

## Installation

Requires Go 1.25+.

```bash
git clone https://github.com/toejough/engram.git
```

Enable in Claude Code via the `/plugin` command. The binary auto-builds on first hook invocation and rebuilds when source files change.

## Skills

| Skill | Purpose |
|-------|---------|
| `/recall` | Load context from previous sessions. No args = raw transcript summary (what was decided, what got done, what's outstanding). With query = Haiku-filtered search across sessions. |
| `/adapt` | Review and act on adaptation proposals — approve, reject, or retire policies that tune extraction and surfacing behavior. |
| `/memory-triage` | Interactive review of maintenance signals — noise removal, keyword broadening, consolidation, graduation to higher tiers. |

See [Skills Guide](docs/howto/skills.md) for detailed workflows and example interactions.

## Project structure

```
cmd/engram/          CLI entry point (thin wiring layer)
internal/            Business logic (36 packages, all DI boundaries)
hooks/               Shell scripts for Claude Code hook integration
skills/              Plugin skills (recall, adapt, memory-triage)
.claude-plugin/      Plugin manifest
docs/                Documentation (design, use-cases, standards, howto)
archive/             Historical planning artifacts
```

## Design principles

- **DI everywhere** — No function in `internal/` calls `os.*`, `http.*`, or any I/O directly. All I/O through injected interfaces, wired at CLI edges.
- **Pure Go, no CGO** — TF-IDF and BM25 for retrieval. External API for LLM classification only.
- **Fire and forget** — Hook errors are logged, never propagated. Hooks always exit 0.
- **Measure impact, not frequency** — A memory surfaced 1000 times but never followed is a leech, not a success.

## What about built-in memory?

Anthropic has introduced built-in auto-memory and dreaming features for Claude Code. Memory is one of the primary problems that needs solving, and lots of smart people are working on it.

I tried auto-memory when it was introduced and found it unhelpful for the same reasons I'd had trouble with "please write this down somewhere" — non-deterministic recording, unreliable surfacing. I ended up turning it off because it was conflicting with engram. I haven't tried the new dreaming features yet — maybe they've materially improved things.

If the built-in memory management becomes good enough that most of engram is unnecessary, I'll be glad — less to maintain. Till then, engram exists because I wanted memory that **measures outcomes**, **self-corrects**, and **keeps the user in control** of what changes.

## What it doesn't do

- **No vector embeddings** — Uses TF-IDF/BM25, not dense vectors. Keeps the dependency footprint minimal.
- **No cross-project sharing** — Memories are per-data-directory. No sync, no cloud storage.
- **No automatic execution** — Maintenance proposals require user approval before applying.
- **No GUI** — CLI-only, designed for terminal workflows.

## Documentation

### Design
- [Architecture](docs/design/architecture.md) — Pipeline design, package map, key decisions
- [System Context](docs/design/context.md) — C4 context diagram
- [Data Model](docs/design/data-model.md) — Memory TOML schema, supporting files
- [Session Lifecycle](docs/design/session-lifecycle.md) — Hook events and sequence diagram
- [Memory Lifecycle](docs/design/memory-lifecycle.md) — Creation to deletion with state diagram
- [Formulas](docs/design/formulas.md) — Effectiveness, ranking, budget, and penalty formulas

### How-to
- [Installation & Usage](docs/howto/installation.md) — Setup and configuration
- [Skills Guide](docs/howto/skills.md) — /recall, /adapt, /memory-triage with examples

### Reference
- [Use Cases](docs/use-cases/README.md) — UC-1 through UC-34
- [Coding Standards](docs/standards/coding.md) — DI rules, TDD, targ usage
