//go:build targ

package dev

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

// Test861a3a1a_TestPkg_TheCommandLineNamesOnlyThatPackage verifies 861a3a1a: whatever package the developer names,
// the one go test command that runs names that package and no other.
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
