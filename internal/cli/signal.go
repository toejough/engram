package cli

import (
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

// Exported constants.
const (
	ExitCodeSigInt = 130
)

// ForceExitOnRepeatedSignal starts a goroutine that waits for two signals
// on the given channel, then calls exitFn. The first signal allows graceful
// shutdown; the second signal forces immediate exit.
func ForceExitOnRepeatedSignal(signals <-chan os.Signal, exitFn func(int)) {
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
func SetupSignalHandling(stdout, stderr io.Writer, exitFn func(int)) []any {
	sigCh := make(chan os.Signal, signalChannelBuffer)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ForceExitOnRepeatedSignal(sigCh, exitFn)

	return Targets(stdout, stderr)
}

// unexported constants.
const (
	secondSignal        = 2  // Force exit on second signal
	signalChannelBuffer = 10 // Buffer size for signal channel
)
