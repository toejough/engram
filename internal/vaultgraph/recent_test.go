package vaultgraph_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/vaultgraph"
)

func TestRecent_DropsBasenamesWithoutDatePrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "MEMORY"},
		{Basename: "1.2026-03-15.b"},
		{Basename: "no-date"},
	}

	got := vaultgraph.Recent(notes, 10)

	g.Expect(got).To(Equal([]string{"1.2026-03-15.b"}))
}

func TestRecent_EmptyInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(vaultgraph.Recent(nil, 10)).To(BeEmpty())
	g.Expect(vaultgraph.Recent([]vaultgraph.Note{}, 10)).To(BeEmpty())
}

func TestRecent_OrdersByDateDescending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "1.2026-01-15.alpha"},
		{Basename: "2.2026-03-20.bravo"},
		{Basename: "3.2026-02-10.charlie"},
	}

	got := vaultgraph.Recent(notes, 10)

	g.Expect(got).To(Equal([]string{
		"2.2026-03-20.bravo",
		"3.2026-02-10.charlie",
		"1.2026-01-15.alpha",
	}))
}

func TestRecent_RespectsLimit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "1.2026-01-15.a"},
		{Basename: "2.2026-02-15.b"},
		{Basename: "3.2026-03-15.c"},
	}

	got := vaultgraph.Recent(notes, 2)

	g.Expect(got).To(Equal([]string{
		"3.2026-03-15.c",
		"2.2026-02-15.b",
	}))
}
