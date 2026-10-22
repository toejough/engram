package cli_test

// Scenario tests for the Memory Query product capability, part 5 (861c2zp): trigger text (792a7y), a trigger hit
// comes first (792a7za), triggers match whole words (792a7zb), the embedding model (792a7zc), a pending refit
// (792a7zd) and the stage timings (792a7ze). Spec: docs/specs/product/memory-query.md.
//
// The scenarios run cli.RunQuery over the in-memory world of memory_query_part1_scenarios_test.go (mq1World, a
// phrase-keyed embedder, so every cosine is a number the test chose). That embedder refuses any text it was not
// given as a phrase, so a trigger-only query (--text and no --phrase) that succeeded in a world with no phrases
// embedded nothing. The helpers carry the mq5 prefix.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a7y1a_RunQuery_TextAloneIsATriggerLookup verifies 792a7y1a (Text alone is a trigger lookup): over a
// vault with a runbook with triggers ["/release"] and a fact, engram query --text "please /Release it now" with
// no phrase returns only that runbook, with provenances [trigger] and score 0, and empty phrases and clusters.
func Test792a7y1a_RunQuery_TextAloneIsATriggerLookup(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const runbookName = "1.release-runbook"

	// Given a runbook with triggers ["/release"] (and a fact the text does not mention); the world declares no
	// phrase, so its embedder fails on any text it is asked to embed.
	world := newMq1World(mq1SmallDims)
	mq5Plant(world, runbookName, mq5Content(mq5Note{kind: "runbook", triggers: []string{"/release"}}),
		map[int]float64{})
	mq5Plant(world, "2.some-fact", mq5Content(mq5Note{kind: "fact"}), map[int]float64{})

	// When the user runs engram query --text "please /Release it now".
	outcome := mq5Run(t, world, cli.QueryArgs{Text: "please /Release it now"}, mq5Hooks{})

	// Then items holds only that runbook, with provenances [trigger] and score 0, and phrases and clusters are empty.
	g.Expect(outcome.payload.Items).To(HaveLen(1))

	if len(outcome.payload.Items) != 1 {
		return
	}

	item := outcome.payload.Items[0]
	g.Expect(item.Path).To(Equal(runbookName + ".md"))
	g.Expect(item.Kind).To(Equal("runbook"))
	g.Expect(item.Provenances).To(Equal([]string{"trigger"}))
	g.Expect(item.Score).To(BeZero())
	g.Expect(outcome.payload.Phrases).To(BeEmpty())
	g.Expect(outcome.payload.Clusters).To(BeEmpty())
}

// Test792a7y1b_RunQuery_WithoutTextNothingChanges verifies 792a7y1b (Without text nothing changes): over a vault
// whose runbook carries triggers, engram query with a phrase and no --text ranks the notes by similarity alone,
// no item has the provenance trigger, the payload has the keys it had before --text existed, and it equals the
// payload of a query whose --text hits nothing.
func Test792a7y1b_RunQuery_WithoutTextNothingChanges(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "releasing a module"
		runbookName = "1.release-runbook"
		factName    = "2.release-fact"
	)

	// Given a vault whose runbooks carry triggers.
	world := newMq1World(mq1SmallDims, phrase)
	mq5Plant(world, runbookName, mq5Content(mq5Note{kind: "runbook", triggers: []string{"/release"}}),
		map[int]float64{0: mq5Top})
	mq5Plant(world, factName, mq5Content(mq5Note{kind: "fact"}), map[int]float64{0: mq5Mid})

	// When the user runs engram query with phrases and no --text.
	outcome := mq5Run(t, world, cli.QueryArgs{Phrases: []string{phrase}}, mq5Hooks{})

	// Then the notes rank by similarity alone and no item has provenance trigger,
	g.Expect(mq5Paths(outcome.payload)).To(Equal([]string{runbookName + ".md", factName + ".md"}))

	for _, item := range outcome.payload.Items {
		g.Expect(item.Provenances).To(Equal([]string{"direct"}), item.Path)
	}

	// and the payload is the one it was before --text existed: the same keys, and the same document as a query
	// whose message mentions no trigger.
	keys := make([]string, 0, len(outcome.raw))
	for key := range outcome.raw {
		keys = append(keys, key)
	}

	g.Expect(keys).To(ConsistOf("version", "phrases", "items", "clusters", "budget", "model_id"))

	missed := mq5Run(t, world, cli.QueryArgs{Phrases: []string{phrase}, Text: "nothing relevant here"}, mq5Hooks{})
	g.Expect(missed.stdout).To(Equal(outcome.stdout))
}

