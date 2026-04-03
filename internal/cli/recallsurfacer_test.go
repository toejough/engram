package cli_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"engram/internal/cli"

	. "github.com/onsi/gomega"
)

func TestBuildRecallSurfacer(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when memories dir missing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		dataDir := t.TempDir() // no memories subdirectory
		surfacer, err := cli.ExportBuildRecallSurfacer(context.Background(), dataDir)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(surfacer).To(BeNil())
	})

	t.Run("returns error for non-directory data path", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Create a file where memories dir would be, causing a non-ErrNotExist error.
		dataDir := t.TempDir()
		memoriesPath := filepath.Join(dataDir, "memories")
		writeErr := os.WriteFile(memoriesPath, []byte("not a dir"), 0o600)
		g.Expect(writeErr).NotTo(HaveOccurred())

		if writeErr != nil {
			return
		}

		surfacer, err := cli.ExportBuildRecallSurfacer(context.Background(), dataDir)
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("listing memories"))
		}

		g.Expect(surfacer).To(BeNil())
	})

	t.Run("returns surfacer when memories dir exists", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		dataDir := t.TempDir()
		memoriesDir := filepath.Join(dataDir, "memories")
		mkErr := os.MkdirAll(memoriesDir, 0o750)
		g.Expect(mkErr).NotTo(HaveOccurred())

		if mkErr != nil {
			return
		}

		surfacer, err := cli.ExportBuildRecallSurfacer(context.Background(), dataDir)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(surfacer).NotTo(BeNil())
	})

	t.Run("returns surfacer when new layout exists", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		dataDir := t.TempDir()
		feedbackDir := filepath.Join(dataDir, "memory", "feedback")
		mkErr := os.MkdirAll(feedbackDir, 0o750)
		g.Expect(mkErr).NotTo(HaveOccurred())

		if mkErr != nil {
			return
		}

		tomlContent := "type = \"feedback\"\nsituation = \"test\"\n\n[content]\nbehavior = \"test\"\n"
		writeErr := os.WriteFile(filepath.Join(feedbackDir, "test.toml"), []byte(tomlContent), 0o640)
		g.Expect(writeErr).NotTo(HaveOccurred())

		if writeErr != nil {
			return
		}

		surfacer, err := cli.ExportBuildRecallSurfacer(context.Background(), dataDir)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(surfacer).NotTo(BeNil())
	})
}

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

func TestRecallSurfacer_SBIAFormat(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	runner := &fakeSurfaceRunner{
		writeOutput: "<system-reminder source=\"engram\">\n  1. test-mem\n     Situation: when testing\n</system-reminder>\n",
	}

	surfacer := cli.NewRecallSurfacer(runner, "/tmp/data")
	result, err := surfacer.Surface("testing")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("Situation:"))
}

func TestRecordSurfacing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memPath := filepath.Join(dataDir, "test-memory.toml")

	initialContent := `situation = "test memory"
type = "feedback"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"
surfaced_count = 0

[content]
behavior = "some behavior"
action = "some action"
`
	writeErr := os.WriteFile(memPath, []byte(initialContent), 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	err := cli.ExportRecordSurfacing(memPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	updated, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(updated)
	g.Expect(content).To(ContainSubstring("surfaced_count = 1"))
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
