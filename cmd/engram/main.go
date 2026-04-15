// Package main provides the engram CLI binary entry point (ARCH-6).
package main

import (
	"os"

	"engram/internal/cli"

	"github.com/toejough/targ"
)

func main() {
	targ.Main(cli.SetupSignalHandling(os.Stdout, os.Stderr, os.Stdin, os.Exit)...)
}