// Test792a7y1c_RunQuery_TextIsNotMatchedBySimilarity verifies 792a7y1c (Text is not matched by similarity): a
// runbook whose situation embeds exactly where the message does, but whose triggers are not in it, is not
// returned by engram query --text, whether the query has no phrase or an unrelated one.
func Test792a7y1c_RunQuery_TextIsNotMatchedBySimilarity(t *testing.T) {
	t.Parallel()

	const (
		otherPhrase = "an unrelated phrase"
		text        = "shipping a tagged module version"
	)

	cases := map[string]cli.QueryArgs{
		"no phrase":        {Text: text},
		"an unrelated one": {Text: text, Phrases: []string{otherPhrase}},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a runbook whose situation is close to the text but whose triggers are not in it: the text is
			// declared as a phrase (axis 1), so were it embedded, the runbook would match it at cosine 1.
			world := newMq1World(mq1SmallDims, otherPhrase, text)
			mq5Plant(world, "1.release-runbook", mq5Content(mq5Note{kind: "runbook", triggers: []string{"/release"}}),
				map[int]float64{1: 1})

			// When the user runs engram query --text "<that text>".
			outcome := mq5Run(t, world, args, mq5Hooks{})

			// Then the runbook is not returned.
			g.Expect(outcome.payload.Items).To(BeEmpty())
		})
	}
}

// Test792a7za1a_RunQuery_AHitWithNoSimilarityMatchIsFirst verifies 792a7za1a (A hit with no similarity match is
// first): a runbook with triggers ["/please"] that no phrase matches above the floor is items[0] of engram query
// --text "/please fix the flaky login test" --phrase "fixing a flaky test", with provenances [trigger] and kind
// runbook, ahead of the fact the phrase does match.
func Test792a7za1a_RunQuery_AHitWithNoSimilarityMatchIsFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "fixing a flaky test"
		runbookName = "1.drive-ask"
		factName    = "2.flaky-fact"
	)

	// Given a runbook with triggers ["/please"] and no similarity match above the floor.
	world := newMq1World(mq1SmallDims, phrase)
	mq5Plant(world, runbookName, mq5Content(mq5Note{kind: "runbook", triggers: []string{"/please"}}),
		map[int]float64{})
	mq5Plant(world, factName, mq5Content(mq5Note{kind: "fact"}), map[int]float64{0: mq5Top})

	// When the user runs engram query --text "/please fix the flaky login test" --phrase "fixing a flaky test".
	outcome := mq5Run(t, world,
		cli.QueryArgs{Text: "/please fix the flaky login test", Phrases: []string{phrase}}, mq5Hooks{})

	// Then the runbook is items[0] with provenances [trigger] and kind runbook.
	g.Expect(mq5Paths(outcome.payload)).To(Equal([]string{runbookName + ".md", factName + ".md"}))

	if len(outcome.payload.Items) == 0 {
		return
	}

	first := outcome.payload.Items[0]
	g.Expect(first.Provenances).To(Equal([]string{"trigger"}))
	g.Expect(first.Kind).To(Equal("runbook"))
}

// Test792a7za1b_RunQuery_AHitOutranksAStrongerMatch verifies 792a7za1b (A hit outranks a stronger match): a
// feedback note that scores 0.9 and a runbook that scores 0.3 and is a trigger hit come back runbook first when
// the query has the message and a phrase, though without the message the feedback note ranks first.
func Test792a7za1b_RunQuery_AHitOutranksAStrongerMatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase       = "fixing a flaky test"
		feedbackName = "2.flaky-feedback"
		runbookName  = "1.drive-ask"
	)

	// Given a feedback note that scores 0.9 and a runbook that scores 0.3 and is a trigger hit.
	world := newMq1World(mq1SmallDims, phrase)
	mq5Plant(world, feedbackName, mq5Content(mq5Note{kind: "feedback"}), map[int]float64{0: mq5Top})
	mq5Plant(world, runbookName, mq5Content(mq5Note{kind: "runbook", triggers: []string{"/please"}}),
		map[int]float64{0: mq5Low})

	// Without the message the stronger match ranks first.
	plain := mq5Run(t, world, cli.QueryArgs{Phrases: []string{phrase}}, mq5Hooks{})
	g.Expect(mq5Paths(plain.payload)).To(Equal([]string{feedbackName + ".md", runbookName + ".md"}))

	// When the user runs engram query with the message and a phrase.
	outcome := mq5Run(t, world, cli.QueryArgs{Text: "/please fix it", Phrases: []string{phrase}}, mq5Hooks{})

	// Then the runbook precedes the feedback note.
	g.Expect(mq5Paths(outcome.payload)).To(Equal([]string{runbookName + ".md", feedbackName + ".md"}))
}

