package main

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestForwardAsPulses_ForwardsEachSignal(t *testing.T) {
	t.Parallel()

	const signalCount = 2

	sigCh := make(chan os.Signal, signalCount)
	pulses := make(chan struct{}, signalCount)

	go forwardAsPulses(sigCh, pulses)

	sigCh <- syscall.SIGUSR1

	sigCh <- syscall.SIGUSR1

	const pulseTimeout = time.Second

	for range signalCount {
		select {
		case <-pulses:
		case <-time.After(pulseTimeout):
			t.Fatal("pulse not forwarded within timeout")
		}
	}
}

// TestRegisterForceExit_SecondSignalForcesExit delivers real OS signals to
// this process. SIGUSR2 is registered only by this test, and Notify overrides
// the default (terminate) disposition, so the test run is safe. Pending
// same-signal coalescing can swallow a rapid second delivery, so the test
// keeps signalling (paced) until exit fires — over-delivery still means
// "force exit", so this can never false-pass.
func TestRegisterForceExit_SecondSignalForcesExit(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	exitCodes := make(chan int, 1)

	registerForceExit(func(code int) { exitCodes <- code }, syscall.SIGUSR2)

	proc, err := os.FindProcess(os.Getpid())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	const (
		pace    = 20 * time.Millisecond
		maxWait = 5 * time.Second
	)

	deadline := time.After(maxWait)

	for {
		g.Expect(proc.Signal(syscall.SIGUSR2)).To(gomega.Succeed())

		select {
		case code := <-exitCodes:
			g.Expect(code).To(gomega.Equal(cli.ExitCodeSigInt))

			return
		case <-deadline:
			t.Fatal("force-exit not fired within timeout")
		case <-time.After(pace):
		}
	}
}
