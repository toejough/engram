## Context

Engram reaches a parent vault in two exclusive ways today (code at `e1930b56`):

- **`ENGRAM_SERVER`**, the thin client. `serverBase` (`internal/cli/serve_client.go:400`) is checked first in ten dispatch branches in `internal/cli/targets.go`: amend, query, query-chunks, the three learn types, register-skills, activate, show and show-chunk. When it is set, the CLI sends those commands to the server over HTTP and touches no local files.
  - A served `learn` or `amend` becomes a pending offer on the server, which replies `{"status":"offer received","luhmann":"<id>"}` (`offerReceipt`, `serve.go:162`).
  - Nothing is written locally, and if the server is down the write fails.
- **`ENGRAM_PARENT`**, read-only merge. `runMergedQuery` (`merged_query.go:177`) fetches the parent's `/query`, runs the local query, and `mergeQueryPayloads` (`:100`) then:
  - tags every item `from_parent`;
  - sorts the non-trigger items by raw score and caps them;
  - does no dedupe;
  - keeps only the local clusters;
  - drops the budget block.

  `show`/`show-chunk` fall back to the parent on a local miss. `learn`, `amend` and `activate` never contact the parent.

Joe's 2026-09-27 decision (vault note 784a, updating 784) replaces both with one local-first model that offers notes in both directions. Joe's rulings in review round 1 (all binding):

- **Q1:** no parent chunks in merged results. `show-chunk --parent` and the whole parent chunk path are removed.
- **Q2:** pull-down does a best-effort `/activate` on the parent.
- **Q3:** D5's classification is accepted as written.
- **Q4:** `/query-chunks` is deleted.
- **Q5:** `serve` gets no exemption from the `ENGRAM_SERVER` error.

Joe's further rulings: **M14**, accepted offers propagate upward; **B1**, a locally folded pulled note bouncing up once is intended. Both are recorded in D12.

Gaps in the current code that the design has to close:

