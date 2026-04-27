package cli_test

import (
	"bytes"
	"os"
	"syscall"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestForceExitOnRepeatedSignal(t *testing.T) {
	t.Parallel()

	t.Run("calls exit on second signal", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		sigCh := make(chan os.Signal, 2)
		exitCalled := make(chan int, 1)

		cli.ForceExitOnRepeatedSignal(sigCh, func(code int) {
			exitCalled <- code
		})

		sigCh <- syscall.SIGINT

		sigCh <- syscall.SIGINT

		select {
		case code := <-exitCalled:
			g.Expect(code).To(Equal(cli.ExitCodeSigInt))
		case <-time.After(time.Second):
			t.Fatal("exit not called within 1s of second signal")
		}
	})

	t.Run("does not exit on first signal alone", func(t *testing.T) {
		t.Parallel()

		sigCh := make(chan os.Signal, 1)
		exitCalled := make(chan int, 1)

		cli.ForceExitOnRepeatedSignal(sigCh, func(code int) {
			exitCalled <- code
		})

		sigCh <- syscall.SIGINT // first only

		const shortWait = 100 * time.Millisecond

		select {
		case <-exitCalled:
			t.Fatal("exit called after only one signal")
		case <-time.After(shortWait):
			// good — no exit after one signal
		}
	})
}

func TestSetupSignalHandling_ReturnsTargets(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	targets := cli.SetupSignalHandling(&stdout, &stderr, func(_ int) {})
	g.Expect(targets).To(HaveLen(5))
}
