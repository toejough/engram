package parser_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/parser"
)

// TEST-116 traces: TASK-017
// Test detecting YAML format
func TestDetectFormat_YAML(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `---
id: REQ-001
type: REQ
---

Body content.
`

	format, err := parser.DetectFormat(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(format).To(Equal("yaml"))
}

// TEST-117 traces: TASK-017
// Test detecting TOML format
func TestDetectFormat_TOML(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `id = "REQ-001"
type = "REQ"
title = "A requirement"
`

	format, err := parser.DetectFormat(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(format).To(Equal("toml"))
}

// TEST-118 traces: TASK-017
// Test empty content returns error
func TestDetectFormat_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := parser.DetectFormat("")
	g.Expect(err).To(HaveOccurred())
}

// TEST-119 traces: TASK-017
// Test whitespace-only content returns error
func TestDetectFormat_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := parser.DetectFormat("   \n\t\n   ")
	g.Expect(err).To(HaveOccurred())
}

// TEST-120 traces: TASK-017
// Test unrecognized format returns error
func TestDetectFormat_Unknown(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `Just plain text without any format markers.
No YAML frontmatter or TOML key=value pairs.
`

	_, err := parser.DetectFormat(content)
	g.Expect(err).To(HaveOccurred())
}

// TEST-121 traces: TASK-017
// Test YAML with leading whitespace
func TestDetectFormat_YAMLWithWhitespace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `

---
id: REQ-001
---
`

	format, err := parser.DetectFormat(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(format).To(Equal("yaml"))
}

// TEST-122 traces: TASK-017
// Property test: content starting with --- is YAML
func TestDetectFormat_PropertyYAML(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		body := rapid.StringMatching(`[A-Za-z0-9: \n]{1,50}`).Draw(rt, "body")
		content := "---\n" + body + "\n---"

		format, err := parser.DetectFormat(content)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(format).To(Equal("yaml"))
	})
}
