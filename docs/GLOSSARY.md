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
each harness's skills directory by `engram update`. Engram ships five:
[`recall`](#recall-skill), [`learn`](#learn-skill), `please` (end-to-end
orchestration), `route` (delegation doctrine — agent/model/effort selection),
and [`write-memory`](#write-memory-worker-skill) (vault-write execution on handoff).

### atom
The skill-decomposition concept from the ROADMAP charter: one behavior, one skill
(read-memory, write-memory, route-a-task, orchestrate-a-workflow). Its
reference-card form — mechanical procedures fetched from another skill
mid-procedure — was explored first and superseded by the worker form
(2026-07-04; see `docs/design/2026-07-04-atomic-skills-options.md` and its
postscript for the deployed-measurement caveat).

### write-memory (worker skill)
The skill at `skills/write-memory/SKILL.md`. Executes a vault write handed off by
recall or learn: composes the `engram learn fact|feedback|qa` command from the
handoff fields, runs it, retries on CLI errors (max 2), and reports the written
note path. Parents judge; the worker writes. Amends stay in recall (single-site).

### handoff contract
The field set a parent skill passes when invoking write-memory: **kind**
(`fact|feedback|qa`), the kind's content fields, **source** (provenance string),
optional **chunk-sources**, optional **supersedes** (`basename|type|claim`).
The worker asks for missing required fields rather than inventing content.

### non-triggering description
A skill `description:` that names only parent-instructed invocation ("requires a
handoff — do not fire on your own judgment") so the skill never competes for
autonomous firing. Uncharted in official guidance and the ecosystem; validated
here by non-fire arms (0 autonomous invocations across 6 generic and
vault-adjacent prompts, in both the atom and worker rounds).

### harness
A coding-agent host that runs skills. Engram supports two: **Claude Code**
and **OpenCode**. The plural is *harnesses*. When the same concept appears
in code, it is sometimes called a *source* (see triage).

### binary
The compiled `engram` Go program. Subcommands: `learn`, `query`, `embed`,
`ingest`, `prune`, `show`, `show-chunk`, `amend`, `activate`, `resituate`, `update`, and the `vocab` family
(`vocab bootstrap`, `vocab propose`, `vocab stats`, `vocab refit`). The `--supersedes` flag on `learn`/`amend`
writes typed supersession frontmatter. The binary handles all I/O (vault read/write, chunk indexing,
file locking); skills handle behavior and prompting.

---

## Vault structure

### Permanent (note)
An atomic, principle-stated note — *one coherent topic with its full
load-bearing detail and complete sets*, not one micro-fact (over-fragmenting a
topic across notes harms retrieval; see `learn/SKILL.md`). Notes originally
lived in `<vault>/Permanent/`, retired 2026-06-12 in the flat-vault migration:
they now live flat at the vault root, and the old folder's notes are archived
under `<vault>/_legacy/` (ignored by the scanner). Filenames follow
`<luhmann-id>.<YYYY-MM-DD>.<slug>.md`. Plural: *Permanents*. In running
prose, prefer *permanent note* (lowercase noun) for the concept.

### MOC (Map of Content)
A note whose body is framing prose synthesizing related permanent notes. MOCs
originally lived in `<vault>/MOCs/`, retired 2026-06-12 in the flat-vault
migration: notes now live flat at the vault root, and historical MOCs are
archived under `<vault>/_legacy/MOCs/` (ignored by the scanner). Plural:
**MOCs**. The full form **Map of Content** (capital M, capital C, no plural
"s") is the canonical expansion.

### Luhmann ID
The position string in a note's filename (e.g. `87`, `4a`, `4g1a`). Encodes
lineage: `4a` is a continuation of `4`; `4b` is `4a`'s sibling; `4a1` is
`4a`'s child. Allocated under a file lock by the binary; never computed by
the agent. Canonical capitalization: **Luhmann ID** in prose,
`luhmann` (lowercase) in frontmatter and flag values.

### wikilink
A bracketed reference of the form `[[<luhmann-id>.<date>.<slug>]]` or a
shorter `[[<slug>]]` form. Three roles in a note: (1) **prose links** — running-text references
with human-readable context for the connection (no per-link rationale required by `--relation` —
that flag was removed 2026-07-03); (2) **`Vocab:` member→term links** — `Vocab: [[vocab.<term>]], ...`
body line written by the binary's write-time vocab assigner (dual channel: body line + `vocab:` frontmatter
list; idempotent replace-whole on every write); (3) **`Supersedes:` body line** —
`Supersedes: [[<note>]] — <type>: <claim>`, written by the binary when `--supersedes` is passed.
Structural linking is done by the binary, not by hand.

### slug
The kebab-case tag at the end of a note filename. Passed via `--slug` on
`engram learn`. Variants seen: *kebab-case tag*, *slug*. **Canonical:
slug.**

### bootstrap
The first-time creation of a missing vault (or its child directories and
metadata files) on first `engram learn`. Creates `.obsidian/` (so
Obsidian recognizes the directory), `.gitignore`, and a `README.md`.
Other subcommands do **not** bootstrap — they error out so the user notices.

### sidecar (embedding sidecar)
The `.vec.json` file written alongside each note (e.g.
`87.2026-06-01.foo.vec.json`). Holds a **dual-vector** representation:
`situation_vector` (embedding of the note's `situation:` frontmatter
field, falling back to body if absent) and `body_vector` (embedding of
the markdown body). At query time, `bestVector` picks the axis with the
higher cosine against the query phrase. Also stores `embedding_model_id`,
`dims`, `content_hash` (sha256 over situation + body text), and
`last_used` (YYYY-MM-DD date last activated — drives ACT-R-style recency
decay). Note + sidecar writes are atomic (temp-file + rename) and serialized
under the vault flock (`.luhmann.lock`) across all vault writers — `engram
learn`, `amend`, `resituate`, and `activate`. Chunk-index writes (the separate
`manifest.json` + per-source indices) use a different lock, `.manifest.lock`,
to avoid contention with note writes. `engram embed apply` re-embeds notes in bulk.

### vocab.centroids.json lifecycle fields

Three fields written to `vocab.centroids.json` by the write-time trigger check (2026-07-03):

- **`refit_pending`** (`bool`, omitted when false): set by `checkAndPersistVocabRefitTrigger`
  when any trigger trips; cleared by `engram vocab refit` and `engram vocab bootstrap`.
- **`refit_reason`** (`string`): human-readable reason recorded with the flag, e.g.
  `"growth: 42 notes, 15 days"`, `"untagged: 9.2%"`, or `"hub: token-budget (27%)"`
  (the exact formats emitted by `evaluateVocabTriggers`). Present only when
  `refit_pending` is true.
- **`last_refit`** (`{note_count: int, date: YYYY-MM-DD}`): vault state at the time of the
  last bootstrap or refit — the baseline the growth trigger measures against. Seeded at
  bootstrap and refreshed on each refit.

---

## Vocab

### vocab term-note
A `vocab.<term>.md` file (no Luhmann number) with frontmatter `type: vocab`, `term`, `description`,
`vocab_version`, and `created`. The note body is the term's description prose plus a refit-maintained
exemplar list — this body IS the term's embedding text (a `.vec.json` sidecar is written on embed like any
note). Created by `engram vocab bootstrap` or `engram vocab propose`. Excluded from the matched set,
note-floor, and clustering at query time (`type: vocab` filter). Plural: **vocab term-notes**.

