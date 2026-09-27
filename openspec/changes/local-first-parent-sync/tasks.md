> Every Go task follows the full red/green/refactor cycle: first write the failing test (imptest mocks, rapid properties where the task names one, gomega), then the code, then `targ check-full`. No test or verification step touches the real vault, `~/.local/share/engram`, or the real `~/.claude`. All HTTP goes through the `Deps.Fetch` seam with fakes. `E#` points to a row in `enumeration.md`. A task that performs a row is ticked only after the new text is grepped present and the old text grepped absent (vault note 1072).

## 1. Remove the `ENGRAM_SERVER` thin client (design D1)

- [ ] 1.1 RED: `TestEngramServerSet_HardErrorsEveryCommand`, a table over every subcommand **including `serve`**. Each one:
  - exits non-zero;
  - names `ENGRAM_PARENT` and the same URL;
  - makes zero FS, lock, Fetch or bind calls.

  A second test: `ENGRAM_SERVER` alone never produces a Fetch call. (E54)
- [ ] 1.2 GREEN: add the pre-dispatch guard, and delete every thin-client branch, function and error. (E1–E9, E11, E17, E19, E61)
- [ ] 1.3 Delete or rewrite the thin-client tests and comments. Make the subprocess env helpers strip `ENGRAM_SERVER`. (E53, E55, E57–E60, E62, E63)
- [ ] 1.4 Rewrite the Go comments and flag docs that name `ENGRAM_SERVER`. (E21, E35, E39, E48–E50)

## 2. Vault, vault ID, and exchange state (design D2, closes #766)

- [ ] 2.1 RED:
  - Every vault-resolving command (query, show, amend, activate, resituate, update, register-skills, serve, count, check) on a missing path creates the `.obsidian/` starters, `.engram-vault-id`, and `.engram/` (with a `.gitignore` of `*`), then prints one creation line. A second run creates and prints nothing.
  - `amend`/`activate` on a missing vault fail only with "not found", never with a lock error.
  - The existing root `.gitignore` is never modified.
  - Read-only use of an existing vault with no parent writes nothing.
  - `serve` stamps a missing vault ID, and later starts reuse it.
  - `update` warns when a git-backed vault's ID file is untracked.
  - Self-parent guard: when the parent reports the local ID, there is no merge, offer or pull-down, and one warning is printed.
- [ ] 2.2 GREEN: shared `ensureVault` in dispatch, plus an `exchangestate` adapter (vault ID and `.engram/`) behind DI. (E30, E46, E47, E52)

## 3. Exchange hash, `xid`, links, aliases (design D3, D4)

- [ ] 3.1 RED→GREEN: the exchange hash as a pure function. Rapid properties:
  - changing any one offered field (per type) changes the hash;
  - changing any one non-offered field (`repo`, `user`, `vault`, `pending`, `tags`, `sources`, `supersedes`, `xid`, `parent`, `aliases`, `offer`, `skill_*`) does not.

  Also a golden test that the server and the child compute the same value for the same file. (E52)
- [ ] 3.2 RED: exchange-field survival tests next to `skillfields_survival_test.go`. For every rewrite site listed in `vault-note-identity`, with a note carrying `xid`, `parent` (vault, links incl. `covered`, author), `aliases` and `offer`, all four survive byte-equal. The sites are:
  - amend (content, `--supersedes`, `--activate`, `--clear-pending`);
  - resituate;
  - identity backfill;
  - reparent/rename;
  - wikilink rewrite;
  - vocab assign, clear, legacy cleanup, self-tag and version stamp;
  - `scrubDeletedNoteReferences`;
  - register-skills refresh and adopt.

  Include a rapid property over random notes. (E69)
