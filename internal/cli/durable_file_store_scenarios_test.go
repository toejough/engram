package cli_test

import (
	"io"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792b3c1a8_EdgeFS_WriteFileAtomicExhaustedTempNamesGiveBoundedError verifies 792b3c1a8 (Exhausted temp names
// give a bounded error): when every candidate temp name already exists, the atomic replace tries a bounded number of
// distinct candidates, then returns an error that says no unique temp name was available and names the target, and
// it neither sets permissions, renames nor removes anything (no temp was ever created by the write).
func Test792b3c1a8_EdgeFS_WriteFileAtomicExhaustedTempNamesGiveBoundedError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	target := filepath.Join("some", "dir", "note.md")
	tried := make([]string, 0, danceMaxAttempts)

	prims := cli.Primitives{
		FS: cli.FSPrims{
			OpenFileExcl: func(path string, _ fs.FileMode) (io.WriteCloser, error) {
				tried = append(tried, path)

				return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
			},
			Chmod: func(string, fs.FileMode) error {
				t.Error("no temp was created, so no permissions may be set")

				return nil
			},
			Rename: func(string, string) error {
				t.Error("no temp was created, so nothing may be renamed onto the target")

				return nil
			},
			Remove: func(string) error {
				t.Error("every colliding candidate belongs to someone else and must not be removed")

				return nil
			},
		},
		Proc: cli.ProcPrims{Now: func() time.Time { return time.Unix(0, fakeDanceNanos) }},
	}

	err := fsFromPrims(prims).WriteFileAtomic(target, []byte("new"), atomicPerm)

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("no unique temp name available")))
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring(target)), "the error names the target path")
	g.Expect(err).To(gomega.MatchError(fs.ErrExist), "the last collision stays in the error chain")
	g.Expect(tried).To(gomega.HaveLen(danceMaxAttempts), "the retry loop is bounded at the spec'd attempts")

	distinct := make(map[string]struct{}, len(tried))
	for _, candidate := range tried {
		distinct[candidate] = struct{}{}
	}

	g.Expect(distinct).To(gomega.HaveLen(danceMaxAttempts), "each attempt tries a fresh candidate name")
}
