package cli_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

func TestCommander_CollectsOutputOnSuccess(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	run := func(_ context.Context, _, _ string, _ []string, stdout, stderr io.Writer) error {
		_, _ = stdout.Write([]byte("out-bytes"))
		_, _ = stderr.Write([]byte("err-bytes"))

		return nil
	}

	stdout, stderr, err := commanderOver(run, nil).Run(context.Background(), "", "tool")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(stdout)).To(Equal("out-bytes"))
	g.Expect(string(stderr)).To(Equal("err-bytes"))
}

func TestCommander_NilNotFoundErrNeverTranslates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errSpawn := errors.New("spawn failed")
	run := func(_ context.Context, _, _ string, _ []string, _, _ io.Writer) error {
		return errSpawn
	}

	_, _, err := commanderOver(run, nil).Run(context.Background(), "", "tool")
	g.Expect(err).To(MatchError(errSpawn))
	g.Expect(err).NotTo(MatchError(update.ErrCommandNotFound))
}

func TestCommander_PassesCallThrough(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var gotDir, gotName string

	var gotArgs []string

	run := func(_ context.Context, dir, name string, args []string, _, _ io.Writer) error {
		gotDir, gotName, gotArgs = dir, name, args

		return nil
	}

	_, _, err := commanderOver(run, nil).Run(context.Background(), "/work", "git", "clone", "url")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gotDir).To(Equal("/work"))
	g.Expect(gotName).To(Equal("git"))
	g.Expect(gotArgs).To(Equal([]string{"clone", "url"}))
}

func TestCommander_TranslatesInjectedNotFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errPlatformNotFound := errors.New("platform: executable file not found")
	run := func(_ context.Context, _, _ string, _ []string, _, _ io.Writer) error {
		return fmt.Errorf("spawning: %w", errPlatformNotFound)
	}

	_, _, err := commanderOver(run, errPlatformNotFound).Run(context.Background(), "", "ghost")
	g.Expect(err).To(MatchError(update.ErrCommandNotFound))
	g.Expect(err).To(MatchError(errPlatformNotFound))
}

func TestCommander_WrapsFailureAndKeepsOutput(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errBoom := errors.New("boom")
	run := func(_ context.Context, _, _ string, _ []string, stdout, stderr io.Writer) error {
		_, _ = stdout.Write([]byte("partial"))
		_, _ = stderr.Write([]byte("diagnostic"))

		return errBoom
	}

	stdout, stderr, err := commanderOver(run, errors.New("not-found")).Run(
		context.Background(), "", "tool", "arg")
	g.Expect(err).To(MatchError(errBoom))
	g.Expect(err).NotTo(MatchError(update.ErrCommandNotFound))
	g.Expect(err).To(MatchError(ContainSubstring("tool [arg]")))
	g.Expect(string(stdout)).To(Equal("partial"))
	g.Expect(string(stderr)).To(Equal("diagnostic"))
}

// commanderOver builds the composed update.Commander from a fake RunCommand
// primitive and an injected platform not-found sentinel, through the real
// NewDeps wiring path (nil Getenv skips Embed; nil exit skips the
// force-exit watcher).
func commanderOver(
	run func(ctx context.Context, dir, name string, args []string, stdout, stderr io.Writer) error,
	notFound error,
) update.Commander {
	prims := cli.Primitives{RunCommand: run, NotFoundErr: notFound}

	return cli.NewDeps(prims, io.Discard, io.Discard, nil).Commander
}
