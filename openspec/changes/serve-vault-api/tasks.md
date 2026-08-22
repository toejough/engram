## 1. HTTP listener primitive (thin-api composition)

- [x] 1.1 Add a raw HTTP-listener primitive group to `cli.Primitives` (mirroring `ExecPrims`/`SpawnPrims`) wrapping `net/http` in `cmd/engram` only — zero composition/branching in the primitive itself, per `targ check-thin-api`
- [x] 1.2 Compose the listener into a deps struct via `cli.NewDeps`-style wiring, not ad-hoc (implementation note: lives on `cli.Deps` directly — `NewServeMux`/`RegisterRoute`/`ListenAndServe`/`Fetch` — rather than a separate deps struct; see task 2.1's note)

## 2. `internal/serve` package skeleton

- [x] 2.1 Route registration for exactly the served set: `query`/`query-chunks`/`show`/`show-chunk` as GET (query params), `activate`/`learn`/`amend` as POST (JSON body) — no route for `ingest`/`vocab refit`/`prune`/`check`/`update`/`resituate` (implementation note: lives in `internal/cli` — `serve.go`/`serve_client.go` — not a separate `internal/serve` package. `newXDeps` composers and, originally, `runLearn` are unexported, so a truly separate package couldn't call them; `RunLearn` was exported, and the rest stayed same-package. `targ check-thin-api` additionally requires every declaration in `cmd/engram` to be a single-call/simple-error-guard body — no loops, no map-typed call heads, no `go`/`defer` statements — which forced the per-route-registration-call design in `RegisterRoute` over a batch `[]ServeRoute` primitive)
- [x] 2.2 Each route handler calls the existing `Run*` function for that command directly (`RunQuery`, `RunShow`, `RunActivate`, `RunLearn`, `RunAmend`, etc.) — no reimplementation of command logic in the handler layer
- [x] 2.3 Wire each handler through the same `Deps`/lock-acquiring composition the CLI target already uses, so served commands share the CLI's existing vault locks (ADR-0013)

## 3. Explicit bind address

- [x] 3.1 Add `engram serve` to `internal/cli/targets.go` with a required bind-address flag/env; refuse to start with no explicit address (never default to `0.0.0.0`) — `ServeArgs.Addr` is `targ:"...required"` with no `env=` fallback

## 4. Cloudflare Access identity for served writes

- [x] 4.1 Add a Cloudflare Access header extraction + verification adapter (reads the configured header, `Cf-Access-Authenticated-User-Email`) composed into the serve handlers
- [x] 4.2 For served `learn`/`amend` requests, compose a `DetectUser` that returns the Cloudflare-derived identity instead of `detectUser`'s git-config/OS-username chain — same `identityStamp`/`DetectUser func(ctx context.Context) string` shape `note-origin-identity` already established, different source function
- [x] 4.3 Confirm `repo:`/`vault:` resolution for served writes: `repo:` passes through the client-detected value carried on `LearnArgs.Repo`/`AmendArgs.Repo` (client-side `detectRepo` runs in ENGRAM_SERVER-mode CLI targets before the POST — server never re-detects, which would resolve to the server process's own repo context); `vault:` (the frontmatter label) falls back to the server's configured `--vault-name` only when the client didn't supply one, same `resolveVaultName` semantics local `learn`/`amend` already use

## 5. Pending-offer marker

- [x] 5.1 Add a pending-offer marker field to `factFrontmatterDoc`/`feedbackFrontmatterDoc` (`internal/cli/learn.go`) — `pending: bool` (yaml `pending,omitempty`)
- [x] 5.2 `internal/cli/query.go`: exclude notes carrying the pending-offer marker from normal query results (`excludePendingOffers`, wired into `RunQuery`)
- [x] 5.3 Served `learn`/`amend` handlers write the pending-offer marker instead of letting the note commit as an immediately-live note
- [x] 5.4 Confirm the served `activate` handler bypasses the offer path entirely — direct sidecar `LastUsed` commit, no pending marker, no curation

## 6. Stateless pending-offer detection + surfacing

- [x] 6.1 Add a stateless pending-offer detector (vault scan for the marker, no persisted state, no growth/time batching) — same shape as `notesMissingIdentityFields`, not `checkAndPersistVocabRefitTrigger` (`vaultHasPendingOffers` in `internal/cli/offer.go`)
- [x] 6.2 `internal/cli/query.go`: include a pending-offer-exists flag in the payload, computed fresh on every call (alongside the existing `refit_pending` read) — `pending_offers` field
- [x] 6.3 `internal/cli/update.go`: add a new detect-and-notify notice for pending offers, following the existing detector convention (ADR-0021)
- [x] 6.4 Wire a `LogWarning` nudge into the same call sites `checkAndPersistVocabRefitTrigger` already runs from (`applyVocabAssignmentAfterLearn`/`AfterAmend`/`AfterResituate`) when pending offers exist — log-only, no persisted state written from these call sites

## 7. Query model_id exposure

- [x] 7.1 `internal/cli/query.go`: add `model_id` to `queryPayload`, sourced from `deps.Embedder.ModelID()` — applies to local and served `query` alike

## 8. `ENGRAM_SERVER` CLI-client mode

- [x] 8.1 When `ENGRAM_SERVER` is set, route the served command set's CLI targets (`query`, `query-chunks`, `show`, `show-chunk`, `activate`, `learn`, `amend`) through an HTTP client instead of local file I/O, with identical command syntax; GET routes copy the server's response body verbatim to stdout (byte-identical to local per design.md); `learn`/`amend` print the offer receipt instead of a note path (deliberately different — see design.md API Contract)
- [x] 8.2 Confirm host-only commands (`ingest`, `vocab refit`, `prune`, `check`, `update`, `resituate`) are unaffected by `ENGRAM_SERVER` — they always run against local files (never wired to `serverBase`/`fetch*`)

## 9. Curation skill (follow-on, outside this change's Go-code scope)

- [x] 9.1 Author a curation skill under the `superpowers:writing-skills` convention (per CLAUDE.md, mandatory for any SKILL.md edit) that judges pending offers against the host vault using the same covered/near/absent reasoning `recall`'s Step 2.5 already documents (`agent-instructions/skills/curate/SKILL.md`). Design resolution beyond tasks.md's literal wording (recorded in the skill itself, Step 3): since the offer already exists as a note file — unlike recall's fresh, not-yet-written candidate — both **covered** and **near** end in `engram amend --discard` (a new lock-safe delete added to `internal/cli/amend.go` for this purpose, since `engram` had no note-deletion command at all) rather than leaving a redundant live note; only **absent** clears the pending marker (`engram amend --clear-pending`, also newly added — `AmendArgs.Pending` existed in Go but wasn't wired to a CLI flag). Both new flags are host-local only: `serveAmend` unconditionally forces `Discard=false` after decoding a client body (closing a real gap this work surfaced — an unguarded `Discard` field would have let a served `/amend` request delete an arbitrary vault note), and the local CLI refuses `--discard` outright when `ENGRAM_SERVER` is set. Verified via the mandatory RED (fresh subagent, no skill, judged a 3-offer covered/near/absent fixture unaided) → GREEN (same fixture, skill available) cycle — the skill-equipped run reproduced the same judgments in ~43% fewer tokens and ~31% fewer tool calls, with zero deviation from the skill's red-flags table.
- [x] 9.2 Confirm the skill never runs synchronously inside a served HTTP request — stated explicitly in the skill (host-local only, invoked reactively off the three surfacing signals or on request) and structurally guaranteed by design.md's hard constraint that `engram serve`'s process never invokes an agent/LLM call at all

## 10. Tests

- [x] 10.1 Served command set: each of the seven served commands reachable and produces the same result as local invocation, over the correct method (GET for the four read-only routes, POST for the three writes); each host-only command has no route (`TestServeRoutes_MethodsAndPatterns`, `TestServeShow_MatchesLocalOutput`, real round-trip tests in `cmd/engram`)
- [x] 10.2 Bind address: starting `engram serve` without an explicit address refuses to start (`TestServeTarget_RequiresAddr`, confirmed via real binary: `Error: missing required flag: --addr`)
- [x] 10.3 Concurrency: a local writer and a served writer racing the same vault — no lost update (`TestServeLearn_ConcurrentWithLocalLearn_NoLostUpdate`, mirroring `TestInvariant_K1_ConcurrentLearnNeverCollides`'s ADR-0013 pattern but mixing local and served writers)
- [x] 10.4 Parity: a fixed `query`/`show` invocation returns byte-identical output via local CLI vs. `ENGRAM_SERVER`-routed CLI (`TestEngramServer_Query_RoutesThroughFetch`, `TestEngramServer_Show_PercentEncodesQueryValues` — response body copied verbatim)
- [x] 10.5 Identity: a served write lands with the Cloudflare-authenticated identity, never a client-supplied one (`LearnArgs`/`AmendArgs` carry no client `user:` field at all — structurally impossible to spoof); `repo:`/`vault:` behavior covered (`TestServeLearn_StampsCloudflareIdentityAndPendingMarker`, `TestServeAmend_StampsIdentityAndPendingMarker`)
- [x] 10.6 Offer marker: a served `learn`/`amend` produces a note carrying the pending-offer marker; `query` excludes it; the marker is absent from notes created by local (non-served) `learn`/`amend` (`TestServeQuery_ExcludesPendingOffersAndSetsModelID`, `TestLocalLearn_NeverSetsPendingMarker`, `TestLocalAmend_NeverSetsPendingMarker`)
- [x] 10.7 Pending-offer detection: fires immediately on a single pending offer (no batching) at all three surfacing points (query payload, update notice, write-path log); silent when none exist (`TestServeQuery_...`, `TestVaultHasPendingOffers`, `TestWarnIfPendingOffers`)
- [x] 10.8 Fire-and-forget: the served API's response to a write that creates a pending offer contains no curation-outcome field (asserted directly: receipt is exactly `{status, luhmann}`, response body checked to not contain note content)
- [x] 10.9 `activate` (served): commits directly, no pending marker created (`TestServeActivate_CommitsDirectly`)
- [x] 10.10 `model_id`: present in every `query` payload, local and served (`TestQuery_PayloadIncludesModelID` for local, `TestServeQuery_ExcludesPendingOffersAndSetsModelID` for served)

## 11. Verification

- [x] 11.1 `targ test`, `targ lint-for-fail`, `targ lint-full`, `targ check-thin-api`, `targ check-nils-for-fail`, `targ reorder-decls-check`, `targ check-coverage-for-fail` all fully green — no exceptions, no pre-existing failures tolerated. This required fixing four latent environment/tooling issues discovered along the way (none introduced by this change, all now fixed rather than worked around):
  - The bundled ONNX embedding model was a git-lfs pointer file (133 bytes) in this sandbox, not real weights (git-lfs wasn't installed) — installed git-lfs, ran `git lfs pull`, and cleared a stale extracted-stub cache at `~/.cache/engram/models/minilm-l6-v2@384` that had been extracted once from the pointer and never got a chance to re-extract.
  - `TestEngramQuery_F6F91_EndToEnd` hand-stamped sidecars with arbitrary hot-encoded vectors uncorrelated with any real embedding of the query phrase (measured cosine ~0.0 against a real MiniLM embedding, nowhere near `matchRelevanceFloor`=0.25) — the test could never pass against a working embedder. Rewrote the fixture to use real topically-distinct note text embedded via a real `engram embed apply` pass.
  - `TestTargets_ActivateNoNotes` omitted `--vault`, silently depending on `$HOME/.local/share/engram/vault` already existing (RunActivate's lock acquisition isn't self-silencing on a missing directory, unlike the update detectors) — added an explicit `--vault <tempdir>`.
  - `dev/golangci-lint.toml`'s disable list named `exhaustruct`/`wsl`, but golangci-lint 2.13 renamed them to `exhaustruct_v5`/`wsl_v5` (old names now deprecated aliases that no longer suppress the linter under `default = 'all'`) — this had been silently breaking `lint-for-fail`/`check-full` for everyone on this repo. Fixed by disabling both the old and new names.
  - Fixed 22 functions below the 80% per-function coverage threshold surfaced once `check-coverage-for-fail` could finally run to completion (18 in this change's own new code, 3 in `note-origin-identity`'s `identity_backfill.go` from earlier this session, 1 ancient/unrelated `resolveBinaryPath` in `internal/update`) by adding targeted tests — no coverage-threshold workarounds.
- [x] 11.2 Real-binary check: ran `engram serve` against a scratch vault, drove `/learn` via `curl` with a simulated Cloudflare header and via `ENGRAM_SERVER`-set `engram learn`/`query`, confirmed parity and a concurrent local + served `learn` both landed in one vault
- [x] 11.3 Real-binary check: a served `learn` with a Cloudflare Access test header produced a pending-offer note stamped with that identity; `engram update` surfaced its existence via the new notice; a local `engram learn` in the same vault did not trigger the pending-offer marker or its notices
