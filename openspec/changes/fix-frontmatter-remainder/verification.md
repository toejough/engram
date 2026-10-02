# Verification

## Task 0: parity goldens from `0f5d91b7`, before any code change (2026-10-01)

The tree's code was at `0f5d91b7` (only the proposal was committed on top).

**0.1 Backfill goldens.** A temporary generator test, run under `targ test` with an env var and then deleted, ran `backfillIdentity` in-process on the 14 inputs of `backfillParityCases()` (fact and feedback; minimal with an unquoted `created:`, `project:`, every optional modeled key, exchange fields, explicitly empty identity, no detectable repo, vocab tags) and wrote `internal/cli/testdata/backfill_identity_parity/*.md`.

**0.2 Amend goldens.** The same generator re-ran amend's 32 parity cases (`amendParityCases()`) with the code at `0f5d91b7`: all 32 outputs were byte-identical to the committed `testdata/amend_hash_parity/` goldens (written at `49cfc120`). Report line: `amend identical: 32 differ: 0`.

**0.3 Pre-change binary.** A binary built from `0f5d91b7` (`git worktree add --detach` into the scratchpad, `go build -o $S/bin/engram-pre ./cmd/engram`, no `go install`), run under `env -i PATH=/usr/bin:/bin HOME=$S/home XDG_DATA_HOME=$S/data` from a scratch git repo whose origin is `github.com/acme/widgets` (the parity test's detected repo), with `user.email = bob@example.com` in the scratch `$HOME/.gitconfig`:

- **Amend:** each of the 26 cases that need no chunk index was written to its own scratch vault and amended with the matching flags (`--clear-pending` cases passed the `--expect-hash` that `engram show` printed). **26/26 notes byte-identical to the goldens** (not only the exchange hash). The 6 `--chunk-source` cases are pinned in-process only, as before.
- **Backfill:** not reachable from the binary in isolation. `engram update --backfill-identity` runs the backfill only after the full self-update (`resolveSource` clones the remote into `$TMPDIR/engram-update-clone`, builds and installs a binary, and re-execs it), so the backfill would run in a freshly built *remote* binary, not the `0f5d91b7` one. The first attempt stopped at the clone's git-lfs check, before any vault write; the stray `/tmp/engram-update-clone` it created was removed. Backfill parity is pinned in-process against goldens written by the `0f5d91b7` code (0.1), as the archived change pinned its receipt cases.
