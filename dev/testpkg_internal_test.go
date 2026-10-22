//go:build targ

package dev

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

// Test861a3a1a_TestPkg_TheCommandLineNamesOnlyThatPackage verifies 861a3a1a: whatever package the developer names,
// the one go test command that runs names that package and no other, and reports the result of that run.
func Test861a3a1a_TestPkg_TheCommandLineNamesOnlyThatPackage(t *testing.T) {
	t.Parallel()

	for _, pkg := range []string{"./internal/luhmann", "./dev/acceptance", "./cmd/engram"} {
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			var calls []recordedCommand

			err := runTestPkg(t.Context(), testPkgArgs{Package: pkg}, recordingRunner(&calls, nil))

			g.Expect(err).NotTo(HaveOccurred())
			if err != nil {
				return
			}

			g.Expect(calls).To(Equal([]recordedCommand{
				{"go", "test", "-count=1", "-timeout=2m", "-race", pkg},
			}))

			failure := runTestPkg(t.Context(), testPkgArgs{Package: pkg}, recordingRunner(&calls, errGoTestFailed))

			g.Expect(failure).To(MatchError(errGoTestFailed))
		})
	}
}

// Test861b2a1a_TestPkg_ARunWithNoPackagePathStartsNoGoTest verifies 861b2a1a: without a package path the target
// reports the missing path and starts no go test, with or without a filter.
func Test861b2a1a_TestPkg_ARunWithNoPackagePathStartsNoGoTest(t *testing.T) {
	t.Parallel()

	for name, args := range map[string]testPkgArgs{
		"no arguments":  {},
		"only a filter": {Run: "TestSomething"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			var calls []recordedCommand

			err := runTestPkg(t.Context(), args, recordingRunner(&calls, nil))

			g.Expect(err).To(MatchError(errTestPkgNoPackage))
			g.Expect(calls).To(BeEmpty(), "go test was started")
		})
	}
}

// Test861b2a1b_TestPkg_AFilterGoesBeforeThePackagePath verifies 861b2a1b: with --run, go test receives -run and the
// expression before the package path.
func Test861b2a1b_TestPkg_AFilterGoesBeforeThePackagePath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var calls []recordedCommand

	err := runTestPkg(t.Context(), testPkgArgs{Package: "./internal/cli", Run: "Test792b7c1a|Test855d4h1f"},
		recordingRunner(&calls, nil))

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(calls).To(Equal([]recordedCommand{{
		"go", "test", "-count=1", "-timeout=2m", "-race", "-run", "Test792b7c1a|Test855d4h1f", "./internal/cli",
	}}))
}

// Test861b2a1c_TestPkg_TheCommandLineIsPinned verifies 861b2a1c: with a package path and no filter, go test gets
// -count=1, -timeout=2m, -race and the package path, and no other flag, in particular no build tag.
func Test861b2a1c_TestPkg_TheCommandLineIsPinned(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	argv, err := testPkgCommandLine(testPkgArgs{Package: "./internal/embed"})

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(argv).To(Equal([]string{"test", "-count=1", "-timeout=2m", "-race", "./internal/embed"}))
}

// Test861b2a1e_TestPkg_NoRaceLeavesOutTheRaceDetector verifies 861b2a1e: with --no-race, go test receives no -race and
// keeps the count, the timeout, the filter and the package path.
func Test861b2a1e_TestPkg_NoRaceLeavesOutTheRaceDetector(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var calls []recordedCommand

	err := runTestPkg(t.Context(), testPkgArgs{Package: "./internal/luhmann", NoRace: true, Run: "TestX"},
		recordingRunner(&calls, nil))

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(calls).To(Equal([]recordedCommand{
		{"go", "test", "-count=1", "-timeout=2m", "-run", "TestX", "./internal/luhmann"},
	}))
}

// Test861b2a1f_TestPkg_TagsGoBeforeTheFilterAndThePackagePath verifies 861b2a1f: with --tags, go test receives
// -tags=<value> before the filter and the package path.
func Test861b2a1f_TestPkg_TagsGoBeforeTheFilterAndThePackagePath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var calls []recordedCommand

	err := runTestPkg(t.Context(), testPkgArgs{Package: "./dev", Tags: "targ,integration", Run: "TestX"},
		recordingRunner(&calls, nil))

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(calls).To(Equal([]recordedCommand{
		{"go", "test", "-count=1", "-timeout=2m", "-race", "-tags=targ,integration", "-run", "TestX", "./dev"},
	}))
}

// TestTestPkg_AFailureOfTheRunnerIsTheFailureOfTheTarget pins that the target returns what the command it started
// returned, so that a failing go test fails the target (861b2a1d runs a real go test on a fixture package).
func TestTestPkg_AFailureOfTheRunnerIsTheFailureOfTheTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var calls []recordedCommand

	err := runTestPkg(t.Context(), testPkgArgs{Package: "./dev"}, recordingRunner(&calls, errGoTestFailed))

	g.Expect(err).To(MatchError(errGoTestFailed))
	g.Expect(calls).To(HaveLen(1))
}

// TestTestPkg_TheCommandLineFollowsTheOptions pins the command line of every combination of the options, in order:
// the count, the timeout, the race detector, the tags, the filter, the package path.
func TestTestPkg_TheCommandLineFollowsTheOptions(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		args testPkgArgs
		want []string
	}{
		"no options": {
			args: testPkgArgs{Package: "./p"},
			want: []string{"test", "-count=1", "-timeout=2m", "-race", "./p"},
		},
		"no-race": {
			args: testPkgArgs{Package: "./p", NoRace: true},
			want: []string{"test", "-count=1", "-timeout=2m", "./p"},
		},
		"tags": {
			args: testPkgArgs{Package: "./p", Tags: "targ"},
			want: []string{"test", "-count=1", "-timeout=2m", "-race", "-tags=targ", "./p"},
		},
		"no-race and tags": {
			args: testPkgArgs{Package: "./p", NoRace: true, Tags: "targ"},
			want: []string{"test", "-count=1", "-timeout=2m", "-tags=targ", "./p"},
		},
		"every option": {
			args: testPkgArgs{Package: "./p", NoRace: true, Tags: "targ", Run: "TestX"},
			want: []string{"test", "-count=1", "-timeout=2m", "-tags=targ", "-run", "TestX", "./p"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			argv, err := testPkgCommandLine(testCase.args)

			g.Expect(err).NotTo(HaveOccurred())
			if err != nil {
				return
			}

			g.Expect(argv).To(Equal(testCase.want))
		})
	}
}

// TestTestPkg_TheUsageNamesEveryOption pins that the refusal for a missing package path names all three options.
func TestTestPkg_TheUsageNamesEveryOption(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(errTestPkgNoPackage).To(MatchError(And(
		ContainSubstring("--run"), ContainSubstring("--no-race"), ContainSubstring("--tags"),
	)))
}

// unexported variables.
var (
	errGoTestFailed = errors.New("go test failed")
)

// recordedCommand is one command a recording runner was asked to run: its name, then its arguments.
type recordedCommand []string

// recordingRunner returns a runner that records the command it is asked to run into calls and returns result.
func recordingRunner(calls *[]recordedCommand, result error) commandRunner {
	return func(_ context.Context, name string, args ...string) error {
		*calls = append(*calls, append(recordedCommand{name}, args...))

		return result
	}
}
