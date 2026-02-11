package corrections_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/corrections"
	"pgregory.net/rapid"
)

// MockFS implements corrections.FileSystem for testing
type MockFS struct {
	Files map[string][]byte
	Dirs  map[string]bool
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

func (m *MockFS) MkdirAll(path string) error {
	if m.Dirs == nil {
		m.Dirs = make(map[string]bool)
	}
	m.Dirs[path] = true
	return nil
}

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
	fs := &MockFS{}

	err := corrections.Log("testdir", "Variable renamed incorrectly", "refactoring foo to bar", corrections.LogOpts{
		SessionID: "sess-001",
	}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", "corrections.jsonl"))
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
	fs := &MockFS{}

	err := corrections.Log("testdir", "First correction", "initial context", corrections.LogOpts{}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(fs.FileExists(filepath.Join("testdir", "corrections.jsonl"))).To(BeTrue())
}

// TEST-702 traces: TASK-040
// Test Log appends multiple entries as separate lines
func TestLog_AppendsMultipleEntries(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	err := corrections.Log("testdir", "first", "context1", corrections.LogOpts{SessionID: "sess-1"}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	err = corrections.Log("testdir", "second", "context2", corrections.LogOpts{SessionID: "sess-2"}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", "corrections.jsonl"))
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
	fs := &MockFS{}

	err := corrections.Log("testdir", "correction without session", "some context", corrections.LogOpts{}, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", "corrections.jsonl"))
	g.Expect(err).ToNot(HaveOccurred())

	line := strings.TrimSpace(string(content))
	g.Expect(line).ToNot(ContainSubstring(`"session_id"`))
}

// TEST-704 traces: TASK-040
// Test Log property: valid entries are always parseable
func TestLog_Property_ValidEntries(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		fs := &MockFS{}
		message := rapid.String().Draw(rt, "message")
		context := rapid.String().Draw(rt, "context")
		sessionID := rapid.String().Draw(rt, "session_id")

		err := corrections.Log("testdir", message, context, corrections.LogOpts{
			SessionID: sessionID,
		}, nowFunc(), fs)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify it's valid JSONL
		content, err := fs.ReadFile(filepath.Join("testdir", "corrections.jsonl"))
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
	homeDir := "testhome"
	fs := &MockFS{}

	err := corrections.LogGlobal("correction message", "global context", corrections.LogOpts{
		SessionID: "sess-global",
	}, homeDir, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	expectedPath := filepath.Join(homeDir, ".claude", "corrections.jsonl")
	content, err := fs.ReadFile(expectedPath)
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
	homeDir := "testhome"
	fs := &MockFS{}

	err := corrections.LogGlobal("test", "context", corrections.LogOpts{}, homeDir, nowFunc(), fs)
	g.Expect(err).ToNot(HaveOccurred())

	claudeDir := filepath.Join(homeDir, ".claude")
	g.Expect(fs.Dirs[claudeDir]).To(BeTrue())
}

// TEST-707 traces: TASK-040
// Test Read returns all entries from corrections file
func TestRead_AllEntries(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	// Write some corrections
	_ = corrections.Log("testdir", "first", "ctx1", corrections.LogOpts{SessionID: "sess-1"}, nowFunc(), fs)
	_ = corrections.Log("testdir", "second", "ctx2", corrections.LogOpts{SessionID: "sess-2"}, nowFunc(), fs)

	entries, err := corrections.Read("testdir", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(HaveLen(2))
	g.Expect(entries[0].Message).To(Equal("first"))
	g.Expect(entries[1].Message).To(Equal("second"))
}

// TEST-708 traces: TASK-040
// Test Read returns empty slice when file missing
func TestRead_NoFile(t *testing.T) {
	g := NewWithT(t)
	fs := &MockFS{}

	entries, err := corrections.Read("testdir", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

// TEST-709 traces: TASK-040
// Test ReadGlobal reads from ~/.claude/corrections.jsonl
func TestReadGlobal_ReadsFromGlobalFile(t *testing.T) {
	g := NewWithT(t)
	homeDir := "testhome"
	fs := &MockFS{}

	// Write a global correction
	_ = corrections.LogGlobal("global msg", "global ctx", corrections.LogOpts{SessionID: "sess-g"}, homeDir, nowFunc(), fs)

	entries, err := corrections.ReadGlobal(homeDir, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Message).To(Equal("global msg"))
	g.Expect(entries[0].SessionID).To(Equal("sess-g"))
}

// TEST-710 traces: TASK-040
// Test ReadGlobal returns empty slice when file missing
func TestReadGlobal_NoFile(t *testing.T) {
	g := NewWithT(t)
	homeDir := "testhome"
	fs := &MockFS{}

	entries, err := corrections.ReadGlobal(homeDir, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}