// Test792a7za1c_RunQuery_OtherKindsNeverHit verifies 792a7za1c (Other kinds never hit): a fact note, or a
// feedback note, whose frontmatter has a triggers list matching the text has no trigger provenance, is not
// returned by --text alone, and with a phrase ranks by similarity only, below a better match.
func Test792a7za1c_RunQuery_OtherKindsNeverHit(t *testing.T) {
	t.Parallel()

	const (
		phrase    = "fixing a flaky test"
		text      = "/please fix the flaky login test"
		bestName  = "1.best-fact"
		otherName = "2.other-note"
	)

	for _, kind := range []string{"fact", "feedback"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a note of another kind whose frontmatter has a triggers list matching the text.
			world := newMq1World(mq1SmallDims, phrase)
			mq5Plant(world, bestName, mq5Content(mq5Note{kind: "fact"}), map[int]float64{0: mq5Top})
			mq5Plant(world, otherName, mq5Content(mq5Note{kind: kind, triggers: []string{"/please"}}),
				map[int]float64{0: mq5Mid})

			// When the user runs engram query --text "<the text>".
			textOnly := mq5Run(t, world, cli.QueryArgs{Text: text}, mq5Hooks{})

			// Then it is not a hit, and with a phrase it has no trigger provenance and ranks by similarity only.
			g.Expect(textOnly.payload.Items).To(BeEmpty())

			withPhrase := mq5Run(t, world, cli.QueryArgs{Text: text, Phrases: []string{phrase}}, mq5Hooks{})
			g.Expect(mq5Paths(withPhrase.payload)).To(Equal([]string{bestName + ".md", otherName + ".md"}))

			for _, item := range withPhrase.payload.Items {
				g.Expect(item.Provenances).To(Equal([]string{"direct"}), item.Path)
			}
		})
	}
}

// Test792a7za1d_RunQuery_AHitThatMatchedKeepsItsScore verifies 792a7za1d (A hit that matched keeps its score): a
// runbook that matches the phrase at cosine 0.7 and is a trigger hit has the provenances direct and trigger and
// its similarity score.
func Test792a7za1d_RunQuery_AHitThatMatchedKeepsItsScore(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "releasing a module"
		runbookName = "1.release-runbook"
	)

	// Given a runbook that both matches a phrase and is a trigger hit.
	world := newMq1World(mq1SmallDims, phrase)
	mq5Plant(world, runbookName, mq5Content(mq5Note{kind: "runbook", triggers: []string{"/release"}}),
		map[int]float64{0: mq5Mid})

	// When the user runs engram query with the phrase and the message.
	outcome := mq5Run(t, world, cli.QueryArgs{Phrases: []string{phrase}, Text: "do /release now"}, mq5Hooks{})

	// Then its provenances include direct and trigger and its score is its similarity score.
	g.Expect(outcome.payload.Items).To(HaveLen(1))

	if len(outcome.payload.Items) != 1 {
		return
	}

	item := outcome.payload.Items[0]
	g.Expect(item.Path).To(Equal(runbookName + ".md"))
	g.Expect(item.Provenances).To(ConsistOf("direct", "trigger"))
	g.Expect(float64(item.Score)).To(BeNumerically("~", mq5Mid, mq1Tolerance))
}

