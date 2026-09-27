## Context

Engram reaches a parent vault in two exclusive ways today (code at `e1930b56`):

- **`ENGRAM_SERVER`**, the thin client. `serverBase` (`internal/cli/serve_client.go:397`) is checked first in ten dispatch branches in `internal/cli/targets.go`: amend, query, query-chunks, the three learn types, register-skills, activate, show and show-chunk. When it is set, the CLI sends those commands to the server over HTTP and never touches local files.
  - A served `learn` or `amend` becomes a pending offer on the server, which replies `{"status":"offer received","luhmann":"<id>"}` (`offerReceipt`, `serve.go:162`).
  - Nothing is written locally. If the server is unreachable the write fails.
- **`ENGRAM_PARENT`**, read-only merge. `runMergedQuery` (`merged_query.go:177`) fetches the parent's `/query` and then runs the local query.
  - `mergeQueryPayloads` (`merged_query.go:100`) tags every item with `from_parent`, sorts the non-trigger items from both sources by raw score, and caps them.
  - It does **no dedupe**. It keeps only local clusters and discards the budget block.
  - `show`/`show-chunk` fall back to the parent on a local miss.
  - `learn`, `amend` and `activate` never contact the parent.

Joe's 2026-09-27 decision (vault note 784a, updating 784's exchange model) replaces both modes with one model: local-first, with two-way offers. The research for this change found these gaps in the current code, and the design has to deal with each of them:

