package cli_test

import (
	"encoding/json"
	"errors"
	"maps"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a10f1a_ReparentLuhmann_TheNewPathIsIndexedAndTheOldEntryDetached verifies Scenario 792a10f1a: a note with
// chunks in the index, renamed by one `engram update --reparent-luhmann --answers` run, has its new path indexed and
// its old path's manifest entry detached in that same run, with no separate ingest or prune.
func Test792a10f1a_ReparentLuhmann_TheNewPathIsIndexedAndTheOldEntryDetached(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	oldPath := vm2Vault + "/12.2026-01-02.second-note.md"
	newPath := vm2Vault + "/7a.2026-01-02.second-note.md"

	files[vm2Chunks+"/manifest.json"] = []byte(
		`{"` + oldPath + `":{"mtime_unix_nano":1,"size":10,"file_hash":"sha256:old"}}`)
	files[cli.ExportIndexPathFor(vm2Chunks, oldPath)] = []byte(cis1Record(oldPath, "sha256:chunk"))

	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "continuation", "7"))

	_, err := harness.update(t, vm2Apply())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(harness.renames).To(HaveKeyWithValue(oldPath, newPath))

	manifest := map[string]json.RawMessage{}
	g.Expect(json.Unmarshal(harness.files[vm2Chunks+"/manifest.json"], &manifest)).To(Succeed())
	g.Expect(manifest).NotTo(HaveKey(oldPath), "the old path's manifest entry is detached")
	g.Expect(manifest).To(HaveKey(newPath), "the new path is indexed")
	g.Expect(harness.files).To(HaveKey(cli.ExportIndexPathFor(vm2Chunks, newPath)), "the new path has its chunks")
}

// Test792a10f1b_ReparentLuhmann_AFailedReindexLeavesTheRename verifies Scenario 792a10f1b: when the rename succeeded
// and the re-index step (or the step that detaches the old entries) fails, the renamed files stay renamed, and the
// output names the failed stage, says the vault rename is intact and names the manual fallback.
func Test792a10f1b_ReparentLuhmann_AFailedReindexLeavesTheRename(t *testing.T) {
	t.Parallel()

	oldPath := vm2Vault + "/12.2026-01-02.second-note.md"
	newPath := vm2Vault + "/7a.2026-01-02.second-note.md"

	cases := map[string]struct {
		sabotage func(deps *cli.ReparentDeps)
		cause    error
		stage    string
		manual   []string
	}{
		"the re-index fails": {
			sabotage: func(deps *cli.ReparentDeps) { deps.Ingest.Embedder = reparentFailingEmbedder{} },
			cause:    errReparentEmbedFailed,
			stage:    "re-indexing renamed notes failed",
			manual:   []string{"`engram ingest --auto`", "`engram prune`"},
		},
		"the detach of the old entries fails": {
			sabotage: func(deps *cli.ReparentDeps) {
				deps.Prune.WriteFile = func(string, []byte) error { return errVM2ManifestWrite }
			},
			cause:  errVM2ManifestWrite,
			stage:  "pruning stale chunk-manifest entries failed",
			manual: []string{"`engram prune`"},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			files, names := twoRelatedTopLevelNotesFixture()
			files[vm2Chunks+"/manifest.json"] = []byte(
				`{"` + oldPath + `":{"mtime_unix_nano":1,"size":10,"file_hash":"sha256:old"}}`)

			harness := vm2NewHarness(files, names)
			testCase.sabotage(&harness.deps)
			harness.writeAnswers(t, vm2Entry("12", "continuation", "7"))

			out, err := harness.update(t, vm2Apply())
			g.Expect(err).To(MatchError(testCase.cause))

			g.Expect(harness.renames).To(HaveKeyWithValue(oldPath, newPath), "the rename happened")
			g.Expect(harness.files).NotTo(HaveKey(oldPath), "the old file is gone: the rename is not undone")
			g.Expect(harness.files).To(HaveKey(newPath), "the renamed file stays renamed")
			g.Expect(out).To(ContainSubstring(testCase.stage), "the failed stage is named")
			g.Expect(out).To(ContainSubstring("vault rename above is intact and authoritative"))

			for _, command := range testCase.manual {
				g.Expect(out).To(ContainSubstring(command), "the manual fallback names "+command)
			}
		})
	}
}