// Test792a7za1e_RunQuery_AHitOutsideTheProjectIsNotSurfaced verifies 792a7za1e (A hit outside the project is not
// surfaced): a runbook trigger hit whose project is beta is not returned by --project alpha, whether the
// runbook matched the phrase or not, and is returned by --project beta.
func Test792a7za1e_RunQuery_AHitOutsideTheProjectIsNotSurfaced(t *testing.T) {
	t.Parallel()

	const (
		phrase      = "releasing a module"
		runbookName = "1.release-runbook"
	)

	cases := map[string]struct {
		cosines map[int]float64
		phrases []string
	}{
		"a hit that matched the phrase": {cosines: map[int]float64{0: mq5Top}, phrases: []string{phrase}},
		"a hit that did not":            {cosines: map[int]float64{}, phrases: []string{phrase}},
		"a hit of a trigger lookup":     {cosines: map[int]float64{}, phrases: nil},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a runbook trigger hit whose project is beta.
			world := newMq1World(mq1SmallDims, phrase)
			mq5Plant(world, runbookName,
				mq5Content(mq5Note{kind: "runbook", triggers: []string{"/release"}, project: "beta"}), tc.cosines)

			// When the user runs engram query --project alpha with the message.
			outside := mq5Run(t, world,
				cli.QueryArgs{Phrases: tc.phrases, Text: "do /release now", Project: "alpha"}, mq5Hooks{})

			// Then the runbook is not returned.
			g.Expect(outside.payload.Items).To(BeEmpty())

			// (control) --project beta returns it as a trigger hit.
			inside := mq5Run(t, world,
				cli.QueryArgs{Phrases: tc.phrases, Text: "do /release now", Project: "beta"}, mq5Hooks{})
			g.Expect(mq5Paths(inside.payload)).To(Equal([]string{runbookName + ".md"}))
			g.Expect(inside.payload.Items[0].Provenances).To(ContainElement("trigger"))
		})
	}
}

// Test792a7zb1a_RunQuery_CaseAndWhitespaceAreIgnored verifies 792a7zb1a (Case and whitespace are ignored): for
// triggers ["take this end-to-end"], the message "PLEASE   take this end-to-end: rename X" is a trigger hit, as
// are the same message with tabs and a line break between its words, and a trigger written in capitals with
// runs of blanks against a plain message.
func Test792a7zb1a_RunQuery_CaseAndWhitespaceAreIgnored(t *testing.T) {
	t.Parallel()

	const trigger = "take this end-to-end"

	mq5RunHitCases(t, []mq5HitCase{
		{trigger: trigger, text: "PLEASE   take this end-to-end: rename X", hit: true},
		{trigger: trigger, text: "please\ttake \n this\t\tend-to-end: rename X", hit: true},
		{trigger: "Take  THIS   end-to-end", text: "please take this end-to-end", hit: true},
	})
}

// Test792a7zb1b_RunQuery_ABareWordHitsAsAWholeWord verifies 792a7zb1b (A bare word hits as a whole word): for
// triggers ["curate"], the messages "please curate the offers", "Curate!" and "(curate)" are trigger hits.
func Test792a7zb1b_RunQuery_ABareWordHitsAsAWholeWord(t *testing.T) {
	t.Parallel()

	mq5RunHitCases(t, []mq5HitCase{
		{trigger: "curate", text: "please curate the offers", hit: true},
		{trigger: "curate", text: "Curate!", hit: true},
		{trigger: "curate", text: "(curate)", hit: true},
	})
}

// Test792a7zb1c_RunQuery_ABareWordDoesNotHitInsideALongerWord verifies 792a7zb1c (A bare word does not hit inside
// a longer word): for triggers ["curate"], the messages "that is accurate", "inaccurate", "curated offers" and
// "curates" are not trigger hits.
func Test792a7zb1c_RunQuery_ABareWordDoesNotHitInsideALongerWord(t *testing.T) {
	t.Parallel()

	mq5RunHitCases(t, []mq5HitCase{
		{trigger: "curate", text: "that is accurate", hit: false},
		{trigger: "curate", text: "inaccurate", hit: false},
		{trigger: "curate", text: "curated offers", hit: false},
		{trigger: "curate", text: "curates", hit: false},
	})
}

// Test792a7zb1d_RunQuery_ASlashFormNeedsOnlyARightBoundary verifies 792a7zb1d (A slash form needs only a right
// boundary): for triggers ["/please"], the messages "/please fix it" and "do /please" are trigger hits, as is
// "do/please" (no left boundary is required), and "/pleased" is not.
func Test792a7zb1d_RunQuery_ASlashFormNeedsOnlyARightBoundary(t *testing.T) {
	t.Parallel()

	mq5RunHitCases(t, []mq5HitCase{
		{trigger: "/please", text: "/please fix it", hit: true},
		{trigger: "/please", text: "do /please", hit: true},
		{trigger: "/please", text: "do/please", hit: true},
		{trigger: "/please", text: "/pleased", hit: false},
	})
}

