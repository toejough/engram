package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Exported constants.
const (
	// SkillAnswerAccept, SkillAnswerDecline and SkillAnswerNone are
	// SkillAnswers.Resolve's results: the winning flag, or no answer.
	SkillAnswerAccept  SkillAnswer = "accept"
	SkillAnswerDecline SkillAnswer = "decline"
	SkillAnswerNone    SkillAnswer = ""
)

// SkillAnswer is the explicit answer an offer received from --accept or
// --decline, or SkillAnswerNone.
type SkillAnswer string

// SkillAnswers holds the parsed --accept and --decline tokens (design D6):
// each an exact key, a key-prefix pattern ending in `*`, or a scope
// selector `@<scope-id>`. Build it with ParseSkillAnswers.
type SkillAnswers struct {
	accept  []string
	decline []string
}

// Resolve returns offer's explicit answer (design D6 precedence): an exact
// key wins; otherwise the longest matching `*` pattern (by prefix length);
// otherwise a matching `@<scope-id>` selector. A removal offer is matched
// only by its exact key, never by a pattern or selector.
func (a SkillAnswers) Resolve(offer SkillOffer) SkillAnswer {
	answer := SkillAnswerNone
	best := answerRankNone

	for _, flag := range []struct {
		tokens []string
		answer SkillAnswer
	}{{a.accept, SkillAnswerAccept}, {a.decline, SkillAnswerDecline}} {
		for _, token := range flag.tokens {
			if rank := answerTokenRank(offer, token); rank > best {
				best, answer = rank, flag.answer
			}
		}
	}

	return answer
}

// reportUnmatched prints skillRegistrationNoOfferFormat for each token that
// matches no offer.
func (a SkillAnswers) reportUnmatched(stdout io.Writer, offers []SkillOffer) {
	for _, flag := range []struct {
		tokens []string
		name   string
	}{{a.accept, string(SkillAnswerAccept)}, {a.decline, string(SkillAnswerDecline)}} {
		for _, token := range flag.tokens {
			if !tokenMatchesAnyOffer(token, offers) {
				_, _ = fmt.Fprintf(stdout, skillRegistrationNoOfferFormat, token, flag.name)
			}
		}
	}
}

// SkillOfferAnswering is AnswerSkillOffers' input: the comparison, the
// explicit answers, whether to prompt (and the prompt input), and the
// actions an accepted or declined offer triggers.
type SkillOfferAnswering struct {
	Comparison  SkillOfferComparison
	Answers     SkillAnswers
	Interactive bool
	Stdin       io.Reader
	// Accept carries an accepted offer out (register, refresh or remove).
	Accept func(SkillOffer) error
	// Decline records an offer's hash as declined under its key.
	Decline func(SkillOffer) error
}

// AnswerSkillOffers answers each scope's offers, scope by scope (design D6):
//   - an offer with an explicit answer (SkillAnswers.Resolve) is accepted or
//     declined without prompting;
//   - when interactive, a scope with more than one remaining register or
//     refresh offer is asked once — accept all, decline all, review each, or
//     skip (skip, end of input or any other answer records nothing); a
//     single such offer, and every removal, gets its own y/N prompt, where
//     end of input likewise records nothing;
//   - otherwise the offer is left outstanding.
//
// It then notes each --accept/--decline token that matched no offer, prints
// the one-line summary of outstanding offers (their total, `@<scope-id>
// <count>` per scope and the answering command), and finally reports the
// comparison's warnings and conflicts (ReportSkillOfferProblems).
func AnswerSkillOffers(input SkillOfferAnswering, stdout io.Writer) error {
	answerer := skillOfferAnswerer{input: input, stdout: stdout}
	if input.Interactive {
		answerer.scanner = bufio.NewScanner(input.Stdin)
	}

	for _, scope := range groupOffersByScope(input.Comparison.Offers) {
		scopeErr := answerer.answerScope(scope)
		if scopeErr != nil {
			return scopeErr
		}
	}

	input.Answers.reportUnmatched(stdout, input.Comparison.Offers)
	printOutstandingSummary(stdout, answerer.outstanding)

	return ReportSkillOfferProblems(stdout, input.Comparison)
}

// ParseSkillAnswers builds SkillAnswers from the --accept and --decline
// tokens, refusing (errAnswerInBothFlags) any token — key, pattern or
// selector — named in both, before anything is acted on (design D6).
func ParseSkillAnswers(accept, decline []string) (SkillAnswers, error) {
	declineSet := toStringSet(decline)

	for _, token := range accept {
		if declineSet[token] {
			return SkillAnswers{}, fmt.Errorf("%w: %q", errAnswerInBothFlags, token)
		}
	}

	return SkillAnswers{accept: accept, decline: decline}, nil
}