### vocab-index
`vocab.index.md` — a binary-generated Map of Content with frontmatter `type: vocab-index`. Lists one line
per term (`[[vocab.<term>]]` — description — N members), plus `vocab_version`. Human entry point and machine
manifest. Regenerated by `engram vocab` commands; never hand-edited. Excluded from query results like
vocab term-notes.

### vocab nomination
The tag-match extension to `candidate_l2s` (shipped 2026-07-03): notes sharing ≥1 vocab term with the
top-3 delivered notes are **nominated** into each cluster's candidate pool regardless of their cosine to
the cluster centroid. Budget fields `tag_nominations_added` / `dropped` in the query payload report the
count (pool cap 40/cluster). Zero collateral: notes with no shared vocab term are unaffected. A nominated
note may cross cluster boundaries.

### `Vocab:` line
A body line on a member note: `Vocab: [[vocab.<term>]], [[vocab.<term2>]], ...`. Written and maintained by
the binary's write-time vocab assigner on every `learn`/`amend` write — never hand-authored. The binary
also writes a parallel `vocab: [<term>, <term2>]` frontmatter key (Dataview consumer). Both channels are
replaced whole on every write (idempotency rule: replace, never append/merge). Notes with no qualifying term
get no `Vocab:` line and no `vocab:` key (absence = untagged; counted by `engram vocab stats`).

