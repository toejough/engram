package cli_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestOsPromoteFS_ListIDs_BadVaultReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// vault is a file, not a dir; ReadDir on vault/Permanent → ENOTDIR (not IsNotExist).
	vault := filepath.Join(t.TempDir(), "file")
	g.Expect(os.WriteFile(vault, []byte("x"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsPromoteFS()
	_, err := fs.ListIDs(vault)
	g.Expect(err).To(HaveOccurred())
}

func TestOsPromoteFS_ListIDs_MissingSubdirsTolerated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// vault exists but neither Permanent nor MOCs subdirs.
	vault := t.TempDir()

	fs := cli.ExportNewOsPromoteFS()
	got, err := fs.ListIDs(vault)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(BeEmpty())
}

func TestOsPromoteFS_Lock_BadVaultReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := cli.ExportNewOsPromoteFS()
	_, err := fs.Lock("/nonexistent/parent/that/does/not/exist")
	g.Expect(err).To(HaveOccurred())
}

func TestOsPromoteFS_StatDir_OnDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	fs := cli.ExportNewOsPromoteFS()
	g.Expect(fs.StatDir(dir)).To(Succeed())
}

func TestOsPromoteFS_StatDir_OnFileFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "file.txt")
	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsPromoteFS()
	g.Expect(fs.StatDir(path)).To(MatchError(ContainSubstring("not a directory")))
}

func TestOsPromoteFS_StatDir_OnMissingPathFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := cli.ExportNewOsPromoteFS()
	g.Expect(fs.StatDir("/nonexistent/path/here")).To(HaveOccurred())
}

func TestOsPromoteFS_WriteNew_CreatesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "new.md")

	fs := cli.ExportNewOsPromoteFS()
	g.Expect(fs.WriteNew(path, []byte("hello"))).To(Succeed())

	data, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(Equal("hello"))
}

func TestOsPromoteFS_WriteNew_ErrorsOnExisting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "existing.md")
	g.Expect(os.WriteFile(path, []byte("already"), 0o600)).To(Succeed())

	fs := cli.ExportNewOsPromoteFS()
	g.Expect(fs.WriteNew(path, []byte("nope"))).To(HaveOccurred())
}

func TestOsPromoteFS_WriteNew_OnBadDirectoryFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := cli.ExportNewOsPromoteFS()
	g.Expect(fs.WriteNew("/nonexistent/dir/file.md", []byte("x"))).To(HaveOccurred())
}

// runPromoteFrom*Args use newOsPromoteDeps() and call runPromote. Driving these
// with a real vault dir exercises the full struct-conversion + delegation path
// (which previously had 0% coverage).

func TestRunPromoteFromFactArgs_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(Succeed())

	args := cli.PromoteFactArgs{
		CommonPromoteArgs: cli.CommonPromoteArgs{
			Slug:     "fact-slug",
			Vault:    vault,
			Relation: "top",
		},
		Situation: "running tests",
		Subject:   "subj",
		Predicate: "is",
		Object:    "obj",
	}

	err := cli.ExportRunPromoteFromFactArgs(context.Background(), args, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// One file landed in Permanent/.
	entries, readErr := os.ReadDir(filepath.Join(vault, "Permanent"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}

func TestRunPromoteFromFeedbackArgs_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(Succeed())

	args := cli.PromoteFeedbackArgs{
		CommonPromoteArgs: cli.CommonPromoteArgs{
			Slug:     "feedback-slug",
			Vault:    vault,
			Relation: "top",
		},
		Situation: "writing code",
		Behavior:  "no tests",
		Impact:    "regressions",
		Action:    "write tests",
	}

	err := cli.ExportRunPromoteFromFeedbackArgs(context.Background(), args, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	entries, readErr := os.ReadDir(filepath.Join(vault, "Permanent"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}

func TestRunPromoteFromMOCArgs_WritesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(Succeed())

	args := cli.PromoteMOCArgs{
		CommonPromoteArgs: cli.CommonPromoteArgs{
			Slug:     "moc-slug",
			Vault:    vault,
			Relation: "top",
		},
		Topic: "engram",
	}

	err := cli.ExportRunPromoteFromMOCArgs(context.Background(), args, io.Discard)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	entries, readErr := os.ReadDir(filepath.Join(vault, "MOCs"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).NotTo(BeEmpty())
}
