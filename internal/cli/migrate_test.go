package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRunMigrate_AlreadyV2Skipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	v2TOML := `schema_version = 2
type = "feedback"
source = "human"
situation = "when running tests"

[content]
behavior = "running go test directly"
impact = "misses coverage"
action = "use targ test"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(feedbackDir, "mem.toml"), []byte(v2TOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{},
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Migrated 0"))
	g.Expect(stdout.String()).To(ContainSubstring("skipped 1"))
}

func TestRunMigrate_ErrorFile_ContinuesAndReports(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	// Write an invalid TOML file.
	writeErr := os.WriteFile(
		filepath.Join(feedbackDir, "bad.toml"),
		[]byte("this is not valid { toml [[["),
		0o640,
	)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	// Write a valid v1 file.
	v1TOML := `schema_version = 1
type = "feedback"
source = "agent"
situation = "test"

[content]
behavior = "b"
impact = "i"
action = "a"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr2 := os.WriteFile(filepath.Join(feedbackDir, "good.toml"), []byte(v1TOML), 0o640)
	g.Expect(writeErr2).NotTo(HaveOccurred())

	if writeErr2 != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{},
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("ERROR:"))
	g.Expect(output).To(ContainSubstring("bad.toml"))
	g.Expect(output).To(ContainSubstring("Migrated 1"))
}

func TestRunMigrate_LegacyMemoriesDir(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	legacyDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(legacyDir, 0o750)).To(Succeed())

	v1TOML := `schema_version = 1
type = "feedback"
source = "agent"
situation = "test"

[content]
behavior = "b"
impact = "i"
action = "a"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(legacyDir, "mem.toml"), []byte(v1TOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{},
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Migrated 1"))
}

func TestRunMigrate_NoDirs_NoError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{},
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Migrated 0 memories, skipped 0 (already v2)"))
}

func TestRunMigrate_OutputFormat(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	// Two v1 files and one v2 file.
	v1TOML := `schema_version = 1
type = "feedback"
source = "agent"
situation = "test"

[content]
behavior = "b"
impact = "i"
action = "a"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	v2TOML := `schema_version = 2
type = "feedback"
source = "human"
situation = "test"

[content]
behavior = "b"
impact = "i"
action = "a"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	for _, name := range []string{"mem1.toml", "mem2.toml"} {
		writeErr := os.WriteFile(filepath.Join(feedbackDir, name), []byte(v1TOML), 0o640)
		g.Expect(writeErr).NotTo(HaveOccurred())

		if writeErr != nil {
			return
		}
	}

	writeErr := os.WriteFile(filepath.Join(feedbackDir, "mem3.toml"), []byte(v2TOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{},
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Migrated 2 memories, skipped 1 (already v2)"))
}

func TestRunMigrate_SituationFallbackFact(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	factsDir := filepath.Join(dataDir, "memory", "facts")

	g.Expect(os.MkdirAll(factsDir, 0o750)).To(Succeed())

	// Fact with no subject and no situation.
	v1TOML := `schema_version = 1
type = "fact"
source = "agent"
situation = ""

