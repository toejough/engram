# Enumeration: local-first-parent-sync

This change alters five invariants:

- **Remote mode.** There is no `ENGRAM_SERVER` thin client; setting it is a hard error for every command.
- **Where writes land.** Always locally first, then offered to the parent when `ENGRAM_PARENT` is set.
- **What a merged query shows.** One copy of each note (the local one), no parent chunks, and pull-down on activate.
- **What the served API exposes.** `query`, `show`, `activate`, `learn`.
- **Which amends re-stamp identity.** Content amends only.

Every row below that names any of the following must be rewritten, deleted, or explicitly kept:

- `ENGRAM_SERVER` or the thin client;
- served `amend`, `query-chunks`, or `show-chunk`;
- `show-chunk --parent`;
- the `{status, luhmann}` receipt;
- runbook-pending-needs-`skill_hash`;
- amend re-stamping on every write;
- `resituate`'s field loss;
- read-only-parent wording.

**Search (revised 2026-09-27, round 2, worktree `runbook-vs-skill`, branch `local-first-parent-sync`).**

1. `grep -rln ENGRAM_SERVER --exclude-dir=.git .` gives **20 files outside the archive**:
   - `.review/events.jsonl`
   - `cmd/engram/serve_integration_test.go`
   - `cmd/engram/serve.go`
   - `dev/eval/audit/results/transcript-events.jsonl`
   - `docs/GLOSSARY.md`
   - `internal/cli/{deps.go, learn.go, merged_query_dispatch_test.go, primitives.go, register_skills_cli_test.go, serve_client_test.go, serve_client.go, show_chunk.go, show.go, targets_test.go, targets.go}`
   - `openspec/specs/{recall-payload-cuts, vault-merged-recall, vault-serve-api}/spec.md`
   - `README.md`

   Round 1 said 19. That was a miscount of the same list. The archive holds a further 15 files / 54 lines.

2. The same search for these related identifiers, across `internal/`, `cmd/`, `docs/`, `openspec/specs/`, `agent-instructions/`, `README.md` and `CLAUDE.md`:
   - `serverBase`, `fetchQuery`, `fetchQueryChunks`, `fetchAmend`, `fetchShowChunk`, `dispatchShowChunk`
   - `serveAmend`, `serveQueryChunks`, `serveShowChunk`, `/amend`, `/query-chunks`, `/show-chunk`
   - `errDiscardOverServer`, `errRegisterSkillsOverServer`, `ExportServerBase`, `noteHasPendingMarker`
   - `RenameAndRewriteReferences`, `adoptRenderInput`, `rerenderFact`, `rerenderFeedback`
   - `served write`, `two doors`, `re-stamp`

3. Every row was read at the cited line before its disposition was written (vault note 1072).

The `CLAUDE.md` and `agent-instructions/` search returned **no** `ENGRAM_SERVER` hits. The skill rows come from the served/offer wording.

**Disposition key:**

- *delete*: remove the code or text.
- *rewrite*: replace it.
- *append*: add without editing history.
- *code*: a Go change in a TDD task.
- *archive-time*: a Purpose edit, which deltas cannot make.
- *no change*: the reason is given in the row.

**Verification rule for performing rows.** After each edit, grep for the new text (it must be present) and for the old text (it must be absent) before ticking the row. Row 72 runs **after archive** (task 12.4).

## A. Go production code

