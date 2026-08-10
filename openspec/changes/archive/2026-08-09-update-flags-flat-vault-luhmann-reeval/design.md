## Context

`engram update` already has a "detect vault-shape drift, notify, do not silently fix" pattern
(old-format vocab files, untagged vocab definitions, duplicate chunk-index backlog) AND a
separate "opt-in one-shot migration flag" pattern: `engram update --regen-vocab` migrates
old-format vocab files only when the user explicitly asks (never silently during a routine
refresh — a deliberate mid-cycle design reversal recorded in ROADMAP.md's duplicate-backlog
entry: "a destructive migration should run only when the user asks"). This change's notice
half follows the first pattern; its remedy half — `--reparent-luhmann` — follows the second.

`vocab refit` is the closest existing precedent for a derive-judge-apply flow: the binary
derives candidate clusters from embeddings, and when a cluster is genuinely new (not resolvable
mechanically), it emits a `naming_requests` + `fingerprint` JSON payload and exits without
writing; the calling agent names each cluster and re-runs with an answers file; a stale-vault
change between derive and apply voids the answers. `--reparent-luhmann` reuses this shape,
because — like cluster naming — "is note B a sub-point of note A, the same thought continued,
or unrelated" is a content judgment the binary cannot make from embeddings alone (per #701's
design, which explicitly rejected embedding-similarity search as a stand-in for the disposition
test).

**Filename coupling (the hard part).** Vault notes are files named `<id>.<date>.<slug>.md`
(confirmed: `/Users/joe/.claude/jobs/.../1.2026-07-12.ratelimit-token-bucket.md`), and the
wikilink graph's canonical node key is that same extension-less basename
(`vaultgraph.ParseBasename`). Renumbering a note's Luhmann ID therefore means renaming its file
— which breaks every `[[old-basename]]` wikilink and `supersedes:`/`Supersedes: [[...]]`
reference elsewhere in the vault unless those are rewritten too. `internal/vaultgraph` today is
read-only (`ScanVault`, `ParseWikilinks`, `ParseBasename`, graph/BFS queries) — there is no
existing rename-and-rewrite-backlinks helper; this change has to build one.

## Goals / Non-Goals

**Goals:**
- Notice (as originally scoped): detect an all-top-level vault, print a one-line notice naming
  `engram update --reparent-luhmann` as the remedy.
- Derive: given the vault's existing embedding sidecars, propose candidate parent/sibling
  relationships among top-level notes for agent review.
- Apply: given agent-authored disposition answers, compute new IDs, rename files + sidecars,
  and rewrite every incoming reference vault-wide, atomically and idempotently re-runnable.
- Dry-run: preview the full rename/rewrite map with zero writes.

**Non-Goals:**
- Fully automating the disposition judgment — the binary proposes candidates from similarity;
  it does not decide continuation-vs-sibling-vs-unrelated on its own (same boundary #701 drew
  for per-capture disposition).
- Running `--reparent-luhmann` automatically as part of a routine `engram update` — it is
  opt-in only, named by the notice but never auto-invoked (matches the `--regen-vocab`
  precedent).
- Re-deriving embeddings or changing the embedder — sidecars are read as-is.
- Touching the chunk index directly — see Risks below for how the existing chunk-dedup/prune
  machinery is expected to absorb renamed sources on the next sweep, without new code in this
  change.

## Decisions

**Decision 1 — Candidate derivation uses existing `.vec.json` sidecars, not a full LLM pass
over every note pair.** For each top-level note, compute cosine similarity against all other
top-level notes' embeddings; keep the top-K (e.g. K=3) above a similarity floor as candidates.
This bounds the derive payload's size on a large vault (447 notes) instead of an O(n²) full
matrix with no floor. Alternative considered: ask an agent to read the whole vault and propose
structure freeform — rejected as unbounded cost and non-reproducible across runs; a fixed
candidate-generation step keeps `--dry-run` output stable and diffable.

**Decision 2 — Answer format mirrors `vocab refit --names`:** a JSON file
`{"reparenting": [{"note": "<id>", "position": "continuation|sibling|top", "target": "<id>"}],
"fingerprint": "<echoed>"}`. `position: top` (explicit no-op) is a valid answer — the agent may
determine a candidate pair is unrelated despite embedding similarity, and the note stays where
it is. Every top-level note that had ≥1 candidate MUST get exactly one answer entry; notes with
no candidates above the similarity floor are never included in the payload and stay top-level
implicitly (no answer needed) — this bounds the agent's judgment work to only the notes with a
plausible relationship, not the full 447.

**Decision 3 — Apply order is oldest-ID-first, single deterministic pass.** Process answered
notes in ascending original-ID order, computing each new ID via the existing
`nextChild`/`nextSibling`/`nextTopLevel` against the vault's *current* ID set (which updates as
earlier notes in the same run are renumbered) — this matches how `learn` already allocates IDs
incrementally and avoids collisions within one apply run. Notes answered `position: top` are
skipped (no rename).

