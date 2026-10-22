package cli_test

// Scenario-ID tests of the Controlled Vocabulary product capability, part 1
// (docs/specs/product/controlled-vocabulary.md): 792a6a Definition notes, 792a6b The family note, 792a6c Categorical
// tags share one list, 792a6d Write-time vocabulary tagging, 792a6e Tagging is skipped without a vocabulary or an
// embedding, 792a6f CRLF notes, 792a6g Bootstrapping a vocabulary, 792a6i Proposing a term, 792a6j Deriving the
// vocabulary, 792a6k Term names stay stable, 792a6l Dry run and 792a6m Naming requests.
//
// Every scenario drives the real command path (cli.Targets over the production compositions) over an in-memory vault
// (cv1MemFS, a cli.EdgeFS) and a deterministic topic embedder (cv1TopicEmbedder): no real filesystem, home, git or
// embedding model. The notes of a scenario are written by `engram learn fact` itself, the vocabulary by `engram vocab
// bootstrap`, so every note, sidecar and definition note is in the shape the system produces.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a6a1a_VocabBootstrap_ADefinitionNoteCarriesItsSelfTag verifies 792a6a1a (controlled-vocabulary.md:50):
// bootstrapping a seed file that names the term cooking leaves a definition note
// <id>.<date>.vocab-cooking-definition.md with the tags [vocab, vocab/cooking], the description as its object, an
// Exemplars: list holding the exemplar, and a sidecar.
func Test792a6a1a_VocabBootstrap_ADefinitionNoteCarriesItsSelfTag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault and a seed file naming the term cooking with the description "recipes and kitchens" and one exemplar
	vault := newCV1Vault()
	seed := cli.SeedTerm{Term: "cooking", Description: "recipes and kitchens", Exemplars: []string{cv1Exemplar}}

	// When the user runs engram vocab bootstrap --seed with that file
	result := vault.bootstrap(seed)
	g.Expect(result.stderr).To(BeEmpty())

	// Then the vault holds the definition note with the self-tag, the object, the Exemplars: list and a sidecar
	names := vault.namesMatching(`^[0-9a-z]+\.` + cv1Today + `\.vocab-cooking-definition\.md$`)
	g.Expect(names).To(HaveLen(1))

	if len(names) != 1 {
		return
	}

	content := vault.read(names[0])
	front := parseCV1Front(content)
	g.Expect(front.Tags).To(Equal([]string{"vocab", "vocab/cooking"}))
	g.Expect(front.Object).To(Equal("recipes and kitchens"))
	g.Expect(content).To(MatchRegexp(`(?m)^Exemplars:\n- ` + regexp.QuoteMeta(cv1Exemplar) + `\n?$`))

	sidecar, sidecarErr := embed.UnmarshalSidecar([]byte(vault.read(embed.SidecarPath(names[0]))))
	g.Expect(sidecarErr).NotTo(HaveOccurred())

	if sidecarErr != nil {
		return
	}

	g.Expect(sidecar.EmbeddingModelID).To(Equal(cv1ModelID))
	g.Expect(sidecar.BodyVector).To(HaveLen(cv1TopicCount))
}

// Test792a6a1b_VocabRefitAndBootstrap_ADefinitionIsNeverAssignedATerm verifies 792a6a1b (controlled-vocabulary.md:55):
// after a refit or a repeated bootstrap over a vault whose notes sit where the terms do, the cooking definition note's
// tags are still exactly [vocab, vocab/cooking]. The definition's own vector lies on its term's axis, so an assignment
// that did not leave definitions out would rewrite its tags.
func Test792a6a1b_VocabRefitAndBootstrap_ADefinitionIsNeverAssignedATerm(t *testing.T) {
	t.Parallel()

	for name, rerun := range map[string]func(*cv1Vault) cv1Result{
		"engram vocab bootstrap": func(vault *cv1Vault) cv1Result { return vault.bootstrap(cv1Cooking, cv1BuildTooling) },
		"engram vocab refit":     func(vault *cv1Vault) cv1Result { return vault.refit() },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a vault whose vocabulary holds the term cooking (proposed, so that no tagging pass has run yet),
			// over notes about cooking and about builds
			vault := newCV1TwoTopicVault()
			vault.bootstrap()
			vault.run("vocab", "propose", "--term", cv1Cooking.Term, "--description", cv1Cooking.Description)
			vault.run("vocab", "propose", "--term", cv1BuildTooling.Term, "--description", cv1BuildTooling.Description)

			definition := vault.requireNote(g, `\.vocab-cooking-definition\.md$`)
			g.Expect(parseCV1Front(vault.read(definition)).Tags).To(Equal([]string{"vocab", "vocab/cooking"}))

			// When the user runs engram vocab refit or engram vocab bootstrap
			result := rerun(vault)
			g.Expect(result.stderr).To(BeEmpty())

			// Then the cooking definition note's tags are still exactly [vocab, vocab/cooking]
			g.Expect(parseCV1Front(vault.read(definition)).Tags).To(Equal([]string{"vocab", "vocab/cooking"}))
		})
	}
}

// Test792a6b1a_VocabBootstrap_CreatesTheFamilyNote verifies 792a6b1a (controlled-vocabulary.md:77): bootstrapping a
// vault with no vocabulary over a seed file naming two terms leaves one note <id>.<date>.vocab-definition.md tagged
// only vocab, carrying vocab_version "1.0", whose text names neither term.
func Test792a6b1a_VocabBootstrap_CreatesTheFamilyNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with no vocabulary
	vault := newCV1Vault()

	// When the user runs engram vocab bootstrap with a seed file naming two terms
	result := vault.bootstrap(cv1Cooking, cv1BuildTooling)
	g.Expect(result.stderr).To(BeEmpty())

	// Then the vault holds one family note tagged only vocab, carrying vocab_version "1.0", naming neither term
	names := vault.namesMatching(`^[0-9a-z]+\.` + cv1Today + `\.vocab-definition\.md$`)
	g.Expect(names).To(HaveLen(1))

	if len(names) != 1 {
		return
	}

	content := vault.read(names[0])
	front := parseCV1Front(content)
	g.Expect(front.Tags).To(Equal([]string{"vocab"}))
	g.Expect(front.VocabVersion).To(Equal("1.0"))
	g.Expect(content).NotTo(ContainSubstring(cv1Cooking.Term))
	g.Expect(content).NotTo(ContainSubstring(cv1BuildTooling.Term))
}

