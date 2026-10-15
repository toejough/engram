//go:build targ

package dev

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

func Test793e8a_AcceptancePlan(t *testing.T) {
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

	t.Run("a plain run is one node command in the package directory", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		plan, err := acceptancePlan(acceptanceArgs{}, acceptanceEnv{Root: "/repo", LookPath: present, Exists: installed})

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(plan).To(Equal([]acceptanceCommand{{Name: "node", Args: []string{"/repo/dev/acceptance/run.js"}}}))
	})

	t.Run("the filter and the dry-run flag reach run.js", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		plan, err := acceptancePlan(
			acceptanceArgs{Filter: "793a5k1e", DryRun: true},
			acceptanceEnv{Root: "/repo", LookPath: missing("claude"), Exists: installed},
		)

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(plan).To(Equal([]acceptanceCommand{
			{Name: "node", Args: []string{"/repo/dev/acceptance/run.js", "--dry-run", "793a5k1e"}},
		}))
	})

	t.Run("a number of runs reaches run.js, and one run adds nothing", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		env := acceptanceEnv{Root: "/repo", LookPath: present, Exists: installed}

		many, err := acceptancePlan(acceptanceArgs{Filter: "793a2n1a", Runs: 10}, env)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(many).To(Equal([]acceptanceCommand{
			{Name: "node", Args: []string{"/repo/dev/acceptance/run.js", "--runs", "10", "793a2n1a"}},
		}))

		one, err := acceptancePlan(acceptanceArgs{Runs: 1}, env)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(one).To(Equal([]acceptanceCommand{{Name: "node", Args: []string{"/repo/dev/acceptance/run.js"}}}))
	})

	t.Run("a number of runs below one, or with a check or a dry run, is refused", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		for _, args := range []acceptanceArgs{{Runs: -1}, {Runs: 5, Check: true}, {Runs: 5, DryRun: true}} {
			_, err := acceptancePlan(args, acceptanceEnv{Root: "/repo", LookPath: present, Exists: installed})

			g.Expect(err).To(HaveOccurred(), "%+v", args)
		}
	})

	t.Run("missing dependencies are installed from the lockfile first", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		plan, err := acceptancePlan(acceptanceArgs{}, acceptanceEnv{Root: "/repo", LookPath: present, Exists: notInstalled})

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(plan).To(HaveLen(2))
		g.Expect(plan[0]).To(Equal(acceptanceCommand{Name: "npm", Args: []string{"ci", "--prefix", "/repo/dev/acceptance"}}))
	})

	t.Run("check runs the self-tests and the vocabulary check, never a session", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		plan, err := acceptancePlan(
			acceptanceArgs{Check: true},
			acceptanceEnv{Root: "/repo", LookPath: missing("claude"), Exists: installed},
		)

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(plan).To(Equal([]acceptanceCommand{
			{Name: "node", Args: []string{"--test", "/repo/dev/acceptance/test/self/"}},
			{Name: "node", Args: []string{"/repo/dev/acceptance/run.js", "--dry-run"}},
		}))
	})

	t.Run("a missing program fails and names it", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"node", "npm", "claude"} {
			g := NewWithT(t)

			_, err := acceptancePlan(acceptanceArgs{}, acceptanceEnv{Root: "/repo", LookPath: missing(name), Exists: installed})

			g.Expect(err).To(MatchError(errAcceptanceProgram))
			g.Expect(err).To(MatchError(ContainSubstring("`" + name + "`")))
		}
	})

	t.Run("check cannot be combined with a filter or the dry-run flag", func(t *testing.T) {
		t.Parallel()

		for _, args := range []acceptanceArgs{{Check: true, Filter: "793a"}, {Check: true, DryRun: true}} {
			g := NewWithT(t)

			_, err := acceptancePlan(args, acceptanceEnv{Root: "/repo", LookPath: present, Exists: installed})

			g.Expect(err).To(MatchError(errAcceptanceUsage))
		}
	})
}
