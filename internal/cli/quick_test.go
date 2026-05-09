package cli_test

import (
	"errors"
	ioFs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestFleetingPath_BuildsExpectedPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	when := time.Date(2026, time.May, 9, 17, 0, 0, 0, time.UTC)
	got := cli.ExportFleetingPath("/vault", "my-tag", when)
	g.Expect(got).To(Equal("/vault/Fleeting/2026-05-09.my-tag.md"))
}

func TestOsQuickFS_StatDir_ErrorsOnMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	fs := cli.ExportNewOsQuickFS()
	err := fs.StatDir(filepath.Join(t.TempDir(), "missing"))
	g.Expect(err).To(HaveOccurred())
}

func TestOsQuickFS_StatDir_PassesOnExisting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	fs := cli.ExportNewOsQuickFS()
	g.Expect(fs.StatDir(t.TempDir())).To(Succeed())
}

func TestOsQuickFS_WriteNew_CreatesNewFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "new.md")
	fs := cli.ExportNewOsQuickFS()
	g.Expect(fs.WriteNew(path, []byte("hello"))).To(Succeed())

	got, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(got)).To(Equal("hello"))
}

func TestOsQuickFS_WriteNew_ErrorsOnExisting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.md")
	g.Expect(os.WriteFile(path, []byte("old"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsQuickFS()
	err := fs.WriteNew(path, []byte("new"))
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(errors.Is(err, ioFs.ErrExist)).To(BeTrue())
}

func TestRequireFleetingDir_ErrorsWhenStatFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	statFail := func(string) error { return errors.New("not found") }
	err := cli.ExportRequireFleetingDir("/vault", statFail)
	g.Expect(err).To(MatchError(ContainSubstring("Fleeting")))
}

func TestRequireFleetingDir_PassesWhenStatSucceeds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	statOK := func(string) error { return nil }
	g.Expect(cli.ExportRequireFleetingDir("/vault", statOK)).To(Succeed())
}

func TestResolveContent_ErrorsWhenBoth(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	_, err := cli.ExportResolveContent("flag body", strings.NewReader("stdin body"))
	g.Expect(err).To(MatchError(ContainSubstring("content")))
}

func TestResolveContent_ErrorsWhenNeither(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	_, err := cli.ExportResolveContent("", strings.NewReader(""))
	g.Expect(err).To(MatchError(ContainSubstring("content")))
}

func TestResolveContent_FlagOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	got, err := cli.ExportResolveContent("hello body", strings.NewReader(""))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("hello body"))
}

func TestResolveContent_StdinOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	got, err := cli.ExportResolveContent("", strings.NewReader("hello stdin"))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("hello stdin"))
}

func TestResolveVault_ErrorsWhenNeitherSet(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	getenv := func(string) string { return "" }
	_, err := cli.ExportResolveVault("", getenv)
	g.Expect(err).To(MatchError(ContainSubstring("vault")))
}

func TestResolveVault_FallsBackToEnv(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	getenv := func(name string) string {
		if name == "ENGRAM_VAULT_DIR" {
			return "/from/env"
		}

		return ""
	}
	got, err := cli.ExportResolveVault("", getenv)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("/from/env"))
}

func TestResolveVault_FlagWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	getenv := func(string) string { return "/from/env" }
	got, err := cli.ExportResolveVault("/from/flag", getenv)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("/from/flag"))
}

func TestRunQuick_ErrorsWhenWriteNewReportsExist(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	deps := cli.QuickDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 17, 0, 0, 0, time.UTC) },
		Stdin:   strings.NewReader(""),
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		WriteNew: func(string, []byte) error {
			return ioFs.ErrExist
		},
	}
	args := cli.QuickArgs{Slug: "tag", Content: "body", Vault: "/vault"}
	err := cli.ExportRunQuick(t.Context(), args, deps, &strings.Builder{})
	g.Expect(err).To(MatchError(ContainSubstring("exists")))
}

func TestRunQuick_HappyPath_WritesExpectedFileAndPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var (
		gotPath string
		gotData []byte
	)

	deps := cli.QuickDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 17, 0, 0, 0, time.UTC) },
		Stdin:   strings.NewReader(""),
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		WriteNew: func(path string, data []byte) error {
			gotPath = path
			gotData = data

			return nil
		},
	}
	args := cli.QuickArgs{
		Slug:    "test-tag",
		Content: "# tag\n\nbody.\n",
		Vault:   "/vault",
	}

	var stdout strings.Builder

	err := cli.ExportRunQuick(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(gotPath).To(Equal("/vault/Fleeting/2026-05-09.test-tag.md"))
	g.Expect(string(gotData)).To(Equal("# tag\n\nbody.\n"))
	g.Expect(stdout.String()).To(ContainSubstring("/vault/Fleeting/2026-05-09.test-tag.md"))
}

func TestRunQuick_PropagatesSlugValidationError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	deps := cli.QuickDeps{
		Now:      time.Now,
		Stdin:    strings.NewReader(""),
		Getenv:   func(string) string { return "" },
		StatDir:  func(string) error { return nil },
		WriteNew: func(string, []byte) error { return nil },
	}
	args := cli.QuickArgs{Slug: "Bad Slug", Content: "body", Vault: "/vault"}
	err := cli.ExportRunQuick(t.Context(), args, deps, &strings.Builder{})
	g.Expect(err).To(MatchError(ContainSubstring("slug")))
}

func TestRunQuick_PropagatesWriteError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	deps := cli.QuickDeps{
		Now:     func() time.Time { return time.Date(2026, time.May, 9, 17, 0, 0, 0, time.UTC) },
		Stdin:   strings.NewReader(""),
		Getenv:  func(string) string { return "" },
		StatDir: func(string) error { return nil },
		WriteNew: func(string, []byte) error {
			return errors.New("disk full")
		},
	}
	args := cli.QuickArgs{Slug: "tag", Content: "body", Vault: "/vault"}
	err := cli.ExportRunQuick(t.Context(), args, deps, &strings.Builder{})
	g.Expect(err).To(MatchError(ContainSubstring("writing")))
}

func TestValidateSlug_AcceptsKebabCase(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	g.Expect(cli.ExportValidateSlug("graph-connectedness-recall-axis")).To(Succeed())
	g.Expect(cli.ExportValidateSlug("a")).To(Succeed())
	g.Expect(cli.ExportValidateSlug("note-1")).To(Succeed())
}

func TestValidateSlug_RejectsInvalid(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	g.Expect(cli.ExportValidateSlug("")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("Has-Caps")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("has space")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("dot.in.it")).To(MatchError(ContainSubstring("slug")))
	g.Expect(cli.ExportValidateSlug("under_score")).To(MatchError(ContainSubstring("slug")))
}
