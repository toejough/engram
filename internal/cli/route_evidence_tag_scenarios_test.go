package cli_test

// Scenario-ID test of the Route product capability (docs/specs/product/route.md): 792a17k1c, the tag format of the
// evidence note. It drives the real command path over the in-memory vault of the controlled-vocabulary scenarios
// (cv1Vault) and the note-writing helpers (nw1), so it touches no real filesystem, home, git or embedding model.

import (
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a17k1c_TheTagFormatIsValidated verifies 792a17k1c (route.md:306): engram learn writes a fact tagged
// work-kind/<k>, tier/<t> and outcome/<o>, and refuses before the write a tag that is not a family or a
// family/value of lowercase letters, digits and hyphens. The character rule is the binary's: the route skill states the
// three families and no character set.
func Test792a17k1c_TheTagFormatIsValidated(t *testing.T) {
	t.Parallel()

	invalidTags := []string{
		"tier/Cheap",           // uppercase
		"work-kind/go_package", // underscore
		"outcome/pass fail",    // space
		"outcome/",             // empty value
		"tier=cheap",           // wrong separator
		"work-kind/go/package", // two slashes
	}

	for _, tag := range invalidTags {
		t.Run(tag, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a dispatch record in an existing vault
			vault := newCV1Vault()
			before := vault.fsys.snapshot()

			// When its tags are written and one of them is outside the character set
			flags := nw1FactFlags().add("--tag", "work-kind/go-package-implementation").add("--tag", tag)
			result := nw1Learn(vault, "fact", flags)

			// Then the command is refused naming the tag format, and no note is written
			g.Expect(result.stderr).To(ContainSubstring("tag must be <family> or <family>/<value>"))
			g.Expect(vault.fsys.snapshot()).To(Equal(before))
		})
	}

	t.Run("a valid evidence tag set is written", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		// Given a dispatch record in an empty vault
		vault := newCV1Vault()

		// When its tags are work-kind/<k>, tier/<t> and outcome/<o> with lowercase letters, digits and hyphens
		flags := nw1FactFlags().add("--tag", "work-kind/go-package-implementation").add("--tag", "tier/cheap").
			add("--tag", "outcome/pass")
		result := nw1Learn(vault, "fact", flags)

		// Then the note is written with the three tags
		g.Expect(result.stderr).To(BeEmpty())
		g.Expect(vault.tagsOf("nw1-note")).To(ConsistOf(
			"work-kind/go-package-implementation", "tier/cheap", "outcome/pass"))
	})
}
