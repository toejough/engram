package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestFeedback_ContentFieldsPreservedThroughRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `title = "Rich Memory"
content = "Detailed content here"
observation_type = "pattern"
concepts = ["concurrency", "goroutines"]
keywords = ["go", "sync"]
principle = "avoid shared state"
anti_pattern = "shared mutable map"
rationale = "race conditions"
confidence = "high"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-06-01T00:00:00Z"
surfaced_count = 5
followed_count = 2
ignored_count = 1
irrelevant_count = 0
last_surfaced_at = "2024-06-01T00:00:00Z"
`
	err = os.WriteFile(
		filepath.Join(memDir, "rich-mem.toml"),
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
			"rich-mem",
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

	// Verify all content fields survived the round-trip.
	var record memory.MemoryRecord

	_, decErr := toml.DecodeFile(
		filepath.Join(memDir, "rich-mem.toml"), &record,
	)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.Title).To(Equal("Rich Memory"))
	g.Expect(record.Content).To(Equal("Detailed content here"))
	g.Expect(record.ObservationType).To(Equal("pattern"))
	g.Expect(record.Concepts).To(ConsistOf("concurrency", "goroutines"))
	g.Expect(record.Keywords).To(ConsistOf("go", "sync"))
	g.Expect(record.Principle).To(Equal("avoid shared state"))
	g.Expect(record.AntiPattern).To(Equal("shared mutable map"))
	g.Expect(record.Rationale).To(Equal("race conditions"))
	g.Expect(record.Confidence).To(Equal("high"))
	g.Expect(record.CreatedAt).To(Equal("2024-01-01T00:00:00Z"))
	g.Expect(record.UpdatedAt).To(Equal("2024-06-01T00:00:00Z"))
	g.Expect(record.SurfacedCount).To(Equal(5))
	g.Expect(record.FollowedCount).To(Equal(3)) // incremented from 2
	g.Expect(record.IgnoredCount).To(Equal(1))
	g.Expect(record.IrrelevantCount).To(Equal(0))
	g.Expect(record.LastSurfacedAt).To(Equal("2024-06-01T00:00:00Z"))
}

func TestFeedback_IrrelevantQueries_CappedAt20(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Pre-populate with 20 existing queries.
	var queryLines strings.Builder

	for i := range 20 {
		fmt.Fprintf(&queryLines, "  \"query-%d\",\n", i)
	}

	tomlContent := fmt.Sprintf(
		"title = \"cap-test\"\nsurfaced_count = 1\n"+
			"irrelevant_queries = [\n%s]\n",
		queryLines.String(),
	)
	err = os.WriteFile(
		filepath.Join(memDir, "cap-test.toml"),
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
			"--name", "cap-test",
			"--data-dir", dataDir,
			"--surfacing-query", "new query",
			"--irrelevant",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filepath.Join(memDir, "cap-test.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var record memory.MemoryRecord

	_, decErr := toml.Decode(string(data), &record)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	const maxIrrelevantQueries = 20

	g.Expect(record.IrrelevantQueries).To(HaveLen(maxIrrelevantQueries))
	// Oldest ("query-0") dropped, newest ("new query") appended.
	g.Expect(record.IrrelevantQueries[0]).To(Equal("query-1"))
	g.Expect(record.IrrelevantQueries[maxIrrelevantQueries-1]).To(
		Equal("new query"),
	)
}

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
	var record memory.MemoryRecord

	_, decErr := toml.DecodeFile(
		filepath.Join(memDir, "test-mem.toml"), &record,
	)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.FollowedCount).To(Equal(3))
	g.Expect(record.IgnoredCount).To(Equal(1))
	g.Expect(record.IrrelevantCount).To(Equal(1))
}

func TestFeedback_Irrelevant_WithSurfacingQuery_PersistsQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := "title = \"persist-query\"\nsurfaced_count = 1\n"
	err = os.WriteFile(
		filepath.Join(memDir, "persist-query.toml"),
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
			"--name", "persist-query",
			"--data-dir", dataDir,
			"--surfacing-query", "how to test",
			"--irrelevant",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Re-read the TOML and verify query was persisted.
	data, readErr := os.ReadFile(
		filepath.Join(memDir, "persist-query.toml"),
	)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var record memory.MemoryRecord

	_, decErr := toml.Decode(string(data), &record)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.IrrelevantQueries).To(Equal([]string{"how to test"}))
}

func TestFeedback_Irrelevant_WithSurfacingQuery_PrintsContextMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := "title = \"irr-ctx\"\nsurfaced_count = 1\n"
	err = os.WriteFile(
		filepath.Join(memDir, "irr-ctx.toml"),
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
			"--name", "irr-ctx",
			"--data-dir", dataDir,
			"--surfacing-query", "how to test",
			"--irrelevant",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Surfacing context recorded for refinement"))
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
	var record memory.MemoryRecord

	_, decErr := toml.DecodeFile(
		filepath.Join(memDir, "test-mem.toml"), &record,
	)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.IgnoredCount).To(Equal(2))
	g.Expect(record.FollowedCount).To(Equal(3))
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
	var record memory.MemoryRecord

	_, decErr := toml.DecodeFile(
		filepath.Join(memDir, "test-mem.toml"), &record,
	)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.FollowedCount).To(Equal(4))
	g.Expect(record.IgnoredCount).To(Equal(1))
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
	var record memory.MemoryRecord

	_, decErr := toml.DecodeFile(
		filepath.Join(memDir, "before-flags.toml"), &record,
	)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.FollowedCount).To(Equal(1))
}

func TestFeedback_SurfacingContextFlags_Accepted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := "title = \"ctx-test\"\nsurfaced_count = 1\n"
	err = os.WriteFile(
		filepath.Join(memDir, "ctx-test.toml"),
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
			"--name", "ctx-test",
			"--data-dir", dataDir,
			"--relevant", "--used",
			"--surfacing-query", "how to test",
			"--tool-name", "Read",
			"--tool-input", "foo.go",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Surfacing context message should NOT appear for relevant feedback.
	g.Expect(stdout.String()).NotTo(ContainSubstring("Surfacing context"))
}
