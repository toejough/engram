# #713 — Update Notices: Exact Commands, Only When They Work

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `engram update`'s upgrade notices name the exact command inline and appear only when running it would actually do something; the README's Upgrading section (the notices' current destination) is removed.

**Architecture:** Three notice constants in `internal/cli/update.go` change text (drop the README pointer, name the command). The duplicates notice additionally changes *condition*: instead of the manifest-only grouping check (`chunkIndexHasDuplicates`), update reuses prune's own engine (`reconcileDuplicateGroups` in DryRun mode) so the notice fires only when `engram prune --duplicates` would report `removed > 0`. A refusal-only backlog (Joe's reported case) goes silent — refusals are pending-manual-review by design, there is no command to give, and prune/ingest explain refusals when run. The README section deletion is the asked scope; scrubbing the four adjacent doc pointers (GLOSSARY/FEATURES/ADR/docs-index) is **proposed, not committed** — gated on Joe's yes/no per vault notes 345/393/467 (propose-and-gate, never fold).

**Tech Stack:** Go, gomega (blackbox `package cli_test` via `export_test.go` shims), targ for test/lint.

**Issue:** https://github.com/toejough/engram/issues/713 (design and alternatives recorded there).

**Deliberate carry-over (called out per Gate A):** each new notice keeps a preview hint, now as the explicit full command (e.g. ``(preview with `engram prune --duplicates --dry-run`)``) — this preserves the deleted README section's "preview first with --dry-run" guidance inline rather than dropping a safety affordance for destructive commands. It is the one piece of README prose that survives, relocated into the notice it belongs to.

## Global Constraints