// Test792a7zb1e_RunQuery_ASlashBeforeABareWordIsABoundary verifies 792a7zb1e (A slash before a bare word is a
// boundary): for triggers ["curate"], the message "/curate" is a trigger hit.
func Test792a7zb1e_RunQuery_ASlashBeforeABareWordIsABoundary(t *testing.T) {
	t.Parallel()

	mq5RunHitCases(t, []mq5HitCase{
		{trigger: "curate", text: "/curate", hit: true},
	})
}

// Test792a7zb1f_RunQuery_TheEdgesOfTheTextAreBoundaries verifies 792a7zb1f (The edges of the text are
// boundaries): for triggers ["curate"], the message "curate" is a trigger hit.
func Test792a7zb1f_RunQuery_TheEdgesOfTheTextAreBoundaries(t *testing.T) {
	t.Parallel()

	mq5RunHitCases(t, []mq5HitCase{
		{trigger: "curate", text: "curate", hit: true},
	})
}

// Test792a7zb1g_RunQuery_ALaterOccurrenceCanMatch verifies 792a7zb1g (A later occurrence can match): for
// triggers ["curate"], the message "accurate, curate now" is a trigger hit though its first occurrence is inside
// a longer word.
func Test792a7zb1g_RunQuery_ALaterOccurrenceCanMatch(t *testing.T) {
	t.Parallel()

	mq5RunHitCases(t, []mq5HitCase{
		{trigger: "curate", text: "accurate, curate now", hit: true},
	})
}

// Test792a7zb1h_RunQuery_UnicodeLettersAndDigitsAreNotBoundaries verifies 792a7zb1h (Unicode letters and digits
// are not boundaries): for triggers ["curate"], the messages "écurate", "curateé" and "curate2 now" are not
// trigger hits.
func Test792a7zb1h_RunQuery_UnicodeLettersAndDigitsAreNotBoundaries(t *testing.T) {
	t.Parallel()

	mq5RunHitCases(t, []mq5HitCase{
		{trigger: "curate", text: "écurate", hit: false},
		{trigger: "curate", text: "curateé", hit: false},
		{trigger: "curate", text: "curate2 now", hit: false},
	})
}

// Test792a7zb1i_RunQuery_HyphenAndUnderscoreAreBoundaries verifies 792a7zb1i (Hyphen and underscore are
// boundaries): for triggers ["curate"], the messages "re-curate-now" and "do_curate_it" are trigger hits.
func Test792a7zb1i_RunQuery_HyphenAndUnderscoreAreBoundaries(t *testing.T) {
	t.Parallel()

	mq5RunHitCases(t, []mq5HitCase{
		{trigger: "curate", text: "re-curate-now", hit: true},
		{trigger: "curate", text: "do_curate_it", hit: true},
	})
}

// Test792a7zc1a_RunQuery_EveryPayloadNamesItsModel verifies 792a7zc1a (Every payload names its model): over an
// empty vault and over a vault with a note, engram query with a phrase prints the model_id of the embedder, which
// for the bundled model is minilm-l6-v2@384; an embedder of another model is named in turn.
func Test792a7zc1a_RunQuery_EveryPayloadNamesItsModel(t *testing.T) {
	t.Parallel()

	const (
		phrase    = "writing Go tests"
		otherID   = "other-model@8"
		bundledID = "minilm-l6-v2@384"
	)

	g := NewWithT(t)
	g.Expect(embed.BundledModelID).To(Equal(bundledID), "the bundled model is minilm-l6-v2@384")

	cases := map[string]struct {
		modelID   string
		withNotes bool
	}{
		"an empty vault, bundled model":      {modelID: embed.BundledModelID, withNotes: false},
		"a vault with a note, bundled model": {modelID: embed.BundledModelID, withNotes: true},
		"a vault with a note, another model": {modelID: otherID, withNotes: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a vault (whose notes are embedded by the active model).
			world := newMq1World(mq1SmallDims, phrase)

			if tc.withNotes {
				world.plant(mq1Note{
					name: "1.go-tests", content: mq5Content(mq5Note{kind: "fact"}), modelID: tc.modelID,
					situation: world.cosines(map[int]float64{0: mq5Top}), body: world.cosines(map[int]float64{0: mq5Top}),
				})
			}

			// When the user runs engram query with a phrase.
			outcome := mq5Run(t, world, cli.QueryArgs{Phrases: []string{phrase}}, mq5Hooks{modelID: tc.modelID})

			// Then the payload's model_id is that of the embedder.
			g.Expect(outcome.payload.ModelID).To(Equal(tc.modelID))
			g.Expect(outcome.raw).To(HaveKeyWithValue("model_id", tc.modelID))
		})
	}
}

