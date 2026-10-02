package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	// basenameParts is "<luhmann>.<date>.<slug>".
	basenameParts = 3
)

// unexported variables.
var (
	errOfferNoteNoSlug      = errors.New("offer: the note's basename has no <luhmann>.<date>.<slug> form")
	errOfferNoteUnparseable = errors.New("offer: the note's frontmatter does not parse")
)

// offerNoteDoc is every frontmatter field an offer payload reads, across
// the fact, feedback and runbook shapes.
type offerNoteDoc struct {
	Type       string              `yaml:"type"`
	Tier       string              `yaml:"tier"`
	Situation  string              `yaml:"situation"`
	Subject    string              `yaml:"subject"`
	Predicate  string              `yaml:"predicate"`
	Object     string              `yaml:"object"`
	Behavior   string              `yaml:"behavior"`
	Impact     string              `yaml:"impact"`
	Action     string              `yaml:"action"`
	DoneWhen   string              `yaml:"done_when"`
	RedFlags   []string            `yaml:"red_flags"`
	Triggers   []string            `yaml:"triggers"`
	Source     string              `yaml:"source"`
	Project    string              `yaml:"project"`
	Issue      string              `yaml:"issue"`
	Repo       string              `yaml:"repo"`
	User       string              `yaml:"user"`
	Vault      string              `yaml:"vault"`
	Supersedes []supersedesEntry   `yaml:"supersedes"`
	Exchange   exchangeFrontmatter `yaml:",inline"`
}

// offerPayload is the /learn request body an offer sends (design D5): the
// LearnArgs wire shape, restricted to what an offer carries. It has no
// target, position, chunkSources, vault or pending key at all — placement
// and pending are the parent's (G8), chunks never travel (Q1). Tags are
// not sent either: they are this vault's vocabulary, and the parent's own
// vocab assigns its tags (the same reasoning as D8's pull-down strip).
type offerPayload struct {
	Type       string     `json:"type"`
	Slug       string     `json:"slug"`
	VaultName  string     `json:"vaultName,omitempty"`
	Source     string     `json:"source,omitempty"`
	Project    string     `json:"project,omitempty"`
	Issue      string     `json:"issue,omitempty"`
	Tier       string     `json:"tier,omitempty"`
	Supersedes []string   `json:"supersedes,omitempty"`
	Situation  string     `json:"situation"`
	Behavior   string     `json:"behavior,omitempty"`
	Impact     string     `json:"impact,omitempty"`
	Action     string     `json:"action,omitempty"`
	Subject    string     `json:"subject,omitempty"`
	Predicate  string     `json:"predicate,omitempty"`
	Object     string     `json:"object,omitempty"`
	DoneWhen   string     `json:"doneWhen,omitempty"`
	Body       string     `json:"body,omitempty"`
	RedFlags   []string   `json:"redFlags,omitempty"`
	Triggers   []string   `json:"triggers,omitempty"`
	Repo       string     `json:"repo,omitempty"`
	User       string     `json:"user"`
	Offer      LearnOffer `json:"offer"`
}

// offerPayloadContext is what the payload builder needs beyond the note:
// this vault's ID, the configured parent's vault ID ("" when not yet
// known), the sending process's identity detection (a fallback only), and
// the primary-link parent basename of a local note named by supersedes.
type offerPayloadContext struct {
	localVaultID     string
	parentVaultID    string
	detectRepo       func() string
	detectUser       func() string
	supersedesTarget func(local string) (string, bool)
}

// buildOfferPayload builds the /learn request body for a queued note from
// its current content (design D5, D6): a learn-offer, or an amend-offer
// whose offer.for is the note's primary link under the parent's vault ID.
// It carries the note's own user/repo (detection only fills in when they
// are empty, M5), supersedes translated to parent basenames (dropped when
// unlinked), offer.origin = <local vault id>:<xid>, offer.key =
// sha256(origin + exchange hash), and offer.path — the local ID, appended
// to an accepted served offer's own path when the note is one (M14).
func buildOfferPayload(note offerNote, pctx offerPayloadContext) ([]byte, error) {
	raw := toLF(note.Raw) // a CRLF note is offered as its LF form (#789 design D4)

	frontmatter, found := splitFrontmatter(raw)
	if !found {
		return nil, errOfferNoteUnparseable
	}

	var doc offerNoteDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("%w: %w", errOfferNoteUnparseable, unmarshalErr)
	}

	withdrawErr := offerWithdrawnReason(note, doc.Exchange, pctx.parentVaultID)
	if withdrawErr != nil {
		return nil, withdrawErr
	}

	slug, slugErr := slugOfBasename(note.Basename)
	if slugErr != nil {
		return nil, slugErr
	}

	payload := offerPayload{
		Type: doc.Type, Slug: slug, VaultName: doc.Vault, Source: doc.Source, Project: doc.Project,
		Issue: doc.Issue, Tier: doc.Tier, Supersedes: translateSupersedes(doc.Supersedes, pctx.supersedesTarget),
		Situation: doc.Situation, Behavior: doc.Behavior, Impact: doc.Impact, Action: doc.Action,
		Subject: doc.Subject, Predicate: doc.Predicate, Object: doc.Object,
		DoneWhen: doc.DoneWhen, RedFlags: doc.RedFlags, Triggers: doc.Triggers,
		Repo: orDetected(doc.Repo, pctx.detectRepo), User: orDetected(doc.User, pctx.detectUser),
		Offer: offerRecordFor(note, doc.Exchange, pctx),
	}

	if doc.Type == typeRunbook {
		payload.Body = runbookSteps(raw)
	}

	encoded, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return nil, fmt.Errorf("offer: encoding the payload: %w", marshalErr)
	}

	return encoded, nil
}

