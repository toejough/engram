package context

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SessionContext holds the parsed contents of a session context file.
type SessionContext struct {
	Summary   string
	Offset    int64
	SessionID string
}

// SessionFile reads and writes session context files with HTML metadata headers.
type SessionFile struct {
	reader     FileReader
	writer     FileWriter
	dirCreator DirCreator
	renamer    Renamer
	clock      Timestamper
}

// NewSessionFile creates a SessionFile with injected dependencies.
func NewSessionFile(
	reader FileReader,
	writer FileWriter,
	dirCreator DirCreator,
	renamer Renamer,
	clock Timestamper,
) *SessionFile {
	return &SessionFile{
		reader:     reader,
		writer:     writer,
		dirCreator: dirCreator,
		renamer:    renamer,
		clock:      clock,
	}
}

// Read parses a session context file, returning its metadata and summary.
// Returns zero-value SessionContext for missing files (non-fatal).
func (f *SessionFile) Read(path string) (SessionContext, error) {
	content, err := f.reader.Read(path)
	if err != nil {
		return SessionContext{}, nil //nolint:nilerr // missing file is non-fatal
	}

	text := string(content)

	var result SessionContext

	// Parse metadata from first line.
	const headerAndBody = 2

	lines := strings.SplitN(text, "\n", headerAndBody)

	if len(lines) == 0 {
		return result, nil
	}

	matches := metadataPattern.FindStringSubmatch(lines[0])
	if matches != nil {
		const expectedMatches = 4

		if len(matches) >= expectedMatches {
			offset, parseErr := strconv.ParseInt(matches[2], 10, 64)
			if parseErr == nil {
				result.Offset = offset
			}

			result.SessionID = matches[3]
		}
	}

	// Extract summary: everything after the HTML comment line, trimmed.
	if len(lines) > 1 {
		result.Summary = strings.TrimSpace(lines[1])
	}

	return result, nil
}

// Write atomically writes a session context file (temp file + rename).
// Creates the parent directory if it does not exist.
func (f *SessionFile) Write(path string, session SessionContext) error {
	dir := filepath.Dir(path)

	err := f.dirCreator.MkdirAll(dir)
	if err != nil {
		return fmt.Errorf("context: creating directory %s: %w", dir, err)
	}

	header := fmt.Sprintf(
		"<!-- engram session context | updated: %s | offset: %d | session: %s -->",
		f.clock.Now().UTC().Format("2006-01-02T15:04:05Z"),
		session.Offset,
		session.SessionID,
	)

	content := header + "\n\n" + session.Summary

	tmpPath := path + ".tmp"

	err = f.writer.Write(tmpPath, []byte(content))
	if err != nil {
		return fmt.Errorf("context: writing temp file: %w", err)
	}

	err = f.renamer.Rename(tmpPath, path)
	if err != nil {
		return fmt.Errorf("context: renaming temp file: %w", err)
	}

	return nil
}

// unexported variables.
var (
	metadataPattern = regexp.MustCompile(
		`<!-- engram session context \| updated: ([^ ]+) \| offset: (\d+) \| session: ([^ ]+) -->`,
	)
)
