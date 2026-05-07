package cli_test

import (
	"os"
	"path/filepath"
	"testing"

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
