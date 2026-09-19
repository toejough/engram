## Why

`agent-instructions/guidance/shim.md` (the runbook-follow-frame guidance) is built, tested, and
deployed by `engram update --with-guidance` to every harness's engram-owned root — but no real
session's `CLAUDE.md`/`AGENTS.md` actually imports it yet. Confirmed this session: `route`'s
promoted production runbook (from `route-skill-to-runbook`, closing #757) now clears its D8
behavioral bar with `shim.md`'s follow-frame in a shim-only eval harness, but that fix has zero
effect on any real session — including this one — until `shim.md` is actually wired into a real
guidance-import list. Without this, every runbook conversion this session did (route, and future
please/curate/openspec ones) stays confined to eval fixtures indefinitely.

## What Changes

- Activate `shim.md` for real sessions by adding its import alongside the three existing engram
  guidance imports (`recall.md`, `delegate.md`, `learn.md`) — **additive, not a replacement**.
  `shim.md`'s own header comment already states this is the correct shape for now: *"NOT a
  replacement for recall.md/delegate.md/learn.md, which stay as-is for sessions that still install
  the recall/learn/write-memory skills."* Those three skills are not yet promoted to runbooks
  (that's issue #760's separately-deferred, larger scope) and `shim.md`'s own body does not
  reproduce their content (recall's 10-phrase/clustering/coverage-judgment procedure, learn's
  four-kind crystallization scan, delegate's orchestration-vs-object-level routing doctrine) — a
  full replacement is not yet possible without that promotion happening first.
- No code or deployment-mechanism change: `engram update --with-guidance` already syncs `shim.md`
  to each harness's engram-owned root exactly like the other three guidance files (verified this
  session: `~/.claude/engram/guidance/shim.md` deployed, `~/.claude/engram/shim.md` compat symlink
  present, byte-identical to the repo source). The only missing step is the manual import line in
  each harness's own config file — `engram update` does not, and per `update-deploy-sync`'s own
  "Guidance opt-in gates management, not removal" requirement should not, auto-edit those files.
- Documents the exact import line(s) to add, for the orchestrator (not this change's own
  artifacts) to apply directly to the user's personal `~/.claude/CLAUDE.md` and, if in scope for
  this harness, `~/.pi/agent/`'s `AGENTS.md` equivalent.

## Capabilities

### New Capabilities

- `production-guidance-activation`: which engram guidance files a real harness config
  (`CLAUDE.md`/`AGENTS.md`) is expected to import — distinct from `update-deploy-sync`'s concern
  (getting the files onto disk), this covers the policy decision of which deployed files a real
  session's config actually activates. States that `shim.md` joins `recall.md`/`delegate.md`/
  `learn.md` in that set.

### Modified Capabilities

(none — `update-deploy-sync`'s existing sync behavior already covers deploying `shim.md` to disk
correctly; this change does not alter that requirement. See design.md D2 for the verification.)

## Impact

- **Affected file (outside this repo):** the user's personal `~/.claude/CLAUDE.md` (and
  `~/.pi/agent/`'s `AGENTS.md`, if in scope) — a manual edit, not a repo artifact this change
  applies itself.
- **Not affected:** `agent-instructions/skills/{recall,learn,write-memory}/` (still skills;
  promotion is #760's scope), `internal/update/` (no code change — the sync mechanism already
  handles this file the same as the other three guidance files).
- **Behavioral consequence once activated:** any `kind: runbook` note (route's, once its own
  retirement gate clears — see `route-skill-to-runbook`'s design.md D4) becomes followable via
  `shim.md`'s frame in real sessions, not only inside the eval harness's shim-only fixture.
