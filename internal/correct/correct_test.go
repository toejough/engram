package correct_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/correct"
)

// TestRun_Stubbed verifies that the stubbed Run returns empty string.
func TestRun_Stubbed(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	corrector := correct.New()
	result, err := corrector.Run(context.Background(), "message", "context")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}
