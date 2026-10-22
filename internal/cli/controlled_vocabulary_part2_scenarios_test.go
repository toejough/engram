package cli_test

// Scenario-ID tests of the Controlled Vocabulary product capability, part 2
// (docs/specs/product/controlled-vocabulary.md): 792a6n Naming answers, 792a6o Applying a refit, 792a6p Retiring a
// term, 792a6q Proposed terms survive derivation, 792a6r Refit takes no term plan, 792a6s Refit trigger, 792a6t
// Vocabulary statistics and 792a6u Self-tag backfill. The update notices and the migration (792a6v, 792a6w) are in
// controlled_vocabulary_part2_update_scenarios_test.go.
//
// Every scenario drives the real command path (cli.Targets over the production compositions) over the in-memory vault
// and the deterministic topic embedder of the part-1 tests (newCV1Vault, cv1TopicEmbedder): no real filesystem, home,
// git or embedding model. The notes of a scenario are written by `engram learn fact` itself, the vocabulary by
// `engram vocab bootstrap` and `engram vocab propose`, so every note, sidecar and definition note is in the shape the
// system produces. The one hand-made shape is the earlier layout a scenario starts from (a definition note without
// its self-tag, a note tagged by an earlier write, a refit baseline of an earlier date).

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a6n1a_VocabRefitNames_ABadNamingAnswerIsRefused verifies 792a6n1a (controlled-vocabulary.md:388): over a
// vault whose refit emitted naming requests for cluster 1, engram vocab refit --names fails with the error that names
// the problem of each bad answer, and writes nothing.
func Test792a6n1a_VocabRefitNames_ABadNamingAnswerIsRefused(t *testing.T) {
	t.Parallel()

	cooking := cv2NameEntry{Cluster: cv2NewCluster, Term: "cooking", Description: cv1Cooking.Description}
	cases := []struct {
		answer   string
		problems func(fingerprint string) []string
		content  func(g *WithT, fingerprint string) string
	}{
		{
			answer: "a fingerprint that does not match",
			content: func(g *WithT, _ string) string {
				return cv2NamesJSON(g, cv2StaleFingerprint, cooking)
			},
			problems: func(fingerprint string) []string {
				return []string{
					"the vault changed since the naming requests were emitted", cv2StaleFingerprint, fingerprint,
				}
			},
		},
		{
			answer: "no entry for cluster 1",
			content: func(g *WithT, fingerprint string) string {
				return cv2NamesJSON(g, fingerprint)
			},
			problems: func(string) []string {
				return []string{"--names must name every new cluster: cluster 1 unnamed"}
			},
		},
		{
			answer: "an entry for cluster 7",
			content: func(g *WithT, fingerprint string) string {
				return cv2NamesJSON(g, fingerprint, cv2NameEntry{Cluster: cv2UnknownCluster, Term: "x", Description: "y"})
			},
			problems: func(string) []string { return []string{"--names references an unknown cluster"} },
		},
		{
			answer: "two entries for cluster 1",
			content: func(g *WithT, fingerprint string) string {
				return cv2NamesJSON(g, fingerprint, cooking, cooking)
			},
			problems: func(string) []string { return []string{"--names names the same cluster twice"} },
		},
		{
			answer: "an entry with an empty description",
			content: func(g *WithT, fingerprint string) string {
				return cv2NamesJSON(g, fingerprint, cv2NameEntry{Cluster: cv2NewCluster, Term: "cooking"})
			},
			problems: func(string) []string { return []string{"--names entry needs non-empty term and description"} },
		},
		{
			answer:   "a file that is not JSON",
			content:  func(*WithT, string) string { return "this is not JSON" },
			problems: func(string) []string { return []string{"cannot parse --names JSON"} },
		},
	}

	for _, tc := range cases {
		t.Run(tc.answer, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a vault whose refit emitted naming requests for cluster 1
			vault := newCV2NamingVault()
			cluster, fingerprint := cv2EmitNaming(g, vault)
			g.Expect(cluster).To(Equal(cv2NewCluster))

			vault.fsys.put(cv2NamesPath, []byte(tc.content(g, fingerprint)))

			before := vault.fsys.snapshot()
			vault.fsys.resetLog()

			// When the user runs engram vocab refit --names with the bad answer
			result := vault.run("vocab", "refit", "--names", cv2NamesPath)

			// Then it fails with an error saying the problem, and nothing is written
			for _, problem := range tc.problems(fingerprint) {
				g.Expect(result.stderr).To(ContainSubstring(problem))
			}

			g.Expect(result.stdout).To(BeEmpty())
			g.Expect(vault.fsys.writeCount()).To(BeZero())
			g.Expect(vault.fsys.removeCount()).To(BeZero())
			g.Expect(vault.fsys.snapshot()).To(Equal(before))
		})
	}
}

