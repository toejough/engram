package track_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/track"
)

// T-73: First surfacing — count 0→1, LastSurfaced set to now.
func TestT73_FirstSurfacingFromZero(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)

	update := track.ComputeUpdate(0, now)

	g.Expect(update.SurfacedCount).To(Equal(1))
	g.Expect(update.LastSurfaced).To(Equal(now))
}

// T-74: Existing count incremented — count 5→6, LastSurfaced updated.
func TestT74_ExistingCountIncremented(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 5, 14, 0, 0, 0, time.UTC)

	update := track.ComputeUpdate(5, now)

	g.Expect(update.SurfacedCount).To(Equal(6))
	g.Expect(update.LastSurfaced).To(Equal(now))
}
