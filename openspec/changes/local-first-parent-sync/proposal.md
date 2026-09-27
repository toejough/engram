## Why

Engram has two mutually exclusive ways to reach a parent vault. `ENGRAM_SERVER` turns the CLI into a thin HTTP client with no local vault, so its writes land only on the parent. `ENGRAM_PARENT` merges the parent's results into a local query read-only, and its writes stay only local. Neither mode does what an environment needs: keep its own memory, share what it learns upward, and pull down what it actually uses. A host configured with `ENGRAM_PARENT` can even run for weeks with no local vault at all (#766). On 2026-09-27 Joe chose one local-first, symmetric model to replace both modes (vault note 784a, which updates note 784's exchange model), and settled the follow-up questions the same day (design D12 and Context).

## What Changes

- **BREAKING**: remove the thin-client `ENGRAM_SERVER` mode. When `ENGRAM_SERVER` is set, **every** `engram` command, `engram serve` included, exits non-zero before any work, with an error telling the user to set `ENGRAM_PARENT` to the same URL. There is no compat shim and no exemption. `engram serve` itself still exists as the parent side, reached by children through `ENGRAM_PARENT`.
- Every environment always has its own local vault. The first run of any command that resolves the vault path creates it, and a one-line notice says so (folds in #766).
- Each vault gains a stable **vault ID**, stored in a tracked file `.engram-vault-id`. Transient exchange state (the outbox, declined pulls, and the parent cache with backoff) lives in `<vault>/.engram/`, which ignores itself through its own `.gitignore`. The vault's tracked root `.gitignore` is never modified.
- `engram learn` writes locally and then **offers** the note to the parent. A content-changing `engram amend` or `engram resituate` is offered too: as an amend of the linked parent counterpart when one exists, otherwise as a new note.
  - Bookkeeping stays local: `--activate`, `--clear-pending`, `--discard`, identity backfill, `learn qa`, and `--supersedes`/`--chunk-source`-only amends. Joe accepted this split.
  - One exception to the bookkeeping rule: accepting a served offer (`--clear-pending` on it) is itself offered further up, which is multi-level propagation (Joe).
- A **local outbox** holds offers the parent could not take. It drains after the next successful parent contact, and backs off while the parent is unreachable. Offers are idempotent, and nothing is lost offline.
- Change detection uses a new **exchange hash** that covers every offered content field (situation, the fact, feedback or runbook fields, and the body), not `embed.ContentHash`.
- Notes record **links** to their parent counterparts, keyed by the parent's vault ID. Links are multi-valued: one counterpart plus the parent notes this note is known to cover. Notes also carry a rename-stable exchange ID (`xid`) and an `aliases` list that renames and curation folds extend. Every non-exchange frontmatter rewrite preserves all of these.
- Every `engram query` with `ENGRAM_PARENT` set merges local results with the parent's **notes only**. Parent chunks and the whole parent chunk path are dropped, including `show-chunk --parent`. The query dedupes, keeping the local note, and ranks direct matches ahead of explore picks. This fixes #744 (explore picks crowd out direct matches) and #743 (the budget block is all zeros). `pending_offers` reflects local offers only.
- `engram activate` on a parent-only note (recall's "used" signal) fetches it through a JSON raw-show envelope and queues it as a **pending offer in the local vault**. It also bumps the note's use on the parent, best-effort. Local curation decides whether to keep it. A pulled note is never offered back unless it is edited locally, and a near-fold bounce back up once is intended (Joe).
- `engram serve` changes:
  - the served set shrinks to `query`, `show`, `activate` and `learn`, and `/amend`, `/query-chunks` and `/show-chunk` are deleted;
  - `/learn` places offers at top level, updates a pending offer from the same origin in place, and returns a receipt with the basename, the vault ID and the resolved target;
  - every note type marked `pending: true` counts as pending;
  - remote requests cannot set link or identity fields.
- Curation gains `engram amend --discard --into <existing>`. Bookkeeping amends stop re-stamping `repo`/`user`/`vault`, so an accepted offer keeps its author. `resituate` preserves every field it doesn't change.
- Transcript chunks never travel. Only notes do.
- The recall, learn and curate skills are updated through `superpowers:writing-skills` TDD, using hermetic headless arms.
- Supersedes #746. #745 is resolved by design (design D9).

## Capabilities

### New Capabilities
- `vault-local-first`: every environment's own vault. Covers creation on first use, the vault ID and exchange state directory, the self-parent guard, and the `ENGRAM_SERVER` hard error.
- `vault-parent-offers`: the upward edge. Covers the offer classification, the exchange hash, payload translation, the outbox, drain, backoff, idempotency, receipts, reporting, and multi-level propagation.
- `vault-parent-pulldown`: the downward edge. Covers activate's local-first resolution, pull-down as a local pending offer, the skip and decline rules, loop freedom, and the best-effort parent bump.

### Modified Capabilities
- `vault-merged-recall`: merge is the standard path; notes only, with no parent chunks; dedupe before ranking; direct matches ahead of explore picks; the note floor; the merge-applied budget; the parent backoff; and `show`'s parent routing without `show-chunk`. Every `ENGRAM_SERVER` clause is removed.
- `vault-serve-api`: the thin-client requirement is removed. The served set shrinks. `/learn` gains the offer semantics and the extended receipt. Raw show returns a JSON envelope, and `/query` gains dedupe keys.
- `vault-offer-curation`: offers arrive from both directions. Pending detection covers every note type. Folds use `--discard --into`. The merged hint reflects local offers only.
- `vault-note-identity`: adds `xid`, multi-valued parent links, `aliases`, rename aliases, and their survival requirement, and blocks them on the wire. Bookkeeping amends no longer re-stamp identity.
- `recall-payload-cuts`: removes its `ENGRAM_SERVER` mention.

## Impact

- **Fleet**: every environment that already sets `ENGRAM_PARENT` will, after upgrading, offer every learn and content amend to the host. The host's curation queue (`pending_offers`) grows by the fleet's write rate, and curating it is expected upkeep. Hosts that set `ENGRAM_SERVER` get the hard error and must switch to `ENGRAM_PARENT`. Their first command creates an empty local vault.
- **Real vault**: the first `engram serve` start, or the first parent contact, after upgrading creates a tracked `.engram-vault-id`, which needs one deliberate vault commit.
- **Code** (`internal/cli`, `cmd/engram`):
  - existing files: `serve_client.go`, `serve.go`, `targets.go`/`deps.go`/`primitives.go`, `learn.go`, `amend.go`, `activate.go`, `resituate.go`, `merged_query.go`, `show.go`/`show_chunk.go`, `offer.go`, `vault_init.go`/`qa.go`, `luhmann_reparent.go`, `skillreg_accept.go`, `identity_backfill.go`, `update.go`, and the frontmatter structs;
  - new: an exchange-hash function, the outbox and state adapters, and pull-down;
  - all of it through DI (ADR-0013, the thin-API check).
- **HTTP API**: `/learn` gains fields. `/show?raw=1` (JSON envelope) and `/query?dedupe-keys=1` are new. `/amend`, `/query-chunks` and `/show-chunk` are removed. Upgrade the parent first.
- **Docs**: README, GLOSSARY, FEATURES, ADR (new ADR-0029), C1/C2/C3, ROADMAP, LEDGER. See enumeration.md.
- **Skills**: `agent-instructions/skills/{recall,learn,curate}/SKILL.md`, deployed only after merge and install.
- **Issues**: closes #766, #743 and #744; closes #745 (by design) and #746 (superseded).
