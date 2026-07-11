# #678 Vocab→Tags Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the vocab layer from the custom `vocab:` frontmatter key + `Vocab: [[vocab.*]]` body wikilinks + 27 minted hub/index note files to the #674 tags convention: member notes carry `tags: [vocab/<term>]`, definitions become parent-tagged recallable fact notes, the index is emergent, and the binary's four read paths and one write funnel move to the new representation.

**Architecture:** All writes already funnel through `WriteVocabAssignment` (internal/cli/vocab.go); it becomes a namespace-scoped editor of the shared `tags:` list that also strips the legacy channels on touch. The four readers (stats, refit, trigger, query-nomination) move from `vocab:`-key parsing to a shared `vocab/`-prefix extraction over `tags:`. Term definitions move from `vocab.<term>.md` special-type files to ordinary fact notes (`vocab-<term>-definition`, bare `vocab` tag), found by tag + slug parse — no registry file, no special types, no query exclusion. A one-shot idempotent `engram vocab migrate-tags` subcommand performs the vault migration (assignment-preserving representational rewrite; no re-scoring).

**Tech Stack:** Go (pure, no CGO), imptest + rapid + gomega, targ build system, `dev/eval/traps/gate.py` trap gate.

## Global Constraints

- `targ test` / `targ check-full` only — NEVER `go test` / `go vet` directly. `targ check-full` for the complete error list.
- Test stack: impgen-generated mocks driven interactively (imptest), rapid for properties, gomega assertions. nilaway+gomega patterns per `.claude/rules/go.md` (nil guard after `NotTo(HaveOccurred())`, `MatchError` over `err.Error()`, explicit nil guards before pointer field access).
- Line length ≤ 120 chars. Named constants, descriptive names, wrapped errors (`fmt.Errorf("...: %w", err)`), sentinel errors, `t.Parallel()` on every test and subtest (no shared mutable state).
- **Production vault (`/Users/joe/.local/share/engram/vault`) is NEVER touched by tests or fixtures** — all tests pin scratch dirs. Only Task 8 touches the real vault, backup first.
- Commit trailer: `AI-Used: [claude]` (never Co-Authored-By).
- Skill file edits (Task 9's `skills/write-memory/SKILL.md`) require superpowers:writing-skills TDD (RED baseline → edit → GREEN + pressure test).
- Trap gate: `python3 dev/eval/traps/gate.py --tier smoke` — BEFORE (Task 8 step 0, old binary + old vault) and AFTER (Task 8 final step, new binary + migrated vault). A C5b-honored-only RED at n=1 gets ONE same-tree re-run before being treated as real (vault note 209); RED twice or any other cell RED = real, STOP.
- The recall skill's priming nucleus (win-nucleus) is untouched — no recall/learn skill edits in this cycle.
- **Naming decision (Joe, 2026-07-10, supersedes the issue's dot form):** definition-note slugs use dashes everywhere — `vocab-definition` (family), `vocab-<term>-definition` (term) — matching the shipped #674 convention (`work-kind-definition` etc.). No renames of the #674 notes. The issue text's `vocab.definition` / `vocab.<term>.definition` is superseded by this decision.
- Tag grammar (shipped #674, unchanged): `^[a-z0-9-]+(/[a-z0-9-]+)?$`. Vocab namespace: `vocab/<term>` for members; the bare tag `vocab` marks a definition note (family or term). A definition note NEVER carries its own term tag.
- **Assignment is preserved, not re-scored:** the migration maps each note's existing `vocab: [a, b]` terms to `tags: [vocab/a, vocab/b]` verbatim. `vocab.centroids.json` is untouched (derived vectors + lifecycle state — not a term registry; kept per ADR-0011 mechanics).
- `embed.VocabBodyMarker` stripping in `isMachineLine` (internal/embed/hash.go) is KEPT this cycle — zero-risk hash stability if a backed-up note is ever restored. Flag as follow-up cleanup in the close-out.
- `engram count` is untouched (generic frontmatter reader; already handles `tags:` list-contains — verified under #674).

## Pre-measured vault facts (measured 2026-07-10 at HEAD 90f2fbd3, evidence: grep/ls on `/Users/joe/.local/share/engram/vault`)

| Fact | Value |
|---|---|
| Total `.md` notes | 225 |
| Total vault entries | 452 = 225 `.md` + 226 `.vec.json` (the +1 over 225 is the orphan sidecar) + `vocab.centroids.json` |
| Notes with `vocab:` frontmatter key | 187 (includes 5 `qa.*.md` answer notes and the 3 #674 route-family definition notes 204/205/206) |
| Notes with `Vocab:` body line | 187 |
| Hub files `vocab.*.md` | 27 (26 term notes + `vocab.index.md`) |
| Hub sidecars `vocab.*.vec.json` | 28 (27 + 1 orphan: `vocab.behavioral-failure-reproduction.vec.json`) |
| Notes with `tags:` key | 3 (the #674 definition notes; block style, 4-space indent) |
| Current `vocab_version` (from `vocab.index.md`) | "6.0" |
| Term count | 26, all kebab-case (all valid as `vocab/<term>` tags) |
| Highest luhmann ID | 209 |
| ContentHash invariance | `ContentHash` hashes `BodyText`, which strips frontmatter AND `Vocab:` machine lines (internal/embed/hash.go:41-84) — the member rewrite changes NO ContentHash; zero re-embeds expected for surviving notes |
| `vocab:`-key set == `Vocab:`-line set | verified byte-identical (comm -3 empty) — the "channel-consistency repair" branch in Task 7 has ZERO real instances today; it is defensive and needs a synthetic fixture |

The 3 route-family definition notes (204/205/206: `work-kind-definition` etc.) are ordinary MEMBERS of the vocab space — they carry auto-assigned `vocab:` keys alongside their bare family tag in a block-style `tags:` list. They migrate like any member: their vocab terms merge into the existing tags list, categorical tags preserved. This mixed shape is the key Task 1/Task 7 fixture (see Task 7 Step 1).

Expected post-migration: `^vocab:` count 0; `^Vocab:` count 0; `vocab.*.md` 0; `vocab.*.vec.json` 0; total `.md` = 225 (198 survivors + 27 minted definitions); total vault entries = 451 (225 md + 225 vec.json + centroids); `tags:` keys = 214 (187 members + 27 definitions); notes with a `vocab/` tag = 187; notes with bare `vocab` tag = 27; `*.vocab-*-definition.md` = 26; `*.vocab-definition.md` = 1; luhmann IDs 210–236 consumed.

## Decision log

- Dashes-everywhere naming: considered the issue's dot form (`vocab.<term>.definition`); Joe chose dashes 2026-07-10 to match shipped #674 notes and kebab slug/tag grammar.
- No new ADR-0020 (Gate A docs-alignment finding, accepted): ADR-0019 already owns the representation decision and names #678 as its vocab stage — a second ADR would fake an independent decision. Instead Task 9 annotates: ADR-0011 (representation retired per ADR-0019/#678; centroid-assignment + nomination MECHANISM unchanged), ADR-0018 (worked example marked historical — `vocab.index.md` retired), ADR-0019 (shipped-via-#678 note + consequences: definitions recallable, `--backlinks-of vocab.<term>` in-degree drops to 0). The issue's AC allows "ADR-0011/ADR-0018 (or a new ADR)" — annotations satisfy it.
- `count --backlinks-of vocab.<term>` goes to zero for vocab terms (the body wikilinks are the only vocab graph edges and they are removed). Accepted consequence of the ratified single-representation decision; ADR-0018's worked example is annotated historical in Task 9.
- Definitions become recallable (query exclusions deleted) — accepted behavior change per the issue; flagged in docs (Task 9).
- Migration vehicle: `engram vocab migrate-tags` subcommand (DI-testable, runs as the real binary with real args per vault note 54). Kept after the run as an idempotent no-op (enables backup-restore re-run); retirement decision deferred to follow-up triage.

---

### Task 1: Namespace-scoped tags writer (WriteVocabAssignment → tags channel)

**Files:**
- Modify: `internal/cli/vocab.go` (WriteVocabAssignment and its channel helpers)
- Test: `internal/cli/vocab_test.go` (rewrite `TestWriteVocabAssignment_*`), plus the write-site assertion updates in `internal/cli/learn_test.go` (`TestRunLearn_VocabAssignment_*`), `internal/cli/amend_test.go` (`TestRunAmend_VocabAssignment_*` / `TestApplyVocabAssignmentAfterAmend_TriggerFires`), `internal/cli/resituate_test.go` (`TestApplyVocabAssignmentAfterResituate_*`), `internal/cli/vocab_commands_test.go` (sidecar-stays-OK test and any fixture asserting `vocab:` output of the writer)

**Interfaces:**
- Consumes: existing `removeYAMLKey(frontmatter, key string) string`, `fmStart`/`fmEnd` constants, `vocabBodyMarker`.
- Produces: `WriteVocabAssignment(content string, terms []string) string` (same signature — call sites untouched); new exported-for-test helpers `parseTagsFromFrontmatter(frontmatter string) []string` and `vocabTermsFromTags(tags []string) []string`; new constant `vocabTagPrefix = "vocab/"`. Later tasks (2, 3, 4, 7) consume `vocabTermsFromTags` and `parseTagsFromFrontmatter`.

**Behavior spec (the new contract):**
`WriteVocabAssignment(content, terms)`:
1. Parses the existing `tags:` list from frontmatter (handles both block style `tags:\n    - a` and inline `tags: [a, b]`; absent key = empty list).
2. Splits it into non-vocab tags (kept, order preserved) and vocab-namespace tags (discarded).
3. Appends `vocab/<term>` for each term in `terms` (assignment order) after the kept tags.
4. Rewrites the `tags:` key in BLOCK style with 4-space indent (`tags:\n    - <tag>`), byte-identical to the #674 learn renderer's output, at the position of the existing `tags:` key, else where the legacy `vocab:` key sat, else immediately before the closing `---`. If the merged list is empty, the `tags:` key is removed.
5. ALWAYS removes the legacy `vocab:` frontmatter key and the legacy `Vocab:` body line (plus its preceding blank line) when present — migration-by-touch.
6. Idempotent: applying twice with the same terms yields identical bytes. Content without frontmatter is returned unchanged.

**Steps:**

- [ ] **Step 1: Write the failing tests** (replace the four `TestWriteVocabAssignment_*` tests; add the new cases). Core cases, gomega style, all `t.Parallel()`:

```go
func TestWriteVocabAssignment_WritesVocabNamespaceTags(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "---\ntype: fact\nsituation: s\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, []string{"retrieval-design", "token-budget"})

	g.Expect(got).To(gomega.ContainSubstring("tags:\n    - vocab/retrieval-design\n    - vocab/token-budget\n"))
	g.Expect(got).NotTo(gomega.ContainSubstring("\nvocab:"))
	g.Expect(got).NotTo(gomega.ContainSubstring("Vocab:"))
}

func TestWriteVocabAssignment_PreservesCategoricalTags(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "---\ntype: fact\ntags:\n    - work-kind/rename\n    - tier/cheap\n    - vocab/stale-term\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, []string{"retrieval-design"})

	g.Expect(got).To(gomega.ContainSubstring(
		"tags:\n    - work-kind/rename\n    - tier/cheap\n    - vocab/retrieval-design\n"))
	g.Expect(got).NotTo(gomega.ContainSubstring("vocab/stale-term"))
}

func TestWriteVocabAssignment_EmptyTermsRemovesVocabNamespaceOnly(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "---\ntype: fact\ntags:\n    - work-kind/rename\n    - vocab/stale-term\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, nil)

	g.Expect(got).To(gomega.ContainSubstring("tags:\n    - work-kind/rename\n"))
	g.Expect(got).NotTo(gomega.ContainSubstring("vocab/"))
}

func TestWriteVocabAssignment_EmptyTermsNoOtherTagsRemovesTagsKey(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "---\ntype: fact\ntags:\n    - vocab/stale-term\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, nil)

	g.Expect(got).NotTo(gomega.ContainSubstring("tags:"))
}

func TestWriteVocabAssignment_StripsLegacyChannels(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "---\ntype: fact\nvocab: [old-a, old-b]\n---\n\nBody.\n\nVocab: [[vocab.old-a]], [[vocab.old-b]]\n"
	got := cli.WriteVocabAssignment(content, []string{"old-a", "old-b"})

	g.Expect(got).To(gomega.ContainSubstring("tags:\n    - vocab/old-a\n    - vocab/old-b\n"))
	g.Expect(got).NotTo(gomega.ContainSubstring("\nvocab: ["))
	g.Expect(got).NotTo(gomega.ContainSubstring("Vocab: [["))
	g.Expect(strings.HasSuffix(got, "Body.\n")).To(gomega.BeTrue())
}

func TestWriteVocabAssignment_Idempotent(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "---\ntype: fact\ntags:\n    - work-kind/rename\nvocab: [a]\n---\n\nBody.\n\nVocab: [[vocab.a]]\n"
	once := cli.WriteVocabAssignment(content, []string{"a", "b"})
	twice := cli.WriteVocabAssignment(once, []string{"a", "b"})

	g.Expect(twice).To(gomega.Equal(once))
}

func TestWriteVocabAssignment_InlineTagsListParsed(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "---\ntype: fact\ntags: [work-kind/rename, vocab/stale]\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, []string{"fresh"})

	g.Expect(got).To(gomega.ContainSubstring("tags:\n    - work-kind/rename\n    - vocab/fresh\n"))
}

func TestWriteVocabAssignment_NoFrontmatterUnchanged(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	content := "Just a body, no frontmatter.\n"
	g.Expect(cli.WriteVocabAssignment(content, []string{"a"})).To(gomega.Equal(content))
}
```

Plus parser edge cases (semantics: absent key, key-with-no-values, empty inline list, and malformed YAML all return nil without panicking):

```go
func TestParseTagsFromFrontmatter_EdgeCases(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	g.Expect(cli.ExportParseTagsFromFrontmatter("type: fact")).To(gomega.BeNil())
	g.Expect(cli.ExportParseTagsFromFrontmatter("type: fact\ntags:")).To(gomega.BeNil())
	g.Expect(cli.ExportParseTagsFromFrontmatter("type: fact\ntags: []")).To(gomega.BeNil())
	g.Expect(cli.ExportParseTagsFromFrontmatter("type: fact\ntags: [")).To(gomega.BeNil())
}
```

Plus a rapid property test: for random non-vocab tag lists T and term lists V, `parseTagsFromFrontmatter` of the writer's output == T ++ map("vocab/"+_, V), and a second application is byte-identical.

Plus round-trip fidelity through the learn renderer: decode the writer's output with `factFrontmatterDoc` (as `TestRenderFactFrontmatter_TagsRoundtripFidelity` does) and assert the Tags field round-trips — this pins byte-compatibility with the #674 block style.

- [ ] **Step 2: Run to verify failure.** Run: `targ test`. Expected: new tests FAIL (writer still emits `vocab:` + `Vocab:` channels).

- [ ] **Step 3: Implement.** In `internal/cli/vocab.go`:

```go
// vocabTagPrefix namespaces vocab terms inside the shared tags: list.
const vocabTagPrefix = "vocab/"

// WriteVocabAssignment rewrites the vocab/<term> namespace of the note's
// tags: frontmatter list to exactly terms, preserving all non-vocab tags and
// their order. It also strips the legacy vocab: frontmatter key and Vocab:
// body line when present (migration-by-touch). Idempotency rule: the vocab
// namespace is replaced on every call — never appended. When terms is empty,
// the vocab namespace is removed; an emptied tags: key is removed entirely.
func WriteVocabAssignment(content string, terms []string) string {
	frontmatter, rest, ok := splitFrontmatterAndBody(content)
	if !ok {
		return content
	}

	kept := nonVocabTags(parseTagsFromFrontmatter(frontmatter))

	merged := make([]string, 0, len(kept)+len(terms))
	merged = append(merged, kept...)

	for _, term := range terms {
		merged = append(merged, vocabTagPrefix+term)
	}

	insertAt := yamlKeyLineIndex(frontmatter, "tags")
	if insertAt < 0 {
		insertAt = yamlKeyLineIndex(frontmatter, "vocab")
	}

	frontmatter = removeYAMLKey(frontmatter, "tags")
	frontmatter = removeYAMLKey(frontmatter, "vocab")

	if len(merged) > 0 {
		frontmatter = insertYAMLBlock(frontmatter, renderTagsBlock(merged), insertAt)
	}

	return fmStart + frontmatter + fmEnd + removeVocabBodyLine(rest)
}
```

(Reassembly matches the existing working pattern at today's vocab.go:324 — `frontmatter` carries no trailing
newline; the full `fmEnd` (`"\n---\n"`) supplies both the newline after the last key and the closing delimiter.
Do NOT slice `fmEnd` — `fmEnd[1:]` glues the last tag onto `---` and corrupts the frontmatter block.)

with helpers (exact implementations are the implementer's to write against the tests; required signatures and semantics):

```go
// splitFrontmatterAndBody cuts content into (frontmatter-without-delims, body-after-closing-delim, ok).
// NOTE: the name splitFrontmatter is TAKEN — resituate.go:313 declares
// splitFrontmatter(raw []byte) ([]byte, bool) with 9+ call sites across the
// package. Go has no overloading; reuse that helper internally if convenient,
// but the new two-part cut must use this distinct name.
func splitFrontmatterAndBody(content string) (string, string, bool)

// parseTagsFromFrontmatter returns the tags: list values, handling both
// block style ("tags:\n    - a") and inline style ("tags: [a, b]").
// Absent key or empty list returns nil.
func parseTagsFromFrontmatter(frontmatter string) []string

// nonVocabTags filters out entries in the vocab namespace (vocab/<term>)
// AND the bare "vocab" definition marker, preserving order.
func nonVocabTags(tags []string) []string

// vocabTermsFromTags returns the terms of the vocab namespace entries
// (prefix stripped), preserving order. The bare "vocab" tag is not a term.
func vocabTermsFromTags(tags []string) []string

// yamlKeyLineIndex returns the line index of a top-level key, or -1.
func yamlKeyLineIndex(frontmatter, key string) int

// renderTagsBlock renders the block-style list, 4-space indent:
// "tags:\n    - a\n    - b" (no trailing newline).
func renderTagsBlock(tags []string) string

// insertYAMLBlock inserts block at the given line index (append at end when
// index is -1 or out of range).
func insertYAMLBlock(frontmatter, block string, at int) string

// removeVocabBodyLine strips the Vocab: machine line (and one preceding
// blank line) from the body; unchanged when absent.
func removeVocabBodyLine(body string) string
```

Delete `replaceVocabFrontmatterList`, `replaceVocabBodyLine`, `replaceVocabBodyLineInSection`, `renderVocabBodyLine`, `renderVocabYAMLList` (the body-line REMOVAL logic replaces them; nothing writes the old channels anymore). Update the write-site tests named in Files to assert the new tags output.

- [ ] **Step 4: Run tests.** Run: `targ test`. Expected: PASS.
- [ ] **Step 5: Full check.** Run: `targ check-full`. Expected: green (fix all reported issues in one pass).
- [ ] **Step 6: Commit.** `git add -A && git commit` — message: `feat(vocab): WriteVocabAssignment writes the vocab/ namespace of tags:, strips legacy channels (#678)`

### Task 2: Bare-vocab definition exemption (auto-assign + member scans)

**Files:**
- Modify: `internal/cli/vocab.go` (`applyVocabAssignmentCore`), `internal/cli/vocab_commands.go` (member-scan sites currently using `isVocabRewriteExcluded` / filename checks: `assignTermsToAllNotes`, `collectVaultStats`, `clearRemovedTermsFromMembers`), `internal/cli/vocab_centroids.go` (`retagAllNotesTwoPass` AND `loadMemberNoteVectors` :172-199 — see the centroid-purity spec below), `internal/cli/vocab_trigger.go` (`collectTriggerVaultStats*`)
- Test: `internal/cli/vocab_test.go`, `internal/cli/vocab_commands_test.go`, `internal/cli/vocab_trigger_test.go`

**Centroid-purity spec (Gate A ask-alignment CRITICAL — this is the load-bearing part of this task):**
`loadMemberNoteVectors` (vocab_centroids.go:172-199) today filters by FILENAME (`isVocabKindFilename`) before reading sidecars. Post-migration no file matches that pattern, so the filter goes vacuous and a bare-`vocab` definition note's vector — whose body IS the term's own description — would be fed into `AssignVocabTerms` pass-1 and folded into `computeTermCentroids`, skewing every term's centroid toward its own definition on every bootstrap/refit. The fix: `loadMemberNoteVectors` must READ each note's content and skip `isVocabDefinitionNote` notes before including their vectors (the content read is cheap relative to the sidecar unmarshal it already does). This is a functional requirement of AC4 ("assignment similarity unchanged in behavior"), not an optional cleanup.

**Interfaces:**
- Consumes: Task 1's `parseTagsFromFrontmatter`.
- Produces: `isVocabDefinitionNote(content string) bool` — true when the note's tags contain the bare `vocab` tag. Consumed by Tasks 4, 5, 7.

**Steps:**

- [ ] **Step 1: Failing tests.**

```go
func TestIsVocabDefinitionNote(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	definition := "---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines a term.\n"
	member := "---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nA member.\n"
	otherFamily := "---\ntype: fact\ntags:\n    - work-kind\n---\n\nRoute family definition.\n"

	g.Expect(cli.IsVocabDefinitionNote(definition)).To(gomega.BeTrue())
	g.Expect(cli.IsVocabDefinitionNote(member)).To(gomega.BeFalse())
	g.Expect(cli.IsVocabDefinitionNote(otherFamily)).To(gomega.BeFalse())
}

func TestApplyVocabAssignmentCore_SkipsDefinitionNotes(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	definition := "---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines the retrieval-design term.\n"
	sidecar := mustMarshalSidecarWithBodyVector(t, []float32{1, 0, 0})
	loadTerms := func(string) ([]cli.TermWithVector, error) {
		return []cli.TermWithVector{{Term: "retrieval-design", Vector: []float32{1, 0, 0}}}, nil
	}
	read := func(string) ([]byte, error) { return sidecar, nil }

	wrote := false
	write := func(string, []byte) error { wrote = true; return nil }

	cli.ExportApplyVocabAssignmentCore(loadTerms, read, write, nil, "/v", "/v/n.md", definition, "test")

	g.Expect(wrote).To(gomega.BeFalse()) // a definition must never acquire its own term tag
}

func TestLoadMemberNoteVectors_ExcludesDefinitionNotes(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// scratch vault: 1 member note (with sidecar) + 1 bare-vocab definition
	// note (with sidecar). Assert the returned vector map contains ONLY the
	// member — a definition's vector must never reach pass-1 assignment or
	// computeTermCentroids (centroid purity, AC4).
	vault := t.TempDir()
	writeNoteAndSidecar(t, vault, "1.2026-07-10.member.md",
		"---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nMember body.\n", []float32{1, 0, 0})
	writeNoteAndSidecar(t, vault, "2.2026-07-10.vocab-retrieval-design-definition.md",
		"---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines the term.\n", []float32{0, 1, 0})

	vectors, err := cli.ExportLoadMemberNoteVectors(vault)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(vectors).To(gomega.HaveLen(1))
	g.Expect(vectors).To(gomega.HaveKey("1.2026-07-10.member.md"))
}

func TestCollectTriggerVaultStats_DefinitionsAreNeitherMembersNorUntagged(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// fixture: 1 definition + 2 members (one carrying a vocab/ tag, one with
	// no vocab namespace at all) → member count 2, untagged 1. The definition
	// contributes to neither tally.
	vault := t.TempDir()
	writeNote(t, vault, "1.2026-07-10.tagged-member.md",
		"---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nBody.\n")
	writeNote(t, vault, "2.2026-07-10.untagged-member.md", "---\ntype: fact\n---\n\nBody.\n")
	writeNote(t, vault, "3.2026-07-10.vocab-retrieval-design-definition.md",
		"---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines.\n")

	stats, err := cli.ExportCollectTriggerVaultStats(vault)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(stats.NoteCount).To(gomega.Equal(2))
	g.Expect(stats.UntaggedCount).To(gomega.Equal(1))
}
```

(Adapt helper names — `writeNote`/`writeNoteAndSidecar`/`mustMarshalSidecarWithBodyVector`, the exact deps
struct shapes, the stats field names, AND return arities — to the existing suite idioms: the real
`loadMemberNoteVectors(deps, vault)` returns only a map (no error), and `collectTriggerVaultStats` already
has a shim at export_test.go:46 taking `(vault, listMD, readFile)` and returning three bare values. The
ASSERTIONS above are the contract — the vector-map/member/untagged expectations — not the error plumbing.)

Plus: `retagAllNotesTwoPass` and `assignTermsToAllNotes` skip definition notes — fixture assertion: the definition file's bytes are unchanged after a full retag over a scratch vault containing it plus one member.

- [ ] **Step 2: Verify failure.** `targ test` — new tests FAIL (no exemption exists).
- [ ] **Step 3: Implement.** Add `isVocabDefinitionNote` (parse frontmatter tags, check bare `vocab`); guard `applyVocabAssignmentCore` right after the reads (`if isVocabDefinitionNote(content) { return }`); swap the member-scan exclusion sites from filename checks to `isVocabDefinitionNote` where the scan operates on content (keep filename checks where content is not yet loaded — they die in Task 6 with the hub files). Export via the existing `export_test.go` shim pattern.
- [ ] **Step 4: `targ test` → PASS.**
- [ ] **Step 5: `targ check-full` → green.**
- [ ] **Step 6: Commit:** `feat(vocab): bare-vocab definition notes exempt from term assignment and member scans (#678)`

### Task 3: Query tag-nomination reads the vocab/ namespace from tags:

**Files:**
- Modify: `internal/cli/query_nominations.go` (`noteQueryFrontmatter` struct :67-70, `parseNoteQueryFrontmatter` :394-414, `loadAllVaultNotesMeta` :308-354)
- Test: `internal/cli/query_nominations_test.go` (`TestBuildTagNominations_*`, `TestParseNoteQueryFrontmatter_*`, TermIndex tests — fixtures move from `vocab: [a, b]` to `tags:` block style)

**Interfaces:**
- Consumes: Task 1's `vocabTermsFromTags`.
- Produces: unchanged `TermIndex` semantics (bare term strings, prefix stripped) — downstream nomination logic (`buildTagNominations`, caps, budget fields) untouched.

**Steps:**

- [ ] **Step 1: Failing tests.** Update fixtures: every nomination-test note gets `tags:\n    - vocab/<term>` instead of `vocab: [<term>]`. Add two new cases: (a) a note whose tags mix categorical + vocab entries (`work-kind/rename`, `vocab/go-conventions`) — only `go-conventions` enters the TermIndex; (b) a bare-`vocab` definition note — contributes NO terms (bare tag is not a term) and is nominatable like any other note (no exclusion — that deletion completes in Task 6, but nothing in THIS task's code may filter it).
- [ ] **Step 2: Verify failure.** `targ test` — nomination tests FAIL (parser still reads `vocab:`).
- [ ] **Step 3: Implement.**

```go
// noteQueryFrontmatter is the minimal frontmatter shape the query path needs.
type noteQueryFrontmatter struct {
	Tags       []string `yaml:"tags"`
	Supersedes []string `yaml:"supersedes"`
}
```

`parseNoteQueryFrontmatter` decodes Tags and derives terms via `vocabTermsFromTags`; `loadAllVaultNotesMeta` consumes the derived terms exactly where it consumed `.Vocab`. Delete the `Vocab` field.

- [ ] **Step 4: `targ test` → PASS.**
- [ ] **Step 5: `targ check-full` → green.**
- [ ] **Step 6: Commit:** `feat(query): tag nomination reads vocab/ namespace from tags: frontmatter (#678)`

### Task 4: Definition-note read path — stats/refit/trigger + vocab_version home

**Files:**
- Modify: `internal/cli/vocab_commands.go` (`extractNoteVocabTags`, `collectVaultStats`, `collectCurrentTermEntries`, `emitRefitRequest`, `loadCurrentVocabVersion`, version-bump write sites, `loadTermVectors`), `internal/cli/vocab_trigger.go` (`collectTriggerVaultStatsFromNames`), `internal/cli/learn.go` (`factFrontmatterDoc` gains `VocabVersion string \`yaml:"vocab_version,omitempty"\`` between the Sources and Tags fields)
- Test: `internal/cli/vocab_commands_test.go`, `internal/cli/vocab_trigger_test.go`, `internal/cli/learn_test.go` (round-trip: `vocab_version` survives amend re-render)

**Interfaces:**
- Consumes: Tasks 1-2 helpers.
- Produces: `termFromDefinitionSlug(slug string) (string, bool)` — parses `vocab-<term>-definition` → `<term>`; returns false for `vocab-definition` (the family note) and non-matching slugs. `slugFromNoteFilename(name string) string` — extracts `<slug>` from `<id>.<date>.<slug>.md` (verified 2026-07-10: NO such helper exists in internal/cli — write it; the existing filename helpers `extractLuhmannFromFilename`/`termNameFromFilename` extract other segments). New constants `vocabFamilySlug = "vocab-definition"`, `vocabDefinitionPrefix = "vocab-"`, `vocabDefinitionSuffix = "-definition"`. Consumed by Tasks 5, 7.

**Behavior spec:**
- Member term stats (`extractNoteVocabTags`, trigger stats): read `tags:` via `vocabTermsFromTags`. Definition notes (bare `vocab`) are excluded from member/untagged tallies (Task 2's rule, now exercised through the tags reader).
- Term enumeration + descriptions (stats term list, `collectCurrentTermEntries` for refit `--emit-request`): scan vault notes for bare-`vocab`-tagged notes; term name from `termFromDefinitionSlug`; description from the note's `object:` frontmatter field. The family note (slug `vocab-definition`) is skipped by the `(term, false)` return.
- `loadCurrentVocabVersion`: find the family note (bare `vocab` tag + slug `vocab-definition`), read `vocab_version` from frontmatter. Version bumps (`bumpMinorVersion` on propose, `bumpMajorVersion` on refit) rewrite that key in place on the family note. Sentinel error when the family note is missing.
- `loadTermVectors` (the non-centroids fallback): read definition-note sidecars (path from the definition note's filename). `loadAssignmentTermVectors` still prefers `vocab.centroids.json` (unchanged).

**Steps:**

- [ ] **Step 1: Failing tests.** Fixtures: scratch vault with a family note (`210.2026-07-10.vocab-definition.md`, `vocab_version: "6.0"`, tags `[vocab]`), two term definitions (`211....vocab-retrieval-design-definition.md`, `212....vocab-token-budget-definition.md`, tags `[vocab]`, descriptions in `object:`), three member notes with `tags: [vocab/<term>]`. Assert: stats reports 2 terms with correct member counts; emit-request carries both terms + descriptions; `loadCurrentVocabVersion` returns "6.0"; a version bump rewrites only the family note; `termFromDefinitionSlug` table test (family → false, term → term, unrelated slug → false, term containing dashes `vocab-skill-and-guidance-design-definition` → `skill-and-guidance-design`).
- [ ] **Step 2: Verify failure.** `targ test`.
- [ ] **Step 3: Implement** per the behavior spec. Keep `readCentroidsDoc` verdict reading untouched.
- [ ] **Step 4: `targ test` → PASS.**
- [ ] **Step 5: `targ check-full` → green.**
- [ ] **Step 6: Commit:** `feat(vocab): stats/refit/trigger read definition fact notes + tags; vocab_version lives on vocab-definition (#678)`

### Task 5: Definition-note write path — bootstrap/propose/refit mint fact notes, index retired

**Files:**
- Modify: `internal/cli/vocab_commands.go` (`RunVocabBootstrap`, `RunVocabPropose`, `RunVocabRefit`, `renderTermNoteContent` → `renderDefinitionNoteContent`, `writeAndEmbedTermNote` → `writeAndEmbedDefinitionNote`, delete `regenVocabIndex`/`renderVocabIndexContent`/`vocabIndexFrontmatterDoc`, refit removals/renames operate on definition notes + member tags)
- Test: `internal/cli/vocab_commands_test.go` (bootstrap/propose/refit suites)

**Interfaces:**
- Consumes: Task 4's naming helpers + version home; Task 1's writer for member rewrites; the flock-guarded luhmann sequencing learn uses (`writeLearnUnderLock`, learn.go:647, and `nextLuhmannID`, luhmann.go:126 — reuse, do not duplicate).
- Produces: definition notes as ordinary fact notes. Concrete shape (the source of the description + exemplars is today's `vocab.<term>.md`: its `description:` frontmatter key duplicates the body's first paragraph, and the body's `Exemplars:` bullet section carries the exemplar list — carry that section over verbatim):

```markdown
---
type: fact
tier: L2
situation: recalling what the retrieval-design vocab term covers, or assigning vocab terms
subject: the retrieval-design vocab term
predicate: covers
object: 'Design principles for memory retrieval — ranking, filtering, payload shaping, ...'
luhmann: "211"
created: "2026-07-10"
source: 'vocab lifecycle (refit v7.0)'   # or 'migrated from vocab.retrieval-design.md under #678' in Task 7
tags:
    - vocab
---

Information learned: when in recalling what the retrieval-design vocab term covers, or assigning vocab
terms, the retrieval-design vocab term covers Design principles for memory retrieval — ranking,
filtering, payload shaping, ...

Exemplars:
- judging whether engram's verified memory wins hold at scale, ...
- (remaining bullets verbatim from the old term note)
```

  Key order above IS the `factFrontmatterDoc` render order (object before luhmann/created/source — matches the real 204 note); `renderDefinitionNoteContent` marshals through `factFrontmatterDoc` so keys survive amend re-render without reshuffling. The body is the embedding text, as today. Family note minted by bootstrap when absent (subject `the vocab tag family`, `object` documents the convention WITHOUT enumerating terms — see the invariant test in Step 1 — plus `vocab_version` in frontmatter).

**Steps:**

- [ ] **Step 1: Failing tests.** Rewrite the bootstrap/propose/refit suites: bootstrap mints N definition notes + the family note (idempotent — second run mints nothing), no `vocab.index.md` anywhere; propose mints one definition + minor-bumps the family note; refit removals delete the definition note + its sidecar and clear `vocab/<term>` from member tags; renames are a concrete file rename preserving the luhmann ID and date — given `<id>.<date>.vocab-<old>-definition.md`, write `<id>.<date>.vocab-<new>-definition.md` with the content's slug-bearing fields updated, and **re-embed**: the body text embeds the term name, so ContentHash changes — the new sidecar is REGENERATED from the updated body via `writeAndEmbedDefinitionNote`, never renamed (a renamed sidecar carries a stale vector + hash); delete the old `.md` + `.vec.json`; the rename test asserts sidecar freshness (sidecar ContentHash == `embed.ContentHash` of the new file bytes) — and substitute `vocab/<old>` → `vocab/<new>` in member tags; refit major-bumps the family note; `retagAllNotesTwoPass` still re-assigns members via `WriteVocabAssignment` (tags now — assert one member's tags list).

  Two invariant tests this suite MUST also carry (Gate A ask-alignment findings):

```go
func TestVocabFamilyNote_NeverEnumeratesTerms(t *testing.T) {
	// bootstrap a scratch vault with terms {"retrieval-design", "token-budget"};
	// read the minted vocab-definition family note; assert its full content
	// contains NEITHER term name. A maintained term list in the family note is
	// the stale-index problem reborn (issue #678's most explicit warning).
}

func TestComputeTermCentroids_ExcludesDefinitionVectors(t *testing.T) {
	// scratch vault: 2 members assigned to term X with known vectors, plus the
	// term-X definition note with a wildly different known vector. Run the
	// bootstrap/refit centroid pass; assert term X's centroid equals the mean
	// of the TWO member vectors exactly (definition vector absent from the mean).
}
```

  (Write both as full tests against the real deps shapes; the comments above are the contracts.)
- [ ] **Step 2: Verify failure.** `targ test`.
- [ ] **Step 3: Implement.** Delete index machinery outright (`vocabIndexFilename` constant survives only in Task 7's migration reader, so move it there or inline the literal there).
- [ ] **Step 4: `targ test` → PASS.**
- [ ] **Step 5: `targ check-full` → green.**
- [ ] **Step 6: Commit:** `feat(vocab): bootstrap/propose/refit mint parent-tagged definition fact notes; index retired (#678)`

### Task 6: Delete vocab types, query exclusions, and legacy struct fields

**Files:**
- Modify: `internal/cli/vocab.go` (delete `isVocabKind`, `typeVocab`, `typeVocabIndex`), `internal/cli/qa.go` (`isQueryExcludedKind` :254-256 keeps ONLY the qa-question exclusion), `internal/cli/query.go` (seam comments at :435, :844-846, :1083-1084), `internal/cli/learn.go` (delete `Vocab []string` from `factFrontmatterDoc` :153 and `feedbackFrontmatterDoc` :187), `internal/cli/vocab_commands.go` (delete `VocabFrontmatter` struct, `ParseVocabFrontmatter`, `isVocabKindFilename`, `isVocabTermFilename`, `isVocabRewriteExcluded`, `termNameFromFilename`, `termNotePath`, `noteMiniDoc.Vocab` if now unread), `internal/cli/export_test.go` (delete `ExportIsVocabKind = isVocabKind` :57; update the stale vocab-exclusion comment at :575 to describe the inverted recallability tests)
- Test: `internal/cli/vocab_test.go` (delete `TestIsVocabKind_*`, `TestParseVocabFrontmatter_*`, `TestAmendRoundTrip_VocabKey_PreservedAfterField`; REPLACE the two exclusion tests `TestVocabNote_ExcludedFromFloorPromotion` / `TestVocabNote_ExcludedWhenOnlyItem` with their inverses: a bare-vocab-tagged definition note IS floor-promoted / IS returned when it matches), `internal/cli/qa_test.go` (qa-question exclusion still pinned)

**Interfaces:**
- Consumes: everything upstream (this task is the deletion sweep; it must come after Tasks 1-5 so nothing still references the deleted symbols).
- Produces: definitions are ordinary recallable notes — the accepted behavior change, now pinned by tests.

**Steps:**

- [ ] **Step 1: Failing tests.** Write the two inverse recallability tests first (definition note surfaces in the matched set / floor promotion); they FAIL while the exclusion exists.
- [ ] **Step 2: Verify failure.** `targ test`.
- [ ] **Step 3: Implement the deletions.** Compile errors are the checklist — `targ check-full` and fix every reference in one pass (no whack-a-mole).
- [ ] **Step 4: `targ test` → PASS.**
- [ ] **Step 5: `targ check-full` → green.**
- [ ] **Step 6: Commit:** `feat(vocab): definitions are ordinary recallable facts — vocab types and query exclusions deleted (#678)`

### Task 7: `engram vocab migrate-tags` — the one-shot idempotent migration

**Files:**
- Create: `internal/cli/vocab_migrate.go`, `internal/cli/vocab_migrate_test.go`
- Modify: `internal/cli/targets.go` (wire `vocab migrate-tags` subcommand), `internal/cli/vocab.go`/`export_test.go` (test shims as needed)

**Interfaces:**
- Consumes: Task 1's `WriteVocabAssignment` (member rewrite = read old `vocab:` list, call the new writer with those exact terms — assignment preserved, legacy channels stripped); Task 4's naming helpers + version home; Task 5's `writeAndEmbedDefinitionNote` + family-note mint; the flock-guarded luhmann sequencing.
- Produces: `RunVocabMigrateTags(deps, args)` with DI deps mirroring the existing vocab commands' deps pattern.

**Behavior spec (ordered; every step idempotent):**
1. Read `vocab_version` from `vocab.index.md` if it exists (else from the family note; else default `"1.0"` with a printed warning).
2. For every `vocab.<term>.md` term note: if no `vocab-<term>-definition` note exists, mint one (description from `VocabFrontmatter.Description`-equivalent raw parse — the struct is deleted; parse the `description:` and `term:` keys from raw frontmatter here — body exemplars carried over verbatim; `source: migrated from vocab.<term>.md under #678`; tags `[vocab]`; luhmann ID under flock; embed-on-write sidecar).
3. Mint the family note `vocab-definition` if absent (carries the version from step 1).
4. For every non-hub note with a `vocab:` frontmatter key: parse its term list (raw parse of the inline list), call `WriteVocabAssignment(content, thoseTerms)`, write back. Notes without the key but WITH a `Vocab:` body line get `WriteVocabAssignment(content, vocabTermsFromTags(currentTags))` (channel-consistency repair). ContentHash is untouched by construction (frontmatter + machine lines are outside BodyText) — sidecars stay valid.
5. Delete every `vocab.*.md` hub file and every `vocab.*.vec.json` sidecar (this sweeps the orphan `vocab.behavioral-failure-reproduction.vec.json`).
6. Print a counts summary: `members rewritten: N, definitions minted: N, family note: minted|present, hub files deleted: N, sidecars deleted: N`. Second run prints all zeros / `present`.

**Steps:**

- [ ] **Step 1: Failing tests.** Fixture scratch vault reproducing the real shapes — 3 member notes:
  (a) mixed-shape (the real 204 pattern): block-style `tags:\n    - work-kind` PLUS inline `vocab: [lever-tracking, cost-optimization]` PLUS a `Vocab: [[vocab.lever-tracking]], [[vocab.cost-optimization]]` body line — post-migration its tags must read `[work-kind, vocab/lever-tracking, vocab/cost-optimization]` in that order;
  (b) a qa answer note (`qa.` filename prefix, `type: qa-answer`) with `vocab: [retrieval-design]`;
  (c) a plain fact note with `vocab: [retrieval-design]` and a `Vocab:` body line;
  plus (d) a synthetic channel-inconsistency note carrying a `Vocab:` body line but NO `vocab:` key (zero real instances exist — this branch is defensive; the fixture is deliberately synthetic; expected post-state: no vocab tags, no `Vocab:` line — the body-line terms are DISCARDED per behavior-spec step 4, never parsed as assignment);
  2 term notes with exemplars, `vocab.index.md` with `vocab_version: "6.0"`, 1 orphan sidecar. Assert post-state: member tags merged correctly (categorical preserved, vocab/ namespace exact), no `vocab:` keys, no `Vocab:` lines, definitions minted with correct slugs/fields/tags/body exemplars, family note carries "6.0", hubs + sidecars gone, ContentHash of each surviving member unchanged (compute `embed.ContentHash` before/after), counts output exact, second run all-zeros and byte-identical vault.
- [ ] **Step 2: Verify failure.** `targ test`.
- [ ] **Step 3: Implement** with the deps-struct DI pattern of the sibling vocab commands.
- [ ] **Step 4: `targ test` → PASS.**
- [ ] **Step 5: `targ check-full` → green.**
- [ ] **Step 6: Commit:** `feat(vocab): engram vocab migrate-tags — idempotent vault migration to the tags convention (#678)`

### Task 8: Run the real migration (operational; controller-run with exact commands)

**Files:** none in-repo (production vault + installed binary). Execution log records every output.

**Steps:**

- [ ] **Step 0: Trap gate BEFORE** (old binary + old vault): `python3 dev/eval/traps/gate.py --tier smoke`. Expected GREEN (C5b-only RED at n=1 → one same-tree re-run per note 209). Record.
- [ ] **Step 1: Pre-state capture** (evidence for the diff):
```bash
V=/Users/joe/.local/share/engram/vault
grep -l '^vocab:' $V/*.md | wc -l        # expect 187
grep -l '^Vocab:' $V/*.md | wc -l        # expect 187
ls $V/vocab.*.md | wc -l                 # expect 27
ls $V/vocab.*.vec.json | wc -l           # expect 28
ls $V/*.md | wc -l                       # expect 225
ls $V | wc -l                            # expect 452 (225 md + 226 vec.json + centroids.json)
engram count --group-by vocab > /tmp/678-pre-vocab-counts.txt
(cd $V && shasum *.vec.json | sort > /tmp/678-sidecars-pre.txt)
```
- [ ] **Step 2: Backup:** `cp -Rp $V ${V}.bak-2026-07-10-pre678 && ls ${V}.bak-2026-07-10-pre678 | wc -l` — expect exactly **452** (measured 2026-07-10: 225 md + 226 vec.json + vocab.centroids.json). Mismatch with Step 1's live `ls $V | wc -l` = STOP (vault changed since planning; re-measure and reconcile before migrating).
- [ ] **Step 3: Install + run:** `go install ./cmd/engram && cd /tmp && engram vocab migrate-tags` (non-data-dir cwd per vault note 54). Record the counts summary — expected: members rewritten 187, definitions minted 26, family note minted, hub files deleted 27, sidecars deleted 28.
- [ ] **Step 4: Mechanical verification (note 199 — never trust exit codes):**
```bash
V=/Users/joe/.local/share/engram/vault
grep -l '^vocab:' $V/*.md | wc -l                 # 0
grep -l '^Vocab:' $V/*.md | wc -l                 # 0
ls $V/vocab.*.md 2>/dev/null | wc -l              # 0
ls $V/vocab.*.vec.json 2>/dev/null | wc -l        # 0
ls $V/*.md | wc -l                                # 225
grep -l '^    - vocab/' $V/*.md | wc -l           # 187
grep -l '^    - vocab$' $V/*.md | wc -l           # 27
ls $V/*.vocab-*-definition.md | wc -l             # 26
ls $V/*.vocab-definition.md | wc -l               # 1
ls $V | wc -l                                     # 451 (225 md + 225 vec.json + centroids.json)
```
- [ ] **Step 5: Assignment-preservation diff** (byte-level; the representational rewrite re-scored nothing). Output shape measured 2026-07-10: `count --group-by` emits TAB-separated `value<TAB>count` lines plus trailing `(vocab absent): N` and `total: N` summary lines — both summaries must be filtered from BOTH sides or the diff false-STOPs on a pristine migration:
```bash
grep -v '^(vocab absent)' /tmp/678-pre-vocab-counts.txt | grep -v '^total:' | sort > /tmp/678-pre-sorted.txt
engram count --group-by tags | grep '^vocab/' | sed 's|^vocab/||' | sort > /tmp/678-post-vocab-counts.txt
diff /tmp/678-pre-sorted.txt /tmp/678-post-vocab-counts.txt   # expect: empty (identical term<TAB>count pairs)
```
  (The `sed 's|^vocab/||'` strips only the namespace prefix and is tab-safe. The post side has no `(vocab absent)` analogue after the `grep '^vocab/'`; the `total:` line is likewise excluded by it. BAR: identical term→count pairs pre vs post.)
- [ ] **Step 6: Sidecar invariance + re-embed true-up:**
```bash
(cd $V && shasum *.vec.json | sort > /tmp/678-sidecars-post.txt)
diff /tmp/678-sidecars-pre.txt /tmp/678-sidecars-post.txt
# expect: ONLY deletions of vocab.*.vec.json lines (28) and additions of *.vocab-*definition*.vec.json
# lines (27); ZERO changed hashes on any surviving sidecar (ContentHash invariance, verified premise).
engram ingest --auto   # expect: chunk-index true-up only; NO note re-embeds beyond the 27 minted at migration
```
- [ ] **Step 7: Real-binary smoke (from /tmp):**
```bash
engram vocab stats            # output includes the line `verdict: OK` (format measured 2026-07-10); 26 terms listed with member counts
engram query --lazy-chunks --phrase "vocabulary crystallization" | grep tag_nominations_added
                              # budget YAML field `tag_nominations_added: N` with N > 0 (nomination alive on tags)
engram count --group-by type --filter tags=vocab   # expect `fact<TAB>27` + `total: 27` (emergent index: 26 terms + family; format measured 2026-07-10 — count emits TAB-separated value/count lines, NOT colon form)
engram vocab migrate-tags     # idempotency: second run prints all-zero counts / `family note: present`
```
- [ ] **Step 8: Trap gate AFTER** (new binary + migrated vault): `python3 dev/eval/traps/gate.py --tier smoke`. Expected GREEN (note 209 rule applies). Record.
- [ ] **Step 9:** Record everything in the plan's Execution Log. Any expected-count mismatch = STOP, restore from backup, diagnose.

### Task 9: Docs scrub (grep-contract) + ADR annotations + write-memory skill touch

**Files:**
- Modify: `docs/architecture/adr.md` (annotations only — NO new ADR, per the Decision log), `docs/FEATURES.md`, `docs/GLOSSARY.md`, `README.md`, `docs/architecture/c1-system-context.md`, `docs/architecture/c2-containers.md`, `docs/architecture/c3-components.md`, `docs/ROADMAP.md`, `skills/write-memory/SKILL.md`
- Test/verify: the grep contract below; superpowers:writing-skills TDD for the SKILL.md edit; `engram update` to deploy the skill.

**ADR annotations (three, in place):**
- **ADR-0011:** status line gains: "Representation superseded 2026-07-10 (#678, per ADR-0019): the fixed term set now lives as bare-`vocab`-tagged definition fact notes (`vocab-<term>-definition`) and member terms as `tags: [vocab/<term>]` — the centroid-assignment and query-nomination MECHANISM is unchanged." Body references to `vocab.<term>.md` / dual-channel get a `(representation as of 2026-07; see #678)` qualifier where they describe the then-current form.
- **ADR-0018:** the worked example (`vocab.index.md` +1 divergence) gains a dated historical note: "(example vintage 2026-07-08; `vocab.index.md` retired 2026-07-10 under #678 — vocab terms no longer produce wikilink edges, so `--backlinks-of vocab.<term>` reads 0; the divergence relationship itself remains valid for any non-member linker)."
- **ADR-0019:** the Consequences line "Vocab's hub-note channel migrates to this tags convention under #678" becomes "Vocab's hub-note channel migrated to this tags convention 2026-07-10 (#678): definitions are recallable bare-`vocab`-tagged fact notes, `vocab_version` lives on `vocab-definition`, and the vocab query exclusions are deleted." Also reconcile the ADR-0017 reference at adr.md:415 ("vocab migration #678") to past tense.

**GLOSSARY dispositions (beyond wording updates):** the `vocab-index` entry and the `vocab term-note` entry are RETIRED — replaced by entries for `vocab definition note` (bare-vocab-tagged fact note; family + per-term; dash naming) and a pointer from the old names ("retired 2026-07-10, #678 — see vocab definition note"). The `Vocab:` line entry likewise retires to a historical pointer. The query-exclusion text at GLOSSARY lines ~449-451 (isQueryExcludedKind naming `type: vocab-index`; "Q-notes carry no `vocab:` key") updates to the post-#678 reality (only qa-question excluded; Q-notes carry no vocab tags).

**FEATURES:** the §Vocab lifecycle section INCLUDING its heading (line 72: "dual-channel tagging" → tags-based wording) is rewritten; the count-section worked example (165-176) is rewritten generically (divergence = non-member linkers) with the vocab.index example marked historical.

**write-memory SKILL.md target text (lines 79-80), verbatim replacement under writing-skills TDD:**
> `- Never hand-author vocab tags or wikilinks — the binary assigns vocab terms automatically as` `vocab/<term>` `entries in the` `tags:` `list. Handed-off --tag categoricals ride the same list but are NOT vocab: pass them through exactly as provided; never invent tags and never write the` `vocab/` `namespace yourself.`

**The grep contract (vault note 207/186 — the scrub surface is ENUMERATED, not recalled; measured 2026-07-10 pre-scrub):** `grep -rni 'vocab' <file> | grep -viE 'vocab/[a-z]'` hit counts: GLOSSARY.md 41, c2-containers.md 17, c3-components.md 11, README.md 8, FEATURES.md 6, c1-system-context.md 4, write-memory/SKILL.md 2 — 89 hits total; plus `grep -n '678' docs/architecture/adr.md` (lines 415, 495) and `docs/ROADMAP.md` (line 47, the Gated entry). Also surveyed 2026-07-10 (Gate A docs-alignment): `skills/learn/SKILL.md` 8 hits and `skills/recall/SKILL.md` 6 hits — all command-surface (`engram vocab stats`/`refit`) or concept-level ("sharing a vocab term"), still-accurate class; these two files are fenced from edits this cycle (win-nucleus constraint). One borderline: recall/SKILL.md:278 "vocab-tag assignment connects the new note ... structurally" — post-migration the connection is query-time nomination, not graph structure; flag it to the close-out follow-up list alongside the `isMachineLine` cleanup and the migrate-tags retirement decision. The executor re-runs both greps post-scrub and reconciles EVERY remaining hit in the execution log as one of: (a) updated, (b) justified-historical (ADR/decision-log context), or (c) still-accurate (e.g. "sharing a vocab term" in nomination descriptions — the term concept survives; only the representation changed). A hit with no recorded disposition = the task is not done.

**Steps:**

- [ ] **Step 1:** ADR-0011/0018/0019 annotations + adr.md:415 reconcile.
- [ ] **Step 2:** FEATURES/GLOSSARY/README/c1/c2/c3/ROADMAP scrub per the dispositions above (ROADMAP: move #678 out of Gated with a shipped note).
- [ ] **Step 3:** write-memory SKILL.md edit under superpowers:writing-skills TDD (RED: current text asserts the old mechanism wording; GREEN: the verbatim target text above; pressure-check that the "categoricals are NOT vocab" rule still binds).
- [ ] **Step 4:** Re-run the grep contract; record per-file counts + per-hit dispositions in the execution log.
- [ ] **Step 5:** `engram update` (deploy the skill); `targ check-full` → green.
- [ ] **Step 6: Commit:** `docs(vocab): tags-convention scrub — ADR-0011/0018/0019 annotations, GLOSSARY/FEATURES/README/C4, write-memory skill (#678)`

---

## Execution Log

**Tasks 1-7 (code, commits 0a89e052..2ac6c133):** all nine SDD reviews closed-approved. Notable mid-execution decisions: Task 4 shipped a union-read transition (old ∪ new shapes) whose dedup-gap class was closed structurally in Task 5 by the full flip (legacy read branches deleted; tags/definition notes the sole source); Task 6's amend-drops-legacy-key risk accepted with the install+migrate-as-one-step mitigation; Task 7's review reproduced a Critical (hub deletion ungated on mint success) LIVE on a vault copy — fixed with per-term failed-set gating + non-zero exit (2ac6c133), then re-verified live and authorized unconditionally. The Task 7 reviewer's live verification installed the new binary globally mid-cycle; the trap-gate BEFORE baseline was recovered by rebuilding the old binary from main (worktree at 3aa02142).

**Task 8 (real migration, 2026-07-10, binary at 2ac6c133):**
- Trap gate BEFORE (old binary + old vault): **GREEN** all axes first try (C3 5/5, C4i 1/1, C5 1/1, C6 2/2), $≈4 spend — log: session tmp 678-gate-before.log.
- Pre-state: 187/187/27/28/225/452 — matched every pinned bar; backup `vault.bak-2026-07-10-pre678` = 452 entries, listing identical, spot-check hash equal.
- Pre-flight dry-run on fresh copy (reviewer precondition): EXACT counts, zero warnings, exit 0.
- Real run (`engram vocab migrate-tags` from /tmp): `members rewritten: 187, definitions minted: 26, family note: minted, hub files deleted: 27, sidecars deleted: 28`, exit 0.
- Mechanical verification: 0 vocab: keys / 0 Vocab: lines / 0 hub md / 0 hub vec / 225 md / 187 vocab-tagged / 27 bare-vocab / 26 term-definitions / 1 family / 451 entries / 214 tags: keys — **all eleven bars exact**.
- Assignment preservation: per-term member counts identical pre vs post (26 terms, diff empty).
- Sidecar invariance: diff = exactly 28 vocab.* removals + 27 definition additions, ZERO changed survivor hashes (ContentHash invariance held in production).
- Smoke: `vocab stats` verdict OK (26 terms, counts intact); emergent index `fact	27`; `tag_nominations_added: 40` (nomination alive on tags); idempotent second run all-zero/present, exit 0.
- No amend/learn ran between binary install and verification (Task-6 mitigation honored).
- Trap gate AFTER (new binary + migrated vault): **GREEN** all axes first try (C3 5/5, C4i 1/1, C5 1/1, C6 2/2) — log: session tmp 678-gate-after.log. BEFORE/AFTER contrast clean; no C5 re-run needed either side.

**Task 9 (docs scrub, ADR annotations, write-memory skill, 2026-07-10):**

ADR-0011/ADR-0018/ADR-0019 annotations applied per the brief's specs; adr.md:415 ("vocab migration
#678") reconciled to "vocab migration #678 shipped 2026-07-10". `docs/ROADMAP.md`:47's Gated entry
for #678 removed and replaced with an "Also shipped 2026-07-10: #678" paragraph (mirroring #674's
existing shipped entry).

write-memory SKILL.md edit under `superpowers:writing-skills` TDD:
- **RED** (behavior probe against current text): grepped `WriteVocabAssignment`/`removeVocabBodyLine`
  in `internal/cli/vocab.go` — the code actively **strips** any legacy `Vocab:` body line/`vocab:`
  frontmatter key on every write and writes vocab membership as `vocab/<term>` entries into the
  shared `tags:` list; the pre-edit skill text ("the binary assigns vocab automatically" via "vocab
  tags or wikilinks") asserted wikilinks as a still-live channel and never mentioned the `tags:` list
  or explicitly forbade hand-writing a `vocab/` entry into a `--tag` handoff — stale against the
  current mechanism.
- **GREEN**: applied the brief's verbatim replacement text (SKILL.md:79-82). Confirmed byte-identical
  to the brief's target.
- **Pressure test**: dispatched a fresh subagent (no other context) as the write-memory worker, with
  the full post-edit skill text plus an adversarial handoff whose `tags` field includes a
  `vocab/retrieval-design` entry (a parent skill mistakenly proposing a vocab-namespace tag). The
  agent composed the `engram learn fact` command **dropping** `vocab/retrieval-design` and kept only
  the categorical `work-kind/docs` tag, citing the rule "never write the `vocab/` namespace yourself"
  as unconditional (not scoped only to self-invented tags). Verdict: **rule holds** — no loophole
  found, no REFACTOR iteration needed.
- `engram update` run from within the repo clone (`~/repos/personal/engram`) — running it from `/tmp`
  first pulled the remote `main` skill instead of the local branch edit (`engram update`'s local-clone
  detection walks up from cwd; `/tmp` is outside any clone, so it fell back to the remote-install path
  — a real gotcha, corrected by re-running from inside the repo). `diff skills/write-memory/SKILL.md
  ~/.claude/skills/write-memory/SKILL.md` → **identical**.

**Grep contract — pre-scrub counts (pinned, all matched exactly):** GLOSSARY.md 41, c2-containers.md
17, c3-components.md 11, README.md 8, FEATURES.md 6, c1-system-context.md 4, write-memory/SKILL.md 2
(89 total) — plus `grep -n '678'` adr.md:415,495 and ROADMAP.md:47.

**Grep contract — post-scrub dispositions (every remaining hit reconciled; full table also in the
Task 9 report, `.superpowers/sdd/task-9-report.md`):**

| File | Line(s) | Disposition | Note |
|---|---|---|---|
| c1-system-context.md | 93, 203-204, 207 | still-accurate | Trigger-check/nomination mechanism, `vocab.centroids.json`, `engram vocab stats` — unchanged by the migration; no representation claim. |
| c2-containers.md | 42 | updated | "dual-channel" → "vocab-tag assignment ... vocab/<term> entries in the shared tags: list (#678)"; added migrate-tags to the subcommand list. |
| c2-containers.md | 157, 223-224, 236, 240, 246, 249, 253-256 | still-accurate | Mechanism/function names (`AssignVocabTerms`, `WriteVocabAssignment`, `applyVocabAssignmentCore`, `evaluateVocabTriggers`, `hubThreshold`, `RefitPending`) verified still present in code; no stale representation claim. |
| c2-containers.md | 222 | updated | "dual-channel vocab assignment" → "vocab-tag assignment". |
| c2-containers.md | 231 | still-accurate | Section heading, not representation-specific. |
| c2-containers.md | 235 (flowchart node) | updated | `writes body [[vocab.<term>]] wikilinks + frontmatter vocab: list` → `WriteVocabAssignment writes vocab/term entries into the note's shared tags: list (#678)`. |
| c2-containers.md | 250 (flowchart node) | updated | `vocab version bump + vocab.index.md regen` → `vocab version bump on the vocab-definition family note (no index to regenerate — the index is emergent, #678)`. |
| c3-components.md | 7 | still-accurate | Unrelated English word ("vocabulary" in "L3 sequence/flow diagrams"). |
| c3-components.md | 103, 139 | still-accurate | `candidate_l2s`/tag-nomination mechanism description; no representation claim. |
| c3-components.md | 229, 234-235, 257, 259-260, 268, 272 | still-accurate | `applyVocabAssignmentAfterLearn` etc. verified present in code; sequence-diagram notes already phrased generically ("no vocab", "vocab tags assigned") with no stale representation claim. |
| GLOSSARY.md | 3, 667 | still-accurate | Unrelated English word "vocabulary" (section prose/heading). |
| GLOSSARY.md | 151-156 (wikilink entry) | updated | Retired the `Vocab:` member→term wikilink role; two roles remain (prose links, `Supersedes:`); added a dated pointer to the new **vocab definition note** entry. |
| GLOSSARY.md | 183-196 (`vocab.centroids.json lifecycle fields`) | still-accurate | Unchanged — file, fields, and triggers untouched by the migration (plan constraint: centroids file is derived state, not a term registry). |
| GLOSSARY.md | 199-227 (`vocab term-note`, `vocab-index`, `Vocab:` line) | updated | All three RETIRED to dated pointers ("Retired 2026-07-10, #678 — see vocab definition note"); new `### vocab definition note` entry added (family + per-term shapes, dash naming, tags:[vocab], no query exclusion) verified against the real vault (`236.2026-07-10.vocab-definition.md`, 26 `*-definition.md` files). |
| GLOSSARY.md | 214-219→227-233 (`vocab nomination`) | still-accurate + noted | Mechanism unchanged; added one sentence noting the representation move (frontmatter → tags) for completeness. |
| GLOSSARY.md | 320/325→334/339 (`candidate_l2s` entry) | still-accurate | Nomination-channel description; no representation claim. |
| GLOSSARY.md | 393→407-408 (`--supersedes` entry) | updated | "structural linking is done automatically by the binary's vocab-tag assigner" → explicit: rides `tags: [vocab/<term>]`, nominated at query time, not authored wikilinks. |
| GLOSSARY.md | 400→415-418 (`--tag` entry) | updated | "Distinct from the binary-assigned `vocab:` channel" (now false — vocab shares the same `tags:` list) → explicit `vocab/` namespace-sharing rule with the same "never hand-author" prohibition. |
| GLOSSARY.md | 437-438→455-456, 449-451→468-470 (`engram learn qa`, `qa-question`) | updated | "no `vocab:` key" → "no `vocab/` tag"; the FALSE claim "excluded ... same exclusion as `vocab` and `vocab-index`" corrected to name `isQueryExcludedKind` as the sole remaining (qa-question-only) exclusion since #678 retired the vocab-kind exclusion (Task 6 review carry-forward item, explicitly required by the brief). |
| GLOSSARY.md | 453-454, 469→485 (`qa-answer`/machine-line comparisons) | updated | Dropped the stale "(same pattern as `Vocab:` / `Supersedes:`)" comparison (Vocab: retired) — now compares to `Supersedes:` alone; "`vocab:` tags" → "`vocab/` tags". |
| GLOSSARY.md | 612→632 (`engram count` worked example) | updated | `vocab.index.md`-specific example generalized to "a hand-authored MOC/hub page that links every note on a topic without frontmatter-listing them"; `vocab.index.md` named as a retired machine-generated instance of the pattern. |
| README.md | 14 | updated | Vault-graph screenshot description marked historical (predates #678); vocab-hub visual claim qualified as no-longer-current. |
| README.md | 44, 45, 83 | still-accurate | Skill-table / `engram query` mechanism descriptions; no representation claim. |
| README.md | 94, 95, 97 | updated | Bootstrap/propose/refit command descriptions rewritten for the real behavior (definition notes minted, no index regen, `tags:` rewritten) per the brief's explicit instruction. |
| README.md | 96 | still-accurate | `engram vocab stats` description — unaffected. |
| README.md | 98 (new) | added | `engram vocab migrate-tags` was undocumented in README; added for completeness (the subcommand is live and wired — deviation noted in the Task 9 report). |
| FEATURES.md | 72, 74-76 | updated | Heading ("dual-channel tagging" → "tags-based term assignment"); body names the 2026-07-10 migration and the `tags: [vocab/<term>]` representation. |
| FEATURES.md | 81→82 | still-accurate | `dev/eval/LEDGER.md` validation anchors — historical measurement references, unaffected by representation change. |
| FEATURES.md | 165-166→(generalized) | updated | Count-section worked example generalized (divergence = non-member linkers), `vocab.index.md` example dropped in favor of a hand-authored-MOC example. |
| FEATURES.md | 172-176→176-179 | updated | vocab-stats parity validation marked "historical (pre-#678, measured 2026-07-08)"; noted `--backlinks-of vocab.<term>` now reads 0 and that the divergence PROPERTY itself is proven by the two still-live unit tests, not this stale example. |
| write-memory/SKILL.md | 79-82 | updated | Verbatim brief replacement, under writing-skills TDD (RED/GREEN/pressure above). |

**ADDITIONS-from-review items closed:** GLOSSARY:449 area FALSE exclusion claim — corrected (see
`qa-question` row above). ROADMAP #678 Gated entry — moved to shipped (see ROADMAP paragraph above).
adr.md:415 past-tense reconcile — done (see ADR annotations paragraph above).

`targ check-full`: green (7/8 targets PASS; `check-uncommitted` FAILs only because Task 9's own edits
are, at time of this log entry, not yet committed — resolved by Step 6's commit).
