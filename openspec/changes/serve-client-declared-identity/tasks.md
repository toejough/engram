## 1. Server-side identity resolution

- [x] 1.1 Add `User string \`json:"user"\`` to `LearnArgs` (`internal/cli/learn.go`) and
      `AmendArgs` (`internal/cli/amend.go`), placed next to the existing `Repo` field.
- [x] 1.2 In `internal/cli/serve.go`, remove `cloudflareIdentityHeader`,
      `errServeMissingIdentity`, and the `firstHeaderValue(req.Header, cloudflareIdentityHeader)`
      lookups in `serveLearn`/`serveAmend`.
- [x] 1.3 In `serveLearn`/`serveAmend`, unmarshal the request body first, then resolve identity
      from `args.User`: reject (400, reusing the existing `jsonErrorResponse` path with a new
      sentinel error for "empty identity") when it's empty; otherwise wire
      `learnDeps.DetectUser`/`amendDeps.DetectUser` to return it, same as today's pattern.
- [x] 1.4 Confirm `statusUnauthorized` is now dead in `serve.go`; remove it if nothing else
      references it (check before deleting — `targ check-full` will catch an unused const
      either way).

## 2. Client-side identity declaration

- [x] 2.1 In `internal/cli/serve_client.go`, `fetchLearn`: add
      `args.User = detectUser(ctx, deps.Commander, deps.Username)` alongside the existing
      `args.Repo = detectRepo(...)` line, before marshaling the body.
- [x] 2.2 Same change in `fetchAmend`.

## 3. Tests

- [x] 3.1 Update `internal/cli/serve_test.go`: replace tests asserting a 401 on missing
      Cloudflare header with tests asserting a 400 (or equivalent) on an empty `User` field,
      and tests asserting a non-empty `User` value is stamped through unmodified — mirroring
      existing `Repo` pass-through test coverage.
- [x] 3.2 Update `cmd/engram/serve_integration_test.go`: drop the Cloudflare-header setup
      currently required to make `TestServeAndFetch_LearnRoundTrip` pass; confirm the round
      trip now works via client-declared identity with no special header.
- [x] 3.3 Add/update a test in `internal/cli` covering `fetchLearn`/`fetchAmend` populating
      `args.User` via `detectUser`, matching the existing `detectRepo` coverage pattern.
- [x] 3.4 Run `targ check-full` to confirm no other test still depends on the removed
      Cloudflare-header identity path. `targ test` is fully green. `targ check-full`'s
      `check-coverage-for-fail` per-function-coverage gate fails, but verified (via
      `git stash`/re-run against clean HEAD) to fail identically before this change —
      pre-existing across `internal/cli`, unrelated to this change, not addressed here.
      `check-uncommitted` fails as expected mid-implementation (work not yet committed).
      `reorder-decls-check` initially flagged one file; fixed via `targ reorder-decls`
      and reconfirmed passing. Every other check (`lint-fast`, `lint-full`, `deadcode`,
      `check-thin-api`, `check-nils-for-fail`, `test-integration`) passes.

## 4. Documentation

- [x] 4.1 Add a README section documenting the two-doors model: one vault = one node = one
      brain; local CLI is the local door (commits immediately; owns `ingest`/`vocab refit`/
      `prune`/`check`/`update`/`resituate` forever, never served); `engram serve` is the
      network door (lands pending offers, curated later); both share `Run*` code paths and
      ADR-0013's flocks.
- [x] 4.2 Update `engram serve`'s `Description(...)` string in `serveTargets`
      (`internal/cli/targets.go:332`) to state the same model concisely.

## 5. Verification

- [x] 5.1 `targ test` passes fully. `targ check-full` passes except the two known/pre-existing
      items noted under 3.4 (`check-coverage-for-fail`, `check-uncommitted`).
- [x] 5.2 Manual check: started a real `engram serve --addr 127.0.0.1:8931` against a temp
      vault, POSTed `/learn` with a `user` field in the JSON body and no auth header of any
      kind — the resulting note's `user:` matched the declared value exactly
      (`manual-tester@example.com`), landed `pending: true`.
- [x] 5.3 Manual check: POSTed `/learn` with no `user` field — got `HTTP 400
      {"error":"serve: user: must be non-empty"}`, and the vault still contained only the one
      note from 5.2 (no write happened).
