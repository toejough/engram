package embed_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestNotExist_InterfaceFallbackFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(embed.ExportNotExist(&fakeIsNotExistError{flag: false})).To(BeFalse())
}

func TestNotExist_InterfaceFallbackTrue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(embed.ExportNotExist(&fakeIsNotExistError{flag: true})).To(BeTrue())
}

func TestNotExist_NilFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(embed.ExportNotExist(nil)).To(BeFalse())
}

func TestNotExist_UnrelatedErrorFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(embed.ExportNotExist(errors.New("network down"))).To(BeFalse())
}

func TestNotExist_WrapsFsErrNotExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	wrapped := &os.PathError{Op: "open", Path: "x", Err: fs.ErrNotExist}
	g.Expect(embed.ExportNotExist(wrapped)).To(BeTrue())
}

// fakeIsNotExistError implements the interface fallback path.
type fakeIsNotExistError struct{ flag bool }

func (f *fakeIsNotExistError) Error() string { return "fake" }

func (f *fakeIsNotExistError) IsNotExist() bool { return f.flag }
