package signal

import (
	"fmt"
	"path/filepath"
)

// DirCreator ensures a directory exists.
type DirCreator func(path string) error

// FileArchiver moves memory files to an archive directory using injected I/O.
type FileArchiver struct {
	archiveDir string
	rename     Renamer
	mkdirAll   DirCreator
}

// NewFileArchiver creates a FileArchiver.
func NewFileArchiver(archiveDir string, rename Renamer, mkdirAll DirCreator) *FileArchiver {
	return &FileArchiver{archiveDir: archiveDir, rename: rename, mkdirAll: mkdirAll}
}

// Archive moves a memory file to the archive directory, preserving its name.
func (a *FileArchiver) Archive(sourcePath string) error {
	err := a.mkdirAll(a.archiveDir)
	if err != nil {
		return fmt.Errorf("creating archive dir: %w", err)
	}

	destPath := filepath.Join(a.archiveDir, filepath.Base(sourcePath))

	err = a.rename(sourcePath, destPath)
	if err != nil {
		return fmt.Errorf("archiving %s: %w", filepath.Base(sourcePath), err)
	}

	return nil
}

// Renamer moves a file from one path to another.
type Renamer func(oldpath, newpath string) error
