# Doc-surface enumeration: fix-frontmatter-remainder

This change alters these documented behaviours:

- amend converts the CRLF notes it writes to LF instead of refusing them;
- identity backfill keeps keys it does not set, refuses an anchored key it would set, and stamps CRLF notes (as LF) instead of skipping them;
- a refused offer receipt puts its outbox entry in the `attention` state;
- `engram update`'s outbox notice reports `attention` entries;
- exchange readers, `engram show`'s header included, read a note as its LF form;
- amend keeps unknown sub-keys of surviving `supersedes:` entries;
- no write produces `user: ""`;
- every note reader (embedding, query, vocab, check, count) reads a CRLF note as its LF form;
- no offer is sent without a user identity;
- no write produces `vault: ""`;
- the backfill notice counts only notes backfill can stamp now; a no-user `attention` entry returns to queued once its payload builds; kept receipts are recorded before the backoff gate, and one from another parent is discarded (review fixes).

**Search** (2026-10-02, worktree `runbook-vs-skill`, branch `fix-789-frontmatter-remainder`):

```
grep -rn -i -E "crlf|backfill-identity|identity backfill|outbox|receipt|rejected|non-goal|out of scope|exchange_hash|user:|supersedes" \
  --include='*.md' agent-instructions docs README.md openspec/specs
```

**Excluded:** `openspec/changes/archive/` (history), `dev/eval/` (frozen fixtures), `docs/research/` (dated notes), `docs/superpowers/` (dated plans).

**Disposition key:** *rewrite*, *append*, *no change* (with the reason), *spec delta* (this change's `specs/`, applied at archive).

| # | Surface | Disposition | New-present grep | Old-absent grep |
|---|---|---|---|---|
| 1 | `docs/ROADMAP.md` row 9 | rewrite: #789 done on this branch, pending merge, naming all three fixes and backfill's CRLF conversion | `grep -n "fix-789-frontmatter-remainder" docs/ROADMAP.md` | — |
| 2 | `docs/GLOSSARY.md` `### outbox` | append: the `attention` state | `grep -n '`attention`' docs/GLOSSARY.md` | — |
| 3 | `docs/architecture/adr.md` D6 | append: one sentence on the `attention` state | `grep -n '`attention`' docs/architecture/adr.md` | — |
| 4 | `openspec/specs/vault-note-identity/spec.md` "Exchange fields SHALL survive every frontmatter rewrite" | spec delta: amend converts CRLF; fold converts a CRLF existing note | `grep -n "Amend converts a CRLF note" openspec/changes/fix-frontmatter-remainder/specs/vault-note-identity/spec.md` | — |
| 5 | `openspec/specs/vault-note-identity/spec.md` "Backfill for pre-existing notes missing identity fields" | spec delta: node edit, anchor refusal, CRLF conversion | `grep -n "Backfill converts a CRLF note" openspec/changes/fix-frontmatter-remainder/specs/vault-note-identity/spec.md` | — |
| 6 | `openspec/specs/vault-parent-offers/spec.md` receipt and outbox-report requirements | spec delta: the `attention` state and its report | `grep -n "A refused receipt needs attention" openspec/changes/fix-frontmatter-remainder/specs/vault-parent-offers/spec.md` | — |
| 6b | `openspec/specs/vault-offer-curation/spec.md` "engram show SHALL print an exchanged note's exchange hash" | spec delta: CRLF notes get the header | `grep -n "Header on a CRLF note" openspec/changes/fix-frontmatter-remainder/specs/vault-offer-curation/spec.md` | — |
| 6c | `openspec/specs/vault-parent-offers/spec.md` | spec delta (ADDED): exchange readers read a note as its LF form | `grep -n "Exchange readers SHALL read a note as its LF form" openspec/changes/fix-frontmatter-remainder/specs/vault-parent-offers/spec.md` | — |
| 6d | `openspec/specs/vault-note-identity/spec.md` "User field auto-detected at note creation", "Amend re-stamps identity fields on every write" | spec delta: omit an undetectable user, never `user: ""` | `grep -n "No user detectable" openspec/changes/fix-frontmatter-remainder/specs/vault-note-identity/spec.md` | — |
| 6e | `openspec/specs/update-reparent-luhmann-batch/spec.md:157` ("a note converted from CRLF frontmatter hashes the same as its LF-authored equivalent") | no change: still true, and now holds before conversion too | — | — |
| 6f | `docs/architecture/adr.md` D3 (lines ~1166-1173, CRLF separator layout) | no change: describes the archived ruling V2, which still holds | — | — |
| 6g | `openspec/specs/vault-note-identity/spec.md` "Vault field resolved from explicit configuration" | spec delta: amend fills an empty `vault:`; never `vault: ""` | `grep -n "A bookkeeping amend on a note without vault" openspec/changes/fix-frontmatter-remainder/specs/vault-note-identity/spec.md` | — |
| 6h | `openspec/specs/vault-note-identity/spec.md` | spec delta (ADDED): note readers read a CRLF note as its LF form | `grep -n "Note readers SHALL read a CRLF note as its LF form" openspec/changes/fix-frontmatter-remainder/specs/vault-note-identity/spec.md` | — |
| 6i | `openspec/specs/vault-parent-offers/spec.md` receipt requirement | spec delta: no-identity offers wait in `attention` | `grep -n "No user identity holds the offer" openspec/changes/fix-frontmatter-remainder/specs/vault-parent-offers/spec.md` | — |
| 2b | `docs/GLOSSARY.md` `### outbox`, `docs/architecture/adr.md` D6 | append: the no-user `attention` case, the kept-receipt pass and the other-parent discard | `grep -n 'user.email' docs/GLOSSARY.md docs/architecture/adr.md` | — |
| 7 | `openspec/specs/vault-parent-offers/spec.md` "The outbox SHALL drain …" | no change: queued/rejected rules are unchanged; `attention` is specified under the receipt requirement | — | — |
| 8 | `openspec/specs/update-reparent-luhmann-batch`, `skill-runbook-registration` CRLF requirements | no change: rename, adopt and refresh already convert | — | — |
| 9 | `docs/architecture/adr.md` lines 702, 744, 814 ("out of scope") | no change: unrelated (shadowed copies, chunk-index cleanup, update version skew) | — | — |
| 10 | `docs/architecture/c1-system-context.md` "Out of scope at L1" | no change: about route's L1 placement | — | — |

No main spec or doc states the show-header, supersedes sub-key or empty-`user:` gaps, or the amend-CRLF, backfill-unknown-keys, backfill-CRLF or receipt re-send gaps as a Non-Goal; they were stated only in the archived change's design and in ROADMAP row 9 (row 1).
