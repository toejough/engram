# Plan: chunk-index deduplication + two ingest/prune bugfixes

Cycle: /please, 2026-07-25. **Five units**, separately committed.

Two asks, both verbatim:

1. "fix the two bugs you found, as well as implementing the full general fix" — Units 1-4.
2. Mid-cycle addition: "We also need the update command to run the prune once for people
   updating to this version from an old one that might have accumulated copies" — Unit 5.

Unit 4 (retroactive cleanup) was flagged by Gate A's ask-alignment reviewer as beyond the ask.
It was escalated to Joe with fold/split options and he chose **fold it in** — recorded here per
the propose-and-gate rule, not decided by the author.

## Measured starting state

Re-derived live during Gate A (two independent counts, hence the drift):

| Measure | Author count | Reviewer recount | Command |
| --- | --- | --- | --- |
| manifest sources | 62,397 | 62,415 | group `chunks/manifest.json` entries |
| exact-content duplicates of another source | 49,295 (79%) | 49,291 (79.0%) | group by `file_hash`, sum `len(group)-1` |
| … still present on disk | 48,952 | 48,948 | same + `os.path.exists` |
| redundant chunk *records* | ~11,307 of 73,945 (15.3%) | — | prior investigation |
| duplicate share of returned items (production arm, `--lazy-chunks`) | 9.6% mean / 28.6% worst | — | `engram query` |
| scan cost of the junk | −290 ms of ~5.1 s (−5.7%) | — | job-free hardlinked index, 3 runs |

**These numbers are context, not a target.** Every deletion set is re-derived mechanically
against the live manifest at execution time (note 178); no unit compares against a count in
this document.

**Honest bound.** Exact-content dedup reaches the 79%. It does NOT reach a *drifted* copy — a
snapshot of a note since amended hashes differently, so both survive. Measured: 50 of 256
shadowed vault notes have drifted. Rule B covers that class for vault notes specifically.
Neither rule covers a drifted non-vault copy (e.g. an edited fork of a doc in a worktree);
that is out of scope and stated so.

## Assessment of the ask

Sound. The general fix is right over the cheap `jobs` exclude: `jobs` is 29% of the problem and
a path blocklist cannot reach worktrees, backup-restore trees, or `/tmp` config pools. Two
challenges, both recorded and mitigated rather than blocking:

1. **Retroactive deletion is destructive and one-shot** — notes 241/178/25. Mitigations are
   normative in Unit 4, not aspirational.
2. **`--auto`'s ancestor sweep is intentional** — it is how engram finds `~/.claude` memory
   (vault note 296 diagnosed this exact mechanism on 2026-07-19). The fix dedups; it does not
   stop sweeping.

## The append-only invariant — narrowed, not reversed

Gate A found "append-only / never deletes" asserted in `README.md:97`,
`docs/GLOSSARY.md:550-551`, `docs/architecture/c1-system-context.md:194,223`,
`agent-instructions/skills/learn/SKILL.md:36`, and Go doc comments at
`internal/cli/ingest.go:464-465,476,584`.

Two distinct guarantees hide under one phrase, and this cycle touches only one:

- **Within a source** (`mergeChunkRecords`, `ingest.go:464-476`): prior records survive a
  re-chunk. **Unchanged by this cycle.**
