package server

// Exported constants.
const (
	SkillRefreshInterval = 13
)

// RefreshTracker tracks interaction count and signals when to refresh skills.
type RefreshTracker struct {
	interval int
	count    int
}

// NewRefreshTracker creates a tracker that fires every interval interactions.
func NewRefreshTracker(interval int) *RefreshTracker {
	return &RefreshTracker{interval: interval}
}

// ShouldRefresh increments the counter and returns true every N interactions.
func (rt *RefreshTracker) ShouldRefresh() bool {
	rt.count++

	return rt.count%rt.interval == 0
}
