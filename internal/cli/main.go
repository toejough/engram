// Package cli wires the engram CLI binary's entry-point composition.
package cli

import (
	"io"
	"os"

	"engram/internal/debuglog"

	"github.com/toejough/targ"
)

// Main is the engram binary's entry-point composition. Initializes the
// debug log from ENGRAM_DEBUG_LOG, sets up signal handling, and dispatches
// to targ. Intended to be called from cmd/engram/main.go's main() as a
// single-statement thin wrapper.
func Main(stdout, stderr io.Writer, exitFn func(int)) {
	_ = debuglog.Init(os.Getenv("ENGRAM_DEBUG_LOG"), "engram")

	targ.Main(SetupSignalHandling(stdout, stderr, exitFn)...)
}
