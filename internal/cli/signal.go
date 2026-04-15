package cli

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/toejough/targ"
)

// Exported constants.
const (
	ExitCodeSigInt = 130
)

// ForceExitOnRepeatedSignal starts a goroutine that waits for two signals
// on the given channel, then calls exitFn.
func ForceExitOnRepeatedSignal(signals <-chan os.Signal, exitFn func(int)) {
	go func() {
		<-signals // first — handled gracefully by targ's signal.NotifyContext
		<-signals // second — force exit

		exitFn(ExitCodeSigInt)
	}()
}

// Run sets up force-exit signal handling and runs the CLI via targ.Main.
// targ's signal.NotifyContext cancels the context on the first SIGINT
// but keeps consuming subsequent signals until the handler returns,
// preventing default process termination. The force-exit handler
// restores the standard CLI expectation: second Ctrl-C = exit.
func Run(stdout, stderr io.Writer, stdin io.Reader, exitFn func(int)) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ForceExitOnRepeatedSignal(sigCh, exitFn)

	targ.Main(Targets(stdout, stderr, stdin)...)
}
