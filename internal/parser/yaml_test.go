package parser_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/parser"
)

// TEST-055 traces: TASK-008
// Test splitting single item with frontmatter and body
func TestSplitFrontmatter_SingleItem(t *testing.T) {
	g := NewWithT(t)

	content := `---
id: REQ-001
title: A requirement
---

# Requirement Title

This is the body content.
`

	items, err := parser.SplitFrontmatter(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(HaveLen(1))
	g.Expect(items[0].Frontmatter).To(ContainSubstring("id: REQ-001"))
	g.Expect(items[0].Body).To(ContainSubstring("# Requirement Title"))
}

// TEST-056 traces: TASK-008
// Test splitting multiple items separated by ---
func TestSplitFrontmatter_MultipleItems(t *testing.T) {
	g := NewWithT(t)

	content := `---
id: REQ-001
title: First requirement
---

Body one.

---
id: REQ-002
title: Second requirement
---

Body two.
`

	items, err := parser.SplitFrontmatter(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(HaveLen(2))
	g.Expect(items[0].Frontmatter).To(ContainSubstring("id: REQ-001"))
	g.Expect(items[1].Frontmatter).To(ContainSubstring("id: REQ-002"))
}

// TEST-057 traces: TASK-008
// Test item with frontmatter but empty body
func TestSplitFrontmatter_EmptyBody(t *testing.T) {
	g := NewWithT(t)

	content := `---
id: REQ-001
title: A requirement
---
`

	items, err := parser.SplitFrontmatter(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(HaveLen(1))
	g.Expect(items[0].Frontmatter).To(ContainSubstring("id: REQ-001"))
	g.Expect(items[0].Body).To(BeEmpty())
}

// TEST-058 traces: TASK-008
// Test handles leading whitespace/newlines
func TestSplitFrontmatter_LeadingWhitespace(t *testing.T) {
	g := NewWithT(t)

	content := `

---
id: REQ-001
---

Body.
`

	items, err := parser.SplitFrontmatter(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(HaveLen(1))
}

// TEST-059 traces: TASK-008
// Test handles frontmatter with no closing delimiter (invalid)
func TestSplitFrontmatter_NoClosingDelimiter(t *testing.T) {
	g := NewWithT(t)

	content := `---
id: REQ-001
title: Unclosed
`

	_, err := parser.SplitFrontmatter(content)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("delimiter"))
}

// TEST-060 traces: TASK-008
// Test handles no frontmatter (invalid - all items must have frontmatter)
func TestSplitFrontmatter_NoFrontmatter(t *testing.T) {
	g := NewWithT(t)

	content := `Just some plain text without frontmatter.
`

	items, err := parser.SplitFrontmatter(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(BeEmpty())
}

// TEST-061 traces: TASK-008
// Test handles empty content
func TestSplitFrontmatter_EmptyContent(t *testing.T) {
	g := NewWithT(t)

	items, err := parser.SplitFrontmatter("")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(BeEmpty())
}

// TEST-062 traces: TASK-008
// Test handles three dashes within content (not delimiter)
func TestSplitFrontmatter_DashesInContent(t *testing.T) {
	g := NewWithT(t)

	content := `---
id: REQ-001
---

Some content with --- in the middle and
--- at the start of a line but with other text.
`

	items, err := parser.SplitFrontmatter(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(HaveLen(1))
	g.Expect(items[0].Body).To(ContainSubstring("---"))
}

// TEST-063 traces: TASK-008
// Property test: valid items always have matching frontmatter
func TestSplitFrontmatter_PropertyValidItems(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		// Generate random number of items
		itemCount := rapid.IntRange(1, 5).Draw(rt, "itemCount")

		var content string
		for i := 0; i < itemCount; i++ {
			id := "REQ-" + paddedNum(i+1)
			title := rapid.StringMatching(`[A-Za-z ]{1,20}`).Draw(rt, "title")

			content += "---\n"
			content += "id: " + id + "\n"
			content += "title: " + title + "\n"
			content += "---\n"
			content += "\n"

			// Optionally add body
			if rapid.Bool().Draw(rt, "hasBody") {
				body := rapid.StringMatching(`[A-Za-z0-9 .,!?\n]{1,50}`).Draw(rt, "body")
				content += body + "\n"
			}
			content += "\n"
		}

		items, err := parser.SplitFrontmatter(content)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(len(items)).To(Equal(itemCount))

		for i, item := range items {
			expectedID := "REQ-" + paddedNum(i+1)
			g.Expect(item.Frontmatter).To(ContainSubstring(expectedID))
		}
	})
}

// paddedNum returns a 3-digit padded number string
func paddedNum(n int) string {
	if n < 10 {
		return "00" + intToStr(n)
	}
	if n < 100 {
		return "0" + intToStr(n)
	}
	return intToStr(n)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
