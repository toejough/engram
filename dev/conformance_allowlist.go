//go:build targ

package dev

// unverifiedScenarios returns the scenarios held out of the conformance gate, sorted by ID. Each reason names its
// decision row or issue. An entry is removed when its scenario gets a test or leaves the specs: a stale entry fails the
// gate.
func unverifiedScenarios() []allowlistEntry {
	return []allowlistEntry{
		{id: "792a11a1f", reason: "#831, 861d69 (change 861, promoted): Updater.Run runs no go env"},
		{
			id: "792a13a1b",
			reason: "#871, 861d402 (change 861, promoted): " +
				"the skill does not say the worker keeps the parent's content unreworded",
		},
		{
			id: "792a13b1b",
			reason: "#871, 861d403 (change 861, promoted): " +
				"no shipped instruction says a session with only the runbook note follows it",
		},
		{
			id: "792a13c1a",
			reason: "#871, 861d404 (change 861, promoted): " +
				"the skill does not say no command runs while a field is missing",
		},
		{
			id: "792a15b1a",
			reason: "#871, 861d382 (change 861, promoted): " +
				"the sweep report of the learn skill describes output the binary never prints",
		},
		{
			id: "792a16c1b",
			reason: "#871, 861d413 (change 861, promoted): " +
				"the answer to the follow-up is not said to count like an on-time line",
		},
		{
			id:     "792a17f1b",
			reason: "#871, 861d393 (change 861, promoted): the thresholds of the route cold start are not stated",
		},
		{
			id: "792a17k1c",
			reason: "#871, 861d397 (change 861, promoted): " +
				"the binary validates the evidence tag format, the skill does not state it",
		},
		{id: "792a19k1c", reason: "#870, 861d23 (change 861, promoted): needs a live session"},
		{id: "792a2d1b", reason: "#796, 861d27 (change 861, promoted): removed by change 796 (796a3)"},
		{id: "792a2e1c", reason: "#795, 861d27 (change 861, promoted): removed by change 795 (795a15)"},
		{id: "792a2e1d", reason: "#795, 861d27 (change 861, promoted): removed by change 795 (795a16)"},
		{id: "792a2g1a", reason: "#795, 861d27 (change 861, promoted): removed by change 795 (795a17)"},
		{id: "792b28c1a4", reason: "#831, 861d53 (change 861, promoted): Updater.Run runs no go env"},
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
