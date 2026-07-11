package cli_test

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestRunVocabMigrateTags_DeleteHubFileError_LogsWarningAndExcludesFromCount
// covers deleteVaultFile's delErr branch: one hub term note's DeleteFile call
// fails (simulated permission error) — it must be logged and excluded from
// the "hub files deleted" count, while the other hub file still deletes.
func TestRunVocabMigrateTags_DeleteHubFileError_LogsWarningAndExcludesFromCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"

	const brokenDeletePath = "/vault/vocab.broken-delete.md"

	files := map[string][]byte{
		brokenDeletePath: []byte(
			"---\ntype: vocab\nterm: broken-delete\ndescription: d\nvocab_version: \"1.0\"\n" +
				"created: \"2026-07-10\"\n---\n\nd\n"),
		vault + "/vocab.ok-term.md": []byte(
			"---\ntype: vocab\nterm: ok-term\ndescription: d2\nvocab_version: \"1.0\"\ncreated: \"2026-07-10\"\n---\n\nd2\n"),
	}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)

	baseDelete := deps.DeleteFile
	deps.DeleteFile = func(path string) error {
		if path == brokenDeletePath {
			return errors.New("permission denied")
		}

		return baseDelete(path)
	}

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("hub files deleted: 1"))
	g.Expect(files).To(HaveKey(brokenDeletePath), "the failed delete must leave the file in place")
	g.Expect(strings.Join(warnings, "\n")).To(ContainSubstring("deleting " + brokenDeletePath))
}

