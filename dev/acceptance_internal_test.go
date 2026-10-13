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

func Test793e8i_H1ReportPlan(t *testing.T) {
	t.Parallel()

	present := func(name string) (string, error) { return "/bin/" + name, nil }

	t.Run("the report is one node command that saves the report beside the observations", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		plan, err := h1ReportPlan(acceptanceEnv{Root: "/repo", LookPath: present, Exists: func(string) bool { return true }})

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(plan).To(Equal([]acceptanceCommand{
			{Name: "node", Args: []string{"/repo/dev/acceptance/h1/report.js", "--write"}},
		}))
	})

	t.Run("missing dependencies are installed from the lockfile first", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		plan, err := h1ReportPlan(acceptanceEnv{Root: "/repo", LookPath: present, Exists: func(string) bool { return false }})

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(plan).To(HaveLen(2))
		g.Expect(plan[0]).To(Equal(acceptanceCommand{Name: "npm", Args: []string{"ci", "--prefix", "/repo/dev/acceptance"}}))
	})

	t.Run("a missing program fails and names it, and the report never needs claude", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"node", "npm"} {
			g := NewWithT(t)

			lookPath := func(program string) (string, error) {
				if program == name {
					return "", errors.New("not found")
				}

				return "/bin/" + program, nil
			}

			_, err := h1ReportPlan(acceptanceEnv{Root: "/repo", LookPath: lookPath, Exists: func(string) bool { return true }})

			g.Expect(err).To(MatchError(errAcceptanceProgram))
			g.Expect(err).To(MatchError(ContainSubstring("`" + name + "`")))
		}

		g := NewWithT(t)

		onlyClaudeMissing := func(program string) (string, error) {
			if program == "claude" {
				return "", errors.New("not found")
			}

			return "/bin/" + program, nil
		}

		_, err := h1ReportPlan(acceptanceEnv{Root: "/repo", LookPath: onlyClaudeMissing, Exists: func(string) bool { return true }})

		g.Expect(err).NotTo(HaveOccurred())
	})
}
