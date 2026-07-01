# Concurrency & write-safety (Track 0) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development (or executing-plans).
> Steps use `- [ ]`. Gate B/C/D markers are run by the `/please` orchestrator.

**Goal:** Make engram's vault + chunk-index writes concurrency-safe — fix the #660 manifest lost-update/torn
write, the `amend` lost-update, and sidecar torn-writes — so parallel `engram ingest`/`amend`/`activate` runs
(and, later, the payload-prune sub-recalls) cannot corrupt or silently drop state.

**Architecture:** Two mechanisms, both following patterns already in the repo.
1. **Atomic writes** — a shared `atomicWriteFile` helper (write a unique temp file in the target dir, then
   `os.Rename` — atomic on POSIX same-dir) replaces `os.WriteFile` at the vault/index write edges. Kills
   torn/partial files (the "non-atomic write" half of #660 + the sidecar bug).
2. **Cross-process flock on read-modify-write regions** — generalize the existing flock helper
   (`osLearnFS.Lock`, `cli.go:54-79`), add a `Lock` dep to `IngestDeps` (manifest, `<chunksDir>/.manifest.lock`)
   and `AmendDeps` (notes, reuse `vault/.luhmann.lock`), and wrap the RMW regions. Kills lost updates. `learn`
   already does exactly this (`writeLearnUnderLock`, `learn.go:565-571`) — the precedent to mirror.

**Tech Stack:** Go (`internal/cli/`, `internal/embed/`); `syscall.Flock` (already used); tests follow the
existing file's idiom — **plain inline-closure deps** as in `amend_test.go` (NOT imptest; those files don't use
it), Gomega assertions, `t.TempDir()`; `targ` for test/lint (`targ test`, `targ check-full`); real-binary
verify via `go install ./cmd/engram`.

## Global Constraints
- `targ` for all Go test/lint — NEVER `go test`/`go vet`. Binary install is the one exception:
  `go install ./cmd/engram` (targ has no build target).
- DI everywhere: no `os.*`/`syscall.*` in `Run*` business logic — I/O and locking are injected via `*Deps`
  closures, wired at the edges (`newOs*Deps`). `atomicWriteFile` and the flock helper live among the edge
  adapters (`cli.go`), which are allowed to touch `os.*`/`syscall.*`.
- nilaway/gomega guards per `.claude/rules/go.md` (nil-check after `Expect(err).NotTo(HaveOccurred())`);
  line length < 120; descriptive names; `t.Parallel()` with no shared mutable state; `make([]T,0,cap)` when
  size known; wrap errors `fmt.Errorf("...: %w", err)`.
- Commit trailer `AI-Used: [claude]`.

---

## Task 1: Shared `atomicWriteFile` helper + route vault/index write edges through it

**Files:** Create `internal/cli/writesafe.go` + `internal/cli/writesafe_test.go`; modify the write edges
`internal/cli/embed.go:163`, `internal/cli/activate.go:110`, `internal/cli/amend.go:345`,
`internal/cli/resituate.go:116`, `internal/cli/cli.go:166`, `internal/cli/migrate.go:96`, and the manifest
write edge wired into `IngestDeps.WriteFile` (find in `newOsIngestDeps`).

**Interfaces:**
- Produces: `func atomicWriteFile(path string, data []byte, perm os.FileMode) error` — writes `data` to a
  unique temp file in `filepath.Dir(path)` via `os.CreateTemp(dir, "."+base+".tmp-*")`, `Chmod(perm)`,
  `Write`, `Close`, then `os.Rename(tmp, path)`; on any error removes the temp and returns wrapped. Same-dir
  rename guarantees atomicity and that a concurrent reader sees either the old or the new file, never a torn one.

- [ ] **Step 1 — RED test.** In `writesafe_test.go` add `TestAtomicWriteFile` (Gomega, `t.Parallel()`,
  `t.TempDir()`): (a) writes new content and the file reads back exactly; (b) overwrites existing content
  atomically; (c) leaves **no** leftover `.tmp-*` file in the dir after success (glob the dir); (d) on a
  rename/write failure (inject by pointing at a path whose dir is read-only) the original file is untouched and
  no temp remains. Write assertions that call `atomicWriteFile` (which does not yet exist → compile-fail RED).
