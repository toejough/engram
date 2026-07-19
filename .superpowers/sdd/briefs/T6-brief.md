### Task T6 (Q2): query + query-chunks + show-chunk compose from Deps (EdgeFS lister, injected clock)

Sequencing: AFTER Q1 (`newVaultFS`, `osTestEdgeFS`). Per R4, T6 runs BEFORE the prune (T9) and amend (T12) conversions — so T6 flips `listJSONLIndexes` call sites ONLY in the files it converts itself (query.go, query_chunks.go, show_chunk.go; T6 owns the show-chunk conversion) and keeps the legacy os-backed lister alive, renamed `osListJSONLIndexes`, for amend/prune until T9/T12 flip their own lines. T12 (last consumer) deletes it, grep-gated. See R3.

**Files:**
- Modify: `internal/cli/query_chunks.go`, `internal/cli/query.go`, `internal/cli/show_chunk.go` (full `newShowChunkDeps` conversion — step 5), `internal/cli/export_test.go`, `internal/cli/query_chunks_test.go`, `internal/cli/ingest_integration_test.go` (2 lines), `internal/cli/targets.go` (3 lines), `internal/cli/amend.go` (1 line, mechanical rename only), `internal/cli/prune.go` (1 line, mechanical rename only), `internal/cli/deps.go` (only if `logWarningTo` not yet landed by the learn cluster)
- Delete: none

**Interfaces:**
- Consumes: `cli.Deps{FS, Embed, Stderr, Now}`; `logWarningTo(w io.Writer) func(format string, args ...any)`
- Produces: `func listJSONLIndexes(fsys EdgeFS) func(dir string) ([]string, error)` (CANONICAL final shape — T6 consumes it in-file for query/query-chunks/show-chunk; T9 (prune) and T12 (amend) consume it as `listJSONLIndexes(d.FS)` when they convert); `osListJSONLIndexes` (TRANSITIONAL — the renamed legacy os-backed lister, deleted by T12 grep-gated); `func newChunkQueryDeps(d Deps) ChunkQueryDeps`; `func newQueryDeps(d Deps) QueryDeps`; `func newShowChunkDeps(d Deps) ShowChunkDeps`; test shim `ExportNewChunkQueryDeps(fsys EdgeFS, emb embed.Embedder) ChunkQueryDeps`

**Steps:**

1. [ ] RED — add to `internal/cli/query_chunks_test.go` (package cli_test; this file uses `gomega.NewWithT`, non-dot import — add `"os"`, `"path/filepath"` to its imports):
   ```go
   func TestChunkQueryDeps_ListIndexes_WrappedNotExistIsEmptyIndex(t *testing.T) {
   	t.Parallel()
   	g := gomega.NewWithT(t)

   	deps := cli.ExportNewChunkQueryDeps(wrappedNotExistEdgeFS{}, nil)
   	paths, err := deps.ListIndexes("/any/chunks/dir")
   	g.Expect(err).NotTo(gomega.HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(paths).To(gomega.BeEmpty())
   }

   func TestChunkQueryDeps_ListIndexes_ListsOnlyJSONLFiles(t *testing.T) {
   	t.Parallel()
   	g := gomega.NewWithT(t)

   	dir := t.TempDir()
   	g.Expect(os.MkdirAll(filepath.Join(dir, "sub.jsonl"), 0o750)).To(gomega.Succeed())
   	g.Expect(os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0o600)).To(gomega.Succeed())
   	g.Expect(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o600)).To(gomega.Succeed())

   	deps := cli.ExportNewChunkQueryDeps(osTestEdgeFS{}, nil)
   	paths, err := deps.ListIndexes(dir)
   	g.Expect(err).NotTo(gomega.HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(paths).To(gomega.ConsistOf(filepath.Join(dir, "a.jsonl")))
   }
   ```
   Run `targ test` — expected: compile failure (`undefined: cli.ExportNewChunkQueryDeps` — the old shim is `ExportNewOsChunkQueryDeps`).