// offerKey is the idempotency key: sha256(offer.origin + exchange hash),
// hex-encoded (design D6).
func offerKey(origin, hash string) string {
	sum := sha256.Sum256([]byte(origin + hash))

	return hex.EncodeToString(sum[:])
}

// offerRecordFor builds the offer's wire record for a note.
func offerRecordFor(note offerNote, exchange exchangeFrontmatter, pctx offerPayloadContext) LearnOffer {
	origin := pctx.localVaultID + ":" + note.XID

	path := []string{pctx.localVaultID}
	if exchange.Offer.Origin != "" {
		// An accepted served offer going onward (M14, D12): the path it
		// already travelled — at least its origin vault — plus this vault.
		travelled := slices.Clone(exchange.Offer.Path)
		if len(travelled) == 0 {
			originVault, _, _ := strings.Cut(exchange.Offer.Origin, ":")
			travelled = []string{originVault}
		}

		travelled = append(travelled, pctx.localVaultID)
		path = travelled
	}

	record := LearnOffer{Origin: origin, Key: offerKey(origin, note.Hash), Path: path}

	if link, linked := primaryParentLink(exchange, pctx.parentVaultID); linked {
		record.For = link.Note
	}

	return record
}

// offerWithdrawnReason is the send-time re-check, with the parent's ID now
// known (ruling S16): an offer whose origin vault is the parent never goes
// back to it (D12, unconditional), and one whose content equals its
// primary link's hash is a loop (D5). Either is withdrawn, not sent.
func offerWithdrawnReason(note offerNote, exchange exchangeFrontmatter, parentVaultID string) error {
	if originIsVault(exchange.Offer.Origin, parentVaultID) {
		return fmt.Errorf("%w: its origin vault %s is the parent", errOfferWithdrawn, parentVaultID)
	}

	link, linked := primaryParentLink(exchange, parentVaultID)
	if linked && exchangeHashesMatch(link.Hash, note.Hash) {
		return fmt.Errorf("%w: the parent already holds this content (%s)", errOfferWithdrawn, link.Note)
	}

	return nil
}

// orDetected is the note's own value, or the detected one when it is empty.
func orDetected(own string, detect func() string) string {
	if own != "" || detect == nil {
		return own
	}

	return detect()
}

// runbookSteps is a runbook note's caller-authored steps: its body with
// the Supersedes: lines (re-rendered by the parent) removed.
func runbookSteps(raw []byte) string {
	return replaceSupersedes(string(embed.ExtractBody(raw)), nil)
}

// slugOfBasename extracts <slug> from "<luhmann>.<date>.<slug>".
func slugOfBasename(basename string) (string, error) {
	parts := strings.SplitN(strings.TrimSuffix(basename, mdExt), ".", basenameParts)
	if len(parts) != basenameParts || parts[basenameParts-1] == "" {
		return "", fmt.Errorf("%w: %q", errOfferNoteNoSlug, basename)
	}

	return parts[basenameParts-1], nil
}

// translateSupersedes rewrites each supersedes entry to its target's
// primary-link parent basename, as a --supersedes flag value; an entry
// whose target has no parent link is dropped (design D5).
func translateSupersedes(entries []supersedesEntry, target func(string) (string, bool)) []string {
	translated := make([]string, 0, len(entries))

	for _, entry := range entries {
		if target == nil {
			break
		}

		parentNote, found := target(strings.TrimSuffix(entry.Note, mdExt))
		if !found {
			continue
		}

		translated = append(translated, parentNote+"|"+entry.Type+"|"+entry.Claim)
	}

	return translated
}
