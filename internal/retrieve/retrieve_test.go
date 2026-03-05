package retrieve_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/retrieve"
)

// T-24: ListMemories returns all TOML files sorted by updated_at
func TestT24_ListMemoriesSortedByUpdatedAt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Three memories with different updated_at timestamps.
	// Write them out of order to verify sorting.
	writeTestTOML(t, memoriesDir, "oldest.toml", tomlContent{
		Title:     "Oldest Memory",
		Keywords:  []string{"old"},
		Concepts:  []string{"history"},
		UpdatedAt: "2025-01-01T00:00:00Z",
		Principle: "oldest principle",
	})
	writeTestTOML(t, memoriesDir, "newest.toml", tomlContent{
		Title:     "Newest Memory",
		Keywords:  []string{"new"},
		Concepts:  []string{"recent"},
		UpdatedAt: "2025-03-01T00:00:00Z",
		Principle: "newest principle",
	})
	writeTestTOML(t, memoriesDir, "middle.toml", tomlContent{
		Title:       "Middle Memory",
		Keywords:    []string{"mid"},
		Concepts:    []string{"midrange"},
		AntiPattern: "manual git commit",
		UpdatedAt:   "2025-02-01T00:00:00Z",
		Principle:   "middle principle",
	})

	r := retrieve.New()
	memories, err := r.ListMemories(context.Background(), dataDir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(memories).To(HaveLen(3))

	// Sorted by updated_at descending (most recent first).
	g.Expect(memories[0].Title).To(Equal("Newest Memory"))
	g.Expect(memories[1].Title).To(Equal("Middle Memory"))
	g.Expect(memories[2].Title).To(Equal("Oldest Memory"))

	// Verify fields are populated.
	g.Expect(memories[0].Keywords).To(Equal([]string{"new"}))
	g.Expect(memories[0].Concepts).To(Equal([]string{"recent"}))
	g.Expect(memories[0].Principle).To(Equal("newest principle"))
	g.Expect(memories[0].FilePath).To(ContainSubstring("newest.toml"))
	g.Expect(memories[0].UpdatedAt).To(Equal(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)))

	// Verify anti_pattern populates.
	g.Expect(memories[1].AntiPattern).To(Equal("manual git commit"))
}

// T-25: ListMemories returns empty slice when no memories exist
func TestT25_ListMemoriesEmptyDirectory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	r := retrieve.New()
	memories, err := r.ListMemories(context.Background(), dataDir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(memories).To(BeEmpty())
	g.Expect(memories).NotTo(BeNil()) // empty slice, not nil
}

// T-26: ListMemories skips unparseable files
func TestT26_ListMemoriesSkipsUnparseableFiles(t *testing.T) {
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
		Title:     "Valid One",
		Keywords:  []string{"valid"},
		UpdatedAt: "2025-01-01T00:00:00Z",
	})
	writeTestTOML(t, memoriesDir, "valid2.toml", tomlContent{
		Title:     "Valid Two",
		Keywords:  []string{"also-valid"},
		UpdatedAt: "2025-02-01T00:00:00Z",
	})

	// Write an invalid TOML file.
	invalidPath := filepath.Join(memoriesDir, "broken.toml")
	writeErr := os.WriteFile(invalidPath, []byte("{{{{invalid toml!!!!"), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	r := retrieve.New()
	memories, err := r.ListMemories(context.Background(), dataDir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Only 2 valid memories returned.
	g.Expect(memories).To(HaveLen(2))
	g.Expect(memories[0].Title).To(Equal("Valid Two")) // more recent
	g.Expect(memories[1].Title).To(Equal("Valid One"))
}

// tomlContent is a test helper for writing memory TOML files.
type tomlContent struct {
	Title       string
	Keywords    []string
	Concepts    []string
	AntiPattern string
	UpdatedAt   string
	Principle   string
}

func writeTestTOML(t *testing.T, dir, filename string, tc tomlContent) {
	t.Helper()

	keywords := "[]"
	if len(tc.Keywords) > 0 {
		keywords = `["` + tc.Keywords[0] + `"]`
		if len(tc.Keywords) > 1 {
			keywords = "["

			var keywordsSb168 strings.Builder

			for i, k := range tc.Keywords {
				if i > 0 {
					keywordsSb168.WriteString(", ")
				}

				keywordsSb168.WriteString(`"` + k + `"`)
			}

			keywords += keywordsSb168.String()

			keywords += "]"
		}
	}

	concepts := "[]"
	if len(tc.Concepts) > 0 {
		concepts = "["

		var conceptsSb181 strings.Builder

		for i, c := range tc.Concepts {
			if i > 0 {
				conceptsSb181.WriteString(", ")
			}

			conceptsSb181.WriteString(`"` + c + `"`)
		}

		concepts += conceptsSb181.String()

		concepts += "]"
	}

	content := `title = "` + tc.Title + `"
content = "test content"
observation_type = "correction"
concepts = ` + concepts + `
keywords = ` + keywords + `
principle = "` + tc.Principle + `"
anti_pattern = "` + tc.AntiPattern + `"
rationale = "test rationale"
confidence = "B"
created_at = "2025-01-01T00:00:00Z"
updated_at = "` + tc.UpdatedAt + `"
`

	path := filepath.Join(dir, filename)

	err := os.WriteFile(path, []byte(content), 0o640)
	if err != nil {
		t.Fatalf("writeTestTOML: %v", err)
	}
}
