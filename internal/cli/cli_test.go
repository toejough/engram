package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

// T-18: `correct` subcommand runs pipeline
func TestT18_CorrectSubcommandRunsPipeline(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	var buf bytes.Buffer

	err := cli.Run([]string{
		"engram", "correct",
		"--message", "remember to use targ",
		"--data-dir", dataDir,
	}, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify a TOML file was created in <dataDir>/memories/
	memoriesDir := filepath.Join(dataDir, "memories")
	entries, readErr := os.ReadDir(memoriesDir)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).NotTo(BeEmpty())
	g.Expect(entries[0].Name()).To(HaveSuffix(".toml"))

	// Verify stdout contains a system reminder
	g.Expect(buf.String()).To(ContainSubstring("[engram]"))
	g.Expect(buf.String()).To(ContainSubstring("Memory captured"))
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
