package parser

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// RealFS implements CollectableFS using the real file system.
type RealFS struct{}

// NewRealFS creates a new RealFS instance.
func NewRealFS() *RealFS {
	return &RealFS{}
}

// DirExists returns true if the directory exists.
func (r *RealFS) DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info == nil {
		return false
	}

	return info.IsDir()
}

// FileExists returns true if the file exists.
func (r *RealFS) FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info == nil {
		return false
	}

	return !info.IsDir()
}

// ReadFile reads the file content as a string.
func (r *RealFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Walk traverses the directory tree.
func (r *RealFS) Walk(root string, fn func(path string, isDir bool) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		walkErr := fn(path, d.IsDir())
		if walkErr != nil {
			if errors.Is(walkErr, errSkipDir) {
				return fs.SkipDir
			}

			return walkErr
		}

		return nil
	})
}
