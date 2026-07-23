package cli_test

import (
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestEdgeFS_PreservesSentinelChainsThroughWrapping(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := fsFromPrims(cli.Primitives{FS: cli.FSPrims{
		ReadFile: func(string) ([]byte, error) {
			return nil, &fs.PathError{Op: "open", Path: "x", Err: fs.ErrNotExist}
		},
	}})

	_, err := fsys.ReadFile("x")
	g.Expect(err).To(gomega.MatchError(fs.ErrNotExist), "%w wrapping must preserve errors.Is chains")
	g.Expect(err.Error()).To(gomega.ContainSubstring("x"), "wrap must add path context")
}

func TestEdgeFS_WrapsEveryPrimitiveErrorWithContext(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		call    func(cli.EdgeFS) error
		context string
	}{
		{"MkdirAll", func(f cli.EdgeFS) error { return f.MkdirAll("d", atomicPerm) }, "mkdir d"},
		{"MkdirTemp", func(f cli.EdgeFS) error {
			_, err := f.MkdirTemp("d", "p")

			return err
		}, "mkdir temp in d"},
		{"ReadDir", func(f cli.EdgeFS) error {
			_, err := f.ReadDir("d")

			return err
		}, "read dir d"},
		{"Remove", func(f cli.EdgeFS) error { return f.Remove("x") }, "remove x"},
		{"RemoveAll", func(f cli.EdgeFS) error { return f.RemoveAll("x") }, "remove all x"},
		{"Rename", func(f cli.EdgeFS) error { return f.Rename("a", "b") }, "rename a -> b"},
		{"Stat", func(f cli.EdgeFS) error {
			_, err := f.Stat("x")

			return err
		}, "stat x"},
		{"WalkDir", func(f cli.EdgeFS) error { return f.WalkDir("d", nil) }, "walk d"},
		{"WriteFile", func(f cli.EdgeFS) error { return f.WriteFile("x", []byte("v"), atomicPerm) }, "write x"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)

			boom := errors.New("boom")
			err := testCase.call(fsFromPrims(failingPrims(boom)))
			g.Expect(err).To(gomega.MatchError(boom), "the wrap must preserve the error chain")
			g.Expect(err.Error()).To(gomega.ContainSubstring(testCase.context), "the wrap must add context")
		})
	}
}

func TestEdgeFS_WriteFileAtomicFailuresRemoveTemp(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")

	t.Run("rename failure removes the created temp", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var created string

		removed := make([]string, 0, 1)
		prims := cli.Primitives{
			FS: cli.FSPrims{
				OpenFileExcl: func(path string, _ fs.FileMode) (io.WriteCloser, error) {
					created = path

					return &mockWriteCloser{}, nil
				},
				Chmod:  func(string, fs.FileMode) error { return nil },
				Rename: func(string, string) error { return boom },
				Remove: func(path string) error {
					removed = append(removed, path)

					return nil
				},
			},
			Proc: cli.ProcPrims{Now: func() time.Time { return time.Unix(0, fakeDanceNanos) }},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(err.Error()).To(gomega.ContainSubstring("rename"))
		g.Expect(removed).To(gomega.Equal([]string{created}),
			"a failed dance must remove the temp file it created")
	})

	t.Run("chmod failure removes the created temp", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var created string

		removed := make([]string, 0, 1)
		prims := cli.Primitives{
			FS: cli.FSPrims{
				OpenFileExcl: func(path string, _ fs.FileMode) (io.WriteCloser, error) {
					created = path

					return &mockWriteCloser{}, nil
				},
				Chmod: func(string, fs.FileMode) error { return boom },
				Remove: func(path string) error {
					removed = append(removed, path)

					return nil
				},
			},
			Proc: cli.ProcPrims{Now: func() time.Time { return time.Unix(0, fakeDanceNanos) }},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(err.Error()).To(gomega.ContainSubstring("chmod"))
		g.Expect(removed).To(gomega.Equal([]string{created}),
			"a failed dance must remove the temp file it created")
	})

	t.Run("exclusive-create failure aborts with nothing to clean", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		prims := cli.Primitives{
			FS: cli.FSPrims{
				OpenFileExcl: func(string, fs.FileMode) (io.WriteCloser, error) { return nil, boom },
				Remove: func(string) error {
					t.Error("nothing was created, so nothing may be removed")

					return nil
				},
			},
			Proc: cli.ProcPrims{Now: func() time.Time { return time.Unix(0, fakeDanceNanos) }},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(err.Error()).To(gomega.ContainSubstring("create temp"))
	})
}

