//go:build targ

package dev

import (
	"slices"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
)

// Test862a3c1a_TestParity_TheCommandLineNamesTheParityTagAndTheEmbedPackage verifies 862a3c1a: the parity command
// runs the tests the parity tag selects, in the package that holds them.
func Test862a3c1a_TestParity_TheCommandLineNamesTheParityTagAndTheEmbedPackage(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	argv := parityCommandLine()

	g.Expect(argv).To(ContainElement("-tags=parity"))
	g.Expect(slices.Index(argv, "./internal/embed")).To(Equal(len(argv)-1), "the package path is not last")
}

// Test862b2c1a_TestParity_TheCommandLineIsPinned verifies 862b2c1a: go test is run with -count=1, -tags=parity and
// ./internal/embed, and with no other flag.
func Test862b2c1a_TestParity_TheCommandLineIsPinned(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var calls []recordedCommand

	err := runParity(t.Context(), recordingRunner(&calls, nil))

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(calls).To(Equal([]recordedCommand{{"go", "test", "-count=1", "-tags=parity", "./internal/embed"}}))
}

// Test862b2c1b_TestParity_TheAggregatesDoNotRunIt verifies 862b2c1b: test-parity is in no dependency group of
// check-for-fail or check-full.
func Test862b2c1b_TestParity_TheAggregatesDoNotRunIt(t *testing.T) {
	t.Parallel()

	for name, aggregate := range map[string]*targ.Target{
		"check-for-fail": checkForFailTarget,
		"check-full":     checkFullTarget,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			var members []string

			for _, group := range aggregate.GetDepGroups() {
				for _, target := range group.Targets {
					members = append(members, target.GetName())
				}
			}

			g.Expect(members).NotTo(BeEmpty())
			g.Expect(slices.Contains(members, testParityTarget.GetName())).To(BeFalse(),
				"%s depends on %s", name, testParityTarget.GetName())
		})
	}
}

// TestParity_AFailureOfTheRunnerIsTheFailureOfTheTarget pins that the target returns what the command it started
// returned, so that a failing parity test fails the target.
func TestParity_AFailureOfTheRunnerIsTheFailureOfTheTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var calls []recordedCommand

	err := runParity(t.Context(), recordingRunner(&calls, errGoTestFailed))

	g.Expect(err).To(MatchError(errGoTestFailed))
	g.Expect(calls).To(HaveLen(1))
}
