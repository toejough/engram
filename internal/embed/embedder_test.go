package embed_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestState_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		state embed.State
		want  string
	}{
		{embed.StateOK, "ok"},
		{embed.StateMissing, "missing"},
		{embed.StateStale, "stale"},
		{embed.StateIncompatible, "incompatible"},
		{embed.StateBroken, "broken"},
		{embed.State(99), "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			g.Expect(tc.state.String()).To(Equal(tc.want))
		})
	}
}
