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

// ListMD returns the .md filenames in dir. Missing dir → empty, nil. The
// untouched starter README (identified by content) is left out — this is
// the one shared point where every vault scanner (query, check, embed,
// count, ...) stops seeing it, so a freshly created vault reads as empty.
func (v *vaultFS) ListMD(dir string) ([]string, error) {
	names, err := listMDFromFS(v.fs)(dir)
	if err != nil {
		return nil, err
	}

	kept := make([]string, 0, len(names))

	for _, name := range names {
		if name == vaultReadmeFile {
			content, readErr := v.fs.ReadFile(filepath.Join(dir, name))
			if readErr == nil && isVaultStarterReadme(name, content) {
				continue
			}
		}

		kept = append(kept, name)
	}

	return kept, nil
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
