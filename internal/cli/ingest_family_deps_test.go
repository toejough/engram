package cli_test

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestIngestDepsLockMkdirErrorPropagates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := cli.Deps{
		FS: fakeEdgeFS{
			mkdirAll: func(string, fs.FileMode) error { return errBoom },
		},
		Lock: fakeLocker{lock: func(string) (func() error, error) {
			t.Fatal("Lock must not be called when MkdirAll fails")

			return nil, errBoom
		}},
	}

	_, err := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{}).Lock("/chunks")
	g.Expect(err).To(gomega.MatchError(errBoom))
}

func TestIngestDepsLockMkdirsChunksDirBeforeLocking(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var calls []string

	d := cli.Deps{
		FS: fakeEdgeFS{
			mkdirAll: func(path string, perm fs.FileMode) error {
				calls = append(calls, fmt.Sprintf("mkdirall %s %o", path, perm))

				return nil
			},
		},
		Lock: fakeLocker{lock: func(path string) (func() error, error) {
			calls = append(calls, "lock "+path)

			return func() error {
				calls = append(calls, "unlock")

				return nil
			}, nil
		}},
	}

	release, err := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{}).Lock("/chunks")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil || release == nil {
		return
	}

	release()

	g.Expect(calls).To(gomega.Equal([]string{
		"mkdirall /chunks 700",
		"lock " + filepath.Join("/chunks", cli.ExportManifestLockFile()),
		"unlock",
	}), "MkdirAll must precede flock (fresh-dir regression), release must unlock")
}

// TestIngestDepsReadTranscriptReadsViaFS proves ReadTranscript's injected
// transcript.JSONLReader routes its raw read through the composed EdgeFS
// (fsFileReader), not a direct os call — the transcript-strip path exercised
// end-to-end over the pure composition.
func TestIngestDepsReadTranscriptReadsViaFS(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const transcriptPath = "/sessions/s.jsonl"

	line := `{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"hello there"}}`

	var readPath string

	d := cli.Deps{FS: fakeEdgeFS{
		readFile: func(path string) ([]byte, error) {
			readPath = path

			return []byte(line + "\n"), nil
		},
	}}

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	result, err := deps.ReadTranscript(transcriptPath, time.Time{}, 0)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(readPath).To(gomega.Equal(transcriptPath), "fsFileReader must read via the injected EdgeFS")
	g.Expect(result.Content).To(gomega.ContainSubstring("hello there"))
}

func TestIngestDepsSessionDirDefaultsToClaudeProjects(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := newTestDeps(io.Discard, io.Discard)
	d.Getenv = func(string) string { return "" }
	d.UserHomeDir = func() (string, error) { return "/home/u", nil }

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.SessionDir("/anywhere")).
		To(gomega.Equal(filepath.Join("/home/u", ".claude", "projects")))

	d.UserHomeDir = func() (string, error) { return "", errBoom }
	deps = cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.SessionDir("/anywhere")).To(gomega.BeEmpty(),
		"unresolvable home yields empty session dir, not a panic")
}

func TestIngestDepsSessionDirHonorsTranscriptDirEnv(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := newTestDeps(io.Discard, io.Discard)
	d.Getenv = func(key string) string {
		if key == "ENGRAM_TRANSCRIPT_DIR" {
			return "/custom/sessions"
		}

		return ""
	}

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.SessionDir("/anywhere")).To(gomega.Equal("/custom/sessions"))
}

func TestIngestDepsStatMapsFileInfoToSourceStat(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	mtime := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)

	const wantSize = int64(42)

	d := newTestDeps(io.Discard, io.Discard)
	d.FS = fakeEdgeFS{stat: func(string) (fs.FileInfo, error) {
		return fakeFileInfo{mtime: mtime, size: wantSize}, nil
	}}

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	stat, err := deps.Stat("/src.md")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stat).To(gomega.Equal(cli.SourceStat{
		MtimeUnixNano: mtime.UnixNano(),
		Size:          wantSize,
	}))

	d.FS = fakeEdgeFS{stat: func(string) (fs.FileInfo, error) { return nil, errBoom }}
	deps = cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	_, err = deps.Stat("/src.md")
	g.Expect(err).To(gomega.MatchError(errBoom))
}

