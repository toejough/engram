package cli

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"

	"go.yaml.in/yaml/v3"
)

// rawShowResponse is GET /show?raw=1's JSON envelope (design D7 H7):
// the served vault's ID, the note's current basename, the note file's
// bytes verbatim, and its exchange hash. Being JSON with a vault_id, it
// can't be confused with an old parent's rendered show output.
type rawShowResponse struct {
	VaultID      string `json:"vault_id"` //nolint:tagliatelle // design D7 fixes the envelope's snake_case keys
	Basename     string `json:"basename"`
	Content      string `json:"content"`
	ExchangeHash string `json:"exchange_hash"` //nolint:tagliatelle // design D7 fixes the envelope's snake_case keys
}

// addDedupeKeys adds a query payload's dedupe keys (design D7): the vault
// ID at top level and, on each note item, its exchange hash and non-empty
// aliases. Chunk items get none. Only the dedupe-keys request pays for the
// re-encode; a plain served query stays byte-identical to a local one.
func addDedupeKeys(deps Deps, vault string, payloadYAML []byte) ([]byte, error) {
	vaultID, idErr := stampVaultID(exchangeStateFromDeps(deps), vault)
	if idErr != nil {
		return nil, idErr
	}

	var payload queryPayload

	decodeErr := yaml.Unmarshal(payloadYAML, &payload)
	if decodeErr != nil {
		return nil, fmt.Errorf("query: dedupe keys: decoding payload: %w", decodeErr)
	}

	payload.VaultID = vaultID

	for index := range payload.Items {
		item := &payload.Items[index]
		if item.Kind == chunkItemKind {
			continue
		}

		raw, readErr := deps.FS.ReadFile(filepath.Join(vault, item.Path))
		if readErr != nil {
			return nil, fmt.Errorf("query: dedupe keys: reading %s: %w", item.Path, readErr)
		}

		hash, hashErr := exchangeHash(raw)
		if hashErr != nil {
			return nil, fmt.Errorf("query: dedupe keys: %s: %w", item.Path, hashErr)
		}

		item.ExchangeHash = hash

		if note, ok := parseExchangeNote(vault, item.Path, raw); ok {
			item.Aliases = note.exchange.Aliases
		}
	}

	var out bytes.Buffer

	encodeErr := encodeQueryPayload(&out, payload)
	if encodeErr != nil {
		return nil, encodeErr
	}

	return out.Bytes(), nil
}

// rawShowEnvelope resolves ref against the vault's note basenames, then
// against notes' aliases, and returns its raw envelope. The ref is only ever
// matched against listed names, never joined into a path, so it cannot reach
// outside the vault. A miss wraps errShowNoteNotFound.
func rawShowEnvelope(deps Deps, vault, ref string) (rawShowResponse, error) {
	name := normalizeNoteRef(ref)
	if name == "" {
		return rawShowResponse{}, errShowEmptyRef
	}

	vaultID, idErr := stampVaultID(exchangeStateFromDeps(deps), vault)
	if idErr != nil {
		return rawShowResponse{}, idErr
	}

	names, listErr := listMDFromFS(deps.FS)(vault)
	if listErr != nil {
		return rawShowResponse{}, fmt.Errorf("show: listing notes: %w", listErr)
	}

	if slices.Contains(names, name+mdExt) {
		return rawShowFor(deps, vault, vaultID, name+mdExt)
	}

	notes, scanErr := scanExchangeNotes(vault, func(string) ([]string, error) { return names, nil }, deps.FS.ReadFile)
	if scanErr != nil {
		return rawShowResponse{}, scanErr
	}

	for _, note := range notes {
		if slices.Contains(note.exchange.Aliases, name) {
			return rawShowFor(deps, vault, vaultID, note.basename+mdExt)
		}
	}

	return rawShowResponse{}, fmt.Errorf("%w: %q", errShowNoteNotFound, ref)
}

// rawShowFor reads the listed note file and builds its envelope.
func rawShowFor(deps Deps, vault, vaultID, fileName string) (rawShowResponse, error) {
	raw, readErr := deps.FS.ReadFile(filepath.Join(vault, fileName))
	if readErr != nil {
		return rawShowResponse{}, fmt.Errorf("show: read %s: %w", fileName, readErr)
	}

	hash, hashErr := exchangeHash(raw)
	if hashErr != nil {
		return rawShowResponse{}, fmt.Errorf("show: %s: %w", fileName, hashErr)
	}

	return rawShowResponse{
		VaultID:      vaultID,
		Basename:     fileName[:len(fileName)-len(mdExt)],
		Content:      string(raw),
		ExchangeHash: hash,
	}, nil
}
