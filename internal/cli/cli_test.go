package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

// T-18: `correct` subcommand with no API key returns error
func TestT18_CorrectSubcommandWithoutAPIKeyReturnsError(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	// Ensure no API token is set.
	t.Setenv("ENGRAM_API_TOKEN", "")

	var buf bytes.Buffer

	err := cli.Run([]string{
		"engram", "correct",
		"--message", "remember to use targ",
		"--data-dir", dataDir,
	}, &buf)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("no API token"))
	}
}

// T-19: `correct` with non-matching message produces empty stdout
func TestT19_CorrectWithNonMatchingMessageProducesEmptyStdout(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	var buf bytes.Buffer

	err := cli.Run([]string{
		"engram", "correct",
		"--message", "hello world",
		"--data-dir", dataDir,
	}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify stdout is empty
	g.Expect(buf.String()).To(BeEmpty())

	// Verify no memories directory was created
	memoriesDir := filepath.Join(dataDir, "memories")
	_, statErr := os.Stat(memoriesDir)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

func TestRun_CorrectMissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	err := cli.Run([]string{"engram", "correct"}, &buf)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--message"))
	}
}

func TestRun_NoArgs(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	err := cli.Run([]string{"engram"}, &buf)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("usage"))
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	err := cli.Run([]string{"engram", "bogus"}, &buf)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown command"))
	}
}
