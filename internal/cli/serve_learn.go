package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/luhmann"
)

// unexported constants.
const (
	// dryRunLuhmann is the placeholder ID validateServedLearn renders with.
	dryRunLuhmann = "1"
	// maxOfferPathEntries bounds offer.path (design D7 r4 L-C): a real
	// parent tree is a few levels deep, so a longer path is malformed.
	maxOfferPathEntries = 16
)

// unexported variables.
var (
	errOfferCycle   = errors.New("serve: offer already passed through this vault (offer.path holds its vault ID)")
	errOfferInvalid = errors.New("serve: invalid offer")
	// errServedLearnInvalid marks a served learn whose note content fails
	// validation — a client error (400), never retried as transient.
	errServedLearnInvalid = errors.New("serve: invalid learn request")
)

// exchangeNote is one fact, feedback or runbook note as served learn's
// lookup sees it: its name, raw bytes, pending marker, created date and
// exchange frontmatter.
type exchangeNote struct {
	basename string
	path     string
	raw      []byte
	pending  bool
	created  string
	exchange exchangeFrontmatter
}

// offerReceipt is the JSON response body for a served learn (design D7):
// the note the offer landed on, the vault, and the exchange hash as stored.
// It never carries note content or any curation outcome. For names the
// resolved live target, and is omitted when there is none.
type offerReceipt struct {
	Status     string `json:"status"`
	Luhmann    string `json:"luhmann"`
	Basename   string `json:"basename"`
	Pending    bool   `json:"pending"`
	VaultID    string `json:"vault_id"`    //nolint:tagliatelle // design D7 fixes the receipt's snake_case keys
	StoredHash string `json:"stored_hash"` //nolint:tagliatelle // design D7 fixes the receipt's snake_case keys
	For        string `json:"for,omitempty"`
}

// servedLearnDeps is what a served learn needs beyond LearnDeps: the note
// reader for the lookup, the xid source, and the served vault's ID.
type servedLearnDeps struct {
	learn    LearnDeps
	readFile func(path string) ([]byte, error)
	mintXID  func() (string, error)
	vaultID  string
}

// applySameOriginLive handles an offer whose origin matches a live note — an
// already-accepted offer (design D7 case 2): the same key is a retry, answered
// with the live note's receipt and no write; a new key becomes a new pending
// note whose offer.for names the live note.
func applySameOriginLive(
	ctx context.Context, args LearnArgs, deps servedLearnDeps, live exchangeNote, notes []exchangeNote,
) (offerReceipt, error) {
	if offerKeysMatch(live.exchange.Offer.Key, args.Offer.Key) {
		return receiptForNote(live, notes, deps.vaultID)
	}

	args.Offer.For = live.basename

	return writeNewPendingOffer(ctx, args, deps, live.basename)
}

// applySameOriginPending handles an offer whose origin matches a pending
// note (design D7 case 1): the same key writes nothing and returns its
// receipt; a new key rewrites it in place.
func applySameOriginPending(
	ctx context.Context, args LearnArgs, deps servedLearnDeps, pending exchangeNote, notes []exchangeNote,
) (offerReceipt, error) {
	if offerKeysMatch(pending.exchange.Offer.Key, args.Offer.Key) {
		return receiptForNote(pending, notes, deps.vaultID)
	}

	return rewritePendingOffer(ctx, args, deps, pending, notes)
}

// applyServedOffer runs the origin matching and offer.for resolution over
// the vault's notes (design D7 cases 1–3). The caller holds the vault lock.
func applyServedOffer(
	ctx context.Context, args LearnArgs, deps servedLearnDeps, notes []exchangeNote,
) (offerReceipt, error) {
	if args.Offer.Origin != "" {
		if note, found := findByOrigin(notes, args.Offer.Origin, true); found {
			return applySameOriginPending(ctx, args, deps, note, notes)
		}

		if note, found := findByOrigin(notes, args.Offer.Origin, false); found {
			return applySameOriginLive(ctx, args, deps, note, notes)
		}
	}

	target, live := resolveOfferFor(notes, args.Offer.For)
	args.Offer.For = target

	liveFor := ""
	if live {
		liveFor = target
	}

	return writeNewPendingOffer(ctx, args, deps, liveFor)
}

// findByOrigin returns the first note with the given pending state whose
// offer.origin is origin.
func findByOrigin(notes []exchangeNote, origin string, pending bool) (exchangeNote, bool) {
	for _, note := range notes {
		if note.pending == pending && note.exchange.Offer.Origin == origin {
			return note, true
		}
	}

	return exchangeNote{}, false
}

// luhmannOfBasename extracts the leading Luhmann-ID segment of a basename
// ("<luhmann>.<date>.<slug>"), or "" when it has none.
func luhmannOfBasename(basename string) string {
	id, _ := luhmann.FromBasename(basename)

	return id
}

// offerKeysMatch reports whether an incoming offer.key repeats a recorded
// one. An empty key never matches, so a keyless offer is never mistaken for
// a retry.
func offerKeysMatch(recorded, incoming string) bool {
	return incoming != "" && recorded == incoming
}

