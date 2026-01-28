package log_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/log"
	"pgregory.net/rapid"
)

func fixedTime() time.Time {
	return time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
}

func nowFunc() func() time.Time {
	return func() time.Time { return fixedTime() }
}

func TestWrite(t *testing.T) {
	t.Run("appends valid JSONL entry to file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := log.Write(dir, "status", "task-status", "TASK-001 started", log.WriteOpts{
			Task:  "TASK-001",
			Phase: "implementation",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
		g.Expect(err).ToNot(HaveOccurred())

		var entry log.Entry
		err = json.Unmarshal(content[:len(content)-1], &entry) // strip trailing newline
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(entry.Level).To(Equal("status"))
		g.Expect(entry.Subject).To(Equal("task-status"))
		g.Expect(entry.Message).To(Equal("TASK-001 started"))
		g.Expect(entry.Task).To(Equal("TASK-001"))
		g.Expect(entry.Phase).To(Equal("implementation"))
		g.Expect(entry.Timestamp).To(Equal("2026-01-27T12:00:00Z"))
	})

	t.Run("appends multiple entries as separate lines", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := log.Write(dir, "status", "task-status", "first", log.WriteOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		err = log.Write(dir, "phase", "phase-change", "second", log.WriteOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
		g.Expect(err).ToNot(HaveOccurred())

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		g.Expect(lines).To(HaveLen(2))

		// Both lines should be valid JSON
		for _, line := range lines {
			var entry log.Entry
			err := json.Unmarshal([]byte(line), &entry)
			g.Expect(err).ToNot(HaveOccurred())
		}
	})

	t.Run("creates file if it does not exist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := log.Write(dir, "verbose", "thinking", "hello", log.WriteOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = os.Stat(filepath.Join(dir, log.LogFile))
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("rejects invalid level", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := log.Write(dir, "invalid", "thinking", "hello", log.WriteOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid level"))
	})

	t.Run("rejects invalid subject", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := log.Write(dir, "verbose", "invalid", "hello", log.WriteOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid subject"))
	})

	t.Run("omits empty optional fields from JSON", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := log.Write(dir, "phase", "phase-change", "started", log.WriteOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
		g.Expect(err).ToNot(HaveOccurred())

		line := strings.TrimSpace(string(content))
		g.Expect(line).ToNot(ContainSubstring(`"task"`))
		g.Expect(line).ToNot(ContainSubstring(`"detail"`))
	})
}

func TestWriteProperty(t *testing.T) {
	levels := make([]string, 0, len(log.ValidLevels))
	for k := range log.ValidLevels {
		levels = append(levels, k)
	}

	subjects := make([]string, 0, len(log.ValidSubjects))
	for k := range log.ValidSubjects {
		subjects = append(subjects, k)
	}

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		level := rapid.SampledFrom(levels).Draw(rt, "level")
		subject := rapid.SampledFrom(subjects).Draw(rt, "subject")
		message := rapid.String().Draw(rt, "message")

		err := log.Write(dir, level, subject, message, log.WriteOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify it's valid JSONL
		content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
		g.Expect(err).ToNot(HaveOccurred())

		var entry log.Entry
		err = json.Unmarshal([]byte(strings.TrimSpace(string(content))), &entry)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(entry.Level).To(Equal(level))
		g.Expect(entry.Subject).To(Equal(subject))
		g.Expect(entry.Message).To(Equal(message))
	})
}
