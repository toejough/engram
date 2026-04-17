package externalsources_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestProjectSlug_DashSubstitution(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(externalsources.ProjectSlug("/Users/joe/repos/engram")).
		To(Equal("-Users-joe-repos-engram"))
	g.Expect(externalsources.ProjectSlug("/home/alice/work")).
		To(Equal("-home-alice-work"))
}

func TestProjectSlug_RootDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(externalsources.ProjectSlug("/")).To(Equal("-"))
}