// TestRunVocabMigrateTags_FullMigration_ThenIdempotentSecondRun is the
// end-to-end #678 Task 7 fixture: it exercises every behavior-spec step (1-6)
// over a single scratch vault, then re-runs the migration and asserts the
// second run is a byte-identical, all-zero-count no-op.
func TestRunVocabMigrateTags_FullMigration_ThenIdempotentSecondRun(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault, files := buildMigrateTagsFixture(t)

	// ContentHash-before snapshot for every surviving member (BEFORE any
	// mutation) — invariance is asserted AFTER the first run below.
	survivingMembers := []string{
		"300.2026-07-08.mixed-shape-member.md",
		"qa.2026-07-08.sample-question.a.md",
		"301.2026-07-08.plain-fact-note.md",
		"302.2026-07-08.synthetic-inconsistency.md",
		"303.2026-07-08.already-migrated-member.md",
	}
	hashBefore := snapshotContentHashes(g, files, vault, survivingMembers)

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)

	var stdout strings.Builder

	firstErr := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(firstErr).NotTo(HaveOccurred())

	if firstErr != nil {
		return
	}

	// ── Step 6: exact counts summary ──
	g.Expect(stdout.String()).To(Equal(
		"members rewritten: 4, definitions minted: 2, family note: minted, " +
			"hub files deleted: 3, sidecars deleted: 4\n"))

	// ── (a) mixed-shape member: byte-precise post-state ──
	got := string(files[vault+"/300.2026-07-08.mixed-shape-member.md"])
	want := "---\n" +
		"type: fact\n" +
		"tier: L2\n" +
		"situation: classifying route dispatch evidence by work-kind, or auditing work-kind tags\n" +
		"subject: the work-kind tag family\n" +
		"predicate: classifies\n" +
		"object: 'route dispatch evidence notes by shape'\n" +
		"luhmann: \"300\"\n" +
		"created: \"2026-07-08\"\n" +
		"source: test fixture (#678 Task 7)\n" +
		"tags:\n" +
		"    - work-kind\n" +
		"    - vocab/lever-tracking\n" +
		"    - vocab/cost-optimization\n" +
		"---\n" +
		"\n" +
		"Information learned: when in classifying route dispatch evidence by work-kind, " +
		"the work-kind tag family classifies dispatch evidence.\n"
	g.Expect(got).To(Equal(want),
		"fixture (a): tags must read [work-kind, vocab/lever-tracking, vocab/cost-optimization] in order")

	// ── (b) qa answer note: byte-precise post-state ──
	got = string(files[vault+"/qa.2026-07-08.sample-question.a.md"])
	want = "---\n" +
		"type: qa-answer\n" +
		"date: \"2026-07-08\"\n" +
		"answers: qa.2026-07-08.sample-question.q\n" +
		"certainty: high\n" +
		"source: test fixture (#678 Task 7)\n" +
		"tags:\n" +
		"    - vocab/retrieval-design\n" +
		"---\n" +
		"\n" +
		"Answer body text explaining the design choice.\n" +
		"\n" +
		"Answers: [[qa.2026-07-08.sample-question.q]]\n"
	g.Expect(got).To(Equal(want), "fixture (b): qa answer note vocab: key migrates to tags: vocab/retrieval-design")

	// ── (c) plain fact note: byte-precise post-state ──
	got = string(files[vault+"/301.2026-07-08.plain-fact-note.md"])
	want = "---\n" +
		"type: fact\n" +
		"tier: L2\n" +
		"situation: s\n" +
		"subject: subj\n" +
		"predicate: covers\n" +
		"object: desc\n" +
		"luhmann: \"301\"\n" +
		"created: \"2026-07-08\"\n" +
		"source: test fixture (#678 Task 7)\n" +
		"tags:\n" +
		"    - vocab/retrieval-design\n" +
		"---\n" +
		"\n" +
		"Information learned: when in s, subj covers desc.\n"
	g.Expect(got).To(Equal(want), "fixture (c): plain fact note vocab: key migrates to tags:")

	// ── (d) synthetic channel-inconsistency: body-line terms DISCARDED ──
	got = string(files[vault+"/302.2026-07-08.synthetic-inconsistency.md"])
	want = "---\n" +
		"type: fact\n" +
		"tier: L2\n" +
		"situation: s2\n" +
		"subject: subj2\n" +
		"predicate: covers\n" +
		"object: desc2\n" +
		"luhmann: \"302\"\n" +
		"created: \"2026-07-08\"\n" +
		"source: test fixture (#678 Task 7)\n" +
		"---\n" +
		"\n" +
		"Information learned: when in s2, subj2 covers desc2.\n"
	g.Expect(got).To(Equal(want), "fixture (d): no vocab tags gained, Vocab: line stripped, stale-term never assigned")
	g.Expect(got).NotTo(ContainSubstring("vocab"), "fixture (d) must carry no vocab tags at all")

	// ── (e) already-migrated member: untouched ──
	g.Expect(string(files[vault+"/303.2026-07-08.already-migrated-member.md"])).To(Equal(
		"---\n"+
			"type: fact\n"+
			"tier: L2\n"+
			"situation: s3\n"+
			"subject: subj3\n"+
			"predicate: covers\n"+
			"object: desc3\n"+
			"luhmann: \"303\"\n"+
			"created: \"2026-07-08\"\n"+
			"source: test fixture (#678 Task 7)\n"+
			"tags:\n"+
			"    - vocab/lever-tracking\n"+
			"---\n"+
			"\n"+
			"Information learned: when in s3, subj3 covers desc3.\n"),
		"fixture (e): already-migrated member must be byte-identical")

	assertContentHashInvariance(g, files, vault, hashBefore)
	assertDefinitionsAndFamilyNoteMinted(g, files, vault)

	// ── hub files + hub sidecars gone (including the orphan) ──
	for _, gone := range []string{
		"vocab.lever-tracking.md", "vocab.cost-optimization.md", "vocab.index.md",
		"vocab.lever-tracking.vec.json", "vocab.cost-optimization.vec.json",
		"vocab.index.vec.json", "vocab.orphan-term.vec.json",
	} {
		g.Expect(files).NotTo(HaveKey(vault+"/"+gone), "%s must be deleted", gone)
	}

	g.Expect(files).To(HaveKey(vault+"/300.2026-07-08.mixed-shape-member.vec.json"),
		"a member's own sidecar (no vocab. prefix) must survive the sidecar sweep")

	assertSecondRunIsIdempotentNoOp(t, g, deps, vault, files)
}

