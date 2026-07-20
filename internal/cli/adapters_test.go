package cli_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestNewLearnDeps_ListIDs_ReturnsRootNotesOnly exercises listIDsFromFS
// (the #700 T3 replacement for the deleted osLearnFS.ListIDs) through the
// production Deps composition — same flat-root traversal, same MOCs/
// subdirectory + non-luhmann filename skips.
func TestNewLearnDeps_ListIDs_ReturnsRootNotesOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o700)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "1.2026-05-09.foo.md"), nil, 0o600)).
		To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "1a.2026-05-09.bar.md"), nil, 0o600)).
		To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "MOCs", "5.2026-05-09.moc.md"), nil, 0o600)).
		To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "README.md"), nil, 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(newTestDeps(io.Discard, io.Discard))
	got, err := deps.ListIDs(vault)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// flat vault: subdirectories (including legacy MOCs/) are ignored
	g.Expect(got).To(ConsistOf("1", "1a"))
}

func TestOsLearnFS_Lock_ExclusiveAcrossSecondAcquisition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	vault := t.TempDir()

	fs := cli.ExportNewOsLearnFS()
	release1, err := fs.Lock(vault)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	done := make(chan struct{})

	go func() {
		release2, err2 := fs.Lock(vault)
		g.Expect(err2).NotTo(HaveOccurred())

		if release2 != nil {
			release2()
		}

		close(done)
	}()

	select {
	case <-done:
		t.Fatal("second Lock should not have succeeded while first holds")
	case <-time.After(100 * time.Millisecond):
	}

	release1()
	<-done
}
