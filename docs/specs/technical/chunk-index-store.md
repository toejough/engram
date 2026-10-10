# Chunk Index Store - Technical Spec

## 792b18a Orientation

- Ownership: Repository component (caller of [ingestion-source-discovery](./ingestion-source-discovery.md), [transcript-reader](./transcript-reader.md),
  [source-chunking](./source-chunking.md), [text-embedder](./text-embedder.md), [durable-file-store](./durable-file-store.md) and [vault-write-lock](./vault-write-lock.md)).
- Responsibility: keeps the chunk index in its directory: one index file of embedded chunk records
  per indexed source, and a manifest recording every source it knows with its staleness signature
  and, for a copy, the source it copies. Under the manifest lock it reconciles the index with the
  gathered sources: it skips unchanged sources, rebuilds changed ones by appending only new chunks,
  indexes one source per group of identical copies and evicts a copy only when nothing would be lost,
  and drops stray vault-note copies. It detaches vanished sources, removes empty index files and
  collapses old duplicate groups on `engram prune`, and lists index files and chunk IDs for its
  readers.
- Current realization (informational): `RunIngest` and its helpers in `internal/cli/ingest.go`
  (`rebuildIndex`, `mergeChunkRecords`, `readManifest`, `writeManifestFile`, `manifestLockFrom`,
  `buildChunkIDSet`, `sourceSlug`); the dedup pass in `internal/cli/ingest_dedup.go`
  (`buildHashedSources`, `selectCanonical`, `reconcileGroupCanonicalFirst`, `reconcileDuplicate`,
  `duplicateRecordsCoveredBy`, `removeDuplicateIndex`, `dropVaultCopiesOutsideVault`);
  `RunPrune` in `internal/cli/prune.go`; `pruneDuplicatesLocked` in
  `internal/cli/prune_duplicates.go`; `listJSONLIndexes` in `internal/cli/query_chunks.go`; the
  record format in `internal/chunk/index.go`. The chunk index lives in the chunks directory
  ([792b2c6 Chunk index directory resolution](./command-line-surface.md#792b2c6-chunk-index-directory-resolution)), part of the binary's operational state rather than a store of its own.
- Qualification: responsibility, maintaining an index of embedded chunks is stated without naming
  any function; ownership, it owns the index's files and their consistency, while discovery, reading,
  chunking, embedding, writing and locking belong to the components it calls; independent
  verification, `RunIngest` and `RunPrune` are tested directly over in-memory filesystems, fake
  embedders and recording locks, and over the real filesystem; stability, the contract survives any
  change to how sources are found or chunked; necessity, every chunk a query returns comes from it.
- Supported capabilities: [chunk-indexing](../product/chunk-indexing.md), [note-revision](../product/note-revision.md) (chunk IDs), [memory-query](../product/memory-query.md)
  (the first-indexed time), [install-and-update](../product/install-and-update.md) (the backlog check), [learn](../product/learn.md) (the chunk index written at learn time).

Rationale: the index is rebuilt incrementally and offline, with no LLM. A source's index file is append-only: an
embedded chunk is the memory and its source only provenance, so editing or deleting a source never loses what was
learned from it. Across sources that guarantee is narrowed, not reversed: a copy's index file may be removed, but only
when every one of its chunks is provably in a retained twin's index, so no chunk ever becomes unsearchable. Separate
locks protect the chunk index (`.manifest.lock`) and the vault (`.luhmann.lock`), because ingest and prune never touch
notes.

## 792b18c Behavior Requirements

### 792b18c1 Index layout

Requirement: [792b18c1 Index layout](#792b18c1-index-layout) The chunk-index-store MUST keep each indexed source's chunks in the file
`<source base name without extension>-<first 8 hex digits of the SHA-256 of the source path>.jsonl`
in the chunks directory, one JSON record per line holding `source`, `anchor`, `content_hash`
(`sha256:` and the hex SHA-256 of the text), `text`, `vector`, and `ingested_at` when set, and MUST
keep `manifest.json` there mapping each known source path to its `mtime_unix_nano`, `size`,
`file_hash` (`sha256:` and the hex SHA-256 of its bytes) and, for a recorded copy, `duplicate_of`.
It MUST write both through the durable-file-store's ([durable-file-store](./durable-file-store.md)) atomic replace ([792b3c1 Atomic replace](./durable-file-store.md#792b3c1-atomic-replace)) with mode `0600`,
creating the chunks directory with mode `0700` when absent. Reading an index file MUST skip blank
lines and fail on a line that is not a record.

Supports: [792a5b1a A named transcript and document are indexed](../product/chunk-indexing.md#792a5b1-uat-scenarios), [792a5i1b A changed source embeds only its new text](../product/chunk-indexing.md#792a5i1-uat-scenarios), [792a5p1a A deleted source is detached and stays searchable](../product/chunk-indexing.md#792a5p1-uat-scenarios).

Rationale: the path hash keeps two `README.md` files in different directories from sharing an
index file, and the base name keeps the directory readable.

#### 792b18c1a Scenarios

```gherkin
Scenario: 792b18c1a1 A transcript's chunk is recorded with its vector and hash
  Given a transcript /sessions/abc.jsonl whose text forms one chunk
  When it is ingested
  Then its index file holds one record whose source is /sessions/abc.jsonl, whose vector is not empty, and whose content_hash starts sha256:

Scenario: 792b18c1a2 An index file is written atomically into a created directory
  Given a chunks directory that does not exist
  When the index file /chunks/idx.jsonl is written
  Then /chunks is created with mode 0700 and the file is replaced atomically with mode 0600

Scenario: 792b18c1a3 A source's staleness signature is its modification time and size
  Given a source file whose modification time and size are known
  When its signature is taken
  Then it is that modification time in nanoseconds and that size, and a failed stat is an error
```

### 792b18c2 Locked reconciliation

Requirement: [792b18c2 Locked reconciliation](#792b18c2-locked-reconciliation) The chunk-index-store MUST create the chunks directory and take the
[vault-write-lock](./vault-write-lock.md) on `<chunks>/.manifest.lock` ([792b8c1 Exclusive hold](./vault-write-lock.md#792b8c1-exclusive-hold)) before `engram ingest` or `engram prune`
reads the manifest, MUST hold it through every index-file and manifest write of the run, and MUST
release it before returning. Its helpers, the manifest write among them, MUST NOT take the lock
themselves. A failure to create the directory or take the lock MUST be returned with nothing read.
A missing manifest MUST be read as empty, and a manifest that is not valid JSON MUST be an error.

Supports: [792a5o1a An ingest and a prune at the same time both take effect](../product/chunk-indexing.md#792a5o1-uat-scenarios).

Rationale: two concurrent runs that each read the manifest would let the later write overwrite the earlier one's
change, so a lock serializes manifest updates. Creating the directory first lets a prune of a fresh index take its
lock.

#### 792b18c2a Scenarios

```gherkin
Scenario: 792b18c2a1 Ingest holds the lock across its read-modify-write
  Given a lock that records when it is taken and released
  When an ingest adds a source
  Then the lock is taken before the manifest is read and released after it is written

Scenario: 792b18c2a2 Prune holds the lock across its read-modify-write
  Given a lock that records when it is taken and released
  When a prune detaches a source
  Then the lock is taken before the manifest is read and released after it is written

Scenario: 792b18c2a3 A concurrent ingest and prune both take effect
  Given a manifest recording a source whose file is gone, and the real file lock
  When an ingest of a new source and a prune run at the same time against the same chunks directory
  Then the final manifest records the new source and not the gone one

Scenario: 792b18c2a4 The directory is created before the lock is taken
  Given a chunks directory /chunks
  When the manifest lock is taken and released
  Then /chunks is created with mode 0700, then /chunks/.manifest.lock is locked, then unlocked

Scenario: 792b18c2a5 A directory failure takes no lock
  Given creating the chunks directory fails
  When the manifest lock is taken
  Then the error is returned and no lock is attempted
```

### 792b18c3 Change detection

Requirement: [792b18c3 Change detection](#792b18c3-change-detection) The chunk-index-store MUST NOT read a known source whose modification time
and size equal its manifest entry's and which is a recorded copy or has its index file, and MUST use
the recorded content hash for it. It MUST read and hash every other source. A source that is the
preferred copy of its group, whose recorded hash equals its current hash and whose index file exists,
MUST NOT be rebuilt: when it was read, only its recorded signature is refreshed, and otherwise
nothing about it is written. A recorded source whose index file is missing MUST be rebuilt.

Supports: [792a5i1a An unchanged source costs nothing](../product/chunk-indexing.md#792a5i1-uat-scenarios).

Rationale: an automatic sweep runs before every learn over every known source, so an unchanged
source must cost one stat. The index-file check closes the window between removing a copy's index
file and its manifest entry: a crash there would otherwise leave a source recorded as indexed with
no index, forever.

#### 792b18c3a Scenarios

```gherkin
Scenario: 792b18c3a1 An unchanged source is not re-embedded
  Given a swept document that was ingested and has not changed
  When it is ingested again
  Then nothing is embedded

Scenario: 792b18c3a2 An unchanged known source is not read
  Given a source recorded in the manifest with its index file, whose signature has not changed
  When it is ingested again
  Then the source file is not read

Scenario: 792b18c3a3 Re-ingesting identical content adds nothing
  Given a transcript that was ingested
  When it is ingested again with the same text
  Then its index file is unchanged and holds one record

Scenario: 792b18c3a4 A missing index file is rebuilt
  Given a source recorded in the manifest, unchanged, whose index file is gone
  When it is ingested
  Then its index file is rebuilt
```

### 792b18c4 Read and write failures

Requirement: [792b18c4 Read and write failures](#792b18c4-read-and-write-failures) When a source cannot be read, the chunk-index-store MUST return an error
naming it if the source was named ([792b17c1 Named sources](./ingestion-source-discovery.md#792b17c1-named-sources)), and otherwise MUST print `skip <path>: <error>` on
stdout and go on without it. Every other failure, of the [transcript-reader](./transcript-reader.md), the embedder, or an
index-file or manifest write, MUST be returned as an error.

Supports: [792a5b1d An unreadable named file fails the command](../product/chunk-indexing.md#792a5b1-uat-scenarios), [792a5c1b A swept file that vanishes is skipped](../product/chunk-indexing.md#792a5c1-uat-scenarios).

#### 792b18c4a Scenarios

```gherkin
Scenario: 792b18c4a1 A vanished swept source is skipped and a named one fails
  Given a sweep listing /docs/conv.md and /docs/gone.md, where gone.md cannot be read
  When the sweep is ingested, and then /docs/gone.md is ingested by name
  Then the sweep indexes conv.md and writes the manifest, and the named ingest fails

Scenario: 792b18c4a2 A named document that cannot be read fails
  Given no file at /missing.md
  When /missing.md is ingested by name
  Then the ingest fails

Scenario: 792b18c4a3 A transcript-reader failure fails the ingest
  Given a named or Pi transcript whose reader fails
  When it is ingested
  Then the ingest fails with that error

Scenario: 792b18c4a4 An index-file write failure fails the ingest
  Given index-file writes fail
  When a document is ingested
  Then the ingest fails with that error

Scenario: 792b18c4a5 A manifest write failure fails the ingest
  Given manifest writes fail
  When a swept document is ingested
  Then the ingest fails with that error
```

### 792b18c5 Rebuilding a source's index

Requirement: [792b18c5 Rebuilding a source's index](#792b18c5-rebuilding-a-sources-index) To rebuild a source, the chunk-index-store MUST chunk it ([source-chunking](./source-chunking.md)), a `.jsonl`
source as transcript text from the [transcript-reader](./transcript-reader.md) and any other as Markdown, MUST keep
every record already in its index file, an absent or undecodable file counting as none, MUST embed
([792b9c1 Bundled-model embedding](./text-embedder.md#792b9c1-bundled-model-embedding)) and append each chunk whose content hash is not among them, and MUST write the merged
records, or write nothing when there are none. It MUST record the source's signature and content
hash in the manifest and print `ingest <path>: <records> chunks (<reused> reused, <embedded> embedded)`.

Supports: [792a5b1a A named transcript and document are indexed](../product/chunk-indexing.md#792a5b1-uat-scenarios), [792a5h1a Markdown is chunked by heading](../product/chunk-indexing.md#792a5h1-uat-scenarios), [792a5i1b A changed source embeds only its new text](../product/chunk-indexing.md#792a5i1-uat-scenarios), [792a5j1a Edited-away text stays searchable](../product/chunk-indexing.md#792a5j1-uat-scenarios), [792a5k1a An empty transcript writes no index file](../product/chunk-indexing.md#792a5k1-uat-scenarios).

Rationale: reusing a chunk's vector by its text's hash means editing one section re-embeds only
that section. Keeping every earlier record makes the index append-only within a source ([792a5j A source's chunks are never dropped](../product/chunk-indexing.md#792a5j-a-sources-chunks-are-never-dropped)),
which is also why two byte-identical sources can hold different records ([792b18c7 Recording and evicting copies](#792b18c7-recording-and-evicting-copies)).

#### 792b18c5a Scenarios

```gherkin
Scenario: 792b18c5a1 A new document is chunked and embedded
  Given a swept document with sections A and B
  When it is ingested
  Then its index file holds two records and the embedder ran twice

Scenario: 792b18c5a2 A named document is indexed under its heading
  Given /docs/conventions.md with one "## Conventions" section
  When it is ingested by name
  Then its index holds one record anchored Conventions

Scenario: 792b18c5a3 An edited section re-embeds and the old text stays
  Given an ingested document whose section B is rewritten
  When it is ingested again
  Then its index holds three records, the old B text among them, and the embedder ran once

Scenario: 792b18c5a4 New content is appended to a changed source
  Given an ingested transcript to which a turn is appended, or whose content is replaced
  When it is ingested again
  Then the index holds the earlier chunk and the new one

Scenario: 792b18c5a5 A source with no chunks writes no index file
  Given an empty transcript
  When it is ingested
  Then no index file is written for it, and the manifest is written

Scenario: 792b18c5a6 The production wiring indexes a swept document
  Given a temporary directory holding a Markdown document, and the production filesystem wiring
  When it is swept twice and then searched
  Then the second sweep changes nothing and the search finds the document's text
```

### 792b18c6 Grouping identical content

Requirement: [792b18c6 Grouping identical content](#792b18c6-grouping-identical-content) The chunk-index-store MUST group the gathered sources by content hash and
kind, `.jsonl` sources being transcripts and all others Markdown, and MUST pick each group's
preferred copy, the same whatever order the sources were found in, by origin (explicit, then
repository, then claude-ancestor or pi-ancestor, then session-log or manual-sweep), then the smaller
ancestor depth, then the fewer path separators, then the lexicographically smaller path. It MUST
reconcile the preferred copy before any other member of its group.

Supports: [792a5l1a A repository file outranks its copy in an ancestor .claude directory](../product/chunk-indexing.md#792a5l1-uat-scenarios), [792a5l1b A named file outranks the repository copy](../product/chunk-indexing.md#792a5l1-uat-scenarios), [792a5l1c The nearer ancestor directory wins](../product/chunk-indexing.md#792a5l1-uat-scenarios), [792a5l1d A transcript and a document with the same bytes are both indexed](../product/chunk-indexing.md#792a5l1-uat-scenarios).

Rationale: the precedence is a pure function of the candidates, so the same content always resolves
to the same copy. Grouping by kind as well as hash keeps a transcript and a Markdown document with
the same bytes apart, since their chunks differ. Reconciling the preferred copy first means its
index exists, or is known to be empty, before any copy is judged against it, so a copy found earlier
in the sweep than its preferred twin is not refused for an accident of order.

#### 792b18c6a Scenarios

```gherkin
Scenario Outline: 792b18c6a1 Each precedence rule picks the preferred copy
  Given copies whose origins, depths and paths differ as in <case>
  When the preferred copy is picked
  Then it is <winner>

  Examples:
    | case                                               | winner                  |
    | explicit against repository                        | the explicit copy       |
    | repository against ancestor                        | the repository copy     |
    | ancestor against session-log                       | the ancestor copy       |
    | depth 0 against depth 2 ancestors                  | the depth-0 copy        |
    | /sessions/a.md against /tmp/sweep/a.md, both rank 4 | /sessions/a.md          |
    | equal separators                                   | the lexicographically first |

Scenario: 792b18c6a2 The pick does not depend on order
  Given any set of copies, in any two orders
  When the preferred copy is picked from each
  Then both picks are the same

Scenario: 792b18c6a3 A repository copy is indexed and its ancestor copy recorded
  Given byte-identical /repo/doc.md and /repo/.claude/doc.md found by one sweep
  When they are ingested
  Then only /repo/doc.md has an index file, and /repo/.claude/doc.md is recorded as its copy

Scenario: 792b18c6a4 A transcript and a document with the same bytes are both indexed
  Given a .md and a .jsonl source with identical bytes
  When they are ingested
  Then both have index files

Scenario: 792b18c6a5 A nearer .pi copy beats a farther .claude copy
  Given identical files in a .pi directory at the working directory and a .claude directory one level up
  When they are ingested with --auto
  Then the .pi copy is indexed

Scenario: 792b18c6a6 A preferred copy found later is still reconciled first
  Given a lower-precedence copy found by --sweep before its new preferred twin found by --auto, the lower copy already indexed
  When they are ingested
  Then in this one run the preferred copy is indexed and the lower copy evicted
```

### 792b18c7 Recording and evicting copies

Requirement: [792b18c7 Recording and evicting copies](#792b18c7-recording-and-evicting-copies) The chunk-index-store MUST record each other member of a group as a copy of
the preferred copy, with its signature and hash and no index file, writing nothing when it is
already so recorded. A member indexed by an earlier run MUST be evicted, its index file removed and
then its manifest entry, before it is recorded as a copy, but only when the preferred copy's index
file exists and holds every content hash of the member's records; a member with no index file is
covered, and one whose index cannot be decoded is not. Otherwise its entry MUST be left unchanged and
``ingest: refusing to evict <path>: canonical <path> does not verifiably cover its records (needs review — run `engram prune --duplicates` once resolved)``
printed. The preferred copy's records MUST be read at most once per group. A recorded copy whose
preferred copy is no longer gathered MUST become its group's preferred copy and be read and indexed.

Supports: [792a5m1a A copy indexed first gives way to the preferred one](../product/chunk-indexing.md#792a5m1-uat-scenarios), [792a5m1b A copy with a history of its own is kept](../product/chunk-indexing.md#792a5m1-uat-scenarios), [792a5m1c The remaining copy is indexed when the preferred one goes away](../product/chunk-indexing.md#792a5m1-uat-scenarios).

Rationale: byte identity of current content is necessary but not sufficient. A source's index holds every chunk of its
earlier versions, so two files equal today can hold disjoint histories, and a gate that checked only that the
preferred copy's index existed could remove a file holding content found nowhere else. Removing the index file before
the manifest entry means an interrupted eviction leaves an entry without a file, which change detection repairs
([792b18c3 Change detection](#792b18c3-change-detection)), never a file nothing points to.

#### 792b18c7a Scenarios

```gherkin
Scenario: 792b18c7a1 A copy indexed first is evicted by its preferred twin
  Given /repo/.claude/doc.md was indexed while /repo/doc.md did not exist
  When /repo/doc.md appears with the same bytes and both are ingested
  Then /repo/doc.md is indexed, and /repo/.claude/doc.md's index file is gone and it is recorded as a copy

Scenario: 792b18c7a2 A preferred copy with no chunks cannot cover
  Given an indexed copy with a real history, and a new preferred twin whose content yields no chunks
  When they are ingested
  Then the eviction is refused and the copy keeps its index file and entry

Scenario: 792b18c7a3 A copy with records of its own is kept
  Given an indexed copy holding a record its new preferred twin's fresh index lacks
  When they are ingested
  Then the eviction is refused, its index file kept and its entry not marked as a copy

Scenario Outline: 792b18c7a4 The coverage check
  Given <state>
  When the copy's coverage is checked
  Then it is <verdict>

  Examples:
    | state                                                  | verdict     |
    | the preferred copy has no index file                   | not covered |
    | every record of the copy is in the preferred copy's    | covered     |
    | the copy holds a record the preferred copy lacks       | not covered |
    | the copy has no index file                             | covered     |
    | the preferred copy's index file is empty               | not covered |

Scenario: 792b18c7a5 The preferred copy's index is read once per group
  Given a group with one preferred copy and several indexed copies
  When they are ingested
  Then the preferred copy's index file is read once

Scenario: 792b18c7a6 A copy is promoted when its preferred twin goes away
  Given /repo/doc.md indexed and /repo/.claude/doc.md recorded as its copy
  When only /repo/.claude/doc.md is gathered and ingested
  Then /repo/.claude/doc.md is read and indexed
```

### 792b18c8 Stray vault-note copies

Requirement: [792b18c8 Stray vault-note copies](#792b18c8-stray-vault-note-copies) Before reconciling, the chunk-index-store MUST drop every gathered source that
is not explicit, ends in `.md`, lies outside the resolved vault path, and has a sidecar ([792b10c2 Sidecar path](./embedding-sidecar.md#792b10c2-sidecar-path))
beside it. It MUST keep every source when the vault path is empty.

Supports: [792a5n1a A vault-note copy outside the vault is skipped](../product/chunk-indexing.md#792a5n1-uat-scenarios).

Rationale: a sidecar marks a vault note, so such a file outside the vault is a copy, often a drifted
one that identical-content grouping cannot match to its original. The rule is specific to vault
notes; a drifted copy of any other document is not caught.

#### 792b18c8a Scenarios

```gherkin
Scenario: 792b18c8a1 A swept vault-note copy is dropped and a plain document kept
  Given a swept .md with a sidecar outside the vault, and a swept .md without one
  When they are ingested
  Then only the one without a sidecar is indexed

Scenario Outline: 792b18c8a2 Which sources are vault-note copies
  Given <source>
  When it is checked against the vault
  Then it is <verdict>

  Examples:
    | source                                          | verdict          |
    | an explicit .md with a sidecar outside the vault | kept             |
    | a non-.md source                                 | kept             |
    | a .md under the vault                            | kept             |
    | a .md with no sidecar                            | kept             |
    | a .md with a sidecar outside the vault           | a dropped copy   |
```

### 792b18c9 Detaching vanished sources

Requirement: [792b18c9 Detaching vanished sources](#792b18c9-detaching-vanished-sources) With neither `--empty` nor `--duplicates`, the chunk-index-store MUST remove
from the manifest every source whose path no longer exists, leaving every index file in place, MUST
write the manifest unless `--dry-run` is given, and MUST print
`prune: detached <N> source(s) — embedded chunks preserved (still searchable)`, prefixed
`[dry-run] ` for a dry run. With nothing to detach it MUST print `prune: no dead sources` and write
nothing; with no manifest it MUST print `prune: no manifest, nothing to prune` and succeed. A
manifest write failure MUST be returned.

Supports: [792a5p1a A deleted source is detached and stays searchable](../product/chunk-indexing.md#792a5p1-uat-scenarios), [792a5p1b A source that still exists is kept](../product/chunk-indexing.md#792a5p1-uat-scenarios), [792a5p1c An empty index has nothing to prune](../product/chunk-indexing.md#792a5p1-uat-scenarios), [792a5p1d A dry run changes nothing](../product/chunk-indexing.md#792a5p1-uat-scenarios).

#### 792b18c9a Scenarios

```gherkin
Scenario: 792b18c9a1 A dead source is detached and its index kept
  Given a manifest recording a live and a dead source, each with an index file
  When the index is pruned
  Then the dead source's entry is gone, both index files remain, and the output says detached 1 source and preserved

Scenario: 792b18c9a2 With nothing dead the manifest is kept
  Given a manifest whose every source exists
  When the index is pruned
  Then the manifest is unchanged

Scenario: 792b18c9a3 No manifest is not an error
  Given a chunks directory with no manifest
  When the index is pruned
  Then the prune succeeds

Scenario: 792b18c9a4 A dry run writes no manifest
  Given a manifest recording a dead source
  When the index is pruned with --dry-run
  Then the manifest is not written and the output is prefixed [dry-run]

Scenario: 792b18c9a5 A manifest write failure is returned
  Given a manifest recording a dead source and failing writes
  When the index is pruned
  Then the prune fails with that error

Scenario: 792b18c9a6 Existence and removal go through the filesystem
  Given the production prune wiring over a filesystem where /live.jsonl exists and /dead.jsonl does not
  When existence is checked and /chunks/empty.jsonl removed
  Then /live.jsonl exists, /dead.jsonl does not, and the removal reaches the filesystem
```

### 792b18c10 Removing empty index files

Requirement: [792b18c10 Removing empty index files](#792b18c10-removing-empty-index-files) With `--empty`, the chunk-index-store MUST read every index file ([792b18c12 Listing index files](#792b18c12-listing-index-files)),
remove each that is empty unless `--dry-run` is given, leave the manifest and every other file alone,
and print `prune: removed <N> empty chunk-index file(s) of <M> scanned`, prefixed `[dry-run] ` for a
dry run. A read or removal failure MUST be returned.

Supports: [792a5q1a Empty index files are removed](../product/chunk-indexing.md#792a5q1-uat-scenarios), [792a5q1b A dry run removes nothing](../product/chunk-indexing.md#792a5q1-uat-scenarios).

#### 792b18c10a Scenarios

```gherkin
Scenario: 792b18c10a1 Only empty index files are removed
  Given two empty index files, one holding a record, and a manifest
  When the index is pruned with --empty
  Then the two empty files are gone and the other file and the manifest are unchanged

Scenario: 792b18c10a2 A dry run removes nothing
  Given one empty and one non-empty index file
  When the index is pruned with --empty --dry-run
  Then the empty file remains and the output says [dry-run], removed 1 and of 2 scanned
```

### 792b18c11 Collapsing duplicate groups

Requirement: [792b18c11 Collapsing duplicate groups](#792b18c11-collapsing-duplicate-groups) With `--duplicates`, the chunk-index-store MUST group the manifest's entries
that are not recorded copies by content hash and kind, and in each group of two or more MUST keep the
entry with the fewest path separators, then the lexicographically smallest path, and remove each
other entry whose records the kept entry's index file covers ([792b18c7 Recording and evicting copies](#792b18c7-recording-and-evicting-copies)): its index file when present,
then its entry. It MUST count a refusal in a group where no member has an index file in one
`prune: refused <K> removal(s): canonical has no index file (zero-chunk sources; nothing to lose)`
line, and print every other refusal as
`prune: refusing to remove <path>: canonical <path> does not verifiably cover its records (needs review — a sibling's content survives)`
followed by a count line. It MUST report a removal failure through the warning log as
`[FAIL] <path>: <error>` and go on, write the manifest only when something was removed and it is not
a dry run, print `prune: removed <N> duplicate(s), retained <M> canonical(s)`, prefix every line
`[dry-run] ` for a dry run, and fail with `prune: <F> of <T> removals failed` when any removal
failed. No manifest MUST print `prune: no manifest, nothing to prune`; an undecodable one MUST be an
error.

Supports: [792a5r1a A duplicate group is collapsed](../product/chunk-indexing.md#792a5r1-uat-scenarios), [792a5r1b A duplicate with chunks of its own is refused](../product/chunk-indexing.md#792a5r1-uat-scenarios), [792a5r1c A dry run removes nothing](../product/chunk-indexing.md#792a5r1-uat-scenarios), [792a5r1d A second run removes nothing more](../product/chunk-indexing.md#792a5r1-uat-scenarios), [792a5r1e A failed removal is reported and fails the run](../product/chunk-indexing.md#792a5r1-uat-scenarios).

Rationale: this mode exists for groups indexed before ingest-time grouping, so it knows only the paths in the
manifest, not how each was found; the kept copy is chosen by path alone, and a later `--auto` ingest may replace it
once with a preferred copy ([792b18c7 Recording and evicting copies](#792b18c7-recording-and-evicting-copies)).
Recorded copies are skipped because they were never indexed. Groups whose sources yield no chunks are counted rather
than listed, so they do not bury the real removals.

#### 792b18c11a Scenarios

```gherkin
Scenario: 792b18c11a1 A group collapses to its kept entry
  Given two indexed entries with the same hash and kind whose records are covered
  When the index is pruned with --duplicates
  Then the removed entry's index file and entry are gone, the kept entry's remain, and the manifest is rewritten

Scenario: 792b18c11a2 A covered superset is removed
  Given a kept entry whose index holds every record of the other entry, with distinct content hashes
  When the index is pruned with --duplicates
  Then the other entry's index file and entry are removed

Scenario: 792b18c11a3 An entry whose index is already gone is dropped
  Given an entry with no index file whose kept twin's index exists
  When the index is pruned with --duplicates
  Then no removal is attempted and its manifest entry is dropped

Scenario: 792b18c11a4 Groups with no chunks are summarized
  Given a group in which no member has an index file
  When the index is pruned with --duplicates
  Then no "refusing to remove" line is printed, the summary counts the refusal, and every entry remains

Scenario: 792b18c11a5 An uncovered copy is refused and named
  Given a group whose kept entry has no index file while the other entry's index exists, or whose other entry holds a record the kept entry lacks
  When the index is pruned with --duplicates
  Then nothing is removed, the other entry remains, and the output has "refusing to remove <path>" and "refused 1 removal(s) listed above"

Scenario: 792b18c11a6 Kinds never group
  Given a .md and a .jsonl entry with the same hash
  When the index is pruned with --duplicates
  Then nothing is removed and the manifest is unchanged

Scenario: 792b18c11a7 Recorded copies are left alone
  Given an entry recorded as a copy
  When the index is pruned with --duplicates
  Then no removal is attempted and its entry remains

Scenario: 792b18c11a8 A dry run changes nothing
  Given a group with one removable entry
  When the index is pruned with --duplicates --dry-run
  Then nothing is written or removed and the output says [dry-run], removed 1 and retained 1

Scenario: 792b18c11a9 A second run is a no-op
  Given an index just collapsed by --duplicates
  When it is pruned with --duplicates again
  Then the output says removed 0, nothing is written, and the manifest is byte-identical

Scenario: 792b18c11a10 A removal failure is reported and the rest go on
  Given a group with two removable entries, one of whose index files cannot be removed
  When the index is pruned with --duplicates
  Then the warning log has "[FAIL] <path>:", the other entry is removed, the failed entry remains, and the prune fails with "1 of 2 removals failed"

Scenario: 792b18c11a11 Missing and undecodable manifests
  Given no manifest, or a manifest that is not valid JSON
  When the index is pruned with --duplicates
  Then the first prints "no manifest" and succeeds, and the second fails saying reading manifest

Scenario: 792b18c11a12 A manifest write failure is returned
  Given a removable entry and failing manifest writes
  When the index is pruned with --duplicates
  Then the prune fails with that error
```

### 792b18c12 Listing index files

Requirement: [792b18c12 Listing index files](#792b18c12-listing-index-files) The chunk-index-store MUST list its index files as the entries of the chunks
directory that are not directories and end in `.jsonl`, never from the manifest, MUST list none when
the directory does not exist, and MUST return any other listing failure.

Supports: [792a5p1a A deleted source is detached and stays searchable](../product/chunk-indexing.md#792a5p1-uat-scenarios), [792a5q1a Empty index files are removed](../product/chunk-indexing.md#792a5q1-uat-scenarios).

Rationale: a detached source has no manifest entry, so its chunks stay searchable only because readers find index
files by listing the directory.

#### 792b18c12a Scenarios

```gherkin
Scenario: 792b18c12a1 Index files are found by listing the directory
  Given a missing chunks directory, then one holding a.jsonl, notes.md, a sub directory and b.jsonl, then one whose listing fails
  When the index files are listed
  Then the first lists none, the second lists a.jsonl and b.jsonl, and the third fails
```

### 792b18c13 Chunk IDs

Requirement: [792b18c13 Chunk IDs](#792b18c13-chunk-ids) The chunk-index-store MUST give the set of chunk IDs in the index, each
record's `<source>#<anchor>`, read from every index file it lists, and MUST return an error when
listing, reading or decoding an index file fails.

Supports: [792a4e1a Chunk sources are merged without duplicates](../product/note-revision.md#792a4e1-uat-scenarios), [792a4f1a An invalid amend argument is refused before any write](../product/note-revision.md#792a4f1-uat-scenarios).

#### 792b18c13a Scenarios

```gherkin
Scenario: 792b18c13a1 The set holds every record's ID
  Given index files holding /sessions/a.jsonl#turn-1 and /docs/b.md#Heading
  When the chunk IDs are read
  Then the set holds both IDs and no other

Scenario: 792b18c13a2 A failure is an error
  Given a listing, read or decode failure
  When the chunk IDs are read
  Then the error is returned
```

### 792b18c14 First-indexed time

Requirement: [792b18c14 First-indexed time](#792b18c14-first-indexed-time) When rebuilding a source ([792b18c5 Rebuilding a source's index](#792b18c5-rebuilding-a-sources-index)), the chunk-index-store MUST stamp each chunk
it embeds with `ingested_at`: for a transcript the last timestamp the [transcript-reader](./transcript-reader.md) returned
([792b15c6 Last timestamp](./transcript-reader.md#792b15c6-last-timestamp)) when there is one, and otherwise, and for any other source, the current time. It MUST keep
the `ingested_at` of every record it already holds, including one whose chunk is unchanged on a later
rebuild. It MUST give a held record that has none the modification time that the manifest records for
its source, and none when the manifest has no entry. A record MUST decode with the `ingested_at` it was
encoded with, and with none when the field is absent.

Supports: [792a7p1a A Markdown chunk takes the ingest time](../product/memory-query.md#792a7p1-uat-scenarios), [792a7p1b A transcript chunk takes the session's last timestamp](../product/memory-query.md#792a7p1-uat-scenarios), [792a7p1c Ingesting again keeps the first time](../product/memory-query.md#792a7p1-uat-scenarios), [792a7p1d An old chunk gets its source's modification time](../product/memory-query.md#792a7p1-uat-scenarios).

Rationale: stamping each chunk when it is first indexed orders chunks within and across sessions, where a source
file's modification time would make one edit mark every chunk of the source as new. Every chunk of one transcript
shares the session's last timestamp, a sufficient approximation because each session is its own source. A record that
predates the field is stamped from the manifest in place, because that is the only time the store has for it.

#### 792b18c14a Scenarios

```gherkin
Scenario: 792b18c14a1 A Markdown chunk takes the current time
  Given a Markdown document and a clock reading 2026-06-15T12:00:00Z
  When it is ingested
  Then its record's ingested_at is 2026-06-15T12:00:00Z

Scenario: 792b18c14a2 A transcript chunk takes the session's last timestamp
  Given a transcript whose last entry is stamped 2026-05-10T14:00:00Z and a clock reading 2026-06-01
  When it is ingested
  Then its record's ingested_at is 2026-05-10T14:00:00Z

Scenario: 792b18c14a3 A rebuilt chunk keeps its first time
  Given an ingested transcript chunk stamped 2026-04-01, and the same text read again with a last timestamp of 2026-06-01
  When it is ingested again
  Then the index still holds one record for it, stamped 2026-04-01

Scenario: 792b18c14a4 A record without a time takes the manifest's time
  Given an index record with no ingested_at and a manifest entry for its source modified 2026-01-20
  When the source is ingested again with unchanged text
  Then the record's ingested_at is 2026-01-20

Scenario: 792b18c14a5 A record reads back with its time
  Given a record stamped 2026-03-10T08:00:00Z, encoded to an index file and read back
  When it is decoded, by the codec and by the loader of prior records
  Then its ingested_at is 2026-03-10T08:00:00Z

Scenario: 792b18c14a6 A record without the field has no time
  Given an index line that lacks ingested_at
  When it is decoded
  Then its ingested_at is the zero time
```

### 792b18c15 Reporting a prune backlog

Requirement: [792b18c15 Reporting a prune backlog](#792b18c15-reporting-a-prune-backlog) The chunk-index-store MUST report, without changing anything, whether the chunks
directory holds at least one zero-byte file whose name ends in `.jsonl` (skipping directories), and
whether `--duplicates` ([792b18c11 Collapsing duplicate groups](#792b18c11-collapsing-duplicate-groups)) would remove at least one entry. For the second it MUST read the
manifest and run the same grouping, kept-entry choice and coverage check in dry-run mode, discarding the
refusal lines, so an entry already recorded as a copy, a group whose members differ in kind or hash, and
a group whose every removal would be refused count as nothing to remove. It MUST report nothing to prune
when the directory cannot be listed or the manifest is missing, unreadable or undecodable, because a
detection failure must never fail the caller.

Supports: [792a11n1a Update reports the prune command when duplicates exist](../product/install-and-update.md#792a11n1-uat-scenarios), [792a11n1b A refusal-only backlog is silent](../product/install-and-update.md#792a11n1-uat-scenarios), [792a11n1c Empty index files are reported](../product/install-and-update.md#792a11n1-uat-scenarios), [792a11n1d A clean or missing index is silent](../product/install-and-update.md#792a11n1-uat-scenarios).

Rationale: `engram update` only notifies, so the notice must fire exactly when the command it names would work.
Re-running prune's own check is cheaper to keep correct than a second copy of it, and costs one manifest read, with
per-source reads only for groups of two or more.

#### 792b18c15a Scenarios

```gherkin
Scenario: 792b18c15a1 An empty index file is found
  Given a chunks directory holding an empty a.jsonl
  When the backlog is checked
  Then an empty file is reported

Scenario: 792b18c15a2 An empty file among non-empty ones is found
  Given a.jsonl with content and an empty b.jsonl
  When the backlog is checked
  Then an empty file is reported

Scenario: 792b18c15a3 Non-empty files are not a backlog
  Given only a.jsonl with content
  When the backlog is checked
  Then no empty file is reported

Scenario: 792b18c15a4 An empty file of another kind is ignored
  Given an empty manifest.json and no empty .jsonl file
  When the backlog is checked
  Then no empty file is reported

Scenario: 792b18c15a5 A missing directory has no backlog
  Given a chunks directory that does not exist
  When the backlog is checked
  Then no empty file is reported and there is no error

Scenario: 792b18c15a6 No manifest means no duplicates
  Given a chunks directory with no manifest.json
  When the backlog is checked
  Then nothing is reported for removal

Scenario: 792b18c15a7 Entries that are not a removable group are ignored
  Given entries with distinct hashes, entries sharing a hash but differing in kind, or an entry already recorded as a copy
  When the backlog is checked
  Then nothing is reported for removal

Scenario: 792b18c15a8 A group that prune would collapse is reported
  Given two entries with the same hash and kind, whose duplicate was never indexed or whose records the kept entry's index covers
  When the backlog is checked
  Then a removal is reported, and nothing is removed or written

Scenario: 792b18c15a9 A refusal-only group is silent
  Given two entries with the same hash and kind where the kept entry has no index file while the other's exists, or where no member is indexed, or where the other holds a record the kept entry lacks
  When the backlog is checked
  Then nothing is reported for removal

Scenario: 792b18c15a10 A malformed manifest is no backlog
  Given a manifest.json that is not valid JSON
  When the backlog is checked
  Then nothing is reported for removal, and the check neither panics nor returns an error
```
