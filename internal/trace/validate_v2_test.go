package trace_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-179 traces: TASK-029
// Test ValidateV2 passes with valid graph
func TestValidateV2_Valid(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create docs directory with valid trace chain
	docsDir := filepath.Join(dir, "docs")
	g.Expect(os.Mkdir(docsDir, 0755)).To(Succeed())

	// REQ -> ARCH -> TASK chain
	writeDoc(t, docsDir, "requirements.md", `---
id: REQ-001
type: REQ
project: test
title: A requirement
status: active
---
`)

	writeDoc(t, docsDir, "architecture.md", `---
id: ARCH-001
type: ARCH
project: test
title: Architecture decision
status: active
traces_to:
  - REQ-001
---
`)

	writeDoc(t, docsDir, "tasks.md", `---
id: TASK-001
type: TASK
project: test
title: Implementation task
status: active
traces_to:
  - ARCH-001
---
`)

	// Create test file that traces to task
	writeDoc(t, dir, "foo_test.go", `package foo_test

// TEST-001 traces: TASK-001
// Test the feature
func TestFeature(t *testing.T) {}
`)

	result, err := trace.ValidateV2(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeTrue())
	g.Expect(result.Errors).To(BeEmpty())
}

// TEST-180 traces: TASK-029
// Test ValidateV2 fails with cycle
func TestValidateV2_WithCycle(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	docsDir := filepath.Join(dir, "docs")
	g.Expect(os.Mkdir(docsDir, 0755)).To(Succeed())

	// Create a cycle: REQ-001 -> REQ-002 -> REQ-001
	writeDoc(t, docsDir, "requirements.md", `---
id: REQ-001
type: REQ
project: test
title: First
status: active
traces_to:
  - REQ-002
---
---
id: REQ-002
type: REQ
project: test
title: Second
status: active
traces_to:
  - REQ-001
---
`)

	result, err := trace.ValidateV2(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("cycle")))
}

// TEST-181 traces: TASK-029
// Test ValidateV2 fails with dangling reference
func TestValidateV2_DanglingRef(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	docsDir := filepath.Join(dir, "docs")
	g.Expect(os.Mkdir(docsDir, 0755)).To(Succeed())

	// TASK traces to non-existent ARCH
	writeDoc(t, docsDir, "tasks.md", `---
id: TASK-001
type: TASK
project: test
title: Task
status: active
traces_to:
  - ARCH-999
---
`)

	result, err := trace.ValidateV2(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeFalse())
	g.Expect(result.Errors).To(ContainElement(ContainSubstring("ARCH-999")))
}

// TEST-182 traces: TASK-029
// Test ValidateV2 reports coverage warnings
func TestValidateV2_CoverageWarnings(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	docsDir := filepath.Join(dir, "docs")
	g.Expect(os.Mkdir(docsDir, 0755)).To(Succeed())

	// REQ with no downstream ARCH
	writeDoc(t, docsDir, "requirements.md", `---
id: REQ-001
type: REQ
project: test
title: Orphan requirement
status: active
---
`)

	result, err := trace.ValidateV2(dir)
	g.Expect(err).ToNot(HaveOccurred())
	// Coverage gaps are warnings, not errors
	g.Expect(result.Pass).To(BeTrue())
	g.Expect(result.Warnings).To(ContainElement(ContainSubstring("REQ-001")))
}

// TEST-183 traces: TASK-029
// Test ValidateV2 with empty project
func TestValidateV2_EmptyProject(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	result, err := trace.ValidateV2(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeTrue())
	g.Expect(result.Errors).To(BeEmpty())
}

// TEST-184 traces: TASK-029
// Test ValidateV2 returns node count
func TestValidateV2_NodeCount(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	docsDir := filepath.Join(dir, "docs")
	g.Expect(os.Mkdir(docsDir, 0755)).To(Succeed())

	writeDoc(t, docsDir, "requirements.md", `---
id: REQ-001
type: REQ
project: test
title: First
status: active
---
---
id: REQ-002
type: REQ
project: test
title: Second
status: active
---
`)

	result, err := trace.ValidateV2(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.NodeCount).To(Equal(2))
}

func writeDoc(t *testing.T, dir, name, content string) {
	t.Helper()
	g := NewWithT(t)
	g.Expect(os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)).To(Succeed())
}