- [ ] **Step 2 — Run RED.** `targ test` → fails to compile (undefined `atomicWriteFile`).
- [ ] **Step 3 — GREEN.** Implement `atomicWriteFile` in `writesafe.go` per the interface. Then replace the
  `os.WriteFile(path, data, perm)` call at each **vault/index** edge listed in Files with
  `atomicWriteFile(path, data, perm)`. **Do NOT touch** `internal/embed/cache.go` or `internal/embed/hugot.go`
  (model-asset cache, single-writer at model-load, out of scope — note this exclusion in the commit).
- [ ] **Step 4 — Run GREEN.** `targ test` → `TestAtomicWriteFile` + full suite pass.
- [ ] **Step 5 — REFACTOR + Gate B.** Confirm ONE helper, all edges call it, no duplicated temp-rename logic;
  the helper is the only new `os.CreateTemp`/`os.Rename` site. Hand the diff to Gate B (design-fit).

## Task 2: flock the manifest read-modify-write — `ingest` AND `prune` (#660)

**Files:** Modify `internal/cli/cli.go` (generalize the flock helper), `internal/cli/ingest.go` (add
`IngestDeps.Lock`, wrap the RMW, wire in `newOsIngestDeps`), `internal/cli/prune.go` (add `PruneDeps.Lock`,
wrap the RMW, wire in `newOsPruneDeps`); Test `internal/cli/ingest_test.go` + `internal/cli/prune_test.go`.

**Both `RunIngest` (`ingest.go:82`→`:108-110`) and `RunPrune` (`prune.go:31`→`:73`) read `chunks/manifest.json`,
mutate it, and write it whole back with NO lock — the identical #660 lost-update hazard** (a concurrent
`ingest`+`prune` on the same `chunksDir` drops one side's changes). Both take the same
`<chunksDir>/.manifest.lock`. (Found by Gate-A code-alignment: the original plan missed `prune`.)

**Interfaces:**
- Produces: `IngestDeps.Lock` and `PruneDeps.Lock`, both `func(chunksDir string) (release func(), err error)`
  wired to flock `<chunksDir>/.manifest.lock`.

- [ ] **Step 1 — RED test.** Add `TestRunIngest_LocksManifestAroundReadModifyWrite` (plain-closure deps,
  Gomega): record a call-order slice — `Lock`→`"lock"`, `ReadFile`→`"read:"+path`, `WriteFile`→`"write:"+path`,
  release→`"unlock"`. Seed one changed source. Assert order `lock` → manifest `read` → manifest `write` →
  `unlock`. Add the mirror `TestRunPrune_LocksManifestAroundReadModifyWrite` (seed a dead source so a manifest
  write happens). Today `Lock` is never called → RED.
- [ ] **Step 2 — Run RED.** `targ test` → both fail (`IngestDeps`/`PruneDeps` have no `Lock`).
- [ ] **Step 3 — GREEN.**
  (a) In `cli.go`, extract `func flockPath(lockPath string) (func(), error)` holding the current
  `OpenFile`+`Flock(LOCK_EX)`+release logic; rewrite `osLearnFS.Lock(vault)` to
  `return flockPath(filepath.Join(vault, luhmannLockFile))`. Add `const manifestLockFile = ".manifest.lock"`.
  (b) Add `Lock func(chunksDir string) (func(), error)` to `IngestDeps` and `PruneDeps`; wire both in their
  `newOs*Deps` to `func(dir string) (func(), error) { return flockPath(filepath.Join(dir, manifestLockFile)) }`.
  (c) In `RunIngest` (after `gatherSources`) and `RunPrune` (before `readManifest` at `prune.go:31`), acquire
  the lock and `defer release()` around the whole manifest RMW.
- [ ] **Step 4 — Run GREEN.** `targ test` → both order tests + full suite pass.
- [ ] **Step 5 — Concurrent regression test.** Add `TestManifest_ConcurrentWritersDoNotLoseEntries` (real flock
  via `flockPath` on a `t.TempDir()` lockfile): two goroutines — one ingesting a NEW source, one pruning a dead
  source — run concurrently on the SAME `chunksDir` (the deliberate shared subject; each goroutine otherwise owns
  its args), each closure sleeping ~5ms between read and write to widen the window; assert the final manifest
  RETAINS the ingested entry AND drops only the dead one. Deterministic with the lock; document that the pre-fix
  code loses the ingest entry.
