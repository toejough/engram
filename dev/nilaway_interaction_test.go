//go:build targ

package dev

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"
)

func Test794b4b1a_OneRequestCarriesTheContextAndThePolicysArguments(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name string
		fix  bool
	}{{"check", false}, {"fix", true}} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			execute, expectation := MockExecuteNilaway(t)
			placeholder := []string{"placeholder-arguments", "for-" + scenario.name}
			var policyMode []bool
			policy := func(fix bool) []string {
				policyMode = append(policyMode, fix)
				return placeholder
			}
			done := make(chan error, 1)
			go func() { done <- runConfiguredNilaway(ctx, scenario.fix, policy, execute) }()

			call := expectation.ArgsShould(match.BeAny, match.BeAny)
			got := call.GetArgs()
			call.Return(nil)

			g.Expect(got.Ctx).To(BeIdenticalTo(ctx))
			g.Expect(got.Args).To(Equal(placeholder))
			g.Expect(awaitReturn(g, done)).To(Succeed())
			g.Expect(policyMode).To(Equal([]bool{scenario.fix}))
		})
	}
}

func Test794b4b1b_SuccessfulRunReturnsSuccess(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	execute, expectation := MockExecuteNilaway(t)
	done := make(chan error, 1)
	go func() { done <- runConfiguredNilaway(context.Background(), false, nilawayArgs, execute) }()

	expectation.ArgsShould(match.BeAny, match.BeAny).Return(nil)

	g.Expect(awaitReturn(g, done)).To(Succeed())
}

func Test794b4b1c_FailedRunReturnsTheFailureWithoutRetrying(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	execute, expectation := MockExecuteNilaway(t)
	done := make(chan error, 1)
	go func() { done <- runConfiguredNilaway(context.Background(), false, nilawayArgs, execute) }()

	expectation.ArgsShould(match.BeAny, match.BeAny).Return(errAnalyzerFailed)

	g.Expect(awaitReturn(g, done)).To(MatchError(errAnalyzerFailed))
}

// unexported constants.
const (
	returnWait = 2 * time.Second
)

// unexported variables.
var (
	errAnalyzerFailed = errors.New("analyzer failed")
)

// awaitReturn waits for the run to return. The mock blocks on any request after the first one the test answered,
// so a run that retries never returns and fails here.
func awaitReturn(g *WithT, done <-chan error) error {
	select {
	case err := <-done:
		return err
	case <-time.After(returnWait):
		g.Expect(false).To(BeTrue(), "the run did not return after one answered request; it made a second request")
		return nil
	}
}