// Test792a6o1a_VocabRefitNames_ANamedClusterBecomesATerm verifies 792a6o1a (controlled-vocabulary.md:415): over a
// vault at version 1.0 whose refit emitted a naming request for cluster 1, engram vocab refit --names with cooking for
// cluster 1 and the emitted fingerprint prints "vocab refit applied: version → 2.0 (K=2, matched=1, new=1,
// retired=0)", a vocab-cooking-definition note exists, and the cluster's notes carry vocab/cooking.
func Test792a6o1a_VocabRefitNames_ANamedClusterBecomesATerm(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault at version 1.0 whose refit emitted a naming request for cluster 1
	vault := newCV2NamingVault()
	g.Expect(vault.statsVersion()).To(Equal("1.0"))

	cluster, fingerprint := cv2EmitNaming(g, vault)
	g.Expect(cluster).To(Equal(cv2NewCluster))

	// When the user runs engram vocab refit --names with cooking for cluster 1 and the emitted fingerprint
	answer := cv2NamesJSON(g, fingerprint,
		cv2NameEntry{Cluster: cv2NewCluster, Term: "cooking", Description: cv1Cooking.Description})
	vault.fsys.put(cv2NamesPath, []byte(answer))

	result := vault.run("vocab", "refit", "--names", cv2NamesPath)

	// Then it prints the applied line
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(Equal("vocab refit applied: version → 2.0 (K=2, matched=1, new=1, retired=0)\n"))

	// And a vocab-cooking-definition note exists
	vault.requireNote(g, `\.vocab-cooking-definition\.md$`)

	// And the cluster's notes carry vocab/cooking
	for index := range cv1CookingSituations {
		g.Expect(vault.tagsOf(cv1NoteSlug("cooking", index))).To(ContainElement("vocab/cooking"))
	}
}

// Test792a6o1b_VocabRefit_ARefitWithNothingToNameAppliesAtOnce verifies 792a6o1b (controlled-vocabulary.md:420): over
// a vault whose derivation matches every cluster to a term, engram vocab refit applies without emitting naming
// requests, and engram vocab stats reports the new major version and verdict: OK.
func Test792a6o1b_VocabRefit_ARefitWithNothingToNameAppliesAtOnce(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose derivation matches every cluster (builds, cooking) to a term
	vault := newCV1TwoTopicVault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking)
	g.Expect(vault.statsVersion()).To(Equal("1.0"))

	// When the user runs engram vocab refit
	result := vault.refit()

	// Then it applies without emitting naming requests
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(Equal("vocab refit applied: version → 2.0 (K=2, matched=2, new=0, retired=0)\n"))
	g.Expect(result.stdout).NotTo(ContainSubstring("naming_requests"))

	// And engram vocab stats reports the new major version and verdict: OK
	stats := vault.run("vocab", "stats").stdout
	g.Expect(stats).To(ContainSubstring("vocab stats (version: 2.0)\n"))
	g.Expect(stats).To(ContainSubstring("verdict: OK\n"))
}