// TestRunVocabMigrateTags_IndexNoteMissingVersionKey_FallsBackToDefault
// covers vocabVersionFromNoteBytes' empty-VocabVersion branch: vocab.index.md
// has parseable frontmatter but no vocab_version key.
func TestRunVocabMigrateTags_IndexNoteMissingVersionKey_FallsBackToDefault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"
	files := map[string][]byte{
		vault + "/vocab.index.md": []byte("---\ntype: vocab-index\ncreated: \"2026-07-10\"\n---\n\nBody.\n"),
	}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("defaulting vocab_version to 1.0"))
}

// TestRunVocabMigrateTags_IndexNoteUnparseable_FallsBackToDefault covers
// vocabVersionFromNoteBytes' splitFrontmatter-fails branch: vocab.index.md
// exists but has no parseable frontmatter at all.
func TestRunVocabMigrateTags_IndexNoteUnparseable_FallsBackToDefault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"
	files := map[string][]byte{
		vault + "/vocab.index.md": []byte("not a valid frontmatter note at all\n"),
	}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("defaulting vocab_version to 1.0"))
}

// TestRunVocabMigrateTags_ListMDError verifies a vault-listing failure is
// wrapped and returned.
func TestRunVocabMigrateTags_ListMDError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return nil, errors.New("permission denied") },
		Now:    func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
	}

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: "/vault"}, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

// TestRunVocabMigrateTags_ListVecJSONError_LogsWarningAndSkipsSweep covers
// deleteHubSidecars' listErr branch.
func TestRunVocabMigrateTags_ListVecJSONError_LogsWarningAndSkipsSweep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"
	files := map[string][]byte{}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)
	deps.ListVecJSON = func(string) ([]string, error) { return nil, errors.New("permission denied") }

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("sidecars deleted: 0"))
	g.Expect(strings.Join(warnings, "\n")).To(ContainSubstring("listing sidecars"))
}

// TestRunVocabMigrateTags_LockError verifies the lock-acquisition error is
// wrapped and returned, aborting before any vault scan.
func TestRunVocabMigrateTags_LockError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return nil, errors.New("lock held by another process") },
	}

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: "/vault"}, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
}

// TestRunVocabMigrateTags_MemberEdgeCases_SkippedOrWarned covers
// migrateMembers' three untested guard branches in one fixture: a member
// note ReadFile fails on, a member note with no parseable frontmatter at
// all, and a member note whose WriteFile fails after a legacy vocab: key is
// found. None of the three increments "members rewritten"; the write
// failure is logged.
func TestRunVocabMigrateTags_MemberEdgeCases_SkippedOrWarned(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"

	const unreadablePath = "/vault/400.2026-07-08.unreadable-member.md"

	const brokenWritePath = "/vault/401.2026-07-08.write-fails-member.md"

	files := map[string][]byte{
		unreadablePath: []byte("placeholder"),
		brokenWritePath: []byte(
			"---\ntype: fact\nvocab: [retrieval-design]\n---\n\nBody.\n\nVocab: [[vocab.retrieval-design]]\n"),
		vault + "/402.2026-07-08.no-frontmatter-member.md": []byte("Just a body, no frontmatter at all.\n"),
	}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)

	baseRead := deps.ReadFile
	deps.ReadFile = func(path string) ([]byte, error) {
		if path == unreadablePath {
			return nil, errors.New("permission denied")
		}

		return baseRead(path)
	}

	baseWrite := deps.WriteFile
	deps.WriteFile = func(path string, data []byte) error {
		if path == brokenWritePath {
			return errors.New("disk full")
		}

		return baseWrite(path, data)
	}

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("members rewritten: 0"))
	g.Expect(strings.Join(warnings, "\n")).To(ContainSubstring("rewriting " + brokenWritePath))
}

// TestRunVocabMigrateTags_MintDefinitionWriteError_LogsWarningAndContinues
// covers migrateTermDefinitions' mintErr branch: the definition note's
// WriteFile call fails.
func TestRunVocabMigrateTags_MintDefinitionWriteError_LogsWarningAndContinues(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"
	files := map[string][]byte{
		vault + "/vocab.broken-mint.md": []byte(
			"---\ntype: vocab\nterm: broken-mint\ndescription: d\nvocab_version: \"1.0\"\ncreated: \"2026-07-10\"\n---\n\nd\n"),
	}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)
	deps.WriteFile = func(string, []byte) error { return errors.New("disk full") }

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("definitions minted: 0"))
	g.Expect(strings.Join(warnings, "\n")).To(ContainSubstring("minting definition for broken-mint"))
}