// PreviewSkillOffers prints `--dry-run`'s listing (design D6): each scope's
// header `@<scope-id> (<count>)`, then one indented `would offer: <kind>
// <key> (<source>)` line per offer — a removal with no recorded source
// names its note instead — and finally the comparison's warnings and
// conflicts (ReportSkillOfferProblems).
func PreviewSkillOffers(stdout io.Writer, comparison SkillOfferComparison) error {
	for _, scope := range groupOffersByScope(comparison.Offers) {
		_, _ = fmt.Fprintf(stdout, skillScopeHeaderFormat, scope.id, len(scope.offers))

		for _, offer := range scope.offers {
			_, _ = fmt.Fprintf(stdout, skillRegistrationDryRunOfferFormat, offer.Kind, offer.Key, offerSourceLabel(offer))
		}
	}

	return ReportSkillOfferProblems(stdout, comparison)
}

// unexported constants.
const (
	// answerRankNone .. answerRankPatternBase rank an answer token against
	// an offer (answerTokenRank): no match, a selector, then patterns by
	// prefix length above it, and an exact key above every pattern.
	answerRankExact       = int(^uint(0) >> 1)
	answerRankNone        = -1
	answerRankPatternBase = 1
	answerRankSelector    = 0
	// skillAnswerPatternSuffix ends a key-prefix pattern (design D6).
	skillAnswerPatternSuffix = "*"
	// skillAnswerSelectorPrefix starts a scope selector (design D6).
	skillAnswerSelectorPrefix = "@"
	// skillGroupPromptFormat asks once for a scope's register/refresh offers
	// (design D6), after listing them.
	skillGroupPromptFormat = "@%s (%d): [a]ccept all / [d]ecline all / [r]eview each / [s]kip for now "
	// skillGroupPromptOfferFormat lists one offer above the group question.
	skillGroupPromptOfferFormat = "  %s %s%s\n"
	// skillOutstandingScopeFormat is one `@<scope-id> <count>` summary label.
	skillOutstandingScopeFormat = "@%s %d"
	// skillScopeHeaderFormat heads a scope's --dry-run listing (design D6's
	// label: the answerable selector with its count).
	skillScopeHeaderFormat = "@%s (%d)\n"
	// skillSourceNoteFormat names a removal's note when it has no recorded
	// skill_source (e.g. a hand-written note carrying only skill_key).
	skillSourceNoteFormat = "note %s"
)

// groupChoice is an answer to the grouped scope question.
type groupChoice int

// groupChoice values.
const (
	groupSkip groupChoice = iota
	groupAcceptAll
	groupDeclineAll
	groupReviewEach
)

// unexported variables.
var (
	// errAnswerInBothFlags refuses a key, pattern or selector named in both
	// --accept and --decline (design D6).
	errAnswerInBothFlags = errors.New("register-skills: answer named in both --accept and --decline")
)

// offerScope is one scope's offers, in offer order.
type offerScope struct {
	id     string
	offers []SkillOffer
}

// outstandingScope counts one scope's offers left without an answer.
type outstandingScope struct {
	id    string
	count int
}

// skillOfferAnswerer carries AnswerSkillOffers' state across scopes.
type skillOfferAnswerer struct {
	input       SkillOfferAnswering
	stdout      io.Writer
	scanner     *bufio.Scanner
	outstanding []outstandingScope
}

// act carries out answer for offer.
func (a *skillOfferAnswerer) act(offer SkillOffer, accept bool) error {
	if accept {
		return a.input.Accept(offer)
	}

	return a.input.Decline(offer)
}

// answerScope applies explicit answers to scope's offers, then prompts for
// (or records as outstanding) the rest.
func (a *skillOfferAnswerer) answerScope(scope offerScope) error {
	remaining := make([]SkillOffer, 0, len(scope.offers))

	for _, offer := range scope.offers {
		answer := a.input.Answers.Resolve(offer)
		if answer == SkillAnswerNone {
			remaining = append(remaining, offer)

			continue
		}

		actErr := a.act(offer, answer == SkillAnswerAccept)
		if actErr != nil {
			return actErr
		}
	}

	if len(remaining) == 0 {
		return nil
	}

	if !a.input.Interactive {
		a.outstanding = append(a.outstanding, outstandingScope{id: scope.id, count: len(remaining)})

		return nil
	}

	return a.promptScope(scope.id, remaining)
}

// askGroup lists bulk and asks the grouped question once.
// An unrecognised answer is treated as skip, as is end of input.
func (a *skillOfferAnswerer) askGroup(scopeID string, bulk []SkillOffer) groupChoice {
	for _, offer := range bulk {
		_, _ = fmt.Fprintf(a.stdout, skillGroupPromptOfferFormat, offer.Kind, offer.Key, offerPromptSource(offer))
	}

	_, _ = fmt.Fprintf(a.stdout, skillGroupPromptFormat, scopeID, len(bulk))

	if !a.scanner.Scan() {
		return groupSkip
	}

	switch strings.ToLower(strings.TrimSpace(a.scanner.Text())) {
	case "a", "accept all":
		return groupAcceptAll
	case "d", "decline all":
		return groupDeclineAll
	case "r", "review each":
		return groupReviewEach
	default:
		return groupSkip
	}
}