// Test792a6p1a_VocabRefit_ARetiredTermIsDeletedWithItsReferences verifies 792a6p1a (controlled-vocabulary.md:442):
// over a vault whose derived term astronomy matches no cluster, whose definition another note's body links to and
// names in a supersedes: entry, engram vocab refit deletes the astronomy definition note and its sidecar, leaves no
// note with vocab/astronomy, drops the supersedes: entry and its Supersedes: line, and turns the body link into the
// plain basename.
func Test792a6p1a_VocabRefit_ARetiredTermIsDeletedWithItsReferences(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose derived term astronomy matches no cluster
	vault := newCV1TwoTopicVault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking, cv1Astronomy)

	definition := vault.requireNote(g, `\.vocab-astronomy-definition\.md$`)
	basename := strings.TrimSuffix(definition, ".md")

	// And a note tagged vocab/astronomy whose body links to the definition and which names it in a supersedes: entry
	pointer := cv2LearnPointerNote(g, vault, "astronomy-pointer", basename, definition)
	g.Expect(pointer).To(ContainSubstring("supersedes:\n"))
	g.Expect(pointer).To(ContainSubstring("Supersedes: [[" + basename + "]]"))
	g.Expect(pointer).To(ContainSubstring("the routine in [[" + basename + "]]"))
	g.Expect(pointer).To(ContainSubstring("vocab/astronomy"))

	// When the user runs engram vocab refit
	result := vault.refit()
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(Equal("vocab refit applied: version → 2.0 (K=2, matched=2, new=0, retired=1)\n"))

	// Then the astronomy definition note and its sidecar are gone
	g.Expect(vault.exists(definition)).To(BeFalse())
	g.Expect(vault.exists(embed.SidecarPath(definition))).To(BeFalse())

	// And no note carries vocab/astronomy
	for _, name := range vault.namesMatching(`\.md$`) {
		g.Expect(parseCV1Front(vault.read(name)).Tags).NotTo(ContainElement("vocab/astronomy"), name)
	}

	// And the supersedes: entry and its Supersedes: line are gone, and the body link reads as the plain basename
	scrubbed := vault.read(vault.requireNote(g, `\.astronomy-pointer\.md$`))
	g.Expect(scrubbed).NotTo(ContainSubstring("supersedes:"))
	g.Expect(scrubbed).NotTo(ContainSubstring("Supersedes:"))
	g.Expect(scrubbed).NotTo(ContainSubstring("[[" + basename))
	g.Expect(scrubbed).To(ContainSubstring("the routine in " + basename))
}

// Test792a6p1b_VocabRefit_RetirementLeavesNoDanglingBodyLink verifies 792a6p1b (controlled-vocabulary.md:447): over a
// vault whose family note links to a term's definition that refit retires, engram vocab refit and then engram check
// report no dangling link to the deleted basename from a note body.
func Test792a6p1b_VocabRefit_RetirementLeavesNoDanglingBodyLink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose family note links to the definition of the term astronomy, which no cluster matches
	vault := newCV1TwoTopicVault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking, cv1Astronomy)

	basename := strings.TrimSuffix(vault.requireNote(g, `\.vocab-astronomy-definition\.md$`), ".md")
	family := strings.TrimSuffix(vault.requireNote(g, `\.vocab-definition\.md$`), ".md")

	amended := vault.run("amend", "--target", family,
		"--object", "the vocab tag family, with astronomy defined in [["+basename+"]]")
	g.Expect(amended.stderr).To(BeEmpty())
	g.Expect(vault.read(family + ".md")).To(ContainSubstring("[[" + basename + "]]"))
	g.Expect(vault.statsVersion()).To(Equal("1.0"))

	// And engram check finds every link resolving before the refit
	before := vault.run("check").stdout
	g.Expect(before).To(ContainSubstring("PASS  G0 graph-resolution"))
	g.Expect(before).NotTo(ContainSubstring("G3 dangling"))

	// When the user runs engram vocab refit and then engram check
	refit := vault.refit()
	g.Expect(refit.stderr).To(BeEmpty())
	g.Expect(refit.stdout).To(ContainSubstring("retired=1"))

	checked := vault.run("check").stdout

	// Then engram check reports no dangling link to the deleted basename from a note body
	g.Expect(checked).NotTo(ContainSubstring(basename))
	g.Expect(checked).NotTo(ContainSubstring("G3 dangling"))
	g.Expect(vault.read(family + ".md")).NotTo(ContainSubstring("[[" + basename))
}

// Test792a6q1a_VocabRefit_AnUnmatchedProposedTermIsKept verifies 792a6q1a (controlled-vocabulary.md:465): over a vault
// where the term gardening was proposed and no derived cluster matches it, engram vocab refit leaves the gardening
// definition note, and engram vocab stats still lists gardening.
func Test792a6q1a_VocabRefit_AnUnmatchedProposedTermIsKept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault where the term gardening was proposed and no derived cluster matches it
	vault := newCV1TwoTopicVault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking)

	proposed := vault.run("vocab", "propose", "--term", cv1Gardening.Term, "--description", cv1Gardening.Description)
	g.Expect(proposed.stderr).To(BeEmpty())

	dryRun := vault.run("vocab", "refit", "--dry-run").stdout
	g.Expect(dryRun).NotTo(ContainSubstring("gardening"))

	definition := vault.requireNote(g, `\.vocab-gardening-definition\.md$`)

	// When the user runs engram vocab refit
	result := vault.refit()

	// Then the gardening definition note remains
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(Equal("vocab refit applied: version → 2.0 (K=2, matched=2, new=0, retired=0)\n"))
	g.Expect(vault.exists(definition)).To(BeTrue())
	g.Expect(vault.exists(embed.SidecarPath(definition))).To(BeTrue())

	// And its recorded origin is still proposed
	centroids := cv2ReadCentroids(g, vault)
	terms := cv1JSONField[map[string]map[string]json.RawMessage](g, centroids, "terms")
	g.Expect(terms).To(HaveKey("gardening"))
	g.Expect(cv1JSONField[string](g, terms["gardening"], "origin")).To(Equal("proposed"))

	// And engram vocab stats still lists gardening
	g.Expect(vault.run("vocab", "stats").stdout).To(MatchRegexp(`(?m)^  gardening: [0-9]+ members`))
}

