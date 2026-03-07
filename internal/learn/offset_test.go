package learn_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/learn"
)

func TestLearnOffset_ZeroValue(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var offset learn.Offset

	g.Expect(offset.Offset).To(Equal(int64(0)))
	g.Expect(offset.SessionID).To(BeEmpty())
}
