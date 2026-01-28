package parser_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/parser"
	"github.com/toejough/projctl/internal/trace"
)

// Mock file system for collection tests
type mockCollectFS struct {
	dirs   map[string]bool
	files  map[string]string // path -> content
	walked []string          // paths returned during walk
}

func (m *mockCollectFS) DirExists(path string) bool {
	return m.dirs[path]
}

func (m *mockCollectFS) FileExists(path string) bool {
	_, exists := m.files[path]
	return exists
}

func (m *mockCollectFS) ReadFile(path string) (string, error) {
	content, exists := m.files[path]
	if !exists {
		return "", &fileNotFoundError{path: path}
	}
	return content, nil
}

func (m *mockCollectFS) Walk(root string, fn func(path string, isDir bool) error) error {
	// Walk through predefined paths
	for _, p := range m.walked {
		isDir := m.dirs[p]
		if err := fn(p, isDir); err != nil {
			continue // Skip on error
		}
	}
	return nil
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

// TEST-165 traces: TASK-027
// Test CollectTraceItems finds docs and returns items
func TestCollectTraceItems_DocsOnly(t *testing.T) {
	g := NewWithT(t)

	fs := &mockCollectFS{
		dirs: map[string]bool{
			"/project/docs": true,
		},
		files: map[string]string{
			"/project/docs/requirements.md": `---
id: REQ-001
type: REQ
project: test
title: A requirement
status: active
---
`,
		},
		walked: []string{"/project"}, // No test files
	}

	result, err := parser.CollectTraceItems("/project", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(1))
	g.Expect(result.Items[0].ID).To(Equal("REQ-001"))
}

// TEST-166 traces: TASK-027
// Test CollectTraceItems finds test files and returns items
func TestCollectTraceItems_TestsOnly(t *testing.T) {
	g := NewWithT(t)

	fs := &mockCollectFS{
		dirs: map[string]bool{
			"/project": true,
		},
		files: map[string]string{
			"/project/foo_test.go": `package foo_test

// TEST-001 traces: TASK-001
// Test something
func TestSomething(t *testing.T) {}
`,
		},
		walked: []string{
			"/project",
			"/project/foo_test.go",
		},
	}

	result, err := parser.CollectTraceItems("/project", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(1))
	g.Expect(result.Items[0].ID).To(Equal("TEST-001"))
	g.Expect(result.Items[0].Type).To(Equal(trace.NodeTypeTEST))
}

// TEST-167 traces: TASK-027
// Test CollectTraceItems combines docs and tests
func TestCollectTraceItems_Combined(t *testing.T) {
	g := NewWithT(t)

	fs := &mockCollectFS{
		dirs: map[string]bool{
			"/project":      true,
			"/project/docs": true,
		},
		files: map[string]string{
			"/project/docs/requirements.md": `---
id: REQ-001
type: REQ
project: test
title: A requirement
status: active
---
`,
			"/project/pkg/api_test.go": `package pkg_test

// TEST-001 traces: TASK-001
// Test API
func TestAPI(t *testing.T) {}
`,
		},
		walked: []string{
			"/project",
			"/project/pkg",
			"/project/pkg/api_test.go",
		},
	}

	result, err := parser.CollectTraceItems("/project", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(2))

	// Check we have both types
	var hasREQ, hasTEST bool
	for _, item := range result.Items {
		if item.Type == trace.NodeTypeREQ {
			hasREQ = true
		}
		if item.Type == trace.NodeTypeTEST {
			hasTEST = true
		}
	}
	g.Expect(hasREQ).To(BeTrue())
	g.Expect(hasTEST).To(BeTrue())
}

// TEST-168 traces: TASK-027
// Test CollectTraceItems with empty project returns empty
func TestCollectTraceItems_Empty(t *testing.T) {
	g := NewWithT(t)

	fs := &mockCollectFS{
		dirs:   map[string]bool{"/project": true},
		files:  map[string]string{},
		walked: []string{"/project"},
	}

	result, err := parser.CollectTraceItems("/project", fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(BeEmpty())
}

// TEST-169 traces: TASK-027
// Test CollectTraceItems returns parse errors
func TestCollectTraceItems_ParseErrors(t *testing.T) {
	g := NewWithT(t)

	fs := &mockCollectFS{
		dirs: map[string]bool{
			"/project/docs": true,
		},
		files: map[string]string{
			"/project/docs/requirements.md": `---
id: invalid format no type
---
`,
		},
		walked: []string{"/project"},
	}

	result, err := parser.CollectTraceItems("/project", fs)
	// Should return result with errors collected, not fail completely
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Errors).ToNot(BeEmpty())
}

// TEST-170 traces: TASK-027
// Test CollectTraceItems extracts project name for tests
func TestCollectTraceItems_TestProjectName(t *testing.T) {
	g := NewWithT(t)

	fs := &mockCollectFS{
		dirs: map[string]bool{
			"/project":      true,
			"/project/docs": true,
		},
		files: map[string]string{
			// Doc file establishes project name
			"/project/docs/requirements.md": `---
id: REQ-001
type: REQ
project: my-project
title: A requirement
status: active
---
`,
			"/project/foo_test.go": `package foo_test

// TEST-001 traces: TASK-001
// Test something
func TestSomething(t *testing.T) {}
`,
		},
		walked: []string{
			"/project",
			"/project/foo_test.go",
		},
	}

	result, err := parser.CollectTraceItems("/project", fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Find the TEST item and check its project name
	for _, item := range result.Items {
		if item.Type == trace.NodeTypeTEST {
			g.Expect(item.Project).To(Equal("my-project"))
			return
		}
	}
	g.Fail("No TEST item found")
}
