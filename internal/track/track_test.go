package track_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/track"
)

// T-73: First surfacing — count 0→1, empty contexts→["prompt"]
func TestT73_FirstSurfacingFromZero(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	stored := &memory.Stored{
		SurfacedCount:     0,
		SurfacingContexts: nil,
	}

	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)

	update := track.ComputeUpdate(stored, "prompt", now)

	g.Expect(update.SurfacedCount).To(Equal(1))
	g.Expect(update.LastSurfaced).To(Equal(now))
	g.Expect(update.SurfacingContexts).To(Equal([]string{"prompt"}))
}

// T-74: Append to existing — count 5→6, contexts appended
func TestT74_AppendToExistingContexts(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	stored := &memory.Stored{
		SurfacedCount:     5,
		SurfacingContexts: []string{"prompt", "tool", "session-start"},
	}

	now := time.Date(2026, 3, 5, 14, 0, 0, 0, time.UTC)

	update := track.ComputeUpdate(stored, "tool", now)

	g.Expect(update.SurfacedCount).To(Equal(6))
	g.Expect(update.LastSurfaced).To(Equal(now))
	g.Expect(update.SurfacingContexts).To(Equal(
		[]string{"prompt", "tool", "session-start", "tool"},
	))
}

// T-75: FIFO eviction at MaxContextEntries
func TestT75_FIFOEvictionAtMaxEntries(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Build 10 existing context entries (at max).
	existing := make([]string, 0, track.MaxContextEntries)
	for i := range track.MaxContextEntries {
		existing = append(existing, fmt.Sprintf("ctx-%d", i))
	}

	stored := &memory.Stored{
		SurfacedCount:     10,
		SurfacingContexts: existing,
	}

	now := time.Date(2026, 3, 5, 16, 0, 0, 0, time.UTC)

	update := track.ComputeUpdate(stored, "new-mode", now)

	g.Expect(update.SurfacedCount).To(Equal(11))
	g.Expect(update.LastSurfaced).To(Equal(now))
	g.Expect(update.SurfacingContexts).To(HaveLen(track.MaxContextEntries))
	// Oldest entry ("ctx-0") should be evicted, newest should be "new-mode".
	g.Expect(update.SurfacingContexts[0]).To(Equal("ctx-1"))
	g.Expect(update.SurfacingContexts[track.MaxContextEntries-1]).To(Equal("new-mode"))
}
