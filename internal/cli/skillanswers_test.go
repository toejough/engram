package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestAnswerSkillOffers_AcceptAll_AcceptsEveryScopeOffer covers the grouped
// prompt's "accept all": every register/refresh offer of that scope is
// accepted with one answer.
func TestAnswerSkillOffers_AcceptAll_AcceptsEveryScopeOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offers := superpowersOffers("brainstorming", "writing-plans", "writing-skills")
	rec := newAnswerRecorder()

	var stdout bytes.Buffer

	err := cli.AnswerSkillOffers(rec.answering(offers, emptyAnswers(t), "a\n"), &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rec.accepted).To(Equal(offerKeys(offers)))
	g.Expect(rec.declined).To(BeEmpty())
	g.Expect(stdout.String()).To(ContainSubstring(
		"@plugin:superpowers (3): [a]ccept all / [d]ecline all / [r]eview each / [s]kip for now "))
	g.Expect(stdout.String()).To(ContainSubstring("  register superpowers:writing-plans\n"))
	g.Expect(stdout.String()).NotTo(ContainSubstring("[y/N]"))
}

// TestAnswerSkillOffers_DeclineAll_DeclinesEveryScopeOffer covers "decline
// all": each key is declined (recorded per key, design D6 alternative c).
func TestAnswerSkillOffers_DeclineAll_DeclinesEveryScopeOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offers := superpowersOffers("brainstorming", "writing-plans")
	rec := newAnswerRecorder()

	var stdout bytes.Buffer

	err := cli.AnswerSkillOffers(rec.answering(offers, emptyAnswers(t), "d\n"), &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rec.declined).To(Equal(offerKeys(offers)))
	g.Expect(rec.accepted).To(BeEmpty())
}

// TestAnswerSkillOffers_DryRunPreview_ListsOffersUnderScopeHeaders covers
// "`--dry-run` SHALL list each offer with its kind, key, and source path
// under scope headers"; a removal with no recorded source names its note.
func TestAnswerSkillOffers_DryRunPreview_ListsOffersUnderScopeHeaders(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	comparison := cli.SkillOfferComparison{Offers: []cli.SkillOffer{
		{Kind: cli.SkillOfferRegister, Key: "c4", ScopeID: "claude-user", SourcePath: "/h/.claude/skills/c4/SKILL.md"},
		{Kind: cli.SkillOfferRemove, Key: "old", ScopeID: "claude-user", Basename: "7.2026-01-01.skill-old"},
		{
			Kind: cli.SkillOfferRefresh, Key: "superpowers:tdd", ScopeID: "plugin:superpowers",
			SourcePath: "~/.claude/plugins/cache/o/superpowers/6/skills/tdd/SKILL.md",
		},
	}}

	var stdout bytes.Buffer

	err := cli.PreviewSkillOffers(&stdout, comparison)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal("" +
		"@claude-user (2)\n" +
		"  would offer: register c4 (/h/.claude/skills/c4/SKILL.md)\n" +
		"  would offer: remove old (note 7.2026-01-01.skill-old)\n" +
		"@plugin:superpowers (1)\n" +
		"  would offer: refresh superpowers:tdd (~/.claude/plugins/cache/o/superpowers/6/skills/tdd/SKILL.md)\n"))
}

// TestAnswerSkillOffers_DryRunPreview_ReportsConflictsLast covers the dry
// run still reporting conflicts after the offers, with the failure status.
func TestAnswerSkillOffers_DryRunPreview_ReportsConflictsLast(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	comparison := cli.SkillOfferComparison{
		Offers:    []cli.SkillOffer{{Kind: cli.SkillOfferRegister, Key: "c4", ScopeID: "claude-user", SourcePath: "/c4"}},
		Conflicts: []string{"engram: skill key conflict: x at /a and /b"},
	}

	var stdout bytes.Buffer

	err := cli.PreviewSkillOffers(&stdout, comparison)

	g.Expect(err).To(MatchError(cli.ErrSkillOfferConflictForTest))
	g.Expect(stdout.String()).To(HaveSuffix("engram: skill key conflict: x at /a and /b\n"))
}

