// Package main provides the engram CLI binary entry point (ARCH-6).
package main

import (
	"os"

	"engram/internal/cli"
)

func main() {
	// ARCH-6: RunSafe always exits 0 (errors printed to stderr).
	cli.RunSafe(os.Args, os.Stdout, os.Stderr, os.Stdin)
}
