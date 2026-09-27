> Every Go task follows the full red/green/refactor cycle: first write the failing test (imptest mocks, rapid properties where the task names one, gomega), then the code, then `targ check-full`. No Go test touches the real vault or `~/.local/share/engram`. Every HTTP interaction goes through the `Deps.Fetch` seam with fakes. Row numbers (`E#`) point into `enumeration.md`. A task that performs an enumeration row is ticked only after the new text is grepped present and the old text grepped absent (vault note 1072).

## 1. Remove the `ENGRAM_SERVER` thin client (design D1)

- [ ] 1.1 RED: `TestEngramServerSet_HardErrorsEveryCommand`, a table over every subcommand including `serve`. With `Getenv("ENGRAM_SERVER")` non-empty, each subcommand exits non-zero, the message names `ENGRAM_PARENT` and the same URL, and there are zero FS/lock/Fetch calls. Also a subcommand-level test that `ENGRAM_SERVER` alone never produces a Fetch call. (E47)
- [ ] 1.2 GREEN: add the pre-dispatch guard in the targ entry. Delete every `serverBase` branch, `serverBase`, `errDiscardOverServer`, `errRegisterSkillsOverServer`, `fetchQuery`, `fetchQueryChunks`, `fetchAmend` and `ExportServerBase`. (E1–E9, E11, E14, E16, E53)
- [ ] 1.3 Delete or rewrite the thin-client tests and comments. Make the subprocess env helpers strip `ENGRAM_SERVER` too. (E46, E48–E52, E54, E55)
- [ ] 1.4 Rewrite the Go comments and flag docs that name `ENGRAM_SERVER`. (E18, E29, E32, E40–E43)

## 2. Local vault on first use (design D2, closes #766)

- [ ] 2.1 RED: for each vault-resolving command (query, show, amend, activate, resituate, update, register-skills, serve, count, check), with a missing vault path it creates `.obsidian/` plus starters, prints exactly one creation line, and on a second run creates nothing and prints nothing. `amend`/`activate` on a missing vault fail on "target not found", never on a lock error. An existing `.gitignore` gains `.engram-outbox.json` exactly once.
- [ ] 2.2 GREEN: add the shared `ensureVault` step in dispatch, add the outbox line to the vault starter `.gitignore`, and remove the per-command `ensureVaultDir` calls it supersedes. (E38)

## 3. Parent link and aliases in frontmatter (design D3)

- [ ] 3.1 RED: add `parent`/`aliases` survival tests next to `skillfields_survival_test.go`, one per rewrite site: amend (content, `--activate`, `--clear-pending`), resituate, identity backfill, Luhmann reparent/rename, wikilink rewrite, vocab assign/clear/legacy cleanup, `scrubDeletedNoteReferences`, register-skills refresh/adopt. Add a rapid property: a random note with random `parent`/`aliases` put through any non-exchange rewrite keeps both byte-equal. (E59)
- [ ] 3.2 GREEN: add a nested `parentLink{URL, Note, Via, Hash}` and `Aliases []string` to the fact, feedback and runbook frontmatter structs. Hand-copy them in resituate's re-render, identity backfill, and `applySkillNoteBody`. (E28, E34–E36)
- [ ] 3.3 RED→GREEN: a served-learn test proving that JSON bodies carrying `parent`, `aliases`, `Parent` or `Aliases` (camelCase and PascalCase) never populate them. Tag the fields `json:"-"`. (E27; `vault-note-identity` "Origin fields SHALL NOT be settable")
- [ ] 3.4 RED→GREEN: `ENGRAM_PARENT` URL normalization is a pure function with a rapid property: idempotent, case-insensitive on scheme and host, trailing slash stripped. A link counts only when its `url` equals the normalized configured parent.

## 4. Serve-side changes (design D4, D7, D8 raw show)

- [ ] 4.1 RED→GREEN: `noteHasPendingMarker` treats any type with `pending: true` as pending. Invert `offer_test.go:136-147`. Before the change, re-confirm read-only that the real vault holds zero `pending: true` notes (`grep -l '^pending: true' ~/.local/share/engram/vault/*.md`, read-only). (E25, E57)
- [ ] 4.2 RED→GREEN: `POST /amend` is gone. `ServeRoutes` lists exactly query, query-chunks, show, show-chunk, activate, learn, and an `/amend` request gets no route. Delete `serveAmend` and its tests. (E20, E56)
- [ ] 4.3 RED→GREEN: the served-learn receipt is `{status, luhmann, basename, pending: true}`. `target`/`position` are ignored (the note is placed top-level). `offer.for` resolves against live basenames and then `aliases`, and is dropped when it doesn't resolve. `offer.key` dedupe returns the existing pending note's receipt with no write. (E21, E22)
- [ ] 4.4 RED→GREEN: `GET /show?raw=1` is byte-identical to the note file, and a missing note returns 404. The non-raw output is unchanged. (E23)
- [ ] 4.5 RED→GREEN: `GET /query?dedupe-keys=1` adds `content_hash` (from the sidecar) and non-empty `aliases` to note items. Without the parameter the output is byte-identical to a local query. (E24)
- [ ] 4.6 Update `targets.go`'s serve description and `offer.go`'s comment. (E12, E26)