2. [ ] GREEN — rewrite `internal/cli/query_chunks.go` I/O seams. Imports: add `"io/fs"`; KEEP `"os"` (the transitional lister below still uses `os.ReadDir`/`os.IsNotExist`; T12 deletes both). Do NOT delete the legacy os-backed lister (current lines 136-157) — amend.go and prune.go still reference it and have no `d Deps` in scope yet (R3). Instead RENAME it to `osListJSONLIndexes`, body unchanged, with this replacement doc comment:
   ```go
   // osListJSONLIndexes is the TRANSITIONAL os-backed .jsonl lister (#700).
   // Remaining consumers: amend.go (newOsAmendDeps) and prune.go
   // (newOsPruneDeps); T9/T12 flip those lines to listJSONLIndexes(d.FS) when
   // they convert, and T12 (last consumer) deletes this func + the "os" import.
   func osListJSONLIndexes(dir string) ([]string, error) {
   ```
   and ADD the canonical curried lister alongside it:
   ```go
   // listJSONLIndexes returns a lister over fsys for the .jsonl files directly
   // under a dir. A missing dir is an empty index (cold start), not an error —
   // matched via errors.Is so EdgeFS implementations may wrap the not-exist
   // error (os.IsNotExist would not unwrap a %w chain).
   func listJSONLIndexes(fsys EdgeFS) func(dir string) ([]string, error) {
   	return func(dir string) ([]string, error) {
   		entries, err := fsys.ReadDir(dir)
   		if err != nil {
   			if errors.Is(err, fs.ErrNotExist) {
   				return nil, nil
   			}

   			return nil, fmt.Errorf("listing chunk indexes: %w", err)
   		}

   		var paths []string

   		for _, entry := range entries {
   			if !entry.IsDir() && filepath.Ext(entry.Name()) == jsonlExt {
   				paths = append(paths, filepath.Join(dir, entry.Name()))
   			}
   		}

   		return paths, nil
   	}
   }
   ```
   and replace lines 185-195 (current `newOsChunkQueryDeps` using `fs := &osEmbedFS{}` + `sharedEmbedder`) with:
   ```go
   // newChunkQueryDeps wires `engram query-chunks` from the injected CLI
   // capabilities — pure composition (#700).
   func newChunkQueryDeps(d Deps) ChunkQueryDeps {
   	return ChunkQueryDeps{
   		ListIndexes: listJSONLIndexes(d.FS),
   		ReadFile:    d.FS.ReadFile,
   		Embedder:    d.Embed,
   	}
   }
   ```

3. [ ] Replace query.go:1286-1298, current:
   ```go
   // newOsQueryDeps wires the production scan + read for the query command.
   func newOsQueryDeps() QueryDeps {
   	embedDeps := newOsEmbedDeps()

   	return QueryDeps{
   		Scan:             embedDeps.Scan,
   		Read:             embedDeps.Read,
   		Embedder:         embedDeps.Embedder,
   		LogWarning:       logWarningToStderrf,
   		ListChunkIndexes: listJSONLIndexes,
   		Now:              time.Now,
   	}
   }
   ```
   with:
   ```go
   // newQueryDeps wires the query command from the injected CLI capabilities —
   // pure composition, every I/O flows through d (#700).
   func newQueryDeps(d Deps) QueryDeps {
   	vfs := newVaultFS(d.FS)

   	return QueryDeps{
   		Scan: func(vault string) ([]vaultgraph.Note, error) {
   			return vaultgraph.ScanVault(vfs, vault)
   		},
   		Read:             d.FS.ReadFile,
   		Embedder:         d.Embed,
   		LogWarning:       logWarningTo(d.Stderr),
   		ListChunkIndexes: listJSONLIndexes(d.FS),
   		Now:              d.Now,
   	}
   }
   ```
   (`"time"` import stays — `time.Time`/`time.Duration` types remain throughout query.go; `time.Now` is now gone. Behavioral note: `Read` error wrap text changes from osEmbedFS's `"read: %w"` to the cmd adapter's wrap — non-behavioral, both consumers at query.go:1068 and :1434 only branch on error presence.) If the learn cluster has not yet landed `logWarningTo`, add to `internal/cli/deps.go`:
   ```go
   // logWarningTo returns a LogWarning hook writing "warning: ..." lines to w —
   // the pure replacement for the legacy logWarningToStderrf (#700).
   func logWarningTo(w io.Writer) func(format string, args ...any) {
   	return func(format string, args ...any) {
   		_, _ = fmt.Fprintf(w, "warning: "+format+"\n", args...)
   	}
   }
   ```

4. [ ] Mechanical rename of the two remaining foreign references (same commit; rename ONLY — these constructors are still `newOsAmendDeps()`/`newOsPruneDeps()` with no `d Deps` in scope, so NO deps flip here; T12/T9 own those flips per R3):
   - amend.go:365 `ListIndexes: listJSONLIndexes,` → `ListIndexes: osListJSONLIndexes,`
   - prune.go:115 `ListIndexes: listJSONLIndexes,` → `ListIndexes: osListJSONLIndexes,`
   - amend.go:361-362 comment `// listJSONLIndexes (query_chunks.go) lists *.jsonl chunk indexes, treats` → start it with `// osListJSONLIndexes (query_chunks.go) lists ...` (T12 replaces the whole constructor, comment included, when it converts).

