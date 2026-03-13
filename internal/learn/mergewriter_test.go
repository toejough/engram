package learn_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/learn"
	"engram/internal/memory"
)

func TestJSONMergeWriter_WriteError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	writer := &learn.JSONMergeWriter{}
	existing := &memory.Stored{FilePath: "/nonexistent/dir/mem.json", Title: "Test"}

	err := writer.UpdateMerged(existing, "principle", nil, nil, time.Now())

	g.Expect(err).To(HaveOccurred())
}

func TestJSONMergeWriter_WritesJSON(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "mem.json")

	err := os.WriteFile(filePath, []byte("{}"), 0o600)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	writer := &learn.JSONMergeWriter{}
	existing := &memory.Stored{FilePath: filePath, Title: "Test Memory"}
	now := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	err = writer.UpdateMerged(
		existing,
		"merged principle",
		[]string{"alpha"},
		[]string{"concept1"},
		now,
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filePath)

	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("merged principle"))
}

func TestTOMLMergeWriter_ErrorOnMissingFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	writer := &learn.TOMLMergeWriter{}
	existing := &memory.Stored{FilePath: "/nonexistent/path.toml", Title: "Test"}

	err := writer.UpdateMerged(existing, "principle", nil, nil, time.Now())

	g.Expect(err).To(HaveOccurred())
}

func TestTOMLMergeWriter_WritesFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "mem.toml")

	err := os.WriteFile(filePath, []byte(`principle = "old principle"`), 0o600)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	writer := &learn.TOMLMergeWriter{}
	existing := &memory.Stored{FilePath: filePath, Title: "Test Memory"}
	now := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	err = writer.UpdateMerged(
		existing,
		"new principle",
		[]string{"alpha", "beta"},
		[]string{"concept1"},
		now,
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filePath)

	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("new principle"))
	g.Expect(string(data)).To(ContainSubstring("alpha"))
}