// parseExchangeNote reads one listed note for the lookup; ok is false for a
// note that is not a fact, feedback or runbook, or does not parse.
func parseExchangeNote(vault, name string, raw []byte) (exchangeNote, bool) {
	frontmatter, found := splitFrontmatter(raw)
	if !found {
		return exchangeNote{}, false
	}

	noteType := peekNoteType(frontmatter)
	if noteType != typeFact && noteType != typeFeedback && noteType != typeRunbook {
		return exchangeNote{}, false
	}

	var probe struct {
		Created  string              `yaml:"created"`
		Exchange exchangeFrontmatter `yaml:",inline"`
	}

	if yaml.Unmarshal(frontmatter, &probe) != nil {
		return exchangeNote{}, false
	}

	return exchangeNote{
		basename: strings.TrimSuffix(name, mdExt),
		path:     filepath.Join(vault, name),
		raw:      raw,
		pending:  noteHasPendingMarker(raw),
		created:  probe.Created,
		exchange: probe.Exchange,
	}, true
}

// receiptForNote is the receipt for an existing note, with no write: its
// name, pending state and stored exchange hash, and — for a pending offer
// whose offer.for names a live note — that note's current basename.
func receiptForNote(note exchangeNote, notes []exchangeNote, vaultID string) (offerReceipt, error) {
	hash, hashErr := exchangeHash(note.raw)
	if hashErr != nil {
		return offerReceipt{}, fmt.Errorf("serve: %s: %w", note.basename, hashErr)
	}

	liveFor := ""

	if note.pending {
		if target, live := resolveOfferFor(notes, note.exchange.Offer.For); live {
			liveFor = target
		}
	}

	return offerReceipt{
		Status:     offerReceivedStatus,
		Luhmann:    luhmannOfBasename(note.basename),
		Basename:   note.basename,
		Pending:    note.pending,
		VaultID:    vaultID,
		StoredHash: hash,
		For:        liveFor,
	}, nil
}

// resolveOfferFor resolves an offer.for basename against live basenames,
// then live notes' aliases, then pending basenames (design D7 case 3). It
// returns the resolved note's current basename and whether that note is
// live; an unresolved name returns "".
func resolveOfferFor(notes []exchangeNote, name string) (string, bool) {
	name = strings.TrimSuffix(name, mdExt)
	if name == "" {
		return "", false
	}

	byOrder := []func(exchangeNote) bool{
		func(note exchangeNote) bool { return !note.pending && note.basename == name },
		func(note exchangeNote) bool { return !note.pending && slices.Contains(note.exchange.Aliases, name) },
		func(note exchangeNote) bool { return note.pending && note.basename == name },
	}

	for _, matches := range byOrder {
		if index := slices.IndexFunc(notes, matches); index >= 0 {
			return notes[index].basename, !notes[index].pending
		}
	}

	return "", false
}

// rewritePendingOffer rewrites a same-origin pending offer in place (design
// D7 case 1, H3): new content, offer.key and offer.path under the same
// basename, still pending, keeping its xid, links, aliases, origin and
// offer.for. It re-embeds when the exchange hash changed.
func rewritePendingOffer(
	ctx context.Context, args LearnArgs, deps servedLearnDeps, note exchangeNote, notes []exchangeNote,
) (offerReceipt, error) {
	args.XID = note.exchange.XID
	args.Parent = note.exchange.Parent
	args.Aliases = note.exchange.Aliases
	args.Offer.For = note.exchange.Offer.For

	when, parseErr := time.Parse(dateFormat, note.created)
	if parseErr != nil {
		when = deps.learn.Now()
	}

	identity := identityStamp{
		Repo:  deps.learn.DetectRepo(ctx),
		User:  deps.learn.DetectUser(ctx),
		Vault: args.VaultName,
	}

	content, contentErr := assembleLearnContent(args, luhmannOfBasename(note.basename), when, identity)
	if contentErr != nil {
		return offerReceipt{}, fmt.Errorf("serve: %w", contentErr)
	}

	writeErr := deps.learn.WriteNote(note.path, []byte(content))
	if writeErr != nil {
		return offerReceipt{}, fmt.Errorf("serve: rewriting %s: %w", note.basename, writeErr)
	}

	previous, previousErr := exchangeHash(note.raw)
	if previousErr != nil {
		previous = ""
	}

	stored, hashErr := exchangeHash([]byte(content))
	if hashErr != nil {
		return offerReceipt{}, fmt.Errorf("serve: %s: %w", note.basename, hashErr)
	}

	// Re-embedding is safe to repeat, so it runs unless the content is
	// provably the same (an unknown comparison re-embeds).
	if !exchangeHashesMatch(previous, stored) {
		autoEmbedNote(ctx, deps.learn, note.path, content)
	}

	applyVocabAssignmentAfterLearn(deps.learn, args.Vault, note.path, content)

	rewritten := note
	rewritten.raw = []byte(content)

	return receiptForNote(rewritten, notes, deps.vaultID)
}

