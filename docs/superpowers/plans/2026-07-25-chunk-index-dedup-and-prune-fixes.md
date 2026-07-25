# Plan: chunk-index deduplication + two ingest/prune bugfixes

Cycle: /please, 2026-07-25. Ask (verbatim): "fix the two bugs you found, as well as
implementing the full general fix".

The two bugs and the general fix all live in the same subsystem — `--auto` source selection and
the manifest — so they ship as one cycle, three units, separately committed.

## Measured starting state (re-derive at execution time, note 178)

| Measure | Value | Command |
| --- | --- | --- |
| manifest sources | 62,397 | `python3` over `chunks/manifest.json` |
| sources that are exact-content duplicates of another source | **49,295 (79%)** | group manifest entries by `file_hash`, sum `len(group)-1` |
| … still present on disk | 48,952 | same, plus `os.path.exists` |
| redundant chunk *records* | ~11,307 (15.3% of 73,945) | prior investigation |
| duplicate share of returned items, production arm | 9.6% mean / 28.6% worst | `engram query --lazy-chunks` |
| scan cost of the junk | −290 ms of ~5.1 s (−5.7%) | job-free hardlinked index, 3 runs |

**Honest bound on the general fix.** Exact-content dedup reaches the 79%. It does NOT reach a
*drifted* copy — a snapshot of a note that has since been amended hashes differently, so both
survive. Measured: 50 of 256 shadowed vault notes have drifted. Unit 3 therefore carries a
second rule for that class (a `.md` with a sibling `.vec.json` is a vault note, and only the
canonical vault is indexed), and the plan states plainly what neither rule covers.

## Assessment of the ask

Sound, and the general fix is the right call over the one-word `jobs` exclude: `jobs` is 29% of
the problem, and a path-pattern blocklist cannot reach worktrees, backup-restore trees, or
`/tmp` config pools, which are the majority. Two challenges recorded, neither blocking:

1. **Deletion of existing index files is destructive and one-shot.** Notes 241/178/25 apply:
   gate every delete on the survivor verifiably existing, re-derive the set against the live
   tree at execution time, and never delete on a frozen inventory. Unit 4 is therefore
   dry-run-first with an explicit count check, and the reviewer live-runs it against a full
   copy of the real chunks dir.
2. **`--auto`'s ancestor sweep is intentional**, not itself the bug — it is how engram finds the
   user's `~/.claude` memory (vault note 296 records this exact mechanism being diagnosed on
   2026-07-19 in an eval-isolation context). The fix must dedup, not stop sweeping.

## Units

### Unit 1 — `engram prune --dry-run` is ignored on the default path (bug)

`RunPrune` (`internal/cli/prune.go:54-98`) drops dead manifest entries and calls
`deps.WriteFile` unconditionally; `args.DryRun` is read only inside `pruneEmptyLocked`
(`:138,154`). So `engram prune --dry-run` silently mutates the manifest today.

- **RED:** table test — `RunPrune` with `DryRun: true`, a manifest containing a dead source,
  and a `WriteFile` stub that fails the test if called. Expect the current code to call it.
- **GREEN:** guard the write on `!args.DryRun`; prefix the report line `[dry-run] ` exactly as
  `pruneEmptyLocked` does.
- **VERIFY:** `targ test`; then the real binary against a **copy** of the chunks dir — run
  `engram prune --dry-run --chunks-dir <copy>` and confirm `manifest.json`'s mtime and bytes
  are unchanged.

### Unit 2 — non-persistent prefixes apply only to session logs (bug)

`NonPersistentPrefixes` is wired onto the SessionLogs root alone
(`internal/cli/sweepspec.go:133-140`). A sweep root whose **own path** is under a throwaway
prefix (`/tmp`, `/private/tmp`, `/var/folders`) is still swept and permanently indexed — which
is how `/tmp/cummatrix-*/cfgpool/…` markdown reached the index.

- **RED:** `ResolveSweepRoots` test with `env.Cwd` under `/private/tmp/...` — assert no root is
  returned whose path is under a non-persistent prefix. Fails today.
- **GREEN:** filter resolved roots by their own path prefix. Explicit `--sweep`/`--transcript`
  keep bypassing this (`ingest.go:177-182`), which is the documented escape hatch.
- **VERIFY:** `targ test`; real binary run from a `/private/tmp` cwd adds no sources.

