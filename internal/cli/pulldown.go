package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	declinedFileName = "declined.json"
	declinedVersion  = 1
	// pulledSlugFallback names a pulled copy whose parent basename carries
	// no usable slug.
	pulledSlugFallback = "pulled"
)

// unexported variables.
var (
	errPullHashMismatch = errors.New("the pulled copy's exchange hash differs from the parent's " +
		"(the parent note does not round-trip)")
	errPullNoFrontmatter = errors.New("the parent note has no frontmatter")
	errPullNotFound      = errors.New("not found on the parent")
	errPullPaused        = errors.New("exchange with the parent is paused (see the warning above)")
	errPullSelfParent    = errors.New("the parent reports this vault's own ID")
	errPullTooOld        = errors.New("the parent is too old: /show?raw=1 returned no envelope with a vault_id " +
		"(upgrade engram on the parent host)")
	errPullUnreachable     = errors.New("the parent is unreachable")
	errPullUnsupportedType = errors.New("only fact, feedback and runbook notes are pulled down")
)

// declinedFile is .engram/declined.json: the parent notes whose pulled
// copies were discarded outright (design D8, H5).
type declinedFile struct {
	Version int            `json:"version"`
	Entries []declinedPull `json:"entries"`
}

// declinedPull is one declined parent note: the parent's vault ID (a
// decline is keyed by vault, as links are — ruling S16), its basename and
// the exchange hash that was declined.
type declinedPull struct {
	Vault    string `json:"vault"`
	Basename string `json:"basename"`
	Hash     string `json:"hash"`
}

// pullSession is one activate command's parent side (design D8): the
// pre-exchange checks run once, and an outage or a failed check pauses
// every later pull in the same command.
type pullSession struct {
	deps      Deps
	vault     string
	vaultName string
	parentURL string
	store     outboxStore
	learn     LearnDeps
	stdout    func(format string, args ...any)
	prepared  bool
	paused    bool
	localID   string
	contacted bool
}

// fetchEnvelope fetches GET /show?raw=1 and validates the envelope. Any
// answer resets the backoff and caches the parent's reported vault ID
// (design D6, ruling S14); an outage records a failure and pauses the
// session with the one warning.
func (s *pullSession) fetchEnvelope(ctx context.Context, name string) (rawShowResponse, error) {
	resp, fetchErr := fetchRaw(ctx, s.deps, s.parentURL, methodGet, "/show",
		map[string][]string{"note": {name}, "raw": {"1"}}, nil)
	if fetchErr != nil {
		if errors.Is(fetchErr, errParentUnreachable) {
			s.paused = true

			s.warnUnreachable()

			return rawShowResponse{}, fmt.Errorf("%w: %w", errPullUnreachable, fetchErr)
		}

		s.noteContact("")

		return rawShowResponse{}, fmt.Errorf("%w: %w", errPullNotFound, fetchErr)
	}

	var envelope rawShowResponse

	decodeErr := json.Unmarshal(resp.Body, &envelope)
	if decodeErr != nil || envelope.VaultID == "" || envelope.Basename == "" || envelope.ExchangeHash == "" {
		s.noteContact("")

		return rawShowResponse{}, errPullTooOld
	}

	s.noteContact(envelope.VaultID)

	return envelope, nil
}

// finish runs after every pull: a successful parent contact drains the
// outbox (design D6), behind the backoff gate.
func (s *pullSession) finish(ctx context.Context) {
	if !s.contacted || s.paused {
		return
	}

	drainForCommand(ctx, s.deps, s.vault, s.parentURL, false)
}

// noteContact records an answer from the parent: the backoff resets, and
// a reported vault ID is cached (W7 carry). Failing to record is not
// fatal to the pull.
func (s *pullSession) noteContact(vaultID string) {
	s.contacted = true

	_ = recordParentSuccess(s.store, s.vault, s.parentURL, vaultID)
}

// prepare runs the checks a child makes before its first write exchange
// in a command: the first parent contact stamps a missing vault ID (which
// creates .engram/, ruling S4), then the location check and cached
// self-parent guard (ruling S2), then the backoff gate. Each prints its
// one warning; any failure pauses the session.
func (s *pullSession) prepare() bool {
	if s.prepared {
		return !s.paused
	}

	s.prepared = true

	_, stampErr := stampVaultID(s.store.state, s.vault)
	if stampErr != nil {
		logWarningTo(s.deps.Stderr)("activate: could not stamp the vault ID: %v", stampErr)

		s.paused = true

		return false
	}

	localID, cleared, clearErr := clearedForExchange(s.store, s.vault, s.parentURL)
	if clearErr != nil {
		logWarningTo(s.deps.Stderr)("activate: %v", clearErr)
	}

	if !cleared || !gateParentContact(s.store, s.vault, s.parentURL, false) {
		s.paused = true

		return false
	}

	s.localID = localID

	return true
}

