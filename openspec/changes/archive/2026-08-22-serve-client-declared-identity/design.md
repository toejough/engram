## Context

See `proposal.md` - Why. This change corrects two things in the archived `serve-vault-api`
change (`openspec/changes/archive/2026-08-22-serve-vault-api/`), per
[#726](https://github.com/toejough/engram/issues/726):

- `internal/cli/serve.go`'s `serveLearn`/`serveAmend` currently read
  `Cf-Access-Authenticated-User-Email` via `firstHeaderValue` and return `401` through
  `errServeMissingIdentity` when it's absent — the *only* identity path either handler has.
- `LearnArgs`/`AmendArgs` (`internal/cli/learn.go`, `internal/cli/amend.go`) already carry a
  client-detected `Repo string \`json:"repo"\`` field that the server trusts unmodified
  (`serveLearn`/`serveAmend` pass `args.Repo` through as-is). Neither struct has a `User`
  field today — locally, `user:` comes from `LearnDeps.DetectUser`/`AmendDeps.DetectUser`
  closures (`identity.go`'s `detectUser`, called from `newLearnDeps`/`newAmendDeps`), never
  from the request body, because there was never a served path that needed it to travel.
- `identity.go`'s `detectUser(ctx, commander, username)` and `detectRepo(ctx, getwd, commander)`
  are already parallel, already-tested functions. `serve_client.go`'s `fetchLearn`/`fetchAmend`
  already call `detectRepo` to populate `args.Repo` before marshaling the request body — this
  change adds the equivalent `detectUser` call for a new `args.User` field, nothing more.
- The archived `serve-vault-api` design.md made an explicit, opposite decision: *"Only `user:`
  is server-overridden... it's the field a malicious or misconfigured client could lie
  about."* This change reverses that specific line — deliberately, not by omission. See
  Decisions below for why.

## Goals / Non-Goals

**Goals:**
- Make identity resolution for served `learn`/`amend` work with zero external
  services — no Cloudflare Access, no reverse proxy, no operator-managed credential
  configuration.
- Keep `user:`'s treatment symmetric with `repo:`'s existing treatment: both client-detected,
  both passed through unmodified, both trusted at the value they arrive with.
- Preserve a minimal floor: a served write cannot land with a blank `user:`.
- Document the local/served concurrency model (README + `engram serve --help`) that the
  archived change built but never wrote down anywhere.

**Non-Goals:**
- No credential→identity map, no bearer-token config, no new server-side configuration
  surface. Considered (it was #726's original ask) and dropped for this pass — see Decisions.
- No trusted-header mode (a configurable-header-name generalization of today's Cloudflare
  handling, for operators who do run an authenticating proxy). Also #726's original ask, also
  dropped for this pass.
- No verification of the declared identity — no signature, no challenge, no session. The
  trust boundary is "can this caller reach the server's bind address at all," same as it
  already implicitly is for `repo:`.
- No single-serve-per-vault guard (pidfile/lockfile refusing a second concurrent `engram
  serve` against the same vault). Considered and dropped — see Decisions.
- No change to `cli.HTTPPrims.Fetch`'s signature or to any transport primitive. Trusting a
  body field needs nothing at the HTTP layer that isn't already there.

## Decisions

**Client-declared identity, not a credential map or a trusted-header adapter.** #726 proposed
identity as a pluggable port with two adapters: a baseline bearer-token→identity map (server
config) and an opt-in trusted-header mode (generalizing Cloudflare Access to any authenticating
proxy). Both were designed through and explicitly set aside in favor of a third, simpler option:
treat `user:` exactly like `repo:` already is — a value the calling instance detects locally
and the server takes as given. Rationale: identity here has never been about verifying "is this
request cryptographically who it claims" — the archived design's own reason to prefer a server
over a filesystem mount was attribution ("a mount can't attribute a write to who made it"), and
attribution only requires that *some* stable, caller-chosen name lands on the note, not that the
name be adversarially unforgeable. A credential map adds real weight (config shape, secret
storage/rotation, `ps`/shell-history exposure if delivered as a flag) to buy a guarantee
(spoofing resistance) the design doesn't actually need once the trust boundary is understood to
be network reachability rather than cryptographic identity. A trusted-header mode remains a
reasonable future adapter for an operator who *does* want that stronger guarantee — nothing in
this change forecloses adding it later behind the same identity resolution point
(`serveLearn`/`serveAmend`'s body-derived `args.User`); it just isn't built now.

**Reject only empty, verify nothing else.** The floor is "you must claim something," not "your
claim must be checkable." An empty string is the one value that can't be a real identity (every
`detectUser` fallback chain bottoms out at `""`, never at a placeholder name), so it's the one
value cheap and unambiguous to reject without inventing validation rules for what a "valid"
identity string looks like.

**No single-serve-per-vault pidfile guard.** #726's second ask, and the archived design's own
stated reason two servers would be "the one genuinely wrong topology," was three-fold:
independent listeners, independent credential maps, and independent pending-offer surfacing.
Dropping the credential map (this change) removes the middle third of that argument entirely.
The other two don't hold up independently: ADR-0013's flocks are what make concurrent writers
correct in the first place, regardless of process count — ingest/`RunLearn`/`RunAmend`/
`RunActivate` all lock around their read-modify-write today, precondition already verified in
the archived change's own design.md. And pending-offer surfacing is a stateless re-scan (`query`
payload, `update` notice, write-path log nudge — all computed fresh each time, "same shape as
`notesMissingIdentityFields`" per the archived design), not state a server process holds, so two
processes can't disagree about it. What's left if two `engram serve` instances run against the
same vault is redundant resource use (two copies of the embedding model in memory) and the
operator's own confusion about which port a client should hit — real, but an operational
concern, not a correctness one, and not addressed here.

**Two-doors model is documentation, not new mechanism.** The behavior it describes (local
writes commit immediately; host-only verbs — `ingest`, `vocab refit`, `prune`, `check`,
`update`, `resituate` — never get a route; served writes land as pending offers; both paths
share `Run*` functions and ADR-0013's flocks) is exactly what the archived change already
built. This change's docs work is transcription and framing, not a behavior change — no spec
delta accompanies it.

## Risks / Trade-offs

- **[A reachable-but-unintended caller can claim any identity]** → Accepted. This is the same
  risk class the archived design already accepted for the Cloudflare header path ("header
  spoofing if the origin is reachable outside Cloudflare Access... this design does not verify
  the JWT and does not enforce origin unreachability in code — both are deployment-level
  mitigations"), just generalized: the mitigation was never really "Cloudflare verifies this,"
  it was "don't expose the bind address beyond who you trust." Removing Cloudflare from the
  identity path doesn't remove a guarantee that was actually load-bearing; it removes a
  guarantee-shaped piece of infrastructure that only some deployments have.
- **[No forward path signal for a stronger identity mode]** → Mitigated by construction: the
  identity resolution point stays a single seam (`args.User` on the unmarshaled request body,
  read by `serveLearn`/`serveAmend`). A future trusted-header or token-map adapter composes at
  that same seam without restructuring anything this change ships.
- **[Two `engram serve` instances against one vault stay possible]** → Accepted, per the
  Decisions section above — no longer a credential-map correctness question, and the
  flock-covered writes are safe either way.

## Migration Plan

No stored data changes shape. Existing notes' `user:` fields are unaffected. A deployment
currently relying on the Cloudflare-header requirement to gate served writes loses that gate
the moment this change ships — if the server's bind address was reachable by anyone who isn't
supposed to write, it already needed a network-level fix regardless of this change (see Risks).
No rollback concerns beyond a normal revert; the `User` field addition to `LearnArgs`/
`AmendArgs` is additive and ignored by any caller that doesn't set it (falls back to `""`,
which now gets rejected server-side — the same outcome as today's missing-header case, just a
different missing-input).