- [ ] **Step 6 — Gate B** on the diff (design-fit: minimal region, both writers, mirrors `learn`).

## Task 3: flock the vault note read-modify-write — `amend` AND `resituate`

**Files:** Modify `internal/cli/amend.go` (add `AmendDeps.Lock`, wrap RMW, wire in `newOsAmendDeps`),
`internal/cli/resituate.go` (add `ResituateDeps.Lock`, wrap RMW, wire in `newOsResituateDeps`); Test
`internal/cli/amend_test.go` + `internal/cli/resituate_test.go`.

**Both `RunAmend` (`amend.go:80/95/100`) and `RunResituate` (`resituate.go:55/65/70`) read a note, transform it
in memory, and write it back (plus a sidecar) with NO lock — the identical lost-update hazard.** Both take the
SAME `vault/.luhmann.lock` `learn` uses, so amend/resituate/learn serialize against each other on the vault.
(Found by Gate-A code-alignment: the original plan missed `resituate`.)

**Interfaces:**
- Produces: `AmendDeps.Lock` and `ResituateDeps.Lock`, both `func(vault string) (release func(), err error)`
  wired to `vaultFS.Lock`.

- [ ] **Step 1 — RED test.** Add `TestRunAmend_LocksVaultAroundReadModifyWrite` and
  `TestRunResituate_LocksVaultAroundReadModifyWrite` (plain closures): record order — `Lock`→`"lock"`,
  `Read`→`"read"`, `Write`→`"write"`, release→`"unlock"`. Seed a note + the relevant flag (amend: `--relation`;
  resituate: `--situation`). Assert `lock` precedes `read` and `unlock` follows the final `write`. Today `Lock`
  is never called → RED.
- [ ] **Step 2 — Run RED.** `targ test` → both fail (`AmendDeps`/`ResituateDeps` have no `Lock`).
- [ ] **Step 3 — GREEN.** Add `Lock func(vault string) (func(), error)` to `AmendDeps` and `ResituateDeps`;
  wire `Lock: vaultFS.Lock` in both `newOs*Deps` (mirror `learn.go:285`). In `RunAmend` (before `deps.Scan`) and
  `RunResituate` (before `deps.Read` at `resituate.go:55`), acquire the lock and `defer release()`, covering the
  whole read → transform → write (+ sidecar) region.
- [ ] **Step 4 — Run GREEN.** `targ test` → both order tests + full suite pass.
- [ ] **Step 5 — Gate B** on the diff.

## Task 4: flock `activate`, and correct the false-atomicity comment

**Files:** Modify `internal/cli/activate.go` (add `ActivateDeps.Lock`, wrap the bump loop in `RunActivate`, wire
in `newOsActivateDeps`; correct the comment); Test `internal/cli/activate_test.go`.

`RunActivate` (`activate.go:30`) loops `bumpLastUsed`, which reads a sidecar, sets `LastUsed`, and rewrites the
WHOLE sidecar (preserving `Vectors`/`ContentHash` from the read). Unlocked, a standalone `activate` that reads a
sidecar just before a concurrent (now-flocked) `amend`/`resituate` re-embeds it, then writes after, **clobbers
the freshly-written vectors with stale ones** — a lost update of the embedding, not benign metadata. So
`activate` must take the SAME `vault/.luhmann.lock`. `ActivateArgs` already carries `Vault` (`activate.go:15`),
so this is clean. (Found by Gate-A docs-alignment: the C1 flow diagram + my vector-clobber analysis.)

**Deadlock-avoidance invariant (applies to Tasks 2-4):** acquire the flock ONLY at the `Run*` entry point.
Shared write helpers (`bumpLastUsed`, `writeManifestFile`, `reEmbedAndActivate`, `writeAmendedSidecar`) MUST
NOT acquire it — `RunAmend` already holds the flock when it calls `reEmbedAndActivate`→`bumpLastUsed`, so a
helper re-acquiring the same lock on a second fd would self-deadlock (flock is per-open-file-description).

- [ ] **Step 1 — RED test.** Add `TestRunActivate_LocksVaultAroundBumpLoop` (plain closures): record order —
  `Lock`→`"lock"`, `Read`→`"read"`, `Write`→`"write"`, release→`"unlock"`. Seed one note whose sidecar needs a
  bump (`LastUsed` != today). Assert `lock` precedes the first `read` and `unlock` follows the last `write`.
  Today `Lock` is never called → RED.
