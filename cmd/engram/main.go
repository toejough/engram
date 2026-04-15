// Package main provides the engram CLI binary entry point (ARCH-6).
package main

import (
	"os"

	"github.com/toejough/targ"

	"engram/internal/cli"
)

func main() {
	// targ.Main handles dispatch, help, errors-to-stderr, and exit (ARCH-6).
	targ.Main(cli.Targets(os.Stdout, os.Stderr, os.Stdin)...)
}
