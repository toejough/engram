package maintain_test

import (
	"os"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/maintain"
)

// TestPurgeTierC_Stubbed verifies stubbed PurgeTierC returns 0.
func TestPurgeTierC_Stubbed(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	deleted, err := maintain.PurgeTierC(t.TempDir(), os.Remove)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(deleted).To(gomega.Equal(0))
}
