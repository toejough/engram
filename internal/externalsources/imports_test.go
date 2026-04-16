package externalsources_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestExpandImports_DepthCappedAt5Hops(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/0.md": []byte("@1.md\n"),
		"/1.md": []byte("@2.md\n"),
		"/2.md": []byte("@3.md\n"),
		"/3.md": []byte("@4.md\n"),
		"/4.md": []byte("@5.md\n"),
		"/5.md": []byte("@6.md\n"),
		"/6.md": []byte("never reached\n"),
	}

	imports := externalsources.ExpandImports("/0.md", fakeReader(contents))

	paths := make([]string, 0, len(imports))
	for _, file := range imports {
		paths = append(paths, file.Path)
	}

	g.Expect(paths).To(ContainElements("/1.md", "/2.md", "/3.md", "/4.md", "/5.md"))
	g.Expect(paths).NotTo(ContainElement("/6.md"))
}

func TestExpandImports_NoImports(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/a/CLAUDE.md": []byte("Just text, no imports.\n"),
	}

	imports := externalsources.ExpandImports("/a/CLAUDE.md", fakeReader(contents))
	g.Expect(imports).To(BeEmpty())
}

func TestExpandImports_RecursiveAndCycleSafe(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/a.md": []byte("@b.md\n"),
		"/b.md": []byte("@c.md\n"),
		"/c.md": []byte("@a.md\n"), // cycle back to a.md
	}

	imports := externalsources.ExpandImports("/a.md", fakeReader(contents))

	paths := make([]string, 0, len(imports))
	for _, file := range imports {
		paths = append(paths, file.Path)
	}

	g.Expect(paths).To(ConsistOf("/b.md", "/c.md"))
}

func TestExpandImports_RelativePath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/a/CLAUDE.md":   []byte("See @docs/git.md for details.\n"),
		"/a/docs/git.md": []byte("Git workflow.\n"),
	}

	imports := externalsources.ExpandImports("/a/CLAUDE.md", fakeReader(contents))

	paths := make([]string, 0, len(imports))
	for _, file := range imports {
		paths = append(paths, file.Path)
	}

	g.Expect(paths).To(ConsistOf("/a/docs/git.md"))
}

func TestExpandImports_TildePathPreserved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/a/CLAUDE.md": []byte("See @~/notes.md for details.\n"),
		"~/notes.md":   []byte("Notes.\n"),
	}

	imports := externalsources.ExpandImports("/a/CLAUDE.md", fakeReader(contents))

	paths := make([]string, 0, len(imports))
	for _, file := range imports {
		paths = append(paths, file.Path)
	}

	g.Expect(paths).To(ConsistOf("~/notes.md"))
}

// fakeReader returns a ReaderFunc backed by an in-memory map.
func fakeReader(contents map[string][]byte) externalsources.ReaderFunc {
	return func(path string) ([]byte, error) {
		body, ok := contents[path]
		if !ok {
			return nil, nil
		}

		return body, nil
	}
}