// TestRunVocabMigrateTags_NilDeleteFile verifies that a nil DeleteFile skips
// hub deletions without failing the whole command (mirrors
// TestRunVocabRefit_NilDeleteFile's convention).
func TestRunVocabMigrateTags_NilDeleteFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"
	files := map[string][]byte{
		vault + "/vocab.orphan.md": []byte(
			"---\ntype: vocab\nterm: orphan\ndescription: d\nvocab_version: \"1.0\"\ncreated: \"2026-07-10\"\n---\n\nd\n"),
	}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)
	deps.DeleteFile = nil

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred(), "RunVocabMigrateTags with nil DeleteFile must succeed (skip deletions)")
	g.Expect(stdout.String()).To(ContainSubstring("hub files deleted: 0, sidecars deleted: 0"))
}

// TestRunVocabMigrateTags_NilListVecJSON_SkipsSidecarSweep verifies that a
// nil ListVecJSON dep skips the sidecar sweep without failing the command.
func TestRunVocabMigrateTags_NilListVecJSON_SkipsSidecarSweep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"
	files := map[string][]byte{}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)
	deps.ListVecJSON = nil

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("sidecars deleted: 0"))
}

// TestRunVocabMigrateTags_NoIndexOrFamilyNote_DefaultsVersionWithWarning
// covers behavior-spec step 1's final fallback: neither vocab.index.md nor a
// vocab-definition family note exists, so the version defaults to "1.0" and a
// warning is printed.
func TestRunVocabMigrateTags_NoIndexOrFamilyNote_DefaultsVersionWithWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files := map[string][]byte{}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: "/vault"}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout.String()).To(ContainSubstring(
		"warning: vocab migrate-tags: no vocab.index.md or vocab-definition family note found; " +
			"defaulting vocab_version to 1.0"))
	g.Expect(stdout.String()).To(ContainSubstring(
		"members rewritten: 0, definitions minted: 0, family note: minted, hub files deleted: 0, sidecars deleted: 0"))
}

// TestRunVocabMigrateTags_TermNoteWithExistingDefinition_SkippedNotReMinted
// covers migrateTermDefinitions' definitionNoteExistsForTerm branch: a
// partial-migration state where the old-shape term note still exists AND its
// definition note was already minted. The existing definition must be left
// untouched and not counted as newly minted.
func TestRunVocabMigrateTags_TermNoteWithExistingDefinition_SkippedNotReMinted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"

	const existingDefinitionPath = "/vault/1.2026-07-10.vocab-already-defined-definition.md"

	files := map[string][]byte{
		vault + "/vocab.already-defined.md": []byte(
			"---\ntype: vocab\nterm: already-defined\ndescription: d\nvocab_version: \"1.0\"\n" +
				"created: \"2026-07-10\"\n---\n\nd\n"),
		existingDefinitionPath: []byte(
			"---\ntype: fact\ntier: L2\nsituation: s\nsubject: subj\npredicate: covers\n" +
				"object: existing description\nluhmann: \"1\"\ncreated: \"2026-07-10\"\nsource: prior mint\n" +
				"tags:\n    - vocab\n---\n\nInformation learned: prior mint body.\n"),
	}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("definitions minted: 0"))
	g.Expect(string(files[existingDefinitionPath])).To(ContainSubstring("existing description"),
		"the existing definition note must be left untouched, not re-minted")
}

