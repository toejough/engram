package cli

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// VaultInitFS is the filesystem surface needed to bootstrap a fresh vault.
// MkdirAll must be MkdirAll-shaped (creates parents, no-op if exists).
// WriteFileIfMissing must write the file only when it does not already
// exist; existing files are left untouched (idempotent bootstrap).
type VaultInitFS interface {
	MkdirAll(path string, perm fs.FileMode) error
	WriteFileIfMissing(path string, data []byte, perm fs.FileMode) error
}

// unexported constants.
const (
	readmeBody = `# Engram vault

This directory is an engram zettelkasten vault. Notes live under:

- atomic notes (one principle per file) live at the vault root

Engram's ` + "`recall`" + ` and ` + "`learn`" + ` skills read and write this directory.
The ` + "`.obsidian/`" + ` directory lets Obsidian open this vault directly.
`
	vaultDirPerm  fs.FileMode = 0o755
	vaultFilePerm fs.FileMode = 0o644
)

// vaultStarterFile is one (path, body) tuple in the bootstrap content set.
type vaultStarterFile struct {
	relPath string
	body    string
}

// initializeVault creates the standard layout under vaultPath: Permanent/,
// a minimal .obsidian/app.json so Obsidian recognizes the directory as a
// vault, a .gitignore, and a short README. All file writes are
// skip-if-exists; calling on an existing vault is a safe no-op (e.g.,
// re-runs after partial initialization).
func initializeVault(vaultFS VaultInitFS, vaultPath string) error {
	for _, sub := range []string{".obsidian"} {
		mkErr := vaultFS.MkdirAll(filepath.Join(vaultPath, sub), vaultDirPerm)
		if mkErr != nil {
			return fmt.Errorf("initialize vault: mkdir %s: %w", sub, mkErr)
		}
	}

	for _, starter := range vaultStarters() {
		writeErr := vaultFS.WriteFileIfMissing(
			filepath.Join(vaultPath, starter.relPath),
			[]byte(starter.body),
			vaultFilePerm,
		)
		if writeErr != nil {
			return fmt.Errorf("initialize vault: write %s: %w", starter.relPath, writeErr)
		}
	}

	return nil
}

// vaultStarters returns the starter content set written by initializeVault.
// The minimal .obsidian/app.json lets Obsidian recognize the directory as
// a vault on first open without engram having to ship a full Obsidian
// config. The .gitignore keeps lock/workspace files out of git for users
// who choose to track their vault.
func vaultStarters() []vaultStarterFile {
	return []vaultStarterFile{
		{".obsidian/app.json", "{}\n"},
		{".gitignore", ".luhmann.lock\n.obsidian/workspace*\n.obsidian/cache\n"},
		{"README.md", readmeBody},
	}
}
