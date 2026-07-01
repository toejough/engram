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
  no temp remains. Assert against `atomicWriteFile` which does not yet exist.
- [ ] **Step 2 — Run RED.** `targ test` → fails to compile (undefined `atomicWriteFile`).
- [ ] **Step 3 — GREEN.** Implement `atomicWriteFile` in `writesafe.go` per the interface. Then replace the
  `os.WriteFile(path, data, perm)` call at each **vault/index** edge listed in Files with
  `atomicWriteFile(path, data, perm)`. **Do NOT touch** `internal/embed/cache.go` or `internal/embed/hugot.go`
  (model-asset cache, single-writer at model-load, out of scope — note this exclusion in the commit).
- [ ] **Step 4 — Run GREEN.** `targ test` → `TestAtomicWriteFile` + full suite pass.
- [ ] **Step 5 — REFACTOR + Gate B.** Confirm ONE helper, all edges call it, no duplicated temp-rename logic;
  the helper is the only new `os.CreateTemp`/`os.Rename` site. Hand the diff to Gate B (design-fit).

## Task 2: #660 — flock the ingest manifest read-modify-write

**Files:** Modify `internal/cli/cli.go` (generalize the flock helper), `internal/cli/ingest.go` (add
`IngestDeps.Lock`, wrap the RMW, wire in `newOsIngestDeps`); Test `internal/cli/ingest_test.go`.

**Interfaces:**
- Consumes: the generalized flock.
- Produces: `IngestDeps.Lock func(chunksDir string) (release func(), err error)` wired to flock
  `<chunksDir>/.manifest.lock`.

- [ ] **Step 1 — RED test.** Add `TestRunIngest_LocksManifestAroundReadModifyWrite` (plain-closure deps,
  Gomega): record a call-order slice from closures — `Lock` appends `"lock"`, `ReadFile` appends `"read:"+path`,
  `WriteFile` appends `"write:"+path`, and the release func appends `"unlock"`. Seed one changed source so a
  manifest write happens. Assert the order is `lock` → manifest `read` → manifest `write` → `unlock` (lock
  before the manifest read, unlock after the manifest write). Today `Lock` is never called → RED.
- [ ] **Step 2 — Run RED.** `targ test` → fails (`IngestDeps` has no `Lock`; order assertion fails).
- [ ] **Step 3 — GREEN.**
  (a) In `cli.go`, extract `func flockPath(lockPath string) (func(), error)` holding the current
  `OpenFile`+`Flock(LOCK_EX)`+release logic; rewrite `osLearnFS.Lock(vault)` to
  `return flockPath(filepath.Join(vault, luhmannLockFile))`.
  (b) Add `Lock func(chunksDir string) (func(), error)` to `IngestDeps`; wire it in `newOsIngestDeps` to
  `func(dir string) (func(), error) { return flockPath(filepath.Join(dir, manifestLockFile)) }` with a new
  `const manifestLockFile = ".manifest.lock"`.
  (c) In `RunIngest`, immediately after `gatherSources` succeeds, acquire the lock and `defer release()`:
  `release, lockErr := deps.Lock(args.ChunksDir); if lockErr != nil { return fmt.Errorf("ingest: lock: %w",
  lockErr) }; defer release()`. This serializes the whole manifest RMW (read at :82 → per-source index writes →
  write at :109) — the safest region.
- [ ] **Step 4 — Run GREEN.** `targ test` → the order test + full suite pass.
- [ ] **Step 5 — Concurrent regression test.** Add `TestRunIngest_ConcurrentWritersDoNotLoseEntries` (real
  flock via `flockPath` on a `t.TempDir()` lockfile; NOT parallel-shared-state — each goroutine gets its own
  args/source): two goroutines ingest two **distinct** sources concurrently, each closure sleeping ~5ms between
  read and write to widen the window; assert the final manifest (via a real `readManifest`) contains **both**
  entries. With the lock GREEN this is deterministic; document that without it the pre-fix code loses one.
