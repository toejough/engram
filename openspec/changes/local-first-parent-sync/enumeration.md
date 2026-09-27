# Enumeration: local-first-parent-sync

This change alters three invariants:

- **Remote mode.** There is no longer an `ENGRAM_SERVER` thin client. Setting the variable is a hard error.
- **Where writes land.** A write always lands locally first, and is then offered to the parent when `ENGRAM_PARENT` is set.
- **What a merged query shows.** It shows one copy of each note (the local one), plus pull-down on activate.

Every row below that names `ENGRAM_SERVER`, the thin client, served `amend`, the `{status, luhmann}` receipt, runbook-pending-needs-`skill_hash`, or read-only-parent wording must be rewritten, deleted, or explicitly kept.

**Search (2026-09-27, worktree `runbook-vs-skill`, branch `local-first-parent-sync` @ `e1930b56`).**

```
grep -rn "ENGRAM_SERVER" --exclude-dir=.git .
```

This gave 19 files outside the archive (listed below) and 15 files / 54 lines inside `openspec/changes/archive/`.

The same search was repeated for these related identifiers:

- `serverBase`, `envServerBase`, `fetchQuery`, `fetchQueryChunks`, `fetchActivate`, `fetchAmend`, `fetchLearn`
- `printOfferReceipt`, `offerReceipt`, `serveAmend`, `/amend`
- `errDiscardOverServer`, `errRegisterSkillsOverServer`, `ExportServerBase`, `noteHasPendingMarker`
- `thin client`, `transparent HTTP`, `served learn/amend`, `two doors`

That search covered `internal/`, `cmd/`, `docs/`, `openspec/specs/`, `agent-instructions/`, `README.md` and `CLAUDE.md`. Every row was read at the cited line before its disposition was written (vault note 1072). The `CLAUDE.md` and `agent-instructions/` search returned **no** `ENGRAM_SERVER` hits. The skill rows below come from the served/offer wording instead.

**Disposition key:**

- *delete*: remove the code or text.
- *rewrite*: replace the text.
- *append*: add without editing the history.
- *code*: a Go change, made in a TDD task.
- *archive-time*: a Purpose-section edit, which a delta cannot make.
- *no change*: the reason is given in the row.

**Verification rule for performing rows.** After each edit, grep for the new text (it must be present) and for the old text (it must be absent) before ticking the row. Row 60 must end with zero `ENGRAM_SERVER` hits outside the allowlist it names.

## A. Go production code