// TestRunVocabMigrateTags_UnreadableAndMalformedTermNotes_SkippedWithWarning
// covers migrateTermDefinitions' two skip branches: a term note ReadFile
// fails on (simulated permission error) and a term note with malformed
// frontmatter (no term: key). Neither mints a definition, both are logged,
// and both are still swept as hub files in step 5.
func TestRunVocabMigrateTags_UnreadableAndMalformedTermNotes_SkippedWithWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"

	const brokenReadPath = "/vault/vocab.broken-read.md"

	files := map[string][]byte{
		brokenReadPath:                  []byte("placeholder"),
		vault + "/vocab.no-term-key.md": []byte("---\ntype: vocab\ndescription: missing term key\n---\n\nBody.\n"),
	}

	var warnings []string

	when := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	deps := newInMemoryVocabDeps(files, &warnings, when)

	baseRead := deps.ReadFile
	deps.ReadFile = func(path string) ([]byte, error) {
		if path == brokenReadPath {
			return nil, errors.New("permission denied")
		}

		return baseRead(path)
	}

	var stdout strings.Builder

	err := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("definitions minted: 0"))
	g.Expect(stdout.String()).To(ContainSubstring("hub files deleted: 2"))

	joined := strings.Join(warnings, "\n")
	g.Expect(joined).To(ContainSubstring("skipping unreadable term note vocab.broken-read.md"))
	g.Expect(joined).To(ContainSubstring("skipping unreadable term note vocab.no-term-key.md"))
}

// assertContentHashInvariance re-hashes every surviving member AFTER the
// migration and asserts it matches the pre-migration hash — frontmatter and
// machine lines (vocab:/Vocab:) are excluded from ContentHash by
// construction, so stripping them must never change it.
func assertContentHashInvariance(g Gomega, files map[string][]byte, vault string, hashBefore map[string]string) {
	for name, before := range hashBefore {
		raw, ok := files[vault+"/"+name]
		g.Expect(ok).To(BeTrue(), "surviving member %s must still exist after migration", name)

		if !ok {
			continue
		}

		after := embed.ContentHash(raw)
		g.Expect(after).To(Equal(before),
			"ContentHash must be unchanged for %s (frontmatter+machine lines excluded)", name)
	}
}

// assertDefinitionsAndFamilyNoteMinted asserts the two term definitions
// (lever-tracking with exemplars, cost-optimization without) and the family
// note were minted with the correct slugs/fields/tags/body content.
func assertDefinitionsAndFamilyNoteMinted(g Gomega, files map[string][]byte, vault string) {
	leverContent, leverOK := findNoteBySlug(files, vault, "vocab-lever-tracking-definition")
	g.Expect(leverOK).To(BeTrue(), "vocab-lever-tracking-definition must be minted")

	if leverOK {
		g.Expect(leverContent).To(ContainSubstring("type: fact"))
		g.Expect(leverContent).To(ContainSubstring("tags:\n    - vocab\n"))
		g.Expect(leverContent).To(ContainSubstring("the lever-tracking vocab term"))
		g.Expect(leverContent).To(ContainSubstring("Identifying, measuring, and validating optimization levers."))
		g.Expect(leverContent).To(ContainSubstring("source: 'migrated from vocab.lever-tracking.md under #678'"))
		g.Expect(leverContent).To(ContainSubstring("Exemplars:"))
		g.Expect(leverContent).To(ContainSubstring("- quantifying a lever's reach and ceiling before shipping it"))
		g.Expect(leverContent).To(ContainSubstring("- maintaining an honest ledger of what shipped"))
	}

	costContent, costOK := findNoteBySlug(files, vault, "vocab-cost-optimization-definition")
	g.Expect(costOK).To(BeTrue(), "vocab-cost-optimization-definition must be minted")

	if costOK {
		g.Expect(costContent).To(ContainSubstring("the cost-optimization vocab term"))
		g.Expect(costContent).To(ContainSubstring("Reducing LLM and computation costs without capability loss."))
		g.Expect(costContent).To(ContainSubstring("source: 'migrated from vocab.cost-optimization.md under #678'"))
		g.Expect(costContent).NotTo(ContainSubstring("Exemplars:"), "no exemplars in the old note means none minted")
	}

	familyContent, familyOK := findNoteBySlug(files, vault, "vocab-definition")
	g.Expect(familyOK).To(BeTrue(), "vocab-definition family note must be minted")

	if familyOK {
		g.Expect(familyContent).To(ContainSubstring("tags:\n    - vocab\n"))
		g.Expect(familyContent).To(ContainSubstring(`vocab_version: "6.0"`))
		g.Expect(familyContent).NotTo(ContainSubstring("lever-tracking"), "family note must never enumerate terms")
		g.Expect(familyContent).NotTo(ContainSubstring("cost-optimization"), "family note must never enumerate terms")
	}
}

