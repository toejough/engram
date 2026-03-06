package surfacinglog_test

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"engram/internal/surfacinglog"
)

// T-101: Surfacing log append writes JSONL entry.
func TestT101_LogSurfacing_WritesJSONLEntry(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var appendedPath string
	var appendedData []byte

	logger := surfacinglog.New("/data",
		surfacinglog.WithAppendFile(func(name string, data []byte, _ os.FileMode) error {
			appendedPath = name
			appendedData = data
			return nil
		}),
	)

	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	err := logger.LogSurfacing("memories/foo.toml", "prompt", ts)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(appendedPath).To(gomega.Equal("/data/surfacing-log.jsonl"))
	g.Expect(string(appendedData)).To(gomega.ContainSubstring(`"memory_path":"memories/foo.toml"`))
	g.Expect(string(appendedData)).To(gomega.ContainSubstring(`"mode":"prompt"`))
	g.Expect(string(appendedData)).
		To(gomega.ContainSubstring(`"surfaced_at":"2024-06-15T12:00:00Z"`))
	g.Expect(string(appendedData)).To(gomega.HaveSuffix("\n"))
}

// T-102: Surfacing log append for multiple memories in one event.
func TestT102_LogSurfacing_MultipleCallsMultipleLines(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var lines []string

	logger := surfacinglog.New("/data",
		surfacinglog.WithAppendFile(func(_ string, data []byte, _ os.FileMode) error {
			lines = append(lines, strings.TrimRight(string(data), "\n"))
			return nil
		}),
	)

	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	for _, path := range []string{"mem/a.toml", "mem/b.toml", "mem/c.toml"} {
		err := logger.LogSurfacing(path, "session-start", ts)
		g.Expect(err).NotTo(gomega.HaveOccurred())

		if err != nil {
			return
		}
	}

	g.Expect(lines).To(gomega.HaveLen(3))
	g.Expect(lines[0]).To(gomega.ContainSubstring(`"memory_path":"mem/a.toml"`))
	g.Expect(lines[1]).To(gomega.ContainSubstring(`"memory_path":"mem/b.toml"`))
	g.Expect(lines[2]).To(gomega.ContainSubstring(`"memory_path":"mem/c.toml"`))
}

// T-103: Surfacing log append error is returned to caller.
func TestT103_LogSurfacing_AppendErrorReturned(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	appendErr := errors.New("disk full")

	logger := surfacinglog.New("/data",
		surfacinglog.WithAppendFile(func(string, []byte, os.FileMode) error {
			return appendErr
		}),
	)

	err := logger.LogSurfacing("memories/foo.toml", "tool", time.Now())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("disk full")))
}

// T-104: Surfacing log read-and-clear returns events and removes file.
func TestT104_ReadAndClear_ReturnsEventsAndRemovesFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	lines := `` +
		`{"memory_path":"mem/a.toml","mode":"prompt","surfaced_at":"2024-01-01T00:00:00Z"}` + "\n" +
		`{"memory_path":"mem/b.toml","mode":"prompt","surfaced_at":"2024-01-01T00:00:00Z"}` + "\n" +
		`{"memory_path":"mem/c.toml","mode":"session-start","surfaced_at":"2024-01-01T00:00:00Z"}` + "\n" +
		`{"memory_path":"mem/d.toml","mode":"tool","surfaced_at":"2024-01-01T00:00:00Z"}` + "\n" +
		`{"memory_path":"mem/e.toml","mode":"prompt","surfaced_at":"2024-01-01T00:00:00Z"}` + "\n"

	var removedPath string

	logger := surfacinglog.New("/data",
		surfacinglog.WithReadFile(func(string) ([]byte, error) {
			return []byte(lines), nil
		}),
		surfacinglog.WithRemoveFile(func(path string) error {
			removedPath = path
			return nil
		}),
	)

	events, err := logger.ReadAndClear()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(gomega.HaveLen(5))
	g.Expect(events[0].MemoryPath).To(gomega.Equal("mem/a.toml"))
	g.Expect(events[0].Mode).To(gomega.Equal("prompt"))
	g.Expect(events[3].Mode).To(gomega.Equal("tool"))
	g.Expect(removedPath).To(gomega.Equal("/data/surfacing-log.jsonl"))
}

// T-105: Surfacing log read-and-clear with missing file returns empty slice.
func TestT105_ReadAndClear_MissingFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	removeCallCount := 0

	logger := surfacinglog.New("/data",
		surfacinglog.WithReadFile(func(string) ([]byte, error) {
			return nil, os.ErrNotExist
		}),
		surfacinglog.WithRemoveFile(func(string) error {
			removeCallCount++
			return nil
		}),
	)

	events, err := logger.ReadAndClear()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(events).To(gomega.BeEmpty())
	g.Expect(removeCallCount).To(gomega.Equal(0))
}