// pull pulls one ref down (design D8): fetch the raw envelope with no lock
// held, write under the lock (re-checking the skip rule), then signal use
// to the parent after the lock is released.
func (s *pullSession) pull(ctx context.Context, ref string) error {
	if !s.prepare() {
		return errPullPaused
	}

	envelope, fetchErr := s.fetchEnvelope(ctx, normalizeNoteRef(ref)+mdExt)
	if fetchErr != nil {
		return fetchErr
	}

	if selfParentGuard(s.localID, envelope.VaultID, s.deps.Stderr) {
		s.paused = true

		return errPullSelfParent
	}

	source, parseErr := parsePulledSource(envelope)
	if parseErr != nil {
		return parseErr
	}

	writeErr := s.writeUnderLock(ctx, envelope, source)

	s.signalUse(ctx, envelope.Basename)

	return writeErr
}

// queuedOffers is the outbox's queued count for the unreachable warning
// (an unreadable outbox counts as empty).
func (s *pullSession) queuedOffers() int {
	box, _ := loadOutbox(s.store.state, s.vault)

	return queuedOfferCount(box)
}

// signalUse is the best-effort parent bump (design D8 step 5, Q2): POST
// /activate for the parent note after the lock is released. Its failure is
// neither fatal nor queued, and never backs the parent off.
func (s *pullSession) signalUse(ctx context.Context, basename string) {
	//nolint:errchkjson // activateRequest is a plain []string field — never fails to encode
	body, _ := json.Marshal(activateRequest{Notes: []string{basename + mdExt}})

	_, _ = fetchRaw(ctx, s.deps, s.parentURL, methodPost, "/activate", nil, body)
}

// skipOrBump applies the skip rule under the lock: a linked local note
// means nothing is written (a live one is bumped instead), and so does a
// declined pull of the same content. It reports whether to skip.
func (s *pullSession) skipOrBump(envelope rawShowResponse, source pulledSource) (bool, error) {
	notes, scanErr := scanExchangeNotes(s.vault, s.learn.ListMD, s.deps.FS.ReadFile)
	if scanErr != nil {
		return false, fmt.Errorf("pull-down: %w", scanErr)
	}

	names := append([]string{envelope.Basename}, source.aliases...)

	linked := linkedLocalNotes(notes, envelope.VaultID, names, envelope.ExchangeHash)
	if len(linked) > 0 {
		date := s.deps.Now().Format(noteDateFormat)
		write := writeAtomicFromFS(s.deps.FS, "write sidecar")

		for _, note := range linked {
			if note.pending {
				continue
			}

			bumpErr := bumpLastUsed(embed.SidecarPath(note.path), date, s.deps.FS.ReadFile, write)
			if bumpErr != nil && !errors.Is(bumpErr, fs.ErrNotExist) {
				logWarningTo(s.deps.Stderr)("activate: %v", bumpErr)
			}
		}

		return true, nil
	}

	declined, declinedErr := loadDeclined(s.store.state, s.vault)
	if declinedErr != nil {
		return false, declinedErr
	}

	return declineMatches(declined, envelope.VaultID, names, envelope.ExchangeHash), nil
}

// warnUnreachable records a failed contact and prints the one backoff
// warning; when the failure cannot be recorded there is no retry time to
// report, so the warning names the error instead.
func (s *pullSession) warnUnreachable() {
	retry, recordErr := recordParentFailure(s.store, s.vault, s.parentURL)
	if recordErr != nil {
		logWarningTo(s.deps.Stderr)("activate: the parent is unreachable, and the backoff could not be recorded: %v",
			recordErr)

		return
	}

	_, _ = fmt.Fprintf(s.deps.Stderr, parentBackoffWarningFormat, retry.Format(time.RFC3339), s.queuedOffers())
}