// TestAnswerSkillOffers_EOFAtGroupPrompt_RecordsNothing covers "end-of-input
// at that prompt SHALL record nothing".
func TestAnswerSkillOffers_EOFAtGroupPrompt_RecordsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offers := superpowersOffers("brainstorming", "writing-plans")
	rec := newAnswerRecorder()

	var stdout bytes.Buffer

	err := cli.AnswerSkillOffers(rec.answering(offers, emptyAnswers(t), ""), &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rec.accepted).To(BeEmpty())
	g.Expect(rec.declined).To(BeEmpty())
}

// TestAnswerSkillOffers_ExplicitAnswers_SpecScenarios covers the spec's
// answering scenarios non-interactively over fixture comparisons.
func TestAnswerSkillOffers_ExplicitAnswers_SpecScenarios(t *testing.T) {
	t.Parallel()

	fifteen := superpowersOffers("a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "b1", "b2", "b3", "b4",
		"b5", "b6")
	mixed := superpowersOffers("brainstorming", "tdd", "writing-plans", "writing-skills")
	userOffers := []cli.SkillOffer{
		{Kind: cli.SkillOfferRegister, Key: "c4", ScopeID: "claude-user", Hash: "h-c4"},
		{Kind: cli.SkillOfferRegister, Key: "dev", ScopeID: "claude-user", Hash: "h-dev"},
	}
	withRemoval := []cli.SkillOffer{
		{Kind: cli.SkillOfferRegister, Key: "c4", ScopeID: "claude-user", Hash: "h-c4"},
		{Kind: cli.SkillOfferRemove, Key: "old", ScopeID: "claude-user", Hash: "h-old"},
	}

	cases := []struct {
		name         string
		offers       []cli.SkillOffer
		accept       []string
		decline      []string
		wantAccepted []string
		wantDeclined []string
	}{
		{
			name: "decline a whole plugin", offers: fifteen, decline: []string{"@plugin:superpowers"},
			wantDeclined: offerKeys(fifteen),
		},
		{
			name: "longest pattern wins", offers: mixed,
			decline: []string{"superpowers:*"}, accept: []string{"superpowers:writing-*"},
			wantAccepted: []string{"superpowers:writing-plans", "superpowers:writing-skills"},
			wantDeclined: []string{"superpowers:brainstorming", "superpowers:tdd"},
		},
		{
			name: "exact key beats pattern", offers: mixed,
			decline: []string{"superpowers:*"}, accept: []string{"superpowers:brainstorming"},
			wantAccepted: []string{"superpowers:brainstorming"},
			wantDeclined: []string{"superpowers:tdd", "superpowers:writing-plans", "superpowers:writing-skills"},
		},
		{
			name: "pattern beats selector", offers: mixed,
			decline: []string{"@plugin:superpowers"}, accept: []string{"superpowers:writing-*"},
			wantAccepted: []string{"superpowers:writing-plans", "superpowers:writing-skills"},
			wantDeclined: []string{"superpowers:brainstorming", "superpowers:tdd"},
		},
		{
			name: "bare user scope is nameable", offers: userOffers, decline: []string{"@claude-user"},
			wantDeclined: []string{"c4", "dev"},
		},
		{
			name: "removals are never bulk-accepted", offers: withRemoval, accept: []string{"*"},
			wantAccepted: []string{"c4"},
		},
		{
			name: "a removal answered by its exact key", offers: withRemoval, accept: []string{"*", "old"},
			wantAccepted: []string{"c4", "old"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			answers, parseErr := cli.ParseSkillAnswers(tc.accept, tc.decline)
			g.Expect(parseErr).NotTo(HaveOccurred())

			rec := newAnswerRecorder()
			input := rec.answering(tc.offers, answers, "")
			input.Interactive = false

			var stdout bytes.Buffer

			err := cli.AnswerSkillOffers(input, &stdout)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(rec.accepted).To(ConsistOf(stringsOrEmpty(tc.wantAccepted)))
			g.Expect(rec.declined).To(ConsistOf(stringsOrEmpty(tc.wantDeclined)))
		})
	}
}

