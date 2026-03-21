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

func TestJSONMergeWriter_UpdateMerged(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "memory.json")

	err := os.WriteFile(filePath, []byte(`{}`), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	existing := &memory.Stored{FilePath: filePath, Title: "existing"}
	writer := &learn.JSONMergeWriter{}

	err = writer.UpdateMerged(existing, "merged principle", []string{"alpha"}, []string{"c1"}, time.Now())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filePath)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("merged principle"))
}

func TestJSONMergeWriter_UpdateMerged_FileNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	existing := &memory.Stored{FilePath: "/nonexistent/path.json"}
	writer := &learn.JSONMergeWriter{}

	err := writer.UpdateMerged(existing, "principle", nil, nil, time.Now())
	g.Expect(err).To(HaveOccurred())
}

func TestTOMLMergeWriter_UpdateMerged(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "memory.toml")

	err := os.WriteFile(filePath, []byte(`principle = "old"`), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	existing := &memory.Stored{FilePath: filePath, Title: "existing"}
	writer := &learn.TOMLMergeWriter{}

	err = writer.UpdateMerged(existing, "new principle", []string{"alpha", "beta"}, []string{"c1"}, time.Now())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filePath)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("new principle"))
}

func TestTOMLMergeWriter_UpdateMerged_FileNotFound(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	existing := &memory.Stored{FilePath: "/nonexistent/path.toml"}
	writer := &learn.TOMLMergeWriter{}

	err := writer.UpdateMerged(existing, "principle", nil, nil, time.Now())
	g.Expect(err).To(HaveOccurred())
}

func TestTOMLMergeWriter_NormalizesKeywords(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "memory.toml")

	err := os.WriteFile(filePath, []byte(`principle = "test"`), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	existing := &memory.Stored{FilePath: filePath}
	writer := &learn.TOMLMergeWriter{}

	err = writer.UpdateMerged(existing, "principle", []string{"Mixed-Case", "hyphen-sep"}, nil, time.Now())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filePath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(data)
	g.Expect(content).To(ContainSubstring(`"mixed_case"`))
	g.Expect(content).To(ContainSubstring(`"hyphen_sep"`))
	g.Expect(content).NotTo(ContainSubstring(`"Mixed-Case"`))
	g.Expect(content).NotTo(ContainSubstring(`"hyphen-sep"`))
}
