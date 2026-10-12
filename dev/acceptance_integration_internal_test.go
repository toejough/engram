//go:build targ && integration

package dev

import (
	"os/exec"
	"testing"

	. "github.com/onsi/gomega"
)

// Test793e8a_ExtractorSuiteRunsInCheckFull runs the acceptance package's dependency-free node:test suite (the
// extractor, credential, transcript and preflight tests), so that check-full guards the claim that the specs are the
// test source. The live suite never runs here. It skips, with its reason, on a machine without node.
func Test793e8a_ExtractorSuiteRunsInCheckFull(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on PATH: the acceptance package's node:test suite did not run")
	}

	command := exec.CommandContext(t.Context(), node, "--test", "test/*.test.js")
	command.Dir = "acceptance"

	output, err := command.CombinedOutput()

	g.Expect(err).NotTo(HaveOccurred(), string(output))
}
