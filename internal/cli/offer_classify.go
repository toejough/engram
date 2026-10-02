package cli

import (
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// unexported constants.
const (
	linkViaCovered = "covered"
	linkViaOffered = "offered"
	linkViaPulled  = "pulled"
)

// offerCommand names the write a classification is asked about (design D5).
type offerCommand int

// offerCommand values.
const (
	offerCmdLearn offerCommand = iota + 1
	offerCmdLearnQA
	offerCmdAmend
	offerCmdResituate
	offerCmdIdentityBackfill
	offerCmdRegistration
)

// offerClassNote is what the classification reads from a written note.
type offerClassNote struct {
	pending   bool
	skillHash string
	exchange  exchangeFrontmatter
	hash      string
}

// offerWrite is one local write as the offer classification sees it: the
// command, its amend flags (amend only), and the note as written.
type offerWrite struct {
	command offerCommand
	amend   AmendArgs
	raw     []byte
}

// amendHasContentFlag reports whether an amend supplies any content flag
// (design D5): situation, subject/predicate/object, behavior/impact/action,
// done-when, body, red-flag or trigger. --supersedes and --chunk-source are
// relations, not content.
func amendHasContentFlag(args AmendArgs) bool {
	contentFlags := []string{
		args.Situation, args.Subject, args.Predicate, args.Object,
		args.Behavior, args.Impact, args.Action, args.DoneWhen, args.Body,
	}

	return slices.ContainsFunc(contentFlags, func(value string) bool { return value != "" }) ||
		len(args.RedFlags) > 0 || len(args.Triggers) > 0
}

// amendIsOffered is the amend rows of design D5: a content amend is
// offered; clearing the pending marker of a served offer (one carrying
// offer.origin) propagates it upward unless its origin vault is the
// configured parent (M14, D12); every other amend is bookkeeping.
func amendIsOffered(args AmendArgs, note offerClassNote, parentVaultID string) bool {
	if args.Discard {
		return false
	}

	if amendHasContentFlag(args) {
		return true
	}

	clearsPending := args.ClearPending || (args.Pending != nil && !*args.Pending)
	if !clearsPending || note.exchange.Offer.Origin == "" {
		return false
	}

	// With the parent's ID unknown the offer is queued; the drain learns
	// the ID before sending and withdraws it then (offerWithdrawnReason).
	return !originIsVault(note.exchange.Offer.Origin, parentVaultID)
}

// classifyOffer is design D5's table as one pure function over the write
// (command, amend flags, the note as written) and the configured parent's
// vault ID ("" when not yet known). It reports whether the write is queued
// as an offer. Whether that offer is a learn-offer or an amend-offer is the
// payload builder's call, at send time.
func classifyOffer(write offerWrite, parentVaultID string) bool {
	note, ok := parseOfferClassNote(write.raw)
	if !ok || note.pending || note.skillHash != "" {
		return false
	}

	switch write.command {
	case offerCmdLearn, offerCmdResituate:
	case offerCmdAmend:
		if !amendIsOffered(write.amend, note, parentVaultID) {
			return false
		}
	case offerCmdLearnQA, offerCmdIdentityBackfill, offerCmdRegistration:
		return false
	default:
		return false
	}

	// Loop suppression: fires only on an equal current-version hash (D3).
	link, linked := primaryParentLink(note.exchange, parentVaultID)

	return !linked || !exchangeHashesMatch(link.Hash, note.hash)
}

// originIsVault reports whether an offer.origin's vault part is vaultID
// (never for an unknown vault ID).
func originIsVault(origin, vaultID string) bool {
	originVault, _, _ := strings.Cut(origin, ":")

	return vaultID != "" && originVault == vaultID
}

// parseOfferClassNote reads the fields the classification needs; ok is
// false for a note that is not a fact, feedback or runbook, or does not
// parse.
func parseOfferClassNote(raw []byte) (offerClassNote, bool) {
	frontmatter, found := splitFrontmatter(toLF(raw))
	if !found {
		return offerClassNote{}, false
	}

	var doc struct {
		Type      string              `yaml:"type"`
		Pending   bool                `yaml:"pending"`
		SkillHash string              `yaml:"skill_hash"`
		Exchange  exchangeFrontmatter `yaml:",inline"`
	}

	if yaml.Unmarshal(frontmatter, &doc) != nil {
		return offerClassNote{}, false
	}

	if doc.Type != typeFact && doc.Type != typeFeedback && doc.Type != typeRunbook {
		return offerClassNote{}, false
	}

	hash, hashErr := exchangeHash(raw)
	if hashErr != nil {
		return offerClassNote{}, false
	}

	return offerClassNote{
		pending: doc.Pending, skillHash: doc.SkillHash, exchange: doc.Exchange, hash: hash,
	}, true
}

// primaryParentLink returns the note's primary link (via offered or
// pulled) to the configured parent, whose vault ID must be known: a link
// recorded under any other vault ID — or any link at all while the
// parent's ID is unknown — is never the parent's (ruling S16).
func primaryParentLink(exchange exchangeFrontmatter, parentVaultID string) (parentLink, bool) {
	if parentVaultID == "" || exchange.Parent.Vault != parentVaultID {
		return parentLink{}, false
	}

	for _, link := range exchange.Parent.Links {
		if link.Via == linkViaOffered || link.Via == linkViaPulled {
			return link, true
		}
	}

	return parentLink{}, false
}
