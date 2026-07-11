# Engram Glossary

Standardized vocabulary for the engram project. Where a term has variants in
the wild, the **canonical form** is named here; variants are listed for
recognition. Inconsistencies that need a decision are tracked in this file's
trailing [Open Questions](#open-questions) section.

## Top-level concepts

### engram
The project, the CLI binary, and the broader system of "skills + binary +
vault" that gives LLM agents persistent memory. When ambiguity matters,
disambiguate as *engram (project)*, *engram (binary)*, or *engram (CLI)*;
in most prose, context already disambiguates and a bare *engram* is fine.

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
Distinct from **slash command** — the user-facing `/name` trigger that invokes
a skill in a harness (Claude Code's term) — and **command** — an OpenCode-specific
file under `commands/` that wraps a skill invocation for that harness.

### atom
The skill-decomposition concept from the ROADMAP charter: one behavior, one skill
(read-memory, write-memory, route-a-task, orchestrate-a-workflow). Only `write-memory`
was extracted — as a worker invoked at the write seams, not a mid-procedure reference
fetch. The full design history (the superseded reference-card form, the decision to stop
at the write seam) is recorded in `docs/architecture/adr.md` ADR-0015, with its validation
in `dev/eval/LEDGER.md`.

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
here by non-fire arms (see `dev/eval/LEDGER.md#write-memory-worker-fire-rates`,
which records the zero-spurious-fire result). Related measured
anti-pattern: pointer-style "apply X verbatim" references to out-of-context
text (e.g. "see the postscript for details") are unreliable — the referenced
content may not be read in the target's actual deployed context; prefer
inlining the content a skill needs directly in its own prose.

### reversal (capture kind)
A conclusion, design, or verdict that was presented (to the user, a review gate,
or a committed plan) and later overturned — by the agent itself, a reviewer, or
an instrument. The learn skill's third capture kind (Step 2, shipped 2026-07-04):
each reversal crystallizes the ROOT CAUSE of the original error via a write-memory
handoff. Self-discovered reversals qualify; a repo-doc CORRECTION section is not
capture.

### confirmed approach (capture kind)
A specific, generalizable approach validated as good — the positive mirror of the
correction and reversal kinds. The learn skill's fourth capture kind (Step 2, shipped
2026-07-06 via #668): it fires when the user praises a specific behavior (4a) or when a
genuine guess the agent acted on is confirmed by an observable outcome (4b), and
crystallizes via a write-memory handoff. Generic pleasantries, routine successes, and
unconfirmed guesses do not qualify — the signal is a resolved uncertainty or an explicit
confirmation, never "it worked".

### lessons audit
The please skill's step-7 enumeration of a cycle's mechanical corpus — fired
pre-registered STOPs, gate FAIL verdicts, commits whose messages carry
CORRECTION/supersede/instrument-invalid/redraw markers, and mid-cycle escalations —
each mapped to the vault note that captures its lesson or an explicit
"no lesson: <why>" line. Unmapped items become reversal handoffs; the list appears
in the cycle's closing report. The audit is failure-shaped by design; positive
reinforcement (the confirmed-approach kind) has no mechanical marker and is captured
by the closing `/learn`'s Step-2 scan, not here. Pre-registered upgrade: a wrong "no lesson" mapping
triggers a fresh-context lessons reviewer (an externalized audit; upgrade path in the
ROADMAP atoms-arc status block).

### escalation provenance
The please skill's rule that any measured claim (count, rate, cost, duration) in a
mid-cycle escalation — an AskUserQuestion or STOP report — carries its evidence
pointer (file/command) and a one-line validity statement ("verified how?");
unverified claims ship only as explicitly-labeled hypotheses. Pre-registered
upgrade: a shipped unverified number triggers an enforced escalation gate (a
fresh ground-truth review per measured escalation; upgrade path in the ROADMAP
atoms-arc status block).

### harness
A coding-agent host that runs skills. Engram supports two: **Claude Code**
and **OpenCode**. The plural is *harnesses*. Session transcripts are read by
`internal/transcript` (Claude Code JSONL; consumed by `engram ingest`).

### binary
The compiled `engram` Go program. It handles all I/O (vault read/write, chunk indexing,
file locking); skills handle behavior and prompting. The full subcommand and flag
reference lives in `README.md` (Binary commands).

---

## Vault structure

### Permanent (note)
An atomic, principle-stated note — *one coherent topic with its full
load-bearing detail and complete sets*, not one micro-fact (over-fragmenting a
topic across notes harms retrieval; see `skills/learn/SKILL.md`). Notes originally
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
shorter `[[<slug>]]` form. Three roles historically; two live today: (1) **prose links** — running-text references
with human-readable context for the connection (no per-link rationale required by `--relation` —
that flag was removed 2026-07-03); (2) **`Supersedes:` body line** —
`Supersedes: [[<note>]] — <type>: <claim>`, written by the binary when `--supersedes` is passed.
(3) **`Vocab:` member→term links** (retired 2026-07-10, #678) — was a `Vocab: [[vocab.<term>]], ...` body line + `vocab:` frontmatter list written by the binary's write-time vocab assigner, and has been migrated to the tags convention: vocab membership is now a `vocab/<term>` entry in the shared `tags:` list, never a wikilink. See **vocab definition note**, below.
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

### vocab definition note
The current representation of a vocab term (shipped 2026-07-10, #678, superseding the vocab term-note,
vocab-index, and `Vocab:` line below): an ordinary bare-`vocab`-tagged fact note (`tags: [vocab]`,
dash-slug naming). Two shapes: a **per-term definition** (`vocab-<term>-definition`, e.g.
`vocab-retrieval-design-definition`) whose `subject`/`predicate`/`object` state the term's meaning and
whose body carries the refit-maintained exemplar list (the body IS the term's embedding text — a
`.vec.json` sidecar is written on embed like any fact note); and one **family note**
(`vocab-definition`) carrying `vocab_version` in frontmatter, whose object documents the tagging
convention WITHOUT enumerating terms (an enumerated term list is the stale-index problem the migration
retired). Minted by `engram vocab bootstrap`/`propose`/`refit`. Unlike the retired vocab
term-note, a definition note is an ordinary recallable fact — no query exclusion (the vocab-kind
exclusion was deleted with the migration). A member note's own vocab terms live in its `tags:` list as
`vocab/<term>` entries (see `--tag`, below); a definition note never carries its own term tag. A member note with no qualifying term carries no `vocab/` tag — absence = untagged, counted by `engram vocab stats`. Plural:
**vocab definition notes**.

### vocab nomination
The tag-match extension to `candidate_l2s` (shipped 2026-07-03): notes sharing ≥1 vocab term with the
top-3 delivered notes are **nominated** into each cluster's candidate pool regardless of their cosine to
the cluster centroid. Budget fields `tag_nominations_added` / `dropped` in the query payload report the
count (pool cap 40/cluster). Zero collateral: notes with no shared vocab term are unaffected. A nominated
note may cross cluster boundaries. Mechanism unchanged by the 2026-07-10 tags migration (#678) — only the
term-membership representation moved, from `vocab:` frontmatter to `tags: [vocab/<term>]`.

*The (retired)-marked entries below are the retired pre-#678 vocab representations (migrated to the tags convention 2026-07-10). Current form: **vocab definition note**, above.*

### vocab term-note (retired)
Was a `vocab.<term>.md` file (no Luhmann number) with frontmatter `type: vocab`; migrated to a per-term bare-`vocab`-tagged fact note.

### vocab-index (retired)
Was `vocab.index.md`, a binary-generated Map of Content listing every term; superseded by the emergent index (`engram count --group-by type --filter tags=vocab`) — no maintained enumeration file.

### `Vocab:` line (retired)
Was a body line on a member note (`Vocab: [[vocab.<term>]], [[vocab.<term2>]], ...`) plus a parallel `vocab: [<term>, <term2>]` frontmatter key, both written by the binary's write-time vocab assigner. Migrated to a `vocab/<term>` entry per term in the shared `tags:` list (see `--tag`, below).

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
Step 0.5 = sweep (`engram ingest --auto`); Step 1 = generate phrases (10 in
`deep` mode, ~3 in `glance`); Step 2 = run one unified `engram query`; Step 2.5
= per-cluster synthesis — 2.5A reads candidates, 2.5B applies the recency
weight, 2.5C judges coverage and writes (amend/write-memory — **skipped in
`glance`**); Step 2.7 = `engram activate` on the notes actually drawn on;
Step 3 = closing synthesis (how memories changed the plan); Step 3.5 = re-entry
query — a recommendation not in the Step-0 plan (conceived mid-work) gets its own
lever-keyed `engram query` before it ships, with a forced `Re-entry:` line
directly above the final recommendation (#655); Step 4 = persist a sound synthesis
conclusion via write-memory (**skipped in `glance`**). See
**recall modes**, above, for the glance/deep split.

### honored-when-fired
C7 GREEN-validation metric: of the trials where the Step 3.5 re-entry query fired and returned the
closure note, the fraction whose recommendation then honored it (acknowledged the prior attempt and
dropped the lever, or justified revisiting on named new evidence — i.e. scored RECONCILED).
Separates "the query ran" (fire-rate) from "the result was applied." Measured 24/24 across the #655
v2+v3 batches (`dev/eval/LEDGER.md#c7-reentry-query-green`).

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

### matched-note floor
The per-phrase reservation (`noteFloorK`=5, implemented by `capWithNoteFloor` in
`internal/cli/query.go`) that guarantees up to 5 relevance-qualified notes
(baseScore ≥ the relevance floor, below) survive each phrase's top-30
(`matchPhraseLimit`) truncation, evicting the lowest-scoring chunks to make
room. Exists because chunks vastly outnumber notes in the vault: without the
reservation, a crystallized lesson the embedder itself ranks top among notes
could still be evicted from the per-phrase ranking before ever reaching the
union step across phrases. Only ever promotes notes that already clear the
relevance floor, so it never surfaces an irrelevant note — it changes which
items survive the per-phrase cut, not their relative order. Measured effect:
`dev/eval/LEDGER.md#matched-note-floor`.

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
The binary subcommand. Three forms: `engram learn feedback`, `engram learn
fact` (both require `--source` and take body content via flags; stdin is
ignored), and `engram learn qa` (see below). The `moc` subcommand was retired
after the F4 migration (the 25 historical MOCs are archived for audit in
`<vault>/_legacy/MOCs/`); the `episode` form was retired with the lazy-L2
work (issue 649) — chunks in the chunk index are the episodic layer now.

### Feedback (note type)
A note recording something to do differently next time — user corrections,
dead-ends, failed approaches. Auto-generated opener: `Lesson learned: …`. The
opener wording is intentionally distinct from the type name (not a parallel
"Feedback noted: …") — it names the *nature* of the note (something learned
the hard way), not the frontmatter `type:` value.

### Fact (note type)
A note recording how something actually works — tool behaviors, idioms,
conventions, gotchas. Auto-generated opener: `Information learned: …`. Same
non-parallel-by-design relationship to the type name as Feedback, above.

### Step 2 (learn's crystallization gate)
The gate that decides whether `/learn` writes anything: a scan of the session
for exactly four capture kinds — **corrections** (the user corrected an
approach or behavior), **explicit save-requests** ("remember this/that X"),
**reversals** (see above), and **confirmed approaches** (see above — positive
reinforcement, user-praised or self-validated) — each handed to **write-memory**
(kind=`feedback` or `fact` per the kind). No qualifying moment → learn writes nothing (a
two-command sweep-only run). Replaces the retired recall-mirror-test /
injection-locus / scratch-list / Path-A-B-C apparatus, which was never
implemented in the shipped skill.

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
cross-note connections along a shared topic ride the binary's automatic vocab-tag assignment
(`tags: [vocab/<term>]`), nominated at query time — not authored wikilinks.

### `--tag`
Repeatable categorical-tag flag on `engram learn fact` / `engram learn feedback` (not qa, not
amend — amend nonetheless round-trips an existing `tags:` list unchanged). Each value must match
`[a-z0-9-]+` (bare family) or `[a-z0-9-]+/[a-z0-9-]+` (family/value); anything else is rejected
before any write. Writes the frontmatter `tags:` string list — the sole categorical
representation (ADR-0019). The binary also auto-assigns vocab terms into the same list under the
`vocab/` namespace (see **vocab definition note**) — hand-authored `--tag` values and the binary's
`vocab/<term>` entries share the one list but are written by different actors; never hand-author a
`vocab/` tag yourself.

### `--source`
Required provenance field on every `engram learn` invocation. Format:
`session log <project>, <YYYY-MM-DD HH:MM UTC>, context: <short
description>` for session-derived notes.

### `--project` (write side)
Optional kebab-case slug naming the project a note belongs to. Set on
write via `engram learn {fact,feedback} --project <slug>` and
rendered as `project: <slug>` in frontmatter below `source:`. Absent on
notes that capture universal principles. Project name still does not
belong in `--situation` — `--situation` stays retrieval-shaped (phrased the
way a future task would query for it); `--project` is the metadata home for
projectness so cross-project queries become answerable without polluting
the situation phrase.

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
no `vocab/` tag (per D5′, below — Q-note wording loses retrieval against
content; excluded from the main query set at all four query-pipeline seam
points: the pre-clustering filter, the matched-set floor/cap, the
tag-nomination gate, and the TermIndex builder). On A-write failure
the Q-note is removed (best-effort) and a descriptive error is returned.

### qa-question (note type)

Short form in running prose: **Q-note**.
The question half of a QA pair, stored at `qa.<date>.<slug>.q.md` with
frontmatter `type: qa-question`. Excluded from the main query set at all
four seam points (`isQueryExcludedKind`) — the sole remaining exclusion since 2026-07-10 (#678
retired the vocab-kind exclusion; vocab definition notes are now ordinary recallable facts). Q-notes
carry no `vocab/` tag and are skipped by the
write-time vocab assigner and the `engram vocab stats` note counter. The
Q-note body contains the verbatim question text and a machine-written
`Answered by: [[qa.<date>.<slug>.a]]` line (excluded from `BodyText`/
`ContentHash`, the same treatment as other machine-written body lines like `Supersedes:`). The same reference is also
written as `answered_by: <a-basename>` in frontmatter — the programmatic
traversal handle for the Q→A edge. Reachable via a dedicated
q-space channel (round 3, gated on Arm V PASS + round-2 validation).

### qa-answer (note type)

Short form in running prose: **A-note**.
The answer half of a QA pair, stored at `qa.<date>.<slug>.a.md` with
frontmatter `type: qa-answer`. Competes in the main query set as a
synthesis note (D5′ asymmetric participation) — a relevant past answer
surfaces directly alongside fact/feedback notes. Carries auto-assigned
`vocab/` tags and a machine-written `Answers: [[qa.<date>.<slug>.q]]`
body line. If contributors were supplied, also carries a `Contributors:`
body line (see below). Both machine lines are excluded from
`BodyText`/`ContentHash` (same pattern as `Supersedes:`).
Frontmatter also carries `answers: <q-basename>` (the A→Q traversal
handle, inverse of `answered_by`) and `certainty: high|medium|low` (frontmatter-only —
no body-line counterpart; written from the `--certainty` flag, default
`medium`).

### D5′ (design decision — asymmetric QA participation)

The rule governing which QA-pair halves compete in recall's main matched set:
qa-answer notes COMPETE (they are synthesis notes — pre-reasoned conclusions
with provenance); qa-question notes are EXCLUDED (reachable instead via a
dedicated q-space channel with an `answered_by` ride-along, round 3, gated).
Supersedes the original full-exclusion D5. Decision record and rationale:
`docs/architecture/adr.md` ADR-0012.

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

## Route evidence

### evidence note (route)
An ordinary fact note recording one route dispatch, written via write-memory when the dispatch
resolves. Carries frontmatter `tags: [work-kind/<k>, tier/<t>, outcome/<o>]` — the three
low-cardinality categoricals; duration/cost live in the object prose with explicit units, never
in tags. Fully recallable (no query exclusion, no new note type) — the structured replacement for
route's old free-text transcript record. Slug convention: `route-dispatch-<work-kind>`.

### aggregate note (route)
One fact note per work-kind, slug `route-evidence-<work-kind>`, whose object text states the
current tier tallies ("cheap 14/16, mid 2/2 as of <date>") and wikilinks every evidence note it
summarizes (append-only trail). Amended (`engram amend --object`) as each dispatch lands; created
untagged. Route READS it via plain recall — it surfaces as a normal memory; `engram count` over
the evidence notes' tags is the audit that verifies/repairs its tallies (ADR-0019).

### family definition note / bare-tag convention
A fact note documenting a tag family — its meaning, allowed values, and counting pattern —
carrying the BARE family tag (e.g. `tags: [work-kind]`). Convention: a bare family tag marks the
family's definition note; a nested `family/value` tag marks a member. Three ship with #674:
`work-kind` (open kebab-case set), `tier` (cheap|mid|deep), `outcome` (pass|fail) — slugs
`work-kind-definition`, `tier-definition`, `outcome-definition`. Vault data, not repo files.

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
Operator-run maintenance subcommand. Reads the chunk-index manifest and, for
every source whose file no longer exists, drops the stale manifest entry but
KEEPS that source's per-source index file — the embedded chunks stay searchable
(chunk search scans index files directly, not the manifest). Not part of the
recall/learn/please flows — run manually after removing or moving ingested
source files to clear dead manifest entries without losing the embedded memory.

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
without writing. `--with-guidance` additionally deploys the guidance docs
(`recall.md`, `delegate.md`) to `~/.claude/engram/` (Claude Code only;
opt-in). OpenCode is deferred — its `AGENTS.md` import support is
unverified. Plain `engram update` hints about `--with-guidance` until a
guidance file is imported, then keeps it refreshed on every run.

### `engram count` (subcommand)
Read-only aggregation over the vault, deliberately off the query/similarity path
(`internal/cli/count.go`). Two mutually exclusive modes: `--group-by <attr>` counts DISTINCT note
membership per frontmatter-attribute value (a list attr contributes one per distinct element; a
note listing a value twice counts once), optionally restricted by repeatable AND-ed
`--filter attr=value` predicates (scalar equality or list-contains) — output is `value<TAB>count`
lines sorted count-desc then value-asc, an `(attr absent): N` bucket when any in-set note lacks the
attribute, then `total: N` (an empty result prints nothing). `--backlinks-of <basename>` prints the
wikilink in-degree of a vault-graph node (ADR-0007) plus its sorted linkers. Each mode is
independently **Obsidian-verifiable** against its own view — `--group-by` against a
frontmatter/property/tag filter (or Dataview), `--backlinks-of` against the note's backlinks panel
— but the two are **not** equal to each other: `--backlinks-of` counts every linker while
`--group-by` counts only frontmatter members, so they diverge by the number of *non-member
linkers* (e.g. a hand-authored MOC/hub page that links every note on a topic without itself carrying
that topic in frontmatter; `vocab.index.md` was a machine-generated instance of this pattern, retired
2026-07-10 under #678). See ADR-0018. Since #674 it is also the audit surface for route's dispatch-evidence tallies —
`--group-by tags --filter tags=...` recomputes ground truth from evidence-note tags (see
"aggregate note (route)" and ADR-0019); audit only, never the routing read path.

### guidance file
An always-loaded ambient doc under `guidance/` in the engram repo, deployed
to `~/.claude/engram/<name>.md` by `engram update --with-guidance`. Two ship
today: `recall.md` (recall-firing) and `delegate.md` (delegation-firing).
Each is activated independently by adding its own
`@~/.claude/engram/<name>.md` line to `~/.claude/CLAUDE.md` (Claude Code
`@import`; always loaded). `--with-guidance` is a one-time opt-in per file —
once CLAUDE.md imports a guidance file, plain `engram update` keeps it
current (like skills). Claude Code only; OpenCode deferred.

### XDG paths
Engram follows XDG basedir conventions:
- Data: `$XDG_DATA_HOME/engram/` (vault at `vault/`, chunk index at `chunks/`).
Fallback: `~/.local/share/...`.

### DI (dependency injection)
Architectural rule: no function in `internal/` calls `os.*`, `http.*`, or
any I/O directly. All I/O goes through injected interfaces, wired at the
CLI edges. Tests use `imptest`-generated mocks.

### targ
The build tool wrapping `go test`/`go vet` for testing, linting, and coverage
checks — and also the CLI framework the `engram` binary itself is built on
(`internal/cli/targets.go` wires its subcommands). **Always** invoke
`targ test`, `targ check-full` — never the underlying Go commands directly.
targ has no `build` target: it covers test/lint/check, not binary install.
Install or refresh the compiled binary with `go install ./cmd/engram`.

---

## Authoring & process vocabulary

### candidate (note)
A potential note identified from completed work (transcript content
surfaced by recall, or the current session's activity), before it clears
learn's Step 2 gate or recall's Step 2.5C coverage judgment. Becomes a
written note or is dropped with a reason.

### subagent
A fresh-context worker spawned via the Agent tool to do isolated object-level
or review work without polluting the parent's context. Current uses:
**please**'s gate reviewers (a fresh subagent per gate — plan, refactors,
docs, outward prose) and any executor/planner a skill fans out to; agent
type, model, and effort selection for a dispatch follow the **route** skill's
doctrine. The retired parallel-writer architecture — subagents synthesizing
notes in parallel, reconciled afterward by a serial coordinator pass — is
gone: recall's Step 2.5 crystallizes inline from the query payload's own
clusters, and learn's Step 2 hands off inline to write-memory.

### contradiction
Two surfaced notes making incompatible claims about the same thing. The
vault preserves contradictions; recall surfaces both; the noting note passes
`--supersedes "<basename>|refutes|<claim>"` on the `engram learn`/`amend` call
(typed supersession — no body ritual). Never smoothed.

### capability axes (C1–C7)
The memory-value claims the eval suite measures, referenced by code across ROADMAP and
`dev/eval/LEDGER.md`: **C1** faster · **C2** cheaper · **C3** apply-conventions (fewer human
re-statements) · **C4** learn-from-changes-over-time (**C4i** = the recency-supersession
sub-variant) · **C5** remember-recent-history · **C6** compounding / emergent synthesis (A+B→C).
**C7** is not a capability but the lever-recheck regression gate (does a shipped lever stay shipped).

### capture guards (G1–G6)
The six proposed guards against the lesson-capture blind spot (a presented conclusion later
overturned going uncaptured). Shipped 2026-07-04: **G1** reversals as a learn capture kind ·
**G2** please step-7 lessons audit · **G6** escalation-provenance rule. Pre-registered upgrade
paths (in ROADMAP's atoms-arc block): **G3** fresh-context lessons reviewer (upgrade of G2) ·
**G5** enforced escalation gating (upgrade of G6); **G4** crystallize-on-discovery stays parked.

---

## Status / disposition values

| Term | Meaning |
|------|---------|
| `top` | Luhmann position for a brand-new top-level note |
| `continuation` | Luhmann position extending an existing note (e.g., `1` → `1a`) |
| `sibling` | Luhmann position at the same level (e.g., `1a` → `1b`) |
| `--dry-run` | Flag on `engram update` that previews without writing |

---

## Open Questions

None currently — `triage.md`'s items were all resolved 2026-07-05 (folded into
the entries above); `docs/triage.md` was deleted 2026-07, git log recovers it.
