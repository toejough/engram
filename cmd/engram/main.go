// Package main provides the engram CLI binary entry point (ARCH-6).
package main

import (
	"os"

	"engram/internal/cli"
)

func main() {
	// cli.Run sets up signal handling and dispatches via targ (ARCH-6).
	cli.Run(os.Stdout, os.Stderr, os.Stdin, os.Exit)
}
