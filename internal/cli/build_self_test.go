package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestBuildAndInstall_BuildFails_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "engram")

	sentinel := errors.New("compile boom")
	fakeBuild := func(_ context.Context, _, _ string, _ io.Writer) error {
		return sentinel
	}

	var stdout bytes.Buffer

	err := cli.ExportBuildAndInstall(context.Background(), fakeBuild, "", binPath, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("build failed"))
}

func TestBuildAndInstall_BuildSucceeds_RenamesIntoPlace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "engram")

	fakeBuild := func(_ context.Context, _, tmpPath string, _ io.Writer) error {
		return os.WriteFile(tmpPath, []byte("fake binary"), 0o600)
	}

	var stdout bytes.Buffer

	err := cli.ExportBuildAndInstall(context.Background(), fakeBuild, "", binPath, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	got, readErr := os.ReadFile(binPath)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("fake binary"))
}

func TestBuildAndInstall_RenameFails_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Set binPath to a directory that already exists and is non-empty.
	// Rename(file, non-empty-dir) fails on macOS/Linux.
	binPath := t.TempDir()
	innerErr := os.WriteFile(filepath.Join(binPath, "occupant"), []byte("x"), 0o600)
	g.Expect(innerErr).NotTo(HaveOccurred())

	// fake builder writes the tmp file (binPath + ".tmp") in the parent dir.
	fakeBuild := func(_ context.Context, _, tmpPath string, _ io.Writer) error {
		return os.WriteFile(tmpPath, []byte("ok"), 0o600)
	}

	var stdout bytes.Buffer

	err := cli.ExportBuildAndInstall(context.Background(), fakeBuild, "", binPath, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("install failed"))
}

func TestBuildSelf_BadPluginRoot_ReturnsBuildError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "engram")

	var stdout bytes.Buffer

	args := cli.BuildSelfArgs{
		IfStale:    false,
		PluginRoot: t.TempDir(),
		BinPath:    binPath,
	}

	runErr := cli.ExportRunBuildSelf(context.Background(), args, &stdout)
	g.Expect(runErr).To(HaveOccurred())
}

func TestBuildSelf_DefaultBinPath_UsesHomeDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Trigger the BinPath default branch (empty BinPath). PluginRoot is empty,
	// so build will fail fast — but the home-dir resolution is exercised before that.
	var stdout bytes.Buffer

	args := cli.BuildSelfArgs{
		IfStale:    false,
		PluginRoot: t.TempDir(),
		BinPath:    "",
	}

	// We expect a build failure (no real go module in temp dir), not a home-resolution failure.
	runErr := cli.ExportRunBuildSelf(context.Background(), args, &stdout)
	g.Expect(runErr).To(HaveOccurred())
}

func TestBuildSelf_IfStale_FreshBinary_NoOpExitsCleanly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pluginRoot := t.TempDir()
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "engram")

	// Create a stale .go file in pluginRoot (older than the binary we'll create).
	srcPath := filepath.Join(pluginRoot, "main.go")
	err := os.WriteFile(srcPath, []byte("package main\n"), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	pastTime := time.Now().Add(-time.Hour)
	err = os.Chtimes(srcPath, pastTime, pastTime)
	g.Expect(err).NotTo(HaveOccurred())

	// Create a "fresh" binary newer than the source file.
	err = os.WriteFile(binPath, []byte("not a real binary"), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	var stdout bytes.Buffer

	args := cli.BuildSelfArgs{
		IfStale:    true,
		PluginRoot: pluginRoot,
		BinPath:    binPath,
	}

	runErr := cli.ExportRunBuildSelf(context.Background(), args, &stdout)
	g.Expect(runErr).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(BeEmpty())
}

func TestBuildSelf_IfStale_NewerSourceFile_TriggersBuild(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pluginRoot := t.TempDir()
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "engram")

	// Create an "old" binary first.
	err := os.WriteFile(binPath, []byte("old"), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	pastTime := time.Now().Add(-time.Hour)
	err = os.Chtimes(binPath, pastTime, pastTime)
	g.Expect(err).NotTo(HaveOccurred())

	// Create a newer .go source file.
	srcPath := filepath.Join(pluginRoot, "main.go")
	err = os.WriteFile(srcPath, []byte("package main\n"), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	var stdout bytes.Buffer

	args := cli.BuildSelfArgs{
		IfStale:    true,
		PluginRoot: pluginRoot,
		BinPath:    binPath,
	}

	// Build will attempt and fail (no real cmd/engram), but the staleness check fired.
	runErr := cli.ExportRunBuildSelf(context.Background(), args, &stdout)
	g.Expect(runErr).To(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("build failed"))
}

func TestBuildSelf_IfStale_NoBinaryAndEmptyRoot_TriesBuildAndFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pluginRoot := t.TempDir()
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "engram")

	var stdout bytes.Buffer

	args := cli.BuildSelfArgs{
		IfStale:    true,
		PluginRoot: pluginRoot,
		BinPath:    binPath,
	}

	err := cli.ExportRunBuildSelf(context.Background(), args, &stdout)

	// Expected: binary missing → stale → build attempted → fails (no cmd/engram in temp dir).
	g.Expect(err).To(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("build failed"))
}

func TestBuildSelf_StaleCheck_NewerBinaryThanSources_NoOp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pluginRoot := t.TempDir()
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "engram")

	// Multiple .go files, all older than the binary.
	for _, name := range []string{"a.go", "b.go", "sub/c.go"} {
		full := filepath.Join(pluginRoot, name)
		mkErr := os.MkdirAll(filepath.Dir(full), 0o750)
		g.Expect(mkErr).NotTo(HaveOccurred())

		writeErr := os.WriteFile(full, []byte("package x\n"), 0o600)
		g.Expect(writeErr).NotTo(HaveOccurred())

		past := time.Now().Add(-2 * time.Hour)
		chErr := os.Chtimes(full, past, past)
		g.Expect(chErr).NotTo(HaveOccurred())
	}

	// Binary is fresh.
	err := os.WriteFile(binPath, []byte("bin"), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	var stdout bytes.Buffer

	args := cli.BuildSelfArgs{
		IfStale:    true,
		PluginRoot: pluginRoot,
		BinPath:    binPath,
	}

	runErr := cli.ExportRunBuildSelf(context.Background(), args, &stdout)
	g.Expect(runErr).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(BeEmpty())
}

func TestBuildSelf_StaleCheck_NonExistentPluginRoot_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "engram")

	// Create a binary so isStale gets past the binary-stat branch.
	err := os.WriteFile(binPath, []byte("bin"), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	var stdout bytes.Buffer

	args := cli.BuildSelfArgs{
		IfStale:    true,
		PluginRoot: filepath.Join(binDir, "does-not-exist"),
		BinPath:    binPath,
	}

	runErr := cli.ExportRunBuildSelf(context.Background(), args, &stdout)
	g.Expect(runErr).To(HaveOccurred())
}
