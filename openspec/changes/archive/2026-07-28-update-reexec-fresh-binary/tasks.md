## 1. Exec primitive and DI plumbing

- [x] 1.1 Add a process-spawn primitive (run binary with args + env, inherited stdio; returns `(exitCode int, err error)` — err strictly for spawn failure, distinct from a started child's non-zero exit) to `cli.Primitives` and a thin adapter in `cli.NewDeps`; keep `cmd/engram/main.go` checker-thin (`targ check-thin-api` passes)
- [x] 1.2 Extend the update deps with the injected exec interface and an env-getter for the `ENGRAM_UPDATE_REEXEC` sentinel (imptest mock generated)

## 2. Re-exec in Updater.Run (TDD per spec scenarios)

- [x] 2.1 RED: tests for sentinel run — no install invoked, no spawn, sync/checks proceed in-process
- [x] 2.2 RED: tests for successful install → spawn of resolved installed path with original args + sentinel env, parent performs no plan/apply, parent returns child's exit code
- [x] 2.3 RED: tests for dry-run (no install, no spawn) and for spawn failure → in-process fallback with report line carrying the underlying error
- [x] 2.4 GREEN: implement sentinel check in `resolveSource`, re-exec boundary in `Updater.Run` after successful install (recording `Report.ReexecExitCode` per design D8), fallback path, report attribution (install output in parent, single sync report in child)
- [x] 2.5 GREEN (control flow, D8): `runUpdate` in `internal/cli/update.go` branches on the report's handoff signal immediately after `updater.Run` returns and calls `deps.Exit(code)` BEFORE the vault/vocab/chunk-check block; tests cover parent-exits-early and fallback-continues paths
- [x] 2.6 REFACTOR: clean split of pre-/post-boundary phases; fix stale `go install ...@latest` comment at update.go:1149 (clone-based per #645)

## 3. Verification and close-out

- [x] 3.1 `targ test` and `targ check-full` clean
- [x] 3.2 Real-binary verification: `go install ./cmd/engram`, run `engram update` from outside the repo and observe install → re-exec → single sync report; run `--dry-run` and confirm no spawn
- [x] 3.3 Update docs: `docs/architecture/adr.md` (new ADR for install-then-re-exec), c1 sequence diagram for the update flow, and sync `openspec/specs/update-self-reexec/` on archive