func TestEdgeFS_WriteFileAtomicHappyPathDance(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	calls := &callRecorder{}
	target := filepath.Join("some", "dir", "note.md")

	var capturedData []byte

	fsys := fsFromPrims(cli.Primitives{
		FS: cli.FSPrims{
			OpenFileExcl: func(path string, perm fs.FileMode) (io.WriteCloser, error) {
				g.Expect(filepath.Dir(path)).To(gomega.Equal(filepath.Join("some", "dir")),
					"temp must be created in the target's dir — same-directory rename is the ADR-0013 primitive")
				g.Expect(filepath.Base(path)).To(gomega.Equal(".note.md.tmp-12345-0"),
					"candidate names derive from target base + clock nanos + attempt counter (P-4)")
				g.Expect(perm).To(gomega.Equal(atomicPerm), "the target perm reaches the open")
				calls.add("writeexcl " + filepath.Base(path))

				return &mockWriteCloser{
					writeFunc: func(data []byte) (int, error) {
						capturedData = append(capturedData, data...)
						return len(data), nil
					},
				}, nil
			},
			Chmod: func(path string, perm fs.FileMode) error {
				g.Expect(perm).To(gomega.Equal(atomicPerm),
					"chmod must force the EXACT target perm regardless of umask")
				calls.add("chmod " + filepath.Base(path))

				return nil
			},
			Rename: func(oldPath, newPath string) error {
				calls.add("rename " + filepath.Base(oldPath) + "->" + filepath.Base(newPath))

				return nil
			},
			Remove: func(path string) error {
				calls.add("remove " + filepath.Base(path))

				return nil
			},
		},
		Proc: cli.ProcPrims{Now: func() time.Time { return time.Unix(0, fakeDanceNanos) }},
	})

	g.Expect(fsys.WriteFileAtomic(target, []byte("v2"), atomicPerm)).To(gomega.Succeed())
	g.Expect(string(capturedData)).To(gomega.Equal("v2"), "the data lands in the write call")
	g.Expect(calls.list()).To(gomega.Equal([]string{
		"writeexcl .note.md.tmp-12345-0",
		"chmod .note.md.tmp-12345-0",
		"rename .note.md.tmp-12345-0->note.md",
	}), "success path must not remove the renamed file")
}

