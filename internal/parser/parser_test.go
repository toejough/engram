package parser_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/parser"
)

// TEST-131 traces: TASK-019
// Test parsing empty content returns error
func TestParseDoc_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := parser.ParseDoc("")
	g.Expect(err).To(HaveOccurred())
}

// TEST-132 traces: TASK-019
// Test parsing invalid YAML returns error
func TestParseDoc_InvalidYAML(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `---
id: REQ-001
  bad indentation
---
`

	_, err := parser.ParseDoc(content)
	g.Expect(err).To(HaveOccurred())
}

// TEST-129 traces: TASK-019
// Test parsing TOML format document (deprecated)
func TestParseDoc_TOML(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `id = "REQ-001"
type = "REQ"
project = "test-project"
title = "A requirement"
status = "active"
created = 2024-01-15T10:30:00Z
updated = 2024-01-16T14:00:00Z
`

	result, err := parser.ParseDoc(content)
	g.Expect(err).ToNot(HaveOccurred())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.Items).To(HaveLen(1))
	g.Expect(result.Items[0].ID).To(Equal("REQ-001"))
	g.Expect(result.Format).To(Equal("toml"))
	g.Expect(result.Deprecated).To(BeTrue())
}

// TEST-130 traces: TASK-019
// Test parsing unknown format returns error
func TestParseDoc_UnknownFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `Just plain text without format markers.`

	_, err := parser.ParseDoc(content)
	g.Expect(err).To(HaveOccurred())
}

// TEST-128 traces: TASK-019
// Test parsing YAML format document
func TestParseDoc_YAML(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `---
id: REQ-001
type: REQ
project: test-project
title: A requirement
status: active
created: 2024-01-15T10:30:00Z
updated: 2024-01-16T14:00:00Z
---

Body content.
`

	result, err := parser.ParseDoc(content)
	g.Expect(err).ToNot(HaveOccurred())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.Items).To(HaveLen(1))
	g.Expect(result.Items[0].ID).To(Equal("REQ-001"))
	g.Expect(result.Format).To(Equal("yaml"))
	g.Expect(result.Deprecated).To(BeFalse())
}