// Test792a10f1c_ReparentLuhmann_NothingRenamedNothingReindexed verifies Scenario 792a10f1c: answers that rename no
// note leave the chunk index untouched: no ingest or prune capability is called, and every file, the manifest and the
// index among them, is as it was.
func Test792a10f1c_ReparentLuhmann_NothingRenamedNothingReindexed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	oldPath := vm2Vault + "/12.2026-01-02.second-note.md"

	files[vm2Chunks+"/manifest.json"] = []byte(
		`{"` + oldPath + `":{"mtime_unix_nano":1,"size":10,"file_hash":"sha256:old"}}`)
	files[cli.ExportIndexPathFor(vm2Chunks, oldPath)] = []byte(cis1Record(oldPath, "sha256:chunk"))

	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "top", "7"))

	before := maps.Clone(harness.files)

	out, err := harness.update(t, vm2Apply())
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(out).To(HavePrefix("update --reparent-luhmann: renamed 0 note(s)\n"))
	g.Expect(harness.renames).To(BeEmpty())
	g.Expect(*harness.touches).To(BeZero(), "neither the ingest nor the prune capability is called")
	g.Expect(harness.files).To(Equal(before), "the chunk index and the vault are as they were")
}

// Test792a10g1a_ReparentLuhmann_MoreCandidatesRemain verifies Scenario 792a10g1a: an apply whose resulting vault
// still has a similar pair of top-level notes prints `renamed <N> note(s)` first, then the count of the candidates of
// the resulting vault (the renamed note is no longer a candidate) and the command to run again.
func Test792a10g1a_ReparentLuhmann_MoreCandidatesRemain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := vm2TwoPairsVault()
	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("21", "continuation", "20"))

	out, err := harness.update(t, vm2Apply())
	g.Expect(err).NotTo(HaveOccurred())

	lines := vm2Lines(out)
	g.Expect(lines).NotTo(BeEmpty())

	if len(lines) == 0 {
		return
	}

	g.Expect(lines[0]).To(Equal("update --reparent-luhmann: renamed 1 note(s)"), "the renamed count is printed first")
	g.Expect(lines[len(lines)-1]).To(Equal(
		"update --reparent-luhmann: 2 further candidate(s) found — run `engram update --reparent-luhmann` again"),
		"the pair 7 and 12 remains, the pair 20 and 21 is resolved")
}

// Test792a10g1b_ReparentLuhmann_TheVaultIsFullyEvaluated verifies Scenario 792a10g1b: an apply whose resulting vault
// has no similar pair of top-level notes says the vault is fully evaluated, last, and gives no command to run.
func Test792a10g1b_ReparentLuhmann_TheVaultIsFullyEvaluated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := vm2TwoPairsVault()
	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "continuation", "7"), vm2Entry("21", "continuation", "20"))

	out, err := harness.update(t, vm2Apply())
	g.Expect(err).NotTo(HaveOccurred())

	lines := vm2Lines(out)
	g.Expect(lines).NotTo(BeEmpty())

	if len(lines) == 0 {
		return
	}

	g.Expect(lines[0]).To(Equal("update --reparent-luhmann: renamed 2 note(s)"))
	g.Expect(lines[len(lines)-1]).To(Equal("update --reparent-luhmann: no further candidates — vault fully evaluated"))
	g.Expect(out).NotTo(ContainSubstring("again"), "no instruction to run the command again")
	g.Expect(out).NotTo(ContainSubstring("run `engram"), "no further command is given")
}

// Test792a10h1a_ReparentLuhmann_APreviewChangesNothing verifies Scenario 792a10h1a: with valid, current answers,
// `--dry-run` lists each old-to-new basename and the notes whose links would be rewritten, and modifies no file, the
// chunk index included: no ingest or prune capability is called.
func Test792a10h1a_ReparentLuhmann_APreviewChangesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	oldPath := vm2Vault + "/12.2026-01-02.second-note.md"

	files[vm2Vault+"/20.2026-01-03.unrelated-note.md"] = []byte(
		"---\ntype: fact\nluhmann: \"20\"\ncreated: 2026-01-03\n---\n\nsee [[12.2026-01-02.second-note]] also.\n")
	files[vm2Chunks+"/manifest.json"] = []byte(
		`{"` + oldPath + `":{"mtime_unix_nano":1,"size":10,"file_hash":"sha256:old"}}`)

	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "continuation", "7"))

	before := maps.Clone(harness.files)

	out, err := harness.update(t, vm2Preview())
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(vm2Lines(out)).To(ContainElements(
		"  12.2026-01-02.second-note -> 7a.2026-01-02.second-note",
		"  references rewritten in: 20.2026-01-03.unrelated-note.md",
	))
	g.Expect(strings.Count(out, "references rewritten in:")).To(Equal(1), "only the one referrer is listed")
	g.Expect(harness.renames).To(BeEmpty())
	g.Expect(*harness.touches).To(BeZero(), "the chunk index is not read, indexed or pruned")
	g.Expect(harness.files).To(Equal(before), "no file is modified")
}

