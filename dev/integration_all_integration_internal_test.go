//go:build targ && integration

package dev

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// Test862b2a1b_TestIntegrationAll_ATaggedTestIsRunWhateverItsPackageAndName verifies 862b2a1b: a test tagged
// integration that is outside dev and is not named TestIntegration is run, by a real go test on a fixture module.
func Test862b2a1b_TestIntegrationAll_ATaggedTestIsRunWhateverItsPackageAndName(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	dir := writeFixtureModule(t, map[string]string{
		"deep/tagged/tagged_test.go": "//go:build integration\n\npackage tagged\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n" +
			"func TestWritesItsMarker(t *testing.T) {\n" +
			"\tif err := os.WriteFile(\"" + fixtureRunMarker + "\", []byte(\"ran\"), 0o600); err != nil {\n" +
			"\t\tt.Fatal(err)\n\t}\n}\n",
	})

	var output bytes.Buffer

	err := runIntegrationAll(t.Context(), goInDirectory(dir, &output))

	g.Expect(err).NotTo(HaveOccurred(), output.String())

	_, statErr := os.Stat(filepath.Join(dir, "deep", "tagged", fixtureRunMarker))

	g.Expect(statErr).NotTo(HaveOccurred(), "the tagged test did not run: "+output.String())
}

// Test862b2a1c_TestIntegrationAll_AFailingTaggedTestFailsTheCommand verifies 862b2a1c: a failing tagged test makes
// the command fail with the failure of go test, which a real go test on a fixture module shows. Like the test of
// 861b2a1d, it runs runIntegrationAll with the test's own runner, not the production entry testIntegrationAll.
func Test862b2a1c_TestIntegrationAll_AFailingTaggedTestFailsTheCommand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	dir := writeFixtureModule(t, map[string]string{
		"tagged/tagged_test.go": "//go:build integration\n\npackage tagged\n\nimport \"testing\"\n\n" +
			"func TestFailsOnPurpose(t *testing.T) { t.Fatal(\"" + fixtureFailureMessage + "\") }\n",
	})

	var output bytes.Buffer

	err := runIntegrationAll(t.Context(), goInDirectory(dir, &output))

	var exitErr *exec.ExitError

	g.Expect(errors.As(err, &exitErr)).To(BeTrue(), "the command did not fail with the failure of go test: %v", err)
	g.Expect(output.String()).To(ContainSubstring(fixtureFailureMessage))
}

// unexported constants.
const (
	// fixtureRunMarker is the file the tagged fixture test writes into its package directory when it runs.
	fixtureRunMarker = "ran.marker"
)
