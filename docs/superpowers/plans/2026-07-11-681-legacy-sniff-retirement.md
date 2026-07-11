# #681 Legacy-Sniff Retirement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete every legacy-shape tolerance kept after #678's vocab→tags migration — the `migrate-tags` command, filename sniffs, legacy `type:` sniffs, the `Vocab:`/`vocab:` writer+hash machinery, the expired `Related to:` hash exclusion — plus the stale recall-skill wording and one stale test message; the engram repo's git history is the fallback (Joe, 2026-07-11).

**Architecture:** Pure deletion sweep over `internal/cli` and `internal/embed`, ordered so the tree compiles at every commit: the migration command first (it is the heaviest caller of the sniffs), then the sniffs, then the legacy type constants, then the writer/hash channels, then the expired Related-to exclusion, then the skill line (writing-skills TDD) and the doc scrub. Trap gate smoke runs BEFORE (pre-change tree) and AFTER (final tree).

**Tech Stack:** Go (imptest + rapid + gomega tests), targ build system, python trap gate (`dev/eval/traps/gate.py`).

## Global Constraints

- Use `targ` for all test/lint/check operations — NEVER `go test` or `go vet` directly. Tests: `targ test`. Full gate: `targ check-full` (all 8 checks PASS, including lint-full).
- Every commit ends with trailer: `AI-Used: [claude]` (never Co-Authored-By).
- Production vault (`~/.local/share/engram/vault`) is never touched by tests — `t.TempDir()` only.
- Line length ≤ 120 chars. Wrap errors with context. Named constants over magic numbers. `t.Parallel()` on every test/subtest with no shared mutable state.
- nilaway+gomega: after `g.Expect(err).NotTo(HaveOccurred())` add `if err != nil { return }` before use; use `MatchError(...)` not `err.Error()`.
- Diff-scope gate before EVERY commit: `git diff --stat` over the full worktree; anything outside the task's file list is investigated before committing (vault note 150).
- Trap gate: `python3 dev/eval/traps/gate.py --tier smoke` from the repo root. A C5b-honored-only RED at n=1 gets ONE same-tree re-run before being treated as a regression (both runs recorded); RED twice or RED on any other cell is real.
- Skill edits (`skills/recall/SKILL.md`) go through `superpowers:writing-skills` TDD — RED baseline, edit, pressure test. No exceptions.
- Executors re-locate every edit by the verbatim code anchor given, not by line number — deletions in earlier tasks shift later line numbers.
- **Scope disclosures (flagged for Gate A):** two items extend the issue's literal bundle, both discovered by live-tree re-derivation and both the same class (dead tolerance on a fully-migrated vault):
  1. `WriteVocabAssignment`'s legacy-channel strip (`removeVocabBodyLine` + the `vocab:` key removal) — forced by deleting `embed.VocabBodyMarker`, which `vocab.go` aliases.
  2. `RelatedSectionMarker` + `stripRelatedToSection` (hash.go) — its own comment states "then this can go" once the vocab migration lands; the migration landed 2026-07-10.
- **Measured preconditions** (commands run 2026-07-11 against the live tree + real vault; executors re-verify at execution time):
  - Real vault: 233 .md notes; `grep -l '^Vocab:' *.md | wc -l` → 0; `grep -l '^vocab:' *.md | wc -l` → 0; `grep -l '^Related to:' *.md | wc -l` → 0; only `vocab.*` file is `vocab.centroids.json` (live store). ⇒ removing the exclusions changes no BodyText/ContentHash for any real note; zero re-embeds expected.
  - `typeVocab` has 7 non-test usages (live bare-vocab tag marker) — it is KEPT (renamed), per the issue's re-justify clause. `typeVocabIndex` and the `doc.Type` sniff are legacy-only.
  - `noteMiniDoc` has 3 usages, all in `vocab_commands.go`, all in the legacy type-sniff path — dies with it.
  - `ListVecJSON` is migrate-tags-only (interface field `vocab_commands.go`, wiring `newOsVocabDeps`, impl `vault_fs.go`).
  - "both channels" stale message: `internal/cli/vocab_test.go:239` ONLY. `internal/cli/supersedes_test.go:72` says "both channels" about the LIVE supersedes forward+inverse channels — do not touch.
  - Doc files containing retired symbols (grep 2026-07-11): `README.md`, `docs/GLOSSARY.md`, `docs/ROADMAP.md`, `docs/FEATURES.md`, `docs/architecture/c2-containers.md`, `docs/architecture/adr.md`. Task 7 re-greps at execution time.

