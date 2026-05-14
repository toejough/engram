package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestRunRecall_AlreadyReadRejectsBareBasename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	permDir := filepath.Join(vault, "Permanent")
	g.Expect(os.MkdirAll(permDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(permDir, "1.2026-01-01.a.md"), []byte("body"), 0o644)).
		To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{
		VaultPath:   vault,
		Follow:      []string{"Permanent/1.2026-01-01.a.md"},
		AlreadyRead: []string{"2.2026-01-02.b"},
	}, &stdout)

	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("--already-read"))
	g.Expect(err.Error()).To(ContainSubstring("2.2026-01-02.b"))
}

func TestRunRecall_AnchorsScanErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Pass a file path as the vault — ScanVault returns an error.
	tmp := t.TempDir()
	bogus := filepath.Join(tmp, "not-a-dir")
	g.Expect(os.WriteFile(bogus, []byte("x"), 0o600)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{
		VaultPath: bogus,
	}, &stdout)

	g.Expect(err).To(HaveOccurred())
}

func TestRunRecall_FollowRejectsBareBasename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	permDir := filepath.Join(vault, "Permanent")
	g.Expect(os.MkdirAll(permDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(permDir, "1.2026-01-01.a.md"), []byte("body"), 0o644)).
		To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{
		VaultPath: vault,
		Follow:    []string{"1.2026-01-01.a"},
	}, &stdout)

	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("--follow"))
	g.Expect(err.Error()).To(ContainSubstring("1.2026-01-01.a"))
}

func TestRunRecall_FollowReturnsExpansionMinusAlreadyRead(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	permDir := filepath.Join(vault, "Permanent")
	g.Expect(os.MkdirAll(permDir, 0o755)).To(Succeed())

	// a links to b and c (wikilink targets are basenames without .md).
	// b is already-read so only c should appear in output.
	g.Expect(os.WriteFile(
		filepath.Join(permDir, "1.2026-01-01.a.md"),
		[]byte("body with [[2.2026-01-02.b]] and [[3.2026-01-03.c]]"),
		0o644,
	)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(permDir, "2.2026-01-02.b.md"), []byte("b body"), 0o644)).
		To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(permDir, "3.2026-01-03.c.md"), []byte("c body"), 0o644)).
		To(Succeed())

	var stdout bytes.Buffer

	// Follow and AlreadyRead accept ONLY full relative paths
	// (Subdir/basename.md) — same shape as recall's stdout.
	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{
		VaultPath:   vault,
		Follow:      []string{"Permanent/1.2026-01-01.a.md"},
		AlreadyRead: []string{"Permanent/2.2026-01-02.b.md"},
	}, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Output is full relative paths so the caller can Read() directly
	// without guessing which subdirectory the note lives in.
	g.Expect(stdout.String()).To(Equal("Permanent/3.2026-01-03.c.md\n"))
}

func TestRunRecall_NoArgsEmitsAnchorsAsFullRelativePathsOneLinePerEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	mocsDir := filepath.Join(vault, "MOCs")
	g.Expect(os.MkdirAll(mocsDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(mocsDir, "1.2026-03-15.alpha.md"),
		[]byte("MOC content"),
		0o644,
	)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{
		VaultPath: vault,
	}, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Full relative path so the caller can Read() directly without
	// guessing which vault subdirectory the note lives in.
	g.Expect(stdout.String()).To(Equal("MOCs/1.2026-03-15.alpha.md\n"))
}

func TestRunRecall_NoVaultPathReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout bytes.Buffer

	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{}, &stdout)

	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("--vault required"))
}

func TestRunRecall_RecentReturnsMostRecentByDate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	permDir := filepath.Join(vault, "Permanent")
	g.Expect(os.MkdirAll(permDir, 0o755)).To(Succeed())

	for _, name := range []string{
		"1.2026-01-01.old.md",
		"2.2026-05-01.new.md",
		"3.2026-03-15.mid.md",
	} {
		g.Expect(os.WriteFile(filepath.Join(permDir, name), []byte("body"), 0o644)).To(Succeed())
	}

	var stdout bytes.Buffer

	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{
		VaultPath: vault,
		Recent:    true,
		Limit:     2,
	}, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Full relative paths; sorted newest first.
	g.Expect(stdout.String()).To(Equal("Permanent/2.2026-05-01.new.md\nPermanent/3.2026-03-15.mid.md\n"))
}

func TestRunRecall_RecentScanErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	bogus := filepath.Join(tmp, "not-a-dir")
	g.Expect(os.WriteFile(bogus, []byte("x"), 0o600)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRecallForTest(context.Background(), cli.RecallArgs{
		VaultPath: bogus,
		Recent:    true,
	}, &stdout)

	g.Expect(err).To(HaveOccurred())
}
