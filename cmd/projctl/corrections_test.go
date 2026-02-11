package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/corrections"
)

// TEST-711 traces: ARCH-055, REQ-008
// Test correctionsLog command with all flags creates entry
func TestCorrectionsLog_WithAllFlags(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// This will be the actual CLI call pattern once implemented
	// projctl corrections log --dir DIR --message TEXT --context CONTEXT --session SESSION

	// For now, verify the expected behavior through the internal API
	// The CLI wrapper should pass through to corrections.Log
	err := corrections.Log(dir, "Test correction", "Test context", corrections.LogOpts{
		SessionID: "test-session-001",
	}, time.Now, &corrections.RealFS{})

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

// TEST-712 traces: ARCH-055, REQ-008
// Test correctionsLog command without session flag (optional)
func TestCorrectionsLog_WithoutSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := corrections.Log(dir, "Message only", "Context only", corrections.LogOpts{}, time.Now, &corrections.RealFS{})
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(dir, "corrections.jsonl"))
	g.Expect(err).ToNot(HaveOccurred())

	line := strings.TrimSpace(string(content))
	g.Expect(line).ToNot(ContainSubstring(`"session_id"`))
}

// TEST-713 traces: ARCH-055, REQ-008
// Test corrections file location convention
func TestCorrectionsLog_FileLocation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Project-specific: {dir}/corrections.jsonl
	err := corrections.Log(dir, "project correction", "ctx", corrections.LogOpts{}, time.Now, &corrections.RealFS{})
	g.Expect(err).ToNot(HaveOccurred())

	projectPath := filepath.Join(dir, "corrections.jsonl")
	_, err = os.Stat(projectPath)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-714 traces: ARCH-055, REQ-008
// Test global corrections file location
func TestCorrectionsLog_GlobalFileLocation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	homeDir := t.TempDir()

	// Global: ~/.claude/corrections.jsonl
	err := corrections.LogGlobal("global correction", "ctx", corrections.LogOpts{}, homeDir, time.Now, &corrections.RealFS{})
	g.Expect(err).ToNot(HaveOccurred())

	globalPath := filepath.Join(homeDir, ".claude", "corrections.jsonl")
	_, err = os.Stat(globalPath)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-730 traces: ARCH-061, REQ-014
// Test correctionsAnalyze command identifies patterns
func TestCorrectionsAnalyze_IdentifiesPatterns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create repeated corrections
	_ = corrections.Log(dir, "Never amend pushed commits", "git workflow", corrections.LogOpts{}, time.Now, &corrections.RealFS{})
	_ = corrections.Log(dir, "Never amend pushed commits", "git workflow", corrections.LogOpts{}, time.Now, &corrections.RealFS{})
	_ = corrections.Log(dir, "Check VCS type first", "vcs", corrections.LogOpts{}, time.Now, &corrections.RealFS{})

	// This will be the actual CLI call pattern once implemented
	// projctl corrections analyze --dir DIR --min-occurrences N

	// For now, verify expected behavior through internal API
	patterns, err := corrections.Analyze(dir, corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, &corrections.RealFS{})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))
	g.Expect(patterns[0].Count).To(Equal(2))
	g.Expect(patterns[0].Message).To(Equal("Never amend pushed commits"))
	g.Expect(patterns[0].Proposal).ToNot(BeEmpty())
}

// TEST-731 traces: ARCH-061, REQ-014
// Test correctionsAnalyze respects min-occurrences flag
func TestCorrectionsAnalyze_MinOccurrencesFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create corrections with different frequencies
	_ = corrections.Log(dir, "Pattern A", "ctx", corrections.LogOpts{}, time.Now, &corrections.RealFS{})
	_ = corrections.Log(dir, "Pattern A", "ctx", corrections.LogOpts{}, time.Now, &corrections.RealFS{})
	_ = corrections.Log(dir, "Pattern B", "ctx", corrections.LogOpts{}, time.Now, &corrections.RealFS{})
	_ = corrections.Log(dir, "Pattern B", "ctx", corrections.LogOpts{}, time.Now, &corrections.RealFS{})
	_ = corrections.Log(dir, "Pattern B", "ctx", corrections.LogOpts{}, time.Now, &corrections.RealFS{})

	// With min-occurrences=3, should only find Pattern B
	patterns, err := corrections.Analyze(dir, corrections.AnalyzeOpts{
		MinOccurrences: 3,
	}, &corrections.RealFS{})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))
	g.Expect(patterns[0].Message).To(Equal("Pattern B"))
	g.Expect(patterns[0].Count).To(Equal(3))
}

// TEST-732 traces: ARCH-061, REQ-014
// Test correctionsAnalyze with global corrections
func TestCorrectionsAnalyze_GlobalCorrections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	homeDir := t.TempDir()

	// Log global corrections
	_ = corrections.LogGlobal("Global pattern", "context", corrections.LogOpts{}, homeDir, time.Now, &corrections.RealFS{})
	_ = corrections.LogGlobal("Global pattern", "context", corrections.LogOpts{}, homeDir, time.Now, &corrections.RealFS{})

	// Analyze should work with global corrections too
	// Once CLI is implemented: projctl corrections analyze (without --dir flag)
	claudeDir := homeDir + "/.claude"
	patterns, err := corrections.Analyze(claudeDir, corrections.AnalyzeOpts{
		MinOccurrences: 2,
	}, &corrections.RealFS{})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))
	g.Expect(patterns[0].Message).To(Equal("Global pattern"))
}
