package memory_test

import (
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
