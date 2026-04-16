package externalsources_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestFileCache_ErrorIsCached(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	calls := 0
	wantErr := errors.New("permission denied")
	reader := func(_ string) ([]byte, error) {
		calls++
		return nil, wantErr
	}

	cache := externalsources.NewFileCache(reader)

	_, err1 := cache.Read("/no.md")
	_, err2 := cache.Read("/no.md")

	g.Expect(err1).To(MatchError(wantErr))
	g.Expect(err2).To(MatchError(wantErr))
	g.Expect(calls).To(Equal(1))
}

func TestFileCache_FirstReadReadsThrough(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	calls := 0
	reader := func(path string) ([]byte, error) {
		calls++
		return []byte("content of " + path), nil
	}

	cache := externalsources.NewFileCache(reader)

	content, err := cache.Read("/abs/path.md")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(content)).To(Equal("content of /abs/path.md"))
	g.Expect(calls).To(Equal(1))
}

func TestFileCache_RepeatReadsHitCache(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	calls := 0
	reader := func(_ string) ([]byte, error) {
		calls++
		return []byte("body"), nil
	}

	cache := externalsources.NewFileCache(reader)

	for range 3 {
		_, err := cache.Read("/p.md")
		g.Expect(err).NotTo(HaveOccurred())
	}

	g.Expect(calls).To(Equal(1))
}
