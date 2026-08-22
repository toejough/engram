## Why

`serve-vault-api` (shipped 90f28a6e, archived 2026-08-22) stamps every served `learn`/`amend`
write's `user:` field from a Cloudflare Access header, and hard-refuses the request (401) when
that header is absent. Cloudflare Access is one deployment's edge-auth choice, not a property
of `engram` itself — a user running `engram serve` with none of that infrastructure cannot use
`/learn` or `/amend` over the wire at all. [#726](https://github.com/toejough/engram/issues/726)
flags this as a hard dependency that shouldn't exist in a general-purpose tool, filed ahead of a
real consumer (phone-llm) that has no Cloudflare in its path.

## What Changes

- Identity for served `learn`/`amend` becomes **client-declared, not edge-verified**. The
  server drops its Cloudflare-only identity source entirely and instead trusts whatever
  identity the calling `engram` instance detected locally — the same trust treatment `repo:`
  already gets on a served write. **BREAKING**: reverses the archived `serve-vault-api`
  change's explicit decision that "only `user:` is server-overridden... never trust a
  client-supplied `user:`." The new trust boundary is network reachability of the server
  itself, not an edge-authenticated header.
- The server SHALL still reject a served `learn`/`amend` request whose declared identity is
  empty — a floor ("you must claim something"), not identity verification.
- `engram serve`'s and `ENGRAM_SERVER`'s concurrency model — one vault, one node, two doors:
  the local CLI commits immediately and owns every host-only verb forever; `engram serve` is
  the same node's network door and only ever lands pending offers — gets written down in the
  README and in `engram serve --help`'s own description. This documents behavior the archived
  change already built (`vault-offer-curation`'s pending-offer marker, ADR-0013's shared
  flocks) but never stated in either place.

Two items considered and explicitly dropped, not deferred as follow-ups:
- A bearer-token → identity map (config-driven allowlist) as the baseline identity mechanism,
  and an opt-in trusted-header mode (generalizing today's Cloudflare header to any
  authenticating proxy) as a second adapter. Both were #726's originally proposed shape; the
  simpler "trust the client, reject only empty" model was chosen instead for this pass.
- A pidfile/lockfile guard refusing a second concurrent `engram serve` against the same vault.
  ADR-0013's flocks already make concurrent writers correct regardless of how many processes
  are running; pending-offer surfacing is a stateless re-scan with no server-local state to
  diverge. With the credential map gone, nothing left makes two servers a correctness bug —
  only a resource-waste/operational-hygiene concern, not addressed here.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `vault-serve-api`: the "Served writes are attributed to the server-verified caller identity"
  requirement changes from Cloudflare-header-sourced identity (with a hard-401 on a missing
  header) to client-declared identity (rejected only when empty). `repo:`'s existing
  client-detected treatment is unchanged; `user:` now gets the same treatment `repo:` already
  has, instead of the opposite of it.

## Impact

- `internal/cli/serve.go`: `serveLearn`/`serveAmend` drop the `cloudflareIdentityHeader`
  lookup and the `errServeMissingIdentity` 401 path; identity now comes from the unmarshaled
  request body, rejected only when empty.
- `internal/cli/learn.go` / `internal/cli/amend.go`: `LearnArgs`/`AmendArgs` gain a
  `User string \`json:"user"\`` field, symmetric with the existing `Repo` field.
- `internal/cli/serve_client.go`: `fetchLearn`/`fetchAmend` populate the new `User` field via
  the existing `detectUser` function (`identity.go`), mirroring how `Repo` is already
  populated via `detectRepo`.
- `README.md`: new section documenting the two-doors concurrency model.
- `internal/cli/targets.go` (or wherever `engram serve`'s command description lives): updated
  help text stating the same model.
- No change to `cli.HTTPPrims`/`Fetch`, no new server-side configuration, no new dependencies.
