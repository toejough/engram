package embed_test

import (
	stdembed "embed"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// TestProductionTempFS_DirectMethods exercises each adapter method
// directly so the success + error wraps get measured.
func TestProductionTempFS_DirectMethods(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tfs := embed.ExportProductionTempFS()

	// Success: real mkdir.
	tmp, mkErr := tfs.MkdirTemp("", "engram-prodtempfs-*")
	g.Expect(mkErr).NotTo(HaveOccurred())

	defer func() { _ = os.RemoveAll(tmp) }()

	// Success: real write.
	target := filepath.Join(tmp, "x.txt")
	g.Expect(tfs.WriteFile(target, []byte("payload"))).To(Succeed())

	// Error: mkdir under non-existent root surfaces wrapped error.
	_, mkErr2 := tfs.MkdirTemp("/no/such/path", "engram-fail-*")
	g.Expect(mkErr2).To(HaveOccurred())

	// Error: write to non-existent dir surfaces wrapped error.
	writeErr := tfs.WriteFile("/no/such/path/x.txt", []byte("x"))
	g.Expect(writeErr).To(HaveOccurred())

	// Success: removeAll on tmp.
	g.Expect(tfs.RemoveAll(tmp)).To(Succeed())
}

// TestProductionTempFS_RoundTrip walks every method of the production
// adapter (via a parallel impl) to assert the basic Mkdir/Write/Remove
// contract. The actual production type is exercised end-to-end through
// the bundled-embedder smoke test.
func TestProductionTempFS_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tfs := productionTempFSForTest{}

	tmp, mkErr := tfs.MkdirTemp("", "engram-tempfs-test-*")
	g.Expect(mkErr).NotTo(HaveOccurred())

	defer func() { _ = os.RemoveAll(tmp) }()

	target := filepath.Join(tmp, "f.txt")

	g.Expect(tfs.WriteFile(target, []byte("hi"))).To(Succeed())

	bytes, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(bytes)).To(Equal("hi"))

	g.Expect(tfs.RemoveAll(tmp)).To(Succeed())

	_, statErr := os.Stat(tmp)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

// TestUnpackModelToTemp_RealOS exercises the production unpack path
// against a tempdir so MkdirTemp/WriteFile/RemoveAll (on the real
// adapter) get covered by integration.
func TestUnpackModelToTemp_RealOS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp, err := embed.ExportUnpackModelToTempProduction(nonEmptyTestFS, "testdata")
	g.Expect(err).NotTo(HaveOccurred())

	defer func() { _ = os.RemoveAll(tmp) }()

	_, statErr := os.Stat(tmp)
	g.Expect(statErr).NotTo(HaveOccurred())
}

// unexported variables.
var (
	_ stdembed.FS
)

// productionTempFSForTest mirrors the unexported productionTempFS type;
// exercising the wrapping behavior via the exported tempFS-injectable
// unpack helper covers MkdirTemp / WriteFile / RemoveAll under coverage.
type productionTempFSForTest struct{}

func (productionTempFSForTest) MkdirTemp(dir, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern)
}

func (productionTempFSForTest) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (productionTempFSForTest) WriteFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0o600)
}
