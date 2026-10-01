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

// TestSetMappingValueOrdered covers every placement: replace in place,
// insert after the nearest preceding key, before the nearest following key
// when none precedes, and append when the mapping holds no ordered key.
func TestSetMappingValueOrdered(t *testing.T) {
	t.Parallel()

	order := []string{"a", "b", "c", "d"}

	for name, tc := range map[string]struct{ frontmatter, key, want string }{
		"replace":        {"a: 1\nb: 2\n", "b", "a: 1\nb: new\n"},
		"after previous": {"a: 1\nd: 4\n", "b", "a: 1\nb: new\nd: 4\n"},
		"before next":    {"x: 0\nd: 4\n", "b", "x: 0\nb: new\nd: 4\n"},
		"append":         {"x: 0\n", "c", "x: 0\nc: new\n"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			mapping, err := parseFrontmatterMapping([]byte(tc.frontmatter))
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil || mapping == nil {
				return
			}

			setMappingValueOrdered(mapping, tc.key, encodeNode("new"), order)

			rendered, marshalErr := yaml.Marshal(mapping)
			g.Expect(marshalErr).NotTo(HaveOccurred())
			g.Expect(string(rendered)).To(Equal(tc.want))
		})
	}
}
