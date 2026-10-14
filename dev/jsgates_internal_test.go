//go:build targ

package dev

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

func Test793e8l_JSGatePlan(t *testing.T) {
	t.Parallel()

	present := func(name string) (string, error) { return "/bin/" + name, nil }
	missing := func(missingName string) func(string) (string, error) {
		return func(name string) (string, error) {
			if name == missingName {
				return "", errors.New("not found")
			}

			return "/bin/" + name, nil
		}
	}
	installed := func(string) bool { return true }
	notInstalled := func(string) bool { return false }

	t.Run("a gate is one node command with its arguments", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		env := jsGateEnv{Root: "/repo", LookPath: present, Exists: installed}
		plan, err := jsGatePlan("lint", []string{"complexity"}, false, env)

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(plan).To(Equal([]acceptanceCommand{
			{Name: "node", Args: []string{"/repo/dev/acceptance/gates/run.js", "lint", "complexity"}},
		}))
	})

	t.Run("missing dependencies are installed from the lockfile first", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		plan, err := jsGatePlan("budget", nil, false, jsGateEnv{Root: "/repo", LookPath: present, Exists: notInstalled})

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(plan).To(HaveLen(2))
		g.Expect(plan[0]).To(Equal(acceptanceCommand{Name: "npm", Args: []string{"ci", "--prefix", "/repo/dev/acceptance"}}))
	})

	t.Run("claude is needed only by the gates that run the mod's tests", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		_, err := jsGatePlan("budget", nil, false, jsGateEnv{Root: "/repo", LookPath: missing("claude"), Exists: installed})
		g.Expect(err).NotTo(HaveOccurred())

		noClaude := jsGateEnv{Root: "/repo", LookPath: missing("claude"), Exists: installed}
		_, err = jsGatePlan("mod-test", []string{"red", ""}, true, noClaude)
		g.Expect(err).To(MatchError(errAcceptanceProgram))
		g.Expect(err).To(MatchError(ContainSubstring("`claude`")))
	})

	t.Run("a missing node or npm fails and names it", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"node", "npm"} {
			g := NewWithT(t)

			_, err := jsGatePlan("lint", nil, false, jsGateEnv{Root: "/repo", LookPath: missing(name), Exists: installed})

			g.Expect(err).To(MatchError(errAcceptanceProgram))
			g.Expect(err).To(MatchError(ContainSubstring("`" + name + "`")))
		}
	})
}
