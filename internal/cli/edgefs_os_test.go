package cli_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// osTestEdgeFS is a real-filesystem cli.EdgeFS for integration-style tests
// over t.TempDir() fixtures. The production EdgeFS impl is internal/cli's
// unexported primFS (composed by cli.NewDeps — T1-rework); test files are
// exempt from the internal/ purity rules, so this double calls os directly.
// Errors are wrapped with %w, matching the production contract that
// errors.Is(err, fs.ErrNotExist) / errors.Is(err, fs.ErrExist) must
// survive the adapter.
// WriteFileAtomic is a plain write — ADR-0013 atomicity is exercised by
// the internal edgefs/primitives integration suites (T1-rework), never
// through this double.
// WriteFileExcl is a real exclusive create (O_CREATE|O_EXCL) matching
// T3's EdgeFS addition.
type osTestEdgeFS struct{}

func (osTestEdgeFS) MkdirAll(path string, perm fs.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("test edgefs: mkdir %s: %w", path, err)
	}

	return nil
}

func (osTestEdgeFS) MkdirTemp(dir, pattern string) (string, error) {
	path, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("test edgefs: mkdtemp %s: %w", dir, err)
	}

	return path, nil
}

func (osTestEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("test edgefs: readdir %s: %w", path, err)
	}

	return entries, nil
}

func (osTestEdgeFS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("test edgefs: reading %s: %w", path, err)
	}

	return data, nil
}

func (osTestEdgeFS) Remove(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("test edgefs: remove %s: %w", path, err)
	}

	return nil
}

func (osTestEdgeFS) RemoveAll(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("test edgefs: removeall %s: %w", path, err)
	}

	return nil
}

func (osTestEdgeFS) Rename(oldPath, newPath string) error {
	err := os.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("test edgefs: rename %s: %w", oldPath, err)
	}

	return nil
}

func (osTestEdgeFS) Stat(path string) (fs.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("test edgefs: stat %s: %w", path, err)
	}

	return info, nil
}

func (osTestEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	err := filepath.WalkDir(root, fn)
	if err != nil {
		return fmt.Errorf("test edgefs: walk %s: %w", root, err)
	}

	return nil
}

func (osTestEdgeFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	err := os.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("test edgefs: writing %s: %w", path, err)
	}

	return nil
}

func (osTestEdgeFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	err := os.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("test edgefs: writing %s: %w", path, err)
	}

	return nil
}

func (osTestEdgeFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("test edgefs: opening excl %s: %w", path, err)
	}

	defer func() { _ = file.Close() }()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("test edgefs: writing excl %s: %w", path, err)
	}

	return nil
}
