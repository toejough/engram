package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/corrections"
)

// TEST-711 traces: TASK-040
// Test correctionsLog command with all flags creates entry
func TestCorrectionsLog_WithAllFlags(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// This will be the actual CLI call pattern once implemented
	// projctl corrections log --dir DIR --message TEXT --context CONTEXT --session SESSION

	// For now, verify the expected behavior through the internal API
	// The CLI wrapper should pass through to corrections.Log
	err := corrections.Log(dir, "Test correction", "Test context", corrections.LogOpts{
		SessionID: "test-session-001",
	}, nil)

	// This should fail because corrections package doesn't exist yet
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, "corrections.jsonl"))
	g.Expect(err).ToNot(HaveOccurred())

	var entry corrections.Entry
	err = json.Unmarshal(content[:len(content)-1], &entry)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entry.Message).To(Equal("Test correction"))
	g.Expect(entry.Context).To(Equal("Test context"))
	g.Expect(entry.SessionID).To(Equal("test-session-001"))
}

// TEST-712 traces: TASK-040
// Test correctionsLog command without session flag (optional)
func TestCorrectionsLog_WithoutSession(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	err := corrections.Log(dir, "Message only", "Context only", corrections.LogOpts{}, nil)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, "corrections.jsonl"))
	g.Expect(err).ToNot(HaveOccurred())

	line := strings.TrimSpace(string(content))
	g.Expect(line).ToNot(ContainSubstring(`"session_id"`))
}

// TEST-713 traces: TASK-040
// Test corrections file location convention
func TestCorrectionsLog_FileLocation(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Project-specific: {dir}/corrections.jsonl
	err := corrections.Log(dir, "project correction", "ctx", corrections.LogOpts{}, nil)
	g.Expect(err).ToNot(HaveOccurred())

	projectPath := filepath.Join(dir, "corrections.jsonl")
	_, err = os.Stat(projectPath)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-714 traces: TASK-040
// Test global corrections file location
func TestCorrectionsLog_GlobalFileLocation(t *testing.T) {
	g := NewWithT(t)
	homeDir := t.TempDir()

	// Global: ~/.claude/corrections.jsonl
	err := corrections.LogGlobal("global correction", "ctx", corrections.LogOpts{}, homeDir, nil)
	g.Expect(err).ToNot(HaveOccurred())

	globalPath := filepath.Join(homeDir, ".claude", "corrections.jsonl")
	_, err = os.Stat(globalPath)
	g.Expect(err).ToNot(HaveOccurred())
}