// TestAnswerSkillOffers_NonInteractiveSummary_NamesScopesAndCounts covers
// the one-line non-interactive summary (design D6): the total, each scope's
// `@<scope-id> <count>` in scope order, and the answering command.
func TestAnswerSkillOffers_NonInteractiveSummary_NamesScopesAndCounts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offers := append(superpowersOffers("a", "b"),
		cli.SkillOffer{Kind: cli.SkillOfferRegister, Key: "anthropic-skills:pdf", ScopeID: "synced"},
		cli.SkillOffer{
			Kind: cli.SkillOfferRegister, Key: "project:github.com/toejough/engram:c4",
			ScopeID: "project:github.com/toejough/engram",
		},
	)
	sortOffers(offers)

	rec := newAnswerRecorder()
	input := rec.answering(offers, emptyAnswers(t), "")
	input.Interactive = false

	var stdout bytes.Buffer

	err := cli.AnswerSkillOffers(input, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(Equal("engram: 4 skill runbook offers awaiting an answer: " +
		"@plugin:superpowers 2, @project:github.com/toejough/engram 1, @synced 1 — run " +
		"`engram register-skills` in a terminal, or `engram register-skills --accept <key|prefix*|@scope>` / " +
		"`--decline <key|prefix*|@scope>`\n"))
	g.Expect(rec.accepted).To(BeEmpty())
	g.Expect(rec.declined).To(BeEmpty())
}

// TestAnswerSkillOffers_NonInteractiveSummary_SingularOffer covers the
// summary's singular form.
func TestAnswerSkillOffers_NonInteractiveSummary_SingularOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rec := newAnswerRecorder()
	input := rec.answering(superpowersOffers("a"), emptyAnswers(t), "")
	input.Interactive = false

	var stdout bytes.Buffer

	g.Expect(cli.AnswerSkillOffers(input, &stdout)).To(Succeed())
	g.Expect(stdout.String()).To(HavePrefix(
		"engram: 1 skill runbook offer awaiting an answer: @plugin:superpowers 1 — run "))
}

// TestAnswerSkillOffers_PerOfferPrompt_EOFRecordsNothingNoDeclines covers
// ruling R29: end of input at a per-offer y/N prompt is no answer — for a
// single offer, under "review each", and for a removal — so nothing is
// accepted or declined, while an explicit "n" still declines.
func TestAnswerSkillOffers_PerOfferPrompt_EOFRecordsNothingNoDeclines(t *testing.T) {
	t.Parallel()

	removal := []cli.SkillOffer{{Kind: cli.SkillOfferRemove, Key: "old", ScopeID: "claude-user", Hash: "h-old"}}

	cases := []struct {
		name         string
		offers       []cli.SkillOffer
		stdin        string
		wantDeclined []string
	}{
		{name: "single offer EOF", offers: superpowersOffers("a"), stdin: ""},
		{name: "review each EOF", offers: superpowersOffers("a", "b"), stdin: "r\n"},
		{name: "review each EOF after one no", offers: superpowersOffers("a", "b"), stdin: "r\nn\n",
			wantDeclined: []string{"superpowers:a"}},
		{name: "removal EOF", offers: removal, stdin: ""},
		{name: "single offer no", offers: superpowersOffers("a"), stdin: "n\n", wantDeclined: []string{"superpowers:a"}},
		{name: "review each no", offers: superpowersOffers("a", "b"), stdin: "r\n" + strings.Repeat("n\n", 2),
			wantDeclined: []string{"superpowers:a", "superpowers:b"}},
		{name: "removal no", offers: removal, stdin: "n\n", wantDeclined: []string{"old"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rec := newAnswerRecorder()

			var stdout bytes.Buffer

			g.Expect(cli.AnswerSkillOffers(rec.answering(tc.offers, emptyAnswers(t), tc.stdin), &stdout)).To(Succeed())
			g.Expect(rec.accepted).To(BeEmpty())
			g.Expect(rec.declined).To(ConsistOf(stringsOrEmpty(tc.wantDeclined)))
		})
	}
}