// Test792a6r1a_VocabRefit_ThePlanFlowIsGone verifies 792a6r1a (controlled-vocabulary.md:482): over a vault, engram
// vocab refit --plan x and --emit-request x each fail saying the flag provided is not defined, and write nothing.
func Test792a6r1a_VocabRefit_ThePlanFlowIsGone(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"--plan", "--emit-request"} {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a vault
			vault := newCV1TwoTopicVault()
			vault.bootstrap(cv1BuildTooling, cv1Cooking)

			before := vault.fsys.snapshot()
			vault.fsys.resetLog()

			// When the user runs engram vocab refit with the flag and a value
			result := vault.run("vocab", "refit", flag, "x")

			// Then it fails saying the flag provided is not defined, and nothing is written
			g.Expect(result.stderr).To(ContainSubstring("flag provided but not defined: " + flag))
			g.Expect(result.stdout).To(BeEmpty())
			g.Expect(vault.fsys.writeCount()).To(BeZero())
			g.Expect(vault.fsys.removeCount()).To(BeZero())
			g.Expect(vault.fsys.snapshot()).To(Equal(before))
		})
	}
}

// Test792a6s1a_LearnFact_GrowthMarksARefitPending verifies 792a6s1a (controlled-vocabulary.md:510): over a vault whose
// refit baseline is 100 notes 20 days ago, now holding 149 notes, engram learn fact makes 150 and engram vocab stats
// reports "verdict: REFIT_PENDING (growth: 50 notes, 20 days)".
func Test792a6s1a_LearnFact_GrowthMarksARefitPending(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose refit baseline is 100 notes 20 days ago, now holding 149 notes
	vault := newCV2GrowthVault(g, cv2BaselineNotes, cv2BaselineNotes+cv2PendingGrowth-1, cv2PendingDays)
	g.Expect(vault.run("vocab", "stats").stdout).To(ContainSubstring("verdict: OK\n"))

	// When the user runs engram learn fact
	result := vault.learnFact("growth-last", cv1BuildSituations[0])
	g.Expect(result.stderr).To(BeEmpty())

	// Then engram vocab stats reports the pending refit with the growth and the days
	stats := vault.run("vocab", "stats").stdout
	g.Expect(stats).To(ContainSubstring("verdict: REFIT_PENDING (growth: 50 notes, 20 days)\n"))
}

// Test792a6s1b_LearnFact_GrowthWithoutEnoughTimeDoesNotTrigger verifies 792a6s1b (controlled-vocabulary.md:515): over
// a vault whose refit baseline is 100 notes 5 days ago, now holding 149 notes, engram learn fact makes 150 and engram
// vocab stats reports verdict: OK.
func Test792a6s1b_LearnFact_GrowthWithoutEnoughTimeDoesNotTrigger(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose refit baseline is 100 notes 5 days ago, now holding 149 notes
	vault := newCV2GrowthVault(g, cv2BaselineNotes, cv2BaselineNotes+cv2PendingGrowth-1, cv2TooSoonDays)

	// When the user runs engram learn fact
	result := vault.learnFact("growth-last", cv1BuildSituations[0])
	g.Expect(result.stderr).To(BeEmpty())

	// Then engram vocab stats reports verdict: OK
	stats := vault.run("vocab", "stats").stdout
	g.Expect(stats).To(ContainSubstring("verdict: OK\n"))
	g.Expect(stats).NotTo(ContainSubstring("REFIT_PENDING"))
}

