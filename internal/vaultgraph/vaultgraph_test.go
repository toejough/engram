package vaultgraph_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

func TestStartingPoints_EmptyVault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		got, err := vaultgraph.StartingPoints(mock, fixtureVault)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(got).To(BeEmpty())
	}()

	programVaultFSMock(imp, nil)
	<-done
}

func TestStartingPoints_MOCLessComponentEmitsInDegreeWinner(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		got, err := vaultgraph.StartingPoints(mock, fixtureVault)
		g.Expect(err).NotTo(HaveOccurred())
		// C has in-degree 2 (A and B both point to it). Component is {A, B, C}.
		g.Expect(got).To(Equal([]string{"3.2026-05-09.c"}))
	}()

	programVaultFSMock(imp, []inputForStartingPoints{
		{"1.2026-05-09.a.md", "[[3.2026-05-09.c]]"},
		{"2.2026-05-09.b.md", "[[3.2026-05-09.c]]"},
		{"3.2026-05-09.c.md", "leaf"},
	})
	<-done
}

func TestStartingPoints_MultipleComponentsGloballySortedByLuhmann(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		got, err := vaultgraph.StartingPoints(mock, fixtureVault)
		g.Expect(err).NotTo(HaveOccurred())
		// Three components (flat vault — no MOC preference):
		//   {7, 1} mutually linked → in-degree tie (1 each) → earliest Luhmann: 1
		//   {4a} (isolated) → emits 4a
		//   {2, 14} via link 2→14 → emits 14 (in-degree 1, 2 has 0)
		// Globally Luhmann-sorted: 1 < 4a < 14
		g.Expect(got).To(Equal([]string{
			"1.2026-05-09.member",
			"4a.2026-05-09.x",
			"14.2026-05-09.y",
		}))
	}()

	programVaultFSMock(imp, []inputForStartingPoints{
		{"7.2026-05-09.moc.md", "[[1.2026-05-09.member]]"},
		{"1.2026-05-09.member.md", "[[7.2026-05-09.moc]]"},
		{"4a.2026-05-09.x.md", "no links"},
		{"2.2026-05-09.linker.md", "[[14.2026-05-09.y]]"},
		{"14.2026-05-09.y.md", "leaf"},
	})
	<-done
}

func TestStartingPoints_NonLuhmannBasenamesSortAfter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		got, err := vaultgraph.StartingPoints(mock, fixtureVault)
		g.Expect(err).NotTo(HaveOccurred())
		// 9a is a Luhmann ID; "scratch" is not. 9a sorts first.
		g.Expect(got).To(Equal([]string{"9a.2026-05-10.x", "scratch"}))
	}()

	programVaultFSMock(imp, []inputForStartingPoints{
		{"9a.2026-05-10.x.md", "no links"},
		{"scratch.md", "no links"},
	})
	<-done
}

func TestStartingPoints_SameLuhmannIDBasenamesTieBreakLexically(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		got, err := vaultgraph.StartingPoints(mock, fixtureVault)
		g.Expect(err).NotTo(HaveOccurred())
		// Two isolated MOCs share Luhmann ID "5"; both emit. Tie-break is lexical
		// over the full basename, so "5.2026-05-09.alpha" precedes "5.2026-05-10.beta".
		g.Expect(got).To(Equal([]string{
			"5.2026-05-09.alpha",
			"5.2026-05-10.beta",
		}))
	}()

	programVaultFSMock(imp, []inputForStartingPoints{
		{"5.2026-05-10.beta.md", "no links"},
		{"5.2026-05-09.alpha.md", "no links"},
	})
	<-done
}

func TestStartingPoints_SingleMOCComponent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		got, err := vaultgraph.StartingPoints(mock, fixtureVault)
		g.Expect(err).NotTo(HaveOccurred())
		// flat vault: no MOC preference — 4.x wins on in-degree (1 vs 0)
		g.Expect(got).To(Equal([]string{"4.2026-05-09.x"}))
	}()

	programVaultFSMock(imp, []inputForStartingPoints{
		{"7.2026-05-09.zk.md", "links to [[4.2026-05-09.x]]"},
		{"4.2026-05-09.x.md", "body"},
	})
	<-done
}

func TestStartingPoints_TwoIDlessBasenamesSortLexically(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	mock, imp := MockVaultFS(t)

	done := make(chan struct{})

	go func() {
		defer close(done)

		got, err := vaultgraph.StartingPoints(mock, fixtureVault)
		g.Expect(err).NotTo(HaveOccurred())
		// Both basenames have no Luhmann ID → sort lexically.
		g.Expect(got).To(Equal([]string{"alpha", "beta"}))
	}()

	programVaultFSMock(imp, []inputForStartingPoints{
		{"beta.md", "no links"},
		{"alpha.md", "no links"},
	})
	<-done
}

// unexported constants.
const (
	fixtureVault = "/vault"
)

// inputForStartingPoints describes a vault to feed into a mock VaultFS for end-to-end
// tests of StartingPoints. Each entry is one note: filename and body bytes.
type inputForStartingPoints struct {
	filename string
	body     string
}

// programVaultFSMock configures the mock to answer ListMD for the vault root
// (with the given filenames) and ReadFile for each note's path with the given body.
// Must be called from within the goroutine that drives the SUT — call ListMD orders
// follow scanner's loop order: MOCs, then Permanent.
func programVaultFSMock(imp *VaultFSImp, inputs []inputForStartingPoints) {
	// Flat vault: every note lives at the root.
	filenames := make([]string, 0, len(inputs))

	for _, input := range inputs {
		filenames = append(filenames, input.filename)
	}

	imp.ListMD.ArgsEqual(fixtureVault).Return(filenames, nil)

	for _, input := range inputs {
		imp.ReadFile.ArgsEqual(filepath.Join(fixtureVault, input.filename)).
			Return([]byte(input.body), nil)
	}
}
