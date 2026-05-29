//go:build targ

package eval

// LookupArm returns the arm config for name and whether it exists.
func LookupArm(name string) (Arm, bool) {
	arm, ok := m1Arms[name]
	return arm, ok
}

// unexported variables.
var (
	// m1Arms is the Milestone 1 arm set. skills-only and current-state share
	// the same skill bundle; they differ only in binary availability (binary
	// absent → recall falls back to its degraded direct-read mode).
	m1Arms = map[string]Arm{
		"nothing":       {Name: "nothing", Skills: nil, BinaryOnPATH: false},
		"skills-only":   {Name: "skills-only", Skills: []string{"recall", "learn"}, BinaryOnPATH: false},
		"current-state": {Name: "current-state", Skills: []string{"recall", "learn"}, BinaryOnPATH: true},
	}
)
