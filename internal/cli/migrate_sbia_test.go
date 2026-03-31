package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestMigrateSBIA_ArchiveRenameError_Logged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	legacyTOML := `title = "tier-b"
content = "content"
confidence = "B"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`

	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "b.toml"), []byte(legacyTOML), 0o644,
	)).To(Succeed())

	var stdout bytes.Buffer

	deps := cli.MigrationDeps{
		ListDir:  os.ReadDir,
		ReadFile: os.ReadFile,
		MkdirAll: os.MkdirAll,
		Rename: func(_, _ string) error {
			return errSimulatedRename
		},
		Converter: &failingSBIAConverter{},
		Stdout:    &stdout,
	}

	ctx := context.Background()

	err := cli.ExecuteMigration(ctx, dataDir, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("error processing"))
}

func TestMigrateSBIA_ConversionFailure_Archived(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	legacyTOML := `title = "will fail conversion"
content = "some content"
confidence = "A"
principle = "a principle"
anti_pattern = "an anti-pattern"
rationale = "a rationale"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`
	memPath := filepath.Join(memoriesDir, "fail-convert.toml")
	g.Expect(os.WriteFile(memPath, []byte(legacyTOML), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	failConverter := &failingSBIAConverter{}
	deps := cli.MigrationDeps{
		ListDir:   os.ReadDir,
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		MkdirAll:  os.MkdirAll,
		Rename:    os.Rename,
		Converter: failConverter,
		Stdout:    &stdout,
	}

	ctx := context.Background()

	err := cli.ExecuteMigration(ctx, dataDir, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Original file should be gone (archived due to failure).
	_, statErr := os.Stat(memPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "failed conversion should be archived")

	// Should exist in archive.
	archivePath := filepath.Join(dataDir, "archive", "fail-convert.toml")
	_, archiveStatErr := os.Stat(archivePath)
	g.Expect(archiveStatErr).NotTo(HaveOccurred(),
		"failed conversion should be in archive")

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("1 failed"))
	g.Expect(output).To(ContainSubstring("conversion failed"))
}

func TestMigrateSBIA_CounterMapping(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	legacyTOML := `title = "counter test"
content = "test content"
confidence = "A"
principle = "do the thing"
anti_pattern = "don't do the bad thing"
rationale = "because reasons"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
surfaced_count = 10
followed_count = 5
contradicted_count = 2
ignored_count = 3
irrelevant_count = 1
`
	memPath := filepath.Join(memoriesDir, "counters.toml")
	g.Expect(os.WriteFile(memPath, []byte(legacyTOML), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-sbia",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	converted, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(converted)
	// surfaced_count = 10, followed_count = 5 carried forward.
	g.Expect(content).To(ContainSubstring("surfaced_count = 10"))
	g.Expect(content).To(ContainSubstring("followed_count = 5"))
	// contradicted(2) + ignored(3) = not_followed(5).
	g.Expect(content).To(ContainSubstring("not_followed_count = 5"))
	// irrelevant_count = 1 carried forward.
	g.Expect(content).To(ContainSubstring("irrelevant_count = 1"))
}

func TestMigrateSBIA_ScopeMapping_NotProjectScoped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// generalizability = 3 → project_scoped = false
	legacyTOML := `title = "general memory"
content = "applies broadly"
confidence = "A"
principle = "do general thing"
anti_pattern = "bad general thing"
rationale = "general reasons"
project_slug = "my-project"
generalizability = 3
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`
	memPath := filepath.Join(memoriesDir, "general.toml")
	g.Expect(os.WriteFile(memPath, []byte(legacyTOML), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-sbia",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	converted, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(converted)
	g.Expect(content).To(ContainSubstring("project_scoped = false"))
}

func TestMigrateSBIA_ScopeMapping_ProjectScoped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// generalizability = 2 → project_scoped = true
	legacyTOML := `title = "scoped memory"
content = "project-specific"
confidence = "A"
principle = "do scoped thing"
anti_pattern = "bad scoped thing"
rationale = "project reasons"
project_slug = "my-project"
generalizability = 2
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`
	memPath := filepath.Join(memoriesDir, "scoped.toml")
	g.Expect(os.WriteFile(memPath, []byte(legacyTOML), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-sbia",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	converted, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(converted)
	g.Expect(content).To(ContainSubstring("project_scoped = true"))
	g.Expect(content).To(ContainSubstring(`project_slug = "my-project"`))
}

func TestMigrateSBIA_Summary(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	tierATOML := `title = "tier A"
content = "content"
confidence = "A"
principle = "principle"
anti_pattern = "anti"
rationale = "rationale"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`
	tierBTOML := `title = "tier B"
content = "content"
confidence = "B"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`

	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "a.toml"), []byte(tierATOML), 0o644,
	)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "b.toml"), []byte(tierBTOML), 0o644,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-sbia",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("1 converted"))
	g.Expect(output).To(ContainSubstring("1 archived"))
	g.Expect(output).To(ContainSubstring("0 failed"))
}

func TestMigrateSBIA_TierA_ConvertedViaConverter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	legacyTOML := `title = "avoid bare returns"
content = "Always wrap errors with context"
observation_type = "correction"
concepts = ["error-handling"]
keywords = ["errors", "wrapping"]
principle = "Use fmt.Errorf with %%w verb"
anti_pattern = "Returning bare errors without context"
rationale = "Context helps debugging"
project_slug = "engram"
generalizability = 4
confidence = "A"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-15T00:00:00Z"
surfaced_count = 5
followed_count = 3
contradicted_count = 1
ignored_count = 1
irrelevant_count = 2
last_surfaced_at = "2024-01-15T00:00:00Z"
`
	memPath := filepath.Join(memoriesDir, "avoid-bare-returns.toml")
	g.Expect(os.WriteFile(memPath, []byte(legacyTOML), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-sbia",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// The file should still exist (converted in place).
	converted, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(converted)
	// Should have SBIA fields from mock converter.
	g.Expect(content).To(ContainSubstring("situation"))
	g.Expect(content).To(ContainSubstring("behavior"))
	g.Expect(content).To(ContainSubstring("impact"))
	g.Expect(content).To(ContainSubstring("action"))

	// Should NOT have old fields.
	g.Expect(content).NotTo(ContainSubstring("anti_pattern"))
	g.Expect(content).NotTo(ContainSubstring("principle ="))
	g.Expect(content).NotTo(ContainSubstring("rationale"))
	g.Expect(content).NotTo(ContainSubstring("keywords"))

	// Summary output.
	g.Expect(stdout.String()).To(ContainSubstring("1 converted"))
}

func TestMigrateSBIA_TierB_Archived(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	legacyTOML := `title = "tier-b memory"
content = "some content"
confidence = "B"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`
	memPath := filepath.Join(memoriesDir, "tier-b.toml")
	g.Expect(os.WriteFile(memPath, []byte(legacyTOML), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-sbia",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Original file should be gone.
	_, statErr := os.Stat(memPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "tier B memory should be moved")

	// Should exist in archive.
	archivePath := filepath.Join(dataDir, "archive", "tier-b.toml")
	_, archiveStatErr := os.Stat(archivePath)
	g.Expect(archiveStatErr).NotTo(HaveOccurred(), "tier B memory should be in archive")

	g.Expect(stdout.String()).To(ContainSubstring("1 archived"))
}

func TestMigrateSBIA_TierC_Archived(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	legacyTOML := `title = "tier-c memory"
content = "low confidence content"
confidence = "C"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`
	memPath := filepath.Join(memoriesDir, "tier-c.toml")
	g.Expect(os.WriteFile(memPath, []byte(legacyTOML), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-sbia",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	_, statErr := os.Stat(memPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "tier C memory should be moved")

	archivePath := filepath.Join(dataDir, "archive", "tier-c.toml")
	_, archiveStatErr := os.Stat(archivePath)
	g.Expect(archiveStatErr).NotTo(HaveOccurred(), "tier C memory should be in archive")
}

func TestMigrateSBIA_WriteError_CountedAsFailed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	legacyTOML := `title = "tier-a"
content = "content"
confidence = "A"
principle = "principle"
anti_pattern = "anti"
rationale = "rationale"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`

	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "a.toml"), []byte(legacyTOML), 0o644,
	)).To(Succeed())

	var stdout bytes.Buffer

	deps := cli.MigrationDeps{
		ListDir:  os.ReadDir,
		ReadFile: os.ReadFile,
		WriteFile: func(_ string, _ []byte, _ os.FileMode) error {
			return errSimulatedWrite
		},
		MkdirAll:  os.MkdirAll,
		Rename:    os.Rename,
		Converter: &passthroughSBIAConverter{},
		Stdout:    &stdout,
	}

	ctx := context.Background()

	err := cli.ExecuteMigration(ctx, dataDir, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("1 failed"))
}

// unexported variables.
var (
	errConversionFailed = errors.New("simulated conversion failure")
	errSimulatedRename  = errors.New("simulated rename failure")
	errSimulatedWrite   = errors.New("simulated write failure")
)

// failingSBIAConverter always returns an error.
type failingSBIAConverter struct{}

func (c *failingSBIAConverter) Convert(
	_ context.Context,
	_ cli.LegacyMemoryRecord,
) (*memory.MemoryRecord, error) {
	return nil, errConversionFailed
}

// passthroughSBIAConverter returns a minimal valid record.
type passthroughSBIAConverter struct{}

func (c *passthroughSBIAConverter) Convert(
	_ context.Context,
	_ cli.LegacyMemoryRecord,
) (*memory.MemoryRecord, error) {
	return &memory.MemoryRecord{
		Situation: "test situation",
		Behavior:  "test behavior",
		Impact:    "test impact",
		Action:    "test action",
	}, nil
}
