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

// ForwardAsPulses forwards each value received on in as a unit pulse on
// out. It is generic so cmd/engram can feed a chan os.Signal without
// os.Signal entering any internal signature (#700); tests drive it with a
// chan int.
func ForwardAsPulses[T any](in <-chan T, out chan<- struct{}) {
	for range in {
		out <- struct{}{}
	}
}

// SetupSignalHandling registers signal handlers and starts the force-exit goroutine.
// Returns the configured targets for targ.Main.
//
// Deprecated: interim shim only — deleted by the #700 wiring task; cmd/engram
// wires signal pulses through cli.Primitives.StartSignalPulses instead.
func SetupSignalHandling(
	stdout, stderr io.Writer,
	exitFn func(int),
	logger *debuglog.Logger,
) []any {
	sigCh := make(chan os.Signal, signalChannelBuffer)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	pulses := make(chan struct{}, signalChannelBuffer)

	go ForwardAsPulses(sigCh, pulses)

	ForceExitOnRepeatedSignal(pulses, exitFn)

	return Targets(stdout, stderr, exitFn, logger)
}

// unexported constants.
const (
	secondSignal        = 2  // Force exit on second signal
	signalChannelBuffer = 10 // Buffer size for signal + pulse channels
)

// startForceExit starts the repeated-signal force-exit watcher from the
// injected starter primitive. A nil primitive or exit func (minimal test
// Deps) skips registration. The pulse channel, buffer size, and force-exit
// policy live here — cmd only subscribes and forwards (#700).
func startForceExit(prims Primitives, exit func(int)) {
	if prims.StartSignalPulses == nil || exit == nil {
		return
	}

	pulses := make(chan struct{}, signalChannelBuffer)
	prims.StartSignalPulses(pulses, signalChannelBuffer)
	ForceExitOnRepeatedSignal(pulses, exit)
}
