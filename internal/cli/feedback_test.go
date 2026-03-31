package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestRunFeedback_IncrementFollowed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memPath := writeTestMemory(t, dataDir, "test-mem", feedbackTestMemoryTOML)

	err := cli.RunFeedback([]string{"--name", "test-mem", "--data-dir", dataDir, "--relevant", "--used"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	content := readMemoryContent(t, memPath)
	g.Expect(content).To(ContainSubstring("followed_count = 3"))
}

func TestRunFeedback_IncrementIrrelevant(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	customTOML := strings.ReplaceAll(feedbackTestMemoryTOML, "irrelevant_count = 0", "irrelevant_count = 1")
	memPath := writeTestMemory(t, dataDir, "test-mem", customTOML)

	err := cli.RunFeedback([]string{"--name", "test-mem", "--data-dir", dataDir, "--irrelevant"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	content := readMemoryContent(t, memPath)
	g.Expect(content).To(ContainSubstring("irrelevant_count = 2"))
}

func TestRunFeedback_IncrementNotFollowed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memPath := writeTestMemory(t, dataDir, "test-mem", feedbackTestMemoryTOML)

	err := cli.RunFeedback([]string{"--name", "test-mem", "--data-dir", dataDir, "--relevant", "--notused"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	content := readMemoryContent(t, memPath)
	g.Expect(content).To(ContainSubstring("not_followed_count = 2"))
}

func TestRunFeedback_MissingName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.RunFeedback([]string{"--relevant"})
	g.Expect(err).To(HaveOccurred())
}

func TestApplyFeedbackCounters_UsedAlone(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := &memory.MemoryRecord{FollowedCount: 0}
	cli.ExportApplyFeedbackCounters(record, false, false, true, false)
	g.Expect(record.FollowedCount).To(Equal(1))
}

func TestApplyFeedbackCounters_NotusedAlone(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := &memory.MemoryRecord{NotFollowedCount: 0}
	cli.ExportApplyFeedbackCounters(record, false, false, false, true)
	g.Expect(record.NotFollowedCount).To(Equal(1))
}

// unexported constants.
const (
	feedbackTestMemoryTOML = `situation = "test situation"
behavior = "test behavior"
impact = "test impact"
action = "test action"
surfaced_count = 5
followed_count = 2
not_followed_count = 1
irrelevant_count = 0
created_at = "2026-03-29T12:00:00Z"
updated_at = "2026-03-29T12:00:00Z"
`
)

func readMemoryContent(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading memory file: %v", err)
	}

	return string(data)
}

func writeTestMemory(t *testing.T, dataDir, name, content string) string {
	t.Helper()

	memoriesDir := filepath.Join(dataDir, "memories")

	if err := os.MkdirAll(memoriesDir, 0o755); err != nil {
		t.Fatalf("creating memories dir: %v", err)
	}

	path := filepath.Join(memoriesDir, name+".toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing memory file: %v", err)
	}

	return path
}
