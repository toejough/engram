package learnmarker

import (
	"fmt"
	"os"
)

// OSFS is the production FS, backed by package os.
type OSFS struct{}

// MkdirAll forwards to os.MkdirAll.
func (OSFS) MkdirAll(path string, perm os.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("OSFS.MkdirAll %s: %w", path, err)
	}

	return nil
}

// ReadFile forwards to os.ReadFile.
func (OSFS) ReadFile(path string) ([]byte, error) {
	//nolint:gosec // OSFS is a thin os adapter; path validation is the caller's responsibility.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("OSFS.ReadFile %s: %w", path, err)
	}

	return data, nil
}

// WriteFile forwards to os.WriteFile.
func (OSFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	err := os.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("OSFS.WriteFile %s: %w", path, err)
	}

	return nil
}
