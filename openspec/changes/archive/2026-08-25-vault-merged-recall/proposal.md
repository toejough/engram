## Why

An app-environment agent's `engram query` can search its own local vault, or a
configured `ENGRAM_SERVER`, but never both in one call — there is no way to
combine "what I've learned locally" with "what the host vault already knows"
into one ranked answer. This blocks phone-llm#31 (octopus mode: every app
environment runs a full local engram instance plus an edge to the host's
vault), whose read edge needs exactly that fused view. `vault-query-model-
provenance` already ships `model_id` on every query payload for exactly this
future consumer to use; this change is that consumer.

## What Changes

- New `ENGRAM_PARENT=<url>` config: names this node's single parent vault.
  This is additive, not exclusive. `ENGRAM_SERVER` (existing, #720) is a hard
  toggle: set, the CLI runs entirely against that remote over HTTP with zero
  local file access; unset, everything runs entirely against the local
  vault — never both. `ENGRAM_PARENT` doesn't replace or extend that toggle;
  it adds a remote-fusion step on top of the local pipeline (local AND
  remote, rather than local XOR remote). Every node has at most one parent
  (the vault-graph is a tree — personal → team → org — never a mesh), so a
  single URL is the complete config surface; no list, no N-remote fan-out.
- `engram query`, when `ENGRAM_PARENT` is set, runs its existing local query
  pipeline unchanged AND fetches the parent's existing `/query` endpoint
  (reusing the served `/query` route as-is — no new server route, no change
  to that endpoint's byte-identical-output contract), then interleaves both
  result sets into one ranked payload by score.
- The merge is unconditional: it does not gate, refuse, or fall back on
  `model_id` mismatch between local and parent. Score comparison across
  differently-versioned embedding models is an inherent approximation (raw
  cosine ranges aren't linearly rescalable into comparability — evaluated and
  rejected during design), so merged mode accepts the approximation rather
  than breaking recall across a fleet mid model-upgrade-rollout.
- Merged-query items each carry their source node's `model_id` (new
  per-item field, merged-payload-only) as a visible, non-gating advisory
  signal — consistent with the existing non-fatal `warnModelMismatch`
  pattern for local stale sidecars.
- **BREAKING**: `--limit` becomes a real cap on `engram query`'s returned
  `items[]` count, in every mode — local, `ENGRAM_SERVER`-exclusive, and
  merged alike. Today it is report-only metadata (`Budget.Limit` in the
  payload); nothing actually truncates `items[]`, which is sized instead by
  clustering/candidate-nomination knobs (`matchPhraseLimit`=30 per phrase,
  `matchSetCap`=300 total). Discovered while implementing the merge: capping
  only merged-mode output would leave `--limit` meaning two different things
  depending on mode, and merging two sources' independently-nominated
  candidate sets needs *some* real bound to keep total payload size sane —
  the same problem the content-budget/recent-fill fix below addresses for
  content and recency, just for item count. Fixing it once, everywhere, was
  chosen over a merge-only special case. Existing callers (notably the
  `recall` skill, and `dev/eval/LEDGER.md`'s payload-size baselines) will
  see fewer items by default (20) than they do today — see design.md
  Migration Plan.
- Write-side (`learn`/`amend`) is untouched — single-target push via
  `ENGRAM_SERVER` already covers the octopus-mode write edge, and the
  single-parent topology means multi-target push has no destination to target
  even in principle.
- Merged-query items each carry a `from_parent` boolean so the agent can
  tell which items came from the parent — needed because a merged result
  is only useful if the agent can safely follow up on it.
- `engram show` and `engram show-chunk` gain a `--parent` flag: when set
  (and `ENGRAM_PARENT` is configured), that one lookup routes to the
  parent's existing `/show`/`/show-chunk` endpoint instead of the local
  vault. No new server route — both are already in the fixed served
  command set (`vault-serve-api`). Without `--parent`, both commands are
  unchanged. This isn't optional polish: a note ref (bare Luhmann id) is
  minted purely from each vault's own local id set with no cross-vault
  coordination, so the same ref routinely names unrelated notes in two
  different vaults — a "try local, fall back to parent" lookup could
  silently return the wrong note's content with no error. Explicit
  per-item origin plus explicit routing avoids that.

## Capabilities

### New Capabilities
- `vault-merged-recall`: `engram query`'s `ENGRAM_PARENT`-gated local+parent
  result fusion — config surface, fetch/merge behavior, per-item `model_id`
  and `from_parent` tagging, the unconditional-accept mismatch policy, and
  `--parent`-routed follow-up lookups on `show`/`show-chunk`.

### Modified Capabilities
- `recall-payload-cuts`: gains a new requirement that `--limit` really caps
  `items[]` count, alongside the `--lazy-chunks`/`--recent-fill` flags this
  capability already owns (`vault-serve-api`'s `/query` route and
  output-parity contract are otherwise consumed as-is by the parent fetch;
  `vault-query-model-provenance`'s payload-level `model_id` requirement is
  unchanged for the non-merged case)

## Impact

- `internal/cli/query.go`: new merge orchestration — run local `RunQuery`,
  fetch the parent's payload, interleave by score into one payload. Also:
  `renderQueryPayload` gains a real `--limit` truncation step, applying to
  every mode, not just merged.
- `internal/cli/serve_client.go`: today's `fetchQuery` pipes the parent's
  response bytes straight to stdout (`fetchAndCopy`) for the exclusive-mode
  case; merged mode needs the parent's payload parsed into a Go value before
  it can be interleaved with the local payload, so this needs a new
  decode-and-return variant alongside the existing byte-copy one.
- `internal/cli/targets.go`: `engram query`'s dispatch gains a third branch
  (today: `ENGRAM_SERVER` set → remote-only, else → local-only) for
  `ENGRAM_PARENT` set → local+parent merged.
- No `internal/cli/serve.go` (server-side) changes — the parent is queried
  as an ordinary, unmodified `/query`, `/show`, or `/show-chunk` client.
- `internal/cli/show.go` / `show_chunk.go`: new `--parent` flag; when set,
  route through the parent fetch path instead of local `Scan`/`Read`.
- `internal/cli/serve_client.go`: `fetchShow`/`fetchShowChunk` already
  exist for `ENGRAM_SERVER`-exclusive mode; `--parent` needs the same
  request logic pointed at `ENGRAM_PARENT` instead, selected per-call
  rather than by a whole-process env switch.
- `README.md`: document `ENGRAM_PARENT` alongside the existing
  `ENGRAM_SERVER` documentation, including the `--parent` flag on
  `show`/`show-chunk`.
