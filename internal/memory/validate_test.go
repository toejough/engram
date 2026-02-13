package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// ISSUE-215: Actionability validation tests
// ============================================================================

// --- Reject garbage content ---

func TestValidateActionability_RejectsShortStrings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := memory.ValidateActionability("short")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("too short"))
}

func TestValidateActionability_RejectsVaguePhrase_ImportantPattern(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := memory.ValidateActionability("important pattern for review")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("vague"))
}

func TestValidateActionability_RejectsVaguePhrase_LearningNumber(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := memory.ValidateActionability("learning number A about testing patterns")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("vague"))
}

func TestValidateActionability_RejectsVaguePhrase_UsefulReminder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := memory.ValidateActionability("useful reminder to keep in mind")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("vague"))
}

func TestValidateActionability_RejectsVaguePhrase_GoodToKnow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := memory.ValidateActionability("good to know for future reference")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("vague"))
}

func TestValidateActionability_RejectsNonImperative(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Past tense - not imperative
	err := memory.ValidateActionability("We fixed the bug by using dependency injection yesterday")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("imperative"))
}

// --- Accept good content ---

func TestValidateActionability_AcceptsGoodContent_AlwaysUse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := memory.ValidateActionability("Always use dependency injection for IO")
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidateActionability_AcceptsGoodContent_Never(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := memory.ValidateActionability("Never amend pushed commits because it rewrites history")
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidateActionability_AcceptsGoodContent_Prefer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := memory.ValidateActionability("Prefer property-based tests over example-based tests")
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidateActionability_AcceptsGoodContent_Avoid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := memory.ValidateActionability("Avoid global state in production code for testability")
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidateActionability_AcceptsGoodContent_Use(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := memory.ValidateActionability("Use gomega matchers for readable test assertions")
	g.Expect(err).ToNot(HaveOccurred())
}

// --- Property tests ---

func TestPropertyValidateActionability_ShortStringsAlwaysFail(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		// Generate string shorter than 20 chars
		shortStr := rapid.StringMatching(`^.{1,19}$`).Draw(rt, "shortString")

		err := memory.ValidateActionability(shortStr)
		g.Expect(err).To(HaveOccurred())
	})
}

func TestPropertyValidateActionability_ImperativeVerbsPass(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		// Generate principle with imperative verb
		verb := rapid.SampledFrom([]string{"Always", "Never", "Prefer", "Avoid", "Use", "Ensure"}).Draw(rt, "verb")
		action := rapid.StringMatching(`[a-z ]{10,30}`).Draw(rt, "action")
		principle := verb + " " + action

		// Should not reject for imperative structure (may fail on length, but not imperative check)
		err := memory.ValidateActionability(principle)
		if err != nil {
			g.Expect(err.Error()).ToNot(ContainSubstring("imperative"))
		}
	})
}
