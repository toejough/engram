# Engram

> ⚠️ **Breaking change.** The pre-vault TOML memory-record storage layer
> (`~/.local/share/engram/memory/`) was removed. Engram now writes only
> to an agent-memory Obsidian vault. Migration from the
> old layout is not automated. An LLM should be able to migrate easily.

## Overview

Engram gives Claude Code and OpenCode agents persistent memory via a zettelkasten-style vault. Two skills — `recall` and `learn` — read from and write to an agent-memory vault on demand; at their write sites they hand off to `write-memory`, a worker skill that composes and executes the vault-write commands. A further skill, `please`, orchestrates end-to-end work by sequencing recall, learn, and other skills around a user's `<ask>`, and `route` encodes the delegate-everything doctrine `please` draws on to staff its subagents. `recall`, `learn`, and `write-memory` shell out to the `engram` binary; `please` and `route` are pure meta-orchestration.

After a few months of use, the vault's wikilink graph looks like this in Obsidian — each dot is a note, each line a `[[wikilink]]`; the ~25 vocab term-notes form visible hubs (each `Vocab:` body line points member→term, so term nodes accumulate spokes), dense clusters are groups of related notes, and the connective tissue reflects thematic proximity:

![Obsidian graph view of an engram vault](docs/images/vault-graph.png)

## Installing

Requires Go 1.25+ on `PATH`.

1. Install the binary:

   ```bash
   go install github.com/toejough/engram/cmd/engram@latest
   ```

   Make sure `$GOBIN` (or `$GOPATH/bin`, default `~/go/bin`) is on your `PATH`.

2. Copy the skills and commands into every detected harness's user directory:

   ```bash
   engram update                 # install / refresh
   engram update --with-guidance # also deploy recall-firing guidance to ~/.claude/engram/recall.md (Claude Code; opt-in)
   engram update --dry-run       # show what would change
   ```

   `engram update` writes Claude Code skills to `~/.claude/skills/` and OpenCode skills + commands to `~/.config/opencode/{skills,commands}/`. Run it again any time to upgrade — it also reinstalls the binary via `go install`. `--with-guidance` additionally deploys `guidance/recall.md` to `~/.claude/engram/recall.md` for CLAUDE.md `@import` (Claude Code; opt-in). It's a **one-time opt-in** — once your CLAUDE.md imports the file, plain `engram update` keeps it current (like skills). Until then, plain `engram update` prints a one-line hint.

## Skills

| Skill | What it does |
|-------|--------------|
| `recall` | Surfaces relevant notes and raw chunks via a single `engram query` call: a clustered **relevance** channel (recency-biased per-phrase cosine over notes+chunks → bounded matched set → one unified chunk+note clustering that builds `candidate_l2s` from within-cluster top-5 plus tag-nominated notes sharing a vocab term with the top-3 delivered notes) plus an un-clustered **recency** channel (the newest chunks, tagged `recent`). For each cluster it judges coverage inline (covered/near/absent) and crystallizes via `engram amend` (update an existing note) or `engram learn` (create one), activates only the notes it actually used, then reports whether the surfaced memory changed the agent's plan. |
| `learn` | Captures the session's explicit lessons — corrections, explicit save-requests, and self-discovered reversals — as permanent vault notes via `write-memory` handoffs. Along the way it mechanically sweeps every conversation and doc into the searchable chunk index (`engram ingest --auto`) and checks vocab liveness (`engram vocab stats`, auto-refitting when due), so raw event memory stays current even when no explicit lesson exists. |
| `write-memory` | Executes a vault write handed off by `recall` or `learn`: composes the `engram learn`/`amend` command from the provided fields, runs it, verifies the result, and reports the written note path. Never fires on its own judgment — a handoff is required. |
| `please` | Drives an ask end-to-end through a fixed seven-step workflow — capture, orient, plan, execute (TDD), document, complete, capture. Sequences `recall`, `learn`, and other available skills; tracks each step on the task list. Four adversarial review gates dispatch fresh per-angle reviewer subagents over the plan, each refactor, touched docs, and outward prose, blocking step completion until findings are resolved. Triggers on `/please <ask>` and natural-language phrasings of the same intent. |
| `route` | Encodes the delegate-everything doctrine: guides subagent selection (agent type, model, effort) rather than doing object-level work. Easy work goes to a cheap model (not skipped), complex work is decomposed before dispatch, and every dispatched subagent recalls first. `please` consults it when staffing gate reviewers. |

