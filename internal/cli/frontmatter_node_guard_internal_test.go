package cli

import (
	"testing"

	. "github.com/onsi/gomega"
)

// Test792b6c7a5_VerifyFrontmatterDecodes is the write guard: rendered
// content whose frontmatter does not decode — a dangling alias, a
// non-mapping, or no frontmatter at all — is refused.
func Test792b6c7a5_VerifyFrontmatterDecodes(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		content string
		ok      bool
	}{
		"dangling alias": {"---\ntype: fact\nissue: *a\n---\n\nbody\n", false},
		"not a mapping":  {"---\n- a\n---\n\nbody\n", false},
		"no frontmatter": {"just a body\n", false},
		"valid":          {"---\ntype: fact\nx: &a v\ny: *a\n---\n\nbody\n", true},
		"empty block":    {"---\n---\n\nbody\n", true},
		"plain mapping":  {"---\ntype: fact\n---\n", true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			err := verifyFrontmatterDecodes(tc.content)
			if tc.ok {
				g.Expect(err).NotTo(HaveOccurred())

				return
			}

			g.Expect(err).To(MatchError(errFrontmatterUndecodable))
		})
	}
}
