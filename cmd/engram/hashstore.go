package main

import (
	"os"

	"engram/internal/surface"
)

// newHashStore creates a FileHashStore wired to real filesystem operations.
func newHashStore(dir string) *surface.FileHashStore {
	return surface.NewFileHashStore(dir, os.ReadFile, os.WriteFile, os.Remove)
}
