## Context

`engram query` (`internal/cli/query.go`) today: `QueryArgs` (`query.go:23-46`) carries `Phrases`,
`LazyChunks`, `Limit`, `Project`, `Timings`; `RunQuery` (`:69-101`) → `runQuery` (`:1833-1893`)
scans the vault, builds a matched set per phrase, clusters, assembles `resolvedItem`s, and renders
`queryPayload` (`:429-454`) whose `items[]` are `queryItem`s (`:406-427`). Final ordering is
`mergeProvenances` (`:1398-1449`) sorting by `resolvedItemLess` (`:1812-1826`): provenance-count desc
→ `maxProvenanceRank` desc → score desc, where roles map to ranks via `provenanceRankFor` (`:1557`;
constants `provenanceRankDirect = 3`, `provenanceRankClusterRep = 2` at `:150-157`). Items are then
capped by `renderQueryPayload` (`:1719-1771`) / `capItemsToLimit` (`:880`). Kind detection is a
string scan, `kindFromContent` (`:1167-1192`). The whole-vault metadata pass
`loadAllVaultNotesMeta` (`query_vault_meta.go:151-203`) already reads every note's content and
frontmatter once per query.

Runbook frontmatter is `runbookFrontmatterDoc` (`learn.go:340-358`); `red_flags` (commit `f5b44504`
+ the runbook-capture change) is the end-to-end precedent for a list field: `LearnArgs.RedFlags`
(`learn.go:66-70`), `LearnRunbookArgs.RedFlags` (`targets.go:60`), `AmendArgs.RedFlags`
(`amend.go:43`), `applyRunbookAmend` (`amend.go:345-395`), `renderRunbookFrontmatter`
(`learn.go:700-719`). `embed.ContentHash` (`internal/embed/hash.go:49-56`) hashes only the
`situation:` line and the body, so frontmatter list fields do not stale sidecars.

The served path (`engram serve`) round-trips query args as a query string: `buildQueryParams`
(`serve_client.go:36-51`) and `serveQuery` (`serve.go:371-385`).

Existing literal frontmatter matching to reuse: `engram count --filter attr=value`
(`count.go:78-153`: `readNoteAttrs`, `attrValues`, `matchesAllFilters`) and the in-query
`projectLineRE` filter (`query.go:606-625, 1143-1165`).

Spec `recall-runbook-surfacing` currently requires "rank purely by situation-similarity … no boost"
(ADR-0026). That requirement was written against a *task-type classifier* proposal; the measured
failure here (parked change, results `3.3*_please_shim_only_*.md`) is a different mechanism — a
literal cue the author chose — and is the evidence that amends it.

## Goals / Non-Goals

**Goals:**
- A runbook author can name literal cues; when the user's message contains one, that runbook is in
  the payload, first, every time — independent of how the agent paraphrases.
- No change to ranking or payload for runbooks without triggers, or for queries without `--text`.
- Sonnet-executable: every task names its files, functions, tests, and the command that proves it.

**Non-Goals:**
- A harness hook that feeds the raw prompt without the agent (candidate follow-on if `--text`
  fidelity measures poorly).
- Regex, fuzzy, stemmed, or embedding-assisted trigger matching. v1 is case-insensitive substring
  on whitespace-normalized text.
- Triggers on `fact`/`feedback`/`qa` notes.
- Retiring any skill.

## Decisions

**D1. Field shape: `triggers:` list of strings; flag `--trigger <text>` repeatable; amend is
replace-whole.** Mirrors `red_flags` exactly (same structs, same render, same amend semantics via
`slices.Equal` change detection) so the implementer copies a known path. Alternative: a
`trigger:` scalar — rejected, cues are naturally plural (`/please`, "take this end-to-end").

**D2. Raw text enters as `engram query --text "<msg>"`, a single string, distinct from
`--phrase`.** Trigger matching must never run on a paraphrase, and semantic matching must never
run on the raw message (the parked change measured the raw message as a semantic phrase: 0/95
lift, so embedding it is wasted work). `--text` is optional; absent → no trigger matching, payload
byte-identical to today. `--text` with no `--phrase` is allowed (trigger-only lookup). Served
path: `text` becomes one more query-string param in `buildQueryParams`/`serveQuery`.

**D3. Match rule: case-insensitive substring after collapsing runs of whitespace to one space, on
both sides; empty trigger strings are rejected at write time.** Simple, deterministic, explainable
in one sentence to a runbook author. `/please` matches "/please fix the flaky test";
"take this end-to-end" matches "Please take this end-to-end: rename…". Word-boundary and regex
matching are deferred until a measured false-positive appears.

**D4. Surfacing: a new provenance role `trigger` with rank `provenanceRankTrigger = 4` (above
`direct`), and trigger hits are exempt from the relevance floor, the match-set cap, the note
floor, and `--limit`.** `resolvedItemLess` already sorts by provenance rank, so a rank above
`direct` puts hits first with no new sort. Exemptions are required because a trigger hit may have
*no* embedding match at all (score 0) — that is the whole point. The item's `score` stays whatever
similarity produced (0 if none); `provenances` lists `trigger` (plus `direct` if it also matched
semantically). Alternative: a separate top-level `triggered:` payload key — rejected, the shim's
follow frame keys on `items[]` and `kind: runbook`; a second list would need new frame text.