- **G1.** `noteHasPendingMarker` (`offer.go:89-114`) treats a runbook as pending only if it also carries `skill_hash`. A runbook offered to the parent with `pending: true` would therefore go live immediately and never be curated (`offer_test.go:136-147` locks this behavior in).
- **G2.** A served `/amend` sets `pending: true` on the parent's live note in place (`serve.go:315`). That hides the note from every query until it is curated.
- **G3.** The typed frontmatter rewriters drop keys their structs don't declare. `resituate` also hand-copies a partial field list (`resituate.go:223,256`), and already drops `pending`, `sources`, `tags`, `supersedes` and `vocab_version`. #780 tracks adopt dropping unknown keys.
- **G4.** `GET /show` changes the note it returns: it caps `red_flags` and appends a links section. It cannot be used to fetch an exact copy.
- **G5.** `activate` only knows local sidecars. For a parent item it logs a skip and still succeeds unless every note failed.
- **G6.** Only `learn` and `learn qa` create the vault. `amend`, `activate` and `resituate` fail on the lock's `O_CREAT` when the vault directory is missing.
- **G7.** In the merge, explore picks outscore direct matches and push them out (#744). The merged budget block is all zeros (#743).
- **G8.** The served `learn` still honors the client's `target`/`position`, which are Luhmann IDs from the *client's* vault.

Two things already exist and are reused:

- `embed.ContentHash` (`internal/embed/hash.go:49`): `sha256(situation + NUL + BodyText)`. It ignores other frontmatter.
- The `json:"-"` tagging of `SkillHash`/`SkillKey`/`SkillSource` on `LearnArgs` (`learn.go:86-98`, ruling R31).

## Goals / Non-Goals

**Goals:**
- One mode. Every environment has a local vault, and when `ENGRAM_PARENT` is set it exchanges notes with the parent in both directions through offers.
- Local writes always succeed. Offers that fail are kept in an outbox and retried, so nothing is lost offline.
- Merged recall shows one copy of each note, the local one.
- A note is never offered back to the vault it came from, and exchange cannot loop.
- Remote-set fields can't forge identity, skill registration, or origin links.

**Non-Goals:**
- More than one parent, or a mesh. The single-parent tree from note 784 stands.
- Moving transcript chunks between vaults (Joe's decision 7). Parent chunks can still *appear* read-only in merged query results and through `show-chunk --parent`, as they do today. See the open questions.
- Near-duplicate detection in the merge. Near-duplicates are left to curation.
- Authentication on `engram serve`. Trust is still network reachability (`vault-serve-api`).
- Pushing note changes from the parent. The child pulls a note only when it activates it.

## Decisions

### D1: `ENGRAM_SERVER` is a hard error for every command

The check is one pre-dispatch guard in the targ entry. When `ENGRAM_SERVER` is non-empty, every subcommand, including `serve`, exits non-zero before touching any vault. The error names the fix:

`ENGRAM_SERVER is no longer supported; set ENGRAM_PARENT=<same URL> instead — every environment now keeps its own local vault and offers notes to the parent`.

The following are deleted:
- all ten `serverBase` branches, plus `serverBase` itself and `ExportServerBase`;
- `errDiscardOverServer` and `errRegisterSkillsOverServer`;
- `fetchQuery`, `fetchQueryChunks`, `fetchActivate` and `fetchAmend` in their `ENGRAM_SERVER` form.

`fetchLearn`, `printOfferReceipt` and `fetchRaw` become the offer client in D6. Subprocess tests must strip `ENGRAM_SERVER` from their environment, as they already strip `ENGRAM_PARENT`.

*Rejected alternatives:*
- **Treat `ENGRAM_SERVER` as an alias for `ENGRAM_PARENT`.** Joe ruled out a compat shim, and a silent alias would hide the fact that a local vault now exists.
- **Exempt `serve`.** The literal decision is "if it's set, engram exits". An exempt server would leave the variable half-supported.

### D2: The vault is created on first use by any command that resolves it (folds in #766)

A shared `ensureVault` step (today's `ensureVaultDir`/`initializeVault`) runs in dispatch after the vault path is resolved and before the command runs. It applies to every command that takes a vault: query, show, amend, activate, resituate, update, register-skills, serve, count, check, and the rest. When it actually creates the vault, it prints one stderr line: `engram: created new vault at <path>`. That line is the answer to #766's typo concern: a mistyped `--vault` or `ENGRAM_VAULT_PATH` now produces an empty vault, but visibly. The vault starter `.gitignore` gains `.engram-outbox.json` (D6). For existing vaults, `ensureVault` appends that line if it is missing.

*Rejected alternatives:*
- **Create on `update` only** (#766 option 1). This leaves `amend`, `activate` and `resituate` failing on the lock (G6), and Joe chose "any command".
- **Warn without creating** (#766 option 3). This contradicts decision 2.

### D3: The origin link is one nested frontmatter key, `parent`, qualified by the parent's URL

```yaml
parent:
  url: http://host:8093          # normalized ENGRAM_PARENT (lower-case scheme/host, no trailing slash)
  note: 1100.2026-09-27.some-slug  # the counterpart's basename in that parent
  via: offered                     # offered (local → parent) | pulled (parent → local)
  hash: sha256:…                   # embed.ContentHash of the version last exchanged
aliases:                           # any note, either vault: basenames folded into this note by curation
  - 1101.2026-09-27.other-slug
```

**How the link is set.**
- *Offer receipt (up):* after the parent answers an offer, the local note gets `parent.{url,note,via: offered,hash}`, where `hash` is the content hash that was sent. The write is a frontmatter-only rewrite under the vault lock. It does not re-embed and does not re-stamp identity.
- *Pull-down (D8):* the pending local note is created with `parent.{url, note: <the parent basename it was fetched as>, via: pulled, hash: <hash of the fetched note>}`.
- *Curation fold (D10):* `engram amend --discard --into <existing>` appends the discarded note's basename to the existing note's `aliases`. If the discarded note carried a `parent` link and the existing note has none, the link moves to the existing note.

**Qualified by URL** (vault note 1073a). A link only counts when `parent.url` equals the currently configured, normalized `ENGRAM_PARENT`. If `ENGRAM_PARENT` is repointed, old links stop matching, so the error is a missed dedupe (a visible duplicate) and never a wrong one.

**Survival** (the pattern of `vault-note-identity`'s skill-field survival requirement). The `parent` and `aliases` fields are added to the fact, feedback and runbook frontmatter structs. They must be preserved unchanged by every rewrite path that is not the exchange itself:
- `amend` (all flags, including `--clear-pending` and `--activate`);
- `resituate` (explicitly hand-copied, G3);
- identity backfill;
- Luhmann reparent and rename;
- wikilink rewrite;
- vocab tag rewrites and legacy vocab cleanup;
- `scrubDeletedNoteReferences`;
- register-skills refresh and adopt.

Only the offer receipt, pull-down, and `amend --discard --into` may change them. Each site gets a test, modeled on `skillfields_survival_test.go`.

**Wire protection** (R31). `LearnArgs`/`AmendArgs` carry these as `json:"-"`. A remote request can never set `parent` or `aliases` on the parent's vault.

*Rejected alternatives:*
- **Flat top-level fields** (`parent_note`, `parent_via`, …). That means four keys for every survival site to hand-copy. One nested key has one failure mode.
- **An unqualified basename.** It collides when `ENGRAM_PARENT` is repointed (1073a).
- **Recording the child's basename on the parent note.** The parent doesn't know the child's vault. It would also be a remote-set identity field.
- **A separate link index file.** It would drift from the notes and not survive a copy or rename of the vault. Frontmatter travels with the note.

### D4: Receipt and dedupe keys. The counterpart is recorded before curation and resolved afterwards through `aliases`

The served `/learn` response gains fields: `{status, luhmann, basename, pending: true}`. The additions are backward-compatible for readers. The child records `parent.note = basename` at receipt time, even though the offer is still pending on the parent. This is sound because a pending note is excluded from the parent's query results (`vault-offer-curation`), so there is nothing to dedupe against until curation acts:

- **Absent** → the offer becomes live under the same basename, and the link matches directly.
- **Covered or near** → the curator runs `amend --discard --into <existing>` (D10), which adds the offer's basename to `<existing>.aliases`, and the link matches through the alias.
- **Discarded outright** (a rare bare `--discard`) → the link dangles harmlessly. The next content amend re-offers it as a new note (D6).

Merged query asks the parent for dedupe keys with a new `/query` parameter, `dedupe-keys=1`. The served payload then adds `content_hash` and `aliases` to each note item. The parameter is opt-in so that local payloads stay byte-identical (`vault-offer-curation`'s byte-identity scenario). The merged output strips both fields.

**Dedupe rule.** A parent note item P is the same note as a live (non-pending) local note L when either:
1. `L.parent.url` is the current parent and `L.parent.note` equals P's basename or is one of `P.aliases`; or
2. `L`'s content hash (from its sidecar) equals `P.content_hash`, and both are non-empty.

Pending local notes never match, so a note that was pulled down but not yet curated cannot hide its parent copy. The local note is kept:
- If L is already in the local results, P is dropped.
- If it isn't, L's item replaces P at P's rank, with `from_parent: false`.

This applies to trigger hits, Channel 1 and the recency channel, and runs **before** ranking and capping. Parent chunk items are never deduped, because chunks don't travel.

*Rejected alternatives:*
- **Hold the counterpart until curation reports back.** `vault-offer-curation` says the outcome is never reported to the caller, and adding a callback channel would contradict it.
- **Match on basename only.** It misses covered and near folds, which are the common case.
- **Always include hashes in the payload.** That breaks byte-identity for every local query.

### D5: What is offered, and what stays local

| Write | Offered? |
| --- | --- |
| `learn fact`/`feedback`/`runbook` (not registration) | yes, as a new note |
| `amend` with any content flag (situation, subject/predicate/object, behavior/impact/action, done-when, body, red-flag, trigger) | yes, as an amend-offer targeting `parent.note` when a link exists, else as a new note |
| `resituate` (changes `situation`, which is part of the content hash) | yes, like a content amend (**assumption**) |
| `amend --activate`, `--clear-pending`, `--discard`, `--discard --into` | no, bookkeeping |
| `amend` with only `--supersedes`/`--chunk-source` | no. It is relational or local provenance and does not re-embed (**assumption**) |
| identity backfill (`update --backfill-identity`) | no, bookkeeping (**Joe's stated assumption**) |
| `learn qa` notes | no. Their `contributors` wikilinks name local basenames (**assumption**) |
| any note carrying `skill_hash` (registration) | no. The parent runs its own registration |
| a pending note, or any write whose new content hash equals `parent.hash` | no. This is loop suppression (D8) |

**Payload translation.** An offer never sends `target`/`position` (G8). The server also ignores them on served learns and places offers at top level. `supersedes` entries are translated to the parent counterpart when the superseded local note has a `parent` link, and are dropped otherwise. `chunkSources` are never sent (decision 7).

*Rejected alternative:* **offer every amend.** Bookkeeping amends would flood the parent's curation queue with no-op offers.

### D6: The outbox is a coalesced per-note dirty set in `<vault>/.engram-outbox.json`, and payloads are built at send time

```json
{"version": 1,
 "entries": [{"note": "<local basename>", "queued": "2026-09-27T10:00:00Z",
              "attempts": 2, "last_error": "dial tcp …: connection refused",
              "state": "pending|rejected", "rejected_hash": "sha256:…"}]}
```

- **Enqueue.** Every offerable write (D5) adds or updates an entry, keyed by local basename, under the vault lock and in the same critical section as the note write. The local write never fails because of the parent.
- **Payload built at send time.** An entry holds no payload snapshot. When it is sent, the current note is read: a learn-offer is built if there is no current-URL `parent` link, and an amend-offer (`offer.for = parent.note`) if there is. As a result, a learn followed by several amends made offline goes up as **one** offer. If the note has been deleted or has become pending, the entry is dropped.
- **Order.** FIFO by `queued`.
- **When it drains.** The outbox drains after any successful parent contact in the same command: `learn`, content `amend`, `resituate`, `query` (after the parent `/query` succeeds), `activate` (after a pull-down succeeds), and `update`. For learn and amend the order is: local write, then enqueue, then drain. So when the parent is reachable, the note's own offer goes out immediately. The drain sends outside the lock, then re-takes the lock to apply receipts. If the note's content hash changed between send and apply, the entry stays queued.
- **Idempotency.** Each offer carries `offer.key = sha256(user + local basename + content hash)`. The server looks for a pending note with the same key and, if it finds one, returns that note's receipt instead of writing a second one. This covers a response lost after the server accepted the offer.
- **Failures.**
  - A transport error, timeout or 5xx stops the drain. The entries stay, `attempts`/`last_error` are updated, and one stderr warning is printed: `engram: parent unreachable; N offer(s) queued (oldest <age>)`.
  - A 4xx marks that entry `rejected`, records `rejected_hash`, and the drain continues with the next entry. A rejected entry is re-armed only when the note's content hash changes.
  - `engram update` reports the outbox as a notify-only notice: queued count, oldest age, and the rejected entries with their errors.
- **The file.** It is in the vault root, not scanned as a note (`ListMD` reads `.md` only), and gitignored (D2). It is written by atomic temp-rename (ADR-0013).

*Rejected alternatives:*
- **A directory with one file per offer, each holding a payload snapshot.** It would send stale content and duplicate offers per edit, and it needs ordering metadata anyway.
- **JSONL append.** Coalescing needs rewrites, and append-then-compact is two mechanisms where one will do.
- **A file under `$XDG_DATA_HOME/engram/state`.** It is not per-vault, so it breaks for `--vault` overrides.
- **Sending synchronously and failing the write when the parent is down.** That contradicts decision 4.

### D7: Offers use the existing `POST /learn` route, and `/amend` is removed from the served set

An amend-offer is a served learn with `offer.for = <parent basename>`. The server resolves `offer.for` against live note basenames and then against `aliases`. If it doesn't resolve, the server drops it and the offer is judged as a new note. The resulting pending note records `offer: {for, key}` in frontmatter so curation knows the target.

`offer.for` and `offer.key` are the only new remote-settable fields. They land only on a pending note, and curation reviews them. They can't make anything live or change any identity or link field.

`POST /amend` is deleted: its only caller was the `ENGRAM_SERVER` client, and its in-place pending marker hides live notes (G2). This follows the rule that deprecated machinery is not kept around. `/activate` stays, because it is used by D8. `/query-chunks` loses its in-tree caller but is left alone (see the open questions).

The server also:
- forces top-level placement (G8);
- fixes G1: **any** note type with `pending: true` counts as pending. (The real vault was checked read-only on 2026-09-27: it has zero notes with `pending: true`, so the fix changes no live note.)

*Rejected alternatives:*
- **Keep `/amend` and fix its semantics.** That is two offer routes for one concept.
- **A new `POST /offer` route.** It duplicates `/learn`'s code path, locks and identity handling, and `vault-serve-api` requires every served command to reuse the CLI path.

### D8: Pull-down. Activating a parent-sourced note writes a local pending copy, fetched raw

`engram activate --note <ref>` resolves each ref locally first. On a local miss with `ENGRAM_PARENT` set, or when `--parent` is given, it treats the ref as a parent note:

1. **Fetch** the exact bytes with `GET /show?note=<ref>&raw=1`. This is a new `raw` mode that returns the note file verbatim, with no red-flag cap and no links section (G4).
2. **Skip if unchanged.** If a local note (live or pending) already has a current-URL `parent.note == <basename>` and `parent.hash` equals the fetched note's content hash, write nothing. If that local note is live, bump its sidecar `LastUsed` instead.
3. **Otherwise write** a new local note under the vault lock:
   - a fresh local top-level Luhmann ID, and a basename built from that ID, the parent's `created` date and the parent's slug;
   - the body **verbatim**, so the content hash matches the parent's for D4;
   - `pending: true`;
   - `parent: {url, note, via: pulled, hash}`.

   It drops the parent's own `parent`, `aliases`, `offer`, `skill_hash`, `skill_key`, `skill_source`, `sources` (chunk IDs), `supersedes` and `tags`, and lets local vocab reassign tags. It keeps the parent's `repo`/`user`/`vault` as authorship provenance, and the rest of the typed fields. It embeds on write.
4. **Best-effort remote bump.** It also POSTs `/activate` to the parent, so the parent's recency signal still reflects use from its children. This is not queued, since recency is lossy by nature.

A changed parent note just arrives as another pending offer (Joe's decision 6). Local curation judges it against the local copy: covered means discard-into, and near means fold then discard-into.

**Loop rule.** A pulled note is never offered back up:
- it is pending (D5);
- `--clear-pending` is bookkeeping;
- any later amend whose content hash equals `parent.hash` is suppressed.

A genuine local edit to a pulled note *is* offered, as an amend-offer targeting its origin. That is new information, not a loop.

`activate` now reports each ref it skips on stderr, and exits non-zero if any ref failed, not only when all of them did (the #746 addendum).

*Rejected alternatives:*
- **Prefix parent paths in the payload** (`parent:<basename>`). It changes the payload format for every consumer (probes, the shim follow-frame's bare `engram show <basename>`). Local-first resolution with a parent fallback mirrors `show` and works with the paths the agent already has.
- **Write the pulled note live.** That bypasses curation, contrary to decision 6.
- **Reuse `/show`'s rendered output.** It is altered (G4), so the hash would never match.

### D9: Merged-query fixes needed now that merge is the primary path

- **#744 (in).** Explore picks crowd out direct items. The merge now keeps the local path's positional order: trigger hits first, then direct items from both sources merged by score, then explore picks from both sources merged by score, and caps by position. The note floor (`noteFloorK`) is re-applied over the merged direct set. Without this fix, every child recall degrades to explore-only.
- **#743 (in).** The budget block is all zeros. The merged payload reports the merge-applied `limit`, `content_budget`, `lazy_chunks`, `chunks_snippeted` (taken from `capChunkContent`'s return, which is discarded today) and `explore_allocated` (the term-to-count map, summed per term over both sources).
- **#745 (out: resolved by design, not by carrying parent clusters).** Every environment now has a local vault and a local chunk index (D2), so recall Step 2.5 always has local clusters to judge. Parent clusters are built from parent chunks, and decision 7 says chunks never travel, so crystallizing local notes from parent clusters would move chunk content in all but name. Parent items stay ranked-items-only (`Merged results are not re-clustered`). Close #745 with this reasoning.
- **#746 (superseded).** `learn` and `amend` need no `--parent`, because offering is automatic (D5/D6). `activate` gains `--parent` and a local-miss fallback that pulls down (D8), which covers #746's use case of bumping a parent note, and its partial-failure addendum is fixed in D8. Close #746 as superseded by this change.

### D10: Curation gains `--discard --into`, and the skills change through writing-skills TDD

`engram amend --target <offer> --discard --into <existing>` is a bookkeeping amend. It discards the offer, appends the offer's basename to `<existing>.aliases`, and moves the offer's `parent` link when `<existing>` has none (D3). A bare `--discard` is unchanged.

Skill edits, each RED → GREEN → REFACTOR under `superpowers:writing-skills` (repo CLAUDE.md):
- **curate:**
  - pending offers now also arrive *from the parent* (`parent.via: pulled`), and are judged the same way;
  - an offer with `offer.for` is judged against that note first;
  - covered and near dispositions end with `--discard --into <existing>`;
  - the description and red flags are widened from "served write" to "offer";
  - it is still host-local and never run synchronously inside `serve`.
- **recall:**
  - Step 2.7 activates parent items (`from_parent: true`) like local ones, which pulls them down;
  - a new red flag: never `amend` a `from_parent` item (it doesn't resolve locally);
  - after a pull-down, `pending_offers` in the next payload is expected, and curation follows per the shim's standing-instruction rule.
- **learn:**
  - a short note that every learn is offered to the parent automatically when `ENGRAM_PARENT` is set, and that an outbox warning is not a failure;
  - never set `ENGRAM_SERVER`.

The registered runbook mirrors of these skills (`skill-claude-curate` etc.) refresh through the existing registration flow (`skill-runbook-registration`). No manual vault edit is made.

### D11: Tests. DI fakes for the parent, and a real-binary run against a local serve

- **Unit tests** use the existing `Deps.Fetch` seam with fakes for success, transport error, 5xx, 4xx, a lost response followed by a retry, and a malformed receipt. They also use the vault FS/lock fakes (imptest, rapid, gomega; vault note on the engram test stack):
  - rapid properties for outbox coalescing and FIFO ordering;
  - rapid properties for dedupe (a deduped merge never contains both a local note and its linked parent item; pending locals never suppress);
  - a property that the link survives across every rewrite path.
- **Real-binary verification.**
  - Build the binary into a scratch directory.
  - Run `engram serve` on `127.0.0.1:<port>` against a scratch parent vault, and a child with scratch `XDG_DATA_HOME` plus an explicit `--vault`, with `ENGRAM_PARENT` pointing at the scratch server.
  - Check each behavior: offline learn then outbox then drain; receipt linking; absent/covered curation with dedupe in the merged query; pull-down on activate; the no-loop check; the `ENGRAM_SERVER` hard error; the first-use creation notice.
  - Run from a non-repo cwd (vault notes 1066 and "verify CLI with real binary + real args").
  - Never touch the real vault or `~/.local/share/engram`.

## Risks / Trade-offs

- **[A near-duplicate that isn't linked shows twice in merged recall]** → This is accepted (decision 5). The pull-down and curation paths create links, so a duplicate is healed once it is used.
- **[Outbox growth when the parent is down for a long time]** → Coalescing bounds it to one entry per changed note. `update` surfaces the count and age.
- **[A lost response after the parent accepted the offer]** → The `offer.key` idempotency check returns the existing receipt. If curation already discarded the first offer, the retry creates a second offer, and curation judges it covered.
- **[A remote client spams `offer.for` at arbitrary notes]** → It only creates pending notes, which curation judges. No live note changes without host curation (unchanged trust model).
- **[Version skew: an old parent ignores `dedupe-keys` and `raw`, and has no `basename` in its receipt]** → Dedupe falls back to basename links only. Pull-down on an old parent fails with a clear "parent too old" error, and the offer receipt without `basename` records no link (a warning is printed). The parent is expected to upgrade first (migration step 1).
- **[Hosts with `ENGRAM_SERVER` break on upgrade]** → This is the intended hard error, and it names the fix.
- **[Adding `parent`/`aliases` to structs whose sibling paths already drop keys (#780, resituate)]** → The survival tests cover every site listed in D3, and resituate's hand-copy list is fixed for the new fields. #780's general unknown-key loss stays tracked there.
- **[Draining on query adds latency]** → The drain runs only after the parent `/query` has already succeeded, it is bounded by the existing 30s client timeout per request, and usually there is nothing to send.

## Migration Plan

1. Upgrade the parent host first. It gains the extended receipt, `dedupe-keys`, `raw` show, runbook pending detection, `offer` handling, and the `/amend` removal.
2. Upgrade the children. Any host with `ENGRAM_SERVER` set gets the hard error and switches to `ENGRAM_PARENT`. Its first command creates the local vault.
3. Rollback: revert the commit. Notes written with `parent`/`aliases` stay valid YAML, and older binaries' typed rewriters drop those keys, which is harmless. The outbox file is ignored by older binaries.

## Open Questions

1. **Parent chunks in merged results.** Decision 7 is read as "never copied or offered". Parent chunk items still appear read-only in merged `query` and through `show-chunk --parent`, as today. Should merged query drop parent chunks entirely?
2. **Best-effort `/activate` on the parent** during pull-down (D8 step 4). Keep it for the parent's recency signal, or keep activation purely local?
3. **`resituate`, `learn qa`, and `--supersedes`/`--chunk-source`-only amends.** They are classified by assumption in D5: resituate is offered, and the other two stay local. Please confirm.
4. **`/query-chunks`.** It has no in-tree caller after this change. Delete it now (no hoarding), or keep it as a curl-able read route?
5. **The `ENGRAM_SERVER` guard on `engram serve` itself** (D1). Should it be exempt?
