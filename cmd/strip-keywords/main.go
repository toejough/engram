// Package main provides the strip-keywords CLI entry point.
// It removes legacy "\nKeywords: ..." suffixes from situation fields
// in all memory TOML files under the data directory.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"engram/internal/stripkeywords"
)

func main() {
	flags := flag.NewFlagSet("strip-keywords", flag.ContinueOnError)
	dataDir := flags.String(
		"data-dir",
		filepath.Join(os.Getenv("HOME"), ".claude", "engram", "data"),
		"path to data directory",
	)

	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if err := stripkeywords.Run(*dataDir, stripkeywords.DefaultDeps()); err != nil {
		fmt.Fprintf(os.Stderr, "strip-keywords: %v\n", err)
		os.Exit(1)
	}
}
