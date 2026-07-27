# #713 — Update Notices: Exact Commands, Only When They Work

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `engram update`'s upgrade notices name the exact command inline and appear only when running it would actually do something; the README's Upgrading section (the notices' current destination) is removed.

**Architecture:** Three notice constants in `internal/cli/update.go` change text (drop the README pointer, name the command). The duplicates notice additionally changes *condition*: instead of the manifest-only grouping check (`chunkIndexHasDuplicates`), update reuses prune's own engine (`reconcileDuplicateGroups` in DryRun mode) so the notice fires only when `engram prune --duplicates` would report `removed > 0`. A refusal-only backlog (Joe's reported case) goes silent — refusals are pending-manual-review by design, there is no command to give, and prune/ingest explain refusals when run. Docs that point at the removed section are scrubbed; historical records (ROADMAP shipped rows, old plan docs, LEDGER) stay untouched.

**Tech Stack:** Go, gomega (blackbox `package cli_test` via `export_test.go` shims), targ for test/lint.

**Issue:** https://github.com/toejough/engram/issues/713 (design and alternatives recorded there).

## Global Constraints

- Test/lint ONLY via targ: `targ test`, `targ check-full`. NEVER `go test` / `go vet` directly. `targ check-full` reports all errors at once — collect the full list, fix in one pass.
- TDD every code task: write the failing test, SEE it fail, minimal implementation, SEE it pass, refactor.
- Every test and subtest calls `t.Parallel()`; no shared mutable state between subtests (each case builds its own `u1FS`).
- nilaway+gomega: after `g.Expect(err).NotTo(HaveOccurred())` add `if err != nil { return }` before using dependent values.
- Line length under 120 chars; constants over magic numbers; descriptive names.
- Commit per task. Conventional Commits style matching repo history (`feat(update): …`, `test(update): …`, `docs(readme): …`), reference `(#713)`, and end the message with trailer `AI-Used: [claude]` (NOT Co-Authored-By).
- Verification (Task 5) must be read-only against live data: ONLY `--dry-run` forms, ONLY the scratch-built binary. Never run bare `engram update`, never non-dry-run `prune`, never `go install` (that would overwrite the live binary with unreviewed code).
- Work happens in worktree `.claude/worktrees/713-update-notices` (branch `worktree-713-update-notices`). Never `git stash`/`git stash pop`; read historical file states via `git show <sha>:<path>` only; never `checkout <sha>`.

## Doc-surface disposition list (enumeration grep, run 2026-07-27 over the worktree)

Grep terms: `Upgrading section`, `## Upgrading`, `see the README`, `README's Upgrading`, `migrate-tags`, `regen-vocab`, `prune --duplicates`, `prune --empty`, `old-format vocab`.

