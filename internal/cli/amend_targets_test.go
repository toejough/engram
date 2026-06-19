package cli_test

import (
	"testing"

	"github.com/onsi/gomega"
)

// TestTargets_AmendRegistered asserts the amend target is wired into the CLI:
// executing `engram amend` against an empty vault reaches RunAmend (which fails
// loud with "note not found") rather than targ rejecting an unknown command.
func TestTargets_AmendRegistered(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	vault := t.TempDir()

	stderr := executeForTest(t, []string{
		"engram", "amend",
		"--target", "999",
		"--vault", vault,
	})

	// The command is recognized and runs; RunAmend reports the missing note.
	g.Expect(stderr).NotTo(gomega.ContainSubstring("unknown"))
	g.Expect(stderr).To(gomega.ContainSubstring("note not found"))
}
