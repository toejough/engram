package main

// Scenario test for the subprocess-runner technical component (861c2zk): 792b4c4a6. The other scenarios of the
// requirement (792b4c4a1 to 792b4c4a5) are in main_test.go and internal/cli/spawner_test.go. Spec:
// docs/specs/technical/subprocess-runner.md.
//
// The production closure hands the child the process's own os.Stdin, os.Stdout and os.Stderr, so the test cannot
// swap them without racing the parallel tests. It runs the test binary itself as the caller, with pipes for the
// caller's streams: the helper test below does the spawn and the parent reads what reached the caller's streams.
// No installed binary, home or network is involved (a real sh, as the sibling tests in main_test.go).

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test792b4c4a6_SpawnPrimitives_ProgramSharesCallerStreams verifies 792b4c4a6 (The program shares the caller's
// streams): a program that reads its standard input and writes to its standard output and standard error, spawned
// through the production RunInherited, reads the caller's standard input and writes to the caller's standard
// output and error, each on its own stream.
func Test792b4c4a6_SpawnPrimitives_ProgramSharesCallerStreams(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("shell-based fixture assumes a POSIX shell")
	}

	g := NewWithT(t)

	executable, err := os.Executable()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Given a caller (the helper process) whose streams are pipes the test controls.
	caller := exec.CommandContext(t.Context(), executable, "-test.run=^"+srHelperTestName+"$")
	caller.Env = append(os.Environ(), srHelperEnv+"=1")
	caller.Stdin = strings.NewReader(srCallerInput + "\n")

	var callerStdout, callerStderr bytes.Buffer

	caller.Stdout, caller.Stderr = &callerStdout, &callerStderr

	// When it spawns the program that reads stdin and writes stdout and stderr.
	runErr := caller.Run()

	// Then the program read the caller's input and wrote to the caller's output and error.
	g.Expect(runErr).NotTo(HaveOccurred(), "caller stderr: %s", callerStderr.String())
	g.Expect(callerStdout.String()).To(ContainSubstring("out:" + srCallerInput + "\n"))
	g.Expect(callerStdout.String()).NotTo(ContainSubstring(srErrMarker))
	g.Expect(callerStderr.String()).To(ContainSubstring(srErrMarker + "\n"))
	g.Expect(callerStderr.String()).NotTo(ContainSubstring("out:"))
}

// TestSpawnPrimitives_RunInheritedHelperProcess is the caller process of
// Test792b4c4a6_SpawnPrimitives_ProgramSharesCallerStreams: in an ordinary test run it does nothing; when the
// environment names it, it spawns the shell program through the production RunInherited and reports a failure to
// start or a non-zero status through its own exit status.
func TestSpawnPrimitives_RunInheritedHelperProcess(t *testing.T) {
	t.Parallel()

	if os.Getenv(srHelperEnv) != "1" {
		t.Skip("runs only as the caller process of Test792b4c4a6")
	}

	g := NewWithT(t)

	const program = `read line; printf 'out:%s\n' "$line"; printf '` + srErrMarker + `\n' >&2`

	code, err := spawnPrimitives().RunInherited("sh", []string{"-c", program}, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(code).To(Equal(0))
}

// unexported constants.
const (
	srCallerInput    = "caller-input-line"
	srErrMarker      = "err-marker-line"
	srHelperEnv      = "ENGRAM_SR_HELPER_PROCESS"
	srHelperTestName = "TestSpawnPrimitives_RunInheritedHelperProcess"
)
