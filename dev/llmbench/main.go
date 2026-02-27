// dev/llmbench benchmarks different strategies for calling the claude CLI.
//
// Usage:
//
//	go run ./dev/llmbench [--calls N] [--scenario ...] [--model ...]
//
// Scenarios: baseline, parallel, interactive, models, api
//
// Each scenario sends N identical trivial prompts and measures wall-clock time.
package main

import "github.com/toejough/projctl/dev/llmbench/internal/bench"

func main() {
	bench.Main()
}