## 5. Offers and outbox (design D5, D6)

- [ ] 5.1 RED: outbox model tests (pure functions over an FS fake):
  - enqueue coalesces per basename;
  - FIFO drain order;
  - a transport error or 5xx stops the drain and records `attempts`/`last_error`;
  - a 4xx marks the entry `rejected` with `rejected_hash`, continues, and re-arms only on a content-hash change;
  - a deleted or pending note's entry is dropped;
  - atomic temp-rename save.

  Rapid properties: (a) for any sequence of enqueue/drain/fail events, no note is ever lost while its note exists and is offerable; (b) at most one entry per note. (E44)
- [ ] 5.2 GREEN: `internal/cli/outbox.go` behind a DI FS interface wired through `Primitives`/`NewDeps` (thin-api clean).
- [ ] 5.3 RED→GREEN: offer classification (D5) as one pure function over (command, flags, note). Cover every row of design D5's table, including the stated assumptions (identity backfill, resituate, qa, `--supersedes`-only) and the `parent.hash` equality suppression.
- [ ] 5.4 RED→GREEN: payload builder. It builds a learn-offer or an amend-offer (`offer.for` from a current-URL link), never sends `target`/`position`/`chunkSources`, translates or drops `supersedes`, sets `offer.key = sha256(user + basename + content hash)`, and sets the declared `user`/`repo`. (E15, E17)
- [ ] 5.5 RED→GREEN: applying a receipt writes the `parent` link as a frontmatter-only change under the lock, with no re-embed (sidecar vector unchanged) and no identity re-stamp. An amend-offer keeps `note`. A content change between send and apply keeps the entry queued. A missing `basename` records no link and prints one warning. (E19)
- [ ] 5.6 GREEN: wiring. `learn` (fact, feedback, runbook) and content `amend` and `resituate` enqueue in the same lock section as the write, then drain. `query` drains after the parent `/query` succeeds, `update` drains, and `activate` drains after a pull-down. One stderr warning is printed when entries remain queued. HTTP is never sent under the vault lock (a test asserts the lock/fetch call order). (E30, E31, E34, E39)
- [ ] 5.7 RED→GREEN: `engram update` shows an outbox notice (count, oldest age, rejected entries with their errors), and nothing when the outbox is empty.

## 6. Pull-down on activate (design D8)