// writeUnderLock takes the vault lock, re-checks the skip rule, and writes
// the pulled copy as a new pending note (design D8 step 4), embedding it
// and letting local vocab assign its tags.
func (s *pullSession) writeUnderLock(ctx context.Context, envelope rawShowResponse, source pulledSource) error {
	release, lockErr := s.learn.Lock(s.vault)
	if lockErr != nil {
		return fmt.Errorf("pull-down: acquiring vault lock: %w", lockErr)
	}
	defer release()

	skip, skipErr := s.skipOrBump(envelope, source)
	if skipErr != nil || skip {
		return skipErr
	}

	existing, listErr := s.learn.ListIDs(s.vault)
	if listErr != nil {
		return fmt.Errorf("pull-down: listing existing IDs: %w", listErr)
	}

	luhmannID, idErr := nextLuhmannID(existing, "", positionTop)
	if idErr != nil {
		return fmt.Errorf("pull-down: %w", idErr)
	}

	xid, mintErr := mintXID(s.deps.RandRead)
	if mintErr != nil {
		return fmt.Errorf("pull-down: %w", mintErr)
	}

	when := s.learn.Now()
	identity := identityStamp{Repo: s.learn.DetectRepo(ctx), User: s.learn.DetectUser(ctx), Vault: s.vaultName}

	content, buildErr := buildPulledNote(source, envelope, pulledStamp{
		luhmann: luhmannID, created: when.Format(dateFormat), identity: identity, xid: xid,
	})
	if buildErr != nil {
		return buildErr
	}

	path := learnPath(s.vault, luhmannID, pulledSlug(envelope.Basename), when)

	writeErr := s.learn.WriteNew(path, []byte(content))
	if writeErr != nil {
		return fmt.Errorf("pull-down: writing %s: %w", path, writeErr)
	}

	autoEmbedNote(ctx, s.learn, path, content)
	applyVocabAssignmentAfterLearn(s.learn, s.vault, path, content)
	s.stdout("%s\n", path)

	return nil
}

// pulledSource is a fetched parent note, parsed: its frontmatter mapping,
// its body (verbatim), its aliases and its authorship.
type pulledSource struct {
	mapping *yaml.Node
	body    string
	aliases []string
	author  noteAuthor
}

// pulledStamp is what a pulled copy gets locally: its Luhmann ID, created
// date, top-level identity and xid.
type pulledStamp struct {
	luhmann  string
	created  string
	identity identityStamp
	xid      string
}

// buildPulledNote is the pulled copy's content (design D8): the parent's
// frontmatter with the parent-side keys stripped, a local Luhmann ID,
// created date and identity, pending, a fresh xid, and a parent block
// linking the parent note (via pulled, with its exchange hash) and
// recording its authorship — followed by the parent's body verbatim. The
// copy's exchange hash must equal the parent's.
func buildPulledNote(source pulledSource, envelope rawShowResponse, stamp pulledStamp) (string, error) {
	mapping := cloneNode(source.mapping)

	deleteMappingKeys(mapping, pullStrippedKeys()...)
	setMappingValue(mapping, "luhmann", &yaml.Node{
		Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle, Value: stamp.luhmann,
	})
	setMappingValue(mapping, "created", encodeNode(stamp.created))

	if stamp.identity.Repo != "" {
		setMappingValue(mapping, "repo", encodeNode(stamp.identity.Repo))
	} else {
		deleteMappingKeys(mapping, "repo")
	}

	setMappingValue(mapping, "user", encodeNode(stamp.identity.User))
	setMappingValue(mapping, "vault", encodeNode(stamp.identity.Vault))
	setMappingValue(mapping, "pending", encodeNode(true))
	setMappingValue(mapping, xidKey, encodeNode(stamp.xid))
	setMappingValue(mapping, parentKey, encodeNode(parentLinks{
		Vault:  envelope.VaultID,
		Links:  []parentLink{{Note: envelope.Basename, Via: linkViaPulled, Hash: envelope.ExchangeHash}},
		Author: source.author,
	}))

	rendered, marshalErr := yaml.Marshal(mapping)
	if marshalErr != nil {
		return "", fmt.Errorf("pull-down: rendering frontmatter: %w", marshalErr)
	}

	content := fmStart + strings.TrimSuffix(string(rendered), "\n") + fmEnd + source.body

	copied, hashErr := exchangeHash([]byte(content))
	if hashErr != nil {
		return "", fmt.Errorf("pull-down: %w", hashErr)
	}

	if exchangeHashChanged(envelope.ExchangeHash, copied) {
		return "", fmt.Errorf("%w: %s vs %s", errPullHashMismatch, copied, envelope.ExchangeHash)
	}

	return content, nil
}