// assertSecondRunIsIdempotentNoOp re-runs the migration over the same vault
// and asserts all-zero counts, "family note: present", and a byte-identical
// vault (no entry added, removed, or mutated).
func assertSecondRunIsIdempotentNoOp(
	t *testing.T,
	g Gomega,
	deps cli.VocabDeps,
	vault string,
	files map[string][]byte,
) {
	t.Helper()

	before := cloneFileMap(files)

	var stdout strings.Builder

	secondErr := cli.RunVocabMigrateTags(t.Context(), cli.VocabMigrateArgs{Vault: vault}, deps, &stdout)
	g.Expect(secondErr).NotTo(HaveOccurred())

	if secondErr != nil {
		return
	}

	g.Expect(stdout.String()).To(Equal(
		"members rewritten: 0, definitions minted: 0, family note: present, " +
			"hub files deleted: 0, sidecars deleted: 0\n"))

	g.Expect(files).To(HaveLen(len(before)), "second run must not add or remove any vault entry")

	for path, data := range before {
		g.Expect(files).To(HaveKeyWithValue(path, data), "second run must leave %s byte-identical", path)
	}
}

// ── #678 Task 7: engram vocab migrate-tags ───────────────────────────────────

// buildMigrateTagsFixture builds the shared #678 Task 7 scratch vault: the
// real 204-pattern mixed-shape member (a), a qa answer note (b), a plain fact
// note (c), a synthetic channel-inconsistency note (d — zero real instances,
// defensive only), an already-migrated member (e — must never be touched),
// two old-shape vocab.<term>.md term notes (one with Exemplars, one without),
// vocab.index.md (vocab_version "6.0"), and one orphan hub sidecar with no
// surviving .md counterpart. Returns the vault path and the live file map.
func buildMigrateTagsFixture(t *testing.T) (string, map[string][]byte) {
	t.Helper()

	const vault = "/vault"

	files := map[string][]byte{
		// (a) mixed-shape member — the real 204 pattern: block tags: PLUS
		// inline vocab: PLUS a Vocab: body line.
		vault + "/300.2026-07-08.mixed-shape-member.md": []byte(
			"---\n" +
				"type: fact\n" +
				"tier: L2\n" +
				"situation: classifying route dispatch evidence by work-kind, or auditing work-kind tags\n" +
				"subject: the work-kind tag family\n" +
				"predicate: classifies\n" +
				"object: 'route dispatch evidence notes by shape'\n" +
				"luhmann: \"300\"\n" +
				"created: \"2026-07-08\"\n" +
				"source: test fixture (#678 Task 7)\n" +
				"tags:\n" +
				"    - work-kind\n" +
				"vocab: [lever-tracking, cost-optimization]\n" +
				"---\n" +
				"\n" +
				"Information learned: when in classifying route dispatch evidence by work-kind, " +
				"the work-kind tag family classifies dispatch evidence.\n" +
				"\n" +
				"Vocab: [[vocab.lever-tracking]], [[vocab.cost-optimization]]\n",
		),
		// (b) qa answer note — qa. filename prefix, type: qa-answer.
		vault + "/qa.2026-07-08.sample-question.a.md": []byte(
			"---\n" +
				"type: qa-answer\n" +
				"date: \"2026-07-08\"\n" +
				"answers: qa.2026-07-08.sample-question.q\n" +
				"certainty: high\n" +
				"source: test fixture (#678 Task 7)\n" +
				"vocab: [retrieval-design]\n" +
				"---\n" +
				"\n" +
				"Answer body text explaining the design choice.\n" +
				"\n" +
				"Answers: [[qa.2026-07-08.sample-question.q]]\n" +
				"\n" +
				"Vocab: [[vocab.retrieval-design]]\n",
		),
		// (c) plain fact note.
		vault + "/301.2026-07-08.plain-fact-note.md": []byte(
			"---\n" +
				"type: fact\n" +
				"tier: L2\n" +
				"situation: s\n" +
				"subject: subj\n" +
				"predicate: covers\n" +
				"object: desc\n" +
				"luhmann: \"301\"\n" +
				"created: \"2026-07-08\"\n" +
				"source: test fixture (#678 Task 7)\n" +
				"vocab: [retrieval-design]\n" +
				"---\n" +
				"\n" +
				"Information learned: when in s, subj covers desc.\n" +
				"\n" +
				"Vocab: [[vocab.retrieval-design]]\n",
		),
		// (d) synthetic channel-inconsistency: a Vocab: body line with NO
		// vocab: key — zero real instances, defensive branch only.
		vault + "/302.2026-07-08.synthetic-inconsistency.md": []byte(
			"---\n" +
				"type: fact\n" +
				"tier: L2\n" +
				"situation: s2\n" +
				"subject: subj2\n" +
				"predicate: covers\n" +
				"object: desc2\n" +
				"luhmann: \"302\"\n" +
				"created: \"2026-07-08\"\n" +
				"source: test fixture (#678 Task 7)\n" +
				"---\n" +
				"\n" +
				"Information learned: when in s2, subj2 covers desc2.\n" +
				"\n" +
				"Vocab: [[vocab.stale-term]]\n",
		),
		// (e) already-migrated member — must be byte-identical after migration.
		vault + "/303.2026-07-08.already-migrated-member.md": []byte(
			"---\n" +
				"type: fact\n" +
				"tier: L2\n" +
				"situation: s3\n" +
				"subject: subj3\n" +
				"predicate: covers\n" +
				"object: desc3\n" +
				"luhmann: \"303\"\n" +
				"created: \"2026-07-08\"\n" +
				"source: test fixture (#678 Task 7)\n" +
				"tags:\n" +
				"    - vocab/lever-tracking\n" +
				"---\n" +
				"\n" +
				"Information learned: when in s3, subj3 covers desc3.\n",
		),
		// Old-shape term note WITH an Exemplars section.
		vault + "/vocab.lever-tracking.md": []byte(
			"---\n" +
				"type: vocab\n" +
				"term: lever-tracking\n" +
				"description: Identifying, measuring, and validating optimization levers.\n" +
				"vocab_version: \"6.0\"\n" +
				"created: \"2026-07-01\"\n" +
				"---\n" +
				"\n" +
				"Identifying, measuring, and validating optimization levers.\n" +
				"\n" +
				"Exemplars:\n" +
				"- quantifying a lever's reach and ceiling before shipping it\n" +
				"- maintaining an honest ledger of what shipped\n",
		),
		// Old-shape term note WITHOUT an Exemplars section.
		vault + "/vocab.cost-optimization.md": []byte(
			"---\n" +
				"type: vocab\n" +
				"term: cost-optimization\n" +
				"description: Reducing LLM and computation costs without capability loss.\n" +
				"vocab_version: \"6.0\"\n" +
				"created: \"2026-07-01\"\n" +
				"---\n" +
				"\n" +
				"Reducing LLM and computation costs without capability loss.\n",
		),
		vault + "/vocab.index.md": []byte(
			"---\n" +
				"type: vocab-index\n" +
				"vocab_version: \"6.0\"\n" +
				"created: \"2026-07-01\"\n" +
				"---\n" +
				"\n" +
				"[[vocab.lever-tracking]] — Identifying, measuring, and validating optimization levers. — 1 members\n" +
				"[[vocab.cost-optimization]] — Reducing LLM and computation costs without capability loss. — 1 members\n",
		),
		vault + "/vocab.lever-tracking.vec.json":    mustMarshalSidecarWithBodyVector(t, []float32{0.1, 0.2}),
		vault + "/vocab.cost-optimization.vec.json": mustMarshalSidecarWithBodyVector(t, []float32{0.2, 0.3}),
		vault + "/vocab.index.vec.json":             mustMarshalSidecarWithBodyVector(t, []float32{0.3, 0.4}),
		// Orphan hub sidecar — no surviving .md counterpart.
		vault + "/vocab.orphan-term.vec.json": mustMarshalSidecarWithBodyVector(t, []float32{0.4, 0.5}),
		// A member's own sidecar — NOT a hub sidecar (no vocab. prefix) — must
		// survive the sidecar sweep untouched.
		vault + "/300.2026-07-08.mixed-shape-member.vec.json": mustMarshalSidecarWithBodyVector(t, []float32{0.6, 0.7}),
	}

	return vault, files
}

