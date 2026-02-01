package corrections_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/corrections"
	"pgregory.net/rapid"
)

func fixedTime() time.Time {
	return time.Date(2026, 2, 1, 14, 30, 0, 0, time.UTC)
}

func nowFunc() func() time.Time {
	return func() time.Time { return fixedTime() }
}

// TEST-700 traces: TASK-040
// Test Log appends correction entry to project-specific corrections.jsonl
func TestLog_AppendsToProjectFile(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := corrections.Log(dir, "Variable renamed incorrectly", "refactoring foo to bar", corrections.LogOpts{
		SessionID: "sess-001",
	}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, "corrections.jsonl"))
	g.Expect(err).ToNot(HaveOccurred())

	var entry corrections.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry) // strip trailing newline
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.Timestamp).To(Equal("2026-02-01T14:30:00Z"))
	g.Expect(entry.Message).To(Equal("Variable renamed incorrectly"))
	g.Expect(entry.Context).To(Equal("refactoring foo to bar"))
	g.Expect(entry.SessionID).To(Equal("sess-001"))
}

// TEST-701 traces: TASK-040
// Test Log creates corrections.jsonl if it doesn't exist
func TestLog_CreatesFileIfMissing(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := corrections.Log(dir, "First correction", "initial context", corrections.LogOpts{}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	_, err = os.Stat(filepath.Join(dir, "corrections.jsonl"))
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-702 traces: TASK-040
// Test Log appends multiple entries as separate lines
func TestLog_AppendsMultipleEntries(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := corrections.Log(dir, "first", "context1", corrections.LogOpts{SessionID: "sess-1"}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	err = corrections.Log(dir, "second", "context2", corrections.LogOpts{SessionID: "sess-2"}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, "corrections.jsonl"))
	g.Expect(err).ToNot(HaveOccurred())

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	g.Expect(lines).To(HaveLen(2))

	// Both lines should be valid JSON
	for _, line := range lines {
		var entry corrections.Entry
		err := json.Unmarshal([]byte(line), &entry)
		g.Expect(err).ToNot(HaveOccurred())
	}
}

// TEST-703 traces: TASK-040
// Test Log omits session_id when empty (backwards compatible)
func TestLog_OmitsEmptySessionID(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := corrections.Log(dir, "correction without session", "some context", corrections.LogOpts{}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, "corrections.jsonl"))
	g.Expect(err).ToNot(HaveOccurred())

	line := strings.TrimSpace(string(content))
	g.Expect(line).ToNot(ContainSubstring(`"session_id"`))
}

// TEST-704 traces: TASK-040
// Test Log property: valid entries are always parseable
func TestLog_Property_ValidEntries(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		message := rapid.String().Draw(rt, "message")
		context := rapid.String().Draw(rt, "context")
		sessionID := rapid.String().Draw(rt, "session_id")

		err := corrections.Log(dir, message, context, corrections.LogOpts{
			SessionID: sessionID,
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify it's valid JSONL
		content, err := os.ReadFile(filepath.Join(dir, "corrections.jsonl"))
		g.Expect(err).ToNot(HaveOccurred())

		var entry corrections.Entry
		err = json.Unmarshal([]byte(strings.TrimSpace(string(content))), &entry)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(entry.Message).To(Equal(message))
		g.Expect(entry.Context).To(Equal(context))
	})
}

// TEST-705 traces: TASK-040
// Test LogGlobal appends to ~/.claude/corrections.jsonl
func TestLogGlobal_AppendsToGlobalFile(t *testing.T) {
	g := NewWithT(t)
	homeDir := t.TempDir()

	err := corrections.LogGlobal("correction message", "global context", corrections.LogOpts{
		SessionID: "sess-global",
	}, homeDir, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	expectedPath := filepath.Join(homeDir, ".claude", "corrections.jsonl")
	content, err := os.ReadFile(expectedPath)
	g.Expect(err).ToNot(HaveOccurred())

	var entry corrections.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.Message).To(Equal("correction message"))
	g.Expect(entry.Context).To(Equal("global context"))
	g.Expect(entry.SessionID).To(Equal("sess-global"))
}

// TEST-706 traces: TASK-040
// Test LogGlobal creates .claude directory if missing
func TestLogGlobal_CreatesClaudeDir(t *testing.T) {
	g := NewWithT(t)
	homeDir := t.TempDir()

	err := corrections.LogGlobal("test", "context", corrections.LogOpts{}, homeDir, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	claudeDir := filepath.Join(homeDir, ".claude")
	info, err := os.Stat(claudeDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(info.IsDir()).To(BeTrue())
}

// TEST-707 traces: TASK-040
// Test Read returns all entries from corrections file
func TestRead_AllEntries(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Write some corrections
	_ = corrections.Log(dir, "first", "ctx1", corrections.LogOpts{SessionID: "sess-1"}, nowFunc())
	_ = corrections.Log(dir, "second", "ctx2", corrections.LogOpts{SessionID: "sess-2"}, nowFunc())

	entries, err := corrections.Read(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(HaveLen(2))
	g.Expect(entries[0].Message).To(Equal("first"))
	g.Expect(entries[1].Message).To(Equal("second"))
}

// TEST-708 traces: TASK-040
// Test Read returns empty slice when file missing
func TestRead_NoFile(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	entries, err := corrections.Read(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

// TEST-709 traces: TASK-040
// Test ReadGlobal reads from ~/.claude/corrections.jsonl
func TestReadGlobal_ReadsFromGlobalFile(t *testing.T) {
	g := NewWithT(t)
	homeDir := t.TempDir()

	// Write a global correction
	_ = corrections.LogGlobal("global msg", "global ctx", corrections.LogOpts{SessionID: "sess-g"}, homeDir, nowFunc())

	entries, err := corrections.ReadGlobal(homeDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Message).To(Equal("global msg"))
	g.Expect(entries[0].SessionID).To(Equal("sess-g"))
}

// TEST-710 traces: TASK-040
// Test ReadGlobal returns empty slice when file missing
func TestReadGlobal_NoFile(t *testing.T) {
	g := NewWithT(t)
	homeDir := t.TempDir()

	entries, err := corrections.ReadGlobal(homeDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}
