package parser_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/parser"
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
