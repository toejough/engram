package memory

import "time"

// ViolationTrend tracks the trend of a specific hook violation rule over time.
type ViolationTrend struct {
	Rule               string
	TotalViolations    int
	ViolationsByPeriod map[string]int
	Trending           string
	Recommendation     string
}

// ComputeViolationTrends analyzes violation patterns over time to detect trends.
// periodDays specifies the time window to compare (e.g., 7 for weekly periods).
// Returns a map of rule name to trend analysis.
func ComputeViolationTrends(violations []ChangelogEntry, periodDays int) map[string]ViolationTrend {
	if len(violations) == 0 {
		return make(map[string]ViolationTrend)
	}

	// Group violations by rule
	ruleViolations := make(map[string][]ChangelogEntry)
	for _, v := range violations {
		if v.Action != "hook_violation" {
			continue
		}
		rule := v.Metadata["rule"]
		if rule == "" {
			continue
		}
		ruleViolations[rule] = append(ruleViolations[rule], v)
	}

	// Compute trends for each rule
	trends := make(map[string]ViolationTrend)
	now := violations[len(violations)-1].Timestamp // Use latest violation as "now"
	if len(violations) > 0 {
		for _, v := range violations {
			if v.Timestamp.After(now) {
				now = v.Timestamp
			}
		}
	}

	for rule, ruleVios := range ruleViolations {
		trend := computeTrendForRule(rule, ruleVios, periodDays, now)
		trends[rule] = trend
	}

	return trends
}

func computeTrendForRule(rule string, violations []ChangelogEntry, periodDays int, now time.Time) ViolationTrend {
	// Count violations in recent period vs older period
	recentCount := 0
	olderCount := 0
	recentStart := now.Add(-time.Duration(periodDays) * 24 * time.Hour)
	olderStart := now.Add(-2 * time.Duration(periodDays) * 24 * time.Hour)

	for _, v := range violations {
		if v.Timestamp.After(recentStart) {
			recentCount++
		} else if v.Timestamp.After(olderStart) {
			olderCount++
		}
	}

	// Determine trend
	var trending string
	var recommendation string

	// Single violation is always stable (insufficient data for trend)
	if len(violations) == 1 {
		trending = "stable"
		recommendation = "Insufficient data for trend analysis"
	} else if recentCount < olderCount {
		trending = "improving"
		recommendation = "Learning is working - violations declining"
	} else if recentCount > olderCount {
		trending = "degrading"
		recommendation = "Learning not effective - violations increasing"
	} else {
		trending = "stable"
		recommendation = "Violations stable - may need stronger reinforcement"
	}

	// Build violations by period map
	violationsByPeriod := make(map[string]int)
	violationsByPeriod["recent"] = recentCount
	violationsByPeriod["older"] = olderCount

	return ViolationTrend{
		Rule:               rule,
		TotalViolations:    len(violations),
		ViolationsByPeriod: violationsByPeriod,
		Trending:           trending,
		Recommendation:     recommendation,
	}
}
