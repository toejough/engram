## 1. HTTP listener primitive (thin-api composition)

- [ ] 1.1 Add a raw HTTP-listener primitive group to `cli.Primitives` (mirroring `ExecPrims`/`SpawnPrims`) wrapping `net/http` in `cmd/engram` only — zero composition/branching in the primitive itself, per `targ check-thin-api`
- [ ] 1.2 Compose the listener into a new `internal/serve` package's own deps struct via `cli.NewDeps`-style wiring, not ad-hoc

## 2. `internal/serve` package skeleton

- [ ] 2.1 Create `internal/serve/` with route registration for exactly the served set: `query`/`query-chunks`/`show`/`show-chunk` as GET (query params), `activate`/`learn`/`amend` as POST (JSON body) — no route for `ingest`/`vocab refit`/`prune`/`check`/`update`/`resituate`
- [ ] 2.2 Each route handler calls the existing `Run*` function for that command directly (`RunQuery`, `RunShow`, `RunActivate`, `runLearn`, `RunAmend`, etc.) — no reimplementation of command logic in `internal/serve`
- [ ] 2.3 Wire each handler through the same `Deps`/lock-acquiring composition the CLI target already uses, so served commands share the CLI's existing vault locks (ADR-0013)

## 3. Explicit bind address

- [ ] 3.1 Add `engram serve` to `internal/cli/targets.go` with a required bind-address flag/env; refuse to start with no explicit address (never default to `0.0.0.0`)

## 4. Cloudflare Access identity for served writes

- [ ] 4.1 Add a Cloudflare Access header extraction + verification adapter (reads the configured header, e.g. `Cf-Access-Authenticated-User-Email`) composed into `internal/serve`'s deps
- [ ] 4.2 For served `learn`/`amend` requests, compose a `DetectUser` that returns the Cloudflare-derived identity instead of `detectUser`'s git-config/OS-username chain — same `identityStamp`/`DetectUser func(ctx context.Context) string` shape `note-origin-identity` already established, different source function
- [ ] 4.3 Confirm `repo:`/`vault:` resolution for served writes is untouched — same client-detected/flag-resolved path local `learn`/`amend` already use, no server override

## 5. Pending-offer marker

- [ ] 5.1 Add a pending-offer marker field to `factFrontmatterDoc`/`feedbackFrontmatterDoc` (`internal/cli/learn.go`) — exact key TBD at implementation (design.md Open Questions)
- [ ] 5.2 `internal/cli/query.go`: exclude notes carrying the pending-offer marker from normal query results
- [ ] 5.3 Served `learn`/`amend` handlers in `internal/serve` write the pending-offer marker instead of letting the note commit as an immediately-live note
- [ ] 5.4 Confirm the served `activate` handler bypasses the offer path entirely — direct sidecar `LastUsed` commit, no pending marker, no curation

## 6. Stateless pending-offer detection + surfacing

- [ ] 6.1 Add a stateless pending-offer detector (vault scan for the marker, no persisted state, no growth/time batching) — same shape as `notesMissingIdentityFields`, not `checkAndPersistVocabRefitTrigger`
- [ ] 6.2 `internal/cli/query.go`: include a pending-offer-exists flag in the payload, computed fresh on every call (alongside the existing `refit_pending` read)
- [ ] 6.3 `internal/cli/update.go`: add a new detect-and-notify notice for pending offers, following the existing detector convention (ADR-0021)
- [ ] 6.4 Wire a `LogWarning` nudge into the same call sites `checkAndPersistVocabRefitTrigger` already runs from (`applyVocabAssignmentAfterLearn`/`AfterAmend`/`AfterResituate`) when pending offers exist — log-only, no persisted state written from these call sites

## 7. Query model_id exposure

- [ ] 7.1 `internal/cli/query.go`: add `model_id` to `queryPayload`, sourced from `deps.Embedder.ModelID()` (already read internally for sidecar matching, never previously surfaced) — applies to local and served `query` alike

## 8. `ENGRAM_SERVER` CLI-client mode

- [ ] 8.1 When `ENGRAM_SERVER` is set, route the served command set's CLI targets (`query`, `query-chunks`, `show`, `show-chunk`, `activate`, `learn`, `amend`) through an HTTP client instead of local file I/O, with identical command syntax and output shape
- [ ] 8.2 Confirm host-only commands (`ingest`, `vocab refit`, `prune`, `check`, `update`, `resituate`) are unaffected by `ENGRAM_SERVER` — they always run against local files

## 9. Curation skill (follow-on, outside this change's Go-code scope)

- [ ] 9.1 Author a curation skill under the `superpowers:writing-skills` convention (per CLAUDE.md, mandatory for any SKILL.md edit) that judges pending offers against the host vault using the same covered/near/absent reasoning `recall`'s Step 2.5 already documents, issuing `engram amend` (near) / `engram learn` (absent) / discard (covered), clearing the pending-offer marker in each case
- [ ] 9.2 Confirm the skill never runs synchronously inside a served HTTP request — it is invoked separately (see design.md Open Questions for cadence)

## 10. Tests

- [ ] 10.1 Served command set: each of the seven served commands reachable and produces the same result as local invocation, over the correct method (GET for the four read-only routes, POST for the three writes); each host-only command has no route
- [ ] 10.2 Bind address: starting `engram serve` without an explicit address refuses to start
- [ ] 10.3 Concurrency: a local writer and a served writer racing the same vault — no lost update (reuses the existing concurrent-writers regression pattern, ADR-0013)
- [ ] 10.4 Parity: a fixed `query` invocation returns byte-identical output via local CLI vs. `ENGRAM_SERVER`-routed CLI
- [ ] 10.5 Identity: a served write with a spoofed client-supplied `user:` still lands with the Cloudflare-authenticated identity; `repo:`/`vault:` are unaffected by the server
- [ ] 10.6 Offer marker: a served `learn`/`amend` produces a note carrying the pending-offer marker; `query` excludes it; the marker is absent from notes created by local (non-served) `learn`/`amend`
- [ ] 10.7 Pending-offer detection: fires immediately on a single pending offer (no batching) at all three surfacing points (query payload, update notice, write-path log); silent when none exist
- [ ] 10.8 Fire-and-forget: the served API's response to a write that creates a pending offer contains no curation-outcome field
- [ ] 10.9 `activate` (served): commits directly, no pending marker created, note absent from any offer-related surfacing
- [ ] 10.10 `model_id`: present in every `query` payload, local and served, matching `deps.Embedder.ModelID()`

## 11. Verification

- [ ] 11.1 `targ check-full` green
- [ ] 11.2 Real-binary check: run `engram serve` against a scratch vault, drive each served command from a second `engram` invocation with `ENGRAM_SERVER` set, confirm parity and locking against a concurrent local writer
- [ ] 11.3 Real-binary check: a served `learn` with a Cloudflare Access test header produces a pending-offer note stamped with that identity; `engram query`/`engram update` surface its existence; a local `engram learn` in the same vault does not trigger the pending-offer marker or its notices