Note the prefix forms differ: `NonPersistentPrefixes` today holds *slugified* project-dir
names (`-private-tmp-`). Unit 2 needs real path prefixes; add a sibling field rather than
overloading the existing one, and say so in the doc surface.

### Unit 3 — dedup at ingest (the general fix, forward-looking)

The manifest already stores `FileHash` per source (`ingest.go:344-361`), so the mechanism is
available without new state.

- **Rule A (exact content):** when a swept source's content hash matches a source already
  indexed **and still present on disk**, skip building a second index for it. Winner must be
  deterministic and stable across runs regardless of walk order — pick by an explicit
  precedence (explicit flags > repo root > `.claude`/`.pi` ancestors, then shortest path, then
  lexicographic), not by first-seen.
- **Rule B (vault copies):** a `.md` with a sibling `<name>.vec.json` is a vault note; index it
  only from the canonical vault path. This is what catches the 50 drifted snapshots Rule A
  cannot.
- **RED:** unit tests over `gatherSources`/`ingestSource` with mocked FS — two sources, same
  bytes, different paths → exactly one index built, and the winner is the precedence-correct
  one; a drifted vault copy → not indexed.
- **GREEN:** minimal implementation behind the existing DI seams.
- **REFACTOR + Gate B.**
- **VERIFY:** `targ check-full`; then the real binary on a **copy** of the live chunks dir —
  `engram ingest --auto` adds no new duplicate sources, and a chunk-query spot check returns
  the same live notes.

### Unit 4 — `engram prune --duplicates` (the general fix, retroactive)

Cleans what is already there; without it the 49,295 existing duplicates persist.

- Selects manifest sources whose `file_hash` matches a retained source, removes their index
  files and manifest entries, keeps exactly one per hash by the Unit 3 precedence.
- **Deletion safety (notes 241/178):** re-derive the set against the live manifest at run time;
  gate each delete on the retained twin existing *now*; per-item failure set with a non-zero
  exit naming failures; convergent re-run (a second run removes nothing). `--dry-run` must work
  here from the start — Unit 1 makes that real.
- **VERIFY:** reviewer live-runs `--dry-run` then the real delete against a **full copy** of the
  chunks dir, with expected counts stated in advance and zero warnings, before it is ever run
  against the live index. The live index is regenerable but re-embedding costs ~5 min/455 notes.

## Doc-surface disposition (author-grepped: `prune`, `sweep`, `manifest`, `ingest --auto`)

| File | Hits | Disposition |
| --- | --- | --- |
| `README.md` | prune 2, sweep 3, manifest 1 | **update** — CLI reference gains `--duplicates`; `--dry-run` now real on the default path |
| `docs/FEATURES.md` | prune 8, sweep 4, manifest 3 | **update** — dedup is a new user-visible capability; needs an entry with `why:`/`validation:` |
| `docs/GLOSSARY.md` | sweep 6, manifest 5, prune 1 | **update** — define the dedup rule and the canonical-source precedence |
| `docs/architecture/c2-containers.md` | manifest 13, sweep 5, prune 3 | **update** — the manifest's role gains a dedup responsibility |
| `docs/architecture/c3-components.md` | prune 7, manifest 4 | **update** — prune's component description gains the new mode |
| `docs/architecture/adr.md` | prune 5, manifest 3 | **update** — a new ADR: dedup by content hash + canonical-source precedence, with the drifted-copy limitation recorded |
| `docs/architecture/c1-system-context.md` | sweep/ingest flow | **check** — update only if the ingest flow diagram names the source-selection step |
| `docs/ROADMAP.md` | prune 4, sweep 7 | **update** — Provenance row once shipped; check no band row claims this is open |
| `agent-instructions/skills/{recall,learn}/SKILL.md`, `please/SKILL.md` | sweep/ingest mentions | **keep** — they invoke `engram ingest --auto`, whose interface is unchanged. Re-check post-change for newly-misleading omissions (note 383) |
| `agent-instructions/skills/*/tests/*.md` | sweep mentions | **keep** — skill baselines; behavior under test unchanged |
| `dev/eval/LEDGER.md`, `dev/eval/atoms*/**` | many | **keep** — vintage records and eval fixtures |
| `docs/design/2026-07-01-engram-recall-subprocess-design.md` | sweep | **check** — historical design doc; update only if it states current source-selection behavior as live |

## Gates

A (this plan): all four angles. B: after each unit's refactor. C: every doc above that ends up
touched. D: commit messages.

Commits: one per unit, `AI-Used: [claude]`, ff-only main. `targ check-full` green before each.
