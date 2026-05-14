// Package transcript finds and reads Claude Code session transcripts.
// It provides SessionFinder (locates transcript files sorted by recency)
// and JSONLReader (reads and strips transcript noise with a byte budget).
package transcript

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	sessionctx "github.com/toejough/engram/internal/context"
)

// DirLister lists .jsonl files in a directory with their modification times.
type DirLister interface {
	ListJSONL(dir string) ([]FileEntry, error)
}

// FileEntry represents a transcript file with its path and modification time.
type FileEntry struct {
	Path   string
	Mtime  time.Time
	Source string
}

// Finder finds session transcript files.
type Finder interface {
	Find(dirs ...string) ([]FileEntry, error)
}

// JSONLReader reads session transcripts and strips noise.
type JSONLReader struct {
	reader sessionctx.FileReader
}

// NewJSONLReader creates a JSONLReader with the given file reader.
func NewJSONLReader(reader sessionctx.FileReader) *JSONLReader {
	return &JSONLReader{reader: reader}
}

// Read reads a transcript file, strips noise (using context.Strip), and returns
// the stripped content as a single string. Stops accumulating when bytesRead
// exceeds budgetBytes. Returns the stripped content, bytes consumed, and any error.
func (r *JSONLReader) Read(path string, budgetBytes int) (string, int, error) {
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

	cfg := sessionctx.StripConfig{ToolSummaryMode: true}
	stripped := sessionctx.StripWithConfig(nonEmpty, cfg)

	// Accumulate lines from the tail (most recent content first),
	// then reverse to chronological order.
	bytesRead := 0
	startIdx := len(stripped)

	for i, v := range slices.Backward(stripped) {
		lineLen := len(v) + 1 // +1 for newline separator
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

// Reader reads and strips a transcript file.
type Reader interface {
	Read(path string, budgetBytes int) (string, int, error)
}

// SessionFinder finds Claude Code session transcript files for a project.
type SessionFinder struct {
	lister DirLister
}

// NewSessionFinder creates a SessionFinder with the given directory lister.
func NewSessionFinder(lister DirLister) *SessionFinder {
	return &SessionFinder{lister: lister}
}

// Find returns transcript entries from all directories, merged and sorted by
// mtime descending (newest first). Missing directories are silently skipped.
func (f *SessionFinder) Find(dirs ...string) ([]FileEntry, error) {
	seen := make(map[string]struct{})
	all := make([]FileEntry, 0)

	for _, dir := range dirs {
		entries, err := f.lister.ListJSONL(dir)
		if err != nil {
			return nil, fmt.Errorf("listing sessions in %s: %w", dir, err)
		}

		for _, e := range entries {
			if _, ok := seen[e.Path]; !ok {
				seen[e.Path] = struct{}{}
				e.Source = sourceClaude
				all = append(all, e)
			}
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Mtime.After(all[j].Mtime)
	})

	return all, nil
}

// unexported constants.
const (
	sourceClaude   = "claude"
	sourceOpencode = "opencode"
)