- [ ] **Step 6 — Gate B** on the diff (design-fit: is the flock region minimal + mirroring `learn`?).

## Task 3: `amend` — flock the read-modify-write, reusing the vault lock

**Files:** Modify `internal/cli/amend.go` (add `AmendDeps.Lock`, wrap RMW, wire in `newOsAmendDeps`); Test
`internal/cli/amend_test.go`.

**Interfaces:**
- Produces: `AmendDeps.Lock func(vault string) (release func(), err error)` wired to `vaultFS.Lock` (the SAME
  `.luhmann.lock` `learn` uses — so `amend` serializes against `learn` and other amends on the vault).

- [ ] **Step 1 — RED test.** Add `TestRunAmend_LocksVaultAroundReadModifyWrite` (plain closures): record
  order — `Lock`→`"lock"`, `Read`→`"read"`, `Write`→`"write"`, release→`"unlock"`. Seed a note to amend + one
  `--relation`. Assert `lock` precedes `read` and `unlock` follows `write`. Today `Lock` is never called → RED.
- [ ] **Step 2 — Run RED.** `targ test` → fails (`AmendDeps` has no `Lock`).
- [ ] **Step 3 — GREEN.** Add `Lock func(vault string) (func(), error)` to `AmendDeps`; wire `Lock: vaultFS.Lock`
  in `newOsAmendDeps` (mirror `learn.go:285`). In `RunAmend`, acquire the lock at the top (before `deps.Scan`)
  and `defer release()`, so the whole `Scan`/`Read`(:80) → `amendContent`(:95) → `Write`(:100) →
  `reEmbedAndActivate`(:105) region is serialized.
- [ ] **Step 4 — Run GREEN.** `targ test` → order test + full suite pass.
- [ ] **Step 5 — Gate B** on the diff.

## Task 4: Sidecar — correct the false-atomicity comment; confirm coverage

**Files:** Modify `internal/cli/activate.go` (the `bumpLastUsed` doc comment).

- [ ] **Step 1 — GREEN (doc-only, no behavior).** Task 1 already routed `activate.go:110`'s write through
  `atomicWriteFile`, so the sidecar torn-write is fixed. Replace the false comment at `activate.go:65-66`
  ("sidecar writes are atomic per-file") with the truth: writes go through `atomicWriteFile` (temp+rename);
  the LastUsed bump is idempotent metadata, so a concurrent bump losing a race is benign (worst case one
  recency stamp is lost, never corruption) — it deliberately stays outside the vault flock to avoid serializing
  activation. No RED (comment correction on already-passing code — Iron Law: no test-gaming a doc fix).
- [ ] **Step 2 — Verify.** `targ test` still green (no behavior change).

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
vault flock to ingest-manifest and amend read-modify-write regions.
Fixes #660 (concurrent ingest manifest corruption), the amend lost-update,
and the sidecar torn-write. learn was already flock-safe. Model-asset
cache writes (internal/embed) are single-writer and left unchanged.

AI-Used: [claude]
```

## Self-review (writing-plans checklist)
- **Coverage:** atomic-write helper = Task 1; #660 manifest flock = Task 2; amend flock = Task 3; sidecar =
  Task 4 (covered by Task 1 + comment fix); real-binary verify = Task 5. Every map-3 finding has a task.
- **Type consistency:** `Lock func(string)(func(),error)` matches the existing `LearnDeps.Lock` signature
  (`learn.go:66`); `atomicWriteFile(path,data,perm)` mirrors the `os.WriteFile` signature it replaces.
- **Scope:** vault/index writes only; model-asset cache excluded (named); the payload-prune build stays
  deferred (separate spec). DRY: one write helper, one `flockPath` helper shared by vault + manifest locks.
- **Test idiom:** plain inline closures (the `amend_test.go` pattern), order-recording for lock interaction,
  a real-flock concurrent regression test for #660.