See `skills/recall/SKILL.md`, `skills/learn/SKILL.md`, `skills/write-memory/SKILL.md`, `skills/please/SKILL.md`, and `skills/route/SKILL.md` for the full skill definitions.

## Vault location

Engram reads and writes a zettelkasten vault. Resolution order:

1. `--vault <path>` flag
2. `ENGRAM_VAULT_PATH` environment variable
3. `$XDG_DATA_HOME/engram/vault` (fallback: `~/.local/share/engram/vault`)

On first `engram learn` against a missing vault, the directory is
bootstrapped with a minimal `.obsidian/` config so Obsidian recognizes
it, a `.gitignore`, and a `README.md`. Other subcommands do not
bootstrap — they error with "vault not found" so the user notices.

Vault layout (flat since the 2026-06-12 flat-vault migration — notes live at the
vault root; the `Permanent/` and `MOCs/` subdirectories are retired and ignored
by the scanner):

```
<vault>/
  <luhmann-id>.<YYYY-MM-DD>.<slug>.md   atomic notes at the root
  <luhmann-id>.<YYYY-MM-DD>.<slug>.vec.json   sibling embedding sidecar
```

## Binary commands

```
engram learn feedback --slug ... --source ... --situation ... --behavior ... --impact ... --action ... [--project <slug>] [--issue <id>]
engram learn fact     --slug ... --source ... --situation ... --subject ... --predicate ... --object ... [--project <slug>] [--issue <id>]
engram learn qa       --slug ... --source ... [--question <text>] [--answer <text>|--answer-file <path>] [--contributors <basename>...] [--certainty high|medium|low]   Write a QA pair (Q+A notes) to the vault. --slug and --source required; --answer and --answer-file are mutually exclusive; --contributors repeatable, validated against the vault; --certainty defaults to medium.
engram embed apply [--all|--missing|--stale|--force|--dry-run]   (Re-)embed notes per selection (default: missing)
engram embed status                    Report counts per state (total / with-embeddings / without / stale / incompatible / broken)
engram query --phrase <p> [--phrase <p>...] [--limit N] [--project <slug>] [--chunks-dir <dir>] [--content-budget N] [--recent-fill N] [--lazy-chunks]   Semantic search over vault notes + chunk index; YAML output. Recency-weights chunks AND notes. Builds a bounded matched set (per-phrase top-30, union/dedup, relevance floor 0.25, cap ~300), clusters it in one pass (AutoK k-means), emits `candidate_l2s: [{path, cosine, content}]` per cluster — within-cluster top-5 notes plus tag-nominated notes sharing ≥1 vocab term with the top-3 delivered notes (budget: `tag_nominations_added`/`dropped`, pool cap 40/cluster) plus superseded-note ride-alongs at the next rank — and appends the newest chunks un-clustered (tagged `recent`; default 25, controlled by `--recent-fill`). `--content-budget` caps how many chunk items render with full content (default 15; later chunks get a snippet). `--lazy-chunks` renders matched chunk items path/score only — the agent fetches evidence on demand via `engram show-chunk`. Activation is agent-driven — the binary emits no `activated` flag. --project restricts items to notes whose frontmatter `project:` matches.
engram query-chunks --phrase <p> [--phrase <p>...] [--limit N] [--chunks-dir <dir>]   Semantic search over the chunk index only (YAML output). Scores chunks by max cosine across phrases; clusters results with AutoK k-means. No vault notes, no recency channel — chunk-space search only.
engram resituate --note <ref> --situation <text>   Rewrite a note's situation field in sync: frontmatter, body opener, and sidecar situation_vector (D4/INV-S2). Both flags required; no --dry-run.
engram check   Run vault-invariant checks; exit non-zero and list FAIL items on violations
engram ingest [--auto]   Merge-append session transcripts + markdown into the per-source chunk index (append-only — re-chunks/re-embeds only changed content, never deletes). `--auto` sweeps all known sources and skips session-log directories whose slugified project path starts with a non-persistent-workspace prefix (slugified forms of `/private/tmp`, `/tmp`, and macOS `$TMPDIR`), preventing eval/test runs from bloating the main index. Configurable via `.engram/sweep.json` (`non_persistent_prefixes` key); bypassed by explicit `--sweep`/`--transcript`/`--markdown` or an isolated index via `ENGRAM_CHUNKS_DIR`. Used by /learn and /recall.
engram prune            Remove chunk index entries whose source file no longer exists (GC). Operator-run; reads the manifest and deletes stale per-source index files. Not part of the recall/learn/please flows.
engram show <ref>   Print a note (frontmatter + body) and its outbound wikilink targets, read-only. One required positional; no --ref flag. (candidate_l2s carry content inline, so /recall no longer shows candidates.)
engram show-chunk <source#anchor> [--chunks-dir <dir>]   Print a chunk's text by its source#anchor id (read-only). Used by /recall with `--lazy-chunks` to fetch a specific chunk's evidence on demand.
engram amend --target <ref> [--activate] [--supersedes "<basename>|<type>|<claim>"] [--chunk-source <source#anchor>...] [--situation/--subject/--predicate/--object | --behavior/--impact/--action ...]   Amend a note in place: merge chunk-source provenance (idempotent), overwrite only supplied content fields, re-embed only on a content change; `--activate` bumps `LastUsed`; `--supersedes` writes typed supersession frontmatter + inverse + body line. The /recall update path: covered link-enriches, near re-synthesizes content.
engram activate --note <path> [--note <path>...]   Mark note(s) as recently used — bumps `LastUsed` in the sidecar so usefulness keeps useful notes fresh (called by /recall on only the notes the agent actually used). `--note` paths are vault-relative (resolved against the vault root / `ENGRAM_VAULT_PATH`); absolute paths are used as-is.
engram vocab bootstrap --seed <yaml> [--floor <f>]     Seed vocab term-notes from the validated term set (--seed, required); embed them; dual-channel tag all existing notes (body Vocab: line + vocab: frontmatter); regenerate vocab.index.md. --floor sets the minimum cosine similarity for vocab assignment (default 0.35). Idempotent.
engram vocab propose --term <t> --description <d>  LLM-gated: create a new term note if no existing term covers it and projected attachment ≤ 20% of vault (~$0.05/proposal). Both flags required.
engram vocab stats                     Per-term member counts, vault untagged-rate, hub terms (> 25% of vault), orphan terms (< 2 members), version staleness.
engram vocab refit                     LLM-judged: merge orphans, split hubs, rename terms; rewrites member Vocab: lines + frontmatter; major version bump; index regen (~$2).
engram update [--with-guidance]        Refresh binary and harness skills/commands ([--dry-run]); --with-guidance also deploys guidance/recall.md to ~/.claude/engram/recall.md (Claude Code; opt-in; OpenCode deferred)
```

