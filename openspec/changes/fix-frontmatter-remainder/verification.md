# Verification

## Task 0: parity goldens from `0f5d91b7`, before any code change (2026-10-01)

The tree's code was at `0f5d91b7` (only the proposal was committed on top).

**0.1 Backfill goldens.** A temporary generator test, run under `targ test` with an env var and then deleted, ran `backfillIdentity` in-process on the 14 inputs of `backfillParityCases()` (fact and feedback; minimal with an unquoted `created:`, `project:`, every optional modeled key, exchange fields, explicitly empty identity, no detectable repo, vocab tags) and wrote `internal/cli/testdata/backfill_identity_parity/*.md`.

**0.2 Amend goldens.** The same generator re-ran amend's 32 parity cases (`amendParityCases()`) with the code at `0f5d91b7`: all 32 outputs were byte-identical to the committed `testdata/amend_hash_parity/` goldens (written at `49cfc120`). Report line: `amend identical: 32 differ: 0`.

**0.3 Pre-change binary.** A binary built from `0f5d91b7` (`git worktree add --detach` into the scratchpad, `go build -o $S/bin/engram-pre ./cmd/engram`, no `go install`), run under `env -i PATH=/usr/bin:/bin HOME=$S/home XDG_DATA_HOME=$S/data` from a scratch git repo whose origin is `github.com/acme/widgets` (the parity test's detected repo), with `user.email = bob@example.com` in the scratch `$HOME/.gitconfig`:

- **Amend:** each of the 26 cases that need no chunk index was written to its own scratch vault and amended with the matching flags (`--clear-pending` cases passed the `--expect-hash` that `engram show` printed). **26/26 notes byte-identical to the goldens** (not only the exchange hash). The 6 `--chunk-source` cases are pinned in-process only, as before.
- **Backfill:** not reachable from the binary in isolation. `engram update --backfill-identity` runs the backfill only after the full self-update (`resolveSource` clones the remote into `$TMPDIR/engram-update-clone`, builds and installs a binary, and re-execs it), so the backfill would run in a freshly built *remote* binary, not the `0f5d91b7` one. The first attempt stopped at the clone's git-lfs check, before any vault write; the stray `/tmp/engram-update-clone` it created was removed. Backfill parity is pinned in-process against goldens written by the `0f5d91b7` code (0.1), as the archived change pinned its receipt cases.

## Task 5.1: real-binary check on a scratch vault (2026-10-01)

A scratch build of `f7eac26e` (`go build -o $S/bin/engram ./cmd/engram`, no `go install`) under `env -i PATH=/usr/bin:/bin HOME=$S/home XDG_DATA_HOME=$S/data`, every command with `--vault` (or `ENGRAM_VAULT_PATH`) inside `$S`, from cwds with no git repository above them (`$S/plain`, and for `engram update` a plain copy of the source tree, `$S/src`, with no `.git`). `$S` is under the session scratchpad. The real vault and `~/.claude` were never referenced.

### Defect 1: amend converts CRLF

Two fact notes were learned, given `luhmann_old: "12"`, and converted by hand to all-CRLF (17 CR lines each). `engram embed status` then reported `stale: 2`.

| # | Command | Result |
|---|---|---|
| A1 | `engram amend --target 1.2026-10-01.crlf-content --object "an LF note"` | exit 0; 0 CR lines; `object: an LF note`; `luhmann_old: "12"` kept. **PASS** |
| A2 | `engram amend --target 2.2026-10-01.crlf-supersedes --supersedes "1.2026-10-01.crlf-content\|narrows\|old claim"` (an amend kind that does not otherwise re-embed) | exit 0; 0 CR lines; `supersedes:` added; `luhmann_old: "12"` kept. **PASS** |
| A3 | `engram embed status` | `with-embeddings: 2`, `stale: 0` (was `stale: 2`): both converted notes were re-embedded. **PASS** |

### Defect 2: identity backfill keeps unmodeled keys and refuses an anchored key

`engram update --backfill-identity` runs the backfill only after its self-update, so it was run from the scratch source copy with `GOBIN=$S/gobin` (and the machine's existing `GOMODCACHE`/`GOCACHE`, read-only use). Update installed into `$S/gobin/engram` and re-ran itself from there; nothing was installed outside `$S`. The vault held a fact note missing identity, with `luhmann_old: "12"` and a nested `provenance:` map, and a feedback note whose `user: &u ""` is aliased by `x_user: *u`.

| # | Check | Result |
|---|---|---|
| B1 | Exit status and message | exit 1: `update: backfill-identity: refused 2.2026-01-01.anchored-user.md: frontmatter key the edit replaces carries a YAML anchor: user`. **PASS** |
| B2 | The other note is still stamped | `user: joe`, `vault: personal` added (no `repo:`: no git repo and no `project:`); `luhmann_old: "12"` and the whole `provenance` map intact; `created:` re-emitted quoted, as before. **PASS** |
| B3 | The refused note is untouched | sha256 `1dd5d7d7…f5325424` before and after. **PASS** |

### Defect 3: a refused receipt needs attention

A scratch parent ran `engram serve --addr 127.0.0.1:18789 --vault $S/pvault` (its own scratch `HOME` and `XDG_DATA_HOME`). In a child vault, a fact note was learned with no parent configured, given `parent: &p {vault: abab…}` aliased by `parent_copy: *p`, and then amended with `ENGRAM_PARENT` set, which stamps the xid, queues the offer and drains. The same script was run against the pre-change binary (`0f5d91b7`) on a second child vault for contrast.

| # | Step | New binary | Pre-change binary |
|---|---|---|---|
| C1 | `engram amend --object "refused once"` | exit 0; one warning: `engram: warning: outbox: the parent accepted the offer of 1.2026-10-01.receipt-new, but the note refuses its receipt (offer receipt: frontmatter key the edit replaces carries a YAML anchor: parent); the offer will not be re-sent, …`; outbox entry `state: attention`, `attempts: 1`, receipt and `sent_hash` kept | exit 0; warning `recording the receipt … YAML anchor: parent`; entry `state: queued`, `attempts: 1` |
| C2 | `engram query` (drains after the parent `/query` succeeds) | no stderr; `attention`, `attempts: 1` (not sent) | the same warning again; `queued`, `attempts: 2` (re-sent) |
| C3 | `engram query` again | no stderr; `attention`, `attempts: 1` | the same warning again; `queued`, `attempts: 3` (re-sent) |
| C4 | The child note | byte-identical across both drains (sha256 `981c97c5…`) | byte-identical |
| C5 | `engram update --dry-run` (from `$S/src`, so no install) | `parent outbox: 0 offer(s) queued, 0 rejected, 1 need attention, oldest queued 17s ago` and `needs attention: 1.2026-10-01.receipt-new: offer receipt: … YAML anchor: parent` | — |
| C6 | Anchor and alias removed, then `engram query` | no stderr; outbox empty; the note's `parent:` now holds the parent's vault ID and `{note: 1.2026-10-01.receipt-new, via: offered, hash: xh1:3669…}`; the parent vault still holds one note for this child (no new offer) | — |

**PASS** on every row: the new binary warns once, never re-sends, reports the entry, and records the kept receipt by itself once the note is fixed; the pre-change binary re-sends and warns on every drain.

The parent server was stopped, and the temporary `0f5d91b7` worktree used for the pre-change binary was removed (`git worktree remove`).

## Follow-up: identity backfill converts CRLF (2026-10-02)

Same setup as task 5.1: a fresh scratch build (`go build`, no `go install`), `env -i` with scratch `HOME`/`XDG_DATA_HOME`, `ENGRAM_VAULT_PATH` in scratch, and `engram update` run from a plain copy of the source tree with `GOBIN` in scratch. Two fact notes were learned. Note A had `repo:`/`user:`/`vault:` stripped, `luhmann_old: "12"` added, and was converted to all-CRLF (15 CR lines). Note B kept its identity and was converted to all-CRLF (16 CR lines). `engram embed status`: `stale: 2`.

| # | Check | Result |
|---|---|---|
| D1 | `engram update --dry-run` | the notice `notes missing repo:/user:/vault: provenance found — run engram update --backfill-identity …` appears: the CRLF note is now detected. **PASS** |
| D2 | `engram update --backfill-identity` | exit 0; `stamped 1 note(s)`. **PASS** |
| D3 | Note A | 0 CR lines; `user: joe`, `vault: personal` added; `luhmann_old: "12"` kept. **PASS** |
| D4 | Note B (already stamped, so not written) | still 16 CR lines; sha256 `4453e6b0a9997232…` unchanged. **PASS** |
| D5 | `engram embed status` | `stale: 1` (was 2): A's sidecar was rebuilt; the remaining stale sidecar is B's, from the hand conversion, and backfill does not write B. **PASS** |

`TestBackfillIdentity_ParityWithPreChangeBackfill` still reproduces all 14 goldens byte for byte.

## Follow-ups D4–D6: real-binary check (2026-10-02)

A scratch build of the working tree (`go build`, no `go install`), every command under `env -i PATH=/usr/bin:/bin HOME=$S/home XDG_DATA_HOME=$S/data` from `$S/plain` (no git repository above it), each vault inside `$S`. A scratch parent ran `engram serve --addr 127.0.0.1:18790 --vault $S/pvault` with its own scratch `HOME` and `XDG_DATA_HOME`.

| # | Check | Result |
|---|---|---|
| E1 | A hand-written pending served offer (`xid`, `offer.origin`), converted to all-CRLF: `engram show 5.2026-09-28.offer` | line 1: `# exchange_hash: xh1:284b1015…19b`. **PASS** |
| E2 | `engram query --phrase …` on that vault | the CRLF pending offer is not listed (0 hits), and the payload says `pending_offers: true`: it is still a pending offer. **PASS** |
| E3 | `engram amend --target 5.2026-09-28.offer --clear-pending --expect-hash <E1's hash>` | exit 0; the note is written as LF (0 CR lines) and is no longer pending. **PASS** |
| E4 | A note learned while the parent was unreachable (entry `queued`), then converted to all-CRLF (17 CR lines); backoff cleared; `engram query` with `ENGRAM_PARENT` set | exit 0; outbox empty; the note now carries `via: offered` and has 0 CR lines; `engram embed status`: `stale: 0`. The parent vault holds the one offer. **PASS** |
| E5 | A parent note converted to all-CRLF; in an empty child vault, `engram activate --note <parent basename>` | exit 0; one pulled copy, `pending: true`, `via: pulled`, 0 CR lines. **PASS** |
| S1 | A learned note given `supersedes:` entries A (with `x_reason: kept`) and B; `engram amend --supersedes "9.2026-01-01.a\|refutes\|new a" --supersedes "9.2026-01-01.c\|narrows\|new c"` | exit 0; A is `type: refutes`, `claim: new a`, `x_reason: kept`; B is gone; C has only `note`, `type`, `claim`. **PASS** |
| U1 | `engram learn` with no git `user.email` (a scratch `HOME` without `.gitconfig`), from a CGO-free scratch build under `env -i` (no `USER`) | `user: joe`: on macOS the OS username lookup still resolves, so empty detection is not reachable from the real binary on this machine, as the archived change recorded for amend. The empty case is pinned by `TestLearn_OmitsEmptyUser` and, through the production wiring with git and the OS lookup both failing, `TestActivate_PullDownOmitsEmptyUser`. **Recorded, not a failure.** |

The parent server was stopped afterwards and `$S` removed.

Parity after D4–D6: `TestRunAmend_ExchangeHashParityWithPreChangeAmend` matches all 32 exchange hashes and 29 of 32 notes byte for byte; the other 3 (`{fact,feedback,runbook}-minimal-clear-pending`) match byte for byte once the golden's `user: ""` line is removed, the one intended difference (design D6). `TestBackfillIdentity_ParityWithPreChangeBackfill` still matches all 14 goldens.

## Ruling W2: real-binary check (2026-10-02)

Same setup: a scratch build (`go build`, no `go install`), `env -i PATH=/usr/bin:/bin HOME=$S/home XDG_DATA_HOME=$S/data`, cwd `$S/plain`, every vault in `$S`. Two notes were learned and converted to all-CRLF: A (`project: widgets`, 17 CR lines) and B (situation edited to `""`, 16 CR lines).

| # | Check | Result |
|---|---|---|
| R1 | `engram embed status` | `with-embeddings: 1`, `stale: 1`: A's sidecar is fresh although A is now CRLF (its content hash equals its LF form's); the stale one is B, whose situation was edited. **PASS** |
| R2 | `engram check` | `FAIL M5 situation-presence: 1 note(s) missing a situation — 2.2026-10-02.crlf-empty-situation`: the CRLF note's frontmatter is read (before, it was skipped as having none). **PASS** |
| R3 | `engram count --group-by type` | `fact 3`: both CRLF notes are counted. **PASS** |
| R4 | `engram query --phrase … --project widgets` | returns `1.2026-10-02.crlf-reader` (the CRLF note matches the project filter). **PASS** |
| R5 | CR lines after R1–R4 | A 17, B 16: no reader wrote a note. **PASS** |
| V1 | A hand-written pending note with no `user:`/`vault:`; `engram amend --clear-pending` with no vault name configured | exit 0; `vault: personal` written; no `user:` key; `pending:` gone. **PASS** |
| N1 | Offering with no detectable user | Not reachable from the real binary on this machine (macOS resolves the OS username, as recorded for U1). Pinned through the production wiring by `TestUpdateExchange_NoUserIdentityWaitsThenOffers`: learn omits `user:`, the offer is not sent and the warning says `cannot offer`, update reports `1 need attention` with the reason, and once detection works the next exchange offers it. **Recorded.** |

Parity after W2: amend 32/32 exchange hashes; 29/32 byte-identical and the 3 `minimal-clear-pending` notes byte-identical after the golden's `user: ""` and `vault: ""` lines are replaced by `vault: personal` (rulings W1, W2; design D10). Backfill 14/14 and fold/receipt 11/11 byte-identical.
