// Package main provides the engram CLI binary entry point (ARCH-2).
package main

import (
	"fmt"
	"os"

	_ "modernc.org/sqlite"

	"engram/internal/cli"
)

func main() {
	// ARCH-2: Exit 0 always — hook failures must not break Claude Code.
	err := cli.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