**Decision 4 — Rename + backlink rewrite is one atomic-per-note operation, built as a new
`internal/vaultgraph` (or sibling `internal/cli`) helper.** For each note being renamed: (a)
rename the note file and its `.vec.json` sidecar; (b) update the note's own frontmatter `id:`
field (if present) to the new ID; (c) scan the vault (reusing `vaultgraph.ScanVault` +
`ParseWikilinks`) for every note referencing the old basename — in a `[[old-basename]]`
wikilink or a `Supersedes: [[old-basename]]` body line or `supersedes:` frontmatter `note:`
field — and rewrite the reference to the new basename. This is new code; no existing helper
does rewriting (only parsing/resolution). Because renames cascade (a note renamed in an earlier
step might itself be referenced by a note renamed in a later step), reference rewriting must
run as its own vault-wide pass after all renames in the run are known, not interleaved
per-note.

**Decision 5 — `--dry-run` prints old→new basename pairs and, for each, the list of referencing
notes that would be rewritten**, so the user can review the full blast radius before committing
— same transparency bar as `vocab refit --dry-run`'s derivation diff.

## Risks / Trade-offs

- **[Risk] Vault-wide mutation itself has precedent (`vocab refit`/`--regen-vocab` already
  rewrite tags/content across many notes in one run and ship safely) — but renaming a note's
  file (changing its on-disk identity) and chasing every other note's reference to that
  basename does not.** That combination is new: a bug here doesn't just leave a note's content
  wrong (recoverable by re-editing), it can silently orphan a wikilink/supersedes reference or
  collide two notes onto one filename. → Mitigation: `--dry-run` default-first workflow
  (require an explicit confirming flag to actually write — do not default to writing); full
  test coverage of the rename+rewrite helper with `internal/vaultgraph`-style fixtures,
  including the cascading-rename case, before wiring into `update`; consider writing a pre-apply
  vault backup/snapshot note in the report (open question below).
- **[Risk] Chunk index entries for renamed notes reference the old source path/anchor.** →
  Mitigation: no new code in this change touches the chunk index; the existing
  `ingest-chunk-dedup`/`ingest-prune-detach` machinery (content-hash dedup, prune of orphaned
  lower-precedence copies) is expected to absorb the renamed file as new content on the next
  `engram ingest --auto` sweep, with the old path's chunks aging out via the existing prune
  path — verify this assumption holds during implementation (task) rather than asserting it;
  if it doesn't cleanly self-heal, that's a follow-up issue, not silently absorbed scope here.
- **[Risk] Candidate derivation (Decision 1) is a similarity heuristic, not a guarantee of
  finding every real branching relationship** — some genuine continuations may fall below the
  similarity floor and never surface as a candidate. → Mitigation: acceptable — this is a
  best-effort one-shot improvement, not a completeness guarantee; a second `--reparent-luhmann`
  run after new notes are added can find more later. Document this bound in the notice/command
  help text so it isn't read as "fully re-evaluates."
- **[Risk] Answer-derive-apply is three round-trips (derive → agent judgment → answer file →
  apply), more ceremony than a single command.** → Mitigation: matches the accepted
  `vocab refit` UX already in production; consistency with an existing pattern the user already
  knows outweighs saving one round-trip here.

## Migration Plan

Opt-in only, never run as part of routine `engram update`. Steps for a user invoking it:
1. `engram update --reparent-luhmann` (no `--dry-run` — derive never writes regardless) →
   binary derives candidates, emits payload.
2. Agent applies disposition judgment, writes answers file.
3. `engram update --reparent-luhmann --answers <file> --dry-run` → preview the resulting
   rename/rewrite map.
4. `engram update --reparent-luhmann --answers <file>` → apply.
5. `engram ingest --auto` (normal next sweep) → chunk index catches up to renamed sources.

No automatic rollback mechanism is in scope for this change (git is the vault's own safety net,
per the standing "git is the fallback" convention — commit before running).

## Open Questions

- ~~Should `--dry-run` work in derive-only mode, or only once answers exist?~~ Resolved:
  `--dry-run` requires `--answers` and shows the intended renames (old→new basename map +
  referencing notes that would be rewritten). Derive-phase output (the candidate payload) is
  always a preview by construction — it never writes regardless of `--dry-run` — so a separate
  candidate-only dry-run mode adds nothing; `--dry-run` without `--answers` is simply rejected
  as a usage error (nothing to preview).
- Does the chunk index genuinely self-heal on rename via the existing dedup/prune sweep, or
  does this need an explicit "re-ingest the renamed note" step wired into `--reparent-luhmann`
  itself? Verify empirically before implementation is considered done.
- ~~Should the binary refuse to run `--reparent-luhmann` against a vault with uncommitted git
  changes?~~ Resolved: no guard rail — Joe isn't worried about this; `--dry-run` +
  git-as-fallback (per the standing "git is the fallback" convention) is sufficient.
