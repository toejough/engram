// Package main provides the engram CLI binary entry point (ARCH-6).
package main

import (
	"os"

	"engram/internal/cli"
	"engram/internal/debuglog"

	"github.com/toejough/targ"
)

func main() {
	_ = debuglog.Init(os.Getenv("ENGRAM_DEBUG_LOG"), "engram")

	targ.Main(cli.SetupSignalHandling(os.Stdout, os.Stderr, os.Exit)...)
}