// Test792a6b1b_VocabProposeAndRefit_TheVersionFollowsEach verifies 792a6b1b (controlled-vocabulary.md:82): over a
// vault whose family note carries vocab_version "2.0", engram vocab stats reports version 2.1 after an engram vocab
// propose and 3.0 after an engram vocab refit that applies.
func Test792a6b1b_VocabProposeAndRefit_TheVersionFollowsEach(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose family note carries vocab_version "2.0" (the family note a bootstrap wrote, at that version)
	vault := newCV1TwoTopicVault()
	vault.bootstrap(cv1Cooking, cv1BuildTooling)
	vault.setFamilyVersion("1.0", "2.0")
	g.Expect(vault.statsVersion()).To(Equal("2.0"))

	// When the user runs engram vocab propose
	proposed := vault.run("vocab", "propose", "--term", "gardening", "--description", "growing plants in soil")
	g.Expect(proposed.stderr).To(BeEmpty())

	// Then engram vocab stats reports version 2.1
	g.Expect(vault.statsVersion()).To(Equal("2.1"))

	// When an engram vocab refit applies
	refit := vault.refit()
	g.Expect(refit.stderr).To(BeEmpty())

	// Then engram vocab stats reports version 3.0
	g.Expect(refit.stdout).To(ContainSubstring("vocab refit applied: version → 3.0"))
	g.Expect(vault.statsVersion()).To(Equal("3.0"))
}

// Test792a6c1a_LearnFact_ACategoricalTagSurvivesTagging verifies 792a6c1a (controlled-vocabulary.md:108): a fact
// learned with --tag work-kind/review and text close to the term build-tooling carries the tags [work-kind/review,
// vocab/build-tooling], in that order. The catalog row classes the scenario as an interaction by its "in that order"
// cue, but the order is that of the tags in the written note, not of calls at a collaborator, so the test reads the
// written note.
func Test792a6c1a_LearnFact_ACategoricalTagSurvivesTagging(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose vocabulary holds the term build-tooling
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling)

	// When the user runs engram learn fact --tag work-kind/review with text close to build-tooling
	result := vault.learnFact("review-linter", "reviewing the linter setup", "--tag", "work-kind/review")
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note's tags are [work-kind/review, vocab/build-tooling], in that order
	g.Expect(vault.tagsOf("review-linter")).To(Equal([]string{"work-kind/review", "vocab/build-tooling"}))
}

// Test792a6c1b_VocabRefit_ACategoricalTagSurvivesARefit verifies 792a6c1b (controlled-vocabulary.md:113): a note
// tagged [work-kind/review, vocab/old-term] still carries work-kind/review after a refit that retires old-term.
func Test792a6c1b_VocabRefit_ACategoricalTagSurvivesARefit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note tagged [work-kind/review, vocab/old-term]
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking, cv1OldTerm)
	vault.learnTopicNotes("build", cv1BuildSituations)
	vault.learnTopicNotes("cooking", cv1CookingSituations)
	vault.learnFact("review-telescope", "reviewing the telescope schedule", "--tag", "work-kind/review")
	g.Expect(vault.tagsOf("review-telescope")).To(Equal([]string{"work-kind/review", "vocab/old-term"}))

	// When the user runs an engram vocab refit that retires old-term
	result := vault.refit()

	// Then the note still carries work-kind/review
	g.Expect(result.stdout).To(ContainSubstring("retired=1"))
	g.Expect(vault.namesMatching(`\.vocab-old-term-definition\.md$`)).To(BeEmpty())
	g.Expect(vault.tagsOf("review-telescope")).To(Equal([]string{"work-kind/review"}))
}

// Test792a6d1a_LearnFact_ALearnedNoteIsTaggedWithItsClosestTerms verifies 792a6d1a (controlled-vocabulary.md:135): over
// a vocabulary of build-tooling and cooking, a fact about running a linter before committing Go code carries
// vocab/build-tooling and not vocab/cooking.
func Test792a6d1a_LearnFact_ALearnedNoteIsTaggedWithItsClosestTerms(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose vocabulary holds the terms build-tooling and cooking
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking)

	// When the user runs engram learn fact about running a linter before committing Go code
	result := vault.learnFact("lint-before-commit", "running a linter before committing Go code")
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note carries vocab/build-tooling and not vocab/cooking
	tags := vault.tagsOf("lint-before-commit")
	g.Expect(tags).To(ContainElement("vocab/build-tooling"))
	g.Expect(tags).NotTo(ContainElement("vocab/cooking"))
}

// Test792a6d1b_LearnFact_ANoteCloseToNoTermGetsNoTag verifies 792a6d1b (controlled-vocabulary.md:140): over a
// vocabulary holding only cooking, a fact about configuring a TLS certificate has no vocab/ tag and no tags: field.
func Test792a6d1b_LearnFact_ANoteCloseToNoTermGetsNoTag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose vocabulary holds only the term cooking
	vault := newCV1Vault()
	vault.bootstrap(cv1Cooking)

	// When the user runs engram learn fact about configuring a TLS certificate
	result := vault.learnFact("tls-certificate", "configuring a TLS certificate")
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note has no vocab/ tag and no tags: field
	content := vault.read(vault.requireNote(g, `\.tls-certificate\.md$`))
	g.Expect(content).NotTo(ContainSubstring("vocab/"))
	g.Expect(content).NotTo(MatchRegexp(`(?m)^tags:`))
}

