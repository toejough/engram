package vaultgraph

import (
	"fmt"
	"path/filepath"
)

// Vault subdirectory names. Filenames within them are <luhmann-id>.<YYYY-MM-DD>.<slug>.md
// for promoted notes; fleetings may have arbitrary names but must end in .md.
const (
	mocsSubdir      = "MOCs"
	permanentSubdir = "Permanent"
	fleetingSubdir  = "Fleeting"
)

// VaultFS lists and reads markdown files in vault subdirectories. All filesystem
// access in vaultgraph goes through this interface so logic stays pure and testable.
type VaultFS interface {
	// ListMD returns the .md filenames (not paths, just basenames-with-ext) in dir.
	// Returns empty (nil error) when dir does not exist. Errors only on actual read failures.
	ListMD(dir string) ([]string, error)
	// ReadFile reads the bytes at path.
	ReadFile(path string) ([]byte, error)
}

// Note is a vault node: a single markdown file with its parsed metadata and outgoing wikilinks.
// LuhmannID is empty for files (e.g. fleetings) whose filename does not begin with a valid ID.
type Note struct {
	Basename  string   // graph-node key, e.g. "9o1.2026-05-10.cross-cutting"
	LuhmannID string   // "9o1", or "" if the basename has no leading Luhmann ID
	IsMOC     bool     // true if the file lives in MOCs/
	Outgoing  []string // wikilink targets parsed from the body (deduped, in first-appearance order)
}

// ScanVault reads MOCs/, Permanent/, and Fleeting/ under vaultPath and returns one Note per .md file.
// Missing subdirs are silently skipped. The returned slice has stable order: MOCs first by Basename,
// then Permanent, then Fleeting — but downstream consumers should not rely on this and instead
// look up by Basename.
func ScanVault(fs VaultFS, vaultPath string) ([]Note, error) {
	subdirs := []struct {
		name  string
		isMOC bool
	}{
		{mocsSubdir, true},
		{permanentSubdir, false},
		{fleetingSubdir, false},
	}

	var notes []Note

	for _, sub := range subdirs {
		dirPath := filepath.Join(vaultPath, sub.name)

		filenames, err := fs.ListMD(dirPath)
		if err != nil {
			return nil, fmt.Errorf("listing %s: %w", dirPath, err)
		}

		for _, filename := range filenames {
			note, ok, scanErr := scanNote(fs, dirPath, filename, sub.isMOC)
			if scanErr != nil {
				return nil, scanErr
			}

			if !ok {
				continue
			}

			notes = append(notes, note)
		}
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

	luhmannID, _ := LuhmannFromBasename(basename)

	return Note{
		Basename:  basename,
		LuhmannID: luhmannID,
		IsMOC:     isMOC,
		Outgoing:  ParseWikilinks(body),
	}, true, nil
}
