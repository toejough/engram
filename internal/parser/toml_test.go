package parser_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/parser"
	"github.com/toejough/projctl/internal/trace"
)

// TEST-123 traces: TASK-018
// Test parsing valid TOML content
func TestParseTOML_Valid(t *testing.T) {
	g := NewWithT(t)

	content := `id = "REQ-001"
type = "REQ"
project = "test-project"
title = "A requirement"
status = "active"
created = 2024-01-15T10:30:00Z
updated = 2024-01-16T14:00:00Z
traces_to = ["REQ-000"]
`

	result, err := parser.ParseTOML(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(1))
	g.Expect(result.Items[0].ID).To(Equal("REQ-001"))
	g.Expect(result.Items[0].Type).To(Equal(trace.NodeTypeREQ))
	g.Expect(result.Items[0].Project).To(Equal("test-project"))
	g.Expect(result.Items[0].TracesTo).To(Equal([]string{"REQ-000"}))
	g.Expect(result.Deprecated).To(BeTrue())
}

// TEST-124 traces: TASK-018
// Test parsing TOML with multiple items
func TestParseTOML_MultipleItems(t *testing.T) {
	g := NewWithT(t)

	content := `[[item]]
id = "REQ-001"
type = "REQ"
project = "test-project"
title = "First requirement"
status = "active"
created = 2024-01-15T10:30:00Z
updated = 2024-01-16T14:00:00Z

[[item]]
id = "REQ-002"
type = "REQ"
project = "test-project"
title = "Second requirement"
status = "draft"
created = 2024-01-17T08:00:00Z
updated = 2024-01-17T08:00:00Z
`

	result, err := parser.ParseTOML(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(2))
	g.Expect(result.Items[0].ID).To(Equal("REQ-001"))
	g.Expect(result.Items[1].ID).To(Equal("REQ-002"))
}

// TEST-125 traces: TASK-018
// Test parsing invalid TOML returns error
func TestParseTOML_Invalid(t *testing.T) {
	g := NewWithT(t)

	content := `id = "REQ-001
broken toml syntax`

	_, err := parser.ParseTOML(content)
	g.Expect(err).To(HaveOccurred())
}

// TEST-126 traces: TASK-018
// Test parsing TOML with missing required field
func TestParseTOML_MissingField(t *testing.T) {
	g := NewWithT(t)

	content := `id = "REQ-001"
type = "REQ"
title = "Missing project"
status = "active"
`

	_, err := parser.ParseTOML(content)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Project"))
}

// TEST-127 traces: TASK-018
// Test result indicates deprecation
func TestParseTOML_DeprecatedFlag(t *testing.T) {
	g := NewWithT(t)

	content := `id = "REQ-001"
type = "REQ"
project = "test-project"
title = "A requirement"
status = "active"
created = 2024-01-15T10:30:00Z
updated = 2024-01-16T14:00:00Z
`

	result, err := parser.ParseTOML(content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Deprecated).To(BeTrue())
}
