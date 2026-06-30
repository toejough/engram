package cli

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

// TestFindNote_NotFoundError_QuotesOriginalRef verifies the not-found error
// quotes the caller's original ref (e.g. the ".md" form), not the normalized
// one, so resituate surfaces what the user actually typed.
func TestFindNote_NotFoundError_QuotesOriginalRef(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{{Basename: "1.linting", LuhmannID: "1"}}

	_, err := findNote(notes, "99.missing.md")
	g.Expect(err).To(MatchError(ContainSubstring("\"99.missing.md\"")))
}

// TestFindNote_ResolvesMdAndWikilinkRefs verifies that findNote normalizes
// the target ref before matching, accepting all accepted forms:
// bare Luhmann id, full basename (no .md), .md-suffixed basename, and [[wikilink]].
func TestFindNote_ResolvesMdAndWikilinkRefs(t *testing.T) {
	t.Parallel()

	notes := []vaultgraph.Note{{Basename: "1.linting", LuhmannID: "1"}}

	// pathOf("1.linting") = "1.linting.md"
	const wantPath = "1.linting.md"

	cases := []struct {
		name   string
		target string
	}{
		{"full basename", "1.linting"},
		{"bare luhmann id", "1"},
		{"md suffix", "1.linting.md"},
		{"wikilink", "[[1.linting]]"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			got, err := findNote(notes, tc.target)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(got).To(Equal(wantPath))
		})
	}
}
