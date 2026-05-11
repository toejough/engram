package vaultgraph_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/vaultgraph"
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
		{"Permanent", "1.2026-05-09.a.md", "[[3.2026-05-09.c]]"},
		{"Permanent", "2.2026-05-09.b.md", "[[3.2026-05-09.c]]"},
		{"Permanent", "3.2026-05-09.c.md", "leaf"},
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
		// Three components:
		//   {7-moc} → emits MOC 7
		//   {4a-perm} (isolated) → emits 4a
		//   {2-perm, 14-perm} via link 2→14 → emits 14 (in-degree 1, 2 has 0)
		// Globally Luhmann-sorted: 2 < 4a < 7 < 14
		// But 2 is in a component with 14 and loses → 4a, 7, 14
		// Wait — 7 < 14? Yes, top-level numeric: 7 < 14.
		// So sorted output: 4a, 7, 14
		g.Expect(got).To(Equal([]string{
			"4a.2026-05-09.x",
			"7.2026-05-09.moc",
			"14.2026-05-09.y",
		}))
	}()

	programVaultFSMock(imp, []inputForStartingPoints{
		{"MOCs", "7.2026-05-09.moc.md", "[[1.2026-05-09.member]]"},
		{"Permanent", "1.2026-05-09.member.md", "[[7.2026-05-09.moc]]"},
		{"Permanent", "4a.2026-05-09.x.md", "no links"},
		{"Permanent", "2.2026-05-09.linker.md", "[[14.2026-05-09.y]]"},
		{"Permanent", "14.2026-05-09.y.md", "leaf"},
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
		{"Permanent", "9a.2026-05-10.x.md", "no links"},
		{"Fleeting", "scratch.md", "no links"},
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
		g.Expect(got).To(Equal([]string{"7.2026-05-09.zk"}))
	}()

	programVaultFSMock(imp, []inputForStartingPoints{
		{"MOCs", "7.2026-05-09.zk.md", "links to [[4.2026-05-09.x]]"},
		{"Permanent", "4.2026-05-09.x.md", "body"},
	})
	<-done
}

// unexported constants.
const (
	fixtureVault = "/vault"
)

// inputForStartingPoints describes a vault to feed into a mock VaultFS for end-to-end
// tests of StartingPoints. Each entry is one note: subdir, filename, and body bytes.
type inputForStartingPoints struct {
	subdir   string
	filename string
	body     string
}

// programVaultFSMock configures the mock to answer ListMD for the three subdirs
// (with the given filenames) and ReadFile for each note's path with the given body.
// Must be called from within the goroutine that drives the SUT — call ListMD orders
// follow scanner's loop order: MOCs, Permanent, Fleeting.
func programVaultFSMock(imp *VaultFSImp, inputs []inputForStartingPoints) {
	bySubdir := map[string][]inputForStartingPoints{
		"MOCs":      nil,
		"Permanent": nil,
		"Fleeting":  nil,
	}

	for _, input := range inputs {
		bySubdir[input.subdir] = append(bySubdir[input.subdir], input)
	}

	for _, subdir := range []string{"MOCs", "Permanent", "Fleeting"} {
		dirPath := filepath.Join(fixtureVault, subdir)
		filenames := make([]string, 0, len(bySubdir[subdir]))

		for _, input := range bySubdir[subdir] {
			filenames = append(filenames, input.filename)
		}

		imp.ListMD.ArgsEqual(dirPath).Return(filenames, nil)

		for _, input := range bySubdir[subdir] {
			imp.ReadFile.ArgsEqual(filepath.Join(dirPath, input.filename)).Return([]byte(input.body), nil)
		}
	}
}