// Test792a10h1b_ReparentLuhmann_APreviewWithoutAnswersIsRefused verifies Scenario 792a10h1b: `--dry-run` without
// `--answers` fails with a usage error saying `--dry-run` requires `--answers`, prints nothing and writes nothing.
func Test792a10h1b_ReparentLuhmann_APreviewWithoutAnswersIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	harness := vm2NewHarness(files, names)

	before := maps.Clone(harness.files)

	out, err := harness.update(t, cli.UpdateArgs{ReparentLuhmann: true, DryRun: true})
	g.Expect(err).To(MatchError(ContainSubstring("--dry-run requires --answers")))
	g.Expect(out).To(BeEmpty())
	g.Expect(harness.renames).To(BeEmpty())
	g.Expect(*harness.touches).To(BeZero())
	g.Expect(harness.files).To(Equal(before))
}

// Test792a10h1c_ReparentLuhmann_APreviewReportsARefusal verifies Scenario 792a10h1c: answers that would rename a
// note whose luhmann: is anchored fail the preview with the same refusal as the apply, and write nothing.
func Test792a10h1c_ReparentLuhmann_APreviewReportsARefusal(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	preview := vm2NewAnchoredHarness(t)
	apply := vm2NewAnchoredHarness(t)
	before := maps.Clone(preview.files)

	_, previewErr := preview.update(t, vm2Preview())
	g.Expect(previewErr).To(MatchError(cli.ErrRenameUndecodableForTest))
	g.Expect(previewErr).To(MatchError(ContainSubstring(vm2AnchoredBasename)))
	g.Expect(preview.renames).To(BeEmpty())
	g.Expect(preview.files).To(Equal(before), "the preview writes nothing")

	if previewErr == nil {
		return
	}

	_, refusal, found := strings.Cut(previewErr.Error(), "(dry-run): ")
	g.Expect(found).To(BeTrue(), "the preview marks itself")

	_, applyErr := apply.update(t, vm2Apply())
	g.Expect(applyErr).To(MatchError(ContainSubstring(refusal)), "the apply reports the same refusal")
}

// Test792a10i1a_ReparentLuhmann_AnAnchoredLuhmannValueIsRefused verifies Scenario 792a10i1a: a note whose frontmatter
// has `luhmann: &a "12"` and `issue: *a`, to be renamed, fails the invocation naming the note, and no note file or
// sidecar is renamed or rewritten.
func Test792a10i1a_ReparentLuhmann_AnAnchoredLuhmannValueIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	harness := vm2NewAnchoredHarness(t)
	before := maps.Clone(harness.files)

	_, err := harness.update(t, vm2Apply())
	g.Expect(err).To(MatchError(cli.ErrRenameUndecodableForTest))
	g.Expect(err).To(MatchError(ContainSubstring(vm2AnchoredBasename)))
	g.Expect(harness.renames).To(BeEmpty(), "no note file or sidecar is renamed")
	g.Expect(harness.files).To(Equal(before), "no note file or sidecar is rewritten, no chunk index file is written")
}

