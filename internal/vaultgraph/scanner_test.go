package vaultgraph_test

import (
	"errors"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/vaultgraph"
)

//go:generate impgen engram/internal/vaultgraph.VaultFS --dependency

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

	// All three subdirs queried; all return empty.
	imp.ListMD.ArgsEqual(filepath.Join("/vault", "MOCs")).Return([]string{}, nil)
	imp.ListMD.ArgsEqual(filepath.Join("/vault", "Permanent")).Return([]string{}, nil)
	imp.ListMD.ArgsEqual(filepath.Join("/vault", "Fleeting")).Return([]string{}, nil)

	<-done
}

func TestScanVault_FleetingWithoutLuhmannID(t *testing.T) {
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
		g.Expect(notes[0].IsMOC).To(BeFalse())
	}()

	imp.ListMD.ArgsEqual(filepath.Join("/vault", "MOCs")).Return([]string{}, nil)
	imp.ListMD.ArgsEqual(filepath.Join("/vault", "Permanent")).Return([]string{}, nil)
	imp.ListMD.ArgsEqual(filepath.Join("/vault", "Fleeting")).Return([]string{"scratch.md"}, nil)
	imp.ReadFile.ArgsEqual(filepath.Join("/vault", "Fleeting", "scratch.md")).Return([]byte("no links"), nil)

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

	imp.ListMD.ArgsEqual(filepath.Join("/vault", "MOCs")).Return([]string(nil), wantErr)

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

	imp.ListMD.ArgsEqual(filepath.Join("/vault", "MOCs")).Return([]string{"7.2026-05-09.zk.md"}, nil)
	imp.ReadFile.ArgsEqual(filepath.Join("/vault", "MOCs", "7.2026-05-09.zk.md")).Return(
		[]byte(nil), wantErr)

	<-done
}

func TestScanVault_SingleMOC(t *testing.T) {
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
		g.Expect(notes[0].IsMOC).To(BeTrue())
		g.Expect(notes[0].Outgoing).To(Equal([]string{"4.2026-05-09.anti-index"}))
	}()

	imp.ListMD.ArgsEqual(filepath.Join("/vault", "MOCs")).Return([]string{"7.2026-05-09.zk.md"}, nil)
	imp.ReadFile.ArgsEqual(filepath.Join("/vault", "MOCs", "7.2026-05-09.zk.md")).Return(
		[]byte("body with [[4.2026-05-09.anti-index]] reference"), nil)
	imp.ListMD.ArgsEqual(filepath.Join("/vault", "Permanent")).Return([]string{}, nil)
	imp.ListMD.ArgsEqual(filepath.Join("/vault", "Fleeting")).Return([]string{}, nil)

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
	imp.ListMD.ArgsEqual(filepath.Join("/vault", "MOCs")).Return([]string{"notes.txt"}, nil)
	imp.ListMD.ArgsEqual(filepath.Join("/vault", "Permanent")).Return([]string{}, nil)
	imp.ListMD.ArgsEqual(filepath.Join("/vault", "Fleeting")).Return([]string{}, nil)

	<-done
}