// cloneFileMap returns a value-copied snapshot of files for a later
// byte-identical comparison.
func cloneFileMap(files map[string][]byte) map[string][]byte {
	clone := make(map[string][]byte, len(files))
	for path, data := range files {
		clone[path] = append([]byte(nil), data...)
	}

	return clone
}

// findNoteBySlug scans files for the vault entry whose filename slug
// (cli.ExportSlugFromNoteFilename) equals slug, returning its content.
func findNoteBySlug(files map[string][]byte, vault, slug string) (string, bool) {
	prefix := vault + "/"

	for path, data := range files {
		name, ok := strings.CutPrefix(path, prefix)
		if !ok || !strings.HasSuffix(name, ".md") {
			continue
		}

		if cli.ExportSlugFromNoteFilename(name) == slug {
			return string(data), true
		}
	}

	return "", false
}

// listVaultFilesByExt returns the basenames directly inside vault (no nested
// paths) whose name has the given suffix, sorted for determinism.
func listVaultFilesByExt(files map[string][]byte, vault, ext string) []string {
	prefix := vault + "/"
	out := make([]string, 0, len(files))

	for path := range files {
		name, ok := strings.CutPrefix(path, prefix)
		if !ok || strings.Contains(name, "/") || !strings.HasSuffix(name, ext) {
			continue
		}

		out = append(out, name)
	}

	sort.Strings(out)

	return out
}