// Test792a10i1b_ReparentLuhmann_EveryRefusalIsNamed verifies Scenario 792a10i1b: two renamed notes that would each be
// refused, one for an anchored luhmann: and one for a luhmann: the rewrite cannot reach, fail with one error that
// names both notes and the reason of each, and nothing is renamed or written.
func Test792a10i1b_ReparentLuhmann_EveryRefusalIsNamed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		anchored = "12.2026-01-02.second-note"
		stale    = "20.2026-01-03.unrelated-note"
	)

	files, names := twoRelatedTopLevelNotesFixture()
	files[vm2Vault+"/"+anchored+".md"] = []byte(vm2AnchoredNote)
	files[vm2Vault+"/"+stale+".md"] = []byte("---\ntype: fact\n\"luhmann\": \"20\"\ncreated: 2026-01-03\n---\n\nStale.\n")

	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "continuation", "7"), vm2Entry("20", "continuation", "7"))

	before := maps.Clone(harness.files)

	_, err := harness.update(t, vm2Apply())
	g.Expect(err).To(MatchError(cli.ErrRenameUndecodableForTest), "the anchored note's reason")
	g.Expect(err).To(MatchError(cli.ErrRenameStaleLuhmannForTest), "the other note's reason")

	if err == nil {
		return
	}

	reasonOf := map[string]string{}

	for line := range strings.SplitSeq(err.Error(), "\n") {
		for _, basename := range []string{anchored, stale} {
			if strings.Contains(line, basename) {
				reasonOf[basename] = line
			}
		}
	}

	g.Expect(reasonOf).To(HaveLen(2), "one error names both notes")
	g.Expect(reasonOf[anchored]).To(ContainSubstring(cli.ErrRenameUndecodableForTest.Error()))
	g.Expect(reasonOf[stale]).To(ContainSubstring(cli.ErrRenameStaleLuhmannForTest.Error()))
	g.Expect(harness.renames).To(BeEmpty())
	g.Expect(harness.files).To(Equal(before))
}

// Test792a10i1c_ReparentLuhmann_ACRLFNoteThatIsRenamedBecomesLF verifies Scenario 792a10i1c: a renamed note whose lines
// end in \r\n is written without \r\n, with its new luhmann: ID, its old basename as an alias, every other key
// decoding to its prior value, and a fresh sidecar.
func Test792a10i1c_ReparentLuhmann_ACRLFNoteThatIsRenamedBecomesLF(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldBasename = "12.2026-01-02.second-note"

	files, names := twoRelatedTopLevelNotesFixture()
	original := vm2RichNote("12")
	files[vm2Vault+"/"+oldBasename+".md"] = []byte(crlf(original))

	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "continuation", "7"))

	_, err := harness.update(t, vm2Apply())
	g.Expect(err).NotTo(HaveOccurred())

	newPath := vm2Vault + "/7a.2026-01-02.second-note.md"
	written := string(harness.files[newPath])
	g.Expect(written).NotTo(ContainSubstring("\r"), "no \\r\\n is left")

	after := vm2Frontmatter(g, written)
	before := vm2Frontmatter(g, original)

	g.Expect(after).To(HaveKeyWithValue("luhmann", "7a"))
	g.Expect(after).To(HaveKeyWithValue("aliases", []any{oldBasename}), "the alias is recorded")

	for _, key := range []string{"luhmann", "aliases"} {
		delete(before, key)
		delete(after, key)
	}

	g.Expect(after).To(Equal(before), "every other key decodes to its prior value")
	g.Expect(embed.ComputeState(reparentFileMapFS(harness.files), newPath, reparentFakeEmbedder{}.ModelID())).
		To(Equal(embed.StateOK), "the sidecar is fresh")
}

// Test792a10i1d_ReparentLuhmann_ACRLFReferrerIsRewritten verifies Scenario 792a10i1d: a CRLF note whose supersedes:
// names a renamed note is written as LF with that supersedes: note rewritten to the new basename.
func Test792a10i1d_ReparentLuhmann_ACRLFReferrerIsRewritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		oldBasename = "12.2026-01-02.second-note"
		newBasename = "7a.2026-01-02.second-note"
		referrer    = vm2Vault + "/20.2026-01-03.unrelated-note.md"
	)

	referrerLF := "---\ntype: fact\nluhmann: \"20\"\ncreated: 2026-01-03\nsupersedes:\n    - note: " + oldBasename +
		".md\n      type: narrows\n      claim: old\n---\n\nBody text.\n"

	files, names := twoRelatedTopLevelNotesFixture()
	files[referrer] = []byte(crlf(referrerLF))

	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "continuation", "7"))

	_, err := harness.update(t, vm2Apply())
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(string(harness.files[referrer])).To(
		Equal(strings.Replace(referrerLF, oldBasename+".md", newBasename+".md", 1)),
		"written as LF, with the supersedes: note rewritten and nothing else changed")
}

