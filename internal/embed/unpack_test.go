package embed_test

import (
	stdembed "embed"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestUnpackModelToTemp_EmptyFSReportsBundledModelMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tfs := &recordingTempFS{tmpDirToReturn: "/tmp/should-not-be-used"}

	var empty stdembed.FS

	_, err := embed.ExportUnpackModelToTemp(tfs, empty, "assets/model")
	g.Expect(err).To(MatchError(embed.ErrBundledModelUnavailable))
	g.Expect(tfs.mkdirCalls).To(Equal(0))
}

func TestUnpackModelToTemp_HappyPathWritesEveryFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tfs := &recordingTempFS{tmpDirToReturn: "/tmp/recording"}

	tmp, err := embed.ExportUnpackModelToTemp(tfs, nonEmptyTestFS, "testdata")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tmp).To(Equal("/tmp/recording"))
	g.Expect(tfs.writeCalls).To(BeNumerically(">", 0))
	g.Expect(tfs.removeCalls).To(Equal(0), "happy path must not remove temp dir")
}

func TestUnpackModelToTemp_MkdirFailurePropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tfs := &recordingTempFS{mkdirErr: errors.New("no space")}

	_, err := embed.ExportUnpackModelToTemp(tfs, nonEmptyTestFS, "testdata")
	g.Expect(err).To(MatchError(ContainSubstring("temp dir")))
	g.Expect(err).To(MatchError(ContainSubstring("no space")))
}

func TestUnpackModelToTemp_WriteFailureRemovesTemp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tfs := &recordingTempFS{
		tmpDirToReturn: "/tmp/recording",
		writeErr:       errors.New("write blocked"),
	}

	_, err := embed.ExportUnpackModelToTemp(tfs, nonEmptyTestFS, "testdata")
	g.Expect(err).To(MatchError(ContainSubstring("unpack")))
	g.Expect(err).To(MatchError(ContainSubstring("write blocked")))
	g.Expect(tfs.removeCalls).To(Equal(1), "RemoveAll must be called on write failure")
}

//go:embed testdata/gen-reference.py
var nonEmptyTestFS stdembed.FS

// recordingTempFS captures every operation so tests can assert mkdir/
// write/remove semantics without touching the real disk.
type recordingTempFS struct {
	tmpDirToReturn string
	mkdirErr       error
	writeErr       error
	removeErr      error
	mkdirCalls     int
	writeCalls     int
	removeCalls    int
	wroteNames     []string
}

func (r *recordingTempFS) MkdirTemp(_, _ string) (string, error) {
	r.mkdirCalls++

	if r.mkdirErr != nil {
		return "", r.mkdirErr
	}

	return r.tmpDirToReturn, nil
}

func (r *recordingTempFS) RemoveAll(_ string) error {
	r.removeCalls++

	return r.removeErr
}

func (r *recordingTempFS) WriteFile(name string, _ []byte) error {
	r.writeCalls++
	r.wroteNames = append(r.wroteNames, name)

	if r.writeErr != nil {
		return r.writeErr
	}

	return nil
}
