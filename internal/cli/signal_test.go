package cli_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestForceExitOnRepeatedSignal(t *testing.T) {
	t.Parallel()

	t.Run("calls exit on second signal", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		const pulseBuffer = 2

		pulses := make(chan struct{}, pulseBuffer)
		exitCalled := make(chan int, 1)

		cli.ForceExitOnRepeatedSignal(pulses, func(code int) {
			exitCalled <- code
		})

		pulses <- struct{}{}

		pulses <- struct{}{}

		select {
		case code := <-exitCalled:
			g.Expect(code).To(Equal(cli.ExitCodeSigInt))
		case <-time.After(time.Second):
			t.Fatal("exit not called within 1s of second signal")
		}
	})

	t.Run("does not exit on first signal alone", func(t *testing.T) {
		t.Parallel()

		pulses := make(chan struct{}, 1)
		exitCalled := make(chan int, 1)

		cli.ForceExitOnRepeatedSignal(pulses, func(code int) {
			exitCalled <- code
		})

		pulses <- struct{}{} // first only

		const shortWait = 100 * time.Millisecond

		select {
		case <-exitCalled:
			t.Fatal("exit called after only one signal")
		case <-time.After(shortWait):
			// good — no exit after one signal
		}
	})
}