| # | Location (verified) | Current | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 1 | internal/cli/serve_client.go:20 | `envServerBase = "ENGRAM_SERVER"` | code | Keep it only for the D1 guard, renamed `envRemovedServer`. |
| 2 | internal/cli/serve_client.go:397-406 | `serverBase(deps)` | delete | Replaced by the pre-dispatch guard (D1). The guard also covers `serve`. |
| 3 | internal/cli/targets.go:146-156 | amend: `serverBase` branch, `errDiscardOverServer`, `fetchAmend` | delete | Amend is always local. Content amends enqueue offers (D5/D6). |
| 4 | internal/cli/targets.go:214-218 | query: `fetchQuery` branch | delete | The merge is the only remote path. |
| 5 | internal/cli/targets.go:244-248 | query-chunks: `fetchQueryChunks` branch | delete | Local only. The `query-chunks` CLI command itself stays. |
| 6 | internal/cli/targets.go:269-273, 280-284, 291-295 | learn ×3: `fetchLearn` branches | delete | Always local, then enqueue (D6). |
| 7 | internal/cli/targets.go:371-375 + :124-129 | register-skills refusal and `errRegisterSkillsOverServer` | delete | The D1 guard covers it. |
| 8 | internal/cli/targets.go:117-123 | `errDiscardOverServer` + comment | delete | — |
| 9 | internal/cli/targets.go:356-357 | comment naming both refusals | rewrite | Drop the clause. |
| 10 | internal/cli/targets.go:446-450 | activate: `fetchActivate` branch | code | Local-first resolution (`.md` exists), pull-down fallback, `--parent`, and no bare-ID parent lookup (D8). |
| 11 | internal/cli/targets.go:478-482 | show: server branch | delete | `--parent` and the fallback remain. |
| 12 | internal/cli/targets.go:500-517 | show-chunk: server branch **and** the `--parent` branch | delete | Q1: there is no parent chunk path. |
| 13 | internal/cli/show_chunk.go:16-20 | `ShowChunkArgs.Parent` flag | delete | Q1. |
| 14 | internal/cli/serve_client.go:119 `dispatchShowChunk`, :290 `fetchShowChunk`, :295 `fetchShowChunkFallback` | parent chunk fallback | delete | Q1. `show-chunk` resolves locally only. |
| 15 | internal/cli/targets.go:428-431 (serve desc) | "Serve query/query-chunks/show/show-chunk/activate/learn/amend over HTTP … learn/amend land as …" | code | "Serve query/show/activate/learn … learn lands as a pending offer" (D7). |
| 16 | internal/cli/pulldown.go:208 `(*pullSession).signalUse` (corrected from serve_client.go:158-167 `fetchActivate`, which this replaces) | thin-client activate | code | Repurposed as the best-effort parent `/activate` after pull-down (Q2, D8). |
| 17 | internal/cli/serve_client.go:169-184 `fetchAmend` | — | delete | Amend-offers use `/learn`. |
| 18 | internal/cli/outbox.go `newOfferSender` (corrected from serve_client.go:206-225 `fetchLearn`, which this replaces) | "through ENGRAM_SERVER" | code | Becomes the offer sender. It takes the payload built at send time and returns the parsed receipt (D6). |
| 19 | internal/cli/serve_client.go:227-243 `fetchQuery`, `fetchQueryChunks` | — | delete | No callers remain. |
| 20 | internal/cli/serve_client.go:42-61 `buildQueryParams` (doc :43) | shared with `fetchQuery` | code | Parent fetch only. Adds `dedupe-keys=1` (D7). |
| 21 | internal/cli/serve_client.go:93-96, 266, 284-289, 335-337 | comments naming `ENGRAM_SERVER` | rewrite | Say "parent". |
| 22 | internal/cli/outbox.go `newOfferSender`/`classifyOfferResponse` (corrected from serve_client.go:365-383 `printOfferReceipt`, which this replaces) | `{status, luhmann}` | code | Parse `{status, luhmann, basename, pending, vault_id, for}`. A missing `vault_id` means "parent too old" (D6). |
| 23 | internal/cli/serve.go:117, 418 (`/query-chunks`, `serveQueryChunks`) | served chunk query | delete | Q4. |
| 24 | internal/cli/serve.go:119, 455 (`/show-chunk`, `serveShowChunk`) | served chunk show | delete | Q1 (the parent chunk path). |
| 25 | internal/cli/serve.go:122, 292-336 (`/amend`, `serveAmend`) | served amend | delete | D7. |
| 26 | internal/cli/serve.go:158-165 `offerReceipt` | `{Status, Luhmann}` | code | Add `Basename`, `Pending`, `VaultID`, `For`. |
| 27 | internal/cli/serve.go:349-384 `serveLearn` | honors target/position; always a new note | code | Top-level placement, in-place update by `offer.origin`, `offer.key` no-op, `offer.for` resolution (live → alias → pending), `xid` stamp (D7). |
| 28 | internal/cli/serve.go:438-450 `serveShow` | errors → 500; rendered | code | `raw=1` JSON envelope with alias resolution, 404 for not-found (D7). |
| 29 | internal/cli/serve.go:390-414 `serveQuery` | no dedupe keys | code | `dedupe-keys=1`: `vault_id`, `exchange_hash`, `aliases` (D7). |
| 30 | cmd/engram/serve.go (serve startup) + `ensureVault` | no vault ID | code | Stamp `.engram-vault-id` on serve start (D2). |
| 31 | internal/cli/offer.go:89-114 `noteHasPendingMarker` | runbook pending only with `skill_hash` | code | Any type counts (G1). The real vault was re-checked read-only: zero pending notes. |
| 32 | internal/cli/offer.go:150 | "served learn/amend write is awaiting curation" | rewrite | "an offered or pulled-down note is awaiting curation". |
| 33 | internal/cli/learn.go:86-98 | skill fields `json:"-"` | code | Add `Parent`, `Aliases`, `Xid` as `json:"-"`. `Offer{Origin,Key,For}` is the only new wire field (D7). |
| 34 | internal/cli/learn.go:274, 316, 395 | fact/feedback/runbook frontmatter structs | code | Add `xid`, `parent` (nested: `vault`, `links[]`, `author`), `aliases`, `offer` (D4; review M13). |
| 35 | internal/cli/learn.go:573-575 | `learnArgsFrom*` doc: "ENGRAM_SERVER-mode (fetchLearn) dispatch" | rewrite | "and the offer payload builder". |
| 36 | internal/cli/learn.go (RunLearn ~:204) | local write only | code | Stamp `xid`, then enqueue under the lock, then drain (D6). |
| 37 | internal/cli/amend.go:170-181 | identity re-stamp on every call | code | Re-stamp only on content, `--supersedes` and `--chunk-source` amends (D10, G11). |
| 38 | internal/cli/amend.go:22-77, 156, 460-474 | `--discard` only | code | Add `--into` (alias union, link merge). A bare `--discard` of a pulled note records a decline. Content amends enqueue. `--clear-pending` of a served offer enqueues upward (M14). Survival of the exchange fields. |
| 39 | internal/cli/amend.go:53 | "never touches the served /amend path" | rewrite | Drop it. |
| 40 | internal/cli/activate.go:13-16, 38-75 | `{Vault, Notes}`; sidecar-based; errors only when all refs fail | code | `--parent`, `.md`-exists hit, no fetch under the lock, skip re-checked under the write lock, declines, per-ref reporting, non-zero on any failure (D8). |
| 41 | internal/cli/resituate.go:223, 256 | hand-copied field list; drops `pending`/`sources`/`tags`/`supersedes`/`vocab_version` | code | Round-trip that changes only `situation` and the body opener, preserving everything else, including the exchange fields (M7). Enqueue an offer (D5). |
| 42 | internal/cli/identity_backfill.go:61, 90 | typed round-trip | code | Survival of the exchange fields. No offer. |
| 43 | internal/cli/luhmann_reparent.go:82 `RenameAndRewriteReferences` | rename without an alias | code | Append the old basename to `aliases` in the same write (H2). |
| 44 | internal/cli/skillreg_accept.go:300 `adoptRenderInput`, :317 `applySkillNoteBody`, :102, :511 | adopt rename; typed round-trip | code | Append an alias on adopt rename. Survival of the exchange fields. |
| 45 | internal/cli/merged_query.go:100-136, 177-191 | no dedupe; parent chunks kept; flat sort; zero budget; hint ORed | code | Drop parent chunks. Dedupe (D4/D9) with the M4 substitution. Direct-before-explore ordering and the M3 floor (#744). Budget (#743). Local-only hint (H4). Backoff. Drain after success. Self-parent guard. |
| 46 | internal/cli/qa.go:221 + vault_init.go:44 | `ensureVaultDir` from learn/qa only | code | `ensureVault` on every vault-resolving command. Vault ID plus `.engram/` with its self-ignoring `.gitignore`. The root `.gitignore` is untouched (D2). |
| 47 | internal/cli/update.go (vault notices ~:481, 904) | pending notice, etc. | code | Outbox/backoff notice, uncommitted-vault-ID notice, drain (D2/D6). |
| 48 | internal/cli/deps.go:74-80 `Deps.Fetch` doc | "Used by CLI targets when ENGRAM_SERVER is set" | rewrite | "parent requests (ENGRAM_PARENT)". |
| 49 | internal/cli/primitives.go:104-109 | `Primitives.HTTP` doc naming "ENGRAM_SERVER-mode" | rewrite | "parent-sync requests". |
| 50 | internal/cli/show.go:19-23 | `Parent` doc "inert when ENGRAM_SERVER is also set" | rewrite | Drop the clause. |
| 51 | cmd/engram/serve.go:1-2, 29-36 | "ENGRAM_SERVER client mode"; 30s client | code | Say "ENGRAM_PARENT parent-sync client". Add a dialer connect timeout ≤ 3s (M9) as a thin-api primitive. |
| 52 | new: `internal/cli/exchangehash.go`, `outbox.go`, `exchangestate.go`, `pulldown.go` | none | code | D3 (including the key classification table), D6, D2, D8. DI only (ADR-0013, `targ check-thin-api`). |
| 52a | internal/cli/primitives.go + cmd/engram/main.go group functions | no random source | code | Add a `RandRead` primitive (`crypto/rand`) for vault IDs, `xid` and nothing else. Thin-api shape (r3-4). |
| 52b | internal/cli/targets.go (new `vault-id` target) | none | code | `engram vault-id [--regenerate \| --claim]` (D2, r3-4). The ID file is created via `WriteFileExcl` (deps.go:98), then re-read. |
| 52c | internal/cli/show.go (+ serve_client.go:31/435 fallback label order) | no exchange hash output | code | For notes carrying `xid` only, print `# exchange_hash: xh1:…` as the first line. Other notes are byte-identical. On the parent fallback, the `# from_parent: true` label comes first (D10; `vault-offer-curation`; r4 M-B). |
| 52d | internal/cli/amend.go (flags) | no `--expect-hash` | code | `--expect-hash`, required for bookkeeping on notes carrying `offer.origin` (D10, r3-2). |

## B. Go tests

| # | Location (verified) | Current | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 53 | internal/cli/serve_client_test.go:27-448 (`TestEngramServer_*`, 17 tests) | thin-client routing | delete | Replaced by row 54 and the exchange tests. |
| 54 | new test | — | code | `TestEngramServerSet_HardErrorsEveryCommand`: every subcommand including `serve` makes no FS, lock, Fetch or bind call, and names `ENGRAM_PARENT`. |
| 55 | internal/cli/serve_client_test.go:733, 827, 998 | `TestServerBase_NilGetenv`, `TestShowChunkParent_InertWhenEngramServerSet`, `TestShowParent_InertWhenEngramServerSet` | delete | — |
| 56 | internal/cli/serve_client_test.go:743, 780, 803, 850, 875 | `TestShowChunkFallback_*` ×3, `TestShowChunkParent_RoutesThroughFetch`, `TestShowChunkParent_WithoutEngramParentErrors` | code | Delete. Replace with a test that `show-chunk` never contacts the parent and that `--parent` is an unknown flag (Q1). Keep `TestShowChunkTarget_LocalDispatch` (:903). |
| 57 | internal/cli/serve_client_test.go comments :25, 286, 401, 662-664, 896, 970, 1019, 1089, 1116 | mention `ENGRAM_SERVER` | rewrite | Say "parent". |
| 58 | internal/cli/merged_query_dispatch_test.go:19-44 | precedence test | delete | — |
| 59 | internal/cli/register_skills_cli_test.go:269-284 | refusal test | delete | — |
| 60 | internal/cli/targets_test.go:552 | comment | rewrite | — |
| 61 | internal/cli/export_test.go:164 `ExportServerBase` | — | delete | — |
| 62 | cmd/engram/serve_integration_test.go:69 | comment | rewrite | "engram serve / ENGRAM_PARENT". |
| 63 | internal/cli/query_integration_test.go:122-138; register_skills_cli_test.go:254 | strips only `ENGRAM_PARENT` | code | Also strip `ENGRAM_SERVER`. |
| 64 | internal/cli/serve_test.go:56, 101-180, 528 | `/amend` route and its tests | code | Delete them. The route set becomes exactly four. |
| 65 | internal/cli/serve_test.go:398 `TestServeQueryChunks_EmptyIndexSucceeds`, :535 `TestServeShowChunk_NotFoundReturnsError` | chunk routes | delete | Q1/Q4. |
| 66 | internal/cli/serve_test.go:307 `TestServeLearn_IgnoresSkillIdentityFields` | skill fields blocked | code | Extend it with `parent`/`aliases`/`xid` keys, both spellings. Add tests for the receipt, in-place update, key no-op, `for` resolution, placement, raw envelope, and dedupe keys. |
| 67 | internal/cli/offer_test.go:136-147 | runbook without `skill_hash` is not pending | code | Invert it (G1). |
| 68 | internal/cli/serve_client_test.go:608, 626 | local writes never pending | no change | Still true. |
| 69 | internal/cli/skillfields_survival_test.go | skill-field survival per site | code | Add exchange-field survival siblings at every site in `vault-note-identity`. |
| 70 | internal/cli/resituate_test.go | expects the current field set | code | Add assertions that every untouched field is preserved (M7). |
| 71 | internal/cli/amend_test.go, internal/cli/identity_test.go | restamp-on-every-write expectations | code | Update them. Bookkeeping amends preserve identity (D10). |

## C. Final sweep (after archive)

| # | Location | Current | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 72 | whole repo | 20 files with `ENGRAM_SERVER` | code | Run task 12.4 **after** archive: `grep -rln ENGRAM_SERVER --exclude-dir=.git .`. The only files allowed to hit are: `internal/cli/serve_client.go` (guard constant); the guard test file; `openspec/specs/vault-local-first/spec.md`; `openspec/changes/archive/**`; `openspec/specs/vault-serve-api/spec.md` and `openspec/specs/recall-payload-cuts/spec.md` (their synced text names the removed mode); `docs/architecture/adr.md` (ADR-0029); `README.md` (the migration sentence only, row 88); `docs/GLOSSARY.md` (the `ENGRAM_SERVER` removed-mode entry, row 92); `agent-instructions/skills/learn/SKILL.md` (the "never set `ENGRAM_SERVER`" line, row 103); `dev/eval/LEDGER.md` (the new row, row 102); `.review/events.jsonl`; `dev/eval/audit/results/transcript-events.jsonl`. Any other hit fails the task. |

## D. Specs (`openspec/specs/`)

| # | Location (verified) | Current | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 73 | vault-serve-api/spec.md:3 (Purpose) | "Lets remote environments with no local checkout or binary path use the vault over HTTP…" | archive-time | "Lets child environments, each with its own local vault, exchange notes with this parent over HTTP (offers up, notes-only reads and raw pull-down), reusing the CLI's code paths and locks…" |
| 74 | vault-serve-api/spec.md:30-35 | thin-client requirement | delete (REMOVED in delta) | — |
| 75 | vault-serve-api/spec.md:5, 23, 37, 57 | served set incl. `amend`/`query-chunks`/`show-chunk`; learn/amend wording | rewrite (MODIFIED) | — |
| 76 | vault-merged-recall/spec.md:3-6 (Purpose) | "Lets a node with its own local vault also merge…" | archive-time | "…merge its parent's notes (never chunks), deduped to the local copy, as the standard path for any environment with a parent". |
| 77 | vault-merged-recall/spec.md:20, 37, 123, 135 | `ENGRAM_SERVER` clauses; parent chunks in budgets; show-chunk parent routing; no backoff | rewrite (MODIFIED) | — |
| 78 | vault-merged-recall/spec.md:167, 185 | precedence; server-exclusive | delete (REMOVED); :185 is re-ADDED as local-only | — |
| 79 | recall-payload-cuts/spec.md:40-41 | "local, `ENGRAM_SERVER`-exclusive, and `ENGRAM_PARENT`-merged" | rewrite (MODIFIED) | — |
| 80 | vault-offer-curation/spec.md:3 (Purpose) | "Gives served writes a curated acceptance step…" | archive-time | "Gives offered notes, whether arriving from a child over the API or pulled down from the parent, a curated acceptance step…". |
| 81 | vault-offer-curation/spec.md:7, 14, 28, 55, 69 (scenario "Merged query carries the hint" :92), 112 | served learn/amend; activate; curation actions; receipt; hint ORed across sources; runbook scope | rewrite (MODIFIED) | The hint is local-only (H4). |
| 82 | vault-note-identity/spec.md:3 (Purpose) | "…(filtering, attribution, future multi-vault exchange)" | archive-time | "…parent exchange: `xid`, vault-qualified `parent` links, `aliases`". |
| 83 | vault-note-identity/spec.md:48 "Amend re-stamps identity fields on every write" | every amend re-stamps | rewrite (MODIFIED) | Content amends only (D10). |
| 84 | guidance-runbook-follow-frame/spec.md:183-187 | parent runbook via `show` fallback | no change | The `show` fallback is kept. Only `show-chunk`'s is removed. |
| 85 | runbook-lexical-triggers/spec.md:25-34 "Served round-trip" | a client builds a served query | no change | The merged query still sends `--text` to the parent's `/query`. |
| 86 | recall-glance-deep-dial/spec.md:9-15 | glance activates used notes but creates no notes via learn/amend | no change | A glance activation of a `from_parent` note pulls down a *pending* copy through `activate`. That is not learn or amend, and not live knowledge (design D10). The recall skill text says so (row 101). |
| 87 | learn-rate-skill-only (active change) tasks 3.2-3.4 | re-measure of learn firing | no change | The collision sweep found no shared capability or header. Row 103's learn edit adds offer wording and no firing cues. If task 3.2 runs after it, record the date in that change's LEDGER row. |

## E. Docs

| # | Location (verified) | Current (abridged) | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 88 | README.md:111 | "Set `ENGRAM_SERVER=http://host:port` to make the CLI a transparent HTTP client…" | rewrite | "`ENGRAM_SERVER` is no longer supported: setting it is a hard error for every command; set `ENGRAM_PARENT` to the same URL." Keep the identity half (a served `learn` stamps the declared `user:`, and there is no edge auth). This is the only allowed README hit in row 72. |
| 89 | README.md:93 | "On a local miss, with `ENGRAM_SERVER` unset and `--parent` not passed…" | rewrite | Drop "with `ENGRAM_SERVER` unset". |
| 90 | README.md:94 (`engram show-chunk … [--parent]`) | `--parent` resolves against `ENGRAM_PARENT`; fallback | rewrite | Remove `[--parent]` and the fallback sentence: chunks never cross vaults. |
| 91 | README.md:95 (`engram amend …`), :96 (`engram activate …`) | no `--into`; activate local only | rewrite | `amend`: `[--discard [--into <existing>]]`. `activate`: `[--parent]`, where a local miss with `ENGRAM_PARENT` set pulls a parent note down as a local pending offer. |
| 92 | docs/GLOSSARY.md:453 | "On the served path (`ENGRAM_SERVER`), `text` is capped at 2 KB" | rewrite | "On the served path (a parent's `/query`)…". Add an `### ENGRAM_SERVER (removed)` entry giving the migration. |
| 93 | README.md:105-109 (two-doors section) | served subset lists `query-chunks`, `show-chunk`, `amend`; "A served `learn`/`amend` always lands as a pending offer" | rewrite | Subset: `query`, `show`, `activate`, `learn`. "A served `learn` (a child's offer) lands as a pending offer; so does a note a child pulls down from its parent." |
| 94 | README.md:113-115 ("Merged recall: `ENGRAM_PARENT`") | "`ENGRAM_SERVER` is exclusive…"; "…takes full precedence…"; `show`/`show-chunk --parent` | rewrite | Rewrite it as "Parent sync: `ENGRAM_PARENT`". Cover: the local vault and vault ID; learn, content amend and accepted-offer propagation, with the outbox and backoff; notes-only merge with dedupe that keeps local; activate pulls down; chunks never travel; the fleet impact. Keep the model_id, `from_parent` and unreachable-parent sentences. |
| 95 | README.md:117 | "…local, `ENGRAM_SERVER`-exclusive, and `ENGRAM_PARENT`-merged alike" | rewrite | "…local and `ENGRAM_PARENT`-merged alike". |
| 96 | docs/GLOSSARY.md (new entries) | none | append | `### offer` (learn-offer / amend-offer), `### outbox`, `### exchange hash`, `### parent link`, `### pull-down`, `### vault ID`, `### xid`, `### ENGRAM_SERVER (removed)`. |
| 97 | docs/FEATURES.md (whole file, 25 lines) | a pointer to `openspec/specs/` plus a mission rollup | no change | It has no per-capability rows. New capabilities are covered by its pointer to `openspec/specs/`. Verified: `grep -n -i "serve\|parent\|offer\|merge\|show-chunk\|activate"` returns no hits. |
| 98 | docs/architecture/adr.md (after ADR-0028) + :1067 | none; ADR-0027's memory-poisoning bullet says "`serve` accepts external offers" | append | **ADR-0029 — Local-first parent sync** (decision 784a, D1–D12, rejected alternatives). Add a forward pointer at :1067: pulled-down parent content also passes local curation. |
| 99 | docs/architecture/c2-containers.md:41 (C1), :44 (C4 Vault) | "`/curate` (judges `engram serve` pending offers)"; vault = notes + sidecar + `.luhmann.lock` | rewrite | C1: "(judges pending offers, whether served from children or pulled down from the parent)". C4: add `.engram-vault-id` (tracked) and `.engram/` (self-ignored exchange state). |
| 100 | docs/architecture/c3-components.md table (after K13, :112); c1-system-context.md systems table (:46-47) | no parent-sync component; no parent-vault system | append | A K row for serve, serve_client, merged_query, outbox, pulldown and exchangehash. An external system "Parent engram vault (`engram serve`)" with its relation. |
| 101 | docs/ROADMAP.md:100 (#766) | "#766 create the local vault automatically…" | rewrite | Mark it folded into `local-first-parent-sync`. Add a NOW row for this change. |
| 102 | dev/eval/LEDGER.md (new row) | none | append | Unit coverage summary and the group-11 real-binary outcomes. It states that `update`'s drain has no real-binary run. |

## F. Skills (each edit through `superpowers:writing-skills`, hermetic headless arms, task group 9)

| # | Location (verified) | Current | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 103 | agent-instructions/skills/learn/SKILL.md (Step 2 write sites) | no parent wording | append | With `ENGRAM_PARENT` set, writes are offered automatically. "parent unreachable … queued" is not a failure. Never set `ENGRAM_SERVER`. |
| 104 | agent-instructions/skills/curate/SKILL.md:3-8 (description) | "…or engram serve has just accepted a served write…" | rewrite | "…or a child's offer or a pulled-down parent note is pending…". |
| 105 | agent-instructions/skills/curate/SKILL.md:13 | "A served `engram learn`/`engram amend` write lands as a **pending offer**" | rewrite | "An offer (a served `engram learn` from a child, or a parent note pulled down by `engram activate`) lands as a **pending offer**". |
| 106 | agent-instructions/skills/curate/SKILL.md:19-21 | "`engram serve`'s only job on a write is authenticate, stamp identity, persist the pending marker, respond…" | rewrite | "`engram serve`'s only job on an offer is stamp the declared identity, persist the pending marker (or update the same origin's pending offer in place), respond; a pull-down's only job is fetch and persist the pending copy…". Keep host-local and off the request path. |
| 107 | agent-instructions/skills/curate/SKILL.md:63-64 (covered/near) | ends with `engram amend --discard` | rewrite | Read the offer's `# exchange_hash` from `engram show` when judging, and pass it as `--expect-hash` on every bookkeeping step. If the check fails, re-judge. End with `--discard --into <existing>`. Judge `offer.for` first. Discarding a pulled note outright records a decline. Accepting a served offer on a vault that has a parent sends it onward automatically. |
| 108 | agent-instructions/skills/curate/SKILL.md:97, 102 (red flags) | "covered/near both end in `--discard`"; served-request flag | rewrite | "…end in `--discard --into <existing>`". Keep the host-local flag. |
| 109 | agent-instructions/skills/recall/SKILL.md:207-220 (Step 2.7) | activate used notes; local paths | rewrite | Activate the `from_parent` notes you used the same way: this pulls them down as local pending copies. The next payload's `pending_offers` reflects only local offers, so curate them after the user's request. Glance does this too (row 86). |
| 110 | agent-instructions/skills/recall/SKILL.md:330-339 (red flags) | — | append | "You ran `engram amend` on a `from_parent` item → it does not resolve locally; activate it (pull-down) and let curation fold it". |
| 111 | agent-instructions/skills/recall/SKILL.md:108 | "One call; the binary merges ranking server-side." | rewrite | "One call; with `ENGRAM_PARENT` set the binary merges the parent's notes (never chunks) and dedupes, keeping local copies." |
| 112 | vault runbook mirrors `skill-claude-{curate,recall,learn}` | mirror the old SKILL.md | no change (by hand) | Refreshed by registration offers after the post-merge `engram update` (task 12.3). Never hand-edited. |
| 113 | dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskCurate/** | frozen fixtures | no change | Frozen historical fixtures. |

## G. Historical (no change)

| # | Location | Reason |
| --- | --- | --- |
| 114 | openspec/changes/archive/** (15 files, 54 lines naming `ENGRAM_SERVER`) | Archive history. |
| 115 | .review/events.jsonl:26, 28; dev/eval/audit/results/transcript-events.jsonl:44, 61, 324, 516 | Captured events. |
| 116 | docs/research/2026-08-30-memory-taxonomy-engram-map.md:19, 35, 41, 46 | Dated research snapshot. It is accurate for its date. |
| 117 | README.md:87 (`engram query-chunks`, the local command) | The local CLI command stays. Only the served route is deleted (Q4). |
