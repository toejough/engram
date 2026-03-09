package maintain_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/maintain"
)

// T-265: Memory TOML rewriter — atomic write preserves fields.
func TestTOMLRewriter_PreservesUnchangedFields(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	memPath := filepath.Join(dir, "test-memory.toml")

	original := `title = "Test Memory"
content = "Original content"
observation_type = "pattern"
concepts = ["testing", "quality"]
keywords = ["targ", "build"]
principle = "Original principle"
anti_pattern = "Original anti-pattern"
rationale = "Original rationale"
confidence = "A"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-02-01T00:00:00Z"
`

	err := os.WriteFile(memPath, []byte(original), 0o644)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	rewriter := maintain.NewTOMLRewriter()

	updates := map[string]any{
		"content":   "Updated content about testing practices",
		"principle": "Always write tests before implementation",
	}

	rewriteErr := rewriter.Rewrite(memPath, updates)
	g.Expect(rewriteErr).NotTo(gomega.HaveOccurred())

	if rewriteErr != nil {
		return
	}

	// Read back and verify.
	data, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(data)

	// Updated fields.
	g.Expect(content).To(gomega.ContainSubstring("Updated content about testing practices"))
	g.Expect(content).To(gomega.ContainSubstring("Always write tests before implementation"))

	// Preserved fields.
	g.Expect(content).To(gomega.ContainSubstring(`title = "Test Memory"`))
	g.Expect(content).To(gomega.ContainSubstring(`observation_type = "pattern"`))
	g.Expect(content).To(gomega.ContainSubstring(`anti_pattern = "Original anti-pattern"`))
	g.Expect(content).To(gomega.ContainSubstring(`confidence = "A"`))
}

// T-265b: Rewriter updates keywords as string slice.
func TestTOMLRewriter_UpdatesKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	memPath := filepath.Join(dir, "keyword-memory.toml")

	original := `title = "Keyword Test"
content = "Some content"
keywords = ["targ", "build"]
`

	err := os.WriteFile(memPath, []byte(original), 0o644)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	rewriter := maintain.NewTOMLRewriter()

	updates := map[string]any{
		"keywords": []string{"targ", "build", "test", "check"},
	}

	rewriteErr := rewriter.Rewrite(memPath, updates)
	g.Expect(rewriteErr).NotTo(gomega.HaveOccurred())

	if rewriteErr != nil {
		return
	}

	data, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(data)
	g.Expect(content).To(gomega.ContainSubstring("test"))
	g.Expect(content).To(gomega.ContainSubstring("check"))
	g.Expect(content).To(gomega.ContainSubstring("targ"))
	g.Expect(content).To(gomega.ContainSubstring("build"))
}
