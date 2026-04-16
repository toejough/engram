package externalsources_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestParseFrontmatter_FoldedDescription(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte(`---
name: learn
description: >
  Use after completing a task,
  finishing work, or changing direction.
---

body
`)

	fm, _ := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Name).To(Equal("learn"))
	g.Expect(fm.Description).To(ContainSubstring("Use after completing a task"))
	g.Expect(fm.Description).To(ContainSubstring("changing direction."))
}

func TestParseFrontmatter_FoldedDescriptionThenName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte(`---
description: >
  folded line one,
  folded line two.
name: trailing-name
---

body
`)

	fm, _ := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Description).To(ContainSubstring("folded line one"))
	g.Expect(fm.Description).To(ContainSubstring("folded line two."))
	g.Expect(fm.Name).To(Equal("trailing-name"))
}

func TestParseFrontmatter_NameAndDescription(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte(`---
name: prepare
description: Use before starting new work to load context
---

# Skill body
`)

	fm, rest := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Name).To(Equal("prepare"))
	g.Expect(fm.Description).To(Equal("Use before starting new work to load context"))
	g.Expect(string(rest)).To(ContainSubstring("# Skill body"))
	g.Expect(string(rest)).NotTo(ContainSubstring("---"))
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte("# Just a markdown file\n\nNo frontmatter here.\n")

	fm, rest := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Name).To(BeEmpty())
	g.Expect(fm.Description).To(BeEmpty())
	g.Expect(fm.Paths).To(BeEmpty())
	g.Expect(string(rest)).To(Equal(string(body)))
}

func TestParseFrontmatter_PathsList(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte(`---
paths:
  - "src/api/**/*.ts"
  - "lib/**/*.ts"
---

content
`)

	fm, _ := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Paths).To(ConsistOf("src/api/**/*.ts", "lib/**/*.ts"))
}

func TestParseFrontmatter_PathsListThenName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte(`---
paths:
  - "lib/**/*.ts"
name: after-list
---

body
`)

	fm, _ := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Paths).To(ConsistOf("lib/**/*.ts"))
	g.Expect(fm.Name).To(Equal("after-list"))
}

func TestParseFrontmatter_UnterminatedFence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	body := []byte("---\nname: orphan\nno closing fence here\n")

	fm, rest := externalsources.ParseFrontmatter(body)
	g.Expect(fm.Name).To(BeEmpty())
	g.Expect(string(rest)).To(Equal(string(body)))
}
