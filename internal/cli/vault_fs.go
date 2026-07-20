package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// osVaultFS is the LEGACY direct-os adapter satisfying vaultgraph.VaultFS.
// Deleted by the #700 purge task once amend/learn/qa/resituate/embed/vocab
// migrate to newVaultFS(d.FS). Do not add new consumers.
type osVaultFS struct{}

// ListMD returns the .md filenames in dir. Missing dir → empty, nil.
func (*osVaultFS) ListMD(dir string) ([]string, error) {
	return listDirBySuffix(os.ReadDir, dir, ".md")
}

// ReadFile reads the file at path.
func (*osVaultFS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return data, nil
}

// vaultFS adapts the injected EdgeFS to vaultgraph.VaultFS — pure
// composition, all I/O flows through the injected EdgeFS (#700).
// Listing a non-existent directory returns an empty slice (not an error) —
// the scanner uses this to skip missing subdirs (e.g. an absent MOCs/ on a
// brand-new vault).
type vaultFS struct {
	fs EdgeFS
}

// ListMD returns the .md filenames in dir. Missing dir → empty, nil.
func (v *vaultFS) ListMD(dir string) ([]string, error) {
	return listMDFromFS(v.fs)(dir)
}

// ReadFile reads the file at path.
func (v *vaultFS) ReadFile(path string) ([]byte, error) {
	data, err := v.fs.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("vault read: %w", err)
	}

	return data, nil
}

// listDirBySuffix returns the filenames directly inside dir whose name has
// the given suffix, using the injected readDir (os.ReadDir for the legacy
// adapter). Missing dir → empty, nil — matched via errors.Is so wrapped
// not-exist errors are recognized.
// listDirBySuffix serves only the legacy osVaultFS; both die together at
// the #700 purge task (T7).
func listDirBySuffix(
	readDir func(string) ([]fs.DirEntry, error),
	dir, suffix string,
) ([]string, error) {
	entries, err := readDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("list md: %w", err)
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

// newVaultFS returns a vaultgraph.VaultFS view over fsys.
func newVaultFS(fsys EdgeFS) *vaultFS {
	return &vaultFS{fs: fsys}
}
