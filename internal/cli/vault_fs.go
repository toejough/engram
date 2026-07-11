package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// osVaultFS is the production adapter satisfying vaultgraph.VaultFS. Listing a
// non-existent directory returns an empty slice (not an error) — the scanner
// uses this to skip missing subdirs (e.g. an absent MOCs/ on a brand-new vault).
type osVaultFS struct{}

// ListMD returns the .md filenames in dir. Missing dir → empty, nil.
func (*osVaultFS) ListMD(dir string) ([]string, error) {
	return listDirBySuffix(dir, ".md")
}

// ListVecJSON returns the .vec.json filenames in dir. Missing dir → empty,
// nil. Used by vocab migrate-tags to sweep hub sidecars — including any
// orphan with no surviving .md counterpart — found by listing directly
// rather than deriving paths from the .md listing.
func (*osVaultFS) ListVecJSON(dir string) ([]string, error) {
	return listDirBySuffix(dir, ".vec.json")
}

// ReadFile reads the file at path.
func (*osVaultFS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return data, nil
}

// listDirBySuffix returns the filenames directly inside dir whose name has
// the given suffix. Missing dir → empty, nil. Shared by ListMD and
// ListVecJSON — the only difference between the two is the suffix filtered.
func listDirBySuffix(dir, suffix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading dir %s: %w", dir, err)
	}

	out := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}

		out = append(out, entry.Name())
	}

	return out, nil
}
