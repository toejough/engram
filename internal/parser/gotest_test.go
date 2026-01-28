package parser_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/parser"
	"github.com/toejough/projctl/internal/trace"
)

// TEST-087 traces: TASK-012
// Test parsing file with single test function
func TestParseTestFunctions_SingleTest(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

func TestSomething(t *testing.T) {
	// test body
}
`

	funcs, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(funcs).To(HaveLen(1))
	g.Expect(funcs[0].Name).To(Equal("TestSomething"))
	g.Expect(funcs[0].File).To(Equal("foo_test.go"))
	g.Expect(funcs[0].Line).To(BeNumerically(">", 0))
}

// TEST-088 traces: TASK-012
// Test parsing file with multiple test functions
func TestParseTestFunctions_MultipleTests(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

func TestFirst(t *testing.T) {}
func TestSecond(t *testing.T) {}
func TestThird(t *testing.T) {}
`

	funcs, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(funcs).To(HaveLen(3))
	g.Expect(funcs[0].Name).To(Equal("TestFirst"))
	g.Expect(funcs[1].Name).To(Equal("TestSecond"))
	g.Expect(funcs[2].Name).To(Equal("TestThird"))
}

// TEST-089 traces: TASK-012
// Test parsing file with Benchmark function
func TestParseTestFunctions_BenchmarkFunction(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

func BenchmarkSomething(b *testing.B) {}
`

	funcs, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(funcs).To(HaveLen(1))
	g.Expect(funcs[0].Name).To(Equal("BenchmarkSomething"))
}

// TEST-090 traces: TASK-012
// Test parsing ignores non-test functions
func TestParseTestFunctions_IgnoresNonTest(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

func helper() {}
func TestActual(t *testing.T) {}
func anotherHelper() {}
`

	funcs, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(funcs).To(HaveLen(1))
	g.Expect(funcs[0].Name).To(Equal("TestActual"))
}

// TEST-091 traces: TASK-012
// Test parsing empty file returns no functions
func TestParseTestFunctions_EmptyFile(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test
`

	funcs, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(funcs).To(BeEmpty())
}

// TEST-092 traces: TASK-012
// Test parsing invalid Go syntax returns error
func TestParseTestFunctions_InvalidSyntax(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

func invalid syntax here {
`

	_, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).To(HaveOccurred())
}

// TEST-093 traces: TASK-012
// Test line numbers are correct
func TestParseTestFunctions_LineNumbers(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

func TestFirst(t *testing.T) {}

func TestSecond(t *testing.T) {}
`

	funcs, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(funcs).To(HaveLen(2))
	// TestFirst is on line 5, TestSecond is on line 7
	g.Expect(funcs[0].Line).To(Equal(5))
	g.Expect(funcs[1].Line).To(Equal(7))
}

// TEST-094 traces: TASK-012
// Property test: N test functions yields N results
func TestParseTestFunctions_PropertyCount(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		testCount := rapid.IntRange(0, 10).Draw(rt, "testCount")

		src := "package foo_test\n\nimport \"testing\"\n\n"
		for i := 0; i < testCount; i++ {
			src += "func Test" + padNum(i) + "(t *testing.T) {}\n"
		}

		funcs, err := parser.ParseTestFunctions("foo_test.go", src)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(funcs).To(HaveLen(testCount))
	})
}

// padNum returns a 3-digit padded number string for test generation
func padNum(n int) string {
	if n < 10 {
		return "00" + itoa(n)
	}
	if n < 100 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
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

// TEST-095 traces: TASK-013
// Test extracting trace comment from function with doc comment
func TestExtractTraceComment_WithComment(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

// TEST-001 traces: TASK-001
func TestSomething(t *testing.T) {}
`

	funcs, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(funcs).To(HaveLen(1))
	g.Expect(funcs[0].Comment).To(Equal("// TEST-001 traces: TASK-001"))
}

