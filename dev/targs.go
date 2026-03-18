//go:build targ

package dev

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/toejough/targ"
	targdev "github.com/toejough/targ/dev"
)

func init() {
	// Engram's spec-traced tests use TestT<N>_ naming (not TestProperty_).
	if os.Getenv("TARG_BASELINE_PATTERN") == "" {
		os.Setenv("TARG_BASELINE_PATTERN", `TestT[0-9]+_`)
	}

	checkStmtCoverageTarget := targ.Targ(checkStmtCoverage).
		Name("check-stmt-coverage").
		Description("Check per-package statement coverage floor")
	checkStmtCoverageTarget.Deps(targdev.TestForFail)

	targdev.CheckFull.Deps(checkStmtCoverageTarget)

	targ.Register(checkStmtCoverageTarget)
}

// unexported constants.
const (
	stmtCoverageFloor = 4.0
)

// blockKey uniquely identifies a coverage block.
type blockKey struct {
	file   string
	coords string
}

// packageStatements tracks covered and total statement counts for a package.
type packageStatements struct {
	covered int
	total   int
}

func checkStmtCoverage(ctx context.Context) error {
	targ.Print(ctx, "Checking per-package statement coverage...\n")

	coverage, err := readPackageCoverage("coverage.out")
	if err != nil {
		return fmt.Errorf("reading coverage: %w", err)
	}

	var failures []string

	for pkg, stats := range coverage {
		if stats.total == 0 {
			// No statements — nothing to enforce (e.g., declaration-only packages).
			continue
		}

		pct := float64(stats.covered) / float64(stats.total) * 100

		if pct < stmtCoverageFloor {
			failures = append(failures,
				fmt.Sprintf("  %s: %.1f%% (%d/%d stmts)", pkg, pct, stats.covered, stats.total),
			)
		}
	}

	if len(failures) > 0 {
		var msg strings.Builder

		fmt.Fprintf(&msg, "per-package statement coverage below %.1f%% floor:\n", stmtCoverageFloor)

		for _, line := range failures {
			fmt.Fprintln(&msg, line)
		}

		return errors.New(msg.String())
	}

	targ.Printf(ctx, "Per-package statement coverage OK (floor: %.1f%%)\n", stmtCoverageFloor)

	return nil
}

// packageFromFilePath extracts the Go package import path from a coverage file path.
// e.g. "engram/internal/evaluate/evaluator.go" -> "engram/internal/evaluate"
func packageFromFilePath(filePath string) string {
	idx := strings.LastIndex(filePath, "/")
	if idx < 0 {
		return filePath
	}

	return filePath[:idx]
}

// readPackageCoverage parses a coverage.out profile and returns per-package
// statement coverage using the maximum hit count across all test runs for each block.
func readPackageCoverage(path string) (map[string]*packageStatements, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	defer file.Close()

	// Use max count across all duplicate blocks (from cross-package test runs).
	blockMaxCount := make(map[blockKey]int)
	blockStmts := make(map[blockKey]int)

	coverageLineRe := regexp.MustCompile(`^(.+):(\d+\.\d+,\d+\.\d+) (\d+) (\d+)$`)

	const maxLineBytes = 1024 * 1024 // 1 MiB — coverage.out can have very long lines

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, maxLineBytes), maxLineBytes)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		matches := coverageLineRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		filePath, coords, stmtsStr, countStr := matches[1], matches[2], matches[3], matches[4]

		stmts, err := strconv.Atoi(stmtsStr)
		if err != nil {
			return nil, fmt.Errorf("parsing statement count in %q: %w", line, err)
		}

		count, err := strconv.Atoi(countStr)
		if err != nil {
			return nil, fmt.Errorf("parsing hit count in %q: %w", line, err)
		}

		key := blockKey{file: filePath, coords: coords}
		blockStmts[key] = stmts

		if existing, ok := blockMaxCount[key]; !ok || count > existing {
			blockMaxCount[key] = count
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", path, err)
	}

	pkgStats := make(map[string]*packageStatements)

	for key, stmts := range blockStmts {
		pkg := packageFromFilePath(key.file)

		if _, ok := pkgStats[pkg]; !ok {
			pkgStats[pkg] = &packageStatements{}
		}

		pkgStats[pkg].total += stmts

		if blockMaxCount[key] > 0 {
			pkgStats[pkg].covered += stmts
		}
	}

	return pkgStats, nil
}
