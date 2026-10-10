package cli_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

func Test792b4c1a1_CommanderIntegration_RunsCommand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	commander := realDepsForTest().Commander

	stdout, _, err := commander.Run(context.Background(), "", "echo", "hello world")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(string(stdout))).To(Equal("hello world"))
}

func Test792b4c1a2_CommanderIntegration_RunsInDir(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	commander := realDepsForTest().Commander
	dir := t.TempDir()

	// macOS TempDir sits under a symlink (/var → /private/var); compare
	// against the resolved path. `pwd -P` forces the physical path: plain
	// `pwd` is logical by default and Go's os/exec sets PWD=Dir in the
	// child env, so it would echo the unresolved dir verbatim.
	resolved, evalErr := filepath.EvalSymlinks(dir)
	g.Expect(evalErr).NotTo(HaveOccurred())

	if evalErr != nil {
		return
	}

	stdout, _, err := commander.Run(context.Background(), dir, "pwd", "-P")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(string(stdout))).To(Equal(resolved))
}

// These tests drive the composed primCommander over the REAL exec
// primitive (realPrimitives mirrors cmd/engram/main.go's literal).
// The production C-1 closure is guarded end-to-end by cli_test.go's
// Test792a11a1b_RunCommand_EndToEnd (#700). The integration tests below verify
// the composed Commander over real os funcs.

func Test792b4c2a1_CommanderIntegration_ReportsFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	commander := realDepsForTest().Commander

	_, _, err := commander.Run(context.Background(), "", "false")
	g.Expect(err).To(HaveOccurred())
}

func Test792b4c3a1_CommanderIntegration_TranslatesNotFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	commander := realDepsForTest().Commander

	_, _, err := commander.Run(context.Background(), "", "engram-no-such-binary-7f3a")
	g.Expect(err).To(MatchError(update.ErrCommandNotFound))
}
