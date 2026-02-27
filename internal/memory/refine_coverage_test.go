package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestExtractPrinciple_NoMarkers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := "simple rule without any markers"
	result := extractPrinciple(entry)

	g.Expect(result).To(Equal(entry))
}

func TestExtractPrinciple_WithBackticks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := "Always use `go test` to run tests"
	result := extractPrinciple(entry)

	g.Expect(result).ToNot(ContainSubstring("`go test`"))
	g.Expect(result).To(ContainSubstring("Always use"))
	g.Expect(result).To(ContainSubstring("to run tests"))
}

func TestExtractPrinciple_WithEgMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := "Use snake_case for variables. E.g., my_var instead of myVar"
	result := extractPrinciple(entry)

	g.Expect(result).To(Equal("Use snake_case for variables"))
}

// ─── extractPrinciple tests ───────────────────────────────────────────────────

func TestExtractPrinciple_WithExampleMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := "Always run tests. Example: go test ./..."
	result := extractPrinciple(entry)

	g.Expect(result).To(Equal("Always run tests"))
}

func TestExtractPrinciple_WithForExampleMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := "Prefer short names. For example, use i instead of index"
	result := extractPrinciple(entry)

	g.Expect(result).To(Equal("Prefer short names"))
}

func TestExtractPrinciple_WithUsageMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := "Run targ to build. Usage: targ build"
	result := extractPrinciple(entry)

	g.Expect(result).To(Equal("Run targ to build"))
}

func TestHasCodeBlockOrPath_Clean(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasCodeBlockOrPath("always run tests before committing")).To(BeFalse())
}

// ─── hasCodeBlockOrPath tests ─────────────────────────────────────────────────

func TestHasCodeBlockOrPath_WithBacktick(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasCodeBlockOrPath("Use `go test` command")).To(BeTrue())
}

func TestHasCodeBlockOrPath_WithForwardSlash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasCodeBlockOrPath("See /path/to/file")).To(BeTrue())
}

func TestHasCodeBlockOrPath_WithGoExtension(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasCodeBlockOrPath("Edit the file.go to fix this")).To(BeTrue())
}

func TestHasCodeBlockOrPath_WithJsonExtension(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasCodeBlockOrPath("Read config.json first")).To(BeTrue())
}

func TestHasCodeBlockOrPath_WithPyExtension(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasCodeBlockOrPath("Run script.py for setup")).To(BeTrue())
}

func TestIsImperativeWithoutRationale_AvoidNoExplanation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isImperativeWithoutRationale("Avoid global state")).To(BeTrue())
}

func TestIsImperativeWithoutRationale_EnsureNoExplanation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isImperativeWithoutRationale("Ensure coverage is above 80%")).To(BeTrue())
}

// ─── isImperativeWithoutRationale tests ───────────────────────────────────────

func TestIsImperativeWithoutRationale_ImperativeNoExplanation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isImperativeWithoutRationale("Always check the output")).To(BeTrue())
}

func TestIsImperativeWithoutRationale_ImperativeWithBecause(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isImperativeWithoutRationale("Always run tests because failures matter")).To(BeFalse())
}

func TestIsImperativeWithoutRationale_ImperativeWithTo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isImperativeWithoutRationale("Use short names to improve readability")).To(BeFalse())
}

func TestIsImperativeWithoutRationale_NeverNoExplanation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isImperativeWithoutRationale("Never skip tests")).To(BeTrue())
}

func TestIsImperativeWithoutRationale_NotImperative(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isImperativeWithoutRationale("Tests should be run frequently")).To(BeFalse())
}

func TestIsImperativeWithoutRationale_UseNoExplanation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// "to" is an explanation marker, so use a phrase without it
	g.Expect(isImperativeWithoutRationale("Use targ build system")).To(BeTrue())
}
