package cli

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/toejough/engram/internal/luhmann"
	"github.com/toejough/engram/internal/update"
)

// unexported constants.
const (
	// backoffBase is the first backoff window after a failed parent
	// contact; each consecutive failure doubles it up to backoffCap
	// (design D6, M9).
	backoffBase = 30 * time.Second
	backoffCap  = 15 * time.Minute
	// backoffMaxShift is the doubling count past which the window is
	// always the cap (30s << 5 = 16m > 15m), so the shift never overflows.
	backoffMaxShift = 5
	// createdVaultFormat is ensureVault's single stderr line for a vault it
	// just created — it makes a mistyped --vault/ENGRAM_VAULT_PATH visible
	// (#766's typo concern).
	createdVaultFormat = "engram: created new vault at %s\n"
	homeRecordFile     = "home.json"
	locationMismatch   = "mismatch"
	locationMissing    = "missing"
	locationNoID       = "no vault id"
	locationOK         = "ok"
	// locationWarningFormat is the one warning a failed location check
	// prints (child-side exchange pauses; serve still starts). It names both
	// remedies so the user can pick the one matching what happened.
	locationWarningFormat = "engram: warning: vault %s failed its location check " +
		"(%s — a copied, moved or cloned vault?); exchange with the parent is paused; " +
		"if this vault is a copy, run `engram vault-id --regenerate`; " +
		"if it is the same vault moved or re-cloned, run `engram vault-id --claim`\n"
	// parentBackoffWarningFormat is the one warning a command prints when
	// the backoff window makes it skip the parent.
	parentBackoffWarningFormat = "engram: parent unreachable (retry after %s); %d offer(s) queued\n"
	parentCacheFile            = "parent.json"
	// repairedVaultFormat is repairVaultID's single stderr line.
	repairedVaultFormat = "engram: stamped the missing vault ID of %s\n"
	// selfParentWarningFormat is the self-parent guard's one warning.
	selfParentWarningFormat = "engram: warning: the parent reports this vault's own ID (%s) — skipping exchange " +
		"with it; if this vault is a copy of the parent, run `engram vault-id --regenerate`\n"
	stateDirName       = ".engram"
	stateGitignore     = "*\n"
	stateGitignoreFile = ".gitignore"
	vaultIDBytes       = 16
	vaultIDFileName    = ".engram-vault-id"
	vaultIDHexLen      = 32
	vaultIDRereadLimit = 1000
)

// unexported variables.
var (
	errNoVaultIDToClaim = errors.New("vault-id --claim: this vault has no vault ID yet")
	errShortRandomRead  = errors.New("short random read")
	errVaultIDInvalid   = errors.New("invalid vault id: .engram-vault-id must hold 32 lowercase hex characters " +
		"(empty or corrupt after a partial write or a git conflict?) — run `engram vault-id --regenerate`")
	errVaultIDRereadGave = errors.New("vault id: .engram-vault-id is empty or corrupt " +
		"(a partial write or a git conflict?) — run `engram vault-id --regenerate`")
)

// exchangeState is the exchange-state adapter: the vault-ID file, the
// .engram/ state directory and its home.json location record, over
// injected capabilities only.
type exchangeState struct {
	fs           exchangeStateFS
	randRead     func([]byte) (int, error)
	evalSymlinks func(string) (string, error)
	getwd        func() (string, error)
}

