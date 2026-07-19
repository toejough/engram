package cli

import (
	"io"
	"os"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/debuglog"
)

// Main is the engram binary's entry-point composition. Constructs the
// debug logger from ENGRAM_DEBUG_LOG, sets up signal handling, and dispatches
// to targ. Intended to be called from cmd/engram/main.go's main() as a
// single-statement thin wrapper.
func Main(stdout, stderr io.Writer, exitFn func(int)) {
	// New only errors when path is non-empty and OpenFile fails; in that
	// case we proceed with a nil logger (no-op) so the CLI still runs.
	// FIXME(#700): this os.Getenv call is IO. it should be injected from outside of the `internal` package. We should
	// be able to create a deterministic check - no stdlib or third-party packages allowed to be called from within
	// `internal` unless in an explicit allowlist. Plan + validated depguard/forbidigo design in issue #700; remove this
	// FIXME only when #700 is resolved.
	logger, _ := debuglog.New(os.Getenv("ENGRAM_DEBUG_LOG"), "engram")

	targ.Main(SetupSignalHandling(stdout, stderr, exitFn, logger)...)
}
