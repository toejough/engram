# 792g8 vault-counting - Adoption Audit

Audit of [792a8 vault-counting](../792a_product-spec.md#792a8-added-vault-counting) and the [792b25 vault-counter](../792b_technical-spec.md#792b25-added-vault-counter) requirements whose `Supports:` lines cite its scenarios. Method: read `internal/cli/count.go` (the only implementation; last changed by 205e9acf, an EdgeFS refactor with no behavior change, after 56eb2617 added it on 2026-07-08), `internal/vaultgraph/scanner.go` and `parser.go`, then ran every scenario on a binary built from this tree against a temp vault, with HOME, XDG, `ENGRAM_VAULT_PATH` and `ENGRAM_CHUNKS_DIR` in `/tmp/c8`.

| Spec ID (linked) | Verdict | Evidence |
|---|---|---|
| [792a8 vault-counting](../792a_product-spec.md#792a8-added-vault-counting) (section intro) | covered by #795/#796 | The intro says a missing vault is created first. Binary: `count` on a nonexistent `ENGRAM_VAULT_PATH` printed `engram: created new vault at ...` on stderr and created `.engram`, `.engram-vault-id`, `.gitignore`, `.obsidian`, `README.md` (targets.go `runInVault`). #795/#796 change creation and the README, so the intro is left as is. `engram query` contrast and route-skill use are unchanged |
| [792a8a Count notes by attribute value](../792a_product-spec.md#792a8a-count-notes-by-attribute-value) | match | count.go `runCountGroupBy`, `renderCountGroupBy` (sort count desc, value asc), `attrValues` (distinct elements). Binary: `--group-by vocab` gave `dup 1, other 1, x 1, y 1, z 1` |
| [792a8a1a A scalar attribute is counted per value](../792a_product-spec.md#792a8a1-uat-scenarios) | match | binary: `--group-by type` printed `value<TAB>count` lines then `total: N`; order by count desc. Count 349/192/8 is illustrative; also `TestRunCount_*` in internal/cli/count_test.go |
| [792a8a1b A list attribute counts each distinct element once per note](../792a_product-spec.md#792a8a1-uat-scenarios) | match | binary: a note with `vocab: [x, y, z]` gave each of x, y, z once |
| [792a8a1c A repeated element counts once](../792a_product-spec.md#792a8a1-uat-scenarios) | match | binary: `vocab: [dup, dup, other]` gave `dup 1`, `other 1` |
| [792a8a1d Equal counts are ordered by value](../792a_product-spec.md#792a8a1-uat-scenarios) | match | binary: `fact 3` printed before `feedback 3`; `sort.Slice` tie-break on value in `renderCountGroupBy` |
| [792a8a1e Only notes at the vault root are counted](../792a_product-spec.md#792a8a1-uat-scenarios) | match | binary: `sub/s.md` with `type: fact` was not counted (fact stayed at the root count); `ListMD` lists the root only |
| [792a8b Absent attributes and the total](../792a_product-spec.md#792a8b-absent-attributes-and-the-total) | match | count.go `renderCountGroupBy`; `(attr absent)` printed only when `absent > 0`, before `total`; total counts notes. Binary: `--group-by foo` where two notes had `foo:`/`foo: []` printed `(foo absent): 5` and `total: 7` (the two empty ones in the total only) |
| [792a8b1a Notes without the attribute are reported](../792a_product-spec.md#792a8b1-uat-scenarios) | match | binary: `x 1`, `(vocab absent): N`, `total: N` in that order |
| [792a8b1b No note has the attribute](../792a_product-spec.md#792a8b1-uat-scenarios) | match | binary: `--group-by nope` printed only `(nope absent): 7` and `total: 7` |
| [792a8b1c An empty vault prints nothing](../792a_product-spec.md#792a8b1-uat-scenarios) | match | binary: empty vault gave no stdout, no stderr, rc 0 (`renderCountGroupBy` returns when `total == 0`) |
| [792a8b1d An empty attribute counts only toward the total](../792a_product-spec.md#792a8b1-uat-scenarios) | match | binary: `foo:` and `foo: []` counted in the total, in no value line and not absent (`attrValues` returns present=true, no values) |
| [792a8c Filter the counted notes](../792a_product-spec.md#792a8c-filter-the-counted-notes) | match | count.go `parseCountFilters` (`strings.Cut` at first `=`), `matchesAllFilters`, `errCountBadFilter`. Binary: `--filter bad` printed `count: --filter must be attr=value: "bad"`, rc 1, empty stdout; `--filter type=fact` counted only fact notes |
| [792a8c1a A scalar filter matches by equality](../792a_product-spec.md#792a8c1-uat-scenarios) | match | binary: `--group-by type --filter type=fact` printed only `fact 3` and `total: 3` |
| [792a8c1b A list filter matches by membership](../792a_product-spec.md#792a8c1-uat-scenarios) | match | count.go `attrValues` + `slices.Contains` for lists; unit-tested in count_test.go (list filter case) |
| [792a8c1c Every filter must match](../792a_product-spec.md#792a8c1-uat-scenarios) | match | count.go `matchesAllFilters` loops over all filters (AND); count_test.go |
| [792a8c1d A note without the filter's attribute is excluded](../792a_product-spec.md#792a8c1-uat-scenarios) | match | `attrValues` returns no values for an absent key, so `slices.Contains` fails and the note is excluded; count_test.go |
| [792a8c1e No note matches](../792a_product-spec.md#792a8c1-uat-scenarios) | match | binary: `--filter type=zzz` printed nothing, rc 0 |
| [792a8c1f A filter without = is refused](../792a_product-spec.md#792a8c1-uat-scenarios) | match | binary: `--filter bad` failed with the quoted message, rc 1, nothing on stdout |
| [792a8d Notes that cannot be counted](../792a_product-spec.md#792a8d-notes-that-cannot-be-counted) | match | count.go `readNoteAttrs` returns false for an unreadable note, a missing frontmatter or a YAML error; the caller skips before `total++`. Binary: no-frontmatter note and malformed-YAML note were in no line and not in the total; the CRLF note counted as feedback |
| [792a8d1a A note without usable frontmatter is skipped](../792a_product-spec.md#792a8d1-uat-scenarios) | match | binary: a note with no frontmatter and one with `: : bad [` were skipped |
| [792a8d1b An unreadable note is skipped](../792a_product-spec.md#792a8d1-uat-scenarios) | match | binary: with `chmod 000` on a fact note, `--group-by type` skipped it (fact 8 became 7) and exited 0 |
| [792a8d1c A CRLF note is counted](../792a_product-spec.md#792a8d1-uat-scenarios) | match | binary: `---\r\ntype: feedback\r\n---` note counted as feedback (`splitFrontmatter` handles CRLF) |
| [792a8e Count the notes that link to a note](../792a_product-spec.md#792a8e-count-the-notes-that-link-to-a-note) | doc fixed | count.go `runCountBacklinks` -> `vaultgraph.ScanVault`/`BuildGraph`. Binary reproduces every clause. Doc fixed: the requirement listed a link "to a note that is not at the vault root" but not the case that scenario 792a8e1i tests, a link written in a note below the root; it now lists both. Legacy cli-count spec (b865289e) had only the three basic scenarios, so the later clauses were written for #792 and were checked here against the code |
| [792a8e1a A note with incoming links](../792a_product-spec.md#792a8e1-uat-scenarios) | match | binary: `--backlinks-of tgt` printed `in-degree: 3` then `l0`, `l1`, `l3` in ascending order |
| [792a8e1b A note with no incoming links](../792a_product-spec.md#792a8e1-uat-scenarios) | match | binary: `--backlinks-of nosuch` printed `in-degree: 0` only |
| [792a8e1c A basename that is not a note](../792a_product-spec.md#792a8e1-uat-scenarios) | match | binary: `--backlinks-of nosuch` printed `in-degree: 0` only (`graph.Incoming[target]` is empty) |
| [792a8e1d A repeated link counts once](../792a_product-spec.md#792a8e1-uat-scenarios) | match | binary: `l1` has `[[tgt]]` twice plus `[[tgt.md]]` and counted once |
| [792a8e1e A note's link to itself does not count](../792a_product-spec.md#792a8e1-uat-scenarios) | match | binary: `--backlinks-of self` with only a `[[self]]` link printed `in-degree: 0` |
| [792a8e1f A link inside a fenced code block does not count](../792a_product-spec.md#792a8e1-uat-scenarios) | match | binary: `l2` with `[[tgt]]` only inside a fence was not a linker (`ParseWikilinks` skips fences) |
| [792a8e1g A link ending in .md counts as the bare link](../792a_product-spec.md#792a8e1-uat-scenarios) | match | binary: `[[tgt.md]]` in `l1` counted as `tgt`; `--backlinks-of tgt.md` gave 0 because the name is matched exactly as given |
| [792a8e1h A link to a missing note counts for no note](../792a_product-spec.md#792a8e1-uat-scenarios) | match | binary: `--backlinks-of nowhere` with `[[nowhere]]` in `l3` printed `in-degree: 0` |
| [792a8e1i Only notes at the vault root link](../792a_product-spec.md#792a8e1-uat-scenarios) | match | binary: `sub/l4.md` containing `[[tgt]]` was not a linker |
| [792a8e1j An unreadable note fails the report](../792a_product-spec.md#792a8e1-uat-scenarios) | match | binary: `chmod 000` on a root note made `--backlinks-of tgt` fail `count: scan: reading /tmp/c8/vault/c.md: ... permission denied`, rc 1, empty stdout |
| [792a8f One mode per invocation](../792a_product-spec.md#792a8f-one-mode-per-invocation) | match | count.go `RunCount` (`errCountBothModes`), `runCountGroupBy` (`errCountNoMode`). Binary: both flags, no flags and `--filter` alone each failed with the quoted text, rc 1, nothing on stdout |
| [792a8f1a Both flags are refused](../792a_product-spec.md#792a8f1-uat-scenarios) | match | binary: `--group-by type --backlinks-of a` -> `count: --group-by and --backlinks-of are mutually exclusive` |
| [792a8f1b Neither flag is refused](../792a_product-spec.md#792a8f1-uat-scenarios) | match | binary: bare `count` -> `count: specify --group-by <attr> or --backlinks-of <basename>` |
| [792a8f1c A filter alone is not a mode](../792a_product-spec.md#792a8f1-uat-scenarios) | match | binary: `count --filter type=fact` -> the same no-mode message |
| [792a8g The two modes measure different things](../792a_product-spec.md#792a8g-the-two-modes-measure-different-things) | doc fixed | The two counts are independent (`runCountGroupBy` vs `runCountBacklinks`), as stated. Doc fixed: "the in-degree therefore equals the group-by count plus the non-member linkers" holds only if every carrier of the value links to the note. The legacy cli-count spec (b865289e, 2026-07-27, code from 56eb2617 on 2026-07-08) stated the same unqualified equation; the code never encoded it. Added the proviso |
| [792a8g1a The modes agree when every linker is a member](../792a_product-spec.md#792a8g1-uat-scenarios) | match | count_test.go `TestRunCount_GroupByBacklinksAgreement`; both linkers are members so 2 and 2 |
| [792a8g1b The in-degree exceeds the count by the non-member linkers](../792a_product-spec.md#792a8g1-uat-scenarios) | match | count_test.go `TestRunCount_BacklinksExceedGroupByForNonMemberLinkers`; group-by 1, in-degree 2, linkers `idx` and `n1` |
| [792a8h A count changes nothing and is exact](../792a_product-spec.md#792a8h-a-count-changes-nothing-and-is-exact) | match | count.go reads through `ListMD`/`ReadFile`/`Scan` only (`CountDeps` has no write capability); no sidecar is read. Binary: md5 of every vault file identical before and after both modes |
| [792a8h1a A count leaves the vault as it was](../792a_product-spec.md#792a8h1-uat-scenarios) | match | binary: md5 of all files identical after `--group-by` and `--backlinks-of`, no file created (existing vault) |
| [792a8h1b Every note is counted without sidecars](../792a_product-spec.md#792a8h1-uat-scenarios) | match | count.go never touches `.vec.json`; the test vaults had no sidecars. 600 is illustrative: the loop has no cap or ranking |

792b25 component requirements (their scenarios and `Supports:` lines map one to one onto the rows above):

| Spec ID (linked) | Verdict | Evidence |
|---|---|---|
| [792b25c1 Mode selection](../792b_technical-spec.md#792b25c1-mode-selection) | match | `RunCount` order: both-modes check first, then backlinks, else group-by (which rejects an empty attribute). Binary confirms messages with and without a filter |
| [792b25c2 Attribute membership counting](../792b_technical-spec.md#792b25c2-attribute-membership-counting) | match | `runCountGroupBy`, `attrValues`; `count: listing vault: %w` wrap at the `ListMD` error. Binary: `foo: 3` style scalars (`bar: 3`) counted as `3 1`; null/empty counted toward the total only |
| [792b25c3 Filters](../792b_technical-spec.md#792b25c3-filters) | match | `parseCountFilters` runs before `ListMD` in `runCountGroupBy`; a note lacking the attribute fails `slices.Contains` |
| [792b25c4 Uncountable notes](../792b_technical-spec.md#792b25c4-uncountable-notes) | match | `readNoteAttrs`; unmarshalling into `map[string]any` rejects non-mapping YAML |
| [792b25c5 Attribute count report](../792b_technical-spec.md#792b25c5-attribute-count-report) | match | `renderCountGroupBy` |
| [792b25c6 Incoming-link report](../792b_technical-spec.md#792b25c6-incoming-link-report) | match | `runCountBacklinks`; `count: scan: %w` wrap; binary showed the wrapped error |
| [792b25c7 The two measures are independent](../792b_technical-spec.md#792b25c7-the-two-measures-are-independent) | doc fixed | proviso added: equality needs every carrier to link (see 792a8g) |
| [792b25c8 No writes](../792b_technical-spec.md#792b25c8-no-writes) | match | `CountDeps` has no writer; binary md5 check |

## Summary

Behavior matches the spec everywhere. Two imprecisions in the documents, both fixed in the specs, none in the code. No code bugs and no issues filed or referenced.

- [792a8g The two modes measure different things](../792a_product-spec.md#792a8g-the-two-modes-measure-different-things) and [792b25c7 The two measures are independent](../792b_technical-spec.md#792b25c7-the-two-measures-are-independent): the unqualified equation "in-degree = group-by count + non-member linkers" was ported from the July 2026 cli-count spec. It holds only if every note that carries the value links to the node. Added the proviso to both.
- [792a8e Count the notes that link to a note](../792a_product-spec.md#792a8e-count-the-notes-that-link-to-a-note): the exclusion list now also names a link written in a note below the vault root (scenario 792a8e1i), which the requirement text had omitted.
- Vault creation on first use (section intro) is covered by #795/#796. No SKILL.md or guidance follow-ups.
