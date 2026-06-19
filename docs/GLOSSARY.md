# Engram Glossary

Standardized vocabulary for the engram project. Where a term has variants in
the wild, the **canonical form** is named here; variants are listed for
recognition. Inconsistencies that need a decision live in
[`triage.md`](triage.md).

## Top-level concepts

### engram
The project, the CLI binary, and the broader system of "skills + binary +
vault" that gives LLM agents persistent memory. When ambiguity matters,
disambiguate as *engram (project)*, *engram (binary)*, or *engram (CLI)*.

### vault
The on-disk Obsidian directory that holds the agent's persistent memory.
Resolved in order: `--vault` flag → `ENGRAM_VAULT_PATH` env → default
`$XDG_DATA_HOME/engram/vault` (fallback `~/.local/share/engram/vault`).
Always written and read by the `engram` binary; never by skills directly.
Full form: **agent-memory vault**. Short form **vault** is preferred in
running prose once context is established.

### zettelkasten
The vault's organizational style — atomic notes connected via wikilinks, with
Luhmann-ID lineage and Maps of Content for synthesis. Used as both noun
("the vault is a zettelkasten") and adjective ("zettelkasten-style").

### skill
A markdown file (`SKILL.md`) that defines an agent behavior, installed into
each harness's skills directory by `engram update`. Engram ships four:
[`recall`](#recall-skill), [`learn`](#learn-skill), `please` (end-to-end
orchestration), and `route` (delegation doctrine — agent/model/effort selection).

### harness
A coding-agent host that runs skills. Engram supports two: **Claude Code**
and **OpenCode**. The plural is *harnesses*. When the same concept appears
in code, it is sometimes called a *source* (see triage).

### binary
The compiled `engram` Go program. Subcommands: `learn`, `query`, `embed`,
`ingest`, `show`, `amend`, `activate`, `update`. The binary handles all I/O
(vault read/write, chunk indexing, file locking); skills handle behavior
and prompting.

---

## Vault structure