| Location | Disposition | Reason |
|---|---|---|
| `README.md:42-50` (`## Upgrading`) | **remove** | The ask: section deleted entirely (Task 4) |
| `README.md:30-40` (Installing) | keep | "Run it again any time to upgrade" is about the command, not the section |
| `internal/cli/update.go:30-36` (3 notice consts) | **update** | Tasks 1–3: new text, no README pointer |
| `internal/cli/update.go:159-174` (`chunkIndexHasDuplicates` doc) | **rewrite** | Task 3: function replaced by coverage-aware check |
| `internal/cli/update.go:366-372` (`writeDuplicatesHint` doc) | **update** | Task 3: names README/Upgrading |
| `internal/cli/update.go:512-514` (`writeVocabMigrationHint` doc) | **update** | Task 1: names README/Upgrading |
| `internal/update/update.go:164-171` (`ChunkIndexHasDuplicates` field) | **update** | Task 3: rename + comment (mentions the notice's README pointer) |
| `internal/cli/update_test.go:39-46,365-397,426-461,463-491,769-786,832-884` | **update** | Tasks 1–3: assertions pin old text/field |
| `internal/cli/export_test.go:48` | **update** | Task 3: shim rename + add `ExportIndexPathFor` |
| `docs/GLOSSARY.md:669-674` (`engram update` entry) | **update** | Task 4: describes notices pointing at README Upgrading |
| `docs/FEATURES.md:199-200` | **update** | Task 4: "(see README's Upgrading section)" |
| `docs/README.md:14` (index row) | **update** | Task 4: names "Upgrading" as a README destination |
| `docs/architecture/adr.md:680-686` (ADR-0021 decision 9) | **update (append)** | Task 4: append a #713 refinement note; never rewrite the historical decision text |
| `docs/ROADMAP.md:196,198` | keep | Historical shipped-record rows; describe what shipped then, accurately |
| `docs/superpowers/plans/2026-07-25-chunk-index-dedup-and-prune-fixes.md` | keep | Historical plan snapshot (quotes old notice verbatim as history) |
| `agent-instructions/skills/learn/SKILL.md:36-40` | keep | Describes `prune --duplicates` accurately; no README pointer |
| `dev/eval/LEDGER.md:76,83` | keep | Measurement records |
| `internal/cli/ingest_dedup.go:506-509` (ingest refusal message) | keep | Ingest's own message, accurate; #713 scopes update prose + README |
| `internal/cli/vocab_regen.go`, `internal/cli/vocab_commands.go:904` | keep | Historical converter references, no README pointers |

---

### Task 1: Vocab migration notice names `engram update --regen-vocab`

**Files:**
- Modify: `internal/cli/update.go:36` (const), `internal/cli/update.go:512-514` (doc comment)
- Test: `internal/cli/update_test.go:769-786` (`TestWriteUpdateReport_VocabMigrationHint`), `:832-884` (`TestWriteUpdateReport_VocabRegenReport`)

**Interfaces:** none new; the const `vocabMigrationNotice` keeps its name.

- [ ] **Step 1: Make the test demand the new text.** In `TestWriteUpdateReport_VocabMigrationHint` (table around line 769-786), the `old-vocab-present` case currently has `wantContains: []string{"Upgrading", "README.md"}`. Change to:

```go
wantContains:    []string{"run `engram update --regen-vocab`"},
wantNotContains: []string{"Upgrading", "README.md"},
```

(keep the `no-old-vocab` case's `wantNotContains: []string{"Upgrading"}` but change it to `[]string{"old-format vocab"}` — the section name is disappearing from all output, so pin the notice's own text instead). In `TestWriteUpdateReport_VocabRegenReport` (`:832-884`), each `wantNotContains: []string{"see the Upgrading section", ...}` becomes `[]string{"old-format vocab files found", ...}` — the real invariant is "regen summary replaces the plain notice".

- [ ] **Step 2: Run and see it fail.** `targ test` → `TestWriteUpdateReport_VocabMigrationHint` FAILS (output still says "see the Upgrading section…").

- [ ] **Step 3: Minimal implementation.** In `internal/cli/update.go`:

```go
vocabMigrationNotice = "old-format vocab files found — run `engram update --regen-vocab` to migrate them " +
	"(preview with `--dry-run`)\n"
```

Update `writeVocabMigrationHint`'s doc comment (:512-514): it now "prints a one-line notice naming `engram update --regen-vocab` when the vault still holds pre-tags vocab files" (drop the README/Upgrading phrasing; keep the silent-otherwise sentence).

- [ ] **Step 4: Run and see it pass.** `targ test` → PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/cli/update.go internal/cli/update_test.go
git commit -m "feat(update): vocab notice names update --regen-vocab, drops README pointer (#713)" -m "AI-Used: [claude]"
```

### Task 2: Empty-files notice drops the README pointer

**Files:**
- Modify: `internal/cli/update.go:32-33` (const), `internal/cli/update.go:379-381` (doc comment)
- Test: `internal/cli/update_test.go:463-491` (`TestWriteUpdateReport_EmptyChunkHint`)

**Interfaces:** none new; const `emptyChunkFilesNotice` keeps its name. Condition is already accurate (`prune --empty` never refuses) — text-only change.

- [ ] **Step 1: Make the test demand the new text.** In `TestWriteUpdateReport_EmptyChunkHint`, replace

```go
g.Expect(buffer.String()).To(ContainSubstring("Upgrading"))
g.Expect(buffer.String()).To(ContainSubstring("README.md"))
```

with

```go
g.Expect(buffer.String()).To(ContainSubstring("run `engram prune --empty`"))
g.Expect(buffer.String()).NotTo(ContainSubstring("Upgrading"))
g.Expect(buffer.String()).NotTo(ContainSubstring("README.md"))
```

- [ ] **Step 2: Run and see it fail.** `targ test` → FAIL.

- [ ] **Step 3: Minimal implementation.**

```go
emptyChunkFilesNotice = "empty chunk-index files found — run `engram prune --empty` to clear them " +
	"(preview with `--dry-run`)\n"
```

Update `writeEmptyChunkHint`'s doc comment (:379-381): drop the README "Upgrading" phrasing ("prints a one-line notice naming `engram prune --empty` when the chunk index still holds 0-byte .jsonl files").

- [ ] **Step 4: Run and see it pass.** `targ test` → PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/cli/update.go internal/cli/update_test.go
git commit -m "feat(update): empty-files notice self-sufficient, drops README pointer (#713)" -m "AI-Used: [claude]"
```

### Task 3: Duplicates notice fires only when prune would actually remove something

**Files:**
- Modify: `internal/cli/update.go` (const :30-31; replace `chunkIndexHasDuplicates` :159-194; `runUpdate` :333; `writeDuplicatesHint` :366-377)
- Modify: `internal/update/update.go:164-171` (rename field `ChunkIndexHasDuplicates` → `ChunkIndexHasPrunableDuplicates`)
- Modify: `internal/cli/export_test.go:48` (rename shim; add `ExportIndexPathFor`)
- Test: `internal/cli/update_test.go:39-132` (`TestChunkIndexHasDuplicates` → `TestChunkIndexHasPrunableDuplicates`), `:426-461` (`TestWriteUpdateReport_DuplicatesHint`)

**Interfaces:**
- Consumes (already exist, package `cli`): `reconcileDuplicateGroups(args PruneArgs, manifest ingestManifest, deps PruneDeps, stdout io.Writer) duplicatePruneCounts` (`prune_duplicates.go:225`); `PruneArgs{ChunksDir string, DryRun bool}` (`prune.go:12`); `PruneDeps{ReadFile func(string) ([]byte, error), Exists func(string) bool, ...}` (`prune.go:26` — only these two fields are exercised under DryRun; `Remove` is never called because `pruneOneDuplicate` short-circuits, `LogWarning` is nil-guarded, `Lock/WriteFile/ListIndexes` are unused by `reconcileDuplicateGroups`); `indexPathFor(chunksDir, source string) string` (`ingest_dedup.go:328`); `update.Filesystem` (has `Stat`, `ReadFile` — `internal/update/update.go:92`).
- Produces: `chunkIndexHasPrunableDuplicates(chunksDir string, fileSystem update.Filesystem) bool`; report field `update.Report.ChunkIndexHasPrunableDuplicates bool`.

- [ ] **Step 1: Write the failing tests.** Rename `TestChunkIndexHasDuplicates` → `TestChunkIndexHasPrunableDuplicates` and rebuild its table. Keep the `entry` helper and per-case `u1FS` construction exactly as today (`update_test.go:47-121`); index files are placed via the new shim `cli.ExportIndexPathFor("/chunks", "<source>")`. A minimal valid index record line (decodable by `chunk.DecodeRecords`):

```go
rec := func(source, hash string) string {
	return `{"source":"` + source + `","anchor":"turn-1","content_hash":"` + hash + `","text":"t","vector":[1]}` + "\n"
}
```

New/changed cases (each case sets `fileSystem.files[cli.ExportIndexPathFor("/chunks", path)]` as listed; canonical for `{/repo/a.md, /repo/b.md}` is `/repo/a.md` — same separator count, lexicographic tie-break):

| case | manifest | index files | want |
|---|---|---|---|
| no manifest at all | nil | — | false |
| distinct hashes | a:x, b:y | — | false |
| shared hash, different chunking classes | a.md:x, a.jsonl:x | — | false |
| already tagged duplicate_of | a:x, b:x(dup-of a) | — | false |
| malformed manifest | `{not valid json` | — | false |
| **group would be removed: duplicate never indexed (vacuously covered)** | a:x, b:x | a.md: `rec("/repo/a.md","sha256:aaa")` | **true** |
| **group would be removed: duplicate's records covered by canonical** | a:x, b:x | a.md: `rec(a,"sha256:aaa")+rec(a,"sha256:bbb")`; b.md: `rec(b,"sha256:aaa")` | **true** |
| **refusal-only, anomalous (Joe's case): canonical unindexed, sibling index survives** | a:x, b:x | b.md: `rec("/repo/b.md","sha256:aaa")` | **false** |
| **refusal-only, structural: no member indexed** | a:x, b:x | — | **false** (was `true` under the old manifest-only check — the behavior change under test) |
| **refusal-only: duplicate holds a record the canonical lacks** | a:x, b:x | a.md: `rec(a,"sha256:aaa")`; b.md: `rec(b,"sha256:ccc")` | **false** |

Update `TestWriteUpdateReport_DuplicatesHint` (:426-461): field rename to `ChunkIndexHasPrunableDuplicates` everywhere, and assertions become:

```go
g.Expect(buffer.String()).To(ContainSubstring("run `engram prune --duplicates`"))
g.Expect(buffer.String()).NotTo(ContainSubstring("Upgrading"))
g.Expect(buffer.String()).NotTo(ContainSubstring("README.md"))
```

(keep the clean-report NotContain("duplicate") check and the coexists-with-empty-notice check, renaming the field there too). In `export_test.go`, rename `ExportChunkIndexHasDuplicates = chunkIndexHasDuplicates` → `ExportChunkIndexHasPrunableDuplicates = chunkIndexHasPrunableDuplicates` and add `ExportIndexPathFor = indexPathFor`.

- [ ] **Step 2: Run and see it fail.** `targ test` → compile failure (new names) counts as RED for the rename; the semantic RED is the structural-refusal case wanting `false`: temporarily point the shim at the old function if needed — otherwise accept compile-RED, implement, and confirm in Step 4 that the structural case passes *because of the gate* (flip `want` to `true` locally to see it fail if in doubt, then restore).

- [ ] **Step 3: Implementation.** In `internal/update/update.go`, rename the field and rewrite its comment:

```go
// ChunkIndexHasPrunableDuplicates is set by the cli package after Run
// returns: true when the chunk-index manifest holds at least one
// duplicate that `engram prune --duplicates` would actually remove
// right now (prune's own reconciliation, run in dry-run mode, reports
// removed > 0). Refusal-only backlogs — every group's canonical
// missing or not verifiably covering its siblings' records — stay
// silent (#713): there is no command that resolves them, and
// prune/ingest explain refusals when run. update deliberately never
// removes anything on the strength of this field — it only surfaces
// the notice naming `engram prune --duplicates`, which the user runs
// explicitly.
ChunkIndexHasPrunableDuplicates bool
```

In `internal/cli/update.go`, replace `chunkIndexHasDuplicates` (:159-194) with:

```go
// chunkIndexHasPrunableDuplicates reports whether `engram prune
// --duplicates` would actually remove anything right now: it re-runs
// prune's own reconciliation in dry-run mode (reconcileDuplicateGroups —
// the same (FileHash, chunkingClass) grouping, canonical selection, and
// record-level coverage gate), true only when at least one duplicate
// would be removed. A backlog whose every group would be REFUSED stays
// silent (#713) — refusals are pending manual review by design; prune
// and ingest explain them when run. Detection stays stateless and
// notify-only: update never removes anything itself (Unit 5's
// detect-and-notify contract holds).
//
// Cost: one manifest read + decode always; the coverage gate's
// per-source .jsonl reads happen only for hash groups with >= 2 live
// members, so an index with no duplicate backlog does no per-source
// I/O. A missing/unreadable/malformed manifest is false (self-silencing,
// same convention as chunkIndexHasEmptyFiles) — a detection failure must
// never fail `engram update`'s primary job.
func chunkIndexHasPrunableDuplicates(chunksDir string, fileSystem update.Filesystem) bool {
	data, readErr := fileSystem.ReadFile(filepath.Join(chunksDir, manifestName))
	if readErr != nil {
		return false
	}

	manifest := ingestManifest{}

	if json.Unmarshal(data, &manifest) != nil {
		return false // a malformed manifest self-silences; must never fail update
	}

	// DryRun guarantees reconcileDuplicateGroups never removes or writes:
	// pruneOneDuplicate short-circuits before removeDuplicateIndex, so
	// PruneDeps.Remove is never invoked (left nil deliberately — a call
	// would be a bug, and failing loud beats silently deleting). Refusal
	// detail lines go to io.Discard; update only needs the counts.
	counts := reconcileDuplicateGroups(
		PruneArgs{ChunksDir: chunksDir, DryRun: true},
		manifest,
		PruneDeps{
			ReadFile: fileSystem.ReadFile,
			Exists: func(path string) bool {
				_, statErr := fileSystem.Stat(path)

				return statErr == nil
			},
		},
		io.Discard,
	)

	return counts.removed > 0
}
```

(`io` is already imported in `update.go` for `io.Writer`; verify.) Change `runUpdate` (:333) to `report.ChunkIndexHasPrunableDuplicates = chunkIndexHasPrunableDuplicates(chunksDir, deps.FS)`. Change `writeDuplicatesHint` (:366-377) to read the renamed field, with doc comment: "prints a one-line notice naming `engram prune --duplicates` when the chunk index holds a duplicate backlog that command would actually remove (refusal-only backlogs stay silent — #713). Deliberately just a notice: update never removes anything on the user's behalf here." Change the const:

```go
duplicateChunksNotice = "duplicate chunk-index files found — run `engram prune --duplicates` to clear them " +
	"(preview with `--dry-run`)\n"
```

- [ ] **Step 4: Run and see it pass.** `targ test` → PASS, including the structural-refusal `false` case.

- [ ] **Step 5: Full check.** `targ check-full` → collect ALL findings, fix in one pass, re-run to green.

- [ ] **Step 6: Commit.**

```bash
git add internal/cli/update.go internal/cli/update_test.go internal/cli/export_test.go internal/update/update.go
git commit -m "feat(update): duplicates notice gated on prune's own dry-run coverage check (#713)" -m "Refusal-only backlogs go silent; notice text names the command and drops the README pointer." -m "AI-Used: [claude]"
```

### Task 4: Remove README Upgrading section; scrub doc pointers

**Files:**
- Modify: `README.md` (delete lines 42-50: the `## Upgrading` heading + three paragraphs + migrate-tags line; leave `## Skills` and everything else untouched)
- Modify: `docs/README.md:14`, `docs/GLOSSARY.md:669-674`, `docs/FEATURES.md:199-200`, `docs/architecture/adr.md:680-686`

**Interfaces:** none (docs only). Dispositions per the table above.

- [ ] **Step 1: Delete the README section.** Remove `README.md:42-51` (heading through the trailing blank line before `## Skills`). The removed content's only surviving guidance is the notices themselves — that's the point.

- [ ] **Step 2: Scrub the four pointers.**
  - `docs/README.md:14`: `(Installing + Upgrading + Binary commands)` → `(Installing + Binary commands)`.
  - `docs/GLOSSARY.md:669-674` (`engram update` entry): rewrite the notice sentence to: update prints a one-line notice naming the exact command when it detects old-format vocab files (#678 — `engram update --regen-vocab`), leftover empty `.jsonl` chunk-index files (#694 — `engram prune --empty`), or a duplicate backlog that `engram prune --duplicates` would actually remove (ADR-0021, refined by #713: prune's dry-run coverage gate decides; refusal-only backlogs stay silent); `update` never removes anything itself.
  - `docs/FEATURES.md:199-200`: `"…and notifies the user of it (see README's Upgrading section) rather than removing anything on its own."` → `"…and notifies the user with the exact command when \`prune --duplicates\` would actually remove something (#713) rather than removing anything on its own."`
  - `docs/architecture/adr.md` decision 9 (:680-686): append one sentence at the end of the decision-9 paragraph: `Refined by #713 (2026-07-27): detection now runs prune's own dry-run coverage gate, so the notice fires only when a removal would actually happen, and it names the command directly (the README Upgrading section it used to point at is removed).` Do not alter the existing decision text.

- [ ] **Step 3: Verify no stale pointers.** Run the enumeration grep from the disposition table; confirm every remaining hit is a `keep` row (ROADMAP/plan-doc/LEDGER/skill/ingest-message/historical-code-comment). `targ check-full` still green.

- [ ] **Step 4: Commit.**

```bash
git add README.md docs/README.md docs/GLOSSARY.md docs/FEATURES.md docs/architecture/adr.md
git commit -m "docs: remove README Upgrading section; notices are self-sufficient (#713)" -m "AI-Used: [claude]"
```

### Task 5: Read-only verification with the real binary

**Files:** none modified. Scratch build only.

- [ ] **Step 1: Build to scratch (NOT go install).**

```bash
go build -o /private/tmp/claude-501/-Users-joe-repos-personal-engram/ab662746-d7ea-4998-8e01-11bfa07b75f3/scratchpad/engram-713 ./cmd/engram
```

- [ ] **Step 2: Cross-check the duplicates notice against prune's own verdict (both read-only).** From the worktree root:

```bash
/…/scratchpad/engram-713 prune --duplicates --dry-run   # note the "[dry-run] prune: removed N…" count
/…/scratchpad/engram-713 update --dry-run               # inspect the notices block
```

PASS criteria: the duplicates notice appears **iff** prune's dry-run reported `removed > 0`; no notice text anywhere mentions "Upgrading" or "README.md"; the vocab notice (if Joe's vault had old-format files — expected absent) names `engram update --regen-vocab`; the empty-files notice (if present) names `engram prune --empty`. Record the actual outputs in the task report.

- [ ] **Step 3: Confirm Joe's reported case is fixed.** His live index's known duplicate groups are anomalous-refusal (`baseline-recency-l1-episode-*` vs `baseline-recency-conflict-*`). If prune dry-run shows `removed 0` overall, `update --dry-run` must print NO duplicates notice — the nag is gone. If the live index meanwhile holds removable groups (`removed > 0`), the notice appearing is CORRECT — the criterion is agreement, not absence.

---

## Self-review notes

- Spec coverage: issue #713 solution items 1 (Task 1), 2 (Task 3), 3 (Task 2), 4 (Task 4); verification (Task 5). Detect-and-notify contract preserved (no auto-prune — ADR-0021 reversal honored).
- Type consistency: `chunkIndexHasPrunableDuplicates` / `ChunkIndexHasPrunableDuplicates` / `ExportChunkIndexHasPrunableDuplicates` used consistently across Tasks 3 steps; `reconcileDuplicateGroups` signature verified against `prune_duplicates.go:225`; `update.Filesystem.Stat` verified at `internal/update/update.go:92-104`; `u1FS` implements `Stat` (`invariants_u1_test.go:256`).
- Known risk, called out for the executor: `reconcileDuplicateGroups` under DryRun must never call `PruneDeps.Remove` (nil). This is guaranteed by `pruneOneDuplicate`'s order (covered-check → DryRun short-circuit → remove). If a test ever panics on a nil Remove, that is a REAL regression in prune — stop and report, do not paper over by supplying a no-op Remove.