func TestIngestDepsWriteFileMkdirsParentThenWritesAtomically(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var calls []string

	d := newTestDeps(io.Discard, io.Discard)
	d.FS = fakeEdgeFS{
		mkdirAll: func(path string, perm fs.FileMode) error {
			calls = append(calls, fmt.Sprintf("mkdirall %s %o", path, perm))

			return nil
		},
		writeFileAtomic: func(path string, _ []byte, perm fs.FileMode) error {
			calls = append(calls, fmt.Sprintf("writeatomic %s %o", path, perm))

			return nil
		},
	}

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.WriteFile("/chunks/idx.jsonl", []byte("x"))).To(gomega.Succeed())
	g.Expect(calls).To(gomega.Equal([]string{
		"mkdirall /chunks 700",
		"writeatomic /chunks/idx.jsonl 600",
	}), "parent dir created before atomic 0600 write")
}

// fakeEdgeFS implements cli.EdgeFS via injected closures, backing the pure
// newIngestDeps composition tests above. Declared once here (#700 T8, R13) —
// package cli_test is one namespace across files; other clusters consume
// this rather than redeclare it.
type fakeEdgeFS struct {
	readFile        func(string) ([]byte, error)
	writeFile       func(string, []byte, fs.FileMode) error
	writeFileAtomic func(string, []byte, fs.FileMode) error
	writeFileExcl   func(string, []byte, fs.FileMode) error
	mkdirAll        func(string, fs.FileMode) error
	mkdirTemp       func(string, string) (string, error)
	stat            func(string) (fs.FileInfo, error)
	readDir         func(string) ([]fs.DirEntry, error)
	remove          func(string) error
	removeAll       func(string) error
	rename          func(string, string) error
	walkDir         func(string, fs.WalkDirFunc) error
}

func (f fakeEdgeFS) MkdirAll(path string, perm fs.FileMode) error { return f.mkdirAll(path, perm) }

func (f fakeEdgeFS) MkdirTemp(dir, pattern string) (string, error) {
	return f.mkdirTemp(dir, pattern)
}

func (f fakeEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) { return f.readDir(path) }

func (f fakeEdgeFS) ReadFile(path string) ([]byte, error) { return f.readFile(path) }

func (f fakeEdgeFS) Remove(path string) error { return f.remove(path) }

func (f fakeEdgeFS) RemoveAll(path string) error { return f.removeAll(path) }

func (f fakeEdgeFS) Rename(oldPath, newPath string) error { return f.rename(oldPath, newPath) }

func (f fakeEdgeFS) Stat(path string) (fs.FileInfo, error) { return f.stat(path) }

func (f fakeEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error { return f.walkDir(root, fn) }

func (f fakeEdgeFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return f.writeFile(path, data, perm)
}

func (f fakeEdgeFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return f.writeFileAtomic(path, data, perm)
}

func (f fakeEdgeFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	return f.writeFileExcl(path, data, perm)
}

// fakeFileInfo satisfies fs.FileInfo for Stat-mapping tests.
type fakeFileInfo struct {
	mtime time.Time
	size  int64
	dir   bool
}

func (f fakeFileInfo) IsDir() bool { return f.dir }

func (f fakeFileInfo) ModTime() time.Time { return f.mtime }

func (f fakeFileInfo) Mode() fs.FileMode { return 0o600 }

func (f fakeFileInfo) Name() string { return "fake" }

func (f fakeFileInfo) Size() int64 { return f.size }

func (f fakeFileInfo) Sys() any { return nil }

// fakeLocker satisfies cli.FileLocker with an injected func.
type fakeLocker struct {
	lock func(string) (func() error, error)
}

func (l fakeLocker) Lock(path string) (func() error, error) { return l.lock(path) }
