---
level: 3
name: engram-cli-binary
parent: "c2-engram-plugin.md"
children: []
last_reviewed_commit: 44cec351
---

# C3 — engram CLI binary (Component)

Refines L2's E9 engram CLI binary into nine internal components. The shell of the binary (cmd/engram/main.go) only wires cli.Targets into targ.Main; all command logic, I/O adapters, and external integrations live under internal/. Pure-logic packages (recall, memory, tomlwriter) take all I/O as DI interfaces; thin adapter shims live in internal/cli so concrete I/O is wired only at the edge of the binary.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff
    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000

    e3(E3 · Claude Code<br/>agent harness)
    e4(E4 · Claude Code memory surfaces)
    e5(E5 · Anthropic API<br/>Haiku)
    e6(E6 · Engram memory store<br/>~/.local/share/engram/memory/)

    subgraph e9 [E9 · engram CLI binary]
        e20[E20 · main.go<br/>process entry]
        e21[E21 · cli<br/>dispatch + I/O adapters]
        e22[E22 · recall<br/>orchestrator + phases]
        e23[E23 · context<br/>transcript reader]
        e24[E24 · memory<br/>feedback / fact records]
        e25[E25 · externalsources<br/>CLAUDE.md / rules / auto-memory / skills]
        e26[E26 · anthropic<br/>Messages API client]
        e27[E27 · tokenresolver<br/>env + macOS Keychain]
        e28[E28 · tomlwriter<br/>TOML serialization]
    end

    e3 -->|"R1: execs the binary as a subprocess (Bash tool)"| e20
    e20 -->|"R2: builds CLI targets and runs targ.Main"| e21
    e21 -->|"R3: delegates the recall pipeline (Orchestrator + phases) to the dedicated recall package — the one subcommand currently extracted from cli (see Drift Notes)"| e22
    e21 -->|"R4: reads / writes feedback + fact TOML through memory types and helpers"| e24
    e21 -->|"R5: discovers external source paths and shares the cache via discoverExternalSources"| e25
    e21 -->|"R6: builds the Anthropic caller used by recall.NewSummarizer"| e26
    e21 -->|"R7: resolves API token (env or Keychain) before any LLM call"| e27
    e22 -->|"R8: reads + strips session transcripts within budget"| e23
    e22 -->|"R9: lists memories during recall ranking"| e24
    e22 -->|"R10: ranks candidates and extracts snippets via Haiku (through DI Summarizer)"| e26
    e22 -->|"R11: reads CLAUDE.md / rules / auto-memory / skill frontmatter (cached)"| e25
    e21 -->|"R12: writes new TOML on learn / remember / update"| e28
    e21 -->|"R13: prints briefings, recall results, and list / show output to stdout"| e3
    e26 -->|"R14: HTTPS POST /v1/messages (Haiku)"| e5
    e25 -->|"R15: reads project + user CLAUDE.md, .claude/rules, auto-memory, skill frontmatter"| e4
    e24 -->|"R16: reads existing feedback + fact TOML during recall / list / show"| e6
    e28 -->|"R17: writes new feedback + fact TOML on learn / remember / update"| e6

    class e3,e4,e5,e6 external
    class e20,e21,e22,e23,e24,e25,e26,e27,e28 component
    class e9 container

    click e9 href "#e9-engram-cli-binary" "engram CLI binary"
    click e3 href "#e3-claude-code" "Claude Code"
    click e4 href "#e4-claude-code-memory-surfaces" "Claude Code memory surfaces"
    click e5 href "#e5-anthropic-api" "Anthropic API"
    click e6 href "#e6-engram-memory-store" "Engram memory store"
    click e20 href "#e20-main-go" "main.go"
    click e21 href "#e21-cli" "cli"
    click e22 href "#e22-recall" "recall"
    click e23 href "#e23-context" "context"
    click e24 href "#e24-memory" "memory"
    click e25 href "#e25-externalsources" "externalsources"
    click e26 href "#e26-anthropic" "anthropic"
    click e27 href "#e27-tokenresolver" "tokenresolver"
    click e28 href "#e28-tomlwriter" "tomlwriter"
