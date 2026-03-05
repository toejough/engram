package surface_test

import (
	"context"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/surface"
)

func TestFileHashStore_ClearHash_NoFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	store := surface.NewFileHashStore("/tmp/test",
		nil, nil,
		func(string) error { return os.ErrNotExist },
	)

	err := store.ClearHash(context.Background())
	g.Expect(err).NotTo(HaveOccurred())
}

func TestFileHashStore_ClearHash_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	removed := false
	store := surface.NewFileHashStore("/tmp/test",
		nil, nil,
		func(string) error { removed = true; return nil },
	)

	err := store.ClearHash(context.Background())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(removed).To(BeTrue())
}

func TestFileHashStore_LastHash_NoFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	store := surface.NewFileHashStore("/tmp/test",
		func(string) ([]byte, error) { return nil, os.ErrNotExist },
		nil, nil,
	)

	hash, err := store.LastHash(context.Background())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(hash).To(BeEmpty())
}

func TestFileHashStore_LastHash_ReadError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	store := surface.NewFileHashStore("/tmp/test",
		func(string) ([]byte, error) { return nil, os.ErrPermission },
		nil, nil,
	)

	_, err := store.LastHash(context.Background())
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("reading hash"))
	}
}

func TestFileHashStore_LastHash_ReadsAndTrims(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	store := surface.NewFileHashStore("/tmp/test",
		func(string) ([]byte, error) { return []byte("abc123\n"), nil },
		nil, nil,
	)

	hash, err := store.LastHash(context.Background())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(hash).To(Equal("abc123"))
}

func TestFileHashStore_SaveHash_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var written []byte

	store := surface.NewFileHashStore("/tmp/test",
		nil,
		func(_ string, data []byte, _ os.FileMode) error {
			written = data
			return nil
		},
		nil,
	)

	err := store.SaveHash(context.Background(), "deadbeef")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(written)).To(Equal("deadbeef\n"))
}

func TestFileHashStore_SaveHash_WriteError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	store := surface.NewFileHashStore("/tmp/test",
		nil,
		func(string, []byte, os.FileMode) error { return os.ErrPermission },
		nil,
	)

	err := store.SaveHash(context.Background(), "deadbeef")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("saving hash"))
	}
}
