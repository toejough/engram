package trace_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-001 traces: TASK-001
// Test that a valid documentation item passes validation
func TestTraceItem_ValidDocItem(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:        "REQ-001",
		Type:      trace.NodeTypeREQ,
		Project:   "my-project",
		Title:     "A valid requirement",
		Status:    "active",
		TracesTo:  nil, // REQ has no upstream
		Created:   time.Now(),
		Updated:   time.Now(),
		SourceFile:   "requirements.md",
		SourceFormat: "yaml",
	}

	err := item.Validate()
	g.Expect(err).ToNot(HaveOccurred())
}

// Test that a valid ISSUE item passes validation
func TestTraceItem_ValidIssueItem(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:           "ISSUE-001",
		Type:         trace.NodeTypeISSUE,
		Project:      "my-project",
		Title:        "Fix the traceability chain",
		Status:       "active",
		TracesTo:     nil, // ISSUE has no upstream
		Created:      time.Now(),
		Updated:      time.Now(),
		SourceFile:   "issues.md",
		SourceFormat: "yaml",
	}

	err := item.Validate()
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-002 traces: TASK-001
// Test that a valid TEST item passes validation
func TestTraceItem_ValidTestItem(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:       "TEST-042",
		Type:     trace.NodeTypeTEST,
		Project:  "my-project",
		Title:    "Test user validation",
		Status:   "active",
		TracesTo: []string{"TASK-003"},
		Location: "user_test.go",
		Line:     10,
		Function: "TestValidateUser",
		SourceFile:   "user_test.go",
		SourceFormat: "go-ast",
	}

	err := item.Validate()
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-003 traces: TASK-001
// Test that missing required ID field produces validation error
func TestTraceItem_MissingID(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:      "", // Missing
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "A requirement",
		Status:  "active",
	}

	err := item.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ID"))
}

// TEST-004 traces: TASK-001
// Test that missing required Type field produces validation error
func TestTraceItem_MissingType(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:      "REQ-001",
		Type:    "", // Missing/invalid
		Project: "my-project",
		Title:   "A requirement",
		Status:  "active",
	}

	err := item.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Type"))
}

// TEST-005 traces: TASK-001
// Test that missing required Project field produces validation error
func TestTraceItem_MissingProject(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:      "REQ-001",
		Type:    trace.NodeTypeREQ,
		Project: "", // Missing
		Title:   "A requirement",
		Status:  "active",
	}

	err := item.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Project"))
}

// TEST-006 traces: TASK-001
// Test that missing required Title field produces validation error
func TestTraceItem_MissingTitle(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:      "REQ-001",
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "", // Missing
		Status:  "active",
	}

	err := item.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Title"))
}

// TEST-007 traces: TASK-001
// Test that missing required Status field produces validation error
func TestTraceItem_MissingStatus(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:      "REQ-001",
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "A requirement",
		Status:  "", // Missing
	}

	err := item.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Status"))
}

// TEST-008 traces: TASK-001
// Test that invalid ID format produces validation error
func TestTraceItem_InvalidIDFormat(t *testing.T) {
	g := NewWithT(t)

	testCases := []string{
		"INVALID-001",  // Wrong prefix
		"REQ001",       // Missing hyphen
		"REQ-",         // Missing number
		"REQ-1",        // Too few digits (need 3+)
		"REQ-01",       // Too few digits
		"req-001",      // Lowercase prefix
		"TEST-ABC",     // Non-numeric
	}

	for _, id := range testCases {
		item := trace.TraceItem{
			ID:      id,
			Type:    trace.NodeTypeREQ,
			Project: "my-project",
			Title:   "A requirement",
			Status:  "active",
		}

		err := item.Validate()
		g.Expect(err).To(HaveOccurred(), "Expected error for ID: %s", id)
		g.Expect(err.Error()).To(ContainSubstring("ID"), "Error should mention ID for: %s", id)
	}
}

// TEST-009 traces: TASK-001
// Test that TEST node without Location produces validation error
func TestTraceItem_TestNodeMissingLocation(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:       "TEST-001",
		Type:     trace.NodeTypeTEST,
		Project:  "my-project",
		Title:    "Test something",
		Status:   "active",
		TracesTo: []string{"TASK-001"},
		Location: "", // Missing - required for TEST
		Line:     10,
		Function: "TestSomething",
	}

	err := item.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Location"))
}

// TEST-010 traces: TASK-001
// Test that TEST node without Function produces validation error
func TestTraceItem_TestNodeMissingFunction(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:       "TEST-001",
		Type:     trace.NodeTypeTEST,
		Project:  "my-project",
		Title:    "Test something",
		Status:   "active",
		TracesTo: []string{"TASK-001"},
		Location: "foo_test.go",
		Line:     10,
		Function: "", // Missing - required for TEST
	}

	err := item.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Function"))
}

// TEST-011 traces: TASK-001
// Test that ID type must match Type field
func TestTraceItem_IDTypeMismatch(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:      "REQ-001",      // ID says REQ
		Type:    trace.NodeTypeTEST, // Type says TEST
		Project: "my-project",
		Title:   "A test item",
		Status:  "active",
	}

	err := item.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("mismatch"))
}