// Test792a6d1c_LearnFact_AtMostThreeTermsAreKept verifies 792a6d1c (controlled-vocabulary.md:145): a note close to
// four terms carries the three closest, closest first. The closeness order (cooking, gardening, astronomy) is not the
// term-name order (astronomy, cooking, gardening), so only the closeness rule produces the expected tags.
func Test792a6d1c_LearnFact_AtMostThreeTermsAreKept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with four terms, each close to one note's text (mentioned 8, 7, 6 and 5 times)
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking, cv1Astronomy, cv1Gardening)

	situation := "a mixed day of " + strings.Repeat("cook ", 8) + strings.Repeat("garden ", 7) +
		strings.Repeat("telescope ", 6) + strings.Repeat("linter ", 5)

	// When the user writes that note with engram learn
	result := vault.learnFact("mixed-day", situation)
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note carries the three closest terms, closest first
	g.Expect(vault.tagsOf("mixed-day")).To(Equal([]string{"vocab/cooking", "vocab/gardening", "vocab/astronomy"}))
}

// Test792a6d1d_Amend_AnAmendReTagsTheNote verifies 792a6d1d (controlled-vocabulary.md:150): a note tagged vocab/cooking
// whose object is rewritten with engram amend --object to describe a linter carries vocab/build-tooling and no longer
// vocab/cooking.
func Test792a6d1d_Amend_AnAmendReTagsTheNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note tagged vocab/cooking
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking)
	vault.run("learn", "fact", "--slug", "weekly-routine", "--position", "top", "--source", cv1Source,
		"--situation", "the weekly routine", "--subject", "the team", "--predicate", "follows",
		"--object", "a recipe from the kitchen")
	g.Expect(vault.tagsOf("weekly-routine")).To(Equal([]string{"vocab/cooking"}))

	// When the user runs engram amend --object with text describing a linter
	result := vault.run("amend", "--target", vault.basenameOf("weekly-routine"), "--object", "a linter run before commits")
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note carries vocab/build-tooling and no longer vocab/cooking
	g.Expect(vault.tagsOf("weekly-routine")).To(Equal([]string{"vocab/build-tooling"}))
}

// Test792a6d1e_Resituate_AResituatedNoteIsReTagged verifies 792a6d1e (controlled-vocabulary.md:155): after engram
// resituate rewrites a fact note's situation, its vocab/ tags are set from the rewritten text.
func Test792a6d1e_Resituate_AResituatedNoteIsReTagged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a fact note about cooking and a vault vocabulary
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking)
	vault.learnFact("evening-task", "preparing a recipe in the kitchen")
	g.Expect(vault.tagsOf("evening-task")).To(Equal([]string{"vocab/cooking"}))

	// When the user runs engram resituate on it
	result := vault.run("resituate", "--note", vault.basenameOf("evening-task"),
		"--situation", "running a linter before a commit")
	g.Expect(result.stderr).To(BeEmpty())

	// Then its vocab/ tags are set from its rewritten text
	g.Expect(vault.tagsOf("evening-task")).To(Equal([]string{"vocab/build-tooling"}))
}

// Test792a6e1a_LearnFact_AVaultWithoutAVocabularyWritesPlainNotes verifies 792a6e1a (controlled-vocabulary.md:169): in
// a vault with no definition notes engram learn fact writes the note with no vocab/ tag, and does not rewrite it.
func Test792a6e1a_LearnFact_AVaultWithoutAVocabularyWritesPlainNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with no definition notes
	vault := newCV1Vault()

	// When the user runs engram learn fact
	result := vault.learnFact("lint-before-commit", "running a linter before committing Go code")
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note is written with no vocab/ tag, and written once
	name := vault.requireNote(g, `\.lint-before-commit\.md$`)
	g.Expect(vault.read(name)).NotTo(ContainSubstring("vocab/"))
	g.Expect(vault.tagsOf("lint-before-commit")).To(BeEmpty())
	g.Expect(vault.fsys.writesTo(filepath.Join(cv1VaultPath, name))).To(Equal(1))
}

// Test792a6e1b_LearnFact_ANoteWhoseEmbeddingFailedIsNotTagged verifies 792a6e1b (controlled-vocabulary.md:174): when
// embedding the new note fails, engram learn fact still writes the note, stderr carries a warning naming it, and the
// note has no vocab/ tag.
func Test792a6e1b_LearnFact_ANoteWhoseEmbeddingFailedIsNotTagged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with a vocabulary, where embedding the new note fails
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling)
	vault.embedder.failOn = "unembeddable"

	// When the user runs engram learn fact
	result := vault.learnFact("lint-unembeddable", "running a linter on an unembeddable note")

	// Then the note is written, stderr carries a warning naming it, and the note has no vocab/ tag
	name := vault.requireNote(g, `\.lint-unembeddable\.md$`)
	g.Expect(result.stderr).To(ContainSubstring("learn: embed failed for " + filepath.Join(cv1VaultPath, name)))
	g.Expect(vault.read(name)).NotTo(ContainSubstring("vocab/"))
	g.Expect(vault.exists(embed.SidecarPath(name))).To(BeFalse())
}

// Test792a6f1a_VocabBootstrap_TaggingReadsACRLFNote verifies 792a6f1a (controlled-vocabulary.md:192): vocabulary
// tagging over notes whose lines end in \r\n writes the tag and the note as LF where a term is added, and does not
// write a CRLF note whose tags do not change.
func Test792a6f1a_VocabBootstrap_TaggingReadsACRLFNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given notes whose lines end in \r\n: one close to the term build-tooling, one close to no term
	vault := newCV1Vault()
	vault.learnFact("lint-before-commit", "running a linter before committing Go code")
	vault.learnFact("tls-certificate", "configuring a TLS certificate")
	closeName := vault.requireNote(g, `\.lint-before-commit\.md$`)
	farName := vault.requireNote(g, `\.tls-certificate\.md$`)
	vault.toCRLF(closeName)
	vault.toCRLF(farName)

	farBefore := vault.read(farName)

	vault.fsys.resetLog()

	// When vocabulary tagging adds a new term to the first
	result := vault.bootstrap(cv1BuildTooling)
	g.Expect(result.stderr).To(BeEmpty())

	// Then the tag is written and the note is written as LF
	closeAfter := vault.read(closeName)
	g.Expect(parseCV1Front(closeAfter).Tags).To(Equal([]string{"vocab/build-tooling"}))
	g.Expect(closeAfter).NotTo(ContainSubstring("\r"))

	// And the CRLF note whose tags do not change is not written
	g.Expect(vault.fsys.writesTo(filepath.Join(cv1VaultPath, farName))).To(BeZero())
	g.Expect(vault.read(farName)).To(Equal(farBefore))
	g.Expect(farBefore).To(ContainSubstring("\r\n"))
}

