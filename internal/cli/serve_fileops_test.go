package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestOsFileOps_MkdirAll_CreatesDirectory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := cli.ExportNewOsFileOps()
	dir := filepath.Join(t.TempDir(), "sub", "nested")

	err := ops.MkdirAll(dir, 0o755)
	g.Expect(err).NotTo(HaveOccurred())

	info, statErr := os.Stat(dir)
	g.Expect(statErr).NotTo(HaveOccurred())

	if statErr != nil {
		return
	}

	g.Expect(info.IsDir()).To(BeTrue())
}

func TestOsFileOps_MkdirAll_ReturnsErrorForInvalidPath(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := cli.ExportNewOsFileOps()

	err := ops.MkdirAll("/dev/null/impossible", 0o755)
	g.Expect(err).To(HaveOccurred())
}

func TestOsFileOps_Rename_MovesFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := cli.ExportNewOsFileOps()
	dir := t.TempDir()
	src := filepath.Join(dir, "old.txt")
	dst := filepath.Join(dir, "new.txt")

	g.Expect(os.WriteFile(src, []byte("data"), 0o644)).To(Succeed())

	err := ops.Rename(src, dst)
	g.Expect(err).NotTo(HaveOccurred())

	_, srcErr := os.Stat(src)
	g.Expect(os.IsNotExist(srcErr)).To(BeTrue())

	_, dstErr := os.Stat(dst)
	g.Expect(dstErr).NotTo(HaveOccurred())
}

func TestOsFileOps_Rename_ReturnsErrorForMissing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := cli.ExportNewOsFileOps()
	dir := t.TempDir()

	err := ops.Rename(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dst"))
	g.Expect(err).To(HaveOccurred())
}

func TestOsFileOps_Stat_ReturnsErrorForMissing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := cli.ExportNewOsFileOps()

	_, err := ops.Stat(filepath.Join(t.TempDir(), "nonexistent"))
	g.Expect(err).To(HaveOccurred())
}

func TestOsFileOps_Stat_ReturnsFileInfo(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	ops := cli.ExportNewOsFileOps()
	path := filepath.Join(t.TempDir(), "file.txt")

	g.Expect(os.WriteFile(path, []byte("content"), 0o644)).To(Succeed())

	info, err := ops.Stat(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(info.Name()).To(Equal("file.txt"))
}
