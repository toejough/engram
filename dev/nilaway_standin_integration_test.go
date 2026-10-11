//go:build targ && integration

package dev

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// unexported constants.
const (
	standInArgPrefix = "arg="
	standInMarker    = "inherited"
	standInScript    = `#!/bin/sh
{
	printf 'marker=%s\n' "$ENGRAM_NILAWAY_MARKER"
	if [ "${GOMEMLIMIT+set}" = set ]; then printf 'GOMEMLIMIT=%s\n' "$GOMEMLIMIT"; fi
	if [ "${GOGC+set}" = set ]; then printf 'GOGC=%s\n' "$GOGC"; fi
	if [ "${GOMAXPROCS+set}" = set ]; then printf 'GOMAXPROCS=%s\n' "$GOMAXPROCS"; fi
	for argument in "$@"; do printf 'arg=%s\n' "$argument"; done
} > "$ENGRAM_NILAWAY_CAPTURE"
exit "${ENGRAM_NILAWAY_EXIT:-0}"
`
)

// unexported variables.
var (
	resourceVariables = []string{"GOMEMLIMIT", "GOGC", "GOMAXPROCS"}
)

// currentResourceVariables returns the resource variables set in this process, including those set empty.
func currentResourceVariables() map[string]string {
	current := make(map[string]string, len(resourceVariables))
	for _, key := range resourceVariables {
		if value, present := os.LookupEnv(key); present {
			current[key] = value
		}
	}
	return current
}

func formatSettings(settings map[string]string) string {
	pairs := make([]string, 0, len(settings))
	for key, value := range settings {
		pairs = append(pairs, key+"="+value)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}

// readStandInArgs returns the arguments the stand-in was started with.
func readStandInArgs(t *testing.T, capture string) []string {
	t.Helper()
	args := make([]string, 0)
	for _, line := range readStandInLines(t, capture) {
		if value, found := strings.CutPrefix(line, standInArgPrefix); found {
			args = append(args, value)
		}
	}
	return args
}

func readStandInLines(t *testing.T, capture string) []string {
	t.Helper()
	contents, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(contents), "\n"), "\n")
}

// readStandInMarker returns the unrelated marker variable the stand-in saw.
func readStandInMarker(t *testing.T, capture string) string {
	t.Helper()
	for _, line := range readStandInLines(t, capture) {
		if value, found := strings.CutPrefix(line, "marker="); found {
			return value
		}
	}
	return ""
}

// readStandInResources returns the resource variables present in the stand-in's environment.
func readStandInResources(t *testing.T, capture string) map[string]string {
	t.Helper()
	seen := make(map[string]string, len(resourceVariables))
	for _, line := range readStandInLines(t, capture) {
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		for _, resource := range resourceVariables {
			if key == resource {
				seen[key] = value
			}
		}
	}
	return seen
}

// setResourceVariables makes exactly the supplied resource variables present in this test's process environment.
func setResourceVariables(t *testing.T, supplied map[string]string) {
	t.Helper()
	for _, key := range resourceVariables {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	for key, value := range supplied {
		t.Setenv(key, value)
	}
}

// standInNilaway puts a stand-in nilaway first on PATH and returns the file it reports into.
func standInNilaway(t *testing.T, exitCode int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("subprocess fixture requires a POSIX shell")
	}
	dir := t.TempDir()
	capture := filepath.Join(dir, "capture")
	if err := os.WriteFile(filepath.Join(dir, "nilaway"), []byte(standInScript), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ENGRAM_NILAWAY_CAPTURE", capture)
	t.Setenv("ENGRAM_NILAWAY_MARKER", standInMarker)
	t.Setenv("ENGRAM_NILAWAY_EXIT", strconv.Itoa(exitCode))
	return capture
}
