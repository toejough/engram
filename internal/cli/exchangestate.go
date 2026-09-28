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

	"github.com/toejough/engram/internal/update"
)

// unexported constants.
const (
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
	errNoVaultIDToClaim  = errors.New("vault-id --claim: this vault has no vault ID yet")
	errVaultIDInvalid    = errors.New("invalid vault id (want 32 lowercase hex characters)")
	errVaultIDRereadGave = errors.New("vault id: re-read after a lost exclusive create never saw a valid id")
	errVaultIDShortRead  = errors.New("vault id: short random read")
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
	_, statErr := deps.FS.Stat(vault)
	if statErr == nil {
		return nil
	}

	if !errors.Is(statErr, fs.ErrNotExist) {
		return fmt.Errorf("vault %s: %w", vault, statErr)
	}

	initErr := initVaultFromFS(deps.FS)(vault)
	if initErr != nil {
		return initErr
	}

	_, stampErr := stampVaultID(exchangeStateFromDeps(deps), vault)
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

// mintVaultID draws vaultIDBytes from the injected random source and
// returns them hex-encoded (32 lowercase hex characters).
func mintVaultID(randRead func([]byte) (int, error)) (string, error) {
	raw := make([]byte, vaultIDBytes)

	read, readErr := randRead(raw)
	if readErr != nil {
		return "", fmt.Errorf("vault id: random source: %w", readErr)
	}

	if read != vaultIDBytes {
		return "", errVaultIDShortRead
	}

	return hex.EncodeToString(raw), nil
}

func newExchangeState(
	fsys exchangeStateFS,
	randRead func([]byte) (int, error),
	evalSymlinks func(string) (string, error),
	getwd func() (string, error),
) exchangeState {
	return exchangeState{fs: fsys, randRead: randRead, evalSymlinks: evalSymlinks, getwd: getwd}
}

// parseVaultID validates the ID file's content (trailing whitespace
// tolerated) as exactly 32 lowercase hex characters.
func parseVaultID(raw []byte) (string, error) {
	id := strings.TrimSpace(string(raw))
	if len(id) != vaultIDHexLen || strings.ToLower(id) != id {
		return "", errVaultIDInvalid
	}

	_, decodeErr := hex.DecodeString(id)
	if decodeErr != nil {
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
// the ID file exists and git (run in the vault) reports it untracked or
// uncommitted. A vault that is not a git repo (git fails) or has no ID
// file is silent.
func vaultIDUncommitted(
	ctx context.Context, vaultPath string, fileSystem update.Filesystem, cmd update.Commander,
) bool {
	_, statErr := fileSystem.Stat(filepath.Join(vaultPath, vaultIDFileName))
	if statErr != nil {
		return false
	}

	stdout, _, runErr := cmd.Run(ctx, vaultPath, "git", "status", "--porcelain", "--", vaultIDFileName)
	if runErr != nil {
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