// cloneNode deep-copies a YAML node, so building a copy never mutates the
// parsed source.
func cloneNode(node *yaml.Node) *yaml.Node {
	clone := *node
	clone.Content = make([]*yaml.Node, 0, len(node.Content))

	for _, child := range node.Content {
		clone.Content = append(clone.Content, cloneNode(child))
	}

	return &clone
}

// declineMatches reports whether declined.json holds a decline, under
// parentVaultID, of one of names whose hash is not changed from hash
// (unknown counts as not changed, D3).
func declineMatches(file declinedFile, parentVaultID string, names []string, hash string) bool {
	return slices.ContainsFunc(file.Entries, func(entry declinedPull) bool {
		return entry.Vault == parentVaultID && slices.Contains(names, entry.Basename) &&
			!exchangeHashChanged(entry.Hash, hash)
	})
}

// declinedPath is <vault>/.engram/declined.json.
func declinedPath(vault string) string {
	return filepath.Join(vault, stateDirName, declinedFileName)
}

// deleteMappingKeys removes each key (and its value) from a YAML mapping.
func deleteMappingKeys(mapping *yaml.Node, keys ...string) {
	kept := make([]*yaml.Node, 0, len(mapping.Content))

	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if slices.Contains(keys, mapping.Content[index].Value) {
			continue
		}

		kept = append(kept, mapping.Content[index], mapping.Content[index+1])
	}

	mapping.Content = kept
}

// encodeNode is value as a YAML node, rendered as yaml.Marshal would.
func encodeNode(value any) *yaml.Node {
	var node yaml.Node

	_ = node.Encode(value) // strings, bools and plain structs always encode

	return &node
}

// linkedLocalNotes returns the local notes (live or pending) holding any
// link under parentVaultID to one of names whose hash is not changed from
// hash — the skip rule (design D8; unknown counts as not changed, D3).
func linkedLocalNotes(notes []exchangeNote, parentVaultID string, names []string, hash string) []exchangeNote {
	linked := make([]exchangeNote, 0, 1)

	for _, note := range notes {
		if note.exchange.Parent.Vault != parentVaultID {
			continue
		}

		if slices.ContainsFunc(note.exchange.Parent.Links, func(link parentLink) bool {
			return slices.Contains(names, link.Note) && !exchangeHashChanged(link.Hash, hash)
		}) {
			linked = append(linked, note)
		}
	}

	return linked
}

// loadDeclined reads .engram/declined.json; a missing file is empty.
func loadDeclined(state exchangeState, vault string) (declinedFile, error) {
	raw, readErr := state.fs.ReadFile(declinedPath(vault))
	if errors.Is(readErr, fs.ErrNotExist) {
		return declinedFile{Version: declinedVersion}, nil
	}

	if readErr != nil {
		return declinedFile{}, fmt.Errorf("declined pulls: %w", readErr)
	}

	var file declinedFile

	unmarshalErr := json.Unmarshal(raw, &file)
	if unmarshalErr != nil {
		return declinedFile{}, fmt.Errorf("declined pulls: %s: %w", declinedFileName, unmarshalErr)
	}

	return file, nil
}

// newPullingActivateDeps is activate's production dependencies: local
// resolution, plus the pull-down session when ENGRAM_PARENT is set.
func newPullingActivateDeps(deps Deps, args ActivateArgs) ActivateDeps {
	activate := newActivateDeps(deps)

	parentURL := parentBase(deps)
	if parentURL == "" {
		return activate
	}

	session := &pullSession{
		deps: deps, vault: args.Vault, vaultName: args.VaultName, parentURL: parentURL,
		store: outboxStoreFromDeps(deps), learn: newLearnDeps(deps),
		stdout: func(format string, values ...any) { _, _ = fmt.Fprintf(deps.Stdout, format, values...) },
	}

	activate.Pull = session.pull
	activate.Finish = session.finish

	return activate
}