---

## Controller pre-flight (before Task 1)

- [ ] Branch: `git checkout -b 681-legacy-sniff-retirement` (from current main).
- [ ] Trap gate BEFORE on the unchanged tree: `python3 dev/eval/traps/gate.py --tier smoke` → expect `overall verdict: GREEN`; save the log to `$CLAUDE_JOB_DIR/tmp/681-gate-before.log`. Apply the C5b re-run rule if it fires.

---

### Task 1: Delete `engram vocab migrate-tags` and its plumbing

**Files:**
- Delete: `internal/cli/vocab_migrate.go` (548 lines)
- Delete: `internal/cli/vocab_migrate_test.go` (1075 lines)
- Modify: `internal/cli/targets.go` (wiring + comment)
- Modify: `internal/cli/vocab_commands.go` (VocabDeps.ListVecJSON field + wiring + comments)
- Modify: `internal/cli/vault_fs.go` (ListVecJSON method + comments)
- Modify: `internal/cli/vocab_commands_test.go` (any references to migrate-tags symbols — locate by compile errors + `grep -n "MigrateTags\|migrate-tags" internal/cli/vocab_commands_test.go`)

**Interfaces:**
- Consumes: nothing from other tasks (first task).
- Produces: `VocabDeps` without `ListVecJSON`; `vocabTargets` without the migrate-tags target. Later tasks assume `vocab_migrate.go` is gone.

- [ ] **Step 1: RED analogue** — this is a deletion; the "failing test" is the compile break. Run `git rm internal/cli/vocab_migrate.go internal/cli/vocab_migrate_test.go`, then `targ test` → expect compile FAILURES naming the dangling references (targets.go wiring, ListVecJSON, test references). Every failure named by the compiler is the checklist for Step 2.

- [ ] **Step 2: Remove the wiring.** In `internal/cli/targets.go`, delete this block (anchor verbatim):

```go
			targ.Targ(func(ctx context.Context, a VocabMigrateArgs) {
				a.Vault = resolveVault(a.Vault, home, os.Getenv)
				errHandler(RunVocabMigrateTags(withLog(ctx), a, newOsVocabDeps(), stdout))
			}).Name("migrate-tags").Description(
				"One-shot idempotent migration: legacy vocab:/Vocab:/hub-file representation → tags: convention (#678)"),
```

and change the `vocabTargets` doc comment `// propose, refit, migrate-tags).` → `// propose, refit).`

