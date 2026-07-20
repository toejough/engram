package cli_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/update"
)

// These tests drive the composed primCommander over the REAL exec
// primitive (realPrimitives mirrors cmd/engram/main.go's literal —
// doctrine flag DRIFT): the relocated TestOsCommander_* coverage
// (#700 rework — integration tests with real os funcs live in
// internal _test files).

func TestCommanderIntegration_ReportsFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	commander := realDepsForTest().Commander

	_, _, err := commander.Run(context.Background(), "", "false")
	g.Expect(err).To(HaveOccurred())
}

func TestCommanderIntegration_RunsCommand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	commander := realDepsForTest().Commander

	stdout, _, err := commander.Run(context.Background(), "", "echo", "hello world")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(string(stdout))).To(Equal("hello world"))
}

func TestCommanderIntegration_RunsInDir(t *testing.T) {
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

func TestCommanderIntegration_TranslatesNotFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	commander := realDepsForTest().Commander

	_, _, err := commander.Run(context.Background(), "", "engram-no-such-binary-7f3a")
	g.Expect(err).To(MatchError(update.ErrCommandNotFound))
}
