// Package main provides the engram CLI binary entry point (ARCH-6).
package main

import (
	"os"

	"github.com/toejough/targ"

	"engram/internal/cli"
)

func main() {
	// ARCH-6: Target functions never return errors, so targ.Main always exits 0.
	targ.Main(cli.Targets(os.Stdout, os.Stderr, os.Stdin)...)
}
