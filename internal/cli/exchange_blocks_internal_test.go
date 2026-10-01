package cli

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestHeldLinkNodes: the parent's link entries are reused only when they
// line up one-to-one with the decoded links as mappings.
func TestHeldLinkNodes(t *testing.T) {
	t.Parallel()

	twoLinks := parentLinks{Links: []parentLink{{Note: "a"}, {Note: "b"}}}

	for name, tc := range map[string]struct {
		frontmatter string
		before      parentLinks
		held        int
	}{
		"aligned":        {"links:\n    - note: a\n    - note: b\n", twoLinks, 2},
		"no links":       {"vault: v\n", twoLinks, 0},
		"not a sequence": {"links: none\n", twoLinks, 0},
		"length differs": {"links:\n    - note: a\n", twoLinks, 0},
		"scalar entries": {"links: [a, b]\n", twoLinks, 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			parent, err := parseFrontmatterMapping([]byte(tc.frontmatter))
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil || parent == nil {
				return
			}

			g.Expect(heldLinkNodes(parent, tc.before)).To(HaveLen(tc.held))
		})
	}
}