// promptEach asks the per-offer y/N question for each offer; end of input
// is no answer and records nothing (ruling R29).
func (a *skillOfferAnswerer) promptEach(offers []SkillOffer) error {
	for _, offer := range offers {
		answer := promptForOffer(offer, a.scanner, a.stdout)
		if answer == SkillAnswerNone {
			continue
		}

		actErr := a.act(offer, answer == SkillAnswerAccept)
		if actErr != nil {
			return actErr
		}
	}

	return nil
}

// promptScope asks about a scope's unanswered offers: the register and
// refresh offers through the grouped question when there is more than one,
// and every removal through its own prompt.
func (a *skillOfferAnswerer) promptScope(scopeID string, offers []SkillOffer) error {
	bulk := make([]SkillOffer, 0, len(offers))
	removals := make([]SkillOffer, 0, len(offers))

	for _, offer := range offers {
		if offer.Kind == SkillOfferRemove {
			removals = append(removals, offer)
		} else {
			bulk = append(bulk, offer)
		}
	}

	if len(bulk) <= 1 {
		return a.promptEach(offers)
	}

	switch choice := a.askGroup(scopeID, bulk); choice {
	case groupAcceptAll, groupDeclineAll:
		accept := choice == groupAcceptAll
		for _, offer := range bulk {
			actErr := a.act(offer, accept)
			if actErr != nil {
				return actErr
			}
		}
	case groupReviewEach:
		reviewErr := a.promptEach(bulk)
		if reviewErr != nil {
			return reviewErr
		}
	case groupSkip:
	}

	return a.promptEach(removals)
}

// answerTokenRank ranks token against offer (design D6): an exact key
// highest, then a matching pattern by prefix length, then a matching
// selector; a removal is matched only by its exact key.
func answerTokenRank(offer SkillOffer, token string) int {
	if token == offer.Key {
		return answerRankExact
	}

	if offer.Kind == SkillOfferRemove {
		return answerRankNone
	}

	if scopeID, isSelector := strings.CutPrefix(token, skillAnswerSelectorPrefix); isSelector {
		if scopeID == offer.ScopeID {
			return answerRankSelector
		}

		return answerRankNone
	}

	prefix, isPattern := strings.CutSuffix(token, skillAnswerPatternSuffix)
	if isPattern && strings.HasPrefix(offer.Key, prefix) {
		return answerRankPatternBase + len(prefix)
	}

	return answerRankNone
}

// groupOffersByScope splits offers into scopes, in order of each scope's
// first offer (CompareSkillOffers sorts by scope, so scopes are contiguous).
func groupOffersByScope(offers []SkillOffer) []offerScope {
	index := map[string]int{}
	scopes := make([]offerScope, 0, len(offers))

	for _, offer := range offers {
		position, known := index[offer.ScopeID]
		if !known {
			position = len(scopes)
			index[offer.ScopeID] = position
			scopes = append(scopes, offerScope{id: offer.ScopeID})
		}

		scopes[position].offers = append(scopes[position].offers, offer)
	}

	return scopes
}

// offerPromptSource is the " (<source>)" a prompt shows after an offer's key
// so the user sees which file they are accepting, for every offer with a
// source; empty only for a removal with no recorded skill_source.
func offerPromptSource(offer SkillOffer) string {
	if offer.SourcePath == "" {
		return ""
	}

	return " (" + offer.SourcePath + ")"
}

// offerSourceLabel is the source a --dry-run line names: the offer's
// source path, or its note for a removal with no recorded skill_source.
func offerSourceLabel(offer SkillOffer) string {
	if offer.SourcePath == "" {
		return fmt.Sprintf(skillSourceNoteFormat, offer.Basename)
	}

	return offer.SourcePath
}

// printOutstandingSummary prints the one-line non-interactive summary
// (design D6) when any offer is outstanding.
func printOutstandingSummary(stdout io.Writer, scopes []outstandingScope) {
	if len(scopes) == 0 {
		return
	}

	total := 0
	labels := make([]string, 0, len(scopes))

	for _, scope := range scopes {
		total += scope.count
		labels = append(labels, fmt.Sprintf(skillOutstandingScopeFormat, scope.id, scope.count))
	}

	noun := "offers"
	if total == 1 {
		noun = "offer"
	}

	_, _ = fmt.Fprintf(stdout, skillRegistrationAwaitingAnswerFormat, total, noun, strings.Join(labels, ", "))
}

// tokenMatchesAnyOffer reports whether token would answer any offer.
func tokenMatchesAnyOffer(token string, offers []SkillOffer) bool {
	for _, offer := range offers {
		if answerTokenRank(offer, token) != answerRankNone {
			return true
		}
	}

	return false
}
