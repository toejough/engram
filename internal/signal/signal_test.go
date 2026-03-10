// Package signal_test tests signal constants.
package signal_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/signal"
)

// T-P1-9: KindContradiction constant value is "contradiction".
func TestKindContradiction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(signal.KindContradiction).To(Equal("contradiction"))
}