// Test792a6g1a_VocabBootstrap_SeedsTermsAndTagsExistingNotes verifies 792a6g1a (controlled-vocabulary.md:218): over
// notes about Go builds and about cooking and a seed file naming build-tooling, engram vocab bootstrap prints "vocab
// bootstrap: 1 terms, <M> notes assigned", the build notes carry vocab/build-tooling, and engram vocab stats reports
// verdict: OK.
func Test792a6g1a_VocabBootstrap_SeedsTermsAndTagsExistingNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault holding notes about Go builds and about cooking, and a seed file naming build-tooling
	vault := newCV1TwoTopicVault()

	// When the user runs engram vocab bootstrap --seed with that file
	result := vault.bootstrap(cv1BuildTooling)

	// Then it prints the count line, the build notes carry vocab/build-tooling and the cooking notes carry no tag
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(Equal(
		fmt.Sprintf("vocab bootstrap: 1 terms, %d notes assigned\n", len(cv1BuildSituations))))

	for index := range cv1BuildSituations {
		g.Expect(vault.tagsOf(cv1NoteSlug("build", index))).To(Equal([]string{"vocab/build-tooling"}))
		g.Expect(vault.tagsOf(cv1NoteSlug("cooking", index))).To(BeEmpty())
	}

	// And engram vocab stats reports verdict: OK
	g.Expect(vault.run("vocab", "stats").stdout).To(ContainSubstring("verdict: OK"))
}

// Test792a6g1b_VocabBootstrap_ASecondBootstrapCreatesNothingNew verifies 792a6g1b (controlled-vocabulary.md:223):
// running engram vocab bootstrap with the same seed file again creates no new definition note or family note, and
// leaves the definition notes as they were.
func Test792a6g1b_VocabBootstrap_ASecondBootstrapCreatesNothingNew(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault already bootstrapped with a seed file
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking)

	before := vault.fsys.snapshot()
	g.Expect(before.definitionNames()).To(HaveLen(3), "two terms and the family note")

	// When the user runs engram vocab bootstrap with the same seed file again
	result := vault.bootstrap(cv1BuildTooling, cv1Cooking)
	g.Expect(result.stderr).To(BeEmpty())

	// Then no new definition note or family note is created
	after := vault.fsys.snapshot()
	g.Expect(after.definitionNames()).To(Equal(before.definitionNames()))

	for _, name := range before.definitionNames() {
		g.Expect(after[filepath.Join(cv1VaultPath, name)]).To(Equal(before[filepath.Join(cv1VaultPath, name)]))
	}
}

// Test792a6i1a_VocabPropose_AProposedTermIsCreated verifies 792a6i1a (controlled-vocabulary.md:266): over a
// bootstrapped vault at vocab_version "1.0", engram vocab propose --term gardening --description "growing plants"
// prints "vocab propose: created gardening (version → 1.1)" and the vault holds a vocab-gardening-definition note.
func Test792a6i1a_VocabPropose_AProposedTermIsCreated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a bootstrapped vault at vocab_version "1.0"
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling)
	g.Expect(vault.statsVersion()).To(Equal("1.0"))

	// When the user runs engram vocab propose --term gardening --description "growing plants"
	result := vault.run("vocab", "propose", "--term", "gardening", "--description", "growing plants")

	// Then it prints the created line and the vault holds a vocab-gardening-definition note
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(Equal("vocab propose: created gardening (version → 1.1)\n"))
	g.Expect(vault.namesMatching(`^[0-9a-z]+\.` + cv1Today + `\.vocab-gardening-definition\.md$`)).To(HaveLen(1))
}

// Test792a6i1c_VocabPropose_BothFlagsAreRequired verifies 792a6i1c (controlled-vocabulary.md:276): engram vocab
// propose --term x without --description fails saying --description is a missing required flag, and writes nothing.
func Test792a6i1c_VocabPropose_BothFlagsAreRequired(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault
	vault := newCV1Vault()
	before := vault.fsys.snapshot()

	// When the user runs engram vocab propose --term x without --description
	result := vault.run("vocab", "propose", "--term", "x")

	// Then it fails saying --description is a missing required flag, and nothing is written
	g.Expect(result.stderr).To(MatchRegexp(`(?i)missing required flag.*--description|--description.*(missing|required)`))
	g.Expect(result.stdout).To(BeEmpty())
	g.Expect(vault.fsys.writeCount()).To(BeZero())
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// Test792a6j1a_VocabRefitDryRun_TheTermCountFollowsTheVaultsClusters verifies 792a6j1a (controlled-vocabulary.md:298):
// over a vault of notes in two clear topics, engram vocab refit --dry-run prints "vocab refit (dry-run): K=2
// silhouette=<s>".
func Test792a6j1a_VocabRefitDryRun_TheTermCountFollowsTheVaultsClusters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault of notes in two clear topics
	vault := newCV1TwoTopicVault()

	// When the user runs engram vocab refit --dry-run
	result := vault.run("vocab", "refit", "--dry-run")

	// Then it prints the K=2 line with a silhouette
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(MatchRegexp(`(?m)^vocab refit \(dry-run\): K=2 silhouette=-?[0-9]+\.[0-9]{3}$`))
}