// parsePulledSource parses a fetched note: it must have frontmatter and be
// a fact, feedback or runbook.
func parsePulledSource(envelope rawShowResponse) (pulledSource, error) {
	frontmatter, body, found := splitFrontmatterAndBody(envelope.Content)
	if !found {
		return pulledSource{}, errPullNoFrontmatter
	}

	noteType := peekNoteType([]byte(frontmatter))
	if noteType != typeFact && noteType != typeFeedback && noteType != typeRunbook {
		return pulledSource{}, fmt.Errorf("%w: %s is a %q note", errPullUnsupportedType, envelope.Basename, noteType)
	}

	var document yaml.Node

	parseErr := yaml.Unmarshal([]byte(frontmatter), &document)
	if parseErr != nil || len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return pulledSource{}, fmt.Errorf("pull-down: %s: unparseable frontmatter: %w", envelope.Basename, parseErr)
	}

	var probe struct {
		Repo    string   `yaml:"repo"`
		User    string   `yaml:"user"`
		Vault   string   `yaml:"vault"`
		Aliases []string `yaml:"aliases"`
	}

	_ = document.Decode(&probe) // the probe's fields are all optional strings

	return pulledSource{
		mapping: document.Content[0],
		body:    body,
		aliases: probe.Aliases,
		author:  noteAuthor{Repo: probe.Repo, User: probe.User, Vault: probe.Vault},
	}, nil
}

// pullStrippedKeys are the parent-side keys a pulled copy never carries
// (design D8): the parent's own exchange fields, registration identity,
// chunk provenance, supersedes, and its vocab tags (local vocab reassigns
// them). pending, xid and parent are then set anew.
func pullStrippedKeys() []string {
	return []string{
		parentKey, aliasesKey, offerFieldKey, xidKey, "skill_hash", "skill_key", "skill_source",
		"sources", "supersedes", "tags", "vocab_version", "pending",
	}
}

// pulledDecline is the decline a bare discard of a note records: its
// primary link, when that link is via pulled, under the note's parent
// vault ID (design D8).
func pulledDecline(raw []byte) (declinedPull, bool) {
	frontmatter, found := splitFrontmatter(raw)
	if !found {
		return declinedPull{}, false
	}

	var probe struct {
		Parent parentLinks `yaml:"parent"`
	}

	if yaml.Unmarshal(frontmatter, &probe) != nil {
		return declinedPull{}, false
	}

	for _, link := range probe.Parent.Links {
		if link.Via == linkViaOffered || link.Via == linkViaPulled {
			return declinedPull{Vault: probe.Parent.Vault, Basename: link.Note, Hash: link.Hash},
				link.Via == linkViaPulled
		}
	}

	return declinedPull{}, false
}

// pulledSlug is the slug of a parent basename (<id>.<date>.<slug>), or a
// fallback when it has none a local filename can carry.
func pulledSlug(basename string) string {
	slug := basename

	for range basenameParts - 1 {
		_, rest, found := strings.Cut(slug, ".")
		if found {
			slug = rest
		}
	}

	if validateSlug(slug) != nil {
		return pulledSlugFallback
	}

	return slug
}

// recordDeclineUnlessUnstamped records a decline (recordDeclinedPull),
// except in a vault with no parent configured and no vault ID: stamping an
// ID there would break "stamp only on serve start or first parent
// contact" (spec vault-local-first; ruling S18), and such a vault can
// never pull the note again anyway until it contacts a parent.
func recordDeclineUnlessUnstamped(
	state exchangeState, vault string, parentConfigured bool, entry declinedPull,
) error {
	if !parentConfigured {
		_, stamped, idErr := readVaultID(state, vault)
		if idErr != nil {
			return idErr
		}

		if !stamped {
			return nil
		}
	}

	return recordDeclinedPull(state, vault, entry)
}

// recordDeclinedPull adds a declined parent note to declined.json (design
// D8, H5). The caller holds the vault lock; the vault ID is stamped before
// .engram/ is created (ruling S4). A decline already recorded is kept once.
func recordDeclinedPull(state exchangeState, vault string, entry declinedPull) error {
	dirErr := ensureExchangeStateDir(state, vault)
	if dirErr != nil {
		return dirErr
	}

	file, loadErr := loadDeclined(state, vault)
	if loadErr != nil {
		return loadErr
	}

	if slices.Contains(file.Entries, entry) {
		return nil
	}

	file.Version = declinedVersion
	file.Entries = append(file.Entries, entry)

	//nolint:errchkjson // declinedFile is plain strings and ints — never fails to encode
	encoded, _ := json.Marshal(file)

	writeErr := state.fs.WriteFileAtomic(declinedPath(vault), append(encoded, '\n'), vaultFilePerm)
	if writeErr != nil {
		return fmt.Errorf("declined pulls: %w", writeErr)
	}

	return nil
}

// setMappingValue sets key's value in a YAML mapping, in place when the
// key exists, otherwise appended.
func setMappingValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			mapping.Content[index+1] = value

			return
		}
	}

	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
}
