package maintain

import (
	"fmt"
	"maps"
	"os"

	"github.com/BurntSushi/toml"

	"engram/internal/tomlwriter"
)

// TOMLRewriter reads a memory TOML, applies field updates, and writes atomically.
type TOMLRewriter struct {
	readFile func(name string) ([]byte, error)
	writer   *tomlwriter.Writer
}

// NewTOMLRewriter creates a TOMLRewriter with real filesystem operations.
func NewTOMLRewriter(opts ...TOMLRewriterOption) *TOMLRewriter {
	rewriter := &TOMLRewriter{
		readFile: os.ReadFile,
		writer:   tomlwriter.New(),
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

	return rewriter.writer.AtomicWrite(path, existing)
}

// TOMLRewriterOption configures a TOMLRewriter.
type TOMLRewriterOption func(*TOMLRewriter)

// WithReadFile overrides the file reading function.
func WithReadFile(fn func(name string) ([]byte, error)) TOMLRewriterOption {
	return func(rewriter *TOMLRewriter) { rewriter.readFile = fn }
}

// WithWriter sets the tomlwriter.Writer for atomic writes.
func WithWriter(w *tomlwriter.Writer) TOMLRewriterOption {
	return func(rewriter *TOMLRewriter) { rewriter.writer = w }
}
