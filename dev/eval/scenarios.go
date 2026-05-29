//go:build targ

package eval

import "regexp"

// Scenarios returns the M1 scenario set. Keep Go-flavored tasks dominant
// so the vault's Go/process conventions are actually exercised.
func Scenarios() []Scenario {
	goTestCheck := BehaviorCheck{
		Name:    "used-go-test-not-targ",
		Kind:    ConventionViolation,
		Pattern: usedGoTestDirectly,
	}
	return []Scenario{
		{
			Name: "calibration",
			Prompt: "In a new Go module under the current directory, create a tiny CLI " +
				"named `greet` with one subcommand `hello` that prints \"hello, world\". " +
				"Write it test-first and make the tests pass.",
			ExpectedVault: []string{"use targ not go test", "TDD red-first", "AI-Used trailer", "DI for I/O"},
			SuccessCmd:    nil,
			Checks:        []BehaviorCheck{goTestCheck},
		},
		{
			Name: "todo-cli",
			Prompt: "In a new Go module under the current directory, build a `todo` CLI " +
				"supporting `add <text>`, `list`, and `done <n>`, persisting to a JSON file. " +
				"Work test-first.",
			ExpectedVault: []string{"use targ not go test", "TDD red-first", "make with capacity"},
			Checks:        []BehaviorCheck{goTestCheck},
		},
		{
			Name: "sqlite-explorer",
			Prompt: "In a new Go module under the current directory, build a `sqex` CLI that " +
				"opens a SQLite file and prints its table names. Work test-first.",
			ExpectedVault: []string{"DI for I/O", "TDD red-first", "use targ not go test"},
			Checks:        []BehaviorCheck{goTestCheck},
		},
	}
}

// unexported variables.
var (
	// usedGoTestDirectly matches invoking the raw Go test runner instead of
	// the project's targ build tool — a documented engram/Go convention.
	usedGoTestDirectly = regexp.MustCompile(`\bgo\s+(test|vet|build)\b`)
)
