// Package main provides the migrate-v2 CLI entry point.
package main

import (
	"os"

	"engram/internal/migrate"
)

func main() { os.Exit(migrate.RunCLI(os.Args[1:])) }