// exchangeStateFS is the narrow filesystem surface the adapter needs
// (EdgeFS satisfies it).
type exchangeStateFS interface {
	ReadFile(path string) ([]byte, error)
	WriteFileExcl(path string, data []byte, perm fs.FileMode) error
	WriteFileAtomic(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

// homeRecord is .engram/home.json: the vault ID and its canonical path.
// No hostname, by design (r4 H-A).
type homeRecord struct {
	VaultID string `json:"vault_id"` //nolint:tagliatelle // design D2 fixes home.json's snake_case keys
	Path    string `json:"path"`
}

// parentCache is .engram/parent.json: the configured parent's URL, its
// reported vault ID, and the backoff state (design D2, D6). The backoff
// belongs to the URL that failed: a cache for another URL is ignored.
type parentCache struct {
	URL          string    `json:"url"`
	VaultID      string    `json:"vault_id,omitempty"`     //nolint:tagliatelle // design D2 fixes parent.json's keys
	BackoffUntil time.Time `json:"backoff_until,omitzero"` //nolint:tagliatelle // design D2 fixes parent.json's keys
	Failures     int       `json:"failures"`
}

// backoffDelay is the backoff window after the given number of
// consecutive failures: 30s doubling per failure, capped at 15 minutes.
func backoffDelay(failures int) time.Duration {
	if failures < 1 {
		return 0
	}

	shift := failures - 1
	if shift >= backoffMaxShift {
		return backoffCap
	}

	return min(backoffBase<<shift, backoffCap)
}

// canonicalVaultPath returns EvalSymlinks(Abs(Clean(vault))), with Abs
// composed from the injected working directory.
func canonicalVaultPath(state exchangeState, vault string) (string, error) {
	cleaned := filepath.Clean(vault)

	if !filepath.IsAbs(cleaned) {
		cwd, cwdErr := state.getwd()
		if cwdErr != nil {
			return "", fmt.Errorf("canonical vault path: working dir: %w", cwdErr)
		}

		cleaned = filepath.Join(cwd, cleaned)
	}

	resolved, evalErr := state.evalSymlinks(cleaned)
	if evalErr != nil {
		return "", fmt.Errorf("canonical vault path: %w", evalErr)
	}

	return resolved, nil
}

// checkVaultLocation compares home.json with the current vault ID and
// canonical path: locationOK, locationMissing (no record — a git clone),
// locationMismatch (a cp -R or a move), or locationNoID.
func checkVaultLocation(state exchangeState, vault string) (string, error) {
	id, found, idErr := readVaultID(state, vault)
	if idErr != nil {
		return "", idErr
	}

	if !found {
		return locationNoID, nil
	}

	raw, readErr := state.fs.ReadFile(filepath.Join(vault, stateDirName, homeRecordFile))
	if errors.Is(readErr, fs.ErrNotExist) {
		return locationMissing, nil
	}

	if readErr != nil {
		return "", fmt.Errorf("vault location record: %w", readErr)
	}

	var record homeRecord

	unmarshalErr := json.Unmarshal(raw, &record)
	if unmarshalErr != nil {
		return locationMismatch, nil //nolint:nilerr // an unparseable record is a mismatch, not a failure
	}

	canonical, canonErr := canonicalVaultPath(state, vault)
	if canonErr != nil {
		return "", canonErr
	}

	if record.VaultID != id || record.Path != canonical {
		return locationMismatch, nil
	}

	return locationOK, nil
}

// claimVaultLocation (`engram vault-id --claim`) rewrites only home.json for
// the current ID and path: this is the same vault, moved or re-cloned.
func claimVaultLocation(state exchangeState, vault string) (string, error) {
	id, found, idErr := readVaultID(state, vault)
	if idErr != nil {
		return "", idErr
	}

	if !found {
		return "", errNoVaultIDToClaim
	}

	writeErr := writeLocationRecord(state, vault, id)
	if writeErr != nil {
		return "", writeErr
	}

	return id, nil
}

// ensureExchangeStateDir readies .engram/ for an exchange-state writer
// (outbox, parent cache): the vault ID is stamped first, so .engram/ keeps
// meaning created-by-engram (ruling S4), then the directory is ensured.
func ensureExchangeStateDir(state exchangeState, vault string) error {
	_, stampErr := stampVaultID(state, vault)
	if stampErr != nil {
		return stampErr
	}

	return ensureStateDir(state, vault)
}

// ensureStateDir creates <vault>/.engram/ and its self-ignoring .gitignore
// (written only when missing). The vault's tracked root .gitignore is never
// touched (G12).
func ensureStateDir(state exchangeState, vault string) error {
	dir := filepath.Join(vault, stateDirName)

	mkErr := state.fs.MkdirAll(dir, vaultDirPerm)
	if mkErr != nil {
		return fmt.Errorf("vault state dir: %w", mkErr)
	}

	writeErr := state.fs.WriteFileExcl(filepath.Join(dir, stateGitignoreFile), []byte(stateGitignore), vaultFilePerm)
	if writeErr != nil && !errors.Is(writeErr, fs.ErrExist) {
		return fmt.Errorf("vault state dir: .gitignore: %w", writeErr)
	}

	return nil
}

// ensureVault is the shared first-use step every vault-resolving command
// runs in dispatch (design D2, #766): a missing vault is created (starters,
// vault ID, .engram/ and its location record) and announced with one stderr
// line. An existing vault is left completely untouched — read-only use never
// creates the ID (serve and first parent contact stamp it lazily).
func ensureVault(deps Deps, vault string) error {
	info, statErr := deps.FS.Stat(vault)
	if statErr == nil {
		if !info.IsDir() {
			return fmt.Errorf("vault %s: %w", vault, errNotADirectory)
		}

		return repairVaultID(deps, vault)
	}

	if !errors.Is(statErr, fs.ErrNotExist) {
		return fmt.Errorf("vault %s: %w", vault, statErr)
	}

	initErr := initVaultFromFS(deps.FS)(vault)
	if initErr != nil {
		return initErr
	}

	// .engram/ first: it marks the vault as created-by-engram, so a stamp
	// that fails below is repaired by the next run (repairVaultID).
	state := exchangeStateFromDeps(deps)

	dirErr := ensureStateDir(state, vault)
	if dirErr != nil {
		return dirErr
	}

	_, stampErr := stampVaultID(state, vault)
	if stampErr != nil {
		return stampErr
	}

	_, _ = fmt.Fprintf(deps.Stderr, createdVaultFormat, vault)

	return nil
}

// exchangeStateFromDeps composes the adapter over the production carrier.
func exchangeStateFromDeps(deps Deps) exchangeState {
	return newExchangeState(deps.FS, deps.RandRead, deps.EvalSymlinks, deps.Getwd)
}

// gateParentContact decides whether a command may contact the parent now.
// Inside the backoff window it prints the one warning (retry time and
// queued-offer count) and returns false; `engram update` passes
// ignoreBackoff and always tries (design D6).
func gateParentContact(store outboxStore, vault, parentURL string, ignoreBackoff bool) bool {
	if ignoreBackoff {
		return true
	}

	cache, cacheErr := loadParentCache(store.state, vault)
	if cacheErr != nil || cache.URL != parentURL || !store.now().Before(cache.BackoffUntil) {
		return true
	}

	queued := 0

	box, boxErr := loadOutbox(store.state, vault)
	if boxErr == nil {
		queued = queuedOfferCount(box)
	}

	_, _ = fmt.Fprintf(store.stderr, parentBackoffWarningFormat, cache.BackoffUntil.Format(time.RFC3339), queued)

	return false
}

// isExchangeBasename reports whether name is a note basename another vault
// may hand this one (final review F6): a Luhmann basename with no path
// separator and no '|' (which would corrupt the note|type|claim supersedes
// encoding). Parent-supplied basenames are stored in frontmatter,
// declined.json and offers, so anything else is a malformed reply.
func isExchangeBasename(name string) bool {
	if strings.ContainsAny(name, `/\|`) {
		return false
	}

	_, isBasename := luhmann.FromBasename(name)

	return isBasename
}

// isExchangeID reports whether id is an exchange identifier — a vault ID
// or an xid: exactly 32 lowercase hex characters.
func isExchangeID(id string) bool {
	if len(id) != vaultIDHexLen || strings.ToLower(id) != id {
		return false
	}

	_, decodeErr := hex.DecodeString(id)

	return decodeErr == nil
}

// loadParentCache reads .engram/parent.json; a missing file is an empty
// cache.
func loadParentCache(state exchangeState, vault string) (parentCache, error) {
	raw, readErr := state.fs.ReadFile(parentCachePath(vault))
	if errors.Is(readErr, fs.ErrNotExist) {
		return parentCache{}, nil
	}

	if readErr != nil {
		return parentCache{}, fmt.Errorf("parent cache: %w", readErr)
	}

	var cache parentCache

	unmarshalErr := json.Unmarshal(raw, &cache)
	if unmarshalErr != nil {
		return parentCache{}, fmt.Errorf("parent cache: %s: %w", parentCacheFile, unmarshalErr)
	}

	return cache, nil
}

// lockedUpdateParentCache applies change to parent.json under the vault
// lock, writing only when the cache changed.
func lockedUpdateParentCache(
	store outboxStore, vault string, change func(parentCache) parentCache,
) (parentCache, error) {
	unlock, lockErr := store.lock(vault)
	if lockErr != nil {
		return parentCache{}, fmt.Errorf("parent cache: %w", lockErr)
	}

	defer unlock()

	cache, loadErr := loadParentCache(store.state, vault)
	if loadErr != nil {
		return parentCache{}, loadErr
	}

	updated := change(cache)
	if updated == cache {
		return cache, nil
	}

	return updated, saveParentCache(store.state, vault, updated)
}

// mintExchangeID draws vaultIDBytes from the injected random source and
// returns them hex-encoded (32 lowercase hex characters); label names the
// identifier in errors.
func mintExchangeID(randRead func([]byte) (int, error), label string) (string, error) {
	raw := make([]byte, vaultIDBytes)

	read, readErr := randRead(raw)
	if readErr != nil {
		return "", fmt.Errorf("%s: random source: %w", label, readErr)
	}

	if read != vaultIDBytes {
		return "", fmt.Errorf("%s: %w", label, errShortRandomRead)
	}

	return hex.EncodeToString(raw), nil
}

// mintVaultID mints a new vault ID (design D2).
func mintVaultID(randRead func([]byte) (int, error)) (string, error) {
	return mintExchangeID(randRead, "vault id")
}

// mintXID mints a note's exchange ID (design D4).
func mintXID(randRead func([]byte) (int, error)) (string, error) {
	return mintExchangeID(randRead, "xid")
}

func newExchangeState(
	fsys exchangeStateFS,
	randRead func([]byte) (int, error),
	evalSymlinks func(string) (string, error),
	getwd func() (string, error),
) exchangeState {
	return exchangeState{fs: fsys, randRead: randRead, evalSymlinks: evalSymlinks, getwd: getwd}
}

// noteParentFailure is the cache after one more consecutive failure against
// parentURL: the window grows (a cache for another URL starts over).
func noteParentFailure(cache parentCache, parentURL string, now time.Time) parentCache {
	if cache.URL != parentURL {
		cache = parentCache{URL: parentURL}
	}

	cache.Failures++
	cache.BackoffUntil = now.Add(backoffDelay(cache.Failures))

	return cache
}

// noteParentSuccess is the cache after a successful contact with
// parentURL: the failure count and window reset, and a reported vault ID
// is remembered.
func noteParentSuccess(cache parentCache, parentURL, vaultID string) parentCache {
	if cache.URL != parentURL {
		cache = parentCache{URL: parentURL}
	}

	cache.Failures = 0
	cache.BackoffUntil = time.Time{}

	// Only a well-formed vault ID is remembered: the parent reports it, so
	// a malformed one is ignored (final review F6).
	if isExchangeID(vaultID) {
		cache.VaultID = vaultID
	}

	return cache
}

// parentCachePath is <vault>/.engram/parent.json.
func parentCachePath(vault string) string {
	return filepath.Join(vault, stateDirName, parentCacheFile)
}

// parseVaultID validates the ID file's content (trailing whitespace
// tolerated) as exactly 32 lowercase hex characters.
func parseVaultID(raw []byte) (string, error) {
	id := strings.TrimSpace(string(raw))
	if !isExchangeID(id) {
		return "", errVaultIDInvalid
	}

	return id, nil
}

// readVaultID reads .engram-vault-id: found=false (no error) when absent.
func readVaultID(state exchangeState, vault string) (string, bool, error) {
	raw, readErr := state.fs.ReadFile(filepath.Join(vault, vaultIDFileName))
	if errors.Is(readErr, fs.ErrNotExist) {
		return "", false, nil
	}

	if readErr != nil {
		return "", false, fmt.Errorf("reading %s: %w", vaultIDFileName, readErr)
	}

	id, parseErr := parseVaultID(raw)
	if parseErr != nil {
		return "", false, fmt.Errorf("%s: %w", vaultIDFileName, parseErr)
	}

	return id, true, nil
}

// recordParentFailure records a failed contact (a transport error,
// timeout or 5xx) and returns the retry time.
func recordParentFailure(store outboxStore, vault, parentURL string) (time.Time, error) {
	now := store.now()

	cache, err := lockedUpdateParentCache(store, vault, func(cache parentCache) parentCache {
		return noteParentFailure(cache, parentURL, now)
	})

	return cache.BackoffUntil, err
}

// recordParentSuccess records a successful contact: the backoff resets and
// the parent's reported vault ID (when known) is cached. An idle success
// writes nothing.
func recordParentSuccess(store outboxStore, vault, parentURL, vaultID string) error {
	_, err := lockedUpdateParentCache(store, vault, func(cache parentCache) parentCache {
		return noteParentSuccess(cache, parentURL, vaultID)
	})

	return err
}

// regenerateVaultID (`engram vault-id --regenerate`) mints a new ID and
// rewrites .engram-vault-id and home.json: this vault is a copy. No note is
// touched.
func regenerateVaultID(state exchangeState, vault string) (string, error) {
	id, mintErr := mintVaultID(state.randRead)
	if mintErr != nil {
		return "", mintErr
	}

	writeErr := state.fs.WriteFileAtomic(filepath.Join(vault, vaultIDFileName), []byte(id+"\n"), vaultFilePerm)
	if writeErr != nil {
		return "", fmt.Errorf("vault-id --regenerate: %w", writeErr)
	}

	recordErr := writeLocationRecord(state, vault, id)
	if recordErr != nil {
		return "", recordErr
	}

	return id, nil
}

// repairVaultID finishes a vault a failed first-use stamp left half-created:
// it has the .engram/ state dir (created before the ID) but no ID file.
// Every other existing vault — including a pre-existing vault with neither
// — is left untouched, so read-only use never creates the ID.
func repairVaultID(deps Deps, vault string) error {
	_, idErr := deps.FS.Stat(filepath.Join(vault, vaultIDFileName))
	if !errors.Is(idErr, fs.ErrNotExist) {
		return nil
	}

	_, dirErr := deps.FS.Stat(filepath.Join(vault, stateDirName))
	if dirErr != nil {
		return nil //nolint:nilerr // no state dir: a pre-existing vault, not a half-created one
	}

	_, stampErr := stampVaultID(exchangeStateFromDeps(deps), vault)
	if stampErr != nil {
		return stampErr
	}

	_, _ = fmt.Fprintf(deps.Stderr, repairedVaultFormat, vault)

	return nil
}

// rereadWinningVaultID re-reads the ID file after an exclusive create lost
// the race. The winner may still be mid-write (created but empty), so an
// invalid read is retried a bounded number of times.
func rereadWinningVaultID(state exchangeState, vault string) (string, error) {
	for range vaultIDRereadLimit {
		raw, readErr := state.fs.ReadFile(filepath.Join(vault, vaultIDFileName))
		if readErr != nil {
			return "", fmt.Errorf("re-reading %s: %w", vaultIDFileName, readErr)
		}

		id, parseErr := parseVaultID(raw)
		if parseErr == nil {
			return id, nil
		}
	}

	return "", errVaultIDRereadGave
}

// saveParentCache writes parent.json atomically, stamping the vault ID
// before .engram/ is created (ruling S4).
func saveParentCache(state exchangeState, vault string, cache parentCache) error {
	dirErr := ensureExchangeStateDir(state, vault)
	if dirErr != nil {
		return dirErr
	}

	encoded, marshalErr := json.Marshal(cache)
	if marshalErr != nil {
		return fmt.Errorf("parent cache: %w", marshalErr)
	}

	writeErr := state.fs.WriteFileAtomic(parentCachePath(vault), append(encoded, '\n'), vaultFilePerm)
	if writeErr != nil {
		return fmt.Errorf("parent cache: %w", writeErr)
	}

	return nil
}

// selfParentGuard reports whether the parent's reported vault ID is this
// vault's own (non-empty) ID. When it is, the caller must do no exchange
// (no merge, offer or pull-down); the guard prints the one warning naming
// `engram vault-id --regenerate`.
func selfParentGuard(localID, parentID string, stderr io.Writer) bool {
	if localID == "" || localID != parentID {
		return false
	}

	_, _ = fmt.Fprintf(stderr, selfParentWarningFormat, localID)

	return true
}

// stampVaultID returns the vault's ID, creating it when absent: the ID is
// written with an exclusive create and then re-read, so concurrent
// creators all converge on the single ID that won. Only the winner creates
// .engram/ and the location record.
func stampVaultID(state exchangeState, vault string) (string, error) {
	_, readErr := state.fs.ReadFile(filepath.Join(vault, vaultIDFileName))
	if readErr == nil {
		// Present (possibly still being written by a racing creator):
		// converge on its content.
		return rereadWinningVaultID(state, vault)
	}

	if !errors.Is(readErr, fs.ErrNotExist) {
		return "", fmt.Errorf("reading %s: %w", vaultIDFileName, readErr)
	}

	minted, mintErr := mintVaultID(state.randRead)
	if mintErr != nil {
		return "", mintErr
	}

	createErr := state.fs.WriteFileExcl(filepath.Join(vault, vaultIDFileName), []byte(minted+"\n"), vaultFilePerm)
	if errors.Is(createErr, fs.ErrExist) {
		return rereadWinningVaultID(state, vault)
	}

	if createErr != nil {
		return "", fmt.Errorf("vault id: %w", createErr)
	}

	winner, rereadErr := rereadWinningVaultID(state, vault)
	if rereadErr != nil {
		return "", rereadErr
	}

	recordErr := writeLocationRecord(state, vault, winner)
	if recordErr != nil {
		return "", recordErr
	}

	return winner, nil
}

// vaultIDUncommitted is `engram update`'s notify-only detector: true when
// the ID file exists in a git work tree and is not in the index (untracked
// or ignored) or is staged/modified but not committed. A vault that is not a git repo (git fails) or has no ID
// file is silent.
func vaultIDUncommitted(
	ctx context.Context, vaultPath string, fileSystem update.Filesystem, cmd update.Commander,
) bool {
	_, statErr := fileSystem.Stat(filepath.Join(vaultPath, vaultIDFileName))
	if statErr != nil {
		return false
	}

	_, _, repoErr := cmd.Run(ctx, vaultPath, "git", "rev-parse", "--is-inside-work-tree")
	if repoErr != nil {
		return false
	}

	// Not in the index at all — untracked, or ignored (which `git status`
	// alone would hide).
	_, _, trackedErr := cmd.Run(ctx, vaultPath, "git", "ls-files", "--error-unmatch", "--", vaultIDFileName)
	if trackedErr != nil {
		return true
	}

	// In the index but staged or modified relative to HEAD.
	stdout, _, statusErr := cmd.Run(ctx, vaultPath, "git", "status", "--porcelain", "--", vaultIDFileName)
	if statusErr != nil {
		return false
	}

	return len(bytes.TrimSpace(stdout)) > 0
}

// warnVaultLocation runs the location check and, when it fails, prints the
// one warning naming both remedies. It returns true when the check passed.
// A check that cannot run (I/O error) is reported the same way.
func warnVaultLocation(state exchangeState, vault string, stderr io.Writer) bool {
	status, checkErr := checkVaultLocation(state, vault)
	if checkErr != nil {
		status = checkErr.Error()
	}

	if status == locationOK {
		return true
	}

	_, _ = fmt.Fprintf(stderr, locationWarningFormat, vault, status)

	return false
}

// writeLocationRecord writes .engram/home.json = {vault_id, path} (creating
// the state dir first), atomically.
func writeLocationRecord(state exchangeState, vault, id string) error {
	dirErr := ensureStateDir(state, vault)
	if dirErr != nil {
		return dirErr
	}

	canonical, canonErr := canonicalVaultPath(state, vault)
	if canonErr != nil {
		return canonErr
	}

	encoded, marshalErr := json.Marshal(homeRecord{VaultID: id, Path: canonical})
	if marshalErr != nil {
		return fmt.Errorf("vault location record: %w", marshalErr)
	}

	writeErr := state.fs.WriteFileAtomic(
		filepath.Join(vault, stateDirName, homeRecordFile), append(encoded, '\n'), vaultFilePerm)
	if writeErr != nil {
		return fmt.Errorf("vault location record: %w", writeErr)
	}

	return nil
}
