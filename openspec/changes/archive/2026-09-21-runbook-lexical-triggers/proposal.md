## Why

Runbooks are only worth carrying if they fire reliably when their moment comes. Today the only way a
runbook surfaces is semantic similarity between its `situation:` line and two phrases the agent
*paraphrases* from the user's request. The `please-skill-to-runbook` change (parked 2026-09-20)
measured this path on a real `/please` ask: agents dropped "please" and "end-to-end" from their
paraphrase in 3/3 trials and named the deliverable instead, so the runbook never surfaced; after a shim
fix the runbook still missed 1/3 trials and misses whenever the paraphrase names the deliverable;
adding the user's verbatim words as a *third semantic phrase* changed the runbook's rank in 0/95
scratch-vault cells. Semantic matching on a paraphrase cannot reproduce what a slash command does —
fire every time on a literal cue. Without that, no skill whose trigger is a cue word (`/please`,
`/recall`, `/learn`, `/curate`, the openspec `opsx:*` commands) can be retired into a runbook.

## What Changes

- **New runbook frontmatter field `triggers:`** — an optional list of literal strings (e.g.
  `"/please"`, `"take this end-to-end"`). Populated by a repeatable `--trigger <text>` flag on
  `engram learn runbook` and `engram amend` (replace-whole semantics, like `--red-flag`).
- **New `engram query --text "<raw user message>"` input** — the user's message verbatim, separate
  from the semantic `--phrase` values. Every runbook whose `triggers:` contains a case-insensitive
  substring match of `--text` is a *trigger hit*.
- **Trigger hits surface first, always** — a trigger-hit runbook appears in the payload's `items[]`
  ahead of every similarity-ranked item, regardless of embedding score, relevance floor, cap, or
  `--limit`, with a new provenance role `trigger`. Runbooks with no trigger hit are unaffected and
  still compete on similarity as today.
- **Shim update** — the shim's first-action `engram query` block gains `--text` with the instruction
  to paste the user's message word for word (not a paraphrase). The two semantic phrases stay.
- **write-memory template update** — the runbook command template documents `--trigger`, with
  guidance that triggers are specific, command-like cues (never a bare common word like "please").
- **Validation** — a headless RED/GREEN measures whether agents actually pass `--text` verbatim; the
  parked please eval reruns with `triggers: ["/please", "take this end-to-end"]` on the promoted
  please runbook; bar = runbook surfaced in every trial whose ask says `/please`.

Not in scope: a prompt-submit hook that runs the query on the raw prompt without the agent (option B
in the 2026-09-20 discussion — a separate change if `--text` fidelity proves insufficient); regex or
fuzzy triggers; triggers on `fact`/`feedback` kinds; retiring any skill (that is each skill's own
change, unblocked by this one).

## Capabilities

### New Capabilities

- `runbook-lexical-triggers`: literal trigger strings on runbook notes, matched against raw user text
  supplied to `engram query --text`, surfacing hits ahead of similarity-ranked items.

### Modified Capabilities

- `recall-runbook-surfacing`: the requirement "Runbook notes SHALL rank purely by
  situation-similarity" gains one exception — trigger hits are placed first; all other ranking is
  unchanged.
- `learn-runbook-capture`: `engram learn runbook` gains `--trigger` (parallel to the existing
  `red_flags` requirement).
- `guidance-runbook-follow-frame`: the first-action query carries `--text` with the user's verbatim
  message in addition to the two phrases.
- `write-memory-worker`: the runbook command template accepts an optional `triggers` handoff field
  → repeated `--trigger` flags.
- `recall-payload-cuts`: the "Limit caps the relevance-ranked item count" requirement gains an
  exemption — trigger hits are placed first and do not count against `--limit`.
- `vault-merged-recall`: the merged-budgets requirement documents that trigger hits from local and
  parent lead the merged `items[]` (local before parent) and are exempt from `--limit`.
- `vault-serve-api`: new requirement — the served `query` route round-trips `text`, silently capped
  at 2 KB on a UTF-8 rune boundary.

## Impact

- **Go code (`internal/cli/`)**: `query.go` (new `--text` arg, trigger index, provenance role,
  ordering exemption), `query_vault_meta.go` (parse `triggers:` during the existing whole-vault meta
  pass), `learn.go` / `targets.go` / `amend.go` (field + flags + render), `serve_client.go` /
  `serve.go` (round-trip `text` on the served query path), `export_test.go`. `cmd/engram/main.go`
  is untouched (wiring-only).
- **Sidecars**: `embed.ContentHash` covers only `situation:` + body, so adding/editing `triggers:`
  does NOT stale `.vec.json` — no re-embed migration.
- **Guidance/skills**: `agent-instructions/guidance/shim.md` (query block),
  `agent-instructions/skills/write-memory/SKILL.md` (template). Deploy via `engram update
  --with-guidance` (note: that command also reinstalls the binary from the current checkout).
- **Specs**: deltas for `recall-payload-cuts`, `vault-merged-recall`, `vault-serve-api` (in addition
  to the four capabilities above).
- **Docs**: `docs/GLOSSARY.md` (items[], provenance), `docs/architecture/c3-components.md` (K6
  payload shape), `docs/architecture/adr.md` (amend ADR-0026's "no dedicated ranking mechanism"
  rationale with the measured failure).
- **Eval**: reuses `dev/eval/cumulative/runbook_vs_skill/phase2/` (please task, probe harness) and
  the headless RED/GREEN pattern from the parked change's shim validation.
- **Unblocks**: `please-skill-to-runbook` (#758) retirement; the same mechanism applies to #759
  (curate), #760 (recall/learn/write-memory), #761 (opsx).