// Test792a7zd1a_RunQuery_APendingRefitIsReported verifies 792a7zd1a (A pending refit is reported): over a vault
// whose vocab.centroids.json has refit_pending true, engram query with a phrase prints refit_pending: true.
func Test792a7zd1a_RunQuery_APendingRefitIsReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const phrase = "writing Go tests"

	// Given a vault whose vocab.centroids.json has refit_pending true.
	world := mq5RefitWorld(t, phrase, mq5Centroids{pending: true})

	// When the user runs engram query with a phrase.
	outcome := mq5Run(t, world, cli.QueryArgs{Phrases: []string{phrase}}, mq5Hooks{})

	// Then the payload has refit_pending: true.
	g.Expect(outcome.raw).To(HaveKeyWithValue("refit_pending", true))
	g.Expect(outcome.stdout).To(ContainSubstring("refit_pending: true\n"))
}

// Test792a7zd1b_RunQuery_OtherwiseTheRefitFieldIsAbsent verifies 792a7zd1b (Otherwise the field is absent): over
// a vault whose vocab.centroids.json records no pending refit, or has no such file, or holds a malformed
// document that still sets the flag, engram query with a phrase prints no refit_pending.
func Test792a7zd1b_RunQuery_OtherwiseTheRefitFieldIsAbsent(t *testing.T) {
	t.Parallel()

	const phrase = "writing Go tests"

	cases := map[string]mq5Centroids{
		"no pending refit": {pending: false},
		"no such file":     {absent: true},
		"malformed file":   {raw: `{"refit_pending": true, "dims": "x"}`},
	}

	for name, centroids := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a vault whose vocab.centroids.json has no pending refit, or no such file.
			world := mq5RefitWorld(t, phrase, centroids)

			// When the user runs engram query with a phrase.
			outcome := mq5Run(t, world, cli.QueryArgs{Phrases: []string{phrase}}, mq5Hooks{})

			// Then the payload has no refit_pending.
			g.Expect(outcome.raw).NotTo(HaveKey("refit_pending"))
			g.Expect(outcome.stdout).NotTo(ContainSubstring("refit_pending"))
		})
	}
}

// Test792a7ze1a_RunQuery_TimingsAreReportedOnRequest verifies 792a7ze1a (Timings are reported on request): over a
// vault with notes, engram query --timings with a phrase adds a timings block with exactly the five keys scan_ms,
// embed_ms, cluster_ms, nominate_ms and render_ms, each holding the milliseconds of its own stage (a scripted
// clock reads 0, 500, 1500, 1600, 1700 and 1720 ms at the start and after each of the five stages).
func Test792a7ze1a_RunQuery_TimingsAreReportedOnRequest(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const phrase = "writing Go tests"

	// Given a vault with notes.
	world := mq5TimingsWorld(phrase)

	// When the user runs engram query --timings with a phrase.
	outcome := mq5Run(t, world, cli.QueryArgs{Phrases: []string{phrase}, Timings: true},
		mq5Hooks{now: mq5ScriptedClock(mq5StageOffsets)})

	// Then the payload has a timings block with five keys, one per stage.
	g.Expect(outcome.raw).To(HaveKey("timings"))

	timings, isMap := outcome.raw["timings"].(map[string]any)
	g.Expect(isMap).To(BeTrue(), "timings is a block of keys")
	g.Expect(timings).To(HaveLen(len(mq5StageOffsets) - 1))
	g.Expect(timings).To(HaveKeyWithValue("scan_ms", 500))
	g.Expect(timings).To(HaveKeyWithValue("embed_ms", 1000))
	g.Expect(timings).To(HaveKeyWithValue("cluster_ms", 100))
	g.Expect(timings).To(HaveKeyWithValue("nominate_ms", 100))
	g.Expect(timings).To(HaveKeyWithValue("render_ms", 20))
}