// Test792a6s1c_LearnFact_AFirstWriteRecordsTheBaseline verifies 792a6s1c (controlled-vocabulary.md:520): over a vault
// with no refit baseline, engram learn fact records one (the number of member notes and today's date), and engram
// vocab stats reports verdict: OK.
func Test792a6s1c_LearnFact_AFirstWriteRecordsTheBaseline(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault of three notes with no refit baseline
	vault := newCV1Vault()
	vault.learnTopicNotes("early", cv1BuildSituations[:cv2EarlyNotes])

	if vault.exists(cv2CentroidsFile) {
		g.Expect(vault.fsys.Remove(filepath.Join(cv1VaultPath, cv2CentroidsFile))).To(Succeed())
	}

	g.Expect(vault.exists(cv2CentroidsFile)).To(BeFalse())

	// When the user runs engram learn fact
	result := vault.learnFact("early-last", cv1BuildSituations[0])
	g.Expect(result.stderr).To(BeEmpty())

	// Then a baseline is recorded: the four notes held, dated today
	g.Expect(vault.exists(cv2CentroidsFile)).To(BeTrue())

	baseline := cv1JSONField[map[string]json.RawMessage](g, cv2ReadCentroids(g, vault), "last_refit")
	g.Expect(cv1JSONField[int](g, baseline, "note_count")).To(Equal(cv2EarlyNotes + 1))
	g.Expect(cv1JSONField[string](g, baseline, "date")).To(Equal(cv1Today))

	// And engram vocab stats reports verdict: OK
	g.Expect(vault.run("vocab", "stats").stdout).To(ContainSubstring("verdict: OK\n"))
}

// Test792a6s1d_LearnFact_AHighUntaggedRateAloneDoesNotTrigger verifies 792a6s1d (controlled-vocabulary.md:525): over
// a vault where more than 8% of the notes are untagged and growth is below 40 notes, writing a note leaves engram
// vocab stats reporting the untagged rate with [high] and verdict: OK.
func Test792a6s1d_LearnFact_AHighUntaggedRateAloneDoesNotTrigger(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with the term cooking, five cooking notes and two notes close to no term
	vault := newCV1Vault()
	vault.bootstrap(cv1Cooking)
	vault.learnTopicNotes("cooking", cv1CookingSituations)
	vault.learnFact("tls-first", "configuring a TLS certificate")
	vault.learnFact("tls-second", "rotating a TLS certificate")

	// When the user writes another note close to no term
	result := vault.learnFact("tls-third", "revoking a TLS certificate")
	g.Expect(result.stderr).To(BeEmpty())

	// Then engram vocab stats reports the untagged rate with [high] and verdict: OK
	stats := vault.run("vocab", "stats").stdout
	g.Expect(stats).To(MatchRegexp(`(?m)^untagged-rate: [0-9.]+% \[high\]$`))
	g.Expect(stats).To(ContainSubstring("untagged: 3\n"))
	g.Expect(stats).To(ContainSubstring("verdict: OK\n"))
}

// Test792a6t1b_VocabStats_HubsAndOrphansAreFlagged verifies 792a6t1b (controlled-vocabulary.md:556): over a vault with
// term a tagging 3 of 4 member notes and term b tagging none, engram vocab stats prints "  a: 3 members [hub]" and
// "  b: 0 members [orphan]".
func Test792a6t1b_VocabStats_HubsAndOrphansAreFlagged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with term a tagging 3 of 4 member notes and term b tagging none
	vault := newCV1Vault()
	vault.bootstrap(
		cli.SeedTerm{Term: "a", Description: "linters and compilers"},
		cli.SeedTerm{Term: "b", Description: "telescopes and galaxies"},
	)
	vault.learnTopicNotes("a", cv1BuildSituations[:cv2HubMembers])
	vault.learnFact("other-note", cv1CookingSituations[0])

	// When the user runs engram vocab stats
	result := vault.run("vocab", "stats")

	// Then it prints the hub line for a and the orphan line for b
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(ContainSubstring("terms: 2  member-notes: 4  untagged: 1\n"))
	g.Expect(result.stdout).To(ContainSubstring("\n  a: 3 members [hub]\n"))
	g.Expect(result.stdout).To(ContainSubstring("\n  b: 0 members [orphan]\n"))
}