// Test792a10i1e_ReparentLuhmann_AnUntouchedCRLFNoteStaysAsItWas verifies Scenario 792a10i1e: CRLF notes that are
// neither renamed nor link a renamed note, the note renamed under and the other notes of the vault, keep their bytes,
// and so do their sidecars.
func Test792a10i1e_ReparentLuhmann_AnUntouchedCRLFNoteStaysAsItWas(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	untouched := []string{
		vm2Vault + "/7.2026-01-01.first-note.md",
		vm2Vault + "/20.2026-01-03.unrelated-note.md",
	}

	for _, path := range untouched {
		files[path] = []byte(crlf(string(files[path])))
	}

	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "continuation", "7"))

	before := maps.Clone(harness.files)

	_, err := harness.update(t, vm2Apply())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(harness.renames).NotTo(BeEmpty(), "the apply did rename a note")

	for _, path := range untouched {
		g.Expect(harness.files[path]).To(Equal(before[path]), path+" keeps its bytes")

		sidecar := embed.SidecarPath(path)
		g.Expect(harness.files[sidecar]).To(Equal(before[sidecar]), sidecar+" keeps its bytes")
	}
}

// Test792a10i1f_ReparentLuhmann_TheExchangeHashSurvivesTheConversion verifies Scenario 792a10i1f: a renamed note with
// LF frontmatter and a CRLF body keeps its exchange hash, and a renamed note with CRLF frontmatter hashes as its
// LF-authored equivalent.
func Test792a10i1f_ReparentLuhmann_TheExchangeHashSurvivesTheConversion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mixed := vm2ExchangeNote("12", "Body one.\n\nBody two.\n")
	mixedCRLFBody := mixed.frontmatter + crlf(mixed.body)

	allCRLF := vm2ExchangeNote("20", "Another body.\n\nWith two paragraphs.\n")
	allLFAuthored := allCRLF.frontmatter + allCRLF.body

	files, names := twoRelatedTopLevelNotesFixture()
	files[vm2Vault+"/12.2026-01-02.second-note.md"] = []byte(mixedCRLFBody)
	files[vm2Vault+"/20.2026-01-03.unrelated-note.md"] = []byte(crlf(allLFAuthored))

	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "continuation", "7"), vm2Entry("20", "continuation", "7"))

	_, err := harness.update(t, vm2Apply())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	wantMixed := vm2ExchangeHash(g, mixedCRLFBody)
	wantAllCRLF := vm2ExchangeHash(g, allLFAuthored)

	writtenMixed := harness.files[vm2Vault+"/7a.2026-01-02.second-note.md"]
	writtenAllCRLF := harness.files[vm2Vault+"/7b.2026-01-03.unrelated-note.md"]

	g.Expect(string(writtenMixed)).NotTo(ContainSubstring("\r"), "the mixed note was converted")
	g.Expect(string(writtenAllCRLF)).NotTo(ContainSubstring("\r"), "the CRLF note was converted")
	g.Expect(vm2ExchangeHash(g, string(writtenMixed))).To(Equal(wantMixed), "the mixed note keeps its hash")
	g.Expect(vm2ExchangeHash(g, string(writtenAllCRLF))).To(Equal(wantAllCRLF), "the CRLF note hashes as its LF form")
	g.Expect(vm2ExchangeHash(g, crlf(allLFAuthored))).To(Equal(vm2ExchangeHash(g, string(writtenAllCRLF))),
		"the CRLF note hashes the same before and after the conversion")
	g.Expect(wantMixed).NotTo(Equal(wantAllCRLF), "the two notes differ, so the hashes are not trivially equal")
}

// Test792a10i1g_ReparentLuhmann_AnAnchorFreeNoteRenamesWithEveryOtherKeyIntact verifies Scenario 792a10i1g: renamed
// notes whose frontmatter has no anchor each decode, carry their new ID as luhmann:, and have every top-level key
// other than aliases decoding to its earlier value.
func Test792a10i1g_ReparentLuhmann_AnAnchorFreeNoteRenamesWithEveryOtherKeyIntact(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	twelve := vm2RichNote("12")
	twenty := strings.Replace(vm2RichNote("20"), "created: 2026-01-02",
		"created: 2026-01-03\naliases:\n    - 5.2026-01-01.older", 1)
	files[vm2Vault+"/12.2026-01-02.second-note.md"] = []byte(twelve)
	files[vm2Vault+"/20.2026-01-03.unrelated-note.md"] = []byte(twenty)

	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "continuation", "7"), vm2Entry("20", "continuation", "7"))

	_, err := harness.update(t, vm2Apply())
	g.Expect(err).NotTo(HaveOccurred())

	for path, want := range map[string]struct{ original, newID string }{
		vm2Vault + "/7a.2026-01-02.second-note.md":    {twelve, "7a"},
		vm2Vault + "/7b.2026-01-03.unrelated-note.md": {twenty, "7b"},
	} {
		written, found := harness.files[path]
		g.Expect(found).To(BeTrue(), path+" exists")

		after := vm2Frontmatter(g, string(written))
		before := vm2Frontmatter(g, want.original)

		g.Expect(after).To(HaveKeyWithValue("luhmann", want.newID), path)
		g.Expect(after).To(HaveKey("aliases"), path)

		for _, key := range []string{"luhmann", "aliases"} {
			delete(before, key)
			delete(after, key)
		}

		g.Expect(after).To(Equal(before), path+": every other key decodes to its earlier value")
	}
}