## Semantic search (`engram query`) and the embed-on-write pipeline

Engram bundles an embedding model (`sentence-transformers/all-MiniLM-L6-v2`, 384 dims) inside the binary via `go:embed`. Inference runs in pure Go through [Hugot](https://github.com/knights-analytics/hugot) + [GoMLX](https://github.com/gomlx/gomlx)'s `simplego` backend — no CGO, no daemon, no API key.

Each note (`<id>.<date>.<slug>.md`) has a sibling `.vec.json` sidecar at the vault root (flat layout):

```
132.2026-05-23.foo.md
132.2026-05-23.foo.vec.json
```

Sidecar shape (dual-vector):

```json
{
  "schema_version": 1,
  "embedding_model_id": "minilm-l6-v2@384",
  "dims": 384,
  "situation_vector": [-0.044, -0.043, ...],
  "body_vector": [-0.012, 0.031, ...],
  "content_hash": "sha256:...",
  "last_used": "2026-06-20"
}
```

Each note carries **two vectors**: `situation_vector` (embedding of the `situation:` frontmatter field, or body if absent) and `body_vector` (embedding of the markdown body). At query time, `bestVector` picks the axis — whichever of situation or body cosines higher against the query phrase — so both angles compete. `last_used` is updated by `engram activate` and drives ACT-R-style recency decay: frequently-retrieved notes rank higher; never-retrieved notes fade.

`content_hash` is sha256 over the note's situation + body text so adding a machine-written `Vocab:` line or `Supersedes:` line doesn't trigger re-embed.

Pipeline behavior:

- `engram learn` auto-embeds the new note before returning. Embedder failure is a warning, not an error — the Luhmann write is atomic, and `engram embed apply --missing` will fill the gap later.
- `engram embed status` reports per-state counts: `ok` / `missing` (no sidecar) / `stale` (body changed) / `incompatible` (different model_id) / `broken` (malformed JSON or dims mismatch).
- `engram embed apply` modes:
  - `--missing` (default): only notes without sidecars
  - `--stale`: also re-embed notes whose body hash changed (and broken sidecars)
  - `--force`: also re-embed sidecars whose model_id differs from the bundled model
  - `--all`: every note, regardless of state
  - `--dry-run`: report what would change without writing
- `engram query` embeds each `--phrase`, takes the top-30 hits (notes + chunks, recency-biased cosine) per phrase, unions across all phrases (dedup keeping max score), drops items below a relevance floor (baseScore < 0.25), and caps the matched set at ~300. AutoK k-means (k=2..7, silhouette-selected) clusters the matched set once; each cluster carries `candidate_l2s: [{path, cosine, content}]` — within-cluster top-5 notes plus tag-nominated notes sharing a vocab term with the top-3 delivered notes (budget: `tag_nominations_added`/`dropped`, cap 40/cluster) plus superseded-note ride-alongs at next rank. The 25 newest chunks by ingest time (default; configurable via `--recent-fill`) are appended un-clustered (tagged `recent`). Empty vault → `items: []` exit 0. Vault with notes but no sidecars → error with the `engram embed apply --all` recovery hint.

Inputs longer than 1500 chars are truncated to fit MiniLM-L6's 512-token positional limit. For engram's 200–500-word notes this is a non-issue; long notes lose tail context but still embed cleanly.

## Project structure

```
cmd/engram/          CLI entry point (thin wiring layer)
internal/            Business logic (DI boundaries)
  chunk/             Splits transcripts/markdown into embedding-sized chunks for the chunk index (pure string logic, no I/O)
  cli/               CLI command wiring (targ targets)
  cluster/           k-means clustering with silhouette-based auto-K, for recall clustering
  context/           Transcript processing
  debuglog/          Structured debug logging
  embed/             Embedder interface + Hugot/GoMLX backend, sidecar I/O, state classification
  luhmann/           Luhmann-ID allocation under file lock
  transcript/        Session transcript reading (Claude Code JSONL), read by engram ingest
  update/            Self-refresh subcommand
  vaultgraph/        Vault traversal (wikilink graph, note scanning)
skills/              Source for the recall, learn, write-memory, please, and route skills
commands/            Source for OpenCode slash commands
```

## Development

- `go install ./cmd/engram` — install the binary (targ has no `build` target; it covers check/test/lint only)
- `targ test` — run unit + integration tests
- `targ check-full` — lint + coverage (use this to see ALL errors at once)
- Never run `go test` / `go build` / `go vet` directly — use `targ`

## Design principles

Design principles and their rationale live in `docs/architecture/adr.md` (ADR-0001..0003) — the authoritative source; this README covers orientation and the CLI reference only.

## Documentation

See `docs/README.md` for the full documentation index — glossary, roadmap, shipped features, architecture, and proven results, one obvious place to start.
