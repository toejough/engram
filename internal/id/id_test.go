package id_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/id"
)

func TestNext_EmptyPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	_, err := id.Next(dir, "")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("prefix cannot be empty"))
}

func TestNext_InvalidPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	_, err := id.Next(dir, "INVALID")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid prefix"))
}

func TestNext_NoFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-1"))
}

func TestNext_WithMarkdownIDs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	content := "## REQ-3: Some requirement\n## REQ-7: Another requirement\n"
	err := os.WriteFile(filepath.Join(dir, "reqs.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-8"))
}

func TestRunNext_InvalidType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := id.RunNext(id.NextArgs{Type: "INVALID", Dir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunNext_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := id.RunNext(id.NextArgs{Type: "REQ", Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}
