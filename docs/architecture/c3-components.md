# L3 — Component view (inside C2 · engram CLI)

Decomposes **C2 · engram CLI** (from [L2](c2-containers.md)) into its Go packages/
components and the data they exchange. Reflects the current system; ⚠ = a
verified defect, re-verified each time this doc is edited (see
[memory-invariants](memory-invariants.md)). These component IDs are the
vocabulary the [L3 sequence/flow diagrams](#key-flows-l3--component-internal-sequences) reuse.

**Crucial: K1/K4/K6 are SEPARATE PROCESS invocations, not in-process collaborators.**
Each `engram <subcommand>` is its own process. The **C1·Skills orchestrator** (off-binary)
wires them: it reads one subcommand's **stdout** and shells the next. The binary's components
never call each other across subcommands. (This corrects an earlier draft that drew a
fabricated in-process `query→learn` edge — `query.go` has 0 `learn` refs; it ends at
`renderQueryPayload(stdout,…)`.)

```mermaid
flowchart TB
    classDef comp  fill:#1168bd,stroke:#0b4884,color:#fff
    classDef defect fill:#b1331f,stroke:#7a1f12,color:#fff
    classDef store fill:#23a,stroke:#127,color:#fff
    classDef ext fill:#999,stroke:#666,color:#fff
    classDef xcut fill:#5a7d5a,stroke:#3a5a3a,color:#fff,stroke-dasharray:4 3

    skills(["C1 · Skills orchestrator (off-binary)<br/>reads stdout, shells next subcommand"])

    subgraph PI[engram ingest — process]
      ing["K1 · ingest<br/>internal/transcript: Finder · JSONLReader<br/>internal/context: Strip · byte-budget · marker advance"]
    end
    subgraph PL[engram learn — process]
      learn["K4 · learn<br/>tier defaults · write-under-flock · O_EXCL"]
    end
    subgraph PQ[engram query — process]
      query["K6 · query orchestrator"]
      vg["K7 · vaultgraph<br/>ScanVault (note scan) — query uses ScanVault only"]
      cl["K8 · cluster<br/>KMeans · Silhouette · AutoK · BestMatch"]
    end
    subgraph PE[engram embed — process]
      eb["K5b · cli/embed (operator-run migration)<br/>RunEmbedApply · RunEmbedStatus"]
    end
    subgraph PP[engram prune — process]
      prune["K12 · prune (operator-run GC)<br/>RunPrune · read manifest · drop stale per-source manifest entries (index files preserved); --empty removes 0-byte .jsonl index files (#694)"]
    end
    subgraph PU[engram update — process]
      upd["K9 · update<br/>go install + copy skills/commands; --with-guidance adds guidance (Claude Code)"]
    end
    subgraph PW[engram amend / resituate / activate — write-modifier processes]
      wm["K4w · amend · resituate · activate<br/>read-modify-write a note/sidecar under .luhmann.lock; share K4/K5/K10 kernels"]
    end

    %% shared kernels: compiled into multiple subcommand processes; never call across them
    embed["K5 · embed (shared kernel)<br/>Text(body) · ContentHash · Sidecar · embedder"]
    lz["K10 · luhmann (shared kernel)<br/>ParseID · LetterLess"]
    dbg["K11 · debuglog (cross-cutting, all targets)"]

    vault[("C4 · Vault")]
    model[["C3 · MiniLM"]]
    sessions(["S5 · Session stores"])
    gotool(["S6 · Go toolchain"])

    skills -->|"shell engram ingest --auto"| ing
    skills -->|"shell engram learn (args)"| learn
    skills -->|"shell engram query --phrase"| query
    skills -->|"shell engram update"| upd
    skills -->|"shell engram embed apply (rare)"| eb
    skills -->|"shell engram amend / resituate / activate"| wm

    prune -->|"check whether each source still exists"| sessions

    ing --- sessions
    ing -->|stdout chunk identifiers| skills

    learn --> embed
    learn --> lz
    learn -->|note+sidecar under flock| vault

    query --> embed
    query --> vg
    query --> cl
    query -->|stdout payload| skills
    vg --> lz
    vg -->|read notes/wikilinks| vault

    eb --> embed
    eb -->|re-embed sidecars| vault
    embed --- model
    upd -->|go install| gotool

    class ing,learn,query,vg,cl,eb,embed,upd,lz,prune comp
    class dbg xcut
    class vault store
    class skills,sessions,gotool ext

    g0[["⚠ G0: BuildGraph resolves basename; learn writes bare ids → most edges dropped (census in memory-invariants.md)"]]:::defect
    vg -.-> g0
```

## Component catalog
| ID | Component | Key functions | Responsibility | ⚠ |
|---|---|---|---|---|
| K1 | `internal/transcript` + `internal/context` (via `engram ingest`) | `Finder.Find`, `JSONLReader.ReadFrom`, `context.Strip`, manifest write | Find sessions; check mtime/size/hash vs `manifest.json`; re-chunk and re-embed only changed sources within a byte budget; strip harness noise; emit chunk identifiers + write/update the per-source `manifest.json` entry (mtime/size/hash staleness). The manifest read-modify-write is serialized under `flock(.manifest.lock)` (shared with `prune`), and all index/manifest writes are atomic (temp-file + rename) — #660. | — |
| K4 | `cli/learn.go` + `cli/qa.go` | `writeLearnUnderLock`, `autoEmbedNote`; calls `nextLuhmannID` (in `cli/luhmann.go`); `cli/qa.go` is the sibling `qa` subcommand in the `learn` group (own process, shares these kernels — see "Flow: engram learn qa") | Write fact or feedback — no tier assignment (tier/L1/L2/L3 removed in recall-v2; `--tier` flag removed); compute next Luhmann id and write the note + sidecar atomically (temp-file + rename) under `flock(.luhmann.lock)` + `O_EXCL`. The same `.luhmann.lock` is held by **every** vault writer — `amend`, `resituate`, and `activate` acquire it at their command entry too. | **K1-lock invariant** untested |
| K5 | `internal/embed` | `Text`, `ContentHash`, `Sidecar`, embedder (Hugot/GoMLX simplego) | Embed situation and body text; write/read dual-vector `.vec.json` (`situation_vector` + `body_vector` + `embedding_model_id` + `content_hash` + `last_used`). `bestVector` selects the axis with the higher query cosine at recall time. | **M4** (model homogeneity) |
| K6 | `cli/query.go` | `RunQuery`, `runQuery`, `buildMatchedSetFromPhrases`, `buildRecentFillItems`, `capWithNoteFloor`, payload assembly | Single query path: per phrase embed → top-30 (notes+chunks, recency-biased cosine), reserving up to `noteFloorK=5` of those slots for relevance-qualified notes (`capWithNoteFloor`) so higher-cosine chunks cannot fully evict a note that cleared the floor; union across 10 phrases, dedup max score, relevance floor (baseScore < 0.25), cap matched set at ~300 (`matchSetCap`); ONE AutoK cluster over matched set (D1 preserved); `candidate_l2s` = within-cluster top-5 from matched note members **plus tag-nominated notes** sharing ≥1 vocab term with the top-3 delivered notes (budget: `tag_nominations_added`/`dropped`, pool cap 40/cluster) plus superseded-note ride-alongs; Channel 2 appends the newest chunks by IngestedAt (`recentFillChunks`, default 25, configurable via `--recent-fill` / `ENGRAM_RECENT_FILL`), deduped, tagged `recent`, not in any cluster. All notes cluster as normal notes. Optional `--lazy-chunks` renders chunk items path/source-only (notes keep content), fetched on demand via the `show-chunk` subcommand (`cli/show_chunk.go`). | — |
| K7 | `internal/vaultgraph` | `ParseWikilinks`, `ParseBasename`, `BuildGraph`, `ScanVault`, `UnresolvedTargets` | Build the directed wikilink graph (node=basename); scan vault notes for query; identify unresolved links for `engram check`. | **G0** (basename-only resolution); G5 **RETIRED** (episode kind removed; `[[x]]` in chunk bodies no longer parsed as vault edges) |
| K8 | `internal/cluster` | `KMeans`, `Silhouette`, `AutoK`, `CosineDistance` | Pick k by silhouette; cluster the matched set. Silhouette is O(n²) per k swept, so clustering inputs are bounded: the matched set is hard-capped at `matchSetCap`=300 (10 phrases × top-30 per phrase) before clustering; recency-channel chunks (`recentFillChunks`, default 25; `--recent-fill`) are appended un-clustered and never enter K8. | C1 resolved — `TestInvariant_C1_ClusteringDeterminism`, `TestKMeans_DeterministicAcrossRuns`; L3-1 RETIRED (L3 note kind removed) |
| K9 | `internal/update` | `Run`, `SourceLocal/Remote` | `go install` the binary; copy refreshed skills/commands per harness; `--with-guidance` deploys the guidance docs under `guidance/` (`recall.md`, `delegate.md`) to `~/.claude/engram/` (Claude Code only; opt-in; OpenCode deferred); sentinels `ErrGoNotFound`/`ErrNoHarness`/`ErrSkillsSrcMissing`. | **U1** resolved — `TestUpdater_Run_Local_Idempotent_Property` |
| K10 | `internal/luhmann` | `ParseID`, `LetterLess`, sort/tie-break | Parse and order Luhmann ids; **shared kernel** consumed by K4 (`cli/learn.go`, `cli/luhmann.go`) AND K7 (`vaultgraph/{selector,scanner}.go`). | — |
| K11 | `internal/debuglog` | tail-friendly sink | Cross-cutting debug log threaded through every CLI target (`targets.go`, `cli/signal.go`); L1 deferred it to here. | — |
| K5b | `cli/embed.go` | `RunEmbedApply`, `RunEmbedStatus`, `selectStates` | The `engram embed apply/status` subcommand (separate process, operator-run for model migration): re-embeds notes whose sidecar is missing/stale/incompatible via the shared K5 package; `apply` writes sidecars, `status` reports counts. Wired in `targets.go` (grep `Name("embed")`). | drives **M4** remediation |
| K12 | `cli/prune.go` | `RunPrune` | The `engram prune` subcommand (operator-run GC): reads the chunk-index manifest and, for every source whose file no longer exists, drops its manifest entry — the per-source index file (embedded chunk vectors) is left on disk, since chunk search discovers `.jsonl` files by directory scan and never consults the manifest, so detached chunks stay fully searchable (#659). Acquires `flock(.manifest.lock)` around the manifest read-modify-write (shared with `ingest`) so a concurrent ingest/prune cannot lose updates — #660. A separate `--empty` mode (`pruneEmptyLocked`, + `--dry-run`) instead removes existing 0-byte `.jsonl` index files left by a zero-record source under the `rebuildIndex` guard (#694) — ranking-neutral, re-reading each file live at delete time rather than deleting off a frozen enumeration. Not part of the recall/learn/please flows — manual cleanup only. Wired in `targets.go` (grep `Name("prune")`) alongside ingest/query. | — |

## The recurring defect shape (feeds the Phase-4 ADR) — corrected per Phase-2 antagonist
The canonical example of the silent-mismatch bug class:
- **G0** — write an edge as `[[id]]`; resolve an edge as `[[basename]]`. (disjoint keys)

The unifying invariant: **for every write/read pair over the same datum, the read key
must be a function of (or equal to) the write key, and a mismatch must be loud, not silent.**

**M4 is a DIFFERENT mechanism — do not fold it in.** It compares the *same* key (`model@v`) for
*equality* — that's correct — and the defect is the **policy on a legitimate non-match**: off-model
sidecars are dropped, silent only under *partial* migration (when all hits filter out, `query.go:62`
*does* raise `errQueryNoEmbeddings`). So M4 = "version-gate drops off-model sidecars; guarded only in
the all-empty case," a separate finding.

## Dead/test-only surface
`internal/vaultgraph`'s graph-expansion half — `BuildGraph`, `BFSWithCap`, `InDegreeIn` (and the
MOC-navigation helpers) — has **zero production consumers** (the 3-hop-subgraph recall path was
removed in recall-v2; no `vault`/`graph`/`follow` subcommand). Only `ScanVault`, `ParseWikilinks`,
`ParseBasename`, and `UnresolvedTargets` are live — consumed by `query` (note scan), `check`
(unresolved-link report), and `amend`/`resituate`/`show`/`embed`. K7 still bundles the dead
graph-expansion subsystem; a future cleanup should confirm and propose its deletion.

## Data contracts (what crosses component edges) — corrected
- **ingest → skill → learn (NOT in-process):** `engram ingest --auto` scans chunk sources, re-chunks
  changed content, emits chunk identifiers + status line to **stdout**; the skill reads them and
  shells `engram learn fact|feedback` as a *new process* per candidate.
- **K6 payload (to stdout → skill):** `items[]` (matched notes+chunks + recency-channel chunks
  tagged `recent`; notes carry full content inline, chunk items path/source-only under `--lazy-chunks`, fetched via `show-chunk`) ∪ `clusters[].members` (paths, from matched set only) ∪
  `clusters[].candidate_l2s` (`[{path, cosine, content}]`, within-cluster top-5 plus tag-nominated notes sharing a vocab term with top-3 delivered notes; budget: `tag_nominations_added`/`dropped`, cap 40/cluster; ride-alongs for superseded notes) ∪ `budget`.
  No `nearest_l3` field; no `hubs` field; recency-channel items appear in `items[]` but never in
  any cluster's `members[]`. The skill — not the binary — consumes it and may shell `engram amend`
  (covered/near) or `engram learn` (absent) for recall-time lazy synthesis. Activation is
  agent-driven: the binary emits no `activated` flag; the skill calls `engram activate` on only
  the notes it actually used.
- **K5 sidecar:** `{situation_vector[384], body_vector[384], embedding_model_id, dims, content_hash, last_used}` — dual-vector; `bestVector` selects the axis with the higher query cosine. `content_hash` covers the note's situation + body text. `last_used` is bumped by `engram activate`; it is excluded from `content_hash` so activation never marks a note stale. Staleness tracking (mtime/size/hash `manifest.json`) lives in `engram ingest` (K1); there is no separate learnmarker package. GC of manifest entries whose source file no longer exists is handled by K12 (`engram prune`, operator-run).

## Key flows (L3 — component-internal sequences)

These zoom into a single `engram` subcommand process and show the K-component call order verified
against the code. Each subcommand is its OWN process; nothing here crosses to another subcommand.
[L2](c2-containers.md) shows the skill↔binary orchestration; this is what one binary call does inside.

### Flow: `engram query` internals (RunQuery)

This is the sole query path (the `--synthesis`/BFS-subgraph path was removed in the 2026-06-20 deep clean; unified clustering is now the only mode).

Verified order: `Scan` → `loadCompatibleSidecars` → `loadClusterChunkRecords`
→ `buildMatchedSetFromPhrases` (per-phrase unified ranking) → `applyFloorAndCap` →
`clusterMatchedSet` (K8) →
`mergeProvenances` → `buildRecentFillItems` → `renderQueryPayload`.

```mermaid
sequenceDiagram
    autonumber
    participant Q as K6 query
    participant Em as K5 embed
    participant Md as C3 model
    participant Vg as K7 vaultgraph
    participant Cl as K8 cluster
    participant V as C4 vault

    Note over Q: RunQuery — one process; args from the skill, output to stdout
    Q->>Vg: Scan = vaultgraph.ScanVault
    Vg->>V: read note files; ParseWikilinks → Outgoing at scan time
    Vg-->>Q: notes (+ parsed wikilinks)
    Q->>V: loadCompatibleSidecars — read sidecars, drop off-model [M4]
    Q->>V: loadClusterChunkRecords — read chunk index
    loop per phrase (10 phrases)
        Q->>Em: Embed(phrase)
        Em->>Md: encode
        Md-->>Em: vector
        Em-->>Q: query vector
        Note over Q: score notes (rankCandidates, recency-biased) + chunks (per-phrase scorer, recencyMultiplier); merge into one list; take top-30 (capWithNoteFloor reserves up to noteFloorK=5 slots for floor-qualified notes)
    end
    Note over Q: union across phrases, dedup keeping max score; drop baseScore < 0.25; cap at matchSetCap=300 → matched set
    Note over Q: matched set holds both note and chunk members (unified ranking)
    Q->>Cl: clusterUnionForSynthesis — ONE AutoK k-means + silhouette over matched set (D1)
    Cl-->>Q: clusters with candidate_l2s (within-cluster top-5 by centroid cosine; tag-nominated notes + ride-alongs added post-cluster)
    Note over Q: mergeProvenances; applyProjectFilter
    Note over Q: buildRecentFillItems — newest chunks by IngestedAt (recentFillChunks, default 25), deduped vs matched set, tagged recent; NOT in any cluster
    Note over Q: renderQueryPayload → stdout (items[matched+recent], clusters[candidate_l2s], budget)
```

### Flow: `engram learn` write internals (writeLearnUnderLock)

Verified order: `Lock` → `ListIDs` → `nextLuhmannID` → `assembleLearnContent` → `WriteNew(O_EXCL)`
→ `autoEmbedNote` (`Text` → `ContentHash` → encode → `Sidecar` write).

```mermaid
sequenceDiagram
    autonumber
    participant L as K4 learn
    participant Lz as K10 luhmann
    participant Em as K5 embed
    participant Md as C3 model
    participant V as C4 vault

    Note over L: runLearn → writeLearnUnderLock — one process
    L->>V: Lock(.luhmann.lock) — flock spans id-compute→write [K1-lock]
    L->>V: ListIDs (existing Luhmann ids)
    Note over L: nextLuhmannID (K4, cli/luhmann.go)
    L->>Lz: ParseID · LetterLess — sort/tie-break existing ids
    Lz-->>L: ordered ids → next id
    Note over L: assembleLearnContent — frontmatter + body
    L->>V: WriteNew note (O_EXCL — create-only, errors if exists)
    L->>Em: autoEmbedNote(path, content)
    Note over Em: Text — body; ContentHash hashes body
    Em->>Md: encode
    Md-->>Em: vector
    Em->>V: write .vec.json sidecar (vector + model_id + content_hash)
    Note over L: release lock; emit written path → stdout
```

### Flow: `engram learn qa` capture path (RunLearnQA)

Captures a question/answer pair as two notes with asymmetric query-time participation
(D5′ — see `adr.md`, ADR-0012). `engram learn qa` is a sibling subcommand in the same
`learn` process group as K4 (`cli/qa.go`, sharing K4's `autoEmbedNote` and
`applyVocabAssignmentAfterLearn` kernels) — its own process, invoked when a skill (recall
Step 4 / learn Step 2.5) hands off to write-memory with `kind=qa`.

Verified order: `RunLearnQA` → `validateLearnQAArgs` → `ensureQAVault` → (`ReadFile` if `--answer-file`) → `validateContributors`
→ `writeQANotesUnderLock` (`Lock` → `WriteNew` Q → `WriteNew` A, orphan-cleanup on A-write
failure → release) → `autoEmbedNote` (Q, no vocab) → `autoEmbedNote` (A) →
`applyVocabAssignmentAfterLearn` (A only) → stdout (`cli/qa.go`).

```mermaid
sequenceDiagram
    autonumber
    participant L as K4 learn
    participant Em as K5 embed
    participant Md as C3 model
    participant V as C4 vault

    Note over L: RunLearnQA (cli/qa.go) — one process, args from the skill (write-memory kind=qa handoff, per recall Step 4 / learn Step 2.5)
    Note over L: validateLearnQAArgs — slug, question, exactly one of --answer/--answer-file, source, certainty
    L->>V: ensureQAVault — stat vault dir, init if absent
    opt --answer-file provided (instead of --answer)
        L->>V: ReadFile(answer-file) — load the answer body from disk
    end
    L->>V: ListMD + validateContributors — every --contributors basename must exist in the vault
    L->>V: Lock(vault) — flock spans both note writes [K1-lock]
    L->>V: WriteNew Q note (O_EXCL) — type: qa-question, answered_by: (a-basename), body + "Answered by: [[a-basename]]"
    L->>V: WriteNew A note (O_EXCL) — type: qa-answer, answers: (q-basename), certainty, contributors?, body + "Answers: [[q-basename]]" (+ "Contributors: [[...]]" if provided)
    alt A-write fails
        L->>V: RemoveFile(Q note) — best-effort orphan cleanup (lock released by defer)
        Note over L: return descriptive error to caller — NO embed / vocab / stdout on this path
    else A-write succeeds
        Note over L: release lock (defer) — embed/vocab run OUTSIDE the lock, success path only
        L->>Em: autoEmbedNote(Q note) — embed only, NO vocab assignment (D5′: Q notes carry no vocab)
        Em->>Md: encode
        Md-->>Em: vector
        Em->>V: write Q .vec.json sidecar
        L->>Em: autoEmbedNote(A note)
        Em->>Md: encode
        Md-->>Em: vector
        Em->>V: write A .vec.json sidecar
        L->>V: applyVocabAssignmentAfterLearn(A note) — vocab tags assigned, A-note ONLY
        Note over L: emit qPath, aPath → stdout
    end
    Note over L,V: at later, SEPARATE `engram query` (K6) processes (see the query flow above) — A-note competes in the main matched set like any fact/feedback note, Q-note EXCLUDED at all four query-pipeline seam points (isQueryExcludedKind) — applyFloorAndCap, isFloorQualifyingNote (capWithNoteFloor reservation), mergePhraseIntoUnion (per-phrase union), tag-nomination (TermIndex build + addNominationsForTerm guard)
    Note over L,V: at later, SEPARATE `engram vocab stats` processes — Q-note excluded from the totalNotes/untagged counter (collectVaultStats skips isQAQuestionFilename), A-note counts normally
    Note over L,V: deferred, NOT built — a dedicated q-space channel + answered_by ride-along (round 3) requires Arm V's larger-n premise check to reach PASS (≥80%); the larger-n run already came back BORDERLINE (63%, 19/30), so round-3 remains unlicensed pending a further check (ADR-0012; ROADMAP Track C)
```