- **G1.** `noteHasPendingMarker` (`offer.go:89-114`) treats a runbook as pending only if it also has `skill_hash`.
- **G2.** A served `/amend` sets `pending: true` on the parent's live note, which hides it.
- **G3.** The typed frontmatter rewriters drop keys that aren't declared. `resituate` hand-copies a partial field list (`resituate.go:223,256`) and drops `pending`, `sources`, `tags`, `supersedes` and `vocab_version`.
- **G4.** `GET /show` changes what it returns: it caps red flags and appends links.
- **G5.** `activate` knows only local sidecars. It skips misses silently and succeeds unless every ref failed.
- **G6.** Only `learn`/`learn qa` create the vault. `amend`, `activate` and `resituate` fail on the lock's `O_CREAT` when the vault is missing.
- **G7.** In the merge, explore picks crowd out direct matches (#744), and the budget block is all zeros (#743).
- **G8.** A served `learn` honors the client's `target`/`position`.
- **G9.** `embed.ContentHash` (`internal/embed/hash.go:49`) covers only situation and body text. Feedback `behavior`/`impact` and runbook `done_when`/`red_flags`/`triggers` edits leave it unchanged (review H1).
- **G10.** `update --reparent-luhmann` and adopt rename notes (`RenameAndRewriteReferences`, `luhmann_reparent.go:82`; `adoptRenderInput`, `skillreg_accept.go:300`). New top-level notes are exactly the ones that get renamed, and the offers are among them (review H2).
- **G11.** Every amend, including `--activate` and `--clear-pending`, re-stamps `repo`/`user`/`vault` (`amend.go:170-181`). Accepting a served offer therefore overwrites the child's declared identity with the host's (review M6).
- **G12.** The real vault's `.gitignore` is tracked in the vault's git. Appending to it would dirty that repo.

## Goals / Non-Goals

**Goals:**
- One mode. Every environment has a local vault, and when `ENGRAM_PARENT` is set it exchanges **notes** (never chunks) with the parent in both directions, through offers that the receiving side curates.
- Local writes always succeed. Offers are never lost offline, and retries never duplicate them.
- Merged recall shows one copy of each note, the local one, and nothing from the parent's chunk index.
- Exchange cannot loop. Every identity used in exchange survives renames, rewrites and hostname changes.
- A remote caller cannot set identity, registration or link fields.

**Non-Goals:**
- Several parents, or a mesh. The single-parent tree from note 784 stands.
- Near-duplicate detection in the merge. That is left to curation.
- Authentication on `engram serve` (unchanged trust model).
- Pushing note changes from the parent. The child pulls only on use (activate).

## Decisions

### D1: `ENGRAM_SERVER` is a hard error for every command, including `serve`

A single pre-dispatch guard in the targ entry exits non-zero before any vault, chunk-index or network access, with the message:

`ENGRAM_SERVER is no longer supported; set ENGRAM_PARENT=<same URL> instead — every environment now keeps its own local vault and offers notes to the parent`

Deleted:
- all ten `serverBase` branches, `serverBase` itself, `ExportServerBase`;
- `errDiscardOverServer` and `errRegisterSkillsOverServer`;
- `fetchQuery`, `fetchQueryChunks` and `fetchAmend`.

`fetchLearn`, `fetchActivate`, `fetchRaw` and `printOfferReceipt` are reused by the parent client (D6, D8). Subprocess tests strip `ENGRAM_SERVER` from their environment.

*Rejected:* an alias shim (Joe ruled it out); exempting `serve` (Q5).

### D2: Vault identity and state (folds in #766)

**Local vault on first use.** A shared `ensureVault` step runs in dispatch for every command that resolves a vault path: query, show, amend, activate, resituate, update, register-skills, serve, count, check, and the rest. If the vault is missing, it creates the vault (today's `initializeVault` starters) and prints one stderr line: `engram: created new vault at <path>`. That line makes a mistyped `--vault`/`ENGRAM_VAULT_PATH` visible, which answers #766's typo concern.

**Vault ID.** Every vault has a stable random ID in a tracked file, `<vault>/.engram-vault-id`, which holds 32 hex characters.
- `ensureVault` writes it when it creates a vault.
- For an existing vault, it is written lazily by only two things: `engram serve` at startup, and the first command that contacts a parent. Read-only use of an existing vault never creates it.
- Because the file is tracked, its first creation in an existing git-backed vault is a **deliberate one-time vault commit**: migration step 3 for Joe's real vault. `engram update` prints a notice while the file exists but is untracked or uncommitted.
- The server reports its vault ID in every exchange response (D4, D7, D8). Links are keyed by that ID, never by URL, so a hostname or IP change breaks nothing (review M11).
- **Creation (review r3-4).** The ID comes from a `RandRead` primitive: `crypto/rand` behind DI, the first random source in `cli.Primitives`, and faked in tests. It is written with the existing exclusive-create primitive (`WriteFileExcl`, O_CREATE|O_EXCL) and then **re-read**. If two processes race, both end up using whichever ID won.
- **Location record.** When the ID is created, the untracked `.engram/home.json` records `{vault_id, host, path}` (hostname and absolute vault path).
- **Copy and clone detection (r3-4).** Before any exchange (serve start, merge, offer, pull-down), the command checks `home.json` against the current vault ID, host and path. If `home.json` is missing, which is what a `git clone` looks like because `.engram/` is not tracked, or doesn't match, which is what a `cp -R` or a move to another path or host looks like, the command does no exchange (`serve` refuses to start) and prints one warning naming both remedies:
  - `engram vault-id --regenerate`: this is a copy. Mint a new ID, rewrite `.engram-vault-id` and `home.json`, and keep every note unchanged. Links point at *parent* IDs and stay valid, and outbox payloads take the new origin ID when they are sent.
  - `engram vault-id --claim`: this is the same vault, moved or re-cloned. Rewrite `home.json` only.

  `engram vault-id` with no flag prints the ID and the location check result.

  This covers the collision cases:
  - Two children sharing an ID (one cloned from the other) are caught at the clone's first exchange, before any offer carries a shared `offer.origin`.
  - A child cloned from its parent is caught by the same check. If `home.json` was also copied, it is caught by the self-parent guard below, whose warning names `engram vault-id --regenerate`.
- **Self-parent guard:** when the parent reports the local vault's own ID, the command does no exchange (no merge, no offers, no pull-down) and prints one warning naming `engram vault-id --regenerate`.

**State directory.** Transient exchange state lives in `<vault>/.engram/`:
- `outbox.json` (D6);
- `declined.json` (D8);
- `parent.json`, which caches `{url, vault_id, backoff_until, failures}` (D6);
- `home.json`, which holds the ID's location record (see above);
- `.gitignore` containing `*`, so git ignores the directory and everything in it, **including that `.gitignore` itself**.

The vault's tracked root `.gitignore` is never modified (G12). The directory is created at the same moment as the vault ID and holds no identity. If it is lost (for example, by cloning a vault), the only effect is that queued offers are re-queued on the next content write.

A copied vault (`cp -R`, as `runbook-retrieval-probe` does) carries the same ID and state, but its path differs from `home.json`, so it does no exchange until someone runs `--regenerate` or `--claim`. A read-only copy used by probes never exchanges at all.

*Rejected alternatives:*
- **Create only on `update`, or warn without creating** (#766's options 1 and 3). Joe chose "any command".
- **Put the outbox in the vault root and append it to the tracked `.gitignore`.** That dirties the vault repo on every host.
- **Put state under `$XDG_DATA_HOME` keyed by vault path.** Moving the vault orphans it, and it doesn't travel with the vault.
- **Key links by URL.** Round 1 did this. It breaks on hostname or IP changes (M11).

### D3: The exchange hash, over every offered content field (review H1)

`exchange hash = "xh1:" + sha256(canonical JSON of {type, situation, subject, predicate, object, behavior, impact, action, done_when, red_flags[], triggers[], body})`. `body` is `embed.BodyText` of the note, and absent fields are empty strings or empty lists. It is a pure function in `internal/cli`, and the server and child compute it the same way.

**It replaces `embed.ContentHash` everywhere in exchange:**
- the link `hash`;
- the loop-suppression check;
- `offer.key`;
- rejected re-arm;
- the send/apply change check;
- pull-down skip and decline matching;
- dedupe rule 2.

`embed.ContentHash` stays unchanged for sidecar staleness, which is its own job.

**Field classification (r3-5).** One explicit table lists every YAML key of the fact, feedback and runbook frontmatter structs as either *offered* (hashed) or *not offered*. A reflection test over the three structs' `yaml` tags fails when any key is missing from the table. A future field therefore can't silently fall outside the hash.

**Versioning (r3-5).** Hashes carry a version prefix (`xh1:`). Comparing two hashes gives one of three results: *equal*, *changed*, or *unknown*. The result is *unknown* when either side has a different or missing prefix. What each consumer does with *unknown*:
- **Loop suppression** fires only on *equal*. An *unknown* write is offered, which is harmless because server-side idempotency and in-place updates absorb it.
- **Pull-down skip, decline match, rejected-entry re-arm, and the send/apply change check** treat *unknown* as *not changed*. So a version bump causes no mass re-pull, re-arm or re-queue.
- **Dedupe rule 2** requires *equal*.
- A stale-version hash is replaced the next time that link or entry is written by an exchange.

**Stored hash (r3-5).** The receipt returns `stored_hash`: the exchange hash of the pending note as the parent actually wrote it. The child records that value in the link, not the hash it computed for what it sent.

**Hash stability.** The hash is stable across the exchange's own rewrites. None of the following touch a hashed field:
- the parent link;
- identity (`repo`/`user`/`vault`);
- `pending`;
- `tags`;
- `sources`;
- `supersedes`.

*Rejected alternatives:*
- **Extend `embed.ContentHash`.** That would make every existing sidecar look stale and force a full re-embed.
- **Hash the whole file.** Identity and link fields would change the hash, and loop suppression would never fire.

### D4: Links, aliases, and the note's exchange ID (review H2, H3, H5, M11)

Frontmatter (all notes, optional):

```yaml
xid: 7f3c…               # this note's own exchange id: random, stamped the first time the note is queued, pulled, or served-received; never changes
parent:                  # links to the configured parent's notes (multi-valued, review H5)
  vault: 9a1e…           # the parent's vault id (D2)
  links:
    - {note: 1100.2026-09-27.x, via: offered, hash: xh1:…}   # primary: this note's counterpart
    - {note: 0812.2026-08-01.y, via: covered, hash: xh1:…}   # parent notes this note was judged to cover
  author: {repo: …, user: …, vault: …}   # pulled notes only: the parent note's authorship (review M6)
aliases: [1101.2026-09-27.z, 12.2026-06-01.old-name]   # basenames this note answers to in ITS OWN vault
offer: {origin: <child vault id>:<child xid>, key: …, for: <basename>, path: [<vault id>, …]}   # served offers (D7); kept after acceptance
```

Every new field (`xid`, `parent` and its members, `aliases`, `offer` and its members) is `omitempty`. A note that never takes part in exchange serializes exactly as it does today.

**`xid` is stamped lazily and never backfilled.** A note gets its `xid` the first time it takes part in exchange: when it is queued as an offer, pulled down, or received as a served offer. No migration or `update` step stamps existing notes.

**Link roles (`via`).**
- `offered` / `pulled`: the note's primary counterpart. There is at most one primary, and amend-offers target it.
- `covered`: a parent note this local note was judged to cover when a pulled offer was folded into it.

**How fields are set.** Only the exchange paths may set or change `parent`, `aliases`, `xid` and `offer`:
- the offer receipt (D6);
- pull-down (D8);
- `amend --discard --into` (D10);
- a rename (below);
- served learn (D7).

All other rewrite paths preserve all four byte-for-byte, and each path gets a survival test (review M13, G3):
- `amend` (every flag);
- `resituate`;
- identity backfill;
- Luhmann reparent/rename;
- wikilink rewrite;
- vocab assign, clear, legacy cleanup, self-tag and version stamp;
- `scrubDeletedNoteReferences`;
- register-skills refresh and adopt.

**Renames record aliases (H2).** `RenameAndRewriteReferences` and the adopt rename append the old basename to the renamed note's `aliases`, in the same write. This keeps every other vault's reference to the old name resolvable:
- a child's links;
- a pending offer's `offer.for`;
- the raw show lookup.

On the child, the outbox is keyed by `xid`, which is rename-stable, so a child-side rename needs no outbox rewrite.

**Dedupe rule** (merged query, D9). A parent note item P and a live (non-pending) local note L are the same note if either of these holds:
1. L's `parent.vault` equals the parent's reported vault ID, and some link in `L.parent.links` names P's basename or one of `P.aliases`.
2. Both have a non-empty exchange hash, and the hashes are equal.

*Rejected alternatives:*
- **A single link per note** (round 1). This can't record that L covers P when L already has a counterpart, so P shows twice and is pulled forever (H5).
- **Keying the outbox and idempotency by basename** (round 1). A rename loses the entry and breaks idempotency (H2).
- **Rewriting outbox keys on rename.** This works, but `xid` makes it unnecessary.
- **Recording the child's basename on the parent.** It is remote-set, and the child can rename.

### D5: What is offered, and what stays local (Joe accepted in Q3 and M14)

| Write | Offered? |
| --- | --- |
| `learn fact`/`feedback`/`runbook` (not registration) | yes, a learn-offer |
| content `amend` (any of situation, subject/predicate/object, behavior/impact/action, done-when, body, red-flag, trigger) | yes: an amend-offer targeting the primary link when there is one, otherwise a learn-offer |
| `resituate` | yes, like a content amend |
| `amend --clear-pending` of a **served** offer (one carrying `offer.origin`), on a vault that itself has a parent | yes, a learn-offer upward (Joe, M14; D12). Not when the origin vault is the configured parent |
| `amend --activate`, `--clear-pending` (any other note), `--discard`, `--discard --into` | no, bookkeeping |
| `amend` with only `--supersedes`/`--chunk-source` | no |
| identity backfill | no (Joe's stated assumption) |
| `learn qa` notes | no |
| any note carrying `skill_hash` | no |
| a pending note | no |
| any write whose exchange hash equals the primary link's `hash` | no (loop suppression) |

**Payload** (built at send time, D6):
- It never carries `target`, `position` or `chunkSources`.
- Each `supersedes` entry is translated to that note's primary-link parent basename, or dropped if there is none.
- `user`/`repo` are the **note's own** frontmatter values; the draining caller's detection is used only when they are empty (review M5).
- It carries `offer.origin = <local vault id>:<xid>`, `offer.key`, and (for amend-offers) `offer.for`.

### D6: Outbox, drain, backoff (review M9, M10)

`<vault>/.engram/outbox.json`:

```json
{"version": 1,
 "entries": [{"xid": "<note xid>", "queued": "2026-09-27T10:00:00Z", "attempts": 2,
              "last_error": "…", "state": "queued|rejected", "rejected_hash": "xh1:…"}]}
```

- **Enqueue.** An offerable write (D5) stamps `xid` if it is missing and adds or refreshes the note's entry. This happens under the vault lock, in the same critical section as the note write. The local write never fails because of the parent.
- **Payload at send time.** The drain finds the note by `xid` (a frontmatter scan, rename-safe) and builds the payload from the note's current content. An offline learn followed by amends goes up as **one** offer. An entry whose note is gone, or is now pending, is dropped.
- **Order.** Oldest `queued` first.
- **Drain flow:**
  1. Read a snapshot of the outbox under the lock, then release the lock.
  2. Send each entry, with **no lock held**.
  3. Re-take the lock, **re-read the outbox from disk**, and merge (M10). For each sent entry:
     - If the note no longer exists, drop the entry and the receipt (the parent's pending offer is left to parent curation).
     - Otherwise, record the link as described under Receipt.
     - If the note's current exchange hash differs from the hash that was sent, keep the entry queued.
     - Entries enqueued concurrently are kept.
- **Idempotency.** `offer.key = sha256(offer.origin + exchange hash)`, and `offer.path = [local vault id]` (D7). The server's handling is in D7.
- **When it drains.** After any successful parent contact in the same command: `learn`, content `amend`, `resituate`, `query` (after the parent `/query` succeeds), `activate` (after a pull-down), and `update`.
- **Failures and backoff (M9).**
  - A transport error, timeout or 5xx stops the drain and records `attempts`/`last_error` on the entry.
  - It also sets `parent.json.backoff_until` to `now + min(15m, 30s × 2^(failures-1))`.
  - While backoff is in effect, every command except `update` skips parent contact. Query returns local-only results, and offers are only queued. One stderr warning is printed: `engram: parent unreachable (retry after <t>); N offer(s) queued`.
  - Parent requests use a 3-second connect timeout in addition to the existing 30-second total, so an unreachable host costs 3s at most once per backoff window.
  - Success resets `failures`.
  - A 4xx marks the entry `rejected` with its `rejected_hash`, and the drain continues. The entry re-arms only when the note's exchange hash changes.
- **Reporting.** `engram update` prints a notify-only outbox notice: the count, the oldest entry's age, the rejected entries, and the backoff state.
- **Receipt.** The receipt is `{status, luhmann, basename, pending: true, vault_id, stored_hash, for?}` (D7).
  - Under the lock, the receipt sets `parent.vault` and makes `{note: basename, via: offered, hash: <stored_hash>}` the primary link. The send/apply change check still compares the note's current hash with the hash that was *sent*.
  - When `for` is present, the primary becomes the resolved target's basename (H3).
  - The write is frontmatter-only: no re-embed and no identity re-stamp.
  - A receipt without `vault_id`/`basename` comes from a pre-change parent. The command stops exchange with an error saying the parent is too old (H7).

*Rejected alternatives:*
- **A per-offer snapshot directory.** It sends stale content and duplicates.
- **JSONL.** Coalescing needs rewrites.
- **Sending synchronously and failing the write.** That contradicts decision 4.
- **No backoff.** Up to 30s on every command while the parent is down (M9).

### D7: Server side: `/learn` is the only write route, and pending offers update in place (review H3, H7)

The served set becomes `query`, `show`, `activate` and `learn`:
- `POST /amend` is deleted. Its only caller was the thin client, and it hid live notes (G2).
- `GET /query-chunks` is deleted (Q4).
- `GET /show-chunk` is deleted (Q1: no parent chunk path).

A served learn is handled as follows:

- **Placement.** The server ignores `target`/`position` and places the note at top level (G8).
- **Loop refusal (r3-7).** `offer.path` lists every vault ID the offer has already passed through. The child sends `[own id]`, and propagation (D12) appends the propagating vault's ID. A server whose own ID is already in `offer.path` answers 409 and writes nothing. This covers misconfigured cycles of three or more vaults. A 409 is a 4xx, so the child marks the entry rejected.
- **Origin matching (H3, r3-3, r3-7).** All of the following happens in **one locked section**: the lookup, the rewrite, the re-embed, and building the receipt. The server checks these cases in order:
  1. **A pending note with the same `offer.origin`.**
     - If its `offer.key` also matches, write nothing and return its receipt.
     - Otherwise rewrite that pending note in place: its content, `offer.key` and `offer.path`. It stays pending and keeps its basename.
     - **Re-embed it** when its exchange hash changed.
     - Return its receipt.
  2. **A live note with the same `offer.origin`** (an offer already accepted, since `offer` survives acceptance).
     - If the `offer.key` matches, write nothing and return that live note's receipt. This is a retry after acceptance, and it must not create a second pending offer.
     - Otherwise write a new pending note with `offer.for` pointing at that live note.
  3. **Otherwise, resolve `offer.for`** against live basenames, then live notes' `aliases`, then pending notes.
     - If it resolves to a **pending note of a different origin**, the server does **not** overwrite it. It writes a new pending note whose `offer.for` names that pending note (no cross-origin overwrite).
     - If it resolves to a live note, the server writes a new pending note whose `offer.for` names it.
     - If it doesn't resolve, `offer.for` is dropped and the offer becomes a new pending note.
- **Receipt.** `{status: "offer received", luhmann, basename, pending: true, vault_id, stored_hash, for}`. `stored_hash` is the exchange hash of what the server stored. `for` is the resolved **live** target's basename, when there is one.
- **Pending detection (G1).** Any note type with `pending: true` is pending. The real vault was checked read-only on 2026-09-27 and holds zero pending notes.
- **Wire safety.** `LearnArgs` carries `Parent`, `Aliases`, `Xid`, `SkillHash`, `SkillKey` and `SkillSource` as `json:"-"`. The only new remote-settable fields are `offer.{origin,key,for,path}`, and they land only on pending notes, which curation reviews.

`GET /show?note=<basename>&raw=1` (H7) returns a JSON envelope, `{vault_id, basename, content, exchange_hash}`. `content` is the file's bytes verbatim, and `basename` is the current name after alias resolution. A missing note returns 404. A parent that predates this change returns rendered text, and the child detects this (not JSON, or no `vault_id`) and fails with "parent too old".

`GET /query?dedupe-keys=1` adds a top-level `vault_id` and, on each note item, `exchange_hash` plus a non-empty `aliases` list. Without the parameter the payload is byte-identical to a local query.

*Rejected alternatives:*
- **Keeping `/amend`.** That is two offer routes for one concept.
- **A new `/offer` route.** It duplicates `/learn`'s code path.
- **A raw text body for `/show?raw=1`.** An old parent's rendered response can't be told apart from it (H7).

### D8: Pull-down on activate (review M6, M8, H5; Q2)

`engram activate --note <ref>`, for each ref:

1. **Resolve locally.** A local hit means the note's `.md` exists (not its sidecar) (M8). On a hit, bump the sidecar `LastUsed`, as today.
2. **Parent candidate.** On a local miss with `ENGRAM_PARENT` set (or always with `--parent`), the ref is a parent candidate only if it is a basename or `<basename>.md`. A bare Luhmann ID never goes to the parent (M8), because IDs are minted per vault.
3. **Fetch.** With **no lock held**, fetch `/show?raw=1` and validate the envelope.
4. **Write under the lock.**
   - Re-check the skip rule. Skip when a local note has a link (any `via`) to the envelope's basename, or to any of its aliases, and that link's `hash` equals the envelope's exchange hash. Also skip when `declined.json` holds that basename with that hash. When the skip hits a live linked note, bump that note's `LastUsed`.
   - Otherwise write a new pending local note with:
     - a fresh top-level Luhmann ID and its own `xid`;
     - the parent body **verbatim**, so the exchange hash matches;
     - `pending: true`;
     - `parent.vault` set to the envelope's vault ID, with primary link `{note, via: pulled, hash}`;
     - `parent.author` set to the parent note's `repo`/`user`/`vault`.
   - Top-level `repo`/`user`/`vault` are stamped locally, as any local write is.
   - The parent's own `parent`, `aliases`, `offer`, `xid`, `skill_*`, `sources`, `supersedes` and `tags` are stripped. Local vocab reassigns tags.
   - The note is embedded on write.
   - A type other than fact, feedback or runbook is refused.
5. **Best-effort parent bump (Q2).** After releasing the lock, POST `/activate` for the parent note. A failure here is not queued and not fatal.

**Other activate behavior.**
- Each ref that could not be activated is reported on stderr. The command exits non-zero if any ref failed (the #746 addendum).
- **Declines (H5).** A bare `amend --discard` of a pulled note (one with a `via: pulled` primary link) adds `{basename, hash}` to `declined.json`. The decline check matches the envelope's basename **or any alias in the fetched content**, so a parent-side rename doesn't bring a declined note back (r3-8). A declined note is not pulled again unless its hash changes. A changed parent note is new information (Joe's decision 6).
- **Loop rule.** A pulled note is never offered back up:
  - it is pending;
  - `--clear-pending` is bookkeeping;
  - any later write whose exchange hash equals the primary link's hash is suppressed.

  A real local edit to a pulled note *is* offered, as an amend-offer targeting its origin. That is intended (decision 3), and so is the B1 bounce-once, which Joe confirmed (D12).

*Rejected alternatives:*
- **A `parent:` path prefix in payloads.** It breaks consumers, and bare `engram show <basename>` in the shim.
- **Writing the pulled note live.** That bypasses curation.
- **Reusing `/show`'s rendered output.** It is altered (G4).

### D9: Merged query is the primary path (Q1, review H4, M2, M3, M4; #743, #744)

- The child requests `dedupe-keys=1`. It **drops every parent item with `kind: chunk`**, including the parent's recency channel, before any other step (Q1). Channel 2 (recency, all chunks) is therefore local only, and so are chunk content budgets. `show-chunk --parent` and the `show-chunk` parent fallback are removed.
- **Dedupe** (D4 rule) runs before ordering and capping.
  - If L is already in the local items, P is dropped.
  - Otherwise L's item takes P's place. The **substituted item (M4)** has:
    - `path`: L's basename plus `.md`;
    - `kind: note`;
    - `content`: L's content, rendered exactly as a local query renders a note;
    - `score` and `provenances`: P's;
    - `model_id`: the local model;
    - `from_parent: false`;
    - no `source_term`, unless P's provenances include `explore`, in which case P's `source_term` is kept.
  - Dedupe keys are stripped from the output. Pending local notes never suppress parent items.
- **Ordering (#744, M2):**
  1. Trigger hits: local first, then parent.
  2. **Direct items**: every item whose `provenances` include none of `trigger`, `explore` or `recent`, from both sources, by descending `score`.
  3. **Explore items**: `provenances` include `explore`, from both sources, by descending `score`.

  `--limit` then caps the direct and explore items by position.
- **Note floor (M3).** Let Q be the direct items with `kind: note` and `score` ≥ 0.25. When the cap keeps fewer than `min(5, |Q|)` items from Q, the lowest-positioned kept direct `kind: chunk` items are replaced by the highest-scoring excluded members of Q, one for one, until that minimum is met or no kept direct chunk item remains. The replacements are placed back in score order.
- **Budget (#743).** The budget block reports the merge-applied values:
  - `limit`, `content_budget`, `lazy_chunks`;
  - `chunks_snippeted`, taken from `capChunkContent`'s return;
  - `explore_allocated`, as a term-to-count map summed per term.
- **Pending hint (H4).** `pending_offers`/`pending_offers_hint` reflect **local** pending offers only. A child can't curate its parent. The parent's own flag is ignored, and the parent's host sees its flag on its own queries.
- **#745: resolved by design, not by carrying clusters.** Every environment now has local clusters. Parent clusters are built from parent chunks, and chunks never travel (decision 7, Q1). Close #745 with this reasoning.
- **#746: superseded.** `learn`/`amend` offer automatically, and `activate` gains `--parent` plus the pull-down fallback (D8, which also fixes the addendum). Close #746 as superseded.

### D10: Curation: `--discard --into`, identity-preserving bookkeeping (review M6, M12)

- `engram amend --target O --discard --into E` deletes O and its sidecar. It unions O's basename **and all of O's `aliases`** into `E.aliases` (M12), and it unions O's `parent.links` into E's:
  - O's `offered` or `pulled` primary becomes a `covered` link on E when E already has a primary.
  - Otherwise it becomes E's primary.

  E is not re-embedded, and nothing is queued.
- **Judged-version check (r3-2).** An in-place update (D7) can land between curation's judgment and its bookkeeping, which would silently accept or discard content nobody judged. So `--clear-pending`, `--discard --into` and a bare `--discard` on a note carrying `offer.origin` **require** `--expect-hash <exchange hash>`, and fail without changing anything when the note's current hash differs. The curator then re-judges. `engram show` prints a note's current exchange hash as a header line (`# exchange_hash: xh1:…`), and the curate skill passes that value.
- **Bookkeeping amends no longer re-stamp identity (M6, G11).** `--activate`, `--clear-pending`, `--discard --into` and a link-only receipt write preserve `repo`/`user`/`vault`. So an accepted served offer (absent) keeps the child's declared identity.
  - For a covered or near fold, the offer's author is not carried onto E. E keeps its own author, because it is E's content. This loss is documented and accepted, and the offer's basename survives in `E.aliases`.
  - Content amends still re-stamp, as `vault-note-identity` requires. The requirement "Amend re-stamps identity fields on every write" is MODIFIED to say this.
- **`resituate` preserves every field it doesn't change (M7):** `pending`, `sources`, `tags`, `supersedes`, `vocab_version`, `xid`, `parent`, `aliases`, `offer`, `issue`, `project`. Its field-list hand-copy is replaced by a round-trip that changes only `situation` and the body opener.
- **Skills** are edited through writing-skills TDD, in hermetic headless arms (D11):
  - **curate:**
    - offers now also arrive by pull-down (`via: pulled`);
    - judge an offer with `offer.for` against that note first;
    - read the offer's `# exchange_hash` from `engram show` and pass it as `--expect-hash` on every bookkeeping step, re-judging if the check fails;
    - covered and near end with `--discard --into <existing>`;
    - discarding a pulled note outright records a decline;
    - curation stays host-local.
  - **recall:**
    - Step 2.7 activates the `from_parent` notes the agent used, which pulls them down;
    - after that, `pending_offers` reflects the local copies (curate them after the user's request);
    - a new red flag: never `amend` a `from_parent` item.
    - glance keeps its Step 2.7 activation, so a glance can pull a parent note down. This matches `recall-glance-deep-dial`: the pulled note is a pending offer, not live knowledge, and it is written by `activate`, not by learn or amend.
  - **learn:** writes are offered to the parent automatically, a "parent unreachable; N offer(s) queued" warning is not a failure, and `ENGRAM_SERVER` must never be set.

### D11: Tests: DI fakes, hermetic skill arms, and a real-binary run

- **Go unit tests** use the `Deps.Fetch` seam and the FS/lock fakes (imptest + rapid + gomega). They cover:
  - every failure mode of each exchange;
  - the exchange-hash field-coverage property: changing any offered field changes the hash, and changing any non-offered field does not;
  - outbox coalescing and FIFO;
  - never losing an offer;
  - rename-safety (xid);
  - dedupe (never both copies);
  - pull-down idempotency;
  - link survival at every rewrite site.
- **Skill TDD (H6, r3-1).** Each RED and GREEN arm is a fresh headless `claude -p` process, not a subagent (user feedback: subagents inherit session context).
  - **Environment.** Each arm is launched with `env -i` and an explicit allowlist, so none of this session's variables reach it: `CLAUDECODE`, `CLAUDE_CODE_SESSION_ID`, `CLAUDE_CODE_CHILD_SESSION`, `CLAUDE_CODE_ENTRYPOINT`, `CLAUDE_CODE_MESSAGING_SOCKET`/`_TOKEN`, `CLAUDE_CODE_BRIDGE_SESSION_ID`, `CLAUDE_PID` and the rest. `ENGRAM_PARENT`, `ENGRAM_SERVER` and `ENGRAM_VAULT_NAME` are therefore unset. A scenario that needs a parent sets `ENGRAM_PARENT` explicitly in the allowlist.
  - **Working directory.** The arm runs from a non-repo `$ARM/work`, so no project `CLAUDE.md` or rules load.
  - **Auth.** The access token is read from the macOS keychain item `Claude Code-credentials` (JSON field `claudeAiOauth.accessToken`). It is passed only as `CLAUDE_CODE_OAUTH_TOKEN`, and is never printed or written to a file.
  - **Permissions.** Permission mode is `--permission-mode bypassPermissions`, which is safe because everything the arm can touch is scratch.
  - **Verified invocation** (2026-09-27, claude 2.1.282, run from `$ARM/work`). A trivial prompt returned `PONG`. A second probe returned the marker from a skill installed only under `$ARM/home/.claude/skills/marker-probe/`, and answered NO to "does your context mention 'AI-Used' or 'engram query'", which shows that neither the real `~/.claude/CLAUDE.md` nor the project `CLAUDE.md` loaded. The command:

    ```
    TOK=$(security find-generic-password -s "Claude Code-credentials" -w | python3 -c 'import json,sys; print(json.load(sys.stdin)["claudeAiOauth"]["accessToken"])')
    cd "$ARM/work" && env -i HOME="$ARM/home" USER="$USER" PATH="$ARM/bin:/usr/bin:/bin" TERM=dumb \
      XDG_DATA_HOME="$ARM/xdg" ENGRAM_VAULT_PATH="$ARM/vault" CLAUDE_CODE_OAUTH_TOKEN="$TOK" \
      /Users/joe/.local/bin/claude -p "<prompt>" --permission-mode bypassPermissions
    ```

    `$ARM/bin/engram` is the branch build. The keychain token is short-lived, so it is re-read per arm batch.
  - **Skill placement.** The skill under test (old for RED, new for GREEN) is installed only at `$ARM/home/.claude/skills/<name>/SKILL.md`.

  Each arm is gated by a delivery check before scoring: a marker from the treatment text must appear in the transcript (vault note on verifying treatment delivery). Deploying with `engram update` happens only after merge and `go install`.
- **Real-binary verification.** `engram serve` on `127.0.0.1` over a scratch parent vault, a scratch child, and a recording TCP proxy in front of serve (serve does not log requests), so request counts are observable. See tasks group 11. `update`'s drain is covered only by unit tests: `update` runs `go install` and re-execs, so it gets no real-binary run in scratch. That is stated in the LEDGER row.

### D12: Multi-level propagation and bounce-once (Joe decided 2026-09-27)

- **M14: accepted offers propagate (decided: yes).** When curation accepts a served offer (`--clear-pending` on a note carrying `offer.origin`), that counts as a local learn at that level. It is enqueued as a learn-offer to that vault's own parent.
  - Near folds on that vault are content amends and go up anyway (D5).
  - **No loop:** an accepted note is never offered to the vault it came from. If the offer's origin vault ID (the part of `offer.origin` before `:`) equals the configured parent's vault ID, nothing is enqueued. In the single-parent tree this can happen only through a misconfiguration, but the check is unconditional.
  - **Longer cycles:** the onward offer carries the accepted note's `offer.path` with this vault's ID appended. Any server already on that path refuses it (D7), which covers misconfigured cycles of three or more.
  - Pulled notes (`via: pulled`) are never propagated, because their accept is bookkeeping (D8).
  - *Rejected:* stopping at the first level. It is inconsistent with near folds propagating, and it defeats the personal → team → org tree.
- **B1: the bounce-once is intended (decided: yes).** A pulled note P that local curation judges *near* is folded into local note L through a content amend. L is then offered up once: as an amend-offer targeting L's primary link, or as a learn-offer when L has none. Parent curation then judges P's content plus the local addition.
  - The exchange ends there. The parent's curated result comes back down only on the next use (activate), and only if its exchange hash changed.
  - *Rejected:* a `--from-pull` fold marker that suppresses the offer. It would lose the local addition upstream, which is the point of decision 3.

## Risks / Trade-offs

- **[Fleet impact]** Every environment that already sets `ENGRAM_PARENT` starts offering all of its learns and content amends on upgrade, so the host's curation queue grows by the fleet's write rate. → The host's `pending_offers` hint and `update` notice surface it, and curation is expected upkeep. The proposal's Impact calls this out.
- **[An unlinked near-duplicate shows twice]** → Accepted (decision 5). Use heals it through pull-down and fold links.
- **[A covered fold loses the offer author's attribution]** → Documented (D10). The basename survives in `aliases`.
- **[Version skew]** → A pre-change parent is detected by a missing `vault_id` in the receipt, the envelope or the query payload. Exchange stops with a "parent too old" error, and queries stay local-only with a warning. Upgrade the parent first.
- **[The tracked `.engram-vault-id` dirties an existing vault's git once]** → This is a deliberate one-time commit (migration step 3), and `update` flags it until it is committed.
- **[`.engram/` state is lost on a fresh clone]** → Queued offers re-queue on the next content write. Links and IDs live in tracked files, so nothing that matters is lost.
- **[A remote client aims `offer.for` or `offer.origin` at arbitrary notes]** → It can only create or update pending notes, and it can't change a live note. Curation reviews every one. `offer.origin` impersonation can overwrite another child's *pending* offer, which is the same trust level as today's unauthenticated serve (network reachability).
- **[Drain latency on query]** → The drain runs only after a successful parent contact, with backoff and a connect timeout, and usually has nothing to send.

## Migration Plan

1. Upgrade the parent host first: vault ID, receipt, envelope, dedupe keys, in-place pending, route removals.
2. Upgrade the children. Any host with `ENGRAM_SERVER` set gets the hard error and switches to `ENGRAM_PARENT`. The first command creates the local vault.
3. On a git-backed vault, commit the new `.engram-vault-id` once, deliberately. **When this happens on Joe's real vault:** the ID file is created only when `engram serve` next starts on it after the upgrade (the host's served vault), or at its first parent contact if that host ever sets `ENGRAM_PARENT`. The commit (`vault: add vault id`) is made right after that first creation. `engram update` keeps flagging the untracked file until it is committed. If neither event ever happens on that vault, no file is created and the step is skipped.
4. Rollback: revert. Older binaries' typed rewriters drop `xid`/`parent`/`aliases`/`offer`, which is harmless. The `.engram/` directory is ignored.

## Open Questions

None. Q1–Q5, M14 and B1 are all decided by Joe (2026-09-27).
