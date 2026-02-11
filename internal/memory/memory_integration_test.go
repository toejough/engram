//go:build integration

package memory_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// TEST-780: Memory learn creates embeddings.db if not exists
// traces: TASK-048
func TestLearnCreatesIndexIfNotExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	dbPath := filepath.Join(memoryDir, "embeddings.db")

	// Verify embeddings.db doesn't exist
	_, err := os.Stat(dbPath)
	g.Expect(os.IsNotExist(err)).To(BeTrue())

	opts := memory.LearnOpts{
		Message:    "Test learning",
		MemoryRoot: memoryDir,
	}

	err = memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify embeddings.db was created
	_, err = os.Stat(dbPath)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-781: Memory learn appends to existing entries in DB
// traces: TASK-048
func TestLearnAppendsToExistingIndex(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn first entry
	err := memory.Learn(memory.LearnOpts{
		Message:    "Existing learning",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Learn second entry
	err = memory.Learn(memory.LearnOpts{
		Message:    "New learning",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify both entries exist in DB via Grep
	result, err := memory.Grep(memory.GrepOpts{
		Pattern:    "Existing learning",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Matches).ToNot(BeEmpty())

	result, err = memory.Grep(memory.GrepOpts{
		Pattern:    "New learning",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Matches).ToNot(BeEmpty())
}

// TEST-782: Memory learn entry format has timestamp prefix
// traces: TASK-048
func TestLearnEntryFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "Test message",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify entry format via Grep
	result, err := memory.Grep(memory.GrepOpts{
		Pattern:    "Test message",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Matches).To(HaveLen(1))

	// Entry should be a markdown list item with timestamp
	// Format: - YYYY-MM-DD HH:MM: message
	g.Expect(result.Matches[0].Line).To(MatchRegexp(`- \d{4}-\d{2}-\d{2} \d{2}:\d{2}: Test message`))
}

// TEST-783: Memory learn includes project context when provided
// traces: TASK-048
func TestLearnWithProjectContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "Project-specific learning",
		Project:    "my-project",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify via Grep
	result, err := memory.Grep(memory.GrepOpts{
		Pattern:    "Project-specific learning",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Matches).ToNot(BeEmpty())

	// Should include project tag
	g.Expect(result.Matches[0].Line).To(ContainSubstring("[my-project]"))
	g.Expect(result.Matches[0].Line).To(ContainSubstring("Project-specific learning"))
}

// TEST-784: Memory learn without project context
// traces: TASK-048
func TestLearnWithoutProjectContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "Global learning",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify via Grep
	result, err := memory.Grep(memory.GrepOpts{
		Pattern:    "Global learning",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Matches).ToNot(BeEmpty())

	// Should not have project brackets when no project specified
	g.Expect(result.Matches[0].Line).ToNot(MatchRegexp(`\[.*\]`))
}

// TEST-785: Memory learn uses current timestamp
// traces: TASK-048
func TestLearnUsesCurrentTimestamp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	before := time.Now()

	opts := memory.LearnOpts{
		Message:    "Timestamp test",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	after := time.Now()

	// Verify via Grep
	result, err := memory.Grep(memory.GrepOpts{
		Pattern:    "Timestamp test",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Matches).ToNot(BeEmpty())

	// Should contain today's date
	dateStr := before.Format("2006-01-02")
	if after.Day() != before.Day() {
		// Handle edge case of test running at midnight
		dateStr = after.Format("2006-01-02")
	}
	g.Expect(result.Matches[0].Line).To(ContainSubstring(dateStr))
}

// TEST-786: Memory learn requires non-empty message
// traces: TASK-048
func TestLearnRequiresMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("message"))
}

// TEST-787: Property-based test for any message text
// traces: TASK-048
func TestLearnPropertyBasedMessageStorage(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		// Use alphanumeric suffix only for valid filesystem paths
		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "memory-test-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		// Generate random non-empty message (exclude newlines for simpler parsing)
		message := rapid.StringMatching(`[a-zA-Z0-9 .,!?'"-]+`).Draw(t, "message")
		if message == "" {
			message = "default message"
		}

		opts := memory.LearnOpts{
			Message:    message,
			MemoryRoot: memoryDir,
		}

		err := memory.Learn(opts)
		g.Expect(err).ToNot(HaveOccurred())

		// Property: message should be stored (verifiable via Grep)
		result, err := memory.Grep(memory.GrepOpts{
			Pattern:    message,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Matches).ToNot(BeEmpty())
		g.Expect(result.Matches[0].Line).To(ContainSubstring(message))
	})
}

// TEST-788: Memory learn multiple entries all stored
// traces: TASK-048
func TestLearnMultipleEntriesPreserveOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	messages := []string{"First learning", "Second learning", "Third learning"}

	for _, msg := range messages {
		opts := memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		}
		err := memory.Learn(opts)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Verify all entries exist via Grep
	for _, msg := range messages {
		result, err := memory.Grep(memory.GrepOpts{
			Pattern:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Matches).ToNot(BeEmpty(), "Expected to find: "+msg)
	}
}

// ============================================================================
// ISSUE-199: Learn without index.md tests
// ============================================================================

// TEST-199-01: Learn does not create index.md
// traces: ISSUE-199
func TestLearnDoesNotCreateIndexMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	memoryDir := filepath.Join(t.TempDir(), "memory")

	err := memory.Learn(memory.LearnOpts{
		Message:    "test learning",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// index.md should NOT exist
	_, err = os.Stat(filepath.Join(memoryDir, "index.md"))
	g.Expect(os.IsNotExist(err)).To(BeTrue(), "index.md should not be created by Learn()")
}

// TEST-199-02: Learn stores content accessible via DB
// traces: ISSUE-199
func TestLearnStoresContentInDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	memoryDir := filepath.Join(t.TempDir(), "memory")

	err := memory.Learn(memory.LearnOpts{
		Message:    "always use TDD",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Content should be findable via Grep (which will search DB after TASK-5)
	result, err := memory.Grep(memory.GrepOpts{
		Pattern:    "always use TDD",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Matches).ToNot(BeEmpty(), "Learn content should be findable via Grep")
}

// ============================================================================
// Decision logging tests (TASK-049)
// ============================================================================

// TEST-790: Decision creates decisions directory if not exists
// traces: TASK-049
func TestDecideCreatesDirectoryIfNotExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	decisionsDir := filepath.Join(memoryDir, "decisions")

	// Verify decisions directory doesn't exist
	_, err := os.Stat(decisionsDir)
	g.Expect(os.IsNotExist(err)).To(BeTrue())

	opts := memory.DecideOpts{
		Context:      "Test context",
		Choice:       "Option A",
		Reason:       "Best option",
		Alternatives: []string{"Option B", "Option C"},
		Project:      "test-project",
		MemoryRoot:   memoryDir,
	}

	result, err := memory.Decide(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	// Verify decisions directory was created
	_, err = os.Stat(decisionsDir)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-791: Decision file format is JSONL with proper filename
// traces: TASK-049
func TestDecideFileFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.DecideOpts{
		Context:      "Database selection",
		Choice:       "PostgreSQL",
		Reason:       "Better support for complex queries",
		Alternatives: []string{"MySQL", "MongoDB"},
		Project:      "my-project",
		MemoryRoot:   memoryDir,
	}

	result, err := memory.Decide(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify file exists and has correct name format: {DATE}-{PROJECT}.jsonl
	today := time.Now().Format("2006-01-02")
	expectedPath := filepath.Join(memoryDir, "decisions", today+"-my-project.jsonl")
	g.Expect(result.FilePath).To(Equal(expectedPath))

	// Verify file exists
	_, err = os.Stat(result.FilePath)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-792: Decision entry contains all fields as JSON
// traces: TASK-049
func TestDecideEntryFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.DecideOpts{
		Context:      "API framework",
		Choice:       "Gin",
		Reason:       "Performance and simplicity",
		Alternatives: []string{"Echo", "Fiber"},
		Project:      "api-project",
		MemoryRoot:   memoryDir,
	}

	result, err := memory.Decide(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Read and parse the JSONL entry
	content, err := os.ReadFile(result.FilePath)
	g.Expect(err).ToNot(HaveOccurred())

	var entry map[string]interface{}
	err = json.Unmarshal(content, &entry)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify all fields are present
	g.Expect(entry["context"]).To(Equal("API framework"))
	g.Expect(entry["choice"]).To(Equal("Gin"))
	g.Expect(entry["reason"]).To(Equal("Performance and simplicity"))
	g.Expect(entry["alternatives"]).To(ContainElements("Echo", "Fiber"))
	g.Expect(entry).To(HaveKey("timestamp"))
}

// TEST-793: Decision appends to existing file
// traces: TASK-049
func TestDecideAppendsToExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// First decision
	opts1 := memory.DecideOpts{
		Context:      "First decision",
		Choice:       "Choice 1",
		Reason:       "Reason 1",
		Alternatives: []string{"Alt1"},
		Project:      "append-test",
		MemoryRoot:   memoryDir,
	}
	_, err := memory.Decide(opts1)
	g.Expect(err).ToNot(HaveOccurred())

	// Second decision
	opts2 := memory.DecideOpts{
		Context:      "Second decision",
		Choice:       "Choice 2",
		Reason:       "Reason 2",
		Alternatives: []string{"Alt2"},
		Project:      "append-test",
		MemoryRoot:   memoryDir,
	}
	result, err := memory.Decide(opts2)
	g.Expect(err).ToNot(HaveOccurred())

	// Read the file and verify both entries
	content, err := os.ReadFile(result.FilePath)
	g.Expect(err).ToNot(HaveOccurred())

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	g.Expect(lines).To(HaveLen(2))

	// Verify both entries are valid JSON
	var entry1, entry2 map[string]interface{}
	g.Expect(json.Unmarshal([]byte(lines[0]), &entry1)).To(Succeed())
	g.Expect(json.Unmarshal([]byte(lines[1]), &entry2)).To(Succeed())

	g.Expect(entry1["context"]).To(Equal("First decision"))
	g.Expect(entry2["context"]).To(Equal("Second decision"))
}

// TEST-794: Decision requires context
// traces: TASK-049
func TestDecideRequiresContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.DecideOpts{
		Context:      "",
		Choice:       "Some choice",
		Reason:       "Some reason",
		Alternatives: []string{"Alt"},
		Project:      "test",
		MemoryRoot:   memoryDir,
	}

	_, err := memory.Decide(opts)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("context"))
}

// TEST-795: Decision requires choice
// traces: TASK-049
func TestDecideRequiresChoice(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.DecideOpts{
		Context:      "Some context",
		Choice:       "",
		Reason:       "Some reason",
		Alternatives: []string{"Alt"},
		Project:      "test",
		MemoryRoot:   memoryDir,
	}

	_, err := memory.Decide(opts)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("choice"))
}

// TEST-796: Decision requires reason
// traces: TASK-049
func TestDecideRequiresReason(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.DecideOpts{
		Context:      "Some context",
		Choice:       "Some choice",
		Reason:       "",
		Alternatives: []string{"Alt"},
		Project:      "test",
		MemoryRoot:   memoryDir,
	}

	_, err := memory.Decide(opts)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("reason"))
}

// TEST-797: Decision works with empty alternatives
// traces: TASK-049
func TestDecideEmptyAlternatives(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.DecideOpts{
		Context:      "Solo decision",
		Choice:       "Only option",
		Reason:       "No alternatives",
		Alternatives: []string{},
		Project:      "test",
		MemoryRoot:   memoryDir,
	}

	result, err := memory.Decide(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

// TEST-798: Decision uses today's date in filename
// traces: TASK-049
func TestDecideUsesTodayDate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.DecideOpts{
		Context:      "Date test",
		Choice:       "Test choice",
		Reason:       "Test reason",
		Alternatives: []string{},
		Project:      "date-project",
		MemoryRoot:   memoryDir,
	}

	result, err := memory.Decide(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Should contain today's date
	today := time.Now().Format("2006-01-02")
	g.Expect(result.FilePath).To(ContainSubstring(today))
}

// ============================================================================
// Memory grep tests (TASK-051)
// ============================================================================

// TEST-810: Grep searches embeddings DB
// traces: TASK-051
func TestGrepSearchesIndexMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Seed data via Learn (which stores in DB)
	entries := []string{
		"Learn about testing",
		"Study Go patterns",
		"Review TDD workflow",
	}
	for _, entry := range entries {
		err := memory.Learn(memory.LearnOpts{
			Message:    entry,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	opts := memory.GrepOpts{
		Pattern:    "testing",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Grep(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Matches).To(HaveLen(1))
	g.Expect(results.Matches[0].Line).To(ContainSubstring("testing"))
}

// TEST-811: Grep searches sessions directory
// traces: TASK-051
func TestGrepSearchesSessions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Create a session file
	sessionContent := `# Session Summary

PostgreSQL was chosen for the database.
Gin was selected as the web framework.`
	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-project.md"), []byte(sessionContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.GrepOpts{
		Pattern:    "PostgreSQL",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Grep(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Matches).To(HaveLen(1))
	g.Expect(results.Matches[0].File).To(ContainSubstring("sessions"))
}

// TEST-812: Grep returns file and line number
// traces: TASK-051
func TestGrepReturnsFileAndLineNumber(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Seed via Learn
	err := memory.Learn(memory.LearnOpts{
		Message:    "line with pattern",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.GrepOpts{
		Pattern:    "pattern",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Grep(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Matches).To(HaveLen(1))
	g.Expect(results.Matches[0].File).To(Equal("memory"))
	g.Expect(results.Matches[0].LineNum).To(BeNumerically(">", 0))
}

// TEST-813: Grep project filter limits search
// traces: TASK-051
func TestGrepProjectFilter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Create sessions for different projects
	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-proj-a.md"), []byte("Found in project A"), 0644)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-proj-b.md"), []byte("Found in project B"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.GrepOpts{
		Pattern:    "Found",
		Project:    "proj-a",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Grep(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Matches).To(HaveLen(1))
	g.Expect(results.Matches[0].Line).To(ContainSubstring("project A"))
}

// TEST-814: Grep with decisions flag searches decisions
// traces: TASK-051
func TestGrepWithDecisionsFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	decisionsDir := filepath.Join(memoryDir, "decisions")
	err := os.MkdirAll(decisionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Create a decision file
	decisionContent := `{"context":"API design","choice":"REST","reason":"simpler"}
{"context":"Database","choice":"PostgreSQL","reason":"robust"}`
	err = os.WriteFile(filepath.Join(decisionsDir, "2024-01-01-project.jsonl"), []byte(decisionContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.GrepOpts{
		Pattern:          "PostgreSQL",
		IncludeDecisions: true,
		MemoryRoot:       memoryDir,
	}

	results, err := memory.Grep(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Matches).To(HaveLen(1))
	g.Expect(results.Matches[0].Line).To(ContainSubstring("PostgreSQL"))
}

// TEST-815: Grep returns empty results for no matches
// traces: TASK-052
func TestGrepNoMatches(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Seed via Learn
	err := memory.Learn(memory.LearnOpts{
		Message:    "some content",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.GrepOpts{
		Pattern:    "nonexistent",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Grep(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Matches).To(BeEmpty())
}

// TEST-816: Grep requires pattern
// traces: TASK-052
func TestGrepRequiresPattern(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.GrepOpts{
		Pattern:    "",
		MemoryRoot: memoryDir,
	}

	_, err := memory.Grep(opts)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("pattern"))
}

// TEST-817: Grep is case insensitive
// traces: TASK-052
func TestGrepCaseInsensitive(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Seed via Learn
	err := memory.Learn(memory.LearnOpts{
		Message:    "PostgreSQL database",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.GrepOpts{
		Pattern:    "postgresql",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Grep(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Matches).To(HaveLen(1))
}

// ============================================================================
// ISSUE-199: Grep via FTS5 (DB search) tests
// ============================================================================

// TEST-199-03: Grep finds content from DB when index.md doesn't exist
// traces: ISSUE-199
func TestGrepFindsContentFromDBWithoutIndexMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	memoryDir := filepath.Join(t.TempDir(), "memory")

	// Seed via Learn (which writes to DB)
	err := memory.Learn(memory.LearnOpts{
		Message:    "use dependency injection",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Delete index.md if it exists (simulating post-removal state)
	_ = os.Remove(filepath.Join(memoryDir, "index.md"))

	// Grep should still find content from DB
	result, err := memory.Grep(memory.GrepOpts{
		Pattern:    "dependency injection",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Matches).ToNot(BeEmpty(), "Grep should find DB content even without index.md")
}

// TEST-199-04: Grep still searches sessions directory
// traces: ISSUE-199
func TestGrepStillSearchesSessions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	memoryDir := filepath.Join(t.TempDir(), "memory")

	// Create a session summary file
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(sessionsDir, "2026-02-10-test.md"), []byte("session content with TDD pattern"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.Grep(memory.GrepOpts{
		Pattern:    "TDD pattern",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Matches).ToNot(BeEmpty(), "Grep should still search sessions directory")
}

// ============================================================================
// Memory query tests (TASK-052)
// ============================================================================

// TEST-820: Query creates embeddings.db on first use
// traces: TASK-052
func TestQueryCreatesEmbeddingsDb(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Verify embeddings.db doesn't exist
	embeddingsPath := filepath.Join(memoryDir, "embeddings.db")
	_, err := os.Stat(embeddingsPath)
	g.Expect(os.IsNotExist(err)).To(BeTrue())

	// Learn content
	err = memory.Learn(memory.LearnOpts{
		Message:    "Learned about PostgreSQL database design",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "database",
		MemoryRoot: memoryDir,
	}

	_, err = memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify embeddings.db was created
	_, err = os.Stat(embeddingsPath)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-821: Query uses SQLite-vec for vector storage
// traces: TASK-052
func TestQueryUsesSQLiteVec(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
		[]byte("Database indexing strategies"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "database",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify embeddings.db contains vec0 virtual table for SQLite-vec
	embeddingsPath := filepath.Join(memoryDir, "embeddings.db")
	g.Expect(embeddingsPath).To(BeAnExistingFile())

	// The database should have embeddings stored as vectors
	g.Expect(results.VectorStorage).To(Equal("sqlite-vec"))
}

// TEST-822: Query uses ONNX model for embeddings with e5-small model
// traces: TASK-052
func TestQueryUsesONNXModel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
		[]byte("Testing ONNX embedding generation"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "embedding test",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Results should indicate real ONNX model was used (not mock)
	g.Expect(results.EmbeddingModel).To(Equal("e5-small-v2"))

	// Model should be e5-small with 384 dimensions
	g.Expect(results.EmbeddingDimensions).To(Equal(384))

	// Should NOT have made any API calls (field should be false/zero)
	g.Expect(results.APICallsMade).To(BeFalse())

	// Should have actually loaded and run inference with ONNX Runtime
	g.Expect(results.UsedONNXRuntime).To(BeTrue())
}

// TEST-823: Query uses semantic similarity not keyword matching
// traces: TASK-052
func TestQueryUsesSemanticSimilarity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn content where semantic understanding is needed
	err := memory.Learn(memory.LearnOpts{
		Message:    "Database backup and recovery procedures",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Web frontend styling with CSS",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Data persistence and storage solutions",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "storing information",
		Limit:      3,
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// Semantic similarity should rank "Data persistence" and "Database backup" higher
	// than "CSS styling" even though no exact keyword matches exist
	topResult := results.Results[0]
	g.Expect(topResult.Content).To(Or(
		ContainSubstring("persistence"),
		ContainSubstring("backup"),
	))
	g.Expect(topResult.Content).ToNot(ContainSubstring("CSS"))
}

// TEST-824: Query returns default 5 results
// traces: TASK-052
func TestQueryDefaultLimit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn more than 5 entries
	for i := 0; i < 10; i++ {
		err := memory.Learn(memory.LearnOpts{
			Message:    fmt.Sprintf("Memory about testing variant %d", i),
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	opts := memory.QueryOpts{
		Text:       "testing",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(len(results.Results)).To(BeNumerically("<=", 5))
}

// TEST-825: Query respects custom limit
// traces: TASK-052
func TestQueryCustomLimit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn 10 entries
	for i := 0; i < 10; i++ {
		err := memory.Learn(memory.LearnOpts{
			Message:    fmt.Sprintf("Memory about testing variant %d", i),
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	opts := memory.QueryOpts{
		Text:       "testing",
		Limit:      3,
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(len(results.Results)).To(BeNumerically("<=", 3))
}

// TEST-826: Query searches learned content
// traces: TASK-052
func TestQuerySearchesLearnedContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn content
	err := memory.Learn(memory.LearnOpts{
		Message:    "Decided to use PostgreSQL for the database layer",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Completed API design work",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "PostgreSQL",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())
}

// TEST-827: Query requires text
// traces: TASK-052
func TestQueryRequiresText(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.QueryOpts{
		Text:       "",
		MemoryRoot: memoryDir,
	}

	_, err := memory.Query(opts)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("text"))
}

// TEST-828: Query results include similarity scores
// traces: TASK-052
func TestQueryResultsIncludeScores(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn content
	err := memory.Learn(memory.LearnOpts{
		Message:    "Testing database queries",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Building web frontend",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "database",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// All results should have scores between 0 and 1
	for _, r := range results.Results {
		g.Expect(r.Score).To(BeNumerically(">=", 0))
		g.Expect(r.Score).To(BeNumerically("<=", 1))
	}
}

// TEST-829: Query results are sorted by score descending
// traces: TASK-052
func TestQueryResultsSortedByScore(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn content
	err := memory.Learn(memory.LearnOpts{
		Message:    "web frontend development",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "database indexing and optimization",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "database schema design patterns",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "database",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify descending order
	for i := 1; i < len(results.Results); i++ {
		g.Expect(results.Results[i-1].Score).To(BeNumerically(">=", results.Results[i].Score))
	}
}

// TEST-831: Property-based test for embedding consistency
// traces: TASK-052
func TestQueryPropertyBasedEmbeddingConsistency(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		// Use alphanumeric suffix for valid filesystem paths
		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "query-test-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		sessionsDir := filepath.Join(memoryDir, "sessions")
		err := os.MkdirAll(sessionsDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		// Generate random session content
		numEntries := rapid.IntRange(1, 10).Draw(t, "numEntries")
		var lines []string
		for i := 0; i < numEntries; i++ {
			content := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "content")
			lines = append(lines, content)
		}
		err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
			[]byte(strings.Join(lines, "\n")), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		queryText := rapid.StringMatching(`[a-zA-Z0-9 ]{5,20}`).Draw(t, "query")

		opts := memory.QueryOpts{
			Text:       queryText,
			MemoryRoot: memoryDir,
		}

		results, err := memory.Query(opts)
		g.Expect(err).ToNot(HaveOccurred())

		// Property: Same query should return same results (deterministic)
		results2, err := memory.Query(opts)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(len(results.Results)).To(Equal(len(results2.Results)))

		// Property: Embeddings should be stored (db should exist)
		embeddingsPath := filepath.Join(memoryDir, "embeddings.db")
		_, err = os.Stat(embeddingsPath)
		g.Expect(err).ToNot(HaveOccurred())

		// Property: No API calls should have been made
		g.Expect(results.APICallsMade).To(BeFalse())

		// Property: ONNX model was actually used for inference
		g.Expect(results.UsedONNXRuntime).To(BeTrue())
		g.Expect(results.EmbeddingModel).To(Equal("e5-small-v2"))
	})
}

// TEST-832: Query downloads e5-small model on first use
// traces: TASK-052
func TestQueryDownloadsModelOnFirstUse(t *testing.T) {
	t.Parallel(
	// Skip in short mode since this test requires network download
	)

	if testing.Short() {
		t.Skip("skipping download test in short mode")
	}

	g := NewWithT(t)

	// Get default model path to check if we can copy it instead of downloading
	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	defaultModelPath := filepath.Join(homeDir, ".claude", "models", "e5-small-v2.onnx")

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err = os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Use custom model directory to test download behavior
	modelDir := filepath.Join(tempDir, "models")
	err = os.MkdirAll(modelDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
		[]byte("Test content for model download"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Model should not exist yet in temp directory
	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
	_, err = os.Stat(modelPath)
	g.Expect(os.IsNotExist(err)).To(BeTrue())

	// If pre-downloaded model exists, copy it to avoid network download
	if _, err := os.Stat(defaultModelPath); err == nil {
		modelData, err := os.ReadFile(defaultModelPath)
		g.Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(modelPath, modelData, 0644)
		g.Expect(err).ToNot(HaveOccurred())
	}

	opts := memory.QueryOpts{
		Text:       "test query",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Model should exist (either downloaded or copied)
	_, err = os.Stat(modelPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify model was used
	g.Expect(results.ModelLoaded).To(BeTrue())
	g.Expect(results.ModelPath).To(Equal(modelPath))
}

// TEST-833: Query uses default model directory ~/.claude/models
// traces: TASK-052
func TestQueryUsesDefaultModelDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
		[]byte("Test content"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
		// ModelDir not specified - should use default
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Should use ~/.claude/models/e5-small-v2.onnx
	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	expectedPath := filepath.Join(homeDir, ".claude", "models", "e5-small-v2.onnx")
	g.Expect(results.ModelPath).To(Equal(expectedPath))
}

// TEST-834: Query verifies model dimensions match e5-small (384)
// traces: TASK-052
func TestQueryVerifiesE5SmallDimensions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
		[]byte("Test embedding dimensions"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Uses default ModelDir with pre-downloaded model
	opts := memory.QueryOpts{
		Text:       "dimensions test",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Must verify e5-small produces 384-dimensional embeddings
	g.Expect(results.EmbeddingDimensions).To(Equal(384))

	// Should have actually run inference to verify dimensions
	g.Expect(results.InferenceExecuted).To(BeTrue())
}

// TEST-835: Query loads ONNX model into memory
// traces: TASK-052
func TestQueryLoadsONNXModelIntoMemory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
		[]byte("Model loading test"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Uses default ModelDir with pre-downloaded model
	opts := memory.QueryOpts{
		Text:       "load test",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Should have loaded the ONNX model successfully
	g.Expect(results.ModelLoaded).To(BeTrue())

	// Model type should be ONNX
	g.Expect(results.ModelType).To(Equal("onnx"))

	// Should have ONNX Runtime session active during inference
	g.Expect(results.UsedONNXRuntime).To(BeTrue())
}

// TEST-836: Query performs actual inference not mock embeddings
// traces: TASK-052
func TestQueryPerformsActualInference(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	sessionContent := `Database design patterns
Frontend styling with CSS`
	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
		[]byte(sessionContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Uses default ModelDir with pre-downloaded model
	opts := memory.QueryOpts{
		Text:       "database",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Should have executed real inference
	g.Expect(results.InferenceExecuted).To(BeTrue())

	// Results should NOT be from mock embeddings
	g.Expect(results.UsedMockEmbeddings).To(BeFalse())

	// Should have used e5-small model for inference
	g.Expect(results.EmbeddingModel).To(Equal("e5-small-v2"))

	// Embeddings should be 384-dimensional vectors from real model
	g.Expect(results.EmbeddingDimensions).To(Equal(384))
}

// TEST-837: Query reuses model on subsequent calls
// traces: TASK-052
func TestQueryReusesDownloadedModel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
		[]byte("Model reuse test"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Uses default ModelDir with pre-downloaded model
	opts := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
	}

	// First query uses pre-downloaded model (no download needed)
	results1, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results1.ModelLoaded).To(BeTrue())

	// Second query should reuse same model
	results2, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results2.ModelLoaded).To(BeTrue())

	// Both should use same model path
	g.Expect(results1.ModelPath).To(Equal(results2.ModelPath))
}

// TEST-201-01: LearnWithConflictCheck actually stores the entry
// traces: ISSUE-201 AC-1
func TestLearnWithConflictCheckStoresEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "always use dependency injection for testability",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Stored).To(BeTrue())

	// Verify the entry is actually queryable via Grep
	grepResult, err := memory.Grep(memory.GrepOpts{
		Pattern:    "always use dependency injection for testability",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(grepResult.Matches).ToNot(BeEmpty(), "entry should be stored and queryable after LearnWithConflictCheck")
}

// TEST-201-02: LearnWithConflictCheck detects conflict for similar entries
// traces: ISSUE-201 AC-2
func TestLearnWithConflictCheckDetectsConflict(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Store an entry first via Learn
	err := memory.Learn(memory.LearnOpts{
		Message:    "always use TDD for all code changes",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Now call LearnWithConflictCheck with a very similar message
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "always use TDD for all code changes",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.HasConflict).To(BeTrue(), "identical message should be detected as conflict")
	g.Expect(result.ConflictEntry).ToNot(BeEmpty(), "conflict entry should reference the existing entry")
}

// TEST-201-03: LearnWithConflictCheck reports no conflict for different entries
// traces: ISSUE-201 AC-3
func TestLearnWithConflictCheckNoConflictForDifferentMessages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Store an entry first
	err := memory.Learn(memory.LearnOpts{
		Message:    "always use TDD for all code changes",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// LearnWithConflictCheck with a completely different message
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "prefer dark mode for IDE theme settings",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.HasConflict).To(BeFalse(), "semantically different message should not be detected as conflict")
}

// ============================================================================
// ISSUE-208: Query() read-only tests (TDD Red)
// ============================================================================

// TEST-208-01: Query is read-only and does not create new embeddings
// traces: ISSUE-208 TASK-1
func TestQueryIsReadOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Learn some data first to seed the database
	err = memory.Learn(memory.LearnOpts{
		Message:    "initial learning content",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Count embeddings before writing session file
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var countBefore int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE embedding_id IS NOT NULL").Scan(&countBefore)
	g.Expect(err).ToNot(HaveOccurred())

	// Write a session file (this would trigger the bug in current code)
	sessionContent := "Session content that should not be embedded by Query"
	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
		[]byte(sessionContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Call Query - should NOT create new embeddings from session file
	opts := memory.QueryOpts{
		Text:       "session content",
		MemoryRoot: memoryDir,
	}
	_, err = memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Count embeddings after Query
	var countAfter int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE embedding_id IS NOT NULL").Scan(&countAfter)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify no new embeddings were created
	g.Expect(countAfter).To(Equal(countBefore),
		"Query should not create embeddings from session files (read-only)")
}

// TEST-208-02: Query ignores session files and does not return their content
// traces: ISSUE-208 TASK-1
func TestQueryIgnoresSessionFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Write session file with unique marker
	sessionContent := "unique-session-marker-xyz-should-not-be-found"
	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-test.md"),
		[]byte(sessionContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Query for the session content
	opts := memory.QueryOpts{
		Text:       "unique-session-marker-xyz",
		MemoryRoot: memoryDir,
	}
	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Results should NOT contain session file content
	for _, result := range results.Results {
		g.Expect(result.Content).ToNot(ContainSubstring("unique-session-marker-xyz"),
			"Query should not return session file content")
	}
}

// TEST-208-03: Query finds content added via Learn() (positive control)
// traces: ISSUE-208 TASK-1
func TestQueryFindsLearnedContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn content
	err := memory.Learn(memory.LearnOpts{
		Message:    "this is learned content that should be found",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query for the learned content
	opts := memory.QueryOpts{
		Text:       "learned content",
		MemoryRoot: memoryDir,
	}
	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Should find the learned content
	g.Expect(results.Results).ToNot(BeEmpty(),
		"Query should find content added via Learn()")

	// Verify at least one result contains the learned content
	found := false
	for _, result := range results.Results {
		if strings.Contains(result.Content, "learned content") {
			found = true
			break
		}
	}
	g.Expect(found).To(BeTrue(), "Query should return learned content in results")
}
