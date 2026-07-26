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

## Unit 5 — `engram update` runs the dedup prune once (Joe, mid-cycle addition)

Requested verbatim: "We also need the update command to run the prune once for people updating
to this version from an old one that might have accumulated copies." Without it, only people who read
the release notes and run `prune --duplicates` by hand ever get the cleanup — everyone else
keeps the accumulated copies forever, which is the whole population this cycle is for.

**Divergence from precedent, stated once and then committed to.** The existing upgrade
convention *notifies* rather than acts: for #694's empty files, `engram update` "prints a
one-line notice pointing here" and the user runs `engram prune --empty` themselves
(`README.md:44`). Unit 5 has update perform the deletion instead. That is defensible here —
the removal is safe by construction (only a duplicate whose byte-identical twin is verifiably
retained), and unlike the empty-file case there is no reason a user would want the copies —
but it is a genuine change in what `update` is allowed to do, and it is Joe's call, made
explicitly.

- **Once, not every run.** Idempotence marker: a sentinel file `<chunks-dir>/.dedup-applied`
  containing the schema tag and the ISO date. Present → skip entirely (no manifest read, so no
  cost on the common path). Absent → run the Unit 4 removal, then write it. A sentinel file
  rather than a manifest key because `ingestManifest` is a bare `map[string]manifestEntry` and
  adding a metadata key would change its type and break every existing reader.
- **Loud, never silent.** Report the removed and retained counts through the existing update
  report (`internal/update/update.go`'s `Report`), in the same style as the harness sections —
  a deletion the user did not ask for individually must be visible in the output they do read.
- **`engram update --dry-run` must preview, never delete.** `update` has its own `--dry-run`
  flag, documented as "show what would change" (`README.md:37`) and "previews without writing"
  (`docs/GLOSSARY.md:717`). Unit 5 must thread it through to the dedup pass and must not write
  the sentinel on a dry run — otherwise the first `update --dry-run` after upgrading would
  silently delete thousands of index files, which is the exact opposite of what that flag
  promises. RED test: `update --dry-run` with duplicates present → removal not called, sentinel
  not written, report still names what would be removed.
- **Failure is non-fatal.** A dedup failure must not fail `engram update`'s primary job
  (binary + skills refresh). On error: report it, do NOT write the sentinel (so the next update
  retries), and continue. Exit code follows the update's own result.
- **RED** (`internal/update/update_test.go`, memFS): sentinel absent + a manifest with a
  duplicate group → removal called once, sentinel written, counts in the report; sentinel
  present → removal never called; removal returns an error → sentinel NOT written, update still
  succeeds.
- **GREEN + REFACTOR + Gate B.**
- **VERIFY:** installed binary, `engram update` against a **copy** of the chunks dir — first run
  removes and writes the sentinel, second run reports nothing and leaves the index byte-identical.

Note the DI boundary: `internal/update` must not reach into `internal/cli` directly. Wire the
dedup entry point as an injected function on the updater's deps, composed in `cli.NewDeps`
alongside the existing update wiring, so the dependency direction stays cli → update.

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
and — added after Gate A found it missing, since Unit 5 changes what `update` does — `engram
update`.

| File:line | Disposition |
| --- | --- |
| `README.md:44` | **update** — the notify-only upgrade convention for #694's empty files; Unit 5 makes update *act* for duplicates, so this section gains the dedup entry and the contrast is made explicit rather than left as a silent inconsistency |
| `README.md:37`, `docs/GLOSSARY.md:717` | **update** — both document `update --dry-run` as preview-only; they stay true only because Unit 5 threads the flag through, and should say the dedup pass honours it |
| `README.md:12,35-40,108` | **check** — general `update` descriptions; update only if they enumerate what update does step by step |
| `docs/GLOSSARY.md:605-612` | **update** — the `engram update` entry gains the one-shot dedup behaviour and the sentinel |
| `docs/architecture/adr.md` (ADR-0021) | **update** — ADR-0020 is the current highest, so ADR-0021 is free. It records the WHOLE decision cluster in one ADR: dedup by content hash + canonical precedence; the explicit narrowing of the cross-source append-only guarantee; the drifted-copy limitation; and update's new authority to delete on the user's behalf — which is only safe because of the twin-retained guarantee, so the two belong together rather than in separate ADRs |
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