// Test792a10j1a_Update_NotesMissingIdentityAreNoticed verifies Scenario 792a10j1a: a vault with a fact note that has no
// repo:, user: or vault: gets a notice naming `engram update --backfill-identity` in the report of `engram update`.
func Test792a10j1a_Update_NotesMissingIdentityAreNoticed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.fileSystem.seedFile(iu3Vault+"/1.2026-01-01.a.md", vm2FactNote(""))

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm2IdentityNotice))
}

// Test792a10j1b_Update_NoNoticeWhenNothingIsMissing verifies Scenario 792a10j1b: a vault in which every fact and
// feedback note has user: gets no identity-backfill notice.
func Test792a10j1b_Update_NoNoticeWhenNothingIsMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.fileSystem.seedFile(iu3Vault+"/1.2026-01-01.a.md", vm2FactNote("user: agent@example.com\n"))
	machine.fileSystem.seedFile(iu3Vault+"/2.2026-01-01.b.md", vm2FeedbackNote("user: agent@example.com\n"))

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	g.Expect(outcome.stdout).NotTo(ContainSubstring("--backfill-identity"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("provenance found"))
}

// Test792a10j1c_Update_NoNoticeForANoteThatCannotBeStamped verifies Scenario 792a10j1c: when user detection resolves
// empty (the machine has no git user.email and no OS username) and the only note without user: has vault:, the
// backfill could stamp nothing, so there is no identity-backfill notice.
func Test792a10j1c_Update_NoNoticeForANoteThatCannotBeStamped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.fileSystem.seedFile(iu3Vault+"/1.2026-01-01.a.md", vm2FactNote("vault: my-vault\n"))

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	g.Expect(outcome.stdout).NotTo(ContainSubstring("--backfill-identity"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("provenance found"))
}

// Test792a10j1d_Update_ACRLFNoteIsNoticed verifies Scenario 792a10j1d: a fact note missing identity whose lines end in
// \r\n counts as its LF form, so the notice appears.
func Test792a10j1d_Update_ACRLFNoteIsNoticed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	machine := iu3NewMachine()
	machine.fileSystem.seedFile(iu3Vault+"/1.2026-01-01.a.md", crlf(vm2FactNote("")))

	outcome := machine.update(t, cli.UpdateArgs{})
	g.Expect(outcome.err).NotTo(HaveOccurred())

	g.Expect(iu2Lines(outcome.stdout)).To(ContainElement(vm2IdentityNotice))
}

// unexported constants.
const (
	// vm2AnchoredBasename is the note of vm2AnchoredNote, as the two-note vault names it.
	vm2AnchoredBasename = "12.2026-01-02.second-note"
	// vm2AnchoredNote is a note whose luhmann: carries an anchor another key aliases.
	vm2AnchoredNote = "---\ntype: fact\nluhmann: &a \"12\"\nissue: *a\ncreated: 2026-01-02\n---\n\nsecond.\n"
	vm2Answers      = "/answers.json"
	vm2Chunks       = "/chunks"
	// vm2IdentityNotice is the line the update prints when the backfill could stamp a note.
	vm2IdentityNotice = "notes missing repo:/user:/vault: provenance found — run " +
		"`engram update --backfill-identity` to stamp them " +
		"(preview with `engram update --backfill-identity --dry-run`)"
	vm2Vault = "/vault"
)

// unexported variables.
var (
	errVM2ManifestWrite = errors.New("vault migration part 2 test: manifest write failed")
)

// vm2ExchangeNoteParts is a note as its frontmatter and its body, so a test can give each its own line endings.
type vm2ExchangeNoteParts struct {
	frontmatter string
	body        string
}

