package vaultgraph_test

import (
	"errors"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

//go:generate impgen vaultgraph.VaultFS --dependency --import-path github.com/toejough/engram/internal/vaultgraph

func TestScanVault_EmptyVault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		notes, err := vaultgraph.ScanVault(mock, "/vault")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(notes).To(BeEmpty())
	}()

	// Flat vault: one list of the root.
	imp.ListMD.ArgsEqual("/vault").Return([]string{}, nil)

	<-done
}

func TestScanVault_NoteWithoutLuhmannID(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		notes, err := vaultgraph.ScanVault(mock, "/vault")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(notes).To(HaveLen(1))
		g.Expect(notes[0].Basename).To(Equal("scratch"))
		g.Expect(notes[0].LuhmannID).To(BeEmpty())
	}()

	imp.ListMD.ArgsEqual("/vault").Return([]string{"scratch.md"}, nil)
	imp.ReadFile.ArgsEqual(filepath.Join("/vault", "scratch.md")).
		Return([]byte("no links"), nil)

	<-done
}

func TestScanVault_ParsesLuhmannAndWikilinks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		notes, err := vaultgraph.ScanVault(mock, "/vault")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(notes).To(HaveLen(1))
		g.Expect(notes[0].Basename).To(Equal("7.2026-05-09.zk"))
		g.Expect(notes[0].LuhmannID).To(Equal("7"))
		g.Expect(notes[0].Outgoing).To(Equal([]string{"4.2026-05-09.anti-index"}))
	}()

	imp.ListMD.ArgsEqual("/vault").
		Return([]string{"7.2026-05-09.zk.md"}, nil)
	imp.ReadFile.ArgsEqual(filepath.Join("/vault", "7.2026-05-09.zk.md")).Return(
		[]byte("body with [[4.2026-05-09.anti-index]] reference"), nil)

	<-done
}

func TestScanVault_PropagatesListError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	wantErr := errors.New("disk failed")

	go func() {
		defer close(done)

		_, err := vaultgraph.ScanVault(mock, "/vault")
		g.Expect(err).To(MatchError(wantErr))
	}()

	imp.ListMD.ArgsEqual("/vault").Return([]string(nil), wantErr)

	<-done
}

func TestScanVault_PropagatesReadError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	wantErr := errors.New("read failed")

	go func() {
		defer close(done)

		_, err := vaultgraph.ScanVault(mock, "/vault")
		g.Expect(err).To(MatchError(wantErr))
	}()

	imp.ListMD.ArgsEqual("/vault").
		Return([]string{"7.2026-05-09.zk.md"}, nil)
	imp.ReadFile.ArgsEqual(filepath.Join("/vault", "7.2026-05-09.zk.md")).Return(
		[]byte(nil), wantErr)

	<-done
}

func TestScanVault_SkipsNonMD(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		notes, err := vaultgraph.ScanVault(mock, "/vault")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(notes).To(BeEmpty())
	}()

	// `notes.txt` returned by ListMD but ParseBasename rejects it → no ReadFile call.
	imp.ListMD.ArgsEqual("/vault").Return([]string{"notes.txt"}, nil)

	<-done
}
