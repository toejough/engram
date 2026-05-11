package cli_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestOsDirLister_ListJSONL(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "session1.jsonl"), []byte("{}"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "session2.jsonl"), []byte("{}"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not jsonl"), 0o644)).
		To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(dir, "subdir"), 0o755)).To(Succeed())

	lister := cli.ExportNewOsDirLister()
	entries, err := lister.ListJSONL(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(2))
}

func TestOsDirLister_ListJSONL_NotADirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	filePath := filepath.Join(t.TempDir(), "notadir.jsonl")
	g.Expect(os.WriteFile(filePath, []byte("{}"), 0o644)).To(Succeed())

	lister := cli.ExportNewOsDirLister()
	_, err := lister.ListJSONL(filePath)
	g.Expect(err).To(HaveOccurred())
}

func TestOsDirLister_ListJSONL_NotExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := cli.ExportNewOsDirLister()
	entries, err := lister.ListJSONL("/nonexistent/path")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

func TestOsFileReader_Read(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "test.txt")
	g.Expect(os.WriteFile(path, []byte("hello world"), 0o644)).To(Succeed())

	reader := cli.ExportNewOsFileReader()
	data, err := reader.Read(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(Equal("hello world"))
}

func TestOsFileReader_ReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reader := cli.ExportNewOsFileReader()
	_, err := reader.Read("/nonexistent/file.txt")
	g.Expect(err).To(HaveOccurred())
}

func TestOsPromoteFS_ListIDs_ReturnsBothPermanentAndMOC(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o700)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "Permanent", "1.2026-05-09.foo.md"), nil, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "Permanent", "1a.2026-05-09.bar.md"), nil, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "MOCs", "5.2026-05-09.moc.md"), nil, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "Permanent", "README.md"), nil, 0o600)).To(Succeed())

	fs := cli.ExportNewOsPromoteFS()
	got, err := fs.ListIDs(vault)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(ConsistOf("1", "1a", "5"))
}

func TestOsPromoteFS_Lock_ExclusiveAcrossSecondAcquisition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	vault := t.TempDir()

	fs := cli.ExportNewOsPromoteFS()
	release1, err := fs.Lock(vault)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	done := make(chan struct{})

	go func() {
		release2, err2 := fs.Lock(vault)
		g.Expect(err2).NotTo(HaveOccurred())

		if release2 != nil {
			release2()
		}

		close(done)
	}()

	select {
	case <-done:
		t.Fatal("second Lock should not have succeeded while first holds")
	case <-time.After(100 * time.Millisecond):
	}

	release1()
	<-done
}