// TEST-096 traces: TASK-013
// Test extracting comment when no trace comment exists
func TestExtractTraceComment_NoComment(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

func TestSomething(t *testing.T) {}
`

	funcs, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(funcs).To(HaveLen(1))
	g.Expect(funcs[0].Comment).To(BeEmpty())
}

// TEST-097 traces: TASK-013
// Test extracting comment with multiple doc comment lines
func TestExtractTraceComment_MultipleLines(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

// This is a description of the test.
// It has multiple lines.
// TEST-042 traces: TASK-005, ARCH-001
func TestSomething(t *testing.T) {}
`

	funcs, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(funcs).To(HaveLen(1))
	// Should extract the trace comment line
	g.Expect(funcs[0].Comment).To(ContainSubstring("TEST-042 traces:"))
}

// TEST-098 traces: TASK-013
// Test extracting comment with blank line gap (should not extract)
func TestExtractTraceComment_BlankLineGap(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

// TEST-001 traces: TASK-001

func TestSomething(t *testing.T) {}
`

	funcs, err := parser.ParseTestFunctions("foo_test.go", src)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(funcs).To(HaveLen(1))
	// Blank line breaks association - should have no comment
	g.Expect(funcs[0].Comment).To(BeEmpty())
}

// TEST-099 traces: TASK-014
// Test parsing valid trace comment
func TestParseTraceComment_Valid(t *testing.T) {
	g := NewWithT(t)

	comment := "// TEST-001 traces: TASK-001"
	result, err := parser.ParseTraceComment(comment)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.TestID).To(Equal("TEST-001"))
	g.Expect(result.Targets).To(Equal([]string{"TASK-001"}))
}

// TEST-100 traces: TASK-014
// Test parsing trace comment with multiple targets
func TestParseTraceComment_MultipleTargets(t *testing.T) {
	g := NewWithT(t)

	comment := "// TEST-042 traces: TASK-001, ARCH-005, REQ-010"
	result, err := parser.ParseTraceComment(comment)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.TestID).To(Equal("TEST-042"))
	g.Expect(result.Targets).To(Equal([]string{"TASK-001", "ARCH-005", "REQ-010"}))
}

// TEST-101 traces: TASK-014
// Test parsing trace comment with uppercase Traces
func TestParseTraceComment_UppercaseTraces(t *testing.T) {
	g := NewWithT(t)

	comment := "// TEST-001 Traces: TASK-001"
	result, err := parser.ParseTraceComment(comment)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.TestID).To(Equal("TEST-001"))
	g.Expect(result.Targets).To(Equal([]string{"TASK-001"}))
}

// TEST-102 traces: TASK-014
// Test parsing trace comment with flexible whitespace
func TestParseTraceComment_FlexibleWhitespace(t *testing.T) {
	g := NewWithT(t)

	comment := "// TEST-001   traces:   TASK-001,   TASK-002"
	result, err := parser.ParseTraceComment(comment)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Targets).To(Equal([]string{"TASK-001", "TASK-002"}))
}

// TEST-103 traces: TASK-014
// Test parsing trace comment with lowercase target IDs (should uppercase)
func TestParseTraceComment_UppercasesTargets(t *testing.T) {
	g := NewWithT(t)

	comment := "// TEST-001 traces: task-001, arch-005"
	result, err := parser.ParseTraceComment(comment)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Targets).To(Equal([]string{"TASK-001", "ARCH-005"}))
}

// TEST-104 traces: TASK-014
// Test parsing malformed trace comment returns error
func TestParseTraceComment_Malformed(t *testing.T) {
	g := NewWithT(t)

	comment := "// TEST-001 invalid"
	_, err := parser.ParseTraceComment(comment)
	g.Expect(err).To(HaveOccurred())
}

// TEST-105 traces: TASK-014
// Test parsing empty string returns error
func TestParseTraceComment_Empty(t *testing.T) {
	g := NewWithT(t)

	_, err := parser.ParseTraceComment("")
	g.Expect(err).To(HaveOccurred())
}

// TEST-106 traces: TASK-014
// Property test: valid format always parses
func TestParseTraceComment_PropertyValid(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		testNum := rapid.IntRange(1, 999).Draw(rt, "testNum")
		testID := "TEST-" + padNum(testNum)

		targetCount := rapid.IntRange(1, 5).Draw(rt, "targetCount")
		targetTypes := []string{"TASK", "REQ", "ARCH", "DES"}
		var targets []string
		for i := 0; i < targetCount; i++ {
			targetType := rapid.SampledFrom(targetTypes).Draw(rt, "targetType")
			targetNum := rapid.IntRange(1, 999).Draw(rt, "targetNum")
			targets = append(targets, targetType+"-"+padNum(targetNum))
		}

		comment := "// " + testID + " traces: " + strings.Join(targets, ", ")
		result, err := parser.ParseTraceComment(comment)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TestID).To(Equal(testID))
		g.Expect(result.Targets).To(HaveLen(targetCount))
	})
}

// TEST-107 traces: TASK-015
// Test parsing complete test file with traced tests
func TestParseGoTestFile_TracedTests(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

// TEST-001 traces: TASK-001
func TestFirst(t *testing.T) {}

// TEST-002 traces: TASK-001, ARCH-005
func TestSecond(t *testing.T) {}
`

	result, err := parser.ParseGoTestFile("foo_test.go", src, "test-project")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(2))
	g.Expect(result.Items[0].ID).To(Equal("TEST-001"))
	g.Expect(result.Items[0].TracesTo).To(Equal([]string{"TASK-001"}))
	g.Expect(result.Items[1].ID).To(Equal("TEST-002"))
	g.Expect(result.Items[1].TracesTo).To(ConsistOf("TASK-001", "ARCH-005"))
}