// Test792a6t1c_VocabStats_QuestionsAndDefinitionsAreNotMemberNotes verifies 792a6t1c (controlled-vocabulary.md:561):
// over a vault with one definition note, one fact note and a question-and-answer pair, engram vocab stats counts the
// fact and the answer as member notes only, and prints "qa pairs: 1" and "qa round-2 gate: accumulating (1/20)".
func Test792a6t1c_VocabStats_QuestionsAndDefinitionsAreNotMemberNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with one definition note (the term cooking), one fact note and a question-and-answer pair
	vault := newCV1Vault()
	vault.bootstrap(cv1Cooking)
	vault.learnFact("a-fact", cv1CookingSituations[0])

	qa := vault.run("learn", "qa", "--slug", "scaling", "--question", "How do I scale a recipe?",
		"--answer", "Multiply every quantity by the ratio of guests.", "--source", cv1Source)
	g.Expect(qa.stderr).To(BeEmpty())
	g.Expect(vault.namesMatching(`\.q\.md$`)).To(HaveLen(1))
	g.Expect(vault.namesMatching(`\.a\.md$`)).To(HaveLen(1))

	// When the user runs engram vocab stats
	result := vault.run("vocab", "stats")

	// Then member-notes counts the fact and the answer only, and the pair is counted
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(result.stdout).To(MatchRegexp(`(?m)^terms: 1  member-notes: 2  untagged: [0-9]+$`))
	g.Expect(result.stdout).To(ContainSubstring("qa pairs: 1\n"))
	g.Expect(result.stdout).To(ContainSubstring("qa round-2 gate: accumulating (1/20)\n"))
}

// Test792a6t1d_VocabStats_APendingRefitIsTheVerdict verifies 792a6t1d (controlled-vocabulary.md:566): over a vault
// whose write marked a refit pending for "growth: 45 notes, 20 days", engram vocab stats prints "verdict:
// REFIT_PENDING (growth: 45 notes, 20 days)".
func Test792a6t1d_VocabStats_APendingRefitIsTheVerdict(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose baseline is 6 notes 20 days ago and which the last write grew to 51 notes
	vault := newCV2GrowthVault(g, cv2ShortVaultBaseline, cv2ShortVaultNotes, cv2PendingDays)

	result := vault.learnFact("growth-last", cv1BuildSituations[0])
	g.Expect(result.stderr).To(BeEmpty())

	// When the user runs engram vocab stats
	stats := vault.run("vocab", "stats")

	// Then it prints the verdict with the reason
	g.Expect(stats.stderr).To(BeEmpty())
	g.Expect(stats.stdout).To(ContainSubstring("verdict: REFIT_PENDING (growth: 45 notes, 20 days)\n"))
}

// Test792a6u1a_VocabTagDefinitions_BackfillAddsMissingSelfTags verifies 792a6u1a (controlled-vocabulary.md:589): over
// a vault whose definition notes carry only the bare vocab tag, engram vocab tag-definitions makes each per-term
// definition's tags [vocab, vocab/<term>], leaves none of them stale in engram embed status, and prints one "added"
// line per note.
func Test792a6u1a_VocabTagDefinitions_BackfillAddsMissingSelfTags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose definition notes carry only the bare vocab tag
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking, cv1Astronomy)

	terms := []string{"astronomy", "build-tooling", "cooking"}
	definitions := cv2StripSelfTags(g, vault, terms)
	g.Expect(definitions).To(HaveLen(len(terms)))

	for _, definition := range definitions {
		g.Expect(parseCV1Front(vault.read(definition)).Tags).To(Equal([]string{"vocab"}))
	}

	// When the user runs engram vocab tag-definitions
	result := vault.run("vocab", "tag-definitions")
	g.Expect(result.stderr).To(BeEmpty())

	// Then each per-term definition's tags become [vocab, vocab/<term>]
	for index, definition := range definitions {
		g.Expect(parseCV1Front(vault.read(definition)).Tags).To(Equal([]string{"vocab", "vocab/" + terms[index]}))
		g.Expect(result.stdout).To(MatchRegexp(`(?m)^` + regexp.QuoteMeta(definition) + `: added$`))
	}

	// And one "added" line is printed per note
	g.Expect(strings.Count(result.stdout, ": added\n")).To(Equal(len(terms)))

	// And engram embed status reports none of them stale
	status := vault.run("embed", "status").stdout
	g.Expect(status).To(MatchRegexp(`(?m)^stale: +0$`))
	g.Expect(status).To(MatchRegexp(`(?m)^without: +0$`))
}

