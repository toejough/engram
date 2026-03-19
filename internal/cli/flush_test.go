package cli_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

// T-370: flush command runs learn, evaluate, context-update in order.
func TestT370_FlushRunsInOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	runner := cli.NewFlushRunner(
		func() error { callOrder = append(callOrder, "learn"); return nil },
		func() error { callOrder = append(callOrder, "evaluate"); return nil },
		func() error { callOrder = append(callOrder, "context-update"); return nil },
	)

	err := runner.Run()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callOrder).To(Equal([]string{"learn", "evaluate", "context-update"}))
}

// T-371: flush command stops on first step error.
func TestT371_FlushStopsOnError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	evalErr := errors.New("evaluate failed")

	runner := cli.NewFlushRunner(
		func() error { callOrder = append(callOrder, "learn"); return nil },
		func() error { callOrder = append(callOrder, "evaluate"); return evalErr },
		func() error { callOrder = append(callOrder, "context-update"); return nil },
	)

	err := runner.Run()
	g.Expect(err).To(MatchError(ContainSubstring("evaluate failed")))

	g.Expect(callOrder).To(Equal([]string{"learn", "evaluate"}))
}