- **Across sources**: no index file is ever removed, so content survives even when its source
  file is deleted (`RunPrune`'s doc comment, `prune.go:31-39`). **Narrowed**: a duplicate index
  may be removed *only when a byte-identical retained twin verifiably exists*, so no content
  becomes unsearchable. The user-facing promise — "deleting a source file never loses the
  recovered memory" — still holds exactly.

ADR-0021 states this narrowing explicitly and the doc updates use this framing. The reviewer's
"append-only is being revoked" reading is what the docs would say if we changed the text
carelessly; the plan's job is to make the narrowing legible.

## Unit 1 — `engram prune --dry-run` is ignored on the default path (bug)

`RunPrune` (`internal/cli/prune.go:40-98`) calls `deps.WriteFile` at `:91` unconditionally;
`args.DryRun` is read only at `:138` and `:154`, inside `pruneEmptyLocked`.

- **RED:** in `internal/cli/prune_test.go`, add `TestRunPrune_DryRunDoesNotWriteManifest`:
  manifest fixture with two sources — one whose `Exists` stub returns true, one false (that is
  the "dead source": a manifest key whose file no longer exists, the condition at `:71`).
  `PruneArgs{ChunksDir: "/c", DryRun: true}`. `WriteFile` stub calls `t.Fatal` if invoked.
  Assert stdout contains `[dry-run] `. Fails today at the `t.Fatal`.
- **GREEN:** guard `:86-94` on `!args.DryRun`; emit the same `[dry-run] ` prefix
  `pruneEmptyLocked` uses at `:137-140`.
- **VERIFY:** `targ test`; then the installed binary against a **copy** of the chunks dir —
  `shasum manifest.json` before and after `engram prune --dry-run --chunks-dir <copy>` must be
  identical (a hash, not mtime — JSON key order is not stable).

Existing `prune_test.go` / `prune_integration_test.go` construct `PruneArgs` with `DryRun`
defaulting false, so none of their assertions change.

## Unit 2 — non-persistent prefixes apply only to session logs (bug)

`ExcludePrefixes: spec.NonPersistentPrefixes` is wired only onto the SessionLogs root
(`sweepspec.go:133-140`). `sweepListerFrom` (`ingest.go:718-757`) exempts the root's own path
from every check (`if path == root.Path { return nil }`, `:735-736`), so a root that *is*
under a throwaway prefix is swept whole. Confirmed live: `/tmp/cummatrix-n1/cfgpool/warm0/…`
plus ~495 siblings are in the manifest today.

`NonPersistentPrefixes` holds slugified directory-**name** prefixes (`-private-tmp-`) matched
against `entry.Name()` — not path prefixes. So this needs a sibling field, not an overload.

- **RED:** in `internal/cli/sweepspec_test.go`, add
  `TestResolveSweepRootsDropsRootsUnderNonPersistentPaths`: `SweepEnv{Cwd:
  "/private/tmp/work/proj"}` with `IsDir` true for that tree; assert no returned root has a
  path under `/private/tmp`. Fails today. Existing
  `TestResolveSweepRootsAttachesPrefixesToSessionRootOnly` (`:46`) still passes — the new field
  is additive.
- **GREEN:** add `NonPersistentPathPrefixes []string` to `SweepSpec` beside
  `NonPersistentPrefixes` (default `{"/tmp", "/private/tmp", "/var/folders",
  "/private/var/folders"}`); at the end of `ResolveSweepRoots` (`sweepspec.go:108-147`) drop
  any root whose own `Path` is under one of them. **Apply the drop only to the auto-derived root
  groups (RepoMarkdown, AncestorClaudeDirs, AncestorPiDirs, SessionLogs), before `ExtraRoots` is
  appended.** Three things bypass it, all deliberate configuration rather than accidental
  sweeping: explicit `--sweep`/`--transcript`/`--markdown` (`ingest.go:177-182`, the documented
  escape hatch), and `SweepSpec.ExtraRoots`, whose own doc comment at `sweepspec.go:51-52`
  promises it is "swept verbatim". Gate B caught the first implementation filtering `ExtraRoots`
  too — `DefaultSweepSpec()` plus an extra root under `/tmp` returned zero roots, silently. The
  regression test must use `DefaultSweepSpec()`, not a bare `SweepSpec{}` literal: the existing
  `TestResolveSweepRootsHonorsSpecToggles` uses a literal, so the new field is nil there and the
  filter never engages, which is exactly why it masked the gap.
- **VERIFY:** `targ test`; installed binary run with cwd under `/private/tmp` adds zero sources.

**Shipped addendum, 2026-07-26 (late addition — not in the RED/GREEN above).** A second,
independent sweep gap was found and fixed alongside this one: the `.claude` ancestor sweep had no
exclude for its own `jobs/` subdirectory — agent-harness scratch that can include whole snapshot
copies of the vault — while the `.pi` ancestor sweep already excluded it
(`piExcludes := []string{"jobs", "projects"}`, pre-existing, unchanged by this plan). `ClaudeExcludeDirs`
now carries the matching `jobs` exclude, restoring parity between the two sweeps. This is
name-based (a subdirectory name to prune during an ancestor `.claude` walk) and distinct from the
`NonPersistentPathPrefixes` fix above, which is path-based (dropping a whole resolved root whose
own path sits under a throwaway prefix) — the two together are what stops `.claude/jobs` content
from being swept at all. Test: `TestResolveSweepRootsClaudeAncestorExcludesJobsScratch`
(`internal/cli/sweepspec_test.go`). Real-world impact, measured via Unit 4's retroactive prune:
forensics attributed 1,397 of the 1,960 removed duplicates (71%) to `~/.claude/jobs` alone — the
single largest identifiable source of the whole backlog. See ADR-0021 decision 4.

## Unit 3 — dedup at ingest (the general fix, forward-looking)

Gate A refuted the first draft's "minimal change behind existing DI seams". Two structural
additions are required and are specified here.

**Structural change 1 — resolve duplicates across sources, not one at a time.**
`gatherSources` (`ingest.go:268-324`) returns bare `sourceRef`s with no hash; `RunIngest`
(`:103-120`) calls `ingestSource` per path with no cross-source view. So:

0. **Prerequisite — carry origin on `sourceRef`.** Gate B on the baseline refactor established
   that origin is currently LOST at the merge point: `piSessionSources` tags its files
   `explicit: true` (indistinguishable from literal `--transcript`/`--markdown` flags) and
   `sweptSources` tags repo-markdown, `.claude` ancestors, `.pi` ancestors and session logs all
   `explicit: false`. `sourceRef{path, explicit}` therefore cannot answer the question
   `selectCanonical` asks. Add an origin tag (explicit / repo / claude-ancestor / pi-ancestor /
   session-log / manual-sweep) set by each stage helper at the point it constructs the
   `sourceRef` — each helper already knows its own origin — and carry it through the merge.
   Without this step the precedence rule below is unimplementable except by fragile
   path-matching. While here, fold the duplicated `ListSources → filter by ext → append` loop
   body shared by `piSessionSources` and `sweptSources` into one helper parameterised by origin
   and the chunks-prefix skip (Gate B flagged the duplication; it becomes natural once origin is
   a parameter).
1. Build a `hash → []sourceRef` registry before the ingest loop, seeded from the manifest's
   cached `FileHash` (no extra reads for unchanged sources; only unknown/changed sources are
   read and hashed). **This seeding is load-bearing, not an optimisation:** `gatherSources`
   never reads file bytes, and the pipeline's cheap-skip deliberately avoids re-reading
   unchanged files, so a dedup pass that eagerly hashed every source would defeat the design it
   sits in.
2. Rank each hash group by `selectCanonical(candidates []sourceRef) sourceRef` — a **pure
   function**, first match wins, evaluated in this order:
   1. `explicit == true` (came from `--transcript`/`--markdown`/`--pi-sessions`);
   2. under the repo root (`repoRootFor(env)`);
   3. under an ancestor `.claude`/`.pi` dir, **closest ancestor first**;
   4. anything else;
   5. tie-break: fewest path separators, then byte-wise lexicographic on the full path
      (case-sensitive). Deterministic regardless of walk order.
3. Index only the canonical member; record the others in the manifest as duplicates pointing at
   the canonical source, so a later run does not re-read them. **This needs a new field:**
   `manifestEntry` (`ingest.go:158-162`) is only `{SourceStat, FileHash}` today. Add
   `DuplicateOf string` (empty for canonical entries), rather than relying on the implicit
   "entry with no index file = suppressed duplicate" convention, which is indistinguishable
   from the crash state below.

**Structural change 2 — eviction, not just skipping.** A low-precedence copy indexed on day 1
and a high-precedence twin appearing on day 5 must converge: when a newly-ranked canonical
outranks an already-indexed member of its hash group, remove the stale index file and manifest
entry. This shares one helper with Unit 4 (`removeDuplicateIndex`) rather than leaving Unit 4 as
a one-off — without it, out-of-order arrivals re-accumulate exactly as today.

**Rule B (vault copies) needs a new dependency.** `IngestArgs`/`IngestDeps` (`ingest.go:20-52`)
has no vault path today; the analog is `resolveVault` (`learn.go:432`), wired only into
`LearnDeps`. Add `Vault string` to `IngestArgs` and the resolver to `IngestDeps`, wired in
`newIngestDeps` + `targets.go` exactly as `ResolveChunksDir` is. Rule: a `.md` with a sibling
`<name>.vec.json` is a vault note (verified live: the vault is 461 `.md` / 461 `.vec.json`,
1:1, `internal/embed/sidecar.go` `SidecarPath`); index it only when it sits under the resolved
vault path.

- **RED** (`internal/cli/ingest_test.go`, imptest mocks):
  - `TestIngestDedupsExactContentByPrecedence`: `/repo/notes/a.md` and
    `/home/u/.claude/copy/a.md` with identical bytes → exactly one index build; canonical is the
    repo-root path.
  - `TestIngestEvictsLowerPrecedenceDuplicate`: manifest pre-seeded with the `.claude` copy
    indexed; the repo copy then appears → repo copy indexed, `.claude` index removed.
  - `TestIngestSkipsVaultCopyOutsideCanonicalVault`: `/jobs/tmp/snap/vault/1.md` +
    sibling `1.vec.json`, differing bytes from the live note (the drifted case Rule A misses)
    → not indexed.
  - `TestSelectCanonicalIsOrderIndependent`: rapid property — shuffle a candidate slice, assert
    the same winner every time.
- **GREEN:** minimal implementation of the three pieces above.
- **REFACTOR + Gate B.**
- **VERIFY:** `targ check-full`; then the installed binary against a **copy** of the live chunks
  dir — `engram ingest --auto` adds no new duplicate hash groups, and a chunk-query spot check
  returns the same live notes it did before.

**Shipped addendum, 2026-07-26 (found and fixed after the initial ship of Units 3/4 in this same
cycle — not in the RED/GREEN above; shipped together with Unit 4's matching addendum below).**
"Structural change 2" above, and Unit 4's original gate below, assumed byte-hash identity of a
source's CURRENT content is sufficient proof that evicting a duplicate's index loses nothing.
That premise is FALSE: `mergeChunkRecords` is append-only WITHIN one source, so a source's
`.jsonl` index file accumulates every chunk record from every past ingest of that path — not just
its current content. Two sources can therefore be byte-identical RIGHT NOW (same hash group) while
their index files hold entirely disjoint historical chunk records; evicting one on the strength of
the other's mere existence can silently destroy real, unique content. The weaker, existence-only
gate this replaces had already shipped and run against the real chunk index before the flaw was
found; measured on that real corpus, 124 of 6,612 surviving index files carry chunk records from
more than one ingest day — exactly the shape the weaker gate could not distinguish from a
safe-to-delete duplicate. (This does not establish that any record was actually lost by the
weaker gate — only that it was capable of losing one under exactly this condition, which is why
it was replaced.) Shipped fix: a record-level subset gate
(`canonicalCoversDuplicateRecords` / `duplicateRecordsCoveredBy`,
`internal/cli/ingest_dedup.go`) additionally requires, immediately before any eviction, that the
canonical's index file exists right now AND every one of the duplicate's own chunk records (by
content hash) is already present among the canonical's; when this cannot be confirmed the eviction
is refused (reported to stdout, pointing at `prune --duplicates` for later resolution) rather than
performed. `reconcileGroupCanonicalFirst` reconciles the group's canonical member first, so its
records are loaded once per group (not once per duplicate) and reused across every duplicate in
it. See ADR-0021 decision 5.

## Unit 4 — `engram prune --duplicates` (retroactive cleanup; folded in with Joe's approval)

Removes index files + manifest entries for non-canonical members of each hash group, keeping
exactly one per hash by Unit 3's `selectCanonical`.

Deletion safety, all normative:

- Re-derive the duplicate set against the live manifest at run time; never from this document.
- Gate every delete on its retained twin existing **now** (`deps.Exists` on the canonical
  source's index file immediately before removing the duplicate).
- Per-item failure handling: on any removal error, write `[FAIL] <path>: <err>` to stderr,
  continue, and exit non-zero with `N of M removals failed`. Never exit 0 with failures.
- Convergent: a second run removes nothing and exits 0.
- `--dry-run` works from the start (Unit 1 makes it real) and reports the counts it *would*
  remove plus the retained count.
- **VERIFY:** the Gate B reviewer live-runs `--dry-run` then the real delete against a **full
  copy** of the chunks dir (note 241's copy-verification), confirms convergence on re-run and
  zero warnings, and confirms `engram query` on the copy returns the same live notes. Only then
  does it run against the live index.

**Shipped addendum, 2026-07-26 (found and shipped alongside Unit 3's matching addendum above — not
in the bullets above).** "Gate every delete on its retained twin existing now" above describes
existence-only gating (`deps.Exists` on the canonical's index file). This was strengthened by the
same post-ship finding Unit 3's addendum describes: existence of the canonical's index file
is necessary but not sufficient, since a byte-identical twin's index file can hold different
historical chunk records (`mergeChunkRecords` is append-only within a source). The shipped gate
additionally requires every one of the duplicate's own chunk records to already be present among
the canonical's records, via the same `canonicalCoversDuplicateRecords` / `duplicateRecordsCoveredBy`
helpers Unit 3 uses. A group failing this check is refused, not removed. Refusals are classified
for reporting, added after Gate B measured a real run printing 47,396 near-identical
"canonical index missing" lines, burying genuine removals and anomalies in noise: a
bulk-summarized STRUCTURAL refusal (the canonical has no index file at all — a zero-chunk source,
nothing lost; ~88% of a real manifest) versus an individually-named ANOMALOUS refusal (a sibling's
surviving index proves the group holds real content, worth a human look). See ADR-0021 decisions
5 and 8.

## Unit 5 — `engram update` detects the dedup backlog and notifies (reversed mid-cycle from an auto-run design)

Requested verbatim: "We also need the update command to run the prune once for people updating
to this version from an old one that might have accumulated copies." Without SOME signal, only
people who read the release notes and run `prune --duplicates` by hand ever get the cleanup —
everyone else keeps the accumulated copies forever, which is the whole population this cycle is
for.

**Reversed, 2026-07-26 (Joe, mid-cycle): detect-and-notify, not auto-run.** Everything below the
next paragraph describes what actually shipped. This unit originally specified an idempotent
AUTO-RUN design: `update` would run `prune --duplicates` itself, once, gated by a sentinel file
`<chunks-dir>/.dedup-applied`; a dry run would thread through without writing the sentinel; a
failure would be non-fatal and would not write the sentinel, so the next update retried. **That
design was never implemented or committed at any point** (verified: no commit in this repo's
history touches a `dedup-applied`/sentinel path for this feature) — Joe reversed it before any
code was written, verbatim: "let's not auto run... we need to detect duplicates so that we know
to tell the user." Reasoning, recorded authoritatively in ADR-0021 decision 9:
removing index files is a destructive, one-shot migration over a user's memory store, and unlike
#694's empty-file backlog (unconditionally safe — empty files hold zero records), a duplicate
removal's safety depends on the record-subset gate (Unit 3/4's addenda above) holding at the
moment it runs. A destructive migration should run only when the user asks for it, not silently
as a side effect of a routine binary/skills refresh. This reversal is recorded here as history —
the paragraph above and this one — precisely so a future reader does not mistake the sentinel/
auto-run shape for what to build; do not implement a sentinel file or an auto-run removal path
from anything in this unit.

**What shipped: stateless detection + a notice, matching the #694 empty-file convention exactly.**

- `chunkIndexHasDuplicates` (`internal/cli/update.go`) reads `manifest.json` once — no per-source
  `.jsonl` opens, so it stays cheap even against a large/decades-accreted index — groups it by
  `(FileHash, chunkingClass)` (`groupManifestByHash`, the same partition `prune --duplicates`
  uses, skipping entries already tagged `DuplicateOf`), and reports whether any group has ≥2 live
  members. A missing/unreadable/malformed manifest is treated as false (self-silencing for a
  fresh install, same convention as `chunkIndexHasEmptyFiles`) — a detection failure must never
  fail `engram update`'s primary job.
- `writeDuplicatesHint` prints one line when detected, silent otherwise: "duplicate chunk-index
  files found — run `engram prune --duplicates` to clear them; see the Upgrading section in
  README.md". No sentinel file: detection is stateless and idempotent by construction — it
  re-notifies every run while duplicates remain and goes quiet the run after a human actually
  clears them with `prune --duplicates`, so there is nothing to mark "done."
- `update` never calls `prune --duplicates` and never removes anything itself, under any flag
  combination including `--dry-run` — there is no removal path to gate on `--dry-run` in the
  first place, so the dry-run-safety concern the original design bullets raised is moot by
  construction rather than solved by threading a flag through.
- **Tests, as actually shipped** (`internal/cli/update_test.go`): `TestChunkIndexHasDuplicates`
  (a live duplicate group → true; distinct hashes, different chunking classes, or an
  already-`DuplicateOf`-tagged entry → false; malformed manifest → false, no panic) and
  `TestWriteUpdateReport_DuplicatesHint` (notice names `prune --duplicates`, "Upgrading", and
  "README.md"; coexists with the empty-file notice; a clean report contains no mention of
  "duplicate" at all). No sentinel test exists, because there is no sentinel.

The DI boundary is unchanged from the original intent: `internal/update` does not reach into
`internal/cli`. Detection lives entirely in `internal/cli` (`update.go`) and is surfaced to
`internal/update`'s `Report` only as an opaque `ChunkIndexHasDuplicates bool` field the `cli`
package sets after `Run` returns — mirroring `ChunkIndexHasEmptyFiles`'s existing convention
exactly, rather than composing a new cli→update dependency injection point as the auto-run design
would have needed.

See ADR-0021 decision 9 for the authoritative rationale record; this unit only narrates it.

## Cross-unit coordination

- Unit 2 removes `/tmp` roots from future sweeps; it does **not** remove already-indexed `/tmp`
  sources — Unit 4 does, if they duplicate a retained source. `/tmp` sources that are unique are
  left alone; removing those is not in this ask.
- Unit 3 and Unit 4 share `selectCanonical` and `removeDuplicateIndex`. Unit 3 lands first so
  Unit 4 consumes them rather than duplicating the logic.
- Deletion order in both: remove the index file first, then the manifest entry. The reverse
  order would strand an unreferenced index file that nothing cleans up (confirmed: nothing
  deletes a `.jsonl` with no manifest entry).

  **The crash window between the two is real and does NOT self-heal today** — Gate A refuted the
  first draft's claim that it does. `ingestSource`'s cheap-skip (`ingest.go:335-341`) compares
  only the *source file's* mtime+size against the manifest and never checks whether the
  `.jsonl` index file exists, so a crash that removed the index but not the entry leaves that
  source with no searchable content, silently and indefinitely, until something unrelated
  changes the source's mtime. `prune`'s default path tolerates it only by never looking
  (`prune.go:71` checks the source file, not the index).

  **Fix, in Unit 3:** extend the cheap-skip to also require the index file to exist — a stat per
  known source, negligible against the read it replaces, and it makes "the next ingest rebuilds
  it" true rather than aspirational. RED test `TestIngestRebuildsWhenIndexFileMissing`: manifest
  entry present, source unchanged, index file absent → index is rebuilt (fails today; the cheap
  skip returns early).

## Doc-surface disposition

Author-grepped: `prune`, `sweep`, `manifest`, `ingest --auto`, `append-only`, `never delete`,
and — added after Gate A found it missing, since Unit 5 was originally going to change what
`update` does (reversed — see Unit 5's shipped-vs-superseded note above; `update` ends up only
gaining a third notify-only line, not new authority) — `engram update`.

| File:line | Disposition |
| --- | --- |
| `README.md:44` | **update** — the notify-only upgrade convention for #694's empty files; Unit 5's shipped form (detect-and-notify, not the superseded auto-run) adds duplicates as a THIRD entry following the SAME notify-only convention — no contrast to make explicit, since update never acts for duplicates either |
| `README.md:37`, `docs/GLOSSARY.md:717` | **check, not update, as shipped** — both document `update --dry-run` as preview-only; they stay true as-is, since Unit 5's shipped form has no removal path to gate on `--dry-run` in the first place (moot by construction, not solved by threading a flag through, per the superseded-design bullets originally here) |
| `README.md:12,35-40,108` | **check** — general `update` descriptions; update only if they enumerate what update does step by step |
| `docs/GLOSSARY.md:605-612` | **update** — the `engram update` entry gains a third detect-and-notify line for the duplicate backlog (no one-shot behaviour, no sentinel — that design was reversed before shipping, see Unit 5) |
| `docs/architecture/adr.md` (ADR-0021) | **update** — ADR-0020 is the current highest, so ADR-0021 is free. It records the WHOLE decision cluster in one ADR: dedup by content hash + canonical precedence; the explicit narrowing of the cross-source append-only guarantee; the drifted-copy limitation; and — as shipped — update's detect-and-notify surface for the duplicate backlog, explicitly NOT a delete-on-the-user's-behalf authority (that auto-run design was reversed before it shipped; decision 9 records why) |
| `README.md:97` | **update** — the `ingest` row says "append-only … never deletes"; narrow per the invariant section above and add `--duplicates` to the `prune` row |
| `docs/GLOSSARY.md:550-551` | **update** — `engram ingest` entry's "(append-only chunk history)" narrowed |
| `docs/GLOSSARY.md` `engram prune` entry | **update** — currently says prune "KEEPS that source's per-source index file"; the new mode is the exception |
| `docs/GLOSSARY.md:519` | **keep** — "append-only trail" describes route evidence aggregates, unrelated |
| `docs/architecture/c1-system-context.md:194,223` | **update** — both state "never deleting existing records"; the `:223` sequence-diagram note is a diagram label |
| `docs/architecture/c2-containers.md` | **update** — manifest responsibilities gain dedup; prune flow gains the mode |
| `docs/architecture/c3-components.md` | **update** — prune component description; the diagram shows prune's behavior, so the new mode is added as a branch, not a rewrite of the base flow |
| `docs/FEATURES.md` | **update** — new capability entry with `why:` (ADR-0021) and `validation:` (LEDGER row from this cycle) |
| `docs/ROADMAP.md` | **update** — Provenance row on ship; verify no band row claims this open |
| `agent-instructions/skills/learn/SKILL.md:36` | **update** — "existing chunks are never deleted (append-only history)" is the same over-broad claim, in deployed guidance |
| `internal/cli/ingest.go:464-465,476,584` | **keep** — these describe `mergeChunkRecords`' within-source guarantee, which is genuinely unchanged. Re-read post-change for newly-misleading omission (note 383) |
| `internal/cli/prune.go:31-39` | **update** — `RunPrune`'s doc comment states the keep-index rationale; add the dedup exception |
| `agent-instructions/skills/{recall,please}/SKILL.md`, `*/tests/*.md` | **keep** — they invoke `engram ingest --auto`; interface unchanged |
| `dev/eval/LEDGER.md`, `dev/eval/atoms*/**`, `docs/design/2026-07-01-*.md` | **keep** — vintage records, eval fixtures, historical design |

## Gates

A (this plan): all four angles, one round complete — this revision addresses every finding.
B: after each unit's refactor; Unit 4's Gate B reviewer additionally performs the live copy-run.
C: every doc above. D: commit messages.

Commits: one per unit, `AI-Used: [claude]`, ff-only main, `targ check-full` green before each.

## Rebutted finding

Gate A's clarity reviewer asked the plan to state expected deletion counts up front (62,397 →
13,102) and check against them. **Rejected**: note 178's measured lesson is that a
planning-phase count is stale by execution time (that cycle's own work drifted the number and a
count-check would have false-STOPped), and Gate A's own recount already drifted by 18 sources
within one session. The plan instead requires mechanical re-derivation at run time plus a
keep-list complement, which is strictly stronger. All of that reviewer's other findings are
adopted above.