// vm2Harness is one `engram update --reparent-luhmann` run over an in-memory vault and chunk index: the files by
// path, the renames the rename capability recorded, and the count of the calls the ingest and prune capabilities got.
type vm2Harness struct {
	files   map[string][]byte
	renames map[string]string
	deps    cli.ReparentDeps
	touches *int
}

// update runs `engram update` with args over the harness and returns its output and error.
func (h *vm2Harness) update(t *testing.T, args cli.UpdateArgs) (string, error) {
	t.Helper()

	env := iu1Env{
		home: "/home/u", cwd: "/",
		vars: map[string]string{"ENGRAM_VAULT_PATH": vm2Vault, "ENGRAM_CHUNKS_DIR": vm2Chunks},
	}
	deps := cli.ExportNewUpdateDepsFromWithReparent(nil, nil, env, h.deps)

	var stdout strings.Builder

	err := cli.ExportRunUpdate(t.Context(), args, deps, &stdout)

	return stdout.String(), err
}

// writeAnswers writes the answers file for entries, echoing the fingerprint of the vault as it is now.
func (h *vm2Harness) writeAnswers(t *testing.T, entries ...string) {
	t.Helper()

	fingerprint := reparentDeriveFingerprint(t, h.deps)
	h.files[vm2Answers] = []byte(
		`{"reparenting":[` + strings.Join(entries, ",") + `],"fingerprint":"` + fingerprint + `"}`)
}

// vm2Apply is the arguments of `engram update --reparent-luhmann --answers <file>`.
func vm2Apply() cli.UpdateArgs {
	return cli.UpdateArgs{ReparentLuhmann: true, Answers: vm2Answers}
}

// vm2CountChunkIndexTouches wraps every capability the ingest and prune steps use, so the returned counter is the
// number of calls the chunk index got.
func vm2CountChunkIndexTouches(deps *cli.ReparentDeps) *int {
	touches := new(int)
	ingest, prune := deps.Ingest, deps.Prune
	count := func() { *touches++ }

	deps.Ingest.ReadFile = func(path string) ([]byte, error) {
		count()

		return ingest.ReadFile(path)
	}
	deps.Ingest.WriteFile = func(path string, data []byte) error {
		count()

		return ingest.WriteFile(path, data)
	}
	deps.Ingest.Stat = func(path string) (cli.SourceStat, error) {
		count()

		return ingest.Stat(path)
	}
	deps.Ingest.ListSources = func(root cli.SweepRoot) ([]string, error) {
		count()

		return ingest.ListSources(root)
	}
	deps.Ingest.Remove = func(path string) error {
		count()

		return ingest.Remove(path)
	}
	deps.Prune.ReadFile = func(path string) ([]byte, error) {
		count()

		return prune.ReadFile(path)
	}
	deps.Prune.WriteFile = func(path string, data []byte) error {
		count()

		return prune.WriteFile(path, data)
	}
	deps.Prune.Exists = func(path string) bool {
		count()

		return prune.Exists(path)
	}
	deps.Prune.ListIndexes = func(dir string) ([]string, error) {
		count()

		return prune.ListIndexes(dir)
	}
	deps.Prune.Remove = func(path string) error {
		count()

		return prune.Remove(path)
	}

	return touches
}

// vm2Entry is one answer of an answers file.
func vm2Entry(note, position, target string) string {
	return `{"note":"` + note + `","position":"` + position + `","target":"` + target + `"}`
}

// vm2ExchangeHash is the exchange hash of content.
func vm2ExchangeHash(g *WithT, content string) string {
	hash, err := cli.ExportExchangeHash([]byte(content))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(hash).To(HavePrefix(cli.ExportExchangeHashPrefix))

	return hash
}

// vm2ExchangeNote is a fact note with offered fields, as LF frontmatter and an LF body.
func vm2ExchangeNote(id, body string) vm2ExchangeNoteParts {
	return vm2ExchangeNoteParts{
		frontmatter: "---\ntype: fact\nsituation: reparenting note " + id + "\nsubject: the vault\n" +
			"predicate: keeps\nobject: its hash\nluhmann: \"" + id + "\"\ncreated: 2026-01-02\n---\n\n",
		body: body,
	}
}

// vm2FactNote is a fact note without identity, plus the extra frontmatter lines (each ending in a newline).
func vm2FactNote(extra string) string {
	return "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\nluhmann: \"1\"\n" +
		"created: 2026-01-01\nsource: agent\n" + extra + "---\n\nbody\n"
}

