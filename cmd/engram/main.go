// Package main provides the engram CLI binary entry point (ARCH-6).
package main

import (
	"fmt"
	"os"

	"engram/internal/cli"
)

func main() {
	// ARCH-6: Exit 0 always — hook failures must not break Claude Code.
	err := cli.Run(os.Args, os.Stdout, newHashStore(os.Getenv("ENGRAM_DATA")))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
