//go:build targ

package dev

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

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
			ctx := t.Context()
			mock, expectation := MockExecuteNilaway(t)
			execute, requests := countRequests(mock)
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
			g.Expect(awaitReturn(g, done, requests)).To(Succeed())
			g.Expect(policyMode).To(Equal([]bool{scenario.fix}))
		})
	}
}

func Test794b4b1b_SuccessfulRunReturnsSuccess(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	mock, expectation := MockExecuteNilaway(t)
	execute, requests := countRequests(mock)
	done := make(chan error, 1)
	go func() { done <- runConfiguredNilaway(context.Background(), false, nilawayArgs, execute) }()

	expectation.ArgsShould(match.BeAny, match.BeAny).Return(nil)

	g.Expect(awaitReturn(g, done, requests)).To(Succeed())
}

func Test794b4b1c_FailedRunReturnsTheFailureWithoutRetrying(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	mock, expectation := MockExecuteNilaway(t)
	execute, requests := countRequests(mock)
	done := make(chan error, 1)
	go func() { done <- runConfiguredNilaway(context.Background(), false, nilawayArgs, execute) }()

	expectation.ArgsShould(match.BeAny, match.BeAny).Return(errAnalyzerFailed)

	g.Expect(awaitReturn(g, done, requests)).To(MatchError(errAnalyzerFailed))
}

// unexported variables.
var (
	errAnalyzerFailed = errors.New("analyzer failed")
)

// awaitReturn waits for the run to return and checks that it made exactly one request.
func awaitReturn(g *WithT, done <-chan error, requests *atomic.Int32) error {
	err := <-done

	g.Expect(requests.Load()).To(Equal(int32(1)), "the run made more than one request: it retried")

	return err
}

// countRequests wraps the mock, counting requests. Only the first reaches the mock; a retry is answered at once with
// success so the run still returns and the count shows it, with no timer to wait out.
func countRequests(
	mock func(context.Context, []string) error,
) (func(context.Context, []string) error, *atomic.Int32) {
	var requests atomic.Int32

	return func(ctx context.Context, args []string) error {
		if requests.Add(1) > 1 {
			return nil
		}

		return mock(ctx, args)
	}, &requests
}