// newInMemoryVocabDeps wires cli.VocabDeps to an in-memory file map keyed by
// full path (both .md notes and .vec.json sidecars share the map — their
// paths never collide). ListMD/ListVecJSON derive their listing live from the
// map on every call, so writes/deletes made during a run (and across repeat
// Run calls sharing the same map) are immediately visible — required for the
// idempotent-second-run assertion.
func newInMemoryVocabDeps(files map[string][]byte, warnings *[]string, when time.Time) cli.VocabDeps {
	return cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(vault string) ([]string, error) {
			return listVaultFilesByExt(files, vault, ".md"), nil
		},
		ListVecJSON: func(vault string) ([]string, error) {
			return listVaultFilesByExt(files, vault, ".vec.json"), nil
		},
		ReadFile: func(path string) ([]byte, error) {
			data, ok := files[path]
			if !ok {
				return nil, &testNotFoundError{path: path}
			}

			return data, nil
		},
		WriteFile: func(path string, data []byte) error {
			files[path] = data

			return nil
		},
		DeleteFile: func(path string) error {
			if _, ok := files[path]; !ok {
				return &testNotFoundError{path: path}
			}

			delete(files, path)

			return nil
		},
		WriteSidecar: func(path string, data []byte) error {
			files[path] = data

			return nil
		},
		Embedder: &fakeEmbedder{},
		LogWarning: func(format string, args ...any) {
			*warnings = append(*warnings, fmt.Sprintf(format, args...))
		},
		Now: func() time.Time { return when },
	}
}

// snapshotContentHashes computes embed.ContentHash for each named vault entry
// (BEFORE any mutation), failing the assertion if a fixture entry is missing.
func snapshotContentHashes(g Gomega, files map[string][]byte, vault string, names []string) map[string]string {
	hashes := make(map[string]string, len(names))

	for _, name := range names {
		raw, ok := files[vault+"/"+name]
		g.Expect(ok).To(BeTrue(), "fixture must include %s", name)

		if !ok {
			continue
		}

		hashes[name] = embed.ContentHash(raw)
	}

	return hashes
}