func TestEdgeFS_WriteFileAtomicUniqueNameRetry(t *testing.T) {
	t.Parallel()

	t.Run("collision retries a fresh candidate then succeeds", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		target := filepath.Join("some", "dir", "note.md")
		tried := make([]string, 0, 2)

		var renamed string

		prims := cli.Primitives{
			FS: cli.FSPrims{
				OpenFileExcl: func(path string, _ fs.FileMode) (io.WriteCloser, error) {
					tried = append(tried, path)
					if len(tried) == 1 {
						return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
					}

					return &mockWriteCloser{}, nil
				},
				Chmod: func(string, fs.FileMode) error { return nil },
				Rename: func(oldPath, _ string) error {
					renamed = oldPath

					return nil
				},
				Remove: func(string) error {
					t.Error("a colliding candidate was not created by the dance and must not be removed")

					return nil
				},
			},
			Proc: cli.ProcPrims{Now: func() time.Time { return time.Unix(0, fakeDanceNanos) }},
		}

		g.Expect(fsFromPrims(prims).WriteFileAtomic(target, []byte("v2"), atomicPerm)).To(gomega.Succeed())
		g.Expect(tried).To(gomega.HaveLen(2))
		g.Expect(tried[0]).NotTo(gomega.Equal(tried[1]), "each retry must try a FRESH candidate name")
		g.Expect(renamed).To(gomega.Equal(tried[1]), "the created candidate is the one renamed into place")
	})

	t.Run("exhausted candidates yield a bounded wrapped error", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		attempts := 0
		prims := cli.Primitives{
			FS: cli.FSPrims{
				OpenFileExcl: func(path string, _ fs.FileMode) (io.WriteCloser, error) {
					attempts++

					return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
				},
			},
			Proc: cli.ProcPrims{Now: func() time.Time { return time.Unix(0, fakeDanceNanos) }},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(fs.ErrExist), "the last collision stays in the error chain")
		g.Expect(err.Error()).To(gomega.ContainSubstring("create temp"))
		g.Expect(err.Error()).To(gomega.ContainSubstring("attempts"))
		g.Expect(attempts).To(gomega.Equal(danceMaxAttempts), "the retry loop must be BOUNDED")
	})
}

func TestEdgeFS_WriteFileExclCloseErrorWhenWriteOK(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	closeErr := errors.New("close failed")
	fsys := fsFromPrims(cli.Primitives{FS: cli.FSPrims{
		OpenFileExcl: func(string, fs.FileMode) (io.WriteCloser, error) {
			return &mockWriteCloser{closeErr: closeErr}, nil
		},
	}})

	err := fsys.WriteFileExcl("path", []byte("data"), atomicPerm)
	g.Expect(err).To(gomega.HaveOccurred())

	if err != nil {
		g.Expect(err).To(gomega.MatchError(closeErr),
			"close error must be returned when write succeeds")
		g.Expect(err.Error()).To(gomega.ContainSubstring("write excl"))
	}
}

func TestEdgeFS_WriteFileExclOpenErrorWrapped(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	openErr := &fs.PathError{Op: "open", Path: "path", Err: fs.ErrPermission}
	fsys := fsFromPrims(cli.Primitives{FS: cli.FSPrims{
		OpenFileExcl: func(string, fs.FileMode) (io.WriteCloser, error) {
			return nil, openErr
		},
	}})

	err := fsys.WriteFileExcl("path", []byte("data"), atomicPerm)
	g.Expect(err).To(gomega.HaveOccurred())

	if err != nil {
		g.Expect(err).To(gomega.MatchError(fs.ErrPermission),
			"open error chain must survive wrap")
		g.Expect(err.Error()).To(gomega.ContainSubstring("path"))
	}
}

func TestEdgeFS_WriteFileExclPassesDataAndPermToPrimitive(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var (
		gotPath string
		gotData []byte
		gotPerm fs.FileMode
	)

	fsys := fsFromPrims(cli.Primitives{FS: cli.FSPrims{
		OpenFileExcl: func(path string, perm fs.FileMode) (io.WriteCloser, error) {
			gotPath = path
			gotPerm = perm

			return &mockWriteCloser{
				writeFunc: func(data []byte) (int, error) {
					gotData = append([]byte(nil), data...)
					return len(data), nil
				},
			}, nil
		},
	}})

	g.Expect(fsys.WriteFileExcl("new.md", []byte("body"), atomicPerm)).To(gomega.Succeed())
	g.Expect(gotPath).To(gomega.Equal("new.md"))
	g.Expect(string(gotData)).To(gomega.Equal("body"))
	g.Expect(gotPerm).To(gomega.Equal(atomicPerm),
		"the caller's perm must reach the primitive unchanged")
}

func TestEdgeFS_WriteFileExclPreservesErrExistAndAddsPath(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := fsFromPrims(cli.Primitives{FS: cli.FSPrims{
		OpenFileExcl: func(path string, _ fs.FileMode) (io.WriteCloser, error) {
			return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
		},
	}})

	err := fsys.WriteFileExcl("existing.md", []byte("x"), atomicPerm)
	g.Expect(err).To(gomega.MatchError(fs.ErrExist),
		"K1 backstop: errors.Is(err, fs.ErrExist) must survive the internal wrap")
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("existing.md")),
		"wrap must add path context")
}

