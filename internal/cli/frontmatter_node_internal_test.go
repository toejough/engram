package cli

import (
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
)

// TestParseFrontmatterMapping covers each shape a frontmatter block can
// take: a mapping, an empty block (an empty mapping), a non-mapping, and
// malformed YAML.
func TestParseFrontmatterMapping(t *testing.T) {
	t.Parallel()

	t.Run("mapping", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		mapping, err := parseFrontmatterMapping([]byte("type: fact\nluhmann_old: \"12\"\n"))
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil || mapping == nil {
			return
		}

		g.Expect(mapping.Kind).To(Equal(yaml.MappingNode))
		g.Expect(mapping.Content).To(HaveLen(4))
	})

	t.Run("empty block", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		mapping, err := parseFrontmatterMapping(nil)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil || mapping == nil {
			return
		}

		g.Expect(mapping.Kind).To(Equal(yaml.MappingNode))
		g.Expect(mapping.Content).To(BeEmpty())
	})

	t.Run("not a mapping", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		_, err := parseFrontmatterMapping([]byte("- a\n- b\n"))
		g.Expect(err).To(MatchError(errFrontmatterNotMapping))
	})

	t.Run("malformed", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		_, err := parseFrontmatterMapping([]byte("type: [unclosed\n"))
		g.Expect(err).To(HaveOccurred())
	})
}
