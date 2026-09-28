package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/vaultgraph"
)

// unexported variables.
var (
	errAmendExpectHashMismatch = errors.New(
		"amend: the note changed since it was judged; re-read it with engram show and judge it again")
	errAmendExpectHashRequired = errors.New(
		"amend: this note carries offer.origin; pass --expect-hash with the exchange hash you judged " +
			"(the # exchange_hash line of engram show)")
	errAmendIntoNotFound       = errors.New("amend: --into note not found")
	errAmendIntoSelf           = errors.New("amend: --into must name a different note than --target")
	errAmendIntoWithoutDiscard = errors.New("amend: --into is only valid with --discard")
)

// checkJudgedVersion is the judged-version check (design D10 r3-2; spec
// vault-offer-curation "Bookkeeping on a served offer SHALL verify the
// judged version"). A given --expect-hash is verified on any note; it is
// required for --clear-pending and --discard (bare or --into) on a note
// carrying offer.origin, pending or live. Only an equal hash passes: a
// hash of another version is not the version that was judged, so it fails
// as a changed one does (the spec fails the command whenever the current
// hash "differs from that value"; design D3's unknown-as-not-changed rule
// covers the exchange's own change checks, not a curator's judgment).
func checkJudgedVersion(args AmendArgs, raw []byte) error {
	if args.ExpectHash == "" {
		if judgedVersionRequired(args) && noteOfferOrigin(raw) != "" {
			return errAmendExpectHashRequired
		}

		return nil
	}

	current, hashErr := exchangeHash(raw)
	if hashErr != nil {
		return fmt.Errorf("amend: %w", hashErr)
	}

	if !exchangeHashesMatch(args.ExpectHash, current) {
		return fmt.Errorf("%w (expected %s, current %s)", errAmendExpectHashMismatch, args.ExpectHash, current)
	}

	return nil
}

// decodeExchangeFrontmatter reads a note's exchange frontmatter.
func decodeExchangeFrontmatter(raw []byte) (exchangeFrontmatter, error) {
	frontmatter, found := splitFrontmatter(raw)
	if !found {
		return exchangeFrontmatter{}, errAmendNoFrontmatter
	}

	var probe struct {
		Exchange exchangeFrontmatter `yaml:",inline"`
	}

	unmarshalErr := yaml.Unmarshal(frontmatter, &probe)
	if unmarshalErr != nil {
		return exchangeFrontmatter{}, fmt.Errorf("amend: parsing exchange fields: %w", unmarshalErr)
	}

	return probe.Exchange, nil
}

// discardTarget is --discard: with --into, the fold (foldInto); bare, the
// decline-recording delete (discardWithDecline).
func discardTarget(
	deps AmendDeps, args AmendArgs, notes []vaultgraph.Note, full string, raw []byte, stdout io.Writer,
) error {
	if args.Into != "" {
		return foldInto(deps, args.Vault, notes, full, raw, args.Into, stdout)
	}

	return discardWithDecline(deps, args.Vault, full, stdout)
}

// foldAliases is into's aliases after a fold (design D10 M12): its own,
// then the offer's basename and every alias the offer answered to, each
// once, never into's own basename.
func foldAliases(into []string, intoBase, offerBase string, offer []string) []string {
	folded := make([]string, 0, len(into)+len(offer)+1)

	for _, name := range slices.Concat(into, []string{offerBase}, offer) {
		if name != intoBase && !slices.Contains(folded, name) {
			folded = append(folded, name)
		}
	}

	return folded
}

// foldInto is `amend --discard --into` (design D10): the offer's basename,
// aliases and parent links are recorded on the existing note, and then the
// offer and its sidecar are deleted. Only into's aliases: and parent: keys
// are rewritten: nothing is re-embedded, identity is not re-stamped
// (ruling S7), no xid is stamped, nothing is queued, and no decline is
// recorded — the links themselves keep the parent note from being pulled
// again.
func foldInto(
	deps AmendDeps, vault string, notes []vaultgraph.Note, offerFull string, offerRaw []byte, into string,
	stdout io.Writer,
) error {
	intoRel, findErr := findNote(notes, into)
	if findErr != nil {
		return fmt.Errorf("%w: %q", errAmendIntoNotFound, into)
	}

	intoFull := filepath.Join(vault, intoRel)
	if intoFull == offerFull {
		return errAmendIntoSelf
	}

	intoRaw, readErr := deps.Read(intoFull)
	if readErr != nil {
		return fmt.Errorf("amend: read %s: %w", intoRel, readErr)
	}

	folded, foldErr := foldedContent(intoRaw, strings.TrimSuffix(intoRel, mdExt), offerRaw,
		strings.TrimSuffix(filepath.Base(offerFull), mdExt))
	if foldErr != nil {
		return foldErr
	}

	if folded != string(intoRaw) {
		writeErr := deps.Write(intoFull, []byte(folded))
		if writeErr != nil {
			return fmt.Errorf("amend: write %s: %w", intoRel, writeErr)
		}
	}

	return discardNote(deps, offerFull, stdout)
}

