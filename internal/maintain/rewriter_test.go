package maintain_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/maintain"
)

// TestTOMLRewriter_DecodeError verifies error when TOML is invalid.
func TestTOMLRewriter_DecodeError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	rewriter := maintain.NewTOMLRewriter(
		maintain.WithReadFile(func(_ string) ([]byte, error) {
			return []byte("not = [valid toml"), nil
		}),
	)

	err := rewriter.Rewrite("/fake/bad.toml", map[string]any{"title": "x"})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("decoding memory TOML")))
}

// Rewrite with empty updates still writes (no-op merge).
func TestTOMLRewriter_EmptyUpdates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writeCalled := false

	rewriter := maintain.NewTOMLRewriter(
		maintain.WithReadFile(func(_ string) ([]byte, error) {
			return []byte("title = \"test\"\n"), nil
		}),
		maintain.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			writeCalled = true

			return nil
		}),
		maintain.WithRenameFile(func(_, _ string) error { return nil }),
	)

	err := rewriter.Rewrite("/f.toml", map[string]any{})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writeCalled).To(gomega.BeTrue())
}

// Rewrite merges updates into existing fields and adds new ones.
func TestTOMLRewriter_MergesUpdates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var writtenData []byte

	rewriter := maintain.NewTOMLRewriter(
		maintain.WithReadFile(func(_ string) ([]byte, error) {
			return []byte("title = \"original\"\nkeep = \"preserved\"\n"), nil
		}),
		maintain.WithWriteFile(func(_ string, data []byte, _ os.FileMode) error {
			writtenData = data

			return nil
		}),
		maintain.WithRenameFile(func(_, _ string) error { return nil }),
	)

	err := rewriter.Rewrite("/f.toml", map[string]any{
		"title": "updated",
		"added": "new field",
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	content := string(writtenData)
	g.Expect(content).To(gomega.ContainSubstring(`title = "updated"`))
	g.Expect(content).To(gomega.ContainSubstring(`keep = "preserved"`))
	g.Expect(content).To(gomega.ContainSubstring(`added = "new field"`))
}

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

// TestTOMLRewriter_ReadError verifies error when read fails.
func TestTOMLRewriter_ReadError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	rewriter := maintain.NewTOMLRewriter(
		maintain.WithReadFile(func(_ string) ([]byte, error) {
			return nil, errors.New("file not found")
		}),
	)

	err := rewriter.Rewrite("/fake/missing.toml", map[string]any{"title": "x"})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("reading memory TOML")))
}

// TestTOMLRewriter_RenameError verifies error when rename fails.
func TestTOMLRewriter_RenameError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	rewriter := maintain.NewTOMLRewriter(
		maintain.WithReadFile(func(_ string) ([]byte, error) {
			return []byte("title = \"test\"\n"), nil
		}),
		maintain.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			return nil
		}),
		maintain.WithRenameFile(func(_, _ string) error {
			return errors.New("cross-device link")
		}),
	)

	err := rewriter.Rewrite("/fake/path.toml", map[string]any{"title": "x"})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("renaming temp to final")))
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

// TestTOMLRewriter_WithOptions verifies functional options are applied.
func TestTOMLRewriter_WithOptions(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	readCalled := false
	writeCalled := false
	renameCalled := false

	rewriter := maintain.NewTOMLRewriter(
		maintain.WithReadFile(func(_ string) ([]byte, error) {
			readCalled = true

			return []byte("title = \"test\"\n"), nil
		}),
		maintain.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			writeCalled = true

			return nil
		}),
		maintain.WithRenameFile(func(_, _ string) error {
			renameCalled = true

			return nil
		}),
	)

	err := rewriter.Rewrite("/fake/path.toml", map[string]any{"title": "updated"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(readCalled).To(gomega.BeTrue())
	g.Expect(writeCalled).To(gomega.BeTrue())
	g.Expect(renameCalled).To(gomega.BeTrue())
}

// WithReadFile alone overrides the default os.ReadFile.
func TestTOMLRewriter_WithReadFileAlone(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	customCalled := false

	rewriter := maintain.NewTOMLRewriter(
		maintain.WithReadFile(func(_ string) ([]byte, error) {
			customCalled = true

			return nil, errors.New("custom read")
		}),
	)

	err := rewriter.Rewrite("/fake.toml", map[string]any{"k": "v"})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(customCalled).To(gomega.BeTrue())
}

// WithRenameFile alone overrides the default os.Rename.
func TestTOMLRewriter_WithRenameFileAlone(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var renamedFrom, renamedTo string

	rewriter := maintain.NewTOMLRewriter(
		maintain.WithReadFile(func(_ string) ([]byte, error) {
			return []byte("title = \"test\"\n"), nil
		}),
		maintain.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			return nil
		}),
		maintain.WithRenameFile(func(oldpath, newpath string) error {
			renamedFrom = oldpath
			renamedTo = newpath

			return nil
		}),
	)

	err := rewriter.Rewrite("/dir/file.toml", map[string]any{"title": "new"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(renamedFrom).To(gomega.Equal("/dir/.tmp-rewrite"))
	g.Expect(renamedTo).To(gomega.Equal("/dir/file.toml"))
}

// WithWriteFile alone overrides the default os.WriteFile.
func TestTOMLRewriter_WithWriteFileAlone(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writtenPath := ""

	rewriter := maintain.NewTOMLRewriter(
		maintain.WithReadFile(func(_ string) ([]byte, error) {
			return []byte("title = \"test\"\n"), nil
		}),
		maintain.WithWriteFile(func(name string, _ []byte, _ os.FileMode) error {
			writtenPath = name

			return nil
		}),
		maintain.WithRenameFile(func(_, _ string) error { return nil }),
	)

	err := rewriter.Rewrite("/dir/file.toml", map[string]any{"title": "new"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writtenPath).To(gomega.Equal("/dir/.tmp-rewrite"))
}

// TestTOMLRewriter_WriteError verifies error when temp file write fails.
func TestTOMLRewriter_WriteError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	rewriter := maintain.NewTOMLRewriter(
		maintain.WithReadFile(func(_ string) ([]byte, error) {
			return []byte("title = \"test\"\n"), nil
		}),
		maintain.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			return errors.New("disk full")
		}),
	)

	err := rewriter.Rewrite("/fake/path.toml", map[string]any{"title": "x"})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("writing temp file")))
}