// runServedLearn performs a validated served learn (design D7): it refuses
// an offer whose path already holds this vault's ID, then does the lookup,
// any rewrite, any re-embed and building the receipt in one locked section.
func runServedLearn(ctx context.Context, args LearnArgs, deps servedLearnDeps) (offerReceipt, error) {
	if slices.Contains(args.Offer.Path, deps.vaultID) {
		return offerReceipt{}, errOfferCycle
	}

	// Placement is the server's: every new pending note lands at top level,
	// whatever target/position the caller sent (G8).
	args.Target = ""
	args.Position = positionTop
	args.Pending = true

	release, lockErr := deps.learn.Lock(args.Vault)
	if lockErr != nil {
		return offerReceipt{}, fmt.Errorf("serve: acquiring lock: %w", lockErr)
	}
	defer release()

	notes, scanErr := scanExchangeNotes(args.Vault, deps.learn.ListMD, deps.readFile)
	if scanErr != nil {
		return offerReceipt{}, scanErr
	}

	return applyServedOffer(ctx, args, deps, notes)
}

// scanExchangeNotes reads every fact, feedback and runbook note in vault
// for served learn's lookup. An unreadable or unparseable note is skipped.
func scanExchangeNotes(
	vault string, listMD func(string) ([]string, error), readFile func(string) ([]byte, error),
) ([]exchangeNote, error) {
	names, listErr := listMD(vault)
	if listErr != nil {
		return nil, fmt.Errorf("serve: listing notes: %w", listErr)
	}

	notes := make([]exchangeNote, 0, len(names))

	for _, name := range names {
		raw, readErr := readFile(filepath.Join(vault, name))
		if readErr != nil {
			continue
		}

		if note, ok := parseExchangeNote(vault, name, raw); ok {
			notes = append(notes, note)
		}
	}

	return notes, nil
}

// validateOffer checks a served learn's offer before anything else runs
// (design D7 r4 L-C): at most maxOfferPathEntries path entries, each an
// exchange ID, and an origin that is empty or "<vault id>:<xid>".
func validateOffer(offer LearnOffer) error {
	if len(offer.Path) > maxOfferPathEntries {
		return fmt.Errorf("%w: offer.path has %d entries (at most %d)",
			errOfferInvalid, len(offer.Path), maxOfferPathEntries)
	}

	for _, entry := range offer.Path {
		if !isExchangeID(entry) {
			return fmt.Errorf("%w: offer.path entry %q is not 32 lowercase hex characters", errOfferInvalid, entry)
		}
	}

	if offer.Origin == "" {
		return nil
	}

	vaultPart, xidPart, found := strings.Cut(offer.Origin, ":")
	if !found || !isExchangeID(vaultPart) || !isExchangeID(xidPart) {
		return fmt.Errorf("%w: offer.origin %q is not <32 hex>:<32 hex>", errOfferInvalid, offer.Origin)
	}

	return nil
}

// validateServedLearn checks everything a served learn can reject as a
// client error before touching the vault: the declared identity, the
// offer, and the note input.
func validateServedLearn(args LearnArgs) error {
	if args.User == "" {
		return errServeEmptyIdentity
	}

	offerErr := validateOffer(args.Offer)
	if offerErr != nil {
		return offerErr
	}

	inputErr := validateLearnInput(args)
	if inputErr != nil {
		return fmt.Errorf("%w: %w", errServedLearnInvalid, inputErr)
	}

	// A dry render checks the content rules (type, tier, supersedes,
	// required fields) with no I/O, so a malformed offer is a 400 before the
	// vault lock is taken, never a 5xx the child would retry.
	_, contentErr := assembleLearnContent(args, dryRunLuhmann, time.Time{}, identityStamp{})
	if contentErr != nil {
		return fmt.Errorf("%w: %w", errServedLearnInvalid, contentErr)
	}

	return nil
}

// writeNewPendingOffer writes the offer as a new top-level pending note with
// a fresh server xid and its offer record, embeds it, and returns its
// receipt; liveFor is the resolved live target reported as `for`.
func writeNewPendingOffer(
	ctx context.Context, args LearnArgs, deps servedLearnDeps, liveFor string,
) (offerReceipt, error) {
	xid, mintErr := deps.mintXID()
	if mintErr != nil {
		return offerReceipt{}, fmt.Errorf("serve: %w", mintErr)
	}

	args.XID = xid

	path, content, writeErr := writeLearnLocked(ctx, args, deps.learn, args.Vault)
	if writeErr != nil {
		return offerReceipt{}, writeErr
	}

	stored, hashErr := exchangeHash([]byte(content))
	if hashErr != nil {
		return offerReceipt{}, fmt.Errorf("serve: %w", hashErr)
	}

	basename := strings.TrimSuffix(filepath.Base(path), mdExt)

	return offerReceipt{
		Status:     offerReceivedStatus,
		Luhmann:    luhmannOfBasename(basename),
		Basename:   basename,
		Pending:    true,
		VaultID:    deps.vaultID,
		StoredHash: stored,
		For:        liveFor,
	}, nil
}
