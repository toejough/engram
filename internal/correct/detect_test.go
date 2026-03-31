package correct_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/correct"
)

func TestDetectFastPath_MatchesKeywords(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	keywords := []string{"remember", "always", "never", "don't", "stop"}

	for _, keyword := range keywords {
		keyword := keyword

		t.Run(keyword, func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)

			result := correct.DetectFastPath("please "+keyword+" this", keywords)

			g.Expect(result).To(BeTrue())
		})
	}

	_ = g // used for outer test; subtests have their own
}

func TestDetectFastPath_CaseInsensitive(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	keywords := []string{"remember"}

	g.Expect(correct.DetectFastPath("REMEMBER this", keywords)).To(BeTrue())
	g.Expect(correct.DetectFastPath("Remember this", keywords)).To(BeTrue())
	g.Expect(correct.DetectFastPath("remember this", keywords)).To(BeTrue())
}

func TestDetectFastPath_NoMatch(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	keywords := []string{"remember", "always", "never", "don't", "stop"}

	result := correct.DetectFastPath("run the tests", keywords)

	g.Expect(result).To(BeFalse())
}

func TestDetectFastPath_EmptyKeywords(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	result := correct.DetectFastPath("remember this", []string{})

	g.Expect(result).To(BeFalse())
}

func TestDetectHaiku_Correction(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "CORRECTION", nil
	}

	result, err := correct.DetectHaiku(context.Background(), mockCaller, "always use tabs", "classify prompt")

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(result).To(BeTrue())
}

func TestDetectHaiku_NotCorrection(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "NOT_CORRECTION", nil
	}

	result, err := correct.DetectHaiku(context.Background(), mockCaller, "run the tests", "classify prompt")

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(result).To(BeFalse())
}

func TestDetectHaiku_ErrorPropagation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	callerErr := errors.New("caller failed")

	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "", callerErr
	}

	_, err := correct.DetectHaiku(context.Background(), mockCaller, "always use tabs", "classify prompt")

	g.Expect(err).To(MatchError(callerErr))
}
