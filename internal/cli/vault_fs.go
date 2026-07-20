package cli

import (
	"fmt"
	"path/filepath"
)

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

// newVaultFS returns a vaultgraph.VaultFS view over fsys.
func newVaultFS(fsys EdgeFS) *vaultFS {
	return &vaultFS{fs: fsys}
}