- [ ] 3.3 GREEN: add `xid`, `parent{vault, links[{note,via,hash}], author{repo,user,vault}}`, `aliases` and `offer{origin,key,for}` to the fact, feedback and runbook frontmatter structs, and hand-copy them wherever a site re-renders. (E34, E42, E44)
- [ ] 3.4 RED→GREEN: `resituate` preserves every untouched field (`pending`, `sources`, `tags`, `supersedes`, `vocab_version`, `issue`, `project`, and the exchange fields), changing only `situation` and the body opener. Replace the field-list hand-copy with a round-trip. (E41, E70; review M7)
- [ ] 3.5 RED→GREEN: a rename appends the old basename to `aliases` in the same write (`RenameAndRewriteReferences`, and the adopt rename). A rename keeps `xid`. (E43, E44; review H2)
- [ ] 3.6 RED→GREEN: bookkeeping amends (`--activate` alone, `--clear-pending`, `--discard --into` on the target, link-only receipt writes) do not re-stamp `repo`/`user`/`vault`. Content, `--supersedes` and `--chunk-source` amends still do. Update the existing expectations. (E37, E71; review M6)
- [ ] 3.7 RED→GREEN: the wire. Served-learn JSON carrying `parent`/`aliases`/`xid`/`skill*` (camelCase and PascalCase) never populates them. Tag the fields `json:"-"`, and make `offer` the only new decoded field. (E33, E66)

## 4. Serve side (design D7)

- [ ] 4.1 RED→GREEN: every note type with `pending: true` is pending. Invert `offer_test.go:136-147`. Before the change, re-confirm read-only that the real vault holds zero `pending: true` notes. (E31, E67)
- [ ] 4.2 RED→GREEN: `ServeRoutes` is exactly query, show, activate and learn. `/amend`, `/query-chunks` and `/show-chunk` have no route. Delete the handlers and their tests. (E23–E25, E64, E65)
- [ ] 4.3 RED→GREEN: served learn:
  - top-level placement;
  - a new pending note gets a server `xid` and records `offer.{origin,key,for}`;
  - same `offer.origin` pending → update in place (the H3 scenario: an amend before curation leaves exactly one pending note);
  - same key → no write;
  - `offer.for` resolves live → alias → pending;
  - unresolved → dropped;
  - receipt `{status, luhmann, basename, pending, vault_id, for?}`.

  (E26, E27)
- [ ] 4.4 RED→GREEN: `/show?raw=1` returns a JSON envelope `{vault_id, basename, content, exchange_hash}`; `content` is byte-exact; it resolves through `aliases`; a missing note is a 404. Non-raw output is unchanged. (E28; review H7)
- [ ] 4.5 RED→GREEN: `/query?dedupe-keys=1` adds a top-level `vault_id` and per-note `exchange_hash`/`aliases`. Without the parameter the payload is byte-identical to a local query. (E29)
- [ ] 4.6 Update the serve description and `offer.go` comment. (E15, E32)

## 5. Offers, outbox, backoff (design D5, D6)

- [ ] 5.1 RED: outbox model over FS fakes:
  - `xid`-keyed coalescing;
  - FIFO;
  - payload built at send time;
  - a deleted or pending note's entry is dropped;
  - a renamed note is still found by `xid`;
  - transport/5xx stops the drain and records attempts/error;
  - 4xx rejects the entry and re-arms it only on an exchange-hash change;
  - send with no lock held (the lock/fetch call order is asserted);
  - re-read and merge under the lock after sending: a concurrent enqueue survives, and a receipt for a deleted note is discarded (M10);
  - atomic save.

  Rapid properties: no offer is lost while its note exists and is offerable; at most one entry per note. (E52)