```

## Element Catalog

| ID | Name | Type | Responsibility | Code Pointer |
|---|---|---|---|---|
| <a id="e9-engram-cli-binary"></a>E9 | engram CLI binary | Container in focus | Go binary entry cmd/engram/main.go, wiring internal/cli.Targets into targ.Main. | — |
| <a id="e3-claude-code"></a>E3 | Claude Code | External system | Execs the binary as a subprocess each time the agent's Bash tool runs an engram subcommand; consumes its stdout. | — |
| <a id="e4-claude-code-memory-surfaces"></a>E4 | Claude Code memory surfaces | External system | Read-only inputs: project + user CLAUDE.md, .claude/rules/*.md, auto-memory, skill frontmatter. | — |
| <a id="e5-anthropic-api"></a>E5 | Anthropic API | External system | Haiku model used for recall ranking, snippet extraction, and learn-time classification. | — |
| <a id="e6-engram-memory-store"></a>E6 | Engram memory store | External system | On-disk feedback + fact TOML state under ~/.local/share/engram/memory/. | — |
| <a id="e20-main-go"></a>E20 | main.go | Component | Process entry. Calls cli.SetupSignalHandling and forwards cli.Targets into targ.Main. No business logic; excluded from coverage per project convention. | [../../cmd/engram/main.go](../../cmd/engram/main.go) |
| <a id="e21-cli"></a>E21 | cli | Component | Subcommand dispatch for all five subcommands plus the embedded business-logic handlers for show, list, learn, and update (show.go, list.go, learn.go, update.go). Owns *Args arg structs and thin I/O adapter shims (os.ReadFile, os.ReadDir, os.UserHomeDir, os.Getwd, os.Getenv, exec.Command). Wires DI interfaces into the pure-logic packages. See Drift Notes — these handlers are intended to live as peer packages alongside `recall`. | [../../internal/cli](../../internal/cli) |
| <a id="e22-recall"></a>E22 | recall | Component | Recall pipeline: orchestrator + per-source phases (CLAUDE.md, auto-memory, skill, transcript). Pure logic; consumes Finder, TranscriptReader, Summarizer, MemoryLister via DI. Currently the only subcommand with a dedicated package; the other four are absorbed into cli. | [../../internal/recall](../../internal/recall) |
| <a id="e23-context"></a>E23 | context | Component | Session transcript parsing: reads .jsonl lines, computes deltas, strips tool-summary noise within a budget. | [../../internal/context](../../internal/context) |
| <a id="e24-memory"></a>E24 | memory | Component | Shared types and read-modify-write helpers for feedback (feedback/) and fact (facts/) memory TOML files; defines FactsDir / FeedbackDir paths under the data directory. | [../../internal/memory](../../internal/memory) |
| <a id="e25-externalsources"></a>E25 | externalsources | Component | Reads ranking inputs outside the engram store: project + user CLAUDE.md, .claude/rules/*.md, auto-memory, skill frontmatter; resolves frontmatter imports; caches per-discover-call. | [../../internal/externalsources](../../internal/externalsources) |
| <a id="e26-anthropic"></a>E26 | anthropic | Component | Anthropic Messages API client. Owns the HTTP request, error sentinels, and exposes a CallerFunc consumed by recall.NewSummarizer. Pinned to claude-haiku-4-5-20251001. | [../../internal/anthropic](../../internal/anthropic) |
| <a id="e27-tokenresolver"></a>E27 | tokenresolver | Component | Resolves the Anthropic API token from ANTHROPIC_API_KEY env or, on darwin, the macOS Keychain via security. Documented to never return a non-nil error. | [../../internal/tokenresolver](../../internal/tokenresolver) |
| <a id="e28-tomlwriter"></a>E28 | tomlwriter | Component | TOML serialization for new / updated feedback and fact memory files. | [../../internal/tomlwriter](../../internal/tomlwriter) |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-claude-code-main-go"></a>R1 | Claude Code | main.go | execs the binary as a subprocess (Bash tool) | Subprocess exec |
| <a id="r2-main-go-cli"></a>R2 | main.go | cli | builds CLI targets and runs targ.Main | Go function call |
| <a id="r3-cli-recall"></a>R3 | cli | recall | delegates the recall pipeline (Orchestrator + phases) to the dedicated recall package — the one subcommand currently extracted from cli (see Drift Notes) | Go function call |
| <a id="r4-cli-memory"></a>R4 | cli | memory | reads / writes feedback + fact TOML through memory types and helpers | Go function call |
| <a id="r5-cli-externalsources"></a>R5 | cli | externalsources | discovers external source paths and shares the cache via discoverExternalSources | Go function call |
| <a id="r6-cli-anthropic"></a>R6 | cli | anthropic | builds the Anthropic caller used by recall.NewSummarizer | Go function call |
| <a id="r7-cli-tokenresolver"></a>R7 | cli | tokenresolver | resolves API token (env or Keychain) before any LLM call | Go function call |
| <a id="r8-recall-context"></a>R8 | recall | context | reads + strips session transcripts within budget | Go function call |
| <a id="r9-recall-memory"></a>R9 | recall | memory | lists memories during recall ranking | Go function call |
| <a id="r10-recall-anthropic"></a>R10 | recall | anthropic | ranks candidates and extracts snippets via Haiku (through DI Summarizer) | Go function call |
| <a id="r11-recall-externalsources"></a>R11 | recall | externalsources | reads CLAUDE.md / rules / auto-memory / skill frontmatter (cached) | Go function call |
| <a id="r12-cli-tomlwriter"></a>R12 | cli | tomlwriter | writes new TOML on learn / remember / update | Go function call |
| <a id="r13-cli-claude-code"></a>R13 | cli | Claude Code | prints briefings, recall results, and list / show output to stdout | stdout |
| <a id="r14-anthropic-anthropic-api"></a>R14 | anthropic | Anthropic API | HTTPS POST /v1/messages (Haiku) | HTTPS, Anthropic Messages API |
| <a id="r15-externalsources-claude-code-memory-surfaces"></a>R15 | externalsources | Claude Code memory surfaces | reads project + user CLAUDE.md, .claude/rules, auto-memory, skill frontmatter | Local file reads (read-only) |
| <a id="r16-memory-engram-memory-store"></a>R16 | memory | Engram memory store | reads existing feedback + fact TOML during recall / list / show | Local file I/O, TOML |
| <a id="r17-tomlwriter-engram-memory-store"></a>R17 | tomlwriter | Engram memory store | writes new feedback + fact TOML on learn / remember / update | Local file I/O, TOML |

## Cross-links

- Parent: [c2-engram-plugin.md](c2-engram-plugin.md) (refines **E9 · engram CLI binary**)
- Siblings:
  - [c3-hooks.md](c3-hooks.md)
  - [c3-skills.md](c3-skills.md)
- Refined by: *(none yet)*

## Drift Notes

- **2026-04-26** — Subcommands are not architectural equals in code: of recall, show, list, learn, and update, only recall has its business logic extracted into a peer package (internal/recall). The other four handlers live as files inside internal/cli/ (show.go, list.go, learn.go, update.go). Reason: Persisted misalignment between intent (subcommands as equals, each with its own package) and current code. Resolution: when next touching show / list / learn / update business logic, prefer extracting to internal/<subcommand>/ packages with DI interfaces, mirroring internal/recall. Update this diagram and the catalog row for E21 once peer packages exist.
