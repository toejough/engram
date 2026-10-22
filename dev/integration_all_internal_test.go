//go:build targ

package dev

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
)

// Test862a3a1a_TestIntegrationAll_BothAggregatesRunItInTheirFinalGroup verifies 862a3a1a: the full check, and the
// check that fails fast, run test-integration-all, in the final dependency group where Engram's other dev-package
// checks are.
func Test862a3a1a_TestIntegrationAll_BothAggregatesRunItInTheirFinalGroup(t *testing.T) {
	t.Parallel()

	for name, aggregate := range map[string][]string{
		"check-for-fail": groupNames(checkForFailTarget.GetDepGroups()),
		"check-full":     groupNames(checkFullTarget.GetDepGroups()),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(aggregate).To(ContainElement(testIntegrationAllTarget.GetName()))
		})
	}
}

// Test862b2a1a_TestIntegrationAll_TheCommandSelectsByTagOverTheWholeModule verifies 862b2a1a: go test is run with
// -count=1, -timeout=10m, -race, -tags=targ,integration and ./..., and with no -run.
func Test862b2a1a_TestIntegrationAll_TheCommandSelectsByTagOverTheWholeModule(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var calls []recordedCommand

	err := runIntegrationAll(t.Context(), recordingRunner(&calls, nil))

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(calls).To(Equal([]recordedCommand{{
		"go", "test", "-count=1", "-timeout=10m", "-race", "-tags=targ,integration", "./...",
	}}))
	g.Expect(calls[0]).NotTo(ContainElement("-run"))
}

// TestIntegrationAll_AFailureOfTheRunnerIsTheFailureOfTheTarget pins that the target returns what the command it
// started returned (862b2a1c runs a real go test on a fixture module).
func TestIntegrationAll_AFailureOfTheRunnerIsTheFailureOfTheTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var calls []recordedCommand

	err := runIntegrationAll(t.Context(), recordingRunner(&calls, errGoTestFailed))

	g.Expect(err).To(MatchError(errGoTestFailed))
	g.Expect(calls).To(HaveLen(1))
}

// groupNames returns the names of the targets of the last dependency group.
func groupNames(groups []targ.DepGroup) []string {
	if len(groups) == 0 {
		return []string{}
	}

	last := groups[len(groups)-1]
	names := make([]string, 0, len(last.Targets))

	for _, target := range last.Targets {
		names = append(names, target.GetName())
	}

	return names
}