// Test792a6u1b_VocabTagDefinitions_BackfillIsIdempotent verifies 792a6u1b (controlled-vocabulary.md:594): over a vault
// on which engram vocab tag-definitions has run, running it a second time changes no note, reports every definition
// "already present" and the family note "family note (skipped)".
func Test792a6u1b_VocabTagDefinitions_BackfillIsIdempotent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault on which engram vocab tag-definitions has run
	vault := newCV1Vault()
	vault.bootstrap(cv1BuildTooling, cv1Cooking)

	terms := []string{"build-tooling", "cooking"}
	definitions := cv2StripSelfTags(g, vault, terms)
	g.Expect(vault.run("vocab", "tag-definitions").stdout).To(ContainSubstring(": added\n"))

	before := vault.fsys.snapshot()
	vault.fsys.resetLog()

	// When the user runs it a second time
	result := vault.run("vocab", "tag-definitions")

	// Then no note changes
	g.Expect(result.stderr).To(BeEmpty())
	g.Expect(vault.fsys.writeCount()).To(BeZero())
	g.Expect(vault.fsys.snapshot()).To(Equal(before))

	// And every definition is reported "already present", the family note "family note (skipped)"
	for _, definition := range definitions {
		g.Expect(result.stdout).To(MatchRegexp(`(?m)^` + regexp.QuoteMeta(definition) + `: already present$`))
	}

	family := vault.requireNote(g, `\.vocab-definition\.md$`)
	g.Expect(result.stdout).To(MatchRegexp(`(?m)^` + regexp.QuoteMeta(family) + `: family note \(skipped\)$`))
	g.Expect(strings.Count(result.stdout, "\n")).To(Equal(len(terms) + 1))
}

// unexported constants.
const (
	// cv2BaselineNotes is the number of notes of the refit baseline of the growth scenarios.
	cv2BaselineNotes = 100
	cv2CentroidsFile = "vocab.centroids.json"
	// cv2EarlyNotes is the number of notes of the vault whose baseline is missing.
	cv2EarlyNotes = 3
	// cv2HubMembers is the number of member notes the term a tags: three of the four member notes.
	cv2HubMembers = 3
	// cv2NamesPath is where the answer of the naming requests is.
	cv2NamesPath = "/names.json"
	// cv2NewCluster is the index of the cluster the cooking notes make: the build notes are cluster 0.
	cv2NewCluster = 1
	// cv2PendingDays is the days since the baseline that arm the growth trigger (14 or more).
	cv2PendingDays = 20
	// cv2PendingGrowth is the growth since the baseline that arms the growth trigger (40 or more) in the scenarios.
	cv2PendingGrowth = 50
	// cv2ShortVaultBaseline and cv2ShortVaultNotes make a growth of 45 notes: the baseline of 6 notes, 50 notes and one
	// more write.
	cv2ShortVaultBaseline = 6
	cv2ShortVaultNotes    = 50
	cv2StaleFingerprint   = "n10-0000000000000000"
	// cv2TooSoonDays is the days since the baseline that do not arm the growth trigger.
	cv2TooSoonDays = 5
	// cv2UnknownCluster is an index no cluster of the vault has.
	cv2UnknownCluster = 7
)

// cv2NameEntry is one entry of the --names answer of a refit.
type cv2NameEntry struct {
	Cluster     int    `json:"cluster"`
	Term        string `json:"term"`
	Description string `json:"description"`
}

// cv2NamesFile is the --names answer of a refit.
type cv2NamesFile struct {
	Names       []cv2NameEntry `json:"names"`
	Fingerprint string         `json:"fingerprint"`
}

// cv2EmitNaming runs engram vocab refit over a vault with a cluster to name, and returns the index of the cluster the
// naming request is for and the fingerprint the answer must echo.
func cv2EmitNaming(g *WithT, vault *cv1Vault) (cluster int, fingerprint string) {
	result := vault.refit()
	g.Expect(result.stderr).To(BeEmpty())

	var document map[string]json.RawMessage

	g.Expect(json.Unmarshal([]byte(result.stdout), &document)).To(Succeed())

	requests := cv1JSONField[[]map[string]json.RawMessage](g, document, "naming_requests")
	g.Expect(requests).To(HaveLen(1))

	if len(requests) != 1 {
		return 0, ""
	}

	return cv1JSONField[int](g, requests[0], "cluster"), cv1JSONField[string](g, document, "fingerprint")
}

// cv2LearnPointerNote learns a note about builds whose body links to the definition and which names it in a
// supersedes: entry, and tags it vocab/astronomy as an earlier write could have. It returns the note's content.
func cv2LearnPointerNote(g *WithT, vault *cv1Vault, slug, basename, definition string) string {
	situation := "running the linter and the compiler in the build-tooling routine with the linter and the compiler"

	result := vault.run("learn", "fact", "--slug", slug, "--position", "top", "--source", cv1Source,
		"--situation", situation, "--subject", "the team", "--predicate", "follows",
		"--object", "the routine in [["+basename+"]]",
		"--supersedes", definition+"|updates|the astronomy term is obsolete")
	g.Expect(result.stderr).To(BeEmpty())

	name := vault.requireNote(g, `\.`+regexp.QuoteMeta(slug)+`\.md$`)
	content := vault.read(name)
	tagged := strings.Replace(content, "tags:\n", "tags:\n    - vocab/astronomy\n", 1)
	g.Expect(tagged).NotTo(Equal(content), "the learned note must carry a tags: field to add the tag to")

	vault.fsys.put(filepath.Join(cv1VaultPath, name), []byte(tagged))

	return tagged
}

