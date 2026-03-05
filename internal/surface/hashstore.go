package surface

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileHashStore implements BlockHashStore using injected filesystem operations.
type FileHashStore struct {
	dir       string
	readFile  func(string) ([]byte, error)
	writeFile func(string, []byte, os.FileMode) error
	remove    func(string) error
}

// NewFileHashStore creates a FileHashStore with the given data directory
// and injected I/O functions.
func NewFileHashStore(
	dir string,
	readFile func(string) ([]byte, error),
	writeFile func(string, []byte, os.FileMode) error,
	remove func(string) error,
) *FileHashStore {
	return &FileHashStore{
		dir:       dir,
		readFile:  readFile,
		writeFile: writeFile,
		remove:    remove,
	}
}

// ClearHash removes the stored hash file. Returns nil if the file does not exist.
func (f *FileHashStore) ClearHash(_ context.Context) error {
	err := f.remove(filepath.Join(f.dir, hashStoreFile))
	if os.IsNotExist(err) {
		return nil
	}

	return err
}

// LastHash reads the last stored hash. Returns empty string if no hash is stored.
func (f *FileHashStore) LastHash(_ context.Context) (string, error) {
	data, err := f.readFile(filepath.Join(f.dir, hashStoreFile))
	if os.IsNotExist(err) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("reading hash: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// SaveHash persists a hash to the store file.
func (f *FileHashStore) SaveHash(_ context.Context, hash string) error {
	err := f.writeFile(
		filepath.Join(f.dir, hashStoreFile),
		[]byte(hash+"\n"),
		hashFilePermission,
	)
	if err != nil {
		return fmt.Errorf("saving hash: %w", err)
	}

	return nil
}

// unexported constants.
const (
	hashFilePermission = 0o600
	hashStoreFile      = ".last-blocked-hash"
)
