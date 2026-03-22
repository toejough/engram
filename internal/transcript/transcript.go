// Package transcript reads recent transcript context for the unified classifier (ARCH-3).
package transcript

import "strings"

// FileReader is the interface for reading file contents.
// Wire os.ReadFile in production (via cli.go).
type FileReader func(name string) ([]byte, error)

// Reader reads recent transcript context from session transcript files.
type Reader struct {
	readFile FileReader
	strip    StripFunc
}

// New creates a new transcript Reader with the given file reader.
func New(readFile FileReader) *Reader {
	return &Reader{readFile: readFile}
}

// ReadRecent reads the most recent portion of a transcript file.
// If the file exceeds maxTokens characters, the tail is returned.
// Returns empty string for missing files (non-fatal, context is advisory).
func (r *Reader) ReadRecent(transcriptPath string, maxTokens int) (string, error) {
	if transcriptPath == "" {
		return "", nil
	}

	content, err := r.readFile(transcriptPath)
	if err != nil {
		return "", nil //nolint:nilerr // non-fatal: transcript context is advisory
	}

	text := string(content)

	if r.strip != nil {
		lines := strings.Split(text, "\n")
		stripped := r.strip(lines)
		text = strings.Join(stripped, "\n")
	}

	if len(text) <= maxTokens {
		return text, nil
	}

	// Take the tail (most recent portion)
	return text[len(text)-maxTokens:], nil
}

// SetStrip sets an optional function to clean transcript lines.
func (r *Reader) SetStrip(fn StripFunc) {
	r.strip = fn
}

// StripFunc transforms raw transcript lines into cleaned conversation text.
type StripFunc func(lines []string) []string
