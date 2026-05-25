package cli_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestOsLearnFS_ListIDs_BadVaultReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// vault is a file, not a dir; ReadDir on vault/Permanent → ENOTDIR (not IsNotExist).
	vault := filepath.Join(t.TempDir(), "file")
	g.Expect(os.WriteFile(vault, []byte("x"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsLearnFS()
	_, err := fs.ListIDs(vault)
	g.Expect(err).To(HaveOccurred())
}

func TestOsLearnFS_ListIDs_MissingSubdirsTolerated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// vault exists but neither Permanent nor MOCs subdirs.
	vault := t.TempDir()

	fs := cli.ExportNewOsLearnFS()
	got, err := fs.ListIDs(vault)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(BeEmpty())
}

func TestOsLearnFS_Lock_BadVaultReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := cli.ExportNewOsLearnFS()
	_, err := fs.Lock("/nonexistent/parent/that/does/not/exist")
	g.Expect(err).To(HaveOccurred())
}

func TestOsLearnFS_MkdirAll_CreatesNestedDirs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.MkdirAll(nested, 0o755)).To(Succeed())

	info, statErr := os.Stat(nested)
	g.Expect(statErr).NotTo(HaveOccurred())

	if statErr != nil {
		return
	}

	g.Expect(info.IsDir()).To(BeTrue())
}

func TestOsLearnFS_MkdirAll_FailsWhenParentIsFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := t.TempDir()
	filePath := filepath.Join(root, "f")
	g.Expect(os.WriteFile(filePath, []byte("x"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.MkdirAll(filepath.Join(filePath, "child"), 0o755)).
		To(MatchError(ContainSubstring("mkdir")))
}

func TestOsLearnFS_MkdirAll_IdempotentOnExistingDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.MkdirAll(dir, 0o755)).To(Succeed())
}

func TestOsLearnFS_StatDir_OnDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.StatDir(dir)).To(Succeed())
}

func TestOsLearnFS_StatDir_OnFileFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "file.txt")
	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.StatDir(path)).To(MatchError(ContainSubstring("not a directory")))
}

func TestOsLearnFS_StatDir_OnMissingPathFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.StatDir("/nonexistent/path/here")).To(HaveOccurred())
}

func TestOsLearnFS_WriteFileIfMissing_CreatesNew(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "f.txt")

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.WriteFileIfMissing(path, []byte("hello"), 0o600)).To(Succeed())

	got, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(got)).To(Equal("hello"))
}

func TestOsLearnFS_WriteFileIfMissing_FailsOnMissingDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.WriteFileIfMissing("/nonexistent/dir/f.txt", []byte("x"), 0o600)).To(HaveOccurred())
}

func TestOsLearnFS_WriteFileIfMissing_LeavesExistingUntouched(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "f.txt")
	g.Expect(os.WriteFile(path, []byte("original"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.WriteFileIfMissing(path, []byte("replacement"), 0o600)).To(Succeed())

	got, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(got)).To(Equal("original"))
}

func TestOsLearnFS_WriteNew_CreatesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "new.md")

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.WriteNew(path, []byte("hello"))).To(Succeed())

	data, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(Equal("hello"))
}

func TestOsLearnFS_WriteNew_ErrorsOnExisting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "existing.md")
	g.Expect(os.WriteFile(path, []byte("already"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.WriteNew(path, []byte("nope"))).To(HaveOccurred())
}

func TestOsLearnFS_WriteNew_OnBadDirectoryFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := cli.ExportNewOsLearnFS()
	g.Expect(fs.WriteNew("/nonexistent/dir/file.md", []byte("x"))).To(HaveOccurred())
}

func TestRunLearnFromFactArgs_BootstrapsMissingVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Vault dir does NOT exist; runLearn must bootstrap it before writing.
	vault := filepath.Join(t.TempDir(), "fresh-vault")

	args := cli.LearnFactArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "bootstrap-fact",
			Vault:    vault,
			Position: "top",
			Source:   "test",
		},
		Situation: "first run",
		Subject:   "engram",
		Predicate: "bootstraps",
		Object:    "the vault",
	}

	err := cli.ExportRunLearnFromFactArgs(context.Background(), args, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Bootstrap created Permanent/, MOCs/, and .obsidian/.
	for _, sub := range []string{"Permanent", "MOCs", ".obsidian"} {
		info, statErr := os.Stat(filepath.Join(vault, sub))
		g.Expect(statErr).NotTo(HaveOccurred())

		if statErr != nil {
			return
		}

		g.Expect(info.IsDir()).To(BeTrue())
	}

	// And the actual fact note landed.
	entries, readErr := os.ReadDir(filepath.Join(vault, "Permanent"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}

// runLearnFrom*Args use newOsLearnDeps() and call runLearn. Driving these
// with a real vault dir exercises the full struct-conversion + delegation path
// (which previously had 0% coverage).

func TestRunLearnFromFactArgs_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(Succeed())

	args := cli.LearnFactArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "fact-slug",
			Vault:    vault,
			Position: "top",
		},
		Situation: "running tests",
		Subject:   "subj",
		Predicate: "is",
		Object:    "obj",
	}

	err := cli.ExportRunLearnFromFactArgs(context.Background(), args, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// One file landed in Permanent/.
	entries, readErr := os.ReadDir(filepath.Join(vault, "Permanent"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}

func TestRunLearnFromFeedbackArgs_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(Succeed())

	args := cli.LearnFeedbackArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "feedback-slug",
			Vault:    vault,
			Position: "top",
		},
		Situation: "writing code",
		Behavior:  "no tests",
		Impact:    "regressions",
		Action:    "write tests",
	}

	err := cli.ExportRunLearnFromFeedbackArgs(context.Background(), args, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	entries, readErr := os.ReadDir(filepath.Join(vault, "Permanent"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}

func TestRunLearnFromMOCArgs_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(Succeed())

	args := cli.LearnMOCArgs{
		CommonLearnArgs: cli.CommonLearnArgs{
			Slug:     "moc-slug",
			Vault:    vault,
			Position: "top",
		},
		Topic: "engram",
	}

	err := cli.ExportRunLearnFromMOCArgs(context.Background(), args, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	entries, readErr := os.ReadDir(filepath.Join(vault, "MOCs"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}
