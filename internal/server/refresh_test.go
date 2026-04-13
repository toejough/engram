package server_test

import (
	"testing"

	"engram/internal/server"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

func TestRefreshTracker_AlwaysFiresAtInterval(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		interval := rapid.IntRange(2, 20).Draw(rt, "interval")
		interactions := rapid.IntRange(1, 100).Draw(rt, "interactions")

		tracker := server.NewRefreshTracker(interval)
		refreshCount := 0

		for range interactions {
			if tracker.ShouldRefresh() {
				refreshCount++
			}
		}

		expectedRefreshes := interactions / interval
		g.Expect(refreshCount).To(Equal(expectedRefreshes))
	})
}

func TestRefreshTracker_DefaultInterval(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(server.SkillRefreshInterval).To(Equal(13))
}

func TestRefreshTracker_NeverFiresBeforeInterval(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const interval = 5

	tracker := server.NewRefreshTracker(interval)

	for range interval - 1 {
		g.Expect(tracker.ShouldRefresh()).To(BeFalse())
	}

	// The Nth call should fire.
	g.Expect(tracker.ShouldRefresh()).To(BeTrue())
}