func TestEdgeFS_WriteFileExclSuccessWritesData(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var writtenData []byte

	fsys := fsFromPrims(cli.Primitives{FS: cli.FSPrims{
		OpenFileExcl: func(string, fs.FileMode) (io.WriteCloser, error) {
			return &mockWriteCloser{
				writeFunc: func(data []byte) (int, error) {
					writtenData = append(writtenData, data...)
					return len(data), nil
				},
			}, nil
		},
	}})

	err := fsys.WriteFileExcl("path", []byte("hello world"), atomicPerm)
	g.Expect(err).To(gomega.Succeed())
	g.Expect(writtenData).To(gomega.Equal([]byte("hello world")))
}

func TestEdgeFS_WriteFileExclWriteErrorPrecedence(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writeErr := io.ErrUnexpectedEOF
	fsys := fsFromPrims(cli.Primitives{FS: cli.FSPrims{
		OpenFileExcl: func(string, fs.FileMode) (io.WriteCloser, error) {
			return &mockWriteCloser{writeErr: writeErr}, nil
		},
	}})

	err := fsys.WriteFileExcl("path", []byte("data"), atomicPerm)
	g.Expect(err).To(gomega.HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(gomega.ContainSubstring("write excl"))
		g.Expect(err).To(gomega.MatchError(writeErr),
			"write error must be wrapped and returned even if close also fails")
	}
}

// unexported constants.
const (
	atomicPerm fs.FileMode = 0o600
	// danceMaxAttempts mirrors edgefs.go's maxTempAttempts — the spec'd
	// bound on unique-temp-name candidates (doctrine flag P-4).
	danceMaxAttempts = 10
	// fakeDanceNanos is the fixed clock reading the dance fakes inject;
	// candidate temp names embed it.
	fakeDanceNanos = 12345
)

// callRecorder records call labels in order (single-goroutine use).
type callRecorder struct{ calls []string }

func (c *callRecorder) add(call string) { c.calls = append(c.calls, call) }

func (c *callRecorder) list() []string { return c.calls }

// mockWriteCloser is a test helper WriteCloser that can inject errors at write or close time.
type mockWriteCloser struct {
	writeFunc func([]byte) (int, error)
	writeErr  error
	closeErr  error
}

func (m *mockWriteCloser) Close() error {
	return m.closeErr
}

func (m *mockWriteCloser) Write(p []byte) (int, error) {
	if m.writeFunc != nil {
		return m.writeFunc(p)
	}

	if m.writeErr != nil {
		return 0, m.writeErr
	}

	return len(p), nil
}

// failingPrims returns fresh Primitives whose every filesystem capability
// fails with the given error, for exercising primFS's error-wrap paths.
func failingPrims(boom error) cli.Primitives {
	return cli.Primitives{FS: cli.FSPrims{
		MkdirAll:     func(string, fs.FileMode) error { return boom },
		MkdirTemp:    func(string, string) (string, error) { return "", boom },
		ReadDir:      func(string) ([]fs.DirEntry, error) { return nil, boom },
		Remove:       func(string) error { return boom },
		RemoveAll:    func(string) error { return boom },
		Rename:       func(string, string) error { return boom },
		Stat:         func(string) (fs.FileInfo, error) { return nil, boom },
		WalkDir:      func(string, fs.WalkDirFunc) error { return boom },
		WriteFile:    func(string, []byte, fs.FileMode) error { return boom },
		OpenFileExcl: func(string, fs.FileMode) (io.WriteCloser, error) { return nil, boom },
	}}
}

// fsFromPrims composes the production EdgeFS from fake primitives via the
// public composition root.
func fsFromPrims(prims cli.Primitives) cli.EdgeFS {
	return cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).FS
}