// Test792a6k1a_VocabRefitDryRun_AClusterKeepsItsTermsName verifies 792a6k1a (controlled-vocabulary.md:322): over a
// vault whose term build-tooling sits at the centre of one of its clusters, engram vocab refit --dry-run prints
// "matched: build-tooling (cluster <n>, similarity <s>)" with s at least 0.800.
func Test792a6k1a_VocabRefitDryRun_AClusterKeepsItsTermsName(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose term build-tooling sits at the centre of one of its clusters
	vault := newCV1TwoTopicVault()
	vault.bootstrap(cv1BuildTooling)

	// When the user runs engram vocab refit --dry-run
	result := vault.run("vocab", "refit", "--dry-run")

	// Then it prints the matched line for build-tooling with a similarity of at least 0.800
	g.Expect(result.stderr).To(BeEmpty())

	matched := regexp.MustCompile(`(?m)^matched: build-tooling \(cluster [0-9]+, similarity ([0-9.]+)\)$`).
		FindStringSubmatch(result.stdout)
	g.Expect(matched).To(HaveLen(2), "the dry run must print the matched line for build-tooling:\n"+result.stdout)

	if len(matched) != 2 {
		return
	}

	similarity, parseErr := strconv.ParseFloat(matched[1], 64)
	g.Expect(parseErr).NotTo(HaveOccurred())
	g.Expect(similarity).To(BeNumerically(">=", cv1NameMatchFloor))
}

// Test792a6k1b_VocabRefitDryRun_UnmatchedClustersAndTermsAreReported verifies 792a6k1b (controlled-vocabulary.md:327):
// over a vault with an unmatched cluster and an unmatched derived term astronomy, engram vocab refit --dry-run prints
// "new: cluster <n> (<m> members, needs naming)" and "retired: astronomy".
func Test792a6k1b_VocabRefitDryRun_UnmatchedClustersAndTermsAreReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with an unmatched cluster (cooking) and an unmatched derived term astronomy
	vault := newCV1MatchedNewRetiredVault()

	// When the user runs engram vocab refit --dry-run
	result := vault.run("vocab", "refit", "--dry-run")

	// Then it prints the new line for the cooking cluster and the retired line for astronomy
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(MatchRegexp(
		fmt.Sprintf(`(?m)^new: cluster [0-9]+ \(%d members, needs naming\)$`, len(cv1CookingSituations))))
	g.Expect(result.stdout).To(MatchRegexp(`(?m)^retired: astronomy$`))
}