// TestAnswerSkillOffers_PropagatesActionErrors covers an accept or decline
// action failing: the run stops with that error.
func TestAnswerSkillOffers_PropagatesActionErrors(t *testing.T) {
	t.Parallel()

	for _, stdin := range []string{"a\n", "d\n", "r\ny\n"} {
		t.Run(strings.ReplaceAll(strings.TrimSpace(stdin), "\n", "-"), func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rec := newAnswerRecorder()
			input := rec.answering(superpowersOffers("a", "b"), emptyAnswers(t), stdin)
			input.Accept = func(cli.SkillOffer) error { return errAnswerActionForTest }
			input.Decline = input.Accept

			var stdout bytes.Buffer

			g.Expect(cli.AnswerSkillOffers(input, &stdout)).To(MatchError(errAnswerActionForTest))
		})
	}
}

// TestAnswerSkillOffers_RemovalsPromptIndividually covers "Removal offers
// SHALL be answered only by ... an individual prompt, never by accept-all":
// the scope's two register offers get the group question, and each removal
// its own y/N prompt.
func TestAnswerSkillOffers_RemovalsPromptIndividually(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offers := []cli.SkillOffer{
		{Kind: cli.SkillOfferRegister, Key: "c4", ScopeID: "claude-user"},
		{Kind: cli.SkillOfferRegister, Key: "dev", ScopeID: "claude-user"},
		{Kind: cli.SkillOfferRemove, Key: "gone", ScopeID: "claude-user"},
		{Kind: cli.SkillOfferRemove, Key: "old", ScopeID: "claude-user"},
	}
	rec := newAnswerRecorder()

	var stdout bytes.Buffer

	err := cli.AnswerSkillOffers(rec.answering(offers, emptyAnswers(t), "a\ny\nn\n"), &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rec.accepted).To(Equal([]string{"c4", "dev", "gone"}))
	g.Expect(rec.declined).To(Equal([]string{"old"}))
	g.Expect(stdout.String()).To(ContainSubstring("@claude-user (2): [a]ccept all"))
	g.Expect(stdout.String()).To(ContainSubstring("Skill `gone` is no longer shipped. Remove its runbook note? [y/N] "))
}

// TestAnswerSkillOffers_ReviewEach_PromptsPerOffer covers "review each":
// each offer then gets its own y/N prompt.
func TestAnswerSkillOffers_ReviewEach_PromptsPerOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offers := superpowersOffers("brainstorming", "writing-plans")
	rec := newAnswerRecorder()

	var stdout bytes.Buffer

	err := cli.AnswerSkillOffers(rec.answering(offers, emptyAnswers(t), "r\ny\nn\n"), &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rec.accepted).To(Equal([]string{"superpowers:brainstorming"}))
	g.Expect(rec.declined).To(Equal([]string{"superpowers:writing-plans"}))
	g.Expect(stdout.String()).To(ContainSubstring(
		"Register skill `superpowers:writing-plans` as a vault runbook? [y/N] "))
}

// TestAnswerSkillOffers_SingleOffer_UsesPerOfferPrompt covers "A single
// offer uses the per-offer y/N prompt", with no group question.
func TestAnswerSkillOffers_SingleOffer_UsesPerOfferPrompt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offers := superpowersOffers("brainstorming")
	rec := newAnswerRecorder()

	var stdout bytes.Buffer

	err := cli.AnswerSkillOffers(rec.answering(offers, emptyAnswers(t), "y\n"), &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rec.accepted).To(Equal([]string{"superpowers:brainstorming"}))
	g.Expect(stdout.String()).To(Equal("Register skill `superpowers:brainstorming` as a vault runbook? [y/N] "))
}

// TestAnswerSkillOffers_Skip_RecordsNothingAndMovesOn covers "Skip leaves
// the scope for next time": a skipped scope records nothing, and the next
// scope is still asked.
func TestAnswerSkillOffers_Skip_RecordsNothingAndMovesOn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offers := append(superpowersOffers("a", "b"),
		cli.SkillOffer{Kind: cli.SkillOfferRegister, Key: "anthropic-skills:pdf", ScopeID: "synced"},
		cli.SkillOffer{Kind: cli.SkillOfferRegister, Key: "anthropic-skills:xlsx", ScopeID: "synced"},
	)
	rec := newAnswerRecorder()

	var stdout bytes.Buffer

	err := cli.AnswerSkillOffers(rec.answering(offers, emptyAnswers(t), "s\nd\n"), &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rec.accepted).To(BeEmpty())
	g.Expect(rec.declined).To(Equal([]string{"anthropic-skills:pdf", "anthropic-skills:xlsx"}))
}