// Test792a7ze1b_RunQuery_TimingsAreAbsentByDefault verifies 792a7ze1b (Timings are absent by default): over the
// same vault, engram query with a phrase and no --timings has no timings block, and with --timings the rest of
// the payload is unchanged.
func Test792a7ze1b_RunQuery_TimingsAreAbsentByDefault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const phrase = "writing Go tests"

	// Given the same vault.
	world := mq5TimingsWorld(phrase)

	// When the user runs engram query with a phrase and no --timings.
	plain := mq5Run(t, world, cli.QueryArgs{Phrases: []string{phrase}},
		mq5Hooks{now: mq5ScriptedClock(mq5StageOffsets)})

	// Then the payload has no timings block,
	g.Expect(plain.raw).NotTo(HaveKey("timings"))
	g.Expect(plain.stdout).NotTo(ContainSubstring("timings"))

	// and the flag changes nothing else in it.
	timed := mq5Run(t, world, cli.QueryArgs{Phrases: []string{phrase}, Timings: true},
		mq5Hooks{now: mq5ScriptedClock(mq5StageOffsets)})
	g.Expect(timed.raw).To(HaveKey("timings"))

	delete(timed.raw, "timings")
	g.Expect(timed.raw).To(Equal(plain.raw))
}

// unexported constants.
const (
	mq5Low = 0.3
	mq5Mid = 0.7
	mq5Top = 0.9
)

// unexported variables.
var (
	// mq5StageOffsets are the offsets from mq1Now of the six clock reads of a timed query: the start and the end
	// of the scan, embed, cluster, nominate and render stages.
	mq5StageOffsets = []time.Duration{
		0, 500 * time.Millisecond, 1500 * time.Millisecond, 1600 * time.Millisecond,
		1700 * time.Millisecond, 1720 * time.Millisecond,
	}
)

// mq5Centroids describes the vocab.centroids.json of mq5RefitWorld: absent, raw text, or a document.
type mq5Centroids struct {
	absent  bool
	raw     string
	pending bool
}

// mq5Embedder is the phrase-keyed embedder of mq1World with a model id of its own.
type mq5Embedder struct {
	mq1PhraseEmbedder

	modelID string
}

func (e mq5Embedder) ModelID() string { return e.modelID }

// mq5HitCase is one message and whether it hits a runbook carrying the trigger.
type mq5HitCase struct {
	trigger string
	text    string
	hit     bool
}

// mq5Hooks overrides the model of the embedder and the clock of mq5Run.
type mq5Hooks struct {
	modelID string
	now     func() time.Time
}

// mq5Note describes the file text of a note for mq5Content.
type mq5Note struct {
	kind     string
	triggers []string
	project  string
}

// mq5Outcome is what one successful query printed.
type mq5Outcome struct {
	stdout  string
	payload mq1Payload
	raw     map[string]any
}

// mq5Content returns the file text of a note of the given kind, with the triggers and project of the description.
func mq5Content(note mq5Note) string {
	var text strings.Builder

	text.WriteString("---\ntype: " + note.kind + "\nsituation: a situation\n")

	if note.project != "" {
		text.WriteString("project: " + note.project + "\n")
	}

	if len(note.triggers) > 0 {
		text.WriteString("triggers:\n")

		for _, trigger := range note.triggers {
			text.WriteString("    - " + fmt.Sprintf("%q", trigger) + "\n")
		}
	}

	text.WriteString("---\n\nBody of the " + note.kind + ".\n")

	return text.String()
}

// mq5Paths returns the paths of the items of a payload, in item order.
func mq5Paths(payload mq1Payload) []string {
	paths := make([]string, 0, len(payload.Items))

	for _, item := range payload.Items {
		paths = append(paths, item.Path)
	}

	return paths
}

// mq5Plant plants a note with a current sidecar whose vectors have the given cosines to the phrase axes.
func mq5Plant(world *mq1World, name, content string, cosines map[int]float64) {
	vec := world.cosines(cosines)
	world.plant(mq1Note{name: name, content: content, situation: vec, body: vec})
}

