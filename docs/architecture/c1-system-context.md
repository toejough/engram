# L1 — System context

The system in scope is **Engram**, persistent memory for LLM coding agents. This
diagram shows the people and external systems engram interacts with at runtime.
Containers, components, technologies, and protocols are hidden — those live at L2
and below (not yet authored). The [Key flows](#key-flows) section below pairs the
static view with sequence diagrams for the four user-initiated runtime flows.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff

    user([S1 · Engram operator])
    engram[S2 · Engram]
    harness("S3 · LLM coding harness<br/>(Claude Code, OpenCode)")
    vault(S4 · Agent-memory vault)
    sessions(S5 · Harness session stores)
    gotool(S6 · Go toolchain)

    user -->|"R1: directs work via prompts"| harness
    harness -->|"R2: invokes /recall, /learn, /please; runs engram CLI"| engram
    engram -->|"R3: reads & writes notes and MOCs"| vault
    engram -->|"R4: reads session transcripts via per-harness markers"| sessions
    engram -->|"R5: invokes go install / go list for self-update"| gotool
    engram -->|"R6: writes refreshed skill and command files during engram update"| harness

    class user person
    class harness,vault,sessions,gotool external
    class engram container

    click user href "#s1-engram-operator"
    click engram href "#s2-engram"
    click harness href "#s3-llm-coding-harness"
    click vault href "#s4-agent-memory-vault"
    click sessions href "#s5-harness-session-stores"
    click gotool href "#s6-go-toolchain"
```

## Element catalog

| ID | Name | Type | Responsibility | Source |
|---|---|---|---|---|
| <a id="s1-engram-operator"></a>S1 | Engram operator | Person | Directs work through the LLM coding harness; configures engram via environment variables (`ENGRAM_VAULT_PATH`, `ENGRAM_STATE_DIR`, `ENGRAM_TRANSCRIPT_DIR`, etc.) | Human |
| <a id="s2-engram"></a>S2 | Engram | System in scope | Persistent memory for LLM coding agents: reads & writes a Luhmann zettelkasten vault, reads per-harness session transcripts via markers, and self-updates | This repo (`cmd/engram/`, `internal/`, `skills/`) |
| <a id="s3-llm-coding-harness"></a>S3 | LLM coding harness | External system | Hosts engram's slash commands and subprocess-invokes the engram CLI. Engram skills are loaded by the harness's skill mechanism. | Claude Code (`~/.claude/`), OpenCode (`~/.config/opencode/`) |
| <a id="s4-agent-memory-vault"></a>S4 | Agent-memory vault | External system | Luhmann zettelkasten on the local filesystem — `Permanent/` notes and `MOCs/` | `$ENGRAM_VAULT_PATH` or `$XDG_DATA_HOME/engram/vault` (typically `~/.local/share/engram/vault`) |
| <a id="s5-harness-session-stores"></a>S5 | Harness session stores | External system | The LLM harness's per-session transcript storage; engram reads them at the filesystem level, not via a harness API | Claude Code: `~/.claude/projects/<slug>/*.jsonl` · OpenCode: `~/.local/share/opencode/opencode.db` (SQLite) |
| <a id="s6-go-toolchain"></a>S6 | Go toolchain | External system | Resolves module versions and installs the engram binary during `engram update` | `go` binary on `$PATH` |

## Relationships

| ID | From | To | Description |
|---|---|---|---|
| <a id="r1"></a>R1 | S1 Engram operator | S3 LLM coding harness | Directs work via prompts in the harness; configures engram via environment variables |
| <a id="r2"></a>R2 | S3 LLM coding harness | S2 Engram | Invokes `/recall`, `/learn`, `/please` slash commands; subprocess-executes the engram CLI for each invocation |
| <a id="r3"></a>R3 | S2 Engram | S4 Agent-memory vault | Reads & writes notes and MOCs under a `flock`-held vault lock; rendered as a single unidirectional arrow per the C4 read+write CRUD convention |
| <a id="r4"></a>R4 | S2 Engram | S5 Harness session stores | Reads JSONL transcripts (Claude Code) and SQLite rows (OpenCode) starting from a per-harness marker held in `$XDG_STATE_HOME/engram` |
| <a id="r5"></a>R5 | S2 Engram | S6 Go toolchain | During `engram update`, invokes `go list -m -json` and `go install` to self-update |
| <a id="r6"></a>R6 | S2 Engram | S3 LLM coding harness | During `engram update`, copies refreshed `skills/` and `commands/` files into each detected harness's install root (`~/.claude/`, `~/.config/opencode/`) |

## Key flows

Four user-initiated flows span the L1 edges. Each diagram below uses the
shorthand participant aliases `Op` (S1), `H` (S3), `E` (S2), `V` (S4), `Tr`
(S5), `Go` (S6) and only declares the participants that flow touches. Source
file:line references point at the entry points on `main`.

### Flow: recall

Operator asks a question that needs prior memory. The harness loads the `recall`
skill, prints its Step 0 judgement (Ask, Situation, Plan), then drives a cascade
of subprocess calls into `engram recall` until the budget is spent or the
wikilink frontier empties. Source: `internal/cli/cli.go:308` (`runRecall`) and
its three branches `runRecallAnchors`, `runRecallRecent`, `runRecallFollow`.

```mermaid
sequenceDiagram
    autonumber
    actor Op as S1 Operator
    participant H as S3 Harness
    participant E as S2 Engram CLI
    participant V as S4 Vault

    Op->>H: prompt that may need memory
    Note over H: print Step 0 (Ask, Situation, Plan), phrase 5-15 queries

    H->>E: engram recall (anchors)
    E->>V: scan MOCs and Permanent for starting points
    V-->>E: anchor paths
    E-->>H: vault-relative paths on stdout

    H->>E: engram recall --recent --limit 20
    E->>V: rank notes by Mtime
    V-->>E: recent paths
    E-->>H: vault-relative paths on stdout

    Note over H: union frontier, score against queries and Step 1 phrases

    loop until 100 surfaced or frontier empty
        Note over H: read relevant notes inline or via subagent
        H->>E: engram recall --follow A,B --already-read X,Y
        E->>V: expand wikilink graph from follow set, minus already-read
        V-->>E: expanded paths
        E-->>H: vault-relative paths on stdout
        Note over H: print round status, score new frontier
    end

    Note over H: Step 4 synthesis against the Step 0 plan
    H-->>Op: reply naming which plan items were confirmed, adjusted, or contradicted
```

### Flow: learn

Operator runs `/learn` (or the harness self-fires after substantive work). The
harness first invokes `engram transcript --mark` to read session JSONL or
SQLite from S5 and advance the per-harness marker forward, then writes any
captured lessons into the vault via `engram learn {feedback|fact}`. Each
write acquires a `flock` on the vault root before computing the Luhmann ID and
emitting the new file. Source: `internal/cli/transcript.go:117`
(`advanceAndReportMarker`) and `internal/cli/learn.go:338` (`runLearn`).

```mermaid
sequenceDiagram
    autonumber
    actor Op as S1 Operator
    participant H as S3 Harness
    participant E as S2 Engram CLI
    participant Tr as S5 Session stores
    participant V as S4 Vault

    Op->>H: invoke /learn (or self-fire after substantive work)

    H->>E: engram transcript --mark
    E->>Tr: read JSONL or SQLite from per-harness marker forward
    Tr-->>E: session entries up to byte cap
    Note over E: write marker forward (XDG_STATE_HOME)
    E-->>H: status line, scanned range, advanced marker

    alt no marker yet (first run)
        E-->>H: exit non-zero, earliest session date
        H->>Op: ask scan start date via AskUserQuestion
        Op-->>H: chosen start date
        H->>E: engram transcript --mark --from CHOSEN
        E-->>H: status line
    end

    Note over H: read transcript output plus in-context turns, identify candidates, apply recall-mirror test

    loop per candidate (one parallel tool-use block)
        H->>E: engram learn feedback|fact --slug ... --source ... --situation ...
        alt vault dir missing
            E->>V: bootstrap Permanent, .obsidian, README, .gitignore
        end
        E->>V: acquire flock, compute Luhmann ID, write note
        V-->>E: written path
        E-->>H: emit written path on stdout
    end

    H-->>Op: report scanned status line plus written permanent paths
```

### Flow: please

`/please` is a skill-only orchestration of the engram repo's other skills — it
has no dedicated subcommand. The diagram below shows the seven-step bracket;
each step that crosses an L1 edge appears as a call into Engram (with the
implementation of `recall`, `learn`, etc. shown in their own diagrams above).
The diagram is intentionally workflow-shaped, not call-surface-shaped — at L1
all engram subprocess calls collapse onto the same R2 edge.

```mermaid
sequenceDiagram
    autonumber
    actor Op as S1 Operator
    participant H as S3 Harness
    participant E as S2 Engram CLI

    Op->>H: /please ASK
    Note over H: load please skill, push 7 tasks to the list

    rect rgb(245,245,255)
        Note over H: Step 1 — opening /learn
        H->>E: engram transcript --mark (and zero-or-more engram learn writes)
        E-->>H: marker status plus any written paths
    end

    rect rgb(245,245,255)
        Note over H: Step 2 — orient via /recall
        loop cascade rounds
            H->>E: engram recall (anchors, recent, follow)
            E-->>H: vault-relative paths
        end
        Note over H: clarify with the operator via AskUserQuestion if intent is unclear
    end

    rect rgb(245,245,255)
        Note over H: Step 3 — plan, then git commit the plan
    end

    rect rgb(245,245,255)
        Note over H: Step 4 — execute under TDD discipline, repeated FS and build calls
    end

    rect rgb(245,245,255)
        Note over H: Step 5 — update docs touched by the change
    end

    rect rgb(245,245,255)
        Note over H: Step 6 — commit via /commit and delete planning artifacts
    end

    rect rgb(245,245,255)
        Note over H: Step 7 — closing /learn
        H->>E: engram transcript --mark
        E-->>H: marker status
        loop per lesson
            H->>E: engram learn feedback|fact ...
            E-->>H: written path
        end
    end

    H-->>Op: terminal report (commits made, paths written, follow-ups offered)
```

### Flow: update

`engram update` refreshes both the engram binary (via Go) and the harness's
installed skills and commands. It walks up from `cwd` to detect a local clone:
on hit it runs `go install ./cmd/engram/` from the clone; on miss it runs
`go install ...@latest` followed by `go list -m -json` to resolve the module
root for the skill source. The CLI then copies each skill file and command
file into every detected harness install root. Source:
`internal/cli/update.go:199` (`runUpdate`) and `internal/update/update.go:152`
(`Updater.Run`).

```mermaid
sequenceDiagram
    autonumber
    actor Op as S1 Operator
    participant H as S3 Harness
    participant E as S2 Engram CLI
    participant Go as S6 Go toolchain

    Op->>H: invoke engram update (or --dry-run)
    H->>E: engram update

    Note over E: walk up from cwd searching for the local module

    alt local clone found
        E->>Go: go install ./cmd/engram/
        Go-->>E: installed engram into GOBIN
        Note over E: read skills and commands from the local module dir
    else no local clone
        E->>Go: go install github.com/toejough/engram/cmd/engram@latest
        Go-->>E: installed engram into GOBIN
        E->>Go: go list -m -json github.com/toejough/engram
        Go-->>E: module Dir and Version JSON
        Note over E: read skills and commands from the resolved module dir
    end

    Note over E: plan copy ops for each detected harness (Claude Code, OpenCode)

    loop per harness, per skill or command file
        Note over E: write into the harness install root (~/.claude/skills, ~/.claude/commands, OpenCode equivalents)
    end

    E-->>H: per-harness report (skill paths, command paths, errors)
    H-->>Op: rendered report
```

The copy loop and the Go-toolchain calls are modeled in the static L1 as
relationships [R6](#r6) and [R5](#r5) respectively.

## Out of scope at L1

L1 hides containers, components, technologies, protocols, and internal structure.
Engram's internal containers (CLI binary, skills, transcript reader, vault writer,
update subsystem, debug logger) are deferred to L2.

The Voyage embedding API discussed in
[`docs/superpowers/specs/2026-05-14-tiered-memory-design.md`](../superpowers/specs/2026-05-14-tiered-memory-design.md)
is **not** an external at L1: that design is not built. When it lands, it joins as
an additional external system.

## Related

- L2 container diagram: not yet authored.