**D5. Trigger index is built inside `loadAllVaultNotesMeta`** by parsing a `triggers:` list from
each note's frontmatter during the pass that already exists (no second vault scan, no new I/O;
`QueryDeps.Read/Scan` only). Only `type: runbook` notes contribute. Parse with the same
string-scan idiom as `capRedFlagsForPreview` (`redflags_truncation.go:27-62`) or the YAML doc
struct — implementer's choice, but the property test in task 2 must cover multi-line list rendering.

**D6. Shim: add `--text` to the first-action block with a one-line rule — the user's message,
pasted word for word, first ~300 chars if long, never rewritten.** Kept minimal because shim bytes
are paid on every session. The parked change's D10 caveat (agents copy the shim's example verbatim)
means the validation must check that agents paste *the user's* words, not the example's.

**D7. Trigger authoring guidance lives in write-memory's template, not the shim:** triggers are
specific cues — slash forms, distinctive multi-word phrases — never a lone common word. Over-fire is
the author's responsibility, and a hit is still only a *candidate*: the follow frame's "announce,
then decide" behavior is unchanged, and the runbook body's own applicability guard applies.

**D8. Spec amendment, not a new ranking philosophy.** `recall-runbook-surfacing`'s purity
requirement is narrowed by one explicit exception (author-declared literal cue present in the raw
user text). ADR-0026's rationale is amended with the measurement that motivated it. Nothing else
about ranking changes.

**D9. Validation bar.** (a) Unit/property tests prove ordering, exemptions, and parse/render.
(b) Headless RED/GREEN (shim without/with `--text`, 5 varied asks × 3 trials × 2 arms, fresh
`claude -p` each): GREEN passes `--text` containing the user's message verbatim up to whitespace differences (runs of whitespace, incl. newlines, compare equal — the matcher collapses them) in ≥ 27/30.
(c) Please eval rerun (`probe_phase2.py --task please --arms R --shim-only --model sonnet5 --n 3`)
with `triggers: ["/please", "take this end-to-end"]` amended onto note
`1045.2026-09-20.please-drive-ask-end-to-end` in the trial vault: runbook surfaced 3/3. Cost ~$4–8
+ ~$4.

**D9(c) note (2026-09-20).** After task 6.5, Joe added the plain word `please` as a third trigger:
the list is now `["/please", "take this end-to-end", "please"]` (real note 1045 and the fixture note).
This is a deliberate exception to the write-memory authoring rule against generic single-word
triggers. Over-firing is accepted: a hit is only a candidate, and the agent decides relevance from
the runbook's own applicability guard. The 6.2 measurement used the two-trigger list.

**D9(b) amendment (2026-09-20).** The original bar was strict verbatim, ≥ 27/30. Measured: strict
24/30, whitespace-tolerant 29/30 (RED 0/30 both; contamination gate passed). Joe amended the bar
to whitespace-tolerant after seeing the result, justified by matcher behavior
(`internal/cli/query_triggers.go` lowercases and collapses whitespace runs on both sides, so
whitespace differences cannot change a trigger match). The shim wording stays as deployed. Of the 6
strict misses, 5 were whitespace-only (newline flattened to a space on the quotes/backticks/newline ask); 1/30 is a real miss: an agent
dropped the opening "Please take this end-to-end:" prefix, which is the cue a trigger keys on.
That residual risk is known and left unfixed by choice. Original result: `results/6.1_text_fidelity_shim_sonnet5.md`.

## Risks / Trade-offs

- **[Risk] Agents paraphrase `--text` too.** → D9(b) measures it before (c). If < 27/30, the
  follow-on is the prompt-submit hook (option B); this change still ships the field + query path,
  which the hook would reuse.
- **[Risk] Over-fire from loose triggers.** → D7 authoring rule; `engram learn runbook` rejects
  empty/whitespace triggers; a trigger shorter than 3 characters is rejected too. A hit is a
  candidate the agent still judges.
- **[Risk] Served query-string encoding of long `--text`.** → round-trip test in
  `serve_client_test.go` with a 1 KB message containing quotes, newlines, and `&`.
- **[Trade-off] Bypassing `--limit` can push the payload over Claude Code's ~2 KB Bash preview.**
  → trigger hits are rendered with the same `red_flags` preview cap; bodies are already fetched via
  `engram show`. Acceptable: a hit is rare and is the item the agent must see.
- **[Risk] Shim growth.** → one added flag line + one sentence; report byte delta in task 5.

## Migration Plan

1. Ship the binary change (tasks 1–4); existing vaults need no migration (field optional, sidecars
   unaffected).
2. Update shim + write-memory template (task 5); deploy with `engram update --with-guidance` (this
   reinstalls the binary from the checkout — intended here).
3. Amend the promoted please runbook with triggers (task 6) — `engram amend`, never a hand edit.
4. Run validation (task 6). If (b) fails, stop and report; do not proceed to unblock #758.
Rollback: revert the commit; notes carrying `triggers:` remain valid (unknown field is ignored by
older binaries only if parsing is lenient — task 1 verifies `runbookFrontmatterDoc` decoding ignores
unknown keys, otherwise pin that behavior).

## Open Questions

- Should `--text` be capped server-side (e.g. 2 KB) to bound query-string size on the served path?
  Default: yes, truncate with no error, since trigger cues occur early in a message.
- Whether `/recall glance`-style two-word commands should be registered as triggers on the
  recall runbook when #760 promotes it — out of scope here, but D3's substring rule supports it.