// foldParentLinks is into's parent block after a fold (design D10, D4):
// every link the offer held joins into's links under the offer's parent
// vault. The offer's primary (offered or pulled) stays primary only when
// into has none; otherwise it becomes covered. A link into already holds
// keeps its role and takes the offer's hash, the version just judged.
// When the offer links a different parent vault than into does, into's
// links under the old vault are dropped, as a receipt drops them (D6).
// The offer's author is not carried: into keeps its own (D10).
func foldParentLinks(into, offer parentLinks) parentLinks {
	if len(offer.Links) == 0 {
		return into
	}

	folded := parentLinks{
		Vault:  offer.Vault,
		Links:  make([]parentLink, 0, len(into.Links)+len(offer.Links)),
		Author: into.Author,
	}
	if into.Vault == offer.Vault {
		folded.Links = append(folded.Links, into.Links...)
	}

	hasPrimary := slices.ContainsFunc(folded.Links, func(link parentLink) bool { return isPrimaryLink(link.Via) })

	for _, link := range offer.Links {
		index := slices.IndexFunc(folded.Links, func(held parentLink) bool { return held.Note == link.Note })
		if index >= 0 {
			folded.Links[index].Hash = link.Hash

			continue
		}

		if isPrimaryLink(link.Via) {
			if hasPrimary {
				link.Via = linkViaCovered
			}

			hasPrimary = true
		}

		folded.Links = append(folded.Links, link)
	}

	return folded
}

// foldedContent is into's content after folding the offer into it: only
// the aliases: and parent: keys change, and only when the fold adds to
// them.
func foldedContent(intoRaw []byte, intoBase string, offerRaw []byte, offerBase string) (string, error) {
	into, intoErr := decodeExchangeFrontmatter(intoRaw)
	if intoErr != nil {
		return "", intoErr
	}

	offer, offerErr := decodeExchangeFrontmatter(offerRaw)
	if offerErr != nil {
		return "", offerErr
	}

	content := string(intoRaw)

	aliases := foldAliases(into.Aliases, intoBase, offerBase, offer.Aliases)
	if !slices.Equal(aliases, into.Aliases) {
		var setErr error

		content, setErr = setFrontmatterBlock(content, aliasesKey, map[string][]string{aliasesKey: aliases}, offerFieldKey)
		if setErr != nil {
			return "", fmt.Errorf("amend: fold: %w", setErr)
		}
	}

	parent := foldParentLinks(into.Parent, offer.Parent)
	if !sameParentLinks(parent, into.Parent) {
		var setErr error

		content, setErr = setFrontmatterBlock(content, parentKey, map[string]parentLinks{parentKey: parent},
			aliasesKey, offerFieldKey)
		if setErr != nil {
			return "", fmt.Errorf("amend: fold: %w", setErr)
		}
	}

	return content, nil
}

// isPrimaryLink reports whether a link role is a note's primary
// counterpart (design D4).
func isPrimaryLink(via string) bool {
	return via == linkViaOffered || via == linkViaPulled
}

// judgedVersionRequired reports whether an amend is curation bookkeeping
// that must name the judged version on a served offer: clearing the
// pending marker, or a discard (bare or --into).
func judgedVersionRequired(args AmendArgs) bool {
	return args.Discard || args.ClearPending || (args.Pending != nil && !*args.Pending)
}

// noteOfferOrigin is the note's offer.origin ("" when it has none, or when
// the note does not parse).
func noteOfferOrigin(raw []byte) string {
	exchange, err := decodeExchangeFrontmatter(raw)
	if err != nil {
		return ""
	}

	return exchange.Offer.Origin
}

// readJudgedTarget reads the amend's target note and runs the
// judged-version check on it (checkJudgedVersion) before anything is
// written.
func readJudgedTarget(args AmendArgs, deps AmendDeps, full, relPath string) ([]byte, error) {
	raw, readErr := deps.Read(full)
	if readErr != nil {
		return nil, fmt.Errorf("amend: read %s: %w", relPath, readErr)
	}

	judgedErr := checkJudgedVersion(args, raw)
	if judgedErr != nil {
		return nil, judgedErr
	}

	return raw, nil
}

// sameParentLinks reports whether two parent blocks are equal.
func sameParentLinks(first, second parentLinks) bool {
	return first.Vault == second.Vault && first.Author == second.Author && slices.Equal(first.Links, second.Links)
}