- [ ] **Step 3: Remove ListVecJSON.** In `internal/cli/vocab_commands.go`: delete the `ListVecJSON func(vault string) ([]string, error)` field and its comment block (anchor: `// ListVecJSON returns the .vec.json filenames in vault — used by`), and the `ListVecJSON:  (&osVaultFS{}).ListVecJSON,` wiring line in `newOsVocabDeps`. In `internal/cli/vault_fs.go`: delete the `ListVecJSON` method and its comment (anchor: `func (*osVaultFS) ListVecJSON(dir string) ([]string, error)`), and reword the `listDirBySuffix` comment that says it is "Shared by ListMD and ListVecJSON — the only difference between the two is the suffix filtered." to describe ListMD only (or inline the suffix if the linter flags a single-use helper — implementer's choice, minimal).

- [ ] **Step 4: Comment scrub in place.** In `vocab_commands.go`, reword the three comments that present migrate-tags as a live caller (anchors: `// already present — bootstrap's (and migrate-tags') idempotency requirement.`, `// after an attempt let migrate-tags' counts summary report "minted" for a`, `// findVocabFamilyNote, e.g. RunVocabMigrateTags's familyOK gate).`) to bootstrap-only phrasing. Historical mentions ("retired in #681, git log recovers it") are acceptable where the history aids the reader; presenting the command as current is not.

- [ ] **Step 5: Fix test references.** `grep -n "MigrateTags\|migrate-tags\|ListVecJSON" internal/cli/*_test.go` — delete or rewire every hit (tests OF the migration die with it; tests that merely wired `ListVecJSON: nil` in a deps literal drop the field).

- [ ] **Step 6: Verify.** `targ test` → PASS. `targ check-full` → all 8 PASS. `git diff --stat` scope check.

- [ ] **Step 7: Commit.** `git add -A && git commit` — message: `feat(vocab): retire the migrate-tags command — migration landed, git log recovers it (#681)` + trailer.

---

### Task 2: Delete the legacy filename sniffs and simplify their call sites

**Files:**
- Modify: `internal/cli/vocab_commands.go` (helpers + consts + 4 call sites)
- Modify: `internal/cli/vocab_centroids.go` (firstTermSidecarMeta + loadMemberNoteVectors)
- Modify: `internal/cli/vocab_trigger.go` (scanNonVocabNotes)
- Modify: `internal/cli/vocab_commands_test.go` (fixtures/tests exercising old-shape skips)

**Interfaces:**
- Consumes: Task 1 (vocab_migrate.go gone — it was the heaviest caller of these helpers).
- Produces: `isVocabKindFilename`, `isVocabTermFilename`, `isVocabRewriteExcluded`, `vocabIndexFilename`, `vocabNotePrefix` no longer exist. `isQAQuestionFilename` and `isVocabDefinitionNote`/`definitionNoteTerm` remain the only exclusion primitives.

- [ ] **Step 1: RED** — write (or adapt) a test proving the LIVE exclusions survive the sniff removal: a bare-vocab definition note and a `qa.*.q.md` question note are still excluded from member scans, and an ordinary note whose filename happens to start with `vocab.` (no such file exists in a migrated vault, but the name must no longer be special) IS scanned. Name: `TestMemberScan_FilenamePrefixNoLongerSpecial` in `vocab_commands_test.go`. It FAILS before the edit (the prefix skip excludes the note).

- [ ] **Step 2: Delete the helpers + consts** in `vocab_commands.go`: the const block entries `vocabIndexFilename = "vocab.index.md"` and `vocabNotePrefix = "vocab."` (with their comments), and the three functions (anchors: `func isVocabKindFilename(name string) bool`, `func isVocabRewriteExcluded(name string) bool`, `func isVocabTermFilename(name string) bool`) plus the stale cross-reference comment above them (anchor: `// vocab_centroids.go's old-shape sidecar-metadata scan (firstTermSidecarMeta).`).

- [ ] **Step 3: Simplify call sites.** Exact edits (anchors verbatim from the live tree):

  a. Member-count loop (`vocab_commands.go`, anchor `if isVocabKindFilename(name) {` followed by a separate `isQAQuestionFilename` skip): delete the `isVocabKindFilename` skip; keep the QA skip. (The definition-note exemption lives inside `assignVocabToNote` → `loadVocabAssignmentBodyVector`.)

  b. `clearRemovedTermsFromNote` loop and `rewriteMemberTermRename` loop (two sites, anchor `if isVocabRewriteExcluded(name) {`): replace with `if isQAQuestionFilename(name) {`.

  c. Stats loop (anchor `if name == vocabIndexFilename {`): delete that skip; keep the QA skip and the `definitionNoteTerm` branch.

  d. `vocab_centroids.go` `firstTermSidecarMeta`: delete the `if name == vocabIndexFilename { continue }` skip and the old-shape union — anchor:

```go
		_, isNewShape := definitionNoteTerm(vault, name, readFile)
		if !isVocabTermFilename(name) && !isNewShape {
			continue
		}
```

  becomes:

```go
		if _, isNewShape := definitionNoteTerm(vault, name, readFile); !isNewShape {
			continue
		}
```

  and rewrite the function's doc comment (anchor: `// Recognizes BOTH the old-shape vocab.<term>.md filename`) to definition-note-only wording.

  e. `vocab_centroids.go` `loadMemberNoteVectors` (anchor `if isVocabKindFilename(name) {` inside the member-vector loop): delete that skip — the `isVocabDefinitionNote(string(content))` check just below is the live exemption.

  f. `vocab_trigger.go` `scanNonVocabNotes` (anchor `if isVocabKindFilename(name) || isQAQuestionFilename(name) {`): drop the first disjunct → `if isQAQuestionFilename(name) {`. NOTE: this scan feeds the refit-trigger note COUNT. The #678 final-review fix (e69b14f3) made seed and check use the same content-based count via this function's callers; the definition-note exclusion there is content-based (`isVocabDefinitionNote`), so dropping the filename disjunct does not change the count on a migrated vault — verify by the test in Step 1 and `targ test`.

- [ ] **Step 4: Rework old-shape tests.** `grep -n "isVocabKindFilename\|isVocabTermFilename\|isVocabRewriteExcluded\|vocabIndexFilename\|vocabNotePrefix\|vocab\.index" internal/cli/*_test.go` — rework each hit: tests OF the sniffs die; tests using `vocab.<term>.md` fixtures to prove exclusion flip to definition-note (bare-vocab tag) fixtures. Known named test: `TestRetagAllNotesTwoPass_LoadMemberNoteVectors_SkipsOldShapeAndUnreadable` → becomes definition-note + unreadable variant.

- [ ] **Step 5: Verify.** `targ test` (Step-1 test now PASSES) → `targ check-full` → diff-scope check.

- [ ] **Step 6: Commit.** `feat(vocab): delete legacy filename sniffs — content-based exclusions only (#681)` + trailer.

---

### Task 3: Delete the legacy `type:` sniff; re-home the bare-vocab tag constant

**Files:**
- Modify: `internal/cli/vocab.go` (constants)
- Modify: `internal/cli/vocab_commands.go` (extractNoteVocabTags + noteMiniDoc + two `Tags: []string{...}` writes)
- Modify: tests referencing `typeVocab` (compile-led)

**Interfaces:**
- Consumes: Tasks 1–2 (their callers gone).
- Produces: constant `vocabDefinitionTag = "vocab"` (renamed from `typeVocab`); `typeVocabIndex` and `noteMiniDoc` no longer exist.

- [ ] **Step 1: RED** — adapt/confirm a test that a post-migration note with frontmatter `type: fact` + `tags: [vocab]` is excluded from term extraction AND that `extractNoteVocabTags` no longer needs a `type:` parse: write `TestExtractNoteVocabTags_TypeFieldIsIrrelevant` — a note with `type: vocab` in frontmatter but ordinary tags IS extracted (returns its `vocab/<term>` tags, ok=true). FAILS before the edit (the sniff returns nil,false).

- [ ] **Step 2: Edit `extractNoteVocabTags`** — delete the `noteMiniDoc` unmarshal block and the sniff (anchor verbatim):

```go
	var doc noteMiniDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return nil, false
	}

	if doc.Type == typeVocab || doc.Type == typeVocabIndex {
		return nil, false
	}
```

  Update the function's doc comment (it already says terms are read "SOLELY from the tags: vocab/<term> namespace" — drop the "is a vocab/vocab-index type note" clause from the failure list). Delete the `noteMiniDoc` struct (anchor: `// noteMiniDoc is used to parse only the type: key`).

- [ ] **Step 3: Rename the constant.** In `vocab.go`: `typeVocab` → `vocabDefinitionTag`; delete `typeVocabIndex` and its comment; replace the `typeVocab` comment block (which narrates the #678 task split and the legacy sniff) with the live story only, e.g.:

```go
	// vocabDefinitionTag is the bare-vocab DEFINITION marker: as a tags: entry
	// it identifies a definition note (isVocabDefinitionNote, nonVocabTags).
	vocabDefinitionTag = "vocab"
```

  Mechanical rename at the 7 non-test usages (`grep -rn "typeVocab" --include="*.go" internal/` and follow the compiler) — includes the two `Tags: []string{typeVocab}` mint sites in `vocab_commands.go` and the reads in `isVocabDefinitionNote`, `nonVocabTags`, `vocabTermsFromTags`'s comment.

- [ ] **Step 4: Verify.** `targ test` → `targ check-full` → diff-scope check.

- [ ] **Step 5: Commit.** `feat(vocab): drop the legacy type: sniff; typeVocab becomes vocabDefinitionTag (#681)` + trailer.

---

### Task 4: Retire the `Vocab:` body-line + `vocab:` key machinery (writer + hash)

**Files:**
- Modify: `internal/cli/vocab.go` (WriteVocabAssignment + removeVocabBodyLine + vocabBodyMarker)
- Modify: `internal/embed/hash.go` (VocabBodyMarker + isMachineLine + comments)
- Modify: `internal/cli/vocab_test.go`, `internal/embed/hash_test.go`, `internal/cli/vocab_commands_test.go` (legacy-channel tests/fixtures)

**Interfaces:**
- Consumes: nothing (independent of T1–T3 symbol-wise; runs after them to keep hash.go churn in one place before T5).
- Produces: `embed.VocabBodyMarker`, `vocabBodyMarker`, `removeVocabBodyLine` no longer exist; `WriteVocabAssignment` no longer parses/strips legacy channels; `Vocab:` is ordinary body text to BodyText/ContentHash.

- [ ] **Step 1: RED** — write `TestWriteVocabAssignment_BodyIsOpaque` in `vocab_test.go`: a note whose body contains a line starting `Vocab: [[vocab.old-term]]` keeps that line byte-identical through `WriteVocabAssignment` (it is user prose now). FAILS before the edit (the writer strips it). Similarly `TestBodyText_VocabLineIsOrdinaryBody` in `hash_test.go`: BodyText of a body containing `Vocab: [[x]]` INCLUDES the line. FAILS before the edit.

- [ ] **Step 2: Writer edit** (`vocab.go`, anchor region inside `WriteVocabAssignment`):

```go
	if yamlKeyLineIndex(frontmatter, "tags") >= 0 {
		frontmatter = removeYAMLKey(frontmatter, "vocab") // may shift tags up; that's fine
		insertAt = yamlKeyLineIndex(frontmatter, "tags")  // recompute on the vocab-free text
		frontmatter = removeYAMLKey(frontmatter, "tags")  // removal at insertAt shifts followers into insertAt
	} else {
		insertAt = yamlKeyLineIndex(frontmatter, "vocab") // -1 when absent → append
		frontmatter = removeYAMLKey(frontmatter, "vocab")
	}
```

  becomes:

```go
	insertAt = yamlKeyLineIndex(frontmatter, "tags") // -1 when absent → append
	frontmatter = removeYAMLKey(frontmatter, "tags") // removal at insertAt shifts followers into insertAt
```

  (When no `tags:` key exists both calls are no-ops and insertAt=-1 appends — same behavior as today's else-branch on a legacy-free note.) Then the return line `return fmStart + frontmatter + fmEnd + removeVocabBodyLine(rest)` → `return fmStart + frontmatter + fmEnd + rest`. Delete `removeVocabBodyLine` (anchor: `// removeVocabBodyLine strips the Vocab: machine line`) and the `vocabBodyMarker` const + comment. Update `WriteVocabAssignment`'s doc comment (anchor: `// their order. It also strips the legacy vocab: frontmatter key and Vocab:`) — it no longer strips legacy channels.

- [ ] **Step 3: Hash edit** (`hash.go`): delete the `VocabBodyMarker` const + comment; drop its clause from `isMachineLine` and its doc comment ("Recognised prefixes: Supersedes:, Contributors:, Answered by:, and Answers:"). Reword the three comments that cite it ("Same exclusion rationale as VocabBodyMarker" on AnsweredBy/Answers → cite SupersedesBodyMarker; the SupersedesBodyMarker comment's "for the same reason as VocabBodyMarker" → state the reason directly: a channel-only write must not stale the sidecar). Update the `BodyText` doc comment's channel enumeration.

- [ ] **Step 4: Test rework.** Delete: `TestWriteVocabAssignment_BlockStyleLegacyVocabRemoved`, `TestWriteVocabAssignment_LegacyVocabAfterTagsKeepsTagsPosition`, `TestWriteVocabAssignment_LegacyVocabBeforeTagsKeepsTagsPosition`, `TestWriteVocabAssignment_StripsLegacyChannels` (vocab_test.go); `TestBodyText_ExcludesVocabLine`, `TestContentHash_IgnoresVocabLineAfterRelatedBlock` (hash_test.go — the latter also touches Related-to; if simpler, convert it in Task 5 instead — implementer's call, disclosed in the report). Reword the stale message at the anchor `"assignment must have written both channels"` → `"assignment must have written the tags"`. In the rapid property test, remove legacy-channel draws (`buildTestFrontmatterWithLegacyVocab`'s legacy `vocab:` key generation and any `Vocab:` body-line draws) — do NOT expand the oracle (that is #683). Update `vocab_commands_test.go` fixtures carrying `Vocab: [[` lines (grep) to tag-only fixtures unless the fixture exists to prove tolerance — those die.

- [ ] **Step 5: Verify.** `targ test` (Step-1 tests PASS) → `targ check-full` → diff-scope check.

- [ ] **Step 6: Commit.** `feat(vocab): retire the Vocab:/vocab: legacy channels — writer and hash treat them as prose (#681)` + trailer.

---

### Task 5: Retire the expired `Related to:` hash exclusion

**Files:**
- Modify: `internal/embed/hash.go` (RelatedSectionMarker, stripRelatedToSection, isRelatedToBlock, BodyText)
- Modify: `internal/embed/hash_test.go`, `internal/embed/hash_property_test.go`, `internal/embed/state_test.go`, `internal/cli/check_test.go`, `internal/cli/resituate_test.go`, `internal/cli/resituate.go` (one comment)

**Interfaces:**
- Consumes: Task 4 (hash.go already reshaped).
- Produces: `RelatedSectionMarker`, `stripRelatedToSection`, `isRelatedToBlock` no longer exist; `BodyText(raw) = normalizeTrailingBlanks(stripMachineLines(ExtractBody(raw)))`.

- [ ] **Step 1: RED** — `TestBodyText_RelatedToIsOrdinaryBody` in `hash_test.go`: BodyText of a body ending in a `Related to:` block INCLUDES the block. FAILS before the edit.

- [ ] **Step 2: Edit.** Delete the `RelatedSectionMarker` const + its "retained for backward compatibility" comment (its own retirement condition — "then this can go" — is met: migration landed 2026-07-10; measured 0 vault notes carry the section). Delete `stripRelatedToSection` and `isRelatedToBlock`. `BodyText` return becomes `normalizeTrailingBlanks(stripMachineLines(ExtractBody(raw)))`; update its doc comment (drop the Related-to narrative and the "Machine lines are stripped BEFORE the Related-to pass" paragraph).

- [ ] **Step 3: Test rework.** Delete/flip: `TestBodyText_ExcludesRelatedToSection`, `TestBodyText_InlineRelatedToProseIsNotStripped`, `TestBodyText_MarkerFollowedByProseIsNotStripped` (now vacuous — delete), `TestContentHash_IgnoresRelatedToLinkEdits` (flips: a Related-to edit now CHANGES the hash — rewrite as `TestContentHash_RelatedToEditsChangeHash`), `TestContentHash_RelatedToInsensitivityProperty` (hash_property_test.go — delete or invert; deletion preferred, the property is gone), `state_test.go`'s "adding a Related to: section must [not re-embed]" cases (flip: it now re-embeds), and `TestContentHash_IgnoresVocabLineAfterRelatedBlock` if deferred from Task 4. `check_test.go`/`resituate_test.go` fixtures keep their Related-to tails ONLY where the tail is opaque payload (resituate preserves any tail); where a test asserts hash insensitivity, flip it. Reword the `relatedTail` doc comment in `resituate.go` (anchor: "followed by `\nRelated to:\n...`") — the tail is any machine-line block (Supersedes:), not Related-to.

- [ ] **Step 4: Verify.** `targ test` → `targ check-full` → diff-scope check.

- [ ] **Step 5: Commit.** `feat(embed): retire the expired Related-to hash exclusion (#681)` + trailer.

---

### Task 6: Fix the stale recall-skill mechanism line (writing-skills TDD)

**Files:**
- Modify: `skills/recall/SKILL.md` (one sentence, Step 4 bullet)

**Interfaces:** none (prose).

**REQUIRED SUB-SKILL:** `superpowers:writing-skills`.

- [ ] **Step 1: RED baseline** — neutral framing (vault note 138: do not spotlight the moment). Headless probe, fresh context, OLD text: give the agent the current Step-4 bullet and ask "a write-memory note was just written with no supersedes link. How will future recall sessions find it alongside related notes? Answer in one sentence." Old text drives "the binary links/connects it structurally into the graph"-class answers (mechanism claim: a standing structural connection). Run 3×; record. RED = ≥2/3 assert a structural/graph link exists after write.
- [ ] **Step 2: GREEN edit.** Anchor (verbatim): `binary's vocab-tag assignment connects the new note to related notes structurally. Do not` — the full sentence currently reads "Otherwise no link ritual is needed; the binary's vocab-tag assignment connects the new note to related notes structurally." Replace with: "Otherwise no link ritual is needed; the binary auto-assigns vocab tags at write time, and recall surfaces tag-sharing notes at query time (tag nomination). Do not". Keep the trailing "hand-author wikilinks for structural linking." clause intact — reword its tail if "structural" now dangles (e.g. "hand-author wikilinks to connect notes").
- [ ] **Step 3: Re-run the probe on the new text** → answers should name query-time nomination, not a standing graph link. Pressure-test per writing-skills (one rationalization probe: "shouldn't you add a wikilink so they're connected?" → the text still forbids it).
- [ ] **Step 4: Deploy.** From inside the repo clone: `engram update` (it deploys skills; running it from elsewhere pulls remote main). Verify `~/.claude/skills/recall/SKILL.md` carries the new sentence.
- [ ] **Step 5: Commit.** `fix(recall-skill): vocab tags nominate at query time, not structurally (#681)` + trailer.

---

### Task 7: Doc scrub — no surviving reference to retired machinery as current

**Files:**
- Modify (grep-enumerated at execution time; measured 2026-07-11 surface): `README.md`, `docs/GLOSSARY.md`, `docs/ROADMAP.md`, `docs/FEATURES.md`, `docs/architecture/c2-containers.md`, `docs/architecture/adr.md`

**Interfaces:** consumes all prior tasks (describes the post-sweep reality).

- [ ] **Step 1: Re-enumerate.** `grep -rn "migrate-tags\|VocabBodyMarker\|RelatedSectionMarker\|isVocabKindFilename\|vocabNotePrefix\|vocab\.index\.md" README.md docs/ skills/ commands/ guidance/` — build a per-hit disposition table (update / keep-as-history / delete) in the task report. The list above is the planning-time snapshot; the live grep governs.
- [ ] **Step 2: Known dispositions** (planning-time, re-verify):
  - `README.md` (anchor: `engram vocab migrate-tags               One-shot idempotent migration`): DELETE the command row — the "Kept as a no-op safety net" framing is the overruled restore-window story. If the command list has a retired-commands note elsewhere, follow its form; otherwise plain removal (retired 2026-07-11, #681; git log recovers it).
  - `docs/architecture/c2-containers.md` (anchor: `vocab` subcommand family `(bootstrap/propose/stats/refit/migrate-tags)`): drop `/migrate-tags`.
  - `docs/GLOSSARY.md` (anchor: `Minted by \`engram vocab bootstrap\`/\`propose\`/\`refit\`/\`migrate-tags\`.`): drop `/`migrate-tags``. Retired-entry sections describing the old representation as retired STAY (they are history, correctly labeled) — but any "kept as safety net" phrasing goes.
  - `docs/ROADMAP.md` (anchor: `the one-shot idempotent \`engram vocab migrate-tags\` subcommand; the real-vault migration verified`): past-tense history — keep, appending "(command retired 2026-07-11, #681)".
  - `docs/FEATURES.md`, `docs/architecture/adr.md`: disposition per hit — ADR entries are decision records (keep, tense-checked); FEATURES rows describing current behavior must drop retired machinery.
- [ ] **Step 3: Verify.** Re-run the Step-1 grep: every remaining hit is in a disposition-table "keep-as-history" row. `targ check-full` (docs don't compile, but the gate includes lint/format checks that may cover markdown) → diff-scope check.
- [ ] **Step 4: Commit.** `docs(vocab): scrub retired legacy-sniff machinery from current-tool references (#681)` + trailer.

---

## Controller close-out (after Task 7)

- [ ] Trap gate AFTER on the final tree: `python3 dev/eval/traps/gate.py --tier smoke` → GREEN; log to `$CLAUDE_JOB_DIR/tmp/681-gate-after.log` (C5b re-run rule applies).
- [ ] Hash-invariance check on the real vault (read-only): `go install ./cmd/engram`, then `engram embed status` → expect ZERO stale sidecars (measured precondition: no real note contains the retired patterns). Any nonzero count STOPs the merge (a hash drifted that the plan said could not).
- [ ] Real-binary CLI check from a non-data-dir cwd: `engram vocab stats` works; `engram vocab migrate-tags` fails with an unknown-target error; `engram query --lazy-chunks --phrase "smoke"` returns a payload.
- [ ] Git-recovery check (note 179): `git log --diff-filter=D --name-only` on the branch lists `internal/cli/vocab_migrate.go` and `internal/cli/vocab_migrate_test.go`.
- [ ] Final whole-branch review (most capable model) with a review package over `git merge-base main HEAD`..HEAD.
- [ ] Gate D → rebase on main → re-run `targ check-full` → ff-only merge → push → close #681 with a closing comment (Gate D covers it).
