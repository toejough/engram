//go:build targ

package dev

// unverifiedScenarios returns the scenarios held out of the conformance gate, sorted by ID. Each reason names its
// decision row or issue. An entry is removed when its scenario gets a test or leaves the specs: a stale entry fails the
// gate.
func unverifiedScenarios() []allowlistEntry {
	return []allowlistEntry{
		{id: "792a19k1c", reason: "#870, 861d23 (change 861, promoted): needs a live session"},
		{id: "792a2d1b", reason: "#796, 861d27 (change 861, promoted): removed by change 796 (796a3)"},
		{id: "792a2e1c", reason: "#795, 861d27 (change 861, promoted): removed by change 795 (795a15)"},
		{id: "792a2e1d", reason: "#795, 861d27 (change 861, promoted): removed by change 795 (795a16)"},
		{id: "792a2g1a", reason: "#795, 861d27 (change 861, promoted): removed by change 795 (795a17)"},
		{
			id:     "792b2c3a2",
			reason: "#795, 861d27 (change 861, promoted): change 795 (795b12) defers the technical spec follow-up",
		},
		{
			id:     "792b3c4a4",
			reason: "#796, 861d27 (change 861, promoted): change 796 (796b6) follows the technical spec",
		},
	}
}
