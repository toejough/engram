package embed_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// TestIsExistErr covers the isExistErr helper for both positive and
// negative inputs including the LinkError / string-match branches.
func TestIsExistErr(t *testing.T) {
	t.Parallel()

	t.Run("nil is not an exist error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(embed.ExportIsExistErr(nil)).To(BeFalse())
	})

	t.Run("os.ErrExist is recognized", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(embed.ExportIsExistErr(os.ErrExist)).To(BeTrue())
	})

	t.Run("LinkError with ErrExist is recognized", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linkErr := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: os.ErrExist}
		g.Expect(embed.ExportIsExistErr(linkErr)).To(BeTrue())
	})

	t.Run("LinkError with 'directory not empty' string is recognized", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linkErr := &os.LinkError{Op: "rename", Old: "a", New: "b",
			Err: errStringError("directory not empty"),
		}
		g.Expect(embed.ExportIsExistErr(linkErr)).To(BeTrue())
	})

	t.Run("unrelated error is not an exist error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linkErr := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: os.ErrPermission}
		g.Expect(embed.ExportIsExistErr(linkErr)).To(BeFalse())
	})
}

// TestProductionCacheFS_Methods exercises each productionCacheFS method directly
// so error branches and the success paths are covered without going through the
// full extractToCache wrapper.
func TestProductionCacheFS_Methods(t *testing.T) {
	t.Parallel()

	t.Run("MkdirAll creates nested dirs", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		path := filepath.Join(t.TempDir(), "a", "b", "c")
		g.Expect(cfs.MkdirAll(path)).To(Succeed())
		_, err := os.Stat(path)
		g.Expect(err).NotTo(HaveOccurred())
	})

	t.Run("MkdirAll on non-writable parent fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		g.Expect(cfs.MkdirAll("/no/such/root/path")).NotTo(Succeed())
	})

	t.Run("MkdirTemp creates a temp dir", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		parent := t.TempDir()
		tmp, err := cfs.MkdirTemp(parent, ".tmp-test-*")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(tmp).To(ContainSubstring(".tmp-test-"))

		defer func() { _ = os.RemoveAll(tmp) }()
	})

	t.Run("MkdirTemp under non-existent parent fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		_, err := cfs.MkdirTemp("/no/such/path", ".tmp-*")
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("WriteFile writes and RemoveAll cleans up", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		dir := t.TempDir()
		path := filepath.Join(dir, "test.bin")

		g.Expect(cfs.WriteFile(path, []byte("payload"))).To(Succeed())
		g.Expect(cfs.RemoveAll(dir)).To(Succeed())

		_, err := os.Stat(dir)
		g.Expect(os.IsNotExist(err)).To(BeTrue())
	})

	t.Run("WriteFile to non-existent dir fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		g.Expect(cfs.WriteFile("/no/such/dir/x.bin", []byte("x"))).NotTo(Succeed())
	})

	t.Run("WriteSentinel writes .complete file", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		dir := t.TempDir()
		g.Expect(cfs.WriteSentinel(dir)).To(Succeed())

		_, err := os.Stat(filepath.Join(dir, ".complete"))
		g.Expect(err).NotTo(HaveOccurred())
	})

	t.Run("WriteSentinel to non-existent dir fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		g.Expect(cfs.WriteSentinel("/no/such/dir")).NotTo(Succeed())
	})

	t.Run("StatSentinel returns false when sentinel missing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		present, err := cfs.StatSentinel(t.TempDir())
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(present).To(BeFalse())
	})

	t.Run("StatSentinel returns true when sentinel present", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		dir := t.TempDir()
		g.Expect(os.WriteFile(filepath.Join(dir, ".complete"), []byte{}, 0o600)).To(Succeed())

		present, err := cfs.StatSentinel(dir)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(present).To(BeTrue())
	})

	t.Run("Rename moves dir atomically", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		parent := t.TempDir()
		src := filepath.Join(parent, "src")
		dst := filepath.Join(parent, "dst")

		g.Expect(os.Mkdir(src, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(src, "f.txt"), []byte("hi"), 0o600)).To(Succeed())

		g.Expect(cfs.Rename(src, dst)).To(Succeed())

		_, srcErr := os.Stat(src)
		g.Expect(os.IsNotExist(srcErr)).To(BeTrue())

		_, dstErr := os.Stat(dst)
		g.Expect(dstErr).NotTo(HaveOccurred())
	})

	t.Run("Rename over non-existent path fails wrapped", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := embed.ExportProductionCacheFS()
		g.Expect(cfs.Rename("/no/such/src", "/no/such/dst")).NotTo(Succeed())
	})
}

// errStringError is a minimal error whose message matches the "directory not empty" string.
type errStringError string

func (e errStringError) Error() string { return string(e) }