// vm2FeedbackNote is a feedback note without identity, plus the extra frontmatter lines (each ending in a newline).
func vm2FeedbackNote(extra string) string {
	return "---\ntype: feedback\nsituation: s\nbehavior: a\nimpact: b\naction: c\nluhmann: \"2\"\n" +
		"created: 2026-01-01\nsource: agent\n" + extra + "---\n\nbody\n"
}

// vm2Frontmatter decodes the frontmatter of content, failing the test when it does not decode as a mapping.
func vm2Frontmatter(g *WithT, content string) map[string]any {
	rest, found := strings.CutPrefix(content, "---\n")
	g.Expect(found).To(BeTrue(), "the note opens with frontmatter")

	block, _, _ := strings.Cut(rest, "\n---\n")
	fields := map[string]any{}

	// The block keeps the newline that ends its last line, as a decoder of the whole note sees it.
	g.Expect(yaml.Unmarshal([]byte(block+"\n"), &fields)).To(Succeed(), "the frontmatter decodes as a mapping")

	return fields
}

// vm2Lines splits output into its lines.
func vm2Lines(out string) []string {
	return strings.Split(strings.TrimSuffix(out, "\n"), "\n")
}

// vm2ListMD lists the notes under vault as they are now, so the listing follows the renames.
func vm2ListMD(files map[string][]byte) func(string) ([]string, error) {
	return func(vault string) ([]string, error) {
		names := []string{}

		for path := range files {
			name, found := strings.CutPrefix(path, vault+"/")
			if found && !strings.Contains(name, "/") && strings.HasSuffix(name, ".md") {
				names = append(names, name)
			}
		}

		sort.Strings(names)

		return names, nil
	}
}

// vm2NewAnchoredHarness is the two-note vault in which note 12, answered as a continuation of 7, has an anchored
// luhmann: value.
func vm2NewAnchoredHarness(t *testing.T) *vm2Harness {
	t.Helper()

	files, names := twoRelatedTopLevelNotesFixture()
	files[vm2Vault+"/"+vm2AnchoredBasename+".md"] = []byte(vm2AnchoredNote)

	harness := vm2NewHarness(files, names)
	harness.writeAnswers(t, vm2Entry("12", "continuation", "7"))

	return harness
}

// vm2NewHarness is a harness over files, with the vault listing following the renames.
func vm2NewHarness(files map[string][]byte, names []string) *vm2Harness {
	deps, renames := newReparentDeps(files, names)
	deps.Rename.ListMD = vm2ListMD(files)
	touches := vm2CountChunkIndexTouches(&deps)

	return &vm2Harness{files: files, renames: renames, deps: deps, touches: touches}
}

// vm2Preview is the arguments of `engram update --reparent-luhmann --answers <file> --dry-run`.
func vm2Preview() cli.UpdateArgs {
	return cli.UpdateArgs{ReparentLuhmann: true, Answers: vm2Answers, DryRun: true}
}

// vm2RichNote is an anchor-free fact note with keys of every shape the frontmatter holds, as LF text.
func vm2RichNote(id string) string {
	return "---\ntype: fact\nsituation: working on the vault\nsubject: reparenting\npredicate: keeps\n" +
		"object: \"every key: even this one\"\nluhmann: \"" + id + "\"\nluhmann_old: \"3\"\ncreated: 2026-01-02\n" +
		"tags:\n    - vault\n    - migration\nextra:\n    nested: [a, b]\n    count: 3\n    flag: true\n" +
		"summary: >\n    folded text\n    on two lines\nnote: |\n    first line\n    second line\n---\n\n" +
		"Body one.\n\nBody two.\n"
}

// vm2TwoPairsVault is the three-note vault plus note 21, a near twin of note 20: two pairs of similar top-level
// notes, 7 with 12 and 20 with 21.
func vm2TwoPairsVault() (map[string][]byte, []string) {
	files, names := twoRelatedTopLevelNotesFixture()

	const twin = "21.2026-01-04.twenty-first-note"

	files[vm2Vault+"/"+twin+".md"] = []byte(
		"---\ntype: fact\nluhmann: \"21\"\ncreated: 2026-01-04\n---\n\ntwenty-first note body text.\n")
	files[vm2Vault+"/"+twin+".vec.json"] = reparentSidecar([]float32{0, 0.01, 0.99})

	return files, append(names, twin+".md")
}
