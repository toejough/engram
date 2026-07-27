## Why

Vocab definition notes carry only the bare `vocab` marker tag, while member notes carry `vocab/<term>` tags — so in the Obsidian graph and tag pane, a term's definition note and its member cluster are disconnected: clicking `vocab/cli-verification` shows every member but never the note that defines the term. The user wants each definition connected to its own term's tag cluster.

## What Changes

- Every vocab definition note additionally carries its own term tag (`vocab/<term>` alongside the existing bare `vocab` marker) — e.g. `vocab-cli-verification-definition` gains `vocab/cli-verification`.
- Definition minting (`vocab bootstrap`, refit-minted definitions) writes the self-tag at creation time.
- A new explicit, idempotent one-shot subcommand backfills the self-tag onto the ~29 existing definition notes (never a side effect of `engram update` — destructive/mutating ops stay explicit user commands).
- Engram's member semantics are explicitly preserved: definition notes remain excluded from term-member counts (`vocab stats`, refit triggers, untagged-rate math), tag nomination pools, and route-evidence count audits, via the existing bare-`vocab` definition marker (`isVocabDefinitionNote`). The self-tag is an Obsidian-graph affordance only.
- The vault's own convention record (note `236.2026-07-10.vocab-definition`) is updated at apply time to state the new convention.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `vault-vocab-lifecycle`: the definition-note tagging requirement changes — definitions SHALL carry `vocab/<term>` in addition to the bare `vocab` marker; member-semantics (assignment exemption, stats/trigger counts, tag nomination) SHALL continue to exclude definition notes via the bare marker.

## Impact

- Code: `internal/cli/vocab_commands.go` (definition minting — currently pins "Tagged bare `vocab` only — never `vocab/<term>`" at :720; the new backfill subcommand + targets wiring), `internal/cli/vocab.go` (definition recognition / tag helpers), and every member-semantics read site that would otherwise start counting definitions as members: `vocab stats`/trigger math (`vocab_trigger.go`), tag nomination (`query_nominations.go`), and `count.go` group-by results (definitions will now appear under `tags=vocab/<term>` filters — the count-as-audit workflow reads `work-kind/tier/outcome` tags, unaffected, but vocab-tag group-bys will include definitions unless filtered; the design decides and documents this surface).
- Vault data: one-shot additive tag rewrite across ~29 definition notes (tags-only edit; embeddings unaffected since tags are not embedded content — verify in design).
- Tests: imptest/gomega unit coverage for minting, backfill idempotency, and each preserved-exclusion site.
- Docs: `openspec/specs/vault-vocab-lifecycle/spec.md` via delta; vault note 236 amended at apply.
