package log_test

import (
	"encoding/json"
	"fmt"
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

// MockFS implements log.FileSystem for testing
type MockFS struct {
	Files map[string][]byte
}

func (m *MockFS) AppendFile(path string, data []byte) error {
	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}
	m.Files[path] = append(m.Files[path], data...)
	return nil
}

func (m *MockFS) ReadFile(path string) ([]byte, error) {
	content, exists := m.Files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return content, nil
}

func (m *MockFS) FileExists(path string) bool {
	_, exists := m.Files[path]
	return exists
}

func TestWrite(t *testing.T) {
	t.Parallel()
	t.Run("appends valid JSONL entry to file", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		err := log.Write("testdir", "status", "task-status", "TASK-001 started", log.WriteOpts{
			Task:  "TASK-001",
			Phase: "implementation",
		}, nowFunc(), fs)
		g.Expect(err).ToNot(HaveOccurred())

		content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
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
		fs := &MockFS{}

		err := log.Write("testdir", "status", "task-status", "first", log.WriteOpts{}, nowFunc(), fs)
		g.Expect(err).ToNot(HaveOccurred())

		err = log.Write("testdir", "phase", "phase-change", "second", log.WriteOpts{}, nowFunc(), fs)
		g.Expect(err).ToNot(HaveOccurred())

		content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
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
		fs := &MockFS{}

		err := log.Write("testdir", "verbose", "thinking", "hello", log.WriteOpts{}, nowFunc(), fs)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(fs.FileExists(filepath.Join("testdir", log.LogFile))).To(BeTrue())
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("rejects invalid level", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		err := log.Write("testdir", "invalid", "thinking", "hello", log.WriteOpts{}, nowFunc(), fs)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid level"))
	})

	t.Run("rejects invalid subject", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		err := log.Write("testdir", "verbose", "invalid", "hello", log.WriteOpts{}, nowFunc(), fs)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid subject"))
	})

	t.Run("omits empty optional fields from JSON", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{}

		err := log.Write("testdir", "phase", "phase-change", "started", log.WriteOpts{}, nowFunc(), fs)
		g.Expect(err).ToNot(HaveOccurred())

		content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
		g.Expect(err).ToNot(HaveOccurred())

		line := strings.TrimSpace(string(content))
		g.Expect(line).ToNot(ContainSubstring(`"task"`))
		g.Expect(line).ToNot(ContainSubstring(`"detail"`))
	})
}

// TEST-410 traces: TASK-016
// Test Write includes model field in entry.
func TestWrite_ModelField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	err := log.Write("testdir", "status", "task-status", "dispatched", log.WriteOpts{
		Task:  "TASK-001",
		Model: "haiku",
	}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.Model).To(Equal("haiku"))
}

// TEST-411 traces: TASK-016
// Test Write omits model when empty (backwards compatible).
func TestWrite_ModelOmittedWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	err := log.Write("testdir", "status", "task-status", "started", log.WriteOpts{
		Task: "TASK-001",
	}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	line := strings.TrimSpace(string(content))
	g.Expect(line).ToNot(ContainSubstring(`"model"`))
}

func TestWriteProperty(t *testing.T) {
	t.Parallel()
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
		fs := &MockFS{}
		level := rapid.SampledFrom(levels).Draw(rt, "level")
		subject := rapid.SampledFrom(subjects).Draw(rt, "subject")
		message := rapid.String().Draw(rt, "message")

		err := log.Write("testdir", level, subject, message, log.WriteOpts{}, nowFunc(), fs)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify it's valid JSONL
		content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
		g.Expect(err).ToNot(HaveOccurred())

		var entry log.Entry
		err = json.Unmarshal([]byte(strings.TrimSpace(string(content))), &entry)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(entry.Level).To(Equal(level))
		g.Expect(entry.Subject).To(Equal(subject))
		g.Expect(entry.Message).To(Equal(message))
	})
}

// TEST-412 traces: TASK-016
// Test Read returns all entries when no filter.
func TestRead_AllEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	// Write some entries
	_ = log.Write("testdir", "status", "task-status", "first", log.WriteOpts{Task: "TASK-001"}, nowFunc(), fs)
	_ = log.Write("testdir", "phase", "phase-change", "second", log.WriteOpts{}, nowFunc(), fs)

	entries, err := log.Read("testdir", log.ReadOpts{}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(HaveLen(2))
	g.Expect(entries[0].Message).To(Equal("first"))
	g.Expect(entries[1].Message).To(Equal("second"))
}

// TEST-413 traces: TASK-016
// Test Read filters by model.
func TestRead_FilterByModel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	// Write entries with different models
	_ = log.Write("testdir", "status", "task-status", "haiku task", log.WriteOpts{Model: "haiku"}, nowFunc(), fs)
	_ = log.Write("testdir", "status", "task-status", "sonnet task", log.WriteOpts{Model: "sonnet"}, nowFunc(), fs)
	_ = log.Write("testdir", "status", "task-status", "opus task", log.WriteOpts{Model: "opus"}, nowFunc(), fs)
	_ = log.Write("testdir", "status", "task-status", "no model", log.WriteOpts{}, nowFunc(), fs)

	// Filter by haiku
	entries, err := log.Read("testdir", log.ReadOpts{Model: "haiku"}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Message).To(Equal("haiku task"))
	g.Expect(entries[0].Model).To(Equal("haiku"))
}

// TEST-414 traces: TASK-016
// Test Read returns empty slice when log file missing.
func TestRead_NoLogFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	entries, err := log.Read("testdir", log.ReadOpts{}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

// TEST-500 traces: TASK-027
// Test Write calculates token estimate from message length.
func TestWrite_TokenEstimate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	// 40 characters / 4 = 10 tokens
	message := "This is exactly forty characters long!!!"
	g.Expect(len(message)).To(Equal(40))

	err := log.Write("testdir", "status", "task-status", message, log.WriteOpts{}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.TokensEstimate).To(Equal(10))
}

// TEST-501 traces: TASK-027
// Test Write uses explicit token override when provided.
func TestWrite_TokenOverride(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	err := log.Write("testdir", "status", "task-status", "short", log.WriteOpts{
		Tokens: 1000, // Override regardless of message length
	}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.TokensEstimate).To(Equal(1000))
}

// TEST-502 traces: TASK-027
// Test token estimate rounds up for partial tokens.
func TestWrite_TokenEstimateRoundsUp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	// 5 characters / 4 = 1.25, should round up to 2
	message := "hello"
	g.Expect(len(message)).To(Equal(5))

	err := log.Write("testdir", "status", "task-status", message, log.WriteOpts{}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.TokensEstimate).To(Equal(2))
}

// TEST-503 traces: TASK-027
// Test token estimate for empty message is zero.
func TestWrite_TokenEstimateEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	err := log.Write("testdir", "status", "task-status", "", log.WriteOpts{}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.TokensEstimate).To(Equal(0))
}

// TEST-610 traces: TASK-061
// Test Write includes context estimate field when provided.
func TestWrite_ContextEstimate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	err := log.Write("testdir", "status", "task-status", "skill dispatched", log.WriteOpts{
		ContextEstimate: 45000, // 45% of 100K context
	}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.ContextEstimate).To(Equal(45000))
}

// TEST-611 traces: TASK-061
// Test Write omits context estimate when zero (backwards compatible).
func TestWrite_ContextEstimateOmittedWhenZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	err := log.Write("testdir", "status", "task-status", "no context tracked", log.WriteOpts{}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	line := strings.TrimSpace(string(content))
	g.Expect(line).ToNot(ContainSubstring(`"context_estimate"`))
}