// cv2NamesJSON is the --names answer with the fingerprint and the entries.
func cv2NamesJSON(g *WithT, fingerprint string, entries ...cv2NameEntry) string {
	data, err := json.Marshal(cv2NamesFile{Names: entries, Fingerprint: fingerprint})
	g.Expect(err).NotTo(HaveOccurred())

	return string(data)
}

// cv2ReadCentroids is the centroids file of the vault, by key.
func cv2ReadCentroids(g *WithT, vault *cv1Vault) map[string]json.RawMessage {
	var document map[string]json.RawMessage

	g.Expect(json.Unmarshal([]byte(vault.read(cv2CentroidsFile)), &document)).To(Succeed())

	if document == nil {
		return map[string]json.RawMessage{}
	}

	return document
}

// cv2RunAt runs `engram <args> --vault /vault` as cv1Vault.run does, with the clock of the command at now, for a
// vault whose earlier notes were written on an earlier day.
func cv2RunAt(vault *cv1Vault, now time.Time, args ...string) cv1Result {
	var stdout, stderr bytes.Buffer

	deps := cli.Deps{
		Stdout:       &stdout,
		Stderr:       &stderr,
		Exit:         func(int) {},
		Getenv:       func(string) string { return "" },
		Now:          func() time.Time { return now },
		Getwd:        func() (string, error) { return "/", nil },
		UserHomeDir:  func() (string, error) { return "/home/cv1", nil },
		Username:     func() (string, error) { return "cv1-user", nil },
		FS:           vault.fsys,
		Lock:         fakeLocker{lock: func(string) (func() error, error) { return func() error { return nil }, nil }},
		Commander:    stubCommander{},
		Embed:        vault.embedder,
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

	return cv1Result{stdout: stdout.String(), stderr: stderr.String()}
}

// cv2StripSelfTags rewrites the definition note of each term as the earlier layout held it, tagged only vocab, and
// returns the names of the notes in the order of the terms.
func cv2StripSelfTags(g *WithT, vault *cv1Vault, terms []string) []string {
	names := make([]string, 0, len(terms))

	for _, term := range terms {
		name := vault.requireNote(g, `\.vocab-`+regexp.QuoteMeta(term)+`-definition\.md$`)
		content := vault.read(name)
		stripped := strings.Replace(content, "    - vocab/"+term+"\n", "", 1)
		g.Expect(stripped).NotTo(Equal(content), "the definition of %s must carry its self-tag", term)

		vault.fsys.put(filepath.Join(cv1VaultPath, name), []byte(stripped))

		names = append(names, name)
	}

	return names
}

// newCV2GrowthVault is a vault of existing notes about builds, written daysAgo days before the vault clock, whose
// vocabulary was bootstrapped with the term build-tooling when the vault held baselineNotes of them: the bootstrap
// records the refit baseline, and the later writes are the growth since.
func newCV2GrowthVault(g *WithT, baselineNotes, existing, daysAgo int) *cv1Vault {
	vault := newCV1Vault()
	earlier := cv1Clock.AddDate(0, 0, -daysAgo)

	for index := range existing {
		learned := cv2RunAt(vault, earlier, "learn", "fact", "--slug", cv1NoteSlug("growth", index), "--position", "top",
			"--source", cv1Source, "--situation", cv1BuildSituations[index%len(cv1BuildSituations)],
			"--subject", "the team", "--predicate", "follows", "--object", "the routine")
		g.Expect(learned.stderr).To(BeEmpty())

		if index+1 == baselineNotes {
			seed, err := yaml.Marshal([]cli.SeedTerm{cv1BuildTooling})
			if err != nil {
				return vault
			}

			vault.fsys.put(cv1SeedPath, seed)
			bootstrapped := cv2RunAt(vault, earlier, "vocab", "bootstrap", "--seed", cv1SeedPath)
			g.Expect(bootstrapped.stderr).To(BeEmpty())
		}
	}

	return vault
}

// newCV2NamingVault is a vault at version 1.0 whose derivation matches the build notes to build-tooling and finds the
// cooking notes as one new cluster.
func newCV2NamingVault() *cv1Vault {
	vault := newCV1TwoTopicVault()
	vault.bootstrap(cv1BuildTooling)

	return vault
}