- [ ] **Step 2 — Run RED.** `targ test` → fails (`ActivateDeps` has no `Lock`).
- [ ] **Step 3 — GREEN.** Add `Lock func(vault string) (func(), error)` to `ActivateDeps`; wire
  `Lock: vaultFS.Lock` in `newOsActivateDeps`. In `RunActivate`, acquire `deps.Lock(args.Vault)` before the
  loop and `defer release()`; keep `bumpLastUsed` lock-free (per the invariant). Replace the false comment at
  `activate.go:66-67` ("No lock: sidecar writes are atomic per-file") with the truth: sidecar writes go through
  `atomicWriteFile` (temp+rename) AND `RunActivate` holds the vault flock, so a concurrent amend/resituate
  re-embed cannot be clobbered.
- [ ] **Step 4 — Run GREEN.** `targ test` → order test + full suite pass.
- [ ] **Step 5 — Gate B** on the diff.

## Task 5: Verify with the real binary + full check

- [ ] **Step 1 — `go install ./cmd/engram`** (NOT `targ build`).
- [ ] **Step 2 — Real concurrent smoke (throwaway dirs — do NOT touch the live vault/chunks).**
```bash
V="$(mktemp -d)/vault"; C="$(mktemp -d)/chunks"; mkdir -p "$V" "$C"
printf -- '---\ntype: fact\n---\nseed\n' > "$V/1.seed.md"
# two concurrent ingests of the same throwaway sources must not corrupt manifest.json:
ENGRAM_VAULT_PATH="$V" ENGRAM_CHUNKS_DIR="$C" engram ingest --auto &
ENGRAM_VAULT_PATH="$V" ENGRAM_CHUNKS_DIR="$C" engram ingest --auto &
wait
python3 -c "import json;json.load(open('$C/manifest.json'));print('manifest.json parses OK')"
```
- [ ] **Step 3 — `targ check-full`** green (lint + coverage + nils + uncommitted).
- [ ] **Step 4 — Commit** (Gate D over the message):
```
fix(cli): make vault + chunk-index writes concurrency-safe (#660)

Atomic temp-rename writes at the vault/index edges + extend the existing
vault flock (previously learn-only) to every read-modify-write writer:
manifest (ingest + prune) and vault notes/sidecars (amend + resituate +
activate). Fixes #660 (concurrent-ingest manifest corruption) plus the
prune/amend/resituate lost-updates and the activate sidecar vector-clobber
found in Gate A. Flock is acquired only at Run* entry points (shared write
helpers stay lock-free) to avoid nested-flock self-deadlock. Model-asset
cache writes (internal/embed) are single-writer and left unchanged.

AI-Used: [claude]
```

## Self-review (writing-plans checklist)
- **Coverage:** the complete write-safety surface (established via the Step-2 code map + Gate-A code-alignment)
  is: atomic-write helper for torn writes = Task 1 (all edges); manifest RMW flock = Task 2 (`ingest` +
  `prune`); vault-note RMW flock = Task 3 (`amend` + `resituate`); vault-sidecar RMW flock = Task 4
  (`activate`); `learn` already flock-safe. Real-binary verify = Task 5. Every RMW writer in the CLI is covered.
- **Type consistency:** all `Lock func(string)(func(),error)` match the existing `LearnDeps.Lock` signature
  (`learn.go:66`); `atomicWriteFile(path,data,perm)` mirrors the `os.WriteFile` signature it replaces.
- **Scope:** vault/index writes only; model-asset cache excluded (named); the payload-prune build stays
  deferred (separate spec). DRY: one `atomicWriteFile` helper, one `flockPath` helper shared by the vault
  (`.luhmann.lock`) and manifest (`.manifest.lock`) locks.
- **Deadlock:** flock only at `Run*` entry points; shared write helpers never acquire (verified no re-entrancy
  in Gate A).
- **Issue tracking:** #660 covers the manifest RMW (ingest + prune). The vault-write lost-updates
  (amend/resituate/activate) are untracked today — a single umbrella issue is filed at completion (Step 6) and
  closed by this work; the ROADMAP Track-0 entry is updated with the numbers.
- **Test idiom:** plain inline closures (the `amend_test.go` pattern), order-recording for lock interaction,
  a real-flock concurrent regression test for the manifest.
