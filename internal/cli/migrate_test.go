package cli_test

import (
	"bytes"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestMigrateScoresFlags_BuildsCorrectArgs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	args := cli.MigrateScoresArgs{
		DataDir: "/data/test",
		Apply:   true,
	}

	flags := cli.MigrateScoresFlags(args)

	g.Expect(flags).To(ContainElement("--data-dir"))
	g.Expect(flags).To(ContainElement("/data/test"))
	g.Expect(flags).To(ContainElement("--apply"))
}

func TestMigrateScores_ApplyFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-scores",
			"--data-dir", dataDir,
			"--apply",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("(apply)"))
	g.Expect(output).NotTo(ContainSubstring("dry-run"))
}

func TestMigrateScores_DefaultDataDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate-scores"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// applyDataDirDefault resolves to ~/.claude/engram/data
	g.Expect(stdout.String()).To(ContainSubstring("engram"))
}

func TestMigrateScores_DryRunByDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-scores",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("dry-run"))
	g.Expect(stdout.String()).NotTo(ContainSubstring("apply"))
}

func TestMigrateScores_FlagParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate-scores", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("migrate-scores"))
	}
}

func TestMigrateScores_ParsesFlags(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-scores",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring(dataDir))
}
