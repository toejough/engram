package parser_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/parser"
	"github.com/toejough/projctl/internal/trace"
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

// TEST-064 traces: TASK-009
// Test parsing valid YAML frontmatter into TraceItem
func TestParseFrontmatter_ValidREQ(t *testing.T) {
	g := NewWithT(t)

	frontmatter := `id: REQ-001
type: REQ
project: test-project
title: A test requirement
status: active
created: 2024-01-15T10:30:00Z
updated: 2024-01-16T14:00:00Z
traces_to:
  - REQ-000
tags:
  - priority:high
  - scope:mvp`

	item, err := parser.ParseFrontmatter(frontmatter)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(item.ID).To(Equal("REQ-001"))
	g.Expect(item.Type).To(Equal(trace.NodeTypeREQ))
	g.Expect(item.Project).To(Equal("test-project"))
	g.Expect(item.Title).To(Equal("A test requirement"))
	g.Expect(item.Status).To(Equal("active"))
	g.Expect(item.TracesTo).To(Equal([]string{"REQ-000"}))
	g.Expect(item.Tags).To(Equal([]string{"priority:high", "scope:mvp"}))
	g.Expect(item.Created.Year()).To(Equal(2024))
	g.Expect(item.Updated.Year()).To(Equal(2024))
}

// TEST-065 traces: TASK-009
// Test parsing TASK type frontmatter
func TestParseFrontmatter_ValidTASK(t *testing.T) {
	g := NewWithT(t)

	frontmatter := `id: TASK-042
type: TASK
project: test-project
title: Implement feature X
status: draft
created: 2024-02-01T09:00:00Z
updated: 2024-02-01T09:00:00Z
traces_to:
  - ARCH-005
  - DES-003`

	item, err := parser.ParseFrontmatter(frontmatter)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(item.ID).To(Equal("TASK-042"))
	g.Expect(item.Type).To(Equal(trace.NodeTypeTASK))
	g.Expect(item.TracesTo).To(ConsistOf("ARCH-005", "DES-003"))
}

// TEST-066 traces: TASK-009
// Test parsing TEST type frontmatter with location fields
func TestParseFrontmatter_ValidTEST(t *testing.T) {
	g := NewWithT(t)

	frontmatter := `id: TEST-001
type: TEST
project: test-project
title: Unit test for validation
status: active
created: 2024-01-20T08:00:00Z
updated: 2024-01-20T08:00:00Z
traces_to:
  - TASK-042
location: internal/validate_test.go
line: 45
function: TestValidateInput`

	item, err := parser.ParseFrontmatter(frontmatter)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(item.ID).To(Equal("TEST-001"))
	g.Expect(item.Type).To(Equal(trace.NodeTypeTEST))
	g.Expect(item.Location).To(Equal("internal/validate_test.go"))
	g.Expect(item.Line).To(Equal(45))
	g.Expect(item.Function).To(Equal("TestValidateInput"))
}

// TEST-067 traces: TASK-009
// Test missing required field returns error
func TestParseFrontmatter_MissingID(t *testing.T) {
	g := NewWithT(t)

	frontmatter := `type: REQ
project: test-project
title: Missing ID
status: active
created: 2024-01-15T10:30:00Z
updated: 2024-01-16T14:00:00Z`

	_, err := parser.ParseFrontmatter(frontmatter)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ID"))
}

// TEST-068 traces: TASK-009
// Test missing project field returns error
func TestParseFrontmatter_MissingProject(t *testing.T) {
	g := NewWithT(t)

	frontmatter := `id: REQ-001
type: REQ
title: Missing project
status: active
created: 2024-01-15T10:30:00Z
updated: 2024-01-16T14:00:00Z`

	_, err := parser.ParseFrontmatter(frontmatter)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Project"))
}

// TEST-069 traces: TASK-009
// Test invalid YAML syntax returns error
func TestParseFrontmatter_InvalidYAML(t *testing.T) {
	g := NewWithT(t)

	frontmatter := `id: REQ-001
type: REQ
  bad indentation here
project: test-project`

	_, err := parser.ParseFrontmatter(frontmatter)
	g.Expect(err).To(HaveOccurred())
}

// TEST-070 traces: TASK-009
// Test invalid date format returns error
func TestParseFrontmatter_InvalidDateFormat(t *testing.T) {
	g := NewWithT(t)

	frontmatter := `id: REQ-001
type: REQ
project: test-project
title: Bad date
status: active
created: not-a-date
updated: 2024-01-16T14:00:00Z`

	_, err := parser.ParseFrontmatter(frontmatter)
	g.Expect(err).To(HaveOccurred())
}

// TEST-071 traces: TASK-009
// Test invalid ID format returns error (via TraceItem.Validate)
func TestParseFrontmatter_InvalidIDFormat(t *testing.T) {
	g := NewWithT(t)

	frontmatter := `id: INVALID
type: REQ
project: test-project
title: Bad ID format
status: active
created: 2024-01-15T10:30:00Z
updated: 2024-01-16T14:00:00Z`

	_, err := parser.ParseFrontmatter(frontmatter)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ID"))
}

// TEST-072 traces: TASK-009
// Test invalid status value returns error
func TestParseFrontmatter_InvalidStatus(t *testing.T) {
	g := NewWithT(t)

	frontmatter := `id: REQ-001
type: REQ
project: test-project
title: Bad status
status: invalid-status
created: 2024-01-15T10:30:00Z
updated: 2024-01-16T14:00:00Z`

	_, err := parser.ParseFrontmatter(frontmatter)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Status"))
}

// TEST-073 traces: TASK-009
// Test empty traces_to is allowed
func TestParseFrontmatter_EmptyTracesTo(t *testing.T) {
	g := NewWithT(t)

	frontmatter := `id: REQ-001
type: REQ
project: test-project
title: Root requirement
status: active
created: 2024-01-15T10:30:00Z
updated: 2024-01-16T14:00:00Z`

	item, err := parser.ParseFrontmatter(frontmatter)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(item.TracesTo).To(BeEmpty())
}

// TEST-074 traces: TASK-009
// Property test: valid frontmatter parses successfully
func TestParseFrontmatter_PropertyValid(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		nodeTypes := []string{"REQ", "DES", "ARCH", "TASK"}
		nodeType := rapid.SampledFrom(nodeTypes).Draw(rt, "nodeType")
		num := rapid.IntRange(1, 999).Draw(rt, "num")
		id := nodeType + "-" + paddedNum(num)

		project := rapid.StringMatching(`[a-z][a-z0-9\-]{2,15}`).Draw(rt, "project")
		title := rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{5,30}`).Draw(rt, "title")
		statuses := []string{"draft", "active", "completed", "deprecated"}
		status := rapid.SampledFrom(statuses).Draw(rt, "status")

		frontmatter := "id: " + id + "\n"
		frontmatter += "type: " + nodeType + "\n"
		frontmatter += "project: " + project + "\n"
		frontmatter += "title: " + title + "\n"
		frontmatter += "status: " + status + "\n"
		frontmatter += "created: 2024-01-15T10:30:00Z\n"
		frontmatter += "updated: 2024-01-16T14:00:00Z\n"

		item, err := parser.ParseFrontmatter(frontmatter)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(item.ID).To(Equal(id))
		g.Expect(item.Project).To(Equal(project))
		g.Expect(item.Status).To(Equal(status))
	})
}