- [ ] 6.1 RED:
  - a local hit bumps the sidecar and makes no Fetch;
  - a local miss with `ENGRAM_PARENT` set does a raw fetch and writes a pending local note with a fresh top-level Luhmann ID, the body verbatim (content hash equals the parent's), and `parent{url, note, via: pulled, hash}`;
  - it strips `parent`/`aliases`/`offer`/`skill_*`/`sources`/`supersedes`/`tags` and keeps `repo`/`user`/`vault`;
  - it refuses a non-fact/feedback/runbook type;
  - `--parent` without `ENGRAM_PARENT` is an error;
  - it reports failures per ref and exits non-zero on any failure;
  - it sends a best-effort parent `/activate` whose failure does not fail the command;
  - an old parent (no raw mode) gives a clear error. (E33, E45, E13)
- [ ] 6.2 RED: idempotency and loops.
  - An unchanged re-activation writes nothing, and bumps the linked live note's `LastUsed`.
  - A changed parent hash writes a second pending offer.
  - `--clear-pending` on a pulled note queues no offer.
  - A local content edit of a pulled note queues an amend-offer targeting its origin.
  - Rapid property: activate → clear-pending → activate on an unchanged parent never grows the vault beyond one note per parent note.
- [ ] 6.3 GREEN: `internal/cli/pulldown.go`, and the activate dispatch in `targets.go`. (E10)

## 7. Merged query as the primary path (design D4 dedupe, D9)

- [ ] 7.1 RED: dedupe tests for each `vault-merged-recall` dedupe scenario: link by basename, link by alias, identical hash, pending local copy not suppressing, substitution when the local copy is unranked, near-duplicates left alone, parent chunks never deduped, and dedupe keys stripped from the output. Rapid property: the merged items never contain both a live local note and a parent item it matches.
- [ ] 7.2 RED: #744. Explore picks never displace direct items under `--limit`, and the note floor is re-applied over the merged direct set. Use the issue's reproduction shape (10 phrases, explore scores 0.75–0.89) as a fixture.
- [ ] 7.3 RED: #743. The merged `budget` reports the applied `limit`, `content_budget`, `lazy_chunks`, `chunks_snippeted` and the summed `explore_allocated`.
- [ ] 7.4 GREEN: rework `mergeQueryPayloads`/`runMergedQuery`: request `dedupe-keys=1`, build the local link/hash index from all live local notes, and dedupe → order → cap → content-budget. (E37)
- [ ] 7.5 Close #745 with design D9's reasoning, and close #746 as superseded by this change, citing D8. Do this after the implementation merges, not before.

## 8. Curation fold (design D10)

- [ ] 8.1 RED→GREEN: `engram amend --target O --discard --into E` removes O and its sidecar, adds O's basename to `E.aliases` without duplicates, moves O's `parent` link when E has none, and does not re-embed E or queue an offer. A bare `--discard` is unchanged. `--into` without `--discard` is an error. (E31)

## 9. Skill edits (design D10). Each uses `superpowers:writing-skills` TDD: a RED baseline pressure test, the edit, a GREEN re-test, then REFACTOR

- [ ] 9.1 curate. RED scenario: a vault holding a pulled-down pending offer (`parent.via: pulled`) covered by a local note, and a served amend-offer carrying `offer.for`. Expect the baseline to use a bare `--discard` (losing the link) and to ignore `offer.for`. Edit per E89–E92. GREEN: the agent judges `offer.for` first and ends covered/near with `--discard --into <existing>`.
- [ ] 9.2 recall. RED scenario: a merged payload in which the agent used a `from_parent` note. Expect the baseline either to skip activation or to try `engram amend` on it. Edit per E93–E95. GREEN: the agent activates it (pull-down) and does not amend it.
- [ ] 9.3 learn. RED scenario: a learn under `ENGRAM_PARENT` that prints "parent unreachable; 1 offer(s) queued". Expect the baseline to treat it as a failure, retry, or suggest `ENGRAM_SERVER`. Edit per E96. GREEN: the agent treats the write as done.
- [ ] 9.4 Deploy with `engram update` and verify the deployed copies are byte-identical to the sources. Do not hand-edit the registered vault mirrors (E97). Their refresh offers come from registration.

## 10. Docs (enumeration sections D and E)

- [ ] 10.1 Perform E74–E80 (README, GLOSSARY).
- [ ] 10.2 Perform E81 (GLOSSARY entries) and E82 (ADR-0029, plus the ADR-0027 pointer).
- [ ] 10.3 Perform E83–E86 (C1/C2/C3).
- [ ] 10.4 Perform E87 (ROADMAP) and E88 (LEDGER row, after group 11).
- [ ] 10.5 Perform the E61, E64, E68, E70 Purpose rewrites at archive time.
- [ ] 10.6 E60 sweep: `grep -rn ENGRAM_SERVER --exclude-dir=.git .` hits only the allowlist in E60.

## 11. Real-binary verification (never the real vault)

- [ ] 11.1 Set up the scratch environment:
  - `S=$(mktemp -d)`.
  - `go build -o $S/engram ./cmd/engram`.
  - Parent: run `XDG_DATA_HOME=$S/parent-xdg $S/engram serve --addr 127.0.0.1:<free port> --vault $S/parent-vault` with `ENGRAM_PARENT` and `ENGRAM_SERVER` unset.
  - Child: every command runs with `XDG_DATA_HOME=$S/child-xdg ENGRAM_PARENT=http://127.0.0.1:<port>` and `--vault $S/child-vault`, from a non-repo cwd (`cd $S`).
  - Confirm `~/.local/share/engram/vault` is unchanged afterwards: `git -C ~/.local/share/engram/vault status --porcelain` is empty, and HEAD is the same before and after.
- [ ] 11.2 Check first use and the hard error:
  - the first child `query` creates `$S/child-vault` and prints the one creation line;
  - `ENGRAM_SERVER=http://127.0.0.1:<port> $S/engram query --phrase x` exits non-zero, names `ENGRAM_PARENT`, and the server log shows no request.
- [ ] 11.3 Offline and drain:
  - stop serve, run a child `learn fact` (exit 0, note live locally, warning, outbox has 1 entry), then amend it (still 1 entry);
  - start serve, run a child `query`: the outbox is empty, the parent has exactly one pending note carrying the amended content, and the child note carries `parent.note` equal to the receipt basename.
- [ ] 11.4 Curation and dedupe:
  - on the parent, `amend --clear-pending` the offer (absent);
  - a child merged `query` shows the note once, with `from_parent: false`;
  - repeat with a second offer folded on the parent via `--discard --into <existing>`, and confirm the child still sees one copy (the alias path).
- [ ] 11.5 Pull-down:
  - learn a note directly on the parent;
  - the child `activate --note <parent basename>.md` creates a pending local copy with `parent.via: pulled`, and the next child query shows `pending_offers: true` with the parent item still visible;
  - running `activate` again writes nothing;
  - `--clear-pending` on the child sends no offer (the parent's pending count is unchanged);
  - change the note on the parent, activate again, and a second pending offer appears.
- [ ] 11.6 Record the outcomes, with the exact commands, in the LEDGER row (E88). Any empty or missing field that the design says is populated means the task is not done (Joe's "passing tests ≠ usable system").

## 12. Close-out

- [ ] 12.1 `targ check-full` is clean. Run the requirement-collision sweep (vault note 744) again across all active changes, and read every sibling's tasks.md (vault note 757) before archive.
- [ ] 12.2 Run a fresh-context implementation review (with argumentation) before merge, per the repo's worktree rules. Then rebase on main, re-test, and merge with `git merge --ff-only`.
- [ ] 12.3 Archive with `/opsx:archive`. Before archiving, diff every MODIFIED block's scenario headers against the live spec (vault notes 651/756), and do the E61/E64/E68/E70 Purpose rewrites.