// Test792a6l1a_VocabRefitDryRun_ReportsTheDerivationDiffWithoutWriting verifies 792a6l1a
// (controlled-vocabulary.md:345): over a vault whose derivation matches one term, finds one new cluster and retires
// one term, engram vocab refit --dry-run prints the K and silhouette line and the matched, new and retired lines, and
// modifies no vault file or centroids file.
func Test792a6l1a_VocabRefitDryRun_ReportsTheDerivationDiffWithoutWriting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose derivation matches one term, finds one new cluster and retires one term
	vault := newCV1MatchedNewRetiredVault()
	before := vault.fsys.snapshot()
	g.Expect(before).To(HaveKey(filepath.Join(cv1VaultPath, "vocab.centroids.json")))

	vault.fsys.resetLog()

	// When the user runs engram vocab refit --dry-run
	result := vault.run("vocab", "refit", "--dry-run")

	// Then the K and silhouette line, the matched, new and retired lines are printed
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(MatchRegexp(`(?m)^vocab refit \(dry-run\): K=2 silhouette=-?[0-9]+\.[0-9]{3}$`))
	g.Expect(result.stdout).To(MatchRegexp(`(?m)^matched: build-tooling \(cluster [0-9]+, similarity [0-9.]+\)$`))
	g.Expect(result.stdout).To(MatchRegexp(`(?m)^new: cluster [0-9]+ \([0-9]+ members, needs naming\)$`))
	g.Expect(result.stdout).To(MatchRegexp(`(?m)^retired: astronomy$`))

	// And no vault file or centroids file is modified
	g.Expect(vault.fsys.writeCount()).To(BeZero())
	g.Expect(vault.fsys.removeCount()).To(BeZero())
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// Test792a6m1a_VocabRefit_NewClustersAreEmittedAsNamingRequests verifies 792a6m1a (controlled-vocabulary.md:365):
// over a vault whose derivation finds one cluster that matches no term, engram vocab refit prints a naming_requests
// document for that cluster with its member count, three exemplars (the members nearest its centroid) and a
// fingerprint, and changes no vault file.
func Test792a6m1a_VocabRefit_NewClustersAreEmittedAsNamingRequests(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose derivation finds one cluster (cooking) that matches no term; the cooking notes mention an
	// unrelated topic 0, 1, 2, 4 and 8 times, so their distances to the cluster centroid are all different
	vault := newCV1Vault()
	vault.learnTopicNotes("build", cv1BuildSituations)

	noise := []int{0, 1, 2, 4, 8}
	for index, count := range noise {
		vault.learnFact(cv1NoteSlug("cooking", index),
			"tasting "+strings.Repeat("recipe ", cv1RecipeMentions)+strings.Repeat("telescope ", count))
	}

	vault.bootstrap(cv1BuildTooling)

	before := vault.fsys.snapshot()
	vault.fsys.resetLog()

	// When the user runs engram vocab refit
	result := vault.refit()

	// Then it prints a naming_requests document for that cluster ...
	g.Expect(result.stderr).To(BeEmpty())

	// the snake_case contract keys are read by key, so a renamed key fails here
	var document map[string]json.RawMessage

	g.Expect(json.Unmarshal([]byte(result.stdout), &document)).To(Succeed())

	requests := cv1JSONField[[]map[string]json.RawMessage](g, document, "naming_requests")
	g.Expect(requests).To(HaveLen(1))
	g.Expect(cv1JSONField[string](g, document, "instruction")).NotTo(BeEmpty())
	g.Expect(cv1JSONField[string](g, document, "fingerprint")).NotTo(BeEmpty())

	if len(requests) != 1 {
		return
	}

	request := requests[0]
	g.Expect(cv1JSONField[int](g, request, "member_count")).To(Equal(len(noise)))

	// ... with three exemplars, the members nearest its centroid, nearest first (mentions 4, 2 and 8)
	exemplars := cv1JSONField[[]cv1NamingExemplar](g, request, "exemplars")
	g.Expect(exemplars).To(HaveLen(cv1ExemplarLimit))

	if len(exemplars) != cv1ExemplarLimit {
		return
	}

	for position, noiseIndex := range []int{3, 2, 4} {
		exemplar := exemplars[position]
		slug := cv1NoteSlug("cooking", noiseIndex)
		name := vault.requireNote(g, `\.`+slug+`\.md$`)

		g.Expect(exemplar.Note).To(Equal(name))
		g.Expect(exemplar.Title).To(Equal(slug))
		g.Expect(exemplar.Snippet).NotTo(BeEmpty())
		g.Expect(len(exemplar.Snippet)).To(BeNumerically("<=", cv1SnippetLimit))
		g.Expect(vault.read(name)).To(ContainSubstring(exemplar.Snippet))
	}

	// ... and no vault file changes
	g.Expect(vault.fsys.writeCount()).To(BeZero())
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// unexported constants.
const (
	cv1DirPerm       = fs.FileMode(0o750)
	cv1Exemplar      = "Roast a chicken with fresh herbs"
	cv1ExemplarLimit = 3
	cv1FilePerm      = fs.FileMode(0o600)
	// cv1LearnFactArgCount is the number of arguments of engram learn fact before the extra ones.
	cv1LearnFactArgCount = 16
	cv1ModelID           = "cv1-topics@4"
	// cv1NameMatchFloor is the 0.800 similarity at which a cluster keeps its term's name (792a6k).
	cv1NameMatchFloor = 0.8
	cv1RecipeMentions = 3
	cv1SeedPath       = "/seed.yaml"
	cv1SnippetLimit   = 200
	cv1Source         = "cv1"
	// cv1Today is the date of the vault clock; note filenames embed it.
	cv1Today = "2026-10-22"
	// cv1TopicCount is the number of topic axes of the cv1TopicEmbedder: build, cooking, astronomy, gardening.
	cv1TopicCount = 4
	cv1VaultKey   = "vault"
	cv1VaultPath  = "/vault"
)

// unexported variables.
var (
	cv1Astronomy       = cli.SeedTerm{Term: "astronomy", Description: "telescopes and galaxies"}
	cv1BuildSituations = []string{
		"running the linter before a commit",
		"waiting for the compiler on a clean build",
		"fixing a linter warning in CI",
		"upgrading the compiler toolchain",
		"caching compiler output",
	}
	cv1BuildTooling = cli.SeedTerm{Term: "build-tooling", Description: "linters, compilers and build scripts"}
	cv1Clock        = time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC)
	cv1Cooking      = cli.SeedTerm{
		Term: "cooking", Description: "recipes and kitchens", Exemplars: []string{cv1Exemplar},
	}
	cv1CookingSituations = []string{
		"tasting a recipe",
		"preheating the kitchen oven",
		"scaling a recipe for guests",
		"cleaning the kitchen after a meal",
		"adjusting the salt in a recipe",
	}
	cv1Gardening = cli.SeedTerm{Term: "gardening", Description: "growing plants in soil"}
	cv1OldTerm   = cli.SeedTerm{Term: "old-term", Description: "telescopes and galaxies that nobody studies any more"}
	// cv1TopicKeywords are the keywords of each topic axis, lower case; a text counts a keyword once per occurrence.
	cv1TopicKeywords = [cv1TopicCount][]string{
		{"linter", "compiler", "build-tooling"},
		{"cook", "recipe", "kitchen"},
		{"astronom", "telescope", "galax"},
		{"garden", "soil", "tomato"},
	}
	errCV1Embed       = errors.New("cv1TopicEmbedder: embedding refused")
	errCV1Unsupported = errors.New("cv1MemFS: operation not supported")
)

// cv1Front is the part of a note's frontmatter the scenarios read.
type cv1Front struct {
	Tags         []string `yaml:"tags"`
	Object       string   `yaml:"object"`
	VocabVersion string   `yaml:"vocab_version"`
}

// cv1MemFS is an in-memory cli.EdgeFS over fstest.MapFS that records the paths written and removed. Paths are
// absolute; the map keys are the same paths without the leading slash.
type cv1MemFS struct {
	mu      sync.Mutex
	files   fstest.MapFS
	writes  []string
	removes []string
}

func (m *cv1MemFS) Lstat(path string) (fs.FileInfo, error) { return m.Stat(path) }

func (m *cv1MemFS) MkdirAll(path string, _ fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for dir := cv1Key(path); dir != "." && dir != ""; dir = cv1Key(filepath.Dir(dir)) {
		if _, found := m.files[dir]; !found {
			m.files[dir] = &fstest.MapFile{Mode: fs.ModeDir | cv1DirPerm}
		}
	}

	return nil
}

func (m *cv1MemFS) MkdirTemp(string, string) (string, error) { return "", errCV1Unsupported }

func (m *cv1MemFS) ReadDir(path string) ([]fs.DirEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := fs.ReadDir(m.files, cv1Key(path))
	if err != nil {
		return nil, fmt.Errorf("cv1MemFS readdir %s: %w", path, err)
	}

	return entries, nil
}

func (m *cv1MemFS) ReadFile(path string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := fs.ReadFile(m.files, cv1Key(path))
	if err != nil {
		return nil, fmt.Errorf("cv1MemFS read %s: %w", path, err)
	}

	return data, nil
}

func (m *cv1MemFS) Readlink(string) (string, error) { return "", errCV1Unsupported }

func (m *cv1MemFS) Remove(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := cv1Key(path)
	if _, found := m.files[key]; !found {
		return &fs.PathError{Op: "remove", Path: path, Err: fs.ErrNotExist}
	}

	delete(m.files, key)

	m.removes = append(m.removes, path)

	return nil
}

func (m *cv1MemFS) RemoveAll(string) error { return errCV1Unsupported }

