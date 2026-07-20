package embed_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// TestCacheFS_MkdirAll asserts the internal dir-perm policy (0o755) and
// the error wrap.
func TestCacheFS_MkdirAll(t *testing.T) {
	t.Parallel()

	t.Run("passes 0o755 and succeeds", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).MkdirAll("/cache")).To(Succeed())
		g.Expect(recorder.mkdirAllPath).To(Equal("/cache"))
		g.Expect(recorder.mkdirAllPerm).To(Equal(fs.FileMode(0o755)))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{mkdirAllErr: errors.New("denied")}

		err := embed.NewCacheFS(recorder.prims()).MkdirAll("/cache")
		g.Expect(err).To(MatchError(ContainSubstring("mkdir all")))
		g.Expect(err).To(MatchError(ContainSubstring("denied")))
	})
}

// TestCacheFS_MkdirTemp covers passthrough and wrap.
func TestCacheFS_MkdirTemp(t *testing.T) {
	t.Parallel()

	t.Run("returns the created dir", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		tmp, err := embed.NewCacheFS(recorder.prims()).MkdirTemp("/cache", ".tmp-*")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(tmp).To(Equal("/tmp/fake-extract"))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{mkdirTempErr: errors.New("full")}

		_, err := embed.NewCacheFS(recorder.prims()).MkdirTemp("/cache", ".tmp-*")
		g.Expect(err).To(MatchError(ContainSubstring("mkdir temp")))
	})
}

// TestCacheFS_RemoveAllPassesThroughRaw pins the nil-on-missing contract:
// the raw primitive's error (or nil) passes through unwrapped.
func TestCacheFS_RemoveAllPassesThroughRaw(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(embed.NewCacheFS((&cachePrimRecorder{}).prims()).RemoveAll("/tmp/x")).To(Succeed())

	rawErr := errors.New("busy")
	recorder := &cachePrimRecorder{removeAllErr: rawErr}

	err := embed.NewCacheFS(recorder.prims()).RemoveAll("/tmp/x")
	g.Expect(err).To(MatchError(rawErr))
}

// TestCacheFS_RenameExistContract pins the load-bearing contract: every
// destination-exists flavor the raw primitive can produce must surface as
// errors.Is(err, fs.ErrExist); everything else wraps without the sentinel.
func TestCacheFS_RenameExistContract(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		raw       error
		wantExist bool
	}{
		"raw fs.ErrExist": {raw: fs.ErrExist, wantExist: true},
		"LinkError wrapping ErrExist": {
			raw:       &os.LinkError{Op: "rename", Old: "a", New: "b", Err: os.ErrExist},
			wantExist: true,
		},
		"LinkError directory not empty": {
			raw:       &os.LinkError{Op: "rename", Old: "a", New: "b", Err: errors.New("directory not empty")},
			wantExist: true,
		},
		"bare directory-not-empty message": {
			raw:       errors.New("rename a b: directory not empty"),
			wantExist: true,
		},
		"unrelated error": {raw: os.ErrPermission, wantExist: false},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			recorder := &cachePrimRecorder{renameErr: testCase.raw}

			err := embed.NewCacheFS(recorder.prims()).Rename("/tmp/src", "/cache/dst")
			g.Expect(err).To(HaveOccurred())
			g.Expect(errors.Is(err, fs.ErrExist)).To(Equal(testCase.wantExist))

			if !testCase.wantExist {
				g.Expect(err).To(MatchError(ContainSubstring("rename")))
			}
		})
	}

	t.Run("success is nil", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).Rename("/tmp/src", "/cache/dst")).To(Succeed())
	})
}

// TestCacheFS_StatSentinel covers the sentinel-probe branches and proves
// the ".complete" sentinel name is internal policy (the raw Stat sees the
// joined path).
func TestCacheFS_StatSentinel(t *testing.T) {
	t.Parallel()

	t.Run("missing sentinel is false, nil", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{statErr: fs.ErrNotExist}

		present, err := embed.NewCacheFS(recorder.prims()).StatSentinel("/cache/m1")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(present).To(BeFalse())
		g.Expect(recorder.statPath).To(Equal("/cache/m1/.complete"))
	})

	t.Run("stat failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{statErr: errors.New("disk gone")}

		_, err := embed.NewCacheFS(recorder.prims()).StatSentinel("/cache/m1")
		g.Expect(err).To(MatchError(ContainSubstring("stat sentinel")))
		g.Expect(err).To(MatchError(ContainSubstring("disk gone")))
	})

	t.Run("present sentinel is true, nil", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		present, err := embed.NewCacheFS(recorder.prims()).StatSentinel("/cache/m1")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(present).To(BeTrue())
	})
}

// TestCacheFS_WriteFile asserts the internal file-perm policy (0o600) and
// the error wrap.
func TestCacheFS_WriteFile(t *testing.T) {
	t.Parallel()

	t.Run("passes 0o600 and succeeds", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).WriteFile("/tmp/x/model.onnx", []byte("m"))).
			To(Succeed())
		g.Expect(recorder.writePath).To(Equal("/tmp/x/model.onnx"))
		g.Expect(recorder.writePerm).To(Equal(fs.FileMode(0o600)))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{writeErr: errors.New("denied")}

		err := embed.NewCacheFS(recorder.prims()).WriteFile("/tmp/x/model.onnx", []byte("m"))
		g.Expect(err).To(MatchError(ContainSubstring("write file")))
	})
}

// TestCacheFS_WriteSentinel proves the sentinel write is an empty
// ".complete" file under the internal perm policy.
func TestCacheFS_WriteSentinel(t *testing.T) {
	t.Parallel()

	t.Run("writes empty .complete", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).WriteSentinel("/tmp/extract")).To(Succeed())
		g.Expect(recorder.writePath).To(Equal("/tmp/extract/.complete"))
		g.Expect(recorder.writeData).To(BeEmpty())
		g.Expect(recorder.writePerm).To(Equal(fs.FileMode(0o600)))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{writeErr: errors.New("denied")}

		err := embed.NewCacheFS(recorder.prims()).WriteSentinel("/tmp/extract")
		g.Expect(err).To(MatchError(ContainSubstring("write sentinel")))
	})
}

// cachePrimRecorder scripts and records the raw-primitive calls the
// composed CacheFS makes, so sentinel/permission policy and error wraps
// are assertable without a real disk. Each (sub)test builds its own
// recorder — no shared mutable state across parallel tests.
type cachePrimRecorder struct {
	statPath     string
	statErr      error
	mkdirAllPath string
	mkdirAllPerm fs.FileMode
	mkdirAllErr  error
	mkdirTempErr error
	writePath    string
	writeData    []byte
	writePerm    fs.FileMode
	writeErr     error
	renameErr    error
	removeAllErr error
}

func (r *cachePrimRecorder) prims() embed.CacheFSPrims {
	return embed.CacheFSPrims{
		Stat: func(path string) (fs.FileInfo, error) {
			r.statPath = path

			return nil, r.statErr
		},
		MkdirAll: func(path string, perm fs.FileMode) error {
			r.mkdirAllPath = path
			r.mkdirAllPerm = perm

			return r.mkdirAllErr
		},
		MkdirTemp: func(_, _ string) (string, error) {
			return "/tmp/fake-extract", r.mkdirTempErr
		},
		WriteFile: func(path string, data []byte, perm fs.FileMode) error {
			r.writePath = path
			r.writeData = data
			r.writePerm = perm

			return r.writeErr
		},
		Rename: func(_, _ string) error {
			return r.renameErr
		},
		RemoveAll: func(_ string) error {
			return r.removeAllErr
		},
	}
}
