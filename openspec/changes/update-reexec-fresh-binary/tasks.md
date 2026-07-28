## 1. Exec primitive and DI plumbing

- [ ] 1.1 Add a process-spawn primitive (run binary with args + env, inherited stdio, returns exit code) to `cli.Primitives` and a thin adapter in `cli.NewDeps`; keep `cmd/engram/main.go` checker-thin (`targ check-thin-api` passes)
- [ ] 1.2 Extend the update deps with the injected exec interface and an env-getter for the `ENGRAM_UPDATE_REEXEC` sentinel (imptest mock generated)

## 2. Re-exec in Updater.Run (TDD per spec scenarios)

- [ ] 2.1 RED: tests for sentinel run — no install invoked, no spawn, sync/checks proceed in-process
- [ ] 2.2 RED: tests for successful install → spawn of resolved installed path with original args + sentinel env, parent performs no plan/apply, parent returns child's exit code
- [ ] 2.3 RED: tests for dry-run (no install, no spawn) and for spawn failure → in-process fallback with report line carrying the underlying error
- [ ] 2.4 GREEN: implement sentinel check in `resolveSource`, re-exec boundary in `Updater.Run` after successful install, fallback path, report attribution (install output in parent, single sync report in child)
- [ ] 2.5 REFACTOR: clean split of pre-/post-boundary phases; ensure vault/vocab/chunk checks in `internal/cli/update.go` run only on the child/fallback path

## 3. Verification and close-out

- [ ] 3.1 `targ test` and `targ check-full` clean
- [ ] 3.2 Real-binary verification: `go install ./cmd/engram`, run `engram update` from outside the repo and observe install → re-exec → single sync report; run `--dry-run` and confirm no spawn
- [ ] 3.3 Update docs: `docs/architecture/adr.md` (new ADR for install-then-re-exec), c1 sequence diagram for the update flow, and sync `openspec/specs/update-self-reexec/` on archive