---

## Recall

### recall (skill)
The skill at `skills/recall/SKILL.md`, invoked as `/recall` in a harness or
self-fired by the agent. Issues `engram query` with 10 phrases (deep mode)
and runs the inline coverage-synthesis loop over the returned clusters.

### recall modes — `glance` / `deep`
Recall's two rungs (the depth dial, #662). `deep` (default) = the full procedure, including the
crystallization writes that grow the vault. `glance` (opt-in) = read-only (no crystallization
writes): ~3 phrases; *applies* memory to the decision without growing the vault. Glance escalates
to `deep` for C5 (recency-channel standards) — it surfaces the recent marker but won't elevate it
to a requirement (#661).


### Step 0 / Step 1 / …
Numbered pipeline stages in the recall skill. Step 0 = print Ask/Situation/Plan;
Step 1 = generate 10 phrases (one per fixed angle); Step 2 = run `engram query`;
Step 2.5 = per-cluster coverage synthesis (inline, blocking);
Step 3 = closing synthesis (how memories changed the plan).

### surfaced notes
Notes returned in the `items[]` payload from `engram query`. Includes both
matched notes (Channel 1, relevance) and recent chunks (Channel 2, recency,
tagged `recent`). Coverage synthesis is judged from matched clusters only (Channel
1); the recency channel is situational context the agent reads, not clustered.

### matched set
The bounded set of notes and chunks fed to clustering in the query path.
Built by: per-phrase top-30 (notes+chunks combined, recency-biased cosine) → union
across 10 phrases with dedup keeping max score → drop items below the relevance
floor (baseScore < 0.25) → hard cap at `matchSetCap`=300. Only the matched set
enters clustering (D1 preserved). Recency-channel chunks are appended after
clustering, deduped against the matched set, and never appear in any cluster's
`members[]`.

### relevance floor
The minimum raw cosine (baseScore, pre-recency-decay) required for an item to
enter the matched set: 0.25. Dropping below the floor removes topically-irrelevant
items before clustering. Recency-biased ranking — not the floor — handles
superseded notes (they rank below fresh competition and fall out of the cap).

### recency channel
The second retrieval channel in `engram query`: the newest chunks by
`IngestedAt` (`recentFillChunks`, default 25, configurable via `--recent-fill` / `ENGRAM_RECENT_FILL`), deduped against the matched set, appended to `items[]` with
provenance `recent`, and not added to any cluster. Surfaces recent raw session
context so a post-context-loss agent re-encounters its own narration. Coverage
synthesis is not run against recency-channel items.

### lazy-chunks (query mode)
The `--lazy-chunks` flag (recall's default invocation) on `engram query` renders
matched **and** recency-channel **chunk** items path/source-only — no `content`
field — while notes (`fact`/`feedback`) keep full content inline. Surfaced as
`budget.lazy_chunks: true`. The agent fetches a specific chunk's text on demand via
`engram show-chunk <source#anchor>`. Shrinks the query payload ~−34% (chunks are
supplementary, notes load-bearing — measured 0 chunk fetches in 13/13 realistic
recalls, with on-target fetch when a chunk is the sole carrier of a needed fact).

### candidate_l2s
The `[{path, cosine, content}]` field on each cluster in the query payload (content inlined per O2/#657).
Nominates notes for the agent's covered/near/absent coverage decision via **two channels**:
(1) **within-cluster top-5** by centroid cosine from the cluster's own matched note members;
(2) **tag-nominated notes** — notes sharing ≥1 vocab term with the top-3 delivered notes join the pool
regardless of cluster membership (budget fields `tag_nominations_added`/`dropped` track the count; pool cap
40/cluster). A **superseded-note ride-along** inserts the superseder at the next rank when a delivered note
has a superseder. A cluster with no members and no nominations has an empty `candidate_l2s`. Full-vault
nomination was dropped in recall-v2 (DECISION-2, reversing D7) because it surfaced unrelated notes the agent
had not matched; tag-nomination restores targeted cross-cluster reach on the shared-vocabulary axis.

### cluster (query-time)
A partition of the matched set produced by AutoK k-means (k=2..7 chosen
by max silhouette score). Deterministic per query against an unchanged
vault. Each cluster has a `representative` (member closest to centroid),
`members` (path + score + is_representative), and `candidate_l2s`
(top-5 vault notes from within-cluster note members, by centroid cosine).
Matched sets smaller than 6 items skip clustering and return `clusters: []`.

### provenances (item roles)
A query item's `provenances` list names every role it fills: `direct`
(top-k cosine hit), `cluster_rep` (cluster representative), `recent`
(recency-channel chunk, un-clustered). Items dedup across roles; a path
appears once regardless of how many roles it fills.

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
list targets, surface this note?"* Per-candidate, not
session-global — current-locus candidates target this session's work,
retro-locus candidates target the injecting agent's work. If no, rephrase.
If still no, drop.

### injection locus
The work that *caused* a lesson, distinct from the work that *surfaced*
it. **Current-locus** = the mistake or discovery originated in this
session. **Retro-locus** = the cause is in a prior session, even though
the candidate may have surfaced through current-session work (or come
from prior-session chunk history surfaced by recall). Discriminated cheaply by `git blame` / `git log` on the offending line, prior-session transcript
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

### `--supersedes`
Typed supersession flag on `engram learn` and `engram amend`. Format:
`--supersedes "<basename>|<type>|<claim>"` where `<type>` ∈ `updates|narrows|refutes`.
Writes a `supersedes:` frontmatter list on the noting note; the binary maintains the derived inverse
for O(1) ride-along at query; also renders a `Supersedes: [[<note>]] — <type>: <claim>` body line for
Obsidian visibility. Repeatable. The `--relation` flag (which added `Related to:` bullets) was
**removed 2026-07-03** — use `--supersedes` only when a note corrects, narrows, or refutes another;
structural linking is done automatically by the binary's vocab-tag assigner.

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
frontmatter `project:` does not match on the bounded matched set.
Items with elided content (no body in the payload) are dropped under a
non-empty `--project` since a match cannot be verified.

### `--issue`
Optional identifier for the originating GitHub / Jira / etc. issue. Set
on write via `engram learn {fact,feedback} --issue <id>` and
rendered as `issue: "<id>"` (quoted to survive YAML's numeric coercion
on read-back) in frontmatter below `project:`. Free-form non-whitespace
string — `636`, `#636`, `GH-636`, `PROJ-1234` are all valid. Recorded
for provenance; no read-side filter.

### `engram learn qa` (subcommand)
Writes a QA pair: one `qa.<date>.<slug>.q.md` (question note) and one
`qa.<date>.<slug>.a.md` (answer note) atomically-ish. Requires `--slug`,
`--source`, `--question`, and exactly one of `--answer` / `--answer-file`.
Optional: `--contributors` (repeatable, validated against vault),
`--certainty high|medium|low` (default `medium`). Embed-on-write runs for
both notes; auto-vocab assignment runs on the A-note only. Q-notes carry
no `vocab:` key (per D5′, below — Q-note wording loses retrieval against
content; excluded from the main query set at all four query-pipeline seam
points: the pre-clustering filter, the matched-set floor/cap, the
tag-nomination gate, and the TermIndex builder). On A-write failure
the Q-note is removed (best-effort) and a descriptive error is returned.

### qa-question (note type)

Short form in running prose: **Q-note**.
The question half of a QA pair, stored at `qa.<date>.<slug>.q.md` with
frontmatter `type: qa-question`. Excluded from the main query set at all
four seam points (`isQueryExcludedKind`, same exclusion as `vocab` and
`vocab-index`). Q-notes carry no `vocab:` key and are skipped by the
write-time vocab assigner and the `engram vocab stats` note counter. The
Q-note body contains the verbatim question text and a machine-written
`Answered by: [[qa.<date>.<slug>.a]]` line (excluded from `BodyText`/
`ContentHash` like `Vocab:` / `Supersedes:`). The same reference is also
written as `answered_by: <a-basename>` in frontmatter — the programmatic
traversal handle for the Q→A edge. Reachable via a dedicated
q-space channel (round 3, gated on Arm V PASS + round-2 validation).

### qa-answer (note type)

Short form in running prose: **A-note**.
The answer half of a QA pair, stored at `qa.<date>.<slug>.a.md` with
frontmatter `type: qa-answer`. Competes in the main query set as a
synthesis note (D5′ asymmetric participation) — a relevant past answer
surfaces directly alongside fact/feedback notes. Carries auto-assigned
`vocab:` tags and a machine-written `Answers: [[qa.<date>.<slug>.q]]`
body line. If contributors were supplied, also carries a `Contributors:`
body line (see below). Both machine lines are excluded from
`BodyText`/`ContentHash` (same pattern as `Vocab:` / `Supersedes:`).
Frontmatter also carries `answers: <q-basename>` (the A→Q traversal
handle, inverse of `answered_by`) and `certainty: high|medium|low` (frontmatter-only —
no body-line counterpart; written from the `--certainty` flag, default
`medium`).

### D5′ (design decision — asymmetric QA participation)

Settled by Joe 2026-07-03 (superseding the original full-exclusion D5):
qa-answer notes COMPETE in the main matched set (they are synthesis notes —
pre-reasoned conclusions with provenance); qa-question notes are EXCLUDED
from the main set (question wording measurably loses retrieval against
content — the qanchor finding) and become reachable via a dedicated q-space
channel with an `answered_by` ride-along (round 3, gated). Decision record:
`docs/design/2026-07-03-qa-memory-proposals.md`.

### contributors (QA frontmatter + body field)
A frontmatter list (`contributors: [<basename>, ...]`) and a matching
body line (`Contributors: [[<basename>]], ...`) on `qa-answer` notes.
Lists full note basenames (no `.md`) cited in the answer text. Written by
the binary at capture time from `--contributors` flags (validated against
vault). Machine-written and excluded from `BodyText`/`ContentHash` so a
contributors-only update leaves the embedding and content hash unchanged.
Powers `vaultgraph.InDegreeIn` usage counting as a graded signal that
scales with accumulated Q&A capture.

---

## Transcript

### transcript
The recorded content of one session, read by the binary (via `engram
ingest`) from a harness's on-disk store. Engram reads Claude Code
transcripts — JSONL files at `~/.claude/projects/<slug>/*.jsonl`. A
*session* is the time-bound interaction; a *transcript* is its serialized
record. (The standalone `engram transcript` subcommand and its per-harness
progress marker were retired with the lazy-L2 work — issue 649;
`internal/transcript` is retained as the JSONL reader for `engram
ingest`.)

### session
One conversation between a user and an agent in a harness. Plural:
*sessions*. Sessions produce transcripts; the binary reads transcripts.

### `engram ingest` (subcommand)
Merge-appends session transcripts + markdown into the per-source chunk
index — re-chunks/re-embeds only changed content, never deletes
(append-only chunk history). `--auto` sweeps all known sources, skips
session-log directories whose slugified project path starts with a
**non-persistent-workspace prefix** (`-private-tmp-`, `-tmp-`,
`-var-folders-`, `-private-var-folders-`), and is called by `/learn` and
`/recall`. The skip keeps eval/test runs from bloating the main chunk index;
configure it via `.engram/sweep.json` (`non_persistent_prefixes` key), or
bypass it with explicit `--sweep`/`--transcript`/`--markdown` or an isolated
index via `ENGRAM_CHUNKS_DIR`.
Chunks are the episodic layer (raw event memory); at recall they compete with
notes in the per-phrase ranking (matched set, Channel 1) and the newest
(default 25, configurable via `--recent-fill` / `ENGRAM_RECENT_FILL`) also appear un-clustered in the recency channel (Channel 2). Chunk-grounding
is recorded as frontmatter provenance on written notes, not as wikilinks.

### `engram show-chunk` (subcommand)
Read-only lookup that prints a single chunk's text by its `source#anchor` id (the
id format carried by chunk items in the query payload). Mirrors `engram show`
(which resolves vault **notes** only) for the chunk index, enabling on-demand fetch
of deferred chunk content under `--lazy-chunks`. Matches by full-id equality, so
heading anchors containing spaces/punctuation resolve. Errors `chunk not found:
<ref>` (exit 1) on a miss. Never writes.

### `engram prune` (subcommand)
Operator-run GC subcommand. Reads the chunk-index manifest and, for every
source whose file no longer exists, deletes that source's per-source index
file and drops its manifest entry. Not part of the recall/learn/please flows —
run manually to reclaim space after removing or moving ingested source files.

### non-persistent workspace
A project directory located under a throwaway filesystem root
(`/private/tmp`, `/tmp`, or macOS `$TMPDIR` at `/var/folders` and its
`/private/var/folders` canonical form). `engram ingest --auto` identifies
and skips these by checking whether the slugified project-directory name
starts with one of the configured `non_persistent_prefixes`
(`-private-tmp-`, `-tmp-`, `-var-folders-`, `-private-var-folders-`),
preventing eval/test sessions from bloating the main chunk index.
Explicit sweep roots (`--sweep`, `--transcript`, `--markdown`) and an
isolated index (`ENGRAM_CHUNKS_DIR`) bypass the exclusion for deliberate
test ingestion.

---

## CLI conventions

### subcommand
A named operation on the binary: `learn`, `query`, `ingest`, `update`,
etc. The whole CLI is a single binary with subcommands, never a sprawl of
separate executables.

### `engram update`
Installs/refreshes skills and commands into every detected harness, and
reinstalls the binary via `go install`. `--dry-run` shows the diff
without writing. `--with-guidance` additionally deploys the recall-firing
guidance file to `~/.claude/engram/recall.md` (Claude Code only; opt-in).
OpenCode is deferred — its `AGENTS.md` import support is unverified. Plain
`engram update` hints about `--with-guidance` until the guidance is imported,
then keeps it refreshed on every run.

### guidance file
`guidance/recall.md` in the engram repo — engram's always-loaded
recall-firing guidance, deployed to `~/.claude/engram/recall.md` by
`engram update --with-guidance`. Activated by adding
`@~/.claude/engram/recall.md` to `~/.claude/CLAUDE.md` (Claude Code
`@import`; always loaded). `--with-guidance` is a one-time opt-in — once
CLAUDE.md imports the file, plain `engram update` keeps it current (like
skills). Claude Code only; OpenCode deferred.

### XDG paths
Engram follows XDG basedir conventions:
- Data: `$XDG_DATA_HOME/engram/` (vault at `vault/`, chunk index at `chunks/`).
Fallback: `~/.local/share/...`.

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
polluting parent context.

### coordinator
A serial pass after parallel writer subagents finish, whose job is
cross-document references the parallel writers can't see.

### contradiction
Two surfaced notes making incompatible claims about the same thing. The
vault preserves contradictions; recall surfaces both; the noting note passes
`--supersedes "<basename>|refutes|<claim>"` on the `engram learn`/`amend` call
(typed supersession — no body ritual). Never smoothed.

---

## Status / disposition values

| Term | Meaning |
|------|---------|
| `top` | Luhmann position for a brand-new top-level note |
| `continuation` | Luhmann position extending an existing note (e.g., `1` → `1a`) |
| `sibling` | Luhmann position at the same level (e.g., `1a` → `1b`) |
| `--dry-run` | Flag on `engram update` that previews without writing |
