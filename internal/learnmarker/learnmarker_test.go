package learnmarker_test

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/toejough/engram/internal/learnmarker"
)

type fakeFS struct {
	files     map[string][]byte
	mkdirCall string
	writeErr  error
	readErr   error
}

func newFakeFS() *fakeFS { return &fakeFS{files: map[string][]byte{}} }

func (f *fakeFS) ReadFile(path string) ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	b, ok := f.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return b, nil
}

func (f *fakeFS) WriteFile(path string, data []byte, _ os.FileMode) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.files[path] = data
	return nil
}

func (f *fakeFS) MkdirAll(path string, _ os.FileMode) error {
	f.mkdirCall = path
	return nil
}

func TestStateDirFromHome_DefaultsToLocalState(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := learnmarker.StateDirFromHome("/Users/joe", func(string) string { return "" })

	g.Expect(dir).To(gomega.Equal("/Users/joe/.local/state/engram"))
}

func TestStateDirFromHome_RespectsXDGStateHome(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	getenv := func(key string) string {
		if key == "XDG_STATE_HOME" {
			return "/custom/state"
		}
		return ""
	}

	dir := learnmarker.StateDirFromHome("/Users/joe", getenv)

	g.Expect(dir).To(gomega.Equal("/custom/state/engram"))
}

func TestMarkerPath_JoinsStateDirAndSlug(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := learnmarker.MarkerPath("/state/engram", "Users-joe-repos-foo")

	g.Expect(path).To(gomega.Equal("/state/engram/projects/Users-joe-repos-foo/last-learn-at"))
}

func TestRead_ReturnsNotFoundWhenMarkerMissing(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	fs := newFakeFS()

	_, found, err := learnmarker.Read(fs, "/state/engram/projects/foo/last-learn-at")

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(found).To(gomega.BeFalse())
}

func TestRead_ReturnsTimestampWhenMarkerPresent(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	fs := newFakeFS()
	want := time.Date(2026, 5, 13, 18, 30, 0, 0, time.UTC)
	fs.files["/state/engram/projects/foo/last-learn-at"] = []byte(want.Format(time.RFC3339Nano))

	got, found, err := learnmarker.Read(fs, "/state/engram/projects/foo/last-learn-at")

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(found).To(gomega.BeTrue())
	g.Expect(got.Equal(want)).To(gomega.BeTrue())
}

func TestRead_WrapsCorruptTimestampError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	fs := newFakeFS()
	fs.files["/p"] = []byte("not-a-timestamp")

	_, _, err := learnmarker.Read(fs, "/p")

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("learnmarker: parsing")))
}

func TestWrite_CreatesParentDirAndWritesRFC3339Nano(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	fs := newFakeFS()
	when := time.Date(2026, 5, 13, 18, 30, 0, 0, time.UTC)

	err := learnmarker.Write(fs, "/state/engram/projects/foo/last-learn-at", when)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(fs.mkdirCall).To(gomega.Equal("/state/engram/projects/foo"))
	g.Expect(string(fs.files["/state/engram/projects/foo/last-learn-at"])).
		To(gomega.Equal(when.Format(time.RFC3339Nano)))
}

func TestWrite_PropagatesWriteError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	fs := &fakeFS{files: map[string][]byte{}, writeErr: errors.New("disk full")}

	err := learnmarker.Write(fs, "/p", time.Now())

	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("learnmarker: writing")))
}
