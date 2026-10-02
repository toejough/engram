# Doc-surface enumeration: fix-frontmatter-remainder

This change alters four documented behaviours:

- amend converts the CRLF notes it writes to LF instead of refusing them;
- identity backfill keeps keys it does not set, refuses an anchored key it would set, and stamps CRLF notes (as LF) instead of skipping them;
- a refused offer receipt puts its outbox entry in the `attention` state;
- `engram update`'s outbox notice reports `attention` entries.

**Search** (2026-10-02, worktree `runbook-vs-skill`, branch `fix-789-frontmatter-remainder`):

```
grep -rn -i -E "crlf|backfill-identity|identity backfill|outbox|receipt|rejected|non-goal|out of scope" \
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
| 7 | `openspec/specs/vault-parent-offers/spec.md` "The outbox SHALL drain …" | no change: queued/rejected rules are unchanged; `attention` is specified under the receipt requirement | — | — |
| 8 | `openspec/specs/update-reparent-luhmann-batch`, `skill-runbook-registration` CRLF requirements | no change: rename, adopt and refresh already convert | — | — |
| 9 | `docs/architecture/adr.md` lines 702, 744, 814 ("out of scope") | no change: unrelated (shadowed copies, chunk-index cleanup, update version skew) | — | — |
| 10 | `docs/architecture/c1-system-context.md` "Out of scope at L1" | no change: about route's L1 placement | — | — |

No main spec or doc states the amend-CRLF, backfill-unknown-keys, backfill-CRLF or receipt re-send gaps as a Non-Goal; they were stated only in the archived change's design and in ROADMAP row 9 (row 1).
