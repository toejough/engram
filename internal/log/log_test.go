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

// TEST-410 traces: TASK-016
// Test Write includes model field in entry.
func TestWrite_ModelField(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := log.Write(dir, "status", "task-status", "dispatched", log.WriteOpts{
		Task:  "TASK-001",
		Model: "haiku",
	}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.Model).To(Equal("haiku"))
}

// TEST-411 traces: TASK-016
// Test Write omits model when empty (backwards compatible).
func TestWrite_ModelOmittedWhenEmpty(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := log.Write(dir, "status", "task-status", "started", log.WriteOpts{
		Task: "TASK-001",
	}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	line := strings.TrimSpace(string(content))
	g.Expect(line).ToNot(ContainSubstring(`"model"`))
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

// TEST-412 traces: TASK-016
// Test Read returns all entries when no filter.
func TestRead_AllEntries(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Write some entries
	_ = log.Write(dir, "status", "task-status", "first", log.WriteOpts{Task: "TASK-001"}, nowFunc())
	_ = log.Write(dir, "phase", "phase-change", "second", log.WriteOpts{}, nowFunc())

	entries, err := log.Read(dir, log.ReadOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(HaveLen(2))
	g.Expect(entries[0].Message).To(Equal("first"))
	g.Expect(entries[1].Message).To(Equal("second"))
}

// TEST-413 traces: TASK-016
// Test Read filters by model.
func TestRead_FilterByModel(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Write entries with different models
	_ = log.Write(dir, "status", "task-status", "haiku task", log.WriteOpts{Model: "haiku"}, nowFunc())
	_ = log.Write(dir, "status", "task-status", "sonnet task", log.WriteOpts{Model: "sonnet"}, nowFunc())
	_ = log.Write(dir, "status", "task-status", "opus task", log.WriteOpts{Model: "opus"}, nowFunc())
	_ = log.Write(dir, "status", "task-status", "no model", log.WriteOpts{}, nowFunc())

	// Filter by haiku
	entries, err := log.Read(dir, log.ReadOpts{Model: "haiku"})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Message).To(Equal("haiku task"))
	g.Expect(entries[0].Model).To(Equal("haiku"))
}

// TEST-414 traces: TASK-016
// Test Read returns empty slice when log file missing.
func TestRead_NoLogFile(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	entries, err := log.Read(dir, log.ReadOpts{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

// TEST-500 traces: TASK-027
// Test Write calculates token estimate from message length.
func TestWrite_TokenEstimate(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// 40 characters / 4 = 10 tokens
	message := "This is exactly forty characters long!!!"
	g.Expect(len(message)).To(Equal(40))

	err := log.Write(dir, "status", "task-status", message, log.WriteOpts{}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.TokensEstimate).To(Equal(10))
}

// TEST-501 traces: TASK-027
// Test Write uses explicit token override when provided.
func TestWrite_TokenOverride(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := log.Write(dir, "status", "task-status", "short", log.WriteOpts{
		Tokens: 1000, // Override regardless of message length
	}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.TokensEstimate).To(Equal(1000))
}

// TEST-502 traces: TASK-027
// Test token estimate rounds up for partial tokens.
func TestWrite_TokenEstimateRoundsUp(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// 5 characters / 4 = 1.25, should round up to 2
	message := "hello"
	g.Expect(len(message)).To(Equal(5))

	err := log.Write(dir, "status", "task-status", message, log.WriteOpts{}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.TokensEstimate).To(Equal(2))
}

// TEST-503 traces: TASK-027
// Test token estimate for empty message is zero.
func TestWrite_TokenEstimateEmpty(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := log.Write(dir, "status", "task-status", "", log.WriteOpts{}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.TokensEstimate).To(Equal(0))
}

// TEST-610 traces: TASK-061
// Test Write includes context estimate field when provided.
func TestWrite_ContextEstimate(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := log.Write(dir, "status", "task-status", "skill dispatched", log.WriteOpts{
		ContextEstimate: 45000, // 45% of 100K context
	}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	var entry log.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.ContextEstimate).To(Equal(45000))
}

// TEST-611 traces: TASK-061
// Test Write omits context estimate when zero (backwards compatible).
func TestWrite_ContextEstimateOmittedWhenZero(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := log.Write(dir, "status", "task-status", "no context tracked", log.WriteOpts{}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, log.LogFile))
	g.Expect(err).ToNot(HaveOccurred())

	line := strings.TrimSpace(string(content))
	g.Expect(line).ToNot(ContainSubstring(`"context_estimate"`))
}
