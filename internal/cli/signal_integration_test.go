package cli_test

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestForceExit_RealSignalDeliveryThroughForwardAsPulses replicates
// cmd/engram/main.go's StartSignalPulses closure with SIGUSR2 (registered
// only by this test; Notify overrides the default terminate disposition, so
// the test run is safe) and proves the full chain: real signal -> Notify ->
// ForwardAsPulses -> ForceExitOnRepeatedSignal. Pending same-signal
// coalescing can swallow a rapid second delivery, so the test keeps
// signalling (paced) until exit fires — over-delivery still means "force
// exit", so this can never false-pass.
func TestForceExit_RealSignalDeliveryThroughForwardAsPulses(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const buffer = 10

	exitCodes := make(chan int, 1)

	sigCh := make(chan os.Signal, buffer)
	signal.Notify(sigCh, syscall.SIGUSR2)

	pulses := make(chan struct{}, buffer)

	go cli.ForwardAsPulses(sigCh, pulses)

	cli.ForceExitOnRepeatedSignal(pulses, func(code int) { exitCodes <- code })

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
