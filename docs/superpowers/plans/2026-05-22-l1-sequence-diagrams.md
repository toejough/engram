# Plan: L1 sequence diagrams for the four key engram flows

**Date:** 2026-05-22
**Branch:** main (direct — additive doc edit, single file)

## Goal

Add a `## Key flows` section to `docs/architecture/c1-system-context.md` containing four mermaid `sequenceDiagram` blocks, one per user-initiated flow at L1 granularity:

1. **recall** — operator → harness → engram → vault (read)
2. **learn** — operator → harness → engram → session stores + vault (write)
3. **please** — operator → harness orchestrates a 7-step bracket (engram invoked repeatedly)
4. **update** — operator → harness → engram → Go toolchain + harness install root

User confirmed scope (all four flows) and layout (embed in `c1-system-context.md`).

## Convention notes (load-bearing)

- The repo's L1 is hand-authored markdown under `docs/architecture/`, **not** the SNMPR `architecture/c4/*.json` + `targ c4-l*-build` workflow the global `c4` skill describes. Honor the repo's existing convention.
- Mermaid sequenceDiagram parser quirks: no semicolons in labels/notes (use `,` or `.`); no `<angle-brackets>` in Note text (use bare letters or HTML entities).
- Quote any participant or message label containing punctuation.
- Enumerate early-exit branches explicitly (budget exhausted, empty frontier, no transcripts, error paths).
- Render-check one diagram before authoring the others — parser breakage discovered after the fourth is expensive.

## Participants (shared across all four diagrams)

To keep diagrams visually consistent and aligned to the static L1 IDs:

- `actor Op as S1 Operator`
- `participant H as S3 Harness`
- `participant E as S2 Engram CLI`
- `participant V as S4 Vault`
- `participant Tr as S5 Session stores`
- `participant Go as S6 Go toolchain`

Each diagram includes only the participants involved.

## Flow 1: recall

Source: `internal/cli/cli.go:308 runRecall` — three branches (anchors default, recent, follow). All read `vaultgraph.ScanVault` → emit relative paths to stdout. The skill drives the cascade as a loop of subprocess calls.

Sequence shape:

1. Op asks H a question; H decides recall is relevant.
2. H prints Step 0 (Ask/Situation/Plan) — internal cognition, no I/O.
3. H invokes `engram recall` (anchors) → E reads V → emits paths.
4. H invokes `engram recall --recent --limit 20` → E reads V → emits paths.
5. H unions, scores against query + situation phrases.
6. **Loop while budget left and frontier non-empty:** H invokes `engram recall --follow ... --already-read ...` → E reads V → emits expanded frontier.
7. Termination: `≥100 surfaced` OR `empty frontier`. Note both branches.
8. H performs Step 4 synthesis and replies to Op.

## Flow 2: learn

Source: `internal/cli/learn.go:338 runLearn` (write under flock, init vault if missing); `internal/cli/transcript.go:117 advanceAndReportMarker` (read session stores, advance marker, emit status line).

Sequence shape:

1. Op invokes `/learn` (or H self-fires after substantive work).
2. H invokes `engram transcript --mark` → E reads Tr (session JSONL / SQLite) starting from the per-harness marker; E writes the marker forward; E emits the `[engram transcript: scanned ...]` status line.
3. **Early-exit branch:** if no marker exists (first run), E exits non-zero with `transcript: no progress marker`; H prompts the user via `AskUserQuestion` and re-runs with `--from <chosen>`.
4. H reads the in-context conversation + transcript output, identifies candidates, applies recall-mirror test, categorizes Feedback/Fact/MOC.
5. **Loop in parallel per candidate:** H invokes `engram learn {feedback|fact|moc} --slug ... --source ... --situation ... ...` → E acquires flock on V → writes the note → emits the written path.
6. **Early-exit branch:** if vault dir missing, E first calls vault init (bootstrap Permanent/, MOCs/, etc.) under lock before writing.

## Flow 3: please