func (m *cv1MemFS) Rename(string, string) error { return errCV1Unsupported }

func (m *cv1MemFS) Stat(path string) (fs.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, err := fs.Stat(m.files, cv1Key(path))
	if err != nil {
		return nil, fmt.Errorf("cv1MemFS stat %s: %w", path, err)
	}

	return info, nil
}

func (m *cv1MemFS) Symlink(string, string) error { return errCV1Unsupported }

func (m *cv1MemFS) WalkDir(root string, walk fs.WalkDirFunc) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return fs.WalkDir(m.files, cv1Key(root), walk)
}

func (m *cv1MemFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.files[cv1Key(path)] = &fstest.MapFile{Data: slices.Clone(data), Mode: perm}
	m.writes = append(m.writes, path)

	return nil
}

func (m *cv1MemFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return m.WriteFile(path, data, perm)
}

func (m *cv1MemFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	m.mu.Lock()
	_, found := m.files[cv1Key(path)]
	m.mu.Unlock()

	if found {
		return &fs.PathError{Op: "create", Path: path, Err: fs.ErrExist}
	}

	return m.WriteFile(path, data, perm)
}

// put stores a file without recording a write: the fixture of a scenario, not a write of the command under test.
func (m *cv1MemFS) put(path string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.files[cv1Key(path)] = &fstest.MapFile{Data: slices.Clone(data), Mode: cv1FilePerm}
}

func (m *cv1MemFS) removeCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.removes)
}

// resetLog forgets the writes and removals so far.
func (m *cv1MemFS) resetLog() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.writes = nil
	m.removes = nil
}

// snapshot is every regular file under the vault, by absolute path.
func (m *cv1MemFS) snapshot() cv1Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	files := make(cv1Snapshot, len(m.files))

	for key, file := range m.files {
		if !file.Mode.IsDir() {
			files["/"+key] = string(file.Data)
		}
	}

	return files
}

func (m *cv1MemFS) writeCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.writes)
}

// writesTo counts the writes of path since the last resetLog.
func (m *cv1MemFS) writesTo(path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(slices.DeleteFunc(slices.Clone(m.writes), func(written string) bool { return written != path }))
}

