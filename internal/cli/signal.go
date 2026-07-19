package cli

import (
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/toejough/engram/internal/debuglog"
)

// Exported constants.
const (
	ExitCodeSigInt = 130
)

// ForceExitOnRepeatedSignal starts a goroutine that waits for two pulses
// on the given channel, then calls exitFn. The first pulse allows graceful
// shutdown; the second forces immediate exit. Pulses are pure struct{}
// units — cmd/engram adapts real os.Signal deliveries into them, so no
// os.Signal type crosses into internal/ (#700).
func ForceExitOnRepeatedSignal(signals <-chan struct{}, exitFn func(int)) {
	var signalCount atomic.Int32

	go func() {
		for range signals {
			count := signalCount.Add(1)
			if count >= secondSignal {
				// Second signal or later: force exit immediately
				exitFn(ExitCodeSigInt)

				return
			}
			// First signal: will be handled by targ's context cancellation
		}
	}()
}

// SetupSignalHandling registers signal handlers and starts the force-exit goroutine.
// Returns the configured targets for targ.Main.
//
// Deprecated: interim shim only — deleted by the #700 wiring task; cmd/engram
// registers signals itself and calls Targets directly.
func SetupSignalHandling(
	stdout, stderr io.Writer,
	exitFn func(int),
	logger *debuglog.Logger,
) []any {
	sigCh := make(chan os.Signal, signalChannelBuffer)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	pulses := make(chan struct{}, signalChannelBuffer)

	go func() {
		for range sigCh {
			pulses <- struct{}{}
		}
	}()

	ForceExitOnRepeatedSignal(pulses, exitFn)

	return Targets(stdout, stderr, exitFn, logger)
}

// unexported constants.
const (
	secondSignal        = 2  // Force exit on second signal
	signalChannelBuffer = 10 // Buffer size for signal channel
)
