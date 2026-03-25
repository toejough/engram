package cli //nolint:testpackage // white-box tests for unexported readRecord function

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
)

func TestReadRecord_LoadsMemoryRecord(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	content := `title = "Test Memory"
principle = "Test principle"
keywords = ["kw1", "kw2"]
surfaced_count = 5
source_path = "stale/path.toml"
`
	err := os.WriteFile(path, []byte(content), 0o600)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	record, readErr := readRecord(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(record).NotTo(gomega.BeNil())

	if record == nil {
		return
	}

	g.Expect(record.Title).To(gomega.Equal("Test Memory"))
	g.Expect(record.Principle).To(gomega.Equal("Test principle"))
	g.Expect(record.Keywords).To(gomega.Equal([]string{"kw1", "kw2"}))
	g.Expect(record.SurfacedCount).To(gomega.Equal(5))
	// SourcePath must be overwritten with actual path, not stale on-disk value
	g.Expect(record.SourcePath).To(gomega.Equal(path))
}

func TestReadRecord_NonexistentFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	_, err := readRecord("/nonexistent/test.toml")
	g.Expect(err).To(gomega.HaveOccurred())
}
