// Package main provides the strip-keywords CLI entry point.
// It removes legacy "\nKeywords: ..." suffixes from situation fields
// in all memory TOML files under the data directory.
package main

import (
	"os"

	"engram/internal/stripkeywords"
)

func main() { os.Exit(stripkeywords.RunCLI(os.Args[1:])) }