- [ ] 5.2 RED: backoff. After a failure, commands inside the window make no parent request (query returns local results, and each command prints one warning). The window grows exponentially from 30s to a 15-minute cap and resets on success. `update` ignores the window. The connect timeout is ≤ 3s (a primitive, asserted by config). (E51; review M9)
- [ ] 5.3 GREEN: the outbox and backoff in `internal/cli/outbox.go` and `exchangestate.go`, behind DI, wired through `Primitives`/`NewDeps` (thin-api clean).
- [ ] 5.4 RED→GREEN: offer classification as one pure function over (command, flags, note, configured parent's vault ID). It covers every row of D5's table, including:
  - accepted served offers propagating (M14);
  - suppression when the origin vault is the parent;
  - `learn qa`, `--supersedes`-only, `skill_hash` and pending notes not offered;
  - primary-link hash suppression;
  - the near-fold bounce-once (B1) queuing exactly one offer.
- [ ] 5.5 RED→GREEN: the payload builder:
  - learn-offer vs amend-offer (`offer.for` from a primary link under the current parent vault ID);
  - no `target`/`position`/`chunkSources`;
  - `supersedes` translated or dropped;
  - the note's own `user`/`repo`, falling back to detection only when empty (M5);
  - `offer.origin` and `offer.key`.

  (E18, E20)
- [ ] 5.6 RED→GREEN: applying a receipt:
  - a frontmatter-only write of `parent.vault` plus the primary link (with no re-embed and no re-stamp);
  - the `for` re-link (H3);
  - a changed hash keeps the entry queued;
  - no `vault_id` → "parent too old", and the entry stays queued.

  (E22)
- [ ] 5.7 GREEN: wiring:
  - learn and content amend/resituate stamp `xid` and enqueue in the same lock section as the write, then drain;
  - `--clear-pending` on a served offer enqueues upward (M14);
  - query drains after the parent query succeeds;
  - update drains;
  - activate drains after a pull-down.

  (E36, E38, E41, E47)
- [ ] 5.8 RED→GREEN: the `update` notice (count, oldest age, rejected entries, backoff), and nothing when idle.

## 6. Pull-down on activate (design D8)

- [ ] 6.1 RED:
  - A local hit means the `.md` exists (it bumps the sidecar, and a note without a sidecar still counts as a hit); no Fetch.
  - A bare Luhmann ID is never sent to the parent.
  - A basename miss triggers a raw-envelope fetch with **no lock held**, then under the lock a pending note with: a fresh top-level ID and `xid`; the body verbatim (equal exchange hash); `parent.vault` plus a `via: pulled` link; `parent.author`; local top-level identity; the listed fields stripped.
  - A non-fact/feedback/runbook type is refused.
  - A non-envelope response gives "parent too old", and nothing is written.
  - `--parent` without `ENGRAM_PARENT` is an error.
  - Failures are reported per ref, and the command exits non-zero on any failure.
  - The best-effort parent `/activate` happens after the lock is released; its failure is neither fatal nor queued (Q2).

  (E40, E16)
- [ ] 6.2 RED: the skip rule, re-checked under the write lock:
  - an unchanged re-activation writes nothing and bumps the linked live note;
  - a `covered` link skips;
  - a bare `--discard` of a pulled note records a decline, and an unchanged re-pull writes nothing;
  - a changed hash creates a new pending offer;
  - `--clear-pending` on a pulled note queues nothing;
  - a local content edit queues an amend-offer to its origin.

  Rapid property: any sequence of activate, clear-pending, discard and fold on an unchanged parent note never leaves more than one local note linked to it. (Review H5, M8)
- [ ] 6.3 GREEN: `internal/cli/pulldown.go`, and the activate dispatch. (E10, E52)

## 7. Merged query as the primary path (design D9)

- [ ] 7.1 RED: parent `kind: chunk` items are dropped, including from the parent's recency channel, before any other step. `show-chunk` never contacts the parent, and `--parent` is unknown for it. (E12–E14, E56; Q1)
- [ ] 7.2 RED: one dedupe test for each `vault-merged-recall` dedupe scenario:
  - primary link;
  - `covered` link;
  - alias;
  - equal hash;
  - pending local copy does not suppress;
  - substitution with exactly the M4 fields;
  - near-duplicates left alone;
  - the keys stripped from the output;
  - a link under a different vault ID ignored.

  Rapid property: the output never contains both a live local note and a parent item it matches.
- [ ] 7.3 RED: ordering and floor (#744, M2, M3). Direct before explore, and `--limit` by position. The floor fixture from the spec scenario (5 chunks at 0.9 plus 3 notes at 0.5 under `--limit 5`). The issue's reproduction shape (10 phrases, explore scores 0.75–0.89).
- [ ] 7.4 RED: the budget block (#743), and the pending hint reflecting local offers only (H4).
- [ ] 7.5 GREEN: rework `mergeQueryPayloads`/`runMergedQuery`, including the backoff check, the self-parent guard, and the drain after success. (E45)
- [ ] 7.6 After merge: close #745 (design D9 reasoning), #746 (superseded, citing D8), #743, #744 and #766.

## 8. Curation fold (design D10)

- [ ] 8.1 RED→GREEN: `engram amend --target O --discard --into E`:
  - deletes O and its sidecar;
  - unions O's basename and all of O's `aliases` into E's `aliases` (M12);
  - merges O's links into E's (O's primary becomes `covered` when E has a primary);
  - no re-embed, no re-stamp, no offer.

  A bare `--discard` is unchanged, except that it records a decline for a pulled note. `--into` without `--discard` is an error. (E38)

## 9. Skill edits (design D10, D11): `superpowers:writing-skills` TDD in hermetic headless arms

> Each RED and GREEN arm is a fresh `claude -p` process, never a subagent (subagents inherit session context). Each arm runs with:
> - `HOME=$ARM/home`, `XDG_DATA_HOME=$ARM/xdg`, `ENGRAM_VAULT_PATH=$ARM/vault`;
> - `PATH=$S/bin:$PATH`, where `$S/bin/engram` is built from this branch;
> - the skill under test (old for RED, new for GREEN) installed only at `$ARM/home/.claude/skills/<name>/SKILL.md`;
> - auth only through env (`CLAUDE_CODE_OAUTH_TOKEN` or `ANTHROPIC_API_KEY`);
> - the vault state seeded by the scenario script;
> - a `timeout`, with no `--max-turns`.
>
> Before scoring, gate each arm on treatment delivery: a marker phrase unique to the arm's SKILL.md must appear in its transcript. Otherwise the arm is invalid, not a result. The real `~/.claude` and the real vault are never read or written. Deploying with `engram update` is task 12.3, after merge.

- [ ] 9.1 curate. The scenario vault holds a pulled-down pending offer covered by a local note, a served amend-offer with `offer.for`, and a pulled note the agent should reject.
  - RED expectation: a bare `--discard` (the link is lost), `offer.for` ignored.
  - Edit per E104–E108.
  - GREEN: `offer.for` is judged first, covered/near end with `--discard --into`, and the rejection uses a bare `--discard`.
- [ ] 9.2 recall. The scenario is a scratch child plus a scratch parent `serve`, whose merged payload has a used `from_parent` note.
  - RED expectation: activation is skipped, or `engram amend` is tried on the parent note.
  - Edit per E109–E111.
  - GREEN: the agent activates it (the pending copy appears), never amends it, and treats the local `pending_offers` as after-task curation.
- [ ] 9.3 learn. The scenario is a learn under `ENGRAM_PARENT` pointed at a dead port, which prints "parent unreachable … 1 offer(s) queued".
  - RED expectation: the agent treats it as a failure, retries, or suggests `ENGRAM_SERVER`.
  - Edit per E103.
  - GREEN: the agent treats the write as done.

## 10. Docs (enumeration sections D and E)

- [ ] 10.1 Perform E88–E95 (README, GLOSSARY line edits).
- [ ] 10.2 Perform E96 (GLOSSARY entries) and E98 (ADR-0029, plus the ADR-0027 pointer).
- [ ] 10.3 Perform E99–E100 (C1/C2/C3).
- [ ] 10.4 Perform E101 (ROADMAP) and E102 (LEDGER row, after group 11).

## 11. Real-binary verification (scratch only, never the real vault)

- [ ] 11.1 Set up the scratch environment:
  - `S=$(mktemp -d)`, then `go build -o $S/bin/engram ./cmd/engram`.
  - Record the real vault's `git -C ~/.local/share/engram/vault rev-parse HEAD` and `status --porcelain` before and after group 11. Both must be unchanged.
  - Parent: `XDG_DATA_HOME=$S/p-xdg $S/bin/engram serve --addr 127.0.0.1:$PP --vault $S/p-vault`, with `ENGRAM_PARENT`/`ENGRAM_SERVER` unset in its env.
  - In front of it, a **recording TCP proxy** on `127.0.0.1:$PX` that appends `method path` per request to `$S/requests.log`. A small script in `$S` is enough. Serve does not log requests, so every "no request" or "N requests" check reads this log.
  - Child: every command runs with `XDG_DATA_HOME=$S/c-xdg ENGRAM_PARENT=http://127.0.0.1:$PX`, from `cd $S` (not a repo).
- [ ] 11.2 Check first use and the hard error:
  - The first child `query --vault $S/c-vault` creates the vault, the vault ID and `.engram/`, and prints one creation line.
  - A child `query` **without** `--vault` creates and uses `$S/c-xdg/engram/vault`, which is the default-path resolution.
  - `ENGRAM_SERVER=http://127.0.0.1:$PX` on `query` and on `serve --addr 127.0.0.1:$PX2` exits non-zero and names `ENGRAM_PARENT`. `requests.log` does not grow, and `$PX2` is not bound.
- [ ] 11.3 Offline and drain:
  - Stop the proxy. A child `learn fact` exits 0, the note is live, and there is one warning plus one outbox entry. Amend it twice: still one entry. A second command inside the backoff window adds no connection attempt (measure its elapsed time).
  - Restart the proxy and wait out the backoff. A child `query` empties the outbox. The log shows exactly one `POST /learn`, the parent has one pending note carrying the latest content, and the child note carries `parent.vault` plus the primary link to the receipt's basename.
- [ ] 11.4 Pending in place, curation, dedupe:
  - Amend the child note again before curating on the parent, then run a draining query. The parent still has **one** pending note, now updated (H3).
  - `--clear-pending` it on the parent (absent). Its `user:` is still the child's declared user.
  - A child merged `query` shows the note once, with `from_parent: false`.
  - Repeat with a second offer folded on the parent through `--discard --into <existing>`. The child still sees one copy, through the alias.
  - Run `update --reparent-luhmann` on the parent, if it has a candidate, or rename through adopt. The child still dedupes, through the rename alias.
- [ ] 11.5 Pull-down:
  - Learn a note directly on the parent, then `activate --note <parent basename>.md` on the child. A pending local copy appears with `via: pulled` and `parent.author`, and `requests.log` shows `GET /show` plus `POST /activate`.
  - The next child query shows `pending_offers: true`, the parent item is still visible, and no parent chunk is present.
  - Activate again: nothing is written.
  - `--clear-pending` the copy on the child, **then run a draining child query**. The parent's pending count and `requests.log` show no `POST /learn`.
  - Change the note on the parent and activate again: a second pending offer appears.
  - Discard it outright on the child and activate again: nothing is written (the decline).
- [ ] 11.6 Multi-level (M14): add a grandparent `serve` and point the parent's `ENGRAM_PARENT` at it through a second recording proxy. A child offer accepted on the parent reaches the grandparent as a pending offer. A pulled note accepted on the child never does.
- [ ] 11.7 Record the outcomes, with the exact commands, in the LEDGER row (E102). State that `update`'s drain is covered only by unit tests: `update` runs `go install` and re-execs, so it gets no real-binary run in scratch. Any empty or missing field that the design says is populated means the task is not done.

## 12. Close-out

- [ ] 12.1 `targ check-full` is clean. Run the requirement-collision sweep (vault note 744) again across all active changes and any archived-but-unsynced change, and read every sibling's tasks.md in full (vault note 757).
- [ ] 12.2 Run a fresh-context implementation review with argumentation. Then rebase on main, re-test, and `git merge --ff-only`.
- [ ] 12.3 After the merge and `go install ./cmd/engram`: deploy the skills with `engram update`, and verify the deployed copies are byte-identical to the sources. The registration refresh offers for the skill mirrors (E112) go through normal curation.
- [ ] 12.4 Archive with `/opsx:archive`, after running the note-651 scenario parity diff and the E73/E76/E80/E82 Purpose rewrites. **Then** run the E72 sweep against the post-archive tree, with its allowlist.
- [ ] 12.5 On Joe's real vault, after deployment, commit the new `.engram-vault-id` as a single deliberate vault commit (`vault: add vault id`).
