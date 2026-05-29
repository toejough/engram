//go:build targ

package eval

// DetectBehaviors runs each of the scenario's checks against the agent's
// Bash command stream, returning one outcome per check. Occurred=true
// means the (undesirable) behavior was detected.
func DetectBehaviors(s Scenario, cmds []string) []BehaviorOutcome {
	out := make([]BehaviorOutcome, 0, len(s.Checks))
	for _, c := range s.Checks {
		occurred := false
		for _, cmd := range cmds {
			if c.Pattern.MatchString(cmd) {
				occurred = true
				break
			}
		}
		out = append(out, BehaviorOutcome{Name: c.Name, Kind: c.Kind, Occurred: occurred})
	}
	return out
}