| # | Location (verified) | Current | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 1 | internal/cli/serve_client.go:20 | `envServerBase = "ENGRAM_SERVER"` | code | Keep the constant only for the hard-error guard (D1), and rename it `envRemovedServer`. |
| 2 | internal/cli/serve_client.go:397-406 | `serverBase(deps)` | delete | It is replaced by the pre-dispatch guard (D1). |
| 3 | internal/cli/targets.go:146-156 | amend: `serverBase` branch, `errDiscardOverServer` refusal, `fetchAmend` | delete | Amend always runs locally; content amends enqueue an offer (D5/D6). |
| 4 | internal/cli/targets.go:214-218 | query: `fetchQuery` branch ahead of the parent merge | delete | The merge is the only remote path. |
| 5 | internal/cli/targets.go:244-248 | query-chunks: `fetchQueryChunks` branch | delete | It runs locally. |
| 6 | internal/cli/targets.go:269-273, 280-284, 291-295 | learn feedback/fact/runbook: `fetchLearn` branches | delete | Learn is always local and then enqueues an offer (D5/D6). |
| 7 | internal/cli/targets.go:371-375 + errRegisterSkillsOverServer (124-129) | register-skills refusal | delete | The D1 guard covers this. |
| 8 | internal/cli/targets.go:117-123 | `errDiscardOverServer` + its comment | delete | There is no server mode to guard. |
| 9 | internal/cli/targets.go:356-357 | comment naming both refusal errors | rewrite | Drop the `ENGRAM_SERVER` refusal clause. |
| 10 | internal/cli/targets.go:446-450 | activate: `fetchActivate` branch | code | Replace it with local-first resolution, a parent fallback that pulls down, and `--parent` (D8). |
| 11 | internal/cli/targets.go:478-482, 500-504 | show/show-chunk: server branches ahead of `--parent` | delete | `--parent` and the fallback remain. |
| 12 | internal/cli/targets.go:428-431 (serve desc) | "Serve query/query-chunks/show/show-chunk/activate/learn/amend over HTTP … learn/amend land as …" | code | Drop `amend` and say "learn lands as a pending offer" (D7). |
| 13 | internal/cli/serve_client.go:158-167 | `fetchActivate` | code | Repurpose it as the best-effort parent `/activate` after a pull-down (D8 step 4), and say "parent" in its doc. |
| 14 | internal/cli/serve_client.go:169-184 | `fetchAmend` | delete | Amend-offers go over `/learn` (D7). |
| 15 | internal/cli/serve_client.go:206-225 | `fetchLearn` "through ENGRAM_SERVER" | code | Becomes the offer sender used by the outbox drain (D6): it takes a payload built from the note, adds `offer.{for,key}`, and returns the parsed receipt. |
| 16 | internal/cli/serve_client.go:227-243 | `fetchQuery`, `fetchQueryChunks` | delete | No caller is left. |
| 17 | internal/cli/serve_client.go:42-61 (`buildQueryParams` doc :43) | "shared by fetchQuery (ENGRAM_SERVER…) and fetchQueryPayload" | code | Its only caller will be the parent fetch. Add `dedupe-keys=1` (D4). |
| 18 | internal/cli/serve_client.go:93-96, 266, 284-292, 335-337 | comments naming `ENGRAM_SERVER` | rewrite | Say "parent" instead. |
| 19 | internal/cli/serve_client.go:365-383 | `printOfferReceipt` `{status, luhmann}` | code | Parse the extended receipt (`basename`, `pending`) for linking (D4), and warn when `basename` is missing. |
| 20 | internal/cli/serve.go:122 + serveAmend (292-336) | `POST /amend` route | delete | Removed from the served set (D7). |
| 21 | internal/cli/serve.go:158-165 (`offerReceipt`) | `{Status, Luhmann}` | code | Add `Basename`, `Pending` (D4). |
| 22 | internal/cli/serve.go:349-384 (`serveLearn`) | honors client target/position | code | Force top-level placement, resolve `offer.for`, dedupe on `offer.key`, and return the extended receipt (D7). |
| 23 | internal/cli/serve.go:438-450 (`serveShow`) | any error → 500; rendered output | code | Add the `raw=1` byte-exact mode and a 404 for not-found (D8). |
| 24 | internal/cli/serve.go:390-414 (`serveQuery`) | no dedupe keys | code | Add the `dedupe-keys=1` parameter (D4). |
| 25 | internal/cli/offer.go:89-114 (`noteHasPendingMarker`) | runbook pending only with `skill_hash` | code | Any type with `pending: true` counts as pending (D7/G1). The real vault was checked read-only on 2026-09-27: no pending notes. |
| 26 | internal/cli/offer.go:150 | comment "served learn/amend write is awaiting curation" | rewrite | "an offered or pulled-down note is awaiting curation". |
| 27 | internal/cli/learn.go:86-98 | `SkillHash`/`SkillKey`/`SkillSource` `json:"-"` | code | Add the `Parent`/`Aliases` fields tagged `json:"-"`, plus `Offer` (the only new wire-settable field) (D3/D7). |
| 28 | internal/cli/learn.go:274, 316, 395 | fact/feedback/runbook frontmatter structs | code | Add `parent` (nested) and `aliases` (D3). |
| 29 | internal/cli/learn.go:573-575 | `learnArgsFrom*` doc: "ENGRAM_SERVER-mode (fetchLearn) dispatch" | rewrite | Say "and the offer payload builder". |
| 30 | internal/cli/learn.go (RunLearn ~:204) | local write only | code | Enqueue under the lock, then drain (D6). |
| 31 | internal/cli/amend.go:22-77, 156, 460-474 | `--discard` only; no offer | code | Add `--into` (D10). A content amend enqueues an offer (D5). Survival of `parent`/`aliases` in `overrideFactFields`/`overrideFeedbackFields`/`applyRunbookAmend`. |
| 32 | internal/cli/amend.go:53 | comment "never touches the served /amend path" | rewrite | That path no longer exists. Drop the clause. |
| 33 | internal/cli/activate.go:13-16, 38-75 | `{Vault, Notes}`; errors only when all fail | code | Add `--parent`, pull-down, per-ref stderr reporting, and non-zero on any failure (D8). |
| 34 | internal/cli/resituate.go:223, 256 | hand-copied field list | code | Copy `parent`/`aliases` (D3 survival) and enqueue an offer (D5). |
| 35 | internal/cli/identity_backfill.go:61, 90 | typed round-trip | code | Survival of `parent`/`aliases`. No offer. |
| 36 | internal/cli/skillreg_accept.go:317 (`applySkillNoteBody`), 102, 511 | typed runbook round-trip | code | Survival of `parent`/`aliases`. |
| 37 | internal/cli/merged_query.go:100-136, 177-191 | no dedupe; flat score sort; zero budget | code | Dedupe (D4), direct-before-explore ordering + note floor (#744), budget block (#743), drain after success (D6). |
| 38 | internal/cli/qa.go:221 + vault_init.go:44 | `ensureVaultDir` from learn/qa only | code | Run the shared `ensureVault` on every vault-resolving command, print the creation notice, and add the outbox line to `.gitignore` (D2). |
| 39 | internal/cli/update.go (vault notices, ~:481, 904) | pending-offer notice etc. | code | Add the outbox notice and the drain (D6). |
| 40 | internal/cli/deps.go:74-80 | `Deps.Fetch` doc: "Used by CLI targets when ENGRAM_SERVER is set" | rewrite | "Used for parent requests (ENGRAM_PARENT) and by nothing else". |
| 41 | internal/cli/primitives.go:104-109 | `Primitives.HTTP` doc naming "ENGRAM_SERVER-mode CLI targets" | rewrite | "parent-sync requests". |
| 42 | internal/cli/show.go:19-23, show_chunk.go:16-20 | `Parent` flag doc: "inert when ENGRAM_SERVER is also set" | rewrite | Drop the clause. |
| 43 | cmd/engram/serve.go:1-2, 29-36 | file comment and `fetchClientTimeout`/`fetchHTTPClient` docs "ENGRAM_SERVER client mode" | rewrite | "ENGRAM_PARENT parent-sync client". The code is unchanged (thin-api). |
| 44 | new: `internal/cli/outbox.go` (+ FS adapter via `Primitives`) | none | code | Outbox load/save (temp-rename), enqueue, drain (D6). DI only (ADR-0013, `targ check-thin-api`). |
| 45 | new: `internal/cli/pulldown.go` | none | code | Raw fetch, local pending write, idempotency (D8). |

## B. Go tests

| # | Location (verified) | Current | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 46 | internal/cli/serve_client_test.go:27-448 (`TestEngramServer_*`, 17 tests) | thin-client routing | delete | Replaced by row 47's guard test and by the offer, outbox and pull-down tests. |
| 47 | new test | none | code | `TestEngramServerSet_HardErrorsEveryCommand`: table over every subcommand; no FS or Fetch call; message names `ENGRAM_PARENT`. |
| 48 | internal/cli/serve_client_test.go:733 `TestServerBase_NilGetenv`; :827 `TestShowChunkParent_InertWhenEngramServerSet`; :998 `TestShowParent_InertWhenEngramServerSet` | server-mode assertions | delete | The behavior they test is gone. |
| 49 | internal/cli/serve_client_test.go comments :25, 286, 401, 662-664, 896, 970, 1019, 1089, 1116 | mention `ENGRAM_SERVER` | rewrite | Say "parent". |
| 50 | internal/cli/merged_query_dispatch_test.go:19-44 `TestTargets_Query_BothEnvVarsSet_ServerTakesPrecedence` | precedence | delete | Row 47 covers `ENGRAM_SERVER`. |
| 51 | internal/cli/register_skills_cli_test.go:269-284 `TestRegisterSkillsCLI_RefusesOverServer` | refusal | delete | Row 47 covers it. |
| 52 | internal/cli/targets_test.go:552 | comment | rewrite | Drop the mention. |
| 53 | internal/cli/export_test.go:164 `ExportServerBase` | export | delete | — |
| 54 | cmd/engram/serve_integration_test.go:69 | comment "engram serve/ENGRAM_SERVER" | rewrite | "engram serve / ENGRAM_PARENT". |
| 55 | internal/cli/query_integration_test.go:122-138 `envWithoutEngramParent`; register_skills_cli_test.go:254 | strips only `ENGRAM_PARENT` | code | Also strip `ENGRAM_SERVER`, so that a developer shell that still has it set cannot fail the subprocess tests. |
| 56 | internal/cli/serve_test.go:56, 101-180, 528 (`/amend` route and tests) | served amend | code | Delete the `/amend` route tests. Keep `TestServeLearn_IgnoresSkillIdentityFields` (:307) and extend it with `parent`/`aliases` keys. Add tests for receipt, `offer.for`, `offer.key`, placement, raw show and dedupe keys. |
| 57 | internal/cli/offer_test.go:136-147 | asserts a runbook without `skill_hash` is NOT pending | code | Invert it (G1). |
| 58 | internal/cli/serve_client_test.go:608, 626 `TestLocal{Amend,Learn}_NeverSetsPendingMarker` | local writes never pending | no change | Still true. Offers are pending on the parent, not locally. |
| 59 | internal/cli/skillfields_survival_test.go | skill-field survival per rewrite site | code | Add sibling `parent`/`aliases` survival tests for every site named in `vault-note-identity`. |

## C. Final sweep

| # | Location | Current | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 60 | whole repo | 19 files with `ENGRAM_SERVER` | code | After all rows, `grep -rn ENGRAM_SERVER --exclude-dir=.git .` hits **only**: the D1 guard constant and its test; the four spec deltas/REMOVED blocks and ADR-0029; README's migration sentence; `openspec/changes/archive/**`; `.review/events.jsonl`; `dev/eval/audit/results/transcript-events.jsonl`. The last two are historical capture and stay. |

## D. Specs (`openspec/specs/`)

| # | Location (verified) | Current | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 61 | vault-serve-api/spec.md:3 (Purpose) | "Lets remote environments with no local checkout or binary path use the vault over HTTP…" | archive-time | "Lets child environments, each with its own local vault, exchange notes with this parent over HTTP (offers up, pull-down reads), reusing the CLI's code paths and locks…" |
| 62 | vault-serve-api/spec.md:30-35 "The CLI is a transparent HTTP client when ENGRAM_SERVER is set" | requirement | delete (REMOVED in delta) | See the delta. |
| 63 | vault-serve-api/spec.md:5, 23, 37, 57 | served set incl. `amend`; `learn`/`amend` wording | rewrite (MODIFIED in delta) | See the delta. |
| 64 | vault-merged-recall/spec.md:3-6 (Purpose) | "Lets a node with its own local vault also merge…" | archive-time | Add: "…and dedupe linked or identical notes, keeping the local copy; merge is the standard path for any environment with a parent". |
| 65 | vault-merged-recall/spec.md:20, 135 | "`ENGRAM_SERVER` is not" clauses | rewrite (MODIFIED in delta) | — |
| 66 | vault-merged-recall/spec.md:167, 185 | precedence and server-exclusive requirements | delete (REMOVED in delta); :185 is re-ADDED as local-only | — |
| 67 | recall-payload-cuts/spec.md:40-41 | "local, `ENGRAM_SERVER`-exclusive, and `ENGRAM_PARENT`-merged" | rewrite (MODIFIED in delta) | — |
| 68 | vault-offer-curation/spec.md:3 (Purpose) | "Gives served writes a curated acceptance step…: a note arriving over the API…" | archive-time | "Gives offered notes — arriving from a child over the API, or pulled down from the parent — a curated acceptance step…". |
| 69 | vault-offer-curation/spec.md:7, 14, 28, 55, 112 | served learn/amend; activate; curation actions; receipt; runbook scope | rewrite (MODIFIED in delta) | — |
| 70 | vault-note-identity/spec.md:3 (Purpose) | "…(filtering, attribution, future multi-vault exchange)" | archive-time | Replace "future multi-vault exchange" with "parent exchange: URL-qualified `parent` links and `aliases`". |
| 71 | guidance-runbook-follow-frame/spec.md:183-187 | parent runbook resolved via the `show` fallback | no change | The show fallback is kept (MODIFIED text only drops the `ENGRAM_SERVER` clause). |
| 72 | runbook-lexical-triggers/spec.md:25-34 "Served round-trip" | a client builds a served query | no change | The parent's served `/query` still receives `--text` from the merged query. |
| 73 | learn-rate-skill-only (active change) tasks 3.2-3.4 | re-measures learn firing against the baseline | no change | Collision sweep: no shared capability or requirement header. Row 96's learn SKILL.md edit adds offer wording and no firing cues, so it does not confound W2. Record the edit date in that change's LEDGER row if 3.2 runs after it. |

## E. Docs

| # | Location (verified) | Current (abridged) | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 74 | README.md:93 (text on :93; :94 shares the fallback) | "On a local miss, with `ENGRAM_SERVER` unset and `--parent` not passed, `engram show`/`engram show-chunk` fall back to `ENGRAM_PARENT`…" | rewrite | Drop "with `ENGRAM_SERVER` unset". |
| 75 | README.md:95 (`engram amend …`), :96 (`engram activate …`) | no `--into`; activate local only | rewrite | `amend` gains `[--discard [--into <existing>]]`. `activate` gains `[--parent]`: a local miss with `ENGRAM_PARENT` set pulls the parent note down as a local pending offer. |
| 76 | README.md:105-109 (two-doors section) | "exposing only a fixed subset (`…activate, learn, amend`)"; "A served `learn`/`amend` always lands as a pending offer" | rewrite | Drop `amend` from the served subset. "A served `learn` (an offer from a child) always lands as a pending offer; so does a note a child pulls down from its parent." |
| 77 | README.md:111 | "Set `ENGRAM_SERVER=http://host:port` to make the CLI a transparent HTTP client…" | rewrite | "`ENGRAM_SERVER` is no longer supported — setting it is a hard error; set `ENGRAM_PARENT` to the same URL." Keep the identity half (a served `learn` stamps the declared `user:`, and there is no edge auth). |
| 78 | README.md:113-115 ("Merged recall: `ENGRAM_PARENT`") | "`ENGRAM_SERVER` is exclusive…"; "…takes full precedence…" | rewrite | Rewrite it as "Parent sync: `ENGRAM_PARENT`". Cover: the local vault is always created; learn and content amend are offered, with the outbox; merged query dedupes, keeping local; activate pulls down; chunks never travel. Keep the model_id, `from_parent`, `--parent` and unreachable-parent sentences. |
| 79 | README.md:117 | "…in every mode — local, `ENGRAM_SERVER`-exclusive, and `ENGRAM_PARENT`-merged alike" | rewrite | "…local and `ENGRAM_PARENT`-merged alike". |
| 80 | docs/GLOSSARY.md:453 | "On the served path (`ENGRAM_SERVER`), `text` is capped at 2 KB" | rewrite | "On the served path (a parent's `/query`, reached via `ENGRAM_PARENT`)…". |
| 81 | docs/GLOSSARY.md (new entries) | no entries for pending offer / outbox / parent link / pull-down | append | Add `### parent link`, `### outbox`, `### pull-down`, and `### offer` (learn-offer / amend-offer), each pointing to its capability. |
| 82 | docs/architecture/adr.md (after ADR-0028, ~:1086+) | none | append | New **ADR-0029 — Local-first parent sync**: decision 784a; removal of `ENGRAM_SERVER`; D1–D11 in summary; alternatives rejected; link to this change. Also append a one-line forward pointer to ADR-0027's memory-poisoning bullet (:1067) noting that pull-down makes parent content pass local curation. |
| 83 | docs/architecture/c2-containers.md:41 (C1 row) | "`/curate` (judges `engram serve` pending offers)" | rewrite | "(judges pending offers — served from children or pulled down from the parent)". |
| 84 | docs/architecture/c2-containers.md:44 (C4 Vault row) | "`.luhmann.lock` (flock)" | rewrite | Add "`.engram-outbox.json` (queued parent offers, gitignored)". |
| 85 | docs/architecture/c3-components.md component table (after K13, :112) | no serve, merge, or parent-sync component | append | Add a K row for `cli/serve.go` + `cli/serve_client.go` + `cli/merged_query.go` + `cli/outbox.go` + `cli/pulldown.go`: offers, outbox, merged dedupe, pull-down. |
| 86 | docs/architecture/c1-system-context.md (systems table :46-47) | no parent-vault external system | append | Add an external system "Parent engram vault (`engram serve`)" and a relation "offers notes / merged query / pull-down". |
| 87 | docs/ROADMAP.md:100 (NOW rank 17, #766) | "#766 create the local vault automatically…" | rewrite | Mark it folded into `local-first-parent-sync`. Add a NOW row for this change (next free rank). |
| 88 | dev/eval/LEDGER.md (new row) | none | append | A `local-first-parent-sync` row: unit coverage summary and real-binary verification outcomes (task group 12). No paid eval. |

## F. Skills (each edit through `superpowers:writing-skills` RED→GREEN→REFACTOR)

| # | Location (verified) | Current | Disposition | Replacement / reason |
| --- | --- | --- | --- | --- |
| 89 | agent-instructions/skills/curate/SKILL.md:3-8 (description) | "…or engram serve has just accepted a served write…" | rewrite | "…or a child's offer or a pulled-down parent note is pending…". Host-local only, as now. |
| 90 | agent-instructions/skills/curate/SKILL.md:13 | "A served `engram learn`/`engram amend` write lands as a **pending offer**" | rewrite | "An offer — a served `engram learn` from a child, or a parent note pulled down by `engram activate` — lands as a **pending offer**". |
| 91 | agent-instructions/skills/curate/SKILL.md:63-64 (covered/near rows) | `… then engram amend --discard` | rewrite | End with `engram amend --target <offer> --discard --into <existing>` (D10). Add: judge an offer with `offer.for` against that note first. |
| 92 | agent-instructions/skills/curate/SKILL.md:97, 102 (red flags) | "covered/near both end in `--discard`"; "curated from inside a served HTTP request" | rewrite | "…end in `--discard --into <existing>`". Keep the host-local flag. |
| 93 | agent-instructions/skills/recall/SKILL.md:207-220 (Step 2.7) | activate used notes; paths are local | rewrite | "Activate `from_parent` items you used the same way — this pulls them down as local pending offers; the next payload's `pending_offers` then routes to curation." |
| 94 | agent-instructions/skills/recall/SKILL.md:330-339 (red flags) | none about parent items | append | New row: "You ran `engram amend` on a `from_parent` item → it does not resolve locally; activate it (pull-down) and let curation fold it". |
| 95 | agent-instructions/skills/recall/SKILL.md:108 | "One call; the binary merges ranking server-side." | rewrite | "One call; with `ENGRAM_PARENT` set the binary merges local and parent results and dedupes, keeping local copies." |
| 96 | agent-instructions/skills/learn/SKILL.md (Step 2 write sites) | no parent wording | append | One sentence: with `ENGRAM_PARENT` set, every write is also offered to the parent automatically. A "parent unreachable; N offer(s) queued" warning is not a failure. Never set `ENGRAM_SERVER`. |
| 97 | vault runbook mirrors `skill-claude-curate`, `skill-claude-recall`, `skill-claude-learn` | mirror the old SKILL.md | no change (by hand) | Refreshed through `engram update` registration offers after deploy (`skill-runbook-registration`). Never edited by hand, and not in the real vault during this change. |
| 98 | dev/eval/cumulative/runbook_vs_skill/phase2/encodings/taskCurate/** | frozen eval fixtures | no change | Frozen historical fixtures. |

## G. Historical (no change)

| # | Location | Reason |
| --- | --- | --- |
| 99 | openspec/changes/archive/** (15 files, 54 lines naming `ENGRAM_SERVER`) | Archive history of the changes that formed the specs. |
| 100 | .review/events.jsonl:26, 28; dev/eval/audit/results/transcript-events.jsonl:44, 61, 324, 516 | Captured review and transcript events. |
| 101 | docs/research/2026-08-30-memory-taxonomy-engram-map.md:19, 41, 87 | Dated research snapshot. It mentions `ENGRAM_PARENT`/`--parent` accurately for its date. |
