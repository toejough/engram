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
| Notes with `vocab:` frontmatter key | 187 (includes 5 `qa.*.md` answer notes and the 3 #674 definition notes 204/205/206) |
| Notes with `Vocab:` body line | 187 |
| Hub files `vocab.*.md` | 27 (26 term notes + `vocab.index.md`) |
| Hub sidecars `vocab.*.vec.json` | 28 (27 + 1 orphan: `vocab.behavioral-failure-reproduction.vec.json`) |
| Notes with `tags:` key | 3 (the #674 definition notes; block style, 4-space indent) |
| Current `vocab_version` (from `vocab.index.md`) | "6.0" |
| Term count | 26, all kebab-case (all valid as `vocab/<term>` tags) |
| Highest luhmann ID | 209 |
| ContentHash invariance | `ContentHash` hashes `BodyText`, which strips frontmatter AND `Vocab:` machine lines (internal/embed/hash.go:41-84) — the member rewrite changes NO ContentHash; zero re-embeds expected for surviving notes |

Expected post-migration: `^vocab:` count 0; `^Vocab:` count 0; `vocab.*.md` 0; `vocab.*.vec.json` 0; total `.md` = 225 (198 survivors + 27 minted definitions); `tags:` keys = 214 (187 members + 27 definitions); notes with a `vocab/` tag = 187; notes with bare `vocab` tag = 27; `*.vocab-*-definition.md` = 26; `*.vocab-definition.md` = 1; luhmann IDs 210–236 consumed.

## Decision log

- Dashes-everywhere naming: considered the issue's dot form (`vocab.<term>.definition`); Joe chose dashes 2026-07-10 to match shipped #674 notes and kebab slug/tag grammar.
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
	frontmatter, rest, ok := splitFrontmatter(content)
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

	return fmStart + frontmatter + fmEnd[1:] + removeVocabBodyLine(rest)
}
```

with helpers (exact implementations are the implementer's to write against the tests; required signatures and semantics):

```go
// splitFrontmatter cuts content into (frontmatter-without-delims, body-after-closing-delim, ok).
func splitFrontmatter(content string) (string, string, bool)

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
- Modify: `internal/cli/vocab.go` (`applyVocabAssignmentCore`), `internal/cli/vocab_commands.go` (member-scan sites currently using `isVocabRewriteExcluded` / filename checks: `assignTermsToAllNotes`, `collectVaultStats`, `clearRemovedTermsFromMembers`), `internal/cli/vocab_centroids.go` (`retagAllNotesTwoPass`), `internal/cli/vocab_trigger.go` (`collectTriggerVaultStats*`)
- Test: `internal/cli/vocab_test.go`, `internal/cli/vocab_commands_test.go`, `internal/cli/vocab_trigger_test.go`

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
	// a bare-vocab-tagged note runs through the core with valid term vectors
	// and a valid sidecar; assert write is NEVER called (definition notes are
	// exempt from term assignment — they must never acquire their own term tag).
}
```

Plus: trigger/stats member scans treat bare-vocab-tagged notes as neither members nor untagged (fixture with 1 definition + 2 members, one tagged one not → member_count 2, untagged 1); `retagAllNotesTwoPass` and `assignTermsToAllNotes` skip definition notes (fixture assertion: definition file bytes unchanged after a full retag).

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
- Produces: `termFromDefinitionSlug(slug string) (string, bool)` — parses `vocab-<term>-definition` → `<term>`; returns false for `vocab-definition` (the family note) and non-matching slugs. `slugFromNoteFilename(name string) string` — extracts `<slug>` from `<id>.<date>.<slug>.md` (may already exist; reuse if so). New constants `vocabFamilySlug = "vocab-definition"`, `vocabDefinitionPrefix = "vocab-"`, `vocabDefinitionSuffix = "-definition"`. Consumed by Tasks 5, 7.

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
- Consumes: Task 4's naming helpers + version home; Task 1's writer for member rewrites; the flock-guarded luhmann sequencing learn uses (`writeLearnUnderLock` and the next-ID helper in `internal/cli/learn.go` — reuse, do not duplicate).
- Produces: definition notes as ordinary fact notes: frontmatter `type: fact`, `situation: recalling what the <term> vocab term covers, or assigning vocab terms`, `subject: the <term> vocab term`, `predicate: covers`, `object: <description>`, luhmann ID, created, `source: vocab lifecycle (<bootstrap|propose|refit> v<version>)`, `tags:\n    - vocab`; body = the standard fact body sentence + optional `Exemplars:` bullet list (the body is the embedding text, as today). Family note minted by bootstrap when absent (subject `the vocab tag family`, `object` documents the convention WITHOUT enumerating terms, carries `vocab_version`).

**Steps:**

- [ ] **Step 1: Failing tests.** Rewrite the bootstrap/propose/refit suites: bootstrap mints N definition notes + the family note (idempotent — second run mints nothing), no `vocab.index.md` anywhere; propose mints one definition + minor-bumps the family note; refit removals delete the definition note + its sidecar and clear `vocab/<term>` from member tags; renames rewrite the definition slug (new note file name via the normal rename-as-new-luhmann? NO — rename rewrites the existing file's slug portion and its sidecar name, preserving luhmann ID) and substitute `vocab/<old>` → `vocab/<new>` in member tags; refit major-bumps the family note; `retagAllNotesTwoPass` still re-assigns members via `WriteVocabAssignment` (tags now — assert one member's tags list).
- [ ] **Step 2: Verify failure.** `targ test`.
- [ ] **Step 3: Implement.** Delete index machinery outright (`vocabIndexFilename` constant survives only in Task 7's migration reader, so move it there or inline the literal there).
- [ ] **Step 4: `targ test` → PASS.**
- [ ] **Step 5: `targ check-full` → green.**
- [ ] **Step 6: Commit:** `feat(vocab): bootstrap/propose/refit mint parent-tagged definition fact notes; index retired (#678)`

### Task 6: Delete vocab types, query exclusions, and legacy struct fields

**Files:**
- Modify: `internal/cli/vocab.go` (delete `isVocabKind`, `typeVocab`, `typeVocabIndex`), `internal/cli/qa.go` (`isQueryExcludedKind` :254-256 keeps ONLY the qa-question exclusion), `internal/cli/query.go` (seam comments at :435, :844-846, :1083-1084), `internal/cli/learn.go` (delete `Vocab []string` from `factFrontmatterDoc` :153 and `feedbackFrontmatterDoc` :187), `internal/cli/vocab_commands.go` (delete `VocabFrontmatter` struct, `ParseVocabFrontmatter`, `isVocabKindFilename`, `isVocabTermFilename`, `isVocabRewriteExcluded`, `termNameFromFilename`, `termNotePath`, `noteMiniDoc.Vocab` if now unread)
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

- [ ] **Step 1: Failing tests.** Fixture scratch vault reproducing the real shapes: 3 member notes (one with existing categorical tags — the 204-shape; one qa answer note; one plain), 2 term notes with exemplars, `vocab.index.md` with `vocab_version: "6.0"`, 1 orphan sidecar. Assert post-state: member tags merged correctly (categorical preserved, vocab/ namespace exact), no `vocab:` keys, no `Vocab:` lines, definitions minted with correct slugs/fields/tags/body exemplars, family note carries "6.0", hubs + sidecars gone, ContentHash of each surviving member unchanged (compute `embed.ContentHash` before/after), counts output exact, second run all-zeros and byte-identical vault.
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
engram count --group-by vocab > /tmp/678-pre-vocab-counts.txt
```
- [ ] **Step 2: Backup:** `cp -Rp $V ${V}.bak-2026-07-10-pre678 && ls ${V}.bak-2026-07-10-pre678 | wc -l` (expect 481 = 225 md + 253 vec.json + vocab.centroids.json + 2 extra hub sidecar entries; pin the exact number from the live `ls | wc -l` before migrating).
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
```
- [ ] **Step 5: Assignment-preservation diff:** `engram count --group-by tags | grep '^vocab/'` vs `/tmp/678-pre-vocab-counts.txt` — every term's member count identical (the representational rewrite re-scored nothing).
- [ ] **Step 6: Re-embed true-up:** `engram ingest --auto` — expect ~27 new embeddings (definitions), ~0 re-embeds of surviving notes (ContentHash invariance). Record the tally.
- [ ] **Step 7: Real-binary smoke (from /tmp):** `engram vocab stats` (verdict line present; 26 terms; member counts sane); `engram query --lazy-chunks --phrase "vocabulary crystallization"` (payload well-formed; `tag_nominations_added` > 0); `engram count --group-by tags | grep -c '^vocab$'` (the bare-vocab row = 27); idempotency: `engram vocab migrate-tags` second run prints all zeros.
- [ ] **Step 8: Trap gate AFTER** (new binary + migrated vault): `python3 dev/eval/traps/gate.py --tier smoke`. Expected GREEN (note 209 rule applies). Record.
- [ ] **Step 9:** Record everything in the plan's Execution Log. Any expected-count mismatch = STOP, restore from backup, diagnose.

### Task 9: Docs scrub + ADR-0020 + write-memory skill touch

**Files:**
- Modify: `docs/architecture/adr.md` (new ADR-0020; annotate ADR-0011 status; annotate ADR-0018's vocab.index example as historical), `docs/FEATURES.md` (§Vocab lifecycle 72-81; §count 165-166/175-176), `docs/GLOSSARY.md` (`Vocab:` line 151-152/221-226, vocab section 199-218, vocab-index, term-note, 319-332/393 assigner wording), `README.md` (17, 44-45, 94-97), `docs/architecture/c2-containers.md` (42, 157, 222-256 flowcharts), `docs/architecture/c1-system-context.md` (92-93, 203-207), `docs/ROADMAP.md` (#678 out of Gated → shipped note), `skills/write-memory/SKILL.md` (79-80: "the binary assigns vocab tags (`vocab/<term>` in `tags:`) automatically — handed-off `--tag` categoricals ride the same list")
- Test/verify: grep procedures below; superpowers:writing-skills TDD for the SKILL.md edit; `engram update` to deploy the skill.

**ADR-0020 content (draft to refine in place):** "Vocab rides the tags convention — definitions are recallable fact notes." Status accepted 2026-07-10, shipped via #678. Context: ADR-0011's mechanism (centroid assignment + query nomination) kept; its representation (custom key + wikilinks + special-type hub files) retired per ADR-0019. Decision: `tags: [vocab/<term>]` members; bare-`vocab` definition notes (`vocab-definition` family holds `vocab_version`; `vocab-<term>-definition` per term, dash naming per Joe 2026-07-10); emergent index; no query exclusion. Consequences: definitions recallable (behavior change); `count --backlinks-of vocab.<term>` = 0 (vocab edges were the wikilinks — historical ADR-0018 example annotated); migration was assignment-preserving; `vocab.centroids.json` unchanged as derived state; `isMachineLine` Vocab-stripping kept for restored-backup hash stability.

**Grep procedures (EXECUTED against the post-scrub tree; each hit is either updated or justified-historical in the execution log — vault note 207: these were composed from the measured surface map, and the executor runs them, records counts, and reconciles every hit):**
```bash
grep -rn 'vocab\.index\|vocab-index' docs README.md skills   # expect: ADR historical mentions only
grep -rn 'Vocab: \[\[' docs README.md skills                 # expect: ADR/GLOSSARY historical mentions only
grep -rn 'vocab\.<term>\|vocab\.\*\.md' docs README.md skills # expect: ADR historical + migration-note mentions only
grep -rn 'dual-channel' docs README.md skills                # expect: 0 current-mechanism uses (historical ADR context allowed)
grep -rn 'vocab:' docs/GLOSSARY.md docs/FEATURES.md README.md # expect: 0 current-mechanism uses of the retired key
```

**Steps:**

- [ ] **Step 1:** ADR-0020 + ADR-0011/0018 annotations.
- [ ] **Step 2:** FEATURES/GLOSSARY/README/c1/c2/ROADMAP scrub per the pinned line spans.
- [ ] **Step 3:** write-memory SKILL.md edit under superpowers:writing-skills TDD (RED: current text asserts the old mechanism wording; GREEN: new wording; pressure-check the "categoricals are NOT vocab" rule still holds — it does: handed-off tags and auto-assigned vocab/ tags now share the list, and the rule becomes "never hand-author vocab/ tags").
- [ ] **Step 4:** Run the grep procedures; record counts + per-hit disposition in the execution log.
- [ ] **Step 5:** `engram update` (deploy the skill); `targ check-full` → green.
- [ ] **Step 6: Commit:** `docs(vocab): tags-convention scrub — ADR-0020, FEATURES/GLOSSARY/README/C4, write-memory skill (#678)`

---

## Execution Log

(filled during execution — gate verdicts, migration outputs, grep dispositions, deviations)
