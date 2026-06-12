package vaultgraph

import (
	"fmt"
	"path/filepath"

	"github.com/toejough/engram/internal/luhmann"
)

// Note is a vault node: a single markdown file with its parsed metadata and outgoing wikilinks.
// LuhmannID is empty for files whose filename does not begin with a valid Luhmann ID.
type Note struct {
	Basename  string   // graph-node key, e.g. "9o1.2026-05-10.cross-cutting"
	LuhmannID string   // "9o1", or "" if the basename has no leading Luhmann ID
	IsMOC     bool     // true if the file lives in MOCs/
	Outgoing  []string // wikilink targets parsed from the body (deduped, in first-appearance order)
}

// VaultFS lists and reads markdown files in vault subdirectories. All filesystem
// access in vaultgraph goes through this interface so logic stays pure and testable.
type VaultFS interface {
	// ListMD returns the .md filenames (not paths, just basenames-with-ext) in dir.
	// Returns empty (nil error) when dir does not exist. Errors only on actual read failures.
	ListMD(dir string) ([]string, error)
	// ReadFile reads the bytes at path.
	ReadFile(path string) ([]byte, error)
}

// ScanVault reads the .md notes at the ROOT of vaultPath and returns one Note
// per file. The vault is flat: no Permanent/ or MOCs/ tiers (the chunk index
// replaced raw-event episodes; MOCs were retired earlier). Subdirectories —
// including the historical _legacy/ — are ignored.
func ScanVault(fs VaultFS, vaultPath string) ([]Note, error) {
	var notes []Note

	filenames, err := fs.ListMD(vaultPath)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", vaultPath, err)
	}

	for _, filename := range filenames {
		note, ok, scanErr := scanNote(fs, vaultPath, filename, false)
		if scanErr != nil {
			return nil, scanErr
		}

		if !ok {
			continue
		}

		notes = append(notes, note)
	}

	return notes, nil
}

func scanNote(fs VaultFS, dirPath, filename string, isMOC bool) (Note, bool, error) {
	basename, ok := ParseBasename(filename)
	if !ok {
		return Note{}, false, nil
	}

	path := filepath.Join(dirPath, filename)

	body, err := fs.ReadFile(path)
	if err != nil {
		return Note{}, false, fmt.Errorf("reading %s: %w", path, err)
	}

	luhmannID, _ := luhmann.FromBasename(basename)

	return Note{
		Basename:  basename,
		LuhmannID: luhmannID,
		IsMOC:     isMOC,
		Outgoing:  ParseWikilinks(body),
	}, true, nil
}