// TestAnswerSkillOffers_UnmatchedToken_PrintsExistingNote covers "A token
// matching no offer → the existing one-line note", for each token form, and
// a pattern or selector that would match only a removal.
func TestAnswerSkillOffers_UnmatchedToken_PrintsExistingNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offers := []cli.SkillOffer{{Kind: cli.SkillOfferRemove, Key: "old", ScopeID: "claude-user"}}
	answers, parseErr := cli.ParseSkillAnswers([]string{"bogus", "ol*"}, []string{"@claude-user", "@plugin:nope"})
	g.Expect(parseErr).NotTo(HaveOccurred())

	rec := newAnswerRecorder()
	input := rec.answering(offers, answers, "")
	input.Interactive = false

	var stdout bytes.Buffer

	g.Expect(cli.AnswerSkillOffers(input, &stdout)).To(Succeed())

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("engram: no pending offer for skill \"bogus\" — ignoring --accept\n"))
	g.Expect(out).To(ContainSubstring("engram: no pending offer for skill \"ol*\" — ignoring --accept\n"))
	g.Expect(out).To(ContainSubstring("engram: no pending offer for skill \"@claude-user\" — ignoring --decline\n"))
	g.Expect(out).To(ContainSubstring("engram: no pending offer for skill \"@plugin:nope\" — ignoring --decline\n"))
	g.Expect(rec.accepted).To(BeEmpty())
	g.Expect(rec.declined).To(BeEmpty())
}

// TestParseSkillAnswers_Property_SameTokenInBothFlagsIsRefused is the
// refusal rule over any two token lists: parsing fails exactly when some
// token appears in both --accept and --decline.
func TestParseSkillAnswers_Property_SameTokenInBothFlagsIsRefused(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		accept := rapid.SliceOfN(genAnswerToken(), 0, 4).Draw(rt, "accept")
		decline := rapid.SliceOfN(genAnswerToken(), 0, 4).Draw(rt, "decline")

		if rapid.Bool().Draw(rt, "forceShared") && len(accept) > 0 {
			decline = append(decline, accept[rapid.IntRange(0, len(accept)-1).Draw(rt, "shared")])
		}

		shared := slices.ContainsFunc(accept, func(token string) bool { return slices.Contains(decline, token) })

		_, err := cli.ParseSkillAnswers(accept, decline)

		if shared {
			g.Expect(err).To(MatchError(cli.ErrAnswerInBothFlagsForTest))
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
	})
}

// TestResolveSkillAnswer_Property_ExactThenLongestPatternThenSelector is the
// precedence rule over any offer and any disjoint accept/decline token
// sets. The oracle is the spec's ranking of each token against the offer:
// an exact key outranks every pattern, a longer matching prefix outranks a
// shorter one, any pattern outranks a selector, and a removal is matched
// only by its exact key. The winning answer must come from the flag holding
// the unique top-ranked matching token, or be none when nothing matches.
func TestResolveSkillAnswer_Property_ExactThenLongestPatternThenSelector(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		offer := cli.SkillOffer{
			Kind:    rapid.SampledFrom(allOfferKinds()).Draw(rt, "kind"),
			Key:     genAnswerKey().Draw(rt, "key"),
			ScopeID: rapid.SampledFrom(answerScopes()).Draw(rt, "scope"),
		}

		tokens := rapid.SliceOfNDistinct(genAnswerToken(), 0, 8, func(token string) string { return token }).
			Draw(rt, "tokens")

		var accept, decline []string

		for index, token := range tokens {
			if rapid.Bool().Draw(rt, fmt.Sprintf("accept%d", index)) {
				accept = append(accept, token)
			} else {
				decline = append(decline, token)
			}
		}

		answers, err := cli.ParseSkillAnswers(accept, decline)
		g.Expect(err).NotTo(HaveOccurred())

		want := cli.SkillAnswerNone
		bestRank := noMatchRank

		for _, token := range tokens {
			rank := specRank(offer, token)
			if rank > bestRank {
				bestRank = rank
				want = cli.SkillAnswerDecline

				if slices.Contains(accept, token) {
					want = cli.SkillAnswerAccept
				}
			}
		}

		g.Expect(answers.Resolve(offer)).To(Equal(want))
	})
}

