# Plan — L1 C4 system context diagram for engram

## Deliverable

One file: `docs/architecture/c1-system-context.md`. Inline mermaid block (no SVG
pre-rendering, because the `c4` skill's `targ c4-render` target does not exist in
this repo). The path and format are per user directive.

## Why not the full `c4` skill ceremony

The `c4` skill expects `architecture/c4/` plus a JSON spec consumed by `targ
c4-l1-build`, `targ c4-render`, `targ c4-audit`. None of those targets exist in
engram's `targ`. The user explicitly requested `docs/architecture/` + markdown +
mermaid. Honor the directive. Substantive C4 conventions from the skill
(classDef styling, S<n>/R<n> IDs, element catalog + relationships tables, click
directives to in-page anchors) still apply.

## L1 elements (Tier-2 verified against engram source)

| ID | Type | Element | Source evidence |
|---|---|---|---|
| S1 | Person | Engram operator | Human |
| S2 | In-scope system | Engram | `cmd/engram/main.go`, `internal/`, `skills/` |
| S3 | External | LLM coding harness (Claude Code, OpenCode) | invokes engram skills + CLI; `internal/update/update.go:98-169` probes `~/.claude` and `~/.config/opencode/` |
| S4 | External | Agent-memory vault | `internal/cli/targets.go:72-75`, vault lock at `internal/cli/cli.go:117-142` |
| S5 | External | Harness session stores (Claude Code JSONL, OpenCode SQLite) | `internal/cli/transcript.go:196`, `internal/transcript/opencode.go:361-367` |
| S6 | External | macOS Keychain | `internal/tokenresolver/tokenresolver.go:46` (`security find-generic-password`) |
| S7 | External | Go toolchain | `internal/update/update.go:357,391,401` (`go list`, `go install`) |

## Relationships

| ID | From | To | Action |
|---|---|---|---|
| R1 | S1 Operator | S3 LLM harness | Directs work via prompts; configures via env vars |
| R2 | S3 LLM harness | S2 Engram | Invokes `/recall`, `/learn`, `/please` slash commands; subprocess-executes the engram CLI |
| R3 | S2 Engram | S4 Vault | Reads & writes notes, MOCs, MEMORY.md under a vault lock (single unidirectional arrow per c4 rule 27c) |
| R4 | S2 Engram | S5 Session stores | Reads JSONL / SQLite transcripts using per-harness markers |
| R5 | S2 Engram | S6 Keychain | On darwin, reads Anthropic API token via `security` subprocess |
| R6 | S2 Engram | S7 Go toolchain | During `engram update`, runs `go install` and `go list -m -json` |

## File structure

1. H1 title
2. One short paragraph: what L1 shows + names the in-scope system
3. Mermaid `flowchart LR` block with classDef, S<n>-ID node labels, R<n>-ID edge
   labels, click directives
4. Element catalog table with anchored IDs
5. Relationships table with anchored IDs
6. Footer: link to L2 (note it is not yet authored)

## Conventions honored

From `c4` skill `references/mermaid-conventions.md` and vault MOC 58:

- classDef block: person, external, container.
- Stadium `id([Name])` for person, rounded `id(Name)` for external, rectangle
  `id[Name]` for in-scope.
- Single unidirectional arrow for read+write (vault R3); no separate read/write.
- Quote node labels containing punctuation; `<br/>` for line breaks.
- `click <id> href "#anchor"` after the diagram; anchors as `<a id="…"></a>` in
  the first table cell.

## Out of scope

- L2/L3/L4 diagrams.
- SVG pre-render (no `mmdc` confirmed in environment; would add `targ c4-render`
  later if SVG is wanted).
- JSON spec sidecar (no build target to consume it in this repo).

## Cleanup

Delete this file (`docs/plan-l1-c4.md`) in step 6 after the diagram lands.
