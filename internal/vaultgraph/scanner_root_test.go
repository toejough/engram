package vaultgraph_test

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

func TestScanVaultReadsRootLevelNotes(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := rootFS{files: map[string][]byte{
		"/vault/1.2026-06-12.flat-note.md": []byte("---\ntype: fact\n---\nbody links [[2.other]]\n"),
	}}

	notes, err := vaultgraph.ScanVault(fs, "/vault")

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(notes).To(gomega.HaveLen(1))

	if len(notes) != 1 {
		return
	}

	g.Expect(notes[0].Basename).To(gomega.Equal("1.2026-06-12.flat-note"))
	g.Expect(notes[0].IsMOC).To(gomega.BeFalse())
	g.Expect(notes[0].Outgoing).To(gomega.ConsistOf("2.other"))
}

// rootFS serves notes from the vault ROOT — the flat layout (no ,
// no MOCs/ — those tiers are gone).
type rootFS struct{ files map[string][]byte }

func (f rootFS) ListMD(dir string) ([]string, error) {
	if dir != "/vault" {
		return nil, nil
	}

	return []string{"1.2026-06-12.flat-note.md"}, nil
}

func (f rootFS) ReadFile(path string) ([]byte, error) { return f.files[path], nil }
