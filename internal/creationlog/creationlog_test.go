package creationlog_test

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"engram/internal/creationlog"
)

func TestLogReader_ReadAndClear_MissingFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	removeCallCount := 0

	reader := creationlog.NewLogReader(
		creationlog.WithReaderReadFile(func(string) ([]byte, error) {
			return nil, os.ErrNotExist
		}),
		creationlog.WithRemoveFile(func(string) error {
			removeCallCount++
			return nil
		}),
	)

	entries, err := reader.ReadAndClear("/data")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(gomega.BeEmpty())
	g.Expect(removeCallCount).To(gomega.Equal(0))
}

func TestLogReader_ReadAndClear_ReadErrorReturnsError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	readErr := errors.New("permission denied")
	removeCallCount := 0

	reader := creationlog.NewLogReader(
		creationlog.WithReaderReadFile(func(string) ([]byte, error) {
			return nil, readErr
		}),
		creationlog.WithRemoveFile(func(string) error {
			removeCallCount++
			return nil
		}),
	)

	_, err := reader.ReadAndClear("/data")
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("permission denied")))
	g.Expect(removeCallCount).To(gomega.Equal(0))
}

func TestLogReader_ReadAndClear_ReturnsEntriesAndRemovesFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	lines := `{"timestamp":"2024-01-01T00:00:00Z","title":"Alpha","tier":"A","filename":"alpha.toml"}
{"timestamp":"2024-01-02T00:00:00Z","title":"Beta","tier":"B","filename":"beta.toml"}
{"timestamp":"2024-01-03T00:00:00Z","title":"Gamma","tier":"C","filename":"gamma.toml"}
`

	var removedPath string

	reader := creationlog.NewLogReader(
		creationlog.WithReaderReadFile(func(string) ([]byte, error) {
			return []byte(lines), nil
		}),
		creationlog.WithRemoveFile(func(path string) error {
			removedPath = path
			return nil
		}),
	)

	entries, err := reader.ReadAndClear("/data")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(3))
	g.Expect(entries[0].Title).To(gomega.Equal("Alpha"))
	g.Expect(entries[1].Title).To(gomega.Equal("Beta"))
	g.Expect(entries[2].Title).To(gomega.Equal("Gamma"))
	g.Expect(removedPath).To(gomega.Equal("/data/creation-log.jsonl"))
}

func TestLogReader_ReadAndClear_SkipsMalformedLines(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	lines := `{"timestamp":"2024-01-01T00:00:00Z","title":"Good One","tier":"A","filename":"good1.toml"}
not valid json at all
{"timestamp":"2024-01-03T00:00:00Z","title":"Good Two","tier":"C","filename":"good2.toml"}
`
	removeCallCount := 0

	reader := creationlog.NewLogReader(
		creationlog.WithReaderReadFile(func(string) ([]byte, error) {
			return []byte(lines), nil
		}),
		creationlog.WithRemoveFile(func(string) error {
			removeCallCount++
			return nil
		}),
	)

	entries, err := reader.ReadAndClear("/data")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(2))
	g.Expect(entries[0].Title).To(gomega.Equal("Good One"))
	g.Expect(entries[1].Title).To(gomega.Equal("Good Two"))
	g.Expect(removeCallCount).To(gomega.Equal(1))
}

func TestLogWriter_Append_AppendsToExistingFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()

	// Pre-populate file with existing content.
	existing := `{"timestamp":"2024-01-01T00:00:00Z","title":"Existing","tier":"B","filename":"existing.toml"}` + "\n"
	logPath := dir + "/creation-log.jsonl"

	writeErr := os.WriteFile(logPath, []byte(existing), 0o644)
	g.Expect(writeErr).NotTo(gomega.HaveOccurred())

	if writeErr != nil {
		return
	}

	writer := creationlog.NewLogWriter(
		creationlog.WithNow(func() time.Time {
			return time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
		}),
	)

	entry := creationlog.LogEntry{
		Title:    "New Memory",
		Tier:     "A",
		Filename: "new-memory.toml",
	}

	err := writer.Append(entry, dir)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	content, readErr := os.ReadFile(logPath)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	g.Expect(lines).To(gomega.HaveLen(2))
	g.Expect(lines[0]).To(gomega.ContainSubstring(`"title":"Existing"`))
	g.Expect(lines[1]).To(gomega.ContainSubstring(`"title":"New Memory"`))
}

func TestLogWriter_Append_CreatesNewFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()

	writer := creationlog.NewLogWriter(
		creationlog.WithNow(func() time.Time {
			return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		}),
	)

	entry := creationlog.LogEntry{
		Title:    "Test Memory",
		Tier:     "A",
		Filename: "test-memory.toml",
	}

	err := writer.Append(entry, dir)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	logPath := dir + "/creation-log.jsonl"

	content, readErr := os.ReadFile(logPath)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	g.Expect(lines).To(gomega.HaveLen(1))
	g.Expect(lines[0]).To(gomega.ContainSubstring(`"title":"Test Memory"`))
}

func TestLogWriter_Append_OpenFileErrorReturned(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	openErr := errors.New("disk full")

	writer := creationlog.NewLogWriter(
		creationlog.WithOpenFile(func(string, int, os.FileMode) (*os.File, error) {
			return nil, openErr
		}),
		creationlog.WithNow(time.Now),
	)

	err := writer.Append(creationlog.LogEntry{Title: "X", Tier: "A", Filename: "x.toml"}, "/data")
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("disk full")))
}

func TestLogWriter_Append_SetsTimestampFromClock(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fixedTime := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)

	writer := creationlog.NewLogWriter(
		creationlog.WithNow(func() time.Time {
			return fixedTime
		}),
	)

	entry := creationlog.LogEntry{
		Title:    "Timestamped Memory",
		Tier:     "C",
		Filename: "ts-memory.toml",
		// Timestamp intentionally empty
	}

	err := writer.Append(entry, dir)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	logPath := dir + "/creation-log.jsonl"

	content, readErr := os.ReadFile(logPath)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(content)).
		To(gomega.ContainSubstring(`"timestamp":"2024-06-15T12:30:00Z"`))
}

func TestLogWriter_Append_WriteErrorReturned(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writer := creationlog.NewLogWriter(
		creationlog.WithOpenFile(func(_ string, _ int, _ os.FileMode) (*os.File, error) {
			// Return a file that is immediately closed so Write fails.
			file, err := os.CreateTemp(t.TempDir(), "test-*.jsonl")
			if err != nil {
				return nil, err
			}

			_ = file.Close()

			return file, nil
		}),
		creationlog.WithNow(time.Now),
	)

	err := writer.Append(creationlog.LogEntry{Title: "X", Tier: "A", Filename: "x.toml"}, "/data")
	g.Expect(err).To(gomega.HaveOccurred())
}