- Test/lint ONLY via targ: `targ test`, `targ check-full`. NEVER `go test` / `go vet` directly. `targ check-full` reports all errors at once — collect the full list, fix in one pass.
- TDD every code task: write the failing test, SEE the semantic failure, minimal implementation, SEE it pass; renames are REFACTOR steps done only under green.
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
| `README.md:42-51` (`## Upgrading` + trailing blank) | **remove** | The ask: section deleted entirely (Task 4) |
| `README.md:30-40` (Installing) | keep | "Run it again any time to upgrade" is about the command, not the section |
| `internal/cli/update.go:30-36` (3 notice consts) | **update** | Tasks 1–3: new text, no README pointer |
| `internal/cli/update.go:159-174` (`chunkIndexHasDuplicates` doc) | **rewrite** | Task 3: function becomes coverage-aware |
| `internal/cli/update.go:366-372` (`writeDuplicatesHint` doc) | **update** | Task 3: names README/Upgrading |
| `internal/cli/update.go:512-514` (`writeVocabMigrationHint` doc) | **update** | Task 1: names README/Upgrading |
| `internal/update/update.go:164-171` (`ChunkIndexHasDuplicates` field) | **update** | Task 3: rename + comment (mentions the notice's README pointer) |
| `internal/cli/update_test.go:39-132,426-461,463-491,769-786,832-886` | **update** | Tasks 1–3: assertions pin old text/field (`:39-132` is the full `TestChunkIndexHasDuplicates` function) |
| `internal/cli/export_test.go:48` | **update** | Task 3: shim rename + add `ExportIndexPathFor` |
| `docs/README.md:14` (index row) | **proposed follow-up** | Names "Upgrading" as a README destination — gated on Joe's yes/no |
| `docs/GLOSSARY.md:669-674` (`engram update` entry) | **proposed follow-up** | Describes notices pointing at README Upgrading — gated on Joe's yes/no |
| `docs/FEATURES.md:199-200` | **proposed follow-up** | "(see README's Upgrading section)" — gated on Joe's yes/no |
| `docs/architecture/adr.md:680-692` (ADR-0021 decision 9; paragraph ends line 692) | **proposed follow-up (append)** | Append a #713 refinement note after line 692; never rewrite the decision text — gated on Joe's yes/no |
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
- Test: `internal/cli/update_test.go:769-786` (`TestWriteUpdateReport_VocabMigrationHint`), `:832-886` (`TestWriteUpdateReport_VocabRegenReport`)

**Interfaces:** none new; the const `vocabMigrationNotice` keeps its name.

- [ ] **Step 1: Make the test demand the new text.** In `TestWriteUpdateReport_VocabMigrationHint` (table around line 769-786), the `old-vocab-present` case currently has `wantContains: []string{"Upgrading", "README.md"}`. Change to:

```go
wantContains:    []string{"run `engram update --regen-vocab`"},
wantNotContains: []string{"Upgrading", "README.md"},
```

The `no-old-vocab` case's `wantNotContains: []string{"Upgrading"}` becomes `[]string{"old-format vocab"}` — the section name is disappearing from all output, so pin the notice's own text instead. In `TestWriteUpdateReport_VocabRegenReport` (`:832-886`), each `wantNotContains: []string{"see the Upgrading section", ...}` becomes `[]string{"old-format vocab files found", ...}` — the invariant is "the regen summary replaces the plain notice", and pinning the notice's opening clause is the same convention `TestRunUpdate_RegenVocabFlag_RunsRegenAndUpdatesReport` already uses at `update_test.go:397`.

- [ ] **Step 2: Run and see it fail.** `targ test` → `TestWriteUpdateReport_VocabMigrationHint` FAILS (output still says "see the Upgrading section…").

- [ ] **Step 3: Minimal implementation.** In `internal/cli/update.go`, replace the const (current text: `"old-format vocab files found — see the Upgrading section in README.md for migration steps\n"`):

```go
vocabMigrationNotice = "old-format vocab files found — run `engram update --regen-vocab` to migrate them " +
	"(preview with `engram update --regen-vocab --dry-run`)\n"
```

Replace `writeVocabMigrationHint`'s doc comment. Current (:512-514):

```go
// writeVocabMigrationHint prints a one-line pointer to the README
// "Upgrading" section when the vault still holds pre-tags vocab files.
// Silent otherwise — a vault that never had the old format never sees it.
```

New:

```go
// writeVocabMigrationHint prints a one-line notice naming `engram update
// --regen-vocab` when the vault still holds pre-tags vocab files. Silent
// otherwise — a vault that never had the old format never sees it.
```

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

(The "both notices coexist" sub-test at `:481-490` is unaffected: its line-490 `ContainSubstring("old-format vocab")` still matches Task 1's new vocab notice.)

- [ ] **Step 2: Run and see it fail.** `targ test` → FAIL.

- [ ] **Step 3: Minimal implementation.** Replace the const (current text: `"empty chunk-index files found — run `engram prune --empty` to clear them; see the Upgrading section in README.md\n"`):

```go
emptyChunkFilesNotice = "empty chunk-index files found — run `engram prune --empty` to clear them " +
	"(preview with `engram prune --empty --dry-run`)\n"
```

Replace `writeEmptyChunkHint`'s doc comment. Current (:379-381):

```go
// writeEmptyChunkHint prints a one-line pointer to the README "Upgrading"
// section when the chunk index still holds 0-byte .jsonl files. Silent
// otherwise — a vault whose index was already pruned never sees it.
```

New:

```go
// writeEmptyChunkHint prints a one-line notice naming `engram prune --empty`
// when the chunk index still holds 0-byte .jsonl files. Silent otherwise — a
// vault whose index was already pruned never sees it.
```

- [ ] **Step 4: Run and see it pass.** `targ test` → PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/cli/update.go internal/cli/update_test.go
git commit -m "feat(update): empty-files notice self-sufficient, drops README pointer (#713)" -m "AI-Used: [claude]"
```

### Task 3: Duplicates notice fires only when prune would actually remove something

**Files:**
- Modify: `internal/cli/update.go` (const :30-31; `chunkIndexHasDuplicates` :159-194 — behavior then rename; `runUpdate` :333; `writeDuplicatesHint` :366-377)
- Modify: `internal/update/update.go:164-171` (field comment; rename `ChunkIndexHasDuplicates` → `ChunkIndexHasPrunableDuplicates` in the refactor step)
- Modify: `internal/cli/export_test.go:48` (add `ExportIndexPathFor`; rename shim in the refactor step)
- Test: `internal/cli/update_test.go:39-132` (`TestChunkIndexHasDuplicates` — full function), `:426-461` (`TestWriteUpdateReport_DuplicatesHint`)

**Interfaces:**
- Consumes (already exist, package `cli`): `reconcileDuplicateGroups(args PruneArgs, manifest ingestManifest, deps PruneDeps, stdout io.Writer) duplicatePruneCounts` (`prune_duplicates.go:225`); `PruneArgs{ChunksDir string, DryRun bool}` (`prune.go:12`); `PruneDeps{ReadFile func(string) ([]byte, error), Exists func(string) bool, ...}` (`prune.go:26` — only these two fields are exercised under DryRun; `Remove` is never called because `pruneOneDuplicate` short-circuits, `LogWarning` is nil-guarded, `Lock/WriteFile/ListIndexes` are unused by `reconcileDuplicateGroups` — verified by Gate A call-chain trace); `indexPathFor(chunksDir, source string) string` (`ingest_dedup.go:328`); `update.Filesystem` (has `Stat`, `ReadFile` — `internal/update/update.go:92`).
- Produces: `chunkIndexHasPrunableDuplicates(chunksDir string, fileSystem update.Filesystem) bool` (final name, after the refactor step); report field `update.Report.ChunkIndexHasPrunableDuplicates bool`.

- [ ] **Step 1: Write the failing behavior tests (names unchanged).** In `export_test.go`, add `ExportIndexPathFor = indexPathFor` to the var block. In `TestChunkIndexHasDuplicates` (keep the name for now — renaming happens under green in Step 6), keep the `entry` helper and per-case `u1FS` construction (`update_test.go:47-121`) and rebuild the table per the rows below. Add a record helper (a minimal line decodable by `chunk.DecodeRecords`):

```go
rec := func(source, hash string) string {
	return `{"source":"` + source + `","anchor":"turn-1","content_hash":"` + hash + `","text":"t","vector":[1]}` + "\n"
}
```

Table notation: `a:x` = a manifest entry for source path `/repo/a.md` with `file_hash` `"sha256:x"`, built via the existing `entry` helper (fields `mtime_unix_nano`, `size`, `file_hash`, optional `duplicate_of` — the real `manifestEntry` schema). "Chunking class" = the extension-derived class from `chunkingClass` (`ingest_dedup.go:208`: `.jsonl` vs everything-else). "Index file" rows mean `fileSystem.files[cli.ExportIndexPathFor("/chunks", "<source path>")] = []byte(...)` in that case's own `u1FS`. Canonical for `{/repo/a.md, /repo/b.md}` is `/repo/a.md` (Gate-A-verified tie-break: equal origin rank and separator count → lexicographic).

| case | manifest | index files | want |
|---|---|---|---|
| no manifest at all | nil | — | false |
| distinct hashes | a:x, b:y | — | false |
| shared hash, different chunking classes | a.md:x, a.jsonl:x | — | false |
| already tagged duplicate_of | a:x, b:x(dup-of a) | — | false |
| malformed manifest | `{not valid json` | — | false |
| **group would be removed: duplicate never indexed (vacuously covered)** | a:x, b:x | a.md: `rec("/repo/a.md","sha256:aaa")` | **true** |
| **group would be removed: duplicate's records covered by canonical** | a:x, b:x | a.md: `rec("/repo/a.md","sha256:aaa")+rec("/repo/a.md","sha256:bbb")`; b.md: `rec("/repo/b.md","sha256:aaa")` | **true** |
| **refusal-only, anomalous (Joe's case): canonical unindexed, sibling index survives** | a:x, b:x | b.md: `rec("/repo/b.md","sha256:aaa")` | **false** |
| **refusal-only, structural: no member indexed** | a:x, b:x | — | **false** (was `true` under the manifest-only check — the behavior change under test) |
| **refusal-only: duplicate holds a record the canonical lacks** | a:x, b:x | a.md: `rec("/repo/a.md","sha256:aaa")`; b.md: `rec("/repo/b.md","sha256:ccc")` | **false** |

- [ ] **Step 2: Run and SEE the semantic failures.** `targ test` → `TestChunkIndexHasDuplicates` FAILS on exactly the three refusal-only cases (old manifest-only code returns `true`, tests want `false`). The two "would be removed" cases already pass (old code also says `true`) — that is expected; the RED is the refusal cases.

- [ ] **Step 3: GREEN — make detection coverage-aware (name unchanged).** Replace the BODY of `chunkIndexHasDuplicates` (:175-194; doc comment updated in Step 6 with the rename):

```go
func chunkIndexHasDuplicates(chunksDir string, fileSystem update.Filesystem) bool {
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

(The `if json.Unmarshal(data, &manifest) != nil` form is the file's existing convention at :183. `io` is already imported at `update.go:9`.) `targ test` → PASS.

- [ ] **Step 4: RED — notice text.** In `TestWriteUpdateReport_DuplicatesHint` (:426-461), rewrite the test's doc comment (:426-431) to describe the new invariant ("when a prunable duplicate backlog was detected, the report names `engram prune --duplicates` with no README pointer, and update NEVER performs the removal itself") and replace the three positive assertions with:

```go
g.Expect(buffer.String()).To(ContainSubstring("run `engram prune --duplicates`"))
g.Expect(buffer.String()).NotTo(ContainSubstring("Upgrading"))
g.Expect(buffer.String()).NotTo(ContainSubstring("README.md"))
```

(keep the clean-report `NotContain("duplicate")` check and the coexists-with-empty-notice check as-is). `targ test` → FAIL on the two NotContain assertions.

- [ ] **Step 5: GREEN — const text.** Replace the const (current text: `"duplicate chunk-index files found — run `engram prune --duplicates` to clear them; see the Upgrading section in README.md\n"`):

```go
duplicateChunksNotice = "duplicate chunk-index files found — run `engram prune --duplicates` to clear them " +
	"(preview with `engram prune --duplicates --dry-run`)\n"
```

`targ test` → PASS.

- [ ] **Step 6: REFACTOR (under green) — rename to say what it now is.** Behavior-preserving renames: `chunkIndexHasDuplicates` → `chunkIndexHasPrunableDuplicates`; `update.Report.ChunkIndexHasDuplicates` → `ChunkIndexHasPrunableDuplicates` (usage sites verified by Gate A grep: only `internal/cli/update.go`, `internal/cli/update_test.go`, `internal/cli/export_test.go`, `internal/update/update.go`); shim → `ExportChunkIndexHasPrunableDuplicates`; test func → `TestChunkIndexHasPrunableDuplicates`; `runUpdate` (:333) and `writeDuplicatesHint` (:373-377) follow. Rewrite the function's doc comment:

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
```

the Report field's comment:

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
```

and `writeDuplicatesHint`'s comment: "prints a one-line notice naming `engram prune --duplicates` when the chunk index holds a duplicate backlog that command would actually remove (refusal-only backlogs stay silent — #713). Deliberately just a notice: update never removes anything on the user's behalf here." `targ test` → still PASS (pure rename).

- [ ] **Step 7: Full check.** `targ check-full` → collect ALL findings, fix in one pass, re-run to green.

- [ ] **Step 8: Commit.**

```bash
git add internal/cli/update.go internal/cli/update_test.go internal/cli/export_test.go internal/update/update.go
git commit -m "feat(update): duplicates notice gated on prune's own dry-run coverage check (#713)" -m "Refusal-only backlogs go silent; notice text names the command and drops the README pointer." -m "AI-Used: [claude]"
```

### Task 4: Remove the README Upgrading section (the asked scope)

**Files:**
- Modify: `README.md` (delete lines 42-51: the `## Upgrading` heading, three paragraphs, migrate-tags line, and the trailing blank line; `## Skills` and everything else untouched)

- [ ] **Step 1: Delete the section.** Remove `README.md:42-51`. The removed content's only surviving guidance is the notices themselves — that's the point. (The preview-first guidance survives inside the new notice texts; see "Deliberate carry-over" above.)

- [ ] **Step 2: Verify the remaining references are exactly the known ones.** Run (bare-word pattern — Gate A verified the earlier phrase-based pattern structurally missed `docs/README.md:14`, whose parenthetical says just "Upgrading", and `docs/architecture/adr.md:684`, where "README Upgrading / section" wraps across lines):

```bash
grep -rn "Upgrading" --include="*.go" --include="*.md" . \
  | grep -v "\.claude/worktrees\|docs/superpowers/plans/\|docs/ROADMAP\.md\|dev/eval/LEDGER\.md"
```

Expected output after Tasks 1-4: EXACTLY the four proposed-follow-up rows (`docs/README.md:14`, `docs/GLOSSARY.md:670`, `docs/FEATURES.md:200`, `docs/architecture/adr.md:684`) — each awaiting Joe's yes/no below — and nothing else. (Pre-Task-4, the same command additionally shows README.md:42 and the update.go/update_test.go hits Tasks 1-3 remove, including the test doc comment at `update_test.go:428` handled in Task 3 Step 4.) Any other hit is a missed scrub: stop and report it.

- [ ] **Step 3: Commit.**

```bash
git add README.md
git commit -m "docs(readme): remove the Upgrading section; update notices are self-sufficient (#713)" -m "AI-Used: [claude]"
```

### Task 5: Read-only verification with the real binary

**Files:** none modified. Scratch build only.

- [ ] **Step 1: Build to scratch (NOT go install).**

```bash
BIN="$(mktemp -d)/engram-713"
go build -o "$BIN" ./cmd/engram
```

- [ ] **Step 2: Cross-check the duplicates notice against prune's own verdict (both read-only).** From the worktree root:

```bash
"$BIN" prune --duplicates --dry-run   # note the "[dry-run] prune: removed N…" count
"$BIN" update --dry-run               # inspect the notices block
```

PASS criteria: the duplicates notice appears **iff** prune's dry-run reported `removed > 0`; no notice text anywhere mentions "Upgrading" or "README.md"; the vocab notice (if Joe's vault had old-format files — expected absent) names `engram update --regen-vocab`; the empty-files notice (if present) names `engram prune --empty`. Record the actual outputs in the task report.

- [ ] **Step 3: Confirm Joe's reported case is fixed.** His live index's known duplicate groups are anomalous-refusal (`baseline-recency-l1-episode-*` vs `baseline-recency-conflict-*`). If prune dry-run shows `removed 0` overall, `update --dry-run` must print NO duplicates notice — the nag is gone. If the live index meanwhile holds removable groups (`removed > 0`), the notice appearing is CORRECT — the criterion is agreement, not absence.

---

## Proposed follow-up (gated on Joe's yes/no — do NOT execute without it)

Removing the README section and changing the notice behavior makes four adjacent doc statements false. Per vault notes 345/393/467 these are outside the verbatim ask, so they are **proposed** here, not committed:

1. `docs/README.md:14`: `(Installing + Upgrading + Binary commands)` → `(Installing + Binary commands)`.
2. `docs/GLOSSARY.md:669-674` (`engram update` entry): rewrite the notice sentence — update prints a one-line notice naming the exact command when it detects old-format vocab files (#678 — `engram update --regen-vocab`), leftover empty `.jsonl` chunk-index files (#694 — `engram prune --empty`), or a duplicate backlog that `engram prune --duplicates` would actually remove (ADR-0021, refined by #713: prune's dry-run coverage gate decides; refusal-only backlogs stay silent); `update` never removes anything itself.
3. `docs/FEATURES.md:199-200`: `"…and notifies the user of it (see README's Upgrading section) rather than removing anything on its own."` → `"…and notifies the user with the exact command when \`prune --duplicates\` would actually remove something (#713) rather than removing anything on its own."`
4. `docs/architecture/adr.md` decision 9 (paragraph is `:680-692`; append AFTER line 692, before the blank line preceding decision 10): `Refined by #713 (2026-07-27): detection now runs prune's own dry-run coverage gate, so the notice fires only when a removal would actually happen, and it names the command directly (the README Upgrading section it used to point at is removed).` Do not alter the existing decision text.

**What folding buys:** docs stay truthful — after Task 4, GLOSSARY/FEATURES/docs-index describe a README section that no longer exists, and GLOSSARY/ADR describe notice behavior that is no longer accurate. **What deferring costs:** those four statements are false until Joe answers. **Recommendation:** approve — this is consistency repair of statements the asked change falsifies, not new content. Commit message if approved: `docs: scrub pointers to the removed README Upgrading section (#713)` + `AI-Used: [claude]` trailer.

---

## Self-review notes

- Ask coverage checked against Joe's VERBATIM words (not just issue #713's restatement): "tell them to run that command in the update prose instead" → Task 1; "the other instructions in the upgrading section — just remove them" → Task 4 deletes the whole section; "the commands they need to run, and only when they need to run them" → Tasks 2-3 (the duplicates condition is the only one that needed to change); his refusal transcript → Task 3's anomalous-case row + Task 5 Step 3. Adjacent doc scrub → proposed-and-gated, not folded (notes 345/393/467).
- Type consistency: `chunkIndexHasPrunableDuplicates` / `ChunkIndexHasPrunableDuplicates` / `ExportChunkIndexHasPrunableDuplicates` appear only in Task 3 Step 6 (the refactor); Steps 1-5 use the current names. `reconcileDuplicateGroups` signature verified against `prune_duplicates.go:225`; `update.Filesystem.Stat` verified at `internal/update/update.go:92-104`; `u1FS` implements `Stat` (`invariants_u1_test.go:256`); DryRun-safety call-chain independently verified by Gate A code-alignment review.
- Known risk, called out for the executor: `reconcileDuplicateGroups` under DryRun must never call `PruneDeps.Remove` (nil). This is guaranteed by `pruneOneDuplicate`'s order (covered-check → DryRun short-circuit → remove). If a test ever panics on a nil Remove, that is a REAL regression in prune — stop and report, do not paper over by supplying a no-op Remove.