// TEST-012 traces: TASK-001
// Property test: valid TraceItems never produce validation errors
func TestTraceItem_PropertyValidItemsPass(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		nodeType := rapid.SampledFrom([]trace.NodeType{
			trace.NodeTypeISSUE,
			trace.NodeTypeREQ,
			trace.NodeTypeDES,
			trace.NodeTypeARCH,
			trace.NodeTypeTASK,
		}).Draw(rt, "nodeType")

		// Generate valid ID matching the type
		prefix := string(nodeType)
		num := rapid.IntRange(1, 999).Draw(rt, "num")
		id := rapid.Just(prefix + "-" + padNum(num)).Draw(rt, "id")

		project := rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(rt, "project")
		title := rapid.StringMatching(`[A-Za-z0-9 ]{1,50}`).Draw(rt, "title")
		status := rapid.SampledFrom([]string{"draft", "active", "completed", "deprecated"}).Draw(rt, "status")

		item := trace.TraceItem{
			ID:           id,
			Type:         nodeType,
			Project:      project,
			Title:        title,
			Status:       status,
			Created:      time.Now(),
			Updated:      time.Now(),
			SourceFile:   "test.md",
			SourceFormat: "yaml",
		}

		// Add TracesTo for types that require it
		if nodeType != trace.NodeTypeREQ {
			item.TracesTo = []string{"REQ-001"}
		}

		err := item.Validate()
		g.Expect(err).ToNot(HaveOccurred(), "Valid item should pass: %+v", item)
	})
}

// TEST-013 traces: TASK-001
// Property test: valid TEST items with all required fields pass
func TestTraceItem_PropertyValidTestItemsPass(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		num := rapid.IntRange(1, 999).Draw(rt, "num")
		id := "TEST-" + padNum(num)

		project := rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(rt, "project")
		title := rapid.StringMatching(`[A-Za-z0-9 ]{1,50}`).Draw(rt, "title")
		location := rapid.StringMatching(`[a-z_]+_test\.go`).Draw(rt, "location")
		funcName := rapid.StringMatching(`Test[A-Z][A-Za-z0-9_]{0,30}`).Draw(rt, "funcName")
		line := rapid.IntRange(1, 10000).Draw(rt, "line")

		item := trace.TraceItem{
			ID:           id,
			Type:         trace.NodeTypeTEST,
			Project:      project,
			Title:        title,
			Status:       "active",
			TracesTo:     []string{"TASK-001"},
			Location:     location,
			Line:         line,
			Function:     funcName,
			SourceFile:   location,
			SourceFormat: "go-ast",
		}

		err := item.Validate()
		g.Expect(err).ToNot(HaveOccurred(), "Valid TEST item should pass: %+v", item)
	})
}

// TEST-014 traces: TASK-001
// Property test: missing any required field produces error
func TestTraceItem_PropertyMissingRequiredFieldFails(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		// Start with valid item
		item := trace.TraceItem{
			ID:           "REQ-001",
			Type:         trace.NodeTypeREQ,
			Project:      "my-project",
			Title:        "A requirement",
			Status:       "active",
			Created:      time.Now(),
			Updated:      time.Now(),
			SourceFile:   "test.md",
			SourceFormat: "yaml",
		}

		// Clear one required field
		field := rapid.SampledFrom([]string{"ID", "Type", "Project", "Title", "Status"}).Draw(rt, "field")
		switch field {
		case "ID":
			item.ID = ""
		case "Type":
			item.Type = ""
		case "Project":
			item.Project = ""
		case "Title":
			item.Title = ""
		case "Status":
			item.Status = ""
		}

		err := item.Validate()
		g.Expect(err).To(HaveOccurred(), "Missing %s should produce error", field)
	})
}

// TEST-015 traces: TASK-001
// Test valid status values
func TestTraceItem_ValidStatuses(t *testing.T) {
	g := NewWithT(t)

	validStatuses := []string{"draft", "active", "completed", "deprecated"}

	for _, status := range validStatuses {
		item := trace.TraceItem{
			ID:           "REQ-001",
			Type:         trace.NodeTypeREQ,
			Project:      "my-project",
			Title:        "A requirement",
			Status:       status,
			Created:      time.Now(),
			Updated:      time.Now(),
			SourceFile:   "test.md",
			SourceFormat: "yaml",
		}

		err := item.Validate()
		g.Expect(err).ToNot(HaveOccurred(), "Status %q should be valid", status)
	}
}

// TEST-016 traces: TASK-001
// Test invalid status values
func TestTraceItem_InvalidStatus(t *testing.T) {
	g := NewWithT(t)

	item := trace.TraceItem{
		ID:      "REQ-001",
		Type:    trace.NodeTypeREQ,
		Project: "my-project",
		Title:   "A requirement",
		Status:  "invalid-status",
	}

	err := item.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Status"))
}

// padNum pads a number to 3 digits
func padNum(n int) string {
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
