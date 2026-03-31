package retrieve_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/retrieve"
)

// TestListMemories_EmptyDirectory verifies empty directory handling.
func TestListMemories_EmptyDirectory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	retriever := retrieve.New()
	memories, err := retriever.ListMemories(context.Background(), dataDir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(memories).To(BeEmpty())
	g.Expect(memories).NotTo(BeNil())
}

// TestListMemories_SkipsUnparseableFiles verifies unparseable files are skipped.
func TestListMemories_SkipsUnparseableFiles(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	writeTestTOML(t, memoriesDir, "valid1.toml", tomlContent{
		Situation: "Valid One",
		UpdatedAt: "2025-01-01T00:00:00Z",
	})
	writeTestTOML(t, memoriesDir, "valid2.toml", tomlContent{
		Situation: "Valid Two",
		UpdatedAt: "2025-02-01T00:00:00Z",
	})

	invalidPath := filepath.Join(memoriesDir, "broken.toml")
	writeErr := os.WriteFile(invalidPath, []byte("{{{{invalid toml!!!!"), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	retriever := retrieve.New()
	memories, err := retriever.ListMemories(context.Background(), dataDir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(memories).To(HaveLen(2))
	g.Expect(memories[0].Situation).To(Equal("Valid Two"))
	g.Expect(memories[1].Situation).To(Equal("Valid One"))
}

// TestListMemories_SortedByUpdatedAt verifies sorting order.
func TestListMemories_SortedByUpdatedAt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	writeTestTOML(t, memoriesDir, "oldest.toml", tomlContent{
		Situation: "oldest situation",
		Action:    "oldest action",
		UpdatedAt: "2025-01-01T00:00:00Z",
	})
	writeTestTOML(t, memoriesDir, "newest.toml", tomlContent{
		Situation: "newest situation",
		Action:    "newest action",
		UpdatedAt: "2025-03-01T00:00:00Z",
	})
	writeTestTOML(t, memoriesDir, "middle.toml", tomlContent{
		Situation: "middle situation",
		Behavior:  "middle behavior",
		Action:    "middle action",
		UpdatedAt: "2025-02-01T00:00:00Z",
	})

	retriever := retrieve.New()
	memories, err := retriever.ListMemories(context.Background(), dataDir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(memories).To(HaveLen(3))

	// Sorted by updated_at descending (most recent first).
	g.Expect(memories[0].Situation).To(Equal("newest situation"))
	g.Expect(memories[1].Situation).To(Equal("middle situation"))
	g.Expect(memories[2].Situation).To(Equal("oldest situation"))

	// Verify fields are populated.
	g.Expect(memories[0].Action).To(Equal("newest action"))
	g.Expect(memories[0].FilePath).To(ContainSubstring("newest.toml"))
	g.Expect(memories[0].UpdatedAt).To(Equal(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)))

	// Verify behavior populates.
	g.Expect(memories[1].Behavior).To(Equal("middle behavior"))
}

// TestProjectScopedAndProjectSlugWired verifies SBIA project fields are wired.
func TestProjectScopedAndProjectSlugWired(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	writeTestTOML(t, memoriesDir, "scoped.toml", tomlContent{
		Situation:     "when building",
		Action:        "use targ",
		UpdatedAt:     "2026-03-01T00:00:00Z",
		ProjectScoped: true,
		ProjectSlug:   "my-project",
	})

	retriever := retrieve.New()
	memories, err := retriever.ListMemories(context.Background(), dataDir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(memories).To(HaveLen(1))
	g.Expect(memories[0].ProjectScoped).To(BeTrue())
	g.Expect(memories[0].ProjectSlug).To(Equal("my-project"))
}

// tomlContent is a test helper for writing memory TOML files.
type tomlContent struct {
	Situation     string
	Behavior      string
	Action        string
	UpdatedAt     string
	ProjectScoped bool
	ProjectSlug   string
}

func writeTestTOML(t *testing.T, dir, filename string, tc tomlContent) {
	t.Helper()

	content := fmt.Sprintf(`situation = "%s"
behavior = "%s"
impact = "test impact"
action = "%s"
created_at = "2025-01-01T00:00:00Z"
updated_at = "%s"
`,
		tc.Situation,
		tc.Behavior,
		tc.Action,
		tc.UpdatedAt,
	)

	if tc.ProjectScoped {
		content += "project_scoped = true\n"
	}

	if tc.ProjectSlug != "" {
		content += fmt.Sprintf("project_slug = \"%s\"\n", tc.ProjectSlug)
	}

	path := filepath.Join(dir, filename)

	err := os.WriteFile(path, []byte(content), 0o640)
	if err != nil {
		t.Fatalf("writeTestTOML: %v", err)
	}
}