// TEST-108 traces: TASK-015
// Test parsing file with no trace comments
func TestParseGoTestFile_NoTraceComments(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

func TestSomething(t *testing.T) {}
func TestOther(t *testing.T) {}
`

	result, err := parser.ParseGoTestFile("foo_test.go", src, "test-project")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(BeEmpty())
	g.Expect(result.Warnings).To(BeEmpty())
}

// TEST-109 traces: TASK-015
// Test parsing file with duplicate TEST IDs returns error
func TestParseGoTestFile_DuplicateTestID(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

// TEST-001 traces: TASK-001
func TestFirst(t *testing.T) {}

// TEST-001 traces: TASK-002
func TestSecond(t *testing.T) {}
`

	_, err := parser.ParseGoTestFile("foo_test.go", src, "test-project")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("duplicate"))
}

// TEST-110 traces: TASK-015
// Test parsing file with malformed comment continues
func TestParseGoTestFile_MalformedComment(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

// TEST-001 traces: TASK-001
func TestFirst(t *testing.T) {}

// Malformed trace comment
func TestBad(t *testing.T) {}

// TEST-002 traces: TASK-002
func TestSecond(t *testing.T) {}
`

	result, err := parser.ParseGoTestFile("foo_test.go", src, "test-project")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(2))
	g.Expect(result.Items[0].ID).To(Equal("TEST-001"))
	g.Expect(result.Items[1].ID).To(Equal("TEST-002"))
}

// TEST-111 traces: TASK-015
// Test TraceItem fields are populated correctly
func TestParseGoTestFile_ItemFields(t *testing.T) {
	g := NewWithT(t)

	src := `package foo_test

import "testing"

// TEST-042 traces: TASK-001
func TestSomething(t *testing.T) {}
`

	result, err := parser.ParseGoTestFile("foo_test.go", src, "test-project")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(1))

	item := result.Items[0]
	g.Expect(item.ID).To(Equal("TEST-042"))
	g.Expect(item.Type).To(Equal(trace.NodeTypeTEST))
	g.Expect(item.Project).To(Equal("test-project"))
	g.Expect(item.Location).To(Equal("foo_test.go"))
	g.Expect(item.Line).To(BeNumerically(">", 0))
	g.Expect(item.Function).To(Equal("TestSomething"))
	g.Expect(item.Status).To(Equal("active"))
}
