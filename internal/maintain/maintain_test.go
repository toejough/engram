package maintain_test

import (
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/maintain"
)

// TestNew_ReturnsNonNilGenerator verifies Generator creation works during SBIA stub.
func TestNew_ReturnsNonNilGenerator(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	gen := maintain.New()
	g.Expect(gen).NotTo(gomega.BeNil())
}