`/please` is a skill-only orchestration — no dedicated engram subcommand. At L1 the diagram shows the seven-step bracket where each step may or may not invoke E.

Sequence shape (one combined diagram showing the bracket; nested calls are not unrolled):

1. Op invokes `/please <ask>`.
2. H runs Step 1 (opening `/learn`) — invokes E twice (transcript --mark + zero-or-more learn writes).
3. H runs Step 2 (orient — `/recall`) — multiple E calls in a cascade.
4. H runs Step 3 (plan) — internal cognition + possible git commit via FS subprocess.
5. H runs Step 4 (execute, TDD) — repeated FS / build-tool calls; no engram invocation unless the work itself involves engram.
6. H runs Step 5 (document).
7. H runs Step 6 (commit) — git subprocess via `/commit` skill.
8. H runs Step 7 (closing `/learn`) — invokes E twice again.

Frame this as an `alt`/`loop` block over phases rather than unrolling every nested call — the diagram's job is to show the bracket shape, not duplicate the recall/learn flows.

## Flow 4: update

Source: `internal/cli/update.go:199 runUpdate` → `internal/update/update.go:152 Updater.Run`. Two branches: local clone (`go install ./cmd/engram/`) vs remote module (`go install ...@latest` + `go list -m -json` to resolve module root). Then copies skills + commands from source into each harness's install root.

Sequence shape:

1. Op invokes `engram update [--dry-run]`.
2. H spawns E.
3. E walks up from cwd searching for the local module.
4. **alt local:** E invokes Go (`go install ./cmd/engram/`); E reads source skills/commands from the local module dir.
5. **alt remote:** E invokes Go (`go install ...@latest`); E invokes Go (`go list -m -json`) to find the module root; E reads source skills/commands from that dir.
6. E plans copy ops (skills + commands) per detected harness.
7. **Loop per harness, per file:** E writes to the harness install root (`~/.claude/skills/`, `~/.claude/commands/`, OpenCode equivalents).
8. E emits the success report; H displays it.

### Drift note for update flow

The static L1 R5 ("S2 Engram → S6 Go toolchain") models only the `go install` half of the update flow. The update sequence diagram below additionally shows engram writing to the harness install root, which the static diagram does not have an edge for. That gap belongs to the static L1 and is out of scope for this PR; surface it to the user as a follow-up after these diagrams land.

## File-level changes

Single file: `docs/architecture/c1-system-context.md`. Append a new top-level section `## Key flows` after `## Relationships` and before `## Out of scope at L1`. Each flow gets a `### Flow: <name>` subheading, a short prose orientation (1–2 sentences naming the entry point and the L1 edges traversed), the mermaid sequenceDiagram code block, and any explicit drift notes.

Also update the L1 doc's opening paragraph (or add a new sentence) to mention the Key flows section.

## TDD framing

For non-code documentation:

- **RED:** the four flows are not documented; the L1 only shows static structure.
- **GREEN:**
  1. Author `### Flow: recall` first. Visually verify the mermaid block renders without parser errors (mermaid.live or local preview). Confirm participants are correctly labelled, early-exit branches are explicit, and no banned tokens (semicolons in labels, angle brackets in notes) slipped in.
  2. Repeat for learn, please, update — each as a separate sub-step.
- **REFACTOR:** verify cross-flow consistency (same participant aliases, same arrow conventions, same level of detail). Trim duplicate prose if any flow inadvertently re-explains a shared concept.

## Non-goals

- Adding/changing static L1 nodes or edges. R5's incompleteness is surfaced as a drift note and a post-delivery offer, NOT fixed here.
- Authoring L2–L4 diagrams.
- Migrating the L1 to the global c4 skill's SNMPR/json convention.
- Pre-rendering to SVG. Repo convention is mermaid-only; no `.mmd` / `.svg` files alongside the markdown.

## Commit

Single conventional commit: `docs(architecture): add L1 sequence diagrams for the four key flows` via `/commit` skill. Includes the plan file deletion in the same commit (artifact cleanup).
