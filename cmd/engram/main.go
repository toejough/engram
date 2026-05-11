// Package main provides the engram CLI binary entry point (ARCH-6).
package main

import (
	"os"

	"engram/internal/cli"
)

func main() {
	cli.Main(os.Stdout, os.Stderr, os.Exit)
}
