package maintain_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/maintain"
)

// T-395: PurgeTierC removes only tier C memory files, leaving A and B intact.
func TestPurgeTierC(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	dir := t.TempDir()

	tierAContent := `title = "Tier A memory"
confidence = "A"
content = "some content"
`
	tierBContent := `title = "Tier B memory"
confidence = "B"
content = "some content"
`
	tierCContent := `title = "Tier C memory"
confidence = "C"
content = "some content"
`

	writeErr := os.WriteFile(filepath.Join(dir, "mem-a.toml"), []byte(tierAContent), 0o600)
	g.Expect(writeErr).NotTo(gomega.HaveOccurred())

	if writeErr != nil {
		return
	}

	writeErr = os.WriteFile(filepath.Join(dir, "mem-b.toml"), []byte(tierBContent), 0o600)
	g.Expect(writeErr).NotTo(gomega.HaveOccurred())

	if writeErr != nil {
		return
	}

	writeErr = os.WriteFile(filepath.Join(dir, "mem-c.toml"), []byte(tierCContent), 0o600)
	g.Expect(writeErr).NotTo(gomega.HaveOccurred())

	if writeErr != nil {
		return
	}

	deleted, err := maintain.PurgeTierC(dir, os.Remove)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(deleted).To(gomega.Equal(1))

	_, statErrA := os.Stat(filepath.Join(dir, "mem-a.toml"))
	g.Expect(statErrA).NotTo(gomega.HaveOccurred())

	_, statErrB := os.Stat(filepath.Join(dir, "mem-b.toml"))
	g.Expect(statErrB).NotTo(gomega.HaveOccurred())

	_, statErrC := os.Stat(filepath.Join(dir, "mem-c.toml"))
	g.Expect(os.IsNotExist(statErrC)).To(gomega.BeTrue())
}
