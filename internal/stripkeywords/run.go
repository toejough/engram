package stripkeywords

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// memRecord is the minimal TOML shape needed to read/write situation fields.
type memRecord struct {
	Type      string `toml:"type"`
	Situation string `toml:"situation"`
	UpdatedAt string `toml:"updated_at"`
	CreatedAt string `toml:"created_at"`
}

// Deps holds injected I/O dependencies for the cleanup run.
type Deps struct {
	ReadDir    func(string) ([]os.DirEntry, error)
	ReadFile   func(string) ([]byte, error)
	CreateTemp func(string, string) (*os.File, error)
	Rename     func(string, string) error
	Remove     func(string) error
	Stdout     io.Writer
	Stderr     io.Writer
}

// DefaultDeps returns Deps wired to real filesystem operations.
func DefaultDeps() Deps {
	return Deps{
		ReadDir:    os.ReadDir,
		ReadFile:   os.ReadFile,
		CreateTemp: os.CreateTemp,
		Rename:     os.Rename,
		Remove:     os.Remove,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
}

// dirConfig describes a memory directory and whether it must exist.
type dirConfig struct {
	path     string
	required bool
}

// Run walks memory/feedback/ and memory/facts/ under dataDir, stripping
// "\nKeywords: ..." suffixes from situation fields. It is idempotent.
// The memory/feedback directory is required; memory/facts is optional.
func Run(dataDir string, deps Deps) error {
	dirs := []dirConfig{
		{path: filepath.Join(dataDir, "memory", "feedback"), required: true},
		{path: filepath.Join(dataDir, "memory", "facts"), required: false},
	}

	totalStripped := 0
	totalUnchanged := 0

	for _, dc := range dirs {
		stripped, unchanged, err := processDir(dc.path, dc.required, deps)
		if err != nil {
			return err
		}

		totalStripped += stripped
		totalUnchanged += unchanged
	}

	_, _ = fmt.Fprintf(deps.Stdout, "\nStripped: %d, Unchanged: %d\n", totalStripped, totalUnchanged)

	return nil
}

func processDir(dir string, required bool, deps Deps) (stripped, unchanged int, err error) {
	entries, err := deps.ReadDir(dir)
	if err != nil {
		if !required && os.IsNotExist(err) {
			return 0, 0, nil
		}

		return 0, 0, fmt.Errorf("reading %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		changed, processErr := processFile(path, entry.Name(), deps)
		if processErr != nil {
			return 0, 0, processErr
		}

		if changed {
			stripped++
		} else {
			unchanged++
		}
	}

	return stripped, unchanged, nil
}

func processFile(path, name string, deps Deps) (bool, error) {
	data, err := deps.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", name, err)
	}

	var rec memRecord

	if _, decErr := toml.Decode(string(data), &rec); decErr != nil {
		return false, fmt.Errorf("%s: decoding TOML: %w", name, decErr)
	}

	stripped := StripKeywordsSuffix(rec.Situation)
	if stripped == rec.Situation {
		_, _ = fmt.Fprintf(deps.Stdout, "OK (no change): %s\n", name)

		return false, nil
	}

	rec.Situation = stripped

	if writeErr := atomicWrite(path, rec, deps); writeErr != nil {
		return false, fmt.Errorf("%s: writing: %w", name, writeErr)
	}

	_, _ = fmt.Fprintf(deps.Stdout, "STRIPPED: %s\n", name)

	return true, nil
}

func atomicWrite(path string, rec memRecord, deps Deps) error {
	dir := filepath.Dir(path)

	tmpFile, err := deps.CreateTemp(dir, ".tmp-stripkw-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	cleanup := func() { _ = deps.Remove(tmpPath) }

	encErr := toml.NewEncoder(tmpFile).Encode(rec)
	closeErr := tmpFile.Close()

	if encErr != nil {
		cleanup()
		return fmt.Errorf("encoding TOML: %w", encErr)
	}

	if closeErr != nil {
		cleanup()
		return fmt.Errorf("closing temp file: %w", closeErr)
	}

	if renameErr := deps.Rename(tmpPath, path); renameErr != nil {
		cleanup()
		return fmt.Errorf("renaming temp to destination: %w", renameErr)
	}

	return nil
}