// mq5RefitWorld returns a world with one note and the vocab.centroids.json of the description.
func mq5RefitWorld(t *testing.T, phrase string, centroids mq5Centroids) *mq1World {
	t.Helper()

	world := newMq1World(mq1SmallDims, phrase)
	mq5Plant(world, "1.go-tests", mq5Content(mq5Note{kind: "fact"}), map[int]float64{0: mq5Top})

	path := filepath.Join(mq1Vault, "vocab.centroids.json")

	switch {
	case centroids.absent:
		return world
	case centroids.raw != "":
		world.files.files[path] = []byte(centroids.raw)

		return world
	}

	data, err := json.Marshal(cli.ExportVocabCentroidsDoc{SchemaVersion: 1, RefitPending: centroids.pending})
	if err != nil {
		t.Fatalf("encoding the centroids document: %v", err)
	}

	world.files.files[path] = data

	return world
}

// mq5Run runs cli.RunQuery over the world with the given arguments (the vault and the chunk directory are the
// world's) and decodes the payload it printed.
func mq5Run(t *testing.T, world *mq1World, args cli.QueryArgs, hooks mq5Hooks) mq5Outcome {
	t.Helper()

	modelID := hooks.modelID
	if modelID == "" {
		modelID = mq1ModelID
	}

	clock := hooks.now
	if clock == nil {
		clock = world.clock
	}

	phraseEmbedder := mq1PhraseEmbedder{dims: world.dims, vectors: world.vectors}

	deps := cli.QueryDeps{
		Scan:     world.files.Scan,
		Read:     world.files.Read,
		Embedder: mq5Embedder{mq1PhraseEmbedder: phraseEmbedder, modelID: modelID},
		Now:      clock,
	}

	args.VaultPath = mq1Vault
	args.ChunksDir = mq1ChunksDir

	var out bytes.Buffer

	err := cli.RunQuery(t.Context(), args, deps, &out)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	outcome := mq5Outcome{stdout: out.String()}

	err = yaml.Unmarshal(out.Bytes(), &outcome.payload)
	if err != nil {
		t.Fatalf("decoding the payload: %v", err)
	}

	err = yaml.Unmarshal(out.Bytes(), &outcome.raw)
	if err != nil {
		t.Fatalf("decoding the payload keys: %v", err)
	}

	return outcome
}

// mq5RunHitCases runs each case in its own world: a runbook carrying the trigger, a text-only query for the message;
// the runbook is the one item, with provenances [trigger], exactly when the case is a hit.
func mq5RunHitCases(t *testing.T, cases []mq5HitCase) {
	t.Helper()

	const runbookName = "1.release-runbook"

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q in %q", tc.trigger, tc.text), func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a runbook with the trigger.
			world := newMq1World(mq1SmallDims)
			mq5Plant(world, runbookName, mq5Content(mq5Note{kind: "runbook", triggers: []string{tc.trigger}}),
				map[int]float64{})

			// When the user runs engram query --text with the message.
			outcome := mq5Run(t, world, cli.QueryArgs{Text: tc.text}, mq5Hooks{})

			// Then the runbook is a trigger hit exactly when the case says so.
			if !tc.hit {
				g.Expect(outcome.payload.Items).To(BeEmpty())

				return
			}

			g.Expect(mq5Paths(outcome.payload)).To(Equal([]string{runbookName + ".md"}))

			if len(outcome.payload.Items) == 1 {
				g.Expect(outcome.payload.Items[0].Provenances).To(Equal([]string{"trigger"}))
			}
		})
	}
}

// mq5ScriptedClock returns a clock that reads mq1Now plus each offset in turn, repeating the last offset after the
// last read.
func mq5ScriptedClock(offsets []time.Duration) func() time.Time {
	reads := 0

	return func() time.Time {
		offset := offsets[min(reads, len(offsets)-1)]
		reads++

		return mq1Now.Add(offset)
	}
}

// mq5TimingsWorld returns a world with two notes matching the phrase.
func mq5TimingsWorld(phrase string) *mq1World {
	world := newMq1World(mq1SmallDims, phrase)

	for index, cosine := range []float64{mq5Top, mq5Mid} {
		mq5Plant(world, fmt.Sprintf("%d.note", index+1), mq5Content(mq5Note{kind: "fact"}),
			map[int]float64{0: cosine})
	}

	return world
}
