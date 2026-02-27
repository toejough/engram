package log_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/log"
)

func TestRealFS_AppendFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	fs := log.RealFS{}

	err := fs.AppendFile(path, []byte("hello"))
	g.Expect(err).ToNot(HaveOccurred())

	err = fs.AppendFile(path, []byte(" world"))
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal("hello world"))
}

func TestRealFS_FileExists_Exists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	err := os.WriteFile(path, []byte("content"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	fs := log.RealFS{}
	g.Expect(fs.FileExists(path)).To(BeTrue())
}

func TestRealFS_FileExists_NotExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	fs := log.RealFS{}
	g.Expect(fs.FileExists(filepath.Join(dir, "nonexistent.txt"))).To(BeFalse())
}

func TestRealFS_ReadFile_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	fs := log.RealFS{}
	_, err := fs.ReadFile(filepath.Join(dir, "nonexistent.txt"))
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_ReadFile_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	err := os.WriteFile(path, []byte("file content"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	fs := log.RealFS{}
	content, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal("file content"))
}

func TestRunRead_NoLogFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := log.RunRead(log.ReadArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunRead_ReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	logPath := filepath.Join(dir, log.LogFile)
	err := os.MkdirAll(logPath, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	err = log.RunRead(log.ReadArgs{Dir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunRead_WithEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := log.RunWrite(log.WriteArgs{
		Dir:     dir,
		Level:   "status",
		Subject: "task-status",
		Message: "test message",
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = log.RunRead(log.ReadArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunWrite_InvalidLevel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := log.RunWrite(log.WriteArgs{
		Dir:     dir,
		Level:   "invalid",
		Subject: "task-status",
		Message: "test message",
	})
	g.Expect(err).To(HaveOccurred())
}

func TestRunWrite_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := log.RunWrite(log.WriteArgs{
		Dir:     dir,
		Level:   "status",
		Subject: "task-status",
		Message: "test message",
	})
	g.Expect(err).ToNot(HaveOccurred())

	logPath := filepath.Join(dir, log.LogFile)
	g.Expect(logPath).To(BeAnExistingFile())
}
