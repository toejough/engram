// Package recall finds and reads Claude Code session transcripts.
// It provides SessionFinder (locates transcript files sorted by recency)
// and TranscriptReader (reads and strips transcript noise with a byte budget).
package recall

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"engram/internal/sessionctx"
)

// DirLister lists .jsonl files in a directory with their modification times.
type DirLister interface {
	ListJSONL(dir string) ([]FileEntry, error)
}

// FileEntry represents a transcript file with its path and modification time.
type FileEntry struct {
	Path  string
	Mtime time.Time
}

// SessionFinder finds Claude Code session transcript files for a project.
type SessionFinder struct {
	lister DirLister
}

// NewSessionFinder creates a SessionFinder with the given directory lister.
func NewSessionFinder(lister DirLister) *SessionFinder {
	return &SessionFinder{lister: lister}
}

// Find returns transcript paths sorted by mtime descending (newest first).
func (f *SessionFinder) Find(projectDir string) ([]string, error) {
	entries, err := f.lister.ListJSONL(projectDir)
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Mtime.After(entries[j].Mtime)
	})

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}

	return paths, nil
}

// TranscriptReader reads session transcripts and strips noise.
type TranscriptReader struct {
	reader sessionctx.FileReader
}

// NewTranscriptReader creates a TranscriptReader with the given file reader.
func NewTranscriptReader(reader sessionctx.FileReader) *TranscriptReader {
	return &TranscriptReader{reader: reader}
}

// Read reads a transcript file, strips noise (using context.Strip), and returns
// the stripped content as a single string. Stops accumulating when bytesRead
// exceeds budgetBytes. Returns the stripped content, bytes consumed, and any error.
func (r *TranscriptReader) Read(path string, budgetBytes int) (string, int, error) {
	content, err := r.reader.Read(path)
	if err != nil {
		return "", 0, fmt.Errorf("reading transcript: %w", err)
	}

	rawLines := strings.Split(string(content), "\n")

	// Remove empty trailing line from final newline.
	nonEmpty := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		if line != "" {
			nonEmpty = append(nonEmpty, line)
		}
	}

	stripped := sessionctx.Strip(nonEmpty)

	// Accumulate lines from the tail (most recent content first),
	// then reverse to chronological order.
	bytesRead := 0
	startIdx := len(stripped)

	for i := len(stripped) - 1; i >= 0; i-- {
		lineLen := len(stripped[i]) + 1 // +1 for newline separator
		if bytesRead+lineLen > budgetBytes && bytesRead > 0 {
			break
		}

		startIdx = i
		bytesRead += lineLen
	}

	var builder strings.Builder

	for _, line := range stripped[startIdx:] {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}

	return builder.String(), bytesRead, nil
}
