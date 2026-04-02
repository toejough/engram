package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestMigrateSlugsFlags_BuildsCorrectArgs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	flags := cli.MigrateSlugsFlags(cli.MigrateSlugsArgs{
		DataDir: "/data",
		Slug:    "my-project",
		Apply:   true,
	})

	g.Expect(flags).To(ContainElement("--data-dir"))
	g.Expect(flags).To(ContainElement("/data"))
	g.Expect(flags).To(ContainElement("--slug"))
	g.Expect(flags).To(ContainElement("my-project"))
	g.Expect(flags).To(ContainElement("--apply"))
}

func TestMigrateSlugs_BackfillsEmptySlugs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Write two memories without project_slug.
	for _, name := range []string{"mem-a.toml", "mem-b.toml"} {
		path := filepath.Join(memoriesDir, name)
		g.Expect(os.WriteFile(path, []byte(testMemoryTOML), 0o644)).To(Succeed())
	}

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-slugs",
			"--data-dir", dataDir,
			"--slug", "test-project",
			"--apply",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	lister := memory.NewLister()
	records, listErr := lister.ListAll(memoriesDir)
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(records).To(HaveLen(2))

	for _, stored := range records {
		g.Expect(stored.Record.ProjectSlug).To(Equal("test-project"),
			"memory %s should have project_slug backfilled", stored.Path)
	}
}

func TestMigrateSlugs_DryRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	memPath := filepath.Join(memoriesDir, "dry.toml")
	g.Expect(os.WriteFile(memPath, []byte(testMemoryTOML), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-slugs",
			"--data-dir", dataDir,
			"--slug", "dry-run-slug",
			// No --apply flag → dry-run mode.
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Output should mention "would set".
	g.Expect(stdout.String()).To(ContainSubstring("would set"))

	// Memory file must not be modified.
	lister := memory.NewLister()
	records, listErr := lister.ListAll(memoriesDir)
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Record.ProjectSlug).To(BeEmpty(),
		"dry-run must not write project_slug")
}

func TestMigrateSlugs_SkipsPopulated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Write one memory with a slug, one without.
	alreadySluggedTOML := testMemoryTOML + `project_slug = "existing-slug"` + "\n"
	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "slugged.toml"), []byte(alreadySluggedTOML), 0o644,
	)).To(Succeed())

	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "empty.toml"), []byte(testMemoryTOML), 0o644,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "migrate-slugs",
			"--data-dir", dataDir,
			"--slug", "new-slug",
			"--apply",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	lister := memory.NewLister()
	records, listErr := lister.ListAll(memoriesDir)
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(records).To(HaveLen(2))

	slugMap := make(map[string]string, len(records))
	for _, stored := range records {
		slugMap[filepath.Base(stored.Path)] = stored.Record.ProjectSlug
	}

	g.Expect(slugMap["slugged.toml"]).To(Equal("existing-slug"),
		"pre-populated slug must not be overwritten")
	g.Expect(slugMap["empty.toml"]).To(Equal("new-slug"),
		"empty slug must be backfilled")
}

// unexported constants.
const (
	testMemoryTOML = `title = "test memory"
content = "some content"
confidence = "high"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`
)
