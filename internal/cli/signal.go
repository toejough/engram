package cli

import (
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/toejough/targ"
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

// Run sets up force-exit signal handling and runs the CLI via targ.Main.
// targ's signal.NotifyContext cancels the context on the first SIGINT
// but we intercept subsequent signals to force exit via the second-signal handler.
func Run(stdout, stderr io.Writer, stdin io.Reader, exitFn func(int)) {
	sigCh := make(chan os.Signal, signalChannelBuffer)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ForceExitOnRepeatedSignal(sigCh, exitFn)

	targ.Main(Targets(stdout, stderr, stdin)...)
}

// unexported constants.
const (
	secondSignal        = 2  // Force exit on second signal
	signalChannelBuffer = 10 // Buffer size for signal channel
)
