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
	logger, _ := debuglog.New(os.Getenv("ENGRAM_DEBUG_LOG"), "engram")

	targ.Main(SetupSignalHandling(stdout, stderr, exitFn, logger)...)
}