[content]
subject = ""
predicate = "exists"
object = "something"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(factsDir, "mem.toml"), []byte(v1TOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{},
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filepath.Join(factsDir, "mem.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`situation = "General knowledge"`))
}

func TestRunMigrate_SituationFallbackFeedback(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	// Feedback with no behavior and no situation.
	v1TOML := `schema_version = 1
type = "feedback"
source = "agent"
situation = ""

[content]
behavior = ""
impact = "something"
action = "do something"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(feedbackDir, "mem.toml"), []byte(v1TOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{},
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filepath.Join(feedbackDir, "mem.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`situation = "General development"`))
}

func TestRunMigrate_SituationInferenceFromBehavior(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	v1TOML := `schema_version = 1
type = "feedback"
source = "agent"
situation = ""

[content]
behavior = "Running go test directly"
impact = "misses coverage"
action = "use targ test"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(feedbackDir, "mem.toml"), []byte(v1TOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{},
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filepath.Join(feedbackDir, "mem.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`situation = "When running go test directly"`))
}

func TestRunMigrate_SituationInferenceFromSubject(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	factsDir := filepath.Join(dataDir, "memory", "facts")

	g.Expect(os.MkdirAll(factsDir, 0o750)).To(Succeed())

	v1TOML := `schema_version = 1
type = "fact"
source = "agent"
situation = ""

[content]
subject = "engram"
predicate = "uses"
object = "TF-IDF"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(factsDir, "mem.toml"), []byte(v1TOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{},
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(filepath.Join(factsDir, "mem.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`situation = "When working with engram"`))
}

func TestRunMigrate_SourceNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{name: "user_correction", source: "user correction, 2026-04-02", expected: "human"},
		{name: "human_string", source: "Human", expected: "human"},
		{name: "empty_source", source: "", expected: "agent"},
		{name: "agent_string", source: "agent", expected: "agent"},
		{name: "llm_source", source: "claude-3-haiku", expected: "agent"},
		{name: "contains_user", source: "user feedback", expected: "human"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			dataDir := t.TempDir()
			feedbackDir := filepath.Join(dataDir, "memory", "feedback")

			g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

			v1TOML := `schema_version = 1
type = "feedback"
source = "` + testCase.source + `"
situation = "test"

[content]
behavior = "b"
impact = "i"
action = "a"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"
`

			writeErr := os.WriteFile(filepath.Join(feedbackDir, "mem.toml"), []byte(v1TOML), 0o640)
			g.Expect(writeErr).NotTo(HaveOccurred())

			if writeErr != nil {
				return
			}

			var stdout bytes.Buffer

			err := cli.Run(
				[]string{"engram", "migrate", "--data-dir", dataDir},
				&stdout, &bytes.Buffer{},
				strings.NewReader(""),
			)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			data, readErr := os.ReadFile(filepath.Join(feedbackDir, "mem.toml"))
			g.Expect(readErr).NotTo(HaveOccurred())

			if readErr != nil {
				return
			}

			g.Expect(string(data)).To(ContainSubstring(`source = "` + testCase.expected + `"`))
		})
	}
}

func TestRunMigrate_UsageInError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("migrate"))
	}
}

func TestRunMigrate_V1FeedbackMigratedToV2(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	v1TOML := `schema_version = 1
type = "feedback"
source = "user correction, 2026-04-02"
situation = "when running tests"
core = true
project_scoped = true
project_slug = "engram"

[content]
behavior = "running go test directly"
impact = "misses coverage"
action = "use targ test"

created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-02T00:00:00Z"
surfaced_count = 5
followed_count = 3
not_followed_count = 1
irrelevant_count = 0
missed_count = 2
initial_confidence = 0.8
`

	writeErr := os.WriteFile(filepath.Join(feedbackDir, "test-mem.toml"), []byte(v1TOML), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{},
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Migrated 1"))
	g.Expect(stdout.String()).To(ContainSubstring("skipped 0"))

	// Verify the migrated file has v2 fields.
	data, readErr := os.ReadFile(filepath.Join(feedbackDir, "test-mem.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(data)
	g.Expect(content).To(ContainSubstring(`schema_version = 2`))
	g.Expect(content).To(ContainSubstring(`source = "human"`))
	g.Expect(content).To(ContainSubstring(`type = "feedback"`))
	g.Expect(content).To(ContainSubstring(`behavior = "running go test directly"`))

	// Legacy fields must be stripped.
	g.Expect(content).NotTo(ContainSubstring("core"))
	g.Expect(content).NotTo(ContainSubstring("project_scoped"))
	g.Expect(content).NotTo(ContainSubstring("surfaced_count"))
	g.Expect(content).NotTo(ContainSubstring("followed_count"))
	g.Expect(content).NotTo(ContainSubstring("initial_confidence"))
}
