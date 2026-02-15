//go:build integration

package memory_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestCorrectionRecurrence_DetectsRecurrentCorrection verifies that when a similar correction
// already exists in the database, the new correction is flagged as recurrent.
func TestCorrectionRecurrence_DetectsRecurrentCorrection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store a prior correction
	priorCorrection := "Never use git amend on pushed commits"
	err = memory.Learn(memory.LearnOpts{
		Message:    priorCorrection,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Extract a session with a similar correction
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")
	transcript := []map[string]any{
		{
			"type": "assistant",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "I'll amend the pushed commit",
					},
				},
			},
		},
		{
			"type": "user",
			"message": map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "No, never use git amend on pushed commits",
					},
				},
			},
		},
	}

	// Write transcript
	f, err := os.Create(transcriptPath)
	g.Expect(err).ToNot(HaveOccurred())
	for _, msg := range transcript {
		data, _ := json.Marshal(msg)
		_, _ = f.Write(data)
		_, _ = f.Write([]byte("\n"))
	}
	f.Close()

	// Extract session
	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Status).To(Equal("success"))

	// Verify correction was extracted
	g.Expect(result.Items).ToNot(BeEmpty())
	var foundCorrection bool
	for _, item := range result.Items {
		if item.Type == "correction" {
			foundCorrection = true
			break
		}
	}
	g.Expect(foundCorrection).To(BeTrue(), "should extract correction")

	// Verify changelog entry for recurrence
	changelogPath := filepath.Join(memoryDir, "changelog.jsonl")
	data, err := os.ReadFile(changelogPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Parse changelog entries
	lines := splitJSONLines(data)
	var foundRecurrence bool
	for _, line := range lines {
		var entry memory.ChangelogEntry
		if err := json.Unmarshal(line, &entry); err == nil {
			if entry.Action == "correction_recurrence" {
				foundRecurrence = true
				g.Expect(entry.Reason).To(ContainSubstring("same correction"))
				break
			}
		}
	}
	g.Expect(foundRecurrence).To(BeTrue(), "should log recurrence in changelog")
}

// TestCorrectionRecurrence_NoRecurrenceForNovelCorrection verifies that novel corrections
// are not flagged as recurrent.
func TestCorrectionRecurrence_NoRecurrenceForNovelCorrection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store a correction about a different topic
	priorCorrection := "Use AI-Used trailer, not Co-Authored-By"
	err = memory.Learn(memory.LearnOpts{
		Message:    priorCorrection,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Extract a session with a different correction
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")
	transcript := []map[string]any{
		{
			"type": "assistant",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "I'll use mage for tests",
					},
				},
			},
		},
		{
			"type": "user",
			"message": map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "No, use targ for tests, not mage",
					},
				},
			},
		},
	}

	// Write transcript
	f, err := os.Create(transcriptPath)
	g.Expect(err).ToNot(HaveOccurred())
	for _, msg := range transcript {
		data, _ := json.Marshal(msg)
		_, _ = f.Write(data)
		_, _ = f.Write([]byte("\n"))
	}
	f.Close()

	// Extract session
	result, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Status).To(Equal("success"))

	// Verify correction was extracted
	g.Expect(result.Items).ToNot(BeEmpty())

	// Verify NO changelog entry for recurrence
	changelogPath := filepath.Join(memoryDir, "changelog.jsonl")
	data, err := os.ReadFile(changelogPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("unexpected error reading changelog: %v", err)
	}

	if len(data) > 0 {
		lines := splitJSONLines(data)
		for _, line := range lines {
			var entry memory.ChangelogEntry
			if err := json.Unmarshal(line, &entry); err == nil {
				g.Expect(entry.Action).ToNot(Equal("correction_recurrence"), "should not flag novel correction as recurrent")
			}
		}
	}
}

// TestCorrectionRecurrence_IncrementsRecurrenceCount verifies that recurrence_count
// metadata is incremented when the same correction recurs.
func TestCorrectionRecurrence_IncrementsRecurrenceCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store a prior correction
	priorCorrection := "Use go test -tags sqlite_fts5, not plain go test"
	err = memory.Learn(memory.LearnOpts{
		Message:    priorCorrection,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Extract first recurrence
	transcriptPath1 := filepath.Join(tempDir, "transcript1.jsonl")
	transcript1 := []map[string]any{
		{
			"type": "assistant",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "I'll run go test",
					},
				},
			},
		},
		{
			"type": "user",
			"message": map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "No, use go test -tags sqlite_fts5, not plain go test",
					},
				},
			},
		},
	}
	f1, err := os.Create(transcriptPath1)
	g.Expect(err).ToNot(HaveOccurred())
	for _, msg := range transcript1 {
		data, _ := json.Marshal(msg)
		_, _ = f1.Write(data)
		_, _ = f1.Write([]byte("\n"))
	}
	f1.Close()

	result1, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath1,
		MemoryRoot:     memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result1.Status).To(Equal("success"))

	// Extract second recurrence
	transcriptPath2 := filepath.Join(tempDir, "transcript2.jsonl")
	transcript2 := []map[string]any{
		{
			"type": "assistant",
			"message": map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "I'll run plain go test",
					},
				},
			},
		},
		{
			"type": "user",
			"message": map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "No, use go test -tags sqlite_fts5 always",
					},
				},
			},
		},
	}
	f2, err := os.Create(transcriptPath2)
	g.Expect(err).ToNot(HaveOccurred())
	for _, msg := range transcript2 {
		data, _ := json.Marshal(msg)
		_, _ = f2.Write(data)
		_, _ = f2.Write([]byte("\n"))
	}
	f2.Close()

	result2, err := memory.ExtractSession(memory.ExtractSessionOpts{
		TranscriptPath: transcriptPath2,
		MemoryRoot:     memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result2.Status).To(Equal("success"))

	// Verify changelog has multiple recurrence entries with incrementing counts
	changelogPath := filepath.Join(memoryDir, "changelog.jsonl")
	data, err := os.ReadFile(changelogPath)
	g.Expect(err).ToNot(HaveOccurred())

	lines := splitJSONLines(data)
	recurrenceCount := 0
	for _, line := range lines {
		var entry memory.ChangelogEntry
		if err := json.Unmarshal(line, &entry); err == nil {
			if entry.Action == "correction_recurrence" {
				recurrenceCount++
			}
		}
	}
	g.Expect(recurrenceCount).To(Equal(2), "should log 2 recurrence events")
}

// splitJSONLines splits JSONL data into individual JSON objects.
func splitJSONLines(data []byte) [][]byte {
	var lines [][]byte
	for _, line := range splitLines(data) {
		trimmed := trimBytes(line)
		if len(trimmed) > 0 {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

func trimBytes(b []byte) []byte {
	start := 0
	for start < len(b) && (b[start] == ' ' || b[start] == '\t' || b[start] == '\r') {
		start++
	}
	end := len(b)
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t' || b[end-1] == '\r') {
		end--
	}
	return b[start:end]
}
