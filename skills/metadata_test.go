package skills_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
)

func TestSkillDescriptionsFitMetadataLimit(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	paths, err := filepath.Glob(filepath.Join("*", "SKILL.md"))
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(paths).NotTo(gomega.BeEmpty())

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			g := gomega.NewWithT(t)
			contents, err := os.ReadFile(path)
			g.Expect(err).NotTo(gomega.HaveOccurred())

			parts := strings.SplitN(string(contents), "---", 3)
			g.Expect(parts).To(gomega.HaveLen(3), "skill must contain YAML frontmatter")

			var metadata skillMetadata
			g.Expect(yaml.Unmarshal([]byte(parts[1]), &metadata)).To(gomega.Succeed())
			g.Expect(metadata.Description).NotTo(gomega.BeEmpty())
			g.Expect(utf8.RuneCountInString(metadata.Description)).To(
				gomega.BeNumerically("<", maxDescriptionLength),
				"parsed skill description must be fewer than %d characters", maxDescriptionLength,
			)
		})
	}
}

// unexported constants.
const (
	maxDescriptionLength = 1024
)

type skillMetadata struct {
	Description string `yaml:"description"`
}