// unexported constants.
const (
	// exactKeyRank outranks any pattern's prefix length (keys here are short).
	exactKeyRank = 1000
	noMatchRank  = -1
	selectorRank = 0
)

// unexported variables.
var (
	errAnswerActionForTest = errors.New("answer fixture: injected action failure")
)

// answerRecorder records the keys AnswerSkillOffers accepts and declines.
type answerRecorder struct {
	accepted []string
	declined []string
}

// answering builds an interactive SkillOfferAnswering over offers, reading
// stdin, recording each accept and decline.
func (r *answerRecorder) answering(offers []cli.SkillOffer, answers cli.SkillAnswers, stdin string,
) cli.SkillOfferAnswering {
	return cli.SkillOfferAnswering{
		Comparison:  cli.SkillOfferComparison{Offers: offers},
		Answers:     answers,
		Interactive: true,
		Stdin:       strings.NewReader(stdin),
		Accept: func(offer cli.SkillOffer) error {
			r.accepted = append(r.accepted, offer.Key)

			return nil
		},
		Decline: func(offer cli.SkillOffer) error {
			r.declined = append(r.declined, offer.Key)

			return nil
		},
	}
}

func allOfferKinds() []cli.SkillOfferKind {
	return []cli.SkillOfferKind{cli.SkillOfferRegister, cli.SkillOfferRefresh, cli.SkillOfferRemove}
}

func answerScopes() []string {
	return []string{"claude-user", "plugin:a", "project:github.com/x/y"}
}

func emptyAnswers(t *testing.T) cli.SkillAnswers {
	t.Helper()

	answers, err := cli.ParseSkillAnswers(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	return answers
}

// genAnswerKey draws a short key over a tiny alphabet so tokens often match.
func genAnswerKey() *rapid.Generator[string] {
	return rapid.StringMatching(`[ab]{1,2}(:[ab]{1,2}){0,2}`)
}

// genAnswerToken draws an exact key, a `prefix*` pattern (including `*`),
// or an `@<scope-id>` selector.
func genAnswerToken() *rapid.Generator[string] {
	return rapid.OneOf(
		genAnswerKey(),
		rapid.Map(rapid.StringMatching(`[ab:]{0,4}`), func(prefix string) string { return prefix + "*" }),
		rapid.Map(rapid.SampledFrom(answerScopes()), func(scope string) string { return "@" + scope }),
	)
}

func newAnswerRecorder() *answerRecorder {
	return &answerRecorder{}
}

func offerKeys(offers []cli.SkillOffer) []string {
	keys := make([]string, 0, len(offers))
	for _, offer := range offers {
		keys = append(keys, offer.Key)
	}

	return keys
}

func sortOffers(offers []cli.SkillOffer) {
	slices.SortFunc(offers, func(left, right cli.SkillOffer) int {
		if left.ScopeID != right.ScopeID {
			return strings.Compare(left.ScopeID, right.ScopeID)
		}

		return strings.Compare(left.Key, right.Key)
	})
}

// specRank ranks token against offer per the spec's precedence.
func specRank(offer cli.SkillOffer, token string) int {
	switch {
	case token == offer.Key:
		return exactKeyRank
	case offer.Kind == cli.SkillOfferRemove:
		return noMatchRank
	case strings.HasPrefix(token, "@"):
		if token == "@"+offer.ScopeID {
			return selectorRank
		}

		return noMatchRank
	case strings.HasSuffix(token, "*") && strings.HasPrefix(offer.Key, strings.TrimSuffix(token, "*")):
		return 1 + len(token)
	default:
		return noMatchRank
	}
}

// superpowersOffers builds register offers in the plugin:superpowers scope,
// one per name, in key order.
func superpowersOffers(names ...string) []cli.SkillOffer {
	offers := make([]cli.SkillOffer, 0, len(names))
	for _, name := range names {
		offers = append(offers, cli.SkillOffer{
			Kind: cli.SkillOfferRegister, Key: "superpowers:" + name, ScopeID: "plugin:superpowers", Hash: "h-" + name,
		})
	}

	return offers
}
