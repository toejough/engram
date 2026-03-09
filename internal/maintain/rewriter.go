package maintain

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// TOMLRewriter reads a memory TOML, applies field updates, and writes atomically.
type TOMLRewriter struct {
	readFile  func(name string) ([]byte, error)
	writeFile func(name string, data []byte, perm os.FileMode) error
	rename    func(oldpath, newpath string) error
}

// NewTOMLRewriter creates a TOMLRewriter with real filesystem operations.
func NewTOMLRewriter(opts ...TOMLRewriterOption) *TOMLRewriter {
	rewriter := &TOMLRewriter{
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		rename:    os.Rename,
	}
	for _, opt := range opts {
		opt(rewriter)
	}

	return rewriter
}

// Rewrite reads the TOML at path, merges updates, and writes atomically.
func (rewriter *TOMLRewriter) Rewrite(path string, updates map[string]any) error {
	data, err := rewriter.readFile(path)
	if err != nil {
		return fmt.Errorf("reading memory TOML: %w", err)
	}

	var existing map[string]any
	_, decodeErr := toml.Decode(string(data), &existing)
	if decodeErr != nil {
		return fmt.Errorf("decoding memory TOML: %w", decodeErr)
	}

	maps.Copy(existing, updates)

	var buf bytes.Buffer

	encodeErr := toml.NewEncoder(&buf).Encode(existing)
	if encodeErr != nil {
		return fmt.Errorf("encoding memory TOML: %w", encodeErr)
	}

	// Atomic write: write to temp, then rename.
	dir := filepath.Dir(path)
	tmpPath := filepath.Join(dir, ".tmp-rewrite")

	const filePerm = 0o644

	writeErr := rewriter.writeFile(tmpPath, buf.Bytes(), filePerm)
	if writeErr != nil {
		return fmt.Errorf("writing temp file: %w", writeErr)
	}

	renameErr := rewriter.rename(tmpPath, path)
	if renameErr != nil {
		return fmt.Errorf("renaming temp to final: %w", renameErr)
	}

	return nil
}

// TOMLRewriterOption configures a TOMLRewriter.
type TOMLRewriterOption func(*TOMLRewriter)

// WithReadFile overrides the file reading function.
func WithReadFile(fn func(name string) ([]byte, error)) TOMLRewriterOption {
	return func(rewriter *TOMLRewriter) { rewriter.readFile = fn }
}

// WithRenameFile overrides the file rename function.
func WithRenameFile(fn func(oldpath, newpath string) error) TOMLRewriterOption {
	return func(rewriter *TOMLRewriter) { rewriter.rename = fn }
}

// WithWriteFile overrides the file writing function.
func WithWriteFile(fn func(name string, data []byte, perm os.FileMode) error) TOMLRewriterOption {
	return func(rewriter *TOMLRewriter) { rewriter.writeFile = fn }
}
