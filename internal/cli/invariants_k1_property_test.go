package cli_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestInvariant_K1_ConcurrentLearnNeverCollides locks invariant K1: the
// vault write-lock (flock on .luhmann.lock spanning id-compute→write, with
// O_EXCL on WriteNew as a backstop) guarantees concurrent `engram learn`
// never computes the same next Luhmann id and never overwrites a note.
//
// We drive the REAL locked write path: LearnDeps built from the production
// osLearnFS (real flock + real ListIDs + real O_EXCL WriteNew) against a
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
			g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o755)).To(Succeed())

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
			entries, readErr := os.ReadDir(filepath.Join(vault, "Permanent"))
			g.Expect(readErr).NotTo(HaveOccurred())

			if readErr != nil {
				return
			}

			g.Expect(entries).To(HaveLen(workers), "expected %d note files on disk", workers)
		})
	}
}

// k1RealLockDeps wires LearnDeps to the production osLearnFS so the test
// exercises the real flock + real O_EXCL write path. The Permanent dir is
// pre-created by the caller so StatDir succeeds and InitVault never fires.
// Embedder is nil to skip the auto-embed step (irrelevant to the lock).
func k1RealLockDeps(vault string) cli.LearnDeps {
	osFS := cli.ExportNewOsLearnFS()

	return cli.LearnDeps{
		Now:      time.Now,
		Getenv:   os.Getenv,
		StatDir:  osFS.StatDir,
		ListIDs:  osFS.ListIDs,
		Lock:     osFS.Lock,
		WriteNew: osFS.WriteNew,
		InitVault: func(string) error {
			return fmt.Errorf("k1: vault %s should already exist", vault)
		},
	}
}
