package cli_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"engram/internal/cli"

	. "github.com/onsi/gomega"
)

func TestRecallSurfacer(t *testing.T) {
	t.Parallel()

	t.Run("surfaces memories in prompt mode", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		runner := &fakeSurfaceRunner{
			writeOutput: "[engram] memory one\n[engram] memory two",
		}
		surfacer := cli.NewRecallSurfacer(runner, "/data")

		result, err := surfacer.Surface("my query text")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result).To(Equal("[engram] memory one\n[engram] memory two"))
		g.Expect(runner.opts.Mode).To(Equal("prompt"))
		g.Expect(runner.opts.Message).To(Equal("my query text"))
		g.Expect(runner.opts.DataDir).To(Equal("/data"))
	})

	t.Run("empty query returns empty string", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		runner := &fakeSurfaceRunner{}
		surfacer := cli.NewRecallSurfacer(runner, "/data")

		result, err := surfacer.Surface("")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result).To(BeEmpty())
		g.Expect(runner.called).To(BeFalse())
	})

	t.Run("runner error propagates", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		runner := &fakeSurfaceRunner{
			err: errors.New("surface failed"),
		}
		surfacer := cli.NewRecallSurfacer(runner, "/data")

		_, err := surfacer.Surface("query")
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("surface failed"))
		}
	})
}

type fakeSurfaceRunner struct {
	opts        cli.SurfaceRunnerOptions
	writeOutput string
	err         error
	called      bool
}

func (f *fakeSurfaceRunner) Run(_ context.Context, w io.Writer, opts cli.SurfaceRunnerOptions) error {
	f.called = true
	f.opts = opts

	if f.err != nil {
		return f.err
	}

	if f.writeOutput != "" {
		_, _ = io.WriteString(w, f.writeOutput)
	}

	return nil
}
