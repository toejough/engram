//go:build targ && integration

package dev

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// Test861b2a1d_TestPkg_AFailingTestFailsTheTargetAndShowsTheOutput verifies 861b2a1d, and that the Go toolchain
// exits non-zero when a test fails: the target runs a real go test on a fixture package whose test fails.
func Test861b2a1d_TestPkg_AFailingTestFailsTheTargetAndShowsTheOutput(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	dir := writeFixtureModule(t, map[string]string{
		"failing/failing_test.go": "package failing\n\nimport \"testing\"\n\n" +
			"func TestFailsOnPurpose(t *testing.T) { t.Fatal(\"" + fixtureFailureMessage + "\") }\n",
	})

	var output bytes.Buffer

	err := runTestPkg(t.Context(), testPkgArgs{Package: "./failing"}, goInDirectory(dir, &output))

	var exitErr *exec.ExitError

	g.Expect(errors.As(err, &exitErr)).To(BeTrue(), "the target did not fail with the failure of go test: %v", err)
	g.Expect(output.String()).To(ContainSubstring(fixtureFailureMessage))
	g.Expect(output.String()).To(ContainSubstring("FAIL"))
}

// unexported constants.
const (
	fixtureFailureMessage = "deliberate failure of the fixture test"
	fixtureModuleName     = "example.com/fixture"
)

// goInDirectory returns a runner that runs the command for real in dir and writes its output to output.
func goInDirectory(dir string, output *bytes.Buffer) commandRunner {
	return func(ctx context.Context, name string, args ...string) error {
		command := exec.CommandContext(ctx, name, args...)
		command.Dir = dir
		command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOPROXY=off")
		command.Stdout = output
		command.Stderr = output

		return command.Run()
	}
}

// writeFixtureModule writes a module without dependencies and the given files, named relative to the module, into a
// new temporary directory and returns the directory.
func writeFixtureModule(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	all := map[string]string{"go.mod": "module " + fixtureModuleName + "\n\ngo " + fixtureGoVersion + "\n"}

	maps.Copy(all, files)

	for name, contents := range all {
		path := filepath.Join(dir, name)

		err := os.MkdirAll(filepath.Dir(path), 0o700)
		if err != nil {
			t.Fatal(err)
		}

		err = os.WriteFile(path, []byte(contents), 0o600)
		if err != nil {
			t.Fatal(err)
		}
	}

	return dir
}
