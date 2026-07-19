package cli_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestInvariant_K1_ConcurrentLearnNeverCollides locks invariant K1: the
// vault write-lock (flock on .luhmann.lock spanning id-compute→write, with
// O_EXCL on WriteNew as a backstop) guarantees concurrent `engram learn`
// never computes the same next Luhmann id and never overwrites a note.
//
// We drive the REAL locked write path: LearnDeps built from the production
// composition (real flock + real ListIDs + real O_EXCL WriteNew) against a
// real temp vault. N goroutines each write a top-level note. If the lock
// failed to span compute→write, two goroutines would read the same existing
// set, compute the same id, and either collide (O_EXCL error) or lose a
// write. We assert all N succeed, with N distinct leading ids and N distinct
// files.
func TestInvariant_K1_ConcurrentLearnNeverCollides(t *testing.T) {
	t.Parallel()

	counts := []int{2, 5, 10, 20}

	for _, workers := range counts {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Each subtest gets its own temp vault — no shared mutable state.
			vault := t.TempDir()
			g.Expect(os.MkdirAll(vault, 0o755)).To(Succeed())

			deps := k1RealLockDeps(vault)

			paths := make([]string, workers)
			errs := make([]error, workers)

			var wg sync.WaitGroup

			wg.Add(workers)

			for i := range workers {
				go func() {
					defer wg.Done()

					var out strings.Builder

					args := cli.LearnArgs{
						Type:      "fact",
						Slug:      fmt.Sprintf("k1-worker-%d", i),
						Vault:     vault,
						Position:  "top",
						Source:    "k1 stress",
						Situation: fmt.Sprintf("concurrent learn worker %d", i),
						Subject:   "the vault write-lock",
						Predicate: "prevents",
						Object:    "duplicate luhmann ids under concurrency",
					}

					errs[i] = cli.ExportRunLearn(context.Background(), args, deps, &out)
					paths[i] = strings.TrimSpace(out.String())
				}()
			}

			wg.Wait()

			for i, err := range errs {
				g.Expect(err).NotTo(HaveOccurred(), "worker %d failed", i)
			}

			ids := make(map[string]bool, workers)
			files := make(map[string]bool, workers)

			for i, path := range paths {
				g.Expect(path).NotTo(BeEmpty(), "worker %d wrote no path", i)

				base := filepath.Base(path)
				files[base] = true

				id, ok := cli.ExportExtractLuhmannFromFilename(base)
				g.Expect(ok).To(BeTrue(), "worker %d path %q has no luhmann id", i, path)

				ids[id] = true
			}

			g.Expect(ids).To(HaveLen(workers), "expected %d distinct luhmann ids, got %v", workers, ids)
			g.Expect(files).To(HaveLen(workers), "expected %d distinct files, got %v", workers, files)

			// Backstop: the files actually exist on disk (no lost write).
			entries, readErr := os.ReadDir(vault)
			g.Expect(readErr).NotTo(HaveOccurred())

			if readErr != nil {
				return
			}

			notes := 0

			for _, entry := range entries {
				// vault root also holds the luhmann lock + .obsidian; count notes only
				if strings.HasSuffix(entry.Name(), ".md") {
					notes++
				}
			}

			g.Expect(notes).To(Equal(workers), "expected %d note files on disk", workers)
		})
	}
}

// unexported variables.
var (
	errK1VaultMissing = errors.New("k1: vault should already exist")
)

// k1RealLockDeps wires LearnDeps through the PRODUCTION composition
// (newLearnDeps) over the internally-composed primFS EdgeFS and primLocker
// FileLocker with real OS primitives — the exact flock + exclusive-create
// (EdgeFS.WriteFileExcl over the base WriteFileExcl primitive) path the
// shipped binary builds via cli.NewDeps. Embed is nil (realFSDepsForTest
// forces it) so auto-embed skips; InitVault errors because the caller
// pre-creates the vault.
func k1RealLockDeps(vault string) cli.LearnDeps {
	deps := cli.ExportNewLearnDeps(realFSDepsForTest())

	deps.InitVault = func(string) error {
		return fmt.Errorf("%w: %s", errK1VaultMissing, vault)
	}

	return deps
}
