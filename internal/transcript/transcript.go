// Package transcript reads recent transcript context for the unified classifier (ARCH-3).
package transcript

// FileReader is the interface for reading file contents.
// Wire os.ReadFile in production (via cli.go).
type FileReader func(name string) ([]byte, error)

// Reader reads recent transcript context from session transcript files.
type Reader struct {
	readFile FileReader
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

	if len(content) <= maxTokens {
		return string(content), nil
	}

	// Take the tail (most recent portion)
	return string(content[len(content)-maxTokens:]), nil
}
