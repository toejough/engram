## Why

Engram has two mutually exclusive ways to reach a parent vault. `ENGRAM_SERVER` turns the CLI into a thin HTTP client with no local vault, so its writes land only on the parent. `ENGRAM_PARENT` merges the parent's results into a local query read-only, and its writes stay only local. Neither mode does what an environment needs: keep its own memory, share what it learns upward, and pull down what it actually uses. A host configured with `ENGRAM_PARENT` can even run for weeks with no local vault at all (#766). On 2026-09-27 Joe chose one local-first, symmetric model to replace both modes (vault note 784a, which updates note 784's exchange model).

## What Changes

- **BREAKING**: remove the thin-client `ENGRAM_SERVER` mode. When `ENGRAM_SERVER` is set, every `engram` command except `serve` exits non-zero before any work, with an error telling the user to set `ENGRAM_PARENT` to the same URL. No compat shim is provided. `engram serve` (the parent side) is unchanged apart from the additions below.
- Every environment always has its own local vault. The first run of any command that resolves the vault path creates it, and a one-line notice says so (folds in #766).
- `engram learn` writes locally and then **offers** the note to the parent (`ENGRAM_PARENT`). A content-changing `engram amend` is offered too: as an amend of the parent counterpart when an origin link is recorded, otherwise as a new note. Bookkeeping amends stay local: `--activate`, `--clear-pending`, `--discard`, and identity backfill. Treating identity backfill as bookkeeping is an assumption, stated in design D5.
- A **local outbox** holds offers the parent could not take. It is drained after the next successful parent contact by any command (`learn`, content `amend`, `resituate`, `query`, `activate` or `update`). The local write always succeeds, and nothing is lost offline.
- A new **origin link** is recorded in both directions: a local note records its parent counterpart, and a note accepted from the parent records where it came from. Every frontmatter rewrite path preserves it.
- Every `engram query` with `ENGRAM_PARENT` set merges local and parent results, dedupes (keeping the local note), and then ranks. "Same note" means a recorded origin link first, then an identical content hash. Near-duplicates are left to curation.
- `engram activate` on a parent-sourced note (recall's "used" signal) fetches that parent note and queues it as a **pending offer in the local vault**. Local curation (covered/near/absent) then decides whether to keep it, the same offer/accept edge the parent uses, run in the other direction. A note that came from the parent is never offered back up.
- `engram serve`'s `/learn` response gains the offered note's basename and an explicit `pending` flag, so the child can record the counterpart. Amend-offers travel over `/learn` with an `offer.for` target. `POST /amend` is removed: its only caller was the thin client, and it hides live notes. Any note type marked `pending: true` now counts as pending; today a runbook counts only when it also carries `skill_hash`. Remote-set fields stay blocked: the origin fields are tagged `json:"-"` on the wire, like the skill fields.
- Curation gains `engram amend --discard --into <existing>`. It records the discarded offer's basename as an alias on the note it was folded into, which keeps the offering side's link resolvable.
- Merged query fixes #744 (explore picks crowd out direct matches) and #743 (the budget block is all zeros).
- Transcript chunks never travel. Only notes do.
- The recall, learn and curate skills are updated through `superpowers:writing-skills` TDD so that they handle parent-sourced items, the offer-on-learn behavior, and pending offers that arrive from the parent.
- Supersedes #746 (`--parent` on activate, learn and amend). #745 (parent clusters dropped) is resolved by design rather than by code; see design D9.

## Capabilities

### New Capabilities
- `vault-parent-offers`: the upward edge. Covers offer-on-learn and offer-on-amend, the local outbox with its retry and drain behavior, idempotency, and failure reporting.
- `vault-parent-pulldown`: the downward edge. Covers activating a parent-sourced note, which queues it as a local pending offer, and the rule that such a note is never offered back up.
- `vault-local-first`: every environment's own vault. Covers vault creation on first use and the hard error for `ENGRAM_SERVER`.

### Modified Capabilities
- `vault-merged-recall`: merged query becomes the standard path whenever `ENGRAM_PARENT` is set. It dedupes before ranking and keeps the local copy of a duplicate. Every `ENGRAM_SERVER` precedence and exclusivity clause is removed.
- `vault-serve-api`: the "transparent HTTP client when ENGRAM_SERVER is set" requirement is removed. `amend` leaves the served set. `/learn` accepts offer targets and idempotency keys, ignores the client's placement, and returns the basename and a pending flag. `/show` gains a raw mode, and `/query` gains an opt-in `dedupe-keys` mode.
- `vault-offer-curation`: pending offers can now also arrive from the parent (pull-down). Pending detection covers every note type. Curation folds with `--discard --into`.
- `vault-note-identity`: adds origin-link fields and a requirement that they survive every frontmatter rewrite path, and adds a content hash for dedupe.
- `recall-payload-cuts`: removes its `ENGRAM_SERVER` mention (see enumeration.md).

## Impact

- **Code** (`internal/cli`, `cmd/engram`): `serve_client.go` (the thin-client fetchers are deleted, and the parent fetchers are extended), dispatch in `targets.go`/`deps.go`/`primitives.go`, `learn.go`, amend, activate, `merged_query.go`, `show.go`/`show_chunk.go`, `serve.go`, vault init (`vault_init.go`, `qa.go`), frontmatter struct(s), and a new outbox adapter. Every change goes through DI, per ADR-0013 and the thin-API check.
- **HTTP API**: the `/learn` receipt gains fields (additive). `/show?raw=1` and `/query?dedupe-keys=1` are new, and `POST /amend` is removed. Upgrade the parent first.
- **Docs**: README, GLOSSARY, ADR (a new amendment), C1/C2/C3 diagrams, FEATURES, ROADMAP, LEDGER. See enumeration.md for the verified list.
- **Skills**: `agent-instructions/skills/{recall,learn,curate}/SKILL.md`, each through writing-skills TDD.
- **Users**: hosts that set `ENGRAM_SERVER` must switch to `ENGRAM_PARENT`, and the hard error tells them how. Their first command after upgrading creates an empty local vault.
- **Issues**: closes #766; supersedes #746; dispositions for #743, #744 and #745 are in design D9.
