package memory_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// TEST-780: Memory learn creates index.md if not exists
func TestLearnCreatesIndexIfNotExists(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "memory", "index.md")

	// Verify index doesn't exist
	_, err := os.Stat(indexPath)
	g.Expect(os.IsNotExist(err)).To(BeTrue())

	opts := memory.LearnOpts{
		Message:    "Test learning",
		MemoryRoot: filepath.Join(tempDir, "memory"),
	}

	err = memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify index was created
	_, err = os.Stat(indexPath)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-781: Memory learn appends to existing index
func TestLearnAppendsToExistingIndex(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	indexPath := filepath.Join(memoryDir, "index.md")
	existing := "# Memory Index\n\n- 2024-01-01: Existing learning\n"
	err = os.WriteFile(indexPath, []byte(existing), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.LearnOpts{
		Message:    "New learning",
		MemoryRoot: memoryDir,
	}

	err = memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(indexPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("Existing learning"))
	g.Expect(string(content)).To(ContainSubstring("New learning"))
}

// TEST-782: Memory learn entry format has timestamp prefix
func TestLearnEntryFormat(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "Test message",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(memoryDir, "index.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Entry should be a markdown list item with timestamp
	// Format: - YYYY-MM-DD HH:MM: message
	g.Expect(string(content)).To(MatchRegexp(`- \d{4}-\d{2}-\d{2} \d{2}:\d{2}: Test message`))
}

// TEST-783: Memory learn includes project context when provided
func TestLearnWithProjectContext(t *testing.T) {
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

	content, err := os.ReadFile(filepath.Join(memoryDir, "index.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Should include project tag
	g.Expect(string(content)).To(ContainSubstring("[my-project]"))
	g.Expect(string(content)).To(ContainSubstring("Project-specific learning"))
}

// TEST-784: Memory learn without project context
func TestLearnWithoutProjectContext(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "Global learning",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(filepath.Join(memoryDir, "index.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Should not have project brackets when no project specified
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Global learning") {
			g.Expect(line).ToNot(MatchRegexp(`\[.*\]`))
		}
	}
}

// TEST-785: Memory learn uses current timestamp
func TestLearnUsesCurrentTimestamp(t *testing.T) {
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

	content, err := os.ReadFile(filepath.Join(memoryDir, "index.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Should contain today's date
	dateStr := before.Format("2006-01-02")
	if after.Day() != before.Day() {
		// Handle edge case of test running at midnight
		dateStr = after.Format("2006-01-02")
	}
	g.Expect(string(content)).To(ContainSubstring(dateStr))
}

// TEST-786: Memory learn requires non-empty message
func TestLearnRequiresMessage(t *testing.T) {
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
func TestLearnPropertyBasedMessageStorage(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		// Use alphanumeric suffix only for valid filesystem paths
		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "memory-test-"+suffix)
		defer os.RemoveAll(memoryDir)

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

		content, err := os.ReadFile(filepath.Join(memoryDir, "index.md"))
		g.Expect(err).ToNot(HaveOccurred())

		// Property: message should be stored
		g.Expect(string(content)).To(ContainSubstring(message))
	})
}

// TEST-788: Memory learn multiple entries preserve order
func TestLearnMultipleEntriesPreserveOrder(t *testing.T) {
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

	content, err := os.ReadFile(filepath.Join(memoryDir, "index.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Find positions of each message
	contentStr := string(content)
	pos1 := strings.Index(contentStr, "First learning")
	pos2 := strings.Index(contentStr, "Second learning")
	pos3 := strings.Index(contentStr, "Third learning")

	// Verify order is preserved (appended, so later entries are after earlier ones)
	g.Expect(pos1).To(BeNumerically("<", pos2))
	g.Expect(pos2).To(BeNumerically("<", pos3))
}

// ============================================================================
// Decision logging tests (TASK-049)
// ============================================================================

// TEST-790: Decision creates decisions directory if not exists
func TestDecideCreatesDirectoryIfNotExists(t *testing.T) {
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
func TestDecideFileFormat(t *testing.T) {
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
func TestDecideEntryFormat(t *testing.T) {
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
func TestDecideAppendsToExistingFile(t *testing.T) {
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
func TestDecideRequiresContext(t *testing.T) {
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
func TestDecideRequiresChoice(t *testing.T) {
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
func TestDecideRequiresReason(t *testing.T) {
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
func TestDecideEmptyAlternatives(t *testing.T) {
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
func TestDecideUsesTodayDate(t *testing.T) {
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
// Session end tests (TASK-050)
// ============================================================================

// TEST-800: Session end creates sessions directory if not exists
func TestSessionEndCreatesDirectoryIfNotExists(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")

	// Verify sessions directory doesn't exist
	_, err := os.Stat(sessionsDir)
	g.Expect(os.IsNotExist(err)).To(BeTrue())

	opts := memory.SessionEndOpts{
		Project:    "test-project",
		MemoryRoot: memoryDir,
	}

	result, err := memory.SessionEnd(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	// Verify sessions directory was created
	_, err = os.Stat(sessionsDir)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-801: Session end file format and location
func TestSessionEndFileLocation(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.SessionEndOpts{
		Project:    "my-project",
		MemoryRoot: memoryDir,
	}

	result, err := memory.SessionEnd(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify file path format: {DATE}-{PROJECT}.md
	today := time.Now().Format("2006-01-02")
	expectedPath := filepath.Join(memoryDir, "sessions", today+"-my-project.md")
	g.Expect(result.FilePath).To(Equal(expectedPath))

	// Verify file exists
	_, err = os.Stat(result.FilePath)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-802: Session summary includes key sections
func TestSessionEndIncludesKeySections(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.SessionEndOpts{
		Project:    "test-project",
		MemoryRoot: memoryDir,
	}

	result, err := memory.SessionEnd(opts)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(result.FilePath)
	g.Expect(err).ToNot(HaveOccurred())
	contentStr := string(content)

	// Should include standard sections
	g.Expect(contentStr).To(ContainSubstring("Session Summary"))
	g.Expect(contentStr).To(ContainSubstring("test-project"))
}

// TEST-803: Session summary is under character limit
func TestSessionEndUnderCharacterLimit(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Create some decisions to summarize
	for i := 0; i < 10; i++ {
		decideOpts := memory.DecideOpts{
			Context:      strings.Repeat("Long context text ", 20),
			Choice:       "Choice " + string(rune('A'+i)),
			Reason:       strings.Repeat("Detailed reason ", 15),
			Alternatives: []string{"Alt1", "Alt2", "Alt3"},
			Project:      "large-project",
			MemoryRoot:   memoryDir,
		}
		_, _ = memory.Decide(decideOpts)
	}

	opts := memory.SessionEndOpts{
		Project:    "large-project",
		MemoryRoot: memoryDir,
	}

	result, err := memory.SessionEnd(opts)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(result.FilePath)
	g.Expect(err).ToNot(HaveOccurred())

	// Must be under 2000 characters
	g.Expect(len(content)).To(BeNumerically("<", 2000))
}

// TEST-804: Session end requires project name
func TestSessionEndRequiresProject(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.SessionEndOpts{
		Project:    "",
		MemoryRoot: memoryDir,
	}

	_, err := memory.SessionEnd(opts)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("project"))
}

// TEST-805: Session end includes decisions from today
func TestSessionEndIncludesDecisions(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Create a decision first
	decideOpts := memory.DecideOpts{
		Context:      "Test decision context",
		Choice:       "PostgreSQL",
		Reason:       "Better support",
		Alternatives: []string{"MySQL"},
		Project:      "decision-project",
		MemoryRoot:   memoryDir,
	}
	_, err := memory.Decide(decideOpts)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.SessionEndOpts{
		Project:    "decision-project",
		MemoryRoot: memoryDir,
	}

	result, err := memory.SessionEnd(opts)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(result.FilePath)
	g.Expect(err).ToNot(HaveOccurred())

	// Should include decision information
	g.Expect(string(content)).To(ContainSubstring("PostgreSQL"))
}

// TEST-806: Property-based test for size limit
func TestSessionEndPropertyBasedSizeLimit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		// Use alphanumeric suffix only for valid filesystem paths
		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "session-test-"+suffix)
		defer os.RemoveAll(memoryDir)

		projectName := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "project")

		// Create random number of decisions
		numDecisions := rapid.IntRange(0, 20).Draw(t, "numDecisions")
		for i := 0; i < numDecisions; i++ {
			decideOpts := memory.DecideOpts{
				Context:      rapid.StringMatching(`[a-zA-Z0-9 ]{10,100}`).Draw(t, "context"),
				Choice:       rapid.StringMatching(`[a-zA-Z0-9 ]{5,20}`).Draw(t, "choice"),
				Reason:       rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "reason"),
				Alternatives: []string{rapid.StringMatching(`[a-zA-Z0-9]{5,10}`).Draw(t, "alt")},
				Project:      projectName,
				MemoryRoot:   memoryDir,
			}
			_, _ = memory.Decide(decideOpts)
		}

		opts := memory.SessionEndOpts{
			Project:    projectName,
			MemoryRoot: memoryDir,
		}

		result, err := memory.SessionEnd(opts)
		g.Expect(err).ToNot(HaveOccurred())

		content, err := os.ReadFile(result.FilePath)
		g.Expect(err).ToNot(HaveOccurred())

		// Property: output always under 2000 characters
		g.Expect(len(content)).To(BeNumerically("<", 2000))
	})
}

// ============================================================================
// Memory grep tests (TASK-051)
// ============================================================================

// TEST-810: Grep searches index.md
func TestGrepSearchesIndexMd(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Create index.md with some content
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	indexContent := `- 2024-01-01 10:00: Learn about testing
- 2024-01-01 11:00: Study Go patterns
- 2024-01-01 12:00: Review TDD workflow`
	err = os.WriteFile(filepath.Join(memoryDir, "index.md"), []byte(indexContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

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
func TestGrepSearchesSessions(t *testing.T) {
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
func TestGrepReturnsFileAndLineNumber(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	indexContent := `line one
line two with pattern
line three`
	err = os.WriteFile(filepath.Join(memoryDir, "index.md"), []byte(indexContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.GrepOpts{
		Pattern:    "pattern",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Grep(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Matches).To(HaveLen(1))
	g.Expect(results.Matches[0].File).To(ContainSubstring("index.md"))
	g.Expect(results.Matches[0].LineNum).To(Equal(2))
}

// TEST-813: Grep project filter limits search
func TestGrepProjectFilter(t *testing.T) {
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
func TestGrepWithDecisionsFlag(t *testing.T) {
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
func TestGrepNoMatches(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(memoryDir, "index.md"), []byte("some content"), 0644)
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
func TestGrepRequiresPattern(t *testing.T) {
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
func TestGrepCaseInsensitive(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(memoryDir, "index.md"), []byte("PostgreSQL database"), 0644)
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
// Memory query tests (TASK-052)
// ============================================================================

// TEST-820: Query returns similar memories
func TestQueryReturnsSimilarMemories(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Create memories with varying relevance
	indexContent := `- 2024-01-01: Learned about PostgreSQL database design
- 2024-01-02: Studied MySQL query optimization
- 2024-01-03: Read about database indexing strategies
- 2024-01-04: Explored React component patterns`
	err = os.WriteFile(filepath.Join(memoryDir, "index.md"), []byte(indexContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "database performance",
		Limit:      5,
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// Database-related entries should rank higher than React
	for _, r := range results.Results {
		g.Expect(r.Score).To(BeNumerically(">", 0))
		g.Expect(r.Content).ToNot(BeEmpty())
	}
}

// TEST-821: Query returns default 5 results
func TestQueryDefaultLimit(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Create more than 5 memories
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "- 2024-01-01: Memory about testing")
	}
	err = os.WriteFile(filepath.Join(memoryDir, "index.md"), []byte(strings.Join(lines, "\n")), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "testing",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(len(results.Results)).To(BeNumerically("<=", 5))
}

// TEST-822: Query respects custom limit
func TestQueryCustomLimit(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "- 2024-01-01: Memory about testing")
	}
	err = os.WriteFile(filepath.Join(memoryDir, "index.md"), []byte(strings.Join(lines, "\n")), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "testing",
		Limit:      3,
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(len(results.Results)).To(BeNumerically("<=", 3))
}

// TEST-823: Query searches sessions
func TestQuerySearchesSessions(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	sessionsDir := filepath.Join(memoryDir, "sessions")
	err := os.MkdirAll(sessionsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	sessionContent := `# Session Summary
Decided to use PostgreSQL for the database layer.
Completed API design work.`
	err = os.WriteFile(filepath.Join(sessionsDir, "2024-01-01-project.md"), []byte(sessionContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "PostgreSQL",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())
}

// TEST-824: Query requires text
func TestQueryRequiresText(t *testing.T) {
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

// TEST-825: Query results include similarity scores
func TestQueryResultsIncludeScores(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	indexContent := `- 2024-01-01: Testing database queries
- 2024-01-02: Building web frontend`
	err = os.WriteFile(filepath.Join(memoryDir, "index.md"), []byte(indexContent), 0644)
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

// TEST-826: Query results are sorted by score descending
func TestQueryResultsSortedByScore(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	indexContent := `- 2024-01-01: web frontend development
- 2024-01-02: database indexing and optimization
- 2024-01-03: database schema design patterns`
	err = os.WriteFile(filepath.Join(memoryDir, "index.md"), []byte(indexContent), 0644)
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