### Permanent (note)
An atomic, principle-stated note in `<vault>/Permanent/`. Atomic here means
*one coherent topic with its full load-bearing detail and complete sets* —
not one micro-fact; over-fragmenting a topic across notes harms retrieval
(see `learn/SKILL.md`). Filenames follow
`<luhmann-id>.<YYYY-MM-DD>.<slug>.md`. Plural: *Permanents*. The folder
keeps a capital P. In running prose, prefer *permanent note* (lowercase
noun) for the concept and *Permanent/* (capital, with slash) for the folder.

### MOC (Map of Content)
A note in `<vault>/MOCs/` whose body is framing prose synthesizing related
permanent notes. Plural: **MOCs**. The full form **Map of Content** (capital
M, capital C, no plural "s") is the canonical expansion. Folder name keeps
the trailing "s": `MOCs/`.

### Luhmann ID
The position string in a note's filename (e.g. `87`, `4a`, `4g1a`). Encodes
lineage: `4a` is a continuation of `4`; `4b` is `4a`'s sibling; `4a1` is
`4a`'s child. Allocated under a file lock by the binary; never computed by
the agent. Canonical capitalization: **Luhmann ID** in prose,
`luhmann` (lowercase) in frontmatter and flag values.

### wikilink
A bracketed reference of the form `[[<luhmann-id>.<date>.<slug>]]` or a
shorter `[[<slug>]]` form. Wikilinks appear in prose with surrounding
context that explains the connection (the "per-link rationale" required by
`--relation`).

### slug
The kebab-case tag at the end of a note filename. Passed via `--slug` on
`engram learn`. Variants seen: *kebab-case tag*, *slug*. **Canonical:
slug.**

### bootstrap
The first-time creation of a missing vault (or its child directories and
metadata files) on first `engram learn`. Other subcommands do **not**
bootstrap — they error out so the user notices.

---

## Recall

### recall (skill)
The skill at `skills/recall/SKILL.md`, invoked as `/recall` in a harness or
self-fired by the agent. Walks the vault and surfaces relevant notes.

### cascade
The recall loop that expands the frontier round by round, following
wikilinks from notes scored relevant in the previous round, until ≥100
notes are surfaced or the frontier empties.

### frontier
The set of notes to read this round. The **initial frontier** is anchors ∪
recent; **expanded frontiers** are the wikilink targets of relevant notes
from the prior round.

### anchors
Legacy term from the pre-`engram query` recall cascade — every MOC plus
the in-degree winner of each MOC-less connected component. **In code** the
same concept is named `StartingPoints` (see triage); still computed inside
the vault graph package but no longer exposed via a binary subcommand.
Hubs from `engram query` (top-5 in-degree within the query subgraph) are
the live successor concept.

### explicit query
The user-named topic (or the agent-formed topic from context). One of the
two retrieval streams; the other is the **situational baseline**.

### situational baseline
Step-1 phrases derived from ambient features of the current situation
(repo, language, what's loaded into context, the operation underway).
Surfaces what the user didn't know to ask about.

### Step 0 / Step 1 / …
Numbered pipeline stages in the recall skill. Step 0 = print Ask/Situation
/Plan; Step 1 = phrase queries; Step 2 = form explicit query; Step 3 =
cascade; Step 4 = synthesis.

### surfaced notes
Notes that scored relevant during the cascade. Distinct from *read* (every
frontier note is read; only some are surfaced).

### subgraph (query-time)
The set of notes `engram query` operates on after expanding 3 hops via
authored wikilinks (both outgoing and inbound) from each direct hit. Hard
cap of 200 notes. Notes without compatible `.vec.json` sidecars are
excluded. Reported in the query payload as `budget.subgraph_size` /
`subgraph_size_capped`.

### cluster (query-time)
A subgraph partition produced by k-means with k=2..7 chosen by max
silhouette score. Deterministic per query against an unchanged vault.
Each cluster has a `representative` (member closest to centroid) and
`members` (path + score + is_representative). Tiny subgraphs
(`subgraph_size < 6`) skip clustering entirely; subgraphs with
max-silhouette < 0.10 return `clusters: []`.

### hub (query-time)
A subgraph note with high in-degree (counted only within the subgraph).
Top 5 returned. Hubs appear in `items.provenances` as the `hub` role with
the `in_degree` field populated.

### provenances (item roles)
A query item's `provenances` list names every role it fills: `direct`
(top-k cosine hit), `cluster_rep` (cluster representative), `hub` (top-5
by in-degree). Items dedup across roles; a path appears once regardless
of how many roles it fills. Item ordering: provenance count desc → role
priority (direct > cluster_rep > hub) → score desc.

---

## Learn

### learn (skill)
The skill at `skills/learn/SKILL.md`, invoked as `/learn` or fired after
recall-flow work. Writes new notes to the vault.

### `engram learn` (subcommand)
The binary subcommand. Two forms: `engram learn feedback` and
`engram learn fact`. Both require `--source` and take body content via
flags (stdin is ignored). The `moc` subcommand was retired after the F4
migration (the 25 historical MOCs are archived for audit in
`<vault>/_legacy/MOCs/`); the `episode` form was retired with the lazy-L2
work (issue 649) — chunks in the chunk index are the episodic layer now.

### Feedback (note type)
A note recording something to do differently next time — user corrections,
dead-ends, failed approaches. Auto-generated opener: `Lesson learned: …`.

### Fact (note type)
A note recording how something actually works — tool behaviors, idioms,
conventions, gotchas. Auto-generated opener: `Information learned: …`.

### recall-mirror test
The gate every candidate note must pass before being written: *"Would a
future agent, querying for the same kind of work this candidate's scratch
list targets, see this note in their cascade?"* Per-candidate, not
session-global — current-locus candidates target this session's work,
retro-locus candidates target the injecting agent's work. If no, rephrase.
If still no, drop.

### injection locus
The work that *caused* a lesson, distinct from the work that *surfaced*
it. **Current-locus** = the mistake or discovery originated in this
session. **Retro-locus** = the cause is in a prior session, even though
the candidate may have surfaced through current-session work (or come
from prior-session chunk history surfaced by recall). Discriminated cheaply by `git
blame` / `git log` on the offending line, prior-session transcript
content, or behavioral inference for purely conceptual mistakes. Locus
classification determines which framing path applies in §2.

### scratch list
The 5–15 short queryable phrases written internally for a candidate
before scoring it. One scratch list per candidate (not one per session):
in Path A copied from the recall whose Step 0/1 bracketed the candidate's
segment of work; in Path B reconstructed from what a current-session agent
doing that candidate's kind of work would have queried at the time; in
Path C reconstructed from what the **injecting** agent (prior session)
was doing — sourced from git blame, prior-session transcript, or
behavioral inference.

### Path A / Path B / Path C
Per-candidate framing selection, chosen after classifying the candidate's
injection locus. **Path A** = current-locus, a recall ran during *this
candidate's* segment of work (lift its Step 1 phrases verbatim). **Path
B** = current-locus, no recall bracketed this candidate (reconstruct what
Step 1 would have been at the time). **Path C** = retro-locus —
the lesson's cause is in a prior session, regardless of whether a
current-session recall bracketed the discovery (reconstruct the scratch
list from the injecting agent's situation via git blame / prior-session
transcript / behavioral inference). Path C overrides Path A: a retro-locus
candidate must not be framed against the current-session recall, even when
that recall bracketed the discovery. Selection is per-candidate, not
session-global.

### `--target` / `--position`
Luhmann placement flags. `--position top` creates a new top-level note;
`--position continuation` extends `--target <id>`; `--position sibling`
creates a parallel branch at the same level. The binary computes the
actual ID under lock.

### `--relation`
Repeatable flag that adds one `Related to:` bullet per occurrence. Format:
`--relation "<wikilink-target>|<per-link rationale>"`. Every related entry
must include rationale; bare wikilinks are rejected.

### `--source`
Required provenance field on every `engram learn` invocation. Format:
`session log <project>, <YYYY-MM-DD HH:MM UTC>, context: <short
description>` for session-derived notes.

### `--project` (write side)
Optional kebab-case slug naming the project a note belongs to. Set on
write via `engram learn {fact,feedback} --project <slug>` and
rendered as `project: <slug>` in frontmatter below `source:`. Absent on
notes that capture universal principles. Project name still does not
belong in `--situation` — `--situation` stays retrieval-shaped per the
recall-mirror test; `--project` is the metadata home for projectness so
cross-project queries become answerable without polluting the situation
phrase.

### `--project` (read side)
Optional filter on `engram query`. When set, drops items whose
frontmatter `project:` does not match. The underlying wikilink graph is
unaffected — cross-project bridges still bridge during the 3-hop BFS,
the filter only restricts which items are emitted. Items with elided
content (no body in the payload) are dropped under a non-empty
`--project` since a match cannot be verified.

### `--issue`
Optional identifier for the originating GitHub / Jira / etc. issue. Set
on write via `engram learn {fact,feedback} --issue <id>` and
rendered as `issue: "<id>"` (quoted to survive YAML's numeric coercion
on read-back) in frontmatter below `project:`. Free-form non-whitespace
string — `636`, `#636`, `GH-636`, `PROJ-1234` are all valid. Recorded
for provenance; no read-side filter.

---

## Transcript

### transcript
The recorded content of one session, read by the binary (via `engram
ingest`) from a harness's on-disk store. Claude Code transcripts are JSONL
files; OpenCode transcripts come from a SQLite database. A *session* is the
time-bound interaction; a *transcript* is its serialized record. (The
standalone `engram transcript` subcommand and its per-harness progress
marker were retired with the lazy-L2 work — issue 649; `internal/transcript`
is retained as the reader for `engram ingest`.)

### session
One conversation between a user and an agent in a harness. Plural:
*sessions*. Sessions produce transcripts; the binary reads transcripts.

### `engram ingest` (subcommand)
Merge-appends session transcripts + markdown into the per-source chunk
index — re-chunks/re-embeds only changed content, never deletes
(append-only chunk history). `--auto` sweeps all known sources; called by
`/learn` and `/recall`. Chunks are the episodic layer (raw event memory)
and are matched/clustered alongside L2 notes at recall; chunk-grounding is
recorded as frontmatter provenance, not as wikilinks.

### `--from <date|all>`
Overrides the marker by scanning from an explicit date (`YYYY-MM-DD`) or
from the Unix epoch (`all`). The latter scans everything.

---

## CLI conventions

### subcommand
A named operation on the binary: `learn`, `query`, `ingest`, `update`,
etc. The whole CLI is a single binary with subcommands, never a sprawl of
separate executables.

### `engram update`
Installs/refreshes skills and commands into every detected harness, and
reinstalls the binary via `go install`. `--dry-run` shows the diff
without writing.

### XDG paths
Engram follows XDG basedir conventions:
- Data: `$XDG_DATA_HOME/engram/` (vault).
- State: `$XDG_STATE_HOME/engram/projects/<slug>/` (markers).
Fallbacks: `~/.local/share/...` and `~/.local/state/...`.

### DI (dependency injection)
Architectural rule: no function in `internal/` calls `os.*`, `http.*`, or
any I/O directly. All I/O goes through injected interfaces, wired at the
CLI edges. Tests use `imptest`-generated mocks.

### targ
The build tool wrapping `go test`/`go vet`/`go build`. **Always** invoke
`targ build`, `targ test`, `targ check-full` — never the underlying Go
commands.

---

## Authoring & process vocabulary

### candidate (note)
A potential note identified from completed work (transcript content
surfaced by recall, or the current session's activity), before passing the
recall-mirror test. Becomes a written note or is dropped with a reason.

### subagent
A parallel worker spawned by a skill to read or score notes without
polluting parent context. Used during cascade rounds with ≥10 frontier
notes.

### coordinator
A serial pass after parallel writer subagents finish, whose job is
cross-document references the parallel writers can't see.

### contradiction
Two surfaced notes making incompatible claims about the same thing. The
vault preserves contradictions; recall surfaces both; learn writes a
`Related to:` bullet whose rationale names the discrepancy. Never
smoothed.

---

## Status / disposition values

| Term | Meaning |
|------|---------|
| `top` | Luhmann position for a brand-new top-level note |
| `continuation` | Luhmann position extending an existing note (e.g., `1` → `1a`) |
| `sibling` | Luhmann position at the same level (e.g., `1a` → `1b`) |
| `--dry-run` | Flag on `engram update` that previews without writing |
