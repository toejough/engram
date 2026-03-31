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

	content := `situation = "when running tests"
behavior = "use go test directly"
impact = "misses coverage"
action = "use targ test"
surfaced_count = 5
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

	g.Expect(record.Situation).To(gomega.Equal("when running tests"))
	g.Expect(record.Action).To(gomega.Equal("use targ test"))
	g.Expect(record.SurfacedCount).To(gomega.Equal(5))
}

func TestReadRecord_NonexistentFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	_, err := readRecord("/nonexistent/test.toml")
	g.Expect(err).To(gomega.HaveOccurred())
}