// cv1NamingExemplar is one exemplar of a naming request.
type cv1NamingExemplar struct {
	Note    string `json:"note"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// cv1Result is what one command printed.
type cv1Result struct {
	stdout string
	stderr string
}

// cv1Snapshot maps the absolute path of every regular file to its content.
type cv1Snapshot map[string]string

// definitionNames are the basenames of the definition notes (family note included) in the snapshot, sorted.
func (s cv1Snapshot) definitionNames() []string {
	names := make([]string, 0, len(s))

	for path := range s {
		if strings.HasSuffix(path, "-definition.md") {
			names = append(names, filepath.Base(path))
		}
	}

	slices.Sort(names)

	return names
}

// cv1TopicEmbedder embeds a text as its keyword counts per topic axis (cv1TopicKeywords), so that the cosine of two
// texts is decided by the topics they mention. It refuses a text that contains failOn when that is set.
type cv1TopicEmbedder struct {
	failOn string
}

func (e *cv1TopicEmbedder) Dims() int { return cv1TopicCount }

func (e *cv1TopicEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if e.failOn != "" && strings.Contains(text, e.failOn) {
		return nil, errCV1Embed
	}

	lower := strings.ToLower(text)
	vector := make([]float32, cv1TopicCount)

	for axis, keywords := range cv1TopicKeywords {
		for _, keyword := range keywords {
			vector[axis] += float32(strings.Count(lower, keyword))
		}
	}

	return vector, nil
}

func (e *cv1TopicEmbedder) ModelID() string { return cv1ModelID }

// cv1Vault is one in-memory vault and the commands that run against it.
type cv1Vault struct {
	fsys     *cv1MemFS
	embedder *cv1TopicEmbedder
}

// basenameOf is the basename (without .md) of the note with the slug.
func (v *cv1Vault) basenameOf(slug string) string {
	return strings.TrimSuffix(v.onlyNameMatching(`\.`+regexp.QuoteMeta(slug)+`\.md$`), ".md")
}

// bootstrap runs engram vocab bootstrap over a seed file naming the terms.
func (v *cv1Vault) bootstrap(terms ...cli.SeedTerm) cv1Result {
	seed, marshalErr := yaml.Marshal(terms)
	if marshalErr != nil {
		return cv1Result{stderr: marshalErr.Error()}
	}

	v.fsys.put(cv1SeedPath, seed)

	return v.run("vocab", "bootstrap", "--seed", cv1SeedPath)
}

// dispatch runs `engram <args> --vault /vault` through the real targets over the in-memory vault; the error is the
// one targ.Execute returns, the non-zero exit status of the command.
func (v *cv1Vault) dispatch(args ...string) (cv1Result, error) {
	var stdout, stderr bytes.Buffer

	deps := cli.Deps{
		Stdout:       &stdout,
		Stderr:       &stderr,
		Exit:         func(int) {},
		Getenv:       func(string) string { return "" },
		Now:          func() time.Time { return cv1Clock },
		Getwd:        func() (string, error) { return "/", nil },
		UserHomeDir:  func() (string, error) { return "/home/cv1", nil },
		Username:     func() (string, error) { return "cv1-user", nil },
		FS:           v.fsys,
		Lock:         fakeLocker{lock: func(string) (func() error, error) { return func() error { return nil }, nil }},
		Commander:    stubCommander{},
		Embed:        v.embedder,
		RandRead:     seqRand(1),
		EvalSymlinks: identityPath,
	}

	command := append([]string{"engram"}, args...)
	command = append(command, "--vault", cv1VaultPath)

	executed, err := targ.Execute(command, cli.Targets(deps)...)
	if err != nil {
		stderr.WriteString(executed.Output)
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return cv1Result{stdout: stdout.String(), stderr: stderr.String()}, err
}

func (v *cv1Vault) exists(name string) bool {
	_, statErr := v.fsys.Stat(filepath.Join(cv1VaultPath, name))

	return statErr == nil
}

// learnFact runs engram learn fact with the slug and situation and a subject, predicate and object that mention no
// topic.
func (v *cv1Vault) learnFact(slug, situation string, extra ...string) cv1Result {
	args := make([]string, 0, cv1LearnFactArgCount+len(extra))
	args = append(args,
		"learn", "fact", "--slug", slug, "--position", "top", "--source", cv1Source, "--situation", situation,
		"--subject", "the team", "--predicate", "follows", "--object", "the routine")

	return v.run(append(args, extra...)...)
}

// learnTopicNotes learns one note per situation, slugged by the prefix and the position.
func (v *cv1Vault) learnTopicNotes(prefix string, situations []string) {
	for index, situation := range situations {
		v.learnFact(cv1NoteSlug(prefix, index), situation)
	}
}

// namesMatching lists the vault's notes (.md basenames) matching the pattern.
func (v *cv1Vault) namesMatching(pattern string) []string {
	entries, readErr := v.fsys.ReadDir(cv1VaultPath)
	if readErr != nil {
		return nil
	}

	matcher := regexp.MustCompile(pattern)
	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		if matcher.MatchString(entry.Name()) {
			names = append(names, entry.Name())
		}
	}

	return names
}

// onlyNameMatching is the one note matching the pattern, or "" when there are none or several.
func (v *cv1Vault) onlyNameMatching(pattern string) string {
	names := v.namesMatching(pattern)
	if len(names) != 1 {
		return ""
	}

	return names[0]
}

// read is the content of the vault file.
func (v *cv1Vault) read(name string) string {
	data, readErr := v.fsys.ReadFile(filepath.Join(cv1VaultPath, name))
	if readErr != nil {
		return ""
	}

	return string(data)
}

// refit runs engram vocab refit with no flags, and, when that prints naming requests, answers none.
func (v *cv1Vault) refit() cv1Result { return v.run("vocab", "refit") }

// requireNote is the one note matching the pattern; it fails the scenario when there is not exactly one.
func (v *cv1Vault) requireNote(g *WithT, pattern string) string {
	names := v.namesMatching(pattern)
	g.Expect(names).To(HaveLen(1), "notes matching %s", pattern)

	if len(names) != 1 {
		return ""
	}

	return names[0]
}

// run runs `engram <args> --vault /vault` through the real targets over the in-memory vault.
func (v *cv1Vault) run(args ...string) cv1Result {
	result, _ := v.dispatch(args...)

	return result
}

// setFamilyVersion rewrites the vocab_version of the family note from one value to another.
func (v *cv1Vault) setFamilyVersion(from, to string) {
	name := v.onlyNameMatching(`\.vocab-definition\.md$`)
	content := strings.Replace(v.read(name), `vocab_version: "`+from+`"`, `vocab_version: "`+to+`"`, 1)

	v.fsys.put(filepath.Join(cv1VaultPath, name), []byte(content))
}

// statsVersion is the vocab_version engram vocab stats reports.
func (v *cv1Vault) statsVersion() string {
	matched := regexp.MustCompile(`vocab stats \(version: ([0-9.]+)\)`).FindStringSubmatch(v.run("vocab", "stats").stdout)
	if len(matched) != 2 {
		return ""
	}

	return matched[1]
}

// tagsOf is the tags: list of the note with the slug.
func (v *cv1Vault) tagsOf(slug string) []string {
	return parseCV1Front(v.read(v.onlyNameMatching(`\.` + regexp.QuoteMeta(slug) + `\.md$`))).Tags
}

// toCRLF rewrites the vault note with CRLF line endings.
func (v *cv1Vault) toCRLF(name string) {
	v.fsys.put(filepath.Join(cv1VaultPath, name), []byte(strings.ReplaceAll(v.read(name), "\n", "\r\n")))
}

// cv1JSONField decodes the value under the key of a JSON object, failing the scenario when the key is absent.
func cv1JSONField[T any](g *WithT, object map[string]json.RawMessage, key string) T {
	var value T

	raw, found := object[key]
	g.Expect(found).To(BeTrue(), "key %s", key)

	if !found {
		return value
	}

	g.Expect(json.Unmarshal(raw, &value)).To(Succeed())

	return value
}

// cv1Key is the fstest.MapFS key of an absolute path.
func cv1Key(path string) string {
	return strings.TrimPrefix(filepath.Clean(path), "/")
}

// cv1NoteSlug is the slug of the position-th note learned under the prefix.
func cv1NoteSlug(prefix string, index int) string {
	return fmt.Sprintf("%s-note-%d", prefix, index)
}

// newCV1MatchedNewRetiredVault is a bootstrapped vault whose derivation matches build-tooling, finds the cooking
// notes as a new cluster and retires the derived term astronomy.
func newCV1MatchedNewRetiredVault() *cv1Vault {
	vault := newCV1TwoTopicVault()
	vault.bootstrap(cv1BuildTooling, cv1Astronomy)

	return vault
}

// newCV1TwoTopicVault is a vault of five notes about builds and five about cooking, with no vocabulary.
func newCV1TwoTopicVault() *cv1Vault {
	vault := newCV1Vault()
	vault.learnTopicNotes("build", cv1BuildSituations)
	vault.learnTopicNotes("cooking", cv1CookingSituations)

	return vault
}

// newCV1Vault is an empty vault.
func newCV1Vault() *cv1Vault {
	return &cv1Vault{
		fsys:     &cv1MemFS{files: fstest.MapFS{cv1VaultKey: {Mode: fs.ModeDir | cv1DirPerm}}},
		embedder: &cv1TopicEmbedder{},
	}
}

// parseCV1Front parses the frontmatter of a note, or returns the zero value when it has none.
func parseCV1Front(content string) cv1Front {
	var front cv1Front

	block, _, found := embed.SplitFrontmatter([]byte(content))
	if !found {
		return front
	}

	if yaml.Unmarshal(block, &front) != nil {
		return cv1Front{}
	}

	return front
}
