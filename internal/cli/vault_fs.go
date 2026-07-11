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

// ReadFile reads the file at path.
func (*osVaultFS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return data, nil
}

// listDirBySuffix returns the filenames directly inside dir whose name has
// the given suffix. Missing dir → empty, nil. Used by ListMD to filter to
// ".md" files.
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