5. [ ] Convert `internal/cli/show_chunk.go` — the query family owns show-chunk; this also retires the last `osEmbedFS` consumer outside the files T8/T9/T12/T15 already handle, so T15's `osEmbedFS` deletion compiles. Current code (show_chunk.go:66-75, verified — `&osEmbedFS{}` at :69, lister reference at :72):
   ```go
   // newOsShowChunkDeps wires the production filesystem index loader for
   // `engram show-chunk`. No embedder is needed — lookup is by id, not similarity.
   func newOsShowChunkDeps() ShowChunkDeps {
   	fs := &osEmbedFS{}

   	return ShowChunkDeps{
   		ListIndexes: listJSONLIndexes,
   		ReadFile:    fs.Read,
   	}
   }
   ```
   Replace with:
   ```go
   // newShowChunkDeps wires `engram show-chunk` from the injected CLI
   // capabilities — pure composition (#700). No embedder is needed — lookup
   // is by id, not similarity.
   func newShowChunkDeps(d Deps) ShowChunkDeps {
   	return ShowChunkDeps{
   		ListIndexes: listJSONLIndexes(d.FS),
   		ReadFile:    d.FS.ReadFile,
   	}
   }
   ```
   No import changes in show_chunk.go (it never imported `os`; `osEmbedFS` came from package-mate embed.go). Behavioral note: `ReadFile`'s error wrap text changes from osEmbedFS's `"read: %w"` to the EdgeFS adapter's wrap — non-behavioral; `loadChunkRecords` only re-wraps. Test adjustments: NONE — show_chunk_test.go's three tests (lines 18, 39, 61) inject `cli.ShowChunkDeps{...}` literals directly and never touch the constructor; there is no `ExportNewOsShowChunkDeps` shim (verified); the wiring is exercised by targets_test.go:179's show-chunk test through `executeForTest` and the step-9 suite run.

6. [ ] Update export_test.go lines 514-521, current:
   ```go
   // ExportNewOsChunkQueryDeps returns production ChunkQueryDeps with an
   // injected embedder, mirroring ExportNewOsIngestDeps.
   func ExportNewOsChunkQueryDeps(emb embed.Embedder) ChunkQueryDeps {
   	deps := newOsChunkQueryDeps()
   	deps.Embedder = emb

   	return deps
   }
   ```
   replacement:
   ```go
   // ExportNewChunkQueryDeps returns production ChunkQueryDeps over the given
   // EdgeFS with an injected embedder.
   func ExportNewChunkQueryDeps(fsys EdgeFS, emb embed.Embedder) ChunkQueryDeps {
   	deps := newChunkQueryDeps(Deps{FS: fsys})
   	deps.Embedder = emb

   	return deps
   }
   ```
   Update ingest_integration_test.go lines 100 and 204: `cli.ExportNewOsChunkQueryDeps(fakeIngestEmbedder{})` → `cli.ExportNewChunkQueryDeps(osTestEdgeFS{}, fakeIngestEmbedder{})`.

7. [ ] Update targets.go call sites (identifier is `deps`, per T2's landed `ingestQueryTargets(deps Deps, ...)`): line 155 `newOsQueryDeps()` → `newQueryDeps(deps)`; line 169 `newOsChunkQueryDeps()` → `newChunkQueryDeps(deps)`; line 186 `newOsShowChunkDeps()` → `newShowChunkDeps(deps)` (line numbers are pre-T2 anchors — locate by constructor name).
8. [ ] Verify purity of the migrated files: `grep -n '"os"' internal/cli/query.go internal/cli/show_chunk.go` — expected: no output. `grep -n 'time\.Now' internal/cli/query.go` — expected: no output. `grep -n 'os\.' internal/cli/query_chunks.go` — expected: hits ONLY inside `osListJSONLIndexes` (the transitional lister; full query_chunks.go purity is T12's exit criterion, not this task's). `grep -rn 'newOsShowChunkDeps\|osEmbedFS' internal/cli/show_chunk.go` — expected: no output.
9. [ ] Run `targ test` — expected: all green (step-1 tests now pass; show-chunk, ingest integration + query suites unchanged). Run `targ check-full` — expected: clean. Run `targ check-thin-api` — expected: PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress.
10. [ ] Commit: `refactor(cli): query + show-chunk compose from Deps (#700)`

---

