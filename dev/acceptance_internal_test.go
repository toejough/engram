//go:build targ

package dev

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestAcceptanceTestArgs_TagsPackageAndFilter pins what targ acceptance runs: the tests of the acceptance package
// with the network tag, which no default target selects, at most four at once (793b49), and only the tests the filter
// names when it is given.
func TestAcceptanceTestArgs_TagsPackageAndFilter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(acceptanceTestArgs(acceptanceArgs{})).To(Equal(
		[]string{"test", "-count=1", "-timeout=30m", "-v", "-parallel=4", "-tags=network", "./dev/acceptance/..."}))
	g.Expect(acceptanceTestArgs(acceptanceArgs{Run: "Test793a2a1a"})).To(Equal(
		[]string{"test", "-count=1", "-timeout=30m", "-v", "-parallel=4", "-tags=network", "-run", "Test793a2a1a",
			"./dev/acceptance/..."}))
}
