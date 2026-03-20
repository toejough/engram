package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestFeedback_Irrelevant_IncrementsIrrelevantCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `title = "Test Memory"
content = "Some content"
followed_count = 3
ignored_count = 1
`
	err = os.WriteFile(
		filepath.Join(memDir, "test-mem.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "feedback",
			"test-mem",
			"--data-dir", dataDir,
			"--irrelevant",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Feedback recorded"))
	g.Expect(stdout.String()).To(ContainSubstring("irrelevant"))

	// Verify irrelevant_count incremented and other counters unchanged.
	var record map[string]any

	_, decErr := toml.DecodeFile(
		filepath.Join(memDir, "test-mem.toml"), &record,
	)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record["followed_count"]).To(BeEquivalentTo(3))
	g.Expect(record["ignored_count"]).To(BeEquivalentTo(1))
	g.Expect(record["irrelevant_count"]).To(BeEquivalentTo(1))
}

func TestFeedback_MemoryNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "feedback",
			"nonexistent-memory",
			"--data-dir", dataDir,
			"--relevant",
			"--used",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("nonexistent-memory"))
	}
}

func TestFeedback_MemoryWriteError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `title = "Read Only"
content = "Cannot write back"
followed_count = 0
`
	memPath := filepath.Join(memDir, "readonly-mem.toml")

	err = os.WriteFile(memPath, []byte(tomlContent), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Make directory read-only so the temp file write fails.
	err = os.Chmod(memDir, 0o500)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	t.Cleanup(func() {
		_ = os.Chmod(memDir, 0o750)
	})

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "feedback",
			"readonly-mem",
			"--data-dir", dataDir,
			"--relevant",
			"--used",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("writing temp"))
	}
}

func TestFeedback_MissingDataDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "feedback",
			"test-mem",
			"--relevant",
			"--used",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--data-dir"))
	}
}

func TestFeedback_MissingRelevanceFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "feedback",
			"test-mem",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--relevant"))
	}
}

func TestFeedback_MissingSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "feedback",
			"--data-dir", dataDir,
			"--relevant",
			"--used",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("slug"))
	}
}

func TestFeedback_RelevantNotused_IncrementsIgnored(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `title = "Test Memory"
content = "Some content"
followed_count = 3
ignored_count = 1
`
	err = os.WriteFile(
		filepath.Join(memDir, "test-mem.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "feedback",
			"test-mem",
			"--data-dir", dataDir,
			"--relevant",
			"--notused",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Feedback recorded"))
	g.Expect(stdout.String()).To(ContainSubstring("relevant, not used"))

	// Verify ignored_count incremented from 1 to 2.
	var record map[string]any

	_, decErr := toml.DecodeFile(
		filepath.Join(memDir, "test-mem.toml"), &record,
	)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record["ignored_count"]).To(BeEquivalentTo(2))
	g.Expect(record["followed_count"]).To(BeEquivalentTo(3))
}

func TestFeedback_RelevantUsed_IncrementsFollowed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `title = "Test Memory"
content = "Some content"
followed_count = 3
ignored_count = 1
`
	err = os.WriteFile(
		filepath.Join(memDir, "test-mem.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "feedback",
			"test-mem",
			"--data-dir", dataDir,
			"--relevant",
			"--used",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Feedback recorded"))
	g.Expect(stdout.String()).To(ContainSubstring("relevant, used"))

	// Verify followed_count incremented from 3 to 4.
	var record map[string]any

	_, decErr := toml.DecodeFile(
		filepath.Join(memDir, "test-mem.toml"), &record,
	)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record["followed_count"]).To(BeEquivalentTo(4))
	g.Expect(record["ignored_count"]).To(BeEquivalentTo(1))
}

func TestFeedback_SlugBeforeFlags(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `title = "Before Flags"
content = "Slug comes before --data-dir"
followed_count = 0
`
	err = os.WriteFile(
		filepath.Join(memDir, "before-flags.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	// Slug before flags: engram feedback before-flags --data-dir /path --relevant --used
	err = cli.Run(
		[]string{
			"engram", "feedback",
			"before-flags",
			"--data-dir", dataDir,
			"--relevant",
			"--used",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Feedback recorded"))
	g.Expect(stdout.String()).To(ContainSubstring("before-flags"))

	// Verify followed_count incremented from 0 to 1.
	var record map[string]any

	_, decErr := toml.DecodeFile(
		filepath.Join(memDir, "before-flags.toml"), &record,
	)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record["followed_count"]).To(BeEquivalentTo(1))
}
